# SPDX-License-Identifier: Apache-2.0
"""Exercise the actual bundled probe without installing engine dependencies."""
import importlib.util
import json
import hashlib
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("worker_probe", ROOT / "internal/workers/probe.py")
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)

class WorkerProbeDiagnostics(unittest.TestCase):
    def test_distribution_inventory_matches_canonical_files_with_directory_work(self):
        with tempfile.TemporaryDirectory(prefix="probe paths ") as directory:
            root = Path(directory)
            for name in ("one", "nested unicode \u96ea"):
                parent = root / name
                parent.mkdir()
                for index in range(100):
                    (parent / (str(index) + ".py")).write_bytes(str(index).encode())
            items = [p.relative_to(root) for p in root.rglob("*.py")]
            items += items[:30] + [Path("missing.py"), Path("missing-parent/missing.py")]
            dist = SimpleNamespace(files=items, locate_file=lambda item: root / item)
            expected = {str((root / item).resolve()) for item in items if (root / item).is_file()}
            paths, observation = set(), probe.Observation(True, None)
            original = Path.resolve
            calls = []
            def resolving(path, *args, **kwargs):
                calls.append(path)
                return original(path, *args, **kwargs)
            with patch.object(Path, "resolve", resolving):
                probe.inventory_distributions(paths, [dist], observation)
            self.assertEqual(paths, expected)
            self.assertLessEqual(len(calls), 6, "canonicalization repeated for ordinary file leaves")

    def test_distribution_links_and_parent_aliases_keep_canonical_identity(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            target = root / "real"
            target.mkdir()
            (target / "data.py").write_bytes(b"independent bytes")
            try:
                (root / "parent-alias").symlink_to(target, target_is_directory=True)
                (target / "file-alias.py").symlink_to(target / "data.py")
            except OSError:
                self.skipTest("file/directory symlinks unavailable on this host")
            items = [Path("real/data.py"), Path("parent-alias/data.py"), Path("real/file-alias.py")]
            dist = SimpleNamespace(files=items, locate_file=lambda item: root / item)
            paths = set()
            probe.inventory_distributions(paths, [dist], probe.Observation(True, None))
            self.assertEqual(paths, {str((target / "data.py").resolve())})

    def test_distribution_inventory_keeps_limit_and_unreadable_parent_failures(self):
        class Full(set):
            def __len__(self):
                return 30001
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "file.py").write_bytes(b"bytes")
            dist = SimpleNamespace(files=[Path("file.py")], locate_file=lambda item: root / item)
            with self.assertRaises(probe.ProbeFailure) as caught:
                probe.inventory_distributions(Full(), [dist], probe.Observation(True, None))
            self.assertEqual(caught.exception.reason, "inventory_limit")
            with patch.object(probe.os, "scandir", side_effect=PermissionError("private path")):
                with self.assertRaises(PermissionError):
                    probe.inventory_distributions(set(), [dist], probe.Observation(True, None))

    @unittest.skipUnless(os.name == "nt", "actual Windows short filename alias")
    def test_distribution_short_leaf_alias_matches_canonical_inventory(self):
        import ctypes
        with tempfile.TemporaryDirectory(prefix="long worker directory ") as directory:
            target = Path(directory) / "a long dependency filename.py"
            target.write_bytes(b"exact alias bytes")
            buffer = ctypes.create_unicode_buffer(32768)
            if not ctypes.windll.kernel32.GetShortPathNameW(str(target), buffer, len(buffer)):
                self.skipTest("Windows short paths unavailable")
            alias = Path(buffer.value)
            if alias.name == target.name:
                self.skipTest("Windows short leaf names disabled on this volume")
            dist = SimpleNamespace(files=[Path(alias.name)], locate_file=lambda item: alias.parent / item)
            paths = set()
            probe.inventory_distributions(paths, [dist], probe.Observation(True, None))
            self.assertEqual(paths, {str(target.resolve())})

    def test_snapshot_hashes_empty_small_and_large_files_with_one_buffer(self):
        with tempfile.TemporaryDirectory() as directory:
            paths, expected, total = set(), {}, 0
            for index, content in enumerate((b"", b"abc", b"large" * 100000)):
                path = Path(directory) / str(index)
                path.write_bytes(content)
                paths.add(str(path))
                expected[str(path)] = hashlib.sha256(content).hexdigest()
                total += len(content)
            observation = probe.Observation(True, None)
            with patch.object(probe, "bytearray", wraps=bytearray, create=True) as allocated:
                result = probe.hash_inventory(paths, observation)
            self.assertEqual(result, [{"path": name, "sha256": expected[name]} for name in sorted(paths)])
            self.assertEqual(allocated.call_count, 1)
            self.assertEqual((observation.value["hashed_files"], observation.value["hashed_bytes"]), (3, total))
            Path(next(iter(paths))).write_bytes(b"changed")
            self.assertNotEqual(probe.hash_inventory(paths, probe.Observation(True, None)), result)

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
                 patch.object(probe, "hash_inventory", side_effect=OSError("secret-token private/path")):
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
            self.assertEqual([p["stage"] for p in result["probe"]["checkpoints"]],
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
