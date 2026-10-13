# SPDX-License-Identifier: Apache-2.0
import base64
import copy
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('pages_transport', ROOT / 'scripts/pages.py')
pages = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pages)
REPO = 'shruggietech/insonic'
REVISION = 'a' * 40
OLD = 'b' * 40
NEW = 'c' * 40


class PagesFixture:
    def __init__(self):
        self.calls = []
        self.destination = {'build_type': 'legacy', 'source': {'branch': 'gh-pages', 'path': '/'}, 'cname': 'insonic.io', 'https_enforced': True}
        self.policy = {'total_count': 1, 'branch_policies': [{'name': 'gh-pages', 'type': 'branch'}]}
        self.head, self.tree = OLD, 'd' * 40
        self.entries = [{'path': 'CNAME', 'mode': '100644', 'type': 'blob', 'sha': 'e' * 40},
                        {'path': '.nojekyll', 'mode': '100644', 'type': 'blob', 'sha': 'f' * 40},
                        {'path': 'docs/v0.0.0/index.html', 'mode': '100644', 'type': 'blob', 'sha': '1' * 40}]
        self.results = [{'status': 'built', 'commit': NEW}]
        self.race = False
        self.uncertain = False

    def call(self, method, endpoint, value=None):
        self.calls.append((method, endpoint, copy.deepcopy(value)))
        if endpoint.endswith('/pages') and method == 'GET': return copy.deepcopy(self.destination)
        if endpoint.endswith('/environments/github-pages'): return {'deployment_branch_policy': {'protected_branches': False, 'custom_branch_policies': True}}
        if endpoint.endswith('/deployment-branch-policies'): return copy.deepcopy(self.policy)
        if endpoint.endswith('/git/ref/heads/gh-pages'): return {'object': {'type': 'commit', 'sha': self.head}}
        if endpoint.endswith('/git/commits/' + self.head): return {'tree': {'sha': self.tree}}
        if '/git/trees/' in endpoint: return {'truncated': False, 'tree': copy.deepcopy(self.entries)}
        if endpoint.endswith('/git/blobs/' + 'e' * 40): return {'encoding': 'base64', 'content': base64.b64encode(b'insonic.io\n').decode()}
        if endpoint.endswith('/git/blobs') and method == 'POST': return {'sha': '2' * 40}
        if endpoint.endswith('/git/trees') and method == 'POST': return {'sha': '3' * 40}
        if endpoint.endswith('/git/commits') and method == 'POST': return {'sha': NEW}
        if endpoint.endswith('/git/refs/heads/gh-pages') and method == 'PATCH':
            if self.race: raise ValueError('concurrent ref update')
            self.head = value['sha']
            if self.uncertain: raise OSError('uncertain response')
            return {'object': {'sha': self.head}}
        if endpoint.endswith('/pages/builds') and method == 'POST': return {'status': 'queued'}
        if endpoint.endswith('/pages/builds/latest'): return copy.deepcopy(self.results.pop(0) if len(self.results) > 1 else self.results[0])
        raise AssertionError((method, endpoint))


class PagesTransportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.archive = self.root / 'documentation.tar.gz'
        self.api = PagesFixture()
        self.files = {'index.html': b'landing', 'docs/v0.0.0/index.html': b'baseline', 'docs/v1.0.1/index.html': b'released',
                      'release-documentation.json': json.dumps({'version': '1.0.1', 'revision': REVISION, 'kind': 'documentation'}).encode(),
                      'documentation-build.json': json.dumps({'version': '1.0.1', 'revision': REVISION, 'source_dirty': False, 'documentation_target': 'release'}).encode()}

    def tearDown(self): self.temp.cleanup()

    def archive_files(self, files=None, extra=None):
        with tarfile.open(self.archive, 'w:gz') as archive:
            for name, raw in (files or self.files).items():
                member = tarfile.TarInfo(name); member.size = len(raw); archive.addfile(member, io.BytesIO(raw))
            if extra is not None: archive.addfile(extra, io.BytesIO(b'x' * extra.size))

    def prepared(self):
        self.archive_files()
        destination = pages.preflight(self.api, REPO)
        return destination, pages.prepare_tree(self.archive, destination, '1.0.1', REVISION)

    def test_read_only_preflight_preserves_existing_legacy_destination(self):
        self.api.entries.append({'path': 'docs', 'mode': '040000', 'type': 'tree', 'sha': '4' * 40})
        value = pages.preflight(self.api, REPO)
        self.assertEqual(value['head'], OLD)
        self.assertTrue(all(call[0] == 'GET' for call in self.api.calls))
        self.assertEqual(value['domain'], 'insonic.io')
        self.assertNotIn('docs', value['entries'])

    def test_previous_completed_build_is_not_confused_with_new_queued_build(self):
        destination, files = self.prepared()
        self.api.results = [{'status': 'built', 'commit': OLD}, {'status': 'queued', 'commit': NEW}, {'status': 'built', 'commit': NEW}]
        slept = []
        value = pages.promote_tree(self.api, REPO, destination, files, '1.0.1', REVISION, sleep=slept.append)
        self.assertEqual(value['pages_commit'], NEW)
        self.assertEqual(slept, [5, 5])

    def test_unexpected_source_domain_policy_or_truncated_tree_is_rejected(self):
        for change in ({'cname': 'other.example'}, {'build_type': 'workflow'}, {'source': {'branch': 'main', 'path': '/' }}, {'https_enforced': False}):
            api = PagesFixture(); api.destination.update(change)
            with self.assertRaisesRegex(ValueError, 'destination'): pages.preflight(api, REPO)
            self.assertTrue(all(c[0] == 'GET' for c in api.calls))
        self.api.policy['branch_policies'][0]['name'] = 'main'
        with self.assertRaisesRegex(ValueError, 'policy'): pages.preflight(self.api, REPO)
        class Truncated(PagesFixture):
            def call(self, method, endpoint, value=None):
                result = super().call(method, endpoint, value)
                if '/git/trees/' in endpoint: result['truncated'] = True
                return result
        with self.assertRaisesRegex(ValueError, 'tree'): pages.preflight(Truncated(), REPO)

    def test_missing_domain_marker_is_rejected(self):
        self.api.entries = [e for e in self.api.entries if e['path'] != 'CNAME']
        with self.assertRaisesRegex(ValueError, 'marker'): pages.preflight(self.api, REPO)

    def test_preparation_is_read_only_and_preserves_domain_and_historical_routes(self):
        destination, files = self.prepared()
        self.assertEqual(files['CNAME'], b'insonic.io\n')
        self.assertEqual(files['.nojekyll'], b'')
        self.assertIn('docs/v0.0.0/index.html', files)
        self.assertTrue(all(c[0] == 'GET' for c in self.api.calls))

    def test_historical_nested_next_segments_alias_only_verified_flat_replacements(self):
        old = 'docs/v0.0.0/__next.docs/$d$version/__PAGE__.txt'
        replacement = 'docs/v0.0.0/__next.docs.$d$version.__PAGE__.txt'
        self.api.entries.append({'path': old, 'mode': '100644', 'type': 'blob', 'sha': '6' * 40})
        self.files[replacement] = b'verified replacement segment'
        destination, files = self.prepared()
        self.assertEqual(files[old], self.files[replacement])
        self.assertEqual(files[replacement], self.files[replacement])
        self.assertTrue(all(c[0] == 'GET' for c in self.api.calls))
        self.assertTrue(set(destination['entries']).issubset(files))

    def test_history_compatibility_does_not_ignore_missing_pages_schemas_or_segments(self):
        for name in ['docs/v0.0.0/removed/index.html', 'schemas/v0.0.0/removed.json',
                     'docs/v0.0.0/__next.docs/$d$unknown/__PAGE__.txt',
                     'docs/v0.0.0/__next.docs.$d$unknown.__PAGE__.txt']:
            with self.subTest(name=name):
                api = PagesFixture()
                api.entries.append({'path': name, 'mode': '100644', 'type': 'blob', 'sha': '6' * 40})
                destination = pages.preflight(api, REPO)
                self.archive_files()
                with self.assertRaisesRegex(ValueError, 'historical'):
                    pages.prepare_tree(self.archive, destination, '1.0.1', REVISION)
                self.assertTrue(all(c[0] == 'GET' for c in api.calls))

    def test_historical_flat_next_segments_require_exact_nested_page_replacements(self):
        for directory, suffix in [('docs/v0.0.0', '$d$version'),
                                  ('docs/v1.0.0/architecture', '$d$version.$d$slug'),
                                  ('docs/v1.0.0/references/contributing-md', '$d$version.references.$d$reference')]:
            with self.subTest(directory=directory):
                old = directory + '/__next.docs.' + suffix + '.__PAGE__.txt'
                replacement = directory + '/__next.docs/' + suffix.replace('.', '/') + '/__PAGE__.txt'
                self.api = PagesFixture()
                self.api.entries.append({'path': old, 'mode': '100644', 'type': 'blob', 'sha': '6' * 40})
                self.files[directory + '/index.html'] = b'checked page'
                self.files[replacement] = b'verified nested segment'
                destination, files = self.prepared()
                self.assertEqual(files[old], self.files[replacement])
                self.assertTrue(set(destination['entries']).issubset(files))
                self.assertTrue(all(c[0] == 'GET' for c in self.api.calls))
                without_page = {name: data for name, data in self.files.items() if name != directory + '/index.html'}
                self.archive_files(without_page)
                with self.assertRaisesRegex(ValueError, 'historical'):
                    pages.prepare_tree(self.archive, destination, '1.0.1', REVISION)

    def test_historical_aliases_reject_case_collisions_and_bounded_payload_amplification(self):
        old = 'docs/v0.0.0/__next.docs/$d$version/__PAGE__.txt'
        self.api.entries.append({'path': old, 'mode': '100644', 'type': 'blob', 'sha': '6' * 40})
        self.files['docs/v0.0.0/__next.docs.$d$version.__PAGE__.txt'] = b'verified segment'
        self.files[old.upper()] = b'ambiguous alias'
        self.archive_files()
        with self.assertRaisesRegex(ValueError, 'ambiguous'):
            pages.prepare_tree(self.archive, pages.preflight(self.api, REPO), '1.0.1', REVISION)
        self.api = PagesFixture()
        self.files = {name: data for name, data in self.files.items() if '__next.' not in name.lower()}
        self.files['docs/v0.0.0/__next.docs.a.b.c.__PAGE__.txt'] = b'x' * (13 << 20)
        for suffix in ['__next.docs/a/b/c', '__next.docs/a.b/c', '__next.docs/a/b.c',
                       '__next.docs.a/b/c', '__next.docs.a.b/c']:
            self.api.entries.append({'path': 'docs/v0.0.0/' + suffix + '/__PAGE__.txt',
                                     'mode': '100644', 'type': 'blob', 'sha': '6' * 40})
        self.archive_files()
        with self.assertRaisesRegex(ValueError, 'bounded'):
            pages.prepare_tree(self.archive, pages.preflight(self.api, REPO), '1.0.1', REVISION)

    def test_archive_alias_git_link_and_traversal_fail_before_writes(self):
        destination = pages.preflight(self.api, REPO)
        for name in ('../escape', 'x/../escape', '.git/config', 'assets/.GIT/hooks/x', '/absolute', 'a//b', 'A:/outside', 'a\\b', 'INDEX.html'):
            self.archive_files({**self.files, name: b'x'})
            with self.assertRaises(ValueError): pages.prepare_tree(self.archive, destination, '1.0.1', REVISION)
        link = tarfile.TarInfo('alias'); link.type = tarfile.SYMTYPE; link.linkname = 'index.html'
        self.archive_files(extra=link)
        with self.assertRaises(ValueError): pages.prepare_tree(self.archive, destination, '1.0.1', REVISION)
        self.assertTrue(all(c[0] == 'GET' for c in self.api.calls))

    def test_conflicting_domain_lost_history_or_candidate_target_is_rejected(self):
        destination = pages.preflight(self.api, REPO)
        cases = [{**self.files, 'CNAME': b'other.example\n'}, {k:v for k,v in self.files.items() if not k.startswith('docs/v0')},
                 {**self.files, 'documentation-build.json': json.dumps({'version': '1.0.1', 'revision': REVISION, 'source_dirty': False, 'documentation_target': 'snapshot'}).encode()}]
        for files in cases:
            self.archive_files(files)
            with self.assertRaises(ValueError): pages.prepare_tree(self.archive, destination, '1.0.1', REVISION)

    def test_promotion_advances_without_force_and_requires_exact_build_commit(self):
        destination, files = self.prepared()
        result = pages.promote_tree(self.api, REPO, destination, files, '1.0.1', REVISION, sleep=lambda _: None)
        self.assertEqual(result['pages_commit'], NEW)
        mutation = next(c for c in self.api.calls if c[0] == 'PATCH')
        self.assertEqual(mutation[2], {'sha': NEW, 'force': False})
        commit = next(c for c in self.api.calls if c[1].endswith('/git/commits') and c[0] == 'POST')
        self.assertEqual(commit[2]['parents'], [OLD])
        self.assertTrue(any(c[0] == 'POST' and c[1].endswith('/pages/builds') for c in self.api.calls))

    def test_concurrent_update_does_not_request_a_build(self):
        destination, files = self.prepared(); self.api.race = True
        with self.assertRaisesRegex(ValueError, 'concurrent'): pages.promote_tree(self.api, REPO, destination, files, '1.0.1', REVISION)
        self.assertFalse(any(c[0] == 'POST' and c[1].endswith('/pages/builds') for c in self.api.calls))

    def test_uncertain_successful_update_is_reconciled_by_exact_remote_head(self):
        destination, files = self.prepared(); self.api.uncertain = True
        result = pages.promote_tree(self.api, REPO, destination, files, '1.0.1', REVISION, sleep=lambda _: None)
        self.assertEqual(result['pages_commit'], NEW)

    def test_wrong_commit_or_failed_build_never_reports_success(self):
        for result in ({'status': 'built', 'commit': '9' * 40}, {'status': 'errored', 'commit': NEW}):
            destination, files = self.prepared(); self.api.results = [result]
            with self.assertRaises(ValueError): pages.promote_tree(self.api, REPO, destination, files, '1.0.1', REVISION, sleep=lambda _: None)

    def test_queued_build_has_a_bounded_deadline(self):
        destination, files = self.prepared(); self.api.results = [{'status': 'queued', 'commit': NEW}]
        times = iter((0, 0, 301))
        with self.assertRaisesRegex(ValueError, 'deadline'):
            pages.promote_tree(self.api, REPO, destination, files, '1.0.1', REVISION, sleep=lambda _: None, clock=lambda: next(times))


if __name__ == '__main__': unittest.main()
