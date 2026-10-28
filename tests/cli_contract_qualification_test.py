# SPDX-License-Identifier: Apache-2.0
from collections import Counter
from contextlib import ExitStack
import importlib
import importlib.util
from pathlib import Path
import sys
import tempfile
import threading
import time
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parents[1] / 'scripts'
sys.path.insert(0, str(SCRIPTS))
spec = importlib.util.spec_from_file_location('cli_contract_qualification', SCRIPTS / 'cli_contract_qualification.py')
qualification = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualification)


class ReadOnlyInvocationFixture:
    """Model CLI responses without launching a native client or changing files."""
    def __init__(self, fail=False):
        self.calls = []
        self.active = self.maximum = 0
        self.threads = set()
        self.lock = threading.Lock()
        self.first_wave = threading.Barrier(4)
        self.fail = fail

    def __call__(self, args, *, expected=0, scoped=True):
        args = tuple(map(str, args))
        with self.lock:
            ordinal = len(self.calls)
            self.calls.append((args, expected, scoped))
            self.active += 1
            self.maximum = max(self.maximum, self.active)
            self.threads.add(threading.get_ident())
        try:
            if ordinal < 4:
                self.first_wave.wait(timeout=2)
            time.sleep(.002)
            if expected == 2:
                return {'error': {'code': 'invalid_request'}}
            if args[0] == '--workspace':
                command = args[2:-1]
            elif args[0] == 'help':
                command = args[1:]
                if self.fail and command == ('media', 'import'):
                    time.sleep(.02)
                    return {'result': {'help': 'different contextual output'}}
            else:
                if self.fail:
                    raise ValueError('later group failure')
                return {'result': {'help': 'Commands: ' + ' '.join(args)}}
            return {'result': {'help': 'Usage: ' + ' '.join(command)}}
        finally:
            with self.lock:
                self.active -= 1


class CliReadOnlyQualificationTests(unittest.TestCase):
    paths = [['media', 'import'], ['media', 'show'], ['settings', 'set']]

    def expected_calls(self, help_workspace, missing):
        calls = []
        for command in self.paths:
            calls += [
                ((str(help_workspace), *command, '--help'), 0),
                ((str(missing), *command, '-h'), 0),
                (('help', *command), 0),
            ]
        calls += [(('media',), 0), (('settings',), 0)]
        calls += [(tuple(args), 2) for args in qualification.PARSER_REJECTIONS]
        return Counter(calls)

    def test_every_read_only_check_runs_with_four_workers_and_no_workspace_writes(self):
        with tempfile.TemporaryDirectory(prefix='insonic-cli-model-') as directory:
            workspace = Path(directory)
            sentinel = workspace / 'existing workspace contract.json'
            sentinel.write_bytes(b'unchanged fixture bytes\n')
            missing = workspace / 'absent-help-workspace'
            invoke = ReadOnlyInvocationFixture()
            qualification.qualify_read_only(invoke, self.paths, workspace, missing)
            actual = Counter(((args[1:] if args[0] == '--workspace' else args), expected)
                             for args, expected, scoped in invoke.calls)
            self.assertEqual(actual, self.expected_calls(workspace, missing))
            self.assertTrue(all(scoped is False for _, _, scoped in invoke.calls))
            self.assertEqual(invoke.maximum, 4)
            self.assertLessEqual(len(invoke.threads), 4)
            self.assertEqual(invoke.active, 0)
            self.assertEqual(list(workspace.iterdir()), [sentinel])
            self.assertEqual(sentinel.read_bytes(), b'unchanged fixture bytes\n')

    def test_failure_is_reported_in_check_order_after_all_owned_checks_join(self):
        with tempfile.TemporaryDirectory(prefix='insonic-cli-model-') as directory:
            workspace = Path(directory)
            invoke = ReadOnlyInvocationFixture(fail=True)
            with self.assertRaisesRegex(ValueError, 'Contextual help differs: media import'):
                qualification.qualify_read_only(invoke, self.paths, workspace, workspace / 'absent')
            self.assertEqual(invoke.active, 0)
            self.assertEqual(invoke.maximum, 4)
            rejected = Counter(args for args, expected, _ in invoke.calls if expected == 2)
            self.assertEqual(rejected, Counter(tuple(args) for args in qualification.PARSER_REJECTIONS))
            self.assertEqual(list(workspace.iterdir()), [])

    def test_workspace_snapshot_still_rejects_files_created_by_group_help(self):
        with tempfile.TemporaryDirectory(prefix='insonic-cli-model-') as directory:
            workspace = Path(directory)
            model = ReadOnlyInvocationFixture()
            def invoke(args, **options):
                if args == ['media']:
                    (workspace / 'unexpected write').write_bytes(b'fixture violation')
                return model(args, **options)
            with self.assertRaisesRegex(ValueError, 'Help created workspace files'):
                qualification.qualify_read_only(invoke, self.paths, workspace, workspace / 'absent')
            self.assertEqual(model.active, 0)



    def test_optional_phase_observer_runs_before_launch_and_receives_no_command_data(self):
        class StopLaunch(RuntimeError):pass
        phases=[]
        def launch(*args,**kwargs):
            self.assertEqual(phases,['help'])
            self.assertIn('--help',args[0])
            raise StopLaunch('bounded fixture stop')
        with tempfile.TemporaryDirectory() as temporary,patch.object(qualification,'ProcessTree',side_effect=launch):
            with self.assertRaises(StopLaunch):
                qualification.qualify('fixed-cli.exe',Path(temporary),progress=phases.append)
        self.assertEqual(phases,['help'])

    def test_phase_observation_is_optional_and_preserves_launch_failure(self):
        with tempfile.TemporaryDirectory() as temporary,patch.object(qualification,'ProcessTree',side_effect=RuntimeError('original launch failure')) as launch:
            with self.assertRaisesRegex(RuntimeError,'original launch failure'):
                qualification.qualify('fixed-cli.exe',Path(temporary))
            launch.assert_called_once()

    def test_readiness_regression_is_isolated_and_every_worker_joins_on_failure(self):
        names = ['operator', 'input', 'text_duration', 'readiness', 'terminal', 'recovery']
        modules = [importlib.import_module('cli_' + name + '_qualification') for name in names]
        barrier = threading.Barrier(4)
        finished, roots = [], []
        lock = threading.Lock()

        def fixture(name):
            def run(invoke, workspace, version, **options):
                with lock:
                    roots.append(workspace)
                if name not in ('terminal', 'recovery'):
                    barrier.wait(timeout=2)
                try:
                    if name == 'readiness':
                        self.assertEqual(options['executable'], 'fixture-cli')
                        raise ValueError('readiness fixture failure')
                    time.sleep(.01)
                    return {'state': 'passed'}
                finally:
                    with lock:
                        finished.append(name)
            return run

        with tempfile.TemporaryDirectory() as directory, ExitStack() as stack:
            for name, module in zip(names, modules):
                stack.enter_context(patch.object(module, 'qualify', side_effect=fixture(name)))
            with self.assertRaisesRegex(ValueError, 'readiness fixture failure'):
                with qualification.parallel_regressions(lambda *a, **k: {}, Path(directory),
                        '1.0.2', False, {}, 'fixture-cli', {}) as checks:
                    checks['readiness'].result()
        self.assertEqual(set(finished), set(names))
        self.assertEqual(len(set(roots)), 6)

if __name__ == '__main__':
    unittest.main()
