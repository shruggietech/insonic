#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Explicit connected CLI qualification using owned scratch and selected models.

Both modes require maintainer election outside CI. Controlled mode runs no
acoustic models. Real mode uses already installed pinned offline models only.
No product release, deployment or operator workspace is modified.
"""
import argparse
import base64
import hashlib
import http.server
import importlib.util
import json
import os
from pathlib import Path
import platform
import subprocess
import sys
import tempfile
import threading
import time
import uuid

ROOT = Path(__file__).resolve().parents[1]
module_spec = importlib.util.spec_from_file_location('journey_engine', ROOT / 'scripts/qualify-processing.py')
engine = importlib.util.module_from_spec(module_spec)
module_spec.loader.exec_module(engine)


def require_mode(mode):
    if mode not in ('real', 'controlled'):
        raise ValueError('Journey requires explicit --mode real or controlled')
    if any(os.environ.get(name) for name in engine.CI_VARIABLES):
        raise ValueError('Connected elected worker qualification is outside CI')


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False, allow_nan=False) + '\n', encoding='utf-8', newline='\n')


def withdraw_source(source, base):
    if source.is_symlink():
        raise ValueError('Source withdrawal is restricted to the owned scratch workspace')
    source, base = source.resolve(strict=True), base.resolve(strict=True)
    if source != base / 'source-workspace' or source.is_symlink():
        raise ValueError('Source withdrawal is restricted to the owned scratch workspace')
    destination = base / 'unavailable-source-workspace'
    if destination.exists():
        raise ValueError('Owned unavailable destination already exists')
    source.rename(destination)
    return destination


def compare_authority(before, after):
    if before != after:
        raise ValueError('Restored current authority differs from completed source authority')


class Journey:
    def __init__(self, args, base):
        self.args, self.base = args, base
        self.workspace = base / 'source-workspace'
        self.cli = Path(args.cli).resolve(strict=True)
        self.started = time.monotonic()
        self.deadline = self.started + args.timeout
        self.sequence = 0
        self.runtime = None
        self.environment = engine.local_environment()
        if os.environ.get('INSONIC_NATIVE_LIBRARY'):
            self.environment['INSONIC_NATIVE_LIBRARY'] = os.environ['INSONIC_NATIVE_LIBRARY']
        home = base / 'private-user'
        home.mkdir()
        engine.secure_private_directory(str(home))
        for name in ('roaming', 'local', 'cache', 'config'):
            (home / name).mkdir()
        self.environment.update(HOME=str(home), USERPROFILE=str(home), APPDATA=str(home / 'roaming'),
                                LOCALAPPDATA=str(home / 'local'), XDG_CACHE_HOME=str(home / 'cache'),
                                XDG_CONFIG_HOME=str(home / 'config'), PYTHONIOENCODING='utf-8',
                                GH_PROMPT_DISABLED='1', GIT_TERMINAL_PROMPT='0')
        revision = engine._child(['git', 'rev-parse', 'HEAD'], self.environment, timeout=10).decode().strip()
        dirty = bool(engine._child(['git', 'status', '--porcelain'], self.environment, timeout=10).strip())
        identity = hashlib.sha256()
        inputs = [ROOT / name for name in ('VERSION', 'go.mod', 'go.sum')]
        for prefix in ('internal', 'cmd', 'scripts', 'schemas/v' + (ROOT / 'VERSION').read_text().strip()):
            inputs.extend(path for path in (ROOT / prefix).rglob('*') if path.is_file() and path.suffix in ('.go', '.py', '.json'))
        inputs.append(ROOT / 'tests/fixtures/qualification-worker.py')
        for path in sorted(inputs):
            identity.update(path.relative_to(ROOT).as_posix().encode() + b'\0' + bytes.fromhex(engine.sha(path)))
        self.receipt = {'kind': 'connected-product-qualification', 'mode': args.mode, 'required_check': False,
                        'platform': platform.system(), 'device': 'cpu', 'cli_sha256': engine.sha(self.cli),
                        'worker_sha256': engine.sha(ROOT / ('scripts/processing-worker.py' if args.mode == 'real' else 'tests/fixtures/qualification-worker.py')),
                        'product_version': (ROOT / 'VERSION').read_text().strip(),
                        'source_revision': revision, 'source_dirty': dirty, 'source_inputs_sha256': identity.hexdigest(),
                        'operation_bound_seconds': 180, 'overall_bound_seconds': args.timeout,
                        'hard_rss_limit_bytes': None, 'operations': [], 'records': []}

    def remaining(self, maximum=180):
        remaining = self.deadline - time.monotonic()
        if remaining <= 0:
            raise ValueError('Journey overall deadline exceeded')
        return min(maximum, remaining)

    def command(self, *args, workspace=True, maximum=180):
        command = [self.cli, '--json']
        if workspace:
            command.extend(('--workspace', self.workspace))
        command.extend(args)
        print(json.dumps({'operation': str(args[0]) + '.' + str(args[1])}), flush=True)
        try:
            raw = engine._child(command, self.environment, timeout=self.remaining(maximum), max_output=16 << 20, cli_response=True)
        except engine.CLIResponseError as error:
            result = error.response.get('result', {})
            self.receipt['failure_code'] = error.response['error']['code']
            self.receipt['failed_operation'] = str(args[0]) + '.' + str(args[1])
            self.receipt['failed_work'] = {key: result.get(key) for key in ('kind', 'state', 'phase', 'error')} if isinstance(result, dict) else None
            if isinstance(result, dict) and isinstance(result.get('result'), dict):
                self.receipt['failed_work_result'] = {key: result['result'].get(key) for key in ('state', 'error')}
            raise
        response = json.loads(raw)
        if response.get('error'):
            raise ValueError('CLI qualification operation failed: ' + response['error']['code'])
        return response.get('result', response)

    def payload(self, value):
        self.sequence += 1
        path = self.base / 'inputs' / f'{self.sequence}.json'
        write_json(path, value)
        return path

    def input(self, *args, value):
        return self.command(*args, '--input', self.payload(value))

    def finish(self, queued):
        started = time.monotonic()
        work_id = queued.get('work_id', queued.get('id'))
        if not work_id:
            raise ValueError('Operation did not return durable work identity')
        current = self.command('work', 'wait', work_id, '--timeout-ms', str(int(self.remaining() * 1000)), maximum=185)
        if current.get('state') != 'succeeded':
            raise ValueError('Work did not reach success')
        self.receipt['operations'].append({'kind': current['kind'], 'state': current['state'], 'phase': current['phase'],
                                           'wait_wall_seconds': time.monotonic() - started})
        return current['result']

    def start_runtime(self):
        self.runtime = subprocess.Popen([str(self.cli), '--workspace', str(self.workspace), 'runtime', 'serve'],
                                        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                        env=self.environment, creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)

    def idle_runtime(self):
        if self.runtime is not None:
            self.runtime.wait(timeout=self.remaining(45))
            if self.runtime.returncode:
                raise ValueError('Owned runtime failed before maintenance')
            self.runtime = None

    def query(self, operation, filters=None):
        result = self.input('query', 'run', value={'definition': {'mode': 'normalized', 'operation': operation,
                            'filters': filters or {}, 'pagination': {'limit': 100}}})
        if result.get('graph_error'):
            raise ValueError('Connected query fell back from its native graph')
        return result

    def current_record(self, media_id):
        shown = self.command('recordings', 'show', media_id)
        record = shown['recording']
        if record['state'] != 'ready' or not record['document']:
            raise ValueError('Processing did not publish a current document')
        page = self.input('recordings', 'document', media_id, value={'document_digest': record['document_digest'], 'limit': 65536})
        if not page['complete'] or hashlib.sha256(base64.b64decode(page['data'])).hexdigest() != record['document_digest']:
            raise ValueError('Current document bytes differ from accepted digest')
        return record

    def authority(self, ids, speaker_id, version_id):
        recordings = {}
        for media_id in ids:
            entry = self.command('media', 'show', media_id)
            entry = entry.get('entry', entry)
            record = self.current_record(media_id)
            recordings[media_id] = {'media': {key: entry[key] for key in ('id', 'digest', 'asset_id', 'duration_us', 'facts', 'dates')},
                                   'recording': {key: record[key] for key in ('id', 'document', 'document_digest', 'source_map', 'source_digest')},
                                   'roster': self.command('recordings', 'roster', 'show', media_id)}
        model = self.command('models', 'speaker', 'show', version_id)
        listing = self.input('models', 'speaker', 'list', value={'speaker_id': speaker_id})
        return {'recordings': recordings, 'model': model, 'profile': listing.get('profile')}

    def run(self):
        actual_version = engine._child([self.cli, 'version'], self.environment, timeout=10).decode().strip()
        if actual_version != self.receipt['product_version']:
            raise ValueError('Selected CLI version differs from current source')
        self.command('workspace', 'init', self.workspace, workspace=False)
        self.start_runtime()
        capabilities = self.command('graph', 'capabilities')
        if capabilities.get('state') != 'available' or capabilities.get('adapter_id') != 'ladybugdb':
            raise ValueError('Journey requires the actual native LadybugDB graph')
        self.receipt['graph'] = capabilities
        version = self.receipt['product_version']
        tools = json.loads(Path(self.args.tools).read_text(encoding='utf-8'))
        tools['schema_version'] = version
        self.command('media', 'tools', self.payload(tools))
        cueson = Path(self.args.cueson).resolve(strict=True)
        worker = ROOT / ('scripts/processing-worker.py' if self.args.mode == 'real' else 'tests/fixtures/qualification-worker.py')
        pythons = {'transcribe': Path(self.args.recognition_python).resolve(strict=True),
                   'diarize': Path(self.args.diarization_python).resolve(strict=True)}
        processing = {'threads': 2, 'timeout_ms': 120000, 'max_duration_us': 60000000, 'max_input_bytes': 67108864,
                      'worker': {'path': str(worker), 'sha256': engine.sha(worker)}}
        for operation, field in (('transcribe', 'recognition_python'), ('diarize', 'diarization_python')):
            processing[field] = {'path': str(pythons[operation]), 'sha256': engine.sha(pythons[operation])}
        self.command('processing', 'tools', self.payload({'kind': 'processing-tools', 'schema_version': version,
                     'cueson': {'executable': str(cueson), 'executable_sha256': engine.sha(cueson)}, 'processing': processing}))
        served, pins = {}, {}
        for operation, pin in engine.PINS.items():
            if self.args.mode == 'real':
                root = engine.verify_model(operation, getattr(self.args, 'recognition_model' if operation == 'transcribe' else 'diarization_model'))
                files = pin['files']
            else:
                root = self.base / 'controlled-models' / operation
                files = {}
                for role in pin['files']:
                    path = root / role
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_bytes(b'controlled inert model bytes')
                    files[role] = (path.stat().st_size, engine.sha(path))
            pins[operation] = {**pin, 'files': files}
            served.update({f'/{operation}/{role}': root / role for role in files})

        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                path = served.get(self.path)
                if path is None:
                    self.send_error(404); return
                self.send_response(200)
                self.send_header('Content-Length', str(path.stat().st_size)); self.end_headers()
                with path.open('rb') as source:
                    while chunk := source.read(65536): self.wfile.write(chunk)
            def log_message(self, *args): pass

        server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
        model_ids = {}
        try:
            for operation, pin in pins.items():
                capabilities = ['transcription'] if operation == 'transcribe' else ['diarization', 'speaker-model-training', 'voice-matching']
                manifest = {'kind': 'base-model-manifest', 'schema_version': version, 'name': pin['id'],
                            'model_version': pin['revision'], 'upstream_revision': pin['revision'], 'license': pin['license'],
                            'capabilities': capabilities, 'files': [{'role': role, 'size': size, 'sha256': digest,
                            'url': f'http://127.0.0.1:{server.server_port}/{operation}/{role}', 'local_http': True}
                            for role, (size, digest) in pin['files'].items()]}
                path = self.payload(manifest)
                registered = self.finish(self.command('models', 'register', path))
                acquired = self.finish(self.command('models', 'acquire', path))
                if registered['model_id'] != acquired['model_id'] or not self.command('models', 'verify', acquired['model_id'])['verified']:
                    raise ValueError('Exact loopback model acquisition did not verify')
                model_ids[operation] = acquired['model_id']
        finally:
            server.shutdown(); server.server_close(); thread.join(timeout=5)

        ffmpeg = tools['ffmpeg']
        if engine.sha(ffmpeg['path']) != ffmpeg['sha256']:
            raise ValueError('Selected media decoder digest differs')
        crop = self.base / 'cross-clip.wav'
        source = ROOT / 'tests/fixtures/media/speech.flac'
        engine._child([ffmpeg['path'], '-nostdin', '-hide_banner', '-loglevel', 'error', '-ss', '1', '-i', source,
                       '-t', '8', '-ac', '1', '-ar', '16000', '-c:a', 'pcm_s16le', crop], self.environment, timeout=30)
        ids = []
        for name, source, channel in (('speech', source, None), ('cross-clip', crop, None),
                                      ('sintel-dialogue', ROOT / 'tests/fixtures/media/sintel-dialogue.mkv', 2)):
            imported = self.finish(self.command('media', 'import', source, '--originated-at', '2020-01-01T00:00:00.123456789Z'))
            media_id = imported['items'][0]['media_id']; ids.append(media_id)
            entry = self.command('media', 'show', media_id); entry = entry.get('entry', entry)
            stream_index = entry['facts']['canonical']['tracks'][0]['index']
            options = {'transcription': 'generate', 'diarization': 'run', 'recognition_model_id': model_ids['transcribe'],
                       'diarization_model_id': model_ids['diarize'], 'recognition': {'device': 'cpu', 'language': 'en'},
                       'attribution': {'device': 'cpu'}, 'audio': {'stream_index': stream_index}}
            if channel is not None: options['audio']['channel'] = channel
            self.finish(self.input('recordings', 'process', media_id, value=options))
            record = self.current_record(media_id)
            self.receipt['records'].append({'fixture': name, 'media_id': media_id, 'document_digest': record['document_digest'],
                                           'source_map': record['source_map'], 'provenance': record['provenance'],
                                           'diagnostics': record['diagnostics']})
        training = self.current_record(ids[0])
        voices = sorted({attribution['speaker_id'] for cue in training['document']['cues'] for attribution in cue.get('speaker_attributions', [])})
        if not voices: raise ValueError('Generated speech has no enrollment intervals')
        speaker_id = str(uuid.uuid4())
        self.input('speakers', 'set', speaker_id, value={'expected_revision': 0,
                   'speaker': {'id': speaker_id, 'name': 'Qualification speaker', 'revision': 0, 'state': 'active'}, 'aliases': []})
        self.input('recordings', 'map-speaker', ids[0], value={'revision': training['revision'], 'mapping_revision': 0,
                   'local_speaker_id': voices[0], 'speaker_id': speaker_id, 'document_digest': training['document_digest']})
        dataset = self.input('models', 'dataset', 'create', value={'speaker_id': speaker_id})
        adapter = {'id': 'pyannote-profile', 'contract_version': '1', 'mode': 'local', 'architecture': 'pyannote',
                   'output_kinds': ['voice-embedding'], 'consumers': ['voice-matching'], 'supported_operations': ['voice-matching']}
        trained = self.finish(self.input('models', 'train', value={'dataset_id': dataset['dataset']['id'], 'name': 'Qualification profile',
                              'output_kind': 'voice-embedding', 'base_model_id': model_ids['diarize'], 'adapter': adapter}))
        version_id = trained['version_id']
        self.input('models', 'profile', 'set', speaker_id, value={'expected_revision': 0, 'version_id': version_id})
        roster = self.command('recordings', 'roster', 'show', ids[1])
        self.command('recordings', 'roster', 'add', ids[1], '--speaker', speaker_id, '--expected-revision', str(roster['revision']))
        matched = self.finish(self.input('recordings', 'match', ids[1], value={'model_id': model_ids['diarize'], 'adapter': adapter,
                             'threshold': 0.75, 'ambiguity_margin': 0.1, 'min_evidence_us': 1000000}))
        self.receipt['speaker_model'] = {'dataset_id': dataset['dataset']['id'], 'version_id': version_id,
                                         'enrollment': 'duration-weighted normalized embeddings; no base-weight retraining',
                                         'matching_result': matched, 'quality_claim': 'Cross-clip consistency, not independent identification accuracy'}
        self.command('models', 'speaker', 'fetch', version_id, '--destination', self.base / 'fetched-source')
        before = self.authority(ids, speaker_id, version_id)
        before_queries = {operation: self.query(operation) for operation in ('media-list', 'speaker-search', 'model-list')}
        search = 'Controlled' if self.args.mode == 'controlled' else 'we'
        before_queries['text-search'] = self.query('text-search', {'text': search})
        if any(not result.get('items') for result in before_queries.values()):
            raise ValueError('Connected query returned no expected current data')
        self.idle_runtime()
        bundle = self.base / 'backup'
        self.command('backup', 'create', bundle, '--mode', 'self-contained')
        self.command('backup', 'verify', bundle)
        withdraw_source(self.workspace, self.base)
        self.workspace = self.base / 'restored-workspace'
        self.command('workspace', 'init', self.workspace, workspace=False)
        self.command('backup', 'restore', bundle)
        self.start_runtime()
        after = self.authority(ids, speaker_id, version_id)
        compare_authority(before, after)
        self.command('models', 'speaker', 'fetch', version_id, '--destination', self.base / 'fetched-restored')
        source_files = {path.name: engine.sha(path) for path in (self.base / 'fetched-source').iterdir() if path.is_file()}
        restored_files = {path.name: engine.sha(path) for path in (self.base / 'fetched-restored').iterdir() if path.is_file()}
        compare_authority(source_files, restored_files)
        after_queries = {operation: self.query(operation, {'text': search} if operation == 'text-search' else None)
                         for operation in before_queries}
        compare_authority({op: result['items'] for op, result in before_queries.items()},
                          {op: result['items'] for op, result in after_queries.items()})
        self.receipt.update(state='passed', source_store='unavailable at original path', authority_comparison='exact values and hashes preserved',
                            query_operations=list(before_queries), queried_after_restore=True,
                            retrieved_model_files=source_files, acoustic_quality='measured separately' if self.args.mode == 'real' else 'not measured',
                            wall_seconds=time.monotonic() - self.started)
        self.idle_runtime()
        self.receipt['wall_seconds'] = time.monotonic() - self.started
        return self.receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--mode', choices=('real', 'controlled'), required=True)
    parser.add_argument('--cli', required=True)
    parser.add_argument('--tools', required=True)
    parser.add_argument('--cueson', required=True)
    parser.add_argument('--recognition-python', required=True)
    parser.add_argument('--diarization-python', required=True)
    parser.add_argument('--recognition-model')
    parser.add_argument('--diarization-model')
    parser.add_argument('--timeout', type=int, default=900)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    require_mode(args.mode)
    if not 60 <= args.timeout <= 3600: raise ValueError('Invalid journey bound')
    if args.mode == 'real' and (not args.recognition_model or not args.diarization_model): raise ValueError('Real mode requires exact installed models')
    with tempfile.TemporaryDirectory(prefix='insonic-connected-journey-') as temporary:
        journey = Journey(args, Path(temporary))
        try:
            receipt = journey.run()
        except Exception as error:
            failed = {**journey.receipt, 'state': 'failed', 'failure_type': type(error).__name__,
                      'wall_seconds': time.monotonic() - journey.started}
            write_json(Path(args.output).resolve(), failed)
            raise
        finally:
            if journey.runtime is not None:
                try: journey.runtime.wait(timeout=40)
                except subprocess.TimeoutExpired:
                    journey.runtime.terminate(); journey.runtime.wait(timeout=10)
    write_json(Path(args.output).resolve(), receipt)
    print(json.dumps({'state': receipt['state'], 'mode': args.mode, 'wall_seconds': receipt['wall_seconds'], 'receipt': str(Path(args.output).resolve())}))


if __name__ == '__main__':
    main()
