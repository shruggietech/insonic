#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Capture actual authored MSI pages on disposable CI, never on the owner host.

The restricted session API renders actual Dialog/Control tables and elected
properties. These fixtures are not observed reputation/UAC or transaction prompts.
Cleanup terminates only the exact owned worker process; no window-close events.
"""
import argparse
import ctypes as C
import ctypes.wintypes as W
import json
import os
from pathlib import Path
import struct
import subprocess
import sys
import threading
import time
import zlib
import windows_msi as m
PAGES=[('Welcome','installer-01-welcome.png','Install insonic'),('License','installer-02-license.png','Apache License 2.0'),('Choices','installer-03-choices.png','Select clients and shortcuts'),('Location','installer-04-location.png','Choose program directory'),('Ready','installer-05-ready.png','Ready to install'),('Maintenance','installer-06-maintenance.png','Change, repair or remove'),('Finish','installer-07-finish.png','insonic installed')]
FIXTURE_DIRECTORY=r'C:\Users\InstallerExample\AppData\Local\Programs\insonic'
FIXTURE_PROPERTIES={'INSTALLDIR':FIXTURE_DIRECTORY,'CLIENT_DESKTOP':'1','CLIENT_CLI':'1','START_MENU':'1','DESKTOP_SHORTCUT':'1','LAUNCH':'1'}

def initialize_restricted_session(dll,session,get_property):
    m.disposable_runner()
    def check(rc):
        if rc:raise RuntimeError('restricted MSI session API failed: '+str(rc))
    for name,value in FIXTURE_PROPERTIES.items():check(dll.MsiSetPropertyW(session,name,value))
    for action in ('CostInitialize','FileCost','CostFinalize'):check(dll.MsiDoActionW(session,action))
    check(dll.MsiSetInstallLevel(session,1))
    states={}
    for feature in ('Core','Desktop','CLI'):
        check(dll.MsiSetFeatureStateW(session,feature,3));installed=C.c_int();planned=C.c_int()
        check(dll.MsiGetFeatureStateW(session,feature,C.byref(installed),C.byref(planned)))
        if planned.value!=3:raise RuntimeError('restricted session did not elect local feature '+feature)
        states[feature]={'installed':installed.value,'action':planned.value}
    properties={name:get_property(name) for name in FIXTURE_PROPERTIES}
    for name,value in FIXTURE_PROPERTIES.items():
        actual=properties[name]
        if name=='INSTALLDIR':
            paths=__import__('ntpath')
            if paths.normcase(paths.normpath(actual))!=paths.normcase(paths.normpath(value)):raise RuntimeError('restricted session directory fixture changed')
        elif actual!=value:raise RuntimeError('restricted session property fixture changed: '+name)
    if get_property('LICENSE_ACCEPTED'):raise RuntimeError('license fixture must remain unchecked')
    return {'properties':properties,'features':states,'license_accepted':False,'open_package_flags':'IGNOREMACHINESTATE','machine_state_changes_permitted':False,'executable_custom_actions_permitted':False}

def png_bytes(width,height,bgra):
    def chunk(kind,data):return struct.pack('>I',len(data))+kind+data+struct.pack('>I',zlib.crc32(kind+data)&0xffffffff)
    raw=bytearray()
    for y in range(height):
        raw.append(0);line=bgra[y*width*4:(y+1)*width*4]
        for x in range(0,len(line),4):raw.extend((line[x+2],line[x+1],line[x]))
    return b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',width,height,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(raw,6))+chunk(b'IEND',b'')

def worker(path,output,page):
    m.disposable_runner()
    if page not in {item[0] for item in PAGES}:raise ValueError('unknown authored page')
    _,name,landmark=next(item for item in PAGES if item[0]==page)
    u=C.WinDLL('user32',use_last_error=True);g=C.WinDLL('gdi32',use_last_error=True);dll=C.WinDLL('msi',use_last_error=True)
    CB=C.WINFUNCTYPE(W.BOOL,W.HWND,W.LPARAM)
    for api,args,result in [('EnumWindows',[CB,W.LPARAM],W.BOOL),('EnumChildWindows',[W.HWND,CB,W.LPARAM],W.BOOL),('GetWindowThreadProcessId',[W.HWND,C.POINTER(W.DWORD)],W.DWORD),('GetWindowTextW',[W.HWND,W.LPWSTR,C.c_int],C.c_int),('GetClassNameW',[W.HWND,W.LPWSTR,C.c_int],C.c_int),('SendMessageTimeoutW',[W.HWND,W.UINT,W.WPARAM,W.LPARAM,W.UINT,W.UINT,C.POINTER(C.c_size_t)],W.LPARAM),('GetWindowRect',[W.HWND,C.POINTER(W.RECT)],W.BOOL),('GetDC',[W.HWND],W.HDC),('ReleaseDC',[W.HWND,W.HDC],C.c_int),('PrintWindow',[W.HWND,W.HDC,W.UINT],W.BOOL)]:
        function=getattr(u,api);function.argtypes=args;function.restype=result
    for api,args,result in [('CreateCompatibleDC',[W.HDC],W.HDC),('CreateCompatibleBitmap',[W.HDC,C.c_int,C.c_int],W.HBITMAP),('SelectObject',[W.HDC,W.HANDLE],W.HANDLE),('DeleteObject',[W.HANDLE],W.BOOL),('DeleteDC',[W.HDC],W.BOOL),('GetDIBits',[W.HDC,W.HBITMAP,W.UINT,W.UINT,C.c_void_p,C.c_void_p,W.UINT],C.c_int)]:
        function=getattr(g,api);function.argtypes=args;function.restype=result
    methods={'MsiSetInternalUI':[W.UINT,C.c_void_p],'MsiOpenPackageExW':[W.LPCWSTR,W.DWORD,C.POINTER(W.UINT)],'MsiSetPropertyW':[W.UINT,W.LPCWSTR,W.LPCWSTR],'MsiGetPropertyW':[W.UINT,W.LPCWSTR,W.LPWSTR,C.POINTER(W.DWORD)],'MsiDoActionW':[W.UINT,W.LPCWSTR],'MsiSetInstallLevel':[W.UINT,C.c_int],'MsiSetFeatureStateW':[W.UINT,W.LPCWSTR,C.c_int],'MsiGetFeatureStateW':[W.UINT,W.LPCWSTR,C.POINTER(C.c_int),C.POINTER(C.c_int)],'MsiCloseHandle':[W.UINT]}
    for api,args in methods.items():function=getattr(dll,api);function.argtypes=args;function.restype=W.UINT
    class Header(C.Structure):_fields_=[('size',W.DWORD),('width',W.LONG),('height',W.LONG),('planes',W.WORD),('bits',W.WORD),('compression',W.DWORD),('image_size',W.DWORD),('x',W.LONG),('y',W.LONG),('used',W.DWORD),('important',W.DWORD)]
    state={'phase':'starting-render-thread','page':page};ready=threading.Event()
    def render():
        session=W.UINT();ole=C.WinDLL('ole32',use_last_error=True);ole.CoInitializeEx.argtypes=[C.c_void_p,W.DWORD];ole.CoInitializeEx.restype=C.c_long;ole.CoUninitialize.argtypes=[];ole.CoUninitialize.restype=None;initialized=False
        def check(rc):
            if rc:raise RuntimeError('native restricted-session API failed: '+str(rc))
        try:
            status=ole.CoInitializeEx(None,2)
            if status not in (0,1):raise RuntimeError('renderer COM initialization failed: '+hex(status&0xffffffff))
            initialized=True;state['phase']='opening-restricted-session'
            if dll.MsiSetInternalUI(5,None)==0:raise RuntimeError('cannot select full authored session UI')
            check(dll.MsiOpenPackageExW(str(path),1,C.byref(session)))
            def get_property(prop):
                value=C.create_unicode_buffer(4096);size=W.DWORD(4096);check(dll.MsiGetPropertyW(session,prop,value,C.byref(size)));return value.value
            state['session']=initialize_restricted_session(dll,session,get_property);state['phase']='authored-dialog';ready.set()
            check(dll.MsiDoActionW(session,page));state['phase']='dialog-returned'
        except BaseException as error:state['error']=str(error);state['phase']='render-failed';ready.set()
        finally:
            if session.value:dll.MsiCloseHandle(session)
            if initialized:ole.CoUninitialize()
    threading.Thread(target=render,name='owned-restricted-msi-dialog',daemon=True).start()
    def text(hwnd):value=C.create_unicode_buffer(4096);u.GetWindowTextW(hwnd,value,4096);return value.value
    def class_name(hwnd):value=C.create_unicode_buffer(256);u.GetClassNameW(hwnd,value,256);return value.value
    def checked(hwnd):
        value=C.c_size_t()
        if not u.SendMessageTimeoutW(hwnd,0x00f0,0,0,2,1000,C.byref(value)):raise RuntimeError('owned checkbox state read timed out')
        return value.value==1
    try:
        if not ready.wait(timeout=15):raise RuntimeError('native session initialization exceeded its bound')
        deadline=time.monotonic()+10;found=[]
        while time.monotonic()<deadline:
            if 'error' in state:raise RuntimeError(state['error'])
            found=[]
            def owned(hwnd,_):
                pid=W.DWORD();u.GetWindowThreadProcessId(hwnd,C.byref(pid))
                if pid.value==os.getpid() and text(hwnd)=='insonic Setup':found.append(hwnd)
                return True
            u.EnumWindows(CB(owned),0)
            if len(found)==1:break
            time.sleep(.025)
        if len(found)!=1:raise RuntimeError('expected exactly one owned authored session window')
        hwnd=found[0];time.sleep(.15);controls=[]
        def child(control,_):
            pid=W.DWORD();u.GetWindowThreadProcessId(control,C.byref(pid))
            if pid.value==os.getpid():controls.append(control)
            return True
        u.EnumChildWindows(hwnd,CB(child),0)
        if not any(landmark in text(control) for control in controls):raise RuntimeError('native page landmark is absent: '+page)
        elections=[]
        labels={'Choices':['Desktop application','CLI integrations (persistent help/setup session)','Create Start menu shortcuts','Create Desktop shortcut for the desktop application'],'Finish':['Launch insonic'],'License':['I accept the Apache License 2.0']}.get(page,[])
        for label in labels:
            matches=[control for control in controls if text(control)==label and class_name(control).casefold()=='button']
            if len(matches)!=1:raise RuntimeError('expected one native checkbox for '+label)
            expected=page!='License';observed=checked(matches[0])
            if observed!=expected:raise RuntimeError('native property-backed checkbox state differs: '+label)
            elections.append({'control_label':label,'checked':observed,'method':'verified session property/BM_GETCHECK'})
        if page=='Location':
            matches=[control for control in controls if class_name(control).casefold()=='edit']
            if len(matches)!=1:raise RuntimeError('expected one native program-directory edit')
            value=text(matches[0]);paths=__import__('ntpath')
            if paths.normcase(paths.normpath(value))!=paths.normcase(paths.normpath(FIXTURE_DIRECTORY)):raise RuntimeError('native property-backed directory differs')
            elections.append({'control_label':'Program directory','value':value,'method':'verified session property/GetWindowText'})
        rect=W.RECT()
        if not u.GetWindowRect(hwnd,C.byref(rect)):raise C.WinError(C.get_last_error())
        width=rect.right-rect.left;height=rect.bottom-rect.top
        if not 300<=width<=2000 or not 200<=height<=1600:raise RuntimeError('invalid bounded native capture dimensions')
        dc=u.GetDC(hwnd);memory=g.CreateCompatibleDC(dc);bitmap=g.CreateCompatibleBitmap(dc,width,height);old=g.SelectObject(memory,bitmap)
        try:
            if not u.PrintWindow(hwnd,memory,0):raise RuntimeError('owned window PrintWindow failed')
            g.SelectObject(memory,old);header=Header(40,width,-height,1,32,0,width*height*4,0,0,0,0);pixels=C.create_string_buffer(width*height*4)
            if g.GetDIBits(memory,bitmap,0,height,pixels,C.byref(header),0)!=height:raise RuntimeError('native bounded pixel read failed')
            if len(set(pixels.raw[::4]))<8:raise RuntimeError('native screenshot is blank')
            output.joinpath(name).write_bytes(png_bytes(width,height,pixels.raw))
            image={'path':name,'sha256':m.sha(output/name),'width':width,'height':height,'landmark':landmark,'caption':'Native Windows Installer authored '+page.lower()+' page in an elected restricted-session fixture.','native_control_elections':elections,'native_session':state['session']}
        finally:g.SelectObject(memory,old);g.DeleteObject(bitmap);g.DeleteDC(memory);u.ReleaseDC(hwnd,dc)
        print(json.dumps(image),flush=True)
        # End only this exact owned process; the modal rendering thread cannot
        # click Finish, execute a launch action or change installed machine state.
        os._exit(0)
    except BaseException as error:
        output.joinpath(name+'.failure.json').write_text(json.dumps({'page':page,'error':str(error),'renderer':state},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
        print(str(error),file=sys.stderr,flush=True);os._exit(1)

def source_identity(receipt):
    m.disposable_runner()
    if receipt.get('source_dirty') is not False or not __import__('re').fullmatch('[0-9a-f]{40}',receipt.get('revision','')):raise ValueError('native capture requires exact clean source identity')
    def git(*args):
        completed=subprocess.run(['git',*args],cwd=m.ROOT,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=15)
        if completed.returncode:raise RuntimeError('native capture source verification failed')
        return completed.stdout.decode('utf-8').strip()
    if git('rev-parse','HEAD')!=receipt['revision'] or git('status','--porcelain','--untracked-files=normal'):raise ValueError('native capture checkout differs from exact clean source')
    files=['scripts/windows_msi.py','scripts/capture-installer-ui.py','LICENSE',m.ICON]
    identities=[{'path':p,'sha256':m.sha(m.ROOT/p)} for p in files]
    if not identities or any(not (m.ROOT/x['path']).is_file() or m.sha(m.ROOT/x['path'])!=x['sha256'] for x in identities):raise ValueError('native capture source files are missing or changed')
    return {'revision':receipt['revision'],'dirty':False,'files':identities}

def capture(path,receipt,output=None):
    m.disposable_runner();source=source_identity(receipt);path=Path(path);output=Path(output or m.ROOT/'build/native/installer-capture');output.mkdir(parents=True,exist_ok=True)
    if m.sha(path)!=receipt['artifact_sha256']:raise ValueError('native capture MSI artifact differs from its exact receipt')
    model=m.installer_model(receipt['payload_inventory'],receipt['payload_receipt']['inventory_sha256'],(m.ROOT/'LICENSE').read_text(encoding='utf-8'));m.verify_model_bytes(path,model)
    images=[]
    for page,_,_ in PAGES:
        result=subprocess.run([sys.executable,__file__,'--worker',str(path),'--page',page,'--output',str(output)],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=35)
        if result.returncode:raise RuntimeError('native authored capture failed: '+result.stderr.decode(errors='replace')[-4000:])
        images.append(json.loads(result.stdout))
    if m.sha(path)!=receipt['artifact_sha256']:raise ValueError('native capture changed the exact MSI artifact')
    if source_identity(receipt)!=source:raise ValueError('native capture source changed while rendering')
    evidence={'kind':'installer-ui-capture','schema_version':receipt['schema_version'],'source':source,'artifact_sha256':receipt['artifact_sha256'],'renderer':{'kind':'windows-installer-restricted-session','windows_build':str(sys.getwindowsversion().build),'msi':'native msi.dll restricted MsiOpenPackageEx session and authored Dialog/Control tables','native_webview':False},'fixture':{'isolated':True,'private_inputs':False,'kind':'authored MSI restricted-session fixture','user':'InstallerExample','selection':'both','control_election_method':'MsiSetProperty/MsiSetInstallLevel/MsiSetFeatureState, native controls read back before capture','defaults_proven':False,'transaction_proven':False,'machine_state_changes_permitted':False,'executable_custom_actions_permitted':False,'conditional_finish_binding':'actual elected Desktop feature action state, no Finish click or launch'},'locale':'en-US','signing':'unsigned','images':images,'policy_prompts':'not-observed'}
    (output/'installer-capture.json').write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n');return evidence
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('installer',type=Path,nargs='?');p.add_argument('--receipt',type=Path);p.add_argument('--output',type=Path);p.add_argument('--worker',type=Path);p.add_argument('--page',choices=[item[0] for item in PAGES]);args=p.parse_args();m.disposable_runner()
    if args.worker:worker(args.worker,args.output,args.page)
    elif args.installer and args.receipt:capture(args.installer,json.loads(args.receipt.read_text(encoding='utf-8')),args.output)
    else:p.error('installer and --receipt are required')
