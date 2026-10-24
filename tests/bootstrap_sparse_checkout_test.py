# SPDX-License-Identifier: Apache-2.0
"""Execute the actual sparse workflow closure without a repository fallback."""
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
import qualification_timing as timing


class SparseAutomationTests(unittest.TestCase):
    def test_every_sparse_bootstrap_and_toolchain_checkout_starts_in_isolation(self):
        checked = 0
        for workflow in sorted((ROOT / '.github/workflows').glob('*.yml')):
            lines = workflow.read_text(encoding='utf-8').splitlines()
            for index, line in enumerate(lines):
                if line.strip() != 'sparse-checkout: |':
                    continue
                indentation = len(line) - len(line.lstrip())
                selected = []
                for entry in lines[index + 1:]:
                    if entry.strip() and len(entry) - len(entry.lstrip()) <= indentation:
                        break
                    if entry.strip():
                        selected.append(entry.strip())
                commands = [name for name in selected if name in
                            ('scripts/bootstrap-actions.py', 'scripts/source-toolchain.py')]
                for command in commands:
                    with self.subTest(workflow=workflow.name, line=index + 1, command=command):
                        with tempfile.TemporaryDirectory() as temporary:
                            root = Path(temporary)
                            for relative in selected:
                                destination = root / relative
                                destination.parent.mkdir(parents=True, exist_ok=True)
                                shutil.copyfile(ROOT / relative, destination)
                            result = subprocess.run(
                                [sys.executable, '-I', str(root / command), '--help'],
                                cwd=root, stdin=subprocess.DEVNULL, capture_output=True,
                                timeout=timing.READINESS_SECONDS,
                                creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)
                            self.assertEqual(result.returncode, 0, result.stderr.decode('utf-8', 'replace'))
                        checked += 1
        self.assertGreaterEqual(checked, 12, 'Sparse workflow coverage disappeared')
