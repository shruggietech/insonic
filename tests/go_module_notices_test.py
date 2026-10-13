# SPDX-License-Identifier: Apache-2.0
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

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
        return {'Path':name,'Version':version,'Sum':'h1:'+'a'*43+'=','Dir':str(directory)}

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

if __name__=='__main__': unittest.main()
