# SPDX-License-Identifier: Apache-2.0
"""Notice evidence for the module union linked into selected product binaries."""
import json
from pathlib import Path
import re
import shutil


def json_stream(raw):
    remaining = raw.decode('utf-8') if isinstance(raw, bytes) else raw
    decoder = json.JSONDecoder()
    values = []
    while remaining.strip():
        value, end = decoder.raw_decode(remaining.lstrip())
        if not isinstance(value, dict):
            raise ValueError('Go metadata must contain objects')
        values.append(value)
        remaining = remaining.lstrip()[end:]
    return values


def escaped(value):
    return ''.join('!' + letter.lower() if 'A' <= letter <= 'Z' else letter for letter in value)


def source_identity(value):
    path, version, checksum = (value.get(key, '') for key in ('Path', 'Version', 'Sum'))
    if (not isinstance(path, str) or not re.fullmatch(r'[A-Za-z0-9._~/-]+', path)
            or any(part in ('', '.', '..') for part in path.split('/'))
            or not isinstance(version, str) or not re.fullmatch(r'v[0-9][0-9A-Za-z.+-]*', version)
            or not isinstance(checksum, str) or not re.fullmatch(r'h1:[A-Za-z0-9+/]{43}=', checksum)):
        raise ValueError('module source identity requires path, version and checksum')
    return path, version, checksum


def effective_source(value):
    replacement = value.get('Replace')
    if replacement is not None and not isinstance(replacement, dict):
        raise ValueError('module replacement identity is invalid')
    return replacement if replacement is not None else value


def logical_identity(value):
    # A replaced logical module has no linked source checksum of its own.
    source = effective_source(value)
    effective = source_identity(source)
    if source is value:
        return effective[0], effective[1], effective
    path, version = value.get('Path'), value.get('Version')
    source_identity({'Path': path, 'Version': version, 'Sum': effective[2]})
    return path, version, effective


def linked_modules(binaries, run, go_version):
    if not binaries:
        raise ValueError('selected product binaries are required')
    infos = json_stream(run(['go', 'version', '-m', '-json', *[str(path) for path, _ in binaries]]))
    if len(infos) != len(binaries):
        raise ValueError('product binary metadata count differs')
    logical, sources = {}, {}
    for info, (_, command) in zip(infos, binaries):
        if info.get('GoVersion') != 'go' + go_version or info.get('Path') != command:
            raise ValueError('product binary compiler or command differs')
        dependencies = info.get('Deps')
        if not isinstance(dependencies, list) or not dependencies:
            raise ValueError('product binary dependency metadata is missing')
        seen = set()
        for dependency in dependencies:
            identity = logical_identity(dependency)
            path = identity[0]
            if path in seen or (path in logical and logical[path] != identity):
                raise ValueError('conflicting linked module identity: ' + path)
            seen.add(path)
            logical[path] = identity
            effective = identity[2]
            if effective[0] in sources and sources[effective[0]] != effective:
                raise ValueError('conflicting effective source identity: ' + effective[0])
            sources[effective[0]] = effective
    return logical, sources


def notice_prefix(path, version):
    return Path('go') / (escaped(path) + '@' + escaped(version))


def source_notices(directory):
    paths = []
    for path in sorted(directory.iterdir()):
        if path.name.upper().startswith(('LICENSE', 'LICENCE', 'COPYING', 'NOTICE')):
            if path.is_symlink() or not path.is_file() or not path.read_bytes().strip():
                raise ValueError('upstream notice is missing or empty: ' + str(path))
            paths.append(path)
    if not any(path.name.upper().startswith(('LICENSE', 'LICENCE', 'COPYING')) for path in paths):
        raise ValueError('upstream module license missing: ' + str(directory))
    return paths


def collect_notices(target, binaries, run, go_version):
    logical, sources = linked_modules(binaries, run, go_version)
    selected = json_stream(run(['go', 'list', '-mod=readonly', '-m', '-json', *sorted(logical)]))
    environment = json.loads(run(['go', 'env', '-json', 'GOMODCACHE', 'GOROOT', 'GOVERSION']))
    if environment.get('GOVERSION') != 'go' + go_version:
        raise ValueError('notice compiler version differs from product binaries')
    cache = Path(environment['GOMODCACHE']).resolve()
    resolved = {}
    seen = set()
    for value in selected:
        identity = logical_identity(value)
        if value.get('Main') or identity[0] in seen or logical.get(identity[0]) != identity:
            raise ValueError('resolved module identity does not match linked binary')
        seen.add(identity[0])
        source = effective_source(value)
        directory_text = source.get('Dir')
        if not isinstance(directory_text, str) or not directory_text or not Path(directory_text).is_absolute():
            raise ValueError('module source directory is absent or not absolute: ' + identity[0])
        directory = Path(directory_text)
        effective = identity[2]
        expected = cache / (escaped(effective[0]) + '@' + escaped(effective[1]))
        if not directory.is_dir() or directory.resolve() != expected:
            raise ValueError('module source directory does not match cache identity: ' + identity[0])
        paths = source_notices(directory)
        if effective[0] in resolved and resolved[effective[0]] != (directory, paths):
            raise ValueError('conflicting resolved source directory: ' + effective[0])
        resolved[effective[0]] = directory, paths
    if seen != set(logical) or set(resolved) != set(sources):
        raise ValueError('resolved module membership differs from linked binary')
    modules = []
    for path, version, _ in sorted(sources.values()):
        references = []
        for original in resolved[path][1]:
            relative = notice_prefix(path, version) / original.name
            destination = target / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(original, destination)
            if destination.read_bytes() != original.read_bytes():
                raise ValueError('copied module notice bytes differ')
            references.append(relative.as_posix())
        modules.append({'module': path, 'version': version, 'notices': references})
    goroot_text = environment.get('GOROOT', '')
    if not goroot_text or not Path(goroot_text).is_absolute():
        raise ValueError('compiler license directory missing')
    compiler_license = Path(goroot_text) / 'LICENSE'
    if not compiler_license.is_file() or not compiler_license.read_bytes().strip():
        raise ValueError('compiler license missing or empty')
    shutil.copyfile(compiler_license, target / 'Go-LICENSE')
    if (target / 'Go-LICENSE').read_bytes() != compiler_license.read_bytes():
        raise ValueError('copied compiler notice bytes differ')
    return modules


def verify_notices(target, modules, binaries, run, go_version):
    _, sources = linked_modules(binaries, run, go_version)
    expected = {(path, version) for path, version, _ in sources.values()}
    if not isinstance(modules, list):
        raise ValueError('module inventory membership is missing')
    actual = set()
    for module in modules:
        identity = (module.get('module'), module.get('version'))
        if identity in actual:
            raise ValueError('duplicate module inventory identity')
        actual.add(identity)
        if identity not in expected:
            raise ValueError('module inventory membership differs from linked binary')
        references = module.get('notices')
        if not isinstance(references, list) or not references or len(references) != len(set(references)):
            raise ValueError('module inventory notices are missing or duplicated')
        prefix = notice_prefix(*identity)
        names = []
        for reference in references:
            if not isinstance(reference, str):
                raise ValueError('module notice reference is invalid')
            relative = Path(reference)
            notice = target / relative
            if (relative.is_absolute() or '..' in relative.parts or relative.parent != prefix
                    or not relative.name.upper().startswith(('LICENSE', 'LICENCE', 'COPYING', 'NOTICE'))
                    or notice.is_symlink() or not notice.is_file()
                    or not notice.resolve().is_relative_to(target.resolve()) or not notice.read_bytes().strip()):
                raise ValueError('module notice reference or bytes differ')
            names.append(relative.name)
        if not any(name.upper().startswith(('LICENSE', 'LICENCE', 'COPYING')) for name in names):
            raise ValueError('module inventory license missing')
    if actual != expected:
        raise ValueError('module inventory membership differs from linked binary')
    compiler_notice = target / 'Go-LICENSE'
    if not compiler_notice.is_file() or not compiler_notice.read_bytes().strip():
        raise ValueError('compiler notice missing or empty')
