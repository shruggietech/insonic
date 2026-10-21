# SPDX-License-Identifier: Apache-2.0
"""Shared, committed CI qualification thresholds (not runtime configuration)."""
import json
from pathlib import Path

POLICY = json.loads(Path(__file__).with_name('ci-timing-policy.json').read_text(encoding='utf-8'))
QUALIFICATION_TARGET_SECONDS = POLICY['qualification_target_seconds']
QUALIFICATION_LIMIT_SECONDS = POLICY['qualification_limit_seconds']
FINALIZATION_RESERVE_SECONDS = POLICY['finalization_reserve_seconds']
