# SPDX-License-Identifier: Apache-2.0
"""Exercise the actual bundled probe without installing engine dependencies."""
import importlib.util
import json
from pathlib import Path
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("worker_probe", ROOT / "internal/workers/probe.py")
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)

class WorkerProbeDiagnostics(unittest.TestCase):
    def test_version_mismatch_has_fixed_stage_and_reason(self):
        with patch.object(probe.metadata, "version", return_value="wrong-secret-token"):
            result = probe.inspect("recognition", False)
        self.assertFalse(result["engine_available"])
        self.assertEqual(result.get("probe", {}).get("reason"), "version_mismatch")
        self.assertEqual(result["probe"]["stage"], "package_versions")
        self.assertNotIn("secret-token", json.dumps(result["probe"]))

    def test_missing_package_is_distinct_and_redacted(self):
        with patch.object(probe.metadata, "version", side_effect=probe.metadata.PackageNotFoundError("secret-token")):
            result = probe.inspect("recognition", False)
        self.assertEqual(result.get("probe", {}).get("reason"), "package_missing")
        self.assertEqual(result["probe"]["stage"], "package_versions")
        self.assertNotIn("secret-token", json.dumps(result["probe"]))


    def ready(self):
        from contextlib import ExitStack
        from types import SimpleNamespace
        import sys
        stack = ExitStack()
        stack.enter_context(patch.object(probe.metadata, "version", side_effect=lambda name: {
            "faster-whisper": "1.2.1", "ctranslate2": "4.8.2", "av": "18.1.0"}[name]))
        stack.enter_context(patch.dict(sys.modules, {
            "av": SimpleNamespace(),
            "ctranslate2": SimpleNamespace(get_supported_compute_types=lambda _: {"int8"}, get_cuda_device_count=lambda: 0),
            "faster_whisper": SimpleNamespace(WhisperModel=object),
        }))
        return stack

    def test_import_failure_has_no_external_text(self):
        import builtins
        original = builtins.__import__
        def importing(name, *args, **kwargs):
            if name == "av":
                raise RuntimeError("secret-token private/path\n\x1b[31m")
            return original(name, *args, **kwargs)
        with self.ready(), patch.object(builtins, "__import__", side_effect=importing):
            result = probe.inspect("recognition", False)
        self.assertEqual(result["probe"]["reason"], "import_failed")
        self.assertEqual(result["probe"]["stage"], "library_import")
        self.assertNotIn("secret-token", json.dumps(result))
        self.assertNotIn("private/path", json.dumps(result))

    def test_cpu_prerequisite_is_distinct_from_import(self):
        import sys
        with self.ready():
            sys.modules["ctranslate2"].get_supported_compute_types = lambda _: set()
            result = probe.inspect("recognition", False)
        self.assertEqual(result["probe"]["reason"], "cpu_unavailable")
        self.assertEqual(result["probe"]["stage"], "cpu_prerequisites")

    def test_snapshot_hash_failure_retains_only_fixed_facts(self):
        import tempfile
        import sys
        with tempfile.TemporaryDirectory() as directory:
            exe = Path(directory) / "python.exe"
            exe.write_bytes(b"fixture")
            with self.ready(), patch.object(probe.metadata, "distributions", return_value=[]), \
                 patch.object(sys, "executable", str(exe)), patch.object(sys, "_base_executable", str(exe), create=True), \
                 patch.object(sys, "prefix", directory), patch.object(sys, "base_prefix", directory), \
                 patch.object(probe.hashlib, "file_digest", side_effect=OSError("secret-token private/path")):
                result = probe.inspect("recognition", True)
        self.assertEqual(result["probe"]["reason"], "inventory_unreadable")
        self.assertEqual(result["probe"]["stage"], "dependency_hashing")
        self.assertFalse(result["engine_available"])
        self.assertEqual(result["files"], [])
        self.assertNotIn("secret-token", json.dumps(result["probe"]))

    def test_inventory_walk_errors_are_not_silently_omitted(self):
        def broken(*args, **kwargs):
            kwargs["onerror"](OSError("secret-token"))
            return []
        with self.ready(), patch.object(probe.metadata, "distributions", return_value=[]), patch.object(probe.os, "walk", side_effect=broken):
            result = probe.inspect("recognition", True)
        self.assertEqual(result["probe"]["reason"], "inventory_unreadable")
        self.assertEqual(result["probe"]["stage"], "dependency_inventory")

    def test_atomic_checkpoint_is_bounded_and_matches_terminal_output(self):
        import tempfile
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / "probe-progress.json"
            with self.ready():
                result = probe.inspect("recognition", False, str(target))
            raw = target.read_bytes()
            self.assertLessEqual(len(raw), 16384)
            self.assertEqual(json.loads(raw), result["probe"])
            self.assertEqual(result["probe"]["reason"], "success")
            self.assertEqual([p["stage"] for p in result["probe"]["observations"]],
                             ["package_versions", "library_import", "cpu_prerequisites", "cuda_prerequisites", "complete"])
            self.assertFalse(target.with_name(target.name + ".tmp").exists())
            self.assertNotIn(directory, json.dumps(result["probe"]))

    def test_unwritable_checkpoint_cannot_report_success(self):
        import tempfile
        with tempfile.TemporaryDirectory() as directory:
            with self.ready():
                result = probe.inspect("recognition", False, directory)
        self.assertEqual(result["probe"]["reason"], "checkpoint_invalid")
        self.assertFalse(result["engine_available"])


if __name__ == "__main__":
    unittest.main()
