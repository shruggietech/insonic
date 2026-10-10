#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Checksum-pinned native extractors/decoder and real media acceptance."""
import argparse
import base64
import gzip
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import platform
import shutil
import subprocess
import tarfile
import urllib.request
import importlib.util

from process_tree import ProcessTree

ROOT = Path(__file__).resolve().parent.parent
BUILD = ROOT / 'build/native/media'
LOCK = json.loads((ROOT / 'internal/qualification/media-tools.json').read_text(encoding='utf-8'))
PRODUCT_VERSION = (ROOT / 'VERSION').read_text(encoding='utf-8').strip()


def sha(path):
    with path.open('rb') as data:
        return hashlib.file_digest(data, 'sha256').hexdigest()


def fetch(name, url, digest, integrity=None):
    directory = ROOT / 'build/media-pins'
    directory.mkdir(parents=True, exist_ok=True)
    path = directory / name
    if not path.exists() or sha(path) != digest:
        temporary = path.with_suffix(path.suffix + '.download')
        try:
            request = urllib.request.Request(url, headers={'User-Agent': 'insonic-media-qualification'})
            with urllib.request.urlopen(request, timeout=90) as source, temporary.open('wb') as output:
                shutil.copyfileobj(source, output)
            if sha(temporary) != digest:
                raise ValueError('media archive checksum mismatch: ' + name)
            temporary.replace(path)
        finally:
            temporary.unlink(missing_ok=True)
    if integrity:
        with path.open('rb') as data:
            actual = 'sha512-' + base64.b64encode(hashlib.file_digest(data, 'sha512').digest()).decode()
        if actual != integrity:
            raise ValueError('package maintainer integrity mismatch')
    return path


def extract(archive, target):
    # Target is one controlled, verified build child. Never follow archive links.
    if not target.resolve().is_relative_to(BUILD.resolve()):
        raise ValueError('extract destination escapes build')
    if target.exists():
        shutil.rmtree(target)
    target.mkdir(parents=True)
    total = 0
    with tarfile.open(archive) as source:
        for entry in source:
            relative = PurePosixPath(entry.name)
            if relative.is_absolute() or '..' in relative.parts or ':' in entry.name or '\\' in entry.name:
                raise ValueError('archive member escapes build')
            destination = target.joinpath(*relative.parts)
            if entry.isdir():
                destination.mkdir(parents=True, exist_ok=True)
            elif entry.isfile():
                total += entry.size
                if total > 120 << 20:
                    raise ValueError('archive expanded limit')
                destination.parent.mkdir(parents=True, exist_ok=True)
                with source.extractfile(entry) as data, destination.open('wb') as output:
                    shutil.copyfileobj(data, output)
                destination.chmod(entry.mode & 0o755)
            else:
                raise ValueError('archive contains a link or device')


def child(args, env=None, timeout=180):
    selected = os.environ.copy()
    selected.update({'PERL5OPT': '', 'PERL5LIB': '', 'PERLLIB': '', 'EXIFTOOL_HOME': '', 'LC_ALL': 'C', 'LANG': 'C'})
    if env:
        selected.update(env)
    with ProcessTree([str(arg) for arg in args], cwd=ROOT, env=selected) as tree:
        try:
            stdout, stderr = tree.process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            tree.kill()
            tree.process.communicate(timeout=5)
            raise
        if tree.process.returncode:
            raise RuntimeError('media qualification child failed: ' + (stdout + stderr).decode(errors='replace'))
        return stdout


def tool_identity(path, version_output):
    lines = version_output.decode('utf-8').splitlines()
    tokens = lines[0].split() if lines else []
    if len(tokens) < 3 or tokens[0] not in ['ffmpeg', 'ffprobe'] or tokens[1] != 'version':
        raise ValueError('unexpected native media version output')
    return {'path': str(path.resolve()), 'sha256': sha(path), 'version': tokens[2]}


def prepare_avtool(name, key, os_name):
    spec = importlib.util.spec_from_file_location('source_media', ROOT / 'scripts/build-media-source.py')
    source_media = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(source_media)
    directory, receipt, source = source_media.prepare()
    executable = BUILD / (name + '.exe' if os_name == 'windows' else name)
    shutil.copyfile(directory / executable.name, executable)
    executable.chmod(0o755)
    notices = BUILD / 'notices'
    notices.mkdir(exist_ok=True)
    for dependency, pin in source_media.source_pins().items():
        for notice in pin['notices']:
            shutil.copyfile(directory / (dependency + '-source') / notice, notices / (dependency + '-' + Path(notice).name))
    for item in receipt.get('runtime_notices', []):
        shutil.copyfile(directory / item['path'], notices / Path(item['path']).name)
    license_name, readme_name = key + '.LICENSE', key + '.README'
    shutil.copyfile(directory / 'ffmpeg-source/COPYING.LGPLv2.1', notices / license_name)
    (notices / readme_name).write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')
    (ROOT / 'build/native/media-source-build-receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')
    return tool_identity(executable, child([executable, '-version'])), {
        'name': source.name, 'sha256': source_media.SOURCE_SHA256, 'license': license_name, 'readme': readme_name}


def prepare():
    os_name = {'Windows': 'windows', 'Linux': 'linux', 'Darwin': 'darwin'}[platform.system()]
    arch = {'AMD64': 'amd64', 'x86_64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}[platform.machine()]
    key = os_name + '_' + arch
    BUILD.mkdir(parents=True, exist_ok=True)
    notices = BUILD / 'notices'
    if notices.exists():
        if notices.is_symlink() or not notices.resolve().is_relative_to(BUILD.resolve()):
            raise ValueError('media notice refresh escapes controlled build')
        shutil.rmtree(notices)
    exif = LOCK['exiftool']['windows' if os_name == 'windows' else 'unix']
    archive = fetch(exif['name'], exif['url'], exif['sha256'], exif['integrity'])
    target = BUILD / 'exiftool'
    extract(archive, target)
    executable = next(target.rglob('exiftool.exe' if os_name == 'windows' else 'exiftool'))
    executable.chmod(0o755)
    interpreter = None
    command = [executable]
    if os_name != 'windows':
        perl = Path('/usr/bin/perl').resolve()
        if not perl.is_file():
            raise ValueError('native Perl fixture unavailable')
        interpreter = {'path': str(perl), 'sha256': sha(perl)}
        command = [perl, executable]
    exif_version = child(command + ['-config', '', '-ver']).decode().strip()
    if exif_version != LOCK['exiftool']['version']:
        raise ValueError('unexpected ExifTool fixture version')
    support = [{'path': str(path), 'sha256': sha(path)} for path in sorted(target.rglob('*')) if path.is_file() and path != executable]
    probe, pin = prepare_avtool('ffprobe', key, os_name)
    decoder, decoder_pin = prepare_avtool('ffmpeg', key, os_name)
    tools = {'kind': 'media-tools', 'schema_version': PRODUCT_VERSION,
             'exiftool': {'path': str(executable), 'sha256': sha(executable), 'version': exif_version, 'support_files': support},
             'ffprobe': probe, 'ffmpeg': decoder}
    if interpreter:
        tools['exiftool']['interpreter'] = interpreter
    manifest = ROOT / 'build/native/media-tools.json'
    manifest.write_text(json.dumps(tools, indent=2) + '\n', encoding='utf-8')
    receipt = {'platform': key, 'exiftool_package': exif['name'], 'exiftool_archive_sha256': exif['sha256'],
               'ffprobe_archive': pin['name'], 'ffprobe_archive_sha256': pin['sha256'],
               'ffmpeg_archive': decoder_pin['name'], 'ffmpeg_archive_sha256': decoder_pin['sha256'],
               'exiftool_version': exif_version, 'ffprobe_version': probe['version'],
               'ffmpeg_version': decoder['version'], 'ffmpeg_sha256': decoder['sha256'],
               'notices': [pin['license'], pin['readme']]}
    (ROOT / 'build/native/media-tool-receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    return manifest, receipt


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--prepare-only', action='store_true')
    args = parser.parse_args()
    manifest, receipt = prepare()
    if not args.prepare_only:
        env = {'INSONIC_LIBRARY_TOOLS_FILE': str(manifest)}
        candidates = list((ROOT / 'build/native/cueson').rglob('cueson.exe' if os.name == 'nt' else 'cueson'))
        if candidates:
            env['CUESON_EXECUTABLE'] = str(candidates[0])
        compiler_bin = Path(os.environ.get('INSONIC_MSYS2_ROOT', 'C:/msys64')) / 'ucrt64/bin'
        if os.name == 'nt' and compiler_bin.is_dir():
            env['PATH'] = str(compiler_bin) + os.pathsep + os.environ['PATH']
        output = child(['go', 'test', '-count=1', '-v', './internal/library', './internal/app',
                        '-run', 'TestNativeMedia|TestNativeCanonical|TestNativeStereo|TestNativeAdmission|TestNativeDirectPlaybackNonzeroSourceClock'], env=env)
        print(output.decode(), end='')
        receipt['native_media'] = 'passed'
        (ROOT / 'build/native/media-receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(receipt))


if __name__ == '__main__':
    main()
