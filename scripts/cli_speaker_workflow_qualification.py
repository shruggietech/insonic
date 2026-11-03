#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Installed shared-runtime speaker workflow using an owned loopback provider."""
import base64
from contextlib import ExitStack
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import queue
import threading
import uuid
from cli_corpus import finish_server
from documented_workflows import example
import qualification_timing as bounds
from cli_terminal_qualification import owned_runtime
from process_tree import ProcessTree


def qualify(call, write, recording_id, speaker_id, directory, version):
    state = {'abstain': False, 'fail_training': True, 'discovery_calls': 0, 'training_calls': 0, 'delay': False}
    started, released = threading.Event(), threading.Event()
    weights = b'owned contextual training fixture'

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            request = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
            if self.path == '/discover':
                state['discovery_calls'] += 1
                if state['delay']:
                    started.set()
                    if not released.wait(bounds.phase_bound(bounds.REQUEST_SECONDS)):
                        self.send_error(504)
                        return
                output = {'version': 1, 'observations': [
                    {'class': 'self-identification', 'name': 'Alex', 'voice_id': cue['voices'][0],
                     'references': [{'cue_id': cue['id'], 'quote': cue['text']}], 'uncertain': False}
                    for cue in request['context']['cues'] if cue.get('voices')]}
                if state['abstain']:
                    output['observations'] = []
            elif self.path == '/train':
                state['training_calls'] += 1
                if state['fail_training']:
                    self.send_error(503)
                    return
                output = {'contract_version': '1', 'kind': 'weights', 'architecture': 'custom',
                    'consumers': ['custom'], 'supported_operations': ['custom-inference'],
                    'artifacts': [{'role': 'weights.bin', 'format': 'binary', 'size': len(weights),
                        'sha256': hashlib.sha256(weights).hexdigest(), 'data': base64.b64encode(weights).decode()}]}
            else:
                self.send_error(404)
                return
            data = json.dumps(output).encode()
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(data)))
            try:
                self.end_headers()
                self.wfile.write(data)
            except (BrokenPipeError, ConnectionResetError):
                pass

        def log_message(self, *_args):
            pass

    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()

    def supplied(args, value, **kwargs):
        return call([*args, '--input', write('speaker-workflow-input.json', value)], **kwargs)

    def mappings():
        return call(['recordings', 'mappings', recording_id])['result']['items']

    try:
        # This controlled one-voice fixture explicitly supplies its known interval.
        # Imported display labels alone intentionally remain untimed in the product.
        exported = call(['recordings', 'document', recording_id])['result']
        if not exported['complete']:
            raise ValueError('Owned fixture document unexpectedly required paging')
        document = json.loads(base64.b64decode(exported['data']))
        for cue in document['cues']:
            for attribution in cue.get('speaker_attributions', []):
                attribution.update(start_milliseconds=cue['timing']['start_milliseconds'],
                                   end_milliseconds=cue['timing']['end_milliseconds'])
        source = write('timed speaker fixture.cueson.json', document)
        imported = call(['transcript', 'import', source, '--record', recording_id, '--replace-transcript'])['result']
        call(['work', 'wait', imported['work_id']])
        endpoint = f'http://127.0.0.1:{server.server_port}'
        config = example(version, 'pipelines', 'speaker-workflow.json')
        config['speaker_workflow']['provider']['endpoint'] = endpoint + '/discover'
        config['speaker_workflow']['provider']['model'] = 'owned-context-fixture'
        training = example(version, 'voice-models', 'training-hosted.json')
        config['speaker_training'] = {key: training[key] for key in ['adapter', 'output_kind']}
        config['speaker_training']['adapter']['endpoint'] = endpoint + '/train'
        config['speaker_workflow'].update(enroll=True, select_profile=False)
        pipeline_id = str(uuid.uuid4())
        saved = supplied(['pipelines', 'set', pipeline_id], {'expected_revision': 0, 'pipeline': {
            'id': pipeline_id, 'name': 'Owned contextual workflow', 'preset': 'custom', 'configuration': config}})['result']
        supplied(['pipelines', 'inspect', pipeline_id], {'revision': saved['revision']})
        election = {'pipeline_id': pipeline_id, 'pipeline_revision': saved['revision']}
        inline_probe = {'configuration': config['speaker_workflow'] | {'enroll': False, 'select_profile': False}}
        identity = call(['speakers', 'show', speaker_id])['result']
        identity['speaker']['state'] = 'inactive'
        identity = supplied(['speakers', 'set', speaker_id], {'expected_revision': identity['speaker']['revision'],
            'speaker': identity['speaker'], 'aliases': []})['result']
        probe = supplied(['speakers', 'run', recording_id], inline_probe)['result']
        outcome = call(['work', 'wait', probe['work_id']])['result']['result']['application']
        if outcome['unknown'] != 1 or mappings():
            raise ValueError('Unregistered contextual identity invented a registered assignment')
        identity['speaker']['state'] = 'active'
        supplied(['speakers', 'set', speaker_id], {'expected_revision': identity['speaker']['revision'],
            'speaker': identity['speaker'], 'aliases': []})
        probe = supplied(['speakers', 'run', recording_id], inline_probe)['result']
        call(['work', 'wait', probe['work_id']])
        before_probe = mappings()
        if len(before_probe) != 1:
            raise ValueError('Initial contextual mapping was not accepted')
        state['abstain'] = True
        probe = supplied(['speakers', 'run', recording_id], inline_probe)['result']
        outcome = call(['work', 'wait', probe['work_id']])['result']['result']['application']
        if outcome['unknown'] != 1 or mappings() != before_probe:
            raise ValueError('Unknown discovery erased accepted context')
        state['abstain'] = False
        duplicate = str(uuid.uuid4())
        identity = supplied(['speakers', 'set', duplicate], {'expected_revision': 0, 'speaker': {
            'id': duplicate, 'name': 'Alex', 'state': 'active'}, 'aliases': []})['result']
        probe = supplied(['speakers', 'run', recording_id], inline_probe)['result']
        outcome = call(['work', 'wait', probe['work_id']])['result']['result']['application']
        if outcome['ambiguous'] != 1 or mappings() != before_probe:
            raise ValueError('Ambiguous discovery invented or erased an assignment')
        identity['speaker']['state'] = 'inactive'
        supplied(['speakers', 'set', duplicate], {'expected_revision': identity['speaker']['revision'],
            'speaker': identity['speaker'], 'aliases': []})
        state['discovery_calls'] = 0
        queued = supplied(['speakers', 'run', recording_id], election)['result']
        failed = call(['work', 'wait', queued['work_id']], expected=1)['result']
        if failed['result']['stage'] != 'enrollment' or failed['result']['application']['applied'] != 1:
            raise ValueError('Workflow did not retain accepted application before training failure')
        accepted = mappings()
        state['fail_training'] = False
        call(['work', 'retry', queued['work_id']])
        done = call(['work', 'wait', queued['work_id']])['result']
        if state['discovery_calls'] != 1 or mappings() != accepted or len(done['result']['enrollments']) != 1:
            raise ValueError('Retry repeated discovery/application or lost enrollment')
        enrollment = done['result']['enrollments'][0]
        if enrollment['state'] != 'enrolled':
            raise ValueError('Configured training did not complete')
        human = call(['work', 'show', queued['work_id']], machine=False)
        if 'Speaker workflow stage: complete' not in human or 'Contextual mappings:' not in human:
            raise ValueError('Human workflow summary lost evidence classes')
        # Revoking current evidence retires its dataset, while exact weights remain retrievable.
        recording = call(['recordings', 'show', recording_id])['result']['recording']
        supplied(['speakers', 'retract', recording_id], {'expected_recording_revision': recording['revision'],
            'mappings': [{'id': row['id'], 'revision': row['revision']} for row in accepted]})
        dataset = call(['models', 'dataset', 'show', enrollment['dataset_id']])['result']
        if dataset['dataset']['state'] != 'invalidated':
            raise ValueError('Retraction retained current training authority')
        destination = directory / 'historical contextual weights'
        call(['models', 'speaker', 'fetch', enrollment['version_id'], '--destination', destination])
        if not any(p.is_file() and p.read_bytes() == weights for p in destination.rglob('*')):
            raise ValueError('Retraction discarded immutable trained bytes')
        # Cancellation must revoke the running discovery child and remain retryable.
        state['delay'] = True
        inline = {'configuration': config['speaker_workflow'] | {'enroll': False, 'select_profile': False}}
        pending = supplied(['speakers', 'run', recording_id], inline)['result']
        if not started.wait(bounds.phase_bound(bounds.READINESS_SECONDS)):
            raise ValueError('Workflow child never reached the elected provider')
        call(['work', 'cancel', pending['work_id']])
        released.set()
        cancelled = call(['work', 'show', pending['work_id']])['result']
        if cancelled['state'] != 'cancelled' or mappings():
            raise ValueError('Cancelled workflow accepted a mapping')
        state['delay'] = False
        call(['work', 'retry', pending['work_id']])
        call(['work', 'wait', pending['work_id']])
        current = mappings()
        if len(current) != 1 or current[0]['origin'] != 'contextual':
            raise ValueError('Cancelled workflow did not resume')
        # An owner correction wins over a fresh contextual rerun.
        row = current[0]
        supplied(['recordings', 'map-speaker', recording_id], {'revision': recording['revision'],
            'mapping_revision': row['revision'], 'local_speaker_id': row['local_speaker_id'],
            'speaker_id': speaker_id, 'document_digest': recording['document_digest']})
        queued = supplied(['speakers', 'run', recording_id], inline)['result']
        final = call(['work', 'wait', queued['work_id']])['result']
        if final['result']['application']['preserved'] != 1 or mappings()[0]['origin'] != 'manual':
            raise ValueError('Rerun overwrote the owner correction')
        return {'state': 'passed', 'adapter': 'owned-loopback-fixture', 'acoustic_quality': 'not-measured',
            'cases': ['unsupported-identity', 'unknown', 'ambiguity', 'saved-pipeline', 'partial-failure-retry', 'contextual-enrollment', 'human-json',
                      'retraction', 'historical-fetch', 'cancel-retry', 'owner-correction-rerun']}
    finally:
        released.set()
        finish_server(server, thread)


def qualify_restart(invoke, directory, environment, audio, subtitle, *, executable, process_factory=None, workspace=None):
    """Interrupt only an explicitly owned runtime, then resume the same parent."""
    library = workspace if workspace is not None else directory / 'owned workflow restart library'
    selected = dict(environment)
    for marker in ['CI', 'GITHUB_ACTIONS', 'TF_BUILD', 'BUILD_BUILDID', 'JENKINS_URL']:
        selected.pop(marker, None)
    started, released = threading.Event(), threading.Event()
    state = {'delay': True, 'calls': 0}

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            self.rfile.read(int(self.headers['Content-Length']))
            state['calls'] += 1
            if state['delay']:
                started.set()
                if not released.wait(bounds.phase_bound(bounds.REQUEST_SECONDS)):
                    self.send_error(504)
                    return
            body = b'{"version":1,"observations":[]}'
            self.send_response(200)
            self.send_header('Content-Length', str(len(body)))
            try:
                self.end_headers()
                self.wfile.write(body)
            except (BrokenPipeError, ConnectionResetError):
                pass

        def log_message(self, *_args):
            pass

    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()

    def call(args, **options):
        return invoke(args, workspace_path=library, execution_environment=selected, **options)

    def start_owner(stack):
        startup_id = str(uuid.uuid4())
        owner = stack.enter_context(owned_runtime(
            [str(executable), '--workspace', str(library), '--startup-id', startup_id, '--json', 'runtime', 'serve'],
            cwd=directory, env=selected, gates={'provider': {'release': released}}, factory=process_factory or ProcessTree))
        mailbox = queue.Queue(maxsize=1)
        reader = threading.Thread(target=lambda: mailbox.put(owner.process.stdout.readline(4097)), daemon=True)
        reader.start()
        try:
            report = json.loads(mailbox.get(timeout=bounds.phase_bound(bounds.READINESS_SECONDS)))
        except BaseException:
            owner.kill()
            raise
        finally:
            reader.join(bounds.CLEANUP_SECONDS)
            if reader.is_alive():
                raise RuntimeError('Workflow runtime readiness reader did not release')
        session = call(['workspace', 'show'])
        if (report.get('startup_id') != startup_id or report.get('state') != 'ready'
                or report.get('workspace_id') != session.get('result', {}).get('workspace_id')
                or not session.get('runtime_session_id')):
            raise ValueError('Owned workflow runtime readiness differs')
        return owner, session['runtime_session_id']

    try:
        if workspace is None:
            call(['workspace', 'init', library], scoped=False)
            call(['credentials', 'select', 'session'])
        with ExitStack() as owners:
            first, session = start_owner(owners)
            queued = call(['media', 'import', audio, '--no-pipeline'])['result']
            recording_id = call(['work', 'wait', queued['work_id']])['result']['result']['items'][0]['media_id']
            queued = call(['transcript', 'import', subtitle, '--record', recording_id])['result']
            call(['work', 'wait', queued['work_id']])
            path = directory / 'restart workflow.json'
            path.write_text(json.dumps({'configuration': {'provider': {
                'enabled': True, 'adapter': 'insonic-http', 'route': 'local',
                'endpoint': f'http://127.0.0.1:{server.server_port}/discover', 'model': 'owned-restart-fixture'},
                'apply': False, 'enroll': False, 'select_profile': False}}) + '\n', encoding='utf-8', newline='\n')
            queued = call(['speakers', 'run', recording_id, '--input', path])['result']
            if not started.wait(bounds.phase_bound(bounds.READINESS_SECONDS)):
                raise ValueError('Workflow did not reach the interruption boundary')
            first.kill()
            first.process.communicate(timeout=bounds.CLEANUP_SECONDS)
            if not first.wait_for_descendants(bounds.CLEANUP_SECONDS):
                raise RuntimeError('Interrupted workflow descendants did not release')
            state['delay'] = False
            released.set()
            second, resumed_session = start_owner(owners)
            done = call(['work', 'wait', queued['work_id']])['result']
            if (session == resumed_session or done['state'] != 'succeeded'
                    or done['result']['stage'] != 'complete' or state['calls'] != 2
                    or call(['recordings', 'mappings', recording_id])['result']['items']):
                raise ValueError('Workflow restart lost identity, repeated accepted work, or invented a mapping')
            second.kill()
            second.process.communicate(timeout=bounds.CLEANUP_SECONDS)
            if not second.wait_for_descendants(bounds.CLEANUP_SECONDS):
                raise RuntimeError('Resumed workflow descendants did not release')
        return {'state': 'passed', 'work_id': queued['work_id'], 'provider_attempts': state['calls'],
                'runtime_session_changed': True, 'owned_process_cleanup': True}
    finally:
        released.set()
        finish_server(server, thread)


def qualify_backend(executable, workspace, directory, environment, version):
    """Use the already provisioned disposable backend matrix, without changing adapters."""
    from cli_contract_qualification import capture_invocation
    from cli_media_qualification import pcm_fixture

    def call(args, *, expected=0, machine=True, workspace_path=None, execution_environment=None, **options):
        argv = [str(executable)] + (['--json'] if machine else [])
        argv += ['--workspace', str(workspace_path or workspace)] + [str(arg) for arg in args]
        if args[:2] == ['work', 'wait']:
            argv += ['--timeout-ms', str(bounds.DURABLE_WORK_SECONDS * 1000)]
        out, err, code, _ = capture_invocation(ProcessTree, argv, cwd=directory,
            env=execution_environment or environment, scoped=True, input=None,
            timeout_seconds=options.get('timeout_seconds'))
        if code != expected or (err if code == 0 else out):
            raise ValueError('Backend speaker CLI failed: ' + ' '.join(map(str, args[:3]))
                             + ': ' + (err if code else out).decode('utf-8')[:2000])
        raw = err if code else out
        return json.loads(raw) if machine else raw.decode('utf-8')

    def write(name, value):
        path = directory / name
        path.write_text(json.dumps(value) + '\n', encoding='utf-8', newline='\n')
        return path

    audio, subtitle = directory / 'speaker.wav', directory / 'speaker.vtt'
    pcm_fixture(audio, 1)
    subtitle.write_text('WEBVTT\n\n00:00:00.000 --> 00:00:00.500\n<v Alex>I am Alex.\n\n',
                        encoding='utf-8', newline='\n')
    # Start with an owned runtime before any auto-started shared-runtime client.
    restart = qualify_restart(call, directory, environment, audio, subtitle,
                              executable=executable, workspace=workspace)
    queued = call(['media', 'import', audio, '--no-pipeline'])['result']
    recording = call(['work', 'wait', queued['work_id']])['result']['result']['items'][0]['media_id']
    queued = call(['transcript', 'import', subtitle, '--record', recording])['result']
    call(['work', 'wait', queued['work_id']])
    speaker_id = str(uuid.uuid4())
    call(['speakers', 'set', speaker_id, '--input', write('backend speaker.json', {
        'expected_revision': 0, 'speaker': {'id': speaker_id, 'name': 'Alex', 'state': 'active'}, 'aliases': []})])
    receipt = qualify(call, write, recording, speaker_id, directory, version)
    receipt['restart'] = restart
    return receipt
