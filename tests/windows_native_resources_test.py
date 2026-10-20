# SPDX-License-Identifier: Apache-2.0
import ctypes
import importlib.util
import json
import os
import sys
from pathlib import Path
import time
import unittest
from unittest.mock import MagicMock,patch
from threading import Event

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'scripts'))
spec=importlib.util.spec_from_file_location('native_resources',ROOT/'scripts/windows_native_resources.py')
resources=importlib.util.module_from_spec(spec);spec.loader.exec_module(resources)


class NativeResourcesTests(unittest.TestCase):
    def test_windows_x64_layouts(self):
        self.assertEqual(ctypes.sizeof(resources.FileTime),8)
        self.assertEqual(ctypes.sizeof(resources.JobAccounting),48)
        if ctypes.sizeof(ctypes.c_void_p)==8:
            self.assertEqual(ctypes.sizeof(resources.Performance),104)
            self.assertEqual(ctypes.sizeof(resources.ProcessMemory),80)

    def test_cpu_interval_subtracts_idle_from_kernel_and_is_not_peak(self):
        before={'cpu':{'available':True,'idle_ticks':100,'kernel_ticks':200,'user_ticks':300}}
        after={'cpu':{'available':True,'idle_ticks':150,'kernel_ticks':300,'user_ticks':400}}
        result=resources.cpu_interval(before,after)
        self.assertEqual(result['busy_percent'],75)
        self.assertEqual(result['kind'],'interval-average-not-peak')
        self.assertFalse(resources.cpu_interval({},after)['available'])
        self.assertFalse(resources.cpu_interval(after,before)['available'])
        self.assertFalse(resources.cpu_interval(before,before)['available'])
        invalid={'cpu':{'available':True,'idle_ticks':1000,'kernel_ticks':201,'user_ticks':301}}
        self.assertFalse(resources.cpu_interval(before,invalid)['available'])

    def test_pid_reuse_or_foreign_membership_prevents_all_process_queries(self):
        selected=MagicMock();selected.QueryInformationJobObject.return_value=0
        selected.OpenProcess.return_value=77
        selected.IsProcessInJob.side_effect=lambda handle,job,out:1  # Leaves owned=false.
        with patch.object(resources.os,'name','nt'),patch.object(resources,'api',return_value=selected), \
             patch.object(resources,'native_failed',return_value={'available':False,'error':5}):
            result=resources.owned_snapshot(99,[123])
        self.assertEqual(result['processes'][0]['reason'],'not-in-owned-job')
        selected.OpenProcess.assert_called_once_with(0x1000,False,123)
        selected.GetPriorityClass.assert_not_called();selected.GetProcessTimes.assert_not_called()
        selected.K32GetProcessMemoryInfo.assert_not_called();selected.CloseHandle.assert_called_once_with(77)

    def test_process_queries_are_bounded_and_unavailable_is_not_zero(self):
        selected=MagicMock();selected.QueryInformationJobObject.return_value=0
        selected.OpenProcess.return_value=77;selected.GetPriorityClass.return_value=32
        def owned(handle,job,out):out._obj.value=1;return 1
        selected.IsProcessInJob.side_effect=owned
        selected.GetProcessTimes.return_value=0;selected.K32GetProcessMemoryInfo.return_value=0
        with patch.object(resources.os,'name','nt'),patch.object(resources,'api',return_value=selected), \
             patch.object(resources,'native_failed',return_value={'available':False,'error':5}):
            result=resources.owned_snapshot(99,list(range(1,31)))
        self.assertTrue(result['overflow']);self.assertEqual(len(result['processes']),16)
        self.assertEqual(selected.CloseHandle.call_count,16)
        for row in result['processes']:
            self.assertTrue(row['owned']);self.assertFalse(row['cpu']['available'])
            self.assertNotIn('kernel_ticks',row['cpu']);self.assertFalse(row['memory']['available'])

    def test_observer_failure_does_not_emit_exception_text(self):
        with patch.object(resources.os,'name','nt'),patch.object(resources,'api',side_effect=RuntimeError('private value')):
            snapshots=[resources.system_snapshot(time.monotonic()),resources.owned_snapshot(99,[123])]
        self.assertTrue(all(not row['available'] for row in snapshots))
        self.assertNotIn('private value',json.dumps(snapshots))

    def test_clock_anchor_brackets_utc_integers_without_float_roundtrip(self):
        value=1792454400123456789
        with patch.object(resources.time,'monotonic',side_effect=[10.0,10.001]), \
             patch.object(resources.time,'time_ns',return_value=value):
            row=resources.clock_anchor(9.0)
        self.assertEqual(row,{'unix_micros':value//1000,'begin_elapsed_seconds':1.0,'end_elapsed_seconds':1.001})
        self.assertIs(type(row['unix_micros']),int);self.assertLess(row['unix_micros'],2**53)

    def test_live_duplicate_failure_is_unavailable_without_starting_worker(self):
        selected=MagicMock();selected.DuplicateHandle.return_value=0
        with patch.object(resources,'api',return_value=selected):
            observer=resources.LiveEndpoint(99,time.monotonic(),time.monotonic());row=observer.finish()
        self.assertIsNone(observer.thread);self.assertFalse(row['available'])
        self.assertEqual(row['reason'],'query-handle-unavailable');selected.CloseHandle.assert_not_called()

    def test_live_finish_does_not_wait_for_a_blocked_observer(self):
        entered,release=Event(),Event()
        selected=MagicMock();selected.GetCurrentProcess.return_value=1
        def duplicate(source,job,target,out,*rest):out._obj.value=88;return 1
        selected.DuplicateHandle.side_effect=duplicate
        def collect(job):
            entered.set();release.wait(2);return {'available':True,'pids':[123]}
        with patch.object(resources,'api',return_value=selected),patch.object(resources,'owned_members',side_effect=collect), \
             patch.object(resources,'system_snapshot',return_value={'available':True}), \
             patch.object(resources,'owned_snapshot',return_value={'available':True}):
            observer=resources.LiveEndpoint(99,time.monotonic(),time.monotonic(),delay=0)
            try:
                self.assertTrue(entered.wait(1))
                started=time.monotonic();row=observer.finish(cleanup_elapsed_seconds=0.1)
                self.assertLess(time.monotonic()-started,0.1)
                self.assertFalse(row['available']);self.assertTrue(row['overlapped_cleanup'])
                self.assertEqual(row['reason'],'collection-not-completed')
            finally:release.set();observer.thread.join(2)
        selected.CloseHandle.assert_called_once_with(88)

    def test_live_cancel_before_endpoint_performs_no_process_queries(self):
        selected=MagicMock();selected.GetCurrentProcess.return_value=1
        def duplicate(source,job,target,out,*rest):out._obj.value=88;return 1
        selected.DuplicateHandle.side_effect=duplicate
        with patch.object(resources,'api',return_value=selected),patch.object(resources,'owned_members') as members:
            observer=resources.LiveEndpoint(99,time.monotonic(),time.monotonic())
            row=observer.finish();observer.thread.join(1)
        self.assertFalse(observer.thread.is_alive());members.assert_not_called()
        selected.CloseHandle.assert_called_once_with(88)
        self.assertFalse(row['available']);self.assertEqual(row['scheduled_native_seconds'],15)

    @unittest.skipUnless(os.name=='nt','actual Windows live owned endpoint')
    def test_actual_live_owned_endpoint_uses_duplicate_and_bounded_members(self):
        from process_tree import ProcessTree
        with ProcessTree([sys.executable,'-c','import time;time.sleep(30)'],cwd=ROOT,env=dict(os.environ)) as tree:
            observer=resources.LiveEndpoint(tree.job,time.monotonic(),time.monotonic(),delay=0)
            try:
                observer.thread.join(2);row=observer.finish()
                self.assertFalse(observer.thread.is_alive());self.assertTrue(row['available'])
                self.assertTrue(row['owned']['job']['available'])
                self.assertGreaterEqual(row['owned']['selected_count'],1)
                self.assertLessEqual(row['owned']['selected_count'],16)
                self.assertIn(tree.process.pid,[item['pid'] for item in row['owned']['processes']])
                self.assertTrue(all(item['owned'] and item['priority']['available'] for item in row['owned']['processes']))
            finally:
                tree.kill();tree.process.communicate(timeout=5)

    @unittest.skipUnless(os.name=='nt','actual Windows API endpoint')
    def test_actual_system_counter_endpoints_without_gui(self):
        result=resources.system_snapshot(time.monotonic())
        self.assertTrue(result['available']);self.assertTrue(result['cpu']['available'])
        self.assertTrue(result['memory']['available'])
        self.assertGreater(result['memory']['physical_bytes'],0)
        self.assertGreater(result['memory']['processes'],0)
        self.assertLessEqual(result['memory']['available_physical_bytes'],result['memory']['physical_bytes'])
        self.assertGreaterEqual(result['end_elapsed_seconds'],result['start_elapsed_seconds'])

    @unittest.skipUnless(os.name=='nt','actual Windows owned job endpoint')
    def test_actual_owned_job_counters_with_hidden_noninteractive_child(self):
        from process_tree import ProcessTree
        with ProcessTree([sys.executable,'-c','import time;time.sleep(30)'],cwd=ROOT,env=dict(os.environ)) as tree:
            result=resources.owned_snapshot(tree.job,[tree.process.pid])
            self.assertTrue(result['available']);self.assertTrue(result['job']['available'])
            self.assertGreaterEqual(result['job']['active_processes'],1)
            row=result['processes'][0]
            self.assertTrue(row['owned']);self.assertTrue(row['cpu']['available'])
            self.assertTrue(row['memory']['available']);self.assertTrue(row['priority']['available'])
        tree.process.communicate(timeout=5)


if __name__=='__main__':unittest.main()
