#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Deterministic worker/maintainer guard tests. Never import inference engines."""
import importlib.util
import io
import json
import os
from pathlib import Path
import struct
import sys
import tempfile
import unittest
from unittest.mock import patch
import wave
from fractions import Fraction
from decimal import Decimal

ROOT = Path(__file__).resolve().parents[1]


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, ROOT / "scripts" / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class WorkerTests(unittest.TestCase):
    def test_context_is_exact_bounded_and_applied_each_window(self):
        worker = load("processing_hints", "processing-worker.py")
        hints = ["Exact Name", "R&D"]
        request = {"hints": hints, "context_digest": worker.hints_digest(hints)}
        self.assertEqual(worker.recognition_arguments(request)["hotwords"], "Exact Name, R&D")
        self.assertFalse(worker.recognition_arguments(request)["condition_on_previous_text"])
        self.assertNotIn("initial_prompt", worker.recognition_arguments(request))
        self.assertEqual(worker.hints_digest([]), "4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945")
        for invalid in (["x" * 201], ["name", "name"], [" name"], ["name\n"], [""], ["é" * 101]):
            with self.assertRaises(worker.WorkerError):
                worker.validate_hints({"hints": invalid})
        with self.assertRaises(worker.WorkerError):
            worker.validate_hints({"hints": hints, "context_digest": "0" * 64})
    def test_maintainer_child_environment_excludes_credentials(self):
        qualifier = load("processing_clean_qualification", "qualify-processing.py")
        selected = {name: "" for name in qualifier.CI_VARIABLES}
        selected.update(INSONIC_SESSION_CREDENTIALS="fake-session-value", OPENAI_API_KEY="fake-provider-value",
                        ARBITRARY_PROVIDER_TOKEN="fake-arbitrary-value", HF_TOKEN="fake-hf-value",
                        PYTHONPATH="fake-python-path", HOME="fake-ambient-home", USERPROFILE="fake-ambient-home", LANG="C")
        with patch.dict(os.environ, selected):
            actual = json.loads(qualifier.child([sys.executable, "-I", "-B", "-c",
                "import os,json;print(json.dumps({'keys':list(os.environ),'home_is_directory':os.path.isdir(os.environ.get('HOME','')),**{name:os.environ.get(name) for name in ('HOME','USERPROFILE','LANG','HF_HUB_OFFLINE','TRANSFORMERS_OFFLINE','TOKENIZERS_PARALLELISM')}}))"], timeout=10))
        for name in ("INSONIC_SESSION_CREDENTIALS", "OPENAI_API_KEY", "ARBITRARY_PROVIDER_TOKEN", "HF_TOKEN", "PYTHONPATH"):
            self.assertFalse(name in actual["keys"], "Unexpected inherited credential/configuration key: " + name)
        self.assertEqual(actual["LANG"], "C")
        self.assertEqual(actual["HF_HUB_OFFLINE"], "1")
        self.assertEqual(actual["TRANSFORMERS_OFFLINE"], "1")
        self.assertEqual(actual["TOKENIZERS_PARALLELISM"], "false")
        self.assertTrue(Path(actual["HOME"]).is_absolute())
        self.assertTrue(actual["home_is_directory"])
        self.assertEqual(actual["HOME"], actual["USERPROFILE"])
        self.assertFalse(actual["HOME"] == "fake-ambient-home")
        self.assertFalse(Path(actual["HOME"]).exists(), "Private child home survived cleanup")

    def test_maintainer_private_home_is_cleaned_after_child_failure(self):
        qualifier = load("processing_failed_home", "qualify-processing.py")
        original = qualifier.tempfile.TemporaryDirectory
        allocated = []
        def tracked(*args, **kwargs):
            temporary = original(*args, **kwargs)
            allocated.append(temporary.name)
            return temporary
        with patch.dict(os.environ, {name:"" for name in qualifier.CI_VARIABLES}), patch.object(qualifier.tempfile, "TemporaryDirectory", tracked):
            with self.assertRaisesRegex(ValueError, "engine_failed"):
                qualifier.child([sys.executable, "-I", "-B", "-c", "raise SystemExit(1)"], timeout=10)
            with self.assertRaises(qualifier.subprocess.TimeoutExpired):
                qualifier.child([sys.executable, "-I", "-B", "-c", "import time;time.sleep(30)"], timeout=0.1)
        self.assertEqual(len(allocated), 2)
        self.assertTrue(all(not Path(home).exists() for home in allocated), "Failed or timed-out child left private home")

    def test_maintainer_ci_guard_runs_before_child_creation(self):
        qualifier = load("processing_child_guard", "qualify-processing.py")
        with patch.dict(os.environ, {"CI":"true"}), patch.object(qualifier.subprocess, "Popen") as launch:
            with self.assertRaisesRegex(ValueError, "CI"):
                qualifier.child([sys.executable, "-c", "raise SystemExit(0)"])
            launch.assert_not_called()

    def test_import_is_lazy(self):
        before = set(sys.modules)
        load("processing_lazy", "processing-worker.py")
        self.assertFalse({"torch", "faster_whisper", "pyannote.audio"} & (set(sys.modules) - before))

    def test_ci_refuses_before_engine_initialization(self):
        worker = load("processing_guard", "processing-worker.py")
        with patch.dict(os.environ, {"CI": "true"}), patch("sys.stdin", io.StringIO("{}")):
            with self.assertRaisesRegex(worker.WorkerError, "engine_ci_forbidden"):
                worker.main()

    def test_maintainer_qualification_requires_explicit_opt_in(self):
        qualifier = load("qualifier_guard", "qualify-processing.py")
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaisesRegex(ValueError, "explicit"):
                qualifier.require_election(False)
        for variable in ("CI", "GITHUB_ACTIONS", "TF_BUILD", "BUILD_BUILDID"):
            with patch.dict(os.environ, {variable: "true"}, clear=True):
                with self.assertRaisesRegex(ValueError, "CI"):
                    qualifier.require_election(True)

    def test_wav_is_real_pcm_and_bounded(self):
        worker = load("processing_wave", "processing-worker.py")
        with tempfile.TemporaryDirectory() as directory:
            audio = Path(directory) / "input.wav"
            with wave.open(str(audio), "wb") as writer:
                writer.setparams((1, 2, 16000, 0, "NONE", "not compressed"))
                writer.writeframes(struct.pack("<4h", 0, 320, -320, 0))
            pcm, count, rate = worker.read_audio(audio, 1000000)
            self.assertEqual((count, rate), (4, 16000))
            self.assertEqual(len(pcm), 8)
            with self.assertRaisesRegex(worker.WorkerError, "audio_limit"):
                worker.read_audio(audio, 1)

    def test_timing_validation_and_real_srt_projection(self):
        worker = load("processing_normalize", "processing-worker.py")
        result = worker.recognition_result([
            {"start": 0.25, "end": 1.75, "text": "Hello, world."},
        ], 2000000)
        self.assertIn("00:00:00,250 --> 00:00:01,750", result["srt"])
        self.assertIn("Hello, world.", result["srt"])
        self.assertFalse(result["no_speech"])
        for segment in (
            {"start": float("nan"), "end": 1, "text": "bad"},
            {"start": -1, "end": 1, "text": "bad"},
            {"start": 2, "end": 1, "text": "bad"},
            {"start": 0, "end": 3, "text": "bad"},
        ):
            with self.assertRaises(worker.WorkerError):
                worker.recognition_result([segment], 2000000)
        self.assertTrue(worker.recognition_result([], 2000000)["no_speech"])

    def test_overlap_is_preserved_and_invalid_turns_rejected(self):
        worker = load("processing_turns", "processing-worker.py")
        turns = worker.diarization_result([
            (0.1, 1.3, "speaker_A"), (0.9, 1.9, "speaker_B")
        ], 2000000)
        self.assertEqual(len(turns["turns"]), 2)
        self.assertLess(turns["turns"][1]["start_us"], turns["turns"][0]["end_us"])
        with self.assertRaises(worker.WorkerError):
            worker.diarization_result([(0, float("inf"), "speaker_A")], 2000000)

    def test_whisper_end_overhang_is_explicit_bounded_and_not_generic(self):
        worker = load("processing_whisper_boundary", "processing-worker.py")
        segment = {"start": 5.36, "end": 8.08, "text": "observed final segment"}
        result = worker.recognition_result([segment], 8000000, intersect_whisper_end=True)
        self.assertIn("00:00:05,360 --> 00:00:08,000", result["srt"])
        self.assertEqual(result["diagnostics"], [{"code": "whisper_end_intersected_decoded_media", "count": 1, "value": 0.08}])
        with self.assertRaises(worker.WorkerError): worker.recognition_result([segment], 8000000)
        for start, end in ((0, 8.250001), (8, 8.08), (-0.01, 1), (1, float('inf'))):
            with self.assertRaises(worker.WorkerError):
                worker.recognition_result([{"start": start, "end": end, "text": "invalid"}], 8000000, intersect_whisper_end=True)

    def test_source_origin_is_exact_and_conservatively_projected(self):
        worker = load("processing_source_clock", "processing-worker.py")
        result = worker.recognition_result([{"start": 0.25, "end": 1.75, "text": "hello"}], 2000000, Fraction(96001, 48000))
        self.assertIn("00:00:02,251 --> 00:00:03,750", result["srt"])
        result = worker.recognition_result([{"start": 0, "end": 1, "text": "hello"}], 1000000, Fraction(-1, 2))
        self.assertEqual(result["srt"], "")
        self.assertFalse(result["no_speech"])
        self.assertEqual(result["diagnostics"][0]["code"], "negative_source_interval_unrepresentable")

    def test_request_rejects_unknown_fields_and_remote_model(self):
        worker = load("processing_request", "processing-worker.py")
        with self.assertRaises(worker.WorkerError):
            worker.validate_request({"operation": "transcribe", "extra": True})
        with self.assertRaises(worker.WorkerError):
            worker.validate_request({"operation": "transcribe", "model_path": "Systran/faster-whisper-tiny"})

    def test_embedding_batch_requires_bounded_exact_local_paths(self):
        worker = load("processing_batch_request", "processing-worker.py")
        with tempfile.TemporaryDirectory() as directory:
            audio = Path(directory) / "input.wav"
            audio.write_bytes(b"fixture")
            model = Path(directory) / "model"
            model.mkdir()
            request = {"operation": "embed-batch", "audio_path": str(audio), "audio_paths": [str(audio)], "model_path": str(model)}
            worker.validate_request(request)
            for paths in ([], [str(audio)] * 65, ["relative.wav"], [str(model / "missing.wav")]):
                with self.assertRaises(worker.WorkerError):
                    worker.validate_request({**request, "audio_paths": paths})
            with self.assertRaises(worker.WorkerError):
                worker.validate_request({**request, "operation": "embed"})

    def test_duplicate_json_and_srt_delimiter_are_rejected(self):
        worker = load("processing_strict", "processing-worker.py")
        with self.assertRaises(worker.WorkerError):
            worker.read_request(io.StringIO('{"operation":"transcribe","operation":"diarize"}'))
        with self.assertRaises(worker.WorkerError):
            worker.recognition_result([
                {"start": 0, "end": 1, "text": "hello\n\n2\n00:00:02,000 --> 00:00:03,000\nforged"},
            ], 1000000)

    def test_pyannote_frame_padding_is_diagnosed_and_bounded(self):
        worker = load("processing_padding", "processing-worker.py")
        turns, diagnostics = worker.intersect_pyannote_media([(7.92846875, 10.91534375, "speaker")], 10800000)
        self.assertEqual(turns[0][1], Decimal("10.8"))
        self.assertEqual(diagnostics[0]["code"], "model_frame_intersected_decoded_media")
        with self.assertRaises(worker.WorkerError):
            worker.intersect_pyannote_media([(0, 20, "speaker")], 10800000)
        self.assertEqual(worker.minimum_embedding_samples(lambda n: n >= 37, 1000), 37)


if __name__ == "__main__":
    unittest.main()
