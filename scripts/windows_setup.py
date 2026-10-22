#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Build/independently inspect the ordinary-user setup EXE and exact MSI overlay.

No installation APIs are invoked here. Native lifecycle qualification belongs
on acknowledged disposable Windows hosts.
"""
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import struct
import subprocess
import sys
import tempfile
import threading

sys.path.insert(0, str(Path(__file__).resolve().parent))
from pe_resources import append_resources, read_resources as pe_resources
import windows_msi as m

MAGIC = b'INSONIC-SETUP-01'
MAX_SIZE = 8 << 30
QUALIFICATION_KEYS = ('ordinary_user', 'source_integrity', 'legacy_source', 'failure_recovery',
                      'cancellation_recovery', 'prior_choices', 'source_cleanup', 'native_entry_point')


def identity_valid(value):
    return (set(value) == {'product_code', 'upgrade_code', 'version', 'sha256', 'size_bytes'}
            and re.fullmatch(r'\{[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}\}', value['product_code'])
            and value['upgrade_code'] == m.UPGRADE_CODE and m.msi_version(value['version']) == value['version']
            and re.fullmatch('[a-f0-9]{64}', value['sha256'])
            and type(value['size_bytes']) is int and 0 < value['size_bytes'] <= MAX_SIZE)


def append_payload(executable, msi, identity):
    if not identity_valid(identity) or m.sha(msi) != identity['sha256'] or msi.stat().st_size != identity['size_bytes']:
        raise ValueError('setup MSI source identity differs')
    metadata = m.canonical(identity)
    with Path(executable).open('ab') as output, Path(msi).open('rb') as source:
        shutil.copyfileobj(source, output, 1024 * 1024)
        output.write(metadata)
        output.write(MAGIC + struct.pack('<QII', identity['size_bytes'], len(metadata), 0))


def read_payload(path):
    size = Path(path).stat().st_size
    if size < 96: raise ValueError('setup footer is missing')
    with Path(path).open('rb') as source:
        source.seek(-32, io.SEEK_END)
        footer = source.read(32)
        length, metadata_length, reserved = struct.unpack('<QII', footer[16:])
        offset = size - 32 - metadata_length - length
        if footer[:16] != MAGIC or reserved or not 0 < length <= MAX_SIZE or not 0 < metadata_length <= 4096 or offset < 64:
            raise ValueError('invalid setup payload bounds')
        source.seek(size - 32 - metadata_length)
        identity = json.loads(source.read(metadata_length))
        if not identity_valid(identity) or identity['size_bytes'] != length: raise ValueError('invalid setup payload identity')
        source.seek(offset)
        digest = hashlib.sha256()
        remaining = length
        while remaining:
            block = source.read(min(1024 * 1024, remaining))
            if not block: raise ValueError('truncated setup MSI source')
            digest.update(block); remaining -= len(block)
        if digest.hexdigest() != identity['sha256']: raise ValueError('setup embedded MSI digest differs')
    return identity, offset


def manifest(version):
    m.msi_version(version)
    return (f'''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity name="ShruggieTech.insonic.Setup" version="{version}.0" processorArchitecture="amd64" type="win32"/>
  <description>insonic per-user setup</description>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security></trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1"><application><supportedOS Id="{{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}}"/></application></compatibility>
</assembly>
''').encode('utf-8')


def icon_resources(icon):
    if len(icon) < 6: raise ValueError('truncated setup icon')
    reserved, kind, count = struct.unpack_from('<HHH', icon)
    if reserved or kind != 1 or not 0 < count <= 256 or len(icon) < 6 + count * 16: raise ValueError('invalid setup icon')
    images, group = [], bytearray(icon[:6])
    for index in range(count):
        entry = 6 + index * 16
        length, offset = struct.unpack_from('<II', icon, entry + 8)
        if not length or offset < 6 + count * 16 or offset + length > len(icon): raise ValueError('setup icon image escapes file')
        images.append(icon[offset:offset + length])
        group.extend(icon[entry:entry + 12] + struct.pack('<H', index + 1))
    return images, bytes(group)


def add_resources(executable, version, source_root=None):
    """Append the same deterministic setup manifest and icon resources."""
    def tree():
        images, group = icon_resources((Path(source_root or m.ROOT) / m.ICON).read_bytes())
        return {24: {1: {1033: manifest(version)}}, 14: {1: {1033: group}},
                3: {i + 1: {1033: image} for i, image in enumerate(images)}}
    Path(executable).write_bytes(append_resources(Path(executable).read_bytes(), tree))


def verify_resources(path, version, source_root=None):
    source_root = Path(source_root or m.ROOT)
    _, prefix_size = read_payload(path)
    if prefix_size > 128 << 20: raise ValueError('setup executable prefix exceeds bound')
    with Path(path).open('rb') as source: resources = pe_resources(source.read(prefix_size))
    if resources.get((24, 1, 1033)) != manifest(version): raise ValueError('setup asInvoker manifest differs')
    images, group = icon_resources((source_root / m.ICON).read_bytes())
    if resources.get((14, 1, 1033)) != group or any(resources.get((3, i + 1, 1033)) != image for i, image in enumerate(images)): raise ValueError('setup approved icon bytes differ')
    return resources


_code_cache = None
_code_cache_lock = threading.Lock()


def compiler_environment(isolated=False):
    allowed = {'PATH', 'PATHEXT', 'SYSTEMROOT', 'WINDIR', 'SYSTEMDRIVE', 'COMSPEC', 'TEMP', 'TMP', 'TMPDIR',
               'HOME', 'USERPROFILE', 'LOCALAPPDATA', 'APPDATA', 'PROGRAMFILES', 'PROGRAMFILES(X86)',
               'HTTP_PROXY', 'HTTPS_PROXY', 'ALL_PROXY', 'NO_PROXY', 'SSL_CERT_FILE', 'SSL_CERT_DIR'}
    if not isolated: allowed.update({'GOCACHE', 'GOMODCACHE', 'GOPATH'})
    selected = {key: value for key, value in os.environ.items() if key.upper() in allowed}
    selected.update(GOOS='windows', GOARCH='amd64', GOAMD64='v1', CGO_ENABLED='0',
                    GOENV='off', GOFLAGS='', GOWORK='off', GOTOOLCHAIN='local', GOEXPERIMENT='',
                    GO111MODULE='on', GOFIPS140='off', GOTELEMETRY='off', GIT_TERMINAL_PROMPT='0')
    if isolated:
        global _code_cache
        with _code_cache_lock:
            parent = (m.ROOT / 'build').resolve()
            if not parent.is_relative_to(m.ROOT.resolve()): raise ValueError('setup verifier cache root is redirected')
            if _code_cache is None: _code_cache = tempfile.TemporaryDirectory(prefix='setup-code-', dir=parent)
            cache = Path(_code_cache.name)
        selected.update(GOCACHE=str(cache / 'compiled'), GOMODCACHE=str(cache / 'modules'),
                        GOPROXY='https://proxy.golang.org', GOSUMDB='sum.golang.org',
                        GONOSUMDB='', GOPRIVATE='', GONOPROXY='')
    return selected


def run_compiler(arguments, source_root, environment):
    # Selected Go source is compiler input only: no cgo, go generate, workspace,
    # alternate toolchain or inherited toolexec/build flags. Never run setup.
    from process_tree import ProcessTree
    with ProcessTree(arguments, cwd=source_root, env=environment) as tree:
        try: output, error = tree.process.communicate(timeout=180)
        except subprocess.TimeoutExpired as expired:
            tree.kill()
            try: tree.process.communicate(timeout=5)
            except subprocess.TimeoutExpired: pass
            raise RuntimeError('setup source rebuild exceeded its bound') from expired
        if tree.process.returncode: raise RuntimeError('setup source rebuild failed: ' + error.decode(errors='replace')[-4000:])
        return output


def compile_prefix(target, version, source_root=None, *, isolated=False):
    source_root = Path(source_root or m.ROOT).resolve(); target = Path(target).resolve()
    compiler = shutil.which('go')
    if not compiler: raise RuntimeError('pinned Go compiler is required to verify setup executable code')
    environment = compiler_environment(isolated)
    expected = json.loads((source_root / 'internal/qualification/dependencies.json').read_text(encoding='utf-8'))['go']
    if run_compiler([compiler, 'env', 'GOVERSION'], source_root, environment).decode().strip() != 'go' + expected:
        raise ValueError('setup rebuild compiler differs from selected source toolchain')
    target.parent.mkdir(parents=True, exist_ok=True)
    run_compiler([compiler, 'build', '-mod=readonly', '-trimpath', '-buildvcs=false',
                  '-ldflags', '-H windowsgui -buildid=', '-o', str(target), './cmd/insonic-setup'], source_root, environment)
    add_resources(target, version, source_root)
    return target


def verify_code(path, version, source_root=None):
    _, prefix_size = read_payload(path)
    if prefix_size > 128 << 20: raise ValueError('setup executable prefix exceeds bound')
    parent = (m.ROOT / 'build').resolve()
    if not parent.is_relative_to(m.ROOT.resolve()): raise ValueError('setup verifier output root is redirected')
    with tempfile.TemporaryDirectory(prefix='setup-rebuild-', dir=parent) as temporary:
        expected = compile_prefix(Path(temporary) / 'setup.exe', version, source_root, isolated=True)
        with Path(path).open('rb') as source: actual = source.read(prefix_size)
        if expected.stat().st_size != prefix_size or hashlib.sha256(actual).hexdigest() != m.sha(expected):
            raise ValueError('setup executable code differs from independently rebuilt source')


def build(msi, receipt):
    version = receipt['product_version']
    target = Path(msi).with_name('insonic_' + version + '_windows_amd64_setup.exe')
    if m.sha(msi) != receipt['artifact_sha256']: raise ValueError('setup requires exact authored MSI')
    compile_prefix(target, version)
    identity = {'product_code': receipt['product_code'], 'upgrade_code': receipt['upgrade_code'],
                'version': version, 'sha256': receipt['artifact_sha256'], 'size_bytes': receipt['artifact_size_bytes']}
    append_payload(target, msi, identity)
    verify_resources(target, version)
    return {'kind': 'windows-setup-receipt', 'format': 'exe', 'artifact': target.name,
            'artifact_sha256': m.sha(target), 'artifact_size_bytes': target.stat().st_size,
            'embedded_msi': identity, 'manifest': 'asInvoker', 'scope': 'per-user',
            'signing': dict(m.UNSIGNED_SIGNING), 'qualification': {key: 'pending' for key in QUALIFICATION_KEYS}}


def verify(path, setup, installer_receipt, source_root=None):
    version = installer_receipt['product_version']
    expected = {'product_code': installer_receipt['product_code'], 'upgrade_code': installer_receipt['upgrade_code'],
                'version': version, 'sha256': installer_receipt['artifact_sha256'], 'size_bytes': installer_receipt['artifact_size_bytes']}
    if (setup.get('kind') != 'windows-setup-receipt' or setup.get('format') != 'exe'
            or setup.get('artifact') != 'insonic_' + version + '_windows_amd64_setup.exe'
            or setup.get('scope') != 'per-user' or setup.get('manifest') != 'asInvoker'
            or setup.get('signing') != m.UNSIGNED_SIGNING or setup.get('embedded_msi') != expected): raise ValueError('setup receipt identity differs')
    if Path(path).name != setup['artifact'] or m.sha(path) != setup.get('artifact_sha256') or Path(path).stat().st_size != setup.get('artifact_size_bytes'): raise ValueError('setup artifact differs')
    if read_payload(path)[0] != expected: raise ValueError('setup embedded MSI differs')
    verify_resources(path, version, source_root)
    verify_code(path, version, source_root)
    if any(setup.get('qualification', {}).get(key) != 'passed' for key in QUALIFICATION_KEYS): raise ValueError('setup recovery qualification is incomplete')
