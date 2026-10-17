#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Assemble and qualify relocated native packages without inference or release."""
import argparse
from contextlib import contextmanager, ExitStack
from concurrent.futures import ThreadPoolExecutor
from copy import deepcopy
import hashlib
import importlib.util
import json
import os
from pathlib import Path, PurePosixPath
import platform
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
import go_module_notices

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
    os_name = {'Windows': 'windows', 'Linux': 'linux'}.get(platform.system())
    if os_name is None:
        raise ValueError('unsupported host platform; choose Windows or Linux x86-64')
    arch = {'AMD64': 'amd64', 'x86_64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}[platform.machine()]
    key = os_name + '_' + arch
    if key not in ['windows_amd64', 'linux_amd64']:
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
        except subprocess.TimeoutExpired as expired:
            before = tree.process.poll()
            tree.kill()
            cleanup_timeout = False
            try:
                output, error = tree.process.communicate(timeout=5)
            except subprocess.TimeoutExpired as cleanup:
                cleanup_timeout = True
                output, error = cleanup.output or expired.output or b'', cleanup.stderr or expired.stderr or b''
            # Native qualification already emits fixed fixture checkpoints. Keep
            # their bounded tail when a deadline expires, before fixture cleanup.
            sys.stdout.buffer.write(output[-65536:])
            sys.stderr.buffer.write(error[-65536:])
            try:
                write_json(ROOT / ('build/native/package-child-timeout-' + str(tree.process.pid) + '-receipt.json'), {
                    'kind': 'package-child-timeout',
                    'stage': 'native-webview' if '--webview-qualification' in args else 'package-tool',
                    'pid': tree.process.pid, 'timeout_seconds': timeout,
                    'poll_before_kill': before, 'poll_after_kill': tree.process.poll(),
                    'cleanup_timeout': cleanup_timeout,
                    'stdout_bytes': len(output), 'stderr_bytes': len(error),
                    'retained_tail_bytes_per_stream': 65536,
                })
            except OSError:
                pass  # Diagnostics cannot replace the actual child timeout.
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
    selected['CGO_LDFLAGS'] = shlex.join(flags)
    return selected




def verify_loader_paths(root, key, variant='desktop'):
    if key.startswith('windows'):
        if not (root / 'lbug_shared.dll').is_file():
            raise ValueError('Windows startup graph DLL must be beside both executables')
        return
    if not key.startswith('linux'):
        raise ValueError('unsupported package platform')
    for name in ['insonic', *(['insonic-desktop'] if variant == 'desktop' else [])]:
        output = child(['/usr/bin/readelf', '-d', root / name]).decode()
        paths = [line.split('[', 1)[1].split(']', 1)[0]
                 for line in output.splitlines() if '(RPATH)' in line or '(RUNPATH)' in line]
        if paths != ['$ORIGIN/native']:
            raise ValueError('packaged executable lacks its sole installation-relative ELF loader path')


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
            'worker': {'path': 'companions/processing/processing-worker.py',
                       'sha256': sha(root / 'companions/processing/processing-worker.py')},
            'worker_resources': [{'path': path.relative_to(root).as_posix(), 'sha256': sha(path)}
                                 for path in sorted((root / 'companions/processing').glob('*'))
                                 if path.is_file() and path.name != 'processing-worker.py'],
            'media': portable, 'cueson': {'executable': 'companions/cueson/' + name,
                                         'executable_sha256': sha(root / ('companions/cueson/' + name)) if (root / ('companions/cueson/' + name)).is_file() else sha(cueson)},
            'system_perl': platform.system() != 'Windows'}


def product_binaries(root, key, variant):
    suffix = '.exe' if key.startswith('windows') else ''
    names = ['insonic'] + (['insonic-desktop'] if variant == 'desktop' else [])
    if key.startswith('windows'):
        names.append('insonic-launcher')
    return [(root / (name + suffix), 'github.com/shruggietech/insonic/cmd/' + name) for name in names]


def collect_module_notices(target, binaries):
    return go_module_notices.collect_notices(target, binaries, child, qualify.LOCK['go'])


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


def installers(root, key, variant='desktop', signing=None):
    if key not in ('windows_amd64', 'linux_amd64') or variant not in ('cli', 'desktop'):
        raise ValueError('unsupported installation platform or variant')
    desktop = variant == 'desktop'
    label = 'Desktop + CLI' if desktop else 'CLI only'
    if key.startswith('windows'):
        helper = (ROOT / 'scripts/install-windows.ps1').read_text(encoding='utf-8')
        helper = helper.replace('InsonicInstall', 'InsonicInstall_' + hashlib.sha256(helper.encode()).hexdigest()[:16])
        instructions = ('Launch insonic-desktop.exe or run insonic.exe in an open terminal. Microsoft Visual C++ x64 and WebView2 runtimes are required.' if desktop
                        else 'Run insonic.exe in an open terminal. Microsoft Visual C++ x64 runtime is required.')
        (root / 'install.ps1').write_text(helper.replace('CLIENT_INSTRUCTIONS', instructions), encoding='utf-8', newline='\n')
        signing = signing or {'configured': False, 'verified': False}
        signature = ('Publisher signature was configured and verified for this package.' if signing.get('configured') and signing.get('verified')
                     else 'This package has no verified publisher signature (unsigned). A checksum checks bytes, not publisher identity or Windows reputation.')
        text = f'insonic {VERSION}: Windows x86-64 {label} ZIP\n\n'
        text += ('Includes application windows (insonic-desktop.exe) and terminal commands (insonic.exe).\n' if desktop
                 else 'Includes terminal commands (insonic.exe); no desktop application.\n')
        text += 'Both choices include native libraries, media companions, Cueson/schema, help, notices and corresponding sources.\n'
        text += 'Prerequisite: Microsoft Visual C++ x64 runtime. Official guidance/download:\nhttps://learn.microsoft.com/en-us/cpp/windows/latest-supported-vc-redist?view=msvc-170\nhttps://aka.ms/vc14/vc_redist.x64.exe\n'
        if desktop:
            text += 'Also install WebView2 Runtime. Use the online Evergreen Bootstrapper or the x64 offline Standalone Installer:\nhttps://developer.microsoft.com/en-us/microsoft-edge/webview2/\nhttps://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution\n'
        text += signature + '\n\n'
        text += r'''1. Download the exact official release/variant ZIP and install its prerequisites. Complete prerequisite installers can be transferred for offline use.
2. Extract the entire tree. Do not run inside the ZIP or copy only an EXE.
3. Choose a new stable program directory outside your library. Its parent must already exist. Keep all files and companions together.
4. Optionally open PowerShell in the extracted directory and run:
   .\install.ps1 -Destination "C:\YOUR_EXISTING_PROGRAM_PARENT\insonic VERSION"
   Replace the example path/version, retaining quotes. If script policy prevents the helper, copy the complete tree manually. The helper only copies: no PATH, shortcuts, MSI registration, Installed Apps entry or uninstaller.
5. Open PowerShell/Windows Terminal in the installed directory and keep it open:
   .\insonic.exe version
   .\insonic.exe --help
6. Choose a separate library folder and initialize it once:
   .\insonic.exe workspace init "C:\YOUR_LIBRARY_PARENT\insonic library"
7. Select that library explicitly on subsequent commands:
   .\insonic.exe --workspace "C:\YOUR_LIBRARY_PARENT\insonic library" doctor
   .\insonic.exe --workspace "C:\YOUR_LIBRARY_PARENT\insonic library" workspace show
'''
        if desktop:
            text += '8. Launch insonic-desktop.exe from this program directory and open the library.\n'
        text += '\nRecognition/diarization requires separately configured local environments/models or an explicitly selected compatible connected pipeline. Media companions alone do not make inference ready.\n'
        text += '\nUpdate: back up library configuration/catalog/artifacts, install the new complete version in a separate directory and retain the old version for rollback. Finish/cancel jobs, close GUI/CLI clients and allow at least 31 seconds without requests for the idle runtime to exit. Check processes belonging to the old program directory before changing its files; there is no runtime stop command. Validate version/help and --workspace "YOUR_EXISTING_LIBRARY" doctor from the replacement. Stored-contract changes can require restoring a matching backup for rollback.\n'
        text += '\nRemove: close work/clients and the idle runtime, then remove only the chosen program directory and integrations you created manually. Preserve library/media/models/credentials. A local library uses its chosen .insonic control directory; alternate profiles can store data elsewhere. Per-user configuration/data/cache/credentials are not program files.\n'
        text += '\nRead help/docs/v' + VERSION + '/desktop/index.html for the complete installation, update and removal procedure, and the storage/secrets pages for actual configured data locations.\n'
    else:
        script = root / 'install.sh'
        text_script = r'''#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
set -eu
if [ "$#" -ne 1 ]; then echo 'Usage: ./install.sh NEW_INSTALL_DIRECTORY' >&2; exit 2; fi
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
parent=$(CDPATH= cd -- "$(dirname -- "$1")" && pwd -P)
destination=$parent/$(basename -- "$1")
case "$destination/" in "$source_dir/"*) echo 'Choose a directory outside the source tree.' >&2; exit 2 ;; esac
if [ -e "$destination" ]; then echo 'Choose a new installation directory.' >&2; exit 2; fi
mkdir -- "$destination"
cp -R -- "$source_dir/." "$destination/"
'''
        if desktop:
            text_script += r'''launcher_path=$(printf '%s' "$destination/insonic-desktop" | sed 's/\\/\\\\/g; s/"/\\"/g')
printf '[Desktop Entry]\nType=Application\nName=insonic\nExec="%s"\nTerminal=false\nCategories=AudioVideo;\n' "$launcher_path" > "$destination/insonic.desktop"
'''
        requirement = ('GTK 3, WebKitGTK 4.1, GStreamer decoders and Perl' if desktop else 'Perl and the platform C/C++ runtime')
        text_script += "printf 'Installed insonic in %s. " + requirement + " are required.\\n' \"$destination\"\n"
        script.write_text(text_script, encoding='utf-8', newline='\n'); script.chmod(0o755)
        text = f'insonic {VERSION}: Linux x86-64 {label} tar.gz\n\n'
        text += ('Includes insonic-desktop application windows and insonic terminal commands.\n' if desktop else 'Includes insonic terminal commands; no desktop application or launcher.\n')
        text += 'Prerequisites: ' + requirement + '.\n'
        if desktop:
            text += 'Ubuntu decoder packages: gstreamer1.0-plugins-base, gstreamer1.0-plugins-good, gstreamer1.0-plugins-bad and gstreamer1.0-libav.\n'
        text += 'Extract the complete tree and keep native libraries, companions, help and sources together. Choose a new destination outside the source tree whose parent exists; optionally run ./install.sh NEW_DIRECTORY. The desktop helper writes an application launcher.\n'
        text += './insonic version\n./insonic --help\n./insonic workspace init "YOUR_SEPARATE_LIBRARY"\n./insonic --workspace "YOUR_SEPARATE_LIBRARY" doctor\n'
        text += 'Local inference environments/models remain separately configured. For updates keep a separate complete version directory and rollback copy; finish/cancel work and close clients before program removal. Preserve workspace/media/models/credentials. See bundled help/docs/v' + VERSION + '/desktop/index.html and storage guidance.\n'
    (root / 'INSTALL.txt').write_text(text, encoding='utf-8', newline='\n')


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
    # Retained macOS layout handling is historical archive validation only.
    if key.startswith('darwin') and variant == 'desktop':
        return root / 'MacOS', root / 'Resources'
    return root, root




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
                for name in ['insonic-companions.json', 'LICENSE', 'NOTICE', 'help/index.html', 'INSTALL.txt',
                             'companions/processing/processing-worker.py']]
    executable = '.exe' if value['platform'].startswith('windows') else ''
    required += [(executables / ('insonic' + executable)).relative_to(root).as_posix(),
                 (resources / ('companions/cueson/cueson' + executable)).relative_to(root).as_posix(),
                 (resources / 'companions/cueson/cueson.schema.json').relative_to(root).as_posix()]
    if value.get('variant', 'desktop') == 'desktop':
        required.append((executables / ('insonic-desktop' + executable)).relative_to(root).as_posix())
    metadata = value.get('dependencies', {}).get('processing_workers')
    if metadata:
        processing = resources / 'companions/processing'
        for name in ['catalog.json', 'models.json', 'probe.py']:
            required.append((processing / name).relative_to(root).as_posix())
        catalog = json.loads((processing / 'catalog.json').read_text(encoding='utf-8'))
        if sha(processing / 'catalog.json') != metadata['catalog_sha256'] or sha(processing / 'models.json') != metadata['model_catalog_sha256']:
            raise ValueError('processing acquisition metadata differs')
        declared = [{'id': item['id'], 'platform': item['platform'], 'python': item['python'],
                     'engines': item['engines'], 'lock_sha256': item['lock_sha256']}
                    for item in catalog['profiles']]
        if declared != metadata['profiles']:
            raise ValueError('processing profile closure differs')
        for profile in catalog['profiles']:
            lock = path_inside(processing, profile['lock'])
            required.append(lock.relative_to(root).as_posix())
            if sha(lock) != profile['lock_sha256']:
                raise ValueError('processing hash lock differs')
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
        'Linux uses GCC.\n'
        'Arrange the included .py files in scripts/, media-tools.json in\n'
        'internal/qualification/, and exact source archives in build/media-source-pins/.\n'
        'On Windows, run these commands in order from that same root:\n'
        'python scripts/build-media-source.py --stage dependencies\n'
        'python scripts/build-media-source.py --stage codec\n'
        'python scripts/build-media-source.py --stage complete\n'
        'Keep the configured tree and dependency prefix unchanged between stages.\n'
        'Other platforms run python scripts/build-media-source.py from that root.\n'
        'The receipt records\n'
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
    return state




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
    root.mkdir(parents=True)
    (root / 'native').mkdir()
    for path in (ROOT / 'build/native/ladybug').rglob('*'):
        if path.is_file() and (path.suffix.lower() in ['.dll', '.so', '.dylib'] or '.so.' in path.name):
            shutil.copyfile(path, root / 'native' / path.name)
    tags = 'desktop,production,system_ladybug' + (',webkit2_41' if key.startswith('linux') else '')
    env = package_build_environment(qualify.native_env(), key, root / 'native')
    exe = '.exe' if key.startswith('windows') else ''
    child(['go', 'build', '-tags', 'system_ladybug', '-o', root / ('insonic' + exe), './cmd/insonic'], env=env)
    if key.startswith('windows'):
        child(['go', 'build', '-ldflags', '-H windowsgui', '-o', root / 'insonic-launcher.exe', './cmd/insonic-launcher'], env=env)
        shutil.copyfile(ROOT / 'scripts/insonic-session.cmd', root / 'insonic-session.cmd')
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
    processing = root / 'companions/processing'
    processing.mkdir()
    shutil.copyfile(ROOT / 'scripts/processing-worker.py', processing / 'processing-worker.py')
    for source in sorted((ROOT / 'internal/workers/assets').glob('*')):
        shutil.copyfile(source, processing / source.name)
    shutil.copyfile(ROOT / 'internal/workers/probe.py', processing / 'probe.py')
    shutil.copyfile(ROOT / 'internal/models/defaults.json', processing / 'models.json')
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
    module_notices = collect_module_notices(notices, product_binaries(root, key, variant))
    frontend_notices = collect_frontend_notices(notices)
    source_media = source_module()
    source_directory, source_receipt, _ = source_media.prepare()
    for binary, digest in source_receipt['binaries'].items():
        if sha(root / 'companions/media' / binary) != digest:
            raise ValueError('packaged media binary differs from its owned source build')
    collect_corresponding_sources(root, source_media, source_receipt, source_directory)
    signing = sign_native(root, key)
    installers(root, key, variant, signing)
    resource_root = root
    manifest = installed_manifest(resource_root)
    write_json(resource_root / 'insonic-companions.json', manifest)
    distribution = {'binary_publication': 'eligible', 'corresponding_source': 'included',
                    'source_complete': True, 'source_manifest_sha256': sha(resource_root / 'sources/source-manifest.json')}
    revision = child(['git', 'rev-parse', 'HEAD']).decode().strip()
    source_dirty = bool(child(['git', 'status', '--porcelain', '--untracked-files=normal']).strip())
    dependencies = {'go': qualify.LOCK['go'], 'wails': qualify.LOCK['wails'],
                    'ladybug': qualify.LOCK['ladybug']['version'], 'ladybug_binding': qualify.LOCK['ladybug']['binding'],
                    'cueson': qualify.LOCK['cueson']['version'], 'cueson_schema_sha256': qualify.LOCK['cueson']['schema_sha256'],
                    'media': json.loads((ROOT / 'build/native/media-tool-receipt.json').read_text()),
                    'modules': module_notices, 'javascript': frontend_notices,
                    'processing_workers': {'disposition': 'optional_acquisition_metadata',
                                           'catalog_sha256': sha(processing / 'catalog.json'),
                                           'model_catalog_sha256': sha(processing / 'models.json'),
                                           'profiles': [{'id': item['id'], 'platform': item['platform'],
                                                         'python': item['python'], 'engines': item['engines'],
                                                         'lock_sha256': item['lock_sha256']}
                                                        for item in json.loads((processing / 'catalog.json').read_text(encoding='utf-8'))['profiles']]}}
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
    archive = Path(str(target) + ('.zip' if key.startswith('windows') else '.tar.gz'))
    if zipfile.is_zipfile(archive) or archive.exists():
        archive.unlink()
    if key.startswith('windows'):
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
           'LD_LIBRARY_PATH': '', 'DYLD_LIBRARY_PATH': '', 'DYLD_FALLBACK_LIBRARY_PATH': '',
           'LD_PRELOAD': '', 'DYLD_INSERT_LIBRARIES': ''}
    if os.name == 'nt':
        system = os.environ.get('SystemRoot', 'C:/Windows')
        env['PATH'] = str(Path(system) / 'System32') + os.pathsep + system
        env['PATHEXT'] = '.COM;.EXE;.BAT;.CMD'
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
        binaries, resources = package_directories(root, receipt['platform'], receipt['variant'])
        # Historical published inventories retain their original semantics.
        # Current candidate evidence must independently match its own binaries.
        if value.get('schema_version') == VERSION:
            if value.get('dependencies', {}).get('go') != qualify.LOCK['go']:
                raise ValueError('package compiler inventory differs from elected toolchain')
            go_module_notices.verify_notices(
                resources / 'notices', value.get('dependencies', {}).get('modules'),
                product_binaries(binaries, receipt['platform'], receipt['variant']), child, qualify.LOCK['go'])
        if sha(resources / 'companions/processing/processing-worker.py') != sha(expected_source_root / 'scripts/processing-worker.py'):
            raise ValueError('package processing worker differs from selected source')
        worker_assets = expected_source_root / 'internal/workers/assets'
        if worker_assets.is_dir():
            sources = [(source, source.name) for source in worker_assets.glob('*')]
            sources += [(expected_source_root / 'internal/workers/probe.py', 'probe.py'),
                        (expected_source_root / 'internal/models/defaults.json', 'models.json')]
            for source, name in sources:
                if sha(resources / 'companions/processing' / name) != sha(source):
                    raise ValueError('packaged worker metadata differs from elected source')
        manifest = verify_corresponding_sources(resources, expected_source_root=expected_source_root)
        if sha(resources / 'sources/source-manifest.json') != receipt['distribution']['source_manifest_sha256']:
            raise ValueError('package source manifest differs from receipt identity')
        if not receipt['signing']['configured']:
            media = json.loads((resources / 'sources/ffmpeg-build.json').read_text(encoding='utf-8'))
            for name, digest in media['binaries'].items():
                if sha(resources / 'companions/media' / name) != digest:
                    raise ValueError('unsigned packaged media differs from corresponding source build')
        return {'inventory': value, 'sources': manifest}


def qualify_windows_install(extracted, temporary, env):
    """Execute the documented copy and first-use commands in Windows PowerShell."""
    installed = Path(temporary) / 'installed program with spaces'
    library = Path(temporary) / 'first-use library with spaces'
    def literal(value):
        return "'" + str(value).replace("'", "''") + "'"
    powershell = Path(os.environ.get('SystemRoot', 'C:/Windows')) / 'System32/WindowsPowerShell/v1.0/powershell.exe'
    script = (
        "$ErrorActionPreference = 'Stop'; [Console]::OutputEncoding = [Text.UTF8Encoding]::new($false); "
        "& " + literal(extracted / 'install.ps1') + " -Destination " + literal(installed) + " | Out-Null; "
        "$cli = " + literal(installed / 'insonic.exe') + "; $library = " + literal(library) + "; "
        "function Invoke-FixtureCli([string]$Arguments) { "
        "$startInfo = [Diagnostics.ProcessStartInfo]::new(); $startInfo.FileName = $cli; $startInfo.Arguments = $Arguments; "
        "$startInfo.UseShellExecute = $false; $startInfo.CreateNoWindow = $true; $startInfo.WorkingDirectory = [IO.Path]::GetDirectoryName($cli); "
        "$startInfo.RedirectStandardInput = $true; $startInfo.RedirectStandardOutput = $true; $startInfo.RedirectStandardError = $true; "
        "$startInfo.StandardOutputEncoding = [Text.UTF8Encoding]::new($false); $startInfo.StandardErrorEncoding = [Text.UTF8Encoding]::new($false); "
        "$firstProcess = [Diagnostics.Process]::new(); $firstProcess.StartInfo = $startInfo; "
        "try { if (-not $firstProcess.Start()) { throw 'first-use start failed' }; $firstProcess.StandardInput.Close(); "
        "$outputTask = $firstProcess.StandardOutput.ReadToEndAsync(); $errorTask = $firstProcess.StandardError.ReadToEndAsync(); "
        "if (-not $firstProcess.WaitForExit(30000)) { $firstProcess.Kill(); throw 'first-use timeout' }; "
        "if ($firstProcess.ExitCode -ne 0) { throw ('first-use CLI failed: ' + $errorTask.Result) }; return $outputTask.Result.TrimEnd([char[]]@([char]13,[char]10)); "
        "} finally { $firstProcess.Dispose() } }; "
        "$version = Invoke-FixtureCli 'version'; $help = Invoke-FixtureCli '--help'; "
        "$init = Invoke-FixtureCli ('workspace init ' + [char]34 + $library + [char]34); "
        "$doctor = Invoke-FixtureCli ('--workspace ' + [char]34 + $library + [char]34 + ' doctor'); "
        "$show = Invoke-FixtureCli ('--workspace ' + [char]34 + $library + [char]34 + ' workspace show'); "
        "@{version=$version; help=$help; init=$init; doctor=$doctor; show=$show} | ConvertTo-Json -Compress"
    )
    output = child([powershell, '-NoProfile', '-NonInteractive', '-Command', script],
                   directory=extracted, env=env, timeout=60, allow_detached=True)
    # The helper emits one operator status line before the final JSON object.
    first_use = json.loads(output.decode('utf-8-sig').splitlines()[-1])
    if first_use.get('version') != VERSION or any(not first_use.get(key) for key in ['help', 'init', 'doctor', 'show']):
        raise ValueError('installed Windows first-use command output differs: ' + str({'version': first_use.get('version'), 'lengths': {key: len(str(first_use.get(key, ''))) for key in ['help', 'init', 'doctor', 'show']}}))
    inventory = json.loads((installed / 'package-inventory.json').read_text(encoding='utf-8'))
    verify_inventory(installed, inventory)
    if not (library / '.insonic/workspace.json').is_file():
        raise ValueError('installed Windows library initialization did not persist')
    return installed


def qualify_package_gui(gui, directory, env):
    runtime_env = dict(env)
    if os.environ.get('GITHUB_ACTIONS') == 'true':
        runtime_env['INSONIC_PROCESS_STARTUP_DIAGNOSTICS'] = '1'
    # The shared runtime can legitimately restart as its own idle-exit owner.
    # Permit its explicit Windows breakaway, retaining this GUI's owned job.
    qualification_output = child([gui, '--qualification'], directory=directory, env=runtime_env, timeout=30, allow_detached=True)
    qualify.write_receipt(ROOT / 'build/native/package-desktop-bridge-receipt.json', qualification_output,
                          {'desktop_bridge': 'passed', 'offline_help': 'packaged', 'schema_version': VERSION})
    help_args = [gui, '--help-qualification']
    if platform.system() == 'Linux':
        help_args = ['/usr/bin/xvfb-run', '-a', *help_args]
    help_output = child(help_args, directory=directory, env=runtime_env, timeout=35, allow_detached=True)
    qualify.write_receipt(ROOT / 'build/native/package-help-receipt.json', help_output,
                          {'offline_help_activation': 'passed', 'workspace': 'not-selected', 'network_policy': 'local-assets-only', 'schema_version': VERSION})
    args = [gui, '--webview-qualification']
    if platform.system() == 'Linux':
        args = ['/usr/bin/xvfb-run', '-a', *args]
    qualification_output = child(args, directory=directory, env=runtime_env, timeout=90, allow_detached=True)
    qualify.write_receipt(ROOT / 'build/native/package-webview-receipt.json', qualification_output,
                          {'frontend_bridge_ipc': 'passed', 'native_webview': 'passed', 'ui_model_references': 'passed', 'schema_version': VERSION})


@contextmanager
def background_gui_qualification(gui, directory, env):
    # Probes own their fixture workspaces. Join every probe, including any
    # bootstrap activity, before restoring libraries or removing fixtures.
    with ThreadPoolExecutor(max_workers=1) as workers:
        future = workers.submit(qualify_package_gui, gui, directory, dict(env))
        yield future
        future.result()


def windows_image_released(executable):
    # Request write-data and delete access without modifying the image.
    # Write-data access observes mapped-image sharing on Windows Server too.
    import ctypes
    from ctypes import wintypes
    kernel = ctypes.WinDLL('kernel32', use_last_error=True)
    kernel.CreateFileW.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD,
                                  ctypes.c_void_p, wintypes.DWORD, wintypes.DWORD, wintypes.HANDLE]
    kernel.CreateFileW.restype = wintypes.HANDLE
    kernel.CloseHandle.argtypes = [wintypes.HANDLE]
    kernel.CloseHandle.restype = wintypes.BOOL
    handle = kernel.CreateFileW(str(executable), 0x10002, 7, None, 3, 0, None)
    if handle == ctypes.c_void_p(-1).value:
        error = ctypes.get_last_error()
        if error in (5, 32):
            return False
        raise ctypes.WinError(error)
    if not kernel.CloseHandle(handle):
        raise ctypes.WinError(ctypes.get_last_error())
    return True


def windows_image_owners(executable):
    """Read exact-image process owners without a slow, separate CIM process."""
    import ctypes as C
    import ctypes.wintypes as W
    class Entry(C.Structure):
        _fields_=[('size',W.DWORD),('usage',W.DWORD),('pid',W.DWORD),('heap',C.c_size_t),('module',W.DWORD),('threads',W.DWORD),('parent',W.DWORD),('priority',W.LONG),('flags',W.DWORD),('image',W.WCHAR*260)]
    kernel=C.WinDLL('kernel32',use_last_error=True)
    kernel.CreateToolhelp32Snapshot.argtypes=[W.DWORD,W.DWORD];kernel.CreateToolhelp32Snapshot.restype=W.HANDLE
    for name in ('Process32FirstW','Process32NextW'):
        function=getattr(kernel,name);function.argtypes=[W.HANDLE,C.POINTER(Entry)];function.restype=W.BOOL
    kernel.OpenProcess.argtypes=[W.DWORD,W.BOOL,W.DWORD];kernel.OpenProcess.restype=W.HANDLE
    kernel.QueryFullProcessImageNameW.argtypes=[W.HANDLE,W.DWORD,W.LPWSTR,C.POINTER(W.DWORD)];kernel.QueryFullProcessImageNameW.restype=W.BOOL
    kernel.CloseHandle.argtypes=[W.HANDLE]
    snapshot=kernel.CreateToolhelp32Snapshot(2,0)
    if snapshot==C.c_void_p(-1).value:raise C.WinError(C.get_last_error())
    found=[];entry=Entry();entry.size=C.sizeof(entry);target=os.path.normcase(os.path.abspath(executable))
    try:
        present=kernel.Process32FirstW(snapshot,C.byref(entry))
        while present:
            if entry.image.casefold()==Path(executable).name.casefold():
                process=kernel.OpenProcess(0x1000,False,entry.pid)
                if process:
                    try:
                        image=C.create_unicode_buffer(32768);size=W.DWORD(32768)
                        if kernel.QueryFullProcessImageNameW(process,0,image,C.byref(size)) and os.path.normcase(os.path.abspath(image.value))==target:found.append({'pid':entry.pid,'parent_pid':entry.parent,'role':'other'})
                    finally:kernel.CloseHandle(process)
            if len(found)>=16:break
            present=kernel.Process32NextW(snapshot,C.byref(entry))
        return found
    finally:kernel.CloseHandle(snapshot)


def retain_busy_image_diagnostic(executable):
    value = {'kind': 'package-image-busy', 'image_writable': os.access(executable, os.W_OK), 'owners': []}
    try:
        owners=windows_image_owners(executable)
        value['owners']=[{'pid':int(owner['pid']),'parent_pid':int(owner['parent_pid']),
                          'role':owner['role'] if owner['role'] in ['other','cold-credentials','documentation','first-use','gui','main'] else 'other'} for owner in owners[:16]]
        value['observer']='native exact-image process snapshot'
    except (OSError, RuntimeError, ValueError, TypeError, KeyError):
        value['diagnostic']='unavailable'
    try:
        write_json(ROOT / 'build/native/package-image-busy-receipt.json', value)
    except OSError:
        pass  # Diagnostic failure cannot replace the package cleanup failure.


def wait_for_detached_idle(last_completed_at=None, executable=None):
    # Keep the full idle lease, then observe Windows image release rather than
    # assuming shutdown and native-library teardown finish on the next tick.
    remaining = 31 if last_completed_at is None else max(0, 31 - (time.monotonic() - last_completed_at))
    if remaining:
        time.sleep(remaining)
    if executable is not None and platform.system() == 'Windows':
        deadline = time.monotonic() + 10
        while not windows_image_released(executable):
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                retain_busy_image_diagnostic(executable)
                raise TimeoutError('packaged CLI image remained busy after detached idle allowance')
            time.sleep(min(0.1, remaining))


def finish_package_checks(gui_future, last_completed_at, executable):
    try:
        if gui_future is not None:
            gui_future.result()
    finally:
        # GUI bootstrap can elect another detached owner. Its final activity can
        # occur after the last CLI request, including on a failed CLI journey.
        wait_for_detached_idle(None if gui_future is not None else last_completed_at, executable)


def _smoke(archive=None, dirname=None, receipt=None):
    if archive is None:
        receipt = json.loads((ROOT / 'build/native/package-receipt.json').read_text())
        archive = BUILD / receipt['archive']
        dirname = receipt['dirname']
    if sha(archive) != receipt['archive_sha256']:
        raise ValueError('package archive differs from qualified identity')
    verify_archive_receipt(archive, receipt)
    started = time.monotonic()
    with tempfile.TemporaryDirectory(prefix='insonic package relocation ') as temporary, ExitStack() as managed_checks:
        location = Path(temporary) / 'extracted é with spaces'
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
        installation = 'not-applicable'
        if receipt['platform'] == 'windows_amd64':
            root = qualify_windows_install(root, temporary, package_environment())
            installation = 'passed'
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
        def call(*args, allow_detached=True):
            response = json.loads(child([cli, '--workspace', workspace, *args, '--json'], directory=root, env=env, allow_detached=allow_detached))
            if response.get('error'):
                raise ValueError('packaged shared operation failed: ' + response['error']['code'])
            return response.get('result', response)
        media = ROOT / 'tests/fixtures/media'
        idle_started = None
        gui_future = None
        try:
            if receipt['variant'] == 'desktop':
                gui_future = managed_checks.enter_context(background_gui_qualification(gui, root, env))
            setup = call('setup', 'show')
            if setup['revision'] != 0 or setup['state']['choices']['processing_mode'] != 'none':
                raise ValueError('packaged fresh setup defaults differ')
            setup_input = Path(temporary) / 'optional setup.json'
            setup_state = setup['state']
            setup_state['status'] = 'deferred'
            for stage in setup_state['stages']:
                if stage['id'] in ['processing', 'services', 'import']:
                    stage['state'] = 'skipped'
                    stage.pop('proof', None)
            write_json(setup_input, {'expected_revision': setup['revision'], 'state': setup_state})
            setup_request = str(uuid.uuid4())
            saved_setup = call('setup', 'save', '--input', setup_input, '--request-id', setup_request)
            replayed_setup = call('setup', 'save', '--input', setup_input, '--request-id', setup_request)
            if saved_setup['accepted_revision'] != replayed_setup['accepted_revision'] or call('settings', 'setup')['state']['status'] != 'deferred':
                raise ValueError('packaged setup save/replay/reopen differs')
            if len(call('setup', 'readiness')['tasks']) != 5:
                raise ValueError('packaged independent task readiness missing')
            import_input = Path(temporary) / 'import fixtures.json'
            write_json(import_input, {'kind': 'import-manifest', 'schema_version': VERSION, 'items': [
                {'source': str(media / 'speech.flac'), 'subtitle': str(media / 'speech.srt'), 'originated_at': '2020-01-01T00:00:00.123456789Z'},
                {'source': str(media / 'sintel-dialogue.mkv'), 'subtitle': str(media / 'sintel.srt'),
                 'originated_at': '2024-11-03T01:30:00', 'timezone': 'America/New_York'}]})
            accepted = call('media', 'import', '--manifest', import_input, allow_detached=True)
            job_id = accepted['work_id']
            current = call('work', 'wait', job_id, '--timeout-ms', '30000')
            if current['state'] != 'succeeded':
                raise ValueError('packaged real media import did not succeed')
            items = current['result']['items']
            if not current['result']['partial'] or not items[1].get('date_unresolved'):
                raise ValueError('Ambiguous date incorrectly failed work or lost uncertainty')
            human_wait = child([cli, '--workspace', workspace, 'work', 'wait', job_id], directory=root, env=env).decode('utf-8')
            quiet_wait = child([cli, '--workspace', workspace, '--quiet', 'work', 'wait', job_id], directory=root, env=env).decode('utf-8')
            if any('Warning:' not in text or items[1]['media_id'] not in text or '--originated-on' not in text or '\x1b' in text for text in [human_wait, quiet_wait]):
                raise ValueError('Accepted partial date did not reach readable operator output')
            if len(items) != 2 or any(item.get('state') != 'admitted' or item.get('capture_state') not in ['captured', 'no-embedded-metadata'] for item in items):
                raise ValueError('packaged import did not complete both licensed fixtures')
            from cli_contract_qualification import qualify as qualify_contract
            contract_receipt = qualify_contract(cli, workspace, env=env, cwd=root, documentation=True)
            entry = call('media', 'show', items[0]['media_id'])
            if entry['dates']['selected']['resolved'] != {'iso': '2020-01-01T00:00:00.123456789Z', 'unix_ns': 1577836800123456789}:
                raise ValueError('Packaged timestamp or numeric fidelity differs')
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
            write_json(input_path, {'filter': {'include_undated': True}})
            if call('timeline', 'calendar', '--input', input_path).get('total') != len(call('media', 'list')['items']):
                raise ValueError('packaged whole-library calendar missing retained undated media')
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
            idle_started = time.monotonic()
            if replacement['state'] != 'succeeded' or recording['recording']['state'] != 'untranscribed' or retained['revision'] != roster['revision'] or retained['members']:
                raise ValueError('packaged replacement/declared-empty roster authority changed')
        except BaseException:
            # Preserve bounded execution metadata before the owned disposable
            # workspace is removed. Never dump payloads or result documents.
            import sqlite3
            try:
                database=workspace/'.insonic/catalog.sqlite'
                with sqlite3.connect(database.as_uri()+'?mode=ro',uri=True,timeout=1) as connection:
                    state={}
                    for name,query in {
                        'revision':'SELECT revision FROM workspace',
                        'graph':'SELECT next_sequence,checkpoint,generation,lease_until FROM graph_target LIMIT 4',
                        'work':'SELECT kind,state,phase,error,generation,lease_until FROM work_operation LIMIT 8',
                    }.items():
                        try:
                            rows=connection.execute(query)
                            state[name]=[dict(zip([column[0] for column in rows.description],row)) for row in rows]
                        except sqlite3.Error:state[name]='unavailable'
                print('Package fixture execution state: '+json.dumps(state,sort_keys=True),file=sys.stderr,flush=True)
            except (OSError,sqlite3.Error):pass
            raise
        finally:
            # Join both activity paths before allowing every detached owner its
            # idle exit; an image still held afterward remains a hard failure.
            finish_package_checks(gui_future, idle_started, cli)
    receipt.update({'windows_copy_install': installation, 'windows_first_use': installation, 'cli_contract': contract_receipt, 'extracted_inventory': 'passed', 'relocation_path_spaces': 'passed',
                    'cli_real_audio_video_import': 'passed', 'recording_replacement': 'passed', 'declared_roster': 'passed', 'cueson_assembly': 'passed', 'native_subtitle_export': 'passed',
                    'gui_bridge': 'passed' if receipt['variant'] == 'desktop' else 'not-included',
                    'native_webview': 'passed' if receipt['variant'] == 'desktop' else 'not-included', 'development_paths': 'not-required',
                    'optional_setup': 'passed', 'setup_retry_readback': 'passed', 'setup_deferral_preserves_import': 'passed',
                    'partial_date_feedback': 'passed', 'inference': 'not-run', 'elapsed_seconds': round(time.monotonic() - started, 3)})
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
