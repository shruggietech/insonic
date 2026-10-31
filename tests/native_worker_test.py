# SPDX-License-Identifier: Apache-2.0
"""Deterministic native-library election; no engine imports or GPU access."""
import hashlib
import importlib.util
import json
import os
import sys
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]


def load_native():
    spec = importlib.util.spec_from_file_location("native_fixture", ROOT / "internal/workers/native.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class NativeWorkerTests(unittest.TestCase):
    def test_linux_existing_elected_files_do_not_probe_conflicting_system_copy(self):
        native = load_native()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            files = [Path(name) for name in ("libcublas.so.12", "libcudnn.so.9", "libnvrtc.so.12")]
            for name in files:
                (root / name).write_bytes(b"elected environment")
            distribution = SimpleNamespace(files=files, locate_file=lambda name: root / name)
            with patch.object(native, "WINDOWS", False), patch.object(native.metadata, "distribution", return_value=distribution), \
                 patch.dict(os.environ, {"LD_LIBRARY_PATH": ""}), patch.object(native, "linux_linker_path", side_effect=AssertionError("unneeded system override")) as resolver:
                self.assertEqual(native.native_files(), sorted((root / name).resolve() for name in files))
                resolver.assert_not_called()

    @unittest.skipUnless(sys.platform == "linux", "Linux system-loader API")
    def test_linux_system_loader_reports_real_absolute_loaded_path(self):
        native = load_native()
        path = native.linux_linker_path("libc.so.6")
        self.assertTrue(path.is_absolute())
        self.assertTrue(path.is_file())
        self.assertEqual(path, path.resolve())

    def test_linux_loader_distinguishes_missing_and_incompatible_library(self):
        native = load_native()
        for message, code in (("libcudnn.so.9: cannot open shared object file: No such file or directory", None),
                              ("libcudnn.so.9: wrong ELF class: ELFCLASS32", "cuda_library_incompatible"),
                              ("libcudnn.so.9: undefined symbol: cudnnGetVersion", "cuda_library_incompatible")):
            with self.subTest(code=code), patch.object(native.ctypes, "CDLL", side_effect=OSError(message)):
                if code:
                    with self.assertRaisesRegex(native.NativeFailure, code):
                        native.linux_linker_path("libcudnn.so.9")
                else:
                    self.assertIsNone(native.linux_linker_path("libcudnn.so.9"))

    def test_linux_discovery_freezes_system_cuda_closure_without_ambient_override(self):
        native = load_native()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            names = ("libcublas.so.12.9.0", "libcublasLt.so.12.9.0", "libcudnn.so.9.1.0", "libcudnn_engines_runtime_compiled.so.9.1.0", "libnvrtc-builtins.so.12.9", "unrelated.so")
            for name in names:
                (root / name).write_bytes(name.encode())
            dependency_directory = root / "separate-linker-directory"
            dependency_directory.mkdir()
            dependency = dependency_directory / "libcublasLt.so.12.8.0"
            dependency.write_bytes(b"mapped dependency from another directory")
            native._linker_cuda_files.add(dependency.resolve())
            resolved = {"libcublas.so.12": root / names[0], "libcudnn.so.9": root / names[2]}
            with patch.object(native, "WINDOWS", False), patch.object(native.metadata, "distribution", side_effect=native.metadata.PackageNotFoundError), \
                 patch.dict(os.environ, {"LD_LIBRARY_PATH": ""}), patch.object(native, "linux_linker_path", side_effect=lambda name: resolved.get(name)) as resolver:
                files = native.native_files()
                self.assertEqual(files, sorted([(root / name).resolve() for name in names[:-1]] + [dependency.resolve()]))
                self.assertEqual(resolver.call_count, 3)
                self.assertEqual(os.environ["LD_LIBRARY_PATH"], "")
                receipt = root / "receipt.json"
                receipt.write_text(json.dumps({"inspection": {"files": [{"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()} for path in files]}}), encoding="utf-8")
                native.set_receipt(str(receipt), hashlib.sha256(receipt.read_bytes()).hexdigest())
                (root / "libcudnn_new_component.so.9").write_bytes(b"not elected")
                resolver.reset_mock()
                self.assertEqual(native.native_files(), files)
                resolver.assert_not_called()

    def test_classifier_never_uses_arbitrary_engine_failure_for_retry(self):
        native = load_native()
        for failure in [ValueError("bad model"), RuntimeError("input path /cuda/private failed"),
                        RuntimeError("invalid argument"), MemoryError("host allocation")]:
            self.assertIsNone(native.cuda_failure(failure))
        self.assertEqual(native.cuda_failure(RuntimeError("CUDA out of memory. Tried to allocate 1 GiB")), "cuda_resource_exhausted")
        self.assertEqual(native.cuda_failure(RuntimeError("CUDA failed with error no CUDA-capable device is detected")), "cuda_no_device")
        self.assertEqual(native.cuda_failure(RuntimeError("CUDA error: no kernel image is available for execution on the device")), "cuda_compute_unsupported")
        self.assertEqual(native.cuda_failure(RuntimeError("CUDA driver version is insufficient for CUDA runtime version")), "cuda_library_incompatible")

    def test_receipt_elects_only_pinned_native_files_and_detects_changes(self):
        native = load_native()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            selected = root / "cublas64_12.dll"
            selected.write_bytes(b"selected")
            newer = root / "cudnn64_9.dll"
            newer.write_bytes(b"unselected")
            receipt = root / "receipt.json"
            receipt.write_text(json.dumps({"inspection": {"files": [{"path": str(selected), "sha256": hashlib.sha256(b"selected").hexdigest()}]}}), encoding="utf-8")
            native.set_receipt(str(receipt), hashlib.sha256(receipt.read_bytes()).hexdigest())
            with patch.object(native, "WINDOWS", True):
                self.assertEqual(native.native_files(), [selected.resolve()])
                selected.write_bytes(b"changed")
                with self.assertRaisesRegex(native.NativeFailure, "checksum_mismatch"):
                    native.native_files()

    def test_windows_absolute_libraries_and_directory_handles_are_retained(self):
        native = load_native()
        with tempfile.TemporaryDirectory() as directory:
            paths = [Path(directory) / name for name in ("cublas64_12.dll", "cudnn64_9.dll")]
            for path in paths:
                path.write_bytes(b"fixture")
            handle = SimpleNamespace(close=lambda: None)
            with patch.object(native, "WINDOWS", True), patch.object(native, "native_files", return_value=paths), \
                 patch.object(native.os, "add_dll_directory", return_value=handle, create=True) as add, \
                 patch.object(native.ctypes, "CDLL", return_value=object()) as load:
                with native.NativeLibraries() as libraries:
                    libraries.recognition(SimpleNamespace(get_cuda_device_count=lambda: 1, get_supported_compute_types=lambda _: {"float16"}))
                    self.assertEqual(libraries.handles, [handle])
                    self.assertEqual([call.args[0] for call in load.call_args_list], [str(path) for path in paths])
                    add.assert_called_once_with(str(Path(directory)))
                self.assertEqual(libraries.handles, [])

    def test_missing_and_incompatible_library_are_distinct(self):
        native = load_native()
        engine = SimpleNamespace(get_cuda_device_count=lambda: 1, get_supported_compute_types=lambda _: {"float16"})
        with patch.object(native, "native_files", return_value=[]):
            with native.NativeLibraries() as libraries:
                with self.assertRaisesRegex(native.NativeFailure, "cuda_library_missing"):
                    libraries.recognition(engine)
                with self.assertRaisesRegex(native.NativeFailure, "cuda_no_device"):
                    libraries.recognition(SimpleNamespace(get_cuda_device_count=lambda: 0))


if __name__ == "__main__":
    unittest.main()
