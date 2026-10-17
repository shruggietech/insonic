#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Build/independently inspect the ordinary-user setup EXE and exact MSI overlay.

No installation APIs are invoked here. Native lifecycle qualification belongs
on acknowledged disposable Windows hosts.
"""
import ctypes
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import struct
import sys
import xml.etree.ElementTree as ET

sys.path.insert(0, str(Path(__file__).resolve().parent))
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


def add_resources(executable, version):
    if os.name != 'nt': raise RuntimeError('native setup resource authoring requires Windows')
    dll = ctypes.WinDLL('kernel32', use_last_error=True)
    begin, update, end = dll.BeginUpdateResourceW, dll.UpdateResourceW, dll.EndUpdateResourceW
    begin.argtypes = [ctypes.c_wchar_p, ctypes.c_int]; begin.restype = ctypes.c_void_p
    update.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_ushort, ctypes.c_void_p, ctypes.c_uint32]; update.restype = ctypes.c_int
    end.argtypes = [ctypes.c_void_p, ctypes.c_int]; end.restype = ctypes.c_int
    handle = begin(str(executable), False)
    if not handle: raise ctypes.WinError(ctypes.get_last_error())
    images, group = icon_resources((m.ROOT / m.ICON).read_bytes())
    complete = False
    try:
        for kind, identity, data in [(24, 1, manifest(version)), (14, 1, group), *[(3, i + 1, data) for i, data in enumerate(images)]]:
            buffer = ctypes.create_string_buffer(data)
            if not update(handle, kind, identity, 1033, buffer, len(data)): raise ctypes.WinError(ctypes.get_last_error())
        complete = True
    finally:
        if not end(handle, not complete) and complete: raise ctypes.WinError(ctypes.get_last_error())


def pe_resources(data):
    """Independent bounded PE resource reader (no LoadLibrary or execution)."""
    if data[:2] != b'MZ' or len(data) < 64: raise ValueError('setup PE header missing')
    pe = struct.unpack_from('<I', data, 60)[0]
    if pe + 24 > len(data) or data[pe:pe + 4] != b'PE\0\0': raise ValueError('invalid setup PE signature')
    machine, sections, _, _, _, optional_size, _ = struct.unpack_from('<HHIIIHH', data, pe + 4)
    optional = pe + 24
    if machine != 0x8664 or not 0 < sections <= 96 or optional_size < 136 or optional + optional_size + sections * 40 > len(data): raise ValueError('invalid setup PE architecture')
    if struct.unpack_from('<H', data, optional)[0] != 0x20b: raise ValueError('setup requires PE32+')
    if struct.unpack_from('<H', data, optional + 68)[0] != 2: raise ValueError('setup must use the GUI subsystem')
    rva, length = struct.unpack_from('<II', data, optional + 112 + 16)
    mappings = [struct.unpack_from('<IIII', data, optional + optional_size + i * 40 + 8) for i in range(sections)]
    def offset(address, size):
        matches = [raw + address - start for virtual, start, raw_size, raw in mappings if start <= address and address + size <= start + raw_size]
        if len(matches) != 1 or matches[0] + size > len(data): raise ValueError('setup resource escapes PE')
        return matches[0]
    base = offset(rva, length)
    def entries(relative):
        if relative < 0 or relative + 16 > length: raise ValueError('invalid setup resource directory')
        named, count = struct.unpack_from('<HH', data, base + relative + 12)
        if named or count > 1024 or relative + 16 + count * 8 > length: raise ValueError('unsupported setup resource directory')
        return [struct.unpack_from('<II', data, base + relative + 16 + i * 8) for i in range(count)]
    result = {}
    for kind, name_dir in entries(0):
        if not name_dir & 0x80000000: raise ValueError('invalid setup resource type')
        for identity, language_dir in entries(name_dir & 0x7fffffff):
            if not language_dir & 0x80000000: raise ValueError('invalid setup resource name')
            for language, leaf in entries(language_dir & 0x7fffffff):
                if leaf & 0x80000000 or leaf + 16 > length: raise ValueError('invalid setup resource leaf')
                address, size, _, _ = struct.unpack_from('<IIII', data, base + leaf)
                start = offset(address, size)
                key = kind, identity, language
                if key in result: raise ValueError('duplicate setup resource')
                result[key] = data[start:start + size]
    return result


def verify_resources(path, version, source_root=None):
    source_root = Path(source_root or m.ROOT)
    _, prefix_size = read_payload(path)
    if prefix_size > 128 << 20: raise ValueError('setup executable prefix exceeds bound')
    with Path(path).open('rb') as source: resources = pe_resources(source.read(prefix_size))
    if resources.get((24, 1, 1033)) != manifest(version): raise ValueError('setup asInvoker manifest differs')
    images, group = icon_resources((source_root / m.ICON).read_bytes())
    if resources.get((14, 1, 1033)) != group or any(resources.get((3, i + 1, 1033)) != image for i, image in enumerate(images)): raise ValueError('setup approved icon bytes differ')
    return resources


def build(msi, receipt, child):
    version = receipt['product_version']
    target = Path(msi).with_name('insonic_' + version + '_windows_amd64_setup.exe')
    if m.sha(msi) != receipt['artifact_sha256']: raise ValueError('setup requires exact authored MSI')
    child(['go', 'build', '-ldflags', '-H windowsgui', '-o', target, './cmd/insonic-setup'])
    add_resources(target, version)
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
    if any(setup.get('qualification', {}).get(key) != 'passed' for key in QUALIFICATION_KEYS): raise ValueError('setup recovery qualification is incomplete')
