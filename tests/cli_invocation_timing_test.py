# SPDX-License-Identifier: Apache-2.0
import importlib.util
from pathlib import Path
import sys
import unittest
from unittest.mock import MagicMock, Mock, patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))
spec = importlib.util.spec_from_file_location("cli_timing", ROOT / "scripts/cli_contract_qualification.py")
cli = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cli)


class InvocationTimingTests(unittest.TestCase):
    def test_nested_allowance_includes_launch_and_expiry_still_closes_owner(self):
        tree = MagicMock()
        tree.__enter__.return_value = tree
        tree.process.communicate.return_value = (b'', b'')
        tree.process.returncode = 0
        with patch.object(cli.time, 'monotonic', side_effect=[10, 12, 12.5, 12.6]):
            cli.capture_invocation(Mock(return_value=tree), ['fixture'], cwd=ROOT, env={}, scoped=True,
                                   input=None, timeout_seconds=5)
        self.assertEqual(tree.process.communicate.call_args.kwargs['timeout'], 3)
        tree.reset_mock()
        with patch.object(cli.time, 'monotonic', side_effect=[10, 16]), self.assertRaises(TimeoutError):
            cli.capture_invocation(Mock(return_value=tree), ['fixture'], cwd=ROOT, env={}, scoped=True,
                                   input=None, timeout_seconds=5)
        tree.process.communicate.assert_not_called()
        tree.__exit__.assert_called_once()

    def test_owned_launch_and_process_wait_both_remain_in_total(self):
        tree = MagicMock()
        tree.__enter__.return_value = tree
        tree.process.communicate.return_value = (b"output", b"error")
        tree.process.returncode = 4
        factory = Mock(return_value=tree)
        with patch.object(cli.time, "monotonic", side_effect=[10, 12, 12.8, 12.9]):
            result = cli.capture_invocation(factory, ["fixture"], cwd=ROOT, env={}, scoped=True, input=b"fixture")
        self.assertEqual(result[:3], (b"output", b"error", 4))
        timing = result[3]
        self.assertAlmostEqual(timing["owned_launch_seconds"], 2)
        self.assertAlmostEqual(timing["process_wait_seconds"], .8)
        self.assertAlmostEqual(timing["total_seconds"], 2.9)
        self.assertTrue(factory.call_args.kwargs["input_pipe"])
        self.assertTrue(factory.call_args.kwargs["allow_detached"])

    def test_correct_classification_keeps_expanded_readiness_limit(self):
        response = {"error": {"code": "unavailable", "reason": "credential_bootstrap"}}
        timing = {"owned_launch_seconds": 3, "process_wait_seconds": 1, "total_seconds": 4.1}
        cli.verify_cold_bootstrap(response, timing)
        for changed in ({**timing, "process_wait_seconds": cli.bounds.READINESS_SECONDS + .01, "total_seconds": cli.bounds.READINESS_SECONDS + 3.1},
                        {**timing, "total_seconds": .5}, {**timing, "owned_launch_seconds": float("nan")}):
            with self.subTest(timing=changed), self.assertRaises(ValueError):
                cli.verify_cold_bootstrap(response, changed)
        with self.assertRaises(ValueError):
            cli.verify_cold_bootstrap({"error": {"code": "unavailable", "reason": "other"}}, timing)


if __name__ == "__main__":
    unittest.main()
