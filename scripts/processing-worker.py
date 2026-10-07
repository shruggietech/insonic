#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Explicit offline local inference worker. Importing this file loads no engine."""
from decimal import Decimal, InvalidOperation, ROUND_HALF_UP
from fractions import Fraction
import hashlib
import importlib.metadata
import json
import math
import os
from pathlib import Path
import re
import struct
import sys
import time
import unicodedata
import wave

MAX_REQUEST = 65536
MAX_RESULTS = 20000
CI_VARIABLES = ("CI", "GITHUB_ACTIONS", "TF_BUILD", "BUILD_BUILDID", "JENKINS_URL")


class WorkerError(ValueError):
    pass


def forbid_ci():
    if any(os.environ.get(name) for name in CI_VARIABLES):
        raise WorkerError("engine_ci_forbidden")


def _pairs(items):
    result = {}
    for key, value in items:
        if key in result:
            raise WorkerError("invalid_request")
        result[key] = value
    return result


def read_request(stream):
    raw = stream.read(MAX_REQUEST + 1)
    if len(raw) > MAX_REQUEST:
        raise WorkerError("input_limit")
    try:
        return json.loads(raw, object_pairs_hook=_pairs,
                          parse_constant=lambda value: (_ for _ in ()).throw(WorkerError("invalid_request")))
    except (json.JSONDecodeError, UnicodeError) as error:
        raise WorkerError("invalid_request") from error


def validate_request(request):
    allowed = {"operation", "audio_path", "model_path", "device", "language",
               "min_speakers", "max_speakers", "threads", "max_duration_us",
               "source_start_numerator", "source_start_denominator", "hints", "context_digest"}
    if not isinstance(request, dict) or set(request) - allowed:
        raise WorkerError("invalid_request")
    if request.get("operation") not in {"transcribe", "diarize"}:
        raise WorkerError("invalid_request")
    for key in ("audio_path", "model_path"):
        value = request.get(key)
        if not isinstance(value, str) or not Path(value).is_absolute():
            raise WorkerError("invalid_request")
        if key == "audio_path" and not Path(value).is_file():
            raise WorkerError("audio_unavailable")
        if key == "model_path" and not Path(value).is_dir():
            raise WorkerError("model_unavailable")
    if request.get("device", "cpu") not in {"cpu", "cuda"}:
        raise WorkerError("unsupported_device")
    language = request.get("language", "en")
    if not isinstance(language, str) or not re.fullmatch(r"[a-z]{2,3}|auto", language):
        raise WorkerError("invalid_request")
    for key, lower, upper, default in (("threads", 1, 64, 2),
                                        ("max_duration_us", 1, 604800000000, 3600000000),
                                        ("min_speakers", 0, 64, 0), ("max_speakers", 0, 64, 0)):
        value = request.get(key, default)
        if type(value) is not int or not lower <= value <= upper:
            raise WorkerError("invalid_request")
    if request.get("max_speakers", 0) and request.get("min_speakers", 0) > request["max_speakers"]:
        raise WorkerError("invalid_request")
    for key, default in (("source_start_numerator", "0"), ("source_start_denominator", "1")):
        value = request.get(key, default)
        if not isinstance(value, str) or len(value) > 128 or not re.fullmatch(r"-?[0-9]+", value):
            raise WorkerError("invalid_request")
    if int(request.get("source_start_denominator", "1")) <= 0:
        raise WorkerError("invalid_request")
    validate_hints(request)
    if request["operation"] == "diarize" and request.get("hints"):
        raise WorkerError("unsupported_capability")
    return request


def hints_digest(hints):
    # Match the frozen Go JSON sequence, including its HTML-safe escapes.
    encoded = json.dumps(hints, ensure_ascii=False, separators=(",", ":"))
    for text, escape in (("&", "\\u0026"), ("<", "\\u003c"), (">", "\\u003e"),
                         ("\u2028", "\\u2028"), ("\u2029", "\\u2029")):
        encoded = encoded.replace(text, escape)
    return hashlib.sha256(encoded.encode("utf-8")).hexdigest()


def validate_hints(request):
    hints = request.get("hints", [])
    if not isinstance(hints, list) or len(hints) > 1024:
        raise WorkerError("invalid_request")
    seen = set()
    for hint in hints:
        if not isinstance(hint, str) or not hint or hint != hint.strip() or hint in seen or any(unicodedata.category(c) == "Cc" for c in hint):
            raise WorkerError("invalid_request")
        seen.add(hint)
    try:
        if len(", ".join(hints).encode("utf-8")) > 200:
            raise WorkerError("input_limit")
    except UnicodeError as error:
        raise WorkerError("invalid_request") from error
    digest = hints_digest(hints)
    if request.get("context_digest", digest) != digest:
        raise WorkerError("invalid_request")
    return hints, digest


def recognition_arguments(request):
    hints, _digest = validate_hints(request)
    return {"language": None if request.get("language", "en") == "auto" else request.get("language", "en"),
            "beam_size": 5, "vad_filter": False, "condition_on_previous_text": False,
            "hotwords": ", ".join(hints) if hints else None}


def read_audio(path, max_duration_us):
    try:
        with wave.open(str(path), "rb") as audio:
            if audio.getnchannels() != 1 or audio.getsampwidth() != 2 or audio.getframerate() != 16000 or audio.getcomptype() != "NONE":
                raise WorkerError("unsupported_audio")
            count, rate = audio.getnframes(), audio.getframerate()
            if count * 1000000 > max_duration_us * rate:
                raise WorkerError("audio_limit")
            pcm = audio.readframes(count + 1)
            if len(pcm) != count * 2:
                raise WorkerError("invalid_audio")
            return pcm, count, rate
    except (wave.Error, OSError, EOFError) as error:
        raise WorkerError("invalid_audio") from error


def _us(value):
    if isinstance(value, bool):
        raise WorkerError("invalid_timing")
    try:
        seconds = Decimal(str(value))
        if not seconds.is_finite() or seconds < 0:
            raise WorkerError("invalid_timing")
        return int((seconds * 1000000).quantize(Decimal(1), rounding=ROUND_HALF_UP))
    except (InvalidOperation, ValueError, OverflowError) as error:
        raise WorkerError("invalid_timing") from error


def _interval(start, end, duration_us):
    start, end = _us(start), _us(end)
    if start >= end or end > duration_us:
        raise WorkerError("invalid_timing")
    return start, end


def _stamp(milliseconds):
    seconds, ms = divmod(milliseconds, 1000)
    minutes, seconds = divmod(seconds, 60)
    hours, minutes = divmod(minutes, 60)
    return f"{hours:02d}:{minutes:02d}:{seconds:02d},{ms:03d}"


def recognition_result(segments, duration_us, source_start=Fraction(0)):
    output, diagnostics, count = [], [], 0
    for segment in segments:
        count += 1
        if count > MAX_RESULTS:
            raise WorkerError("output_limit")
        start, end = _interval(segment["start"], segment["end"], duration_us)
        text = segment["text"]
        if not isinstance(text, str) or len(text) > 65536 or any(ord(c) < 32 and c not in "\n\t\r" for c in text):
            raise WorkerError("invalid_text")
        text = text.replace("\r\n", "\n").replace("\r", "\n").strip()
        if "\n\n" in text:
            raise WorkerError("invalid_text")
        if not text:
            continue
        source_start_ms = (source_start + Fraction(start, 1000000)) * 1000
        source_end_ms = (source_start + Fraction(end, 1000000)) * 1000
        start_ms, end_ms = -(-source_start_ms.numerator // source_start_ms.denominator), source_end_ms.numerator // source_end_ms.denominator
        if start_ms < 0 or end_ms < 0:
            diagnostics.append({"code": "negative_source_interval_unrepresentable", "count": 1})
            continue
        if start_ms >= end_ms:
            diagnostics.append({"code": "submillisecond_segment_unrepresentable", "count": 1})
            continue
        output.append(f"{len(output)+1}\n{_stamp(start_ms)} --> {_stamp(end_ms)}\n{text}\n\n")
    return {"srt": "".join(output), "no_speech": count == 0,
            "diagnostics": diagnostics, "provenance": {"segment_count": count,
            "native_timing_projection": "nearest-microsecond;exact-source-origin;ceil-start/floor-end-millisecond"}}


def diarization_result(segments, duration_us):
    turns = []
    for start, end, label in segments:
        if len(turns) >= MAX_RESULTS or not isinstance(label, str) or not label or len(label) > 256 or any(ord(c) < 32 for c in label):
            raise WorkerError("invalid_diarization")
        start, end = _interval(start, end, duration_us)
        turns.append({"label": label, "start_us": start, "end_us": end})
    turns.sort(key=lambda turn: (turn["start_us"], turn["end_us"], turn["label"]))
    diagnostics = []
    events = []
    for turn in turns:
        events.extend(((turn["start_us"], 1), (turn["end_us"], -1)))
    active, last, speech, overlap = 0, 0, 0, 0
    for timestamp, change in sorted(events):
        if active:
            speech += timestamp - last
        if active > 1:
            overlap += timestamp - last
        active += change
        last = timestamp
    if duration_us:
        diagnostics.extend(({"code": "speech_coverage_fraction", "value": speech / duration_us},
                            {"code": "overlap_fraction", "value": overlap / duration_us}))
    short = sum(turn["end_us"] - turn["start_us"] < 250000 for turn in turns)
    if short:
        diagnostics.append({"code": "short_turns", "count": short})
    if not turns:
        diagnostics.append({"code": "no_speech", "count": 1})
    return {"turns": turns, "no_speech": not turns, "diagnostics": diagnostics,
            "provenance": {"speaker_count": len({turn["label"] for turn in turns}),
                           "turn_count": len(turns), "quality_diagnostics_enabled": True}}


def _versions(names):
    try:
        return {name: importlib.metadata.version(name) for name in names}
    except importlib.metadata.PackageNotFoundError as error:
        raise WorkerError("engine_unavailable") from error


def intersect_pyannote_media(segments, duration_us):
    """Intersect pinned segmentation-frame padding with actual decoded media.

    Community-1 receptive frames can extend beyond the waveform. These are
    model windows, not audio outside the source. Diagnose each intersection;
    excursions larger than 250 ms remain invalid, as do generic adapter turns.
    """
    result, diagnostics = [], []
    for start, end, label in segments:
        values = []
        for value in (start, end):
            try:
                number = Decimal(str(value))
                if not number.is_finite():
                    raise WorkerError("invalid_timing")
                values.append(int((number * 1000000).quantize(Decimal(1), rounding=ROUND_HALF_UP)))
            except InvalidOperation as error:
                raise WorkerError("invalid_timing") from error
        first, last = values
        if first >= last or first < -250000 or last > duration_us + 250000:
            raise WorkerError("invalid_timing")
        bounded_first, bounded_last = max(first, 0), min(last, duration_us)
        if bounded_first != first or bounded_last != last:
            diagnostics.append({"code": "model_frame_intersected_decoded_media", "count": 1,
                                "value": (max(0, -first) + max(0, last-duration_us)) / 1000000})
        if bounded_first >= bounded_last:
            diagnostics.append({"code": "model_frame_outside_decoded_media", "count": 1})
            continue
        result.append((Decimal(bounded_first) / 1000000, Decimal(bounded_last) / 1000000, label))
    return result, diagnostics


def minimum_embedding_samples(probe, sample_rate):
    lower, upper = 2, sample_rate // 2
    if upper <= lower or not probe(upper):
        raise WorkerError("invalid_embedding")
    while lower + 1 < upper:
        middle = (lower + upper) // 2
        if probe(middle):
            upper = middle
        else:
            lower = middle
    return upper


def guard_pyannote_embeddings(pipeline, torch, np):
    embedding = pipeline._embedding
    model = getattr(embedding, "model_", None)
    if model is None:
        raise WorkerError("incompatible_version")
    pools = [module for module in model.modules() if type(module).__name__ == "StatsPool"]
    if not pools:
        raise WorkerError("incompatible_version")

    def enough_frames(_module, args, kwargs):
        sequences = args[0] if args else kwargs["sequences"]
        if sequences.shape[-1] < 2:
            raise WorkerError("invalid_embedding")

    for pool in pools:
        pool.register_forward_pre_hook(enough_frames, with_kwargs=True)
    generator = torch.Generator(device="cpu").manual_seed(0)

    def probe(samples):
        waveform = torch.randn(1, 1, samples, generator=generator).to(embedding.device)
        try:
            with torch.inference_mode():
                values = model(waveform)
            return bool(torch.isfinite(values).all())
        except (WorkerError, RuntimeError, ValueError, AssertionError):
            return False

    minimum = minimum_embedding_samples(probe, embedding.sample_rate)
    embedding.__dict__["min_num_samples"] = minimum
    original = pipeline.get_embeddings

    def checked(file, binary_segmentations, **options):
        values = original(file, binary_segmentations, **options)
        activity = np.asarray(binary_segmentations.data)
        if values.ndim != 3 or activity.ndim != 3 or values.shape[:2] != (activity.shape[0], activity.shape[2]) or np.isinf(activity).any():
            raise WorkerError("invalid_embedding")
        active = np.nan_to_num(activity, nan=0.0).sum(axis=1) != 0
        finite = np.isfinite(values).all(axis=2)
        missing = np.isnan(values).all(axis=2)
        if not (finite | (~active & missing)).all():
            raise WorkerError("invalid_embedding")
        norms = np.linalg.norm(values[active & finite], axis=1)
        if not np.isfinite(norms).all() or (norms < 1e-12).any():
            raise WorkerError("invalid_embedding")
        return values

    pipeline.get_embeddings = checked
    return minimum


def _peak_memory():
    if os.name == "nt":
        import ctypes
        from ctypes import wintypes
        class Counters(ctypes.Structure):
            _fields_ = [("cb", wintypes.DWORD), ("PageFaultCount", wintypes.DWORD)] + [
                (name, ctypes.c_size_t) for name in ("PeakWorkingSetSize", "WorkingSetSize", "QuotaPeakPagedPoolUsage", "QuotaPagedPoolUsage", "QuotaPeakNonPagedPoolUsage", "QuotaNonPagedPoolUsage", "PagefileUsage", "PeakPagefileUsage")]
        counters = Counters()
        counters.cb = ctypes.sizeof(counters)
        kernel = ctypes.WinDLL("kernel32", use_last_error=True)
        kernel.GetCurrentProcess.restype = wintypes.HANDLE
        api = ctypes.WinDLL("psapi", use_last_error=True).GetProcessMemoryInfo
        api.argtypes = (wintypes.HANDLE, ctypes.POINTER(Counters), wintypes.DWORD)
        if api(kernel.GetCurrentProcess(), ctypes.byref(counters), counters.cb):
            return counters.PeakWorkingSetSize
        return None
    import resource
    peak = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    return peak if sys.platform == "darwin" else peak * 1024


def run(request):
    forbid_ci()
    request = validate_request(request)
    os.environ.update(HF_HUB_OFFLINE="1", TRANSFORMERS_OFFLINE="1",
                      HF_HUB_DISABLE_TELEMETRY="1", PYANNOTE_METRICS_ENABLED="0",
                      TOKENIZERS_PARALLELISM="false")
    threads = request.get("threads", 2)
    os.environ["OMP_NUM_THREADS"] = str(threads)
    wall, cpu = time.perf_counter(), time.process_time()
    pcm, count, rate = read_audio(request["audio_path"], request.get("max_duration_us", 3600000000))
    duration_us = count * 1000000 // rate
    device = request.get("device", "cpu")
    if request["operation"] == "transcribe":
        versions = _versions(("faster-whisper", "ctranslate2", "av"))
        if versions["faster-whisper"] != "1.2.1" or versions["ctranslate2"] != "4.8.2":
            raise WorkerError("incompatible_version")
        import numpy as np
        from faster_whisper import WhisperModel
        model = WhisperModel(request["model_path"], device=device,
                             compute_type="int8" if device == "cpu" else "float16",
                             cpu_threads=threads, num_workers=1, local_files_only=True)
        audio = np.frombuffer(pcm, dtype="<i2").astype(np.float32) / 32768.0
        segments, info = model.transcribe(audio, **recognition_arguments(request))
        origin = Fraction(int(request.get("source_start_numerator", "0")), int(request.get("source_start_denominator", "1")))
        result = recognition_result(({"start": segment.start, "end": segment.end, "text": segment.text} for segment in segments), duration_us, origin)
        result["provenance"].update(language=info.language, beam_size=5, vad_filter=False,
                                     condition_on_previous_text=False, compute_type="int8" if device == "cpu" else "float16")
        hints, context_digest = validate_hints(request)
        result["provenance"].update(context_digest=context_digest, hint_count=len(hints),
                                     context_application="hotwords-each-decoding-window")
    else:
        versions = _versions(("pyannote.audio", "torch", "torchaudio"))
        if versions["pyannote.audio"] != "4.0.7":
            raise WorkerError("incompatible_version")
        import numpy as np
        import torch
        from pyannote.audio import Pipeline
        torch.set_num_threads(threads)
        if device == "cuda" and not torch.cuda.is_available():
            raise WorkerError("unsupported_device")
        pipeline = Pipeline.from_pretrained(request["model_path"])
        if pipeline is None:
            raise WorkerError("model_unavailable")
        pipeline.to(torch.device(device))
        minimum = guard_pyannote_embeddings(pipeline, torch, np)
        audio = torch.from_numpy(np.frombuffer(pcm, dtype="<i2").astype(np.float32) / 32768.0).reshape(1, -1)
        options = {key: request[key] for key in ("min_speakers", "max_speakers") if request.get(key)}
        with torch.inference_mode():
            output = pipeline({"waveform": audio, "sample_rate": rate}, **options)
        turns, padding_diagnostics = intersect_pyannote_media(((turn.start, turn.end, str(label)) for turn, _, label in output.speaker_diarization.itertracks(yield_label=True)), duration_us)
        result = diarization_result(turns, duration_us)
        result["diagnostics"].extend(padding_diagnostics)
        result["provenance"].update(embedding_minimum_samples=minimum,
                                     model_boundary_policy="Community-1 frame/media intersection;max250ms overhang;diagnose each change")
        embeddings = getattr(output, "speaker_embeddings", None)
        if embeddings is not None and not np.isfinite(embeddings).all():
            raise WorkerError("invalid_embedding")
    with Path(request["audio_path"]).open("rb") as source:
        audio_digest = hashlib.file_digest(source, "sha256").hexdigest()
    result["provenance"].update(packages=versions, device=device, threads=threads,
                                 wall_seconds=time.perf_counter()-wall, cpu_seconds=time.process_time()-cpu,
                                 peak_rss_bytes=_peak_memory(), audio_sha256=audio_digest,
                                 sample_count=count, sample_rate=rate, telemetry_enabled=False, offline=True)
    return result


def main():
    forbid_ci()
    result = run(read_request(sys.stdin))
    encoded = json.dumps(result, ensure_ascii=False, allow_nan=False, separators=(",", ":"))
    if len(encoded.encode("utf-8")) > 16 * 1024 * 1024:
        raise WorkerError("output_limit")
    sys.stdout.write(encoded + "\n")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        # Raw engine exceptions can contain paths, request data or credentials.
        # Return an actionable bounded code without printing their contents.
        code = str(error) if isinstance(error, WorkerError) else "engine_failed"
        sys.stderr.write(json.dumps({"error": code}) + "\n")
        raise SystemExit(1)
