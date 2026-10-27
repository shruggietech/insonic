# SPDX-License-Identifier: Apache-2.0
"""Hidden qualification children with descendant lifetime ownership."""
from contextlib import contextmanager
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing
import os
import shutil
import tempfile
import time
import signal
import subprocess


def selected_executable(args, cwd, env):
    argv = [os.fspath(value) for value in args]
    if os.name != 'nt' or not argv or os.path.dirname(argv[0]):
        return argv
    # CreateProcess does not use the supplied child PATH for bare executable
    # discovery. Resolve it explicitly before launching with the hidden flags.
    program = argv[0]
    extensions = [''] if os.path.splitext(program)[1] else [''] + env.get('PATHEXT', '.COM;.EXE;.BAT;.CMD').split(';')
    for directory in env.get('PATH', '').split(os.pathsep):
        if not directory:
            continue
        parent = os.path.abspath(os.path.join(os.fspath(cwd), directory.strip('"')))
        for extension in extensions:
            candidate = os.path.join(parent, program + extension)
            if os.path.isfile(candidate):
                argv[0] = candidate
                return argv
    raise FileNotFoundError('selected child executable is unavailable: ' + program)


NATIVE_PROFILE = 'INSONIC_NATIVE_WEBVIEW_PROFILE'


def native_webview_probe(args, env):
    return os.name == 'nt' and bool(env.get(NATIVE_PROFILE)) and any(
        flag in args for flag in ['--help-qualification', '--webview-qualification'])


@contextmanager
def native_webview_environment(env):
    selected = dict(env)
    if os.name != 'nt':
        yield selected
        return
    # The caller must observe the owned job reaching zero before returning.
    # Preserve failures for diagnosis; never remove a profile still in use.
    root = Path(tempfile.mkdtemp(prefix='insonic-native-webview-'))
    selected[NATIVE_PROFILE] = str(root / 'webview')
    try:
        yield selected
    except BaseException:
        raise
    else:
        shutil.rmtree(root)


def unix_process_identity(pid, proc_root=Path('/proc')):
    try:
        raw = (proc_root / str(pid) / 'stat').read_bytes()
    except (FileNotFoundError, ProcessLookupError):
        return None
    # comm may contain spaces and parentheses. Fields after its final ')' are
    # state, ppid, pgrp, session, ... starttime (the stable process identity).
    fields = raw.rsplit(b')', 1)[1].split()
    return fields[0].decode('ascii'), int(fields[3]), int(fields[19])


def unix_session_members(session_id, proc_root=Path('/proc')):
    if not proc_root.is_dir():
        raise RuntimeError('Linux process inventory is unavailable for owned session cleanup')
    result = {}
    for path in proc_root.iterdir():
        if path.name.isdecimal():
            identity = unix_process_identity(int(path.name), proc_root)
            if identity and identity[1] == session_id and identity[0] not in ('Z', 'X'):
                result[int(path.name)] = identity[2]
    return result


def kill_unix_member(pid, start_time, session_id):
    # pidfds keep PID reuse from redirecting termination to another process.
    handle = None
    try:
        handle = os.pidfd_open(pid)
        identity = unix_process_identity(pid)
        if identity and identity[1:] == (session_id, start_time):
            signal.pidfd_send_signal(handle, signal.SIGKILL)
    except ProcessLookupError:
        pass
    finally:
        if handle is not None:
            os.close(handle)


class ProcessTree:
    def __init__(self, args, *, cwd, env, allow_detached=False, input_pipe=False):
        self.job = None
        self.process = None
        self.session_id = None
        args = selected_executable(args, cwd, env)
        if os.name == 'nt':
            self._windows_job(allow_detached)
        try:
            self.process = subprocess.Popen(args, cwd=cwd, env=env, stdin=subprocess.PIPE if input_pipe else subprocess.DEVNULL,
                                            stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                            start_new_session=os.name != 'nt',
                                            creationflags=0x08000004 if os.name == 'nt' else 0)
            if self.job:
                self._assign_and_resume()
            elif os.name != 'nt':
                self.session_id = self.process.pid
        except BaseException as primary:
            try:
                if self.process:
                    self.process.kill()
                    self.process.communicate(timeout=timing.CLEANUP_SECONDS)
                self.close()
            except BaseException:
                primary.add_note("Owned process creation cleanup did not complete")
            raise

    def _windows_job(self, allow_detached):
        import ctypes
        from ctypes import wintypes
        self.ctypes = ctypes
        self.api = ctypes.WinDLL('kernel32', use_last_error=True)
        size = ctypes.c_size_t
        class Basic(ctypes.Structure):
            _fields_ = [('process_time', ctypes.c_int64), ('job_time', ctypes.c_int64),
                        ('flags', wintypes.DWORD), ('minimum', size), ('maximum', size),
                        ('active', wintypes.DWORD), ('affinity', size),
                        ('priority', wintypes.DWORD), ('scheduling', wintypes.DWORD)]
        class IO(ctypes.Structure):
            _fields_ = [(name, ctypes.c_uint64) for name in ['read_ops', 'write_ops', 'other_ops', 'read_bytes', 'write_bytes', 'other_bytes']]
        class Extended(ctypes.Structure):
            _fields_ = [('basic', Basic), ('io', IO), ('process_memory', size),
                        ('job_memory', size), ('peak_process', size), ('peak_job', size)]
        class Thread(ctypes.Structure):
            _fields_ = [('size', wintypes.DWORD), ('usage', wintypes.DWORD),
                        ('id', wintypes.DWORD), ('owner', wintypes.DWORD),
                        ('base_priority', wintypes.LONG), ('delta_priority', wintypes.LONG),
                        ('flags', wintypes.DWORD)]
        self.thread_type = Thread
        signatures = {
            'CreateJobObjectW': ([ctypes.c_void_p, wintypes.LPCWSTR], wintypes.HANDLE),
            'SetInformationJobObject': ([wintypes.HANDLE, ctypes.c_int, ctypes.c_void_p, wintypes.DWORD], wintypes.BOOL),
            'QueryInformationJobObject': ([wintypes.HANDLE, ctypes.c_int, ctypes.c_void_p, wintypes.DWORD, ctypes.c_void_p], wintypes.BOOL),
            'AssignProcessToJobObject': ([wintypes.HANDLE, wintypes.HANDLE], wintypes.BOOL),
            'OpenProcess': ([wintypes.DWORD, wintypes.BOOL, wintypes.DWORD], wintypes.HANDLE),
            'CreateToolhelp32Snapshot': ([wintypes.DWORD, wintypes.DWORD], wintypes.HANDLE),
            'Thread32First': ([wintypes.HANDLE, ctypes.POINTER(Thread)], wintypes.BOOL),
            'Thread32Next': ([wintypes.HANDLE, ctypes.POINTER(Thread)], wintypes.BOOL),
            'OpenThread': ([wintypes.DWORD, wintypes.BOOL, wintypes.DWORD], wintypes.HANDLE),
            'ResumeThread': ([wintypes.HANDLE], wintypes.DWORD),
            'TerminateJobObject': ([wintypes.HANDLE, wintypes.UINT], wintypes.BOOL),
            'CloseHandle': ([wintypes.HANDLE], wintypes.BOOL),
        }
        for name, (arguments, result) in signatures.items():
            function = getattr(self.api, name)
            function.argtypes, function.restype = arguments, result
        self.job = self.api.CreateJobObjectW(None, None)
        if not self.job:
            raise ctypes.WinError(ctypes.get_last_error())
        limits = Extended()
        limits.basic.flags = 0x2000  # JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
        if allow_detached:
            # Only the owned runtime bootstrap elects an independent lifetime.
            limits.basic.flags |= 0x0800  # JOB_OBJECT_LIMIT_BREAKAWAY_OK
        if not self.api.SetInformationJobObject(self.job, 9, ctypes.byref(limits), ctypes.sizeof(limits)):
            self.close()
            raise ctypes.WinError(ctypes.get_last_error())

    def _assign_and_resume(self):
        c = self.ctypes
        handle = self.api.OpenProcess(0x0100 | 0x0001, False, self.process.pid)
        if not handle:
            raise c.WinError(c.get_last_error())
        try:
            if not self.api.AssignProcessToJobObject(self.job, handle):
                raise c.WinError(c.get_last_error())
        finally:
            self.api.CloseHandle(handle)
        primary = getattr(self.process, 'primary_thread', None)
        if primary is not None:
            # Native terminal creation retains the exact suspended primary
            # handle. An auxiliary console thread must never substitute for it.
            suspended = self.api.ResumeThread(primary)
            if suspended == 0xffffffff:
                raise c.WinError(c.get_last_error())
            if suspended != 1:
                raise RuntimeError('Owned primary thread had unexpected suspend count')
            return
        snapshot = self.api.CreateToolhelp32Snapshot(4, 0)
        if snapshot == c.c_void_p(-1).value:
            raise c.WinError(c.get_last_error())
        try:
            entry = self.thread_type()
            entry.size = c.sizeof(entry)
            found = self.api.Thread32First(snapshot, c.byref(entry))
            while found:
                if entry.owner == self.process.pid:
                    thread = self.api.OpenThread(2, False, entry.id)
                    if not thread:
                        raise c.WinError(c.get_last_error())
                    try:
                        if self.api.ResumeThread(thread) == 0xffffffff:
                            raise c.WinError(c.get_last_error())
                        return
                    finally:
                        self.api.CloseHandle(thread)
                found = self.api.Thread32Next(snapshot, c.byref(entry))
            raise RuntimeError('suspended process thread is unavailable')
        finally:
            self.api.CloseHandle(snapshot)

    def wait_for_descendants(self, timeout):
        if not self.job:
            if self.session_id is None:
                return True
            deadline = time.monotonic() + max(0, timeout)
            while unix_session_members(self.session_id):
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    return False
                time.sleep(min(.02, remaining))
            self.session_id = None
            return True
        c = self.ctypes
        from ctypes import wintypes
        class Accounting(c.Structure):
            _fields_ = [(name, c.c_int64) for name in ['user', 'kernel', 'period_user', 'period_kernel']] + [
                (name, wintypes.DWORD) for name in ['faults', 'total', 'active', 'terminated']]
        deadline = time.monotonic() + max(0, timeout)
        while True:
            value = Accounting()
            if not self.api.QueryInformationJobObject(self.job, 1, c.byref(value), c.sizeof(value), None):
                raise c.WinError(c.get_last_error())
            if value.active == 0:
                return True
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                return False
            time.sleep(min(0.02, remaining))

    def kill(self):
        if self.job:
            self.api.TerminateJobObject(self.job, 1)
        elif self.session_id is not None:
            deadline = time.monotonic() + timing.CLEANUP_SECONDS
            while members := unix_session_members(self.session_id):
                for pid, start_time in members.items():
                    kill_unix_member(pid, start_time, self.session_id)
                if time.monotonic() >= deadline:
                    if unix_session_members(self.session_id):
                        raise RuntimeError('Owned Linux session did not exit after termination')
                    break
                time.sleep(.02)
            self.session_id = None

    def close(self):
        if self.job:
            self.api.CloseHandle(self.job)
            self.job = None
        elif self.process and os.name != 'nt':
            self.kill()

    def __enter__(self):
        return self

    def __exit__(self, kind, primary, traceback):
        try:
            self.close()
        except BaseException:
            if primary is None: raise
            primary.add_note("Owned process tree cleanup did not complete")
