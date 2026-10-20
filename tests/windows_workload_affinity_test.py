# SPDX-License-Identifier: Apache-2.0
import json
import ctypes
import os
from pathlib import Path
import sys
from threading import Lock
import unittest
from unittest.mock import MagicMock,patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'scripts'))
from windows_workload_affinity import WorkloadAffinity,selected_mask,AffinityProcessTree,_AffinityAssignment


def first_available_processor():
    api=ctypes.WinDLL('kernel32',use_last_error=True)
    api.GetCurrentProcess.restype=ctypes.c_void_p
    api.GetProcessAffinityMask.argtypes=[ctypes.c_void_p,ctypes.POINTER(ctypes.c_size_t),ctypes.POINTER(ctypes.c_size_t)]
    process,system=ctypes.c_size_t(),ctypes.c_size_t()
    if not api.GetProcessAffinityMask(api.GetCurrentProcess(),ctypes.byref(process),ctypes.byref(system)):
        raise ctypes.WinError(ctypes.get_last_error())
    return process.value & -process.value


class WorkloadAffinityTests(unittest.TestCase):
    def test_assignment_verifies_after_real_assignment_and_before_resume(self):
        events=[];api=MagicMock()
        api.OpenProcess.return_value=77
        api.SetProcessAffinityMask.side_effect=lambda *args:events.append('root-default') or 1
        api.AssignProcessToJobObject.side_effect=lambda *args:events.append('assigned') or 1
        def queried(handle,process,system):
            events.append('verified');process._obj.value=7;return 1
        api.GetProcessAffinityMask.side_effect=queried
        forwarding=_AffinityAssignment(api,7)
        self.assertEqual(forwarding.OpenProcess(0x101,False,123),77)
        api.OpenProcess.assert_called_once_with(0x701,False,123)
        self.assertTrue(forwarding.AssignProcessToJobObject(99,77))
        self.assertEqual(events,['root-default','assigned','verified'])
        self.assertEqual(forwarding.verified,7)

    @unittest.skipUnless(os.name=='nt','actual suspended-root failure cleanup')
    def test_failed_control_never_resumes_root_and_preserves_original_error(self):
        import tempfile
        mask=first_available_processor()
        for phase in ['root-default','assignment','query','mismatch']:
            with self.subTest(phase=phase),tempfile.TemporaryDirectory() as temporary:
                marker=Path(temporary)/'must-not-run'
                instances=[]
                def failed(*args):
                    ctypes.set_last_error(5);return 0
                class Broken(AffinityProcessTree):
                    def _windows_job(self,allow_detached):
                        super()._windows_job(allow_detached)
                        instances.append(self)
                        if phase=='root-default':self.api.SetProcessAffinityMask=failed
                        elif phase=='assignment':self.api.AssignProcessToJobObject=failed
                        elif phase=='query':self.api.GetProcessAffinityMask=failed
                        else:
                            def mismatch(handle,process,system):process._obj.value=0;return 1
                            self.api.GetProcessAffinityMask=mismatch
                expected=RuntimeError if phase=='mismatch' else OSError
                with self.assertRaises(expected) as error:
                    Broken([sys.executable,'-c','import pathlib,sys;pathlib.Path(sys.argv[1]).write_text("ran")',str(marker)],
                           cwd=temporary,env=dict(os.environ),cpu_affinity=mask)
                if phase!='mismatch':self.assertEqual(error.exception.winerror,5)
                self.assertFalse(marker.exists())
                self.assertIsNotNone(instances[0].process.poll())
                self.assertIsNone(instances[0].job)
                self.assertNotIsInstance(instances[0].api,_AffinityAssignment)

    @unittest.skipUnless(os.name=='nt','actual prelaunch job failure cleanup')
    def test_job_setup_failure_closes_before_base_constructor_cleanup_exists(self):
        from process_tree import ProcessTree
        original=ProcessTree._windows_job
        for phase in ['initial-query','set-limit','verification-query','mismatch']:
            with self.subTest(phase=phase):
                instances=[]
                def setup(instance,allow_detached):
                    original(instance,allow_detached)
                    instances.append(instance)
                    real_query=instance.api.QueryInformationJobObject
                    queries=0
                    def failed(*args):ctypes.set_last_error(5);return 0
                    def query(*args):
                        nonlocal queries
                        queries+=1
                        if phase=='initial-query' or (queries==2 and phase=='verification-query'):
                            return failed()
                        result=real_query(*args)
                        if queries==2 and phase=='mismatch':args[2]._obj.basic.affinity=0
                        return result
                    instance.api.QueryInformationJobObject=query
                    if phase=='set-limit':instance.api.SetInformationJobObject=failed
                expected=RuntimeError if phase=='mismatch' else OSError
                with patch.object(ProcessTree,'_windows_job',setup),patch('process_tree.subprocess.Popen') as launch:
                    with self.assertRaises(expected) as error:
                        AffinityProcessTree([sys.executable],cwd='.',env=dict(os.environ),cpu_affinity=first_available_processor())
                if phase!='mismatch':self.assertEqual(error.exception.winerror,5)
                launch.assert_not_called();self.assertIsNone(instances[0].job)

    def test_explicit_affinity_rejects_invalid_values_before_launch(self):
        for mask in [0,-1,True,'3',1 << 64]:
            with self.subTest(mask=mask),patch('process_tree.subprocess.Popen') as launch:
                with self.assertRaises(ValueError):AffinityProcessTree(['unused.exe'],cwd='.',env={},cpu_affinity=mask)
                launch.assert_not_called()

    @unittest.skipUnless(os.name == 'nt', 'actual owned Windows affinity inheritance')
    def test_owned_affinity_precedes_execution_and_inherits_through_breakaway(self):
        api=ctypes.WinDLL('kernel32',use_last_error=True)
        api.GetCurrentProcess.restype=ctypes.c_void_p
        api.GetProcessAffinityMask.argtypes=[ctypes.c_void_p,ctypes.POINTER(ctypes.c_size_t),ctypes.POINTER(ctypes.c_size_t)]
        before,system=ctypes.c_size_t(),ctypes.c_size_t()
        self.assertTrue(api.GetProcessAffinityMask(api.GetCurrentProcess(),ctypes.byref(before),ctypes.byref(system)))
        mask=before.value & -before.value
        common="""import ctypes as C,json
api=C.WinDLL('kernel32',use_last_error=True)
api.GetCurrentProcess.restype=C.c_void_p
api.GetProcessAffinityMask.argtypes=[C.c_void_p,C.POINTER(C.c_size_t),C.POINTER(C.c_size_t)]
def current_mask():
    process,system=C.c_size_t(),C.c_size_t()
    if not api.GetProcessAffinityMask(api.GetCurrentProcess(),C.byref(process),C.byref(system)):raise RuntimeError('query failed')
    return process.value
"""
        probe=common+'print(json.dumps(current_mask()))'
        parent=common+"\nimport subprocess,sys\nrows={'root':current_mask()}\n"
        parent+=f"probe={probe!r}\n"
        parent+="for name,flags in [('normal',0x08000000),('breakaway',0x09000000)]:\n"
        parent+="    result=subprocess.run([sys.executable,'-c',probe],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=flags,timeout=5,check=True)\n    rows[name]=json.loads(result.stdout)\n"
        # A fresh nested job without its own affinity must preserve inheritance.
        parent+=f"sys.path.insert(0,{str(Path(__file__).resolve().parents[1]/'scripts')!r})\nfrom process_tree import ProcessTree\n"
        parent+="with ProcessTree([sys.executable,'-c',probe],cwd='.',env=dict(__import__('os').environ)) as nested:\n    output,error=nested.process.communicate(timeout=5)\n    assert nested.process.returncode==0,error\n    rows['nested_job']=json.loads(output)\nprint(json.dumps(rows))\n"
        with AffinityProcessTree([sys.executable,'-c',parent],cwd=Path(__file__).resolve().parents[1],env=dict(os.environ),
                         allow_detached=True,cpu_affinity=mask) as tree:
            self.assertEqual(tree.cpu_affinity_verified,mask)
            output,error=tree.process.communicate(timeout=15)
            self.assertEqual(tree.process.returncode,0,error.decode(errors='replace'))
            self.assertEqual(json.loads(output),dict.fromkeys(['root','normal','breakaway','nested_job'],mask))
            self.assertTrue(tree.wait_for_descendants(5))
        after,system=ctypes.c_size_t(),ctypes.c_size_t()
        self.assertTrue(api.GetProcessAffinityMask(api.GetCurrentProcess(),ctypes.byref(after),ctypes.byref(system)))
        self.assertEqual(after.value,before.value)

    def test_only_one_of_exactly_four_available_processors_is_reserved(self):
        self.assertEqual(selected_mask(15,15,1),7)
        self.assertEqual(selected_mask(0b110110,0b110110,1),0b010110)
        for process,system,groups in [(0,0,1),(7,7,1),(31,31,1),(15,31,1),(15,15,2),(True,True,1)]:
            with self.subTest(process=process,system=system,groups=groups),self.assertRaises(ValueError):
                selected_mask(process,system,groups)

    def test_factory_keeps_override_environments_and_records_only_fixed_counts(self):
        selected=WorkloadAffinity.__new__(WorkloadAffinity)
        selected.mask=7;selected.permitted=15;selected.lock=Lock()
        selected.started=selected.verified=selected.failed=selected.breakaway=0
        tree=MagicMock()
        with patch('windows_workload_affinity.AffinityProcessTree',return_value=tree) as launch:
            self.assertIs(selected(['private-executable'],cwd='private-path',env={'private':'value'},allow_detached=True),tree)
        self.assertEqual(launch.call_args.kwargs,{'cwd':'private-path','env':{'private':'value'},'allow_detached':True,'cpu_affinity':7})
        with patch('windows_workload_affinity.AffinityProcessTree',side_effect=RuntimeError('private failure')):
            with self.assertRaises(RuntimeError):selected(['other-private'],cwd='.',env={})
        receipt=selected.receipt()
        self.assertEqual((receipt['launches'],receipt['verified_before_resume'],receipt['failed_launches'],receipt['breakaway_permitted_roots']),(2,1,1,1))
        self.assertFalse(receipt['parent_or_gui_affinity_changed'])
        self.assertFalse(receipt['feature_qualification'])
        self.assertNotIn('private',json.dumps(receipt))


if __name__=='__main__':unittest.main()
