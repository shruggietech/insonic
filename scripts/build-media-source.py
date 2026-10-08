#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Pinned redistributable macOS media companions; never builds model engines."""
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import platform
import shutil
import subprocess
import tarfile
import time
import urllib.request

from process_tree import ProcessTree

ROOT = Path(__file__).resolve().parent.parent
PIN = json.loads((ROOT / 'internal/qualification/media-tools.json').read_text(encoding='utf-8'))['source_build']
COMMIT = PIN['commit']
SOURCE_SHA256 = PIN['sha256']
SOURCE_URL = PIN['url']
CONFIGURE = PIN['configure']
BUILD = ROOT / 'build/native/source-media'


def sha(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def fetch_source(commit=COMMIT, digest=SOURCE_SHA256):
    cache = ROOT / 'build/media-source-pins'
    cache.mkdir(parents=True, exist_ok=True)
    target = cache / ('ffmpeg-' + commit + '.tar.gz')
    if not target.exists() or sha(target) != digest:
        temporary = target.with_suffix('.download')
        try:
            with urllib.request.urlopen('https://codeload.github.com/FFmpeg/FFmpeg/tar.gz/' + commit, timeout=90) as source, temporary.open('wb') as output:
                shutil.copyfileobj(source, output)
            if sha(temporary) != digest:
                raise ValueError('FFmpeg source digest mismatch')
            temporary.replace(target)
        finally:
            temporary.unlink(missing_ok=True)
    return target


def child(args, directory, timeout=360):
    env = os.environ.copy()
    env['MACOSX_DEPLOYMENT_TARGET'] = '13.3'
    with ProcessTree([str(arg) for arg in args], cwd=directory, env=env) as tree:
        try:
            output, error = tree.process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            tree.kill()
            tree.process.communicate(timeout=5)
            raise
        if tree.process.returncode:
            raise RuntimeError('source media build failed: ' + error.decode(errors='replace')[-8000:])
        return output


def extract_source(source, directory):
    root_name = 'FFmpeg-' + COMMIT
    prefix = root_name + '/'
    with tarfile.open(source) as archive:
        for item in archive:
            # tarfile normalizes codeload's top directory without its slash.
            if item.name == root_name and item.isdir():
                continue
            if not item.name.startswith(prefix):
                raise ValueError('unexpected FFmpeg source root')
            relative = PurePosixPath(item.name[len(prefix):])
            if not relative.parts:
                continue
            if relative.is_absolute() or '..' in relative.parts or item.issym() or item.islnk():
                raise ValueError('unexpected FFmpeg source member')
            target = directory.joinpath(*relative.parts)
            if item.isdir():
                target.mkdir(parents=True, exist_ok=True)
            elif item.isfile():
                target.parent.mkdir(parents=True, exist_ok=True)
                with archive.extractfile(item) as data, target.open('wb') as output:
                    shutil.copyfileobj(data, output)
                target.chmod(item.mode & 0o755)
            else:
                raise ValueError('unexpected FFmpeg source type')


def prepare():
    if platform.system() != 'Darwin' or platform.machine() not in ['arm64', 'aarch64']:
        raise ValueError('source media build is qualified only for darwin_arm64')
    key = hashlib.sha256(json.dumps({'source': SOURCE_SHA256, 'configuration': CONFIGURE,
                                    'platform': 'darwin_arm64', 'deployment': '13.3'}, sort_keys=True).encode()).hexdigest()
    source = fetch_source()
    directory = BUILD / key
    receipt_path = directory / 'build-receipt.json'
    if receipt_path.exists():
        receipt = json.loads(receipt_path.read_text(encoding='utf-8'))
        if receipt.get('key') == key and all((directory / name).is_file() and sha(directory / name) == receipt['binaries'][name] for name in ['ffmpeg', 'ffprobe']):
            return directory, receipt, source
    directory.mkdir(parents=True, exist_ok=True)
    started = time.monotonic()
    extract_source(source, directory)
    child(['/bin/sh', './configure', *CONFIGURE], directory, timeout=90)
    child(['/usr/bin/make', '-j' + str(min(os.cpu_count() or 2, 4)), 'ffmpeg', 'ffprobe'], directory, timeout=360)
    versions = {}
    configurations = {}
    for name in ['ffmpeg', 'ffprobe']:
        output = child([directory / name, '-version'], directory, timeout=10).decode()
        if '--enable-nonfree' in output or '--enable-gpl' in output:
            raise ValueError('source media companion enabled incompatible distribution flags')
        versions[name] = output.splitlines()[0].split()[2]
        configurations[name] = output
    receipt = {'key': key, 'source_commit': COMMIT, 'source_sha256': SOURCE_SHA256,
               'source_url': SOURCE_URL, 'configure': CONFIGURE, 'platform': 'darwin_arm64',
               'binaries': {name: sha(directory / name) for name in ['ffmpeg', 'ffprobe']},
               'versions': versions, 'configuration_output': configurations,
               'elapsed_seconds': round(time.monotonic() - started, 3), 'license': 'LGPL-2.1-or-later',
               'external_codec_libraries': 'none', 'system_framework': 'VideoToolbox'}
    receipt_path.write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    return directory, receipt, source


if __name__ == '__main__':
    directory, receipt, source = prepare()
    print(json.dumps(receipt))
