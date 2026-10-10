import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('apt_mirror', Path(__file__).parents[1] / 'scripts/prepare-apt-mirror.py')
mirror = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mirror)


class AptMirrorTests(unittest.TestCase):
    def test_runner_mirror_indirection_retains_priorities_and_repository_policy(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'sources.list.d').mkdir()
            source = root / 'sources.list.d/ubuntu.sources'
            original = ('Types: deb\nURIs: mirror+file:/etc/apt/apt-mirrors.txt\n'
                        'Suites: noble noble-updates noble-backports\nComponents: main restricted universe multiverse\n'
                        'Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg\n')
            source.write_text(original, encoding='utf-8')
            selection = root / 'apt-mirrors.txt'
            selection.write_text('http://azure.archive.ubuntu.com/ubuntu/\tpriority:1\n'
                                 'https://archive.ubuntu.com/ubuntu/\tpriority:2\n'
                                 'https://security.ubuntu.com/ubuntu/\tpriority:3\n', encoding='utf-8')
            self.assertEqual(mirror.prepare(root), 1)
            self.assertEqual(source.read_text(), original)
            self.assertEqual(selection.read_text(), 'https://archive.ubuntu.com/ubuntu/\tpriority:1\n'
                             'https://archive.ubuntu.com/ubuntu/\tpriority:2\n'
                             'https://security.ubuntu.com/ubuntu/\tpriority:3\n')
            self.assertEqual(mirror.prepare(root), 0)

    def test_legacy_and_deb822_sources_keep_signatures_suites_and_other_hosts(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'sources.list.d').mkdir()
            originals = {
                'sources.list': 'deb [signed-by=/keys/ubuntu.gpg] http://azure.archive.ubuntu.com/ubuntu noble main universe\n',
                'sources.list.d/ubuntu.sources': 'URIs: https://azure.archive.ubuntu.com/ubuntu/\nSuites: noble-security\nSigned-By: /keys/ubuntu.gpg\n',
                'sources.list.d/vendor.list': 'deb https://azure.archive.ubuntu.com/ubuntu-example noble main\ndeb https://packages.example.org/repo stable main\n',
            }
            for name, value in originals.items():
                (root / name).write_text(value, encoding='utf-8')
            self.assertEqual(mirror.prepare(root), 2)
            for name, value in originals.items():
                expected = value if name.endswith('vendor.list') else value.replace('http://azure.archive.ubuntu.com/ubuntu', 'https://archive.ubuntu.com/ubuntu').replace('https://azure.archive.ubuntu.com/ubuntu', 'https://archive.ubuntu.com/ubuntu')
                self.assertEqual((root / name).read_text(), expected)
            self.assertFalse(any(path.read_bytes().startswith(b'\xef\xbb\xbf') for path in root.rglob('*') if path.is_file()))

    def test_missing_repository_files_do_not_create_configuration(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.assertEqual(mirror.prepare(root), 0)
            self.assertEqual(list(root.iterdir()), [])
