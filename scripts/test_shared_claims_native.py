#!/usr/bin/env python3
"""Opt-in actual signed five-home development workflow through the real gateway.

Set ZERONE_SHARED_TEST_BINARY to a current native zeroned. The chain is an
isolated --local-test fixture with accelerated review windows. Five separate key
homes are controlled by this test; no human/reviewer independence is asserted.
Optional ZERONE_SHARED_EVIDENCE_PARENT retains curated public observations only.
"""
import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import select
import signal
import socket
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock
import urllib.request
import urllib.error

ROOT = Path(__file__).resolve().parents[1]
CLIENT = ROOT / "scripts/shared-claims.py"
RUNTIME = ROOT / "deploy/networks/zerone-dev-1/runtime.py"
GATEWAY = RUNTIME.with_name("gateway.py")
SPEC = importlib.util.spec_from_file_location("shared_fixture", Path(__file__).with_name("test_claim_workflow_native.py"))
fixture = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(fixture)
PEOPLE = ("author", "reviewer1", "reviewer2", "reviewer3", "challenger")
REVIEWERS = PEOPLE[1:4]


@unittest.skipUnless(os.environ.get("ZERONE_SHARED_TEST_BINARY"), "set ZERONE_SHARED_TEST_BINARY for actual separate-home signing")
class NativeSharedClaimsTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="zerone-shared-claims-")
        self.root = Path(self.temporary.name).resolve()
        self.node = self.root / "node"
        self.homes = {name: self.root / name for name in PEOPLE}
        self.binary = Path(os.environ["ZERONE_SHARED_TEST_BINARY"]).resolve()
        self.binary_sha = hashlib.sha256(self.binary.read_bytes()).hexdigest()
        self.process = None
        self.addresses = {}
        self.txs = {}
        self.evidence = {"synthetic_accounts_one_controller": True, "independent_reviewers_claimed": False,
                         "network_scope": "isolated local-test zerone-dev-1 runtime; all participant traffic uses actual gateway"}
        self.reports = None
        parent = os.environ.get("ZERONE_SHARED_EVIDENCE_PARENT")
        if parent:
            parent = Path(parent).expanduser()
            self.assertTrue(parent.is_absolute() and parent.is_dir() and not parent.is_symlink())
            self.reports = Path(tempfile.mkdtemp(prefix="shared-claims-", dir=parent))
            self.reports.chmod(0o700)
        files = (CLIENT, CLIENT.with_name("claim-workflow.py"), CLIENT.with_name("local-node.py"), RUNTIME, GATEWAY,
                 Path(__file__), fixture.EXAMPLE / "prime-polynomial.py", fixture.EXAMPLE / "scoped-review.json", fixture.EXAMPLE / "counterexample.json")
        self.source_hashes = {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
        self.evidence["source_hashes_before"] = self.source_hashes
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), fixture.NoRedirect())

    def tearDown(self):
        try:
            self.stop()
        finally:
            if self.reports:
                path = self.reports / "observations.json"
                path.write_text(json.dumps(self.evidence, indent=2, sort_keys=True) + "\n")
                path.chmod(0o600)
        self.temporary.cleanup()

    def command(self, args, timeout=120, failure_context=None):
        result = subprocess.run(list(map(str, args)), capture_output=True, text=True, timeout=timeout,
            env={k: v for k, v in os.environ.items() if not k.upper().startswith("ZERONED_")})
        if result.returncode and failure_context is not None:
            self.capture_client_failure(failure_context, result)
        self.assertEqual(result.returncode, 0, result.stderr[-2000:])
        return json.loads(result.stdout)

    def client(self, person, command, *args):
        value = self.command([sys.executable, "-I", "-B", CLIENT, command, "--home", self.homes[person], *args],
                             failure_context={"participant": person, "action": command})
        if "txhash" in value:
            self.assertEqual(value["status"], "committed")
            self.assertGreater(int(value["height"]), 0)
            self.assertEqual(int(value["code"]), 0)
            self.txs[value["txhash"]] = value
        return value

    def capture_client_failure(self, context, result):
        """After refusal only: retain selected public execution fields.

        Diagnostic failures must not mask the original assertion. Query H-1/H
        labels are actual committed contexts, not a reconstruction of intra-H
        BeginBlock state. The execution error itself remains authoritative.
        """
        diagnostic = {**context, "exit_code": result.returncode, "queries_after_failure_only": True,
                      "private_keys_or_workflow_journals_copied": False,
                      "transaction_body_retained": False}
        self.evidence.setdefault("client_failures", []).append(diagnostic)
        try:
            matches = set(re.findall(r"(?i)transaction ([0-9a-f]{64})\b", result.stderr[-8192:]))
            if len(matches) != 1:
                diagnostic["capture_status"] = "no_unique_public_transaction_hash_in_client_error"
                return
            txhash = matches.pop().upper()
            diagnostic["txhash"] = txhash
            tx = self.rpc("tx", hash=base64.b64encode(bytes.fromhex(txhash)).decode(), prove=False)
            raw = base64.b64decode(tx["tx"], validate=True)
            if len(raw) > 256 * 1024 or hashlib.sha256(raw).hexdigest().upper() != txhash or tx["hash"].upper() != txhash:
                raise ValueError("Public transaction hash/size mismatch")
            height = int(tx["height"])
            if height < 1 or len(json.dumps(tx).encode()) > 1024 * 1024:
                raise ValueError("Public execution context exceeds diagnostic bound")
            execution = tx["tx_result"]
            diagnostic.update(capture_status="public_execution_observed", height=height,
                              signed_tx_bytes=len(raw), signed_tx_sha256=hashlib.sha256(raw).hexdigest(),
                              execution={"code": int(execution["code"]),
                                         "codespace": str(execution.get("codespace", ""))[:128],
                                         "raw_log": str(execution.get("log", ""))[:4096],
                                         "raw_log_truncated": len(str(execution.get("log", ""))) > 4096})
            prior = self.evidence.get("accepted_before_counterclaim", {}).get("record", {}).get("claim", {})
            diagnostic["previous_accepted_claim"] = {key: prior[key] for key in ("id", "submitter", "submitted_at_block", "domain") if key in prior}
            diagnostic["admission_context"] = []
            for query_height in sorted({max(1, height - 1), height}):
                for label, arguments in (("params", ["knowledge", "params"]),
                                         ("pacing", ["alignment", "global-pacing"]),
                                         ("domain_capacity", ["knowledge", "domain-capacity", "physics"])):
                    row = {"height": query_height, "query": label}
                    diagnostic["admission_context"].append(row)
                    try:
                        # Fixed read commands only; never sign, send, retry, or
                        # directly inspect keyring/identity/attempt files. The
                        # query CLI may read its ordinary client configuration.
                        observed = subprocess.run([str(self.homes[context["participant"]] / "bin/zeroned"), "query", *arguments,
                            "--home", str(self.homes[context["participant"]]), "--node", self.origin,
                            "--height", str(query_height), "--output", "json"],
                            capture_output=True, text=True, timeout=15,
                            env={k: v for k, v in os.environ.items() if not k.upper().startswith("ZERONED_")})
                        row["exit_code"] = observed.returncode
                        if observed.returncode == 0 and len(observed.stdout.encode()) <= 65536:
                            response = json.loads(observed.stdout)
                            fields = {"params": ("claim_cooldown_blocks", "min_review_fee", "commit_phase_blocks", "reveal_phase_blocks"),
                                      "pacing": ("health_category", "creation_multiplier_bps", "analysis_multiplier_bps"),
                                      "domain_capacity": ("domain", "active_count", "at_risk_count", "capacity", "pressure_bps", "category")}[label]
                            if label == "params":
                                response = response["params"]
                            row["response"] = {key: response[key] for key in fields
                                               if key in response and isinstance(response[key], (str, int, bool))}
                        else:
                            row["capture_status"] = "query_failed_or_exceeded_bound"
                    except Exception as error:
                        row["capture_error_type"] = type(error).__name__
        except Exception as error:
            diagnostic["capture_error_type"] = type(error).__name__
        finally:
            if self.reports:
                try:
                    path = self.reports / ("client-failure-" + str(len(self.evidence["client_failures"])) + ".json")
                    payload = (json.dumps(diagnostic, indent=2, sort_keys=True) + "\n").encode()
                    if len(payload) > 2 * 1024 * 1024:
                        raise ValueError("Combined public diagnostic exceeds 2 MiB")
                    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
                    with os.fdopen(fd, "wb") as stream:
                        stream.write(payload)
                        stream.flush()
                        os.fsync(stream.fileno())
                except Exception as error:
                    diagnostic["persistence_error_type"] = type(error).__name__

    def get(self, path):
        with self.opener.open(self.origin + path, timeout=25) as response:
            raw = response.read(8 * 1024 * 1024 + 1)
        self.assertLessEqual(len(raw), 8 * 1024 * 1024)
        return raw

    def rpc(self, method, **params):
        data = json.dumps({"jsonrpc": "2.0", "id": 1, "method": method, "params": params}).encode()
        self.assertEqual(method, "tx", "this bounded read retry must never cover a broadcast or faucet")
        time.sleep(0.1)  # Below the real gateway's 40 requests/second bound.
        for attempt in range(5):
            try:
                with self.opener.open(urllib.request.Request(self.origin, data=data, headers={"Content-Type": "application/json"}), timeout=25) as response:
                    raw = response.read(8 * 1024 * 1024 + 1)
                break
            except urllib.error.HTTPError as error:
                code = error.code
                error.close()
                if code != 429 or attempt == 4:
                    raise
                self.evidence["read_rate_limit_retries"] = self.evidence.get("read_rate_limit_retries", 0) + 1
                time.sleep(1.1)
        self.assertLessEqual(len(raw), 8 * 1024 * 1024)
        value = json.loads(raw)
        self.assertNotIn("error", value)
        return value["result"]

    def start(self):
        self.process = subprocess.Popen([sys.executable, "-I", "-B", str(RUNTIME), "run", "--home", str(self.node), "--binary", str(self.binary)],
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        readable, _, _ = select.select([self.process.stdout], [], [], 125)
        self.assertTrue(readable, "runtime did not report readiness")
        line = self.process.stdout.readline()
        if not line:
            self.fail("runtime exited before readiness; " + self.process.stderr.read()[-2000:])
        value = json.loads(line)
        self.assertTrue(value["ready"])
        self.assertEqual(value["chain_id"], "zerone-dev-1")
        deadline = time.monotonic() + 15
        while True:
            try:
                self.get("/network.json")
                break
            except OSError:
                self.assertLess(time.monotonic(), deadline, "gateway did not become ready")
                self.assertIsNone(self.process.poll())
                time.sleep(0.25)
        return value

    def stop(self):
        if self.process is None:
            return
        if self.process.poll() is None:
            self.process.send_signal(signal.SIGTERM)
        _, error = self.process.communicate(timeout=45)
        self.assertEqual(self.process.returncode, 0, error[-2000:])
        self.process = None

    def history(self, claim, person="author"):
        value = self.client(person, "history", "--claim", claim)
        self.assertEqual(value["chain_id"], "zerone-dev-1")
        self.assertEqual(value["record"]["claim_id"], claim)
        self.assertFalse(value["record"].get("missing_round_ids"))
        return value

    def wait_phase(self, claim, phase, seconds=160):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            # Public read route avoids any participant signing/key access while waiting.
            value = json.loads(self.get("/claims/" + claim))
            row = value["record"]["rounds"][0]
            if fixture.enum_is(row["phase"], phase, ("", "VERIFICATION_PHASE_COMMIT", "VERIFICATION_PHASE_REVEAL", "VERIFICATION_PHASE_AGGREGATION", "VERIFICATION_PHASE_COMPLETE")[phase]):
                return row
            self.assertIsNone(self.process.poll())
            time.sleep(0.8)
        self.fail("Expected round phase was not observed within the isolated test window")

    def commit(self, claim, votes, reasons, scope, evidence):
        for person, vote, reason in zip(REVIEWERS, votes, reasons):
            self.client(person, "commit", "--claim", claim, "--vote", vote, "--reason", reason,
                        "--scope", scope, "--evidence", evidence, "--method", "M-COMPUTATIONAL")
        row = self.history(claim)["record"]["rounds"][0]
        self.assertEqual({r["verifier"] for r in row["commits"]}, {self.addresses[p] for p in REVIEWERS})
        self.assertEqual(len(row["commits"]), 3)
        self.assertEqual((int(row["commitment_scheme"]), int(row["review_policy_version"]), row["commitment_chain_id"]), (2, 1, "zerone-dev-1"))
        return row

    def reveal(self, claim, votes, reasons, scope, evidence, verdict):
        self.wait_phase(claim, 2)
        for person in REVIEWERS:
            self.client(person, "reveal", "--claim", claim)
        row = self.wait_phase(claim, 4, 35)
        self.assertTrue(fixture.enum_is(row["verdict"], verdict, {1: "VERDICT_ACCEPT", 3: "VERDICT_INCONCLUSIVE"}[verdict]))
        self.assertEqual(len(row["reveals"]), 3)
        for person, vote, reason in zip(REVIEWERS, votes, reasons):
            reveal = next(r for r in row["reveals"] if r["verifier"] == self.addresses[person])
            self.assertEqual(reveal["vote"], vote)
            self.assertEqual(int(reveal["confidence"]), 800000)
            self.assertEqual(reveal["attestation"], {"reason": reason, "scope": scope, "evidence_ids": [evidence], "method_id": "M-COMPUTATIONAL"})
            self.assertEqual(len(base64.b64decode(reveal["salt"], validate=True)), 32)
        return row

    def test_five_separate_participants_signed_rounds_gateway_and_restart(self):
        version = subprocess.run([str(self.binary), "version", "--long"], capture_output=True, text=True, check=True, timeout=30).stdout
        source = re.search(r"(?m)^commit:\s*([0-9a-f]{40})\s*$", version)
        self.assertIsNotNone(source)
        self.evidence.update(binary_sha256=self.binary_sha, binary_source_commit=source[1])
        ports = []
        sockets = []
        try:
            for _ in range(3):
                sock = socket.socket()
                sockets.append(sock)
                sock.bind(("127.0.0.1", 0))
                ports.append(sock.getsockname()[1])
        finally:
            for sock in sockets:
                sock.close()
        self.origin = "http://127.0.0.1:" + str(ports[2])
        self.command([sys.executable, "-I", "-B", RUNTIME, "init", "--home", self.node, "--binary", self.binary,
            "--source-commit", source[1], "--local-test", "--review-window-blocks", "60", "--rpc-port", ports[0],
            "--p2p-port", ports[1], "--gateway-port", ports[2]])
        self.start()
        raw = self.get("/network.json")
        descriptor_path = self.root / "network.json"
        descriptor_path.write_bytes(raw)
        descriptor_sha = hashlib.sha256(raw).hexdigest()
        descriptor = json.loads(raw)
        self.assertTrue(descriptor["local_test"])
        self.assertEqual(descriptor["review_window_blocks"], 60)
        self.evidence.update(descriptor=descriptor, descriptor_sha256=descriptor_sha)
        # In this fixture the trusted descriptor comes from the explicitly owned
        # runtime, not a claim that hashing an unknown public download adds trust.
        for person in PEOPLE:
            result = self.client(person, "init", "--descriptor", descriptor_path, "--descriptor-sha256", descriptor_sha,
                                 "--binary", self.binary, "--binary-sha256", self.binary_sha, "--allow-loopback-test")
            self.addresses[person] = result["address"]
            grant = self.client(person, "fund")["faucet_response"]
            self.assertEqual(grant["status"], "committed")
            self.assertGreater(int(grant["height"]), 0)
            self.assertEqual(int(grant["code"]), 0)
            self.txs[grant["txhash"]] = grant
            self.client(person, "onboard", "--type", "human" if person == "author" else "agent")
        self.assertEqual(len(set(self.addresses.values())), 5)
        self.evidence["participant_addresses"] = self.addresses
        for person, home in self.homes.items():
            for name in ("config/node_key.json", "config/priv_validator_key.json", "data/priv_validator_state.json"):
                self.assertFalse(os.path.lexists(home / name))
            self.assertTrue((home / "keyring-test").is_dir())
            self.assertEqual(len(list((home / "identities").glob("*.ed25519.json"))), 1)
        node_keys = self.command([self.binary, "keys", "list", "--home", self.node, "--keyring-backend", "test", "--output", "json"])
        self.assertEqual({row["name"] for row in node_keys}, {"validator", "faucet"})
        self.assertFalse({row["address"] for row in node_keys} & set(self.addresses.values()))
        self.evidence["participant_keys_absent_from_runtime_keyring"] = True

        evidence_ids = {}
        for kind, name in (("review", "scoped-review.json"), ("counterexample", "counterexample.json")):
            generated = subprocess.run([sys.executable, fixture.EXAMPLE / "prime-polynomial.py", kind], capture_output=True, check=True, timeout=10).stdout
            self.assertEqual(generated, (fixture.EXAMPLE / name).read_bytes())
            evidence_ids[kind] = "sha256:" + hashlib.sha256(generated).hexdigest()
            self.evidence[kind + "_evidence"] = json.loads(generated)
        submitted = self.client("author", "submit", "--content", fixture.CLAIM, "--method", "M-COMPUTATIONAL", "--reasoning", fixture.REASONING)
        accepted_id = submitted["claim_id"]
        recovered = self.client("author", "retry", "--tx", submitted["txhash"])
        self.assertEqual(recovered, submitted, "an already committed retry must return the exact observed receipt")
        pending = self.history(accepted_id)
        self.assertEqual(pending["record"]["claim"]["fact_content"], fixture.CLAIM)
        self.assertEqual(pending["record"]["claim"]["reasoning_trace"], fixture.REASONING)
        self.assertFalse(pending["record"].get("facts"))
        self.client("challenger", "watch", "--claim", accepted_id)
        self.assertEqual([c["claim_id"] for c in self.client("challenger", "snapshot")["claims"]], [accepted_id])
        reasons = [fixture.review_reason(p) for p in REVIEWERS]
        before = self.commit(accepted_id, ["accept"] * 3, reasons, fixture.REVIEW_SCOPE, evidence_ids["review"])
        self.stop()
        self.start()
        self.assertEqual(self.get("/network.json"), raw, "restart must not rebind the descriptor")
        self.assertEqual(self.history(accepted_id)["record"]["rounds"][0], before)
        self.reveal(accepted_id, ["accept"] * 3, reasons, fixture.REVIEW_SCOPE, evidence_ids["review"], 1)
        accepted = self.history(accepted_id)
        self.assertEqual(len(accepted["record"]["facts"]), 1)
        fact = accepted["record"]["facts"][0]["fact"]
        self.assertEqual(fact["content"], fixture.CLAIM)
        self.evidence["accepted_before_counterclaim"] = accepted

        second_id = self.client("author", "submit", "--content", fixture.SECOND_CLAIM, "--method", "M-COMPUTATIONAL",
                                "--reasoning", "Synthetic signed disagreement; one rejection is intentional.")["claim_id"]
        counter_id = self.client("challenger", "challenge", "--fact", fact["id"], "--content", fixture.COUNTER_CONTENT,
                                 "--reason", fixture.COUNTER_REASON, "--evidence", evidence_ids["counterexample"])["claim_id"]
        second_reasons = [f"Synthetic {p}: intentional {v} for the bounded test." for p, v in zip(REVIEWERS, ("accept", "accept", "reject"))]
        counter_reasons = [f"Synthetic {p}: checked 1681 = 41*41 at n=40." for p in REVIEWERS]
        scope = "Isolated separate-home test; all participants are controlled by one harness, not independent people."
        self.commit(second_id, ["accept", "accept", "reject"], second_reasons, scope, evidence_ids["review"])
        self.commit(counter_id, ["accept"] * 3, counter_reasons, scope, evidence_ids["counterexample"])
        self.reveal(second_id, ["accept", "accept", "reject"], second_reasons, scope, evidence_ids["review"], 3)
        self.reveal(counter_id, ["accept"] * 3, counter_reasons, scope, evidence_ids["counterexample"], 1)
        final = self.history(accepted_id)
        related = next(row["record"] for row in final["related_claims"] if row["record"]["claim_id"] == counter_id)
        self.assertEqual(related["claim"]["argument_text"], fixture.COUNTER_REASON)
        self.assertEqual(related["claim"]["evidence_ids"], [evidence_ids["counterexample"]])
        relation = next(row for row in related["facts"][0]["outgoing_relations"] if row["target_fact_id"] == fact["id"])
        self.assertTrue(fixture.enum_is(relation["relation"], 2, "RELATION_TYPE_CONTRADICTS"))
        self.assertEqual(relation["creator"], self.addresses["challenger"])
        self.assertTrue(fixture.enum_is(final["record"]["facts"][0]["fact"]["status"], 5, "FACT_STATUS_CONTESTED"))
        inconclusive = self.history(second_id)
        self.assertFalse(inconclusive["record"].get("facts"))
        self.assertTrue(fixture.enum_is(inconclusive["record"]["claim"]["status"], 10, "CLAIM_STATUS_INSUFFICIENT"))
        self.evidence.update(final_root_history=final, inconclusive_history=inconclusive)
        self.stop()
        self.start()
        self.assertEqual(self.get("/network.json"), raw)
        after = self.history(accepted_id)
        self.assertEqual(after["record"]["claim"], final["record"]["claim"])
        self.assertEqual(after["record"]["rounds"], final["record"]["rounds"])
        for person in REVIEWERS:
            snapshot = self.client(person, "snapshot")
            self.assertFalse(snapshot["local_only"])
            self.assertTrue(snapshot["loopback_test"])
            self.assertEqual(len(snapshot["pending_reviews"]), 3)
            self.assertTrue(all(row["status"] == "revealed" for row in snapshot["pending_reviews"]))
            self.assertTrue(all(not {"salt", "vote", "reason", "attestation"} & row.keys() for row in snapshot["pending_reviews"]))
        sizes = []
        for txhash, receipt in sorted(self.txs.items()):
            tx = self.rpc("tx", hash=base64.b64encode(bytes.fromhex(txhash)).decode(), prove=False)
            txraw = base64.b64decode(tx["tx"], validate=True)
            self.assertEqual(hashlib.sha256(txraw).hexdigest().upper(), txhash)
            self.assertGreater(int(tx["height"]), 0)
            self.assertEqual(int(tx["tx_result"]["code"]), 0)
            sizes.append({"txhash": txhash, "height": tx["height"], "bytes": len(txraw), "code": 0})
        self.assertEqual(len(sizes), 31, "five grants, five onboard, three submissions, nine commits, nine reveals")
        self.stop()
        after_hashes = {name: hashlib.sha256((ROOT / name).read_bytes()).hexdigest() for name in self.source_hashes}
        self.evidence["source_hashes_after"] = after_hashes
        self.assertEqual(after_hashes, self.source_hashes, "tooling changed during the native run")
        self.assertEqual(hashlib.sha256(self.binary.read_bytes()).hexdigest(), self.binary_sha)
        self.evidence.update(result="PASS", final_root_history=final, inconclusive_history=inconclusive,
                             tx_sizes=sizes, clean_restart=True, participant_node_keys_absent=True)


class NativeDiagnosticTests(unittest.TestCase):
    def test_failed_client_retains_public_execution_and_preserves_original_assertion(self):
        with tempfile.TemporaryDirectory(prefix="zerone-shared-diagnostic-") as temporary:
            root = Path(temporary)
            native = NativeSharedClaimsTests("test_five_separate_participants_signed_rounds_gateway_and_restart")
            native.evidence = {}
            native.reports = root
            native.homes = {"author": root / "unread-private-home"}
            native.origin = "http://127.0.0.1:12345"
            raw = b"public committed fixture bytes, no private key or unrevealed preimage"
            txhash = hashlib.sha256(raw).hexdigest().upper()
            public_tx = {"hash": txhash, "height": "100", "tx": base64.b64encode(raw).decode(),
                         "tx_result": {"code": 1, "log": "claim cooldown active:15 blocks remaining"}}
            native.rpc = mock.Mock(return_value=public_tx)
            failed = subprocess.CompletedProcess([], 1, "", f"shared-claims: Transaction {txhash} ended failed; receipt retained.")
            observed = subprocess.CompletedProcess([], 0,
                '{"params":{"claim_cooldown_blocks":"50","unrelated":"omitted"},"creation_multiplier_bps":"500000","pressure_bps":"1000000","unrelated":"omitted"}', "")
            with mock.patch.object(subprocess, "run", side_effect=[failed] + [observed] * 6) as run:
                with self.assertRaisesRegex(AssertionError, "ended failed"):
                    native.command(["unused-client"], failure_context={"participant": "author", "action": "submit"})
            row = native.evidence["client_failures"][0]
            self.assertEqual(row["signed_tx_sha256"], txhash.lower())
            self.assertEqual(row["signed_tx_bytes"], len(raw))
            self.assertEqual(row["execution"]["raw_log"], public_tx["tx_result"]["log"])
            self.assertEqual(row["execution"]["code"], 1)
            self.assertNotIn("transaction", row)
            self.assertNotIn(public_tx["tx"], json.dumps(row))
            self.assertNotIn("unrelated", json.dumps(row))
            self.assertEqual(row["capture_status"], "public_execution_observed")
            self.assertEqual(len(row["admission_context"]), 6)
            self.assertEqual({r["height"] for r in row["admission_context"]}, {99, 100})
            self.assertFalse(native.homes["author"].exists(), "diagnostics must not create a private home")
            self.assertEqual(json.loads((root / "client-failure-1.json").read_text()), row)
            self.assertTrue(all(call.args[0][1] == "query" for call in run.call_args_list[1:]))

    def test_diagnostic_read_failure_never_masks_original_client_failure(self):
        native = NativeSharedClaimsTests("test_five_separate_participants_signed_rounds_gateway_and_restart")
        native.evidence = {}
        native.reports = None
        native.rpc = mock.Mock(side_effect=TimeoutError("public RPC unavailable"))
        failed = subprocess.CompletedProcess([], 1, "", "shared-claims: Transaction " + "A" * 64 + " ended failed; receipt retained.")
        with mock.patch.object(subprocess, "run", return_value=failed):
            with self.assertRaisesRegex(AssertionError, "ended failed"):
                native.command(["unused-client"], failure_context={"participant": "author", "action": "submit"})
        self.assertEqual(native.evidence["client_failures"][0]["capture_error_type"], "TimeoutError")


if __name__ == "__main__":
    unittest.main()
