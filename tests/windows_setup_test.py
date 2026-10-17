# SPDX-License-Identifier: Apache-2.0
import hashlib
import importlib.util
import json
from pathlib import Path
import struct
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('windows_setup', ROOT / 'scripts/windows_setup.py')
setup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)


class WindowsSetupTests(unittest.TestCase):
    def test_overlay_retains_exact_complete_msi_with_bounded_identity(self):
        with tempfile.TemporaryDirectory(dir=ROOT / 'build') as temporary:
            root = Path(temporary)
            executable, msi = root / 'setup.exe', root / 'source.msi'
            executable.write_bytes(b'MZ' + b'0' * 126)
            msi.write_bytes(b'exact complete cabinet MSI')
            identity = {'product_code': '{11111111-1111-5111-8111-111111111111}',
                        'upgrade_code': setup.m.UPGRADE_CODE, 'version': '1.0.1',
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
        manifest = ET.fromstring(setup.manifest('1.0.1'))
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
