# SPDX-License-Identifier: Apache-2.0
"""Validated committed CI thresholds, separate from product configuration."""
import json
from pathlib import Path


def validate_policy(value):
    keys = {'qualification_target_seconds', 'qualification_limit_seconds', 'finalization_reserve_seconds'}
    if (not isinstance(value, dict) or set(value) != keys
            or any(type(value[key]) is not int or not 1 <= value[key] <= 2147483 for key in keys)
            or value['qualification_target_seconds'] > value['qualification_limit_seconds']
            or value['finalization_reserve_seconds'] >= value['qualification_limit_seconds']):
        raise ValueError('Invalid aggregate qualification timing policy')
    return dict(value)


def load_policy(raw):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError('Duplicate aggregate timing field')
            result[key] = value
        return result
    return validate_policy(json.loads(raw, object_pairs_hook=unique))


POLICY = load_policy(Path(__file__).with_name('ci-timing-policy.json').read_text(encoding='utf-8'))
QUALIFICATION_TARGET_SECONDS = POLICY['qualification_target_seconds']
QUALIFICATION_LIMIT_SECONDS = POLICY['qualification_limit_seconds']
FINALIZATION_RESERVE_SECONDS = POLICY['finalization_reserve_seconds']
