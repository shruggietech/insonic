# SPDX-License-Identifier: Apache-2.0
"""Exercise the two installed campaigns' disposable backend elections."""
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import types
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
spec = importlib.util.spec_from_file_location('answer_backends', Path(__file__).resolve().parents[1] / 'scripts/qualify-answer-backends.py')
backend = importlib.util.module_from_spec(spec)
spec.loader.exec_module(backend)


class BackendCampaignTests(unittest.TestCase):
    def test_campaigns_provision_independent_namespaces_and_retain_both_receipts(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'scripts').mkdir()
            (root / 'build/native').mkdir(parents=True)
            (root / 'VERSION').write_text('1.2.0\n', encoding='utf-8')
            (root / 'build/native/media-tools.json').write_text('{}\n', encoding='utf-8')
            (root / 'scripts/qualify-answers.py').write_text(
                'def campaign(*args, **kwargs):\n    return {"passed": True, "model_quality": "not-measured"}\n', encoding='utf-8')
            executable, cueson, output = root / 'cli', root / 'cueson', root / 'receipt.json'
            executable.write_bytes(b'fixture')
            cueson.write_bytes(b'fixture cueson')
            created, elections = set(), []

            def request(req, **kwargs):
                if req.get_method() == 'PUT':
                    key = ('storage', req.full_url)
                else:
                    key = ('graph', req.full_url, json.loads(req.data)['command'])
                self.assertNotIn(key, created, 'Second campaign reused a workspace-owned backend namespace')
                created.add(key)
                class Response:
                    status = 201
                    def __enter__(self): return self
                    def __exit__(self, *_args): pass
                return Response()

            def provision(_factory, argv, **kwargs):
                election = json.loads(Path(argv[argv.index('--input') + 1]).read_text(encoding='utf-8'))
                elections.append(election)
                workspace = Path(argv[4])
                (workspace / '.insonic').mkdir(parents=True)
                self.assertEqual(election['profiles']['catalog']['adapter_id'], 'postgresql')
                self.assertEqual(election['profiles']['storage']['adapter_id'], 's3')
                self.assertEqual(election['profiles']['graph']['adapter_id'], 'arcadedb')
                return b'{}', b'', 0, None

            def speaker(_exe, workspace, _folder, _env, _version):
                self.assertEqual(len(elections), 2)
                self.assertEqual((workspace / '.insonic/media-tools.json').read_text(encoding='utf-8'), '{}\n')
                return {'state': 'passed', 'restart': {'state': 'passed'}}

            environment = {'INSONIC_FIXTURE_POSTGRES': 'postgres://fixture:password@localhost:5432/fixture',
                           'INSONIC_FIXTURE_S3': 'localhost:9090', 'INSONIC_FIXTURE_ARCADE': 'http://localhost:2480'}
            with patch.object(backend, '__file__', str(root / 'scripts/qualify-answer-backends.py')), \
                 patch.dict(backend.os.environ, environment), \
                 patch.object(backend.native, 'native_env', return_value={'CUESON_EXECUTABLE': str(cueson)}), \
                 patch.object(backend.urllib.request, 'urlopen', side_effect=request), \
                 patch.object(backend, 'capture_invocation', side_effect=provision), \
                 patch.dict(sys.modules, {'cli_speaker_workflow_qualification': types.SimpleNamespace(qualify_backend=speaker)}), \
                 patch.object(sys, 'argv', ['qualify-answer-backends.py', '--executable', str(executable), '--output', str(output)]):
                backend.main()
            first, second = [p['profiles'] for p in elections]
            for kind, field in [('catalog', 'schema'), ('storage', 'bucket'), ('graph', 'database')]:
                self.assertNotEqual(first[kind]['configuration'][field], second[kind]['configuration'][field])
            result = json.loads(output.read_text(encoding='utf-8'))
            self.assertTrue(result['passed'])
            self.assertEqual(result['speaker_workflow']['state'], 'passed')
            self.assertEqual(result['speaker_workflow']['restart']['state'], 'passed')


if __name__ == '__main__':
    unittest.main()
