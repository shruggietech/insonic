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
    density = json.loads((options.output / 'density-capture.json').read_text(encoding='utf-8'))
    matrix = density.get('verification', {}).get('compact_density', {})
    scenarios = matrix.get('scenarios', [])
    expected = {(view, width, scale, text) for view in ['Setup', 'Library', 'Jobs', 'Pipelines', 'Models', 'Settings']
                for width in [1200, 720] for scale in [1, 1.25, 1.5, 2] for text in [1, 2]}
    observed = {(item['view'], item['width'], item['device_scale_factor'], item['text_scale']) for item in scenarios}
    if any(item.get('keyboard_focus') != 'passed' or item.get('selection_activation') not in ['not-present', 'label-padding-and-space-passed'] for item in scenarios):
        raise ValueError('Mounted workflow interaction proof is incomplete')
    if matrix.get('state') != 'passed' or len(scenarios) != 96 or observed != expected or len(density.get('images', [])) != 7:
        raise ValueError('Desktop density matrix is incomplete')
    if density['source'] != receipt['source'] or density['verification'].get('cleanup_complete') is not True:
        raise ValueError('Density capture source or cleanup differs')
    if receipt['source']['dirty'] or receipt['source']['revision'] != os.environ.get('SOURCE_REVISION'):
        raise ValueError('CI capture requires clean exact source identity')

if __name__ == '__main__':
    main()
