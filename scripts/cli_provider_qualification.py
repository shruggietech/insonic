#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Exercise exact installed HTTP contracts with owned, non-inference fixtures."""
import base64
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import threading
import time
from cli_media_qualification import pcm_fixture


def qualify(invoke, supplied, workspace):
    started = time.monotonic()
    directory = workspace / 'provider contract fixtures'
    directory.mkdir()
    audio, subtitle = directory / 'source.wav', directory / 'source.srt'
    pcm_fixture(audio, 2)
    subtitle.write_text(''.join(f'{i + 1}\n00:00:00,{i:03d} --> 00:00:00,{i + 1:03d}\nprovider-cue-{i}\n\n'
                                for i in range(130)), encoding='utf-8', newline='\n')
    work = invoke(['media', 'import', audio, '--new-entry', '--no-pipeline'])['result']
    media_id = invoke(['work', 'wait', work['work_id']])['result']['result']['items'][0]['media_id']
    work = invoke(['transcript', 'import', subtitle, '--record', media_id])['result']
    invoke(['work', 'wait', work['work_id']])
    record = invoke(['recordings', 'show', media_id])['result']['recording']
    cues = read_document(invoke, supplied, media_id, record['document_digest'])['cues']
    foreign = directory / 'foreign.srt'
    foreign.write_text('1\n00:00:00,000 --> 00:00:00,001\nforeign-current-cue\n\n', encoding='utf-8', newline='\n')
    work = invoke(['media', 'import', audio, '--new-entry', '--no-pipeline', '--transcript', foreign])['result']
    other_id = invoke(['work', 'wait', work['work_id']])['result']['result']['items'][0]['media_id']
    other_record = invoke(['recordings', 'show', other_id])['result']['recording']
    foreign_cue = read_document(invoke, supplied, other_id, other_record['document_digest'])['cues'][0]['id']
    mode, calls, redirected = 'extract', [], []

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            request = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
            calls.append((self.path, request.get('operation')))
            if self.path == '/followed':
                redirected.append(True)
            status, payload = 200, b''
            if self.path == '/extract':
                chunk = request['chunk']
                if mode == 'spoof':
                    output = {'contract_version': '1', 'assertions': [], 'source_start_us': 'spoof-secret'}
                else:
                    ids = [chunk['cues'][0]['id']]
                    if chunk['index'] == 0:
                        ids += ['unknown-cue', foreign_cue, cues[-1]['id']]
                    output = {'contract_version': '1', 'assertions': [
                        {'subject': 'provider-source', 'relation': 'states', 'object': 'conditional claim',
                         'polarity': 'negative', 'modality': 'conditional', 'conditions': ['unless revised'],
                         'cue_ids': [cue_id]} for cue_id in ids]}
                payload = json.dumps(output).encode()
            else:
                status = int(mode) if mode.isdigit() else 200
                if status == 302:
                    self.send_response(status)
                    self.send_header('Location', '/followed')
                    self.send_header('Content-Length', '0')
                    self.end_headers()
                    return
                if mode == 'oversize':
                    payload = b'x' * 1025
                elif mode == 'timeout':
                    time.sleep(0.2)
                    payload = b'{}'
                elif mode == 'valid':
                    payload = json.dumps({'contract_version': '1', 'query': {'definition': {
                        'mode': 'normalized', 'operation': 'media-list', 'pagination': {'limit': 5}}},
                        'explanation': 'List current media'}).encode()
                else:
                    payload = b'private-provider-body-secret'
            self.send_response(status)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(payload)))
            self.end_headers()
            try:
                self.wfile.write(payload)
            except ConnectionError:
                pass

        def log_message(self, *_args):
            pass

    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        endpoint = f'http://127.0.0.1:{server.server_port}'
        extraction = {'expected_revision': record['revision'], 'document_digest': record['document_digest'],
                      'configuration': {'adapter': 'insonic-http', 'contract_version': '1',
                          'endpoint': endpoint + '/extract', 'model': 'deterministic-fixture',
                          'window_cues': 128, 'overlap_cues': 0}}
        for alias in ['insonic-http', 'http']:
            extraction['configuration']['adapter'] = alias
            work = supplied(['evidence', 'extract', media_id], extraction)['result']
            invoke(['work', 'wait', work['work_id']])
            evidence = invoke(['evidence', 'show', media_id])['result']
            if len(evidence['assertions']) != 2 or len(evidence['diagnostics']) == 0:
                raise ValueError('Installed extraction lost assertions or accepted foreign/chunk citations')
            if evidence['provenance']['configuration']['adapter'] != 'insonic-http':
                raise ValueError('Legacy extraction alias was not canonicalized')
            query = {'definition': {'mode': 'normalized', 'operation': 'text-search',
                      'filters': {'media_ids': [media_id], 'text': 'provider-source'}, 'pagination': {'limit': 10}}}
            rows = supplied(['query', 'run'], query)['result']['items']
            assertions = [row for row in rows if row['kind'] == 'assertion']
            if len(assertions) != 2:
                raise ValueError('Installed extraction was not visible to normalized reads')
            for row in assertions:
                assertion = row['statement']
                cue = next(c for c in cues if c['id'] == assertion['cue_ids'][0])
                expected = str(cue['timing']['start_milliseconds'] * 1000)
                if row['source_start_us'] != expected or row['document_digest'] != record['document_digest']:
                    raise ValueError('Provider changed authoritative source provenance')
                if assertion['polarity'] != 'negative' or assertion['modality'] != 'conditional' or assertion['conditions'] != ['unless revised']:
                    raise ValueError('Provider proposition qualifiers were lost')
        mode = 'spoof'
        work = supplied(['evidence', 'extract', media_id], extraction)['result']
        invoke(['work', 'wait', work['work_id']])
        evidence = invoke(['evidence', 'show', media_id])['result']
        if evidence['assertions'] or not evidence['diagnostics']:
            raise ValueError('Spoofed provider fields were published')
        if not invoke(['search', 'provider-cue-0', '--media', media_id])['result']['items']:
            raise ValueError('Rejected extraction blocked current transcript search')

        configuration = {'enabled': True, 'adapter': 'insonic-http', 'contract_version': '1',
                         'endpoint': endpoint + '/query', 'model': 'deterministic-fixture',
                         'limits': {'max_response_bytes': 1024, 'timeout_ms': 1000}}
        request = {'prompt': 'private-source-secret', 'configuration': configuration}
        mode = 'valid'
        supplied(['query', 'assist'], request)
        for mode, code, reason in [('401', 'unavailable', 'provider_auth'), ('403', 'unavailable', 'provider_auth'),
                ('404', 'unavailable', 'provider_missing'), ('429', 'unavailable', 'provider_busy'),
                ('503', 'unavailable', 'provider_busy'), ('302', 'unavailable', 'provider_redirect'),
                ('malformed', 'operation_failed', 'provider_protocol'), ('oversize', 'output_limit', 'provider_response_limit'),
                ('timeout', 'unavailable', 'provider_timeout')]:
            configuration['limits']['timeout_ms'] = 30 if mode == 'timeout' else 1000
            path = directory / 'assistance.json'
            path.write_text(json.dumps(request) + '\n', encoding='utf-8', newline='\n')
            error = invoke(['query', 'assist', '--input', path], expected=4 if code == 'unavailable' else 1)['error']
            check_error(error, code, reason)
        if redirected:
            raise ValueError('Installed provider followed a redirect')
        artifact = directory / 'artifact.bin'
        artifact.write_bytes(b'x' * 25)
        publication = invoke(['artifacts', 'publish', artifact, 'other'])['result']
        check_error(invoke(['artifacts', 'materialize', publication['id'], '24'], expected=1)['error'],
                    'output_limit', 'artifact_materialization_limit')
        query = directory / 'bounded.cypher'
        query.write_text('UNWIND [' + ','.join(str(i) for i in range(501)) + '] AS n RETURN n', encoding='utf-8')
        check_error(invoke(['query', 'native', '--dialect', 'ladybug-cypher', '--file', query], expected=1)['error'],
                    'output_limit', 'native_result_limit')
        human = invoke(['artifacts', 'materialize', publication['id'], '24'], expected=1, machine=False)
        if 'materialization' not in human or 'max_bytes' not in human or 'child output' in human:
            raise ValueError('Human output did not explain the elected materialization bound')
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
    return {'state': 'passed', 'issues': [146, 127, 130], 'extraction_aliases': 2, 'cues': len(cues),
            'provider_failures': 9, 'operation_limits': 3, 'elapsed_seconds': round(time.monotonic() - started, 3)}


def check_error(error, code, reason):
    if error['code'] != code or error.get('reason') != reason:
        raise ValueError('Installed diagnostic classification differs: ' + json.dumps(error))
    if any(marker in json.dumps(error) for marker in ['private-provider-body-secret', 'private-source-secret', 'spoof-secret']):
        raise ValueError('Installed diagnostic exposed provider or source data')


def read_document(invoke, supplied, media_id, digest):
    data, offset = bytearray(), 0
    for _ in range(8):
        page = supplied(["recordings", "document", media_id], {"document_digest": digest, "offset": offset})["result"]
        if page.get("encoding") != "base64" or page.get("document_digest") != digest:
            raise ValueError("Provider fixture document identity changed")
        data.extend(base64.b64decode(page["data"]))
        if page.get("complete"):
            return json.loads(data)
        if page["next_offset"] <= offset:
            raise ValueError("Document page did not advance")
        offset = page["next_offset"]
    raise ValueError("Provider fixture exceeded its bounded document read")
