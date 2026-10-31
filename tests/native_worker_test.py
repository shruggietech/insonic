# SPDX-License-Identifier: Apache-2.0
"""Deterministic native-library election; no engine imports or GPU access."""
import hashlib
import importlib.util
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))
import qualification_timing as test_timing


def load_native():
    spec = importlib.util.spec_from_file_location("native_fixture", ROOT / "internal/workers/native.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class NativeWorkerTests(unittest.TestCase):
    @unittest.skipUnless(sys.platform == "linux" and shutil.which("cc"), "Linux shared-library compiler")
    def test_linux_lazy_companion_retarget_is_rejected_in_fresh_children(self):
        native = load_native()
        with tempfile.TemporaryDirectory(prefix="native receipt ") as directory:
            root = Path(directory).resolve()
            marker = root / "marker.c"
            marker.write_text("int fixture_value(void) { return MARKER; }\n", encoding="utf-8")
            lazy = root / "lazy.c"
            lazy.write_text('#include <dlfcn.h>\nint fixture_lazy(void) { void *h = dlopen("libcudnn_ops.so.9", RTLD_NOW | RTLD_LOCAL); if (!h) return -1; int (*value)(void) = dlsym(h, "fixture_value"); return value ? value() : -2; }\n', encoding="utf-8")

            def compile_library(source, destination, soname, *extra):
                subprocess.run([shutil.which("cc"), "-shared", "-fPIC", str(source), "-o", str(destination),
                                "-Wl,-soname," + soname, *extra], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=test_timing.BUILD_SECONDS)

            cublas, cudnn, old = root / "libcublas.so.12.1", root / "libcudnn.so.9.1", root / "libcudnn_ops.so.9.1"
            compile_library(marker, cublas, "libcublas.so.12", "-DMARKER=1")
            compile_library(marker, old, "libcudnn_ops.so.9", "-DMARKER=1")
            compile_library(lazy, cudnn, "libcudnn.so.9", "-Wl,-rpath,$ORIGIN", "-ldl")
            (root / "libcublas.so.12").symlink_to(cublas.name)
            (root / "libcudnn.so.9").symlink_to(cudnn.name)
            alias = root / "libcudnn_ops.so.9"
            alias.symlink_to(old.name)
            paths = {"libcublas.so.12": cublas, "libcudnn.so.9": cudnn}
            with patch.object(native.metadata, "distribution", side_effect=native.metadata.PackageNotFoundError), \
                 patch.dict(os.environ, {"LD_LIBRARY_PATH": ""}), patch.object(native, "linux_linker_path", side_effect=paths.get):
                files = native.native_files()
            legacy_files = sorted({path.resolve() for path in files})
            receipts = []
            for label, elected in (("current", files), ("legacy", legacy_files)):
                receipt = root / (label + ".json")
                receipt.write_text(json.dumps({"inspection": {"files": [{"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()} for path in elected]}}), encoding="utf-8")
                receipts.append(receipt)
            child = root / "child.py"
            child.write_text('''import ctypes,hashlib,importlib.util,json,sys
from pathlib import Path
from types import SimpleNamespace
spec=importlib.util.spec_from_file_location("native",sys.argv[1]);native=importlib.util.module_from_spec(spec);spec.loader.exec_module(native)
try:
 if sys.argv[2]!="raw":
  receipt=Path(sys.argv[2]);native.set_receipt(str(receipt),hashlib.sha256(receipt.read_bytes()).hexdigest())
  with native.NativeLibraries() as selected:selected.recognition(SimpleNamespace(get_cuda_device_count=lambda:1,get_supported_compute_types=lambda _: {"float16"}))
 library=ctypes.CDLL(sys.argv[3]);library.fixture_lazy.restype=ctypes.c_int
 print(json.dumps({"value":library.fixture_lazy()}))
except native.NativeFailure as error:print(json.dumps({"error":str(error)}))
''', encoding="utf-8")

            def execute(receipt, loader_directory=None):
                environment = os.environ.copy()
                if loader_directory is not None:
                    environment["LD_LIBRARY_PATH"] = str(loader_directory)
                observed = subprocess.run([sys.executable, "-I", "-B", str(child), str(ROOT / "internal/workers/native.py"), str(receipt), str(cudnn)],
                                          env=environment, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=test_timing.SUBPROCESS_SECONDS)
                return json.loads(observed.stdout)

            for receipt in receipts:
                self.assertEqual(execute(receipt), {"value": 1})
            retained = hashlib.sha256(old.read_bytes()).hexdigest()
            newer = root / "libcudnn_ops.so.9.2"
            compile_library(marker, newer, "libcudnn_ops.so.9", "-DMARKER=2")
            alias.unlink()
            alias.symlink_to(newer.name)
            self.assertEqual(execute("raw"), {"value": 2}, "fixture did not exercise a real lazy SONAME load")
            self.assertEqual(hashlib.sha256(old.read_bytes()).hexdigest(), retained)
            for receipt in receipts:
                self.assertEqual(execute(receipt), {"error": "checksum_mismatch"})
            self.assertIn(alias, files, "snapshot discarded the lazy loader binding")
            fallback = root / "other-loader-directory"
            fallback.mkdir()
            newer.rename(fallback / "libcudnn_ops.so.9")
            alias.unlink()
            self.assertEqual(execute("raw", fallback), {"value": 2}, "missing alias did not exercise another loader directory")
            for receipt in receipts:
                self.assertEqual(execute(receipt, fallback), {"error": "checksum_mismatch"})

    def test_linux_alias_receipts_keep_bindings_and_reject_missing_or_broken_aliases(self):
        native = load_native()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            target = root / "libcudnn_ops.so.9.1"
            target.write_bytes(b"elected CUDA bytes")
            alias = root / "libcudnn_ops.so.9"
            try:
                alias.symlink_to(target.name)
            except OSError:
                self.skipTest("file symlinks unavailable")
            receipt = root / "receipt.json"

            def elect(files):
                receipt.write_text(json.dumps({"inspection": {"files": [{"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()} for path in files]}}), encoding="utf-8")
                native.set_receipt(str(receipt), hashlib.sha256(receipt.read_bytes()).hexdigest())

            with patch.object(native, "WINDOWS", False):
                elect([target, alias])
                self.assertEqual(native.native_files(), sorted([alias, target]))
                alias.unlink()
                with self.assertRaisesRegex(native.NativeFailure, "checksum_mismatch"):
                    native.native_files()
                alias.symlink_to(target.name)
                elect([target])
                (root / "unrelated.so").write_bytes(b"not a CUDA dependency")
                self.assertEqual(native.native_files(), [target])
                alias.unlink()
                with self.assertRaisesRegex(native.NativeFailure, "checksum_mismatch"):
                    native.native_files()
                alias.symlink_to("missing.so")
                with self.assertRaisesRegex(native.NativeFailure, "checksum_mismatch"):
                    native.native_files()

    def test_linux_legacy_alias_accepts_one_of_multiple_pinned_versions(self):
        native = load_native()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            old, newer = root / "libcudnn_ops.so.9.1", root / "libcudnn_ops.so.9.2"
            old.write_bytes(b"first elected version")
            newer.write_bytes(b"second elected version")
            alias = root / "libcudnn_ops.so.9"
            try:
                alias.symlink_to(old.name)
            except OSError:
                self.skipTest("file symlinks unavailable")
            receipt = root / "receipt.json"
            receipt.write_text(json.dumps({"inspection": {"files": [{"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()} for path in [old, newer]]}}), encoding="utf-8")
            with patch.object(native, "WINDOWS", False):
                native.set_receipt(str(receipt), hashlib.sha256(receipt.read_bytes()).hexdigest())
                self.assertEqual(native.native_files(), [old, newer])
                alias.unlink()
                alias.symlink_to(newer.name)
                self.assertEqual(native.native_files(), [old, newer])

    def test_linux_existing_elected_files_do_not_probe_conflicting_system_copy(self):
        native = load_native()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
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
            root = Path(directory).resolve()
            names = ("libcublas.so.12.9.0", "libcublasLt.so.12.9.0", "libcudnn.so.9.1.0", "libcudnn_engines_runtime_compiled.so.9.1.0", "libnvrtc-builtins.so.12.9", "unrelated.so")
            for name in names:
                (root / name).write_bytes(name.encode())
            dependency_directory = root / "separate-linker-directory"
            dependency_directory.mkdir()
            dependency = dependency_directory / "libcublasLt.so.12.8.0"
            dependency.write_bytes(b"mapped dependency from another directory")
            aliases = []
            for target in [root / name for name in names[:-2]] + [dependency]:
                soname = target.name.split(".so.")[0] + ".so." + target.name.split(".so.")[1].split(".")[0]
                alias = target.with_name(soname)
                try:
                    alias.symlink_to(target.name)
                except OSError:
                    self.skipTest("file symlinks unavailable")
                aliases.append(alias)
            native._linker_cuda_files.add(dependency.resolve())
            resolved = {"libcublas.so.12": root / names[0], "libcudnn.so.9": root / names[2]}
            with patch.object(native, "WINDOWS", False), patch.object(native.metadata, "distribution", side_effect=native.metadata.PackageNotFoundError), \
                 patch.dict(os.environ, {"LD_LIBRARY_PATH": ""}), patch.object(native, "linux_linker_path", side_effect=lambda name: resolved.get(name)) as resolver:
                files = native.native_files()
                self.assertEqual(files, sorted([(root / name).resolve() for name in names[:-1]] + [dependency.resolve()] + aliases))
                self.assertEqual(resolver.call_count, 3)
                self.assertEqual(os.environ["LD_LIBRARY_PATH"], "")
                receipt = root / "receipt.json"
                receipt.write_text(json.dumps({"inspection": {"files": [{"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()} for path in files]}}), encoding="utf-8")
                native.set_receipt(str(receipt), hashlib.sha256(receipt.read_bytes()).hexdigest())
                (root / "libcudnn_new_component.so.9").write_bytes(b"not elected")
                resolver.reset_mock()
                with self.assertRaisesRegex(native.NativeFailure, "checksum_mismatch"):
                    native.native_files()
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
            root = Path(directory).resolve()
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
