# SPDX-License-Identifier: Apache-2.0
import importlib.util
import os
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('startup_qualify', Path(__file__).resolve().parents[1] / 'scripts/qualify.py')
qualify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualify)


class StartupDiagnosticQualificationTests(unittest.TestCase):
    def test_only_github_ci_desktop_children_elect_sanitized_startup_evidence(self):
        for ci in ['true', 'false']:
            with self.subTest(ci=ci), patch.dict(os.environ, {'GITHUB_ACTIONS': ci}), \
                    patch.object(qualify, 'native_env', return_value={}), \
                    patch.object(qualify, 'build_cli', return_value=Path('fixture-cli')), \
                    patch.object(qualify, 'child', return_value=b'') as child, \
                    patch.object(qualify, 'write_receipt'):
                qualify.desktop()
                self.assertEqual(len(child.call_args_list), 4)
                for call in child.call_args_list:
                    self.assertEqual(call.kwargs['env'].get('INSONIC_PROCESS_STARTUP_DIAGNOSTICS'),
                                     '1' if ci == 'true' else None)


if __name__ == '__main__':
    unittest.main()
