#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Installed-output validator rejects the observed duplicate/recovery defects."""
import sys
import unittest
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from cli_operator_qualification import validate_work_outcome


class WorkOutcomeTests(unittest.TestCase):
    def test_complete_outcome_and_readback(self):
        validate_work_outcome('Work: example\nState: Succeeded\nComplete JSON: insonic work show example --json\n',
                              'example', 'succeeded')

    def test_rejects_observed_duplicates_empty_error_and_invalid_actions(self):
        text = 'Work: example\nState: Succeeded\nComplete JSON: insonic work show example --json\n'
        for detail in ['State: Succeeded\n', 'State: ready\n', 'Error: \n',
                       'Wait: insonic work wait example\n', 'Cancel: insonic work cancel example\n',
                       'Generation: 1\n', 'Payload: omitted from human output.\n', 'Provenance:\n', 'Result:\n']:
            with self.subTest(detail=detail), self.assertRaises(ValueError):
                validate_work_outcome(text + detail, 'example', 'succeeded')

    def test_recording_outcomes_are_distinct_from_successful_work(self):
        base = 'Work: example\nState: Succeeded\nComplete JSON: insonic work show example --json\n'
        for outcome in ['ready', 'no-speech', 'no-timed-subtitles']:
            line = 'Outcome: ' + outcome.replace('-', ' ').capitalize() + '\n'
            with self.subTest(outcome=outcome):
                validate_work_outcome(base + line, 'example', 'succeeded', recording_outcome=outcome)
                for text in [base, base + line + line, base + 'Outcome: Failed\n']:
                    with self.assertRaises(ValueError):
                        validate_work_outcome(text, 'example', 'succeeded', recording_outcome=outcome)

    def test_missing_complete_readback_is_not_acceptance(self):
        with self.assertRaises(ValueError):
            validate_work_outcome('State: Succeeded\n', 'example', 'succeeded')


if __name__ == '__main__':
    unittest.main()
