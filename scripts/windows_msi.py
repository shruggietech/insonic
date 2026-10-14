#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Native MSI database authoring and portable independent table/CAB verification.

This module never invokes installation or UI APIs. Qualification and UI capture
require a disposable GitHub-hosted Windows runner and explicit acknowledgement.
"""
import ctypes
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import struct
import subprocess
import tempfile
import uuid
import zlib
ROOT=Path(__file__).resolve().parent.parent
NAMESPACE=uuid.UUID('7cb51873-d4e8-4b86-92de-69fb9a53dc78')
ICON='brand/kit/icons/windows/classic/app.ico'
UNSIGNED_SIGNING={'configured':False,'verified':False,'status':'unconfigured'}
OWNERSHIP='insonic-msi-ownership.json'
OWNERSHIP_BYTES=b'{"family":"insonic","kind":"windows-msi-installation"}\n'
QUALIFICATION_KEYS=('client_selections','license_and_prerequisites','maintenance','upgrade_rollback','downgrade_rejection','active_work_preserved','user_data_preserved','launch_contract','native_ui_capture','cancellation_rollback','user_context')

def guid(s):return '{'+str(uuid.uuid5(NAMESPACE,s)).upper()+'}'
UPGRADE_CODE=guid('insonic/windows_amd64/per-user')
def sha(p):
    with Path(p).open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def digest(b):return hashlib.sha256(b).hexdigest()
def canonical(x):return (json.dumps(x,sort_keys=True,separators=(',',':'))+'\n').encode()
def inventory_bytes(x):return (json.dumps(x,indent=2,sort_keys=True)+'\n').encode()
def ident(prefix,s):return prefix+digest(s.encode())[:28]
def msi_version(v):
    if not re.fullmatch(r'\d+\.\d+\.\d+',v) or any(a>b for a,b in zip(map(int,v.split('.')),[255,255,65535])):raise ValueError('MSI requires a representable three-part release version')
    return v
def product_code(v,upgrade=UPGRADE_CODE):return guid(upgrade+'/'+msi_version(v))
def disposable_runner():
    if os.name!='nt' or os.environ.get('GITHUB_ACTIONS')!='true' or os.environ.get('RUNNER_ENVIRONMENT')!='github-hosted' or os.environ.get('INSONIC_DISPOSABLE_INSTALLER_TEST')!='1':raise RuntimeError('native MSI UI/installation requires acknowledged disposable GitHub-hosted Windows CI')
def relative(s):
    p=PurePosixPath(s)
    if not s or p.is_absolute() or '..' in p.parts or any(c in s for c in '\\\r\n":'):raise ValueError('invalid installer-owned path')
    return p
# s=string reference, i=16-bit biased integer, l=32-bit biased integer.
_LAYOUT={
'Property':'Property:s Value:s',
'Directory':'Directory:s Directory_Parent:s DefaultDir:s',
'Component':'Component:s ComponentId:s Directory_:s Attributes:i Condition:s KeyPath:s',
'Feature':'Feature:s Feature_Parent:s Title:s Description:s Display:i Level:i Directory_:s Attributes:i',
'FeatureComponents':'Feature_:s Component_:s',
'File':'File:s Component_:s FileName:s FileSize:l Version:s Language:s Attributes:i Sequence:i',
'Media':'DiskId:i LastSequence:i DiskPrompt:s Cabinet:s VolumeLabel:s Source:s',
'Registry':'Registry:s Root:i Key:s Name:s Value:s Component_:s',
'Shortcut':'Shortcut:s Directory_:s Name:s Component_:s Target:s Arguments:s Description:s Hotkey:i Icon_:s IconIndex:i ShowCmd:i WkDir:s',
'Upgrade':'UpgradeCode:s VersionMin:s VersionMax:s Language:s Attributes:i Remove:s ActionProperty:s',
'AppSearch':'Property:s Signature_:s',
'Signature':'Signature:s FileName:s MinVersion:s MaxVersion:s MinSize:l MaxSize:l MinDate:l MaxDate:l Languages:s',
'CompLocator':'Signature_:s ComponentId:s Type:i',
'IniLocator':'Signature_:s FileName:s Section:s Key:s Field:i Type:i',
'DrLocator':'Signature_:s Parent:s Path:s Depth:i',
'ActionText':'Action:s Description:s Template:s',
'RegLocator':'Signature_:s Root:i Key:s Name:s Type:i',
'LaunchCondition':'Condition:s Description:s',
'CustomAction':'Action:s Type:i Source:s Target:s',
'InstallExecuteSequence':'Action:s Condition:s Sequence:i',
'InstallUISequence':'Action:s Condition:s Sequence:i',
'Dialog':'Dialog:s HCentering:i VCentering:i Width:i Height:i Attributes:l Title:s Control_First:s Control_Default:s Control_Cancel:s',
'Control':'Dialog_:s Control:s Type:s X:i Y:i Width:i Height:i Attributes:l Property:s Text:s Control_Next:s Help:s',
'ControlEvent':'Dialog_:s Control_:s Event:s Argument:s Condition:s Ordering:i',
'ControlCondition':'Dialog_:s Control_:s Action:s Condition:s',
'CheckBox':'Property:s Value:s',
'TextStyle':'TextStyle:s FaceName:s Size:i Color:l StyleBits:i',
'Error':'Error:i Message:s',
'EventMapping':'Dialog_:s Control_:s Event:s Attribute:s',
'RemoveFile':'FileKey:s Component_:s FileName:s DirProperty:s InstallMode:i',
}
SCHEMA={n:[tuple(c.split(':')) for c in v.split()] for n,v in _LAYOUT.items()}
KEY_COUNTS={'FeatureComponents':2,'Upgrade':5,'Control':2,'ControlEvent':5,'ControlCondition':4,'EventMapping':3}

def effective_feature_local(feature):
    # Native reinstallation reports DEFAULT (5) for an already-local feature.
    # An explicit ABSENT (2) request never inherits its installed state.
    return '(&'+feature+'=3 OR ((&'+feature+'=-1 OR &'+feature+'=5) AND !'+feature+'=3))'

def pe_file_metadata(data):
    """Read exact RT_VERSION fixed-file version and language from PE bytes.

    Source: Microsoft's PE resource directory, VS_VERSIONINFO and File-table
    contracts. No image loading, process execution or platform API is involved.
    """
    if data[:2]!=b'MZ':return None,None
    def unpack(fmt,offset):
        if offset<0 or offset+struct.calcsize(fmt)>len(data):raise ValueError('truncated PE file metadata')
        return struct.unpack_from(fmt,data,offset)
    header=unpack('<I',60)[0]
    if data[header:header+4]!=b'PE\0\0':raise ValueError('invalid PE signature')
    sections=unpack('<H',header+6)[0];optional_size=unpack('<H',header+20)[0];optional=header+24
    magic=unpack('<H',optional)[0];directory_offset={0x10b:96,0x20b:112}.get(magic)
    if directory_offset is None or optional_size<directory_offset+24 or not 0<sections<=96:raise ValueError('invalid PE optional header')
    if unpack('<I',optional+directory_offset-4)[0]<3:return None,None
    resource_rva,resource_size=unpack('<II',optional+directory_offset+16)
    if not resource_rva or not resource_size:return None,None
    section_rows=[unpack('<IIII',optional+optional_size+40*i+8) for i in range(sections)]
    def rva_offset(rva,size):
        matches=[raw+rva-start for virtual,start,length,raw in section_rows if start<=rva and rva+size<=start+length]
        if len(matches)!=1 or matches[0]+size>len(data):raise ValueError('PE resource RVA escapes file')
        return matches[0]
    base=rva_offset(resource_rva,resource_size)
    def resource_entries(offset):
        if offset<0 or offset+16>resource_size:raise ValueError('PE resource directory escapes section')
        named,ids=unpack('<HH',base+offset+12);count=named+ids
        if count>4096 or offset+16+count*8>resource_size:raise ValueError('invalid PE resource entry count')
        return [unpack('<II',base+offset+16+8*i) for i in range(count)]
    versions=[];languages=set()
    for resource_type,name_dir in resource_entries(0):
        if resource_type!=16:continue
        if not name_dir&0x80000000:raise ValueError('invalid PE version-name directory')
        for _,language_dir in resource_entries(name_dir&0x7fffffff):
            if not language_dir&0x80000000:raise ValueError('invalid PE version-language directory')
            for language,leaf in resource_entries(language_dir&0x7fffffff):
                if leaf&0x80000000 or leaf+16>resource_size or language&0x80000000:raise ValueError('invalid PE version leaf')
                value_rva,value_size,_,_=unpack('<IIII',base+leaf);value_offset=rva_offset(value_rva,value_size)
                value=data[value_offset:value_offset+value_size]
                if len(value)<6:raise ValueError('truncated PE version resource')
                length,value_length,value_type=struct.unpack_from('<HHH',value);key='VS_VERSION_INFO\0'.encode('utf-16le')
                if length>len(value) or value[6:6+len(key)]!=key or value_type!=0:raise ValueError('invalid PE fixed version header')
                fixed=(6+len(key)+3)&~3
                if value_length!=52 or fixed+52>length or struct.unpack_from('<I',value,fixed)[0]!=0xfeef04bd:raise ValueError('invalid PE fixed version value')
                major,minor=struct.unpack_from('<II',value,fixed+8)
                versions.append('.'.join(map(str,(major>>16,major&65535,minor>>16,minor&65535))));languages.add(language)
    if not versions:return None,None
    if len(set(versions))!=1:raise ValueError('conflicting PE fixed file versions')
    return versions[0],','.join(str(language) for language in sorted(languages))

def installer_model(inv,inv_sha,license_text,upgrade=UPGRADE_CODE,version=None):
    version=msi_version(version or inv['schema_version'])
    core,desktop,cli=(effective_feature_local(feature) for feature in ('Core','Desktop','CLI'))
    if inv.get('platform')!='windows_amd64' or inv.get('variant')!='desktop':raise ValueError('MSI requires exact Windows desktop payload')
    files=[dict(x) for x in inv['files']]+[{'path':'package-inventory.json','sha256':inv_sha,'size_bytes':len(inventory_bytes(inv))},{'path':OWNERSHIP,'sha256':digest(OWNERSHIP_BYTES),'size_bytes':len(OWNERSHIP_BYTES)}]
    if len({f['path'].casefold() for f in files})!=len(files):raise ValueError('duplicate/case-colliding MSI paths')
    if not {'insonic.exe','insonic-desktop.exe','insonic-launcher.exe','insonic-session.cmd','LICENSE','NOTICE','insonic-companions.json'}<={f['path'] for f in files}:raise ValueError('missing MSI runtime/client authority')
    tables={n:[] for n in SCHEMA}
    def add(t,*r):tables[t].append(list(r))
    props={'ProductCode':product_code(version,upgrade),'UpgradeCode':upgrade,'ProductVersion':version,'ProductLanguage':'1033','Manufacturer':'ShruggieTech','ProductName':'insonic','DefaultUIFont':'Normal','INSTALLLEVEL':'1','ARPCOMMENTS':'Voice archive CLI and desktop application','ARPHELPLINK':'https://insonic.io/docs/','ARPURLINFOABOUT':'https://insonic.io/','ARPPRODUCTICON':'InsonicIcon','MSIRESTARTMANAGERCONTROL':'Disable','REBOOT':'ReallySuppress','SecureCustomProperties':'OLDPRODUCTS;NEWPRODUCTS;INSTALLDIR;SAVED_INSTALLDIR;LICENSE_ACCEPTED;START_MENU;DESKTOP_SHORTCUT;CLIENT_DESKTOP;CLIENT_CLI','CLIENT_DESKTOP':'1','CLIENT_CLI':'1','START_MENU':'1','DESKTOP_SHORTCUT':'1','LAUNCH':'1'}
    for k,v in props.items():add('Property',k,v)
    for r in [('TARGETDIR',None,'SourceDir'),('LocalAppDataFolder','TARGETDIR','.'),('Programs','LocalAppDataFolder','Programs'),('INSTALLDIR','Programs','insonic'),('ProgramMenuFolder','TARGETDIR','.'),('InsonicMenu','ProgramMenuFolder','insonic'),('DesktopFolder','TARGETDIR','.')]:add('Directory',*r)
    dirs={'':'INSTALLDIR'}
    for f in sorted(files,key=lambda f:f['path']):
        p=relative(f['path'])
        for parent in reversed(p.parents):
            if str(parent)=='.':continue
            s=parent.as_posix()
            if s not in dirs:
                dirs[s]=ident('d',s);parent_key='' if str(parent.parent)=='.' else parent.parent.as_posix();add('Directory',dirs[s],dirs[parent_key],'D'+digest(s.encode())[:7]+'|'+parent.name)
    payload=[]
    for seq,f in enumerate(sorted(files,key=lambda f:f['path']),1):
        p=relative(f['path']);fid=ident('f',f['path']);cid=ident('c',f['path']);parent='' if str(p.parent)=='.' else p.parent.as_posix();feature='Desktop' if f['path']=='insonic-desktop.exe' else 'Core'
        add('Component',cid,guid('component/'+f['path']),dirs[parent],256,None,fid);add('FeatureComponents',feature,cid);add('File',fid,cid,'F'+digest(f['path'].encode())[:7]+'|'+p.name,f['size_bytes'],None,None,512,seq);payload.append({**f,'file_id':fid,'feature':feature})
    add('Media',1,len(files),None,'#payload.cab',None,None)
    for r in [('Core',None,'Shared runtime','Required CLI/runtime, media tools, help, licenses and sources.',0,1,'INSTALLDIR',16),('Desktop',None,'Desktop application','Requires shared CLI/runtime and Microsoft WebView2.',1,1,'INSTALLDIR',0),('CLI',None,'CLI integrations','Persistent CLI help/setup session. Shared bytes remain with Desktop-only.',2,1,'INSTALLDIR',0)]:add('Feature',*r)
    key='Software\\ShruggieTech\\insonic'
    for component,feature,name,value,condition in [('InstallRegistration','Core','InstallDir','[INSTALLDIR]',None),('CliIntegration','CLI','CLI','#1',None),('GuiMenu','Desktop','GuiMenu','#1','START_MENU=1'),('CliMenu','CLI','CliMenu','#1','START_MENU=1'),('GuiDesktop','Desktop','GuiDesktop','#1','DESKTOP_SHORTCUT=1')]:
        rid='Reg'+component;add('Component',component,guid('integration/'+component),'INSTALLDIR',324,condition,rid);add('Registry',rid,1,key,name,value,component);add('FeatureComponents',feature,component)
    for name,value in [('StartMenu','[START_MENU]'),('DesktopShortcut','[DESKTOP_SHORTCUT]')]:add('Registry','Reg'+name,1,key,name,value,'InstallRegistration')
    for sid,directory,name,component,args in [('GuiStart','InsonicMenu','insonic','GuiMenu','--desktop'),('CliStart','InsonicMenu','insonic CLI','CliMenu','--cli'),('GuiDesktop','DesktopFolder','insonic','GuiDesktop','--desktop')]:add('Shortcut',sid,directory,'S'+digest((sid+'/'+name).encode())[:7]+'|'+name,component,'[INSTALLDIR]insonic-launcher.exe',args,'Open '+name,None,'InsonicIcon',0,1,'INSTALLDIR')
    add('RemoveFile','RemoveMenu','GuiMenu',None,'InsonicMenu',2);add('RemoveFile','RemoveMenuCli','CliMenu',None,'InsonicMenu',2)
    add('Upgrade',upgrade,None,version,None,1,None,'OLDPRODUCTS');add('Upgrade',upgrade,version,None,None,2,None,'NEWPRODUCTS')
    for prop,sig,hive,reg,name,kind in [('OS_BUILD','WindowsBuild',2,'SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion','CurrentBuildNumber',18),('SAVED_INSTALLDIR','SavedInstall',1,key,'InstallDir',18),('START_MENU','SavedMenu',1,key,'StartMenu',18),('DESKTOP_SHORTCUT','SavedDesktop',1,key,'DesktopShortcut',18),('VC_RUNTIME','VCRuntime',2,'SOFTWARE\\Microsoft\\VisualStudio\\14.0\\VC\\Runtimes\\x64','Installed',18),('WEBVIEW_MACHINE','WebviewMachine',2,'SOFTWARE\\WOW6432Node\\Microsoft\\EdgeUpdate\\Clients\\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}','pv',18),('WEBVIEW_USER','WebviewUser',1,'Software\\Microsoft\\EdgeUpdate\\Clients\\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}','pv',18)]:add('AppSearch',prop,sig);add('RegLocator',sig,hive,reg,name,kind)
    for condition,message in [('NOT ALLUSERS','This is a per-user installer. Remove ALLUSERS and choose a writable user directory.'),('VersionNT64 AND OS_BUILD >= 10240','insonic requires Windows 10 or later, x86-64.'),('NOT NEWPRODUCTS','A newer insonic version is installed. Downgrades are not supported.'),('Installed OR LICENSE_ACCEPTED=1','Read and accept Apache License 2.0. For unattended installation set LICENSE_ACCEPTED=1.'),('REMOVE="ALL" OR '+core,'Shared runtime is mandatory. Include Core with Desktop/CLI selections.'),('REMOVE="ALL" OR ('+desktop+' OR '+cli+')','Select Desktop, CLI integrations, or both.'),('REMOVE="ALL" OR VC_RUNTIME="#1"','Install Microsoft Visual C++ v14 x64 Redistributable, then retry: https://aka.ms/vc14/vc_redist.x64.exe'),('REMOVE="ALL" OR NOT '+desktop+' OR (WEBVIEW_MACHINE AND WEBVIEW_MACHINE<>"0.0.0.0") OR (WEBVIEW_USER AND WEBVIEW_USER<>"0.0.0.0")','Desktop requires WebView2 Evergreen Runtime. For offline use install the x64 standalone installer: https://developer.microsoft.com/en-us/microsoft-edge/webview2/')]:add('LaunchCondition',condition,message)
    add('CustomAction','Preflight',2,'Launcher','--preflight "[INSTALLDIR]." both');add('CustomAction','LaunchDesktop',18,ident('f','insonic-launcher.exe'),'--desktop');add('CustomAction','LaunchCLI',18,ident('f','insonic-launcher.exe'),'--cli')
    for action,path,component in [('CheckGuiMenu','[InsonicMenu]insonic.lnk','GuiMenu'),('CheckCliMenu','[InsonicMenu]insonic CLI.lnk','CliMenu'),('CheckGuiDesktop','[DesktopFolder]insonic.lnk','GuiDesktop')]:add('CustomAction',action,2,'Launcher','--check-shortcut "'+path+'" '+component)
    add('CustomAction','RestoreInstallDir',51,'INSTALLDIR','[SAVED_INSTALLDIR]')
    add('CustomAction','NormalizeStartMenu',51,'START_MENU','0');add('CustomAction','NormalizeDesktopShortcut',51,'DESKTOP_SHORTCUT','0')
    for action,prop,value in [('ApplySavedDesktop','CLIENT_DESKTOP','1'),('ClearSavedDesktop','CLIENT_DESKTOP',None),('ApplySavedCLI','CLIENT_CLI','1'),('ClearSavedCLI','CLIENT_CLI',None)]:add('CustomAction',action,51,prop,value)
    for r in [('FindRelatedProducts',None,25),('AppSearch',None,50),('RestoreInstallDir','SAVED_INSTALLDIR AND NOT INSTALLDIR',60),('NormalizeStartMenu','NOT START_MENU',700),('NormalizeDesktopShortcut','NOT DESKTOP_SHORTCUT',701),('CostInitialize',None,800),('FileCost',None,900),('CostFinalize',None,1000),('MigrateFeatureStates',None,1100),('LaunchConditions',None,1200),('Preflight',None,1300),('CheckGuiMenu','START_MENU=1 AND '+desktop,1310),('CheckCliMenu','START_MENU=1 AND '+cli,1311),('CheckGuiDesktop','DESKTOP_SHORTCUT=1 AND '+desktop,1312),('InstallValidate',None,1400),('InstallInitialize',None,1500),('RemoveExistingProducts','OLDPRODUCTS',1510),('ProcessComponents',None,1600),('UnpublishFeatures',None,1800),('RemoveShortcuts',None,3200),('RemoveRegistryValues',None,3300),('RemoveFiles',None,3500),('InstallFiles',None,4000),('CreateShortcuts',None,4500),('WriteRegistryValues',None,5000),('RegisterUser',None,6000),('RegisterProduct',None,6100),('PublishFeatures',None,6300),('PublishProduct',None,6400),('InstallFinalize',None,6600)]:add('InstallExecuteSequence',*r)
    for r in [('FindRelatedProducts',None,25),('AppSearch',None,50),('RestoreInstallDir','SAVED_INSTALLDIR AND NOT INSTALLDIR',60),('CostInitialize',None,800),('FileCost',None,900),('CostFinalize',None,1000),('MigrateFeatureStates',None,1100),('ApplySavedDesktop','(Installed OR OLDPRODUCTS) AND '+desktop,1110),('ClearSavedDesktop','(Installed OR OLDPRODUCTS) AND NOT '+desktop,1111),('ApplySavedCLI','(Installed OR OLDPRODUCTS) AND '+cli,1112),('ClearSavedCLI','(Installed OR OLDPRODUCTS) AND NOT '+cli,1113),('Welcome','NOT Installed',1200),('Maintenance','Installed',1201),('ExecuteAction',None,1300),('Finish',None,-1),('Cancelled',None,-2),('FatalError',None,-3)]:add('InstallUISequence',*r)
    add('TextStyle','Normal','Segoe UI',10,0,0);add('TextStyle','Heading','Segoe UI',14,0,1)
    add('Error',1722,'Setup could not continue. Close insonic clients, finish/cancel jobs, then wait for the idle runtime to exit. Choose an empty writable directory for a fresh installation. If a shortcut already exists, deselect its Start menu or Desktop integration, or move that shortcut yourself. Your library/settings were not removed. Action: [2].')
    for prop in ['CLIENT_DESKTOP','CLIENT_CLI','START_MENU','DESKTOP_SHORTCUT','LAUNCH','LICENSE_ACCEPTED']:add('CheckBox',prop,'1')
    def control(d,n,t,x,y,w,h,text=None,prop=None,attr=3,nxt=None):add('Control',d,n,t,x,y,w,h,attr,prop,text,nxt,None)
    def event(d,n,e,arg,condition='1',order=1):add('ControlEvent',d,n,e,arg,condition,order)
    def page(d,title,body,back=None,next_page=None):
        add('Dialog',d,50,50,420,290,3,'insonic Setup','Next','Next','Cancel')
        control(d,'Title','Text',18,16,384,30,'{&Heading}'+title);control(d,'Body','Text',18,54,384,40,body)
        control(d,'Next','PushButton',261,250,65,23,'Next >',nxt='Back' if back else 'Cancel');control(d,'Cancel','PushButton',335,250,65,23,'Cancel',nxt='Next');event(d,'Cancel','EndDialog','Exit')
        if back:control(d,'Back','PushButton',187,250,65,23,'< Back',nxt='Cancel');event(d,'Back','NewDialog',back)
        if next_page:event(d,'Next','NewDialog',next_page,order=10)
    page('Welcome','Install insonic','Install for this Windows account. No download or restart is performed.',next_page='License')
    page('License','Apache License 2.0','Read the project license. Acceptance is initially unchecked.',back='Welcome',next_page='Choices')
    rtf='{\\rtf1\\ansi '+license_text.replace('\\','\\\\').replace('{','\\{').replace('}','\\}').replace('\n','\\par ')+'}'
    control('License','LicenseText','ScrollableText',18,96,384,108,rtf);control('License','Accept','CheckBox',18,217,384,20,'I accept the Apache License 2.0','LICENSE_ACCEPTED',nxt='Next')
    add('ControlCondition','License','Next','Disable','NOT LICENSE_ACCEPTED');add('ControlCondition','License','Next','Enable','LICENSE_ACCEPTED=1')
    page('Choices','Select clients and shortcuts','Desktop retains the shared CLI/runtime. CLI-only omits the GUI and WebView2 requirement.',back='License',next_page='Location')
    for n,p,text,y in [('Desktop','CLIENT_DESKTOP','Desktop application',98),('CLI','CLIENT_CLI','CLI integrations (persistent help/setup session)',128),('Menu','START_MENU','Create Start menu shortcuts',168),('Shortcut','DESKTOP_SHORTCUT','Create Desktop shortcut for the desktop application',198)]:control('Choices',n,'CheckBox',18,y,384,20,text,p,nxt='Next')
    add('ControlCondition','Choices','Next','Disable','NOT CLIENT_DESKTOP AND NOT CLIENT_CLI');add('ControlCondition','Choices','Next','Enable','CLIENT_DESKTOP=1 OR CLIENT_CLI=1')
    for feature,prop,order in [('Desktop','CLIENT_DESKTOP',1),('CLI','CLIENT_CLI',3)]:event('Choices','Next','AddLocal',feature,prop+'=1',order);event('Choices','Next','Remove',feature,'NOT '+prop,order+1)
    page('Location','Choose program directory','Program files go here. Libraries, models, credentials and settings remain outside installer ownership.',back='Choices',next_page='Ready')
    control('Location','Directory','PathEdit',18,108,384,25,prop='INSTALLDIR',attr=3,nxt='Browse');event('Location','Next','SetTargetPath','INSTALLDIR')
    control('Location','Browse','PushButton',282,146,120,24,'Browse...',nxt='Next');event('Location','Browse','SpawnDialog','Browse')
    add('ControlCondition','Location','Directory','Disable','Installed OR OLDPRODUCTS');add('ControlCondition','Location','Browse','Disable','Installed OR OLDPRODUCTS')
    page('Browse','Browse program directory','Select a directory or enter the complete new program path.')
    control('Browse','Folders','DirectoryCombo',18,96,384,20,prop='INSTALLDIR',nxt='Next')
    control('Browse','List','DirectoryList',18,122,384,88,prop='INSTALLDIR',nxt='Next')
    control('Browse','Path','PathEdit',18,215,384,24,prop='INSTALLDIR',nxt='Next')
    event('Browse','Next','SetTargetPath','INSTALLDIR');event('Browse','Next','EndDialog','Return',order=2)
    for r in tables['ControlEvent']:
        if r[:2]==['Browse','Cancel']:r[3]='Return'
    page('Ready','Ready to install','Setup verifies capacity and prerequisites. Install missing Microsoft dependencies separately from official downloads.',back='Location');event('Ready','Next','DoAction','NormalizeStartMenu','NOT START_MENU',1);event('Ready','Next','DoAction','NormalizeDesktopShortcut','NOT DESKTOP_SHORTCUT',2);event('Ready','Next','[REINSTALL]','Core','Installed AND NOT REINSTALL',3);event('Ready','Next','EndDialog','Return',order=4)
    page('Maintenance','Change, repair or remove','Close clients and finish/cancel work first. Libraries, models, credentials and settings are preserved.',next_page='Choices')
    control('Maintenance','Repair','PushButton',18,120,115,25,'Repair',nxt='Remove');event('Maintenance','Repair','[REINSTALL]','ALL');event('Maintenance','Repair','[REINSTALLMODE]','vomus',order=2);event('Maintenance','Repair','EndDialog','Return',order=3)
    control('Maintenance','Remove','PushButton',145,120,115,25,'Remove',nxt='Next');event('Maintenance','Remove','[REMOVE]','ALL');event('Maintenance','Remove','Remove','ALL',order=2);event('Maintenance','Remove','EndDialog','Return',order=3)
    page('Finish','insonic installed','Open the selected client from its program directory or optional shortcuts. First-run setup can be deferred.')
    control('Finish','Launch','CheckBox',18,118,384,22,'Launch insonic','LAUNCH',nxt='Next');control('Finish','CliLaunch','CheckBox',18,118,384,22,'Open insonic CLI','LAUNCH',attr=2,nxt='Next')
    add('ControlCondition','Finish','Launch','Hide','NOT '+desktop+' OR Installed OR OLDPRODUCTS');add('ControlCondition','Finish','Launch','Show',desktop+' AND NOT Installed AND NOT OLDPRODUCTS');add('ControlCondition','Finish','CliLaunch','Show','NOT '+desktop+' AND '+cli+' AND NOT Installed AND NOT OLDPRODUCTS')
    event('Finish','Next','DoAction','LaunchDesktop','LAUNCH=1 AND '+desktop+' AND NOT Installed AND NOT OLDPRODUCTS AND UILevel=5');event('Finish','Next','DoAction','LaunchCLI','LAUNCH=1 AND NOT '+desktop+' AND '+cli+' AND NOT Installed AND NOT OLDPRODUCTS AND UILevel=5',2);event('Finish','Next','EndDialog','Return',order=3)
    for d,title,body in [('Cancelled','Setup cancelled','No selected installation was accepted.'),('FatalError','Setup could not finish','Read the correction message/log. Close clients and install missing prerequisites, then retry.')]:page(d,title,body);event(d,'Next','EndDialog','Return')
    for r in tables['Control']:
        if r[:2]==['Ready','Next']:r[9]='Install'
        if r[:2]==['Finish','Next']:r[9]='Finish'
        if r[:2]==['Browse','Next']:r[9]='OK'
    # One tab-order cycle, including every focusable authored control.
    for dialog in tables['Dialog']:
        inputs=[r for r in tables['Control'] if r[0]==dialog[0] and r[2] in ('CheckBox','PathEdit','ScrollableText','DirectoryCombo','DirectoryList')]
        buttons=[r for r in tables['Control'] if r[0]==dialog[0] and r[2]=='PushButton']
        cycle=inputs+buttons
        for i,r in enumerate(cycle):r[10]=cycle[(i+1)%len(cycle)][1]
        dialog[7]=inputs[0][1] if inputs else 'Next'
        if dialog[0] in ('Finish','Location'):dialog[7]='Next'
    model={'kind':'windows-installer-model','version':version,'product_code':product_code(version,upgrade),'upgrade_code':upgrade,'scope':'per-user','payload':payload,'tables':tables,'license_sha256':digest(license_text.encode())}
    validate_model(model);return model

def validate_model(model):
    tables=model['tables'];files={r[0]:r for r in tables['File']};components={r[0]:r for r in tables['Component']};controls={(r[0],r[1]) for r in tables['Control']}
    if len(files)!=len(tables['File']) or len(components)!=len(tables['Component']):raise ValueError('duplicate MSI identity')
    for r in files.values():
        if r[1] not in components or components[r[1]][5]!=r[0]:raise ValueError('MSI file/key path mismatch')
    for r in tables['Dialog']:
        if any((r[0],c) not in controls for c in r[7:10] if c):raise ValueError('unknown dialog control')
    for r in tables['ControlEvent']:
        if (r[0],r[1]) not in controls:raise ValueError('unknown event control')
    seq={r[0]:r[2] for r in tables['InstallExecuteSequence']}
    if not seq['InstallInitialize']<seq['RemoveExistingProducts']<seq['InstallFinalize']:raise ValueError('upgrade removal must be transactional')
    if 'LICENSE_ACCEPTED' in dict(tables['Property']):raise ValueError('license must initially be unchecked')

class NativeDatabase:
    """Typed msi.dll database APIs only; no _msi/msilib dependency or UI."""
    def __init__(self,path,create=False,write=False):
        if os.name!='nt':raise RuntimeError('native MSI authoring requires Windows')
        self.dll=ctypes.WinDLL('msi',use_last_error=True)
        methods={'MsiOpenDatabaseW':[ctypes.c_wchar_p,ctypes.c_void_p,ctypes.POINTER(ctypes.c_uint)],'MsiDatabaseOpenViewW':[ctypes.c_uint,ctypes.c_wchar_p,ctypes.POINTER(ctypes.c_uint)],'MsiViewExecute':[ctypes.c_uint,ctypes.c_uint],'MsiCreateRecord':[ctypes.c_uint],'MsiRecordSetStringW':[ctypes.c_uint,ctypes.c_uint,ctypes.c_wchar_p],'MsiRecordSetInteger':[ctypes.c_uint,ctypes.c_uint,ctypes.c_int],'MsiRecordSetStreamW':[ctypes.c_uint,ctypes.c_uint,ctypes.c_wchar_p],'MsiCloseHandle':[ctypes.c_uint],'MsiDatabaseCommit':[ctypes.c_uint],'MsiGetSummaryInformationW':[ctypes.c_uint,ctypes.c_wchar_p,ctypes.c_uint,ctypes.POINTER(ctypes.c_uint)],'MsiSummaryInfoSetPropertyW':[ctypes.c_uint,ctypes.c_uint,ctypes.c_uint,ctypes.c_int,ctypes.c_void_p,ctypes.c_wchar_p],'MsiSummaryInfoPersist':[ctypes.c_uint]}
        for name,args in methods.items():f=getattr(self.dll,name);f.argtypes=args;f.restype=ctypes.c_uint
        h=ctypes.c_uint();self.check(self.dll.MsiOpenDatabaseW(str(path),ctypes.c_void_p(3 if create else 1 if write else 0),ctypes.byref(h)));self.handle=h.value
    def check(self,rc):
        if rc:raise RuntimeError('Windows Installer database API failed: '+str(rc))
    def close(self):
        if self.handle:self.dll.MsiCloseHandle(self.handle);self.handle=0
    def execute(self,query,values=None,streams=()):
        view=ctypes.c_uint();record=0;self.check(self.dll.MsiDatabaseOpenViewW(self.handle,query,ctypes.byref(view)))
        try:
            if values is not None:
                record=self.dll.MsiCreateRecord(len(values))
                for i,v in enumerate(values,1):
                    if v is None:continue
                    method=self.dll.MsiRecordSetStreamW if i in streams else self.dll.MsiRecordSetInteger if isinstance(v,int) else self.dll.MsiRecordSetStringW
                    self.check(method(record,i,v if isinstance(v,int) else str(v)))
            self.check(self.dll.MsiViewExecute(view,record))
        finally:
            if record:self.dll.MsiCloseHandle(record)
            self.dll.MsiCloseHandle(view)
    def summary(self,package):
        h=ctypes.c_uint();self.check(self.dll.MsiGetSummaryInformationW(self.handle,None,20,ctypes.byref(h)))
        try:
            for prop,v in {1:1252,2:'Installation Database',3:'insonic',4:'ShruggieTech',7:'x64;1033',9:package,14:500,15:10,18:'insonic native MSI authoring'}.items():self.check(self.dll.MsiSummaryInfoSetPropertyW(h,prop,3 if isinstance(v,int) else 30,v if isinstance(v,int) else 0,None,None if isinstance(v,int) else v))
            self.check(self.dll.MsiSummaryInfoPersist(h))
        finally:self.dll.MsiCloseHandle(h)

def quote(s):return chr(96)+s+chr(96)
def author(path,payload_root,model,fixture_failure=False):
    if os.name!='nt':raise RuntimeError('native MSI authoring requires Windows')
    path=Path(path);payload_root=Path(payload_root)
    if path.exists():path.unlink()
    with tempfile.TemporaryDirectory(prefix='insonic-msi-author-') as temp:
        temp=Path(temp);cab=temp/'payload.cab';marker=temp/OWNERSHIP;marker.write_bytes(OWNERSHIP_BYTES)
        directives=['.OPTION EXPLICIT','.Set Cabinet=ON','.Set Compress=ON','.Set CompressionType=MSZIP','.Set CabinetNameTemplate=payload.cab','.Set DiskDirectoryTemplate='+str(temp),'.Set MaxDiskSize=0','.Set MaxCabinetSize=0','.Set GenerateInf=ON','.Set InfFileName='+str(temp/'cabinet.inf'),'.Set RptFileName='+str(temp/'cabinet.rpt')]
        metadata={}
        for entry in model['payload']:
            p=marker if entry['path']==OWNERSHIP else payload_root.joinpath(*relative(entry['path']).parts)
            if not p.is_file() or p.is_symlink() or p.stat().st_size!=entry['size_bytes'] or sha(p)!=entry['sha256']:raise ValueError('MSI input differs from exact payload inventory: '+entry['path'])
            metadata[entry['file_id']]=pe_file_metadata(p.read_bytes())
            directives.append('"'+str(p)+'" '+entry['file_id'])
        ddf=temp/'payload.ddf';ddf.write_text('\n'.join(directives)+'\n',encoding='utf-8',newline='\n')
        makecab=Path(os.environ.get('SystemRoot','C:/Windows'))/'System32/makecab.exe'
        completed=subprocess.run([str(makecab),'/F',str(ddf)],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=180)
        if completed.returncode:raise RuntimeError('makecab failed: '+completed.stderr.decode(errors='replace')[-3000:]+completed.stdout.decode(errors='replace')[-3000:])
        db=NativeDatabase(path,create=True)
        try:
            for name,columns in SCHEMA.items():
                keys=KEY_COUNTS.get(name,1);decl=[]
                for i,(col,typ) in enumerate(columns):
                    nullable=name=='Upgrade' and i in (1,2,3)
                    decl.append(quote(col)+' '+('CHAR(72)' if typ=='s' and i<keys else 'CHAR(0)' if typ=='s' else 'SHORT' if typ=='i' else 'LONG')+(' NOT NULL' if i<keys and not nullable else ''))
                db.execute('CREATE TABLE '+quote(name)+' ('+','.join(decl)+' PRIMARY KEY '+','.join(quote(c[0]) for c in columns[:keys])+')')
                for original in model['tables'][name]:
                    values=list(original)
                    if name=='File':values[4:6]=metadata[values[0]]
                    db.execute('INSERT INTO '+quote(name)+' ('+','.join(quote(c[0]) for c in columns)+') VALUES ('+','.join('?' for _ in columns)+')',values)
            for table,entry,p in [('Binary','Launcher',payload_root/'insonic-launcher.exe'),('Icon','InsonicIcon',ROOT/ICON)]:
                db.execute('CREATE TABLE '+quote(table)+' ('+quote('Name')+' CHAR(72) NOT NULL, '+quote('Data')+' OBJECT NOT NULL PRIMARY KEY '+quote('Name')+')')
                db.execute('INSERT INTO '+quote(table)+' ('+quote('Name')+','+quote('Data')+') VALUES (?,?)',[entry,p],streams=(2,))
            db.execute('INSERT INTO '+quote('_Streams')+' ('+quote('Name')+','+quote('Data')+') VALUES (?,?)',['payload.cab',cab],streams=(2,))
            if fixture_failure:
                disposable_runner();db.execute('INSERT INTO '+quote('CustomAction')+' ('+','.join(quote(c[0]) for c in SCHEMA['CustomAction'])+') VALUES (?,?,?,?)',['FixtureFailure',19,None,'Intentional transactional fixture failure'])
                db.execute('INSERT INTO '+quote('InstallExecuteSequence')+' ('+','.join(quote(c[0]) for c in SCHEMA['InstallExecuteSequence'])+') VALUES (?,?,?)',['FixtureFailure','1',4100])
            db.summary(guid('package/'+digest(canonical({'model':model,'file_metadata':metadata}))));db.check(db.dll.MsiDatabaseCommit(db.handle))
        finally:db.close()
    verify_model_bytes(path,model,fixture_failure=fixture_failure)
    return path

ALPHABET='0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz._'
def decode_stream_name(name):
    out=''
    for ch in name:
        c=ord(ch)
        if c==0x4840:continue
        if 0x3800<=c<0x4800:out+=ALPHABET[(c-0x3800)&63]+ALPHABET[((c-0x3800)>>6)&63]
        elif 0x4800<=c<0x4840:out+=ALPHABET[c-0x4800]
        else:out+=ch
    return out
class CompoundFile:
    """Bounded read-only CFB parser; never executes or extracts artifact bytes."""
    def __init__(self,path):
        self.data=Path(path).read_bytes();h=self.data[:512]
        if h[:8]!=bytes.fromhex('d0cf11e0a1b11ae1') or len(h)!=512:raise ValueError('artifact is not an MSI compound file')
        self.sector=1<<struct.unpack_from('<H',h,30)[0];self.mini=1<<struct.unpack_from('<H',h,32)[0]
        if self.sector not in (512,4096) or self.mini!=64:raise ValueError('invalid compound sector size')
        self.cutoff=struct.unpack_from('<I',h,56)[0];ids=list(struct.unpack_from('<109I',h,76));difat=struct.unpack_from('<I',h,68)[0];count=struct.unpack_from('<I',h,72)[0]
        if count>len(self.data)//self.sector:raise ValueError('invalid DIFAT bound')
        seen=set()
        for _ in range(count):
            if difat in seen:raise ValueError('cyclic DIFAT')
            seen.add(difat);v=struct.unpack('<'+str(self.sector//4)+'I',self.sec(difat));ids.extend(v[:-1]);difat=v[-1]
        ids=[i for i in ids if i<0xfffffffa]
        if len(ids)!=struct.unpack_from('<I',h,44)[0]:raise ValueError('incomplete compound FAT')
        self.fat=[]
        for i in ids:self.fat.extend(struct.unpack('<'+str(self.sector//4)+'I',self.sec(i)))
        raw=self.chain(struct.unpack_from('<I',h,48)[0],self.fat,self.sec);entries=[]
        for i in range(0,len(raw),128):
            b=raw[i:i+128]
            if len(b)!=128 or not b[66]:continue
            n=struct.unpack_from('<H',b,64)[0]
            if n<2 or n>64 or n%2:raise ValueError('invalid compound directory name')
            name=b[:n-2].decode('utf-16le');start=struct.unpack_from('<I',b,116)[0];size=struct.unpack_from('<Q',b,120)[0]
            if size>len(self.data):raise ValueError('compound stream exceeds artifact')
            entries.append((name,b[66],start,size))
        roots=[e for e in entries if e[1]==5]
        if len(roots)!=1:raise ValueError('invalid compound root')
        self.minidata=self.chain(roots[0][2],self.fat,self.sec)[:roots[0][3]]
        start=struct.unpack_from('<I',h,60)[0];mini=self.chain(start,self.fat,self.sec) if start<0xfffffffa else b'';minifat=list(struct.unpack('<'+str(len(mini)//4)+'I',mini));self.streams={}
        for name,kind,start,size in entries:
            if kind!=2:continue
            name=decode_stream_name(name)
            if name in self.streams:raise ValueError('duplicate compound stream')
            data=self.chain(start,minifat,lambda i:self.minidata[i*64:(i+1)*64]) if size<self.cutoff else self.chain(start,self.fat,self.sec)
            if len(data)<size:raise ValueError('truncated compound stream')
            self.streams[name]=data[:size]
    def sec(self,i):
        start=(i+1)*self.sector;b=self.data[start:start+self.sector]
        if len(b)!=self.sector:raise ValueError('compound sector escapes artifact')
        return b
    def chain(self,start,fat,read):
        out=[];seen=set()
        while start<0xfffffffa:
            if start>=len(fat) or start in seen:raise ValueError('invalid compound chain')
            seen.add(start);out.append(read(start));start=fat[start]
        if start not in (0xfffffffe,0xffffffff):raise ValueError('invalid compound chain terminator')
        return b''.join(out)

def table_rows(streams,name):
    pool=streams['_StringPool'];data=streams['_StringData'];codepage=struct.unpack_from('<I',pool)[0]&0xffff;strings=[None];offset=0
    for i in range(4,len(pool),4):
        size,refs=struct.unpack_from('<HH',pool,i)
        if not size and refs:raise ValueError('oversized MSI string unsupported')
        strings.append(data[offset:offset+size].decode('utf-8' if codepage==65001 else 'cp'+str(codepage or 1252)));offset+=size
    if offset!=len(data):raise ValueError('MSI string pool differs from string data')
    raw=streams.get(name,b'');columns=[('Name','s')] if name=='_Tables' else SCHEMA[name];width=sum(4 if typ=='l' else 2 for _,typ in columns)
    if len(raw)%width:raise ValueError('invalid MSI table length')
    count=len(raw)//width;rows=[[None]*len(columns) for _ in range(count)];cursor=0
    for col,(_,typ) in enumerate(columns):
        size=4 if typ=='l' else 2
        for row in rows:
            v=int.from_bytes(raw[cursor:cursor+size],'little');cursor+=size
            if typ=='s':
                if v>=len(strings):raise ValueError('invalid MSI string reference')
                row[col]=strings[v]
            else:row[col]=None if not v else v-(1<<(size*8-1))
    return rows

def cabinet_members(data,metadata=False):
    """Independently decompress single-cabinet MSZIP files and hash each member."""
    if data[:4]!=b'MSCF' or len(data)<36:raise ValueError('missing embedded cabinet')
    size=struct.unpack_from('<I',data,8)[0];file_offset=struct.unpack_from('<I',data,16)[0];folders,files,flags=struct.unpack_from('<HHH',data,26)
    if size!=len(data) or flags or not folders:raise ValueError('unsupported split/reserved cabinet')
    folder_rows=[struct.unpack_from('<IHH',data,36+8*i) for i in range(folders)];contents=[]
    for start,blocks,compression in folder_rows:
        raw=bytearray();cursor=start;dictionary=b''
        for _ in range(blocks):
            if cursor+8>len(data):raise ValueError('truncated cabinet block')
            _,packed,length=struct.unpack_from('<IHH',data,cursor);cursor+=8;chunk=data[cursor:cursor+packed];cursor+=packed
            if len(chunk)!=packed or length>32768:raise ValueError('invalid cabinet block length')
            if compression==0:value=chunk
            elif compression==1:
                if chunk[:2]!=b'CK':raise ValueError('invalid MSZIP marker')
                d=zlib.decompressobj(-15,zdict=dictionary) if dictionary else zlib.decompressobj(-15);value=d.decompress(chunk[2:],32769)
                if d.unconsumed_tail or not d.eof or d.unused_data:raise ValueError('cabinet deflate block exceeds bound')
                dictionary=(dictionary+value)[-32768:]
            else:raise ValueError('unsupported cabinet compression')
            if len(value)!=length:raise ValueError('cabinet decompression size differs')
            raw.extend(value)
        contents.append(raw)
    cursor=file_offset;members={}
    for _ in range(files):
        if cursor+16>len(data):raise ValueError('truncated cabinet file')
        length,offset,folder,_,_,_=struct.unpack_from('<IIHHHH',data,cursor);cursor+=16;end=data.find(b'\0',cursor)
        if end<0 or end-cursor>256:raise ValueError('invalid cabinet file name')
        name=data[cursor:end].decode('ascii');cursor=end+1
        if name in members or folder>=len(contents) or offset+length>len(contents[folder]):raise ValueError('cabinet file escapes payload')
        value=contents[folder][offset:offset+length]
        members[name]={'size_bytes':length,'sha256':digest(value)}
        if metadata:members[name]['file_metadata']=pe_file_metadata(value)
    return members

def verify_model_bytes(path,model,fixture_failure=False):
    streams=CompoundFile(path).streams
    catalog={row[0] for row in table_rows(streams,'_Tables')}
    if not (set(SCHEMA)|{'Binary','Icon'})<=catalog:raise ValueError('MSI is missing required table definitions')
    details=cabinet_members(streams['payload.cab'],metadata=True)
    actual_members={name:{key:value for key,value in entry.items() if key!='file_metadata'} for name,entry in details.items()}
    expected={x['file_id']:{'size_bytes':x['size_bytes'],'sha256':x['sha256']} for x in model['payload']}
    if actual_members!=expected:raise ValueError('embedded MSI cabinet differs from complete inventory')
    for name,expected in model['tables'].items():
        if name=='File':
            expected=[list(row) for row in expected]
            for row in expected:row[4:6]=details[row[0]]['file_metadata']
        actual=table_rows(streams,name)
        if fixture_failure and name in ('CustomAction','InstallExecuteSequence'):actual=[r for r in actual if r[0]!='FixtureFailure']
        order=lambda rows:sorted(rows,key=lambda r:json.dumps(r,sort_keys=True))
        if order(actual)!=order(expected):raise ValueError('actual MSI table differs from source model: '+name)
    return {'streams':streams,'members':actual_members}

def verify_installer(path,receipt,source_root=None):
    source_root=Path(source_root or ROOT);path=Path(path);version=json.loads((source_root/'package.json').read_text(encoding='utf-8'))['version']
    if receipt.get('publication_eligible') is False or ('payload_qualification' in receipt and receipt['payload_qualification']!='passed'):raise ValueError('provisional MSI lacks joined qualified ZIP evidence')
    if receipt.get('kind')!='windows-installer-receipt' or receipt.get('format')!='msi' or receipt.get('platform')!='windows_amd64' or receipt.get('variant')!='selectable':raise ValueError('wrong MSI kind/platform/format')
    if any(receipt.get(k)!=version for k in ('schema_version','product_version','documentation_version')):raise ValueError('MSI version differs')
    if receipt.get('artifact')!=path.name or sha(path)!=receipt.get('artifact_sha256') or path.stat().st_size!=receipt.get('artifact_size_bytes'):raise ValueError('MSI artifact differs from receipt')
    if receipt.get('source_dirty') is not False or not re.fullmatch('[0-9a-f]{40}',receipt.get('revision','')):raise ValueError('MSI source is dirty/missing')
    if any(receipt.get('qualification',{}).get(k)!='passed' for k in QUALIFICATION_KEYS):raise ValueError('MSI lacks actual lifecycle/native UI qualification')
    payload=receipt['payload_receipt'];inventory=receipt['payload_inventory']
    from release import verify_qualification
    verify_qualification(payload)
    for k in ('revision','source_dirty','platform','schema_version','distribution'):
        if receipt.get(k)!=payload.get(k) or payload.get(k)!=inventory.get(k):raise ValueError('MSI/payload authority differs: '+k)
    if payload.get('signing')!=inventory.get('signing'):raise ValueError('payload signing authority differs')
    if receipt.get('signing')!=UNSIGNED_SIGNING:raise ValueError('MSI remains independently unsigned')
    if receipt.get('payload_archive')!=payload.get('archive') or receipt.get('payload_archive_sha256')!=payload.get('archive_sha256'):raise ValueError('MSI archive authority differs')
    inv_sha=digest(inventory_bytes(inventory))
    if inv_sha!=payload.get('inventory_sha256'):raise ValueError('MSI inventory identity differs')
    model=installer_model(inventory,inv_sha,(source_root/'LICENSE').read_text(encoding='utf-8'))
    if receipt.get('artifact')!='insonic_'+version+'_windows_amd64_installer.msi':raise ValueError('MSI filename/version/format identity differs')
    for k,value in [('msi_product_version',model['version']),('product_code',model['product_code']),('upgrade_code',model['upgrade_code']),('scope','per-user'),('license_sha256',sha(source_root/'LICENSE')),('icon_sha256',sha(source_root/ICON))]:
        if receipt.get(k)!=value:raise ValueError('MSI declared identity differs: '+k)
    streams=verify_model_bytes(path,model)['streams']
    if any(name in streams for name in ('\x05DigitalSignature','\x05MsiDigitalSignatureEx')):raise ValueError('unexpected signature in unsigned MSI')
    launcher=next(x for x in inventory['files'] if x['path']=='insonic-launcher.exe')
    if digest(streams.get('Binary.Launcher',b''))!=launcher['sha256']:raise ValueError('MSI preflight differs from qualified launcher')
    if streams.get('Icon.InsonicIcon')!=(source_root/ICON).read_bytes():raise ValueError('MSI icon differs from governed source')
    return {'inventory':inventory,'model':model}
