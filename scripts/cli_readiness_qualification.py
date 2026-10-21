#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Installed readiness regressions with isolated, generated local fixtures."""
import hashlib
import json
from pathlib import Path
import uuid


def qualify(invoke, workspace, version, *, executable):
    workspace = Path(workspace)
    request = workspace / 'readiness input.json'

    def supplied(args, value, **options):
        request.write_text(json.dumps(value) + '\n', encoding='utf-8', newline='\n')
        return invoke([*args, '--input', request], **options)

    # Valid pinned placeholders are never invoked: incomplete offline archives
    # must fail before any installer/worker launch. Only this owned library is changed.
    executable = Path(executable).resolve()
    digest = hashlib.sha256(executable.read_bytes()).hexdigest()
    tools = {'kind': 'processing-tools', 'schema_version': version,
             'cueson': {'executable': str(executable), 'executable_sha256': digest},
             'processing': {'worker': {'path': str(executable), 'sha256': digest},
                            'ffmpeg': {'path': str(executable), 'sha256': digest, 'version': 'fixture'}}}
    (workspace / '.insonic' / 'processing-tools.json').write_text(json.dumps(tools) + '\n', encoding='utf-8', newline='\n')

    offline = workspace / 'offline fixtures'
    offline.mkdir()
    (offline / 'unrelated.txt').write_bytes(b'unrelated bytes')
    plan = supplied(['workers', 'plan'], {'role': 'recognition', 'mode': 'managed',
                    'offline_assets': str(offline)})['result']
    assets = plan['local_assets']
    if plan['download_bytes'] != 0 or assets['required_bytes'] != plan['profile_download_bytes'] or assets['verified_bytes'] != 0 or assets['missing_bytes'] != assets['required_bytes']:
        raise ValueError('Offline network/local budgets differ')
    expected = [plan['profile']['runtime'], plan['profile']['installer'], *plan['profile']['wheels']]
    if [e['asset']['name'] for e in assets['failures']] != [a['name'] for a in expected] or any(e['reason'] != 'asset_missing' for e in assets['failures']):
        raise ValueError('Offline plan lost declared logical file observations')
    # The first file has the wrong length, so no installer or engine can run.
    first = offline / expected[0]['name']
    first.write_bytes(b'tiny incorrect archive')
    changed = supplied(['workers', 'plan'], plan['election'])['result']
    if changed['local_assets']['failures'][0]['reason'] != 'asset_size':
        raise ValueError('Offline plan lost size-verification stage')
    human = supplied(['workers', 'plan'], plan['election'], machine=False)
    if expected[0]['name'] not in human:
        raise ValueError('Human plan omitted the failed logical asset')
    for reason in ['asset_size', 'asset_missing']:
        if reason == 'asset_missing':
            first.unlink()
        election = supplied(['workers', 'plan'], plan['election'])['result']['election']
        work = supplied(['workers', 'setup'], election)['result']
        failure = invoke(['work', 'wait', work['work_id']], expected=1 if reason == 'asset_size' else 4)
        if failure['error'].get('reason') != reason or failure['error'].get('asset', {}).get('name') != expected[0]['name']:
            raise ValueError('Setup failure lost its logical asset or verification reason')
        shown = invoke(['work', 'show', work['work_id']])['result']
        if shown['result']['cause'] != failure['error']:
            raise ValueError('Durable setup cause differs from returned failure')
        human = invoke(['work', 'wait', work['work_id']], expected=1 if reason == 'asset_size' else 4, machine=False)
        if expected[0]['name'] not in human or str(offline) in human:
            raise ValueError('Human setup failure omitted logical detail or disclosed a local path')

    missing_model = supplied(['models', 'prepare'], {'choice': 'whisper-tiny', 'local_directory': str(offline)}, expected=4)
    if missing_model['error'].get('reason') != 'asset_missing' or missing_model['error'].get('asset', {}).get('role') != 'model_file':
        raise ValueError('Missing local model role lost its generated observation')

    credential = str(uuid.uuid4())
    preview_input = {'choice': 'community-1', 'credential_id': credential}
    missing = supplied(['models', 'prepare'], preview_input)['result']
    if missing['access_ready'] or missing['credential_state'] != 'missing' or missing['upstream_authorization'] != 'not_checked':
        raise ValueError('Nonexistent protected reference reported ready')
    invoke(['credentials', 'select', 'vault'])
    protected = {'passphrase': 'synthetic-readiness-passphrase', 'value': 'synthetic-readiness-token'}
    invoke(['credentials', 'add', credential], input=json.dumps(protected).encode())
    invoke(['credentials', 'select', 'vault'])
    locked = supplied(['models', 'prepare'], preview_input)['result']
    if locked['access_ready'] or locked['credential_state'] != 'locked':
        raise ValueError('Locked protected reference reported ready')
    invoke(['credentials', 'unlock'], input=json.dumps({'passphrase': protected['passphrase']}).encode())
    available = supplied(['models', 'prepare'], preview_input)['result']
    if not available['access_ready'] or available['credential_state'] != 'available' or available['upstream_authorization'] != 'not_checked':
        raise ValueError('Locally resolvable protected reference was not distinguished from upstream access')
    if protected['value'] in json.dumps(available):
        raise ValueError('Preview disclosed resolved bytes')

    # Known-object integrity uses the actual filesystem backend and real CLI.
    source = workspace / 'published fixture.bin'
    source.write_bytes(b'abcdef')
    published = invoke(['artifacts', 'publish', source, 'other'])['result']
    publication = published.get('publication', published)
    artifact_id = publication['id']
    config = json.loads((workspace / '.insonic' / 'workspace.json').read_text(encoding='utf-8'))
    storage = Path(config['profiles']['storage']['configuration']['root'])
    if not storage.is_absolute():
        storage = workspace / '.insonic' / storage
    target = storage / publication['key']
    original = target.read_bytes()
    for mode in ['corrupt', 'truncated', 'missing']:
        if mode == 'missing':
            target.unlink()
        else:
            target.write_bytes(b'abcdeg' if mode == 'corrupt' else b'abc')
        for operation in ['verify', 'materialize']:
            response = invoke(['artifacts', operation, artifact_id], expected=1)
            if response['error']['code'] != 'checksum_mismatch':
                raise ValueError('Physical artifact integrity misclassified: ' + mode + '/' + operation)
        target.write_bytes(original)
    invoke(['artifacts', 'verify', artifact_id])
    return {'offline_budget_and_logical_assets': 'passed',
            'credential_local_resolution': 'passed', 'artifact_integrity_matrix': 'passed'}
