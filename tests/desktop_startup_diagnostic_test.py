# SPDX-License-Identifier: Apache-2.0
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch, MagicMock

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

    def test_output_flood_is_drained_and_retention_is_bounded_before_exit(self):
        result=diagnostic.capture([sys.executable,'-c',"import os;os.write(1,b'x'*4000000);os.write(2,b'y'*4000000)"],ROOT,dict(os.environ),5)
        self.assertEqual(result['exit_code'],0)
        self.assertTrue(result['truncated'])
        self.assertFalse(result['capture_incomplete'])
        for name in ['stdout','stderr']:
            self.assertEqual(len(result[name]),diagnostic.LIMIT)
            self.assertEqual(result[name+'_bytes'],4000000)
        stream=diagnostic.BoundedStream()
        class Flood:
            def read(self,count):
                self.count=getattr(self,'count',0)+1
                if self.count==100:return b''
                return b'x'*count
            def close(self):pass
        stream.drain(Flood())
        self.assertEqual(len(stream.data),diagnostic.LIMIT)
        self.assertEqual(stream.total,99*16384)

    def test_derived_gui_never_overwrites_the_historical_executable(self):
        package=MagicMock();package.package_directories.side_effect=lambda root,*args:(root,root)
        def build(package,gui,root):
            gui.write_bytes(b'derived GUI');return hashlib.sha256(gui.read_bytes()).hexdigest()
        with tempfile.TemporaryDirectory() as temporary:
            location=Path(temporary);original=location/'original';original.mkdir()
            (original/'insonic-desktop.exe').write_bytes(b'incident GUI')
            (original/'insonic.exe').write_bytes(b'incident CLI')
            with patch.object(diagnostic,'build_probe_gui',side_effect=build):
                gui,root,digest=diagnostic.derive_gui(package,original,location)
            self.assertEqual((original/'insonic-desktop.exe').read_bytes(),b'incident GUI')
            self.assertEqual(gui.read_bytes(),b'derived GUI')
            self.assertEqual((root/'insonic.exe').read_bytes(),b'incident CLI')
            self.assertNotEqual(gui,original/'insonic-desktop.exe')

    def test_source_observation_runs_both_distinct_production_parents(self):
        result={'exit_code':0,'timed_out':False,'truncated':False}
        summary={'observations':[]}
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'observe_predecessors') as preceding,patch.object(diagnostic,'capture',side_effect=lambda *args:dict(result)) as capture,patch('builtins.print'):
            root=Path(temporary);old=root/'old.exe';new=root/'new.exe'
            diagnostic.observe_native(new,root,root,{},root/'evidence',summary,True,(old,root/'old'))
        self.assertEqual(capture.call_args_list[0].args[0],[old,'--exit-qualification'])
        self.assertEqual(capture.call_args_list[1].args[0],[new,'--exit-qualification'])
        self.assertEqual(len(capture.call_args_list),13)
        self.assertEqual([call.args[-1] for call in preceding.call_args_list],['incident','source'])
        self.assertEqual(summary['observations'][0]['file'],'incident-parent.json')
        self.assertEqual(summary['observations'][1]['file'],'source-parent.json')

    def test_historical_runtime_failure_remains_evidence_and_current_probes_still_run(self):
        result={'exit_code':0,'timed_out':False,'truncated':False}
        summary={'observations':[]}
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'observe_predecessors',side_effect=[diagnostic.PredecessorFailure('baseline controller failure'),None]) as preceding,patch.object(diagnostic,'capture',side_effect=lambda *args:dict(result)) as capture,patch('builtins.print'):
            root=Path(temporary)
            diagnostic.observe_native(root/'new.exe',root,root,{},root/'evidence',summary,True,(root/'old.exe',root/'old'))
        self.assertEqual(summary['historical_predecessor'],'failed-baseline-evidence-retained')
        self.assertEqual(len(capture.call_args_list),13)
        self.assertEqual(preceding.call_args_list[-1].args[-1],'source')

    def test_current_predecessor_failure_retains_exit_observations_and_still_fails(self):
        result={'exit_code':0,'timed_out':False,'truncated':False}
        summary={'observations':[]}
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'observe_predecessors',side_effect=diagnostic.PredecessorFailure('current controller failure')),patch.object(diagnostic,'capture',side_effect=lambda *args:dict(result)) as capture,patch('builtins.print'):
            root=Path(temporary)
            with self.assertRaisesRegex(diagnostic.PredecessorFailure,'current controller'):
                diagnostic.observe_native(root/'new.exe',root,root,{},root/'evidence',summary,True)
            self.assertEqual(capture.call_count,12,'a predecessor failure suppressed the actual Exit diagnosis')
            self.assertTrue((root/'evidence/source-parent.json').is_file())
            self.assertEqual(summary['source_predecessor'],'failed-evidence-retained')
            self.assertEqual(summary['result'],'predecessor-failure-observed')

    def test_exit_failure_keeps_the_failed_predecessor_as_its_cause(self):
        predecessor=diagnostic.PredecessorFailure('current controller failure')
        result={'exit_code':1,'timed_out':True,'truncated':False,'stderr':'native-exit-stage=watchdog-timeout'}
        summary={'observations':[]}
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'observe_predecessors',side_effect=predecessor),patch.object(diagnostic,'capture',side_effect=lambda *args:dict(result)),patch('builtins.print'):
            root=Path(temporary)
            with self.assertRaisesRegex(RuntimeError,'source-built native parent failed') as caught:
                diagnostic.observe_native(root/'new.exe',root,root,{},root/'evidence',summary,True)
            self.assertIs(caught.exception.__cause__,predecessor)
            self.assertEqual(json.loads((root/'evidence/source-parent.json').read_text())['stderr'],result['stderr'])
            self.assertTrue((root/'evidence/parent-00.json').is_file())

    def test_historical_source_authority_must_be_present_in_owned_checkout(self):
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'ROOT',Path(temporary)):
            with self.assertRaisesRegex(ValueError,'incident source'):diagnostic.historical_source()
            source=Path(temporary)/'build/incident-source'
            (source/'internal/qualification').mkdir(parents=True)
            (source/'internal/qualification/dependencies.json').write_text('{}')
            with patch.object(diagnostic,'source_child',side_effect=[{'stdout':diagnostic.INCIDENT_REVISION},{'stdout':''}]):
                self.assertEqual(diagnostic.historical_source(),source)
            for outputs in [[{'stdout':'0'*40},{'stdout':''}],[{'stdout':diagnostic.INCIDENT_REVISION},{'stdout':' M scripts/processing-worker.py'}]]:
                with patch.object(diagnostic,'source_child',side_effect=outputs):
                    with self.assertRaisesRegex(ValueError,'revision or tracked bytes'):diagnostic.historical_source()

    def test_source_compiler_receives_selected_inputs_without_ci_credentials(self):
        package=MagicMock();package.qualify.native_env.return_value={'PATH':'selected-path','CGO_ENABLED':'1'}
        package.sha.return_value='a'*64
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);(root/'help').mkdir();node=root/'node.exe';node.write_bytes(b'fixture')
            with patch.dict(os.environ,{'GITHUB_TOKEN':'protected-fixture-token','GOFLAGS':'unselected-override'}),patch.object(diagnostic.shutil,'copytree'),patch.object(diagnostic.shutil,'which',return_value=str(node)),patch.object(diagnostic,'source_child') as build_child:
                self.assertEqual(diagnostic.build_probe_gui(package,root/'insonic-desktop.exe',root),'a'*64)
        for call in build_child.call_args_list:
            env=call.args[1]
            self.assertNotIn('GITHUB_TOKEN',env)
            self.assertEqual(env['GOFLAGS'],'')
            self.assertEqual(env['GOTOOLCHAIN'],'local')
            self.assertEqual(env['CGO_ENABLED'],'1')
        self.assertEqual(build_child.call_args_list[-1].args[0][1:5],['build','-tags','desktop,production,system_ladybug','-ldflags'])

    def test_load_compiles_current_cli_with_selected_credential_free_environment(self):
        package=MagicMock();package.qualify.native_env.return_value={'PATH':'selected-path','CGO_ENABLED':'1'};package.sha.return_value='b'*64
        with tempfile.TemporaryDirectory() as temporary,patch.dict(os.environ,{'GITHUB_TOKEN':'protected-fixture-token','GOFLAGS':'unselected-override'}),patch.object(diagnostic,'source_child') as build:
            cli=Path(temporary)/'insonic.exe'
            self.assertEqual(diagnostic.build_probe_cli(package,cli),'b'*64)
        self.assertEqual(build.call_args.args[0],['go','build','-tags','system_ladybug','-o',cli,'./cmd/insonic'])
        self.assertNotIn('GITHUB_TOKEN',build.call_args.args[1]);self.assertEqual(build.call_args.args[1]['GOFLAGS'],'')

    def test_source_child_uses_exact_environment_without_parent_credentials(self):
        selected={name:value for name,value in os.environ.items() if name.upper() in {'PATH','SYSTEMROOT','WINDIR','TEMP','TMP'}}
        with patch.dict(os.environ,{'GITHUB_TOKEN':'protected-fixture-token','GOFLAGS':'unselected-override'}):
            result=diagnostic.source_child([sys.executable,'-c',"import json,os;print(json.dumps({name:os.getenv(name) for name in ['GITHUB_TOKEN','GOFLAGS']}))"],selected)
        self.assertEqual(json.loads(result['stdout']),{'GITHUB_TOKEN':None,'GOFLAGS':None})

    def test_all_owned_cli_images_share_one_idle_lease(self):
        package=MagicMock();old=Path('old.exe');new=Path('new.exe')
        with patch.object(diagnostic.time,'monotonic',return_value=42):
            diagnostic.wait_owned_clis(package,[old,new,old])
        self.assertEqual(package.wait_for_detached_idle.call_args_list,[unittest.mock.call(42,old),unittest.mock.call(42,new)])

    def test_no_load_native_failure_still_checks_owned_release(self):
        idle=MagicMock(side_effect=RuntimeError('cleanup failure'))
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaisesRegex(RuntimeError,'native failure'):
                diagnostic.observe_with_load(MagicMock(side_effect=RuntimeError('native failure')),None,idle,Path(temporary))
        idle.assert_called_once()

    def test_concurrent_load_joins_and_idles_without_masking_native_failure(self):
        future=MagicMock();future.result.side_effect=RuntimeError('CLI failure');idle=MagicMock()
        native=MagicMock(side_effect=RuntimeError('native failure'))
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaisesRegex(RuntimeError,'native failure'):
                diagnostic.observe_with_load(native,future,idle,Path(temporary))
            self.assertEqual(json.loads((Path(temporary)/'concurrent-cli.json').read_text())['error'],'CLI failure')
        future.result.assert_called_once();idle.assert_called_once()

    def test_concurrent_load_failure_is_not_a_success(self):
        future=MagicMock();future.result.side_effect=RuntimeError('CLI failure');idle=MagicMock()
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaisesRegex(RuntimeError,'CLI failure'):
                diagnostic.observe_with_load(MagicMock(),future,idle,Path(temporary))
        idle.assert_called_once()

    def test_refuses_owner_host_before_any_payload_execution(self):
        with patch.dict(os.environ,{},clear=True),patch.object(diagnostic,'incident_archive') as source:
            with self.assertRaises((ValueError,RuntimeError)):diagnostic.run()
            source.assert_not_called()

if __name__=='__main__':unittest.main()
