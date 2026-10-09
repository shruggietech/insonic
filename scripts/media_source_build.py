# SPDX-License-Identifier: Apache-2.0
"""Owned source-complete media companions on the qualified native platforms."""
import hashlib
from concurrent.futures import ThreadPoolExecutor
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import shlex
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

from process_tree import ProcessTree

ROOT = Path(__file__).resolve().parent.parent
PIN = json.loads((ROOT / 'internal/qualification/media-tools.json').read_text(encoding='utf-8'))['source_build']
COMMIT, SOURCE_SHA256, SOURCE_URL, VERSION = (PIN[name] for name in ['commit', 'sha256', 'url', 'version'])
CONFIGURE, LAME = PIN['configure'], PIN['lame']
VERSION_POLICY = 'pinned-release-VERSION-v1'
BUILD = ROOT / 'build/native/source-media'


def sha(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + '\n', encoding='utf-8', newline='\n')


def fetch_pin(pin):
    cache = ROOT / 'build/media-source-pins'
    cache.mkdir(parents=True, exist_ok=True)
    target = cache / pin['name']
    if not target.is_file() or sha(target) != pin['sha256']:
        temporary = target.with_suffix('.download')
        deadline = time.monotonic() + 60
        try:
            request = urllib.request.Request(pin['url'], headers={'User-Agent': 'insonic-source-build'})
            host = urllib.parse.urlsplit(pin['url']).hostname
            for attempt in range(3):
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise RuntimeError('media source acquisition budget exceeded: ' + pin['name'] + ' from ' + host)
                try:
                    with urllib.request.urlopen(request, timeout=min(20, remaining)) as source, temporary.open('wb') as output:
                        while True:
                            if time.monotonic() >= deadline:
                                raise TimeoutError('source acquisition deadline reached')
                            chunk = source.read(1024 * 1024)
                            if not chunk:
                                break
                            output.write(chunk)
                    break
                except (urllib.error.URLError, TimeoutError, ConnectionError) as error:
                    retry = (not isinstance(error, urllib.error.HTTPError)
                             or error.code in [408, 429, 500, 502, 503, 504])
                    if not retry or attempt == 2:
                        raise RuntimeError('media source acquisition failed: ' + pin['name'] + ' from ' + host
                                           + ' after ' + str(attempt + 1) + ' attempt(s): ' + str(error)) from error
                    time.sleep(attempt + 1)
            if sha(temporary) != pin['sha256'] or ('size_bytes' in pin and temporary.stat().st_size != pin['size_bytes']):
                raise ValueError('media source identity mismatch: ' + pin['name'])
            temporary.replace(target)
        finally:
            temporary.unlink(missing_ok=True)
    if 'size_bytes' in pin and target.stat().st_size != pin['size_bytes']:
        raise ValueError('media source size mismatch: ' + pin['name'])
    return target


def fetch_source(commit=COMMIT, digest=SOURCE_SHA256):
    return fetch_pin({'name': 'ffmpeg-' + commit + '.tar.gz',
                      'url': 'https://codeload.github.com/FFmpeg/FFmpeg/tar.gz/' + commit, 'sha256': digest})


def fetch_lame():
    return fetch_pin(LAME)


def extract_source(source, directory, root_name=None):
    root_name = root_name or 'FFmpeg-' + COMMIT
    prefix, seen = root_name + '/', set()
    with tarfile.open(source) as archive:
        for item in archive:
            if item.name == root_name and item.isdir():
                continue
            if not item.name.startswith(prefix):
                raise ValueError('unexpected media source root')
            relative = PurePosixPath(item.name[len(prefix):])
            if not relative.parts:
                continue
            name = relative.as_posix()
            if (relative.is_absolute() or '..' in relative.parts or ':' in name or '\\' in name
                    or name in seen or not (item.isdir() or item.isfile())):
                raise ValueError('unexpected media source member')
            seen.add(name)
            target = directory.joinpath(*relative.parts)
            if not target.resolve().is_relative_to(directory.resolve()) or target.is_symlink():
                raise ValueError('media source escapes extraction directory')
            if item.isdir():
                target.mkdir(parents=True, exist_ok=True)
            else:
                target.parent.mkdir(parents=True, exist_ok=True)
                with archive.extractfile(item) as data, target.open('wb') as output:
                    shutil.copyfileobj(data, output)
                target.chmod(item.mode & 0o755)
                # Autotools release archives carry generated configure and
                # Makefile timestamps. Preserve them so extraction never elects
                # an unpinned autoreconf toolchain or changes the source recipe.
                os.utime(target, (item.mtime, item.mtime))


def pin_release_version(directory):
    if (directory / 'RELEASE').read_text(encoding='utf-8').strip() != VERSION:
        raise ValueError('FFmpeg source release differs from pinned version')
    (directory / 'VERSION').write_text(VERSION + '\n', encoding='utf-8', newline='\n')


def platform_key():
    systems = {'Windows': 'windows', 'Linux': 'linux', 'Darwin': 'darwin'}
    arches = {'AMD64': 'amd64', 'x86_64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}
    key = systems.get(platform.system(), '') + '_' + arches.get(platform.machine(), '')
    if key not in PIN['platforms']:
        raise ValueError('media source build platform is not qualified')
    return key


def build_environment(key, prefix):
    env = os.environ.copy()
    if key.startswith('windows'):
        msys = Path(env.get('INSONIC_MSYS2_ROOT', 'C:/msys64'))
        if not (msys / 'usr/bin/bash.exe').is_file():
            raise ValueError('source build requires the configured MSYS2 toolchain')
        env['PATH'] = os.pathsep.join([str(msys / 'ucrt64/bin'), str(msys / 'usr/bin'), env.get('PATH', '')])
        env['INSONIC_SOURCE_SHELL'] = str(msys / 'usr/bin/bash.exe')
        env.update({'MSYSTEM': 'UCRT64', 'MSYS2_PATH_TYPE': 'inherit', 'CHERE_INVOKING': '1',
                    'CC': 'gcc', 'CXX': 'g++', 'AR': 'ar', 'RANLIB': 'ranlib'})
    elif key.startswith('darwin'):
        env.update({'MACOSX_DEPLOYMENT_TARGET': '13.3', 'CC': 'clang', 'CXX': 'clang++'})
    else:
        env.update({'CC': 'gcc', 'CXX': 'g++'})
    env.update({'PKG_CONFIG_PATH': str(prefix / 'lib/pkgconfig'),
                'PKG_CONFIG_LIBDIR': str(prefix / 'lib/pkgconfig'),
                'CFLAGS': '-O2 -fPIC', 'CXXFLAGS': '-O2 -fPIC',
                'CPPFLAGS': '-I' + prefix.as_posix() + '/include',
                'LDFLAGS': '-L' + prefix.as_posix() + '/lib', 'LC_ALL': 'C', 'LANG': 'C'})
    return env


def validate_build_tools(platform_id, env):
    # A verified complete cache needs its compiler identity and hashes, without
    # requiring tools used only to produce that cache. Cold builds check these
    # before extracting archives or spending time compiling dependencies.
    if platform_id.startswith('windows'):
        msys = Path(env['INSONIC_SOURCE_SHELL']).parents[2]
        for relative in ['ucrt64/bin/gcc.exe', 'ucrt64/bin/g++.exe', 'ucrt64/bin/nasm.exe',
                         'ucrt64/bin/cmake.exe', 'ucrt64/bin/ninja.exe', 'ucrt64/bin/pkg-config.exe',
                         'usr/bin/make.exe']:
            if not (msys / relative).is_file():
                raise ValueError('configured MSYS2 source tool is missing: ' + str(msys / relative))


def child(args, directory, timeout=360, *, env=None, commands=None, shell=False):
    args = [str(arg) for arg in args]
    selected = env or os.environ.copy()
    if commands is not None:
        commands.append({'directory': str(directory.relative_to(BUILD)) if directory.is_relative_to(BUILD) else str(directory),
                         'arguments': args})
    if shell:
        executable = selected.get('INSONIC_SOURCE_SHELL', '/bin/sh')
        args = [executable, '-c', shlex.join(args)]
    with ProcessTree(args, cwd=directory, env=selected) as tree:
        try:
            output, error = tree.process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired as error:
            tree.kill()
            output, error_output = tree.process.communicate(timeout=5)
            raise RuntimeError('source media build timed out after ' + str(timeout) + ' seconds: '
                               + shlex.join(args) + '\n' + (output + error_output).decode(errors='replace')[-8000:]) from error
        if tree.process.returncode:
            raise RuntimeError('source media build failed: ' + (output + error).decode(errors='replace')[-8000:])
        return output


def source_pins():
    return {'ffmpeg': {'name': 'ffmpeg-' + COMMIT + '.tar.gz', 'url': SOURCE_URL, 'sha256': SOURCE_SHA256,
                       'root': 'FFmpeg-' + COMMIT, 'version': VERSION, 'license': 'LGPL-2.1-or-later',
                       'notices': ['COPYING.LGPLv2.1', 'LICENSE.md']}, 'lame': LAME, **PIN['libraries']}


def validate_source_notices(directory, pins):
    for name, pin in pins.items():
        root = directory / (name + '-source')
        for notice in pin['notices']:
            path = root / notice
            if not path.resolve().is_relative_to(root.resolve()) or not path.is_file() or path.is_symlink():
                raise ValueError('corresponding source notice missing: ' + name + '/' + notice)


def validate_capabilities(output, required, category):
    available = set()
    for line in output.splitlines():
        columns = line.split()
        if len(columns) > 1 and re.fullmatch(r'[A-Z.]{1,8}', columns[0]):
            available.update(columns[1].split(','))
    missing = sorted(set(required) - available)
    if missing:
        raise ValueError('source companion lacks required ' + category + ': ' + ', '.join(missing))
    return sorted(available)


def cache_valid(directory, receipt, key, names):
    try:
        return (receipt.get('key') == key and receipt.get('source_complete') is True
                and receipt.get('sources') == source_pins()
                and len(receipt['static_libraries']) >= len(PIN['libraries']) + 1
                and receipt.get('versions') == {'ffmpeg': VERSION, 'ffprobe': VERSION}
                and all((directory / name).is_file() and sha(directory / name) == receipt['binaries'][name] for name in names)
                and all((directory / item['path']).is_file() and sha(directory / item['path']) == item['sha256']
                        for item in receipt['static_libraries'])
                and (not names[0].endswith('.exe') or bool(receipt.get('runtime_notices')))
                and all((directory / item['path']).is_file() and sha(directory / item['path']) == item['sha256']
                        for item in receipt.get('runtime_notices', [])))
    except (KeyError, OSError, TypeError):
        return False


def normalize_prefix_aliases(prefix):
    """Materialize only private-prefix aliases to ordinary installed files."""
    if prefix.is_symlink():
        raise ValueError('dependency install prefix is a symbolic link: ' + str(prefix))
    root = prefix.resolve(strict=True)
    aliases = []
    # Validate the whole install before replacing any link. An invalid later
    # alias must not leave a partially normalized dependency installation.
    for path in sorted(prefix.rglob('*')):
        if not path.is_symlink():
            if not (path.is_dir() or stat.S_ISREG(path.lstat().st_mode)):
                raise ValueError('dependency install has a nonregular member: ' + str(path))
            continue
        try:
            target = path.resolve(strict=True)
        except (OSError, RuntimeError) as error:
            raise ValueError('dependency alias is missing or cyclic: ' + str(path)) from error
        if not target.is_relative_to(root):
            raise ValueError('dependency alias escapes private prefix: ' + str(path))
        if not stat.S_ISREG(target.stat().st_mode):
            raise ValueError('dependency alias target is not a regular file: ' + str(path))
        aliases.append((path, target, sha(target)))
    audit = []
    for path, target, digest in aliases:
        with tempfile.NamedTemporaryFile(dir=path.parent, prefix='.insonic-alias-', delete=False) as output:
            temporary = Path(output.name)
        try:
            shutil.copy2(target, temporary)
            if sha(temporary) != digest:
                raise ValueError('dependency alias changed while materializing: ' + str(path))
            temporary.replace(path)
        finally:
            temporary.unlink(missing_ok=True)
        audit.append({'path': path.relative_to(prefix).as_posix(),
                      'target': target.relative_to(root).as_posix(), 'sha256': digest})
    return audit


def dependency_cache_errors(directory, receipt, key):
    """Explain rejected closure identities without silently accepting aliases."""
    errors = []
    try:
        prefix = directory / 'install'
        files = receipt['files']
        if not isinstance(files, dict):
            return ['dependency files inventory is not an object']
        for field, expected in [('key', key), ('kind', 'media-dependency-build'),
                                ('sources', source_pins()), ('install_prefix', str(prefix))]:
            if receipt.get(field) != expected:
                errors.append('receipt ' + field + ' differs')
        if len([name for name in files if name.endswith('.a')]) < len(PIN['libraries']) + 1:
            errors.append('static archive closure has too few members')
        if prefix.is_symlink():
            errors.append('install prefix is a symbolic link')
        actual = set()
        for path in prefix.rglob('*'):
            name = path.relative_to(directory).as_posix()
            if path.is_symlink():
                errors.append('symbolic link remains: ' + name)
            elif path.is_file():
                actual.add(name)
            elif not path.is_dir():
                errors.append('nonregular member: ' + name)
        errors.extend('unrecorded member: ' + name for name in sorted(actual - set(files)))
        errors.extend('missing member: ' + name for name in sorted(set(files) - actual))
        for name, digest in files.items():
            path = directory / name
            relative = PurePosixPath(name)
            if (relative.is_absolute() or '..' in relative.parts or ':' in name or '\\' in name
                    or not path.resolve().is_relative_to(prefix.resolve())):
                errors.append('inventory member escapes private prefix: ' + name)
            elif path.is_file() and not path.is_symlink() and sha(path) != digest:
                errors.append('member digest differs: ' + name)
        for alias in receipt.get('normalized_aliases', []):
            path = prefix / alias['path']
            target = prefix / alias['target']
            if (not path.resolve().is_relative_to(prefix.resolve())
                    or not target.resolve().is_relative_to(prefix.resolve())
                    or files.get(path.relative_to(directory).as_posix()) != alias['sha256']
                    or files.get(target.relative_to(directory).as_posix()) != alias['sha256']):
                errors.append('normalized alias audit differs: ' + alias['path'])
    except (KeyError, OSError, TypeError, ValueError, RuntimeError) as error:
        errors.append('invalid dependency receipt: ' + str(error))
    return errors


def dependency_cache_valid(directory, receipt, key):
    return not dependency_cache_errors(directory, receipt, key)


def cache_identity():
    """Return the exact identity used by source builds and artifact transfers."""
    platform_id = platform_key()
    suffix = '.exe' if platform_id.startswith('windows') else ''
    names = ['ffmpeg' + suffix, 'ffprobe' + suffix]
    tool_env = build_environment(platform_id, BUILD / 'identity')
    compiler = child([tool_env['CC'], '--version'], ROOT, env=tool_env).decode().splitlines()[0]
    recipe_sha = sha(Path(__file__))
    key = hashlib.sha256(json.dumps({'pin': PIN, 'recipe_sha256': recipe_sha,
                                    'launcher_sha256': sha(ROOT / 'scripts/process_tree.py'), 'compiler': compiler,
                                    'platform': platform_id, 'version_policy': VERSION_POLICY}, sort_keys=True).encode()).hexdigest()
    return platform_id, names, key, compiler, recipe_sha


def compile_dav1d(target, build, prefix, parallel_jobs, run):
    run([sys.executable, '-m', 'mesonbuild.mesonmain', 'setup', build, target, '--prefix=' + prefix.as_posix(),
         '--libdir=lib', '--default-library=static', '--buildtype=release', '-Db_staticpic=true',
         '-Denable_tools=false', '-Denable_examples=false', '-Denable_tests=false', '-Denable_docs=false'], build.parent)
    # Assembly-heavy AV1 compilation can outlive the other independent builds.
    # Use the available cores rather than a permanently divided worker count.
    run(['ninja', '-C', build, '-j', str(max(1, min(parallel_jobs, 8)))], build.parent, 240)
    run([sys.executable, '-m', 'mesonbuild.mesonmain', 'install', '-C', build], build.parent, 60)


def prepare(stage='complete'):
    if stage not in ['dependencies', 'complete']:
        raise ValueError('unknown source build stage')
    platform_id, names, key, compiler, recipe_sha = cache_identity()
    suffix = '.exe' if platform_id.startswith('windows') else ''
    directory = BUILD / key
    receipt_path = directory / 'build-receipt.json'
    dependency_path = directory / 'dependency-receipt.json'
    source = fetch_source()
    if stage == 'complete' and receipt_path.is_file():
        receipt = json.loads(receipt_path.read_text(encoding='utf-8'))
        if cache_valid(directory, receipt, key, names):
            return directory, receipt, source
    directory.mkdir(parents=True, exist_ok=True)
    started = time.monotonic()
    pins, commands = source_pins(), []
    dependency_receipt = json.loads(dependency_path.read_text(encoding='utf-8')) if dependency_path.is_file() else {}
    dependencies_ready = dependency_cache_valid(directory, dependency_receipt, key)
    if dependencies_ready and stage == 'dependencies':
        return directory, dependency_receipt, source
    if dependencies_ready:
        commands.extend(dependency_receipt['commands'])
    prefix = directory / 'install'
    env = build_environment(platform_id, prefix)
    validate_build_tools(platform_id, env)
    source_directories = {}
    for name, pin in pins.items():
        target = directory / (name + '-source')
        target.mkdir(exist_ok=True)
        extract_source(fetch_pin(pin), target, pin['root'])
        source_directories[name] = target
    validate_source_notices(directory, pins)
    prefix.mkdir(exist_ok=True)
    parallel_jobs = min(os.cpu_count() or 2, 8)
    jobs = str(max(1, parallel_jobs // 3))
    def run(args, cwd, timeout=180, shell=False):
        return child(args, cwd, timeout, env=env, commands=commands, shell=shell)
    def autotools(name, flags):
        target = source_directories[name]
        host = 'x86_64-w64-mingw32' if platform_id.startswith('windows') else ('aarch64-apple-darwin' if platform_id.startswith('darwin') else 'x86_64-pc-linux-gnu')
        run(['./configure', '--prefix=' + prefix.as_posix(), '--host=' + host, '--disable-shared', '--enable-static', *flags], target, shell=True)
        run(['make', '-j' + jobs], target, 240, shell=platform_id.startswith('windows'))
        run(['make', 'install'], target, 60, shell=platform_id.startswith('windows'))
    def cmake(name, flags):
        target, build = source_directories[name], directory / (name + '-build')
        run(['cmake', '-S', target, '-B', build, '-G', 'Ninja', '-DCMAKE_BUILD_TYPE=Release',
             '-DCMAKE_POLICY_VERSION_MINIMUM=3.5', '-DCMAKE_INSTALL_PREFIX=' + prefix.as_posix(),
             '-DCMAKE_INSTALL_LIBDIR=lib', '-DCMAKE_POSITION_INDEPENDENT_CODE=ON',
             '-DBUILD_SHARED_LIBS=OFF', '-DCMAKE_PREFIX_PATH=' + prefix.as_posix(), *flags], directory)
        run(['cmake', '--build', build, '--parallel', jobs], directory, 180)
        run(['cmake', '--install', build], directory, 60)
    (prefix / 'lib').mkdir(exist_ok=True); (prefix / 'include').mkdir(exist_ok=True)
    def build_zlib():
        run(['./configure', '--static', '--prefix=' + prefix.as_posix()], source_directories['zlib'], shell=True)
        run(['make', '-j' + jobs], source_directories['zlib'], shell=platform_id.startswith('windows'))
        run(['make', 'install'], source_directories['zlib'], shell=platform_id.startswith('windows'))
    def build_bzip2():
        run(['make', '-j' + jobs, 'libbz2.a', 'CC=' + env['CC'], 'CFLAGS=-O2 -fPIC'], source_directories['bzip2'], shell=platform_id.startswith('windows'))
        shutil.copyfile(source_directories['bzip2'] / 'libbz2.a', prefix / 'lib/libbz2.a')
        shutil.copyfile(source_directories['bzip2'] / 'bzlib.h', prefix / 'include/bzlib.h')
    def build_dav1d():
        compile_dav1d(source_directories['dav1d'], directory / 'dav1d-build', prefix, parallel_jobs, run)
    def parallel(operations):
        with ThreadPoolExecutor(max_workers=3) as executor:
            futures = [executor.submit(operation) for operation in operations]
            for future in futures:
                future.result()
    # Independent builds own separate working directories and install distinct
    # archives/headers/pkg-config files. Dependency groups join before consumers
    # configure; no partially installed library enters a link command.
    jobs = str(parallel_jobs)
    if not dependencies_ready:
        jobs = str(max(1, parallel_jobs // 3))
        parallel([build_dav1d, build_zlib, build_bzip2,
                  lambda: cmake('xz', ['-DBUILD_TESTING=OFF', '-DXZ_TOOL_XZ=OFF', '-DXZ_TOOL_XZDEC=OFF', '-DXZ_TOOL_LZMADEC=OFF', '-DXZ_TOOL_LZMAINFO=OFF']),
                  lambda: autotools('iconv', ['--disable-nls']),
                  lambda: autotools('lame', LAME['configure']),
                  lambda: cmake('ogg', ['-DBUILD_TESTING=OFF', '-DINSTALL_DOCS=OFF']),
                  lambda: autotools('mpg123', ['--disable-network'])])
        parallel([lambda: cmake('xml2', ['-DLIBXML2_WITH_PROGRAMS=OFF', '-DLIBXML2_WITH_TESTS=OFF', '-DLIBXML2_WITH_PYTHON=OFF', '-DLIBXML2_WITH_ZLIB=ON', '-DLIBXML2_WITH_LZMA=ON']),
                  lambda: cmake('gme', ['-DENABLE_UBSAN=OFF']),
                  lambda: cmake('vorbis', ['-DBUILD_TESTING=OFF'])])
        jobs = str(parallel_jobs)
        autotools('openmpt', ['--disable-openmpt123', '--disable-examples', '--disable-tests',
                            '--without-portaudio', '--without-portaudiocpp', '--without-pulseaudio', '--without-sdl2', '--without-sndfile', '--without-flac'])
        normalized_aliases = normalize_prefix_aliases(prefix)
        dependency_receipt = {'kind': 'media-dependency-build', 'key': key, 'sources': pins,
                              'platform': platform_id, 'compiler': compiler, 'recipe_sha256': recipe_sha,
                              'launcher_sha256': sha(ROOT / 'scripts/process_tree.py'),
                              'install_prefix': str(prefix), 'commands': commands,
                              'normalized_aliases': normalized_aliases,
                              'files': {path.relative_to(directory).as_posix(): sha(path)
                                        for path in sorted(prefix.rglob('*')) if path.is_file()},
                              'elapsed_seconds': round(time.monotonic() - started, 3)}
        write_json(dependency_path, dependency_receipt)
        closure_errors = dependency_cache_errors(directory, dependency_receipt, key)
        if closure_errors:
            raise ValueError('source dependency closure is incomplete: ' + '; '.join(closure_errors[:20]))
    if stage == 'dependencies':
        return directory, dependency_receipt, source
    ffmpeg_started = time.monotonic()
    ffmpeg = source_directories['ffmpeg']
    pin_release_version(ffmpeg)
    effective_configure = [*CONFIGURE, *PIN['platform_configure'][platform_id],
                           '--pkg-config-flags=--static', '--extra-cflags=-I' + prefix.as_posix() + '/include',
                           '--extra-ldflags=-L' + prefix.as_posix() + '/lib']
    run(['./configure', *effective_configure], ffmpeg, shell=True)
    run(['make', '-j' + jobs, 'ffmpeg' + suffix, 'ffprobe' + suffix], ffmpeg, 360, shell=platform_id.startswith('windows'))
    configurations, capabilities, versions = {}, {}, {}
    for name in names:
        shutil.copyfile(ffmpeg / name, directory / name)
        (directory / name).chmod(0o755)
        output = child([directory / name, '-version'], directory, 10, env=env).decode()
        if '--enable-nonfree' in output or '--enable-gpl' in output:
            raise ValueError('source media companion enabled incompatible distribution flags')
        versions[name.removesuffix('.exe')] = output.splitlines()[0].split()[2]
        if versions[name.removesuffix('.exe')] != VERSION:
            raise ValueError('source media companion reports a different release version')
        configurations[name] = output
    for category, required in PIN['required_capabilities'].items():
        output = child([directory / names[0], '-hide_banner', '-' + category], directory, 10, env=env).decode()
        capabilities[category] = validate_capabilities(output, required, category)
    if platform_id.startswith('darwin'):
        for name in names:
            if 'libmp3lame' in child(['/usr/bin/otool', '-L', directory / name], directory, 10, env=env).decode():
                raise ValueError('media companion retained external LAME dylib')
    libraries = [{'path': path.relative_to(directory).as_posix(), 'sha256': sha(path)} for path in sorted((prefix / 'lib').glob('*.a'))]
    if len(libraries) < len(PIN['libraries']) + 1:
        raise ValueError('media companion static library closure is incomplete')
    runtime_notices = []
    if platform_id.startswith('windows'):
        licenses = Path(env['INSONIC_SOURCE_SHELL']).parents[2] / 'ucrt64/share/licenses'
        target = directory / 'compiler-notices'
        target.mkdir(exist_ok=True)
        for component in ['gcc', 'libgcc', 'libstdc++', 'crt', 'winpthreads']:
            paths = list((licenses / component).glob('COPYING*'))
            if not paths:
                raise ValueError('source compiler runtime license missing: ' + component)
            for path in paths:
                selected = target / (component + '-' + path.name)
                shutil.copyfile(path, selected)
                runtime_notices.append({'path': selected.relative_to(directory).as_posix(), 'sha256': sha(selected)})
    receipt = {'key': key, 'source_commit': COMMIT, 'source_sha256': SOURCE_SHA256, 'source_url': SOURCE_URL,
               'configure': effective_configure, 'platform': platform_id, 'compiler': compiler,
               'recipe_sha256': recipe_sha, 'sources': pins, 'source_complete': True, 'commands': commands,
               'launcher_sha256': sha(ROOT / 'scripts/process_tree.py'),
               'binaries': {name: sha(directory / name) for name in names}, 'versions': versions,
               'configuration_output': configurations, 'capabilities': capabilities, 'static_libraries': libraries,
               'runtime_notices': runtime_notices,
               'normalized_aliases': dependency_receipt.get('normalized_aliases', []),
               'dependency_elapsed_seconds': dependency_receipt['elapsed_seconds'],
               'ffmpeg_elapsed_seconds': round(time.monotonic() - ffmpeg_started, 3),
               'version_override': {'file': 'VERSION', 'value': VERSION, 'policy': VERSION_POLICY},
               'elapsed_seconds': round(time.monotonic() - started, 3), 'license': 'LGPL-2.1-or-later',
               'system_framework': 'VideoToolbox' if platform_id.startswith('darwin') else None}
    write_json(receipt_path, receipt)
    return directory, receipt, source


if __name__ == '__main__':
    directory, receipt, source = prepare()
    print(json.dumps(receipt))
