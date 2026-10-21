# SPDX-License-Identifier: Apache-2.0
import importlib.util
import json
from pathlib import Path
import sys
import unittest

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'scripts'))
spec=importlib.util.spec_from_file_location('window_calibration',ROOT/'scripts/diagnose-window-calibration.py')
calibration=importlib.util.module_from_spec(spec)
spec.loader.exec_module(calibration)

class WindowCalibrationTests(unittest.TestCase):
    def passing(self, seconds):
        error=[]
        for workspace in (False,True):
            for action in ('menu','close','alt-f4'):
                for _ in range(2):
                    records=[{'kind':'native-ui-join','result':'completed','returned_ns':2_500_000_000,'budget_ns':seconds*1_000_000_000} for _ in range(2)]
                    records.append({'kind':'native-desktop-observation','stage':'before-native-input'})
                    records.append({'kind':'native-window-environment','windows':[{'environment':{'created':True,'version_hresult':0,'actual_runtime_version':'152.0.4191.66'}}]})
                    error.append(json.dumps({'kind':'native-exit-owned-lifecycle','action':action,'workspace':workspace,'exit_code':0,'truncated':False,'descendants':{'released':True,'remaining':0},'stderr':'\n'.join(map(json.dumps,records))}))
        return json.dumps({'native_exit':'passed','window_join_seconds':seconds}).encode(), '\n'.join(error).encode()

    def test_failed_baseline_is_retained_and_comparison_runs_once_in_order(self):
        calls=[]
        def arm(seconds,label):
            calls.append((seconds,label))
            return {'native_passed':seconds==10,'allowance_seconds':seconds}
        result=calibration.compare(arm)
        self.assertEqual(calls,[(2,'baseline'),(10,'comparison')])
        self.assertFalse(result[0]['native_passed'])
        self.assertTrue(result[1]['native_passed'])

    def test_complete_delayed_joins_and_natural_release_are_observed(self):
        output,error=self.passing(10)
        result=calibration.parse_observations(output,b'null\n[]\n'+error,0,10)
        self.assertTrue(result['native_passed'])
        self.assertEqual(result['max_join_completion_seconds'],2.5)
        self.assertEqual(len(result['environments']),12)
        self.assertFalse(calibration.parse_observations(output,error,1,10)['native_passed'])
        self.assertFalse(calibration.parse_observations(output,error,0,10,True)['native_passed'])

    def test_success_marker_cannot_replace_missing_cases_release_or_completion(self):
        output,error=self.passing(10)
        for changed in [b'\n'.join(error.splitlines()[:-1]),error.replace(b'"remaining": 0',b'"remaining": 1',1),error.replace(b'completed',b'failed',1)]:
            with self.assertRaises(ValueError):
                calibration.parse_observations(output,changed,0,10)

    def test_unknown_runtime_observation_is_not_substituted_by_available_lookup(self):
        output,error=self.passing(10)
        error=error.replace(b'152.0.4191.66',b'')
        result=calibration.parse_observations(output,error,0,10)
        self.assertTrue(result['native_passed'])
        self.assertEqual(result['environments'][0]['windows'][0]['environment']['actual_runtime_version'],'')


    def test_environment_delta_preserves_profile_and_isolates_each_cohort(self):
        from unittest.mock import patch
        with patch.dict(calibration.os.environ, {'APPDATA':'profile-config','USERPROFILE':'profile-home','SystemRoot':'windows','PATH':'ambient'}, clear=True):
            delta={'PATH':'package-system-path'}
            env=calibration.cohort_environment(delta,Path('owned-temp'),10)
            self.assertEqual(env['APPDATA'],'profile-config')
            self.assertEqual(env['USERPROFILE'],'profile-home')
            self.assertEqual({key.upper():value for key,value in env.items()}['SYSTEMROOT'],'windows')
            self.assertEqual(env['PATH'],'package-system-path')
            self.assertEqual(env['TEMP'],str(Path('owned-temp')))
            self.assertEqual(env['TMP'],str(Path('owned-temp')))
            self.assertEqual(env['INSONIC_NATIVE_WINDOW_JOIN_SECONDS'],'10')
            self.assertEqual(delta,{'PATH':'package-system-path'})

if __name__=='__main__':unittest.main()
