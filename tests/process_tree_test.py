# SPDX-License-Identifier: Apache-2.0
import os
from pathlib import Path
import shutil
import sys
import subprocess
import time
import tempfile
import unittest
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
import qualification_timing as test_timing
from process_tree import ProcessTree, selected_executable, native_webview_environment, NATIVE_PROFILE
import process_tree


class SelectedChildEnvironment(unittest.TestCase):
    def test_session_inventory_includes_nested_groups_and_excludes_other_sessions_and_zombies(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for pid, group, session, state in [(100, 100, 100, 'S'), (200, 200, 100, 'S'),
                                                (300, 300, 300, 'S'), (400, 400, 100, 'Z')]:
                directory = root / str(pid)
                directory.mkdir()
                fields = [state, '1', str(group), str(session)] + ['0'] * 15 + [str(pid + 10)]
                (directory / 'stat').write_bytes(str(pid).encode() + b' (name with ) \xff) ' + ' '.join(fields).encode())
            self.assertEqual(process_tree.unix_session_members(100, root), {100: 110, 200: 210})

    def test_unix_pid_reuse_cannot_redirect_termination(self):
        with patch.object(process_tree.os, 'pidfd_open', return_value=9, create=True), \
                patch.object(process_tree, 'unix_process_identity', return_value=('S', 123, 999)), \
                patch.object(process_tree.signal, 'pidfd_send_signal', create=True) as send, \
                patch.object(process_tree.os, 'close') as close:
            process_tree.kill_unix_member(456, 789, 123)
            send.assert_not_called()
            close.assert_called_once_with(9)

    def test_unix_wait_cannot_report_empty_while_a_nested_group_is_alive(self):
        tree = ProcessTree.__new__(ProcessTree)
        tree.job, tree.process, tree.session_id = None, Mock(), 123
        with patch('process_tree.unix_session_members', return_value={456: 789}, create=True):
            self.assertFalse(tree.wait_for_descendants(.01))

    @unittest.skipUnless(sys.platform.startswith('linux'), 'actual Linux nested process groups')
    def test_unix_nested_group_is_observed_then_killed_without_touching_other_session(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            ready = root / 'ready'
            worker = "import os,pathlib,sys,time;os.setpgid(0,0);pathlib.Path(sys.argv[1]).write_text(str(os.getpid()));time.sleep(60)"
            parent = "import subprocess,sys;subprocess.Popen([sys.executable,'-c',sys.argv[1],sys.argv[2]],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)"
            unrelated = subprocess.Popen([sys.executable, '-c', 'import time;time.sleep(60)'],
                                         stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                         start_new_session=True)
            try:
                with ProcessTree([sys.executable, '-c', parent, worker, ready], cwd=root, env=dict(os.environ)) as tree:
                    tree.process.communicate(timeout=test_timing.READINESS_SECONDS)
                    deadline = time.monotonic() + test_timing.READINESS_SECONDS
                    while not ready.exists() and time.monotonic() < deadline:
                        time.sleep(.01)
                    self.assertTrue(ready.exists())
                    self.assertFalse(tree.wait_for_descendants(.02))
                    tree.kill()
                    self.assertTrue(tree.wait_for_descendants(test_timing.NATURAL_RELEASE_SECONDS))
                    self.assertIsNone(unrelated.poll())
            finally:
                unrelated.kill()
                unrelated.wait(timeout=test_timing.CLEANUP_SECONDS)

    def test_selected_path_owns_bare_program_and_missing_program_cannot_fall_back(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            parent, selected = root / 'parent', root / 'selected'
            parent.mkdir(); selected.mkdir()
            for directory in [parent, selected]:
                (directory / 'compiler.EXE').write_bytes(b'fixture')
            with patch('process_tree.os.name', 'nt'):
                self.assertEqual(selected_executable(['compiler', '--version'], root, {'PATH': str(selected), 'PATHEXT': '.EXE'})[0], str(selected / 'compiler.EXE'))
                with self.assertRaises(FileNotFoundError):
                    selected_executable(['compiler'], root, {'PATH': str(root / 'missing')})
                explicit = str(parent / 'compiler.EXE')
                self.assertEqual(selected_executable([explicit], root, {'PATH': str(selected)}), [explicit])

    @unittest.skipUnless(os.name == 'nt', 'actual Windows executable discovery')
    def test_windows_hidden_child_uses_selected_executable(self):
        with tempfile.TemporaryDirectory() as temporary:
            selected = Path(temporary) / 'selected'
            selected.mkdir()
            executable = selected / 'selected-python.exe'
            shutil.copyfile(sys.executable, executable)
            environment = dict(os.environ, PATH=str(selected) + os.pathsep + os.environ.get('PATH', ''))
            with ProcessTree(['selected-python', '-c', 'import sys; print(sys.executable)'], cwd=temporary, env=environment) as tree:
                output, error = tree.process.communicate(timeout=test_timing.READINESS_SECONDS)
                self.assertEqual(tree.process.returncode, 0, error.decode(errors='replace'))
                self.assertEqual(Path(output.decode().strip()).resolve(), executable.resolve())


    @unittest.skipUnless(os.name == 'nt', 'actual Windows job accounting')
    def test_normal_root_return_waits_for_owned_descendant_without_killing_it(self):
        with tempfile.TemporaryDirectory() as temporary:
            marker = Path(temporary) / 'released'
            worker = "import pathlib,sys,time; p=pathlib.Path(sys.argv[1]); deadline=time.monotonic()+10\nwhile not p.exists() and time.monotonic()<deadline:time.sleep(.02)\nif p.exists():p.write_text('natural')"
            parent = "import subprocess,sys;subprocess.Popen([sys.executable,'-c',sys.argv[1],sys.argv[2]],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW)"
            with ProcessTree([sys.executable, '-c', parent, worker, marker], cwd=temporary, env=dict(os.environ)) as tree:
                tree.process.communicate(timeout=test_timing.READINESS_SECONDS)
                self.assertEqual(tree.process.returncode, 0)
                self.assertFalse(tree.wait_for_descendants(.02))
                marker.write_text('release')
                self.assertTrue(tree.wait_for_descendants(test_timing.NATURAL_RELEASE_SECONDS))
                self.assertEqual(marker.read_text(), 'natural')

    @unittest.skipUnless(os.name == 'nt', 'actual Windows qualification profile')
    def test_profiles_are_unique_fresh_and_retained_on_failure(self):
        paths = []
        for _ in range(2):
            with native_webview_environment({'retained': 'value'}) as env:
                profile = Path(env[NATIVE_PROFILE]); paths.append(profile)
                self.assertTrue(profile.is_absolute()); self.assertFalse(profile.exists())
                self.assertEqual(env['retained'], 'value')
                profile.mkdir(); (profile / 'fixture').write_text('owned')
            self.assertFalse(profile.parent.exists())
        self.assertNotEqual(paths[0], paths[1])
        try:
            with self.assertRaisesRegex(RuntimeError, 'native failure'):
                with native_webview_environment({}) as env:
                    profile = Path(env[NATIVE_PROFILE]); profile.mkdir()
                    raise RuntimeError('native failure')
            self.assertTrue(profile.is_dir())
        finally:
            shutil.rmtree(profile.parent)


if __name__ == '__main__':
    unittest.main()
