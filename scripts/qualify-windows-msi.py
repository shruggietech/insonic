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

def service_diagnostic(configuration=False):
    """Read-only Installer service status on disposable CI. Never stop services."""
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    dll=C.WinDLL('advapi32',use_last_error=True)
    class Status(C.Structure):_fields_=[('service_type',W.DWORD),('state',W.DWORD),('controls',W.DWORD),('win32_exit',W.DWORD),('service_exit',W.DWORD),('checkpoint',W.DWORD),('wait_hint',W.DWORD),('pid',W.DWORD),('flags',W.DWORD)]
    dll.OpenSCManagerW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.DWORD];dll.OpenSCManagerW.restype=W.HANDLE
    dll.OpenServiceW.argtypes=[W.HANDLE,W.LPCWSTR,W.DWORD];dll.OpenServiceW.restype=W.HANDLE
    dll.QueryServiceStatusEx.argtypes=[W.HANDLE,C.c_int,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];dll.QueryServiceStatusEx.restype=W.BOOL
    dll.GetSecurityInfo.argtypes=[W.HANDLE,C.c_int,W.DWORD,C.c_void_p,C.c_void_p,C.c_void_p,C.c_void_p,C.POINTER(C.c_void_p)];dll.GetSecurityInfo.restype=W.DWORD
    dll.CloseServiceHandle.argtypes=[W.HANDLE];dll.CloseServiceHandle.restype=W.BOOL
    manager=dll.OpenSCManagerW(None,None,1)
    if not manager:return {'query_error':C.get_last_error(),'api':'OpenSCManagerW','requested_access':1}
    service=None
    try:
        service=dll.OpenServiceW(manager,'msiserver',0x20005 if configuration else 0x20004)
        if not service:return {'query_error':C.get_last_error(),'api':'OpenServiceW','requested_access':0x20004}
        value=Status();needed=W.DWORD()
        if not dll.QueryServiceStatusEx(service,0,C.byref(value),C.sizeof(value),C.byref(needed)):return {'query_error':C.get_last_error()}
        descriptor=C.c_void_p();result=dll.GetSecurityInfo(service,2,0x7,None,None,None,None,C.byref(descriptor))
        kernel=C.WinDLL('kernel32',use_last_error=True);kernel.LocalFree.argtypes=[C.c_void_p];kernel.LocalFree.restype=C.c_void_p
        try:security={'query_error':result} if result else descriptor_observation(descriptor,dll,kernel,information=0x7)
        finally:
            if descriptor:kernel.LocalFree(descriptor)
        observed={'pid':value.pid,'state':value.state,'accepted_controls':value.controls,'checkpoint':value.checkpoint,'wait_hint_ms':value.wait_hint,'win32_exit':value.win32_exit,'actual_service_security':security}
        if configuration:
            class Config(C.Structure):_fields_=[('type',W.DWORD),('start',W.DWORD),('error',W.DWORD),('binary',W.LPWSTR),('group',W.LPWSTR),('tag',W.DWORD),('dependencies',W.LPWSTR),('account',W.LPWSTR),('display',W.LPWSTR)]
            dll.QueryServiceConfigW.argtypes=[W.HANDLE,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];dll.QueryServiceConfigW.restype=W.BOOL
            needed=W.DWORD();dll.QueryServiceConfigW(service,None,0,C.byref(needed))
            if not 0<needed.value<=65536:raise RuntimeError('Installer service configuration exceeds bound')
            data=C.create_string_buffer(needed.value)
            if not dll.QueryServiceConfigW(service,data,len(data),C.byref(needed)):raise C.WinError(C.get_last_error())
            config=C.cast(data,C.POINTER(Config)).contents;raw=os.path.expandvars(config.binary or '').strip();expected=Path(os.environ['SystemRoot'])/'System32/msiexec.exe'
            observed.update({'binary_command':raw,'canonical_binary':str(expected.resolve()),'binary_identity_valid':raw.casefold() in ((str(expected)+' /V').casefold(),('"'+str(expected)+'" /V').casefold()),'service_name':'msiserver','start_type':config.start})
        return observed
    finally:
        if service:dll.CloseServiceHandle(service)
        dll.CloseServiceHandle(manager)

def release_owned_installer_service(initial,install_receipt,remove_receipt,scheduler,sid,deadline):
    """Gracefully restore a proven fixture-activated service, never kill a PID."""
    m.disposable_runner()
    if not initial or initial.get('state')!=1 or initial.get('pid')!=0 or not initial.get('binary_identity_valid'):raise RuntimeError('Installer service was not initially stopped and owned by this disposable fixture')
    if scheduler.get('phase')!='completed' or scheduler.get('exit_code')!=0 or scheduler.get('deleted') is not True:raise RuntimeError('ordinary task did not fully complete before service release')
    witnesses=[]
    for path,action in ((install_receipt,'/i'),(remove_receipt,'/x')):
        receipt=json.loads(Path(path).read_text(encoding='utf-8'))
        if receipt.get('kind')!='native-installer-worker' or receipt.get('phase')!='completed' or receipt.get('exit_code')!=0 or receipt.get('action')!=action:raise RuntimeError('native transaction did not fully complete before service release')
        witnesses.append(receipt['service'])
    pid=witnesses[-1].get('pid')
    if not pid or any(witness.get('state')!=4 or witness.get('pid')!=pid for witness in witnesses):raise RuntimeError('Installer service PID changed across owned ordinary transactions')
    if owned_actor_processes(sid):raise RuntimeError('ordinary actor processes remain; no service release attempted')
    current=service_diagnostic(configuration=True)
    if current.get('state')!=4 or current.get('pid')!=pid or not current.get('accepted_controls',0)&1 or not current.get('binary_identity_valid') or current.get('canonical_binary')!=initial.get('canonical_binary'):raise RuntimeError('Installer service is unowned, changed, busy, or refuses graceful stop')
    import ctypes as C
    import ctypes.wintypes as W
    dll=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True)
    class Status(C.Structure):_fields_=[('type',W.DWORD),('state',W.DWORD),('controls',W.DWORD),('win32_exit',W.DWORD),('service_exit',W.DWORD),('checkpoint',W.DWORD),('wait_hint',W.DWORD),('pid',W.DWORD),('flags',W.DWORD)]
    dll.OpenSCManagerW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.DWORD];dll.OpenSCManagerW.restype=W.HANDLE;dll.OpenServiceW.argtypes=[W.HANDLE,W.LPCWSTR,W.DWORD];dll.OpenServiceW.restype=W.HANDLE
    dll.QueryServiceStatusEx.argtypes=[W.HANDLE,C.c_int,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];dll.QueryServiceStatusEx.restype=W.BOOL;dll.ControlService.argtypes=[W.HANDLE,W.DWORD,C.c_void_p];dll.ControlService.restype=W.BOOL;dll.CloseServiceHandle.argtypes=[W.HANDLE]
    kernel.OpenProcess.argtypes=[W.DWORD,W.BOOL,W.DWORD];kernel.OpenProcess.restype=W.HANDLE;kernel.QueryFullProcessImageNameW.argtypes=[W.HANDLE,W.DWORD,W.LPWSTR,C.POINTER(W.DWORD)];kernel.QueryFullProcessImageNameW.restype=W.BOOL;kernel.CloseHandle.argtypes=[W.HANDLE]
    manager=dll.OpenSCManagerW(None,None,1);service=None;process=None
    if not manager:raise C.WinError(C.get_last_error())
    try:
        service=dll.OpenServiceW(manager,'msiserver',0x24)
        if not service:raise C.WinError(C.get_last_error())
        value=Status();needed=W.DWORD()
        def status():
            if not dll.QueryServiceStatusEx(service,0,C.byref(value),C.sizeof(value),C.byref(needed)):raise C.WinError(C.get_last_error())
            return value.state
        if status()!=4 or value.pid!=pid or not value.controls&1:raise RuntimeError('Installer service changed before graceful release')
        process=kernel.OpenProcess(0x1000,False,pid)
        if not process:raise C.WinError(C.get_last_error())
        image=C.create_unicode_buffer(32768);size=W.DWORD(len(image))
        if not kernel.QueryFullProcessImageNameW(process,0,image,C.byref(size)) or Path(image.value).resolve()!=Path(initial['canonical_binary']).resolve():raise RuntimeError('SCM service PID is not the exact Windows Installer image')
        if time.monotonic()>=deadline or status()!=4 or value.pid!=pid or not value.controls&1:raise RuntimeError('Installer service cleanup deadline or ownership changed')
        if not dll.ControlService(service,1,C.byref(value)):raise C.WinError(C.get_last_error())
        while status()!=1:
            if value.state!=3 or time.monotonic()>=deadline:raise RuntimeError('owned Installer service did not stop within cleanup bound')
            time.sleep(.05)
        return {'method':'SCM SERVICE_CONTROL_STOP','initial_state':initial['state'],'activated_pid':pid,'final_state':value.state,'restored_initial_stopped_state':True,'process_terminated':False,'registry_hive_forced':False}
    finally:
        if process:kernel.CloseHandle(process)
        if service:dll.CloseServiceHandle(service)
        dll.CloseServiceHandle(manager)

def descriptor_observation(descriptor,security,kernel,information=0x17):
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
    if not security.ConvertSecurityDescriptorToStringSecurityDescriptorW(descriptor,1,0xf if information&0x10 else information,C.byref(value),None):return {'descriptor_sha256':m.digest(C.string_at(descriptor,size)),'conversion_error':C.get_last_error()}
    try:
        raw=C.wstring_at(value);sanitized=__import__('re').sub(r'S-1-(?:\d+-)*\d+',lambda match:'SID_SHA256_'+m.digest(match.group().encode()),raw)
        return {'descriptor_sha256':m.digest(C.string_at(descriptor,size)),'owner_group_dacl_label_sddl':sanitized,'security_information':information,'machine_state_modified':False}
    finally:kernel.LocalFree(value)

class NativeObjectDirectories:
    """Bounded native object-directory access, exclusively in disposable CI."""
    def __init__(self):
        m.disposable_runner()
        import ctypes as C
        import ctypes.wintypes as W
        self.C=C;self.W=W;self.nt=C.WinDLL('ntdll');self.kernel=C.WinDLL('kernel32',use_last_error=True);self.security=C.WinDLL('advapi32',use_last_error=True)
        class Unicode(C.Structure):_fields_=[('length',W.USHORT),('maximum',W.USHORT),('buffer',W.LPWSTR)]
        class Attributes(C.Structure):_fields_=[('length',W.ULONG),('root',W.HANDLE),('name',C.POINTER(Unicode)),('flags',W.ULONG),('security',C.c_void_p),('qos',C.c_void_p)]
        self.Unicode=Unicode;self.Attributes=Attributes
        self.nt.NtOpenDirectoryObject.argtypes=[C.POINTER(W.HANDLE),W.DWORD,C.POINTER(Attributes)];self.nt.NtOpenDirectoryObject.restype=C.c_long
        self.nt.NtQuerySecurityObject.argtypes=[W.HANDLE,W.ULONG,C.c_void_p,W.ULONG,C.POINTER(W.ULONG)];self.nt.NtQuerySecurityObject.restype=C.c_long
        self.nt.NtSetSecurityObject.argtypes=[W.HANDLE,W.ULONG,C.c_void_p];self.nt.NtSetSecurityObject.restype=C.c_long
        self.nt.RtlNtStatusToDosError.argtypes=[W.ULONG];self.nt.RtlNtStatusToDosError.restype=W.ULONG
        self.kernel.CloseHandle.argtypes=[W.HANDLE];self.kernel.CloseHandle.restype=W.BOOL;self.kernel.LocalFree.argtypes=[C.c_void_p];self.kernel.LocalFree.restype=C.c_void_p
        self.security.ConvertStringSecurityDescriptorToSecurityDescriptorW.argtypes=[W.LPCWSTR,W.DWORD,C.POINTER(C.c_void_p),C.c_void_p];self.security.ConvertStringSecurityDescriptorToSecurityDescriptorW.restype=W.BOOL
        self.security.ConvertSecurityDescriptorToStringSecurityDescriptorW.argtypes=[C.c_void_p,W.DWORD,W.DWORD,C.POINTER(C.c_void_p),C.c_void_p];self.security.ConvertSecurityDescriptorToStringSecurityDescriptorW.restype=W.BOOL
    def checked(self,name,status):
        if status<0:
            error=self.C.WinError(self.nt.RtlNtStatusToDosError(status&0xffffffff));error.api=name;error.ntstatus=status&0xffffffff;raise error
    def open(self,path,access):
        C=self.C;W=self.W;length=len(path.encode('utf-16-le'));name=self.Unicode(length,length+2,path);attributes=self.Attributes(C.sizeof(self.Attributes),None,C.pointer(name),0x40,None,None);handle=W.HANDLE()
        status=self.nt.NtOpenDirectoryObject(C.byref(handle),access,C.byref(attributes));return handle,status
    def query(self,handle,information):
        C=self.C;size=self.W.ULONG();status=self.nt.NtQuerySecurityObject(handle,information,None,0,C.byref(size))
        if status<0 and status&0xffffffff not in (0xc0000023,0x80000005):self.checked('NtQuerySecurityObject-size',status)
        if not 0<size.value<=65536:raise RuntimeError('native object-directory descriptor exceeds bound')
        data=C.create_string_buffer(size.value);self.checked('NtQuerySecurityObject',self.nt.NtQuerySecurityObject(handle,information,data,len(data),C.byref(size)));return data
    def dacl(self,handle):
        C=self.C;data=self.query(handle,4);text=C.c_void_p()
        if not self.security.ConvertSecurityDescriptorToStringSecurityDescriptorW(data,1,4,C.byref(text),None):raise C.WinError(C.get_last_error())
        try:return data,C.wstring_at(text)
        finally:self.kernel.LocalFree(text)
    def describe(self,handle):return descriptor_observation(self.query(handle,0x17),self.security,self.kernel)
    @__import__('contextlib').contextmanager
    def convert(self,text):
        C=self.C;value=C.c_void_p()
        if not self.security.ConvertStringSecurityDescriptorToSecurityDescriptorW(text,1,C.byref(value),None):raise C.WinError(C.get_last_error())
        try:yield value
        finally:self.kernel.LocalFree(value)
    def set_dacl(self,handle,value):self.checked('NtSetSecurityObject-DACL-only',self.nt.NtSetSecurityObject(handle,4,value))
    def close(self,handle):
        if not self.kernel.CloseHandle(handle):raise self.C.WinError(self.C.get_last_error())

def object_namespace_diagnostic():
    """Read directory access independently of Win32 Local/Global alias setup."""
    m.disposable_runner();session=token_info()['session_id'];api=NativeObjectDirectories();rows=[]
    for path in (r'\Sessions','\\Sessions\\'+str(session),'\\Sessions\\'+str(session)+r'\BaseNamedObjects',r'\BaseNamedObjects'):
        for access in (1,2,4,8,15,0x20000):
            handle,status=api.open(path,access);row={'path':path,'requested_access':access,'ntstatus':status&0xffffffff,'opened':status>=0,'error':0 if status>=0 else int(api.nt.RtlNtStatusToDosError(status&0xffffffff))};rows.append(row)
            if status>=0:
                try:
                    if access&0x20000:row['security']=api.describe(handle)
                finally:api.close(handle)
    return {'kind':'read-only-native-object-directory-access','objects':rows,'machine_state_modified':False}

@__import__('contextlib').contextmanager
def interactive_context_namespace(stage,user_sid,caller_sid,desktop,probe):
    """Temporarily grant only the owned actor non-inheriting session-directory access."""
    m.disposable_runner();stage=Path(stage);base=Path(os.environ['ProgramData']).resolve()
    if stage.is_symlink() or stage.resolve()!=base/stage.name or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name):raise ValueError('namespace fixture requires the exact owned account stage')
    for sid in (user_sid,caller_sid):private_desktop_sddl(sid)
    marker=json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'))
    if not __import__('re').fullmatch('inso_ci_[0-9a-f]{12}',marker.get('account','')) or marker.get('sid_sha256')!=m.digest(user_sid.encode()):raise ValueError('namespace fixture account ownership differs')
    actor=token_info();session=actor['session_id'];target='\\Sessions\\'+str(session)+r'\BaseNamedObjects'
    if session<=0 or actor['user_sid_sha256']!=m.digest(caller_sid.encode()) or desktop['session_id']!=session or desktop['caller_sid_sha256']!=actor['user_sid_sha256'] or desktop['target_user_sid_sha256']!=marker['sid_sha256'] or probe['token']['session_id']!=session or probe['token']['user_sid_sha256']!=marker['sid_sha256']:raise RuntimeError('namespace fixture caller, ordinary actor or exact session differs')
    rows=[row for row in probe['directories']['objects'] if row['path']==target and row['requested_access']==15]
    if len(rows)!=1:raise RuntimeError('exact ordinary session-directory preflight is missing')
    evidence={'kind':'owned-CI-session-object-directory-fixture','session_id':session,'path':target,'rights_mask':15,'non_inheriting':True,'target_user_sid_sha256':marker['sid_sha256'],'global_namespace_modified':False,'child_object_security_modified':False,'security_restored':False,'modified':False}
    if rows[0]['opened']:
        evidence['security_restored']=True;yield evidence;return
    if rows[0]['error']!=5:raise RuntimeError('ordinary session directory failed for a reason other than access denied')
    api=NativeObjectDirectories();handle,status=api.open(target,0x6000f);api.checked('NtOpenDirectoryObject-owned-session',status);before=None;granted=None
    try:
        before,before_text=api.dacl(handle);original=api.describe(handle)
        if not original.get('descriptor_sha256') or not before_text.startswith('D:') or 'NO_ACCESS_CONTROL' in before_text:raise RuntimeError('exact session directory security is unavailable or has a null DACL')
        evidence['original_security']=original
        grant_text=before_text+'(A;;0xf;;;'+user_sid+')'
        with api.convert(grant_text) as descriptor:api.set_dacl(handle,descriptor)
        granted=grant_text
        _,granted=api.dacl(handle);evidence['modified']=True;evidence['granted_dacl_sha256']=m.digest(granted.encode())
        yield evidence
    finally:
        try:
            if granted is not None:
                _,current=api.dacl(handle)
                if current!=granted:raise RuntimeError('CI session-directory DACL changed outside the exact owned grant; restoration refused')
                api.set_dacl(handle,before);_,restored=api.dacl(handle);evidence['restored_security']=api.describe(handle);evidence['security_restored']=restored==before_text and evidence['restored_security']==original
                if not evidence['security_restored']:raise RuntimeError('CI session-directory owner/group/DACL/label restoration differs')
        finally:api.close(handle)

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
    for name in ('_MSIExecute','Global\\_MSIExecute','Local\\_MSIExecute','Local\\insonic-absent-'+uuid.uuid4().hex,'Global\\insonic-absent-'+uuid.uuid4().hex):
        for access in (0x100000,0x20000,0x1f0001):
            handle=kernel.OpenMutexW(access,False,name);row={'name':name,'requested_access':access,'opened':bool(handle),'error':0 if handle else C.get_last_error()};rows.append(row)
            if not handle:continue
            descriptor=C.c_void_p()
            try:
                if access&0x20000:
                    result=security.GetSecurityInfo(handle,6,7,None,None,None,None,C.byref(descriptor));row['descriptor_query_error']=result
                    if result==0:row['security']=descriptor_observation(descriptor,security,kernel,7)
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
 "fmt"
 "os"
 "os/exec"
 "syscall"
 "path/filepath"
 "runtime"
 "unsafe"
 "golang.org/x/sys/windows"
)
func main() {
 exe,e:=os.Executable();if e!=nil {os.Exit(1)}
 directory:=filepath.Dir(exe)
 diagnostic:=func(stage string,err error) {
  observed:=map[string]any{"stage":stage,"pid":os.Getpid(),"mode":filepath.Base(exe)}
  if err!=nil {observed["error"]=err.Error()}
  data,_:=json.Marshal(observed)
  if os.WriteFile(filepath.Join(directory,filepath.Base(exe)+".diagnostic.json"),append(data,'\n'),0600)!=nil {
   _=os.WriteFile(filepath.Join(os.TempDir(),fmt.Sprintf("owned-token-recorder-%d.diagnostic.json",os.Getpid())),append(data,'\n'),0600)
  }
 }
 fatal:=func(stage string,err error){diagnostic(stage,err);os.Exit(1)}
 diagnostic("process-started",nil)
 if b,e:=os.ReadFile(filepath.Join(directory,"owned-ci-fixture.json"));e!=nil||string(b)!="insonic disposable launcher context fixture\n" {fatal("owned-marker-read",e)}
 var token windows.Token
 if e:=windows.OpenProcessToken(windows.CurrentProcess(),windows.TOKEN_QUERY,&token);e!=nil {fatal("OpenProcessToken",e)}
 defer token.Close()
 var elevated,size uint32
 if e:=windows.GetTokenInformation(token,windows.TokenElevation,(*byte)(unsafe.Pointer(&elevated)),uint32(unsafe.Sizeof(elevated)),&size);e!=nil {fatal("TokenElevation",e)}
 user,e:=token.GetTokenUser();if e!=nil {fatal("TokenUser",e)}
 admin,e:=windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid);if e!=nil {fatal("CreateWellKnownSid",e)}
 member,e:=windows.Token(0).IsMember(admin);if e!=nil {fatal("CheckTokenMembership",e)}
 var required uint32
 e=windows.GetTokenInformation(token,windows.TokenIntegrityLevel,nil,0,&required)
 if e!=windows.ERROR_INSUFFICIENT_BUFFER {fatal("TokenIntegrityLevelSize",e)}
 if required<uint32(unsafe.Sizeof(windows.SIDAndAttributes{}))||required>65536 {fatal("TokenIntegrityLevelSize",fmt.Errorf("unexpected size %d",required))}
 // Native token structures require alignment. A byte stack array has alignment1.
 // Dynamic uint64 storage guarantees alignment and owns the embedded SID bytes.
 integrity:=make([]uint64,(uint64(required)+7)/8)
 if e:=windows.GetTokenInformation(token,windows.TokenIntegrityLevel,(*byte)(unsafe.Pointer(&integrity[0])),uint32(len(integrity)*8),&size);e!=nil {fatal("TokenIntegrityLevel",e)}
 base:=uintptr(unsafe.Pointer(&integrity[0]));label:=(*windows.SIDAndAttributes)(unsafe.Pointer(&integrity[0]));sidAddress:=uintptr(unsafe.Pointer(label.Sid))
 if sidAddress<base||sidAddress+8>base+uintptr(size) {fatal("IntegritySidBounds",fmt.Errorf("native SID outside returned token buffer"))}
 count:=label.Sid.SubAuthorityCount();if count==0||count>15||sidAddress+8+uintptr(count)*4>base+uintptr(size) {fatal("IntegritySidCount",fmt.Errorf("unexpected count %d",count))}
 integrityRID:=label.Sid.SubAuthority(uint32(count-1));integritySID:=label.Sid.String();runtime.KeepAlive(integrity)
 var session uint32
 if e:=windows.GetTokenInformation(token,windows.TokenSessionId,(*byte)(unsafe.Pointer(&session)),4,&size);e!=nil {fatal("TokenSessionId",e)}
 var statistics [7]uint64
 if e:=windows.GetTokenInformation(token,windows.TokenStatistics,(*byte)(unsafe.Pointer(&statistics[0])),uint32(unsafe.Sizeof(statistics)),&size);e!=nil {fatal("TokenStatistics",e)}
 authentication:=map[string]uint32{"low":uint32(statistics[1]),"high":uint32(statistics[1]>>32)}
 sum:=sha256.Sum256([]byte(user.User.Sid.String()))
 observed:=map[string]any{"elevated":elevated!=0,"administrator_enabled":member,"session_id":session,"authentication_luid":authentication,"user_sid_sha256":hex.EncodeToString(sum[:]),"integrity_rid":integrityRID,"integrity_sid":integritySID,"mode":filepath.Base(exe),"pid":os.Getpid(),"fixture":"owned native Go token reporter"}
 data,e:=json.Marshal(observed);if e!=nil {fatal("MarshalToken",e)}
 if e:=os.WriteFile(filepath.Join(directory,filepath.Base(exe)+".token.json"),append(data,'\n'),0600);e!=nil {fatal("WriteTokenReceipt",e)}
 diagnostic("token-recorded",nil)
 if filepath.Base(exe)=="documentation-handler.exe" { _=os.WriteFile(filepath.Join(directory,"documentation-handler.exe.last-token.json"),append(data,'\n'),0600) }
 if filepath.Base(exe)=="documentation-handler.exe" {
  var config struct { Node string; Script string; Environment map[string]string }
  configFile:=filepath.Join(directory,"documentation-handler.json")
  data,e:=os.ReadFile(configFile);if e!=nil {fatal("ReadRenderer",e)}
  if e=json.Unmarshal(data,&config);e!=nil {fatal("ParseRenderer",e)}
  if len(os.Args)!=2 {fatal("AssociationArgument",fmt.Errorf("expected exact HTML filename"))}
  var owned struct { SIDHash string `json:"sid_sha256"` }
  marker,e:=os.ReadFile(filepath.Join(directory,"owned-ci-account.json"));if e!=nil {fatal("ReadOwnedAccount",e)}
  if e=json.Unmarshal(marker,&owned);e!=nil||owned.SIDHash!=hex.EncodeToString(sum[:]) {fatal("RendererOwnedAccount",fmt.Errorf("renderer account differs from the staged native SID"))}
  expected:=map[string]string{"GITHUB_ACTIONS":"true","RUNNER_ENVIRONMENT":"github-hosted","INSONIC_DISPOSABLE_INSTALLER_TEST":"1"}
  if len(config.Environment)!=len(expected) {fatal("RendererCIACK",fmt.Errorf("missing exact disposable runner acknowledgement"))}
  environment:=os.Environ()
  for key,value:=range expected {if config.Environment[key]!=value {fatal("RendererCIACK",fmt.Errorf("disposable runner acknowledgement differs"))};environment=append(environment,key+"="+value)}
  diagnostic("offline-renderer-starting",nil)
  command:=exec.Command(config.Node,config.Script,configFile,os.Args[1]);command.Env=environment;command.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000}
  output,e:=command.CombinedOutput();if e!=nil {fatal("OfflineRenderer",fmt.Errorf("%v: %.2000s",e,output))}
  diagnostic("offline-renderer-completed",nil)
 }
}
'''

def prepare_launcher_fixture(installed,directory):
    m.disposable_runner();directory.mkdir(parents=True,exist_ok=False)
    shutil.copyfile(installed/'insonic-launcher.exe',directory/'insonic-launcher.exe')
    for name in ('LICENSE','NOTICE','insonic-companions.json'):
        shutil.copyfile(installed/name,directory/name)
    shutil.copytree(installed/'help',directory/'help')
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

def native_recorder_probe(directory):
    """Observe the exact GUI fixture directly; never substitute elected proof."""
    m.disposable_runner();source=Path(directory)/'insonic-desktop.exe';target=Path(directory)/'owned-native-recorder-probe.exe'
    shutil.copyfile(source,target)
    if m.sha(source)!=m.sha(target):raise ValueError('native diagnostic reporter bytes differ')
    with subprocess.Popen([str(target)],cwd=directory,stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW) as process:
        try:code=process.wait(timeout=15)
        except subprocess.TimeoutExpired:process.terminate();process.wait(timeout=5);raise RuntimeError('exact owned native recorder probe exceeded bound')
        observation={'pid':process.pid,'exit_code':code,'same_gui_bytes_sha256':m.sha(source),'elected_launch_proof':False,'native_child_image':target.name}
    (Path(directory)/'native-recorder-probe.launch-attempt.json').write_text(json.dumps(observation,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    return observation

def elected_launcher_context(directory,allow_refusal=False):
    m.disposable_runner();caller=token_info();modes={};native_recorder_probe(directory)
    linked=linked_token_evidence() if caller['elevated'] else {'available':False,'lookup_error':None}
    refusal_permitted=allow_refusal and caller['elevated'] and not linked['available']
    for mode,exe in [('desktop','insonic-desktop.exe'),('cli','insonic.exe')]:
        output=directory/(exe+'.token.json')
        if output.exists():output.unlink()
        attempted=subprocess.run([str(directory/'insonic-launcher.exe'),'--'+mode],cwd=directory,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=15)
        (directory/(mode+'.launch-attempt.json')).write_text(json.dumps({'mode':mode,'launcher_exit':attempted.returncode,'caller_token':caller,'stderr':attempted.stderr.decode(errors='replace')[-2000:]},sort_keys=True)+'\n',encoding='utf-8',newline='\n')
        if attempted.returncode:
            message=attempted.stderr.decode(errors='replace')
            if not refusal_permitted or attempted.returncode!=1 or 'cannot launch at normal user privilege' not in message or output.exists():raise RuntimeError('elected '+mode+' launcher failed: '+message[-2000:])
            modes[mode]={'outcome':'refused-normal-launch-action','child_record_absent':True,'elevated_fallback':False};continue
        deadline=time.monotonic()+10;observed=None
        while time.monotonic()<deadline:
            try:observed=json.loads(output.read_text(encoding='utf-8'));break
            except (FileNotFoundError,json.JSONDecodeError):time.sleep(.05)
        if observed is None:
            diagnostic=directory/(exe+'.diagnostic.json')
            details=diagnostic.read_text(encoding='utf-8')[-2000:] if diagnostic.is_file() else 'native recorder did not start or could not write its diagnostic'
            raise RuntimeError('elected '+mode+' child did not record its actual token: '+details)
        if observed['elevated'] or observed['administrator_enabled'] or observed['integrity_rid']>8192 or observed['user_sid_sha256']!=caller['user_sid_sha256']:raise RuntimeError('elected '+mode+' launch changed user or retained elevation/high integrity')
        modes[mode]={'outcome':'observed-same-user-non-elevated-child','observed_token':observed}
    return {'caller_token':caller,'linked_token':linked,'modes':modes,'actual_elected_launch':True,'describe_is_proof':False}

def retain_launcher_diagnostics(directory,output):
    """Retain bounded native actor receipts, never copy executable fixtures."""
    for item in Path(directory).iterdir():
        if item.is_file() and (item.name.endswith('.token.json') or item.name.endswith('.diagnostic.json') or item.name.endswith('.launch-attempt.json')):
            if item.is_symlink() or item.stat().st_size>16384:raise ValueError('native launcher diagnostic is redirected or oversized')
            shutil.copyfile(item,Path(output).with_name(Path(output).stem+'.launcher-'+item.name))
    temporary=Path(os.environ['TEMP'])
    for item in temporary.glob('owned-token-recorder-*.diagnostic.json'):
        if item.is_symlink() or not item.is_file() or item.stat().st_size>16384:raise ValueError('native recorder fallback diagnostic is redirected or oversized')
        shutil.copyfile(item,Path(output).with_name(Path(output).stem+'.launcher-'+item.name))

def token_info(existing=None):
    """Observe the current or explicitly borrowed token without changing it."""
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
    owns_token=existing is None;token=W.HANDLE() if owns_token else existing;membership=W.HANDLE()
    if owns_token:
        if not security.OpenProcessToken(kernel.GetCurrentProcess(),8,C.byref(token)):raise C.WinError(C.get_last_error())
    else:
        security.DuplicateToken.argtypes=[W.HANDLE,C.c_int,C.POINTER(W.HANDLE)];security.DuplicateToken.restype=W.BOOL
        if not security.DuplicateToken(token,2,C.byref(membership)):raise C.WinError(C.get_last_error())
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
        if not security.CheckTokenMembership(membership if not owns_token else None,admin,C.byref(member)):raise C.WinError(C.get_last_error())
        logon_groups={}
        for key,kind in [('interactive_logon',11),('batch_logon',10)]:
            group=C.create_string_buffer(68);group_size=W.DWORD(68);present=W.BOOL()
            if not security.CreateWellKnownSid(kind,None,group,C.byref(group_size)) or not security.CheckTokenMembership(membership if not owns_token else None,group,C.byref(present)):raise C.WinError(C.get_last_error())
            logon_groups[key]=bool(present.value)
        privileges=info(3);count=struct.unpack_from('<I',privileges.raw)[0]
        if count>100:raise RuntimeError('invalid token privilege count')
        enabled=sum(bool(struct.unpack_from('<I',privileges.raw,4+i*12+8)[0]&2) for i in range(count))
        authentication_low,authentication_high=struct.unpack_from('<II',info(10).raw,8)
        return {'administrator_enabled':bool(member.value),'elevated':elevated,'elevation_type':elevation_type,'enabled_privileges':enabled,'user_sid_sha256':user_hash,'integrity_sid':integrity_sid,'integrity_rid':integrity_rid,'session_id':int(C.cast(info(12),C.POINTER(W.DWORD))[0]),'authentication_luid':{'low':authentication_low,'high':authentication_high},**logon_groups}
    finally:
        if membership:kernel.CloseHandle(membership)
        if owns_token:kernel.CloseHandle(token)

def ordinary_namespace_probe(stage):
    m.disposable_runner();stage=Path(stage)
    proof={'state':'passed','token':token_info(),'directories':object_namespace_diagnostic(),'mutex':installer_mutex_diagnostic()}
    (stage/'native/context.namespace-probe.json').write_text(json.dumps(proof,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')

DIAGNOSTIC_MODES=('completion','rollback-reused','rollback-isolated','rollback-primed','rollback-late','rollback-msiexec','rollback-recovery','rollback-exit-full','rollback-exit-quiet','rollback-container','rollback-container-independent','rollback-standalone','rollback-standalone-full','rollback-setup','rollback-setup-full')

def diagnostic_mode(stage):
    m.disposable_runner()
    raw=(Path(stage)/'installer-diagnostic-mode.json').read_bytes()
    if len(raw)>1024:raise RuntimeError('owned diagnostic mode exceeds its bound')
    mode=json.loads(raw)
    if set(mode)!={'feature_qualification','mode'} or mode['feature_qualification'] is not False or mode['mode'] not in DIAGNOSTIC_MODES:raise RuntimeError('invalid owned diagnostic mode')
    return mode['mode']


def ordinary_ui_worker(stage):
    """Run real full UI in the normal logon created in the CI caller session."""
    m.disposable_runner();stage=Path(stage);output=stage/'native/context.ui.json'
    observed=token_info();desktop=json.loads((stage/'ui-actor-desktop.json').read_text(encoding='utf-8'))
    before={'phase':'before-first-install','token':observed,'directories':object_namespace_diagnostic(),'mutex':installer_mutex_diagnostic()}
    output.with_suffix('.boundary.json').write_text(json.dumps(before,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    if observed['administrator_enabled'] or observed['elevated'] or observed['enabled_privileges']>1 or observed['integrity_rid']!=8192 or not observed['interactive_logon'] or observed['session_id']!=desktop['session_id'] or observed['user_sid_sha256']!=desktop['target_user_sid_sha256']:raise RuntimeError('full-UI worker did not retain its exact ordinary interactive logon')
    target=r'\Sessions'+'\\'+str(observed['session_id'])+r'\BaseNamedObjects'
    directory=[row for row in before['directories']['objects'] if row['path']==target and row['requested_access']==15]
    absent=[row for row in before['mutex']['objects'] if 'insonic-absent-' in row['name']]
    if len(directory)!=1 or not directory[0]['opened'] or not absent or any(row['opened'] or row['error']!=2 for row in absent):raise RuntimeError('ordinary native object namespace still denies exact directory access or absent mutex lookup')
    if (stage/'setup-qualification.json').is_file():
        from qualify_installer_setup import qualify as qualify_setup
        proof=qualify_setup(stage,observed,run_installer)
    elif (stage/'completion-invocation-diagnostic.json').is_file():
        import runpy
        diagnostic=runpy.run_path(str(stage/'scripts/diagnose-installer-completion.py'),run_name='owned_invocation_diagnostic')
        mode=diagnostic_mode(stage)
        proof=diagnostic['observe_setup' if mode in ('rollback-setup','rollback-setup-full') else 'observe_invocation' if mode=='completion' else 'observe_rollback'](stage,observed,run_installer)
    else:
        from qualify_installer_ui import qualify as qualify_ui
        proof=qualify_ui(stage,stage/'fixture.msi',stage/'installed program é with spaces',stage/'native/context.json',observed,run_installer)
    output.write_text(json.dumps(proof,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')

def interactive_ordinary(stage,name,sid,password,profile,desktop,record,probe_only=False):
    """Use Secondary Logon from the existing interactive caller, without a second profile load."""
    m.disposable_runner();stage=Path(stage)
    if json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'))!={'account':name,'sid_sha256':m.digest(sid.encode())}:raise RuntimeError('interactive actor ownership differs')
    if desktop['path']!=r'WinSta0\Default' or desktop['session_id']!=token_info()['session_id'] or desktop['target_user_sid_sha256']!=m.digest(sid.encode()):raise RuntimeError('interactive actor desktop/session differs')
    import ctypes as C
    import ctypes.wintypes as W
    security=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True)
    class Startup(C.Structure):_fields_=[('cb',W.DWORD),('reserved',W.LPWSTR),('desktop',W.LPWSTR),('title',W.LPWSTR),('x',W.DWORD),('y',W.DWORD),('width',W.DWORD),('height',W.DWORD),('charsx',W.DWORD),('charsy',W.DWORD),('fill',W.DWORD),('flags',W.DWORD),('show',W.WORD),('reserved2',W.WORD),('reserved_pointer',C.c_void_p),('stdin',W.HANDLE),('stdout',W.HANDLE),('stderr',W.HANDLE)]
    class Process(C.Structure):_fields_=[('process',W.HANDLE),('thread',W.HANDLE),('pid',W.DWORD),('tid',W.DWORD)]
    security.CreateProcessWithLogonW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.LPCWSTR,W.DWORD,W.LPCWSTR,W.LPWSTR,W.DWORD,C.c_void_p,W.LPCWSTR,C.POINTER(Startup),C.POINTER(Process)];security.CreateProcessWithLogonW.restype=W.BOOL
    kernel.WaitForSingleObject.argtypes=[W.HANDLE,W.DWORD];kernel.WaitForSingleObject.restype=W.DWORD;kernel.GetExitCodeProcess.argtypes=[W.HANDLE,C.POINTER(W.DWORD)];kernel.GetExitCodeProcess.restype=W.BOOL;kernel.CloseHandle.argtypes=[W.HANDLE];kernel.TerminateProcess.argtypes=[W.HANDLE,W.UINT]
    purpose='namespace-probe' if probe_only else 'ui';executable=str(Path(sys.executable).with_name('pythonw.exe'));bootstrap=stage/('scripts/interactive-'+purpose+'-bootstrap.py');script=stage/'scripts/qualify-windows-msi.py'
    bootstrap.write_text('import os,runpy,sys\nsys.stdout=sys.stderr=open('+repr(str(stage/('native/context.'+purpose+'-stream.log')))+',"w",encoding="utf-8",buffering=1)\nsys.argv='+repr([str(script),'--ordinary-namespace-probe' if probe_only else '--ordinary-ui-worker',str(stage)])+'\nsys.path.insert(0,os.path.dirname(sys.argv[0]))\nrunpy.run_path(sys.argv[0],run_name="__main__")\n',encoding='utf-8',newline='\n')
    command=C.create_unicode_buffer(subprocess.list2cmdline([executable,str(bootstrap)]))
    if len(command.value)>1023:raise RuntimeError('interactive Secondary Logon command exceeds its documented bound')
    environment=C.create_unicode_buffer(ordinary_environment(profile,name,stage/'temp'));startup=Startup();startup.cb=C.sizeof(startup);startup.flags=1;startup.show=0;startup.desktop=desktop['path'];process=Process()
    try:
        # flags0 authenticates a fresh local ordinary logon, retaining the
        # caller session. Its profile is already held by the parent's one lease.
        record('CreateProcessWithLogonW-before',logon_flags=0,profile_loaded_by_child=False,session_token_reassignment=False)
        if not security.CreateProcessWithLogonW(name,'.',password,0,executable,command,0x08000400,environment,str(stage),C.byref(startup),C.byref(process)):
            error=C.WinError(C.get_last_error());error.api='CreateProcessWithLogonW';raise error
        record('interactive-ordinary-created',pid=process.pid,logon_flags=0,profile_loaded_by_child=False,session_token_reassignment=False)
        if kernel.WaitForSingleObject(process.process,30000 if probe_only else 230000)!=0:raise RuntimeError('exact owned full-UI actor exceeded its bound')
        code=W.DWORD()
        if not kernel.GetExitCodeProcess(process.process,C.byref(code)):
            error=C.WinError(C.get_last_error());error.api='GetExitCodeProcess-interactive';raise error
        record('interactive-ordinary-exited',pid=process.pid,exit_code=code.value)
        if code.value:raise RuntimeError('ordinary interactive worker failed '+str(code.value))
        proof=json.loads((stage/('native/context.namespace-probe.json' if probe_only else 'native/context.ui.json')).read_text(encoding='utf-8'));child_observed=proof['token']
        if proof.get('state')!='passed' or child_observed['user_sid_sha256']!=desktop['target_user_sid_sha256'] or child_observed['session_id']!=desktop['session_id'] or child_observed['elevated'] or child_observed['administrator_enabled'] or child_observed['integrity_rid']!=8192 or child_observed['enabled_privileges']>1 or not child_observed['interactive_logon']:raise RuntimeError('actual interactive worker token differs from the exact ordinary account/session')
        record('interactive-ordinary-token-observed',pid=process.pid,token=child_observed,observation='GetTokenInformation in the exact completed ordinary worker',purpose=purpose)
        return proof
    finally:
        if process.process and kernel.WaitForSingleObject(process.process,0)!=0:kernel.TerminateProcess(process.process,1);kernel.WaitForSingleObject(process.process,5000)
        for handle in (process.thread,process.process):
            if handle:kernel.CloseHandle(handle)

def ordinary_worker(installer,directory,output):
    m.disposable_runner()
    progress=output.with_suffix('.progress.json');launcher_fixture=directory.parent/'launcher-context-fixture'
    def record(phase,**extra):progress.write_text(json.dumps({'pid':os.getpid(),'phase':phase,**extra},sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    record('python-started');observed=token_info();record('token-observed',token=observed,mutex_before_install=installer_mutex_diagnostic())
    if observed['administrator_enabled'] or observed['enabled_privileges']>1 or observed['elevated'] or observed['integrity_rid']!=8192:raise RuntimeError('ordinary worker retained administrator membership/privileges/elevation or did not run at medium integrity')
    if os.environ.get('INSONIC_CI_ORDINARY_SCHEDULER')=='1':
        desktop=json.loads((directory.parent/'ui-actor-desktop.json').read_text(encoding='utf-8'))
        if not observed['interactive_logon'] or observed['session_id']!=0 or observed['user_sid_sha256']!=desktop['target_user_sid_sha256']:raise RuntimeError('ordinary quiet worker did not retain its exact session0 account and ordinary logon')
    product=dict(m.table_rows(m.CompoundFile(installer).streams,'Property'))['ProductCode']
    ui_evidence=json.loads((directory.parent/'native/context.ui.json').read_text(encoding='utf-8'))
    if ui_evidence['state']!='passed' or ui_evidence['token']['user_sid_sha256']!=observed['user_sid_sha256']:raise RuntimeError('ordinary full-UI evidence differs from the exact quiet account')
    installed=False
    try:
        run_installer(['/i',installer,'INSTALLDIR='+str(directory),'LICENSE_ACCEPTED=1','ADDLOCAL=Core,CLI'],output.with_suffix('.install.log'));installed=True
        record('installed')
        marker=directory/m.OWNERSHIP
        if not marker.is_file() or marker.read_bytes()!=m.OWNERSHIP_BYTES:raise RuntimeError('ordinary per-user install missing ownership marker')
        model=json.loads((directory.parent/'context-model.json').read_text(encoding='utf-8'));installed_hashes(directory,model,False)
        version=dict(m.table_rows(m.CompoundFile(installer).streams,'Property'))['ProductVersion']
        if child([directory/'insonic.exe','version'],dict(os.environ),directory).decode().strip()!=version:raise RuntimeError('ordinary-user installed CLI version differs')
        launch_evidence=elected_launcher_context(launcher_fixture)
        record('elected-launch-observed')
        run_installer(['/x',product],output.with_suffix('.remove.log'));installed=False
        if marker.exists():raise RuntimeError('ordinary per-user removal retained owned marker')
        evidence={'kind':'ordinary-disposable-ci-user','token':observed,'install_exit':0,'remove_exit':0,'payload_hashes_verified':True,'separate_user_account':True,'os_policy_mutations':{'worker_changed_global_policy':False,'product_changes_endpoint_policy':False,'owned_account_batch_logon_right':False,'ci_actor_broker':'SYSTEM service-account task'},'profile_loaded':True,'user_profile':os.environ['USERPROFILE'],'elected_launcher':launch_evidence,'interactive':ui_evidence}
        output.write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
        record('completed')
    finally:
        retain_launcher_diagnostics(launcher_fixture,output)
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
$inputObject=$null;$folder=$null;$registered=$null;$created=$false;$pipe=$null;$writer=$null
$result=@{phase='starting';deleted=$false}
try {
  $result.phase='console-encoding'
  [Console]::InputEncoding=[Text.UTF8Encoding]::new($false)
  [Console]::OutputEncoding=[Text.UTF8Encoding]::new($false)
  $result.phase='credential-input'
  $inputObject=[Console]::In.ReadToEnd() | ConvertFrom-Json
  $result.task_name=$inputObject.task_name
  $result.phase='scheduler-connect'
  $service=New-Object -ComObject Schedule.Service;$service.Connect();$folder=$service.GetFolder('\')
  if ($inputObject.task_name -notmatch '^insonic-ci-ordinary-[0-9a-f]{32}$') { throw 'invalid owned task name' }
  if ($inputObject.mode -eq 'cleanup') {
    try { $existing=$folder.GetTask($inputObject.task_name) } catch { if ($_.Exception.HResult -eq -2147024894) { $existing=$null } else { throw } }
    if ($null -ne $existing) {
      if ($existing.Definition.RegistrationInfo.Description -ne $inputObject.description) { throw 'task ownership mismatch' }
      $principal=$existing.Definition.Principal.UserId
      if ($principal -notmatch '^S-1-') { $principal=([Security.Principal.NTAccount]::new($principal)).Translate([Security.Principal.SecurityIdentifier]).Value }
      if ($principal -ne 'S-1-5-18') { throw 'task actor mismatch' }
      try { if ($existing.State -eq 2 -or $existing.State -eq 4) { $existing.Stop(0) } } catch { if ($_.Exception.HResult -ne -2147216629) { $result.stop_hresult=$_.Exception.HResult } }
      $folder.DeleteTask($inputObject.task_name,0)
    }
    $result.deleted=$true;$result.phase='cleanup-completed'
  } else {
    $result.phase='private-credential-pipe'
    $pipeSecurity=[IO.Pipes.PipeSecurity]::new();$pipeSecurity.SetAccessRuleProtection($true,$false)
    foreach ($subject in @($inputObject.sid,([Security.Principal.WindowsIdentity]::GetCurrent()).User.Value,'S-1-5-18')) {
      $identity=[Security.Principal.SecurityIdentifier]::new($subject)
      $rule=[IO.Pipes.PipeAccessRule]::new($identity,[IO.Pipes.PipeAccessRights]::FullControl,[Security.AccessControl.AccessControlType]::Allow);$pipeSecurity.AddAccessRule($rule)
    }
    $pipe=[IO.Pipes.NamedPipeServerStream]::new($inputObject.pipe_name,[IO.Pipes.PipeDirection]::Out,1,[IO.Pipes.PipeTransmissionMode]::Byte,[IO.Pipes.PipeOptions]::Asynchronous,4096,4096,$pipeSecurity)
    $connection=$pipe.BeginWaitForConnection($null,$null)
    $definition=$service.NewTask(0)
    $definition.RegistrationInfo.Description=$inputObject.description
    $definition.Principal.UserId='SYSTEM';$definition.Principal.LogonType=5;$definition.Principal.RunLevel=0
    $definition.Settings.Enabled=$true;$definition.Settings.Hidden=$true;$definition.Settings.ExecutionTimeLimit='PT240S'
    $definition.Settings.DisallowStartIfOnBatteries=$false;$definition.Settings.StopIfGoingOnBatteries=$false
    $action=$definition.Actions.Create(0);$action.Path=$inputObject.executable;$action.Arguments=$inputObject.arguments;$action.WorkingDirectory=$inputObject.directory
    $registered=$folder.RegisterTaskDefinition($inputObject.task_name,$definition,2,'SYSTEM',$null,5,$null)
    $created=$true;$result.phase='registered';$result.requested_run_level=0;$result.logon_type=5;$result.broker_role='SYSTEM';$result.target_task_profile_loaded=$false
    $running=$registered.Run($null);$result.engine_pid=$running.EnginePID;$deadline=[DateTime]::UtcNow.AddSeconds(245)
    $result.phase='private-pipe-connect'
    if (-not $connection.AsyncWaitHandle.WaitOne(15000)) { throw 'owned credential pipe connection timed out' }
    $pipe.EndWaitForConnection($connection)
    $writer=[IO.StreamWriter]::new($pipe,[Text.UTF8Encoding]::new($false));$writer.AutoFlush=$true
    $writer.WriteLine((@{account=$inputObject.account;sid=$inputObject.sid;password=$inputObject.password}|ConvertTo-Json -Compress));$inputObject.password=$null;$writer.Dispose();$writer=$null;$pipe=$null
    $result.phase='worker-running'
    do { Start-Sleep -Milliseconds 100;if ([DateTime]::UtcNow -gt $deadline) { throw 'owned ordinary task exceeded bound' } } while ($registered.State -eq 2 -or $registered.State -eq 4)
    $result.exit_code=$registered.LastTaskResult;$result.phase='completed'
  }
} catch { $result.failed_boundary=$result.phase;$result.phase='failed';$result.hresult=$_.Exception.HResult }
finally {
  if ($null -ne $inputObject) { $inputObject.password=$null }
  if ($null -ne $writer) { $writer.Dispose() } elseif ($null -ne $pipe) { $pipe.Dispose() }
  if ($created) {
    try { if ($registered.State -eq 2 -or $registered.State -eq 4) { $registered.Stop(0) } } catch { if ($_.Exception.HResult -ne -2147216629) { $result.stop_hresult=$_.Exception.HResult } }
    try { $folder.DeleteTask($inputObject.task_name,0);$result.deleted=$true } catch { $result.cleanup_hresult=$_.Exception.HResult }
  }
}
$result | ConvertTo-Json -Compress
'''

def scheduler_ordinary(stage,name,sid,password,args,record,profile):
    """Run one owned SYSTEM broker task; target credentials use the private pipe."""
    m.disposable_runner();stage=Path(stage)
    marker=json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'))
    if marker!={'account':name,'sid_sha256':m.digest(sid.encode())} or stage.is_symlink() or stage.resolve().parent!=Path(os.environ['ProgramData']).resolve() or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name):raise ValueError('ordinary scheduler target is not the exact owned account stage')
    import base64
    executable=Path(os.environ['SystemRoot'])/'System32/WindowsPowerShell/v1.0/powershell.exe'
    command=[str(executable),'-NoLogo','-NoProfile','-NonInteractive','-EncodedCommand',base64.b64encode(SCHEDULER_BRIDGE.encode('utf-16le')).decode('ascii')]
    task='insonic-ci-ordinary-'+uuid.uuid4().hex;description='insonic disposable ordinary actor '+m.digest(sid.encode())+' '+task;pipe='insonic-ci-credential-'+uuid.uuid4().hex
    environment=dict(item.split('=',1) for item in ordinary_environment(profile,name,stage/'temp').split('\0') if item);environment['INSONIC_CI_ORDINARY_SCHEDULER']='1'
    bootstrap=[str(args[0]),'--ordinary-broker',pipe,*list(map(str,args[2:]))]
    code='import os,runpy,sys;os.environ.clear();os.environ.update('+repr(environment)+');sys.argv='+repr(bootstrap)+';sys.path.insert(0,os.path.dirname(sys.argv[0]));runpy.run_path(sys.argv[0],run_name="__main__")'
    metadata={'task_name':task,'description':description,'sid':sid}
    request={**metadata,'mode':'run','account':os.environ['COMPUTERNAME']+'\\'+name,'password':password,'pipe_name':pipe,'executable':sys.executable,'arguments':subprocess.list2cmdline(['-c',code]),'directory':str(stage)}
    outcome=None
    def invoke(value):
        completed=subprocess.run(command,input=json.dumps(value).encode('utf-8'),stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=260)
        if completed.returncode:raise RuntimeError('hidden ordinary scheduler bridge failed '+str(completed.returncode))
        return json.loads(completed.stdout.decode('utf-8-sig'))
    record('TaskScheduler-ordinary-before-register',task_name=task,password_published=False,requested_run_level=0,logon_type=5,broker_role='SYSTEM')
    try:
        outcome=invoke(request);record('TaskScheduler-ordinary-result',scheduler=outcome)
        if outcome.get('phase')!='completed' or outcome.get('exit_code')!=0 or outcome.get('deleted') is not True:raise RuntimeError('ordinary scheduled context did not complete: '+json.dumps(outcome,sort_keys=True))
        return outcome
    finally:
        request['password']=None
        if outcome is None or outcome.get('deleted') is not True:
            cleanup=invoke({**metadata,'mode':'cleanup'});record('TaskScheduler-owned-cleanup',scheduler=cleanup)
            if cleanup.get('deleted') is not True:raise RuntimeError('owned ordinary task cleanup failed')
        if outcome is not None and outcome.get('engine_pid'):
            outcome['engine_cleanup']={'pid':outcome['engine_pid'],'owned_task_completed':outcome.get('phase')=='completed','SYSTEM_process_stop_requested':False};record('TaskScheduler-owned-engine-cleanup',engine=outcome['engine_cleanup'])

def retain_ordinary_diagnostics(native,output):
    """Keep each actor's exact evidence distinct before deleting its stage."""
    for item in Path(native).iterdir():
        if item.is_file() and item.name.startswith('context.'):
            if item.is_symlink():raise ValueError('ordinary diagnostic source is redirected')
            shutil.copyfile(item,Path(output).with_name(Path(output).stem+item.name[len('context'):]))

def stop_owned_task_engine(pid,sid,stage=None):
    """Release only the reported task engine if it belongs to this new CI actor."""
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    kernel=C.WinDLL('kernel32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True)
    kernel.OpenProcess.argtypes=[W.DWORD,W.BOOL,W.DWORD];kernel.OpenProcess.restype=W.HANDLE
    kernel.QueryFullProcessImageNameW.argtypes=[W.HANDLE,W.DWORD,W.LPWSTR,C.POINTER(W.DWORD)];kernel.QueryFullProcessImageNameW.restype=W.BOOL
    kernel.WaitForSingleObject.argtypes=[W.HANDLE,W.DWORD];kernel.WaitForSingleObject.restype=W.DWORD;kernel.TerminateProcess.argtypes=[W.HANDLE,W.UINT];kernel.TerminateProcess.restype=W.BOOL;kernel.CloseHandle.argtypes=[W.HANDLE]
    security.OpenProcessToken.argtypes=[W.HANDLE,W.DWORD,C.POINTER(W.HANDLE)];security.OpenProcessToken.restype=W.BOOL
    if not isinstance(pid,int) or pid<=0:raise ValueError('invalid reported task-engine PID')
    process=kernel.OpenProcess(0x101001,False,pid)
    if not process:
        if C.get_last_error()==87:return {'pid':pid,'already_exited':True,'terminated':False}
        raise C.WinError(C.get_last_error())
    token=W.HANDLE()
    try:
        if kernel.WaitForSingleObject(process,0)==0:return {'pid':pid,'already_exited':True,'terminated':False}
        if not security.OpenProcessToken(process,8,C.byref(token)):raise C.WinError(C.get_last_error())
        if account_sid(token)!=sid:raise RuntimeError('reported task engine belongs to another actor; no process stopped')
        image=C.create_unicode_buffer(32768);size=W.DWORD(len(image))
        if not kernel.QueryFullProcessImageNameW(process,0,image,C.byref(size)):raise C.WinError(C.get_last_error())
        path=Path(image.value);system=Path(os.environ['SystemRoot'])/'System32'
        windows_host=path.parent.resolve()==system.resolve() and path.name.casefold() in ('taskeng.exe','taskhost.exe','taskhostw.exe')
        owned_python=False
        if not windows_host and stage is not None and path.resolve()==Path(sys.executable).resolve():
            stage=Path(stage);marker=json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'));script=stage/'scripts/qualify-windows-msi.py'
            owned_python=not stage.is_symlink() and stage.resolve().parent==Path(os.environ['ProgramData']).resolve() and bool(__import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name)) and marker.get('sid_sha256')==m.digest(sid.encode()) and bool(__import__('re').fullmatch('inso_ci_[0-9a-f]{12}',marker.get('account',''))) and script.is_file() and not script.is_symlink() and m.sha(script)==m.sha(m.ROOT/'scripts/qualify-windows-msi.py')
        if not windows_host and not owned_python:raise RuntimeError('reported task engine is not an exact owned-account Windows task host or authenticated bootstrap action; no process stopped')
        if not kernel.TerminateProcess(process,0) or kernel.WaitForSingleObject(process,5000)!=0:raise RuntimeError('owned account task engine did not exit')
        return {'pid':pid,'user_sid_sha256':m.digest(sid.encode()),'image':path.name,'authenticated_bootstrap_action':owned_python,'terminated':True,'shared_system_process_stopped':False}
    finally:
        if token:kernel.CloseHandle(token)
        kernel.CloseHandle(process)

def inherited_ordinary_desktop(observed):
    """Observe the task's existing noninteractive context without changing it."""
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    user=C.WinDLL('user32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True)
    user.GetProcessWindowStation.restype=W.HANDLE;user.GetThreadDesktop.argtypes=[W.DWORD];user.GetThreadDesktop.restype=W.HANDLE;kernel.GetCurrentThreadId.restype=W.DWORD
    user.GetUserObjectInformationW.argtypes=[W.HANDLE,C.c_int,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)];user.GetUserObjectInformationW.restype=W.BOOL
    kernel.LocalFree.argtypes=[C.c_void_p];kernel.LocalFree.restype=C.c_void_p
    station=user.GetProcessWindowStation();desktop=user.GetThreadDesktop(kernel.GetCurrentThreadId())
    def name(handle):
        value=C.create_unicode_buffer(512);needed=W.DWORD()
        if not user.GetUserObjectInformationW(handle,2,value,C.sizeof(value),C.byref(needed)):raise C.WinError(C.get_last_error())
        return value.value
    flags=(W.DWORD*3)();needed=W.DWORD()
    if not user.GetUserObjectInformationW(station,1,flags,C.sizeof(flags),C.byref(needed)):raise C.WinError(C.get_last_error())
    station_name=name(station);desktop_name=name(desktop)
    if flags[2]&1 or station_name.casefold()=='winsta0':raise RuntimeError('ordinary bootstrap is not on its noninteractive task station')
    station_security=owned_user_object_observation(station,user,security,kernel);desktop_security=owned_user_object_observation(desktop,user,security,kernel)
    return {'kind':'inherited-task-noninteractive-context','station':station_name,'desktop':desktop_name,'visible':False,'station_or_desktop_modified':False,'user_sid_sha256':observed['user_sid_sha256'],'authentication_luid':observed['authentication_luid'],'actual_station_security':station_security,'actual_desktop_security':desktop_security,'startup_desktop':None}

def ordinary_system_broker(pipe,installer,directory,output):
    """Create the real normal actor without target task/secondary profile loads."""
    m.disposable_runner()
    if not __import__('re').fullmatch('insonic-ci-credential-[0-9a-f]{32}',pipe):raise ValueError('invalid owned credential pipe')
    import ctypes as C
    import ctypes.wintypes as W
    observed=token_info();evidence=output.with_suffix('.broker.json')
    def record(phase,**extra):evidence.write_text(json.dumps({'phase':phase,'broker_role':'SYSTEM disposable CI actor broker','broker_token':observed,'target_profile_loaded_by_broker':False,'password_published':False,**extra},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    record('broker-started')
    if observed['user_sid_sha256']!=m.digest(b'S-1-5-18') or observed['session_id']!=0:raise RuntimeError('ordinary actor broker is not SYSTEM in session0')
    with open('\\\\.\\pipe\\'+pipe,'rb',buffering=0) as source:raw=source.readline(8193)
    if not raw.endswith(b'\n') or len(raw)>8192:raise RuntimeError('owned credential pipe message exceeds bound')
    credentials=json.loads(raw);raw=None
    if set(credentials)!={'account','sid','password'} or credentials['account']!=os.environ['COMPUTERNAME']+'\\'+os.environ['USERNAME'] or not __import__('re').fullmatch('inso_ci_[0-9a-f]{12}',os.environ['USERNAME']):raise RuntimeError('owned credential pipe actor differs')
    kernel=C.WinDLL('kernel32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True)
    class Startup(C.Structure):_fields_=[('cb',W.DWORD),('reserved',W.LPWSTR),('desktop',W.LPWSTR),('title',W.LPWSTR),('x',W.DWORD),('y',W.DWORD),('width',W.DWORD),('height',W.DWORD),('charsx',W.DWORD),('charsy',W.DWORD),('fill',W.DWORD),('flags',W.DWORD),('show',W.WORD),('reserved2',W.WORD),('reserved_pointer',C.c_void_p),('stdin',W.HANDLE),('stdout',W.HANDLE),('stderr',W.HANDLE)]
    class Process(C.Structure):_fields_=[('process',W.HANDLE),('thread',W.HANDLE),('pid',W.DWORD),('tid',W.DWORD)]
    security.LogonUserW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.LPCWSTR,W.DWORD,W.DWORD,C.POINTER(W.HANDLE)];security.LogonUserW.restype=W.BOOL
    security.CreateProcessAsUserW.argtypes=[W.HANDLE,W.LPCWSTR,W.LPWSTR,C.c_void_p,C.c_void_p,W.BOOL,W.DWORD,C.c_void_p,W.LPCWSTR,C.POINTER(Startup),C.POINTER(Process)];security.CreateProcessAsUserW.restype=W.BOOL
    kernel.WaitForSingleObject.argtypes=[W.HANDLE,W.DWORD];kernel.WaitForSingleObject.restype=W.DWORD;kernel.GetExitCodeProcess.argtypes=[W.HANDLE,C.POINTER(W.DWORD)];kernel.GetExitCodeProcess.restype=W.BOOL;kernel.CloseHandle.argtypes=[W.HANDLE];kernel.TerminateProcess.argtypes=[W.HANDLE,W.UINT]
    process=Process();target=W.HANDLE()
    try:
        record('before-target-interactive-logon')
        if not security.LogonUserW(os.environ['USERNAME'],'.',credentials['password'],2,0,C.byref(target)):raise C.WinError(C.get_last_error())
        credentials['password']=None;target_observed=token_info(target)
        if target_observed['user_sid_sha256']!=m.digest(credentials['sid'].encode()) or target_observed['administrator_enabled'] or target_observed['elevated'] or target_observed['enabled_privileges']>1 or target_observed['integrity_rid']!=8192 or target_observed['session_id']!=0 or not target_observed['interactive_logon']:raise RuntimeError('broker target did not retain its actual normal-user identity/token')
        record('target-token-observed',target_token=target_observed)
        with private_context_desktop(credentials['sid'],'S-1-5-18') as desktop:
            startup=Startup();startup.cb=C.sizeof(startup);startup.flags=1;startup.show=0;startup.desktop=desktop['path']
            # Cross-session standard console handles cannot be inherited. The GUI
            # interpreter has no console initialization dependency; redirect Python
            # streams before loading the worker so even import/startup errors persist.
            executable=str(Path(sys.executable).with_name('pythonw.exe'))
            worker=[__file__,'--ordinary-worker',str(installer),'--context-dir',str(directory),'--output',str(output)]
            bootstrap='import os,runpy,sys;sys.stdout=sys.stderr=open('+repr(str(output.with_suffix('.worker-stream.log')))+',"w",encoding="utf-8",buffering=1);sys.argv='+repr(worker)+';sys.path.insert(0,os.path.dirname(sys.argv[0]));runpy.run_path(sys.argv[0],run_name="__main__")'
            args=[executable,'-u','-c',bootstrap];command=C.create_unicode_buffer(subprocess.list2cmdline(args));environment=C.create_unicode_buffer('\0'.join(key+'='+value for key,value in sorted(os.environ.items()))+'\0\0')
            record('before-target-process',target_token=target_observed,desktop=desktop)
            with qualifier_profile_privileges(('SeAssignPrimaryTokenPrivilege','SeIncreaseQuotaPrivilege')) as privilege_scope:
                if not security.CreateProcessAsUserW(target,executable,command,None,None,False,0x08000400,environment,str(directory.parent),C.byref(startup),C.byref(process)):raise C.WinError(C.get_last_error())
            record('target-process-created',pid=process.pid,target_token=target_observed,desktop=desktop,broker_privilege_scope=privilege_scope)
            if kernel.WaitForSingleObject(process.process,230000)!=0:
                kernel.TerminateProcess(process.process,1);kernel.WaitForSingleObject(process.process,5000);raise RuntimeError('exact owned normal worker exceeded full-UI task bound')
            code=W.DWORD()
            if not kernel.GetExitCodeProcess(process.process,C.byref(code)):raise C.WinError(C.get_last_error())
            record('target-process-exited',pid=process.pid,exit_code=code.value,target_token=target_observed,desktop=desktop)
            if code.value:raise RuntimeError('ordinary worker failed '+str(code.value))
    finally:
        credentials['password']=None
        if process.process and kernel.WaitForSingleObject(process.process,0)!=0:
            kernel.TerminateProcess(process.process,1);kernel.WaitForSingleObject(process.process,5000)
        for handle in (process.thread,process.process,target):
            if handle:kernel.CloseHandle(handle)

@__import__('contextlib').contextmanager
def owned_user_object_access(handle,user_sid,mask,user,security,kernel):
    """Temporary exact-account DACL only; retain owner/group/mandatory label."""
    m.disposable_runner();private_desktop_sddl(user_sid)
    import ctypes as C
    import ctypes.wintypes as W
    info=W.DWORD(4);converted=C.c_void_p();attempted=False;evidence={'restored':False,'trustee_sid_sha256':m.digest(user_sid.encode()),'ace_mask':mask}
    def checked(value):
        if not value:raise C.WinError(C.get_last_error())
        return value
    def descriptor(value):
        text=C.c_void_p();checked(security.ConvertSecurityDescriptorToStringSecurityDescriptorW(value,1,4,C.byref(text),None))
        try:return C.wstring_at(text)
        finally:kernel.LocalFree(text)
    def current():
        size=W.DWORD();user.GetUserObjectSecurity(handle,C.byref(info),None,0,C.byref(size))
        if not 0<size.value<=65536:raise RuntimeError('invalid CI user-object descriptor bound')
        raw=C.create_string_buffer(size.value);checked(user.GetUserObjectSecurity(handle,C.byref(info),raw,size.value,C.byref(size)))
        return raw,descriptor(raw)
    before,before_text=current();evidence['original_security']=owned_user_object_observation(handle,user,security,kernel)
    if not evidence['original_security'].get('descriptor_sha256'):raise RuntimeError('CI user-object owner/group/DACL/label observation is unavailable')
    try:
        checked(security.ConvertStringSecurityDescriptorToSecurityDescriptorW(before_text+'(A;;'+hex(mask)+';;;'+user_sid+')',1,C.byref(converted),None))
        expected=descriptor(converted);attempted=True
        checked(user.SetUserObjectSecurity(handle,C.byref(info),converted))
        if current()[1]!=expected:raise RuntimeError('CI exact-account user-object grant differs')
        evidence.update({'original_dacl_sha256':m.digest(before_text.encode()),'granted_dacl_sha256':m.digest(expected.encode())})
        yield evidence
    finally:
        try:
            if attempted:
                if current()[1] not in (before_text,expected):raise RuntimeError('CI user-object DACL changed outside the exact grant; restoration refused')
                checked(user.SetUserObjectSecurity(handle,C.byref(info),before))
                evidence['restored_security']=owned_user_object_observation(handle,user,security,kernel)
                evidence['restored']=current()[1]==before_text and evidence['restored_security']==evidence['original_security']
                if not evidence['restored']:raise RuntimeError('CI user-object owner/group/DACL/label restoration differs')
        finally:
            if converted:kernel.LocalFree(converted)


@__import__('contextlib').contextmanager
def interactive_context_desktop(user_sid,caller_sid,stage):
    """Use the disposable CI Default desktop selected by MSI EXE actions.

    Add only the unique owned account's temporary station/desktop access.
    Never switch input desktops or change owner/group/mandatory labels.
    """
    m.disposable_runner();private_desktop_sddl(user_sid,caller_sid)
    import ctypes as C
    import ctypes.wintypes as W
    stage=Path(stage);marker=json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'))
    if stage.is_symlink() or stage.resolve().parent!=Path(os.environ['ProgramData']).resolve() or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name) or marker.get('sid_sha256')!=m.digest(user_sid.encode()) or not __import__('re').fullmatch('inso_ci_[0-9a-f]{12}',marker.get('account','')):raise RuntimeError('CI desktop grant requires the exact owned disposable account stage')
    user=C.WinDLL('user32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True)
    class Flags(C.Structure):_fields_=[('inherit',W.BOOL),('reserved',W.BOOL),('flags',W.DWORD)]
    functions={'GetProcessWindowStation':([],W.HANDLE),'SetProcessWindowStation':([W.HANDLE],W.BOOL),'OpenWindowStationW':([W.LPCWSTR,W.BOOL,W.DWORD],W.HANDLE),'OpenDesktopW':([W.LPCWSTR,W.DWORD,W.BOOL,W.DWORD],W.HANDLE),'CloseWindowStation':([W.HANDLE],W.BOOL),'CloseDesktop':([W.HANDLE],W.BOOL),'GetUserObjectInformationW':([W.HANDLE,C.c_int,C.c_void_p,W.DWORD,C.POINTER(W.DWORD)],W.BOOL),'GetUserObjectSecurity':([W.HANDLE,C.POINTER(W.DWORD),C.c_void_p,W.DWORD,C.POINTER(W.DWORD)],W.BOOL),'SetUserObjectSecurity':([W.HANDLE,C.POINTER(W.DWORD),C.c_void_p],W.BOOL)}
    for name,(args,result) in functions.items():function=getattr(user,name);function.argtypes=args;function.restype=result
    kernel.LocalFree.argtypes=[C.c_void_p]
    security.ConvertStringSecurityDescriptorToSecurityDescriptorW.argtypes=[W.LPCWSTR,W.DWORD,C.POINTER(C.c_void_p),C.c_void_p];security.ConvertStringSecurityDescriptorToSecurityDescriptorW.restype=W.BOOL
    security.ConvertSecurityDescriptorToStringSecurityDescriptorW.argtypes=[C.c_void_p,W.DWORD,W.DWORD,C.POINTER(C.c_void_p),C.c_void_p];security.ConvertSecurityDescriptorToStringSecurityDescriptorW.restype=W.BOOL
    def checked(value):
        if not value:raise C.WinError(C.get_last_error())
        return value
    observed=token_info()
    if observed['session_id']<=0 or observed['user_sid_sha256']!=m.digest(caller_sid.encode()):raise RuntimeError('full-UI fixture requires the existing interactive CI caller session')
    original=user.GetProcessWindowStation();station=checked(user.OpenWindowStationW('WinSta0',False,0x0006000a));desktop=None;flags=Flags();size=W.DWORD()
    evidence={'kind':'exact-account-disposable-CI-default-desktop','path':r'WinSta0\Default','session_id':observed['session_id'],'target_user_sid_sha256':m.digest(user_sid.encode()),'caller_sid_sha256':m.digest(caller_sid.encode()),'visible_switch':False,'default_desktop_acl_modified':True,'station_dacl_restored':False,'desktop_dacl_restored':False}
    try:
        checked(user.GetUserObjectInformationW(station,1,C.byref(flags),C.sizeof(flags),C.byref(size)))
        if not flags.flags&1:raise RuntimeError('CI WinSta0 is not interactive; full UI cannot be qualified here')
        with owned_user_object_access(station,user_sid,0x000f037f,user,security,kernel) as station_grant:
            evidence['station_grant']=station_grant
            checked(user.SetProcessWindowStation(station))
            try:desktop=checked(user.OpenDesktopW('Default',0,False,0x00060081))
            finally:checked(user.SetProcessWindowStation(original))
            with owned_user_object_access(desktop,user_sid,0x000f01ff,user,security,kernel) as desktop_grant:
                evidence.update({'desktop_grant':desktop_grant,'station_visible':True,'caller_station_restored':user.GetProcessWindowStation()==original})
                Path(stage,'ui-actor-desktop.json').write_text(json.dumps(evidence),encoding='utf-8',newline='\n')
                yield evidence
            evidence['desktop_dacl_restored']=desktop_grant['restored']
        evidence['station_dacl_restored']=station_grant['restored']
    finally:
        try:
            if user.GetProcessWindowStation()!=original:checked(user.SetProcessWindowStation(original))
            evidence['desktop_dacl_restored']=evidence.get('desktop_grant',{}).get('restored',False)
            evidence['station_dacl_restored']=evidence.get('station_grant',{}).get('restored',False)
        finally:
            if desktop:checked(user.CloseDesktop(desktop))
            checked(user.CloseWindowStation(station))


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

@__import__('contextlib').contextmanager
def disposable_server_installer_policy(record=None):
    """Temporarily model client install policy on this disposable Server2022 host.

    Product installers never call this CI qualification fixture or modify policy.
    The original raw value, type and absence are restored even after failure.
    """
    m.disposable_runner()
    version=sys.getwindowsversion()
    if version.product_type!=3 or version.build!=20348:raise RuntimeError('installer policy fixture requires disposable Windows Server2022 build20348')
    import winreg
    path=r'Software\Policies\Microsoft\Windows\Installer';view=winreg.KEY_WOW64_64KEY
    def read(root,name):
        try:
            with winreg.OpenKey(root,path,0,winreg.KEY_READ|view) as key:
                try:value,kind=winreg.QueryValueEx(key,name);return {'key_present':True,'value_present':True,'type':kind,'value':value}
                except FileNotFoundError:return {'key_present':True,'value_present':False}
        except FileNotFoundError:return {'key_present':False,'value_present':False}
    def public(raw):
        return {key:({'bytes_hex':value.hex()} if isinstance(value,bytes) else value) for key,value in raw.items()}
    original=read(winreg.HKEY_LOCAL_MACHINE,'DisableMSI')
    elevated_before={'machine':read(winreg.HKEY_LOCAL_MACHINE,'AlwaysInstallElevated'),'calling_user':read(winreg.HKEY_CURRENT_USER,'AlwaysInstallElevated')}
    absent=[];parts=path.split('\\')
    for count in range(1,len(parts)+1):
        ancestor='\\'.join(parts[:count])
        try:
            with winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE,ancestor,0,winreg.KEY_READ|view):pass
        except FileNotFoundError:absent.append(ancestor)
    evidence={'kind':'disposable-ci-server-policy-fixture','host':{'product_type':version.product_type,'build':version.build},'registry_path':'HKLM\\'+path,'value_name':'DisableMSI','original':public(original),'fixture':{'type':winreg.REG_DWORD,'value':0},'restored':False,'always_install_elevated_before':{key:public(value) for key,value in elevated_before.items()},'product_changes_endpoint_policy':False,'policy_reference':'https://learn.microsoft.com/en-us/windows/win32/msi/disablemsi'}
    applied=False
    try:
        if record:record('CI-server-policy-before-fixture',policy=evidence)
        with winreg.CreateKeyEx(winreg.HKEY_LOCAL_MACHINE,path,0,winreg.KEY_READ|winreg.KEY_WRITE|view) as key:
            applied=True;winreg.SetValueEx(key,'DisableMSI',0,winreg.REG_DWORD,0)
        if read(winreg.HKEY_LOCAL_MACHINE,'DisableMSI')!={'key_present':True,'value_present':True,'type':winreg.REG_DWORD,'value':0}:raise RuntimeError('CI installer policy fixture was not observed')
        if record:record('CI-server-policy-fixture-observed',policy=evidence)
        yield evidence
    finally:
        if applied:
            with winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE,path,0,winreg.KEY_READ|winreg.KEY_WRITE|view) as key:
                if original['value_present']:winreg.SetValueEx(key,'DisableMSI',0,original['type'],original['value'])
                else:
                    try:winreg.DeleteValue(key,'DisableMSI')
                    except FileNotFoundError:pass
            # Delete only newly created, still-empty keys, never preexisting keys.
            removed=[]
            for ancestor in reversed(absent):
                try:
                    with winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE,ancestor,0,winreg.KEY_READ|view) as key:
                        subkeys,values,_=winreg.QueryInfoKey(key)
                    if subkeys or values:continue
                    winreg.DeleteKeyEx(winreg.HKEY_LOCAL_MACHINE,ancestor,view,0);removed.append(ancestor)
                except FileNotFoundError:pass
            actual=read(winreg.HKEY_LOCAL_MACHINE,'DisableMSI')
            elevated_after={'machine':read(winreg.HKEY_LOCAL_MACHINE,'AlwaysInstallElevated'),'calling_user':read(winreg.HKEY_CURRENT_USER,'AlwaysInstallElevated')}
            evidence.update({'restored':actual==original,'restored_value':public(actual),'removed_empty_owned_keys':removed,'always_install_elevated_unchanged':elevated_after==elevated_before})
            if record:record('CI-server-policy-restored',policy=evidence)
            if actual!=original or elevated_after!=elevated_before:raise RuntimeError('CI installer policy fixture did not exactly restore original policy')

PROFILE_LEASE_OBSERVER=r'''
$ErrorActionPreference='Stop'
[Console]::InputEncoding=[Text.UTF8Encoding]::new($false);[Console]::OutputEncoding=[Text.UTF8Encoding]::new($false)
$inputObject=([Console]::In.ReadToEnd()|ConvertFrom-Json)
$sid=[string]$inputObject.sid;$ownedPath=[IO.Path]::GetFullPath([string]$inputObject.profile)
if ($sid -notmatch '^S-1-5-21-(?:[0-9]+-){3}[0-9]+$') { throw 'invalid owned profile SID' }
$result=@{profile=$null;events=@();machine_state_modified=$false}
try {
  $profiles=@(Get-CimInstance -ClassName Win32_UserProfile -Filter ("SID='"+$sid+"'"))
  if ($profiles.Count -gt 1) { throw 'multiple exact profile registrations' }
  if ($profiles.Count -eq 1) { $profile=$profiles[0];$result.profile=@{local_path=$profile.LocalPath;loaded=$profile.Loaded;ref_count=$profile.refCount;special=$profile.Special;status=$profile.Status} }
} catch { $result.profile_query_hresult=$_.Exception.HResult }
foreach ($channel in @('Application','Microsoft-Windows-User Profile Service/Operational')) {
  try {
    $filter=@{LogName=$channel;StartTime=[DateTime]::UtcNow.AddMinutes(-10)}
    if ($channel -eq 'Application') { $filter.ProviderName='Microsoft-Windows-User Profiles Service' }
    $events=@(Get-WinEvent -FilterHashtable $filter -MaxEvents 100 -ErrorAction Stop | Where-Object {
      $candidate=$_;$matchesOwned=$candidate.Message -and ($candidate.Message.Contains($sid) -or $candidate.Message.IndexOf($ownedPath,[StringComparison]::OrdinalIgnoreCase) -ge 0)
      if (-not $matchesOwned) { foreach ($property in $candidate.Properties) { $value=[string]$property.Value;if ($value.Contains($sid) -or $value.IndexOf($ownedPath,[StringComparison]::OrdinalIgnoreCase) -ge 0) { $matchesOwned=$true;break } } }
      $matchesOwned
    } | Select-Object -First 12)
    foreach ($event in $events) { $message=$event.Message;if ($message.Length -gt 6000) { $message=$message.Substring(0,6000) };$result.events+=@{channel=$channel;id=$event.Id;time=$event.TimeCreated.ToUniversalTime().ToString('o');message=$message} }
  } catch { $result[('events_error_'+$channel)]=$_.Exception.HResult }
}
$result|ConvertTo-Json -Depth 5 -Compress
'''

def profile_lease_observation(sid,profile,deadline=None):
    """Read exact owned profile leases and related events, without unloading."""
    m.disposable_runner()
    if not __import__('re').fullmatch(r'S-1-5-21-(?:[0-9]+-){3}[0-9]+',sid):raise ValueError('profile lease observation requires the exact owned SID')
    import base64
    executable=Path(os.environ['SystemRoot'])/'System32/WindowsPowerShell/v1.0/powershell.exe'
    # Optional observations cannot consume the service-release/deletion budget.
    bound=min(2,max(.1,deadline-time.monotonic())) if deadline is not None else 2
    try:result=subprocess.run([str(executable),'-NoLogo','-NoProfile','-NonInteractive','-EncodedCommand',base64.b64encode(PROFILE_LEASE_OBSERVER.encode('utf-16le')).decode('ascii')],input=json.dumps({'sid':sid,'profile':str(profile)}).encode('utf-8'),stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=bound)
    except subprocess.TimeoutExpired:return {'observer_timeout':True,'machine_state_modified':False}
    if result.returncode:return {'observer_exit':result.returncode,'machine_state_modified':False}
    observed=json.loads(result.stdout.decode('utf-8-sig'));owned=observed.get('profile')
    if owned and (owned.get('special') or Path(owned['local_path']).resolve()!=Path(profile).resolve()):return {'provider_identity_mismatch':True,'machine_state_modified':False}
    for event in observed.get('events',[]):event['message']=__import__('re').sub(r'S-1-(?:[0-9]+-)*[0-9]+',lambda match:'SID_SHA256_'+m.digest(match.group().encode()),event['message'])
    observed['sid_sha256']=m.digest(sid.encode());return observed

PROFILE_PROVIDER_DELETE=r'''
$ErrorActionPreference='Stop'
[Console]::InputEncoding=[Text.UTF8Encoding]::new($false);[Console]::OutputEncoding=[Text.UTF8Encoding]::new($false)
$result=@{method='Remove-CimInstance exact owned Win32_UserProfile';complete_absence=$false;phase='starting';forced_hive_unload=$false;foreign_process_stopped=$false}
try {
  $request=([Console]::In.ReadToEnd()|ConvertFrom-Json);$sid=[string]$request.sid;$path=[IO.Path]::GetFullPath([string]$request.profile)
  if ($sid -notmatch '^S-1-5-21-(?:[0-9]+-){3}[0-9]+$') { throw 'invalid owned SID' }
  $result.phase='exact-provider-query'
  $profiles=@(Get-CimInstance -ClassName Win32_UserProfile -Filter ("SID='"+$sid+"'"))
  if ($profiles.Count -ne 1) { throw 'expected exactly one owned profile instance' }
  $profile=$profiles[0]
  if ($profile.SID -ne $sid -or $profile.Special -or -not [string]::Equals([IO.Path]::GetFullPath($profile.LocalPath),$path,[StringComparison]::OrdinalIgnoreCase)) { throw 'provider profile identity differs' }
  $result.before=@{loaded=$profile.Loaded;ref_count=$profile.refCount;special=$profile.Special;status=$profile.Status}
  $result.phase='exact-provider-delete'
  Remove-CimInstance -InputObject $profile -Confirm:$false -OperationTimeoutSec 2 -ErrorAction Stop
  $result.phase='verify-complete-absence'
  $remaining=@(Get-CimInstance -ClassName Win32_UserProfile -Filter ("SID='"+$sid+"'"))
  $result.provider_absent=($remaining.Count -eq 0)
  $result.registration_absent=(-not (Test-Path -LiteralPath ('Registry::HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\'+$sid)))
  $result.user_hive_absent=(-not (Test-Path -LiteralPath ('Registry::HKEY_USERS\'+$sid)))
  $result.classes_hive_absent=(-not (Test-Path -LiteralPath ('Registry::HKEY_USERS\'+$sid+'_Classes')))
  $result.folder_absent=(-not (Test-Path -LiteralPath $path))
  $result.complete_absence=($result.provider_absent -and $result.registration_absent -and $result.user_hive_absent -and $result.classes_hive_absent -and $result.folder_absent)
  $result.phase='completed'
} catch { $result.failed_boundary=$result.phase;$result.phase='failed';$result.hresult=$_.Exception.HResult;$result.error=$_.Exception.Message }
$result|ConvertTo-Json -Depth 5 -Compress
'''

def delete_owned_profile_provider(sid,profile,deadline):
    """Ask the documented provider to delete one fully authenticated CI profile."""
    m.disposable_runner();validate_account_profile(Path(profile).name,profile)
    if not __import__('re').fullmatch(r'S-1-5-21-(?:[0-9]+-){3}[0-9]+',sid):raise ValueError('profile provider deletion requires the exact owned SID')
    if owned_actor_processes(sid):raise RuntimeError('owned actor still has processes; provider deletion refused')
    import base64
    remaining=deadline-time.monotonic()
    if remaining<=0:return {'complete_absence':False,'deadline_expired':True}
    executable=Path(os.environ['SystemRoot'])/'System32/WindowsPowerShell/v1.0/powershell.exe'
    try:result=subprocess.run([str(executable),'-NoLogo','-NoProfile','-NonInteractive','-EncodedCommand',base64.b64encode(PROFILE_PROVIDER_DELETE.encode('utf-16le')).decode('ascii')],input=json.dumps({'sid':sid,'profile':str(profile)}).encode('utf-8'),stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=min(4,remaining))
    except subprocess.TimeoutExpired:return {'complete_absence':False,'provider_timeout':True}
    if result.returncode:return {'complete_absence':False,'provider_exit':result.returncode}
    proof=json.loads(result.stdout.decode('utf-8-sig'))
    absent_keys=('provider_absent','registration_absent','user_hive_absent','classes_hive_absent','folder_absent')
    if proof.get('complete_absence') is True and (proof.get('phase')!='completed' or not all(proof.get(key) is True for key in absent_keys)):raise RuntimeError('profile provider did not prove every required absence')
    if proof.get('complete_absence') is True:
        actual=profile_cleanup_observation(sid,profile)
        if Path(profile).exists() or actual.get('profile_registration_error')!=2 or any(actual.get('hives',{}).get(key,{}).get('present') is not False or actual['hives'][key].get('winerror')!=2 for key in ('user','_Classes')):raise RuntimeError('profile provider success differs from actual owned folder/registration/hive absence')
        proof['independent_absence']=actual
    proof['sid_sha256']=m.digest(sid.encode());return proof

def owned_actor_processes(sid):
    """Observe exact account processes for profile cleanup, never stop any."""
    m.disposable_runner()
    import ctypes as C
    import ctypes.wintypes as W
    kernel=C.WinDLL('kernel32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True)
    class Entry(C.Structure):_fields_=[('size',W.DWORD),('usage',W.DWORD),('pid',W.DWORD),('heap',C.c_size_t),('module',W.DWORD),('threads',W.DWORD),('parent',W.DWORD),('priority',W.LONG),('flags',W.DWORD),('image',W.WCHAR*260)]
    kernel.CreateToolhelp32Snapshot.argtypes=[W.DWORD,W.DWORD];kernel.CreateToolhelp32Snapshot.restype=W.HANDLE
    kernel.Process32FirstW.argtypes=[W.HANDLE,C.POINTER(Entry)];kernel.Process32FirstW.restype=W.BOOL;kernel.Process32NextW.argtypes=[W.HANDLE,C.POINTER(Entry)];kernel.Process32NextW.restype=W.BOOL
    kernel.OpenProcess.argtypes=[W.DWORD,W.BOOL,W.DWORD];kernel.OpenProcess.restype=W.HANDLE;kernel.CloseHandle.argtypes=[W.HANDLE]
    kernel.QueryFullProcessImageNameW.argtypes=[W.HANDLE,W.DWORD,W.LPWSTR,C.POINTER(W.DWORD)];kernel.QueryFullProcessImageNameW.restype=W.BOOL
    security.OpenProcessToken.argtypes=[W.HANDLE,W.DWORD,C.POINTER(W.HANDLE)];security.OpenProcessToken.restype=W.BOOL
    snapshot=kernel.CreateToolhelp32Snapshot(2,0)
    if snapshot==C.c_void_p(-1).value:raise C.WinError(C.get_last_error())
    result=[];entry=Entry();entry.size=C.sizeof(entry)
    try:
        present=kernel.Process32FirstW(snapshot,C.byref(entry))
        while present:
            process=kernel.OpenProcess(0x1000,False,entry.pid);token=W.HANDLE()
            if process:
                try:
                    if security.OpenProcessToken(process,8,C.byref(token)) and account_sid(token)==sid:
                        image=C.create_unicode_buffer(32768);size=W.DWORD(len(image));observed={'pid':entry.pid,'parent_pid':entry.parent,'image':entry.image,'machine_state_modified':False}
                        if kernel.QueryFullProcessImageNameW(process,0,image,C.byref(size)):observed['image_path']=image.value
                        result.append(observed)
                finally:
                    if token:kernel.CloseHandle(token)
                    kernel.CloseHandle(process)
            present=kernel.Process32NextW(snapshot,C.byref(entry))
        if C.get_last_error()!=18:raise C.WinError(C.get_last_error())
    finally:kernel.CloseHandle(snapshot)
    return result

def profile_cleanup_observation(sid,profile):
    """Read only this exact created actor's profile registration and hive state."""
    m.disposable_runner()
    import winreg
    if not __import__('re').fullmatch(r'S-1-5-21-(?:[0-9]+-){3}[0-9]+',sid):raise ValueError('profile observation requires a created ordinary account SID')
    result={'sid_sha256':m.digest(sid.encode()),'hives':{},'profile_registration':{},'machine_state_modified':False,'owned_actor_processes':owned_actor_processes(sid)}
    for suffix in ('','_Classes'):
        try:
            with winreg.OpenKey(winreg.HKEY_USERS,sid+suffix,0,winreg.KEY_READ):pass
            result['hives'][suffix or 'user']={'present':True}
        except OSError as error:result['hives'][suffix or 'user']={'present':False,'winerror':getattr(error,'winerror',None)}
    try:
        with winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE,r'Software\Microsoft\Windows NT\CurrentVersion\ProfileList'+'\\'+sid,0,winreg.KEY_READ|winreg.KEY_WOW64_64KEY) as key:
            for name in ('ProfileImagePath','State','RefCount'):
                try:
                    value,kind=winreg.QueryValueEx(key,name)
                    if name=='ProfileImagePath':result['profile_registration'][name]={'matches_owned_path':Path(os.path.expandvars(value)).resolve()==Path(profile).resolve(),'type':kind}
                    else:result['profile_registration'][name]={'value':value,'type':kind}
                except OSError as error:result['profile_registration'][name]={'present':False,'winerror':getattr(error,'winerror',None)}
    except OSError as error:result['profile_registration_error']=getattr(error,'winerror',None)
    return result

def worker_failure_path(output,bootstrap=False):
    return Path(output).with_suffix('.broker-failure.json' if bootstrap else '.failure.json')

@__import__('contextlib').contextmanager
def qualifier_profile_privileges(names=('SeBackupPrivilege','SeRestorePrivilege')):
    """Scope required existing rights on the calling CI token and restore them."""
    m.disposable_runner()
    if names not in (('SeBackupPrivilege','SeRestorePrivilege'),('SeAssignPrimaryTokenPrivilege','SeIncreaseQuotaPrivilege')):raise ValueError('unsupported CI process privilege scope')
    import ctypes as C
    import ctypes.wintypes as W
    kernel=C.WinDLL('kernel32',use_last_error=True);security=C.WinDLL('advapi32',use_last_error=True)
    class Luid(C.Structure):_fields_=[('low',W.DWORD),('high',W.LONG)]
    class Entry(C.Structure):_fields_=[('luid',Luid),('attributes',W.DWORD)]
    class Privileges(C.Structure):_fields_=[('count',W.DWORD),('entries',Entry*2)]
    kernel.GetCurrentProcess.restype=W.HANDLE;kernel.CloseHandle.argtypes=[W.HANDLE]
    security.OpenProcessToken.argtypes=[W.HANDLE,W.DWORD,C.POINTER(W.HANDLE)];security.OpenProcessToken.restype=W.BOOL
    security.LookupPrivilegeValueW.argtypes=[W.LPCWSTR,W.LPCWSTR,C.POINTER(Luid)];security.LookupPrivilegeValueW.restype=W.BOOL
    security.AdjustTokenPrivileges.argtypes=[W.HANDLE,W.BOOL,C.c_void_p,W.DWORD,C.c_void_p,C.POINTER(W.DWORD)];security.AdjustTokenPrivileges.restype=W.BOOL
    token=W.HANDLE();requested=Privileges();previous=Privileges();size=W.DWORD();changed=False
    if not security.OpenProcessToken(kernel.GetCurrentProcess(),0x28,C.byref(token)):raise C.WinError(C.get_last_error())
    try:
        requested.count=len(names)
        for index,name in enumerate(names):
            if not security.LookupPrivilegeValueW(None,name,C.byref(requested.entries[index].luid)):raise C.WinError(C.get_last_error())
            requested.entries[index].attributes=2
        C.set_last_error(0)
        ok=security.AdjustTokenPrivileges(token,False,C.byref(requested),C.sizeof(previous),C.byref(previous),C.byref(size));error=C.get_last_error();changed=bool(ok)
        if not ok or error:raise C.WinError(error)
        yield {'scope':'calling CI process token only','privileges':list(names),'ordinary_actor_token_changed':False}
    finally:
        try:
            if changed:
                C.set_last_error(0)
                if not security.AdjustTokenPrivileges(token,False,C.byref(previous),0,None,None) or C.get_last_error():raise C.WinError(C.get_last_error())
        finally:kernel.CloseHandle(token)

def ordinary_context(installer,directory,output,model=None,service_ownership=None,diagnostic_only=False,setup_only=False,setup_path=None,setup_receipt=None):
    """Qualify an actual ordinary account, exclusively on a disposable runner."""
    m.disposable_runner()
    import secrets
    import ctypes as C
    import ctypes.wintypes as W
    security=C.WinDLL('advapi32',use_last_error=True);kernel=C.WinDLL('kernel32',use_last_error=True);network=C.WinDLL('netapi32',use_last_error=True);profiles=C.WinDLL('userenv',use_last_error=True)
    class User(C.Structure):_fields_=[('name',W.LPWSTR),('password',W.LPWSTR),('password_age',W.DWORD),('privilege',W.DWORD),('home',W.LPWSTR),('comment',W.LPWSTR),('flags',W.DWORD),('script',W.LPWSTR)]
    class Member(C.Structure):_fields_=[('name',W.LPWSTR)]
    class Profile(C.Structure):_fields_=[('size',W.DWORD),('flags',W.DWORD),('name',W.LPWSTR),('path',W.LPWSTR),('default_path',W.LPWSTR),('server',W.LPWSTR),('policy',W.LPWSTR),('hive',W.HANDLE)]
    kernel.GetCurrentProcess.restype=W.HANDLE;kernel.CloseHandle.argtypes=[W.HANDLE];kernel.CloseHandle.restype=W.BOOL
    kernel.WaitForSingleObject.argtypes=[W.HANDLE,W.DWORD];kernel.WaitForSingleObject.restype=W.DWORD;kernel.GetExitCodeProcess.argtypes=[W.HANDLE,C.POINTER(W.DWORD)];kernel.GetExitCodeProcess.restype=W.BOOL;kernel.TerminateProcess.argtypes=[W.HANDLE,W.UINT];kernel.TerminateProcess.restype=W.BOOL
    security.OpenProcessToken.argtypes=[W.HANDLE,W.DWORD,C.POINTER(W.HANDLE)];security.OpenProcessToken.restype=W.BOOL
    security.LogonUserW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.LPCWSTR,W.DWORD,W.DWORD,C.POINTER(W.HANDLE)];security.LogonUserW.restype=W.BOOL
    profiles.LoadUserProfileW.argtypes=[W.HANDLE,C.POINTER(Profile)];profiles.LoadUserProfileW.restype=W.BOOL;profiles.UnloadUserProfile.argtypes=[W.HANDLE,W.HANDLE];profiles.UnloadUserProfile.restype=W.BOOL
    network.NetUserAdd.argtypes=[W.LPCWSTR,W.DWORD,C.c_void_p,C.POINTER(W.DWORD)];network.NetUserAdd.restype=W.DWORD;network.NetUserDel.argtypes=[W.LPCWSTR,W.LPCWSTR];network.NetUserDel.restype=W.DWORD
    network.NetLocalGroupAddMembers.argtypes=[W.LPCWSTR,W.LPCWSTR,W.DWORD,C.c_void_p,W.DWORD];network.NetLocalGroupAddMembers.restype=W.DWORD
    profiles.GetUserProfileDirectoryW.argtypes=[W.HANDLE,W.LPWSTR,C.POINTER(W.DWORD)];profiles.GetUserProfileDirectoryW.restype=W.BOOL;profiles.DeleteProfileW.argtypes=[W.LPCWSTR,W.LPCWSTR,W.LPCWSTR];profiles.DeleteProfileW.restype=W.BOOL
    name='inso_ci_'+uuid.uuid4().hex[:12];password='Aa1!'+secrets.token_hex(24);created=False;profile=None;stage=None;token=W.HANDLE();caller=W.HANDLE();profile_info=Profile();profile_loaded=False;sid=None;evidence=None;failure=None;cleanup=[];batch_original=None
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
        profile_info.size=C.sizeof(profile_info);profile_info.flags=1;profile_info.name=name
        record('LoadUserProfileW-owned')
        with qualifier_profile_privileges() as privilege_scope:
            if not profiles.LoadUserProfileW(token,C.byref(profile_info)):raise C.WinError(C.get_last_error())
            profile_loaded=True
        value=C.create_unicode_buffer(4096);size=W.DWORD(4096)
        record('GetUserProfileDirectoryW-ordinary')
        if not profiles.GetUserProfileDirectoryW(token,value,C.byref(size)):raise C.WinError(C.get_last_error())
        profile=validate_account_profile(name,value.value)
        if not profile.is_dir():raise RuntimeError('native profile load did not create the observed ordinary-user directory')
        record('ordinary-profile-observed',profile=str(profile),method='owned LoadUserProfileW handle, GetUserProfileDirectoryW',privilege_scope=privilege_scope)
        record('Copy-exact-owned-worker-inputs');scripts=stage/'scripts';scripts.mkdir();(stage/'temp').mkdir();native=stage/'native';native.mkdir()
        for filename in ('qualify-windows-msi.py','windows_msi.py','windows_setup.py','qualify_installer_setup.py','qualify_installer_ui.py','render-installed-help.mjs'):
            shutil.copyfile(m.ROOT/'scripts'/filename,scripts/filename)
            if m.sha(scripts/filename)!=m.sha(m.ROOT/'scripts'/filename):raise RuntimeError('ordinary staged script identity differs')
        target=stage/'fixture.msi';shutil.copyfile(installer,target)
        if m.sha(target)!=m.sha(Path(installer)):raise RuntimeError('ordinary MSI fixture copy differs')
        fixture=Path(directory).parent/'launcher-context-fixture';shutil.copytree(fixture,stage/'launcher-context-fixture')
        for original in fixture.iterdir():
            if original.is_file() and m.sha(original)!=m.sha(stage/'launcher-context-fixture'/original.name):raise RuntimeError('ordinary launcher fixture copy differs')
        from qualify_installer_ui import prepare
        prepare(stage,fixture,clone,diagnostic_only=diagnostic_only or setup_only,setup_probe=setup_only)
        if diagnostic_only:
            shutil.copyfile(m.ROOT/'scripts/diagnose-installer-completion.py',scripts/'diagnose-installer-completion.py')
            shutil.copyfile(m.ROOT/'scripts/windows_setup.py',scripts/'windows_setup.py')
            (stage/'completion-invocation-diagnostic.json').write_text('{"qualification":false}',encoding='utf-8',newline='\n')
            (stage/'installer-diagnostic-mode.json').write_text(json.dumps({'feature_qualification':False,'mode':os.environ.get('INSONIC_INSTALLER_DIAGNOSTIC_MODE','completion')}),encoding='utf-8',newline='\n')
            diagnostic_mode(stage)
        if setup_only:
            if not setup_path or not setup_receipt or m.sha(setup_path)!=setup_receipt['artifact_sha256'] or m.sha(target)!=setup_receipt['embedded_msi']['sha256']:raise RuntimeError('exact setup context source identity differs')
            shutil.copyfile(setup_path,stage/'exact-setup.exe')
            (stage/'setup-qualification.json').write_text(json.dumps({'feature_qualification':True,'msi_sha256':m.sha(target),'setup_sha256':m.sha(stage/'exact-setup.exe')}),encoding='utf-8',newline='\n')
        (stage/'context-model.json').write_text(json.dumps(model),encoding='utf-8',newline='\n')
        install=stage/'installed program é with spaces';worker_output=native/'context.json'
        args=[str(scripts/'qualify-windows-msi.py'),'--ordinary-worker',str(target),'--context-dir',str(install),'--output',str(worker_output)]
        ui_desktop={};namespace={}
        try:
            with disposable_server_installer_policy(record) as policy_fixture:
                with interactive_context_desktop(sid,caller_sid,stage) as ui_desktop:
                    record('parent-before-first-full-UI',token=token_info(),directories=object_namespace_diagnostic(),mutex=installer_mutex_diagnostic())
                    probe=interactive_ordinary(stage,name,sid,password,profile,ui_desktop,record,probe_only=True)
                    with interactive_context_namespace(stage,sid,caller_sid,ui_desktop,probe) as namespace:
                        interactive_ordinary(stage,name,sid,password,profile,ui_desktop,record)
                if diagnostic_only:
                    scheduler={'skipped':'invocation diagnostic only, no feature qualification'}
                    invocation=json.loads((native/'context.ui.json').read_text(encoding='utf-8'))
                    worker_output.write_text(json.dumps({'token':invocation['token'],'user_profile':str(profile),'os_policy_mutations':{},'diagnostic':invocation,'feature_qualification':False}),encoding='utf-8',newline='\n')
                elif setup_only:
                    scheduler={'skipped':'setup source/quiet and full lifecycle executed in the same ordinary interactive actor'}
                    setup_proof=json.loads((native/'context.ui.json').read_text(encoding='utf-8'))
                    worker_output.write_text(json.dumps({'token':setup_proof['token'],'user_profile':str(profile),'os_policy_mutations':{},'setup':setup_proof,'feature_qualification':True}),encoding='utf-8',newline='\n')
                else:scheduler=scheduler_ordinary(stage,name,sid,password,args,record,profile)
        finally:
            if namespace:record('owned-session-namespace-restoration',namespace=namespace)
            if ui_desktop:record('full-UI-default-desktop-restoration',desktop=ui_desktop)
        evidence=json.loads(worker_output.read_text(encoding='utf-8'))
        if evidence['token']['user_sid_sha256']!=m.digest(sid.encode()) or Path(evidence['user_profile']).resolve()!=profile.resolve():raise RuntimeError('ordinary worker changed account/profile identity')
        evidence.update({'process_creation':'full UI via CreateProcessWithLogonW flags0 in the existing interactive session on the disposable CI WinSta0 Default desktop with exact-account station/desktop DACL grants restored; separate quiet LogonUserW/CreateProcessAsUserW SYSTEM broker in private session0; exact station ACE restored and parent is sole profile-load owner','scheduler':scheduler,'profile_creation':'explicit owned LoadUserProfileW handle and observed GetUserProfileDirectoryW','fixture_msi_sha256':m.sha(target),'fixture_script_sha256':m.sha(scripts/'qualify-windows-msi.py'),'elevated_actor_sid_sha256':m.digest(caller_sid.encode()),'owned_account_rights_before':batch_original,'ci_host_policy_fixture':policy_fixture,'ci_session_namespace_fixture':namespace})
        evidence['os_policy_mutations'].update({'ci_server_disable_msi_fixture':True,'ci_server_disable_msi_restored':policy_fixture['restored'],'always_install_elevated_unchanged':policy_fixture['always_install_elevated_unchanged']})
    except BaseException as error:
        failure=error;record('failed',failed_api=getattr(error,'api',current_phase['name']),winerror=getattr(error,'winerror',None),errno=getattr(error,'errno',None),error_type=type(error).__name__,error=str(error))
    finally:
        if stage is not None and (stage/'native').is_dir():
            try:retain_ordinary_diagnostics(stage/'native',output)
            except BaseException as error:
                cleanup.append({'object':'ordinary diagnostic retention','complete':False,'error':str(error)})
                if failure is None:failure=error
        if profile_loaded:
            try:
                with qualifier_profile_privileges():unloaded=bool(profiles.UnloadUserProfile(token,profile_info.hive));unload_error=0 if unloaded else C.get_last_error()
                cleanup.append({'object':'explicit owned profile handle','unloaded':unloaded,'winerror':unload_error,'sid_sha256':m.digest(sid.encode())})
                if not unloaded and failure is None:failure=RuntimeError('owned profile handle did not unload '+str(unload_error))
            except BaseException as error:
                cleanup.append({'object':'explicit owned profile handle','unloaded':False,'error_type':type(error).__name__})
                if failure is None:failure=error
        for handle in (token,caller):
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
            validate_account_profile(name,profile);deleted=False;deadline=time.monotonic()+10;lease={}
            before_profile_delete=profile_cleanup_observation(sid,profile);delete_errors=[]
            if profiles.DeleteProfileW(sid,str(profile),None):deleted=True
            else:delete_errors.append(C.get_last_error())
            if not deleted and delete_errors[-1]==32 and service_ownership is not None and failure is None:
                try:
                    lease['before_service_release']=profile_lease_observation(sid,profile,deadline)
                    released=release_owned_installer_service(service_ownership,worker_output.with_suffix('.install.worker.json'),worker_output.with_suffix('.remove.worker.json'),scheduler,sid,deadline)
                    cleanup.append({'object':'owned Installer service activation','restored':released['restored_initial_stopped_state'],**released})
                    lease['after_service_release']=profile_lease_observation(sid,profile,deadline)
                except BaseException as error:
                    cleanup.append({'object':'owned Installer service activation','restored':False,'error_type':type(error).__name__,'error':str(error)})
                    if failure is None:failure=error
            if not deleted and delete_errors[-1]==32 and failure is None:
                lease['provider_delete']=delete_owned_profile_provider(sid,profile,deadline)
                deleted=lease['provider_delete'].get('complete_absence') is True
            for _ in range(49):
                if deleted or time.monotonic()>=deadline:break
                if profiles.DeleteProfileW(sid,str(profile),None):deleted=True;break
                delete_errors.append(C.get_last_error());time.sleep(min(.2,max(0,deadline-time.monotonic())))
            if not deleted and not lease:lease['remaining']=profile_lease_observation(sid,profile,deadline)
            cleanup.append({'object':'created profile','deleted':deleted,'sid_sha256':m.digest(sid.encode()),'delete_profile_winerrors':delete_errors,'remaining_lease':lease,'before_delete':before_profile_delete,'after_delete':profile_cleanup_observation(sid,profile)})
            if not deleted and failure is None:failure=RuntimeError('owned ordinary profile cleanup failed '+str(delete_errors[-1] if delete_errors else 0))
        if created:
            rc=network.NetUserDel(None,name);cleanup.append({'object':'created account','deleted':rc==0})
            if rc and failure is None:failure=RuntimeError('owned ordinary account cleanup failed '+str(rc))
        if stage is not None:
            base=Path(os.environ['ProgramData']).resolve()
            if stage.is_symlink() or stage.resolve().parent!=base or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name):raise ValueError('owned stage cleanup target changed')
            try:shutil.rmtree(stage);cleanup.append({'object':'owned staging directory','deleted':True})
            except BaseException as error:
                cleanup.append({'object':'owned staging directory','deleted':False,'error':str(error)})
                if failure is None:failure=error
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
    m.disposable_runner();service_ownership=service_diagnostic(configuration=True);path=Path(path);started=time.monotonic()
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
            phase('ordinary-first-full-UI-and-quiet-install-remove')
            payload=m.ROOT/'build/packages'/receipt['payload_receipt']['dirname']
            if payload.is_symlink() or payload.resolve().parent!=(m.ROOT/'build/packages').resolve():raise ValueError('ordinary-first payload root is redirected')
            for entry in model['payload']:
                if entry['path']!=m.OWNERSHIP and m.sha(payload/entry['path'])!=entry['sha256']:raise ValueError('ordinary-first payload differs from exact MSI model')
            fixture_evidence=prepare_launcher_fixture(payload,tmp/'launcher-context-fixture')
            import windows_setup
            setup_path=m.ROOT/'build/packages'/receipt['setup']['artifact']
            setup_identity,stub_size=windows_setup.read_payload(setup_path)
            if setup_identity!=receipt['setup']['embedded_msi'] or stub_size>128*1024*1024:raise RuntimeError('setup fixture program source differs')
            with setup_path.open('rb') as source,(tmp/'launcher-context-fixture/owned-setup.exe').open('wb') as target:target.write(source.read(stub_size))
            build_barrier=subprocess.run(['go','build','-ldflags','-H windowsgui','-o',str(tmp/'launcher-context-fixture/owned-cancel-barrier.exe'),str(m.ROOT/'tests/fixtures/setup-cancel-barrier.go')],cwd=m.ROOT,env=dict(os.environ,CGO_ENABLED='0'),stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=60)
            if build_barrier.returncode:raise RuntimeError('owned cancellation barrier compile failed: '+build_barrier.stderr.decode(errors='replace')[-2000:])
            context_evidence=ordinary_context(old,tmp/'ordinary per-user installation',logs/'ordinary-context.json',model,service_ownership)
            phase('ordinary-exact-setup-and-recovery')
            setup_evidence=ordinary_context(path,tmp/'ordinary setup installation',logs/'ordinary-setup-context.json',model,service_ownership,setup_only=True,setup_path=setup_path,setup_receipt=receipt['setup'])
            receipt['setup']['qualification'].update(setup_evidence['setup']['qualification'])
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
                    elevated_launch_evidence=elected_launcher_context(tmp/'launcher-context-fixture',allow_refusal=True)
                run_installer(['/x',installed],logs/'selection-remove.log');installed=None
                if any((install/f['path']).exists() for f in model['payload']):raise ValueError('uninstall retained installer-owned bytes')
            result['client_selections']='passed';result['launch_contract']='passed'
            phase('administrator-maintenance')
            run_installer(call(old,'Core,CLI',START_MENU='',DESKTOP_SHORTCUT=''),logs/'maintenance-initial.log');installed=m.product_code(version,upgrade);assert_unchecked_shortcuts()
            registered_feature_states(installed,logs/'feature-change.registered.json')
            run_installer(call(old,'Core,Desktop,CLI',REINSTALL='Core',REINSTALLMODE='omus'),logs/'feature-change.log');installed_hashes(install,model,True)
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
            from qualify_installer_ui import export_captures
            export_captures(m.ROOT/'build/native/installer-capture',logs,context_evidence['interactive'],receipt)
            result['user_context']='passed';result['interactive_transactions']='passed';result['documentation_completion']='passed'
            proof={'artifact_sha256':receipt['artifact_sha256'],'revision':receipt['revision'],'windows_build':str(__import__('sys').getwindowsversion().build),'fixture_upgrade_code':upgrade,'checks':result,'elapsed_seconds':round(time.monotonic()-started,3),'active_work_fixture':{'process':'running owned ping image, no process shutdown','rejected_actions':['alternate-install-directory upgrade','uninstall'],'alternate_directory_absent':True,'old_payload_hashes_preserved':True},'exact_shortcut_collision':{'fresh_cli_exact_name':'rejected before installation changes','unrelated_bytes':'preserved','failed_action':'CheckCliMenu'},'unrelated_integrations':{'start_menu_sibling':'preserved','desktop_sibling':'preserved','product_key_sibling_value':'preserved','sentinel_content_sha256':m.digest(integration_sentinels['content'])},'shortcut_migration':{'fresh_unchecked_properties':'empty values normalized to zero','upgrade_overrides':False,'saved_elections':final_shortcut_elections},'policy_prompts':'not-observed','signing':'unsigned','cancellation':cancellation_evidence,'installer_service_initial':service_ownership,'user_context':{'original_elevated_token':original_token,'ordinary_process':context_evidence},'setup':setup_evidence['setup'],'elected_launcher':{'fixture':fixture_evidence,'elevated_caller':elevated_launch_evidence,'ordinary_caller':context_evidence['elected_launcher']}}
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
    p=argparse.ArgumentParser();p.add_argument('installer',type=Path,nargs='?');p.add_argument('--receipt',type=Path);p.add_argument('--cancel-worker',type=Path);p.add_argument('--installer-worker',type=Path);p.add_argument('--ordinary-worker',type=Path);p.add_argument('--ordinary-broker');p.add_argument('--ordinary-ui-worker',type=Path);p.add_argument('--ordinary-namespace-probe',type=Path);p.add_argument('--context-dir',type=Path);p.add_argument('--output',type=Path);a=p.parse_args();m.disposable_runner()
    if a.ordinary_namespace_probe:ordinary_namespace_probe(a.ordinary_namespace_probe)
    elif a.ordinary_ui_worker:ordinary_ui_worker(a.ordinary_ui_worker)
    elif a.installer_worker:native_installer_worker(a.installer_worker)
    elif a.cancel_worker:cancellation_worker(a.cancel_worker,a.context_dir,a.output)
    elif a.ordinary_worker or a.ordinary_broker:
        try:
            if a.ordinary_broker:ordinary_system_broker(a.ordinary_broker,a.installer,a.context_dir,a.output)
            else:ordinary_worker(a.ordinary_worker,a.context_dir,a.output)
        except BaseException as error:
            worker_failure_path(a.output,bool(a.ordinary_broker)).write_text(json.dumps({'kind':'ordinary-context-broker-failure' if a.ordinary_broker else 'ordinary-context-worker-failure','pid':os.getpid(),'error_type':type(error).__name__,'error':str(error)[-3000:]},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
            raise
    elif a.installer and a.receipt:r=json.loads(a.receipt.read_text(encoding='utf-8'));print(json.dumps(qualify(a.installer,r)))
    else:p.error('installer and --receipt are required')
