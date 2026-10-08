# SPDX-License-Identifier: Apache-2.0
import importlib.util
import io
from pathlib import Path
import tarfile
import tempfile
import unittest
ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('windows_openssl', ROOT / 'scripts/windows_openssl.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
class OpenSSLArchiveTests(unittest.TestCase):
    def archive(self, entries):
        data = io.BytesIO()
        with tarfile.open(fileobj=data, mode='w') as archive:
            for name, kind in entries:
                info = tarfile.TarInfo(name)
                info.type = kind
                info.size = 3 if kind == tarfile.REGTYPE else 0
                archive.addfile(info, io.BytesIO(b'DLL') if info.size else None)
        data.seek(0)
        return tarfile.open(fileobj=data, mode='r')
    def test_only_declared_regular_companions_are_written(self):
        with tempfile.TemporaryDirectory() as temporary:
            destination = Path(temporary)
            found = set()
            with self.archive([('../outside', tarfile.REGTYPE), ('Library/bin/libssl-3-x64.dll', tarfile.REGTYPE)]) as source:
                module.extract_members(source, destination, found)
            self.assertEqual(found, {'libssl-3-x64.dll'})
            self.assertEqual(list(destination.iterdir()), [destination / 'libssl-3-x64.dll'])
    def test_declared_links_and_duplicates_fail(self):
        for entries in [
            [('Library/bin/libssl-3-x64.dll', tarfile.SYMTYPE)],
            [('Library/bin/libssl-3-x64.dll', tarfile.REGTYPE)] * 2]:
            with tempfile.TemporaryDirectory() as temporary:
                with self.archive(entries) as source:
                    with self.assertRaises(ValueError):
                        module.extract_members(source, Path(temporary), set())
