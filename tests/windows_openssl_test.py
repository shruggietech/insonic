# SPDX-License-Identifier: Apache-2.0
import importlib.util
import io
import hashlib
from pathlib import Path
import tarfile
import tempfile
import unittest
import urllib.error
from unittest.mock import patch
ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('windows_openssl', ROOT / 'scripts/windows_openssl.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
class OpenSSLArchiveTests(unittest.TestCase):
    def test_transient_download_retries_without_relaxing_identity(self):
        payload = b'pinned archive'
        pin = {'url': 'https://example.invalid/openssl', 'sha256': hashlib.sha256(payload).hexdigest()}
        with tempfile.TemporaryDirectory() as temporary:
            target = Path(temporary) / 'openssl.conda'
            with patch.object(module.urllib.request, 'urlopen', side_effect=[TimeoutError('read timed out'), io.BytesIO(payload)]) as fetch, patch.object(module.time, 'sleep'):
                module.fetch_archive(pin, target)
                self.assertEqual(target.read_bytes(), payload)
                self.assertEqual(fetch.call_count, 2)
                self.assertTrue(all(0 < call.kwargs['timeout'] <= 20 for call in fetch.call_args_list))
            with patch.object(module.urllib.request, 'urlopen', side_effect=AssertionError('verified cache requires no network')):
                module.fetch_archive(pin, target)
            self.assertFalse(target.with_suffix('.download').exists())

    def test_corrupt_and_permanent_failures_are_not_retried_or_cached(self):
        pin = {'url': 'https://example.invalid/openssl', 'sha256': hashlib.sha256(b'pinned').hexdigest()}
        for response, expected in [(io.BytesIO(b'corrupt'), ValueError), (urllib.error.HTTPError(pin['url'], 404, 'missing', {}, None), urllib.error.HTTPError)]:
            with tempfile.TemporaryDirectory() as temporary:
                target = Path(temporary) / 'openssl.conda'
                with patch.object(module.urllib.request, 'urlopen', side_effect=[response]) as fetch, patch.object(module.time, 'sleep'):
                    with self.assertRaises(expected):
                        module.fetch_archive(pin, target)
                    self.assertEqual(fetch.call_count, 1)
                self.assertFalse(target.exists())
                self.assertFalse(target.with_suffix('.download').exists())

    def test_repeated_network_failures_are_bounded(self):
        pin = {'url': 'https://example.invalid/openssl', 'sha256': '0' * 64}
        with tempfile.TemporaryDirectory() as temporary:
            target = Path(temporary) / 'openssl.conda'
            with patch.object(module.urllib.request, 'urlopen', side_effect=TimeoutError('read timed out')) as fetch, patch.object(module.time, 'sleep'):
                with self.assertRaises(TimeoutError):
                    module.fetch_archive(pin, target)
                self.assertEqual(fetch.call_count, 3)
            self.assertFalse(target.with_suffix('.download').exists())

    def test_total_acquisition_budget_stops_further_network_attempts(self):
        pin = {'url': 'https://example.invalid/openssl', 'sha256': '0' * 64}
        with tempfile.TemporaryDirectory() as temporary:
            with patch.object(module.time, 'monotonic', side_effect=[0, 61]), patch.object(module.urllib.request, 'urlopen') as fetch:
                with self.assertRaisesRegex(TimeoutError, 'budget exceeded'):
                    module.fetch_archive(pin, Path(temporary) / 'openssl.conda')
                fetch.assert_not_called()

    def archive(self, entries):
        data = io.BytesIO()
        with tarfile.open(fileobj=data, mode='w') as archive:
            for name, kind in entries:
                info = tarfile.TarInfo(name)
                info.type = kind
                info.size = 3 if kind == tarfile.REGTYPE else 0
                archive.addfile(info, io.BytesIO(b'DLL') if info.size else None)
        data.seek(0)
        return tarfile.open(fileobj=data, mode='r')
    def test_only_declared_regular_companions_are_written(self):
        with tempfile.TemporaryDirectory() as temporary:
            destination = Path(temporary)
            found = set()
            with self.archive([('../outside', tarfile.REGTYPE), ('Library/bin/libssl-3-x64.dll', tarfile.REGTYPE)]) as source:
                module.extract_members(source, destination, found)
            self.assertEqual(found, {'libssl-3-x64.dll'})
            self.assertEqual(list(destination.iterdir()), [destination / 'libssl-3-x64.dll'])
    def test_declared_links_and_duplicates_fail(self):
        for entries in [
            [('Library/bin/libssl-3-x64.dll', tarfile.SYMTYPE)],
            [('Library/bin/libssl-3-x64.dll', tarfile.REGTYPE)] * 2]:
            with tempfile.TemporaryDirectory() as temporary:
                with self.archive(entries) as source:
                    with self.assertRaises(ValueError):
                        module.extract_members(source, Path(temporary), set())
