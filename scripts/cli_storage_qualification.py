# SPDX-License-Identifier: Apache-2.0
"""Bounded actual filesystem import refusal and ordinary re-import acceptance."""
from contextlib import ExitStack
import copy
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

from cli_media_qualification import pcm_fixture
from cli_recovery_qualification import derived, digest, entered_date_preserved, hashed, identifier, media_tools, sha
from cli_terminal_qualification import owned_directory, owned_runtime
from process_tree import ProcessTree
import qualification_timing as bounds

CHECKS = ('source_unchanged', 'sentinel_unchanged', 'no_incomplete_admission', 'blocker_removed',
          'canonical_verified', 'report_verified', 'metadata_provenance', 'date_provenance',
          'idempotent_replay', 'prior_work_retained')


def validate_receipt(receipt, executable_sha256, platform):
    fields = {'format', 'platform', 'executable_sha256', 'origin', 'physical_exhaustion',
              'cleanup', 'fixture_removed', 'runtime_natural_exit', 'cases'}
    if (not isinstance(receipt, dict) or set(receipt) != fields
            or receipt['format'] != 'insonic-storage-failure-qualification-v1'
            or platform not in ('windows', 'linux') or receipt['platform'] != platform
            or not hashed(executable_sha256) or receipt['executable_sha256'] != executable_sha256
            or receipt['origin'] != 'actual-filesystem-obstruction' or receipt['physical_exhaustion'] is not False
            or any(receipt[k] is not True for k in ('cleanup', 'fixture_removed', 'runtime_natural_exit'))
            or not isinstance(receipt['cases'], dict) or set(receipt['cases']) != {'canonical-publication'}):
        raise ValueError('Incomplete or mislabeled storage-failure receipt')
    case = receipt['cases']['canonical-publication']
    if (not isinstance(case, dict) or set(case) != {'source_sha256', 'source_size', 'sentinel_id', 'sentinel_sha256',
                                                  'blocker_sha256', 'blocked_digest', 'failed', 'recovered', 'checks'}
            or any(not hashed(case[k]) for k in ('source_sha256', 'sentinel_sha256', 'blocker_sha256', 'blocked_digest'))
            or type(case['source_size']) is not int or case['source_size'] <= 0 or not identifier(case['sentinel_id'])
            or not isinstance(case['checks'], dict) or set(case['checks']) != set(CHECKS)
            or any(value is not True for value in case['checks'].values())):
        raise ValueError('Storage-failure source/outcome authority differs')
    failed, recovered = case['failed'], case['recovered']
    if (not isinstance(failed, dict) or set(failed) != {'work_id', 'state', 'partial', 'item_state', 'error', 'stage'}
            or not identifier(failed['work_id']) or failed['state'] != 'failed' or failed['partial'] is not True
            or failed['item_state'] != 'failed' or failed['error'] != 'unavailable' or failed['stage'] != 'canonical-publication'
            or not isinstance(recovered, dict) or set(recovered) != {'work_id', 'state', 'partial', 'item_state', 'entry_id', 'canonical', 'report'}
            or not identifier(recovered['work_id']) or recovered['work_id'] == failed['work_id']
            or recovered['state'] != 'succeeded' or recovered['partial'] is not False or recovered['item_state'] != 'admitted'
            or not identifier(recovered['entry_id']) or recovered['entry_id'] == case['sentinel_id']):
        raise ValueError('Storage-failure partial batch/recovery identity differs')
    for kind, label in [('canonical', 'canonical-0'), ('report', 'canonical-report-0')]:
        publication = recovered[kind]
        if (not isinstance(publication, dict) or set(publication) != {'id', 'digest', 'size'}
                or publication['id'] != derived(recovered['work_id'], label) or not hashed(publication['digest'])
                or type(publication['size']) is not int or publication['size'] <= 0):
            raise ValueError('Storage recovery publication is not bound to its work')
    if recovered['canonical']['digest'] != case['blocked_digest']:
        raise ValueError('Recovered canonical bytes differ from the obstructed source')


def remove_blocker(library, marker):
    """Remove exactly the owned regular-file obstruction, never a computed tree."""
    record = json.loads(Path(marker).read_text(encoding='utf-8'))
    config = json.loads((Path(library) / '.insonic/workspace.json').read_text(encoding='utf-8'))
    profile = config['profiles']['storage']
    if (not identifier(config.get('workspace_id')) or not identifier(profile.get('profile_id'))
            or type(profile.get('profile_revision')) is not int or profile['profile_revision'] < 1
            or profile.get('adapter_id') != 'filesystem' or profile.get('configuration', {}).get('root') != 'artifacts'
            or not isinstance(record, dict) or set(record) != {'path', 'digest', 'device', 'inode', 'size', 'sha256'}
            or not hashed(record.get('digest')) or not hashed(record.get('sha256'))):
        raise ValueError('Storage obstruction authority is invalid')
    wanted = (Path(library) / '.insonic/artifacts/objects' / config['workspace_id'] / profile['profile_id']
              / str(profile['profile_revision']) / 'sha256' / record['digest'][:2] / record['digest'])
    if not hashed(record['digest']) or str(wanted) != record['path']:
        raise ValueError('Storage obstruction path differs from owned workspace authority')
    library_root = Path(library).resolve(strict=True)
    relative = wanted.relative_to(Path(library))
    resolved = wanted.resolve(strict=True)
    if resolved != library_root / relative or not resolved.is_relative_to(library_root):
        raise ValueError('Storage obstruction escapes its owned library')
    observed = wanted.lstat()
    if (not stat.S_ISREG(observed.st_mode) or wanted.is_symlink()
            or (observed.st_dev, observed.st_ino, observed.st_size) != (record['device'], record['inode'], record['size'])
            or sha(wanted) != record['sha256']):
        raise ValueError('Storage obstruction ownership changed')
    wanted.unlink()
    return record


def fixture_tools(tools, root, library):
    selected = copy.deepcopy(tools)
    probe = root / 'storage probe.py'
    original, original_sha = selected['ffprobe']['path'], selected['ffprobe']['sha256']
    if sha(original) != original_sha:
        raise ValueError('Selected storage fixture probe differs')
    source = '''import hashlib, json, os, pathlib, subprocess, sys
root=pathlib.Path(__file__).parent
library=pathlib.Path(LIBRARY)
with open(ORIGINAL,'rb') as f:
    if hashlib.file_digest(f,'sha256').hexdigest()!=PINNED: raise ValueError('Pinned probe changed')
args=sys.argv[1:]
result=subprocess.run([ORIGINAL,*args],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,
    creationflags=subprocess.CREATE_NO_WINDOW if os.name=='nt' else 0)
candidate=pathlib.Path(args[-1]) if args else pathlib.Path('')
if result.returncode==0 and (root/'armed').exists() and candidate.name.startswith('canonical.') and candidate.parent.name.startswith('admission-'):
    with candidate.open('rb') as f: digest=hashlib.file_digest(f,'sha256').hexdigest()
    config=json.loads((library/'.insonic/workspace.json').read_text(encoding='utf-8'))
    profile=config['profiles']['storage']
    if profile['adapter_id']!='filesystem' or profile['configuration']['root']!='artifacts': raise ValueError('Fixture storage differs')
    blocker=library/'.insonic/artifacts/objects'/config['workspace_id']/profile['profile_id']/str(profile['profile_revision'])/'sha256'/digest[:2]/digest
    blocker.parent.mkdir(parents=True,exist_ok=True)
    with blocker.open('xb') as f:
        f.write((root/'blocker-bytes').read_bytes()); f.flush(); os.fsync(f.fileno())
    observed=blocker.lstat()
    with blocker.open('rb') as f: blocker_sha=hashlib.file_digest(f,'sha256').hexdigest()
    (root/'obstruction.json').write_text(json.dumps({'path':str(blocker),'digest':digest,'device':observed.st_dev,'inode':observed.st_ino,
        'size':observed.st_size,'sha256':blocker_sha})+'\\n',encoding='utf-8')
sys.stdout.buffer.write(result.stdout);sys.stderr.buffer.write(result.stderr);sys.exit(result.returncode)
'''.replace('LIBRARY', repr(str(library))).replace('ORIGINAL', repr(original)).replace('PINNED', repr(original_sha))
    probe.write_text(source, encoding='utf-8', newline='\n')
    selected['ffprobe'] = {'path': str(probe), 'sha256': sha(probe), 'version': tools['ffprobe']['version'],
                           'interpreter': {'path': sys.executable, 'sha256': sha(sys.executable)}}
    return selected


def qualify(invoke, workspace, version, executable, environment, *, source_tools=None, process_factory=None):
    deadline = time.monotonic() + bounds.phase_bound(bounds.SUBPROCESS_SECONDS)
    executable = Path(executable).resolve(strict=True)
    image = sha(executable)

    def remaining(limit=bounds.SUBPROCESS_SECONDS):
        value = min(limit, deadline - time.monotonic(), bounds.phase_bound(limit))
        if value <= 0:
            raise TimeoutError('Storage fixture enclosing deadline exceeded')
        return value

    with ExitStack() as resources:
        directory = Path(workspace) / 'storage failure café'
        directory.mkdir()
        resources.callback(directory.rmdir)
        root = resources.enter_context(owned_directory(directory))
        private = Path(resources.enter_context(tempfile.TemporaryDirectory(prefix='sf-', dir='/tmp' if sys.platform == 'linux' else None)))
        environment = dict(environment)
        for key in ('HOME', 'USERPROFILE', 'APPDATA', 'LOCALAPPDATA', 'XDG_RUNTIME_DIR', 'XDG_CONFIG_HOME', 'XDG_DATA_HOME', 'XDG_CACHE_HOME'):
            environment[key] = str(private)
        library = root / 'owned library'

        def call(args, **options):
            options['timeout_seconds'] = min(options.get('timeout_seconds', remaining()), remaining())
            return invoke(args, workspace_path=library, execution_environment=environment, **options)

        call(['workspace', 'init', library], scoped=False)
        call(['credentials', 'select', 'session'])
        tools = fixture_tools(media_tools(executable, source_tools), root, library)
        (library / '.insonic/media-tools.json').write_text(json.dumps(tools) + '\n', encoding='utf-8', newline='\n')
        source, sentinel_source = root / 'source résumé.wav', root / 'sentinel.wav'
        pcm_fixture(source, 1)
        pcm_fixture(sentinel_source, 2)
        source_digest, source_size = sha(source), source.stat().st_size

        def snapshot():
            target = root / ('snapshot-' + str(uuid.uuid4()) + '.json')
            call(['catalog', 'export', target])
            return json.loads(target.read_text(encoding='utf-8'))

        startup_id = str(uuid.uuid4())
        owner = resources.enter_context(owned_runtime([str(executable), '--workspace', str(library), '--startup-id', startup_id,
                                                       '--json', 'runtime', 'serve'], cwd=root, env=environment, gates={},
                                                      factory=process_factory or ProcessTree))
        mailbox = queue.Queue(maxsize=1)
        reader = threading.Thread(target=lambda: mailbox.put(owner.process.stdout.readline(4097)), daemon=True)
        reader.start()
        try:
            ready = json.loads(mailbox.get(timeout=remaining(bounds.READINESS_SECONDS)))
        finally:
            reader.join(bounds.CLEANUP_SECONDS)
            if reader.is_alive():
                raise RuntimeError('Storage fixture startup reader remains')
        shown = call(['workspace', 'show'])
        if ready.get('startup_id') != startup_id or ready.get('state') != 'ready' or ready.get('workspace_id') != shown.get('result', {}).get('workspace_id') or not shown.get('runtime_session_id'):
            raise ValueError('Storage fixture runtime readiness differs')
        first = call(['media', 'import', sentinel_source, '--new-entry', '--no-pipeline'])['result']['work_id']
        baseline = call(['work', 'wait', first])['result']
        before = snapshot()['records']['library']
        if baseline['state'] != 'succeeded' or baseline['result']['partial'] or len(before) != 1:
            raise ValueError('Storage sentinel did not become accepted')
        sentinel = before[0]
        sentinel_digest = digest(sentinel)
        (root / 'blocker-bytes').write_bytes(('owned-storage-obstruction-' + str(uuid.uuid4())).encode())
        (root / 'armed').write_bytes(b'owned\n')
        args = ['media', 'import', source, '--new-entry', '--no-pipeline', '--originated-at', '2020-01-02T03:04:05.123456789Z']
        failed_request = str(uuid.uuid4())
        failed_id = call(args, request_id=failed_request)['result']['work_id']
        failed = call(['work', 'wait', failed_id], expected=4)['result']
        failed_item = failed['result']['items'][0]
        after_failure = snapshot()
        records = after_failure['records']['library']
        if (failed['state'] != 'failed' or failed['result']['partial'] is not True or failed_item['state'] != 'failed'
                or failed_item['error'] != 'unavailable' or 'admission_stage:canonical-publication' not in failed_item.get('notices', [])
                or len(records) != 1 or digest(records[0]) != sentinel_digest):
            raise ValueError('Actual filesystem refusal did not preserve accepted authority: ' + json.dumps(failed)[:2500])
        if sha(source) != source_digest or source.stat().st_size != source_size:
            raise ValueError('Filesystem refusal changed original input')
        marker = root / 'obstruction.json'
        record = remove_blocker(library, marker)
        (root / 'armed').unlink()
        request = str(uuid.uuid4())
        recovered_id = call(args, request_id=request)['result']['work_id']
        recovered = call(['work', 'wait', recovered_id])['result']
        item = recovered['result']['items'][0]
        final = snapshot()
        records = final['records']['library']
        entry = next((e for e in records if e['id'] != sentinel['id']), None)
        kept = next((e for e in records if e['id'] == sentinel['id']), None)
        if (recovered_id == failed_id or recovered['state'] != 'succeeded' or recovered['result']['partial']
                or item['state'] != 'admitted' or len(records) != 2 or entry is None or kept is None):
            raise ValueError('Cleared filesystem refusal did not admit exactly one new recording')
        pubs = []
        for label in ('canonical-0', 'canonical-report-0'):
            value = call(['artifacts', 'show', derived(recovered_id, label)])['result']
            if value['state'] != 'available' or call(['artifacts', 'verify', value['id']])['result']['verified'] is not True:
                raise ValueError('Recovered storage publication is not verified')
            pubs.append({k: value[k] for k in ('id', 'digest', 'size')})
        canonical, report = pubs
        materialized = call(['artifacts', 'materialize', report['id'], str(report['size'])])['result']
        try:
            raw = Path(materialized['path']).read_bytes()
            if hashlib.sha256(raw).hexdigest() != report['digest']:
                raise ValueError('Recovered capture report bytes differ')
            bundle = json.loads(raw)
        finally:
            call(['artifacts', 'lease-release', report['id'], materialized['lease']['id']])
        replay = call(args, request_id=request)['result']['work_id']
        prior = next((w for w in final['records']['works'] if w['id'] == failed_id), None)
        checks = {'source_unchanged': sha(source) == source_digest and source.stat().st_size == source_size,
                  'sentinel_unchanged': digest(kept) == sentinel_digest and call(['artifacts', 'verify', sentinel['original_publication_id']])['result']['verified'] is True,
                  'no_incomplete_admission': len(after_failure['records']['library']) == 1,
                  'blocker_removed': Path(record['path']).is_dir() and not Path(record['path']).is_symlink(),
                  'canonical_verified': entry['original_publication_id'] == canonical['id'] and entry['digest'] == canonical['digest'] == record['digest'],
                  'report_verified': entry['report_publication_ids'] == [report['id']],
                  'metadata_provenance': entry['metadata'] == bundle['metadata'] and bundle['source_digest'] == source_digest and bundle['source_size'] == source_size,
                  'date_provenance': entered_date_preserved(entry['dates']),
                  'idempotent_replay': replay == recovered_id and len(snapshot()['records']['library']) == 2,
                  'prior_work_retained': prior is not None and prior['state'] == 'failed'}
        case = {'source_sha256': source_digest, 'source_size': source_size, 'sentinel_id': sentinel['id'], 'sentinel_sha256': sentinel_digest,
                'blocker_sha256': record['sha256'], 'blocked_digest': record['digest'],
                'failed': {'work_id': failed_id, 'state': failed['state'], 'partial': failed['result']['partial'],
                           'item_state': failed_item['state'], 'error': failed_item['error'], 'stage': 'canonical-publication'},
                'recovered': {'work_id': recovered_id, 'state': recovered['state'], 'partial': recovered['result']['partial'],
                              'item_state': item['state'], 'entry_id': entry['id'], 'canonical': canonical, 'report': report}, 'checks': checks}
        receipt = {'format': 'insonic-storage-failure-qualification-v1', 'platform': 'windows' if os.name == 'nt' else sys.platform,
                   'executable_sha256': image, 'origin': 'actual-filesystem-obstruction', 'physical_exhaustion': False,
                   'cases': {'canonical-publication': case}, 'cleanup': True, 'fixture_removed': True, 'runtime_natural_exit': True}
        validate_receipt(receipt, image, receipt['platform'])
        output, error = owner.process.communicate(timeout=remaining(bounds.FIXTURE_IDLE_SECONDS))
        if owner.process.returncode != 0 or error or not owner.wait_for_descendants(remaining(bounds.CLEANUP_SECONDS)):
            raise ValueError('Storage fixture runtime did not exit naturally')
    receipt['fixture_removed'] = not directory.exists() and not private.exists()
    remaining()
    validate_receipt(receipt, image, receipt['platform'])
    return receipt
