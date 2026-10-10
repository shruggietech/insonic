#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Pinned Windows OpenSSL ABI-3 companions for the embedded graph loader."""
import hashlib
import json
from pathlib import Path
import sys
import tarfile
import time
import urllib.error
import urllib.request
import zipfile

ROOT = Path(__file__).resolve().parent.parent
BUILD = ROOT / 'build/native'
SELECTED = {'Library/bin/libssl-3-x64.dll': 'libssl-3-x64.dll',
            'Library/bin/libcrypto-3-x64.dll': 'libcrypto-3-x64.dll',
            'info/licenses/LICENSE.txt': 'OpenSSL-LICENSE.txt'}

def fetch_archive(pin, archive):
    if archive.is_file() and hashlib.sha256(archive.read_bytes()).hexdigest() == pin['sha256']:
        return
    deadline = time.monotonic() + 60
    temporary = archive.with_suffix('.download')
    try:
        for attempt in range(3):
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise TimeoutError('OpenSSL acquisition budget exceeded')
            try:
                request = urllib.request.Request(pin['url'], headers={'User-Agent': 'insonic-native-qualification'})
                with urllib.request.urlopen(request, timeout=min(20, remaining)) as source, temporary.open('wb') as output:
                    size = 0
                    while True:
                        if time.monotonic() >= deadline:
                            raise TimeoutError('OpenSSL acquisition deadline reached')
                        chunk = source.read(1024 * 1024)
                        if not chunk:
                            break
                        size += len(chunk)
                        if size > 32 << 20:
                            raise ValueError('oversized OpenSSL archive')
                        output.write(chunk)
                break
            except (urllib.error.URLError, TimeoutError, ConnectionError) as error:
                if (isinstance(error, urllib.error.HTTPError) and error.code not in [408, 429, 500, 502, 503, 504]) or attempt == 2:
                    raise
                time.sleep(attempt + 1)
        if hashlib.sha256(temporary.read_bytes()).hexdigest() != pin['sha256']:
            raise ValueError('OpenSSL archive checksum mismatch')
        temporary.replace(archive)
    finally:
        temporary.unlink(missing_ok=True)

def extract_members(source, destination, found):
    for entry in source:
        name = SELECTED.get(entry.name)
        if name is None:
            continue
        if not entry.isfile() or entry.size > 16 << 20 or name in found:
            raise ValueError('invalid or duplicate OpenSSL companion')
        data = source.extractfile(entry)
        if data is None:
            raise ValueError('missing OpenSSL companion data')
        payload = data.read()
        target = destination / name
        if not target.is_file() or target.read_bytes() != payload:
            target.write_bytes(payload)
        found.add(name)

def prepare():
    pin = json.loads((ROOT / 'internal/qualification/dependencies.json').read_text())['windows_openssl']
    BUILD.mkdir(parents=True, exist_ok=True)
    archive = BUILD / pin['filename']
    fetch_archive(pin, archive)
    sys.path.insert(0, str(BUILD / 'python'))
    import zstandard
    destination = BUILD / 'openssl'
    destination.mkdir(exist_ok=True)
    found = set()
    with zipfile.ZipFile(archive) as source:
        members = [entry for entry in source.infolist() if entry.filename.endswith('.tar.zst')]
        if len(members) != 2:
            raise ValueError('invalid conda package members')
        for entry in members:
            if entry.file_size > 32 << 20:
                raise ValueError('oversized OpenSSL package')
            with source.open(entry) as compressed:
                with zstandard.ZstdDecompressor().stream_reader(compressed) as data:
                    with tarfile.open(fileobj=data, mode='r|') as contents:
                        extract_members(contents, destination, found)
    if found != set(SELECTED.values()):
        raise ValueError('OpenSSL DLLs or redistribution license missing')
    receipt = {**pin, 'files': {name: hashlib.sha256((destination / name).read_bytes()).hexdigest() for name in sorted(found)}}
    (BUILD / 'openssl-receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')
    return destination
if __name__ == '__main__':
    prepare()
