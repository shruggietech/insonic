#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Assemble and qualify relocated native packages without inference or release."""
import argparse
from copy import deepcopy
import hashlib
import importlib.util
import json
import os
from pathlib import Path, PurePosixPath
import platform
import plistlib
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.request
import zipfile

sys.path.insert(0, str(Path(__file__).resolve().parent))
from process_tree import ProcessTree
import qualify

ROOT = Path(__file__).resolve().parent.parent
BUILD = ROOT / 'build/packages'
VERSION = json.loads((ROOT / 'package.json').read_text(encoding='utf-8'))['version']
LADYBUG_LICENSE_URL = 'https://raw.githubusercontent.com/LadybugDB/ladybug/v0.21.2/LICENSE'
LADYBUG_LICENSE_SHA = '1c495c9546d0de02e83c9d50d5f7eb21f0085bc8f77a0ee333081a123a9c8d0c'
WINDOWS_FFMPEG_SOURCE = ('e38092ef9395d7049f871ef4d5411eb410e283e0',
                         '7578feb5284b22e6172c6f4020ddc43d746eb3ca44363756d1b47a96fd2d68f5')


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
        for support in tool.get('support_files', []):
            support['path'] = 'companions/media/' + Path(support['path']).relative_to(media_root).as_posix()
        tool.pop('interpreter', None)
    name = 'cueson.exe' if platform.system() == 'Windows' else 'cueson'
    cueson = next((ROOT / 'build/native/cueson').rglob(name))
    return {'kind': 'installed-companions', 'schema_version': VERSION, 'platform': platform_key(),
            'media': portable, 'cueson': {'executable': 'companions/cueson/' + name,
                                         'executable_sha256': sha(cueson)},
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


def installers(root, key):
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
        'This package has no signing/notarization claim and is not a promoted release.\n'
        'See package-inventory.json for exact included bytes, notices and distribution status.\n',
        encoding='utf-8', newline='\n')


def inventory(root, revision, dependencies, distribution):
    files = []
    for path in sorted(root.rglob('*')):
        if path.is_symlink():
            raise ValueError('package output contains a link')
        if path.is_file() and path != root / 'package-inventory.json':
            files.append({'path': path.relative_to(root).as_posix(), 'size_bytes': path.stat().st_size,
                          'sha256': sha(path), 'executable': bool(path.stat().st_mode & 0o111)})
    return {'kind': 'native-package-inventory', 'schema_version': VERSION, 'revision': revision,
            'platform': platform_key(), 'files': files, 'dependencies': dependencies,
            'distribution': distribution, 'inference': 'not-run', 'model_weights': 'not-included'}


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
    required = ['insonic-companions.json', 'LICENSE', 'NOTICE', 'help/index.html', 'INSTALL.txt']
    executable = '.exe' if value['platform'].startswith('windows') else ''
    required += ['insonic' + executable, 'insonic-desktop' + executable,
                 'companions/cueson/cueson' + executable, 'companions/cueson/cueson.schema.json']
    if any(name not in expected for name in required):
        raise ValueError('package lacks required product/help/companion bytes')


def build():
    key = platform_key()
    BUILD.mkdir(parents=True, exist_ok=True)
    target = BUILD / ('insonic_' + VERSION + '_' + key)
    if target.exists():
        if not target.resolve().is_relative_to(BUILD.resolve()):
            raise ValueError('package target escapes build directory')
        shutil.rmtree(target)
    root = target
    if key.startswith('darwin'):
        root = target / 'insonic.app/Contents/MacOS'
        root.mkdir(parents=True)
        with (root.parent / 'Info.plist').open('wb') as output:
            plistlib.dump({'CFBundleName': 'insonic', 'CFBundleDisplayName': 'insonic',
                          'CFBundleExecutable': 'insonic-desktop', 'CFBundleIdentifier': 'tech.shruggie.insonic',
                          'CFBundleVersion': VERSION, 'CFBundleShortVersionString': VERSION,
                          'CFBundlePackageType': 'APPL', 'LSMinimumSystemVersion': '13.3',
                          'NSHighResolutionCapable': True}, output)
    else:
        root.mkdir(parents=True)
    tags = 'desktop,production' + (',webkit2_41' if key.startswith('linux') else '')
    env = qualify.native_env()
    exe = '.exe' if key.startswith('windows') else ''
    child(['go', 'build', '-o', root / ('insonic' + exe), './cmd/insonic'], env=env)
    args = ['go', 'build', '-tags', tags]
    if key.startswith('windows'):
        args.extend(['-ldflags', '-H windowsgui'])
    child([*args, '-o', root / ('insonic-desktop' + exe), './cmd/insonic-desktop'], env=env)
    copy_tree(ROOT / 'build/native/media', root / 'companions/media')
    cueson_root = next((ROOT / 'build/native/cueson').rglob('cueson' + exe)).parent
    copy_tree(cueson_root, root / 'companions/cueson')
    (root / 'native').mkdir()
    for path in (ROOT / 'build/native/ladybug').rglob('*'):
        if path.is_file() and path.suffix.lower() in ['.dll', '.so', '.dylib'] or path.is_file() and '.so.' in path.name:
            shutil.copyfile(path, root / 'native' / path.name)
    copy_tree(ROOT / 'site/offline', root / 'help')
    for name in ['LICENSE', 'NOTICE']:
        shutil.copyfile(ROOT / name, root / name)
    notices = root / 'notices'
    notices.mkdir()
    for name in ['LICENSE', 'NOTICE', 'LICENSE-BRAND.md']:
        shutil.copyfile(ROOT / 'brand/kit' / name, notices / ('brand-' + name))
    copy_tree(ROOT / 'brand/kit/fonts/licenses', notices / 'fonts')
    shutil.copyfile(fetch_license(), notices / 'LadybugDB-LICENSE')
    module_notices = collect_module_notices(notices)
    frontend_notices = collect_frontend_notices(notices)
    source_media = source_module()
    commit, digest = WINDOWS_FFMPEG_SOURCE if key.startswith('windows') else (source_media.COMMIT, source_media.SOURCE_SHA256)
    source = source_media.fetch_source(commit, digest)
    source_directory = root / 'sources'
    source_directory.mkdir()
    shutil.copyfile(source, source_directory / source.name)
    if key.startswith('darwin'):
        _, source_receipt, _ = source_media.prepare()
        write_json(source_directory / 'ffmpeg-build.json', source_receipt)
        distribution = {'binary_publication': 'not-promoted', 'corresponding_source': 'included',
                        'codec_libraries': 'built-in LGPL source; OS VideoToolbox', 'source_review_required_before_release': True}
    else:
        source_receipt = {'source_commit': commit, 'source_sha256': digest,
                          'upstream_distribution': 'eugeneware/ffmpeg-static:b6.1.1',
                          'build_information': 'companions/media/notices/*README',
                          'external_library_corresponding_source': 'release distribution work remains'}
        write_json(source_directory / 'ffmpeg-build.json', source_receipt)
        distribution = {'binary_publication': 'blocked', 'corresponding_source': 'FFmpeg core included; static external-library sources incomplete',
                        'public_binary_archive_upload': False}
    installers(root, key)
    manifest = installed_manifest(root)
    write_json(root / 'insonic-companions.json', manifest)
    revision = child(['git', 'rev-parse', 'HEAD']).decode().strip()
    source_dirty = bool(child(['git', 'status', '--porcelain', '--untracked-files=normal']).strip())
    dependencies = {'go': qualify.LOCK['go'], 'wails': qualify.LOCK['wails'],
                    'ladybug': qualify.LOCK['ladybug']['version'], 'ladybug_binding': qualify.LOCK['ladybug']['binding'],
                    'cueson': qualify.LOCK['cueson']['version'], 'cueson_schema_sha256': qualify.LOCK['cueson']['schema_sha256'],
                    'media': json.loads((ROOT / 'build/native/media-tool-receipt.json').read_text()),
                    'modules': module_notices, 'javascript': frontend_notices}
    value = inventory(root, revision, dependencies, distribution)
    value['source_dirty'] = source_dirty
    if root != target:
        value['bundle_files'] = [{'path': path.relative_to(target).as_posix(), 'size_bytes': path.stat().st_size,
                                  'sha256': sha(path)} for path in sorted(target.rglob('*'))
                                 if path.is_file() and not path.is_relative_to(root)]
    write_json(root / 'package-inventory.json', value)
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
    receipt = {'kind': 'native-package-receipt', 'schema_version': VERSION, 'platform': key,
               'revision': revision, 'archive': archive.name, 'archive_sha256': sha(archive),
               'source_dirty': source_dirty,
               'archive_size_bytes': archive.stat().st_size, 'inventory_sha256': sha(root / 'package-inventory.json'),
               'file_count': len(value['files']), 'distribution': distribution, 'inference': 'not-run'}
    write_json(ROOT / 'build/native/package-receipt.json', receipt)
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


def smoke(archive=None, dirname=None, receipt=None):
    if archive is None:
        receipt = json.loads((ROOT / 'build/native/package-receipt.json').read_text())
        archive = BUILD / receipt['archive']
        dirname = 'insonic_' + VERSION + '_' + receipt['platform']
    if sha(archive) != receipt['archive_sha256']:
        raise ValueError('package archive differs from qualified identity')
    started = time.monotonic()
    with tempfile.TemporaryDirectory(prefix='insonic package relocation ') as temporary:
        location = Path(temporary) / 'extracted with spaces'
        extract_package(archive, location)
        root = location / dirname
        if receipt['platform'].startswith('darwin'):
            root /= 'insonic.app/Contents/MacOS'
        value = json.loads((root / 'package-inventory.json').read_text())
        verify_inventory(root, value)
        for entry in value.get('bundle_files', []):
            bundle_file = path_inside(location / dirname, entry['path'])
            if not bundle_file.is_file() or bundle_file.stat().st_size != entry['size_bytes'] or sha(bundle_file) != entry['sha256']:
                raise ValueError('macOS app bundle inventory mismatch')
        if sha(root / 'package-inventory.json') != receipt['inventory_sha256']:
            raise ValueError('extracted inventory identity differs')
        env = package_environment()
        env['INSONIC_DESKTOP_FIXTURE_DIRECTORY'] = str(ROOT / 'tests/fixtures/media')
        suffix = '.exe' if os.name == 'nt' else ''
        cli = root / ('insonic' + suffix)
        gui = root / ('insonic-desktop' + suffix)
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
                {'source': str(media / 'sintel-dialogue.mkv'), 'subtitle': str(media / 'sintel-dialogue.srt')}]})
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
            qualification_output = child([gui, '--qualification'], directory=root, env=env, timeout=30)
            qualify.write_receipt(ROOT / 'build/native/package-desktop-receipt.json', qualification_output,
                                  {'desktop_bridge': 'passed', 'offline_help': 'packaged', 'schema_version': VERSION})
            args = [gui, '--webview-qualification']
            if platform.system() == 'Linux':
                args = ['/usr/bin/xvfb-run', '-a', *args]
            qualification_output = child(args, directory=root, env=env, timeout=90)
            qualify.write_receipt(ROOT / 'build/native/package-webview-receipt.json', qualification_output,
                                  {'frontend_bridge_ipc': 'passed', 'native_webview': 'passed', 'schema_version': VERSION})
        finally:
            # The detached runtime owns its jobs and releases its own scratch
            # catalog handles at the existing bounded idle exit.
            time.sleep(31)
    receipt.update({'extracted_inventory': 'passed', 'relocation_path_spaces': 'passed',
                    'cli_real_audio_video_import': 'passed', 'cueson_assembly': 'passed', 'native_subtitle_export': 'passed',
                    'gui_bridge': 'passed', 'native_webview': 'passed', 'development_paths': 'not-required',
                    'inference': 'not-run', 'elapsed_seconds': round(time.monotonic() - started, 3)})
    write_json(ROOT / 'build/native/package-receipt.json', receipt)
    print(json.dumps(receipt))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('stage', choices=['build', 'smoke', 'all'])
    args = parser.parse_args()
    if args.stage in ['build', 'all']:
        archive, dirname, receipt = build()
    if args.stage == 'smoke':
        smoke()
    elif args.stage == 'all':
        smoke(archive, dirname, receipt)
