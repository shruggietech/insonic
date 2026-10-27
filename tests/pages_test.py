# SPDX-License-Identifier: Apache-2.0
import base64
import copy
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import subprocess
import os

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
        self.trees, self.blobs, self.timeouts = {}, {}, []

    def call(self, method, endpoint, value=None, *, timeout=None):
        self.calls.append((method, endpoint, copy.deepcopy(value)))
        self.timeouts.append(timeout)
        if endpoint.endswith('/pages') and method == 'GET': return copy.deepcopy(self.destination)
        if endpoint.endswith('/environments/github-pages'): return {'deployment_branch_policy': {'protected_branches': False, 'custom_branch_policies': True}}
        if endpoint.endswith('/deployment-branch-policies'): return copy.deepcopy(self.policy)
        if endpoint.endswith('/git/ref/heads/gh-pages'): return {'object': {'type': 'commit', 'sha': self.head}}
        if endpoint.endswith('/git/commits/' + self.head): return {'tree': {'sha': self.tree}}
        if '/git/trees/' in endpoint:
            key = endpoint.rsplit('/', 1)[-1].split('?')[0]
            if key not in self.trees: return {'truncated': False, 'tree': copy.deepcopy(self.entries)}
            files = self.trees[key]
            parents = {p.as_posix() for n in files for p in Path(n).parents if p.as_posix() != '.'}
            return {'truncated': False, 'tree': [dict(path=n, mode='100644', type='blob', sha=s, size=len(self.blobs[s])) for n, s in files.items()] +
                    [dict(path=n, mode='040000', type='tree', sha='4' * 40) for n in sorted(parents)]}
        if endpoint.endswith('/git/blobs/' + 'e' * 40): return {'encoding': 'base64', 'content': base64.b64encode(b'insonic.io\n').decode()}
        if endpoint.endswith('/git/blobs') and method == 'POST':
            data = base64.b64decode(value['content'], validate=True)
            digest = hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()
            self.blobs[digest] = data
            return {'sha': digest}
        if endpoint.endswith('/git/trees') and method == 'POST':
            if len(json.dumps(value).encode()) > 1 << 20:
                raise ValueError('fixture rejects oversized tree request')
            staged = dict(self.trees.get(value.get('base_tree'), {}))
            for item in value['tree']:
                if 'content' in item:
                    data = item['content'].encode('utf-8')
                    digest = hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()
                    self.blobs[digest] = data
                else: digest = item['sha']
                staged[item['path']] = digest
            key = hashlib.sha1(json.dumps(staged, sort_keys=True).encode()).hexdigest()
            self.trees[key] = staged
            return {'sha': key}
        if endpoint.endswith('/git/commits') and method == 'POST': return {'sha': NEW}
        if endpoint.endswith('/git/refs/heads/gh-pages') and method == 'PATCH':
            if self.race:
                self.head = '9' * 40
                raise ValueError('concurrent ref update')
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
        self.files = {'index.html': b'landing', 'docs/v0.0.0/index.html': b'baseline', 'docs/v1.0.2/index.html': b'released',
                      'release-documentation.json': json.dumps({'version': '1.0.2', 'revision': REVISION, 'kind': 'documentation'}).encode(),
                      'documentation-build.json': json.dumps({'version': '1.0.2', 'revision': REVISION, 'source_dirty': False, 'documentation_target': 'release'}).encode()}

    def tearDown(self): self.temp.cleanup()

    def archive_files(self, files=None, extra=None):
        with tarfile.open(self.archive, 'w:gz') as archive:
            for name, raw in (files or self.files).items():
                member = tarfile.TarInfo(name); member.size = len(raw); archive.addfile(member, io.BytesIO(raw))
            if extra is not None: archive.addfile(extra, io.BytesIO(b'x' * extra.size))

    def prepared(self):
        self.archive_files()
        destination = pages.preflight(self.api, REPO)
        return destination, pages.prepare_tree(self.archive, destination, '1.0.2', REVISION)

    def promote(self, destination, files, **options):
        elapsed = [0]
        if 'clock' not in options:
            options['clock'] = lambda: elapsed[0]
            observed_sleep = options.get('sleep', lambda _: None)
            def advance(seconds):
                observed_sleep(seconds)
                elapsed[0] += seconds
            options['sleep'] = advance
        return pages.promote_tree(self.api, REPO, destination, files, '1.0.2', REVISION, **options)

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
        value = self.promote(destination, files, sleep=slept.append)
        self.assertEqual(value['pages_commit'], NEW)
        self.assertEqual([seconds for seconds in slept if seconds == 5], [5, 5])
        self.assertTrue(all(seconds in (1, 5) for seconds in slept))

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
                    pages.prepare_tree(self.archive, destination, '1.0.2', REVISION)
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
                    pages.prepare_tree(self.archive, destination, '1.0.2', REVISION)

    def test_historical_aliases_reject_case_collisions_and_bounded_payload_amplification(self):
        old = 'docs/v0.0.0/__next.docs/$d$version/__PAGE__.txt'
        self.api.entries.append({'path': old, 'mode': '100644', 'type': 'blob', 'sha': '6' * 40})
        self.files['docs/v0.0.0/__next.docs.$d$version.__PAGE__.txt'] = b'verified segment'
        self.files[old.upper()] = b'ambiguous alias'
        self.archive_files()
        with self.assertRaisesRegex(ValueError, 'ambiguous'):
            pages.prepare_tree(self.archive, pages.preflight(self.api, REPO), '1.0.2', REVISION)
        self.api = PagesFixture()
        self.files = {name: data for name, data in self.files.items() if '__next.' not in name.lower()}
        self.files['docs/v0.0.0/__next.docs.a.b.c.__PAGE__.txt'] = b'x' * (13 << 20)
        for suffix in ['__next.docs/a/b/c', '__next.docs/a.b/c', '__next.docs/a/b.c',
                       '__next.docs.a/b/c', '__next.docs.a.b/c']:
            self.api.entries.append({'path': 'docs/v0.0.0/' + suffix + '/__PAGE__.txt',
                                     'mode': '100644', 'type': 'blob', 'sha': '6' * 40})
        self.archive_files()
        with self.assertRaisesRegex(ValueError, 'bounded'):
            pages.prepare_tree(self.archive, pages.preflight(self.api, REPO), '1.0.2', REVISION)

    def test_archive_alias_git_link_and_traversal_fail_before_writes(self):
        destination = pages.preflight(self.api, REPO)
        for name in ('../escape', 'x/../escape', '.git/config', 'assets/.GIT/hooks/x', '/absolute', 'a//b', 'A:/outside', 'a\\b', 'INDEX.html'):
            self.archive_files({**self.files, name: b'x'})
            with self.assertRaises(ValueError): pages.prepare_tree(self.archive, destination, '1.0.2', REVISION)
        link = tarfile.TarInfo('alias'); link.type = tarfile.SYMTYPE; link.linkname = 'index.html'
        self.archive_files(extra=link)
        with self.assertRaises(ValueError): pages.prepare_tree(self.archive, destination, '1.0.2', REVISION)
        self.assertTrue(all(c[0] == 'GET' for c in self.api.calls))

    def test_conflicting_domain_lost_history_or_candidate_target_is_rejected(self):
        destination = pages.preflight(self.api, REPO)
        cases = [{**self.files, 'CNAME': b'other.example\n'}, {k:v for k,v in self.files.items() if not k.startswith('docs/v0')},
                 {**self.files, 'documentation-build.json': json.dumps({'version': '1.0.2', 'revision': REVISION, 'source_dirty': False, 'documentation_target': 'snapshot'}).encode()}]
        for files in cases:
            self.archive_files(files)
            with self.assertRaises(ValueError): pages.prepare_tree(self.archive, destination, '1.0.2', REVISION)

    def test_promotion_advances_without_force_and_requires_exact_build_commit(self):
        destination, files = self.prepared()
        result = self.promote(destination, files, sleep=lambda _: None)
        self.assertEqual(result['pages_commit'], NEW)
        mutation = next(c for c in self.api.calls if c[0] == 'PATCH')
        self.assertEqual(mutation[2], {'sha': NEW, 'force': False})
        commit = next(c for c in self.api.calls if c[1].endswith('/git/commits') and c[0] == 'POST')
        self.assertEqual(commit[2]['parents'], [OLD])
        self.assertTrue(any(c[0] == 'POST' and c[1].endswith('/pages/builds') for c in self.api.calls))

    def test_concurrent_update_does_not_request_a_build(self):
        destination, files = self.prepared(); self.api.race = True
        with self.assertRaisesRegex(ValueError, 'concurrent'): self.promote(destination, files)
        self.assertFalse(any(c[0] == 'POST' and c[1].endswith('/pages/builds') for c in self.api.calls))

    def test_uncertain_successful_update_is_reconciled_by_exact_remote_head(self):
        destination, files = self.prepared(); self.api.uncertain = True
        result = self.promote(destination, files, sleep=lambda _: None)
        self.assertEqual(result['pages_commit'], NEW)

    def test_wrong_commit_or_failed_build_never_reports_success(self):
        for result in ({'status': 'built', 'commit': '9' * 40}, {'status': 'errored', 'commit': NEW}):
            destination, files = self.prepared(); self.api.results = [result]
            with self.assertRaises(ValueError): self.promote(destination, files, sleep=lambda _: None)

    def test_queued_build_has_a_bounded_deadline(self):
        destination, files = self.prepared(); self.api.results = [{'status': 'queued', 'commit': NEW}]
        elapsed = [0]
        with self.assertRaisesRegex(ValueError, 'deadline'):
            self.promote(destination, files,
                               sleep=lambda seconds: elapsed.__setitem__(0, pages.timing.ARTIFACT_PRODUCER_SECONDS + 1 if seconds == 5 else elapsed[0] + seconds), clock=lambda: elapsed[0])

    def test_representative_export_is_batched_and_complete(self):
        destination, files = self.prepared()
        files.update({f'docs/v1.0.2/page-{i:04d}.html': (str(i) + '\n').encode() + b'x' * 49000 for i in range(1103)})
        self.assertEqual(len(files), 1110)
        self.assertGreater(sum(map(len, files.values())), 54_000_000)
        self.promote(destination, files)
        writes = [v for m, e, v in self.api.calls if m == 'POST' and e.endswith('/git/trees')]
        self.assertGreater(len(writes), 1)
        self.assertTrue(all(len(json.dumps(v).encode()) <= 1 << 20 for v in writes))
        self.assertNotIn('base_tree', writes[0])
        commit = next(v for m, e, v in self.api.calls if m == 'POST' and e.endswith('/git/commits'))
        staged = self.api.trees[commit['tree']]
        self.assertEqual({n: self.api.blobs[s] for n, s in staged.items()}, files)

    def test_large_unicode_binary_and_repeated_content_preserve_original_bytes(self):
        destination, files = self.prepared()
        files.update({'large.txt': b'z' * (2 << 20), 'same.txt': b'z' * (2 << 20),
                      'unicode.txt': ('\u2603\U0001f642\n' * 100000).encode(), 'binary.bin': bytes(range(256)), 'empty': b''})
        self.promote(destination, files)
        uploads = [v for m, e, v in self.api.calls if m == 'POST' and e.endswith('/git/blobs')]
        self.assertGreaterEqual(len(uploads), 2)
        self.assertEqual(sum(base64.b64decode(v['content']) == files['large.txt'] for v in uploads), 1)
        commit = next(v for m, e, v in self.api.calls if m == 'POST' and e.endswith('/git/commits'))
        self.assertEqual({n: self.api.blobs[s] for n, s in self.api.trees[commit['tree']].items()}, files)

    def test_verified_inventory_rejects_corruption_before_commit_or_ref(self):
        for change in ('missing', 'extra', 'duplicate', 'mode', 'sha', 'size', 'truncated'):
            with self.subTest(change=change):
                self.api = PagesFixture(); destination, files = self.prepared()
                original = self.api.call
                def corrupt(method, endpoint, value=None, **kwargs):
                    result = original(method, endpoint, value, **kwargs)
                    if '/git/trees/' in endpoint:
                        if change == 'missing': result['tree'].pop()
                        elif change == 'extra': result['tree'].append(dict(result['tree'][0], path='extra'))
                        elif change == 'duplicate': result['tree'].append(result['tree'][0])
                        elif change == 'truncated': result['truncated'] = True
                        else: result['tree'][0][change] = {'mode': '100755', 'sha': '0' * 40, 'size': 999}[change]
                    return result
                self.api.call = corrupt
                with self.assertRaisesRegex(ValueError, 'inventory|tree'):
                    self.promote(destination, files)
                self.assertFalse(any(m == 'PATCH' or e.endswith('/git/commits') for m, e, _ in self.api.calls))

    def test_failed_intermediate_tree_and_wrong_blob_do_not_advance_branch(self):
        for failure in ('tree', 'blob'):
            with self.subTest(failure=failure):
                self.api = PagesFixture(); destination, files = self.prepared()
                files.update({f'large-{i}.txt': bytes([65+i]) * 600000 for i in range(3)})
                if failure == 'blob': files['binary'] = b'\xff'
                original = self.api.call; count = [0]
                def fail(method, endpoint, value=None, **kwargs):
                    result = original(method, endpoint, value, **kwargs)
                    if method == 'POST' and endpoint.endswith('/git/trees'):
                        count[0] += 1
                        if failure == 'tree' and count[0] == 2: raise ValueError('injected staging failure')
                    if failure == 'blob' and endpoint.endswith('/git/blobs'): result['sha'] = '0' * 40
                    return result
                self.api.call = fail
                with self.assertRaises(ValueError): self.promote(destination, files)
                self.assertEqual(self.api.head, OLD)
                self.assertFalse(any(m == 'PATCH' or e.endswith('/pages/builds') for m, e, _ in self.api.calls))

    def test_all_requests_share_decreasing_remaining_deadline(self):
        destination, files = self.prepared(); elapsed = [0]
        original = self.api.call
        def timed(method, endpoint, value=None, **kwargs):
            elapsed[0] += 7
            return original(method, endpoint, value, **kwargs)
        self.api.call = timed; self.api.timeouts.clear()
        with patch.object(pages.timing, 'SUBPROCESS_SECONDS', pages.timing.ARTIFACT_PRODUCER_SECONDS):
            self.promote(destination, files, clock=lambda: elapsed[0])
        self.assertTrue(all(t is not None and 0 < t <= pages.timing.ARTIFACT_PRODUCER_SECONDS for t in self.api.timeouts))
        self.assertTrue(all(a > b for a, b in zip(self.api.timeouts, self.api.timeouts[1:])))

    def test_staging_deadline_exhaustion_prevents_later_calls(self):
        destination, files = self.prepared(); elapsed = [0]; original = self.api.call
        def expire(method, endpoint, value=None, **kwargs):
            result = original(method, endpoint, value, **kwargs)
            elapsed[0] = pages.timing.ARTIFACT_PRODUCER_SECONDS + 1
            return result
        self.api.call = expire
        with self.assertRaisesRegex(ValueError, 'deadline'):
            self.promote(destination, files, clock=lambda: elapsed[0])
        self.assertEqual(self.api.head, OLD)
        self.assertFalse(any(m == 'PATCH' for m, _, _ in self.api.calls))

    def test_staged_inventory_matches_independent_real_git_objects(self):
        destination, files = self.prepared()
        files.update({'assets/snowman-\u2603.txt': b'original\n', 'assets/raw.bin': bytes(range(256)), 'empty': b''})
        destination['entries']['obsolete.html'] = '9' * 40
        self.promote(destination, files)
        commit = next(v for m, e, v in self.api.calls if m == 'POST' and e.endswith('/git/commits'))
        staged = self.api.trees[commit['tree']]
        environment = {k: v for k, v in os.environ.items() if k.upper() in {'PATH', 'SYSTEMROOT', 'WINDIR', 'TEMP', 'TMP', 'TMPDIR'}}
        environment.update(GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=os.devnull, GIT_TERMINAL_PROMPT='0')
        options = dict(capture_output=True, check=True, timeout=pages.timing.SUBPROCESS_SECONDS,
                       env=environment, creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)
        repository = self.root / 'oracle.git'
        subprocess.run(['git', 'init', '--bare', str(repository)], **options)
        stream = io.BytesIO()
        for index, (name, raw) in enumerate(files.items(), 1):
            stream.write(f'blob\nmark :{index}\ndata {len(raw)}\n'.encode() + raw + b'\n')
        stream.write(b'commit refs/heads/main\ncommitter Fixture <fixture@example.invalid> 0 +0000\ndata 0\n\n')
        for index, name in enumerate(files, 1):
            stream.write(f'M 100644 :{index} {json.dumps(name, ensure_ascii=False)}\n'.encode())
        stream.write(b'\ndone\n')
        subprocess.run(['git', '--git-dir', str(repository), 'fast-import', '--quiet', '--done'], input=stream.getvalue(), **options)
        result = subprocess.run(['git', '--git-dir', str(repository), 'ls-tree', '-rlz', 'refs/heads/main'], **options).stdout
        actual = {}
        for item in result.split(b'\0'):
            if not item: continue
            metadata, name = item.split(b'\t', 1)
            mode, kind, digest, size = metadata.split()
            self.assertEqual((mode, kind), (b'100644', b'blob'))
            self.assertEqual(int(size), len(files[name.decode()]))
            actual[name.decode()] = digest.decode()
        self.assertEqual(actual, staged)
        self.assertNotIn('obsolete.html', actual)

    def test_unchanged_tree_reuses_blobs_without_a_commit_or_ref_update(self):
        destination, files = self.prepared()
        staged = {n: pages.git_blob_sha(raw) for n, raw in files.items()}
        destination['entries'] = staged
        destination['tree'] = hashlib.sha1(json.dumps(staged, sort_keys=True).encode()).hexdigest()
        self.api.blobs.update({staged[n]: raw for n, raw in files.items()})
        self.api.results = [{'status': 'built', 'commit': OLD}]
        result = self.promote(destination, files)
        self.assertEqual(result['pages_commit'], OLD)
        self.assertFalse(any(m == 'PATCH' or e.endswith(('/git/commits', '/git/blobs')) for m, e, _ in self.api.calls))
        tree = next(v for m, e, v in self.api.calls if m == 'POST' and e.endswith('/git/trees'))
        self.assertTrue(all('sha' in entry and 'content' not in entry for entry in tree['tree']))

    def test_mutation_pacing_uses_the_shared_clock_and_remaining_allowance(self):
        destination, files = self.prepared(); elapsed = [0]; writes = []
        original = self.api.call
        def record(method, endpoint, value=None, **kwargs):
            if method in ('POST', 'PATCH'): writes.append(elapsed[0])
            return original(method, endpoint, value, **kwargs)
        self.api.call = record
        self.promote(destination, files, clock=lambda: elapsed[0], sleep=lambda seconds: elapsed.__setitem__(0, elapsed[0] + seconds))
        self.assertTrue(all(b - a >= 1 for a, b in zip(writes, writes[1:])))
        self.assertEqual(self.api.timeouts[-1], pages.timing.SUBPROCESS_SECONDS if pages.timing.SUBPROCESS_SECONDS < pages.timing.ARTIFACT_PRODUCER_SECONDS - elapsed[0] else pages.timing.ARTIFACT_PRODUCER_SECONDS - elapsed[0])

    def test_expired_pacing_wait_cannot_start_the_next_mutation(self):
        destination, files = self.prepared(); elapsed = [0]
        with self.assertRaisesRegex(ValueError, 'deadline'):
            self.promote(destination, files, clock=lambda: elapsed[0],
                         sleep=lambda _: elapsed.__setitem__(0, pages.timing.ARTIFACT_PRODUCER_SECONDS + 1))
        self.assertEqual(self.api.head, OLD)
        self.assertFalse(any(m == 'PATCH' or e.endswith('/git/commits') for m, e, _ in self.api.calls))


if __name__ == '__main__': unittest.main()
