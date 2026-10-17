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

def versioned_pe_fixture():
    """One standard PE32+ RT_VERSION leaf; no executable code or native API."""
    data=bytearray(1024);data[:2]=b'MZ';struct.pack_into('<I',data,60,128);data[128:132]=b'PE\0\0'
    struct.pack_into('<HH',data,132,0x8664,1);struct.pack_into('<H',data,148,240)
    optional=152;struct.pack_into('<H',data,optional,0x20b);struct.pack_into('<I',data,optional+108,16);struct.pack_into('<II',data,optional+128,0x1000,512)
    section=optional+240;data[section:section+8]=b'.rsrc\0\0\0';struct.pack_into('<IIII',data,section+8,512,0x1000,512,512)
    for offset,identifier,next_offset in [(0,16,24),(24,1,48),(48,1033,72)]:
        struct.pack_into('<H',data,512+offset+14,1);struct.pack_into('<II',data,512+offset+16,identifier,next_offset|(0x80000000 if offset!=48 else 0))
    key='VS_VERSION_INFO\0'.encode('utf-16le');value=bytearray(92);struct.pack_into('<HHH',value,0,92,52,0);value[6:6+len(key)]=key
    struct.pack_into('<IIII',value,40,0xfeef04bd,0x10000,0x00010006,0x00000002)
    struct.pack_into('<IIII',data,512+72,0x1000+88,len(value),0,0);data[600:600+len(value)]=value
    return bytes(data)
class InstallerModelTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(prefix='s027-msi-test-');self.path=Path(self.temp.name);self.inv,self.model=fixture(self.path)
    def tearDown(self):self.temp.cleanup()
    def test_execution_progress_is_native_and_terminal_controls_match_outcome(self):
        tables=self.model['tables']
        sequence={r[0]:r[2] for r in tables['InstallUISequence']}
        self.assertLess(sequence['Progress'],sequence['ExecuteAction'])
        self.assertIn(['Progress','Bar','SetProgress','Progress'],tables['EventMapping'])
        self.assertIn(['Progress','Action','ActionText','Text'],tables['EventMapping'])
        dialog=next(r for r in tables['Dialog'] if r[0]=='Progress')
        self.assertFalse(dialog[5]&2)
        for name in ['Finish','Cancelled','FatalError']:
            controls=[r for r in tables['Control'] if r[0]==name and r[2]=='PushButton']
            self.assertEqual(len(controls),1)
            self.assertNotIn(controls[0][9],['Cancel','Next >'])
            row=next(r for r in tables['Dialog'] if r[0]==name)
            self.assertEqual(row[8],row[9])
            self.assertTrue(any(row[:5]==[name,controls[0][1],'EndDialog','Return','1'] for row in tables['ControlEvent']))
        self.assertNotIn('rollback',next(r[1] for r in tables['ActionText'] if r[0]=='RollbackCleanup').casefold())
        for action in ['Rollback','RollbackCleanup','GenerateScript']:
            self.assertEqual(next(r for r in tables['ActionText'] if r[0]==action)[2],'[1]')

    def test_maintenance_repair_does_not_recache_its_open_installer(self):
        events=[row for row in self.model['tables']['ControlEvent'] if row[:2]==['Maintenance','Repair']]
        self.assertEqual(next(row[3] for row in events if row[2]=='[REINSTALL]'),'ALL')
        mode=next(row[3] for row in events if row[2]=='[REINSTALLMODE]')
        self.assertEqual(mode,'omus')
        self.assertNotIn('v',mode)

    def test_documentation_completion_election_is_independent_and_fresh_only(self):
        tables=self.model['tables'];props=dict(tables['Property'])
        self.assertEqual(props['LAUNCH'],'1');self.assertNotIn('OPEN_DOCUMENTATION',props)
        self.assertIn(['OPEN_DOCUMENTATION','1'],tables['CheckBox'])
        action=next(r for r in tables['CustomAction'] if r[0]=='OpenDocumentation')
        self.assertEqual(action[1],82);self.assertEqual(action[3],'--documentation --interactive')
        events=[r for r in tables['ControlEvent'] if r[2]=='DoAction' and r[3]=='OpenDocumentation']
        self.assertEqual(len(events),1)
        for clause in ['OPEN_DOCUMENTATION=1','UILevel=5','NOT Installed','NOT OLDPRODUCTS']:
            self.assertIn(clause,events[0][4])

    def test_client_dependency_and_owned_marker(self):
        items={x['path']:x for x in self.model['payload']}
        self.assertEqual(items['insonic-desktop.exe']['feature'],'Desktop');self.assertEqual(items['insonic.exe']['feature'],'Core');self.assertIn(m.OWNERSHIP,items);self.assertNotIn(m.OWNERSHIP,{x['path'] for x in self.inv['files']})
    def test_removal_does_not_require_fresh_license_acceptance(self):
        condition=next(row[0] for row in self.model['tables']['LaunchCondition'] if row[1].startswith('Read and accept'))
        self.assertEqual(condition,'REMOVE="ALL" OR Installed OR LICENSE_ACCEPTED=1')

    def test_license_initially_unchecked_and_exact(self):
        self.assertNotIn('LICENSE_ACCEPTED',dict(self.model['tables']['Property']));self.assertEqual(self.model['license_sha256'],m.sha(ROOT/'LICENSE'))
    def test_three_selection_contract(self):
        cond=dict(self.model['tables']['LaunchCondition']);self.assertIn('REMOVE="ALL" OR ('+m.effective_feature_local('Desktop')+' OR '+m.effective_feature_local('CLI')+')',cond)
        self.assertEqual({r[0] for r in self.model['tables']['Feature']},{'Core','Desktop','CLI'})
        self.assertIn('REMOVE="ALL" OR '+m.effective_feature_local('Core'),cond)
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
    def test_saved_directory_is_only_a_non_overriding_fallback(self):
        tables=self.model['tables'];search=dict(tables['AppSearch'])
        self.assertEqual(search['SAVED_INSTALLDIR'],'SavedInstall');self.assertNotIn('INSTALLDIR',search)
        self.assertEqual(next(r[1:] for r in tables['CustomAction'] if r[0]=='RestoreInstallDir'),[51,'INSTALLDIR','[SAVED_INSTALLDIR]'])
        for name in ('InstallUISequence','InstallExecuteSequence'):
            sequence={r[0]:r for r in tables[name]}
            self.assertEqual(sequence['RestoreInstallDir'][1],'SAVED_INSTALLDIR AND NOT INSTALLDIR')
            self.assertLess(sequence['AppSearch'][2],sequence['RestoreInstallDir'][2]);self.assertLess(sequence['RestoreInstallDir'][2],sequence['CostInitialize'][2])
        secure=set(dict(tables['Property'])['SecureCustomProperties'].split(';'))
        self.assertTrue({'INSTALLDIR','SAVED_INSTALLDIR','LICENSE_ACCEPTED'}<=secure)
    def test_exact_shortcut_names_checked_before_install_changes(self):
        tables=self.model['tables'];actions={r[0]:r for r in tables['CustomAction']};sequence={r[0]:r for r in tables['InstallExecuteSequence']}
        checks=[('CheckGuiMenu','[InsonicMenu]insonic.lnk','GuiMenu','START_MENU=1 AND '+m.effective_feature_local('Desktop')),('CheckCliMenu','[InsonicMenu]insonic CLI.lnk','CliMenu','START_MENU=1 AND '+m.effective_feature_local('CLI')),('CheckGuiDesktop','[DesktopFolder]insonic.lnk','GuiDesktop','DESKTOP_SHORTCUT=1 AND '+m.effective_feature_local('Desktop'))]
        for action,path,component,condition in checks:
            self.assertEqual(actions[action][1:],[2,'Launcher','--check-shortcut "'+path+'" '+component])
            self.assertEqual(sequence[action][1],condition)
            self.assertLess(sequence['Preflight'][2],sequence[action][2]);self.assertLess(sequence[action][2],sequence['InstallValidate'][2])
        self.assertIn('deselect its Start menu or Desktop integration',dict(tables['Error'])[1722])
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
        c=[r for r in self.model['tables']['LaunchCondition'] if 'WEBVIEW_MACHINE' in r[0]][0];self.assertIn('NOT '+m.effective_feature_local('Desktop'),c[0])
    def test_retained_local_features_count_but_explicit_removal_does_not(self):
        import re
        tables=self.model['tables'];conditions={message:condition for condition,message in tables['LaunchCondition']}
        core=next(value for key,value in conditions.items() if key.startswith('Shared runtime'))
        clients=next(value for key,value in conditions.items() if key.startswith('Select Desktop'))
        webview=next(value for key,value in conditions.items() if key.startswith('Desktop requires WebView'))
        sequence={row[0]:row[1] for row in tables['InstallUISequence']};checks={row[0]:row[1] for row in tables['InstallExecuteSequence']}
        def evaluate(condition,actions,installed,**properties):
            values={'REMOVE':'','Installed':1,'OLDPRODUCTS':'','START_MENU':1,'DESKTOP_SHORTCUT':1,'WEBVIEW_MACHINE':'','WEBVIEW_USER':''};values.update(properties)
            expression=re.sub(r'&([A-Za-z]+)',lambda match:str(actions.get(match.group(1),-1)),condition)
            expression=re.sub(r'!([A-Za-z]+)',lambda match:str(installed.get(match.group(1),2)),expression)
            for name,value in values.items():expression=re.sub(r'\b'+name+r'\b',repr(value),expression)
            expression=expression.replace('<>','!=');expression=re.sub(r'(?<![!<>=])=(?!=)','==',expression)
            for old,new in [('AND','and'),('OR','or'),('NOT','not')]:expression=re.sub(r'\b'+old+r'\b',new,expression)
            return bool(eval(expression,{'__builtins__':{}},{}))
        retained={'Core':3,'Desktop':3,'CLI':3};unchanged={'Core':-1,'Desktop':-1,'CLI':-1}
        self.assertTrue(evaluate(core,unchanged,retained));self.assertTrue(evaluate(clients,unchanged,retained))
        self.assertTrue(evaluate(core,{**unchanged,'Core':5},retained));self.assertFalse(evaluate(core,{'Core':5},{}))
        self.assertFalse(evaluate(core,{**unchanged,'Core':2},retained));self.assertFalse(evaluate(clients,{**unchanged,'Desktop':2,'CLI':2},retained))
        self.assertTrue(evaluate(core,{'Core':3},{}));self.assertTrue(evaluate(clients,{'CLI':3},{}));self.assertFalse(evaluate(clients,unchanged,{}))
        self.assertFalse(evaluate(webview,unchanged,retained));self.assertTrue(evaluate(webview,unchanged,retained,WEBVIEW_USER='126.0.0.0'))
        self.assertTrue(evaluate(webview,{**unchanged,'Desktop':2},retained))
        self.assertTrue(evaluate(sequence['ApplySavedDesktop'],unchanged,retained));self.assertFalse(evaluate(sequence['ClearSavedDesktop'],unchanged,retained))
        self.assertFalse(evaluate(sequence['ApplySavedDesktop'],{**unchanged,'Desktop':2},retained));self.assertTrue(evaluate(sequence['ClearSavedDesktop'],{**unchanged,'Desktop':2},retained))
        for action,feature in [('CheckGuiMenu','Desktop'),('CheckGuiDesktop','Desktop'),('CheckCliMenu','CLI')]:
            self.assertTrue(evaluate(checks[action],unchanged,retained));self.assertFalse(evaluate(checks[action],{**unchanged,feature:2},retained))
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
            for function,args in [(module.token_info,()),(module.owned_actor_processes,(None,)),(module.installer_mutex_diagnostic,()),(module.owned_user_object_observation,(None,None,None,None)),(module.ordinary_context,(None,None,None)),(module.ordinary_worker,(None,None,None)),(module.cancellation,(None,None,None)),(module.cancellation_worker,(None,None,None)),(module.prepare_launcher_fixture,(None,None)),(module.elected_launcher_context,(None,)),(module.linked_token_evidence,()),(module.service_diagnostic,()),(module.native_installer_worker,(None,)),(module.native_installer_dll,(None,)),(module.run_installer,(['/i','unused'],None)),(module.create_removal_sentinels,()),(module.assert_removal_sentinels,(None,)),(module.clean_removal_sentinels,(None,)),(module.private_context_desktop,('S-1-5-21-1-2-3-1001',)),(module.account_sid,(None,)),(module.grant_owned_account_stage,(None,None,None)),(module.registered_feature_states,(None,None))]:
                with self.assertRaisesRegex(RuntimeError,'disposable'):function(*args)
        self.assertIn('cancellation_rollback',m.QUALIFICATION_KEYS);self.assertIn('user_context',m.QUALIFICATION_KEYS)
    def test_full_ui_prepare_rebuilds_inherited_inventory_without_native_execution(self):
        import qualify_installer_ui as ui
        source=self.path/'diagnostic-input';fixture(source)
        (source/'help/offline-manifest.json').write_text(json.dumps({'version':'1.0.1','entrypoint':'index.html','public_landing_included':False}),encoding='utf-8')
        stage=self.path/'diagnostic-stage';stage.mkdir();models=[]
        def author(path,payload,model):models.append(model);path.write_bytes(b'owned portable MSI placeholder')
        def clone(old,target,*args,**kwargs):target.write_bytes(old.read_bytes())
        with patch.object(m,'disposable_runner',return_value=None),patch.object(m,'author',side_effect=author),patch.object(m,'NativeDatabase'):
            ui.prepare(stage,source,clone,diagnostic_only=True)
        self.assertEqual(len(models),1)
        inventory=[row for row in models[0]['payload'] if row['path']=='package-inventory.json']
        self.assertEqual(len(inventory),1)
        self.assertEqual(inventory[0]['sha256'],m.sha(stage/'ui-payload/package-inventory.json'))

    def test_invocation_diagnostic_refuses_owner_host_before_native_calls(self):
        spec=importlib.util.spec_from_file_location('invocation_guard',ROOT/'scripts/diagnose-installer-completion.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True),patch.object(module,'qualifier') as native:
            for function,args in [(module.main,()),(module.observe_invocation,(None,None,None)),(module.observe_rollback,(None,None,None)),(module.rollback_case,(None,None)),(module.recover_registration,(None,None,None)),(module.reinstall_advertised_probe,(None,None,None,None)),(module.author_recovery_exit_probe,(None,)),(module.author_recovery_container_probe,(None,))]:
                with self.assertRaisesRegex(RuntimeError,'disposable'):function(*args)
            native.assert_not_called()

    def test_recovery_reinstall_probe_refuses_missing_source_before_unregister(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('recovery_reinstall_fixture',ROOT/'scripts/diagnose-installer-completion.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        calls=[]
        dll=SimpleNamespace(MsiQueryProductStateW=lambda product:1,MsiConfigureProductExW=lambda *args:calls.append(args),MsiInstallProductW=lambda *args:calls.append(args))
        with patch.object(m,'disposable_runner',return_value=None):
            with self.assertRaisesRegex(RuntimeError,'source'):module.reinstall_advertised_probe(dll,'old',self.path/'missing.msi','ADDLOCAL=Core,CLI')
        self.assertEqual(calls,[])

    def test_container_probe_commits_no_product_or_payload_before_coordinator(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('recovery_container_author',ROOT/'scripts/diagnose-installer-completion.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        stage=self.path/'container-probe';(stage/'scripts').mkdir(parents=True);(stage/'native').mkdir();(stage/'ui-old.msi').write_bytes(b'owned native-valid costing fixture');rows=[]
        db=SimpleNamespace(execute=lambda sql,args=None,streams=():rows.append((sql,args,streams)),check=lambda code:None,handle=1,dll=SimpleNamespace(MsiDatabaseCommit=lambda handle:0),summary=lambda value:None,close=lambda:None)
        with patch.object(m,'disposable_runner',return_value=None),patch.object(m,'NativeDatabase',return_value=db),patch.dict(os.environ,{'GITHUB_ACTIONS':'true','RUNNER_ENVIRONMENT':'github-hosted','INSONIC_DISPOSABLE_INSTALLER_TEST':'1'}):module.author_recovery_container_probe(stage)
        actions=[args[0] for sql,args,streams in rows if args and 'InstallExecuteSequence' in sql]
        self.assertEqual(actions,['CostInitialize','FileCost','CostFinalize','InstallValidate','InstallInitialize','InstallFinalize'])
        self.assertNotIn('RegisterProduct',actions);self.assertNotIn('RemoveExistingProducts',actions)
        self.assertTrue(any(args and args[0]=='OwnedRecoveryCoordinator' and streams==(2,) for sql,args,streams in rows))
        self.assertTrue((stage/'container-product.json').is_file())

    def test_recovery_exit_probe_retains_startup_and_uses_failure_termination(self):
        import ast
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('recovery_exit_author',ROOT/'scripts/diagnose-installer-completion.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        stage=self.path/'owned-exit-probe';(stage/'scripts').mkdir(parents=True);(stage/'native').mkdir();(stage/'ui-model.json').write_text(json.dumps({'old_product':'owned-old-product'}),encoding='utf-8');rows=[]
        db=SimpleNamespace(execute=lambda sql,args:rows.append(args),check=lambda code:None,handle=1,dll=SimpleNamespace(MsiDatabaseCommit=lambda handle:0),close=lambda:None)
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'qualifier',return_value=SimpleNamespace(diagnostic_mode=lambda path:'rollback-exit-quiet')),patch.object(m,'NativeDatabase',return_value=db),patch.dict(os.environ,{'GITHUB_ACTIONS':'true','RUNNER_ENVIRONMENT':'github-hosted','INSONIC_DISPOSABLE_INSTALLER_TEST':'1'}):module.author_recovery_exit_probe(stage)
        self.assertIn(['OwnedRecoveryFailure','OLDPRODUCTS',-3],rows)
        action=next(row for row in rows if row[0]=='OwnedRecoveryFailure' and len(row)==4)
        self.assertEqual(action[1:3],[114,'OWNED_RECOVERY_PYTHON']);self.assertTrue(action[3].startswith('-I "'))
        source=(stage/'scripts/recovery-exit-bootstrap.py').read_text(encoding='utf-8');ast.parse(source)
        self.assertIn('environment_before',source);self.assertIn('context.recovery-exit-stream.log',source);self.assertIn('--recover-registration',source)

    def test_recovery_reinstall_probe_does_not_install_after_failed_unregister(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('recovery_reinstall_failure',ROOT/'scripts/diagnose-installer-completion.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        source=self.path/'old.msi';source.write_bytes(b'original owned MSI');calls=[]
        dll=SimpleNamespace(MsiQueryProductStateW=lambda product:1,MsiConfigureProductExW=lambda *args:1603,MsiInstallProductW=lambda *args:calls.append(args))
        with patch.object(m,'disposable_runner',return_value=None):result=module.reinstall_advertised_probe(dll,'old',source,'ADDLOCAL=Core,CLI LAUNCH=0')
        self.assertEqual(calls,[]);self.assertEqual(result['uninstall_code'],1603);self.assertNotIn('install_code',result)

    def test_recovery_reinstall_probe_uses_public_absent_then_original_package(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('recovery_reinstall_order',ROOT/'scripts/diagnose-installer-completion.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        calls=[];state=[1];source=self.path/'old.msi';source.write_bytes(b'original owned MSI')
        def uninstall(*args):calls.append(('uninstall',args));state[0]=-1;return 0
        def install(*args):calls.append(('install',args));state[0]=5;return 0
        dll=SimpleNamespace(MsiQueryProductStateW=lambda product:state[0],MsiConfigureProductExW=uninstall,MsiInstallProductW=install)
        with patch.object(m,'disposable_runner',return_value=None):result=module.reinstall_advertised_probe(dll,'old',source,'ADDLOCAL=Core,CLI LAUNCH=0')
        self.assertEqual(calls,[('uninstall',('old',0,2,'REMOVE=ALL LAUNCH=0 REBOOT=ReallySuppress')),('install',(str(source),'ADDLOCAL=Core,CLI LAUNCH=0'))])
        self.assertEqual(result,{'uninstall_code':0,'after_uninstall':-1,'install_code':0,'after_install':5,'feature_qualification':False})

    def test_rollback_diagnostic_retains_observation_when_native_cleanup_fails(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('rollback_retention_fixture',ROOT/'scripts/diagnose-installer-completion.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        stage=self.path/'rollback-diagnostic';(stage/'native').mkdir(parents=True);directory=stage/'UI program é & spaces';directory.mkdir();(directory/'owned.txt').write_bytes(b'old')
        (stage/'ui-model.json').write_text(json.dumps({'payload':[{'path':'owned.txt','sha256':m.digest(b'old'),'feature':'Core'}]}),encoding='utf-8')
        token={'user_sid_sha256':'owned','elevated':False}
        def state(product):return 1 if product=='old' else -1
        dll=SimpleNamespace(MsiQueryProductStateW=state)
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'qualifier',return_value=SimpleNamespace(diagnostic_mode=lambda stage:'rollback-reused')),patch.object(module,'rollback_case',side_effect=lambda stage,name:{'name':name,'token':token}),patch.object(module.C,'WinDLL',return_value=dll),patch.object(m,'CompoundFile',side_effect=lambda path:SimpleNamespace(streams=Path(path).name)),patch.object(m,'table_rows',side_effect=lambda stream,table:[['ProductCode','old' if stream=='ui-old.msi' else 'new']]):
            result=module.observe_rollback(stage,token,lambda *args:(_ for _ in ()).throw(RuntimeError('native cleanup1603')))
        self.assertEqual(result['transactions'][-1]['registered_states'],{'old':1,'new':-1})
        self.assertTrue(result['restored_old_payload']);self.assertFalse(result['feature_qualification'])
        self.assertEqual(result['state'],'passed')  # completed trace, explicitly never feature qualification
        self.assertFalse(result['cleanup'][0]['completed'])
        retained=json.loads((stage/'native/context.rollback-comparison.json').read_text(encoding='utf-8'))
        self.assertEqual(retained,result)

    def test_diagnostic_mode_uses_explicit_record_without_inherited_environment(self):
        spec=importlib.util.spec_from_file_location('diagnostic_mode_fixture',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        target=self.path/'installer-diagnostic-mode.json'
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{},clear=True):
            for mode in ('completion','rollback-reused','rollback-isolated','rollback-primed','rollback-late','rollback-msiexec','rollback-recovery','rollback-exit-full','rollback-exit-quiet','rollback-container'):
                target.write_text(json.dumps({'feature_qualification':False,'mode':mode}),encoding='utf-8')
                self.assertEqual(module.diagnostic_mode(self.path),mode)
            for invalid in ({'feature_qualification':True,'mode':'completion'},{'feature_qualification':False,'mode':'unknown'},{'feature_qualification':False,'mode':'completion','extra':1}):
                target.write_text(json.dumps(invalid),encoding='utf-8')
                with self.assertRaisesRegex(RuntimeError,'diagnostic mode'):module.diagnostic_mode(self.path)

    def test_native_client_preparation_is_not_an_error_dialog(self):
        import qualify_installer_ui as ui
        self.assertFalse(ui.native_client_error_dialog(['','Preparing to install...','Cancel','']))
        self.assertTrue(ui.native_client_error_dialog(['Invalid command line argument.','OK']))

    def test_msiexec_diagnostic_has_hidden_explicit_owned_launch(self):
        from types import SimpleNamespace
        import qualify_installer_ui as ui
        stage=self.path/'insonic-ci-account-00000000000000000000000000000000';stage.mkdir();(stage/'completion-invocation-diagnostic.json').write_text('{"qualification":false}',encoding='utf-8')
        (stage/'native').mkdir();msi=stage/'ui-old.msi';msi.write_bytes(b'owned fixture')
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{'ProgramData':str(self.path),'SystemRoot':str(self.path/'windows')}),patch.object(ui.subprocess,'Popen',return_value=SimpleNamespace(pid=123)) as start:
            ui.launch_msiexec_diagnostic(msi,stage/'UI program é & spaces',stage/'native/context.ui-rollback-install.log')
        args,kwargs=start.call_args
        self.assertIsInstance(args[0],str)
        self.assertIn('/qf',args[0]);self.assertIn('/l*v',args[0]);self.assertNotIn('LICENSE_ACCEPTED',args[0]);self.assertNotIn('LAUNCH=',args[0])
        self.assertIn(' INSTALLDIR="'+str(stage/'UI program é & spaces')+'" ',args[0])
        self.assertEqual(kwargs['executable'],str(self.path/'windows/System32/msiexec.exe'))
        self.assertEqual(kwargs['creationflags'],0x08000000);self.assertEqual(kwargs['stdin'],ui.subprocess.DEVNULL)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):ui.launch_msiexec_diagnostic(None,None,None)

    def test_full_ui_and_association_helpers_refuse_owner_host_before_native_calls(self):
        import qualify_installer_ui as ui
        with patch.dict(os.environ,{},clear=True):
            for function,args in [(ui.prepare,(None,None,None)),(ui.NativeUI,(None,)),(ui.qualify,(None,None,None,None,None,None)),(ui.export_captures,(None,None,None,None))]:
                with self.assertRaisesRegex(RuntimeError,'disposable'):function(*args)
            with self.assertRaisesRegex(RuntimeError,'disposable'):
                with ui.association(None,None):self.fail('owner association was changed')

    def test_private_context_desktop_restores_caller_on_success_and_failure(self):
        import ctypes as C
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_private_desktop',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        sid='S-1-5-21-1-2-3-1001';sddl=module.private_desktop_sddl(sid)
        self.assertIn('(A;;GA;;;'+sid+')',sddl);self.assertIn('(A;;GA;;;SY)',sddl);self.assertIn('S:(ML;;NW;;;ME)',sddl);self.assertNotIn(';;;WD',sddl);self.assertNotIn(';;;BA',sddl)
        with self.assertRaises(ValueError):module.private_desktop_sddl(sid+')(A;;GA;;;WD)')
        caller='S-1-5-21-1-2-3-1000';both=module.private_desktop_sddl(sid,caller)
        self.assertIn('(A;;GA;;;'+caller+')',both);self.assertIn('(A;;GA;;;'+sid+')',both)
        self.assertEqual(module.private_desktop_sddl(sid,sid).count('(A;;GA;;;'+sid+')'),1)
        with self.assertRaises(ValueError):module.private_desktop_sddl(sid,caller+')(A;;GA;;;WD)')
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        for failure in (False,True):
            state={'station':100,'desktop':200};closed=[];descriptors=[]
            def create_desktop(*args):state['desktop']=2000 if not failure else 200;return 0 if failure else 2000
            def descriptor(text,revision,out,size):descriptors.append(text);out._obj.value=1234;return True
            user=SimpleNamespace(GetProcessWindowStation=Function(lambda:state['station']),GetThreadDesktop=Function(lambda tid:state['desktop']),SetProcessWindowStation=Function(lambda value:state.update(station=value) or True),SetThreadDesktop=Function(lambda value:state.update(desktop=value) or True),CreateWindowStationW=Function(lambda name,flags,access,attrs:1000 if flags==1 else 0),CreateDesktopW=Function(create_desktop),CloseDesktop=Function(lambda value:closed.append(('desktop',value)) or True),CloseWindowStation=Function(lambda value:closed.append(('station',value)) or True))
            kernel=SimpleNamespace(GetCurrentThreadId=Function(lambda:42),LocalFree=Function(lambda value:closed.append(('descriptor',value.value)) or None))
            security=SimpleNamespace(ConvertStringSecurityDescriptorToSecurityDescriptorW=Function(descriptor))
            dlls={'user32':user,'kernel32':kernel,'advapi32':security}
            with patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(C,'WinError',side_effect=lambda code:OSError('fixture native failure'),create=True),patch.object(C,'get_last_error',return_value=5,create=True),patch.object(module,'owned_user_object_observation',return_value={'actual_native_descriptor':'mocked portable evidence'}):
                if failure:
                    with self.assertRaises(OSError) as failure_result:
                        with module.private_context_desktop(sid,caller):self.fail('failed creation yielded a desktop')
                    self.assertEqual(failure_result.exception.api,'CreateDesktopW-owned')
                else:
                    with module.private_context_desktop(sid,caller) as result:
                        self.assertEqual(state,{'station':100,'desktop':200});self.assertTrue(result['path'].startswith('insonic-ci-'));self.assertTrue(result['path'].endswith('\\Default'));self.assertFalse(result['shared_desktop_acl_modified'])
                        self.assertEqual(result['acl_subjects'],['SYSTEM','ordinary target user','calling CI user'])
                        self.assertEqual(closed,[])
            self.assertEqual(state,{'station':100,'desktop':200});self.assertIn(('station',1000),closed);self.assertIn(('descriptor',1234),closed)
            self.assertEqual(('desktop',2000) in closed,not failure);self.assertEqual(descriptors,[both])
    def test_ordinary_environment_uses_new_profile_and_no_owner_secrets(self):
        spec=importlib.util.spec_from_file_location('qualify_ordinary_environment',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        env={'SystemRoot':r'C:\Windows','COMPUTERNAME':'CI-FIXTURE','GITHUB_ACTIONS':'true','RUNNER_ENVIRONMENT':'github-hosted','INSONIC_DISPOSABLE_INSTALLER_TEST':'1','GITHUB_TOKEN':'owner-secret','APPDATA':'owner-private','LOCALAPPDATA':'owner-private','HOME':'owner-private','PATH':'owner-private'}
        profile=self.path/'inso_ci_0123456789ab'
        with patch.dict(os.environ,env,clear=True):text=module.ordinary_environment(profile,'inso_ci_0123456789ab',self.path/'temp')
        self.assertTrue(text.endswith('\0\0'));values=dict(item.split('=',1) for item in text.split('\0') if item)
        self.assertEqual(values['USERPROFILE'],str(profile));self.assertEqual(values['HOME'],str(profile));self.assertEqual(values['APPDATA'],str(profile/'AppData/Roaming'));self.assertEqual(values['LOCALAPPDATA'],str(profile/'AppData/Local'));self.assertEqual(values['GITHUB_ACTIONS'],'true')
        self.assertNotIn('owner-secret',text);self.assertNotIn('owner-private',text);self.assertNotIn('GITHUB_TOKEN',values)
    def test_profile_cleanup_requires_created_account_and_exact_owned_target(self):
        spec=importlib.util.spec_from_file_location('qualify_ordinary_cleanup_bounds',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        users=self.path/'users';users.mkdir();name='inso_ci_0123456789ab';target=users/name;target.mkdir()
        with patch.dict(os.environ,{'USERPROFILE':str(users/'original-ci-actor')}):
            self.assertEqual(module.validate_account_profile(name,target),target)
            for account,path in [('original-ci-actor',target),(name,users/'original-ci-actor'),(name,self.path/name)]:
                with self.assertRaises(ValueError):module.validate_account_profile(account,path)
    def test_scheduler_password_stays_in_stdin_and_failure_cleans_exact_owned_task(self):
        import base64
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_owned_scheduler',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.scheduler_ordinary(None,None,None,None,None,None,None)
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.owned_batch_right(None)
        base=self.path/'programdata';base.mkdir();stage=base/('insonic-ci-account-'+'a'*32);stage.mkdir();sid='S-1-5-21-1-2-3-1001';name='inso_ci_0123456789ab';password='private-password-material'
        (stage/'owned-ci-account.json').write_text(json.dumps({'account':name,'sid_sha256':m.digest(sid.encode())}),encoding='utf-8');calls=[];records=[]
        def bridge(command,**options):
            self.assertNotIn(password,repr(command));self.assertNotIn(password,repr(options.get('env')))
            self.assertEqual(options['creationflags'],0x08000000);self.assertEqual(options['timeout'],260)
            script=base64.b64decode(command[-1]).decode('utf-16le');self.assertIn("ExecutionTimeLimit='PT240S'",script);self.assertIn('RegisterTaskDefinition($inputObject.task_name,$definition,2,',script);self.assertIn("$definition.Principal.UserId='SYSTEM';$definition.Principal.LogonType=5;",script);self.assertIn("'SYSTEM',$null,5,$null)",script);self.assertIn('$result.logon_type=5',script)
            request=json.loads(options['input']);calls.append(request)
            if request['mode']=='run':
                self.assertEqual(request['password'],password);self.assertNotIn('provider-secret',request['arguments']);self.assertNotIn('owner-private',request['arguments']);self.assertIn('os.environ.clear()',request['arguments'])
                return SimpleNamespace(returncode=0,stdout=b'{"phase":"failed","hresult":5,"deleted":false}')
            self.assertNotIn('password',request);return SimpleNamespace(returncode=0,stdout=b'{"phase":"cleanup-completed","deleted":true}')
        env={'ProgramData':str(base),'SystemRoot':r'C:\Windows','COMPUTERNAME':'CI-FIXTURE','GITHUB_ACTIONS':'true','RUNNER_ENVIRONMENT':'github-hosted','INSONIC_DISPOSABLE_INSTALLER_TEST':'1','GITHUB_TOKEN':'provider-secret','APPDATA':'owner-private','LOCALAPPDATA':'owner-private','HOME':'owner-private'}
        with patch.dict(os.environ,env,clear=True),patch.object(m,'disposable_runner',return_value=None),patch.object(module.subprocess,'CREATE_NO_WINDOW',0x08000000,create=True),patch.object(module.subprocess,'run',side_effect=bridge):
            with self.assertRaisesRegex(RuntimeError,'did not complete'):module.scheduler_ordinary(stage,name,sid,password,[str(stage/'scripts/qualify-windows-msi.py'),'--ordinary-worker',str(stage/'fixture.msi')],lambda phase,**extra:records.append((phase,extra)),self.path/name)
        self.assertEqual(len(calls),2);self.assertEqual(calls[0]['task_name'],calls[1]['task_name']);self.assertEqual(calls[0]['description'],calls[1]['description']);self.assertEqual(calls[1]['sid'],sid);self.assertNotIn(password,repr(records));self.assertTrue(stage.exists())
    def test_ordinary_first_and_later_diagnostic_bytes_remain_distinct(self):
        spec=importlib.util.spec_from_file_location('qualify_actor_retention',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        native=self.path/'native';native.mkdir();item=native/'context.install.log';item.write_bytes(b'first actor actual log');progress=native/'context.progress.json';progress.write_bytes(b'first actor actual token')
        module.retain_ordinary_diagnostics(native,self.path/'ordinary-first-context.json')
        item.write_bytes(b'later actor actual log');progress.write_bytes(b'later actor actual token');module.retain_ordinary_diagnostics(native,self.path/'ordinary-context.json')
        self.assertEqual((self.path/'ordinary-first-context.install.log').read_bytes(),b'first actor actual log');self.assertEqual((self.path/'ordinary-context.install.log').read_bytes(),b'later actor actual log')
        self.assertEqual((self.path/'ordinary-first-context.progress.json').read_bytes(),b'first actor actual token');self.assertEqual((self.path/'ordinary-context.progress.json').read_bytes(),b'later actor actual token')
    def test_borrowed_primary_token_membership_uses_temporary_impersonation_not_system(self):
        import ctypes as C
        import ctypes.wintypes as W
        import struct
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_primary_token_query',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        primary=W.HANDLE(400);closed=[];membership_handles=[];strings=[];sid='S-1-5-21-1-2-3-1001'
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def raw(value):return C.string_at(C.byref(value),C.sizeof(value))
        values={20:raw(W.DWORD(0)),18:raw(W.DWORD(1)),1:raw(C.c_void_p(123)),25:raw(C.c_void_p(124)),3:struct.pack('<I',1)+struct.pack('<QI',0,2),10:bytes(16),12:raw(W.DWORD(0))}
        def info(token,kind,output,size,required):
            self.assertEqual(token.value,400);required._obj.value=len(values[kind])
            if output is None:return False
            C.memmove(output,values[kind],len(values[kind]));return True
        def duplicate(token,level,out):self.assertEqual(token.value,400);self.assertEqual(level,2);out._obj.value=600;return True
        def convert(value,out):
            address=value.value if hasattr(value,'value') else value;text=sid if address==123 else 'S-1-16-8192';strings.append(C.create_unicode_buffer(text));out._obj.value=C.cast(strings[-1],C.c_void_p).value;return True
        def well_known(kind,domain,output,size):C.memmove(output,struct.pack('<I',kind),4);return True
        def membership(token,group,out):
            self.assertIsNotNone(token);self.assertEqual(token.value,600);membership_handles.append(token.value);out._obj.value=struct.unpack('<I',C.string_at(group,4))[0]==11;return True
        kernel=SimpleNamespace(GetCurrentProcess=Function(lambda:self.fail('borrowed target must not query SYSTEM current process')),CloseHandle=Function(lambda handle:closed.append(handle.value)),LocalFree=Function(lambda handle:None))
        security=SimpleNamespace(OpenProcessToken=Function(lambda *args:self.fail('borrowed target must not open SYSTEM token')),GetTokenInformation=Function(info),CreateWellKnownSid=Function(well_known),CheckTokenMembership=Function(membership),ConvertSidToStringSidW=Function(convert),DuplicateToken=Function(duplicate));dlls={'kernel32':kernel,'advapi32':security}
        with patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True):proof=module.token_info(primary)
        self.assertEqual(proof['user_sid_sha256'],m.digest(sid.encode()));self.assertFalse(proof['administrator_enabled']);self.assertTrue(proof['interactive_logon']);self.assertEqual(proof['integrity_rid'],8192);self.assertEqual(closed,[600]);self.assertEqual(membership_handles,[600,600,600])
    def test_full_ui_desktop_and_session_fixture_refuse_owner_machine(self):
        spec=importlib.util.spec_from_file_location('qualify_full_ui_guard',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            for context in (module.interactive_context_desktop('S-1-5-21-1-2-3-1001','S-1-5-18',self.path),):
                with self.assertRaisesRegex(RuntimeError,'disposable'):
                    with context:self.fail('fixture entered the owner machine')

    def test_exact_account_user_object_grant_restores_after_failure_and_refuses_outside_drift(self):
        import ctypes as C
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_user_object_access',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        sid='S-1-5-21-1-2-3-1001';original='D:(A;;GA;;;SY)';state={'sddl':original};buffers=[];writes=[]
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):
                with module.owned_user_object_access(None,sid,0xf01ff,None,None,None):self.fail('owner object was touched')
        def text_pointer(value,out):
            buffer=C.create_unicode_buffer(value);buffers.append(buffer);out._obj.value=C.addressof(buffer);return True
        def get_security(handle,info,target,capacity,needed):
            buffer=C.create_unicode_buffer(state['sddl']);needed._obj.value=C.sizeof(buffer)
            if target is None:return False
            C.memmove(target,buffer,C.sizeof(buffer));return True
        def set_security(handle,info,value):
            self.assertEqual(info._obj.value,4);state['sddl']=C.wstring_at(value);writes.append(state['sddl']);return True
        user=SimpleNamespace(GetUserObjectSecurity=get_security,SetUserObjectSecurity=set_security)
        security=SimpleNamespace(ConvertSecurityDescriptorToStringSecurityDescriptorW=lambda raw,revision,info,out,length:text_pointer(C.wstring_at(raw),out),ConvertStringSecurityDescriptorToSecurityDescriptorW=lambda text,revision,out,length:text_pointer(text,out))
        kernel=SimpleNamespace(LocalFree=lambda pointer:None)
        def observation(*args):return {'descriptor_sha256':m.digest(state['sddl'].encode()),'owner':'unchanged','group':'unchanged','mandatory_label':'unchanged'}
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'owned_user_object_observation',side_effect=observation):
            for fail in (False,True):
                state['sddl']=original
                try:
                    with module.owned_user_object_access(55,sid,0xf01ff,user,security,kernel) as evidence:
                        self.assertEqual(state['sddl'],original+'(A;;0xf01ff;;;'+sid+')')
                        if fail:raise RuntimeError('injected actor failure')
                except RuntimeError as error:self.assertEqual(str(error),'injected actor failure')
                self.assertEqual(state['sddl'],original);self.assertTrue(evidence['restored']);self.assertEqual(evidence['original_security'],evidence['restored_security'])
            with self.assertRaisesRegex(RuntimeError,'outside the exact grant'):
                with module.owned_user_object_access(55,sid,0xf01ff,user,security,kernel):state['sddl']='D:(A;;GR;;;WD)'
            self.assertEqual(state['sddl'],'D:(A;;GR;;;WD)')
        self.assertEqual(writes[-1],original+'(A;;0xf01ff;;;'+sid+')')

    def test_interactive_actor_authenticates_normal_user_without_second_profile_or_session_reassignment(self):
        import ctypes as C
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_interactive_actor',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.interactive_ordinary(None,None,None,None,None,None,None)
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.ordinary_ui_worker(None)
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.ordinary_namespace_probe(None)
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.object_namespace_diagnostic()
        stage=self.path/'interactive';stage.mkdir();(stage/'scripts').mkdir();(stage/'native').mkdir()
        name='inso_ci_0123456789ab';sid='S-1-5-21-1-2-3-1001';password='private-memory-password';closed=[];events=[]
        (stage/'owned-ci-account.json').write_text(json.dumps({'account':name,'sid_sha256':m.digest(sid.encode())}),encoding='utf-8')
        target={'user_sid_sha256':m.digest(sid.encode()),'session_id':2,'elevated':False,'administrator_enabled':False,'integrity_rid':8192,'authentication_luid':{'low':123,'high':0},'enabled_privileges':1,'interactive_logon':True}
        desktop={'path':r'WinSta0\Default','session_id':2,'target_user_sid_sha256':target['user_sid_sha256']}
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def create(account,domain,secret,logon,exe,command,flags,environment,cwd,startup,out):
            self.assertEqual((account,domain,secret,logon),(name,'.',password,0));self.assertTrue(exe.endswith('pythonw.exe'));self.assertLess(len(command.value),1024);self.assertEqual(flags,0x08000400);self.assertEqual(startup._obj.desktop,desktop['path']);self.assertEqual(cwd,str(stage));self.assertNotIn(password,command.value+environment.value)
            purpose='namespace-probe' if 'namespace-probe' in command.value else 'ui'
            bootstrap=(stage/('scripts/interactive-'+purpose+'-bootstrap.py')).read_text();self.assertNotIn(password,bootstrap);self.assertIn('--ordinary-namespace-probe' if purpose=='namespace-probe' else '--ordinary-ui-worker',bootstrap);out._obj.process=555;out._obj.thread=556;out._obj.pid=888
            (stage/('native/context.namespace-probe.json' if purpose=='namespace-probe' else 'native/context.ui.json')).write_text(json.dumps({'token':target,'state':'passed'}),encoding='utf-8');return True
        def code(process,out):out._obj.value=0;return True
        security=SimpleNamespace(CreateProcessWithLogonW=Function(create))
        kernel=SimpleNamespace(WaitForSingleObject=Function(lambda *args:0),GetExitCodeProcess=Function(code),CloseHandle=Function(lambda handle:closed.append(handle.value if hasattr(handle,'value') else handle)),TerminateProcess=Function(lambda *args:self.fail('successful child must not be killed')))
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{'COMPUTERNAME':'CIHOST','SystemRoot':str(self.path/'Windows')},clear=True),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:{'advapi32':security,'kernel32':kernel}[name],create=True),patch.object(module,'token_info',return_value={'session_id':2}):
            for probe_only in (True,False):
                proof=module.interactive_ordinary(stage,name,sid,password,self.path/name,desktop,lambda phase,**extra:events.append({'phase':phase,**extra}),probe_only=probe_only)
                self.assertEqual(proof['token'],target)
        self.assertEqual(closed,[556,555,556,555]);self.assertEqual(events[-1]['token'],target);self.assertFalse(events[0]['profile_loaded_by_child']);self.assertFalse(events[0]['session_token_reassignment']);self.assertNotIn(password,repr(events))

    def test_existing_html_user_choice_is_preserved_and_owned_command_restored_after_failure(self):
        import copy
        import ctypes as C
        from types import SimpleNamespace
        import qualify_installer_ui as ui
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):
                with ui.fixture_html_registration(None,None,None,None):self.fail('owner association entered')
        name='inso_ci_0123456789ab';sid='S-1-5-21-1-2-3-1001';stage=self.path/('insonic-ci-account-'+'a'*32);stage.mkdir();(stage/'native').mkdir();handler=stage/'documentation-handler.exe';handler.write_bytes(b'owned image')
        token={'user_sid_sha256':m.digest(sid.encode()),'administrator_enabled':False,'elevated':False}
        (stage/'owned-ci-account.json').write_text(json.dumps({'account':name,'sid_sha256':token['user_sid_sha256']}),encoding='utf-8')
        prog='MSEdgeHTM';base='Software\\Classes\\'+prog;command=base+r'\shell\open\command';choice=r'Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\.html\UserChoice'
        class Key:
            def __init__(self,path):self.path=path
            def __enter__(self):return self
            def __exit__(self,*args):pass
        class Registry:
            HKEY_CURRENT_USER='exact-current-user';KEY_QUERY_VALUE=1;KEY_SET_VALUE=2;REG_SZ=1
            def __init__(self,existing):
                self.data={choice:{'ProgId':(prog,1),'Hash':('private-native-choice-hash',1)}}
                if existing:
                    for suffix in ('',r'\shell',r'\shell\open',r'\shell\open\command'):self.data[base+suffix]={}
                    self.data[command]={'':('original-browser-command',1),'DelegateExecute':('original-delegate',1),'Keep':('unrelated',1)}
            def OpenKey(self,root,path,*args):
                if root!=self.HKEY_CURRENT_USER:raise AssertionError('machine association requested')
                if path not in self.data:raise FileNotFoundError(path)
                return Key(path)
            def CreateKey(self,root,path):self.data.setdefault(path,{});return self.OpenKey(root,path)
            def QueryValueEx(self,key,name):
                if name not in self.data[key.path]:raise FileNotFoundError(name)
                return self.data[key.path][name]
            def SetValueEx(self,key,name,reserved,kind,value):self.data[key.path][name]=(value,kind)
            def DeleteValue(self,key,name):
                if name not in self.data[key.path]:raise FileNotFoundError(name)
                del self.data[key.path][name]
            def QueryInfoKey(self,key):return sum(path.rpartition('\\')[0]==key.path for path in self.data),len(self.data[key.path]),0
            def EnumValue(self,key,index):name=sorted(self.data[key.path])[index];value,kind=self.data[key.path][name];return name,value,kind
            def DeleteKey(self,root,path):del self.data[path]
        class Function:
            def __call__(self,account,size):account.value=name;return True
        security=SimpleNamespace(GetUserNameW=Function())
        for existing in (False,True):
            registry=Registry(existing);before=copy.deepcopy(registry.data)
            with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{'ProgramData':str(self.path)}),patch.object(C,'WinDLL',return_value=security,create=True),patch.dict(sys.modules,{'winreg':registry}):
                with self.assertRaisesRegex(ValueError,'native actor failure'):
                    with ui.fixture_html_registration(stage,token,prog,handler):
                        self.assertEqual(registry.data[choice],before[choice]);self.assertNotIn('DelegateExecute',registry.data[command]);self.assertIn(str(handler),registry.data[command][''][0]);raise ValueError('native actor failure')
            self.assertEqual(registry.data,before)
            evidence=json.loads((stage/'native/context.association.json').read_text());self.assertTrue(evidence['restored']);self.assertFalse(evidence['user_choice_modified']);self.assertNotIn('private-native-choice-hash',repr(evidence))

    def test_progress_inspection_does_not_scale_pause_with_validation_or_message_count(self):
        import qualify_installer_ui as ui
        class Event:
            def __init__(self):self.ready=False;self.total=0;self.now=0
            def is_set(self):return self.ready
            def wait(self,seconds):self.total+=seconds;self.now+=seconds
        event=Event();inspection=ui.ProgressInspection(event,lambda:event.now)
        for _ in range(2000):inspection.progress()
        self.assertEqual(event.total,0)
        for _ in range(2000):inspection.start();inspection.progress()
        self.assertLessEqual(event.total,.251)
        event.ready=True;before=event.total
        for _ in range(2000):inspection.progress()
        self.assertEqual(event.total,before)

    def test_session_namespace_fixture_restores_exact_descriptor_without_touching_global_or_child_objects(self):
        from contextlib import nullcontext
        spec=importlib.util.spec_from_file_location('qualify_session_namespace',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):
                with module.interactive_context_namespace(None,None,None,None,None):self.fail('owner namespace entered')
        stage=self.path/('insonic-ci-account-'+'a'*32);stage.mkdir();sid='S-1-5-21-1-2-3-1001';caller='S-1-5-21-1-2-3-1000';session=2
        (stage/'owned-ci-account.json').write_text(json.dumps({'account':'inso_ci_0123456789ab','sid_sha256':m.digest(sid.encode())}),encoding='utf-8')
        desktop={'session_id':session,'target_user_sid_sha256':m.digest(sid.encode()),'caller_sid_sha256':m.digest(caller.encode())}
        target=r'\Sessions\2\BaseNamedObjects';probe={'token':{'session_id':session,'user_sid_sha256':m.digest(sid.encode())},'directories':{'objects':[{'path':target,'requested_access':15,'error':5,'opened':False}]}}
        class API:
            def __init__(self):self.current='D:(A;;GA;;;SY)';self.calls=[]
            def open(self,name,access):self.calls.append(('open',name,access));return 77,0
            def checked(self,name,status):self.asserted=(name,status)
            def dacl(self,handle):return self.current.encode(),self.current
            def describe(self,handle):return {'descriptor_sha256':m.digest(self.current.encode()),'owner_group_dacl_label_sddl':'O:SYG:SY'+self.current+'S:(ML;;NW;;;ME)'}
            def convert(self,text):return nullcontext(text)
            def set_dacl(self,handle,value):self.calls.append(('set',handle,value));self.current=value.decode() if isinstance(value,bytes) else value
            def close(self,handle):self.calls.append(('close',handle))
        api=API();observed=None
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{'ProgramData':str(self.path)}),patch.object(module,'token_info',return_value={'session_id':session,'user_sid_sha256':m.digest(caller.encode())}),patch.object(module,'NativeObjectDirectories',return_value=api):
            with self.assertRaisesRegex(ValueError,'native worker failed'):
                with module.interactive_context_namespace(stage,sid,caller,desktop,probe) as observed:raise ValueError('native worker failed')
        self.assertEqual(api.current,'D:(A;;GA;;;SY)');self.assertEqual(api.calls[0],('open',target,0x6000f));self.assertEqual(api.calls[-1],('close',77));self.assertEqual(len([x for x in api.calls if x[0]=='set']),2)
        self.assertTrue(api.calls[1][2].endswith('(A;;0xf;;;'+sid+')'));self.assertTrue(observed['security_restored']);self.assertFalse(observed['global_namespace_modified']);self.assertEqual(observed['rights_mask'],15)
        api=API()
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{'ProgramData':str(self.path)}),patch.object(module,'token_info',return_value={'session_id':session,'user_sid_sha256':m.digest(caller.encode())}),patch.object(module,'NativeObjectDirectories',return_value=api):
            with self.assertRaisesRegex(RuntimeError,'changed outside'):
                with module.interactive_context_namespace(stage,sid,caller,desktop,probe):api.current='D:(A;;0x1;;;SY)'
        self.assertEqual(api.current,'D:(A;;0x1;;;SY)');self.assertEqual(len([x for x in api.calls if x[0]=='set']),1)
        for error,opened in ((0,True),(2,False)):
            probe['directories']['objects'][0].update(error=error,opened=opened)
            with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{'ProgramData':str(self.path)}),patch.object(module,'token_info',return_value={'session_id':session,'user_sid_sha256':m.digest(caller.encode())}),patch.object(module,'NativeObjectDirectories') as native:
                if opened:
                    with module.interactive_context_namespace(stage,sid,caller,desktop,probe) as healthy:self.assertFalse(healthy['modified']);self.assertTrue(healthy['security_restored'])
                else:
                    with self.assertRaisesRegex(RuntimeError,'other than access denied'):
                        with module.interactive_context_namespace(stage,sid,caller,desktop,probe):self.fail('unexpected native error entered fixture')
                native.assert_not_called()


    def test_system_broker_creates_only_the_verified_normal_target_without_profile_load(self):
        import ctypes as C
        import io
        from contextlib import nullcontext
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_system_broker',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.ordinary_system_broker(None,None,None,None)
        account='inso_ci_0123456789ab';sid='S-1-5-21-1-2-3-1001';password='private-memory-password';pipe='insonic-ci-credential-'+'a'*32;credentials={'account':'CIHOST\\'+account,'sid':sid,'password':password};calls=[];closed=[]
        system={'user_sid_sha256':m.digest(b'S-1-5-18'),'session_id':0};target={'user_sid_sha256':m.digest(sid.encode()),'administrator_enabled':False,'elevated':False,'enabled_privileges':1,'integrity_rid':8192,'session_id':0,'interactive_logon':True}
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def logon(name,domain,secret,kind,provider,out):self.assertEqual(secret,password);self.assertEqual(kind,2);out._obj.value=444;calls.append('target-interactive-logon');return True
        def create(token,exe,command,process_acl,thread_acl,inherit,flags,environment,cwd,startup,out):
            self.assertEqual(token.value,444);self.assertTrue(str(exe).endswith('pythonw.exe'));self.assertIn('worker-stream.log',command.value);self.assertIn('runpy.run_path',command.value);self.assertEqual(flags,0x08000400);self.assertFalse(inherit);self.assertNotIn(password,command.value);self.assertNotIn(password,environment.value);self.assertEqual(startup._obj.desktop,'owned-CI-station\\Default');out._obj.process=555;out._obj.thread=556;out._obj.pid=888;calls.append('normal-target-created');return True
        def code(process,out):out._obj.value=0;return True
        kernel=SimpleNamespace(WaitForSingleObject=Function(lambda *args:0),GetExitCodeProcess=Function(code),CloseHandle=Function(lambda handle:closed.append(handle.value if hasattr(handle,'value') else handle)),TerminateProcess=Function(lambda *args:self.fail('no successful child or SYSTEM process may be killed')))
        security=SimpleNamespace(LogonUserW=Function(logon),CreateProcessAsUserW=Function(create));dlls={'kernel32':kernel,'advapi32':security};output=self.path/'context.json';env={'USERNAME':account,'COMPUTERNAME':'CIHOST','APPDATA':'owned-ordinary-profile'}
        def privileges(names):self.assertEqual(names,('SeAssignPrimaryTokenPrivilege','SeIncreaseQuotaPrivilege'));calls.append('broker-only-privileges');return nullcontext({'ordinary_actor_token_changed':False})
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,env,clear=True),patch.object(module,'token_info',side_effect=[system,target]),patch.object(module,'open',return_value=io.BytesIO((json.dumps(credentials)+'\n').encode()),create=True),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(module,'private_context_desktop',return_value=nullcontext({'path':'owned-CI-station\\Default'})) as desktop,patch.object(module,'qualifier_profile_privileges',side_effect=privileges):module.ordinary_system_broker(pipe,self.path/'fixture.msi',self.path/'owned-install',output)
        self.assertEqual(calls,['target-interactive-logon','broker-only-privileges','normal-target-created']);self.assertEqual(closed,[556,555,444]);self.assertEqual(desktop.call_args.args,(sid,'S-1-5-18'));proof=json.loads(output.with_suffix('.broker.json').read_text());self.assertEqual(proof['broker_token'],system);self.assertEqual(proof['target_token'],target);self.assertFalse(proof['target_profile_loaded_by_broker']);self.assertNotIn(password,output.with_suffix('.broker.json').read_text())
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,env,clear=True),patch.object(module,'token_info',return_value={**system,'user_sid_sha256':m.digest(sid.encode())}),patch.object(module,'open',create=True) as pipe_open:
            with self.assertRaisesRegex(RuntimeError,'SYSTEM'):module.ordinary_system_broker(pipe,None,None,output)
            pipe_open.assert_not_called()
    def test_service_release_requires_ownership_completed_transactions_and_idle_identity(self):
        import ctypes as C
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_owned_service_release',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.release_owned_installer_service(None,None,None,None,None,None)
        sid='S-1-5-21-1-2-3-1001';binary=self.path/'Windows/System32/msiexec.exe';initial={'state':1,'pid':0,'binary_identity_valid':True,'canonical_binary':str(binary.resolve())};current={**initial,'state':4,'pid':4242,'accepted_controls':1};task={'phase':'completed','exit_code':0,'deleted':True};install=self.path/'owned-install.worker.json';remove=self.path/'owned-remove.worker.json'
        for path,action in ((install,'/i'),(remove,'/x')):path.write_text(json.dumps({'kind':'native-installer-worker','phase':'completed','exit_code':0,'action':action,'service':{'state':4,'pid':4242}}),encoding='utf-8')
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'owned_actor_processes',return_value=[]),patch.object(module,'service_diagnostic',return_value=current),patch.object(C,'WinDLL',create=True) as native:
            for owned in (None,{**initial,'state':4,'pid':4242},{**initial,'binary_identity_valid':False}):
                with self.assertRaisesRegex(RuntimeError,'initially stopped'):module.release_owned_installer_service(owned,install,remove,task,sid,__import__('time').monotonic()+10)
            for actual in ({**current,'pid':4243},{**current,'accepted_controls':0},{**current,'binary_identity_valid':False}):
                with patch.object(module,'service_diagnostic',return_value=actual):
                    with self.assertRaisesRegex(RuntimeError,'unowned, changed, busy'):module.release_owned_installer_service(initial,install,remove,task,sid,__import__('time').monotonic()+10)
            with patch.object(module,'owned_actor_processes',return_value=[{'pid':91}]):
                with self.assertRaisesRegex(RuntimeError,'processes remain'):module.release_owned_installer_service(initial,install,remove,task,sid,__import__('time').monotonic()+10)
            native.assert_not_called()
        calls=[];closed=[];state=[4];image=[str(binary)]
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def status(service,kind,out,size,needed):out._obj.state=state[0];out._obj.pid=4242 if state[0]==4 else 0;out._obj.controls=1;return True
        def stop(service,control,out):self.assertEqual(control,1);calls.append(('graceful-SCM-stop',service));state[0]=1;return True
        def query(process,flags,out,size):out.value=image[0];return True
        service=SimpleNamespace(OpenSCManagerW=Function(lambda *args:11),OpenServiceW=Function(lambda manager,name,access:12),QueryServiceStatusEx=Function(status),ControlService=Function(stop),CloseServiceHandle=Function(lambda handle:closed.append(handle)))
        kernel=SimpleNamespace(OpenProcess=Function(lambda access,inherit,pid:13),QueryFullProcessImageNameW=Function(query),CloseHandle=Function(lambda handle:closed.append(handle)));dlls={'advapi32':service,'kernel32':kernel}
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'owned_actor_processes',return_value=[]),patch.object(module,'service_diagnostic',return_value=current),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True):
            image[0]=str(binary.with_name('foreign.exe'))
            with self.assertRaisesRegex(RuntimeError,'exact Windows Installer image'):module.release_owned_installer_service(initial,install,remove,task,sid,__import__('time').monotonic()+10)
            self.assertEqual(calls,[]);self.assertEqual(closed,[13,12,11]);closed.clear();image[0]=str(binary)
            proof=module.release_owned_installer_service(initial,install,remove,task,sid,__import__('time').monotonic()+10)
        self.assertEqual(calls,[('graceful-SCM-stop',12)]);self.assertEqual(closed,[13,12,11]);self.assertTrue(proof['restored_initial_stopped_state']);self.assertFalse(proof['process_terminated']);self.assertFalse(proof['registry_hive_forced'])
    def test_owned_profile_provider_requires_every_absence_and_refuses_foreign_or_live_actors(self):
        from types import SimpleNamespace
        import shutil
        spec=importlib.util.spec_from_file_location('qualify_owned_profile_provider',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.delete_owned_profile_provider(None,None,None)
        users=self.path/'provider-users';users.mkdir();caller=users/'caller';caller.mkdir();profile=users/'inso_ci_0123456789ab';profile.mkdir();sid='S-1-5-21-1-2-3-1001';calls=[]
        keys=('provider_absent','registration_absent','user_hive_absent','classes_hive_absent','folder_absent');receipt={'phase':'completed','complete_absence':True,**{key:True for key in keys}}
        def run(command,**options):calls.append((command,options));return SimpleNamespace(returncode=0,stdout=json.dumps(receipt).encode(),stderr=b'')
        absent={'hives':{'user':{'present':False,'winerror':2},'_Classes':{'present':False,'winerror':2}},'profile_registration_error':2,'machine_state_modified':False}
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{'USERPROFILE':str(caller),'SystemRoot':str(self.path/'Windows')}),patch.object(module,'owned_actor_processes',return_value=[]),patch.object(module,'profile_cleanup_observation',return_value=absent),patch.object(module.subprocess,'CREATE_NO_WINDOW',0x08000000,create=True),patch.object(module.subprocess,'run',side_effect=run):
            with self.assertRaisesRegex(ValueError,'exact CI-owned account'):module.delete_owned_profile_provider(sid,caller,__import__('time').monotonic()+10)
            with patch.object(module,'owned_actor_processes',return_value=[{'pid':66}]):
                with self.assertRaisesRegex(RuntimeError,'processes'):module.delete_owned_profile_provider(sid,profile,__import__('time').monotonic()+10)
            self.assertEqual(calls,[])
            receipt['classes_hive_absent']=False
            with self.assertRaisesRegex(RuntimeError,'every required absence'):module.delete_owned_profile_provider(sid,profile,__import__('time').monotonic()+10)
            receipt['classes_hive_absent']=True
            with self.assertRaisesRegex(RuntimeError,'actual owned folder'):module.delete_owned_profile_provider(sid,profile,__import__('time').monotonic()+10)
            shutil.rmtree(profile);proof=module.delete_owned_profile_provider(sid,profile,__import__('time').monotonic()+10)
        self.assertTrue(proof['complete_absence']);self.assertEqual(proof['independent_absence'],absent);self.assertNotIn(sid,repr(calls[-1][0]));self.assertEqual(json.loads(calls[-1][1]['input']),{'sid':sid,'profile':str(profile)});self.assertEqual(calls[-1][1]['creationflags'],0x08000000)
    def test_profile_lease_observer_is_exact_sid_readonly_and_does_not_publish_sid_in_argv(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_profile_lease',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.profile_lease_observation(None,None)
        sid='S-1-5-21-1-2-3-1001';profile=self.path/'inso_ci_0123456789ab';observed={'profile':{'local_path':str(profile),'loaded':True,'ref_count':1,'special':False},'events':[{'message':'exact hive '+sid}],'machine_state_modified':False};calls=[]
        def run(command,**options):calls.append((command,options));return SimpleNamespace(returncode=0,stdout=json.dumps(observed).encode(),stderr=b'')
        with patch.object(m,'disposable_runner',return_value=None),patch.dict(os.environ,{'SystemRoot':str(self.path/'Windows')}),patch.object(module.subprocess,'CREATE_NO_WINDOW',0x08000000,create=True),patch.object(module.subprocess,'run',side_effect=run):
            with self.assertRaisesRegex(ValueError,'exact owned SID'):module.profile_lease_observation('foreign;script',profile)
            proof=module.profile_lease_observation(sid,profile)
        self.assertEqual(len(calls),1);self.assertNotIn(sid,repr(calls[0][0]));self.assertEqual(json.loads(calls[0][1]['input']),{'sid':sid,'profile':str(profile)});self.assertEqual(calls[0][1]['creationflags'],0x08000000);self.assertFalse(proof['machine_state_modified']);self.assertNotIn(sid,proof['events'][0]['message']);self.assertEqual(proof['profile']['ref_count'],1)
    def test_profile_process_observation_selects_exact_sid_and_never_stops_processes(self):
        import ctypes as C
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_owned_process_observation',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        closed=[];position=[0];opened=[];sid='S-1-5-21-1-2-3-1001'
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def entry(snapshot,out):
            if position[0]>=2:return False
            out._obj.pid=100+position[0];out._obj.parent=55;out._obj.image='owned-worker.exe' if position[0]==0 else 'foreign-system.exe';position[0]+=1;return True
        def opened_process(rights,inherit,pid):self.assertEqual(rights,0x1000);opened.append(pid);return pid
        def token(process,access,out):self.assertEqual(access,8);out._obj.value=process+100;return True
        def image(process,flags,out,size):out.value=str(self.path/'owned-worker.exe');return True
        kernel=SimpleNamespace(CreateToolhelp32Snapshot=Function(lambda flags,pid:99),Process32FirstW=Function(entry),Process32NextW=Function(entry),OpenProcess=Function(opened_process),CloseHandle=Function(lambda handle:closed.append(handle.value if hasattr(handle,'value') else handle)),QueryFullProcessImageNameW=Function(image))
        security=SimpleNamespace(OpenProcessToken=Function(token));dlls={'kernel32':kernel,'advapi32':security}
        with patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(C,'get_last_error',return_value=18,create=True),patch.object(module,'account_sid',side_effect=lambda token:sid if token.value==200 else 'S-1-5-18'):
            proof=module.owned_actor_processes(sid)
        self.assertEqual([item['pid'] for item in proof],[100]);self.assertFalse(proof[0]['machine_state_modified']);self.assertEqual(opened,[100,101]);self.assertEqual(closed,[200,100,201,101,99])
    def test_qualifier_profile_privileges_restore_previous_state_on_failure(self):
        import ctypes as C
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_profile_privileges',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):
                with module.qualifier_profile_privileges():pass
        events=[];last_error=[0]
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def token(process,access,out):self.assertEqual(access,0x28);out._obj.value=99;return True
        def lookup(system,name,luid):events.append(name);return True
        def adjust(handle,disable,requested,size,previous,needed):
            self.assertEqual(handle.value,99)
            if previous is not None:
                self.assertEqual(requested._obj.count,len(names));previous._obj.count=len(names)
                for index in range(len(names)):previous._obj.entries[index].attributes=index*2
                events.append('enable-own-token')
            else:
                self.assertEqual(requested._obj.count,len(names))
                for index in range(len(names)):self.assertEqual(requested._obj.entries[index].attributes,index*2)
                events.append('restore-exact-own-token')
            return True
        kernel=SimpleNamespace(GetCurrentProcess=Function(lambda:1),CloseHandle=Function(lambda handle:events.append('close-own-token')))
        security=SimpleNamespace(OpenProcessToken=Function(token),LookupPrivilegeValueW=Function(lookup),AdjustTokenPrivileges=Function(adjust));dlls={'kernel32':kernel,'advapi32':security}
        for names in (('SeBackupPrivilege','SeRestorePrivilege'),('SeAssignPrimaryTokenPrivilege','SeIncreaseQuotaPrivilege')):
            events.clear()
            with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'account_sid',return_value='S-1-5-18'),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(C,'set_last_error',side_effect=lambda value:last_error.__setitem__(0,value),create=True),patch.object(C,'get_last_error',side_effect=lambda:last_error[0],create=True):
                with self.assertRaisesRegex(ValueError,'profile API failure'):
                    with module.qualifier_profile_privileges(names) as scope:
                        self.assertFalse(scope['ordinary_actor_token_changed']);raise ValueError('profile API failure')
            self.assertEqual(events,[*names,'enable-own-token','restore-exact-own-token','close-own-token'])
    def test_native_recorder_probe_retains_exit_and_never_substitutes_elected_receipt(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_native_probe',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.native_recorder_probe(None)
        directory=self.path/'native-recorder-fixture';directory.mkdir();(directory/'insonic-desktop.exe').write_bytes(b'exact GUI native reporter bytes');calls=[]
        class Process:
            pid=734
            def __init__(self,command,**options):calls.append((command,options))
            def __enter__(self):return self
            def __exit__(self,*args):pass
            def wait(self,timeout):return 3221225794
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module.subprocess,'CREATE_NO_WINDOW',0x08000000,create=True),patch.object(module.subprocess,'Popen',side_effect=Process):proof=module.native_recorder_probe(directory)
        self.assertEqual(proof['exit_code'],3221225794);self.assertEqual(proof['pid'],734);self.assertFalse(proof['elected_launch_proof']);self.assertEqual((directory/'owned-native-recorder-probe.exe').read_bytes(),(directory/'insonic-desktop.exe').read_bytes());self.assertEqual(calls[0][1]['creationflags'],0x08000000);self.assertNotEqual(calls[0][0][0],str(directory/'insonic-desktop.exe'));self.assertFalse((directory/'insonic-desktop.exe.token.json').exists())
        (directory/'insonic-desktop.exe.diagnostic.json').write_bytes(b'{"stage":"actual native failure"}');temporary=self.path/'owned-temp';temporary.mkdir()
        with patch.dict(os.environ,{'TEMP':str(temporary)}):module.retain_launcher_diagnostics(directory,self.path/'context.json')
        self.assertEqual((self.path/'context.launcher-insonic-desktop.exe.diagnostic.json').read_bytes(),b'{"stage":"actual native failure"}');self.assertFalse((self.path/'context.launcher-insonic-desktop.exe').exists())
    def test_disposable_server_policy_restores_raw_value_and_absence_on_failure(self):
        from types import SimpleNamespace
        import types
        spec=importlib.util.spec_from_file_location('qualify_ci_policy',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):
                with module.disposable_server_installer_policy():pass
        path=r'Software\Policies\Microsoft\Windows\Installer'
        for original in (None,(1,4),('original typed value',1)):
            with self.subTest(original=original):
                keys={(1,'Software'):{},(1,r'Software\Policies'):{},(1,r'Software\Policies\Microsoft'):{},(1,r'Software\Policies\Microsoft\Windows'):{},(2,path):{'AlwaysInstallElevated':(0,4)}}
                if original is not None:keys[(1,path)]={'DisableMSI':original,'unrelated':('keep',1)}
                before={key:dict(value) for key,value in keys.items()};writes=[]
                class Key:
                    def __init__(self,root,path):self.root=root;self.path=path
                    def __enter__(self):return self
                    def __exit__(self,*args):pass
                def opened(root,path,*args):
                    if (root,path) not in keys:raise FileNotFoundError()
                    return Key(root,path)
                def created(root,path,*args):keys.setdefault((root,path),{});return Key(root,path)
                def query(key,name):
                    if name not in keys[(key.root,key.path)]:raise FileNotFoundError()
                    return keys[(key.root,key.path)][name]
                def write(key,name,reserved,kind,value):writes.append(name);keys[(key.root,key.path)][name]=(value,kind)
                def info(key):return (sum(root==key.root and child.startswith(key.path+'\\') for root,child in keys),len(keys[(key.root,key.path)]),0)
                registry=SimpleNamespace(HKEY_LOCAL_MACHINE=1,HKEY_CURRENT_USER=2,KEY_WOW64_64KEY=256,KEY_READ=1,KEY_WRITE=2,REG_DWORD=4,OpenKey=opened,CreateKeyEx=created,QueryValueEx=query,SetValueEx=write,DeleteValue=lambda key,name:keys[(key.root,key.path)].pop(name),QueryInfoKey=info,DeleteKeyEx=lambda root,path,*args:keys.pop((root,path)))
                evidence=None;events=[]
                with patch.object(m,'disposable_runner',return_value=None),patch.object(module.sys,'getwindowsversion',return_value=SimpleNamespace(product_type=3,build=20348),create=True),patch.dict(sys.modules,{'winreg':registry}):
                    with self.assertRaisesRegex(ValueError,'real native failure'):
                        with module.disposable_server_installer_policy(lambda phase,**data:events.append(phase)) as evidence:
                            self.assertEqual(keys[(1,path)]['DisableMSI'],(0,4));raise ValueError('real native failure')
                self.assertEqual(keys,before);self.assertTrue(evidence['restored']);self.assertTrue(evidence['always_install_elevated_unchanged']);self.assertFalse(evidence['product_changes_endpoint_policy']);self.assertNotIn('AlwaysInstallElevated',writes);self.assertEqual(events[-1],'CI-server-policy-restored')
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module.sys,'getwindowsversion',return_value=SimpleNamespace(product_type=1,build=20348),create=True),patch.dict(sys.modules,{'winreg':registry}):
            with self.assertRaisesRegex(RuntimeError,'Server2022'):
                with module.disposable_server_installer_policy():pass
    def test_bootstrap_failure_retention_does_not_overwrite_actual_worker_failure(self):
        import shutil
        spec=importlib.util.spec_from_file_location('qualify_failure_paths',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        output=self.path/'context.json';module.worker_failure_path(output).write_bytes(b'actual quiet MSI exit1625');module.worker_failure_path(output,True).write_bytes(b'bootstrap observed child exit1')
        native=self.path/'native-failures';native.mkdir()
        shutil.copyfile(module.worker_failure_path(output),native/'context.failure.json');shutil.copyfile(module.worker_failure_path(output,True),native/'context.broker-failure.json')
        module.retain_ordinary_diagnostics(native,self.path/'ordinary-first-context.json')
        self.assertEqual((self.path/'ordinary-first-context.failure.json').read_bytes(),b'actual quiet MSI exit1625');self.assertEqual((self.path/'ordinary-first-context.broker-failure.json').read_bytes(),b'bootstrap observed child exit1')
    def test_task_engine_cleanup_rejects_foreign_sid_and_closes_only_verified_owned_host(self):
        import ctypes as C
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_owned_engine_cleanup',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.stop_owned_task_engine(None,None)
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.inherited_ordinary_desktop(None)
        sid='S-1-5-21-1-2-3-1001';system=self.path/'Windows';calls=[];closed=[]
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def token(process,access,out):out._obj.value=200;return True
        def image(process,flags,out,size):out.value=str(system/'System32/taskeng.exe');return True
        kernel=SimpleNamespace(OpenProcess=Function(lambda rights,inherit,pid:300),QueryFullProcessImageNameW=Function(image),WaitForSingleObject=Function(lambda handle,milliseconds:258 if milliseconds==0 else 0),TerminateProcess=Function(lambda handle,code:calls.append(handle) or True),CloseHandle=Function(lambda handle:closed.append(handle.value if hasattr(handle,'value') else handle)))
        security=SimpleNamespace(OpenProcessToken=Function(token));dlls={'kernel32':kernel,'advapi32':security}
        with patch.dict(os.environ,{'SystemRoot':str(system)}),patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(module,'account_sid',return_value='S-1-5-18'):
            with self.assertRaisesRegex(RuntimeError,'another actor'):module.stop_owned_task_engine(777,sid)
        self.assertEqual(calls,[]);self.assertEqual(closed,[200,300]);closed.clear()
        with patch.dict(os.environ,{'SystemRoot':str(system)}),patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(module,'account_sid',return_value=sid):
            result=module.stop_owned_task_engine(777,sid)
        self.assertEqual(calls,[300]);self.assertTrue(result['terminated']);self.assertFalse(result['shared_system_process_stopped']);self.assertEqual(closed,[200,300])
    def test_existing_ci_account_is_never_reused_or_deleted(self):
        import ctypes as C
        import secrets
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_ordinary_collision',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        calls=[]
        class Function:
            def __init__(self,name,result=0):self.name=name;self.result=result
            def __call__(self,*args):calls.append(self.name);return self.result
        kernel=SimpleNamespace(**{name:Function(name) for name in ('GetCurrentProcess','CloseHandle','WaitForSingleObject','GetExitCodeProcess','TerminateProcess')})
        security=SimpleNamespace(**{name:Function(name) for name in ('OpenProcessToken','LogonUserW','CreateProcessWithLogonW')})
        network=SimpleNamespace(NetUserAdd=Function('NetUserAdd',2224),NetUserDel=Function('NetUserDel'),NetLocalGroupAddMembers=Function('NetLocalGroupAddMembers'))
        profiles=SimpleNamespace(GetUserProfileDirectoryW=Function('GetUserProfileDirectoryW'),DeleteProfileW=Function('DeleteProfileW'),LoadUserProfileW=Function('LoadUserProfileW'),UnloadUserProfile=Function('UnloadUserProfile'));dlls={'kernel32':kernel,'advapi32':security,'netapi32':network,'userenv':profiles}
        output=self.path/'ordinary.json'
        with patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(secrets,'token_hex',return_value='private-password-material'):
            with self.assertRaisesRegex(RuntimeError,'no existing account'):module.ordinary_context(None,None,output,None)
        self.assertEqual(calls,['NetUserAdd']);self.assertNotIn('private-password-material',output.with_suffix('.startup.json').read_text(encoding='utf-8'));self.assertNotIn('private-password-material',output.with_suffix('.cleanup.json').read_text(encoding='utf-8'))
    def test_ordinary_cleanup_removes_only_created_profile_account_and_stage(self):
        import ctypes as C
        import shutil
        import re
        from contextlib import nullcontext
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_ordinary_success_cleanup',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        users=self.path/'users';users.mkdir();original=users/'original';original.mkdir();(original/'owner-data').write_bytes(b'keep')
        programdata=self.path/'programdata';programdata.mkdir();(programdata/'owner-data').write_bytes(b'keep')
        fixture=self.path/'launcher-context-fixture';fixture.mkdir();(fixture/'precompiled-reporter.exe').write_bytes(b'qualified fixture bytes')
        installer=self.path/'ordinary-source.msi';installer.write_bytes(b'qualified MSI fixture bytes');output=self.path/'ordinary.json'
        actor_sid='S-1-5-21-1-2-3-1001';events=[];state={};deletions=[];rights=[]
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def load_profile(token,info):
            state['name']=info._obj.name;state['profile']=users/state['name'];state['profile'].mkdir();info._obj.hive=666;events.append('profile-created-by-native-load');return True
        def unload_profile(token,hive):
            self.assertEqual(token.value,100);self.assertEqual(hive,666);events.append('profile-handle-unloaded');return True
        def profile_directory(token,value,size):value.value=str(state['profile']);return True
        def delete_profile(sid,path,remote):
            self.assertEqual(sid,actor_sid);target=Path(path);self.assertEqual(target.resolve().parent,users.resolve());self.assertEqual(target.name,state['name']);deletions.append(target)
            if len(deletions)<3:return False
            shutil.rmtree(target);events.append('profile-deleted');return True
        def create_process(name,domain,password,flags,exe,command,creation,environment,cwd,startup,process):
            self.assertNotIn(password,command.value);self.assertEqual(flags,1)
            if environment is None:
                self.assertTrue(str(exe).endswith('whoami.exe'));state['name']=name;state['profile']=users/name;state['profile'].mkdir();events.append('profile-created-by-logon')
            else:
                self.assertEqual(name,state['name']);target=Path(re.search(r'--output ("[^"]+"|\S+)',command.value).group(1).strip('"'))
                target.write_text(json.dumps({'token':{'user_sid_sha256':m.digest(actor_sid.encode())},'user_profile':str(state['profile']),'elected_launcher':{'actual_elected_launch':True},'os_policy_mutations':{}}),encoding='utf-8')
            process._obj.process=333;process._obj.thread=444;process._obj.pid=555;return True
        def set_token(process,access,out):out._obj.value=200;return True
        def logon(name,domain,password,kind,provider,out):out._obj.value=100;return True
        def exit_code(process,out):out._obj.value=0;return True
        kernel=SimpleNamespace(GetCurrentProcess=Function(lambda:1),CloseHandle=Function(lambda handle:True),WaitForSingleObject=Function(lambda *args:0),GetExitCodeProcess=Function(exit_code),TerminateProcess=Function(lambda *args:True))
        security=SimpleNamespace(OpenProcessToken=Function(set_token),LogonUserW=Function(logon),CreateProcessWithLogonW=Function(create_process))
        network=SimpleNamespace(NetUserAdd=Function(lambda *args:events.append('account-created') or 0),NetUserDel=Function(lambda server,name:events.append('account-deleted') or 0),NetLocalGroupAddMembers=Function(lambda *args:0))
        profiles=SimpleNamespace(GetUserProfileDirectoryW=Function(profile_directory),DeleteProfileW=Function(delete_profile),LoadUserProfileW=Function(load_profile),UnloadUserProfile=Function(unload_profile));dlls={'kernel32':kernel,'advapi32':security,'netapi32':network,'userenv':profiles}
        env={'USERPROFILE':str(original),'ProgramData':str(programdata),'SystemRoot':r'C:\Windows','COMPUTERNAME':'CI-FIXTURE'}
        def right(sid,change=None):
            self.assertEqual(sid,actor_sid)
            if change=='add':rights.append('SeBatchLogonRight');events.append('owned-batch-added')
            if change=='remove':rights.remove('SeBatchLogonRight');events.append('owned-batch-restored')
            return list(rights)
        def scheduled(stage,name,sid,password,args,record,profile):
            self.assertEqual(name,state['name']);self.assertEqual(sid,actor_sid);self.assertNotIn(password,repr(args));events.append('ordinary-task-completed')
            target=Path(args[args.index('--output')+1]);target.write_text(json.dumps({'token':{'user_sid_sha256':m.digest(actor_sid.encode())},'user_profile':str(state['profile']),'elected_launcher':{'actual_elected_launch':True},'os_policy_mutations':{}}),encoding='utf-8')
            for action,suffix in (('/i','.install.worker.json'),('/x','.remove.worker.json')):target.with_suffix(suffix).write_text(json.dumps({'kind':'native-installer-worker','phase':'completed','exit_code':0,'action':action,'service':{'pid':42,'state':4}}),encoding='utf-8')
            return {'phase':'completed','exit_code':0,'deleted':True}
        ownership={'state':1,'pid':0,'binary_identity_valid':True,'canonical_binary':'exact-CI-msiexec'}
        def release(initial,install,remove,task,sid,deadline):
            self.assertEqual(initial,ownership);self.assertEqual(sid,actor_sid);self.assertEqual(task['exit_code'],0);self.assertEqual(json.loads(install.read_text())['action'],'/i');self.assertEqual(json.loads(remove.read_text())['action'],'/x');events.append('owned-service-restored');return {'restored_initial_stopped_state':True,'process_terminated':False,'registry_hive_forced':False}
        def lease(sid,profile,deadline):events.append('profile-lease-observed');return {'machine_state_modified':False}
        with __import__('contextlib').ExitStack() as patches:
            for context in (patch.dict(os.environ,env),patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(module,'account_sid',side_effect=[actor_sid,'S-1-5-21-1-2-3-1000']),patch.object(module,'grant_owned_account_stage'),patch.object(module,'qualifier_profile_privileges',side_effect=lambda:nullcontext({'ordinary_actor_token_changed':False})),patch.object(module,'owned_batch_right',side_effect=right),patch.object(module,'scheduler_ordinary',side_effect=scheduled),patch('qualify_installer_ui.prepare'),patch.object(module,'interactive_context_desktop',return_value=nullcontext({'station_dacl_restored':True})),patch.object(module,'interactive_ordinary'),patch.object(module,'interactive_context_namespace',return_value=nullcontext({'security_restored':True})),patch.object(module,'object_namespace_diagnostic',return_value={'objects':[]}),patch.object(module,'token_info',return_value={}),patch.object(module,'installer_mutex_diagnostic',return_value={'objects':[]}),patch.object(module,'disposable_server_installer_policy',return_value=nullcontext({'restored':True,'always_install_elevated_unchanged':True})),patch.object(module,'profile_cleanup_observation',return_value={'machine_state_modified':False}),patch.object(module,'profile_lease_observation',side_effect=lease),patch.object(module,'release_owned_installer_service',side_effect=release),patch.object(module,'delete_owned_profile_provider',return_value={'complete_absence':False}),patch.object(C,'get_last_error',return_value=32,create=True),patch.object(module.time,'sleep')):patches.enter_context(context)
            proof=module.ordinary_context(installer,self.path/'installation',output,self.model,ownership)
        self.assertEqual(events,['account-created','profile-created-by-native-load','ordinary-task-completed','profile-handle-unloaded','profile-lease-observed','owned-service-restored','profile-lease-observed','profile-deleted','account-deleted']);self.assertEqual(len(deletions),3);self.assertEqual(rights,[])
        self.assertTrue(all(item.get('deleted',item.get('restored',item.get('unloaded'))) for item in proof['cleanup']));self.assertEqual((original/'owner-data').read_bytes(),b'keep');self.assertEqual((programdata/'owner-data').read_bytes(),b'keep')
        self.assertEqual(list(programdata.iterdir()),[programdata/'owner-data']);self.assertEqual(installer.read_bytes(),b'qualified MSI fixture bytes')
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
    def test_restricted_capture_sets_and_verifies_real_session_elections(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('capture_restricted_fixture',ROOT/'scripts/capture-installer-ui.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            for function,args in [(module.worker,(None,None,None)),(module.capture,(None,None)),(module.initialize_restricted_session,(None,None,None))]:
                with self.assertRaisesRegex(RuntimeError,'disposable'):function(*args)
        calls=[];properties={}
        def set_property(session,name,value):calls.append(('property',name));properties[name]=value;return 0
        def feature(session,name,installed,planned):installed._obj.value=2;planned._obj.value=3;return 0
        dll=SimpleNamespace(MsiSetPropertyW=set_property,MsiDoActionW=lambda session,action:calls.append(('action',action)) or 0,MsiSetInstallLevel=lambda session,level:calls.append(('level',level)) or 0,MsiSetFeatureStateW=lambda session,name,state:calls.append(('feature',name,state)) or 0,MsiGetFeatureStateW=feature)
        with patch.object(m,'disposable_runner',return_value=None):
            proof=module.initialize_restricted_session(dll,1,lambda name:properties.get(name,''))
            self.assertFalse(proof['machine_state_changes_permitted']);self.assertFalse(proof['executable_custom_actions_permitted']);self.assertFalse(proof['license_accepted'])
            self.assertEqual([call[1] for call in calls if call[0]=='action'],['CostInitialize','FileCost','CostFinalize'])
            self.assertLess(calls.index(('level',1)),calls.index(('feature','Core',3)))
            self.assertEqual({name:value['action'] for name,value in proof['features'].items()},{'Core':3,'Desktop':3,'CLI':3})
            dll.MsiGetFeatureStateW=lambda session,name,installed,planned:0
            with self.assertRaisesRegex(RuntimeError,'did not elect'):module.initialize_restricted_session(dll,1,lambda name:properties.get(name,''))
    def test_native_directory_control_uses_verified_value_without_class_guess(self):
        spec=importlib.util.spec_from_file_location('capture_directory_value',ROOT/'scripts/capture-installer-ui.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        values={10:'Choose program directory',20:module.FIXTURE_DIRECTORY+'\\',30:'Browse...'}
        self.assertEqual(module.directory_control(values,values.__getitem__),20)
        values[40]=module.FIXTURE_DIRECTORY
        with self.assertRaisesRegex(RuntimeError,'expected one'):module.directory_control(values,values.__getitem__)
    def test_all_native_gates_collect_both_failures_without_publication(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('package_msi_gate_errors',ROOT/'scripts/package-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        called=[];receipt={'qualification':{key:'pending' for key in m.QUALIFICATION_KEYS}}
        def capture(*args):called.append('capture');raise RuntimeError('actual renderer failed')
        def qualify(*args):called.append('qualify');raise RuntimeError('actual context failed')
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'load',side_effect=[SimpleNamespace(qualify=qualify),SimpleNamespace(capture=capture)]),patch.object(module,'write'),patch.object(m,'verify_installer') as verify:
            with self.assertRaisesRegex(RuntimeError,'gates failed'):module.run_gates(None,receipt,'all')
            self.assertEqual(called,['qualify','capture']);verify.assert_not_called()
            self.assertEqual(receipt['qualification']['native_ui_capture'],'failed');self.assertEqual(receipt['qualification']['user_context'],'pending')
            self.assertEqual(set(receipt['qualification_errors']),{'native_ui_capture','native_lifecycle'})
    def test_provisional_all_native_pass_still_requires_qualified_zip_join(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('package_msi_provisional',ROOT/'scripts/package-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        called=[];artifact=self.path/'provisional.msi';artifact.write_bytes(b'byte-identity fixture')
        receipt={'kind':'windows-installer-receipt','format':'msi','platform':'windows_amd64','variant':'selectable','schema_version':'1.0.1','product_version':'1.0.1','documentation_version':'1.0.1','artifact':artifact.name,'artifact_sha256':m.sha(artifact),'artifact_size_bytes':artifact.stat().st_size,'source_dirty':False,'revision':'a'*40,'payload_receipt':dict(self.inv),'payload_inventory':self.inv,'qualification':{key:'pending' for key in m.QUALIFICATION_KEYS}}
        capture=SimpleNamespace(capture=lambda *args:called.append('native-ui'))
        qualifier=SimpleNamespace(qualify=lambda *args:called.append('actual-lifecycle') or {key:'passed' for key in m.QUALIFICATION_KEYS if key!='native_ui_capture'})
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'load',side_effect=[qualifier,capture]),patch.object(module,'write'),patch.object(m,'verify_installer') as verify:
            module.run_gates(artifact,receipt,'provisional');verify.assert_not_called()
        self.assertEqual(called,['actual-lifecycle','native-ui']);self.assertTrue(all(value=='passed' for value in receipt['qualification'].values()))
        self.assertIs(receipt['publication_eligible'],False);self.assertEqual(receipt['payload_qualification'],'pending')
        with self.assertRaisesRegex(ValueError,'provisional'):m.verify_installer(artifact,receipt)
        # Changing flags cannot fabricate the missing ZIP's actual relocation,
        # application/native-webview and ordinary archive installation evidence.
        receipt.update({'publication_eligible':True,'payload_qualification':'passed'})
        with self.assertRaisesRegex(ValueError,'package relocation and native qualification'):m.verify_installer(artifact,receipt)
    def test_native_capture_retries_only_one_session_initialization_failure(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('package_msi_capture_retry',ROOT/'scripts/package-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        transient=RuntimeError('native authored capture failed: native session initialization exceeded its bound\r\n')
        for errors,passes,count in [([transient,None],True,2),([transient,transient],False,2),([RuntimeError('native page landmark is absent')],False,1)]:
            calls=[];receipt={'qualification':{key:'pending' for key in m.QUALIFICATION_KEYS}}
            def capture(*args):
                error=errors[len(calls)];calls.append('capture')
                if error:raise error
            with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'load',return_value=SimpleNamespace(capture=capture)),patch.object(module,'write'):
                if passes:module.run_gates(None,receipt,'capture')
                else:
                    with self.assertRaisesRegex(RuntimeError,'gates failed'):module.run_gates(None,receipt,'capture')
            self.assertEqual(len(calls),count)
            self.assertEqual(receipt['qualification']['native_ui_capture'],'passed' if passes else 'failed')
            self.assertTrue(all(value=='pending' for key,value in receipt['qualification'].items() if key!='native_ui_capture'))
    def test_diagnostic_reports_failed_feature_add_without_qualification(self):
        import shutil
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('msi_diagnostic_observations',ROOT/'scripts/diagnose-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        with patch.dict(os.environ,{},clear=True):
            with self.assertRaisesRegex(RuntimeError,'disposable'):module.run(None,None,None)
        scripts=self.path/'scripts';scripts.mkdir()
        for name in ('diagnose-windows-msi.py','qualify-windows-msi.py','windows_msi.py'):shutil.copyfile(ROOT/'scripts'/name,scripts/name)
        name='insonic_1.0.1_windows_amd64_desktop';archive=self.path/(name+'.zip');archive.write_bytes(b'exact qualified archive fixture')
        receipt={'kind':'native-package-receipt','platform':'windows_amd64','variant':'desktop','schema_version':'1.0.1','dirname':name,'archive':archive.name,'revision':'a'*40,'source_dirty':False,'inventory_sha256':m.digest(m.inventory_bytes(self.inv)),'signing':self.inv['signing']};receipt_path=self.path/'diagnostic-input.json';receipt_path.write_text(json.dumps(receipt),encoding='utf-8')
        calls=[]
        def extract(archive,destination):fixture(destination/name)
        def install(args,log):
            if args[0]=='/x':calls.append('cleanup');return
            if 'REINSTALL=Core' in args:calls.append('feature-failed');raise RuntimeError('observed native 1603')
            calls.append('installed')
        qualifier=SimpleNamespace(service_diagnostic=lambda **kwargs:{'state':1,'pid':0,'binary_identity_valid':True,'canonical_binary':'fixture-native-installer'},clone=lambda source,target,*args:shutil.copyfile(source,target),run_installer=install,installed_hashes=lambda *args:None,registered_feature_states=lambda *args:{'Core':3,'CLI':3,'Desktop':2},prepare_launcher_fixture=lambda *args:None,ordinary_context=lambda *args:calls.append('ordinary-first-observed' if Path(args[0]).name=='ordinary-first.msi' else 'ordinary-after-observed') or {'separate_user_account':True},token_info=lambda:{'session_id':1,'administrator_enabled':True},installer_mutex_diagnostic=lambda:{'machine_state_modified':False})
        package=SimpleNamespace(verify_archive_receipt=lambda *args,**kwargs:{'inventory':self.inv},extract_package=extract,verify_inventory=lambda *args:None)
        release=SimpleNamespace(verify_qualification=lambda *args:None,verify_signing=lambda *args:None)
        with patch.object(m,'disposable_runner',return_value=None),patch.object(m,'ROOT',self.path),patch.object(m,'author',side_effect=lambda path,*args:path.write_bytes(b'authored diagnostic fixture')),patch.object(module,'load',side_effect=[release,package,qualifier]),patch.object(module,'git_identity',side_effect=[{'revision':'a'*40,'dirty':False},{'revision':'b'*40,'dirty':False}]):
            report=module.run(archive,receipt_path,self.path/'prior-source',True,True)
        self.assertEqual(calls,['ordinary-first-observed','installed','feature-failed','cleanup','ordinary-after-observed'])
        self.assertTrue(report['diagnostic_only']);self.assertTrue(report['ineligible_for_publication']);self.assertFalse(report['qualification_pass']);self.assertEqual(report['observations']['feature_add']['outcome'],'observed-failure')
        self.assertNotEqual(report['payload_source']['revision'],report['tool_source']['revision']);self.assertFalse((self.path/'build/native/windows-installer-receipt.json').exists())
        identity=report['observations']['ordinary_first_fixture_identity'];self.assertTrue(identity['before_any_elevated_install_or_capture']);self.assertFalse(identity['publication_eligible'])
        self.assertNotEqual(identity['product_code'],report['observations']['fixture_identity']['product_code']);self.assertNotEqual(identity['upgrade_code'],report['observations']['fixture_identity']['upgrade_code'])
    def test_diagnostic_upgrade_hash_failure_cleans_new_product_and_observes_ordinary_actor(self):
        import shutil
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('msi_diagnostic_upgrade_failure',ROOT/'scripts/diagnose-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        scripts=self.path/'scripts';scripts.mkdir()
        for name in ('diagnose-windows-msi.py','qualify-windows-msi.py','windows_msi.py'):shutil.copyfile(ROOT/'scripts'/name,scripts/name)
        name='insonic_1.0.1_windows_amd64_desktop';archive=self.path/(name+'.zip');archive.write_bytes(b'qualified archive fixture')
        receipt={'kind':'native-package-receipt','platform':'windows_amd64','variant':'desktop','schema_version':'1.0.1','dirname':name,'archive':archive.name,'revision':'a'*40,'source_dirty':False,'inventory_sha256':m.digest(m.inventory_bytes(self.inv)),'signing':self.inv['signing']};receipt_path=self.path/'diagnostic-input.json';receipt_path.write_text(json.dumps(receipt),encoding='utf-8')
        calls=[];identities={};upgraded=False
        def clone(source,target,version,upgrade):shutil.copyfile(source,target);identities[target.name]=m.product_code(version,upgrade)
        def install(args,log):
            nonlocal upgraded
            if args[0]=='/x':calls.append(('cleanup',args[1]));return
            if Path(args[1]).name=='upgraded.msi':upgraded=True;calls.append(('upgrade-api-success',None))
        def hashes(*args):
            if upgraded:raise ValueError('installed MSI bytes differ: missing versioned owned file')
        qualifier=SimpleNamespace(service_diagnostic=lambda **kwargs:{'state':1,'pid':0,'binary_identity_valid':True,'canonical_binary':'fixture-native-installer'},clone=clone,run_installer=install,installed_hashes=hashes,registered_feature_states=lambda *args:{'Core':3,'CLI':3,'Desktop':2},prepare_launcher_fixture=lambda *args:None,ordinary_context=lambda *args:calls.append(('ordinary-observed',None)) or {'separate_user_account':True})
        package=SimpleNamespace(verify_archive_receipt=lambda *args,**kwargs:{'inventory':self.inv},extract_package=lambda archive,destination:fixture(destination/name),verify_inventory=lambda *args:None)
        release=SimpleNamespace(verify_qualification=lambda *args:None,verify_signing=lambda *args:None)
        with patch.object(m,'disposable_runner',return_value=None),patch.object(m,'ROOT',self.path),patch.object(m,'author',side_effect=lambda path,*args:path.write_bytes(b'diagnostic fixture')),patch.object(module,'load',side_effect=[release,package,qualifier]),patch.object(module,'git_identity',side_effect=[{'revision':'a'*40,'dirty':False},{'revision':'b'*40,'dirty':False}]):
            report=module.run(archive,receipt_path,self.path/'prior-source',True)
        self.assertEqual(calls,[('upgrade-api-success',None),('cleanup',identities['upgraded.msi']),('ordinary-observed',None)])
        self.assertEqual(report['observations']['major_upgrade']['outcome'],'observed-failure');self.assertEqual(report['observations']['major_upgrade']['cleanup_product_code'],identities['upgraded.msi'])
        self.assertTrue(report['ineligible_for_publication']);self.assertFalse(report['qualification_pass'])
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
            with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'native_installer_dll',return_value=fake),patch.object(module,'service_diagnostic',return_value={'pid':0,'state':1}),patch.object(module,'installer_mutex_diagnostic',return_value={'objects':[],'machine_state_modified':False}):module.native_installer_worker(request)
            evidence=json.loads(log.with_suffix('.worker.json').read_text(encoding='utf-8'))
            self.assertEqual(evidence['phase'],'completed');self.assertEqual(evidence['exit_code'],expected_code);self.assertIs(evidence['captured_pipes'],False)
            if state is not None:self.assertEqual(calls[-1][1][2],state)
            if action=='/fa':self.assertIn('REINSTALLMODE="amus"',calls[-1][1][-1])
    def test_exact_mutex_observation_preserves_access_denial_and_never_acquires_or_mutates(self):
        import ctypes as C
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('qualify_readonly_mutex',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        calls=[];state={'error':0};closed=[]
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
        def open_mutex(access,inherit,name):
            self.assertIs(inherit,False);calls.append((name,access))
            if name!='Global\\_MSIExecute':state['error']=2;return 0
            if access==0x1f0001:state['error']=5;return 0
            return access
        def descriptor(handle,kind,information,*args):
            self.assertEqual(kind,6);self.assertEqual(information,7);args[-1]._obj.value=1234;return 0
        kernel=SimpleNamespace(OpenMutexW=Function(open_mutex),CloseHandle=Function(lambda handle:closed.append(handle) or True),LocalFree=Function(lambda value:None))
        security=SimpleNamespace(GetSecurityInfo=Function(descriptor));dlls={'kernel32':kernel,'advapi32':security}
        with patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(C,'get_last_error',side_effect=lambda:state['error'],create=True),patch.object(module,'descriptor_observation',return_value={'owner_group_dacl_label_sddl':'O:SYS:(ML;;NW;;;HI)'}) as observed:
            result=module.installer_mutex_diagnostic()
        self.assertEqual(len(calls),15);self.assertEqual(closed,[0x100000,0x20000]);observed.assert_called_once()
        denied=next(row for row in result['objects'] if row['name']=='Global\\_MSIExecute' and row['requested_access']==0x1f0001)
        self.assertFalse(denied['opened']);self.assertEqual(denied['error'],5);self.assertFalse(result['mutex_acquired']);self.assertFalse(result['machine_state_modified'])
    def test_intentional_rejection_requires_message_and_correct_action(self):
        spec=importlib.util.spec_from_file_location('qualify_rejection_proof',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        log=self.path/'rejected.log'
        log.write_text('DEBUG: Error 2228: Unknown table Signature\nAction ended 3:12:57: AppSearch. Return value 3.\n',encoding='utf-16')
        with self.assertRaisesRegex(RuntimeError,'intended'):module.assert_rejection(log,'Read and accept Apache License 2.0')
        log.write_text('Read and accept Apache License 2.0\nAction ended 3:12:57: AppSearch. Return value 3.\n',encoding='utf-16')
        with self.assertRaisesRegex(RuntimeError,'intended'):module.assert_rejection(log,'Read and accept Apache License 2.0')
        log.write_text('Read and accept Apache License 2.0\nAction ended 3:12:57: LaunchConditions. Return value 3.\n',encoding='utf-16')
        self.assertEqual(module.assert_rejection(log,'Read and accept Apache License 2.0'),'Read and accept Apache License 2.0')
    def test_required_empty_search_table_definitions_are_authored(self):
        for table in ('Signature','CompLocator','IniLocator','DrLocator'):
            self.assertIn(table,self.model['tables']);self.assertEqual(self.model['tables'][table],[])
    def test_alternate_directory_proof_rejects_registry_override(self):
        spec=importlib.util.spec_from_file_location('qualify_directory_proof',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        log=self.path/'alternate-directory.log';target=r'C:\fixture spaces é\alternate'
        log.write_text("PROPERTY CHANGE: Adding INSTALLDIR property. Its value is '"+target+"'.\n",encoding='utf-16')
        module.assert_effective_directory(log,target+'\\')
        log.write_text("PROPERTY CHANGE: Adding INSTALLDIR property. Its value is '"+target+"'.\nPROPERTY CHANGE: Modifying INSTALLDIR property. Its new value is 'C:\\registered old directory'.\n",encoding='utf-16')
        with self.assertRaisesRegex(ValueError,'did not retain'):module.assert_effective_directory(log,target)
    def test_bad_compound_and_cabinet_rejected(self):
        p=self.path/'invalid.msi';p.write_bytes(b'not an MSI')
        with self.assertRaises(ValueError):m.CompoundFile(p)
        with self.assertRaises(ValueError):m.cabinet_members(b'not a cabinet')
    def test_exact_pe_file_version_and_language_are_read_without_loading(self):
        data=versioned_pe_fixture();self.assertEqual(m.pe_file_metadata(data),('1.6.0.2','1033'))
        self.assertEqual(m.pe_file_metadata(b'ordinary text containing '+struct.pack('<I',0xfeef04bd)),(None,None))
        unversioned=bytearray(data);struct.pack_into('<II',unversioned,152+128,0,0)
        self.assertEqual(m.pe_file_metadata(unversioned),(None,None))
        for offset,value in [(600,0xffff),(512+72,0xfffff000),(512+48+20,0x80000048)]:
            damaged=bytearray(data);struct.pack_into('<I',damaged,offset,value)
            with self.assertRaises(ValueError):m.pe_file_metadata(damaged)
    @unittest.skipUnless(os.name=='nt','native file database authoring only, no UI or installation')
    def test_actual_file_table_metadata_is_verified_against_embedded_pe_bytes(self):
        value=versioned_pe_fixture();(self.path/'insonic.exe').write_bytes(value)
        item=next(row for row in self.inv['files'] if row['path']=='insonic.exe');item.update({'size_bytes':len(value),'sha256':m.digest(value)})
        (self.path/'package-inventory.json').write_bytes(m.inventory_bytes(self.inv));model=m.installer_model(self.inv,m.sha(self.path/'package-inventory.json'),(ROOT/'LICENSE').read_text(encoding='utf-8'))
        path=self.path/'versioned-file.msi';m.author(path,self.path,model)
        file_id=next(row['file_id'] for row in model['payload'] if row['path']=='insonic.exe')
        rows=m.table_rows(m.CompoundFile(path).streams,'File');self.assertEqual(next(row[4:6] for row in rows if row[0]==file_id),['1.6.0.2','1033'])
        db=m.NativeDatabase(path,write=True)
        try:
            q=m.quote;db.execute('UPDATE '+q('File')+' SET '+q('Version')+'=? WHERE '+q('File')+'=?',['99.0.0.0',file_id]);db.check(db.dll.MsiDatabaseCommit(db.handle))
        finally:db.close()
        with self.assertRaisesRegex(ValueError,'File'):m.verify_model_bytes(path,model)
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
        self.assertEqual(dict(m.table_rows(streams,'AppSearch'))['SAVED_INSTALLDIR'],'SavedInstall')
        self.assertNotIn('INSTALLDIR',dict(m.table_rows(streams,'AppSearch')))
        self.assertEqual(next(r[1:] for r in m.table_rows(streams,'CustomAction') if r[0]=='RestoreInstallDir'),[51,'INSTALLDIR','[SAVED_INSTALLDIR]'])
        for table in ('InstallUISequence','InstallExecuteSequence'):
            self.assertEqual(next(r[1] for r in m.table_rows(streams,table) if r[0]=='RestoreInstallDir'),'SAVED_INSTALLDIR AND NOT INSTALLDIR')
        diagnostics=module.feature_diagnostic_rows();actual_actions=m.table_rows(streams,'CustomAction');actual_sequence=m.table_rows(streams,'InstallExecuteSequence')
        self.assertEqual(len(diagnostics['CustomAction']),42)
        for row in diagnostics['CustomAction']:self.assertIn(row,actual_actions);self.assertEqual(row[1],51)
        for row in diagnostics['InstallExecuteSequence']:self.assertIn(row,actual_sequence);self.assertGreater(row[2],1000);self.assertLess(row[2],1100)
        self.assertFalse(any(row[0].startswith('S027Diag_') for row in m.table_rows(m.CompoundFile(original).streams,'CustomAction')))
        self.assertEqual(m.cabinet_members(streams['payload.cab']),m.cabinet_members(m.CompoundFile(original).streams['payload.cab']))
    @unittest.skipUnless(os.name=='nt','native summary database inspection only, no UI')
    def test_clones_have_actual_distinct_package_codes(self):
        import ctypes as C
        import re
        spec=importlib.util.spec_from_file_location('qualify_clone_package_identity',ROOT/'scripts/qualify-windows-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        original=self.path/'package-original.msi';m.author(original,self.path,self.model)
        first=self.path/'package-first.msi';second=self.path/'package-second.msi';upgrade=m.guid('summary-code-test')
        module.clone(original,first,'1.0.2',upgrade);module.clone(original,second,'1.0.3',upgrade)
        def package_code(path):
            db=m.NativeDatabase(path);summary=C.c_uint()
            db.dll.MsiSummaryInfoGetPropertyW.argtypes=[C.c_uint,C.c_uint,C.POINTER(C.c_uint),C.POINTER(C.c_int),C.c_void_p,C.c_wchar_p,C.POINTER(C.c_uint)];db.dll.MsiSummaryInfoGetPropertyW.restype=C.c_uint
            try:
                db.check(db.dll.MsiGetSummaryInformationW(db.handle,None,0,C.byref(summary)))
                value=C.create_unicode_buffer(128);size=C.c_uint(128);kind=C.c_uint();integer=C.c_int()
                db.check(db.dll.MsiSummaryInfoGetPropertyW(summary,9,C.byref(kind),C.byref(integer),None,value,C.byref(size)))
                self.assertEqual(kind.value,30);return value.value
            finally:
                if summary.value:db.dll.MsiCloseHandle(summary)
                db.close()
        codes={package_code(path) for path in (original,first,second)}
        self.assertEqual(len(codes),3)
        for value in codes:self.assertRegex(value,r'^\{[0-9A-F]{8}(?:-[0-9A-F]{4}){3}-[0-9A-F]{12}\}$')
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
        payload.update({'archive':'desktop.zip','archive_sha256':'b'*64,'inventory_sha256':inv_sha,'variant':'desktop','extracted_inventory':'passed','relocation_path_spaces':'passed','cli_real_audio_video_import':'passed','development_native_libraries':'isolated-and-restored','windows_copy_install':'passed','windows_first_use':'passed','gui_bridge':'passed','native_webview':'passed'})
        receipt={k:self.inv[k] for k in ('revision','source_dirty','platform','schema_version','distribution')}
        receipt.update({'kind':'windows-installer-receipt','format':'msi','variant':'selectable','product_version':'1.0.1','documentation_version':'1.0.1','signing':dict(m.UNSIGNED_SIGNING),'artifact':artifact.name,'artifact_sha256':m.sha(artifact),'artifact_size_bytes':artifact.stat().st_size,'payload_receipt':payload,'payload_inventory':self.inv,'payload_archive':'desktop.zip','payload_archive_sha256':'b'*64,'qualification':{k:'passed' for k in m.QUALIFICATION_KEYS},'msi_product_version':'1.0.1','product_code':model['product_code'],'upgrade_code':model['upgrade_code'],'scope':'per-user','license_sha256':m.sha(ROOT/'LICENSE'),'icon_sha256':m.sha(ROOT/m.ICON)})
        self.assertEqual(m.verify_installer(artifact,receipt)['inventory']['signing'],self.inv['signing'])
        receipt['signing']=dict(self.inv['signing'])
        with self.assertRaisesRegex(ValueError,'independently unsigned'):m.verify_installer(artifact,receipt)
    @unittest.skipUnless(os.name=='nt','native database authoring Windows only, no UI')
    def test_native_actual_tables_and_cabinet_independently_verified(self):
        path=self.path/'fixture.msi';m.author(path,self.path,self.model);result=m.verify_model_bytes(path,self.model);self.assertEqual(len(result['members']),len(self.model['payload']))
        catalog={row[0] for row in m.table_rows(result['streams'],'_Tables')}
        self.assertTrue({'Signature','CompLocator','IniLocator','DrLocator','ActionText'}<=catalog)
        db=m.NativeDatabase(path,write=True)
        try:
            q=m.quote;db.execute('UPDATE '+q('CustomAction')+' SET '+q('Target')+'=? WHERE '+q('Action')+'=?',['--cli','LaunchDesktop']);db.check(db.dll.MsiDatabaseCommit(db.handle))
        finally:db.close()
        with self.assertRaisesRegex(ValueError,'CustomAction'):m.verify_model_bytes(path,self.model)
if __name__=='__main__':unittest.main()
