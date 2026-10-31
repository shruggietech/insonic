# SPDX-License-Identifier: Apache-2.0
import copy
from pathlib import Path
import sys
import unittest
import uuid

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))


def receipt_fixture(image='a' * 64, platform='windows'):
    import cli_backup_qualification as module
    source, bundle, publication = [str(uuid.uuid4()) for _ in range(3)]
    return {'format': 'insonic-backup-recovery-qualification-v1', 'platform': platform,
            'executable_sha256': image, 'origin': 'actual-filesystem-obstruction', 'physical_exhaustion': False,
            'source_workspace_id': source, 'bundle_id': bundle, 'bundle_digest': 'b' * 64,
            'publication': {'id': publication, 'digest': 'c' * 64, 'size': 1000},
            'cases': {name: {'failed_request_id': bundle if name == 'backup-parent' else str(uuid.uuid4()),
                             'retry_request_id': bundle if name == 'backup-parent' else str(uuid.uuid4()),
                             'failed_code': module.FAILURES[name][0], 'bundle_id': bundle, 'bundle_digest': 'b' * 64,
                             'verified_digest': 'c' * 64, 'destination_profile_id': str(uuid.uuid4()),
                             'checks': {key: True for key in module.checks_for(name)}} for name in module.CASES},
            'cleanup': True, 'fixture_removed': True, 'runtime_natural_exit': True}


class BackupReceiptTests(unittest.TestCase):
    def test_complete_bound_receipt(self):
        import cli_backup_qualification as module
        for platform in ('windows', 'linux'):
            module.validate_receipt(receipt_fixture(platform=platform), 'a' * 64, platform)

    def test_incomplete_mislabeled_and_mismatched_evidence_is_refused(self):
        import cli_backup_qualification as module
        mutations = [lambda r: r.pop('origin'), lambda r: r.update(origin='injected-ENOSPC'),
                     lambda r: r.update(physical_exhaustion=True), lambda r: r.update(physical_exhaustion=0),
                     lambda r: r.update(executable_sha256='0' * 64), lambda r: r.update(platform='other'),
                     lambda r: r.update(runtime_natural_exit=False), lambda r: r.update(fixture_removed=False),
                     lambda r: r['cases'].pop('restore-progress'), lambda r: r['cases'].update(unknown={}),
                     lambda r: r['publication'].update(size=True), lambda r: r['publication'].update(id='invalid')]
        for name in module.CASES:
            mutations += [lambda r, n=name: r['cases'][n].update(failed_code='conflict'),
                          lambda r, n=name: r['cases'][n].update(bundle_id=str(uuid.uuid4())),
                          lambda r, n=name: r['cases'][n].update(bundle_digest='e' * 64),
                          lambda r, n=name: r['cases'][n].update(verified_digest='f' * 64),
                          lambda r, n=name: r['cases'][n]['checks'].pop('authority_preserved'),
                          lambda r, n=name: r['cases'][n]['checks'].update(accepted_bytes_verified=False)]
        mutations += [lambda r: r['cases']['backup-parent'].update(retry_request_id=str(uuid.uuid4()))]
        for mutate in mutations:
            value = receipt_fixture()
            mutate(value)
            with self.subTest(mutation=mutate), self.assertRaises(ValueError):
                module.validate_receipt(value, 'a' * 64, 'windows')

    def test_owned_blocker_cannot_remove_replaced_or_escaped_input(self):
        import tempfile
        import cli_backup_qualification as module
        for changed in ('none', 'bytes', 'path'):
            with self.subTest(changed=changed), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                blocker = module.block(root, root / 'nested' / 'obstruction', directory=True)
                outside = root / 'untouched'
                outside.write_bytes(b'owner sentinel')
                if changed == 'bytes': blocker['marker'].write_bytes(b'changed')
                if changed == 'path': blocker['marker'] = outside
                if changed == 'none': module.unblock(root, blocker)
                else:
                    with self.assertRaises(ValueError): module.unblock(root, blocker)
                self.assertEqual(outside.read_bytes(), b'owner sentinel')


if __name__ == '__main__':
    unittest.main()
