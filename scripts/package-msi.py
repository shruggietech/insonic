#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Build selectable MSI from the exact already-qualified desktop archive."""
import argparse
import importlib.util
import json
from pathlib import Path
import sys
sys.path.insert(0,str(Path(__file__).resolve().parent))
import windows_msi as m
ROOT=m.ROOT

def load(name,file):
    spec=importlib.util.spec_from_file_location(name,ROOT/'scripts'/file);mod=importlib.util.module_from_spec(spec);spec.loader.exec_module(mod);return mod

def build(payload,receipt_path):
    payload=Path(payload).resolve();archive_receipt=json.loads(Path(receipt_path).read_text(encoding='utf-8'))
    package=load('package_desktop_msi','package-desktop.py');archive=ROOT/'build/packages'/archive_receipt['archive']
    verified=package.verify_archive_receipt(archive,archive_receipt)
    inventory=json.loads((payload/'package-inventory.json').read_text(encoding='utf-8'));package.verify_inventory(payload,inventory)
    if verified['inventory']!=inventory:raise ValueError('MSI payload differs from qualified archive inventory')
    model=m.installer_model(inventory,m.sha(payload/'package-inventory.json'),(ROOT/'LICENSE').read_text(encoding='utf-8'))
    version=inventory['schema_version'];output=ROOT/'build/packages'/('insonic_'+version+'_windows_amd64_installer.msi')
    m.author(output,payload,model)
    receipt={k:archive_receipt[k] for k in ('schema_version','product_version','documentation_version','revision','source_dirty','signing','distribution')}
    receipt.update({'kind':'windows-installer-receipt','platform':'windows_amd64','variant':'selectable','format':'msi','artifact':output.name,'artifact_sha256':m.sha(output),'artifact_size_bytes':output.stat().st_size,'payload_archive':archive_receipt['archive'],'payload_archive_sha256':archive_receipt['archive_sha256'],'payload_receipt':archive_receipt,'payload_inventory':inventory,'msi_product_version':model['version'],'product_code':model['product_code'],'upgrade_code':model['upgrade_code'],'scope':'per-user','license_sha256':m.sha(ROOT/'LICENSE'),'icon_sha256':m.sha(ROOT/m.ICON),'qualification':{k:'pending' for k in m.QUALIFICATION_KEYS},'authoring':{'kind':'native-windows-installer-database','cabinet':'windows-makecab-mszip','tables_independently_verified':True,'cabinet_files_independently_verified':True}})
    write(receipt);return output,receipt

def write(receipt):
    path=ROOT/'build/native/windows-installer-receipt.json';path.parent.mkdir(parents=True,exist_ok=True);path.write_text(json.dumps(receipt,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('stage',choices=['build','qualify','capture','all']);p.add_argument('--payload',type=Path);p.add_argument('--receipt',type=Path,default=ROOT/'build/native/package-desktop-receipt.json');args=p.parse_args()
    if args.stage in ('all','qualify','capture'):m.disposable_runner()
    if args.stage in ('build','all'):
        if args.payload is None:p.error('--payload is required for building')
        output,receipt=build(args.payload,args.receipt)
    else:
        receipt=json.loads((ROOT/'build/native/windows-installer-receipt.json').read_text(encoding='utf-8'));output=ROOT/'build/packages'/receipt['artifact']
    if args.stage in ('qualify','all'):
        qualify=load('qualify_windows_msi','qualify-windows-msi.py');receipt['qualification'].update(qualify.qualify(output,receipt));write(receipt)
    if args.stage in ('capture','all'):
        capture=load('capture_installer_ui','capture-installer-ui.py');capture.capture(output,receipt);receipt['qualification']['native_ui_capture']='passed';write(receipt)
    if args.stage=='all':m.verify_installer(output,receipt)
    print(json.dumps({'artifact':receipt['artifact'],'sha256':receipt['artifact_sha256'],'qualification':receipt['qualification']}))
