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

def feature_diagnostic_rows():
    """Property-only observations added exclusively to qualification clones."""
    rows={'CustomAction':[],'InstallExecuteSequence':[]};sequence=1001
    for feature in ('Core','CLI','Desktop'):
        for kind,prefix in [('ACTION','&'),('INSTALLED','!')]:
            prop='S027_'+feature.upper()+'_'+kind
            for state in range(-1,6):
                action='S027Diag_'+feature+'_'+kind+'_'+('m1' if state==-1 else str(state))
                rows['CustomAction'].append([action,51,prop,str(state)])
                rows['InstallExecuteSequence'].append([action,prefix+feature+'='+str(state),sequence]);sequence+=1
    return rows

def clone(source,target,version,upgrade,failure=False,missing_prerequisites=False):
    upgrade_rows=m.table_rows(m.CompoundFile(source).streams,'Upgrade')
    shutil.copyfile(source,target);db=m.NativeDatabase(target,write=True)
    try:
        q=m.quote
        for key,value in [('ProductCode',m.product_code(version,upgrade)),('ProductVersion',version),('UpgradeCode',upgrade)]:db.execute('UPDATE '+q('Property')+' SET '+q('Value')+'=? WHERE '+q('Property')+'=?',[value,key])
        # UpgradeCode, version bounds and language are composite primary keys.
        # Windows Installer does not permit UPDATE of those fields (1627).
        db.execute('DELETE FROM '+q('Upgrade'))
        for original in upgrade_rows:
            row=list(original);row[0]=upgrade
            if row[6]=='OLDPRODUCTS':row[2]=version
            elif row[6]=='NEWPRODUCTS':row[1]=version
            else:raise ValueError('unexpected qualification upgrade relation')
            db.execute('INSERT INTO '+q('Upgrade')+' ('+','.join(q(c[0]) for c in m.SCHEMA['Upgrade'])+') VALUES (?,?,?,?,?,?,?)',row)
        if failure:
            db.execute('INSERT INTO '+q('CustomAction')+' ('+','.join(q(c[0]) for c in m.SCHEMA['CustomAction'])+') VALUES (?,?,?,?)',['FixtureFailure',19,None,'Intentional transactional fixture failure'])
            db.execute('INSERT INTO '+q('InstallExecuteSequence')+' ('+','.join(q(c[0]) for c in m.SCHEMA['InstallExecuteSequence'])+') VALUES (?,?,?)',['FixtureFailure','1',4100])
        if missing_prerequisites:db.execute('UPDATE '+q('RegLocator')+' SET '+q('Key')+'=? WHERE '+q('Signature_')+'=?',['SOFTWARE\\insonic-ci-missing-prerequisite','VCRuntime'])
        for table,rows in feature_diagnostic_rows().items():
            for row in rows:db.execute('INSERT INTO '+q(table)+' ('+','.join(q(column[0]) for column in m.SCHEMA[table])+') VALUES ('+','.join('?' for _ in row)+')',row)
        db.summary(m.guid('ci-package/'+str(uuid.uuid4())));db.check(db.dll.MsiDatabaseCommit(db.handle))
    finally:db.close()

def registered_feature_states(product,output):
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    dll=C.WinDLL('msi',use_last_error=True);dll.MsiQueryFeatureStateW.argtypes=[W.LPCWSTR,W.LPCWSTR];dll.MsiQueryFeatureStateW.restype=C.c_int
    values={feature:int(dll.MsiQueryFeatureStateW(product,feature)) for feature in ('Core','CLI','Desktop')}
    output.write_text(json.dumps({'kind':'registered-feature-state-observation','product_code':product,'states':values,'api':'MsiQueryFeatureStateW','machine_state_modified':False},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    return values

def installer_properties(arguments):
    """Build the MSI API property language, independently of Windows argv quoting."""
    properties={}
    for argument in arguments:
        key,sep,value=str(argument).partition('=')
        if not sep or not __import__('re').fullmatch('[A-Z_][A-Z0-9_]*',key) or any(c in value for c in '\0\r\n'):raise ValueError('invalid native installer property')
        properties[key]=value
    properties.update({'LAUNCH':'0','REBOOT':'ReallySuppress'})
    return ' '.join(key+'="'+value.replace('"','""')+'"' for key,value in properties.items())

def service_diagnostic():
    """Read-only Installer service status on disposable CI. Never stop services."""
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    dll=C.WinDLL('advapi32',use_last_error=True)
    class Status(C.Structure):_fields_=[('service_type',W.DWORD),('state',W.DWORD),('controls',W.DWORD),('win32_exit',W.DWORD),('service_exit',W.DWORD),('checkpoint',W.DWORD),('wait_hint',W.DWORD),('pid',W.DWORD),('flags',W.DWORD)]
    dll.OpenSCManagerW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.DWORD];dll.OpenSCManagerW.restype=W.HANDLE
    dll.OpenServiceW.argtypes=[W.HANDLE,W.LPCWSTR,W.DWORD];dll.OpenServiceW.restype=W.HANDLE
    dll.QueryServiceStatusEx.argtypes=[W.HANDLE,C.c_int,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];dll.QueryServiceStatusEx.restype=W.BOOL
    dll.CloseServiceHandle.argtypes=[W.HANDLE];dll.CloseServiceHandle.restype=W.BOOL
    manager=dll.OpenSCManagerW(None,None,1)
    if not manager:return {'query_error':C.get_last_error()}
    service=None
    try:
        service=dll.OpenServiceW(manager,'msiserver',4)
        if not service:return {'query_error':C.get_last_error()}
        value=Status();needed=W.DWORD()
        if not dll.QueryServiceStatusEx(service,0,C.byref(value),C.sizeof(value),C.byref(needed)):return {'query_error':C.get_last_error()}
        return {'pid':value.pid,'state':value.state,'checkpoint':value.checkpoint,'wait_hint_ms':value.wait_hint,'win32_exit':value.win32_exit}
    finally:
        if service:dll.CloseServiceHandle(service)
        dll.CloseServiceHandle(manager)

def descriptor_observation(descriptor,security,kernel):
    """Describe an already-read native descriptor, without publishing user SIDs."""
    import ctypes as C
    import ctypes.wintypes as W
    security.GetSecurityDescriptorLength.argtypes=[C.c_void_p];security.GetSecurityDescriptorLength.restype=W.DWORD
    security.ConvertSecurityDescriptorToStringSecurityDescriptorW.argtypes=[C.c_void_p,W.DWORD,W.DWORD,C.POINTER(C.c_void_p),C.POINTER(W.DWORD)];security.ConvertSecurityDescriptorToStringSecurityDescriptorW.restype=W.BOOL
    size=security.GetSecurityDescriptorLength(descriptor)
    if not 0<size<=65536:raise RuntimeError('security descriptor exceeds observation bound')
    value=C.c_void_p()
    # The descriptor query requests only the mandatory label from SACL. Its
    # already-retrieved ACE is serialized using SACL_SECURITY_INFORMATION.
    if not security.ConvertSecurityDescriptorToStringSecurityDescriptorW(descriptor,1,0xf,C.byref(value),None):return {'descriptor_sha256':m.digest(C.string_at(descriptor,size)),'conversion_error':C.get_last_error()}
    try:
        raw=C.wstring_at(value);sanitized=__import__('re').sub(r'S-1-(?:\d+-)*\d+',lambda match:'SID_SHA256_'+m.digest(match.group().encode()),raw)
        return {'descriptor_sha256':m.digest(C.string_at(descriptor,size)),'owner_group_dacl_label_sddl':sanitized,'security_information':0x17,'machine_state_modified':False}
    finally:kernel.LocalFree(value)

def installer_mutex_diagnostic():
    """Read exact Installer mutex access, never create, acquire or modify one."""
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    kernel=C.WinDLL('kernel32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True)
    kernel.OpenMutexW.argtypes=[W.DWORD,W.BOOL,W.LPCWSTR];kernel.OpenMutexW.restype=W.HANDLE
    kernel.CloseHandle.argtypes=[W.HANDLE];kernel.CloseHandle.restype=W.BOOL;kernel.LocalFree.argtypes=[C.c_void_p];kernel.LocalFree.restype=C.c_void_p
    security.GetSecurityInfo.argtypes=[W.HANDLE,C.c_int,W.DWORD,C.c_void_p,C.c_void_p,C.c_void_p,C.c_void_p,C.POINTER(C.c_void_p)];security.GetSecurityInfo.restype=W.DWORD
    rows=[]
    for name in ('_MSIExecute','Global\\_MSIExecute','Local\\_MSIExecute'):
        for access in (0x100000,0x20000,0x1f0001):
            handle=kernel.OpenMutexW(access,False,name);row={'name':name,'requested_access':access,'opened':bool(handle),'error':0 if handle else C.get_last_error()};rows.append(row)
            if not handle:continue
            descriptor=C.c_void_p()
            try:
                if access&0x20000:
                    result=security.GetSecurityInfo(handle,6,0x17,None,None,None,None,C.byref(descriptor));row['descriptor_query_error']=result
                    if result==0:row['security']=descriptor_observation(descriptor,security,kernel)
            finally:
                if descriptor:kernel.LocalFree(descriptor)
                kernel.CloseHandle(handle)
    return {'kind':'read-only-installer-mutex-access','objects':rows,'machine_state_modified':False,'mutex_acquired':False}

def owned_user_object_observation(handle,user,security,kernel):
    """Read actual security of an already-created owned station or desktop."""
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    user.GetUserObjectSecurity.argtypes=[W.HANDLE,C.POINTER(W.DWORD),C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];user.GetUserObjectSecurity.restype=W.BOOL
    information=W.DWORD(0x17);size=W.DWORD();user.GetUserObjectSecurity(handle,C.byref(information),None,0,C.byref(size))
    if not 0<size.value<=65536:return {'query_error':C.get_last_error(),'information':information.value}
    descriptor=C.create_string_buffer(size.value)
    if not user.GetUserObjectSecurity(handle,C.byref(information),descriptor,size.value,C.byref(size)):return {'query_error':C.get_last_error(),'information':information.value}
    return descriptor_observation(descriptor,security,kernel)

def native_installer_dll(log):
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    dll=C.WinDLL('msi',use_last_error=True)
    dll.MsiSetInternalUI.argtypes=[W.UINT,C.c_void_p];dll.MsiSetInternalUI.restype=W.UINT
    dll.MsiEnableLogW.argtypes=[W.DWORD,W.LPCWSTR,W.DWORD];dll.MsiEnableLogW.restype=W.UINT
    dll.MsiInstallProductW.argtypes=[W.LPCWSTR,W.LPCWSTR];dll.MsiInstallProductW.restype=W.UINT
    dll.MsiConfigureProductExW.argtypes=[W.LPCWSTR,C.c_int,C.c_int,W.LPCWSTR];dll.MsiConfigureProductExW.restype=W.UINT
    if dll.MsiSetInternalUI(2,None)==0:raise RuntimeError('cannot select native Installer UI level NONE')
    rc=dll.MsiEnableLogW(0x1000,str(log),0) # INSTALLLOGMODE_VERBOSE, default buffered flush.
    if rc:raise RuntimeError('cannot enable native Installer verbose log: '+str(rc))
    return dll

def native_installer_worker(request_path):
    m.disposable_runner();request=json.loads(request_path.read_text(encoding='utf-8'));log=Path(request['log']);phase=log.with_suffix('.worker.json');log.parent.mkdir(parents=True,exist_ok=True)
    def record(state,**extra):phase.write_text(json.dumps({'kind':'native-installer-worker','pid':os.getpid(),'phase':state,'action':request['action'],'captured_pipes':False,**extra},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    record('starting',service=service_diagnostic())
    dll=native_installer_dll(log);properties=installer_properties(request['properties']);mutex=installer_mutex_diagnostic();record('before-api',ui_level='NONE',log_mode='VERBOSE',service=service_diagnostic(),mutex=mutex)
    if request['action']=='/i':rc=dll.MsiInstallProductW(request['subject'],properties)
    elif request['action']=='/x':rc=dll.MsiConfigureProductExW(request['subject'],0,2,properties)
    elif request['action']=='/fa':rc=dll.MsiConfigureProductExW(request['subject'],0,5,installer_properties([*request['properties'],'REINSTALL=ALL','REINSTALLMODE=amus']))
    else:raise ValueError('unsupported native Installer lifecycle operation')
    record('completed',exit_code=rc,ui_level='NONE',log_mode='VERBOSE',service=service_diagnostic(),mutex_before_api=mutex,mutex_after_api=installer_mutex_diagnostic())

def assert_rejection(log,expected_reason,expected_action='LaunchConditions'):
    text=Path(log).read_text(encoding='utf-16',errors='replace')
    if expected_reason not in text or not __import__('re').search(r'Action ended[^\r\n]*'+__import__('re').escape(expected_action)+r'\. Return value 3\.',text):raise RuntimeError('MSI failed before the intended '+expected_action+' rejection: '+str(log))
    return expected_reason

def assert_effective_directory(log,expected):
    """Require engine evidence that registry discovery did not override a target."""
    text=Path(log).read_text(encoding='utf-16',errors='replace');re=__import__('re');paths=__import__('ntpath')
    values=re.findall(r'PROPERTY CHANGE: (?:Adding|Modifying) INSTALLDIR property\. Its (?:value|new value) is \'([^\']*)\'\.',text)
    final=re.findall(r'Property\(S\): INSTALLDIR = ([^\r\n]*)',text)
    value=final[-1] if final else values[-1] if values else None
    if value is None or paths.normcase(paths.normpath(value))!=paths.normcase(paths.normpath(str(expected))):raise ValueError('MSI engine did not retain the explicit alternate installation directory')

def run_installer(args,log,expected=True,rejection_reason=None,rejection_action='LaunchConditions'):
    m.disposable_runner();args=list(map(str,args));log=Path(log);log.parent.mkdir(parents=True,exist_ok=True)
    if len(args)<2 or args[0] not in ('/i','/x','/fa'):raise ValueError('unsupported native Installer lifecycle operation')
    request={'action':args[0],'subject':args[1],'properties':args[2:],'log':str(log)};installer_properties(request['properties'])
    request_path=log.with_suffix('.request.json');request_path.write_text(json.dumps(request,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    # Wait on the owned process, never communicate()/PIPE EOF: Installer RPC
    # services and children must not prolong this worker through inherited pipes.
    with log.with_suffix('.stderr.log').open('wb') as error:
        process=subprocess.Popen([sys.executable,__file__,'--installer-worker',str(request_path)],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=error,creationflags=subprocess.CREATE_NO_WINDOW)
        try:process.wait(timeout=90)
        except subprocess.TimeoutExpired:
            phase=log.with_suffix('.worker.json');observed=json.loads(phase.read_text(encoding='utf-8')) if phase.exists() else {'phase':'worker-not-initialized'}
            diagnostic={'pid':process.pid,'worker':observed,'service':service_diagnostic(),'captured_pipes':False,'log_exists':log.is_file(),'outcome':'unknown until engine rollback/completion'}
            log.with_suffix('.timeout.json').write_text(json.dumps(diagnostic,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n');process.kill();process.wait(timeout=5)
            raise RuntimeError('native Installer worker exceeded its bound; diagnostics: '+str(log.with_suffix('.timeout.json')))
    if process.returncode:raise RuntimeError('native Installer worker failed: '+log.with_suffix('.stderr.log').read_text(encoding='utf-8',errors='replace')[-3000:])
    phase=json.loads(log.with_suffix('.worker.json').read_text(encoding='utf-8'))
    if phase.get('phase')!='completed':raise RuntimeError('native Installer worker did not complete')
    code=phase['exit_code']
    if expected and code!=0:raise RuntimeError('MSI lifecycle failed '+str(code)+': '+str(log))
    if not expected and code==0:raise RuntimeError('MSI accepted intentionally invalid fixture: '+str(log))
    if not expected and rejection_reason is not None:assert_rejection(log,rejection_reason,rejection_action)
    return code

def child(args,env,cwd):
    m.disposable_runner()
    result=subprocess.run(list(map(str,args)),cwd=cwd,env=env,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=30)
    if result.returncode:raise RuntimeError('installed noninteractive client failed: '+result.stderr.decode(errors='replace')[-2000:])
    return result.stdout

def saved_shortcut_elections():
    m.disposable_runner()
    import winreg
    with winreg.OpenKey(winreg.HKEY_CURRENT_USER,r'Software\ShruggieTech\insonic',0,winreg.KEY_READ|winreg.KEY_WOW64_64KEY) as key:
        return {'start_menu':str(winreg.QueryValueEx(key,'StartMenu')[0]),'desktop_shortcut':str(winreg.QueryValueEx(key,'DesktopShortcut')[0])}

def assert_unchecked_shortcuts():
    m.disposable_runner()
    if saved_shortcut_elections()!={'start_menu':'0','desktop_shortcut':'0'}:raise ValueError('unchecked shortcut elections were not durably stored as zero')
    # GitHub-hosted disposable profiles have no redirected Start/Desktop folders;
    # these are also the concrete per-user paths used in the lifecycle fixture.
    menu=Path(os.environ['APPDATA'])/'Microsoft/Windows/Start Menu/Programs/insonic'
    desktop=Path(os.environ['USERPROFILE'])/'Desktop'
    if any(path.exists() for path in (menu/'insonic.lnk',menu/'insonic CLI.lnk',desktop/'insonic.lnk')):raise ValueError('upgrade re-enabled an unchecked shortcut integration')

def create_removal_sentinels():
    """Fixture-owned siblings test exact integration removal, never blanket cleanup."""
    m.disposable_runner()
    import winreg
    name='insonic-ci-unrelated-'+uuid.uuid4().hex
    content=b'unrelated owner integration preserved by exact MSI removal'
    menu=Path(os.environ['APPDATA'])/'Microsoft/Windows/Start Menu/Programs/insonic'
    desktop=Path(os.environ['USERPROFILE'])/'Desktop'
    files=[menu/(name+'.lnk'),desktop/(name+'.lnk')]
    for path in files:
        path.parent.mkdir(parents=True,exist_ok=True)
        with path.open('xb') as stream:stream.write(content)
    with winreg.CreateKeyEx(winreg.HKEY_CURRENT_USER,r'Software\ShruggieTech\insonic',0,winreg.KEY_SET_VALUE|winreg.KEY_WOW64_64KEY) as key:
        winreg.SetValueEx(key,name,0,winreg.REG_SZ,content.decode('ascii'))
    return {'files':files,'registry_value':name,'content':content}

def assert_removal_sentinels(sentinels):
    m.disposable_runner()
    import winreg
    if any(not path.is_file() or path.read_bytes()!=sentinels['content'] for path in sentinels['files']):raise ValueError('MSI modified an unrelated shortcut/file sibling')
    with winreg.OpenKey(winreg.HKEY_CURRENT_USER,r'Software\ShruggieTech\insonic',0,winreg.KEY_READ|winreg.KEY_WOW64_64KEY) as key:
        if winreg.QueryValueEx(key,sentinels['registry_value'])[0]!=sentinels['content'].decode('ascii'):raise ValueError('MSI modified an unrelated registry value')

def clean_removal_sentinels(sentinels):
    m.disposable_runner()
    import winreg
    for path in sentinels['files']:path.unlink(missing_ok=True)
    try:
        with winreg.OpenKey(winreg.HKEY_CURRENT_USER,r'Software\ShruggieTech\insonic',0,winreg.KEY_SET_VALUE|winreg.KEY_WOW64_64KEY) as key:winreg.DeleteValue(key,sentinels['registry_value'])
    except FileNotFoundError:pass

def installed_hashes(directory,model,desktop):
    for f in model['payload']:
        p=directory/f['path']
        if f['feature']=='Desktop' and not desktop:
            if p.exists():raise ValueError('CLI-only installed GUI bytes')
            continue
        actual=m.sha(p) if p.is_file() else 'missing'
        if actual!=f['sha256']:raise ValueError('installed MSI bytes differ: '+f['path']+'; expected_sha256='+f['sha256']+'; actual_sha256='+actual)

LAUNCH_REPORTER_SOURCE=r'''// SPDX-License-Identifier: Apache-2.0
// Temporary CI instrumentation, never included in product payload.
package main
import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "unsafe"
 "golang.org/x/sys/windows"
)
func main() {
 exe,e:=os.Executable();if e!=nil {os.Exit(1)}
 directory:=filepath.Dir(exe)
 if b,e:=os.ReadFile(filepath.Join(directory,"owned-ci-fixture.json"));e!=nil||string(b)!="insonic disposable launcher context fixture\n" {os.Exit(1)}
 var token windows.Token
 if windows.OpenProcessToken(windows.CurrentProcess(),windows.TOKEN_QUERY,&token)!=nil {os.Exit(1)}
 defer token.Close()
 var elevated,size uint32
 if windows.GetTokenInformation(token,windows.TokenElevation,(*byte)(unsafe.Pointer(&elevated)),uint32(unsafe.Sizeof(elevated)),&size)!=nil {os.Exit(1)}
 user,e:=token.GetTokenUser();if e!=nil {os.Exit(1)}
 admin,e:=windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid);if e!=nil {os.Exit(1)}
 member,e:=windows.Token(0).IsMember(admin);if e!=nil {os.Exit(1)}
 var integrity [256]byte
 if windows.GetTokenInformation(token,windows.TokenIntegrityLevel,&integrity[0],uint32(len(integrity)),&size)!=nil {os.Exit(1)}
 label:=(*windows.SIDAndAttributes)(unsafe.Pointer(&integrity[0]));count:=label.Sid.SubAuthorityCount();if count==0||count>15 {os.Exit(1)}
 integrityRID:=label.Sid.SubAuthority(uint32(count-1))
 sum:=sha256.Sum256([]byte(user.User.Sid.String()))
 observed:=map[string]any{"elevated":elevated!=0,"administrator_enabled":member,"user_sid_sha256":hex.EncodeToString(sum[:]),"integrity_rid":integrityRID,"integrity_sid":label.Sid.String(),"mode":filepath.Base(exe),"pid":os.Getpid(),"fixture":"owned native Go token reporter"}
 data,e:=json.Marshal(observed);if e!=nil {os.Exit(1)}
 if os.WriteFile(filepath.Join(directory,filepath.Base(exe)+".token.json"),append(data,'\n'),0600)!=nil {os.Exit(1)}
}
'''

def prepare_launcher_fixture(installed,directory):
    m.disposable_runner();directory.mkdir(parents=True,exist_ok=False)
    shutil.copyfile(installed/'insonic-launcher.exe',directory/'insonic-launcher.exe')
    session=(installed/'insonic-session.cmd').read_bytes()
    # Run the production session prefix and close this owned CI console after the
    # token recorder returns. Installed production bytes are never modified.
    (directory/'insonic-session.cmd').write_bytes(session+b'exit\n')
    (directory/'owned-ci-fixture.json').write_bytes(b'insonic disposable launcher context fixture\n')
    source=directory/'token-recorder.go';source.write_text(LAUNCH_REPORTER_SOURCE,encoding='utf-8',newline='\n')
    env=dict(os.environ);env['CGO_ENABLED']='0'
    compiled=subprocess.run(['go','build','-ldflags','-H windowsgui','-o',str(directory/'insonic-desktop.exe'),str(source)],cwd=m.ROOT,env=env,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=60)
    if compiled.returncode:raise RuntimeError('native launcher token recorder compilation failed: '+compiled.stderr.decode(errors='replace')[-3000:])
    shutil.copyfile(directory/'insonic-desktop.exe',directory/'insonic.exe')
    return {'kind':'temporary-owned-native-token-recorder','launcher_sha256':m.sha(directory/'insonic-launcher.exe'),'production_session_prefix_sha256':m.digest(session),'instrumented_session_suffix':'exit after production help/setup prefix','recorder_source_sha256':m.digest(LAUNCH_REPORTER_SOURCE.encode()),'product_payload_modified':False}

def linked_token_evidence():
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    kernel=C.WinDLL('kernel32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True)
    kernel.GetCurrentProcess.restype=W.HANDLE;kernel.CloseHandle.argtypes=[W.HANDLE];kernel.CloseHandle.restype=W.BOOL
    security.OpenProcessToken.argtypes=[W.HANDLE,W.DWORD,C.POINTER(W.HANDLE)];security.OpenProcessToken.restype=W.BOOL
    security.GetTokenInformation.argtypes=[W.HANDLE,C.c_int,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];security.GetTokenInformation.restype=W.BOOL
    token=W.HANDLE();linked=W.HANDLE();size=W.DWORD()
    if not security.OpenProcessToken(kernel.GetCurrentProcess(),8,C.byref(token)):raise C.WinError(C.get_last_error())
    try:
        found=security.GetTokenInformation(token,19,C.byref(linked),C.sizeof(linked),C.byref(size))
        if found and linked.value:return {'available':True,'lookup_error':0}
        error=C.get_last_error()
        if error!=1312 and not (error==87 and token_info()['elevation_type']==1):raise C.WinError(error)
        return {'available':False,'lookup_error':error}
    finally:
        if linked:kernel.CloseHandle(linked)
        kernel.CloseHandle(token)

def elected_launcher_context(directory,allow_refusal=False):
    m.disposable_runner();caller=token_info();modes={}
    linked=linked_token_evidence() if caller['elevated'] else {'available':False,'lookup_error':None}
    refusal_permitted=allow_refusal and caller['elevated'] and not linked['available']
    for mode,exe in [('desktop','insonic-desktop.exe'),('cli','insonic.exe')]:
        output=directory/(exe+'.token.json')
        if output.exists():output.unlink()
        attempted=subprocess.run([str(directory/'insonic-launcher.exe'),'--'+mode],cwd=directory,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=15)
        if attempted.returncode:
            message=attempted.stderr.decode(errors='replace')
            if not refusal_permitted or attempted.returncode!=1 or 'cannot launch at normal user privilege' not in message or output.exists():raise RuntimeError('elected '+mode+' launcher failed: '+message[-2000:])
            modes[mode]={'outcome':'refused-normal-launch-action','child_record_absent':True,'elevated_fallback':False};continue
        deadline=time.monotonic()+10;observed=None
        while time.monotonic()<deadline:
            try:observed=json.loads(output.read_text(encoding='utf-8'));break
            except (FileNotFoundError,json.JSONDecodeError):time.sleep(.05)
        if observed is None:raise RuntimeError('elected '+mode+' child did not record its actual token')
        if observed['elevated'] or observed['administrator_enabled'] or observed['integrity_rid']>8192 or observed['user_sid_sha256']!=caller['user_sid_sha256']:raise RuntimeError('elected '+mode+' launch changed user or retained elevation/high integrity')
        modes[mode]={'outcome':'observed-same-user-non-elevated-child','observed_token':observed}
    return {'caller_token':caller,'linked_token':linked,'modes':modes,'actual_elected_launch':True,'describe_is_proof':False}

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
        integrity=info(25);integrity_value=C.c_void_p()
        if not security.ConvertSidToStringSidW(C.cast(integrity,C.POINTER(C.c_void_p))[0],C.byref(integrity_value)):raise C.WinError(C.get_last_error())
        try:integrity_sid=C.wstring_at(integrity_value);integrity_rid=int(integrity_sid.rsplit('-',1)[1])
        finally:kernel.LocalFree(integrity_value)
        admin=C.create_string_buffer(68);size=W.DWORD(68)
        if not security.CreateWellKnownSid(26,None,admin,C.byref(size)):raise C.WinError(C.get_last_error())
        member=W.BOOL()
        if not security.CheckTokenMembership(None,admin,C.byref(member)):raise C.WinError(C.get_last_error())
        privileges=info(3);count=struct.unpack_from('<I',privileges.raw)[0]
        if count>100:raise RuntimeError('invalid token privilege count')
        enabled=sum(bool(struct.unpack_from('<I',privileges.raw,4+i*12+8)[0]&2) for i in range(count))
        return {'administrator_enabled':bool(member.value),'elevated':elevated,'elevation_type':elevation_type,'enabled_privileges':enabled,'user_sid_sha256':user_hash,'integrity_sid':integrity_sid,'integrity_rid':integrity_rid,'session_id':int(C.cast(info(12),C.POINTER(W.DWORD))[0])}
    finally:kernel.CloseHandle(token)

def ordinary_worker(installer,directory,output):
    m.disposable_runner()
    progress=output.with_suffix('.progress.json')
    def record(phase,**extra):progress.write_text(json.dumps({'pid':os.getpid(),'phase':phase,**extra},sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    record('python-started');observed=token_info();record('token-observed',token=observed,mutex_before_install=installer_mutex_diagnostic())
    if observed['administrator_enabled'] or observed['enabled_privileges']>1 or observed['elevated'] or observed['integrity_rid']!=8192:raise RuntimeError('ordinary worker retained administrator membership/privileges/elevation or did not run at medium integrity')
    product=dict(m.table_rows(m.CompoundFile(installer).streams,'Property'))['ProductCode']
    installed=False
    try:
        run_installer(['/i',installer,'INSTALLDIR='+str(directory),'LICENSE_ACCEPTED=1','ADDLOCAL=Core,CLI'],output.with_suffix('.install.log'));installed=True
        record('installed')
        marker=directory/m.OWNERSHIP
        if not marker.is_file() or marker.read_bytes()!=m.OWNERSHIP_BYTES:raise RuntimeError('ordinary per-user install missing ownership marker')
        model=json.loads((directory.parent/'context-model.json').read_text(encoding='utf-8'));installed_hashes(directory,model,False)
        version=dict(m.table_rows(m.CompoundFile(installer).streams,'Property'))['ProductVersion']
        if child([directory/'insonic.exe','version'],dict(os.environ),directory).decode().strip()!=version:raise RuntimeError('ordinary-user installed CLI version differs')
        launch_evidence=elected_launcher_context(directory.parent/'launcher-context-fixture')
        record('elected-launch-observed')
        run_installer(['/x',product],output.with_suffix('.remove.log'));installed=False
        if marker.exists():raise RuntimeError('ordinary per-user removal retained owned marker')
        evidence={'kind':'ordinary-disposable-ci-user','token':observed,'install_exit':0,'remove_exit':0,'payload_hashes_verified':True,'separate_user_account':True,'os_policy_mutations':{'global_policy_changed':False,'owned_account_batch_logon_right':os.environ.get('INSONIC_CI_ORDINARY_SCHEDULER')=='1'},'profile_loaded':True,'user_profile':os.environ['USERPROFILE'],'elected_launcher':launch_evidence}
        output.write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
        record('completed')
    finally:
        if installed:run_installer(['/x',product],output.with_suffix('.cleanup.log'))

def private_desktop_sddl(user_sid,caller_sid=None):
    values=[]
    for sid in (user_sid,caller_sid):
        if sid is None:continue
        if not __import__('re').fullmatch(r'S-1-5-(?:\d+-)*\d+',sid):raise ValueError('invalid exact owned-desktop SID')
        if sid not in values:values.append(sid)
    return 'D:P(A;;GA;;;SY)'+''.join('(A;;GA;;;'+sid+')' for sid in values)+'S:(ML;;NW;;;ME)'

def private_context_desktop(user_sid,caller_sid=None,progress=None):
    """Create only owned CI objects, restoring the caller before running a child.

    Alternate-user processes require access to both objects. The inherited
    runner station need not grant access to the separately created CI account.
    Never alter shared WinSta0/default permissions or make this station visible.
    """
    m.disposable_runner();sddl=private_desktop_sddl(user_sid,caller_sid)
    from contextlib import contextmanager
    import ctypes as C
    import ctypes.wintypes as W
    @contextmanager
    def owned():
        user=C.WinDLL('user32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True)
        class Attributes(C.Structure):_fields_=[('size',W.DWORD),('descriptor',C.c_void_p),('inherit',W.BOOL)]
        methods={'GetProcessWindowStation':([],W.HANDLE),'SetProcessWindowStation':([W.HANDLE],W.BOOL),'CreateWindowStationW':([W.LPCWSTR,W.DWORD,W.DWORD,C.POINTER(Attributes)],W.HANDLE),'CreateDesktopW':([W.LPCWSTR,W.LPCWSTR,C.c_void_p,W.DWORD,W.DWORD,C.POINTER(Attributes)],W.HANDLE),'CloseWindowStation':([W.HANDLE],W.BOOL),'CloseDesktop':([W.HANDLE],W.BOOL),'GetThreadDesktop':([W.DWORD],W.HANDLE),'SetThreadDesktop':([W.HANDLE],W.BOOL)}
        for name,(args,result) in methods.items():function=getattr(user,name);function.argtypes=args;function.restype=result
        kernel.GetCurrentThreadId.argtypes=[];kernel.GetCurrentThreadId.restype=W.DWORD
        kernel.LocalFree.argtypes=[C.c_void_p];kernel.LocalFree.restype=C.c_void_p
        security.ConvertStringSecurityDescriptorToSecurityDescriptorW.argtypes=[W.LPCWSTR,W.DWORD,C.POINTER(C.c_void_p),C.POINTER(W.DWORD)];security.ConvertStringSecurityDescriptorToSecurityDescriptorW.restype=W.BOOL
        def checked(name,function,*args):
            if progress:progress(name)
            value=function(*args)
            if not value:
                error=C.WinError(C.get_last_error());error.api=name;raise error
            return value
        original=checked('GetProcessWindowStation',user.GetProcessWindowStation);thread_id=kernel.GetCurrentThreadId();original_desktop=checked('GetThreadDesktop',user.GetThreadDesktop,thread_id)
        descriptor=C.c_void_p();station=None;desktop=None;name='insonic-ci-'+uuid.uuid4().hex
        try:
            checked('ConvertStringSecurityDescriptorToSecurityDescriptorW-owned-desktop',security.ConvertStringSecurityDescriptorToSecurityDescriptorW,sddl,1,C.byref(descriptor),None)
            attrs=Attributes(C.sizeof(Attributes),descriptor,False)
            # CWF_CREATE_ONLY rejects an existing station instead of reusing it.
            station=checked('CreateWindowStationW-owned',user.CreateWindowStationW,name,1,0x000f037f,C.byref(attrs))
            try:
                checked('SetProcessWindowStation-owned',user.SetProcessWindowStation,station)
                desktop=checked('CreateDesktopW-owned',user.CreateDesktopW,'Default',None,None,0,0x000f01ff,C.byref(attrs))
            finally:
                checked('SetProcessWindowStation-restore',user.SetProcessWindowStation,original)
                if user.GetThreadDesktop(thread_id)!=original_desktop:checked('SetThreadDesktop-restore',user.SetThreadDesktop,original_desktop)
            if user.GetProcessWindowStation()!=original or user.GetThreadDesktop(thread_id)!=original_desktop:raise RuntimeError('caller station/desktop restoration failed')
            roles=['SYSTEM','ordinary target user']+(['calling CI user'] if caller_sid and caller_sid!=user_sid else [])
            evidence={'path':name+'\\Default','kind':'owned-noninteractive-actor-desktop','shared_desktop_acl_modified':False,'caller_station_restored':True,'visible_switch':False,'acl_subjects':roles,'target_user_sid_sha256':m.digest(user_sid.encode()),'caller_sid_sha256':m.digest(caller_sid.encode()) if caller_sid else None,'actual_station_security':owned_user_object_observation(station,user,security,kernel),'actual_desktop_security':owned_user_object_observation(desktop,user,security,kernel)}
            yield evidence
        finally:
            if desktop:user.CloseDesktop(desktop)
            if station:user.CloseWindowStation(station)
            if descriptor:kernel.LocalFree(descriptor)
    return owned()

def ordinary_environment(profile,account,temporary):
    values={key:os.environ[key] for key in ('SystemRoot','windir','COMPUTERNAME','SystemDrive','ProgramData','ProgramFiles','ProgramFiles(x86)','CommonProgramFiles','CommonProgramFiles(x86)','PROCESSOR_ARCHITECTURE','OS','NUMBER_OF_PROCESSORS','GITHUB_ACTIONS','RUNNER_ENVIRONMENT','INSONIC_DISPOSABLE_INSTALLER_TEST','GITHUB_RUN_ID') if key in os.environ}
    profile=Path(profile);values.update({'USERPROFILE':str(profile),'HOME':str(profile),'APPDATA':str(profile/'AppData/Roaming'),'LOCALAPPDATA':str(profile/'AppData/Local'),'USERNAME':account,'USERDOMAIN':os.environ['COMPUTERNAME'],'TEMP':str(temporary),'TMP':str(temporary),'PATH':str(Path(os.environ['SystemRoot'])/'System32')+';'+os.environ['SystemRoot']})
    return '\0'.join(key+'='+value for key,value in sorted(values.items(),key=lambda item:item[0].casefold()))+'\0\0'

def account_sid(token):
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    security=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True)
    security.GetTokenInformation.argtypes=[W.HANDLE,C.c_int,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];security.GetTokenInformation.restype=W.BOOL
    security.ConvertSidToStringSidW.argtypes=[C.c_void_p,C.POINTER(C.c_void_p)];security.ConvertSidToStringSidW.restype=W.BOOL
    kernel.LocalFree.argtypes=[C.c_void_p];kernel.LocalFree.restype=C.c_void_p
    size=W.DWORD();security.GetTokenInformation(token,1,None,0,C.byref(size))
    if not size.value or size.value>65536:raise RuntimeError('invalid account-token SID bound')
    data=C.create_string_buffer(size.value)
    if not security.GetTokenInformation(token,1,data,size.value,C.byref(size)):raise C.WinError(C.get_last_error())
    text=C.c_void_p()
    if not security.ConvertSidToStringSidW(C.cast(data,C.POINTER(C.c_void_p))[0],C.byref(text)):raise C.WinError(C.get_last_error())
    try:return C.wstring_at(text)
    finally:kernel.LocalFree(text)

def owned_batch_right(sid,change=None):
    """Query or change only SeBatchLogonRight of the newly created CI actor."""
    m.disposable_runner()
    if not __import__('re').fullmatch(r'S-1-5-21-(?:\d+-){3}\d+',sid) or change not in (None,'add','remove'):raise ValueError('invalid owned account right operation')
    import ctypes as C
    import ctypes.wintypes as W
    dll=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True)
    class Attributes(C.Structure):_fields_=[('length',W.ULONG),('root',W.HANDLE),('name',C.c_void_p),('attributes',W.ULONG),('descriptor',C.c_void_p),('quality',C.c_void_p)]
    class String(C.Structure):_fields_=[('length',W.USHORT),('maximum',W.USHORT),('buffer',C.c_void_p)]
    dll.ConvertStringSidToSidW.argtypes=[W.LPCWSTR,C.POINTER(C.c_void_p)];dll.ConvertStringSidToSidW.restype=W.BOOL
    dll.LsaOpenPolicy.argtypes=[C.c_void_p,C.POINTER(Attributes),W.ULONG,C.POINTER(C.c_void_p)];dll.LsaOpenPolicy.restype=W.ULONG
    dll.LsaEnumerateAccountRights.argtypes=[C.c_void_p,C.c_void_p,C.POINTER(C.c_void_p),C.POINTER(W.ULONG)];dll.LsaEnumerateAccountRights.restype=W.ULONG
    for name in ('LsaAddAccountRights','LsaRemoveAccountRights'):
        getattr(dll,name).argtypes=[C.c_void_p,C.c_void_p,*([W.BOOL] if name=='LsaRemoveAccountRights' else []),C.POINTER(String),W.ULONG];getattr(dll,name).restype=W.ULONG
    dll.LsaNtStatusToWinError.argtypes=[W.ULONG];dll.LsaNtStatusToWinError.restype=W.ULONG
    dll.LsaClose.argtypes=[C.c_void_p];dll.LsaFreeMemory.argtypes=[C.c_void_p];kernel.LocalFree.argtypes=[C.c_void_p]
    native_sid=C.c_void_p();policy=C.c_void_p();allocation=C.c_void_p()
    def checked(result):
        if result:raise C.WinError(dll.LsaNtStatusToWinError(result))
    if not dll.ConvertStringSidToSidW(sid,C.byref(native_sid)):raise C.WinError(C.get_last_error())
    try:
        attributes=Attributes();attributes.length=C.sizeof(attributes);checked(dll.LsaOpenPolicy(None,C.byref(attributes),0x810 if change=='add' else 0x800,C.byref(policy)))
        if change:
            text=C.create_unicode_buffer('SeBatchLogonRight');value=String(len('SeBatchLogonRight')*2,C.sizeof(text),C.cast(text,C.c_void_p))
            checked(dll.LsaAddAccountRights(policy,native_sid,C.byref(value),1) if change=='add' else dll.LsaRemoveAccountRights(policy,native_sid,False,C.byref(value),1))
        count=W.ULONG();result=dll.LsaEnumerateAccountRights(policy,native_sid,C.byref(allocation),C.byref(count))
        if result==0xc0000034:return []
        checked(result)
        if count.value>128:raise RuntimeError('account rights observation exceeds bound')
        return sorted(C.wstring_at(item.buffer,item.length//2) for item in C.cast(allocation,C.POINTER(String))[:count.value])
    finally:
        if allocation:dll.LsaFreeMemory(allocation)
        if policy:dll.LsaClose(policy)
        kernel.LocalFree(native_sid)

SCHEDULER_BRIDGE=r'''
$ErrorActionPreference='Stop'
[Console]::InputEncoding=[Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding=[Text.UTF8Encoding]::new($false)
$inputObject=[Console]::In.ReadToEnd() | ConvertFrom-Json
$service=New-Object -ComObject Schedule.Service
$service.Connect()
$folder=$service.GetFolder('\')
$created=$false; $running=$null; $result=@{phase='starting';task_name=$inputObject.task_name;deleted=$false}
try {
  if ($inputObject.task_name -notmatch '^insonic-ci-ordinary-[0-9a-f]{32}$') { throw 'invalid owned task name' }
  if ($inputObject.mode -eq 'cleanup') {
    try { $existing=$folder.GetTask($inputObject.task_name) } catch { if ($_.Exception.HResult -eq -2147024894) { $existing=$null } else { throw } }
    if ($null -ne $existing) {
      if ($existing.Definition.RegistrationInfo.Description -ne $inputObject.description) { throw 'task ownership mismatch' }
      $principal=$existing.Definition.Principal.UserId
      if ($principal -notmatch '^S-1-') { $principal=([Security.Principal.NTAccount]::new($principal)).Translate([Security.Principal.SecurityIdentifier]).Value }
      if ($principal -ne $inputObject.sid) { throw 'task actor mismatch' }
      if ($existing.State -eq 2 -or $existing.State -eq 4) { $existing.Stop(0) };$folder.DeleteTask($inputObject.task_name,0)
    }
    $result.deleted=$true;$result.phase='cleanup-completed'
  } else {
    $definition=$service.NewTask(0)
    $definition.RegistrationInfo.Description=$inputObject.description
    $definition.Principal.UserId=$inputObject.account;$definition.Principal.LogonType=1;$definition.Principal.RunLevel=0
    $definition.Settings.Enabled=$true;$definition.Settings.Hidden=$true;$definition.Settings.ExecutionTimeLimit='PT90S'
    $definition.Settings.DisallowStartIfOnBatteries=$false;$definition.Settings.StopIfGoingOnBatteries=$false
    $action=$definition.Actions.Create(0);$action.Path=$inputObject.executable;$action.Arguments=$inputObject.arguments;$action.WorkingDirectory=$inputObject.directory
    $registered=$folder.RegisterTaskDefinition($inputObject.task_name,$definition,2,$inputObject.account,$inputObject.password,1,$null)
    $created=$true;$inputObject.password=$null;$result.phase='registered';$result.requested_run_level=0;$result.logon_type=1
    $running=$registered.Run($null);$deadline=[DateTime]::UtcNow.AddSeconds(95)
    do { Start-Sleep -Milliseconds 100;$running.Refresh();if ([DateTime]::UtcNow -gt $deadline) { throw 'owned ordinary task exceeded bound' } } while ($running.State -eq 2 -or $running.State -eq 4)
    $result.exit_code=$registered.LastTaskResult;$result.phase='completed'
  }
} catch { $result.phase='failed';$result.hresult=$_.Exception.HResult }
finally {
  $inputObject.password=$null
  if ($created) {
    try { if ($null -ne $running) { $running.Refresh();if ($running.State -eq 2 -or $running.State -eq 4) { $running.Stop() } };$folder.DeleteTask($inputObject.task_name,0);$result.deleted=$true } catch { $result.cleanup_hresult=$_.Exception.HResult }
  }
}
$result | ConvertTo-Json -Compress
'''

def scheduler_ordinary(stage,name,sid,password,args,record,profile):
    """Run one least-privilege password task; credentials travel only via stdin."""
    m.disposable_runner();stage=Path(stage)
    marker=json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'))
    if marker!={'account':name,'sid_sha256':m.digest(sid.encode())} or stage.is_symlink() or stage.resolve().parent!=Path(os.environ['ProgramData']).resolve() or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name):raise ValueError('ordinary scheduler target is not the exact owned account stage')
    import base64
    executable=Path(os.environ['SystemRoot'])/'System32/WindowsPowerShell/v1.0/powershell.exe'
    command=[str(executable),'-NoLogo','-NoProfile','-NonInteractive','-EncodedCommand',base64.b64encode(SCHEDULER_BRIDGE.encode('utf-16le')).decode('ascii')]
    task='insonic-ci-ordinary-'+uuid.uuid4().hex;description='insonic disposable ordinary actor '+m.digest(sid.encode())+' '+task
    environment=dict(item.split('=',1) for item in ordinary_environment(profile,name,stage/'temp').split('\0') if item);environment['INSONIC_CI_ORDINARY_SCHEDULER']='1'
    code='import os,runpy,sys;os.environ.clear();os.environ.update('+repr(environment)+');sys.argv='+repr(list(map(str,args)))+';sys.path.insert(0,os.path.dirname(sys.argv[0]));runpy.run_path(sys.argv[0],run_name="__main__")'
    metadata={'task_name':task,'description':description,'sid':sid}
    request={**metadata,'mode':'run','account':os.environ['COMPUTERNAME']+'\\'+name,'password':password,'executable':sys.executable,'arguments':subprocess.list2cmdline(['-c',code]),'directory':str(stage)}
    outcome=None
    def invoke(value):
        completed=subprocess.run(command,input=json.dumps(value).encode('utf-8'),stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=110)
        if completed.returncode:raise RuntimeError('hidden ordinary scheduler bridge failed '+str(completed.returncode))
        return json.loads(completed.stdout.decode('utf-8-sig'))
    record('TaskScheduler-ordinary-before-register',task_name=task,password_published=False,requested_run_level=0,logon_type=1)
    try:
        outcome=invoke(request);record('TaskScheduler-ordinary-result',scheduler=outcome)
        if outcome.get('phase')!='completed' or outcome.get('exit_code')!=0 or outcome.get('deleted') is not True:raise RuntimeError('ordinary scheduled context did not complete: '+json.dumps(outcome,sort_keys=True))
        return outcome
    finally:
        request['password']=None
        if outcome is None or outcome.get('deleted') is not True:
            cleanup=invoke({**metadata,'mode':'cleanup'});record('TaskScheduler-owned-cleanup',scheduler=cleanup)
            if cleanup.get('deleted') is not True:raise RuntimeError('owned ordinary task cleanup failed')

def retain_ordinary_diagnostics(native,output):
    """Keep each actor's exact evidence distinct before deleting its stage."""
    for item in Path(native).iterdir():
        if item.is_file() and item.name.startswith('context.'):
            if item.is_symlink():raise ValueError('ordinary diagnostic source is redirected')
            shutil.copyfile(item,Path(output).with_name(Path(output).stem+item.name[len('context'):]))

def grant_owned_account_stage(stage,user_sid,caller_sid,progress=None):
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    base=Path(os.environ['ProgramData']).resolve();stage=Path(stage)
    if stage.is_symlink() or stage.resolve().parent!=base or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name):raise ValueError('account staging ACL target is not the exact owned directory')
    for sid in (user_sid,caller_sid):private_desktop_sddl(sid)
    security=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True)
    security.ConvertStringSecurityDescriptorToSecurityDescriptorW.argtypes=[W.LPCWSTR,W.DWORD,C.POINTER(C.c_void_p),C.c_void_p];security.ConvertStringSecurityDescriptorToSecurityDescriptorW.restype=W.BOOL
    security.SetFileSecurityW.argtypes=[W.LPCWSTR,W.DWORD,C.c_void_p];security.SetFileSecurityW.restype=W.BOOL
    kernel.LocalFree.argtypes=[C.c_void_p];kernel.LocalFree.restype=C.c_void_p
    descriptor=C.c_void_p();sddl='D:P(A;OICI;GA;;;SY)(A;OICI;GA;;;'+user_sid+')(A;OICI;GA;;;'+caller_sid+')S:(ML;OICI;NW;;;ME)'
    if progress:progress('ConvertStringSecurityDescriptorToSecurityDescriptorW-owned-stage')
    if not security.ConvertStringSecurityDescriptorToSecurityDescriptorW(sddl,1,C.byref(descriptor),None):
        error=C.WinError(C.get_last_error());error.api='ConvertStringSecurityDescriptorToSecurityDescriptorW-owned-stage';raise error
    try:
        if progress:progress('SetFileSecurityW-owned-stage-DACL-label')
        if not security.SetFileSecurityW(str(stage),0x80000014,descriptor):
            error=C.WinError(C.get_last_error());error.api='SetFileSecurityW-owned-stage-DACL-label';raise error
    finally:kernel.LocalFree(descriptor)

def validate_account_profile(account,profile):
    if not __import__('re').fullmatch('inso_ci_[0-9a-f]{12}',account):raise ValueError('invalid exact CI-owned account identity')
    profile=Path(profile);base=Path(os.environ['USERPROFILE']).resolve().parent
    if profile.is_symlink() or profile.resolve().parent!=base or profile.name.casefold()!=account.casefold():raise ValueError('profile cleanup target is not the exact created account directory')
    return profile

def ordinary_context(installer,directory,output,model=None):
    """Qualify an actual ordinary account, exclusively on a disposable runner."""
    m.disposable_runner()
    import secrets
    import ctypes as C
    import ctypes.wintypes as W
    security=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True);network=C.WinDLL('netapi32',use_last_error=True);profiles=C.WinDLL('userenv',use_last_error=True)
    class User(C.Structure):_fields_=[('name',W.LPWSTR),('password',W.LPWSTR),('password_age',W.DWORD),('privilege',W.DWORD),('home',W.LPWSTR),('comment',W.LPWSTR),('flags',W.DWORD),('script',W.LPWSTR)]
    class Member(C.Structure):_fields_=[('name',W.LPWSTR)]
    class Startup(C.Structure):_fields_=[('cb',W.DWORD),('reserved',W.LPWSTR),('desktop',W.LPWSTR),('title',W.LPWSTR),('x',W.DWORD),('y',W.DWORD),('width',W.DWORD),('height',W.DWORD),('chars_x',W.DWORD),('chars_y',W.DWORD),('fill',W.DWORD),('flags',W.DWORD),('show',W.WORD),('reserved_size',W.WORD),('reserved_bytes',C.c_void_p),('stdin',W.HANDLE),('stdout',W.HANDLE),('stderr',W.HANDLE)]
    class Process(C.Structure):_fields_=[('process',W.HANDLE),('thread',W.HANDLE),('pid',W.DWORD),('tid',W.DWORD)]
    kernel.GetCurrentProcess.restype=W.HANDLE;kernel.CloseHandle.argtypes=[W.HANDLE];kernel.CloseHandle.restype=W.BOOL
    kernel.WaitForSingleObject.argtypes=[W.HANDLE,W.DWORD];kernel.WaitForSingleObject.restype=W.DWORD;kernel.GetExitCodeProcess.argtypes=[W.HANDLE,C.POINTER(W.DWORD)];kernel.GetExitCodeProcess.restype=W.BOOL;kernel.TerminateProcess.argtypes=[W.HANDLE,W.UINT];kernel.TerminateProcess.restype=W.BOOL
    security.OpenProcessToken.argtypes=[W.HANDLE,W.DWORD,C.POINTER(W.HANDLE)];security.OpenProcessToken.restype=W.BOOL
    security.LogonUserW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.LPCWSTR,W.DWORD,W.DWORD,C.POINTER(W.HANDLE)];security.LogonUserW.restype=W.BOOL
    security.CreateProcessWithLogonW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.LPCWSTR,W.DWORD,W.LPCWSTR,W.LPWSTR,W.DWORD,C.c_void_p,W.LPCWSTR,C.POINTER(Startup),C.POINTER(Process)];security.CreateProcessWithLogonW.restype=W.BOOL
    network.NetUserAdd.argtypes=[W.LPCWSTR,W.DWORD,C.c_void_p,C.POINTER(W.DWORD)];network.NetUserAdd.restype=W.DWORD;network.NetUserDel.argtypes=[W.LPCWSTR,W.LPCWSTR];network.NetUserDel.restype=W.DWORD
    network.NetLocalGroupAddMembers.argtypes=[W.LPCWSTR,W.LPCWSTR,W.DWORD,C.c_void_p,W.DWORD];network.NetLocalGroupAddMembers.restype=W.DWORD
    profiles.GetUserProfileDirectoryW.argtypes=[W.HANDLE,W.LPWSTR,C.POINTER(W.DWORD)];profiles.GetUserProfileDirectoryW.restype=W.BOOL;profiles.DeleteProfileW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.LPCWSTR];profiles.DeleteProfileW.restype=W.BOOL
    name='inso_ci_'+uuid.uuid4().hex[:12];password='Aa1!'+secrets.token_hex(24);created=False;profile=None;stage=None;token=W.HANDLE();caller=W.HANDLE();process=Process();sid=None;evidence=None;failure=None;cleanup=[];batch_original=None
    diagnostics=Path(output).with_suffix('.startup.json');current_phase={'name':'starting'};history=[]
    def record(phase,**extra):
        current_phase['name']=phase;history.append({'phase':phase,**extra});diagnostics.write_text(json.dumps({'phase':phase,'actor':'actual ordinary disposable CI account','separate_user_account':True,'password_published':False,'boundaries':history,**extra},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    record('before-account-create')
    try:
        info=User(name,password,0,1,None,'Disposable insonic MSI qualification',0x10201,None);parameter=W.DWORD();record('NetUserAdd');rc=network.NetUserAdd(None,1,C.byref(info),C.byref(parameter));record('NetUserAdd-result',api_status=rc)
        if rc:raise RuntimeError('ordinary account creation failed '+str(rc)+'; no existing account is reused')
        created=True;member=Member(os.environ['COMPUTERNAME']+'\\'+name);record('NetLocalGroupAddMembers-Users');rc=network.NetLocalGroupAddMembers(None,'Users',3,C.byref(member),1);record('NetLocalGroupAddMembers-result',api_status=rc)
        if rc not in (0,1378):raise RuntimeError('ordinary Users membership failed '+str(rc))
        record('LogonUserW-ordinary')
        if not security.LogonUserW(name,'.',password,2,0,C.byref(token)):raise C.WinError(C.get_last_error())
        record('GetTokenUser-ordinary')
        sid=account_sid(token)
        record('OpenProcessToken-calling-CI-user')
        if not security.OpenProcessToken(kernel.GetCurrentProcess(),8,C.byref(caller)):raise C.WinError(C.get_last_error())
        record('GetTokenUser-calling-CI-user')
        caller_sid=account_sid(caller)
        if sid==caller_sid:raise RuntimeError('ordinary account unexpectedly reused the elevated actor')
        stage=Path(os.environ['ProgramData'])/('insonic-ci-account-'+uuid.uuid4().hex);record('Create-owned-account-stage');stage.mkdir(exist_ok=False);grant_owned_account_stage(stage,sid,caller_sid,record)
        (stage/'owned-ci-account.json').write_text(json.dumps({'account':name,'sid_sha256':m.digest(sid.encode())}),encoding='utf-8',newline='\n')
        # LOGON_WITH_PROFILE lets Windows create/load the real ordinary profile.
        # This bootstrap uses its native profile environment, before we rebuild
        # the worker environment with the SDK-observed account directory.
        record('Create-private-profile-bootstrap-desktop')
        with private_context_desktop(sid,caller_sid,record) as desktop:
            startup=Startup();startup.cb=C.sizeof(startup);startup.flags=1;startup.show=0;startup.desktop=desktop['path']
            whoami=Path(os.environ['SystemRoot'])/'System32/whoami.exe';command=C.create_unicode_buffer(subprocess.list2cmdline([str(whoami),'/user']))
            record('before-native-profile-logon',desktop=desktop,same_user_sid_sha256=m.digest(sid.encode()))
            record('CreateProcessWithLogonW-profile-bootstrap')
            if not security.CreateProcessWithLogonW(name,'.',password,1,str(whoami),command,0x08000400,None,str(stage),C.byref(startup),C.byref(process)):raise C.WinError(C.get_last_error())
            record('WaitForSingleObject-profile-bootstrap',pid=process.pid)
            if kernel.WaitForSingleObject(process.process,15000)!=0:
                kernel.TerminateProcess(process.process,1);kernel.WaitForSingleObject(process.process,5000);raise RuntimeError('owned profile bootstrap exceeded its bound')
            code=W.DWORD()
            record('GetExitCodeProcess-profile-bootstrap',pid=process.pid)
            if not kernel.GetExitCodeProcess(process.process,C.byref(code)) or code.value!=0:raise RuntimeError('native ordinary profile bootstrap failed '+str(code.value))
            value=C.create_unicode_buffer(4096);size=W.DWORD(4096)
            record('GetUserProfileDirectoryW-ordinary')
            if not profiles.GetUserProfileDirectoryW(token,value,C.byref(size)):raise C.WinError(C.get_last_error())
            profile=validate_account_profile(name,value.value)
            if not profile.is_dir():raise RuntimeError('native profile logon did not create the observed ordinary-user directory')
            record('ordinary-profile-observed',profile=str(profile),method='CreateProcessWithLogonW LOGON_WITH_PROFILE, GetUserProfileDirectoryW')
            kernel.CloseHandle(process.thread);kernel.CloseHandle(process.process);process=Process()
        record('Copy-exact-owned-worker-inputs');scripts=stage/'scripts';scripts.mkdir();(stage/'temp').mkdir();native=stage/'native';native.mkdir()
        for filename in ('qualify-windows-msi.py','windows_msi.py'):
            shutil.copyfile(m.ROOT/'scripts'/filename,scripts/filename)
            if m.sha(scripts/filename)!=m.sha(m.ROOT/'scripts'/filename):raise RuntimeError('ordinary staged script identity differs')
        target=stage/'fixture.msi';shutil.copyfile(installer,target)
        if m.sha(target)!=m.sha(Path(installer)):raise RuntimeError('ordinary MSI fixture copy differs')
        fixture=Path(directory).parent/'launcher-context-fixture';shutil.copytree(fixture,stage/'launcher-context-fixture')
        for original in fixture.iterdir():
            if original.is_file() and m.sha(original)!=m.sha(stage/'launcher-context-fixture'/original.name):raise RuntimeError('ordinary launcher fixture copy differs')
        (stage/'context-model.json').write_text(json.dumps(model),encoding='utf-8',newline='\n')
        install=stage/'installed program é with spaces';worker_output=native/'context.json'
        record('LsaEnumerateAccountRights-owned-before');batch_original=owned_batch_right(sid)
        if 'SeBatchLogonRight' not in batch_original:
            record('LsaAddAccountRights-owned-batch');owned_batch_right(sid,'add')
        record('owned-batch-logon-right-observed',original_rights=batch_original,current_rights=owned_batch_right(sid),global_policy_changed=False)
        args=[str(scripts/'qualify-windows-msi.py'),'--ordinary-worker',str(target),'--context-dir',str(install),'--output',str(worker_output)]
        scheduler=scheduler_ordinary(stage,name,sid,password,args,record,profile)
        evidence=json.loads(worker_output.read_text(encoding='utf-8'))
        if evidence['token']['user_sid_sha256']!=m.digest(sid.encode()) or Path(evidence['user_profile']).resolve()!=profile.resolve():raise RuntimeError('ordinary worker changed account/profile identity')
        evidence.update({'process_creation':'Task Scheduler password logon, least privilege, OS-created noninteractive session','scheduler':scheduler,'profile_creation':'native LOGON_WITH_PROFILE bootstrap and observed GetUserProfileDirectoryW','fixture_msi_sha256':m.sha(target),'fixture_script_sha256':m.sha(scripts/'qualify-windows-msi.py'),'elevated_actor_sid_sha256':m.digest(caller_sid.encode()),'owned_account_rights_before':batch_original})
    except BaseException as error:
        failure=error;record('failed',failed_api=getattr(error,'api',current_phase['name']),winerror=getattr(error,'winerror',None),errno=getattr(error,'errno',None),error_type=type(error).__name__,error=str(error))
    finally:
        if stage is not None and (stage/'native').is_dir():
            retain_ordinary_diagnostics(stage/'native',output)
        for handle in (process.thread,process.process,token,caller):
            if handle:kernel.CloseHandle(handle)
        if batch_original is not None:
            try:
                restored=owned_batch_right(sid)
                if 'SeBatchLogonRight' not in batch_original and 'SeBatchLogonRight' in restored:restored=owned_batch_right(sid,'remove')
                cleanup.append({'object':'owned account batch logon right','restored':restored==batch_original,'original_rights':batch_original,'final_rights':restored})
                if restored!=batch_original and failure is None:failure=RuntimeError('owned ordinary account rights were not restored')
            except BaseException as error:
                cleanup.append({'object':'owned account batch logon right','restored':False,'error_type':type(error).__name__})
                if failure is None:failure=error
        if profile is not None:
            validate_account_profile(name,profile);deleted=False
            for _ in range(50):
                if profiles.DeleteProfileW(sid,str(profile),None):deleted=True;break
                time.sleep(.2)
            cleanup.append({'object':'created profile','deleted':deleted,'sid_sha256':m.digest(sid.encode())})
            if not deleted and failure is None:failure=RuntimeError('owned ordinary profile cleanup failed '+str(C.get_last_error()))
        if created:
            rc=network.NetUserDel(None,name);cleanup.append({'object':'created account','deleted':rc==0})
            if rc and failure is None:failure=RuntimeError('owned ordinary account cleanup failed '+str(rc))
        if stage is not None:
            base=Path(os.environ['ProgramData']).resolve()
            if stage.is_symlink() or stage.resolve().parent!=base or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name):raise ValueError('owned stage cleanup target changed')
            shutil.rmtree(stage);cleanup.append({'object':'owned staging directory','deleted':True})
        password=None
        Path(output).with_suffix('.cleanup.json').write_text(json.dumps({'cleanup':cleanup,'failure':str(failure) if failure else None},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    if failure is not None:raise failure
    evidence['cleanup']=cleanup;Path(output).write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n');return evidence

def cancellation_worker(installer,directory,output):
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    dll=native_installer_dll(output.with_suffix('.msi.log'))
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
    dll.MsiSetExternalUIW(callback,(1<<8)|(1<<10),None)
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
    def phase(name,**extra):
        (logs/'lifecycle-progress.json').write_text(json.dumps({'artifact_sha256':receipt['artifact_sha256'],'revision':receipt['revision'],'fixture_upgrade_code':upgrade,'phase':name,'completed_checks':dict(result),**extra},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    with tempfile.TemporaryDirectory(prefix='insonic-ci-msi-') as tmp:
        tmp=Path(tmp);install=tmp/'installed program é with spaces';profile=tmp/'fixture profile';data=tmp/'library';data.mkdir();sentinel=data/'accepted-user-data';sentinel.write_bytes(b'preserve library media settings models credentials')
        env=dict(os.environ);env.update({'APPDATA':str(profile/'Roaming'),'LOCALAPPDATA':str(profile/'Local')});env.pop('INSONIC_DESKTOP_FIXTURE_DIRECTORY',None)
        old=tmp/'old.msi';new=tmp/'new.msi';bad=tmp/'failed-upgrade.msi';missing=tmp/'missing-prerequisite.msi'
        clone(path,old,version,upgrade);clone(path,new,new_version,upgrade);clone(path,bad,new_version,upgrade,failure=True);clone(path,missing,version,upgrade,missing_prerequisites=True)
        active=None;installed=None;integration_sentinels=None;collision_shortcut=None
        def call(msi,selection=None,**properties):
            args=['/i',msi,'INSTALLDIR='+str(install),'LICENSE_ACCEPTED=1']
            if selection is not None:args+=['ADDLOCAL='+selection]
            args += [k+'='+v for k,v in properties.items()]
            return args
        try:
            phase('administrator-prerequisite-rejections')
            run_installer(['/i',old,'INSTALLDIR='+str(install),'ADDLOCAL=Core'],logs/'license-rejected.log',False,'Read and accept Apache License 2.0')
            run_installer(call(old,'Core'),logs/'empty-selection-rejected.log',False,'Select Desktop, CLI integrations, or both.')
            run_installer(call(old,'Desktop,CLI'),logs/'missing-core-rejected.log',False,'Shared runtime is mandatory.')
            run_installer(call(missing,'Core,CLI'),logs/'prerequisite-rejected.log',False,'Install Microsoft Visual C++ v14 x64 Redistributable')
            if install.exists() and any(install.iterdir()):raise ValueError('invalid fixture left program files')
            result['license_and_prerequisites']='passed'
            collision_shortcut=Path(os.environ['APPDATA'])/'Microsoft/Windows/Start Menu/Programs/insonic/insonic CLI.lnk'
            collision_shortcut.parent.mkdir(parents=True,exist_ok=True)
            collision_bytes=b'unrelated preexisting exact-name CLI shortcut fixture'
            with collision_shortcut.open('xb') as stream:stream.write(collision_bytes)
            run_installer(call(old,'Core,CLI',START_MENU='1',DESKTOP_SHORTCUT='0'),logs/'shortcut-collision-rejected.log',False,'If a shortcut already exists, deselect its Start menu or Desktop integration','CheckCliMenu')
            if collision_shortcut.read_bytes()!=collision_bytes:raise ValueError('MSI overwrote an unrelated exact-name shortcut')
            if install.exists() and any(install.iterdir()):raise ValueError('shortcut rejection left program files')
            collision_shortcut.unlink();collision_shortcut=None
            for selection,desktop in [('Core,CLI',False),('Core,Desktop',True),('Core,Desktop,CLI',True)]:
                run_installer(call(old,selection),logs/('selection-'+selection.replace(',','-')+'.log'));installed=m.product_code(version,upgrade);installed_hashes(install,model,desktop)
                if child([install/'insonic.exe','version'],env,install).decode().strip()!=version:raise ValueError('installed CLI version differs')
                plan=json.loads(child([install/'insonic-launcher.exe','--describe','--desktop' if desktop else '--cli'],env,install))
                if not plan:raise ValueError('installed launcher describe returned no plan')
                if selection=='Core,Desktop,CLI':
                    fixture_evidence=prepare_launcher_fixture(install,tmp/'launcher-context-fixture')
                    elevated_launch_evidence=elected_launcher_context(tmp/'launcher-context-fixture',allow_refusal=True)
                run_installer(['/x',installed],logs/'selection-remove.log');installed=None
                if any((install/f['path']).exists() for f in model['payload']):raise ValueError('uninstall retained installer-owned bytes')
            result['client_selections']='passed';result['launch_contract']='passed'
            phase('administrator-maintenance')
            run_installer(call(old,'Core,CLI',START_MENU='',DESKTOP_SHORTCUT=''),logs/'maintenance-initial.log');installed=m.product_code(version,upgrade);assert_unchecked_shortcuts()
            registered_feature_states(installed,logs/'feature-change.registered.json')
            run_installer(call(old,'Core,Desktop,CLI',REINSTALL='Core',REINSTALLMODE='vomus'),logs/'feature-change.log');installed_hashes(install,model,True)
            run_installer(call(old,REMOVE='Core'),logs/'maintenance-core-removal-rejected.log',False,'Shared runtime is mandatory.');installed_hashes(install,model,True)
            run_installer(call(old,REMOVE='Desktop,CLI'),logs/'maintenance-all-clients-removal-rejected.log',False,'Select Desktop, CLI integrations, or both.');installed_hashes(install,model,True)
            (install/'insonic-session.cmd').unlink();(install/m.OWNERSHIP).unlink();run_installer(['/fa',installed],logs/'repair.log');installed_hashes(install,model,True);result['maintenance']='passed'
            integration_sentinels=create_removal_sentinels()
            # A real running console image under the program tree represents an
            # active client/runtime. No application termination is delegated to MSI.
            active_path=install/'active-owned-ci-fixture.exe';shutil.copyfile(Path(os.environ['SystemRoot'])/'System32/ping.exe',active_path)
            active=subprocess.Popen([str(active_path),'127.0.0.1','-n','60'],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW)
            time.sleep(.15)
            alternate_install=tmp/'alternate new program directory é'
            run_installer(['/i',new,'INSTALLDIR='+str(alternate_install),'LICENSE_ACCEPTED=1'],logs/'active-old-root-upgrade-rejected.log',False,'Close insonic clients, finish/cancel jobs','Preflight')
            assert_effective_directory(logs/'active-old-root-upgrade-rejected.log',alternate_install)
            if active.poll() is not None:raise ValueError('alternate-target upgrade terminated active old-root work')
            installed_hashes(install,model,True)
            if alternate_install.exists():raise ValueError('rejected alternate-target upgrade created a new installation directory')
            assert_removal_sentinels(integration_sentinels)
            run_installer(['/x',installed],logs/'active-work-rejected.log',False,'Close insonic clients, finish/cancel jobs','Preflight')
            if active.poll() is not None:raise ValueError('MSI terminated active work')
            installed_hashes(install,model,True);active.terminate();active.wait(timeout=5);active=None;active_path.unlink();result['active_work_preserved']='passed'
            phase('administrator-cancellation-and-rollback')
            # Deliberately remove Desktop before upgrading to prove feature migration.
            run_installer(call(old,'Core,CLI',REMOVE='Desktop',REINSTALLMODE='vomus'),logs/'cli-election.log');installed_hashes(install,model,False)
            cancellation_evidence=cancellation(new,install,logs/'cancelled-upgrade.json');installed_hashes(install,model,False)
            if sentinel.read_bytes()!=b'preserve library media settings models credentials':raise ValueError('cancelled MSI changed user data')
            result['cancellation_rollback']='passed'
            run_installer(call(bad),logs/'upgrade-rollback.log',False,'Intentional transactional fixture failure','FixtureFailure');installed_hashes(install,model,False)
            run_installer(call(new),logs/'upgrade-success.log');installed=m.product_code(new_version,upgrade);installed_hashes(install,model,False);assert_unchecked_shortcuts();result['upgrade_rollback']='passed'
            run_installer(call(old),logs/'downgrade-rejected.log',False,'A newer insonic version is installed.');installed_hashes(install,model,False);result['downgrade_rejection']='passed'
            final_shortcut_elections=saved_shortcut_elections()
            run_installer(['/x',installed],logs/'final-remove.log');installed=None
            if sentinel.read_bytes()!=b'preserve library media settings models credentials':raise ValueError('MSI modified user data')
            if any((install/f['path']).exists() for f in model['payload']):raise ValueError('MSI removal retained owned files')
            assert_removal_sentinels(integration_sentinels)
            result['user_data_preserved']='passed'
            phase('ordinary-user-install-remove-and-elected-launch')
            context_evidence=ordinary_context(old,tmp/'ordinary per-user installation',logs/'ordinary-context.json',model);result['user_context']='passed'
            proof={'artifact_sha256':receipt['artifact_sha256'],'revision':receipt['revision'],'windows_build':str(__import__('sys').getwindowsversion().build),'fixture_upgrade_code':upgrade,'checks':result,'elapsed_seconds':round(time.monotonic()-started,3),'active_work_fixture':{'process':'running owned ping image, no process shutdown','rejected_actions':['alternate-install-directory upgrade','uninstall'],'alternate_directory_absent':True,'old_payload_hashes_preserved':True},'exact_shortcut_collision':{'fresh_cli_exact_name':'rejected before installation changes','unrelated_bytes':'preserved','failed_action':'CheckCliMenu'},'unrelated_integrations':{'start_menu_sibling':'preserved','desktop_sibling':'preserved','product_key_sibling_value':'preserved','sentinel_content_sha256':m.digest(integration_sentinels['content'])},'shortcut_migration':{'fresh_unchecked_properties':'empty values normalized to zero','upgrade_overrides':False,'saved_elections':final_shortcut_elections},'policy_prompts':'not-observed','signing':'unsigned','cancellation':cancellation_evidence,'user_context':{'original_elevated_token':original_token,'ordinary_process':context_evidence},'elected_launcher':{'fixture':fixture_evidence,'elevated_caller':elevated_launch_evidence,'ordinary_caller':context_evidence['elected_launcher']}}
            phase('completed')
        except BaseException as error:
            phase('failed',error_type=type(error).__name__,error=str(error)[-4000:]);raise
        finally:
            if active is not None:active.terminate();active.wait(timeout=5)
            if installed is not None:run_installer(['/x',installed],logs/'fixture-cleanup.log')
            if integration_sentinels is not None:clean_removal_sentinels(integration_sentinels)
            if collision_shortcut is not None and collision_shortcut.is_file() and collision_shortcut.read_bytes()==collision_bytes:collision_shortcut.unlink()
    (m.ROOT/'build/native/msi-lifecycle.json').write_text(json.dumps(proof,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    return result
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('installer',type=Path,nargs='?');p.add_argument('--receipt',type=Path);p.add_argument('--cancel-worker',type=Path);p.add_argument('--installer-worker',type=Path);p.add_argument('--ordinary-worker',type=Path);p.add_argument('--context-dir',type=Path);p.add_argument('--output',type=Path);a=p.parse_args();m.disposable_runner()
    if a.installer_worker:native_installer_worker(a.installer_worker)
    elif a.cancel_worker:cancellation_worker(a.cancel_worker,a.context_dir,a.output)
    elif a.ordinary_worker:
        try:ordinary_worker(a.ordinary_worker,a.context_dir,a.output)
        except BaseException as error:
            a.output.with_suffix('.failure.json').write_text(json.dumps({'kind':'ordinary-context-worker-failure','pid':os.getpid(),'error_type':type(error).__name__,'error':str(error)[-3000:]},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
            raise
    elif a.installer and a.receipt:r=json.loads(a.receipt.read_text(encoding='utf-8'));print(json.dumps(qualify(a.installer,r)))
    else:p.error('installer and --receipt are required')
