#!/usr/bin/env python3
"""Fresh loopback-only e0f9 survival predecessor -> record integrity rehearsal.

Disposable locally funded keys sign real account-registration, claim, review and
SDK-governance transactions. These are synthetic participants, not independent
reviewers or production evidence. Existing homes and remote RPCs are not inputs.
"""
from __future__ import annotations
import argparse
import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil

spec = importlib.util.spec_from_file_location("survival_rehearsal", Path(__file__).with_name("survival-handoff-rehearsal.py"))
survival = importlib.util.module_from_spec(spec)
spec.loader.exec_module(survival)
base = survival.base
PLAN = "knowledge-record-integrity-v1"
base.PLAN = PLAN


class Rehearsal(survival.Rehearsal):
    def __init__(self, before, after, directory):
        super().__init__(before, after, directory)
        self.chain = "record-local-" + self.root.name.rsplit("-", 1)[-1]
        self.evidence.update({"schema": "zerone.record-integrity/local-rehearsal-v1", "chain_id": self.chain,
            "fixture": {"synthetic_genesis_balances": True, "real_signed_transactions": True,
                        "independent_reviewers": False, "production_state": False},
            "script_sha256": base.sha256(Path(__file__))})

    def round(self, round_id):
        return self.query("knowledge", "verification-round", round_id)["round"]

    def versions(self):
        return {r["name"]: int(r.get("version", 0)) for r in self.query("upgrade", "module-versions")["module_versions"]}

    def submit(self, name, current):
        content = "Synthetic local record integrity " + name + " claim: 2 plus 2 equals 4."
        extra = ["--reasoning-trace", "Local arithmetic demonstration; not an independent scientific review."] if current else []
        receipt = self.tx("knowledge", "submit-claim", content, "physics", "empirical", "200000", *extra,
                          signer=name, label=name + "-claim")
        ids = {a["value"] for e in receipt.get("events", []) for a in e.get("attributes", []) if a.get("key") == "claim_id"}
        if len(ids) != 1:
            raise RuntimeError("submission receipt has no unique claim identity")
        claim_id = ids.pop()
        claim = self.query("knowledge", "claim", claim_id)["claim"]
        if claim["submitter"] != self.addresses[name] or claim["fact_content"] != content:
            raise RuntimeError("signed submission identity/content mismatch")
        if current and claim.get("reasoning_trace") != extra[-1]:
            raise RuntimeError("new claim lost signed reasoning")
        base.write_json(self.reports / (name + "-claim-record.json"), claim)
        return claim_id, claim["verification_round_id"]

    @staticmethod
    def review_flags(index):
        return ["--confidence", "800000", "--review-reason", f"Synthetic reviewer {index} checked the stated arithmetic.",
                "--review-scope", "This local fixture only", "--review-evidence", "local-arithmetic-example"]

    @staticmethod
    def salt(index):
        return f"record-integrity-public-test-salt-{index}".encode().hex()

    def commit(self, round_id, index, current):
        flags = self.review_flags(index) if current else []
        self.tx("knowledge", "submit-commitment", round_id, "--vote", "accept", "--salt", self.salt(index),
                *flags, signer=f"reviewer{index}", label=("v2" if current else "legacy") + f"-commit-{index}")

    def reveal(self, round_id, index, current):
        flags = self.review_flags(index) if current else []
        self.tx("knowledge", "submit-reveal", round_id, "accept", self.salt(index), *flags,
                signer=f"reviewer{index}", label=("v2" if current else "legacy") + f"-reveal-{index}")

    def initialize(self):
        self.cli("init", "local-record-integrity", "--chain-id", self.chain, "--default-denom", "uzrn")
        self.addresses = {}
        for name in ("validator", "legacy", "current", "reviewer1", "reviewer2", "reviewer3", "reviewer4", "copier"):
            self.cli("keys", "add", name, "--keyring-backend", "test", secret=True)
            address = self.cli("keys", "show", name, "-a", "--keyring-backend", "test").stdout.strip()
            self.addresses[name] = address
            self.cli("add-genesis-account", address, "1000000000000uzrn")
        path = self.home / "config/genesis.json"
        genesis = json.loads(path.read_text())
        genesis["app_state"]["gov"]["params"].update({"voting_period": "5s", "expedited_voting_period": "2s",
            "min_deposit": [{"denom": "uzrn", "amount": "1000000"}],
            "expedited_min_deposit": [{"denom": "uzrn", "amount": "2000000"}]})
        genesis["app_state"]["knowledge"]["params"].update({"commit_phase_blocks": "65", "reveal_phase_blocks": "25", "aggregation_phase_blocks": "5"})
        base.write_json(path, genesis)
        self.cli("genesis", "gentx", "validator", "1000000000uzrn", "--chain-id", self.chain,
                 "--keyring-backend", "test", "--commission-rate", "0.1", "--commission-max-rate", "0.2", "--commission-max-change-rate", "0.01")
        self.cli("genesis", "collect-gentxs")
        self.cli("genesis", "validate")
        config = self.home / "config/config.toml"
        text = config.read_text()
        for name, value in {"timeout_commit": '"1s"', "timeout_propose": '"500ms"', "prometheus": "false", "pex": "false",
                            "seeds": '""', "persistent_peers": '""', "addr_book_strict": "false", "pprof_laddr": '""'}.items():
            text = re.sub(r"^" + name + r" = .*?$", name + " = " + value, text, flags=re.MULTILINE)
        config.write_text(text)
        self.start(self.before, "predecessor-initial")
        self.wait(lambda: self.height() >= 3)
        self.progress("registering disposable accounts and submitting a real legacy claim/commitment")
        for name in ("legacy", "current", "reviewer1", "reviewer2", "reviewer3", "reviewer4", "copier"):
            self.tx("zerone_auth", "onboard", "agent", signer=name, label=name + "-registration")
        self.legacy_claim, self.legacy_round = self.submit("legacy", False)
        self.commit(self.legacy_round, 1, False)
        self.legacy_before = self.round(self.legacy_round)
        if int(self.legacy_before.get("commitment_scheme", 0)) != 0 or len(self.legacy_before["commits"]) != 1:
            raise RuntimeError("predecessor did not produce one genuine legacy commitment")
        base.write_json(self.reports / "legacy-round-before.json", self.legacy_before)
        self.source_versions = self.versions()
        if (self.source_versions.get("knowledge"), self.source_versions.get("vesting_rewards")) != (7, 3):
            raise RuntimeError("predecessor is not the frozen survival target")
        self.source_tuple = {"height": self.height()}
        self.stop_cleanly()
        self.snapshot = self.root / "source-snapshot"
        shutil.copytree(self.home, self.snapshot)

    def halt(self, target, prefix):
        self.progress("waiting for the real predecessor at committed H-1")
        path = self.home / "data/upgrade-info.json"
        self.wait(path.exists, seconds=75)
        plan = json.loads(path.read_text())
        if plan["name"] != PLAN or int(plan["height"]) != target or plan.get("info", "") != "":
            raise RuntimeError("predecessor wrote unexpected record upgrade plan")
        observed = self.rpc_get("/abci_info")["response"]
        if int(observed["last_block_height"]) != target - 1:
            raise RuntimeError("predecessor did not halt at committed H-1")
        base.write_json(self.reports / "halt.json", {"plan": plan, "abci_info": observed})
        self.stop_cleanly()

    def refused_tx(self, *args, signer, label):
        result = self.cli("tx", *args, "--from", signer, "--chain-id", self.chain, "--keyring-backend", "test",
                          "--node", self.rpc, "--fees", "2000000uzrn", "--gas", "2000000", "--yes", "--output", "json", check=False)
        if result.returncode:
            raise RuntimeError("negative fixture failed before signed broadcast: " + result.stderr[:1000])
        response = json.loads(result.stdout)
        if int(response.get("code", 0)) == 0:
            def committed():
                value = self.cli("query", "tx", response["txhash"], "--node", self.rpc, "--output", "json", check=False)
                return json.loads(value.stdout) if value.returncode == 0 else False
            response = self.wait(committed)
        if int(response.get("height", 0)) <= 0 or int(response.get("code", 0)) != 26 or response.get("codespace") != "knowledge":
            raise RuntimeError("copied reveal did not reach DeliverTx with exact knowledge ErrRevealMismatch")
        base.write_json(self.reports / (label + "-tx.json"), response)

    def happy(self):
        self.progress("scheduling the named record boundary through SDK governance")
        target = self.schedule("record")
        if target + 10 >= int(self.legacy_before["commit_deadline"]):
            raise RuntimeError("legacy round deadline is too close to migration")
        self.halt(target, "record")
        self.start(self.after, "candidate")
        self.wait(lambda: self.height() >= target + 2)
        observed = self.round(self.legacy_round)
        if observed != self.legacy_before:
            raise RuntimeError("migration changed an in-flight legacy round")
        expected = dict(self.source_versions, knowledge=8)
        if self.versions() != expected or int(self.query("upgrade", "applied", PLAN)["height"]) != target:
            raise RuntimeError("upgrade changed unexpected versions or applied height")
        base.write_json(self.reports / "versions-after.json", expected)
        for index in range(2, 5):
            self.commit(self.legacy_round, index, False)
        self.current_claim, self.current_round = self.submit("current", True)
        current = self.round(self.current_round)
        if int(current.get("commitment_scheme", 0)) != 2 or current.get("commitment_chain_id") != self.chain:
            raise RuntimeError("new signed claim did not create a chain-bound v2 round")
        for index in range(1, 5):
            self.commit(self.current_round, index, True)
        committed = self.round(self.current_round)
        original = next(c for c in committed["commits"] if c["verifier"] == self.addresses["reviewer1"])
        self.tx("knowledge", "submit-commitment", self.current_round, base64.b64decode(original["commit_hash"]).hex(),
                signer="copier", label="v2-copied-hash-commit")
        self.progress("revealing the preserved legacy round with original v1 preimages")
        self.wait(lambda: int(self.round(self.legacy_round)["phase"]) == 2, seconds=110)
        for index in range(1, 5):
            self.reveal(self.legacy_round, index, False)
        self.wait(lambda: int(self.round(self.legacy_round)["phase"]) == 4)
        legacy = self.round(self.legacy_round)
        if int(legacy.get("commitment_scheme", 0)) != 0 or any(r.get("attestation") or int(r.get("confidence", 0)) for r in legacy["reveals"]):
            raise RuntimeError("legacy review was falsely relabelled as v2 evidence")
        base.write_json(self.reports / "legacy-round-completed.json", legacy)
        self.progress("revealing signed v2 reviews and refusing another signer's copied preimage")
        self.wait(lambda: int(self.round(self.current_round)["phase"]) == 2, seconds=110)
        payout_before = {name: self.query("bank", "balances", self.addresses[name]) for name in ("reviewer1", "reviewer2", "reviewer3", "reviewer4")}
        self.reveal(self.current_round, 1, True)
        self.refused_tx("knowledge", "submit-reveal", self.current_round, "accept", self.salt(1), *self.review_flags(1),
                        signer="copier", label="v2-copy-refused")
        for index in range(2, 5):
            self.reveal(self.current_round, index, True)
        self.wait(lambda: int(self.round(self.current_round)["phase"]) == 4, seconds=65)
        final = self.round(self.current_round)
        if len(final["reveals"]) != 4 or int(final.get("verdict", 0)) != 1:
            raise RuntimeError("new round did not accept exactly four authenticated reveals")
        for index in range(1, 5):
            matches = [r for r in final["reveals"] if r["verifier"] == self.addresses[f"reviewer{index}"]]
            if len(matches) != 1:
                raise RuntimeError("new review signer identity was lost or duplicated")
            reveal = matches[0]
            att = reveal.get("attestation") or {}
            expected_att = {"reason": self.review_flags(index)[3], "scope": "This local fixture only",
                            "evidence_ids": ["local-arithmetic-example"], "method_id": ""}
            if reveal["vote"] != "accept" or base64.b64decode(reveal["salt"]).hex() != self.salt(index) or int(reveal["confidence"]) != 800000 or {key: att.get(key, "") for key in expected_att} != expected_att:
                raise RuntimeError("retained v2 review differs from its exact signed preimage")
        plan = final.get("verifier_reward_settlement")
        if not plan or int(plan.get("paid_at_block", 0)) == 0:
            raise RuntimeError("ordinary paid submission did not retain completed verifier payment instructions")
        def amount(response):
            return sum(int(coin["amount"]) for coin in response.get("balances", []) if coin["denom"] == "uzrn")
        payout_after = {name: self.query("bank", "balances", self.addresses[name]) for name in payout_before}
        payments = {row["verifier"]: int(row["amount"]) for row in plan["payments"]}
        if set(payments) != {self.addresses[name] for name in payout_before}:
            raise RuntimeError("payment plan contains unexpected reviewer identities")
        for name in payout_before:
            if amount(payout_after[name]) - amount(payout_before[name]) != payments[self.addresses[name]] - 2000000:
                raise RuntimeError("actual bank change differs from frozen payment less one signed reveal transaction fee")
        base.write_json(self.reports / "v2-bank-payment-comparison.json", {"before": payout_before, "after": payout_after,
            "one_reveal_fee_uzrn_each": "2000000", "plan": plan})
        base.write_json(self.reports / "v2-round-completed.json", final)
        before_height = self.height()
        before_supply = self.query("bank", "total")
        before_balances = {name: self.query("bank", "balances", self.addresses[name]) for name in ("reviewer1", "reviewer2", "reviewer3", "reviewer4")}
        self.stop_cleanly()
        self.start(self.after, "candidate-restart")
        self.wait(lambda: self.height() >= before_height + 3)
        if self.round(self.current_round) != final or self.round(self.legacy_round) != legacy:
            raise RuntimeError("restart changed retained terminal reviews/payments")
        if self.query("bank", "total") != before_supply or any(self.query("bank", "balances", self.addresses[name]) != balance for name, balance in before_balances.items()):
            raise RuntimeError("restart repeated reviewer payment or changed supply")
        after_height = self.height()
        self.stop_cleanly()
        self.evidence.update({"upgrade_height": target, "legacy_round_id": self.legacy_round, "v2_round_id": self.current_round,
                              "restart": {"before_height": before_height, "after_height": after_height}})
        self.evidence["checks"].update({"real_signed_account_registration": True, "paid_signed_ordinary_claims": True,
            "legacy_commitment_preserved_across_h": True, "legacy_v1_reveals_complete_after_h": True,
            "new_round_uses_scheme2_and_original_chain": True, "signed_review_reason_confidence_retained": True,
            "copied_commitment_preimage_refused_for_other_signer": True, "only_knowledge7_to8": True,
            "paid_plan_matches_actual_bank_changes": True, "paid_plan_retained_without_restart_repayment": True, "supply_unchanged_on_restart": True,
            "owned_nodes_stopped_cleanly": True})

    def run(self):
        try:
            self.initialize()
            source_home = self.home
            self.home = self.root / "premature-candidate"
            shutil.copytree(self.snapshot, self.home)
            self.refuse_start("premature-candidate", "upgrade plan not found")
            self.home = source_home
            self.evidence["checks"]["premature_startup_refused"] = True
            self.happy()
            self.evidence.update({"result": "passed", "stops": self.stops, "owned_node_running": self.proc is not None})
            base.write_json(self.reports / "summary.json", self.evidence)
            self.progress("PASS: local evidence at " + str(self.reports))
        except BaseException as exc:
            self.evidence.update({"result": "failed", "stage": self.stage, "error": str(exc)})
            base.write_json(self.reports / "summary.json", self.evidence)
            raise
        finally:
            self.stop()
            self.evidence.update({"stops": self.stops, "owned_node_running": self.proc is not None})
            base.write_json(self.reports / "summary.json", self.evidence)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--before", type=Path, required=True)
    parser.add_argument("--after", type=Path, required=True)
    parser.add_argument("--evidence-parent", type=Path, default=Path("/tmp"))
    args = parser.parse_args()
    os.umask(0o077)
    before, after = args.before.resolve(strict=True), args.after.resolve(strict=True)
    if before == after or not os.access(before, os.X_OK) or not os.access(after, os.X_OK):
        parser.error("two distinct executable binaries are required")
    trial = Rehearsal(before, after, args.evidence_parent.resolve(strict=True))
    print("Owned local evidence directory: " + str(trial.root), flush=True)
    trial.run()


if __name__ == "__main__":
    main()
