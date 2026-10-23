# SPDX-License-Identifier: Apache-2.0
"""Archive provenance must survive optimized Python and path replacement."""
from contextlib import contextmanager
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
CORPUS = ROOT / "tests/fixtures/acoustic"
ADMIT = CORPUS / "admit.py"


def fixture_archive(path):
    corpus = json.loads((CORPUS / "manifest.json").read_text(encoding="utf-8"))
    with tarfile.open(path, "w:gz") as archive:
        def add(name, data):
            item = tarfile.TarInfo(name)
            item.size = len(data)
            archive.addfile(item, io.BytesIO(data))
        for clip in corpus["clips"]:
            add(clip["archive_member"], (CORPUS / clip["path"]).read_bytes())
            parent = clip["archive_member"].rsplit("/", 1)[0]
            add(parent + "/" + clip["id"].rsplit("-", 1)[0] + ".trans.txt",
                (clip["id"] + " " + clip["transcript"] + "\n").encode())
        for name in ("SPEAKERS.TXT", "CHAPTERS.TXT", "LICENSE.TXT"):
            add("LibriSpeech/" + name, b"; fixture metadata\n")
    return corpus


def load():
    spec = importlib.util.spec_from_file_location("tested_acoustic_admission", ADMIT)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class AdmissionTests(unittest.TestCase):
    def test_optimized_python_cannot_admit_an_unpinned_archive(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive, output = root / "different.tar.gz", root / "corpus"
            fixture_archive(archive)
            result = subprocess.run([sys.executable, "-B", "-O", str(ADMIT), str(archive), str(output)],
                                    stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                    timeout=20, creationflags=subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0)
            self.assertNotEqual(result.returncode, 0, result.stdout.decode())
            self.assertFalse(output.exists())

    def test_both_parsing_passes_use_verified_snapshot_after_source_path_replacement(self):
        admission = load()
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, output = root / "source.tar.gz", root / "corpus"
            original = fixture_archive(source)
            original_bytes = source.read_bytes()
            digest = hashlib.sha256(original_bytes).hexdigest()
            verified = admission.verified_snapshot

            @contextmanager
            def replace_after_verification(path):
                with verified(path) as snapshot:
                    replacement = root / "replacement.tar.gz"
                    replacement.write_bytes(b"replacement after verification")
                    replacement.replace(source)
                    yield snapshot

            with patch.object(admission, "ARCHIVE_SHA256", digest), patch.object(admission, "ARCHIVE_SIZE", len(original_bytes)), \
                    patch.object(admission, "verified_snapshot", replace_after_verification):
                admission.reproduce(source, output)
            result = json.loads((output / "manifest.json").read_text(encoding="utf-8"))
            self.assertEqual(result["source"]["sha256"], digest)
            self.assertEqual(result["source"]["size_bytes"], len(original_bytes))
            self.assertEqual([(c["id"], c["sha256"]) for c in result["clips"]],
                             [(c["id"], c["sha256"]) for c in original["clips"]])

    def test_correct_size_still_requires_the_official_sha256(self):
        admission = load()
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "archive"
            path.write_bytes(b"incorrect source bytes")
            with patch.object(admission, "ARCHIVE_SIZE", path.stat().st_size):
                with self.assertRaisesRegex(ValueError, "SHA-256"):
                    with admission.verified_snapshot(path):
                        self.fail("Unverified bytes became a parsing snapshot")


if __name__ == "__main__":
    unittest.main()
