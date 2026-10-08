# SPDX-License-Identifier: Apache-2.0
import importlib.util
import io
import json
from pathlib import Path
import stat
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import zipfile

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('package_desktop', ROOT / 'scripts/package-desktop.py')
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)


class NativePackageIntegrityTests(unittest.TestCase):
    def test_macos_app_transport_exception_is_only_its_loopback_media_host(self):
        info = package.qualify.desktop_application_info()
        self.assertEqual(info['CFBundleExecutable'], 'insonic-desktop')
        self.assertEqual(info['LSMinimumSystemVersion'], '13.3')
        self.assertEqual(info['NSAppTransportSecurity'], {'NSExceptionDomains': {
            '127.0.0.1': {'NSExceptionAllowsInsecureHTTPLoads': True}}})

    def test_packaged_loader_flags_replace_development_rpaths(self):
        original = {'CGO_LDFLAGS': '-L/checkout/native -Wl,-rpath,/checkout/native',
                    'CGO_CFLAGS': '-I/checkout/native', 'CGO_ENABLED': '1'}
        for platform, loader in [('linux_amd64', '$ORIGIN/native'), ('darwin_arm64', '@loader_path/native')]:
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
                        'insonic', 'insonic-desktop', 'companions/cueson/cueson', 'companions/cueson/cueson.schema.json']
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
        fake = {'kind': 'media-tools', 'schema_version': '0.0.0', 'exiftool': {
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
        self.assertEqual(lock['source_build']['platform'], 'darwin_arm64')
        self.assertNotIn('--enable-nonfree', lock['source_build']['configure'])
        self.assertIn('--disable-autodetect', lock['source_build']['configure'])
        environment = package.package_environment()
        self.assertEqual(environment['CUESON_EXECUTABLE'], '')
        self.assertEqual(environment['INSONIC_LIBRARY_TOOLS_FILE'], '')
        self.assertNotIn('build', environment['PATH'])


if __name__ == '__main__':
    unittest.main()
