# SPDX-License-Identifier: Apache-2.0
import copy
import importlib.util
from pathlib import Path
import sys
import unittest
import uuid

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'scripts'))

def load():
    spec=importlib.util.spec_from_file_location('storage_fixture',ROOT/'scripts/cli_storage_qualification.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    return module

def receipt(image='a'*64,platform='windows'):
    from cli_recovery_qualification import derived
    failed,recovered,sentinel=[str(uuid.uuid4()) for _ in range(3)]
    canonical={'id':derived(recovered,'canonical-0'),'digest':'b'*64,'size':101}
    report={'id':derived(recovered,'canonical-report-0'),'digest':'c'*64,'size':102}
    checks={k:True for k in ('source_unchanged','sentinel_unchanged','no_incomplete_admission','blocker_removed','canonical_verified','report_verified','metadata_provenance','date_provenance','idempotent_replay','prior_work_retained')}
    return {'format':'insonic-storage-failure-qualification-v1','platform':platform,'executable_sha256':image,
            'origin':'actual-filesystem-obstruction','physical_exhaustion':False,'cleanup':True,'fixture_removed':True,'runtime_natural_exit':True,
            'cases':{'canonical-publication':{'source_sha256':'d'*64,'source_size':1000,'sentinel_id':sentinel,'sentinel_sha256':'e'*64,
                     'blocker_sha256':'f'*64,'blocked_digest':'b'*64,'failed':{'work_id':failed,'state':'failed','partial':True,'item_state':'failed','error':'unavailable','stage':'canonical-publication'},
                     'recovered':{'work_id':recovered,'state':'succeeded','partial':False,'item_state':'admitted','entry_id':str(uuid.uuid4()),'canonical':canonical,'report':report},
                     'checks':checks}}}

class StorageQualificationTests(unittest.TestCase):
    def test_complete_bounded_actual_receipt(self):
        module=load();value=receipt()
        module.validate_receipt(value,'a'*64,'windows')
        value['platform']='linux';module.validate_receipt(value,'a'*64,'linux')

    def test_missing_mislabeled_and_stale_receipts_are_refused(self):
        module=load()
        mutations=[
            lambda r:r.pop('origin'),lambda r:r.update(origin='injected-ENOSPC'),lambda r:r.update(physical_exhaustion=True),
            lambda r:r.update(physical_exhaustion=0),lambda r:r.update(executable_sha256='0'*64),
            lambda r:r.update(platform='linux'),lambda r:r.update(runtime_natural_exit=False),lambda r:r.update(fixture_removed=False),
            lambda r:r['cases'].clear(),lambda r:r['cases']['canonical-publication']['checks'].pop('source_unchanged'),
            lambda r:r['cases']['canonical-publication']['checks'].update(sentinel_unchanged=False),
            lambda r:r['cases']['canonical-publication']['failed'].update(item_state='admitted'),
            lambda r:r['cases']['canonical-publication']['failed'].update(partial=False),
            lambda r:r['cases']['canonical-publication']['failed'].update(stage='prepare'),
            lambda r:r['cases']['canonical-publication']['recovered'].update(work_id=r['cases']['canonical-publication']['failed']['work_id']),
            lambda r:r['cases']['canonical-publication']['recovered']['canonical'].update(digest='0'*64),
            lambda r:r['cases']['canonical-publication']['recovered']['report'].update(id=str(uuid.uuid4())),
            lambda r:r['cases']['canonical-publication'].update(source_size=True),
        ]
        for mutation in mutations:
            value=receipt();mutation(value)
            with self.subTest(mutation=mutation),self.assertRaises(ValueError):module.validate_receipt(value,'a'*64,'windows')


    def test_blocker_removal_requires_unchanged_confined_authority(self):
        import hashlib
        import json
        import tempfile
        module=load()
        for mutation in ('none','path','digest','inode','bytes','profile'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as directory:
                root=Path(directory);library=root/'owned library';control=library/'.insonic';control.mkdir(parents=True)
                workspace,profile=str(uuid.uuid4()),str(uuid.uuid4())
                config={'workspace_id':workspace,'profiles':{'storage':{'profile_id':profile,'profile_revision':1,'adapter_id':'filesystem','configuration':{'root':'artifacts'}}}}
                (control/'workspace.json').write_text(json.dumps(config),encoding='utf8')
                target=control/'artifacts/objects'/workspace/profile/'1/sha256'/'aa'/('a'*64);target.parent.mkdir(parents=True);target.write_bytes(b'owned blocker')
                original=root/'unchanged source';original.write_bytes(b'original bytes')
                st=target.stat()
                record={'path':str(target),'digest':'a'*64,'device':st.st_dev,'inode':st.st_ino,'size':st.st_size,'sha256':hashlib.sha256(target.read_bytes()).hexdigest()}
                if mutation=='path':record['path']=str(original)
                if mutation=='digest':record['digest']='../escape'
                if mutation=='inode':record['inode']+=1
                if mutation=='bytes':target.write_bytes(b'changed bytes')
                if mutation=='profile':
                    config['profiles']['storage']['profile_id']='../../outside'
                    (control/'workspace.json').write_text(json.dumps(config),encoding='utf8')
                marker=root/'marker.json';marker.write_text(json.dumps(record),encoding='utf8')
                if mutation=='none':
                    module.remove_blocker(library,marker);self.assertFalse(target.exists())
                else:
                    with self.assertRaises(ValueError):module.remove_blocker(library,marker)
                    self.assertTrue(target.exists())
                self.assertEqual(original.read_bytes(),b'original bytes')

if __name__=='__main__':unittest.main()
