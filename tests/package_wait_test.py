# SPDX-License-Identifier: Apache-2.0
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import MagicMock, Mock, patch
import json

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("package_wait", ROOT / "scripts/package-desktop.py")
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)


def response(state="running", work_id="work", code="unavailable", reason="wait_timeout"):
    return {"kind": "runtime-response", "result": {"id": work_id, "work_id": work_id, "state": state},
            "error": {"code": code, "reason": reason}}


class DurablePackageWaitTests(unittest.TestCase):
    def test_only_requested_structured_cli_failures_retain_machine_response(self):
        tree = MagicMock()
        tree.__enter__.return_value = tree
        tree.process.returncode = 2
        tree.process.communicate.return_value = (b"", json.dumps(response()).encode())
        with self.assertRaises(package.PackageCLIResponseError) as failure:
            package.child(["fixture"], process_factory=Mock(return_value=tree), cli_response=True)
        self.assertEqual(failure.exception.response["result"]["id"], "work")
        with self.assertRaises(RuntimeError) as ordinary:
            package.child(["fixture"], process_factory=Mock(return_value=tree))
        self.assertNotIsInstance(ordinary.exception, package.PackageCLIResponseError)

    def test_pending_wait_retains_the_original_operation_deadline(self):
        request = Mock(side_effect=[package.PackageCLIResponseError(response()),
                                   {"result": {"id": "work", "state": "succeeded"}}])
        with patch.object(package.time, "monotonic", side_effect=[1000, 1000, 1031, 1061]):
            self.assertEqual(package.wait_for_work(request, "work")["state"], "succeeded")
        self.assertEqual(request.call_args_list[0].args, (30000, 180))
        self.assertEqual(request.call_args_list[1].args, (30000, 149))

    def test_repeated_pending_results_cannot_restart_the_bound(self):
        request = Mock(side_effect=package.PackageCLIResponseError(response()))
        with patch.object(package.time, "monotonic", side_effect=[0, 0, 31, 180]):
            with self.assertRaisesRegex(RuntimeError, "operation bound"):
                package.wait_for_work(request, "work")
        self.assertEqual(request.call_count, 2)

    def test_foreign_failed_or_other_error_is_never_retried(self):
        for value in (response(work_id="foreign"), response(state="failed"),
                      response(code="cancelled"), response(reason="other")):
            error = package.PackageCLIResponseError(value)
            request = Mock(side_effect=error)
            with self.subTest(value=value), patch.object(package.time, "monotonic", return_value=0):
                with self.assertRaises(package.PackageCLIResponseError) as raised:
                    package.wait_for_work(request, "work")
                self.assertIs(raised.exception, error)
                self.assertEqual(request.call_count, 1)

    def test_late_or_mismatched_success_cannot_qualify(self):
        for result, clock in (({"id": "work", "state": "succeeded"}, [0, 0, 181]),
                              ({"id": "foreign", "state": "succeeded"}, [0, 0]),
                              ({"id": "work", "state": "running"}, [0, 0])):
            with self.subTest(result=result), patch.object(package.time, "monotonic", side_effect=clock):
                with self.assertRaises(RuntimeError):
                    package.wait_for_work(Mock(return_value={"result": result}), "work")


if __name__ == "__main__":
    unittest.main()
