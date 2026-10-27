# SPDX-License-Identifier: Apache-2.0
import os
from pathlib import Path
import sys
import time
import unittest
from types import SimpleNamespace
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from terminal_transport import Terminal, normalize_terminal
import qualification_timing as test_timing


class TerminalTransportTests(unittest.TestCase):
    def test_actual_terminal_interrupt_and_natural_release(self):
        # A background launcher may pass Windows' ignore-Ctrl+C attribute to
        # Python. Elect reception in this disposable probe, never in the parent.
        # Only the terminal input byte delivers the interrupt.
        program = ("import os,signal,sys,time\n"
                   "if os.name == 'nt':\n"
                   " import ctypes\n"
                   " assert ctypes.windll.kernel32.SetConsoleCtrlHandler(None,False)\n"
                   "signal.signal(signal.SIGINT,lambda *_:(print('INTERRUPTED',flush=True),sys.exit(7)))\n"
                   "print('TTY:'+str(all(os.isatty(i) for i in (0,1,2))),flush=True)\n"
                   "while True: time.sleep(.02)\n")
        with Terminal([sys.executable, '-c', program], cwd=Path.cwd(), env=os.environ.copy()) as terminal:
            terminal.wait_for('TTY:True')
            terminal.interrupt()
            result = terminal.finish()
        self.assertEqual(result['exit_code'], 7)
        self.assertIn('INTERRUPTED', result['text'])
        self.assertTrue(result['natural_exit'])
        self.assertTrue(result['cleanup'])

    @unittest.skipUnless(os.name == 'nt', 'actual Windows Ctrl+C inheritance')
    def test_interrupt_probe_elects_reception_under_background_group_ancestry(self):
        from process_tree import ProcessTree
        import subprocess
        program = ("import sys,unittest\n"
                   "sys.path.insert(0,'tests')\n"
                   "from terminal_transport_test import TerminalTransportTests\n"
                   "suite=unittest.TestSuite([TerminalTransportTests('test_actual_terminal_interrupt_and_natural_release')])\n"
                   "result=unittest.TextTestRunner().run(suite)\n"
                   "sys.exit(not result.wasSuccessful())\n")
        # The isolated launcher propagates the native ignore flag exactly as a
        # background Windows host can. Parent console state is never changed.
        with ProcessTree([sys.executable, '-c', 'import subprocess,sys; p=subprocess.run([sys.executable,"-c",sys.argv[1]],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=0x08000200);sys.stdout.buffer.write(p.stdout);sys.stderr.buffer.write(p.stderr);sys.exit(p.returncode)', program],
                         cwd=Path.cwd(), env=os.environ.copy()) as owner:
            output, error = owner.process.communicate(timeout=test_timing.SUBPROCESS_SECONDS)
            self.assertEqual(owner.process.returncode, 0, error.decode(errors='replace'))
            self.assertTrue(owner.wait_for_descendants(test_timing.CLEANUP_SECONDS))

    def test_output_overflow_is_failure_even_after_natural_exit(self):
        with self.assertRaisesRegex(ValueError, 'capture bound'):
            with Terminal([sys.executable, '-c', "print('x'*200000)"], cwd=Path.cwd(), env=os.environ.copy()) as terminal:
                terminal.finish()

    def test_forced_cleanup_cannot_be_success(self):
        with self.assertRaisesRegex(RuntimeError, 'naturally'):
            with Terminal([sys.executable, '-c', "import time; print('READY',flush=True); time.sleep(600)"], cwd=Path.cwd(), env=os.environ.copy()) as terminal:
                terminal.wait_for('READY')

    def test_transport_controls_are_complete_and_utf8_is_strict(self):
        self.assertEqual(normalize_terminal(b'\x1b[?25lwork\r\n\x1b[0m'), 'work\n')
        for malformed in (b'\x1b[', b'\xff', b'\x1bZ'):
            with self.subTest(value=malformed), self.assertRaises(ValueError):
                normalize_terminal(malformed)

    def test_inherited_deadline_does_not_restart_for_the_client(self):
        with self.assertRaises(TimeoutError):
            with Terminal([sys.executable, '-c', 'import time; time.sleep(600)'],
                          cwd=Path.cwd(), env=os.environ.copy(), deadline=time.monotonic() + .1) as terminal:
                terminal.finish()

    def test_failed_descendant_observation_still_releases_console_and_handles(self):
        import terminal_transport as transport
        terminal = Terminal.__new__(Terminal)
        terminal.cleaned = False
        terminal.hpc, terminal.input, terminal.output = 123, 1, 2
        terminal.reader = Mock()
        terminal.reader.is_alive.return_value = False
        api = SimpleNamespace(ClosePseudoConsole=Mock(), CloseHandle=Mock(return_value=1))
        terminal.owner = SimpleNamespace(api=api, process=SimpleNamespace(handle=3), close=Mock(),
            wait_for_descendants=Mock(side_effect=RuntimeError('observation failed')))
        with patch.object(transport, 'os', SimpleNamespace(name='nt')), self.assertRaisesRegex(RuntimeError, 'observation failed'):
            terminal._cleanup(force=False)
        api.ClosePseudoConsole.assert_called_once_with(123)
        self.assertEqual(api.CloseHandle.call_count, 3)
        terminal.owner.close.assert_called_once()

    def test_cleanup_failure_preserves_the_original_failure(self):
        terminal = Terminal.__new__(Terminal)
        terminal.cleaned = False
        terminal._cleanup = Mock(side_effect=RuntimeError('cleanup failure'))
        primary = ValueError('original failure')
        terminal.__exit__(ValueError, primary, None)
        self.assertEqual(str(primary), 'original failure')
        self.assertTrue(primary.__notes__)


if __name__ == '__main__':
    unittest.main()
