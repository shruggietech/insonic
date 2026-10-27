# SPDX-License-Identifier: Apache-2.0
"""Owned headless terminals for qualification, never the operator's console."""
import codecs
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import threading
import time

import qualification_timing as bounds
from process_tree import ProcessTree


CAPTURE_LIMIT = 65536
CONTROL = re.compile(r'\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[=>])')


def normalize_terminal(raw, *, final=True):
    try:
        text = codecs.getincrementaldecoder('utf-8')('strict').decode(raw, final=final)
    except UnicodeError as error:
        raise ValueError('Terminal output is not UTF-8') from error
    text = CONTROL.sub('', text)
    if '\x1b' in text:
        # A snapshot may end partway through a transport sequence. The complete
        # final capture must parse; never discard an unknown complete escape.
        last = text.rfind('\x1b')
        suffix = text[last:]
        if not final and re.fullmatch(r'\x1b(?:\[[0-?]*[ -/]*|\][^\x07\x1b]*)?', suffix):
            text = text[:last]
        else:
            raise ValueError('Unknown or incomplete terminal transport control')
    return text.replace('\r\n', '\n').replace('\r', '')


class Terminal:
    def __init__(self, argv, *, cwd, env, deadline=None, owner=None):
        self.started = time.monotonic()
        self.bound = bounds.phase_bound(bounds.SUBPROCESS_SECONDS)
        self.deadline = self.started + self.bound
        if deadline is not None:
            self.deadline = min(self.deadline, deadline)
            self.bound = self.deadline - self.started
            if self.bound <= 0:
                raise TimeoutError('Terminal fixture deadline exceeded before launch')
        self.condition = threading.Condition()
        self.buffer = bytearray()
        self.overflow = False
        self.read_error = None
        self.reader = None
        self.input = self.output = self.hpc = None
        self.cleaned = False
        self.owner = owner if owner is not None else ProcessTree.__new__(ProcessTree)
        self.owner.job = self.owner.process = self.owner.session_id = None
        self.kind = 'conpty' if os.name == 'nt' else 'linux-pty'
        try:
            if os.name == 'nt':
                self._windows(argv, cwd, env)
            elif sys.platform == 'linux':
                self._linux(argv, cwd, env)
            else:
                raise RuntimeError('No elected terminal transport for this platform')
        except BaseException as primary:
            try:
                self._cleanup(force=True)
            except BaseException as error:
                primary.add_note('Terminal launch cleanup failed: ' + type(error).__name__)
            raise

    def remaining(self):
        remaining = min(self.deadline - time.monotonic(), bounds.phase_bound(bounds.SUBPROCESS_SECONDS))
        if remaining <= 0:
            raise TimeoutError('Terminal fixture deadline exceeded')
        return remaining

    def _drain(self):
        try:
            while True:
                if os.name == 'nt':
                    c = self.owner.ctypes
                    chunk = c.create_string_buffer(4096)
                    count = c.c_ulong()
                    if not self.owner.api.ReadFile(self.output, chunk, 4096, c.byref(count), None):
                        error = c.get_last_error()
                        if error in (109, 232):
                            break
                        raise c.WinError(error)
                    data = chunk.raw[:count.value]
                else:
                    try:
                        data = os.read(self.output, 4096)
                    except OSError as error:
                        if error.errno == 5:  # Linux PTY slave closed.
                            break
                        raise
                if not data:
                    break
                with self.condition:
                    available = CAPTURE_LIMIT - len(self.buffer)
                    self.buffer.extend(data[:available])
                    self.overflow |= len(data) > available
                    self.condition.notify_all()
                # Keep draining after overflow so teardown cannot deadlock.
        except BaseException as error:
            self.read_error = error
        finally:
            with self.condition:
                self.condition.notify_all()

    def _start_reader(self):
        self.reader = threading.Thread(target=self._drain, name='owned-terminal-reader', daemon=True)
        self.reader.start()

    def _linux(self, argv, cwd, env):
        import termios
        master, slave = os.openpty()
        self.input = self.output = master
        try:
            settings = termios.tcgetattr(slave)
            settings[3] |= termios.ISIG
            settings[6][termios.VINTR] = b'\x03'
            termios.tcsetattr(slave, termios.TCSANOW, settings)
            # Fresh interpreter establishes the controlling terminal before exec.
            # No preexec/fork callback runs in this multithreaded parent.
            self.owner.process = subprocess.Popen(
                [sys.executable, str(Path(__file__).resolve()), '--exec', json.dumps(list(map(os.fspath, argv)))],
                cwd=cwd, env=env, stdin=slave, stdout=slave, stderr=slave, start_new_session=True)
            self.owner.session_id = self.owner.process.pid
        finally:
            os.close(slave)
        self._start_reader()

    def _windows(self, argv, cwd, env):
        import ctypes as c
        from ctypes import wintypes as w
        self.owner._windows_job(False)
        api = self.owner.api

        class Coord(c.Structure):
            _fields_ = [('x', c.c_short), ('y', c.c_short)]

        class Startup(c.Structure):
            _fields_ = [('cb', w.DWORD), ('reserved', w.LPWSTR), ('desktop', w.LPWSTR), ('title', w.LPWSTR),
                        *[(name, w.DWORD) for name in ['x', 'y', 'xsize', 'ysize', 'xchars', 'ychars', 'fill', 'flags']],
                        ('show', w.WORD), ('reserved_size', w.WORD), ('reserved_bytes', c.c_void_p),
                        ('stdin', w.HANDLE), ('stdout', w.HANDLE), ('stderr', w.HANDLE)]

        class ExtendedStartup(c.Structure):
            _fields_ = [('startup', Startup), ('attributes', c.c_void_p)]

        class Information(c.Structure):
            _fields_ = [('process', w.HANDLE), ('thread', w.HANDLE), ('pid', w.DWORD), ('tid', w.DWORD)]

        signatures = {
            'CreatePipe': ([c.POINTER(w.HANDLE), c.POINTER(w.HANDLE), c.c_void_p, w.DWORD], w.BOOL),
            'CreatePseudoConsole': ([Coord, w.HANDLE, w.HANDLE, w.DWORD, c.POINTER(w.HANDLE)], c.c_long),
            'ClosePseudoConsole': ([w.HANDLE], None),
            'InitializeProcThreadAttributeList': ([c.c_void_p, w.DWORD, w.DWORD, c.POINTER(c.c_size_t)], w.BOOL),
            'UpdateProcThreadAttribute': ([c.c_void_p, w.DWORD, c.c_size_t, c.c_void_p, c.c_size_t, c.c_void_p, c.c_void_p], w.BOOL),
            'DeleteProcThreadAttributeList': ([c.c_void_p], None),
            'CreateProcessW': ([w.LPCWSTR, w.LPWSTR, c.c_void_p, c.c_void_p, w.BOOL, w.DWORD, c.c_void_p, w.LPCWSTR, c.c_void_p, c.POINTER(Information)], w.BOOL),
            'ReadFile': ([w.HANDLE, c.c_void_p, w.DWORD, c.POINTER(w.DWORD), c.c_void_p], w.BOOL),
            'WriteFile': ([w.HANDLE, c.c_void_p, w.DWORD, c.POINTER(w.DWORD), c.c_void_p], w.BOOL),
            'WaitForSingleObject': ([w.HANDLE, w.DWORD], w.DWORD),
            'GetExitCodeProcess': ([w.HANDLE, c.POINTER(w.DWORD)], w.BOOL),
            'TerminateProcess': ([w.HANDLE, w.UINT], w.BOOL),
        }
        for name, (args, result) in signatures.items():
            getattr(api, name).argtypes, getattr(api, name).restype = args, result

        def checked(value):
            if not value:
                raise c.WinError(c.get_last_error())
            return value

        child_input, child_output = w.HANDLE(), w.HANDLE()
        parent_input, parent_output, hpc = w.HANDLE(), w.HANDLE(), w.HANDLE()
        attributes = None
        try:
            checked(api.CreatePipe(c.byref(child_input), c.byref(parent_input), None, 0))
            self.input = parent_input.value
            checked(api.CreatePipe(c.byref(parent_output), c.byref(child_output), None, 0))
            self.output = parent_output.value
            # ConPTY is the hidden console host; no visible host is ever created.
            # Do not use NEW_PROCESS_GROUP (disables Ctrl+C) or INHERIT_CURSOR.
            result = api.CreatePseudoConsole(Coord(4096, 128), child_input, child_output, 0, c.byref(hpc))
            if result != 0:
                raise OSError('CreatePseudoConsole failed: ' + hex(result & 0xffffffff))
            self.hpc = hpc.value
            self._start_reader()
            size = c.c_size_t()
            api.InitializeProcThreadAttributeList(None, 1, 0, c.byref(size))
            storage = c.create_string_buffer(size.value)
            checked(api.InitializeProcThreadAttributeList(storage, 1, 0, c.byref(size)))
            attributes = storage
            checked(api.UpdateProcThreadAttribute(storage, 0, 0x20016, self.hpc, c.sizeof(w.HANDLE), None, None))
            startup = ExtendedStartup()
            startup.startup.cb = c.sizeof(startup)
            # ConPTY creates its own headless host, with no presentation window.
            # Do not inherit the qualifier's redirected standard handles. Invalid
            # handles let the attached console establish its own terminal handles.
            startup.startup.flags = 0x100
            startup.startup.stdin = startup.startup.stdout = startup.startup.stderr = c.c_void_p(-1).value
            startup.attributes = c.cast(storage, c.c_void_p)
            info = Information()
            command = c.create_unicode_buffer(subprocess.list2cmdline(list(map(os.fspath, argv))))
            environment = c.create_unicode_buffer('\0'.join(k + '=' + v for k, v in sorted(env.items(), key=lambda item: item[0].upper())) + '\0\0')
            # ConPTY supplies the hidden console. CREATE_NO_WINDOW removes its
            # terminal handles, so it must not be combined with this attachment.
            checked(api.CreateProcessW(os.fspath(argv[0]), command, None, None, False,
                0x00080000 | 0x00000400 | 0x00000004,
                environment, os.fspath(cwd), c.byref(startup), c.byref(info)))

            class Process:
                pid, handle, code = info.pid, info.process, None
                primary_thread = info.thread

                def poll(self):
                    if self.code is None:
                        status = api.WaitForSingleObject(self.handle, 0)
                        if status == 0xffffffff:
                            raise c.WinError(c.get_last_error())
                        if status == 0:
                            code = w.DWORD()
                            checked(api.GetExitCodeProcess(self.handle, c.byref(code)))
                            self.code = code.value
                    return self.code

            self.owner.process = Process()
            try:
                self.owner._assign_and_resume()
            except BaseException as primary:
                # Assignment can fail while the suspended child is still outside
                # the job. Kill and join that exact handle before releasing it.
                if (not api.TerminateProcess(info.process, 1)
                        or api.WaitForSingleObject(info.process, bounds.CLEANUP_SECONDS * 1000) != 0):
                    primary.add_note('Unassigned terminal process cleanup did not complete')
                raise
            finally:
                api.CloseHandle(info.thread)
        finally:
            if attributes is not None:
                api.DeleteProcThreadAttributeList(attributes)
            for handle in [child_input, child_output]:
                if handle.value:
                    api.CloseHandle(handle)

    def text(self, *, final=False):
        with self.condition:
            if self.overflow:
                raise ValueError('Terminal capture bound exceeded')
            raw = bytes(self.buffer)
        if self.read_error:
            raise RuntimeError('Terminal reader failed') from self.read_error
        return normalize_terminal(raw, final=final)

    def wait_for(self, token):
        while token not in self.text():
            if self.owner.process.poll() is not None:
                # Process exit can precede the reader's last buffered bytes.
                # Join/drain the owned transport before deciding observation failed.
                result = self.finish()
                if token in result['text']:
                    return
                raise ValueError('Terminal client exited before expected observation: ' + token
                                 + f" (exit {result['exit_code']}, output {result['text'][:2048]!r})")
            with self.condition:
                self.condition.wait(min(.02, self.remaining()))

    def interrupt(self):
        if os.name == 'nt':
            c = self.owner.ctypes
            count = c.c_ulong()
            if not self.owner.api.WriteFile(self.input, b'\x03', 1, c.byref(count), None) or count.value != 1:
                raise c.WinError(c.get_last_error())
        else:
            os.write(self.input, b'\x03')

    def finish(self):
        while self.owner.process.poll() is None:
            with self.condition:
                self.condition.wait(min(.02, self.remaining()))
        code = self.owner.process.poll()
        self._cleanup(force=False)
        self.remaining()
        return {'text': self.text(final=True), 'exit_code': code, 'natural_exit': True,
                'cleanup': self.cleaned, 'elapsed_seconds': time.monotonic() - self.started,
                'bound_seconds': self.bound, 'transport': self.kind}

    def _cleanup(self, *, force):
        if self.cleaned:
            return
        end = time.monotonic() + bounds.CLEANUP_SECONDS
        closer = None
        failures = []
        try:
            if force:
                try:
                    self.owner.kill()
                except BaseException as error:
                    failures.append(type(error).__name__)
            try:
                if not self.owner.wait_for_descendants(max(0, end - time.monotonic())):
                    failures.append('descendants did not release naturally')
            except BaseException as error:
                failures.append('descendant observation failed: ' + type(error).__name__)
            if os.name == 'nt' and self.hpc is not None:
                hpc, self.hpc = self.hpc, None
                def close_console():
                    try:
                        self.owner.api.ClosePseudoConsole(hpc)
                    except BaseException as error:
                        failures.append('console close failed: ' + type(error).__name__)
                closer = threading.Thread(target=close_console, daemon=True)
                closer.start()
                closer.join(max(0, end - time.monotonic()))
                if closer.is_alive():
                    failures.append('console close did not release')
            if self.reader:
                self.reader.join(max(0, end - time.monotonic()))
                if self.reader.is_alive():
                    failures.append('reader did not release')
        finally:
            # Even a failed join must release channel/process/job ownership.
            try:
                if os.name == 'nt':
                    for handle in [self.input, self.output, getattr(self.owner.process, 'handle', None)]:
                        if handle is not None and not self.owner.api.CloseHandle(handle):
                            failures.append('handle did not release')
                elif self.output is not None:
                    os.close(self.output)
                    if self.owner.process:
                        self.owner.process.wait(timeout=max(.001, end - time.monotonic()))
            finally:
                self.owner.close()
                self.input = self.output = None
                self.cleaned = True
        if failures:
            raise RuntimeError('Terminal cleanup failed: ' + ', '.join(failures))

    def __enter__(self):
        return self

    def __exit__(self, kind, primary, traceback):
        if not self.cleaned:
            try:
                self._cleanup(force=True)
            except BaseException as error:
                if primary is None:
                    raise
                primary.add_note('Terminal failure cleanup failed: ' + type(error).__name__)
            if primary is None:
                raise RuntimeError('Terminal client did not exit naturally')


if __name__ == '__main__':
    if sys.platform != 'linux' or len(sys.argv) != 3 or sys.argv[1] != '--exec':
        raise SystemExit('Only the Linux terminal bootstrap is directly executable')
    import fcntl
    import termios
    fcntl.ioctl(0, termios.TIOCSCTTY, 0)
    os.tcsetpgrp(0, os.getpgrp())
    command = json.loads(sys.argv[2])
    os.execve(command[0], command, os.environ)
