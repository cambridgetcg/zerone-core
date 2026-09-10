#!/usr/bin/env python3
"""Loopback-only real predecessor -> survival-reward-handoff-v1 rehearsal.

Imports a labelled synthetic normal claim/ready round through existing genesis
fields. The unchanged predecessor creates the pending reward during actual
block execution. This is not a paid submission or independent review fixture.
No existing home, remote endpoint, production keys, or database edits accepted.
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
import time
import urllib.parse

spec = importlib.util.spec_from_file_location(
    "accounting_rehearsal", Path(__file__).with_name("accounting-authority-rehearsal.py"))
base = importlib.util.module_from_spec(spec)
spec.loader.exec_module(base)
PLAN = "survival-reward-handoff-v1"
base.PLAN = PLAN  # Reuse only the existing scoped process/RPC/SDK-governance helpers.
CLAIM = "synthetic-survival-handoff-claim"
ROUND = "synthetic-survival-handoff-round"
WINDOW = 80


class Rehearsal(base.Rehearsal):
    def __init__(self, before, after, directory):
        super().__init__(before, after, directory)
        self.chain = "survival-local-" + self.root.name.rsplit("-", 1)[-1]
        self.evidence.update({"schema": "zerone.survival-handoff/local-rehearsal-v1",
                              "chain_id": self.chain, "fixture": {
                                  "synthetic_genesis_import": True,
                                  "paid_submission": False, "independent_reviewers": False,
                                  "nominal_reward_uzrn": "200000",
                                  "synthetic_bootstrap_fund_allocation_uzrn": "0",
                                  "knowledge_fee_pool_funded": False}})
        self.plan_info = ""
        self.stops = []

    def stop(self):
        proc = self.proc
        super().stop()
        if proc is not None:
            self.stops.append({"stage": self.stage, "returncode": proc.returncode,
                               "forced_kill": proc.returncode == -9})

    def stop_cleanly(self):
        if self.proc is None:
            raise RuntimeError("expected a running owned node to stop")
        self.stop()
        if self.stops[-1]["returncode"] != 0:
            raise RuntimeError("owned node did not shut down cleanly")

    def raw(self, store, key):
        query = urllib.parse.urlencode({"path": json.dumps(f"/store/{store}/key"),
                                       "data": "0x" + key.hex(), "prove": "false"})
        response = self.rpc_get("/abci_query?" + query)["response"]
        if int(response.get("code", 0)) != 0:
            raise RuntimeError("local committed-state query failed")
        if response.get("key") and base64.b64decode(response["key"]) != key:
            raise RuntimeError("local query substituted requested key")
        return base64.b64decode(response.get("value") or "")

    def pending(self):
        return self.raw("knowledge", b"\x45" + self.fact_id.encode())

    def initialize(self):
        self.cli("init", "local-survival", "--chain-id", self.chain,
                 "--default-denom", "uzrn", binary=self.before)
        self.addresses = {}
        for name in ("validator", "delegator", "reviewer1", "reviewer2", "reviewer3", "reviewer4"):
            self.cli("keys", "add", name, "--keyring-backend", "test", secret=True)
            self.addresses[name] = self.cli("keys", "show", name, "-a", "--keyring-backend", "test").stdout.strip()
        for name in ("validator", "delegator"):
            self.cli("add-genesis-account", self.addresses[name], "1000000000000uzrn")
        path = self.home / "config/genesis.json"
        genesis = json.loads(path.read_text())
        gov = genesis["app_state"]["gov"]["params"]
        gov.update({"voting_period": "5s", "expedited_voting_period": "2s",
                    "min_deposit": [{"denom": "uzrn", "amount": "1000000"}],
                    "expedited_min_deposit": [{"denom": "uzrn", "amount": "2000000"}]})
        knowledge = genesis["app_state"]["knowledge"]
        knowledge["params"]["challenge_duration_blocks"] = str(WINDOW)
        knowledge["bootstrap_fund_allocation"] = "0"
        claim = {"id": CLAIM, "fact_content": "Synthetic imported handoff rehearsal claim",
                 "domain": "physics", "category": "empirical", "submitter": self.addresses["delegator"],
                 "status": 5, "claim_type": 1, "stake": "200000", "verification_round_id": ROUND}
        round_record = {"id": ROUND, "claim_id": CLAIM, "phase": 2,
                        "commit_deadline": "1", "reveal_deadline": "2", "aggregation_deadline": "3",
                        "selected_verifiers": [], "commits": [], "reveals": []}
        for index in range(1, 5):
            verifier = self.addresses[f"reviewer{index}"]
            salt = f"synthetic-handoff-salt-{index}".encode()
            commitment = hashlib.sha256(f"ZRN.commit.v1:{ROUND}:accept:800000:{salt.hex()}".encode()).digest()
            round_record["selected_verifiers"].append(verifier)
            round_record["commits"].append({"verifier": verifier, "commit_hash": base64.b64encode(commitment).decode(), "committed_at_block": "0"})
            round_record["reveals"].append({"verifier": verifier, "vote": "accept", "salt": base64.b64encode(salt).decode(), "revealed_at_block": "0"})
        knowledge.setdefault("pending_claims", []).append(claim)
        knowledge.setdefault("active_rounds", []).append(round_record)
        base.write_json(path, genesis)
        base.write_json(self.reports / "synthetic-genesis-fixture.json", {
            "claim": claim, "round": round_record, "challenge_duration_blocks": WINDOW,
            "bootstrap_fund_allocation_uzrn": "0", "knowledge_fee_pool_funded": False, "paid_submission": False,
            "independent_reviewers": False})
        self.cli("genesis", "gentx", "validator", "1000000000uzrn", "--chain-id", self.chain,
                 "--keyring-backend", "test", "--commission-rate", "0.1",
                 "--commission-max-rate", "0.2", "--commission-max-change-rate", "0.01")
        self.cli("genesis", "collect-gentxs")
        self.cli("genesis", "validate")
        config = self.home / "config/config.toml"
        text = config.read_text()
        for name, value in {"timeout_commit": '"1s"', "timeout_propose": '"500ms"',
                            "prometheus": "false", "pex": "false", "seeds": '""',
                            "persistent_peers": '""', "addr_book_strict": "false", "pprof_laddr": '""'}.items():
            text = re.sub(r"^" + name + r" = .*?$", name + " = " + value, text, flags=re.MULTILINE)
        config.write_text(text)
        self.start(self.before, "predecessor-initial")
        self.wait(lambda: self.height() >= 3)
        round_result = self.query("knowledge", "verification-round", ROUND)["round"]
        base.write_json(self.reports / "predecessor-accepted-round.json", round_result)
        if round_result["verdict"] not in (1, "VERDICT_ACCEPT"):
            raise RuntimeError("real predecessor did not accept the imported ready round")
        verdict_height = int(round_result["verdict_block"])
        self.fact_id = hashlib.sha256(f"ZRN.fact.id.v1:{CLAIM}:{verdict_height}".encode()).hexdigest()[:32]
        self.initial_pending = self.pending()
        pending = json.loads(self.initial_pending)
        if pending["claim_id"] != CLAIM or pending["recipient"] != self.addresses["delegator"] or pending["amount"] != "200000":
            raise RuntimeError("old node created an unexpected pending obligation")
        if self.raw("vesting_rewards", b"\x02" + CLAIM.encode()):
            raise RuntimeError("pending fixture already has a vesting schedule")
        self.deadline = int(pending["deadline"])
        base.write_json(self.reports / "pending-before.json", pending)
        account = self.query("auth", "module-account", "knowledge")["account"]
        self.knowledge_address = account["value"]["address"]
        fee_pool = self.query("bank", "balances", self.knowledge_address)
        base.write_json(self.reports / "synthetic-unfunded-fee-pool.json", {
            "paid_submission": False, "knowledge_fee_pool": fee_pool,
            "limitation": "Imported ready-round fixture proves nominal handoff; it does not prove review-fee payment or verifier payout."})
        self.source_tuple = {"height": int(self.rpc_get("/abci_info")["response"]["last_block_height"])}
        versions = self.query("upgrade", "module-versions")
        self.source_versions = {row["name"]: int(row.get("version", 0)) for row in versions["module_versions"]}
        if (self.source_versions.get("knowledge"), self.source_versions.get("vesting_rewards")) != (6, 2):
            raise RuntimeError("supplied predecessor is not at the frozen handoff source versions")
        base.write_json(self.reports / "versions-before.json", self.source_versions)
        self.stop_cleanly()
        self.snapshot = self.root / "source-snapshot"
        shutil.copytree(self.home, self.snapshot)
        self.evidence["checks"]["real_predecessor_created_pending"] = True

    def halt(self, target, prefix):
        self.progress("waiting for real predecessor to halt at committed H-1")
        path = self.home / "data/upgrade-info.json"
        self.wait(path.exists, seconds=75)
        plan = json.loads(path.read_text())
        if plan["name"] != PLAN or int(plan["height"]) != target or plan.get("info", "") != "":
            raise RuntimeError("predecessor wrote unexpected handoff plan")
        observed = self.rpc_get("/abci_info")["response"]
        if int(observed["last_block_height"]) != target - 1:
            raise RuntimeError("predecessor did not halt at committed H-1")
        if self.pending() != self.initial_pending:
            raise RuntimeError("pending obligation changed before migration")
        base.write_json(self.reports / "halt.json", {"plan": plan, "abci_info": observed})
        self.stop_cleanly()

    def happy(self):
        self.progress("scheduling survival handoff through SDK governance")
        target = self.schedule("handoff")
        if target + 8 >= self.deadline:
            raise RuntimeError("synthetic challenge window leaves insufficient handoff observation time")
        self.halt(target, "handoff")
        happy_home = self.home
        self.home = self.root / "wrong-local-plan"
        shutil.copytree(happy_home, self.home)
        plan_path = self.home / "data/upgrade-info.json"
        altered = json.loads(plan_path.read_text())
        altered["info"] = "unreviewed-local-plan"
        base.write_json(plan_path, altered)
        self.refuse_start("wrong-local-plan", "exact on-chain and local H-1 plan")
        self.home = happy_home
        self.progress("applying current candidate at H with pending bytes retained")
        self.start(self.after, "candidate")
        self.wait(lambda: self.height() >= target + 2)
        if self.pending() != self.initial_pending:
            raise RuntimeError("migration changed pending reward bytes")
        versions = self.query("upgrade", "module-versions")
        observed_versions = {row["name"]: int(row.get("version", 0)) for row in versions["module_versions"]}
        expected_versions = dict(self.source_versions, knowledge=7, vesting_rewards=3)
        if observed_versions != expected_versions:
            raise RuntimeError("handoff changed unexpected module versions")
        applied = self.query("upgrade", "applied", PLAN)
        if int(applied["height"]) != target:
            raise RuntimeError("handoff applied at unexpected height")
        base.write_json(self.reports / "versions-after.json", observed_versions)
        base.write_json(self.reports / "applied-upgrade.json", applied)
        supply = self.query("bank", "total")
        self.progress("waiting for real survival deadline to create exactly one schedule")
        self.wait(lambda: self.height() >= self.deadline + 2, seconds=120)
        if self.pending():
            raise RuntimeError("survived pending obligation was not removed after schedule creation")
        schedule_id = self.raw("vesting_rewards", b"\x02" + CLAIM.encode()).decode()
        if not schedule_id:
            raise RuntimeError("completed handoff has no claim-indexed schedule")
        schedules = self.query("vesting_rewards", "schedule", schedule_id)
        schedule = schedules["schedule"]
        if schedule["claim_id"] != CLAIM or schedule["fact_id"] != self.fact_id or schedule["recipient"] != self.addresses["delegator"] or schedule["total_amount"] != "200000":
            raise RuntimeError("schedule differs from pending nominal obligation")
        if int(schedule["created_at"]) < target or int(schedule["created_at"]) > self.deadline + 2:
            raise RuntimeError("schedule was not created by post-upgrade deadline processing")
        if self.query("bank", "total") != supply:
            raise RuntimeError("schedule creation changed bank supply")
        self.assert_one_schedule(schedule_id, "recipient-schedules-after")
        base.write_json(self.reports / "schedule-after.json", schedules)
        before_height = self.height()
        self.stop_cleanly()
        self.start(self.after, "candidate-restart")
        self.wait(lambda: self.height() >= before_height + 2)
        if self.pending() or self.raw("vesting_rewards", b"\x02" + CLAIM.encode()).decode() != schedule_id:
            raise RuntimeError("restart recreated pending or changed schedule identity")
        after = self.query("vesting_rewards", "schedule", schedule_id)["schedule"]
        for field in ("id", "claim_id", "fact_id", "recipient", "total_amount", "created_at", "accepted_at_block", "released_amount"):
            if after[field] != schedule[field]:
                raise RuntimeError("restart changed immutable schedule history")
        self.assert_one_schedule(schedule_id, "recipient-schedules-after-restart")
        base.write_json(self.reports / "schedule-after-restart.json", after)
        after_height = self.height()
        self.stop_cleanly()
        self.evidence.update({"upgrade_height": target, "survival_deadline": self.deadline,
                              "restart": {"before_height": before_height, "after_height": after_height}})
        self.evidence["checks"].update({"sdk_governance_scheduled": True, "old_halted_at_h_minus_1": True,
            "candidate_applied_at_h": True, "pending_bytes_preserved_at_upgrade": True,
            "only_expected_module_versions_changed": True, "wrong_local_plan_refused": True,
            "deadline_created_schedule_once": True, "nominal_reward_preserved": True,
            "schedule_creation_supply_unchanged": True, "candidate_restart_passed": True,
            "owned_nodes_stopped_cleanly": True})

    def assert_one_schedule(self, schedule_id, label):
        response = self.query("vesting_rewards", "schedules-by-recipient", self.addresses["delegator"])
        matches = [schedule for schedule in response.get("schedules", []) if schedule.get("claim_id") == CLAIM]
        if len(matches) != 1 or matches[0]["id"] != schedule_id:
            raise RuntimeError("recipient primary inventory contains a duplicate or missing handoff schedule")
        base.write_json(self.reports / (label + ".json"), response)

    def run(self):
        try:
            self.progress("creating isolated predecessor with labelled imported ready-round fixture")
            self.initialize()
            happy_home = self.home
            self.home = self.root / "premature"
            shutil.copytree(self.snapshot, self.home)
            self.refuse_start("premature-candidate", "upgrade plan not found")
            self.home = happy_home
            self.evidence["checks"]["premature_startup_refused"] = True
            self.happy()
            if any(row["forced_kill"] for row in self.stops):
                raise RuntimeError("owned node required forced termination")
            self.evidence.update({"result": "passed", "stops": self.stops,
                                  "owned_node_running": self.proc is not None})
            base.write_json(self.reports / "summary.json", self.evidence)
            self.progress("PASS: local evidence at " + str(self.reports))
        except BaseException as exc:
            self.evidence.update({"result": "failed", "stage": self.stage, "error": str(exc)})
            base.write_json(self.reports / "summary.json", self.evidence)
            raise
        finally:
            self.stop()


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
    rehearsal = Rehearsal(before, after, args.evidence_parent.resolve(strict=True))
    print("Owned local evidence directory: " + str(rehearsal.root), flush=True)
    rehearsal.run()


if __name__ == "__main__":
    main()
