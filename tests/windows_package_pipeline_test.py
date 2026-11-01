# SPDX-License-Identifier: Apache-2.0
from copy import deepcopy
import importlib.util
from pathlib import Path
import tempfile
import sys
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'scripts'))
spec=importlib.util.spec_from_file_location('windows_pipeline_fixture',ROOT/'scripts/windows_package_pipeline.py')
pipeline=importlib.util.module_from_spec(spec);spec.loader.exec_module(pipeline)

class WindowsPipelineTests(unittest.TestCase):
    def fixture(self):
        prefix='insonic_1.1.0_windows_amd64'
        initial={'schema_version':'1.1.0','revision':'a'*40,'source_dirty':False,'platform':'windows_amd64','variant':'desktop',
                 'dirname':prefix+'_desktop','archive':prefix+'_desktop.zip','archive_sha256':'b'*64,'archive_size_bytes':12,
                 'signing':{'status':'unconfigured'},'distribution':{'source_complete':True}}
        base={'revision':initial['revision'],'source_dirty':False,'payload_receipt':deepcopy(initial),'payload_archive':initial['archive'],
              'payload_archive_sha256':initial['archive_sha256'],'artifact':prefix+'_installer.msi','artifact_sha256':'c'*64,'artifact_size_bytes':14,
              'publication_eligible':False,'payload_qualification':'pending','qualification':{key:'pending' for key in pipeline.m.QUALIFICATION_KEYS},
              'setup':{'artifact':prefix+'_setup.exe','artifact_sha256':'d'*64,'artifact_size_bytes':16,'qualification':{key:'pending' for key in pipeline.windows_setup.QUALIFICATION_KEYS}}}
        identity=pipeline.bindings(base,initial)
        proofs={scope:{'kind':'windows-qualification-proof','scope':scope,'identity':deepcopy(identity),
                       'checks':{key:'passed' for key in ({'archive'} if scope=='archive' else pipeline.GATES[scope])}}
                for scope in pipeline.SCOPES}
        archive=deepcopy(initial);archive['native_webview']='passed'
        return base,initial,archive,proofs

    def test_every_scope_joins_the_original_three_assets_without_rewriting_inputs(self):
        base,initial,archive,proofs=self.fixture();before=deepcopy(base)
        final=pipeline.combine(base,initial,archive,proofs)
        self.assertTrue(final['publication_eligible']);self.assertEqual(final['payload_qualification'],'passed')
        self.assertEqual(final['payload_receipt'],archive)
        self.assertEqual(set(final['qualification']),set(pipeline.m.QUALIFICATION_KEYS))
        self.assertTrue(all(value=='passed' for value in final['qualification'].values()))
        self.assertTrue(all(value=='passed' for value in final['setup']['qualification'].values()))
        self.assertEqual(final['artifact_sha256'],base['artifact_sha256']);self.assertEqual(final['setup']['artifact_sha256'],base['setup']['artifact_sha256'])
        self.assertEqual(base,before)

    def test_any_missing_scope_or_failed_gate_prevents_eligibility(self):
        for scope in pipeline.SCOPES:
            for failure in ('missing','failed','missing-gate'):
                base,initial,archive,proofs=self.fixture()
                if failure=='missing':proofs.pop(scope)
                else:
                    key=next(iter(proofs[scope]['checks']))
                    if failure=='failed':proofs[scope]['checks'][key]='failed'
                    else:proofs[scope]['checks'].pop(key)
                with self.subTest(scope=scope,failure=failure),self.assertRaises(ValueError):pipeline.combine(base,initial,archive,proofs)
                self.assertFalse(base['publication_eligible'])

    def test_other_revision_archive_msi_or_setup_proof_is_rejected(self):
        for field in ('revision','archive','msi','setup'):
            base,initial,archive,proofs=self.fixture();identity=proofs['setup']['identity']
            if field=='revision':identity[field]='e'*40
            else:identity['sources'][field]['sha256']='e'*64
            with self.subTest(field=field),self.assertRaises(ValueError):pipeline.combine(base,initial,archive,proofs)

    def test_archive_cannot_rewrite_original_signing_distribution_or_digest(self):
        for field in ('signing','distribution','archive_sha256'):
            base,initial,archive,proofs=self.fixture();archive[field]='changed'
            with self.subTest(field=field),self.assertRaises(ValueError):pipeline.combine(base,initial,archive,proofs)

    def test_evidence_transfer_detects_modified_bytes_and_escaping_paths(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);item=root/'capture.json';item.write_bytes(b'controlled native evidence')
            files=pipeline.evidence_files(root);pipeline.validate_files(root,files)
            item.write_bytes(b'changed native evidence')
            with self.assertRaisesRegex(ValueError,'bytes differ'):pipeline.validate_files(root,files)
            for name in ('../outside','/outside','C:/outside','folder\\outside'):
                with self.subTest(name=name),self.assertRaises(ValueError):pipeline.validate_files(root,[{'path':name}])

    def test_evidence_transfer_rejects_unrecorded_captures(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);(root/'receipt.json').write_bytes(b'original')
            files=pipeline.evidence_files(root);(root/'extra.png').write_bytes(b'unrecorded')
            with self.assertRaisesRegex(ValueError,'unrecorded'):pipeline.validate_files(root,files)

    def test_archive_scope_consumes_the_receipt_written_by_smoke(self):
        base,initial,completed,_=self.fixture()
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary)
            def smoke(*args):pipeline.write(root/'build/native/package-desktop-receipt.json',completed)
            package=SimpleNamespace(smoke=smoke)
            restore=SimpleNamespace(restore_payload=Mock())
            release=SimpleNamespace(verify_qualification=Mock())
            with patch.object(pipeline,'ROOT',root),patch.object(pipeline.m,'disposable_runner'),patch.object(pipeline,'restore_inputs',return_value=(base,initial,pipeline.bindings(base,initial))),patch.object(pipeline,'load',side_effect=[package,restore,release]):
                proof=pipeline.qualify('archive')
            self.assertEqual(proof['checks'],{'archive':'passed'})
            release.verify_qualification.assert_called_once_with(completed)
            self.assertEqual(pipeline.read(root/'build/native/scope-proof.json'),proof)

if __name__=='__main__':unittest.main()
