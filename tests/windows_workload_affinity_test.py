# SPDX-License-Identifier: Apache-2.0
import json
from pathlib import Path
import sys
from threading import Lock
import unittest
from unittest.mock import MagicMock,patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'scripts'))
from windows_workload_affinity import WorkloadAffinity,selected_mask


class WorkloadAffinityTests(unittest.TestCase):
    def test_only_one_of_exactly_four_available_processors_is_reserved(self):
        self.assertEqual(selected_mask(15,15,1),7)
        self.assertEqual(selected_mask(0b110110,0b110110,1),0b010110)
        for process,system,groups in [(0,0,1),(7,7,1),(31,31,1),(15,31,1),(15,15,2),(True,True,1)]:
            with self.subTest(process=process,system=system,groups=groups),self.assertRaises(ValueError):
                selected_mask(process,system,groups)

    def test_factory_keeps_override_environments_and_records_only_fixed_counts(self):
        selected=WorkloadAffinity.__new__(WorkloadAffinity)
        selected.mask=7;selected.permitted=15;selected.lock=Lock()
        selected.started=selected.verified=selected.failed=selected.breakaway=0
        tree=MagicMock()
        with patch('windows_workload_affinity.ProcessTree',return_value=tree) as launch:
            self.assertIs(selected(['private-executable'],cwd='private-path',env={'private':'value'},allow_detached=True),tree)
        self.assertEqual(launch.call_args.kwargs,{'cwd':'private-path','env':{'private':'value'},'allow_detached':True,'cpu_affinity':7})
        with patch('windows_workload_affinity.ProcessTree',side_effect=RuntimeError('private failure')):
            with self.assertRaises(RuntimeError):selected(['other-private'],cwd='.',env={})
        receipt=selected.receipt()
        self.assertEqual((receipt['launches'],receipt['verified_before_resume'],receipt['failed_launches'],receipt['breakaway_permitted_roots']),(2,1,1,1))
        self.assertFalse(receipt['parent_or_gui_affinity_changed'])
        self.assertFalse(receipt['feature_qualification'])
        self.assertNotIn('private',json.dumps(receipt))


if __name__=='__main__':unittest.main()
