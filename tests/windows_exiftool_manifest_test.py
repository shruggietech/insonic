# SPDX-License-Identifier: Apache-2.0
import hashlib
import json
import os
from pathlib import Path
import shutil
import struct
import subprocess
import sys
import tempfile
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'scripts'))
from pe_resources import append_resources, read_resources, utf8_image, enable_utf8, verify_utf8
from windows_exiftool_manifest import transform, verify_transform, unsigned_image, prepare, verify_package, UPSTREAM_SHA256, TARGET, ORIGINAL, RECEIPT, RECIPES, main

XML = b'<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0"><application xmlns="urn:schemas-microsoft-com:asm.v3"><windowsSettings><longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware></windowsSettings></application></assembly>'

def pe_fixture():
    data=bytearray(1536);data[:2]=b'MZ';struct.pack_into('<I',data,60,128);data[128:132]=b'PE\0\0'
    struct.pack_into('<HHIIIHH',data,132,0x8664,1,1234,0,0,240,0x22)
    optional=152;struct.pack_into('<H',data,optional,0x20b)
    struct.pack_into('<II',data,optional+32,4096,512);struct.pack_into('<II',data,optional+56,8192,1024)
    struct.pack_into('<H',data,optional+68,3);struct.pack_into('<I',data,optional+108,16)
    struct.pack_into('<8sIIIIIIHHI',data,392,b'.text',32,4096,512,1024,0,0,0,0,0x60000020)
    data[1024:1040]=b'constructed-code'
    return append_resources(data,{24:{1:{1033:XML}},16:{1:{1033:b'original-version-data'}}},subsystem=3)

class ManifestTests(unittest.TestCase):
    def test_preserves_resources_and_reproduces_exact_unsigned_bytes(self):
        original=pe_fixture();pin=hashlib.sha256(original).hexdigest()
        changed,receipt=transform(original,expected_sha256=pin)
        self.assertEqual(transform(original,expected_sha256=pin),(changed,receipt))
        self.assertEqual(read_resources(changed,subsystem=3)[16,1,1033],b'original-version-data')
        self.assertIn(b'UTF-8',read_resources(changed,subsystem=3)[24,1,1033])
        self.assertIn(b'longPathAware',read_resources(changed,subsystem=3)[24,1,1033])
        self.assertEqual(original,pe_fixture())
        verify_transform(original,changed,receipt,expected_sha256=pin)
        corrupt=bytearray(changed);corrupt[1024]^=1
        with self.assertRaises(ValueError):verify_transform(original,corrupt,receipt,expected_sha256=pin)
        with self.assertRaises(ValueError):transform(original,expected_sha256='0'*64)
    def test_product_manifests_preserve_cli_and_desktop_resources(self):
        with tempfile.TemporaryDirectory() as temporary:
            for subsystem in [2,3]:
                original=bytearray(pe_fixture());struct.pack_into("<H",original,152+68,subsystem)
                executable=Path(temporary)/("product-"+str(subsystem)+".exe");executable.write_bytes(original)
                before=read_resources(original,subsystem=subsystem)
                with self.assertRaises(ValueError):verify_utf8(executable,subsystem=subsystem)
                enable_utf8(executable,subsystem=subsystem);verify_utf8(executable,subsystem=subsystem)
                once=executable.read_bytes();enable_utf8(executable,subsystem=subsystem)
                self.assertEqual(executable.read_bytes(),once)
                after=read_resources(executable.read_bytes(),subsystem=subsystem)
                self.assertEqual(after[16,1,1033],before[16,1,1033])
                self.assertEqual(executable.read_bytes()[1024:1040],original[1024:1040])
                executable.write_bytes(executable.read_bytes().replace(b"UTF-8",b"Other"))
                with self.assertRaises(ValueError):verify_utf8(executable,subsystem=subsystem)

    def test_product_manifest_is_added_to_an_image_without_resources(self):
        original=bytearray(pe_fixture());struct.pack_into("<II",original,152+112+16,0,0)
        self.assertEqual(read_resources(original,subsystem=3,allow_empty=True),{})
        changed,_=utf8_image(original)
        manifests=read_resources(changed,subsystem=3)
        self.assertEqual(set(manifests),{(24,1,1033)})
        self.assertIn(b"UTF-8",manifests[24,1,1033])
        self.assertEqual(changed[1024:1040],original[1024:1040])

    def test_existing_go_manifest_keeps_compatibility_without_application_settings(self):
        data=bytearray(pe_fixture())
        manifest=b'<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0"><compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1"><application><supportedOS Id="fixture"/></application></compatibility></assembly>'
        original=append_resources(data,{24:{1:{1033:manifest}}},subsystem=3)
        changed,_=utf8_image(original)
        value=read_resources(changed,subsystem=3)[24,1,1033]
        self.assertIn(b"supportedOS",value);self.assertIn(b"UTF-8",value)
        self.assertEqual(changed[1024:1040],original[1024:1040])

    def test_normalizes_only_authenticode_header_and_certificate(self):
        original=pe_fixture();changed,receipt=transform(original,expected_sha256=hashlib.sha256(original).hexdigest())
        signed=bytearray(changed);struct.pack_into('<I',signed,152+64,123456)
        struct.pack_into('<II',signed,152+112+32,len(signed),32);signed.extend(b'certificate-fixture'.ljust(32,b'\0'))
        self.assertEqual(unsigned_image(signed),unsigned_image(changed))
        with self.assertRaises(ValueError):verify_transform(original,signed,receipt,expected_sha256=hashlib.sha256(original).hexdigest())
        verify_transform(original,signed,receipt,expected_sha256=hashlib.sha256(original).hexdigest(),allow_signature=True)
        signed[-1:]=b''
        with self.assertRaises(ValueError):unsigned_image(signed)
    def test_wrong_input_cannot_overwrite_output_or_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);source=root/'upstream.bin';output=root/'launcher.exe'
            source.write_bytes(pe_fixture());output.write_bytes(b'unrelated')
            with self.assertRaises(ValueError):prepare(source,output)
            self.assertEqual(output.read_bytes(),b'unrelated');self.assertEqual(source.read_bytes(),pe_fixture())

    def test_receipt_alias_is_rejected_before_mutating_input_or_output(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);source=root/'upstream.bin';output=root/'launcher.exe'
            source.write_bytes(pe_fixture());output.write_bytes(b'unrelated')
            for receipt in [source, output]:
                with self.assertRaises(ValueError):main([str(source),str(output),'--receipt',str(receipt)])
                self.assertEqual(source.read_bytes(),pe_fixture());self.assertEqual(output.read_bytes(),b'unrelated')

    @unittest.skipUnless(os.name=='nt','actual hidden Windows process manifest')
    def test_pinned_launcher_runs_from_mixed_unicode_install_directory(self):
        source=Path(__file__).resolve().parents[1]/'build/native/media/exiftool/package/bin'
        if not (source/'exiftool.exe').is_file():self.skipTest('pinned native companion not prepared in this lane')
        self.assertEqual(hashlib.sha256((source/'exiftool.exe').read_bytes()).hexdigest(),UPSTREAM_SHA256)
        with tempfile.TemporaryDirectory(prefix='insonic manifest fixture ') as temporary:
            root=Path(temporary)/'program 文 café Ж ع with spaces'
            target=root/Path(TARGET).parent;shutil.copytree(source,target)
            receipt=prepare(source/'exiftool.exe',target/'exiftool.exe')
            result=subprocess.run([str(target/'exiftool.exe'),'-config','','-ver'],stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=10,cwd=target)
            self.assertEqual((result.returncode,result.stdout.strip(),result.stderr),(0,b'13.59',b''))
            verify_transform((source/'exiftool.exe').read_bytes(),(target/'exiftool.exe').read_bytes(),receipt)
            sources=root/'sources';sources.mkdir()
            shutil.copyfile(source/'exiftool.exe',sources/ORIGINAL)
            for name in RECIPES:shutil.copyfile(Path(__file__).resolve().parents[1]/'scripts'/name,sources/name)
            (sources/RECEIPT).write_text(json.dumps(receipt),encoding='utf-8')
            verify_package(root)
            changed=bytearray((target/'exiftool.exe').read_bytes());changed[-1]^=1;(target/'exiftool.exe').write_bytes(changed)
            with self.assertRaises(ValueError):verify_package(root)

if __name__=='__main__':unittest.main()
