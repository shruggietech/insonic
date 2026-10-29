# SPDX-License-Identifier: Apache-2.0
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import struct
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('windows_setup', ROOT / 'scripts/windows_setup.py')
setup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)


class WindowsSetupTests(unittest.TestCase):
    def test_source_rebuild_does_not_receive_publication_credentials_or_executable_overrides(self):
        with patch.dict(os.environ, {'GH_TOKEN': 'private sentinel', 'GITHUB_TOKEN': 'other private sentinel',
                                     'INSONIC_DOCUMENTATION_DEPLOYMENT_CREDENTIAL': 'private deployment sentinel',
                                     'GOFLAGS': '-toolexec=untrusted', 'GOROOT': 'untrusted', 'CGO_ENABLED': '1'}):
            environment = setup.compiler_environment(isolated=True)
        for name in ('GH_TOKEN', 'GITHUB_TOKEN', 'INSONIC_DOCUMENTATION_DEPLOYMENT_CREDENTIAL', 'GOROOT'):
            self.assertNotIn(name, environment)
        self.assertEqual(environment['GOFLAGS'], '')
        self.assertEqual(environment['CGO_ENABLED'], '0')
        self.assertEqual(environment['GOWORK'], 'off')
        self.assertEqual(environment['GOTOOLCHAIN'], 'local')
        self.assertTrue(Path(environment['GOCACHE']).is_relative_to(ROOT / 'build'))

    def test_overlay_retains_exact_complete_msi_with_bounded_identity(self):
        with tempfile.TemporaryDirectory(dir=ROOT / 'build') as temporary:
            root = Path(temporary)
            executable, msi = root / 'setup.exe', root / 'source.msi'
            executable.write_bytes(b'MZ' + b'0' * 126)
            msi.write_bytes(b'exact complete cabinet MSI')
            identity = {'product_code': '{11111111-1111-5111-8111-111111111111}',
                        'upgrade_code': setup.m.UPGRADE_CODE, 'version': '1.0.3',
                        'sha256': setup.m.sha(msi), 'size_bytes': msi.stat().st_size}
            setup.append_payload(executable, msi, identity)
            actual, offset = setup.read_payload(executable)
            self.assertEqual(actual, identity)
            with executable.open('rb') as source:
                source.seek(offset)
                self.assertEqual(source.read(identity['size_bytes']), msi.read_bytes())
            data = bytearray(executable.read_bytes()); data[128] ^= 1
            executable.write_bytes(data)
            with self.assertRaisesRegex(ValueError, 'digest'):
                setup.read_payload(executable)

    def test_bounds_cannot_select_outside_executable(self):
        with tempfile.TemporaryDirectory(dir=ROOT / 'build') as temporary:
            path = Path(temporary) / 'setup.exe'
            path.write_bytes(b'MZ' * 64 + b'{}' + setup.MAGIC + struct.pack('<QII', 2**63, 2, 0))
            with self.assertRaises(ValueError): setup.read_payload(path)

    def test_manifest_declares_as_invoker_and_preserves_approved_icon(self):
        import xml.etree.ElementTree as ET
        manifest = ET.fromstring(setup.manifest('1.0.3'))
        requested = manifest.find('.//{urn:schemas-microsoft-com:asm.v3}requestedExecutionLevel')
        self.assertEqual(requested.attrib, {'level': 'asInvoker', 'uiAccess': 'false'})
        icon = (ROOT / setup.m.ICON).read_bytes()
        images, group = setup.icon_resources(icon)
        self.assertTrue(images)
        self.assertEqual(group[:6], icon[:6])
        for index, image in enumerate(images):
            size, offset = struct.unpack_from('<II', icon, 6 + index * 16 + 8)
            self.assertEqual(image, icon[offset:offset + size])

    def test_invalid_icon_offsets_cannot_read_unowned_bytes(self):
        icon = bytearray((ROOT / setup.m.ICON).read_bytes())
        struct.pack_into('<I', icon, 18, 0xffffffff)
        with self.assertRaises(ValueError): setup.icon_resources(icon)


class WindowsSetupSourceBindingTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temporary = tempfile.TemporaryDirectory(dir=ROOT / 'build')
        cls.root = Path(cls.temporary.name)
        cls.prefix = setup.compile_prefix(cls.root / 'first/setup.exe', '1.0.3')

    @classmethod
    def tearDownClass(cls): cls.temporary.cleanup()

    def test_reproducible_code_and_resources_reject_forged_build_lane_receipts(self):
        other = setup.compile_prefix(self.root / 'different output directory/setup.exe', '1.0.3')
        self.assertEqual(other.read_bytes(), self.prefix.read_bytes())
        target = self.root / 'insonic_1.0.3_windows_amd64_setup.exe'
        target.write_bytes(self.prefix.read_bytes())
        msi = self.root / 'controlled.msi'; msi.write_bytes(b'controlled exact MSI overlay')
        identity = {'product_code': '{11111111-1111-5111-8111-111111111111}',
                    'upgrade_code': setup.m.UPGRADE_CODE, 'version': '1.0.3',
                    'sha256': setup.m.sha(msi), 'size_bytes': msi.stat().st_size}
        setup.append_payload(target, msi, identity)
        receipt = {'kind': 'windows-setup-receipt', 'format': 'exe', 'artifact': target.name,
                   'scope': 'per-user', 'manifest': 'asInvoker', 'signing': dict(setup.m.UNSIGNED_SIGNING),
                   'embedded_msi': identity, 'qualification': {key: 'passed' for key in setup.QUALIFICATION_KEYS}}
        installer = {'product_version': '1.0.3', 'product_code': identity['product_code'],
                     'upgrade_code': identity['upgrade_code'], 'artifact_sha256': identity['sha256'],
                     'artifact_size_bytes': identity['size_bytes']}
        def verify():
            receipt.update(artifact_sha256=setup.m.sha(target), artifact_size_bytes=target.stat().st_size)
            setup.verify(target, receipt, installer)
        verify()
        original = target.read_bytes(); pe = struct.unpack_from('<I', original, 60)[0]
        count = struct.unpack_from('<H', original, pe + 6)[0]
        optional_size = struct.unpack_from('<H', original, pe + 20)[0]
        table = pe + 24 + optional_size
        text = next(struct.unpack_from('<I', original, table + i * 40 + 20)[0]
                    for i in range(count) if original[table + i * 40:table + i * 40 + 8].rstrip(b'\0') == b'.text')
        for position in (text + 128, pe + 24 + 16):
            with self.subTest(position=position):
                changed = bytearray(original); changed[position] ^= 1; target.write_bytes(changed)
                # Both the appended MSI and approved resources are still valid;
                # the attacker also controls this lane's digest/qualification.
                self.assertEqual(setup.read_payload(target)[0], identity)
                setup.verify_resources(target, '1.0.3')
                with self.assertRaisesRegex(ValueError, 'independently rebuilt source'): verify()

    @unittest.skipUnless(os.name == 'nt', 'read-only Windows resource loader')
    def test_windows_reads_portable_resources_without_executing_the_program(self):
        import ctypes as C
        dll = C.WinDLL('kernel32', use_last_error=True)
        dll.LoadLibraryExW.argtypes = [C.c_wchar_p, C.c_void_p, C.c_uint32]; dll.LoadLibraryExW.restype = C.c_void_p
        dll.FindResourceExW.argtypes = [C.c_void_p, C.c_void_p, C.c_void_p, C.c_ushort]; dll.FindResourceExW.restype = C.c_void_p
        dll.LoadResource.argtypes = [C.c_void_p, C.c_void_p]; dll.LoadResource.restype = C.c_void_p
        dll.SizeofResource.argtypes = [C.c_void_p, C.c_void_p]; dll.SizeofResource.restype = C.c_uint32
        dll.LockResource.argtypes = [C.c_void_p]; dll.LockResource.restype = C.c_void_p
        dll.FreeLibrary.argtypes = [C.c_void_p]; dll.FreeLibrary.restype = C.c_int
        module = dll.LoadLibraryExW(str(self.prefix), None, 0x60)
        if not module: raise C.WinError(C.get_last_error())
        try:
            resource = dll.FindResourceExW(module, 24, 1, 1033)
            if not resource: raise C.WinError(C.get_last_error())
            size = dll.SizeofResource(module, resource)
            pointer = dll.LockResource(dll.LoadResource(module, resource))
            self.assertEqual(C.string_at(pointer, size), setup.manifest('1.0.3'))
        finally: dll.FreeLibrary(module)
