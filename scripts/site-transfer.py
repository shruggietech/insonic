#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Same-run source-bound documentation reuse, without stale output merging."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re


def sibling(name):
    spec = importlib.util.spec_from_file_location('transfer_' + name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


pages, release = sibling('pages'), sibling('release')
ARCHIVES = {'documentation.tar.gz': 'out', 'offline-help.tar.gz': 'offline'}


def identity(revision, version, run_id, run_attempt):
    if (not re.fullmatch(r'[a-f0-9]{40}', revision or '') or not re.fullmatch(release.VERSION_PATTERN, version or '')
            or any(not re.fullmatch(r'[1-9][0-9]*', value or '') for value in (run_id, run_attempt))):
        raise ValueError('documentation transfer context is invalid')
    return {'kind': 'ci-documentation-transfer', 'format': 1, 'revision': revision,
            'version': version, 'run_id': run_id, 'run_attempt': run_attempt}


def entry(path):
    if path.is_symlink() or not path.is_file() or path.stat().st_size > pages.STATIC_PAYLOAD_BYTES:
        raise ValueError('documentation transfer archive is missing or excessive')
    data = path.read_bytes()
    return {'size_bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}


def verify(directory, revision, version, run_id, run_attempt):
    expected = identity(revision, version, run_id, run_attempt)
    receipt = json.loads((directory / 'transfer.json').read_text(encoding='utf-8'))
    if set(receipt) != set(expected) | {'archives'} or any(receipt.get(k) != v for k, v in expected.items()):
        raise ValueError('documentation transfer source or run differs')
    if receipt['archives'] != {name: entry(directory / name) for name in ARCHIVES}:
        raise ValueError('documentation transfer archive digest differs')
    files = {}
    for name, target in ARCHIVES.items():
        files[target] = pages.archive_files(directory / name)
        release.verify_documentation(directory / name, version, revision)
        if 'index.html' not in files[target]: raise ValueError('documentation transfer export is incomplete')
    return files


def pack(directory, revision, version, run_id, run_attempt):
    receipt = identity(revision, version, run_id, run_attempt)
    receipt['archives'] = {name: entry(directory / name) for name in ARCHIVES}
    for name in ARCHIVES:
        pages.archive_files(directory / name)
        release.verify_documentation(directory / name, version, revision)
    (directory / 'transfer.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')
    return receipt


def restore(directory, site, revision, version, run_id, run_attempt):
    files = verify(directory, revision, version, run_id, run_attempt)
    # Never remove or merge caller data, including redirected parent paths.
    if any(parent.is_symlink() or getattr(parent, 'is_junction', lambda: False)() for parent in (site, *site.parents)) or any((site / name).exists() or (site / name).is_symlink() for name in files):
        raise ValueError('documentation restore requires new nonredirected outputs')
    for entries in files.values():
        for name in entries:
            for component in name.split('/'):
                if (component.rstrip(' .') != component or re.search(r'[<>"|?*\x00-\x1f]', component)
                        or re.fullmatch(r'CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9]', component.split('.')[0], re.I)):
                    raise ValueError('documentation restore path has a Windows filesystem alias')
    for target, entries in files.items():
        for name, data in entries.items():
            path = site / target / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
    return {'documentation_transfer': 'verified', 'revision': revision}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('pack', 'restore'))
    parser.add_argument('--directory', type=Path, required=True)
    parser.add_argument('--site', type=Path, default=Path('site'))
    parser.add_argument('--revision', required=True)
    args = parser.parse_args()
    values = (args.directory, args.revision, release.versions(), os.environ.get('GITHUB_RUN_ID', ''), os.environ.get('GITHUB_RUN_ATTEMPT', ''))
    result = pack(*values) if args.command == 'pack' else restore(values[0], args.site, *values[1:])
    print(json.dumps(result))


if __name__ == '__main__': main()
