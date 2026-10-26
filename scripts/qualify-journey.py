#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Explicit connected CLI qualification using owned scratch and selected models.

Both modes require maintainer election outside CI. Controlled mode runs no
acoustic models. Real mode uses already installed pinned offline models only.
No product release, deployment or operator workspace is modified.
"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing
import argparse
import base64
import hashlib
import http.server
import importlib.util
import json
import os
import platform
import re
import subprocess
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
    data = json.dumps(value, indent=2, ensure_ascii=False, allow_nan=False) + '\n'
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(mode='w', encoding='utf-8', newline='\n', dir=path.parent, prefix='.' + path.name + '.', delete=False) as stream:
            temporary = Path(stream.name)
            stream.write(data)
        temporary.replace(path)
    finally:
        if temporary is not None:
            temporary.unlink(missing_ok=True)


def package_module():
    spec = importlib.util.spec_from_file_location('journey_package', ROOT / 'scripts/package-desktop.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def validate_selection(args):
    if args.package:
        if any((args.cli, args.tools, args.cueson)):
            raise ValueError('Package selection forbids development CLI or companion overrides')
        if not args.package_receipt or not args.revision:
            raise ValueError('Package selection requires its receipt and exact elected revision')
    elif not all((args.cli, args.tools, args.cueson)) or args.package_receipt or args.revision:
        raise ValueError('Source selection requires CLI, media tools and Cueson only')


def validate_package_election(receipt, revision, version, platform_key):
    if not re.fullmatch('[0-9a-f]{40}', revision or '') or receipt.get('revision') != revision:
        raise ValueError('Package revision differs from exact election')
    if any(receipt.get(name) != version for name in ('schema_version', 'product_version', 'documentation_version')):
        raise ValueError('Package versions differ from current product contract')
    if receipt.get('source_dirty') is not False or receipt.get('platform') != platform_key or receipt.get('variant') not in ('cli', 'desktop'):
        raise ValueError('Package requires clean source and the current supported platform and variant')


def packaged_environment():
    environment = engine.local_environment()
    environment.update(package_module().package_environment())
    return environment


def select_package(args, base):
    package = package_module()
    archive = Path(args.package).resolve(strict=True)
    receipt = json.loads(Path(args.package_receipt).read_text(encoding='utf-8'))
    validate_package_election(receipt, args.revision, (ROOT / 'VERSION').read_text().strip(), package.platform_key())
    package.verify_archive_receipt(archive, receipt, expected_source_root=ROOT)
    location = base / 'relocated installed package Ω 文 with spaces'
    location.mkdir()
    package.extract_package(archive, location)
    root = package.path_inside(location, receipt['inventory_root'])
    inventory_path = package.path_inside(location, receipt['inventory_path'])
    if package.sha(archive) != receipt['archive_sha256'] or package.sha(inventory_path) != receipt['inventory_sha256']:
        raise ValueError('Selected archive or inventory changed during extraction')
    package.verify_inventory(root, json.loads(inventory_path.read_text(encoding='utf-8')))
    executables, resources = package.package_directories(root, receipt['platform'], receipt['variant'])
    manifest = json.loads((resources / 'insonic-companions.json').read_text(encoding='utf-8'))
    if manifest.get('kind') != 'installed-companions' or manifest.get('schema_version') != receipt['schema_version'] or manifest.get('platform') != receipt['platform']:
        raise ValueError('Installed companion manifest differs from package election')
    tools = manifest['media']
    for name in ('ffmpeg', 'ffprobe', 'exiftool'):
        for item in [tools[name], *tools[name].get('support_files', [])]:
            path = package.path_inside(resources, item['path'])
            if not path.is_file() or package.sha(path) != item['sha256']:
                raise ValueError('Installed media companion differs from manifest')
            item['path'] = str(path)
    cueson = package.path_inside(resources, manifest['cueson']['executable'])
    if package.sha(cueson) != manifest['cueson']['executable_sha256']:
        raise ValueError('Installed Cueson differs from manifest')
    worker = resources / 'companions/processing/processing-worker.py'
    if package.sha(worker) != package.sha(ROOT / 'scripts/processing-worker.py'):
        raise ValueError('Installed processing worker differs from selected source')
    executable = executables / ('insonic.exe' if receipt['platform'].startswith('windows') else 'insonic')
    identity = {name: receipt[name] for name in ('revision', 'platform', 'variant', 'schema_version', 'product_version',
                'documentation_version', 'archive_sha256', 'inventory_sha256', 'source_dirty')}
    return executable, tools, cueson, worker, identity


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


def harness_identity():
    environment = engine.local_environment()
    revision = engine._child(['git', 'rev-parse', 'HEAD'], environment, timeout=timing.GIT_SECONDS).decode().strip()
    dirty = bool(engine._child(['git', 'status', '--porcelain'], environment, timeout=timing.GIT_SECONDS).strip())
    identity = hashlib.sha256()
    inputs = [ROOT / name for name in ('VERSION', 'go.mod', 'go.sum')]
    for prefix in ('internal', 'cmd', 'scripts', 'schemas/v' + (ROOT / 'VERSION').read_text().strip()):
        inputs.extend(path for path in (ROOT / prefix).rglob('*') if path.is_file() and path.suffix in ('.go', '.py', '.json'))
    inputs.append(ROOT / 'tests/fixtures/qualification-worker.py')
    for path in sorted(inputs):
        identity.update(path.relative_to(ROOT).as_posix().encode() + b'\0' + bytes.fromhex(engine.sha(path)))
    return revision, dirty, identity.hexdigest()


class Journey:
    def __init__(self, args, base):
        self.args, self.base = args, base
        self.started = time.monotonic()
        self.deadline = self.started + args.timeout
        self.workspace = base / 'source-workspace'
        packaged = bool(getattr(args, 'package', None))
        package_identity = None
        if packaged:
            self.cli, self.tools, self.cueson, self.worker, package_identity = select_package(args, base)
        else:
            self.cli = Path(args.cli).resolve(strict=True)
            self.tools = json.loads(Path(args.tools).read_text(encoding='utf-8'))
            self.cueson = Path(args.cueson).resolve(strict=True)
            self.worker = ROOT / 'scripts/processing-worker.py'
        if args.mode == 'controlled':
            self.worker = ROOT / 'tests/fixtures/qualification-worker.py'
        self.sequence = 0
        self.runtime = None
        self.runtime_tree = None
        self.environment = packaged_environment() if packaged else engine.local_environment()
        if not packaged and os.environ.get('INSONIC_NATIVE_LIBRARY'):
            self.environment['INSONIC_NATIVE_LIBRARY'] = os.environ['INSONIC_NATIVE_LIBRARY']
        home = base / 'private-user'
        home.mkdir()
        engine.secure_private_directory(str(home))
        for name in ('roaming', 'local', 'cache', 'config', 'tmp'):
            (home / name).mkdir()
        self.environment.update(HOME=str(home), USERPROFILE=str(home), APPDATA=str(home / 'roaming'),
                                LOCALAPPDATA=str(home / 'local'), XDG_CACHE_HOME=str(home / 'cache'),
                                XDG_CONFIG_HOME=str(home / 'config'), TEMP=str(home / 'tmp'), TMP=str(home / 'tmp'),
                                TMPDIR=str(home / 'tmp'), PYTHONIOENCODING='utf-8',
                                GH_PROMPT_DISABLED='1', GIT_TERMINAL_PROMPT='0')
        revision, dirty, identity = harness_identity()
        if packaged and getattr(args, 'reference_corpus', None) and (dirty or revision != package_identity['revision']):
            raise ValueError('Reference acceptance requires a clean exact package and harness revision')
        self.receipt = {'kind': 'connected-product-qualification', 'mode': args.mode, 'required_check': False,
                        'platform': platform.system(), 'device': 'cpu', 'cli_sha256': engine.sha(self.cli),
                        'worker_sha256': engine.sha(self.worker),
                        'product_version': (ROOT / 'VERSION').read_text().strip(),
                        'source_revision': revision, 'source_dirty': dirty, 'source_inputs_sha256': identity,
                        'operation_bound_seconds': timing.SUBPROCESS_SECONDS, 'overall_bound_seconds': args.timeout,
                        'hard_rss_limit_bytes': None, 'operations': [], 'records': []}
        if packaged:
            self.receipt.update(package=package_identity, source_revision=package_identity['revision'], source_dirty=False,
                                harness_revision=revision, harness_dirty=dirty, harness_inputs_sha256=identity,
                                worker_origin='installed-package' if args.mode == 'real' else 'controlled-fixture')
            self.receipt.pop('source_inputs_sha256')

    def assert_harness_unchanged(self):
        self.receipt['harness_unchanged'] = False
        revision, dirty, identity = harness_identity()
        if dirty or revision != self.receipt['harness_revision'] or identity != self.receipt['harness_inputs_sha256']:
            raise ValueError('Elected harness source changed during real acceptance')
        self.receipt['harness_unchanged'] = True

    def remaining(self, maximum=timing.SUBPROCESS_SECONDS):
        remaining = self.deadline - time.monotonic()
        if remaining <= 0:
            raise ValueError('Journey overall deadline exceeded')
        return min(maximum, remaining)

    def engine_child(self, command, *, maximum=timing.SUBPROCESS_SECONDS, max_output=16 << 20):
        return engine._child(command, self.environment, timeout=self.remaining(maximum), max_output=max_output)

    def retain_worker_probe(self, result):
        facts = result.get('worker_probe') if isinstance(result, dict) else None
        if not isinstance(facts, dict) or facts.get('role') not in ('recognition', 'diarization'):
            return
        from journey_cpu import probe_measurement
        probe = facts.get('probe')
        try:
            mode = probe.get('mode') if isinstance(probe, dict) else None
            if mode in ('inspect', 'snapshot'):
                self.receipt['worker_probe_failure'] = {
                    'role': facts['role'], 'probe': probe_measurement(probe, mode, success=False)}
        except ValueError:
            pass  # Invalid diagnostics cannot enter the acceptance receipt.

    def command(self, *args, workspace=True, maximum=timing.SUBPROCESS_SECONDS, pending_work_id=None):
        command = [self.cli, '--json']
        if workspace:
            command.extend(('--workspace', self.workspace))
        command.extend(args)
        print(json.dumps({'operation': str(args[0]) + '.' + str(args[1])}), flush=True)
        started = time.monotonic()
        try:
            raw = engine._child(command, self.environment, timeout=self.remaining(maximum), max_output=16 << 20, cli_response=True)
        except engine.CLIResponseError as error:
            result = error.response.get('result', {})
            failure = error.response.get('error', {})
            if (pending_work_id is not None and failure.get('code') == 'unavailable'
                    and failure.get('reason') == 'wait_timeout' and isinstance(result, dict)
                    and result.get('id') == pending_work_id and result.get('work_id', pending_work_id) == pending_work_id
                    and result.get('state') in ('queued', 'running')):
                return result
            self.receipt['failure_code'] = error.response['error']['code']
            reason = error.response['error'].get('reason')
            if isinstance(reason, str) and re.fullmatch(r'[a-z_]{1,64}', reason):
                self.receipt['failure_reason'] = reason
            try:
                runtime = getattr(self, 'runtime', None)
                returncode = runtime.poll() if runtime is not None else None
                pid = getattr(runtime, 'pid', None)
                overflow = getattr(self, 'runtime_overflow', None)
                self.receipt['failure_runtime'] = {
                    'started': runtime is not None, 'alive': runtime is not None and returncode is None,
                    'pid': pid if type(pid) is int and pid > 0 else None,
                    'returncode': returncode, 'output_overflow': bool(overflow and overflow.is_set())}
            except Exception:
                self.receipt['failure_runtime'] = {'observation': 'unavailable'}
                error.add_note('Owned runtime failure observation was unavailable')
            self.receipt['failed_invocation_seconds'] = time.monotonic() - started
            self.receipt['failed_operation'] = str(args[0]) + '.' + str(args[1])
            self.receipt['failed_work'] = {key: result.get(key) for key in ('kind', 'state', 'phase', 'error')} if isinstance(result, dict) else None
            if isinstance(result, dict) and isinstance(result.get('result'), dict):
                self.receipt['failed_work_result'] = {key: result['result'].get(key) for key in ('state', 'error')}
                self.retain_worker_probe(result['result'])
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

    def finish(self, queued, *, maximum=timing.SUBPROCESS_SECONDS):
        started = time.monotonic()
        work_id = queued.get('work_id', queued.get('id'))
        if not work_id:
            raise ValueError('Operation did not return durable work identity')
        deadline = min(self.deadline, started + maximum)
        while True:
            remaining = deadline - time.monotonic()
            if remaining <= 0: raise ValueError('Work exceeded its original operation deadline')
            query_seconds = min(600, remaining)  # The ordinary CLI API retains its ten-minute per-query maximum.
            current = self.command('work', 'wait', work_id, '--timeout-ms', str(max(1, int(query_seconds * 1000))),
                                   maximum=query_seconds + timing.CLEANUP_SECONDS, pending_work_id=work_id)
            if current.get('state') in ('queued', 'running'):
                continue
            if (current.get('state') != 'succeeded' or current.get('id') != work_id
                    or current.get('work_id', work_id) != work_id or time.monotonic() > deadline):
                self.retain_worker_probe(current.get('result'))
                raise ValueError('Work did not reach matching success within its original operation deadline')
            break
        self.receipt['operations'].append({'kind': current['kind'], 'state': current['state'], 'phase': current['phase'],
                                           'work_id': work_id, 'wait_wall_seconds': time.monotonic() - started})
        return current['result']

    def start_runtime(self):
        import queue
        from journey_cpu import startup_report
        started = time.monotonic()
        deadline = min(self.deadline, started + timing.READINESS_SECONDS)
        correlation = str(uuid.uuid4())
        workspace = ''
        entries = self.receipt.setdefault('runtime_startups', [])
        if len(entries) >= 8:
            raise ValueError('Owned runtime startup inventory exceeds its bound')
        def expired_before_launch():
            entries.append({'kind': 'runtime-startup', 'schema_version': self.receipt['product_version'],
                'startup_id': correlation, 'workspace_id': workspace, 'state': 'failed', 'stage': 'startup',
                'code': 'unavailable', 'reason': 'startup_timeout', 'reported': False,
                'elapsed_seconds': max(0, time.monotonic() - started), 'budget_seconds': max(0, deadline - started)})
            raise ValueError('Owned runtime startup original budget has expired')
        if deadline <= started:
            expired_before_launch()
        configuration = json.loads((self.workspace / '.insonic/workspace.json').read_text(encoding='utf-8'))
        workspace = configuration['workspace_id']
        if time.monotonic() >= deadline:
            expired_before_launch()
        self.runtime_tree = engine.ProcessTree(
            [str(self.cli), '--workspace', str(self.workspace), 'runtime', 'serve', '--startup-id', correlation],
            cwd=self.base, env=self.environment)
        self.runtime = self.runtime_tree.process
        self.runtime_overflow = threading.Event()
        self.runtime_readers = []
        ready = queue.Queue(maxsize=1)

        def drain(stream, startup=False):
            count, reported = 0, False
            try:
                if startup:
                    raw = stream.readline(4097)
                    count = len(raw)
                    ready.put_nowait((raw, time.monotonic()))
                    reported = True
                while chunk := stream.read(65536):
                    count += len(chunk)
                    if count > 16 << 20:
                        self.runtime_overflow.set()
                        self.runtime_tree.kill()
                        break
            except (OSError, ValueError):
                if startup and not reported:
                    ready.put_nowait((b'', time.monotonic()))
            finally:
                stream.close()
        for stream, startup in ((self.runtime.stdout, True), (self.runtime.stderr, False)):
            reader = threading.Thread(target=drain, args=(stream, startup), daemon=True)
            reader.start()
            self.runtime_readers.append(reader)
        report, reason = None, 'startup_report'
        try:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise queue.Empty()
            raw, observed = ready.get(timeout=remaining)
            if observed > deadline or time.monotonic() > deadline:
                raise queue.Empty()
            if not raw and self.runtime.poll() is not None:
                reason = 'startup_exit'
                raise ValueError()
            report = startup_report(raw, correlation, workspace, self.receipt['product_version'])
            if report['state'] != 'ready':
                reason = report['reason']
                raise ValueError()
            if self.runtime.poll() is not None or self.runtime_overflow.is_set():
                reason = 'startup_exit'
                raise ValueError()
            completed = time.monotonic()
            if completed > deadline:
                raise queue.Empty()
        except (queue.Empty, ValueError) as failure:
            if isinstance(failure, queue.Empty):
                reason = 'startup_timeout'
            native = report is not None and report['state'] == 'failed'
            failed = report if native else {
                'kind': 'runtime-startup', 'schema_version': self.receipt['product_version'],
                'startup_id': correlation, 'workspace_id': workspace, 'state': 'failed',
                'stage': 'startup', 'code': 'unavailable', 'reason': reason}
            entries.append({**failed, 'reported': native, 'elapsed_seconds': max(0, time.monotonic() - started),
                            'budget_seconds': max(0, deadline - started)})
            self.runtime_tree.kill()
            raise ValueError('Owned runtime did not report readiness within its original budget') from None
        entries.append({**report, 'reported': True, 'elapsed_seconds': completed - started,
                        'budget_seconds': deadline - started})

    def idle_runtime(self):
        if self.runtime is not None:
            self.runtime.wait(timeout=self.remaining(timing.SERVICE_SECONDS))
            if self.runtime.returncode or self.runtime_overflow.is_set():
                raise ValueError('Owned runtime failed before maintenance')
            if not self.runtime_tree.wait_for_descendants(self.remaining(timing.NATURAL_RELEASE_SECONDS)):
                raise ValueError('Owned runtime descendants did not release naturally')
            for reader in self.runtime_readers:
                reader.join(timeout=timing.CLEANUP_SECONDS)
            if any(reader.is_alive() for reader in self.runtime_readers):
                raise ValueError('Owned runtime output readers did not exit')
            if self.runtime_overflow.is_set():
                raise ValueError('Owned runtime output exceeded bounded limit')
            self.runtime_tree.close()
            self.runtime = None
            self.runtime_tree = None

    def query(self, operation, filters=None):
        result = self.input('query', 'run', value={'definition': {'mode': 'normalized', 'operation': operation,
                            'filters': filters or {}, 'pagination': {'limit': 100}}})
        if result.get('graph_error'):
            raise ValueError('Connected query fell back from its native graph')
        return result

    def withdraw_source(self):
        return withdraw_source(self.workspace, self.base)

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
                                   'recording': {key: record[key] for key in ('id', 'document', 'document_digest', 'source_map', 'source_digest', 'provenance', 'diagnostics')},
                                   'roster': self.command('recordings', 'roster', 'show', media_id)}
        model = self.command('models', 'speaker', 'show', version_id)
        listing = self.input('models', 'speaker', 'list', value={'speaker_id': speaker_id})
        return {'recordings': recordings, 'model': model, 'profile': listing.get('profile')}

    def run(self):
        actual_version = engine._child([self.cli, 'version'], self.environment, timeout=timing.GIT_SECONDS).decode().strip()
        if actual_version != self.receipt['product_version']:
            raise ValueError('Selected CLI version differs from current source')
        self.command('workspace', 'init', self.workspace, workspace=False)
        self.start_runtime()
        capabilities = self.command('graph', 'capabilities')
        if capabilities.get('state') != 'available' or capabilities.get('adapter_id') != 'ladybugdb':
            raise ValueError('Journey requires the actual native LadybugDB graph')
        self.receipt['graph'] = capabilities
        version = self.receipt['product_version']
        tools = self.tools
        tools['schema_version'] = version
        self.command('media', 'tools', self.payload(tools))
        cueson, worker = self.cueson, self.worker
        pythons = {'transcribe': Path(self.args.recognition_python).resolve(strict=True),
                   'diarize': Path(self.args.diarization_python).resolve(strict=True)}
        self.receipt['python_executable_sha256'] = {operation: engine.sha(path) for operation, path in pythons.items()}
        self.receipt['companion_sha256'] = {**{name: tools[name]['sha256'] for name in ('ffmpeg', 'ffprobe', 'exiftool')},
                                           'cueson': engine.sha(cueson)}
        processing = {'threads': 2, 'timeout_ms': timing.REAL_PROCESSING_SECONDS * 1000, 'max_duration_us': 60000000, 'max_input_bytes': 67108864,
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
        self.receipt['model_inputs'] = {operation: {'id': pin['id'], 'revision': pin['revision'],
                                       'files': {role: {'size_bytes': size, 'sha256': digest}
                                                 for role, (size, digest) in pin['files'].items()}}
                                       for operation, pin in pins.items()}

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
            server.shutdown(); server.server_close(); thread.join(timeout=timing.CLEANUP_SECONDS)

        ffmpeg = tools['ffmpeg']
        if engine.sha(ffmpeg['path']) != ffmpeg['sha256']:
            raise ValueError('Selected media decoder digest differs')
        crop = self.base / 'cross-clip.wav'
        source = ROOT / 'tests/fixtures/media/speech.flac'
        engine._child([ffmpeg['path'], '-nostdin', '-hide_banner', '-loglevel', 'error', '-ss', '1', '-i', source,
                       '-t', '8', '-ac', '1', '-ar', '16000', '-c:a', 'pcm_s16le', crop], self.environment, timeout=timing.SUBPROCESS_SECONDS)
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


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--mode', choices=('real', 'controlled'), required=True)
    selection = parser.add_mutually_exclusive_group(required=True)
    selection.add_argument('--cli')
    selection.add_argument('--package')
    parser.add_argument('--tools')
    parser.add_argument('--cueson')
    parser.add_argument('--package-receipt')
    parser.add_argument('--revision')
    parser.add_argument('--recognition-python')
    parser.add_argument('--diarization-python')
    parser.add_argument('--recognition-model')
    parser.add_argument('--diarization-model')
    parser.add_argument('--reference-corpus')
    parser.add_argument('--managed-offline-assets')
    parser.add_argument('--timeout', type=int)
    parser.add_argument('--output', required=True)
    args = parser.parse_args(argv)
    require_mode(args.mode)
    validate_selection(args)
    if args.reference_corpus:
        spec = importlib.util.spec_from_file_location('reference_cpu_journey', ROOT / 'scripts/journey_cpu.py')
        cpu = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cpu)
        cpu.validate_election(args)
    elif not args.recognition_python or not args.diarization_python or args.managed_offline_assets:
        raise ValueError('Legacy journey requires both explicit interpreters and no managed bundle override')
    if args.timeout is None:
        args.timeout = timing.JOURNEY_SECONDS
    if not 60 <= args.timeout <= 3600: raise ValueError('Invalid journey bound')
    if args.mode == 'real' and (not args.recognition_model or not args.diarization_model): raise ValueError('Real mode requires exact installed models')
    temporary = tempfile.TemporaryDirectory(prefix='insonic-connected-journey-')
    temporary_path = Path(temporary.name)
    journey, receipt, primary = None, None, None
    runtime_released, temporary_removed = True, False
    started = time.monotonic()
    try:
        journey = Journey(args, temporary_path)
        receipt = cpu.run(journey) if args.reference_corpus else journey.run()
    except BaseException as error:
        primary = error
        receipt = dict(journey.receipt) if journey else {'kind': 'connected-product-qualification', 'mode': args.mode}
        if journey is None and args.reference_corpus:
            receipt.update(kind='insonic-real-cpu-acceptance-v2', first_failure_stage='package',
                           stages=[{'name': 'package', 'state': 'failed'}])
    finally:
        if journey is not None and journey.runtime is not None:
            runtime_released = False
            tree = getattr(journey, 'runtime_tree', None)
            try:
                if tree is not None and primary is not None:
                    tree.kill()
                    journey.runtime.wait(timeout=timing.CLEANUP_SECONDS)
                else:
                    try:
                        journey.runtime.wait(timeout=timing.SERVICE_SECONDS)
                    except subprocess.TimeoutExpired:
                        if tree is not None:
                            tree.kill()
                        else:
                            journey.runtime.terminate()
                        journey.runtime.wait(timeout=timing.CLEANUP_SECONDS)
                        raise ValueError('Owned runtime required forced cleanup')
                if tree is not None and not tree.wait_for_descendants(timing.NATURAL_RELEASE_SECONDS):
                    raise ValueError('Owned runtime cleanup is incomplete')
                runtime_released = tree is not None
            except BaseException as error:
                if primary is None:
                    primary = error
                    if args.reference_corpus:
                        receipt.setdefault('first_failure_stage', 'cleanup')
                else:
                    primary.add_note('Owned runtime cleanup failed')
                if receipt is not None:
                    receipt['cleanup_error'] = type(error).__name__
            finally:
                if tree is not None:
                    for reader in journey.runtime_readers:
                        reader.join(timeout=timing.CLEANUP_SECONDS)
                    tree.close()
        try:
            temporary.cleanup()
            temporary_removed = not temporary_path.exists()
        except BaseException as error:
            if primary is None:
                primary = error
                if args.reference_corpus:
                    receipt.setdefault('first_failure_stage', 'cleanup')
            else:
                primary.add_note('Owned temporary-directory cleanup failed')
            if receipt is not None:
                receipt['temporary_cleanup_error'] = type(error).__name__
    if receipt is None:
        receipt = {'kind': 'connected-product-qualification', 'mode': args.mode}
    if args.reference_corpus:
        receipt['cleanup'] = {'runtime_released': runtime_released, 'temporary_removed': temporary_removed}
    receipt['wall_seconds'] = time.monotonic() - started
    if primary is None and args.reference_corpus:
        try:
            receipt['combined_seconds'] = receipt['wall_seconds']
            receipt['qualification_seconds'] = receipt['combined_seconds'] - receipt['preparation_seconds']
            receipt['stages'].append({'name': 'cleanup', 'state': 'succeeded'})
            cpu.validate_acceptance(receipt, cpu.references().load_frozen_corpus(args.reference_corpus))
        except BaseException as error:
            primary = error
            receipt.setdefault('first_failure_stage', 'cleanup')
    if primary is not None:
        receipt.update(state='failed', failure_type=type(primary).__name__)
        try:
            write_json(Path(args.output).resolve(), receipt)
        except BaseException as error:
            primary.add_note('Failure receipt could not be written: ' + type(error).__name__)
        raise primary
    receipt['state'] = 'passed'
    write_json(Path(args.output).resolve(), receipt)
    print(json.dumps({'state': receipt['state'], 'mode': args.mode,
                      'wall_seconds': receipt['wall_seconds'], 'receipt': str(Path(args.output).resolve())}))


if __name__ == '__main__':
    main()
