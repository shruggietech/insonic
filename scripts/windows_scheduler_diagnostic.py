# SPDX-License-Identifier: Apache-2.0
"""Disposable-only scheduler observation; raw ETL never enters evidence."""
import json
import os
import re
from pathlib import Path
import shutil
import sys
import tempfile
from threading import Event, Thread
import time
import uuid

from process_tree import ProcessTree
import windows_native_resources as resources


LIMIT=65536
FIXTURE="""import json,os,threading,time,sys
print(json.dumps({'pid':os.getpid(),'tid':threading.get_native_id()}),flush=True)
if sys.stdin.buffer.read(1)!=b'G':sys.exit(78)
time.sleep(.2)
end=time.perf_counter()+.15
while time.perf_counter()<end:pass
"""


class Unavailable(RuntimeError):
    pass


def observation_input(start,end,owned,gui=0,wrapper=0):
    if not owned.get('available') or owned.get('overflow'):
        raise Unavailable('owned-identities-unavailable')
    identities=[]
    for row in owned.get('processes',[]):
        identity=row.get('identity',{})
        if row.get('owned') is not True or identity.get('available') is not True:
            raise Unavailable('owned-identity-unavailable')
        identities.append({'pid':row['pid'],**{key:identity[key] for key in [
            'created_unix_micros','query_begin_unix_micros','query_end_unix_micros']}})
    if not 1<=len(identities)<=16 or len({row['pid'] for row in identities})!=len(identities):
        raise Unavailable('owned-identity-bound')
    return {'begin_unix_micros':start['unix_micros'],'end_unix_micros':end['unix_micros'],
            'owned':identities,'gui_pid':gui,'wrapper_pid':wrapper,'start_anchor':start,'end_anchor':end}


def preflight_valid(result,pid,tid):
    if result.get('available') is not True:return False
    matches=[row for row in result.get('processes',[]) if row.get('pid')==pid and row.get('matched_lifetime') is True]
    if len(matches)!=1:return False
    threads=[row for row in matches[0].get('threads',[]) if row.get('tid')==tid]
    if len(threads)!=1:return False
    summary=threads[0].get('summary',{})
    return (summary.get('waiting',{}).get('total_ns',0)>=100_000_000 and
            summary.get('scheduled',{}).get('total_ns',0)>1_000_000)


class Scheduler:
    def __init__(self,root,capture,disposable):
        disposable()
        if os.name!='nt':raise Unavailable('scheduler-platform-unavailable')
        self.root=Path(root).resolve();self.capture=capture;self.active=False;self.cleanup_confirmed=True
        self.instance='insonic-s040-'+uuid.uuid4().hex
        self.temporary=tempfile.TemporaryDirectory(prefix='s040-private-scheduler-',dir=self.root/'build')
        self.private=Path(self.temporary.name).resolve()
        if not self.private.is_relative_to((self.root/'build').resolve()):raise Unavailable('private-directory-invalid')
        self.env=dict(os.environ,DOTNET_CLI_TELEMETRY_OPTOUT='1',DOTNET_SKIP_FIRST_TIME_EXPERIENCE='1',
                      DOTNET_NOLOGO='1',MSBUILDDISABLENODEREUSE='1',
                      NUGET_PACKAGES=str(self.root/'build/s040-scheduler-packages'))
        self.dotnet=shutil.which('dotnet',path=self.env.get('PATH'))
        self.wpr=Path(self.env.get('WINDIR',''))/'System32/wpr.exe'
        self.project=self.root/'scripts/diagnostics/scheduler'
        self.dll=self.private/'decoder/SchedulerDecoder.dll'
        self.preflight=None
        self.stage='not-started';self.last_tool=None;self.decoder_hresult=None

    def __enter__(self):return self

    def __exit__(self,*args):
        try:
            if self.active:self.cancel()
        finally:
            # Resolve again before recursive cleanup. This is never evidence.
            try:
                if self.private.resolve().parent!=(self.root/'build').resolve():raise OSError('private path changed')
                self.temporary.cleanup()
            except OSError:self.cleanup_confirmed=False
            print(json.dumps({'kind':'scheduler-cleanup','confirmed':self.cleanup_confirmed,
                              'raw_directory_removed':not self.private.exists(),'feature_qualification':False}),flush=True)

    def tool(self,args,timeout):
        result=self.capture(args,self.root,self.env,timeout,require_release=True)
        self.last_tool={key:result[key] for key in ['exit_code','timed_out','truncated','capture_incomplete','owned_release']}
        self.last_tool['native_error_codes']=[int(code,16) for code in re.findall(r'\b0x[0-9a-fA-F]{8}\b',result['stdout']+'\n'+result['stderr'])[:4]]
        if (result['exit_code'] or result['timed_out'] or result['truncated'] or
                result['capture_incomplete'] or not result['owned_release']):
            raise Unavailable('scheduler-tool-unavailable')
        return result

    def prepare(self):
        self.stage='build'
        if not self.dotnet or not self.wpr.is_file():raise Unavailable('scheduler-tools-missing')
        self.tool([self.dotnet,'build',self.project/'SchedulerDecoder.csproj','--no-self-contained',
            '--verbosity','quiet','--configfile',self.project/'NuGet.Config','-p:RestoreLockedMode=true',
            '--disable-build-servers','--artifacts-path',self.private/'artifacts','--output',self.dll.parent],90)
        self.stage='self-test';self.tool([self.dotnet,self.dll,'--self-test'],10)
        self.stage='preflight-start'
        self.start()
        try:
            self.stage='preflight-child'
            origin=time.monotonic();start=resources.clock_anchor(origin);header=[];ready=Event()
            deadline=time.monotonic()+5
            with ProcessTree([sys.executable,'-c',FIXTURE],cwd=self.root,env=self.env,input_pipe=True) as tree:
                def read_header():
                    try:header.append(tree.process.stdout.readline(1024))
                    finally:ready.set()
                reader=Thread(target=read_header,daemon=True);reader.start()
                try:
                    if not ready.wait(max(0,deadline-time.monotonic())):raise Unavailable('preflight-header-timeout')
                    reader.join(0)
                    value=json.loads(header[0])
                    if (not isinstance(value,dict) or set(value)!={'pid','tid'} or type(value['pid']) is not int or
                            type(value['tid']) is not int or value['pid']!=tree.process.pid or value['tid']<=0):
                        raise Unavailable('preflight-identity-invalid')
                    owned=resources.owned_snapshot(tree.job,[tree.process.pid],identity=True)
                    # The native handle/creation query occurs while this exact
                    # owned child is blocked for the parent's one-byte ACK.
                    tree.process.communicate(input=b'G',timeout=max(0,deadline-time.monotonic()))
                    if tree.process.returncode or not tree.wait_for_descendants(max(0,deadline-time.monotonic())):
                        raise Unavailable('preflight-child-failed')
                    end=resources.clock_anchor(origin)
                except BaseException:
                    tree.kill()
                    raise
            self.stage='preflight-stop';trace=self.stop()
            self.stage='preflight-decode'
            result=self.decode(trace,observation_input(start,end,owned))
            self.stage='preflight-validation'
            if not preflight_valid(result,value['pid'],value['tid']):raise Unavailable('preflight-scheduler-validation-failed')
            self.preflight={'available':True,'known_wait_and_scheduled_activity':True,
                            'owned_lifetime_matched':True,'console_only':True,'gui_launches':0}
        finally:
            if self.active:self.cancel()
            # Discard the preflight raw trace before the separate actual session.
            for selected in self.private.glob('*.etl'):selected.unlink()

    def start(self):
        if self.active:raise Unavailable('scheduler-session-already-active')
        # Mark potentially active before launching: a command timeout does not
        # prove that the ETW session never started. Cancel only our unique name.
        self.active=True
        self.tool([self.wpr,'-start',str(self.project/'scheduler.wprp')+'!Scheduler.Light',
                   '-filemode','-recordtempto',self.private,'-instancename',self.instance],15)

    def cancel(self):
        if not self.active:return
        try:self.tool([self.wpr,'-cancel','-instancename',self.instance],15)
        except Exception:self.cleanup_confirmed=False
        finally:self.active=False

    def stop(self):
        trace=self.private/'scheduler.etl'
        self.tool([self.wpr,'-stop',trace,'-skipPdbGen','-instancename',self.instance],30)
        self.active=False
        if not trace.is_file() or not 0<trace.stat().st_size<256*1024*1024:
            raise Unavailable('scheduler-trace-size-unavailable')
        return trace

    def decode(self,trace,inputs):
        source=self.private/'input.json';output=self.private/'decoded.json'
        encoded=json.dumps(inputs).encode('utf-8')
        if len(encoded)>LIMIT:raise Unavailable('scheduler-input-bound')
        source.write_bytes(encoded)
        if output.exists():output.unlink()
        try:self.tool([self.dotnet,self.dll,trace,source,output],30)
        except Unavailable:
            if output.is_file() and output.stat().st_size<=LIMIT:
                try:
                    row=json.loads(output.read_text(encoding='utf-8'))
                    if row.get('kind')=='owned-scheduler-observation' and type(row.get('hresult')) is int:
                        self.decoder_hresult=row['hresult']
                except (ValueError,OSError):pass
            raise
        if not output.is_file() or output.stat().st_size>LIMIT:raise Unavailable('scheduler-output-bound')
        result=json.loads(output.read_text(encoding='utf-8'))
        if result.get('kind')!='owned-scheduler-observation' or result.get('available') is not True:
            raise Unavailable('scheduler-decoder-unavailable')
        return result

    def finish(self,observation,control):
        try:
            self.stage='actual-stop'
            trace=self.stop()
            self.stage='actual-input'
            timing=observation['timing'];live=observation['resources']['live']
            if live.get('available') is not True or live.get('overlapped_cleanup') is True or not control or control.get('available') is not True:
                raise Unavailable('scheduler-live-control-unavailable')
            inputs=observation_input(timing['start_clock_anchor'],timing['end_clock_anchor'],live['owned'],
                gui=control['gui_pid'],wrapper=control['wrapper_pid'])
            self.stage='actual-decode';result=self.decode(trace,inputs)
            result={**result,'preflight':self.preflight,'raw_trace_retained':False}
        except Exception:
            result={'kind':'owned-scheduler-observation',**self.failure(),'preflight':self.preflight}
        finally:
            if self.active:self.cancel()
            try:
                for selected in self.private.glob('*.etl'):selected.unlink()
            except OSError:self.cleanup_confirmed=False
        result['cleanup_confirmed']=self.cleanup_confirmed
        if not self.cleanup_confirmed:result.update(available=False,reason='scheduler-cleanup-unconfirmed')
        return result

    def failure(self):
        return {'available':False,'feature_qualification':False,'reason':'scheduler-observation-unavailable',
                'stage':self.stage,'tool':self.last_tool,'decoder_hresult':self.decoder_hresult,
                'raw_trace_retained':False,'cleanup_confirmed':self.cleanup_confirmed}
