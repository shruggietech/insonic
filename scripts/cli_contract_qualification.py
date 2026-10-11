#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Exercise installed CLI contracts with disposable, nonsecret fixtures."""
import copy
from concurrent.futures import ThreadPoolExecutor
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import threading
import uuid

from process_tree import ProcessTree


def qualify(executable, workspace, *, env=None, cwd=None):
    executable, workspace = Path(executable), Path(workspace)
    environment = os.environ.copy()
    environment.update(env or {})
    version = (Path(__file__).resolve().parents[1] / 'VERSION').read_text().strip()

    def invoke(args, *, expected=0, input=None, scoped=True):
        correlation_id = str(uuid.uuid4())
        argv = [str(executable), '--json', '--request-id', correlation_id]
        if scoped:
            argv += ['--workspace', str(workspace)]
        argv += [str(a) for a in args]
        with ProcessTree(argv, cwd=cwd or workspace, env=environment,
                         allow_detached=scoped, input_pipe=input is not None) as tree:
            stdout, stderr = tree.process.communicate(input=input, timeout=35)
            code = tree.process.returncode
        if code != expected or (stderr if expected == 0 else stdout):
            raise ValueError('CLI stream/exit contract differs: ' + ' '.join(map(str, args[:3]))
                             + ' (exit ' + str(code) + ', response ' + (stderr if code else stdout).decode('utf-8')[:2000] + ')')
        result = json.loads(stderr if expected else stdout)
        if result.get('kind') != 'runtime-response' or result.get('schema_version') != version or (result.get('request_id') != correlation_id and '--request-id' not in args):
            raise ValueError('CLI envelope/correlation differs')
        return result

    root_help = invoke(['--help'], scoped=False)['result']['help']
    paths = [line.strip().split() for line in root_help.splitlines() if line.startswith('  ')]
    help_workspace = workspace / 'isolated help workspace'
    invoke(['workspace', 'init', help_workspace], scoped=False)
    before = sorted(str(p.relative_to(help_workspace)) for p in help_workspace.rglob('*'))
    def check_help(path):
        inside = invoke(['--workspace', help_workspace, *path, '--help'], scoped=False)['result']['help']
        outside = invoke(['--workspace', workspace / 'absent-help-workspace', *path, '-h'], scoped=False)['result']['help']
        prefixed = invoke(['help', *path], scoped=False)['result']['help']
        if inside != outside or inside != prefixed or 'Usage:' not in inside:
            raise ValueError('Contextual help differs: ' + ' '.join(path))
    with ThreadPoolExecutor(max_workers=4) as workers:
        list(workers.map(check_help, paths))
    for group in {tuple(path[:n]) for path in paths for n in range(1, len(path))}:
        if 'Commands:' not in invoke(list(group), scoped=False)['result']['help']:
            raise ValueError('No-argument group help differs')
    after = sorted(str(p.relative_to(help_workspace)) for p in help_workspace.rglob('*'))
    if before != after:
        raise ValueError('Help created workspace files')
    for args in [['doctor', '--unknown'], ['unknown'], ['doctor', '--workspace='],
                 ['doctor', '--request-id', 'invalid'], ['doctor', '--json'],
                 ['work', 'list', '--limit=0'], ['media', 'import', '--copy', '--reference', 'fixture.wav']]:
        invoke(args, expected=2, scoped=False)

    shown = invoke(['settings', 'show'])['result']
    input_file = workspace / 'cli-contract-input.json'
    def supplied(args, data, expected=0):
        input_file.write_text(json.dumps(data) + '\n', encoding='utf-8', newline='\n')
        return invoke([*args, '--input', input_file], expected=expected)
    original = shown['sections']['appearance']
    supplied(['settings', 'set'], {'section': 'appearance', 'revision': original['revision'],
                       'value': {'theme': 'dark', 'reduced_motion': True}})['result']
    supplied(['settings', 'set'], {'section': 'appearance', 'revision': original['revision'], 'value': None}, expected=1)
    current = invoke(['settings', 'show'])['result']['sections']['appearance']
    if current['revision'] == original['revision'] or current['value']['theme'] != 'dark':
        raise ValueError('Settings mutation did not reach the shared runtime')
    supplied(['settings', 'set'], {'section': 'appearance', 'revision': current['revision'], 'value': None})
    speaker = str(uuid.uuid4())
    page = invoke(['models', 'list', '--speaker', speaker])['result']
    exact = supplied(['models', 'speaker', 'list'], {'speaker_id': speaker})['result']
    if page != exact:
        raise ValueError('Initial speaker model cursor alias differs')

    # The same protected input must work while the existing runtime is locked.
    credential = str(uuid.uuid4())
    secret = {'passphrase': 'synthetic-cli-fixture-passphrase', 'value': 'synthetic-cli-fixture-value'}
    invoke(['credentials', 'select', 'vault'])
    invoke(['credentials', 'add', credential], input=json.dumps(secret).encode())
    invoke(['credentials', 'select', 'vault'])
    invoke(['credentials', 'delete', credential], input=b'{}', expected=2)
    invoke(['credentials', 'delete', credential], input=json.dumps({'passphrase': 'wrong-fixture-passphrase'}).encode(), expected=4)
    invoke(['credentials', 'delete', credential], input=json.dumps({'passphrase': secret['passphrase']}).encode())
    invoke(['credentials', 'select', 'session'])

    body = b'synthetic CLI model bytes'
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            data = body if self.path == '/good' else b'x' * len(body)
            self.send_response(200)
            self.send_header('Content-Length', str(len(data)))
            self.end_headers()
            self.wfile.write(data)
        def log_message(self, *_args):
            pass
    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    failures = []
    try:
        manifest = {'kind': 'base-model-manifest', 'schema_version': version, 'name': 'contract-fixture-' + speaker,
                    'model_version': '1', 'upstream_revision': 'fixture-v1', 'capabilities': ['transcription'],
                    'license': 'fixture', 'files': [{'role': 'weights', 'sha256': hashlib.sha256(body).hexdigest(),
                    'size': len(body), 'url': f'http://127.0.0.1:{server.server_port}/good', 'local_http': True}]}
        model_file = workspace / 'cli-contract-model.json'
        def submit(action, value):
            model_file.write_text(json.dumps(value) + '\n', encoding='utf-8', newline='\n')
            return invoke(['models', action, model_file])['result']
        registered = submit('register', manifest)
        invoke(['work', 'wait', registered['work_id']])
        changed = copy.deepcopy(manifest)
        changed['license'] = 'different-fixture'
        conflict = submit('register', changed)
        result = invoke(['work', 'wait', conflict['work_id']], expected=1)
        if result['error']['code'] != 'conflict' or result['result']['error'] != 'conflict':
            raise ValueError('Registration conflict lost its durable cause')
        failures.append((conflict['work_id'], 'conflict'))
        damaged = copy.deepcopy(manifest)
        damaged['name'] += '-damaged'
        damaged['files'][0]['url'] = damaged['files'][0]['url'].replace('/good', '/bad')
        acquired = submit('acquire', damaged)
        result = invoke(['work', 'wait', acquired['work_id']], expected=1)
        if result['error']['code'] != 'model_acquisition_failed':
            raise ValueError('Acquisition dependency cause lost')
        dependency = acquired['acquisition_ids'][0]
        result = invoke(['work', 'wait', dependency], expected=1)
        if result['error']['code'] != 'checksum_mismatch':
            raise ValueError('Acquisition checksum cause lost')
        failures += [(acquired['work_id'], 'model_acquisition_failed'), (dependency, 'checksum_mismatch')]
        for work, cause in failures:
            for action in ['show', 'results']:
                if invoke(['work', action, work])['result']['error'] != cause:
                    raise ValueError('Durable cause differs across result routes')
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
    input_file.unlink(missing_ok=True)
    return {'help_paths': len(paths), 'settings': 'passed', 'initial_speaker_cursor': 'passed',
            'protected_vault_delete': 'passed', 'typed_failures': failures}
