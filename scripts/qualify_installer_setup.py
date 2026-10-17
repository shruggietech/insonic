# SPDX-License-Identifier: Apache-2.0
"""Disposable ordinary-user setup lifecycle and recovery qualification."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import windows_msi as m
import windows_setup

def qualifier():
    spec=importlib.util.spec_from_file_location('owned_setup_qualifier',Path(__file__).with_name('qualify-windows-msi.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module

def validate_scope(stage,token,qualification):
    m.disposable_runner();stage=Path(stage);q=qualifier()
    marker=json.loads((stage/'owned-ci-account.json').read_text(encoding='utf-8'))
    if stage.resolve().parent!=Path(os.environ['ProgramData']).resolve() or not stage.name.startswith('insonic-ci-account-') or marker['sid_sha256']!=token['user_sid_sha256'] or token['elevated'] or token['administrator_enabled'] or token['integrity_rid']!=8192:raise RuntimeError('setup qualification is outside the owned ordinary account')
    if qualification:
        scope=json.loads((stage/'setup-qualification.json').read_text(encoding='utf-8'))
        if scope.get('feature_qualification') is not True or m.sha(stage/'fixture.msi')!=scope['msi_sha256'] or m.sha(stage/'exact-setup.exe')!=scope['setup_sha256']:raise RuntimeError('setup qualification exact artifact authority differs')
    elif q.diagnostic_mode(stage) not in ('rollback-setup','rollback-setup-full'):raise RuntimeError('focused setup probe is not elected')

def exercise(stage,token,quiet_install,*,full,qualification=False):
    validate_scope(stage,token,qualification);q=qualifier();stage=Path(stage)
    model=json.loads((stage/'ui-model.json').read_text(encoding='utf-8'))
    if model['family']!=m.UPGRADE_CODE:raise RuntimeError('production coordinator requires the exact insonic family in its otherwise empty owned profile')
    import ctypes as C
    import ctypes.wintypes as W
    dll=C.WinDLL('msi');dll.MsiQueryProductStateW.argtypes=[W.LPCWSTR];dll.MsiQueryProductStateW.restype=C.c_int
    if any(dll.MsiQueryProductStateW(model[key])!=-1 for key in ('old_product','new_product')):raise RuntimeError('setup probe refuses pre-existing registration')
    root=Path(os.environ['LOCALAPPDATA'])/'insonic-setup';directory=stage/'UI program é & spaces'
    result={'kind':'windows-setup-recovery','feature_qualification':qualification,'token':token,'transactions':[],'cleanup':[]}
    def run(name,args,expected):
        command=[str(stage/('ui-'+name+'-setup.exe')),*args]
        done=subprocess.run(command,cwd=stage,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=75)
        row={'name':name,'actual_code':done.returncode,'stdout':done.stdout.decode(errors='replace')[-4000:],'stderr':done.stderr.decode(errors='replace')[-4000:]};result['transactions'].append(row)
        for source in root.glob('*'):
            if source.is_file() and source.suffix in ('.json','.log') and source.stat().st_size<=8*1024*1024:shutil.copyfile(source,stage/'native'/('context.setup-'+name+'-'+source.name))
        if done.returncode!=expected:raise RuntimeError('production setup returned unexpected result: '+json.dumps(row))
        return row
    try:
        old_source=stage/'ui-old.msi';old_sha=m.sha(old_source)
        legacy=stage/'legacy-full-source.msi';shutil.copyfile(old_source,legacy)
        if full:
            from qualify_installer_ui import NativeUI
            ui=NativeUI(stage/'native/context.setup-ui.json')
            row=ui.run(old_source,directory,'setup-install',selection='cli',native_process=True,setup_process=True)
            result['transactions'].append(row)
        else:run('old',['--quiet','--accept-license','--install-dir',str(directory),'--clients','cli','--start-menu','0','--desktop-shortcut','0'],0)
        if dll.MsiQueryProductStateW(model['old_product'])!=5:raise RuntimeError('setup did not install old registration')
        expected={item['path']:m.sha(directory/item['path']) for item in model['payload'] if item['feature']!='Desktop'}
        def unchanged():
            if dll.MsiQueryProductStateW(model['old_product'])!=5 or dll.MsiQueryProductStateW(model['new_product'])!=-1 or any(m.sha(directory/name)!=sha for name,sha in expected.items()):raise RuntimeError('source preflight changed the working installation')
        cache_source=root/model['old_product']/'source.msi'
        if not full:
            # An actual legacy raw-MSI install has no source producer. Recreate
            # that state, then prove missing-source refusal before old removal.
            quiet_install(['/x',model['old_product']],stage/'native/context.setup-legacy-reset.log')
            if (root/model['old_product']).exists():raise RuntimeError('committed raw removal retained old source cache')
            quiet_install(['/i',old_source,'INSTALLDIR='+str(directory),'LICENSE_ACCEPTED=1','ADDLOCAL=Core,CLI','REMOVE=Desktop','START_MENU=0','DESKTOP_SHORTCUT=0','LAUNCH=0'],stage/'native/context.setup-legacy-install.log')
            run('bad',['--quiet','--accept-license'],1603);unchanged()
            if (root/'pending.json').exists():raise RuntimeError('missing source began a destructive upgrade')
            wrong=stage/'ui-wrong-source.msi'
            run('bad',['--quiet','--accept-license','--previous-msi',str(wrong),'--previous-sha256',m.sha(wrong)],1603);unchanged()
            refused=json.loads((root/'last-result.json').read_text(encoding='utf-8'))
            if refused.get('native_code') is not None or refused.get('recovery')!='not-started' or (root/model['old_product']).exists():raise RuntimeError('mismatched inventory was retained or reached MSI execution')
            result['same_version_wrong_inventory_refused']='passed'
            run('bad',['--quiet','--accept-license','--previous-msi',str(legacy),'--previous-sha256',old_sha],1603);unchanged()
            result['legacy_source_refusal_and_reseed']='passed'
            backup=stage/'owned-cached-source.backup';shutil.copyfile(cache_source,backup)
            with cache_source.open('r+b') as stream:stream.seek(128);value=stream.read(1);stream.seek(128);stream.write(bytes([value[0]^1]))
            run('bad',['--quiet','--accept-license'],1603);unchanged()
            if (root/'pending.json').exists():raise RuntimeError('corrupt source began a destructive upgrade')
            run('bad',['--quiet','--accept-license','--previous-msi',str(legacy),'--previous-sha256',old_sha],1603);unchanged()
            repair=json.loads((root/'last-result.json').read_text(encoding='utf-8'))
            if m.sha(cache_source)!=old_sha or repair.get('recovery')!='completed' or repair.get('native_code')!=1603:raise RuntimeError('explicit previous source did not repair the corrupt owned cache')
            backup.unlink();result['corrupt_source_refusal']='passed';result['corrupt_source_reseed']='passed'
        old_source.unlink();(stage/'ui-old-setup.exe').unlink();legacy.unlink()
        result['downloaded_old_sources_absent']=True
        if full:
            for name,cancel,code in [('cancel','execution',1602),('bad',None,1603)]:
                row=ui.run(stage/('ui-'+name+'.msi'),directory,'setup-'+name,operation='upgrade',selection='cli',cancel=cancel,expected=code,native_process=True,setup_process=True)
                result['transactions'].append(row);unchanged()
                if cancel and row.get('old_removed_before_cancel') is not True:raise RuntimeError('actual Cancel did not occur after old removal')
                receipt=json.loads((root/'last-result.json').read_text(encoding='utf-8'))
                if receipt['native_code']!=code or receipt['setup_code']!=code or receipt['recovery']!='completed':raise RuntimeError('setup did not complete recovery before native return')
        else:run('bad',['--quiet','--accept-license'],1603)
        unchanged();q.assert_unchecked_shortcuts()
        receipt=json.loads((root/'last-result.json').read_text(encoding='utf-8'))
        if receipt['native_code']!=1603 or receipt['setup_code']!=1603 or receipt['recovery']!='completed':raise RuntimeError('production coordinator result does not prove synchronous recovery')
        run('new',['--quiet','--accept-license'],0)
        if dll.MsiQueryProductStateW(model['old_product'])!=-1 or dll.MsiQueryProductStateW(model['new_product'])!=5:raise RuntimeError('successful coordinated upgrade registration differs')
        q.assert_unchecked_shortcuts()
        if (root/model['old_product']).exists() or not (root/model['new_product']/'source.msi').is_file():raise RuntimeError('successful upgrade did not replace its retained source')
        run('new',['--quiet','--remove'],0)
        if dll.MsiQueryProductStateW(model['new_product'])!=-1 or (root/model['new_product']).exists():raise RuntimeError('coordinated removal retained registration or full source')
        result['successful_upgrade_and_removal_source_cleanup']='passed'
        result['state']='passed';return result
    finally:
        for source in root.glob('*'):
            if source.is_file() and source.suffix in ('.json','.log') and source.stat().st_size<=8*1024*1024:
                try:shutil.copyfile(source,stage/'native'/('context.setup-final-'+source.name))
                except OSError as error:result.setdefault('retention_errors',[]).append({'file':source.name,'error':str(error)})
        for key in ('old_product','new_product'):
            product=model[key]
            if dll.MsiQueryProductStateW(product) in (1,5):
                try:quiet_install(['/x',product],stage/'native'/('context.setup-cleanup-'+key+'.log'));result['cleanup'].append({'product':product,'completed':True})
                except BaseException as error:result['cleanup'].append({'product':product,'completed':False,'error':str(error)[-2000:]})
        (stage/'native/context.setup-probe.json').write_text(json.dumps(result,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')


def qualify(stage,token,quiet_install):
    """Exact shipping entry point plus instrumented transactional fault cases."""
    m.disposable_runner();stage=Path(stage);validate_scope(stage,token,True)
    from qualify_installer_ui import NativeUI,prepare
    q=qualifier();ui=NativeUI(stage/'native/context.setup-exact.json')
    source=stage/'fixture.msi';directory=stage/'exact setup program é & spaces'
    props=dict(m.table_rows(m.CompoundFile(source).streams,'Property'));product=props['ProductCode']
    if ui.dll.MsiQueryProductStateW(product)!=-1:raise RuntimeError('exact setup refuses an existing ordinary-user product')
    transactions=[]
    try:
        transactions.append(ui.run(source,directory,'setup-exact-install',selection='cli',native_process=True,setup_process=True))
        if ui.dll.MsiQueryProductStateW(product)!=5:raise RuntimeError('exact setup did not install its actual product')
        q.installed_hashes(directory,json.loads((stage/'context-model.json').read_text(encoding='utf-8')),False)
        transactions.append(ui.run(source,directory,'setup-exact-remove',operation='remove',selection='cli',native_process=True,setup_process=True))
        if ui.dll.MsiQueryProductStateW(product)!=-1:raise RuntimeError('exact setup did not remove its actual product')
        if (Path(os.environ['LOCALAPPDATA'])/'insonic-setup'/product).exists():raise RuntimeError('exact setup removal retained the full MSI source')
        quiet=exercise(stage,token,quiet_install,full=False,qualification=True)
        # Each complete journey removed its products and source cache. Reauthor
        # the small native fault capsules from the exact same setup program.
        prepare(stage,stage/'launcher-context-fixture',q.clone,diagnostic_only=True,setup_probe=True)
        full=exercise(stage,token,quiet_install,full=True,qualification=True)
        if quiet.get('state')!='passed' or full.get('state')!='passed':raise RuntimeError('setup recovery journeys are incomplete')
        result={'state':'passed','token':token,'feature_qualification':True,'exact_entry_point':transactions,
                'quiet':quiet,'full':full,'qualification':{key:'passed' for key in windows_setup.QUALIFICATION_KEYS}}
        (stage/'native/context.setup-qualification.json').write_text(json.dumps(result,indent=2,sort_keys=True)+'\n',encoding='utf-8',newline='\n')
        return result
    finally:
        if ui.dll.MsiQueryProductStateW(product) in (1,5):quiet_install(['/x',product],stage/'native/context.setup-exact-cleanup.log')
