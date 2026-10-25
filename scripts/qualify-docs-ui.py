#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Run documentation UI qualification with the existing discovered hidden browser."""
import argparse
import importlib.util
import json
from pathlib import Path
import tempfile
import qualify
import qualification_timing as timing

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', type=Path, default=ROOT / 'build/docs-ui')
    options = parser.parse_args()
    spec = importlib.util.spec_from_file_location('docs_renderer', ROOT / 'scripts/qualify-docs-renderer.py')
    renderer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(renderer)
    output = qualify.child(['node', ROOT / 'scripts/check-docs-ui.mjs', '--browser', renderer.browser_executable(),
                            '--out', ROOT / 'site/out', '--offline', ROOT / 'site/offline', '--output', options.output],
                           timeout=timing.DOCS_BROWSER_SECONDS + timing.CLEANUP_SECONDS)
    print(output.decode('utf-8'))
    report = json.loads((options.output / 'report.json').read_text(encoding='utf-8'))
    fixture = report.get('fixture', {})
    allocated = Path(fixture.get('path', ''))
    parent = Path(tempfile.gettempdir()).resolve()
    if (not report.get('passed') or not fixture.get('removed') or not fixture.get('non_root')
            or allocated.parent != parent or not allocated.name.startswith('insonic-docs-ui-')
            or allocated.exists() or allocated.is_symlink()):
        raise ValueError('Documentation fixture confinement or post-run absence was not verified')


if __name__ == '__main__':
    main()
