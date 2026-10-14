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
    def test_client_dependency_and_owned_marker(self):
        items={x['path']:x for x in self.model['payload']}
        self.assertEqual(items['insonic-desktop.exe']['feature'],'Desktop');self.assertEqual(items['insonic.exe']['feature'],'Core');self.assertIn(m.OWNERSHIP,items);self.assertNotIn(m.OWNERSHIP,{x['path'] for x in self.inv['files']})
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
            for function,args in [(module.token_info,()),(module.installer_mutex_diagnostic,()),(module.owned_user_object_observation,(None,None,None,None)),(module.ordinary_context,(None,None,None)),(module.ordinary_worker,(None,None,None)),(module.cancellation,(None,None,None)),(module.cancellation_worker,(None,None,None)),(module.prepare_launcher_fixture,(None,None)),(module.elected_launcher_context,(None,)),(module.linked_token_evidence,()),(module.service_diagnostic,()),(module.native_installer_worker,(None,)),(module.native_installer_dll,(None,)),(module.run_installer,(['/i','unused'],None)),(module.create_removal_sentinels,()),(module.assert_removal_sentinels,(None,)),(module.clean_removal_sentinels,(None,)),(module.private_context_desktop,('S-1-5-21-1-2-3-1001',)),(module.account_sid,(None,)),(module.grant_owned_account_stage,(None,None,None)),(module.registered_feature_states,(None,None))]:
                with self.assertRaisesRegex(RuntimeError,'disposable'):function(*args)
        self.assertIn('cancellation_rollback',m.QUALIFICATION_KEYS);self.assertIn('user_context',m.QUALIFICATION_KEYS)
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
        profiles=SimpleNamespace(GetUserProfileDirectoryW=Function('GetUserProfileDirectoryW'),DeleteProfileW=Function('DeleteProfileW'));dlls={'kernel32':kernel,'advapi32':security,'netapi32':network,'userenv':profiles}
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
        actor_sid='S-1-5-21-1-2-3-1001';events=[];state={};deletions=[]
        class Function:
            def __init__(self,call):self.call=call
            def __call__(self,*args):return self.call(*args)
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
                target.write_text(json.dumps({'token':{'user_sid_sha256':m.digest(actor_sid.encode())},'user_profile':str(state['profile']),'elected_launcher':{'actual_elected_launch':True}}),encoding='utf-8')
            process._obj.process=333;process._obj.thread=444;process._obj.pid=555;return True
        def set_token(process,access,out):out._obj.value=200;return True
        def logon(name,domain,password,kind,provider,out):out._obj.value=100;return True
        def exit_code(process,out):out._obj.value=0;return True
        kernel=SimpleNamespace(GetCurrentProcess=Function(lambda:1),CloseHandle=Function(lambda handle:True),WaitForSingleObject=Function(lambda *args:0),GetExitCodeProcess=Function(exit_code),TerminateProcess=Function(lambda *args:True))
        security=SimpleNamespace(OpenProcessToken=Function(set_token),LogonUserW=Function(logon),CreateProcessWithLogonW=Function(create_process))
        network=SimpleNamespace(NetUserAdd=Function(lambda *args:events.append('account-created') or 0),NetUserDel=Function(lambda server,name:events.append('account-deleted') or 0),NetLocalGroupAddMembers=Function(lambda *args:0))
        profiles=SimpleNamespace(GetUserProfileDirectoryW=Function(profile_directory),DeleteProfileW=Function(delete_profile));dlls={'kernel32':kernel,'advapi32':security,'netapi32':network,'userenv':profiles}
        env={'USERPROFILE':str(original),'ProgramData':str(programdata),'SystemRoot':r'C:\Windows','COMPUTERNAME':'CI-FIXTURE'}
        with patch.dict(os.environ,env),patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(module,'account_sid',side_effect=[actor_sid,'S-1-5-21-1-2-3-1000']),patch.object(module,'grant_owned_account_stage'),patch.object(module,'private_context_desktop',return_value=nullcontext({'path':'owned\\Default'})),patch.object(module.time,'sleep'):
            proof=module.ordinary_context(installer,self.path/'installation',output,self.model)
        self.assertEqual(events,['account-created','profile-created-by-logon','profile-deleted','account-deleted']);self.assertEqual(len(deletions),3)
        self.assertTrue(all(item['deleted'] for item in proof['cleanup']));self.assertEqual((original/'owner-data').read_bytes(),b'keep');self.assertEqual((programdata/'owner-data').read_bytes(),b'keep')
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
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'load',side_effect=[SimpleNamespace(capture=capture),SimpleNamespace(qualify=qualify)]),patch.object(module,'write'),patch.object(m,'verify_installer') as verify:
            with self.assertRaisesRegex(RuntimeError,'gates failed'):module.run_gates(None,receipt,'all')
            self.assertEqual(called,['capture','qualify']);verify.assert_not_called()
            self.assertEqual(receipt['qualification']['native_ui_capture'],'failed');self.assertEqual(receipt['qualification']['user_context'],'pending')
            self.assertEqual(set(receipt['qualification_errors']),{'native_ui_capture','native_lifecycle'})
    def test_provisional_all_native_pass_still_requires_qualified_zip_join(self):
        from types import SimpleNamespace
        spec=importlib.util.spec_from_file_location('package_msi_provisional',ROOT/'scripts/package-msi.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        called=[];artifact=self.path/'provisional.msi';artifact.write_bytes(b'byte-identity fixture')
        receipt={'kind':'windows-installer-receipt','format':'msi','platform':'windows_amd64','variant':'selectable','schema_version':'1.0.1','product_version':'1.0.1','documentation_version':'1.0.1','artifact':artifact.name,'artifact_sha256':m.sha(artifact),'artifact_size_bytes':artifact.stat().st_size,'source_dirty':False,'revision':'a'*40,'payload_receipt':dict(self.inv),'payload_inventory':self.inv,'qualification':{key:'pending' for key in m.QUALIFICATION_KEYS}}
        capture=SimpleNamespace(capture=lambda *args:called.append('native-ui'))
        qualifier=SimpleNamespace(qualify=lambda *args:called.append('actual-lifecycle') or {key:'passed' for key in m.QUALIFICATION_KEYS if key!='native_ui_capture'})
        with patch.object(m,'disposable_runner',return_value=None),patch.object(module,'load',side_effect=[capture,qualifier]),patch.object(module,'write'),patch.object(m,'verify_installer') as verify:
            module.run_gates(artifact,receipt,'provisional');verify.assert_not_called()
        self.assertEqual(called,['native-ui','actual-lifecycle']);self.assertTrue(all(value=='passed' for value in receipt['qualification'].values()))
        self.assertIs(receipt['publication_eligible'],False);self.assertEqual(receipt['payload_qualification'],'pending')
        with self.assertRaisesRegex(ValueError,'provisional'):m.verify_installer(artifact,receipt)
        # Changing flags cannot fabricate the missing ZIP's actual relocation,
        # application/native-webview and ordinary archive installation evidence.
        receipt.update({'publication_eligible':True,'payload_qualification':'passed'})
        with self.assertRaisesRegex(ValueError,'package relocation and native qualification'):m.verify_installer(artifact,receipt)
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
        qualifier=SimpleNamespace(clone=lambda source,target,*args:shutil.copyfile(source,target),run_installer=install,installed_hashes=lambda *args:None,registered_feature_states=lambda *args:{'Core':3,'CLI':3,'Desktop':2},prepare_launcher_fixture=lambda *args:None,ordinary_context=lambda *args:calls.append('ordinary-first-observed' if Path(args[0]).name=='ordinary-first.msi' else 'ordinary-after-observed') or {'separate_user_account':True},token_info=lambda:{'session_id':1,'administrator_enabled':True},installer_mutex_diagnostic=lambda:{'machine_state_modified':False})
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
        qualifier=SimpleNamespace(clone=clone,run_installer=install,installed_hashes=hashes,registered_feature_states=lambda *args:{'Core':3,'CLI':3,'Desktop':2},prepare_launcher_fixture=lambda *args:None,ordinary_context=lambda *args:calls.append(('ordinary-observed',None)) or {'separate_user_account':True})
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
            self.assertEqual(kind,6);self.assertEqual(information,0x17);args[-1]._obj.value=1234;return 0
        kernel=SimpleNamespace(OpenMutexW=Function(open_mutex),CloseHandle=Function(lambda handle:closed.append(handle) or True),LocalFree=Function(lambda value:None))
        security=SimpleNamespace(GetSecurityInfo=Function(descriptor));dlls={'kernel32':kernel,'advapi32':security}
        with patch.object(m,'disposable_runner',return_value=None),patch.object(C,'WinDLL',side_effect=lambda name,**kwargs:dlls[name],create=True),patch.object(C,'get_last_error',side_effect=lambda:state['error'],create=True),patch.object(module,'descriptor_observation',return_value={'owner_group_dacl_label_sddl':'O:SYS:(ML;;NW;;;HI)'}) as observed:
            result=module.installer_mutex_diagnostic()
        self.assertEqual(len(calls),9);self.assertEqual(closed,[0x100000,0x20000]);observed.assert_called_once()
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
        for table in ('Signature','CompLocator','IniLocator','DrLocator','ActionText'):
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
