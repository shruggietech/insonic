# SPDX-License-Identifier: Apache-2.0
from contextlib import contextmanager
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import MagicMock, patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
spec = importlib.util.spec_from_file_location('package_allocation', ROOT / 'scripts/package-desktop.py')
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)


class PackageWorkloadAllocationTests(unittest.TestCase):
    def test_child_uses_only_the_explicit_factory(self):
        tree = MagicMock()
        tree.process.communicate.return_value = (b'passed', b'')
        tree.process.returncode = 0
        factory = MagicMock(return_value=tree)
        tree.__enter__.return_value = tree
        with patch.object(package, 'ProcessTree') as ordinary:
            self.assertEqual(package.child(['fixture'], env={'FIXTURE': 'kept'}, process_factory=factory), b'passed')
        ordinary.assert_not_called()
        self.assertEqual(factory.call_args.kwargs['env']['FIXTURE'], 'kept')
        self.assertNotIn('cpu_affinity', factory.call_args.kwargs)

    def test_election_is_disposable_windows_desktop_only_and_default_is_unchanged(self):
        receipt = {'platform': 'windows_amd64', 'variant': 'desktop'}
        with patch.dict(os.environ, {}, clear=True), patch.object(package, 'WorkloadAffinity') as create:
            with package.package_workload_allocation(receipt) as factory:
                self.assertIsNone(factory)
            create.assert_not_called()
        for selected in [dict(receipt, platform='linux_amd64'), dict(receipt, variant='cli')]:
            with patch.dict(os.environ, {'INSONIC_PACKAGE_WORKLOAD_CPU_CONTROL': 'true'}, clear=True), \
                    patch.object(package, 'WorkloadAffinity') as create:
                with self.assertRaises(ValueError):
                    with package.package_workload_allocation(selected):
                        self.fail('unsupported election reached launches')
                create.assert_not_called()
        with patch.dict(os.environ, {'INSONIC_PACKAGE_WORKLOAD_CPU_CONTROL': 'true'}, clear=True), \
                patch('windows_msi.disposable_runner', side_effect=RuntimeError('not disposable')), \
                patch.object(package, 'WorkloadAffinity') as create:
            with self.assertRaisesRegex(RuntimeError, 'not disposable'):
                with package.package_workload_allocation(receipt):
                    self.fail('unapproved host reached launches')
            create.assert_not_called()

    def test_receipt_is_after_join_and_cannot_replace_original_failure(self):
        events = []
        factory = MagicMock()
        factory.receipt.side_effect = lambda: events.append('receipt') or {'launches': 1}
        with tempfile.TemporaryDirectory() as temporary, \
                patch.object(package, 'ROOT', Path(temporary)), \
                patch.dict(os.environ, {'INSONIC_PACKAGE_WORKLOAD_CPU_CONTROL': 'true'}, clear=True), \
                patch('windows_msi.disposable_runner'), patch.object(package, 'WorkloadAffinity', return_value=factory):
            with self.assertRaisesRegex(RuntimeError, 'native failure'):
                with package.package_workload_allocation({'platform': 'windows_amd64', 'variant': 'desktop'}) as selected:
                    self.assertIs(selected, factory)
                    events.append('joined')
                    raise RuntimeError('native failure')
            value = json.loads((Path(temporary) / 'build/native/package-workload-cpu-receipt.json').read_text())
            self.assertEqual(value['launches'], 1)
            self.assertEqual(events, ['joined', 'receipt'])
            with patch.object(package, 'write_json', side_effect=OSError('secondary receipt failure')):
                with self.assertRaisesRegex(RuntimeError, 'native failure'):
                    with package.package_workload_allocation({'platform': 'windows_amd64', 'variant': 'desktop'}):
                        raise RuntimeError('native failure')
                with self.assertRaisesRegex(OSError, 'secondary receipt failure'):
                    with package.package_workload_allocation({'platform': 'windows_amd64', 'variant': 'desktop'}):
                        pass

    def test_installed_first_use_allocates_the_powershell_root(self):
        factory = MagicMock()
        with tempfile.TemporaryDirectory() as temporary:
            location = Path(temporary)
            installed = location / 'installed program 文 with spaces'
            installed.mkdir()
            (installed / 'package-inventory.json').write_text('{}')
            library = location / 'first-use library 文 with spaces' / '.insonic'
            library.mkdir(parents=True)
            (library / 'workspace.json').write_text('{}')
            result = {'version': package.VERSION, 'help': 'help', 'init': 'init', 'doctor': 'doctor', 'show': 'show'}
            with patch.object(package, 'child', return_value=json.dumps(result).encode()) as launch, \
                    patch.object(package, 'verify_inventory'):
                self.assertEqual(package.qualify_windows_install(location / 'extracted', location, {}, process_factory=factory), installed)
            self.assertIs(launch.call_args.kwargs['process_factory'], factory)
            self.assertTrue(launch.call_args.kwargs['allow_detached'])
            self.assertIn("' doctor'", launch.call_args.args[0][-1])

    def test_smoke_routes_initial_and_full_cli_journey_but_not_gui(self):
        factory = MagicMock()
        calls = []
        @contextmanager
        def allocation(receipt):
            yield factory
        @contextmanager
        def gui(*args, **options):
            self.assertNotIn('process_factory', options)
            yield MagicMock()
        def launch(args, **options):
            calls.append((list(args), options))
            self.assertIs(options['process_factory'], factory)
            if args[1] == 'version':
                return package.VERSION.encode()
            if args[1:3] == ['workspace', 'init']:
                Path(args[3]).mkdir(parents=True)
                return b'{}'
            command = list(args[3:])
            if command[:1] == ['--quiet']:
                command = command[1:]
            if command[:2] == ['setup', 'show']:
                result = {'revision': 0, 'state': {'choices': {'processing_mode': 'none'}, 'stages': []}}
            elif command[:2] == ['setup', 'save']:
                result = {'accepted_revision': 1}
            elif command[:2] == ['settings', 'setup']:
                result = {'state': {'status': 'deferred'}}
            elif command[:2] == ['setup', 'readiness']:
                result = {'tasks': [0] * 5}
            elif command[:2] == ['media', 'import']:
                result = {'work_id': 'fixture-work'}
            elif command[:2] == ['work', 'wait']:
                if '--json' not in command:
                    return b'Warning: second --originated-on'
                result = {'id': 'fixture-work', 'state': 'succeeded', 'result': {'partial': True, 'items': [
                    {'state': 'admitted', 'capture_state': 'captured', 'media_id': 'first'},
                    {'state': 'admitted', 'capture_state': 'captured', 'media_id': 'second', 'date_unresolved': True}]}}
            else:
                result = {}
            return json.dumps({'result': result}).encode()
        def extract(archive, location):
            (location / 'program').mkdir(parents=True)
            (location / 'program/package-inventory.json').write_bytes(b'{}')
        with tempfile.TemporaryDirectory() as temporary:
            archive = Path(temporary) / 'archive.zip'
            archive.write_bytes(b'archive')
            receipt = {'platform': 'windows_amd64', 'variant': 'desktop', 'archive_sha256': package.sha(archive),
                       'inventory_root': 'program', 'inventory_path': 'program/package-inventory.json',
                       'inventory_sha256': __import__('hashlib').sha256(b'{}').hexdigest()}
            with patch.object(package, 'package_workload_allocation', allocation), \
                    patch.object(package, 'verify_archive_receipt'), patch.object(package, 'extract_package', extract), \
                    patch.object(package, 'verify_inventory'), patch.object(package, 'verify_loader_paths'), \
                    patch.object(package, 'qualify_windows_install', side_effect=lambda root, *args, **kw: root) as install, \
                    patch.object(package, 'package_directories', side_effect=lambda root, *args: (root, root)), \
                    patch.object(package, 'package_environment', return_value={}), patch.object(package, 'child', launch), \
                    patch.object(package, 'background_gui_qualification', gui), patch.object(package, 'finish_package_checks') as finish, \
                    patch('cli_contract_qualification.qualify', side_effect=RuntimeError('full contract reached')) as contract:
                with self.assertRaisesRegex(RuntimeError, 'full contract reached'):
                    package._smoke(archive, 'program', receipt)
            self.assertIs(install.call_args.kwargs['process_factory'], factory)
            self.assertIs(contract.call_args.kwargs['process_factory'], factory)
            finish.assert_called_once()
            self.assertTrue(any(args[1:3] == ['workspace', 'init'] for args, _ in calls))
            self.assertTrue(any(args[3:5] == ['setup', 'show'] for args, _ in calls))


if __name__ == '__main__':
    unittest.main()
