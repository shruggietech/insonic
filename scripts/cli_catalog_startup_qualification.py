# SPDX-License-Identifier: Apache-2.0
"""Installed cold-start incompatibility over an exclusively owned disposable catalog."""
from contextlib import closing
import hashlib
import json
from pathlib import Path
import queue
import sqlite3
import threading
import time
import uuid

from cli_terminal_qualification import owned_runtime
from process_tree import ProcessTree
import qualification_timing as bounds


def qualify(invoke, workspace, version, *, executable, environment, process_factory=None):
    started = time.monotonic()
    deadline = started + bounds.phase_bound(bounds.SUBPROCESS_SECONDS)
    original_invoke = invoke
    def remaining():
        left = deadline - time.monotonic()
        if left <= 0:
            raise TimeoutError('Catalog startup qualification deadline exceeded')
        return left
    def invoke(args, **options):
        options['timeout_seconds'] = min(options.get('timeout_seconds', bounds.SUBPROCESS_SECONDS), remaining())
        return original_invoke(args, **options)
    workspace = Path(workspace)
    invoke(['credentials', 'select', 'session'])
    # Own the valid cold runtime explicitly so release is observed, not guessed
    # from a failed lock probe or repeated client process polling.
    startup = str(uuid.uuid4())
    with owned_runtime([str(executable), '--workspace', str(workspace), '--startup-id', startup,
                        '--json', 'runtime', 'serve'], cwd=workspace, env=environment,
                       gates={}, factory=process_factory or ProcessTree) as owner:
        mailbox = queue.Queue(maxsize=1)
        reader = threading.Thread(target=lambda: mailbox.put(owner.process.stdout.readline(4097)), daemon=True)
        reader.start()
        try:
            ready = json.loads(mailbox.get(timeout=min(bounds.READINESS_SECONDS, remaining())))
        finally:
            reader.join(bounds.CLEANUP_SECONDS)
            if reader.is_alive():
                raise ValueError('Catalog startup reader did not release')
        workspace_id = json.loads((workspace / '.insonic/workspace.json').read_text(encoding='utf-8'))['workspace_id']
        if (ready.get('kind') != 'runtime-startup' or ready.get('stage') != 'ready'
                or ready.get('workspace_id') != workspace_id or ready.get('state') != 'ready' or ready.get('startup_id') != startup
                or ready.get('schema_version') != version):
            raise ValueError('Valid catalog cold startup did not report correlated readiness')
        invoke(['catalog', 'show'])
        appearance = invoke(['settings', 'show'])['result']['sections']['appearance']
        setting = workspace / 'accepted-setting.json'
        setting.write_text(json.dumps({'section': 'appearance', 'revision': appearance['revision'],
                                       'value': {**appearance['value'], 'theme': 'dark'}}) + '\n', encoding='utf-8', newline='\n')
        invoke(['settings', 'set', '--input', setting])
        accepted = invoke(['settings', 'show'])['result']['sections']['appearance']
        if accepted['value']['theme'] != 'dark':
            raise ValueError('Catalog fixture did not accept its setting')
        # This succeeds only after the exact owned process naturally releases.
        stdout, stderr = owner.process.communicate(timeout=min(bounds.FIXTURE_IDLE_SECONDS, remaining()))
        if owner.process.returncode or stdout or stderr or not owner.wait_for_descendants(bounds.CLEANUP_SECONDS):
            raise ValueError('Valid catalog runtime did not release naturally')
    snapshot = workspace / 'exclusive-export.json'
    invoke(['catalog', 'export', snapshot])
    catalog = workspace / '.insonic/catalog.sqlite'
    with closing(sqlite3.connect(catalog)) as connection:
        connection.execute('UPDATE catalog_schema SET version=999')
        connection.commit()
        before = '\n'.join(connection.iterdump())
    for command in (['catalog', 'show'], ['catalog', 'migrate']):
        failure = invoke(command, expected=3)['error']
        if failure.get('code') != 'incompatible_version' or failure.get('reason') != 'catalog_schema_incompatible':
            raise ValueError('Catalog incompatibility lost through installed CLI')
        message = failure.get('message', '').lower()
        if 'compatible' not in message or 'backup' not in message:
            raise ValueError('Catalog recovery guidance is absent')
        human = invoke(command, expected=3, machine=False).lower()
        if 'compatible' not in human or 'backup' not in human:
            raise ValueError('Human catalog recovery guidance is absent')
        with closing(sqlite3.connect(catalog)) as connection:
            after = '\n'.join(connection.iterdump())
            marker = connection.execute('SELECT version FROM catalog_schema').fetchone()[0]
        if after != before or marker != 999:
            raise ValueError('Refusal changed catalog accepted data or version')
    return {'state': 'passed', 'issues': [255], 'valid_cold_start': True,
            'runtime_natural_exit': True, 'exclusive_export': True, 'schema_marker': 999,
            'commands': ['catalog show', 'catalog migrate'], 'exit_code': 3,
            'reason': 'catalog_schema_incompatible', 'accepted_records_unchanged': True, 'accepted_setting': True,
            'logical_catalog_sha256': hashlib.sha256(before.encode()).hexdigest(),
            'elapsed_seconds': round(time.monotonic() - started, 3)}
