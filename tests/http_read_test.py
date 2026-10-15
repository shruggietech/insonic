# SPDX-License-Identifier: Apache-2.0
import importlib.util
from pathlib import Path
import unittest
import ssl

spec = importlib.util.spec_from_file_location('http_read', Path(__file__).resolve().parents[1] / 'scripts/http_read.py')
http = importlib.util.module_from_spec(spec); spec.loader.exec_module(http)


class HttpTests(unittest.TestCase):
    def run_read(self, responses, **options):
        self.calls = []
        self.now = 0
        def transport(url, limit, deadline, clock):
            self.calls.append(url)
            value = responses.pop(0)
            if isinstance(value, Exception): raise value
            return value
        def sleep(seconds): self.now += seconds
        return http.read('https://example.com/a', max_bytes=100, deadline=10,
                         transport=transport, sleep=sleep, clock=lambda: self.now, **options)

    def test_transient_recovery_and_exhaustion(self):
        self.assertEqual(self.run_read([(503, {}, b''), (429, {}, b''), (200, {}, b'ok')]), b'ok')
        self.assertEqual(len(self.calls), 3)
        with self.assertRaises(OSError): self.run_read([(503, {}, b'')] * 3)
        self.assertEqual(len(self.calls), 3)

    def test_pinned_archive_404_is_explicit_only(self):
        with self.assertRaises(OSError): self.run_read([(404, {}, b'')])
        self.assertEqual(len(self.calls), 1)
        self.assertEqual(self.run_read([(404, {}, b''), (200, {}, b'ok')], retry_404=True), b'ok')

    def test_authentication_errors_do_not_retry(self):
        for status in (401, 403):
            with self.assertRaises(OSError): self.run_read([(status, {}, b'')])
            self.assertEqual(len(self.calls), 1)

    def test_transport_error_retries(self):
        self.assertEqual(self.run_read([TimeoutError(), (200, {}, b'ok')]), b'ok')

    def test_certificate_errors_are_terminal(self):
        with self.assertRaisesRegex(ValueError, 'certificate'):
            self.run_read([ssl.SSLCertVerificationError('fixture')])
        self.assertEqual(len(self.calls), 1)

    def test_redirects_are_confined_before_request(self):
        for target in ('http://example.com/b', 'https://evil.test/', 'https://user:secret@example.com/', 'https://example.com/a?token=secret'):
            with self.assertRaises(ValueError): self.run_read([(302, {'location': target}, b'')])
            self.assertEqual(len(self.calls), 1)

    def test_bounded_redirects_and_response_size(self):
        with self.assertRaisesRegex(ValueError, 'redirect'):
            self.run_read([(302, {'location': '/b'}, b'')] * 6)
        with self.assertRaisesRegex(ValueError, 'size'):
            self.run_read([(200, {}, b'x' * 101)])

    def test_deadline(self):
        with self.assertRaisesRegex(TimeoutError, 'deadline'):
            http.read('https://example.com/a', max_bytes=1, deadline=0, clock=lambda: 1,
                      transport=lambda *args: self.fail('network'))


if __name__ == '__main__': unittest.main()
