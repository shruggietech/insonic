# SPDX-License-Identifier: Apache-2.0
"""Bounded generated failure observations, captured before owned teardown."""
from contextlib import contextmanager
import json
import os
from pathlib import Path
import re
import stat
import time
import uuid


STAGES = {'stdout_pipe', 'stdin_pipe', 'command_start'}
STATES = {'queued', 'running', 'succeeded', 'failed', 'cancelled', 'interrupted'}
MODES = {'human', 'json', 'quiet', 'redirected', 'interrupt', 'timeout', 'cancelled', 'failed'}
IDENTITIES = {'startup_id', 'workspace_id', 'runtime_session_id', 'work_id'}
NATIVE_FIELDS = {'kind', 'stage', 'platform', 'pid', 'has_errno', 'error_number'}


def diagnostic_environment(environment):
    selected = dict(environment)
    if selected.get('GITHUB_ACTIONS') == 'true':
        selected['INSONIC_PROCESS_STARTUP_DIAGNOSTICS'] = '1'
    return selected


def canonical_id(value):
    try:
        return isinstance(value, str) and str(uuid.UUID(value)) == value
    except ValueError:
        return False


def bounded_read(path, limit, *, not_before=0):
    if path.is_symlink():
        return None
    try:
        with path.open('rb') as file:
            metadata = os.fstat(file.fileno())
            if not stat.S_ISREG(metadata.st_mode) or metadata.st_mtime_ns < not_before:
                return None
            raw = file.read(limit + 1)
        return raw if len(raw) <= limit else None
    except OSError:
        return None


def process_observation(process):
    pid = getattr(process, 'pid', None)
    if type(pid) is not int or not 0 < pid <= 0xffffffff:
        pid = None
    try:
        code = process.poll()
        if code is not None and (type(code) is not int or not -0x80000000 <= code <= 0xffffffff):
            raise ValueError('Invalid process outcome')
        observation = 'running' if code is None else 'exited'
    except Exception:
        code, observation = None, 'unavailable'
    return {'pid': pid, 'exit_code': code, 'observation': observation}


class FailureEvidence:
    def __init__(self, digest, environment, *, platform=None):
        root = Path(environment.get('GITHUB_WORKSPACE', ''))
        self.directory = root / 'build' / 'native' if environment.get('GITHUB_ACTIONS') == 'true' and root.is_absolute() else None
        self.digest = digest if isinstance(digest, str) and re.fullmatch('[0-9a-f]{64}', digest) else None
        self.platform = platform or ('windows' if os.name == 'nt' else 'linux')
        self.identities, self.clients = {}, []
        self.written = False
        self.campaign_id = str(uuid.uuid4())
        self.started = time.time_ns()

    def track(self, child, request_id, mode, log, *, work_id=None):
        if (not canonical_id(request_id) or mode not in MODES or len(self.clients) >= 8
                or work_id is not None and not canonical_id(work_id)):
            raise ValueError('Invalid terminal failure correlation')
        self.clients.append((child, request_id, mode, Path(log), work_id))

    def native_records(self, pid):
        if pid is None:
            return []
        records = []
        # The existing producer has exactly sixteen bounded sequence slots.
        # Never scan or copy another client's receipts.
        for index in range(1, 17):
            raw = bounded_read(self.directory / f'process-startup-{pid}-{index}-receipt.json', 1024, not_before=self.started)
            if raw is None:
                continue
            try:
                record = json.loads(raw)
            except (ValueError, UnicodeError, RecursionError):
                continue
            if (isinstance(record, dict) and set(record) == NATIVE_FIELDS
                    and record['kind'] == 'process-startup-diagnostic'
                    and isinstance(record['stage'], str) and record['stage'] in STAGES and record['platform'] == self.platform
                    and type(record['pid']) is int and record['pid'] == pid
                    and type(record['has_errno']) is bool
                    and type(record['error_number']) is int and 0 <= record['error_number'] <= 0xffffffffffffffff
                    and (record['has_errno'] or record['error_number'] == 0)):
                records.append(record)
        return records

    @staticmethod
    def events(path, request_id):
        raw = bounded_read(path, 1 << 20)
        if raw is None:
            return []
        events = []
        for line in raw.splitlines(keepends=True):
            if not line.endswith(b'\n'):
                continue
            try:
                event = json.loads(line)
            except (ValueError, UnicodeError, RecursionError):
                continue
            if (isinstance(event, dict) and event.get('request_id') == request_id
                    and event.get('operation') == 'work.wait'
                    and isinstance(event.get('state'), str) and event['state'] in STATES):
                events.append({'request_id': request_id, 'operation': 'work.wait', 'state': event['state']})
        return events[-12:]

    def capture(self, owner):
        if self.written or self.directory is None or self.digest is None:
            return
        clients = []
        for child, request_id, mode, log, work_id in self.clients:
            observed = process_observation(child.owner.process if child is not None else None)
            native = self.native_records(observed['pid'])
            clients.append({'request_id': request_id, 'work_id': work_id, 'mode': mode, 'process': observed,
                            'startup': native, 'startup_observation': 'recorded' if native else 'unavailable',
                            'events': self.events(log, request_id)})
        record = {'format': 'insonic-terminal-failure-v1', 'platform': self.platform,
                  'executable_sha256': self.digest, 'campaign_id': self.campaign_id,
                  'identities': {key: value for key, value in self.identities.items() if key in IDENTITIES and canonical_id(value)},
                  'owner': process_observation(owner.process), 'clients': clients}
        raw = (json.dumps(record, separators=(',', ':'), ensure_ascii=True) + '\n').encode('utf-8')
        if len(raw) > 65536:
            raise ValueError('Terminal failure evidence exceeded its bound')
        self.directory.mkdir(parents=True, exist_ok=True)
        # Exclusive creation: an unrelated artifact can never be overwritten.
        with (self.directory / f'terminal-failure-{self.campaign_id}-receipt.json').open('xb') as file:
            file.write(raw)
        self.written = True

    @contextmanager
    def watch(self, owner):
        try:
            yield
        except BaseException as primary:
            self.preserve(owner, primary)
            raise

    def preserve(self, owner, primary):
        try:
            self.capture(owner)
        except Exception:
            primary.add_note('Terminal failure evidence could not be retained')
