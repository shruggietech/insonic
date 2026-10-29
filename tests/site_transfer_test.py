# SPDX-License-Identifier: Apache-2.0
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('site_transfer', ROOT / 'scripts/site-transfer.py')
transfer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transfer)
REV = 'a' * 40


class SiteTransferTest(unittest.TestCase):
    def fixture(self, directory):
        for name in ('documentation', 'offline-help'):
            files = {'index.html': b'complete export',
                     'documentation-build.json': json.dumps({'version': '1.0.3', 'revision': REV, 'source_dirty': False, 'documentation_target': 'snapshot'}).encode(),
                     'release-documentation.json': json.dumps({'version': '1.0.3', 'revision': REV, 'kind': name}).encode()}
            with tarfile.open(directory / (name + '.tar.gz'), 'w:gz') as archive:
                for path, data in files.items():
                    item = tarfile.TarInfo(path); item.size = len(data)
                    archive.addfile(item, io.BytesIO(data))
        return transfer.pack(directory, REV, '1.0.3', '123', '2')

    def test_roundtrip_restores_both_exact_exports(self):
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary); data = base / 'data'; data.mkdir()
            self.fixture(data)
            transfer.restore(data, base / 'site', REV, '1.0.3', '123', '2')
            for name in ('out', 'offline'):
                self.assertEqual((base / 'site' / name / 'index.html').read_bytes(), b'complete export')

    def test_stale_attempt_source_and_archive_fail_before_restore(self):
        for field, wrong in [('run_attempt', '1'), ('revision', 'b' * 40), ('run_id', '999')]:
            with self.subTest(field=field), tempfile.TemporaryDirectory() as temporary:
                base = Path(temporary); data = base / 'data'; data.mkdir(); receipt = self.fixture(data)
                receipt[field] = wrong
                (data / 'transfer.json').write_text(json.dumps(receipt), encoding='utf-8')
                with self.assertRaises(ValueError): transfer.restore(data, base / 'site', REV, '1.0.3', '123', '2')
                self.assertFalse((base / 'site').exists())
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary); data = base / 'data'; data.mkdir(); self.fixture(data)
            with (data / 'offline-help.tar.gz').open('ab') as stream: stream.write(b'changed')
            with self.assertRaises(ValueError): transfer.restore(data, base / 'site', REV, '1.0.3', '123', '2')
            self.assertFalse((base / 'site').exists())

    def test_existing_output_is_never_merged_or_deleted(self):
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary); data = base / 'data'; data.mkdir(); self.fixture(data)
            (base / 'site/offline').mkdir(parents=True); keep = base / 'site/offline/keep'; keep.write_bytes(b'owner')
            with self.assertRaises(ValueError): transfer.restore(data, base / 'site', REV, '1.0.3', '123', '2')
            self.assertEqual(keep.read_bytes(), b'owner')


if __name__ == '__main__': unittest.main()
