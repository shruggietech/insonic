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

    def test_package_election_requires_exact_clean_revision_version_and_platform(self):
        receipt = {'revision': 'a' * 40, 'schema_version': '1.0.1', 'product_version': '1.0.1', 'documentation_version': '1.0.1',
                   'platform': 'windows_amd64', 'source_dirty': False, 'variant': 'cli'}
        journey.validate_package_election(receipt, 'a' * 40, '1.0.1', 'windows_amd64')
        for change in ({'revision': 'b' * 40}, {'schema_version': '0.0.0'}, {'product_version': '0.0.0'},
                       {'documentation_version': '0.0.0'}, {'platform': 'linux_amd64'}, {'source_dirty': True}, {'variant': 'unknown'}):
            with self.assertRaises(ValueError): journey.validate_package_election({**receipt, **change}, 'a' * 40, '1.0.1', 'windows_amd64')

    def test_package_selection_rejects_source_tool_or_cli_overrides(self):
        for field in ('cli', 'tools', 'cueson'):
            args = SimpleNamespace(package='archive', package_receipt='receipt', revision='a' * 40, cli=None, tools=None, cueson=None)
            setattr(args, field, 'development fallback')
            with self.assertRaisesRegex(ValueError, 'override'): journey.validate_selection(args)

    def test_missing_package_receipt_and_unselected_source_tools_are_rejected(self):
        for args in (SimpleNamespace(package='archive', package_receipt=None, revision='a' * 40, cli=None, tools=None, cueson=None),
                     SimpleNamespace(package=None, package_receipt=None, revision=None, cli='cli', tools=None, cueson=None)):
            with self.assertRaises(ValueError): journey.validate_selection(args)

    def test_package_environment_strips_native_tool_and_credential_overrides(self):
        original = {'PATH': 'development tooling', 'INSONIC_NATIVE_LIBRARY': 'development native', 'GH_TOKEN': 'private token',
                    'LD_LIBRARY_PATH': 'development native', 'CUESON_EXECUTABLE': 'development cueson'}
        with patch.dict(os.environ, original):
            environment = journey.packaged_environment()
        self.assertNotIn('GH_TOKEN', environment)
        self.assertEqual(environment['INSONIC_NATIVE_LIBRARY'], '')
        self.assertEqual(environment['CUESON_EXECUTABLE'], '')
        self.assertNotIn('development', environment['PATH'])
        self.assertEqual(environment['LD_LIBRARY_PATH'], '')
        self.assertEqual(environment['DYLD_FALLBACK_LIBRARY_PATH'], '')

    def test_extracted_selector_rejects_companion_escape_worker_drift_and_changed_archive(self):
        package = journey.package_module()
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary)
            archive = base / 'package.zip'; archive.write_bytes(b'controlled archive')
            receipt_path = base / 'receipt.json'
            receipt = {'revision': 'a' * 40, 'schema_version': '1.0.1', 'product_version': '1.0.1',
                       'documentation_version': '1.0.1', 'platform': 'linux_amd64', 'source_dirty': False, 'variant': 'cli',
                       'archive_sha256': package.sha(archive), 'inventory_sha256': journey.hashlib.sha256(b'{}').hexdigest(),
                       'inventory_root': 'installed', 'inventory_path': 'installed/package-inventory.json'}
            journey.write_json(receipt_path, receipt)
            args = SimpleNamespace(package=str(archive), package_receipt=str(receipt_path), revision='a' * 40)
            def extract(selected, location):
                root = location / 'installed'; root.mkdir()
                (root / 'package-inventory.json').write_bytes(b'{}')
                for name in ('ffmpeg', 'ffprobe', 'exiftool', 'cueson', 'insonic'):
                    (root / name).write_bytes(b'fixture')
                worker = root / 'companions/processing/processing-worker.py'; worker.parent.mkdir(parents=True)
                worker.write_bytes((ROOT / 'scripts/processing-worker.py').read_bytes())
                manifest = {'kind': 'installed-companions', 'schema_version': '1.0.1', 'platform': 'linux_amd64',
                            'media': {name: {'path': name, 'sha256': package.sha(root / name)} for name in ('ffmpeg', 'ffprobe', 'exiftool')},
                            'cueson': {'executable': 'cueson', 'executable_sha256': package.sha(root / 'cueson')}}
                if case == 'escape': manifest['media']['ffmpeg']['path'] = '../outside'
                if case == 'worker': worker.write_bytes(b'changed worker')
                if case == 'archive': selected.write_bytes(b'changed archive')
                journey.write_json(root / 'insonic-companions.json', manifest)
            for case in ('escape', 'worker', 'archive', 'valid'):
                archive.write_bytes(b'controlled archive')
                selected_base = base / case; selected_base.mkdir()
                with patch.object(journey, 'package_module', return_value=package), patch.object(package, 'platform_key', return_value='linux_amd64'), \
                     patch.object(package, 'verify_archive_receipt'), patch.object(package, 'verify_inventory'), \
                     patch.object(package, 'extract_package', side_effect=extract):
                    if case != 'valid':
                        with self.assertRaises(ValueError): journey.select_package(args, selected_base)
                    else:
                        cli, tools, cueson, worker, identity = journey.select_package(args, selected_base)
                        self.assertTrue(all(path.is_relative_to(selected_base) for path in (cli, cueson, worker)))
                        self.assertIn('with spaces', str(cli))
                        self.assertEqual(identity['revision'], receipt['revision'])
                        self.assertTrue(Path(tools['ffmpeg']['path']).is_relative_to(selected_base))

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
