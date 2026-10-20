# SPDX-License-Identifier: Apache-2.0
import importlib.util
import os
import io
import tempfile
from types import SimpleNamespace
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('startup_qualify', Path(__file__).resolve().parents[1] / 'scripts/qualify.py')
qualify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualify)
package_spec = importlib.util.spec_from_file_location('startup_package', Path(__file__).resolve().parents[1] / 'scripts/package-desktop.py')
package = importlib.util.module_from_spec(package_spec)
package_spec.loader.exec_module(package)


class StartupDiagnosticQualificationTests(unittest.TestCase):
    def assert_probe_environment(self, call, expected):
        selected = dict(call.kwargs['env'])
        profile = selected.pop('INSONIC_NATIVE_WEBVIEW_PROFILE', None)
        self.assertEqual(selected, expected)
        if os.name == 'nt' and any(flag in call.args[0] for flag in ('--help-qualification', '--webview-qualification')):
            self.assertTrue(Path(profile).is_absolute())
            self.assertFalse(Path(profile).parent.exists(), 'successful owned fixture was not removed')
        else:
            self.assertIsNone(profile)

    def test_packaged_gui_elects_existing_diagnostic_only_in_its_runtime_copy(self):
        for ci in ['true', 'false']:
            original = {'PATH': 'controlled-system-tools'}
            with self.subTest(ci=ci), patch.dict(os.environ, {'GITHUB_ACTIONS': ci}), \
                    patch.object(package, 'child', return_value=b'actual receipt bytes') as child, \
                    patch.object(package.qualify, 'write_receipt') as receipt:
                package.qualify_package_gui(Path('fixture-desktop'), Path('fixture-directory'), original)
            self.assertEqual(original, {'PATH': 'controlled-system-tools'})
            self.assertEqual(len(child.call_args_list), 4)
            for call in child.call_args_list:
                expected = dict(original)
                if ci == 'true': expected['INSONIC_PROCESS_STARTUP_DIAGNOSTICS'] = '1'
                self.assert_probe_environment(call, expected)
                self.assertIsNot(call.kwargs['env'], original)
                self.assertIs(call.kwargs['allow_detached'], True)
            self.assertEqual([call.kwargs['timeout'] for call in child.call_args_list], [30, 35, 90, 100])
            self.assertEqual(receipt.call_count, 4)
            self.assertIn('--exit-qualification', child.call_args_list[-1].args[0])
            required = receipt.call_args_list[-1].args[2]
            self.assertEqual(required['runtime_work_survival'], 'passed')
            self.assertEqual(required['playback_cleanup'], 'passed')
            self.assertEqual(required['native_exit'], 'passed')
            self.assertTrue(all(call.args[1] == b'actual receipt bytes' for call in receipt.call_args_list))

    def test_windows_exit_proof_requires_observed_browser_release(self):
        for platform_name in ('Windows','Linux'):
            with self.subTest(platform=platform_name), patch.object(package.platform,'system',return_value=platform_name), \
                    patch.object(package,'child',return_value=b'actual receipt bytes'), \
                    patch.object(package.qualify,'write_receipt') as receipt:
                package.qualify_package_gui(Path('fixture-desktop'),Path('fixture-directory'),{})
                required=receipt.call_args_list[-1].args[2]
                if platform_name=='Windows': self.assertEqual(required['owned_browser_release'],'passed')
                else: self.assertNotIn('owned_browser_release',required)

    def test_native_group_keeps_all_packages_and_default_deadline_with_streamed_events(self):
        with tempfile.TemporaryDirectory() as temporary, \
                patch.object(qualify, 'BUILD', Path(temporary)), \
                patch.object(qualify, 'native_env', return_value={'CGO_ENABLED': '1'}), \
                patch.object(qualify, 'cueson_probe', return_value='actual cueson identity'), \
                patch.object(qualify, 'child', return_value=b'') as child, \
                patch.object(qualify.sys, 'stdout', SimpleNamespace(buffer=io.BytesIO())):
            qualify.native()
            call = child.call_args_list[-1]
            self.assertEqual(call.args[0], ['go', 'test', '-json', '-v', '-tags', 'native_ladybug,system_ladybug',
                                           './internal/qualification', './internal/backendsetup', './internal/subtitles',
                                           './internal/graph', './internal/app', './internal/backup'])
            self.assertEqual(call.kwargs, {'env': {'CGO_ENABLED': '1'}})
        self.assertEqual(qualify.child.__kwdefaults__['timeout'], 180)

    def test_only_github_ci_desktop_children_elect_sanitized_startup_evidence(self):
        for ci in ['true', 'false']:
            with self.subTest(ci=ci), patch.dict(os.environ, {'GITHUB_ACTIONS': ci}), \
                    patch.object(qualify, 'native_env', return_value={}), \
                    patch.object(qualify, 'build_cli', return_value=Path('fixture-cli')) as build, \
                    patch.object(qualify, 'child', return_value=b'') as child, \
                    patch.object(qualify, 'write_receipt') as receipt, \
                    patch.object(qualify.platform, 'system', return_value='Windows'):
                qualify.desktop()
                self.assertEqual(receipt.call_args_list[-1].args[2]['owned_browser_release'], 'passed')
                build.assert_called_once_with(True, {})
                self.assertEqual(len(child.call_args_list), 7)
                self.assertEqual(child.call_args_list[0].args[0], ['go', 'test', '-timeout=15s', '-run', '^TestOwnedWindowDestructionClosesControllerOnceOnItsUIThread$', 'github.com/wailsapp/go-webview2/pkg/edge'])
                for call in child.call_args_list[:3]:
                    self.assertEqual(call.kwargs['env'], {})
                    self.assertNotIn('allow_detached', call.kwargs)
                for call in child.call_args_list[3:]:
                    self.assert_probe_environment(call, {'INSONIC_PROCESS_STARTUP_DIAGNOSTICS': '1'} if ci == 'true' else {})
                    self.assertIs(call.kwargs['allow_detached'], True)
                self.assertIsNot(child.call_args_list[0].kwargs['env'], child.call_args_list[3].kwargs['env'])


if __name__ == '__main__':
    unittest.main()
