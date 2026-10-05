import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import subprocess
import sys
import time
import unittest
import zipfile

spec = importlib.util.spec_from_file_location('qualify', Path(__file__).resolve().parents[1] / 'scripts/qualify.py')
qualify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualify)

class QualificationArchiveTests(unittest.TestCase):
    def test_receipt_is_json_despite_native_loader_warnings(self):
        output = b'libEGL warning: no accelerated rendering\n{"native_webview":"passed","frontend_bridge_ipc":"passed","schema_version":"0.0.0"}\n'
        expected = {'native_webview': 'passed', 'frontend_bridge_ipc': 'passed', 'schema_version': '0.0.0'}
        with tempfile.TemporaryDirectory() as directory:
            receipt = Path(directory) / 'webview-receipt.json'
            qualify.write_receipt(receipt, output, expected)
            self.assertEqual(json.loads(receipt.read_text(encoding='utf-8')), expected)
            for invalid in [b'warning only\n', b'{"native_webview":"failed"}\n', output + output]:
                with self.assertRaises(ValueError):
                    qualify.write_receipt(receipt, invalid, expected)

    def test_timeout_terminates_descendant_writers(self):
        with tempfile.TemporaryDirectory() as directory:
            marker = Path(directory) / 'heartbeat'
            worker = "import pathlib,sys,time; p=pathlib.Path(sys.argv[1]); n=0\nwhile True:\n p.write_text(str(n)); n+=1; time.sleep(.02)"
            parent = "import subprocess,sys,time; subprocess.Popen([sys.executable,'-c',sys.argv[1],sys.argv[2]]); time.sleep(60)"
            with self.assertRaises(subprocess.TimeoutExpired):
                qualify.child([sys.executable, '-c', parent, worker, marker], timeout=2)
            before = marker.read_bytes()
            time.sleep(.12)
            self.assertEqual(marker.read_bytes(), before, 'descendant survived timeout')

    def test_rejects_windows_and_posix_path_escape(self):
        with tempfile.TemporaryDirectory() as directory:
            for name in ['../outside', '/absolute', 'C:/outside', '..\\outside']:
                with self.assertRaises(ValueError):
                    qualify.archive_path(Path(directory), name)

    def test_tar_aliases_materialize_inside_destination(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive = root / 'source.tar.gz'
            with tarfile.open(archive, 'w:gz') as target:
                item = tarfile.TarInfo('./lib.so.1'); item.size = 4
                target.addfile(item, io.BytesIO(b'test'))
                for name, link in [('./lib.so', 'lib.so.0'), ('./lib.so.0', 'lib.so.1')]:
                    item = tarfile.TarInfo(name); item.type = tarfile.SYMTYPE; item.linkname = link
                    target.addfile(item)
            qualify.extract(archive, root / 'output')
            self.assertEqual((root / 'output/lib.so').read_bytes(), b'test')
            self.assertFalse((root / 'output/lib.so').is_symlink())

    def test_archive_link_cannot_leave_destination(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); archive = root / 'source.tar.gz'
            with tarfile.open(archive, 'w:gz') as target:
                item = tarfile.TarInfo('lib.so'); item.type = tarfile.SYMTYPE; item.linkname = '../outside'
                target.addfile(item)
            with self.assertRaises(ValueError):
                qualify.extract(archive, root / 'output')

    def test_zip_parent_escape_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); archive = root / 'source.zip'
            with zipfile.ZipFile(archive, 'w') as target:
                target.writestr('../outside', b'test')
            with self.assertRaises(ValueError):
                qualify.extract(archive, root / 'output')

if __name__ == '__main__':
    unittest.main()
