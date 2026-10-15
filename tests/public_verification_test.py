# SPDX-License-Identifier: Apache-2.0
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('public_verification', ROOT / 'scripts/public-verification.py')
public = importlib.util.module_from_spec(spec); spec.loader.exec_module(public)
VERSION = '1.2.3'
REVISION = 'a' * 40
COMMIT = 'b' * 40


class PublicTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(); self.addCleanup(self.temp.cleanup)
        self.archive = Path(self.temp.name) / 'documentation.tar.gz'
        self.asset = 'https://github.com/shruggietech/insonic/releases/download/v1.2.3/insonic.zip'
        self.files = {
            'index.html': ('<a href="docs/latest/index.html">Docs</a><a href="' + self.asset + '">Package</a><script src="/app.js"></script>').encode(),
            'docs/index.html': b'<a href="latest/index.html">Latest</a>',
            'docs/latest/index.html': b'<link rel="canonical" href="https://insonic.io/docs/v1.2.3/index/">',
            'docs/v1.2.3/index.html': b'<a href="index/">Start</a>',
            'docs/v1.2.3/index/index.html': b'<link rel="stylesheet" href="/style.css">',
            'app.js': b'console.log("fixture")', 'style.css': b'body{color:blue}',
            'schemas/v1.2.3/master.schema.json': b'{"type":"object"}',
            'release-documentation.json': json.dumps({'kind': 'documentation', 'version': VERSION, 'revision': REVISION}).encode(),
            'documentation-build.json': json.dumps({'version': VERSION, 'revision': REVISION, 'source_dirty': False, 'documentation_target': 'release'}).encode(),
            'documentation-manifest.json': json.dumps({'latest': VERSION, 'versions': [{'version': VERSION, 'status': 'released', 'pages': [{'slug': 'index'}]}]}).encode(),
        }
        self.calls = []
        self.live = self.files.copy()

    def pack(self):
        with tarfile.open(self.archive, 'w:gz') as archive:
            for name, data in self.files.items():
                member = tarfile.TarInfo(name); member.size = len(data)
                archive.addfile(member, io.BytesIO(data))

    def reader(self, url, **kwargs):
        self.calls.append(url)
        path = url.removeprefix('https://example.com/')
        if not path or path.endswith('/'): path += 'index.html'
        return self.live[path]

    def verify(self, **kwargs):
        self.pack()
        return public.verify(self.archive, VERSION, REVISION, 'https://example.com/',
                             package_urls=[self.asset], reader=self.reader, **kwargs)

    def test_exact_original_pages_schemas_and_resources(self):
        result = self.verify()
        self.assertEqual(result['version'], VERSION)
        paths = {x['path'] for x in result['paths']}
        self.assertTrue({'app.js', 'style.css', 'schemas/v1.2.3/master.schema.json', 'docs/latest/index.html'} <= paths)
        self.assertEqual(result['content_proof'], 'downloaded-original-byte-match')

    def test_stale_content_fails_even_with_matching_marker(self):
        for path in ('index.html', 'app.js', 'style.css', 'schemas/v1.2.3/master.schema.json'):
            with self.subTest(path=path):
                self.live = self.files.copy(); self.live[path] = b'stale'
                with self.assertRaisesRegex(ValueError, 'differ'): self.verify()

    def test_identity_latest_alias_and_package_link_contracts(self):
        for path, data in [
            ('documentation-build.json', b'{}'),
            ('documentation-manifest.json', b'{"latest":"0.0.0","versions":[]}'),
            ('docs/latest/index.html', b'<link rel="canonical" href="https://insonic.io/docs/v0.0.0/index/">'),
            ('index.html', b'<a href="docs/v0.0.0/">Old docs</a>'),
        ]:
            original = self.files[path]
            with self.subTest(path=path):
                self.files[path] = data
                with self.assertRaises(ValueError): self.verify()
                self.files[path] = original

    def test_missing_local_resource_fails_without_external_fetch(self):
        self.files['docs/v1.2.3/index/index.html'] = b'<script src="/missing.js"></script>'
        with self.assertRaisesRegex(ValueError, 'resource'): self.verify()
        self.assertFalse(self.calls)

    def test_destination_rejected_before_network(self):
        self.pack()
        for site in ('http://example.com/', 'https://user:secret@example.com/', 'https://example.com/?token=secret'):
            with self.assertRaises(ValueError): public.verify(self.archive, VERSION, REVISION, site, package_urls=[], reader=self.reader)
        self.assertFalse(self.calls)

    def test_prefixed_site_checks_actual_root_relative_resources(self):
        self.pack()
        requests = []
        def reader(url, **kwargs):
            requests.append(url)
            name = url.removeprefix('https://example.com/').removeprefix('mirror/')
            if not name or name.endswith('/'): name += 'index.html'
            return self.files[name]
        public.verify(self.archive, VERSION, REVISION, 'https://example.com/mirror/',
                      package_urls=[self.asset], reader=reader)
        self.assertIn('https://example.com/app.js', requests)
        self.assertNotIn('https://example.com/mirror/app.js', requests)
        self.assertIn('https://example.com/mirror/schemas/v1.2.3/master.schema.json', requests)
        self.assertIn('https://example.com/style.css', requests)

    def test_external_render_resource_and_query_are_rejected(self):
        for value in ('https://evil.test/track.js', '/app.js?token=secret'):
            self.files['docs/v1.2.3/index/index.html'] = f'<script src="{value}"></script>'.encode()
            with self.assertRaises(ValueError): self.verify()
        self.assertFalse(self.calls)

    def test_pages_identity_before_and_after_and_no_writes(self):
        class API:
            def __init__(inner): inner.count = 0
            def call(inner, method, endpoint):
                self.assertEqual(method, 'GET'); inner.count += 1
                if endpoint.endswith('/pages/builds/latest'): return {'status': 'built', 'commit': COMMIT}
                return {'object': {'type': 'commit', 'sha': COMMIT}}
        api = API()
        result = self.verify(api=api, pages_commit=COMMIT)
        self.assertEqual(result['pages_commit'], COMMIT)
        self.assertEqual(api.count, 4)
        class RacingAPI(API):
            def call(inner, method, endpoint):
                result = super().call(method, endpoint)
                if inner.count > 2: return {'object': {'sha': 'c' * 40}}
                return result
        with self.assertRaisesRegex(ValueError, 'Pages'): self.verify(api=RacingAPI(), pages_commit=COMMIT)


if __name__ == '__main__': unittest.main()
