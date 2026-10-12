#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Execute authoritative current documentation with disposable package fixtures.

HTTP responses are deterministic adapter fixtures, never inference engines.
"""
import base64
import csv
from email import policy
from email.parser import BytesParser
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import re
import threading
import uuid

ROOT = Path(__file__).resolve().parents[1]


def example(version, page, name):
    source = (ROOT / 'docs' / ('v' + version) / (page + '.md')).read_text(encoding='utf-8')
    pattern = r'<!-- executable-example: ' + re.escape(name) + r' -->\s*```json\s*(.*?)\s*```'
    matches = re.findall(pattern, source, re.S)
    if len(matches) != 1:
        raise ValueError('Missing or ambiguous documentation input: ' + name)
    return json.loads(matches[0])


def qualify(invoke, supplied, workspace, version):
    media = ROOT / 'tests/fixtures/media'
    preset = example(version, 'ingestion', 'preset-create.json')
    preset_id, profile_id = str(uuid.uuid4()), str(uuid.uuid4())
    preset['preset']['id'] = preset_id
    destination = workspace / 'documented destination'
    preset['preset']['options']['destination']['profile_id'] = profile_id
    preset['preset']['options']['destination']['configuration']['root'] = str(destination)
    saved = supplied(['import-presets', 'create'], preset)['result']
    invoke(['import-presets', 'list'])
    shown = invoke(['import-presets', 'show', saved['name']])['result']
    if shown['id'] != preset_id or shown['revision'] != saved['revision']:
        raise ValueError('Documented preset identity/readback differs')
    absent = invoke(['media', 'import', media / 'speech.flac', '--preset', 'missing-preset'], expected=1)
    if absent['error']['code'] != 'not_found' or 'import-presets list' not in absent['error']['message']:
        raise ValueError('Missing preset lacks documented actionable rejection')

    queued = invoke(['media', 'import', media / 'speech.flac', '--transcript', media / 'speech.srt',
                     '--preset', saved['name'], '--new-entry', '--originated-on', '2026-10-04'])['result']
    completed = invoke(['work', 'wait', queued['work_id']])['result']
    item = completed['result']['items'][0]
    media_id = item['media_id']
    choice = item['selection']
    if choice['preset_revision'] != saved['revision'] or choice['options']['destination']['profile_id'] != profile_id or item['queued_job_ids']:
        raise ValueError('Admission-only preset did not elect destination/date choices')
    if not destination.is_dir() or not any(p.is_file() for p in destination.rglob('*')):
        raise ValueError('Elected destination received no actual published bytes')

    # JSON defaults, item values and CLI overrides must yield the same election.
    manifest = workspace / 'documented imports.json'
    manifest.write_text(json.dumps({'kind': 'import-manifest', 'schema_version': version,
        'defaults': {'preset': preset_id, 'timezone': 'UTC'}, 'items': [
        {'source': str(media / 'speech.flac'), 'new_entry': True, 'timezone': 'Europe/London'}]}) + '\n', encoding='utf-8', newline='\n')
    for path in [manifest, workspace / 'documented imports.csv']:
        if path.suffix == '.csv':
            with path.open('w', encoding='utf-8', newline='') as stream:
                writer = csv.writer(stream)
                writer.writerow(['source', 'preset', 'timezone', 'new_entry', 'subtitle_stream_index', 'transcript'])
                writer.writerow([str(media / 'speech.flac'), preset_id, 'Europe/London', 'true', '0', str(media / 'speech.srt')])
        imported = invoke(['media', 'import', '--manifest', path, '--timezone', 'Asia/Tokyo'])['result']
        result = invoke(['work', 'wait', imported['work_id']])['result']['result']['items'][0]
        if result['selection']['options']['timezone'] != 'Asia/Tokyo':
            raise ValueError('Documented manifest/CLI precedence differs')
        if path.suffix == '.csv' and result['selection']['options'].get('subtitle_stream_index') != 0:
            raise ValueError('Explicit zero CSV option was discarded')
    refreshed = invoke(['media', 'refresh', media_id, '--preset', preset_id])['result']
    refresh = invoke(['work', 'wait', refreshed['work_id']])['result']['result']
    if refresh['selection']['preset_revision'] != saved['revision']:
        raise ValueError('Refresh did not apply the saved revision')

    calls = []
    weights = b'synthetic documented trained artifact'

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            raw = self.rfile.read(int(self.headers['Content-Length']))
            if self.path == '/train':
                request = json.loads(raw)
                if request['operation'] != 'train' or not request.get('uploads'):
                    self.send_error(400)
                    return
                output = {'contract_version': '1', 'kind': 'weights', 'architecture': 'custom',
                    'consumers': ['custom'], 'supported_operations': ['custom-inference'],
                    'artifacts': [{'role': 'weights.bin', 'format': 'binary', 'size': len(weights),
                        'sha256': hashlib.sha256(weights).hexdigest(), 'data': base64.b64encode(weights).decode()}]}
            else:
                message = BytesParser(policy=policy.default).parsebytes(
                    ('Content-Type: ' + self.headers['Content-Type'] + '\r\nMIME-Version: 1.0\r\n\r\n').encode() + raw)
                parts = {part.get_param('name', header='content-disposition'): part.get_payload(decode=True)
                         for part in message.iter_parts()}
                request = json.loads(parts['request'])
                operation = request['operation']
                calls.append(operation)
                output = {'contract_version': '1', 'operation': operation, 'no_speech': False}
                if operation == 'transcription':
                    output['segments'] = [{'start_us': 0, 'end_us': 2000000, 'text': 'Documented fixture text'}]
                else:
                    output['turns'] = [{'label': 'voice', 'start_us': 0, 'end_us': 2000000}]
            payload = json.dumps(output).encode()
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        def log_message(self, *_args):
            pass

    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        endpoint = f'http://127.0.0.1:{server.server_port}'
        pipeline = example(version, 'pipelines', 'pipeline-connected.json')
        pipeline_id = str(uuid.uuid4())
        pipeline['pipeline']['id'] = pipeline_id
        for stage in ['recognition', 'diarization']:
            pipeline['pipeline']['configuration'][stage]['endpoint'] = endpoint + '/process'
        saved_pipeline = supplied(['pipelines', 'set', pipeline_id], pipeline)['result']
        supplied(['pipelines', 'inspect', pipeline_id], example(version, 'pipelines', 'inspection.json'))
        elected = dict(preset)
        elected['expected_revision'] = saved['revision']
        elected['preset']['options']['pipeline_id'] = pipeline_id
        saved = supplied(['import-presets', 'update', preset_id], elected)['result']
        queued = invoke(['media', 'import', media / 'speech.flac', '--transcript', media / 'speech.srt',
                         '--new-entry', '--preset', preset_id])['result']
        elected_item = invoke(['work', 'wait', queued['work_id']])['result']['result']['items'][0]
        if len(elected_item['queued_job_ids']) != 1:
            raise ValueError('Preset did not queue exactly one actual processing child')
        child = invoke(['work', 'wait', elected_item['queued_job_ids'][0]])['result']
        if child['state'] != 'succeeded' or calls != ['diarization']:
            raise ValueError('Supplied-text preset did not bypass recognition')
        processing = example(version, 'pipelines', 'processing.json')
        processing.update(pipeline_id=pipeline_id, pipeline_revision=saved_pipeline['revision'])
        media_id = elected_item['media_id']
        work = supplied(['recordings', 'process', media_id], processing)['result']
        invoke(['work', 'wait', work['work_id']])
        record = invoke(['recordings', 'show', media_id])['result']['recording']
        document = invoke(['recordings', 'document', media_id])['result']
        # Decode the public byte-exact page without float-round-tripping integers.
        if document.get('encoding') != 'base64' or not document.get('complete'):
            raise ValueError('Small fixture document unexpectedly requires more pages')
        document = json.loads(base64.b64decode(document['data']))
        token = document['cues'][0]['speaker_attributions'][0]['speaker_id']

        identity = example(version, 'speakers', 'speaker-create.json')
        speaker_id = str(uuid.uuid4())
        identity['speaker']['id'] = speaker_id
        identity['aliases'][0].update(id=str(uuid.uuid4()), speaker_id=speaker_id)
        supplied(['speakers', 'set', speaker_id], identity)
        invoke(['speakers', 'show', speaker_id])
        mapping = example(version, 'speakers', 'mapping.json')
        mapping.update(revision=record['revision'], document_digest=record['document_digest'],
                       local_speaker_id=token, speaker_id=speaker_id)
        mapped = supplied(['recordings', 'map-speaker', media_id], mapping)['result']
        invoke(['recordings', 'mappings', media_id])
        mapping['mapping_revision'] = mapped['revision']
        supplied(['recordings', 'map-speaker', media_id], mapping)

        extraction = example(version, 'graph', 'extraction.json')
        extraction.update(expected_revision=record['revision'], document_digest=record['document_digest'])
        work = supplied(['evidence', 'extract', media_id], extraction)['result']
        invoke(['work', 'wait', work['work_id']])
        invoke(['evidence', 'show', media_id])
        for action in ['run', 'explain', 'validate']:
            supplied(['query', action], example(version, 'graph', 'query.json'))
        query_id, view_id = str(uuid.uuid4()), str(uuid.uuid4())
        supplied(['query', 'save', query_id], example(version, 'graph', 'saved-update.json'))
        invoke(['query', 'show', query_id])
        layout = example(version, 'graph', 'layout-update.json')
        layout['query_id'] = query_id
        supplied(['views', 'save', view_id], layout)
        invoke(['views', 'show', view_id])
        supplied(['timeline', 'calendar'], example(version, 'graph', 'calendar.json'))
        invoke(['timeline', 'recording', media_id])

        dataset = example(version, 'voice-models', 'dataset.json')
        dataset['speaker_id'] = speaker_id
        prepared = supplied(['models', 'dataset', 'create'], dataset)['result']
        if not prepared['reference_count']:
            raise ValueError('Documented dataset has no current evidence')
        filters = example(version, 'voice-models', 'speaker-filter.json')
        filters['speaker_id'] = speaker_id
        supplied(['models', 'dataset', 'list'], filters)
        supplied(['models', 'speaker', 'list'], filters)
        training = example(version, 'voice-models', 'training-hosted.json')
        training['dataset_id'] = prepared['dataset']['id']
        training['adapter']['endpoint'] = endpoint + '/train'
        trained = supplied(['models', 'train'], training)['result']
        version_id = invoke(['work', 'wait', trained['work_id']])['result']['result']['version_id']
        invoke(['models', 'speaker', 'show', version_id])
        profile = example(version, 'voice-models', 'profile.json')
        profile['version_id'] = version_id
        supplied(['models', 'profile', 'set', speaker_id], profile)
        fetched = workspace / 'documented model output'
        invoke(['models', 'speaker', 'fetch', version_id, '--destination', fetched])
        if not any(p.is_file() and p.read_bytes() == weights for p in fetched.rglob('*')):
            raise ValueError('Documented model fetch did not write exact fixture weights')

        source = example(version, 'models', 'source-update.json')
        source_id = str(uuid.uuid4())
        source['source']['id'] = source_id
        supplied(['models', 'source', 'set', source_id], source)
        invoke(['models', 'source', 'show', source_id])
        update = example(version, 'ingestion', 'preset-update.json')
        update['expected_revision'] = saved['revision']
        update['preset']['id'] = preset_id
        edited = supplied(['import-presets', 'update', preset_id], update)['result']
        delete = example(version, 'ingestion', 'preset-delete.json')
        delete['expected_revision'] = edited['revision']
        supplied(['import-presets', 'delete', preset_id], delete)
        invoke(['import-presets', 'show', preset_id], expected=1)
        invoke(['media', 'refresh', media_id, '--preset', preset_id], expected=1)
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
    return {'preset_lifecycle_destination_precedence_refresh': 'passed',
            'pipeline_actual_child_supplied_bypass': 'passed', 'documented_advanced_inputs': 'passed',
            'deterministic_training_fetch': 'passed', 'acoustic_accuracy': 'not-tested'}
