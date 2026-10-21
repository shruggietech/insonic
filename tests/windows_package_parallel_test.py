# SPDX-License-Identifier: Apache-2.0
from copy import deepcopy
import hashlib
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import threading
import time
from types import SimpleNamespace
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parents[1] / 'scripts'
sys.path.insert(0, str(SCRIPTS))
spec = importlib.util.spec_from_file_location('windows_parallel', SCRIPTS / 'qualify-windows-package.py')
parallel = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parallel)
release_spec = importlib.util.spec_from_file_location('parallel_release_fixture', SCRIPTS / 'release.py')
release = importlib.util.module_from_spec(release_spec)
release_spec.loader.exec_module(release)


class ParallelPackageFixture:
    def __init__(self, root, failure=None, tamper=None, missing_proof=False):
        self.root, self.failure, self.tamper, self.missing_proof = root, failure, tamper, missing_proof
        self.native = root / 'build/native'
        self.packages = root / 'build/packages'
        self.native.mkdir(parents=True)
        self.packages.mkdir(parents=True)
        self.archive = self.packages / 'desktop.zip'
        self.archive.write_bytes(b'immutable current desktop archive fixture')
        self.payload = self.packages / 'desktop'
        self.payload.mkdir()
        self.initial = {
            'kind': 'native-package-receipt', 'schema_version': '1.0.1',
            'product_version': '1.0.1', 'documentation_version': '1.0.1',
            'revision': 'a' * 40, 'source_dirty': False,
            'platform': 'windows_amd64', 'variant': 'desktop', 'dirname': 'desktop',
            'archive': self.archive.name, 'archive_sha256': self.sha(self.archive),
            'archive_size_bytes': self.archive.stat().st_size,
            'inventory_sha256': 'b' * 64,
            'signing': {'configured': False, 'verified': False, 'status': 'unconfigured'},
            'distribution': {'source_complete': True, 'binary_publication': 'eligible', 'corresponding_source': 'included'},
        }
        self.completed = deepcopy(self.initial)
        self.completed.update({key: 'passed' for key in [
            'extracted_inventory', 'relocation_path_spaces', 'cli_real_audio_video_import',
            'windows_copy_install', 'windows_first_use', 'gui_bridge', 'native_webview',
        ]})
        self.completed['development_native_libraries'] = 'isolated-and-restored'
        self.completed['elapsed_seconds'] = .01
        self.barrier = threading.Barrier(2)
        self.lock = threading.Lock()
        self.active = self.maximum = 0
        self.started, self.closed, self.verification = [], [], []
        self.package = SimpleNamespace(BUILD=self.packages, build=self.build,
                                       write_json=self.write, verify_archive_receipt=self.verify_archive)

    @staticmethod
    def sha(path):
        return hashlib.sha256(path.read_bytes()).hexdigest()

    @staticmethod
    def write(path, value):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(value) + '\n', encoding='utf-8')

    def build(self, variant):
        if variant != 'desktop':
            raise AssertionError('wrong elected package variant')
        self.write(self.native / 'package-desktop-receipt.json', self.initial)
        return self.archive, 'desktop', deepcopy(self.initial)

    def verify_archive(self, archive, receipt):
        self.verification.append('archive')
        if receipt['archive_sha256'] != self.sha(archive):
            raise ValueError('archive bytes changed')

    def verify_installer(self, path, receipt):
        self.verification.append('installer')
        on_disk = json.loads((self.native / 'windows-installer-receipt.json').read_text())
        if on_disk.get('publication_eligible') is not False:
            raise AssertionError('eligible receipt was written before strict verification')
        if self.failure == 'strict':
            raise ValueError('strict MSI verification failed')
        if (receipt['publication_eligible'] is not True or receipt['payload_qualification'] != 'passed'
                or receipt['payload_receipt'] != self.completed
                or receipt['artifact_sha256'] != self.sha(path)):
            raise ValueError('final MSI authority differs')
        if not all(receipt['qualification'].get(key) == 'passed' for key in parallel.m.QUALIFICATION_KEYS):
            raise ValueError('native MSI qualification incomplete')

    def child(self, arguments, **options):
        if arguments[0] != sys.executable or options['cwd'] != self.root or options['allow_detached'] is not True:
            raise AssertionError('child ownership or interpreter differs')
        role = 'archive' if Path(arguments[1]).name == 'package-desktop.py' else 'installer'
        with self.lock:
            self.started.append((role, list(arguments)))
            self.active += 1
            self.maximum = max(self.maximum, self.active)
        fixture = self

        class Child:
            returncode = 0
            def communicate(self, timeout):
                if timeout != 900:
                    raise AssertionError('unexpected outer qualification bound')
                fixture.barrier.wait(timeout=2)
                if role == 'archive':
                    if arguments[2:] != ['smoke', '--variant', 'desktop']:
                        raise AssertionError('archive gates changed')
                    completed = deepcopy(fixture.completed)
                    if fixture.tamper == 'archive_bytes':
                        fixture.archive.write_bytes(b'changed archive bytes after qualification')
                    elif fixture.tamper:
                        completed[fixture.tamper] = {'tampered': True} if fixture.tamper in ('signing', 'distribution') else 'tampered'
                    if fixture.missing_proof:
                        completed.pop('native_webview')
                    fixture.write(fixture.native / 'package-desktop-receipt.json', completed)
                else:
                    if arguments[2] != 'provisional':
                        raise AssertionError('native gates did not run provisionally')
                    snapshot = Path(arguments[arguments.index('--receipt') + 1])
                    if json.loads(snapshot.read_text()) != fixture.initial:
                        raise AssertionError('MSI did not receive immutable original archive receipt')
                    time.sleep(.03)  # A failed archive must still join this owned child.
                    path = fixture.packages / 'installer.msi'
                    path.write_bytes(b'one authored MSI, never rebuilt at join')
                    fixture.write(fixture.native / 'windows-installer-receipt.json', {
                        'artifact': path.name, 'artifact_sha256': fixture.sha(path),
                        'payload_receipt': fixture.initial,
                        'payload_archive': fixture.initial['archive'],
                        'payload_archive_sha256': fixture.initial['archive_sha256'],
                        'publication_eligible': False, 'payload_qualification': 'pending',
                        'qualification': {key: 'passed' for key in parallel.m.QUALIFICATION_KEYS},
                    })
                self.returncode = 1 if fixture.failure == role else 0
                return b'', b''

        class Tree:
            process = Child()
            def __enter__(self):
                return self
            def __exit__(self, *_):
                with fixture.lock:
                    fixture.active -= 1
                    fixture.closed.append(role)
        return Tree()

    def invoke(self):
        def module(name, filename):
            return self.package if filename == 'package-desktop.py' else release
        with patch.object(parallel, 'ROOT', self.root), patch.object(parallel, 'load', side_effect=module), \
                patch.object(parallel, 'ProcessTree', side_effect=self.child), \
                patch.object(parallel.m, 'disposable_runner'), \
                patch.object(parallel.m, 'verify_installer', side_effect=self.verify_installer):
            return parallel.qualify()

    def assert_provisional(self, testcase):
        path = self.native / 'windows-installer-receipt.json'
        testcase.assertFalse(json.loads(path.read_text()).get('publication_eligible'))
        testcase.assertEqual(self.active, 0)
        testcase.assertEqual(sorted(self.closed), ['archive', 'installer'])


class WindowsParallelQualificationTests(unittest.TestCase):
    def test_two_owned_children_overlap_and_strict_verification_precedes_eligibility(self):
        with tempfile.TemporaryDirectory(prefix='insonic-parallel-model-') as directory:
            fixture = ParallelPackageFixture(Path(directory))
            result = fixture.invoke()
            self.assertEqual(fixture.maximum, 2)
            self.assertEqual(sorted(role for role, _ in fixture.started), ['archive', 'installer'])
            self.assertEqual(fixture.active, 0)
            self.assertEqual(fixture.verification, ['archive', 'installer'])
            self.assertEqual(result['payload_receipt'], fixture.completed)
            self.assertTrue(result['publication_eligible'])
            self.assertEqual(result, json.loads((fixture.native / 'windows-installer-receipt.json').read_text()))

    def test_either_child_failure_joins_both_and_never_finalizes(self):
        for failure in ['archive', 'installer']:
            with self.subTest(failure=failure), tempfile.TemporaryDirectory(prefix='insonic-parallel-model-') as directory:
                fixture = ParallelPackageFixture(Path(directory), failure=failure)
                fixture.write(fixture.native / 'windows-installer-receipt.json', {'publication_eligible': True})
                with self.assertRaisesRegex(RuntimeError, 'parallel native qualification failed'):
                    fixture.invoke()
                fixture.assert_provisional(self)
                self.assertEqual(fixture.maximum, 2)
                self.assertEqual(fixture.verification, [])

    def test_archive_inventory_source_signing_and_distribution_tampering_blocks_finalization(self):
        for field in ['archive_sha256', 'inventory_sha256', 'revision', 'signing', 'distribution']:
            with self.subTest(field=field), tempfile.TemporaryDirectory(prefix='insonic-parallel-model-') as directory:
                fixture = ParallelPackageFixture(Path(directory), tamper=field)
                with self.assertRaisesRegex(ValueError, 'immutable assembly receipt'):
                    fixture.invoke()
                fixture.assert_provisional(self)

    def test_missing_actual_zip_qualification_cannot_be_replaced_with_native_passes(self):
        with tempfile.TemporaryDirectory(prefix='insonic-parallel-model-') as directory:
            fixture = ParallelPackageFixture(Path(directory), missing_proof=True)
            with self.assertRaisesRegex(ValueError, 'native qualification are incomplete'):
                fixture.invoke()
            fixture.assert_provisional(self)

    def test_changed_archive_bytes_are_independently_rejected_before_publication(self):
        with tempfile.TemporaryDirectory(prefix='insonic-parallel-model-') as directory:
            fixture = ParallelPackageFixture(Path(directory), tamper='archive_bytes')
            with self.assertRaisesRegex(ValueError, 'archive bytes changed'):
                fixture.invoke()
            fixture.assert_provisional(self)

    def test_strict_msi_failure_leaves_the_original_provisional_receipt(self):
        with tempfile.TemporaryDirectory(prefix='insonic-parallel-model-') as directory:
            fixture = ParallelPackageFixture(Path(directory), failure='strict')
            with self.assertRaisesRegex(ValueError, 'strict MSI verification failed'):
                fixture.invoke()
            fixture.assert_provisional(self)
            self.assertEqual(fixture.verification, ['archive', 'installer'])


if __name__ == '__main__':
    unittest.main()
