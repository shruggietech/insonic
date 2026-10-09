# SPDX-License-Identifier: Apache-2.0
"""Verify compact same-run media build transfers without compiling on consumers."""
import argparse
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import sys
import tarfile
import tempfile

import media_source_build as source


def relative(name):
    path = PurePosixPath(name)
    if (not isinstance(name, str) or not path.parts or path.is_absolute()
            or '..' in path.parts or ':' in name or '\\' in name or path.as_posix() != name):
        raise ValueError('invalid transfer member path')
    return path


def source_pins_key():
    import hashlib
    return hashlib.sha256(json.dumps(source.source_pins(), sort_keys=True, separators=(',', ':')).encode()).hexdigest()


def prepare_sources():
    pins = source.source_pins()
    for pin in pins.values():
        path = source.fetch_pin(pin)
        if path.is_symlink() or source.sha(path) != pin['sha256']:
            raise ValueError('original media source archive differs from the pinned identity')
    return {'source_archives': len(pins), 'source_pins_key': source_pins_key()}


def context(revision):
    if not re.fullmatch(r'[0-9a-f]{40}', revision):
        raise ValueError('transfer requires an exact source revision')
    if source.child(['git', 'rev-parse', 'HEAD'], source.ROOT).decode().strip() != revision:
        raise ValueError('transfer source revision differs from this checkout')
    platform, names, key, compiler, recipe = source.cache_identity()
    return {'revision': revision, 'platform': platform, 'key': key,
            'compiler': compiler, 'recipe_sha256': recipe}, names


def receipt_files(stage, receipt, names):
    if stage == 'dependencies':
        files = set(receipt['files'])
        if any(not name.startswith('install/') for name in files):
            raise ValueError('dependency receipt contains files outside its private prefix')
        files.add('dependency-receipt.json')
    else:
        files = set(names) | {'build-receipt.json'}
        files.update(item['path'] for item in receipt['static_libraries'])
        files.update(item['path'] for item in receipt.get('runtime_notices', []))
    for name in files:
        relative(name)
    return files


def verified_receipt(directory, stage, identity, names, temporary=False):
    filename = 'dependency-receipt.json' if stage == 'dependencies' else 'build-receipt.json'
    receipt = json.loads((directory / filename).read_text(encoding='utf-8'))
    if any(receipt.get(field) != identity[field] for field in ['key', 'platform', 'compiler', 'recipe_sha256']):
        raise ValueError('media receipt does not match the current source toolchain')
    receipt_files(stage, receipt, names)
    if stage == 'dependencies':
        final = source.BUILD / identity['key'] / 'install'
        if receipt.get('install_prefix') != str(final):
            raise ValueError('dependency prefix does not match this runner workspace')
        check = dict(receipt, install_prefix=str(directory / 'install')) if temporary else receipt
        valid = source.dependency_cache_valid(directory, check, identity['key'])
    else:
        valid = source.cache_valid(directory, receipt, identity['key'], names)
    if not valid:
        raise ValueError('media transfer has incomplete or changed build files')
    return receipt


def ready(stage, revision):
    """Read-only cache probe; a missing or invalid closure is never ready."""
    identity, names = context(revision)
    directory = source.BUILD / identity['key']
    try:
        verified_receipt(directory, stage, identity, names)
        for pin in source.source_pins().values():
            name = pin['name']
            relative(name)
            if '/' in name:
                return False
            path = source.ROOT / 'build/media-source-pins' / name
            if (not path.is_file() or path.is_symlink() or source.sha(path) != pin['sha256']
                    or ('size_bytes' in pin and path.stat().st_size != pin['size_bytes'])):
                return False
    except (ValueError, KeyError, TypeError, AttributeError, OSError):
        return False
    return True


def pack(stage, revision, output):
    identity, names = context(revision)
    directory = source.BUILD / identity['key']
    receipt = verified_receipt(directory, stage, identity, names)
    files = {'media/' + name: directory / name for name in receipt_files(stage, receipt, names)}
    for pin in source.source_pins().values():
        name = pin['name']
        relative(name)
        if '/' in name:
            raise ValueError('source pin must be a basename')
        path = source.ROOT / 'build/media-source-pins' / name
        if not path.is_file() or path.is_symlink() or source.sha(path) != pin['sha256']:
            raise ValueError('missing or changed original media source archive')
        files['sources/' + name] = path
    inventory = {}
    for name, path in sorted(files.items()):
        if not path.is_file() or path.is_symlink():
            raise ValueError('transfer requires regular build files')
        inventory[name] = {'sha256': source.sha(path), 'size_bytes': path.stat().st_size,
                           'mode': path.stat().st_mode & 0o777}
    manifest = dict(identity, kind='media-build-transfer', stage=stage, files=inventory,
                    source_pins=source.source_pins())
    output = Path(output)
    output.parent.mkdir(parents=True, exist_ok=True)
    temporary = output.with_name(output.name + '.partial')
    try:
        with tarfile.open(temporary, 'w:gz') as archive:
            import io
            data = (json.dumps(manifest, indent=2, sort_keys=True) + '\n').encode()
            member = tarfile.TarInfo('transfer.json')
            member.size, member.mode = len(data), 0o644
            archive.addfile(member, io.BytesIO(data))
            for name, path in sorted(files.items()):
                member = archive.gettarinfo(str(path), name)
                member.uid = member.gid = 0
                member.uname = member.gname = ''
                with path.open('rb') as stream:
                    archive.addfile(member, stream)
        # The exported bytes must still describe the source files read for the inventory.
        for name, path in files.items():
            if source.sha(path) != inventory[name]['sha256']:
                raise ValueError('build files changed while transferring')
        temporary.replace(output)
    finally:
        temporary.unlink(missing_ok=True)
    return manifest


def restore(stage, revision, archive_path):
    identity, names = context(revision)
    build = source.ROOT / 'build'
    build.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='media-transfer-', dir=build) as staging_name:
        staging = Path(staging_name)
        seen = set()
        with tarfile.open(archive_path) as archive:
            manifest = None
            for member in archive:
                path = relative(member.name)
                folded = member.name.casefold()
                if folded in seen or not member.isfile():
                    raise ValueError('transfer contains duplicate or nonregular members')
                seen.add(folded)
                if member.name == 'transfer.json':
                    if member.size > 2 * 1024 * 1024:
                        raise ValueError('transfer inventory is too large')
                    manifest = json.loads(archive.extractfile(member).read())
                    if (manifest.get('kind') != 'media-build-transfer' or manifest.get('stage') != stage
                            or any(manifest.get(field) != value for field, value in identity.items())
                            or manifest.get('source_pins') != source.source_pins()):
                        raise ValueError('transfer identity does not match exact source and compiler')
                    for name, item in manifest['files'].items():
                        parts = relative(name).parts
                        if parts[0] not in ['media', 'sources'] or len(parts) < 2:
                            raise ValueError('transfer inventory has an unexpected root')
                        if not isinstance(item.get('size_bytes'), int) or item['size_bytes'] < 0:
                            raise ValueError('transfer inventory has an invalid size')
                else:
                    if manifest is None or member.name not in manifest['files']:
                        raise ValueError('transfer member is absent from completion inventory')
                    item = manifest['files'][member.name]
                    if member.size != item['size_bytes'] or member.mode != item['mode']:
                        raise ValueError('transfer member metadata changed')
                    target = staging.joinpath(*path.parts)
                    target.parent.mkdir(parents=True, exist_ok=True)
                    with archive.extractfile(member) as stream, target.open('xb') as output:
                        shutil.copyfileobj(stream, output)
                    target.chmod(member.mode & 0o777)
                    if source.sha(target) != item['sha256']:
                        raise ValueError('transfer member digest changed')
        if manifest is None or seen != {'transfer.json'} | {name.casefold() for name in manifest['files']}:
            raise ValueError('transfer is incomplete')
        receipt = verified_receipt(staging / 'media', stage, identity, names, temporary=True)
        expected = {'media/' + name for name in receipt_files(stage, receipt, names)}
        for pin in source.source_pins().values():
            name = pin['name']
            relative(name)
            if '/' in name:
                raise ValueError('source pin must be a basename')
            path = staging / 'sources' / name
            if (not path.is_file() or source.sha(path) != pin['sha256']
                    or ('size_bytes' in pin and path.stat().st_size != pin['size_bytes'])):
                raise ValueError('transfer source archive differs from the pinned original')
            expected.add('sources/' + name)
        if set(manifest['files']) != expected:
            raise ValueError('transfer inventory is not the exact required build closure')
        # Extract only after every transferred file and identity has been checked.
        if stage == 'complete':
            for name, pin in source.source_pins().items():
                source.extract_source(staging / 'sources' / pin['name'], staging / 'media' / (name + '-source'), pin['root'])
        destination = source.BUILD / identity['key']
        if not destination.resolve().is_relative_to(build.resolve()):
            raise ValueError('transfer destination escapes the workspace build directory')
        destination.parent.mkdir(parents=True, exist_ok=True)
        pins = build / 'media-source-pins'
        pins.mkdir(exist_ok=True)
        if destination.is_symlink() or pins.is_symlink():
            raise ValueError('transfer destination is a symbolic link')
        originals = staging / 'previous-sources'
        originals.mkdir()
        for path in (staging / 'sources').iterdir():
            target = pins / path.name
            if target.is_symlink() or (target.exists() and not target.is_file()):
                raise ValueError('source archive destination is not a regular file')
        previous = staging / 'previous-media'
        moved_pins = []
        if destination.exists():
            destination.replace(previous)
        try:
            (staging / 'media').replace(destination)
            verified_receipt(destination, stage, identity, names)
            for path in (staging / 'sources').iterdir():
                target = pins / path.name
                if target.exists():
                    target.replace(originals / path.name)
                moved_pins.append(path.name)
                path.replace(target)
        except BaseException:
            if destination.exists():
                shutil.rmtree(destination)
            if previous.exists():
                previous.replace(destination)
            for name in moved_pins:
                target = pins / name
                target.unlink(missing_ok=True)
                if (originals / name).exists():
                    (originals / name).replace(target)
            raise
    return identity


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    commands.add_parser('identity')
    commands.add_parser('sources')
    for command in ['pack', 'restore', 'ready']:
        item = commands.add_parser(command)
        item.add_argument('--stage', choices=['dependencies', 'complete'], required=True)
        item.add_argument('--revision', required=True)
        if command == 'pack':
            item.add_argument('--output', required=True)
        elif command == 'restore':
            item.add_argument('archive')
    args = parser.parse_args()
    if args.command == 'identity':
        platform, _, key, compiler, recipe = source.cache_identity()
        result = {'platform': platform, 'key': key, 'compiler': compiler, 'recipe_sha256': recipe,
                  'source_pins_key': source_pins_key()}
        if os.environ.get('GITHUB_OUTPUT'):
            with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8', newline='\n') as output:
                output.write('key=' + key + '\n')
                output.write('source_pins_key=' + result['source_pins_key'] + '\n')
    elif args.command == 'sources':
        result = prepare_sources()
    elif args.command == 'pack':
        manifest = pack(args.stage, args.revision, args.output)
        result = {field: manifest[field] for field in ['stage', 'revision', 'platform', 'key']}
        result['file_count'] = len(manifest['files'])
    elif args.command == 'ready':
        result = {'ready': ready(args.stage, args.revision)}
        if os.environ.get('GITHUB_OUTPUT'):
            with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8', newline='\n') as output:
                output.write('ready=' + str(result['ready']).lower() + '\n')
    else:
        result = restore(args.stage, args.revision, args.archive)
    print(json.dumps(result, sort_keys=True))


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError, tarfile.TarError) as error:
        print('media transfer failed: ' + str(error), file=sys.stderr)
        sys.exit(1)
