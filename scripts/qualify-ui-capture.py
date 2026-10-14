#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Qualify real application rendering in an isolated headless browser."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import sys
import qualify

ROOT = Path(__file__).resolve().parents[1]

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', type=Path, default=ROOT / 'build/ui-capture')
    options = parser.parse_args()
    values = qualify.native_env()
    suffix = '.exe' if os.name == 'nt' else ''
    cli = ROOT / ('build/insonic' + suffix)
    if not cli.is_file():
        qualify.build_cli(True, values)
    helper = ROOT / ('build/insonic-capture-bridge' + suffix)
    qualify.child(['go', 'build', '-tags', 'system_ladybug', '-o', helper, './cmd/insonic-capture-bridge'], env=values)
    spec = importlib.util.spec_from_file_location('docs_renderer', ROOT / 'scripts/qualify-docs-renderer.py')
    renderer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(renderer)
    browser = renderer.browser_executable()
    values['INSONIC_CAPTURE_NATIVE_PATH'] = os.pathsep.join(str(ROOT / p) for p in ['build/native/ladybug', 'build/native/openssl'])
    output = qualify.child(['node', ROOT / 'scripts/capture-application-ui.mjs', '--cli', cli, '--bridge', helper, '--browser', browser, '--output', options.output], env=values, timeout=150)
    print(output.decode('utf-8'))
    receipt = json.loads((options.output / 'application-capture.json').read_text(encoding='utf-8'))
    if receipt['source']['dirty'] or receipt['source']['revision'] != os.environ.get('SOURCE_REVISION'):
        raise ValueError('CI capture requires clean exact source identity')

if __name__ == '__main__':
    main()
