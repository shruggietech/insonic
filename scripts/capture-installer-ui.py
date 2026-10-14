#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Capture actual authored MSI pages on disposable CI, never on the owner host.

The native preview API renders actual Dialog/Control tables. These are authored
installer previews, not observed Windows reputation/UAC or transaction prompts.
Cleanup terminates only the exact owned worker process; no window-close events.
"""
import argparse
import ctypes as C
import ctypes.wintypes as W
import json
import os
from pathlib import Path
import shutil
import struct
import subprocess
import sys
import tempfile
import time
import zlib
import windows_msi as m
PAGES=[('Welcome','installer-01-welcome.png','Install insonic'),('License','installer-02-license.png','Apache License 2.0'),('Choices','installer-03-choices.png','Select clients and shortcuts'),('Location','installer-04-location.png','Choose program directory'),('Ready','installer-05-ready.png','Ready to install'),('Maintenance','installer-06-maintenance.png','Change, repair or remove'),('Finish','installer-07-finish.png','insonic installed')]

def png_bytes(width,height,bgra):
    def chunk(kind,data):return struct.pack('>I',len(data))+kind+data+struct.pack('>I',zlib.crc32(kind+data)&0xffffffff)
    raw=bytearray()
    for y in range(height):
        raw.append(0);line=bgra[y*width*4:(y+1)*width*4]
        for x in range(0,len(line),4):raw.extend((line[x+2],line[x+1],line[x]))
    return b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',width,height,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(raw,6))+chunk(b'IEND',b'')

def worker(path,output):
    m.disposable_runner();u=C.WinDLL('user32',use_last_error=True);g=C.WinDLL('gdi32',use_last_error=True);dll=C.WinDLL('msi',use_last_error=True)
    CB=C.WINFUNCTYPE(W.BOOL,W.HWND,W.LPARAM)
    for name,args,ret in [('EnumWindows',[CB,W.LPARAM],W.BOOL),('EnumChildWindows',[W.HWND,CB,W.LPARAM],W.BOOL),('GetWindowThreadProcessId',[W.HWND,C.POINTER(W.DWORD)],W.DWORD),('GetWindowTextW',[W.HWND,W.LPWSTR,C.c_int],C.c_int),('GetWindowRect',[W.HWND,C.POINTER(W.RECT)],W.BOOL),('GetDC',[W.HWND],W.HDC),('ReleaseDC',[W.HWND,W.HDC],C.c_int),('PrintWindow',[W.HWND,W.HDC,W.UINT],W.BOOL),('PeekMessageW',[C.POINTER(W.MSG),W.HWND,W.UINT,W.UINT,W.UINT],W.BOOL),('TranslateMessage',[C.POINTER(W.MSG)],W.BOOL),('DispatchMessageW',[C.POINTER(W.MSG)],W.LPARAM)]:f=getattr(u,name);f.argtypes=args;f.restype=ret
    for name,args,ret in [('CreateCompatibleDC',[W.HDC],W.HDC),('CreateCompatibleBitmap',[W.HDC,C.c_int,C.c_int],W.HBITMAP),('SelectObject',[W.HDC,W.HANDLE],W.HANDLE),('DeleteObject',[W.HANDLE],W.BOOL),('DeleteDC',[W.HDC],W.BOOL),('GetDIBits',[W.HDC,W.HBITMAP,W.UINT,W.UINT,C.c_void_p,C.c_void_p,W.UINT],C.c_int)]:f=getattr(g,name);f.argtypes=args;f.restype=ret
    for name,args in [('MsiSetInternalUI',[W.UINT,C.c_void_p]),('MsiEnableUIPreview',[W.UINT,C.POINTER(W.UINT)]),('MsiPreviewDialogW',[W.UINT,W.LPCWSTR]),('MsiCloseHandle',[W.UINT])]:f=getattr(dll,name);f.argtypes=args;f.restype=W.UINT
    class Header(C.Structure):_fields_=[('size',W.DWORD),('width',W.LONG),('height',W.LONG),('planes',W.WORD),('bits',W.WORD),('compression',W.DWORD),('image_size',W.DWORD),('x',W.LONG),('y',W.LONG),('used',W.DWORD),('important',W.DWORD)]
    db=m.NativeDatabase(path);preview=W.UINT();dll.MsiSetInternalUI(5,None);db.check(dll.MsiEnableUIPreview(db.handle,C.byref(preview)));images=[]
    def text(hwnd):b=C.create_unicode_buffer(4096);u.GetWindowTextW(hwnd,b,4096);return b.value
    try:
        for page,name,landmark in PAGES:
            db.check(dll.MsiPreviewDialogW(preview,page));msg=W.MSG()
            for _ in range(8):
                while u.PeekMessageW(C.byref(msg),None,0,0,1):u.TranslateMessage(C.byref(msg));u.DispatchMessageW(C.byref(msg))
                time.sleep(.05)
            found=[]
            def owned(hwnd,_):
                pid=W.DWORD();u.GetWindowThreadProcessId(hwnd,C.byref(pid))
                if pid.value==os.getpid() and text(hwnd)=='insonic Setup':found.append(hwnd)
                return True
            u.EnumWindows(CB(owned),0)
            if len(found)!=1:raise RuntimeError('expected exactly one owned authored preview window')
            hwnd=found[0];labels=[]
            def child(ch,_):labels.append(text(ch));return True
            u.EnumChildWindows(hwnd,CB(child),0)
            if not any(landmark in s for s in labels):raise RuntimeError('native page landmark is absent: '+page)
            rect=W.RECT();u.GetWindowRect(hwnd,C.byref(rect));width=rect.right-rect.left;height=rect.bottom-rect.top
            if not 300<=width<=2000 or not 200<=height<=1600:raise RuntimeError('invalid bounded native capture dimensions')
            dc=u.GetDC(hwnd);memory=g.CreateCompatibleDC(dc);bitmap=g.CreateCompatibleBitmap(dc,width,height);old=g.SelectObject(memory,bitmap)
            try:
                if not u.PrintWindow(hwnd,memory,0):raise RuntimeError('owned window PrintWindow failed')
                g.SelectObject(memory,old);header=Header(40,width,-height,1,32,0,width*height*4,0,0,0,0);pixels=C.create_string_buffer(width*height*4)
                if g.GetDIBits(memory,bitmap,0,height,pixels,C.byref(header),0)!=height:raise RuntimeError('native bounded pixel read failed')
                if len(set(pixels.raw[::4]))<8:raise RuntimeError('native screenshot is blank')
                output.joinpath(name).write_bytes(png_bytes(width,height,pixels.raw))
                images.append({'path':name,'sha256':m.sha(output/name),'width':width,'height':height,'landmark':landmark,'caption':'Native Windows Installer authored '+page.lower()+' page preview.'})
            finally:g.SelectObject(memory,old);g.DeleteObject(bitmap);g.DeleteDC(memory);u.ReleaseDC(hwnd,dc)
    finally:dll.MsiPreviewDialogW(preview,None);dll.MsiCloseHandle(preview);db.close()
    print(json.dumps(images))

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
    with tempfile.TemporaryDirectory(prefix='insonic-ci-msi-preview-') as tmp:
        preview=Path(tmp)/'preview.msi';shutil.copyfile(path,preview);db=m.NativeDatabase(preview,write=True)
        try:
            q=m.quote;db.execute('INSERT INTO '+q('Property')+' ('+q('Property')+','+q('Value')+') VALUES (?,?)',['INSTALLDIR','C:\\Users\\InstallerExample\\AppData\\Local\\Programs\\insonic'])
            # Preview has no costed feature election. Keep the authored default
            # Desktop Launch control visible; receipt explicitly identifies this
            # renderer fixture rather than claiming transaction finish evidence.
            db.execute('DELETE FROM '+q('ControlCondition')+' WHERE '+q('Dialog_')+'=?',['Finish'])
            db.check(db.dll.MsiDatabaseCommit(db.handle))
        finally:db.close()
        result=subprocess.run([sys.executable,__file__,'--worker',str(preview),'--output',str(output)],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=45)
        if result.returncode:raise RuntimeError('native authored capture failed: '+result.stderr.decode(errors='replace')[-4000:])
        images=json.loads(result.stdout)
    if source_identity(receipt)!=source:raise ValueError('native capture source changed while rendering')
    evidence={'kind':'installer-ui-capture','schema_version':receipt['schema_version'],'source':source,'artifact_sha256':receipt['artifact_sha256'],'renderer':{'kind':'windows-installer-preview','windows_build':str(sys.getwindowsversion().build),'msi':'native msi.dll Dialog/Control tables','native_webview':False},'fixture':{'isolated':True,'private_inputs':False,'kind':'authored MSI page preview','user':'InstallerExample','selection':'both','conditional_finish_binding':'authored Desktop Launch visible without feature costing'},'locale':'en-US','signing':'unsigned','images':images,'policy_prompts':'not-observed'}
    (output/'installer-capture.json').write_text(json.dumps(evidence,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n');return evidence
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('installer',type=Path,nargs='?');p.add_argument('--receipt',type=Path);p.add_argument('--output',type=Path);p.add_argument('--worker',type=Path);args=p.parse_args();m.disposable_runner()
    if args.worker:worker(args.worker,args.output)
    elif args.installer and args.receipt:capture(args.installer,json.loads(args.receipt.read_text(encoding='utf-8')),args.output)
    else:p.error('installer and --receipt are required')
