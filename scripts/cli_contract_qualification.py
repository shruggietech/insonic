#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Exercise installed CLI contracts with disposable, nonsecret fixtures."""
import copy
from contextlib import contextmanager
from concurrent.futures import ThreadPoolExecutor
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import math
from cli_corpus import Coverage, finish_server, failure_class
import os
from pathlib import Path
import threading
import time
import uuid

from process_tree import ProcessTree
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as bounds


PARSER_REJECTIONS = [
    ['doctor', '--unknown'], ['unknown'], ['doctor', '--workspace='],
    ['doctor', '--request-id', 'invalid'], ['doctor', '--json'],
    ['work', 'list', '--limit=0'],
    ['media', 'import', '--copy', '--reference', 'fixture.wav'],
]


def capture_invocation(factory, argv, *, cwd, env, scoped, input, timeout_seconds=None):
    started = time.monotonic()
    with factory(argv, cwd=cwd, env=env, allow_detached=scoped, input_pipe=input is not None) as tree:
        launched = time.monotonic()
        allowance = bounds.phase_bound(bounds.SUBPROCESS_SECONDS)
        if timeout_seconds is not None:
            allowance = min(allowance, timeout_seconds - (launched - started))
        if allowance <= 0:
            raise TimeoutError('CLI invocation enclosing deadline exceeded during launch')
        stdout, stderr = tree.process.communicate(input=input, timeout=allowance)
        finished = time.monotonic()
        code = tree.process.returncode
    timing = {"owned_launch_seconds": launched - started, "process_wait_seconds": finished - launched,
              "total_seconds": time.monotonic() - started}
    return stdout, stderr, code, timing


def verify_cold_bootstrap(response, timing):
    values = [timing.get(key) for key in ("owned_launch_seconds", "process_wait_seconds", "total_seconds")]
    if (response.get('error', {}).get('code') != 'unavailable'
            or response.get('error', {}).get('reason') != 'credential_bootstrap'
            or any(type(value) not in (int, float) or not math.isfinite(value) or value < 0 for value in values)
            or values[1] > bounds.READINESS_SECONDS or values[2] < values[0] + values[1]):
        raise ValueError('Cold bootstrap failure was not promptly classified')


def qualify_read_only(invoke, paths, help_workspace, missing_workspace, *, observations=None):
    """Keep all help/parser assertions in one bounded, joined worker pool."""
    before = sorted(str(p.relative_to(help_workspace)) for p in help_workspace.rglob('*'))
    groups = sorted({tuple(path[:n]) for path in paths for n in range(1, len(path))})
    help_pages = {}
    checks = [('context', path) for path in paths]
    checks += [('group', list(group)) for group in groups]
    checks += [('rejection', args) for args in PARSER_REJECTIONS]

    def remember(args, response, code=0, help_only=False):
        if observations is not None:
            if len(observations) >= 6000: raise ValueError('Read-only coverage observation bound exceeded')
            # Only static command IDs/options and dummy placeholders survive.
            observations.append((['--json', '--request-id', 'VALUE', *args], code,
                failure_class(response['error']) if response.get('error') else None, help_only))

    def check(item):
        kind, args = item
        if kind == 'context':
            inside = invoke(['--workspace', help_workspace, *args, '--help'], scoped=False)['result']['help']
            outside = invoke(['--workspace', missing_workspace, *args, '-h'], scoped=False)['result']['help']
            prefixed = invoke(['help', *args], scoped=False)['result']['help']
            remember(['--workspace', 'VALUE', *args, '--help'], {})
            remember(['--workspace', 'VALUE', *args, '-h'], {})
            remember(['help', *args], {})
            if inside != outside or inside != prefixed or 'Usage:' not in inside:
                raise ValueError('Contextual help differs: ' + ' '.join(args))
            help_pages[' '.join(args)] = inside
        elif kind == 'group':
            response = invoke(args, scoped=False)
            remember(args, response, help_only=True)
            if 'Commands:' not in response['result']['help']:
                raise ValueError('No-argument group help differs')
        else:
            remember(args, invoke(args, expected=2, scoped=False), 2)

    with ThreadPoolExecutor(max_workers=4) as workers:
        # map observes failures in authored check order; exiting the context
        # joins every submitted check, including its owned ProcessTree cleanup.
        list(workers.map(check, checks))
    after = sorted(str(p.relative_to(help_workspace)) for p in help_workspace.rglob('*'))
    if before != after:
        raise ValueError('Help created workspace files')
    return help_pages



def isolated_invoker(invoke, workspace, scenario=None):
    """Keep one helper's defaults local while preserving explicit overrides."""
    def run(args, **kwargs):
        kwargs.setdefault('workspace_path', workspace)
        if scenario is not None:kwargs.setdefault('scenario', scenario)
        return invoke(args, **kwargs)
    return run


@contextmanager
def parallel_regressions(invoke, workspace, version, documentation, help_pages, executable, environment, *, process_factory=None, observe=None):
    """Join every owned regression even when another branch raises."""
    from cli_operator_qualification import qualify as qualify_operator
    from cli_input_qualification import qualify as qualify_inputs
    from cli_text_duration_qualification import qualify as qualify_text_duration
    from cli_readiness_qualification import qualify as qualify_readiness
    from cli_terminal_qualification import qualify as qualify_terminal
    from cli_recovery_qualification import qualify as qualify_recovery
    from cli_storage_qualification import qualify as qualify_storage
    from cli_backup_qualification import qualify as qualify_backup
    jobs = [('operator', qualify_operator, {'media': documentation}),
            ('input', qualify_inputs, {'media': documentation}),
            ('text_duration', qualify_text_duration, {}),
            ('readiness', qualify_readiness, {'executable': executable}),
            ('terminal', qualify_terminal, {'executable': executable, 'environment': environment, 'observe': observe}),
            ('recovery', qualify_recovery, {'executable': executable, 'environment': environment}),
            ('storage', qualify_storage, {'executable': executable, 'environment': environment}),
            ('backup', qualify_backup, {'executable': executable, 'environment': environment})]
    if process_factory is not None:
        for name, _, keywords in jobs:
            if name in ('terminal', 'recovery', 'storage', 'backup'):
                keywords['process_factory'] = process_factory
    if documentation:
        from cli_help_examples import qualify as qualify_examples
        jobs.append(('examples', qualify_examples, {}))
    prepared = []
    for name, function, keywords in jobs:
        root = workspace / ('isolated ' + name + ' qualification')
        invoke(['workspace', 'init', root], scoped=False)
        scoped = isolated_invoker(invoke, root, name.replace('_', '-'))
        args = (scoped, root, version)
        if name == 'examples':
            args = (scoped, None, root, version, help_pages)
            keywords = {'executable': executable, 'environment': environment}
            if process_factory is not None:keywords['process_factory'] = process_factory
        prepared.append((name, function, args, keywords))
    # Independent libraries prevent credential, settings, model and
    # revision interference. Joining all workers leaves the existing package
    # idle allowance and image-release check after every library's activity.
    with ThreadPoolExecutor(max_workers=4) as workers:
        futures = {name: workers.submit(function, *args, **keywords)
                   for name, function, args, keywords in prepared}
        yield futures

def qualify(executable, workspace, *, env=None, cwd=None, documentation=False, progress=None, process_factory=None, artifact_identity=None):
    executable, workspace = Path(executable), Path(workspace)
    environment = os.environ.copy()
    environment.update(env or {})
    version = (Path(__file__).resolve().parents[1] / 'VERSION').read_text().strip()

    coverage = None
    current_scenario = 'core'

    def invoke(args, *, expected=0, input=None, scoped=True, machine=True, workspace_path=None, execution_environment=None, scenario=None, timing=None, request_id=None, timeout_seconds=None):
        correlation_id = request_id or str(uuid.uuid4())
        argv = [str(executable), '--request-id', correlation_id]
        if machine:
            argv.append('--json')
        if scoped:
            argv += ['--workspace', str(workspace_path or workspace)]
        argv += [str(a) for a in args]
        if list(args[:2]) == ['work', 'wait'] and not any(str(a).startswith('--timeout-ms') for a in args):
            argv += ['--timeout-ms', str(bounds.DURABLE_WORK_SECONDS * 1000)]
        stdout, stderr, code, observed = capture_invocation(process_factory or ProcessTree, argv, cwd=cwd or workspace,
            env=execution_environment if execution_environment is not None else environment, scoped=scoped, input=input,
            timeout_seconds=timeout_seconds)
        if timing is not None:
            timing.update(observed)
        accepted = expected if isinstance(expected, tuple) else (expected,)
        if code not in accepted or (stderr if code == 0 else stdout):
            raise ValueError('CLI stream/exit contract differs: ' + ' '.join(map(str, args[:3]))
                             + ' (exit ' + str(code) + ', response ' + (stderr if code else stdout).decode('utf-8')[:2000] + ')')
        if not machine:
            text = (stderr if code else stdout).decode("utf-8")
            if "\x1b" in text or len(text.encode("utf-8")) > 65536:
                raise ValueError("Human display control/size contract differs")
            if coverage is not None:
                coverage.observe(argv[1:], code, scenario=scenario or current_scenario)
            return text
        result = json.loads(stderr if code else stdout)
        if result.get('kind') != 'runtime-response' or result.get('schema_version') != version or (result.get('request_id') != correlation_id and '--request-id' not in args):
            raise ValueError('CLI envelope/correlation differs')
        if coverage is not None:
            coverage.observe(argv[1:], code, scenario=scenario or current_scenario,
                failure=result.get('error'), help_only=isinstance(result.get('result'), dict) and 'help' in result['result'])
        return result

    def phase(label):
        nonlocal current_scenario
        current_scenario = {'media-lifecycle': 'media', 'query-contracts': 'query', 'documented-workflows': 'documented', 'provider-contracts': 'provider'}.get(label, 'core')
        if progress is not None:progress(label)

    phase('help')
    root_help = invoke(['--help'], scoped=False)['result']['help']
    paths = [line.strip().split() for line in root_help.splitlines() if line.startswith('  ')]
    help_workspace = workspace / 'isolated help workspace'
    invoke(['workspace', 'init', help_workspace], scoped=False)
    observations = []
    help_pages = qualify_read_only(invoke, paths, help_workspace, workspace / 'absent-help-workspace', observations=observations)
    coverage = Coverage(help_pages, executable, version, artifact_identity, root_help=root_help)
    coverage.observe(['--json', '--request-id', 'VALUE', '--help'], 0)
    coverage.observe(['--json', '--request-id', 'VALUE', 'workspace', 'init'], 0)
    for args, code, failure, help_only in observations:
        coverage.observe(args, code, failure=failure, help_only=help_only)

    # Human display, clean redirected waits and elected log preflight are part
    # of the same installed CLI contract, rather than source-only rendering.
    phase('operator-feedback')
    human = invoke(['workspace', 'show'], machine=False)
    if human.lstrip().startswith('{') or version not in human or 'Next import:' not in human:
        raise ValueError('Human workspace result is not actionable')
    logfile = workspace / 'operator diagnostic.jsonl'
    invoke(['--debug', '--log-file', logfile, 'doctor'])
    if not logfile.is_file() or not logfile.read_text(encoding='utf-8').strip():
        raise ValueError('Elected diagnostic log missing')
    forbidden_target = workspace / 'invalid log must not initialize'
    invoke(['--log-file', workspace / 'missing-log-parent' / 'operator.jsonl',
            'workspace', 'init', forbidden_target], expected=4, scoped=False)
    if forbidden_target.exists():
        raise ValueError('Log preflight failure performed initialization')
    invoke(['settings', 'set', '--input', workspace / 'absent-input.json'], expected=4)
    shown = invoke(['settings', 'show'])['result']
    query_file = workspace / 'public-column-names.cypher'
    query_file.write_text("RETURN 'public value' AS value, 'public data' AS data, 'public owner' AS owner", encoding='utf-8', newline='\n')
    human_query = invoke(['query', 'native', '--dialect', 'ladybug-cypher', '--file', query_file], machine=False)
    if any(value not in human_query for value in ['public value', 'public data', 'public owner']):
        raise ValueError('Native query public column values omitted')
    input_file = workspace / 'cli-contract-input.json'
    def supplied(args, data, expected=0):
        input_file.write_text(json.dumps(data) + '\n', encoding='utf-8', newline='\n')
        return invoke([*args, '--input', input_file], expected=expected)
    original = shown['sections']['appearance']
    supplied(['settings', 'set'], {'section': 'appearance', 'revision': original['revision'],
                       'value': {'theme': 'dark', 'reduced_motion': True}})['result']
    supplied(['settings', 'set'], {'section': 'appearance', 'revision': original['revision'], 'value': None}, expected=1)
    human_settings = invoke(['settings', 'show'], machine=False)
    human_doctor = invoke(['doctor'], machine=False)
    if 'Theme: dark' not in human_settings or 'Data:' not in human_doctor:
        raise ValueError('Public settings values or doctor data path omitted')
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
    phase('credentials')
    credential = str(uuid.uuid4())
    secret = {'passphrase': 'synthetic-cli-fixture-passphrase', 'value': 'synthetic-cli-fixture-value'}
    invoke(['credentials', 'select', 'vault'])
    invoke(['credentials', 'add', credential], input=json.dumps(secret).encode())
    invoke(['credentials', 'select', 'vault'])
    status = invoke(['credentials', 'status', credential])['result']
    if status.get('state') != 'locked' or not status.get('stored') or status.get('authentication') != 'not-checked':
        raise ValueError('Saved locked vault was misrepresented')
    missing_human = invoke(['credentials', 'status', str(uuid.uuid4())], machine=False)
    if 'Stored: no' not in missing_human or 'remain stored' in missing_human:
        raise ValueError('Missing locked credential falsely reported stored')
    duplicate = invoke(['credentials', 'add', credential], input=json.dumps(secret).encode(), expected=1)
    if duplicate['error'].get('reason') != 'credential_duplicate':
        raise ValueError('Duplicate credential lost its actual cause')
    invoke(['credentials', 'delete', credential], input=b'{}', expected=2)
    invoke(['credentials', 'delete', credential], input=json.dumps({'passphrase': 'wrong-fixture-passphrase'}).encode(), expected=4)
    invoke(['credentials', 'delete', credential], input=json.dumps({'passphrase': secret['passphrase']}).encode())
    invoke(['credentials', 'select', 'session'])

    # A separate stopped workspace proves the real detached report path. The
    # enclosing journey already waits for all elected owners' idle exit.
    cold = workspace / 'cold operator workspace'
    invoke(['workspace', 'init', cold], scoped=False)
    invoke(['--workspace', cold, 'credentials', 'select', 'vault'], scoped=False)
    cold_id = str(uuid.uuid4())
    invoke(['--workspace', cold, 'credentials', 'add', cold_id], scoped=False, input=json.dumps(secret).encode())
    cold_timing = {}
    rejected = invoke(['credentials', 'unlock'], workspace_path=cold,
                      expected=4, input=json.dumps({'passphrase': 'wrong-fixture-passphrase'}).encode(), timing=cold_timing)
    verify_cold_bootstrap(rejected, cold_timing)
    # scoped=True elects the existing detached job lifetime for the cold owner.
    invoke(['credentials', 'unlock'], workspace_path=cold, input=json.dumps({'passphrase': secret['passphrase']}).encode())
    first = invoke(['workspace', 'show'], workspace_path=cold)
    second = invoke(['workspace', 'show'], workspace_path=cold)
    if not first.get('runtime_session_id') or first['runtime_session_id'] != second['runtime_session_id']:
        raise ValueError('Detached runtime did not survive launching-client exit')
    phase('model-acquisition')
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
        completed = invoke(['work', 'wait', registered['work_id']])
        model_id = completed['result']['result']['model_id']
        absent = invoke(['models', 'verify', model_id], expected=4)
        if absent['error'].get('reason') != 'model_not_acquired' or absent['result'].get('readiness') != 'not_acquired':
            raise ValueError('Model readiness context was discarded')
        acquired_good = submit('acquire', manifest)
        waited = invoke(['work', 'wait', acquired_good['work_id']], machine=False)
        if 'State: Succeeded' not in waited or f"Work {acquired_good['work_id']}:" in waited:
            raise ValueError('Redirected wait result contains transient progress')
        named = invoke(['models', 'materialize', model_id])['result']
        file = named['files'][0]
        if named.get('verification') != 'verified' or file['role'] != 'weights' or file['sha256'] != manifest['files'][0]['sha256'] or Path(file['path']).read_bytes() != body:
            raise ValueError('Named model materialization differs from immutable manifest')
        invoke(['artifacts', 'lease-release', file['publication_id'], file['lease_id']])
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
        finish_server(server, thread)
    phase('parallel-regressions')
    process_options = {'process_factory': process_factory} if process_factory is not None else {}
    with parallel_regressions(invoke, workspace, version, documentation, help_pages, executable, environment, observe=coverage.observe, **process_options) as regressions:
        lifecycle = {'state': 'not-included', 'reason': 'exact companions required in extracted packages'}
        if documentation:
            from cli_media_qualification import qualify as qualify_media
            phase('media-lifecycle')
            lifecycle = qualify_media(invoke, supplied, workspace, executable, environment, version, **process_options)
        query_contracts = {'state': 'not-included', 'reason': 'exact companions required in extracted packages'}
        if documentation:
            from cli_query_qualification import qualify as qualify_queries
            phase('query-contracts')
            query_contracts = qualify_queries(invoke, supplied, workspace)
        provider_contracts = {'state': 'not-included', 'reason': 'required in extracted packages'}
        documented = {'state': 'not-included', 'reason': 'source CLI lane; required in extracted packages'}
        examples = {'state': 'not-included', 'reason': 'requires exact installed companions'}
        if documentation:
            from documented_workflows import qualify as qualify_documentation
            # Exercise a post-install workspace against owned loopback adapter fixtures.
            # The main qualification runtime keeps CI markers and all production refusal
            # gates; no inference engine is selected or initialized in this workspace.
            documented_workspace = workspace / 'isolated documented workflows'
            fixture_environment = dict(environment)
            for marker in ['CI', 'GITHUB_ACTIONS', 'TF_BUILD', 'BUILD_BUILDID', 'JENKINS_URL']:
                fixture_environment.pop(marker, None)
            invoke(['workspace', 'init', documented_workspace], scoped=False)
            def documented_invoke(args, **kwargs):
                return invoke(args, workspace_path=documented_workspace,
                              execution_environment=fixture_environment, **kwargs)
            def documented_supplied(args, value, **kwargs):
                path = documented_workspace / 'documented request.json'
                path.write_text(json.dumps(value) + '\n', encoding='utf-8', newline='\n')
                return documented_invoke([*args, '--input', path], **kwargs)
            phase('documented-workflows')
            documented = qualify_documentation(documented_invoke, documented_supplied, documented_workspace, version)
            from cli_provider_qualification import qualify as qualify_providers
            phase('provider-contracts')
            provider_contracts = qualify_providers(documented_invoke, documented_supplied, documented_workspace)
            # Package smoke's existing bounded idle-owner cleanup covers this workspace too.
        phase('join-regressions')
        operator_contracts = regressions['operator'].result()
        input_contracts = regressions['input'].result()
        text_duration_contracts = regressions['text_duration'].result()
        readiness_contracts = regressions['readiness'].result()
        terminal_contracts = regressions['terminal'].result()
        recovery_contracts = regressions['recovery'].result()
        storage_contracts = regressions['storage'].result()
        backup_contracts = regressions['backup'].result()
        if documentation:
            examples = regressions['examples'].result()
    input_file.unlink(missing_ok=True)
    for path in [logfile, Path(str(logfile) + '.1'), Path(str(logfile) + '.2')]:
        if path.exists():
            logged = path.read_text(encoding='utf-8')
            if len(logged.encode('utf-8')) > 1 << 20 or any(value in logged for value in [secret['passphrase'], secret['value'], 'wrong-fixture-passphrase', str(workspace)]):
                raise ValueError('Diagnostic log retained protected or arbitrary input')
    for entry in examples.get('ledger', []):
        coverage.assert_example(entry)
    ledger = coverage.finish({'core': True, 'operator': True, 'input': True, 'text-duration': True, 'readiness': True, 'terminal': True, 'recovery': True, 'storage': True, 'backup': True,
                              **{name: documentation for name in ['media', 'query', 'provider', 'documented', 'examples']}})
    phase('complete')
    return {'coverage': ledger, 'terminal_contracts': terminal_contracts, 'recovery_contracts': recovery_contracts, 'storage_contracts': storage_contracts, 'backup_contracts': backup_contracts, 'readiness_contracts': readiness_contracts, 'operator_contracts': operator_contracts, 'input_contracts': input_contracts, 'text_duration_contracts': text_duration_contracts, 'actual_help_examples': examples, 'provider_contracts': provider_contracts, 'media_lifecycle': lifecycle, 'query_contracts': query_contracts, 'documented_workflows': documented, 'help_paths': len(paths), 'settings': 'passed', 'initial_speaker_cursor': 'passed',
            'protected_vault_delete': 'passed', 'typed_failures': failures,
            'operator_feedback': 'passed', 'cold_startup_failure': 'passed', 'cold_startup_timing': cold_timing, 'detached_owner': 'passed',
            'model_roles_leases': 'passed', 'diagnostic_redaction': 'passed'}
