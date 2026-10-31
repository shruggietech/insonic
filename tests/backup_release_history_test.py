# SPDX-License-Identifier: Apache-2.0
import copy
import gzip
import importlib.util
import json
from pathlib import Path
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
from release_history import ORIGINAL_V103_MANIFEST, ORIGINAL_V103_PACKAGES, digest, original_backup_absence


class OriginalBackupCompatibilityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        value = json.loads(gzip.decompress((ROOT / 'tests/fixtures/releases/original-v1.0.3-package-receipts.json.gz').read_bytes()))
        if value['manifest_sha256'] != ORIGINAL_V103_MANIFEST or value['revision'] != '72098446d435aa1e3d6d6726a9f07e04e0b227a6':
            raise ValueError('Original v1.0.3 fixture source differs')
        cls.receipts = value['packages']
        spec = importlib.util.spec_from_file_location('backup_release', ROOT / 'scripts/release.py')
        cls.release = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.release)

    def test_original_publication_keeps_only_its_absent_new_evidence(self):
        self.assertEqual(len(self.receipts), 4)
        self.assertEqual({digest(r) for r in self.receipts}, ORIGINAL_V103_PACKAGES)
        for receipt in self.receipts:
            self.assertNotIn('backup_contracts', receipt['cli_contract'])
            self.assertTrue(original_backup_absence(receipt))
            self.release.verify_qualification(receipt)

    def test_altered_receipts_cannot_inherit_historical_exemption(self):
        for receipt in self.receipts:
            for key, value in [('revision', '1' * 40), ('source_dirty', True), ('inventory_sha256', 'e' * 64)]:
                changed = copy.deepcopy(receipt)
                changed[key] = value
                self.assertFalse(original_backup_absence(changed))
                with self.assertRaises(ValueError): self.release.verify_qualification(changed)

    def test_current_receipts_require_complete_backup_evidence(self):
        from cli_backup_qualification_test import receipt_fixture
        for original in self.receipts:
            current = copy.deepcopy(original)
            current['revision'] = '2' * 40
            identity = current['cli_contract']['coverage']['identity']
            current['cli_contract']['backup_contracts'] = receipt_fixture(identity['executable_sha256'], identity['platform'])
            self.release.verify_qualification(current)
            del current['cli_contract']['backup_contracts']['cases']['restore-progress']
            with self.assertRaises(ValueError): self.release.verify_qualification(current)


if __name__ == '__main__':
    unittest.main()
