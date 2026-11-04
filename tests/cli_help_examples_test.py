#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Regression checks for actual help-example extraction and exact substitutions."""
import unittest
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "scripts"))
from cli_help_examples import emitted_example, substitute


class HelpExampleTests(unittest.TestCase):
    def test_requires_one_actual_example_for_the_current_path(self):
        self.assertEqual(emitted_example('Purpose: Read.\nExample: insonic query native --dialect ladybug-cypher --file FILE\n',
                                         ['query', 'native']),
                         ['query', 'native', '--dialect', 'ladybug-cypher', '--file', 'FILE'])
        for text in ['', 'Example: insonic media list\n', 'Example: insonic work results WORK_UUID [AFTER]\n',
                     'Example: insonic media list\nExample: insonic media list\n']:
            with self.assertRaises(ValueError):
                emitted_example(text, ['work', 'results'])

    def test_answer_package_operand_requires_current_retained_identity(self):
        identity = "10000000-0000-4000-8000-000000000001"
        for action in ["show", "export"]:
            actual = ["answers", action, "PACKAGE_UUID"]
            with self.assertRaises(ValueError):
                substitute(actual, {})
            self.assertEqual(substitute(actual, {"PACKAGE_UUID": identity}), ["answers", action, identity])

    def test_recording_operand_requires_current_recording_identity(self):
        identity = "10000000-0000-4000-8000-000000000002"
        for action in ["discover", "hypotheses", "resolve", "apply", "retract", "run"]:
            actual = ["speakers", action, "RECORDING_UUID"]
            with self.assertRaises(ValueError):
                substitute(actual, {})
            self.assertEqual(substitute(actual, {"RECORDING_UUID": identity}), ["speakers", action, identity])

    def test_only_documented_substitutions_change_actual_argv(self):
        actual = ['query', 'native', '--dialect', 'ladybug-cypher', '--file', 'FILE']
        self.assertEqual(substitute(actual, {'FILE': '/tmp/owned query.cypher'}),
                         ['query', 'native', '--dialect', 'ladybug-cypher', '--file', '/tmp/owned query.cypher'])
        with self.assertRaises(ValueError):
            substitute(['work', 'show', 'WORK_UUID'], {})
        with self.assertRaises(ValueError):
            substitute(actual, {'--dialect': 'arcade-sql', 'FILE': '/tmp/query.cypher'})


class DiscoveryFixtureTests(unittest.TestCase):
    def test_owned_provider_runtime_is_isolated_from_ci_runtime(self):
        from cli_help_examples import discovery_fixture
        from tempfile import TemporaryDirectory
        calls = []
        environment = {'CI': 'true', 'GITHUB_ACTIONS': 'true', 'TF_BUILD': '1',
                       'BUILD_BUILDID': '1', 'JENKINS_URL': 'fixture', 'PATH': 'owned-path'}
        original = dict(environment)
        def invoke(args, **kwargs):
            calls.append((args, kwargs))
            self.assertEqual(kwargs['execution_environment'], {'PATH': 'owned-path'})
            if args[:2] == ['media', 'import'] or args[:2] == ['transcript', 'import']:
                return {'result': {'work_id': 'owned-work'}}
            if args[:2] == ['work', 'wait']:
                return {'result': {'result': {'items': [{'media_id': 'owned-recording'}]}}}
            return {'result': {}}
        with TemporaryDirectory() as temp:
            directory = Path(temp)
            fixture = discovery_fixture(invoke, directory, environment, directory / 'audio.wav', directory / 'source.vtt')
            self.assertEqual(fixture['recording_id'], 'owned-recording')
            fixture['call'](['speakers', 'hypotheses', fixture['recording_id']])
            self.assertTrue(all(k['workspace_path'] == fixture['library'] for _, k in calls[1:]))
        self.assertEqual(environment, original)
        self.assertEqual(calls[0][0][:2], ['workspace', 'init'])
        self.assertFalse(calls[0][1]['scoped'])
        self.assertEqual(sum(a[:2] == ['work', 'wait'] for a, _ in calls), 2)


class ParallelRegressionTests(unittest.TestCase):
    def test_helpers_keep_isolated_defaults_and_explicit_overrides(self):
        import cli_contract_qualification as runner
        from unittest.mock import patch
        from threading import Barrier, Lock
        barrier, lock, calls = Barrier(4), Lock(), []
        selected_factory = object()
        def invoke(args, **kwargs):
            with lock:
                calls.append((args, kwargs))
            return {}
        def fake(scoped, *args, **_kwargs):
            root = args[1] if args[0] is None else args[0]
            if root.name in ('isolated examples qualification', 'isolated terminal qualification', 'isolated recovery qualification', 'isolated storage qualification', 'isolated backup qualification'):
                self.assertIs(_kwargs['process_factory'], selected_factory)
            scoped(['settings', 'show'])
            scoped(['settings', 'show'], workspace_path=root / 'explicit override')
            scoped(['--help'], scoped=False)
            if root.name not in ('isolated examples qualification', 'isolated terminal qualification', 'isolated recovery qualification', 'isolated storage qualification', 'isolated backup qualification'):
                barrier.wait(timeout=2)
            return str(root)
        with patch('cli_operator_qualification.qualify', fake), \
             patch('cli_input_qualification.qualify', fake), \
             patch('cli_text_duration_qualification.qualify', fake), \
             patch('cli_readiness_qualification.qualify', fake), \
             patch('cli_terminal_qualification.qualify', fake), \
             patch('cli_recovery_qualification.qualify', fake), \
             patch('cli_storage_qualification.qualify', fake), \
             patch('cli_backup_qualification.qualify', fake), \
             patch('cli_help_examples.qualify', fake):
            with runner.parallel_regressions(invoke, Path('/owned'), '1.3.0', True, {}, '/owned/cli', {}, process_factory=selected_factory) as futures:
                roots = {name: Path(future.result()) for name, future in futures.items()}
        self.assertEqual(len(set(roots.values())), 9)
        self.assertTrue(all(root.parent == Path('/owned') for root in roots.values()))
        for name, root in roots.items():
            scenario=name.replace('_','-')
            self.assertIn((['settings', 'show'], {'workspace_path': root, 'scenario': scenario}), calls)
            self.assertIn((['settings', 'show'], {'workspace_path': root / 'explicit override', 'scenario': scenario}), calls)
            self.assertIn((['--help'], {'scoped': False, 'workspace_path': root, 'scenario': scenario}), calls)

    def test_failure_waits_for_every_other_owned_helper(self):
        import cli_contract_qualification as runner
        from unittest.mock import patch
        import time
        finished = []
        def fail(*_args, **_kwargs):
            raise ValueError('fixture regression')
        def complete(*_args, **_kwargs):
            time.sleep(.02)
            finished.append(True)
        with patch('cli_operator_qualification.qualify', fail), \
             patch('cli_input_qualification.qualify', complete), \
             patch('cli_text_duration_qualification.qualify', complete), \
             patch('cli_readiness_qualification.qualify', complete), \
             patch('cli_terminal_qualification.qualify', complete), \
             patch('cli_recovery_qualification.qualify', complete), \
             patch('cli_storage_qualification.qualify', complete), \
             patch('cli_backup_qualification.qualify', complete):
            with self.assertRaisesRegex(ValueError, 'fixture regression'):
                with runner.parallel_regressions(lambda *a, **k: {}, Path('/owned'), '1.3.0', False, {}, '/owned/cli', {}) as futures:
                    futures['operator'].result()
        self.assertEqual(len(finished), 7)


if __name__ == '__main__':
    unittest.main()

class TruthfulOutcomeTests(unittest.TestCase):
    def test_immediate_receipt_is_not_semantic_success(self):
        from cli_corpus import example_outcome
        self.assertEqual(example_outcome({'work_id':'accepted'},None)['outcome'],'invocation-only')
    def test_async_refusal_retains_failure_classification(self):
        from cli_corpus import example_outcome
        value=example_outcome({},None,completed={'result':{'state':'failed'},'error':{'code':'unavailable'}})
        self.assertEqual(value['outcome'],'refusal-observed')
        self.assertEqual(value['failure_code'],'unavailable')
