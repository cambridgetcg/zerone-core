#!/usr/bin/env python3
"""Synthetic fee-quote persistence tests; no native signing or network traffic."""
import argparse
import base64
import hashlib
import importlib.util
from pathlib import Path
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location("quote_fixture", Path(__file__).with_name("test_claim_workflow.py"))
assert SPEC is not None and SPEC.loader is not None
fixture = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(fixture)
workflow = fixture.workflow


class FeeQuoteTests(unittest.TestCase):
    # Reuse the existing real persistence/send implementation with its explicitly
    # synthetic CLI boundary, without inheriting and recollecting unrelated tests.
    def setUp(self):
        case = fixture.WorkflowTests()
        case.setUp()
        self.addCleanup(case.doCleanups)
        self.case = case
        self.w, self.calls, self.observation = case.w, case.calls, case.observation
        self.home, self.raw, self.encoded, self.digest = case.home, case.raw, case.encoded, case.digest

    def submit_args(self):
        return argparse.Namespace(content="Synthetic fee-boundary assertion.", domain="physics",
                                  category="empirical", method="M-COMPUTATIONAL",
                                  reasoning="Check the fee quote and later refusal, not scientific truth.")

    def test_failed_execution_retains_original_quote_before_broadcast(self):
        self.w.committed_height.return_value = 99
        self.w.query = mock.Mock(side_effect=[
            {"params": {"min_review_fee": "100000"}},
            {"creation_multiplier_bps": "1000000"},
        ])
        self.observation.update(height="101", code=1, events=[],
                                raw_log="review fee 100000uzrn below minimum 133333uzrn")
        expected = {"height": "99", "base_fee_uzrn": "100000",
                    "creation_multiplier_bps": "1000000", "effective_multiplier_bps": "1000000",
                    "offered_fee_uzrn": "100000"}
        original_cli = self.w.cli.side_effect
        observed_before_broadcast = []

        def cli(*args, **kwargs):
            if args[:2] == ("tx", "broadcast"):
                attempts = self.w.records("attempts")
                self.assertEqual(len(attempts), 1)
                self.assertEqual(attempts[0][1].get("review_fee_quote"), expected)
                observed_before_broadcast.append(True)
            return original_cli(*args, **kwargs)

        self.w.cli.side_effect = cli
        with self.assertRaisesRegex(workflow.WorkflowError, "ended failed"):
            self.w.submit(self.submit_args())
        self.assertEqual(observed_before_broadcast, [True])
        attempts = self.w.records("attempts")
        self.assertEqual(len(attempts), 1)
        attempt = attempts[0][1]
        self.assertEqual(attempt["review_fee_quote"], expected)
        self.assertEqual(attempt["status"], "failed")
        self.assertEqual(attempt["receipt"]["height"], "101")
        self.assertEqual(attempt["context"]["review_fee_uzrn"], "100000")
        self.assertEqual(self.w.query.call_args_list, [
            mock.call("knowledge", "params", height=99),
            mock.call("alignment", "global-pacing", height=99),
        ])
        self.assertEqual(sum(call[:2] == ("tx", "sign") for call in self.calls), 1)
        self.assertEqual(sum(call[:2] == ("tx", "broadcast") for call in self.calls), 1)


    def test_newly_reconciled_unresolved_commit_blocks_signing(self):
        path, _ = fixture.WorkflowTests.attempt(self, status="pending", action="submit", actor="user")
        self.observation["events"] = []
        with self.assertRaises(workflow.WorkflowError) as error:
            self.w.send("submit", "user", ["knowledge", "submit-claim", "different content"])
        self.assertEqual(self.calls, [], "Unresolved commitment must block generation/signing, not just broadcast.")
        self.assertIn("earlier transaction", str(error.exception))
        self.assertEqual(workflow.read_json(path)["status"], "committed-unresolved")
        self.assertEqual(len(self.w.records("attempts")), 1)


    def quote_at(self, height=99, pacing="1000000"):
        self.w.committed_height.return_value = height
        values = {("knowledge", "params"): {"params": {"min_review_fee": "100000"}},
                  ("alignment", "global-pacing"): {} if pacing is None else {"creation_multiplier_bps": pacing}}
        self.w.query = mock.Mock(side_effect=lambda *args, **kwargs: values[args])

    def failed_submission(self):
        self.quote_at()
        self.observation.update(code=1, events=[], raw_log="synthetic fee refusal")
        with self.assertRaisesRegex(workflow.WorkflowError, "ended failed"):
            self.w.submit(self.submit_args())
        return self.w.records("attempts")[0]

    def test_failed_submit_repetition_cannot_create_another_signature(self):
        path, original = self.failed_submission()
        before = path.read_bytes()
        self.quote_at(height=102, pacing="750000")
        self.calls.clear()
        with self.assertRaises(workflow.WorkflowError) as error:
            self.w.submit(self.submit_args())
        self.assertEqual(self.calls, [], "Repeating terminal failure is not approval for another transaction.")
        self.assertIn("resubmit-after", str(error.exception))
        self.assertEqual(path.read_bytes(), before)
        self.assertEqual(len(self.w.records("attempts")), 1)


    def fresh_transaction(self, label="replacement", success=True):
        self.raw = ("synthetic unsigned-by-any-key " + label).encode()
        self.encoded = base64.b64encode(self.raw).decode()
        self.digest = hashlib.sha256(self.raw).hexdigest().upper()
        self.case.raw, self.case.encoded, self.case.digest = self.raw, self.encoded, self.digest
        self.observation.update(txhash=self.digest, height="104", code=0 if success else 1,
                                events=[{"type": "zerone.knowledge.submit_claim",
                                         "attributes": [{"key": "claim_id", "value": "claim-2"}]}] if success else [])

    def approved_args(self, digest, cap="133333"):
        args = self.submit_args()
        args.resubmit_after = digest
        args.max_review_fee_uzrn = cap
        return args

    def test_explicit_bounded_replacement_reserves_before_signing_and_keeps_both_quotes(self):
        parent_path, parent = self.failed_submission()
        self.fresh_transaction()
        self.quote_at(height=103, pacing="750000")
        original_cli = self.w.cli.side_effect
        reservations = []

        def cli(*args, **kwargs):
            if args[:2] == ("tx", "sign"):
                reservation = workflow.read_json(parent_path)["replacement"]
                self.assertEqual(reservation["attempt_id"], args[2].name.removesuffix("-unsigned.json"))
                self.assertEqual(reservation["max_review_fee_uzrn"], "200000")
                reservations.append(reservation)
            return original_cli(*args, **kwargs)

        self.w.cli.side_effect = cli
        self.calls.clear()
        try:
            result = self.w.submit(self.approved_args(parent["txhash"], cap="200000"))
        except workflow.WorkflowError as error:
            self.fail(f"Explicit bounded replacement was refused: {error}")
        self.assertEqual(result["status"], "committed")
        self.assertEqual(len(reservations), 1)
        self.assertEqual(sum(call[:2] == ("tx", "sign") for call in self.calls), 1)
        self.assertEqual(sum(call[:2] == ("tx", "broadcast") for call in self.calls), 1)
        generated = next(call for call in self.calls if "--generate-only" in call)
        self.assertEqual(generated[6], "133333", "Send the calculated minimum, not the approved cap.")
        child = next(row for _, row in self.w.records("attempts") if row["id"] == result["id"])
        self.assertEqual(child["resubmission_of"], parent["txhash"])
        self.assertEqual(child["review_fee_quote"]["height"], "103")
        self.assertEqual(child["review_fee_quote"]["offered_fee_uzrn"], "133333")
        retained_parent = workflow.read_json(parent_path)
        self.assertEqual({key: value for key, value in retained_parent.items() if key != "replacement"}, parent)
        self.assertNotIn("review_fee_quote", result)
        self.assertNotIn("resubmission_of", result)


    def test_replacement_requires_intact_terminal_receipt_and_signer(self):
        mutations = {
            "missing receipt": lambda row: row.pop("receipt"),
            "receipt hash": lambda row: row["receipt"].update(txhash="A" * 64),
            "receipt code": lambda row: row["receipt"].update(code=0),
            "receipt height": lambda row: row.update(height="999"),
            "signer": lambda row: row.update(address="zrn1" + "9" * 38),
            "saved bytes": lambda row: row.update(tx_bytes_base64=base64.b64encode(b"different").decode()),
        }
        for name, mutate in mutations.items():
            with self.subTest(name=name):
                case = FeeQuoteTests()
                case.setUp()
                self.addCleanup(case.doCleanups)
                path, parent = case.failed_submission()
                mutate(parent)
                workflow.save_json(path, parent)
                before = path.read_bytes()
                case.fresh_transaction()
                case.quote_at(pacing="750000")
                case.calls.clear()
                with self.assertRaises(workflow.WorkflowError):
                    case.w.submit(case.approved_args(parent["txhash"]))
                self.assertEqual(case.calls, [])
                self.assertEqual(path.read_bytes(), before)


    def test_quote_must_match_both_context_and_transaction_arguments(self):
        for mismatch in ("context", "arguments"):
            with self.subTest(mismatch=mismatch):
                case = FeeQuoteTests()
                case.setUp()
                self.addCleanup(case.doCleanups)
                case.quote_at()
                args = case.submit_args()
                context = {**vars(args), "review_fee_uzrn": "100000"}
                arguments = ["knowledge", "submit-claim", args.content, args.domain, args.category, "100000"]
                quote = {"height": "99", "base_fee_uzrn": "100000", "creation_multiplier_bps": "1000000",
                         "effective_multiplier_bps": "1000000", "offered_fee_uzrn": "100000"}
                if mismatch == "context":
                    context["review_fee_uzrn"] = "200000"
                else:
                    arguments[5] = "200000"
                with self.assertRaises(workflow.WorkflowError):
                    case.w.send("submit", "user", arguments, context, fee_quote=quote)
                self.assertEqual(case.calls, [])
                self.assertEqual(case.w.records("attempts"), [])


    def test_crash_after_reservation_blocks_even_a_different_new_intent(self):
        path, parent = self.failed_submission()
        self.fresh_transaction()
        self.quote_at(pacing="750000")
        original_cli = self.w.cli.side_effect

        def crash(*args, **kwargs):
            if "--generate-only" in args:
                raise workflow.WorkflowError("synthetic crash after durable reservation")
            return original_cli(*args, **kwargs)

        self.w.cli.side_effect = crash
        with self.assertRaisesRegex(workflow.WorkflowError, "synthetic crash"):
            self.w.submit(self.approved_args(parent["txhash"]))
        self.assertIn("replacement", workflow.read_json(path))
        self.assertEqual(len(self.w.records("attempts")), 1)
        self.w.cli.side_effect = original_cli
        self.calls.clear()
        unrelated = self.submit_args()
        unrelated.content = "A different intent must not conceal an orphan reservation."
        with self.assertRaises(workflow.WorkflowError) as error:
            self.w.submit(unrelated)
        self.assertIn("reserved", str(error.exception).lower())
        self.assertEqual(self.calls, [])
        self.assertEqual(len(self.w.records("attempts")), 1)


    def test_replacement_approval_is_submit_only(self):
        with self.assertRaises(workflow.WorkflowError):
            self.w.send("onboard", "user", ["zerone_auth", "onboard", "agent"],
                        resubmit_after="A" * 64, max_review_fee_uzrn="100000")
        self.assertEqual(self.calls, [])

    def test_both_submit_parsers_offer_explicit_bounded_replacement(self):
        shared = workflow.sibling("shared-claims")
        for parser in (workflow.parser(), shared.parser()):
            with self.subTest(parser=parser.description):
                try:
                    with mock.patch("sys.stderr"):
                        args = parser.parse_args(["submit", "--home", str(self.home), "--content", "example",
                                                  "--reasoning", "synthetic", "--resubmit-after", "A" * 64,
                                                  "--max-review-fee-uzrn", "133333"])
                except SystemExit as error:
                    self.fail(f"Bounded submit approval is missing from its parser: {error}")
                self.assertEqual(args.resubmit_after, "A" * 64)
                self.assertEqual(args.max_review_fee_uzrn, "133333")


    def test_invalid_or_unbound_approval_never_generates_a_transaction(self):
        cases = ("missing hash", "missing cap", "wrong hash", "lowercase hash", "short hash", "zero cap",
                 "noncanonical cap", "boolean cap", "over budget", "changed intent", "duplicate predecessor", "wrong chain")
        for name in cases:
            with self.subTest(name=name):
                case = FeeQuoteTests()
                case.setUp()
                self.addCleanup(case.doCleanups)
                path, parent = case.failed_submission()
                case.fresh_transaction()
                case.quote_at(pacing="750000")
                args = case.approved_args(parent["txhash"])
                if name == "missing hash": args.resubmit_after = None
                elif name == "missing cap": args.max_review_fee_uzrn = None
                elif name == "wrong hash": args.resubmit_after = "A" * 64
                elif name == "lowercase hash": args.resubmit_after = args.resubmit_after.lower()
                elif name == "short hash": args.resubmit_after = args.resubmit_after[:-1]
                elif name == "zero cap": args.max_review_fee_uzrn = "0"
                elif name == "noncanonical cap": args.max_review_fee_uzrn = "0133333"
                elif name == "boolean cap": args.max_review_fee_uzrn = True
                elif name == "over budget": args.max_review_fee_uzrn = "133332"
                elif name == "changed intent": args.reasoning = "This is a different contribution."
                elif name == "wrong chain":
                    parent["chain_id"] = "different-chain"
                    workflow.save_json(path, parent)
                elif name == "duplicate predecessor":
                    workflow.save_json(path.with_name("b" * 32 + ".json"), {**parent, "id": "b" * 32}, fresh=True)
                before = path.read_bytes()
                case.calls.clear()
                with self.assertRaises(workflow.WorkflowError):
                    case.w.submit(args)
                self.assertEqual(case.calls, [])
                self.assertEqual(path.read_bytes(), before)

    def test_fee_formula_and_raw_zero_or_absence_are_preserved(self):
        for pacing, offered, effective in ((None, "100000", "1000000"), ("0", "100000", "1000000"),
                                          (0, "100000", "1000000"), ("1000000", "100000", "1000000"),
                                          ("750000", "133333", "750000"), ("1000001", "100000", "1000001")):
            with self.subTest(pacing=pacing):
                case = FeeQuoteTests()
                case.setUp()
                self.addCleanup(case.doCleanups)
                case.quote_at(pacing=pacing)
                receipt = case.w.submit(case.submit_args())
                quote = case.w.records("attempts")[0][1]["review_fee_quote"]
                self.assertEqual(quote["creation_multiplier_bps"], pacing)
                self.assertEqual(quote["effective_multiplier_bps"], effective)
                self.assertEqual(quote["offered_fee_uzrn"], offered)
                self.assertEqual(receipt["review_fee_uzrn"], offered)
                self.assertNotIn("review_fee_quote", receipt)
                generated = next(call for call in case.calls if "--generate-only" in call)
                self.assertEqual(generated[6], offered)
                self.assertEqual(generated[generated.index("--fees") + 1], "2000000uzrn")

    def test_malformed_quote_inputs_refuse_before_signing(self):
        for pacing in ("-1", "01", "1.0", True, "garbage"):
            with self.subTest(pacing=pacing):
                self.quote_at(pacing=pacing)
                with self.assertRaises(workflow.WorkflowError):
                    self.w.submit(self.submit_args())
                self.assertEqual(self.calls, [])
                self.assertEqual(self.w.records("attempts"), [])

    def test_failed_child_requires_its_own_new_approval(self):
        _, parent = self.failed_submission()
        self.fresh_transaction(success=False)
        self.quote_at(pacing="750000")
        approval = self.approved_args(parent["txhash"])
        with self.assertRaisesRegex(workflow.WorkflowError, "ended failed"):
            self.w.submit(approval)
        child = next(row for _, row in self.w.records("attempts") if row["txhash"] == self.digest)
        self.calls.clear()
        with self.assertRaisesRegex(workflow.WorkflowError, "reserved"):
            self.w.submit(approval)
        self.assertEqual(self.calls, [])
        self.fresh_transaction(label="second replacement")
        result = self.w.submit(self.approved_args(child["txhash"]))
        self.assertEqual(result["status"], "committed")
        self.assertEqual(sum(call[:2] == ("tx", "sign") for call in self.calls), 1)
        self.assertEqual(len(self.w.records("attempts")), 3)

    def test_recovering_committed_child_ignores_new_fee_above_cap(self):
        _, parent = self.failed_submission()
        self.fresh_transaction()
        self.quote_at(pacing="750000")
        approval = self.approved_args(parent["txhash"])
        committed = self.w.submit(approval)
        before = {path.name: path.read_bytes() for path, _ in self.w.records("attempts")}
        self.quote_at(height=111, pacing="500000")
        self.calls.clear()
        self.assertEqual(self.w.submit(approval), committed)
        self.assertEqual(self.calls, [])
        self.assertEqual({path.name: path.read_bytes() for path, _ in self.w.records("attempts")}, before)

    def test_crash_boundaries_never_reuse_parent_approval(self):
        for point in ("after signing", "after durable attempt", "after broadcast"):
            with self.subTest(point=point):
                case = FeeQuoteTests()
                case.setUp()
                self.addCleanup(case.doCleanups)
                path, parent = case.failed_submission()
                case.fresh_transaction()
                case.quote_at(pacing="750000")
                case.w.tx_observation.return_value = None
                original_cli, original_broadcast = case.w.cli.side_effect, case.w.broadcast

                def cli(*args, **kwargs):
                    if point == "after signing" and args[:2] == ("tx", "encode"):
                        raise workflow.WorkflowError("synthetic post-sign interruption")
                    if point == "after broadcast" and args[:2] == ("tx", "broadcast"):
                        original_cli(*args, **kwargs)
                        raise fixture.subprocess.TimeoutExpired("synthetic broadcast", 60)
                    return original_cli(*args, **kwargs)

                case.w.cli.side_effect = cli
                if point == "after durable attempt":
                    case.w.broadcast = mock.Mock(side_effect=workflow.WorkflowError("synthetic post-persist interruption"))
                approval = case.approved_args(parent["txhash"])
                with self.assertRaises(workflow.WorkflowError):
                    case.w.submit(approval)
                self.assertIn("replacement", workflow.read_json(path))
                saved = {p.name: p.read_bytes() for p, _ in case.w.records("attempts")}
                case.w.cli.side_effect, case.w.broadcast = original_cli, original_broadcast
                case.calls.clear()
                with self.assertRaises(workflow.WorkflowError):
                    case.w.submit(approval)
                self.assertEqual(case.calls, [])
                self.assertEqual({p.name: p.read_bytes() for p, _ in case.w.records("attempts")}, saved)
                if point != "after signing":
                    case.w.tx_observation.return_value = case.observation
                    recovered = case.w.retry(case.digest)
                    self.assertEqual(recovered["status"], "committed")
                    self.assertEqual(case.calls, [])

    def test_shared_actor_normalization_forwards_approval(self):
        shared = workflow.sibling("shared-claims")
        receiver = object.__new__(shared.SharedWorkflow)
        quote = {"height": "12"}
        with mock.patch.object(shared.core.Workflow, "send", return_value={"status": "committed"}) as sender:
            receiver.send("submit", "user", ["arguments"], {"context": "unchanged"}, fee_quote=quote,
                          resubmit_after="A" * 64, max_review_fee_uzrn="133333")
        sender.assert_called_once_with("submit", shared.ACTOR, ["arguments"], {"context": "unchanged"},
                                       fee_quote=quote, resubmit_after="A" * 64, max_review_fee_uzrn="133333")


    def test_replacement_arguments_cannot_change_the_approved_intent(self):
        _, parent = self.failed_submission()
        self.fresh_transaction()
        self.quote_at(pacing="750000")
        context = {**parent["context"], "review_fee_uzrn": "133333"}
        quote = {"height": "99", "base_fee_uzrn": "100000", "creation_multiplier_bps": "750000",
                 "effective_multiplier_bps": "750000", "offered_fee_uzrn": "133333"}
        arguments = ["knowledge", "submit-claim", "Different actual signed content", context["domain"],
                     context["category"], "133333", "--method-id", context["method"], "--reasoning-trace", context["reasoning"]]
        self.calls.clear()
        with self.assertRaises(workflow.WorkflowError):
            self.w.send("submit", "user", arguments, context, fee_quote=quote,
                        resubmit_after=parent["txhash"], max_review_fee_uzrn="133333")
        self.assertEqual(self.calls, [])


    def test_rejected_first_delivery_requires_approval_but_keeps_exact_byte_retry(self):
        self.quote_at()
        original_cli = self.w.cli.side_effect

        def refuse(*args, **kwargs):
            if args[:2] == ("tx", "broadcast"):
                return fixture.json.dumps({"txhash": self.digest, "code": 11})
            return original_cli(*args, **kwargs)

        self.w.cli.side_effect = refuse
        with self.assertRaisesRegex(workflow.WorkflowError, "CheckTx rejected"):
            self.w.submit(self.submit_args())
        path, parent = self.w.records("attempts")[0]
        before = path.read_bytes()
        self.w.cli.side_effect = original_cli
        self.calls.clear()
        with self.assertRaisesRegex(workflow.WorkflowError, "resubmit-after"):
            self.w.submit(self.submit_args())
        self.assertEqual(self.calls, [])
        self.assertEqual(path.read_bytes(), before)
        self.w.query = mock.Mock(side_effect=AssertionError("Retry must not query fees."))
        self.w.tx_observation.side_effect = [None, self.observation]
        result = self.w.retry(parent["txhash"])
        self.assertEqual(result["status"], "committed")
        self.assertFalse(any(call[:2] == ("tx", "sign") for call in self.calls))
        self.assertEqual(workflow.read_json(path)["review_fee_quote"], parent["review_fee_quote"])
        self.assertEqual(workflow.read_json(path)["tx_bytes_base64"], parent["tx_bytes_base64"])

    def test_legacy_failure_does_not_invent_original_quote(self):
        path, parent = self.failed_submission()
        parent.pop("review_fee_quote")
        workflow.save_json(path, parent)
        self.fresh_transaction()
        self.quote_at(pacing="750000")
        result = self.w.submit(self.approved_args(parent["txhash"]))
        self.assertEqual(result["status"], "committed")
        self.assertNotIn("review_fee_quote", workflow.read_json(path))
        child = next(row for _, row in self.w.records("attempts") if row["id"] == result["id"])
        self.assertEqual(child["review_fee_quote"]["offered_fee_uzrn"], "133333")

    def test_real_shared_workflow_replacement_uses_normalized_participant_identity(self):
        shared = workflow.sibling("shared-claims")
        receiver = object.__new__(shared.SharedWorkflow)
        receiver.__dict__.update(self.w.__dict__)
        receiver.chain = shared.CHAIN
        receiver.accounts = {shared.ACTOR: self.w.accounts["user"]}
        receiver.manifest = {"descriptor_sha256": "d" * 64, "genesis_sha256": "e" * 64}
        self.w = self.case.w = receiver
        self.quote_at()
        self.observation.update(code=1, events=[])
        with self.assertRaisesRegex(shared.core.WorkflowError, "ended failed"):
            receiver.submit(self.submit_args())
        parent = receiver.records("attempts")[0][1]
        self.assertEqual(parent["actor"], shared.ACTOR)
        self.fresh_transaction()
        self.quote_at(pacing="750000")
        self.calls.clear()
        result = receiver.submit(self.approved_args(parent["txhash"]))
        self.assertEqual(result["actor"], shared.ACTOR)
        self.assertEqual(result["status"], "committed")
        self.assertEqual(sum(call[:2] == ("tx", "sign") for call in self.calls), 1)
        for _, row in receiver.records("attempts"):
            self.assertEqual(row["participant_address"], receiver.accounts[shared.ACTOR])
            self.assertEqual(row["descriptor_sha256"], receiver.manifest["descriptor_sha256"])


    def test_linked_replacement_cannot_hide_another_actor_or_intent(self):
        for field in ("actor", "address", "action", "context"):
            with self.subTest(field=field):
                case = FeeQuoteTests()
                case.setUp()
                self.addCleanup(case.doCleanups)
                _, parent = case.failed_submission()
                case.fresh_transaction()
                case.quote_at(pacing="750000")
                result = case.w.submit(case.approved_args(parent["txhash"]))
                path, child = next((path, row) for path, row in case.w.records("attempts") if row["id"] == result["id"])
                child[field] = {**child["context"], "content": "conflicting intent"} if field == "context" else "conflicting identity"
                workflow.save_json(path, child)
                case.calls.clear()
                with self.assertRaises(workflow.WorkflowError):
                    case.w.reconcile()
                self.assertEqual(case.calls, [])

    def test_nonstring_terminal_receipt_hash_fails_closed(self):
        path, parent = self.failed_submission()
        parent["receipt"]["txhash"] = None
        workflow.save_json(path, parent)
        self.fresh_transaction()
        self.quote_at(pacing="750000")
        self.calls.clear()
        try:
            self.w.submit(self.approved_args(parent["txhash"]))
        except Exception as error:
            self.assertIsInstance(error, workflow.WorkflowError)
        else:
            self.fail("Malformed terminal receipt was accepted.")
        self.assertEqual(self.calls, [])


class IdenticalReplacementTests(unittest.TestCase):
    """FQ-1: closed synthetic CLI bytes, real locking/persistence/send/retry."""

    def setUp(self):
        self.c = FeeQuoteTests()
        self.c.setUp()
        self.addCleanup(self.c.doCleanups)
        for target in ("subprocess.Popen", "os.system", "socket.socket", "socket.create_connection"):
            guard = mock.patch(target, side_effect=AssertionError("No real process/network in synthetic FQ-1 tests."))
            guard.start()
            self.addCleanup(guard.stop)

    def rejected_parent(self):
        c = self.c
        c.quote_at(height=99)
        original_cli = c.w.cli.side_effect

        def reject(*args, **kwargs):
            if args[:2] == ("tx", "broadcast"):
                c.calls.append(args)
                return fixture.json.dumps({"txhash": c.digest, "code": 11})
            return original_cli(*args, **kwargs)

        with c.w.locked(), mock.patch.object(c.w, "cli", side_effect=reject):
            with self.assertRaisesRegex(workflow.WorkflowError, "CheckTx rejected"):
                c.w.submit(c.submit_args())
        c.quote_at(height=100)  # Height changed; fee, account sequence and synthetic bytes did not.
        return c.w.records("attempts")[0]

    def test_same_hash_prebroadcast_boundary_keeps_original_uniquely_retryable(self):
        c = self.c
        _, parent = self.rejected_parent()
        with c.w.locked(), mock.patch.object(c.w, "broadcast", side_effect=workflow.WorkflowError("synthetic pre-broadcast crash")):
            with self.assertRaises(workflow.WorkflowError):
                c.w.submit(c.approved_args(parent["txhash"], cap="100000"))
        c.calls.clear()
        c.w.query = mock.Mock(side_effect=AssertionError("Retry cannot quote."))
        c.w.tx_observation.side_effect = [None, c.observation]
        try:
            with c.w.locked():
                receipt = c.w.retry(parent["txhash"])
        except workflow.WorkflowError as error:
            self.fail(f"Original hash must remain uniquely retryable: {error}")
        self.assertEqual(receipt["status"], "committed")
        self.assertEqual([call[:2] for call in c.calls], [("tx", "encode"), ("tx", "broadcast")])

    def completed_identical(self):
        path, parent = self.rejected_parent()
        with self.c.w.locked(), self.assertRaisesRegex(workflow.WorkflowError, "identical"):
            self.c.w.submit(self.c.approved_args(parent["txhash"], cap="100000"))
        return path, parent

    def test_completion_does_not_hide_invalid_signer_or_terminal_receipt(self):
        mutations = {
            "actor": lambda row: row.update(actor="reviewer1"),
            "address": lambda row: row.update(address="zrn1" + "9" * 38),
            "missing receipt": lambda row: row.pop("check_tx"),
            "receipt hash": lambda row: row["check_tx"].update(txhash="A" * 64),
            "receipt code": lambda row: row["check_tx"].update(code=0),
            "parent code": lambda row: row.update(code=12),
        }
        path, _ = self.completed_identical()
        original = workflow.read_json(path)
        for name, mutate in mutations.items():
            with self.subTest(mutation=name):
                altered = fixture.json.loads(fixture.json.dumps(original))
                mutate(altered)
                workflow.save_json(path, altered)
                self.c.calls.clear()
                with self.c.w.locked(), self.assertRaises(workflow.WorkflowError):
                    self.c.w.reconcile()
                self.assertEqual(self.c.calls, [])
        workflow.save_json(path, original)

    def test_identical_replacement_refuses_without_duplicate_and_keeps_exact_retry(self):
        c = self.c
        path, parent = self.rejected_parent()
        public_before = workflow.public_receipt(parent)
        original_signed = c.w.directory / (parent["id"] + "-signed.json")
        signed_before = original_signed.read_bytes()
        approval = c.approved_args(parent["txhash"], cap="100000")
        # On the failed candidate this interrupts AFTER a duplicate child is saved.
        # On the fix, identical bytes must refuse BEFORE reaching this boundary.
        with c.w.locked(), mock.patch.object(c.w, "broadcast", side_effect=workflow.WorkflowError("synthetic pre-broadcast crash")) as broadcast:
            with self.assertRaises(workflow.WorkflowError) as refused:
                c.w.submit(approval)
        rows = c.w.records("attempts")
        self.assertEqual(len(rows), 1, "Identical replacement must not create a second hash-addressed attempt.")
        self.assertIn("identical", str(refused.exception).lower())
        self.assertIn("retry --txhash " + parent["txhash"], str(refused.exception))
        broadcast.assert_not_called()
        retained = workflow.read_json(path)
        self.assertEqual({k: v for k, v in retained.items() if k != "replacement"}, parent)
        self.assertEqual(retained["replacement"]["outcome"], {"status": "identical-transaction", "txhash": parent["txhash"]})
        self.assertEqual(retained["replacement"]["max_review_fee_uzrn"], "100000")
        self.assertEqual(workflow.public_receipt(retained), public_before)
        self.assertEqual(original_signed.read_bytes(), signed_before)
        generated = [call for call in c.calls if "--generate-only" in call]
        self.assertEqual(len(generated), 2)
        self.assertEqual(generated[0], generated[1], "No hidden fee, nonce, timeout or semantic change.")
        self.assertEqual(sum(call[:2] == ("tx", "sign") for call in c.calls), 2)
        c.calls.clear()
        with c.w.locked():
            self.assertEqual(c.w.reconcile(), [])
            with self.assertRaisesRegex(workflow.WorkflowError, "reserved"):
                c.w.submit(approval)
        self.assertEqual(c.calls, [])
        c.w.query = mock.Mock(side_effect=AssertionError("Exact-byte retry must not refresh the fee quote."))
        c.w.tx_observation.side_effect = [None, c.observation]
        with c.w.locked():
            receipt = c.w.retry(parent["txhash"])
        self.assertEqual(receipt["status"], "committed")
        self.assertEqual([call[:2] for call in c.calls], [("tx", "encode"), ("tx", "broadcast")])
        self.assertTrue(all(call[2] == original_signed for call in c.calls))
        recovered = workflow.read_json(path)
        for field in ("txhash", "tx_bytes_base64", "review_fee_quote", "check_tx"):
            self.assertEqual(recovered[field], parent[field])
        self.assertEqual(len(c.w.records("attempts")), 1)


    def test_completion_write_failures_keep_approval_consumed_and_fail_closed(self):
        for point in ("before completion write", "temporary fsync", "replace", "after durable completion"):
            with self.subTest(point=point):
                probe = IdenticalReplacementTests()
                probe.setUp()
                self.addCleanup(probe.doCleanups)
                c = probe.c
                path, parent = probe.rejected_parent()
                save, fsync, replace = workflow.save_json, workflow.os.fsync, workflow.os.replace
                completing = False

                def save_boundary(target, value, fresh=False):
                    nonlocal completing
                    if target == path and "outcome" in value.get("replacement", {}):
                        completing = True
                        if point == "before completion write":
                            raise OSError("synthetic completion-write failure")
                        save(target, value, fresh=fresh)
                        if point == "after durable completion":
                            raise OSError("synthetic completion-write failure")
                        return
                    return save(target, value, fresh=fresh)

                def fsync_boundary(fd):
                    if completing and point == "temporary fsync":
                        raise OSError("synthetic completion-write failure")
                    return fsync(fd)

                def replace_boundary(source, target):
                    if completing and point == "replace":
                        raise OSError("synthetic completion-write failure")
                    return replace(source, target)

                c.calls.clear()
                approval = c.approved_args(parent["txhash"], cap="100000")
                with mock.patch.object(workflow, "save_json", side_effect=save_boundary), \
                     mock.patch.object(workflow.os, "fsync", side_effect=fsync_boundary), \
                     mock.patch.object(workflow.os, "replace", side_effect=replace_boundary):
                    with c.w.locked(), self.assertRaisesRegex(OSError, "synthetic completion-write failure"):
                        c.w.submit(approval)
                self.assertEqual(sum(call[:2] == ("tx", "sign") for call in c.calls), 1)
                self.assertFalse(any(call[:2] == ("tx", "broadcast") for call in c.calls))
                retained = workflow.read_json(path)
                self.assertEqual({k: v for k, v in retained.items() if k != "replacement"}, parent)
                self.assertEqual("outcome" in retained["replacement"], point == "after durable completion")
                c.calls.clear()
                with c.w.locked(), self.assertRaises(workflow.WorkflowError):
                    c.w.submit(approval)
                self.assertEqual(c.calls, [], "Even failed completion persistence cannot reuse approval.")
                unrelated = c.submit_args()
                unrelated.content = "Unrelated intent after the completion boundary."
                if point == "after durable completion":
                    self.assertEqual(c.w.reconcile(), [])
                    c.fresh_transaction(label="unrelated after durable identical outcome")
                    with c.w.locked():
                        self.assertEqual(c.w.submit(unrelated)["status"], "committed")
                else:
                    with c.w.locked(), self.assertRaises(workflow.WorkflowError):
                        c.w.submit(unrelated)
                    self.assertEqual(c.calls, [])
                    # Atomic-write leftovers remain evidence; no auto-cleanup/resolution.
                    if point in ("temporary fsync", "replace"):
                        self.assertTrue(any(p.name.startswith(".write-") for p in path.parent.iterdir()))

    def test_malformed_completion_cannot_release_an_orphan(self):
        path, _ = self.completed_identical()
        original = workflow.read_json(path)
        mutations = {
            "missing outcome": lambda r: r["replacement"].pop("outcome"),
            "null outcome": lambda r: r["replacement"].update(outcome=None),
            "string outcome": lambda r: r["replacement"].update(outcome="identical-transaction"),
            "wrong status": lambda r: r["replacement"]["outcome"].update(status="committed"),
            "wrong hash": lambda r: r["replacement"]["outcome"].update(txhash="A" * 64),
            "missing hash": lambda r: r["replacement"]["outcome"].pop("txhash"),
            "extra outcome field": lambda r: r["replacement"]["outcome"].update(allow=True),
            "extra reservation field": lambda r: r["replacement"].update(allow=True),
            "invalid reservation id": lambda r: r["replacement"].update(attempt_id="not-an-id"),
            "self reservation": lambda r: r["replacement"].update(attempt_id=r["id"]),
            "missing cap": lambda r: r["replacement"].pop("max_review_fee_uzrn"),
            "zero cap": lambda r: r["replacement"].update(max_review_fee_uzrn="0"),
            "wrong bytes": lambda r: r.update(tx_bytes_base64=base64.b64encode(b"other synthetic bytes").decode()),
            "wrong action": lambda r: r.update(action="onboard"),
            "missing context": lambda r: r.pop("context"),
        }
        for name, mutate in mutations.items():
            with self.subTest(mutation=name):
                altered = fixture.json.loads(fixture.json.dumps(original))
                mutate(altered)
                workflow.save_json(path, altered)
                before = path.read_bytes()
                self.c.calls.clear()
                with self.c.w.locked(), self.assertRaises(workflow.WorkflowError):
                    self.c.w.send("onboard", "user", ["zerone_auth", "onboard", "agent"])
                self.assertEqual(self.c.calls, [])
                self.assertEqual(path.read_bytes(), before)
        workflow.save_json(path, original)

    def test_completion_cannot_conceal_child_path_id_link_or_duplicate_hash(self):
        path, _ = self.completed_identical()
        retained = workflow.read_json(path)
        child_id = retained["replacement"]["attempt_id"]
        for conflict in ("path", "id", "link", "hash"):
            with self.subTest(conflict=conflict):
                other_id = "b" * 32
                child = {**retained, "id": other_id, "txhash": "A" * 64}
                child.pop("replacement")
                child_path = path.with_name(other_id + ".json")
                if conflict == "path": child_path = path.with_name(child_id + ".json")
                elif conflict == "id": child["id"] = child_id
                elif conflict == "link": child["resubmission_of"] = retained["txhash"]
                else: child["txhash"] = retained["txhash"]
                workflow.save_json(child_path, child, fresh=True)
                self.c.calls.clear()
                with self.c.w.locked(), self.assertRaises(workflow.WorkflowError):
                    self.c.w.reconcile()
                if conflict == "hash":
                    with self.c.w.locked(), self.assertRaisesRegex(workflow.WorkflowError, "No unique saved"):
                        self.c.w.retry(retained["txhash"])
                self.assertEqual(self.c.calls, [])
                child_path.unlink()
        self.assertEqual(self.c.w.reconcile(), [])

    def test_distinct_byte_replacement_crash_still_retries_exact_child(self):
        c = self.c
        _, parent = self.rejected_parent()
        c.fresh_transaction(label="synthetic independently advanced sequence")
        with c.w.locked(), mock.patch.object(c.w, "broadcast", side_effect=workflow.WorkflowError("synthetic pre-broadcast crash")):
            with self.assertRaisesRegex(workflow.WorkflowError, "pre-broadcast crash"):
                c.w.submit(c.approved_args(parent["txhash"], cap="100000"))
        rows = c.w.records("attempts")
        self.assertEqual(len(rows), 2)
        child = next(row for _, row in rows if row["txhash"] == c.digest)
        self.assertEqual(child["status"], "prepared")
        self.assertEqual(child["resubmission_of"], parent["txhash"])
        self.assertNotEqual(child["tx_bytes_base64"], parent["tx_bytes_base64"])
        c.calls.clear()
        c.w.query = mock.Mock(side_effect=AssertionError("Retry cannot quote."))
        c.w.tx_observation.side_effect = [None, c.observation]
        with c.w.locked():
            self.assertEqual(c.w.retry(child["txhash"])["status"], "committed")
        self.assertEqual([call[:2] for call in c.calls], [("tx", "encode"), ("tx", "broadcast")])
        self.assertTrue(all(call[2] == c.w.directory / (child["id"] + "-signed.json") for call in c.calls))

    def test_completion_does_not_release_unknown_delivery_and_recovers_commit(self):
        c = self.c
        path, parent = self.completed_identical()
        cli = c.w.cli.side_effect

        def lose_response(*args, **kwargs):
            if args[:2] == ("tx", "broadcast"):
                raise fixture.subprocess.TimeoutExpired("synthetic no-network delivery", 60)
            return cli(*args, **kwargs)

        c.w.tx_observation.return_value = None
        with c.w.locked(), mock.patch.object(c.w, "cli", side_effect=lose_response):
            with self.assertRaisesRegex(workflow.WorkflowError, "unresolved outcome"):
                c.w.retry(parent["txhash"])
        self.assertEqual(workflow.read_json(path)["status"], "broadcasting")
        c.calls.clear()
        unrelated = c.submit_args()
        unrelated.content = "Must stay blocked while original delivery is unknown."
        with c.w.locked(), self.assertRaisesRegex(workflow.WorkflowError, "earlier transaction"):
            c.w.submit(unrelated)
        self.assertEqual(c.calls, [])
        c.w.tx_observation.return_value = c.observation
        with c.w.locked():
            receipt = c.w.retry(parent["txhash"])
        c.quote_at(height=200, pacing="500000")
        with c.w.locked():
            self.assertEqual(c.w.submit(c.approved_args(parent["txhash"], cap="100000")), receipt)
        self.assertEqual(c.calls, [])
        self.assertEqual(workflow.read_json(path)["review_fee_quote"], parent["review_fee_quote"])
        self.assertEqual(c.w.reconcile(), [])

    def test_replacement_cannot_alias_an_unrelated_retained_transaction(self):
        c = self.c
        path, parent = self.rejected_parent()
        c.fresh_transaction(label="synthetic unrelated transaction")
        unrelated = c.submit_args()
        unrelated.content = "A separate contribution."
        with c.w.locked():
            c.w.submit(unrelated)
        c.calls.clear()
        with c.w.locked(), self.assertRaisesRegex(workflow.WorkflowError, "conflicts with a retained transaction hash"):
            c.w.submit(c.approved_args(parent["txhash"], cap="100000"))
        self.assertNotIn("outcome", workflow.read_json(path)["replacement"])
        self.assertEqual(len(c.w.records("attempts")), 2)
        self.assertFalse(any(call[:2] == ("tx", "broadcast") for call in c.calls))
        c.calls.clear()
        with c.w.locked(), self.assertRaisesRegex(workflow.WorkflowError, "reserved replacement"):
            c.w.submit(c.approved_args(parent["txhash"], cap="100000"))
        self.assertEqual(c.calls, [])


class SharedIdenticalReplacementTests(unittest.TestCase):
    """Actual SharedWorkflow constructor, parser, actor and pin checks; fake CLI/HTTP only."""

    def setUp(self):
        for target in ("subprocess.Popen", "os.system", "socket.socket", "socket.create_connection"):
            guard = mock.patch(target, side_effect=AssertionError("No native backend in synthetic shared FQ-1 tests."))
            guard.start()
            self.addCleanup(guard.stop)
        self.fixture = workflow.sibling("test_shared_claims")
        self.shared = self.fixture.shared
        self.core = self.shared.core
        self.case = self.fixture.ParticipantTests()
        self.case.setUp()
        self.addCleanup(self.case.doCleanups)
        self.case.initialize()
        request = mock.patch.object(self.shared, "request", side_effect=self.case.network)
        self.request = request.start()
        self.addCleanup(request.stop)
        self.w = self.case.workflow()
        self.w.wait = 0.02
        self.raw = b"Explicitly synthetic shared identical transaction; no keys or signatures."
        self.encoded = base64.b64encode(self.raw).decode()
        self.digest = hashlib.sha256(self.raw).hexdigest().upper()
        self.calls, self.tx_results = [], []
        self.key_address = self.case.address
        self.check_tx_code = 11
        self.observation = {"txhash": self.digest, "height": "21", "code": 0, "gas_wanted": "2000000", "gas_used": "1",
                            "events": [{"type": "zerone.knowledge.submit_claim", "attributes": [{"key": "claim_id", "value": "synthetic-claim"}]}]}

        def cli(*args, **kwargs):
            self.calls.append(args)
            if args[:2] == ("keys", "show"): return self.key_address
            if args[:3] == ("query", "knowledge", "params"): return fixture.json.dumps({"params": {"min_review_fee": "100000"}})
            if args[:3] == ("query", "alignment", "global-pacing"): return fixture.json.dumps({"creation_multiplier_bps": "1000000"})
            if args[:2] == ("query", "tx"):
                value = self.tx_results.pop(0) if self.tx_results else None
                return None if value is None else fixture.json.dumps(value)
            if args[:2] == ("tx", "encode"): return self.encoded
            if args[:2] == ("tx", "broadcast"): return fixture.json.dumps({"txhash": self.digest, "code": self.check_tx_code})
            if args[:3] == ("tx", "knowledge", "submit-claim") or args[:2] == ("tx", "sign"):
                return fixture.json.dumps({"body": {"messages": [{"@type": "/zerone.knowledge.v1.MsgSubmitClaim", "claim_type": "CLAIM_TYPE_ASSERTION"}]},
                                           "auth_info": {}, "signatures": []})
            raise AssertionError("Unexpected synthetic CLI command: " + str(args[:3]))

        self.w.cli = mock.Mock(side_effect=cli)
        self.argv = ["submit", "--home", str(self.case.home), "--content", "Synthetic shared unchanged-byte contribution.",
                     "--reasoning", "Offline fixture only, no scientific or payment claim."]

    def rejected_parent(self):
        with self.w.locked(), self.assertRaisesRegex(self.core.WorkflowError, "CheckTx rejected"):
            self.w.submit(self.shared.parser().parse_args(self.argv))
        self.check_tx_code = 0
        self.approval = self.shared.parser().parse_args(self.argv + ["--resubmit-after", self.digest, "--max-review-fee-uzrn", "100000"])
        return self.w.records("attempts")[0]

    def test_shared_completion_keeps_original_retry_and_binding(self):
        path, parent = self.rejected_parent()
        with self.w.locked(), mock.patch.object(self.w, "broadcast", side_effect=self.core.WorkflowError("synthetic pre-broadcast crash")) as broadcast:
            with self.assertRaisesRegex(self.core.WorkflowError, "identical") as error:
                self.w.submit(self.approval)
        self.assertIn("shared-claims retry --tx " + self.digest, str(error.exception))
        broadcast.assert_not_called()
        self.assertEqual(len(self.w.records("attempts")), 1)
        retained = self.core.read_json(path)
        self.assertEqual({k: v for k, v in retained.items() if k != "replacement"}, parent)
        self.assertEqual(retained["replacement"]["outcome"], {"status": "identical-transaction", "txhash": self.digest})
        self.assertEqual(retained["actor"], self.shared.ACTOR)
        self.assertEqual(retained["address"], self.case.address)
        self.assertEqual(retained["participant_address"], self.case.address)
        self.assertEqual(retained["descriptor_sha256"], self.case.args.descriptor_sha256)
        generated = [call for call in self.calls if "--generate-only" in call]
        self.assertEqual(len(generated), 2)
        self.assertEqual(generated[0], generated[1])
        self.assertEqual(generated[0][generated[0].index("--from") + 1], self.shared.ACTOR)
        self.calls.clear()
        with self.w.locked(), self.assertRaisesRegex(self.core.WorkflowError, "reserved"):
            self.w.submit(self.approval)
        self.assertFalse(any(call[0] == "tx" for call in self.calls))
        self.calls.clear()
        self.tx_results = [None, self.observation]
        retry = self.shared.parser().parse_args(["retry", "--home", str(self.case.home), "--tx", self.digest])
        with self.w.locked():
            receipt = self.w.retry(retry.tx)
        self.assertEqual(receipt["status"], "committed")
        self.assertFalse(any("--generate-only" in call or call[:2] == ("tx", "sign") for call in self.calls))
        self.assertFalse(any(call[:2] in (("query", "knowledge"), ("query", "alignment")) for call in self.calls))
        tx_calls = [call for call in self.calls if call[0] == "tx"]
        self.assertEqual([call[:2] for call in tx_calls], [("tx", "encode"), ("tx", "broadcast")])
        self.assertTrue(all(call[2] == self.w.directory / (parent["id"] + "-signed.json") for call in tx_calls))
        recovered = self.core.read_json(path)
        for field in ("txhash", "tx_bytes_base64", "review_fee_quote", "check_tx"):
            self.assertEqual(recovered[field], parent[field])
        self.assertEqual(self.w.reconcile(), [])
        self.w.check_files()  # Real source hashes and private home checks, not a mocked method.

    def test_shared_source_pin_and_actor_refusals_survive_completion(self):
        path, parent = self.rejected_parent()
        with self.w.locked(), self.assertRaisesRegex(self.core.WorkflowError, "identical"):
            self.w.submit(self.approval)
        before = path.read_bytes()
        self.calls.clear()
        self.key_address = "zrn1" + "9" * 38
        with self.w.locked(), self.assertRaisesRegex(self.core.WorkflowError, "signing key differs"):
            self.w.retry(parent["txhash"])
        self.assertFalse(any(call[0] == "tx" for call in self.calls))
        self.assertEqual(path.read_bytes(), before)
        self.key_address = self.case.address
        marker_path = self.case.home / self.shared.MARKER
        marker = self.core.read_json(marker_path)
        marker["client_files"]["claim-workflow.py"] = "0" * 64
        self.core.save_json(marker_path, marker)
        self.request.reset_mock()
        with self.assertRaisesRegex(self.core.WorkflowError, "Client source differs"):
            self.shared.SharedWorkflow(self.case.home)
        self.request.assert_not_called()
        self.assertEqual(path.read_bytes(), before)


if __name__ == "__main__":
    unittest.main()
