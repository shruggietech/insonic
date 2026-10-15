# SPDX-License-Identifier: Apache-2.0
import hashlib
import importlib.util
import io
from pathlib import Path
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, ROOT / 'scripts' / path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


bootstrap = module('bootstrap_actions', 'bootstrap-actions.py')


def archive(members=None):
    out = io.BytesIO()
    with tarfile.open(fileobj=out, mode='w:gz') as bundle:
        for name, data, kind in members or [('setup-go-' + 'a' * 40 + '/action.yml', b'runs: node24', tarfile.REGTYPE)]:
            item = tarfile.TarInfo(name); item.type = kind
            item.size = len(data) if kind == tarfile.REGTYPE else 0
            bundle.addfile(item, io.BytesIO(data) if item.size else None)
    return out.getvalue()


class BootstrapTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.output = Path(self.temp.name) / 'setup-go'
        self.payload = archive()
        self.lock = {'revision': 'a' * 40, 'sha256': hashlib.sha256(self.payload).hexdigest()}

    def run_bootstrap(self, data=None):
        return bootstrap.acquire('setup-go', self.output, self.lock,
                                 reader=lambda url, **kwargs: self.payload if data is None else data)

    def test_verified_action_is_atomically_available(self):
        receipt = self.run_bootstrap()
        self.assertEqual((self.output / 'action.yml').read_bytes(), b'runs: node24')
        self.assertEqual(receipt['revision'], 'a' * 40)

    def test_digest_mismatch_leaves_no_destination_or_staging(self):
        with self.assertRaisesRegex(ValueError, 'digest'):
            self.run_bootstrap(self.payload + b'corrupt')
        self.assertEqual(list(self.output.parent.iterdir()), [])

    def test_existing_destination_refused(self):
        self.output.mkdir()
        with self.assertRaisesRegex(ValueError, 'new'):
            self.run_bootstrap()

    def test_unsafe_or_duplicate_members_refused(self):
        prefix = 'setup-go-' + 'a' * 40 + '/'
        for members in [
            [(prefix + '../escape', b'x', tarfile.REGTYPE)],
            [('/absolute', b'x', tarfile.REGTYPE)],
            [(prefix + 'link', b'', tarfile.SYMTYPE)],
            [(prefix + 'hardlink', b'', tarfile.LNKTYPE)],
            [(prefix + 'a', b'x', tarfile.REGTYPE), (prefix + 'A', b'x', tarfile.REGTYPE)],
            [(prefix + 'a\\escape', b'x', tarfile.REGTYPE)],
            [(prefix + '.git/config', b'x', tarfile.REGTYPE)],
            [(prefix + 'a.', b'x', tarfile.REGTYPE)],
            [(prefix + 'NUL.txt', b'x', tarfile.REGTYPE)],
            [('other-root/action.yml', b'x', tarfile.REGTYPE)],
        ]:
            with self.subTest(members=members):
                data = archive(members)
                self.lock['sha256'] = hashlib.sha256(data).hexdigest()
                with self.assertRaises(ValueError): self.run_bootstrap(data)
                self.assertFalse(self.output.exists())

    def test_compressed_and_expanded_limits(self):
        with self.assertRaisesRegex(ValueError, 'size'):
            self.run_bootstrap(b'x' * (bootstrap.MAX_COMPRESSED + 1))
        data = archive([('setup-go-' + 'a' * 40 + '/huge', b'x' * 50, tarfile.REGTYPE)])
        self.lock['sha256'] = hashlib.sha256(data).hexdigest()
        old = bootstrap.MAX_EXPANDED
        try:
            bootstrap.MAX_EXPANDED = 20
            with self.assertRaisesRegex(ValueError, 'size'): self.run_bootstrap(data)
        finally: bootstrap.MAX_EXPANDED = old

    def test_unapproved_action_and_lock_rejected_before_network(self):
        for name, lock in [('other', self.lock), ('setup-go', {'revision': 'main', 'sha256': 'x'}), ('setup-go', {'revision': 1, 'sha256': []})]:
            with self.assertRaises(ValueError):
                bootstrap.acquire(name, self.output, lock, reader=lambda *a, **k: self.fail('network'))

    def test_invalid_and_excessive_archive_fails_without_output(self):
        for data in (b'not a tar', self.payload[:20]):
            self.lock['sha256'] = hashlib.sha256(data).hexdigest()
            with self.assertRaises(ValueError): self.run_bootstrap(data)
            self.assertFalse(self.output.exists())
        old = bootstrap.MAX_MEMBERS
        try:
            bootstrap.MAX_MEMBERS = 0
            with self.assertRaises(ValueError): self.run_bootstrap()
        finally: bootstrap.MAX_MEMBERS = old


if __name__ == '__main__': unittest.main()
