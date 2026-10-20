#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Execute the exact examples emitted by the installed CLI, with owned fixtures.

The ledger records positive lifecycle results and intentional, classified
prerequisite refusals separately. It never treats parser/schema rejection as
successful semantic qualification.
"""
import hashlib
import json
from pathlib import Path
import re
import shlex
import time
import uuid

PLACEHOLDERS = {'UUID', 'MEDIA_UUID', 'JOB_UUID', 'PUBLICATION_UUID', 'REFERENCE_UUID',
                'WORK_UUID', 'MODEL_UUID', 'SOURCE_UUID', 'VERSION_UUID', 'SPEAKER_UUID',
                'QUERY_UUID', 'VIEW_UUID', 'CURRENT_REVISION', 'REFERENCE', 'FILE',
                'INPUT_FILE', 'MANIFEST_FILE', 'DIRECTORY', 'ABSOLUTE_DIRECTORY'}
TEMPLATE_MARKER = 'JSON file template (replace illustrative IDs/revisions with current records):\n'


def emitted_example(text, path):
    examples = [line.removeprefix('Example: insonic ') for line in text.splitlines()
                if line.startswith('Example: insonic ')]
    if len(examples) != 1:
        raise ValueError('Expected exactly one actual emitted example: ' + ' '.join(path))
    argv = shlex.split(examples[0])
    if argv[:len(path)] != path or any(re.search(r'[\[\]{}]', arg) for arg in argv):
        raise ValueError('Example is another command or contains grammar: ' + ' '.join(path))
    return argv


def substitute(argv, values):
    if set(values) - PLACEHOLDERS:
        raise ValueError('Only documented placeholders may change')
    result = []
    for arg in argv:
        if arg in PLACEHOLDERS:
            if arg not in values:
                raise ValueError('Unresolved documented placeholder: ' + arg)
            result.append(str(values[arg]))
        else:
            result.append(arg)
    return result


def template(text):
    if TEMPLATE_MARKER not in text:
        return None
    return json.JSONDecoder().raw_decode(text.split(TEMPLATE_MARKER, 1)[1])[0]


def qualify(invoke, supplied, workspace, version, help_pages, *, executable=None, environment=None, process_factory=None):
    """Run every emitted argv in explicit fixtures; fail on unreviewed outcomes."""
    started = time.monotonic()
    workspace = Path(workspace)
    directory = workspace / 'actual help examples'
    directory.mkdir()
    library = directory / 'library'
    invoke(['workspace', 'init', library], scoped=False)
    def call(args, **kwargs):
        return invoke(args, workspace_path=library, **kwargs)
    def write(name, value):
        target = directory / name
        target.write_text(json.dumps(value) + '\n', encoding='utf-8', newline='\n')
        return target
    def input_call(args, value, **kwargs):
        return call([*args, '--input', write('preparation.json', value)], **kwargs)
    def wait(queued, **kwargs):
        return call(['work', 'wait', queued['work_id']], **kwargs)['result']
    ids = {key: str(uuid.uuid4()) for key in PLACEHOLDERS if key.endswith('_UUID') or key == 'UUID'}
    ids.update({'CURRENT_REVISION': '1', 'REFERENCE': 'missing-example-model',
                'ABSOLUTE_DIRECTORY': directory / 'fresh library', 'DIRECTORY': directory / 'new destination'})
    textfile = directory / 'publication.txt'
    textfile.write_text('owned semantic help fixture\n', encoding='utf-8', newline='\n')
    queryfile = directory / 'read.cypher'
    queryfile.write_text("RETURN 'fixture' AS value\n", encoding='utf-8', newline='\n')
    ids['FILE'] = textfile
    ids['INPUT_FILE'] = directory / 'input.json'
    ids['MANIFEST_FILE'] = directory / 'model.json'
    by_path = {tuple(key.split()): value for key, value in help_pages.items()}
    inventory = json.loads((Path(__file__).resolve().parents[1] / 'cmd/insonic/command_guidance.json').read_text(encoding='utf-8'))
    if set(inventory['commands']) != set(help_pages):
        raise ValueError('Reviewed command inventory differs from installed root help')
    ledger = []

    # Model bytes and manifests stay local. A file URL is intentionally not a
    # supported acquisition source; acquisition fixtures are supplied below by
    # the same owned loopback fixture used by installed model qualification.
    from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
    import threading
    body = b'actual emitted model example bytes'
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            self.send_response(200)
            data = json.dumps({'kind': 'model-catalog', 'schema_version': version, 'entries': []}).encode() if self.path == '/catalog' else body
            self.send_header('Content-Length', str(len(data)))
            self.end_headers()
            self.wfile.write(data)
        def log_message(self, *_args):
            pass
    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        model = {'kind': 'base-model-manifest', 'schema_version': version, 'name': 'help-example-model',
                 'model_version': '1', 'upstream_revision': 'fixture-v1', 'capabilities': ['transcription'],
                 'license': 'fixture', 'files': [{'role': 'weights', 'sha256': hashlib.sha256(body).hexdigest(),
                 'size': len(body), 'url': f'http://127.0.0.1:{server.server_port}/model', 'local_http': True}]}
        ids['MANIFEST_FILE'] = write('model.json', model)
        registered = wait(call(['models', 'acquire', ids['MANIFEST_FILE']])['result'])['result']
        ids['MODEL_UUID'] = registered['model_id']
        ids['REFERENCE'] = ids['MODEL_UUID']

        # Existing accepted records establish real current authority for read,
        # paging, roster, export and processing examples.
        from cli_media_qualification import pcm_fixture
        audio, subtitle = directory / 'source.wav', directory / 'source.vtt'
        pcm_fixture(audio, 1)
        subtitle.write_text('WEBVTT\n\n00:00:00.000 --> 00:00:00.500\n<v Alex>fixture résumé 文 😀\n\n', encoding='utf-8', newline='\n')
        media = wait(call(['media', 'import', audio, '--no-pipeline'])['result'])['result']['items'][0]
        ids['MEDIA_UUID'] = media['media_id']
        wait(call(['transcript', 'import', subtitle, '--record', ids['MEDIA_UUID']])['result'])
        record = call(['recordings', 'show', ids['MEDIA_UUID']])['result']['recording']
        speaker_input = template(by_path[('speakers', 'set')])
        speaker_input['speaker']['id'] = ids['SPEAKER_UUID']
        input_call(['speakers', 'set', ids['SPEAKER_UUID']], speaker_input)
        publication = call(['artifacts', 'publish', textfile, 'derived'])['result']
        ids['PUBLICATION_UUID'] = publication['id']
        materialization = call(['artifacts', 'materialize', ids['PUBLICATION_UUID']])['result']
        ids['REFERENCE_UUID'] = materialization['lease']['id']
        job = call(['jobs', 'start', '1'])['result']
        ids['JOB_UUID'] = job['job_id']
        work = call(['media', 'refresh', ids['MEDIA_UUID'], '--no-pipeline'])['result']
        ids['WORK_UUID'] = work['work_id']
        wait(work)

        def current_media():
            return call(['media', 'show', ids['MEDIA_UUID']])['result']['revision']
        def selected_input(name, help_text):
            value = template(help_text)
            if value is None:
                return None
            # Every replacement is a documented current identity/revision/path or
            # elected fixture selection. Constants/enums remain the emitted ones.
            def replace(value, key=''):
                if isinstance(value, list):
                    return [replace(v, key) for v in value]
                if isinstance(value, dict):
                    return {k: replace(v, k) for k, v in value.items()}
                if isinstance(value, str):
                    if key in ['speaker_id', 'canonical_speaker_id']:
                        return ids['SPEAKER_UUID']
                    if key in ['media_id', 'recording_id']:
                        return ids['MEDIA_UUID']
                    if key == 'query_id':
                        return ids['QUERY_UUID']
                    if key in ['model_id', 'base_model_id']:
                        return ids['MODEL_UUID']
                    if key in ['path', 'destination']:
                        return str(directory / (name.replace(' ', '-') + '-destination'))
                    if value == '11111111-1111-4111-8111-111111111111':
                        return ids['UUID']
                return value
            return replace(value)

        # Seed current records before their read examples. Set examples subsequently
        # use their observed revision rather than inventing an update receipt.
        for family, field in [('pipelines', 'pipeline'), ('terms', 'term')]:
            seed = selected_input(family + ' set', by_path[(family, 'set')])
            seed[field]['id'] = ids['UUID']
            input_call([family, 'set', ids['UUID']], seed)
        alias = selected_input('models alias set', by_path[('models', 'alias', 'set')])
        alias['alias']['id'] = ids['UUID']
        alias['alias']['target']['id'] = ids['MODEL_UUID']
        input_call(['models', 'alias', 'set', ids['UUID']], alias)
        source = selected_input('models source set', by_path[('models', 'source', 'set')])
        source['source']['id'] = ids['UUID']
        source['source']['url'] = f'http://127.0.0.1:{server.server_port}/catalog'
        source['source']['local_http'] = True
        input_call(['models', 'source', 'set', ids['UUID']], source)
        ids['SOURCE_UUID'] = ids['UUID']

        # Explicit expected refusals describe real configured prerequisites and
        # retained lifecycle states. Syntax/type/schema errors are never permitted.
        expected = {
            'media relocate': {'unsupported_capability', 'conflict', 'invalid_request'},
            'jobs retry': {'conflict'}, 'jobs cancel': {'conflict'},
            'work retry': {'conflict'}, 'work cancel': {'conflict'},
            'artifacts abort': {'conflict'}, 'artifacts retire': {'conflict'}, 'artifacts cache-prune': {'conflict'},
            'artifacts lease-renew': {'not_found', 'conflict'}, 'artifacts lease-release': {'not_found', 'conflict'},
            'artifacts release-reference': {'not_found'},
            'models dataset show': {'not_found'}, 'models speaker show': {'not_found'},
            'models speaker fetch': {'not_found'}, 'models profile set': {'not_found'},
            'models train': {'not_found', 'unsupported_capability', 'engine_ci_forbidden', 'conflict'},
            'models dataset create': {'not_found', 'unsupported_capability'},
            'models discover': {'not_found', 'unavailable'}, 'models resolve': {'not_found'},
            'query assist': {'unsupported_capability', 'unavailable'},
            'query assistance-test': {'conflict', 'unsupported_capability', 'unavailable'},
            'recordings process': {'not_found', 'unsupported_capability'},
            'recordings match': {'not_found', 'unsupported_capability'},
            'recordings resolve-segment': {'not_found', 'conflict'},
            'workers inspect': {'not_found', 'unavailable', 'unsupported_capability'},
            'workers setup': {'engine_ci_forbidden', 'engine_unavailable', 'unsupported_capability', 'unavailable'},
            'workers plan': {'unsupported_capability', 'unavailable'},
            'processing readiness': {'unsupported_capability'},
        }
        # Maintenance has its own stopped library, so examples prove exclusive
        # operations without interfering with the running fixture library.
        maintenance = directory / 'maintenance'
        invoke(['workspace', 'init', maintenance], scoped=False)
        backup = directory / 'backup'
        snapshot = directory / 'snapshot.json'
        # Executed order is explicit to preserve real dependency lifecycles.
        order = sorted(help_pages, key=lambda p: (p.startswith('backup release'), p.endswith(' remove'), p.endswith(' delete'), p))
        for name in order:
            command = name.split()
            help_text = help_pages[name]
            actual = emitted_example(help_text, command)
            replacements = {k: v for k, v in ids.items()}
            kwargs = {'expected': (0, 1, 2, 4)}
            value = selected_input(name, help_text)
            if value is not None:
                write('input.json', value)
            if name in ['backup restore', 'catalog restore']:
                restore_target = directory / name.replace(' ', '-')
                invoke(['workspace', 'init', restore_target], scoped=False)
                kwargs['workspace_path'] = restore_target
            if name.startswith('backup '):
                replacements['DIRECTORY'] = backup
                if name != 'backup create' and not backup.exists():
                    invoke(['backup', 'create', backup], workspace_path=maintenance)
                if name != 'backup restore':
                    kwargs['workspace_path'] = maintenance
            elif name.startswith('catalog '):
                # catalog show is runtime-backed inspection. Keep its owner in
                # the live library, away from exclusive backup/catalog examples.
                if name not in ['catalog restore', 'catalog show']:
                    kwargs['workspace_path'] = maintenance
                replacements['FILE'] = snapshot
                if name == 'catalog restore':
                    if not snapshot.exists():
                        invoke(['catalog', 'export', snapshot], workspace_path=maintenance)
                    source_config = maintenance / '.insonic' / 'workspace.json'
                    (restore_target / '.insonic' / 'workspace.json').write_bytes(source_config.read_bytes())
            elif name == 'workspace init':
                replacements['DIRECTORY'] = directory / 'initialized-example'
            elif name.startswith('workspace '):
                if name != 'workspace show':
                    write('input.json', {'display_name': 'Help example library'})
                    replacements['DIRECTORY'] = directory / ('planned-' + command[-1])
                    kwargs['scoped'] = False
            elif name == 'setup location':
                kwargs['scoped'] = False
            elif name.startswith('setup draft ') or name == 'setup provision':
                kwargs['scoped'] = False
                if name == 'setup draft save':
                    current_draft = invoke(['setup', 'draft', 'show'], scoped=False)['result']
                    write('input.json', {'expected_revision': current_draft['revision'], 'path': str(directory / 'draft-library'),
                                        'plan': {'display_name': 'Help example library'}, 'stage': 'workspace', 'status': 'active', 'credential_backend': 'native'})
            elif name.startswith('credentials '):
                call(['credentials', 'select', 'session'])
                replacements['UUID'] = ids['UUID']
                if name in ['credentials replace', 'credentials delete', 'credentials status']:
                    call(['credentials', 'load'], input=json.dumps({'session': {ids['UUID']: 'fixture-value'}}).encode())
                if name == 'credentials unlock':
                    call(['credentials', 'select', 'vault'])
                    call(['credentials', 'add', ids['UUID']], input=b'{"passphrase":"fixture-passphrase","value":"fixture-value"}')
                    kwargs['input'] = b'{"passphrase":"fixture-passphrase"}'
                elif name == 'credentials load':
                    kwargs['input'] = json.dumps({'session': {ids['UUID']: 'fixture-value'}}).encode()
                elif name in ['credentials add', 'credentials replace']:
                    kwargs['input'] = b'{"value":"fixture-value"}'
            elif name == 'recordings assemble':
                value['turns'] = [{'label': 'fixture voice', 'start_us': 0, 'end_us': 500000}]
                write('input.json', value)
            elif name == 'evidence extract':
                current_record = call(['recordings', 'show', ids['MEDIA_UUID']])['result']['recording']
                value['expected_revision'] = current_record['revision']
                value['document_digest'] = current_record['document_digest']
                write('input.json', value)
            elif name == 'media import':
                replacements['FILE'] = audio
            elif name == 'transcript import':
                replacements['FILE'] = subtitle
            elif name == 'query native':
                replacements['FILE'] = queryfile
            elif name == 'media relocate':
                replacements['CURRENT_REVISION'] = current_media()
            elif name == 'media set-origin':
                replacements['CURRENT_REVISION'] = current_media()
            elif name.startswith('recordings roster ') and command[-1] != 'show':
                replacements['CURRENT_REVISION'] = call(['recordings', 'roster', 'show', ids['MEDIA_UUID']])['result']['revision']
            elif name in ['speakers set', 'speakers show', 'speakers aliases', 'speakers select', 'speakers diagnostics']:
                replacements['UUID'] = ids['SPEAKER_UUID']
                if name == 'speakers set':
                    current = call(['speakers', 'show', ids['SPEAKER_UUID']])['result']
                    value['speaker']['id'] = ids['SPEAKER_UUID']
                    value['expected_revision'] = current['speaker']['revision']
                    write('input.json', value)
            elif name in ['query run', 'query explain', 'query validate']:
                write('input.json', {'definition': {'mode': 'normalized', 'operation': 'media-list'}})
            elif name == 'setup save':
                shown = call(['setup', 'show'])['result']
                write('input.json', {'expected_revision': shown['revision'], 'state': shown['state']})
            elif name in ['workers plan', 'workers setup']:
                value['offline_assets'] = str(directory / 'absent offline assets')
                if name == 'workers setup':
                    value['expected_revision'] = call(['settings', 'show'])['result']['sections']['processing_tools']['revision']
                write('input.json', value)
            elif name in ['pipelines set', 'terms set']:
                field = 'pipeline' if command[0] == 'pipelines' else 'term'
                current = call([command[0], 'show', ids['UUID']])['result']
                value[field]['id'] = ids['UUID']
                value['expected_revision'] = current['revision']
                write('input.json', value)
            elif name == 'recordings map-speaker':
                current_record = call(['recordings', 'show', ids['MEDIA_UUID']])['result']['recording']
                mappings = call(['recordings', 'mappings', ids['MEDIA_UUID']])['result']['items']
                document = current_record['document']
                if isinstance(document, str):
                    document = json.loads(document)
                voices = [a['speaker_id'] for cue in document['cues'] for a in cue.get('speaker_attributions', [])]
                if not voices:
                    raise ValueError('Supplied turn fixture retained no local speaker attribution')
                value['revision'] = current_record['revision']
                value['mapping_revision'] = mappings[0]['revision'] if mappings else 0
                value['document_digest'] = current_record['document_digest']
                value['local_speaker_id'] = voices[0]
                write('input.json', value)
            elif name == 'models dataset show':
                replacements['UUID'] = dataset_id
            elif name == 'models train':
                value['dataset_id'] = dataset_id
                write('input.json', value)
            elif name in ['models alias set', 'models source set']:
                field = command[1]
                current = call(['models', field, 'show', ids['UUID']])['result']
                value[field]['id'] = ids['UUID']
                value['expected_revision'] = current['revision']
                if field == 'alias':
                    value[field]['target']['id'] = ids['MODEL_UUID']
                else:
                    value[field]['url'] = f'http://127.0.0.1:{server.server_port}/catalog'
                    value[field]['local_http'] = True
                write('input.json', value)
            elif name in ['models alias remove', 'models source remove']:
                replacements['CURRENT_REVISION'] = call(['models', command[1], 'show', ids['UUID']])['result']['revision']
            elif name in ['import-presets update', 'import-presets delete']:
                current = call(['import-presets', 'show', ids['UUID']])['result']
                value['expected_revision'] = current['revision']
                write('input.json', value)
            elif name in ['media playback', 'media playback-check', 'media playback-close']:
                if name == 'media playback':
                    value['revision'] = current_media()
                else:
                    value['playback_id'] = playback_id
                write('input.json', value)
            elif name in ['media tools', 'processing tools']:
                companions = json.loads((Path(executable).parent / 'insonic-companions.json').read_text(encoding='utf-8'))
                def absolute(value):
                    if isinstance(value, list):
                        return [absolute(v) for v in value]
                    if isinstance(value, dict):
                        return {k: str(Path(executable).parent / v) if k in ['path', 'executable'] and isinstance(v, str) else absolute(v) for k, v in value.items()}
                    return value
                value = absolute(companions['media']) if name == 'media tools' else {'kind': 'processing-tools', 'schema_version': version, 'cueson': absolute(companions['cueson']), 'processing': {}}
                replacements['FILE'] = write('tools.json', value)
            elif name == 'settings set':
                shown = call(['settings', 'show'])['result']['sections']['appearance']
                write('input.json', {'section': 'appearance', 'revision': shown['revision'], 'value': shown['value']})
            if name == 'runtime serve':
                # A foreground owner blocks by design. The exact emitted argv
                # starts in its own ProcessTree; readiness is a real client call.
                if executable is None:
                    raise ValueError('runtime serve qualification needs the installed executable')
                from process_tree import ProcessTree
                serve = directory / 'explicit owner'
                invoke(['workspace', 'init', serve], scoped=False)
                argv = substitute(actual, replacements)
                with (process_factory or ProcessTree)([str(executable), '--workspace', str(serve), '--json', *argv],
                                 cwd=directory, env=environment) as tree:
                    time.sleep(.2)
                    if tree.process.poll() is not None:
                        raise ValueError('Emitted runtime serve did not remain alive')
                    observed = invoke(['workspace', 'show'], workspace_path=serve)
                    if not observed.get('runtime_session_id'):
                        raise ValueError('Explicit owner has no observed session')
                ledger.append({'command': name, 'example': actual, 'outcome': 'running-owner-observed'})
                continue
            argv = substitute(actual, replacements)
            target = kwargs.pop('workspace_path', library)
            response = invoke(argv, workspace_path=target, **kwargs)
            if target == maintenance and response.get('runtime_session_id'):
                raise ValueError('Exclusive maintenance fixture elected a runtime owner: ' + name)
            error = response.get('error')
            if error and error['code'] not in expected.get(name, set()):
                raise ValueError('Unreviewed semantic help example failure: ' + name + ': ' + json.dumps(error))
            result = response.get('result')
            if error and error['code'] == 'invalid_request' and (name != 'media relocate' or not isinstance(result, dict) or 'digest' not in result or result.get('mode') != 'copy'):
                raise ValueError('Parser/schema rejection cannot qualify an emitted example: ' + name)
            if name == 'models dataset create' and not error:
                dataset_id = result['dataset']['id']
            if name == 'media playback' and not error:
                playback_id = result['playback_id']
            if name == 'artifacts materialize' and not error:
                if Path(result['path']).read_bytes() != textfile.read_bytes():
                    raise ValueError('Actual artifact materialize example returned different bytes')
                call(['artifacts', 'lease-release', ids['PUBLICATION_UUID'], result['lease']['id']])
            if name in ['models materialize', 'models base materialize'] and not error:
                for file in result['files']:
                    if Path(file['path']).read_bytes() != body or file['sha256'] != hashlib.sha256(body).hexdigest():
                        raise ValueError('Actual model materialize example lost immutable fixture bytes')
                    call(['artifacts', 'lease-release', file['publication_id'], file['lease_id']])
            if not error and result is None:
                raise ValueError('Emitted example returned no established result: ' + name)
            outcome = error['code'] if error else 'succeeded'
            if not error and isinstance(result, dict) and result.get('work_id'):
                completed = call(['work', 'wait', result['work_id']], expected=(0, 1, 4))
                failure = completed.get('error')
                if failure and failure['code'] not in expected.get(name, set()):
                    raise ValueError('Unexpected emitted-example completion failure: ' + name + ': ' + json.dumps(failure))
                outcome = 'completed' if not failure else 'accepted-then-' + failure['code']
            entry = {'command': name, 'example': actual, 'outcome': outcome}
            if error:
                entry['scenario'] = {'media relocate': 'Managed canonical audio has no relocatable legacy source.',
                                     'jobs cancel': 'The bounded synthetic attempt already completed.',
                                     'jobs retry': 'A completed successful synthetic attempt is not retryable.',
                                     'work cancel': 'The accepted refresh already completed.',
                                     'work retry': 'Successful durable work is not retryable.',
                                     'artifacts abort': 'The publication is already committed.',
                                     'artifacts retire': 'A retained publication or active lease fences retirement.',
                                     'artifacts cache-prune': 'A live materialization lease fences pruning.',
                                     'artifacts lease-renew': 'The returned materialization lease was released or expired.',
                                     'artifacts lease-release': 'The returned materialization lease was already released or expired.',
                                     'models train': 'The selected fixture pipeline has no speaker-training stage.',
                                     'models speaker show': 'No trained speaker output exists in this fixture library.',
                                     'models speaker fetch': 'No trained speaker output exists in this fixture library.',
                                     'models profile set': 'No compatible trained version exists in this fixture library.',
                                     'recordings match': 'No compatible voice-matching model/profile is configured.',
                                     'recordings process': 'No inference environment or selected compatible diarization model is configured.',
                                     'recordings resolve-segment': 'No segment with the elected fixture UUID exists.',
                                     'query assist': 'No query-assistance route has been elected.',
                                     'query assistance-test': 'A current elected assistance route and settings revision are required.',
                                     'workers inspect': 'No recognition environment has been configured.',
                                     'workers setup': 'The explicitly elected offline asset directory contains no runtime/wheels.',
                                     'workers plan': 'The explicitly elected offline assets are absent.'}.get(name, 'The documented optional prerequisite is absent in this owned fixture.')
                if name == 'media relocate':
                    entry['observed'] = {k: result[k] for k in ['mode', 'digest', 'revision']}
                elif name.startswith('jobs '):
                    observed = call(['jobs', 'show', ids['JOB_UUID']])['result']
                    entry['observed'] = {k: observed[k] for k in ['job_id', 'claim_generation', 'state'] if k in observed}
                elif name.startswith('work '):
                    entry['observed'] = call(['work', 'show', ids['WORK_UUID']])['result']['state']
            ledger.append(entry)
            write('example-progress.json', ledger)
        if len(ledger) != len(help_pages) or {e['command'] for e in ledger} != set(help_pages):
            raise ValueError('Actual example ledger does not cover every registered command')
        return {'state': 'passed', 'executed': len(ledger), 'elapsed_seconds': round(time.monotonic() - started, 3), 'ledger': ledger}
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
