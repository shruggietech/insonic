# SPDX-License-Identifier: Apache-2.0
"""Restored inference evidence must prove fresh work, not reused documents."""
import copy
import hashlib
import json
import unittest
import uuid
from types import SimpleNamespace
from unittest.mock import Mock, patch

from journey_cpu_test import FakeJourney, complete_receipt, load


def sha(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def restored_receipt():
    receipt, corpus = complete_receipt()
    history = {}

    def work(label, kind):
        identity = str(uuid.uuid5(uuid.NAMESPACE_URL, label))
        history[identity] = {"kind": kind, "state": "succeeded", "generation": 1, "result_sha256": sha(label)}
        return identity

    for case in receipt["cases"]:
        case["media_id"] = str(uuid.uuid5(uuid.NAMESPACE_URL, case["fixture"]))
        case["processing_work_id"] = work("source-process-" + case["fixture"], "recordings.process")
        case["revision"] = 1
    for profile in receipt["speaker_profiles"]:
        profile["training_work_id"] = work("source-train-" + profile["reader"], "models.train")
    for case in receipt["speaker_cases"]:
        case["matching_work_id"] = work("source-match-" + case["fixture"], "recordings.match")
    enrollment = {c["id"] for c in corpus["clips"] if c["role"] == "enrollment"}
    destination = {k: copy.deepcopy(receipt[k]) for k in ("environments", "model_inputs", "speaker_profiles",
                                                        "speaker_cases", "speaker_outcomes", "speaker_denominators")}
    destination["cases"] = [copy.deepcopy(c) for c in receipt["cases"] if c["fixture"] not in enrollment]
    for case in destination["cases"]:
        case.update(prior_revision=1, revision=2,
                    processing_work_id=str(uuid.uuid5(uuid.NAMESPACE_URL, "destination-process-" + case["fixture"])),
                    result_sha256=sha("destination-process-" + case["fixture"]))
    for case in destination["speaker_cases"]:
        case["matching_work_id"] = str(uuid.uuid5(uuid.NAMESPACE_URL, "destination-match-" + case["fixture"]))
        case["result_sha256"] = sha("destination-match-" + case["fixture"])
    destination["completed_work"] = {c["processing_work_id"]: {"kind": "recordings.process", "state": "succeeded",
        "generation": 1, "result_sha256": c["result_sha256"]} for c in destination["cases"]}
    destination["completed_work"].update({c["matching_work_id"]: {"kind": "recordings.match", "state": "succeeded",
        "generation": 1, "result_sha256": c["result_sha256"]} for c in destination["speaker_cases"]})
    destination["recognition_totals"] = {key: sum(c["word_error"][key] for c in destination["cases"]) for key in (
        "reference_words", "generated_words", "substitutions", "deletions", "insertions", "edit_distance")}
    destination["recognition_totals"]["word_error_rate"] = 0
    tasks = ("import", "existing_subtitles", "transcription", "diarization", "training")
    destination["readiness_before"] = {name: {"state": "needs_attention", "usable": False} for name in tasks}
    destination["readiness_after"] = {name: {"state": "prerequisites_available", "usable": True} for name in tasks}
    destination["readiness_after"]["training"] = {"state": "needs_attention", "usable": False}
    destination["source_withdrawal"] = {"before_processing": True, "after_processing": True}
    destination["retention"] = {"before_sha256": "9" * 64, "restored_sha256": "9" * 64, "after_sha256": "9" * 64,
                                "history": history, "history_sha256": sha(history),
                                "history_restored_sha256": sha(history), "history_after_sha256": sha(history)}
    receipt["restored"] = destination
    return receipt, corpus


class RestoredReceiptTests(unittest.TestCase):
    def test_destination_worker_preparation_preserves_source_and_models(self):
        cpu = load("journey_cpu")
        journey = FakeJourney()
        journey.receipt["environments"] = {"source": "unchanged"}
        destination = {}
        cpu.prepare_workers(journey, destination)
        self.assertEqual(journey.receipt["environments"], {"source": "unchanged"})
        self.assertEqual(set(destination), {"recognition", "diarization"})
        self.assertFalse(any(name.startswith("models.") for name, _ in journey.calls))
        self.assertEqual(journey.receipt["stages"], [])

    def test_restored_generation_uses_existing_media_and_new_revision(self):
        cpu = load("journey_cpu")
        media_id = str(uuid.uuid4())
        work_id = str(uuid.uuid4())
        before = {"revision": 4}
        after = {"revision": 5, "source_digest": "a" * 64, "document_digest": "b" * 64, "diagnostics": [],
                 "document": {"cues": [{"payload": {"plain_text": "independent text"}}]}}
        journey = SimpleNamespace(receipt={"environments": {}}, current_record=Mock(side_effect=[before, after]),
                                  input=Mock(return_value={"work_id": work_id}),
                                  finish=Mock(return_value={"recording_id": media_id, "revision": 5,
                                                            "document_digest": "b" * 64}), command=Mock())
        case = {"fixture": "held-out", "role": "held-out", "media_id": media_id,
                "reference": "independent text", "original_sha256": "c" * 64}
        evidence = {"environments": {}}
        with patch.object(cpu, "media_entry", return_value={"digest": "a" * 64,
                          "facts": {"canonical": {"tracks": [{"index": 0}]}}}), \
             patch.object(cpu, "engine_observation", return_value={}):
            cpu.process_cases(journey, [case], {"transcribe": "r", "diarize": "d"},
                              evidence=evidence, import_media=False)
        journey.command.assert_not_called()
        self.assertEqual(evidence["cases"][0]["processing_work_id"], work_id)
        self.assertEqual(evidence["cases"][0]["prior_revision"], 4)
        options = journey.input.call_args.kwargs["value"]
        self.assertEqual((options["transcription"], options["diarization"]), ("generate", "run"))
        self.assertNotIn("cases", journey.receipt)

    def test_destination_evidence_requires_new_work_and_retained_history(self):
        cpu = load("journey_cpu")
        receipt, corpus = restored_receipt()
        cpu.validate_restored(receipt, corpus)
        mutations = (
            lambda r: r.pop("restored"),
            lambda r: r["restored"]["cases"].pop(),
            lambda r: r["restored"]["cases"][0].update(processing_work_id=r["cases"][0]["processing_work_id"]),
            lambda r: r["restored"]["cases"][0].update(revision=1),
            lambda r: r["restored"]["cases"][0].update(media_id="other"),
            lambda r: r["restored"]["cases"][0]["transcription"].update(device="cuda"),
            lambda r: r["restored"]["cases"][0]["diarization"].update(python_sha256="0" * 64),
            lambda r: r["restored"]["cases"][0]["word_error"].update(word_error_rate=1),
            lambda r: r["restored"]["speaker_cases"][0].update(matching_work_id=r["speaker_cases"][0]["matching_work_id"]),
            lambda r: r["restored"]["speaker_outcomes"].update(correct_match=4),
            lambda r: r["restored"].pop("completed_work"),
            lambda r: next(iter(r["restored"]["completed_work"].values())).update(state="running"),
            lambda r: r["restored"]["model_inputs"]["transcribe"].update(model_id="changed"),
            lambda r: r["restored"]["speaker_profiles"][0].update(files={"changed": "a" * 64}),
            lambda r: r["restored"]["readiness_after"]["transcription"].update(usable=False),
            lambda r: r["restored"]["readiness_after"]["training"].update(usable="false"),
            lambda r: r["restored"]["source_withdrawal"].update(after_processing=False),
            lambda r: r["restored"]["retention"].update(after_sha256="0" * 64),
            lambda r: r["restored"]["retention"].update(history_after_sha256="0" * 64),
            lambda r: next(iter(r["restored"]["retention"]["history"].values())).update(result_sha256="0" * 64),
        )
        for index, mutate in enumerate(mutations):
            with self.subTest(index=index):
                changed = copy.deepcopy(receipt)
                mutate(changed)
                with self.assertRaises(ValueError):
                    cpu.validate_restored(changed, corpus)

    def test_missing_or_duplicate_fresh_work_is_not_inference(self):
        cpu = load("journey_cpu")
        receipt, corpus = restored_receipt()
        receipt["restored"]["cases"][1]["processing_work_id"] = receipt["restored"]["cases"][0]["processing_work_id"]
        with self.assertRaises(ValueError):
            cpu.validate_restored(receipt, corpus)

    def test_extended_campaign_requires_destination_stages_and_evidence(self):
        cpu = load("journey_cpu")
        receipt, corpus = restored_receipt()
        receipt["kind"] = "insonic-real-cpu-acceptance-v2"
        receipt["stages"] = [{"name": s, "state": "succeeded"} for s in
                             cpu.REQUIRED_STAGES[:-1] + cpu.RESTORED_STAGES + cpu.REQUIRED_STAGES[-1:]]
        cpu.validate_acceptance(receipt, corpus)
        for key in ("restored", "stages"):
            changed = copy.deepcopy(receipt)
            changed.pop(key)
            with self.assertRaises(ValueError):
                cpu.validate_acceptance(changed, corpus)

    def test_retained_history_is_measured_from_terminal_work(self):
        cpu = load("journey_cpu")
        identity = str(uuid.uuid4())
        work = {"id": identity, "kind": "recordings.process", "state": "succeeded", "generation": 2,
                "result": {"recording_id": "recording", "revision": 3}}
        journey = SimpleNamespace(command=Mock(return_value=work))
        self.assertEqual(cpu.history_snapshot(journey, {identity: work["kind"]}), {
            identity: {"kind": work["kind"], "state": "succeeded", "generation": 2,
                       "result_sha256": sha(work["result"])}})
        for change in ({"id": "other"}, {"state": "running"}, {"kind": "workers.setup"}):
            journey.command.return_value = {**work, **change}
            with self.assertRaises(ValueError):
                cpu.history_snapshot(journey, {identity: work["kind"]})


if __name__ == "__main__":
    unittest.main()
