#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Join independent CI input verification/installations before product builds."""
from concurrent.futures import ThreadPoolExecutor
import os
from pathlib import Path
import shutil
import sys
import time

from process_tree import ProcessTree
from ci_timing_policy import QUALIFICATION_LIMIT_SECONDS

ROOT = Path(__file__).resolve().parent.parent


def join_inputs(tasks):
    with ThreadPoolExecutor(max_workers=4) as workers:
        # Executor shutdown joins every worker even when a verifier fails.
        return list(workers.map(lambda task: task(), tasks))


def run(arguments):
    remaining = (int(os.environ['INSONIC_QUALIFICATION_DEADLINE']) - time.time() * 1000) / 1000
    if remaining <= 0 or remaining > QUALIFICATION_LIMIT_SECONDS: raise ValueError('invalid original qualification deadline')
    with ProcessTree([str(arg) for arg in arguments], cwd=ROOT, env=os.environ.copy()) as tree:
        try:
            output, error = tree.process.communicate(timeout=min(180, remaining))
        except BaseException:
            tree.kill()
            tree.process.communicate(timeout=5)
            raise
        if tree.process.returncode:
            raise RuntimeError('CI package input failed: ' + (output + error).decode(errors='replace')[-8000:])
        return output.decode(errors='replace')[-8000:]


def main():
    node = shutil.which('node')
    if not node: raise ValueError('selected Node runtime is missing')
    npm = Path(node).resolve().parent / 'node_modules/npm/bin/npm-cli.js'
    if not npm.is_file(): raise ValueError('selected Node npm CLI is missing')
    revision = os.environ['SOURCE_REVISION']
    def source():
        run([sys.executable, ROOT / 'scripts/media-transfer.py', 'restore', '--stage', 'complete', '--revision', revision,
             ROOT / 'build/transfer/media-source.tar.gz'])
        return run([sys.executable, ROOT / 'scripts/media-tools.py', '--prepare-only'])
    # npm's default five-minute request timeout exceeds this stage's 180-second bound.
    # Keep advisory auditing enabled, with bounded request/retry time inside that bound.
    npm_limits = ['--audit=true', '--fetch-timeout=15000', '--fetch-retries=2',
                  '--fetch-retry-mintimeout=1000', '--fetch-retry-maxtimeout=5000']
    def install(prefix):
        return run([node, npm, '--prefix', prefix, 'ci', '--ignore-scripts', *npm_limits])
    values = join_inputs([
        source,
        lambda: run([sys.executable, ROOT / 'scripts/go-dependency-transfer.py', 'restore']),
        lambda: install('desktop/frontend'),
        lambda: install('site'),
    ])
    for value in values: print(value)
    print('All independent CI inputs verified/installed; product compilation follows.')


if __name__ == '__main__': main()
