# SPDX-License-Identifier: Apache-2.0
from concurrent.futures import ThreadPoolExecutor
import importlib.util
from pathlib import Path
import sys
import threading
import unittest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / 'scripts'))
spec = importlib.util.spec_from_file_location('ci_package_inputs', ROOT / 'scripts/ci-package-inputs.py')
inputs = importlib.util.module_from_spec(spec); spec.loader.exec_module(inputs)


class ParallelInputTests(unittest.TestCase):
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
