# SPDX-License-Identifier: Apache-2.0
"""Deterministic bounded PE resources shared by setup and companion packaging."""
import struct


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


def read_resources(data, *, subsystem=2):
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


