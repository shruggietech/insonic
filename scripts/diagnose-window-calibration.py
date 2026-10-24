#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""One sequential fixture-deadline comparison, never candidate acceptance."""
import json
import os
from pathlib import Path
from ci_timing_policy import QUALIFICATION_TARGET_SECONDS, QUALIFICATION_LIMIT_SECONDS, FINALIZATION_RESERVE_SECONDS
import re
import subprocess
import time

import windows_package_pipeline as pipeline
from process_tree import ProcessTree
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing

ROOT = pipeline.ROOT
EVIDENCE = ROOT / 'build/window-calibration/evidence'


def parse_observations(output, error, code, seconds, timed_out=False):
    children = []
    failures = []
    for line in error.decode('utf-8', errors='replace').splitlines():
        try:
            value = json.loads(line)
        except ValueError:
            continue
        if not isinstance(value, dict):
            continue
        if value.get('kind') in ('native-exit-owned-lifecycle', 'native-exit-child-failure'):
            records = []
            for nested in value.get('stderr', '').splitlines():
                try:
                    record = json.loads(nested)
                    if isinstance(record, dict):
                        records.append(record)
                except ValueError:
                    pass
            selected = failures if value.get('kind') == 'native-exit-child-failure' else children
            selected.append({'action': value['action'], 'workspace': value.get('workspace'),
                             'exit_code': value.get('exit_code'), 'truncated': value.get('truncated'),
                             'descendants': value.get('descendants', value.get('owned_descendants')), 'records': records, 'failure_stage':value.get('stage')})
    try:
        receipt = json.loads(output)
    except ValueError:
        receipt = None
    joins = [record for child in children + failures for record in child['records'] if record.get('kind') == 'native-ui-join']
    desktops = [record for child in children for record in child['records'] if record.get('kind') == 'native-desktop-observation']
    environments = [record for child in children for record in child['records'] if record.get('kind') == 'native-window-environment']
    native_passed = (not timed_out and code == 0 and isinstance(receipt, dict)
                     and receipt.get('native_exit') == 'passed' and receipt.get('window_join_seconds') == seconds)
    if native_passed:
        if failures:
            raise ValueError('passing comparison contains a failed native child')
        expected = [(action, workspace) for workspace in (False, True) for action in ('menu', 'close', 'alt-f4') for _ in range(2)]
        actual = [(child['action'], child['workspace']) for child in children]
        if actual != expected or any(child['exit_code'] != 0 or child['truncated'] is not False
                                     or not isinstance(child['descendants'], dict)
                                     or child['descendants'].get('released') is not True
                                     or child['descendants'].get('remaining') != 0 for child in children):
            raise ValueError('passing comparison lacks the complete twelve-case natural-release sequence')
        if len(joins) != 24 or any(record.get('result') != 'completed' or record.get('returned_ns', 0) <= 0
                                   or record.get('budget_ns') != seconds * 1_000_000_000 for record in joins):
            raise ValueError('passing comparison lacks actual per-operation completion observations')
        if len(environments) != 12 or sum(record.get('stage') == 'before-native-input' for record in desktops) != 12:
            raise ValueError('passing comparison lacks the created-environment observations')
    return {'allowance_seconds': seconds, 'native_passed': native_passed, 'timed_out': timed_out,
            'exit_code': code, 'executed_lifecycle_receipts': len(children), 'receipt': receipt,
            'joins': joins, 'desktops': desktops, 'environments': environments, 'children': children, 'failures':failures,
            'max_join_completion_seconds': max((r.get('returned_ns', 0) / 1e9 for r in joins), default=None)}


def compare(run_arm):
    # Retain a failed baseline. Each arm itself stops at its first native failure.
    return [run_arm(2, 'baseline'), run_arm(10, 'comparison')]


def cohort_environment(base_env, temporary, seconds):
    env = os.environ.copy()
    env.update(base_env)  # package_environment is a delta, as in package.child.
    env.update(TEMP=str(temporary), TMP=str(temporary),
               INSONIC_NATIVE_WINDOW_JOIN_SECONDS=str(seconds),
               INSONIC_PROCESS_STARTUP_DIAGNOSTICS='1')
    return env


def run_arm(package, gui, directory, base_env, seconds, label):
    evidence = EVIDENCE / label
    evidence.mkdir()
    temporary = evidence / 'fixture-temp'
    temporary.mkdir()
    env = cohort_environment(base_env, temporary, seconds)
    before = set((ROOT / 'build/native').glob('native-exit-child-*-receipt.json'))
    started = time.monotonic()
    timed_out = False
    with ProcessTree([str(gui), '--exit-qualification'], cwd=directory, env=env, allow_detached=True) as tree:
        try:
            output, error = tree.process.communicate(timeout=timing.GUI_SECONDS)
        except subprocess.TimeoutExpired:
            timed_out = True
            tree.kill()  # Failed-arm cleanup is never natural-release acceptance.
            output, error = tree.process.communicate(timeout=timing.CLEANUP_SECONDS)
        code = tree.process.returncode
        remaining = max(0, 100 - (time.monotonic() - started))
        natural_tree_release = code == 0 and not timed_out and tree.wait_for_descendants(remaining)
        quiesced = True
        if not natural_tree_release:
            tree.kill()
            quiesced = tree.wait_for_descendants(timing.CLEANUP_SECONDS)
    (evidence / 'stdout.log').write_bytes(output)
    (evidence / 'stderr.log').write_bytes(error)
    import shutil
    for path in set((ROOT / 'build/native').glob('native-exit-child-*-receipt.json')) - before:
        shutil.copyfile(path, evidence / path.name)
    result = parse_observations(output, error, code, seconds, timed_out)
    result.update(label=label, elapsed_seconds=time.monotonic()-started,
                  natural_matrix_tree_release=natural_tree_release,
                  failed_arm_cleanup=not natural_tree_release, owned_job_quiesced=quiesced)
    if result['native_passed'] and not natural_tree_release:
        result['native_passed'] = False
    pipeline.write(evidence / 'result.json', result)
    if not quiesced:
        raise RuntimeError('failed cohort cleanup did not quiesce its owned job; evidence retained, comparison cannot proceed')
    return result


def main():
    pipeline.m.disposable_runner()
    EVIDENCE.mkdir(parents=True, exist_ok=False)
    run = os.environ.get('CALIBRATION_RUN', '')
    attempt = os.environ.get('CALIBRATION_ATTEMPT', '')
    revision = os.environ.get('SOURCE_REVISION', '')
    if not re.fullmatch('[1-9][0-9]*', run) or not re.fullmatch('[1-9][0-9]*', attempt) or not re.fullmatch('[0-9a-f]{40}', revision):
        raise ValueError('comparison requires explicit run, attempt and source revision')
    base, receipt, identity = pipeline.restore_inputs()
    package = pipeline.load('window_calibration_package', 'package-desktop.py')
    archive = ROOT / 'build/windows-desktop-input' / receipt['archive']
    selected_source = ROOT / 'build/calibration-source'
    selected_revision = package.child(['git','-C',selected_source,'rev-parse','HEAD']).decode().strip()
    if selected_revision != revision:
        raise ValueError('candidate checkout differs from the selected assembly source')
    package.verify_archive_receipt(archive, receipt, expected_source_root=selected_source)
    location = EVIDENCE.parent / 'relocated package with spaces'
    package.extract_package(archive, location)
    directory = package.path_inside(location, receipt['inventory_root'])
    inventory = pipeline.read(package.path_inside(location, receipt['inventory_path']))
    package.verify_inventory(directory, inventory)
    gui = directory / 'insonic-desktop.exe'
    metadata = {'kind': 'native-window-calibration', 'candidate_acceptance': False,
                'source_run': run, 'source_attempt': attempt, 'revision': revision,
                'workflow_revision': os.environ.get('GITHUB_WORKFLOW_SHA'),
                'diagnostic_checkout_revision': package.child(['git','rev-parse','HEAD']).decode().strip(),
                'candidate_checkout_revision': selected_revision,
                'diagnostic_run': os.environ.get('GITHUB_RUN_ID'),
                'diagnostic_attempt': os.environ.get('GITHUB_RUN_ATTEMPT'),
                'runner': {key: os.environ.get(key) for key in ('RUNNER_OS','RUNNER_ARCH','ImageOS','ImageVersion')},
                'identity': identity, 'executable_sha256': package.sha(gui),
                'package_sha256': package.sha(archive), 'dependencies': inventory['dependencies'],
                'outer_limits_seconds': {'window_search':5,'startup_readiness':8,'menu_readiness':8,
                                         'watchdog':12,'child_capture_release':18,'matrix':100,'qualification_target':QUALIFICATION_TARGET_SECONDS,
                                         'qualification':QUALIFICATION_LIMIT_SECONDS,'finalization_reserve':FINALIZATION_RESERVE_SECONDS},
                'differences_from_ordinary_archive': ['No concurrent CLI workload in either diagnostic arm; actual ordinary archive observations are retained separately.'],
                'runtime_environment_selection': {key: os.environ.get(key) for key in
                    ('WEBVIEW2_BROWSER_EXECUTABLE_FOLDER','WEBVIEW2_RELEASE_CHANNEL_PREFERENCE','WEBVIEW2_RELEASE_CHANNELS')}}
    pipeline.write(EVIDENCE / 'identity.json', metadata)
    env = package.package_environment()
    env['INSONIC_DESKTOP_FIXTURE_DIRECTORY'] = str(ROOT / 'tests/fixtures/media')
    results = compare(lambda seconds, label: run_arm(package, gui, directory, env, seconds, label))
    summary = {**metadata, 'arms': [{key:value for key,value in arm.items()
                                  if key not in ('children','joins','desktops','environments')} for arm in results]}
    pipeline.write(EVIDENCE / 'comparison.json', summary)
    print(json.dumps(summary))
    # A failed baseline stays recorded. Only the comparison arm is the diagnostic
    # question here; no diagnostic result is a candidate scope proof.
    return 0 if results[1]['native_passed'] else 1


if __name__ == '__main__':
    raise SystemExit(main())
