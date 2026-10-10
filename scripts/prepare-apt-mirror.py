#!/usr/bin/env python3
"""Prefer Ubuntu HTTPS while retaining runner repositories and signature policy."""
from pathlib import Path
import re


def prepare(root=Path('/etc/apt')):
    root = Path(root)
    paths = [root / 'sources.list', root / 'apt-mirrors.txt',
             *(root / 'sources.list.d').glob('*.sources'),
             *(root / 'sources.list.d').glob('*.list')]
    changed = 0
    for path in paths:
        if not path.is_file():
            continue
        original = path.read_text(encoding='utf-8')
        updated = re.sub(r'https?://azure[.]archive[.]ubuntu[.]com/ubuntu(?=/|\s|$)',
                         'https://archive.ubuntu.com/ubuntu', original)
        if updated != original:
            path.write_text(updated, encoding='utf-8', newline='\n')
            changed += 1
    return changed


if __name__ == '__main__':
    print(f'Prepared Ubuntu HTTPS repository selection in {prepare()} apt files')
