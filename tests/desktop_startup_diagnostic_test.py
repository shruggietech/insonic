# SPDX-License-Identifier: Apache-2.0
import hashlib
import importlib.util
import json
import os
from pathlib import Path
from io import BytesIO
import sys
import tempfile
import unittest
from unittest.mock import patch, MagicMock
from threading import Event

ROOT=Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('desktop_startup_diagnostic',ROOT/'scripts/diagnose-desktop-startup.py')
diagnostic=importlib.util.module_from_spec(spec)
spec.loader.exec_module(diagnostic)

class DesktopStartupDiagnosticTests(unittest.TestCase):
    def test_normal_control_rejects_unbound_election_before_acquisition(self):
        with patch.dict(os.environ,{'GITHUB_SHA':'a'*40,'INSONIC_STARTUP_NORMAL':'true'},clear=True), \
             patch.object(diagnostic.os,'name','nt'),patch.object(diagnostic.pipeline.m,'disposable_runner'), \
             patch.object(diagnostic,'incident_archive') as acquire:
            with self.assertRaisesRegex(ValueError,'live single-child'):diagnostic.run()
        acquire.assert_not_called()

    def test_live_election_rejects_missing_gate_before_acquisition(self):
        with patch.dict(os.environ,{'GITHUB_SHA':'a'*40,'INSONIC_STARTUP_LIVE':'true'},clear=True), \
             patch.object(diagnostic.os,'name','nt'),patch.object(diagnostic.pipeline.m,'disposable_runner'), \
             patch.object(diagnostic,'incident_archive') as acquire:
            with self.assertRaisesRegex(ValueError,'single load gate'):diagnostic.run()
        acquire.assert_not_called()
        with self.assertRaisesRegex(ValueError,'owned-release'):
            diagnostic.capture(['fixture.exe'],ROOT,{},18,observe_live=True)

    @unittest.skipUnless(os.name=='nt','actual Windows live owned-job cleanup')
    def test_live_pending_query_does_not_postpone_actual_timeout_cleanup(self):
        import windows_native_resources
        entered,release=Event(),Event();created=[];original=windows_native_resources.LiveEndpoint
        def observer(*args):
            result=original(*args,delay=0);created.append(result);return result
        def members(job):entered.set();release.wait(2);return {'available':True,'pids':[]}
        try:
            with patch.object(windows_native_resources,'LiveEndpoint',side_effect=observer), \
                 patch.object(windows_native_resources,'owned_members',side_effect=members):
                result=diagnostic.capture([sys.executable,'-c','import time;time.sleep(30)'],ROOT,dict(os.environ),0.1,
                    require_release=True,observe_resources=True,observe_live=True)
                self.assertTrue(entered.is_set());self.assertTrue(result['timed_out'])
                self.assertLess(result['elapsed_seconds'],1)
                self.assertEqual(result['job_before_close']['assigned'],0)
                self.assertFalse(result['resources']['live']['available'])
                self.assertTrue(result['resources']['live']['overlapped_cleanup'])
        finally:
            release.set()
            for item in created:item.thread.join(2)

    def test_live_duplicate_never_defers_failed_job_cleanup_to_last_handle_close(self):
        import windows_native_resources
        tree=MagicMock();tree.__enter__.return_value=tree;tree.job=99
        tree.process.stdout=BytesIO();tree.process.stderr=BytesIO();tree.process.returncode=23;tree.process.poll.return_value=23
        live=MagicMock();live.finish.return_value={'available':False,'reason':'collection-not-completed'}
        with patch.object(diagnostic,'ProcessTree',return_value=tree), \
             patch.object(diagnostic,'job_processes',return_value={'available':True,'assigned':2,'pids':[12,13]}), \
             patch.object(windows_native_resources,'LiveEndpoint',return_value=live):
            result=diagnostic.capture(['fixture.exe'],ROOT,{},18,require_release=True,observe_resources=True,observe_live=True)
        tree.kill.assert_called_once();live.finish.assert_called_once()
        self.assertEqual(result['exit_code'],23);self.assertFalse(result['owned_release']);self.assertFalse(result['timed_out'])
        self.assertEqual(result['job_before_close']['assigned'],2)

    def test_live_duplicate_cannot_defer_cleanup_after_unexpected_wait_failure(self):
        import windows_native_resources
        tree=MagicMock();tree.__enter__.return_value=tree;tree.job=99
        tree.process.stdout=BytesIO();tree.process.stderr=BytesIO();tree.process.wait.side_effect=RuntimeError('original wait failure')
        live=MagicMock();live.finish.return_value={'available':False}
        with patch.object(diagnostic,'ProcessTree',return_value=tree), \
             patch.object(windows_native_resources,'LiveEndpoint',return_value=live):
            with self.assertRaisesRegex(RuntimeError,'original wait failure'):
                diagnostic.capture(['fixture.exe'],ROOT,{},18,require_release=True,observe_resources=True,observe_live=True)
        tree.kill.assert_called_once();live.finish.assert_called_once()

    def test_prelaunch_resource_observer_does_not_consume_native_release_budget(self):
        import windows_native_resources
        allowances=[]
        for mode in ['disabled','resources','live']:
            observe=mode!='disabled';live=mode=='live'
            clock=[0.0];tree=MagicMock();tree.__enter__.return_value=tree;tree.job=99
            tree.process.stdout=BytesIO();tree.process.stderr=BytesIO();tree.process.returncode=0
            tree.process.poll.return_value=0
            def wait(timeout):clock[0]+=17
            def release(timeout):
                allowances.append(timeout);clock[0]+=0.5
                return timeout>=0.5
            snapshots=[0]
            def snapshot(origin):
                if not snapshots[0]:clock[0]+=1
                snapshots[0]+=1
                return {'cpu':{'available':False}}
            anchors=[0]
            def anchor(origin):
                if not anchors[0]:clock[0]+=2
                anchors[0]+=1
                return {'unix_micros':123}
            tree.process.wait.side_effect=wait;tree.wait_for_descendants.side_effect=release
            with patch.object(diagnostic.time,'monotonic',side_effect=lambda:clock[0]), \
                 patch.object(diagnostic,'ProcessTree',return_value=tree), \
                 patch.object(diagnostic,'job_processes',return_value={'available':True,'assigned':0,'pids':[]}), \
                 patch.object(windows_native_resources,'system_snapshot',side_effect=snapshot), \
                 patch.object(windows_native_resources,'owned_snapshot',return_value={'available':True}), \
                 patch.object(windows_native_resources,'clock_anchor',side_effect=anchor), \
                 patch.object(windows_native_resources,'LiveEndpoint',return_value=MagicMock()):
                result=diagnostic.capture(['fixture.exe'],ROOT,{},18,require_release=True,observe_resources=observe,observe_live=live)
            self.assertTrue(result['owned_release']);self.assertEqual(result['timing']['root_exit_elapsed_seconds'],17)
        self.assertEqual(allowances,[1.0,1.0,1.0])

    def test_resource_observer_failure_never_replaces_the_original_child_result(self):
        import windows_native_resources
        with patch.object(windows_native_resources,'system_snapshot',side_effect=RuntimeError('private observer detail')):
            result=diagnostic.capture([sys.executable,'-c','import sys;sys.exit(23)'],ROOT,dict(os.environ),5,observe_resources=True)
        self.assertEqual(result['exit_code'],23);self.assertFalse(result['timed_out'])
        self.assertFalse(result['resources']['available'])
        self.assertNotIn('private observer detail',json.dumps(result['resources']))

    def test_resource_timeout_snapshot_is_explicitly_after_immediate_cleanup(self):
        result=diagnostic.capture([sys.executable,'-c','import time;time.sleep(30)'],ROOT,dict(os.environ),0.1,observe_resources=True)
        self.assertTrue(result['timed_out']);self.assertNotEqual(result['exit_code'],0)
        self.assertEqual(result['resources']['point'],'after-timeout-kill')
        self.assertFalse(result['feature_qualification'])

    def test_load_gate_fires_after_worker_submission_without_blocking_cli(self):
        values=[0.0]
        timeline=diagnostic.LoadTimeline(clock=lambda:values[0],gate=True)
        timeline.worker_start();timeline.phase('parallel-regressions')
        self.assertIsNone(timeline.receipt()['gate']['signal_elapsed_seconds'])
        values[0]=2.0;timeline.phase('media-lifecycle')
        # CLI can continue before the observer accepts the one-shot gate.
        values[0]=3.0;timeline.wait_gate()
        gate=timeline.receipt()['gate']
        self.assertEqual(gate['signal_elapsed_seconds'],2.0)
        self.assertEqual(gate['released_elapsed_seconds'],3.0)
        self.assertEqual(gate['prior_gui_launches'],0);self.assertEqual(gate['child_limit'],1)
        timeline.phase('media-lifecycle')
        self.assertEqual(timeline.receipt()['gate']['signal_elapsed_seconds'],2.0)

    def test_expired_load_gate_does_not_launch_after_phase_or_worker_end(self):
        for finished in [False,True]:
            timeline=diagnostic.LoadTimeline(gate=True);timeline.worker_start();timeline.phase('media-lifecycle')
            if finished:timeline.worker_end(True)
            else:timeline.phase('query-contracts')
            with self.assertRaisesRegex(RuntimeError,'phase gate expired'):timeline.wait_gate()
            self.assertIsNone(timeline.receipt()['gate']['released_elapsed_seconds'])

    def test_load_gate_wakes_on_early_worker_failure_and_has_a_fixed_wait(self):
        timeline=diagnostic.LoadTimeline(gate=True);timeline.worker_start();timeline.worker_end(False)
        with self.assertRaisesRegex(RuntimeError,'ended before'):timeline.wait_gate()
        pending=diagnostic.LoadTimeline(gate=True)
        with patch.object(pending._gate_event,'wait',return_value=False) as wait:
            with self.assertRaisesRegex(RuntimeError,'gate deadline'):pending.wait_gate()
        wait.assert_called_once_with(90)
        with self.assertRaisesRegex(ValueError,'not elected'):diagnostic.LoadTimeline().wait_gate()

    def test_load_gate_rejects_unbound_asset_control_before_acquisition(self):
        with patch.dict(os.environ,{'GITHUB_SHA':'a'*40,'INSONIC_STARTUP_LOAD_GATE':'true'},clear=True), \
             patch.object(diagnostic.os,'name','nt'),patch.object(diagnostic.pipeline.m,'disposable_runner'), \
             patch.object(diagnostic,'incident_archive') as acquire:
            with self.assertRaisesRegex(ValueError,'exact E008 CSS'):diagnostic.run()
        acquire.assert_not_called()

    def test_gated_observation_launches_exactly_one_child_with_original_deadline(self):
        timeline=diagnostic.LoadTimeline(gate=True);timeline.worker_start();timeline.phase('media-lifecycle');timeline.wait_gate()
        child=MagicMock(return_value={'exit_code':0,'timed_out':False,'truncated':False,
            'capture_incomplete':False,'owned_release':True,'stdout':'{"native_exit":"passed","action":"close","playback":"not-selected"}'})
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'capture',child),patch('builtins.print'):
            root=Path(temporary);summary={'observations':[]}
            diagnostic.observe_cold_start(root/'gui.exe',root,root,{},root/'evidence',summary,
                startup_hide=True,timeline=timeline,sample_limit=1,observe_resources=True)
        self.assertEqual(child.call_count,1);self.assertEqual(child.call_args.args[3],18)
        self.assertEqual(child.call_args.kwargs,{'require_release':True,'startup_hide':True,
            'observe_resources':True,'resource_origin':timeline.origin})
        self.assertEqual(len(summary['observations']),1);self.assertEqual(summary['sample_limit'],1)
        self.assertEqual(timeline.receipt()['gate']['prior_gui_launches'],0)
        with self.assertRaisesRegex(ValueError,'single load-gated'):
            diagnostic.observe_cold_start(root/'gui.exe',root,root,{},root/'evidence',{},observe_resources=True)

    def test_css_size_control_preserves_prefix_and_restores_after_success_or_failure(self):
        original=(ROOT/'desktop/frontend/src/app.css').read_bytes()
        script=b'x'*(len(original)+100)
        for failed in [False,True]:
            with self.subTest(failed=failed),tempfile.TemporaryDirectory() as temporary:
                root=Path(temporary);bundle=root/'desktop/assets/bundle';bundle.mkdir(parents=True)
                css=bundle/'product.css';css.write_bytes(original);(bundle/'app.js').write_bytes(script)
                with patch.object(diagnostic,'ROOT',root), \
                     patch.object(diagnostic,'SCRIPT_CONTROL_SHA256',hashlib.sha256(script).hexdigest()), \
                     patch.object(diagnostic,'CSS_CONTROL_BYTES',len(script)):
                    def compile_control():
                        with diagnostic.css_size_control_assets() as receipt:
                            padded=css.read_bytes()
                            self.assertEqual(len(padded),len(script))
                            self.assertTrue(padded.startswith(original+b'/*'))
                            self.assertEqual(padded[len(original):],b'/*'+b' '*95+b'*/\n')
                            self.assertEqual(receipt['padded_sha256'],hashlib.sha256(padded).hexdigest())
                            self.assertEqual(receipt['unchanged_app_script_sha256'],hashlib.sha256(script).hexdigest())
                            self.assertFalse(receipt['feature_qualification'])
                            if failed:raise RuntimeError('compiler failed')
                    if failed:
                        with self.assertRaisesRegex(RuntimeError,'compiler failed'):compile_control()
                    else:compile_control()
                self.assertEqual(css.read_bytes(),original)
                self.assertEqual((bundle/'app.js').read_bytes(),script)

    def test_css_size_control_rejects_changed_assets_without_writing(self):
        original=(ROOT/'desktop/frontend/src/app.css').read_bytes()
        for changed in ['css','script']:
            with self.subTest(changed=changed),tempfile.TemporaryDirectory() as temporary:
                root=Path(temporary);bundle=root/'desktop/assets/bundle';bundle.mkdir(parents=True)
                css=bundle/'product.css';css.write_bytes(original+(b'changed' if changed=='css' else b''))
                script=b'x'*diagnostic.CSS_CONTROL_BYTES;(bundle/'app.js').write_bytes(script)
                before=css.read_bytes()
                with patch.object(diagnostic,'ROOT',root), \
                     patch.object(diagnostic,'SCRIPT_CONTROL_SHA256',hashlib.sha256(script).hexdigest() if changed=='css' else '0'*64):
                    with self.assertRaisesRegex(ValueError,'exact E007'):
                        with diagnostic.css_size_control_assets():self.fail('changed assets accepted')
                self.assertEqual(css.read_bytes(),before)

    def test_css_size_control_rejects_incompatible_election_before_acquisition(self):
        for field in ['INSONIC_STARTUP_BROWSER','INSONIC_STARTUP_COLD','INSONIC_STARTUP_HIDE','INSONIC_STARTUP_SOURCE','INSONIC_STARTUP_LOAD','INSONIC_STARTUP_ABI_CONTROL']:
            env={'GITHUB_SHA':'a'*40,'INSONIC_STARTUP_CSS_CONTROL':'true','INSONIC_STARTUP_BROWSER':'true',
                 'INSONIC_STARTUP_COLD':'true','INSONIC_STARTUP_HIDE':'true','INSONIC_STARTUP_SOURCE':'true','INSONIC_STARTUP_LOAD':'true',
                 'INSONIC_STARTUP_ABI_CONTROL':'false'}
            env[field]='true' if field=='INSONIC_STARTUP_ABI_CONTROL' else 'false'
            with self.subTest(field=field),patch.dict(os.environ,env,clear=True), \
                 patch.object(diagnostic.os,'name','nt'),patch.object(diagnostic.pipeline.m,'disposable_runner'), \
                 patch.object(diagnostic,'incident_archive') as acquire:
                with self.assertRaisesRegex(ValueError,'CSS size control requires'):diagnostic.run()
                acquire.assert_not_called()

    def test_browser_observer_rejects_stale_or_incompatible_source_before_acquisition(self):
        for cold,source,control in [('true','false','false'),('false','true','false'),('true','true','true')]:
            with self.subTest(cold=cold,source=source,control=control), \
                 patch.dict(os.environ,{'GITHUB_SHA':'a'*40,'INSONIC_STARTUP_BROWSER':'true',
                    'INSONIC_STARTUP_COLD':cold,'INSONIC_STARTUP_SOURCE':source,
                    'INSONIC_STARTUP_ABI_CONTROL':control},clear=True), \
                 patch.object(diagnostic.os,'name','nt'), \
                 patch.object(diagnostic.pipeline.m,'disposable_runner'), \
                 patch.object(diagnostic,'incident_archive') as acquire:
                with self.assertRaisesRegex(ValueError,'browser observation requires current cold source'):
                    diagnostic.run()
                acquire.assert_not_called()

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

    def test_renderer_overlay_changes_only_actual_options_and_preserves_maintained_source(self):
        selected=Path('third_party/go-webview2/pkg/edge/chromium.go')
        original=(ROOT/selected).read_bytes()
        before=b'browserArgs := strings.Join(e.AdditionalBrowserArgs, " ")'
        after=b'browserArgs := strings.Join(append(append([]string{}, e.AdditionalBrowserArgs...), "--disable-renderer-backgrounding"), " ")'
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);source=root/selected;source.parent.mkdir(parents=True);source.write_bytes(original)
            with patch.object(diagnostic,'ROOT',root):
                receipt={};overlay=diagnostic.renderer_backgrounding_overlay(root/'control',receipt)
                replacement=json.loads(overlay.read_text())['Replace']
                self.assertEqual(set(replacement),{str(source.resolve())})
                changed=Path(replacement[str(source.resolve())]).read_bytes()
                self.assertEqual(changed,original.replace(before,after))
                self.assertEqual(changed.count(b'--disable-renderer-backgrounding'),1)
                self.assertEqual(source.read_bytes(),original)
                self.assertEqual(receipt['source_sha256'],hashlib.sha256(original).hexdigest())
                self.assertEqual(receipt['replacement_sha256'],hashlib.sha256(changed).hexdigest())
                self.assertFalse(receipt['feature_qualification'])
                self.assertNotIn(str(root),json.dumps(receipt))
                source.write_bytes(original+b'\nchanged\n')
                with self.assertRaisesRegex(ValueError,'checksum'):
                    diagnostic.renderer_backgrounding_overlay(root/'bad-control',{})
                self.assertFalse((root/'bad-control').exists())

    def test_renderer_overlay_is_build_only_and_combines_with_existing_asset_control(self):
        package=MagicMock();package.package_directories.side_effect=lambda root,*args:(root,root)
        for elected in [False,True]:
            with self.subTest(elected=elected),tempfile.TemporaryDirectory() as temporary:
                location=Path(temporary);original=location/'original';original.mkdir()
                (original/'insonic-desktop.exe').write_bytes(b'incident GUI')
                receipt={};css={}
                with patch.object(diagnostic,'build_probe_gui',return_value='a'*64) as build, \
                     patch.object(diagnostic,'renderer_backgrounding_overlay',return_value=location/'overlay.json') as overlay:
                    options={'renderer_control':receipt} if elected else {}
                    diagnostic.derive_gui(package,original,location,css_control=css,**options)
                self.assertEqual((original/'insonic-desktop.exe').read_bytes(),b'incident GUI')
                expected={'css_control':css}
                if elected:
                    overlay.assert_called_once_with(location/'renderer-control-source',receipt)
                    expected['overlay']=location/'overlay.json'
                else:overlay.assert_not_called()
                self.assertEqual(build.call_args.kwargs,expected)

    def test_renderer_control_rejects_missing_control_chain_before_acquisition(self):
        env={'GITHUB_SHA':'a'*40,'INSONIC_STARTUP_RENDERER_CONTROL':'true',
             'INSONIC_STARTUP_SCHEDULER':'true','INSONIC_STARTUP_NORMAL':'true','INSONIC_STARTUP_LIVE':'true',
             'INSONIC_STARTUP_LOAD_GATE':'true','INSONIC_STARTUP_CSS_CONTROL':'true',
             'INSONIC_STARTUP_BROWSER':'true','INSONIC_STARTUP_COLD':'true','INSONIC_STARTUP_HIDE':'true',
             'INSONIC_STARTUP_SOURCE':'true','INSONIC_STARTUP_LOAD':'true','INSONIC_STARTUP_ABI_CONTROL':'false'}
        for field in [key for key in env if key.startswith('INSONIC_') and key!='INSONIC_STARTUP_RENDERER_CONTROL']:
            selected={**env,field:'true' if field=='INSONIC_STARTUP_ABI_CONTROL' else 'false'}
            with self.subTest(field=field),patch.dict(os.environ,selected,clear=True), \
                 patch.object(diagnostic.os,'name','nt'),patch.object(diagnostic.pipeline.m,'disposable_runner'), \
                 patch.object(diagnostic,'incident_archive') as acquire:
                with self.assertRaises(ValueError):diagnostic.run()
                acquire.assert_not_called()

    def test_workload_cpu_control_requires_full_scheduler_chain_without_renderer_flag(self):
        env={'GITHUB_SHA':'a'*40,'INSONIC_STARTUP_WORKLOAD_CPU_CONTROL':'true',
             'INSONIC_STARTUP_RENDERER_CONTROL':'false','INSONIC_STARTUP_SCHEDULER':'true',
             'INSONIC_STARTUP_NORMAL':'true','INSONIC_STARTUP_LIVE':'true',
             'INSONIC_STARTUP_LOAD_GATE':'true','INSONIC_STARTUP_CSS_CONTROL':'true',
             'INSONIC_STARTUP_BROWSER':'true','INSONIC_STARTUP_COLD':'true','INSONIC_STARTUP_HIDE':'true',
             'INSONIC_STARTUP_SOURCE':'true','INSONIC_STARTUP_LOAD':'true','INSONIC_STARTUP_ABI_CONTROL':'false'}
        for field in [None]+[key for key in env if key.startswith('INSONIC_') and
                             key not in ('INSONIC_STARTUP_WORKLOAD_CPU_CONTROL','INSONIC_STARTUP_RENDERER_CONTROL')]:
            selected=dict(env)
            if field is not None:selected[field]='true' if field=='INSONIC_STARTUP_ABI_CONTROL' else 'false'
            with self.subTest(field=field),patch.dict(os.environ,selected,clear=True), \
                 patch.object(diagnostic.os,'name','nt'),patch.object(diagnostic.pipeline.m,'disposable_runner'), \
                 patch.object(diagnostic,'incident_archive',side_effect=RuntimeError('acquisition reached')) as acquire:
                if field is None:
                    with self.assertRaisesRegex(RuntimeError,'acquisition reached'):diagnostic.run()
                    acquire.assert_called_once()
                else:
                    with self.assertRaises(ValueError):diagnostic.run()
                    acquire.assert_not_called()

    def test_workload_receipt_covers_initial_launch_and_preflight_failures(self):
        for phase in ['initial-launch','scheduler-preflight']:
            with self.subTest(phase=phase),tempfile.TemporaryDirectory() as temporary:
                evidence=Path(temporary);counts={'launches':0,'verified_before_resume':0,'failed_launches':0}
                factory=MagicMock();factory.receipt.side_effect=lambda:dict(counts)
                with self.assertRaisesRegex(RuntimeError,phase):
                    with diagnostic.ExitStack() as stack:
                        diagnostic.retain_workload_receipt(stack,evidence,factory)
                        counts.update(launches=1,verified_before_resume=int(phase=='scheduler-preflight'),
                                      failed_launches=int(phase=='initial-launch'))
                        raise RuntimeError(phase)
                self.assertEqual(json.loads((evidence/'workload-cpu-control.json').read_text()),counts)
        with patch.object(diagnostic,'save',side_effect=OSError('secondary receipt error')):
            with self.assertRaisesRegex(RuntimeError,'original native failure'):
                with diagnostic.ExitStack() as stack:
                    diagnostic.retain_workload_receipt(stack,Path('.'),factory)
                    raise RuntimeError('original native failure')

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


    def test_normal_control_requires_one_complete_effective_receipt_before_comparison(self):
        receipt={'kind':'native-priority-control','wrapper_pid':101,'gui_pid':102,
                 'wrapper_priority_available':True,'wrapper_priority_class':16384,
                 'requested_gui_priority_class':32,'gui_priority_available':True,
                 'gui_priority_class':32,'valid':True,
                 'probe_begin_unix_micros':1000,'probe_end_unix_micros':1001}
        encoded=json.dumps(receipt)
        valid={'exit_code':0,'timed_out':False,'truncated':False,'capture_incomplete':False,'owned_release':True,
               'stdout':json.dumps({'action':'close','native_exit':'passed','playback':'not-selected'}),'stderr':encoded}
        cases=[({'exit_code':78,'stdout':'','stderr':json.dumps({'kind':'native-priority-control','valid':False,
                    'reason':'expected-below-normal-wrapper-unavailable'})},False),
               ({'stderr':''},False),({'stderr':encoded+'\n'+encoded},False),
               ({'stderr':'{"kind":"native-priority-control"'},False),
               ({'stderr':json.dumps(dict(receipt,gui_priority_class=16384))},False),
               ({'stderr':json.dumps(dict(receipt,wrapper_priority_available=False))},False),
               ({'stderr':json.dumps(dict(receipt,gui_pid=True))},False),
               ({'stderr':json.dumps(dict(receipt,probe_end_unix_micros=999))},False),
               ({'capture_incomplete':True},False),({'truncated':True},False),
               ({'exit_code':23,'stdout':''},True),({},True)]
        for update,available in cases:
            with self.subTest(update=update),tempfile.TemporaryDirectory() as temporary, \
                 patch.object(diagnostic,'capture',return_value=dict(valid,**update)) as child,patch('builtins.print'):
                root=Path(temporary);summary={'observations':[]}
                if available and not update:
                    diagnostic.observe_cold_start(root/'gui.exe',root,root,{'INSONIC_STARTUP_NORMAL':'true'},root/'evidence',summary,sample_limit=1)
                    expected='cold-start-no-failure-observed-not-a-cause-proof'
                else:
                    expected='cold-start-failure-observed' if available else 'normal-priority-control-unavailable'
                    with self.assertRaisesRegex(RuntimeError,'cold startup' if available else 'priority control unavailable'):
                        diagnostic.observe_cold_start(root/'gui.exe',root,root,{'INSONIC_STARTUP_NORMAL':'true'},root/'evidence',summary,sample_limit=1)
                self.assertEqual(summary['result'],expected);child.assert_called_once()
                retained=json.loads((root/'evidence/cold-000.json').read_text())
                self.assertEqual(retained['exit_code'],update.get('exit_code',0))
                self.assertEqual(retained['priority_control']['available'],available)
                self.assertEqual(summary['observations'][0]['observed_pass'],available and not update)

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


    def test_load_timeline_distinguishes_submission_actual_overlap_and_finished_work(self):
        now=[100.0]
        timeline=diagnostic.LoadTimeline(clock=lambda:now[0])
        now[0]=101;pending=timeline.case_begin(0)
        now[0]=102;pending=timeline.case_end(pending)
        now[0]=103;timeline.worker_start();timeline.phase('help')
        now[0]=104;active=timeline.case_begin(1)
        now[0]=106;timeline.phase('credentials')
        now[0]=108;active=timeline.case_end(active)
        now[0]=109;timeline.worker_end(True)
        now[0]=110;finished=timeline.case_begin(2)
        now[0]=112;finished=timeline.case_end(finished)
        self.assertEqual(pending['begin'],{'elapsed_seconds':1,'phase':'not-started','worker_state':'pending'})
        self.assertEqual(pending['overlap_seconds'],0)
        self.assertEqual(active['begin'],{'elapsed_seconds':4,'phase':'help','worker_state':'running'})
        self.assertEqual(active['end'],{'elapsed_seconds':8,'phase':'credentials','worker_state':'running'})
        self.assertEqual(active['overlap_seconds'],4)
        self.assertEqual(finished['overlap_seconds'],0)
        self.assertEqual(finished['begin']['worker_state'],'passed')
        receipt=timeline.receipt()
        self.assertEqual(receipt['worker'],{'start_elapsed_seconds':3,'end_elapsed_seconds':9,'state':'passed'})
        self.assertEqual(receipt['cases'],[pending,active,finished])
        self.assertFalse(receipt['feature_qualification'])

    def test_load_timeline_worker_interval_can_start_and_end_inside_one_case(self):
        now=[20.0];timeline=diagnostic.LoadTimeline(clock=lambda:now[0])
        now[0]=21;case=timeline.case_begin(0)
        def workload(*args,**options):
            self.assertEqual(args,('fixed-input',));self.assertEqual(options['documentation'],True)
            now[0]=23;options['progress']('operator-feedback')
            now[0]=24;raise RuntimeError('private fixture details')
        now[0]=22
        with self.assertRaisesRegex(RuntimeError,'private fixture details'):
            diagnostic.run_load(timeline,workload,'fixed-input',documentation=True)
        now[0]=25;case=timeline.case_end(case)
        self.assertEqual(case['overlap_seconds'],2)
        self.assertEqual(case['begin']['worker_state'],'pending')
        self.assertEqual(case['end']['worker_state'],'failed')
        encoded=json.dumps(timeline.receipt())
        self.assertNotIn('private',encoded);self.assertNotIn('fixed-input',encoded)

    def test_load_timeline_bounds_threaded_phases_cases_and_rejects_unfixed_labels(self):
        from concurrent.futures import ThreadPoolExecutor
        timeline=diagnostic.LoadTimeline();timeline.worker_start()
        with ThreadPoolExecutor(max_workers=2) as workers:
            list(workers.map(timeline.phase,['help','credentials']*40))
        for index in range(240):timeline.case_end(timeline.case_begin(index))
        with self.assertRaises(ValueError):timeline.case_begin(240)
        with self.assertRaises(ValueError):timeline.phase('private arbitrary argv')
        timeline.worker_end(True)
        receipt=timeline.receipt()
        self.assertEqual(len(receipt['phase_events']),16)
        self.assertEqual(receipt['dropped_phase_events'],64)
        self.assertEqual(len(receipt['cases']),240)
        self.assertNotIn('private',json.dumps(receipt))
        self.assertTrue(all(case['overlap_seconds']>=0 for case in receipt['cases']))

    def test_native_failure_still_exports_actual_load_timeline_before_owned_idle(self):
        timeline=diagnostic.LoadTimeline()
        with self.assertRaisesRegex(RuntimeError,'CLI failure'):
            diagnostic.run_load(timeline,MagicMock(side_effect=RuntimeError('CLI failure')))
        future=MagicMock();future.result.side_effect=RuntimeError('CLI failure')
        with tempfile.TemporaryDirectory() as temporary:
            evidence=Path(temporary)
            def idle():
                receipt=json.loads((evidence/'concurrent-load-timeline.json').read_text())
                self.assertEqual(receipt['worker']['state'],'failed')
            with self.assertRaisesRegex(RuntimeError,'native failure'):
                diagnostic.observe_with_load(MagicMock(side_effect=RuntimeError('native failure')),future,idle,evidence,timeline=timeline)

    def test_cold_case_retains_begin_end_phases_and_original_launch_contract(self):
        now=[0.0];timeline=diagnostic.LoadTimeline(clock=lambda:now[0]);timeline.worker_start();timeline.phase('help')
        def child(args,cwd,env,timeout,**options):
            self.assertEqual(timeout,18);self.assertEqual(options,{'require_release':True,'startup_hide':True})
            now[0]=3;timeline.phase('credentials');now[0]=4
            return {'exit_code':1,'timed_out':False,'truncated':False,'capture_incomplete':False,'owned_release':False,'stdout':'{}'}
        with tempfile.TemporaryDirectory() as temporary,patch.object(diagnostic,'capture',side_effect=child),patch('builtins.print'):
            root=Path(temporary);summary={'observations':[]}
            with self.assertRaisesRegex(RuntimeError,'cold startup'):
                diagnostic.observe_cold_start(root/'gui.exe',root,root,{},root/'evidence',summary,startup_hide=True,timeline=timeline)
            result=json.loads((root/'evidence/cold-000.json').read_text())
        self.assertEqual(result['concurrent_load']['begin']['phase'],'help')
        self.assertEqual(result['concurrent_load']['end']['phase'],'credentials')
        self.assertEqual(result['concurrent_load']['overlap_seconds'],4)

if __name__=='__main__':unittest.main()
