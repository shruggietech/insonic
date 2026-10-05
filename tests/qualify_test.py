import importlib.util
import io
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
