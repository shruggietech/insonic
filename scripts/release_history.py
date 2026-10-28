# SPDX-License-Identifier: Apache-2.0
"""Immutable original-release data authority, implemented by current tooling.

No version range authorizes a historical exemption. Only the reviewed original
manifest and complete selected source can create the original-v1.0.1 profile.
The pinned MSI digest covers its original tables and embedded streams without
executing historical authoring code or substituting the current MSI model.
"""
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import subprocess

import qualification_timing as timing

ROOT = Path(__file__).resolve().parents[1]
ORIGINAL_REVISION = '136f5c0307a6fce00ba8240ef69c907a944cd8c0'
ORIGINAL_TREE = '1c4ae59c053a3d8d751cde5c736f67f2bbeecd3c'
ORIGINAL_MANIFEST = 'ce885efde09f7ba9d4b12d3b60ec9e87f918921e2db5fcc6ec2021fab743b8d8'
ORIGINAL_MSI = 'beff9343561e160e157ed2ce3f8f41a287b6bdd9f88beec0779ae408fceae22d'
ORIGINAL_MSI_SIZE = 137895936
ORIGINAL_V102_MANIFEST = '02a55b6ec51855881a1fc7592b002c5d3ce5d5dce0bf09985094da27beee3449'
ORIGINAL_V102_PACKAGES = frozenset((
    '0f8e936f6689ef7d268640b93c19f98090d0306f7b8dc47dd143fe1d1a47d8a2',
    '72fcff4fd5efb4f01b2c42edadf93b5b3c9aebe7e1d24d4f712bb541cf2b7714',
    'ca15fcad7d64f18a24ec87ee649ebbd7c85c93257b12ce8320ba0acfbe50b6fc',
    'c711023426571ef6798de9cc1507aa90c949e18ba3610e2589cb03d92f2002f5',
))


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(',', ':')).encode('utf-8')).hexdigest()


def original_recovery_absence(receipt):
    """Only original v1.0.2 bytes may predate the new recovery evidence.

    This exempts one new field, never existing source/archive/signing/terminal
    or installer checks. Every receipt field, including revision and nested
    inventories, is anchored by the immutable publication's canonical digest.
    """
    return digest(receipt) in ORIGINAL_V102_PACKAGES


def git_metadata(args, repository):
    result = subprocess.run(['git', '--no-replace-objects', *args], cwd=repository,
        stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        timeout=timing.SUBPROCESS_SECONDS,
        creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0,
        env={**os.environ, 'GIT_TERMINAL_PROMPT': '0', 'GH_PROMPT_DISABLED': '1', 'GIT_OPTIONAL_LOCKS': '0'})
    if result.returncode:
        raise ValueError('Original release source metadata is unavailable')
    return result.stdout.decode('utf-8')


def verify_source(source_root, *, repository=ROOT, git=git_metadata):
    """Check every pinned tracked file; source scripts remain inert data."""
    root = Path(source_root).resolve()
    tree = git(['rev-parse', ORIGINAL_REVISION + '^{tree}'], repository).strip()
    if tree != ORIGINAL_TREE:
        raise ValueError('Original release source tree identity differs')
    inventory = git(['ls-tree', '-rz', '--full-tree', ORIGINAL_REVISION], repository)
    records = inventory.split('\0')
    if not records or records[-1] != '' or len(records) > 50001:
        raise ValueError('Original release source inventory is incomplete')
    seen = set()
    for record in records[:-1]:
        metadata, separator, name = record.partition('\t')
        parts = metadata.split(' ')
        path = PurePosixPath(name)
        if (not separator or len(parts) != 3 or parts[0] not in ('100644', '100755')
                or parts[1] != 'blob' or not re.fullmatch('[0-9a-f]{40}', parts[2])
                or path.is_absolute() or not name or path.as_posix() != name
                or any(piece in ('..', '.') for piece in path.parts) or '\\' in name
                or ':' in name or name.casefold() in seen):
            raise ValueError('Original release source inventory member is invalid')
        seen.add(name.casefold())
        selected = root.joinpath(*path.parts)
        try:
            for ancestor in [selected, *selected.parents]:
                if ancestor == root: break
                if ancestor.is_symlink() or (hasattr(ancestor, 'is_junction') and ancestor.is_junction()):
                    raise ValueError('Original release source member is linked')
            info = selected.lstat()
            if not stat.S_ISREG(info.st_mode):
                raise ValueError('Original release source member is not regular')
            blob = hashlib.sha1(b'blob ' + str(info.st_size).encode() + b'\0')
            with selected.open('rb') as stream:
                while chunk := stream.read(1 << 20): blob.update(chunk)
            if blob.hexdigest() != parts[2]:
                raise ValueError('Original release source bytes differ: ' + name)
        except OSError as error:
            raise ValueError('Original release source member is missing or unreadable: ' + name) from error
    if not seen:
        raise ValueError('Original release source inventory is empty')
    return {'revision': ORIGINAL_REVISION, 'tree': ORIGINAL_TREE, 'files': len(seen)}


class OriginalProfile:
    version = '1.0.1'
    revision = ORIGINAL_REVISION

    def __init__(self, manifest, source_root):
        self.source_root = Path(source_root).resolve()
        self.packages = frozenset(digest(receipt) for receipt in manifest['packages'])
        self.installer = digest(manifest['installers'][0])

    def verify_receipt(self, receipt):
        selected = digest(receipt)
        if (receipt.get('kind') == 'native-package-receipt' and selected in self.packages):
            return
        if receipt.get('kind') == 'windows-installer-receipt' and selected == self.installer:
            return
        raise ValueError('Original release receipt differs from immutable publication')

    def verify_installer(self, path, receipt, source_root):
        if Path(source_root).resolve() != self.source_root:
            raise ValueError('Original installer selected source differs from profile')
        self.verify_receipt(receipt)
        verify_source(self.source_root)
        path = Path(path)
        if path.name != 'insonic_1.0.1_windows_amd64_installer.msi' or path.stat().st_size != ORIGINAL_MSI_SIZE:
            raise ValueError('Original MSI artifact identity differs')
        with path.open('rb') as stream:
            if hashlib.file_digest(stream, 'sha256').hexdigest() != ORIGINAL_MSI:
                raise ValueError('Original MSI bytes/tables differ from immutable publication')
        return {'inventory': receipt['payload_inventory'], 'profile': 'original-v1.0.1', 'msi_sha256': ORIGINAL_MSI}


def select_profile(value, manifest_path, source_root):
    if (value.get('version'), value.get('revision')) != ('1.0.1', ORIGINAL_REVISION):
        return None
    raw = Path(manifest_path).read_bytes()
    if hashlib.sha256(raw).hexdigest() != ORIGINAL_MANIFEST or json.loads(raw) != value:
        raise ValueError('Original release manifest differs from immutable publication')
    verify_source(source_root)
    return OriginalProfile(value, source_root)
