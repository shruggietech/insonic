#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Incident-bound disposable Windows observations; never delivery qualification."""
from concurrent.futures import ThreadPoolExecutor
from contextlib import contextmanager
from threading import Event, Lock, Thread
import ctypes
import hashlib
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
ABI_CONTROL_FILE_SHA256='788ed4d41dc722f387c67e0ba873608ae5ec5debd338316ba1e81440021782c5'
CSS_CONTROL_SHA256='b737892a3b91fec94430eb9d779b0abba6b8b71e0daa8f5419f126179d00bc0b'
SCRIPT_CONTROL_SHA256='fce52dc675bb0a1fd360324d143a8ed48824e29f5d92fdaf6fadf55e39a35f79'
CSS_CONTROL_BYTES=402310


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


def capture(args,directory,env,timeout,*,require_release=False,startup_hide=False,observe_resources=False,resource_origin=None):
    started=time.monotonic();timed_out=False;readers=[]
    timing={}
    resource_api=None;resources=None
    if observe_resources:
        resource_origin=started if resource_origin is None else resource_origin
        try:
            import windows_native_resources as resource_api
            resources={'before':resource_api.system_snapshot(resource_origin)}
        except Exception:resources={'available':False,'reason':'observer-unavailable'}
    # Endpoint observation must not consume the original native release budget.
    if observe_resources:started=time.monotonic()
    output,error=BoundedStream(),BoundedStream()
    try:
        selected=[str(arg) for arg in args]
        if startup_hide:selected=[sys.executable,str(Path(__file__).with_name('diagnose-native-launch.py')),*selected]
        with ProcessTree(selected,cwd=directory,env=env) as tree:
            for stream,pipe in [(output,tree.process.stdout),(error,tree.process.stderr)]:
                reader=Thread(target=stream.drain,args=(pipe,),daemon=True);reader.start();readers.append(reader)
            try:
                tree.process.wait(timeout=timeout)
                timing['root_exit_elapsed_seconds']=round(time.monotonic()-started,6)
                if tree.process.returncode == 0 and (require_release or native_webview_probe(args,env)):
                    release_started=time.monotonic()
                    remaining=max(0,timeout-(release_started-started))
                    timing['release_budget_seconds']=round(remaining,6)
                    released=tree.wait_for_descendants(remaining)
                    timing['release_wait_elapsed_seconds']=round(time.monotonic()-release_started,6)
                    if not released:
                        raise subprocess.TimeoutExpired(args,timeout)
            except subprocess.TimeoutExpired:
                timed_out=True;tree.kill()
                try:tree.process.wait(timeout=5)
                except subprocess.TimeoutExpired:pass
            members=job_processes(tree)
            code=tree.process.poll()
            if observe_resources and resource_api is not None:
                try:
                    resources.update(after=resource_api.system_snapshot(resource_origin),
                        owned=(resource_api.owned_snapshot(tree.job,members.get('pids',[])) if members.get('available')
                               else {'available':False,'reason':'owned-enumeration-unavailable'}),
                        point='after-timeout-kill' if timed_out else 'after-root-before-job-close')
                    resources['cpu_interval']=resource_api.cpu_interval(resources['before'],resources['after'])
                except Exception:resources.update(available=False,reason='observer-unavailable')
    finally:
        # Closing the owned job/group releases inherited pipe handles too.
        for reader in readers:reader.join(timeout=5)
    incomplete=any(reader.is_alive() for reader in readers) or output.failed or error.failed
    result={'feature_qualification':False,'exit_code':code if code is not None else -1,'timed_out':timed_out,
            'elapsed_seconds':round(time.monotonic()-started,3),
            'stdout':bytes(output.data).decode('utf-8',errors='replace'),'stderr':bytes(error.data).decode('utf-8',errors='replace'),
            'stdout_bytes':output.total,'stderr_bytes':error.total,'capture_incomplete':incomplete,
            'truncated':output.total>LIMIT or error.total>LIMIT or incomplete,'job_before_close':members,
            'timing':timing,
            'owned_release':bool(require_release and not timed_out and code==0 and members.get('available') and members.get('assigned')==0 and not members.get('pids'))}
    if observe_resources:result['resources']=resources
    return result


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


@contextmanager
def css_size_control_assets():
    # Challenge the script/size correlation without changing executable code,
    # MIME, routes or native response handling. Restore generated bytes even
    # when compilation fails; this output can never qualify a product package.
    bundle=ROOT/'desktop/assets/bundle'
    selected=bundle/'product.css'
    original=selected.read_bytes()
    script=(bundle/'app.js').read_bytes()
    if (len(original)!=6784 or hashlib.sha256(original).hexdigest()!=CSS_CONTROL_SHA256
            or len(script)!=CSS_CONTROL_BYTES or hashlib.sha256(script).hexdigest()!=SCRIPT_CONTROL_SHA256):
        raise ValueError('CSS size control requires the exact E007 CSS and script bytes')
    padded=original+b'/*'+b' '*(CSS_CONTROL_BYTES-len(original)-5)+b'*/\n'
    receipt={'kind':'initial-css-size-control','feature_qualification':False,
             'class':'product-css','original_bytes':len(original),
             'original_sha256':hashlib.sha256(original).hexdigest(),
             'padded_bytes':len(padded),'padded_sha256':hashlib.sha256(padded).hexdigest(),
             'unchanged_app_script_bytes':len(script),'unchanged_app_script_sha256':hashlib.sha256(script).hexdigest()}
    try:
        selected.write_bytes(padded)
        yield receipt
    finally:
        selected.write_bytes(original)


def build_probe_gui(package,gui,root,*,overlay=None,css_control=None):
    # Derive one diagnostic GUI from selected source; the verified incident CLI,
    # dependencies and assets remain explicit inputs. This is no longer an exact
    # incident archive and can never confer package/publication qualification.
    shutil.copytree(root/'help',ROOT/'site/offline',dirs_exist_ok=True)
    package.qualify.assets()
    env=probe_build_env(package)
    node=Path(shutil.which('node',path=env['PATH']) or '')
    if not node.is_file():raise ValueError('selected diagnostic Node executable is missing')
    source_child([node,node.parent/'node_modules/npm/bin/npm-cli.js','--prefix','desktop/frontend','run','build'],env)
    args=['go','build','-tags','desktop,production,system_ladybug','-ldflags','-H windowsgui','-o',gui,'./cmd/insonic-desktop']
    if overlay is not None:args[2:2]=['-overlay',overlay]
    if css_control is None:
        source_child(args,env)
    else:
        with css_size_control_assets() as receipt:
            css_control.update(receipt)
            source_child(args,env)
    return package.sha(gui)


def historical_abi_overlay(location):
    # A diagnostic-only Go overlay supplies the exact historical upstream file.
    # The working source remains corrected; no arbitrary overlay is accepted.
    selected=ROOT/'third_party/go-webview2/pkg/edge/ICoreWebView2Controller3.go'
    fixed=selected.read_bytes()
    after=b'uintptr(boolToInt(value))'
    before=b'uintptr(unsafe.Pointer(&value))'
    if fixed.count(after)!=1:raise ValueError('corrected BOOL source differs from the selected upstream method')
    historical=fixed.replace(after,before)
    if hashlib.sha256(historical).hexdigest()!=ABI_CONTROL_FILE_SHA256:
        raise ValueError('historical BOOL control source checksum differs')
    location.mkdir(parents=True,exist_ok=True)
    control=location/'ICoreWebView2Controller3.go'
    control.write_bytes(historical)
    overlay=location/'overlay.json'
    overlay.write_text(json.dumps({'Replace':{str(selected.resolve()):str(control.resolve())}})+'\n',encoding='utf-8',newline='\n')
    return overlay


def observe_abi_comparison(package,gui,root,location,env,evidence,summary,*,startup_hide):
    # Two predeclared arms, each with its own fixed budget. Stop each arm at its
    # first failure; a failed corrected arm remains a failed workflow. The old
    # ABI control is never used in a product package or qualification claim.
    overlay=historical_abi_overlay(location/'abi-control-source')
    control_root=location/'abi-control-payload'
    if control_root.resolve().is_relative_to(root.resolve()):raise ValueError('ABI control payload overlaps selected source payload')
    shutil.copytree(root,control_root)
    control_binaries,_=package.package_directories(control_root,'windows_amd64','desktop')
    control_gui=control_binaries/gui.name
    digest=build_probe_gui(package,control_gui,control_root,overlay=overlay)
    summary['abi_control']={'historical_file_sha256':ABI_CONTROL_FILE_SHA256,
                            'overlay_sha256':package.sha(overlay),'gui_sha256':digest,
                            'source_revision':summary['derived_gui_source_revision'],
                            'feature_qualification':False,'difference':'one historical native BOOL argument'}
    summary['arms']=[]
    for label,image,cwd in [('old-ABI-control',control_gui,control_root),('corrected',gui,root)]:
        arm={'arm':label,'observations':[],'feature_qualification':False}
        summary['arms'].append(arm)
        owned=location/label;owned.mkdir()
        try:
            observe_cold_start(image,cwd,owned,env,evidence/label,arm,
                               startup_hide=startup_hide,sample_seconds=150,sample_limit=120)
        except RuntimeError:
            if arm.get('result')!='cold-start-failure-observed':raise
        save(evidence,'diagnostic.json',summary)
        if label=='corrected' and arm.get('result')=='cold-start-failure-observed':
            summary['result']='corrected-startup-failure-observed'
            save(evidence,'diagnostic.json',summary)
            raise RuntimeError('corrected startup failed; comparison and first failure retained')
    summary['result']='bounded-ABI-comparison-not-a-cause-proof'
    save(evidence,'diagnostic.json',summary)


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


def derive_gui(package,root,location,*,css_control=None):
    derived=location/'derived-source'
    if derived.resolve().is_relative_to(root.resolve()):raise ValueError('derived source destination overlaps incident package')
    shutil.copytree(root,derived)
    binaries,_=package.package_directories(derived,'windows_amd64','desktop')
    gui=binaries/'insonic-desktop.exe'
    digest=build_probe_gui(package,gui,derived) if css_control is None else build_probe_gui(package,gui,derived,css_control=css_control)
    return gui,derived,digest


def wait_owned_clis(package,images):
    # All observers have joined. Share one full idle lease, then check every
    # distinct copied image before removing its containing temporary tree.
    last_activity=time.monotonic()
    for image in dict.fromkeys(images):package.wait_for_detached_idle(last_activity,image)


class LoadTimeline:
    """Bounded metadata from actual worker/capture entry and exit, with one clock."""
    PHASES=frozenset({'help','operator-feedback','credentials','model-acquisition',
                     'parallel-regressions','media-lifecycle','query-contracts',
                     'documented-workflows','provider-contracts','join-regressions','complete'})

    def __init__(self,*,clock=time.monotonic,gate=False):
        self._clock=clock;self._origin=clock();self._lock=Lock()
        self._start=None;self._end=None;self._state='pending';self._phase='not-started'
        self._events=[];self._dropped=0;self._cases=[]
        self._gate_enabled=gate;self._gate_event=Event();self._gate_signaled=None;self._gate_released=None

    def _elapsed(self):return round(self._clock()-self._origin,6)

    def _snapshot(self):
        return {'elapsed_seconds':self._elapsed(),'phase':self._phase,'worker_state':self._state}

    def worker_start(self):
        with self._lock:
            if self._state!='pending':raise ValueError('load worker already started')
            self._start=self._elapsed();self._state='running';self._phase='starting'

    def worker_end(self,passed):
        with self._lock:
            if self._state!='running':raise ValueError('load worker is not running')
            self._end=self._elapsed();self._state='passed' if passed else 'failed'
            if self._gate_enabled:self._gate_event.set()

    def phase(self,label):
        if label not in self.PHASES:raise ValueError('unknown load phase label')
        with self._lock:
            if self._state!='running':raise ValueError('load phase outside running worker')
            self._phase=label
            if self._gate_enabled and label=='media-lifecycle' and self._gate_signaled is None:
                self._gate_signaled=self._elapsed();self._gate_event.set()
            if len(self._events)<16:self._events.append({'phase':label,'elapsed_seconds':self._elapsed()})
            else:self._dropped+=1

    def wait_gate(self):
        if not self._gate_enabled:raise ValueError('load gate was not elected')
        if not self._gate_event.wait(90):raise RuntimeError('load phase gate deadline')
        with self._lock:
            if self._gate_signaled is None:raise RuntimeError('load worker ended before the phase gate')
            if self._state!='running' or self._phase!='media-lifecycle':raise RuntimeError('load phase gate expired')
            self._gate_released=self._elapsed()

    @property
    def origin(self):return self._origin

    def case_begin(self,index):
        with self._lock:
            if index!=len(self._cases) or not 0<=index<240:raise ValueError('load timeline case bound/order differs')
            self._cases.append({'index':index,'begin':self._snapshot(),'end':None})
            return index

    def _case_record(self,case):
        value={'index':case['index'],'begin':dict(case['begin']),
               'end':dict(case['end']) if case['end'] is not None else None}
        overlap=0
        if self._start is not None and case['end'] is not None:
            end=case['end']['elapsed_seconds']
            overlap=max(0,min(end,self._end if self._end is not None else end)
                        -max(case['begin']['elapsed_seconds'],self._start))
        value['overlap_seconds']=round(overlap,6)
        return value

    def case_end(self,index):
        with self._lock:
            case=self._cases[index]
            if case['end'] is not None:raise ValueError('load timeline case already ended')
            case['end']=self._snapshot()
            return self._case_record(case)

    def receipt(self):
        with self._lock:
            result={'kind':'desktop-startup-load-timeline','feature_qualification':False,
                    'clock':'monotonic-seconds-since-observer-origin',
                    'phase_scope':'contract-main-thread-with-joined-parallel-regressions',
                    'worker':{'start_elapsed_seconds':self._start,'end_elapsed_seconds':self._end,'state':self._state},
                    'phase_events':[dict(event) for event in self._events],'dropped_phase_events':self._dropped,
                    'cases':[self._case_record(case) for case in self._cases]}
            if self._gate_enabled:result['gate']={'phase':'media-lifecycle','wait_limit_seconds':90,
                'signal_elapsed_seconds':self._gate_signaled,'released_elapsed_seconds':self._gate_released,
                'prior_gui_launches':0,'child_limit':1}
            return result


def run_load(timeline,qualify,*args,**kwargs):
    # Record real thread entry, rather than assuming submit means active overlap.
    timeline.worker_start()
    try:
        result=qualify(*args,**kwargs,progress=timeline.phase)
    except BaseException:
        timeline.worker_end(False)
        raise
    timeline.worker_end(True)
    return result


def run():
    pipeline.m.disposable_runner()
    if os.name!='nt':raise RuntimeError('native startup diagnosis requires disposable Windows')
    tool=os.environ.get('GITHUB_SHA','')
    if not re.fullmatch('[a-f0-9]{40}',tool):raise ValueError('exact diagnostic tool revision is required')
    cold=os.environ.get('INSONIC_STARTUP_COLD')=='true'
    startup_hide=os.environ.get('INSONIC_STARTUP_HIDE')=='true'
    abi_control=os.environ.get('INSONIC_STARTUP_ABI_CONTROL')=='true'
    browser_observer=os.environ.get('INSONIC_STARTUP_BROWSER')=='true'
    css_control=os.environ.get('INSONIC_STARTUP_CSS_CONTROL')=='true'
    load_gate=os.environ.get('INSONIC_STARTUP_LOAD_GATE')=='true'
    if load_gate and not css_control:
        raise ValueError('load phase gate requires the exact E008 CSS control')
    if css_control and (not browser_observer or not cold or not startup_hide or os.environ.get('INSONIC_STARTUP_SOURCE')!='true'
                        or os.environ.get('INSONIC_STARTUP_LOAD')!='true' or abi_control):
        raise ValueError('CSS size control requires current cold hidden source, browser observation and concurrent load without ABI control')
    if browser_observer and (not cold or os.environ.get('INSONIC_STARTUP_SOURCE')!='true' or abi_control):
        raise ValueError('browser observation requires current cold source without historical ABI control')
    if startup_hide and not cold:raise ValueError('startup hide comparison requires cold startup observation')
    if cold and os.environ.get('INSONIC_STARTUP_SOURCE')!='true':raise ValueError('cold startup observation requires current source')
    if abi_control and (not cold or not startup_hide or os.environ.get('INSONIC_STARTUP_LOAD')=='true'):
        raise ValueError('ABI comparison requires current cold source, startup hide and no concurrent CLI load')
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
        env['INSONIC_STARTUP_BROWSER']='true' if browser_observer else 'false'
        archive_root=root
        incident=None
        rebuilt=os.environ.get('INSONIC_STARTUP_SOURCE')=='true'
        if rebuilt:
            incident=(gui,root)
            control_receipt={} if css_control else None
            gui,root,digest=derive_gui(package,root,location) if control_receipt is None else derive_gui(package,root,location,css_control=control_receipt)
            if control_receipt is not None:summary['css_size_control']=control_receipt
            summary['derived_gui_sha256']=digest
            summary['derived_gui_source_revision']=tool
            summary['boundary']='source-built-gui-with-verified-incident-companions-not-a-qualified-package'
            save(evidence,'diagnostic.json',summary)
        idle=lambda:wait_owned_clis(package,[binaries/'insonic.exe',gui.with_name('insonic.exe')])
        timeline=None
        def observe():
            if load_gate:timeline.wait_gate()
            if cold:
                return observe_cold_start(gui,root,location,env,evidence,summary,startup_hide=startup_hide,timeline=timeline,
                    **({'sample_limit':1,'observe_resources':True} if load_gate else {}))
            return observe_native(gui,root,location,env,evidence,summary,rebuilt,incident)
        if abi_control:
            observe=lambda:observe_abi_comparison(package,gui,root,location,env,evidence,summary,startup_hide=startup_hide)
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
            timeline=LoadTimeline(gate=load_gate)
            with ThreadPoolExecutor(max_workers=1) as workers:
                future=workers.submit(run_load,timeline,qualify_contract,cli,workspace,env=env,cwd=root,documentation=True)
                observe_with_load(observe,future,idle,evidence,timeline=timeline)
        else:
            observe_with_load(observe,None,idle,evidence)
    return summary


def observe_with_load(observe,future,idle,evidence,*,timeline=None):
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
        if timeline is not None:
            try:save(evidence,'concurrent-load-timeline.json',timeline.receipt())
            except BaseException as error:
                if failure is None:failure=error
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


def observe_cold_start(gui,root,location,env,evidence,summary,*,startup_hide=False,sample_seconds=300,sample_limit=240,timeline=None,observe_resources=False):
    # Diagnostic sampling only. Each real close child owns a fresh profile and
    # must naturally release its job before another launch. Stop on first failure.
    if observe_resources and (timeline is None or sample_limit!=1):
        raise ValueError('resource endpoints require the single load-gated case timeline')
    summary.update(boundary='source-cold-close-startup-not-package-qualification',sample_limit=sample_limit,sampling_seconds=sample_seconds,startup_hide=startup_hide)
    deadline=time.monotonic()+sample_seconds
    summary['result']='cold-start-no-failure-observed-not-a-cause-proof'
    for index in range(sample_limit):
        if deadline-time.monotonic()<18:
            summary['result']='cold-start-sampling-budget-exhausted-not-a-cause-proof'
            break
        directory=location/f'cold-{index:03d}';directory.mkdir()
        fixture=directory/'native-exit.json'
        fixture.write_text(json.dumps({'action':'close'})+'\n',encoding='utf-8',newline='\n')
        case=timeline.case_begin(index) if timeline is not None else None
        try:
            options={'startup_hide':True} if startup_hide else {}
            if observe_resources:options.update(observe_resources=True,resource_origin=timeline.origin)
            observation=capture([gui,'--exit-qualification-child',fixture],root,env,18,require_release=True,**options)
        finally:
            if timeline is not None:load_case=timeline.case_end(case)
        if timeline is not None:observation['concurrent_load']=load_case
        observation.update(boundary=summary['boundary'],action='close',profile='cold',index=index,startup_hide=startup_hide,arm=summary.get('arm','selected-source'))
        try:
            result=json.loads(observation['stdout'])
        except (ValueError,TypeError):
            result={}
        if not isinstance(result,dict):result={}
        passed=(not observation['exit_code'] and not observation['timed_out'] and not observation['truncated']
                and not observation['capture_incomplete'] and observation['owned_release'] is True
                and result.get('native_exit')=='passed' and result.get('action')=='close' and result.get('playback')=='not-selected')
        name=f'cold-{index:03d}.json'
        save(evidence,name,observation);print(json.dumps(observation,sort_keys=True),flush=True)
        summary['observations'].append({'file':name,'exit_code':observation['exit_code'],'observed_pass':passed})
        if not passed:summary['result']='cold-start-failure-observed'
        save(evidence,'diagnostic.json',summary)
        if not passed:raise RuntimeError('cold startup failed; first failure evidence retained')
    save(evidence,'diagnostic.json',summary)


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
    predecessor_failure=None
    try:
        observe_predecessors(gui,root,env,evidence,label)
    except PredecessorFailure as error:
        if not rebuilt:raise
        # This workflow is diagnostic only. Retain the failed prerequisite and
        # still observe actual Exit; later passing observations cannot clear it.
        predecessor_failure=error
        summary[label+'_predecessor']='failed-evidence-retained'
        save(evidence,'diagnostic.json',summary)
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
            if observation['exit_code'] or observation['timed_out'] or observation['truncated']:raise RuntimeError('source-built native parent failed; inspect retained child evidence') from predecessor_failure
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
    if parent['exit_code'] or parent['timed_out'] or parent['truncated']:raise RuntimeError('production parent failed; boundary observations retained') from predecessor_failure
    if predecessor_failure is not None:
        summary['result']='predecessor-failure-observed'
        save(evidence,'diagnostic.json',summary)
        raise predecessor_failure


if __name__=='__main__':print(json.dumps(run(),sort_keys=True))
