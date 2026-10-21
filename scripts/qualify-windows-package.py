#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Overlap exact-payload ZIP/MSI qualification on acknowledged disposable CI."""
from concurrent.futures import ThreadPoolExecutor
from copy import deepcopy
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
from process_tree import ProcessTree
from ci_timing_policy import QUALIFICATION_LIMIT_SECONDS
import windows_msi as m

ROOT = m.ROOT


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, ROOT / 'scripts' / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def run_child(label, arguments):
    # Window-station changes belong to the MSI child, never either parent thread.
    # Keep the existing runtime breakaway election and hidden tree ownership.
    with ProcessTree(arguments, cwd=ROOT, env=dict(os.environ), allow_detached=True) as tree:
        try:
            output, error = tree.process.communicate(timeout=QUALIFICATION_LIMIT_SECONDS)
        except subprocess.TimeoutExpired:
            tree.kill()
            output, error = tree.process.communicate(timeout=5)
            sys.stdout.buffer.write(output)
            sys.stderr.buffer.write(error)
            raise
        if tree.process.returncode:
            sys.stdout.buffer.write(output)
            sys.stderr.buffer.write(error)
            raise RuntimeError(label + ' qualification child failed (' + str(tree.process.returncode) + ')')


def assert_immutable_receipt(initial, completed):
    # Qualification may add proof fields; it may not rewrite a single assembly
    # identity, including source, archive/inventory, signing or distribution.
    if any(key not in completed or completed[key] != value for key, value in initial.items()):
        raise ValueError('qualified ZIP changed an immutable assembly receipt field')


def qualify():
    m.disposable_runner()
    package = load('windows_parallel_package', 'package-desktop.py')
    release = load('windows_parallel_release', 'release.py')
    native = ROOT / 'build' / 'native'
    native.mkdir(parents=True, exist_ok=True)
    installer_receipt = native / 'windows-installer-receipt.json'
    if installer_receipt.is_symlink():
        raise ValueError('installer receipt is redirected')
    # A failed new attempt cannot leave an earlier eligible receipt behind.
    installer_receipt.unlink(missing_ok=True)
    archive, dirname, built = package.build('desktop')
    initial = deepcopy(built)
    if initial.get('source_dirty') is not False:
        raise ValueError('parallel native qualification requires clean source')
    snapshot = native / 'windows-package-input-receipt.json'
    package.write_json(snapshot, initial)
    snapshot_bytes = snapshot.read_bytes()
    payload = package.BUILD / dirname
    commands = [
        ('archive', [sys.executable, str(ROOT / 'scripts/package-desktop.py'), 'smoke', '--variant', 'desktop']),
        ('installer', [sys.executable, str(ROOT / 'scripts/package-msi.py'), 'provisional', '--payload', str(payload), '--receipt', str(snapshot)]),
    ]
    failures = []
    with ThreadPoolExecutor(max_workers=2) as workers:
        futures = [(label, workers.submit(run_child, label, arguments)) for label, arguments in commands]
        # Observe failures in declared order, but always join both owned children.
        for label, future in futures:
            try:
                future.result()
            except Exception as error:
                failures.append((label, error))
    if failures:
        raise RuntimeError('parallel native qualification failed: ' + ', '.join(label for label, _ in failures)) from failures[0][1]
    if snapshot.read_bytes() != snapshot_bytes:
        raise ValueError('immutable ZIP assembly receipt snapshot changed')
    completed = json.loads((native / 'package-desktop-receipt.json').read_text(encoding='utf-8'))
    assert_immutable_receipt(initial, completed)
    release.verify_qualification(completed)
    package.verify_archive_receipt(archive, completed)
    provisional = json.loads(installer_receipt.read_text(encoding='utf-8'))
    if (provisional.get('publication_eligible') is not False
            or provisional.get('payload_qualification') != 'pending'
            or provisional.get('payload_receipt') != initial):
        raise ValueError('MSI did not retain its exact provisional ZIP authority')
    if (provisional.get('payload_archive') != completed['archive']
            or provisional.get('payload_archive_sha256') != completed['archive_sha256']):
        raise ValueError('MSI qualified a different archive')
    final = deepcopy(provisional)
    final['payload_receipt'] = completed
    final['publication_eligible'] = True
    final['payload_qualification'] = 'passed'
    installer = package.BUILD / final['artifact']
    # Verify in memory first. Any failure leaves only the provisional receipt.
    m.verify_installer(installer, final)
    package.write_json(installer_receipt, final)
    return final


if __name__ == '__main__':
    if len(sys.argv) != 1:
        raise SystemExit('Usage: python scripts/qualify-windows-package.py')
    receipt = qualify()
    print(json.dumps({'artifact': receipt['artifact'], 'sha256': receipt['artifact_sha256'],
                      'payload_archive_sha256': receipt['payload_archive_sha256'],
                      'qualification': receipt['qualification'], 'publication_eligible': True}))
