# SPDX-License-Identifier: Apache-2.0
"""Real terminal wait semantics over owned, deterministic durable work."""
from concurrent.futures import ThreadPoolExecutor
from contextlib import ExitStack, contextmanager
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import queue
import re
import shutil
import sys
import tempfile
import threading
import time
import uuid

from cli_corpus import finish_server
from process_tree import ProcessTree
import qualification_timing as bounds
from terminal_transport import Terminal
from terminal_failure_evidence import FailureEvidence, diagnostic_environment


REQUIRED = {
    'human': (0, 'succeeded', None), 'json': (0, 'succeeded', None),
    'quiet': (0, 'succeeded', None), 'redirected': (0, 'succeeded', None),
    'interrupt': (1, 'succeeded', 'wait_interrupted'),
    'timeout': (4, 'succeeded', 'wait_timeout'),
    'cancelled': (1, 'cancelled', 'cancelled'),
    'failed': (1, 'failed', 'checksum_mismatch'),
}
CASE_FIELDS = {'work_id', 'request_id', 'exit_code', 'durable_state', 'reason', 'natural_exit',
               'cleanup', 'elapsed_seconds', 'bound_seconds', 'progress', 'count'}


def canonical_id(value):
    try:
        return isinstance(value, str) and str(uuid.UUID(value)) == value
    except ValueError:
        return False


def validate_receipt(receipt, digest, platform):
    fields = {'format', 'platform', 'transport', 'executable_sha256', 'cases', 'cleanup',
              'runtime_natural_exit', 'continued_work', 'server_released', 'fixture_removed'}
    if (not isinstance(receipt, dict) or set(receipt) != fields
            or platform not in ('windows', 'linux')
            or receipt['format'] != 'insonic-terminal-qualification-v1'
            or not isinstance(digest, str) or not re.fullmatch('[0-9a-f]{64}', digest)
            or receipt['executable_sha256'] != digest or receipt['platform'] != platform
            or receipt['transport'] != {'windows': 'conpty', 'linux': 'linux-pty'}.get(platform)
            or any(receipt[key] is not True for key in
                   ['cleanup', 'runtime_natural_exit', 'continued_work', 'server_released', 'fixture_removed'])
            or not isinstance(receipt['cases'], dict) or set(receipt['cases']) != set(REQUIRED)):
        raise ValueError('Incomplete terminal qualification receipt')
    for name, (code, state, reason) in REQUIRED.items():
        case = receipt['cases'][name]
        if (not isinstance(case, dict) or set(case) != CASE_FIELDS
                or not canonical_id(case['work_id']) or not canonical_id(case['request_id'])
                or type(case['exit_code']) is not int or case['exit_code'] != code
                or case['durable_state'] != state or case['reason'] != reason
                or case['natural_exit'] is not True or case['cleanup'] is not True
                or any(type(case[key]) not in (int, float)
                       for key in ['elapsed_seconds', 'bound_seconds'])
                or not 0 <= case['elapsed_seconds'] <= case['bound_seconds'] <= bounds.SUBPROCESS_SECONDS
                or case['bound_seconds'] <= 0
                or any(type(case[key]) is not bool for key in ['progress', 'count'])):
            raise ValueError('Invalid terminal case ' + name)
        if name in ('human', 'interrupt', 'cancelled', 'failed') and not (case['progress'] and case['count']):
            raise ValueError('Missing actual terminal progress ' + name)
        if name in ('json', 'quiet', 'redirected') and (case['progress'] or case['count']):
            raise ValueError('Suppressed mode retained progress ' + name)
    cases = receipt['cases']
    if (any(cases[name]['work_id'] != cases['human']['work_id'] for name in ('json', 'quiet', 'redirected'))
            or len({cases[name]['work_id'] for name in ('human', 'interrupt', 'timeout', 'cancelled', 'failed')}) != 5
            or len({case['request_id'] for case in cases.values()}) != len(REQUIRED)):
        raise ValueError('Terminal scenario correlation differs')


@contextmanager
def owned_directory(workspace):
    directory = workspace / 'terminal fixtures café'
    directory.mkdir()  # Existing owner data is refused before cleanup is armed.
    try:
        yield directory
    finally:
        primary = sys.exception()
        try:
            if (directory.is_symlink() or (hasattr(directory, 'is_junction') and directory.is_junction())
                    or directory.resolve().parent != workspace.resolve()):
                raise ValueError('Owned terminal fixture removal boundary changed')
            shutil.rmtree(directory)
        except BaseException as error:
            if primary is None:
                raise
            primary.add_note('Owned terminal directory cleanup failed: ' + type(error).__name__)


@contextmanager
def owned_runtime(argv, cwd, env, gates, factory=ProcessTree, failure_evidence=None):
    with factory(argv, cwd=cwd, env=env) as owner:
        try:
            if failure_evidence is None:
                yield owner
            else:
                with failure_evidence.watch(owner):
                    yield owner
        finally:
            primary = sys.exception()
            for gate in gates.values():
                gate['release'].set()
            if primary is not None:
                try:
                    end = time.monotonic() + bounds.CLEANUP_SECONDS
                    owner.kill()
                    owner.process.communicate(timeout=max(.001, end - time.monotonic()))
                    if not owner.wait_for_descendants(max(0, end - time.monotonic())):
                        raise RuntimeError('Owned runtime descendants did not release')
                except BaseException as error:
                    primary.add_note('Owned runtime failure cleanup failed: ' + type(error).__name__)


class TrackedServer(ThreadingHTTPServer):
    def __init__(self, *args):
        self.request_threads = []
        self.released = False
        super().__init__(*args)

    def process_request(self, request, address):
        worker = threading.Thread(target=self.process_request_thread, args=(request, address), daemon=True)
        self.request_threads.append(worker)
        worker.start()


@contextmanager
def owned_server(handler, gates):
    server = TrackedServer(('127.0.0.1', 0), handler)
    thread = threading.Thread(target=server.serve_forever, name='owned-terminal-fixture-server', daemon=True)
    try:
        thread.start()
        yield server
    finally:
        primary, failures = sys.exception(), []
        end = time.monotonic() + bounds.CLEANUP_SECONDS
        for gate in gates.values():
            gate['release'].set()
        try:
            if thread.ident is not None:
                finish_server(server, thread)
            else:
                server.server_close()
        except BaseException as error:
            failures.append(type(error).__name__)
        for worker in server.request_threads:
            worker.join(max(0, end - time.monotonic()))
            if worker.is_alive():
                failures.append('request handler did not release')
        server.released = server.fileno() == -1 and not thread.is_alive() and not failures
        if failures:
            error = RuntimeError('Owned terminal server cleanup failed: ' + ', '.join(failures))
            if primary is None:
                raise error
            primary.add_note(str(error))


def qualify(invoke, workspace, version, *, executable, environment, observe=None, process_factory=None):
    started = time.monotonic()
    campaign_bound = bounds.phase_bound(bounds.SUBPROCESS_SECONDS)
    deadline = started + campaign_bound

    def remaining(seconds=bounds.SUBPROCESS_SECONDS):
        value = min(seconds, deadline - time.monotonic(), bounds.phase_bound(seconds))
        if value <= 0:
            raise TimeoutError('Terminal campaign enclosing deadline exceeded')
        return value

    executable = Path(executable).resolve(strict=True)
    digest = hashlib.sha256(executable.read_bytes()).hexdigest()
    terminal_factory = Terminal if process_factory is None else getattr(process_factory, 'terminal', None)
    if terminal_factory is None:
        raise ValueError('Elected CLI process factory does not support owned terminals')
    with ExitStack() as resources:
        directory = resources.enter_context(owned_directory(Path(workspace)))
        # The library keeps its realistic deep/Unicode path. A separate short
        # private host cache keeps Linux Unix-socket names within the runtime's
        # existing 100-byte endpoint bound, as on an ordinary user profile.
        private = Path(resources.enter_context(tempfile.TemporaryDirectory(
            prefix='it-', dir='/tmp' if sys.platform == 'linux' else None)))
        selected = diagnostic_environment(environment)
        for name in ['HOME', 'USERPROFILE', 'APPDATA', 'LOCALAPPDATA', 'XDG_RUNTIME_DIR', 'XDG_CONFIG_HOME', 'XDG_DATA_HOME', 'XDG_CACHE_HOME']:
            selected[name] = str(private)
        library = directory / 'owned library'
        evidence = FailureEvidence(digest, selected)

        def call(args, **options):
            options.setdefault('timeout_seconds', remaining())
            return invoke(args, workspace_path=library, execution_environment=selected, **options)

        gates, cases = {}, {}
        owner = None
        runtime_natural = continued = False
        request_file = directory / 'manifest.json'

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                token, role = self.path.lstrip('/').split('/')
                gate = gates[token]
                if role == 'config':
                    gate['started'].set()
                    if not gate['release'].wait(max(0, deadline - time.monotonic())):
                        return
                data = gate[role]
                if role == 'config' and gate['bad']:
                    data = b'x' * len(data)
                try:
                    self.send_response(200)
                    self.send_header('Content-Length', str(len(data)))
                    self.end_headers()
                    self.wfile.write(data)
                except (BrokenPipeError, ConnectionResetError, ConnectionAbortedError):
                    pass  # Explicit durable cancellation can close its HTTP request.

            def log_message(self, *_args):
                pass

        server = resources.enter_context(owned_server(Handler, gates))

        def work(work_id, state):
            value = call(['work', 'show', work_id])['result']
            if value['id'] != work_id or value['state'] != state:
                raise ValueError('Independently observed durable work differs')
            return value

        def held(*, bad=False):
            token = str(uuid.uuid4())
            gate = {'weights': (token + '-weights').encode(), 'config': (token + '-config').encode(),
                    'started': threading.Event(), 'release': threading.Event(), 'bad': bad}
            gates[token] = gate
            manifest = {'kind': 'base-model-manifest', 'schema_version': version, 'name': 'terminal-' + token,
                        'model_version': '1', 'upstream_revision': 'owned-fixture', 'license': 'synthetic fixture',
                        'capabilities': ['transcription'], 'files': [
                            {'role': role, 'size': len(gate[role]), 'sha256': hashlib.sha256(gate[role]).hexdigest(),
                             'url': f'http://127.0.0.1:{server.server_port}/{token}/{role}', 'local_http': True}
                            for role in ['weights', 'config']]}
            request_file.write_text(json.dumps(manifest) + '\n', encoding='utf-8', newline='\n')
            result = call(['models', 'acquire', request_file])['result']
            identity, parent = result['acquisition_ids'][0], result['work_id']
            if not gate['started'].wait(remaining(bounds.READINESS_SECONDS)):
                raise TimeoutError('Owned acquisition did not reach the held second file')
            current = work(identity, 'running')
            if current.get('result', {}).get('files_verified') != 1:
                raise ValueError('Held acquisition has no independently observed verified count')
            return identity, parent, gate

        def terminal(stack, identity, flags=(), *, timeout=None):
            request_id = str(uuid.uuid4())
            args = ['--workspace', str(library), '--request-id', request_id, '--log-level', 'debug',
                    '--log-file', str(directory / (request_id + '.jsonl')), *flags, 'work', 'wait', identity,
                    '--timeout-ms', str(timeout if timeout is not None else bounds.DURABLE_WORK_SECONDS * 1000)]
            child = stack.enter_context(terminal_factory([str(executable), *args], cwd=directory, env=selected, deadline=deadline))
            evidence.identities['work_id'] = identity
            evidence.track(child, request_id, flags[0][2:] if flags else 'human', directory / (request_id + '.jsonl'), work_id=identity)
            # Register after each child so the newest observer runs before any
            # terminal teardown and can see every live correlated client.
            stack.enter_context(evidence.watch(owner))
            return child, args, request_id

        def running_barrier(request_id, label, child=None):
            path = directory / (request_id + '.jsonl')
            while True:
                raw = path.read_bytes() if path.exists() else b''
                if len(raw) > 1 << 20:
                    raise ValueError('Terminal diagnostic observation bound exceeded')
                for line in raw.splitlines(keepends=True):
                    if not line.endswith(b'\n'):
                        continue  # Concurrent append can leave a partial final line.
                    event = json.loads(line)
                    if (event.get('request_id') == request_id and event.get('operation') == 'work.wait'
                            and event.get('state') == 'running'):
                        return
                if child is not None and child.owner.process.poll() is not None:
                    # Retain failure evidence while handles still exist, then
                    # drain final bytes before classifying the running miss.
                    primary = ValueError('Terminal ' + label + ' exited before its running barrier')
                    evidence.preserve(owner, primary)
                    try:
                        result = child.finish()
                    except BaseException as error:
                        primary.add_note('Terminal final drain or cleanup failed')
                        raise primary from error
                    primary.args = (str(primary) + ': ' + result['text'][:2048],)
                    raise primary
                try:
                    time.sleep(min(.02, remaining()))
                except TimeoutError as error:
                    raise TimeoutError('Terminal ' + label + ' running barrier was not observed') from error

        def progress(child, identity):
            child.wait_for(f'Work {identity}: running (verified-files), 1 verified files')

        def record(name, identity, request_id, result, *, args=None):
            code, state, reason = REQUIRED[name]
            prefix = f'Work {identity}:'
            shown = result['text']
            if result['exit_code'] != code or not result['natural_exit'] or not result['cleanup']:
                raise ValueError('Terminal outcome differs: ' + name
                                 + f" (exit {result['exit_code']}, expected {code}, output {shown[:2048]!r})")
            actual_progress = prefix in shown
            count = ', 1 verified files' in shown
            if name in ('human', 'interrupt', 'cancelled', 'failed') and not (actual_progress and count):
                raise ValueError('Terminal progress/count missing: ' + name)
            if name == 'human' and ('State: Succeeded' not in shown or prefix + ' succeeded' not in shown):
                raise ValueError('Terminal success lost changed terminal state')
            if name in ('json', 'quiet', 'redirected') and actual_progress:
                raise ValueError('Suppressed mode emitted progress')
            if name == 'json':
                response = json.loads(shown)
                if response.get('request_id') != request_id or response.get('result', {}).get('id') != identity or response['result']['state'] != 'succeeded':
                    raise ValueError('Terminal JSON result differs')
            if name == 'quiet' and shown.strip():
                raise ValueError('Quiet terminal emitted informational output')
            if name == 'redirected' and 'State: Succeeded' not in shown:
                raise ValueError('Redirected result lost success')
            if name == 'interrupt' and ('Waiting stopped; durable work was not cancelled.' not in shown or 'State: Running' not in shown):
                raise ValueError('Terminal interruption lost last observation or meaning')
            if name == 'timeout' and 'The selected wait limit elapsed.' not in shown:
                raise ValueError('Terminal timeout lost its classified meaning')
            if name == 'cancelled' and 'State: Cancelled' not in shown:
                raise ValueError('Explicit cancellation lost durable state')
            if name == 'failed' and 'checksum_mismatch' not in shown:
                raise ValueError('Failed acquisition lost its durable cause')
            cases[name] = {'work_id': identity, 'request_id': request_id, 'exit_code': code, 'durable_state': state,
                           'reason': reason, 'natural_exit': True, 'cleanup': True,
                           'elapsed_seconds': result['elapsed_seconds'], 'bound_seconds': result['bound_seconds'],
                           'progress': actual_progress, 'count': count}
            if args is not None and observe is not None:
                observe(args, code, scenario='terminal')

        def complete(identity, parent, gate):
            gate['release'].set()
            call(['work', 'wait', identity])
            call(['work', 'wait', parent])
            work(identity, 'succeeded')

        call(['workspace', 'init', library], scoped=False)
        startup_id = str(uuid.uuid4())
        evidence.identities['startup_id'] = startup_id
        with owned_runtime([str(executable), '--workspace', str(library), '--startup-id', startup_id,
                          '--json', 'runtime', 'serve'], cwd=directory, env=selected, gates=gates,
                          factory=process_factory or ProcessTree, failure_evidence=evidence) as owner:
            mailbox = queue.Queue(maxsize=1)
            ready_reader = threading.Thread(target=lambda: mailbox.put(owner.process.stdout.readline()), daemon=True)
            ready_reader.start()
            try:
                report = json.loads(mailbox.get(timeout=remaining(bounds.READINESS_SECONDS)))
            except BaseException:
                owner.kill()
                raise
            finally:
                ready_reader.join(bounds.CLEANUP_SECONDS)
                if ready_reader.is_alive():
                    raise RuntimeError('Runtime startup reader did not release')
            session = call(['workspace', 'show'])
            evidence.identities.update(workspace_id=session.get('workspace_id'), runtime_session_id=session.get('runtime_session_id'))
            if (report.get('startup_id') != startup_id or report.get('state') != 'ready'
                    or report.get('workspace_id') != session.get('workspace_id') or not session.get('runtime_session_id')):
                raise ValueError('Owned runtime readiness identity differs')

            identity, parent, gate = held()
            with ExitStack() as stack, ThreadPoolExecutor(max_workers=1) as pool:
                human, human_args, human_request = terminal(stack, identity)
                progress(human, identity)
                machine, machine_args, machine_request = terminal(stack, identity, ['--json'])
                quiet, quiet_args, quiet_request = terminal(stack, identity, ['--quiet'])
                redirected_timing = {}
                redirected_request = str(uuid.uuid4())
                redirected = pool.submit(call, ['--log-level', 'debug', '--log-file',
                    str(directory / (redirected_request + '.jsonl')), 'work', 'wait', identity], request_id=redirected_request,
                                         machine=False, timing=redirected_timing)
                evidence.track(None, redirected_request, 'redirected', directory / (redirected_request + '.jsonl'), work_id=identity)
                stack.enter_context(evidence.watch(owner))
                for request_id, label, child in ((machine_request, 'json', machine),
                                                 (quiet_request, 'quiet', quiet),
                                                 (redirected_request, 'redirected', None)):
                    running_barrier(request_id, label, child)
                gate['release'].set()
                record('human', identity, human_request, human.finish(), args=human_args)
                record('json', identity, machine_request, machine.finish(), args=machine_args)
                record('quiet', identity, quiet_request, quiet.finish(), args=quiet_args)
                text = redirected.result(timeout=remaining())
                if len(text.encode('utf-8')) > 65536:
                    raise ValueError('Redirected capture bound exceeded')
                record('redirected', identity, redirected_request, {'text': text, 'exit_code': 0,
                    'natural_exit': True, 'cleanup': True, 'elapsed_seconds': redirected_timing['total_seconds'],
                    'bound_seconds': bounds.SUBPROCESS_SECONDS})
            complete(identity, parent, gate)

            identity, parent, gate = held()
            with ExitStack() as stack:
                child, args, request_id = terminal(stack, identity)
                progress(child, identity)
                child.interrupt()
                record('interrupt', identity, request_id, child.finish(), args=args)
            observed = call(['work', 'show', identity])
            if observed.get('runtime_session_id') != session['runtime_session_id']:
                raise ValueError('Interrupt changed runtime ownership')
            work(identity, 'running')
            complete(identity, parent, gate)
            continued = True

            identity, parent, gate = held()
            with ExitStack() as stack:
                child, args, request_id = terminal(stack, identity, timeout=500)
                record('timeout', identity, request_id, child.finish(), args=args)
            work(identity, 'running')
            complete(identity, parent, gate)

            identity, parent, gate = held()
            with ExitStack() as stack:
                child, args, request_id = terminal(stack, identity)
                progress(child, identity)
                call(['work', 'cancel', identity])
                gate['release'].set()
                record('cancelled', identity, request_id, child.finish(), args=args)
            work(identity, 'cancelled')
            call(['work', 'wait', parent], expected=1)

            identity, parent, gate = held(bad=True)
            with ExitStack() as stack:
                child, args, request_id = terminal(stack, identity)
                progress(child, identity)
                gate['release'].set()
                record('failed', identity, request_id, child.finish(), args=args)
            if work(identity, 'failed').get('error') != 'checksum_mismatch':
                raise ValueError('Checksum failure differs from observed work')
            call(['work', 'wait', parent], expected=1)
            output, error = owner.process.communicate(timeout=remaining(bounds.FIXTURE_IDLE_SECONDS))
            if owner.process.returncode != 0 or error or not owner.wait_for_descendants(remaining(bounds.CLEANUP_SECONDS)):
                raise ValueError('Owned runtime did not exit naturally')
            runtime_natural = True
        receipt = {'format': 'insonic-terminal-qualification-v1', 'platform': 'windows' if os.name == 'nt' else sys.platform,
                   'transport': 'conpty' if os.name == 'nt' else 'linux-pty', 'executable_sha256': digest,
                   'cases': cases, 'cleanup': True, 'runtime_natural_exit': runtime_natural,
                   'continued_work': continued, 'server_released': False, 'fixture_removed': False}
    receipt['server_released'] = server.released
    receipt['fixture_removed'] = not directory.exists() and not private.exists()
    remaining()
    validate_receipt(receipt, digest, receipt['platform'])
    return receipt
