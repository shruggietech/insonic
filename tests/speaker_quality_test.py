# SPDX-License-Identifier: Apache-2.0
import copy
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
import speaker_quality as quality


class SpeakerQualityTests(unittest.TestCase):
    def test_real_mode_requires_election_and_is_separate_from_ci(self):
        config = {'enabled': True, 'adapter': 'ollama', 'route': 'local',
                  'endpoint': 'http://127.0.0.1:11435', 'model': 'qwen2.5:14b'}
        quality.validate_election('real', True, config, {})
        for mode, elected, supplied, environment in (
                ('real', False, config, {}), ('real', True, config, {'CI': 'true'}),
                ('deterministic', True, config, {}), ('deterministic', False, config, {})):
            with self.assertRaises(ValueError): quality.validate_election(mode, elected, supplied, environment)
        quality.validate_election('deterministic', False, None, {})

    def test_real_provider_does_not_expand_routing_or_accept_credentials(self):
        config = {'enabled': True, 'adapter': 'ollama', 'route': 'local',
                  'endpoint': 'http://127.0.0.1:11435', 'model': 'qwen2.5:14b'}
        for field, value in (('endpoint', 'https://example.com'), ('endpoint', 'http://user:secret@localhost'),
                             ('route', 'hosted'), ('credential_id', 'secret'), ('adapter', 'insonic-http')):
            broken = dict(config, **{field: value})
            with self.assertRaises(ValueError): quality.validate_election('real', True, broken, {})

    def test_device_receipt_requires_actual_elected_execution(self):
        for device in ('cpu', 'cuda'):
            rows = [{'requested_device': device, 'device': device, 'attempted_devices': [device], 'count': 1}]
            self.assertEqual(quality.device_receipt(rows, device)[0]['device'], device)
            for mutation in ({'device': 'cpu' if device == 'cuda' else 'cuda'},
                             {'requested_device': 'auto'}, {'attempted_devices': ['cuda', 'cpu']},
                             {'fallback_reason': 'cuda_no_device'}, {'count': 0}):
                with self.assertRaises(ValueError): quality.device_receipt([rows[0] | mutation], device)
        with self.assertRaises(ValueError): quality.device_receipt([], 'cuda')

    def test_model_drift_or_missing_identity_fails(self):
        before = {'version': '0.1', 'model': 'qwen2.5:14b', 'digest': 'a' * 64}
        quality.same_model(before, copy.deepcopy(before), 'a' * 64)
        for after in ({**before, 'digest': 'b' * 64}, {**before, 'version': '0.2'}, {}):
            with self.assertRaises(ValueError): quality.same_model(before, after, 'a' * 64)
        with self.assertRaises(ValueError): quality.same_model(before, before, 'b' * 64)

    def test_acoustic_source_overlap_is_rejected(self):
        clips = [{'id': 'one', 'role': 'enrollment', 'sha256': 'a' * 64},
                 {'id': 'two', 'role': 'held-out', 'sha256': 'b' * 64}]
        quality.independent_sources(clips)
        clips[1]['sha256'] = clips[0]['sha256']
        with self.assertRaises(ValueError): quality.independent_sources(clips)

    def test_acoustic_metrics_include_unseen_false_match(self):
        rows = [{'outcome': 'correct_match', 'expected_reader': '84'},
                {'outcome': 'unknown', 'expected_reader': '174'},
                {'outcome': 'incorrect', 'expected_reader': '251'}]
        metrics = quality.acoustic_metrics(rows, ['84', '174'])
        self.assertEqual(metrics['identity_precision']['value'], .5)
        self.assertEqual(metrics['identity_coverage']['value'], .5)
        self.assertEqual(metrics['unseen_rejection']['value'], 0)

    def test_context_reply_uses_only_current_case_cue_references(self):
        case = {'cues': [{'voice': 'v1', 'text': 'My name is Alex.'}],
                'observations': [{'cue': 0, 'class': 'self-identification', 'name': 'Alex'}]}
        context = {'cues': [{'id': 'cue-one', 'text': 'My name is Alex.', 'voices': ['voice-one']}]}
        reply = quality.controlled_reply(case, context)
        self.assertEqual(reply['observations'][0]['voice_id'], 'voice-one')
        self.assertEqual(reply['observations'][0]['references'], [{'cue_id': 'cue-one', 'quote': 'My name is Alex.'}])
        context['cues'][0]['text'] = 'Another source.'
        with self.assertRaises(ValueError): quality.controlled_reply(case, context)

    def test_partial_case_is_error_not_silently_dropped(self):
        case = {'id': 'one'}
        def failed(): raise TimeoutError('private path and token must not enter receipt')
        row = quality.case_result(case, failed)
        self.assertEqual(row, {'id': 'one', 'state': 'error', 'observations': [], 'error': 'timeouterror'})
        self.assertNotIn('private', str(row))


if __name__ == '__main__': unittest.main()
