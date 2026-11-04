# SPDX-License-Identifier: Apache-2.0
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
import speaker_reference as reference


class SpeakerReferenceTests(unittest.TestCase):
    def setUp(self):
        self.corpus, self.digest = reference.load_corpus()

    def rows(self):
        return [{'id': case['id'], 'state': 'completed',
                 'observations': copy.deepcopy(case['expected'])} for case in self.corpus['cases']]

    def test_perfect_observations_account_for_every_voice(self):
        score = reference.score(self.corpus, self.rows())
        self.assertEqual(score['cases'], {'total': 8, 'completed': 8, 'errors': 0})
        self.assertEqual(score['identity_precision'], {'numerator': 3, 'denominator': 3, 'value': 1.0})
        self.assertEqual(score['identity_coverage']['value'], 1.0)
        self.assertEqual(score['outcome_accuracy']['denominator'], 9)
        self.assertEqual(score['outcome_counts']['ambiguous'], 2)
        self.assertEqual(score['unregistered'], 1)

    def test_false_identity_counts_against_precision(self):
        rows = self.rows()
        rows[-1]['observations'][0].update(outcome='supported-contextual', speaker='alex', identity_status='registered')
        score = reference.score(self.corpus, rows)
        self.assertEqual(score['identity_precision'], {'numerator': 3, 'denominator': 4, 'value': .75})
        self.assertEqual(score['outcome_accuracy']['numerator'], 8)

    def test_wrong_unregistered_name_is_not_a_correct_outcome(self):
        rows = self.rows()
        row = next(item for item in rows if item['id'] == 'unregistered-introduction')
        row['observations'][0]['name'] = 'Another person'
        score = reference.score(self.corpus, rows)
        self.assertEqual(score['outcome_accuracy']['numerator'], 8)
        self.assertEqual(score['outcome_accuracy']['denominator'], 9)
        row['observations'][0]['name'] = None
        with self.assertRaises(ValueError): reference.score(self.corpus, rows)

    def test_provider_error_remains_in_coverage_and_outcome_denominators(self):
        rows = self.rows()
        rows[0].update(state='error', observations=[], error='unavailable')
        score = reference.score(self.corpus, rows)
        self.assertEqual(score['cases']['errors'], 1)
        self.assertEqual(score['identity_coverage'], {'numerator': 2, 'denominator': 3, 'value': 2 / 3})
        self.assertEqual(score['outcome_accuracy']['denominator'], 9)
        self.assertEqual(score['outcome_accuracy']['numerator'], 8)

    def test_no_accepted_identities_has_undefined_precision(self):
        rows = self.rows()
        for row in rows:
            for voice in row['observations']:
                voice.update(outcome='unknown', speaker=None, identity_status='unknown', name=None)
        score = reference.score(self.corpus, rows)
        self.assertEqual(score['identity_precision'], {'numerator': 0, 'denominator': 0, 'value': None})
        self.assertEqual(score['identity_coverage']['value'], 0)

    def test_missing_duplicate_or_unexpected_case_is_not_success(self):
        rows = self.rows()
        for broken in (rows[:-1], rows + [rows[0]], rows[:-1] + [dict(rows[-1], id='invented')]):
            with self.assertRaises(ValueError): reference.score(self.corpus, broken)

    def test_missing_duplicate_or_invented_voice_is_rejected(self):
        for observations in ([], [self.rows()[0]['observations'][0]] * 2,
                             [dict(self.rows()[0]['observations'][0], voice='invented')]):
            rows = self.rows()
            rows[0]['observations'] = observations
            with self.assertRaises(ValueError): reference.score(self.corpus, rows)

    def test_registered_identity_requires_a_key(self):
        rows = self.rows()
        rows[0]['observations'][0]['speaker'] = None
        with self.assertRaises(ValueError): reference.score(self.corpus, rows)

    def test_truth_must_cover_cues_and_registered_identities(self):
        for mutation in ('voice', 'speaker', 'duplicate'):
            corpus = copy.deepcopy(self.corpus)
            if mutation == 'voice': corpus['cases'][0]['expected'][0]['voice'] = 'missing'
            elif mutation == 'speaker': corpus['cases'][0]['expected'][0]['speaker'] = 'absent'
            else: corpus['cases'].append(corpus['cases'][0])
            with self.assertRaises(ValueError): reference.validate_corpus(corpus)

    def test_duplicate_json_keys_and_substituted_frozen_corpus_are_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / 'manifest.json'
            path.write_text('{"version":1,"version":2}', encoding='utf-8')
            with self.assertRaises(ValueError): reference.load_corpus(path)
            changed = copy.deepcopy(self.corpus)
            changed['cases'][0]['cues'][0]['text'] += ' Altered.'
            path.write_text(json.dumps(changed), encoding='utf-8')
            with self.assertRaises(ValueError): reference.load_corpus(path)

    def test_metrics_reject_invalid_counts(self):
        for numerator, denominator in ((-1, 2), (2, 1), (True, 1), (1, 0)):
            with self.assertRaises(ValueError): reference.metric(numerator, denominator)


if __name__ == '__main__':
    unittest.main()
