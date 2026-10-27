# SPDX-License-Identifier: Apache-2.0
import copy
import gzip
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
import release
import release_history as history
import windows_msi

RAW = gzip.decompress((ROOT / 'tests/fixtures/release-history/v1.0.1-manifest.json.gz').read_bytes())
ORIGINAL = json.loads(RAW)


class OriginalReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.manifest = self.root / 'release-manifest.json'
        self.manifest.write_bytes(RAW)

    def profile(self):
        with patch.object(history, 'verify_source') as proof:
            result = history.select_profile(ORIGINAL, self.manifest, self.root)
        self.assertIsNotNone(result)
        proof.assert_called_once()
        return result

    def test_authentic_manifest_elects_profile_only_after_source_proof(self):
        profile = self.profile()
        self.assertEqual(profile.version, '1.0.1')
        self.assertEqual(profile.revision, ORIGINAL['revision'])
        with patch.object(history, 'verify_source', side_effect=ValueError('changed source')):
            with self.assertRaisesRegex(ValueError, 'source'):
                history.select_profile(ORIGINAL, self.manifest, self.root)

    def test_unknown_identity_does_not_elect_a_historical_profile(self):
        for field, value in [('version', '1.0.2'), ('revision', 'f' * 40)]:
            altered = copy.deepcopy(ORIGINAL); altered[field] = value
            with self.subTest(field=field), patch.object(history, 'verify_source') as proof:
                self.assertIsNone(history.select_profile(altered, self.manifest, self.root))
                proof.assert_not_called()

    def test_changed_manifest_bytes_are_refused_even_when_identity_is_retained(self):
        self.manifest.write_bytes(RAW + b' ')
        with patch.object(history, 'verify_source') as proof:
            with self.assertRaisesRegex(ValueError, 'manifest'):
                history.select_profile(ORIGINAL, self.manifest, self.root)
            proof.assert_not_called()

    def test_parsed_manifest_must_equal_the_pinned_bytes(self):
        changed = copy.deepcopy(ORIGINAL)
        changed['packages'][0]['signing']['verified'] = True
        with self.assertRaisesRegex(ValueError, 'manifest'):
            history.select_profile(changed, self.manifest, self.root)

    def test_original_receipts_keep_first_use_and_exact_original_declarations(self):
        profile = self.profile()
        for receipt in ORIGINAL['packages']:
            release.verify_qualification(receipt, history_profile=profile)
            for field, value in [('inventory_sha256', 'f' * 64), ('archive_sha256', 'f' * 64),
                                 ('signing', {'configured': True, 'verified': True, 'status': 'signed'})]:
                changed = copy.deepcopy(receipt); changed[field] = value
                with self.subTest(platform=receipt['platform'], field=field):
                    with self.assertRaises(ValueError):
                        release.verify_qualification(changed, history_profile=profile)
            if receipt['platform'] == 'windows_amd64':
                changed = copy.deepcopy(receipt); changed['windows_first_use'] = 'failed'
                with self.assertRaisesRegex(ValueError, 'qualification'):
                    release.verify_qualification(changed, history_profile=profile)

    def test_original_version_alone_cannot_skip_modern_terminal_or_setup_evidence(self):
        for receipt in ORIGINAL['packages']:
            with self.assertRaisesRegex(ValueError, 'Terminal'):
                release.verify_qualification(receipt)
        installer = ORIGINAL['installers'][0]
        with self.assertRaisesRegex(ValueError, 'setup'):
            release.installer_identity(installer, '1.0.1', ORIGINAL['revision'], ORIGINAL['packages'])
        release.installer_identity(installer, '1.0.1', ORIGINAL['revision'], ORIGINAL['packages'],
                                   history_profile=self.profile())

    def test_altered_installer_inventory_lifecycle_and_signing_are_refused(self):
        profile = self.profile()
        for field in ['payload_inventory', 'qualification', 'signing', 'artifact_sha256']:
            changed = copy.deepcopy(ORIGINAL['installers'][0])
            changed[field] = {} if field != 'artifact_sha256' else 'f' * 64
            with self.subTest(field=field), self.assertRaises(ValueError):
                release.installer_identity(changed, '1.0.1', ORIGINAL['revision'], ORIGINAL['packages'],
                                           history_profile=profile)

    def test_pinned_msi_bytes_cannot_be_replaced_by_rehashed_table_changes(self):
        profile = self.profile()
        installer = copy.deepcopy(ORIGINAL['installers'][0])
        artifact = self.root / installer['artifact']; artifact.write_bytes(b'changed MSI Property table')
        installer['artifact_sha256'] = hashlib.sha256(artifact.read_bytes()).hexdigest()
        installer['artifact_size_bytes'] = artifact.stat().st_size
        with patch.object(history, 'verify_source'), self.assertRaises(ValueError):
            windows_msi.verify_installer(artifact, installer, self.root, history_profile=profile)

    def test_original_profile_cannot_authorize_another_source_root(self):
        profile = self.profile()
        installer = ORIGINAL['installers'][0]
        with self.assertRaisesRegex(ValueError, 'source'):
            windows_msi.verify_installer(self.root / installer['artifact'], installer,
                                        self.root / 'different', history_profile=profile)

    def test_candidate_loading_propagates_one_profile_and_rechecks_source(self):
        entries = {entry['name']: entry for entry in ORIGINAL['assets']}
        entries['release-manifest.json'] = release.file_entry(self.manifest)
        sums = ''.join(f"{entry['sha256']}  {entry['name']}\n" for entry in sorted(entries.values(), key=lambda e:e['name']))
        checksum_path = self.root / 'SHA256SUMS'; checksum_path.write_text(sums, encoding='utf-8')
        entries['SHA256SUMS'] = release.file_entry(checksum_path)
        with (patch.object(history, 'verify_source') as source,
              patch.object(release, 'selected_release_platforms', return_value=release.PLATFORMS),
              patch.object(release, 'selected_installer_count', return_value=1),
              patch.object(release, 'selected_setup_count', return_value=0),
              patch.object(release, 'file_entry', side_effect=lambda path: entries[path.name]),
              patch.object(release, 'verify_selected_package') as nested,
              patch.object(release, 'verify_documentation')):
            loaded, expected = release.load_candidate(self.root, source_root=self.root)
            source.reset_mock(); nested.reset_mock()
            source.side_effect = [None, ValueError('source changed during verification')]
            with self.assertRaisesRegex(ValueError, 'source changed'):
                release.load_candidate(self.root, source_root=self.root)
        self.assertEqual(loaded, ORIGINAL)
        self.assertEqual(len(expected), 9)
        self.assertEqual(source.call_count, 2)
        self.assertEqual(nested.call_count, 5)
        profiles = [call.kwargs['history_profile'] for call in nested.call_args_list]
        self.assertTrue(all(profile is profiles[0] for profile in profiles))
        self.assertIsInstance(profiles[0], history.OriginalProfile)

    def test_nested_msi_verifier_receives_explicit_profile(self):
        profile = self.profile(); installer = ORIGINAL['installers'][0]
        path = self.root / installer['artifact']
        with patch.object(release, 'verify_installer') as verify:
            release.verify_selected_package(path, installer, release.verify_package,
                                            self.root, history_profile=profile)
        verify.assert_called_once_with(path, installer, self.root, history_profile=profile)

    def test_unknown_and_current_receipts_still_require_modern_evidence(self):
        for version, revision in [('1.0.1', 'f' * 40), ('1.0.2', 'a' * 40), ('99.0.0', 'a' * 40)]:
            package = copy.deepcopy(ORIGINAL['packages'][0])
            package.update(schema_version=version, product_version=version,
                           documentation_version=version, revision=revision)
            with self.subTest(version=version), self.assertRaisesRegex(ValueError, 'Terminal'):
                release.verify_qualification(package)
            installer = copy.deepcopy(ORIGINAL['installers'][0])
            installer.update(schema_version=version, product_version=version,
                             documentation_version=version, revision=revision,
                             artifact=f'insonic_{version}_windows_amd64_installer.msi')
            desktop = installer['payload_receipt']
            with self.subTest(version=version), self.assertRaisesRegex(ValueError, 'setup'):
                release.installer_identity(installer, version, revision, [desktop])


class CompleteSourceTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.contents = {'VERSION': b'1.0.1\n', 'scripts/recipe.py': b'raise RuntimeError("never execute")\n'}
        records = []
        for name, raw in self.contents.items():
            destination = self.root / name; destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(raw)
            blob = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
            records.append('100644 blob ' + blob + '\t' + name)
        self.inventory = '\0'.join(records) + '\0'

    def metadata(self, args, repository):
        if 'rev-parse' in args: return history.ORIGINAL_TREE
        if 'ls-tree' in args: return self.inventory
        self.fail('Unexpected metadata command: ' + repr(args))

    def test_all_tracked_files_are_checked_without_running_selected_scripts(self):
        result = history.verify_source(self.root, git=self.metadata)
        self.assertEqual(result['files'], 2)

    def test_missing_changed_or_nonregular_source_is_refused(self):
        recipe = self.root / 'scripts/recipe.py'
        for mutation in ['missing', 'changed', 'directory']:
            with self.subTest(mutation=mutation):
                if recipe.exists():
                    if recipe.is_dir(): recipe.rmdir()
                    else: recipe.unlink()
                if mutation == 'changed': recipe.write_bytes(b'changed')
                if mutation == 'directory': recipe.mkdir()
                with self.assertRaisesRegex(ValueError, 'source'):
                    history.verify_source(self.root, git=self.metadata)

    def test_schema_only_source_is_not_complete(self):
        (self.root / 'scripts/recipe.py').unlink()
        with self.assertRaisesRegex(ValueError, 'source'):
            history.verify_source(self.root, git=self.metadata)

    def test_linked_source_member_is_refused_before_reading(self):
        original = Path.is_symlink
        def linked(path):
            return path.name == 'recipe.py' or original(path)
        with patch.object(Path, 'is_symlink', linked), self.assertRaisesRegex(ValueError, 'linked'):
            history.verify_source(self.root, git=self.metadata)

    def test_untrusted_tree_and_unsafe_member_paths_are_refused(self):
        with self.assertRaisesRegex(ValueError, 'tree'):
            history.verify_source(self.root, git=lambda args, repository: 'f' * 40)
        self.inventory += '100644 blob ' + 'a' * 40 + '\t../outside\0'
        with self.assertRaisesRegex(ValueError, 'source'):
            history.verify_source(self.root, git=self.metadata)


if __name__ == '__main__': unittest.main()
