#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Offline checks for the elected native decoder identities."""
import importlib.util
import json
from pathlib import Path
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


if __name__ == '__main__':
    unittest.main()
