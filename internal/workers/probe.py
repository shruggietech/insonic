# SPDX-License-Identifier: Apache-2.0
"""Inspect only selected installed libraries; never load a model or download."""
import ctypes
import hashlib
import importlib.metadata as metadata
import json
import os
from pathlib import Path
import platform
import sys


def inspect(role, snapshot):
    result = {"python_version": platform.python_version(), "cpus": os.cpu_count() or 1,
              "packages": {}, "devices": {"cpu": {"available": False, "hardware_count": 0,
                  "dependency_state": "engine_unavailable"}, "cuda": {"available": False,
                  "hardware_count": 0, "dependency_state": "not_probed"}}, "engine_available": False}
    selected = ("faster-whisper", "ctranslate2", "av") if role == "recognition" else (
        "pyannote-audio", "torch", "torchaudio", "torchcodec")
    try:
        for name in selected:
            result["packages"][name] = metadata.version(name)
        if role == "recognition":
            if result["packages"]["faster-whisper"] != "1.2.1" or result["packages"]["ctranslate2"] != "4.8.2":
                raise ValueError("version")
            import av
            import ctranslate2
            from faster_whisper import WhisperModel  # library import, no construction
            if not ctranslate2.get_supported_compute_types("cpu"):
                raise ValueError("cpu")
            try:
                count = ctranslate2.get_cuda_device_count()
                result["devices"]["cuda"]["hardware_count"] = count
                if count:
                    libraries = ("cublas64_12.dll", "cudnn64_9.dll") if os.name == "nt" else (
                        "libcublas.so.12", "libcudnn.so.9")
                    handles = [ctypes.CDLL(name) for name in libraries]
                    available = bool(ctranslate2.get_supported_compute_types("cuda"))
                    result["devices"]["cuda"].update(available=available,
                        dependency_state="libraries_available" if available else "unsupported_compute_type")
                else:
                    result["devices"]["cuda"]["dependency_state"] = "no_device"
            except Exception:
                result["devices"]["cuda"]["dependency_state"] = "missing_cuda_libraries"
        else:
            if result["packages"]["pyannote-audio"] != "4.0.7":
                raise ValueError("version")
            import torch
            import torchaudio
            from pyannote.audio import Pipeline  # library import, no construction
            from pyannote.audio.core.io import Audio
            waveform = torch.zeros(1, 1600)
            checked, rate = Audio()({"waveform": waveform, "sample_rate": 16000})
            if rate != 16000 or checked.shape != waveform.shape:
                raise ValueError("audio")
            torch.nn.functional.conv1d(waveform.reshape(1, 1, -1), torch.ones(1, 1, 3))
            try:
                available = bool(torch.cuda.is_available())
                result["devices"]["cuda"].update(available=available,
                    hardware_count=torch.cuda.device_count(),
                    dependency_state="libraries_available" if available else "cuda_unavailable")
            except Exception:
                result["devices"]["cuda"]["dependency_state"] = "cuda_unavailable"
        result["engine_available"] = True
        result["devices"]["cpu"].update(available=True, dependency_state="libraries_available")
        if snapshot:
            files = {str(Path(sys.executable).resolve())}
            for dist in metadata.distributions():
                for item in dist.files or ():
                    target = Path(dist.locate_file(item)).resolve()
                    if target.is_file():
                        files.add(str(target))
                    if len(files) > 30000:
                        raise ValueError("files")
            # Include interpreter configuration/native runtime identity too.
            for base in (Path(sys.prefix), Path(sys.base_prefix)):
                for name in ("pyvenv.cfg", "python312.dll", "python3.dll"):
                    target = base / name
                    if target.is_file():
                        files.add(str(target.resolve()))
            # A venv executes its base runtime and standard library too. Capture
            # these bytes without traversing unrelated environments or caches.
            files.add(str(Path(getattr(sys, "_base_executable", sys.executable)).resolve()))
            for directory, names, filenames in os.walk(sys.base_prefix):
                names[:] = [name for name in names if name not in {
                    "site-packages", "__pycache__", "include", "share", "Scripts"}]
                for name in filenames:
                    target = Path(directory) / name
                    if target.suffix in {".py", ".pyd", ".dll", ".so", ".zip"} or ".so." in name:
                        files.add(str(target.resolve()))
                        if len(files) > 30000:
                            raise ValueError("files")
            result["files"] = []
            for name in sorted(files):
                with open(name, "rb") as stream:
                    digest = hashlib.file_digest(stream, "sha256").hexdigest()
                result["files"].append({"path": name, "sha256": digest})
    except Exception:
        result["engine_available"] = False
        result["files"] = []
    return result


if __name__ == "__main__":
    role = sys.argv[1]
    if role not in {"recognition", "diarization"}:
        raise SystemExit(2)
    print(json.dumps(inspect(role, sys.argv[2] == "snapshot"), separators=(",", ":")))
