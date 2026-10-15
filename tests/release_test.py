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
MSI_QUALIFICATION = {'client_selections':'passed','license_and_prerequisites':'passed','maintenance':'passed','upgrade_rollback':'passed','downgrade_rejection':'passed','active_work_preserved':'passed','user_data_preserved':'passed','launch_contract':'passed','native_ui_capture':'passed','cancellation_rollback':'passed','user_context':'passed'}
CURRENT = (ROOT / 'VERSION').read_text(encoding='utf-8').strip()
VERSION = f'{int(CURRENT.split(".")[0]) + 1}.0.0'
NEXT_VERSION = f'{int(CURRENT.split(".")[0]) + 2}.0.0'


def runner(args, root=None):
    if args[:3] == ['git', 'rev-parse', 'HEAD'] or args[:2] == ['git', 'rev-parse']: return REVISION
    if args[:2] == ['git', 'status']: return ''
    if args[:3] == ['git', 'merge-base', '--is-ancestor']: return ''
    raise AssertionError(args)

def verify_fixture(path, receipt):
    if receipt.get('kind') == 'windows-installer-receipt':
        if path.read_bytes() != b'controlled-msi' or receipt.get('qualification') != MSI_QUALIFICATION:
            raise ValueError('controlled installer qualification or inventory mismatch')
    elif path.read_bytes() != (receipt['platform'] + receipt['variant']).encode(): raise ValueError('controlled package inventory mismatch')


def fixture(root):
    for name in ('VERSION', 'CHANGELOG.md', 'CONTRIBUTING.md', 'docs/versions.json', 'internal/contracts/contracts.go', '.github/bootstrap.json'):
        destination = root / name; destination.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(ROOT / name, destination)
    for prefix in ('', 'site', 'desktop/frontend'):
        for name in ('package.json', 'package-lock.json'):
            destination = root / prefix / name; destination.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(ROOT / prefix / name, destination)
    shutil.copytree(ROOT / 'schemas', root / 'schemas')
    shutil.copytree(ROOT / f'docs/v{CURRENT}', root / f'docs/v{CURRENT}')
    release.prepare(VERSION, '2026-10-09', 'Portable packages and current-authority recovery.', root)


def candidates(root, platforms=None, version=VERSION):
    assets = root / 'assets'; assets.mkdir(); receipts = []
    for platform in platforms or release.PLATFORMS:
        for variant in release.VARIANTS:
            archive = assets / f'insonic_{version}_{platform}_{variant}.zip'; archive.write_bytes((platform + variant).encode())
            entry = release.file_entry(archive)
            receipt = {'kind': 'native-package-receipt', 'schema_version': version, 'product_version': version, 'documentation_version': version, 'platform': platform, 'variant': variant, 'revision': REVISION, 'source_dirty': False, 'archive': archive.name, 'archive_sha256': entry['sha256'], 'archive_size_bytes': entry['size_bytes'], 'distribution': {'binary_publication': 'eligible', 'source_complete': True, 'corresponding_source': 'included'}, 'signing': {'configured': False, 'verified': False, 'status': 'unconfigured'}}
            receipt['inventory_sha256'] = 'c' * 64
            receipt.update(windows_copy_install='passed', windows_first_use='passed', extracted_inventory='passed', relocation_path_spaces='passed', cli_real_audio_video_import='passed', development_native_libraries='isolated-and-restored', gui_bridge='passed' if variant == 'desktop' else 'not-included', native_webview='passed' if variant == 'desktop' else 'not-included')
            path = root / f'{platform}-{variant}-receipt.json'; release.write_json(path, receipt); receipts.append(path)
    if version != '1.0.0':
        payload = next(release.read_json(path) for path in receipts if release.read_json(path)['platform'] == 'windows_amd64' and release.read_json(path)['variant'] == 'desktop')
        artifact = assets / f'insonic_{version}_windows_amd64_installer.msi'; artifact.write_bytes(b'controlled-msi'); entry = release.file_entry(artifact)
        receipt = {'kind':'windows-installer-receipt','schema_version':version,'product_version':version,'documentation_version':version,
                   'platform':'windows_amd64','variant':'selectable','format':'msi','revision':REVISION,'source_dirty':False,
                   'artifact':artifact.name,'artifact_sha256':entry['sha256'],'artifact_size_bytes':entry['size_bytes'],
                   'payload_archive':payload['archive'],'payload_archive_sha256':payload['archive_sha256'],
                   'payload_receipt':payload,'payload_inventory':{'kind':'native-package-inventory'},
                   'qualification':copy.deepcopy(MSI_QUALIFICATION),'distribution':copy.deepcopy(payload['distribution']),'signing':{'configured':False,'verified':False,'status':'unconfigured'}}
        path = root / 'windows-installer-receipt.json'; release.write_json(path, receipt); receipts.append(path)
    for name in ('documentation', 'offline-help'):
        with tarfile.open(assets / (name + '.tar.gz'), 'w:gz') as archive:
            for filename, value in [('release-documentation.json', {'version': version, 'revision': REVISION, 'kind': name}), ('documentation-build.json', {'version': version, 'revision': REVISION, 'source_dirty': False, 'documentation_target': 'release'})]:
                raw = json.dumps(value).encode(); info = tarfile.TarInfo(filename); info.size = len(raw); archive.addfile(info, io.BytesIO(raw))
    return assets, receipts


class GitHubFixture:
    def __init__(self, fail_upload=False):
        self.release = None; self.assets = []; self.calls = []; self.fail_upload = fail_upload
        self.asset_bytes = {}; self.downloads = []
        self.checks = [{'name': name, 'status': 'completed', 'conclusion': 'success', 'check_suite': {'id': 11}} for name in ('foundation', 'docs')]
        self.workflow = {'id': 77, 'repository': {'full_name': release.REPOSITORY},
                         'path': '.github/workflows/release.yml@main', 'event': 'workflow_dispatch',
                         'head_sha': REVISION, 'check_suite_id': 42, 'head_branch': 'main', 'workflow_id': 9}
        self.past_workflows = {}

    def call(self, method, endpoint, value=None, path=None):
        self.calls.append((method, endpoint, copy.deepcopy(value)))
        if endpoint == f'repos/{release.REPOSITORY}': return {'default_branch': 'main'}
        if endpoint.endswith('/branches/main'): return {'name': 'main', 'protected': True, 'commit': {'sha': REVISION}}
        if endpoint.endswith('/environments/documentation-deployment'): return {'deployment_branch_policy': {'protected_branches': False, 'custom_branch_policies': True}}
        if endpoint.endswith('/deployment-branch-policies'): return {'total_count': 1, 'branch_policies': [{'name': 'main', 'type': 'branch'}]}
        if '/git/ref/' in endpoint: return {'object': {'type': 'commit', 'sha': REVISION}}
        if '/check-runs?' in endpoint:
            page = int(release.urllib.parse.parse_qs(release.urllib.parse.urlsplit(endpoint).query).get('page', ['1'])[0])
            checks = [{**check, 'id': index + 1} for index, check in enumerate(self.checks)]
            return {'total_count': len(checks), 'check_runs': copy.deepcopy(checks[(page - 1) * 100:page * 100])}
        if '/actions/runs/' in endpoint: return copy.deepcopy(self.past_workflows.get(int(endpoint.rsplit('/', 1)[-1]), self.workflow))
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
    def test_notarization_is_historical_validation_only(self):
        signing = {'configured': True, 'verified': True, 'status': 'notarized'}
        with self.assertRaisesRegex(ValueError, 'signing'):
            release.verify_signing(signing)
        release.verify_signing(signing, historical=True)

    def test_complete_historical_six_package_record_still_validates(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary); schema = root / 'schemas/v1.0.0/release-candidate.schema.json'
            schema.parent.mkdir(parents=True); shutil.copyfile(ROOT / 'schemas/v1.0.0/release-candidate.schema.json', schema)
            assets, receipts = candidates(root, release.HISTORICAL_PLATFORMS, '1.0.0')
            target = root / 'historical'; target.mkdir()
            packages = [release.read_json(path) for path in receipts]
            entries = [release.file_entry(assets / p['archive']) for p in packages]
            entries += [release.file_entry(assets / name) for name in ('documentation.tar.gz','offline-help.tar.gz')]
            for entry in entries: shutil.copyfile(assets / entry['name'], target / entry['name'])
            value = {'kind':'release-candidate','schema_version':'1.0.0','repository':release.REPOSITORY,'version':'1.0.0','tag':'v1.0.0','revision':REVISION,'packages':packages,'assets':entries,
                     'capabilities':{'platforms':list(release.HISTORICAL_PLATFORMS),'variants':list(release.VARIANTS),'advanced_cli':['backup'],'inference_qualification':'not-executed'}}
            release.write_json(target / 'release-manifest.json', value)
            entries += [release.file_entry(target / 'release-manifest.json')]
            (target / 'SHA256SUMS').write_text(''.join(f"{e['sha256']}  {e['name']}\n" for e in sorted(entries,key=lambda e:e['name'])))
            loaded, expected = release.load_candidate(target, verify_fixture, root)
            self.assertEqual(len(loaded['packages']),6); self.assertEqual(len(expected),10)

    def test_false_advertised_platform_cannot_reach_release_validation(self):
        target = self.candidate(); value = release.read_json(target / 'release-manifest.json')
        value['capabilities']['platforms'].append('darwin_arm64')
        release.write_json(target / 'release-manifest.json',value)
        with self.assertRaisesRegex(ValueError,'capability matrix'):
            release.load_candidate(target,verify_fixture,self.root)

    def test_historical_matrix_remains_bound_to_immutable_version_schema(self):
        self.assertEqual(set(release.selected_release_platforms(ROOT, "1.0.0")), {"windows_amd64", "linux_amd64", "darwin_arm64"})
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            schema = json.loads((ROOT / "schemas/v1.0.1/release-candidate.schema.json").read_text())
            schema["properties"]["packages"]["items"]["properties"]["platform"]["enum"].append("darwin_arm64")
            path = root / "schemas/v1.0.1/release-candidate.schema.json"
            path.parent.mkdir(parents=True); path.write_text(json.dumps(schema))
            with self.assertRaisesRegex(ValueError, "platform contract"):
                release.selected_release_platforms(root, "1.0.1")

    def test_release_highlights_state_package_choices_and_real_signature_disposition(self):
        target = self.candidate(); api = GitHubFixture()
        release.publish(target, "Package delivery", api, self.root, runner, verify_fixture)
        self.assertIn("Desktop + CLI includes", api.release["body"])
        self.assertIn("CLI-only contains no desktop", api.release["body"])
        self.assertIn("unsigned, no verified publisher signature", api.release["body"])
        self.assertTrue(api.release["body"].endswith(f"[Full changelog](https://github.com/{release.REPOSITORY}/blob/v{VERSION}/CHANGELOG.md)"))

    def test_current_distribution_requires_exactly_four_windows_linux_variants(self):
        self.assertEqual(set(release.PLATFORMS), {"windows_amd64", "linux_amd64"})
        target = self.candidate()
        value = json.loads((target / "release-manifest.json").read_text(encoding="utf-8"))
        self.assertEqual(len(value["packages"]), 4)
        self.assertEqual(len(value["assets"]), 7)


    def test_unpublished_candidate_cannot_reach_any_pages_transport_write(self):
        target = self.candidate()
        api = GitHubFixture()
        with patch.object(release, 'pages_module', side_effect=AssertionError('Pages transport must not be reached')):
            with self.assertRaisesRegex(ValueError, 'published'):
                release.pages_promote(target, api, verify_fixture, self.root, runner, self.root)
        self.assertTrue(all(call[0] == 'GET' for call in api.calls))

    def test_corrupt_published_candidate_cannot_reach_pages_transport(self):
        target = self.candidate()
        api = GitHubFixture()
        release.publish(target, 'Exact candidate.', api, self.root, runner, verify_fixture)
        (target / 'documentation.tar.gz').write_bytes(b'corrupt documentation')
        with patch.object(release, 'pages_module', side_effect=AssertionError('Pages transport must not be reached')):
            with self.assertRaises(ValueError): release.pages_promote(target, api, verify_fixture, self.root, runner, self.root)

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
        self.assertEqual((self.root / 'schemas/v1.0.0/base-model-manifest.schema.json').read_bytes(), (ROOT / 'schemas/v1.0.0/base-model-manifest.schema.json').read_bytes())
        self.assertEqual((self.root / f'docs/v{CURRENT}/index.md').read_bytes(), (ROOT / f'docs/v{CURRENT}/index.md').read_bytes())
        self.assertEqual((self.root / f'docs/v{VERSION}/references/CHANGELOG.md').read_bytes(), (self.root / 'CHANGELOG.md').read_bytes())
        self.assertTrue((self.root / f'docs/v{VERSION}/references/CONTRIBUTING.md').is_file())
        self.assertTrue(release.read_json(self.root / 'build/release/version-preparation.json')['glossary_scan']['technical_terms'])
        self.assertTrue((self.root / 'build/release/highlights.md').read_text().rstrip().endswith(f'/blob/v{VERSION}/CHANGELOG.md)'))
        with self.assertRaisesRegex(ValueError, 'newer'): release.prepare(VERSION, '2026-10-09', 'Highlights.', self.root)

    def test_preparation_is_an_unpublished_candidate_and_advances_owned_doc_examples(self):
        manifest = release.read_json(self.root / 'docs/versions.json')
        entry = next(entry for entry in manifest['versions'] if entry['version'] == VERSION)
        self.assertEqual(entry['status'], 'candidate')
        index = (self.root / f'docs/v{VERSION}/index.md').read_text(encoding='utf-8')
        self.assertIn('prepared release candidate', index)
        self.assertIn('Official product downloads have not been published', index)
        ingestion = (self.root / f'docs/v{VERSION}/ingestion.md').read_text(encoding='utf-8')
        self.assertNotIn('"schema_version": "' + CURRENT + '"', ingestion)
        self.assertIn('"schema_version": "' + VERSION + '"', ingestion)
        self.assertFalse(release.read_json(self.root / 'build/release/version-preparation.json')['published'])

    def test_preparation_does_not_rewrite_generic_baseline_test_fixture(self):
        target = self.root / 'tests/check.test.mjs'
        target.parent.mkdir(parents=True, exist_ok=True)
        text = "const version='" + VERSION + "'; const path='v" + VERSION + "';\n"
        target.write_text(text, encoding='utf-8', newline='\n')
        release.prepare(NEXT_VERSION, '2026-10-10', 'Candidate update.', self.root)
        self.assertEqual(target.read_text(encoding='utf-8'), text)

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

    def test_preparation_advances_owned_asset_examples_and_common_description(self):
        schema = release.read_json(self.root / f'schemas/v{VERSION}/release-candidate.schema.json')
        expected = {f'insonic_{VERSION}_{platform}_{variant}' + ('.tar.gz' if platform == 'linux_amd64' else '.zip')
                    for platform in release.PLATFORMS for variant in release.VARIANTS}
        for example in schema['examples']:
            self.assertEqual({item['archive'] for item in example['packages']}, expected)
            self.assertEqual({item['name'] for item in example['assets'] if item['name'].startswith('insonic_')}, expected | {f'insonic_{VERSION}_windows_amd64_installer.msi'})
        common = release.read_json(self.root / f'schemas/v{VERSION}/common.schema.json')
        self.assertTrue(common['description'].startswith(f'Shared logical types for the v{VERSION} contract.'))
        self.assertEqual((self.root / f'schemas/v{CURRENT}/common.schema.json').read_bytes(),
                         (ROOT / f'schemas/v{CURRENT}/common.schema.json').read_bytes())

    def test_next_preparation_preserves_upstream_versions_equal_to_owned_version(self):
        upstream = self.root / 'tests/fixtures/cueson/upstream.json'
        upstream.parent.mkdir(parents=True, exist_ok=True)
        upstream.write_text(json.dumps({'schema_version': VERSION, 'upstream_version': VERSION}), encoding='utf-8')
        reader = self.root / 'internal/subtitles/admission.go'
        reader.parent.mkdir(parents=True, exist_ok=True)
        reader.write_text('const upstream = "' + VERSION + '"\n', encoding='utf-8')
        brand = self.root / 'tests/brand-sync.test.mjs'
        brand.parent.mkdir(parents=True, exist_ok=True)
        brand.write_text('const brand = "' + VERSION + '";\n', encoding='utf-8')
        page = self.root / f'docs/v{VERSION}/technology.md'
        upstream_link = f'https://github.com/shruggietech/cueson/blob/v{VERSION}/docs/schema.md'
        page.write_text(page.read_text() + '\n[Cueson upstream](' + upstream_link + ')\n', encoding='utf-8')
        release.prepare(NEXT_VERSION, '2026-10-10', 'Candidate update.', self.root)
        self.assertEqual(json.loads(upstream.read_text())['schema_version'], VERSION)
        self.assertIn(VERSION, reader.read_text())
        self.assertIn(VERSION, brand.read_text())
        self.assertIn(upstream_link, (self.root / f'docs/v{NEXT_VERSION}/technology.md').read_text())

    def test_candidate_snapshot_cannot_be_published_or_promoted(self):
        for name in ('documentation', 'offline-help'):
            with tarfile.open(self.assets / (name + '.tar.gz'), 'w:gz') as archive:
                for filename, value in [('release-documentation.json', {'version': VERSION, 'revision': REVISION, 'kind': name}), ('documentation-build.json', {'version': VERSION, 'revision': REVISION, 'source_dirty': False, 'documentation_target': 'snapshot'})]:
                    raw = json.dumps(value).encode(); info = tarfile.TarInfo(filename); info.size = len(raw); archive.addfile(info, io.BytesIO(raw))
        target = self.candidate()
        api = GitHubFixture()
        with self.assertRaisesRegex(ValueError, 'release-target documentation'):
            release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        self.assertFalse(any(call[0] in ('POST', 'PATCH') for call in api.calls))
        with self.assertRaisesRegex(ValueError, 'release-target documentation'):
            release.promote(target, {'argv': [str(Path(sys.executable).resolve()), '{archive}']}, api,
                            lambda args: self.fail('deployment ran'), verify_fixture, self.root, runner, source_root=self.root)

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
        with self.assertRaisesRegex(ValueError, 'all four'): self.candidate(receipts=self.receipts[1:])
        receipt = release.read_json(self.receipts[0]); original = copy.deepcopy(receipt)
        for field, value, reason in [('source_dirty', True, 'source'), ('product_version', '9.0.0', 'versions'), ('extracted_inventory', 'not-run', 'qualification'), ('windows_copy_install', 'not-run', 'qualification'), ('windows_first_use', 'not-run', 'qualification'), ('distribution', {'source_complete': False}, 'sources'), ('signing', {'configured': True, 'verified': False}, 'signing'), ('signing', {'configured': False, 'verified': True, 'status': 'signed'}, 'signing'), ('archive_sha256', 'f' * 64, 'archive')]:
            receipt = copy.deepcopy(original); receipt[field] = value; release.write_json(self.receipts[0], receipt)
            with self.assertRaisesRegex(ValueError, reason): self.candidate()
            self.assertFalse((self.root / 'candidate').exists())
        release.write_json(self.receipts[0], original)
        self.candidate()
        with self.assertRaisesRegex(ValueError, 'new'): self.candidate()

    def test_msi_missing_duplicate_format_source_payload_and_qualification_rejected(self):
        msi_path = self.receipts[-1]; original = release.read_json(msi_path)
        for receipts in (self.receipts[:-1], self.receipts + [msi_path]):
            with self.assertRaisesRegex(ValueError, 'exactly one MSI'): self.candidate(receipts=receipts)
        mutations = [('format','zip','format'),('platform','linux_amd64','format'),('source_dirty',True,'source'),
                     ('artifact','insonic_wrong.exe','format'),('artifact_sha256','f'*64,'artifact'),
                     ('payload_archive_sha256','f'*64,'payload'),('payload_receipt',{},'payload'),
                     ('qualification',{},'qualification'),('qualification',{**MSI_QUALIFICATION,'cancellation_rollback':'not-executed'},'qualification'),('qualification',{**MSI_QUALIFICATION,'user_context':'not-executed'},'qualification'),('qualification',{**MSI_QUALIFICATION,'upgrade_rollback':'not-executed'},'qualification')]
        for field, value, reason in mutations:
            receipt = copy.deepcopy(original); receipt[field] = value; release.write_json(msi_path, receipt)
            with self.assertRaisesRegex(ValueError, reason): self.candidate()
            self.assertFalse((self.root / 'candidate').exists())
        release.write_json(msi_path, original)
        target = self.candidate(); value = release.read_json(target / 'release-manifest.json')
        self.assertEqual(len(value['packages']),4); self.assertEqual(len(value['installers']),1)
        self.assertEqual(len(value['assets']),7)

    def test_msi_published_retrieval_keeps_artifact_and_rejects_missing_inventory(self):
        target = self.candidate(); api = GitHubFixture()
        release.publish(target, 'MSI installation.', api, self.root, runner, verify_fixture)
        self.assertEqual(len(api.assets),9)
        outcome = release.published_candidate(self.root / 'retrieved-msi',REVISION,api,self.root,runner,verify_fixture)
        self.assertEqual(outcome['candidate'],'retrieved-and-verified')
        self.assertTrue((self.root / 'retrieved-msi' / f'insonic_{VERSION}_windows_amd64_installer.msi').is_file())
        api.assets = [item for item in api.assets if not item['name'].endswith('.msi')]
        with self.assertRaisesRegex(ValueError,'asset set'): release.published_candidate(self.root / 'missing-msi',REVISION,api,self.root,runner,verify_fixture)

    def test_retrieved_msi_revalidates_native_inventory_after_forged_hashes(self):
        target = self.candidate(); value = release.read_json(target / 'release-manifest.json')
        receipt = value['installers'][0]; artifact = target / receipt['artifact']; artifact.write_bytes(b'forged-msi')
        entry = release.file_entry(artifact); receipt['artifact_sha256'] = entry['sha256']; receipt['artifact_size_bytes'] = entry['size_bytes']
        value['assets'] = [entry if item['name'] == entry['name'] else item for item in value['assets']]
        release.write_json(target / 'release-manifest.json', value)
        entries = value['assets'] + [release.file_entry(target / 'release-manifest.json')]
        (target / 'SHA256SUMS').write_text(''.join(f"{item['sha256']}  {item['name']}\n" for item in sorted(entries,key=lambda item:item['name'])),encoding='utf-8',newline='\n')
        with self.assertRaisesRegex(ValueError,'installer qualification or inventory'):
            release.load_candidate(target,verify_fixture,self.root)

    def test_signed_payload_does_not_make_the_new_msi_signed(self):
        desktop_path = next(path for path in self.receipts if release.read_json(path).get('platform') == 'windows_amd64' and release.read_json(path).get('variant') == 'desktop')
        desktop = release.read_json(desktop_path); desktop['signing'] = {'configured':True,'verified':True,'status':'signed'}
        release.write_json(desktop_path,desktop)
        msi_path = self.receipts[-1]; msi = release.read_json(msi_path); msi['payload_receipt'] = copy.deepcopy(desktop)
        self.assertEqual(msi['signing'],{'configured':False,'verified':False,'status':'unconfigured'})
        release.write_json(msi_path,msi)
        target = self.candidate(); loaded,_ = release.load_candidate(target,verify_fixture,self.root)
        self.assertTrue(next(item for item in loaded['packages'] if item['platform'] == 'windows_amd64' and item['variant'] == 'desktop')['signing']['verified'])
        self.assertFalse(loaded['installers'][0]['signing']['verified'])
        msi['signing'] = copy.deepcopy(desktop['signing']); release.write_json(msi_path,msi)
        with self.assertRaisesRegex(ValueError,'unsigned'): self.candidate(output=self.root / 'false-signature')

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
                            deployment_child, verify_fixture, self.root, runner, source_root=self.root)
        self.assertEqual(invoked[0][1][secret_name], 'dedicated deployment credential')
        self.assertNotIn('GH_TOKEN', invoked[0][1]); self.assertNotIn('GITHUB_TOKEN', invoked[0][1])

    def test_stale_or_forged_documentation_marker_fails(self):
        for directory in ('site/out', 'site/offline'):
            path = self.root / directory; path.mkdir(parents=True); (path / 'index.html').write_text('<html></html>')
            release.write_json(path / 'documentation-build.json', {'version': VERSION, 'revision': 'b' * 40, 'source_dirty': False})
        with self.assertRaisesRegex(ValueError, 'source'): release.document_archives(self.root / 'doc-assets', REVISION, self.root, runner)
        with tarfile.open(self.assets / 'documentation.tar.gz', 'w:gz') as archive:
            for filename, value in [('release-documentation.json', {'version': VERSION, 'revision': REVISION, 'kind': 'documentation'}), ('documentation-build.json', {'version': VERSION, 'revision': 'b' * 40, 'source_dirty': False})]:
                raw = json.dumps(value).encode(); info = tarfile.TarInfo(filename); info.size = len(raw); archive.addfile(info, io.BytesIO(raw))
        with self.assertRaisesRegex(ValueError, 'build source'): self.candidate()

    def test_publication_uploads_complete_draft_then_publishes_and_reads_back(self):
        target = self.candidate(); api = GitHubFixture()
        outcome = release.publish(target, 'Portable recovery and complete package sources.', api, self.root, runner, verify_fixture)
        self.assertTrue(outcome['published']); self.assertFalse(api.release['draft']); self.assertEqual(len(api.assets), 9)
        self.assertTrue(api.release['body'].endswith(f'/blob/v{VERSION}/CHANGELOG.md)'))
        publishing = next(i for i, call in enumerate(api.calls) if call[0] == 'PATCH')
        self.assertEqual(sum(1 for call in api.calls[:publishing] if call[0] == 'POST' and 'uploads.github.com' in call[1]), 9)
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
        with self.assertRaisesRegex(ValueError, 'inventory'): release.load_candidate(target, verify_fixture, source_root=self.root)

    def test_github_transport_rejects_unselected_hosts_ports_and_userinfo_before_io(self):
        api = release.GitHub('synthetic-not-a-real-token')
        for endpoint in ['https://evil.example/upload', 'http://api.github.com/path', 'https://api.github.com:444/path', 'https://:secret@api.github.com/path']:
            with self.assertRaisesRegex(ValueError, 'endpoint'): api.call('POST', endpoint, {})

    def test_interrupted_draft_does_not_publish_or_promote_and_resume_is_idempotent(self):
        target = self.candidate(); api = GitHubFixture(fail_upload=True)
        with self.assertRaisesRegex(ValueError, 'interruption'): release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        self.assertTrue(api.release['draft']); self.assertFalse(any(call[0] == 'PATCH' for call in api.calls))
        invoked = []
        with self.assertRaisesRegex(ValueError, 'published'): release.promote(target, {'argv': ['/usr/bin/deploy', '{archive}']}, api, lambda args: invoked.append(args), verify_fixture, self.root, runner, source_root=self.root)
        self.assertEqual(invoked, [])
        api.fail_upload = False; release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
        self.assertEqual(len(api.assets), 9)
        release.promote(target, {'argv': [str(Path(sys.executable).resolve()), '{archive}', '{version}', '{revision}']}, api, lambda args: invoked.append(args), verify_fixture, self.root, runner, source_root=self.root)
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

    def superseded_release(self, api, run_id=66, suite_id=33):
        api.past_workflows[run_id] = {**api.workflow, 'id': run_id, 'check_suite_id': suite_id,
                                      'head_sha': REVISION, 'status': 'completed', 'conclusion': 'cancelled'}
        api.checks.append({'name': 'obsolete release lane', 'status': 'completed', 'conclusion': 'cancelled',
                           'check_suite': {'id': suite_id},
                           'details_url': f'https://github.com/{release.REPOSITORY}/actions/runs/{run_id}/job/123'})

    def test_superseded_release_suites_allow_current_and_historical_source_publication(self):
        target = self.candidate()
        for current_revision in [REVISION, 'b' * 40]:
            with self.subTest(current_revision=current_revision):
                api = GitHubFixture(); api.workflow['head_sha'] = current_revision
                api.checks.extend({'name': f'qualified lane {index}', 'status': 'completed', 'conclusion': 'success',
                                   'check_suite': {'id': 23}} for index in range(100))
                self.superseded_release(api)
                api.checks.append({**api.checks[-1], 'name': 'another obsolete lane', 'conclusion': 'failure'})
                with patch.dict(os.environ, {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY,
                                             'GITHUB_SHA': current_revision}):
                    self.assertTrue(release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)['published'])
                self.assertEqual(sum(endpoint.endswith('/actions/runs/66') for _, endpoint, _ in api.calls), 1)

    def test_main_workflow_path_forms_preserve_current_and_superseded_identity(self):
        target = self.candidate()
        paths = ['.github/workflows/release.yml', '.github/workflows/release.yml@main',
                 '.github/workflows/release.yml@refs/heads/main']
        context = {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}
        for current_path in paths:
            for previous_path in paths:
                with self.subTest(current_path=current_path, previous_path=previous_path):
                    api = GitHubFixture(); api.workflow['path'] = current_path
                    self.superseded_release(api); api.past_workflows[66]['path'] = previous_path
                    with patch.dict(os.environ, context):
                        self.assertTrue(release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)['published'])

    def test_all_documented_terminal_release_outcomes_are_superseded(self):
        target = self.candidate()
        context = {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}
        outcomes = ['action_required', 'cancelled', 'failure', 'neutral', 'skipped', 'stale', 'success', 'timed_out']
        for outcome in outcomes:
            with self.subTest(outcome=outcome):
                api = GitHubFixture(); self.superseded_release(api)
                api.past_workflows[66]['conclusion'] = outcome
                api.checks[-1]['conclusion'] = outcome
                selected = release.external_source_checks(api, REVISION, api.checks, context)
                self.assertEqual({check['check_suite']['id'] for check in selected}, {11})
                with patch.dict(os.environ, context):
                    self.assertTrue(release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)['published'])

    def test_superseded_release_identity_mismatches_and_active_or_newer_runs_block_all_writes(self):
        target = self.candidate()
        invalid = [{'head_sha': 'c' * 40}, {'repository': {'full_name': 'someone/else'}},
                   {'path': '.github/workflows/ci.yml'}, {'path': '.github/workflows/release.yml@feature'},
                   {'path': '.github/workflows/release.yml@refs/heads/feature'},
                   {'path': '.github/workflows/release.yml@mainish'}, {'head_branch': 'feature'}, {'event': 'pull_request'},
                   {'workflow_id': 1234}, {'check_suite_id': 999}, {'id': 67}, {'status': 'in_progress'}, {'conclusion': None},
                   {'conclusion': ''}, {'conclusion': 'unknown-outcome'}]
        context = {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}
        for changes in invalid + [None]:
            with self.subTest(changes=changes):
                api = GitHubFixture(); self.superseded_release(api, run_id=88 if changes is None else 66)
                if changes: api.past_workflows[66].update(changes)
                with patch.dict(os.environ, context), self.assertRaisesRegex(ValueError, 'not green'):
                    release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
                self.assertFalse(any(method in ('POST', 'PATCH') for method, _, _ in api.calls))

    def test_superseded_release_does_not_hide_security_ci_or_missing_required_checks(self):
        target = self.candidate()
        context = {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}
        for kind in ['security', 'ci', 'missing-required']:
            with self.subTest(kind=kind):
                api = GitHubFixture(); self.superseded_release(api)
                if kind == 'missing-required':
                    api.checks[0].update(check_suite={'id': 33}, details_url=api.checks[-1]['details_url'])
                else:
                    api.checks.append({'name': kind, 'status': 'completed', 'conclusion': 'failure',
                                       'check_suite': {'id': 23}, 'details_url': api.checks[-1]['details_url']})
                with patch.dict(os.environ, context), self.assertRaisesRegex(ValueError, 'not green'):
                    release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
                self.assertFalse(any(method in ('POST', 'PATCH') for method, _, _ in api.calls))

    def test_superseded_release_requires_verified_current_context_and_canonical_run_url(self):
        target = self.candidate()
        context = {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}
        for prefix in ['https://untrusted.example/', 'https://github.com:444/', 'https://user@github.com/',
                       'https://github.com/', None]:
            api = GitHubFixture(); self.superseded_release(api)
            if prefix: api.checks[-1]['details_url'] = prefix + release.REPOSITORY + '/actions/runs/66/job/123' + ('?unverified=1' if prefix == 'https://github.com/' else '')
            selected = context if prefix else dict(context, GITHUB_RUN_ID='')
            with self.subTest(prefix=prefix), patch.dict(os.environ, selected), self.assertRaisesRegex(ValueError, 'not green'):
                release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
            self.assertFalse(any(method in ('POST', 'PATCH') for method, _, _ in api.calls))

    def test_unverified_or_irrelevant_suites_cannot_hide_checks(self):
        context = {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}
        api = GitHubFixture()
        for changes in [{'path': '.github/workflows/ci.yml'}, {'id': 78}, {'check_suite_id': None},
                        {'repository': {'full_name': 'someone/else'}}, {'head_sha': 'b' * 40},
                        {'head_branch': 'feature'}, {'workflow_id': None}]:
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

    def test_publication_reads_required_checks_after_the_first_hundred(self):
        target = self.candidate(); api = GitHubFixture()
        api.checks = [{'name': f'qualified lane {index}', 'status': 'completed', 'conclusion': 'success',
                       'check_suite': {'id': 23}} for index in range(105)] + api.checks
        self.assertTrue(release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)['published'])
        self.assertEqual(len([call for call in api.calls if '/check-runs?' in call[1]]), 2)

    def test_publication_blocks_red_or_pending_checks_on_later_pages(self):
        target = self.candidate()
        for status, conclusion in [('in_progress', None), ('completed', 'failure')]:
            api = GitHubFixture()
            api.checks.extend({'name': f'qualified lane {index}', 'status': 'completed', 'conclusion': 'success',
                               'check_suite': {'id': 23}} for index in range(100))
            api.checks.append({'name': 'external review', 'status': status, 'conclusion': conclusion,
                               'check_suite': {'id': 23}})
            with self.assertRaisesRegex(ValueError, 'not green'):
                release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
            self.assertFalse(any(call[0] in ('POST', 'PATCH') for call in api.calls))

    def test_verified_current_suite_exclusion_applies_on_later_pages(self):
        target = self.candidate(); api = GitHubFixture()
        api.checks.extend({'name': f'qualified lane {index}', 'status': 'completed', 'conclusion': 'success',
                           'check_suite': {'id': 23}} for index in range(100))
        api.checks.append({'name': 'publish', 'status': 'in_progress', 'conclusion': None,
                           'check_suite': {'id': 42}})
        with patch.dict(os.environ, {'GITHUB_RUN_ID': '77', 'GITHUB_REPOSITORY': release.REPOSITORY, 'GITHUB_SHA': REVISION}):
            self.assertTrue(release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)['published'])

    def test_incomplete_or_changing_check_inventory_cannot_publish(self):
        target = self.candidate()
        for malformed in ('empty', 'count', 'duplicate'):
            api = GitHubFixture()
            api.checks.extend({'name': f'qualified lane {index}', 'status': 'completed', 'conclusion': 'success',
                               'check_suite': {'id': 23}} for index in range(100))
            original = api.call
            def changed(method, endpoint, value=None, path=None):
                response = original(method, endpoint, value, path)
                if '/check-runs?' in endpoint and 'page=2' in endpoint:
                    if malformed == 'empty': response['check_runs'] = []
                    if malformed == 'count': response['total_count'] += 1
                    if malformed == 'duplicate': response['check_runs'][0]['id'] = 1
                return response
            api.call = changed
            with self.assertRaisesRegex(ValueError, 'check inventory'):
                release.publish(target, 'Highlights.', api, self.root, runner, verify_fixture)
            self.assertFalse(any(call[0] in ('POST', 'PATCH') for call in api.calls))

    def test_promotion_only_retrieves_original_candidate_without_rebuilding_or_publishing(self):
        original = self.candidate(); api = GitHubFixture()
        release.publish(original, 'Highlights.', api, self.root, runner, verify_fixture)
        api.calls.clear()
        retrieved = self.root / 'retrieved'
        result = release.published_candidate(retrieved, REVISION, api, self.root, runner, verify_fixture)
        self.assertEqual(result, {'candidate': 'retrieved-and-verified', 'version': VERSION, 'revision': REVISION})
        self.assertEqual(len(api.downloads), 9)
        for path in original.iterdir():
            self.assertEqual((retrieved / path.name).read_bytes(), path.read_bytes())
        invoked = []
        release.promote(retrieved, {'argv': [str(Path(sys.executable).resolve()), '{archive}']}, api,
                        lambda args: invoked.append(args), verify_fixture, self.root, runner, source_root=self.root)
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
        with self.assertRaisesRegex(ValueError, 'digest'): release.promote(target, {'argv': [str(self.root / 'deploy')]}, api, lambda args: self.fail('deployment ran'), verify_fixture, self.root, runner, source_root=self.root)
        (target / 'offline-help.tar.gz').write_bytes(b'tampered')
        with self.assertRaisesRegex(ValueError, 'changed'): release.load_candidate(target, verify_fixture, source_root=self.root)


if __name__ == '__main__': unittest.main()
