#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Actual native MSI lifecycle checks, exclusively on acknowledged disposable CI."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import struct
import sys
import time
import uuid
import windows_msi as m

def clone(source,target,version,upgrade,failure=False,missing_prerequisites=False):
    shutil.copyfile(source,target);db=m.NativeDatabase(target,write=True)
    try:
        q=m.quote
        for key,value in [('ProductCode',m.product_code(version,upgrade)),('ProductVersion',version),('UpgradeCode',upgrade)]:db.execute('UPDATE '+q('Property')+' SET '+q('Value')+'=? WHERE '+q('Property')+'=?',[value,key])
        db.execute('UPDATE '+q('Upgrade')+' SET '+q('UpgradeCode')+'=?',[upgrade])
        db.execute('UPDATE '+q('Upgrade')+' SET '+q('VersionMax')+'=? WHERE '+q('ActionProperty')+'=?',[version,'OLDPRODUCTS'])
        db.execute('UPDATE '+q('Upgrade')+' SET '+q('VersionMin')+'=? WHERE '+q('ActionProperty')+'=?',[version,'NEWPRODUCTS'])
        if failure:
            db.execute('INSERT INTO '+q('CustomAction')+' ('+','.join(q(c[0]) for c in m.SCHEMA['CustomAction'])+') VALUES (?,?,?,?)',['FixtureFailure',19,None,'Intentional transactional fixture failure'])
            db.execute('INSERT INTO '+q('InstallExecuteSequence')+' ('+','.join(q(c[0]) for c in m.SCHEMA['InstallExecuteSequence'])+') VALUES (?,?,?)',['FixtureFailure','1',4100])
        if missing_prerequisites:db.execute('UPDATE '+q('RegLocator')+' SET '+q('Key')+'=? WHERE '+q('Signature_')+'=?',['SOFTWARE\\insonic-ci-missing-prerequisite','VCRuntime'])
        db.summary(m.guid('ci-package/'+str(uuid.uuid4())));db.check(db.dll.MsiDatabaseCommit(db.handle))
    finally:db.close()

def run_installer(args,log,expected=True):
    m.disposable_runner()
    exe=Path(os.environ['SystemRoot'])/'System32/msiexec.exe'
    value=subprocess.run([str(exe),*map(str,args),'/qn','/norestart','/L*v',str(log),'LAUNCH=0','START_MENU=0','DESKTOP_SHORTCUT=0','REBOOT=ReallySuppress'],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=90)
    if expected and value.returncode!=0:raise RuntimeError('MSI lifecycle failed '+str(value.returncode)+': '+str(log))
    if not expected and value.returncode==0:raise RuntimeError('MSI accepted intentionally invalid fixture: '+str(log))
    return value.returncode

def child(args,env,cwd):
    m.disposable_runner()
    result=subprocess.run(list(map(str,args)),cwd=cwd,env=env,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=30)
    if result.returncode:raise RuntimeError('installed noninteractive client failed: '+result.stderr.decode(errors='replace')[-2000:])
    return result.stdout

def installed_hashes(directory,model,desktop):
    for f in model['payload']:
        p=directory/f['path']
        if f['feature']=='Desktop' and not desktop:
            if p.exists():raise ValueError('CLI-only installed GUI bytes')
            continue
        if not p.is_file() or m.sha(p)!=f['sha256']:raise ValueError('installed MSI bytes differ: '+f['path'])

def token_info():
    """Observe the actual calling process token, without changing policy."""
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    kernel=C.WinDLL('kernel32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True)
    kernel.GetCurrentProcess.restype=W.HANDLE;kernel.CloseHandle.argtypes=[W.HANDLE];kernel.CloseHandle.restype=W.BOOL
    security.OpenProcessToken.argtypes=[W.HANDLE,W.DWORD,C.POINTER(W.HANDLE)];security.OpenProcessToken.restype=W.BOOL
    security.GetTokenInformation.argtypes=[W.HANDLE,C.c_int,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];security.GetTokenInformation.restype=W.BOOL
    security.CreateWellKnownSid.argtypes=[C.c_int,C.c_void_p,C.c_void_p,C.POINTER(W.DWORD)];security.CreateWellKnownSid.restype=W.BOOL
    security.CheckTokenMembership.argtypes=[W.HANDLE,C.c_void_p,C.POINTER(W.BOOL)];security.CheckTokenMembership.restype=W.BOOL
    security.ConvertSidToStringSidW.argtypes=[C.c_void_p,C.POINTER(C.c_void_p)];security.ConvertSidToStringSidW.restype=W.BOOL
    kernel.LocalFree.argtypes=[C.c_void_p];kernel.LocalFree.restype=C.c_void_p
    token=W.HANDLE()
    if not security.OpenProcessToken(kernel.GetCurrentProcess(),8,C.byref(token)):raise C.WinError(C.get_last_error())
    def info(kind):
        size=W.DWORD();security.GetTokenInformation(token,kind,None,0,C.byref(size))
        if size.value>65536 or not size.value:raise RuntimeError('invalid token information bound')
        data=C.create_string_buffer(size.value)
        if not security.GetTokenInformation(token,kind,data,size.value,C.byref(size)):raise C.WinError(C.get_last_error())
        return data
    try:
        elevated=bool(C.cast(info(20),C.POINTER(W.DWORD))[0]);elevation_type=int(C.cast(info(18),C.POINTER(W.DWORD))[0])
        user=info(1);sid=C.cast(user,C.POINTER(C.c_void_p))[0];value=C.c_void_p()
        if not security.ConvertSidToStringSidW(sid,C.byref(value)):raise C.WinError(C.get_last_error())
        try:user_hash=m.digest(C.wstring_at(value).encode())
        finally:kernel.LocalFree(value)
        admin=C.create_string_buffer(68);size=W.DWORD(68)
        if not security.CreateWellKnownSid(26,None,admin,C.byref(size)):raise C.WinError(C.get_last_error())
        member=W.BOOL()
        if not security.CheckTokenMembership(None,admin,C.byref(member)):raise C.WinError(C.get_last_error())
        privileges=info(3);count=struct.unpack_from('<I',privileges.raw)[0]
        if count>100:raise RuntimeError('invalid token privilege count')
        enabled=sum(bool(struct.unpack_from('<I',privileges.raw,4+i*12+8)[0]&2) for i in range(count))
        return {'administrator_enabled':bool(member.value),'elevated':elevated,'elevation_type':elevation_type,'enabled_privileges':enabled,'user_sid_sha256':user_hash}
    finally:kernel.CloseHandle(token)

def restricted_worker(installer,directory,output):
    m.disposable_runner();observed=token_info()
    if observed['administrator_enabled'] or observed['enabled_privileges']>1:raise RuntimeError('restricted worker retained administrator membership/privileges')
    product=dict(m.table_rows(m.CompoundFile(installer).streams,'Property'))['ProductCode']
    installed=False
    try:
        run_installer(['/i',installer,'INSTALLDIR='+str(directory),'LICENSE_ACCEPTED=1','ADDLOCAL=Core,CLI'],output.with_suffix('.install.log'));installed=True
        marker=directory/m.OWNERSHIP
        if not marker.is_file() or marker.read_bytes()!=m.OWNERSHIP_BYTES:raise RuntimeError('restricted per-user install missing ownership marker')
        run_installer(['/x',product],output.with_suffix('.remove.log'));installed=False
        if marker.exists():raise RuntimeError('restricted per-user removal retained owned marker')
        evidence={'kind':'standard-equivalent-restricted-token','token':observed,'install_exit':0,'remove_exit':0,'separate_user_account':False,'os_policy_mutations':False}
        output.write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    finally:
        if installed:run_installer(['/x',product],output.with_suffix('.cleanup.log'))

def restricted_context(installer,directory,output):
    """Run a real process with the same user, deny-only Administrators and no
    elevated privileges. Exact token values distinguish this from another account.
    """
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    kernel=C.WinDLL('kernel32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True)
    class Sid(C.Structure):_fields_=[('sid',C.c_void_p),('attributes',W.DWORD)]
    class Startup(C.Structure):_fields_=[('cb',W.DWORD),('reserved',W.LPWSTR),('desktop',W.LPWSTR),('title',W.LPWSTR),('x',W.DWORD),('y',W.DWORD),('width',W.DWORD),('height',W.DWORD),('chars_x',W.DWORD),('chars_y',W.DWORD),('fill',W.DWORD),('flags',W.DWORD),('show',W.WORD),('reserved_size',W.WORD),('reserved_bytes',C.c_void_p),('stdin',W.HANDLE),('stdout',W.HANDLE),('stderr',W.HANDLE)]
    class Process(C.Structure):_fields_=[('process',W.HANDLE),('thread',W.HANDLE),('pid',W.DWORD),('tid',W.DWORD)]
    kernel.GetCurrentProcess.restype=W.HANDLE;kernel.CloseHandle.argtypes=[W.HANDLE];kernel.CloseHandle.restype=W.BOOL
    kernel.WaitForSingleObject.argtypes=[W.HANDLE,W.DWORD];kernel.WaitForSingleObject.restype=W.DWORD
    kernel.GetExitCodeProcess.argtypes=[W.HANDLE,C.POINTER(W.DWORD)];kernel.GetExitCodeProcess.restype=W.BOOL
    kernel.TerminateProcess.argtypes=[W.HANDLE,W.UINT];kernel.TerminateProcess.restype=W.BOOL
    security.OpenProcessToken.argtypes=[W.HANDLE,W.DWORD,C.POINTER(W.HANDLE)];security.OpenProcessToken.restype=W.BOOL
    security.CreateWellKnownSid.argtypes=[C.c_int,C.c_void_p,C.c_void_p,C.POINTER(W.DWORD)];security.CreateWellKnownSid.restype=W.BOOL
    security.CreateRestrictedToken.argtypes=[W.HANDLE,W.DWORD,W.DWORD,C.POINTER(Sid),W.DWORD,C.c_void_p,W.DWORD,C.c_void_p,C.POINTER(W.HANDLE)];security.CreateRestrictedToken.restype=W.BOOL
    security.CreateProcessAsUserW.argtypes=[W.HANDLE,W.LPCWSTR,W.LPWSTR,C.c_void_p,C.c_void_p,W.BOOL,W.DWORD,C.c_void_p,W.LPCWSTR,C.POINTER(Startup),C.POINTER(Process)];security.CreateProcessAsUserW.restype=W.BOOL
    token=W.HANDLE();restricted=W.HANDLE();process=Process()
    if not security.OpenProcessToken(kernel.GetCurrentProcess(),0x000b,C.byref(token)):raise C.WinError(C.get_last_error())
    try:
        admin=C.create_string_buffer(68);size=W.DWORD(68)
        if not security.CreateWellKnownSid(26,None,admin,C.byref(size)):raise C.WinError(C.get_last_error())
        disable=Sid(C.cast(admin,C.c_void_p),0)
        # DISABLE_MAX_PRIVILEGE + LUA_TOKEN; SANDBOX_INERT is deliberately absent.
        if not security.CreateRestrictedToken(token,5,1,C.byref(disable),0,None,0,None,C.byref(restricted)):raise C.WinError(C.get_last_error())
        startup=Startup();startup.cb=C.sizeof(startup);startup.flags=1;startup.show=0
        args=[sys.executable,__file__,'--context-worker',str(installer),'--context-dir',str(directory),'--output',str(output)]
        command=C.create_unicode_buffer(subprocess.list2cmdline(args))
        if not security.CreateProcessAsUserW(restricted,sys.executable,command,None,None,False,0x08000000,None,str(m.ROOT),C.byref(startup),C.byref(process)):raise C.WinError(C.get_last_error())
        status=kernel.WaitForSingleObject(process.process,90000)
        if status!=0:
            kernel.TerminateProcess(process.process,1);kernel.WaitForSingleObject(process.process,5000);raise RuntimeError('restricted MSI context worker exceeded its bound')
        code=W.DWORD()
        if not kernel.GetExitCodeProcess(process.process,C.byref(code)) or code.value!=0:raise RuntimeError('restricted MSI context worker failed '+str(code.value))
        evidence=json.loads(output.read_text(encoding='utf-8'))
        if evidence['token']['user_sid_sha256']!=token_info()['user_sid_sha256']:raise RuntimeError('per-user install changed invoking user identity')
        return evidence
    finally:
        for h in (process.thread,process.process,restricted,token):
            if h:kernel.CloseHandle(h)

def cancellation_worker(installer,directory,output):
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    dll=C.WinDLL('msi',use_last_error=True)
    Callback=C.WINFUNCTYPE(C.c_int,C.c_void_p,W.UINT,W.LPCWSTR)
    state={'armed':False,'cancelled':False,'actions':[]}
    def handler(context,kind,message):
        kind=kind&0xff000000
        if kind==0x08000000:
            # ActionStart contains the standard action name, independent of paths.
            if message and 'InstallFiles' in message:state['armed']=True;state['actions'].append('InstallFiles')
            elif message and 'RemoveExistingProducts' in message:state['actions'].append('RemoveExistingProducts')
        elif kind==0x0a000000 and state['armed']:
            state['cancelled']=True;return 2 # IDCANCEL during progress is MSI cancellation.
        return 0
    callback=Callback(handler)
    dll.MsiSetInternalUI.argtypes=[W.UINT,C.c_void_p];dll.MsiSetInternalUI.restype=W.UINT
    dll.MsiSetExternalUIW.argtypes=[Callback,W.DWORD,C.c_void_p];dll.MsiSetExternalUIW.restype=C.c_void_p
    dll.MsiInstallProductW.argtypes=[W.LPCWSTR,W.LPCWSTR];dll.MsiInstallProductW.restype=W.UINT
    dll.MsiEnableLogW.argtypes=[W.DWORD,W.LPCWSTR,W.DWORD];dll.MsiEnableLogW.restype=W.UINT
    dll.MsiSetInternalUI(2,None);dll.MsiEnableLogW(0xffff,str(output.with_suffix('.msi.log')),0);dll.MsiSetExternalUIW(callback,(1<<8)|(1<<10),None)
    properties='INSTALLDIR="'+str(directory)+'" LICENSE_ACCEPTED=1 START_MENU=0 DESKTOP_SHORTCUT=0 LAUNCH=0 REBOOT=ReallySuppress'
    code=dll.MsiInstallProductW(str(installer),properties)
    evidence={'exit_code':code,'callback_requested_cancel':state['cancelled'],'phase':'InstallFiles progress','actions':state['actions'],'policy_mutations':False}
    output.write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    if code!=1602 or not state['cancelled']:raise RuntimeError('native MSI cancellation did not return ERROR_INSTALL_USEREXIT: '+str(code))

def cancellation(installer,directory,output):
    m.disposable_runner()
    completed=subprocess.run([sys.executable,__file__,'--cancel-worker',str(installer),'--context-dir',str(directory),'--output',str(output)],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=90)
    if completed.returncode:raise RuntimeError('native cancellation worker failed: '+completed.stderr.decode(errors='replace')[-3000:])
    return json.loads(output.read_text(encoding='utf-8'))

def qualify(path,receipt):
    m.disposable_runner();path=Path(path);started=time.monotonic()
    inventory=receipt['payload_inventory'];model=m.installer_model(inventory,receipt['payload_receipt']['inventory_sha256'],(m.ROOT/'LICENSE').read_text(encoding='utf-8'));m.verify_model_bytes(path,model)
    original_token=token_info()
    if not original_token['administrator_enabled'] or not original_token['elevated']:raise RuntimeError('elevated disposable runner token is required for dual-context qualification')
    result={};logs=m.ROOT/'build/native/msi-logs';logs.mkdir(parents=True,exist_ok=True);proof=[]
    upgrade=m.guid('disposable-ci/'+os.environ.get('GITHUB_RUN_ID','')+'/'+str(uuid.uuid4()))
    version=model['version'];major,minor,patch=map(int,version.split('.'));new_version=f'{major}.{minor}.{patch+1}'
    with tempfile.TemporaryDirectory(prefix='insonic-ci-msi-') as tmp:
        tmp=Path(tmp);install=tmp/'installed program é with spaces';profile=tmp/'fixture profile';data=tmp/'library';data.mkdir();sentinel=data/'accepted-user-data';sentinel.write_bytes(b'preserve library media settings models credentials')
        env=dict(os.environ);env.update({'APPDATA':str(profile/'Roaming'),'LOCALAPPDATA':str(profile/'Local')});env.pop('INSONIC_DESKTOP_FIXTURE_DIRECTORY',None)
        old=tmp/'old.msi';new=tmp/'new.msi';bad=tmp/'failed-upgrade.msi';missing=tmp/'missing-prerequisite.msi'
        clone(path,old,version,upgrade);clone(path,new,new_version,upgrade);clone(path,bad,new_version,upgrade,failure=True);clone(path,missing,version,upgrade,missing_prerequisites=True)
        active=None;installed=None
        def call(msi,selection=None,**properties):
            args=['/i',msi,'INSTALLDIR='+str(install),'LICENSE_ACCEPTED=1']
            if selection is not None:args+=['ADDLOCAL='+selection]
            args += [k+'='+v for k,v in properties.items()]
            return args
        try:
            run_installer(['/i',old,'INSTALLDIR='+str(install),'ADDLOCAL=Core'],logs/'license-rejected.log',False)
            run_installer(call(old,'Core'),logs/'empty-selection-rejected.log',False)
            run_installer(call(old,'Desktop,CLI'),logs/'missing-core-rejected.log',False)
            run_installer(call(missing,'Core,CLI'),logs/'prerequisite-rejected.log',False)
            if install.exists() and any(install.iterdir()):raise ValueError('invalid fixture left program files')
            result['license_and_prerequisites']='passed'
            for selection,desktop in [('Core,CLI',False),('Core,Desktop',True),('Core,Desktop,CLI',True)]:
                run_installer(call(old,selection),logs/('selection-'+selection.replace(',','-')+'.log'));installed=m.product_code(version,upgrade);installed_hashes(install,model,desktop)
                if child([install/'insonic.exe','version'],env,install).decode().strip()!=version:raise ValueError('installed CLI version differs')
                plan=json.loads(child([install/'insonic-launcher.exe','--describe','--desktop' if desktop else '--cli'],env,install))
                if not plan:raise ValueError('installed launcher describe returned no plan')
                run_installer(['/x',installed],logs/'selection-remove.log');installed=None
                if any((install/f['path']).exists() for f in model['payload']):raise ValueError('uninstall retained installer-owned bytes')
            result['client_selections']='passed';result['launch_contract']='passed'
            context_evidence=restricted_context(old,tmp/'restricted per-user installation',logs/'restricted-context.json');result['user_context']='passed'
            run_installer(call(old,'Core,CLI'),logs/'maintenance-initial.log');installed=m.product_code(version,upgrade)
            run_installer(call(old,'Core,Desktop,CLI',REINSTALL='ALL',REINSTALLMODE='vomus'),logs/'feature-change.log');installed_hashes(install,model,True)
            (install/'insonic-session.cmd').unlink();(install/m.OWNERSHIP).unlink();run_installer(['/fa',installed],logs/'repair.log');installed_hashes(install,model,True);result['maintenance']='passed'
            # A real running console image under the program tree represents an
            # active client/runtime. No application termination is delegated to MSI.
            active_path=install/'active-owned-ci-fixture.exe';shutil.copyfile(Path(os.environ['SystemRoot'])/'System32/ping.exe',active_path)
            active=subprocess.Popen([str(active_path),'127.0.0.1','-n','60'],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW)
            time.sleep(.15);run_installer(['/x',installed],logs/'active-work-rejected.log',False)
            if active.poll() is not None:raise ValueError('MSI terminated active work')
            installed_hashes(install,model,True);active.terminate();active.wait(timeout=5);active=None;active_path.unlink();result['active_work_preserved']='passed'
            # Deliberately remove Desktop before upgrading to prove feature migration.
            run_installer(call(old,'Core,CLI',REMOVE='Desktop',REINSTALLMODE='vomus'),logs/'cli-election.log');installed_hashes(install,model,False)
            cancellation_evidence=cancellation(new,install,logs/'cancelled-upgrade.json');installed_hashes(install,model,False)
            if sentinel.read_bytes()!=b'preserve library media settings models credentials':raise ValueError('cancelled MSI changed user data')
            result['cancellation_rollback']='passed'
            run_installer(call(bad),logs/'upgrade-rollback.log',False);installed_hashes(install,model,False)
            run_installer(call(new),logs/'upgrade-success.log');installed=m.product_code(new_version,upgrade);installed_hashes(install,model,False);result['upgrade_rollback']='passed'
            run_installer(call(old),logs/'downgrade-rejected.log',False);installed_hashes(install,model,False);result['downgrade_rejection']='passed'
            run_installer(['/x',installed],logs/'final-remove.log');installed=None
            if sentinel.read_bytes()!=b'preserve library media settings models credentials':raise ValueError('MSI modified user data')
            if any((install/f['path']).exists() for f in model['payload']):raise ValueError('MSI removal retained owned files')
            result['user_data_preserved']='passed'
            proof={'artifact_sha256':receipt['artifact_sha256'],'revision':receipt['revision'],'windows_build':str(__import__('sys').getwindowsversion().build),'fixture_upgrade_code':upgrade,'checks':result,'elapsed_seconds':round(time.monotonic()-started,3),'active_work_fixture':'running owned ping image, no process shutdown','policy_prompts':'not-observed','signing':'unsigned','cancellation':cancellation_evidence,'user_context':{'original_elevated_token':original_token,'restricted_process':context_evidence}}
        finally:
            if active is not None:active.terminate();active.wait(timeout=5)
            if installed is not None:run_installer(['/x',installed],logs/'fixture-cleanup.log')
    (m.ROOT/'build/native/msi-lifecycle.json').write_text(json.dumps(proof,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    return result
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('installer',type=Path,nargs='?');p.add_argument('--receipt',type=Path);p.add_argument('--cancel-worker',type=Path);p.add_argument('--context-worker',type=Path);p.add_argument('--context-dir',type=Path);p.add_argument('--output',type=Path);a=p.parse_args();m.disposable_runner()
    if a.cancel_worker:cancellation_worker(a.cancel_worker,a.context_dir,a.output)
    elif a.context_worker:restricted_worker(a.context_worker,a.context_dir,a.output)
    elif a.installer and a.receipt:r=json.loads(a.receipt.read_text(encoding='utf-8'));print(json.dumps(qualify(a.installer,r)))
    else:p.error('installer and --receipt are required')
