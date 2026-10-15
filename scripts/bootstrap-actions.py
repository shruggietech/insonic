#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Acquire locked setup actions at step time before normal local action execution."""
import argparse
import hashlib
import importlib.util
import io
import json
from pathlib import Path, PurePosixPath
import re
import tarfile
import tempfile
import time

MAX_COMPRESSED = 10 << 20
MAX_EXPANDED = 32 << 20
MAX_MEMBERS = 1000
NAMES = ('setup-go', 'setup-node')

spec = importlib.util.spec_from_file_location('insonic_http_read', Path(__file__).with_name('http_read.py'))
http = importlib.util.module_from_spec(spec); spec.loader.exec_module(http)


def acquire(name, output, lock, *, reader=http.read, clock=time.monotonic):
    started = clock()
    if (name not in NAMES or not isinstance(lock, dict)
            or not isinstance(lock.get('revision'), str) or not isinstance(lock.get('sha256'), str)
            or not re.fullmatch(r'[a-f0-9]{40}', lock.get('revision', ''))
            or not re.fullmatch(r'[a-f0-9]{64}', lock.get('sha256', ''))):
        raise ValueError('action name or immutable lock is invalid')
    output = Path(output)
    if any(parent.is_symlink() for parent in output.absolute().parents):
        raise ValueError('action output ancestors must not be symlinks')
    if output.exists() or output.is_symlink(): raise ValueError('action output must be new')
    revision = lock['revision']
    attempts = [1]
    data = reader(f'https://codeload.github.com/actions/{name}/tar.gz/{revision}',
                  max_bytes=MAX_COMPRESSED, deadline=started + 60, retry_404=True,
                  on_attempt=lambda value: attempts.__setitem__(0, value))
    if len(data) > MAX_COMPRESSED: raise ValueError('action compressed size exceeds limit')
    if hashlib.sha256(data).hexdigest() != lock['sha256']:
        raise ValueError('action archive digest differs from approved lock')
    output.parent.mkdir(parents=True, exist_ok=True)
    try:
        with tempfile.TemporaryDirectory(prefix='action-', dir=output.parent) as temporary:
            stage = Path(temporary) / 'tree'; stage.mkdir()
            seen, total, count = set(), 0, 0
            prefix = f'{name}-{revision}'
            with tarfile.open(fileobj=io.BytesIO(data), mode='r:gz') as archive:
                for member in archive:
                    if clock() > started + 60: raise TimeoutError('action bootstrap deadline exceeded')
                    count += 1
                    raw = member.name.removesuffix('/') if member.isdir() else member.name
                    path = PurePosixPath(raw)
                    if (not raw or path.as_posix() != raw or path.is_absolute()
                            or path.parts[0] != prefix or '..' in path.parts
                            or '\\' in raw or ':' in raw or any(p.casefold() == '.git' for p in path.parts)
                            or any(p.endswith(('.', ' ')) or re.fullmatch(r'(?i)(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])(?:\..*)?', p) for p in path.parts)
                            or not (member.isfile() or member.isdir())
                            or raw.casefold() in seen or count > MAX_MEMBERS):
                        raise ValueError('action archive has unsafe, duplicate or excessive members')
                    seen.add(raw.casefold())
                    if len(path.parts) == 1:
                        if not member.isdir(): raise ValueError('action archive root must be a directory')
                        continue
                    destination = stage.joinpath(*path.parts[1:])
                    if member.isdir(): destination.mkdir(parents=True, exist_ok=True); continue
                    total += member.size
                    if member.size < 0 or total > MAX_EXPANDED:
                        raise ValueError('action expanded size exceeds limit')
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    payload = archive.extractfile(member).read(member.size + 1)
                    if len(payload) != member.size: raise ValueError('action member size differs')
                    destination.write_bytes(payload)
                    destination.chmod(0o755 if member.mode & 0o111 else 0o644)
            if not (stage / 'action.yml').is_file(): raise ValueError('action manifest is missing')
            if clock() > started + 60: raise TimeoutError('action bootstrap deadline exceeded')
            if output.exists() or output.is_symlink(): raise ValueError('action output must be new')
            stage.rename(output)
    except (tarfile.TarError, EOFError) as error:
        raise ValueError('action archive is invalid') from error
    return {'action': name, 'revision': revision, 'sha256': lock['sha256'],
            'attempts': attempts[0], 'verified': True}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=NAMES)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    lock = json.loads(Path(__file__).with_suffix('.lock.json').read_text(encoding='utf-8'))
    print(json.dumps(acquire(args.action, args.output, lock[args.action])))


if __name__ == '__main__':
    try: main()
    except (OSError, ValueError) as error:
        # Transport diagnostics deliberately exclude URLs and response bodies.
        raise SystemExit(str(error)) from None
