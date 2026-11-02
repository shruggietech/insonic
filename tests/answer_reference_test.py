# SPDX-License-Identifier: Apache-2.0
import copy
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
import answer_reference as reference


class AnswerReferenceTests(unittest.TestCase):
    def setUp(self):
        self.corpus, self.digest = reference.load_corpus()
        self.case = self.corpus['cases'][0]
        self.recordings = {'plan-a': 'a', 'plan-b': 'b'}
        ref = {'recording_id': 'a', 'cue_id': 'cue', 'source': {'start_us': '0', 'end_us': '1000000'}}
        self.package = {'id': 'p', 'items': [{'id': 'e', 'reference': ref, 'text': 'Launch is on Monday.', 'family': 'one'},
                                           {'id': 'f', 'reference': {'recording_id': 'b'}, 'text': 'Launch is on Monday.', 'family': 'two'}],
                        'independent_families': 2, 'context': {'nodes': [{'level': 'cue', 'id': 'e', 'text': 'Launch is on Monday.'}]}}
        self.answer = {'package_id': 'p', 'state': 'partial', 'claims': [{'id': 'c', 'kind': 'quote', 'text': 'Launch is on Monday.', 'support': 'supported', 'citations': [{'evidence_id': 'e', 'reference': ref, 'quote': 'Launch is on Monday.'}]}]}

    def test_true_reference_cannot_support_false_claim(self):
        good = reference.score_case(self.case, self.recordings, self.package, self.answer)
        self.assertTrue(good['passed'])
        self.answer['claims'][0].update(kind='paraphrase', text='Launch is on Friday.')
        bad = reference.score_case(self.case, self.recordings, self.package, self.answer)
        self.assertFalse(bad['passed'])
        self.assertEqual(bad['metrics']['citation_validity']['value'], 1)
        self.assertEqual(bad['metrics']['unsupported_claims']['numerator'], 1)
        self.assertEqual(bad['metrics']['support_coverage']['value'], 0)
        self.assertTrue(bad['failures'])

    def test_duplicates_do_not_increase_support_or_independence(self):
        case = self.corpus['cases'][1]
        recordings = {'plan-a': 'a', 'plan-copy': 'b'}
        self.package['items'][1]['family'] = 'one'
        self.package['independent_families'] = 1
        before = reference.score_case(case, recordings, self.package, self.answer)
        self.package['items'].append(copy.deepcopy(self.package['items'][0]))
        after = reference.score_case(case, recordings, self.package, self.answer)
        self.assertTrue(before['passed'])
        self.assertEqual(before['metrics'], after['metrics'])
        self.package['independent_families'] = 2
        self.assertFalse(reference.score_case(case, recordings, self.package, self.answer)['passed'])

    def test_withheld_and_unknown_paraphrases_fail_conservative_rubric(self):
        case = self.corpus['cases'][5]
        self.package['items'] = self.package['items'][:1]
        self.package['independent_families'] = 1
        self.assertFalse(reference.score_case(case, {'plan-a': 'a'}, self.package, self.answer)['passed'])
        self.answer['claims'] = [{'kind': 'paraphrase', 'text': 'Unknown', 'support': 'unsupported', 'uncertainty': 'No budget evidence'}]
        out = reference.score_case(case, {'plan-a': 'a'}, self.package, self.answer)
        self.assertTrue(out['passed'])
        self.assertIsNone(out['metrics']['citation_validity']['value'])
        self.assertEqual(out['metrics']['citation_validity']['reason'], 'not-applicable')
        self.answer['claims'] = [{'kind': 'paraphrase', 'text': 'A day at the start of the week', 'support': 'supported', 'citations': []}]
        self.assertFalse(reference.score_case(self.case, self.recordings, self.package, self.answer)['passed'])

    def test_manifest_duplicate_keys_unknown_fields_and_escape_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = root / 'manifest.json'
            path.write_text('{"kind":"answer-evaluation-corpus","kind":"other"}', encoding='utf-8')
            with self.assertRaises(ValueError): reference.load_corpus(path, root=root)
            value = copy.deepcopy(self.corpus)
            value['sources'][0]['file'] = '../private'
            path.write_text(json.dumps(value), encoding='utf-8')
            with self.assertRaises(ValueError): reference.load_corpus(path, root=root)
            with self.assertRaises(ValueError): reference.load_corpus(ROOT / 'tests/fixtures/answers/manifest.json', root=root)

    def test_fixture_report_cannot_claim_model_quality(self):
        with self.assertRaises(ValueError): reference.make_report(self.corpus, self.digest, [], mode='elected-model', election=None)
        out = reference.make_report(self.corpus, self.digest, [], mode='deterministic', election=None)
        self.assertEqual(out['model_quality'], 'not-evaluated')
        self.assertFalse(out['passed'])


if __name__ == '__main__':
    unittest.main()
