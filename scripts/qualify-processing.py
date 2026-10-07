#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Explicit maintainer-only real inference qualification, never a CI check.

This command performs no installation, model acquisition or hosted operation.
Selected existing model trees must match the exact receipts below. Generated
native text and speaker turns remain transient; the receipt stores aggregates.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import threading
import time

ROOT = Path(__file__).resolve().parents[1]
CI_VARIABLES = ("CI", "GITHUB_ACTIONS", "TF_BUILD", "BUILD_BUILDID", "JENKINS_URL")
PINS = {
    "transcribe": {
        "id": "Systran/faster-whisper-tiny", "revision": "d90ca5fe260221311c53c58e660288d3deb8d356", "license": "MIT",
        "files": {
            "config.json": (2249, "a73a28cdfe1c43ccc7202fa333d1f89c202477271407ae9a7f19afa52039cac8"),
            "model.bin": (75538270, "dcb76c6586fc06cbdac6dd21f14cfd129cc4cdd9dce19bf4ffa62e59cbe6e6d1"),
            "tokenizer.json": (2203239, "fb7b63191e9bb045082c79fd742a3106a12c99513ab30df4a0d47fa6cb6fd0ab"),
            "vocabulary.txt": (459861, "34ce3fe1c5041027b3f8d42912270993f986dbc4bb34cf27f951e34a1e453913"),
        },
    },
    "diarize": {
        "id": "pyannote/speaker-diarization-community-1", "revision": "3533c8cf8e369892e6b79ff1bf80f7b0286a54ee", "license": "CC-BY-4.0",
        "files": {
            "config.yaml": (444, "5ce2bfa9a938dc132cec1172592d65173cbb8f444ea1e4133f10f9391de155be"),
            "embedding/pytorch_model.bin": (26646242, "6f10ff60898a1d185fa22e1d11e0bfa8a92efec811f11bca48cb8cafebefd929"),
            "segmentation/pytorch_model.bin": (5906507, "7ad24338d844fb95985486eb1a464e32d229f6d7a03c9abe60f978bacf3f816e"),
            "plda/plda.npz": (133852, "9b77bcd840692710dd3496f62ecfeed8d8e5f002fd991b785079b244eab7d255"),
            "plda/xvec_transform.npz": (134376, "325f1ce8e48f7e55e9c8aa47e05d2766b7c48c4b25b8de8dd751e7a4cc5fbe8f"),
        },
    },
}


def require_election(elected):
    if any(os.environ.get(name) for name in CI_VARIABLES):
        raise ValueError("Real engine qualification is forbidden in CI")
    if not elected:
        raise ValueError("Real engine qualification requires explicit --run-real-engines")


def sha(path):
    with Path(path).open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def verify_model(operation, root):
    root = Path(root).resolve(strict=True)
    for role, (size, digest) in PINS[operation]["files"].items():
        target = (root / role).resolve(strict=True)
        # HF snapshot files can be symlinks to their content-addressed blob
        # directory; read exact bytes and report identity, never mutate them.
        if target.stat().st_size != size or sha(target) != digest:
            raise ValueError("Selected model does not match pinned file receipts")
    return root


def local_environment():
    allowed = {"PATH", "SystemRoot", "WINDIR", "TEMP", "TMP", "TMPDIR", "LANG", "LC_ALL"}
    if os.name == "nt":
        allowed.discard("SystemRoot")
        allowed.update(("SYSTEMROOT", "CUDA_PATH", "CUDA_VISIBLE_DEVICES"))
    elif sys.platform == "linux":
        allowed.update(("LD_LIBRARY_PATH", "CUDA_VISIBLE_DEVICES"))
    elif sys.platform == "darwin":
        allowed.update(("DYLD_LIBRARY_PATH", "DYLD_FALLBACK_LIBRARY_PATH"))
    return {key: value for key, value in os.environ.items()
            if (key.upper() if os.name == "nt" else key) in allowed}


def secure_private_directory(directory):
    if os.name != "nt":
        os.chmod(directory, 0o700)
        return
    # Match the product's current-user-only protected Windows directory ACL.
    import ctypes
    from ctypes import wintypes
    api = ctypes.WinDLL("advapi32", use_last_error=True)
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel.GetCurrentProcess.restype = wintypes.HANDLE
    kernel.CloseHandle.argtypes = (wintypes.HANDLE,)
    kernel.LocalFree.argtypes = (ctypes.c_void_p,)
    api.OpenProcessToken.argtypes = (wintypes.HANDLE, wintypes.DWORD, ctypes.POINTER(wintypes.HANDLE))
    api.GetTokenInformation.argtypes = (wintypes.HANDLE, ctypes.c_int, ctypes.c_void_p, wintypes.DWORD, ctypes.POINTER(wintypes.DWORD))
    api.ConvertSidToStringSidW.argtypes = (ctypes.c_void_p, ctypes.POINTER(ctypes.c_void_p))
    api.ConvertStringSecurityDescriptorToSecurityDescriptorW.argtypes = (wintypes.LPCWSTR, wintypes.DWORD, ctypes.POINTER(ctypes.c_void_p), ctypes.c_void_p)
    api.GetSecurityDescriptorOwner.argtypes = (ctypes.c_void_p, ctypes.POINTER(ctypes.c_void_p), ctypes.POINTER(wintypes.BOOL))
    api.GetSecurityDescriptorDacl.argtypes = (ctypes.c_void_p, ctypes.POINTER(wintypes.BOOL), ctypes.POINTER(ctypes.c_void_p), ctypes.POINTER(wintypes.BOOL))
    api.SetNamedSecurityInfoW.argtypes = (wintypes.LPWSTR, ctypes.c_int, wintypes.DWORD, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p)
    api.SetNamedSecurityInfoW.restype = wintypes.DWORD
    token = wintypes.HANDLE()
    if not api.OpenProcessToken(kernel.GetCurrentProcess(), 8, ctypes.byref(token)):
        raise ValueError("Private home token access failed")
    sid_text, descriptor = ctypes.c_void_p(), ctypes.c_void_p()
    try:
        size = wintypes.DWORD()
        api.GetTokenInformation(token, 1, None, 0, ctypes.byref(size))
        data = ctypes.create_string_buffer(size.value)
        if not api.GetTokenInformation(token, 1, data, size.value, ctypes.byref(size)):
            raise ValueError("Private home identity access failed")
        sid = ctypes.cast(data, ctypes.POINTER(ctypes.c_void_p))[0]
        if not api.ConvertSidToStringSidW(sid, ctypes.byref(sid_text)):
            raise ValueError("Private home identity conversion failed")
        identity = ctypes.wstring_at(sid_text)
        sddl = "O:" + identity + "D:P(A;OICI;FA;;;" + identity + ")"
        if not api.ConvertStringSecurityDescriptorToSecurityDescriptorW(sddl, 1, ctypes.byref(descriptor), None):
            raise ValueError("Private home ACL construction failed")
        owner, acl, present, defaulted = ctypes.c_void_p(), ctypes.c_void_p(), wintypes.BOOL(), wintypes.BOOL()
        if not api.GetSecurityDescriptorOwner(descriptor, ctypes.byref(owner), ctypes.byref(defaulted)) or not api.GetSecurityDescriptorDacl(descriptor, ctypes.byref(present), ctypes.byref(acl), ctypes.byref(defaulted)):
            raise ValueError("Private home ACL access failed")
        if not present.value or api.SetNamedSecurityInfoW(str(directory), 1, 0x80000005, owner, None, acl, None):
            raise ValueError("Private home ACL application failed")
    finally:
        if descriptor.value:
            kernel.LocalFree(descriptor)
        if sid_text.value:
            kernel.LocalFree(sid_text)
        kernel.CloseHandle(token)


def child(command, *, request=None, timeout=600, max_output=16 << 20):
    require_election(True)
    with tempfile.TemporaryDirectory(prefix="insonic-private-worker-home-") as home:
        secure_private_directory(home)
        environment = local_environment()
        environment.update(HOME=home, USERPROFILE=home, HF_HUB_OFFLINE="1", TRANSFORMERS_OFFLINE="1", HF_HUB_DISABLE_TELEMETRY="1",
                           PYANNOTE_METRICS_ENABLED="0", PYTHONIOENCODING="utf-8", TOKENIZERS_PARALLELISM="false")
        return _child(command, environment, request=request, timeout=timeout, max_output=max_output)


def _child(command, environment, *, request=None, timeout=600, max_output=16 << 20):
    process = subprocess.Popen([str(item) for item in command], cwd=ROOT, env=environment,
                               stdin=subprocess.PIPE if request is not None else subprocess.DEVNULL,
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                               creationflags=subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0)
    buffers = [bytearray(), bytearray()]
    overflow = threading.Event()

    def drain(stream, result):
        try:
            while chunk := stream.read(65536):
                if len(result) + len(chunk) > max_output:
                    overflow.set()
                    process.kill()
                    break
                result.extend(chunk)
        finally:
            stream.close()

    readers = [threading.Thread(target=drain, args=(stream, result), daemon=True)
               for stream, result in zip((process.stdout, process.stderr), buffers)]
    for reader in readers:
        reader.start()
    try:
        if request is not None:
            process.stdin.write(json.dumps(request).encode("utf-8"))
            process.stdin.close()
        code = process.wait(timeout=timeout)
    except BaseException:
        process.kill()
        process.wait()
        raise
    finally:
        for reader in readers:
            reader.join(timeout=10)
    if overflow.is_set():
        raise ValueError("Qualification output exceeded bounded limit")
    if code:
        # Engine output may contain private paths. Only bounded explicit code
        # objects are included, never raw traceback or arbitrary stderr.
        try:
            failure = json.loads(bytes(buffers[1]).strip().splitlines()[-1])
        except (ValueError, UnicodeError, IndexError):
            failure = {"error": "engine_failed"}
        allowed = {"engine_failed", "engine_ci_forbidden", "engine_unavailable", "model_unavailable",
                   "incompatible_version", "unsupported_device", "invalid_embedding", "invalid_timing", "audio_limit"}
        code = failure.get("error", "engine_failed") if isinstance(failure, dict) else "engine_failed"
        if not isinstance(code, str):
            code = "engine_failed"
        raise ValueError("Local operation failed: " + (code if code in allowed else "engine_failed"))
    return bytes(buffers[0])


def word_error(reference, generated):
    def words(text):
        return re.findall(r"\w+", text.casefold())
    expected, actual = words(reference), words(generated)
    if len(expected) > 8192 or len(actual) > 8192:
        raise ValueError("Reference comparison limit exceeded")
    previous = list(range(len(actual)+1))
    for i, token in enumerate(expected, 1):
        current = [i]
        for j, value in enumerate(actual, 1):
            current.append(min(previous[j]+1, current[j-1]+1, previous[j-1]+(token != value)))
        previous = current
    return {"reference_words": len(expected), "generated_words": len(actual),
            "edit_distance": previous[-1], "word_error_rate": previous[-1]/len(expected) if expected else None}


def srt_text(srt):
    return " ".join(line for line in srt.splitlines() if line.strip() and not line.strip().isdigit() and " --> " not in line)


def qualify(args):
    require_election(args.run_real_engines)
    tools = json.loads(Path(args.tools).read_text(encoding="utf-8"))
    ffmpeg = tools["ffmpeg"]
    if sha(ffmpeg["path"]) != ffmpeg["sha256"]:
        raise ValueError("FFmpeg byte identity mismatch")
    version = child([ffmpeg["path"], "-version"], timeout=10).decode().splitlines()[0].split()[2]
    if version != ffmpeg["version"]:
        raise ValueError("FFmpeg version mismatch")
    models = {"transcribe": verify_model("transcribe", args.recognition_model),
              "diarize": verify_model("diarize", args.diarization_model)}
    pythons = {"transcribe": Path(args.recognition_python).resolve(strict=True),
               "diarize": Path(args.diarization_python).resolve(strict=True)}
    manifest = json.loads((ROOT / "tests/fixtures/media/manifest.json").read_text(encoding="utf-8"))
    receipt = {"kind": "maintainer-real-engine-qualification", "required_check": False,
               "ci": False, "worker_sha256": sha(ROOT / "scripts/processing-worker.py"), "models": PINS,
               "child_environment": "explicit platform execution/native-loader allowlist; private empty home; offline flags; no inherited application or provider credentials",
               "python_sha256": {operation: sha(python) for operation, python in pythons.items()},
               "ffmpeg_sha256": ffmpeg["sha256"], "fixtures": []}
    receipt["resource_limits"] = {"timeout_seconds_per_operation": args.timeout, "threads": args.threads,
                                  "max_decoded_duration_us": 60000000, "max_output_bytes_per_stream": 16 << 20,
                                  "hard_rss_limit_bytes": None, "rss_policy": "measured peak, no portable hard RSS cap claimed"}
    with tempfile.TemporaryDirectory(prefix="insonic-engine-qualification-") as scratch:
        for fixture in manifest["assets"]:
            original = ROOT / "tests/fixtures/media" / fixture["path"]
            if original.stat().st_size != fixture["size_bytes"] or sha(original) != fixture["sha256"]:
                raise ValueError("Fixture integrity mismatch")
            mapped = Path(scratch) / (fixture["id"] + ".wav")
            stream = fixture["media"]["audio_stream_index"]
            channel = 2 if fixture["id"] == "sintel-dialogue" else None
            command = [ffmpeg["path"], "-nostdin", "-hide_banner", "-loglevel", "error", "-i", original, "-map", f"0:{stream}", "-vn"]
            if channel is not None:
                command.extend(("-af", f"pan=mono|c0=c{channel}"))
            command.extend(("-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-t", "61", "-fs", "3000000", "-f", "wav", mapped))
            child(command, timeout=30)
            item = {"id": fixture["id"], "source_sha256": fixture["sha256"], "mapped_sha256": sha(mapped), "stream": stream, "channel": channel}
            for operation in ("transcribe", "diarize"):
                request = {"operation": operation, "audio_path": str(mapped), "model_path": str(models[operation]),
                           "device": args.device, "language": "en", "threads": args.threads, "max_duration_us": 60000000}
                result = json.loads(child([pythons[operation], "-I", "-B", ROOT / "scripts/processing-worker.py"], request=request, timeout=args.timeout))
                entry = {"no_speech": result["no_speech"], "provenance": result["provenance"], "diagnostics": result["diagnostics"]}
                if operation == "transcribe":
                    reference = srt_text((ROOT / "tests/fixtures/media" / ("speech.srt" if fixture["id"] == "speech" else "sintel.srt")).read_text(encoding="utf-8"))
                    entry["reference_comparison"] = word_error(reference, srt_text(result["srt"]))
                else:
                    entry["turn_count"] = len(result["turns"])
                    entry["accuracy_limit"] = "No auditory-verified tight speaker boundaries exist; coverage and counts are diagnostics, not measured diarization error rate."
                    if fixture["id"] == "sintel-dialogue":
                        entry["reference_character_count"] = 2
                        entry["reference_character_count_matches_acoustic_count"] = result["provenance"]["speaker_count"] == 2
                        if not entry["reference_character_count_matches_acoustic_count"]:
                            entry["diagnostics"].append({"code": "acoustic_count_differs_published_dialogue_characters", "count": 1})
                item[operation] = entry
            receipt["fixtures"].append(item)
    destination = Path(args.output).resolve()
    destination.parent.mkdir(parents=True, exist_ok=True)
    temporary = destination.with_suffix(destination.suffix + ".tmp")
    temporary.write_text(json.dumps(receipt, indent=2, ensure_ascii=False, allow_nan=False)+"\n", encoding="utf-8")
    temporary.replace(destination)
    return receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run-real-engines", action="store_true")
    parser.add_argument("--tools", default=str(ROOT / "build/native/media-tools.json"))
    parser.add_argument("--recognition-python", required=True)
    parser.add_argument("--diarization-python", required=True)
    parser.add_argument("--recognition-model", required=True)
    parser.add_argument("--diarization-model", required=True)
    parser.add_argument("--device", choices=("cpu", "cuda"), default="cpu")
    parser.add_argument("--threads", type=int, default=2)
    parser.add_argument("--timeout", type=int, default=600)
    parser.add_argument("--output", default=str(ROOT / "build/native/processing-qualification.json"))
    args = parser.parse_args()
    require_election(args.run_real_engines)
    if not 1 <= args.threads <= 64 or not 1 <= args.timeout <= 3600:
        raise ValueError("Invalid resource limits")
    receipt = qualify(args)
    print(json.dumps({"receipt": str(Path(args.output).resolve()), "fixtures": len(receipt["fixtures"]), "required_check": False}))


if __name__ == "__main__":
    main()
