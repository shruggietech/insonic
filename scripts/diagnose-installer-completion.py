# SPDX-License-Identifier: Apache-2.0
"""CI-only invocation tracing. This produces no product qualification proof."""
import ctypes as C
import ctypes.wintypes as W
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time
import windows_msi as m


def qualifier():
    spec=importlib.util.spec_from_file_location('owned_msi_diagnostic',Path(__file__).with_name('qualify-windows-msi.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module


def observe_invocation(stage,token,quiet_install):
    m.disposable_runner()
    from qualify_installer_ui import NativeUI,fixture_html_registration
    stage=Path(stage);directory=stage/'UI program é & spaces';installer=stage/'ui-old.msi'
    product=dict(m.table_rows(m.CompoundFile(installer).streams,'Property'))['ProductCode']
    handler=stage/'documentation-handler.exe';shutil.copyfile(stage/'launcher-context-fixture/insonic-desktop.exe',handler)
    shutil.copyfile(stage/'launcher-context-fixture/owned-ci-fixture.json',stage/'owned-ci-fixture.json')
    query=C.WinDLL('shlwapi').AssocQueryStringW;query.argtypes=[W.DWORD,C.c_int,W.LPCWSTR,W.LPCWSTR,W.LPWSTR,C.POINTER(W.DWORD)];query.restype=C.c_long
    prog=C.create_unicode_buffer(256);size=W.DWORD(len(prog))
    if query(0,20,'.html',None,prog,C.byref(size)) or not prog.value:raise RuntimeError('diagnostic cannot resolve actual HTML association')
    result={'kind':'instrumented-MSI-invocation-diagnostic','feature_qualification':False,'token':token,'custom_action':next(row for row in m.table_rows(m.CompoundFile(installer).streams,'CustomAction') if row[0]=='OpenDocumentation')}
    def instrument():
        original=directory/'insonic-launcher.exe';backup=directory/'insonic-launcher-real.exe'
        shutil.copyfile(original,backup);result['delegated_production_launcher_sha256']=m.sha(backup)
        shutil.copyfile(directory/'owned-invocation-probe.exe',original)
        result['instrumented_launcher_sha256']=m.sha(original)
    try:
        with fixture_html_registration(stage,token,prog.value,handler):
            shell=C.WinDLL('shell32');shell.SHChangeNotify(0x08000000,0,None,None)
            result['transaction']=NativeUI(stage/'native/context.json').run(installer,directory,'invocation-trace',selection='desktop',docs=True,before_finish=instrument)
            target=directory/'insonic-launcher.exe.invocation.json';deadline=time.monotonic()+20
            while time.monotonic()<deadline:
                if target.is_file():
                    result['invocation']=json.loads(target.read_text(encoding='utf-8'))
                    if result['invocation'].get('stage')=='production-delegate-finished':break
                time.sleep(.05)
            if result.get('invocation',{}).get('stage')!='production-delegate-finished':raise RuntimeError('instrumented MSI action did not finish its exact owned probe')
            result['handler_token_present']=(stage/'documentation-handler.exe.token.json').is_file()
            shell.SHChangeNotify(0x08000000,0,None,None)
    finally:
        if directory.is_dir():
            for item in directory.glob('*.json'):
                if item.name.endswith(('.invocation.json','.token.json','.diagnostic.json')):
                    if item.is_symlink() or item.stat().st_size>65536:raise RuntimeError('owned invocation evidence is redirected or oversized')
                    shutil.copyfile(item,stage/'native'/('context.trace-'+item.name))
        dll=C.WinDLL('msi');dll.MsiQueryProductStateW.argtypes=[W.LPCWSTR];dll.MsiQueryProductStateW.restype=C.c_int
        if dll.MsiQueryProductStateW(product)==5:quiet_install(['/x',product],stage/'native/context.trace-remove.log')
    result['state']='passed'
    # Passed means the trace completed, never that the production MSI passed.
    return result



def rollback_case(stage,name):
    m.disposable_runner();stage=Path(stage);q=qualifier()
    from qualify_installer_ui import NativeUI
    marker=json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'))
    token=q.token_info()
    if stage.resolve().parent!=Path(os.environ['ProgramData']).resolve() or not (stage/'completion-invocation-diagnostic.json').is_file() or marker.get('sid_sha256')!=token['user_sid_sha256'] or token['administrator_enabled'] or token['elevated']:raise RuntimeError('rollback diagnostic is not the exact owned ordinary account')
    if name not in ('install','failed-upgrade'):raise RuntimeError('unknown owned rollback diagnostic case')
    installer=stage/('ui-old.msi' if name=='install' else 'ui-bad.msi')
    row=NativeUI(stage/'native/context.json').run(installer,stage/'UI program é & spaces','rollback-'+name,selection='cli',operation='install' if name=='install' else 'upgrade',expected=0 if name=='install' else 1603)
    row['token']=token
    (stage/'native'/('context.rollback-'+name+'.json')).write_text(json.dumps(row,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    return row


def observe_rollback(stage,token,quiet_install):
    m.disposable_runner();stage=Path(stage)
    isolated=os.environ.get('INSONIC_INSTALLER_DIAGNOSTIC_MODE')=='rollback-isolated'
    result={'kind':'owned-native-rollback-comparison','feature_qualification':False,'token':token,'isolated_transaction_processes':isolated,'transactions':[]}
    products=[dict(m.table_rows(m.CompoundFile(stage/name).streams,'Property'))['ProductCode'] for name in ('ui-old.msi','ui-bad.msi')]
    dll=C.WinDLL('msi');dll.MsiQueryProductStateW.argtypes=[W.LPCWSTR];dll.MsiQueryProductStateW.restype=C.c_int
    try:
        for name in ('install','failed-upgrade'):
            if isolated:
                command=[str(Path(sys.executable).with_name('python.exe')),str(Path(__file__)), '--rollback-case',str(stage),name]
                done=subprocess.run(command,cwd=stage,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=75)
                (stage/'native'/('context.rollback-'+name+'-stream.log')).write_bytes(done.stdout+done.stderr)
                if done.returncode:raise RuntimeError('isolated rollback diagnostic child failed '+str(done.returncode))
                row=json.loads((stage/'native'/('context.rollback-'+name+'.json')).read_text(encoding='utf-8'))
            else:row=rollback_case(stage,name)
            if row['token']!=token:raise RuntimeError('rollback comparison changed the ordinary token')
            row['registered_states']={product:int(dll.MsiQueryProductStateW(product)) for product in products};result['transactions'].append(row)
        directory=stage/'UI program é & spaces';model=json.loads((stage/'ui-model.json').read_text(encoding='utf-8'))
        result['restored_old_payload']=all((directory/item['path']).is_file() and m.sha(directory/item['path'])==item['sha256'] for item in model['payload'] if item['feature']!='Desktop')
        result['state']='passed'
        return result
    finally:
        for product in products:
            if dll.MsiQueryProductStateW(product) in (1,5):quiet_install(['/x',product],stage/'native'/('context.rollback-cleanup-'+product.strip('{}')+'.log'))


WRAPPER=r''' if filepath.Base(exe)=="insonic-launcher.exe" {
  marker,e:=os.ReadFile(filepath.Join(filepath.Dir(directory),"completion-invocation-diagnostic.json"));if e!=nil||string(marker)!="{\"qualification\":false}" {fatal("DiagnosticScope",fmt.Errorf("not the owned invocation fixture"))}
  cwd,_:=os.Getwd()
  trace:=map[string]any{"kind":"MSI-EXE-argv-and-delegate-trace","feature_qualification":false,"token":observed,"args":os.Args,"raw_command_line":windows.UTF16PtrToString(windows.GetCommandLine()),"working_directory":cwd}
  writeTrace:=func(){data,_:=json.Marshal(trace);_ = os.WriteFile(filepath.Join(directory,"insonic-launcher.exe.invocation.json"),append(data,'\n'),0600)}
  trace["stage"]="before-window-object-probe";writeTrace()
  defer func(){if failure:=recover();failure!=nil {trace["stage"]="probe-panicked";trace["panic"]=fmt.Sprint(failure);writeTrace()}}()
  var startup windows.StartupInfo
  startup.Cb=uint32(unsafe.Sizeof(startup))
  windows.NewLazySystemDLL("kernel32.dll").NewProc("GetStartupInfoW").Call(uintptr(unsafe.Pointer(&startup)))
  trace["startup_desktop"]=windows.UTF16PtrToString(startup.Desktop);writeTrace()
  user32:=windows.NewLazySystemDLL("user32.dll")
  objectName:=func(handle uintptr) map[string]any {
   buffer:=make([]uint16,512);var needed uint32
   ok,_,nativeError:=user32.NewProc("GetUserObjectInformationW").Call(handle,2,uintptr(unsafe.Pointer(&buffer[0])),uintptr(len(buffer)*2),uintptr(unsafe.Pointer(&needed)))
   if ok==0 {return map[string]any{"win32_error":fmt.Sprint(nativeError)}}
   return map[string]any{"name":windows.UTF16ToString(buffer)}
  }
  if err:=user32.Load();err!=nil {trace["user32_load_error"]=err.Error()} else {
   station,_,_:=user32.NewProc("GetProcessWindowStation").Call()
   desktop,_,_:=user32.NewProc("GetThreadDesktop").Call(uintptr(windows.GetCurrentThreadId()))
   trace["window_station"]=objectName(station);trace["desktop"]=objectName(desktop)
  }
  trace["stage"]="before-production-delegate";writeTrace()
  context,cancel:=context.WithTimeout(context.Background(),15*time.Second);defer cancel()
  command:=exec.CommandContext(context,filepath.Join(directory,"insonic-launcher-real.exe"),os.Args[1:]...)
  command.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000}
  var stdout,stderr bytes.Buffer;command.Stdout=&stdout;command.Stderr=&stderr
  err:=command.Run();trace["stdout"]=stdout.String();trace["stderr"]=stderr.String();trace["timed_out"]=context.Err()!=nil
  code:=0;if err!=nil {code=1;if exit,ok:=err.(*exec.ExitError);ok {code=exit.ExitCode()};trace["error"]=err.Error()}
  trace["exit_code"]=code;trace["stage"]="production-delegate-finished";writeTrace();os.Exit(code)
 }
'''


def main():
    m.disposable_runner();q=qualifier();work=m.ROOT/'build/completion-invocation-diagnostic';work.mkdir(parents=True,exist_ok=False)
    fixture=work/'launcher-context-fixture';fixture.mkdir();evidence=work/'evidence';evidence.mkdir()
    env=dict(os.environ,CGO_ENABLED='0')
    def build(target,source):
        done=subprocess.run(['go','build','-ldflags','-H windowsgui','-o',str(target),source],cwd=m.ROOT,env=env,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=60)
        if done.returncode:raise RuntimeError(done.stderr.decode(errors='replace')[-4000:])
    build(fixture/'insonic-launcher.exe','./cmd/insonic-launcher')
    source=work/'token-only.go';source.write_text(q.LAUNCH_REPORTER_SOURCE.replace('if filepath.Base(exe)=="documentation-handler.exe" {\n','if filepath.Base(exe)=="documentation-handler.exe" && false {\n'),encoding='utf-8',newline='\n')
    build(fixture/'insonic-desktop.exe',str(source));shutil.copyfile(fixture/'insonic-desktop.exe',fixture/'insonic.exe')
    wrapper=work/'invocation-wrapper.go';body=q.LAUNCH_REPORTER_SOURCE.replace(' "crypto/sha256"',' "context"\n "bytes"\n "time"\n "crypto/sha256"').replace(' diagnostic("token-recorded",nil)',' diagnostic("token-recorded",nil)\n'+WRAPPER)
    wrapper.write_text(body,encoding='utf-8',newline='\n');build(fixture/'owned-invocation-probe.exe',str(wrapper))
    for name in ('LICENSE','NOTICE'):shutil.copyfile(m.ROOT/name,fixture/name)
    (fixture/'insonic-companions.json').write_text('{}',encoding='utf-8',newline='\n');(fixture/'insonic-session.cmd').write_text('@echo off\nexit /b 0\n',encoding='utf-8',newline='\n')
    (fixture/'owned-ci-fixture.json').write_text('insonic disposable launcher context fixture\n',encoding='utf-8',newline='\n')
    help=fixture/'help';help.mkdir();version=json.loads((m.ROOT/'package.json').read_text(encoding='utf-8'))['version']
    (help/'index.html').write_text('<!doctype html><title>Owned invocation diagnostic</title>',encoding='utf-8',newline='\n')
    (help/'offline-manifest.json').write_text(json.dumps({'version':version,'entrypoint':'index.html','public_landing_included':False}),encoding='utf-8',newline='\n')
    files=[{'path':p.relative_to(fixture).as_posix(),'sha256':m.sha(p),'size_bytes':p.stat().st_size} for p in sorted(fixture.rglob('*')) if p.is_file()]
    inventory={'schema_version':version,'platform':'windows_amd64','variant':'desktop','files':files};raw=m.inventory_bytes(inventory)
    (fixture/'package-inventory.json').write_bytes(raw)
    model=m.installer_model(inventory,m.digest(raw),(fixture/'LICENSE').read_text(encoding='utf-8'),upgrade=m.guid('invocation-diagnostic/'+__import__('uuid').uuid4().hex))
    installer=work/'diagnostic.msi';m.author(installer,fixture,model)
    kind=os.environ.get('INSONIC_INSTALLER_DIAGNOSTIC_KIND','completion')
    if kind not in ('completion','rollback'):raise RuntimeError('unknown diagnostic kind')
    modes=['completion'] if kind=='completion' else ['rollback-reused','rollback-isolated']
    results=[]
    for mode in modes:
        os.environ['INSONIC_INSTALLER_DIAGNOSTIC_MODE']=mode
        folder=evidence/mode;folder.mkdir()
        result=q.ordinary_context(installer,work/'installed',folder/'ordinary-context.json',model,diagnostic_only=True)
        results.append({'mode':mode,'diagnostic':result['diagnostic']})
    print(json.dumps({'diagnostic_trace_complete':True,'feature_qualification':False,'results':results},indent=2))


if __name__=='__main__':
    if len(sys.argv)==4 and sys.argv[1]=='--rollback-case':rollback_case(sys.argv[2],sys.argv[3])
    else:main()
