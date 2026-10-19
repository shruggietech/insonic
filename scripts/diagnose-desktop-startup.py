#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Incident-bound disposable Windows observations; never delivery qualification."""
import ctypes
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time

sys.path.insert(0,str(Path(__file__).resolve().parent))
import windows_package_pipeline as pipeline
from process_tree import ProcessTree

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


def capture(args,directory,env,timeout):
    started=time.monotonic();timed_out=False
    with ProcessTree([str(arg) for arg in args],cwd=directory,env=env) as tree:
        try:stdout,stderr=tree.process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out=True;tree.kill()
            stdout,stderr=tree.process.communicate(timeout=5)
        members=job_processes(tree)
        code=tree.process.returncode
    return {'feature_qualification':False,'exit_code':code,'timed_out':timed_out,
            'elapsed_seconds':round(time.monotonic()-started,3),
            'stdout':stdout[:LIMIT].decode('utf-8',errors='replace'),'stderr':stderr[:LIMIT].decode('utf-8',errors='replace'),
            'truncated':len(stdout)>LIMIT or len(stderr)>LIMIT,'job_before_close':members}


def save(directory,name,value):
    directory.mkdir(parents=True,exist_ok=True)
    (directory/name).write_text(json.dumps(value,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')


def run():
    pipeline.m.disposable_runner()
    if os.name!='nt':raise RuntimeError('native startup diagnosis requires disposable Windows')
    tool=os.environ.get('GITHUB_SHA','')
    if not re.fullmatch('[a-f0-9]{40}',tool):raise ValueError('exact diagnostic tool revision is required')
    archive,receipt=incident_archive()
    package=pipeline.load('desktop_startup_package','package-desktop.py')
    package.verify_archive_receipt(archive,receipt)
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
        # Observe the original production parent first. Old bytes may discard the
        # controller's stdout; subsequent direct children are labelled separately.
        parent=capture([gui,'--exit-qualification'],root,env,100)
        parent.update(boundary='original-production-parent')
        save(evidence,'original-parent.json',parent);print(json.dumps(parent,sort_keys=True),flush=True);summary['observations'].append({'file':'original-parent.json','exit_code':parent['exit_code']})
        for index,action in enumerate(['menu','close','alt-f4']*8):
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
        if parent['exit_code']:raise RuntimeError('original production parent failed; boundary observations retained')
    return summary


if __name__=='__main__':print(json.dumps(run(),sort_keys=True))
