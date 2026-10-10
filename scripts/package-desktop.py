#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Assemble and qualify relocated native packages without inference or release."""
import argparse
from contextlib import contextmanager
from copy import deepcopy
import hashlib
import importlib.util
import json
import os
from pathlib import Path, PurePosixPath
import platform
import plistlib
import re
import shutil
import shlex
import stat
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.request
import uuid
import zipfile

sys.path.insert(0, str(Path(__file__).resolve().parent))
from process_tree import ProcessTree
import qualify

ROOT = Path(__file__).resolve().parent.parent
BUILD = ROOT / 'build/packages'
VERSION = json.loads((ROOT / 'package.json').read_text(encoding='utf-8'))['version']
LADYBUG_LICENSE_URL = 'https://raw.githubusercontent.com/LadybugDB/ladybug/v0.21.2/LICENSE'
LADYBUG_LICENSE_SHA = '1c495c9546d0de02e83c9d50d5f7eb21f0085bc8f77a0ee333081a123a9c8d0c'


def sha(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + '\n', encoding='utf-8', newline='\n')


def platform_key():
    os_name = {'Windows': 'windows', 'Linux': 'linux', 'Darwin': 'darwin'}[platform.system()]
    arch = {'AMD64': 'amd64', 'x86_64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}[platform.machine()]
    key = os_name + '_' + arch
    if key not in ['windows_amd64', 'linux_amd64', 'darwin_arm64']:
        raise ValueError('native package architecture is not qualified')
    return key


def path_inside(root, name):
    relative = PurePosixPath(name)
    if not name or relative.is_absolute() or '..' in relative.parts or ':' in name or '\\' in name:
        raise ValueError('package path escapes installation')
    target = root.joinpath(*relative.parts)
    if not target.resolve().is_relative_to(root.resolve()):
        raise ValueError('package path escapes installation')
    return target


def child(args, *, directory=ROOT, env=None, timeout=180, allow_detached=False):
    selected = os.environ.copy()
    if env:
        selected.update(env)
    with ProcessTree([str(arg) for arg in args], cwd=directory, env=selected, allow_detached=allow_detached) as tree:
        try:
            output, error = tree.process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            tree.kill()
            tree.process.communicate(timeout=5)
            raise
        if tree.process.returncode:
            # Package fixtures carry controlled input and no user credentials.
            raise RuntimeError('package child failed: ' + (output + error).decode(errors='replace')[-10000:])
        return output


def source_module():
    spec = importlib.util.spec_from_file_location('source_media', ROOT / 'scripts/build-media-source.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def package_build_environment(values, key, native):
    selected = dict(values)
    flags = ['-L' + native.as_posix()]
    if key.startswith('linux'):
        flags.append('-Wl,-rpath,$ORIGIN/native')
    elif key.startswith('darwin'):
        flags.append('-Wl,-rpath,@loader_path/native')
    selected['CGO_LDFLAGS'] = shlex.join(flags)
    return selected


def normalize_macos_libraries(native):
    libraries = list(native.glob('*.dylib'))
    names = {path.name for path in libraries}
    for library in libraries:
        child(['/usr/bin/install_name_tool', '-id', '@rpath/' + library.name, library])
        dependencies = child(['/usr/bin/otool', '-L', library]).decode().splitlines()[1:]
        for line in dependencies:
            dependency = line.strip().split(' (compatibility version')[0]
            name = Path(dependency).name
            if name in names and dependency != '@rpath/' + name:
                child(['/usr/bin/install_name_tool', '-change', dependency, '@rpath/' + name, library])
        # Install-name edits invalidate existing signatures. Restore local
        # Mach-O integrity with an ad-hoc signature, without a release identity.
        child(['/usr/bin/codesign', '--force', '--sign', '-', library])
        child(['/usr/bin/codesign', '--verify', '--strict', library])


def verify_loader_paths(root, key, variant='desktop'):
    if key.startswith('windows'):
        if not (root / 'lbug_shared.dll').is_file():
            raise ValueError('Windows startup graph DLL must be beside both executables')
        return
    for name in ['insonic', *(['insonic-desktop'] if variant == 'desktop' else [])]:
        executable = root / name
        if key.startswith('linux'):
            output = child(['/usr/bin/readelf', '-d', executable]).decode()
            paths = [line.split('[', 1)[1].split(']', 1)[0]
                     for line in output.splitlines() if '(RPATH)' in line or '(RUNPATH)' in line]
            if paths != ['$ORIGIN/native']:
                raise ValueError('packaged executable lacks its sole installation-relative ELF loader path')
        else:
            output = child(['/usr/bin/otool', '-l', executable]).decode().splitlines()
            paths = [output[index + 2].strip().split(' ', 2)[1]
                     for index, line in enumerate(output) if line.strip() == 'cmd LC_RPATH']
            if paths != ['@loader_path/native']:
                raise ValueError('packaged executable lacks its sole installation-relative Mach-O loader path')
            dependencies = child(['/usr/bin/otool', '-L', executable]).decode().splitlines()[1:]
            if any(str(ROOT) in line or '/build/native/' in line for line in dependencies):
                raise ValueError('packaged executable retains a development library install name')


@contextmanager
def isolated_development_libraries():
    native = (ROOT / 'build/native').resolve()
    original = native / 'ladybug'
    backup = native / ('ladybug-isolated-' + uuid.uuid4().hex)
    if (not native.is_relative_to(ROOT.resolve()) or not native.is_relative_to((ROOT / 'build').resolve())
            or not original.is_dir() or original.is_symlink() or backup.exists()
            or not original.resolve().is_relative_to(native) or not backup.resolve().is_relative_to(native)):
        raise ValueError('development library isolation target is invalid')
    original.rename(backup)
    try:
        if original.exists():
            raise ValueError('development native libraries remain available during extracted smoke')
        yield
    finally:
        if original.exists():
            raise ValueError('development native library path was recreated; preserved isolated tree at ' + str(backup))
        backup.rename(original)


def fetch_license():
    target = ROOT / 'build/package-pins/ladybug-LICENSE'
    target.parent.mkdir(parents=True, exist_ok=True)
    if not target.is_file() or sha(target) != LADYBUG_LICENSE_SHA:
        with urllib.request.urlopen(LADYBUG_LICENSE_URL, timeout=45) as source, target.open('wb') as output:
            shutil.copyfileobj(source, output)
    if sha(target) != LADYBUG_LICENSE_SHA:
        raise ValueError('Ladybug license identity mismatch')
    return target


def copy_tree(source, destination):
    if not source.is_dir():
        raise ValueError('package dependency tree missing: ' + source.name)
    for path in source.rglob('*'):
        if path.is_symlink():
            raise ValueError('package input contains a link')
    shutil.copytree(source, destination)


def installed_manifest(root):
    tools = json.loads((ROOT / 'build/native/media-tools.json').read_text(encoding='utf-8'))
    media_root = ROOT / 'build/native/media'
    portable = deepcopy(tools)
    for name in ['exiftool', 'ffprobe', 'ffmpeg']:
        tool = portable[name]
        tool['path'] = 'companions/media/' + Path(tool['path']).relative_to(media_root).as_posix()
        if (root / tool['path']).is_file():
            tool['sha256'] = sha(root / tool['path'])
        for support in tool.get('support_files', []):
            support['path'] = 'companions/media/' + Path(support['path']).relative_to(media_root).as_posix()
            if (root / support['path']).is_file():
                support['sha256'] = sha(root / support['path'])
        tool.pop('interpreter', None)
    name = 'cueson.exe' if platform.system() == 'Windows' else 'cueson'
    cueson = next((ROOT / 'build/native/cueson').rglob(name))
    return {'kind': 'installed-companions', 'schema_version': VERSION, 'platform': platform_key(),
            'media': portable, 'cueson': {'executable': 'companions/cueson/' + name,
                                         'executable_sha256': sha(root / ('companions/cueson/' + name)) if (root / ('companions/cueson/' + name)).is_file() else sha(cueson)},
            'system_perl': platform.system() != 'Windows'}


def collect_module_notices(target):
    raw = child(['go', 'list', '-m', '-json', 'all']).decode()
    decoder = json.JSONDecoder()
    modules = []
    while raw.strip():
        value, end = decoder.raw_decode(raw.lstrip())
        raw = raw.lstrip()[end:]
        if value.get('Main'):
            continue
        directory = Path(value.get('Dir', ''))
        notices = []
        if directory.is_dir():
            for path in sorted(directory.iterdir()):
                if path.is_file() and path.name.upper().startswith(('LICENSE', 'LICENCE', 'COPYING', 'NOTICE')):
                    relative = Path('go') / value['Path'].replace('/', '_') / path.name
                    (target / relative).parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(path, target / relative)
                    notices.append(relative.as_posix())
        if not notices:
            raise ValueError('module license missing: ' + value['Path'])
        modules.append({'module': value['Path'], 'version': value.get('Version'), 'notices': notices})
    goroot = Path(child(['go', 'env', 'GOROOT']).decode().strip())
    shutil.copyfile(goroot / 'LICENSE', target / 'Go-LICENSE')
    return modules


def collect_frontend_notices(target):
    # The desktop bundle and offline docs contain JavaScript dependencies in
    # addition to Go modules. Preserve runtime dependency closures, including
    # Mermaid's bundled helpers and esbuild's emitted JavaScript helpers.
    modules = []
    for project in [ROOT / 'desktop/frontend', ROOT / 'site']:
        project_package = json.loads((project / 'package.json').read_text(encoding='utf-8'))
        names = project_package.get('dependencies', {})
        if project.name == 'site':
            # Markdown/highlighting and Next's server compiler dependencies
            # produce static HTML; the package ships browser runtime bytes.
            names = ['react', 'react-dom', 'mermaid', 'next']
        queue = [(project, name) for name in names]
        queue.append((project, 'esbuild'))
        visited = set()
        while queue:
            caller, name = queue.pop()
            cursor = caller
            directory = None
            while cursor.is_relative_to(project):
                candidate = cursor / 'node_modules' / name
                if (candidate / 'package.json').is_file():
                    directory = candidate
                    break
                if cursor == project:
                    break
                cursor = cursor.parent
            if directory is None:
                raise ValueError('frontend runtime dependency unavailable: ' + name)
            identity = directory.resolve()
            if identity in visited:
                continue
            visited.add(identity)
            package = json.loads((directory / 'package.json').read_text(encoding='utf-8'))
            prefix = Path('javascript') / project_package['name'] / (name.replace('/', '_') + '@' + package['version'])
            selected = []
            for notice in sorted(directory.iterdir()):
                if notice.is_file() and notice.name.upper().startswith(('LICENSE', 'LICENCE', 'COPYING', 'NOTICE')):
                    (target / prefix).mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(notice, target / prefix / notice.name)
                    selected.append((prefix / notice.name).as_posix())
            if not selected:
                # Next.js's client-only marker publishes an empty browser
                # entrypoint and no separate license text. Retain its exact
                # declared metadata; no server-only error code is bundled.
                if name != 'client-only' or (directory / 'index.js').read_bytes() != b'':
                    raise ValueError('frontend dependency license missing: ' + name)
                (target / prefix).mkdir(parents=True, exist_ok=True)
                shutil.copyfile(directory / 'package.json', target / prefix / 'package.json')
                selected.append((prefix / 'package.json').as_posix())
            modules.append({'package': name, 'version': package['version'], 'license': package.get('license'),
                            'component': project_package['name'], 'notices': selected})
            if name != 'next':
                queue.extend((directory, child_name) for child_name in package.get('dependencies', {}))
    return sorted(modules, key=lambda item: (item['component'], item['package'], item['version']))


def installers(root, key, variant='desktop'):
    if key.startswith('windows'):
        (root / 'install.ps1').write_text('''# SPDX-License-Identifier: Apache-2.0
param([Parameter(Mandatory=$true)][string]$Destination)
$ErrorActionPreference = 'Stop'
$selected = [IO.Path]::GetFullPath($Destination)
if (Test-Path -LiteralPath $selected) { throw 'Choose a new installation directory.' }
$null = New-Item -ItemType Directory -Path $selected
Get-ChildItem -LiteralPath $PSScriptRoot | Copy-Item -Destination $selected -Recurse
Write-Output "Installed insonic in $selected. Launch insonic-desktop.exe or use insonic.exe from this directory. WebView2 Runtime is required."
''', encoding='utf-8', newline='\n')
    elif key.startswith('linux'):
        script = root / 'install.sh'
        script.write_text('''#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
set -eu
if [ "$#" -ne 1 ]; then echo 'Usage: ./install.sh NEW_INSTALL_DIRECTORY' >&2; exit 2; fi
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
case "$1" in /*) destination=$1 ;; *) destination=$(pwd)/$1 ;; esac
if [ -e "$destination" ]; then echo 'Choose a new installation directory.' >&2; exit 2; fi
mkdir -p -- "$destination"
cp -R -- "$source_dir/." "$destination/"
launcher_path=$(printf '%s' "$destination/insonic-desktop" | sed 's/\\\\/\\\\\\\\/g; s/"/\\\\"/g')
printf '[Desktop Entry]\\nType=Application\\nName=insonic\\nExec="%s"\\nTerminal=false\\nCategories=AudioVideo;\\n' "$launcher_path" > "$destination/insonic.desktop"
printf 'Installed insonic in %s. GTK3, WebKit2GTK 4.1, GStreamer base/good/bad/libav plugins and Perl are required.\\n' "$destination"
''', encoding='utf-8', newline='\n')
        script.chmod(0o755)
    (root / 'INSTALL.txt').write_text(
        'insonic ' + VERSION + ' native qualification package\n'
        'Extract the entire tree. Keep the GUI, CLI and companion files together.\n'
        'Windows: WebView2 Runtime and the Microsoft Visual C++ runtime are OS prerequisites.\n'
        'Linux: GTK3, WebKit2GTK 4.1, /usr/bin/perl and GStreamer decoders are OS prerequisites.\n'
        'Debian/Ubuntu decoder packages: gstreamer1.0-plugins-base, gstreamer1.0-plugins-good,\n'
        'gstreamer1.0-plugins-bad (AAC/faad) and gstreamer1.0-libav (H.264).\n'
        'macOS ARM64: macOS 13.3+ and /usr/bin/perl; use the app bundle or its sibling CLI.\n'
        'Local recognition/diarization Python environments and model weights are optional separately configured inputs.\n'
        'Publisher signing/notarization outcomes are recorded in package-inventory.json.\n'
        'Package assembly does not publish or promote a release.\n'
        'See package-inventory.json for exact included bytes, notices and distribution status.\n',
        encoding='utf-8', newline='\n')
    if variant == 'cli':
        for name in ['INSTALL.txt', 'install.ps1', 'install.sh']:
            path = root / name
            if path.is_file():
                text = path.read_text(encoding='utf-8')
                text = text.replace('Launch insonic-desktop.exe or use insonic.exe from this directory. WebView2 Runtime is required.', 'Use insonic.exe from this directory.')
                text = text.replace('Keep the GUI, CLI and companion files together.', 'Keep the CLI and companion files together.')
                text = text.replace('Windows: WebView2 Runtime and the Microsoft Visual C++ runtime are OS prerequisites.', 'Windows: the Microsoft Visual C++ runtime is an OS prerequisite.')
                text = text.replace('Linux: GTK3, WebKit2GTK 4.1, /usr/bin/perl and GStreamer decoders are OS prerequisites.', 'Linux: /usr/bin/perl and the platform C/C++ runtime are OS prerequisites.')
                text = text.replace('use the app bundle or its sibling CLI.', 'use the included CLI.')
                text = text.replace('Debian/Ubuntu decoder packages: gstreamer1.0-plugins-base, gstreamer1.0-plugins-good,\n'
                                    'gstreamer1.0-plugins-bad (AAC/faad) and gstreamer1.0-libav (H.264).\n', '')
                if name == 'install.sh':
                    first = text.index('launcher_path=')
                    last = text.index("printf 'Installed", first)
                    text = text[:first] + text[last:]
                    text = text.replace('GTK3, WebKit2GTK 4.1, GStreamer base/good/bad/libav plugins and Perl are required.', 'Perl and the platform C/C++ runtime are required.')
                path.write_text(text, encoding='utf-8', newline='\n')


def inventory(root, revision, dependencies, distribution, variant='desktop', signing=None):
    files = []
    for path in sorted(root.rglob('*')):
        if path.is_symlink():
            raise ValueError('package output contains a link')
        if path.is_file() and path != root / 'package-inventory.json':
            files.append({'path': path.relative_to(root).as_posix(), 'size_bytes': path.stat().st_size,
                          'sha256': sha(path), 'executable': bool(path.stat().st_mode & 0o111)})
    return {'kind': 'native-package-inventory', 'schema_version': VERSION, 'revision': revision,
            'platform': platform_key(), 'variant': variant, 'files': files, 'dependencies': dependencies,
            'signing': signing or {'configured': False, 'verified': False, 'status': 'unconfigured', 'mechanism': 'none', 'identity': None},
            'distribution': distribution, 'inference': 'not-run', 'model_weights': 'not-included'}


def package_directories(root, key, variant):
    if key.startswith('darwin') and variant == 'desktop':
        return root / 'MacOS', root / 'Resources'
    return root, root


def arrange_macos_bundle(executable_root):
    """Keep code in MacOS and ordinary payloads in the sealed Resources tree."""
    resources = executable_root.parent / 'Resources'
    resources.mkdir()
    for path in sorted(executable_root.iterdir()):
        if path.name not in ['insonic', 'insonic-desktop', 'native']:
            path.replace(resources / path.name)
    return executable_root.parent, resources


def verify_inventory(root, value):
    expected = set()
    for entry in value['files']:
        path = path_inside(root, entry['path'])
        if entry['path'] in expected or path.is_symlink() or not path.is_file():
            raise ValueError('package inventory duplicate, link or missing file')
        expected.add(entry['path'])
        if path.stat().st_size != entry['size_bytes'] or sha(path) != entry['sha256']:
            raise ValueError('package inventory content mismatch')
    actual = {path.relative_to(root).as_posix() for path in root.rglob('*') if path.is_file() and path != root / 'package-inventory.json'}
    if actual != expected:
        raise ValueError('package has unrecorded bytes')
    executables, resources = package_directories(root, value['platform'], value.get('variant', 'desktop'))
    required = [(resources / name).relative_to(root).as_posix()
                for name in ['insonic-companions.json', 'LICENSE', 'NOTICE', 'help/index.html', 'INSTALL.txt']]
    executable = '.exe' if value['platform'].startswith('windows') else ''
    required += [(executables / ('insonic' + executable)).relative_to(root).as_posix(),
                 (resources / ('companions/cueson/cueson' + executable)).relative_to(root).as_posix(),
                 (resources / 'companions/cueson/cueson.schema.json').relative_to(root).as_posix()]
    if value.get('variant', 'desktop') == 'desktop':
        required.append((executables / ('insonic-desktop' + executable)).relative_to(root).as_posix())
    if any(name not in expected for name in required):
        raise ValueError('package lacks required product/help/companion bytes')


def media_runtime_permissions(receipt):
    """Distinguish required library sources from permitted compiler/system runtimes."""
    windows = receipt['platform'].startswith('windows')
    return {'compiler': receipt['compiler'], 'bundled_library_source_count': len(receipt['sources']),
            'compiler_runtime_notices': receipt.get('runtime_notices', []),
            'compiler_runtime_basis': (
                {'gcc_libgcc_libstdcpp': 'GCC Runtime Library Exception 3.1 for eligible compilation',
                 'mingw_crt_winpthreads': 'Permissive runtime licenses, exact copyright and permission notices included'}
                if windows else {}),
            'system_runtime_basis': 'Operating-system libraries/frameworks are not bundled; corresponding-source system-library exception',
            'source_complete_scope': 'FFmpeg and every bundled third-party media library requiring corresponding source'}


def collect_corresponding_sources(root, source_media, receipt, directory):
    if receipt.get('source_complete') is not True or receipt.get('recipe_sha256') != source_media.sha(ROOT / 'scripts/media_source_build.py'):
        raise ValueError('corresponding media source build receipt is incomplete or stale')
    target = root / 'sources'
    target.mkdir()
    pins = source_media.source_pins()
    if receipt.get('sources') != pins:
        raise ValueError('corresponding sources differ from built binary inputs')
    for pin in pins.values():
        shutil.copyfile(source_media.fetch_pin(pin), target / pin['name'])
    for name in ['build-media-source.py', 'media_source_build.py', 'process_tree.py']:
        shutil.copyfile(ROOT / 'scripts' / name, target / name)
    shutil.copyfile(ROOT / 'internal/qualification/media-tools.json', target / 'media-tools.json')
    if not source_media.dependency_part_receipts_valid(directory, receipt):
        raise ValueError('corresponding source dependency group provenance differs')
    for descriptor in receipt.get('dependency_parts', []):
        name = 'dependency-' + descriptor['group'] + '-receipt.json'
        shutil.copyfile(directory / name, target / name)
    write_json(target / 'ffmpeg-build.json', receipt)
    (target / 'REBUILD.txt').write_text(
        'Exact media companion corresponding sources and owned build recipe.\n'
        'Use Python 3.12+, a C/C++ compiler, make, CMake, Ninja, pkg-config, NASM\n'
        'and Meson 1.8.3. Windows uses MSYS2 UCRT64 (INSONIC_MSYS2_ROOT selects it);\n'
        'macOS ARM64 uses Xcode command-line tools with macOS 13.3+; Linux uses GCC.\n'
        'Arrange the included .py files in scripts/, media-tools.json in\n'
        'internal/qualification/, and exact source archives in build/media-source-pins/.\n'
        'Run python scripts/build-media-source.py from that root. The receipt records\n'
        'every configure/build/install invocation, compiler identity and all source hashes.\n'
        'FFmpeg VERSION is pinned to its RELEASE; no other source patches are applied.\n'
        'To modify/relink a library, edit its extracted source and rerun the recorded\n'
        'library and FFmpeg build commands, retaining the same static dependency closure.\n',
        encoding='utf-8', newline='\n')
    manifest = {'kind': 'corresponding-sources', 'schema_version': VERSION, 'source_complete': True,
                'platform': receipt['platform'], 'sources': pins,
                  'redistribution_closure': media_runtime_permissions(receipt),
                'build_receipt': 'ffmpeg-build.json', 'files': [
                    {'path': path.name, 'sha256': sha(path), 'size_bytes': path.stat().st_size}
                    for path in sorted(target.iterdir()) if path.is_file()]}
    write_json(target / 'source-manifest.json', manifest)
    verify_corresponding_sources(root, manifest)
    return manifest


def verify_corresponding_sources(root, manifest=None, expected_source_root=None):
    target = root / 'sources'
    if manifest is None:
        manifest = json.loads((target / 'source-manifest.json').read_text(encoding='utf-8'))
    if manifest.get('kind') != 'corresponding-sources' or manifest.get('source_complete') is not True:
        raise ValueError('package corresponding source manifest is incomplete')
    entries = {}
    for entry in manifest['files']:
        path = path_inside(target, entry['path'])
        if entry['path'] in entries or path.is_symlink() or not path.is_file() or sha(path) != entry['sha256'] or path.stat().st_size != entry['size_bytes']:
            raise ValueError('package corresponding source bytes differ')
        entries[entry['path']] = entry
    actual = {path.relative_to(target).as_posix() for path in target.rglob('*') if path.is_file() and path != target / 'source-manifest.json'}
    if actual != set(entries):
        raise ValueError('package corresponding sources contain unrecorded bytes')
    receipt = json.loads(path_inside(target, manifest['build_receipt']).read_text(encoding='utf-8'))
    if receipt.get('source_complete') is not True or receipt.get('sources') != manifest['sources']:
        raise ValueError('package source receipt has a different dependency closure')
    if manifest.get('redistribution_closure') != media_runtime_permissions(receipt):
        raise ValueError('package compiler/system runtime provenance differs from source receipt')
    pin = json.loads(path_inside(target, 'media-tools.json').read_text(encoding='utf-8'))['source_build']
    expected = {'ffmpeg': {'name': 'ffmpeg-' + pin['commit'] + '.tar.gz', 'url': pin['url'], 'sha256': pin['sha256'],
                           'root': 'FFmpeg-' + pin['commit'], 'version': pin['version'], 'license': 'LGPL-2.1-or-later',
                             'notices': ['COPYING.LGPLv2.1', 'LICENSE.md']}, 'lame': pin['lame'], **pin['libraries']}
    required = {'ffmpeg', 'lame', 'zlib', 'bzip2', 'xz', 'iconv', 'xml2', 'ogg', 'vorbis', 'mpg123', 'gme', 'dav1d', 'openmpt'}
    if set(expected) != required or manifest['sources'] != expected:
        raise ValueError('package corresponding sources differ from configured complete dependency closure')
    suffix = '.exe' if receipt['platform'].startswith('windows') else ''
    if (set(receipt.get('binaries', {})) != {'ffmpeg' + suffix, 'ffprobe' + suffix}
            or receipt.get('versions') != {'ffmpeg': pin['version'], 'ffprobe': pin['version']}):
        raise ValueError('package source receipt lacks exact companion binary identities')
    if suffix:
        runtime_notices = receipt.get('runtime_notices', [])
        if {Path(item['path']).name.split('-')[0] for item in runtime_notices} != {'gcc', 'libgcc', 'libstdc++', 'crt', 'winpthreads'}:
            raise ValueError('package compiler runtime permission notices are incomplete')
        for item in runtime_notices:
            path = root / 'companions/media/notices' / Path(item['path']).name
            if not path.is_file() or sha(path) != item['sha256']:
                raise ValueError('package compiler runtime permission notice bytes differ')
    for pin in manifest['sources'].values():
        if pin['name'] not in entries or entries[pin['name']]['sha256'] != pin['sha256']:
            raise ValueError('package lacks an exact corresponding library source archive')
    if any(name not in entries for name in ['build-media-source.py', 'media_source_build.py', 'process_tree.py', 'media-tools.json', 'REBUILD.txt']):
        raise ValueError('package lacks its corresponding source build recipe')
    if receipt.get('recipe_sha256') != entries['media_source_build.py']['sha256']:
        raise ValueError('package corresponding source recipe differs from executed recipe')
    if receipt.get('launcher_sha256') != entries['process_tree.py']['sha256']:
        raise ValueError('package corresponding source launcher differs from executed launcher')
    if expected_source_root is not None:
        expected_source_root = Path(expected_source_root)
        selected = json.loads((expected_source_root / 'internal/qualification/media-tools.json').read_text(encoding='utf-8'))['source_build']
        packaged = json.loads((target / 'media-tools.json').read_text(encoding='utf-8'))['source_build']
        if packaged != selected or any(sha(target / name) != sha(expected_source_root / 'scripts' / name)
                                      for name in ('build-media-source.py', 'media_source_build.py', 'process_tree.py')):
            raise ValueError('package corresponding source recipe or pins differ from selected checkout')
        if {part.get('group') for part in receipt.get('dependency_parts', [])} != set(source_module().DEPENDENCY_GROUPS):
            raise ValueError('package corresponding source provenance requires all dependency groups')
    if not source_module().dependency_part_receipts_valid(target, receipt):
        raise ValueError('package corresponding source dependency group provenance differs')
    return manifest


def sign_native(root, key):
    state = {'configured': False, 'verified': False, 'status': 'unconfigured', 'mechanism': 'none', 'identity': None}
    if key.startswith('windows'):
        identity = os.environ.get('INSONIC_WINDOWS_SIGN_CERT_SHA1', '').strip()
        if not identity:
            return state
        if not re.fullmatch(r'[0-9a-fA-F]{40}', identity):
            raise ValueError('configured Windows signing certificate thumbprint is invalid')
        tool = os.environ.get('INSONIC_SIGNTOOL_EXECUTABLE', 'signtool.exe')
        timestamp = os.environ.get('INSONIC_WINDOWS_SIGN_TIMESTAMP_URL', '').strip()
        if timestamp and not timestamp.startswith('https://'):
            raise ValueError('configured signing timestamp requires HTTPS')
        for path in sorted(root.rglob('*')):
            if path.is_file() and path.suffix.lower() in ['.exe', '.dll']:
                child([tool, 'sign', '/sha1', identity, '/fd', 'SHA256', *(['/tr', timestamp, '/td', 'SHA256'] if timestamp else []), path])
                child([tool, 'verify', '/pa', path])
        return {'configured': True, 'verified': True, 'status': 'signed', 'mechanism': 'windows-authenticode', 'identity': identity}
    if key.startswith('darwin'):
        identity = os.environ.get('INSONIC_MACOS_SIGN_IDENTITY', '').strip()
        if not identity:
            return state
        for path in sorted(root.rglob('*')):
            if not path.is_file():
                continue
            with path.open('rb') as source:
                magic = source.read(4)
            if magic in [b'\xcf\xfa\xed\xfe', b'\xfe\xed\xfa\xcf', b'\xca\xfe\xba\xbe', b'\xbe\xba\xfe\xca']:
                child(['/usr/bin/codesign', '--force', '--options', 'runtime', '--timestamp', '--sign', identity, path])
                child(['/usr/bin/codesign', '--verify', '--strict', path])
        return {'configured': True, 'verified': True, 'status': 'signed', 'mechanism': 'macos-codesign', 'identity': identity}
    return state


def seal_application(target, key, variant, signing):
    if not key.startswith('darwin'):
        return signing
    application = target / 'insonic.app' if variant == 'desktop' else target
    if variant == 'desktop':
        identity = signing['identity'] if signing['configured'] else '-'
        args = ['/usr/bin/codesign', '--force', '--sign', identity]
        if signing['configured']:
            args += ['--options', 'runtime', '--timestamp']
        child([*args, application])
        child(['/usr/bin/codesign', '--verify', '--strict', application])
    profile = os.environ.get('INSONIC_MACOS_NOTARY_PROFILE', '').strip()
    if profile:
        if not signing['configured']:
            raise ValueError('notarization profile requires configured publisher signing')
        with tempfile.TemporaryDirectory(prefix='insonic notarization ') as temporary:
            archive = Path(temporary) / 'insonic.zip'
            child(['/usr/bin/ditto', '-c', '-k', '--keepParent', application, archive])
            child(['/usr/bin/xcrun', 'notarytool', 'submit', archive, '--keychain-profile', profile, '--wait'], timeout=600)
            if variant == 'desktop':
                child(['/usr/bin/xcrun', 'stapler', 'staple', application])
                child(['/usr/bin/xcrun', 'stapler', 'validate', application])
        signing = {**signing, 'status': 'notarized', 'stapled': variant == 'desktop'}
    return signing


def build(variant='desktop'):
    if variant not in ['cli', 'desktop']:
        raise ValueError('unknown package variant')
    key = platform_key()
    BUILD.mkdir(parents=True, exist_ok=True)
    target = BUILD / ('insonic_' + VERSION + '_' + key + '_' + variant)
    if target.exists():
        if not target.resolve().is_relative_to(BUILD.resolve()):
            raise ValueError('package target escapes build directory')
        shutil.rmtree(target)
    root = target
    if key.startswith('darwin') and variant == 'desktop':
        root = target / 'insonic.app/Contents/MacOS'
        root.mkdir(parents=True)
        with (root.parent / 'Info.plist').open('wb') as output:
            plistlib.dump(qualify.desktop_application_info(), output)
    else:
        root.mkdir(parents=True)
    (root / 'native').mkdir()
    for path in (ROOT / 'build/native/ladybug').rglob('*'):
        if path.is_file() and (path.suffix.lower() in ['.dll', '.so', '.dylib'] or '.so.' in path.name):
            shutil.copyfile(path, root / 'native' / path.name)
    if key.startswith('darwin'):
        normalize_macos_libraries(root / 'native')
    tags = 'desktop,production,system_ladybug' + (',webkit2_41' if key.startswith('linux') else '')
    env = package_build_environment(qualify.native_env(), key, root / 'native')
    exe = '.exe' if key.startswith('windows') else ''
    child(['go', 'build', '-tags', 'system_ladybug', '-o', root / ('insonic' + exe), './cmd/insonic'], env=env)
    args = ['go', 'build', '-tags', tags]
    if key.startswith('windows'):
        args.extend(['-ldflags', '-H windowsgui'])
    if variant == 'desktop':
        child([*args, '-o', root / ('insonic-desktop' + exe), './cmd/insonic-desktop'], env=env)
    if key.startswith('windows'):
        # The PE loader resolves static imports before Go can configure search
        # directories. Place native DLLs beside the executables, without PATH.
        for library in (root / 'native').glob('*.dll'):
            destination = root / library.name
            if not library.resolve().is_relative_to(root.resolve()) or not destination.resolve().is_relative_to(root.resolve()):
                raise ValueError('Windows DLL placement escapes package')
            library.replace(destination)
    if key.startswith('windows'):
        from windows_openssl import prepare as prepare_openssl
        openssl = prepare_openssl()
        for name in ['libssl-3-x64.dll', 'libcrypto-3-x64.dll']:
            shutil.copyfile(openssl / name, root / name)
    verify_loader_paths(root, key, variant)
    copy_tree(ROOT / 'build/native/media', root / 'companions/media')
    cueson_root = next((ROOT / 'build/native/cueson').rglob('cueson' + exe)).parent
    copy_tree(cueson_root, root / 'companions/cueson')
    copy_tree(ROOT / 'site/offline', root / 'help')
    for name in ['LICENSE', 'NOTICE']:
        shutil.copyfile(ROOT / name, root / name)
    notices = root / 'notices'
    notices.mkdir()
    if key.startswith('windows'):
        shutil.copyfile(ROOT / 'build/native/openssl/OpenSSL-LICENSE.txt', notices / 'OpenSSL-LICENSE.txt')
    for name in ['LICENSE', 'NOTICE', 'LICENSE-BRAND.md']:
        shutil.copyfile(ROOT / 'brand/kit' / name, notices / ('brand-' + name))
    copy_tree(ROOT / 'brand/kit/fonts/licenses', notices / 'fonts')
    shutil.copyfile(fetch_license(), notices / 'LadybugDB-LICENSE')
    module_notices = collect_module_notices(notices)
    frontend_notices = collect_frontend_notices(notices)
    source_media = source_module()
    source_directory, source_receipt, _ = source_media.prepare()
    for binary, digest in source_receipt['binaries'].items():
        if sha(root / 'companions/media' / binary) != digest:
            raise ValueError('packaged media binary differs from its owned source build')
    collect_corresponding_sources(root, source_media, source_receipt, source_directory)
    installers(root, key, variant)
    resource_root = root
    if key.startswith('darwin') and variant == 'desktop':
        root, resource_root = arrange_macos_bundle(root)
    signing = sign_native(root, key)
    manifest = installed_manifest(resource_root)
    write_json(resource_root / 'insonic-companions.json', manifest)
    signing = seal_application(target, key, variant, signing)
    distribution = {'binary_publication': 'eligible', 'corresponding_source': 'included',
                    'source_complete': True, 'source_manifest_sha256': sha(resource_root / 'sources/source-manifest.json')}
    revision = child(['git', 'rev-parse', 'HEAD']).decode().strip()
    source_dirty = bool(child(['git', 'status', '--porcelain', '--untracked-files=normal']).strip())
    dependencies = {'go': qualify.LOCK['go'], 'wails': qualify.LOCK['wails'],
                    'ladybug': qualify.LOCK['ladybug']['version'], 'ladybug_binding': qualify.LOCK['ladybug']['binding'],
                    'cueson': qualify.LOCK['cueson']['version'], 'cueson_schema_sha256': qualify.LOCK['cueson']['schema_sha256'],
                    'media': json.loads((ROOT / 'build/native/media-tool-receipt.json').read_text()),
                    'modules': module_notices, 'javascript': frontend_notices}
    if key.startswith('windows'):
        dependencies['openssl'] = json.loads((ROOT / 'build/native/openssl-receipt.json').read_text())
    value = inventory(root, revision, dependencies, distribution, variant, signing)
    value['source_dirty'] = source_dirty
    if root != target:
        value['bundle_files'] = [{'path': path.relative_to(target).as_posix(), 'size_bytes': path.stat().st_size,
                                  'sha256': sha(path)} for path in sorted(target.rglob('*'))
                                 if path.is_file() and not path.is_relative_to(root)]
    inventory_path = target / 'package-inventory.json'
    write_json(inventory_path, value)
    verify_inventory(root, value)
    archive = Path(str(target) + ('.zip' if key.startswith(('windows', 'darwin')) else '.tar.gz'))
    if zipfile.is_zipfile(archive) or archive.exists():
        archive.unlink()
    if key.startswith(('windows', 'darwin')):
        with zipfile.ZipFile(archive, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=6) as output:
            for path in sorted(target.rglob('*')):
                if path.is_file():
                    output.write(path, path.relative_to(target.parent).as_posix())
    else:
        with tarfile.open(archive, 'w:gz') as output:
            output.add(target, arcname=target.name)
    receipt = {'kind': 'native-package-receipt', 'schema_version': VERSION,
               'product_version': VERSION, 'documentation_version': VERSION, 'platform': key, 'variant': variant,
               'revision': revision, 'archive': archive.name, 'archive_sha256': sha(archive),
               'dirname': target.name, 'inventory_path': inventory_path.relative_to(target.parent).as_posix(),
               'inventory_root': root.relative_to(target.parent).as_posix(), 'signing': signing,
               'source_dirty': source_dirty,
               'archive_size_bytes': archive.stat().st_size, 'inventory_sha256': sha(inventory_path),
               'file_count': len(value['files']), 'distribution': distribution, 'inference': 'not-run'}
    write_json(ROOT / 'build/native/package-receipt.json', receipt)
    write_json(ROOT / ('build/native/package-' + variant + '-receipt.json'), receipt)
    write_json(ROOT / 'build/native/package-inventory.json', value)
    return archive, target.name, receipt


def extract_package(archive, destination):
    # Verified package archives preserve executable bits, reject links/devices,
    # duplicate members and both Unix/Windows traversal before writing bytes.
    destination.mkdir(parents=True, exist_ok=True)
    seen = set()
    if zipfile.is_zipfile(archive):
        with zipfile.ZipFile(archive) as source:
            for entry in source.infolist():
                target = path_inside(destination, entry.filename)
                if entry.filename in seen or stat.S_ISLNK(entry.external_attr >> 16):
                    raise ValueError('package archive has duplicate/link member')
                seen.add(entry.filename)
                if entry.is_dir():
                    target.mkdir(parents=True, exist_ok=True)
                else:
                    target.parent.mkdir(parents=True, exist_ok=True)
                    with source.open(entry) as data, target.open('wb') as output:
                        shutil.copyfileobj(data, output)
                    if os.name != 'nt':
                        target.chmod((entry.external_attr >> 16) & 0o755)
    else:
        with tarfile.open(archive) as source:
            for entry in source:
                target = path_inside(destination, entry.name)
                if entry.name in seen or not (entry.isdir() or entry.isfile()):
                    raise ValueError('package archive has duplicate/link/device member')
                seen.add(entry.name)
                if entry.isdir():
                    target.mkdir(parents=True, exist_ok=True)
                else:
                    target.parent.mkdir(parents=True, exist_ok=True)
                    with source.extractfile(entry) as data, target.open('wb') as output:
                        shutil.copyfileobj(data, output)
                    target.chmod(entry.mode & 0o755)


def package_environment():
    env = {'CUESON_EXECUTABLE': '', 'INSONIC_NATIVE_LIBRARY': '', 'INSONIC_LIBRARY_TOOLS_FILE': '',
           'INSONIC_FIXTURE_POSTGRES': '', 'INSONIC_FIXTURE_S3': '', 'INSONIC_FIXTURE_ARCADE': '',
           'PYTHONPATH': '', 'PERL5OPT': '', 'PERL5LIB': '', 'PERLLIB': '', 'EXIFTOOL_HOME': '',
           'LD_LIBRARY_PATH': '', 'DYLD_LIBRARY_PATH': '', 'LD_PRELOAD': '', 'DYLD_INSERT_LIBRARIES': ''}
    if os.name == 'nt':
        system = os.environ.get('SystemRoot', 'C:/Windows')
        env['PATH'] = str(Path(system) / 'System32') + os.pathsep + system
    else:
        env['PATH'] = '/usr/bin:/bin:/usr/sbin:/sbin'
    return env


def verify_archive_receipt(archive, receipt, expected_source_root=ROOT):
    if sha(archive) != receipt['archive_sha256'] or archive.stat().st_size != receipt['archive_size_bytes']:
        raise ValueError('package archive differs from receipt identity')
    with tempfile.TemporaryDirectory(prefix='insonic package integrity ') as temporary:
        location = Path(temporary)
        extract_package(archive, location)
        root = path_inside(location, receipt['inventory_root'])
        inventory_file = path_inside(location, receipt['inventory_path'])
        if receipt['inventory_path'] != receipt['dirname'] + '/package-inventory.json':
            raise ValueError('package inventory path differs from declared package directory')
        if sha(inventory_file) != receipt['inventory_sha256']:
            raise ValueError('package inventory differs from receipt identity')
        value = json.loads(inventory_file.read_text(encoding='utf-8'))
        verify_inventory(root, value)
        for name in ['revision', 'platform', 'variant', 'schema_version', 'source_dirty', 'distribution', 'signing']:
            if value.get(name) != receipt.get(name):
                raise ValueError('package inventory differs from receipt field: ' + name)
        expected = {receipt['inventory_path']}
        expected.update(receipt['inventory_root'] + '/' + item['path'] for item in value['files'])
        for item in value.get('bundle_files', []):
            path = path_inside(location / receipt['dirname'], item['path'])
            if not path.is_file() or sha(path) != item['sha256'] or path.stat().st_size != item['size_bytes']:
                raise ValueError('package app bundle bytes differ from inventory')
            expected.add(receipt['dirname'] + '/' + item['path'])
        if {path.relative_to(location).as_posix() for path in location.rglob('*') if path.is_file()} != expected:
            raise ValueError('package archive contains bytes outside its complete inventory')
        _, resources = package_directories(root, receipt['platform'], receipt['variant'])
        manifest = verify_corresponding_sources(resources, expected_source_root=expected_source_root)
        if sha(resources / 'sources/source-manifest.json') != receipt['distribution']['source_manifest_sha256']:
            raise ValueError('package source manifest differs from receipt identity')
        if not receipt['signing']['configured']:
            media = json.loads((resources / 'sources/ffmpeg-build.json').read_text(encoding='utf-8'))
            for name, digest in media['binaries'].items():
                if sha(resources / 'companions/media' / name) != digest:
                    raise ValueError('unsigned packaged media differs from corresponding source build')
        return {'inventory': value, 'sources': manifest}


def _smoke(archive=None, dirname=None, receipt=None):
    if archive is None:
        receipt = json.loads((ROOT / 'build/native/package-receipt.json').read_text())
        archive = BUILD / receipt['archive']
        dirname = receipt['dirname']
    if sha(archive) != receipt['archive_sha256']:
        raise ValueError('package archive differs from qualified identity')
    verify_archive_receipt(archive, receipt)
    started = time.monotonic()
    with tempfile.TemporaryDirectory(prefix='insonic package relocation ') as temporary:
        location = Path(temporary) / 'extracted with spaces'
        extract_package(archive, location)
        root = path_inside(location, receipt['inventory_root'])
        inventory_file = path_inside(location, receipt['inventory_path'])
        value = json.loads(inventory_file.read_text())
        verify_inventory(root, value)
        for entry in value.get('bundle_files', []):
            bundle_file = path_inside(location / dirname, entry['path'])
            if not bundle_file.is_file() or bundle_file.stat().st_size != entry['size_bytes'] or sha(bundle_file) != entry['sha256']:
                raise ValueError('macOS app bundle inventory mismatch')
        if sha(inventory_file) != receipt['inventory_sha256']:
            raise ValueError('extracted inventory identity differs')
        executables, _ = package_directories(root, receipt['platform'], receipt['variant'])
        verify_loader_paths(executables, receipt['platform'], receipt['variant'])
        env = package_environment()
        env['INSONIC_DESKTOP_FIXTURE_DIRECTORY'] = str(ROOT / 'tests/fixtures/media')
        suffix = '.exe' if os.name == 'nt' else ''
        cli = executables / ('insonic' + suffix)
        gui = executables / ('insonic-desktop' + suffix)
        if child([cli, 'version'], directory=root, env=env).decode().strip() != VERSION:
            raise ValueError('packaged CLI version differs')
        workspace = Path(temporary) / 'workspace with spaces'
        child([cli, 'workspace', 'init', workspace, '--json'], directory=root, env=env)
        child([cli, '--workspace', workspace, 'credentials', 'select', 'session', '--json'], directory=root, env=env)
        def call(*args, allow_detached=False):
            response = json.loads(child([cli, '--workspace', workspace, *args, '--json'], directory=root, env=env, allow_detached=allow_detached))
            if response.get('error'):
                raise ValueError('packaged shared operation failed: ' + response['error']['code'])
            return response.get('result', response)
        media = ROOT / 'tests/fixtures/media'
        try:
            import_input = Path(temporary) / 'import fixtures.json'
            write_json(import_input, {'kind': 'import-manifest', 'schema_version': VERSION, 'items': [
                {'source': str(media / 'speech.flac'), 'subtitle': str(media / 'speech.srt')},
                {'source': str(media / 'sintel-dialogue.mkv'), 'subtitle': str(media / 'sintel.srt')}]})
            accepted = call('media', 'import', '--manifest', import_input, allow_detached=True)
            job_id = accepted['work_id']
            current = call('work', 'wait', job_id, '--timeout-ms', '30000')
            if current['state'] != 'succeeded':
                raise ValueError('packaged real media import did not succeed')
            items = current['result']['items']
            if len(items) != 2 or any(item.get('state') != 'admitted' or item.get('capture_state') not in ['captured', 'no-embedded-metadata'] for item in items):
                raise ValueError('packaged import did not complete both licensed fixtures')
            entry = call('media', 'show', items[0]['media_id'])
            tools_files = [workspace / '.insonic/media-tools.json', workspace / '.insonic/processing-tools.json']
            if any(path.exists() for path in tools_files):
                raise ValueError('package discovery persisted installation paths into workspace')
            input_path = Path(temporary) / 'assembly input.json'
            write_json(input_path, {'subtitle_format': 'srt', 'turns': []})
            assembled = call('recordings', 'assemble', items[0]['media_id'], '--input', input_path)
            if assembled.get('state') != 'ready' or not assembled.get('document_digest'):
                raise ValueError('packaged Cueson assembly did not publish a current document')
            document = call('recordings', 'document', items[0]['media_id'])
            if document.get('document_digest') != assembled['document_digest'] or not document.get('data'):
                raise ValueError('packaged Cueson document missing')
            export_path = Path(temporary) / 'exported subtitles.srt'
            write_json(input_path, {'format': 'srt', 'destination': str(export_path),
                                   'document_digest': assembled['document_digest']})
            call('recordings', 'export', items[0]['media_id'], '--input', input_path)
            if not export_path.is_file() or b'-->' not in export_path.read_bytes():
                raise ValueError('packaged native subtitle export missing')
            calendar = call('timeline', 'calendar')
            if calendar.get('total') != 2:
                raise ValueError('packaged whole-library calendar missing media')
            write_json(input_path, {'definition': {'mode': 'normalized', 'operation': 'text-search', 'pagination': {'limit': 100}}})
            queried = call('query', 'run', '--input', input_path)
            if not queried.get('items') or queried.get('graph_error'):
                raise ValueError('packaged current graph query did not use the embedded adapter')
            capabilities = call('graph', 'capabilities')
            if capabilities.get('backend_version') != qualify.LOCK['ladybug']['version']:
                raise ValueError('packaged graph version differs')
            call('graph', 'rebuild')
            roster = call('recordings', 'roster', 'clear', items[0]['media_id'], '--expected-revision', '0')
            replaced = call('media', 'import', media / 'speech.flac', '--record', items[0]['media_id'], '--replace-audio', '--existing-transcript', 'clear', '--existing-roster', 'retain')
            replacement = call('work', 'wait', replaced['work_id'], '--timeout-ms', '30000')
            retained = call('recordings', 'roster', 'show', items[0]['media_id'])
            recording = call('recordings', 'show', items[0]['media_id'])
            if replacement['state'] != 'succeeded' or recording['recording']['state'] != 'untranscribed' or retained['revision'] != roster['revision'] or retained['members']:
                raise ValueError('packaged replacement/declared-empty roster authority changed')
            if receipt['variant'] == 'desktop':
                qualification_output = child([gui, '--qualification'], directory=root, env=env, timeout=30)
                qualify.write_receipt(ROOT / 'build/native/package-desktop-bridge-receipt.json', qualification_output,
                                      {'desktop_bridge': 'passed', 'offline_help': 'packaged', 'schema_version': VERSION})
                args = [gui, '--webview-qualification']
                if platform.system() == 'Linux':
                    args = ['/usr/bin/xvfb-run', '-a', *args]
                qualification_output = child(args, directory=root, env=env, timeout=90)
                qualify.write_receipt(ROOT / 'build/native/package-webview-receipt.json', qualification_output,
                                      {'frontend_bridge_ipc': 'passed', 'native_webview': 'passed', 'ui_model_references': 'passed', 'schema_version': VERSION})
        finally:
            # The detached runtime owns its jobs and releases its own scratch
            # catalog handles at the existing bounded idle exit.
            time.sleep(31)
    receipt.update({'extracted_inventory': 'passed', 'relocation_path_spaces': 'passed',
                    'cli_real_audio_video_import': 'passed', 'recording_replacement': 'passed', 'declared_roster': 'passed', 'cueson_assembly': 'passed', 'native_subtitle_export': 'passed',
                    'gui_bridge': 'passed' if receipt['variant'] == 'desktop' else 'not-included',
                    'native_webview': 'passed' if receipt['variant'] == 'desktop' else 'not-included', 'development_paths': 'not-required',
                    'inference': 'not-run', 'elapsed_seconds': round(time.monotonic() - started, 3)})
    return receipt


def smoke(archive=None, dirname=None, receipt=None):
    with isolated_development_libraries():
        receipt = _smoke(archive, dirname, receipt)
    receipt['development_native_libraries'] = 'isolated-and-restored'
    write_json(ROOT / 'build/native/package-receipt.json', receipt)
    write_json(ROOT / ('build/native/package-' + receipt['variant'] + '-receipt.json'), receipt)
    print(json.dumps(receipt))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('stage', choices=['build', 'smoke', 'all'])
    parser.add_argument('--variant', choices=['cli', 'desktop', 'both'], default='desktop')
    args = parser.parse_args()
    for variant in (['cli', 'desktop'] if args.variant == 'both' else [args.variant]):
        if args.stage in ['build', 'all']:
            archive, dirname, receipt = build(variant)
        if args.stage == 'smoke':
            receipt = json.loads((ROOT / ('build/native/package-' + variant + '-receipt.json')).read_text())
            smoke(BUILD / receipt['archive'], receipt['dirname'], receipt)
        elif args.stage == 'all':
            smoke(archive, dirname, receipt)
