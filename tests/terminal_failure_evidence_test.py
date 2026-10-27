# SPDX-License-Identifier: Apache-2.0
from pathlib import Path
import json
import os
import sys
import tempfile
from types import SimpleNamespace
import unittest
import uuid

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from terminal_failure_evidence import FailureEvidence, diagnostic_environment


class TerminalFailureEvidenceTests(unittest.TestCase):
    def test_election_is_local_to_ci_environment_copy(self):
        for ci in ('true', 'false'):
            original = {'GITHUB_ACTIONS': ci, 'PATH': 'private-path'}
            selected = diagnostic_environment(original)
            self.assertIsNot(selected, original)
            self.assertNotIn('INSONIC_PROCESS_STARTUP_DIAGNOSTICS', original)
            self.assertEqual(selected.get('INSONIC_PROCESS_STARTUP_DIAGNOSTICS'), '1' if ci == 'true' else None)

    def test_failure_is_correlated_redacted_and_captured_before_cleanup(self):
        with tempfile.TemporaryDirectory() as root:
            directory = Path(root) / 'build' / 'native'
            directory.mkdir(parents=True)
            request, work, startup, workspace, session = [str(uuid.uuid4()) for _ in range(5)]
            log = Path(root) / 'private-provider-token.jsonl'
            log.write_text(json.dumps({'request_id': request, 'operation': 'work.wait', 'state': 'running',
                                       'private': 'private-provider-token'}) + '\n', encoding='utf-8')
            native = {'kind': 'process-startup-diagnostic', 'stage': 'command_start', 'platform': 'windows',
                      'pid': 43, 'has_errno': True, 'error_number': 5}
            evidence = FailureEvidence('a' * 64, {'GITHUB_ACTIONS': 'true', 'GITHUB_WORKSPACE': root}, platform='windows')
            (directory / 'process-startup-43-1-receipt.json').write_text(json.dumps(native), encoding='utf-8')
            (directory / 'process-startup-44-1-receipt.json').write_text(json.dumps({**native, 'pid': 44}), encoding='utf-8')
            evidence.identities.update(startup_id=startup, workspace_id=workspace, runtime_session_id=session, work_id=work)
            live = {'before_cleanup': True}
            def poll_owner():
                self.assertTrue(live['before_cleanup'])
                return None
            owner = SimpleNamespace(process=SimpleNamespace(pid=42, poll=poll_owner))
            child = SimpleNamespace(owner=SimpleNamespace(process=SimpleNamespace(pid=43, poll=lambda: 4)))
            evidence.track(child, request, 'json', log, work_id=work)
            with self.assertRaisesRegex(ValueError, 'private-provider-token'):
                with evidence.watch(owner):
                    raise ValueError('private-provider-token')
            live['before_cleanup'] = False
            log.unlink()
            records = list(directory.glob('terminal-failure-*-receipt.json'))
            self.assertEqual(len(records), 1)
            raw = records[0].read_bytes()
            self.assertLessEqual(len(raw), 16384)
            self.assertNotIn(b'private-provider-token', raw)
            self.assertNotIn(root.encode(), raw)
            record = json.loads(raw)
            self.assertEqual(record['identities'], evidence.identities)
            self.assertEqual(record['owner'], {'pid': 42, 'exit_code': None, 'observation': 'running'})
            client = record['clients'][0]
            self.assertEqual(client['request_id'], request)
            self.assertEqual(client['work_id'], work)
            self.assertEqual(client['process']['exit_code'], 4)
            self.assertEqual(client['startup'], [native])
            self.assertEqual(client['events'], [{'request_id': request, 'operation': 'work.wait', 'state': 'running'}])

    def test_missing_malformed_oversized_foreign_and_private_records_are_not_adopted(self):
        with tempfile.TemporaryDirectory() as root:
            directory = Path(root) / 'build' / 'native'
            directory.mkdir(parents=True)
            evidence = FailureEvidence('b' * 64, {'GITHUB_ACTIONS': 'true', 'GITHUB_WORKSPACE': root}, platform='windows')
            for index, data in enumerate((b'{' , b'x' * 2048,
                    json.dumps({'kind': 'process-startup-diagnostic', 'stage': 'private-token', 'platform': 'windows',
                                'pid': 43, 'has_errno': True, 'error_number': 5}).encode(),
                    json.dumps({'kind': 'process-startup-diagnostic', 'stage': 'command_start', 'platform': 'windows',
                                'pid': 99, 'has_errno': True, 'error_number': 5}).encode())):
                (directory / f'process-startup-43-{index+1}-receipt.json').write_bytes(data)
            stale = directory / 'process-startup-43-5-receipt.json'
            stale.write_text(json.dumps({'kind': 'process-startup-diagnostic', 'stage': 'command_start', 'platform': 'windows',
                                         'pid': 43, 'has_errno': True, 'error_number': 5}), encoding='utf-8')
            os.utime(stale, ns=(1, 1))
            process = SimpleNamespace(pid=43, poll=lambda: 4)
            evidence.track(SimpleNamespace(owner=SimpleNamespace(process=process)), str(uuid.uuid4()), 'json', Path(root)/'absent')
            evidence.capture(SimpleNamespace(process=SimpleNamespace(pid=42, poll=lambda: None)))
            record = json.loads(next(directory.glob('terminal-failure-*-receipt.json')).read_bytes())
            self.assertEqual(record['clients'][0]['startup'], [])
            self.assertEqual(record['clients'][0]['events'], [])
            self.assertEqual(record['clients'][0]['startup_observation'], 'unavailable')

    def test_disabled_evidence_and_success_write_nothing_and_failure_does_not_mask_primary(self):
        with tempfile.TemporaryDirectory() as root:
            owner = SimpleNamespace(process=SimpleNamespace(pid=42, poll=lambda: None))
            disabled = FailureEvidence('c' * 64, {'GITHUB_WORKSPACE': root}, platform='windows')
            with self.assertRaisesRegex(ValueError, 'primary'):
                with disabled.watch(owner):
                    raise ValueError('primary')
            self.assertEqual(list(Path(root).iterdir()), [])
            enabled = FailureEvidence('c' * 64, {'GITHUB_ACTIONS': 'true', 'GITHUB_WORKSPACE': root}, platform='windows')
            with enabled.watch(owner):
                pass
            self.assertEqual(list(Path(root).iterdir()), [])
            (Path(root)/'build').write_bytes(b'block optional evidence')
            with self.assertRaisesRegex(ValueError, 'primary') as raised:
                with enabled.watch(owner):
                    raise ValueError('primary')
            self.assertTrue(any('failure evidence' in note for note in raised.exception.__notes__))


if __name__ == '__main__':
    unittest.main()
