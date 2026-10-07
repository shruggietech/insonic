# SPDX-License-Identifier: Apache-2.0
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('qualify_secrets', Path(__file__).resolve().parents[1] / 'scripts/qualify.py')
qualify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualify)


class SecretQualificationTests(unittest.TestCase):
    def test_native_lifecycle_is_required_and_receipt_records_actual_operations(self):
        result = {'schema_version': qualify.VERSION, 'platform': 'windows',
                  'native_credential_lifecycle': 'passed', 'cross_process_restart': 'passed', 'credential_values': 'not-returned'}
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(qualify, 'BUILD', Path(directory)), patch.object(qualify, 'child', side_effect=[b'', json.dumps(result).encode(), b'']):
                qualify.secrets()
                receipt = json.loads((Path(directory) / 'secret-service-receipt.json').read_text(encoding='utf-8'))
                self.assertEqual(receipt['native_credential_lifecycle'], 'passed')

    def test_unavailable_native_store_cannot_pass_qualification(self):
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(qualify, 'BUILD', Path(directory)), patch.object(qualify, 'child', side_effect=[b'', b'{"native_credential_lifecycle":"failed"}']):
                with self.assertRaises(ValueError):
                    qualify.secrets()


if __name__ == '__main__':
    unittest.main()
