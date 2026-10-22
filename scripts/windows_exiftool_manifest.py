#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Reproduce the pinned Windows ExifTool launcher's UTF-8 process manifest."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import struct
from pe_resources import utf8_image

UPSTREAM_SHA256 = '68c079c32fdae0d6c7130e9a5fb73f8ac9dabdf9ab8da312da4f6c549d6d3385'
ORIGINAL = 'exiftool-launcher.original.bin'
RECEIPT = 'exiftool-windows-utf8.json'
RECIPES = ['windows_exiftool_manifest.py', 'pe_resources.py']
TARGET = 'companions/media/exiftool/package/bin/exiftool.exe'


def transform(data, *, expected_sha256=UPSTREAM_SHA256):
    if len(data) > 2 << 20 or hashlib.sha256(data).hexdigest() != expected_sha256: raise ValueError('Upstream launcher identity differs')
    result, changes = utf8_image(data)
    return result, {'kind': 'windows-exiftool-utf8', 'version': 1, 'upstream_sha256': expected_sha256,
                    'unsigned_sha256': hashlib.sha256(result).hexdigest(), 'manifests': changes}


def unsigned_image(data):
    if len(data) < 64 or len(data) > 128 << 20 or data[:2] != b'MZ': raise ValueError('Invalid launcher PE bound')
    data = bytearray(data)
    pe = struct.unpack_from('<I', data, 60)[0]; optional = pe + 24
    if pe + 24 > len(data) or data[pe:pe+4] != b'PE\0\0': raise ValueError('Invalid launcher PE signature')
    size = struct.unpack_from('<H', data, pe+20)[0]
    if size < 240 or optional+size > len(data) or struct.unpack_from('<H',data,optional)[0] != 0x20b: raise ValueError('Invalid launcher optional header')
    certificate = optional+112+32; start, length = struct.unpack_from('<II', data, certificate)
    if start or length:
        if not start or not length or start % 8 or start < optional+size or start+length != len(data): raise ValueError('Invalid launcher certificate envelope')
        data = data[:start]
    struct.pack_into('<I', data, optional+64, 0);struct.pack_into('<II', data, certificate, 0, 0)
    return bytes(data)


def verify_transform(original, actual, receipt, *, expected_sha256=UPSTREAM_SHA256, allow_signature=False):
    expected, identity = transform(original, expected_sha256=expected_sha256)
    matches = unsigned_image(actual) == unsigned_image(expected) if allow_signature else actual == expected
    if receipt != identity or not matches: raise ValueError('Packaged launcher differs from its exact UTF-8 transformation')


def prepare(source, output):
    source, output = Path(source), Path(output)
    if source.is_symlink() or source.stat().st_size > 2 << 20 or source.resolve() == output.resolve() or (output.exists() and os.path.samefile(source, output)):
        raise ValueError('Launcher transformation requires a separate verified copy')
    original = source.read_bytes(); result, receipt = transform(original)
    if output.exists() and (output.is_symlink() or output.stat().st_size != len(original) or output.read_bytes() != original): raise ValueError('Launcher output is not the verified upstream copy')
    output.write_bytes(result)
    verify_transform(original, output.read_bytes(), receipt)
    return receipt


def verify_package(root, *, allow_signature=False):
    root = Path(root); sources = root/'sources'
    if (sources/ORIGINAL).stat().st_size > 2 << 20 or (root/TARGET).stat().st_size > 16 << 20 or (sources/RECEIPT).stat().st_size > 8192: raise ValueError('Packaged launcher bound exceeded')
    receipt = json.loads((sources/RECEIPT).read_text(encoding='utf-8'))
    verify_transform((sources/ORIGINAL).read_bytes(), (root/TARGET).read_bytes(), receipt, allow_signature=allow_signature)


def main(argv=None):
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('source');parser.add_argument('output');parser.add_argument('--receipt',required=True);args=parser.parse_args(argv)
    paths = [Path(value) for value in [args.source, args.output, args.receipt]]
    for index, left in enumerate(paths):
        for right in paths[index+1:]:
            if left.resolve() == right.resolve() or (left.exists() and right.exists() and os.path.samefile(left, right)):
                raise ValueError('Launcher source, output and receipt must be separate files')
    receipt=prepare(args.source,args.output)
    Path(args.receipt).write_text(json.dumps(receipt,sort_keys=True,indent=2)+'\n',encoding='utf-8',newline='\n')


if __name__ == '__main__':
    main()
