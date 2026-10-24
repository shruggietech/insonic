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

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing

ROOT = Path(__file__).resolve().parent.parent


def join_inputs(tasks):
    with ThreadPoolExecutor(max_workers=4) as workers:
        # Executor shutdown joins every worker even when a verifier fails.
        return list(workers.map(lambda task: task(), tasks))


def run(arguments, *, timeout=timing.SUBPROCESS_SECONDS):
    remaining = (int(os.environ['INSONIC_QUALIFICATION_DEADLINE']) - time.time() * 1000) / 1000
    if remaining <= 0 or remaining > QUALIFICATION_LIMIT_SECONDS: raise ValueError('invalid original qualification deadline')
    with ProcessTree([str(arg) for arg in arguments], cwd=ROOT, env=os.environ.copy()) as tree:
        try:
            output, error = tree.process.communicate(timeout=min(timeout, remaining))
        except BaseException:
            tree.kill()
            tree.process.communicate(timeout=timing.CLEANUP_SECONDS)
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
    # Bound each request/retry inside this stage and the original qualification deadline.
    # Keep advisory auditing enabled, with bounded request/retry time inside that bound.
    # npm can issue a bulk audit and a fallback, each with three attempts.
    request_seconds = min(timing.REQUEST_SECONDS, (timing.DEPENDENCY_INSTALL_SECONDS - 2 * timing.CLEANUP_SECONDS) // 6)
    npm_limits = ['--audit=true', '--fetch-timeout=' + str(request_seconds * 1000), '--fetch-retries=2',
                  '--fetch-retry-mintimeout=1000', '--fetch-retry-maxtimeout=' + str(timing.CLEANUP_SECONDS * 1000)]
    def install(prefix):
        return run([node, npm, '--prefix', prefix, 'ci', '--ignore-scripts', *npm_limits], timeout=timing.DEPENDENCY_INSTALL_SECONDS)
    values = join_inputs([
        source,
        lambda: run([sys.executable, ROOT / 'scripts/go-dependency-transfer.py', 'restore']),
        lambda: install('desktop/frontend'),
        lambda: install('site'),
    ])
    for value in values: print(value)
    print('All independent CI inputs verified/installed; product compilation follows.')


if __name__ == '__main__': main()
