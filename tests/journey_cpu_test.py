# SPDX-License-Identifier: Apache-2.0
"""Managed real-journey orchestration tests. No engine imports or downloads."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

ROOT = Path(__file__).resolve().parents[1]


def load(name):
    spec = importlib.util.spec_from_file_location("tested_" + name.replace("-", "_"), ROOT / "scripts" / (name + ".py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class FakeJourney:
    def __init__(self):
        self.args = SimpleNamespace(recognition_model=str(ROOT / "recognition"),
                                    diarization_model=str(ROOT / "diarization"), managed_offline_assets=None)
        self.receipt = {"stages": []}
        self.calls = []
        self.prepared = set()
        self.acquired = set()
        self.revision = 0
        self.prerequisite = False
        self.verified = True

    def input(self, *args, value):
        value = copy.deepcopy(value)
        self.calls.append((".".join(args), value))
        if args == ("workers", "plan"):
            role = value["role"]
            return {"election": {**value, "expected_revision": str(self.revision),
                                "profile_digest": ("1" if role == "recognition" else "2") * 64},
                    **({"prerequisite": "unavailable"} if self.prerequisite else {}),
                    **({"verified_existing_environment": {"mode": "managed"}} if role in self.prepared else {})}
        if args == ("workers", "setup"):
            self.asserted_election = value
            return {"work_id": value["role"]}
        if args == ("models", "prepare"):
            choice = value["choice"]
            base = {"model_id": "base:" + choice, "manifest_digest": "3" * 64, "total_bytes": 64,
                    "download_bytes": 0, "verified_reuse_bytes": 64}
            if value.get("acquire"):
                self.acquired.add(choice)
                return {**base, "acquisition_work_id": choice}
            return {**base, "state": "verified_available" if choice in self.acquired else "verified_local"}
        if args == ("processing", "readiness"):
            return {"tasks": {name: {"usable": True} for name in ("import", "transcription", "diarization")}}
        raise AssertionError(args)

    def finish(self, work, maximum=180):
        self.calls.append(("wait", {"id": work["work_id"], "maximum": maximum}))
        role = work["work_id"]
        if role in ("recognition", "diarization"):
            self.prepared.add(role)
            self.revision += 1
            return {"role": role, "python": {"sha256": "4" * 64}, "environment_receipt": {"sha256": "5" * 64},
                    "verification": "dependency_bytes_and_library_probe", "inference": "not_run"}
        return {"model_id": "base:" + role}

    def command(self, *args, **kwargs):
        self.calls.append((".".join(args[:2]), {"id": args[2]}))
        if args[:2] == ("models", "verify"):
            return {"verified": self.verified}
        raise AssertionError(args)


class ManagedTests(unittest.TestCase):
    def test_returned_elections_are_used_sequentially_and_reuse_is_verified(self):
        cpu = load("journey_cpu")
        journey = FakeJourney()
        models = cpu.prepare_managed(journey)
        self.assertEqual(set(models), {"transcribe", "diarize"})
        setups = [value for name, value in journey.calls if name == "workers.setup"]
        self.assertEqual([value["role"] for value in setups], ["recognition", "diarization"])
        self.assertEqual([value["expected_revision"] for value in setups], ["0", "1"])
        self.assertTrue(all("profile_digest" in value for value in setups))
        self.assertEqual([stage["state"] for stage in journey.receipt["stages"]], ["succeeded"] * 4)
        self.assertTrue(all(value["maximum"] == 600 for name, value in journey.calls
                            if name == "wait" and value["id"] in ("recognition", "diarization")))
        self.assertEqual(len([name for name, _ in journey.calls if name == "models.verify"]), 2)

    def test_unmet_prerequisite_cannot_enqueue_or_become_success(self):
        cpu = load("journey_cpu")
        journey = FakeJourney()
        journey.prerequisite = True
        with self.assertRaisesRegex(ValueError, "prerequisite"):
            cpu.prepare_managed(journey)
        self.assertNotIn("workers.setup", [name for name, _ in journey.calls])
        self.assertEqual(journey.receipt["stages"][0]["state"], "failed")

    def test_unverified_model_cannot_reach_processing(self):
        cpu = load("journey_cpu")
        journey = FakeJourney()
        journey.verified = False
        with self.assertRaisesRegex(ValueError, "model"):
            cpu.prepare_managed(journey)
        self.assertNotIn("processing.readiness", [name for name, _ in journey.calls])

    def test_reference_mode_is_package_only_and_has_no_external_interpreters(self):
        cpu = load("journey_cpu")
        args = SimpleNamespace(mode="real", package="complete.zip", reference_corpus="manifest.json",
                               recognition_python=None, diarization_python=None, managed_offline_assets=None)
        cpu.validate_election(args)
        for change in ({"mode": "controlled"}, {"package": None}, {"recognition_python": sys.executable}):
            with self.subTest(change=change), self.assertRaisesRegex(ValueError, "reference|package|interpreter"):
                cpu.validate_election(SimpleNamespace(**{**vars(args), **change}))

    def test_ci_rejection_precedes_package_or_engine_creation(self):
        journey = load("qualify-journey")
        with patch.dict(os.environ, {"CI": "true"}), patch.object(journey, "Journey") as constructor:
            with self.assertRaisesRegex(ValueError, "CI"):
                journey.main(["--mode", "real", "--package", "unused.zip", "--package-receipt", "unused.json",
                              "--revision", "a" * 40, "--reference-corpus", "unused-corpus.json",
                              "--recognition-model", "unused", "--diarization-model", "unused", "--output", "unused.json"])
            constructor.assert_not_called()


class OwnershipTests(unittest.TestCase):
    def test_dirty_or_different_reference_harness_stops_before_cli_execution(self):
        journey = load("qualify-journey")
        args = SimpleNamespace(mode="real", package="selected", reference_corpus="selected", timeout=1800)
        selected = (Path(sys.executable), {}, Path(sys.executable), Path(sys.executable), {"revision": "a" * 40})
        for identity in (("b" * 40, False, "c" * 64), ("a" * 40, True, "c" * 64)):
            with self.subTest(identity=identity), tempfile.TemporaryDirectory() as temporary:
                with patch.object(journey, "select_package", return_value=selected), \
                     patch.object(journey, "packaged_environment", return_value={}), \
                     patch.object(journey, "harness_identity", return_value=identity), patch.object(journey.engine, "_child") as child:
                    with self.assertRaisesRegex(ValueError, "clean exact"):
                        journey.Journey(args, Path(temporary))
                    child.assert_not_called()

    def test_cleanup_failure_is_attributed_before_runtime_release(self):
        cpu = load("journey_cpu")
        runner = SimpleNamespace(receipt={"current_stage": "restore"},
                                 idle_runtime=Mock(side_effect=ValueError("first release failure")),
                                 assert_harness_unchanged=Mock())
        with self.assertRaisesRegex(ValueError, "first release failure"):
            cpu.finalize_runtime(runner)
        self.assertEqual(runner.receipt["first_failure_stage"], "cleanup")
        runner.assert_harness_unchanged.assert_not_called()

    def test_reference_source_drift_cannot_retain_success(self):
        journey = load("qualify-journey")
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary)
            for name in ("internal", "cmd", "scripts", "schemas/v1.0.1", "tests/fixtures"):
                (source / name).mkdir(parents=True, exist_ok=True)
            for name, data in {"VERSION": "1.0.1\n", "go.mod": "fixture", "go.sum": "fixture",
                               "tests/fixtures/qualification-worker.py": "fixture", "scripts/elected.py": "original"}.items():
                (source / name).write_text(data, encoding="utf-8")
            def git(args, *unused, **options):
                return b"a" * 40 if "rev-parse" in args else b""
            with patch.object(journey, "ROOT", source), patch.object(journey.engine, "_child", side_effect=git):
                revision, dirty, digest = journey.harness_identity()
                self.assertFalse(dirty)
                runner = journey.Journey.__new__(journey.Journey)
                runner.receipt = {"harness_revision": revision, "harness_inputs_sha256": digest}
                runner.assert_harness_unchanged()
                (source / "scripts/elected.py").write_text("changed", encoding="utf-8")
                with self.assertRaisesRegex(ValueError, "changed"):
                    runner.assert_harness_unchanged()
                self.assertIs(runner.receipt["harness_unchanged"], False)

    def test_receipt_replacement_failure_preserves_prior_file(self):
        journey = load("qualify-journey")
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / "receipt.json"
            output.write_text("prior receipt", encoding="utf-8")
            with patch.object(Path, "replace", side_effect=OSError("replacement failed")):
                with self.assertRaises(OSError):
                    journey.write_json(output, {"state": "passed"})
            self.assertEqual(output.read_text(encoding="utf-8"), "prior receipt")
            self.assertEqual(list(Path(temporary).iterdir()), [output])

    def test_profile_temp_paths_are_confined_alongside_appdata(self):
        journey = load("qualify-journey")
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary)
            tools = base / "tools.json"
            tools.write_text("{}", encoding="utf-8")
            args = SimpleNamespace(mode="real", package=None, cli=sys.executable, tools=str(tools),
                                   cueson=sys.executable, timeout=900)
            with patch.object(journey.engine, "_child", side_effect=[b"a" * 40, b""]):
                selected = journey.Journey(args, base)
            for name in ("TEMP", "TMP", "TMPDIR"):
                self.assertTrue(Path(selected.environment[name]).is_relative_to(base))
                self.assertTrue(Path(selected.environment[name]).is_dir())


class DescendantTests(unittest.TestCase):
    @unittest.skipUnless(os.name == "nt", "actual Windows descendant accounting")
    def test_normal_parent_return_observes_natural_descendant_exit(self):
        import time
        qualifier = load("qualify-processing")
        with tempfile.TemporaryDirectory() as temporary:
            marker = Path(temporary) / "naturally-finished"
            worker = "import pathlib,sys,time;time.sleep(.4);pathlib.Path(sys.argv[1]).write_text('natural')"
            parent = ("import subprocess,sys;subprocess.Popen([sys.executable,'-I','-B','-c',sys.argv[1],sys.argv[2]],"
                      "stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,"
                      "creationflags=subprocess.CREATE_NO_WINDOW)")
            try:
                qualifier._child([sys.executable, "-I", "-B", "-c", parent, worker, marker], dict(os.environ), timeout=5)
                self.assertTrue(marker.exists(), "Parent return did not establish descendant release")
            finally:
                deadline = time.monotonic() + 5
                while not marker.exists() and time.monotonic() < deadline:
                    time.sleep(.02)

    def test_cleanup_failure_cannot_replace_the_first_operation_failure(self):
        from unittest.mock import Mock
        journey = load("qualify-journey")
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / "failed.json"
            runtime = Mock()
            runtime.wait.side_effect = OSError("secondary cleanup failure")
            runner = SimpleNamespace(run=Mock(side_effect=ValueError("first operation failure")),
                                     receipt={"mode": "controlled"}, started=journey.time.monotonic(), runtime=runtime)
            args = ["--mode", "controlled", "--cli", "fixture", "--tools", "fixture", "--cueson", "fixture",
                    "--recognition-python", "fixture", "--diarization-python", "fixture", "--output", str(output)]
            with patch.object(journey, "require_mode"), patch.object(journey, "Journey", return_value=runner):
                with self.assertRaisesRegex(ValueError, "first operation failure"):
                    journey.main(args)
            receipt = json.loads(output.read_text())
            self.assertEqual(receipt["failure_type"], "ValueError")
            self.assertNotIn("secondary cleanup failure", output.read_text())


class ResultContracts(unittest.TestCase):
    def test_quality_diagnostics_retain_numeric_bounds_without_freeform_content(self):
        cpu = load("journey_cpu")
        values = [{"code": "model_frame_outside_decoded_media", "count": 1, "value": .279719,
                   "message": "private freeform content"}]
        self.assertEqual(cpu.quality_diagnostics(values), [
            {"code": "model_frame_outside_decoded_media", "count": 1, "value": .279719}])
        for invalid in (None, [{"code": "bad code"}], [{"code": "valid", "count": True}],
                        [{"code": "valid", "value": float("nan")}], [{"code": "valid", "value": "private"}]):
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                cpu.quality_diagnostics(invalid)

    def test_package_construction_failure_remains_identified_when_receipt_write_fails(self):
        journey = load("qualify-journey")
        arguments = ["--mode", "real", "--package", "fixture", "--package-receipt", "fixture", "--revision", "a" * 40,
                     "--reference-corpus", "fixture", "--recognition-model", "fixture", "--diarization-model", "fixture",
                     "--output", "fixture"]
        with patch.object(journey, "require_mode"), patch.object(journey, "Journey", side_effect=ValueError("package byte identity")), \
             patch.object(journey, "write_json", side_effect=OSError("receipt unavailable")) as writer:
            with self.assertRaisesRegex(ValueError, "package byte identity"):
                journey.main(arguments)
        receipt = writer.call_args.args[1]
        self.assertEqual(receipt["state"], "failed")
        self.assertEqual(receipt["first_failure_stage"], "package")
        self.assertEqual(receipt["stages"], [{"name": "package", "state": "failed"}])

    def test_complete_receipt_rejects_missing_cases_forged_scores_and_totals(self):
        import copy
        cpu = load("journey_cpu")
        ref = cpu.references()
        corpus = ref.load_corpus(ROOT / "tests/fixtures/acoustic/manifest.json")
        readers = corpus["selection"]["enrolled_speakers"]
        receipt = {"kind": "insonic-real-cpu-acceptance-v1", "mode": "real", "device": "cpu",
                   "cli_sha256": "7" * 64, "harness_inputs_sha256": "8" * 64,
                   "preparation_seconds": 1, "qualification_seconds": 2, "combined_seconds": 3,
                   "overall_bound_seconds": 1800,
                   "source_revision": "a" * 40, "harness_revision": "a" * 40, "harness_dirty": False, "harness_unchanged": True,
                   "package": {"revision": "a" * 40, "source_dirty": False, "archive_sha256": "b" * 64, "inventory_sha256": "c" * 64},
                   "stages": [{"name": s, "state": "succeeded"} for s in cpu.REQUIRED_STAGES],
                   "corpus_sha256": cpu.corpus_digest(corpus), "matching_settings": corpus["matching"], "worker_sha256": "d" * 64,
                   "environments": {role: {"profile_digest": "e" * 64, "python_sha256": "f" * 64, "receipt_sha256": "1" * 64,
                                           "verified_reuse": True} for role in ("recognition", "diarization")},
                   "model_inputs": {op: {"model_id": op, "manifest_sha256": "2" * 64, "local_verified_reuse": True}
                                    for op in ("transcribe", "diarize")},
                   "speaker_profiles": [{"reader": r, "speaker_id": "speaker-" + r, "version_id": "version-" + r,
                                         "files": {"embedding.json": "3" * 64}} for r in readers],
                   "cleanup": {"runtime_released": True, "temporary_removed": True},
                   "source_store": "unavailable at original path",
                   "post_restore_checks": {key: True for key in ("authority", "queries", "timeline", "playback", "fetched_models")},
                   "query_operations": ["media-list", "speaker-search", "model-list", "text-search"]}
        clips = {c["id"]: c for c in corpus["clips"]}
        components, cursor = [], 0
        for index, key in enumerate(corpus["sequence"]["clips"]):
            cursor += corpus["sequence"]["gap_samples"] if index else 0
            components.append({"fixture": key, "sha256": clips[key]["sha256"], "start_sample": cursor,
                               "end_sample": cursor + clips[key]["samples"]})
            cursor += clips[key]["samples"]
        receipt["sequence"] = {"sha256": "4" * 64, "sample_rate": 16000, "samples": cursor,
                               "components": components, "gap_samples": corpus["sequence"]["gap_samples"]}
        receipt["cases"] = []
        for clip in [*corpus["clips"], {"id": "sequence", "sha256": "4" * 64,
                    "transcript": " ".join(clips[key]["transcript"] for key in corpus["sequence"]["clips"])}]:
            case = {"fixture": clip["id"], "original_sha256": clip["sha256"], "document_digest": "5" * 64,
                    "canonical_audio_sha256": "6" * 64, "word_error": ref.word_counts(clip["transcript"], clip["transcript"]),
                    "diagnostics": []}
            for op, field in (("transcribe", "transcription"), ("diarize", "diarization")):
                case[field] = {"device": "cpu", "offline": True, "model_id": op, "model_manifest_sha256": "2" * 64,
                               "python_sha256": "f" * 64, "worker_sha256": "d" * 64, "threads": 2,
                               "peak_rss_bytes": 1024, "packages": {"test-only": "1"}, "wall_seconds": 1, "cpu_seconds": .5}
            receipt["cases"].append(case)
        receipt["recognition_totals"] = {key: sum(c["word_error"][key] for c in receipt["cases"]) for key in (
            "reference_words", "generated_words", "substitutions", "deletions", "insertions", "edit_distance")}
        receipt["recognition_totals"]["word_error_rate"] = 0
        receipt["speaker_cases"] = [{"fixture": c["id"], "decisions": [], "missing_prediction": True}
                                    for c in corpus["clips"] if c["role"] != "enrollment"]
        receipt["speaker_cases"].append({"fixture": "sequence", "decisions": [], "missing_prediction": True})
        receipt["speaker_outcomes"] = {**dict.fromkeys(cpu.SPEAKER_OUTCOMES, 0), "missing_prediction": 4}
        receipt["speaker_denominators"] = {"evaluation_cases": 4, "observed_decisions": 0}
        receipt["timeline_cue_counts"] = {c["fixture"]: 1 for c in receipt["cases"]}
        receipt["playback"] = {c["fixture"]: {} for c in receipt["cases"]}
        cpu.validate_acceptance(receipt, corpus)
        for mutation in (
                lambda r: r["cases"].pop(),
                lambda r: r["cases"][0]["word_error"].update(word_error_rate=1),
                lambda r: r["recognition_totals"].update(insertions=1),
                lambda r: r["sequence"]["components"][1].update(start_sample=1),
                lambda r: r["speaker_cases"][0].update(missing_prediction=False),
                lambda r: r["speaker_outcomes"].update(correct_match=4),
                lambda r: r.update(qualification_seconds=float("nan")),
                lambda r: r.update(harness_unchanged=False),
                lambda r: r.update(combined_seconds=1801),
                lambda r: r["cases"][0]["transcription"].pop("peak_rss_bytes"),
                lambda r: r["post_restore_checks"].update(fetched_models=False),
                lambda r: r["cleanup"].update(temporary_removed=False)):
            changed = copy.deepcopy(receipt)
            mutation(changed)
            with self.assertRaises(ValueError):
                cpu.validate_acceptance(changed, corpus)

    def test_playback_requires_current_bytes_document_and_source_identity(self):
        cpu = load("journey_cpu")
        media = {"id": "media", "revision": 2, "digest": "a" * 64}
        record = {"id": "media", "revision": 3, "document_digest": "b" * 64, "source_digest": "a" * 64}
        descriptor = {"media_id": "media", "revision": 2, "digest": "a" * 64, "size_bytes": 12,
                      "source_digest": "a" * 64, "source_size_bytes": 12, "recording_revision": 3,
                      "document_digest": "b" * 64, "preview": False, "mime_type": "audio/flac",
                      "timeline_offset_seconds": 0, "path": "private transient path", "playback_id": "transient"}
        identity = cpu.playback_identity(media, record, descriptor, 12, "a" * 64)
        self.assertNotIn("path", identity)
        self.assertNotIn("playback_id", identity)
        for change in ({"source_digest": "c" * 64}, {"document_digest": "c" * 64}, {"size_bytes": 13},
                       {"recording_revision": 1}, {"timeline_offset_seconds": 1}):
            with self.subTest(change=change), self.assertRaisesRegex(ValueError, "playback"):
                cpu.playback_identity(media, record, {**descriptor, **change}, 12, "a" * 64)

    def test_timeline_cues_retain_exact_current_microseconds(self):
        cpu = load("journey_cpu")
        record = {"id": "media", "document_digest": "b" * 64, "revision": 3,
                  "document": {"cues": [{"id": "cue", "timing": {"start_milliseconds": 25, "end_milliseconds": 900},
                                         "payload": {"plain_text": "Independent text"}}]}}
        row = {"kind": "cue", "media_id": "media", "cue_id": "cue", "document_digest": "b" * 64,
               "recording_revision": 3, "source_start_us": "25000", "source_end_us": "900000",
               "seek_seconds": .025}
        self.assertEqual(cpu.timeline_identity([row], record)[0]["source_start_us"], "25000")
        for rows in ([], [{**row, "source_start_us": "25"}], [{**row, "document_digest": "c" * 64}]):
            with self.subTest(rows=rows), self.assertRaisesRegex(ValueError, "timeline"):
                cpu.timeline_identity(rows, record)

    def test_voice_reference_intervals_use_explicit_attribution_or_cue_fallback(self):
        cpu = load("journey_cpu")
        record = {"document": {"cues": [
            {"timing": {"start_milliseconds": 100, "end_milliseconds": 1000},
             "speaker_attributions": [{"speaker_id": "voice", "start_milliseconds": 300, "end_milliseconds": 700}]},
            {"timing": {"start_milliseconds": 1200, "end_milliseconds": 2000},
             "speaker_attributions": [{"speaker_id": "voice"}]},
        ]}}
        self.assertEqual(cpu.voice_intervals(record, "voice"), [(300000, 700000), (1200000, 2000000)])
        self.assertEqual(cpu.voice_intervals(record, "absent"), [])

    def test_partial_synthetic_or_stale_receipts_cannot_claim_acceptance(self):
        cpu = load("journey_cpu")
        corpus = cpu.references().load_corpus(ROOT / "tests/fixtures/acoustic/manifest.json")
        with self.assertRaisesRegex(ValueError, "receipt"):
            cpu.validate_acceptance({"state": "passed", "mode": "controlled"}, corpus)
        base = {"kind": "insonic-real-cpu-acceptance-v1", "mode": "real", "device": "cpu",
                "cli_sha256": "7" * 64, "worker_sha256": "8" * 64, "harness_inputs_sha256": "9" * 64,
                "source_revision": "a" * 40, "harness_revision": "a" * 40, "harness_dirty": False, "harness_unchanged": True,
                "package": {"revision": "a" * 40, "source_dirty": False, "archive_sha256": "c" * 64, "inventory_sha256": "d" * 64},
                "corpus_sha256": cpu.references().sha(ROOT / "tests/fixtures/acoustic/manifest.json"),
                "stages": [], "cases": [], "speaker_profiles": [], "cleanup": {"runtime_released": True, "temporary_removed": True}}
        with self.assertRaisesRegex(ValueError, "stage"):
            cpu.validate_acceptance(base, corpus)
        complete_stages = [{**base, "stages": [{"name": s, "state": "succeeded"} for s in cpu.REQUIRED_STAGES]}][0]
        with self.assertRaisesRegex(ValueError, "case"):
            cpu.validate_acceptance(complete_stages, corpus)
        with self.assertRaisesRegex(ValueError, "identity"):
            cpu.validate_acceptance({**complete_stages, "source_revision": "b" * 40}, corpus)


if __name__ == "__main__":
    unittest.main()
