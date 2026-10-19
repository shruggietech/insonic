#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Incident-bound disposable Windows observations; never delivery qualification."""
from concurrent.futures import ThreadPoolExecutor
from threading import Thread
import ctypes
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import time

sys.path.insert(0,str(Path(__file__).resolve().parent))
import windows_package_pipeline as pipeline
from process_tree import ProcessTree, native_webview_environment, native_webview_probe

ROOT=pipeline.ROOT
INCIDENT_REVISION='d44d95b6e97d741a32cbf87323bf78c824c3155b'
INCIDENT_RUN=39097965060
LIMIT=65536


def incident_archive():
    _,receipt,identity=pipeline.read_inputs()
    if (identity.get('revision')!=INCIDENT_REVISION or receipt.get('revision')!=INCIDENT_REVISION
            or receipt.get('platform')!='windows_amd64' or receipt.get('variant')!='desktop'
            or receipt.get('source_dirty') is not False):
        raise ValueError('startup diagnosis requires the exact clean incident Windows desktop')
    return pipeline.input_root()/receipt['archive'],receipt


def job_processes(tree):
    if not tree.job:return {'pids':[],'available':False}
    from ctypes import wintypes
    class Processes(ctypes.Structure):
        _fields_=[('assigned',wintypes.DWORD),('count',wintypes.DWORD),('pids',ctypes.c_size_t*128)]
    value=Processes()
    query=tree.api.QueryInformationJobObject
    query.argtypes=[wintypes.HANDLE,ctypes.c_int,ctypes.c_void_p,wintypes.DWORD,ctypes.c_void_p]
    query.restype=wintypes.BOOL
    result=query(tree.job,3,ctypes.byref(value),ctypes.sizeof(value),None)
    return {'available':bool(result),'assigned':value.assigned,'pids':list(value.pids[:min(value.count,128)]),'error':0 if result else ctypes.get_last_error()}


class BoundedStream:
    def __init__(self):
        self.data=bytearray();self.total=0;self.failed=False

    def drain(self,pipe):
        try:
            while chunk:=pipe.read(16384):
                self.total+=len(chunk)
                self.data.extend(chunk[:max(0,LIMIT-len(self.data))])
        except OSError:
            self.failed=True
        finally:
            pipe.close()


def capture(args,directory,env,timeout):
    started=time.monotonic();timed_out=False;readers=[]
    output,error=BoundedStream(),BoundedStream()
    try:
        with ProcessTree([str(arg) for arg in args],cwd=directory,env=env) as tree:
            for stream,pipe in [(output,tree.process.stdout),(error,tree.process.stderr)]:
                reader=Thread(target=stream.drain,args=(pipe,),daemon=True);reader.start();readers.append(reader)
            try:
                tree.process.wait(timeout=timeout)
                if tree.process.returncode == 0 and native_webview_probe(args,env):
                    if not tree.wait_for_descendants(timeout-(time.monotonic()-started)):
                        raise subprocess.TimeoutExpired(args,timeout)
            except subprocess.TimeoutExpired:
                timed_out=True;tree.kill()
                try:tree.process.wait(timeout=5)
                except subprocess.TimeoutExpired:pass
            members=job_processes(tree)
            code=tree.process.poll()
    finally:
        # Closing the owned job/group releases inherited pipe handles too.
        for reader in readers:reader.join(timeout=5)
    incomplete=any(reader.is_alive() for reader in readers) or output.failed or error.failed
    return {'feature_qualification':False,'exit_code':code if code is not None else -1,'timed_out':timed_out,
            'elapsed_seconds':round(time.monotonic()-started,3),
            'stdout':bytes(output.data).decode('utf-8',errors='replace'),'stderr':bytes(error.data).decode('utf-8',errors='replace'),
            'stdout_bytes':output.total,'stderr_bytes':error.total,'capture_incomplete':incomplete,
            'truncated':output.total>LIMIT or error.total>LIMIT or incomplete,'job_before_close':members}


def save(directory,name,value):
    directory.mkdir(parents=True,exist_ok=True)
    (directory/name).write_text(json.dumps(value,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')


def source_child(args,env):
    # package.child deliberately overlays its environment on the parent. Source
    # diagnosis needs exact selection, so use the explicit ProcessTree boundary.
    result=capture(args,ROOT,env,180)
    if result['exit_code'] or result['timed_out'] or result['truncated']:
        raise RuntimeError('diagnostic source build failed: '+result['stdout']+result['stderr'])
    return result


def probe_build_env(package):
    selected=package.qualify.native_env()
    allowed={'PATH','SYSTEMROOT','WINDIR','TEMP','TMP','USERPROFILE','APPDATA','LOCALAPPDATA','GOPATH','GOMODCACHE','GOCACHE'}
    env={name:value for name,value in os.environ.items() if name.upper() in allowed}
    env.update(selected,GOTOOLCHAIN='local',GOWORK='off',GOENV='off',GOFLAGS='')
    return env


def build_probe_cli(package,cli):
    source_child(['go','build','-tags','system_ladybug','-o',cli,'./cmd/insonic'],probe_build_env(package))
    return package.sha(cli)


def build_probe_gui(package,gui,root):
    # Derive one diagnostic GUI from selected source; the verified incident CLI,
    # dependencies and assets remain explicit inputs. This is no longer an exact
    # incident archive and can never confer package/publication qualification.
    shutil.copytree(root/'help',ROOT/'site/offline',dirs_exist_ok=True)
    package.qualify.assets()
    env=probe_build_env(package)
    node=Path(shutil.which('node',path=env['PATH']) or '')
    if not node.is_file():raise ValueError('selected diagnostic Node executable is missing')
    source_child([node,node.parent/'node_modules/npm/bin/npm-cli.js','--prefix','desktop/frontend','run','build'],env)
    source_child(['go','build','-tags','desktop,production,system_ladybug','-ldflags','-H windowsgui','-o',gui,'./cmd/insonic-desktop'],env)
    return package.sha(gui)


def historical_source():
    source=ROOT/'build/incident-source'
    if source.is_symlink() or source.resolve().parent!=(ROOT/'build').resolve() or not (source/'internal/qualification/dependencies.json').is_file():
        raise ValueError('exact incident source checkout is missing or redirected')
    env={name:value for name,value in os.environ.items() if name.upper() in {'PATH','SYSTEMROOT','WINDIR','TEMP','TMP'}}
    env.update(GIT_TERMINAL_PROMPT='0',GIT_CONFIG_NOSYSTEM='1',GIT_CONFIG_GLOBAL=os.devnull)
    args=['git','-c','core.fsmonitor=false','-C',source]
    revision=source_child([*args,'rev-parse','HEAD'],env)['stdout'].strip()
    dirty=source_child([*args,'status','--porcelain','--untracked-files=no'],env)['stdout'].strip()
    if revision!=INCIDENT_REVISION or dirty:raise ValueError('incident source checkout revision or tracked bytes differ')
    return source


def derive_gui(package,root,location):
    derived=location/'derived-source'
    if derived.resolve().is_relative_to(root.resolve()):raise ValueError('derived source destination overlaps incident package')
    shutil.copytree(root,derived)
    binaries,_=package.package_directories(derived,'windows_amd64','desktop')
    gui=binaries/'insonic-desktop.exe'
    return gui,derived,build_probe_gui(package,gui,derived)


def wait_owned_clis(package,images):
    # All observers have joined. Share one full idle lease, then check every
    # distinct copied image before removing its containing temporary tree.
    last_activity=time.monotonic()
    for image in dict.fromkeys(images):package.wait_for_detached_idle(last_activity,image)


def run():
    pipeline.m.disposable_runner()
    if os.name!='nt':raise RuntimeError('native startup diagnosis requires disposable Windows')
    tool=os.environ.get('GITHUB_SHA','')
    if not re.fullmatch('[a-f0-9]{40}',tool):raise ValueError('exact diagnostic tool revision is required')
    archive,receipt=incident_archive()
    package=pipeline.load('desktop_startup_package','package-desktop.py')
    package.verify_archive_receipt(archive,receipt,expected_source_root=historical_source())
    evidence=ROOT/'build/desktop-startup-diagnostic/evidence'
    summary={'kind':'desktop-startup-diagnostic','feature_qualification':False,'tool_revision':tool,
             'payload_revision':INCIDENT_REVISION,'payload_run':INCIDENT_RUN,'archive_sha256':receipt['archive_sha256'],
             'observations':[]}
    save(evidence,'diagnostic.json',summary)
    with tempfile.TemporaryDirectory(prefix='insonic startup diagnosis ',dir=ROOT/'build') as temporary:
        location=Path(temporary);package.extract_package(archive,location)
        root=package.path_inside(location,receipt['inventory_root'])
        inventory=package.path_inside(location,receipt['inventory_path'])
        package.verify_inventory(root,json.loads(inventory.read_text(encoding='utf-8')))
        binaries,_=package.package_directories(root,receipt['platform'],receipt['variant'])
        gui=binaries/'insonic-desktop.exe'
        env=dict(os.environ,**package.package_environment())
        env['INSONIC_DESKTOP_FIXTURE_DIRECTORY']=str(ROOT/'tests/fixtures/media')
        env['INSONIC_PROCESS_STARTUP_DIAGNOSTICS']='1'
        archive_root=root
        incident=None
        rebuilt=os.environ.get('INSONIC_STARTUP_SOURCE')=='true'
        if rebuilt:
            incident=(gui,root)
            gui,root,digest=derive_gui(package,root,location)
            summary['derived_gui_sha256']=digest
            summary['derived_gui_source_revision']=tool
            summary['boundary']='source-built-gui-with-verified-incident-companions-not-a-qualified-package'
            save(evidence,'diagnostic.json',summary)
        idle=lambda:wait_owned_clis(package,[binaries/'insonic.exe',gui.with_name('insonic.exe')])
        if os.environ.get('INSONIC_STARTUP_LOAD')=='true':
            from cli_contract_qualification import qualify as qualify_contract
            if not rebuilt:raise ValueError('maintained concurrent CLI load requires current source rebuild')
            cli=gui.with_name('insonic.exe');workspace=location/'concurrent CLI workspace'
            summary['derived_cli_sha256']=build_probe_cli(package,cli)
            summary['derived_cli_source_revision']=tool
            summary['boundary']='source-built-gui-and-cli-with-verified-incident-companions-not-a-qualified-package'
            save(evidence,'diagnostic.json',summary)
            for args in [[cli,'workspace','init',workspace,'--json'],[cli,'--workspace',workspace,'credentials','select','session','--json']]:
                initial=capture(args,root,env,30)
                if initial['exit_code'] or initial['timed_out'] or initial['truncated']:
                    save(evidence,'cli-initialization.json',initial)
                    raise RuntimeError('concurrent CLI workspace preparation failed')
            summary['concurrent_cli_contract']=True
            with ThreadPoolExecutor(max_workers=1) as workers:
                future=workers.submit(qualify_contract,cli,workspace,env=env,cwd=root,documentation=True)
                observe_with_load(lambda:observe_native(gui,root,location,env,evidence,summary,rebuilt,incident),future,
                                  idle,evidence)
        else:
            observe_with_load(lambda:observe_native(gui,root,location,env,evidence,summary,rebuilt,incident),None,idle,evidence)
    return summary


def observe_with_load(observe,future,idle,evidence):
    failure=None
    try:
        observe()
    except BaseException as error:
        failure=error
    try:
        if future is not None:
            workload=future.result()
            save(evidence,'concurrent-cli.json',{'feature_qualification':False,'result':workload})
    except BaseException as error:
        save(evidence,'concurrent-cli.json',{'feature_qualification':False,'error':str(error)[:LIMIT]})
        if failure is None:failure=error
    finally:
        try:idle()
        except BaseException as error:
            if failure is None:failure=error
    if failure is not None:raise failure


class PredecessorFailure(RuntimeError):
    pass


def observe_predecessors(gui,root,env,evidence,label):
    for flag,timeout in [('--qualification',30),('--help-qualification',35),('--webview-qualification',90)]:
        with native_webview_environment(env) as profile_env:
            observation=capture([gui,flag],root,profile_env,timeout)
            if observation['exit_code'] or observation['timed_out'] or observation['truncated']:
                # Preserve failed profiles as well as both output streams.
                observation.update(boundary=label+'-archive-predecessor-gui',flag=flag,profile_retained=os.name=='nt')
                print(json.dumps(observation,sort_keys=True),flush=True)
                save(evidence,label+'-'+flag[2:]+'.json',observation)
                raise PredecessorFailure('archive predecessor failed; evidence retained')
        observation.update(boundary=label+'-archive-predecessor-gui',flag=flag)
        name=label+'-'+flag[2:]+'.json';save(evidence,name,observation);print(json.dumps(observation,sort_keys=True),flush=True)


def observe_native(gui,root,location,env,evidence,summary,rebuilt,incident=None):
    historical=None
    if incident:
        original,original_root=incident
        try:
            observe_predecessors(original,original_root,env,evidence,'incident')
        except PredecessorFailure:
            summary['historical_predecessor']='failed-baseline-evidence-retained'
            save(evidence,'diagnostic.json',summary)
        historical=capture([original,'--exit-qualification'],original_root,env,100)
        historical.update(boundary='incident-production-parent')
        save(evidence,'incident-parent.json',historical);print(json.dumps(historical,sort_keys=True),flush=True)
        summary['observations'].append({'file':'incident-parent.json','exit_code':historical['exit_code']})
    label='source' if rebuilt else 'incident'
    observe_predecessors(gui,root,env,evidence,label)
    # Observe the selected production parent. Historical and source-built
    # executables keep separate identities and evidence names.
    parent=capture([gui,'--exit-qualification'],root,env,100)
    parent.update(boundary=label+'-production-parent')
    name=label+'-parent.json';save(evidence,name,parent);print(json.dumps(parent,sort_keys=True),flush=True);summary['observations'].append({'file':name,'exit_code':parent['exit_code']})
    if rebuilt:
        for index in range(11):
            observation=capture([gui,'--exit-qualification'],root,env,100)
            observation.update(boundary='source-built-production-parent',index=index)
            name=f'parent-{index:02d}.json';save(evidence,name,observation);print(json.dumps(observation,sort_keys=True),flush=True)
            summary['observations'].append({'file':name,'exit_code':observation['exit_code']});save(evidence,'diagnostic.json',summary)
            if observation['exit_code'] or observation['timed_out'] or observation['truncated']:raise RuntimeError('source-built native parent failed; inspect retained child evidence')
    for index,action in enumerate([] if rebuilt else ['menu','close','alt-f4']*8):
        fixture=location/'native-exit.json'
        fixture.write_text(json.dumps({'action':action})+'\n',encoding='utf-8',newline='\n')
        observation=capture([gui,'--exit-qualification-child',fixture],root,env,18)
        observation.update(boundary='direct-child-python-job-not-production-go-parent',action=action,index=index)
        name=f'child-{index:02d}-{action}.json';save(evidence,name,observation)
        print(json.dumps(observation,sort_keys=True),flush=True)
        summary['observations'].append({'file':name,'exit_code':observation['exit_code']})
        save(evidence,'diagnostic.json',summary)
        if observation['exit_code'] or observation['timed_out'] or observation['truncated']:
            summary['result']='native-child-failure-observed';save(evidence,'diagnostic.json',summary)
            raise RuntimeError('native child failed; inspect retained separate stdout/stderr')
    summary['result']='bounded-observations-completed-not-a-cause-proof'
    save(evidence,'diagnostic.json',summary)
    if historical:
        summary['historical_parent']='failed-baseline-evidence-retained' if historical['exit_code'] or historical['timed_out'] or historical['truncated'] else 'passed-baseline'
        save(evidence,'diagnostic.json',summary)
    if parent['exit_code'] or parent['timed_out'] or parent['truncated']:raise RuntimeError('production parent failed; boundary observations retained')


if __name__=='__main__':print(json.dumps(run(),sort_keys=True))
