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
    def test_unchecked_shortcuts_normalize_before_transfer_and_costing(self):
        actions={r[0]:r for r in self.model['tables']['CustomAction']}
        self.assertEqual(actions['NormalizeStartMenu'][1:],[51,'START_MENU','0']);self.assertEqual(actions['NormalizeDesktopShortcut'][1:],[51,'DESKTOP_SHORTCUT','0'])
        sequence={r[0]:r[2] for r in self.model['tables']['InstallExecuteSequence']}
        self.assertLess(sequence['NormalizeStartMenu'],sequence['CostInitialize']);self.assertLess(sequence['NormalizeDesktopShortcut'],sequence['CostInitialize'])
        events=[r for r in self.model['tables']['ControlEvent'] if r[:2]==['Ready','Next']]
        finish=next(r[5] for r in events if r[2]=='EndDialog')
        for action in ('NormalizeStartMenu','NormalizeDesktopShortcut'):
            self.assertLess(next(r[5] for r in events if r[2]=='DoAction' and r[3]==action),finish)
        self.assertTrue(any(r[2:5]==['[REINSTALL]','Core','Installed AND NOT REINSTALL'] for r in events))
        spec=importlib.util.spec_from_file_location('qualify_preserve_shortcuts',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        self.assertNotIn('START_MENU=',module.installer_properties([]));self.assertNotIn('DESKTOP_SHORTCUT=',module.installer_properties([]))
        self.assertIn('START_MENU=""',module.installer_properties(['START_MENU=']))
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
            for function,args in [(module.token_info,()),(module.restricted_context,(None,None,None)),(module.restricted_worker,(None,None,None)),(module.cancellation,(None,None,None)),(module.cancellation_worker,(None,None,None)),(module.prepare_launcher_fixture,(None,None)),(module.elected_launcher_context,(None,)),(module.linked_token_evidence,()),(module.service_diagnostic,()),(module.native_installer_worker,(None,)),(module.native_installer_dll,(None,)),(module.run_installer,(['/i','unused'],None))]:
                with self.assertRaisesRegex(RuntimeError,'disposable'):function(*args)
        self.assertIn('cancellation_rollback',m.QUALIFICATION_KEYS);self.assertIn('user_context',m.QUALIFICATION_KEYS)
    def test_native_capture_requires_verified_clean_nonempty_source(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('capture_installer_contract',ROOT/'scripts/capture-installer-ui.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.object(m,'disposable_runner',return_value=None), patch.object(module.subprocess,'CREATE_NO_WINDOW',0,create=True):
            with self.assertRaisesRegex(ValueError,'clean source'):module.source_identity({'source_dirty':True,'revision':'a'*40})
            replies=[SimpleNamespace(returncode=0,stdout=('a'*40+'\n').encode()),SimpleNamespace(returncode=0,stdout=b'')]
            with patch.object(module.subprocess,'run',side_effect=replies):source=module.source_identity({'source_dirty':False,'revision':'a'*40})
            self.assertIs(source['dirty'],False);self.assertEqual(len(source['files']),4)
            self.assertTrue(all(item['sha256']==m.sha(ROOT/item['path']) for item in source['files']))
            replies=[SimpleNamespace(returncode=0,stdout=('b'*40+'\n').encode())]
            with patch.object(module.subprocess,'run',side_effect=replies):
                with self.assertRaisesRegex(ValueError,'checkout differs'):module.source_identity({'source_dirty':False,'revision':'a'*40})
    def test_native_installer_property_language_preserves_unicode_and_spaces(self):
        spec=importlib.util.spec_from_file_location('qualify_msi_properties',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        directory='C:\\fixture spaces é\\'
        text=module.installer_properties(['INSTALLDIR='+directory,'ADDLOCAL=Core,CLI','LAUNCH=1'])
        self.assertIn('INSTALLDIR="'+directory+'"',text);self.assertIn('ADDLOCAL="Core,CLI"',text);self.assertIn('LAUNCH="0"',text)
        for invalid in ['missing-equals','bad-key=value','INSTALLDIR=bad\nvalue','INSTALLDIR=bad\0value']:
            with self.assertRaises(ValueError):module.installer_properties([invalid])
    def test_native_worker_records_actual_api_result_and_preserves_action(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_msi_worker_contract',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        calls=[]
        fake=SimpleNamespace(MsiInstallProductW=lambda *args:calls.append(('install',args)) or 1603,MsiConfigureProductExW=lambda *args:calls.append(('configure',args)) or 0)
        for action,expected_code,state in [('/i',1603,None),('/x',0,2),('/fa',0,5)]:
            log=self.path/(action[1:]+'.log');request=self.path/'native-request.json';request.write_text(json.dumps({'action':action,'subject':'controlled-fixture','properties':['INSTALLDIR=C:\\space é'],'log':str(log)}),encoding='utf-8')
            with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'native_installer_dll',return_value=fake),patch.object(module,'service_diagnostic',return_value={'pid':0,'state':1}):module.native_installer_worker(request)
            evidence=json.loads(log.with_suffix('.worker.json').read_text(encoding='utf-8'))
            self.assertEqual(evidence['phase'],'completed');self.assertEqual(evidence['exit_code'],expected_code);self.assertIs(evidence['captured_pipes'],False)
            if state is not None:self.assertEqual(calls[-1][1][2],state)
            if action=='/fa':self.assertIn('REINSTALLMODE="amus"',calls[-1][1][-1])
    def test_bad_compound_and_cabinet_rejected(self):
        p=self.path/'invalid.msi';p.write_bytes(b'not an MSI')
        with self.assertRaises(ValueError):m.CompoundFile(p)
        with self.assertRaises(ValueError):m.cabinet_members(b'not a cabinet')
    def test_restore_verifies_before_replacing_owned_payload(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('package_msi_restore_contract',ROOT/'scripts/package-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        name='insonic_1.0.1_windows_amd64_desktop';base=self.path/'build/packages';target=base/name;target.mkdir(parents=True);sentinel=target/'prior-owned-payload';sentinel.write_bytes(b'keep until verified')
        receipt_path=self.path/'restore.json';receipt={'dirname':name,'archive':name+'.zip','platform':'windows_amd64','variant':'desktop'};receipt_path.write_text(json.dumps(receipt),encoding='utf-8')
        def reject_archive(*args):raise ValueError('tampered archive')
        package=SimpleNamespace(VERSION='1.0.1',verify_archive_receipt=reject_archive)
        with patch.object(module,'ROOT',self.path),patch.object(module,'load',return_value=package):
            with self.assertRaisesRegex(ValueError,'tampered'):module.restore_payload(receipt_path)
        self.assertEqual(sentinel.read_bytes(),b'keep until verified')
        receipt['dirname']='../unrelated';receipt_path.write_text(json.dumps(receipt),encoding='utf-8')
        with patch.object(module,'ROOT',self.path),patch.object(module,'load',return_value=package):
            with self.assertRaisesRegex(ValueError,'exact current'):module.restore_payload(receipt_path)
    @unittest.skipUnless(os.name=='nt','native database authoring Windows only, no UI')
    def test_native_clone_reinserts_composite_upgrade_keys(self):
        spec=importlib.util.spec_from_file_location('qualify_clone_database',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        original=self.path/'original.msi';m.author(original,self.path,self.model)
        target=self.path/'version-fixture.msi';upgrade=m.guid('native-qualification-clone-test');module.clone(original,target,'1.0.2',upgrade)
        streams=m.CompoundFile(target).streams;properties=dict(m.table_rows(streams,'Property'))
        self.assertEqual(properties['ProductCode'],m.product_code('1.0.2',upgrade));self.assertEqual(properties['ProductVersion'],'1.0.2');self.assertEqual(properties['UpgradeCode'],upgrade)
        relations={row[6]:row for row in m.table_rows(streams,'Upgrade')}
        self.assertEqual(relations['OLDPRODUCTS'],[upgrade,None,'1.0.2',None,1,None,'OLDPRODUCTS'])
        self.assertEqual(relations['NEWPRODUCTS'],[upgrade,'1.0.2',None,None,2,None,'NEWPRODUCTS'])
        self.assertEqual(m.cabinet_members(streams['payload.cab']),m.cabinet_members(m.CompoundFile(original).streams['payload.cab']))
    @unittest.skipUnless(os.name=='nt','native database authoring Windows only, no UI')
    def test_signed_payload_and_unsigned_msi_are_independent(self):
        # Disposition regression fixture. Archive signing verification remains the
        # archive verifier's responsibility; no certificate proof is fabricated.
        self.inv['signing']={'configured':True,'verified':True,'status':'signed'}
        (self.path/'package-inventory.json').write_bytes(m.inventory_bytes(self.inv))
        inv_sha=m.sha(self.path/'package-inventory.json')
        model=m.installer_model(self.inv,inv_sha,(ROOT/'LICENSE').read_text(encoding='utf-8'))
        artifact=self.path/'insonic_1.0.1_windows_amd64_installer.msi';m.author(artifact,self.path,model)
        payload={k:self.inv[k] for k in ('revision','source_dirty','platform','schema_version','distribution','signing')}
        payload.update({'archive':'desktop.zip','archive_sha256':'b'*64,'inventory_sha256':inv_sha})
        receipt={k:self.inv[k] for k in ('revision','source_dirty','platform','schema_version','distribution')}
        receipt.update({'kind':'windows-installer-receipt','format':'msi','variant':'selectable','product_version':'1.0.1','documentation_version':'1.0.1','signing':dict(m.UNSIGNED_SIGNING),'artifact':artifact.name,'artifact_sha256':m.sha(artifact),'artifact_size_bytes':artifact.stat().st_size,'payload_receipt':payload,'payload_inventory':self.inv,'payload_archive':'desktop.zip','payload_archive_sha256':'b'*64,'qualification':{k:'passed' for k in m.QUALIFICATION_KEYS},'msi_product_version':'1.0.1','product_code':model['product_code'],'upgrade_code':model['upgrade_code'],'scope':'per-user','license_sha256':m.sha(ROOT/'LICENSE'),'icon_sha256':m.sha(ROOT/m.ICON)})
        self.assertEqual(m.verify_installer(artifact,receipt)['inventory']['signing'],self.inv['signing'])
        receipt['signing']=dict(self.inv['signing'])
        with self.assertRaisesRegex(ValueError,'independently unsigned'):m.verify_installer(artifact,receipt)
    @unittest.skipUnless(os.name=='nt','native database authoring Windows only, no UI')
    def test_native_actual_tables_and_cabinet_independently_verified(self):
        path=self.path/'fixture.msi';m.author(path,self.path,self.model);result=m.verify_model_bytes(path,self.model);self.assertEqual(len(result['members']),len(self.model['payload']))
        db=m.NativeDatabase(path,write=True)
        try:
            q=m.quote;db.execute('UPDATE '+q('CustomAction')+' SET '+q('Target')+'=? WHERE '+q('Action')+'=?',['--cli','LaunchDesktop']);db.check(db.dll.MsiDatabaseCommit(db.handle))
        finally:db.close()
        with self.assertRaisesRegex(ValueError,'CustomAction'):m.verify_model_bytes(path,self.model)
if __name__=='__main__':unittest.main()
