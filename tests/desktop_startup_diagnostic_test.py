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
    def test_abi_overlay_is_exact_historical_source_without_editing_production(self):
        selected=Path('third_party/go-webview2/pkg/edge/ICoreWebView2Controller3.go')
        corrected=(ROOT/selected).read_bytes()
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);source=root/selected;source.parent.mkdir(parents=True);source.write_bytes(corrected)
            with patch.object(diagnostic,'ROOT',root):
                overlay=diagnostic.historical_abi_overlay(root/'control')
                replacement=json.loads(overlay.read_text())['Replace']
                self.assertEqual(set(replacement),{str(source.resolve())})
                historical=Path(replacement[str(source.resolve())]).read_bytes()
                self.assertEqual(hashlib.sha256(historical).hexdigest(),diagnostic.ABI_CONTROL_FILE_SHA256)
                self.assertEqual(source.read_bytes(),corrected)
                self.assertIn(b'uintptr(unsafe.Pointer(&value))',historical)
                source.write_bytes(corrected+b'\nchanged\n')
                with self.assertRaisesRegex(ValueError,'checksum'):diagnostic.historical_abi_overlay(root/'bad-control')

    def test_abi_comparison_bounds_each_arm_and_retains_expected_control_failure(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);payload=root/'payload';payload.mkdir();gui=payload/'insonic-desktop.exe';summary={'derived_gui_source_revision':'a'*40,'observations':[]}
            calls=[]
            def observe(image,cwd,location,env,evidence,arm,**options):
                calls.append((image,location,evidence,dict(options)))
                arm['observations'].append({'observed_pass':image==gui})
                arm['result']='cold-start-failure-observed' if image!=gui else 'cold-start-sampling-budget-exhausted-not-a-cause-proof'
                if image!=gui:raise RuntimeError('expected old ABI failure')
            package=MagicMock();package.sha.return_value='b'*64
            package.package_directories.side_effect=lambda root,*args:(root,root)
            with patch.object(diagnostic,'historical_abi_overlay',return_value=root/'overlay.json'),patch.object(diagnostic,'build_probe_gui',return_value='c'*64) as build,patch.object(diagnostic,'observe_cold_start',side_effect=observe):
                diagnostic.observe_abi_comparison(package,gui,payload,root,{},root/'evidence',summary,startup_hide=True)
            self.assertEqual(len(calls),2)
            self.assertEqual([arm['arm'] for arm in summary['arms']],['old-ABI-control','corrected'])
            self.assertNotEqual(calls[0][1],calls[1][1])
            for call in calls:self.assertEqual(call[3],{'startup_hide':True,'sample_seconds':150,'sample_limit':120})
            self.assertEqual(build.call_args.kwargs,{'overlay':root/'overlay.json'})
            self.assertFalse(summary['abi_control']['feature_qualification'])
            self.assertEqual(summary['result'],'bounded-ABI-comparison-not-a-cause-proof')

    def test_abi_comparison_corrected_failure_is_not_masked_by_the_control(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);payload=root/'payload';payload.mkdir();summary={'derived_gui_source_revision':'a'*40,'observations':[]}
            def fail(image,cwd,location,env,evidence,arm,**options):
                arm['result']='cold-start-failure-observed';raise RuntimeError('retained native failure')
            package=MagicMock();package.sha.return_value='b'*64
            package.package_directories.side_effect=lambda root,*args:(root,root)
            with patch.object(diagnostic,'historical_abi_overlay',return_value=root/'overlay.json'),patch.object(diagnostic,'build_probe_gui',return_value='c'*64),patch.object(diagnostic,'observe_cold_start',side_effect=fail):
                with self.assertRaisesRegex(RuntimeError,'corrected startup failed'):
                    diagnostic.observe_abi_comparison(package,payload/'gui.exe',payload,root,{},root/'evidence',summary,startup_hide=True)
            self.assertEqual(summary['result'],'corrected-startup-failure-observed')
            self.assertEqual(len(summary['arms']),2)

    def test_control_overlay_is_an_explicit_build_only_input(self):
        package=MagicMock();package.qualify.native_env.return_value={'PATH':'selected-path','CGO_ENABLED':'1'}
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);(root/'help').mkdir();node=root/'node.exe';node.write_bytes(b'fixture')
            with patch.object(diagnostic.shutil,'copytree'),patch.object(diagnostic.shutil,'which',return_value=str(node)),patch.object(diagnostic,'source_child') as child:
                diagnostic.build_probe_gui(package,root/'control.exe',root,overlay=root/'pinned-overlay.json')
            args,env=child.call_args.args
            self.assertEqual(args[1:4],['build','-overlay',root/'pinned-overlay.json'])
            self.assertEqual(env['GOFLAGS'],'')
            self.assertNotIn('GITHUB_TOKEN',env)

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


    def test_cold_start_probe_uses_fresh_profiles_and_stops_at_first_failure(self):
        calls=[]
        def child(args,cwd,env,timeout,**options):
            fixture=Path(args[-1]);calls.append(fixture)
            self.assertEqual(json.loads(fixture.read_text()),{'action':'close'})
            self.assertEqual(args[1],'--exit-qualification-child')
            self.assertEqual(timeout,18);self.assertEqual(options,{'require_release':True})
            return {'exit_code':0 if len(calls)==1 else 1,'timed_out':False,'truncated':False,'capture_incomplete':False,'owned_release':len(calls)==1,'stdout':json.dumps({'action':'close','native_exit':'passed','playback':'not-selected'}),'stderr':'native-startup-progress retained'}
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'capture',side_effect=child),patch('builtins.print'):
            root=Path(temporary);summary={'observations':[]}
            with self.assertRaisesRegex(RuntimeError,'cold startup'):
                diagnostic.observe_cold_start(root/'gui.exe',root,root,{},root/'evidence',summary)
            self.assertEqual(len(calls),2)
            self.assertNotEqual(calls[0].parent,calls[1].parent)
            self.assertEqual(summary['result'],'cold-start-failure-observed')
            self.assertIn('native-startup-progress',json.loads((root/'evidence/cold-001.json').read_text())['stderr'])

    def test_cold_start_probe_rejects_root_exit_without_natural_release_or_native_result(self):
        valid={'exit_code':0,'timed_out':False,'truncated':False,'capture_incomplete':False,'owned_release':True,'stdout':json.dumps({'action':'close','native_exit':'passed','playback':'not-selected'})}
        for update in [{'owned_release':False},{'stdout':'{}'},{'stdout':'[]'},{'stdout':'broken'},{'truncated':True},{'capture_incomplete':True}]:
            with self.subTest(update=update),tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'capture',return_value=dict(valid,**update)),patch('builtins.print'):
                root=Path(temporary);summary={'observations':[]}
                with self.assertRaisesRegex(RuntimeError,'cold startup'):
                    diagnostic.observe_cold_start(root/'gui.exe',root,root,{},root/'evidence',summary)
                self.assertEqual(len(summary['observations']),1)

    def test_cold_start_probe_sampling_is_bounded_and_never_claims_a_repair(self):
        valid={'exit_code':0,'timed_out':False,'truncated':False,'capture_incomplete':False,'owned_release':True,'stdout':json.dumps({'action':'close','native_exit':'passed','playback':'not-selected'})}
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'capture',return_value=valid) as child,patch.object(diagnostic.time,'monotonic',return_value=0),patch('builtins.print'):
            root=Path(temporary);summary={'observations':[]}
            diagnostic.observe_cold_start(root/'gui.exe',root,root,{},root/'evidence',summary)
            self.assertEqual(child.call_count,240)
            self.assertEqual(summary['result'],'cold-start-no-failure-observed-not-a-cause-proof')
            self.assertEqual(summary['sample_limit'],240)
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'capture') as child,patch.object(diagnostic.time,'monotonic',side_effect=[0,290]),patch('builtins.print'):
            root=Path(temporary);summary={'observations':[]}
            diagnostic.observe_cold_start(root/'gui.exe',root,root,{},root/'evidence',summary)
            child.assert_not_called()
            self.assertEqual(summary['result'],'cold-start-sampling-budget-exhausted-not-a-cause-proof')

    def test_explicit_direct_child_capture_observes_release_before_job_cleanup(self):
        import io
        from types import SimpleNamespace
        for released in [True,False]:
            calls=[]
            class Tree:
                job=1
                process=SimpleNamespace(stdout=io.BytesIO(),stderr=io.BytesIO(),returncode=0,wait=lambda **kwargs:None,poll=lambda:0)
                def __init__(self,*args,**kwargs):pass
                def __enter__(self):return self
                def __exit__(self,*args):calls.append('close')
                def wait_for_descendants(self,timeout):calls.append('wait');return released
                def kill(self):calls.append('kill')
            with patch.object(diagnostic,'ProcessTree',Tree),patch.object(diagnostic,'job_processes',return_value={'available':True,'assigned':0,'pids':[]}):
                result=diagnostic.capture(['gui.exe','--exit-qualification-child','fixture.json'],ROOT,{},18,require_release=True)
            self.assertEqual(result['owned_release'],released)
            self.assertEqual(result['timed_out'],not released)
            self.assertLess(calls.index('wait'),calls.index('close'))
            self.assertEqual('kill' in calls,not released)

    def test_release_timing_keeps_the_original_combined_budget(self):
        import io
        from types import SimpleNamespace
        for root_seconds in [5,17]:
            waits=[]
            class Tree:
                job=1
                process=SimpleNamespace(stdout=io.BytesIO(),stderr=io.BytesIO(),returncode=0,wait=lambda **kwargs:None,poll=lambda:0)
                def __init__(self,*args,**kwargs):pass
                def __enter__(self):return self
                def __exit__(self,*args):pass
                def wait_for_descendants(self,timeout):waits.append(timeout);return False
                def kill(self):pass
            with self.subTest(root_seconds=root_seconds),patch.object(diagnostic,'ProcessTree',Tree),patch.object(diagnostic,'job_processes',return_value={'available':True,'assigned':2,'pids':[11,12]}),patch.object(diagnostic.time,'monotonic',side_effect=[0,root_seconds,root_seconds,18,18]):
                result=diagnostic.capture(['gui.exe','--exit-qualification-child','fixture.json'],ROOT,{},18,require_release=True)
            self.assertEqual(waits,[18-root_seconds])
            self.assertEqual(result['timing'],{'root_exit_elapsed_seconds':root_seconds,'release_budget_seconds':18-root_seconds,'release_wait_elapsed_seconds':18-root_seconds})
            self.assertTrue(result['timed_out'])
            self.assertFalse(result['owned_release'])
            self.assertEqual(result['job_before_close']['assigned'],2)


    def test_cold_start_hide_counterfactual_is_explicit_in_launch_and_evidence(self):
        value={'exit_code':1,'timed_out':False,'truncated':False,'capture_incomplete':False,'owned_release':False,'stdout':'{}'}
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'capture',return_value=value) as child,patch('builtins.print'):
            root=Path(temporary);summary={'observations':[]}
            with self.assertRaisesRegex(RuntimeError,'cold startup'):
                diagnostic.observe_cold_start(root/'gui.exe',root,root,{},root/'evidence',summary,startup_hide=True)
            self.assertIs(child.call_args.kwargs['startup_hide'],True)
            self.assertIs(summary['startup_hide'],True)
            self.assertIs(json.loads((root/'evidence/cold-000.json').read_text())['startup_hide'],True)

if __name__=='__main__':unittest.main()
