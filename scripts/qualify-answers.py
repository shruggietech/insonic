#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Run synthetic answer qualification or an explicitly elected model evaluation."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import queue
import sys
import threading
import time
import uuid

from cli_answer_synthesis_qualification import qualify
from cli_contract_qualification import capture_invocation
from cli_terminal_qualification import owned_runtime
from process_tree import ProcessTree
import qualification_timing as bounds


def campaign(executable, workspace, output, *, configuration=None, existing=False,
             media_tools=None, processing_tools=None, environment=None):
    executable, workspace, output = Path(executable).resolve(strict=True), Path(workspace).resolve(), Path(output).resolve()
    env = dict(os.environ if environment is None else environment)
    markers = ('CI', 'GITHUB_ACTIONS', 'TF_BUILD', 'BUILD_BUILDID', 'JENKINS_URL')
    if configuration is not None and any(env.get(k) for k in markers):
        raise ValueError('Elected model evaluation cannot run inside deterministic CI')
    if configuration is None:
        for marker in markers: env.pop(marker, None)
    started = time.monotonic()
    deadline = started + bounds.phase_bound(bounds.JOURNEY_SECONDS)
    def invoke(args, *, expected=0, machine=True, scoped=True, **options):
        argv = [str(executable), '--request-id', str(uuid.uuid4())]
        if machine: argv.append('--json')
        if scoped: argv += ['--workspace', str(workspace)]
        argv += [str(a) for a in args]
        if list(args[:2]) == ['work', 'wait']:
            argv += ['--timeout-ms', str(bounds.DURABLE_WORK_SECONDS * 1000)]
        stdout, stderr, code, _ = capture_invocation(ProcessTree, argv, cwd=executable.parent, env=env,
            scoped=scoped, input=options.get('input'), timeout_seconds=min(options.get('timeout_seconds', bounds.SUBPROCESS_SECONDS), bounds.SUBPROCESS_SECONDS, deadline-time.monotonic()))
        accepted = expected if isinstance(expected, tuple) else (expected,)
        if code not in accepted or (stderr if code == 0 else stdout):
            raise ValueError('Answer CLI invocation failed: ' + ' '.join(map(str, args[:2])) +
                ' exit ' + str(code) + ' ' + (stderr if code else stdout).decode('utf-8')[:2000])
        raw = (stderr if code else stdout).decode('utf-8')
        return json.loads(raw) if machine else raw
    def supplied(args, value, **options):
        path = workspace / 'answer-evaluation-input.json'
        path.write_text(json.dumps(value) + '\n', encoding='utf-8', newline='\n')
        return invoke([*args, '--input', path], **options)
    if not existing:
        if workspace.exists(): raise ValueError('Evaluation requires a new workspace path')
        invoke(['workspace', 'init', workspace], scoped=False)
        if configuration is None: invoke(['credentials', 'select', 'session'])
    elif not (workspace / '.insonic/workspace.json').is_file():
        raise ValueError('Existing evaluation workspace is absent')
    # A caller may provision alternative backends and credentials, but evaluation
    # must never populate an existing media library or overwrite its sources.
    snapshot = workspace / ('answer-empty-' + uuid.uuid4().hex + '.json')
    invoke(['catalog', 'export', snapshot])
    try:
        if json.loads(snapshot.read_text(encoding='utf-8'))['records'].get('library'):
            raise ValueError('Evaluation workspace already contains media')
    finally: snapshot.unlink()
    startup = str(uuid.uuid4())
    with owned_runtime([str(executable), '--workspace', str(workspace), '--startup-id', startup,
                        '--json', 'runtime', 'serve'], cwd=executable.parent, env=env, gates={}) as owner:
        mailbox = queue.Queue(maxsize=1)
        reader = threading.Thread(target=lambda: mailbox.put(owner.process.stdout.readline(4097)), daemon=True)
        reader.start()
        try:
            ready = json.loads(mailbox.get(timeout=bounds.READINESS_SECONDS))
        finally:
            reader.join(bounds.CLEANUP_SECONDS)
            if reader.is_alive(): raise ValueError('Answer runtime readiness reader did not release')
        if ready.get('state') != 'ready' or ready.get('startup_id') != startup:
            raise ValueError('Answer evaluation runtime ownership differs')
        if media_tools: invoke(['media', 'tools', Path(media_tools).resolve(strict=True)])
        if processing_tools: invoke(['processing', 'tools', Path(processing_tools).resolve(strict=True)])
        version = (Path(__file__).resolve().parents[1] / 'VERSION').read_text().strip()
        receipt = qualify(invoke, supplied, workspace, version, configuration=configuration, deadline=deadline)
        with executable.open('rb') as source:
            receipt['executable_sha256'] = hashlib.file_digest(source, 'sha256').hexdigest()
        receipt['schema_version'] = version
        receipt['qualification_seconds'] = round(time.monotonic() - started, 3)
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')
        stdout, stderr = owner.process.communicate(timeout=min(bounds.FIXTURE_IDLE_SECONDS, deadline-time.monotonic()))
        if owner.process.returncode or stderr or not owner.wait_for_descendants(bounds.CLEANUP_SECONDS):
            raise ValueError('Answer evaluation runtime did not release naturally')
    return receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('executable', type=Path)
    parser.add_argument('workspace', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--existing-workspace', action='store_true', help='Use an empty, already provisioned evaluation workspace')
    parser.add_argument('--configuration', type=Path, help='Answer synthesis configuration JSON; no inline credential values')
    parser.add_argument('--elect-model', action='store_true', help='Explicitly send synthetic corpus evidence to the configured model')
    parser.add_argument('--media-tools', type=Path)
    parser.add_argument('--processing-tools', type=Path)
    args = parser.parse_args()
    if args.elect_model != (args.configuration is not None):
        parser.error('--configuration and --elect-model are required together')
    configuration = json.loads(args.configuration.read_text(encoding='utf-8')) if args.configuration else None
    receipt = campaign(args.executable, args.workspace, args.output, configuration=configuration,
        existing=args.existing_workspace, media_tools=args.media_tools, processing_tools=args.processing_tools)
    print(json.dumps({k: receipt[k] for k in ('passed', 'mode', 'model_quality', 'corpus_sha256')}))
    return 0 if receipt['passed'] else 1


if __name__ == '__main__':
    sys.exit(main())
