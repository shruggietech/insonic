# SPDX-License-Identifier: Apache-2.0
import os
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from process_tree import ProcessTree, selected_executable


class SelectedChildEnvironment(unittest.TestCase):
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
                output, error = tree.process.communicate(timeout=15)
                self.assertEqual(tree.process.returncode, 0, error.decode(errors='replace'))
                self.assertEqual(Path(output.decode().strip()).resolve(), executable.resolve())


if __name__ == '__main__':
    unittest.main()
