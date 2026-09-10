#!/usr/bin/env python3
"""Local-only real-predecessor accounting-authority-v1 handoff rehearsal.

Requires two independently built binaries. Generates fresh test keys in a new
0700 temporary home; all listeners bind loopback and all peers/discovery are off.
Never accepts an existing node home or contacts a deployed network. Evidence is
local development evidence, not release approval or an activation instruction.
"""
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import socket
import subprocess
import tempfile
import time
import urllib.request

PLAN = "accounting-authority-v1"
STAKING = "zerone_staking"
GOV = "zerone_gov"


def sha256(path):
    h = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def manifest(path):
    """Check every source DB byte and mode independently of the candidate CLI."""
    if path.is_symlink() or not path.is_dir():
        raise RuntimeError("source database must be an owned regular directory")
    out = [[".", path.stat().st_mode & 0o777, "directory"]]
    for entry in sorted(path.rglob("*")):
        if entry.is_symlink():
            raise RuntimeError("symlink in owned source database")
        out.append([str(entry.relative_to(path)), entry.stat().st_mode & 0o777,
                    sha256(entry) if entry.is_file() else "directory"])
    return out


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


class Rehearsal:
    def __init__(self, before, after, directory):
        self.before, self.after = before, after
        self.root = Path(tempfile.mkdtemp(prefix="zerone-accounting-handoff-", dir=directory))
        self.root.chmod(0o700)
        self.reports = self.root / "reports"
        self.reports.mkdir()
        self.home = self.root / "happy"
        self.chain = "accounting-local-" + self.root.name.rsplit("-", 1)[-1]
        self.proc = None
        self.log = None
        self.binary = before
        self.stage = "initializing"
        sockets = [socket.socket() for _ in range(2)]
        try:
            for sock in sockets:
                sock.bind(("127.0.0.1", 0))
            self.rpc_port, self.p2p_port = [sock.getsockname()[1] for sock in sockets]
        finally:
            for sock in sockets:
                sock.close()
        self.rpc = f"http://127.0.0.1:{self.rpc_port}"
        self.evidence = {"schema": "zerone.accounting-authority/local-rehearsal-v1",
                         "release_evidence": False, "chain_id": self.chain,
                         "predecessor_sha256": sha256(before),
                         "candidate_sha256": sha256(after), "checks": {}}

    def progress(self, message):
        self.stage = message
        print(message, flush=True)

    def cli(self, *args, binary=None, check=True, secret=False, timeout=45, log_level="error"):
        completed = subprocess.run([str(binary or self.binary), *map(str, args),
                                    "--home", str(self.home),
                                    *([] if log_level is None else ["--log_level", log_level])], text=True,
                                   capture_output=True, timeout=timeout)
        if check and completed.returncode:
            # Never print key generation output, seed material, or command args.
            detail = "redacted key operation" if secret else completed.stderr[:3000]
            raise RuntimeError(f"CLI failed during {self.stage}: {detail}")
        return completed

    def query(self, *args):
        response = self.cli("query", *args, "--node", self.rpc, "--output", "json")
        return json.loads(response.stdout)

    def rpc_get(self, path):
        with urllib.request.urlopen(self.rpc + path, timeout=3) as response:
            payload = json.load(response)
        if "error" in payload:
            raise RuntimeError(str(payload["error"]))
        return payload["result"]

    def height(self):
        status = self.rpc_get("/status")
        if status["node_info"]["network"] != self.chain:
            raise RuntimeError("loopback node returned an unexpected chain identity")
        return int(status["sync_info"]["latest_block_height"])

    def wait(self, predicate, seconds=60):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            try:
                result = predicate()
                if result:
                    return result
            except (OSError, ValueError, KeyError):
                pass
            if self.proc is not None and self.proc.poll() is not None:
                raise RuntimeError(f"node exited during {self.stage}; see private node log")
            time.sleep(0.15)
        raise RuntimeError(f"timed out during {self.stage}")

    def start(self, binary, label):
        if self.proc is not None:
            raise RuntimeError("owned node already exists")
        self.binary = binary
        self.log = (self.root / (label + ".log")).open("ab")
        args = [str(binary), "start", "--home", str(self.home),
                "--minimum-gas-prices", "1uzrn", "--pruning", "nothing",
                "--rpc.laddr", f"tcp://127.0.0.1:{self.rpc_port}",
                "--rpc.pprof_laddr", "", "--rpc.unsafe=false",
                "--p2p.laddr", f"tcp://127.0.0.1:{self.p2p_port}",
                "--p2p.external-address", f"127.0.0.1:{self.p2p_port}",
                "--p2p.persistent_peers", "", "--p2p.seeds", "", "--p2p.pex=false",
                "--api.enable=false", "--grpc.enable=false", "--grpc-web.enable=false",
                "--consensus.create_empty_blocks=true", "--log_level", "info",
                "--log_no_color"]
        self.proc = subprocess.Popen(args, stdout=self.log, stderr=subprocess.STDOUT,
                                     start_new_session=True)

    def stop(self):
        if self.proc is not None:
            if self.proc.poll() is None:
                self.proc.send_signal(signal.SIGINT)
                try:
                    self.proc.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    self.proc.kill()
                    self.proc.wait(timeout=5)
            self.proc = None
        if self.log is not None:
            self.log.close()
            self.log = None

    def tx(self, *args, signer="validator", label):
        response = json.loads(self.cli("tx", *args, "--from", signer,
                                      "--chain-id", self.chain, "--keyring-backend", "test",
                                      "--node", self.rpc, "--fees", "2000000uzrn",
                                      "--gas", "2000000", "--yes", "--output", "json").stdout)
        if int(response.get("code", 0)) != 0:
            raise RuntimeError(f"transaction {label} rejected: {response.get('raw_log')}")
        digest = response["txhash"]
        def committed():
            result = self.cli("query", "tx", digest, "--node", self.rpc,
                              "--output", "json", check=False)
            if result.returncode:
                return False
            return json.loads(result.stdout)
        receipt = self.wait(committed)
        if int(receipt.get("code", 0)) != 0:
            raise RuntimeError(f"transaction {label} failed: {receipt.get('raw_log')}")
        write_json(self.reports / (label + "-tx.json"), receipt)
        return receipt

    def initialize(self):
        self.cli("init", "local-accounting", "--chain-id", self.chain,
                 "--default-denom", "uzrn", binary=self.before)
        self.addresses = {}
        for name in ("validator", "delegator"):
            self.cli("keys", "add", name, "--keyring-backend", "test", secret=True)
            self.addresses[name] = self.cli("keys", "show", name, "-a",
                                             "--keyring-backend", "test").stdout.strip()
            self.cli("add-genesis-account", self.addresses[name], "1000000000000uzrn")
        path = self.home / "config" / "genesis.json"
        genesis = json.loads(path.read_text())
        params = genesis["app_state"]["gov"]["params"]
        params["voting_period"] = "5s"
        params["expedited_voting_period"] = "2s"
        params["min_deposit"] = [{"denom": "uzrn", "amount": "1000000"}]
        params["expedited_min_deposit"] = [{"denom": "uzrn", "amount": "2000000"}]
        genesis["app_state"][STAKING]["params"]["unbonding_period"] = 6
        write_json(path, genesis)
        self.cli("genesis", "gentx", "validator", "1000000000uzrn", "--chain-id", self.chain,
                 "--keyring-backend", "test", "--commission-rate", "0.1",
                 "--commission-max-rate", "0.2", "--commission-max-change-rate", "0.01")
        self.cli("genesis", "collect-gentxs")
        self.cli("genesis", "validate")
        config = self.home / "config" / "config.toml"
        contents = config.read_text()
        for name, value in {"timeout_commit": '"1s"', "timeout_propose": '"500ms"',
                            "prometheus": "false", "pex": "false", "seeds": '""',
                            "persistent_peers": '""', "addr_book_strict": "false",
                            "pprof_laddr": '""'}.items():
            contents = re.sub(r"^" + name + r" = .*?$", name + " = " + value,
                              contents, flags=re.MULTILINE)
        config.write_text(contents)
        self.start(self.before, "predecessor-initial")
        self.wait(lambda: self.height() >= 3)
        self.progress("funding predecessor custom self and ordinary delegation claims")
        # Public consensus key bytes are harmless; the independent custom ledger
        # does not own CometBFT consensus registration.
        key = json.loads((self.home / "config" / "priv_validator_key.json").read_text())["pub_key"]["value"]
        self.tx(STAKING, "register-validator", base64.b64decode(key).hex(), "333000",
                "--identity", "did:zrn:local-accounting", label="register-custom-validator")
        self.tx(STAKING, "delegate", self.addresses["validator"], "777000",
                signer="delegator", label="ordinary-delegation")
        self.module_addresses = {}
        for name in (STAKING, GOV):
            account = self.query("auth", "module-account", name)["account"]
            self.module_addresses[name] = account["value"]["address"]
        self.initial = self.claims()
        self.assert_claims(self.initial, 333000, 777000, 1110000)
        write_json(self.reports / "claims-before.json", self.initial)
        # Comet /status may advance while ABCI Commit is still in flight.
        # Observe the application\'s newly COMMITTED tuple directly, then stop
        # within the next one-second interval. Preparation rejects any race.
        initial_height = int(self.rpc_get("/abci_info")["response"]["last_block_height"])
        def next_commit():
            response = self.rpc_get("/abci_info")["response"]
            return response if int(response["last_block_height"]) > initial_height else False
        info = self.wait(next_commit)
        self.source_tuple = {"height": int(info["last_block_height"]),
                             "app_hash": base64.b64decode(info["last_block_app_hash"]).hex(),
                             "chain_id": self.chain}
        self.stop()
        self.snapshot = self.root / "source-snapshot"
        shutil.copytree(self.home, self.snapshot)
        write_json(self.reports / "observed-source-tuple.json", self.source_tuple)

    def claims(self):
        operator = self.addresses["validator"]
        return {"validator": self.query(STAKING, "validator", operator)["validator"],
                "self": self.query(STAKING, "delegation", operator, operator)["delegation"],
                "ordinary": self.query(STAKING, "delegation", self.addresses["delegator"], operator)["delegation"],
                "custody": {name: self.query("bank", "balances", address)["balances"]
                            for name, address in self.module_addresses.items()}}

    @staticmethod
    def assert_claims(claims, self_amount, ordinary, custody):
        val = claims["validator"]
        actual = (int(val["self_delegation"]), int(val["delegated_stake"]),
                  int(val["total_stake"]), int(claims["self"]["amount"]),
                  int(claims["ordinary"]["amount"]))
        expected = (self_amount, ordinary, self_amount + ordinary, self_amount, ordinary)
        if actual != expected:
            raise RuntimeError(f"claim/aggregate mismatch: {actual} != {expected}")
        if claims["custody"][STAKING] != [{"denom": "uzrn", "amount": str(custody)}]:
            raise RuntimeError("custom staking custody differs from claimant backing")

    def prepare(self):
        self.progress("preparing exact stopped-source commitment and checking source immutability")
        source = self.home / "data" / "application.db"
        before = manifest(source)
        flags = ["accounting-plan-info", "--expected-chain-id", self.chain,
                 "--expected-height", str(self.source_tuple["height"]),
                 "--expected-app-hash", self.source_tuple["app_hash"]]
        wrong = list(flags)
        wrong[-1] = "00" * 32
        refusal = self.cli(*wrong, binary=self.after, check=False, log_level=None)
        if refusal.returncode == 0 or '"plan_info"' in refusal.stdout:
            raise RuntimeError("wrong independently expected tuple emitted plan evidence")
        (self.reports / "wrong-tuple-refusal.txt").write_text(refusal.stderr)
        report = json.loads(self.cli(*flags, binary=self.after, log_level=None).stdout)
        if manifest(source) != before:
            raise RuntimeError("offline preparation mutated source database")
        if report["height"] != self.source_tuple["height"] or report["app_hash"] != self.source_tuple["app_hash"]:
            raise RuntimeError("preparation output substituted the expected source tuple")
        self.plan_info = report["plan_info"]
        if hashlib.sha256(self.plan_info.encode()).hexdigest() != report["plan_info_sha256"]:
            raise RuntimeError("preparation plan digest mismatch")
        write_json(self.reports / "preparation.json", report)
        self.evidence["checks"].update({"offline_source_unchanged": True, "wrong_tuple_refused": True,
                                         "preparation_default_log_stdout_is_json": True})

    def schedule(self, prefix):
        self.start(self.before, prefix + "-predecessor")
        self.wait(lambda: self.height() >= self.source_tuple["height"])
        authority = self.query("upgrade", "authority")["address"]
        target = self.height() + 22
        proposal = {"messages": [{"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
                                  "authority": authority,
                                  "plan": {"name": PLAN, "height": str(target), "info": self.plan_info}}],
                    "metadata": "local " + PLAN + " rehearsal", "deposit": "1000000uzrn",
                    "title": "Local " + PLAN, "summary": "Private two binary test", "expedited": False}
        path = self.reports / (prefix + "-proposal.json")
        write_json(path, proposal)
        receipt = self.tx("gov", "submit-proposal", path, label=prefix + "-submit")
        proposal_id = None
        for event in receipt.get("events", []):
            for attribute in event.get("attributes", []):
                if attribute.get("key") == "proposal_id":
                    proposal_id = attribute["value"]
        if proposal_id is None:
            proposals = self.query("gov", "proposals")["proposals"]
            proposal_id = proposals[-1]["id"]
        self.tx("gov", "vote", proposal_id, "yes", label=prefix + "-vote")
        def scheduled():
            response = self.cli("query", "upgrade", "plan", "--node", self.rpc,
                                "--output", "json", check=False)
            if response.returncode:
                return False
            plan = json.loads(response.stdout).get("plan", {})
            return plan.get("name") == PLAN and int(plan.get("height", 0)) == target
        self.wait(scheduled)
        return target

    def halt(self, target, prefix):
        self.progress(prefix + ": waiting for real predecessor to halt before the unknown upgrade")
        upgrade_path = self.home / "data" / "upgrade-info.json"
        self.wait(lambda: upgrade_path.exists(), seconds=75)
        plan = json.loads(upgrade_path.read_text())
        if plan["name"] != PLAN or int(plan["height"]) != target or plan["info"] != self.plan_info:
            raise RuntimeError("predecessor wrote a different upgrade boundary")
        # Unknown handler leaves RPC live but consensus halted. Inspect committed
        # H-1; startup and handler independently recheck this boundary.
        observed = self.rpc_get("/abci_info")["response"]
        if int(observed["last_block_height"]) != target - 1:
            raise RuntimeError("predecessor did not halt at exact committed H-1")
        write_json(self.reports / (prefix + "-halt.json"),
                   {"height": target - 1,
                    "app_hash": base64.b64decode(observed["last_block_app_hash"]).hex(), "plan": plan})
        self.stop()

    def refuse_start(self, label, expected):
        self.start(self.after, label)
        deadline = time.monotonic() + 20
        while self.proc.poll() is None and time.monotonic() < deadline:
            time.sleep(0.1)
        code = self.proc.poll()
        self.stop()
        log = (self.root / (label + ".log")).read_text(errors="replace")
        if code in (None, 0) or expected not in log:
            raise RuntimeError(f"candidate did not produce expected {label} refusal")
        (self.reports / (label + "-refusal.txt")).write_text(log)

    def happy(self):
        self.progress("scheduling named accounting upgrade through SDK governance")
        target = self.schedule("happy")
        self.halt(target, "happy")
        self.progress("applying candidate at H and checking exact existing claims and custody")
        self.start(self.after, "happy-candidate")
        self.wait(lambda: self.height() >= target + 2)
        after = self.claims()
        if after != self.initial:
            raise RuntimeError("migration changed existing claim records, aggregates, or module custody")
        write_json(self.reports / "claims-after-migration.json", after)
        versions = self.query("upgrade", "module-versions")
        write_json(self.reports / "versions-after.json", versions)
        pairs = {entry["name"]: int(entry.get("version", 0)) for entry in versions["module_versions"]}
        if pairs.get(STAKING) != 2 or pairs.get(GOV) != 3:
            raise RuntimeError("migration did not install the exact custom version pair")
        applied = self.query("upgrade", "applied", PLAN)
        write_json(self.reports / "applied-upgrade.json", applied)
        if int(applied["height"]) != target:
            raise RuntimeError("applied receipt height mismatch")
        self.progress("checking repaired self undelegation, maturity payout, and candidate restart")
        receipt = self.tx(STAKING, "undelegate", self.addresses["validator"], "111000",
                          label="self-undelegate")
        pending = self.claims()
        self.assert_claims(pending, 222000, 777000, 1110000)
        write_json(self.reports / "claims-pending-unbonding.json", pending)
        self.wait(lambda: self.height() >= int(receipt["height"]) + 7)
        mature = self.claims()
        self.assert_claims(mature, 222000, 777000, 999000)
        write_json(self.reports / "claims-after-payout.json", mature)
        unbondings = self.query(STAKING, "unbondings", self.addresses["validator"])
        write_json(self.reports / "completed-unbonding.json", unbondings)
        restart_height = self.height()
        self.stop()
        self.start(self.after, "happy-restart")
        self.wait(lambda: self.height() >= restart_height + 2)
        if self.claims() != mature:
            raise RuntimeError("claim state changed across candidate restart")
        self.stop()
        self.evidence["checks"].update({"sdk_governance_scheduled": True,
             "old_halted_at_h_minus_1": True, "candidate_applied_at_h": True,
             "existing_claims_and_custody_preserved": True, "self_undelegation_consistent": True,
             "mature_unbonding_paid_once": True, "candidate_restart_passed": True})
        self.evidence["upgrade_height"] = target

    def negatives(self):
        self.progress("checking premature startup and exact-state drift refusals on isolated copies")
        self.home = self.root / "premature"
        shutil.copytree(self.snapshot, self.home)
        self.refuse_start("premature-candidate", "upgrade plan not found")
        self.evidence["checks"]["premature_startup_refused"] = True
        self.home = self.root / "drift"
        shutil.copytree(self.snapshot, self.home)
        target = self.schedule("drift")
        # A valid ordinary delegation changes real backing and claims after the
        # prepared commitment. Both are internally consistent yet stale Plan.Info
        # must refuse activation; no private database edits manufacture this case.
        self.tx(STAKING, "delegate", self.addresses["validator"], "1",
                signer="delegator", label="post-commitment-drift")
        self.halt(target, "drift")
        self.refuse_start("drift-candidate", "drifted from the exact plan commitment")
        self.evidence["checks"]["post_commitment_state_drift_refused"] = True

    def run(self):
        try:
            self.progress("creating isolated legacy native SDK-H3 chain")
            self.initialize()
            self.prepare()
            self.happy()
            self.negatives()
            self.evidence["result"] = "passed"
            write_json(self.reports / "summary.json", self.evidence)
            self.progress("PASS: local evidence at " + str(self.reports))
        except BaseException as exc:
            self.evidence.update({"result": "failed", "stage": self.stage, "error": str(exc)})
            write_json(self.reports / "summary.json", self.evidence)
            raise
        finally:
            self.stop()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--before", type=Path, required=True, help="real predecessor binary")
    parser.add_argument("--after", type=Path, required=True, help="candidate binary")
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
