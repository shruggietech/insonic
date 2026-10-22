# SPDX-License-Identifier: Apache-2.0
from concurrent.futures import ThreadPoolExecutor
import importlib.util
from pathlib import Path
import sys
from unittest.mock import patch
import threading
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / 'scripts'))
spec = importlib.util.spec_from_file_location('ci_package_inputs', ROOT / 'scripts/ci-package-inputs.py')
inputs = importlib.util.module_from_spec(spec); spec.loader.exec_module(inputs)


class ParallelInputTests(unittest.TestCase):
    def test_both_notice_installs_keep_audits_and_bounded_fetches(self):
        with tempfile.TemporaryDirectory() as temporary:
            node=Path(temporary)/'selected node'/'node.exe';node.parent.mkdir()
            npm=node.parent/'node_modules/npm/bin/npm-cli.js';npm.parent.mkdir(parents=True);npm.write_text('',encoding='utf-8')
            with patch.object(inputs.shutil,'which',return_value=str(node)),patch.dict(inputs.os.environ,{'SOURCE_REVISION':'a'*40}),patch.object(inputs,'run',return_value='verified') as run,patch('builtins.print'):
                inputs.main()
            installs=[call.args[0] for call in run.call_args_list if '--prefix' in call.args[0]]
            self.assertEqual({args[args.index('--prefix')+1] for args in installs},{'desktop/frontend','site'})
            self.assertEqual(len(installs),2)
            for args in installs:
                options=dict(value[2:].split('=',1) for value in args if str(value).startswith('--') and '=' in value)
                self.assertEqual(options.get('audit'),'true');self.assertNotIn('--no-audit',args)
                timeout=int(options.get('fetch-timeout','300000'));retries=int(options.get('fetch-retries','2'))
                self.assertLessEqual(timeout,15000);self.assertGreater(timeout,0);self.assertLessEqual(retries,2)
                budget=timeout*(retries+1)+sum(min(int(options['fetch-retry-mintimeout'])*2**i,int(options['fetch-retry-maxtimeout'])) for i in range(retries))
                self.assertLess(budget*2,180000,'bulk and fallback requests must fit inside the unchanged stage bound')

    def test_original_deadline_accepts_above_target_without_extending_operation_limit(self):
        for remaining in (700, 900, 12):
            with self.subTest(remaining=remaining), patch.dict(inputs.os.environ, {
                    'INSONIC_QUALIFICATION_DEADLINE': str((1000 + remaining) * 1000)}), \
                    patch.object(inputs.time, 'time', return_value=1000), \
                    patch.object(inputs, 'ProcessTree') as factory:
                child = factory.return_value.__enter__.return_value
                child.process.communicate.return_value = (b'verified', b'')
                child.process.returncode = 0
                self.assertEqual(inputs.run(['fake-verifier']), 'verified')
                child.process.communicate.assert_called_once_with(timeout=min(180, remaining))

    def test_expired_or_above_hard_limit_deadline_never_starts_child(self):
        for remaining in (0, -1, 900.001):
            with self.subTest(remaining=remaining), patch.dict(inputs.os.environ, {
                    'INSONIC_QUALIFICATION_DEADLINE': str(round((1000 + remaining) * 1000))}), \
                    patch.object(inputs.time, 'time', return_value=1000), \
                    patch.object(inputs, 'ProcessTree') as factory:
                with self.assertRaisesRegex(ValueError, 'original qualification deadline'):
                    inputs.run(['fake-verifier'])
                factory.assert_not_called()

    def test_independent_preparation_overlaps_and_every_result_joins(self):
        barrier = threading.Barrier(4, timeout=3)
        done = []
        def task(index):
            barrier.wait()
            done.append(index)
            return index
        self.assertEqual(inputs.join_inputs([lambda i=i: task(i) for i in range(4)]), [0, 1, 2, 3])
        self.assertEqual(sorted(done), [0, 1, 2, 3])

    def test_failure_joins_other_work_without_reporting_success(self):
        barrier = threading.Barrier(2, timeout=3)
        finished = threading.Event()
        def fail():
            barrier.wait()
            raise ValueError('verification failed')
        def complete():
            barrier.wait()
            finished.set()
        with self.assertRaisesRegex(ValueError, 'verification failed'):
            inputs.join_inputs([fail, complete])
        self.assertTrue(finished.is_set())


if __name__ == '__main__': unittest.main()
