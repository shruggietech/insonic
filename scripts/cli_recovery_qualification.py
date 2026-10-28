# SPDX-License-Identifier: Apache-2.0
"""Actual owned-runtime import interruption, with deterministic external barriers."""
from contextlib import ExitStack
import copy
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import sys
import tempfile
import threading
import time
import uuid

from file_open_barrier import FileOpenBarrier
from process_tree import ProcessTree
from cli_media_qualification import pcm_fixture
from cli_terminal_qualification import owned_directory, owned_runtime
import qualification_timing as bounds

CASES = ('same-source', 'changed-source', 'corrupt-report')
COMMON = ('prior_attempt_retained', 'unrelated_unchanged', 'admission_count_correct', 'cleanup')
CHECKS = {name: COMMON + extra for name, extra in {
    'same-source': ('raw_metadata_preserved', 'date_provenance_preserved', 'canonical_bytes_preserved',
                    'report_bytes_preserved', 'idempotent_replay'),
    'changed-source': ('stale_admission_refused', 'candidate_bytes_preserved'),
    'corrupt-report': ('stale_admission_refused', 'integrity_failure_observed')}.items()}


def sha(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()).hexdigest()


def derived(work, label):
    value = bytearray(hashlib.sha256((work + '\0' + label).encode()).digest()[:16])
    value[6], value[8] = value[6] & 15 | 64, value[8] & 63 | 128
    return str(uuid.UUID(bytes=bytes(value)))


def identifier(value):
    try:
        return type(value) is str and str(uuid.UUID(value)) == value
    except (ValueError, TypeError, AttributeError):
        return False


def hashed(value):
    return type(value) is str and re.fullmatch('[a-f0-9]{64}', value) is not None


def entered_date_preserved(dates):
    selected = dates.get('selected') or {}
    expected = {'literal': '2020-01-02T03:04:05.123456789Z', 'basis': 'owner', 'precision': 'instant',
                'zone': 'UTC', 'zone_source': 'entered-offset', 'offset_seconds': 0,
                'interpretation_state': 'resolved', 'assumptions': [],
                'resolved': {'iso': '2020-01-02T03:04:05.123456789Z', 'unix_ns': 1577934245123456789}}
    return (all(selected.get(key) == value for key, value in expected.items())
            and bool(selected.get('rules_version')) and bool(selected.get('mapping_version')))


def validate_receipt(receipt, executable_sha256, platform):
    fields = {'format', 'platform', 'executable_sha256', 'cases', 'cleanup', 'fixture_removed',
              'runtime_natural_exit', 'barriers_released'}
    if (not isinstance(receipt, dict) or set(receipt) != fields
            or receipt['format'] != 'insonic-import-recovery-qualification-v1'
            or platform not in ('windows', 'linux') or receipt['platform'] != platform
            or not hashed(executable_sha256) or receipt['executable_sha256'] != executable_sha256
            or any(receipt[k] is not True for k in fields - {'format', 'platform', 'executable_sha256', 'cases'})
            or not isinstance(receipt['cases'], dict) or set(receipt['cases']) != set(CASES)):
        raise ValueError('Incomplete import recovery receipt')
    works = set()
    for name, case in receipt['cases'].items():
        if (not isinstance(case, dict) or set(case) != {'work_id', 'source_sha256', 'source_size', 'changed_sha256', 'before', 'after', 'termination', 'checks'}
                or not identifier(case['work_id']) or case['work_id'] in works or not hashed(case['source_sha256'])
                or type(case['source_size']) is not int or case['source_size'] <= 0):
            raise ValueError('Invalid recovery scenario identity')
        works.add(case['work_id'])
        before, after, end = case['before'], case['after'], case['termination']
        if (not isinstance(before, dict) or set(before) != {'pid', 'session', 'generation', 'journal_receipt_id', 'canonical', 'report', 'admission_absent', 'barrier', 'ack_required'}
                or not isinstance(after, dict) or set(after) != {'pid', 'session', 'generation', 'journal_receipt_id', 'state', 'error', 'entry_id', 'metadata_sha256', 'report_sha256'}
                or any(type(v['pid']) is not int or v['pid'] <= 0 or not identifier(v['session'])
                       or not identifier(v['journal_receipt_id']) or type(v['generation']) is not int or v['generation'] <= 0 for v in (before, after))
                or after['session'] == before['session'] or after['generation'] <= before['generation']
                or after['journal_receipt_id'] == before['journal_receipt_id'] or before['admission_absent'] is not True
                or before['ack_required'] is not True or before['barrier'] != {'windows': 'windows-rw-oplock', 'linux': 'linux-write-lease'}[platform]
                or not hashed(after['metadata_sha256']) or not hashed(after['report_sha256'])):
            raise ValueError('Recovery runtime/boundary authority differs')
        for kind in ('canonical', 'report'):
            pub = before[kind]
            if (not isinstance(pub, dict) or set(pub) != {'id', 'digest', 'size'} or not identifier(pub['id'])
                    or not hashed(pub['digest']) or type(pub['size']) is not int or pub['size'] <= 0):
                raise ValueError('Recovery publication identity differs')
            label = 'canonical-0' if kind == 'canonical' else 'canonical-report-0'
            if pub['id'] != derived(case['work_id'], label):
                raise ValueError('Recovery publication is not bound to its work')
        if (not isinstance(end, dict) or set(end) != {'exit_code', 'process_dead', 'within_barrier'}
                or type(end['exit_code']) is not int or end['exit_code'] == 0
                or end['process_dead'] is not True or end['within_barrier'] is not True
                or not isinstance(case['checks'], dict) or set(case['checks']) != set(CHECKS[name])
                or any(value is not True for value in case['checks'].values())):
            raise ValueError('Recovery termination/outcome checks incomplete')
        expected = ('succeeded', '') if name == 'same-source' else ('failed', 'conflict' if name == 'changed-source' else 'checksum_mismatch')
        if ((after['state'], after['error']) != expected
                or (not identifier(after['entry_id']) if name == 'same-source' else after['entry_id'] != '')
                or (not hashed(case['changed_sha256']) or case['changed_sha256'] == case['source_sha256'] if name == 'changed-source' else case['changed_sha256'] is not None)
                or (after['report_sha256'] == before['report']['digest'] if name == 'corrupt-report' else after['report_sha256'] != before['report']['digest'])):
            raise ValueError('Recovery admission/refusal result differs')


def media_tools(executable, source_tools=None):
    root = Path(executable).parent
    manifest = root / 'insonic-companions.json'
    if manifest.exists():
        tools = json.loads(manifest.read_text(encoding='utf-8'))['media']
        for tool in tools.values():
            if not isinstance(tool, dict):
                continue
            if 'path' in tool:
                tool['path'] = str((root / tool['path']).resolve(strict=True))
            for pin in tool.get('support_files', []):
                pin['path'] = str((root / pin['path']).resolve(strict=True))
            if tool.get('interpreter'):
                tool['interpreter']['path'] = str((root / tool['interpreter']['path']).resolve(strict=True))
        return tools
    selected = Path(source_tools) if source_tools else Path(__file__).resolve().parents[1] / 'build/native/media-tools.json'
    return json.loads(selected.read_text(encoding='utf-8'))


def fixture_tools(tools, directory, deadline):
    selected = copy.deepcopy(tools)
    python_pin = {'path': sys.executable, 'sha256': sha(sys.executable)}
    probe = directory / 'controlled ffprobe.py'
    original = selected['ffprobe']['path']
    original_digest = selected['ffprobe']['sha256']
    if sha(original) != original_digest:
        raise ValueError('Recovery probe differs from pinned selected companion')
    # Real media bytes and canonical encoding remain the normal selected tools.
    # Only the configured probe's final return is held while the external file
    # barrier is armed. The generated script is hash-pinned like any tool.
    probe.write_text('''import hashlib, json, os, pathlib, subprocess, sys, time
root = pathlib.Path(__file__).parent
args = sys.argv[1:]
with open(ORIGINAL, 'rb') as image:
    if hashlib.file_digest(image, 'sha256').hexdigest() != PINNED:
        raise ValueError('Selected recovery probe image changed')
result = subprocess.run([ORIGINAL, *args], stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)
if '-show_error' not in args and '-version' not in args and (root/'gate-enabled').exists():
    (root/'probe-ready').write_text(json.dumps({'pid':os.getpid()}), encoding='utf-8')
    end = min(time.time()+20, DEADLINE)
    while not (root/'probe-release').exists():
        if time.time() >= end: raise TimeoutError('Owned canonical probe fixture gate expired')
        time.sleep(.01)
sys.stdout.buffer.write(result.stdout)
sys.stderr.buffer.write(result.stderr)
sys.exit(result.returncode)
'''.replace('ORIGINAL', repr(original)).replace('PINNED', repr(original_digest)).replace('DEADLINE', repr(time.time() + deadline - time.monotonic())), encoding='utf-8', newline='\n')
    exif = directory / 'captured metadata.py'
    exif.write_text('''import json, pathlib, sys
if '-ver' in sys.argv:
    print('fixture-1')
else:
    path=pathlib.Path(__file__).with_suffix('.count')
    count=int(path.read_text())+1 if path.exists() else 1
    path.write_text(str(count))
    print(json.dumps([{'EXIF:DateTimeOriginal':'2001:02:03 04:05:06', 'File:MIMEType':'audio/wav', 'Fixture:CaptureNonce':count}]))
''', encoding='utf-8', newline='\n')
    selected['ffprobe'] = {'path': str(probe), 'sha256': sha(probe), 'version': selected['ffprobe']['version'], 'interpreter': python_pin}
    selected['exiftool'] = {'path': str(exif), 'sha256': sha(exif), 'version': 'fixture-1', 'interpreter': python_pin}
    return selected


def qualify(invoke, workspace, version, *, executable, environment, process_factory=None, source_tools=None):
    deadline = time.monotonic() + bounds.phase_bound(bounds.SUBPROCESS_SECONDS)
    executable = Path(executable).resolve(strict=True)
    image_digest = sha(executable)
    factory = process_factory or ProcessTree

    def remaining(limit=bounds.SUBPROCESS_SECONDS):
        value = min(limit, deadline - time.monotonic(), bounds.phase_bound(limit))
        if value <= 0:
            raise TimeoutError('Import recovery enclosing deadline exceeded')
        return value

    cases, barriers = {}, []
    with ExitStack() as resources:
        directory = Path(workspace) / 'import recovery café'
        directory.mkdir()
        # Existing owned_directory supplies the no-link/no-replacement cleanup
        # boundary. Its fixed child is used under this unique campaign parent.
        resources.callback(directory.rmdir)
        root = resources.enter_context(owned_directory(directory))
        private = Path(resources.enter_context(tempfile.TemporaryDirectory(prefix='ir-', dir='/tmp' if sys.platform == 'linux' else None)))
        selected = dict(environment)
        for name in ['HOME', 'USERPROFILE', 'APPDATA', 'LOCALAPPDATA', 'XDG_RUNTIME_DIR', 'XDG_CONFIG_HOME', 'XDG_DATA_HOME', 'XDG_CACHE_HOME']:
            selected[name] = str(private)
        tools = media_tools(executable, source_tools)
        for name in CASES:
            case_dir = root / name
            case_dir.mkdir()
            library = case_dir / 'owned library'
            active_barrier = None

            def call(args, **options):
                allowance = min(remaining(), active_barrier.remaining()) if active_barrier is not None else remaining()
                options['timeout_seconds'] = min(options.get('timeout_seconds', allowance), allowance)
                return invoke(args, workspace_path=library, execution_environment=selected, **options)

            call(['workspace', 'init', library], scoped=False)
            call(['credentials', 'select', 'session'])
            configured = fixture_tools(tools, case_dir, deadline)
            (library / '.insonic/media-tools.json').write_text(json.dumps(configured) + '\n', encoding='utf-8', newline='\n')

            def start_owner(stack):
                startup_id = str(uuid.uuid4())
                owner = stack.enter_context(owned_runtime([str(executable), '--workspace', str(library), '--startup-id', startup_id,
                                                          '--json', 'runtime', 'serve'], cwd=case_dir, env=selected,
                                                         gates={}, factory=factory))
                mailbox = queue.Queue(maxsize=1)
                reader = threading.Thread(target=lambda: mailbox.put(owner.process.stdout.readline(4097)), daemon=True)
                reader.start()
                try:
                    report = json.loads(mailbox.get(timeout=remaining(bounds.READINESS_SECONDS)))
                except BaseException:
                    owner.kill()
                    raise
                finally:
                    reader.join(bounds.CLEANUP_SECONDS)
                    if reader.is_alive():
                        raise RuntimeError('Recovery startup reader did not release')
                session = call(['workspace', 'show'])
                if (report.get('startup_id') != startup_id or report.get('state') != 'ready'
                        or report.get('workspace_id') != session.get('result', {}).get('workspace_id') or not session.get('runtime_session_id')):
                    raise ValueError('Recovery owned runtime readiness differs: ' + json.dumps({'report': report, 'session': session})[:2500])
                return owner, session['runtime_session_id']

            def snapshot():
                target = case_dir / ('snapshot-' + str(uuid.uuid4()) + '.json')
                call(['catalog', 'export', target])
                return json.loads(target.read_text(encoding='utf-8'))

            def publication(kind, work_id):
                value = call(['artifacts', 'show', derived(work_id, 'canonical-0' if kind == 'canonical' else 'canonical-report-0')])['result']
                if value['state'] != 'available':
                    raise ValueError('Recovery boundary publication is not available')
                return {k: value[k] for k in ('id', 'digest', 'size')}

            def read_report(pub):
                value = call(['artifacts', 'materialize', pub['id'], str(pub['size'])])['result']
                try:
                    raw = Path(value['path']).read_bytes()
                    if hashlib.sha256(raw).hexdigest() != pub['digest']:
                        raise ValueError('Independently read report bytes differ')
                    return json.loads(raw)
                finally:
                    call(['artifacts', 'lease-release', pub['id'], value['lease']['id']])

            source, baseline = case_dir / 'original source résumé.wav', case_dir / 'unrelated.wav'
            pcm_fixture(source, 1)
            pcm_fixture(baseline, 2)
            source_digest, source_size = sha(source), source.stat().st_size
            with ExitStack() as old_resources:
                owner, old_session = start_owner(old_resources)
                initial = call(['media', 'import', baseline, '--new-entry', '--no-pipeline'])['result']['work_id']
                if call(['work', 'wait', initial])['result']['state'] != 'succeeded':
                    raise ValueError('Unrelated accepted fixture failed')
                base_records = snapshot()['records']['library']
                if len(base_records) != 1:
                    raise ValueError('Unrelated accepted fixture count differs')
                base_digest = digest(base_records[0])
                request = str(uuid.uuid4())
                (case_dir / 'gate-enabled').write_bytes(b'owned fixture\n')
                args = ['media', 'import', source, '--new-entry', '--no-pipeline', '--originated-at', '2020-01-02T03:04:05.123456789Z']
                work_id = call(args, request_id=request)['result']['work_id']
                probe_end = time.monotonic() + remaining(bounds.READINESS_SECONDS)
                while not (case_dir / 'probe-ready').exists():
                    if owner.process.poll() is not None or time.monotonic() >= probe_end:
                        raise ValueError('Owned import did not reach canonical probe gate')
                    time.sleep(.01)
                barrier = old_resources.enter_context(FileOpenBarrier(source, cwd=case_dir, env=selected,
                                                                      deadline=deadline, process_factory=factory))
                def stop_blocked_owner():
                    if owner.process.poll() is None:
                        owner.kill()
                        owner.process.communicate(timeout=bounds.CLEANUP_SECONDS)
                        if not owner.wait_for_descendants(bounds.CLEANUP_SECONDS):
                            raise RuntimeError('Interrupted fixture runtime descendants remain')
                old_resources.callback(stop_blocked_owner)
                barriers.append(barrier)
                (case_dir / 'probe-release').write_bytes(b'continue\n')
                barrier.wait_break()
                active_barrier = barrier
                # All commands below are capped by the still-held kernel barrier.
                def held(args, **options):
                    return call(args, timeout_seconds=min(remaining(), barrier.remaining()), **options)
                held_work = held(['work', 'show', work_id])['result']
                canonical, report = publication('canonical', work_id), publication('report', work_id)
                bundle = read_report(report)
                before_snapshot = snapshot()
                before_records = before_snapshot['records']['library']
                before_work = next(w for w in before_snapshot['records']['works'] if w['id'] == work_id)
                if held_work['id'] != before_work['id'] or held_work['state'] != before_work['state']:
                    raise ValueError('Public and exported held work authority differ')
                if len(before_records) != 1 or digest(before_records[0]) != base_digest:
                    raise ValueError('Import admitted before the observed interruption boundary')
                if bundle['source_digest'] != source_digest or bundle['source_size'] != source_size or bundle['canonical_publication'] != canonical:
                    # The bundle uses the same id/digest/size identity keys.
                    raise ValueError('Published capture does not bind fixture source/canonical bytes')
                if before_work['owner'] != old_session or before_work['state'] != 'running':
                    raise ValueError('Held import belongs to a different runtime attempt')
                barrier.remaining()
                owner.kill()
                owner.process.communicate(timeout=min(remaining(bounds.CLEANUP_SECONDS), barrier.remaining()))
                if owner.process.returncode in (None, 0) or not owner.wait_for_descendants(min(remaining(bounds.CLEANUP_SECONDS), barrier.remaining())):
                    raise ValueError('Selected runtime was not abruptly terminated and reaped')
                barrier.remaining()
                termination = {'exit_code': owner.process.returncode, 'process_dead': True, 'within_barrier': True}
                old_pid = owner.process.pid
                barrier.release()
                active_barrier = None
            (case_dir / 'gate-enabled').unlink()
            changed = None
            report_digest = report['digest']
            if name == 'changed-source':
                data = bytearray(source.read_bytes())
                data[-1] ^= 1
                source.write_bytes(data)
                changed = sha(source)
            if name == 'corrupt-report':
                # Mutate only this fixture's exact managed object after the old
                # owner is dead; the catalog digest remains authoritative.
                pub = next(json.loads(row[2]) for table in before_snapshot['state'] if table['name'] == 'artifact_publication'
                           for row in table['rows'] if row[0] == report['id'])
                target = library / '.insonic/artifacts' / pub['key']
                data = bytearray(target.read_bytes())
                data[-1] ^= 1
                target.write_bytes(data)
                report_digest = sha(target)
            with ExitStack() as new_resources:
                replacement, new_session = start_owner(new_resources)
                waited = call(['work', 'wait', work_id], expected=0 if name == 'same-source' else 1)['result']
                observed_work = call(['work', 'show', work_id])['result']
                after_snapshot = snapshot()
                after_work = next(w for w in after_snapshot['records']['works'] if w['id'] == work_id)
                if after_work['owner'] != new_session or observed_work['state'] != after_work['state']:
                    raise ValueError('Recovered import belongs to a different replacement runtime')
                records = after_snapshot['records']['library'] or []
                kept = [entry for entry in records if entry['id'] == base_records[0]['id']]
                checks = {'unrelated_unchanged': len(kept) == 1 and digest(kept[0]) == base_digest,
                          'admission_count_correct': len(records) == (2 if name == 'same-source' else 1), 'cleanup': True}
                rows = next(t['rows'] for t in after_snapshot['state'] if t['name'] == 'operation_receipt')
                old_receipt = next((json.loads(row[3]) for row in rows if row[0] == before_work['journal_receipt_id']), None)
                checks['prior_attempt_retained'] = bool(old_receipt and old_receipt.get('work_id') == work_id and old_receipt.get('generation') == before_work['generation'])
                entry_id = ''
                if name == 'same-source':
                    entry = next(e for e in records if e['id'] != kept[0]['id'])
                    entry_id = entry['id']
                    checks.update(raw_metadata_preserved=entry['metadata'] == bundle['metadata'],
                                  date_provenance_preserved=entered_date_preserved(entry['dates']),
                                  canonical_bytes_preserved=entry['original_publication_id'] == canonical['id'] and entry['digest'] == canonical['digest'] and held_verify(call, canonical),
                                  report_bytes_preserved=entry['report_publication_ids'] == [report['id']] and held_verify(call, report))
                    replay = call(args, request_id=request)['result']['work_id']
                    checks['idempotent_replay'] = replay == work_id and len(snapshot()['records']['library']) == 2
                else:
                    checks['stale_admission_refused'] = len(records) == 1 and after_work['state'] == 'failed'
                    if name == 'changed-source':
                        checks['candidate_bytes_preserved'] = held_verify(call, canonical) and held_verify(call, report)
                    else:
                        checks['integrity_failure_observed'] = after_work['error'] == 'checksum_mismatch'
                cases[name] = {'work_id': work_id, 'source_sha256': source_digest, 'source_size': source_size, 'changed_sha256': changed,
                    'before': {'pid': old_pid, 'session': old_session, 'generation': before_work['generation'],
                               'journal_receipt_id': before_work['journal_receipt_id'], 'canonical': canonical, 'report': report,
                               'admission_absent': True, 'barrier': barrier.mechanism, 'ack_required': True},
                    'termination': termination,
                    'after': {'pid': replacement.process.pid, 'session': new_session, 'generation': after_work['generation'],
                              'journal_receipt_id': after_work['journal_receipt_id'], 'state': after_work['state'], 'error': after_work['error'],
                              'entry_id': entry_id, 'metadata_sha256': digest(bundle['metadata']), 'report_sha256': report_digest}, 'checks': checks}
                output, error = replacement.process.communicate(timeout=remaining(bounds.FIXTURE_IDLE_SECONDS))
                if replacement.process.returncode != 0 or error or not replacement.wait_for_descendants(remaining(bounds.CLEANUP_SECONDS)):
                    raise ValueError('Replacement runtime did not exit naturally')
        receipt = {'format': 'insonic-import-recovery-qualification-v1', 'platform': 'windows' if os.name == 'nt' else sys.platform,
                   'executable_sha256': image_digest, 'cases': cases, 'cleanup': True, 'fixture_removed': False,
                   'runtime_natural_exit': True, 'barriers_released': all(b.released for b in barriers)}
    receipt['fixture_removed'] = not directory.exists() and not private.exists()
    remaining()
    validate_receipt(receipt, image_digest, receipt['platform'])
    return receipt


def held_verify(call, publication):
    return call(['artifacts', 'verify', publication['id']])['result']['verified'] is True


