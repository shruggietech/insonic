#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Bounded native qualification, verified archives and hidden child tooling."""
import argparse
from contextlib import contextmanager
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import urllib.request
import zipfile

sys.path.insert(0, str(Path(__file__).resolve().parent))
from process_tree import ProcessTree

ROOT = Path(__file__).resolve().parent.parent
LOCK = json.loads((ROOT / 'internal/qualification/dependencies.json').read_text(encoding='utf-8'))
VERSION = json.loads((ROOT / 'package.json').read_text(encoding='utf-8'))['version']
BUILD = ROOT / 'build/native'

def validate_pins(go_mod=None, lock=None):
    source = (ROOT / 'go.mod').read_text(encoding='utf-8') if go_mod is None else go_mod
    selected = LOCK if lock is None else lock
    expected = {'go': selected['go'], 'github.com/wailsapp/wails/v2': 'v' + selected['wails'],
                'github.com/LadybugDB/go-ladybug': selected['ladybug']['binding']}
    for dependency, version in expected.items():
        match = re.search(r'^\s*' + re.escape(dependency) + r'\s+(\S+)', source, re.MULTILINE)
        if not match or match.group(1) != version:
            raise ValueError('qualification dependency pin differs: ' + dependency)

def child(args, *, env=None, timeout=180, allow_detached=False):
    merged = os.environ.copy()
    if env:
        merged.update(env)
    with ProcessTree([str(arg) for arg in args], cwd=ROOT, env=merged, allow_detached=allow_detached) as tree:
        try:
            stdout, stderr = tree.process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            tree.kill()
            tree.process.communicate(timeout=5)
            raise
        result = tree.process.returncode
    if result:
        # Controlled compiler/test output is useful; no live/user credentials are supplied.
        sys.stdout.buffer.write(stdout)
        sys.stderr.buffer.write(stderr)
        raise RuntimeError('qualification child failed')
    return stdout

def archive_path(root, name):
    relative = PurePosixPath(name.replace('\\', '/'))
    if relative.is_absolute() or '..' in relative.parts or ':' in name:
        raise ValueError('archive path escapes destination')
    target = root.joinpath(*relative.parts)
    if not target.resolve().is_relative_to(root.resolve()):
        raise ValueError('archive path escapes destination')
    return target

def extract(archive, destination):
    destination.mkdir(parents=True, exist_ok=True)
    if zipfile.is_zipfile(archive):
        with zipfile.ZipFile(archive) as source:
            for entry in source.infolist():
                target = archive_path(destination, entry.filename)
                if entry.is_dir():
                    target.mkdir(parents=True, exist_ok=True)
                else:
                    target.parent.mkdir(parents=True, exist_ok=True)
                    with source.open(entry) as data, target.open('wb') as output:
                        shutil.copyfileobj(data, output)
    else:
        with tarfile.open(archive) as source:
            entries = source.getmembers()
            for entry in entries:
                target = archive_path(destination, entry.name)
                if entry.isdir():
                    target.mkdir(parents=True, exist_ok=True)
                elif entry.isfile():
                    target.parent.mkdir(parents=True, exist_ok=True)
                    with source.extractfile(entry) as data, target.open('wb') as output:
                        shutil.copyfileobj(data, output)
                    target.chmod(entry.mode & 0o755)
                elif not entry.issym() and not entry.islnk():
                    raise ValueError('unsupported archive entry')
            # Materialize verified loader aliases; never follow archive-created links.
            for entry in entries:
                if entry.issym() or entry.islnk():
                    target = archive_path(destination, entry.name)
                    link = str(PurePosixPath(entry.name).parent / entry.linkname) if entry.issym() else entry.linkname
                    original = archive_path(destination, link)
                    if not original.is_file():
                        # SONAME aliases may chain; resolve within the archive map.
                        by_name = {str(PurePosixPath(item.name)): item for item in entries}
                        seen = set()
                        while link in by_name and by_name[link].issym():
                            if link in seen:
                                raise ValueError('archive link cycle')
                            seen.add(link)
                            link = str(PurePosixPath(link).parent / by_name[link].linkname)
                        original = archive_path(destination, link)
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(original, target)

def prepare():
    os_name = {'Windows': 'windows', 'Linux': 'linux', 'Darwin': 'darwin'}[platform.system()]
    arch = {'AMD64': 'amd64', 'x86_64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}[platform.machine()]
    key = os_name + '_' + arch
    BUILD.mkdir(parents=True, exist_ok=True)
    receipt = {'platform': key, 'go': LOCK['go'], 'wails': LOCK['wails'], 'assets': []}
    for name, repo in [('ladybug', 'LadybugDB/ladybug'), ('cueson', 'shruggietech/cueson')]:
        filename, digest = LOCK[name]['assets'][key]
        archive = BUILD / filename
        if not archive.exists() or hashlib.sha256(archive.read_bytes()).hexdigest() != digest:
            url = f'https://github.com/{repo}/releases/download/v{LOCK[name]["version"]}/{filename}'
            request = urllib.request.Request(url, headers={'User-Agent': 'insonic-native-qualification'})
            with urllib.request.urlopen(request, timeout=120) as data, archive.open('wb') as output:
                shutil.copyfileobj(data, output)
        if hashlib.sha256(archive.read_bytes()).hexdigest() != digest:
            raise ValueError('upstream asset checksum mismatch')
        destination = BUILD / name
        extract(archive, destination)
        receipt['assets'].append({'name': filename, 'sha256': digest})
    library = next((BUILD / 'ladybug').rglob('lbug.h')).parent
    cueson = next((BUILD / 'cueson').rglob('cueson.exe' if os.name == 'nt' else 'cueson'))
    cueson.chmod(0o755)
    library_flags = library.as_posix()
    flags = f'-L{library_flags}'
    if os.name != 'nt':
        flags += f' -Wl,-rpath,{library_flags}'
    values = {'CGO_ENABLED': '1', 'CGO_CFLAGS': f'-I{library_flags}', 'CGO_LDFLAGS': flags,
              'CUESON_EXECUTABLE': str(cueson), 'INSONIC_NATIVE_LIBRARY': str(library)}
    if os_name == 'darwin':
        values['MACOSX_DEPLOYMENT_TARGET'] = '13.3'
    (BUILD / 'environment.json').write_text(json.dumps(values, indent=2) + '\n', encoding='utf-8')
    (BUILD / 'asset-receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(receipt))

def native_env():
    values = json.loads((BUILD / 'environment.json').read_text(encoding='utf-8'))
    values['PATH'] = values['INSONIC_NATIVE_LIBRARY'] + os.pathsep + os.environ['PATH']
    if os.name == 'nt' and Path('C:/msys64/ucrt64/bin').is_dir():
        values['PATH'] = 'C:/msys64/ucrt64/bin' + os.pathsep + values['PATH']
    return values

def cueson_probe(env):
    exe = env['CUESON_EXECUTABLE']
    version = child([exe, 'version']).decode().strip()
    if LOCK['cueson']['version'] != version:
        raise ValueError('wrong Cueson version')
    schema = child([exe, 'schema'])
    if hashlib.sha256(schema).hexdigest() != LOCK['cueson']['schema_sha256']:
        raise ValueError('Cueson schema identity mismatch')
    if schema != (ROOT / 'internal/subtitles/schema/cueson.schema.json').read_bytes():
        raise ValueError('packaged Cueson schema differs from runtime schema')
    with tempfile.TemporaryDirectory() as directory:
        directory = Path(directory)
        fixture = directory / 'source.srt'
        data = b'1\n00:00:00,000 --> 00:00:01,000\nQualification source.\n\n'
        fixture.write_bytes(data)
        document = directory / 'document.cueson'
        child([exe, 'encode', '--output', document, fixture])
        source = json.loads(document.read_text(encoding='utf-8'))
        if source['schema_version'] != '1.2.0':
            raise ValueError('Cueson encoded wrong current identity')
        source['media_timing'] = {'duration_milliseconds': 1000}
        source['cues'][0]['speaker_attributions'] = [
            {'speaker_id': '11111111-1111-4111-8111-111111111111', 'start_milliseconds': 100, 'end_milliseconds': 400},
            {'speaker_id': '22222222-2222-4222-8222-222222222222', 'start_milliseconds': 200, 'end_milliseconds': 600},
            {'speaker_id': '11111111-1111-4111-8111-111111111111'},
        ]
        document.write_text(json.dumps(source, separators=(',', ':')) + '\n', encoding='utf-8', newline='\n')
        child([exe, 'validate', '--format', 'cueson', document])
        report = json.loads(child([exe, 'inspect', '--json', document]))
        expected = {'attribution_count': 3, 'timed_attribution_count': 2, 'untimed_attribution_count': 1,
                    'media_timing_present': True, 'media_boundary_check': 'checked', 'cue_media_conflict_count': 0}
        if report.get('consumer_annotations') != expected:
            raise ValueError('Cueson consumer annotation inspection differs')
        child([exe, 'render', '--to', 'srt', '--output', directory / 'rendered.srt', document])
        restored = directory / 'restored.srt'
        child([exe, 'restore', '--no-metadata', '--output', restored, document])
        if restored.read_bytes() != data:
            raise ValueError('subtitle source restoration differs')
        strict = directory / 'strict.srt'
        with ProcessTree([str(arg) for arg in [exe, 'render', '--strict', '--to', 'srt', '--output', strict, document]], cwd=ROOT, env=os.environ.copy()) as tree:
            try:
                stdout, stderr = tree.process.communicate(timeout=30)
            except subprocess.TimeoutExpired:
                tree.kill()
                tree.process.communicate(timeout=5)
                raise
            if tree.process.returncode != 1 or strict.exists() or stdout or b'strict' not in stderr:
                raise ValueError('strict annotated export did not refuse before publication')
    return {'version': version, 'schema_sha256': LOCK['cueson']['schema_sha256'],
            'source_roundtrip': 'passed', 'consumer_overlap_untimed': 'passed', 'strict_loss_refusal': 'passed'}

def native():
    for script in ["test-media-tools.py", "test-media-fixtures.py", "test-processing-worker.py"]:
        sys.stdout.buffer.write(child([sys.executable, ROOT / "scripts" / script]))
    sys.stdout.buffer.write(child([sys.executable, ROOT / "scripts/media-fixtures.py", "--native"]))
    env = native_env()
    cueson = cueson_probe(env)
    output = child(['go', 'test', '-v', '-tags', 'native_ladybug,system_ladybug', './internal/qualification', './internal/subtitles'], env=env)
    sys.stdout.buffer.write(output)
    (BUILD / 'native-receipt.json').write_text(json.dumps({'cueson': cueson, 'native_pair': 'passed'}, indent=2) + '\n', encoding='utf-8')

def assets():
    help_dir = ROOT / 'site/offline'
    if not (help_dir / 'index.html').is_file():
        raise ValueError('build offline help before desktop assets')
    target = ROOT / 'desktop/assets'
    target.mkdir(parents=True, exist_ok=True)
    shutil.copytree(help_dir, target / 'help', dirs_exist_ok=True)
    shutil.copyfile(ROOT / 'brand/kit/tokens/interface.css', target / 'interface.css')
    shutil.copyfile(ROOT / 'brand/kit/tokens/typography.css', target / 'typography.css')
    (target / 'fonts').mkdir(exist_ok=True)
    for font in ['Geist-Regular.woff2', 'SpaceGrotesk-Medium.woff2', 'GeistMono-Regular.woff2']:
        shutil.copyfile(ROOT / 'brand/kit/fonts/woff2' / font, target / 'fonts' / font)

def write_receipt(path, output, expected):
    # Native loaders can print warnings alongside the compact qualification result.
    values = []
    for line in output.splitlines():
        try:
            values.append(json.loads(line))
        except (json.JSONDecodeError, UnicodeDecodeError):
            continue
    if len(values) != 1 or not isinstance(values[0], dict) or any(values[0].get(key) != value for key, value in expected.items()):
        raise ValueError('missing, ambiguous or failed qualification receipt')
    path.write_text(json.dumps(values[0], sort_keys=True) + '\n', encoding='utf-8')


def desktop():
    env = native_env()
    tags = 'desktop,production'
    if platform.system() == 'Linux':
        tags += ',webkit2_41'
    output = child(['go', 'test', '-v', '-tags', tags, './desktop', './cmd/insonic-desktop'], env=env)
    sys.stdout.buffer.write(output)
    executable = ROOT / ('build/insonic-desktop.exe' if os.name == 'nt' else 'build/insonic-desktop')
    child(['go', 'build', '-tags', tags, '-o', executable, './cmd/insonic-desktop'], env=env)
    output = child([executable, '--qualification'], env=env)
    write_receipt(BUILD / 'desktop-receipt.json', output,
                  {'desktop_bridge': 'passed', 'offline_help': 'packaged', 'schema_version': VERSION})
    args = [executable, '--webview-qualification']
    if platform.system() == 'Linux':
        args = ['xvfb-run', '-a', *args]
    output = child(args, env=env, timeout=90)
    write_receipt(BUILD / 'webview-receipt.json', output,
                  {'frontend_bridge_ipc': 'passed', 'native_webview': 'passed', 'schema_version': VERSION})

def secrets():
    BUILD.mkdir(parents=True, exist_ok=True)
    executable = ROOT / ('build/insonic-secret-lifecycle.exe' if os.name == 'nt' else 'build/insonic-secret-lifecycle')
    child(['go', 'build', '-o', executable, './cmd/insonic-secret-lifecycle'])
    output = child([executable], timeout=30)
    write_receipt(BUILD / 'secret-service-receipt.json', output,
                  {'native_credential_lifecycle': 'passed', 'cross_process_restart': 'passed', 'credential_values': 'not-returned', 'schema_version': VERSION})
    child(['go', 'test', './internal/secrets', './internal/credentialcmd'])
    sys.stdout.buffer.write(output)

@contextmanager
def cli_workspace():
    with tempfile.TemporaryDirectory() as directory:
        try:
            yield directory
        except BaseException:
            # Detached owner idle exit releases Windows scratch/catalog handles.
            # Preserve the original failure instead of masking it with cleanup.
            import time
            time.sleep(31)
            raise

def cli():
    executable = ROOT / ('build/insonic.exe' if os.name == 'nt' else 'build/insonic')
    child(['go', 'build', '-o', executable, './cmd/insonic'])
    with cli_workspace() as directory:
        child([executable, 'workspace', 'init', directory, '--json'])
        # General CLI fixtures elect transient credentials explicitly. Native
        # persistence is qualified separately against an isolated OS store.
        child([executable, '--workspace', directory, 'credentials', 'select', 'session', '--json'])
        first = json.loads(child([executable, '--workspace', directory, 'workspace', 'show', '--json'], allow_detached=True))
        second = json.loads(child([executable, '--workspace', directory, 'workspace', 'show', '--json']))
        if first['runtime_session_id'] != second['runtime_session_id']:
            raise ValueError('CLI clients reached different owners')
        # Saved configuration and context run through the actual shared owner.
        # Synthetic managed IDs are inspected, never downloaded or executed.
        pipeline_id = '10000000-0000-4000-8000-000000000010'
        speaker_id = '10000000-0000-4000-8000-000000000011'
        alias_id = '10000000-0000-4000-8000-000000000012'
        term_id = '10000000-0000-4000-8000-000000000013'
        config_input = Path(directory) / 'configuration input.json'
        def configuration_call(family, operation, item=None, data=None):
            args = [executable, '--workspace', directory, family, operation]
            if item is not None:
                args.append(item)
            if data is not None:
                config_input.write_text(json.dumps(data) + '\n', encoding='utf-8')
                args.extend(['--input', config_input])
            return json.loads(child([*args, '--json']))['result']
        definition = {
            'recognition': {'adapter': 'faster-whisper', 'contract_version': '1', 'mode': 'local',
                            'model_id': '10000000-0000-4000-8000-000000000014'},
            'diarization': {'adapter': 'pyannote', 'contract_version': '1', 'mode': 'local',
                           'model_id': '10000000-0000-4000-8000-000000000015'},
        }
        saved = configuration_call('pipelines', 'set', pipeline_id, {
            'expected_revision': 0, 'pipeline': {'id': pipeline_id, 'name': 'Local qualification',
                                                'preset': 'local', 'configuration': definition}})
        inspected = configuration_call('pipelines', 'inspect', pipeline_id)
        if saved['revision'] < 1 or inspected['revision'] != saved['revision'] or inspected['reachability'] != 'not-probed':
            raise ValueError('saved pipeline revision or inspection changed')
        identity = configuration_call('speakers', 'set', speaker_id, {
            'expected_revision': 0, 'speaker': {'id': speaker_id, 'name': 'Example Speaker'},
            'aliases': [{'id': alias_id, 'speaker_id': speaker_id, 'text': 'Example S.'}]})
        if identity['speaker']['revision'] < 1:
            raise ValueError('speaker expected-revision creation failed')
        aliases = configuration_call('speakers', 'aliases', speaker_id)
        if len(aliases['aliases']) != 1 or aliases['aliases'][0]['text'] != 'Example S.':
            raise ValueError('default alias inspection changed')
        configuration_call('terms', 'set', term_id, {'expected_revision': 0,
            'term': {'id': term_id, 'canonical': 'Deterministic terminology'}})
        compiled = configuration_call('terms', 'compile', data={'pipeline_id': pipeline_id,
            'pipeline_revision': saved['revision'], 'speaker_ids': [speaker_id], 'max_hint_bytes': 200})
        if not {'Example Speaker', 'Example S.', 'Deterministic terminology'}.issubset(set(compiled['hints'])):
            raise ValueError('explicit speaker/term hint compilation differs')
        default_context = configuration_call('terms', 'compile')
        if 'Deterministic terminology' not in default_context['hints'] or 'Example Speaker' in default_context['hints']:
            raise ValueError('default context silently elected speaker identities')
        for family, item in [('pipelines', pipeline_id), ('speakers', speaker_id), ('terms', term_id)]:
            if len(configuration_call(family, 'list')['items']) != 1:
                raise ValueError('configuration list failed')
            configuration_call(family, 'show', item)
        selection = configuration_call('speakers', 'select', speaker_id)
        if selection['selection']['items']:
            raise ValueError('selection invented source evidence')
        source = Path(directory) / 'artifact source.bin'
        content = b'CLI artifact\x00\xff\n'
        source.write_bytes(content)
        artifact_args = [executable, '--workspace', directory, '--request-id',
                         '10000000-0000-4000-8000-000000000003', 'artifacts', 'publish', source, 'other', '--json']
        publication = json.loads(child(artifact_args))['result']
        replay = json.loads(child(artifact_args))['result']
        if publication != replay or publication['state'] != 'available' or publication['digest'] != hashlib.sha256(content).hexdigest():
            raise ValueError('artifact publication/replay differs')
        artifact_id = publication['id']
        def artifact_call(operation, *args):
            return json.loads(child([executable, '--workspace', directory, 'artifacts', operation, artifact_id, *args, '--json']))['result']
        if artifact_call('show')['id'] != artifact_id or not artifact_call('verify')['verified']:
            raise ValueError('artifact inspection/verification')
        materialized = artifact_call('materialize', str(len(content)))
        if Path(materialized['path']).read_bytes() != content:
            raise ValueError('artifact materialization differs')
        lease_id = materialized['lease']['id']
        artifact_call('lease-renew', lease_id)
        reference_id = '10000000-0000-4000-8000-000000000004'
        artifact_call('retain', reference_id)
        if artifact_call('show')['reference_count'] != 1:
            raise ValueError('artifact retention')
        artifact_call('release-reference', reference_id)
        artifact_call('lease-release', lease_id)
        if Path(materialized['path']).exists() or artifact_call('cache-prune') != []:
            raise ValueError('artifact cache release')
        request_id = '10000000-0000-4000-8000-000000000002'
        start_args = [executable, '--workspace', directory, '--request-id', request_id, 'jobs', 'start', '1000', '--json']
        started = json.loads(child(start_args))
        replayed = json.loads(child(start_args))
        if replayed['result']['attempt_id'] != started['result']['attempt_id']:
            raise ValueError('lost-response job acceptance reconciliation')
        job = started['result']['job_id']
        child([executable, '--workspace', directory, 'jobs', 'cancel', job, '--json'])
        retried = json.loads(child([executable, '--workspace', directory, 'jobs', 'retry', job, '--json']))
        if retried['result']['claim_generation'] != 2:
            raise ValueError('retry generation')
        child([executable, '--workspace', directory, 'jobs', 'cancel', job, '--json'])
        # Owner remains alive until idle exit; wait before removing its private files.
        history = json.loads(child([executable, '--workspace', directory, 'jobs', 'history', job, '--json']))
        if len(history['result']['attempts']) != 2 or any(attempt['state'] != 'cancelled' for attempt in history['result']['attempts']):
            raise ValueError('durable cancellation/retry history')
        import time
        time.sleep(31)
        snapshot = Path(directory) / 'catalog-export.json'
        child([executable, '--workspace', directory, 'catalog', 'export', snapshot, '--json'])
        exported = json.loads(snapshot.read_text(encoding='utf-8'))
        attempts = next(table['rows'] for table in exported['state'] if table['name'] == 'job_attempt')
        if len(attempts) != 2 or any(attempt[4] != 'cancelled' for attempt in attempts):
            raise ValueError('catalog history did not survive owner exit')
        config_file = Path(directory) / '.insonic' / 'workspace.json'
        config = json.loads(config_file.read_text(encoding='utf-8'))
        config['profiles']['catalog']['configuration']['path'] = 'restored.sqlite'
        config['profiles']['catalog']['profile_revision'] += 1
        config_file.write_text(json.dumps(config) + '\n', encoding='utf-8')
        child([executable, '--workspace', directory, 'catalog', 'restore', snapshot, '--json'])
        roundtrip = Path(directory) / 'catalog-roundtrip.json'
        child([executable, '--workspace', directory, 'catalog', 'export', roundtrip, '--json'])
        restored = json.loads(roundtrip.read_text(encoding='utf-8'))
        restored_attempts = next(table['rows'] for table in restored['state'] if table['name'] == 'job_attempt')
        if restored_attempts != attempts or restored['revision'] != exported['revision'] + 1:
            raise ValueError('CLI restore did not preserve durable history and restoration receipt')
    (BUILD / 'cli-receipt.json').write_text(json.dumps({'owner_reuse': 'passed', 'cancel_retry': 'passed', 'durable_history': 'passed', 'catalog_transfer': 'passed', 'artifact_journey': 'passed', 'saved_configuration': 'passed', 'scoped_hints': 'passed', 'inference': 'not-run'}) + '\n', encoding='utf-8')

if __name__ == '__main__':
    validate_pins()
    parser = argparse.ArgumentParser()
    parser.add_argument('stage', choices=['prepare', 'native', 'assets', 'desktop', 'cli', 'secrets'])
    options = parser.parse_args()
    globals()[options.stage]()
