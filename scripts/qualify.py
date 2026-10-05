#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Bounded native qualification, verified archives and hidden child tooling."""
import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import platform
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
BUILD = ROOT / 'build/native'

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
    if LOCK['cueson']['version'] not in version:
        raise ValueError('wrong Cueson version')
    # Validate exact official packaged schema bytes emitted by upstream.
    schema = child([exe, 'schema'])
    if hashlib.sha256(schema).hexdigest() != LOCK['cueson']['schema_sha256']:
        raise ValueError('Cueson schema identity mismatch')
    with tempfile.TemporaryDirectory() as directory:
        directory = Path(directory)
        fixture = directory / 'source.srt'
        data = b'1\n00:00:00,000 --> 00:00:01,000\nQualification source.\n\n'
        fixture.write_bytes(data)
        document = directory / 'document.cueson'
        child([exe, 'encode', '--output', document, fixture])
        child([exe, 'validate', '--format', 'cueson', document])
        child([exe, 'render', '--to', 'srt', '--output', directory / 'rendered.srt', document])
        restored = directory / 'restored.srt'
        child([exe, 'restore', '--no-metadata', '--output', restored, document])
        if restored.read_bytes() != data:
            raise ValueError('subtitle source restoration differs')
    return {'version': version, 'schema_sha256': LOCK['cueson']['schema_sha256'], 'source_roundtrip': 'passed'}

def native():
    env = native_env()
    cueson = cueson_probe(env)
    output = child(['go', 'test', '-v', '-tags', 'native_ladybug,system_ladybug', './internal/qualification'], env=env)
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
    (BUILD / 'desktop-receipt.json').write_bytes(output)
    args = [executable, '--webview-qualification']
    if platform.system() == 'Linux':
        args = ['xvfb-run', '-a', *args]
    output = child(args, env=env, timeout=30)
    (BUILD / 'webview-receipt.json').write_bytes(output)

def secrets():
    executable = ROOT / ('build/insonic-secret-probe.exe' if os.name == 'nt' else 'build/insonic-secret-probe')
    child(['go', 'build', '-o', executable, './cmd/insonic-secret-probe'])
    try:
        output = child([executable], timeout=5)
        value = json.loads(output)
    except (RuntimeError, subprocess.TimeoutExpired):
        value = {'native_secret_service': 'unavailable', 'credential_values': 'not-returned'}
    (BUILD / 'secret-service-receipt.json').write_text(json.dumps(value) + '\n', encoding='utf-8')
    print(json.dumps(value))

def cli():
    executable = ROOT / ('build/insonic.exe' if os.name == 'nt' else 'build/insonic')
    child(['go', 'build', '-o', executable, './cmd/insonic'])
    with tempfile.TemporaryDirectory() as directory:
        child([executable, 'workspace', 'init', directory, '--json'])
        first = json.loads(child([executable, '--workspace', directory, 'workspace', 'show', '--json'], allow_detached=True))
        second = json.loads(child([executable, '--workspace', directory, 'workspace', 'show', '--json']))
        if first['runtime_session_id'] != second['runtime_session_id']:
            raise ValueError('CLI clients reached different owners')
        started = json.loads(child([executable, '--workspace', directory, 'jobs', 'start', '1000', '--json']))
        job = started['result']['job_id']
        child([executable, '--workspace', directory, 'jobs', 'cancel', job, '--json'])
        retried = json.loads(child([executable, '--workspace', directory, 'jobs', 'retry', job, '--json']))
        if retried['result']['claim_generation'] != 2:
            raise ValueError('retry generation')
        child([executable, '--workspace', directory, 'jobs', 'cancel', job, '--json'])
        # Owner remains alive until idle exit; wait before removing its private files.
        import time
        time.sleep(31)
    (BUILD / 'cli-receipt.json').write_text(json.dumps({'owner_reuse': 'passed', 'cancel_retry': 'passed'}) + '\n', encoding='utf-8')

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('stage', choices=['prepare', 'native', 'assets', 'desktop', 'cli', 'secrets'])
    options = parser.parse_args()
    globals()[options.stage]()
