#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Offline checks for the elected native decoder identities."""
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('media_tools', ROOT / 'scripts/media-tools.py')
tools = importlib.util.module_from_spec(spec)
spec.loader.exec_module(tools)


class DecoderPins(unittest.TestCase):
    def test_all_native_platforms_have_matching_decoder_distribution(self):
        lock = json.loads((ROOT / 'internal/qualification/media-tools.json').read_text())
        self.assertEqual(lock['ffmpeg']['release'], lock['ffprobe']['release'])
        self.assertEqual(set(lock['ffmpeg']['assets']), set(lock['ffprobe']['assets']))
        for key, pin in lock['ffmpeg']['assets'].items():
            self.assertRegex(pin['sha256'], r'^[0-9a-f]{64}$')
            self.assertTrue(pin['name'].startswith('ffmpeg-'))
            for field in ['license', 'readme']:
                self.assertEqual(pin[field], lock['ffprobe']['assets'][key][field])
                self.assertEqual(pin[field + '_sha256'], lock['ffprobe']['assets'][key][field + '_sha256'])

    def test_generated_decoder_identity_is_exact(self):
        with self.assertRaises(ValueError):
            tools.tool_identity(Path(__file__), b'not a version\n')
        with self.assertRaises(ValueError):
            tools.tool_identity(Path(__file__), b'')
        identity = tools.tool_identity(Path(__file__), b'ffmpeg version 6.1.1-fixture Copyright test\n')
        self.assertEqual(identity['version'], '6.1.1-fixture')
        self.assertEqual(identity['sha256'], tools.sha(Path(__file__)))
        self.assertEqual(identity['path'], str(Path(__file__).resolve()))

    def test_macos_companions_build_from_pinned_redistributable_source(self):
        lock = json.loads((ROOT / 'internal/qualification/media-tools.json').read_text())
        source = lock['source_build']
        self.assertEqual(source['platform'], 'darwin_arm64')
        self.assertRegex(source['commit'], r'^[0-9a-f]{40}$')
        self.assertRegex(source['sha256'], r'^[0-9a-f]{64}$')
        self.assertNotIn('darwin_arm64', lock['ffmpeg']['assets'])
        self.assertNotIn('darwin_arm64', lock['ffprobe']['assets'])
        self.assertNotIn('--enable-nonfree', source['configure'])
        self.assertNotIn('--enable-gpl', source['configure'])
        self.assertIn('--disable-autodetect', source['configure'])
        self.assertIn('--enable-videotoolbox', source['configure'])

    def test_source_extraction_accepts_codeload_root_and_rejects_escape(self):
        source_spec = importlib.util.spec_from_file_location('media_source', ROOT / 'scripts/build-media-source.py')
        source = importlib.util.module_from_spec(source_spec)
        source_spec.loader.exec_module(source)
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            archive = directory / 'source.tar'
            root_name = 'FFmpeg-' + source.COMMIT
            with tarfile.open(archive, 'w') as output:
                root = tarfile.TarInfo(root_name)
                root.type = tarfile.DIRTYPE
                output.addfile(root)
                release = tarfile.TarInfo(root_name + '/RELEASE')
                release.size = len(b'7.0.2\n')
                output.addfile(release, io.BytesIO(b'7.0.2\n'))
            source.extract_source(archive, directory / 'extracted')
            self.assertEqual((directory / 'extracted/RELEASE').read_bytes(), b'7.0.2\n')
            with tarfile.open(archive, 'w') as output:
                escape = tarfile.TarInfo(root_name + '/../escape')
                escape.size = 1
                output.addfile(escape, io.BytesIO(b'x'))
            with self.assertRaises(ValueError):
                source.extract_source(archive, directory / 'rejected')
            self.assertFalse((directory / 'escape').exists())


if __name__ == '__main__':
    unittest.main()
