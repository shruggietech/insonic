#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Mirror the Go parent's STARTUPINFO flags only in disposable startup diagnosis."""
import os
from pathlib import Path
import subprocess
import sys


def launch(args):
    if os.name!='nt':raise RuntimeError('native launch comparison requires Windows')
    info=subprocess.STARTUPINFO()
    info.dwFlags|=subprocess.STARTF_USESHOWWINDOW
    info.wShowWindow=0
    # The enclosing ProcessTree already owns this interpreter before it runs.
    # Windows job inheritance owns the GUI and every browser child. The caller
    # retains the original 18-second limit and requires natural zero before close.
    result=subprocess.run(args,stdin=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW,
                          startupinfo=info,check=False)
    return result.returncode


def main(args):
    import windows_package_pipeline as pipeline
    pipeline.m.disposable_runner()
    if len(args)!=3 or not Path(args[0]).is_absolute() or args[1]!='--exit-qualification-child':
        raise ValueError('native launch comparison requires an explicit Exit child')
    return launch(args)


if __name__=='__main__':sys.exit(main(sys.argv[1:]))
