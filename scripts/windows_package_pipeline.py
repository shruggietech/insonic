#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Qualify one immutable Windows delivery across isolated, bounded CI jobs."""
from copy import deepcopy
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import sys

import windows_msi as m
import windows_setup
from importlib.util import spec_from_file_location, module_from_spec

ROOT=m.ROOT
SCOPES=('archive','interactive','maintenance','setup')
INTERACTIVE_KEYS={'user_context','interactive_transactions','documentation_completion','native_ui_capture'}
GATES={'interactive':INTERACTIVE_KEYS,'maintenance':set(m.QUALIFICATION_KEYS)-INTERACTIVE_KEYS,'setup':set(windows_setup.QUALIFICATION_KEYS)}

def load(name,filename):
    spec=spec_from_file_location(name,ROOT/'scripts'/filename);module=module_from_spec(spec);spec.loader.exec_module(module);return module

def write(path,value):
    path=Path(path);path.parent.mkdir(parents=True,exist_ok=True)
    path.write_text(json.dumps(value,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')

def read(path):
    path=Path(path)
    if path.is_symlink() or not path.is_file() or path.stat().st_size>32*1024*1024:raise ValueError('qualification JSON is missing, redirected or oversized')
    return json.loads(path.read_text(encoding='utf-8'))

def bindings(base,initial):
    if (base.get('source_dirty') is not False or initial.get('source_dirty') is not False
            or base.get('revision')!=initial.get('revision') or not re.fullmatch('[0-9a-f]{40}',base.get('revision',''))
            or base.get('payload_receipt')!=initial or base.get('payload_archive')!=initial.get('archive')
            or base.get('payload_archive_sha256')!=initial.get('archive_sha256')
            or base.get('publication_eligible') is not False or base.get('payload_qualification')!='pending'):
        raise ValueError('unqualified Windows assembly authority differs')
    version=initial.get('schema_version','');prefix='insonic_'+version+'_windows_amd64'
    if (not re.fullmatch(r'\d+\.\d+\.\d+',version) or initial.get('platform')!='windows_amd64' or initial.get('variant')!='desktop'
            or initial.get('dirname')!=prefix+'_desktop' or initial.get('archive')!=prefix+'_desktop.zip'
            or base.get('artifact')!=prefix+'_installer.msi' or base.get('setup',{}).get('artifact')!=prefix+'_setup.exe'):
        raise ValueError('Windows input filenames or platform differ')
    sources={
        'archive':{'name':initial['archive'],'sha256':initial['archive_sha256'],'size_bytes':initial['archive_size_bytes']},
        'msi':{'name':base['artifact'],'sha256':base['artifact_sha256'],'size_bytes':base['artifact_size_bytes']},
        'setup':{'name':base['setup']['artifact'],'sha256':base['setup']['artifact_sha256'],'size_bytes':base['setup']['artifact_size_bytes']},
    }
    for item in sources.values():
        if not re.fullmatch('[0-9a-f]{64}',item['sha256']) or type(item['size_bytes']) is not int or not 0<item['size_bytes']<=8*1024**3:
            raise ValueError('Windows input digest or length is invalid')
    return {'revision':base['revision'],'sources':sources}

def input_root():return ROOT/'build/windows-desktop-input'

def read_inputs():
    root=input_root()
    if root.is_symlink() or root.resolve().parent!=(ROOT/'build').resolve():raise ValueError('Windows assembly input root is redirected')
    base=read(root/'installer-input.json');initial=read(root/'archive-input.json');identity=bindings(base,initial)
    if identity['revision']!=os.environ.get('SOURCE_REVISION'):raise ValueError('Windows assembly is not the selected source revision')
    for item in identity['sources'].values():
        source=root/item['name']
        if source.is_symlink() or not source.is_file() or source.stat().st_size!=item['size_bytes'] or m.sha(source)!=item['sha256']:
            raise ValueError('transferred Windows asset bytes differ: '+item['name'])
    return base,initial,identity

def restore_inputs():
    base,initial,identity=read_inputs();packages=ROOT/'build/packages';packages.mkdir(parents=True,exist_ok=True)
    if packages.is_symlink() or packages.resolve().parent!=(ROOT/'build').resolve():raise ValueError('Windows package staging root is redirected')
    for item in identity['sources'].values():shutil.copyfile(input_root()/item['name'],packages/item['name'])
    write(ROOT/'build/native/package-desktop-receipt.json',initial)
    write(ROOT/'build/native/windows-installer-receipt.json',base)
    return base,initial,identity

def assemble():
    m.disposable_runner();package=load('windows_pipeline_assembly','package-desktop.py');msi=load('windows_pipeline_msi','package-msi.py')
    archive,dirname,initial=package.build('desktop')
    if initial.get('source_dirty') is not False:raise ValueError('Windows assembly requires clean source')
    snapshot=ROOT/'build/native/windows-package-input-receipt.json';write(snapshot,initial)
    _,base=msi.build(package.BUILD/dirname,snapshot,provisional=True)
    identity=bindings(base,initial);target=input_root();target.mkdir(parents=True,exist_ok=True)
    for item in identity['sources'].values():shutil.copyfile(package.BUILD/item['name'],target/item['name'])
    write(target/'archive-input.json',initial);write(target/'installer-input.json',base)
    read_inputs()
    return identity

def evidence_files(directory):
    result=[]
    for path in sorted(directory.rglob('*')):
        if path.is_symlink():raise ValueError('qualification evidence is redirected')
        if path.is_file() and path.name!='scope-proof.json':
            result.append({'path':path.relative_to(directory).as_posix(),'sha256':m.sha(path),'size_bytes':path.stat().st_size})
    return result

def validate_files(directory,files):
    if not isinstance(files,list) or not files or len(files)>4096:raise ValueError('qualification evidence inventory is incomplete')
    seen=set()
    for item in files:
        name=item.get('path','');relative=PurePosixPath(name)
        if not name or relative.is_absolute() or '..' in relative.parts or '\\' in name or ':' in name or name in seen:
            raise ValueError('qualification evidence path is invalid')
        seen.add(name);path=directory.joinpath(*relative.parts)
        if path.is_symlink() or not path.resolve().is_relative_to(directory.resolve()) or not path.is_file() or path.stat().st_size!=item.get('size_bytes') or m.sha(path)!=item.get('sha256'):
            raise ValueError('qualification evidence bytes differ: '+name)
    actual={path.relative_to(directory).as_posix() for path in directory.rglob('*') if path.is_file() and path.name!='scope-proof.json'}
    if actual!=seen:raise ValueError('qualification evidence contains unrecorded files')

def qualify(scope):
    if scope not in SCOPES:raise ValueError('unknown Windows qualification scope')
    m.disposable_runner();base,initial,identity=restore_inputs()
    package=load('windows_pipeline_package','package-desktop.py');msi=load('windows_pipeline_restore','package-msi.py')
    msi.restore_payload(ROOT/'build/native/package-desktop-receipt.json')
    if scope=='archive':
        package.smoke(ROOT/'build/packages'/initial['archive'],initial['dirname'],deepcopy(initial))
        completed=read(ROOT/'build/native/package-desktop-receipt.json')
        if any(completed.get(key)!=value for key,value in initial.items()):raise ValueError('archive qualification rewrote immutable assembly authority')
        load('windows_pipeline_release','release.py').verify_qualification(completed)
        checks={'archive':'passed'}
    else:
        checks=load('windows_pipeline_native','qualify-windows-msi.py').qualify(ROOT/'build/packages'/base['artifact'],base,lane=scope)
        if scope=='interactive':
            load('windows_pipeline_capture','capture-installer-ui.py').capture(ROOT/'build/packages'/base['artifact'],base)
            checks['native_ui_capture']='passed'
        if scope=='setup':checks=dict(base['setup']['qualification'])
        if set(checks)!=GATES[scope] or any(value!='passed' for value in checks.values()):raise ValueError('native qualification scope is incomplete')
    native=ROOT/'build/native'
    proof={'kind':'windows-qualification-proof','scope':scope,'identity':identity,'checks':checks,'files':evidence_files(native)}
    write(native/'scope-proof.json',proof)
    return proof

def combine(base,initial,archive,proofs):
    identity=bindings(base,initial)
    if set(proofs)!=set(SCOPES):raise ValueError('Windows qualification scope is missing')
    if any(archive.get(key)!=value for key,value in initial.items()):raise ValueError('qualified archive changed immutable assembly authority')
    final=deepcopy(base)
    for scope in SCOPES:
        proof=proofs[scope];expected={'archive'} if scope=='archive' else GATES[scope]
        if (proof.get('kind')!='windows-qualification-proof' or proof.get('scope')!=scope or proof.get('identity')!=identity
                or set(proof.get('checks',{}))!=expected or any(value!='passed' for value in proof['checks'].values())):
            raise ValueError('Windows qualification proof is incomplete or belongs to other bytes: '+scope)
        if scope=='setup':final['setup']['qualification'].update(proof['checks'])
        elif scope!='archive':final['qualification'].update(proof['checks'])
    if set(final['qualification'])!=set(m.QUALIFICATION_KEYS) or any(v!='passed' for v in final['qualification'].values()):raise ValueError('complete raw MSI qualification is required')
    final.update(payload_receipt=archive,publication_eligible=True,payload_qualification='passed')
    return final

def finalize():
    m.disposable_runner();base,initial,identity=restore_inputs();native=ROOT/'build/native'
    proofs={};directories={};parent=ROOT/'build/windows-desktop-proofs'
    for scope in SCOPES:
        matches=[path for path in parent.iterdir() if path.is_dir() and path.name==f'windows-desktop-proof-{scope}-{identity["revision"]}-{os.environ["GITHUB_RUN_ATTEMPT"]}']
        if len(matches)!=1:raise ValueError('Windows qualification artifact is absent or ambiguous: '+scope)
        directory=matches[0];proof=read(directory/'scope-proof.json');validate_files(directory,proof.get('files'))
        proofs[scope]=proof;directories[scope]=directory
    archive=read(directories['archive']/'package-desktop-receipt.json')
    package=load('windows_pipeline_final_package','package-desktop.py');release=load('windows_pipeline_final_release','release.py')
    release.verify_qualification(archive);package.verify_archive_receipt(ROOT/'build/packages'/initial['archive'],archive)
    final=combine(base,initial,archive,proofs)
    captures=directories['interactive']/'installer-capture'
    if len(list(captures.glob('installer-*.png')))!=14 or not (captures/'installer-transactions-capture.json').is_file() or not (captures/'installer-capture.json').is_file():raise ValueError('completed native transaction captures are incomplete')
    shutil.copytree(captures,native/'installer-capture',dirs_exist_ok=True)
    for scope in SCOPES:
        shutil.copytree(directories[scope],native/'qualification'/scope,dirs_exist_ok=True)
    m.verify_installer(ROOT/'build/packages'/final['artifact'],final)
    write(native/'package-desktop-receipt.json',archive)
    write(native/'msi-lifecycle.json',{'revision':identity['revision'],'identity':identity,'scopes':proofs,'checks':final['qualification'],'setup':final['setup']['qualification']})
    write(native/'windows-installer-receipt.json',final)
    return final

if __name__=='__main__':
    if len(sys.argv)!=2 or sys.argv[1] not in (*SCOPES,'assemble','finalize'):raise SystemExit('Usage: windows_package_pipeline.py assemble|archive|interactive|maintenance|setup|finalize')
    stage=sys.argv[1]
    result=assemble() if stage=='assemble' else finalize() if stage=='finalize' else qualify(stage)
    print(json.dumps({'stage':stage,'state':'passed','publication_eligible':result.get('publication_eligible',False)}))
