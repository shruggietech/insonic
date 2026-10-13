# SPDX-License-Identifier: Apache-2.0
import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('windows_install_package', ROOT / 'scripts/package-desktop.py')
package = importlib.util.module_from_spec(spec); spec.loader.exec_module(package)

@unittest.skipUnless(os.name == 'nt', 'Windows installer execution')
class WindowsInstallOwnershipTests(unittest.TestCase):
    def fixture(self):
        owner = tempfile.TemporaryDirectory(prefix='insonic install ownership ')
        self.addCleanup(owner.cleanup)
        parent = Path(owner.name)
        source = parent / 'source package'; source.mkdir()
        (source / 'a-payload.txt').write_text('qualified fixture', encoding='utf-8')
        package.installers(source, 'windows_amd64', 'cli')
        return parent, source

    def invoke(self, source, target, prelude=''):
        def quoted(path): return "'" + str(path).replace("'", "''") + "'"
        code = "[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false; " + prelude + f"; & {quoted(source / 'install.ps1')} -Destination {quoted(target)}"
        return subprocess.run(['powershell.exe', '-NoProfile', '-NonInteractive', '-Command', code],
                              stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding='utf-8',
                              errors='replace', timeout=60, creationflags=subprocess.CREATE_NO_WINDOW)

    def test_new_spaced_install_copies_complete_tree(self):
        parent, source = self.fixture(); target = parent / 'installed version é'
        result = self.invoke(source, target)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((target / 'a-payload.txt').read_text(), 'qualified fixture')
        self.assertIn('Installed', result.stdout)
        self.assertEqual(list(parent.glob('.insonic-install-*')), [])

    def test_existing_and_source_contained_destinations_preserve_user_data(self):
        parent, source = self.fixture(); existing = parent / 'existing'; existing.mkdir()
        (existing / 'user.txt').write_text('keep')
        for target in (existing, source, source / 'nested install', source / 'elsewhere/../nested'):
            with self.subTest(target=target):
                result = self.invoke(source, target)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn('Installed', result.stdout)
        self.assertEqual((existing / 'user.txt').read_text(), 'keep')
        self.assertFalse((source / 'nested install').exists())
        self.assertFalse((source / 'nested').exists())
        relative = self.invoke(source, Path('source package/relative child'), f"Set-Location -LiteralPath '{parent}'")
        self.assertNotEqual(relative.returncode, 0)
        self.assertFalse((source / 'relative child').exists())

    def test_partial_copy_failure_cleans_only_owned_attempt(self):
        parent, source = self.fixture(); target = parent / 'failed install'
        locked = source / 'z-locked.bin'; locked.write_bytes(b'locked fixture')
        path = str(locked).replace("'", "''")
        result = self.invoke(source, target, f"$locked = [IO.File]::Open('{path}', [IO.FileMode]::Open, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn('Installed', result.stdout)
        self.assertFalse(target.exists())
        self.assertEqual(list(parent.glob('.insonic-install-*')), [])
        self.assertEqual((source / 'a-payload.txt').read_text(), 'qualified fixture')

    def test_late_destination_creation_is_never_removed(self):
        parent, source = self.fixture(); target = parent / 'late existing'
        quoted = str(target).replace("'", "''")
        hook = f"function Copy-Item {{ param($LiteralPath,$Destination,[switch]$Recurse) [IO.Directory]::CreateDirectory('{quoted}') | Out-Null; [IO.File]::WriteAllText('{quoted}\\user.txt','keep'); Microsoft.PowerShell.Management\\Copy-Item -LiteralPath $LiteralPath -Destination $Destination -Recurse }}"
        result = self.invoke(source, target, hook)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual((target / 'user.txt').read_text(), 'keep')
        self.assertEqual(list(parent.glob('.insonic-install-*')), [])

    def test_junction_alias_into_source_is_rejected(self):
        parent, source = self.fixture(); alias = parent / 'source alias'; target = alias / 'nested'
        prelude = f"New-Item -ItemType Junction -Path '{alias}' -Target '{source}' | Out-Null"
        result = self.invoke(source, target, prelude)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((source / 'nested').exists())
        # Remove only the junction itself before TemporaryDirectory traversal.
        subprocess.run(['powershell.exe','-NoProfile','-NonInteractive','-Command',f"[IO.Directory]::Delete('{alias}')"], creationflags=subprocess.CREATE_NO_WINDOW, stdout=subprocess.PIPE,stderr=subprocess.PIPE,check=True)

    def test_subst_alias_into_source_is_rejected(self):
        parent, source = self.fixture()
        drive = next((letter + ':' for letter in 'ZYXWV' if not Path(letter + ':\\').exists()), None)
        if drive is None: self.skipTest('no unused drive for owned SUBST fixture')
        subprocess.run(['subst.exe', drive, str(source)], creationflags=subprocess.CREATE_NO_WINDOW, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)
        try:
            result = self.invoke(source, Path(drive + '\\nested'))
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse((source / 'nested').exists())
        finally:
            subprocess.run(['subst.exe', drive, '/d'], creationflags=subprocess.CREATE_NO_WINDOW, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)

    def test_held_attempt_cannot_be_substituted_before_cleanup(self):
        parent, source = self.fixture(); target = parent / 'selected new'
        hook = "function Copy-Item { param($LiteralPath,$Destination,[switch]$Recurse) try { [IO.Directory]::Move($Destination,$Destination+'.moved') } catch { Write-Output 'attempt replacement blocked' }; throw 'original controlled copy failure' }"
        result = self.invoke(source, target, hook)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('attempt replacement blocked', result.stdout)
        self.assertIn('original controlled copy failure', result.stderr)
        self.assertFalse(target.exists())
        self.assertEqual(list(parent.glob('.insonic-install-*')), [])

