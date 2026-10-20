# SPDX-License-Identifier: Apache-2.0
import importlib.util
import os
import json
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch,MagicMock

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'scripts'))
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

    @unittest.skipUnless(os.name=='nt','Windows priority ABI')
    def test_wrong_parent_priority_stops_before_any_gui_launch(self):
        import windows_native_resources
        selected=MagicMock();selected.GetPriorityClass.return_value=32
        with patch.object(windows_native_resources,'api',return_value=selected), \
             patch.object(launch.subprocess,'Popen') as create,patch('builtins.print') as output:
            self.assertEqual(launch.launch(['fixture.exe'],normal_priority=True),78)
        create.assert_not_called()
        row=json.loads(output.call_args.args[0]);self.assertFalse(row['valid'])
        self.assertEqual(row['reason'],'expected-below-normal-wrapper-unavailable')

    @unittest.skipUnless(os.name=='nt','Windows priority ABI')
    def test_control_verifies_actual_gui_and_preserves_original_failure(self):
        import windows_native_resources
        for actual,code,expected in [(32,23,23),(64,0,78),(64,23,23),(0,0,78)]:
            with self.subTest(actual=actual):
                selected=MagicMock();selected.GetPriorityClass.side_effect=[16384,actual]
                child=MagicMock(pid=42,_handle=99);child.__enter__.return_value=child;child.wait.return_value=code
                with patch.object(windows_native_resources,'api',return_value=selected), \
                     patch.object(launch.subprocess,'Popen',return_value=child) as create,patch('builtins.print') as output:
                    self.assertEqual(launch.launch(['fixture.exe'],normal_priority=True),expected)
                self.assertEqual(create.call_args.kwargs['creationflags'],launch.subprocess.CREATE_NO_WINDOW|launch.subprocess.NORMAL_PRIORITY_CLASS)
                self.assertEqual(create.call_args.kwargs['startupinfo'].wShowWindow,0)
                row=json.loads(output.call_args.args[0]);self.assertEqual(row['valid'],actual==32)
                self.assertEqual(row.get('gui_priority_class'),actual or None);self.assertEqual(row['gui_pid'],42)
                self.assertNotIn('fixture.exe',output.call_args.args[0])

    @unittest.skipUnless(os.name=='nt','Windows priority ABI')
    def test_unavailable_control_never_emits_private_exception_text(self):
        import windows_native_resources
        selected=MagicMock();selected.GetPriorityClass.return_value=16384
        with patch.object(windows_native_resources,'api',return_value=selected), \
             patch.object(launch.subprocess,'Popen',side_effect=OSError('private launch detail')),patch('builtins.print') as output:
            self.assertEqual(launch.launch(['private-fixture.exe'],normal_priority=True),78)
        row=json.loads(output.call_args.args[0]);self.assertFalse(row['valid'])
        self.assertEqual(row['reason'],'gui-create-unavailable');self.assertNotIn('private',output.call_args.args[0])

    @unittest.skipUnless(os.name=='nt','actual hidden Windows priority inheritance')
    def test_actual_below_normal_parent_and_explicit_normal_child(self):
        from process_tree import ProcessTree
        child_code="import ctypes,json;k=ctypes.WinDLL('kernel32',use_last_error=True);k.GetCurrentProcess.restype=ctypes.c_void_p;k.GetPriorityClass.argtypes=[ctypes.c_void_p];k.GetPriorityClass.restype=ctypes.c_uint32;print(json.dumps({'priority':k.GetPriorityClass(k.GetCurrentProcess())}))"
        parent_code=("import ctypes,runpy,sys;k=ctypes.WinDLL('kernel32',use_last_error=True);"
            "k.GetCurrentProcess.restype=ctypes.c_void_p;k.SetPriorityClass.argtypes=[ctypes.c_void_p,ctypes.c_uint32];"
            "assert k.SetPriorityClass(k.GetCurrentProcess(),16384);"
            f"sys.path.insert(0,{str(ROOT/'scripts')!r});m=runpy.run_path({str(ROOT/'scripts/diagnose-native-launch.py')!r});"
            f"assert m['launch']([sys.executable,'-c',{child_code!r}])==0;"
            f"assert m['launch']([sys.executable,'-c',{child_code!r}],normal_priority=True)==0")
        with ProcessTree([sys.executable,'-c',parent_code],cwd=ROOT,env=dict(os.environ)) as tree:
            try:
                stdout,stderr=tree.process.communicate(timeout=10)
                self.assertEqual(tree.process.returncode,0,stderr.decode())
                self.assertTrue(tree.wait_for_descendants(3))
                rows=[json.loads(line) for line in stdout.splitlines()]
                self.assertEqual([row['priority'] for row in rows],[16384,32])
                receipt=json.loads(stderr);self.assertTrue(receipt['valid'])
                self.assertEqual(receipt['wrapper_priority_class'],16384)
                self.assertEqual(receipt['gui_priority_class'],32)
            finally:
                tree.kill();tree.process.communicate(timeout=5)

if __name__=='__main__':unittest.main()
