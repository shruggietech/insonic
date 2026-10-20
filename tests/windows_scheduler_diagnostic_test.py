# SPDX-License-Identifier: Apache-2.0
import copy
import importlib.util
import json
import os
from pathlib import Path
import sys
import time
import tempfile
import unittest
from unittest.mock import MagicMock, patch

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'scripts'))
import windows_scheduler_diagnostic as scheduler
spec=importlib.util.spec_from_file_location('scheduler_startup_diagnostic',ROOT/'scripts/diagnose-desktop-startup.py')
diagnostic=importlib.util.module_from_spec(spec);spec.loader.exec_module(diagnostic)
preflight_spec=importlib.util.spec_from_file_location('console_scheduler_preflight',ROOT/'scripts/diagnose-scheduler-preflight.py')
preflight=importlib.util.module_from_spec(preflight_spec);preflight_spec.loader.exec_module(preflight)


class SchedulerTests(unittest.TestCase):
    def owned(self):
        return {'available':True,'overflow':False,'processes':[{'pid':10,'owned':True,'identity':{
            'available':True,'created_unix_micros':90,'query_begin_unix_micros':100,'query_end_unix_micros':110}}]}

    def test_missing_or_unverified_or_overflowed_identity_cannot_be_interpreted(self):
        start={'unix_micros':95};end={'unix_micros':120}
        valid=self.owned()
        self.assertEqual(scheduler.observation_input(start,end,valid)['owned'][0]['created_unix_micros'],90)
        variants=[dict(valid,available=False),dict(valid,overflow=True),dict(valid,processes=[]),
                  dict(valid,processes=valid['processes']*2)]
        for field in ['owned','identity']:
            value=copy.deepcopy(valid);value['processes'][0][field]=False if field=='owned' else {'available':False}
            variants.append(value)
        for row in variants:
            with self.subTest(row=row),self.assertRaises(scheduler.Unavailable):scheduler.observation_input(start,end,row)

    def test_console_entry_checks_disposable_guard_before_any_tool_creation(self):
        tools=MagicMock();tools.pipeline.m.disposable_runner.side_effect=RuntimeError('not disposable')
        with patch.object(preflight,'Scheduler') as create:
            with self.assertRaisesRegex(RuntimeError,'not disposable'):preflight.run(tools)
        create.assert_not_called();tools.capture.assert_not_called()

    @unittest.skipUnless(os.name=='nt','Windows scratch initialization')
    def test_fresh_workspace_initializes_private_scratch_without_tool_or_trace(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);self.assertFalse((root/'build').exists())
            capture=MagicMock();guard=MagicMock()
            with patch('builtins.print'):
                with scheduler.Scheduler(root,capture,guard) as observer:
                    private=observer.private
                    self.assertEqual(private.parent,root.resolve()/'build')
                    self.assertTrue(private.is_dir());self.assertFalse(observer.active)
            self.assertFalse(private.exists());self.assertTrue(observer.cleanup_confirmed)
            capture.assert_not_called();guard.assert_called_once()

    def test_console_failure_is_unavailable_with_zero_product_launches_and_cleanup_receipt(self):
        tools=MagicMock();observer=MagicMock();observer.__enter__.return_value=observer
        observer.prepare.side_effect=RuntimeError('private detail')
        observer.failure.return_value={'available':False,'reason':'fixed-rejection','cleanup_confirmed':True}
        observer.cleanup_confirmed=True;observer.private=ROOT/'build/nonexistent-scheduler-test-private'
        rows={};tools.save.side_effect=lambda directory,name,row:rows.update({name:copy.deepcopy(row)})
        with patch.object(preflight,'Scheduler',return_value=observer):self.assertEqual(preflight.run(tools),78)
        observer.prepare.assert_called_once();observer.cancel.assert_called_once()
        tools.capture.assert_not_called()
        self.assertEqual(rows['preflight.json']['gui_launches'],0)
        self.assertFalse(rows['preflight.json']['available']);self.assertTrue(rows['cleanup.json']['confirmed'])
        self.assertNotIn('private detail',json.dumps(rows))

    def test_console_initialization_failure_has_receipt_before_constructor(self):
        tools=MagicMock();rows={};tools.save.side_effect=lambda directory,name,row:rows.update({name:copy.deepcopy(row)})
        def fail(*args):
            self.assertIn('preflight.json',rows)
            raise OSError('private path')
        with patch.object(preflight,'Scheduler',side_effect=fail):self.assertEqual(preflight.run(tools),78)
        self.assertEqual(rows['preflight.json']['reason'],'observer-initialization-not-completed')
        self.assertFalse(rows['cleanup.json']['confirmed']);self.assertFalse(rows['cleanup.json']['session_started'])
        self.assertNotIn('private path',json.dumps(rows));tools.capture.assert_not_called()

    def test_stress_fixture_is_a_single_hidden_bounded_console_child(self):
        started=time.monotonic()
        with scheduler.ProcessTree([sys.executable,'-c',scheduler.STRESS_FIXTURE],cwd=ROOT,env=dict(os.environ),input_pipe=True) as tree:
            output,error=tree.process.communicate(input=b'G',timeout=5)
            self.assertEqual(tree.process.returncode,0,error.decode('utf-8',errors='replace'))
            row=json.loads(output)
            self.assertEqual(row['pid'],tree.process.pid);self.assertGreater(row['tid'],0)
            self.assertTrue(tree.wait_for_descendants(max(0,5-(time.monotonic()-started))))
        self.assertLess(time.monotonic()-started,5)

    def test_console_stress_requires_explicit_election(self):
        for elected in ['false','true']:
            tools=MagicMock();observer=MagicMock();observer.__enter__.return_value=observer
            observer.prepare.side_effect=RuntimeError('stop before tools')
            observer.failure.return_value={'available':False};observer.cleanup_confirmed=True
            observer.private=ROOT/'build/nonexistent-scheduler-stress-test'
            with patch.dict(os.environ,{'INSONIC_SCHEDULER_STRESS':elected}),patch.object(preflight,'Scheduler',return_value=observer):
                self.assertEqual(preflight.run(tools),78)
            if elected=='true':observer.prepare.assert_called_once_with(stress=True)
            else:observer.prepare.assert_called_once_with()
            tools.capture.assert_not_called()

    def test_real_preflight_requires_correlated_main_thread_with_known_wait_and_scheduled_work(self):
        row={'available':True,'processes':[{'pid':10,'matched_lifetime':True,'threads':[
            {'tid':11,'summary':{'waiting':{'total_ns':200_000_000},'scheduled':{'total_ns':2_000_000}}}]}]}
        self.assertTrue(scheduler.preflight_valid(row,10,11))
        self.assertFalse(scheduler.preflight_valid(row,20,11));self.assertFalse(scheduler.preflight_valid(row,10,12))
        for state in ['waiting','scheduled']:
            invalid=copy.deepcopy(row);invalid['processes'][0]['threads'][0]['summary'][state]['total_ns']=0
            self.assertFalse(scheduler.preflight_valid(invalid,10,11))

    def test_preflight_receipt_keeps_conflict_uncertainty_without_private_fields_or_weaker_work_gate(self):
        row={'available':True,'state_accounting':'known-lower-bounds','overlap_policy':'conflicts-unknown',
             'processes':[{'pid':10,'matched_lifetime':True,'threads':[{'tid':11,
                 'summary':{'unknown_ns':1000,'waiting':{'total_ns':200_000_000},'scheduled':{'total_ns':2_000_000}},
                 'conflict_unknown_ns':200,'conflict_intervals':1,
                 'first_conflict':{'overlap_ns':200},'closed_inputs':{'unproved-closed-gap':2},'private_path':'not-for-retention'}]}]}
        receipt=scheduler.preflight_receipt(row,10,11,True)
        self.assertEqual(receipt['main_thread']['summary']['unknown_ns'],1000)
        self.assertEqual(receipt['main_thread']['conflict_unknown_ns'],200)
        self.assertEqual(receipt['main_thread']['conflict_intervals'],1)
        self.assertEqual(receipt['main_thread']['first_conflict'],{'overlap_ns':200})
        self.assertEqual(receipt['main_thread']['closed_inputs'],{'unproved-closed-gap':2})
        self.assertEqual(receipt['state_accounting'],'known-lower-bounds');self.assertEqual(receipt['stress_threads'],4)
        self.assertNotIn('private_path',json.dumps(receipt));self.assertNotIn('not-for-retention',json.dumps(receipt))
        row['processes'][0]['threads'][0]['summary']['waiting']['total_ns']=0
        with self.assertRaises(scheduler.Unavailable):scheduler.preflight_receipt(row,10,11,True)

    def test_cancel_targets_only_the_unique_owned_session_and_does_not_retry(self):
        instance=scheduler.Scheduler.__new__(scheduler.Scheduler)
        instance.wpr='wpr.exe';instance.instance='insonic-s040-owned';instance.active=True;instance.cleanup_confirmed=True
        instance.tool=MagicMock(side_effect=scheduler.Unavailable('timeout'))
        instance.cancel()
        instance.tool.assert_called_once_with(['wpr.exe','-cancel','-instancename','insonic-s040-owned'],15)
        self.assertFalse(instance.active);self.assertFalse(instance.cleanup_confirmed)

    def test_invalid_decode_and_cleanup_never_modify_the_original_native_failure(self):
        with tempfile.TemporaryDirectory() as temporary:
            instance=scheduler.Scheduler.__new__(scheduler.Scheduler)
            instance.private=Path(temporary);raw=instance.private/'scheduler.etl';raw.write_bytes(b'private')
            instance.stop=MagicMock(return_value=raw);instance.decode=MagicMock(side_effect=ValueError('private error'))
            instance.active=False;instance.cleanup_confirmed=True;instance.preflight={'available':True}
            instance.stage='';instance.last_tool=None;instance.decoder_hresult=None
            instance.decoder_rejection=None;instance.preflight_input=None
            instance.preflight_projection=None
            native={'exit_code':23,'timed_out':True,'stderr':'original','timing':{'start_clock_anchor':{'unix_micros':95},
                'end_clock_anchor':{'unix_micros':120}},'resources':{'live':{'available':True,'owned':self.owned()}}}
            original=copy.deepcopy(native)
            result=instance.finish(native,{'available':True,'gui_pid':10,'wrapper_pid':9})
            self.assertFalse(result['available']);self.assertEqual(result['stage'],'actual-decode')
            self.assertEqual(native,original);self.assertFalse(raw.exists())
            self.assertNotIn('private error',json.dumps(result));self.assertTrue(result['cleanup_confirmed'])

    def test_scheduler_election_rejected_before_payload_acquisition(self):
        with patch.dict(os.environ,{'GITHUB_SHA':'a'*40,'INSONIC_STARTUP_SCHEDULER':'true'},clear=True), \
             patch.object(diagnostic.os,'name','nt'),patch.object(diagnostic.pipeline.m,'disposable_runner'), \
             patch.object(diagnostic,'incident_archive') as acquire:
            with self.assertRaisesRegex(ValueError,'Normal live control'):diagnostic.run()
        acquire.assert_not_called()

    def test_gate_failure_cancels_owned_trace_before_cli_join_and_idle(self):
        order=[]
        future=MagicMock();future.result.side_effect=lambda:order.append('join')
        def fail():order.append('gate');raise RuntimeError('original gate failure')
        with patch.object(diagnostic,'save'):
            with self.assertRaisesRegex(RuntimeError,'original gate failure'):
                diagnostic.observe_with_load(fail,future,lambda:order.append('idle'),ROOT,
                    abandon=lambda:order.append('cancel'))
        self.assertEqual(order,['gate','cancel','join','idle'])

    def test_native_failure_is_saved_before_scheduler_stop_and_stays_failed(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);summary={'observations':[]};order=[];timeline=MagicMock();timeline.origin=0;timeline.case_end.return_value={}
            native={'exit_code':1,'timed_out':True,'truncated':False,'capture_incomplete':False,'owned_release':False,
                    'stdout':'','stderr':'original','timing':{}}
            observer=MagicMock()
            def finish(observation,control):
                self.assertEqual(order,['cold-000.json','diagnostic.json'])
                self.assertEqual(observation['exit_code'],1)
                return {'available':False}
            observer.finish.side_effect=finish
            with patch.object(diagnostic,'capture',return_value=native) as capture, \
                 patch.object(diagnostic,'save',side_effect=lambda directory,name,value:order.append(name)), \
                 patch('builtins.print'):
                with self.assertRaisesRegex(RuntimeError,'cold startup failed'):
                    diagnostic.observe_cold_start('gui',root,root,{},root/'evidence',summary,sample_limit=1,
                        timeline=timeline,observe_resources=True,observe_live=True,scheduler=observer)
            self.assertEqual(summary['result'],'cold-start-failure-observed')
            self.assertEqual(capture.call_args.args[3],18);self.assertTrue(capture.call_args.kwargs['scheduler_identity'])
            self.assertFalse(summary['scheduler_available'])


if __name__=='__main__':unittest.main()
