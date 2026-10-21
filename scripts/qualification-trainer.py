#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Tiny deterministic adapter fixture. No model inference or training occurs."""
import argparse
import hashlib
import json
from pathlib import Path
import sys
import time

WEIGHTS = b'synthetic local trainer fixture bytes'


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', choices=['success', 'wrong-hash', 'truncated', 'unknown', 'delayed'], default='success')
    mode = parser.parse_args().mode
    request = json.load(sys.stdin)
    if request['operation'] != 'train':
        raise ValueError('fixture requires the train operation')
    if mode == 'delayed':
        time.sleep(3)
    path = Path(request['output_directory']) / 'weights.bin'
    path.write_bytes(WEIGHTS[:-1] if mode == 'truncated' else WEIGHTS)
    output = {'contract_version': '1', 'kind': 'weights', 'architecture': 'custom',
              'consumers': ['custom'], 'supported_operations': ['custom-inference'],
              'artifacts': [{'role': 'weights.bin', 'path': 'weights.bin', 'format': 'binary',
                             'size': len(WEIGHTS), 'sha256': 'f' * 64 if mode == 'wrong-hash' else hashlib.sha256(WEIGHTS).hexdigest()}]}
    if mode == 'unknown':
        output['unexpected'] = 'fixture'
    print(json.dumps(output))


if __name__ == '__main__':
    main()
