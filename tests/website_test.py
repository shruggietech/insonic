# SPDX-License-Identifier: Apache-2.0
import importlib.util
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('website', ROOT / 'scripts/website.py')
website = importlib.util.module_from_spec(spec)
spec.loader.exec_module(website)
REV = 'a' * 40


class API:
    def __init__(self, runs=None):
        self.runs = runs or [dict(id=2, head_sha=REV, head_branch='main', event='push', path='.github/workflows/ci.yml', status='completed', conclusion='success', run_attempt=1)]
        self.writes = []
    def call(self, method, endpoint, value=None):
        if method != 'GET': self.writes.append((method, endpoint, value)); raise AssertionError('unexpected write')
        if endpoint.endswith('/git/ref/heads/main'): return {'object': {'type': 'commit', 'sha': REV}}
        if '/actions/workflows/ci.yml/runs?' in endpoint: return {'total_count': len(self.runs), 'workflow_runs': self.runs}
        if '/attempts/' in endpoint and '/jobs?' in endpoint:
            jobs = [{'name': name, 'status': 'completed', 'conclusion': 'success'} for name in ('foundation', 'docs')]
            return {'total_count': len(jobs), 'jobs': jobs}
        raise AssertionError(endpoint)


class WebsiteTest(unittest.TestCase):
    def test_exact_current_newest_green_main(self):
        api = API(); self.assertEqual(website.green_main(api, REV), 2); self.assertEqual(api.writes, [])
        with self.assertRaises(ValueError): website.green_main(api, 'b' * 40)

    def test_old_success_cannot_hide_new_failure_or_pending_attempt(self):
        for status, conclusion in [('completed', 'failure'), ('in_progress', None), ('completed', 'cancelled')]:
            api = API(); newest = dict(api.runs[0], id=3, status=status, conclusion=conclusion)
            api.runs.insert(0, newest)
            with self.assertRaises(ValueError): website.green_main(api, REV)
            self.assertEqual(api.writes, [])

    def test_wrong_event_branch_and_workflow_do_not_qualify(self):
        for field, wrong in [('event', 'pull_request'), ('head_branch', 'other'), ('path', '.github/workflows/website.yml')]:
            api = API(); api.runs[0][field] = wrong
            with self.assertRaises(ValueError): website.green_main(api, REV)

    def test_overall_success_requires_both_actual_required_jobs(self):
        class Missing(API):
            def call(self, method, endpoint, value=None):
                result = super().call(method, endpoint, value)
                if '/jobs?' in endpoint: result['jobs'][0]['conclusion'] = 'skipped'
                return result
        with self.assertRaisesRegex(ValueError, 'required'): website.green_main(Missing(), REV)

    def test_historical_schema_bytes_cannot_change_or_disappear(self):
        data = b'original'; destination = {'entries': {'schemas/v1.1.0/master.schema.json': website.pages.git_blob_sha(data)}}
        website.immutable_schemas(destination, {'schemas/v1.1.0/master.schema.json': data})
        for files in ({}, {'schemas/v1.1.0/master.schema.json': b'changed'}):
            with self.assertRaises(ValueError): website.immutable_schemas(destination, files)

    def test_only_exact_github_package_redirects_to_approved_cdn(self):
        valid = 'https://github.com/shruggietech/insonic/releases/download/v1.0.2/insonic.msi'
        website.package_url(valid, initial=True)
        website.package_url('https://release-assets.githubusercontent.com/path?signature=secret')
        for url in ['http://github.com/path', 'https://evil.test/path', 'https://user:pass@github.com/path', 'https://github.com/another/repo/releases/download/x/a', 'https://release-assets.githubusercontent.com/path?secret=x#fragment']:
            with self.assertRaises(ValueError): website.package_url(url, initial=True)

    def test_metadata_reader_accepts_full_manifest_and_retains_bounded_reads(self):
        url = 'https://github.com/shruggietech/insonic/releases/download/v1.0.2/release-manifest.json'
        payload = b' ' * 7481000
        response = Mock(status=200)
        response.getheader.return_value = str(len(payload))
        response.read.return_value = payload
        connection = Mock()
        connection.getresponse.return_value = response
        with patch.object(website.http.client, 'HTTPSConnection', return_value=connection):
            self.assertEqual(website.package_response(url), payload)
            response.read.assert_called_once_with((16 << 20) + 1)
            response.reset_mock()
            response.getheader.return_value = str((16 << 20) + 1)
            with self.assertRaisesRegex(ValueError, 'exceeds bound'):
                website.package_response(url)
            response.read.assert_not_called()
            response.getheader.return_value = str(len(payload))
            with self.assertRaisesRegex(ValueError, 'exceeds bound'):
                website.package_response(url, limit=1 << 20)
            response.read.assert_not_called()
            self.assertEqual(website.package_response(url, head=True), len(payload))
            response.read.assert_not_called()
            response.getheader.return_value = str(len(payload) + 1)
            with self.assertRaisesRegex(ValueError, 'incomplete'):
                website.package_response(url)
        self.assertEqual(connection.close.call_count, 5)

    def test_unqualified_receipt_never_enters_destination_transport(self):
        with tempfile.TemporaryDirectory() as temporary:
            archive = Path(temporary) / 'documentation.tar.gz'; archive.write_bytes(b'unqualified')
            api = API()
            with patch.object(website, 'push_tree') as transport:
                with self.assertRaises(ValueError): website.publish(archive, {}, Path(temporary) / 'state.json', api, 'test-token')
                transport.assert_not_called()
            self.assertEqual(api.writes, [])

    def test_git_nonforce_response_loss_and_identical_replay(self):
        old, new, tree = 'b' * 40, 'c' * 40, 'd' * 40
        destination = {'head': old, 'tree': tree}
        class Remote(API):
            head = old
            def call(self, method, endpoint, value=None):
                if endpoint.endswith('/git/ref/heads/gh-pages'): return {'object': {'sha': self.head}}
                return super().call(method, endpoint, value)
        for identical, lost in [(False, False), (False, True), (True, False)]:
            api = Remote(); calls = []; reads = 0
            def runner(args, cwd, env):
                nonlocal reads
                calls.append(args)
                self.assertNotIn('test-secret', ' '.join(args))
                if args[0] == 'clone': Path(args[-1]).mkdir()
                if args[0] == 'rev-parse':
                    reads += 1; return old if reads == 1 else new
                if args[0] == 'write-tree': return tree if identical else 'e' * 40
                if args[0] == 'push':
                    api.head = new
                    if lost: raise OSError('lost response')
                return ''
            self.assertEqual(website.push_tree(api, destination, {'index.html': b'new'}, REV, 'test-secret', runner), old if identical else new)
            self.assertFalse(any('--force' in args or any(x.startswith('--force') for x in args) for args in calls))
            self.assertEqual(sum(args[0] == 'push' for args in calls), 0 if identical else 1)

    def test_destination_race_rejects_without_push(self):
        api = API(); calls = []
        def runner(args, cwd, env):
            calls.append(args)
            if args[0] == 'clone': Path(args[-1]).mkdir()
            if args[0] == 'rev-parse': return 'f' * 40
            return ''
        with self.assertRaisesRegex(ValueError, 'changed'):
            website.push_tree(api, {'head': 'b'*40}, {}, REV, 'token', runner)
        self.assertFalse(any(args[0] == 'push' for args in calls))

    def product_fixture(self):
        names = {f'insonic_1.1.0_{platform}_{variant}{suffix}' for platform, suffix in [('windows_amd64', '.zip'), ('linux_amd64', '.tar.gz')] for variant in ('cli', 'desktop')}
        names |= {'insonic_1.1.0_windows_amd64_installer.msi', 'insonic_1.1.0_windows_amd64_setup.exe', 'documentation.tar.gz', 'offline-help.tar.gz'}
        data = {name: name.encode() for name in names}
        entries = [{'name': name, 'size_bytes': len(data[name]), 'sha256': hashlib.sha256(data[name]).hexdigest()} for name in sorted(names)]
        manifest = {'kind': 'release-candidate', 'repository': website.REPOSITORY, 'version': '1.1.0', 'schema_version': '1.1.0', 'tag': 'v1.1.0', 'revision': REV, 'assets': entries}
        data['release-manifest.json'] = json.dumps(manifest).encode()
        checks = entries + [{'name': 'release-manifest.json', 'sha256': hashlib.sha256(data['release-manifest.json']).hexdigest()}]
        data['SHA256SUMS'] = ''.join(e['sha256'] + '  ' + e['name'] + '\n' for e in sorted(checks, key=lambda e: e['name'])).encode()
        assets = [{'id': i + 1, 'name': name, 'size': len(raw), 'digest': 'sha256:' + hashlib.sha256(raw).hexdigest(), 'state': 'uploaded',
                   'browser_download_url': f'https://github.com/{website.REPOSITORY}/releases/download/v{manifest["version"]}/{name}'} for i, (name, raw) in enumerate(data.items())]
        class Product(API):
            def call(self, method, endpoint, value=None):
                if method != 'GET': raise AssertionError('product mutation')
                if '/releases/tags/' in endpoint: return {'id': 10, 'target_commitish': REV, 'draft': False, 'prerelease': False, 'tag_name': 'v1.1.0'}
                if '/git/ref/tags/' in endpoint: return {'object': {'type': 'commit', 'sha': REV}}
                if '/assets?' in endpoint: return assets
                raise AssertionError(endpoint)
        return Product(), assets, data

    def test_product_metadata_preserves_exact_six_packages_without_mutation(self):
        api, assets, data = self.product_fixture()
        product = website.published_product(api, '1.1.0', lambda url: data[url.rsplit('/', 1)[1]])
        self.assertEqual(len(product['packages']), 6)
        self.assertEqual(product['release_revision'], REV)
        self.assertEqual(api.writes, [])

    def test_product_asset_or_metadata_drift_rejected(self):
        for failure in ('missing', 'digest', 'bytes', 'foreign-url'):
            api, assets, data = self.product_fixture()
            if failure == 'missing': assets.pop()
            if failure == 'digest': assets[0]['digest'] = 'sha256:' + 'f' * 64
            if failure == 'bytes': data['release-manifest.json'] = b'changed'
            if failure == 'foreign-url': assets[0]['browser_download_url'] = 'https://evil.test/file'
            with self.assertRaises(ValueError): website.published_product(api, '1.1.0', lambda url: data[url.rsplit('/', 1)[1]])
            self.assertEqual(api.writes, [])

    def test_real_git_publishes_complete_binary_tree_then_replays_without_commit(self):
        with tempfile.TemporaryDirectory(prefix='website transport spaces ') as temporary:
            base = Path(temporary); remote = base / 'remote.git'; seed = base / 'seed'; seed.mkdir()
            env = dict(os.environ, GIT_AUTHOR_NAME='fixture', GIT_AUTHOR_EMAIL='fixture@example.test', GIT_COMMITTER_NAME='fixture', GIT_COMMITTER_EMAIL='fixture@example.test')
            website.git(['init', '--bare', '--initial-branch=gh-pages', str(remote)], base, env)
            website.git(['init', '--initial-branch=gh-pages'], seed, env)
            (seed / 'old.txt').write_bytes(b'obsolete')
            website.git(['add', '--all'], seed, env); website.git(['commit', '-m', 'seed'], seed, env)
            website.git(['push', str(remote), 'HEAD:refs/heads/gh-pages'], seed, env)
            def oid(name): return website.git(['rev-parse', name], remote, env)
            class Remote(API):
                def call(self, method, endpoint, value=None):
                    if endpoint.endswith('/git/ref/heads/gh-pages'): return {'object': {'sha': oid('gh-pages')}}
                    return super().call(method, endpoint, value)
            def runner(args, cwd, selected):
                args = [str(remote) if arg == 'https://github.com/' + website.REPOSITORY + '.git' else arg for arg in args]
                return website.git(args, cwd, {**selected, **{k:v for k,v in env.items() if k.startswith('GIT_AUTHOR') or k.startswith('GIT_COMMITTER')}})
            selected = {'head': oid('gh-pages'), 'tree': oid('gh-pages^{tree}')}
            files = {'index.html': b'fresh\n', 'asset.bin': b'\x00\xff\x80binary\n'}
            first = website.push_tree(Remote(), selected, files, REV, 'fixture-token', runner)
            self.assertNotEqual(first, selected['head'])
            self.assertEqual(website.git(['ls-tree', '--name-only', 'gh-pages'], remote, env).splitlines(), ['asset.bin', 'index.html'])
            self.assertEqual(website.git(['rev-parse', 'gh-pages:asset.bin'], remote, env), website.pages.git_blob_sha(files['asset.bin']))
            second = website.push_tree(Remote(), {'head': first, 'tree': oid('gh-pages^{tree}')}, files, REV, 'fixture-token', runner)
            self.assertEqual(second, first)


if __name__ == '__main__': unittest.main()
