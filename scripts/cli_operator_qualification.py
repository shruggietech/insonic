#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Installed operator-result regressions using owned, nonsecret fixtures."""
import base64
import hashlib
import json
from pathlib import Path
import time
import uuid


def validate_work_outcome(text, work_id, state, *, recording_outcome=None):
    """Reject duplicate outcomes and inapplicable recovery in installed output."""
    lines = text.splitlines()
    if lines.count('State: ' + state.capitalize()) != 1 or sum(line.startswith('State:') for line in lines) != 1:
        raise ValueError('Work output omitted or repeated its outcome')
    if recording_outcome is not None:
        expected = 'Outcome: ' + recording_outcome.replace('-', ' ').capitalize()
        if lines.count(expected) != 1 or sum(line.startswith('Outcome:') for line in lines) != 1:
            raise ValueError('Work output omitted or repeated its recording outcome')
    if any(line.strip() == 'Error:' for line in lines):
        raise ValueError('Work output printed an empty error')
    if state in {'succeeded', 'failed', 'cancelled', 'interrupted'} and any(
            'work ' + action + ' ' + work_id in text for action in ['wait', 'cancel']):
        raise ValueError('Terminal work suggested waiting or cancellation')
    if any(line.startswith(key) for line in lines for key in ['Generation:', 'Payload:', 'Provenance:', 'Result:']):
        raise ValueError('Normal work output dumped internal detail')
    if work_id not in text or 'Complete JSON:' not in text:
        raise ValueError('Work outcome lost identity or complete readback')


def qualify_untimed_recording(invoke, supplied, audio, directory):
    """Qualify a durable empty recording outcome through the caller's adapter."""
    untimed = directory / 'zero duration.srt'
    untimed.write_text('1\n00:00:00,000 --> 00:00:00,000\nUntimed operator fixture.\n\n',
                       encoding='utf-8', newline='\n')
    queued = invoke(['media', 'import', audio, '--new-entry', '--no-pipeline', '--subtitle', untimed])['result']
    untimed_id = invoke(['work', 'wait', queued['work_id']])['result']['result']['items'][0]['media_id']
    # The caller owns the single request ID and validates its response envelope.
    # Synchronous assembly uses that observed request ID as its durable work ID.
    assembled = supplied(['recordings', 'assemble', untimed_id], {'turns': []})
    assembly_id = assembled['request_id']
    if assembled['result']['state'] != 'no-timed-subtitles':
        raise ValueError('Untimed fixture did not retain its no-timed-subtitles outcome')
    for action in ['show', 'wait']:
        validate_work_outcome(invoke(['work', action, assembly_id], machine=False), assembly_id, 'succeeded',
                              recording_outcome='no-timed-subtitles')
    return 'no-timed-subtitles'


def qualify(invoke, workspace, version, *, media=False):
    started = time.monotonic()
    workspace = Path(workspace)
    directory = workspace / 'operator contract fixtures'
    directory.mkdir()
    request_file = directory / 'page.json'

    def supplied(args, data, **kwargs):
        request_file.write_text(json.dumps(data, ensure_ascii=False) + '\n', encoding='utf-8', newline='\n')
        return invoke([*args, '--input', request_file], **kwargs)

    def absent(response, label):
        if response.get('result') is not None:
            raise ValueError(label + ' invented an unobserved result')

    absent(invoke(['setup', 'location', 'relative-location'], expected=2, scoped=False), 'Location preview')
    location = invoke(['setup', 'location', 'relative-location'], expected=2, scoped=False, machine=False)
    if any(token in location for token in ['Existing workspace:', 'Control directory:', 'Artifact directory:']):
        raise ValueError('Failed location preview emitted zero observation fields')

    credential = str(uuid.uuid4())
    passphrase = 'synthetic-operator-fixture-passphrase'
    protected = json.dumps({'passphrase': passphrase, 'value': 'synthetic-operator-fixture-value'}).encode()
    for temperature in ['cold', 'warm']:
        library = directory / (temperature + ' credential library')
        invoke(['workspace', 'init', library], scoped=False)
        invoke(['--workspace', library, 'credentials', 'select', 'vault'], scoped=False)
        invoke(['--workspace', library, 'credentials', 'add', credential], scoped=False, input=protected)
        if temperature == 'warm':
            invoke(['workspace', 'show'], workspace_path=library)
        for action in ['unlock', 'load']:
            for machine in [True, False]:
                failed = invoke(['credentials', action], workspace_path=library, expected=4, machine=machine,
                                input=json.dumps({'passphrase': 'wrong-operator-fixture-passphrase'}).encode())
                if machine:
                    absent(failed, temperature + ' credential ' + action)
                    if not failed.get('error'):
                        raise ValueError('Credential failure lost its classified error')
                elif 'State: Unlocked' in failed or 'State: Loaded' in failed or 'State: unlocked' in failed or 'State: loaded' in failed:
                    raise ValueError('Failed credential bootstrap claimed success')
        success = invoke(['credentials', 'unlock'], workspace_path=library,
                         input=json.dumps({'passphrase': passphrase}).encode())['result']
        if success.get('state') != 'unlocked':
            raise ValueError('Successful credential bootstrap lost observed success')
        human = invoke(['credentials', 'unlock'], workspace_path=library, machine=False,
                       input=json.dumps({'passphrase': passphrase}).encode())
        if 'unlocked' not in human.lower():
            raise ValueError('Successful human credential bootstrap lost observed success')
        invoke(['credentials', 'select', 'session'], workspace_path=library)
        loaded = invoke(['credentials', 'load'], workspace_path=library,
                        input=json.dumps({'session': {credential: 'synthetic-session-value'}}).encode())['result']
        if loaded.get('state') != 'loaded':
            raise ValueError('Successful session bootstrap lost observed success')
        absent(invoke(['credentials', 'load'], workspace_path=library, expected=2,
                      input=json.dumps({'session': {'invalid': 'synthetic-rejected-value'}}).encode()), 'Rejected session')

    missing = str(uuid.uuid4())
    for args in [['work', 'show', missing], ['jobs', 'show', missing], ['pipelines', 'show', missing],
                 ['speakers', 'show', missing], ['terms', 'show', missing], ['artifacts', 'show', missing],
                 ['artifacts', 'materialize', missing], ['artifacts', 'lease-renew', missing, str(uuid.uuid4())],
                 ['models', 'alias', 'show', missing], ['models', 'source', 'show', missing]]:
        absent(invoke(args, expected=1), ' '.join(args[:2]))

    missing_human = invoke(['work', 'show', missing], expected=1, machine=False)
    if 'work list' not in missing_human or any('work ' + action + ' ' + missing in missing_human
                                              for action in ['show', 'wait', 'cancel']):
        raise ValueError('Missing work suggested operations on its missing ID')
    unknown = invoke(['not-an-insonic-command'], expected=2, scoped=False, machine=False)
    if 'insonic --help' not in unknown or 'Command --help' in unknown:
        raise ValueError('Unknown command emitted an invalid help route')

    published_file = directory / 'retirement.txt'
    published_file.write_text('owned retirement fixture\n', encoding='utf-8', newline='\n')
    publication = invoke(['artifacts', 'publish', published_file, 'derived'])['result']['id']
    refusal = invoke(['artifacts', 'retire', publication], expected=1)['error']
    if refusal.get('code') != 'conflict' or refusal.get('reason') != 'artifact_grace_period':
        raise ValueError('Immediate retirement lost its grace-period reason')
    reference = str(uuid.uuid4())
    invoke(['artifacts', 'retain', publication, reference])
    retained = invoke(['artifacts', 'retire', publication], expected=1)['error']
    if retained.get('reason') != 'artifact_retained':
        raise ValueError('Retained artifact lost its reference reason')
    retained_human = invoke(['artifacts', 'retire', publication], expected=1, machine=False)
    if 'retained by a current reference' not in retained_human or 'current revision' in retained_human:
        raise ValueError('Retained artifact suggested an unrelated revision correction')
    invoke(['artifacts', 'release-reference', publication, reference])

    for mode in [[], ['--verbose'], ['--debug']]:
        response = invoke([*mode, 'processing', 'readiness'])['result']
        human = invoke([*mode, 'processing', 'readiness'], machine=False)
        for name, task in response['tasks'].items():
            if 'Task ' + name + ':' not in human or task['state'] not in human:
                raise ValueError('Bounded readiness omitted a task conclusion')
            corrections = set(task.get('actions', []))
            corrections.update(p['correction'] for p in task.get('prerequisites', []) if p.get('correction'))
            if any(correction not in human for correction in corrections):
                raise ValueError('Bounded readiness omitted a task correction')
        if 'Complete JSON:' not in human or '--json' not in human:
            raise ValueError('Readiness lacks its executable complete retrieval command')

    identity_cases = []
    manifest_file = directory / 'identity model.json'
    for name in [' ', 'line\nname ' + str(uuid.uuid4())]:
        manifest = {'kind': 'base-model-manifest', 'schema_version': version, 'name': name,
                    'model_version': ' \n' + str(uuid.uuid4()), 'upstream_revision': ' first\r\nsecond ',
                    'capabilities': ['transcription'], 'license': 'synthetic fixture',
                    'files': [{'role': 'weights', 'sha256': hashlib.sha256(b'fixture').hexdigest(), 'size': 7,
                               'url': 'http://127.0.0.1:9/not-downloaded', 'local_http': True}]}
        manifest_file.write_text(json.dumps(manifest) + '\n', encoding='utf-8', newline='\n')
        queued = invoke(['models', 'register', manifest_file])['result']
        completed = invoke(['work', 'wait', queued['work_id']])['result']['result']
        for action in ['show', 'wait']:
            validate_work_outcome(invoke(['work', action, queued['work_id']], machine=False), queued['work_id'], 'succeeded')
        model_id = completed['model_id']
        shown = invoke(['models', 'show', model_id])['result']
        human = invoke(['models', 'show', model_id], machine=False)
        for key in ['name', 'model_version', 'upstream_revision']:
            if shown['manifest'][key] != manifest[key] or json.dumps(manifest[key], ensure_ascii=False) not in human:
                raise ValueError('Exact blank/multiline model identity was changed or not visibly quoted')
        identity_cases.append(model_id)

    capture_cases = []
    recording_outcomes = []
    if media:
        from cli_media_qualification import pcm_fixture
        audio, subtitle = directory / 'capture café.wav', directory / 'rich.srt'
        pcm_fixture(audio, 2)
        recording_outcomes.append(qualify_untimed_recording(invoke, supplied, audio, directory))
        subtitle.write_text(''.join(f'{i + 1}\n00:00:00,{i:03d} --> 00:00:00,{i + 1:03d}\noperator-rich-transcript café\n\n'
                                    for i in range(131)), encoding='utf-8', newline='\n')
        queued = invoke(['media', 'import', audio, '--new-entry', '--no-pipeline'])['result']
        identity = invoke(['work', 'wait', queued['work_id']])['result']['result']['items'][0]['media_id']
        queued = invoke(['transcript', 'import', subtitle, '--record', identity])['result']
        invoke(['work', 'wait', queued['work_id']])
        recording = invoke(['recordings', 'show', identity])['result']['recording']
        for mode in [[], ['--verbose'], ['--debug']]:
            human = invoke([*mode, 'recordings', 'show', identity], machine=False)
            for token in [identity, str(recording['revision']), recording['document_digest'], '--json']:
                if token not in human:
                    raise ValueError('Rich transcript hid an identity, revision, digest or retrieval route')
            if mode and ('Request:' not in human or 'Workspace ID:' not in human):
                raise ValueError('Rich transcript hid verbosity context')
        search = ['search', 'operator-rich-transcript', '--media', identity, '--limit', '100']
        page = invoke(search)['result']
        human = invoke(search, machine=False)
        if not page.get('next_cursor') or page['next_cursor'] not in human or 'Matching rows:' not in human:
            raise ValueError('Large search page hid its paging authority')
        entry = invoke(['media', 'show', identity])['result']
        for section in ['metadata', 'facts', 'dates']:
            full = supplied(['media', 'capture', identity], {'section': section, 'revision': entry['revision']})['result']
            expected = base64.b64decode(full['data'])
            if not full['complete']:
                raise ValueError('Operator capture fixture exceeded its elected complete page')
            first = supplied(['media', 'capture', identity], {'section': section, 'revision': entry['revision'], 'limit': 13})['result']
            human = supplied(['media', 'capture', identity], {'section': section, 'revision': entry['revision'], 'limit': 13}, machine=False)
            if ('Capture ' + section + ' page' not in human or first['data'] in human
                    or 'media capture ' + identity not in human or not first['complete'] and 'Next page input' not in human):
                raise ValueError('Capture page is not readable and actionable')
            continuation = json.loads(next(line.split(': ', 1)[1] for line in human.splitlines() if line.startswith('Next page input'))) if not first['complete'] else None
            if continuation is not None and continuation != {'section': section, 'revision': entry['revision'], 'offset': first['next_offset'],
                                'limit': 65536, 'sha256': first['sha256']}:
                raise ValueError('Capture continuation lost authority')
            following = supplied(['media', 'capture', identity], continuation)['result'] if continuation else None
            actual = base64.b64decode(first['data']) + (base64.b64decode(following['data']) if following else b'')
            if actual != expected or hashlib.sha256(actual).hexdigest() != full['sha256']:
                raise ValueError('Capture continuation changed exact bytes')
            for index, byte in enumerate(expected):
                if byte >= 0xc2:
                    split = supplied(['media', 'capture', identity], {'section': section, 'revision': entry['revision'],
                                     'sha256': full['sha256'], 'offset': index, 'limit': 1})['result']
                    text = supplied(['media', 'capture', identity], {'section': section, 'revision': entry['revision'],
                                    'sha256': full['sha256'], 'offset': index, 'limit': 1}, machine=False)
                    if base64.b64decode(split['data']) != expected[index:index + 1] or 'UTF-8 boundary:' not in text:
                        raise ValueError('Split UTF-8 capture byte page was changed or misrepresented')
                    break
            capture_cases.append(section)
    return {'state': 'passed', 'issues': [116, 120, 121, 134, 144, 207], 'credential_temperatures': ['cold', 'warm'],
            'concise_terminal_work': 'passed', 'missing_work_guidance': 'passed', 'unknown_command_help': 'passed',
            'artifact_lifecycle_reasons': ['artifact_grace_period', 'artifact_retained'],
            'recording_outcomes': recording_outcomes, 'capture_sections': capture_cases, 'exact_model_identities': identity_cases, 'media_fixtures_elected': media,
            'elapsed_seconds': round(time.monotonic() - started, 3)}
