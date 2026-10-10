#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Maintainer entry point for exact corresponding-source media builds."""
import json
from media_source_build import *

if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--stage', choices=['dependencies', 'complete', *DEPENDENCY_GROUPS], default='complete')
    directory, receipt, source = prepare(stage=parser.parse_args().stage)
    print(json.dumps(receipt))
