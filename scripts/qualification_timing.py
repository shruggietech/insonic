# SPDX-License-Identifier: Apache-2.0
"""Qualification-only allowances, never ordinary product timeout defaults."""
import hashlib
import json
import os
from pathlib import Path
import time

POLICY_PATH = Path(__file__).resolve().parents[1] / 'internal/qualificationtiming/policy.json'


FIELDS = frozenset((
    'subprocess_seconds', 'cleanup_seconds', 'natural_release_seconds',
    'readiness_seconds', 'frontend_seconds', 'playback_seconds',
    'help_seconds', 'help_watchdog_seconds', 'gui_seconds',
    'gui_watchdog_seconds', 'exit_child_seconds', 'exit_watchdog_seconds',
    'exit_matrix_seconds', 'native_ui_seconds', 'native_action_seconds',
    'admission_seconds', 'durable_work_seconds', 'fixture_idle_seconds',
    'first_use_command_seconds', 'first_use_seconds', 'installer_transaction_seconds',
    'installer_ui_seconds', 'installer_worker_seconds', 'installer_outer_seconds',
    'artifact_source_seconds', 'artifact_producer_seconds', 'request_seconds',
    'source_build_seconds', 'source_download_seconds', 'source_transfer_seconds',
    'docs_browser_seconds', 'docs_renderer_seconds', 'capture_seconds',
    'capture_phase_seconds', 'build_seconds', 'real_processing_seconds',
    'journey_seconds', 'service_seconds', 'git_seconds',
    'go_test_seconds', 'go_test_process_seconds', 'dependency_install_seconds',
))


def validate_policy(value):
    if (not isinstance(value, dict) or set(value) != FIELDS
            or any(type(seconds) is not int or not 0 < seconds <= 86400 for seconds in value.values())):
        raise ValueError('Invalid qualification timing policy')
    required = {'subprocess_seconds': 600, 'cleanup_seconds': 30, 'help_seconds': 180,
                'gui_seconds': 600, 'exit_child_seconds': 120, 'exit_matrix_seconds': 1800,
                'installer_ui_seconds': 3000, 'installer_worker_seconds': 3300,
                'installer_outer_seconds': 3360, 'first_use_command_seconds': 120,
                'first_use_seconds': 660}
    if any(value.get(key, 0) < minimum for key, minimum in required.items()):
        raise ValueError('Qualification timing policy is missing required allowances')
    if (value['installer_outer_seconds'] <= value['installer_worker_seconds']
            or value['installer_worker_seconds'] <= value['installer_ui_seconds']
            or value['first_use_seconds'] < 5 * value['first_use_command_seconds'] + value['cleanup_seconds']):
        raise ValueError('Qualification timing parent is shorter than its children')
    for child in ('help', 'gui', 'exit'):
        execution = value['exit_child_seconds'] if child == 'exit' else value[child + '_seconds']
        if value.get(child + '_watchdog_seconds', execution) + value['cleanup_seconds'] > execution:
            raise ValueError('Qualification watchdog leaves no completion allowance')
    if (value['go_test_process_seconds'] < value['go_test_seconds'] + value['cleanup_seconds']
            or value['docs_browser_seconds'] < value['docs_renderer_seconds'] + value['cleanup_seconds']):
        raise ValueError('Qualification process allowance omits teardown')
    return value


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        if key in value: raise ValueError('Duplicate qualification timing policy key')
        value[key] = item
    return value


def load_policy(path):
    return validate_policy(json.loads(path.read_text(encoding='utf-8'), object_pairs_hook=unique_object))


POLICY = load_policy(POLICY_PATH)
globals().update({key.upper(): value for key, value in POLICY.items()})


def phase_bound(seconds):
    """Cap a child against the caller's original epoch deadline without resetting it."""
    selected = os.environ.get('INSONIC_QUALIFICATION_DEADLINE')
    if selected is None:
        return seconds
    try:
        remaining = (int(selected) - time.time() * 1000) / 1000
    except ValueError:
        raise ValueError('Invalid original qualification deadline') from None
    if remaining <= 0:
        raise TimeoutError('Original qualification deadline exceeded')
    return min(seconds, remaining)


def source_identity():
    return {'policy_sha256': hashlib.sha256(POLICY_PATH.read_bytes()).hexdigest(),
            'loader_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
