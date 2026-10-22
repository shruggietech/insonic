# SPDX-License-Identifier: Apache-2.0
import importlib.util
import base64
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
import shutil
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('go_module_notices', ROOT / 'scripts/go_module_notices.py')
notices = importlib.util.module_from_spec(spec)
spec.loader.exec_module(notices)


class GoModuleNoticesTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.cache = self.root / 'cache'; self.cache.mkdir()
        self.goroot = self.root / 'compiler'; self.goroot.mkdir()
        (self.goroot / 'LICENSE').write_text('compiler license', encoding='utf-8')
        self.target = self.root / 'notices'; self.target.mkdir()
        self.a = self.module('example.com/Upper/module')
        self.b = self.module('example.com/other/module')
        self.binary = [(self.root / 'insonic', 'github.com/shruggietech/insonic/cmd/insonic')]
        self.infos = [self.info(self.binary[0][1], [self.a])]
        self.selected = [self.a]
        self.calls = []

    def module(self, name, version='v1.2.3'):
        directory = self.cache / (notices.escaped(name)+'@'+notices.escaped(version))
        directory.mkdir(parents=True, exist_ok=True)
        (directory / 'LICENSE').write_text('upstream license '+name, encoding='utf-8')
        content=(directory/'LICENSE').read_bytes()
        summary=hashlib.sha256(content).hexdigest()+'  '+name+'@'+version+'/LICENSE\n'
        checksum='h1:'+base64.b64encode(hashlib.sha256(summary.encode('utf-8')).digest()).decode('ascii')
        return {'Path':name,'Version':version,'Sum':checksum,'Dir':str(directory)}

    def info(self, command, deps):
        return {'GoVersion':'go1.27.1','Path':command,'Deps':deps}

    def fake_run(self, args):
        self.calls.append(args)
        if args[:4] == ['go','version','-m','-json']:
            return ''.join(json.dumps(value)+'\n' for value in self.infos).encode()
        if args[:5] == ['go','list','-mod=readonly','-m','-json']:
            return ''.join(json.dumps(value)+'\n' for value in self.selected).encode()
        if args[:3] == ['go','env','-json']:
            return json.dumps({'GOMODCACHE':str(self.cache),'GOROOT':str(self.goroot),'GOVERSION':'go1.27.1'}).encode()
        self.fail('unexpected tool/graph download: '+str(args))

    def collect(self):
        return notices.collect_notices(self.target,self.binary,self.fake_run,'1.27.1')

    def test_cli_membership_selects_only_linked_modules_and_no_tool_graph(self):
        rows = self.collect()
        self.assertEqual([(row['module'],row['version']) for row in rows],[(self.a['Path'],self.a['Version'])])
        lookup = next(call for call in self.calls if call[1]=='list')
        self.assertEqual(lookup[5:],[self.a['Path']])
        self.assertEqual((self.target/rows[0]['notices'][0]).read_bytes(),(Path(self.a['Dir'])/'LICENSE').read_bytes())
        self.assertEqual((self.target/'Go-LICENSE').read_text(),'compiler license')
        self.assertFalse((Path(self.a['Dir'])/'go.mod').exists())

    def test_desktop_union_deduplicates_shared_dependencies(self):
        self.binary.append((self.root/'insonic-desktop','github.com/shruggietech/insonic/cmd/insonic-desktop'))
        self.infos.append(self.info(self.binary[1][1],[self.a,self.b])); self.selected.append(self.b)
        self.assertEqual(len(self.collect()),2)
        self.assertEqual(sum(call[1]=='version' for call in self.calls),1)

    def test_shared_conflicting_binary_identity_is_rejected(self):
        other = dict(self.a,Version='v2.0.0')
        self.infos.append(self.info(self.binary[0][1],[other])); self.binary.append(self.binary[0])
        with self.assertRaisesRegex(ValueError,'conflict'): self.collect()

    def test_versioned_replacement_uses_actual_source_and_refuses_local(self):
        logical = dict(self.a,Replace=self.b)
        self.infos[0]['Deps']=[logical]; self.selected=[logical]
        self.assertEqual(self.collect()[0]['module'],self.b['Path'])
        logical['Replace']={'Path':'../local'}
        with self.assertRaisesRegex(ValueError,'identity|version'): self.collect()

    def test_replacement_and_checksum_mismatch_fail(self):
        for changed in [dict(self.a,Sum='h1:'+'b'*43+'='),dict(self.a,Version='v1.2.4'),dict(self.a,Replace=self.b)]:
            with self.subTest(changed=changed):
                self.selected=[changed]
                with self.assertRaisesRegex(ValueError,'identity|match'): self.collect()

    def test_invalid_source_directories_never_borrow_checkout_notices(self):
        for directory in [None,'','relative',str(self.root),str(self.cache/'absent')]:
            with self.subTest(directory=directory):
                item=dict(self.a)
                if directory is None: item.pop('Dir')
                else: item['Dir']=directory
                self.selected=[item]
                with self.assertRaisesRegex(ValueError,'directory'): self.collect()

    def test_missing_empty_or_notice_only_license_is_rejected(self):
        license=Path(self.a['Dir'])/'LICENSE'
        for mode in ['missing','empty','notice-only']:
            with self.subTest(mode=mode):
                if license.exists(): license.unlink()
                if mode=='empty': license.write_bytes(b'')
                if mode=='notice-only': (license.parent/'NOTICE').write_text('no license',encoding='utf-8')
                with self.assertRaisesRegex(ValueError,'license|notice'): self.collect()

    def test_collision_free_module_paths_preserve_case_and_slashes(self):
        self.a=self.module('example.com/a_b/c'); self.b=self.module('example.com/a/b_c')
        self.infos[0]['Deps']=[self.a,self.b]; self.selected=[self.a,self.b]
        paths=[row['notices'][0] for row in self.collect()]
        self.assertEqual(len(set(paths)),2)
        self.assertIn('!upper',notices.escaped('Upper'))

    def test_wrong_compiler_command_or_missing_build_metadata_is_rejected(self):
        for changed in [dict(self.infos[0],GoVersion='go1.27.0'),dict(self.infos[0],Path='example.com/other'),dict(self.infos[0],Deps=[])]:
            with self.subTest(changed=changed):
                original=self.infos; self.infos=[changed]
                with self.assertRaises(ValueError): self.collect()
                self.infos=original

    def test_independent_inventory_membership_rejects_self_consistent_tooling_or_omission(self):
        rows=self.collect()
        notices.verify_notices(self.target,rows,self.binary,self.fake_run,'1.27.1')
        for bad in [[],rows+[{'module':self.b['Path'],'version':self.b['Version'],'notices':rows[0]['notices']}],rows+rows]:
            with self.subTest(bad=bad):
                with self.assertRaisesRegex(ValueError,'membership|duplicate'): notices.verify_notices(self.target,bad,self.binary,self.fake_run,'1.27.1')

    def test_inventory_notice_reference_cannot_borrow_project_license_or_escape(self):
        rows=self.collect()
        for reference in ['../LICENSE','Go-LICENSE','go/other@v1.2.3/LICENSE']:
            with self.subTest(reference=reference):
                bad=[dict(rows[0],notices=[reference])]
                with self.assertRaisesRegex(ValueError,'notice'): notices.verify_notices(self.target,bad,self.binary,self.fake_run,'1.27.1')
        (self.target/rows[0]['notices'][0]).write_bytes(b'')
        with self.assertRaisesRegex(ValueError,'notice'): notices.verify_notices(self.target,rows,self.binary,self.fake_run,'1.27.1')

    def test_notice_copy_uses_verified_snapshot_after_source_changes(self):
        from unittest.mock import patch
        license=Path(self.a['Dir'])/'LICENSE'
        original=license.read_bytes()
        verify=notices.verified_source_notices
        def mutate_after_verification(*args):
            snapshot=verify(*args)
            license.write_bytes(b'changed after verified snapshot')
            return snapshot
        with patch.object(notices,'verified_source_notices',side_effect=mutate_after_verification):
            rows=self.collect()
        self.assertEqual((self.target/rows[0]['notices'][0]).read_bytes(),original)
        self.assertNotEqual(license.read_bytes(),original)

    def test_real_go_cache_tampered_license_is_rejected_despite_unchanged_metadata(self):
        import os
        import sys
        import zipfile
        sys.path.insert(0,str(ROOT/'scripts'))
        from process_tree import ProcessTree
        go_version=json.loads((ROOT/'internal/qualification/dependencies.json').read_text(encoding='utf-8'))['go']
        with ProcessTree(['go','env','GOROOT'],cwd=ROOT,env=os.environ.copy()) as tree:
            compiler_root,error=tree.process.communicate(timeout=60)
            if tree.process.returncode: raise RuntimeError(error.decode('utf-8'))
        go=Path(compiler_root.decode('utf-8').strip())/'bin'/('go.exe' if os.name=='nt' else 'go')
        application=self.root/'application'; application.mkdir()
        proxy=self.root/'proxy'; version=proxy/'example.test/library/@v'; version.mkdir(parents=True)
        (application/'go.mod').write_text('module example.test/application\n\ngo '+go_version+'\n\nrequire example.test/library v1.0.0\n',encoding='utf-8',newline='\n')
        (application/'main.go').write_text('package main\nimport "example.test/library"\nfunc main() { _ = library.Value }\n',encoding='utf-8',newline='\n')
        module_files={'go.mod':b'module example.test/library\n\ngo 1.27.1\n',
                      'library.go':b'package library\nconst Value = "controlled fixture"\n',
                      'LICENSE':b'controlled upstream license\n'}
        (version/'v1.0.0.mod').write_bytes(module_files['go.mod'])
        (version/'v1.0.0.info').write_text('{"Version":"v1.0.0","Time":"2026-10-13T00:00:00Z"}',encoding='utf-8')
        with zipfile.ZipFile(version/'v1.0.0.zip','w') as archive:
            for name,data in module_files.items(): archive.writestr('example.test/library@v1.0.0/'+name,data)
        environment={**os.environ,'GOPROXY':proxy.as_uri(),'GOSUMDB':'off','GONOPROXY':'none','GOPRIVATE':'',
                     'GOWORK':'off','GOTOOLCHAIN':'local','GOMODCACHE':str(self.cache),'GOFLAGS':'-modcacherw'}
        def run(args):
            with ProcessTree([str(go),*args[1:]],cwd=application,env=environment) as tree:
                output,error=tree.process.communicate(timeout=60)
                if tree.process.returncode: raise RuntimeError(error.decode('utf-8'))
                return output
        run(['go','mod','download','example.test/library'])
        binary=application/'fixture.exe'
        run(['go','build','-o',str(binary),'.'])
        selected=[(binary,'example.test/application')]
        rows=notices.collect_notices(self.target,selected,run,go_version)
        self.assertEqual(rows[0]['module'],'example.test/library')
        before=run(['go','list','-mod=readonly','-m','-json','example.test/library'])
        cached_license=self.cache/'example.test/library@v1.0.0/LICENSE'
        cached_license.chmod(0o600)
        cached_license.write_bytes(b'tampered cached license')
        self.assertEqual(before,run(['go','list','-mod=readonly','-m','-json','example.test/library']))
        with self.assertRaisesRegex(ValueError,'checksum|source bytes'):
            notices.collect_notices(self.target,selected,run,'1.27.1')

    def prepare_repository_patch(self, module=notices.PATCH_MODULE):
        policy = notices.PATCH_POLICIES[module]
        source = self.root / policy['directory']
        shutil.copytree(ROOT / policy['directory'], source)
        manifest = self.root / (policy['directory'] + '.source.json')
        shutil.copyfile(ROOT / (policy['directory'] + '.source.json'), manifest)
        shutil.copyfile(ROOT / 'go.sum', self.root / 'go.sum')
        logical = {'Path': module, 'Version': policy['version'],
                   'Replace': {'Path': './' + policy['directory'], 'Dir': str(source)}}
        self.infos[0]['Deps'] = [dict(logical, Replace=dict(logical['Replace'], Version='(devel)'))]
        self.selected = [logical]
        return source, logical

    def test_webview_repository_patch_has_verified_bytes_and_explicit_notices(self):
        source, logical = self.prepare_repository_patch()
        with patch.object(notices, 'ROOT', self.root):
            rows = self.collect()
            notices.verify_notices(self.target, rows, self.binary, self.fake_run, '1.27.1')
            names = {Path(name).name for name in rows[0]['notices']}
            self.assertTrue({'LICENSE', 'LICENSE.insonic', 'NOTICE.insonic'} <= names)
            self.assertIn(b'repository-local replacement', (self.target / next(name for name in rows[0]['notices'] if name.endswith('/NOTICE.insonic'))).read_bytes())
            bad = [dict(rows[0], notices=[name for name in rows[0]['notices'] if not name.endswith('/NOTICE.insonic')])]
            with self.assertRaisesRegex(ValueError, 'patch notices'):
                notices.verify_notices(self.target, bad, self.binary, self.fake_run, '1.27.1')
            (source / 'pkg/edge/window_lifetime_windows.go').write_bytes(b'tampered maintained source')
            with self.assertRaisesRegex(ValueError, 'source bytes|checksum'):
                self.collect()

    def test_repository_patch_cannot_redirect_source_or_spoof_another_module(self):
        source, logical = self.prepare_repository_patch()
        with patch.object(notices, 'ROOT', self.root):
            self.selected = [dict(logical, Replace=dict(logical['Replace'], Dir=str(self.root)))]
            with self.assertRaisesRegex(ValueError, 'directory'):
                self.collect()
        for value in [dict(logical, Path='example.test/unapproved'), dict(logical, Version='v99.99.99'),
                      dict(logical, Replace={'Path': '../unapproved'}),
                      dict(logical, Replace=dict(logical['Replace'], Version='v1.0.22'))]:
            with self.subTest(value=value), self.assertRaises(ValueError):
                notices.logical_identity(value)

    def test_wails_patch_has_verified_bytes_notices_and_exactly_two_approved_modules(self):
        self.assertEqual(set(notices.PATCH_POLICIES), {'github.com/wailsapp/go-webview2', 'github.com/wailsapp/wails/v2'})
        source, logical = self.prepare_repository_patch('github.com/wailsapp/wails/v2')
        with patch.object(notices, 'ROOT', self.root):
            rows = self.collect()
            notices.verify_notices(self.target, rows, self.binary, self.fake_run, '1.27.1')
            self.assertEqual(rows[0]['module'], logical['Path'])
            names = {Path(name).name for name in rows[0]['notices']}
            self.assertTrue({'LICENSE', 'LICENSE.insonic', 'NOTICE.insonic'} <= names)
            bad = [dict(rows[0], notices=[name for name in rows[0]['notices'] if not name.endswith('/NOTICE.insonic')])]
            with self.assertRaisesRegex(ValueError, 'patch notices'):
                notices.verify_notices(self.target, bad, self.binary, self.fake_run, '1.27.1')
            original = source / 'internal/app/app.go'
            original.chmod(0o600)
            original.write_bytes(original.read_bytes() + b'changed')
            with self.assertRaisesRegex(ValueError, 'source bytes|checksum'):
                self.collect()
        for value in [dict(logical, Path='example.test/unapproved'), dict(logical, Version='v99.99.99'),
                      dict(logical, Replace={'Path': '../unapproved-framework'})]:
            with self.subTest(value=value), self.assertRaises(ValueError):
                notices.logical_identity(value)

    def test_both_repository_upstreams_reject_pruned_or_changed_sums_and_accept_restoration(self):
        for module, policy in notices.PATCH_POLICIES.items():
            self.prepare_repository_patch(module)
            sums = self.root / 'go.sum'
            original = sums.read_text(encoding='utf-8')
            authority = json.loads((self.root / (policy['directory'] + '.source.json')).read_text(encoding='utf-8'))
            with patch.object(notices, 'ROOT', self.root):
                for suffix in ['/go.mod', '']:
                    prefix = module + ' ' + policy['version'] + suffix + ' '
                    matches = [line for line in original.splitlines() if line.startswith(prefix)]
                    self.assertEqual(len(matches), 1)
                    line, bad = matches[0], prefix + 'h1:' + 'A' * 43 + '='
                    cases = ['\n'.join(value for value in original.splitlines() if value != line) + '\n',
                             original.replace(line, bad), original + bad + '\n']
                    for index, changed in enumerate(cases):
                        with self.subTest(module=module, entry=suffix, changed=index):
                            sums.write_text(changed, encoding='utf-8', newline='\n')
                            with self.assertRaisesRegex(ValueError, 'upstream differs from go.sum'):
                                notices.repository_patch_source(module)
                    sums.write_text(original, encoding='utf-8', newline='\n')
                    _, identity = notices.repository_patch_source(module)
                    self.assertEqual(identity, (module, policy['version'], authority['patched_sum']))

if __name__=='__main__': unittest.main()
