# SPDX-License-Identifier: Apache-2.0
"""Actual confined backup/restore obstructions, exact retry and byte recovery."""
from contextlib import ExitStack
import hashlib
import json
import os
from pathlib import Path
import queue
import stat
import sys
import tempfile
import threading
import time
import uuid

from cli_recovery_qualification import hashed, identifier, sha
from cli_terminal_qualification import owned_directory, owned_runtime
from process_tree import ProcessTree
import qualification_timing as bounds

CASES = ('backup-parent', 'restore-publication', 'restore-progress')
FAILURES = {'backup-parent': ('unavailable', 4), 'restore-publication': ('unavailable', 4),
            'restore-progress': ('invalid_request', 2)}


def checks_for(name):
    common = ('authority_preserved', 'blocker_unchanged', 'blocker_removed', 'identical_retry',
              'accepted_bytes_verified', 'replay_verified', 'source_unchanged')
    return common + (('no_completed_bundle',) if name == 'backup-parent' else
                     ('source_unavailable', 'bundle_unchanged', 'progress_removed'))


def validate_receipt(receipt, executable_sha256, platform):
    fields = {'format', 'platform', 'executable_sha256', 'origin', 'physical_exhaustion', 'source_workspace_id',
              'bundle_id', 'bundle_digest', 'publication', 'cases', 'cleanup', 'fixture_removed', 'runtime_natural_exit'}
    if (not isinstance(receipt, dict) or set(receipt) != fields
            or receipt['format'] != 'insonic-backup-recovery-qualification-v1'
            or platform not in ('windows', 'linux') or receipt['platform'] != platform
            or not hashed(executable_sha256) or receipt['executable_sha256'] != executable_sha256
            or receipt['origin'] != 'actual-filesystem-obstruction' or receipt['physical_exhaustion'] is not False
            or not identifier(receipt['source_workspace_id']) or not identifier(receipt['bundle_id'])
            or not hashed(receipt['bundle_digest'])
            or any(receipt[k] is not True for k in ('cleanup', 'fixture_removed', 'runtime_natural_exit'))
            or not isinstance(receipt['cases'], dict) or set(receipt['cases']) != set(CASES)):
        raise ValueError('Incomplete or mislabeled backup recovery receipt')
    publication = receipt['publication']
    if (not isinstance(publication, dict) or set(publication) != {'id', 'digest', 'size'}
            or not identifier(publication['id']) or not hashed(publication['digest'])
            or type(publication['size']) is not int or publication['size'] <= 0):
        raise ValueError('Backup publication identity differs')
    for name, case in receipt['cases'].items():
        if (not isinstance(case, dict) or set(case) != {'failed_request_id', 'retry_request_id', 'failed_code', 'bundle_id',
                                                      'bundle_digest', 'verified_digest', 'destination_profile_id', 'checks'}
                or not identifier(case['failed_request_id']) or not identifier(case['retry_request_id'])
                or case['failed_code'] != FAILURES[name][0] or case['bundle_id'] != receipt['bundle_id']
                or case['bundle_digest'] != receipt['bundle_digest'] or case['verified_digest'] != publication['digest']
                or not identifier(case['destination_profile_id']) or not isinstance(case['checks'], dict)
                or set(case['checks']) != set(checks_for(name)) or any(v is not True for v in case['checks'].values())):
            raise ValueError('Backup failure/retry/integrity evidence differs')
        if name == 'backup-parent' and (case['failed_request_id'] != receipt['bundle_id'] or case['retry_request_id'] != receipt['bundle_id']):
            raise ValueError('Backup retry changed request authority')


def block(root, path, *, directory=False):
    root, path = Path(root).resolve(strict=True), Path(path)
    if path.is_symlink() or path.resolve().is_relative_to(root) is not True or path.resolve() == root:
        raise ValueError('Obstruction escapes owned fixture')
    path.parent.mkdir(parents=True, exist_ok=True)
    if directory: path.mkdir()
    marker = path / '.owned-blocker' if directory else path
    data = ('owned-backup-obstruction-' + str(uuid.uuid4())).encode('utf-8')
    with marker.open('xb') as output:
        output.write(data)
        output.flush()
        os.fsync(output.fileno())
    observed = marker.lstat()
    return {'path': path, 'marker': marker, 'directory': directory, 'device': observed.st_dev,
            'inode': observed.st_ino, 'size': observed.st_size, 'sha256': sha(marker)}


def unblock(root, record):
    root = Path(root).resolve(strict=True)
    path, marker = record['path'], record['marker']
    expected = path / '.owned-blocker' if record['directory'] else path
    resolved = path.resolve(strict=True)
    expected_resolved = resolved / '.owned-blocker' if record['directory'] else resolved
    if (marker != expected or path.is_symlink() or marker.is_symlink()
            or not resolved.is_relative_to(root) or resolved == root
            or marker.resolve(strict=True) != expected_resolved):
        raise ValueError('Owned obstruction path changed')
    observed = marker.lstat()
    if (not stat.S_ISREG(observed.st_mode)
            or (observed.st_dev, observed.st_ino, observed.st_size) != (record['device'], record['inode'], record['size'])
            or sha(marker) != record['sha256']):
        raise ValueError('Owned obstruction bytes or identity changed')
    marker.unlink()
    if record['directory']: path.rmdir()


def qualify(invoke, workspace, version, *, executable, environment, process_factory=None):
    deadline = time.monotonic() + bounds.phase_bound(bounds.SUBPROCESS_SECONDS)
    executable = Path(executable).resolve(strict=True)
    image = sha(executable)

    def remaining(limit=bounds.SUBPROCESS_SECONDS):
        value = min(limit, deadline - time.monotonic(), bounds.phase_bound(limit))
        if value <= 0: raise TimeoutError('Backup campaign enclosing deadline exceeded')
        return value

    cases = {}
    with ExitStack() as resources:
        root = resources.enter_context(owned_directory(Path(workspace)))
        private = Path(resources.enter_context(tempfile.TemporaryDirectory(prefix='br-', dir='/tmp' if sys.platform == 'linux' else None)))
        selected = dict(environment)
        for key in ('HOME', 'USERPROFILE', 'APPDATA', 'LOCALAPPDATA', 'XDG_RUNTIME_DIR', 'XDG_CONFIG_HOME', 'XDG_DATA_HOME', 'XDG_CACHE_HOME'):
            selected[key] = str(private)
        source = root / 'source library'

        def call(args, **options):
            options.setdefault('workspace_path', source)
            options.update(execution_environment=selected, timeout_seconds=remaining(options.get('timeout_seconds', bounds.SUBPROCESS_SECONDS)))
            return invoke(args, **options)

        def initialize(path):
            call(['workspace', 'init', path], scoped=False)
            call(['credentials', 'select', 'session'], workspace_path=path)
            return json.loads((path / '.insonic/workspace.json').read_text(encoding='utf-8'))

        config = initialize(source)
        original = root / 'accepted input résumé.txt'
        original.write_bytes(('immutable backup fixture café\n' * 100).encode('utf-8'))
        input_sha = sha(original)
        startup = str(uuid.uuid4())
        with owned_runtime([str(executable), '--workspace', str(source), '--startup-id', startup, '--json', 'runtime', 'serve'],
                           cwd=root, env=selected, gates={}, factory=process_factory or ProcessTree) as owner:
            mailbox = queue.Queue(maxsize=1)
            reader = threading.Thread(target=lambda: mailbox.put(owner.process.stdout.readline(4097)), daemon=True)
            reader.start()
            try: ready = json.loads(mailbox.get(timeout=remaining(bounds.READINESS_SECONDS)))
            finally:
                reader.join(bounds.CLEANUP_SECONDS)
                if reader.is_alive(): raise ValueError('Backup startup reader remains')
            if ready.get('state') != 'ready' or ready.get('startup_id') != startup or ready.get('workspace_id') != config['workspace_id']:
                raise ValueError('Backup runtime readiness identity differs')
            publication = call(['artifacts', 'publish', original, 'derived'])['result']
            output, error = owner.process.communicate(timeout=remaining(bounds.FIXTURE_IDLE_SECONDS))
            if owner.process.returncode != 0 or error or not owner.wait_for_descendants(remaining(bounds.CLEANUP_SECONDS)):
                raise ValueError('Backup source runtime did not exit naturally')
        publication = {k: publication[k] for k in ('id', 'digest', 'size')}
        if publication['digest'] != input_sha or publication['size'] != original.stat().st_size:
            raise ValueError('Accepted backup fixture differs from original input')
        request = str(uuid.uuid4())
        blocked_parent = root / 'backup parent'
        obstruction = block(root, blocked_parent)
        bundle = blocked_parent / 'bundle'
        failed = call(['backup', 'create', bundle], expected=4, request_id=request)
        no_bundle = not bundle.exists()
        preserved = sha(obstruction['marker']) == obstruction['sha256']
        unblock(root, obstruction)
        blocked_parent.mkdir()
        manifest = call(['backup', 'create', bundle], request_id=request)['result']
        replay = call(['backup', 'create', bundle], request_id=request)['result']
        verified = call(['backup', 'verify', bundle])['result']
        cases['backup-parent'] = {'failed_request_id': request, 'retry_request_id': request, 'failed_code': failed['error']['code'],
            'bundle_id': manifest['id'], 'bundle_digest': manifest['sha256'], 'verified_digest': sha(bundle / 'objects' / input_sha),
            'destination_profile_id': config['profiles']['storage']['profile_id'],
            'checks': {'authority_preserved': sha(original) == input_sha, 'blocker_unchanged': preserved,
                       'blocker_removed': blocked_parent.is_dir(), 'identical_retry': manifest['id'] == request,
                       'accepted_bytes_verified': verified['sha256'] == manifest['sha256'] and sha(bundle / 'objects' / input_sha) == input_sha,
                       'replay_verified': replay['sha256'] == manifest['sha256'], 'source_unchanged': sha(original) == input_sha,
                       'no_completed_bundle': no_bundle}}
        source.rename(root / 'withdrawn source library')
        manifest_sha, catalog_sha = sha(bundle / 'manifest.json'), sha(bundle / 'catalog.json')

        for name in CASES[1:]:
            destination = root / name
            destination_config = initialize(destination)
            profile = destination_config['profiles']['storage']
            if profile['adapter_id'] != 'filesystem' or profile['configuration']['root'] != 'artifacts':
                raise ValueError('Backup fixture destination storage differs')
            marker = destination / '.insonic/workspace.json'
            marker_sha = sha(marker)
            if name == 'restore-publication':
                obstruction_path = destination / '.insonic/artifacts/objects' / config['workspace_id'] / profile['profile_id'] / str(profile['profile_revision']) / 'restored'
                obstruction = block(root, obstruction_path)
            else:
                obstruction_path = destination / '.insonic/backup-restore-progress' / manifest['id'] / (publication['id'] + '.json')
                obstruction = block(root, obstruction_path, directory=True)
            failed_request, retry_request = str(uuid.uuid4()), str(uuid.uuid4())
            failed = call(['backup', 'restore', bundle], workspace_path=destination, expected=FAILURES[name][1], request_id=failed_request)
            preserved = sha(marker) == marker_sha and sha(obstruction['marker']) == obstruction['sha256']
            before = root / (name + '-before.json')
            call(['catalog', 'export', before], workspace_path=destination)
            empty = all(not table['rows'] for table in json.loads(before.read_text(encoding='utf-8'))['state'] if table['name'] == 'artifact_publication')
            unblock(root, obstruction)
            restored = call(['backup', 'restore', bundle], workspace_path=destination, request_id=retry_request)['result']
            replay = call(['backup', 'restore', bundle], workspace_path=destination)['result']
            snapshot = root / (name + '-after.json')
            call(['catalog', 'export', snapshot], workspace_path=destination)
            pubs = [json.loads(row[2]) for table in json.loads(snapshot.read_text(encoding='utf-8'))['state'] if table['name'] == 'artifact_publication' for row in table['rows']]
            current = next(p for p in pubs if p['id'] == publication['id'])
            expected_key = 'objects/' + config['workspace_id'] + '/' + profile['profile_id'] + '/' + str(profile['profile_revision']) + '/restored/' + manifest['id'] + '/' + publication['id']
            if current['key'] != expected_key or current['profile_id'] != profile['profile_id'] or current['state'] != 'available':
                raise ValueError('Restored publication authority differs')
            actual = sha(destination / '.insonic/artifacts' / expected_key)
            cases[name] = {'failed_request_id': failed_request, 'retry_request_id': retry_request, 'failed_code': failed['error']['code'],
                'bundle_id': restored['id'], 'bundle_digest': restored['sha256'], 'verified_digest': actual,
                'destination_profile_id': profile['profile_id'], 'checks': {
                    'authority_preserved': preserved and empty, 'blocker_unchanged': preserved, 'blocker_removed': not obstruction_path.is_file() if name == 'restore-publication' else not obstruction_path.exists(),
                    'identical_retry': restored['sha256'] == manifest['sha256'], 'accepted_bytes_verified': actual == input_sha == current['digest'],
                    'replay_verified': replay['sha256'] == manifest['sha256'], 'source_unchanged': sha(original) == input_sha,
                    'source_unavailable': not source.exists(), 'bundle_unchanged': sha(bundle / 'manifest.json') == manifest_sha and sha(bundle / 'catalog.json') == catalog_sha,
                    'progress_removed': not (destination / '.insonic/backup-restore-progress' / manifest['id']).exists()}}
        receipt = {'format': 'insonic-backup-recovery-qualification-v1', 'platform': 'windows' if os.name == 'nt' else sys.platform,
                   'executable_sha256': image, 'origin': 'actual-filesystem-obstruction', 'physical_exhaustion': False,
                   'source_workspace_id': config['workspace_id'], 'bundle_id': manifest['id'], 'bundle_digest': manifest['sha256'],
                   'publication': publication, 'cases': cases, 'cleanup': True, 'fixture_removed': True, 'runtime_natural_exit': True}
        validate_receipt(receipt, image, receipt['platform'])
    receipt['fixture_removed'] = not root.exists() and not private.exists()
    remaining()
    validate_receipt(receipt, image, receipt['platform'])
    return receipt
