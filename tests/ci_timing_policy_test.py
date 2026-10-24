# SPDX-License-Identifier: Apache-2.0
import sys
from pathlib import Path
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
import ci_timing_policy as policy


class AggregateTimingPolicyTests(unittest.TestCase):
    def test_expanded_policy_and_invalid_contracts(self):
        good = {'qualification_target_seconds': 1800, 'qualification_limit_seconds': 3600, 'finalization_reserve_seconds': 60}
        self.assertEqual(policy.validate_policy(good), good)
        for invalid in [None, [], {}, {**good, 'extra': 1},
                        {**good, 'qualification_target_seconds': True},
                        {**good, 'qualification_limit_seconds': 0},
                        {**good, 'qualification_target_seconds': 3601},
                        {**good, 'finalization_reserve_seconds': 3600},
                        {**good, 'qualification_limit_seconds': float('inf')}]:
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                policy.validate_policy(invalid)

    def test_duplicate_authority_is_rejected_before_last_value_wins(self):
        with self.assertRaises(ValueError):
            policy.load_policy('{"qualification_target_seconds":1800,"qualification_target_seconds":1,"qualification_limit_seconds":3600,"finalization_reserve_seconds":60}')


if __name__ == '__main__':
    unittest.main()
