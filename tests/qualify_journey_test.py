# SPDX-License-Identifier: Apache-2.0
import importlib.util
import os
import sys
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch, Mock
from types import SimpleNamespace

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('journey', ROOT / 'scripts/qualify-journey.py')
journey = importlib.util.module_from_spec(spec)


class JourneyGuardTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls): spec.loader.exec_module(journey)

    def test_real_and_controlled_modes_require_explicit_election_outside_ci(self):
        with patch.dict(os.environ, {'CI': '', 'GITHUB_ACTIONS': '', 'TF_BUILD': '', 'BUILD_BUILDID': '', 'JENKINS_URL': ''}):
            with self.assertRaisesRegex(ValueError, 'explicit'): journey.require_mode(None)
            journey.require_mode('controlled')
            journey.require_mode('real')
            for marker in journey.engine.CI_VARIABLES:
                with patch.dict(os.environ, {marker: '1'}):
                    with self.assertRaisesRegex(ValueError, 'CI'): journey.require_mode('real')
                    with self.assertRaisesRegex(ValueError, 'CI'): journey.require_mode('controlled')

    def test_source_withdrawal_cannot_move_an_unowned_directory(self):
        with tempfile.TemporaryDirectory() as owned, tempfile.TemporaryDirectory() as unrelated:
            base = Path(owned)
            with self.assertRaisesRegex(ValueError, 'owned'): journey.withdraw_source(Path(unrelated), base)
            source = base / 'source-workspace'; source.mkdir(); (source / 'receipt').write_bytes(b'owned')
            unavailable = journey.withdraw_source(source, base)
            self.assertFalse(source.exists())
            self.assertEqual((unavailable / 'receipt').read_bytes(), b'owned')

    def test_restored_current_authority_requires_exact_nested_values(self):
        before = {'audio': {'sha256': 'a' * 64}, 'clock': {'unix_ns': 1577836800123456789}, 'profile': 'exact-version'}
        journey.compare_authority(before, dict(before))
        for change in ({**before, 'profile': 'another-version'}, {**before, 'clock': {'unix_ns': 1577836800123456700}}):
            with self.assertRaisesRegex(ValueError, 'authority'): journey.compare_authority(before, change)

    def test_cli_failure_reports_only_its_structured_code(self):
        command = [sys.executable, '-c', 'import json,sys; print(json.dumps({"error":{"code":"invalid_request","message":"private content"}}),file=sys.stderr); sys.exit(2)']
        with self.assertRaisesRegex(ValueError, '^CLI operation failed: invalid_request$'):
            journey.engine._child(command, dict(os.environ), cli_response=True, timeout=5)

    def test_child_output_is_bounded(self):
        command = [sys.executable, '-c', 'print("x" * 100000)']
        with self.assertRaisesRegex(ValueError, 'bounded limit'):
            journey.engine._child(command, dict(os.environ), max_output=1024, timeout=5)

    def test_failed_journey_writes_failure_receipt_and_reaps_owned_runtime(self):
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / 'receipt.json'
            runtime = Mock()
            runner = SimpleNamespace(run=Mock(side_effect=ValueError('private failure detail')), receipt={'mode': 'controlled'},
                                     started=journey.time.monotonic(), runtime=runtime)
            args = ['qualify-journey.py', '--mode', 'controlled', '--cli', 'fixture', '--tools', 'fixture', '--cueson', 'fixture',
                    '--recognition-python', 'fixture', '--diarization-python', 'fixture', '--output', str(output)]
            with patch.object(sys, 'argv', args), patch.object(journey, 'require_mode'), patch.object(journey, 'Journey', return_value=runner):
                with self.assertRaises(ValueError): journey.main()
            value = journey.json.loads(output.read_text())
            self.assertEqual(value['state'], 'failed')
            self.assertNotIn('private failure detail', output.read_text())
            runtime.wait.assert_called_once_with(timeout=40)
