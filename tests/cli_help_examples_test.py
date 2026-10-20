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

    def test_only_documented_substitutions_change_actual_argv(self):
        actual = ['query', 'native', '--dialect', 'ladybug-cypher', '--file', 'FILE']
        self.assertEqual(substitute(actual, {'FILE': '/tmp/owned query.cypher'}),
                         ['query', 'native', '--dialect', 'ladybug-cypher', '--file', '/tmp/owned query.cypher'])
        with self.assertRaises(ValueError):
            substitute(['work', 'show', 'WORK_UUID'], {})
        with self.assertRaises(ValueError):
            substitute(actual, {'--dialect': 'arcade-sql', 'FILE': '/tmp/query.cypher'})


class ParallelRegressionTests(unittest.TestCase):
    def test_helpers_keep_isolated_defaults_and_explicit_overrides(self):
        import cli_contract_qualification as runner
        from unittest.mock import patch
        from threading import Barrier, Lock
        barrier, lock, calls = Barrier(4), Lock(), []
        def invoke(args, **kwargs):
            with lock:
                calls.append((args, kwargs))
            return {}
        def fake(scoped, *args, **_kwargs):
            root = args[1] if args[0] is None else args[0]
            scoped(['settings', 'show'])
            scoped(['settings', 'show'], workspace_path=root / 'explicit override')
            scoped(['--help'], scoped=False)
            barrier.wait(timeout=2)
            return str(root)
        with patch('cli_operator_qualification.qualify', fake), \
             patch('cli_input_qualification.qualify', fake), \
             patch('cli_text_duration_qualification.qualify', fake), \
             patch('cli_help_examples.qualify', fake):
            with runner.parallel_regressions(invoke, Path('/owned'), '1.0.1', True, {}, '/owned/cli', {}) as futures:
                roots = {name: Path(future.result()) for name, future in futures.items()}
        self.assertEqual(len(set(roots.values())), 4)
        self.assertTrue(all(root.parent == Path('/owned') for root in roots.values()))
        for root in roots.values():
            self.assertIn((['settings', 'show'], {'workspace_path': root}), calls)
            self.assertIn((['settings', 'show'], {'workspace_path': root / 'explicit override'}), calls)
            self.assertIn((['--help'], {'scoped': False, 'workspace_path': root}), calls)

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
             patch('cli_text_duration_qualification.qualify', complete):
            with self.assertRaisesRegex(ValueError, 'fixture regression'):
                with runner.parallel_regressions(lambda *a, **k: {}, Path('/owned'), '1.0.1', False, {}, '/owned/cli', {}) as futures:
                    futures['operator'].result()
        self.assertEqual(len(finished), 2)


if __name__ == '__main__':
    unittest.main()
