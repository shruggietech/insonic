#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
import copy
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import Mock
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from cli_corpus import load_corpus, installed_authority, Coverage, example_outcome, finish_server

HELP = '''Usage: insonic media list [--workspace DIRECTORY]
Global options:
  --workspace VALUE; Meaning: Library. Status: optional global. Repeat: once.
  --request-id VALUE; Meaning: Correlation. Status: optional global. Repeat: once.
  --json; Meaning: Machine. Status: optional global. Repeat: once.
  --help; Meaning: Help. Status: optional global. Repeat: once.
  -h; Meaning: Help. Status: optional global. Repeat: once.
Command options:
  --limit VALUE; Meaning: Page. Status: optional. Repeat: once.; range: 1..100
  --speaker VALUE; Meaning: Speaker. Status: optional. Repeat: repeatable.; repeatable
Diagnostic controls: details.
'''

class CorpusTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'fixture.py').write_text('def make(): return b"fixture"\n', encoding='utf-8')
        self.value = {'kind': 'cli-qualification-corpus', 'version': 1,
            'fixtures': [{'id': 'tone', 'owners': [{'path': 'fixture.py', 'symbol': 'make'}], 'license': 'Apache-2.0', 'provenance': 'Constructed.', 'expectations': ['Exact bytes.']}],
            'scenarios': [{'id': 'core', 'owner': {'path': 'fixture.py', 'symbol': 'make'}, 'fixtures': ['tone']}]}
    def load(self, value=None):
        p = self.root / 'corpus.json'
        p.write_text(json.dumps(self.value if value is None else value), encoding='utf-8')
        return load_corpus(self.root, p)
    def test_valid_inventory_and_pinned_bytes(self):
        self.value['fixtures'][0]['owners'][0]['sha256'] = hashlib.sha256((self.root / 'fixture.py').read_bytes()).hexdigest()
        self.assertEqual(self.load()['version'], 1)
        (self.root / 'fixture.py').write_bytes(b'changed')
        with self.assertRaises(ValueError): self.load()
    def test_rejects_metadata_and_reference_drift(self):
        mutations = [lambda x: x.update(version=2), lambda x: x.update(extra=True),
            lambda x: x['fixtures'].append(copy.deepcopy(x['fixtures'][0])),
            lambda x: x['scenarios'][0]['fixtures'].append('unknown'),
            lambda x: x['fixtures'][0].update(id='x'*65),
            lambda x: x['fixtures'][0].update(provenance='x'*513),
            lambda x: x['fixtures'][0]['owners'][0].update(path='../escape'),
            lambda x: x['fixtures'][0]['owners'][0].update(path='C:/private'),
            lambda x: x['fixtures'][0]['owners'][0].update(symbol='missing')]
        for change in mutations:
            value=copy.deepcopy(self.value);change(value)
            with self.subTest(value=value), self.assertRaises(ValueError): self.load(value)
    def test_bounds_duplicate_members_and_bom(self):
        p=self.root/'corpus.json'
        for data in [b' '*65537, b'{"version":1,"version":1}', b'\xef\xbb\xbf{}']:
            p.write_bytes(data)
            with self.assertRaises(ValueError): load_corpus(self.root,p)
        for name,count in [('fixtures',101),('scenarios',33)]:
            value=copy.deepcopy(self.value);value[name]*=count
            with self.assertRaises(ValueError): self.load(value)
    def test_committed_catalog_and_constructed_pcm(self):
        repo=Path(__file__).resolve().parents[1]
        self.assertGreaterEqual(len(load_corpus(repo)['fixtures']),8)
        from cli_media_qualification import pcm_fixture
        target=self.root/'résumé 文.wav';pcm_fixture(target,1)
        import wave
        with wave.open(str(target),'rb') as audio:
            self.assertEqual((audio.getnchannels(),audio.getframerate(),audio.getnframes()),(1,16000,16000))

class CoverageTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.exe=Path(self.temp.name)/'insonic';self.exe.write_bytes(b'qualified image')
    def recorder(self): return Coverage({'media list':HELP},self.exe,'1.1.0')
    def test_derive_authority_and_reject_malformed_help(self):
        flags=installed_authority({'media list':HELP})['media list']
        self.assertTrue(flags['--workspace']['value'])
        self.assertTrue(flags['--speaker']['repeatable'])
        for text in [HELP.replace('Command options:', 'Missing:'), HELP.replace('--limit VALUE','--limit VALUE\n  --limit VALUE')]:
            with self.assertRaises(ValueError): installed_authority({'media list':text})
    def test_values_are_discarded_and_help_cannot_claim_success(self):
        c=self.recorder()
        c.observe(['--workspace','C:/private token','media','list','--speaker=secret','--speaker','another','--json'],0)
        c.observe(['media','list','--help'],0)
        value=c.finish({})
        text=json.dumps(value)
        for secret in ['C:/private token','secret','another']: self.assertNotIn(secret,text)
        row=value['commands'][0]
        self.assertEqual(row['status'],'invocation-only')
        self.assertEqual(row['options']['--speaker']['invocations'],1)
        self.assertEqual(row['options']['--limit']['status'],'unverified')
    def test_unknown_accepted_options_fail_but_parser_rejections_are_separate(self):
        c=self.recorder()
        c.observe(['media','list','--unknown'],2)
        self.assertEqual(c.finish({})['commands'][0]['parser_rejections'],1)
        with self.assertRaises(ValueError): c.observe(['media','list','--unknown'],0)
        c.observe(['media','list','--','--not-an-option'],0)
        self.assertEqual(c.finish({})['controls']['literal-separator'],1)
    def test_identity_and_outcome_cannot_be_invented(self):
        with self.assertRaises(ValueError): Coverage({'media list':HELP},self.exe,'1.1.0',{'scope':'package','inventory_sha256':'0'*64})
        c=self.recorder()
        with self.assertRaises(ValueError): c.assert_example({'command':'unknown','example':[],'outcome':'asserted-success','assertion':'materialized-bytes'})
        c.observe(['media','list','--limit','1'],0,scenario='examples')
        c.assert_example({'command':'media list','example':['media','list','--limit','1'],'outcome':'asserted-success','assertion':'materialized-bytes'})
        self.assertEqual(c.finish({})['commands'][0]['status'],'asserted-success')
    def test_scenario_attribution_and_identity_binding(self):
        c=self.recorder()
        c.observe(['media','list','--limit','1'],0,scenario='query')
        c.observe(['media','list'],1,scenario='provider')
        row=c.finish({})['commands'][0]
        self.assertEqual(row['scenarios'],['provider','query'])
        self.assertEqual(row['options']['--limit']['scenarios'],['query'])
        with self.assertRaises(ValueError):c.observe(['media','list'],0,scenario='missing')
        identity={'scope':'package','archive_sha256':'a'*64,'inventory_sha256':'b'*64,'executable_sha256':hashlib.sha256(self.exe.read_bytes()).hexdigest()}
        self.assertEqual(Coverage({'media list':HELP},self.exe,'1.1.0',identity).finish({})['identity']['archive_sha256'],'a'*64)
        identity['executable_sha256']='0'*64
        with self.assertRaises(ValueError):Coverage({'media list':HELP},self.exe,'1.1.0',identity)

    def test_failed_observations_retain_only_schema_categories(self):
        c=self.recorder()
        c.observe(['media','list'],4,failure={'code':'unavailable','reason':'provider_timeout','message':'secret output'})
        c.observe(['media','list'],4,failure={'code':'unavailable','reason':'provider_auth'})
        value=c.finish({})
        self.assertEqual({x['reason'] for x in value['commands'][0]['failures']},{'provider_timeout','provider_auth'})
        self.assertNotIn('secret output',json.dumps(value))
        with self.assertRaises(ValueError):c.observe(['media','list'],4,failure={'code':'owner-secret'})

    def test_authority_building_observations_are_replayed(self):
        from cli_contract_qualification import qualify_read_only, PARSER_REJECTIONS
        observations=[]
        def invoke(args,**kwargs):
            if kwargs.get('expected')==2:return {'error':{'code':'invalid_request','message':'not retained'}}
            return {'result':{'help':HELP if len(args)>1 else 'Commands: list'}}
        pages=qualify_read_only(invoke,[['media','list']],self.exe.parent,self.exe.parent/'missing',observations=observations)
        c=Coverage(pages,self.exe,'1.1.0')
        for args,code,failure,help_only in observations:c.observe(args,code,failure=failure,help_only=help_only)
        value=c.finish({});row=value['commands'][0]
        self.assertEqual(value['controls']['help-only'],4)
        self.assertEqual(value['controls']['parser-rejections'],len(PARSER_REJECTIONS))
        self.assertEqual(row['options']['--help']['status'],'help-only')
        self.assertEqual(row['options']['-h']['status'],'help-only')
        self.assertNotIn(str(self.exe.parent),json.dumps(observations))

    def test_observation_bound(self):
        c=self.recorder();c.total=20000
        with self.assertRaises(ValueError):c.observe(['media','list'],0)

class OutcomeCleanupTests(unittest.TestCase):
    def test_no_immediate_success_without_assertion(self):
        self.assertEqual(example_outcome({'any':'result'},None)['outcome'],'invocation-only')
        self.assertEqual(example_outcome({},None,materialized=True)['outcome'],'asserted-success')
        self.assertEqual(example_outcome({},None,completed={'result':{'state':'succeeded'}})['assertion'],'durable-completion')
        failure=example_outcome({},None,completed={'error':{'code':'not_found'},'result':{'state':'failed'}})
        self.assertEqual(failure['outcome'],'refusal-observed')
        self.assertEqual(failure['failure_code'],'not_found')
        with self.assertRaises(ValueError):example_outcome({},None,completed={'result':{'state':'running'}})
    def test_thread_cleanup_must_finish_even_after_shutdown_error(self):
        server=Mock();thread=Mock();thread.is_alive.return_value=False
        finish_server(server,thread);thread.join.assert_called_once_with(timeout=30)
        thread.is_alive.return_value=True
        with self.assertRaises(ValueError):finish_server(server,thread)
        server.shutdown.side_effect=RuntimeError('primary')
        thread.is_alive.return_value=True
        with self.assertRaisesRegex(ValueError,'did not stop') as caught:finish_server(server,thread)
        self.assertIsInstance(caught.exception.__cause__,RuntimeError)
        thread.is_alive.return_value=False
        with self.assertRaises(RuntimeError):finish_server(server,thread)
        self.assertEqual(server.server_close.call_count,4)

if __name__=='__main__':unittest.main()
