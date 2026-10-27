# SPDX-License-Identifier: Apache-2.0
import hashlib
import importlib.util
import io
import json
from copy import deepcopy
from pathlib import Path
import stat
import subprocess
import tarfile
import tempfile
import unittest
from unittest.mock import patch
from types import SimpleNamespace
import zipfile

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('package_desktop', ROOT / 'scripts/package-desktop.py')
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)


class NativePackageIntegrityTests(unittest.TestCase):
    def test_archive_compiler_authority_comes_from_selected_historical_sources(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);historical=root/'historical'
            (historical/'scripts').mkdir(parents=True)
            (historical/'internal/qualification').mkdir(parents=True)
            (historical/'scripts/processing-worker.py').write_bytes(b'worker')
            pins=historical/'internal/qualification/dependencies.json'
            pins.write_text(json.dumps({'go':'1.26.2'}))
            archive=root/'fixture.zip';archive.write_bytes(b'archive')
            manifest=b'{}';media=json.dumps({'binaries':{}}).encode()
            files={'companions/processing/processing-worker.py':b'worker','sources/source-manifest.json':manifest,'sources/ffmpeg-build.json':media}
            receipt={'archive_sha256':package.sha(archive),'archive_size_bytes':archive.stat().st_size,
                     'inventory_root':'program','inventory_path':'program/package-inventory.json','dirname':'program',
                     'revision':'a'*40,'platform':'windows_amd64','variant':'desktop','schema_version':package.VERSION,
                     'source_dirty':False,'distribution':{'source_manifest_sha256':hashlib.sha256(manifest).hexdigest()},'signing':{'configured':False}}
            value={key:receipt[key] for key in ['revision','platform','variant','schema_version','source_dirty','distribution','signing']}
            value.update(dependencies={'go':'1.26.2','modules':[]},files=[{'path':name} for name in files])
            raw=json.dumps(value).encode();receipt['inventory_sha256']=__import__('hashlib').sha256(raw).hexdigest()
            def extract(archive,location):
                for name,data in dict(files,**{'package-inventory.json':raw}).items():
                    target=location/'program'/name;target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(data)
            with patch.object(package,'extract_package',side_effect=extract),patch.object(package,'verify_inventory'),patch.object(package,'product_binaries',return_value=[]),patch.object(package,'verify_corresponding_sources',return_value={}) as sources,patch.object(package.go_module_notices,'verify_notices') as notices:
                package.verify_archive_receipt(archive,receipt,expected_source_root=historical)
                self.assertEqual(notices.call_args.args[-1],'1.26.2')
                self.assertEqual(sources.call_args.kwargs['expected_source_root'],historical)
                pins.write_text(json.dumps({'go':'1.25.0'}))
                with self.assertRaisesRegex(ValueError,'compiler inventory'):
                    package.verify_archive_receipt(archive,receipt,expected_source_root=historical)

    def test_native_timeout_retains_bounded_checkpoints_and_original_deadline(self):
        for cleanup_timeout in (False, True):
            with self.subTest(cleanup_timeout=cleanup_timeout), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                output = b'x' * 70000 + b'controlled stdout checkpoint\n'
                error = b'y' * 70000 + b'Desktop qualification stage: audio-playback\n'
                expired = subprocess.TimeoutExpired(['private/executable', '--webview-qualification'], 90,
                                                    output=output, stderr=error)
                class Tree:
                    def __init__(self, *args, **kwargs):
                        self.process = self; self.pid = 1234; self.calls = []; self.killed = False
                        self.returncode = None
                    def communicate(self, timeout):
                        self.calls.append(timeout)
                        if timeout == 90: raise expired
                        if cleanup_timeout:
                            raise subprocess.TimeoutExpired('cleanup', 5, output=output, stderr=error)
                        self.returncode = 1
                        return output, error
                    def poll(self): return self.returncode
                    def kill(self): self.killed = True
                    def __enter__(self): return self
                    def __exit__(self, *args): pass
                tree = Tree()
                stdout, stderr = io.BytesIO(), io.BytesIO()
                with patch.object(package, 'ROOT', root), patch.object(package, 'ProcessTree', return_value=tree), \
                        patch.object(package.sys, 'stdout', SimpleNamespace(buffer=stdout)), \
                        patch.object(package.sys, 'stderr', SimpleNamespace(buffer=stderr)):
                    with self.assertRaises(subprocess.TimeoutExpired) as failure:
                        package.child(['private/executable', '--webview-qualification'], timeout=90)
                self.assertIs(failure.exception, expired)
                self.assertEqual(tree.calls, [90, package.timing.CLEANUP_SECONDS]); self.assertTrue(tree.killed)
                self.assertEqual(stdout.getvalue(), output[-65536:])
                self.assertEqual(stderr.getvalue(), error[-65536:])
                receipt = json.loads((root / 'build/native/package-child-timeout-1234-receipt.json').read_text())
                self.assertEqual(receipt['stage'], 'native-webview')
                self.assertEqual(receipt['timeout_seconds'], 90)
                self.assertIsNone(receipt['poll_before_kill'])
                self.assertEqual(receipt['poll_after_kill'], None if cleanup_timeout else 1)
                self.assertEqual(receipt['cleanup_timeout'], cleanup_timeout)
                self.assertEqual(receipt['stdout_bytes'], len(output))
                self.assertNotIn('private', json.dumps(receipt))
                self.assertNotIn('audio-playback', json.dumps(receipt))

    def test_detached_idle_wait_keeps_deadline_and_unknown_failure_allowance(self):
        with patch.object(package.time, 'monotonic', return_value=110), patch.object(package.time, 'sleep') as slept:
            package.wait_for_detached_idle(100)
            slept.assert_called_once_with(21)
        with patch.object(package.time, 'monotonic', return_value=145), patch.object(package.time, 'sleep') as slept:
            package.wait_for_detached_idle(100)
            slept.assert_not_called()
        with patch.object(package.time, 'sleep') as slept:
            package.wait_for_detached_idle()
            slept.assert_called_once_with(31)

    def test_detached_idle_wait_observes_image_release_and_preserves_failure(self):
        with patch.object(package.platform, 'system', return_value='Windows'), \
                patch.object(package.time, 'monotonic', side_effect=[145, 145, 145.2, 145.3]), \
                patch.object(package.time, 'sleep') as slept, \
                patch.object(package, 'windows_image_released', side_effect=[False, False, True]) as released:
            package.wait_for_detached_idle(100, Path('fixture.exe'))
            self.assertEqual(released.call_count, 3)
            self.assertEqual(slept.call_args_list, [unittest.mock.call(0.1), unittest.mock.call(0.1)])
        with patch.object(package.platform, 'system', return_value='Windows'), \
                patch.object(package.time, 'monotonic', side_effect=[145, 145, 145 + package.timing.NATURAL_RELEASE_SECONDS]), \
                patch.object(package, 'windows_image_released', return_value=False), \
                patch.object(package, 'retain_busy_image_diagnostic') as diagnostic:
            with self.assertRaisesRegex(TimeoutError, 'image remained busy'):
                package.wait_for_detached_idle(100, Path('fixture.exe'))
            diagnostic.assert_called_once_with(Path('fixture.exe'))

    @unittest.skipUnless(package.os.name == 'nt', 'Windows executable image semantics')
    def test_image_release_probe_observes_loaded_image_without_modifying_files(self):
        import ctypes
        kernel = ctypes.WinDLL('kernel32', use_last_error=True)
        kernel.GetModuleFileNameW.argtypes = [ctypes.c_void_p, ctypes.c_wchar_p, ctypes.c_uint32]
        kernel.GetModuleFileNameW.restype = ctypes.c_uint32
        image_path = ctypes.create_unicode_buffer(32768)
        self.assertGreater(kernel.GetModuleFileNameW(None, image_path, len(image_path)), 0)
        self.assertFalse(package.windows_image_released(Path(image_path.value)))
        with tempfile.TemporaryDirectory() as temporary:
            image = Path(temporary) / 'unused.exe'
            image.write_bytes(b'unchanged image bytes')
            self.assertTrue(package.windows_image_released(image))
            self.assertEqual(image.read_bytes(), b'unchanged image bytes')
            image.unlink()

    def test_package_idle_allowance_starts_after_late_gui_activity_even_on_failure(self):
        for fail_gui in (False, True):
            events = []
            def joined():
                events.append('gui-complete')
                if fail_gui:
                    raise ValueError('actual GUI failure')
            future = SimpleNamespace(result=joined)
            def waited(last, executable):
                self.assertIsNone(last)
                self.assertEqual(executable, Path('fixture.exe'))
                events.append('full-idle-allowance')
            with self.subTest(fail_gui=fail_gui), patch.object(package, 'wait_for_detached_idle', side_effect=waited):
                if fail_gui:
                    with self.assertRaisesRegex(ValueError, 'actual GUI failure'):
                        package.finish_package_checks(future, 100, Path('fixture.exe'))
                else:
                    package.finish_package_checks(future, 100, Path('fixture.exe'))
            self.assertEqual(events, ['gui-complete', 'full-idle-allowance'])
        with patch.object(package, 'wait_for_detached_idle') as waited:
            package.finish_package_checks(None, 100, Path('fixture.exe'))
            waited.assert_called_once_with(100, Path('fixture.exe'))

    def test_busy_image_diagnostic_retains_only_bounded_owner_categories(self):
        with tempfile.TemporaryDirectory() as temporary, patch.object(package, 'ROOT', Path(temporary)), \
                patch.object(package, 'windows_image_owners', return_value=[{'pid': 42, 'parent_pid': 7, 'role': 'main', 'command_line': 'private fixture path'}]):
            package.retain_busy_image_diagnostic(Path(temporary) / 'fixture.exe')
            receipt = json.loads((Path(temporary) / 'build/native/package-image-busy-receipt.json').read_text())
            self.assertEqual(receipt['owners'], [{'pid': 42, 'parent_pid': 7, 'role': 'main'}])
            self.assertNotIn('private', json.dumps(receipt))

    def test_owned_gui_probe_joins_before_library_restoration_even_on_cli_failure(self):
        import threading
        for fail_cli in (False, True):
            with self.subTest(fail_cli=fail_cli), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary); original = root / 'build/native/ladybug'
                original.mkdir(parents=True); (original / 'retained.bin').write_bytes(b'native bytes')
                sibling = root / 'unrelated'; sibling.write_bytes(b'preserved')
                probe_started = threading.Event(); release_probe = threading.Event()
                cli_finished = threading.Event(); scope_finished = threading.Event(); failures = []
                def probe(*args):
                    if original.exists(): raise AssertionError('native development tree exposed')
                    probe_started.set()
                    if not release_probe.wait(3): raise AssertionError('test probe release missing')
                    if original.exists(): raise AssertionError('tree restored before owned probe finished')
                def journey():
                    try:
                        with package.isolated_development_libraries(), package.background_gui_qualification('gui', root, {}) as future:
                            if not probe_started.wait(2): raise AssertionError('probe did not start')
                            cli_finished.set()
                            if fail_cli: raise ValueError('actual CLI journey failure')
                            future.result()
                    except BaseException as error: failures.append(error)
                    finally: scope_finished.set()
                with patch.object(package, 'ROOT', root), patch.object(package, 'qualify_package_gui', side_effect=probe):
                    thread = threading.Thread(target=journey); thread.start()
                    try:
                        self.assertTrue(cli_finished.wait(2))
                        self.assertFalse(scope_finished.is_set()); self.assertFalse(original.exists())
                        self.assertEqual(sibling.read_bytes(), b'preserved')
                    finally:
                        release_probe.set(); thread.join(3)
                    self.assertFalse(thread.is_alive()); self.assertTrue(scope_finished.is_set())
                self.assertEqual((original / 'retained.bin').read_bytes(), b'native bytes')
                self.assertEqual(sibling.read_bytes(), b'preserved')
                self.assertEqual([str(error) for error in failures], ['actual CLI journey failure'] if fail_cli else [])

    def test_install_text_is_platform_and_variant_specific(self):
        for key in ('windows_amd64','linux_amd64'):
            for variant in ('cli','desktop'):
                with tempfile.TemporaryDirectory() as temporary:
                    root = Path(temporary); package.installers(root,key,variant)
                    text = (root / 'INSTALL.txt').read_text(encoding='utf-8')
                    self.assertNotIn('qualification package',text)
                    self.assertNotIn('macOS',text)
                    self.assertNotIn('notarization',text)
                    self.assertIn('workspace init',text)
                    self.assertIn('doctor',text)
                    if key.startswith('windows'):
                        self.assertIn('Microsoft Visual C++ x64',text)
                        self.assertIn('no PATH',text)
                        self.assertEqual('WebView2' in text,variant=='desktop')
                        self.assertIn('unsigned',text)
                    else:
                        self.assertEqual('GTK 3' in text,variant=='desktop')
                        self.assertEqual('insonic.desktop' in (root/'install.sh').read_text(),variant=='desktop')

    def test_historical_macos_archive_layout_remains_validation_data(self):
        root = Path('/historical/package/Contents')
        self.assertEqual(package.package_directories(root,'darwin_arm64','desktop'),(root/'MacOS',root/'Resources'))

    def test_owned_builder_rejects_retired_macos_before_side_effects(self):
        with patch.object(package.platform, "system", return_value="Darwin"), patch.object(package.platform, "machine", return_value="arm64"):
            with self.assertRaisesRegex(ValueError, "unsupported.*platform|platform.*unsupported"):
                package.platform_key()


    def test_both_variants_require_an_inventoried_processing_worker(self):
        for variant in ('cli', 'desktop'):
            with tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                names = ['insonic-companions.json', 'LICENSE', 'NOTICE', 'help/index.html', 'INSTALL.txt',
                         'insonic', 'companions/cueson/cueson', 'companions/cueson/cueson.schema.json']
                if variant == 'desktop': names.append('insonic-desktop')
                for name in names:
                    path = root / name; path.parent.mkdir(parents=True, exist_ok=True); path.write_bytes(b'fixture')
                value = {'platform': 'linux_amd64', 'variant': variant, 'files': [
                    {'path': name, 'size_bytes': 7, 'sha256': package.sha(root / name)} for name in names]}
                with self.assertRaisesRegex(ValueError, 'required'): package.verify_inventory(root, value)


    def test_processing_metadata_and_hash_locks_are_verified_as_a_closure(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            names = ['insonic-companions.json', 'LICENSE', 'NOTICE', 'help/index.html', 'INSTALL.txt',
                     'insonic', 'companions/cueson/cueson', 'companions/cueson/cueson.schema.json',
                     'companions/processing/processing-worker.py', 'companions/processing/probe.py',
                     'companions/processing/models.json', 'companions/processing/requirements.txt']
            for name in names:
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(b'fixture')
            processing = root / 'companions/processing'
            profile = {'id': 'recognition', 'platform': 'linux_amd64', 'python': '3.12.15',
                       'engines': {}, 'lock': 'requirements.txt', 'lock_sha256': package.sha(processing / 'requirements.txt')}
            package.write_json(processing / 'catalog.json', {'profiles': [profile]})
            names.append('companions/processing/catalog.json')
            value = {'platform': 'linux_amd64', 'variant': 'cli', 'dependencies': {'processing_workers': {
                'catalog_sha256': package.sha(processing / 'catalog.json'),
                'model_catalog_sha256': package.sha(processing / 'models.json'),
                'profiles': [{key: profile[key] for key in ['id', 'platform', 'python', 'engines', 'lock_sha256']}]}},
                'files': [{'path': name, 'size_bytes': (root / name).stat().st_size, 'sha256': package.sha(root / name)} for name in names]}
            package.verify_inventory(root, value)
            # Rewriting the ordinary file inventory cannot authorize another lock.
            lock = processing / 'requirements.txt'
            lock.write_bytes(b'other dependency selection')
            entry = next(item for item in value['files'] if item['path'].endswith('requirements.txt'))
            entry.update(size_bytes=lock.stat().st_size, sha256=package.sha(lock))
            with self.assertRaisesRegex(ValueError, 'hash lock'):
                package.verify_inventory(root, value)

    def source_fixture(self, root):
        target = root / 'sources'
        target.mkdir()
        pin = deepcopy(package.source_module().PIN)
        sources = package.source_module().source_pins()
        sources = deepcopy(sources)
        for name, source in sources.items():
            path = target / source['name']
            path.write_bytes(('controlled corresponding source ' + name).encode())
            source['sha256'] = package.sha(path)
            if name == 'ffmpeg':
                pin['sha256'] = source['sha256']
            elif name == 'lame':
                pin['lame'] = source
            else:
                pin['libraries'][name] = source
        for name in ['build-media-source.py', 'media_source_build.py', 'process_tree.py', 'REBUILD.txt']:
            (target / name).write_bytes(b'controlled rebuild recipe')
        package.write_json(target / 'media-tools.json', {'source_build': pin})
        receipt = {'source_complete': True, 'sources': sources, 'platform': 'linux_amd64', 'compiler': 'fixture GCC',
                   'binaries': {'ffmpeg': 'a' * 64, 'ffprobe': 'b' * 64},
                   'versions': {'ffmpeg': pin['version'], 'ffprobe': pin['version']},
                   'launcher_sha256': package.sha(target / 'process_tree.py'),
                   'recipe_sha256': package.sha(target / 'media_source_build.py')}
        package.write_json(target / 'ffmpeg-build.json', receipt)
        manifest = {'kind': 'corresponding-sources', 'source_complete': True, 'sources': sources,
                    'redistribution_closure': package.media_runtime_permissions(receipt),
                    'build_receipt': 'ffmpeg-build.json', 'files': [
                        {'path': path.name, 'size_bytes': path.stat().st_size, 'sha256': package.sha(path)}
                        for path in sorted(target.iterdir())]}
        package.write_json(target / 'source-manifest.json', manifest)
        return manifest

    def test_complete_source_closure_detects_missing_changed_and_extra_sources(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifest = self.source_fixture(root)
            package.verify_corresponding_sources(root)
            filename = manifest['sources']['openmpt']['name']
            original = (root / 'sources' / filename).read_bytes()
            (root / 'sources' / filename).write_bytes(b'other source version')
            with self.assertRaisesRegex(ValueError, 'bytes differ'):
                package.verify_corresponding_sources(root)
            (root / 'sources' / filename).write_bytes(original)
            (root / 'sources/extra-source').write_bytes(b'not inventoried')
            with self.assertRaisesRegex(ValueError, 'unrecorded'):
                package.verify_corresponding_sources(root)
            (root / 'sources/extra-source').unlink()
            (root / 'sources' / filename).unlink()
            with self.assertRaisesRegex(ValueError, 'bytes differ'):
                package.verify_corresponding_sources(root)

    def test_a_claimed_complete_receipt_cannot_omit_an_external_decoder(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifest = self.source_fixture(root)
            del manifest['sources']['dav1d']
            # Even rewriting the receipt and its hash cannot make an incomplete
            # source closure agree with the configured decoder inputs.
            receipt_path = root / 'sources/ffmpeg-build.json'
            receipt = json.loads(receipt_path.read_text())
            del receipt['sources']['dav1d']
            manifest['redistribution_closure'] = package.media_runtime_permissions(receipt)
            package.write_json(receipt_path, receipt)
            for entry in manifest['files']:
                if entry['path'] == 'ffmpeg-build.json':
                    entry.update(sha256=package.sha(receipt_path), size_bytes=receipt_path.stat().st_size)
            with self.assertRaisesRegex(ValueError, 'complete dependency closure'):
                package.verify_corresponding_sources(root, manifest)

    def test_self_consistent_old_source_recipe_cannot_qualify_current_checkout(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifest = self.source_fixture(root)
            expected = root / 'checkout'
            (expected / 'internal/qualification').mkdir(parents=True)
            (expected / 'scripts').mkdir()
            (expected / 'internal/qualification/media-tools.json').write_bytes((root / 'sources/media-tools.json').read_bytes())
            for name in ('build-media-source.py', 'media_source_build.py', 'process_tree.py'):
                (expected / 'scripts' / name).write_bytes((root / 'sources' / name).read_bytes())
            (expected / 'scripts/media_source_build.py').write_bytes(b'current exact recipe')
            with self.assertRaisesRegex(ValueError, 'selected checkout'):
                package.verify_corresponding_sources(root, manifest, expected_source_root=expected)

    def test_missing_dependency_groups_cannot_qualify_current_checkout(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifest = self.source_fixture(root)
            expected = root / 'checkout'
            (expected / 'internal/qualification').mkdir(parents=True)
            (expected / 'scripts').mkdir()
            (expected / 'internal/qualification/media-tools.json').write_bytes((root / 'sources/media-tools.json').read_bytes())
            for name in ('build-media-source.py', 'media_source_build.py', 'process_tree.py'):
                (expected / 'scripts' / name).write_bytes((root / 'sources' / name).read_bytes())
            with self.assertRaisesRegex(ValueError, 'all dependency groups'):
                package.verify_corresponding_sources(root, manifest, expected_source_root=expected)

    def test_configured_signing_failure_never_becomes_verified(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'insonic.exe').write_bytes(b'controlled native program')
            with patch.dict(package.os.environ, {'INSONIC_WINDOWS_SIGN_CERT_SHA1': 'a' * 40}), \
                 patch.object(package, 'child', side_effect=RuntimeError('signer rejected identity')):
                with self.assertRaisesRegex(RuntimeError, 'rejected'):
                    package.sign_native(root, 'windows_amd64')
            with patch.dict(package.os.environ, {'INSONIC_WINDOWS_SIGN_CERT_SHA1': ''}):
                self.assertEqual(package.sign_native(root, 'windows_amd64')['status'], 'unconfigured')

    def test_cli_inventory_does_not_require_a_gui_executable(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            names = ['insonic-companions.json', 'LICENSE', 'NOTICE', 'help/index.html', 'INSTALL.txt',
                     'insonic', 'companions/cueson/cueson', 'companions/cueson/cueson.schema.json',
                     'companions/processing/processing-worker.py']
            for name in names:
                path = root / name; path.parent.mkdir(parents=True, exist_ok=True); path.write_bytes(b'fixture')
            value = {'platform': 'linux_amd64', 'variant': 'cli', 'files': [
                {'path': name, 'size_bytes': 7, 'sha256': package.sha(root / name)} for name in names]}
            package.verify_inventory(root, value)
            value['variant'] = 'desktop'
            with self.assertRaisesRegex(ValueError, 'required'):
                package.verify_inventory(root, value)


    def test_windows_startup_graph_library_is_beside_executables(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'native').mkdir()
            (root / 'native/lbug_shared.dll').write_bytes(b'fixture')
            with self.assertRaisesRegex(ValueError, 'beside'):
                package.verify_loader_paths(root, 'windows_amd64')
            (root / 'native/lbug_shared.dll').replace(root / 'lbug_shared.dll')
            package.verify_loader_paths(root, 'windows_amd64')

    def test_packaged_loader_flags_replace_development_rpaths(self):
        original = {'CGO_LDFLAGS': '-L/checkout/native -Wl,-rpath,/checkout/native',
                    'CGO_CFLAGS': '-I/checkout/native', 'CGO_ENABLED': '1'}
        for platform, loader in [('linux_amd64', '$ORIGIN/native')]:
            flags = package.package_build_environment(original, platform, Path('/installation with spaces/native'))
            self.assertIn('-rpath,' + loader, flags['CGO_LDFLAGS'])
            self.assertNotIn('/checkout', flags['CGO_LDFLAGS'])
        self.assertEqual(original['CGO_LDFLAGS'], '-L/checkout/native -Wl,-rpath,/checkout/native')

    def test_development_library_isolation_restores_after_success_and_failure(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / 'build/native/ladybug'
            source.mkdir(parents=True)
            (source / 'library').write_bytes(b'exact original library')
            with patch.object(package, 'ROOT', root):
                with package.isolated_development_libraries():
                    self.assertFalse(source.exists())
                    self.assertEqual(len(list(source.parent.glob('ladybug-isolated-*'))), 1)
                with self.assertRaisesRegex(RuntimeError, 'controlled failure'):
                    with package.isolated_development_libraries():
                        self.assertFalse(source.exists())
                        raise RuntimeError('controlled failure')
            self.assertEqual((source / 'library').read_bytes(), b'exact original library')
            self.assertFalse(list(source.parent.glob('ladybug-isolated-*')))

    def test_relocated_archive_can_prove_library_isolation_on_an_sdk_free_host(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / 'build/native/ladybug'
            source.parent.mkdir(parents=True)
            with patch.object(package, 'ROOT', root):
                with package.isolated_development_libraries():
                    self.assertFalse(source.exists())
                self.assertFalse(source.exists())
                with self.assertRaisesRegex(RuntimeError, 'controlled failure'):
                    with package.isolated_development_libraries():
                        raise RuntimeError('controlled failure')
                self.assertFalse(source.exists())
                with self.assertRaisesRegex(ValueError, 'recreated'):
                    with package.isolated_development_libraries():
                        source.mkdir()

    def test_extracted_zip_preserves_exact_bytes_and_executable_mode(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / 'package.zip'
            with zipfile.ZipFile(archive, 'w') as output:
                entry = zipfile.ZipInfo('application with spaces/cli')
                entry.external_attr = (stat.S_IFREG | 0o755) << 16
                output.writestr(entry, b'controlled native bytes')
            package.extract_package(archive, root / 'relocated with spaces')
            self.assertEqual((root / 'relocated with spaces/application with spaces/cli').read_bytes(), b'controlled native bytes')

    def test_archive_traversal_links_devices_and_duplicates_fail(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name in ['../outside', '/outside', 'C:/outside', '..\\outside']:
                archive = root / 'invalid.zip'
                with zipfile.ZipFile(archive, 'w') as output:
                    output.writestr(name, b'outside')
                with self.assertRaises(ValueError):
                    package.extract_package(archive, root / 'output')
            archive = root / 'link.tar.gz'
            with tarfile.open(archive, 'w:gz') as output:
                entry = tarfile.TarInfo('linked'); entry.type = tarfile.SYMTYPE; entry.linkname = '../outside'
                output.addfile(entry)
            with self.assertRaises(ValueError):
                package.extract_package(archive, root / 'tar output')
            archive = root / 'link.zip'
            with zipfile.ZipFile(archive, 'w') as output:
                entry = zipfile.ZipInfo('linked'); entry.external_attr = (stat.S_IFLNK | 0o777) << 16
                output.writestr(entry, b'../outside')
            with self.assertRaises(ValueError):
                package.extract_package(archive, root / 'zip link output')

    def test_inventory_detects_changed_missing_and_unrecorded_files(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            required = ['insonic-companions.json', 'LICENSE', 'NOTICE', 'help/index.html', 'INSTALL.txt',
                        'insonic', 'insonic-desktop', 'companions/cueson/cueson', 'companions/cueson/cueson.schema.json',
                        'companions/processing/processing-worker.py']
            for name in required:
                path = package.path_inside(root, name)
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(b'fixture')
            value = {'platform': 'linux_amd64', 'files': [
                {'path': name, 'size_bytes': 7, 'sha256': package.sha(root / name)} for name in required]}
            package.verify_inventory(root, value)
            original = (root / 'LICENSE').read_bytes()
            (root / 'LICENSE').write_bytes(b'tampered')
            with self.assertRaises(ValueError):
                package.verify_inventory(root, value)
            (root / 'LICENSE').write_bytes(original)
            (root / 'extra').write_bytes(b'extra')
            with self.assertRaises(ValueError):
                package.verify_inventory(root, value)
            (root / 'extra').unlink()
            (root / 'NOTICE').unlink()
            with self.assertRaises(ValueError):
                package.verify_inventory(root, value)

    def test_package_tool_defaults_are_relative_and_drop_development_interpreter(self):
        fake = {'kind': 'media-tools', 'schema_version': '1.0.2', 'exiftool': {
            'path': str(ROOT / 'build/native/media/exiftool/exiftool'), 'sha256': 'a' * 64, 'version': '13.59',
            'interpreter': {'path': '/usr/bin/perl', 'sha256': 'b' * 64}, 'support_files': [
                {'path': str(ROOT / 'build/native/media/exiftool/lib/support.pm'), 'sha256': 'c' * 64}]},
            'ffprobe': {'path': str(ROOT / 'build/native/media/ffprobe'), 'sha256': 'd' * 64, 'version': '7.0.2'},
            'ffmpeg': {'path': str(ROOT / 'build/native/media/ffmpeg'), 'sha256': 'e' * 64, 'version': '7.0.2'}}
        # The real relocated smoke tests discovery. This isolates manifest
        # construction so development interpreter paths never become package pins.
        from unittest.mock import patch
        original = Path.read_text
        def read(path, *args, **kwargs):
            if path == ROOT / 'build/native/media-tools.json':
                return json.dumps(fake)
            return original(path, *args, **kwargs)
        with patch.object(Path, 'read_text', read), patch.object(package, 'platform_key', return_value='linux_amd64'), \
             patch.object(package.platform, 'system', return_value='Linux'), patch.object(package, 'sha', return_value='f' * 64), \
             patch.object(Path, 'rglob', return_value=iter([ROOT / 'build/native/cueson/cueson'])):
            manifest = package.installed_manifest(ROOT)
        self.assertNotIn('interpreter', manifest['media']['exiftool'])
        self.assertTrue(manifest['system_perl'])
        for name in ['exiftool', 'ffprobe', 'ffmpeg']:
            self.assertTrue(manifest['media'][name]['path'].startswith('companions/media/'))
        self.assertEqual(manifest['media']['exiftool']['support_files'][0]['path'], 'companions/media/exiftool/lib/support.pm')

    def test_qualified_architectures_and_system_source_configuration(self):
        lock = json.loads((ROOT / 'internal/qualification/media-tools.json').read_text())
        self.assertEqual(set(lock['source_build']['platforms']), {'windows_amd64', 'linux_amd64'})
        self.assertNotIn('--enable-nonfree', lock['source_build']['configure'])
        self.assertIn('--disable-autodetect', lock['source_build']['configure'])
        environment = package.package_environment()
        self.assertEqual(environment['CUESON_EXECUTABLE'], '')
        self.assertEqual(environment['INSONIC_LIBRARY_TOOLS_FILE'], '')
        self.assertNotIn('build', environment['PATH'])


if __name__ == '__main__':
    unittest.main()
