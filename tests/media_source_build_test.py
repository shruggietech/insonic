# SPDX-License-Identifier: Apache-2.0
import io
import os
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest
import urllib.error
from unittest.mock import Mock, patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
import media_source_build as source


class OwnedSourceIntegrity(unittest.TestCase):
    def test_dav1d_uses_available_cores_with_a_bounded_library_compile_budget(self):
        for cores, expected in [(2, '2'), (4, '4'), (64, '8')]:
            with self.subTest(cores=cores):
                root = Path('private-source-build'); run = Mock()
                source.compile_dav1d(root / 'source', root / 'build', root / 'install', cores, run)
                compilation = next(call for call in run.call_args_list if call.args[0][0] == 'ninja')
                self.assertEqual(compilation.args[0][-2:], ['-j', expected])
                self.assertEqual(compilation.args[2], 240)
                self.assertLess(compilation.args[2], 600)

    def test_source_network_retry_keeps_pin_integrity_and_bounded_timeouts(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary); payload = b'exact upstream source'
            archive = root / 'verified'; archive.write_bytes(payload)
            pin = {'name': 'source.tar', 'url': 'https://source.example/archive.tar',
                   'sha256': source.sha(archive), 'size_bytes': len(payload)}
            transient = urllib.error.URLError('network temporarily unreachable')
            with patch.object(source, 'ROOT', root), \
                 patch.object(source.urllib.request, 'urlopen', side_effect=[transient, io.BytesIO(payload)]) as fetch, \
                 patch.object(source.time, 'sleep') as sleep:
                target = source.fetch_pin(pin)
                self.assertEqual(target.read_bytes(), payload)
                self.assertEqual(fetch.call_count, 2)
                self.assertTrue(all(call.kwargs['timeout'] <= 20 for call in fetch.call_args_list))
                sleep.assert_called_once_with(1)

    def test_source_acquisition_fails_fast_for_integrity_and_permanent_http_errors(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary); archive = root / 'verified'; archive.write_bytes(b'exact')
            pin = {'name': 'source.tar', 'url': 'https://source.example/archive.tar', 'sha256': source.sha(archive)}
            for response, message in [(io.BytesIO(b'altered'), 'identity mismatch'),
                                      (urllib.error.HTTPError(pin['url'], 404, 'missing', {}, None), 'source.tar from source.example')]:
                with self.subTest(message=message), patch.object(source, 'ROOT', root), \
                     patch.object(source.urllib.request, 'urlopen', side_effect=response if isinstance(response, Exception) else [response]) as fetch, \
                     patch.object(source.time, 'sleep') as sleep:
                    with self.assertRaisesRegex((ValueError, RuntimeError), message):
                        source.fetch_pin(pin)
                    self.assertEqual(fetch.call_count, 1)
                    sleep.assert_not_called()
                    self.assertFalse((root / 'build/media-source-pins/source.download').exists())

    def test_source_transient_failure_stops_after_three_attempts_with_pin_and_host(self):
        with tempfile.TemporaryDirectory() as temporary:
            pin = {'name': 'source.tar', 'url': 'https://source.example/archive.tar', 'sha256': 'f' * 64}
            with patch.object(source, 'ROOT', Path(temporary)), \
                 patch.object(source.urllib.request, 'urlopen', side_effect=urllib.error.URLError('unreachable')) as fetch, \
                 patch.object(source.time, 'sleep'):
                with self.assertRaisesRegex(RuntimeError, 'source.tar from source.example after 3 attempt'):
                    source.fetch_pin(pin)
                self.assertEqual(fetch.call_count, 3)

    def link(self, path, target, directory=False):
        try:
            path.symlink_to(target, target_is_directory=directory)
        except OSError as error:
            if os.name == 'nt' and error.winerror == 1314:
                self.skipTest('Windows runner does not grant symbolic-link creation')
            raise

    def test_internal_aliases_are_materialized_with_exact_bytes_and_mode(self):
        with tempfile.TemporaryDirectory() as temporary:
            prefix = Path(temporary) / 'install'
            prefix.mkdir()
            target = prefix / 'xzdiff'; target.write_bytes(b'installed helper\n'); target.chmod(0o755)
            self.link(prefix / 'xzcmp', 'xzdiff')
            self.link(prefix / 'second', 'xzcmp')
            expected_mode = target.stat().st_mode
            audit = source.normalize_prefix_aliases(prefix)
            self.assertEqual([item['path'] for item in audit], ['second', 'xzcmp'])
            for name in ['second', 'xzcmp']:
                self.assertFalse((prefix / name).is_symlink())
                self.assertEqual((prefix / name).read_bytes(), target.read_bytes())
                self.assertEqual((prefix / name).stat().st_mode, expected_mode)
            self.assertEqual({item['target'] for item in audit}, {'xzdiff'})
            self.assertEqual({item['sha256'] for item in audit}, {source.sha(target)})

    def test_invalid_aliases_reject_before_any_materialization(self):
        for kind in ['external', 'directory', 'cycle', 'missing']:
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary); prefix = root / 'install'; prefix.mkdir()
                (prefix / 'target').write_bytes(b'helper')
                self.link(prefix / 'a-valid', 'target')
                if kind == 'external':
                    (root / 'outside').write_bytes(b'outside')
                    self.link(prefix / 'z-invalid', '../outside')
                elif kind == 'directory':
                    (prefix / 'subdirectory').mkdir()
                    self.link(prefix / 'z-invalid', 'subdirectory', directory=True)
                elif kind == 'cycle':
                    self.link(prefix / 'z-invalid', 'z-other')
                    self.link(prefix / 'z-other', 'z-invalid')
                else:
                    self.link(prefix / 'z-invalid', 'absent')
                with self.assertRaisesRegex(ValueError, 'dependency alias'):
                    source.normalize_prefix_aliases(prefix)
                self.assertTrue((prefix / 'a-valid').is_symlink())

    def test_wrong_windows_toolchain_fails_before_compiler_discovery(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary); (root / 'usr/bin').mkdir(parents=True)
            (root / 'usr/bin/bash.exe').write_bytes(b'placeholder')
            with patch.dict(os.environ, {'INSONIC_MSYS2_ROOT': str(root)}):
                with self.assertRaisesRegex(ValueError, 'MSYS2 source tool is missing.*gcc.exe'):
                    source.validate_build_tools('windows_amd64', source.build_environment('windows_amd64', root / 'install'))

    def test_verified_complete_cache_does_not_require_cold_build_tools(self):
        with tempfile.TemporaryDirectory() as temporary:
            build = Path(temporary); root = build / 'exact-key'; root.mkdir()
            binaries = ['ffmpeg.exe', 'ffprobe.exe']
            for name in binaries:
                (root / name).write_bytes(b'controlled binary')
            libraries = []
            for index in range(len(source.PIN['libraries']) + 1):
                path = root / ('library' + str(index) + '.a'); path.write_bytes(b'library')
                libraries.append({'path': path.name, 'sha256': source.sha(path)})
            notice = root / 'compiler-notice'; notice.write_bytes(b'permission')
            receipt = {'key': 'exact-key', 'source_complete': True, 'sources': source.source_pins(),
                       'versions': {'ffmpeg': source.VERSION, 'ffprobe': source.VERSION},
                       'binaries': {name: source.sha(root / name) for name in binaries},
                       'static_libraries': libraries,
                       'runtime_notices': [{'path': notice.name, 'sha256': source.sha(notice)}]}
            source.write_json(root / 'build-receipt.json', receipt)
            with patch.object(source, 'BUILD', build), \
                 patch.object(source, 'cache_identity', return_value=('windows_amd64', binaries, 'exact-key', 'gcc', 'recipe')), \
                 patch.object(source, 'fetch_source', return_value=build / 'archive'), \
                 patch.object(source, 'validate_build_tools') as prerequisites:
                self.assertEqual(source.prepare()[1], receipt)
                prerequisites.assert_not_called()

    def test_missing_license_is_rejected_before_any_build_command(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source_root = root / 'ffmpeg-source'
            source_root.mkdir()
            (source_root / 'COPYING.LGPLv2.1').write_bytes(b'license')
            with self.assertRaisesRegex(ValueError, 'notice missing.*LICENSE.md'):
                source.validate_source_notices(root, {'ffmpeg': {'notices': ['COPYING.LGPLv2.1', 'LICENSE.md']}})
            (source_root / 'LICENSE.md').write_bytes(b'copyright and licensing instructions')
            source.validate_source_notices(root, {'ffmpeg': {'notices': ['COPYING.LGPLv2.1', 'LICENSE.md']}})
            pins = {'ffmpeg': {'root': 'source', 'notices': ['COPYING.LGPLv2.1', 'LICENSE.md']}}
            def incomplete_archive(archive, destination, root_name):
                (destination / 'COPYING.LGPLv2.1').write_bytes(b'license')
            with patch.object(source, 'BUILD', root / 'build'), \
                 patch.object(source, 'cache_identity', return_value=('linux_amd64', ['ffmpeg', 'ffprobe'], 'key', 'gcc', 'recipe')), \
                 patch.object(source, 'fetch_source', return_value=root / 'archive.tar'), \
                 patch.object(source, 'fetch_pin', return_value=root / 'archive.tar'), \
                 patch.object(source, 'source_pins', return_value=pins), \
                 patch.object(source, 'extract_source', side_effect=incomplete_archive), \
                 patch.object(source, 'child') as command:
                with self.assertRaisesRegex(ValueError, 'notice missing.*LICENSE.md'):
                    source.prepare()
                command.assert_not_called()

    def test_release_source_extraction_preserves_generated_file_timestamps(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / 'source.tar'
            with tarfile.open(archive, 'w') as output:
                for name, mtime in [('configure.ac', 100000), ('configure', 100100)]:
                    item = tarfile.TarInfo('source/' + name)
                    item.size = 1; item.mtime = mtime
                    output.addfile(item, io.BytesIO(b'x'))
            source.extract_source(archive, root / 'output', 'source')
            self.assertLess((root / 'output/configure.ac').stat().st_mtime, (root / 'output/configure').stat().st_mtime)

    def test_source_duplicates_links_devices_and_windows_paths_fail(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / 'source.tar'
            for name, kind in [('source/C:/outside', tarfile.REGTYPE), ('source/x', tarfile.SYMTYPE), ('source/x', tarfile.CHRTYPE)]:
                with tarfile.open(archive, 'w') as output:
                    item = tarfile.TarInfo(name); item.type = kind
                    if kind == tarfile.REGTYPE:
                        item.size = 1; output.addfile(item, io.BytesIO(b'x'))
                    else:
                        item.linkname = '../outside'; output.addfile(item)
                with self.assertRaises(ValueError):
                    source.extract_source(archive, root / 'output', 'source')
            with tarfile.open(archive, 'w') as output:
                for name in ['source/file', 'source/./file']:
                    item = tarfile.TarInfo(name); item.size = 1; output.addfile(item, io.BytesIO(b'x'))
            with self.assertRaises(ValueError):
                source.extract_source(archive, root / 'duplicate', 'source')

    def test_required_external_capabilities_cannot_be_replaced_by_header_text(self):
        output = 'Demuxers:\n D  matroska,webm  Matroska input\n D  libopenmpt  Module input\n'
        self.assertIn('webm', source.validate_capabilities(output, ['matroska', 'libopenmpt'], 'demuxers'))
        with self.assertRaisesRegex(ValueError, 'libdav1d'):
            source.validate_capabilities('Decoder libdav1d\n V....D h264 native\n', ['libdav1d'], 'decoders')

    def test_cache_needs_unchanged_binaries_and_dependency_archives(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            binaries = ['ffmpeg', 'ffprobe']
            for name in binaries:
                (root / name).write_bytes(b'controlled binary')
            libraries = []
            for index in range(len(source.PIN['libraries']) + 1):
                path = root / ('library' + str(index) + '.a'); path.write_bytes(b'controlled library')
                libraries.append({'path': path.name, 'sha256': source.sha(path)})
            receipt = {'key': 'exact-key', 'source_complete': True, 'sources': source.source_pins(),
                       'versions': {'ffmpeg': source.VERSION, 'ffprobe': source.VERSION},
                       'binaries': {name: source.sha(root / name) for name in binaries}, 'static_libraries': libraries}
            self.assertTrue(source.cache_valid(root, receipt, 'exact-key', binaries))
            (root / 'library0.a').write_bytes(b'other compiled library')
            self.assertFalse(source.cache_valid(root, receipt, 'exact-key', binaries))
            (root / 'library0.a').write_bytes(b'controlled library')
            (root / 'ffmpeg').write_bytes(b'other binary')
            self.assertFalse(source.cache_valid(root, receipt, 'exact-key', binaries))

    def test_dependency_transfer_requires_complete_exact_private_install(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary); prefix = root / 'install/lib'
            prefix.mkdir(parents=True)
            for index in range(len(source.PIN['libraries']) + 1):
                (prefix / ('lib' + str(index) + '.a')).write_bytes(b'controlled archive')
            files = {path.relative_to(root).as_posix(): source.sha(path) for path in prefix.rglob('*')}
            receipt = {'key': 'exact-key', 'kind': 'media-dependency-build', 'sources': source.source_pins(),
                       'install_prefix': str(root / 'install'), 'files': files}
            self.assertTrue(source.dependency_cache_valid(root, receipt, 'exact-key'))
            (prefix / 'unrecorded.a').write_bytes(b'unrecorded')
            self.assertFalse(source.dependency_cache_valid(root, receipt, 'exact-key'))
            self.assertIn('unrecorded member: install/lib/unrecorded.a', source.dependency_cache_errors(root, receipt, 'exact-key'))
            (prefix / 'unrecorded.a').unlink()
            (prefix / 'lib0.a').write_bytes(b'changed')
            self.assertFalse(source.dependency_cache_valid(root, receipt, 'exact-key'))
            self.assertIn('member digest differs: install/lib/lib0.a', source.dependency_cache_errors(root, receipt, 'exact-key'))


if __name__ == '__main__':
    unittest.main()
