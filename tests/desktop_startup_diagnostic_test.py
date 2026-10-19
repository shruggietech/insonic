# SPDX-License-Identifier: Apache-2.0
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('desktop_startup_diagnostic',ROOT/'scripts/diagnose-desktop-startup.py')
diagnostic=importlib.util.module_from_spec(spec)
spec.loader.exec_module(diagnostic)

class DesktopStartupDiagnosticTests(unittest.TestCase):
    def test_requires_the_exact_incident_payload(self):
        receipt={'revision':diagnostic.INCIDENT_REVISION,'platform':'windows_amd64','variant':'desktop','source_dirty':False,'archive':'fixture.zip'}
        with patch.object(diagnostic.pipeline,'read_inputs',return_value=({},receipt,{'revision':diagnostic.INCIDENT_REVISION})):
            self.assertEqual(diagnostic.incident_archive()[1],receipt)
        for field,value in [('revision','0'*40),('platform','linux_amd64'),('variant','cli'),('source_dirty',True)]:
            with self.subTest(field=field),patch.object(diagnostic.pipeline,'read_inputs',return_value=({},dict(receipt,**{field:value}),{'revision':diagnostic.INCIDENT_REVISION})):
                with self.assertRaises(ValueError):diagnostic.incident_archive()

    def test_child_failure_retains_both_streams_and_native_error(self):
        with tempfile.TemporaryDirectory() as temporary:
            observation=diagnostic.capture([sys.executable,'-c',"import sys;print('[WebView2 Error] synthetic 8007139f');print('synthetic stack',file=sys.stderr);sys.exit(23)"],Path(temporary),dict(os.environ),5)
        self.assertEqual(observation['exit_code'],23)
        self.assertIn('synthetic 8007139f',observation['stdout'])
        self.assertIn('synthetic stack',observation['stderr'])
        self.assertFalse(observation['timed_out'])
        self.assertFalse(observation['feature_qualification'])

    def test_timeout_is_failure_and_streams_are_bounded(self):
        with tempfile.TemporaryDirectory() as temporary:
            observation=diagnostic.capture([sys.executable,'-c',"import time;print('before timeout',flush=True);time.sleep(30)"],Path(temporary),dict(os.environ),1)
        self.assertTrue(observation['timed_out'])
        self.assertNotEqual(observation['exit_code'],0)
        self.assertIn('before timeout',observation['stdout'])
        self.assertLessEqual(len(observation['stdout'].encode()),65536)

    def test_refuses_owner_host_before_any_payload_execution(self):
        with patch.dict(os.environ,{},clear=True),patch.object(diagnostic,'incident_archive') as source:
            with self.assertRaises((ValueError,RuntimeError)):diagnostic.run()
            source.assert_not_called()

if __name__=='__main__':unittest.main()
