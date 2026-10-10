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
import os
from unittest.mock import patch
import zipfile

spec = importlib.util.spec_from_file_location('qualify', Path(__file__).resolve().parents[1] / 'scripts/qualify.py')
qualify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualify)

class QualificationArchiveTests(unittest.TestCase):
    def test_generic_cli_lane_does_not_require_native_assets(self):
        with patch.object(qualify, 'native_env', side_effect=AssertionError('generic lane must not require native assets')), patch.object(qualify, 'qualify_cli') as run:
            qualify.cli()
            run.assert_called_once_with(False)

    def test_native_cli_environment_is_selected_and_restored_after_failure(self):
        original = os.environ.copy()
        def run(native):
            self.assertTrue(native)
            self.assertEqual(os.environ['CC'], 'selected-compiler')
            self.assertEqual(os.environ['INSONIC_NATIVE_LIBRARY'], 'selected-library')
            raise ValueError('controlled failure')
        with patch.object(qualify, 'native_env', return_value={'CC': 'selected-compiler', 'INSONIC_NATIVE_LIBRARY': 'selected-library'}), patch.object(qualify, 'qualify_cli', side_effect=run):
            with self.assertRaisesRegex(ValueError, 'controlled failure'): qualify.cli(native=True)
        self.assertEqual(dict(os.environ), original)

    def test_cli_build_uses_the_packaged_native_tag_only_in_native_lanes(self):
        for native in (False, True):
            with patch.object(qualify, 'child', side_effect=RuntimeError('stop before workspace')) as build:
                with self.assertRaisesRegex(RuntimeError, 'stop before workspace'): qualify.qualify_cli(native)
                self.assertEqual('system_ladybug' in build.call_args.args[0], native)

    def test_native_cli_cannot_qualify_a_fallback_or_other_graph(self):
        qualify.require_native_cli_graph({'state': 'available', 'adapter_id': 'ladybugdb'})
        qualify.require_native_cli_graph({'result': {'state': 'available', 'adapter_id': 'ladybugdb'}})
        for value in ({}, {'state': 'unavailable', 'adapter_id': 'ladybugdb'}, {'state': 'available', 'adapter_id': 'arcadedb'}):
            with self.assertRaisesRegex(ValueError, 'actual LadybugDB'): qualify.require_native_cli_graph(value)

    def test_qualification_pins_match_actual_modules(self):
        source = (qualify.ROOT / 'go.mod').read_text(encoding='utf-8')
        qualify.validate_pins(source)
        with self.assertRaises(ValueError):
            qualify.validate_pins(source.replace('v' + qualify.LOCK['wails'], 'v0.0.1'))

    def test_receipt_is_json_despite_native_loader_warnings(self):
        output = b'libEGL warning: no accelerated rendering\n{"native_webview":"passed","frontend_bridge_ipc":"passed","schema_version":"1.0.0"}\n'
        expected = {'native_webview': 'passed', 'frontend_bridge_ipc': 'passed', 'schema_version': '1.0.0'}
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
