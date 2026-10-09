# SPDX-License-Identifier: Apache-2.0
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import shutil
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / 'scripts'))
spec = importlib.util.spec_from_file_location('media_transfer', ROOT / 'scripts/media-transfer.py')
transfer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transfer)
source = transfer.source
REVISION, KEY = '1' * 40, '2' * 64


class MediaTransferTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.directory = self.root / 'build/native/source-media' / KEY
        self.directory.mkdir(parents=True)
        archive = self.root / 'build/media-source-pins/pin.tar.gz'
        archive.parent.mkdir(parents=True)
        with tarfile.open(archive, 'w:gz') as output:
            member = tarfile.TarInfo('root/NOTICE')
            data = b'original upstream notice\n'
            member.size, member.mode = len(data), 0o644
            output.addfile(member, io.BytesIO(data))
        self.pins = {'ffmpeg': {'name': archive.name, 'sha256': source.sha(archive), 'root': 'root'}}
        common = {'key': KEY, 'platform': 'linux_amd64', 'compiler': 'fixture compiler',
                  'recipe_sha256': '3' * 64, 'sources': self.pins}
        files = {'install/lib/libdependency.a': b'owned dependency archive',
                 'install/include/dependency.h': b'header', 'install/lib/pkgconfig/library.pc': b'private prefix'}
        for name, data in files.items():
            path = self.directory / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
        dependencies = dict(common, kind='media-dependency-build',
                            install_prefix=str(self.directory / 'install'),
                            files={name: source.sha(self.directory / name) for name in files})
        source.write_json(self.directory / 'dependency-receipt.json', dependencies)
        for name in ['ffmpeg', 'ffprobe']:
            (self.directory / name).write_bytes(name.encode())
            (self.directory / name).chmod(0o755)
        complete = dict(common, source_complete=True, versions={'ffmpeg': source.VERSION, 'ffprobe': source.VERSION},
                        binaries={name: source.sha(self.directory / name) for name in ['ffmpeg', 'ffprobe']},
                        static_libraries=[{'path': 'install/lib/libdependency.a',
                                           'sha256': source.sha(self.directory / 'install/lib/libdependency.a')}],
                        runtime_notices=[])
        source.write_json(self.directory / 'build-receipt.json', complete)
        for name, value in [('ROOT', self.root), ('BUILD', self.root / 'build/native/source-media'),
                            ('PIN', dict(source.PIN, libraries={})), ('source_pins', lambda: self.pins),
                            ('cache_identity', lambda: ('linux_amd64', ['ffmpeg', 'ffprobe'], KEY,
                                                        'fixture compiler', '3' * 64)),
                            ('child', lambda *args, **kwargs: (REVISION + '\n').encode())]:
            change = patch.object(source, name, value)
            change.start()
            self.addCleanup(change.stop)

    def repack(self, archive, change):
        with tarfile.open(archive) as original:
            members = [(member, original.extractfile(member).read()) for member in original]
        changed = archive.with_name('changed.tar.gz')
        with tarfile.open(changed, 'w:gz') as output:
            for member, data in members:
                member, data = change(member, data)
                if member is not None:
                    member.size = len(data)
                    output.addfile(member, io.BytesIO(data))
        return changed

    def test_dependencies_round_trip_preserves_entire_private_prefix(self):
        archive = self.root / 'dependencies.tar.gz'
        transfer.pack('dependencies', REVISION, archive)
        shutil.rmtree(self.directory)
        transfer.restore('dependencies', REVISION, archive)
        self.assertTrue((self.directory / 'install/include/dependency.h').is_file())
        self.assertTrue((self.directory / 'install/lib/pkgconfig/library.pc').is_file())
        self.assertTrue(source.dependency_cache_valid(self.directory,
                        json.loads((self.directory / 'dependency-receipt.json').read_text()), KEY))

    def test_complete_round_trip_reextracts_licenses_without_build_trees(self):
        archive = self.root / 'complete.tar.gz'
        manifest = transfer.pack('complete', REVISION, archive)
        self.assertNotIn('media/install/include/dependency.h', manifest['files'])
        shutil.rmtree(self.directory)
        transfer.restore('complete', REVISION, archive)
        self.assertEqual((self.directory / 'ffmpeg-source/NOTICE').read_bytes(), b'original upstream notice\n')
        self.assertEqual((self.directory / 'ffmpeg').stat().st_mode & 0o111,
                         manifest['files']['media/ffmpeg']['mode'] & 0o111)

    def test_changed_digest_missing_member_wrong_revision_and_recipe_reject_before_activation(self):
        archive = self.root / 'complete.tar.gz'
        transfer.pack('complete', REVISION, archive)
        (self.directory / 'keep').write_text('prior cache')
        def alter(member, data):
            return (member, b'changed') if member.name == 'media/ffmpeg' else (member, data)
        def omit(member, data):
            return (None, data) if member.name == 'media/ffmpeg' else (member, data)
        for edit in [alter, omit]:
            with self.assertRaises(ValueError):
                transfer.restore('complete', REVISION, self.repack(archive, edit))
            self.assertTrue((self.directory / 'keep').is_file())
        with self.assertRaises(ValueError):
            transfer.restore('complete', '4' * 40, archive)
        with patch.object(source, 'cache_identity', return_value=('linux_amd64', ['ffmpeg', 'ffprobe'], KEY,
                                                                'changed compiler', '3' * 64)):
            with self.assertRaises(ValueError):
                transfer.restore('complete', REVISION, archive)
        self.assertTrue((self.directory / 'keep').is_file())

    def test_traversal_nonregular_duplicates_and_undeclared_files_reject(self):
        archive = self.root / 'complete.tar.gz'
        transfer.pack('complete', REVISION, archive)
        def rename(member, data):
            if member.name == 'media/ffmpeg':
                member.name = '../outside'
            return member, data
        def duplicate(member, data):
            if member.name == 'media/ffprobe':
                member.name = 'media/ffmpeg'
            return member, data
        def symlink(member, data):
            if member.name == 'media/ffmpeg':
                member.type, member.linkname = tarfile.SYMTYPE, '/outside'
            return member, data
        def undeclared(member, data):
            if member.name == 'media/ffmpeg':
                member.name = 'media/extra'
            return member, data
        for edit in [rename, duplicate, symlink, undeclared]:
            with self.assertRaises(ValueError):
                transfer.restore('complete', REVISION, self.repack(archive, edit))

    def test_failed_final_activation_restores_the_prior_build(self):
        archive = self.root / 'complete.tar.gz'
        transfer.pack('complete', REVISION, archive)
        (self.directory / 'prior-cache-marker').write_bytes(b'keep prior build')
        original = transfer.verified_receipt
        calls = []
        def interrupted(directory, *args, **kwargs):
            calls.append(directory)
            if directory == self.directory:
                raise OSError('simulated final readback failure')
            return original(directory, *args, **kwargs)
        with patch.object(transfer, 'verified_receipt', side_effect=interrupted):
            with self.assertRaises(OSError):
                transfer.restore('complete', REVISION, archive)
        self.assertEqual((self.directory / 'prior-cache-marker').read_bytes(), b'keep prior build')
        self.assertEqual(len(calls), 2)

    def test_ready_probe_is_read_only_and_requires_the_entire_exact_closure(self):
        paths = [path for path in self.directory.rglob('*') if path.is_file()]
        original = {str(path): path.read_bytes() for path in paths}
        self.assertTrue(transfer.ready('complete', REVISION))
        self.assertTrue(transfer.ready('dependencies', REVISION))
        self.assertEqual({str(path): path.read_bytes() for path in paths}, original)
        (self.directory / 'ffmpeg').write_bytes(b'forged binary')
        self.assertFalse(transfer.ready('complete', REVISION))
        (self.directory / 'ffmpeg').write_bytes(b'ffmpeg')
        source_archive = self.root / 'build/media-source-pins/pin.tar.gz'
        source_archive.unlink()
        self.assertFalse(transfer.ready('complete', REVISION))
        with self.assertRaises(ValueError):
            transfer.ready('complete', '4' * 40)


if __name__ == '__main__':
    unittest.main()
