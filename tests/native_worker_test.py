# SPDX-License-Identifier: Apache-2.0
"""Deterministic native-library election; no engine imports or GPU access."""
import hashlib
import importlib.util
import json
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
