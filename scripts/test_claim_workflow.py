#!/usr/bin/env python3
"""Private persistence and refusal tests; real CLI signing is tested separately."""
import argparse
import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location("claim_workflow", Path(__file__).with_name("claim-workflow.py"))
workflow = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(workflow)


class WorkflowTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="zerone-claim-workflow-test-")
        self.addCleanup(self.temp.cleanup)
        self.home = Path(self.temp.name).resolve()
        self.home.chmod(0o700)
        self.w = object.__new__(workflow.Workflow)
        self.w.home = self.home
        self.w.directory = self.home / "claim-workflow"
        for path in (self.w.directory, self.w.directory / "attempts", self.w.directory / "reviews"):
            workflow.private_directory(path)
        self.w.chain = "zerone-local-test"
        self.w.rpc = "http://127.0.0.1:47657"
        self.w.accounts = {actor: "zrn1" + str(index + 2) * 38 for index, actor in enumerate(workflow.ACTORS)}
        self.w.wait = 0.02
        self.w.check_node = mock.Mock(return_value=20)
        self.w.actor = mock.Mock(side_effect=lambda actor: self.w.accounts[actor])
        self.raw = b"unit-test-signed-transaction-not-a-real-tx"
        self.encoded = base64.b64encode(self.raw).decode()
        self.digest = hashlib.sha256(self.raw).hexdigest().upper()
        self.observation = {"txhash": self.digest, "height": "22", "code": 0, "gas_used": "210000", "gas_wanted": "2000000",
                            "events": [{"type": "zerone.knowledge.submit_claim", "attributes": [{"key": "claim_id", "value": "claim-1"}]}]}
        self.calls = []

        def cli(*args, **kwargs):
            self.calls.append(args)
            if args[:2] == ("tx", "encode"):
                return self.encoded
            if args[:2] == ("tx", "broadcast"):
                return json.dumps({"txhash": self.digest, "code": 0})
            return json.dumps({"body": {"messages": [{"@type": "test"}]}, "auth_info": {}, "signatures": []})
        self.w.cli = mock.Mock(side_effect=cli)
        self.w.tx_observation = mock.Mock(return_value=self.observation)

    def attempt(self, status="pending", action="commit", actor="reviewer1", context=None):
        value = {"schema": workflow.SCHEMA, "chain_id": self.w.chain, "id": "a" * 32, "txhash": self.digest,
                 "action": action, "actor": actor, "status": status, "tx_bytes_base64": self.encoded,
                 "context": context or {"claim_id": "claim-1", "round_id": "round-1"}}
        path = self.w.directory / "attempts" / (value["id"] + ".json")
        workflow.save_json(path, value, fresh=True)
        workflow.save_json(self.w.directory / (value["id"] + "-signed.json"), {"body": {}}, fresh=True)
        return path, value

    def review_args(self, **overrides):
        values = dict(claim="claim-1", actor="reviewer1", vote="accept", confidence=800000, reason="PRIVATE reason",
                      scope="This example only", method="", evidence=["ref:b", "ref:a"], wait_for_phase=False)
        values.update(overrides)
        return argparse.Namespace(**values)

    def row(self, phase="VERIFICATION_PHASE_COMMIT"):
        return {"id": "round-1", "claim_id": "claim-1", "phase": phase, "commit_deadline": "60", "reveal_deadline": "120"}

    def save_review(self):
        self.w.round_for_claim = mock.Mock(return_value=self.row())
        def fail_before_send(*args, **kwargs):
            self.assertEqual(workflow.read_json(self.w.review_path("claim-1", "reviewer1"))["reason"], "PRIVATE reason")
            raise workflow.WorkflowError("interrupted before signing")
        self.w.send = mock.Mock(side_effect=fail_before_send)
        with self.assertRaisesRegex(workflow.WorkflowError, "interrupted"):
            self.w.review(self.review_args())
        path = self.w.review_path("claim-1", "reviewer1")
        return path, workflow.read_json(path)

    def test_signed_bytes_and_hash_are_durable_before_broadcast(self):
        original = self.w.cli.side_effect
        def cli(*args, **kwargs):
            if args[:2] == ("tx", "broadcast"):
                records = self.w.records("attempts")
                self.assertEqual(len(records), 1)
                self.assertEqual(records[0][1]["tx_bytes_base64"], self.encoded)
                self.assertEqual(records[0][1]["txhash"], self.digest)
                self.assertEqual(records[0][1]["status"], "broadcasting")
            return original(*args, **kwargs)
        self.w.cli.side_effect = cli
        receipt = self.w.send("submit", "user", ["knowledge", "submit-claim", "content"])
        self.assertEqual(receipt["claim_id"], "claim-1")
        self.assertEqual(receipt["height"], "22")
        self.assertEqual(receipt["gas_used"], "210000")
        self.assertEqual(sum(call[:2] == ("tx", "broadcast") for call in self.calls), 1)

    def test_ambiguous_broadcast_blocks_new_signing_and_never_auto_retries(self):
        original = self.w.cli.side_effect
        def cli(*args, **kwargs):
            if args[:2] == ("tx", "broadcast"):
                raise subprocess.TimeoutExpired("broadcast", 60)
            return original(*args, **kwargs)
        self.w.cli.side_effect = cli
        self.w.tx_observation.return_value = None
        with self.assertRaisesRegex(workflow.WorkflowError, "unresolved"):
            self.w.send("submit", "user", ["knowledge", "submit-claim", "content"])
        before = self.w.cli.call_count
        with self.assertRaisesRegex(workflow.WorkflowError, "earlier transaction"):
            self.w.send("submit", "user", ["knowledge", "submit-claim", "content"])
        self.assertEqual(self.w.cli.call_count, before)
        self.assertEqual(self.w.records("attempts")[0][1]["status"], "broadcasting")

    def test_prepared_crash_explicit_retry_uses_exact_bytes_without_signing(self):
        path, _ = self.attempt("prepared")
        self.w.tx_observation.side_effect = [None, self.observation]
        result = self.w.retry(self.digest)
        self.assertEqual(result["status"], "committed")
        self.assertFalse(any(call[:2] == ("tx", "sign") for call in self.calls))
        self.assertEqual(workflow.read_json(path)["tx_bytes_base64"], self.encoded)
        self.assertEqual(sum(call[:2] == ("tx", "broadcast") for call in self.calls), 1)

    def test_retry_first_observes_existing_commit_without_broadcast(self):
        self.attempt("pending")
        result = self.w.retry(self.digest)
        self.assertEqual(result["height"], "22")
        self.w.cli.assert_not_called()

    def test_retry_checktx_refusal_does_not_release_unresolved_original(self):
        path, value = self.attempt("pending")
        value["broadcast_count"] = 1
        workflow.save_json(path, value)
        original = self.w.cli.side_effect
        def cli(*args, **kwargs):
            if args[:2] == ("tx", "broadcast"):
                return json.dumps({"txhash": self.digest, "code": 19, "raw_log": "tx already exists in cache"})
            return original(*args, **kwargs)
        self.w.cli.side_effect = cli
        self.w.tx_observation.return_value = None
        with self.assertRaisesRegex(workflow.WorkflowError, "pending"):
            self.w.retry(self.digest)
        retained = workflow.read_json(path)
        self.assertEqual(retained["status"], "pending")
        self.assertEqual(retained["check_tx"]["code"], 19)
        self.assertEqual(retained["tx_bytes_base64"], self.encoded)
        count = self.w.cli.call_count
        with self.assertRaisesRegex(workflow.WorkflowError, "earlier transaction"):
            self.w.send("submit", "user", ["knowledge", "submit-claim", "different content"])
        self.assertEqual(self.w.cli.call_count, count)
        self.w.tx_observation.return_value = self.observation
        self.assertEqual(self.w.reconcile(), [])
        self.assertEqual(workflow.read_json(path)["status"], "committed")
        self.assertEqual(workflow.read_json(path)["code"], 0)

    def test_retry_checktx_refusal_can_observe_original_commit_without_new_signature(self):
        self.attempt("broadcasting")
        original = self.w.cli.side_effect
        self.w.cli.side_effect = lambda *args, **kwargs: json.dumps({"txhash": self.digest, "code": 32}) if args[:2] == ("tx", "broadcast") else original(*args, **kwargs)
        self.w.tx_observation.side_effect = [None, self.observation]
        self.assertEqual(self.w.retry(self.digest)["status"], "committed")
        self.assertFalse(any(call.args[:2] == ("tx", "sign") for call in self.w.cli.call_args_list))

    def test_retry_refuses_changed_signed_transaction(self):
        self.attempt("prepared")
        self.w.tx_observation.return_value = None
        self.w.cli.return_value = base64.b64encode(b"different bytes").decode()
        self.w.cli.side_effect = None
        with self.assertRaisesRegex(workflow.WorkflowError, "changed"):
            self.w.retry(self.digest)
        self.assertEqual(self.w.cli.call_args.args[:2], ("tx", "encode"))

    def test_retry_refuses_tampered_retained_hash(self):
        path, value = self.attempt("prepared")
        value["tx_bytes_base64"] = base64.b64encode(b"tampered").decode()
        workflow.save_json(path, value)
        self.w.tx_observation.return_value = None
        with self.assertRaisesRegex(workflow.WorkflowError, "match their hash"):
            self.w.retry(self.digest)
        self.w.cli.assert_not_called()

    def test_committed_failure_is_retained_and_not_called_success(self):
        self.observation["code"] = 5
        with self.assertRaisesRegex(workflow.WorkflowError, "ended failed"):
            self.w.send("submit", "user", ["knowledge", "submit-claim", "content"])
        row = self.w.records("attempts")[0][1]
        self.assertEqual((row["status"], row["code"], row["height"]), ("failed", 5, "22"))
        self.assertNotIn("claim_id", row)

    def test_checktx_failure_has_no_fake_committed_receipt(self):
        original = self.w.cli.side_effect
        def cli(*args, **kwargs):
            if args[:2] == ("tx", "broadcast"):
                return json.dumps({"txhash": self.digest, "code": 11})
            return original(*args, **kwargs)
        self.w.cli.side_effect = cli
        with self.assertRaisesRegex(workflow.WorkflowError, "CheckTx rejected"):
            self.w.send("submit", "user", ["knowledge", "submit-claim", "content"])
        row = self.w.records("attempts")[0][1]
        self.assertEqual(row["status"], "rejected")
        self.assertEqual(workflow.public_receipt(row)["code"], 11)
        self.assertNotIn("height", row)
        self.assertNotIn("receipt", row)

    def test_contradiction_uses_counter_claim_event_identity(self):
        self.observation["events"] = [{"type": "zerone.knowledge.submit_contradiction", "attributes": [
            {"key": "counter_claim_id", "value": "counter-1"}, {"key": "claim_id", "value": "unrelated"}]}]
        result = self.w.send("challenge", "challenger", ["knowledge", "submit-contradiction", "fact-1"])
        self.assertEqual(result["claim_id"], "counter-1")

    def test_review_saved_before_send_and_cannot_be_replaced(self):
        path, review = self.save_review()
        before = path.read_bytes()
        with self.assertRaisesRegex(workflow.WorkflowError, "cannot be replaced"):
            self.w.review(self.review_args(vote="reject"))
        self.assertEqual(path.read_bytes(), before)
        self.assertEqual(review["evidence"], ["ref:b", "ref:a"])
        self.assertEqual(path.stat().st_mode & 0o777, 0o600)

    def test_reveal_uses_exact_saved_tuple_and_evidence_order(self):
        path, review = self.save_review()
        before = path.read_bytes()
        self.w.round_for_claim.return_value = self.row("VERIFICATION_PHASE_REVEAL")
        self.w.send.side_effect = None
        self.w.send.return_value = {"status": "committed", "action": "reveal"}
        self.w.review(argparse.Namespace(claim="claim-1", actor="reviewer1", wait_for_phase=False), reveal=True)
        arguments = self.w.send.call_args.args[2]
        self.assertEqual(arguments[2:5], [review["round_id"], review["vote"], review["salt"]])
        self.assertEqual(arguments[arguments.index("--review-reason") + 1], review["reason"])
        self.assertEqual(arguments[-4:], ["--review-evidence", "ref:b", "--review-evidence", "ref:a"])
        self.assertEqual(path.read_bytes(), before)

    def test_reveal_rejects_actor_tuple_cross_use(self):
        _, review = self.save_review()
        review["actor"] = "reviewer2"
        path = self.w.review_path("claim-1", "reviewer1")
        workflow.save_json(path, review)
        with self.assertRaisesRegex(workflow.WorkflowError, "identity mismatch"):
            self.w.review(self.review_args(), reveal=True)

    def test_reveal_refuses_wrong_phase_without_send(self):
        self.save_review()
        self.w.send.reset_mock()
        with self.assertRaisesRegex(workflow.WorkflowError, "was not sent"):
            self.w.review(self.review_args(), reveal=True)
        self.w.send.assert_not_called()

    def test_reveal_parser_has_no_preimage_override(self):
        with mock.patch("sys.stderr"), self.assertRaises(SystemExit):
            workflow.parser().parse_args(["reveal", "--home", str(self.home), "--claim", "claim-1", "--actor", "reviewer1", "--vote", "reject"])

    def test_snapshot_never_exposes_preimage_or_private_receipts(self):
        _, review = self.save_review()
        self.attempt(status="committed")
        self.w.history = mock.Mock(side_effect=lambda claim, height: {"chain_id": self.w.chain, "block_height": str(height), "record": {"claim_id": claim}})
        value = self.w.snapshot()
        raw = json.dumps(value)
        for secret in (review["salt"], "PRIVATE reason", "ref:b", self.encoded):
            self.assertNotIn(secret, raw)
        self.assertEqual(value["pending_reviews"][0]["status"], "awaiting-reveal")
        self.assertEqual(value["transactions"][0]["tx_bytes"], len(self.raw))
        self.w.check_node.return_value = 120
        self.assertEqual(self.w.snapshot()["pending_reviews"][0]["status"], "reveal-missed")

    def test_history_rejects_wrong_context_or_claim(self):
        self.w.query = mock.Mock(return_value={"chain_id": self.w.chain, "block_height": "21", "record": {"claim_id": "claim-1"}})
        with self.assertRaisesRegex(workflow.WorkflowError, "different identity or context"):
            self.w.history("claim-1", 20)

    def test_query_receipt_rejects_hash_height_and_gas_malformation(self):
        self.w.tx_observation = workflow.Workflow.tx_observation.__get__(self.w)
        for field, value in (("txhash", "B" * 64), ("height", "0"), ("gas_used", "not-a-number")):
            bad = dict(self.observation, **{field: value})
            self.w.cli.side_effect = None
            self.w.cli.return_value = json.dumps(bad)
            with self.subTest(field=field), self.assertRaises(workflow.WorkflowError):
                self.w.tx_observation(self.digest)

    def test_pending_reconciliation_records_committed_result(self):
        path, _ = self.attempt()
        self.assertEqual(self.w.reconcile(), [])
        self.assertEqual(workflow.read_json(path)["status"], "committed")
        self.w.cli.assert_not_called()

    def test_duplicate_json_and_cross_chain_inventory_refused(self):
        with self.assertRaisesRegex(workflow.WorkflowError, "Duplicate"):
            workflow.parse_json('{"code":0,"code":1}')
        path, value = self.attempt()
        value["chain_id"] = "zerone-1"
        workflow.save_json(path, value)
        with self.assertRaisesRegex(workflow.WorkflowError, "different schema or chain"):
            self.w.records("attempts")

    def test_symlink_hardlink_and_world_readable_files_refused(self):
        original = self.w.directory / "original.json"
        workflow.save_json(original, {"private": True}, fresh=True)
        link = self.w.directory / "link.json"
        link.symlink_to(original)
        with self.assertRaises(OSError):
            workflow.read_json(link)
        link.unlink()
        os.link(original, link)
        with self.assertRaises(workflow.WorkflowError):
            workflow.read_json(link)
        link.unlink()
        original.chmod(0o644)
        with self.assertRaises(workflow.WorkflowError):
            workflow.read_json(original)

    def test_fifo_is_refused_without_waiting_for_a_writer(self):
        path = self.w.directory / "fifo.json"
        os.mkfifo(path, 0o600)
        # A child timeout makes this regression fail deterministically if the
        # read ever blocks before its regular-file guard.
        script = ("import importlib.util; from pathlib import Path; "
                  f"s=importlib.util.spec_from_file_location('w',{str(Path(workflow.__file__).resolve())!r}); "
                  "m=importlib.util.module_from_spec(s); s.loader.exec_module(m); "
                  f"m.read_json(Path({str(path)!r}))")
        result = subprocess.run([os.sys.executable, "-I", "-B", "-c", script], capture_output=True, timeout=3)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(b"regular files", result.stderr)

    def test_repeating_resolved_submit_returns_prior_receipt_without_new_signature(self):
        context = {"content": "exact contribution", "domain": "physics", "category": "empirical", "method": "", "reasoning": "why", "review_fee_uzrn": "100000"}
        self.attempt(status="pending", action="submit", actor="user", context=context)
        changed_fee = {**context, "review_fee_uzrn": "200000"}
        result = self.w.send("submit", "user", ["knowledge", "submit-claim", "exact contribution"], changed_fee)
        self.assertEqual(result["status"], "committed")
        self.assertEqual(result["claim_id"], "claim-1")
        self.assertEqual(len(self.w.records("attempts")), 1)
        self.w.cli.assert_not_called()

    def test_existing_fresh_write_cannot_replace_tuple(self):
        path = self.w.directory / "record.json"
        workflow.save_json(path, {"version": 1}, fresh=True)
        with self.assertRaises(FileExistsError):
            workflow.save_json(path, {"version": 2}, fresh=True)
        self.assertEqual(workflow.read_json(path), {"version": 1})

    def test_separate_workflow_lock_refuses_concurrent_command(self):
        with self.w.locked(), self.assertRaisesRegex(workflow.WorkflowError, "Another workflow"):
            with self.w.locked():
                self.fail("concurrent command entered")

    def test_bounds_refuse_before_review_tuple_is_created(self):
        self.w.round_for_claim = mock.Mock(return_value=self.row())
        for values in ({"reason": " "}, {"confidence": 1000001}, {"evidence": ["duplicate", "duplicate"]}, {"reason": "x" * 4097}):
            with self.subTest(values=list(values)), self.assertRaises(workflow.WorkflowError):
                self.w.review(self.review_args(**values))
            self.assertFalse(self.w.review_path("claim-1", "reviewer1").exists())

    def test_local_profile_and_account_identity_required(self):
        manifest = {"knowledge_profile": None}
        with mock.patch.object(workflow.local, "home_path", return_value=self.home), mock.patch.object(workflow.local, "load_manifest", return_value=manifest):
            with self.assertRaisesRegex(workflow.WorkflowError, "claims-v1"):
                workflow.Workflow(self.home)
        self.w.actor = workflow.Workflow.actor.__get__(self.w)
        self.w.cli.return_value = "wrong-address"
        self.w.cli.side_effect = None
        with self.assertRaisesRegex(workflow.WorkflowError, "differs"):
            self.w.actor("reviewer1")

    def test_sdk_claim_enum_bridge_preserves_record_text_and_refuses_unknown(self):
        message = {"@type": "/zerone.knowledge.v1.MsgSubmitClaim", "claim_type": "CLAIM_TYPE_ASSERTION",
                   "fact_content": "CLAIM_TYPE_ASSERTION stays literal here", "method_id": "method", "reasoning_trace": "Exact reason"}
        value = {"body": {"messages": [message]}}
        self.assertEqual(workflow.signing_json(value)["body"]["messages"][0]["claim_type"], 1)
        self.assertEqual(message["fact_content"], "CLAIM_TYPE_ASSERTION stays literal here")
        self.assertEqual(message["reasoning_trace"], "Exact reason")
        message["claim_type"] = "CLAIM_TYPE_UNKNOWN"
        with self.assertRaisesRegex(workflow.WorkflowError, "Unexpected ordinary"):
            workflow.signing_json(value)


if __name__ == "__main__":
    unittest.main()
