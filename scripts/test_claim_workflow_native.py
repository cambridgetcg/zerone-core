#!/usr/bin/env python3
"""Opt-in signed local claim workflow, using disposable accounts of one operator.

ZERONE_LOCAL_NODE_TEST_BINARY=/absolute/zeroned python3 -m unittest discover \
    -s scripts -p 'test_claim_workflow_native.py' -v

Optional ZERONE_CLAIM_WORKFLOW_EVIDENCE_PARENT retains only curated public
observations in a new reports directory. Homes, keyrings, unrevealed preimages
and node logs are never copied into that directory. Nothing uses shared RPC.
"""
import base64
import hashlib
import json
import os
from pathlib import Path
import select
import signal
import socket
import subprocess
import sys
import tempfile
import time
import unittest
import urllib.request


ROOT = Path(__file__).resolve().parents[1]
LOCAL = ROOT / "scripts/local-node.py"
WORKFLOW = ROOT / "scripts/claim-workflow.py"
EXAMPLE = ROOT / "docs/examples/local-claims"
CLAIM = ("Synthetic deliberately overbroad assertion: for every integer n from 0 "
         "through 40 inclusive, n*n+n+41 is prime.")
COUNTER_CONTENT = ("Synthetic counterexample: at n=40, n*n+n+41 is 1681 = 41*41, "
                   "so the stated universal primality claim is false.")
REVIEW_SCOPE = ("Synthetic local fixture only: n=0 through 39 inclusive; n=40 was "
                "not checked. These accounts share one controller.")
REASONING = ("Deliberately incomplete synthetic review: trial division checks only "
             "n=0..39, although the assertion includes n=40. The workflow must retain "
             "that scope mismatch; acceptance is not proof of the assertion.")
COUNTER_REASON = "  The omitted endpoint is composite: 40*40+40+41 = 1681 = 41*41.\n"
SECOND_CLAIM = ("Synthetic nondecisive review exercise: for n from 0 through 39 "
                "inclusive, n*n+n+41 is prime; reviewers intentionally disagree for this test.")
ACTORS = ("reviewer1", "reviewer2", "reviewer3")


def review_reason(actor):
    return (f"Synthetic {actor}: trial division found primes for all checked inputs; "
            "this does not establish the broader range.")


def enum_is(value, number, name):
    return value in (number, str(number), name)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise RuntimeError("Local fixture RPC redirected")


@unittest.skipUnless(os.environ.get("ZERONE_LOCAL_NODE_TEST_BINARY"),
                     "set ZERONE_LOCAL_NODE_TEST_BINARY for real signed local workflow")
class NativeClaimWorkflowTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="zerone-local-claims-")
        self.home = Path(self.temporary.name).resolve() / "home"
        self.process = None
        self.observations = {}
        self.tooling_hashes = {str(path.relative_to(ROOT)): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in (LOCAL, WORKFLOW, Path(__file__), EXAMPLE / "prime-polynomial.py",
                         EXAMPLE / "scoped-review.json", EXAMPLE / "counterexample.json")}
        self.observations["tooling_sha256_before"] = self.tooling_hashes
        self.txhashes = set()
        self.receipts = {}
        self.reports = None
        parent = os.environ.get("ZERONE_CLAIM_WORKFLOW_EVIDENCE_PARENT")
        if parent:
            parent = Path(parent).expanduser()
            self.assertTrue(parent.is_absolute() and parent.is_dir() and not parent.is_symlink())
            self.assertEqual(parent.stat().st_uid, os.getuid())
            self.reports = Path(tempfile.mkdtemp(prefix="claim-workflow-", dir=parent)) / "reports"
            self.reports.mkdir(mode=0o700)
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())

    def tearDown(self):
        # Let the helper stop its own child. Do not delete a home if that failed.
        self.stop_node()
        if self.reports:
            self.save("observations.json", self.observations)
        self.temporary.cleanup()

    def save(self, name, value):
        if self.reports:
            path = self.reports / name
            path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")
            path.chmod(0o600)

    def json_command(self, command, timeout=90):
        env = {k: v for k, v in os.environ.items() if not k.upper().startswith("ZERONED_")}
        result = subprocess.run([str(c) for c in command], capture_output=True,
                                text=True, timeout=timeout, env=env)
        self.assertEqual(result.returncode, 0, result.stderr[-2000:])
        return json.loads(result.stdout)

    def workflow(self, command, *args):
        result = self.json_command([sys.executable, WORKFLOW, command, "--home", self.home, *args])
        self.collect_txhashes(result)
        return result

    def collect_txhashes(self, value):
        if isinstance(value, dict):
            digest = value.get("txhash")
            if digest is not None:
                self.assertEqual(len(digest), 64)
                bytes.fromhex(digest)
                self.txhashes.add(digest.upper())
                if "height" in value:
                    self.assertGreater(int(value["height"]), 0, "mempool acceptance is not commitment")
                if "code" in value:
                    self.assertEqual(int(value["code"]), 0)
                self.receipts[digest.upper()] = {key: value[key] for key in
                    ("action", "actor", "status", "txhash", "height", "code", "gas_used", "claim_id", "round_id")
                    if key in value}
            for child in value.values():
                self.collect_txhashes(child)
        elif isinstance(value, list):
            for child in value:
                self.collect_txhashes(child)

    def rpc(self, path):
        with self.opener.open(self.info["rpc"] + path, timeout=10) as response:
            raw = response.read(8 * 1024 * 1024 + 1)
        self.assertLessEqual(len(raw), 8 * 1024 * 1024)
        value = json.loads(raw)
        self.assertNotIn("error", value)
        return value["result"]

    def height(self):
        status = self.rpc("/status")
        self.assertEqual(status["node_info"]["network"], self.info["chain_id"])
        return int(status["sync_info"]["latest_block_height"])

    def history(self, claim_id, height=None):
        if height is None:
            value = self.workflow("history", "--claim", claim_id)
        else:
            value = self.json_command([self.home / "bin/zeroned", "query", "knowledge", "claim-history",
                claim_id, "--height", str(height), "--home", self.home, "--node", self.info["rpc"],
                "--output", "json"])
        self.assertEqual(value["chain_id"], self.info["chain_id"])
        self.assertGreater(int(value["block_height"]), 0)
        if height is not None:
            self.assertEqual(int(value["block_height"]), height)
        self.assertEqual(value["record"]["claim_id"], claim_id)
        self.assertEqual(value["record"]["claim"]["id"], claim_id)
        self.assertFalse(value["record"].get("missing_round_ids"))
        return value

    def one_round(self, claim_id):
        rounds = self.history(claim_id)["record"]["rounds"]
        self.assertEqual(len(rounds), 1)
        self.assertEqual(rounds[0]["claim_id"], claim_id)
        return rounds[0]

    def wait_phase(self, claim_id, number, name, seconds=150):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            value = self.one_round(claim_id)
            if enum_is(value["phase"], number, name):
                return value
            self.assertIsNone(self.process.poll(), "owned local helper exited")
            time.sleep(0.7)
        self.fail(f"Round {claim_id} did not enter {name}; last phase {value['phase']}")

    def start_node(self):
        self.process = subprocess.Popen([sys.executable, str(LOCAL), "start", "--home", str(self.home)],
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        readable, _, _ = select.select([self.process.stdout], [], [], 55)
        self.assertTrue(readable, "local helper readiness timed out")
        line = self.process.stdout.readline()
        self.assertTrue(line, "local helper exited before readiness")
        ready = json.loads(line)
        self.assertTrue(ready["ready"] and ready["advancing"])
        self.assertEqual(ready["chain_id"], self.info["chain_id"])
        return ready

    def stop_node(self):
        if self.process is None:
            return
        process = self.process
        if process.poll() is None:
            process.send_signal(signal.SIGINT)
        output, error = process.communicate(timeout=30)
        self.assertEqual(process.returncode, 0, error[-2000:])
        self.assertTrue(json.loads(output.splitlines()[-1])["state_preserved"])
        self.process = None

    def commit_reviews(self, claim_id, votes, reasons, scope, evidence, method=""):
        for actor, vote, reason in zip(ACTORS, votes, reasons):
            self.workflow("commit", "--claim", claim_id, "--actor", actor, "--vote", vote,
                          "--reason", reason, "--scope", scope, "--evidence", evidence,
                          "--confidence", "800000", *(["--method", method] if method else []))
        current = self.one_round(claim_id)
        self.assertEqual(int(current["commitment_scheme"]), 2)
        self.assertEqual(int(current["review_policy_version"]), 1)
        self.assertEqual(current["commitment_chain_id"], self.info["chain_id"])
        self.assertEqual({c["verifier"] for c in current["commits"]},
                         {self.info["accounts"][actor] for actor in ACTORS})
        self.assertEqual(len(current["commits"]), 3)
        return current

    def reveal_reviews(self, claim_id, votes, reasons, scope, evidence, verdict, verdict_name, method=""):
        self.wait_phase(claim_id, 2, "VERIFICATION_PHASE_REVEAL")
        for actor in ACTORS:
            self.workflow("reveal", "--claim", claim_id, "--actor", actor)
        completed = self.wait_phase(claim_id, 4, "VERIFICATION_PHASE_COMPLETE", seconds=30)
        self.assertTrue(enum_is(completed["verdict"], verdict, verdict_name))
        self.assertEqual(len(completed["reveals"]), 3)
        for actor, vote, reason in zip(ACTORS, votes, reasons):
            matches = [r for r in completed["reveals"] if r["verifier"] == self.info["accounts"][actor]]
            self.assertEqual(len(matches), 1)
            reveal = matches[0]
            self.assertEqual(reveal["vote"], vote)
            self.assertEqual(int(reveal["confidence"]), 800000)
            self.assertGreaterEqual(len(base64.b64decode(reveal["salt"], validate=True)), 16)
            att = reveal["attestation"]
            self.assertEqual(att["reason"], reason)
            self.assertEqual(att["scope"], scope)
            self.assertEqual(att["evidence_ids"], [evidence])
            self.assertEqual(att.get("method_id", ""), method)
        return completed

    def test_signed_scoped_review_counterexample_inconclusive_and_restart(self):
        evidence_ids = {}
        for kind, filename in (("review", "scoped-review.json"), ("counterexample", "counterexample.json")):
            result = subprocess.run([sys.executable, str(EXAMPLE / "prime-polynomial.py"), kind],
                                    capture_output=True, check=True, timeout=10)
            self.assertEqual(result.stdout, (EXAMPLE / filename).read_bytes())
            evidence_ids[kind] = "sha256:" + hashlib.sha256(result.stdout).hexdigest()
            self.observations[kind + "_evidence"] = json.loads(result.stdout)
        checked = self.observations["review_evidence"]
        self.assertEqual(checked["range_inclusive"], [0, 39])
        self.assertEqual([r["n"] for r in checked["rows"]], list(range(40)))
        self.assertTrue(all(r["prime"] and r["value"] == r["n"] ** 2 + r["n"] + 41 for r in checked["rows"]))
        counter = self.observations["counterexample_evidence"]
        self.assertEqual((counter["n"], counter["value"], counter["factors"], counter["prime"]),
                         (40, 1681, [41, 41], False))
        with socket.socket() as rpc_socket, socket.socket() as p2p_socket:
            rpc_socket.bind(("127.0.0.1", 0))
            p2p_socket.bind(("127.0.0.1", 0))
            rpc_port, p2p_port = rpc_socket.getsockname()[1], p2p_socket.getsockname()[1]
        self.info = self.json_command([sys.executable, LOCAL, "init", "--home", self.home,
            "--binary", os.environ["ZERONE_LOCAL_NODE_TEST_BINARY"], "--knowledge-profile",
            "--fast-review",
            "--chain-id", "zerone-local-claims-test", "--rpc-port", str(rpc_port), "--p2p-port", str(p2p_port)])
        self.assertEqual(self.info["knowledge_profile"], "claims-v1")
        genesis = json.loads((self.home / "config/genesis.json").read_text())
        params = genesis["app_state"]["knowledge"]["params"]
        self.assertEqual([int(params[k]) for k in ("commit_phase_blocks", "reveal_phase_blocks", "aggregation_phase_blocks")], [60, 60, 5])
        self.assertEqual((int(params["min_verifiers"]), int(params["confidence_threshold"])), (3, 770000))
        self.start_node()
        self.workflow("onboard")
        submitted = self.workflow("submit", "--content", CLAIM, "--method", "M-COMPUTATIONAL",
                                  "--reasoning", REASONING)
        accepted_id = submitted["claim_id"]
        pending = self.history(accepted_id)
        claim = pending["record"]["claim"]
        self.assertEqual(claim["fact_content"], CLAIM)
        self.assertEqual(claim["reasoning_trace"], REASONING)
        self.assertEqual(claim["method_id"], "M-COMPUTATIONAL")
        self.assertTrue(enum_is(claim["status"], 5, "CLAIM_STATUS_IN_VERIFICATION"))
        self.assertFalse(pending["record"].get("facts"))
        self.observations["pending"] = pending
        reasons = [review_reason(actor) for actor in ACTORS]
        commits = self.commit_reviews(accepted_id, ["accept"] * 3, reasons, REVIEW_SCOPE,
                                      evidence_ids["review"], method="M-COMPUTATIONAL")
        snapshot = self.workflow("snapshot")
        self.assertEqual(len(snapshot["pending_reviews"]), 3)
        self.assertTrue(all(r["status"] == "awaiting-reveal" for r in snapshot["pending_reviews"]))
        self.assertTrue(all(not {"salt", "vote", "reason", "attestation"} & row.keys()
                            for row in snapshot["pending_reviews"]))
        self.observations["before_commit_phase_restart"] = commits
        before_height = self.height()
        self.stop_node()
        self.start_node()
        self.assertGreater(self.height(), before_height)
        self.assertEqual(self.one_round(accepted_id), commits, "restart changed retained COMMIT state")
        self.reveal_reviews(accepted_id, ["accept"] * 3, reasons, REVIEW_SCOPE, evidence_ids["review"],
                            1, "VERDICT_ACCEPT", method="M-COMPUTATIONAL")
        accepted = self.history(accepted_id)
        self.assertTrue(enum_is(accepted["record"]["claim"]["status"], 6, "CLAIM_STATUS_ACCEPTED"))
        self.assertEqual(len(accepted["record"]["facts"]), 1)
        original_fact = accepted["record"]["facts"][0]["fact"]
        self.assertEqual(original_fact["claim_id"], accepted_id)
        self.assertEqual(original_fact["content"], CLAIM)
        self.assertEqual(original_fact["reasoning_trace"], REASONING)
        self.observations["accepted_before_contradiction"] = accepted
        # First completion occurs after the 50-block ordinary submission cooldown.
        nondecisive = self.workflow("submit", "--content", SECOND_CLAIM, "--method", "M-COMPUTATIONAL",
                                   "--reasoning", "Synthetic signed disagreement about the actual checked n=0..39 fixture; one reject is intentional.")
        second_id = nondecisive["claim_id"]
        challenged = self.workflow("challenge", "--fact", original_fact["id"], "--content", COUNTER_CONTENT,
                                   "--reason", COUNTER_REASON, "--evidence", evidence_ids["counterexample"])
        counter_id = challenged["claim_id"]
        pending_counter = self.history(counter_id)["record"]["claim"]
        self.assertEqual(pending_counter["argument_text"], COUNTER_REASON)
        self.assertEqual(pending_counter["evidence_ids"], [evidence_ids["counterexample"]])
        self.assertEqual(pending_counter["fact_content"], COUNTER_CONTENT)
        linked = self.history(accepted_id)
        related = [r for r in linked["related_claims"] if r["record"]["claim_id"] == counter_id]
        self.assertEqual(len(related), 1)
        self.assertIn({"field": "relations.contradicts", "target_id": original_fact["id"]}, related[0]["links"])
        self.assertTrue(enum_is(linked["record"]["facts"][0]["fact"]["status"], 5, "FACT_STATUS_CONTESTED"))
        second_reasons = [f"Synthetic {actor}: intentional {vote} for the nondecisive fixture."
                          for actor, vote in zip(ACTORS, ("accept", "accept", "reject"))]
        second_scope = "Checked n=0..39 only; same-controller synthetic disagreement, not independent scientific verification."
        counter_reasons = [f"Synthetic {actor}: directly checked 1681 = 41*41 at the omitted n=40 endpoint." for actor in ACTORS]
        counter_scope = "One explicit n=40 counterexample; these reviewer accounts share one controller."
        self.commit_reviews(second_id, ["accept", "accept", "reject"], second_reasons, second_scope, evidence_ids["review"])
        self.commit_reviews(counter_id, ["accept"] * 3, counter_reasons, counter_scope, evidence_ids["counterexample"])
        self.reveal_reviews(second_id, ["accept", "accept", "reject"], second_reasons, second_scope, evidence_ids["review"], 3, "VERDICT_INCONCLUSIVE")
        self.reveal_reviews(counter_id, ["accept"] * 3, counter_reasons, counter_scope, evidence_ids["counterexample"], 1, "VERDICT_ACCEPT")
        second = self.history(second_id)
        self.assertTrue(enum_is(second["record"]["claim"]["status"], 10, "CLAIM_STATUS_INSUFFICIENT"))
        self.assertFalse(second["record"].get("facts"))
        final = self.history(accepted_id)
        related = next(r for r in final["related_claims"] if r["record"]["claim_id"] == counter_id)
        self.assertEqual(len(related["record"]["facts"]), 1)
        counter_fact = related["record"]["facts"][0]
        self.assertEqual(counter_fact["fact"]["content"], COUNTER_CONTENT)
        edge = next(e for e in counter_fact["outgoing_relations"] if e["target_fact_id"] == original_fact["id"])
        self.assertTrue(enum_is(edge["relation"], 2, "RELATION_TYPE_CONTRADICTS"))
        self.assertEqual(edge["source_fact_id"], counter_fact["fact"]["id"])
        self.assertEqual(edge["creator"], self.info["accounts"]["challenger"])
        self.assertTrue(enum_is(final["record"]["facts"][0]["fact"]["status"], 5, "FACT_STATUS_CONTESTED"))
        self.assertTrue(final["record"]["facts"][0].get("status_transitions"))
        self.observations.update({"final_root_history": final, "inconclusive_history": second})
        txs = []
        for digest in sorted(self.txhashes):
            tx = self.rpc("/tx?hash=0x" + digest + "&prove=false")
            raw = base64.b64decode(tx["tx"], validate=True)
            self.assertEqual(hashlib.sha256(raw).hexdigest().upper(), digest)
            self.assertGreater(int(tx["height"]), 0)
            self.assertEqual(int(tx["tx_result"]["code"]), 0)
            txs.append({"txhash": digest, "height": tx["height"], "signed_tx_bytes": len(raw),
                        "gas_used": tx["tx_result"]["gas_used"], "code": 0})
        self.assertEqual(len(txs), 26, "expected five onboard, three submissions, nine commits and nine reveals")
        snapshot = self.workflow("snapshot")
        self.assertEqual({row["claim_id"] for row in snapshot["claims"]}, {accepted_id, second_id, counter_id})
        self.assertEqual(len(snapshot["pending_reviews"]), 9)
        self.assertTrue(all(row["status"] == "revealed" for row in snapshot["pending_reviews"]))
        pinned_height = int(final["block_height"])
        pinned = self.history(accepted_id, pinned_height)
        self.stop_node()
        restarted = self.start_node()
        self.assertGreater(int(restarted["height"]), pinned_height)
        self.assertEqual(self.history(accepted_id, pinned_height), pinned)
        after = self.history(accepted_id)
        self.assertEqual(after["record"]["claim"], final["record"]["claim"])
        self.assertEqual(after["record"]["rounds"], final["record"]["rounds"])
        after_related = next(row for row in after["related_claims"] if row["record"]["claim_id"] == counter_id)
        self.assertEqual(after_related["record"]["claim"], related["record"]["claim"])
        self.assertEqual(after_related["record"]["rounds"], related["record"]["rounds"])
        self.assertEqual(self.history(second_id)["record"], second["record"])
        self.stop_node()
        for relative, digest in self.tooling_hashes.items():
            self.assertEqual(hashlib.sha256((ROOT / relative).read_bytes()).hexdigest(), digest,
                             "tooling changed during native validation: " + relative)
        self.observations.update({"result": "PASS", "synthetic_accounts_one_controller": True,
            "scientific_independence_claimed": False, "shared_network": False,
            "chain_id": self.info["chain_id"], "evidence_ids": evidence_ids,
            "tx_sizes": txs, "tx_receipts": list(self.receipts.values()),
            "historical_height_after_restart": pinned_height,
            "genesis_sha256": hashlib.sha256((self.home / "config/genesis.json").read_bytes()).hexdigest(),
            "binary_sha256": hashlib.sha256((self.home / "bin/zeroned").read_bytes()).hexdigest(),
            "script_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest()})


if __name__ == "__main__":
    unittest.main()
