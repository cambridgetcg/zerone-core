#!/usr/bin/env python3
"""Offline tests for h1-proposal.py. Chain responses here are UNIT MOCKS."""
import base64
import contextlib
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import provenance as p
from test_kit import IMAGE, load

tool = load("h1_proposal", "h1-proposal.py")
HERE = Path(__file__).resolve().parent
FAUCET = "zrn1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5"


def chain(latest=3_870_000, *, seconds=2.53, deposit="10000000", voting="120s", plan=None, network="zerone-testnet-1",
          authority="zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89"):
    """A mock RPC/REST pair answering only the GETs the tool makes."""
    def get_json(base, path, params=None):
        if path == "/status":
            return {"result": {"node_info": {"network": network},
                               "sync_info": {"latest_block_height": str(latest),
                                             "latest_block_time": "2026-11-06T12:00:00.123456789Z"}}}
        if path == "/header":
            height = int(params["height"])
            micro = round((height - latest) * seconds * 1_000_000)
            moment = tool._parse_time("2026-11-06T12:00:00.123456Z") + tool.dt.timedelta(microseconds=micro)
            return {"result": {"header": {"time": moment.strftime("%Y-%m-%dT%H:%M:%S.%fZ")}}}
        if path == "/cosmos/gov/v1/params/deposit":
            return {"params": {"min_deposit": [{"denom": "uzrn", "amount": deposit}], "max_deposit_period": "300s",
                               "voting_period": voting, "min_initial_deposit_ratio": "0.000000000000000000"}}
        if path == "/cosmos/auth/v1beta1/module_accounts/gov":
            return {"account": {"base_account": {"address": authority}, "name": "gov"}}
        if path == "/cosmos/upgrade/v1beta1/current_plan":
            return {"plan": plan}
        raise AssertionError(path)
    return get_json


class PlanInfoTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="h1-plan-info-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def test_port_matches_recorded_go_verdicts(self):
        corpus = json.loads((HERE / "plan-info-corpus.json").read_text())
        self.assertEqual(corpus["source_commit"], p.SOURCE_COMMIT)
        self.assertGreater(len(corpus["cases"]), 100)
        for index, case in enumerate(corpus["cases"]):
            raw = base64.b64decode(case["info_b64"])
            try:
                tool.validate_canonical_info(raw)
                verdict = True
            except tool.Refused:
                verdict = False
            self.assertEqual(verdict, case["ok"], f"case {index}: {raw[:60]!r} Go said {case['go_error']!r}")
        # The corpus must exercise both outcomes, including the failing habits seen live.
        refused = {base64.b64decode(c["info_b64"]) for c in corpus["cases"] if not c["ok"]}
        self.assertIn(b"", refused)
        self.assertIn(b"<binary checksums>", refused)

    def test_info_is_the_kits_plan_info_and_canonical(self):
        info = tool.canonical_info(IMAGE)
        self.assertEqual(info, tool.ship.plan_info(IMAGE))
        self.assertEqual(json.loads(info), {"commit": p.SOURCE_COMMIT, "image": IMAGE})
        tool.validate_canonical_info(info)

    def test_live_habits_are_refused(self):
        # zerone-1 proposal 6 and testnet proposals 2 and 3 used Info "".
        for info in ("", "<binary checksums>", json.dumps({"image": IMAGE, "commit": p.SOURCE_COMMIT}),
                     json.dumps({"commit": p.SOURCE_COMMIT, "image": IMAGE}, sort_keys=True, indent=2),
                     '{"image":"https://example.invalid/a?x=1&y=2"}'):
            with self.subTest(info=info), self.assertRaises(tool.Refused):
                tool.validate_canonical_info(info)

    def test_unsigned_tx_pins_authority_deposit_and_window(self):
        info, tx = tool.unsigned_tx("zerone-testnet-1", IMAGE, 3_871_000, FAUCET, "300000", "300000uzrn",
                                    3_870_500, tool.PLAN_NAME, "unit")
        self.assertEqual(tx["signatures"], [])
        self.assertEqual(tx["auth_info"]["signer_infos"], [])
        self.assertEqual(tx["body"]["timeout_height"], "3870500")
        (submit,) = tx["body"]["messages"]
        self.assertEqual(submit["initial_deposit"], [{"denom": "uzrn", "amount": "10000000"}])
        self.assertEqual(submit["proposer"], FAUCET)
        self.assertFalse(submit["expedited"])
        (upgrade,) = submit["messages"]
        self.assertEqual(upgrade["authority"], "zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89")
        self.assertEqual(upgrade["plan"], {"name": "consolidation-safety-v1", "time": "0001-01-01T00:00:00Z",
                                           "height": "3871000", "info": info, "upgraded_client_state": None})
        self.assertEqual(tool.check_unsigned(tx, "zerone-testnet-1", IMAGE, 3_871_000), info)
        _, mainnet = tool.unsigned_tx("zerone-1", IMAGE, 1_500_000, FAUCET, "300000", "300000uzrn", 1, "t", "s")
        self.assertEqual(mainnet["body"]["messages"][0]["initial_deposit"], [{"denom": "uzrn", "amount": "100000000"}])

    def test_unsigned_tx_drift_is_refused(self):
        _, tx = tool.unsigned_tx("zerone-testnet-1", IMAGE, 3_871_000, FAUCET, "300000", "300000uzrn", 3_870_500, "t", "s")
        for mutate in (lambda m: m["initial_deposit"][0].update(amount="1"),
                       lambda m: m.update(expedited=True),
                       lambda m: m["messages"][0]["plan"].update(info=""),
                       lambda m: m["messages"][0]["plan"].update(height="3871001"),
                       lambda m: m["messages"][0].update(authority=FAUCET)):
            altered = json.loads(json.dumps(tx))
            mutate(altered["body"]["messages"][0])
            with self.assertRaises(tool.Refused):
                tool.check_unsigned(altered, "zerone-testnet-1", IMAGE, 3_871_000)
        with self.assertRaises(ValueError):
            tool.unsigned_tx("zerone-testnet-1", IMAGE, 1, "cosmos1notzrn", "300000", "300000uzrn", 1, "t", "s")
        with self.assertRaises(ValueError):
            tool.unsigned_tx("zerone-testnet-1", IMAGE, 1, FAUCET, "auto", "300000uzrn", 1, "t", "s")

    def test_window_uses_the_fastest_cadence(self):
        observed = {"latest_height": 1000, "block_seconds_min_window": 2.0, "block_seconds_max_window": 3.0,
                    "block_seconds_mean": 2.5, "voting_period_seconds": 120}
        window = tool.height_window(observed, 600, 900)
        self.assertEqual(window["earliest_h"], 1000 + (600 + 120 + 900) // 2)
        self.assertEqual(tool.submission_deadline(2000, window), 2000 - (120 + 900) // 2)

    def test_profile_refuses_param_drift_and_pending_plan(self):
        for overrides in ({"deposit": "1"}, {"voting": "600s"}, {"plan": {"name": "other", "height": "5"}},
                          {"authority": FAUCET}):
            with self.subTest(overrides=overrides), mock.patch.object(tool, "get_json", side_effect=chain(**overrides)):
                observed = tool.observe("http://rpc.invalid", "http://rest.invalid", "zerone-testnet-1", 40)
                with self.assertRaises(tool.Refused):
                    tool.check_profile("zerone-testnet-1", observed)
        with mock.patch.object(tool, "get_json", side_effect=chain(network="zerone-1")), self.assertRaises(tool.Refused):
            tool.observe("http://rpc.invalid", "http://rest.invalid", "zerone-testnet-1", 40)

    def test_draft_refuses_zerone_1(self):
        # The kit builds the testnet image; no mainnet bar exists, so no zerone-1 draft is written.
        with mock.patch.object(tool, "get_json", side_effect=AssertionError("no GET before the refusal")), \
                self.assertRaises(tool.Refused):
            tool.draft(self.draft_args(chain_id="zerone-1"))
        self.assertFalse((self.root / "draft").exists())

    def draft_args(self, **overrides):
        values = dict(chain_id="zerone-testnet-1", rpc="http://rpc.invalid:26657", rest="http://rest.invalid:1317",
                      image=IMAGE, proposer=FAUCET, height=None, extra_blocks=120, submit_seconds=600,
                      after_pass_seconds=900, gas="300000", fees="300000uzrn", summary="unit", output=self.root / "draft")
        values.update(overrides)
        return tool.argparse.Namespace(**values)

    def test_draft_writes_unsigned_files_and_never_runs_a_command(self):
        with mock.patch.object(tool, "get_json", side_effect=chain()), \
             mock.patch.object(subprocess, "run", side_effect=AssertionError("command run")), \
             mock.patch.object(subprocess, "check_output", side_effect=AssertionError("command run")):
            receipt = tool.draft(self.draft_args())
        tx = json.loads((self.root / "draft/unsigned-tx.json").read_text())
        self.assertEqual(receipt["unsigned_tx_sha256"], p.sha256(self.root / "draft/unsigned-tx.json"))
        self.assertEqual(tool.check_unsigned(tx, "zerone-testnet-1", IMAGE, receipt["height"]), receipt["info"])
        self.assertGreaterEqual(receipt["height"], receipt["window"]["earliest_h"])
        self.assertLess(receipt["timeout_height"], receipt["height"])
        self.assertGreater(receipt["timeout_height"], receipt["observed"]["latest_height"])
        self.assertFalse(receipt["signed"] or receipt["broadcast"])
        proposal = json.loads((self.root / "draft/proposal.json").read_text())
        self.assertEqual(proposal["deposit"], "10000000uzrn")
        self.assertEqual(proposal["messages"], tx["body"]["messages"][0]["messages"])

    def test_draft_refuses_too_soon_and_existing_output(self):
        with mock.patch.object(tool, "get_json", side_effect=chain()), self.assertRaisesRegex(tool.Refused, "too soon"):
            tool.draft(self.draft_args(height=3_870_100))
        (self.root / "draft").mkdir()
        with mock.patch.object(tool, "get_json", side_effect=chain()), self.assertRaises(ValueError):
            tool.draft(self.draft_args())

    def test_check_plan_is_byte_exact(self):
        info = tool.canonical_info(IMAGE)
        good = {"name": "consolidation-safety-v1", "height": "3871000", "info": info,
                "time": "0001-01-01T00:00:00Z", "upgraded_client_state": None}
        args = tool.argparse.Namespace(rest="http://rest.invalid", image=IMAGE, height=3_871_000)
        with mock.patch.object(tool, "get_json", side_effect=chain(plan=good)):
            self.assertEqual(tool.check_plan(args)["status"], "plan-matches")
        for change in ({"info": ""}, {"info": info + " "}, {"height": "3871001"}, {"name": "x"},
                       {"time": "2026-01-01T00:00:00Z"}):
            with self.subTest(change=change), mock.patch.object(tool, "get_json", side_effect=chain(plan=dict(good, **change))), \
                 self.assertRaises(tool.Refused):
                tool.check_plan(args)
        with mock.patch.object(tool, "get_json", side_effect=chain(plan=None)), self.assertRaises(tool.Refused):
            tool.check_plan(args)

    def test_cli_refuses_empty_info(self):
        with mock.patch.object(sys, "argv", ["h1-proposal.py", "validate-info", "--info", ""]), \
             contextlib.redirect_stderr(io.StringIO()) as errors:
            self.assertEqual(tool.main(), 1)
        self.assertIn("non-empty", errors.getvalue())

    def test_endpoint_validation(self):
        for url in ("ftp://x", "http://user:pw@x", "http://x/path", "http://x?q=1"):
            with self.assertRaises(tool.Refused):
                tool._endpoint(url, "rest")
        self.assertEqual(tool._endpoint("http://37.16.28.121:1317", "rest"), "http://37.16.28.121:1317")


if __name__ == "__main__":
    unittest.main(verbosity=2)
