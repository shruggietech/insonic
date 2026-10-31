# SPDX-License-Identifier: Apache-2.0
"""Inspect only selected installed libraries; never load a model or download."""
import hashlib
import importlib.metadata as metadata
import json
import os
from pathlib import Path
import platform
import stat
import sys
import time



class ProbeFailure(Exception):
    def __init__(self, reason):
        self.reason = reason


class Observation:
    """Generated facts only; native and Python monotonic clocks stay separate."""
    def __init__(self, snapshot, target):
        self.started = time.monotonic_ns()
        self.target = target
        self.write_failed = False
        self.value = {"mode": "snapshot" if snapshot else "inspect", "stage": "package_versions",
                      "reason": "running", "elapsed_ms": 0, "files": 0, "hashed_files": 0,
                      "hashed_bytes": 0, "checkpoints": []}

    def record(self, stage, reason="running"):
        value = self.value
        value.update(stage=stage, reason=reason, elapsed_ms=max(0, (time.monotonic_ns() - self.started) // 1_000_000))
        point = {key: value[key] for key in ("stage", "elapsed_ms", "files", "hashed_files", "hashed_bytes")}
        if value["checkpoints"] and value["checkpoints"][-1]["stage"] == stage:
            value["checkpoints"][-1] = point
        else:
            value["checkpoints"].append(point)
        if self.target:
            temporary = self.target + ".tmp"
            try:
                encoded = json.dumps(value, separators=(",", ":"), allow_nan=False).encode("utf-8")
                if len(encoded) > 16384:
                    raise OSError()
                descriptor = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
                with os.fdopen(descriptor, "wb") as stream:
                    stream.write(encoded)
                os.replace(temporary, self.target)
            except OSError:
                self.write_failed = True
                try:
                    os.unlink(temporary)
                except OSError:
                    pass

    def inventory_count(self, count):
        if count >= self.value["files"] + 256:
            self.value["files"] = count
            self.record("dependency_inventory")

    def finish(self, reason):
        if self.write_failed:
            reason = "checkpoint_invalid"
        self.record("complete" if reason == "success" else self.value["stage"], reason)
        return self.value


def inventory_distributions(files, distributions, observation):
    """Canonicalize directories once; preserve leaf links and Windows aliases."""
    parents = {}
    for dist in distributions:
        for item in dist.files or ():
            target = Path(dist.locate_file(item))
            parents.setdefault(target.parent, set()).add(target.name)
    canonical = {}
    for parent, names in parents.items():
        try:
            resolved = parent.resolve(strict=True)
        except (FileNotFoundError, NotADirectoryError):
            continue  # Missing RECORD entries were never part of the inventory.
        canonical.setdefault(resolved, set()).update(names)

    def add(target):
        files.add(str(target))
        observation.inventory_count(len(files))
        if len(files) > 30000:
            raise ProbeFailure("inventory_limit")

    for parent, names in canonical.items():
        remaining = {os.path.normcase(name): name for name in names}
        try:
            with os.scandir(parent) as entries:
                for entry in entries:
                    if remaining.pop(os.path.normcase(entry.name), None) is None:
                        continue
                    redirected = entry.is_symlink() or (os.name == "nt" and
                        entry.stat(follow_symlinks=False).st_file_attributes & stat.FILE_ATTRIBUTE_REPARSE_POINT)
                    if redirected:
                        target = Path(entry.path).resolve()
                        if target.is_file():
                            add(target)
                    elif entry.is_file(follow_symlinks=False):
                        add(parent / entry.name)
        except (FileNotFoundError, NotADirectoryError):
            continue
        # A Windows short leaf name need not equal its directory-entry spelling.
        # Resolve exceptional or newly appeared entries with the original rules.
        for name in remaining.values():
            target = (parent / name).resolve()
            if target.is_file():
                add(target)


def hash_inventory(files, observation):
    result = []
    buffer = bytearray(256 << 10)
    view = memoryview(buffer)
    for name in sorted(files):
        digest = hashlib.sha256()
        with open(name, "rb") as stream:
            while count := stream.readinto(buffer):
                digest.update(view[:count])
                observation.value["hashed_bytes"] += count
        result.append({"path": name, "sha256": digest.hexdigest()})
        observation.value["hashed_files"] += 1
        if observation.value["hashed_files"] % 128 == 0:
            observation.record("dependency_hashing")
    return result


def inspect(role, snapshot, progress_path=None):
    result = {"python_version": platform.python_version(), "cpus": os.cpu_count() or 1,
              "packages": {}, "devices": {"cpu": {"available": False, "hardware_count": 0,
                  "dependency_state": "engine_unavailable"}, "cuda": {"available": False,
                  "hardware_count": 0, "dependency_state": "not_probed"}}, "engine_available": False}
    observation = Observation(snapshot, progress_path)
    reason = "success"
    native = sys.modules.get("insonic_native")
    libraries = None
    cuda_error = None
    selected = ("faster-whisper", "ctranslate2", "av") if role == "recognition" else (
        "pyannote-audio", "torch", "torchaudio", "torchcodec")
    try:
        observation.record("package_versions")
        for name in selected:
            result["packages"][name] = metadata.version(name)
        if native is None:
            raise ProbeFailure("import_failed")
        try:
            libraries = native.NativeLibraries()
            libraries.__enter__()
        except native.NativeFailure as failure:
            cuda_error = native.cuda_failure(failure) or "engine_failed"
        if role == "recognition":
            if result["packages"]["faster-whisper"] != "1.2.1" or result["packages"]["ctranslate2"] != "4.8.2":
                raise ProbeFailure("version_mismatch")
            observation.record("library_import")
            import av
            import ctranslate2
            from faster_whisper import WhisperModel  # library import, no construction
            observation.record("cpu_prerequisites")
            if not ctranslate2.get_supported_compute_types("cpu"):
                raise ProbeFailure("cpu_unavailable")
            observation.record("cuda_prerequisites")
            try:
                count = ctranslate2.get_cuda_device_count()
                result["devices"]["cuda"]["hardware_count"] = count
                if count:
                    if cuda_error:
                        raise native.NativeFailure(cuda_error)
                    libraries.recognition(ctranslate2)
                    result["devices"]["cuda"].update(available=True, dependency_state="libraries_available")
                else:
                    result["devices"]["cuda"]["dependency_state"] = "cuda_no_device"
            except Exception as error:
                result["devices"]["cuda"]["dependency_state"] = native.cuda_failure(error) or "engine_failed"
        else:
            if result["packages"]["pyannote-audio"] != "4.0.7":
                raise ProbeFailure("version_mismatch")
            observation.record("library_import")
            import torch
            import torchaudio
            from pyannote.audio import Pipeline  # library import, no construction
            from pyannote.audio.core.io import Audio
            observation.record("cpu_prerequisites")
            waveform = torch.zeros(1, 1600)
            checked, rate = Audio()({"waveform": waveform, "sample_rate": 16000})
            if rate != 16000 or checked.shape != waveform.shape:
                raise ProbeFailure("cpu_unavailable")
            torch.nn.functional.conv1d(waveform.reshape(1, 1, -1), torch.ones(1, 1, 3))
            observation.record("cuda_prerequisites")
            try:
                available = bool(torch.cuda.is_available()) and not cuda_error
                result["devices"]["cuda"].update(available=available,
                    hardware_count=torch.cuda.device_count(),
                    dependency_state="libraries_available" if available else cuda_error or "cuda_no_device")
            except Exception as error:
                result["devices"]["cuda"]["dependency_state"] = native.cuda_failure(error) or "engine_failed"
        result["engine_available"] = True
        result["devices"]["cpu"].update(available=True, dependency_state="libraries_available")
        if snapshot:
            observation.record("dependency_inventory")
            files = {str(Path(sys.executable).resolve())}
            inventory_distributions(files, metadata.distributions(), observation)
            if libraries is not None:
                files.update(str(path) for path in libraries.files)
            if len(files) > 30000:
                raise ProbeFailure("inventory_limit")
            # Include interpreter configuration/native runtime identity too.
            for base in (Path(sys.prefix), Path(sys.base_prefix)):
                for name in ("pyvenv.cfg", "python312.dll", "python3.dll"):
                    target = base / name
                    if target.is_file():
                        files.add(str(target.resolve()))
            # A venv executes its base runtime and standard library too. Capture
            # these bytes without traversing unrelated environments or caches.
            files.add(str(Path(getattr(sys, "_base_executable", sys.executable)).resolve()))
            def walk_error(_):
                raise ProbeFailure("inventory_unreadable")
            for directory, names, filenames in os.walk(sys.base_prefix, onerror=walk_error):
                names[:] = [name for name in names if name not in {
                    "site-packages", "__pycache__", "include", "share", "Scripts"}]
                for name in filenames:
                    target = Path(directory) / name
                    if target.suffix in {".py", ".pyd", ".dll", ".so", ".zip"} or ".so." in name:
                        files.add(str(target.resolve()))
                        observation.inventory_count(len(files))
                        if len(files) > 30000:
                            raise ProbeFailure("inventory_limit")
            observation.value["files"] = len(files)
            observation.record("dependency_hashing")
            result["files"] = hash_inventory(files, observation)
    except metadata.PackageNotFoundError:
        reason = "package_missing"
    except ProbeFailure as failure:
        reason = failure.reason
    except Exception:
        reason = {"package_versions": "package_missing", "library_import": "import_failed",
                  "cpu_prerequisites": "cpu_unavailable", "dependency_inventory": "inventory_unreadable",
                  "dependency_hashing": "inventory_unreadable"}.get(observation.value["stage"], "process_failed")
    finally:
        if libraries is not None:
            libraries.__exit__(None, None, None)
    result["probe"] = observation.finish(reason)
    if result["probe"]["reason"] != "success":
        result["engine_available"] = False
        result["files"] = []
    return result


if __name__ == "__main__":
    role = sys.argv[1]
    if role not in {"recognition", "diarization"}:
        raise SystemExit(2)
    print(json.dumps(inspect(role, sys.argv[2] == "snapshot", sys.argv[3] if len(sys.argv) > 3 else None), separators=(",", ":")))
