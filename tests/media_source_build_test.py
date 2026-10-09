# SPDX-License-Identifier: Apache-2.0
import io
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
import media_source_build as source


class OwnedSourceIntegrity(unittest.TestCase):
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
            (prefix / 'unrecorded.a').unlink()
            (prefix / 'lib0.a').write_bytes(b'changed')
            self.assertFalse(source.dependency_cache_valid(root, receipt, 'exact-key'))


if __name__ == '__main__':
    unittest.main()
