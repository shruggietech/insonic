# SPDX-License-Identifier: Apache-2.0
"""Hidden qualification children with descendant lifetime ownership."""
import os
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


class ProcessTree:
    def __init__(self, args, *, cwd, env, allow_detached=False):
        self.job = None
        self.process = None
        args = selected_executable(args, cwd, env)
        if os.name == 'nt':
            self._windows_job(allow_detached)
        try:
            self.process = subprocess.Popen(args, cwd=cwd, env=env, stdin=subprocess.DEVNULL,
                                            stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                            start_new_session=os.name != 'nt',
                                            creationflags=0x08000004 if os.name == 'nt' else 0)
            if self.job:
                self._assign_and_resume()
        except BaseException:
            if self.process:
                self.process.kill()
                self.process.communicate()
            self.close()
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

    def kill(self):
        if self.job:
            self.api.TerminateJobObject(self.job, 1)
        elif self.process:
            try:
                os.killpg(self.process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass

    def close(self):
        if self.job:
            self.api.CloseHandle(self.job)
            self.job = None
        elif self.process and os.name != 'nt':
            self.kill()

    def __enter__(self):
        return self

    def __exit__(self, *ignored):
        self.close()
