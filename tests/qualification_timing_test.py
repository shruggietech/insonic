# SPDX-License-Identifier: Apache-2.0
"""Exercise real harness decisions with controlled elapsed time, never long sleeps."""
import importlib.util
import io
from pathlib import Path
import subprocess
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
spec = importlib.util.spec_from_file_location('timing_package', ROOT / 'scripts/package-desktop.py')
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)

class QualificationTimingTests(unittest.TestCase):
    def test_delayed_healthy_command_survives_former_bound_and_returns_immediately(self):
        elapsed = [0]
        class Tree:
            process = None
            returncode = 0
            def __init__(self, *args, **kwargs): self.process = self
            def communicate(self, timeout):
                self.bound = timeout
                elapsed[0] += 181
                if timeout < 181: raise subprocess.TimeoutExpired('healthy', timeout)
                return b'complete', b''
            def poll(self): return self.returncode
            def kill(self): pass
            def __enter__(self): return self
            def __exit__(self, *args): pass
        tree = Tree()
        with patch.object(package, 'native_webview_probe', return_value=False), patch.object(package.time, 'monotonic', side_effect=lambda: elapsed[0]):
            result = package.child(['healthy'], process_factory=lambda *a, **k: tree)
        self.assertEqual(result, b'complete')
        self.assertEqual(tree.bound, 600)
        self.assertEqual(elapsed[0], 181)

    def test_expanded_timeout_and_failed_cleanup_keep_original_failure(self):
        expired = subprocess.TimeoutExpired('stalled', 600, output=b'checkpoint')
        class Tree:
            pid = 4242
            returncode = None
            def __init__(self, *args, **kwargs): self.process = self; self.calls = []; self.killed = False
            def communicate(self, timeout):
                self.calls.append(timeout)
                if timeout == 600: raise expired
                raise subprocess.TimeoutExpired('cleanup', timeout)
            def poll(self): return None
            def kill(self): self.killed = True
            def __enter__(self): return self
            def __exit__(self, *args): pass
        tree = Tree()
        with patch.object(package, 'write_json'), patch.object(package.sys, 'stdout', SimpleNamespace(buffer=io.BytesIO())), patch.object(package.sys, 'stderr', SimpleNamespace(buffer=io.BytesIO())):
            with self.assertRaises(subprocess.TimeoutExpired) as failure:
                package.child(['stalled'], process_factory=lambda *a, **k: tree)
        self.assertIs(failure.exception, expired)
        self.assertTrue(tree.killed)
        self.assertEqual(tree.calls, [600, 30])

    def test_earlier_cancellation_is_not_replaced_by_the_larger_bound(self):
        cancelled = KeyboardInterrupt('controlled cancellation')
        class Tree:
            def __init__(self, *args, **kwargs): self.process = self; self.closed = False
            def communicate(self, timeout): raise cancelled
            def __enter__(self): return self
            def __exit__(self, *args): self.closed = True
        tree = Tree()
        with self.assertRaises(KeyboardInterrupt) as failure:
            package.child(['cancelled'], process_factory=lambda *a, **k: tree)
        self.assertIs(failure.exception, cancelled)
        self.assertTrue(tree.closed)

    def test_policy_has_consistent_units_and_enclosing_bounds(self):
        import qualification_timing as timing
        self.assertGreaterEqual(timing.HELP_SECONDS, 180)
        self.assertEqual(timing.CAPTURE_SECONDS, 3600)
        self.assertGreater(timing.INSTALLER_OUTER_SECONDS, timing.INSTALLER_WORKER_SECONDS)
        self.assertGreater(timing.INSTALLER_WORKER_SECONDS, timing.INSTALLER_UI_SECONDS)
        with self.assertRaises(ValueError): timing.validate_policy({'subprocess_seconds': True})
        with self.assertRaises(ValueError): timing.validate_policy({**timing.POLICY, 'subprocess_seconds': 86401})
        with self.assertRaisesRegex(ValueError, 'Duplicate'):
            timing.load_policy(SimpleNamespace(read_text=lambda **kwargs: '{"subprocess_seconds":600,"subprocess_seconds":600}'))

    def test_original_phase_deadline_is_never_restarted(self):
        import qualification_timing as timing
        with patch.dict(timing.os.environ, {'INSONIC_QUALIFICATION_DEADLINE': '1200000'}), patch.object(timing.time, 'time', return_value=1000):
            self.assertEqual(timing.phase_bound(600), 200)
        with patch.dict(timing.os.environ, {'INSONIC_QUALIFICATION_DEADLINE': '1200000'}), patch.object(timing.time, 'time', return_value=1200):
            with self.assertRaises(TimeoutError): timing.phase_bound(600)

    def test_failed_tree_cleanup_preserves_earlier_cancellation(self):
        from process_tree import ProcessTree
        tree = object.__new__(ProcessTree)
        cancelled = KeyboardInterrupt('cancelled first')
        with patch.object(tree, 'close', side_effect=RuntimeError('cleanup')):
            tree.__exit__(KeyboardInterrupt, cancelled, None)
        self.assertIn('Owned process tree cleanup did not complete', cancelled.__notes__)

    def test_source_cache_changes_with_consumed_policy_and_loader(self):
        import media_source_build as source
        with patch.object(source, 'platform_key', return_value='linux_amd64'), patch.object(source, 'build_environment', return_value={'CC': 'fixture-cc'}), patch.object(source, 'child', return_value=b'fixture compiler 1'):
            with patch.object(source.timing, 'source_identity', return_value={'policy_sha256': 'a' * 64, 'loader_sha256': 'b' * 64}):
                initial = source.cache_identity()[2]
            for identity in ({'policy_sha256': 'c' * 64, 'loader_sha256': 'b' * 64}, {'policy_sha256': 'a' * 64, 'loader_sha256': 'c' * 64}):
                with patch.object(source.timing, 'source_identity', return_value=identity):
                    self.assertNotEqual(initial, source.cache_identity()[2])

    def test_go_and_dependency_enclosing_allowances_include_teardown(self):
        import qualification_timing as timing
        import qualify
        with patch.object(qualify, 'child', return_value=b'complete') as child:
            self.assertEqual(qualify.go_tests(['./fixture']), b'complete')
        self.assertIn('-timeout=' + str(timing.GO_TEST_SECONDS) + 's', child.call_args.args[0])
        self.assertGreater(child.call_args.kwargs['timeout'], timing.GO_TEST_SECONDS + timing.CLEANUP_SECONDS)
        self.assertGreater(timing.DEPENDENCY_INSTALL_SECONDS, 2 * (3 * timing.REQUEST_SECONDS + 3) + timing.CLEANUP_SECONDS)

    def test_help_progress_uses_native_monotonic_time_not_buffer_arrival(self):
        import json
        import qualify
        observation = {}
        stage = {'kind': 'desktop-qualification-progress', 'mode': 'help', 'observation': 'native-receipt', 'elapsed_ns': 2000000000, 'stage': 'fixtures', 'private': 'must not survive', 'frontend_elapsed_ms': 8.5, 'frontend_origin': 'first-reported-stage'}
        terminal = {**stage, 'elapsed_ns': 36000000000, 'stage': 'terminal', 'scope': 'application-return', 'passed': True, 'classification': 'passed-after-former-bound', 'effective_bound_seconds': 180, 'capture_bound_seconds': 210, 'former_bound_seconds': 35}
        qualify.record_progress(observation, ('warning\n' + json.dumps(stage) + '\n' + json.dumps(terminal)).encode())
        self.assertEqual([row['elapsed_ns'] for row in observation['progress']], [2000000000, 36000000000])
        self.assertEqual(observation['progress'][-1]['classification'], 'passed-after-former-bound')
        self.assertNotIn('private', repr(observation))
        self.assertEqual(observation['progress'][0]['frontend_elapsed_ms'], 8.5)
        self.assertEqual(observation['progress'][0]['frontend_origin'], 'first-reported-stage')
        for invalid in ({**stage, 'frontend_elapsed_ms': float('nan')}, {**stage, 'frontend_elapsed_ms': -1}, {**stage, 'frontend_elapsed_ms': 86400001}, {**stage, 'frontend_origin': 'private-origin'}):
            with self.assertRaises(ValueError): qualify.record_progress({}, json.dumps(invalid).encode())
        with self.assertRaises(ValueError):
            qualify.record_progress({}, (json.dumps(stage) + '\n' + json.dumps({**terminal, 'frontend_elapsed_ms': 8.4})).encode())
        with self.assertRaises(ValueError):
            qualify.record_progress({}, (json.dumps(terminal) + '\n' + json.dumps(stage)).encode())

    def test_forced_cleanup_cannot_promote_a_native_help_terminal_pass(self):
        import json
        import qualify
        receipt = {'offline_help_activation': 'passed'}
        terminal = {'stage': 'terminal', 'passed': True}
        observation = {'progress': [terminal], 'outcome': 'completed', 'natural_release': False}
        with self.assertRaisesRegex(ValueError, 'natural-release'):
            qualify.write_receipt(ROOT / 'unused-fixture.json', json.dumps(receipt).encode(), receipt, observation=observation)

    def test_real_journey_pending_query_can_complete_after_old_api_window(self):
        import json
        spec = importlib.util.spec_from_file_location('timing_journey', ROOT / 'scripts/qualify-journey.py')
        module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
        instance = object.__new__(module.Journey)
        instance.deadline = 5000; instance.cli = Path('fixture'); instance.workspace = ROOT
        instance.environment = {}; instance.receipt = {'operations': []}
        clock = [0]; calls = []
        def child(argv, environment, **options):
            calls.append(argv)
            clock[0] += 601 if len(calls) == 1 else 1
            if len(calls) == 1:
                raise module.engine.CLIResponseError('unavailable', {'error': {'code': 'unavailable', 'reason': 'wait_timeout'}, 'result': {'id': 'work', 'state': 'running'}})
            return json.dumps({'result': {'id': 'work', 'state': 'succeeded', 'kind': 'worker_setup', 'phase': 'complete', 'result': {'verified': True}}}).encode()
        with patch.object(module.time, 'monotonic', side_effect=lambda: clock[0]), patch.object(module.engine, '_child', side_effect=child):
            self.assertEqual(instance.finish({'work_id': 'work'}, maximum=1800), {'verified': True})
        self.assertEqual(len(calls), 2)
        self.assertTrue(all(argv[argv.index('--timeout-ms') + 1] == '600000' for argv in calls))
        self.assertNotIn('failure_code', instance.receipt)
        clock[0] = 0
        def pending(argv, environment, **options):
            clock[0] += 1801
            raise module.engine.CLIResponseError('unavailable', {'error': {'code': 'unavailable', 'reason': 'wait_timeout'}, 'result': {'id': 'work', 'state': 'running'}})
        with patch.object(module.time, 'monotonic', side_effect=lambda: clock[0]), patch.object(module.engine, '_child', side_effect=pending) as launched:
            with self.assertRaisesRegex(ValueError, 'original operation deadline'):
                instance.finish({'work_id': 'work'}, maximum=1800)
            self.assertEqual(launched.call_count, 1)

    def test_help_observation_requires_natural_tree_release_on_both_platforms(self):
        class Tree:
            returncode = 0
            def __init__(self): self.process = self; self.waited = None
            def communicate(self, timeout): return b'complete', b''
            def wait_for_descendants(self, remaining): self.waited = remaining; return True
            def __enter__(self): return self
            def __exit__(self, *args): pass
        tree = Tree(); observation = {}
        with patch.object(package, 'native_webview_probe', return_value=False):
            self.assertEqual(package.child(['help'], process_factory=lambda *a, **k: tree, observation=observation), b'complete')
        self.assertTrue(observation['natural_release'])
        self.assertGreater(tree.waited, 0)
        self.assertLessEqual(tree.waited, observation['capture_bound_seconds'])

if __name__ == '__main__': unittest.main()
