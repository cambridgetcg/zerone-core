#!/usr/bin/env python3
"""Opt-in staged supervisor test with real SDK governance and an unchanged v10 client.

ZERONE_UPGRADE_PREDECESSOR_ROOT supplies the verified old checkout/package;
ZERONE_UPGRADE_TARGET_BINARY supplies a source-built successor. All homes and
accounts are fresh loopback fixtures controlled by this test. No public chain.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import select
import socket
import subprocess
import sys
import time
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[3]

def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value

shared = module("staged_shared_fixture", ROOT / "scripts/test_shared_claims_native.py")
runtime = module("staged_runtime", Path(__file__).with_name("runtime.py"))
BEFORE_ROOT = Path(os.environ.get("ZERONE_UPGRADE_PREDECESSOR_ROOT", "/absent"))
BEFORE = Path(os.environ.get("ZERONE_UPGRADE_PREDECESSOR_BINARY", str(BEFORE_ROOT / "build/zeroned")))
AFTER = Path(os.environ.get("ZERONE_UPGRADE_TARGET_BINARY", "/absent"))
OLD_RUNTIME = BEFORE_ROOT / "deploy/networks/zerone-dev-1/runtime.py"
shared.CLIENT = BEFORE_ROOT / "scripts/shared-claims.py"

@unittest.skipUnless(BEFORE.is_file() and AFTER.is_file(), "set predecessor root and target binary for staged native test")
class StagedNative(shared.NativeSharedClaimsTests):
    __unittest_skip__ = False
    # Do not inherit the unrelated complete five-home scenario.
    test_five_separate_participants_signed_rounds_gateway_and_restart = None

    def setUp(self):
        with mock.patch.dict(os.environ, {"ZERONE_SHARED_TEST_BINARY": str(AFTER)}), mock.patch.object(shared, "CLIENT", ROOT / "scripts/shared-claims.py"):
            super().setUp()
        self.evidence.update(schema="zerone-development-staged-upgrade-test/v1", network_scope="fresh loopback zerone-dev-1 only")
        self.packet_pin = None
        bound = [Path(__file__), Path(runtime.__file__), Path(runtime.__file__).with_name("gateway.py"), OLD_RUNTIME, OLD_RUNTIME.with_name("gateway.py"), BEFORE, AFTER, *[BEFORE_ROOT / "scripts" / name for name in ("shared-claims.py", "claim-workflow.py", "local-node.py")]]
        self.bound_sources = {str(p): runtime.digest(p) for p in bound}
        self.evidence["bound_sources_before"] = self.bound_sources
        if self.reports:
            self.reports = self.reports / "reports"
            self.reports.mkdir(mode=0o700)

    def source(self, binary):
        result = subprocess.run([str(binary), "version", "--long"], capture_output=True, text=True, check=True, timeout=30)
        return re.search(r"(?m)^commit:\s*([0-9a-f]{40})\s*$", result.stdout)[1]

    def start_mode(self, staged=False):
        path = runtime.__file__ if staged else OLD_RUNTIME
        args = [sys.executable, "-I", "-B", str(path), "run-upgrade" if staged else "run", "--home", str(self.node)]
        if staged:
            args += ["--upgrade-packet-sha256", self.packet_pin]
        else:
            args += ["--binary", str(BEFORE)]
        self.process = subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        return self.read_ready()

    def read_ready(self):
        ready, _, _ = select.select([self.process.stdout], [], [], 125)
        self.assertTrue(ready, "supervisor did not report ready")
        line = self.process.stdout.readline()
        if not line:
            self.fail("supervisor exited: " + self.process.stderr.read()[-2000:])
        value = json.loads(line)
        self.assertTrue(value["ready"])
        self.evidence.setdefault("supervisor_readiness", []).append(value)
        deadline = time.monotonic() + 15
        while True:
            try:
                descriptor = json.loads(self.get("/network.json"))
                if descriptor["source_commit"] == value["source_commit"]:
                    break
            except OSError:
                pass
            self.assertLess(time.monotonic(), deadline)
            self.assertIsNone(self.process.poll())
            time.sleep(.2)
        return value

    def query(self, *args):
        return self.command([AFTER, "query", *args, "--home", self.node, "--node", self.rpc_origin, "--output", "json"])

    def gov_tx(self, *args):
        response = self.command([BEFORE, "tx", "gov", *args, "--home", self.node,
            "--from", "validator", "--chain-id", "zerone-dev-1", "--node", self.rpc_origin,
            "--keyring-backend", "test", "--fees", "2000000uzrn", "--gas", "2000000", "--yes", "--output", "json"])
        self.assertEqual(int(response.get("code", 0)), 0)
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            result = subprocess.run([str(BEFORE), "query", "tx", response["txhash"], "--home", str(self.node), "--node", self.rpc_origin, "--output", "json"], capture_output=True, text=True, timeout=10)
            if result.returncode == 0:
                receipt = json.loads(result.stdout)
                self.assertGreater(int(receipt["height"]), 0)
                self.assertEqual(int(receipt.get("code", 0)), 0)
                self.evidence.setdefault("governance_transactions", []).append({k: receipt[k] for k in ("txhash", "height", "code")})
                return receipt
            time.sleep(.3)
        self.fail("governance transaction was not committed")

    def test_staged_upgrade_and_unchanged_participant(self):
        old = module("old_runtime", OLD_RUNTIME)
        ports = []
        for _ in range(3):
            with socket.socket() as sock:
                sock.bind(("127.0.0.1", 0)); ports.append(sock.getsockname()[1])
        self.origin = "http://127.0.0.1:" + str(ports[2])
        self.rpc_origin = "http://127.0.0.1:" + str(ports[0])
        original_check = old.check_genesis
        def fast_genesis(genesis):
            original_check(genesis)
            # Only this fresh synthetic genesis is accelerated, before its immutable
            # runtime manifest and faucet journal are created. No live state edit.
            genesis["app_state"]["gov"]["params"].update(voting_period="5s", expedited_voting_period="2s", min_deposit=[{"denom":"uzrn","amount":"1000000"}])
        args = argparse.Namespace(command="init", binary=BEFORE, source_commit=self.source(BEFORE),
            local_test=True, review_window_blocks=90, rpc_port=ports[0], p2p_port=ports[1], gateway_port=ports[2])
        with old.lock(self.node), mock.patch.object(old, "check_genesis", side_effect=fast_genesis):
            manifest = old.initialize(args, self.node)
        self.start_mode()
        raw = self.get("/network.json")
        descriptor_path = self.root / "network.json"; descriptor_path.write_bytes(raw)
        descriptor_sha = hashlib.sha256(raw).hexdigest()
        old_source, new_source = self.source(BEFORE), self.source(AFTER)
        self.evidence.update(predecessor_source=old_source, target_source=new_source,
            predecessor_binary_sha256=runtime.digest(BEFORE), target_binary_sha256=runtime.digest(AFTER))
        for person in ("author", *shared.REVIEWERS):
            result = self.client(person, "init", "--descriptor", descriptor_path, "--descriptor-sha256", descriptor_sha,
                "--binary", BEFORE, "--binary-sha256", runtime.digest(BEFORE), "--allow-loopback-test")
            self.addresses[person] = result["address"]
            self.client(person, "fund"); self.client(person, "onboard", "--type", "human")
        submitted = self.client("author", "submit", "--content", "Synthetic staged upgrade: 2 + 2 = 4.",
            "--method", "M-COMPUTATIONAL", "--reasoning", "Test-only arithmetic; not independent science.")
        claim = submitted["claim_id"]
        reasons = ["Checked integer addition in the local fixture: " + person for person in shared.REVIEWERS]
        scope = "All test accounts share one controller."
        before_round = self.commit(claim, ["accept"] * 3, reasons, scope, "fixture:integer-addition")
        self.assertFalse(self.history(claim)["record"]["claim"].get("funding_terms"))
        source_versions = {r["name"]: r.get("version", "0") for r in self.query("upgrade", "module-versions")["module_versions"]}
        height = int(self.query("knowledge", "claim-history", claim)["block_height"])
        target = height + 35
        self.assertLess(target + 5, int(before_round["commit_deadline"]))
        self.stop()
        original_files = {name: runtime.digest(self.node / name) for name in (*runtime.CONTROL, *runtime.IDENTITY, runtime.MARKER, "dev-faucet/state.json", "public/network.json")}
        participant_controls = {person: {str(path.relative_to(self.homes[person])): runtime.digest(path) for path in self.homes[person].rglob("*") if path.is_file() and (path.name.startswith(".zerone") or path.parent.name in ("keyring-test", "identities", "bin"))} for person in self.addresses}
        packet = {"schema":"zerone-development-upgrade/v1", "chain_id":"zerone-dev-1",
            "genesis_sha256":manifest["genesis_sha256"], "predecessor_descriptor_sha256":descriptor_sha,
            "plan":{"name":runtime.UPGRADE_NAME,"height":str(target),"info":""},
            "predecessor":{"knowledge_version":10,"source_commit":old_source,"binaries":{p:runtime.digest(BEFORE) for p in ("linux-amd64","darwin-arm64")}},
            "target":{"knowledge_version":11,"source_commit":new_source,"binaries":{p:runtime.digest(AFTER) for p in ("linux-amd64","darwin-arm64")}}}
        self.evidence["packet_scope"] = "Synthetic local platform; unused other-platform pin is not a distributable release assertion"
        path = self.root / "upgrade.json"; path.write_bytes(runtime.json_bytes(packet)); self.packet_pin = runtime.digest(path)
        self.command([sys.executable,"-I","-B",runtime.__file__,"stage-upgrade","--home",self.node,"--binary",AFTER,"--predecessor-binary",BEFORE,"--upgrade-packet",path,"--upgrade-packet-sha256",self.packet_pin])
        self.assertEqual(self.start_mode(True)["source_commit"], old_source)
        self.stop()
        self.assertEqual(self.start_mode(True)["source_commit"], old_source, "before plan restart must still use predecessor")
        authority = self.query("upgrade", "authority")["address"]
        proposal = {"messages":[{"@type":"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade","authority":authority,"plan":packet["plan"]}],
            "metadata":"local staged runtime test","deposit":"1000000uzrn","title":"Synthetic staged transition","summary":"Local test; no production authority","expedited":False}
        path = self.root / "proposal.json"; path.write_bytes(runtime.json_bytes(proposal))
        receipt = self.gov_tx("submit-proposal", path)
        proposal_id = next(a["value"] for e in receipt["events"] for a in e["attributes"] if a["key"] == "proposal_id")
        self.gov_tx("vote", proposal_id, "yes")
        self.assertEqual(self.read_ready()["source_commit"], new_source, "same supervisor must switch after actual upgrade halt")
        self.assertEqual(self.query("upgrade", "applied", runtime.UPGRADE_NAME)["height"], str(target))
        self.assertEqual({r["name"]:r.get("version", "0") for r in self.query("upgrade", "module-versions")["module_versions"]}, dict(source_versions, knowledge="11"))
        current = json.loads(self.get("/network.json")); self.assertEqual(current["knowledge_version"], 11)
        self.assertEqual((self.node / "public/network.json").read_bytes(), raw)
        self.assertNotEqual(self.get("/network.json"), raw)
        self.assertEqual(self.history(claim)["record"]["rounds"][0]["commits"], before_round["commits"])
        self.stop(); self.assertEqual(self.start_mode(True)["source_commit"], new_source, "applied restart must retain target")
        after = self.reveal(claim, ["accept"] * 3, reasons, scope, "fixture:integer-addition", 1)
        for reveal in after["reveals"]:
            self.assertGreaterEqual(int(reveal["revealed_at_block"]), target)
        self.evidence.update(upgrade_height=target, old_claim_before=before_round, old_claim_after=json.loads(self.get("/claims/" + claim)))
        # An old binary still signs the unchanged admission message. The target
        # gateway's current decoder exposes the prospective funding fields.
        new_claim = self.client("reviewer1", "submit", "--content", "Synthetic post-upgrade claim: 3 + 3 = 6.", "--method", "M-COMPUTATIONAL", "--reasoning", "Local compatibility check.")["claim_id"]
        current_history = json.loads(self.get("/claims/" + new_claim))
        self.assertEqual(current_history["record"]["claim"]["funding_terms"]["policy_version"], 1)
        self.evidence["new_claim_current_decoder"] = current_history
        self.evidence["new_claim_old_decoder"] = self.history(new_claim, "reviewer1")
        # A fresh follower uses the exact predecessor initializer on the immutable
        # genesis, then the same staged supervisor replays the real H boundary.
        full_home = self.root / "full-node"
        full_ports = []
        for _ in range(3):
            with socket.socket() as sock:
                sock.bind(("127.0.0.1", 0)); full_ports.append(sock.getsockname()[1])
        join_args = argparse.Namespace(command="join", binary=BEFORE, source_commit=old_source,
            local_test=True, review_window_blocks=None, rpc_port=full_ports[0], p2p_port=full_ports[1], gateway_port=full_ports[2],
            genesis=str(self.node / "config/genesis.json"), genesis_sha256=manifest["genesis_sha256"],
            peer=manifest["advertised_peer"], reference_rpc=self.rpc_origin)
        with old.lock(full_home):
            old.initialize(join_args, full_home)
        packet_path = self.root / "upgrade.json"
        self.command([sys.executable,"-I","-B",runtime.__file__,"stage-upgrade","--home",full_home,"--binary",AFTER,"--predecessor-binary",BEFORE,"--upgrade-packet",packet_path,"--upgrade-packet-sha256",self.packet_pin])
        full = subprocess.Popen([sys.executable,"-I","-B",runtime.__file__,"run-upgrade","--home",str(full_home),"--upgrade-packet-sha256",self.packet_pin],stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
        try:
            deadline = time.monotonic() + 150
            while time.monotonic() < deadline:
                readable, _, _ = select.select([full.stdout], [], [], 1)
                if not readable:
                    self.assertIsNone(full.poll()); continue
                line = full.stdout.readline()
                self.assertTrue(line, "follower exited before ready: " + (full.stderr.read()[-2000:] if full.poll() is not None else ""))
                ready = json.loads(line)
                if ready["source_commit"] == new_source:
                    break
            else:
                self.fail("follower did not replay the staged transition")
            observed = self.command([sys.executable,"-I","-B",runtime.__file__,"upgrade-status","--home",full_home,"--upgrade-packet-sha256",self.packet_pin])
            self.assertEqual(observed["role"], "full-node")
            self.assertEqual(observed["voting_power"], 0)
            self.assertEqual(observed["applied_height"], str(target))
            self.assertGreaterEqual(observed["height"], target)
            local_history = self.command([AFTER,"query","knowledge","claim-history",claim,"--home",full_home,"--node",f"http://127.0.0.1:{full_ports[0]}","--output","json"])
            self.assertEqual(local_history["record"], self.evidence["old_claim_after"]["record"])
            self.evidence.update(fresh_follower_staged_replay=observed, fresh_follower_history=local_history)
        finally:
            if full.poll() is None: full.terminate()
            _, error = full.communicate(timeout=35)
            self.assertEqual(full.returncode, 0, error[-2000:])
        self.stop()
        self.assertEqual({name:runtime.digest(self.node / name) for name in original_files}, original_files)
        for person, pins in participant_controls.items():
            self.assertEqual({name:runtime.digest(self.homes[person] / name) for name in pins}, pins)
        after_sources = {name:runtime.digest(Path(name)) for name in self.bound_sources}
        self.assertEqual(after_sources, self.bound_sources, "sources or binaries changed during the native run")
        self.evidence["bound_sources_after"] = after_sources
        self.evidence.update(result="PASS", original_runtime_controls_identity_genesis_faucet_preserved=True,
            old_participant_controls_identity_and_binary_preserved=True, old_client_precommit_postreveal=True,
            predecessor_restart=True, target_restart=True, same_supervisor_transition=True)

if __name__ == "__main__":
    unittest.main()
