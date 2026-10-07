#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Inference-free fixture integrity and independent reference regressions."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('media_fixtures', ROOT / 'scripts/media-fixtures.py')
fixtures = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixtures)


class FixtureIntegrity(unittest.TestCase):
    def setUp(self):
        self.manifest = json.loads((ROOT / 'tests/fixtures/media/manifest.json').read_text())

    def test_committed_real_assets_and_independent_references(self):
        result = fixtures.validate(self.manifest)
        self.assertEqual(result['fixtures'], 2)
        self.assertLess(result['media_bytes'], 10 << 20)
        self.assertEqual(result['inference'], 'not invoked')

    def test_media_digest_and_size_are_authoritative(self):
        for field, value in [('sha256', '0' * 64), ('size_bytes', 1)]:
            manifest = copy.deepcopy(self.manifest)
            manifest['assets'][0][field] = value
            with self.assertRaises(ValueError):
                fixtures.validate(manifest)

    def test_reference_bytes_cannot_change_silently(self):
        manifest = copy.deepcopy(self.manifest)
        manifest['assets'][0]['references'][0]['sha256'] = '0' * 64
        with self.assertRaises(ValueError):
            fixtures.validate(manifest)

    def test_crop_clock_and_duration_are_exact(self):
        manifest = copy.deepcopy(self.manifest)
        video = next(v for v in manifest['assets'] if v['id'] == 'sintel-dialogue')
        video['source_clock']['origin_ns'] += 1
        with self.assertRaises(ValueError):
            fixtures.validate(manifest)

    def test_fixture_paths_are_confined(self):
        manifest = copy.deepcopy(self.manifest)
        manifest['assets'][0]['path'] = '../../../package.json'
        with self.assertRaises(ValueError):
            fixtures.validate(manifest)

    def test_published_windows_are_not_claimed_as_voice_activity(self):
        reference = json.loads((ROOT / 'tests/fixtures/media/sintel-reference.json').read_text())
        self.assertEqual(reference['timing_basis'], 'published subtitle display windows')
        self.assertIsNone(reference['measured_voice_activity'])
        self.assertFalse(reference['auditory_verification'])
        self.assertEqual(len(reference['cues']), 5)
        self.assertEqual([c['character'] for c in reference['cues']], ['Sintel', 'Shaman', 'Sintel', 'Shaman', 'Sintel'])


if __name__ == '__main__':
    unittest.main()
