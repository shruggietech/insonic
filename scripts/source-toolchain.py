# SPDX-License-Identifier: Apache-2.0
"""Observe resolved build packages without treating them as a pinned closure."""
import argparse
import json
import os
from pathlib import Path
import platform
import subprocess
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing


def inventory():
    windows = os.name == 'nt'
    if windows:
        root = Path(os.environ['INSONIC_MSYS2_ROOT'])
        if not root.is_absolute():
            raise ValueError('source toolkit must be absolute')
        command = [str(root / 'usr/bin/pacman.exe'), '-Q']
    else:
        command = ['dpkg-query', '-W', '-f=${Package}\t${Version}\n']
    result = subprocess.run(command, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, check=True, timeout=timing.READINESS_SECONDS,
                            creationflags=subprocess.CREATE_NO_WINDOW if windows else 0)
    if len(result.stdout) > 4 * 1024 * 1024:
        raise ValueError('toolchain package inventory exceeds bound')
    return {'format': 'insonic-source-toolchain-observation-v1',
            'platform': platform.system(), 'machine': platform.machine(),
            'revision': os.environ['SOURCE_REVISION'],
            'run': os.environ['GITHUB_RUN_ID'], 'attempt': os.environ['GITHUB_RUN_ATTEMPT'],
            'packages': result.stdout.decode('utf-8').splitlines(),
            'authority': 'resolved package observation, not a fully pinned toolchain contract'}


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', type=Path, required=True)
    output = parser.parse_args().output
    value = inventory()
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8', newline='\n')
