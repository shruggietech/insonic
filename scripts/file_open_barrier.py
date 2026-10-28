# SPDX-License-Identifier: Apache-2.0
"""External qualification-only source-open barriers; no product fault switch."""
import argparse
import json
import math
import os
from pathlib import Path
import queue
import stat
import sys
import threading
import time

from process_tree import ProcessTree
import qualification_timing as bounds


def validate_event(value, state, pid, mechanism):
    fields = {'state', 'pid', 'mechanism'} | ({'ack_required', 'break_deadline'} if state == 'break' else set())
    if (not isinstance(value, dict) or value.get('state') != state
            or set(value) != fields
            or type(value.get('pid')) is not int or value['pid'] != pid
            or value.get('mechanism') != mechanism):
        raise ValueError('File barrier event identity differs')
    if state == 'break':
        end = value.get('break_deadline')
        if (value.get('ack_required') is not True or type(end) not in (int, float)
                or not math.isfinite(end) or end <= time.monotonic()):
            raise ValueError('File barrier break is not held')


class FileOpenBarrier:
    def __init__(self, path, *, cwd, env, deadline, process_factory=ProcessTree):
        self.path, self.cwd, self.env = Path(path), cwd, env
        self.deadline, self.factory = deadline, process_factory
        self.mechanism = {'win32': 'windows-rw-oplock', 'linux': 'linux-write-lease'}.get(sys.platform)
        self.owner = self.reader = None
        self.events = queue.Queue(maxsize=8)
        self.broken = None
        self.released = False

    def remaining(self):
        if type(self.deadline) not in (int, float) or not math.isfinite(self.deadline):
            raise ValueError('Invalid file barrier enclosing deadline')
        end = min(self.deadline, self.broken['break_deadline']) if self.broken else self.deadline
        value = min(end - time.monotonic(), bounds.phase_bound(bounds.READINESS_SECONDS))
        if value <= 0:
            raise TimeoutError('Owned file barrier deadline exceeded')
        return value

    def __enter__(self):
        self.remaining()
        info = self.path.lstat()
        if not self.mechanism or not stat.S_ISREG(info.st_mode) or self.path.is_symlink():
            raise ValueError('File barrier requires an owned regular supported-platform file')
        epoch = time.time() + self.remaining()
        try:
            self.owner = self.factory([sys.executable, str(Path(__file__).resolve()), '--hold', str(self.path),
                                       '--deadline', str(epoch)], cwd=self.cwd, env=self.env, input_pipe=True)
            self.owner.__enter__()
            process = self.owner.process

            def read():
                try:
                    while True:
                        line = process.stdout.readline(4097)
                        if not line:
                            break
                        if len(line) > 4096 or not line.endswith(b'\n'):
                            raise ValueError('File barrier event exceeds bound')
                        self.events.put_nowait(json.loads(line))
                except BaseException as error:
                    try: self.events.put_nowait(error)
                    except queue.Full: pass

            self.reader = threading.Thread(target=read, name='owned-file-barrier-reader', daemon=True)
            self.reader.start()
            self._event('armed')
            return self
        except BaseException:
            self._close(failed=True)
            raise

    def _event(self, state):
        while True:
            try:
                value = self.events.get(timeout=min(.05, self.remaining()))
            except queue.Empty:
                if self.owner.process.poll() is not None:
                    raise ValueError('File barrier helper exited before ' + state)
                continue
            if isinstance(value, BaseException):
                raise value
            validate_event(value, state, self.owner.process.pid, self.mechanism)
            return value

    def wait_break(self):
        self.broken = self._event('break')
        self.remaining()
        return self.broken

    def release(self):
        if self.owner is None:
            return
        self._close(failed=False)

    def _close(self, *, failed):
        if self.owner is None:
            return
        owner, self.owner = self.owner, None
        primary = None
        try:
            if owner.process.poll() is None:
                owner.process.stdin.write(b'release\n')
                owner.process.stdin.flush()
                owner.process.stdin.close()
            owner.process.wait(timeout=bounds.phase_bound(bounds.CLEANUP_SECONDS))
            if owner.process.returncode != 0:
                raise ValueError('File barrier helper failed')
        except BaseException as error:
            primary = error
            owner.kill()
            owner.process.wait(timeout=bounds.CLEANUP_SECONDS)
        finally:
            if self.reader is not None:
                self.reader.join(timeout=bounds.CLEANUP_SECONDS)
                if self.reader.is_alive():
                    primary = primary or RuntimeError('File barrier reader did not release')
            error_bytes = owner.process.stderr.read(65537)
            if len(error_bytes) > 65536:
                primary = primary or ValueError('File barrier diagnostics exceed bound')
            try:
                owner.__exit__(None, None, None)
            except BaseException as error:
                primary = primary or error
            for stream in (owner.process.stdin, owner.process.stdout, owner.process.stderr):
                if stream is not None:
                    stream.close()
            self.released = primary is None
        if primary is not None and not failed:
            raise primary

    def __exit__(self, kind, value, traceback):
        try:
            self._close(failed=False)
        except BaseException as error:
            if value is None:
                raise
            value.add_note('Owned file barrier cleanup failed: ' + type(error).__name__)


class WindowsBarrier:
    def __init__(self, path):
        import ctypes
        from ctypes import wintypes as w
        self.c, self.w = ctypes, w
        k = self.k = ctypes.WinDLL('kernel32', use_last_error=True)

        class Overlapped(ctypes.Structure):
            _fields_ = [('Internal', ctypes.c_size_t), ('InternalHigh', ctypes.c_size_t),
                        ('Offset', w.DWORD), ('OffsetHigh', w.DWORD), ('hEvent', w.HANDLE)]

        class Input(ctypes.Structure):
            _fields_ = [('version', w.WORD), ('length', w.WORD), ('level', w.DWORD), ('flags', w.DWORD)]

        class Output(ctypes.Structure):
            _fields_ = [('version', w.WORD), ('length', w.WORD), ('original', w.DWORD),
                        ('new', w.DWORD), ('flags', w.DWORD), ('access', w.DWORD), ('share', w.WORD)]

        k.CreateFileW.argtypes = [w.LPCWSTR, w.DWORD, w.DWORD, w.LPVOID, w.DWORD, w.DWORD, w.HANDLE]
        k.CreateFileW.restype = w.HANDLE
        k.CreateEventW.argtypes = [w.LPVOID, w.BOOL, w.BOOL, w.LPCWSTR]
        k.CreateEventW.restype = w.HANDLE
        k.DeviceIoControl.argtypes = [w.HANDLE, w.DWORD, w.LPVOID, w.DWORD, w.LPVOID, w.DWORD, ctypes.POINTER(w.DWORD), ctypes.POINTER(Overlapped)]
        k.DeviceIoControl.restype = w.BOOL
        k.WaitForSingleObject.argtypes = [w.HANDLE, w.DWORD]
        k.WaitForSingleObject.restype = w.DWORD
        k.GetOverlappedResult.argtypes = [w.HANDLE, ctypes.POINTER(Overlapped), ctypes.POINTER(w.DWORD), w.BOOL]
        k.GetOverlappedResult.restype = w.BOOL
        k.CancelIoEx.argtypes = [w.HANDLE, ctypes.POINTER(Overlapped)]
        k.CancelIoEx.restype = w.BOOL
        k.CloseHandle.argtypes = [w.HANDLE]
        k.CloseHandle.restype = w.BOOL
        self.handle = self.event = None
        self.pending = False
        self.observed = False
        try:
            self.event = k.CreateEventW(None, True, False, None)
            self.handle = k.CreateFileW(str(path), 0xc0000000, 7, None, 3, 0x40000000 | 0x00200000, None)
            if not self.event or self.handle == ctypes.c_void_p(-1).value:
                self.handle = None
                raise ctypes.WinError(ctypes.get_last_error())
            self.overlap, self.output = Overlapped(hEvent=self.event), Output()
            request, count = Input(1, ctypes.sizeof(Input), 5, 1), w.DWORD()
            result = k.DeviceIoControl(self.handle, 0x00090240, ctypes.byref(request), ctypes.sizeof(request),
                                       ctypes.byref(self.output), ctypes.sizeof(self.output), ctypes.byref(count), ctypes.byref(self.overlap))
            if result or ctypes.get_last_error() != 997:
                raise ValueError('RW oplock was not armed asynchronously')
            self.pending = True
        except BaseException:
            self.close()
            raise

    def broken(self):
        if self.observed:
            return True
        state = self.k.WaitForSingleObject(self.event, 0)
        if state == 258:
            return False
        if state != 0:
            raise self.c.WinError(self.c.get_last_error())
        count = self.w.DWORD()
        if not self.k.GetOverlappedResult(self.handle, self.c.byref(self.overlap), self.c.byref(count), False):
            raise self.c.WinError(self.c.get_last_error())
        self.pending = False
        if (count.value != self.c.sizeof(self.output) or self.output.version != 1
                or self.output.length != self.c.sizeof(self.output) or self.output.original != 5
                or not self.output.flags & 1 or self.output.new == 5):
            raise ValueError('RW oplock break does not require acknowledgement')
        self.observed = True
        return True

    def close(self):
        if self.handle:
            if self.pending:
                self.k.CancelIoEx(self.handle, self.c.byref(self.overlap))
                count = self.w.DWORD()
                # The event and OVERLAPPED remain alive until cancellation has drained.
                self.k.GetOverlappedResult(self.handle, self.c.byref(self.overlap), self.c.byref(count), True)
            self.k.CloseHandle(self.handle)
            self.handle = None
        if self.event:
            self.k.CloseHandle(self.event)
            self.event = None


class LinuxBarrier:
    def __init__(self, path):
        import fcntl
        import signal
        self.fcntl, self.signal, self.fd = fcntl, signal, None
        self.observed = False
        self.limit = int(Path('/proc/sys/fs/lease-break-time').read_text())
        if self.limit < 3:
            raise ValueError('Kernel file lease break allowance is insufficient')
        self.previous = signal.signal(signal.SIGIO, self._break)
        try:
            self.fd = os.open(path, os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC)
            info = os.fstat(self.fd)
            if not stat.S_ISREG(info.st_mode) or info.st_uid != os.geteuid():
                raise ValueError('File lease input is not an owned regular file')
            fcntl.fcntl(self.fd, fcntl.F_SETOWN, os.getpid())
            fcntl.fcntl(self.fd, fcntl.F_SETLEASE, fcntl.F_WRLCK)
        except BaseException:
            self.close()
            raise

    def _break(self, *_args):
        if not self.observed:
            self.observed = time.monotonic()

    def broken(self):
        return bool(self.observed)

    def close(self):
        if self.fd is not None:
            try:
                self.fcntl.fcntl(self.fd, self.fcntl.F_SETLEASE, self.fcntl.F_UNLCK)
            finally:
                os.close(self.fd)
                self.fd = None
        self.signal.signal(self.signal.SIGIO, self.previous)


def hold(path, deadline):
    if not math.isfinite(deadline):
        raise ValueError('Invalid file barrier parent deadline')
    end = time.monotonic() + min(deadline - time.time(), bounds.phase_bound(bounds.READINESS_SECONDS))
    if end <= time.monotonic():
        raise TimeoutError('File barrier parent deadline expired')
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or path.is_symlink():
        raise ValueError('File barrier input must be regular')
    barrier = WindowsBarrier(path) if os.name == 'nt' else LinuxBarrier(path)
    mechanism = 'windows-rw-oplock' if os.name == 'nt' else 'linux-write-lease'
    release = threading.Event()

    def control():
        # Exactly one bounded noninteractive command, with EOF also releasing ownership.
        sys.stdin.buffer.readline(32)
        release.set()

    thread = threading.Thread(target=control, name='owned-barrier-control', daemon=True)
    try:
        thread.start()
        print(json.dumps({'state': 'armed', 'pid': os.getpid(), 'mechanism': mechanism}), flush=True)
        sent = False
        while not release.wait(.01):
            if not sent and barrier.broken():
                # Finish before Linux's forced downgrade. Windows is bounded too.
                break_start = barrier.observed if os.name != 'nt' else time.monotonic()
                allowance = min(bounds.READINESS_SECONDS, getattr(barrier, 'limit', bounds.READINESS_SECONDS)) - 2
                end = min(end, break_start + allowance)
                print(json.dumps({'state': 'break', 'pid': os.getpid(), 'mechanism': mechanism,
                                  'ack_required': True, 'break_deadline': end}), flush=True)
                sent = True
            if time.monotonic() >= end:
                raise TimeoutError('Held file barrier expired without parent release')
    finally:
        barrier.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--hold', type=Path, required=True)
    parser.add_argument('--deadline', type=float, required=True)
    args = parser.parse_args()
    hold(args.hold, args.deadline)
