#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Mirror the Go parent's STARTUPINFO flags only in disposable startup diagnosis."""
import os
import json
import time
from pathlib import Path
import subprocess
import sys


def launch(args,*,normal_priority=False):
    if os.name!='nt':raise RuntimeError('native launch comparison requires Windows')
    info=subprocess.STARTUPINFO()
    info.dwFlags|=subprocess.STARTF_USESHOWWINDOW
    info.wShowWindow=0
    # The enclosing ProcessTree already owns this interpreter before it runs.
    # Windows job inheritance owns the GUI and every browser child. The caller
    # retains the original 18-second limit and requires natural zero before close.
    if normal_priority:
        receipt={'kind':'native-priority-control','wrapper_pid':os.getpid(),
                 'requested_gui_priority_class':32,'valid':False}
        try:
            from windows_native_resources import api
            selected=api();parent=selected.GetPriorityClass(selected.GetCurrentProcess())
        except Exception:
            parent=0
        receipt['wrapper_priority_available']=bool(parent)
        if parent:receipt['wrapper_priority_class']=parent
        if parent!=16384:
            receipt['reason']='expected-below-normal-wrapper-unavailable'
            print(json.dumps(receipt,sort_keys=True),file=sys.stderr,flush=True)
            return 78
        # Change only this actual GUI creation call. Keep the wrapper, CLI load,
        # owned job and the browser's later class choices unchanged.
        try:
            child=subprocess.Popen(args,stdin=subprocess.DEVNULL,
                creationflags=subprocess.CREATE_NO_WINDOW|subprocess.NORMAL_PRIORITY_CLASS,startupinfo=info)
        except OSError:
            receipt['reason']='gui-create-unavailable'
            print(json.dumps(receipt,sort_keys=True),file=sys.stderr,flush=True)
            return 78
        with child:
            receipt.update(gui_pid=child.pid,probe_begin_unix_micros=time.time_ns()//1000)
            try:actual=selected.GetPriorityClass(int(child._handle))
            except Exception:actual=0
            receipt.update(gui_priority_available=bool(actual),probe_end_unix_micros=time.time_ns()//1000,valid=actual==32)
            if actual:receipt['gui_priority_class']=actual
            if actual!=32:receipt['reason']='normal-gui-priority-unavailable'
            print(json.dumps(receipt,sort_keys=True),file=sys.stderr,flush=True)
            code=child.wait()
            return code if code or receipt['valid'] else 78
    result=subprocess.run(args,stdin=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW,
                          startupinfo=info,check=False)
    return result.returncode


def main(args):
    import windows_package_pipeline as pipeline
    pipeline.m.disposable_runner()
    if len(args)!=3 or not Path(args[0]).is_absolute() or args[1]!='--exit-qualification-child':
        raise ValueError('native launch comparison requires an explicit Exit child')
    normal=os.environ.get('INSONIC_STARTUP_NORMAL')=='true'
    if normal and os.environ.get('INSONIC_STARTUP_LIVE')!='true':raise ValueError('normal priority control requires the live single-child experiment')
    return launch(args,normal_priority=normal)


if __name__=='__main__':sys.exit(main(sys.argv[1:]))
