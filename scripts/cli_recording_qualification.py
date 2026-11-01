#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Installed admission/window/resume/coverage/repair journey using owned PCM and HTTP fixtures."""
import base64
from collections import Counter
import copy
from email import policy
from email.parser import BytesParser
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import io
import json
import struct
import threading
import time
import uuid
import wave

from cli_corpus import finish_server
import qualification_timing as timing


def qualify(invoke, supplied, workspace, version, example):
    started = time.monotonic()
    source = workspace / 'recording journey stereo.wav'
    # Distinct integer levels identify the submitted canonical lane/window without
    # private transcripts or an inference engine. This is not acoustic accuracy.
    with wave.open(str(source), 'wb') as output:
        output.setparams((2, 2, 16000, 34 * 16000, 'NONE', 'not compressed'))
        for second in range(34):
            index = second // 5
            frame = struct.pack('<hh', 1000 + index * 500, 11000 + index * 500)
            output.writeframes(frame * 16000)
    calls = Counter()
    entered, release = threading.Event(), threading.Event()
    repairing = threading.Event()
    errors = []

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            try:
                raw = self.rfile.read(int(self.headers['Content-Length']))
                message = BytesParser(policy=policy.default).parsebytes(
                    ('Content-Type: ' + self.headers['Content-Type'] + '\r\nMIME-Version: 1.0\r\n\r\n').encode() + raw)
                parts = {p.get_param('name', header='content-disposition'): p.get_payload(decode=True)
                         for p in message.iter_parts()}
                request = json.loads(parts['request'])
                with wave.open(io.BytesIO(parts['audio']), 'rb') as audio:
                    if audio.getnchannels() != 1 or audio.getsampwidth() != 2 or audio.getframerate() != 16000:
                        raise ValueError('Unexpected mapped fixture audio')
                    duration = audio.getnframes() * 1000000 // 16000
                    level = struct.unpack('<h', audio.readframes(1))[0]
                lane = int(level >= 10000)
                index = (level - 1000 - lane * 10000) // 500
                if lane not in [0, 1] or index not in range(7):
                    raise ValueError('Window did not preserve fixture source identity')
                operation = request['operation']
                mode = 'repair' if repairing.is_set() else 'initial'
                key = (mode, lane, index, operation)
                calls[key] += 1
                if key == ('initial', 0, 2, 'transcription') and calls[key] == 1:
                    entered.set()
                    if not release.wait(timing.phase_bound(timing.DURABLE_WORK_SECONDS)):
                        raise TimeoutError('Fixture interruption was not released')
                silence = lane == 1 or operation == 'transcription' and mode == 'initial' and index in [1, 4]
                response = {'contract_version': '1', 'operation': operation, 'no_speech': silence}
                interval = {'start_us': 500000, 'end_us': min(4500000, duration - 100000)}
                if operation == 'transcription':
                    response['segments'] = [] if silence else [{**interval, 'text': 'The same legitimate phrase.'}]
                else:
                    response['turns'] = [] if silence else [{**interval, 'label': 'voice'}]
                payload = json.dumps(response).encode()
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Content-Length', str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)
            except (BrokenPipeError, ConnectionResetError, ConnectionAbortedError):
                pass  # The first selected request is deliberately cancelled.
            except Exception as error:
                errors.append(type(error).__name__)
                self.send_error(500)

        def log_message(self, *_args):
            pass

    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        pipeline_id = str(uuid.uuid4())
        definition = example(version, 'pipelines', 'pipeline-connected.json')
        definition['pipeline']['id'] = pipeline_id
        definition['pipeline']['name'] = 'Recording journey deterministic fixture'
        configuration = definition['pipeline']['configuration']
        for stage in ['recognition', 'diarization']:
            configuration[stage]['endpoint'] = f'http://127.0.0.1:{server.server_port}/process'
        configuration['windows'] = {'core_us': 5000000, 'context_us': 0, 'tracks': [
            {'stream_index': 0, 'channel': 0}, {'stream_index': 0, 'channel': 1}]}
        configuration['coverage'] = example(version, 'pipelines', 'coverage.json')
        configuration['repair'] = example(version, 'pipelines', 'repair-policy.json')
        saved = supplied(['pipelines', 'set', pipeline_id], definition)['result']
        admitted = invoke(['media', 'import', source, '--new-entry', '--no-pipeline'])['result']
        media_id = invoke(['work', 'wait', admitted['work_id']])['result']['result']['items'][0]['media_id']
        canonical = invoke(['media', 'show', media_id])['result']
        process = {'pipeline_id': pipeline_id, 'pipeline_revision': saved['revision'], 'transcription': 'generate', 'diarization': 'run'}
        queued = supplied(['recordings', 'process', media_id], process)['result']
        if not entered.wait(timing.phase_bound(timing.READINESS_SECONDS)):
            raise ValueError('Later window did not reach interruption fixture')
        invoke(['work', 'cancel', queued['work_id']])
        release.set()
        invoke(['work', 'wait', queued['work_id']], expected=1)
        invoke(['work', 'retry', queued['work_id']])
        resumed = invoke(['work', 'wait', queued['work_id']])['result']
        if resumed['state'] != 'succeeded' or errors:
            raise ValueError('Window resume failed: ' + ','.join(errors))
        for index in [0, 1]:
            for operation in ['transcription', 'diarization']:
                if calls[('initial', 0, index, operation)] != 1:
                    raise ValueError('Resume recomputed a verified window')
        if calls[('initial', 0, 2, 'transcription')] != 2:
            raise ValueError('Interrupted window was not retried')
        original = invoke(['recordings', 'show', media_id])['result']['recording']
        def document():
            page = invoke(['recordings', 'document', media_id])['result']
            if not page.get('complete') or page.get('encoding') != 'base64':
                raise ValueError('Fixture document requires unexpected pagination')
            return json.loads(base64.b64decode(page['data']))
        before_document = document()
        coverage = supplied(['recordings', 'coverage', media_id], configuration['coverage'])['result']
        if (len(coverage['tracks']) != 2 or coverage['tracks'][0]['metrics']['uncovered_observed_speech_us'] != 8000000
                or coverage['tracks'][1]['metrics']['observed_no_speech_us'] != 34000000
                or coverage['tracks'][0]['metrics']['duplicate_overlap_us'] != 0):
            raise ValueError('Coverage denominators, deliberate gaps or repetition controls differ')
        if original != invoke(['recordings', 'show', media_id])['result']['recording']:
            raise ValueError('Coverage inspection mutated current authority')
        repairing.set()
        repair = example(version, 'pipelines', 'repair-automatic.json')
        repair.update(expected_revision=original['revision'], source_revision=original['source_revision'], document_digest=original['document_digest'])
        repair['processing'] = {'pipeline_id': pipeline_id, 'pipeline_revision': saved['revision']}
        request_id = str(uuid.uuid4())
        work = supplied(['recordings', 'repair', media_id], repair, request_id=request_id)['result']
        completed = invoke(['work', 'wait', work['work_id']])['result']
        decision = completed['result']
        human = invoke(['work', 'wait', work['work_id']], machine=False)
        if 'Repair: accepted' not in human or 'reason: improved' not in human:
            raise ValueError('Human repair completion hides acceptance or reason')
        if decision['status'] != 'accepted' or len(decision['selection']['windows']) != 2 or set(decision['attempts'].values()) != {1}:
            raise ValueError('Targeted repair was not accepted within its elected budget')
        for key, count in calls.items():
            if key[0] == 'repair' and (key[1] != 0 or key[2] not in [1, 4] or count != 1):
                raise ValueError('Repair executed unrelated windows or repeated inference')
        if sum(count for key, count in calls.items() if key[0] == 'repair') != 4:
            raise ValueError('Repair did not execute the two elected stage pairs')
        after_document = document()
        unaffected = {cue['id']: {k: v for k, v in cue.items() if k != 'ordinal'} for cue in before_document['cues']}
        for cue in after_document['cues']:
            if cue['id'] in unaffected and unaffected.pop(cue['id']) != {k: v for k, v in cue.items() if k != 'ordinal'}:
                raise ValueError('Repair changed an unaffected cue')
        if unaffected or before_document['source'] != after_document['source']:
            raise ValueError('Repair lost unaffected cues or changed native source evidence')
        current = invoke(['recordings', 'show', media_id])['result']['recording']
        after = invoke(['recordings', 'coverage', media_id])['result']
        if after['tracks'][0]['metrics']['uncovered_observed_speech_us'] != 0 or current['revision'] <= original['revision']:
            raise ValueError('Accepted repair did not strictly improve coverage')
        extraction = example(version, 'graph', 'extraction.json')
        extraction.update(expected_revision=current['revision'], document_digest=current['document_digest'])
        extraction_work = supplied(['evidence', 'extract', media_id], extraction)['result']
        invoke(['work', 'wait', extraction_work['work_id']])
        invoke(['evidence', 'show', media_id])
        invoke(['recordings', 'cues', media_id])
        # Readback does not rerun inference. A second automatic request has no
        # repairable gaps and must produce an unchanged accepted decision.
        count_before = sum(calls.values())
        replay = supplied(['recordings', 'repair', media_id], repair, request_id=request_id)['result']
        if replay['work_id'] != work['work_id'] or sum(calls.values()) != count_before:
            raise ValueError('Request UUID replay created new work or inference')
        repair.update(expected_revision=current['revision'], document_digest=current['document_digest'])
        unchanged = supplied(['recordings', 'repair', media_id], repair)['result']
        result = invoke(['work', 'wait', unchanged['work_id']])['result']['result']
        if result['status'] != 'unchanged' or result['reason'] != 'no_repairable_regions' or sum(calls.values()) != count_before:
            raise ValueError('No-target repair ran inference or changed current')
        authority = invoke(['media', 'show', media_id])['result']
        source_fields = ['id', 'revision', 'digest', 'size', 'original_publication_id', 'facts', 'duration_us']
        if any(canonical.get(key) != authority.get(key) for key in source_fields):
            raise ValueError('Transcript repair changed canonical audio authority')
        # Verify actual temporary publication retirement through the public route.
        def verify_retired(completed):
            decision = completed['result']
            plan_digest = decision['plan_digest']
            for window in decision['selection']['windows']:
                label = f"processing-window/{plan_digest}/{window['window_id']}/{completed['generation']}"
                identity = bytearray(hashlib.sha256((decision['work_id'] + '\x00' + label).encode()).digest()[:16])
                identity[6] = (identity[6] & 15) | 64
                identity[8] = (identity[8] & 63) | 128
                publication = str(uuid.UUID(bytes=bytes(identity)))
                if invoke(['artifacts', 'show', publication])['result']['state'] != 'retired':
                    raise ValueError('Decided repair retained temporary checkpoint bytes')
        verify_retired(completed)
        explicit = example(version, 'pipelines', 'repair-ranges.json')
        explicit.update(expected_revision=current['revision'], source_revision=current['source_revision'], document_digest=current['document_digest'])
        explicit['processing'] = {'pipeline_id': pipeline_id, 'pipeline_revision': saved['revision']}
        explicit['ranges'] = [{'stream_index': 0, 'channel': 0, 'start_sample': 0, 'end_sample': 80000}]
        unchanged = supplied(['recordings', 'repair', media_id], explicit)['result']
        completed = invoke(['work', 'wait', unchanged['work_id']])['result']
        if completed['result']['status'] != 'unchanged' or completed['result']['reason'] != 'not_improved':
            raise ValueError('Valid non-improving candidate replaced current')
        if current != invoke(['recordings', 'show', media_id])['result']['recording']:
            raise ValueError('Unchanged repair mutated current recording')
        verify_retired(completed)
        human = invoke(['work', 'wait', unchanged['work_id']], machine=False)
        if 'Repair: unchanged' not in human or 'reason: not_improved' not in human:
            raise ValueError('Human repair completion hides unchanged outcome')
        if errors:
            raise ValueError('Fixture handler error: ' + ','.join(errors))
        return {'state': 'passed', 'elapsed_seconds': time.monotonic() - started,
                'canonical_duration_us': 34000000, 'selected_lanes': 2, 'processing_windows': 14,
                'repair_windows': 2, 'interrupted_resume': 'passed', 'target_only_calls': 'passed',
                'fixed_basis_coverage': 'passed', 'native_preservation': 'passed', 'retrieval': 'passed',
                'checkpoint_retirement': 'passed', 'request_replay': 'passed', 'explicit_unchanged': 'passed', 'acoustic_accuracy': 'not-tested',
                'cpu_engine': 'not-elected', 'nvidia_engine': 'not-elected'}
    finally:
        release.set()
        finish_server(server, thread)
