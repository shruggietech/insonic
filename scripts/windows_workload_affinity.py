# SPDX-License-Identifier: Apache-2.0
"""Explicit disposable-workload CPU control; never changes the shared parent."""
import ctypes as C
import os
from threading import Lock

from process_tree import ProcessTree


class _BasicLimits(C.Structure):
    _fields_ = [('process_time', C.c_int64), ('job_time', C.c_int64), ('flags', C.c_uint32),
                ('minimum', C.c_size_t), ('maximum', C.c_size_t), ('active', C.c_uint32),
                ('affinity', C.c_size_t), ('priority', C.c_uint32), ('scheduling', C.c_uint32)]


class _IOCounters(C.Structure):
    _fields_ = [(name, C.c_uint64) for name in
                ['read_ops', 'write_ops', 'other_ops', 'read_bytes', 'write_bytes', 'other_bytes']]


class _ExtendedLimits(C.Structure):
    _fields_ = [('basic', _BasicLimits), ('io', _IOCounters), ('process_memory', C.c_size_t),
                ('job_memory', C.c_size_t), ('peak_process', C.c_size_t), ('peak_job', C.c_size_t)]


def _native_error():
    return C.WinError(C.get_last_error())


class _AffinityAssignment:
    """Instance-local forwarding during the base launcher's suspended assignment."""
    def __init__(self, api, mask):
        self.api, self.mask, self.verified = api, mask, None

    def __getattr__(self, name):
        return getattr(self.api, name)

    def OpenProcess(self, access, inherit, pid):
        return self.api.OpenProcess(access | 0x0600, inherit, pid)

    def AssignProcessToJobObject(self, job, handle):
        # A job limit alone does not propagate to breakaway descendants. Set the
        # owned root's default mask before assignment, then verify before resume.
        if not self.api.SetProcessAffinityMask(handle, self.mask):
            raise _native_error()
        if not self.api.AssignProcessToJobObject(job, handle):
            raise _native_error()
        effective, system = C.c_size_t(), C.c_size_t()
        if not self.api.GetProcessAffinityMask(handle, C.byref(effective), C.byref(system)):
            raise _native_error()
        if effective.value != self.mask:
            raise RuntimeError('suspended owned root affinity was not applied')
        self.verified = effective.value
        return True


class AffinityProcessTree(ProcessTree):
    """Elected qualification extension; shared source-build launcher stays intact."""
    def __init__(self, args, *, cpu_affinity, **options):
        if os.name != 'nt' or type(cpu_affinity) is not int or not 0 < cpu_affinity < 1 << 64:
            raise ValueError('explicit Windows child affinity must be a positive 64-bit mask')
        self._requested_affinity = cpu_affinity
        self.cpu_affinity_verified = None
        super().__init__(args, **options)

    def _windows_job(self, allow_detached):
        super()._windows_job(allow_detached)
        try:
            self.api.SetProcessAffinityMask.argtypes = [C.c_void_p, C.c_size_t]
            self.api.SetProcessAffinityMask.restype = C.c_int
            self.api.GetProcessAffinityMask.argtypes = [C.c_void_p, C.POINTER(C.c_size_t), C.POINTER(C.c_size_t)]
            self.api.GetProcessAffinityMask.restype = C.c_int
            limits = _ExtendedLimits()
            if not self.api.QueryInformationJobObject(self.job, 9, C.byref(limits), C.sizeof(limits), None):
                raise _native_error()
            limits.basic.flags |= 0x0010  # JOB_OBJECT_LIMIT_AFFINITY; keep base kill/breakaway flags.
            limits.basic.affinity = self._requested_affinity
            if not self.api.SetInformationJobObject(self.job, 9, C.byref(limits), C.sizeof(limits)):
                raise _native_error()
            verified = _ExtendedLimits()
            if not self.api.QueryInformationJobObject(self.job, 9, C.byref(verified), C.sizeof(verified), None):
                raise _native_error()
            if verified.basic.flags != limits.basic.flags or verified.basic.affinity != self._requested_affinity:
                raise RuntimeError('owned job affinity was not applied')
        except BaseException:
            # The base creates its job before entering constructor cleanup.
            try:
                self.close()
            except Exception:
                pass
            raise

    def _assign_and_resume(self):
        original = self.api
        assignment = _AffinityAssignment(original, self._requested_affinity)
        self.api = assignment
        try:
            super()._assign_and_resume()
            self.cpu_affinity_verified = assignment.verified
        finally:
            self.api = original


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
        return self._launch(lambda: AffinityProcessTree(args,**options,cpu_affinity=self.mask), options)

    def terminal(self,args,**options):
        from terminal_transport import Terminal
        owner=AffinityProcessTree.__new__(AffinityProcessTree)
        owner._requested_affinity=self.mask
        owner.cpu_affinity_verified=None
        return self._launch(lambda: Terminal(args,**options,owner=owner), {})

    def _launch(self,launch,options):
        with self.lock:self.started+=1
        try:
            tree=launch()
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
                'coverage':'explicit-cli-media-example-and-terminal-roots-descendants-inherit-native-affinity',
                'parent_or_gui_affinity_changed':False}
