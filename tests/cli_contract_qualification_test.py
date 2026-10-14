# SPDX-License-Identifier: Apache-2.0
from collections import Counter
import importlib.util
from pathlib import Path
import sys
import tempfile
import threading
import time
import unittest

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


if __name__ == '__main__':
    unittest.main()
