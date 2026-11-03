# SPDX-License-Identifier: Apache-2.0
"""Installed speaker evaluation with separate semantic and acoustic truth."""
import base64
import copy
from contextlib import contextmanager
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import ipaddress
import json
import math
from pathlib import Path
import re
import struct
import threading
import time
import urllib.parse
import urllib.request
import uuid
import wave

from cli_corpus import finish_server
import qualification_timing as bounds
import speaker_reference as reference

ADAPTER = {'id': 'pyannote-profile', 'contract_version': '1', 'mode': 'local', 'architecture': 'pyannote',
           'output_kinds': ['voice-embedding'], 'consumers': ['voice-matching'],
           'supported_operations': ['voice-matching']}


def value_digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(',', ':'), allow_nan=False).encode()).hexdigest()


def worker_environment(setup, python_sha256):
    pinned = setup['environment_receipt']
    path = Path(pinned['path'])
    if path.is_symlink() or not path.is_file():
        raise ValueError('Worker environment receipt is not an original file')
    with path.open('rb') as source:
        raw = source.read((16 << 20) + 1)
    if len(raw) > 16 << 20 or hashlib.sha256(raw).hexdigest() != pinned['sha256']:
        raise ValueError('Worker environment receipt identity differs')
    receipt = json.loads(raw, object_pairs_hook=reference.unique_object)
    inspected = receipt['inspection']
    if (setup.get('role') != 'diarization' or inspected.get('role') != 'diarization'
            or setup.get('verification') != 'dependency_bytes_and_library_probe' or setup.get('inference') != 'not_run'
            or receipt.get('mode') != 'reuse' or inspected['python']['sha256'] != python_sha256
            or inspected.get('compatible') is not True or inspected.get('engine_available') is not True
            or not inspected.get('packages') or not reference.text(inspected.get('python_version'), 128)):
        raise ValueError('Selected acoustic environment was not verified')
    return {'role': 'diarization', 'state': 'verified', 'environment_receipt_sha256': pinned['sha256'],
            'python_sha256': python_sha256, 'python_version': inspected['python_version'],
            'packages': inspected['packages'], 'cpus': inspected['cpus'],
            'devices': {device: {key: value for key, value in observed.items()
                                if key in ('available', 'hardware_count', 'dependency_state')}
                        for device, observed in inspected['devices'].items()},
            'processing_tools_revision': setup['processing_tools_revision'], 'includes_inference': False}


def historical_outputs(before_files, after_files, before_manifest, after_manifest):
    if (set(before_files) != set(after_files) or 'manifest.json' not in before_files
            or len(before_files) < 2 or before_manifest.get('state') != 'current'):
        raise ValueError('Historical output inventory differs')
    immutable = {name: digest for name, digest in before_files.items() if name != 'manifest.json'}
    if any(after_files[name] != digest for name, digest in immutable.items()):
        raise ValueError('Correction changed immutable model bytes')
    expected = copy.deepcopy(before_manifest)
    expected.update(state='invalidated', manifest_sha256=None)
    if 'manifest_artifact' in expected: expected['manifest_artifact'] = None
    expected['training']['preparation'] = None
    expected['training']['dataset'].update(state='invalidated', manifest_sha256=None)
    if after_manifest != expected:
        raise ValueError('Corrected manifest changed historical origin or retained obsolete lineage')
    return {'immutable_files': immutable, 'manifest_before_sha256': before_files['manifest.json'],
            'manifest_after_sha256': after_files['manifest.json'], 'manifest_lineage_invalidated': True}


def local_endpoint(endpoint):
    parsed = urllib.parse.urlsplit(endpoint)
    host = parsed.hostname
    try:
        local = host == 'localhost' or ipaddress.ip_address(host).is_loopback
    except (ValueError, TypeError):
        local = False
    if (parsed.scheme != 'http' or not local or parsed.username or parsed.password
            or parsed.query or parsed.fragment or parsed.path not in ('', '/')):
        raise ValueError('Speaker evaluation requires an explicit local Ollama base endpoint')
    return endpoint.rstrip('/')


def validate_election(mode, elected, configuration, environment):
    if mode not in ('deterministic', 'real'):
        raise ValueError('Unknown speaker qualification mode')
    if mode == 'deterministic':
        if elected or configuration is not None:
            raise ValueError('Deterministic checks cannot elect inference')
        return
    if not elected or any(environment.get(key) for key in ('CI', 'GITHUB_ACTIONS', 'TF_BUILD', 'BUILD_BUILDID', 'JENKINS_URL')):
        raise ValueError('Real speaker evaluation requires explicit election outside deterministic CI')
    if (not isinstance(configuration, dict) or configuration.get('enabled') is not True
            or configuration.get('adapter') != 'ollama' or configuration.get('route') != 'local'
            or configuration.get('credential_id') or not reference.text(configuration.get('model'), 512)
            or set(configuration) - {'enabled', 'adapter', 'route', 'endpoint', 'model', 'mode', 'contract_version', 'limits'}):
        raise ValueError('Real evaluation provider election differs')
    local_endpoint(configuration.get('endpoint', ''))


def device_receipt(executions, requested):
    if requested not in ('cpu', 'cuda') or not isinstance(executions, list) or not 1 <= len(executions) <= 16:
        raise ValueError('Actual elected device observations are required')
    retained = []
    for item in executions:
        attempts = item.get('attempted_devices')
        if (item.get('requested_device') != requested or item.get('device') != requested
                or not isinstance(attempts, list) or not 1 <= len(attempts) <= 2
                or any(device != requested for device in attempts) or item.get('fallback_reason')
                or type(item.get('count')) is not int or not 1 <= item['count'] <= 1000000):
            raise ValueError('Device execution does not satisfy the elected qualification row')
        retained.append({key: item[key] for key in ('requested_device', 'device', 'attempted_devices',
                        'count', 'resource_adjustments') if key in item})
    return retained


def same_model(before, after, expected_digest):
    if (not isinstance(expected_digest, str) or not re.fullmatch('[0-9a-f]{64}', expected_digest)
            or before != after or before.get('digest') != expected_digest
            or not reference.text(before.get('model'), 512) or not reference.text(before.get('version'), 128)):
        raise ValueError('Contextual model identity changed or differs from election')


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *_args, **_kwargs):
        raise ValueError('Qualification model metadata redirect refused')


def model_snapshot(configuration, *, timeout=10):
    endpoint = local_endpoint(configuration['endpoint'])
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    def get(path):
        with opener.open(endpoint + path, timeout=timeout) as response:
            raw = response.read((1 << 20) + 1)
        if len(raw) > 1 << 20:
            raise ValueError('Model metadata limit exceeded')
        return json.loads(raw, object_pairs_hook=reference.unique_object)
    version, tags = get('/api/version'), get('/api/tags')
    selected = [model for model in tags.get('models', []) if model.get('name') == configuration['model']]
    if len(selected) != 1:
        raise ValueError('Exact elected local model is unavailable')
    result = {'version': version.get('version'), 'model': configuration['model'], 'digest': selected[0].get('digest')}
    same_model(result, result, result['digest'])
    return result


def independent_sources(clips):
    seen = set()
    for clip in clips:
        digest = clip.get('sha256')
        if not isinstance(digest, str) or not re.fullmatch('[0-9a-f]{64}', digest) or digest in seen:
            raise ValueError('Acoustic sources repeat across enrollment/evaluation')
        seen.add(digest)


def acoustic_metrics(rows, enrolled):
    allowed = ('correct_match', 'incorrect', 'unknown', 'correct_unknown', 'ambiguous', 'error')
    counts = {outcome: 0 for outcome in allowed}
    known = unseen = 0
    for row in rows:
        if row.get('outcome') not in counts:
            raise ValueError('Unaccounted acoustic outcome')
        counts[row['outcome']] += 1
        if row['expected_reader'] in enrolled: known += 1
        else: unseen += 1
    return {'outcome_counts': counts,
            'identity_precision': reference.metric(counts['correct_match'], counts['correct_match'] + counts['incorrect']),
            'identity_coverage': reference.metric(counts['correct_match'], known),
            'unseen_rejection': reference.metric(counts['correct_unknown'], unseen)}


def controlled_reply(case, context):
    cues = context.get('cues', [])
    if len(cues) != len(case['cues']) or any(actual.get('text') != expected['text']
                                           for actual, expected in zip(cues, case['cues'])):
        raise ValueError('Controlled fixture received different source context')
    observations = []
    for expected in case['observations']:
        cue = cues[expected['cue']]
        if len(cue.get('voices', [])) != 1:
            raise ValueError('Controlled fixture requires one supplied voice per cue')
        observations.append({'class': expected['class'], 'name': expected['name'], 'voice_id': cue['voices'][0],
            'references': [{'cue_id': cue['id'], 'quote': cue['text']}], 'uncertain': False})
    return {'version': 1, 'observations': observations}


@contextmanager
def controlled_provider():
    state = {'case': None}
    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            try:
                size = int(self.headers.get('Content-Length', '0'))
                if not 0 < size <= 65536: raise ValueError('Request size differs')
                request = json.loads(self.rfile.read(size))
                data = json.dumps(controlled_reply(state['case'], request['context'])).encode()
            except (ValueError, KeyError, TypeError):
                self.send_error(400)
                return
            self.send_response(200)
            self.send_header('Content-Length', str(len(data)))
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(data)
        def log_message(self, *_args): pass
    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield state, {'enabled': True, 'adapter': 'insonic-http', 'contract_version': '1', 'route': 'local',
                      'endpoint': f'http://127.0.0.1:{server.server_port}/discover', 'model': 'controlled-speaker-fixture'}
    finally:
        finish_server(server, thread)


def case_result(case, action):
    try:
        return {'id': case['id'], 'state': 'completed', **action()}
    except (ValueError, RuntimeError, OSError, TimeoutError, KeyError) as error:
        # Exception text can contain paths or provider output; retain only class.
        return {'id': case['id'], 'state': 'error', 'observations': [], 'error': type(error).__name__.lower()}


def carrier(path, seconds, seed):
    frequency = 180 + int(hashlib.sha256(seed.encode()).hexdigest()[:4], 16) % 300
    with wave.open(str(path), 'wb') as output:
        output.setparams((1, 2, 16000, 0, 'NONE', 'not compressed'))
        output.writeframes(b''.join(struct.pack('<h', int(1000 * math.sin(2 * math.pi * frequency * sample / 16000)))
                                   for sample in range(seconds * 16000)))


def install_recording(journey, audio, cues, stem):
    """Supply explicit known activity; do not claim model-generated timing."""
    media = journey.finish(journey.command('media', 'import', audio, '--new-entry', '--no-pipeline'))['items'][0]['media_id']
    subtitle = journey.base / (stem + '.srt')
    def stamp(ms):
        return f'{ms // 3600000:02d}:{ms // 60000 % 60:02d}:{ms // 1000 % 60:02d},{ms % 1000:03d}'
    subtitle.write_text(''.join(f'{i + 1}\n{stamp(cue["start_ms"])} --> {stamp(cue["end_ms"])}\n{cue["text"]}\n\n'
                               for i, cue in enumerate(cues)), encoding='utf-8', newline='\n')
    journey.finish(journey.command('transcript', 'import', subtitle, '--record', media))
    exported = journey.command('recordings', 'document', media)
    if not exported['complete']: raise ValueError('Supplied fixture document unexpectedly paged')
    document = json.loads(base64.b64decode(exported['data'], validate=True))
    if len(document['cues']) != len(cues): raise ValueError('Imported supplied cue count differs')
    for actual, expected in zip(document['cues'], cues):
        if actual['payload']['plain_text'] != expected['text']:
            raise ValueError('Supplied transcript bytes differ')
        actual['speaker_attributions'] = [{'speaker_id': expected['voice'],
            'start_milliseconds': expected['start_ms'], 'end_milliseconds': expected['end_ms']}]
    path = journey.base / (stem + '.cueson.json')
    path.write_text(json.dumps(document) + '\n', encoding='utf-8', newline='\n')
    journey.finish(journey.command('transcript', 'import', path, '--record', media, '--replace-transcript'))
    return media


def identity(journey, name):
    speaker = str(uuid.uuid4())
    journey.input('speakers', 'set', speaker, value={'expected_revision': 0,
                  'speaker': {'id': speaker, 'name': name, 'state': 'active'}, 'aliases': []})
    return speaker


def contextual(journey, configuration, *, controlled=None, corpus=None, checkpoint=lambda: None):
    corpus, digest = reference.load_corpus() if corpus is None else (corpus, reference.CORPUS_SHA256)
    rows = [{'id': case['id'], 'state': 'error', 'observations': [], 'error': 'not-executed'}
            for case in corpus['cases']]
    result = {'corpus_sha256': digest, 'model_quality': controlled is None,
              'audio_content_not_evaluated': True, 'cases': rows, 'metrics': reference.score(corpus, rows)}
    journey.receipt['contextual'] = result
    checkpoint()
    for index, case in enumerate(corpus['cases']):
        created = {}
        def action():
            for item in case['identities']:
                created[item['key']] = identity(journey, item['name'])
            audio = journey.base / (case['id'] + '.wav')
            carrier(audio, 2 * len(case['cues']), case['id'])
            cues = [cue | {'start_ms': i * 2000, 'end_ms': (i + 1) * 2000} for i, cue in enumerate(case['cues'])]
            media = install_recording(journey, audio, cues, case['id'])
            journey.command('recordings', 'roster', 'add', media, '--expected-revision', '0',
                            *[part for speaker in created.values() for part in ('--speaker', speaker)])
            if controlled is not None: controlled['case'] = case
            queued = journey.input('speakers', 'run', media, value={'configuration': {
                'provider': configuration, 'apply': False, 'enroll': False, 'select_profile': False}})
            journey.finish(queued)
            aggregate = journey.command('speakers', 'hypotheses', media)['aggregate']
            reverse = {value: key for key, value in created.items()}
            observations = []
            for item in aggregate['decisions']:
                speaker = item.get('speaker_id')
                if speaker is not None and speaker not in reverse:
                    raise ValueError('Decision escaped its elected roster')
                observations.append({'voice': item['voice_id'], 'outcome': item['outcome'],
                                     'speaker': reverse.get(speaker), 'identity_status': item['identity_status']})
            return {'observations': observations, 'recording_id': media,
                    'work_id': queued.get('work_id', queued.get('id')), 'aggregate_sha256': value_digest(aggregate),
                    'coverage': aggregate['coverage'], 'provenance': aggregate['provenance'],
                    'hypotheses': aggregate['hypotheses'], 'decisions': aggregate['decisions']}
        try:
            rows[index] = case_result(case, action)
            result['metrics'] = reference.score(corpus, rows)
            checkpoint()
        finally:
            for speaker in created.values():
                saved = journey.command('speakers', 'show', speaker)
                saved['speaker']['state'] = 'inactive'
                journey.input('speakers', 'set', speaker, value={'expected_revision': saved['speaker']['revision'],
                              'speaker': saved['speaker'], 'aliases': []})
    return result


def acoustic(journey, corpus, model_id, *, checkpoint=lambda: None):
    """Measure real product profiles with supplied reader activity, never DER."""
    import acoustic_reference
    from journey_cpu import fetched_files
    independent_sources(corpus['clips'])
    source_root = Path(journey.args.reference_corpus).resolve().parent
    speakers = {reader: identity(journey, name) for reader, name in
                zip(corpus['selection']['enrolled_speakers'], ('Alex', 'Jordan'))}
    profile_revisions = {reader: 0 for reader in speakers}
    recordings, datasets = {}, {}
    receipt = {'corpus_sha256': acoustic_reference.sha(journey.args.reference_corpus),
               'supplied_voice_activity': True, 'diarization_accuracy': 'not-measured',
               'matching_settings': corpus['matching'], 'devices': [], 'independence': []}
    journey.receipt['acoustic'] = receipt
    for clip in corpus['clips']:
        snapshot = journey.base / clip['path']
        snapshot.write_bytes(acoustic_reference.file_bytes(source_root, clip['path'], clip['size_bytes'], clip['sha256']))
        media = install_recording(journey, snapshot, [{'voice': 'v1', 'text': clip['transcript'], 'start_ms': 0,
                                  'end_ms': clip['samples'] * 1000 // clip['sample_rate']}], clip['id'])
        recordings[clip['id']] = media
        if clip['role'] == 'enrollment':
            current = journey.command('recordings', 'show', media)['recording']
            journey.input('recordings', 'map-speaker', media, value={'revision': current['revision'],
                'mapping_revision': 0, 'local_speaker_id': 'v1', 'speaker_id': speakers[clip['speaker']],
                'document_digest': current['document_digest']})
            dataset = journey.input('models', 'dataset', 'create', value={'speaker_id': speakers[clip['speaker']],
                'recipe': {'recording_ids': [media], 'min_duration_us': 1000000}})
            if dataset['reference_count'] != 1: raise ValueError('Independent enrollment source count differs')
            datasets[clip['speaker']] = dataset['dataset']['id']
        else:
            journey.command('recordings', 'roster', 'add', media, '--expected-revision', '0',
                            *[part for speaker in speakers.values() for part in ('--speaker', speaker)])
    evaluations = [clip for clip in corpus['clips'] if clip['role'] != 'enrollment']
    for device in ('cpu', 'cuda'):
        row = {'device': device, 'state': 'running', 'profiles': [], 'cases': []}
        receipt['devices'].append(row)
        checkpoint()
        try:
            for clip in (clip for clip in corpus['clips'] if clip['role'] == 'enrollment'):
                reader = clip['speaker']
                queued = journey.input('models', 'train', value={'dataset_id': datasets[reader],
                    'name': 'Reference ' + reader + ' ' + device, 'output_kind': 'voice-embedding',
                    'base_model_id': model_id, 'adapter': ADAPTER, 'device': device})
                trained = journey.finish(queued)
                observed = device_receipt(trained['device_executions'], device)
                profile = journey.input('models', 'profile', 'set', speakers[reader], value={
                    'expected_revision': profile_revisions[reader], 'version_id': trained['version_id']})
                profile_revisions[reader] = profile['revision']
                files = fetched_files(journey, trained['version_id'], journey.base / ('profile-' + device + '-' + reader))
                row['profiles'].append({'fixture': clip['id'], 'source_sha256': clip['sha256'], 'reader': reader,
                    'speaker_id': speakers[reader], 'dataset_id': datasets[reader], 'version_id': trained['version_id'],
                    'work_id': queued.get('work_id', queued.get('id')), 'device_executions': observed, 'files': files})
                checkpoint()
            for clip in evaluations:
                queued = journey.input('recordings', 'match', recordings[clip['id']], value={
                    'model_id': model_id, 'adapter': ADAPTER, 'device': device, **corpus['matching']})
                matched = journey.finish(queued)
                observed = device_receipt(matched['device_executions'], device)
                if len(matched['decisions']) != 1:
                    raise ValueError('Supplied single-reader matching inventory differs')
                compared = acoustic_reference.compare_decision(clip['speaker'], matched['decisions'][0], speakers)
                row['cases'].append({'fixture': clip['id'], 'source_sha256': clip['sha256'], **compared,
                    'work_id': queued.get('work_id', queued.get('id')), 'device_executions': observed,
                    'decision': matched['decisions'][0]})
                checkpoint()
            row['state'] = 'completed'
        except (ValueError, RuntimeError, OSError, KeyError) as error:
            row.update(state='error', error=type(error).__name__.lower())
            completed = {case['fixture'] for case in row['cases']}
            for clip in evaluations:
                if clip['id'] not in completed:
                    row['cases'].append({'fixture': clip['id'], 'expected_reader': clip['speaker'],
                                         'outcome': 'error', 'error': row['error']})
        row['metrics'] = acoustic_metrics(row['cases'], list(speakers))
        checkpoint()
    for reader, speaker in speakers.items():
        predicted_only = journey.input('models', 'dataset', 'create', value={'speaker_id': speaker,
            'recipe': {'recording_ids': [recordings[clip['id']] for clip in evaluations]}})
        if predicted_only['reference_count'] != 0 or predicted_only['readiness'] != 'empty':
            raise ValueError('Acoustic predictions certified their own independent enrollment')
        receipt['independence'].append({'reader': reader, 'match_only_enrollment_references': 0})
    return receipt, speakers, recordings


def integrated(journey, configuration, corpus, model_id, speakers, recordings):
    """Real configured discovery enrolls only independently mapped licensed audio."""
    from journey_cpu import fetched_files
    receipt = {'state': 'running'}
    journey.receipt['integrated'] = receipt
    clip = next(clip for clip in corpus['clips'] if clip['role'] == 'enrollment')
    reader, source = clip['speaker'], recordings[clip['id']]
    audio = journey.base / 'integrated-context.wav'
    carrier(audio, 2, 'integrated-context')
    media = install_recording(journey, audio, [{'voice': 'v1', 'text': 'My name is Alex.',
                                               'start_ms': 0, 'end_ms': 2000}], 'integrated-context')
    journey.command('recordings', 'roster', 'add', media, '--speaker', speakers[reader], '--expected-revision', '0')
    queued = journey.input('speakers', 'run', media, value={'configuration': {
        'provider': configuration, 'apply': True, 'enroll': True, 'select_profile': True,
        'recipe': {'recording_ids': [source], 'min_duration_us': 1000000}},
        'training': {'adapter': ADAPTER, 'base_model_id': model_id, 'output_kind': 'voice-embedding', 'device': 'cuda'}})
    result = journey.finish(queued)
    if result['stage'] != 'complete' or result['application']['applied'] != 1 or len(result['enrollments']) != 1:
        raise ValueError('Configured real contextual enrollment did not complete')
    enrollment = result['enrollments'][0]
    if enrollment['state'] != 'enrolled' or enrollment['speaker_id'] != speakers[reader]:
        raise ValueError('Configured enrollment identity differs')
    training = journey.command('work', 'show', enrollment['work_id'])['result']
    observed = device_receipt(training['device_executions'], 'cuda')
    before = fetched_files(journey, enrollment['version_id'], journey.base / 'integrated-before')
    before_manifest = json.loads((journey.base / 'integrated-before/manifest.json').read_text(encoding='utf-8'),
                                 object_pairs_hook=reference.unique_object)
    receipt.update(work_id=queued.get('work_id', queued.get('id')), version_id=enrollment['version_id'],
                   device_executions=observed, independently_enrolled=True)
    selected = journey.command('models', 'speaker', 'show', enrollment['version_id'])['profile']
    if selected['usable'] is not True: raise ValueError('Fresh integrated profile is unavailable')
    current = journey.command('recordings', 'show', source)['recording']
    mapping = journey.command('recordings', 'mappings', source)['items']
    if len(mapping) != 1: raise ValueError('Independent source mapping inventory differs')
    correction = next(speaker for key, speaker in speakers.items() if key != reader)
    journey.input('recordings', 'map-speaker', source, value={'revision': current['revision'],
        'mapping_revision': mapping[0]['revision'], 'local_speaker_id': 'v1', 'speaker_id': correction,
        'document_digest': current['document_digest']})
    after = journey.command('models', 'speaker', 'show', enrollment['version_id'])['profile']
    if after['usable'] or after['usability'] != 'evidence-invalidated':
        raise ValueError('Supporting correction left the profile usable')
    receipt['correction_invalidates_current_profile'] = True
    retained = fetched_files(journey, enrollment['version_id'], journey.base / 'integrated-after')
    after_manifest = json.loads((journey.base / 'integrated-after/manifest.json').read_text(encoding='utf-8'),
                                object_pairs_hook=reference.unique_object)
    integrity = historical_outputs(before, retained, before_manifest, after_manifest)
    receipt.update(state='completed', context_source='synthetic semantic carrier, excluded from the training recipe',
                   enrollment_source_sha256=clip['sha256'], independent_manual_enrollment=True,
                   historical_model_bytes_unchanged=True, **integrity)
    return receipt
