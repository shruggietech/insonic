# SPDX-License-Identifier: Apache-2.0
import copy
from pathlib import Path
import sys
import tempfile
import unittest
import uuid
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from cli_terminal_qualification import REQUIRED, qualify, validate_receipt


def receipt_fixture(digest='a' * 64, platform='windows', transport='conpty'):
    outcomes = {}
    for name, (code, state, reason) in REQUIRED.items():
        outcomes[name] = {'work_id': str(uuid.uuid4()), 'request_id': str(uuid.uuid4()),
                          'exit_code': code, 'durable_state': state, 'reason': reason,
                          'natural_exit': True, 'cleanup': True, 'elapsed_seconds': .5,
                          'bound_seconds': 600, 'progress': name in ('human', 'interrupt', 'cancelled', 'failed'),
                          'count': name in ('human', 'interrupt', 'cancelled', 'failed')}
    for name in ('json', 'quiet', 'redirected'):
        outcomes[name]['work_id'] = outcomes['human']['work_id']
    return {'format': 'insonic-terminal-qualification-v1', 'platform': platform, 'transport': transport,
            'executable_sha256': digest, 'cases': outcomes, 'cleanup': True,
            'runtime_natural_exit': True, 'continued_work': True, 'server_released': True,
            'fixture_removed': True}


def package_contract_fixture(receipt):
    platform = 'windows' if receipt['platform'] == 'windows_amd64' else 'linux'
    return {'coverage': {'identity': {'scope': 'package', 'archive_sha256': receipt['archive_sha256'],
        'inventory_sha256': receipt['inventory_sha256'], 'executable_sha256': 'd' * 64, 'platform': platform}},
        'terminal_contracts': receipt_fixture('d' * 64, platform, 'conpty' if platform == 'windows' else 'linux-pty')}


class TerminalReceiptTests(unittest.TestCase):
    def test_complete_evidence_and_both_platforms(self):
        for platform, kind in [('windows', 'conpty'), ('linux', 'linux-pty')]:
            self.assertIsNone(validate_receipt(receipt_fixture(platform=platform, transport=kind), 'a' * 64, platform))

    def test_incomplete_or_false_evidence_is_refused(self):
        original = receipt_fixture()
        mutations = []
        for key in original:
            bad = copy.deepcopy(original); bad.pop(key); mutations.append(bad)
        for name in REQUIRED:
            bad = copy.deepcopy(original); bad['cases'].pop(name); mutations.append(bad)
            for key, value in [('natural_exit', False), ('cleanup', False), ('work_id', []),
                               ('request_id', 'bad'), ('exit_code', True), ('durable_state', 'invented'),
                               ('elapsed_seconds', float('nan')), ('elapsed_seconds', -1),
                               ('bound_seconds', 0), ('elapsed_seconds', 601)]:
                bad = copy.deepcopy(original); bad['cases'][name][key] = value; mutations.append(bad)
        for key, value in [('cleanup', False), ('runtime_natural_exit', False), ('continued_work', False),
                           ('fixture_removed', False), ('server_released', False),
                           ('executable_sha256', 'b' * 64), ('transport', 'pipe'), ('platform', 'linux')]:
            bad = copy.deepcopy(original); bad[key] = value; mutations.append(bad)
        for name in ('human', 'interrupt', 'cancelled', 'failed'):
            for key in ('progress', 'count'):
                bad = copy.deepcopy(original); bad['cases'][name][key] = False; mutations.append(bad)
        for name in ('json', 'quiet', 'redirected'):
            bad = copy.deepcopy(original); bad['cases'][name]['progress'] = True; mutations.append(bad)
        for bad in mutations:
            with self.subTest(index=mutations.index(bad)), self.assertRaises(ValueError):
                validate_receipt(bad, 'a' * 64, 'windows')

    def test_failed_launch_releases_only_new_owned_fixture(self):
        import cli_terminal_qualification as campaign
        environments = []
        def invoke(*_args, **options):
            environments.append(options['execution_environment'])
            return {}
        with tempfile.TemporaryDirectory() as root, patch.object(campaign, 'ProcessTree', side_effect=RuntimeError('owned launch refused')):
            with self.assertRaisesRegex(RuntimeError, 'owned launch refused'):
                qualify(invoke, root, '1.0.2', executable=sys.executable, environment={})
            self.assertEqual(list(Path(root).iterdir()), [])
        self.assertFalse(Path(environments[0]['HOME']).exists())
        if sys.platform == 'linux':
            endpoint = str(Path(environments[0]['XDG_CACHE_HOME']) / 'insonic' / 'runtime' / ('a' * 32 + '.sock'))
            self.assertLessEqual(len(endpoint.encode('utf-8')), 100)

    def test_existing_fixture_root_is_never_removed(self):
        with tempfile.TemporaryDirectory() as root:
            directory = Path(root) / 'terminal fixtures café'
            directory.mkdir()
            sentinel = directory / 'owner.txt'
            sentinel.write_text('unchanged', encoding='utf-8')
            with self.assertRaises(FileExistsError):
                qualify(lambda *_a, **_k: {}, root, '1.0.2', executable=sys.executable, environment={})
            self.assertEqual(sentinel.read_text(encoding='utf-8'), 'unchanged')

    def test_elected_process_policy_cannot_silently_fall_back_for_terminals(self):
        with tempfile.TemporaryDirectory() as root:
            with self.assertRaisesRegex(ValueError, 'does not support owned terminals'):
                qualify(lambda *_a, **_k: {}, root, '1.0.2', executable=sys.executable,
                        environment={}, process_factory=object())
            self.assertEqual(list(Path(root).iterdir()), [])


if __name__ == '__main__':
    unittest.main()
