# SPDX-License-Identifier: Apache-2.0
"""Fixed endpoint counters for disposable Windows startup diagnostics."""
import ctypes as C
from functools import lru_cache
import os
import time


class FileTime(C.Structure):
    _fields_=[('low',C.c_uint32),('high',C.c_uint32)]


class Performance(C.Structure):
    _fields_=[('cb',C.c_uint32)]+[(name,C.c_size_t) for name in [
        'commit','commit_limit','commit_peak','physical','available','cache',
        'kernel','kernel_paged','kernel_nonpaged','page_size']]+[
        (name,C.c_uint32) for name in ['handles','processes','threads']]


class ProcessMemory(C.Structure):
    _fields_=[('cb',C.c_uint32),('faults',C.c_uint32)]+[
        (name,C.c_size_t) for name in ['peak_working','working','peak_paged','paged',
            'peak_nonpaged','nonpaged','pagefile','peak_pagefile','private']]


class JobAccounting(C.Structure):
    _fields_=[(name,C.c_int64) for name in ['user','kernel','period_user','period_kernel']]+[
        (name,C.c_uint32) for name in ['faults','total','active','terminated']]


def ticks(value):return (value.high<<32)|value.low


@lru_cache(maxsize=1)
def api():
    selected=C.WinDLL('kernel32',use_last_error=True)
    signatures={
        'GetSystemTimes':([C.POINTER(FileTime)]*3,C.c_int),
        'K32GetPerformanceInfo':([C.POINTER(Performance),C.c_uint32],C.c_int),
        'QueryInformationJobObject':([C.c_void_p,C.c_int,C.c_void_p,C.c_uint32,C.c_void_p],C.c_int),
        'OpenProcess':([C.c_uint32,C.c_int,C.c_uint32],C.c_void_p),
        'IsProcessInJob':([C.c_void_p,C.c_void_p,C.POINTER(C.c_int)],C.c_int),
        'GetPriorityClass':([C.c_void_p],C.c_uint32),
        'GetProcessTimes':([C.c_void_p]+[C.POINTER(FileTime)]*4,C.c_int),
        'K32GetProcessMemoryInfo':([C.c_void_p,C.POINTER(ProcessMemory),C.c_uint32],C.c_int),
        'CloseHandle':([C.c_void_p],C.c_int),
    }
    for name,(arguments,result) in signatures.items():
        function=getattr(selected,name);function.argtypes=arguments;function.restype=result
    return selected


def failed(reason):return {'available':False,'reason':reason}


def native_failed():return {'available':False,'error':C.get_last_error()}


def system_snapshot(origin):
    if os.name!='nt':return failed('platform-unavailable')
    result={'start_elapsed_seconds':round(time.monotonic()-origin,6)}
    try:
        selected=api();idle,kernel,user=FileTime(),FileTime(),FileTime()
        result['cpu']=({'available':True,'scope':'calling-processor-group','tick_unit':'100ns','idle_ticks':ticks(idle),
                        'kernel_ticks':ticks(kernel),'user_ticks':ticks(user)}
                       if selected.GetSystemTimes(C.byref(idle),C.byref(kernel),C.byref(user)) else native_failed())
        memory=Performance();memory.cb=C.sizeof(memory)
        memory_ok=selected.K32GetPerformanceInfo(C.byref(memory),C.sizeof(memory))
        if memory_ok and memory.page_size:
            page=memory.page_size
            result['memory']={'available':True,'physical_bytes':memory.physical*page,
                'available_physical_bytes':memory.available*page,'commit_bytes':memory.commit*page,
                'commit_limit_bytes':memory.commit_limit*page,'processes':memory.processes,
                'threads':memory.threads,'handles':memory.handles}
        else:result['memory']=failed('invalid-page-size') if memory_ok else native_failed()
        result['available']=True
    except Exception:
        result.update(failed('observer-unavailable'))
    result['end_elapsed_seconds']=round(time.monotonic()-origin,6)
    return result


def cpu_interval(before,after):
    first,last=before.get('cpu',{}),after.get('cpu',{})
    if not first.get('available') or not last.get('available'):return failed('cpu-unavailable')
    differences={key:last[key]-first[key] for key in ['idle_ticks','kernel_ticks','user_ticks']}
    total=differences['kernel_ticks']+differences['user_ticks']
    if any(value<0 for value in differences.values()) or total<=0 or differences['idle_ticks']>total:
        return failed('invalid-cpu-interval')
    return {'available':True,'scope':'calling-processor-group','kind':'interval-average-not-peak','tick_unit':'100ns',
            'busy_percent':round(100*(total-differences['idle_ticks'])/total,3),**differences}


def owned_snapshot(job,pids):
    # PID enumeration belongs to the existing job observer. Verify membership
    # again on each opened handle before querying it, including PID reuse races.
    if os.name!='nt':return failed('platform-unavailable')
    if not job or any(type(pid) is not int or not 0<pid<=0xffffffff for pid in pids):
        return failed('invalid-owned-selection')
    result={'available':True,'selected_count':len(set(pids)),'process_limit':16,
            'overflow':len(set(pids))>16,'processes':[]}
    try:
        selected=api();account=JobAccounting()
        if selected.QueryInformationJobObject(job,1,C.byref(account),C.sizeof(account),None):
            result['job']={'available':True,'scope':'job-lifetime','tick_unit':'100ns','user_ticks':account.user,'kernel_ticks':account.kernel,
                'page_faults':account.faults,'total_processes':account.total,
                'active_processes':account.active,'limit_terminated_processes':account.terminated}
        else:result['job']=native_failed()
        for pid in sorted(set(pids))[:16]:
            row={'pid':pid};result['processes'].append(row)
            handle=selected.OpenProcess(0x1000,False,pid)  # QUERY_LIMITED_INFORMATION only.
            if not handle:
                row.update(native_failed());continue
            try:
                owned=C.c_int()
                if not selected.IsProcessInJob(handle,job,C.byref(owned)):
                    row.update(native_failed());continue
                if not owned.value:
                    row.update(failed('not-in-owned-job'));continue
                row['owned']=True
                priority=selected.GetPriorityClass(handle)
                row['priority']=({'available':True,'class':priority} if priority else native_failed())
                created,exited,kernel,user=FileTime(),FileTime(),FileTime(),FileTime()
                row['cpu']=({'available':True,'scope':'process-lifetime','tick_unit':'100ns','kernel_ticks':ticks(kernel),'user_ticks':ticks(user)}
                    if selected.GetProcessTimes(handle,C.byref(created),C.byref(exited),C.byref(kernel),C.byref(user)) else native_failed())
                memory=ProcessMemory();memory.cb=C.sizeof(memory)
                row['memory']=({'available':True,'working_set_bytes':memory.working,
                    'peak_working_set_bytes':memory.peak_working,'private_commit_bytes':memory.private}
                    if selected.K32GetProcessMemoryInfo(handle,C.byref(memory),C.sizeof(memory)) else native_failed())
            finally:selected.CloseHandle(handle)
    except Exception:
        result.update(failed('observer-unavailable'))
    return result
