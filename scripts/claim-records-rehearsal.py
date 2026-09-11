#!/usr/bin/env python3
"""Signed loopback-only knowledge 9 -> 10 claim-record preservation rehearsal.

Synthetic local accounts and balances exercise real SDK governance, admission,
reviews and reads. No existing home, production keys or remote RPC are inputs.
"""
import importlib.util
import json
from pathlib import Path
import shutil

spec = importlib.util.spec_from_file_location("record_rehearsal", Path(__file__).with_name("record-integrity-rehearsal.py"))
record = importlib.util.module_from_spec(spec)
spec.loader.exec_module(record)
record.PLAN = record.base.PLAN = "knowledge-claim-records-v1"
base = record.base


class Rehearsal(record.Rehearsal):
    source_knowledge_version = 9
    target_knowledge_version = 10
    predecessor_v2 = True

    def __init__(self, before, after, directory):
        super().__init__(before, after, directory)
        self.evidence["schema"] = "zerone.claim-records/local-rehearsal-v1"
        self.evidence["script_sha256"] = base.sha256(Path(__file__))
        self.evidence["record_harness_sha256"] = base.sha256(Path(record.__file__))

    def contradiction(self, target, name, reason, evidence):
        receipt = self.tx("knowledge", "submit-contradiction", target,
            "Synthetic local counterclaim " + name, "200000", reason,
            "--evidence-ids", ",".join(evidence), "--domain", "physics", "--category", "empirical",
            signer="current", label=name)
        ids = {a["value"] for e in receipt.get("events", []) for a in e.get("attributes", [])
               if a.get("key") == "counter_claim_id"}
        if len(ids) != 1:
            raise RuntimeError("signed contradiction has no unique claim identity")
        claim_id = ids.pop()
        return claim_id, self.query("knowledge", "claim", claim_id)["claim"]

    def initialize(self):
        super().initialize()
        self.start(self.before, "predecessor-contradiction")
        self.wait(lambda: self.height() > self.source_tuple["height"])
        facts = self.query("knowledge", "facts")["facts"]
        target = next(f["id"] for f in facts if f.get("category") != "conjecture")
        self.old_id, self.old_claim = self.contradiction(target, "before-activation",
            "This signed historical reason was not retained by the predecessor.", ["local-old-evidence"])
        if self.old_claim.get("argument_text") or self.old_claim.get("evidence_ids"):
            raise RuntimeError("predecessor unexpectedly implements new retention behavior")
        base.write_json(self.reports / "old-contradiction-before.json", self.old_claim)
        self.source_tuple = {"height": self.height()}
        self.stop_cleanly()
        self.snapshot = self.root / "claim-record-source-snapshot"
        shutil.copytree(self.home, self.snapshot)

    def history(self, claim_id, height=0):
        extra = ["--height", str(height)] if height else []
        response = self.query("knowledge", "claim-history", claim_id, *extra)
        if response["chain_id"] != self.chain or int(response["block_height"]) <= 0:
            raise RuntimeError("history returned an invalid observation identity")
        if height and int(response["block_height"]) != height:
            raise RuntimeError("history relabelled its actual query height")
        if response["record"]["claim_id"] != claim_id:
            raise RuntimeError("history substituted the requested claim")
        return response

    def happy(self):
        self.progress("scheduling the exact 9-to-10 boundary through signed SDK governance")
        target = self.schedule("claim-records")
        if target + 10 >= int(self.legacy_before["commit_deadline"]):
            raise RuntimeError("preserved review deadline is too close to migration")
        self.halt(target, "claim-records")
        self.start(self.after, "candidate")
        self.wait(lambda: self.height() >= target + 2)
        if self.versions() != dict(self.source_versions, knowledge=10):
            raise RuntimeError("upgrade changed unexpected module versions")
        if int(self.query("upgrade", "applied", record.PLAN)["height"]) != target:
            raise RuntimeError("upgrade receipt differs from scheduled height")
        if self.round(self.legacy_round) != self.legacy_before:
            raise RuntimeError("migration rewrote a retained in-flight review")
        if self.query("knowledge", "claim", self.old_id)["claim"] != self.old_claim:
            raise RuntimeError("migration rewrote an old contradiction")
        old = self.history(self.old_id)
        if old["record"]["claim"].get("argument_text") or old["record"]["claim"].get("evidence_ids"):
            raise RuntimeError("history fabricated a lost historical reason")
        base.write_json(self.reports / "old-contradiction-history.json", old)
        for index in range(2, 5):
            self.commit(self.legacy_round, index, False)
        self.progress("completing retained signed reviews, then recording a direct contradiction")
        self.wait(lambda: int(self.round(self.legacy_round)["phase"]) == 2, seconds=110)
        for index in range(1, 5):
            self.reveal(self.legacy_round, index, False)
        self.wait(lambda: int(self.round(self.legacy_round)["phase"]) == 4)
        original = self.history(self.legacy_claim)
        rows = original["record"]["rounds"]
        if len(rows) != 1 or len(rows[0]["reveals"]) != 4 or not original["record"]["facts"]:
            raise RuntimeError("history omitted retained signed reviews or the resulting fact")
        for index in range(1, 5):
            reveal = next(r for r in rows[0]["reveals"] if r["verifier"] == self.addresses[f"reviewer{index}"])
            if reveal["attestation"]["reason"] != self.review_flags(index)[3]:
                raise RuntimeError("history lost an authenticated review reason")
        fact_id = original["record"]["facts"][0]["fact"]["id"]
        reason = "  Synthetic correction: inspect both cited records.\n"
        evidence = ["local-evidence-z", "local-evidence-a"]
        self.new_id, claim = self.contradiction(fact_id, "after-activation", reason, evidence)
        if claim.get("argument_text") != reason or claim.get("evidence_ids") != evidence:
            raise RuntimeError("activated admission did not retain exact ordered signed input")
        joined = self.history(self.legacy_claim)
        related = [r for r in joined["related_claims"] if r["record"]["claim_id"] == self.new_id]
        if len(related) != 1 or related[0]["links"] != [{"field": "relations.contradicts", "target_id": fact_id}]:
            raise RuntimeError("history lost or invented a direct contradiction link")
        if related[0]["record"]["claim"]["argument_text"] != reason or related[0]["record"]["claim"]["evidence_ids"] != evidence:
            raise RuntimeError("joined history changed contradiction input")
        pinned_height = int(joined["block_height"])
        if self.history(self.legacy_claim, pinned_height) != joined:
            raise RuntimeError("same retained query height produced different history")
        base.write_json(self.reports / "joined-claim-history.json", joined)
        before = self.history(self.new_id)
        before_supply = self.query("bank", "total")
        before_balances = {name: self.query("bank", "balances", self.addresses[name])
                           for name in ("current", "reviewer1", "reviewer2", "reviewer3", "reviewer4")}
        before_height = self.height()
        base.write_json(self.reports / "before-restart-observations.json", {
            "history": before, "total_supply": before_supply,
            "named_balances": before_balances, "height_after_reads": before_height,
        })
        self.stop_cleanly()
        self.start(self.after, "candidate-restart")
        self.wait(lambda: self.height() >= before_height + 3)
        after = self.history(self.new_id)
        after_supply = self.query("bank", "total")
        after_balances = {name: self.query("bank", "balances", self.addresses[name])
                          for name in before_balances}
        base.write_json(self.reports / "after-restart-observations.json", {
            "history": after, "total_supply": after_supply,
            "named_balances": after_balances, "height_after_reads": self.height(),
        })
        if before["record"] != after["record"] or before["related_claims"] != after["related_claims"]:
            raise RuntimeError("restart changed the retained contradiction history")
        if after_supply != before_supply or after_balances != before_balances:
            raise RuntimeError("restart repeated financial effects")
        self.stop_cleanly()
        exported = json.loads(self.cli("export", binary=self.after).stdout)
        knowledge = exported["app_state"]["knowledge"]
        if knowledge.get("claim_records_enabled") is not True:
            raise RuntimeError("export lost the execution marker")
        retained = next(c for c in knowledge["pending_claims"] if c["id"] == self.new_id)
        if retained.get("argument_text") != reason or retained.get("evidence_ids") != evidence:
            raise RuntimeError("export lost the retained contradiction input")
        base.write_json(self.reports / "exported-claim-record.json", retained)
        self.evidence.update({"upgrade_height": target, "old_claim_id": self.old_id, "new_claim_id": self.new_id,
                              "root_claim_id": self.legacy_claim, "queried_height": pinned_height})
        self.evidence["checks"].update({"only_knowledge9_to10": True, "old_records_unchanged": True,
            "signed_reason_and_ordered_evidence_retained": True, "retained_reviews_and_direct_challenge_joined": True,
            "actual_historical_query_height": True, "restart_preserves_records_without_financial_replay": True,
            "export_preserves_execution_flag_and_claim": True, "owned_nodes_stopped_cleanly": True})


if __name__ == "__main__":
    record.Rehearsal = Rehearsal
    record.main()
