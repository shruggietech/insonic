# SPDX-License-Identifier: Apache-2.0
"""Shared native CUDA resolution, injected from the trusted runtime binary.

No inference engine is imported here. Receipt-backed workers may load only
native files named by that exact verified environment; inspection discovers
selected environment distributions and explicitly elected loader directories.
"""
import ctypes
import hashlib
import importlib.metadata as metadata
import json
import os
from pathlib import Path
import re
import sys

WINDOWS = os.name == "nt"
_receipt_files = None
_linker_handles = []
_linker_cuda_files = set()
LINUX_CUDA_SONAMES = ("libcublas.so.12", "libcudnn.so.9", "libnvrtc.so.12")


class NativeFailure(ValueError):
    pass


def _sha(path):
    with Path(path).open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def set_receipt(path, digest):
    global _receipt_files
    if not path and not digest:
        _receipt_files = None
        return
    try:
        if not Path(path).is_absolute() or not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise NativeFailure("invalid_request")
        raw = Path(path).read_bytes()
        if len(raw) > 16 << 20 or hashlib.sha256(raw).hexdigest() != digest:
            raise NativeFailure("checksum_mismatch")
        files = json.loads(raw)["inspection"]["files"]
        if not isinstance(files, list) or not 1 <= len(files) <= 30000:
            raise NativeFailure("invalid_request")
        elected = {}
        for item in files:
            target, expected = Path(item["path"]), item["sha256"]
            if not target.is_absolute() or not re.fullmatch(r"[0-9a-f]{64}", expected):
                raise NativeFailure("invalid_request")
            elected[str(target.resolve() if WINDOWS else target)] = expected
        _receipt_files = elected
        if not WINDOWS:
            verify_linux_bindings(elected)
    except NativeFailure:
        raise
    except (OSError, ValueError, KeyError, TypeError):
        raise NativeFailure("invalid_request") from None


def _native(path):
    name = path.name.lower()
    return name.endswith(".dll") if WINDOWS else ".so" in name


def _cuda(path):
    return path.name.lower().startswith(("cublas", "cudnn", "libcublas", "libcudnn", "nvrtc", "libnvrtc"))


def native_paths(path):
    # Linux loaders retain SONAME aliases, including lazily opened companions.
    # Pin the loader spelling as well as its current canonical bytes.
    return {path.resolve()} if WINDOWS else {path, path.resolve()}


def verify_linux_bindings(elected):
    """Verify aliases against existing authority; never elect new native files."""
    paths = [Path(name) for name in elected if _native(Path(name)) and _cuda(Path(name))]
    if not paths:
        return
    try:
        hashed = {}

        def current(path):
            target = path.resolve(strict=True)
            if target not in hashed:
                hashed[target] = _sha(target)
            return target, hashed[target]

        targets = {}
        for path in paths:
            expected = elected[str(path)]
            target, actual = current(path)
            if actual != expected:
                raise NativeFailure("checksum_mismatch")
            targets.setdefault(target, set()).add(expected)
        bindings = {}
        for directory in sorted({path.parent for path in paths}):
            for count, path in enumerate(directory.iterdir(), 1):
                if count > 16384:
                    raise NativeFailure("input_limit")
                if not _native(path) or not _cuda(path):
                    continue
                target, actual = current(path)
                if target not in targets or actual not in targets[target]:
                    raise NativeFailure("checksum_mismatch")
                bindings[path] = target
                if len(bindings) > 4096:
                    raise NativeFailure("input_limit")
        families, required = {}, set()
        for target in targets:
            # cuDNN/cuBLAS use major-version SONAMEs. Legacy receipts omitted
            # aliases, so accept only a binding that can still be proved in
            # its elected parent. Never guess NVRTC's minor-version SONAME.
            match = re.fullmatch(r"(lib(?:cudnn|cublas).*\.so\.[0-9]+)(?:\..+)?", target.name)
            if match:
                key = target.parent, match[1]
                families.setdefault(key, set()).add(target)
                if target.name != match[1]:
                    required.add(key)
        for parent, soname in required:
            candidates = families[(parent, soname)]
            if not any(alias.name == soname and resolved in candidates and
                       (alias.parent == parent or str(alias) in elected)
                       for alias, resolved in bindings.items()):
                raise NativeFailure("checksum_mismatch")
    except NativeFailure:
        raise
    except (OSError, RuntimeError):
        raise NativeFailure("checksum_mismatch") from None


def linux_linker_path(soname):
    """Ask the system loader for one SONAME's actual absolute mapped file."""
    try:
        loaded = ctypes.CDLL(soname)
    except OSError as error:
        message = str(error).lower()
        if getattr(error, "errno", None) == 2 or "cannot open shared object file" in message and "no such file or directory" in message:
            return None
        if any(marker in message for marker in ("wrong elf class", "invalid elf header", "elf file", "undefined symbol", "version `")):
            raise NativeFailure("cuda_library_incompatible") from None
        raise NativeFailure("engine_failed") from None

    # glibc exposes dlinfo in libc on current systems and libdl on older ones.
    # Unlike CDLL._name or find_library, link_map records the resolved file.
    class LinkMap(ctypes.Structure):
        pass

    LinkMap._fields_ = [("address", ctypes.c_void_p), ("name", ctypes.c_char_p), ("dynamic", ctypes.c_void_p),
                       ("next", ctypes.POINTER(LinkMap)), ("previous", ctypes.POINTER(LinkMap))]

    for provider in (None, "libdl.so.2"):
        try:
            api = ctypes.CDLL(provider).dlinfo
        except (OSError, AttributeError):
            continue
        api.argtypes = [ctypes.c_void_p, ctypes.c_int, ctypes.c_void_p]
        api.restype = ctypes.c_int
        link = ctypes.POINTER(LinkMap)()
        if api(loaded._handle, 2, ctypes.byref(link)) != 0 or not link or not link.contents.name:
            return None
        path = Path(os.fsdecode(link.contents.name))
        if path.is_absolute() and path.is_file():
            _linker_handles.append(loaded)
            # Record CUDA dependencies the loader actually mapped, including
            # a companion resolved from a different system-cache directory.
            seen = set()
            while link.contents.previous:
                identity = ctypes.addressof(link.contents)
                if identity in seen or len(seen) >= 4096:
                    raise NativeFailure("input_limit")
                seen.add(identity)
                link = link.contents.previous
            seen.clear()
            while link:
                identity = ctypes.addressof(link.contents)
                if identity in seen or len(seen) >= 4096:
                    raise NativeFailure("input_limit")
                seen.add(identity)
                if link.contents.name:
                    dependency = Path(os.fsdecode(link.contents.name))
                    if dependency.is_absolute() and _native(dependency) and _cuda(dependency) and dependency.is_file():
                        _linker_cuda_files.update(native_paths(dependency))
                link = link.contents.next
            return path.resolve()
        return None
    return None


def linux_linker_files(sonames):
    # Fixed engine SONAMEs use the platform's existing loader configuration.
    # Never change LD_LIBRARY_PATH, execute a shell, or search cwd/import paths.
    result, directories = set(), set()
    for soname in sonames:
        path = linux_linker_path(soname)
        if path is not None and path.is_absolute() and path.is_file():
            result.update(native_paths(path))
            directories.update(item.parent for item in native_paths(path))
    result.update(_linker_cuda_files)
    directories.update(path.parent for path in _linker_cuda_files)
    # cuDNN lazily loads CUDA-family components; cuBLAS and NVRTC have their
    # own companion libraries. Freeze those siblings without electing every
    # unrelated system library in a shared /usr/lib directory.
    for directory in sorted(directories):
        try:
            for count, path in enumerate(directory.iterdir(), 1):
                if count > 16384 or len(result) > 4096:
                    raise NativeFailure("input_limit")
                if _native(path) and _cuda(path) and path.is_file():
                    result.update(native_paths(path))
        except OSError:
            raise NativeFailure("cuda_library_missing") from None
    if len(result) > 4096:
        raise NativeFailure("input_limit")
    return result


def native_files():
    if _receipt_files is not None:
        if not WINDOWS:
            verify_linux_bindings(_receipt_files)
        result = []
        for name, digest in _receipt_files.items():
            path = Path(name)
            if _native(path) and (_cuda(path) or "torch" in [part.lower() for part in path.parts] or "nvidia" in [part.lower() for part in path.parts]):
                try:
                    if _sha(path) != digest:
                        raise NativeFailure("checksum_mismatch")
                except OSError:
                    raise NativeFailure("checksum_mismatch") from None
                result.append(path)
        return sorted(result)
    result = set()
    for name in ("nvidia-cublas-cu12", "nvidia-cudnn-cu12", "nvidia-cuda-nvrtc-cu12", "torch"):
        try:
            distribution = metadata.distribution(name)
        except metadata.PackageNotFoundError:
            continue
        for entry in distribution.files or ():
            path = Path(distribution.locate_file(entry))
            if path.is_absolute() and _native(path) and path.is_file():
                result.update(native_paths(path))
    # These are the explicitly retained loader settings from LocalEnvironment,
    # never the current directory or Python's import path.
    directories = os.environ.get("PATH" if WINDOWS else "LD_LIBRARY_PATH", "").split(os.pathsep)
    if WINDOWS and os.environ.get("CUDA_PATH"):
        directories.insert(0, str(Path(os.environ["CUDA_PATH"]) / "bin"))
    for value in directories:
        directory = Path(value)
        if not value or not directory.is_absolute() or not directory.is_dir():
            continue
        try:
            files = [path for path in directory.iterdir() if path.is_file() and _native(path)]
        except OSError:
            continue
        if any(_cuda(path) for path in files):
            for path in files:
                result.update(native_paths(path))
        if len(result) > 4096:
            raise NativeFailure("input_limit")
    if not WINDOWS:
        missing = [name for name in LINUX_CUDA_SONAMES if not any(path.name == name or path.name.startswith(name + ".") for path in result)]
        if missing:
            result.update(linux_linker_files(missing))
    if len(result) > 4096:
        raise NativeFailure("input_limit")
    return sorted(result)


def cuda_failure(error):
    """Recognize known device failures without exporting arbitrary error text."""
    if isinstance(error, NativeFailure):
        return str(error) if str(error).startswith("cuda_") else None
    message = str(error).lower()
    if type(error).__name__ == "OutOfMemoryError" and type(error).__module__.startswith("torch"):
        return "cuda_resource_exhausted"
    if not isinstance(error, RuntimeError):
        return None
    if any(text in message for text in ("cuda out of memory", "cuda failed with error out of memory", "cublas_status_alloc_failed", "cudnn_status_alloc_failed")):
        return "cuda_resource_exhausted"
    if any(text in message for text in ("no cuda-capable device", "no cuda gpus are available", "no nvidia driver")):
        return "cuda_no_device"
    if "cuda" in message and any(text in message for text in ("no kernel image", "invalid device function", "not compiled with cuda", "unsupported compute")):
        return "cuda_compute_unsupported"
    if "cuda driver version is insufficient" in message:
        return "cuda_library_incompatible"
    return None


class NativeLibraries:
    def __init__(self):
        self.files = native_files()
        self.handles = []
        self.loaded = []

    def __enter__(self):
        if WINDOWS:
            try:
                for directory in sorted({str(path.parent) for path in self.files}):
                    self.handles.append(os.add_dll_directory(directory))
            except OSError:
                self.__exit__(None, None, None)
                raise NativeFailure("cuda_library_incompatible") from None
        return self

    def __exit__(self, *_):
        for handle in reversed(self.handles):
            handle.close()
        self.handles.clear()

    def recognition(self, engine):
        try:
            if engine.get_cuda_device_count() < 1:
                raise NativeFailure("cuda_no_device")
            names = ("cublas64_12.dll", "cudnn64_9.dll") if WINDOWS else ("libcublas.so.12", "libcudnn.so.9")
            for name in names:
                choices = [path for path in self.files if path.name.lower() == name or not WINDOWS and path.name.startswith(name + ".")]
                if not choices:
                    raise NativeFailure("cuda_library_missing")
                try:
                    self.loaded.append(ctypes.CDLL(str(choices[0])))
                except OSError as error:
                    code = "cuda_library_missing" if getattr(error, "winerror", None) == 126 or getattr(error, "errno", None) == 2 else "cuda_library_incompatible"
                    raise NativeFailure(code) from None
            if "float16" not in engine.get_supported_compute_types("cuda"):
                raise NativeFailure("cuda_compute_unsupported")
        except NativeFailure:
            raise
        except Exception as error:
            raise NativeFailure(cuda_failure(error) or "engine_failed") from None
