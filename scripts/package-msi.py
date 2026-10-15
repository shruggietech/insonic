#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Build selectable MSI from the exact already-qualified desktop archive."""
import argparse
import importlib.util
import json
import shutil
from pathlib import Path
import sys
sys.path.insert(0,str(Path(__file__).resolve().parent))
import windows_msi as m
ROOT=m.ROOT

def load(name,file):
    spec=importlib.util.spec_from_file_location(name,ROOT/'scripts'/file);mod=importlib.util.module_from_spec(spec);spec.loader.exec_module(mod);return mod

def restore_payload(receipt_path):
    """Consume the exact qualified same-run archive; never rebuild or weaken it."""
    receipt=json.loads(Path(receipt_path).read_text(encoding='utf-8'))
    package=load('package_desktop_restore','package-desktop.py')
    name='insonic_'+package.VERSION+'_windows_amd64_desktop'
    if receipt.get('dirname')!=name or receipt.get('archive')!=name+'.zip' or receipt.get('platform')!='windows_amd64' or receipt.get('variant')!='desktop':raise ValueError('restore requires exact current Windows desktop archive identity')
    base=ROOT/'build/packages';base.mkdir(parents=True,exist_ok=True)
    if base.is_symlink() or not base.resolve().is_relative_to((ROOT/'build').resolve()) or not base.resolve().is_relative_to(ROOT.resolve()):raise ValueError('package restore root is redirected')
    archive=base/receipt['archive'];package.verify_archive_receipt(archive,receipt)
    target=base/name
    if target.is_symlink() or target.resolve().parent!=base.resolve():raise ValueError('package restore target escapes owned build directory')
    if target.exists():shutil.rmtree(target)
    package.extract_package(archive,base)
    inventory=json.loads((target/'package-inventory.json').read_text(encoding='utf-8'));package.verify_inventory(target,inventory)
    if package.sha(target/'package-inventory.json')!=receipt['inventory_sha256']:raise ValueError('restored inventory differs from qualified source')
    return target

def build(payload,receipt_path,provisional=False):
    if provisional:m.disposable_runner()
    payload=Path(payload).resolve();archive_receipt=json.loads(Path(receipt_path).read_text(encoding='utf-8'))
    package=load('package_desktop_msi','package-desktop.py');archive=ROOT/'build/packages'/archive_receipt['archive']
    verified=package.verify_archive_receipt(archive,archive_receipt)
    inventory=json.loads((payload/'package-inventory.json').read_text(encoding='utf-8'));package.verify_inventory(payload,inventory)
    if verified['inventory']!=inventory:raise ValueError('MSI payload differs from qualified archive inventory')
    model=m.installer_model(inventory,m.sha(payload/'package-inventory.json'),(ROOT/'LICENSE').read_text(encoding='utf-8'))
    version=inventory['schema_version'];output=ROOT/'build/packages'/('insonic_'+version+'_windows_amd64_installer.msi')
    m.author(output,payload,model)
    receipt={k:archive_receipt[k] for k in ('schema_version','product_version','documentation_version','revision','source_dirty','distribution')}
    receipt.update({'signing':dict(m.UNSIGNED_SIGNING),'kind':'windows-installer-receipt','platform':'windows_amd64','variant':'selectable','format':'msi','artifact':output.name,'artifact_sha256':m.sha(output),'artifact_size_bytes':output.stat().st_size,'payload_archive':archive_receipt['archive'],'payload_archive_sha256':archive_receipt['archive_sha256'],'payload_receipt':archive_receipt,'payload_inventory':inventory,'msi_product_version':model['version'],'product_code':model['product_code'],'upgrade_code':model['upgrade_code'],'scope':'per-user','license_sha256':m.sha(ROOT/'LICENSE'),'icon_sha256':m.sha(ROOT/m.ICON),'qualification':{k:'pending' for k in m.QUALIFICATION_KEYS},'authoring':{'kind':'native-windows-installer-database','cabinet':'windows-makecab-mszip','tables_independently_verified':True,'cabinet_files_independently_verified':True}})
    if provisional:receipt.update({'publication_eligible':False,'payload_qualification':'pending'})
    write(receipt);return output,receipt

def write(receipt):
    path=ROOT/'build/native/windows-installer-receipt.json';path.parent.mkdir(parents=True,exist_ok=True);path.write_text(json.dumps(receipt,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')

def run_gates(output,receipt,stage):
    m.disposable_runner();failures={}
    if stage=='provisional':receipt.update({'publication_eligible':False,'payload_qualification':'pending'});write(receipt)
    if stage in ('capture','all','provisional'):
        try:
            capture=load('capture_installer_ui','capture-installer-ui.py').capture
            try:capture(output,receipt)
            except RuntimeError as error:
                # A cold native session can miss its intermediate startup bound.
                # The worker has exited; retry the complete unchanged capture
                # exactly once in fresh owned workers. Never retry assertions.
                if str(error).strip()!='native authored capture failed: native session initialization exceeded its bound':raise
                receipt.setdefault('qualification_retries',{})['native_ui_capture']={'attempts':2,'first_failure':'restricted-session-initialization-timeout'}
                write(receipt);capture(output,receipt)
            receipt['qualification']['native_ui_capture']='passed'
        except Exception as error:
            receipt['qualification']['native_ui_capture']='failed';failures['native_ui_capture']=str(error)[-4000:]
        write(receipt)
    if stage in ('qualify','all','provisional'):
        try:receipt['qualification'].update(load('qualify_windows_msi','qualify-windows-msi.py').qualify(output,receipt))
        except Exception as error:failures['native_lifecycle']=str(error)[-4000:]
        write(receipt)
    if failures:
        receipt['qualification_errors']=failures;write(receipt)
        raise RuntimeError('native installer gates failed: '+json.dumps(failures,sort_keys=True))
    if stage=='all':m.verify_installer(output,receipt)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('stage',choices=['build','qualify','capture','all','provisional']);p.add_argument('--payload',type=Path);p.add_argument('--restore-payload',action='store_true');p.add_argument('--receipt',type=Path,default=ROOT/'build/native/package-desktop-receipt.json');args=p.parse_args()
    if args.stage in ('all','qualify','capture','provisional'):m.disposable_runner()
    if args.stage in ('build','all','provisional'):
        if args.restore_payload:args.payload=restore_payload(args.receipt)
        if args.payload is None:p.error('--payload or --restore-payload is required for building')
        output,receipt=build(args.payload,args.receipt,provisional=args.stage=='provisional')
    else:
        receipt=json.loads((ROOT/'build/native/windows-installer-receipt.json').read_text(encoding='utf-8'));output=ROOT/'build/packages'/receipt['artifact']
    if args.stage in ('capture','qualify','all','provisional'):run_gates(output,receipt,args.stage)
    print(json.dumps({'artifact':receipt['artifact'],'sha256':receipt['artifact_sha256'],'qualification':receipt['qualification']}))
