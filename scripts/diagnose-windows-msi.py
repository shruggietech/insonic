#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Disposable-CI observations using prior qualified bytes, never release evidence."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import uuid
import windows_msi as m

def load(name,filename):
    spec=importlib.util.spec_from_file_location(name,m.ROOT/'scripts'/filename);module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module

def git_identity(root):
    values=[]
    for args in [('rev-parse','HEAD'),('status','--porcelain','--untracked-files=normal')]:
        completed=subprocess.run(['git',*args],cwd=root,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=15)
        if completed.returncode:raise RuntimeError('diagnostic source identity lookup failed')
        values.append(completed.stdout.decode('utf-8').strip())
    return {'revision':values[0],'dirty':bool(values[1])}

def run(archive,receipt_path,payload_source_root,ordinary=False,ordinary_first=False):
    m.disposable_runner();archive=Path(archive);receipt=json.loads(Path(receipt_path).read_text(encoding='utf-8'));source=Path(payload_source_root)
    output=m.ROOT/'build/native/msi-diagnostic';output.mkdir(parents=True,exist_ok=True)
    report={'kind':'windows-msi-diagnostic-observations','diagnostic_only':True,'ineligible_for_publication':True,'qualification_pass':False,'payload_source':git_identity(source),'tool_source':git_identity(m.ROOT),'observations':{}}
    report['tool_source']['files']=[{'path':name,'sha256':m.sha(m.ROOT/name)} for name in ('scripts/diagnose-windows-msi.py','scripts/qualify-windows-msi.py','scripts/windows_msi.py')]
    def save():
        (output/'report.json').write_text(json.dumps(report,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
    save()
    if report['payload_source']['dirty'] or report['payload_source']['revision']!=receipt.get('revision') or receipt.get('source_dirty') is not False:raise ValueError('diagnostic payload requires its exact clean prior source checkout')
    if receipt.get('kind')!='native-package-receipt' or receipt.get('platform')!='windows_amd64' or receipt.get('variant')!='desktop' or receipt.get('dirname')!='insonic_'+receipt.get('schema_version','')+'_windows_amd64_desktop' or archive.name!=receipt.get('archive'):raise ValueError('diagnostic requires exact declared Windows desktop archive')
    release=load('diagnostic_release','release.py');release.verify_qualification(receipt);release.verify_signing(receipt['signing'])
    package=load('diagnostic_package','package-desktop.py');verified=package.verify_archive_receipt(archive,receipt,expected_source_root=source)
    report['payload_source'].update({'archive':archive.name,'archive_sha256':m.sha(archive),'inventory_sha256':receipt['inventory_sha256'],'receipt_sha256':m.sha(Path(receipt_path))});save()
    qualifier=load('diagnostic_qualifier','qualify-windows-msi.py');service_ownership=qualifier.service_diagnostic(configuration=True)
    report['observations']['initial_installer_service']=service_ownership;save()
    with tempfile.TemporaryDirectory(prefix='insonic-ci-diagnostic-') as temporary:
        temporary=Path(temporary);package.extract_package(archive,temporary/'payload');payload=temporary/'payload'/receipt['dirname'];inventory=verified['inventory']
        package.verify_inventory(payload,inventory)
        if m.sha(payload/'package-inventory.json')!=receipt['inventory_sha256']:raise ValueError('diagnostic extracted inventory differs')
        model=m.installer_model(inventory,receipt['inventory_sha256'],(m.ROOT/'LICENSE').read_text(encoding='utf-8'));authored=output/'diagnostic-authored.msi';m.author(authored,payload,model)
        fixture=temporary/'observed.msi';upgrade=m.guid('diagnostic-only/'+str(uuid.uuid4()));qualifier.clone(authored,fixture,model['version'],upgrade)
        parts=list(map(int,model['version'].split('.')));parts[2]+=1;new_version=m.msi_version('.'.join(map(str,parts)))
        upgraded=temporary/'upgraded.msi';qualifier.clone(authored,upgraded,new_version,upgrade)
        report['tool_source']['authored_msi_sha256']=m.sha(authored);report['observations']['fixture_identity']={'upgrade_code':upgrade,'product_code':m.product_code(model['version'],upgrade),'fixture_msi_sha256':m.sha(fixture),'publication_eligible':False};save()
        install=temporary/'installed program é with spaces';product=None;prepared=False
        if ordinary_first:
            report['observations']['elevated_actor_before_transactions']={'token':qualifier.token_info(),'mutex':qualifier.installer_mutex_diagnostic()}
            ordinary_fixture=temporary/'ordinary-first.msi';ordinary_upgrade=m.guid('diagnostic-ordinary-first/'+str(uuid.uuid4()));qualifier.clone(authored,ordinary_fixture,model['version'],ordinary_upgrade)
            report['observations']['ordinary_first_fixture_identity']={'upgrade_code':ordinary_upgrade,'product_code':m.product_code(model['version'],ordinary_upgrade),'fixture_msi_sha256':m.sha(ordinary_fixture),'publication_eligible':False,'before_any_elevated_install_or_capture':True}
            qualifier.prepare_launcher_fixture(payload,temporary/'launcher-context-fixture');prepared=True;save()
            try:report['observations']['ordinary_first_account']={'outcome':'observed-success','evidence':qualifier.ordinary_context(ordinary_fixture,temporary/'ordinary first installation',output/'ordinary-first-context.json',model,service_ownership)}
            except Exception as error:report['observations']['ordinary_first_account']={'outcome':'observed-failure','error':str(error)}
            save()
        try:
            qualifier.run_installer(['/i',fixture,'INSTALLDIR='+str(install),'LICENSE_ACCEPTED=1','ADDLOCAL=Core,CLI','START_MENU=0','DESKTOP_SHORTCUT=0'],output/'core-cli-install.log');product=m.product_code(model['version'],upgrade)
            qualifier.installed_hashes(install,model,False);report['observations']['core_cli_install']={'outcome':'observed-success','payload_hashes_verified':True}
            report['observations']['registered_states']=qualifier.registered_feature_states(product,output/'registered-states.json');save()
            if ordinary and not prepared:qualifier.prepare_launcher_fixture(install,temporary/'launcher-context-fixture');prepared=True
            feature_added=False
            try:
                qualifier.run_installer(['/i',fixture,'INSTALLDIR='+str(install),'LICENSE_ACCEPTED=1','ADDLOCAL=Core,Desktop,CLI','REINSTALL=Core','REINSTALLMODE=vomus'],output/'feature-add.log')
                qualifier.installed_hashes(install,model,True);report['observations']['feature_add']={'outcome':'observed-success','payload_hashes_verified':True};feature_added=True
            except Exception as error:report['observations']['feature_add']={'outcome':'observed-failure','error':str(error)}
            save()
            if feature_added:
                try:
                    qualifier.run_installer(['/i',upgraded,'INSTALLDIR='+str(install),'LICENSE_ACCEPTED=1'],output/'major-upgrade.log')
                    # Cleanup follows the successful engine transaction even when a
                    # subsequent exact payload assertion fails.
                    product=m.product_code(new_version,upgrade)
                    qualifier.installed_hashes(install,model,True)
                    report['observations']['major_upgrade']={'outcome':'observed-success','payload_hashes_verified':True,'product_code':product,'fixture_msi_sha256':m.sha(upgraded)}
                except Exception as error:report['observations']['major_upgrade']={'outcome':'observed-failure','error':str(error),'cleanup_product_code':product}
                save()
        except Exception as error:report['observations']['install_sequence']={'outcome':'observed-failure','error':str(error)};save()
        finally:
            if product is not None:
                try:qualifier.run_installer(['/x',product],output/'cleanup.log');report['observations']['cleanup']={'outcome':'observed-success'}
                except Exception as error:report['observations']['cleanup']={'outcome':'observed-failure','error':str(error)}
            save()
        if ordinary and prepared:
            try:report['observations']['ordinary_account']={'outcome':'observed-success','evidence':qualifier.ordinary_context(fixture,temporary/'ordinary installation',output/'ordinary-context.json',model,service_ownership)}
            except Exception as error:report['observations']['ordinary_account']={'outcome':'observed-failure','error':str(error)}
            save()
    return report

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--archive',type=Path,required=True);parser.add_argument('--receipt',type=Path,required=True);parser.add_argument('--payload-source-root',type=Path,required=True);parser.add_argument('--ordinary-context',action='store_true');parser.add_argument('--ordinary-first',action='store_true');args=parser.parse_args()
    m.disposable_runner()
    try:report=run(args.archive,args.receipt,args.payload_source_root,args.ordinary_context,args.ordinary_first);print(json.dumps(report,sort_keys=True))
    except Exception as error:
        output=m.ROOT/'build/native/msi-diagnostic';output.mkdir(parents=True,exist_ok=True);(output/'diagnostic-error.json').write_text(json.dumps({'diagnostic_only':True,'ineligible_for_publication':True,'qualification_pass':False,'error':str(error)},indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n');raise
