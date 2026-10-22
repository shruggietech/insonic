# SPDX-License-Identifier: Apache-2.0
"""Deterministic bounded PE resources shared by setup and companion packaging."""
import struct
import hashlib
from pathlib import Path
import xml.etree.ElementTree as ET


def append_resources(data, tree, *, subsystem=2):
    """Append deterministic PE resources, identically on build and verifier hosts."""
    data = bytearray(data)
    if len(data) < 64 or len(data) > 128 << 20 or data[:2] != b'MZ': raise ValueError('invalid setup resource input')
    pe = struct.unpack_from('<I', data, 60)[0]
    if pe + 24 > len(data) or data[pe:pe + 4] != b'PE\0\0': raise ValueError('invalid setup PE signature')
    machine, count, _, _, _, optional_size, _ = struct.unpack_from('<HHIIIHH', data, pe + 4)
    optional = pe + 24; table = optional + optional_size; end = table + count * 40
    if machine != 0x8664 or not 0 < count < 96 or optional_size < 240 or end + 40 > len(data): raise ValueError('invalid setup PE architecture')
    if struct.unpack_from('<H', data, optional)[0] != 0x20b or struct.unpack_from('<H', data, optional + 68)[0] != subsystem: raise ValueError('unexpected PE subsystem')
    alignment, file_alignment = struct.unpack_from('<II', data, optional + 32)
    headers = struct.unpack_from('<I', data, optional + 60)[0]
    if (not 512 <= file_alignment <= alignment <= 1 << 20 or alignment & (alignment - 1)
            or file_alignment & (file_alignment - 1) or end + 40 > headers or headers > len(data)
            or data[end:end + 40] != bytes(40)):
        raise ValueError('setup PE lacks bounded resource header space')
    if any(struct.unpack_from('<II', data, optional + 112 + 32)): raise ValueError('setup resource input must be unsigned')
    sections = [struct.unpack_from('<IIII', data, table + i * 40 + 8) for i in range(count)]
    if any(raw < headers or raw + size > len(data) for _, _, size, raw in sections if size): raise ValueError('setup PE section escapes file')
    align = lambda value, boundary: (value + boundary - 1) & -boundary
    address = align(max(start + max(virtual, size) for virtual, start, size, _ in sections), alignment)
    raw = align(len(data), file_alignment)
    tree = tree() if callable(tree) else tree
    resource = bytearray(); leaves = []
    def directory(children):
        at = len(resource); resource.extend(bytes(16 + len(children) * 8))
        struct.pack_into('<HH', resource, at + 12, 0, len(children))
        for index, (identity, child) in enumerate(sorted(children.items())):
            if isinstance(child, dict): pointer = directory(child) | 0x80000000
            else:
                pointer = len(resource); resource.extend(bytes(16)); leaves.append((pointer, child))
            struct.pack_into('<II', resource, at + 16 + index * 8, identity, pointer)
        return at
    directory(tree)
    for entry, value in leaves:
        resource.extend(bytes(align(len(resource), 4) - len(resource)))
        struct.pack_into('<IIII', resource, entry, address + len(resource), len(value), 0, 0)
        resource.extend(value)
    size = align(len(resource), file_alignment)
    names = {bytes(data[table + i * 40:table + i * 40 + 8]).rstrip(b'\0') for i in range(count)}
    name = next(name for name in (b'.rsrc', b'.rsrc2', b'.rsrc3') if name not in names)
    struct.pack_into('<8sIIIIIIHHI', data, end, name, len(resource), address, size, raw, 0, 0, 0, 0, 0x40000040)
    struct.pack_into('<H', data, pe + 6, count + 1)
    initialized = struct.unpack_from('<I', data, optional + 8)[0]
    struct.pack_into('<I', data, optional + 8, initialized + size)
    struct.pack_into('<I', data, optional + 56, align(address + len(resource), alignment))
    struct.pack_into('<I', data, optional + 64, 0)
    struct.pack_into('<II', data, optional + 112 + 16, address, len(resource))
    data.extend(bytes(raw - len(data))); data.extend(resource); data.extend(bytes(size - len(resource)))
    return bytes(data)


def read_resources(data, *, subsystem=2, allow_empty=False):
    """Independent bounded PE resource reader (no LoadLibrary or execution)."""
    if data[:2] != b'MZ' or len(data) < 64: raise ValueError('setup PE header missing')
    pe = struct.unpack_from('<I', data, 60)[0]
    if pe + 24 > len(data) or data[pe:pe + 4] != b'PE\0\0': raise ValueError('invalid setup PE signature')
    machine, sections, _, _, _, optional_size, _ = struct.unpack_from('<HHIIIHH', data, pe + 4)
    optional = pe + 24
    if machine != 0x8664 or not 0 < sections <= 96 or optional_size < 136 or optional + optional_size + sections * 40 > len(data): raise ValueError('invalid setup PE architecture')
    if struct.unpack_from('<H', data, optional)[0] != 0x20b: raise ValueError('setup requires PE32+')
    if struct.unpack_from('<H', data, optional + 68)[0] != subsystem: raise ValueError('unexpected PE subsystem')
    rva, length = struct.unpack_from('<II', data, optional + 112 + 16)
    if allow_empty and rva == length == 0: return {}
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




ASM = 'urn:schemas-microsoft-com:asm.v1'
V3 = 'urn:schemas-microsoft-com:asm.v3'
SETTINGS = 'http://schemas.microsoft.com/SMI/2019/WindowsSettings'


def utf8_manifest(data):
    if not 0 < len(data) <= 65536 or b'<!DOCTYPE' in data.upper(): raise ValueError('Invalid launcher manifest bound')
    root = ET.fromstring(data)
    if root.tag != '{'+ASM+'}assembly' or root.get('manifestVersion') != '1.0': raise ValueError('Invalid launcher manifest root')
    apps = root.findall('{'+V3+'}application')
    if len(apps) > 1: raise ValueError('Ambiguous launcher application manifest')
    if not apps: apps = [ET.SubElement(root, '{'+V3+'}application')]
    settings = apps[0].findall('{'+V3+'}windowsSettings')
    if len(settings) > 1: raise ValueError('Ambiguous launcher Windows settings')
    if not settings: settings = [ET.SubElement(apps[0], '{'+V3+'}windowsSettings')]
    pages = settings[0].findall('{'+SETTINGS+'}activeCodePage')
    if len(pages) > 1: raise ValueError('Ambiguous launcher code page')
    page = pages[0] if pages else ET.SubElement(settings[0], '{'+SETTINGS+'}activeCodePage')
    page.text = 'UTF-8'
    for prefix, uri in [('', ASM), ('asmv3', V3), ('ws2', 'http://schemas.microsoft.com/SMI/2016/WindowsSettings'), ('ws2019', SETTINGS)]: ET.register_namespace(prefix, uri)
    return ET.tostring(root, encoding='utf-8', xml_declaration=True)


def utf8_image(data, *, subsystem=3):
    resources = read_resources(data, subsystem=subsystem, allow_empty=True)
    manifests = [key for key in resources if key[:2] == (24, 1)]
    changed = not manifests
    if not manifests:
        resources[24, 1, 1033] = ('<assembly xmlns="'+ASM+'" manifestVersion="1.0"><application xmlns="'+V3+'"><windowsSettings/></application></assembly>').encode()
        manifests = [(24, 1, 1033)]
    if len(manifests) > 8: raise ValueError('Ambiguous process manifests')
    changes = []
    for key in sorted(manifests):
        original = resources[key]; resources[key] = utf8_manifest(original)
        changed |= original != resources[key]
        changes.append({'language': key[2], 'original_sha256': hashlib.sha256(original).hexdigest(), 'utf8_sha256': hashlib.sha256(resources[key]).hexdigest()})
    if not changed: return bytes(data), changes
    tree = {}
    for (kind, identity, language), value in sorted(resources.items()): tree.setdefault(kind, {}).setdefault(identity, {})[language] = value
    return append_resources(data, tree, subsystem=subsystem), changes


def enable_utf8(executable, *, subsystem=3):
    target = Path(executable)
    result, _ = utf8_image(target.read_bytes(), subsystem=subsystem)
    target.write_bytes(result)
    verify_utf8(target, subsystem=subsystem)


def verify_utf8(executable, *, subsystem=3):
    resources = read_resources(Path(executable).read_bytes(), subsystem=subsystem)
    manifests = [value for key, value in resources.items() if key[:2] == (24, 1)]
    if not 1 <= len(manifests) <= 8: raise ValueError('Missing process UTF-8 manifest')
    for value in manifests:
        utf8_manifest(value)  # Validate bounded XML and reject ambiguous application/settings shapes.
        root = ET.fromstring(value)
        pages = root.findall('{'+V3+'}application/{'+V3+'}windowsSettings/{'+SETTINGS+'}activeCodePage')
        if len(pages) != 1 or pages[0].text != 'UTF-8': raise ValueError('Process UTF-8 code page differs')
