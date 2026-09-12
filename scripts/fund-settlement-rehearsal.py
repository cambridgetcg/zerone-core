#!/usr/bin/env python3
"""Signed loopback-only knowledge 10 -> 11 fund lifecycle rehearsal.

Uses two source-built binaries and fresh synthetic accounts. Existing homes,
production keys and remote RPCs are not inputs. Historical claims stay historical;
only new admissions receive prospective funding terms and refund instructions.
"""
import importlib.util
import json
from pathlib import Path
import shutil

spec = importlib.util.spec_from_file_location("record_rehearsal", Path(__file__).with_name("record-integrity-rehearsal.py"))
record = importlib.util.module_from_spec(spec)
spec.loader.exec_module(record)
record.PLAN = record.base.PLAN = "knowledge-fund-settlement-v1"
record.__doc__ = __doc__
base = record.base


class Rehearsal(record.Rehearsal):
    source_knowledge_version = 10
    target_knowledge_version = 11
    predecessor_v2 = True

    def __init__(self, before, after, directory):
        super().__init__(before, after, directory)
        self.chain = "fund-local-" + self.root.name.rsplit("-", 1)[-1]
        self.evidence.update({"schema": "zerone.fund-settlement/local-rehearsal-v1", "chain_id": self.chain,
            "script_sha256": base.sha256(Path(__file__)), "record_harness_sha256": base.sha256(Path(record.__file__))})

    def vote(self, index, current):
        return "reject" if current and index == 4 else "accept"

    @staticmethod
    def amount(response):
        return sum(int(coin["amount"]) for coin in response.get("balances", []) if coin["denom"] == "uzrn")

    def balance(self, name):
        return self.amount(self.query("bank", "balances", self.addresses[name]))

    def tx(self, *args, signer="validator", label):
        # Preserve the actual failure before raising; no retry or replacement tx.
        result = self.cli("tx", *args, "--from", signer, "--chain-id", self.chain,
            "--keyring-backend", "test", "--node", self.rpc, "--fees", "2000000uzrn",
            "--gas", "2000000", "--yes", "--output", "json", check=False)
        if result.returncode:
            base.write_json(self.reports / (label + "-cli-failure.json"),
                {"returncode": result.returncode, "stderr": result.stderr[:12000]})
            raise RuntimeError("transaction " + label + " failed before broadcast; see retained CLI failure")
        response = json.loads(result.stdout)
        base.write_json(self.reports / (label + "-broadcast.json"), response)
        if int(response.get("code", 0)) != 0:
            raise RuntimeError("transaction " + label + " refused CheckTx; see retained response")
        def committed():
            query = self.cli("query", "tx", response["txhash"], "--node", self.rpc,
                "--output", "json", check=False)
            return json.loads(query.stdout) if query.returncode == 0 else False
        receipt = self.wait(committed)
        base.write_json(self.reports / (label + "-tx.json"), receipt)
        if int(receipt.get("height", 0)) <= 0 or int(receipt.get("code", 0)) != 0:
            raise RuntimeError("transaction " + label + " failed execution; see retained receipt")
        return receipt

    def history(self, claim_id, height=0):
        extra = ["--height", str(height)] if height else []
        value = self.query("knowledge", "claim-history", claim_id, *extra)
        observed = int(value["block_height"])
        if value["chain_id"] != self.chain or observed <= 0 or (height and observed != height):
            raise RuntimeError("history substituted its chain or selected height")
        if value["record"]["claim_id"] != claim_id:
            raise RuntimeError("history substituted its requested claim")
        return value

    def contradiction(self, target, label):
        reason = "  Synthetic funding-route correction; retain this exact argument.\n"
        evidence = ["local-fund-evidence-z", "local-fund-evidence-a"]
        receipt = self.tx("knowledge", "submit-contradiction", target,
            "Synthetic counterclaim for " + label, "200000", reason,
            "--evidence-ids", ",".join(evidence), "--domain", "physics", "--category", "empirical",
            signer="current", label=label)
        ids = {a["value"] for event in receipt.get("events", []) for a in event.get("attributes", [])
               if a.get("key") == "counter_claim_id"}
        if len(ids) != 1:
            raise RuntimeError("contradiction has no unique committed claim identity")
        claim_id = ids.pop()
        claim = self.query("knowledge", "claim", claim_id)["claim"]
        if claim.get("argument_text") != reason or claim.get("evidence_ids") != evidence:
            raise RuntimeError("signed contradiction input was not retained exactly")
        return claim_id, claim["verification_round_id"], claim

    def initialize(self):
        super().initialize()
        self.start(self.before, "predecessor-funding-records")
        self.wait(lambda: self.height() > self.source_tuple["height"])
        self.old_claim = self.query("knowledge", "claim", self.legacy_claim)["claim"]
        if self.old_claim.get("funding_terms") or self.legacy_before.get("claim_refund_settlement"):
            raise RuntimeError("predecessor unexpectedly implements new funding fields")
        self.targets = [f["id"] for f in self.query("knowledge", "facts")["facts"]
                        if f.get("category") != "conjecture"][:3]
        if len(self.targets) != 3:
            raise RuntimeError("fixture requires three existing ordinary doctrine facts")
        self.old_counter, self.old_counter_round, self.old_counter_claim = self.contradiction(self.targets[0], "old-counterclaim")
        if self.old_counter_claim.get("funding_terms"):
            raise RuntimeError("predecessor contradiction acquired future terms")
        base.write_json(self.reports / "old-claim-before.json", self.old_claim)
        base.write_json(self.reports / "old-counterclaim-before.json", self.old_counter_claim)
        self.source_tuple = {"height": self.height()}
        self.stop_cleanly()
        self.snapshot = self.root / "fund-source-snapshot"
        shutil.copytree(self.home, self.snapshot)

    @staticmethod
    def check_terms(claim, kind, refundable, retained):
        expected = {"policy_version": 1, "kind": kind, "paid_amount": "200000", "review_budget": "110000",
                    "refundable_amount": str(refundable), "retained_fee": str(retained)}
        if claim.get("funding_terms") != expected:
            raise RuntimeError("funding terms differ from the actual prospective payment route")

    def settled(self, round_id):
        value = self.round(round_id)
        if int(value["phase"]) not in (4, 5):
            return False
        for field in ("verifier_reward_settlement", "claim_refund_settlement"):
            plan = value.get(field)
            if plan and int(plan.get("paid_at_block", 0)) == 0:
                return False
        return value

    def happy(self):
        self.progress("scheduling exact knowledge 10-to-11 through signed SDK governance")
        target = self.schedule("fund-settlement")
        if target + 10 >= int(self.legacy_before["commit_deadline"]):
            raise RuntimeError("preserved review deadline is too close to migration")
        self.halt(target, "fund-settlement")
        self.start(self.after, "candidate")
        self.wait(lambda: self.height() >= target + 2)
        versions = self.versions()
        applied = self.query("upgrade", "applied", record.PLAN)
        if versions != dict(self.source_versions, knowledge=11) or int(applied["height"]) != target:
            raise RuntimeError("unexpected module transition or applied height")
        base.write_json(self.reports / "versions-after.json", versions)
        base.write_json(self.reports / "applied-upgrade.json", applied)
        if self.round(self.legacy_round) != self.legacy_before:
            raise RuntimeError("migration changed the retained in-flight review")
        for claim_id, expected in ((self.legacy_claim, self.old_claim), (self.old_counter, self.old_counter_claim)):
            if self.query("knowledge", "claim", claim_id)["claim"] != expected:
                raise RuntimeError("migration rewrote a historical claim")
        for index in range(2, 5):
            self.commit(self.legacy_round, index, False)

        self.progress("creating explicit ordinary fee and contradiction deposit records")
        payer_before = {name: self.balance(name) for name in ("copier", "current")}
        self.empty_claim, self.empty_round = self.submit("copier", True)
        empty = self.query("knowledge", "claim", self.empty_claim)["claim"]
        self.check_terms(empty, 1, 0, 90000)
        self.zero_counter, self.zero_round, zero = self.contradiction(self.targets[1], "new-zero-review-counterclaim")
        self.check_terms(zero, 2, 90000, 0)
        self.reviewed_counter, self.reviewed_round, reviewed = self.contradiction(self.targets[2], "new-reviewed-counterclaim")
        self.check_terms(reviewed, 2, 90000, 0)
        for index in range(1, 5):
            self.commit(self.reviewed_round, index, True)

        self.wait(lambda: int(self.round(self.legacy_round)["phase"]) == 2, seconds=110)
        for index in range(1, 5):
            self.reveal(self.legacy_round, index, False)
        old_round = self.wait(lambda: self.settled(self.legacy_round))
        if old_round.get("claim_refund_settlement") or self.query("knowledge", "claim", self.legacy_claim)["claim"].get("funding_terms"):
            raise RuntimeError("old review acquired inferred refund terms")
        base.write_json(self.reports / "old-round-completed.json", old_round)
        self.wait(lambda: int(self.round(self.reviewed_round)["phase"]) == 2, seconds=110)
        reviewers_before = {f"reviewer{i}": self.balance(f"reviewer{i}") for i in range(1, 5)}
        for index in range(1, 5):
            self.reveal(self.reviewed_round, index, True)
        final_reviewed = self.wait(lambda: self.settled(self.reviewed_round), seconds=80)
        if int(final_reviewed.get("verdict", 0)) != 3 or len(final_reviewed["reveals"]) != 4:
            raise RuntimeError("expected four retained reviews and an inconclusive verdict")
        payment = final_reviewed.get("verifier_reward_settlement") or {}
        expected_reviewers = {self.addresses[name]: 27500 for name in reviewers_before}
        if {p["verifier"]: int(p["amount"]) for p in payment.get("payments", [])} != expected_reviewers or int(payment.get("withheld_total", 0)) != 0:
            raise RuntimeError("fixed fee pool or dissent payment changed")
        reviewers_after = {name: self.balance(name) for name in reviewers_before}
        if any(reviewers_after[name] - reviewers_before[name] != 27500 - 2000000 for name in reviewers_before):
            raise RuntimeError("actual reviewer bank change differs from reward less one reveal fee")
        final_empty = self.wait(lambda: self.settled(self.empty_round), seconds=80)
        final_zero = self.wait(lambda: self.settled(self.zero_round), seconds=80)
        for final, recipient, expected in ((final_empty, "copier", 110000), (final_zero, "current", 200000), (final_reviewed, "current", 90000)):
            refund = final.get("claim_refund_settlement") or {}
            if refund.get("recipient") != self.addresses[recipient] or int(refund.get("amount", 0)) != expected or int(refund.get("paid_at_block", 0)) == 0:
                raise RuntimeError("terminal refund instruction differs from retained funding terms")
        if final_empty.get("verifier_reward_settlement") or final_zero.get("verifier_reward_settlement"):
            raise RuntimeError("zero-review round invented reviewer payments")
        payer_after = {name: self.balance(name) for name in payer_before}
        if payer_after["copier"] - payer_before["copier"] != -2000000 - 90000:
            raise RuntimeError("ordinary no-review payer did not receive exactly unused55%")
        if payer_after["current"] - payer_before["current"] != -4000000 - 110000:
            raise RuntimeError("contradiction payer refunds differ from actual two-deposit bank accounting")
        old_counter = self.wait(lambda: self.settled(self.old_counter_round))
        if old_counter.get("claim_refund_settlement"):
            raise RuntimeError("historical contradiction acquired an inferred refund")
        base.write_json(self.reports / "bank-comparison.json", {"payers_before": payer_before, "payers_after": payer_after,
            "reviewers_before": reviewers_before, "reviewers_after": reviewers_after, "tx_fee_uzrn": "2000000"})
        terminal = {self.empty_round: final_empty, self.zero_round: final_zero, self.reviewed_round: final_reviewed,
                    self.legacy_round: old_round, self.old_counter_round: old_counter}
        base.write_json(self.reports / "terminal-rounds.json", terminal)
        history = self.history(self.reviewed_counter)
        pinned_height = int(history["block_height"])
        if self.history(self.reviewed_counter, pinned_height) != history:
            raise RuntimeError("historical projection changed at the selected height")
        base.write_json(self.reports / "settled-claim-history.json", history)
        supply = self.query("bank", "total")
        before_height = self.height()
        self.stop_cleanly()
        self.start(self.after, "candidate-restart")
        self.wait(lambda: self.height() >= before_height + 3)
        if any(self.round(rid) != expected for rid, expected in terminal.items()):
            raise RuntimeError("restart changed old or new terminal funding records")
        if self.query("bank", "total") != supply or any(self.balance(name) != value for name, value in {**payer_after, **reviewers_after}.items()):
            raise RuntimeError("restart repeated a payment or changed supply")
        if self.history(self.reviewed_counter, pinned_height) != history:
            raise RuntimeError("restart lost the selected historical record")
        self.stop_cleanly()
        exported = json.loads(self.cli("export", binary=self.after).stdout)["app_state"]["knowledge"]
        if exported.get("fund_settlement_enabled") is not True:
            raise RuntimeError("export lost the explicit execution marker")
        exported_rounds = {r["id"]: r for r in exported["completed_rounds"]}
        for rid, expected in terminal.items():
            # Genesis uses named enums; compare every nested financial instruction.
            for field in ("verifier_reward_settlement", "claim_refund_settlement"):
                if exported_rounds[rid].get(field) != expected.get(field):
                    raise RuntimeError("export changed a retained financial instruction")
        base.write_json(self.reports / "exported-financial-records.json", {"fund_settlement_enabled": True,
            "claims": [c for c in exported["pending_claims"] if c["id"] in (self.empty_claim, self.zero_counter, self.reviewed_counter, self.legacy_claim, self.old_counter)],
            "rounds": [exported_rounds[rid] for rid in terminal]})
        self.evidence.update({"upgrade_height": target, "queried_height": pinned_height,
            "legacy_claim_id": self.legacy_claim, "old_counterclaim_id": self.old_counter,
            "new_no_review_claim_id": self.empty_claim, "new_zero_review_counterclaim_id": self.zero_counter,
            "new_reviewed_counterclaim_id": self.reviewed_counter})
        self.evidence["checks"].update({"only_knowledge10_to11": True, "old_records_unchanged_at_upgrade": True,
            "old_inflight_review_completes_without_repricing": True, "old_contradiction_no_inferred_refund": True,
            "new_admission_terms_match_funding_route": True, "zero_review_budget_refunded": True,
            "contradiction_remainder_refunded_without_verdict_dependence": True, "dissent_paid_from_fixed_pool": True,
            "actual_bank_deltas_match_payments_and_transaction_fees": True,
            "restart_no_financial_replay": True, "export_preserves_funding_and_refund_records": True,
            "retained_historical_query_survives_restart": True, "owned_nodes_stopped_cleanly": True})


if __name__ == "__main__":
    record.Rehearsal = Rehearsal
    record.main()
