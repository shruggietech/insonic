# SPDX-License-Identifier: Apache-2.0
"""Real full-UI transactions, exclusively in the owned disposable CI account.

The ordinary actor uses the disposable CI Default desktop and its private registry profile. Every
product family here is fixture-owned. Native clicks drive the authored dialogs; external MSI
callbacks only observe messages and never substitute an outcome or cancellation.
"""
import ctypes as C
import ctypes.wintypes as W
import json
import os
from pathlib import Path
import shutil
import subprocess
import threading
import time
import uuid
import windows_msi as m


def prepare(stage, fixture, clone, diagnostic_only=False):
    """Build a small instrumented package with the production MSI author/model.

    The separate exact-payload lifecycle and full-UI install remain mandatory.
    Only the clients are token recorders, permitting all completion elections
    without creating persistent GUI/console sessions between MSI transactions.
    """
    m.disposable_runner()
    payload=stage/'ui-payload';shutil.copytree(fixture,payload)
    files=[]
    for p in sorted(payload.rglob('*')):
        if p.is_file() and p.name not in ('token-recorder.go','package-inventory.json') and not p.name.endswith(('.token.json','.diagnostic.json')):
            files.append({'path':p.relative_to(payload).as_posix(),'sha256':m.sha(p),'size_bytes':p.stat().st_size})
    version=json.loads((payload/'help/offline-manifest.json').read_text(encoding='utf-8'))['version']
    inv={'schema_version':version,'platform':'windows_amd64','variant':'desktop','files':files}
    raw=m.inventory_bytes(inv);(payload/'package-inventory.json').write_bytes(raw)
    setup_probe=diagnostic_only and os.environ.get('INSONIC_INSTALLER_DIAGNOSTIC_MODE') in ('rollback-setup','rollback-setup-full')
    if setup_probe:
        inv['kind']='native-package-inventory';raw=m.inventory_bytes(inv);(payload/'package-inventory.json').write_bytes(raw)
    family=m.UPGRADE_CODE if setup_probe else m.guid('owned-full-ui/'+uuid.uuid4().hex)
    model=m.installer_model(inv,m.digest(raw),(payload/'LICENSE').read_text(encoding='utf-8'),upgrade=family)
    old=stage/'ui-old.msi';m.author(old,payload,model)
    parts=list(map(int,version.split('.')));parts[2]+=1;new_version='.'.join(map(str,parts))
    clone(old,stage/'ui-new.msi',new_version,family)
    clone(old,stage/'ui-bad.msi',new_version,family,failure=True)
    clone(old,stage/'ui-missing-prerequisites.msi',version,family,missing_prerequisites=True)
    shutil.copyfile(old,stage/'ui-scaled.msi')
    db=m.NativeDatabase(stage/'ui-scaled.msi',write=True)
    try:
        db.execute('UPDATE `TextStyle` SET `Size`=14 WHERE `TextStyle`=?',['Normal'])
        db.execute('UPDATE `TextStyle` SET `Size`=18 WHERE `TextStyle`=?',['Heading'])
        db.check(db.dll.MsiDatabaseCommit(db.handle))
    finally:db.close()
    config={'version':version,'old_product':model['product_code'],'new_product':m.product_code(new_version,family),'payload':model['payload'],'family':family}
    (stage/'ui-model.json').write_text(json.dumps(config),encoding='utf-8',newline='\n')
    if setup_probe:
        import windows_setup
        clone(old,stage/'ui-cancel.msi',new_version,family)
        db=m.NativeDatabase(stage/'ui-cancel.msi',write=True)
        try:
            db.execute('INSERT INTO `Binary` (`Name`,`Data`) VALUES (?,?)',['OwnedCancellationBarrier',payload/'owned-cancel-barrier.exe'],streams=(2,))
            db.execute('INSERT INTO `CustomAction` (`Action`,`Type`,`Source`,`Target`) VALUES (?,?,?,?)',['OwnedCancellationBarrier',2,'OwnedCancellationBarrier','"'+str(stage)+'"'])
            db.execute('INSERT INTO `InstallExecuteSequence` (`Action`,`Condition`,`Sequence`) VALUES (?,?,?)',['OwnedCancellationBarrier','OLDPRODUCTS',3900])
            db.check(db.dll.MsiDatabaseCommit(db.handle))
        finally:db.close()
        for name in ('old','new','bad','cancel'):
            source=stage/('ui-'+name+'.msi');target=stage/('ui-'+name+'-setup.exe')
            shutil.copyfile(fixture/'owned-setup.exe',target)
            props=dict(m.table_rows(m.CompoundFile(source).streams,'Property'))
            windows_setup.add_resources(target.resolve(),props['ProductVersion'])
            identity={'product_code':props['ProductCode'],'upgrade_code':props['UpgradeCode'],'version':props['ProductVersion'],'sha256':m.sha(source),'size_bytes':source.stat().st_size}
            windows_setup.append_payload(target,source,identity)
    if diagnostic_only:return
    # A system browser is readable by the new ordinary account. Do not use the
    # elevated runner's private Playwright cache or change a machine association.
    browsers=[Path(os.environ.get('ProgramFiles(x86)','C:/Program Files (x86)'))/'Microsoft/Edge/Application/msedge.exe',Path(os.environ.get('ProgramFiles','C:/Program Files'))/'Google/Chrome/Application/chrome.exe']
    browser=next((p for p in browsers if p.is_file()),None)
    if browser is None:raise RuntimeError('ordinary offline renderer requires a readable installed browser')
    node=shutil.which('node')
    if not node:raise RuntimeError('Node missing for owned association renderer')
    shutil.copyfile(node,stage/'node.exe')
    package=m.ROOT/'node_modules/playwright-core'
    if not package.is_dir():package=m.ROOT/'desktop/frontend/node_modules/playwright-core'
    if not package.is_dir():raise RuntimeError('Playwright core fixture dependency is missing')
    shutil.copytree(package,stage/'node_modules/playwright-core')
    (stage/'ui-renderer.json').write_text(json.dumps({'browser':str(browser),'node':str(stage/'node.exe'),'script':str(stage/'scripts/render-installed-help.mjs'),'environment':{key:os.environ[key] for key in ('GITHUB_ACTIONS','RUNNER_ENVIRONMENT','INSONIC_DISPOSABLE_INSTALLER_TEST')}}),encoding='utf-8',newline='\n')


class ProgressInspection:
    """At most 250ms total observation grace, never once per validation tick."""
    def __init__(self,event,clock=time.monotonic):self.event=event;self.clock=clock;self.deadline=None
    def start(self):
        if self.deadline is None:self.deadline=self.clock()+.25
    def progress(self):
        if self.deadline is not None and not self.event.is_set():self.event.wait(min(.02,max(0,self.deadline-self.clock())))


def native_client_error_dialog(labels):
    return 'OK' in labels and 'Preparing to install...' not in labels

def launch_msiexec_diagnostic(msi,directory,log):
    m.disposable_runner();msi=Path(msi);stage=msi.resolve().parent;log=Path(log)
    if stage.parent!=Path(os.environ['ProgramData']).resolve() or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name) or msi.name not in ('ui-old.msi','ui-bad.msi') or (stage/'completion-invocation-diagnostic.json').read_text(encoding='utf-8')!='{"qualification":false}' or Path(directory).resolve()!=stage/'UI program é & spaces' or log.resolve().parent!=stage/'native' or not __import__('re').fullmatch('context.ui-rollback-(install|failed-upgrade).log',log.name):raise RuntimeError('native msiexec diagnostic is outside its owned stage')
    engine=str(Path(os.environ['SystemRoot'])/'System32/msiexec.exe')
    command=subprocess.list2cmdline([engine,'/i',str(msi),'/qf','/l*v',str(log)])+' INSTALLDIR="'+str(directory).replace('"','""')+'" REBOOT=ReallySuppress'
    return subprocess.Popen(command,executable=engine,cwd=stage,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=0x08000000)


def launch_standalone_diagnostic(msi,directory,log):
    m.disposable_runner();msi=Path(msi);stage=msi.resolve().parent;log=Path(log)
    if stage.parent!=Path(os.environ['ProgramData']).resolve() or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name) or msi.name!='ui-bad.msi' or (stage/'completion-invocation-diagnostic.json').read_text(encoding='utf-8')!='{"qualification":false}' or Path(directory).resolve()!=stage/'UI program é & spaces' or log.resolve().parent!=stage/'native' or log.name!='context.ui-rollback-failed-upgrade.log':raise RuntimeError('standalone UI diagnostic is outside its exact owned stage')
    command=[str(stage/'launcher-context-fixture/owned-coordinator.exe'),'--standalone','--full-ui',str(stage)]
    return subprocess.Popen(command,cwd=stage,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=0x08000000)


def launch_setup_probe(msi,directory,log):
    m.disposable_runner();msi=Path(msi);stage=msi.resolve().parent
    mode=json.loads((stage/'installer-diagnostic-mode.json').read_text(encoding='utf-8'))
    if stage.parent!=Path(os.environ['ProgramData']).resolve() or mode!={'feature_qualification':False,'mode':'rollback-setup-full'} or msi.name not in ('ui-old.msi','ui-bad.msi','ui-cancel.msi') or Path(directory).resolve()!=stage/'UI program é & spaces':raise RuntimeError('production setup probe is outside its owned ordinary stage')
    command=[str(msi.with_name(msi.stem+'-setup.exe')),'--install-dir',str(directory)]
    return subprocess.Popen(command,cwd=stage,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=0x08000000)


class NativeUI:
    def __init__(self,output):
        m.disposable_runner();self.output=Path(output);self.actor_pid=os.getpid()
        self.u=C.WinDLL('user32',use_last_error=True);self.dll=C.WinDLL('msi',use_last_error=True)
        self.cb=C.WINFUNCTYPE(W.BOOL,W.HWND,W.LPARAM)
        self.u.EnumWindows.argtypes=[self.cb,W.LPARAM];self.u.EnumChildWindows.argtypes=[W.HWND,self.cb,W.LPARAM]
        self.u.GetWindowThreadProcessId.argtypes=[W.HWND,C.POINTER(W.DWORD)]
        self.u.GetWindowTextW.argtypes=[W.HWND,W.LPWSTR,C.c_int];self.u.GetClassNameW.argtypes=[W.HWND,W.LPWSTR,C.c_int]
        self.u.IsWindowVisible.argtypes=[W.HWND];self.u.IsWindowEnabled.argtypes=[W.HWND]
        self.u.PostMessageW.argtypes=[W.HWND,W.UINT,W.WPARAM,W.LPARAM]
        self.u.SendMessageTimeoutW.argtypes=[W.HWND,W.UINT,W.WPARAM,W.LPARAM,W.UINT,W.UINT,C.POINTER(C.c_size_t)]
        self.u.GetWindowRect.argtypes=[W.HWND,C.POINTER(W.RECT)]
        self.u.GetWindowLongW.argtypes=[W.HWND,C.c_int];self.u.GetWindowLongW.restype=W.LONG
        self.u.GetDlgItem.argtypes=[W.HWND,C.c_int];self.u.GetDlgItem.restype=W.HWND
        self.dll.MsiSetInternalUI.argtypes=[W.UINT,C.POINTER(W.HWND)];self.dll.MsiSetInternalUI.restype=W.UINT
        self.dll.MsiInstallProductW.argtypes=[W.LPCWSTR,W.LPCWSTR];self.dll.MsiInstallProductW.restype=W.UINT
        self.dll.MsiQueryProductStateW.argtypes=[W.LPCWSTR];self.dll.MsiQueryProductStateW.restype=C.c_int
        self.dll.MsiEnableLogW.argtypes=[W.DWORD,W.LPCWSTR,W.DWORD]
        self.handler=C.WINFUNCTYPE(C.c_int,C.c_void_p,W.UINT,W.LPCWSTR)
        self.dll.MsiSetExternalUIW.argtypes=[self.handler,W.DWORD,C.c_void_p];self.dll.MsiSetExternalUIW.restype=C.c_void_p

    def text(self,hwnd):
        b=C.create_unicode_buffer(4096);self.u.GetWindowTextW(hwnd,b,4096)
        if not b.value:
            # GetWindowText cannot read another process's edit controls. System
            # WM_GETTEXT marshaling supports the owned native client's error text.
            count=C.c_size_t();self.u.SendMessageTimeoutW(hwnd,0x000d,len(b),C.addressof(b),2,500,C.byref(count))
        return b.value

    def klass(self,hwnd):
        b=C.create_unicode_buffer(128);self.u.GetClassNameW(hwnd,b,128);return b.value

    def send(self,hwnd,message):
        value=C.c_size_t()
        if not self.u.SendMessageTimeoutW(hwnd,message,0,0,2,1000,C.byref(value)):raise RuntimeError('owned native control read timed out')
        return value.value

    def windows(self,helper=None):
        found=[]
        def visit(hwnd,_):
            pid=W.DWORD();self.u.GetWindowThreadProcessId(hwnd,C.byref(pid))
            owned=pid.value==self.actor_pid
            if not owned and helper and self.text(hwnd)=='Open insonic documentation':
                kernel=C.WinDLL('kernel32',use_last_error=True)
                kernel.OpenProcess.argtypes=[W.DWORD,W.BOOL,W.DWORD];kernel.OpenProcess.restype=W.HANDLE
                kernel.QueryFullProcessImageNameW.argtypes=[W.HANDLE,W.DWORD,W.LPWSTR,C.POINTER(W.DWORD)]
                kernel.CloseHandle.argtypes=[W.HANDLE]
                process=kernel.OpenProcess(0x1000,False,pid.value)
                if process:
                    try:
                        image=C.create_unicode_buffer(32768);size=W.DWORD(32768)
                        owned=bool(kernel.QueryFullProcessImageNameW(process,0,image,C.byref(size))) and Path(image.value).resolve()==helper.resolve()
                    finally:kernel.CloseHandle(process)
            if owned and self.u.IsWindowVisible(hwnd):found.append(hwnd)
            return True
        self.u.EnumWindows(self.cb(visit),0)
        return found

    def controls(self,hwnd):
        found=[]
        def visit(control,_):
            if self.u.IsWindowVisible(control):found.append(control)
            return True
        self.u.EnumChildWindows(hwnd,self.cb(visit),0)
        return found

    def button(self,controls,label):
        found=[c for c in controls if self.klass(c).casefold()=='button' and self.text(c).replace('&','').strip('<> ')==label]
        if len(found)!=1:raise RuntimeError('expected one owned native control: '+label)
        return found[0]

    def click(self,controls,label):
        button=self.button(controls,label)
        if not self.u.IsWindowEnabled(button):raise RuntimeError('native control is disabled: '+label)
        if not self.u.PostMessageW(button,0xf5,0,0):raise C.WinError(C.get_last_error())

    def check(self,controls,label,value):
        hwnd=self.button(controls,label)
        if (self.send(hwnd,0xf0)==1)!=value:
            self.click(controls,label)
            deadline=time.monotonic()+2
            while (self.send(hwnd,0xf0)==1)!=value:
                if time.monotonic()>deadline:raise RuntimeError('native checkbox did not retain election: '+label)
                time.sleep(.01)

    def capture(self,hwnd,name):
        # Actual transaction pixels, bounded to the exact owned window.
        import struct,zlib
        u=self.u;g=C.WinDLL('gdi32',use_last_error=True)
        u.GetDC.argtypes=[W.HWND];u.GetDC.restype=W.HDC;u.ReleaseDC.argtypes=[W.HWND,W.HDC]
        u.PrintWindow.argtypes=[W.HWND,W.HDC,W.UINT]
        g.CreateCompatibleDC.argtypes=[W.HDC];g.CreateCompatibleDC.restype=W.HDC
        g.CreateCompatibleBitmap.argtypes=[W.HDC,C.c_int,C.c_int];g.CreateCompatibleBitmap.restype=W.HBITMAP
        g.SelectObject.argtypes=[W.HDC,W.HANDLE];g.SelectObject.restype=W.HANDLE
        g.DeleteObject.argtypes=[W.HANDLE];g.DeleteDC.argtypes=[W.HDC]
        g.GetDIBits.argtypes=[W.HDC,W.HBITMAP,W.UINT,W.UINT,C.c_void_p,C.c_void_p,W.UINT]
        rect=W.RECT();u.GetWindowRect(hwnd,C.byref(rect));width=rect.right-rect.left;height=rect.bottom-rect.top
        if not 200<width<2000 or not 150<height<1600:raise RuntimeError('unexpected transaction window geometry')
        dc=u.GetDC(hwnd);memory=g.CreateCompatibleDC(dc);bitmap=g.CreateCompatibleBitmap(dc,width,height);old=g.SelectObject(memory,bitmap)
        try:
            if not u.PrintWindow(hwnd,memory,0):raise RuntimeError('native transaction capture failed')
            g.SelectObject(memory,old);header=C.create_string_buffer(struct.pack('<IiiHHIIiiII',40,width,-height,1,32,0,width*height*4,0,0,0,0));pixels=C.create_string_buffer(width*height*4)
            if g.GetDIBits(memory,bitmap,0,height,pixels,header,0)!=height:raise RuntimeError('native transaction pixel read failed')
            raw=pixels.raw;rows=[]
            for y in range(height):
                row=raw[y*width*4:(y+1)*width*4];rgb=bytearray(width*3)
                rgb[0::3]=row[2::4];rgb[1::3]=row[1::4];rgb[2::3]=row[0::4];rows.append(b'\0'+rgb)
            def chunk(kind,data):return struct.pack('>I',len(data))+kind+data+struct.pack('>I',zlib.crc32(kind+data)&0xffffffff)
            png=b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',width,height,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(b''.join(rows)))+chunk(b'IEND',b'')
            target=self.output.parent/('context.ui-'+name+'.png');target.write_bytes(png)
            return {'path':target.name,'sha256':m.sha(target),'width':width,'height':height}
        finally:g.SelectObject(memory,old);g.DeleteObject(bitmap);g.DeleteDC(memory);u.ReleaseDC(hwnd,dc)

    def run(self,msi,directory,name,operation='install',selection='both',launch=False,docs=False,cancel=None,expected=0,before_finish=None,execution_required=True,native_process=False,before_terminal_close=None,standalone_process=False,setup_process=False):
        state={'name':name,'operation':operation,'fixture_msi_sha256':m.sha(msi),'pages':[],'actions':[],'progress_messages':0,'progress_observed':False,'bar_positions':[],'visible_phases':[],'terminal':None,'captures':[]}
        action_text=m.table_rows(m.CompoundFile(msi).streams,'ActionText');descriptions={row[1] for row in action_text if row[1]};described_actions={row[0] for row in action_text if row[1]}
        action_ready=threading.Event();progress_seen=threading.Event();done=threading.Event();inspection=ProgressInspection(progress_seen)
        def observe(_,kind,message):
            category=kind&0xff000000
            if category==0x08000000:
                action=(message or '').split(': ',1)[-1].split('.',1)[0]
                state['actions'].append((message or '')[:350])
                if action=='InstallFiles':action_ready.set()
                if action in described_actions:inspection.start()
            if category==0x0a000000:
                state['progress_messages']+=1
                # Bounded observation grace for authored phases only. Validation
                # can emit thousands of ticks; their count must not add latency.
                inspection.progress()
            return 0
        callback=self.handler(observe)
        def install():
            initialized=False;ole=None
            try:
                if native_process:
                    process=(launch_setup_probe if setup_process else launch_standalone_diagnostic if standalone_process else launch_msiexec_diagnostic)(msi,directory,self.output.parent/('context.ui-'+name+'.log'))
                    self.actor_pid=process.pid;state['native_engine']={'kind':'production setup full UI' if setup_process else 'external coordinator full UI' if standalone_process else 'msiexec.exe full UI','pid':process.pid,'feature_qualification':False,'progress_source':'native controls and private passive MSI event receipt' if setup_process else 'native controls, no in-process API observer'}
                    stdout,stderr=process.communicate(timeout=75)
                    state['code']=process.returncode;state['native_engine'].update(stdout=stdout.decode(errors='replace')[-4000:],stderr=stderr.decode(errors='replace')[-4000:])
                    if setup_process:
                        trace=json.loads((Path(os.environ['LOCALAPPDATA'])/'insonic-setup/setup-events.json').read_text(encoding='utf-8'))
                        if trace.get('passive') is not True or trace.get('ui_level')!=5:raise RuntimeError('production setup native progress observer differs')
                        state['progress_messages']=trace['progress_messages'];state['actions']=trace['actions'];state['native_event_receipt']=trace
                    return
                ole=C.WinDLL('ole32');ole.CoInitializeEx.argtypes=[C.c_void_p,W.DWORD];ole.CoInitializeEx.restype=C.c_long
                ole.CoUninitialize.argtypes=[];ole.CoUninitialize.restype=None
                initialized=ole.CoInitializeEx(None,2) in (0,1)
                if not initialized:raise RuntimeError('native UI COM apartment could not initialize')
                state['initial_ui_level']=self.dll.MsiSetInternalUI(0,None)
                self.dll.MsiSetExternalUIW(callback,(1<<8)|(1<<10),None)
                state['ui_level_after_observer']=self.dll.MsiSetInternalUI(0,None)
                state['previous_ui_level']=self.dll.MsiSetInternalUI(5,None)
                state['configured_ui_level']=self.dll.MsiSetInternalUI(0,None)
                if state['configured_ui_level']!=5:raise RuntimeError('native full UI could not be selected')
                self.dll.MsiEnableLogW(0x1000,str(self.output.parent/('context.ui-'+name+'.log')),0)
                props='INSTALLDIR="'+str(directory)+'" REBOOT=ReallySuppress'
                state['code']=self.dll.MsiInstallProductW(str(msi),props)
            except BaseException as error:state['error']=str(error)
            finally:
                try:
                    if not native_process:
                        self.dll.MsiSetExternalUIW(self.handler(),0,None)
                        state['logging_closed']=self.dll.MsiEnableLogW(0,None,0)==0
                    if initialized:ole.CoUninitialize()
                finally:done.set()
        thread=threading.Thread(target=install,name='owned-native-full-ui',daemon=True);thread.start()
        deadline=time.monotonic()+60;last=None;cancel_sent=False;terminal_closed=False;navigation=0
        try:
            while not done.is_set():
                if time.monotonic()>deadline:raise RuntimeError('full-UI transaction exceeded bound: '+name)
                for hwnd in self.windows(directory/'insonic-launcher.exe'):
                    controls=self.controls(hwnd);labels=[self.text(c).replace('&','') for c in controls];joined='\n'.join(labels)
                    if self.text(hwnd)=='Open insonic documentation':
                        if not docs:raise RuntimeError('unexpected documentation warning')
                        if 'Repair' not in joined or 'installation remains available' not in joined:raise RuntimeError('documentation warning lacks actionable correction')
                        state['documentation_warning']=True
                        self.click(controls,'OK');continue
                    if 'Cancel this operation?' in joined:
                        self.click(controls,'Yes, cancel')
                        if setup_process and cancel=='execution':(Path(msi).parent/'native/setup-cancel-release').write_bytes(b'actual Cancel confirmation clicked\n')
                        time.sleep(.03);continue
                    if setup_process and self.text(hwnd)=='insonic setup':
                        if 'The previous installation was restored.' not in joined:raise RuntimeError('setup recovery correction requires attention: '+joined)
                        state['recovery_outcome_observed']=True;self.click(controls,'OK');continue
                    if 'Install Microsoft Visual C++ v14 x64 Redistributable, then retry:' in joined:
                        if name!='prerequisite-failure':raise RuntimeError('unexpected prerequisite rejection')
                        state['prerequisite_correction_observed']=True;self.click(controls,'OK');continue
                    if 'Intentional transactional fixture failure' in joined:
                        self.click(controls,'OK');time.sleep(.03);continue
                    state['last_native_window']={'title':self.text(hwnd)[:120],'controls':[label[:160] for label in labels[:24]]}
                    if self.text(hwnd)!='insonic Setup':
                        if native_process and self.text(hwnd)=='Windows Installer' and native_client_error_dialog(labels):
                            state['native_client_dialog']={'title':self.text(hwnd),'text':joined,'classes':[self.klass(c) for c in controls]}
                            state['captures'].append(self.capture(hwnd,name+'-native-client-error'))
                            self.click(controls,'OK')
                            raise RuntimeError('native MSI client rejected invocation: '+joined)
                        continue
                    if any(self.klass(c).casefold()=='msctls_progress32' for c in controls):
                        bar=next(c for c in controls if self.klass(c).casefold()=='msctls_progress32')
                        position=self.send(bar,0x0408)
                        if not state['bar_positions'] or state['bar_positions'][-1]!=position:state['bar_positions'].append(position)
                        for description in sorted(descriptions):
                            if description in joined and description not in state['visible_phases']:state['visible_phases'].append(description)
                        if not state['progress_observed'] and position>0 and state['visible_phases']:
                            state['progress_observed']=True;state['captures'].append(self.capture(hwnd,name+'-progress'));progress_seen.set()
                        ready=Path(msi).parent/'native/setup-cancel-ready.json'
                        if setup_process and cancel=='execution' and ready.is_file() and not cancel_sent:
                            model=json.loads((Path(msi).parent/'ui-model.json').read_text(encoding='utf-8'))
                            if self.dll.MsiQueryProductStateW(model['old_product'])!=-1 or (directory/m.OWNERSHIP).exists():raise RuntimeError('cancellation barrier preceded actual old removal')
                            state['old_removed_before_cancel']=True;action_ready.set()
                        if cancel=='execution' and action_ready.is_set() and (state['progress_observed'] or setup_process) and not cancel_sent:
                            self.click(controls,'Cancel');cancel_sent=True
                        continue
                    page=next((p for p,mark in [('Welcome','Install for this Windows account.'),('License','Apache License 2.0'),('Choices','Select clients and shortcuts'),('Location','Choose program directory'),('Ready','Setup verifies capacity'),('Maintenance','Change, repair or remove'),('Cancelled','Setup cancelled'),('FatalError','Setup could not finish'),('Finish','The selected operation completed.')] if mark in joined),None)
                    if page is None and any(x in joined for x in ('insonic installed','insonic updated','insonic changed','insonic repaired','insonic removed')):page='Finish'
                    if page is None:continue
                    signature=(hwnd,page)
                    if signature==last:continue
                    last=signature;state['pages'].append(page)
                    rect=W.RECT();self.u.GetWindowRect(hwnd,C.byref(rect))
                    for control in controls:
                        bounds=W.RECT();self.u.GetWindowRect(control,C.byref(bounds))
                        if bounds.left<rect.left or bounds.top<rect.top or bounds.right>rect.right or bounds.bottom>rect.bottom:raise RuntimeError('visible native control escapes dialog bounds')
                    if name=='enlarged-font-cancel':state['captures'].append(self.capture(hwnd,name+'-'+page.lower()))
                    if cancel=='before' and page=='Welcome':self.click(controls,'Cancel');cancel_sent=True;continue
                    if page=='Welcome':self.click(controls,'Next')
                    elif page=='License':
                        next_button=self.button(controls,'Next')
                        accepted=self.send(self.button(controls,'I accept the Apache License 2.0'),0xf0)
                        if navigation==3:
                            if accepted!=1 or not self.u.IsWindowEnabled(next_button):raise RuntimeError('Back navigation lost accepted license')
                            navigation=4
                        elif accepted!=0 or self.u.IsWindowEnabled(next_button):raise RuntimeError('license acceptance default changed')
                        self.check(controls,'I accept the Apache License 2.0',True);self.click(controls,'Next')
                    elif page=='Choices':
                        if navigation in (2,4):
                            for label,elected in [('Desktop application',selection!='cli'),('CLI integrations (persistent help/setup session)',selection!='desktop'),('Create Start menu shortcuts',False),('Create Desktop shortcut for the desktop application',False)]:
                                if self.send(self.button(controls,label),0xf0)!=int(elected):raise RuntimeError('Back navigation lost selected clients/shortcuts')
                            if navigation==2:navigation=3;self.click(controls,'Back');continue
                            navigation=5
                        self.check(controls,'Desktop application',selection!='cli');self.check(controls,'CLI integrations (persistent help/setup session)',selection!='desktop')
                        self.check(controls,'Create Start menu shortcuts',False);self.check(controls,'Create Desktop shortcut for the desktop application',False);self.click(controls,'Next')
                    elif page=='Location':
                        if navigation==1:navigation=2;self.click(controls,'Back')
                        else:
                            if navigation==5:navigation=6
                            self.click(controls,'Next')
                    elif page=='Ready':
                        if name=='enlarged-font-cancel' and navigation==0:navigation=1;self.click(controls,'Back')
                        elif cancel=='ready':
                            if navigation!=6:raise RuntimeError('configuration Back round trip did not complete')
                            state['navigation_selections_preserved']=True;self.click(controls,'Cancel');cancel_sent=True
                        else:self.click(controls,{'install':'Install','upgrade':'Update','change':'Apply changes'}[operation])
                    elif page=='Maintenance':self.click(controls,{'repair':'Repair','remove':'Remove','change':'Next'}[operation])
                    else:
                        state['terminal']=page
                        if page=='Finish':
                            heading={'install':'installed','upgrade':'updated','change':'changed','repair':'repaired','remove':'removed'}[operation]
                            if 'insonic '+heading not in labels:raise RuntimeError('completion heading does not match operation')
                        if page=='Finish' and operation=='install':
                            client_label='Open insonic CLI' if selection=='cli' else 'Launch insonic'
                            if self.send(self.button(controls,client_label),0xf0)!=1 or self.send(self.button(controls,'Open documentation'),0xf0)!=0:raise RuntimeError('fresh completion defaults changed')
                            self.check(controls,client_label,launch);self.check(controls,'Open documentation',docs)
                            if before_finish:before_finish()
                        elif any(label in labels for label in ('Launch insonic','Open insonic CLI','Open documentation')):raise RuntimeError('completion actions leaked into non-fresh or failed operation')
                        push=[c for c in controls if self.klass(c).casefold()=='button' and (self.u.GetWindowLongW(c,-16)&15) in (0,1)]
                        if len(push)!=1:raise RuntimeError('terminal outcome must expose exactly one action')
                        default=self.u.GetDlgItem(hwnd,self.send(hwnd,0x0400)&0xffff)
                        if not default or self.text(default)!=('Finish' if page=='Finish' else 'Close'):raise RuntimeError('terminal default action differs from the displayed outcome')
                        state['captures'].append(self.capture(hwnd,name+'-'+page.lower()))
                        if before_terminal_close:before_terminal_close(page)
                        if name=='remove':
                            # Escape uses the authored terminal cancel/default
                            # control and must retain the committed success code.
                            self.u.PostMessageW(hwnd,0x0100,0x1b,0);self.u.PostMessageW(hwnd,0x0101,0x1b,0)
                            state['terminal_keyboard']='Escape'
                        else:self.click(controls,'Finish' if page=='Finish' else 'Close')
                        terminal_closed=True
                time.sleep(.01)
            thread.join(1)
            if state.get('error'):raise RuntimeError(state['error'])
            if state.get('code')!=expected or not terminal_closed:raise RuntimeError('native UI/outcome mismatch: '+json.dumps(state))
            want='Cancelled' if expected==1602 else 'FatalError' if expected else 'Finish'
            if state['terminal']!=want:raise RuntimeError('terminal page differs from actual Installer result')
            if execution_required and cancel not in ('before','ready') and (not state['progress_observed'] or not state['progress_messages'] or len(state['bar_positions'])<2 or not state['visible_phases']):raise RuntimeError('visible engine-driven bar movement/action text was not observed')
            if cancel and not cancel_sent:raise RuntimeError('native Cancel control was not exercised')
            return state
        except BaseException as error:
            state['driver_error']=str(error);progress_seen.set()
            # Ask the exact owned UI to cancel and finish its native rollback.
            # Never terminate Windows Installer or replace the original failure.
            cleanup_deadline=time.monotonic()+15
            try:
                while not done.is_set() and time.monotonic()<cleanup_deadline:
                    for hwnd in self.windows(directory/'insonic-launcher.exe'):
                        controls=self.controls(hwnd);joined='\n'.join(self.text(control).replace('&','') for control in controls)
                        if 'Cancel this operation?' in joined:self.click(controls,'Yes, cancel')
                        elif any(mark in joined for mark in ('Setup cancelled','Setup could not finish')):self.click(controls,'Close')
                        elif 'The selected operation completed.' in joined or any('insonic '+word in joined for word in ('installed','updated','changed','repaired','removed')):self.click(controls,'Finish')
                        elif self.text(hwnd)=='insonic Setup':
                            buttons=[control for control in controls if self.klass(control).casefold()=='button' and self.text(control).replace('&','')=='Cancel' and self.u.IsWindowEnabled(control)]
                            if buttons:self.click(controls,'Cancel')
                    time.sleep(.025)
                thread.join(.5);state['failed_driver_native_call_settled']=done.is_set()
            except BaseException as cleanup_error:state['failed_driver_cleanup_error']=str(cleanup_error)
            raise
        finally:
            progress_seen.set();self.output.parent.joinpath('context.ui-'+name+'.json').write_text(json.dumps(state,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')


def qualify(stage,exact_installer,exact_directory,output,token,quiet_install):
    m.disposable_runner();stage=Path(stage);config=json.loads((stage/'ui-model.json').read_text(encoding='utf-8'))
    ui=NativeUI(output);directory=stage/'UI program é & spaces';old=stage/'ui-old.msi';new=stage/'ui-new.msi';bad=stage/'ui-bad.msi';results=[]
    exact_product=dict(m.table_rows(m.CompoundFile(exact_installer).streams,'Property'))['ProductCode']
    def states(row,expected):
        observed={product:ui.dll.MsiQueryProductStateW(product) for product in expected}
        row['registered_product_states']=observed
        if observed!=expected:raise RuntimeError('native registered state differs after '+row['name']+': '+json.dumps(observed))
    try:
        # Real prerequisite rejection reports correction and can be retried with
        # the unchanged supported host prerequisites; no machine dependency edits.
        rejected=ui.run(stage/'ui-missing-prerequisites.msi',directory,'prerequisite-failure',selection='cli',expected=1603,execution_required=False)
        if not rejected.get('prerequisite_correction_observed') or (directory/m.OWNERSHIP).exists():raise RuntimeError('native prerequisite failure did not preserve the untouched destination/correction')
        results.append(rejected);states(rejected,{config['old_product']:-1})
        # Production payload, actual engine/full UI, elected no completion actions.
        results.append(ui.run(exact_installer,exact_directory,'exact-install',selection='cli'))
        states(results[-1],{exact_product:5})
        results.append(ui.run(exact_installer,exact_directory,'exact-remove',operation='remove',selection='cli'))
        states(results[-1],{exact_product:-1})
        results.append(ui.run(old,directory,'before-cancel',cancel='before',expected=1602))
        results.append(ui.run(stage/'ui-scaled.msi',directory,'enlarged-font-cancel',cancel='ready',expected=1602))
        if (directory/m.OWNERSHIP).exists():raise RuntimeError('pre-execution cancellation installed files')
        # Completion election matrix below is also the initial fresh-install UI
        # proof; all three feature choices use the same production launcher.
        with association(stage,token) as renderer:
            for selection in ('desktop','cli','both'):
                for launch,docs in ((False,False),(True,False),(False,True),(True,True)):
                    name=selection+'-'+str(int(launch))+str(int(docs));nonce=uuid.uuid4().hex
                    receipt=stage/'native'/('context.docs-'+name+'.json')
                    renderer.update(directory=str(directory),version=config['version'],output=str(receipt),nonce=nonce)
                    (directory.parent/'documentation-handler.json').write_text(json.dumps(renderer),encoding='utf-8',newline='\n')
                    results.append(ui.run(old,directory,name,selection=selection,launch=launch,docs=docs))
                    states(results[-1],{config['old_product']:5})
                    deadline=time.monotonic()+25
                    client=directory/('insonic.exe.token.json' if selection=='cli' else 'insonic-desktop.exe.token.json')
                    while (launch and not client.is_file()) or (docs and not receipt.is_file()):
                        if time.monotonic()>deadline:raise RuntimeError('elected completion did not finish: '+name)

                        time.sleep(.025)
                    if client.exists()!=launch or receipt.exists()!=docs:raise RuntimeError('independent completion election mismatch')
                    if launch:
                        observed=json.loads(client.read_text(encoding='utf-8'))
                        if observed['elevated'] or observed['administrator_enabled'] or observed['user_sid_sha256']!=token['user_sid_sha256'] or observed['integrity_rid']!=8192 or observed['session_id']!=token['session_id'] or observed['authentication_luid']!=token['authentication_luid']:raise RuntimeError('client completion changed user/elevation')
                    if docs:
                        proof=json.loads(receipt.read_text(encoding='utf-8'))
                        if proof.get('nonce')!=nonce or proof.get('state')!='passed':raise RuntimeError('actual OS HTML association/rendering proof differs')
                    # Token/diagnostic receipts are fixture-owned, not MSI bytes.
                    for extra in directory.glob('*.json'):
                        if extra.name.endswith(('.token.json','.diagnostic.json')):extra.unlink()
                    quiet_install(['/x',config['old_product']],stage/'native'/('context.ui-'+name+'-remove.log'))
            # Explicit unattended documentation election still cannot publish
            # a completion action: no full-UI Finish event exists here.
            unattended=stage/'native/context.docs-unattended.json'
            renderer.update(directory=str(directory),version=config['version'],output=str(unattended),nonce=uuid.uuid4().hex)
            (stage/'documentation-handler.json').write_text(json.dumps(renderer),encoding='utf-8',newline='\n')
            (stage/'documentation-handler.exe.token.json').unlink(missing_ok=True)
            quiet_log=stage/'native/context.ui-unattended.log'
            quiet_install(['/i',old,'INSTALLDIR='+str(directory),'LICENSE_ACCEPTED=1','ADDLOCAL=Core,CLI','OPEN_DOCUMENTATION=1'],quiet_log)
            text=quiet_log.read_text(encoding='utf-16',errors='replace')
            if unattended.exists() or (stage/'documentation-handler.exe.token.json').exists() or 'Doing action: OpenDocumentation' in text:raise RuntimeError('unattended transaction opened documentation')
            quiet_install(['/x',config['old_product']],stage/'native/context.ui-unattended-remove.log')
        missing=ui.run(old,directory,'missing-documentation',docs=True,before_finish=lambda:(directory/'help/offline-manifest.json').unlink())
        if not missing.get('documentation_warning'):raise RuntimeError('elected missing help did not report its correction')
        results.append(missing)
        quiet_install(['/x',config['old_product']],stage/'native'/'context.ui-missing-remove.log')
        results.append(ui.run(old,directory,'maintenance-install'))
        results.append(ui.run(old,directory,'change',operation='change',selection='cli'))
        expected={f['path']:m.sha(directory/f['path']) for f in config['payload'] if (directory/f['path']).is_file()}
        (directory/'insonic-session.cmd').unlink()
        results.append(ui.run(old,directory,'repair',operation='repair',selection='cli'))
        for name in expected:
            if m.sha(directory/name)!=expected[name]:raise RuntimeError('full-UI repair did not restore exact bytes')
        results.append(ui.run(new,directory,'execution-cancel',operation='upgrade',selection='cli',cancel='execution',expected=1602))
        states(results[-1],{config['old_product']:5,config['new_product']:-1})
        for name in expected:
            if m.sha(directory/name)!=expected[name]:raise RuntimeError('full-UI cancellation did not roll back old bytes')
        results.append(ui.run(bad,directory,'failed-upgrade',operation='upgrade',selection='cli',expected=1603))
        states(results[-1],{config['old_product']:5,config['new_product']:-1})
        for name in expected:
            if m.sha(directory/name)!=expected[name]:raise RuntimeError('full-UI cancellation/failure did not roll back old bytes')
        results.append(ui.run(new,directory,'upgrade',operation='upgrade',selection='cli'))
        states(results[-1],{config['old_product']:-1,config['new_product']:5})
        results.append(ui.run(new,directory,'remove',operation='remove',selection='cli'))
        states(results[-1],{config['old_product']:-1,config['new_product']:-1})
        if any((directory/f['path']).exists() for f in config['payload']):raise RuntimeError('full-UI removal retained owned payload')
        return {'state':'passed','token':token,'transactions':results,'completion_cases':12,'exact_payload_full_ui':['install','remove'],'instrumented_clients':'owned native token reporters, production launcher and MSI model','outcomes':[0,1602,1603]}
    finally:
        for suffix in ('.diagnostic.json','.token.json','.last-token.json'):
            source=stage/('documentation-handler.exe'+suffix)
            if source.is_file():
                if source.is_symlink() or source.stat().st_size>65536:raise RuntimeError('owned association diagnostic is redirected or oversized')
                shutil.copyfile(source,stage/('native/context.association-handler'+suffix))
        for product in (exact_product,config['old_product'],config['new_product']):
            # Every identity is an explicitly generated, owned fixture family.
            # Query registration, not marker presence, so successful cleanup is
            # never misclassified as an intentionally invalid installation.
            if ui.dll.MsiQueryProductStateW(product)==5:
                quiet_install(['/x',product],stage/'native'/('context.ui-cleanup-'+product.strip('{}')+'.log'))


@__import__('contextlib').contextmanager
def fixture_html_registration(stage,token,prog,handler):
    """Override only the owned ordinary profile's resolved ProgID command."""
    m.disposable_runner();stage=Path(stage);marker=json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'))
    if stage.resolve()!=Path(os.environ['ProgramData']).resolve()/stage.name or not __import__('re').fullmatch('insonic-ci-account-[0-9a-f]{32}',stage.name) or not __import__('re').fullmatch('inso_ci_[0-9a-f]{12}',marker.get('account','')) or marker.get('sid_sha256')!=token['user_sid_sha256'] or token['administrator_enabled'] or token['elevated']:raise RuntimeError('HTML fixture requires the exact ordinary owned account stage')
    if Path(handler).is_symlink() or Path(handler).resolve()!=stage.resolve()/'documentation-handler.exe':raise RuntimeError('HTML fixture handler is not the exact staged image')
    if not __import__('re').fullmatch('[A-Za-z0-9_.-]{1,128}',prog):raise RuntimeError('resolved HTML ProgID is not one bounded class name')
    security=C.WinDLL('advapi32',use_last_error=True);security.GetUserNameW.argtypes=[W.LPWSTR,C.POINTER(W.DWORD)];security.GetUserNameW.restype=W.BOOL
    account=C.create_unicode_buffer(256);size=W.DWORD(256)
    if not security.GetUserNameW(account,C.byref(size)) or account.value.casefold()!=marker['account'].casefold():raise RuntimeError('HTML fixture is not running as its exact owned account')
    import winreg
    root=winreg.HKEY_CURRENT_USER;choice=r'Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\.html\UserChoice';base='Software\\Classes\\'+prog
    def choice_snapshot():
        try:
            with winreg.OpenKey(root,choice) as key:
                children,count,_=winreg.QueryInfoKey(key)
                if children or count>16:raise RuntimeError('HTML UserChoice observation exceeds bound')
                values=[winreg.EnumValue(key,index) for index in range(count)]
                if len(repr(values).encode())>65536:raise RuntimeError('HTML UserChoice values exceed observation bound')
                return sorted(values,key=lambda item:item[0].casefold())
        except FileNotFoundError:return None
    original_choice=choice_snapshot();created=[];before={};written=False;delegate_removed=False;command=base+r'\shell\open\command'
    evidence={'kind':'owned-ordinary-HTML-class-command-fixture','resolved_prog_id':prog,'user_choice_modified':False,'machine_associations_modified':False,'restored':False,'user_choice_sha256':m.digest(repr(original_choice).encode())}
    try:
        for suffix in ('',r'\shell',r'\shell\open',r'\shell\open\command'):
            name=base+suffix
            try:
                with winreg.OpenKey(root,name):pass
            except FileNotFoundError:
                with winreg.CreateKey(root,name):pass
                created.append(name)
        with winreg.OpenKey(root,command,0,winreg.KEY_QUERY_VALUE|winreg.KEY_SET_VALUE) as key:
            for name in ('','DelegateExecute'):
                try:before[name]=winreg.QueryValueEx(key,name)
                except FileNotFoundError:before[name]=None
            # A duplicate per-user ProgID shadows its machine class. Retain the
            # existing UserChoice; only its owned-profile command is instrumented.
            winreg.SetValueEx(key,'',0,winreg.REG_SZ,'"'+str(handler)+'" "%1"');written=True
            if before['DelegateExecute'] is not None:winreg.DeleteValue(key,'DelegateExecute')
            delegate_removed=True
        yield evidence
    finally:
        try:
            if written:
                with winreg.OpenKey(root,command,0,winreg.KEY_QUERY_VALUE|winreg.KEY_SET_VALUE) as key:
                    if winreg.QueryValueEx(key,'')!=('"'+str(handler)+'" "%1"',winreg.REG_SZ):raise RuntimeError('owned HTML command changed outside the fixture; restoration refused')
                    try:delegate=winreg.QueryValueEx(key,'DelegateExecute')
                    except FileNotFoundError:delegate=None
                    if delegate!=(None if delegate_removed else before['DelegateExecute']):raise RuntimeError('owned HTML delegate changed outside the fixture; restoration refused')
                    for name,value in before.items():
                        if value is None:
                            try:winreg.DeleteValue(key,name)
                            except FileNotFoundError:pass
                        else:winreg.SetValueEx(key,name,0,value[1],value[0])
                    for name,value in before.items():
                        try:restored=winreg.QueryValueEx(key,name)
                        except FileNotFoundError:restored=None
                        if restored!=value:raise RuntimeError('owned HTML class value restoration differs')
            for name in reversed(created):
                with winreg.OpenKey(root,name) as key:
                    children,values,_=winreg.QueryInfoKey(key)
                    if children or values:raise RuntimeError('owned HTML class acquired outside data; deletion refused')
                winreg.DeleteKey(root,name)
            evidence['restored']=choice_snapshot()==original_choice
            if not evidence['restored']:raise RuntimeError('HTML UserChoice changed during the owned fixture')
        finally:(stage/'native/context.association.json').write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')


def association(stage,token):
    from contextlib import contextmanager
    @contextmanager
    def owned():
        m.disposable_runner()
        handler=stage/'documentation-handler.exe';shutil.copyfile(stage/'launcher-context-fixture/insonic-desktop.exe',handler)
        shutil.copyfile(stage/'launcher-context-fixture/owned-ci-fixture.json',stage/'owned-ci-fixture.json')
        shell=C.WinDLL('shell32');shell.SHChangeNotify(0x08000000,0,None,None)
        query=C.WinDLL('shlwapi').AssocQueryStringW;query.argtypes=[W.DWORD,C.c_int,W.LPCWSTR,W.LPCWSTR,W.LPWSTR,C.POINTER(W.DWORD)];query.restype=C.c_long
        prog=C.create_unicode_buffer(256);size=W.DWORD(len(prog))
        if query(0,20,'.html',None,prog,C.byref(size)) or not prog.value:raise RuntimeError('ordinary HTML association has no resolved ProgID')
        try:
            with fixture_html_registration(stage,token,prog.value,handler):
                shell.SHChangeNotify(0x08000000,0,None,None)
                executable=C.create_unicode_buffer(32768);size=W.DWORD(len(executable))
                try:
                    if query(0,2,'.html','open',executable,C.byref(size)) or Path(executable.value).resolve()!=handler.resolve():raise RuntimeError('ordinary HTML association did not resolve to owned handler')
                    renderer=json.loads((stage/'ui-renderer.json').read_text(encoding='utf-8'));yield renderer
                    reports=[p for p in (stage/'native').glob('context.docs-*.json') if p.name!='context.docs-unattended.json']
                    if len(reports)!=6:raise RuntimeError('ordinary association did not complete all six elected browser renders')
                    report=json.loads((stage/'documentation-handler.exe.last-token.json').read_text(encoding='utf-8'))
                    if report['elevated'] or report['administrator_enabled'] or report['user_sid_sha256']!=token['user_sid_sha256'] or report['integrity_rid']!=8192 or report['session_id']!=token['session_id'] or report['authentication_luid']!=token['authentication_luid']:raise RuntimeError('documentation association changed user/elevation')
                finally:shell.SHChangeNotify(0x08000000,0,None,None)
        finally:shell.SHChangeNotify(0x08000000,0,None,None)
    return owned()


def export_captures(output,logs,interactive,receipt):
    """Preserve real transaction images independently of restricted dialogs."""
    m.disposable_runner()
    import struct
    if interactive.get('state')!='passed':raise RuntimeError('transaction captures require completed full-UI proof')
    rows={row['name']:row for row in interactive['transactions']};images=[];output=Path(output);output.mkdir(parents=True,exist_ok=True)
    for operation,suffix,number,title in [('exact-install','progress',8,'Installing insonic'),('repair','finish',9,'insonic repaired'),('remove','finish',10,'insonic removed'),('execution-cancel','cancelled',11,'Setup cancelled'),('failed-upgrade','fatalerror',12,'Setup could not finish'),('change','finish',13,'insonic changed'),('upgrade','finish',14,'insonic updated')]:
        row=rows[operation];filename='context.ui-'+operation+'-'+suffix+'.png'
        item=next((x for x in row['captures'] if x['path']==filename),None)
        if item is None:raise RuntimeError('required native transaction capture is absent: '+filename)
        source=Path(logs)/filename.replace('context.','ordinary-context.',1)
        raw=source.read_bytes()
        if m.digest(raw)!=item['sha256'] or raw[:8]!=b'\x89PNG\r\n\x1a\n' or struct.unpack_from('>II',raw,16)!=(item['width'],item['height']):raise RuntimeError('retained native transaction pixels differ')
        name='installer-'+str(number).zfill(2)+'-'+suffix+'-'+operation+'.png';(output/name).write_bytes(raw)
        images.append({**item,'path':name,'landmark':title,'caption':'Native full-UI '+row['operation']+' transaction on the owned disposable ordinary-user desktop; '+('engine-driven execution progress.' if suffix=='progress' else 'final outcome, native return code '+str(row['code'])+'.'),'fixture_msi_sha256':row['fixture_msi_sha256']})
    sources=['scripts/windows_msi.py','scripts/qualify-windows-msi.py','scripts/qualify_installer_ui.py','LICENSE',m.ICON]
    evidence={'kind':'installer-transactions-ui-capture','schema_version':receipt['schema_version'],'source':{'revision':receipt['revision'],'dirty':False,'files':[{'path':p,'sha256':m.sha(m.ROOT/p)} for p in sources]},'parent_artifact_sha256':receipt['artifact_sha256'],'renderer':{'kind':'windows-installer-full-ui','windows_build':str(__import__('sys').getwindowsversion().build)},'fixture':{'isolated':True,'private_inputs':False,'transaction_proven':True,'native_return_codes':[0,1602,1603],'kind':'unique owned package families, exact payload install/removal plus instrumented native client reporters for bounded maintenance/elections','production_msi_authoring':True,'separate_ordinary_user':True,'private_nonvisible_desktop':False,'disposable_ci_default_desktop':True},'locale':'en-US','images':images}
    (output/'installer-transactions-capture.json').write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    return evidence
