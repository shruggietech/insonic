#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Prepare only dependency libraries, then verify their exact same-attempt cache."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import subprocess
import tarfile

import qualify

toolchain_spec = importlib.util.spec_from_file_location('dependency_toolchain', Path(__file__).with_name('source-toolchain.py'))
toolchain = importlib.util.module_from_spec(toolchain_spec); toolchain_spec.loader.exec_module(toolchain)

ROOT = Path(__file__).resolve().parent.parent
CACHE = ROOT / 'build/dependency-cache'
ARTIFACT = ROOT / 'build/dependency-transfer'
MAX_BYTES = 3 << 30
MAX_FILES = 100000


def run(args, env):
    result = subprocess.run(args, cwd=ROOT, env={**os.environ, **env, 'GIT_TERMINAL_PROMPT': '0', 'GOTOOLCHAIN': 'local'},
                            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0,
                            timeout=300, check=False)
    if len(result.stdout) > 32 << 20: raise ValueError('dependency tool output exceeds bound')
    if result.returncode:
        raise ValueError('dependency tool failed: ' + result.stderr.decode('utf-8', errors='replace')[-8000:])
    return result.stdout.decode('utf-8')


def objects(text):
    decoder, values, offset = json.JSONDecoder(), [], 0
    while text[offset:].strip():
        offset += len(text[offset:]) - len(text[offset:].lstrip())
        value, offset = decoder.raw_decode(text, offset)
        values.append(value)
        if len(values) > 10000: raise ValueError('dependency graph exceeds bound')
    return values


def dependency_targets(packages):
    by_name = {p['ImportPath']: p for p in packages}
    if len(by_name) != len(packages): raise ValueError('ambiguous dependency graph')
    targets = set()
    for package in packages:
        main = package.get('Module', {}).get('Main') is True
        error = package.get('Error')
        if error and (not main or not re.fullmatch(r'pattern .+: no matching files found', error.get('Err', ''))):
            raise ValueError('dependency discovery failed')
        if not main: continue
        for name in package.get('Imports', []):
            if name in ('C', 'unsafe'): continue
            dependency = by_name.get(name, {})
            if dependency.get('Module', {}).get('Main') is True: continue
            module = dependency.get('Module', {})
            # Go omits Main=false in JSON. A named module with Main absent is external.
            external = bool(module.get('Path')) and module.get('Main', False) is False
            if dependency.get('Name') == 'main' or not (dependency.get('Standard') is True or external):
                raise ValueError('dependency target is executable or has ambiguous module ownership')
            if dependency.get('Error'): raise ValueError('dependency target has errors')
            targets.add(name)
    if not targets: raise ValueError('no dependency library targets discovered')
    pending, visited = list(targets), set()
    while pending:
        name = pending.pop()
        if name in visited or name in ('C', 'unsafe'): continue
        visited.add(name)
        dependency = by_name.get(name)
        if not dependency or dependency.get('Module', {}).get('Main') is True or dependency.get('Name') == 'main':
            raise ValueError('dependency closure would compile product or executable code')
        pending.extend(dependency.get('Imports', []))
    return sorted(targets)


def compile_context(context, env):
    tags = ['-tags', context['tags']] if context['tags'] else []
    flags = context.get('flags', [])
    packages = objects(run(['go', 'list', '-deps', '-e', '-json', *flags, *tags, context['package']], env))
    targets = dependency_targets(packages)
    # These are named libraries, never ./cmd, ./..., executable or test targets.
    run(['go', 'build', *flags, *tags, *targets], env)
    return targets


def digest(path):
    if path.is_symlink() or not path.is_file() or path.stat().st_size > MAX_BYTES:
        raise ValueError('dependency archive is missing or excessive')
    with path.open('rb') as stream:
        return {'size_bytes': path.stat().st_size, 'sha256': hashlib.file_digest(stream, 'sha256').hexdigest()}


def valid_name(name):
    parts = PurePosixPath(name).parts
    if not parts or parts[0] not in ('go', 'mod') or str(PurePosixPath(name)) != name or '\\' in name:
        raise ValueError('dependency archive path is invalid')
    for part in parts:
        if (part in ('.', '..') or part.rstrip(' .') != part or re.search(r'[<>:"|?*\x00-\x1f]', part)
                or re.fullmatch(r'CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9]', part.split('.')[0], re.I)):
            raise ValueError('dependency archive path aliases another file')


def pack(cache, artifact, identity):
    artifact.mkdir(parents=True, exist_ok=True)
    count, size = 0, 0
    with tarfile.open(artifact / 'dependencies.tar', 'w') as archive:
        for file in sorted(cache.rglob('*')):
            if file.is_symlink() or getattr(file, 'is_junction', lambda: False)():
                raise ValueError('dependency cache contains redirected paths')
            if file.is_dir(): continue
            name = file.relative_to(cache).as_posix()
            valid_name(name)
            count, size = count + 1, size + file.stat().st_size
            if count > MAX_FILES or size > MAX_BYTES: raise ValueError('dependency cache exceeds bound')
            archive.add(file, arcname=name, recursive=False)
    if not count: raise ValueError('dependency cache is empty')
    receipt = {'format': 'insonic-go-dependencies-v1', 'identity': identity,
               'archive': digest(artifact / 'dependencies.tar'), 'files': count}
    (artifact / 'transfer.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')


def restore(artifact, output, identity):
    manifest = artifact / 'transfer.json'
    if manifest.is_symlink() or manifest.stat().st_size > 4 << 20: raise ValueError('dependency manifest exceeds bound')
    receipt = json.loads(manifest.read_text(encoding='utf-8'))
    if (set(receipt) != {'format', 'identity', 'archive', 'files'} or receipt['format'] != 'insonic-go-dependencies-v1'
            or receipt['identity'] != identity or receipt['archive'] != digest(artifact / 'dependencies.tar')):
        raise ValueError('dependency source, compiler, context, run or archive differs')
    if any(p.is_symlink() or getattr(p, 'is_junction', lambda: False)() for p in (output, *output.parents)) or output.exists():
        raise ValueError('dependency activation requires a fresh nonredirected path')
    with tarfile.open(artifact / 'dependencies.tar') as archive:
        names, size, entries = set(), 0, []
        for entry in archive:
            valid_name(entry.name)
            alias = entry.name.casefold()
            size += entry.size
            if not entry.isfile() or alias in names or size > MAX_BYTES or len(names) >= MAX_FILES:
                raise ValueError('dependency archive contains links, duplicate or excessive entries')
            names.add(alias); entries.append(entry)
        if not entries or len(entries) != receipt['files']: raise ValueError('dependency archive inventory differs')
        if any(str(parent).casefold() in names for entry in entries for parent in PurePosixPath(entry.name).parents):
            raise ValueError('dependency archive has conflicting file parents')
        # Validate the entire archive before writing any cache entry.
        for entry in entries:
            target = output.joinpath(*PurePosixPath(entry.name).parts)
            target.parent.mkdir(parents=True, exist_ok=True)
            with archive.extractfile(entry) as source, target.open('xb') as destination:
                shutil.copyfileobj(source, destination)


def contexts():
    spec = importlib.util.spec_from_file_location('dependency_package', ROOT / 'scripts/package-desktop.py')
    package = importlib.util.module_from_spec(spec); spec.loader.exec_module(package)
    values = qualify.native_env()
    selected = []
    for variant in ('cli', 'desktop'):
        native = ROOT / 'build/packages' / ('insonic_' + package.VERSION + '_windows_amd64_' + variant) / 'native'
        env = package.package_build_environment(values, 'windows_amd64', native)
        targets = [('./cmd/insonic', 'system_ladybug'), ('./cmd/insonic-launcher', '')]
        if variant == 'desktop': targets.append(('./cmd/insonic-desktop', 'desktop,production,system_ladybug'))
        selected.extend(({'variant': variant, 'package': target, 'tags': tags, 'cgo_cflags': env['CGO_CFLAGS'],
                          'cgo_ldflags': env['CGO_LDFLAGS']}, env) for target, tags in targets)
    import windows_setup
    selected.append(({'variant': 'setup', 'package': './cmd/insonic-setup', 'tags': '',
                      'flags': ['-mod=readonly', '-trimpath', '-buildvcs=false'], 'cgo_enabled': '0'},
                     windows_setup.compiler_environment()))
    return selected


def identity(selected):
    revision, run_id, attempt = (os.environ.get(name, '') for name in ('SOURCE_REVISION', 'GITHUB_RUN_ID', 'GITHUB_RUN_ATTEMPT'))
    if not re.fullmatch(r'[a-f0-9]{40}', revision) or any(not re.fullmatch(r'[1-9][0-9]*', x) for x in (run_id, attempt)):
        raise ValueError('dependency transfer identity is invalid')
    files = ['go.mod', 'go.sum', 'internal/qualification/dependencies.json', 'scripts/go-dependency-transfer.py',
             'scripts/package-desktop.py', 'scripts/qualify.py', 'scripts/windows_setup.py']
    files.extend(p.relative_to(ROOT).as_posix() for p in (ROOT / 'third_party/go-webview2').rglob('*') if p.is_file())
    observation = toolchain.inventory()
    return {'revision': revision, 'run': run_id, 'attempt': attempt, 'platform': observation['platform'],
            'machine': observation['machine'], 'packages': observation['packages'],
            'go': run(['go', 'version'], {}), 'gcc': run(['gcc', '--version'], selected[0][1]),
            'recipe': {name: digest(ROOT / name)['sha256'] for name in sorted(files)},
            'native': json.loads((ROOT / 'build/native/asset-receipt.json').read_text(encoding='utf-8')),
            'contexts': [context for context, _ in selected]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('prepare', 'restore'))
    args = parser.parse_args()
    if os.name != 'nt': raise ValueError('Windows package dependencies are the elected preparation target')
    selected = contexts()
    expected = identity(selected)
    if args.command == 'prepare':
        if CACHE.exists(): raise ValueError('preparation requires an empty isolated cache')
        env = {'GOCACHE': str(CACHE / 'go'), 'GOMODCACHE': str(CACHE / 'mod')}
        # Dependency download and verification do not compile application packages.
        run(['go', 'mod', 'download'], env)
        # Match the consumer's CGO library search directories using pinned native
        # inputs only. No application assets or outputs are prepared here.
        for variant in ('cli', 'desktop'):
            native = ROOT / 'build/packages' / ('insonic_' + qualify.VERSION + '_windows_amd64_' + variant) / 'native'
            native.mkdir(parents=True)
            for library in (ROOT / 'build/native/ladybug').rglob('*.dll'):
                shutil.copyfile(library, native / library.name)
        targets = []
        for context, values in selected:
            targets.append(compile_context(context, {**values, **env}))
        run(['go', 'mod', 'verify'], env)
        pack(CACHE, ARTIFACT, expected)
        (ARTIFACT / 'preparation.json').write_text(json.dumps({'contexts': expected['contexts'], 'library_targets': targets}, indent=2) + '\n', encoding='utf-8', newline='\n')
    else:
        restore(ARTIFACT, CACHE, expected)
        env = {'GOCACHE': str(CACHE / 'go'), 'GOMODCACHE': str(CACHE / 'mod')}
        run(['go', 'mod', 'verify'], env)
        with Path(os.environ['GITHUB_ENV']).open('a', encoding='utf-8', newline='\n') as output:
            output.write(''.join(key + '=' + value + '\n' for key, value in env.items()))
    print(json.dumps({'go_dependencies': args.command, 'revision': expected['revision'], 'run': expected['run'], 'attempt': expected['attempt']}))


if __name__ == '__main__': main()
