# SPDX-License-Identifier: Apache-2.0
"""Reference-backed stages for the maintained connected CLI journey."""
from contextlib import contextmanager
import importlib.util
from pathlib import Path
import re
import time


def references():
    spec = importlib.util.spec_from_file_location("journey_reference", Path(__file__).with_name("acoustic_reference.py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def validate_election(args):
    if args.mode != "real" or not args.package or not args.reference_corpus:
        raise ValueError("Reference acceptance requires the complete real package election")
    if args.recognition_python or args.diarization_python:
        raise ValueError("Reference managed mode forbids external interpreter overrides")
    offline = getattr(args, "managed_offline_assets", None)
    if offline and not Path(offline).is_absolute():
        raise ValueError("Reference offline bundle must be an absolute selected path")
    references().load_frozen_corpus(args.reference_corpus)


@contextmanager
def stage(journey, name):
    stages = journey.receipt.setdefault("stages", [])
    if any(entry["name"] == name for entry in stages):
        raise ValueError("Duplicate real acceptance stage")
    entry = {"name": name, "state": "running"}
    stages.append(entry)
    journey.receipt["current_stage"] = name
    started = time.monotonic()
    try:
        yield entry
    except BaseException:
        entry["state"] = "failed"
        journey.receipt.setdefault("first_failure_stage", name)
        raise
    else:
        entry["state"] = "succeeded"
    finally:
        entry["wall_seconds"] = time.monotonic() - started


def digest(value):
    return isinstance(value, str) and re.fullmatch("[0-9a-f]{64}", value)


def prepare_managed(journey):
    journey.receipt["environments"] = {}
    for role in ("recognition", "diarization"):
        with stage(journey, "managed_" + role):
            election = {"role": role, "mode": "managed"}
            if journey.args.managed_offline_assets:
                election["offline_assets"] = journey.args.managed_offline_assets
            plan = journey.input("workers", "plan", value=election)
            if plan.get("prerequisite") or plan.get("space_sufficient") is False:
                raise ValueError("Managed environment prerequisite is unavailable")
            selected = plan["election"]
            if selected.get("role") != role or selected.get("mode") != "managed" or not digest(selected.get("profile_digest")):
                raise ValueError("Managed plan election identity differs")
            prepared = journey.finish(journey.input("workers", "setup", value=selected), maximum=600)
            if (prepared.get("role") != role or prepared.get("inference") != "not_run"
                    or not digest(prepared.get("python", {}).get("sha256"))
                    or not digest(prepared.get("environment_receipt", {}).get("sha256"))):
                raise ValueError("Managed setup identity differs")
            reuse = journey.input("workers", "plan", value=election)
            if not reuse.get("verified_existing_environment") or reuse.get("download_bytes", 0) != 0:
                raise ValueError("Managed environment reuse did not verify")
            journey.receipt["environments"][role] = {
                "profile_digest": selected["profile_digest"],
                "python_sha256": prepared["python"]["sha256"],
                "receipt_sha256": prepared["environment_receipt"]["sha256"],
                "verified_reuse": True,
            }
    model_ids = {}
    with stage(journey, "models"):
        journey.receipt["model_inputs"] = {}
        for operation, choice, directory in (
                ("transcribe", "whisper-tiny", journey.args.recognition_model),
                ("diarize", "community-1", journey.args.diarization_model)):
            election = {"choice": choice, "local_directory": str(directory)}
            preview = journey.input("models", "prepare", value=election)
            if preview.get("state") != "verified_local" or preview.get("download_bytes") != 0 or not digest(preview.get("manifest_digest")):
                raise ValueError("Selected local model identity did not verify")
            queued = journey.input("models", "prepare", value={**election, "acquire": True})
            accepted = journey.finish({"work_id": queued["acquisition_work_id"]})
            model_id = preview["model_id"]
            if accepted.get("model_id") != model_id or queued["model_id"] != model_id:
                raise ValueError("Accepted model identity differs")
            if journey.command("models", "verify", model_id).get("verified") is not True:
                raise ValueError("Acquired model did not verify")
            reuse = journey.input("models", "prepare", value={"choice": choice})
            if reuse.get("state") != "verified_available" or reuse.get("model_id") != model_id or reuse.get("manifest_digest") != preview["manifest_digest"]:
                raise ValueError("Model reuse identity differs")
            model_ids[operation] = model_id
            journey.receipt["model_inputs"][operation] = {
                "choice": choice, "model_id": model_id, "manifest_sha256": preview["manifest_digest"],
                "total_bytes": preview["total_bytes"], "local_verified_reuse": True,
            }
    with stage(journey, "readiness"):
        readiness = journey.input("processing", "readiness", value={
            "recognition_model_id": model_ids["transcribe"], "diarization_model_id": model_ids["diarize"],
            "recognition_device": "cpu", "diarization_device": "cpu"})
        for name in ("import", "transcription", "diarization"):
            if readiness.get("tasks", {}).get(name, {}).get("usable") is not True:
                raise ValueError("Real CPU task prerequisites are unavailable")
    return model_ids


REQUIRED_STAGES = (
    "package", "corpus", "managed_recognition", "managed_diarization", "models", "readiness",
    "processing", "enrollment", "matching", "queries", "timeline", "playback", "fetch", "restore", "cleanup",
)
SPEAKER_OUTCOMES = ("correct_match", "incorrect", "unknown", "correct_unknown", "ambiguous",
                    "mixed_reference", "unreferenced_activity", "missing_prediction")


def corpus_digest(corpus):
    import hashlib
    import json
    return hashlib.sha256((json.dumps(corpus, ensure_ascii=False, indent=2) + "\n").encode("utf-8")).hexdigest()


def document_text(record):
    return " ".join(cue["payload"]["plain_text"] for cue in record["document"]["cues"])


def quality_diagnostics(value):
    import math
    if not isinstance(value, list) or len(value) > 4096:
        raise ValueError("Accepted quality diagnostic inventory differs")
    result = []
    for item in value:
        code = item.get("code") if isinstance(item, dict) else None
        if not isinstance(code, str) or not re.fullmatch(r"[A-Za-z0-9_.:-]{1,128}", code):
            raise ValueError("Accepted quality diagnostic code differs")
        row = {"code": code}
        if "count" in item:
            count = item["count"]
            if type(count) is not int or count < 0:
                raise ValueError("Accepted quality diagnostic count differs")
            row["count"] = count
        if item.get("value") is not None:
            number = item["value"]
            if type(number) not in (int, float) or not math.isfinite(number):
                raise ValueError("Accepted quality diagnostic value differs")
            row["value"] = number
        result.append(row)
    return result


def voice_intervals(record, voice):
    intervals = []
    for cue in record["document"]["cues"]:
        for attribution in cue.get("speaker_attributions", []):
            if attribution["speaker_id"] != voice:
                continue
            timing = cue["timing"]
            start = attribution.get("start_milliseconds")
            end = attribution.get("end_milliseconds")
            start = timing["start_milliseconds"] if start is None else start
            end = timing["end_milliseconds"] if end is None else end
            if type(start) is not int or type(end) is not int or not 0 <= start < end <= 60000:
                raise ValueError("Accepted voice interval differs")
            intervals.append((start * 1000, end * 1000))
    return intervals


def playback_identity(media, record, descriptor, observed_size, observed_digest):
    if (record["source_digest"] != media["digest"] or record["id"] != media["id"]
            or descriptor.get("media_id") != media["id"] or descriptor.get("revision") != media["revision"]
            or descriptor.get("recording_revision") != record["revision"]
            or descriptor.get("document_digest") != record["document_digest"]
            or descriptor.get("source_digest") != media["digest"] or descriptor.get("digest") != observed_digest
            or observed_digest != media["digest"] or descriptor.get("size_bytes") != observed_size
            or descriptor.get("source_size_bytes") != observed_size or descriptor.get("preview") is not False
            or descriptor.get("timeline_offset_seconds") != 0):
        raise ValueError("Current playback authority differs")
    return {key: descriptor[key] for key in (
        "media_id", "revision", "recording_revision", "document_digest", "digest", "size_bytes",
        "source_digest", "source_size_bytes", "mime_type", "preview", "timeline_offset_seconds")}


def timeline_identity(rows, record):
    expected = {cue["id"]: cue["timing"] for cue in record["document"]["cues"]}
    actual = {}
    for row in rows:
        if row.get("kind") != "cue":
            continue
        cue_id = row.get("cue_id")
        timing = expected.get(cue_id)
        if (timing is None or cue_id in actual or row.get("media_id") != record["id"]
                or row.get("recording_revision") != record["revision"] or row.get("document_digest") != record["document_digest"]
                or row.get("source_start_us") != str(timing["start_milliseconds"] * 1000)
                or row.get("source_end_us") != str(timing["end_milliseconds"] * 1000)
                or row.get("seek_seconds") != timing["start_milliseconds"] / 1000):
            raise ValueError("Current timeline cue authority differs")
        actual[cue_id] = {key: row[key] for key in (
            "media_id", "recording_revision", "document_digest", "cue_id", "source_start_us", "source_end_us", "seek_seconds")}
    if set(actual) != set(expected):
        raise ValueError("Current timeline cue inventory differs")
    return [actual[key] for key in sorted(actual)]


def validate_acceptance(receipt, corpus):
    import math
    from collections import Counter
    if receipt.get("kind") != "insonic-real-cpu-acceptance-v1" or receipt.get("mode") != "real" or receipt.get("device") != "cpu":
        raise ValueError("Real acceptance receipt election differs")
    package = receipt.get("package", {})
    revision = receipt.get("source_revision")
    if (not isinstance(revision, str) or not re.fullmatch("[0-9a-f]{40}", revision)
            or package.get("revision") != revision or package.get("source_dirty") is not False
            or receipt.get("harness_revision") != revision or receipt.get("harness_dirty") is not False
            or receipt.get("harness_unchanged") is not True
            or not digest(package.get("archive_sha256")) or not digest(package.get("inventory_sha256"))):
        raise ValueError("Real acceptance source/package identity differs")
    if any(not digest(receipt.get(key)) for key in ("cli_sha256", "worker_sha256", "harness_inputs_sha256")):
        raise ValueError("Real acceptance executable/harness identity differs")
    stages = receipt.get("stages", [])
    if (not isinstance(stages, list) or len(stages) != len(REQUIRED_STAGES)
            or [s.get("name") for s in stages] != list(REQUIRED_STAGES)
            or any(s.get("state") != "succeeded" for s in stages)):
        raise ValueError("Real acceptance required stage inventory differs")
    expected = {clip["id"]: clip for clip in corpus["clips"]}
    cases = receipt.get("cases", [])
    if (not isinstance(cases, list) or len(cases) != len(expected) + 1
            or {c.get("fixture") for c in cases} != set(expected) | {"sequence"}):
        raise ValueError("Real acceptance case inventory differs")
    if receipt.get("corpus_sha256") != corpus_digest(corpus):
        raise ValueError("Real acceptance corpus identity differs")
    if receipt.get("matching_settings") != corpus["matching"]:
        raise ValueError("Real acceptance frozen matching settings differ")
    sequence = receipt.get("sequence", {})
    cursor, components = 0, []
    for index, key in enumerate(corpus["sequence"]["clips"]):
        clip = expected[key]
        cursor += corpus["sequence"]["gap_samples"] if index else 0
        components.append({"fixture": key, "sha256": clip["sha256"], "start_sample": cursor,
                           "end_sample": cursor + clip["samples"]})
        cursor += clip["samples"]
    if (not digest(sequence.get("sha256")) or sequence.get("sample_rate") != 16000
            or sequence.get("samples") != cursor or sequence.get("components") != components
            or sequence.get("gap_samples") != corpus["sequence"]["gap_samples"]):
        raise ValueError("Real acceptance derived sequence identity differs")
    environments = receipt.get("environments", {})
    models = receipt.get("model_inputs", {})
    for role, operation in (("recognition", "transcribe"), ("diarization", "diarize")):
        environment = environments.get(role, {})
        if (not digest(environment.get("profile_digest")) or not digest(environment.get("python_sha256"))
                or not digest(environment.get("receipt_sha256")) or environment.get("verified_reuse") is not True):
            raise ValueError("Real acceptance managed environment identity differs")
        model = models.get(operation, {})
        if not model.get("model_id") or not digest(model.get("manifest_sha256")) or model.get("local_verified_reuse") is not True:
            raise ValueError("Real acceptance model identity differs")
    for case in cases:
        fixture = case["fixture"]
        original = expected.get(fixture)
        if original is not None and case.get("original_sha256") != original["sha256"]:
            raise ValueError("Real acceptance original case identity differs")
        if original is None and case.get("original_sha256") != sequence["sha256"]:
            raise ValueError("Real acceptance derived case identity differs")
        if not digest(case.get("document_digest")) or not digest(case.get("canonical_audio_sha256")):
            raise ValueError("Real acceptance current case authority differs")
        if case.get("diagnostics") != quality_diagnostics(case.get("diagnostics")):
            raise ValueError("Real acceptance quality diagnostic projection differs")
        for role, operation, field in (("recognition", "transcribe", "transcription"),
                                       ("diarization", "diarize", "diarization")):
            observation = case.get(field, {})
            if (observation.get("device") != "cpu" or observation.get("offline") is not True
                    or observation.get("model_id") != models[operation]["model_id"]
                    or observation.get("model_manifest_sha256") != models[operation]["manifest_sha256"]
                    or observation.get("python_sha256") != environments[role]["python_sha256"]
                    or observation.get("worker_sha256") != receipt.get("worker_sha256")):
                raise ValueError("Real acceptance engine case provenance differs")
            if (type(observation.get("threads")) is not int or not 1 <= observation["threads"] <= 64
                    or type(observation.get("peak_rss_bytes")) is not int or observation["peak_rss_bytes"] <= 0
                    or not observation.get("packages")
                    or any(type(observation.get(key)) not in (int, float) or not math.isfinite(observation[key])
                           or observation[key] < 0 for key in ("wall_seconds", "cpu_seconds"))):
                raise ValueError("Real acceptance engine resource measurements differ")
        comparison = case.get("word_error", {})
        text = original["transcript"] if original else " ".join(expected[key]["transcript"] for key in corpus["sequence"]["clips"])
        count = len(references().words(text))
        values = [comparison.get(key) for key in ("substitutions", "deletions", "insertions", "edit_distance")]
        if (comparison.get("reference_words") != count or any(type(v) is not int or v < 0 for v in values)
                or sum(values[:3]) != values[3] or type(comparison.get("generated_words")) is not int
                or comparison["generated_words"] != count - values[1] + values[2]
                or values[0] + values[1] > count or comparison.get("word_error_rate") != values[3] / count):
            raise ValueError("Real acceptance word comparison differs")
    totals = {key: sum(case["word_error"][key] for case in cases) for key in (
        "reference_words", "generated_words", "substitutions", "deletions", "insertions", "edit_distance")}
    totals["word_error_rate"] = totals["edit_distance"] / totals["reference_words"]
    if receipt.get("recognition_totals") != totals:
        raise ValueError("Real acceptance recognition totals differ")
    profiles = receipt.get("speaker_profiles", [])
    if (len(profiles) != 2 or {p.get("reader") for p in profiles} != set(corpus["selection"]["enrolled_speakers"])
            or any(not p.get("version_id") or not p.get("files") or any(not digest(v) for v in p["files"].values()) for p in profiles)):
        raise ValueError("Real acceptance speaker profile inventory differs")
    evaluated = receipt.get("speaker_cases", [])
    evaluation_ids = {c["id"] for c in corpus["clips"] if c["role"] != "enrollment"} | {"sequence"}
    if len(evaluated) != len(evaluation_ids) or {c.get("fixture") for c in evaluated} != evaluation_ids:
        raise ValueError("Real acceptance speaker case inventory differs")
    enrolled = {profile["reader"]: profile["speaker_id"] for profile in profiles}
    versions = {profile["speaker_id"]: profile["version_id"] for profile in profiles}
    outcomes = Counter(dict.fromkeys(SPEAKER_OUTCOMES, 0))
    for case in evaluated:
        decisions = case.get("decisions")
        if not isinstance(decisions, list) or case.get("missing_prediction") is not (not bool(decisions)):
            raise ValueError("Real acceptance speaker decision inventory differs")
        voices = set()
        for row in decisions:
            voice = row.get("local_speaker_id")
            if not isinstance(voice, str) or not voice or voice in voices:
                raise ValueError("Real acceptance repeated speaker decision")
            voices.add(voice)
            windows = row.get("reference_window_microseconds")
            readers = (set(corpus["selection"]["enrolled_speakers"]) if case["fixture"] == "sequence"
                       else {expected[case["fixture"]]["speaker"]})
            if (not isinstance(windows, dict) or not set(windows) <= readers
                    or any(type(v) is not int or not 0 < v <= 60000000 for v in windows.values())
                    or type(row.get("evidence_us")) is not int or row["evidence_us"] < 0):
                raise ValueError("Real acceptance speaker evidence differs")
            expected_reader = next(iter(windows)) if len(windows) == 1 else None
            native = {"state": row.get("observed_state"), "score": row.get("score"), "speaker_id": row.get("selected_speaker_id")}
            compared = references().compare_decision(expected_reader, native, enrolled)
            if len(windows) != 1:
                compared["outcome"] = "mixed_reference" if windows else "unreferenced_activity"
            if any(row.get(key) != value for key, value in compared.items()):
                raise ValueError("Real acceptance speaker outcome differs")
            if native["state"] == "accepted" and row.get("version_id") != versions[native["speaker_id"]]:
                raise ValueError("Real acceptance selected profile version differs")
            runner_up = row.get("runner_up_score")
            if runner_up is not None and (type(runner_up) not in (int, float) or not math.isfinite(runner_up)):
                raise ValueError("Real acceptance speaker runner-up score differs")
            outcomes[row["outcome"]] += 1
        if not decisions:
            outcomes["missing_prediction"] += 1
    if receipt.get("speaker_outcomes") != dict(outcomes):
        raise ValueError("Real acceptance speaker totals differ")
    if receipt.get("speaker_denominators") != {"evaluation_cases": len(evaluated),
                                               "observed_decisions": sum(len(c["decisions"]) for c in evaluated)}:
        raise ValueError("Real acceptance speaker denominators differ")
    fixtures = set(expected) | {"sequence"}
    if (set(receipt.get("timeline_cue_counts", {})) != fixtures or set(receipt.get("playback", {})) != fixtures
            or receipt.get("query_operations") != ["media-list", "speaker-search", "model-list", "text-search"]):
        raise ValueError("Real acceptance retrieval case inventory differs")
    if receipt.get("cleanup") != {"runtime_released": True, "temporary_removed": True}:
        raise ValueError("Real acceptance owned cleanup is incomplete")
    timings = [receipt.get(key) for key in ("preparation_seconds", "qualification_seconds", "combined_seconds")]
    if (any(type(value) not in (int, float) or not math.isfinite(value) or value < 0 for value in timings)
            or timings[2] < timings[0] + timings[1] or type(receipt.get("overall_bound_seconds")) is not int
            or not 60 <= receipt["overall_bound_seconds"] <= 3600 or timings[2] > receipt["overall_bound_seconds"]):
        raise ValueError("Real acceptance phase timings differ")
    if (receipt.get("source_store") != "unavailable at original path"
            or receipt.get("post_restore_checks") != {"authority": True, "queries": True, "timeline": True, "playback": True, "fetched_models": True}):
        raise ValueError("Real acceptance restore comparisons are incomplete")


def build_cases(journey, corpus):
    """Snapshot exact inputs before decode; never feed mutable source paths."""
    import hashlib
    import wave
    ref = references()
    source = Path(journey.args.reference_corpus).resolve().parent
    target = journey.base / "reference-inputs"
    target.mkdir()
    cases = []
    for clip in corpus["clips"]:
        data = ref.file_bytes(source, clip["path"], clip["size_bytes"], clip["sha256"])
        snapshot = target / clip["path"]
        snapshot.write_bytes(data)
        cases.append({"fixture": clip["id"], "path": snapshot, "role": clip["role"], "reader": clip["speaker"],
                      "reference": clip["transcript"], "original_sha256": clip["sha256"], "samples": clip["samples"],
                      "windows": [{"start_us": 0, "end_us": clip["samples"] * 1000000 // 16000, "speaker": clip["speaker"]}]})
    by_id = {case["fixture"]: case for case in cases}
    decoder = journey.tools["ffmpeg"]
    if ref.sha(decoder["path"]) != decoder["sha256"]:
        raise ValueError("Reference decoder identity differs")
    sequence = target / "two-reader-sequence.wav"
    components, windows, texts = [], [], []
    cursor = 0
    with wave.open(str(sequence), "wb") as output:
        output.setparams((1, 2, 16000, 0, "NONE", "not compressed"))
        for index, fixture in enumerate(corpus["sequence"]["clips"]):
            case = by_id[fixture]
            if index:
                gap = corpus["sequence"]["gap_samples"]
                output.writeframes(b"\0" * (gap * 2))
                cursor += gap
            raw = journey.engine_child(
                [decoder["path"], "-nostdin", "-hide_banner", "-loglevel", "error", "-i", case["path"],
                 "-map", "0:a:0", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-f", "s16le", "-"],
                maximum=30, max_output=case["samples"] * 2 + 1)
            if len(raw) != case["samples"] * 2:
                raise ValueError("Reference decoded sample identity differs")
            output.writeframes(raw)
            components.append({"fixture": fixture, "sha256": case["original_sha256"],
                               "start_sample": cursor, "end_sample": cursor + case["samples"]})
            windows.append({"start_us": cursor * 1000000 // 16000,
                            "end_us": (cursor + case["samples"]) * 1000000 // 16000, "speaker": case["reader"]})
            texts.append(case["reference"])
            cursor += case["samples"]
    sequence_digest = ref.sha(sequence)
    journey.receipt["sequence"] = {"sha256": sequence_digest, "sample_rate": 16000, "samples": cursor,
                                  "components": components, "gap_samples": corpus["sequence"]["gap_samples"]}
    cases.append({"fixture": "sequence", "path": sequence, "role": "sequence", "reader": None,
                  "reference": " ".join(texts), "original_sha256": sequence_digest, "samples": cursor, "windows": windows})
    return cases


def media_entry(journey, media_id):
    value = journey.command("media", "show", media_id)
    return value.get("entry", value)


def engine_observation(journey, record, role, operation, models):
    provenance = record["provenance"][role]
    environment = "recognition" if role == "transcription" else "diarization"
    selected = journey.receipt["environments"][environment]
    if (provenance.get("device") != "cpu" or provenance.get("offline") is not True
            or provenance.get("model_id") != models[operation]
            or provenance.get("model_manifest_sha256") != journey.receipt["model_inputs"][operation]["manifest_sha256"]
            or provenance.get("python_sha256") != selected["python_sha256"]
            or provenance.get("worker_sha256") != journey.receipt["worker_sha256"]):
        raise ValueError("Real engine provenance differs from frozen election")
    fields = ("device", "offline", "model_id", "model_manifest_sha256", "model_revision", "python_sha256", "worker_sha256",
              "packages", "threads", "wall_seconds", "cpu_seconds", "peak_rss_bytes", "sample_count", "sample_rate",
              "speaker_count", "turn_count", "segment_count", "compute_type", "audio_sha256", "language", "beam_size",
              "vad_filter", "condition_on_previous_text", "quality_diagnostics_enabled", "telemetry_enabled",
              "recognition_boundary_policy", "model_boundary_policy")
    return {key: provenance[key] for key in fields if key in provenance}


def process_cases(journey, cases, models):
    ref = references()
    journey.receipt["cases"] = []
    for case in cases:
        imported = journey.finish(journey.command("media", "import", case["path"], "--new-entry", "--no-pipeline",
                                                 "--originated-at", "2020-01-01T00:00:00.123456789Z"))
        media_id = imported["items"][0]["media_id"]
        case["media_id"] = media_id
        entry = media_entry(journey, media_id)
        stream = entry["facts"]["canonical"]["tracks"][0]["index"]
        journey.finish(journey.input("recordings", "process", media_id, value={
            "transcription": "generate", "diarization": "run", "recognition_model_id": models["transcribe"],
            "diarization_model_id": models["diarize"], "recognition": {"device": "cpu", "language": "en"},
            "attribution": {"device": "cpu"}, "audio": {"stream_index": stream}}))
        record = journey.current_record(media_id)
        if record["source_digest"] != entry["digest"]:
            raise ValueError("Accepted processing lost current audio authority")
        case["document_digest"] = record["document_digest"]
        comparison = ref.word_counts(case["reference"], document_text(record))
        journey.receipt["cases"].append({
            "fixture": case["fixture"], "role": case["role"], "original_sha256": case["original_sha256"],
            "media_id": media_id, "canonical_audio_sha256": record["source_digest"], "document_digest": record["document_digest"],
            "word_error": comparison,
            "diagnostics": quality_diagnostics(record.get("diagnostics")),
            "transcription": engine_observation(journey, record, "transcription", "transcribe", models),
            "diarization": engine_observation(journey, record, "diarization", "diarize", models),
        })
    total = {key: sum(case["word_error"][key] for case in journey.receipt["cases"])
             for key in ("reference_words", "generated_words", "substitutions", "deletions", "insertions", "edit_distance")}
    total["word_error_rate"] = total["edit_distance"] / total["reference_words"] if total["reference_words"] else None
    journey.receipt["recognition_totals"] = total


ADAPTER = {"id": "pyannote-profile", "contract_version": "1", "mode": "local", "architecture": "pyannote",
           "output_kinds": ["voice-embedding"], "consumers": ["voice-matching"], "supported_operations": ["voice-matching"]}


def enroll_profiles(journey, cases, models, reader_ids):
    profiles = []
    for case in cases:
        if case["role"] != "enrollment":
            continue
        reader, media_id = case["reader"], case["media_id"]
        speaker_id = reader_ids[reader]
        journey.input("speakers", "set", speaker_id, value={
            "expected_revision": 0, "speaker": {"id": speaker_id, "name": "Reference reader " + reader,
                                               "revision": 0, "state": "active"}, "aliases": []})
        record = journey.current_record(media_id)
        voices = sorted({a["speaker_id"] for cue in record["document"]["cues"] for a in cue.get("speaker_attributions", [])})
        if not voices:
            raise ValueError("Real single-reader enrollment has no observed voice activity")
        for voice in voices:
            current = journey.current_record(media_id)
            journey.input("recordings", "map-speaker", media_id, value={
                "revision": current["revision"], "mapping_revision": 0, "local_speaker_id": voice,
                "speaker_id": speaker_id, "document_digest": current["document_digest"]})
        dataset = journey.input("models", "dataset", "create", value={
            "speaker_id": speaker_id, "recipe": {"recording_ids": [media_id], "min_duration_us": 1000000}})
        dataset_id = dataset["dataset"]["id"]
        trained = journey.finish(journey.input("models", "train", value={
            "dataset_id": dataset_id, "name": "Reference reader " + reader, "output_kind": "voice-embedding",
            "base_model_id": models["diarize"], "adapter": ADAPTER}))
        version_id = trained["version_id"]
        journey.input("models", "profile", "set", speaker_id, value={"expected_revision": 0, "version_id": version_id})
        profiles.append({"reader": reader, "speaker_id": speaker_id, "dataset_id": dataset_id, "version_id": version_id,
                         "enrollment_fixture": case["fixture"], "enrollment_document_digest": case["document_digest"]})
    if len(profiles) != 2:
        raise ValueError("Real enrollment profile inventory differs")
    journey.receipt["speaker_profiles"] = profiles
    return profiles


def match_cases(journey, cases, models, reader_ids, settings):
    from collections import Counter
    ref = references()
    comparisons, totals = [], Counter(dict.fromkeys(SPEAKER_OUTCOMES, 0))
    for case in cases:
        if case["role"] == "enrollment":
            continue
        media_id = case["media_id"]
        for speaker_id in reader_ids.values():
            roster = journey.command("recordings", "roster", "show", media_id)
            journey.command("recordings", "roster", "add", media_id, "--speaker", speaker_id,
                            "--expected-revision", str(roster["revision"]))
        result = journey.finish(journey.input("recordings", "match", media_id, value={
            "model_id": models["diarize"], "adapter": ADAPTER, **settings}))
        if result.get("state") != "accepted" or result.get("missing_profiles"):
            raise ValueError("Real matching did not accept all elected profiles")
        record = journey.current_record(media_id)
        rows = []
        for decision in result["decisions"]:
            voice = decision["local_speaker_id"]
            windows = ref.reference_readers(voice_intervals(record, voice), case["windows"])
            if len(windows) == 1:
                expected = next(iter(windows))
                compared = ref.compare_decision(expected, decision, reader_ids)
            else:
                # Validate native state/score even when a predicted voice spans
                # multiple independent source windows. Never choose a majority
                # reader to conceal merging or force an identification pass.
                compared = ref.compare_decision(None, decision, reader_ids)
                compared["outcome"] = "mixed_reference" if windows else "unreferenced_activity"
            row = {**compared, "local_speaker_id": voice, "reference_window_microseconds": windows,
                   "runner_up_score": decision.get("runner_up_score"), "evidence_us": decision["evidence_us"],
                   "version_id": decision.get("version_id")}
            rows.append(row)
            totals[row["outcome"]] += 1
        if not rows:
            totals["missing_prediction"] += 1
        comparisons.append({"fixture": case["fixture"], "decisions": rows, "missing_prediction": not bool(rows)})
    journey.receipt["speaker_cases"] = comparisons
    journey.receipt["speaker_outcomes"] = dict(totals)
    journey.receipt["speaker_denominators"] = {"evaluation_cases": len(comparisons),
                                               "observed_decisions": sum(len(c["decisions"]) for c in comparisons)}


def query_snapshot(journey, search):
    results = {}
    for operation in ("media-list", "speaker-search", "model-list", "text-search"):
        result = journey.query(operation, {"text": search} if operation == "text-search" else None)
        if not result.get("items") or result.get("next_cursor"):
            raise ValueError("Reference query inventory is empty or exceeds its page bound")
        results[operation] = result["items"]
    return results


def timeline_snapshot(journey, cases):
    result = {}
    for case in cases:
        rows, cursor = [], None
        for _ in range(10):
            pagination = {"limit": 100}
            if cursor:
                pagination["cursor"] = cursor
            page = journey.input("timeline", "recording", case["media_id"], value={"pagination": pagination})
            if page.get("graph_error"):
                raise ValueError("Reference timeline fell back from its native graph")
            rows.extend(page["items"])
            cursor = page.get("next_cursor")
            if not cursor:
                break
        else:
            raise ValueError("Reference timeline exceeds its page bound")
        result[case["fixture"]] = timeline_identity(rows, journey.current_record(case["media_id"]))
    return result


def owned_file(path, base, maximum=64 << 20):
    path = Path(path)
    if path.is_symlink():
        raise ValueError("Reference artifact must be an owned regular file")
    resolved = path.resolve(strict=True)
    if not resolved.is_relative_to(base.resolve(strict=True)) or not resolved.is_file() or resolved.stat().st_size > maximum:
        raise ValueError("Reference artifact exceeds its owned path or byte bound")
    return resolved


def playback_snapshot(journey, cases):
    result = {}
    for case in cases:
        media = media_entry(journey, case["media_id"])
        record = journey.current_record(case["media_id"])
        descriptor = journey.input("media", "playback", case["media_id"], value={
            "revision": media["revision"], "recording_revision": record["revision"],
            "document_digest": record["document_digest"]})
        primary = None
        try:
            current = journey.input("media", "playback-check", case["media_id"], value={"playback_id": descriptor["playback_id"]})
            path = owned_file(current["path"], journey.base)
            result[case["fixture"]] = playback_identity(media, record, current, path.stat().st_size, references().sha(path))
        except BaseException as error:
            primary = error
            raise
        finally:
            try:
                closed = journey.input("media", "playback-close", case["media_id"], value={"playback_id": descriptor["playback_id"]})
                if closed.get("closed") is not True:
                    raise ValueError("Reference playback handle did not close")
            except BaseException:
                if primary is None:
                    raise
                primary.add_note("Reference playback handle cleanup also failed")
    return result


def fetched_files(journey, version_id, destination):
    journey.command("models", "speaker", "fetch", version_id, "--destination", destination)
    destination = destination.resolve(strict=True)
    files, total = {}, 0
    for path in sorted(destination.rglob("*")):
        if path.is_symlink():
            raise ValueError("Retrieved profile contains a symbolic link")
        if not path.is_file():
            continue
        path = owned_file(path, destination, 16 << 20)
        total += path.stat().st_size
        if len(files) >= 100 or total > 32 << 20:
            raise ValueError("Retrieved profile exceeds its inventory bound")
        files[path.relative_to(destination).as_posix()] = references().sha(path)
    if not files:
        raise ValueError("Retrieved profile has no files")
    return files


def portable_authority(journey, cases, profiles):
    ids = [case["media_id"] for case in cases]
    first, second = profiles
    return [journey.authority(ids, first["speaker_id"], first["version_id"]),
            journey.authority([], second["speaker_id"], second["version_id"])]


def require_equal(before, after, name):
    if before != after:
        raise ValueError("Restored " + name + " differs from accepted source authority")


def finalize_runtime(journey):
    journey.receipt["current_stage"] = "cleanup"
    try:
        journey.idle_runtime()
        journey.assert_harness_unchanged()
    except BaseException:
        journey.receipt.setdefault("first_failure_stage", "cleanup")
        raise


def run(journey):
    import json
    import uuid
    ref = references()
    journey.receipt.update(kind="insonic-real-cpu-acceptance-v1", state="running", stages=[],
                           operation_bounds_seconds={"default": 180, "managed_setup": 600, "decode": 30},
                           child_output_bound_bytes=16 << 20)
    with stage(journey, "package"):
        actual = journey.engine_child([journey.cli, "version"], maximum=10).decode().strip()
        if actual != journey.receipt["product_version"]:
            raise ValueError("Selected CLI version differs from current product")
        journey.command("workspace", "init", journey.workspace, workspace=False)
        journey.command("credentials", "select", "session")
        journey.start_runtime()
        graph = journey.command("graph", "capabilities")
        if graph.get("state") != "available" or graph.get("adapter_id") != "ladybugdb":
            raise ValueError("Reference acceptance requires the installed native LadybugDB graph")
        journey.receipt["graph"] = {key: graph.get(key) for key in ("state", "adapter_id", "backend_version")}
        journey.receipt["companion_sha256"] = {
            **{name: journey.tools[name]["sha256"] for name in ("ffmpeg", "ffprobe", "exiftool")},
            "cueson": ref.sha(journey.cueson)}
    with stage(journey, "corpus"):
        corpus = ref.load_frozen_corpus(journey.args.reference_corpus)
        journey.receipt["corpus_sha256"] = corpus_digest(corpus)
        journey.receipt["matching_settings"] = corpus["matching"]
        journey.receipt["reference_normalization"] = ref.NORMALIZATION
        cases = build_cases(journey, corpus)
    models = prepare_managed(journey)
    qualification_started = time.monotonic()
    journey.receipt["preparation_seconds"] = qualification_started - journey.started
    with stage(journey, "processing"):
        process_cases(journey, cases, models)
    readers = {reader: str(uuid.uuid4()) for reader in corpus["selection"]["enrolled_speakers"]}
    with stage(journey, "enrollment"):
        profiles = enroll_profiles(journey, cases, models, readers)
    with stage(journey, "matching"):
        match_cases(journey, cases, models, readers, corpus["matching"])
    with stage(journey, "queries"):
        # Retrieval uses an observed token; acoustic scoring always uses the
        # independently frozen transcript and never changes to fit this query.
        tokens = ref.words(document_text(journey.current_record(cases[0]["media_id"])))
        if not tokens:
            raise ValueError("Real recognition has no queryable text")
        search = max(tokens, key=len)
        queries = query_snapshot(journey, search)
        journey.receipt["query_operations"] = list(queries)
    with stage(journey, "timeline"):
        timeline = timeline_snapshot(journey, cases)
        journey.receipt["timeline_cue_counts"] = {key: len(rows) for key, rows in timeline.items()}
    with stage(journey, "playback"):
        playback = playback_snapshot(journey, cases)
        journey.receipt["playback"] = playback
    with stage(journey, "fetch"):
        for profile in profiles:
            profile["files"] = fetched_files(journey, profile["version_id"], journey.base / ("fetched-source-" + profile["reader"]))
    with stage(journey, "restore"):
        authority = portable_authority(journey, cases, profiles)
        journey.idle_runtime()
        bundle = journey.base / "backup"
        journey.command("backup", "create", bundle, "--mode", "self-contained")
        journey.command("backup", "verify", bundle)
        # The existing restricted withdrawal helper refuses arbitrary paths.
        journey.withdraw_source()
        journey.receipt["source_store"] = "unavailable at original path"
        journey.workspace = journey.base / "restored-workspace"
        journey.command("workspace", "init", journey.workspace, workspace=False)
        journey.command("backup", "restore", bundle)
        journey.start_runtime()
        require_equal(authority, portable_authority(journey, cases, profiles), "current authority")
        require_equal(queries, query_snapshot(journey, search), "query items")
        require_equal(timeline, timeline_snapshot(journey, cases), "timeline cue positions")
        require_equal(playback, playback_snapshot(journey, cases), "playback bytes and authority")
        for profile in profiles:
            files = fetched_files(journey, profile["version_id"], journey.base / ("fetched-restored-" + profile["reader"]))
            require_equal(profile["files"], files, "retrieved profile files")
        readiness = journey.input("processing", "readiness", value={
            "recognition_model_id": models["transcribe"], "diarization_model_id": models["diarize"],
            "recognition_device": "cpu", "diarization_device": "cpu"})
        journey.receipt["restored_task_prerequisites"] = {
            name: {"state": task["state"], "usable": task["usable"]} for name, task in readiness["tasks"].items()}
        journey.receipt["post_restore_checks"] = {key: True for key in ("authority", "queries", "timeline", "playback", "fetched_models")}
        journey.receipt["authority_sha256"] = __import__("hashlib").sha256(json.dumps(authority, sort_keys=True).encode()).hexdigest()
    finalize_runtime(journey)
    journey.receipt["qualification_seconds"] = time.monotonic() - qualification_started
    journey.receipt["combined_seconds"] = time.monotonic() - journey.started
    journey.receipt["quality_limits"] = [
        "Five independently transcribed short public clips from three readers; not a representative accuracy estimate.",
        "Sequence reuses held-out speech: totals are scenario-weighted, not unique-word statistics.",
        "Reader windows include silence; no independent speech-activity annotation or DER claim.",
        "Profile enrollment derives embeddings; it does not retrain base-model weights.",
        "Restore checks existing accepted results and prerequisites; fresh restored inference is not claimed.",
    ]
    return journey.receipt
