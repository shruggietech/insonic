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
    def test_virtual_display_wrapper_preserves_literal_arguments(self):
        args = ['/private program/desktop', '--help-qualification', 'quote\'"$(); text']
        command = qualify.xvfb_command(args)
        self.assertEqual(command[-len(args):], args)
        self.assertEqual(command[0], '/bin/sh')
        self.assertIn('/usr/bin/xvfb-run', command)

    @unittest.skipUnless(os.name == 'posix', 'POSIX descriptor inheritance requires a POSIX host')
    def test_virtual_display_restores_stderr_before_strict_receipt_admission(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            runner = root / 'controlled xvfb-run'
            runner.write_text('#!/bin/sh\n[ "$1" = "-a" ] || exit 98\nshift\nexec 3>/dev/null\n"$@" 2>&1\n', encoding='utf-8')
            runner.chmod(0o700)
            progress = {'kind': 'desktop-qualification-progress', 'mode': 'help',
                        'observation': 'native-receipt', 'stage': 'module', 'elapsed_ns': 1}
            receipt = {'offline_help_activation': 'passed'}
            program = ('import json,sys; '
                       'print(json.dumps(' + repr(progress) + '), file=sys.stderr); '
                       'print(json.dumps(' + repr(receipt) + ')); '
                       'assert sys.argv[1:] == ["literal $(); argument"]')
            command = qualify.xvfb_command([sys.executable, '-c', program, 'literal $(); argument'], runner=runner)
            with qualify.ProcessTree(command, cwd=root, env=os.environ.copy()) as tree:
                output, error = tree.process.communicate(timeout=qualify.timing.READINESS_SECONDS)
                self.assertEqual(tree.process.returncode, 0)
            self.assertEqual(json.loads(output), receipt)
            self.assertEqual(json.loads(error), progress)
            observation = {}
            qualify.record_progress(observation, error)
            self.assertEqual(observation['progress'], [progress])
            target = root / 'receipt.json'
            qualify.write_receipt(target, output, receipt)
            self.assertEqual(json.loads(target.read_text(encoding='utf-8')), receipt)
            with self.assertRaises(ValueError):
                qualify.write_receipt(target, output + error, receipt)

    def test_receipt_rejection_preserves_bounded_stream_identity_without_acceptance(self):
        from hashlib import sha256
        from unittest.mock import Mock
        target = Mock()
        for output, count in [(b'', 0), (b'{"passed":true}\n{"passed":true}\n', 2),
                              (b'{"passed":false}\n', 1), (b'controlled-warning' * 1000, 0)]:
            with self.subTest(size=len(output)), self.assertRaises(ValueError) as failure:
                qualify.write_receipt(target, output, {'passed': True})
            prefix = 'missing, ambiguous or failed qualification receipt: '
            self.assertTrue(str(failure.exception).startswith(prefix))
            details = json.loads(str(failure.exception)[len(prefix):])
            self.assertEqual(details['json_records'], count)
            self.assertEqual(details['stdout_bytes'], len(output))
            self.assertEqual(details['stdout_sha256'], sha256(output).hexdigest())
            self.assertEqual(details['stdout_tail'], output[-4096:].decode('utf-8', 'replace'))
            target.write_text.assert_not_called()

    def test_prepare_retries_transient_asset_failure_before_extracting_verified_bytes(self):
        payload = b'controlled pinned native archive'
        import hashlib
        import urllib.error
        pin = {'version': '1', 'assets': {'linux_amd64': ['fixture.zip', hashlib.sha256(payload).hexdigest()]}}
        failure = urllib.error.HTTPError('https://github.com/controlled/asset', 502, 'Bad Gateway', {}, None)
        with tempfile.TemporaryDirectory() as directory, \
                patch.object(qualify, 'BUILD', Path(directory)), \
                patch.object(qualify, 'LOCK', {'go': '1', 'wails': '2', 'ladybug': pin}), \
                patch.object(qualify.platform, 'system', return_value='Linux'), \
                patch.object(qualify.platform, 'machine', return_value='x86_64'), \
                patch('urllib.request.urlopen', side_effect=[failure, io.BytesIO(payload)]) as download, \
                patch.object(qualify.time, 'sleep'), \
                patch.object(qualify, 'extract', side_effect=RuntimeError('stop after verified extraction')) as extract:
            with self.assertRaisesRegex(RuntimeError, 'stop after verified extraction'):
                qualify.prepare()
            self.assertEqual(download.call_count, 2)
            self.assertEqual((Path(directory) / 'fixture.zip').read_bytes(), payload)
            extract.assert_called_once_with(Path(directory) / 'fixture.zip', Path(directory) / 'ladybug')
            self.assertFalse((Path(directory) / 'fixture.download').exists())


    def test_prepare_does_not_retry_permanent_failure_or_publish_bad_checksum(self):
        import hashlib
        import urllib.error
        payload = b'verified fixture'
        for mode in ['not-found', 'wrong-checksum', 'expired-budget']:
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                pin = {'version': '1', 'assets': {'linux_amd64': ['fixture.zip', hashlib.sha256(payload).hexdigest()]}}
                failure = urllib.error.HTTPError('https://github.com/controlled/asset', 404 if mode == 'not-found' else 502, 'controlled failure', {}, None)
                result = io.BytesIO(b'unverified bytes') if mode == 'wrong-checksum' else failure
                with patch.object(qualify, 'BUILD', Path(directory)), \
                        patch.object(qualify, 'LOCK', {'go': '1', 'wails': '2', 'ladybug': pin}), \
                        patch.object(qualify.platform, 'system', return_value='Linux'), \
                        patch.object(qualify.platform, 'machine', return_value='x86_64'), \
                        patch('urllib.request.urlopen', side_effect=[result]) as download, \
                        patch.object(qualify.time, 'sleep'), \
                        patch.object(qualify.time, 'monotonic', side_effect=[0, 0, 601] if mode == 'expired-budget' else [0] * 10), \
                        patch.object(qualify, 'extract') as extract:
                    expected = ValueError if mode == 'wrong-checksum' else RuntimeError
                    with self.assertRaises(expected) as caught:
                        qualify.prepare()
                    if mode == 'not-found':
                        self.assertIsInstance(caught.exception.__cause__, urllib.error.HTTPError)
                        self.assertEqual(caught.exception.__cause__.code, 404)
                    self.assertEqual(download.call_count, 1)
                    self.assertLessEqual(download.call_args.kwargs['timeout'], qualify.timing.REQUEST_SECONDS)
                    extract.assert_not_called()
                    self.assertFalse((Path(directory) / 'fixture.zip').exists())
                    self.assertFalse((Path(directory) / 'fixture.download').exists())

    def test_native_cli_builder_passes_the_selected_environment(self):
        selected = {'CC': 'selected', 'INSONIC_NATIVE_LIBRARY': 'isolated'}
        with patch.object(qualify, 'child') as build, patch('pe_resources.enable_utf8') as manifest:
            path = qualify.build_cli(True, selected)
        if os.name == 'nt': manifest.assert_called_once_with(path)
        else: manifest.assert_not_called()
        self.assertEqual(build.call_args.kwargs['env'], selected)
        self.assertIn('system_ladybug', build.call_args.args[0])
        self.assertEqual(path.name, 'insonic.exe' if os.name == 'nt' else 'insonic')

    def test_standalone_desktop_prepares_its_native_sibling_before_acceptance(self):
        selected_native_env = {'CC': 'selected'}
        for ci in ['false', 'true']:
            with self.subTest(ci=ci), patch.dict(os.environ, {'GITHUB_ACTIONS': ci}), \
                    patch.object(qualify, 'native_env', return_value=dict(selected_native_env)), \
                    patch('pe_resources.enable_utf8'), \
                    patch.object(qualify, 'build_cli', side_effect=RuntimeError('stop after native sibling')) as build, \
                    patch.object(qualify, 'child') as acceptance:
                with self.assertRaisesRegex(RuntimeError, 'native sibling'):
                    qualify.desktop()
                build.assert_called_once_with(True, selected_native_env)
                acceptance.assert_not_called()
        self.assertEqual(selected_native_env, {'CC': 'selected'})

    def test_native_acceptance_runs_every_package_after_separate_compilation(self):
        selected = {'CC': 'selected-compiler'}
        with tempfile.TemporaryDirectory() as directory, \
                patch.object(qualify, 'BUILD', Path(directory)), \
                patch.object(qualify, 'native_env', return_value=selected), \
                patch.object(qualify, 'cueson_probe', return_value={'state': 'passed'}), \
                patch.object(qualify, 'child', return_value=b'') as run:
            qualify.native()
            go = [call for call in run.call_args_list if call.args[0][0] == 'go']
            self.assertEqual(len(go), 2)
            compile_args, acceptance_args = [call.args[0] for call in go]
            self.assertIn('-run', compile_args)
            self.assertEqual(compile_args[compile_args.index('-run') + 1], '^$')
            self.assertNotIn('-run', acceptance_args)
            self.assertIn('-json', acceptance_args)
            expected = {'./internal/qualification', './internal/backendsetup', './internal/subtitles',
                        './internal/graph', './internal/app', './internal/backup'}
            for call in go:
                args = call.args[0]
                self.assertEqual({arg for arg in args if arg.startswith('./')}, expected)
                self.assertEqual(args[args.index('-tags') + 1], 'native_ladybug,system_ladybug')
                self.assertIs(call.kwargs['env'], selected)
                self.assertEqual(call.kwargs['timeout'], qualify.timing.GO_TEST_PROCESS_SECONDS)
            self.assertTrue((Path(directory) / 'native-receipt.json').is_file())

    def test_failed_native_compilation_cannot_publish_an_acceptance_receipt(self):
        def child(args, **kwargs):
            if args[0] == 'go':
                raise subprocess.TimeoutExpired(args, 180)
            return b''
        with tempfile.TemporaryDirectory() as directory, \
                patch.object(qualify, 'BUILD', Path(directory)), \
                patch.object(qualify, 'native_env', return_value={}), \
                patch.object(qualify, 'cueson_probe', return_value={'state': 'passed'}), \
                patch.object(qualify, 'child', side_effect=child) as run:
            with self.assertRaises(subprocess.TimeoutExpired):
                qualify.native()
            self.assertFalse((Path(directory) / 'native-receipt.json').exists())
            self.assertFalse(any('-json' in call.args[0] for call in run.call_args_list))

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

    def test_windows_startup_recovery_is_required_and_binding_selection_is_noninteractive(self):
        import re
        source = (qualify.ROOT / 'scripts/qualify.py').read_text(encoding='utf-8')
        selected = re.search(r"'-run', '(\^\(TestOwnedWindow[^']+)'", source).group(1)
        for name in ['TestOwnedWindowDestructionClosesControllerOnceOnItsUIThread',
                     'TestBootstrapOnlyEnvironmentVersionDoesNotEnableCDPOrGateNavigation',
                     'TestBrowserDiagnosticsEnvironmentHookKeepsNativeCreationOrder',
                     'TestBrowserDiagnosticsVersionOwnershipBoundsAndAllowlist',
                     'TestBrowserDiagnosticsDisabledAddsNoNativeCalls',
                     'TestStartupFailuresCompleteAndReleaseAfterCallbackUnwinds']:
            self.assertRegex(name, selected)
        self.assertIsNone(re.search(selected, 'TestCookieManager'))
        with patch.object(qualify.platform, 'system', return_value='Windows'):
            self.assertEqual(qualify.native_exit_contract()['startup_recovery'], 'passed')
        with patch.object(qualify.platform, 'system', return_value='Linux'):
            self.assertNotIn('startup_recovery', qualify.native_exit_contract())

    def test_maintained_source_metadata_is_exact_and_cannot_add_or_redirect_modules(self):
        qualify.validate_maintained_sources()
        for mode in ['checksum', 'missing', 'extra', 'redirect']:
            selected = json.loads(json.dumps(qualify.LOCK))
            sources = selected['maintained_sources']
            if mode == 'checksum': sources['github.com/wailsapp/wails/v2']['patched_sum'] = 'h1:' + 'A' * 43 + '='
            elif mode == 'missing': del sources['github.com/wailsapp/wails/v2']
            elif mode == 'extra': sources['example.test/unapproved'] = dict(sources['github.com/wailsapp/wails/v2'])
            else: sources['github.com/wailsapp/wails/v2']['manifest'] = '../unapproved.json'
            with self.subTest(mode=mode), self.assertRaisesRegex(ValueError, 'maintained source'):
                qualify.validate_maintained_sources(selected)

    def test_receipt_is_json_despite_native_loader_warnings(self):
        output = b'libEGL warning: no accelerated rendering\n{"native_webview":"passed","frontend_bridge_ipc":"passed","schema_version":"1.0.1"}\n'
        expected = {'native_webview': 'passed', 'frontend_bridge_ipc': 'passed', 'schema_version': '1.0.1'}
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
            parent = "import os,subprocess,sys,time; print('controlled checkpoint',flush=True); print('controlled diagnostic',file=sys.stderr,flush=True); subprocess.Popen([sys.executable,'-c',sys.argv[1],sys.argv[2]],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW if os.name=='nt' else 0); time.sleep(60)"
            stdout, stderr = io.BytesIO(), io.BytesIO()
            with patch.object(sys, 'stdout', io.TextIOWrapper(stdout, encoding='utf-8')), patch.object(sys, 'stderr', io.TextIOWrapper(stderr, encoding='utf-8')):
                with self.assertRaises(subprocess.TimeoutExpired):
                    qualify.child([sys.executable, '-c', parent, worker, marker], timeout=2)
                captured_stdout, captured_stderr = stdout.getvalue(), stderr.getvalue()
            self.assertIn(b'controlled checkpoint', captured_stdout)
            self.assertIn(b'controlled diagnostic', captured_stderr)
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
