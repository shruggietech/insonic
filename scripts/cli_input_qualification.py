#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Installed input-cause, privacy and exclusive-export qualification."""
import json
from pathlib import Path
import time
import uuid


MODES = [('human', [], False), ('quiet', ['--quiet'], False),
         ('verbose', ['--verbose'], False), ('debug', ['--debug'], False),
         ('json', [], True)]


def qualify(invoke, workspace, version, *, media=False):
    """The harness owns executable, streams, exits and process-tree cleanup."""
    started = time.monotonic()
    workspace = Path(workspace)
    directory = workspace / 'input contract fixtures'
    directory.mkdir()
    input_file = directory / 'request.json'
    sentinel = 'PRIVATE_INPUT_VALUE_'
    private_member = 'PRIVATE_INPUT_MEMBER'

    def rejected(args, reason, expected, fragment, *, input=None, scoped=True):
        for mode, flags, machine in MODES:
            response = invoke([*flags, *args], expected=expected, machine=machine, input=input, scoped=scoped)
            encoded = json.dumps(response, ensure_ascii=False) if machine else response
            if sentinel in encoded or private_member in encoded:
                raise ValueError(reason + ' disclosed submitted input in ' + mode)
            if machine:
                error = response.get('error', {})
                if error.get('reason') != reason or not error.get('message'):
                    raise ValueError('Installed ' + reason + ' lost its generated cause')
                if response.get('result') is not None:
                    raise ValueError('Input/export rejection invented a result')
            elif fragment.lower() not in response.lower():
                raise ValueError('Installed ' + reason + ' lost its actionable human correction in ' + mode)

    baseline = invoke(['settings', 'show'])['result']['sections']['appearance']
    valid = {'section': 'appearance', 'revision': baseline['revision'],
             'value': {'theme': 'dark', 'reduced_motion': False}}

    def settings_value(value):
        return json.dumps({**valid, 'value': value})

    invalid = [
        ('empty', '', 'json_empty', 'JSON input is empty'),
        ('syntax', '{"' + private_member + '":', 'json_syntax', 'JSON syntax'),
        ('null', 'null', 'input_shape', 'wrong shape'),
        ('array', '[]', 'input_shape', 'wrong shape'),
        ('extra-document', json.dumps(valid) + ' {}', 'json_extra_document', 'more than one JSON document'),
        ('duplicate', settings_value({'theme': 'dark', 'reduced_motion': False}).replace(
            '"theme": "dark"', '"theme": "dark", "theme": "' + sentinel + '"'),
         'json_duplicate_key', 'repeats an object member'),
        ('unknown', settings_value({'theme': 'dark', 'reduced_motion': False, private_member: sentinel}),
         'input_unknown_field', 'unsupported member'),
        ('type', settings_value({'theme': 9, 'reduced_motion': False}), 'input_type', 'wrong type'),
        ('enum', settings_value({'theme': sentinel, 'reduced_motion': False}), 'input_enum', 'unsupported choice'),
        ('required', json.dumps({'section': 'appearance', 'revision': baseline['revision']}),
         'input_required', 'required input field'),
    ]
    for name, raw, reason, fragment in invalid:
        input_file.write_text(raw, encoding='utf-8', newline='\n')
        rejected(['settings', 'set', '--input', input_file], reason, 2, fragment)
    after = invoke(['settings', 'show'])['result']['sections']['appearance']
    if after != baseline:
        raise ValueError('Rejected settings changed the accepted section')

    credential = str(uuid.uuid4())
    for key in ['value', 'passphrase']:
        value = sentinel + 'x' * (2049 - len(sentinel))
        protected = json.dumps({key: value}).encode('utf-8')
        rejected(['credentials', 'add', credential], 'credential_value_limit', 2, '1..2048 UTF-8 bytes', input=protected)

    missing = directory / 'missing explicit workspace'
    rejected(['--workspace', missing, 'workspace', 'show'], 'workspace_missing', 4,
             'No workspace was found', scoped=False)
    if missing.exists():
        raise ValueError('Failed explicit workspace lookup initialized a directory')

    workspace_failures = []
    for damage, code, expected in [('malformed', 'invalid_request', 2),
                                   ('invalid-config', 'invalid_request', 2),
                                   ('unsupported-version', 'incompatible_version', 3)]:
        damaged = directory / (damage + ' workspace')
        invoke(['workspace', 'init', damaged], scoped=False)
        config_file = damaged / '.insonic' / 'workspace.json'
        config = json.loads(config_file.read_text(encoding='utf-8'))
        if damage == 'malformed':
            raw = '{"' + private_member + '":'
        else:
            if damage == 'unsupported-version':
                config['schema_version'] = '999.0.0'
            else:
                config['display_name'] = None
            raw = json.dumps(config)
        config_file.write_text(raw + '\n', encoding='utf-8', newline='\n')
        for mode, flags, machine in MODES:
            response = invoke([*flags, 'workspace', 'show'], expected=expected,
                              machine=machine, workspace_path=damaged)
            if machine:
                error = response.get('error', {})
                if error.get('code') != code or error.get('reason') == 'workspace_missing' or response.get('result') is not None:
                    raise ValueError('Existing damaged workspace lost its actual open failure')
                encoded = json.dumps(response)
            else:
                encoded = response
                if 'No workspace was found' in encoded:
                    raise ValueError('Existing damaged workspace was reported missing in ' + mode)
            if sentinel in encoded or private_member in encoded:
                raise ValueError('Workspace failure disclosed private configuration')
        workspace_failures.append(damage)

    existing = directory / 'existing catalog.json'
    preserved = b'synthetic existing catalog destination\n'
    existing.write_bytes(preserved)
    destination_directory = directory / 'directory operand'
    destination_directory.mkdir()
    missing_parent = directory / 'missing export parent' / 'snapshot.json'
    for path, reason, expected, fragment in [
            (existing, 'output_exists', 1, 'destination already exists'),
            (destination_directory, 'output_directory', 4, 'operand names a directory'),
            (missing_parent, 'output_parent_missing', 4, 'parent directory does not exist')]:
        rejected(['catalog', 'export', path], reason, expected, fragment)
        if existing.read_bytes() != preserved or missing_parent.exists() or list(destination_directory.iterdir()):
            raise ValueError('Rejected catalog export changed a preexisting destination')
    exported = directory / 'new snapshot.json'
    result = invoke(['catalog', 'export', exported])['result']
    if not result.get('exported') or not isinstance(json.loads(exported.read_text(encoding='utf-8')), dict):
        raise ValueError('Valid catalog destination failed after classified rejection checks')

    recording_export = {'state': 'not-elected', 'reason': 'exact packaged media companions required'}
    if media:
        from cli_media_qualification import pcm_fixture
        audio, subtitle = directory / 'source.wav', directory / 'source.srt'
        pcm_fixture(audio, 2)
        subtitle.write_text('1\n00:00:00,000 --> 00:00:00,500\nInput qualification fixture\n\n',
                            encoding='utf-8', newline='\n')
        queued = invoke(['media', 'import', audio, '--new-entry', '--no-pipeline'])['result']
        identity = invoke(['work', 'wait', queued['work_id']])['result']['result']['items'][0]['media_id']
        queued = invoke(['transcript', 'import', subtitle, '--record', identity])['result']
        invoke(['work', 'wait', queued['work_id']])
        recording = invoke(['recordings', 'show', identity])['result']['recording']
        destination = directory / 'existing recording.json'
        original = b'synthetic existing recording destination\n'
        destination.write_bytes(original)
        data = {'format': 'cueson', 'destination': str(destination), 'document_digest': recording['document_digest']}
        input_file.write_text(json.dumps(data) + '\n', encoding='utf-8', newline='\n')
        rejected(['recordings', 'export', identity, '--input', input_file], 'output_exists', 1,
                 'destination already exists')
        if destination.read_bytes() != original:
            raise ValueError('Recording export overwrote an existing destination')
        fresh = directory / 'new recording.json'
        input_file.write_text(json.dumps({**data, 'destination': str(fresh)}) + '\n', encoding='utf-8', newline='\n')
        written = invoke(['recordings', 'export', identity, '--input', input_file])['result']
        if written['bytes'] != len(fresh.read_bytes()) or not json.loads(fresh.read_text(encoding='utf-8')):
            raise ValueError('New recording export did not publish complete JSON bytes')
        recording_export = {'state': 'passed', 'preserved_existing_bytes': True, 'new_destination': True}
    return {'state': 'passed', 'issues': [117, 131], 'input_cases': [case[0] for case in invalid],
            'modes': [mode[0] for mode in MODES], 'credential_byte_cases': ['value-2049', 'passphrase-2049'],
            'missing_workspace': 'passed', 'workspace_open_failures': workspace_failures, 'catalog_destinations': ['existing-file', 'directory', 'missing-parent', 'new-file'],
            'recording_export': recording_export, 'elapsed_seconds': round(time.monotonic() - started, 3)}
