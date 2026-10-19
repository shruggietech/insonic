# SPDX-License-Identifier: Apache-2.0
import importlib.util
import os
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('diagnostic_native_launch',ROOT/'scripts/diagnose-native-launch.py')
launch=importlib.util.module_from_spec(spec);spec.loader.exec_module(launch)


class NativeLaunchComparisonTests(unittest.TestCase):
    @unittest.skipUnless(os.name=='nt','actual hidden Windows console child')
    def test_explicit_startup_hide_preserves_hidden_noninteractive_creation_and_exit(self):
        with patch.object(launch.subprocess,'run',wraps=subprocess.run) as child:
            code=launch.launch([sys.executable,'-c','import sys;sys.exit(17)'])
        self.assertEqual(code,17)
        opts=child.call_args.kwargs
        self.assertEqual(opts['creationflags'],0x08000000)
        self.assertEqual(opts['stdin'],subprocess.DEVNULL)
        self.assertTrue(opts['startupinfo'].dwFlags&subprocess.STARTF_USESHOWWINDOW)
        self.assertEqual(opts['startupinfo'].wShowWindow,0)
        self.assertNotIn('stdout',opts);self.assertNotIn('stderr',opts)

    def test_owner_host_is_refused_before_launch(self):
        with patch.dict(os.environ,{},clear=True),patch.object(launch,'launch') as child:
            with self.assertRaises((RuntimeError,ValueError)):
                launch.main(['relative.exe','--exit-qualification-child','fixture.json'])
        child.assert_not_called()



    @unittest.skipUnless(os.name=='nt','actual inherited Windows job')
    def test_wrapper_child_remains_in_the_existing_owned_job(self):
        from process_tree import ProcessTree
        code="import importlib.util,sys; s=importlib.util.spec_from_file_location('probe',sys.argv[1]);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);sys.exit(m.launch([sys.executable,'-c','import sys;sys.exit(17)']))"
        with ProcessTree([sys.executable,'-c',code,str(ROOT/'scripts/diagnose-native-launch.py')],cwd=ROOT,env=dict(os.environ)) as tree:
            output,error=tree.process.communicate(timeout=5)
            self.assertEqual(tree.process.returncode,17,error)
            self.assertTrue(tree.wait_for_descendants(5))

if __name__=='__main__':unittest.main()
