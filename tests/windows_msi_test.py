# SPDX-License-Identifier: Apache-2.0
import importlib.util
import json
import os
from pathlib import Path
import struct
import sys
import tempfile
import unittest
from unittest.mock import patch
ROOT=Path(__file__).resolve().parent.parent
sys.path.insert(0,str(ROOT/'scripts'))
import windows_msi as m

def fixture(directory):
    data={'insonic.exe':b'cli','insonic-desktop.exe':b'desktop','insonic-launcher.exe':b'launcher','insonic-session.cmd':b'session','LICENSE':(ROOT/'LICENSE').read_bytes(),'NOTICE':b'notice','insonic-companions.json':b'{}','help/index.html':b'help','sources/source-manifest.json':b'{}'}
    files=[]
    for name,value in data.items():
        p=directory/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes(value);files.append({'path':name,'sha256':m.digest(value),'size_bytes':len(value)})
    inv={'kind':'native-package-inventory','schema_version':'1.0.1','platform':'windows_amd64','variant':'desktop','revision':'a'*40,'source_dirty':False,'signing':{'configured':False,'verified':False,'status':'unconfigured'},'distribution':{'source_complete':True},'files':files}
    (directory/'package-inventory.json').write_bytes(m.inventory_bytes(inv))
    return inv,m.installer_model(inv,m.digest(m.inventory_bytes(inv)),(ROOT/'LICENSE').read_text(encoding='utf-8'))
class InstallerModelTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(prefix='s027-msi-test-');self.path=Path(self.temp.name);self.inv,self.model=fixture(self.path)
    def tearDown(self):self.temp.cleanup()
    def test_client_dependency_and_owned_marker(self):
        items={x['path']:x for x in self.model['payload']}
        self.assertEqual(items['insonic-desktop.exe']['feature'],'Desktop');self.assertEqual(items['insonic.exe']['feature'],'Core');self.assertIn(m.OWNERSHIP,items);self.assertNotIn(m.OWNERSHIP,{x['path'] for x in self.inv['files']})
    def test_license_initially_unchecked_and_exact(self):
        self.assertNotIn('LICENSE_ACCEPTED',dict(self.model['tables']['Property']));self.assertEqual(self.model['license_sha256'],m.sha(ROOT/'LICENSE'))
    def test_three_selection_contract(self):
        cond=dict(self.model['tables']['LaunchCondition']);self.assertIn('REMOVE="ALL" OR (&Desktop=3 OR &CLI=3)',cond)
        self.assertEqual({r[0] for r in self.model['tables']['Feature']},{'Core','Desktop','CLI'})
        self.assertIn('REMOVE="ALL" OR &Core=3',cond)
    def test_per_user_no_machine_integration(self):
        self.assertNotIn('ALLUSERS',dict(self.model['tables']['Property']));self.assertTrue(all(r[1]==1 for r in self.model['tables']['Registry']));self.assertNotIn('Environment',self.model['tables'])
    def test_upgrade_rollback_and_downgrade(self):
        sequence={r[0]:r[2] for r in self.model['tables']['InstallExecuteSequence']};self.assertLess(sequence['InstallInitialize'],sequence['RemoveExistingProducts']);self.assertLess(sequence['RemoveExistingProducts'],sequence['InstallFinalize']);self.assertIn('NOT NEWPRODUCTS',dict(self.model['tables']['LaunchCondition']))
    def test_component_identity_stable_across_version(self):
        newer=m.installer_model(self.inv,m.digest(m.inventory_bytes(self.inv)),(ROOT/'LICENSE').read_text(),version='1.0.2');self.assertEqual(newer['tables']['Component'],self.model['tables']['Component']);self.assertNotEqual(newer['product_code'],self.model['product_code']);self.assertEqual(newer['upgrade_code'],self.model['upgrade_code'])
    def test_ui_controls_form_single_cycle(self):
        for dialog in self.model['tables']['Dialog']:
            controls={r[1]:r for r in self.model['tables']['Control'] if r[0]==dialog[0] and r[10]};start=dialog[7];current=start;seen=set()
            while current not in seen:seen.add(current);current=controls[current][10]
            self.assertEqual(current,start);self.assertEqual(seen,set(controls))
    def test_final_launch_fresh_interactive_only(self):
        for r in self.model['tables']['ControlEvent']:
            if r[2]=='DoAction' and r[3].startswith('Launch'):self.assertIn('UILevel=5',r[4]);self.assertIn('NOT Installed',r[4]);self.assertIn('NOT OLDPRODUCTS',r[4])
    def test_preflight_before_changes_and_no_restart_shutdown(self):
        sequence={r[0]:r[2] for r in self.model['tables']['InstallExecuteSequence']};self.assertLess(sequence['Preflight'],sequence['InstallInitialize']);self.assertEqual(dict(self.model['tables']['Property'])['MSIRESTARTMANAGERCONTROL'],'Disable')
        self.assertEqual(next(r[3] for r in self.model['tables']['CustomAction'] if r[0]=='Preflight'),'--preflight "[INSTALLDIR]." both')
    def test_webview_only_when_desktop_selected(self):
        c=[r for r in self.model['tables']['LaunchCondition'] if 'WEBVIEW_MACHINE' in r[0]][0];self.assertIn('NOT (&Desktop=3)',c[0])
    def test_paths_and_version_reject_wrong_scope(self):
        for p in ['../owner','C:/owner','folder\\owner','/owner','bad\nname']:
            with self.assertRaises(ValueError):m.relative(p)
        for v in ['1.0','1.0.1-beta','256.0.0','1.0.65536']:
            with self.assertRaises(ValueError):m.msi_version(v)
    def test_duplicate_payload_rejected(self):
        self.inv['files'].append(dict(self.inv['files'][0]))
        with self.assertRaises(ValueError):m.installer_model(self.inv,'a'*64,'license')
    def test_owner_host_ui_guard(self):
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):m.disposable_runner()
    def test_cancel_and_context_entrypoints_reject_owner_host_before_native_calls(self):
        spec=importlib.util.spec_from_file_location('qualify_msi_guard',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            for function,args in [(module.token_info,()),(module.restricted_context,(None,None,None)),(module.restricted_worker,(None,None,None)),(module.cancellation,(None,None,None)),(module.cancellation_worker,(None,None,None))]:
                with self.assertRaisesRegex(RuntimeError,'disposable'):function(*args)
        self.assertIn('cancellation_rollback',m.QUALIFICATION_KEYS);self.assertIn('user_context',m.QUALIFICATION_KEYS)
    def test_bad_compound_and_cabinet_rejected(self):
        p=self.path/'invalid.msi';p.write_bytes(b'not an MSI')
        with self.assertRaises(ValueError):m.CompoundFile(p)
        with self.assertRaises(ValueError):m.cabinet_members(b'not a cabinet')
    @unittest.skipUnless(os.name=='nt','native database authoring Windows only, no UI')
    def test_native_actual_tables_and_cabinet_independently_verified(self):
        path=self.path/'fixture.msi';m.author(path,self.path,self.model);result=m.verify_model_bytes(path,self.model);self.assertEqual(len(result['members']),len(self.model['payload']))
        db=m.NativeDatabase(path,write=True)
        try:
            q=m.quote;db.execute('UPDATE '+q('CustomAction')+' SET '+q('Target')+'=? WHERE '+q('Action')+'=?',['--cli','LaunchDesktop']);db.check(db.dll.MsiDatabaseCommit(db.handle))
        finally:db.close()
        with self.assertRaisesRegex(ValueError,'CustomAction'):m.verify_model_bytes(path,self.model)
if __name__=='__main__':unittest.main()
