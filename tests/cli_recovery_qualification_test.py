# SPDX-License-Identifier: Apache-2.0
import copy
import json
from pathlib import Path
import sys
import unittest
import uuid

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from cli_recovery_qualification import CHECKS, derived, validate_receipt
from cli_recovery_qualification import qualify
from cli_recovery_qualification import fixture_tools
from cli_recovery_qualification import entered_date_preserved


def receipt_fixture(digest='a' * 64, platform='windows'):
    cases = {}
    for name in ('same-source', 'changed-source', 'corrupt-report'):
        uid = lambda: str(uuid.uuid4())
        work_id = uid()
        before = {'pid': 10, 'session': uid(), 'generation': 1, 'journal_receipt_id': uid(),
                  'canonical': {'id': derived(work_id, 'canonical-0'), 'digest': 'b' * 64, 'size': 120},
                  'report': {'id': derived(work_id, 'canonical-report-0'), 'digest': 'c' * 64, 'size': 800},
                  'admission_absent': True, 'barrier': 'windows-rw-oplock' if platform == 'windows' else 'linux-write-lease',
                  'ack_required': True}
        after = {'pid': 11, 'session': uid(), 'generation': 2, 'journal_receipt_id': uid(),
                 'state': 'succeeded' if name == 'same-source' else 'failed',
                 'error': '' if name == 'same-source' else 'conflict' if name == 'changed-source' else 'checksum_mismatch',
                 'entry_id': uid() if name == 'same-source' else '', 'metadata_sha256': 'd' * 64,
                 'report_sha256': 'e' * 64 if name == 'corrupt-report' else 'c' * 64}
        cases[name] = {'work_id': work_id, 'source_sha256': 'f' * 64, 'source_size': 100,
                       'changed_sha256': '0' * 64 if name == 'changed-source' else None,
                       'before': before, 'after': after,
                       'termination': {'exit_code': 1, 'process_dead': True, 'within_barrier': True},
                       'checks': {key: True for key in CHECKS[name]}}
    return {'format': 'insonic-import-recovery-qualification-v1', 'platform': platform,
            'executable_sha256': digest, 'cases': cases, 'cleanup': True,
            'fixture_removed': True, 'runtime_natural_exit': True, 'barriers_released': True}


class RecoveryReceiptTests(unittest.TestCase):
    def test_entered_date_keeps_exact_instant_and_interpretation_provenance(self):
        selected = {'literal': '2020-01-02T03:04:05.123456789Z', 'basis': 'owner', 'precision': 'instant',
                    'zone': 'UTC', 'zone_source': 'entered-offset', 'offset_seconds': 0,
                    'interpretation_state': 'resolved', 'assumptions': [], 'rules_version': 'pinned', 'mapping_version': 'pinned',
                    'resolved': {'iso': '2020-01-02T03:04:05.123456789Z', 'unix_ns': 1577934245123456789}}
        self.assertTrue(entered_date_preserved({'selected': selected}))
        for field, changed in [('precision', 'day'), ('zone_source', 'captured-local'), ('offset_seconds', 3600),
                               ('basis', 'filesystem-modified'), ('rules_version', ''),
                               ('resolved', {'iso': selected['literal'], 'unix_ns': 1577934245123456700})]:
            self.assertFalse(entered_date_preserved({'selected': {**selected, field: changed}}))

    def test_controlled_probe_keeps_original_companion_integrity(self):
        import tempfile
        import time
        with tempfile.TemporaryDirectory() as root:
            path = Path(root) / 'selected probe.bin'
            path.write_bytes(b'changed owned image')
            with self.assertRaisesRegex(ValueError, 'pinned selected companion'):
                fixture_tools({'ffprobe': {'path': str(path), 'sha256': '0' * 64}}, Path(root), time.monotonic() + 30)

    def test_early_campaign_failure_removes_only_new_owned_paths(self):
        import tempfile
        observed = []
        def invoke(*_args, **options):
            observed.append(options['execution_environment']['HOME'])
            raise RuntimeError('original fixture refusal')
        with tempfile.TemporaryDirectory() as root:
            tools = Path(root) / 'fixture-tools.json'
            tools.write_text(json.dumps({}), encoding='utf-8')
            with self.assertRaisesRegex(RuntimeError, 'original fixture refusal'):
                qualify(invoke, root, '1.1.0', executable=sys.executable, environment={}, source_tools=tools)
            self.assertEqual(list(Path(root).iterdir()), [tools])
        self.assertFalse(Path(observed[0]).exists())

    def test_preexisting_campaign_directory_is_never_removed(self):
        import tempfile
        with tempfile.TemporaryDirectory() as root:
            owned = Path(root) / 'import recovery café'
            owned.mkdir()
            sentinel = owned / 'owner.txt'
            sentinel.write_text('unchanged', encoding='utf-8')
            with self.assertRaises(FileExistsError):
                qualify(lambda *_a, **_k: {}, root, '1.1.0', executable=sys.executable, environment={})
            self.assertEqual(sentinel.read_text(encoding='utf-8'), 'unchanged')

    def test_complete_matrix_for_both_platforms(self):
        for platform in ('windows', 'linux'):
            validate_receipt(receipt_fixture(platform=platform), 'a' * 64, platform)

    def test_identity_boundary_attempt_and_integrity_mutations_are_rejected(self):
        original = receipt_fixture()
        changes = [lambda x: x.update(executable_sha256='b' * 64),
                   lambda x: x['cases'].pop('corrupt-report'),
                   lambda x: x['cases']['same-source']['before'].update(admission_absent=False),
                   lambda x: x['cases']['same-source']['before'].update(ack_required=False),
                   lambda x: x['cases']['same-source']['after'].update(generation=1),
                   lambda x: x['cases']['same-source']['after'].update(session=x['cases']['same-source']['before']['session']),
                   lambda x: x['cases']['same-source']['termination'].update(exit_code=0),
                   lambda x: x['cases']['same-source']['termination'].update(within_barrier=False),
                   lambda x: x['cases']['changed-source']['after'].update(entry_id=str(uuid.uuid4())),
                   lambda x: x['cases']['corrupt-report']['after'].update(error=''),
                   lambda x: x['cases']['same-source']['checks'].update(raw_metadata_preserved=False),
                   lambda x: x.update(cleanup=False), lambda x: x.update(fixture_removed=False)]
        for index, change in enumerate(changes):
            with self.subTest(index=index):
                value = copy.deepcopy(original)
                change(value)
                with self.assertRaises(ValueError):
                    validate_receipt(value, 'a' * 64, 'windows')


if __name__ == '__main__':
    unittest.main()
