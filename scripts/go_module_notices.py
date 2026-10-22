# SPDX-License-Identifier: Apache-2.0
"""Notice evidence for the module union linked into selected product binaries."""
import base64
import hashlib
import json
from pathlib import Path
import re
import shutil

ROOT = Path(__file__).resolve().parents[1]
PATCH_MODULE = 'github.com/wailsapp/go-webview2'
PATCH_VERSION = 'v1.0.23'
PATCH_DIRECTORY = 'third_party/go-webview2'
PATCH_SOURCE = 'repository-local:' + PATCH_DIRECTORY
PATCH_POLICIES = {
    PATCH_MODULE: {
        'version': PATCH_VERSION, 'directory': PATCH_DIRECTORY,
        'modified': ['pkg/combridge/iunknown.go', 'pkg/edge/ICoreWebView2Controller.go', 'pkg/edge/ICoreWebView2Deferral.go', 'pkg/edge/ICoreWebView2WebResourceRequestedEventArgs.go', 'pkg/edge/ICoreWebView2WebResourceResponse.go', 'pkg/edge/chromium.go', 'pkg/edge/corewebview2.go', 'pkg/edge/create_env_go.go'],
        'added': ['LICENSE.insonic', 'NOTICE.insonic', 'pkg/edge/browser_bootstrap_windows.go', 'pkg/edge/browser_bootstrap_windows_test.go', 'pkg/edge/callback_bridge_windows.go', 'pkg/edge/callback_bridge_windows_test.go', 'pkg/edge/callback_loader_go_windows_test.go', 'pkg/edge/startup_windows.go', 'pkg/edge/startup_windows_test.go', 'pkg/edge/window_lifetime_windows.go', 'pkg/edge/window_lifetime_windows_test.go'],
    },
    'github.com/wailsapp/wails/v2': {
        'version': 'v2.16.0', 'directory': 'third_party/wails',
        'modified': ['internal/app/app_dev.go', 'internal/app/app_production.go', 'internal/frontend/desktop/windows/frontend.go'],
        'added': ['LICENSE.insonic', 'NOTICE.insonic', 'internal/app/app_lifecycle_test.go'],
    },
}
PATCH_SOURCES = {'repository-local:' + policy['directory'] for policy in PATCH_POLICIES.values()}


def repository_patch(value):
    module = value.get('Path')
    policy = PATCH_POLICIES.get(module) if isinstance(module, str) else None
    replacement = value.get('Replace')
    return (policy is not None and value.get('Version') == policy['version']
            and isinstance(replacement, dict) and replacement.get('Path') == './' + policy['directory']
            and replacement.get('Version') in (None, '', '(devel)') and not replacement.get('Sum'))


def repository_patch_source(module=PATCH_MODULE):
    if module not in PATCH_POLICIES:
        raise ValueError('unapproved repository module')
    policy = PATCH_POLICIES[module]
    version, relative = policy['version'], policy['directory']
    manifest = ROOT / (relative + '.source.json')
    if manifest.is_symlink() or manifest.is_junction():
        raise ValueError('repository module authority is linked')
    authority = json.loads(manifest.read_text(encoding='utf-8'))
    if (authority.get('module'), authority.get('version'), authority.get('directory')) != (module, version, relative):
        raise ValueError('repository module authority differs')
    def tree_hash(files):
        if not isinstance(files, dict) or not files:
            raise ValueError('repository module file authority is invalid')
        summary = hashlib.sha256()
        for name, digest in sorted(files.items()):
            if (not isinstance(name, str) or not re.fullmatch(r'[A-Za-z0-9._/+-]+', name)
                    or any(part in ('', '.', '..') for part in name.split('/'))
                    or not isinstance(digest, str) or not re.fullmatch(r'[a-f0-9]{64}', digest)):
                raise ValueError('repository module file identity is invalid')
            summary.update((digest + '  ' + module + '@' + version + '/' + name + '\n').encode('utf-8'))
        return 'h1:' + base64.b64encode(summary.digest()).decode('ascii')
    upstream, current = authority.get('upstream_files'), authority.get('files')
    if tree_hash(upstream) != authority.get('upstream_sum') or tree_hash(current) != authority.get('patched_sum'):
        raise ValueError('repository module tree checksum differs')
    # Go's goModSum uses a single summary entry named go.mod. Its content hash
    # is already bound by the verified pristine archive tree above.
    if 'go.mod' not in upstream:
        raise ValueError('repository module upstream go.mod identity is missing')
    go_mod_sum = 'h1:' + base64.b64encode(hashlib.sha256((upstream['go.mod'] + '  go.mod\n').encode('utf-8')).digest()).decode('ascii')
    sums = (ROOT / 'go.sum').read_text(encoding='utf-8').splitlines()
    for selected_version, checksum in [(version, authority['upstream_sum']), (version + '/go.mod', go_mod_sum)]:
        prefix = module + ' ' + selected_version + ' '
        if [line for line in sums if line.startswith(prefix)] != [prefix + checksum]:
            raise ValueError('repository module upstream differs from go.sum: ' + module + ' ' + selected_version)
    modified = sorted(name for name in upstream if current.get(name) != upstream[name])
    added = sorted(set(current) - set(upstream))
    if (not set(upstream) <= set(current) or modified != policy['modified'] or added != policy['added']
            or sorted(authority.get('modified', [])) != modified or sorted(authority.get('added', [])) != added):
        raise ValueError('repository module patch scope differs')
    directory = ROOT / relative
    if directory.is_symlink() or directory.is_junction() or directory.resolve() != ROOT.resolve() / relative:
        raise ValueError('repository module source directory differs')
    return directory, (module, version, authority['patched_sum'])


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
    if repository_patch(value):
        # Go records local replacements without an upstream Sum. These bytes
        # are main-revision source, verified against their own manifest below.
        module = value['Path']
        policy = PATCH_POLICIES[module]
        return module, policy['version'], (module, policy['version'], 'repository-local:' + policy['directory'])
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


def verified_source_notices(directory, identity, notice_paths):
    # Implement Go's documented h1 directory format for this selected source
    # only. Compare with the binary's Sum, never a potentially altered ziphash.
    # https://go.dev/ref/mod#go-mod-verify
    pending, files = [directory], []
    while pending:
        current = pending.pop()
        for path in current.iterdir():
            if path.is_symlink() or path.is_junction():
                raise ValueError('module source bytes contain a linked entry')
            if path.is_dir():
                pending.append(path)
            elif path.is_file():
                relative = path.relative_to(directory).as_posix()
                name = identity[0] + '@' + identity[1] + '/' + relative
                if '\n' in name:
                    raise ValueError('module source bytes contain an invalid filename')
                files.append((name, path))
            else:
                raise ValueError('module source bytes contain an unsupported entry')
    snapshot, summary = {}, hashlib.sha256()
    selected = set(notice_paths)
    for name, path in sorted(files):
        if path in selected:
            data = path.read_bytes()
            snapshot[path.name] = data
            digest = hashlib.sha256(data).hexdigest()
        else:
            with path.open('rb') as source:
                digest = hashlib.file_digest(source, 'sha256').hexdigest()
        summary.update((digest + '  ' + name + '\n').encode('utf-8'))
    actual = 'h1:' + base64.b64encode(summary.digest()).decode('ascii')
    if actual != identity[2]:
        raise ValueError('module source bytes differ from linked checksum: ' + identity[0])
    return snapshot


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
        if repository_patch(value):
            expected, verified_identity = repository_patch_source(value['Path'])
            expected = expected.resolve()
        else:
            expected = cache / (escaped(effective[0]) + '@' + escaped(effective[1]))
            verified_identity = effective
        if not directory.is_dir() or directory.resolve() != expected:
            raise ValueError('module source directory does not match selected source identity: ' + identity[0])
        paths = source_notices(directory)
        snapshot = verified_source_notices(directory, verified_identity, paths)
        if effective[0] in resolved and resolved[effective[0]] != (directory, snapshot):
            raise ValueError('conflicting resolved source directory: ' + effective[0])
        resolved[effective[0]] = directory, snapshot
    if seen != set(logical) or set(resolved) != set(sources):
        raise ValueError('resolved module membership differs from linked binary')
    modules = []
    for path, version, _ in sorted(sources.values()):
        references = []
        for name, data in sorted(resolved[path][1].items()):
            relative = notice_prefix(path, version) / name
            destination = target / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(data)
            if destination.read_bytes() != data:
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
        if sources[identity[0]][2] in PATCH_SOURCES and not {'LICENSE.insonic', 'NOTICE.insonic'} <= set(names):
            raise ValueError('repository module patch notices are missing')
    if actual != expected:
        raise ValueError('module inventory membership differs from linked binary')
    compiler_notice = target / 'Go-LICENSE'
    if not compiler_notice.is_file() or not compiler_notice.read_bytes().strip():
        raise ValueError('compiler notice missing or empty')
