# SPDX-License-Identifier: Apache-2.0
import copy
import importlib.util
import io
import json
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / 'scripts'))
spec = importlib.util.spec_from_file_location('go_dependency_transfer', ROOT / 'scripts/go-dependency-transfer.py')
transfer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transfer)


class DependencyPreparationTests(unittest.TestCase):
    def test_discovery_only_builds_dependency_libraries(self):
        packages = [
            {'ImportPath': 'example/product/cmd/app', 'Name': 'main', 'Module': {'Path': 'example/product', 'Main': True}, 'Imports': ['fmt', 'example/lib/api']},
            {'ImportPath': 'example/product/internal/core', 'Name': 'core', 'Module': {'Main': True}, 'Imports': ['example/lib/internal/thing']},
            {'ImportPath': 'fmt', 'Name': 'fmt', 'Standard': True},
            {'ImportPath': 'example/lib/api', 'Name': 'api', 'Module': {'Path': 'example/lib', 'Main': False}},
            {'ImportPath': 'example/lib/internal/thing', 'Name': 'thing', 'Module': {'Path': 'example/lib', 'Main': False}},
        ]
        targets = transfer.dependency_targets(packages)
        self.assertEqual(targets, ['example/lib/api', 'example/lib/internal/thing', 'fmt'])
        for bad in [dict(packages[3], Name='main'), dict(packages[3], Module={}), dict(packages[3], Error={'Err': 'broken dependency'})]:
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                transfer.dependency_targets([*packages[:3], bad, packages[4]])
        with self.assertRaises(ValueError):
            transfer.dependency_targets([dict(packages[0], Error={'Err': 'unknown product error'}), *packages[1:]])
        # Discovery may see not-yet-built product embed assets. It must not create them.
        transfer.dependency_targets([dict(packages[0], Error={'Err': 'pattern assets: no matching files found'}), *packages[1:]])
        with self.assertRaises(ValueError):
            transfer.dependency_targets([*packages[:3], dict(packages[3], Imports=['example/product/internal/core']), packages[4]])
        # Main=false is normally omitted by Go's JSON encoder.
        transfer.dependency_targets([*packages[:3], dict(packages[3], Module={'Path': 'example/lib'}), packages[4]])

    def test_prepare_invocations_do_not_build_product_or_tests(self):
        graph = [{'ImportPath': 'example/product/cmd/app', 'Name': 'main', 'Module': {'Main': True}, 'Imports': ['fmt']},
                 {'ImportPath': 'fmt', 'Name': 'fmt', 'Standard': True}]
        calls = []
        def run(args, env):
            calls.append(args)
            return ''.join(json.dumps(p) for p in graph) if args[:2] == ['go', 'list'] else ''
        with patch.object(transfer, 'run', run):
            result = transfer.compile_context({'tags': 'desktop,production,system_ladybug', 'package': './cmd/app'}, {})
        self.assertEqual(result, ['fmt'])
        self.assertEqual(calls[-1], ['go', 'build', '-tags', 'desktop,production,system_ladybug', 'fmt'])
        self.assertFalse(any('-export' in c or 'test' in c for c in calls))

    def test_transfer_rejects_identity_digest_and_archive_aliases_before_activation(self):
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary)
            cache, artifact, output = base / 'cache', base / 'artifact', base / 'output'
            for part in ['go/00', 'mod/cache/download']:
                (cache / part).mkdir(parents=True)
                (cache / part / 'data').write_bytes(b'dependency')
            identity = {'revision': 'a' * 40, 'run': '42', 'attempt': '1', 'compiler': 'compiler', 'native': 'pins', 'contexts': ['exact']}
            transfer.pack(cache, artifact, identity)
            for field in identity:
                bad = dict(identity, **{field: 'altered'})
                with self.subTest(field=field), self.assertRaises(ValueError): transfer.restore(artifact, output, bad)
            self.assertFalse(output.exists())
            receipt = json.loads((artifact / 'transfer.json').read_text())
            for name, kind in [('../escape', 'file'), ('go/FILE', 'symlink'), ('go/NUL.txt', 'file'), ('go/file:ads', 'file')]:
                with self.subTest(name=name):
                    with tarfile.open(artifact / 'dependencies.tar', 'w') as archive:
                        entry = tarfile.TarInfo(name)
                        entry.size = 1
                        if kind == 'symlink': entry.type, entry.linkname = tarfile.SYMTYPE, 'target'
                        archive.addfile(entry, io.BytesIO(b'x') if kind == 'file' else None)
                    altered = copy.deepcopy(receipt)
                    altered['archive'] = transfer.digest(artifact / 'dependencies.tar')
                    (artifact / 'transfer.json').write_text(json.dumps(altered))
                    with self.assertRaises(ValueError): transfer.restore(artifact, output, identity)
                    self.assertFalse(output.exists())
            transfer.pack(cache, artifact, identity)
            with (artifact / 'dependencies.tar').open('ab') as stream: stream.write(b'altered')
            with self.assertRaises(ValueError): transfer.restore(artifact, output, identity)
            transfer.pack(cache, artifact, identity)
            transfer.restore(artifact, output, identity)
            self.assertEqual((output / 'go/00/data').read_bytes(), b'dependency')
            with self.assertRaises(ValueError): transfer.restore(artifact, output, identity)


if __name__ == '__main__': unittest.main()
