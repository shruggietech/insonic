# SPDX-License-Identifier: Apache-2.0
import copy
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('release_tool', ROOT / 'scripts/release.py')
release = importlib.util.module_from_spec(spec); spec.loader.exec_module(release)
REVISION = 'a' * 40
CURRENT = (ROOT / 'VERSION').read_text(encoding='utf-8').strip()
VERSION = f'{int(CURRENT.split(".")[0]) + 1}.0.0'
NEXT_VERSION = f'{int(CURRENT.split(".")[0]) + 2}.0.0'


def runner(args, root=None):
    if args[:3] == ['git', 'rev-parse', 'HEAD'] or args[:2] == ['git', 'rev-parse']: return REVISION
    if args[:2] == ['git', 'status']: return ''
    if args[:3] == ['git', 'merge-base', '--is-ancestor']: return ''
    raise AssertionError(args)

def verify_fixture(path, receipt):
    if path.read_bytes() != (receipt['platform'] + receipt['variant']).encode(): raise ValueError('controlled package inventory mismatch')


def fixture(root):
    for name in ('VERSION', 'CHANGELOG.md', 'CONTRIBUTING.md', 'docs/versions.json', 'internal/contracts/contracts.go', '.github/bootstrap.json'):
        destination = root / name; destination.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(ROOT / name, destination)
    for prefix in ('', 'site', 'desktop/frontend'):
        for name in ('package.json', 'package-lock.json'):
            destination = root / prefix / name; destination.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(ROOT / prefix / name, destination)
    shutil.copytree(ROOT / f'schemas/v{CURRENT}', root / f'schemas/v{CURRENT}')
    shutil.copytree(ROOT / f'docs/v{CURRENT}', root / f'docs/v{CURRENT}')
    release.prepare(VERSION, '2026-10-09', 'Portable packages and current-authority recovery.', root)


def candidates(root):
    assets = root / 'assets'; assets.mkdir(); receipts = []
    for platform in release.PLATFORMS:
        for variant in release.VARIANTS:
            archive = assets / f'insonic_{VERSION}_{platform}_{variant}.zip'; archive.write_bytes((platform + variant).encode())
            entry = release.file_entry(archive)
            receipt = {'kind': 'native-package-receipt', 'schema_version': VERSION, 'product_version': VERSION, 'documentation_version': VERSION, 'platform': platform, 'variant': variant, 'revision': REVISION, 'source_dirty': False, 'archive': archive.name, 'archive_sha256': entry['sha256'], 'archive_size_bytes': entry['size_bytes'], 'distribution': {'binary_publication': 'eligible', 'source_complete': True, 'corresponding_source': 'included'}, 'signing': {'configured': False, 'verified': False, 'status': 'unconfigured'}}
            receipt['inventory_sha256'] = 'c' * 64
            receipt.update(extracted_inventory='passed', relocation_path_spaces='passed', cli_real_audio_video_import='passed', development_native_libraries='isolated-and-restored', gui_bridge='passed' if variant == 'desktop' else 'not-included', native_webview='passed' if variant == 'desktop' else 'not-included')
            path = root / f'{platform}-{variant}-receipt.json'; release.write_json(path, receipt); receipts.append(path)
    for name in ('documentation', 'offline-help'):
        with tarfile.open(assets / (name + '.tar.gz'), 'w:gz') as archive:
            for filename, value in [('release-documentation.json', {'version': VERSION, 'revision': REVISION, 'kind': name}), ('documentation-build.json', {'version': VERSION, 'revision': REVISION, 'source_dirty': False})]:
                raw = json.dumps(value).encode(); info = tarfile.TarInfo(filename); info.size = len(raw); archive.addfile(info, io.BytesIO(raw))
    return assets, receipts


class GitHubFixture:
    def __init__(self, fail_upload=False):
        self.release = None; self.assets = []; self.calls = []; self.fail_upload = fail_upload
        self.asset_bytes = {}; self.downloads = []
        self.checks = [{'name': name, 'status': 'completed', 'conclusion': 'success', 'check_suite': {'id': 11}} for name in ('foundation', 'docs')]
        self.workflow = {'id': 77, 'repository': {'full_name': release.REPOSITORY},
                         'path': '.github/workflows/release.yml@refs/heads/main', 'event': 'workflow_dispatch',
                         'head_sha': REVISION, 'check_suite_id': 42}

    def call(self, method, endpoint, value=None, path=None):
        self.calls.append((method, endpoint, copy.deepcopy(value)))
        if endpoint == f'repos/{release.REPOSITORY}': return {'default_branch': 'main'}
        if endpoint.endswith('/branches/main'): return {'name': 'main', 'protected': True, 'commit': {'sha': REVISION}}
        if endpoint.endswith('/environments/documentation-deployment'): return {'deployment_branch_policy': {'protected_branches': False, 'custom_branch_policies': True}}
        if endpoint.endswith('/deployment-branch-policies'): return {'total_count': 1, 'branch_policies': [{'name': 'main', 'type': 'branch'}]}
        if '/git/ref/' in endpoint: return {'object': {'type': 'commit', 'sha': REVISION}}
        if '/check-runs?' in endpoint: return {'check_runs': copy.deepcopy(self.checks)}
        if '/actions/runs/' in endpoint: return copy.deepcopy(self.workflow)
        if '/releases/tags/' in endpoint: return copy.deepcopy(self.release)
        if endpoint.endswith('/releases') and method == 'POST':
            self.release = {**value, 'id': 7, 'upload_url': 'https://uploads.github.com/repos/shruggietech/insonic/releases/7/assets{?name,label}'}; return copy.deepcopy(self.release)
        if '/assets?per_page=' in endpoint: return copy.deepcopy(self.assets)
        if path is not None:
            if self.fail_upload and self.assets: raise ValueError('controlled interruption')
            entry = release.file_entry(path); asset_id = len(self.assets) + 100
            self.asset_bytes[asset_id] = path.read_bytes()
            value = {'id': asset_id, 'name': entry['name'], 'size': entry['size_bytes'], 'digest': 'sha256:' + entry['sha256'], 'state': 'uploaded'}; self.assets.append(value); return value
        if endpoint.endswith('/releases/7'):
            if method == 'PATCH': self.release.update(value)
            return copy.deepcopy(self.release)
        raise AssertionError((method, endpoint))

    def download_asset(self, asset_id, path, size, digest):
        self.downloads.append(asset_id)
        path.write_bytes(self.asset_bytes[asset_id])


class ReleaseDeliveryTests(unittest.TestCase):
    def setUp(self):
        workflow_environment = patch.dict(os.environ, {'GITHUB_RUN_ID': '', 'GITHUB_REPOSITORY': '', 'GITHUB_SHA': ''})
        workflow_environment.start(); self.addCleanup(workflow_environment.stop)
        self.temporary = tempfile.TemporaryDirectory(); self.root = Path(self.temporary.name); fixture(self.root)
        self.assets, self.receipts = candidates(self.root)

    def tearDown(self): self.temporary.cleanup()

    def candidate(self, output='candidate', receipts=None):
        target = self.root / output
        release.candidate(receipts or self.receipts, self.assets, target, REVISION, self.root, verify_fixture, runner)
        return target

    def test_preparation_preserves_old_bytes_and_freezes_release_authorities(self):
        self.assertEqual(release.versions(self.root), VERSION)
        self.assertEqual((self.root / f'schemas/v{CURRENT}/master.schema.json').read_bytes(), (ROOT / f'schemas/v{CURRENT}/master.schema.json').read_bytes())
        self.assertEqual((self.root / f'docs/v{CURRENT}/index.md').read_bytes(), (ROOT / f'docs/v{CURRENT}/index.md').read_bytes())
        self.assertEqual((self.root / f'docs/v{VERSION}/references/CHANGELOG.md').read_bytes(), (self.root / 'CHANGELOG.md').read_bytes())
        self.assertTrue((self.root / f'docs/v{VERSION}/references/CONTRIBUTING.md').is_file())
        self.assertTrue(release.read_json(self.root / 'build/release/version-preparation.json')['glossary_scan']['technical_terms'])
        self.assertTrue((self.root / 'build/release/highlights.md').read_text().rstrip().endswith(f'/blob/v{VERSION}/CHANGELOG.md)'))
        with self.assertRaisesRegex(ValueError, 'newer'): release.prepare(VERSION, '2026-10-09', 'Highlights.', self.root)

    def test_prepared_actual_schema_tree_and_all_examples_validate_at_new_version(self):
        node = shutil.which('node')
        if not node and Path('C:/nvm4w/nodejs/node.exe').is_file(): node = 'C:/nvm4w/nodejs/node.exe'
        self.assertIsNotNone(node, 'The configured Node runtime is required for release schema verification.')
        code = f"import {{loadSchemaCatalog}} from {json.dumps((ROOT / 'scripts/schema-catalog.mjs').as_uri())}; import {{validateCatalog}} from {json.dumps((ROOT / 'scripts/check-schemas.mjs').as_uri())}; try {{ const catalog=loadSchemaCatalog(process.argv[2],process.argv[1]); const result=validateCatalog(catalog); if(!result.master(catalog.contracts.find(c=>c.schema.properties.kind.const==='runtime-request').schema.examples[0])) throw Error('runtime envelope'); console.log(result.examples); }} catch(error) {{ console.log('ERROR: '+error.message); }}"
        output = release.child([node, '--input-type=module', '-e', code, str(self.root), VERSION], ROOT)
        self.assertRegex(output, r'^\d+$'); self.assertGreater(int(output), 20)
        candidate = release.read_json(self.root / f'schemas/v{VERSION}/release-candidate.schema.json')
        self.assertEqual(candidate['properties']['version']['const'], VERSION)
        self.assertEqual(candidate['properties']['tag']['const'], 'v' + VERSION)
        self.assertEqual(candidate['properties']['packages']['items']['properties']['product_version']['const'], VERSION)

    def test_generated_complete_unsigned_candidate_passes_prepared_release_master(self):
        target = self.candidate(); node = shutil.which('node') or 'C:/nvm4w/nodejs/node.exe'
        code = f"import fs from 'node:fs'; import {{loadSchemaCatalog}} from {json.dumps((ROOT / 'scripts/schema-catalog.mjs').as_uri())}; import {{validateCatalog}} from {json.dumps((ROOT / 'scripts/check-schemas.mjs').as_uri())}; const {{master}}=validateCatalog(loadSchemaCatalog(process.argv[2],process.argv[1])); const value=JSON.parse(fs.readFileSync(process.argv[3],'utf8')); console.log(JSON.stringify({{valid:master(value),errors:master.errors}}));"
        result = json.loads(release.child([node, '--input-type=module', '-e', code, str(self.root), VERSION, str(target / 'release-manifest.json')], ROOT))
        self.assertTrue(result['valid'], result['errors'])

    def test_preparation_rolls_back_bad_major_documentation(self):
        (self.root / f'docs/v{VERSION}/glossary.md').write_text('# Glossary\n', encoding='utf-8')
        old = (self.root / 'VERSION').read_bytes()
        with self.assertRaisesRegex(ValueError, 'glossary'): release.prepare(NEXT_VERSION, '2026-10-09', 'Highlights.', self.root)
        self.assertEqual((self.root / 'VERSION').read_bytes(), old)
        self.assertFalse((self.root / f'docs/v{NEXT_VERSION}').exists()); self.assertFalse((self.root / f'schemas/v{NEXT_VERSION}').exists())

    def test_exact_source_refuses_dirty_or_wrong_tag(self):
        def dirty(args, root): return ' M product.go' if args[1] == 'status' else REVISION
        with self.assertRaisesRegex(ValueError, 'clean'): release.exact_source(REVISION, self.root, dirty)
        def other_tag(args, root): return 'b' * 40 if args[-1].endswith('^{commit}') else runner(args, root)
        with self.assertRaisesRegex(ValueError, 'tag'): release.exact_source(REVISION, self.root, other_tag, 'v' + VERSION)

    def test_missing_variant_dirty_sources_signing_and_hashes_fail_before_output(self):
        with self.assertRaisesRegex(ValueError, 'all six'): self.candidate(receipts=self.receipts[:-1])
        receipt = release.read_json(self.receipts[0]); original = copy.deepcopy(receipt)
        for field, value, reason in [('source_dirty', True, 'source'), ('product_version', '9.0.0', 'versions'), ('extracted_inventory', 'not-run', 'qualification'), ('distribution', {'source_complete': False}, 'sources'), ('signing', {'configured': True, 'verified': False}, 'signing'), ('signing', {'configured': False, 'verified': True, 'status': 'signed'}, 'signing'), ('archive_sha256', 'f' * 64, 'archive')]:
            receipt = copy.deepcopy(original); receipt[field] = value; release.write_json(self.receipts[0], receipt)
            with self.assertRaisesRegex(ValueError, reason): self.candidate()
            self.assertFalse((self.root / 'candidate').exists())
        release.write_json(self.receipts[0], original)
        self.candidate()
        with self.assertRaisesRegex(ValueError, 'new'): self.candidate()

    def test_source_validator_checks_desktop_lock_and_runtime(self):
        lock = self.root / 'desktop/frontend/package-lock.json'; value = release.read_json(lock); value['packages']['']['version'] = '999.999.999'; release.write_json(lock, value)
        with self.assertRaisesRegex(ValueError, 'versions'): release.versions(self.root)

    def test_privileged_publication_uses_trusted_main_history_and_historical_version_data(self):
        target = self.candidate(); api = GitHubFixture(); original_call = api.call
        trusted = self.root / 'trusted-tooling'; trusted.mkdir()
        (trusted / 'VERSION').write_text('999.0.0\n')  # Tooling version is independent of historical release data.
        main_sha = 'b' * 40; calls = []
        def transport(method, endpoint, value=None, path=None):
            if endpoint.endswith('/branches/main'): return {'name': 'main', 'protected': True, 'commit': {'sha': main_sha}}
            return original_call(method, endpoint, value, path)
        api.call = transport
        def historical_runner(args, root):
            calls.append((args, root))
            if args == ['git', 'rev-parse', 'HEAD'] and root == trusted: return main_sha
            if args[:3] == ['git', 'merge-base', '--is-ancestor']:
                self.assertEqual(args[3:], [REVISION, main_sha]); self.assertEqual(root, trusted); return ''
            return runner(args, root)
        result = release.publish(target, 'Highlights.', api, self.root, historical_runner, verify_fixture, trusted)
        self.assertTrue(result['published']); self.assertEqual(api.release['tag_name'], 'v' + VERSION)
        self.assertTrue(any(args[:2] == ['git', 'merge-base'] for args, _ in calls))
        api.calls.clear()
        def outside(args, root):
            if args[:3] == ['git', 'merge-base', '--is-ancestor']: raise ValueError('not ancestor')
            return historical_runner(args, root)
        with self.assertRaisesRegex(ValueError, 'outside trusted main history'):
            release.publish(target, 'Highlights.', api, self.root, outside, lambda *args: self.fail('verifier ran for untrusted source'), trusted)
        self.assertFalse(any(call[0] in ('POST', 'PATCH') for call in api.calls))

    def test_missing_branch_protection_or_wrong_tool_checkout_blocks_privileged_tools(self):
        api = GitHubFixture(); original = api.call
        for protected in (False, None):
            def transport(method, endpoint, value=None, path=None):
                if endpoint.endswith('/branches/main'): return {'name': 'main', 'protected': protected, 'commit': {'sha': REVISION}}
                return original(method, endpoint, value, path)
            api.call = transport
            with self.assertRaisesRegex(ValueError, 'protected default main'):
                release.trusted_history(REVISION, api, self.root, runner)
        api.call = original
        with self.assertRaisesRegex(ValueError, 'exact full current commit'):
            release.trusted_history(REVISION, api, self.root, lambda args, root: 'f' * 40)

    def test_deployment_environment_requires_only_main_branch_policy(self):
        api = GitHubFixture(); original = api.call
        self.assertEqual(release.deployment_environment(api), {'deployment_environment': 'verified'})
        for branches in ([], [{'name': '*', 'type': 'branch'}], [{'name': 'main', 'type': 'tag'}],
                         [{'name': 'main', 'type': 'branch'}, {'name': 'feature', 'type': 'branch'}]):
            def transport(method, endpoint, value=None, path=None):
                if endpoint.endswith('/deployment-branch-policies'): return {'total_count': len(branches), 'branch_policies': branches}
                return original(method, endpoint, value, path)
            api.call = transport
            with self.assertRaisesRegex(ValueError, 'main only'): release.deployment_environment(api)

    def test_automation_credentials_are_rejected_case_insensitively_and_never_forwarded(self):
        executable = str(Path(sys.executable).resolve())
        for name in release.AUTOMATION_CREDENTIAL_NAMES | {'GH_CUSTOM_CREDENTIAL', 'GITHUB_CUSTOM_TOKEN', 'ACTIONS_CUSTOM_TOKEN', 'RUNNER_SECRET'}:
            for spelling in (name, name.lower(), name.swapcase()):
                with self.assertRaisesRegex(ValueError, 'reserved automation'):
                    release.deployment_configuration({'argv': [executable, '{archive}'], 'environment_names': [spelling]}, {spelling: 'automation credential'})
        target = self.candidate(); api = GitHubFixture()
        release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        invoked = []
        def deployment_child(args, environment=None): invoked.append((args, environment))
        secret_name = 'INSONIC_DOCUMENTATION_DEPLOYMENT_CREDENTIAL'
        with patch.dict(os.environ, {'GH_TOKEN': 'write-token', 'GITHUB_TOKEN': 'automation-token', secret_name: 'dedicated deployment credential'}), patch.object(release, 'child', deployment_child):
            release.promote(target, {'argv': [executable, '{archive}'], 'environment_names': [secret_name]}, api,
                            deployment_child, verify_fixture, self.root, runner)
        self.assertEqual(invoked[0][1][secret_name], 'dedicated deployment credential')
        self.assertNotIn('GH_TOKEN', invoked[0][1]); self.assertNotIn('GITHUB_TOKEN', invoked[0][1])

    def test_stale_or_forged_documentation_marker_fails(self):
        for directory in ('site/out', 'site/offline'):
            path = self.root / directory; path.mkdir(parents=True); (path / 'index.html').write_text('<html></html>')
            release.write_json(path / 'documentation-build.json', {'version': VERSION, 'revision': 'b' * 40, 'source_dirty': False})
        with self.assertRaisesRegex(ValueError, 'stale'): release.document_archives(self.root / 'doc-assets', REVISION, self.root, runner)
        with tarfile.open(self.assets / 'documentation.tar.gz', 'w:gz') as archive:
            for filename, value in [('release-documentation.json', {'version': VERSION, 'revision': REVISION, 'kind': 'documentation'}), ('documentation-build.json', {'version': VERSION, 'revision': 'b' * 40, 'source_dirty': False})]:
                raw = json.dumps(value).encode(); info = tarfile.TarInfo(filename); info.size = len(raw); archive.addfile(info, io.BytesIO(raw))
        with self.assertRaisesRegex(ValueError, 'build source'): self.candidate()

    def test_publication_uploads_complete_draft_then_publishes_and_reads_back(self):
        target = self.candidate(); api = GitHubFixture()
        outcome = release.publish(target, 'Portable recovery and complete package sources.', api, self.root, runner, verify_fixture)
        self.assertTrue(outcome['published']); self.assertFalse(api.release['draft']); self.assertEqual(len(api.assets), 10)
        self.assertTrue(api.release['body'].endswith(f'/blob/v{VERSION}/CHANGELOG.md)'))
        publishing = next(i for i, call in enumerate(api.calls) if call[0] == 'PATCH')
        self.assertEqual(sum(1 for call in api.calls[:publishing] if call[0] == 'POST' and 'uploads.github.com' in call[1]), 10)
        old_calls = len(api.calls)
        repeated = release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        self.assertTrue(repeated['reconciled'])
        self.assertFalse(any(call[0] in ('POST', 'PATCH') for call in api.calls[old_calls:]))

    def test_forged_candidate_with_recomputed_hashes_revalidates_native_inventory(self):
        target = self.candidate(); value = release.read_json(target / 'release-manifest.json')
        receipt = value['packages'][0]; path = target / receipt['archive']; path.write_bytes(b'forged native bytes')
        entry = release.file_entry(path); receipt['archive_sha256'] = entry['sha256']; receipt['archive_size_bytes'] = entry['size_bytes']
        value['assets'][0] = entry; release.write_json(target / 'release-manifest.json', value)
        entries = [*value['assets'], release.file_entry(target / 'release-manifest.json')]
        (target / 'SHA256SUMS').write_text(''.join(f"{e['sha256']}  {e['name']}\n" for e in sorted(entries, key=lambda e: e['name'])), encoding='utf-8')
        with self.assertRaisesRegex(ValueError, 'inventory'): release.load_candidate(target, verify_fixture)

    def test_github_transport_rejects_unselected_hosts_ports_and_userinfo_before_io(self):
        api = release.GitHub('synthetic-not-a-real-token')
        for endpoint in ['https://evil.example/upload', 'http://api.github.com/path', 'https://api.github.com:444/path', 'https://:secret@api.github.com/path']:
            with self.assertRaisesRegex(ValueError, 'endpoint'): api.call('POST', endpoint, {})

    def test_interrupted_draft_does_not_publish_or_promote_and_resume_is_idempotent(self):
        target = self.candidate(); api = GitHubFixture(fail_upload=True)
        with self.assertRaisesRegex(ValueError, 'interruption'): release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        self.assertTrue(api.release['draft']); self.assertFalse(any(call[0] == 'PATCH' for call in api.calls))
        invoked = []
        with self.assertRaisesRegex(ValueError, 'published'): release.promote(target, {'argv': ['/usr/bin/deploy', '{archive}']}, api, lambda args: invoked.append(args), verify_fixture, self.root, runner)
        self.assertEqual(invoked, [])
        api.fail_upload = False; release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        self.assertEqual(len(api.assets), 10)
        release.promote(target, {'argv': [str(Path(sys.executable).resolve()), '{archive}', '{version}', '{revision}']}, api, lambda args: invoked.append(args), verify_fixture, self.root, runner)
        self.assertEqual(invoked[0][2:], [VERSION, REVISION])

    def test_deployment_configuration_is_materialized_before_publication_without_invocation(self):
        path = self.root / 'build/release/documentation-deployment.json'
        config = {'argv': [str(Path(sys.executable).resolve()), '{archive}', '{version}', '{revision}'],
                  'environment_names': ['DEPLOY_TOKEN']}
        environment = {'INSONIC_DOCUMENTATION_DEPLOYMENT_JSON': json.dumps(config), 'DEPLOY_TOKEN': 'fixture credential'}
        self.assertEqual(release.materialize_deployment(path, environment), {'deployment_configuration': 'validated'})
        self.assertEqual(release.read_json(path), config)
        self.assertNotIn('fixture credential', path.read_text())
        self.assertFalse(path.read_bytes().startswith(b'\xef\xbb\xbf'))

    def test_missing_or_invalid_deployment_configuration_cannot_reach_publication(self):
        path = self.root / 'build/release/documentation-deployment.json'
        executable = str(Path(sys.executable).resolve())
        invalid = ['', 'not json', '[]', json.dumps({'argv': ['relative-command', '{archive}']}),
                   json.dumps({'argv': [str(self.root / 'absent'), '{archive}']}),
                   json.dumps({'argv': [executable]}), json.dumps({'argv': [executable, '{archive}'], 'host': 'invented'}),
                   json.dumps({'argv': [executable, '{archive}'], 'environment_names': ['MISSING_TOKEN']}),
                   json.dumps({'argv': [executable, '{archive}'], 'environment_names': ['BAD-NAME']})]
        for raw in invalid:
            api = GitHubFixture()
            with self.assertRaises(ValueError):
                release.materialize_deployment(path, {'INSONIC_DOCUMENTATION_DEPLOYMENT_JSON': raw})
                release.publish(self.candidate(), 'Highlights.', api, self.root, runner, verify_fixture)
            self.assertEqual(api.calls, [])
            self.assertFalse(path.exists())

    def test_current_publisher_suite_is_excluded_but_external_red_or_pending_checks_still_block(self):
        target = self.candidate()
        context = {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}
        with patch.dict(os.environ, context):
            api = GitHubFixture()
            api.checks.append({'name': 'publish', 'status': 'in_progress', 'conclusion': None, 'check_suite': {'id': 42}})
            self.assertTrue(release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)['published'])
            for status, conclusion in [('in_progress', None), ('completed', 'failure')]:
                blocked = GitHubFixture()
                blocked.checks.extend([{'name': 'publish', 'status': 'in_progress', 'conclusion': None, 'check_suite': {'id': 42}},
                                       {'name': 'security review', 'status': status, 'conclusion': conclusion, 'check_suite': {'id': 23}}])
                with self.assertRaisesRegex(ValueError, 'not green'):
                    release.publish(target, 'Highlights.', blocked, self.root, runner, verify_fixture)
                self.assertFalse(any(call[0] in ['POST', 'PATCH'] for call in blocked.calls))
                required = GitHubFixture()
                required.checks[0].update(status=status, conclusion=conclusion)
                with self.assertRaisesRegex(ValueError, 'not green'):
                    release.publish(target, 'Highlights.', required, self.root, runner, verify_fixture)

    def test_unverified_or_irrelevant_suites_cannot_hide_checks(self):
        context = {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}
        api = GitHubFixture()
        for changes in [{'path': '.github/workflows/ci.yml'}, {'id': 78}, {'check_suite_id': None},
                        {'repository': {'full_name': 'someone/else'}}, {'head_sha': 'b' * 40}]:
            fixture_api = GitHubFixture(); fixture_api.workflow.update(changes)
            with self.assertRaises(ValueError):
                release.current_release_suite(fixture_api, REVISION, context)
        self.assertEqual(release.current_release_suite(api, 'b' * 40, context), None)
        target = self.candidate()
        api.checks.append({'name': 'publish', 'status': 'in_progress', 'conclusion': None, 'check_suite': {'id': 999}})
        with patch.dict(os.environ, context), self.assertRaisesRegex(ValueError, 'not green'):
            release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        historical = GitHubFixture()
        historical.workflow['head_sha'] = 'b' * 40
        with patch.dict(os.environ, dict(context, GITHUB_SHA='b' * 40)):
            self.assertTrue(release.publish(target, 'Highlights.', historical, self.root, runner, verify_fixture)['published'])

    def test_promotion_only_retrieves_original_candidate_without_rebuilding_or_publishing(self):
        original = self.candidate(); api = GitHubFixture()
        release.publish(original, 'Highlights.', api, self.root, runner, verify_fixture)
        api.calls.clear()
        retrieved = self.root / 'retrieved'
        result = release.published_candidate(retrieved, REVISION, api, self.root, runner, verify_fixture)
        self.assertEqual(result, {'candidate': 'retrieved-and-verified', 'version': VERSION, 'revision': REVISION})
        self.assertEqual(len(api.downloads), 10)
        for path in original.iterdir():
            self.assertEqual((retrieved / path.name).read_bytes(), path.read_bytes())
        invoked = []
        release.promote(retrieved, {'argv': [str(Path(sys.executable).resolve()), '{archive}']}, api,
                        lambda args: invoked.append(args), verify_fixture, self.root, runner)
        self.assertEqual(invoked[0][1], str((retrieved / 'documentation.tar.gz').resolve()))
        self.assertTrue(all(call[0] == 'GET' for call in api.calls))

    def test_retrieved_candidate_rejects_changed_bytes_and_wrong_release_identity_without_completion(self):
        original = self.candidate(); api = GitHubFixture()
        release.publish(original, 'Highlights.', api, self.root, runner, verify_fixture)
        saved = copy.deepcopy(api.release)
        for changes in [{'draft': True}, {'target_commitish': 'b' * 40}, {'tag_name': 'v999.0.0'}]:
            api.release = {**saved, **changes}
            with self.assertRaisesRegex(ValueError, 'exact existing'):
                release.published_candidate(self.root / 'retrieved', REVISION, api, self.root, runner, verify_fixture)
            self.assertFalse((self.root / 'retrieved').exists())
        self.assertEqual(api.downloads, [])
        api.release = saved
        changed = next(asset for asset in api.assets if asset['name'] == 'documentation.tar.gz')
        api.asset_bytes[changed['id']] = b'rebuilt bytes from the same source'
        with self.assertRaisesRegex(ValueError, 'bytes changed'):
            release.published_candidate(self.root / 'retrieved', REVISION, api, self.root, runner, verify_fixture)
        self.assertFalse((self.root / 'retrieved').exists())

    def test_asset_download_strips_auth_on_official_cdn_redirect_and_rejects_other_destinations(self):
        data = b'controlled immutable bytes'; digest = release.hashlib.sha256(data).hexdigest()
        class DownloadOpener:
            def __init__(self, destination): self.destination = destination; self.requests = []
            def open(self, request, timeout):
                self.requests.append(request)
                if len(self.requests) == 1:
                    raise release.urllib.error.HTTPError(request.full_url, 302, 'redirect', {'Location': self.destination}, io.BytesIO())
                return io.BytesIO(data)
        api = release.GitHub('fixture-token')
        api.opener = DownloadOpener('https://release-assets.githubusercontent.com/asset?signed=fixture')
        downloaded = self.root / 'downloaded'
        api.download_asset(100, downloaded, len(data), digest)
        self.assertEqual(downloaded.read_bytes(), data)
        self.assertEqual(api.opener.requests[0].get_header('Authorization'), 'Bearer fixture-token')
        self.assertIsNone(api.opener.requests[1].get_header('Authorization'))
        self.assertNotIn('fixture-token', api.opener.requests[1].full_url)
        api.opener = DownloadOpener('https://release-assets.githubusercontent.com/asset')
        with self.assertRaisesRegex(ValueError, 'download failed'):
            api.download_asset(100, downloaded, len(data), digest)
        self.assertEqual(downloaded.read_bytes(), data)
        for destination in ['https://untrusted.example/asset', 'http://release-assets.githubusercontent.com/asset',
                            'https://user@release-assets.githubusercontent.com/asset', 'https://release-assets.githubusercontent.com:444/asset']:
            api.opener = DownloadOpener(destination)
            with self.assertRaisesRegex(ValueError, 'redirect destination'):
                api.download_asset(100, self.root / 'blocked', len(data), digest)
            self.assertEqual(len(api.opener.requests), 1)
            self.assertFalse((self.root / 'blocked').exists())

    def test_remote_tag_digest_or_candidate_tampering_prevents_promotion(self):
        target = self.candidate(); api = GitHubFixture(); release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        api.assets[0]['digest'] = 'sha256:' + 'f' * 64
        with self.assertRaisesRegex(ValueError, 'digest'): release.promote(target, {'argv': [str(self.root / 'deploy')]}, api, lambda args: self.fail('deployment ran'), verify_fixture, self.root, runner)
        (target / 'offline-help.tar.gz').write_bytes(b'tampered')
        with self.assertRaisesRegex(ValueError, 'changed'): release.load_candidate(target, verify_fixture)


if __name__ == '__main__': unittest.main()
