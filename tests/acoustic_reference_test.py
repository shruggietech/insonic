# SPDX-License-Identifier: Apache-2.0
"""Independent reference and provenance contracts, without inference imports."""
import copy
import importlib.util
import json
from pathlib import Path
import shutil
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
CORPUS = ROOT / "tests/fixtures/acoustic"


def load():
    spec = importlib.util.spec_from_file_location("acoustic_reference_tested", ROOT / "scripts/acoustic_reference.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class CorpusTests(unittest.TestCase):
    def setUp(self):
        self.reference = load()

    def test_official_frozen_corpus_has_disjoint_readers_and_roles(self):
        corpus = self.reference.load_corpus(CORPUS / "manifest.json")
        self.assertEqual(len(corpus["clips"]), 5)
        self.assertEqual({c["speaker"] for c in corpus["clips"] if c["role"] == "enrollment"}, {"84", "174"})
        enrolled = {c["archive_member"] for c in corpus["clips"] if c["role"] == "enrollment"}
        self.assertFalse(enrolled & {c["archive_member"] for c in corpus["clips"] if c["role"] != "enrollment"})
        self.assertEqual(corpus["source"]["sha256"], "76f87d090650617fca0cac8f88b9416e0ebf80350acb97b343a85fa903728ab3")

    def mutate(self, action):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary) / "corpus"
            shutil.copytree(CORPUS, directory)
            manifest = json.loads((directory / "manifest.json").read_text(encoding="utf-8"))
            action(manifest, directory)
            (directory / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
            return self.reference.load_corpus(directory / "manifest.json")

    def test_changed_original_bytes_are_rejected(self):
        def change(c, directory):
            (directory / c["clips"][0]["path"]).write_bytes(b"altered original")
        with self.assertRaisesRegex(ValueError, "identity|hash|size"):
            self.mutate(change)

    def test_reference_text_cannot_be_replaced_by_prediction(self):
        with self.assertRaisesRegex(ValueError, "reference"):
            self.mutate(lambda c, _: c["clips"][0].update(transcript="replacement model prediction"))

    def test_role_overlap_and_duplicate_sources_are_rejected(self):
        def overlap(c, _):
            c["clips"][1]["archive_member"] = c["clips"][0]["archive_member"]
        with self.assertRaisesRegex(ValueError, "source|duplicate|overlap"):
            self.mutate(overlap)

    def test_escape_and_unknown_manifest_fields_are_rejected(self):
        with self.assertRaisesRegex(ValueError, "path"):
            self.mutate(lambda c, _: c["clips"][0].update(path="../outside.flac"))
        with self.assertRaisesRegex(ValueError, "field|shape"):
            self.mutate(lambda c, _: c.update(unexpected=True))

    def test_sample_identity_must_match_flac_header(self):
        with self.assertRaisesRegex(ValueError, "sample|identity"):
            self.mutate(lambda c, _: c["clips"][0].update(samples=c["clips"][0]["samples"] + 1))

    def test_nonfinite_or_changed_frozen_settings_are_rejected(self):
        for value in (float("nan"), -1, True):
            with self.subTest(value=value), self.assertRaisesRegex(ValueError, "setting|matching"):
                self.mutate(lambda c, _: c["matching"].update(threshold=value))

    def test_duplicate_json_keys_are_not_silently_last_wins(self):
        with tempfile.TemporaryDirectory() as temporary:
            target = Path(temporary) / "manifest.json"
            target.write_text('{"format":"wrong","format":"insonic-acoustic-reference-v1"}', encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "(?i)duplicate"):
                self.reference.load_corpus(target)


class ComparisonTests(unittest.TestCase):
    def setUp(self):
        self.reference = load()

    def test_edit_components_and_denominator_are_independent(self):
        actual = self.reference.word_counts("A b c", "a x c d")
        self.assertEqual({k: actual[k] for k in ("substitutions", "deletions", "insertions", "edit_distance", "reference_words")},
                         {"substitutions": 1, "deletions": 0, "insertions": 1, "edit_distance": 2, "reference_words": 3})
        self.assertAlmostEqual(actual["word_error_rate"], 2 / 3)

    def test_empty_prediction_and_reference_have_explicit_denominators(self):
        missing = self.reference.word_counts("one two", "")
        self.assertEqual((missing["deletions"], missing["edit_distance"], missing["reference_words"]), (2, 2, 2))
        no_reference = self.reference.word_counts("", "unexpected")
        self.assertEqual(no_reference["insertions"], 1)
        self.assertIsNone(no_reference["word_error_rate"])

    def test_frozen_unicode_normalization_and_word_bound(self):
        self.assertEqual(self.reference.word_counts("CAFÉ, Straße!", "Cafe\u0301 strasse")["edit_distance"], 0)
        with self.assertRaisesRegex(ValueError, "limit"):
            self.reference.word_counts("word " * 8193, "")

    def test_work_acceptance_is_not_named_speaker_accuracy(self):
        enrolled = {"84": "speaker-a", "174": "speaker-b"}
        self.assertEqual(self.reference.compare_decision("84", {"state": "accepted", "speaker_id": "speaker-a", "score": .9}, enrolled)["outcome"], "correct_match")
        self.assertEqual(self.reference.compare_decision("84", {"state": "unknown", "score": .67}, enrolled)["outcome"], "unknown")
        self.assertEqual(self.reference.compare_decision("84", {"state": "accepted", "speaker_id": "speaker-b", "score": .9}, enrolled)["outcome"], "incorrect")
        with self.assertRaisesRegex(ValueError, "decision"):
            self.reference.compare_decision("84", {"state": "accepted", "decisions": [{"state": "unknown"}]}, enrolled)

    def test_unseen_ambiguity_and_invalid_score_are_explicit(self):
        enrolled = {"84": "speaker-a", "174": "speaker-b"}
        self.assertEqual(self.reference.compare_decision("251", {"state": "unknown", "score": .2}, enrolled)["outcome"], "correct_unknown")
        self.assertEqual(self.reference.compare_decision("84", {"state": "ambiguous", "score": .8}, enrolled)["outcome"], "ambiguous")
        with self.assertRaisesRegex(ValueError, "score"):
            self.reference.compare_decision("84", {"state": "unknown", "score": float("nan")}, enrolled)

    def test_constructed_windows_do_not_collapse_mixed_reference_voice(self):
        windows = [{"start_us": 0, "end_us": 100, "speaker": "84"},
                   {"start_us": 150, "end_us": 250, "speaker": "174"}]
        result = self.reference.reference_readers([(0, 80), (50, 100), (150, 200)], windows)
        self.assertEqual(result, {"84": 100, "174": 50})
        self.assertEqual(self.reference.reference_readers([(100, 150)], windows), {})
        with self.assertRaisesRegex(ValueError, "interval"):
            self.reference.reference_readers([(10, 5)], windows)


if __name__ == "__main__":
    unittest.main()
