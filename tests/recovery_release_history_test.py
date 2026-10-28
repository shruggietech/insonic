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
from release_history import ORIGINAL_V102_MANIFEST, ORIGINAL_V102_PACKAGES, digest, original_recovery_absence


class OriginalRecoveryCompatibilityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        data = gzip.decompress((ROOT / 'tests/fixtures/releases/original-v1.0.2-package-receipts.json.gz').read_bytes())
        value = json.loads(data)
        if value['manifest_sha256'] != ORIGINAL_V102_MANIFEST:
            raise ValueError('Original receipt fixture source manifest differs')
        cls.receipts = value['packages']
        spec = importlib.util.spec_from_file_location('recovery_release', ROOT / 'scripts/release.py')
        cls.release = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.release)

    def test_all_four_original_receipts_retain_preexisting_qualification(self):
        self.assertEqual({digest(receipt) for receipt in self.receipts}, ORIGINAL_V102_PACKAGES)
        self.assertEqual(len(self.receipts), 4)
        for receipt in self.receipts:
            with self.subTest(platform=receipt['platform'], variant=receipt['variant']):
                self.assertNotIn('recovery_contracts', receipt['cli_contract'])
                self.assertNotIn('storage_contracts', receipt['cli_contract'])
                self.assertTrue(original_recovery_absence(receipt))
                self.release.verify_qualification(receipt)

    def test_changed_or_unknown_receipt_cannot_omit_new_evidence(self):
        for receipt in self.receipts:
            for field, value in [('revision', '1' * 40), ('product_version', '9.0.0'), ('source_dirty', True),
                                 ('windows_first_use', 'not-run'), ('inventory_sha256', 'e' * 64)]:
                altered = copy.deepcopy(receipt)
                altered[field] = value
                with self.subTest(platform=receipt['platform'], field=field):
                    self.assertFalse(original_recovery_absence(altered))
                    with self.assertRaises(ValueError):
                        self.release.verify_qualification(altered)


if __name__ == '__main__':
    unittest.main()
