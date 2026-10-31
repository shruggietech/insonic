# SPDX-License-Identifier: Apache-2.0
"""Run the operator fixture through the exact source/package invocation adapter."""
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
import cli_contract_qualification as runner
from cli_operator_qualification import qualify_untimed_recording


class OperatorInvocationTests(unittest.TestCase):
    def exercise(self, scope, *, mismatch=False, missing_outcome=False):
        class FixtureComplete(Exception):
            pass

        calls = []
        imported = '11111111-1111-4111-8111-111111111111'
        recording = '22222222-2222-4222-8222-222222222222'
        version = (ROOT / 'VERSION').read_text().strip()
        assembly = None

        def capture(factory, argv, **options):
            nonlocal assembly
            argv = list(map(str, argv))
            calls.append(argv)
            indexes = [i for i, arg in enumerate(argv) if arg == '--request-id']
            correlation = argv[indexes[0] + 1]
            response = {'kind': 'runtime-response', 'schema_version': version, 'request_id': correlation}
            if len(indexes) != 1:
                response['error'] = {'code': 'invalid_request', 'message': '--request-id may be supplied only once'}
                return b'', json.dumps(response).encode(), 2, {}
            if 'assemble' in argv:
                assembly = correlation
                self.assertEqual(json.loads(Path(argv[argv.index('--input') + 1]).read_text()), {'turns': []})
                response['result'] = {'recording_id': recording, 'state': 'no-timed-subtitles'}
                if mismatch:
                    response['request_id'] = imported
            elif 'import' in argv:
                response['result'] = {'work_id': imported}
            elif 'work' in argv and imported in argv:
                response['result'] = {'result': {'items': [{'media_id': recording}]}}
            elif 'work' in argv and assembly in argv:
                self.assertNotIn('--json', argv)
                outcome = '' if missing_outcome else 'Outcome: No timed subtitles\n'
                text = ('Work: ' + assembly + '\nState: Succeeded\n' + outcome
                        + 'Complete JSON: insonic work show ' + assembly + ' --json\n')
                return text.encode(), b'', 0, {}
            elif '--help' in argv:
                response['result'] = {'help': ''}
            else:
                response['result'] = {}
            self.assertIn('--json', argv)
            return json.dumps(response).encode(), b'', 0, {}

        with tempfile.TemporaryDirectory(prefix='insonic-operator-invocation-') as temporary:
            workspace = Path(temporary)
            audio = workspace / 'fixture.wav'
            audio.write_bytes(b'owned fixture; process launch is modeled')

            def run_operator(invoke, *_args, **_kwargs):
                def supplied(args, value, **options):
                    payload = workspace / 'input.json'
                    payload.write_text(json.dumps(value), encoding='utf-8')
                    return invoke([*args, '--input', payload], **options)
                self.assertEqual(qualify_untimed_recording(invoke, supplied, audio, workspace), 'no-timed-subtitles')
                raise FixtureComplete()

            with patch.object(runner, 'capture_invocation', side_effect=capture), \
                    patch.object(runner, 'qualify_read_only', side_effect=run_operator):
                if mismatch:
                    with self.assertRaisesRegex(ValueError, 'CLI envelope/correlation differs'):
                        runner.qualify('fixture-cli', workspace, documentation=scope != 'source', artifact_identity={'scope': scope})
                elif missing_outcome:
                    with self.assertRaisesRegex(ValueError, 'recording outcome'):
                        runner.qualify('fixture-cli', workspace, documentation=True, artifact_identity={'scope': scope})
                else:
                    with self.assertRaises(FixtureComplete):
                        runner.qualify('fixture-cli', workspace, documentation=scope != 'source', artifact_identity={'scope': scope})
                    human_calls = [argv for argv in calls if '--json' not in argv]
                    self.assertEqual([argv[argv.index('work') + 1] for argv in human_calls], ['show', 'wait'])
                    self.assertTrue(all(assembly in argv for argv in human_calls))
                    self.assertTrue(all(argv.count('--request-id') == 1 for argv in calls))

    def test_source_package_and_diagnostic_adapters_qualify_same_outcome(self):
        for scope in ['source', 'package', 'diagnostic']:
            with self.subTest(scope=scope):
                self.exercise(scope)

    def test_machine_correlation_remains_strict(self):
        self.exercise('package', mismatch=True)

    def test_human_outcome_assertion_remains_active(self):
        self.exercise('package', missing_outcome=True)


if __name__ == '__main__':
    unittest.main()
