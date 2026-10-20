# SPDX-License-Identifier: Apache-2.0
"""Explicit disposable-workload CPU control; never changes the shared parent."""
import ctypes as C
from threading import Lock

from process_tree import ProcessTree


def selected_mask(process_mask, system_mask, groups):
    if (type(process_mask) is not int or type(system_mask) is not int or groups != 1
            or process_mask != system_mask or process_mask <= 0 or process_mask.bit_count() != 4):
        raise ValueError('workload CPU control requires exactly four available processors in one group')
    return process_mask ^ (1 << (process_mask.bit_length() - 1))


class WorkloadAffinity:
    def __init__(self):
        api=C.WinDLL('kernel32',use_last_error=True)
        api.GetCurrentProcess.argtypes=[];api.GetCurrentProcess.restype=C.c_void_p
        api.GetActiveProcessorGroupCount.argtypes=[];api.GetActiveProcessorGroupCount.restype=C.c_ushort
        api.GetProcessAffinityMask.argtypes=[C.c_void_p,C.POINTER(C.c_size_t),C.POINTER(C.c_size_t)]
        api.GetProcessAffinityMask.restype=C.c_int
        process,system=C.c_size_t(),C.c_size_t()
        if not api.GetProcessAffinityMask(api.GetCurrentProcess(),C.byref(process),C.byref(system)):
            raise RuntimeError('workload CPU selection unavailable')
        self.permitted=process.value
        self.mask=selected_mask(process.value,system.value,api.GetActiveProcessorGroupCount())
        self.lock=Lock();self.started=0;self.verified=0;self.failed=0;self.breakaway=0

    def __call__(self,args,**options):
        with self.lock:self.started+=1
        try:
            tree=ProcessTree(args,**options,cpu_affinity=self.mask)
        except BaseException:
            with self.lock:self.failed+=1
            raise
        with self.lock:
            self.verified+=1
            if options.get('allow_detached'):self.breakaway+=1
        return tree

    def receipt(self):
        with self.lock:
            return {'kind':'owned-workload-affinity-control','feature_qualification':False,
                'permitted_mask':self.permitted,'requested_mask':self.mask,'reserved_from_workload_mask':self.permitted^self.mask,
                'available_processors':4,'workload_processors':3,'launches':self.started,'verified_before_resume':self.verified,
                'failed_launches':self.failed,'breakaway_permitted_roots':self.breakaway,
                'coverage':'explicit-cli-media-and-example-roots-descendants-inherit-native-affinity',
                'parent_or_gui_affinity_changed':False}
