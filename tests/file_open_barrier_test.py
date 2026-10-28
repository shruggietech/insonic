# SPDX-License-Identifier: Apache-2.0
from pathlib import Path
import os
import sys
import tempfile
import time
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from file_open_barrier import FileOpenBarrier, validate_event
from process_tree import ProcessTree


class FileOpenBarrierTests(unittest.TestCase):
    def test_real_open_is_blocked_until_owned_barrier_releases(self):
        with tempfile.TemporaryDirectory(prefix='insonic-barrier-') as root:
            source = Path(root) / 'owned source café.txt'
            source.write_bytes(b'original owned bytes')
            deadline = time.monotonic() + 20
            barrier = FileOpenBarrier(source, cwd=root, env=os.environ.copy(), deadline=deadline)
            with barrier:
                with ProcessTree([sys.executable, '-c',
                                  'from pathlib import Path; import sys; print(Path(sys.argv[1]).read_text())', str(source)],
                                 cwd=root, env=os.environ.copy()) as opener:
                    observed = barrier.wait_break()
                    self.assertTrue(observed['ack_required'])
                    self.assertIsNone(opener.process.poll())
                    self.assertGreater(barrier.remaining(), 0)
                    barrier.release()
                    out, err = opener.process.communicate(timeout=5)
                    self.assertEqual((opener.process.returncode, out.strip(), err), (0, b'original owned bytes', b''))
            self.assertTrue(barrier.released)

    def test_failure_before_break_still_releases_helper(self):
        with tempfile.TemporaryDirectory(prefix='insonic-barrier-') as root:
            source = Path(root) / 'owned.txt'
            source.write_bytes(b'x')
            barrier = FileOpenBarrier(source, cwd=root, env=os.environ.copy(), deadline=time.monotonic() + 10)
            with self.assertRaisesRegex(RuntimeError, 'fixture failure'):
                with barrier:
                    raise RuntimeError('fixture failure')
            self.assertTrue(barrier.released)
            self.assertEqual(source.read_bytes(), b'x')

    def test_expired_parent_and_nonregular_input_are_refused(self):
        with tempfile.TemporaryDirectory(prefix='insonic-barrier-') as root:
            for source, deadline in [(Path(root), time.monotonic() + 5),
                                     (Path(root) / 'missing', time.monotonic() - 1),
                                     (Path(root) / 'missing', float('nan'))]:
                with self.assertRaises((ValueError, TimeoutError, OSError)):
                    with FileOpenBarrier(source, cwd=root, env=os.environ.copy(), deadline=deadline):
                        self.fail('Invalid barrier was accepted')

    def test_missing_break_deadline_fails_and_releases_ownership(self):
        with tempfile.TemporaryDirectory(prefix='insonic-barrier-') as root:
            source = Path(root) / 'owned.txt'
            source.write_bytes(b'unchanged')
            barrier = FileOpenBarrier(source, cwd=root, env=os.environ.copy(), deadline=time.monotonic() + 1)
            with self.assertRaises((ValueError, TimeoutError)):
                with barrier:
                    barrier.wait_break()
            self.assertIsNone(barrier.owner)
            self.assertEqual(source.read_bytes(), b'unchanged')

    def test_break_evidence_requires_live_owned_identity_and_ack(self):
        valid = {'state': 'break', 'pid': 123, 'mechanism': 'windows-rw-oplock',
                 'ack_required': True, 'break_deadline': time.monotonic() + 5}
        self.assertIsNone(validate_event(valid, 'break', 123, 'windows-rw-oplock'))
        for field, value in [('state', 'armed'), ('pid', 124), ('ack_required', False),
                             ('mechanism', 'fake'), ('break_deadline', time.monotonic() - 1),
                             ('break_deadline', float('nan'))]:
            with self.subTest(field=field):
                with self.assertRaises(ValueError):
                    validate_event({**valid, field: value}, 'break', 123, 'windows-rw-oplock')


if __name__ == '__main__':
    unittest.main()
