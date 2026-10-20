#!/usr/bin/env python3
"""Opt-in actual signed development workflows through the real gateway.

Set ZERONE_SHARED_TEST_BINARY to a native knowledge-11 zeroned: the frozen
zerone-dev-1 runtime and gateway refuse a knowledge-12 node until the network is
relaunched. Each chain is an isolated --local-test fixture with accelerated review
windows. All private key homes are controlled by this test; no human/reviewer
independence is asserted.
Optional ZERONE_SHARED_EVIDENCE_PARENT retains curated public observations only.
"""
import base64
from contextlib import contextmanager, redirect_stderr, redirect_stdout
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import io
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
import threading
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


@contextmanager
def lost_broadcast_response(upstream):
    """Owned loopback fault fixture; no retries, redirects or ambient proxy."""
    if not re.fullmatch(r"http://127\.0\.0\.1:[0-9]+", upstream):
        raise ValueError("Response-loss fixture requires an owned loopback gateway")
    observed = {"armed": False, "broadcasts": [], "dropped": []}
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), fixture.NoRedirect())

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_args):
            pass

        def forward(self):
            size = int(self.headers.get("Content-Length", "0"))
            if not 0 <= size <= 8 * 1024 * 1024:
                self.send_error(413)
                return
            data = self.rfile.read(size) if self.command == "POST" else None
            request = json.loads(data) if data else {}
            message = urllib.request.Request(upstream + self.path, data=data,
                                             headers={"Content-Type": "application/json"})
            try:
                response = opener.open(message, timeout=25)
            except urllib.error.HTTPError as error:
                response = error
            with response:
                status = response.status
                raw = response.read(8 * 1024 * 1024 + 1)
            if len(raw) > 8 * 1024 * 1024:
                self.send_error(502)
                return
            if request.get("method") == "broadcast_tx_sync":
                tx = base64.b64decode(request["params"]["tx"], validate=True)
                result = json.loads(raw).get("result", {})
                record = {"txhash": hashlib.sha256(tx).hexdigest().upper(),
                          "http_status": status, "check_tx": result}
                observed["broadcasts"].append(record)
                if observed["armed"]:
                    observed["armed"] = False
                    observed["dropped"].append(record)
                    # The unchanged gateway has already replied. Lose only the
                    # caller's response, never replace admission with a mock.
                    self.close_connection = True
                    self.connection.shutdown(socket.SHUT_RDWR)
                    return
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(raw)))
            self.end_headers()
            self.wfile.write(raw)

        do_GET = forward
        do_POST = forward

    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield "http://127.0.0.1:" + str(server.server_port), observed
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
        if thread.is_alive():
            raise RuntimeError("Response-loss fixture did not stop")


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
        self.tx_participants = {}
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
            previous = self.tx_participants.setdefault(value["txhash"], person)
            self.assertEqual(previous, person)
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

    def bank_snapshot(self):
        # All accounts and supply share one committed context. Only public
        # gateway queries are used; the runtime's keyring is never consulted.
        status = json.loads(self.get("/status"))["result"]
        height = int(status["sync_info"]["latest_block_height"])
        self.assertGreater(height, 0)
        def query(*args):
            return self.command([self.binary, "query", *args, "--home", self.homes["author"],
                                 "--node", self.origin, "--height", height, "--output", "json"])
        balances = {}
        for person, address in self.addresses.items():
            result = query("bank", "balances", address)
            self.assertFalse(result.get("pagination", {}).get("next_key"), "balance query must be complete")
            self.assertEqual({coin["denom"] for coin in result["balances"]}, {"uzrn"})
            balances[person] = int(result["balances"][0]["amount"])
        supply = query("bank", "total")
        self.assertFalse(supply.get("pagination", {}).get("next_key"), "supply query must be complete")
        return {"height": height, "balances": balances, "supply": supply["supply"]}

    def funding_terms(self, history, receipt, challenge=False):
        claim = history["record"]["claim"]
        self.assertEqual(claim["id"], receipt["claim_id"])
        paid = int(receipt["collateral_uzrn" if challenge else "review_fee_uzrn"])
        self.assertGreater(paid, 0)
        pool = paid * 55 // 100
        terms = claim["funding_terms"]
        self.assertEqual(set(terms), {"policy_version", "kind", "paid_amount", "review_budget", "refundable_amount", "retained_fee"})
        self.assertEqual(terms["policy_version"], 1)
        self.assertTrue(fixture.enum_is(terms["kind"], 2 if challenge else 1,
                                       "CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT" if challenge else "CLAIM_FUNDING_KIND_REVIEW_FEE"))
        self.assertEqual({key: terms[key] for key in ("paid_amount", "review_budget", "refundable_amount", "retained_fee")},
                         {"paid_amount": str(paid), "review_budget": str(pool),
                          "refundable_amount": str(paid - pool if challenge else 0),
                          "retained_fee": str(0 if challenge else paid - pool)})
        return terms

    def terminal_funding(self, claim, seconds=160):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            value = json.loads(self.get("/claims/" + claim))
            row = value["record"]["rounds"][0]
            terminal = fixture.enum_is(row["phase"], 4, "VERIFICATION_PHASE_COMPLETE") or fixture.enum_is(row["phase"], 5, "VERIFICATION_PHASE_EXPIRED")
            plans = [row[field] for field in ("verifier_reward_settlement", "claim_refund_settlement") if row.get(field)]
            if terminal and all(int(plan.get("paid_at_block", 0)) > 0 for plan in plans):
                return value
            self.assertIsNone(self.process.poll())
            time.sleep(0.8)
        self.fail("Terminal funding was not observed within the isolated test window")

    def assert_funding_settlement(self, history, admission_terms, reviewed=True):
        record = history["record"]
        self.assertEqual(record["claim"]["funding_terms"], admission_terms, "admission terms must remain immutable")
        self.assertEqual(len(record["rounds"]), 1)
        row = record["rounds"][0]
        pool = int(admission_terms["review_budget"])
        plan = row.get("verifier_reward_settlement")
        if reviewed:
            self.assertIsNotNone(plan)
            payments = plan["payments"]
            self.assertEqual(len(payments), len(REVIEWERS))
            self.assertEqual({p["verifier"] for p in payments}, {self.addresses[p] for p in REVIEWERS})
            share, remainder = divmod(pool, len(REVIEWERS))
            self.assertEqual(sorted(int(p["amount"]) for p in payments), sorted([share + remainder, share, share]))
            self.assertTrue(all(p["withheld"] == "0" for p in payments))
            self.assertEqual(plan["withheld_total"], "0", "a dissenting review receives its pool share")
        else:
            self.assertFalse(row.get("commits"))
            self.assertFalse(row.get("reveals"))
            self.assertIsNone(plan, "an empty round must not invent reviewer payments")
            self.assertFalse(record.get("facts"))
        refund_amount = int(admission_terms["refundable_amount"]) + (0 if reviewed else pool)
        refund = row.get("claim_refund_settlement")
        if refund_amount:
            self.assertIsNotNone(refund)
            self.assertEqual(refund["recipient"], record["claim"]["submitter"])
            self.assertEqual(refund["amount"], str(refund_amount))
        else:
            self.assertIsNone(refund)
        for payment in (plan, refund):
            if payment is not None:
                self.assertEqual(int(payment["created_at_block"]), int(row["verdict_block"]))
                self.assertGreaterEqual(int(payment["paid_at_block"]), int(payment["created_at_block"]))
                self.assertGreater(int(payment["paid_at_block"]), 0, "a recorded obligation is not a payment")
                self.assertLessEqual(int(payment["paid_at_block"]), int(history["block_height"]))

    def assert_bank_conservation(self, before, after, histories, fees):
        by_address = {address: person for person, address in self.addresses.items()}
        expected = dict(before["balances"])
        retained = 0
        for person, amount in fees.items():
            expected[person] -= amount
        for history in histories.values():
            record = history["record"]
            terms = record["claim"]["funding_terms"]
            expected[by_address[record["claim"]["submitter"]]] -= int(terms["paid_amount"])
            retained += int(terms["retained_fee"])
            row = record["rounds"][0]
            for payment in (row.get("verifier_reward_settlement") or {}).get("payments", []):
                expected[by_address[payment["verifier"]]] += int(payment["amount"])
            refund = row.get("claim_refund_settlement")
            if refund:
                expected[by_address[refund["recipient"]]] += int(refund["amount"])
        self.assertEqual(after["balances"], expected, "actual bank changes must match admissions, payments, refunds and signed transaction fees")
        self.assertEqual(sum(after["balances"].values()) - sum(before["balances"].values()), -sum(fees.values()) - retained)
        self.assertEqual(after["supply"], before["supply"], "review settlement must not change the supply")

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

    def initialize_runtime(self):
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

    def test_five_separate_participants_signed_rounds_gateway_and_restart(self):
        self.initialize_runtime()
        raw = self.get("/network.json")
        descriptor_path = self.root / "network.json"
        descriptor_path.write_bytes(raw)
        descriptor_sha = hashlib.sha256(raw).hexdigest()
        descriptor = json.loads(raw)
        self.assertTrue(descriptor["local_test"])
        self.assertEqual(descriptor["knowledge_version"], 11)
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
        bank_before = self.bank_snapshot()
        setup_transactions = set(self.txs)

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
        accepted_terms = self.funding_terms(pending, submitted)
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

        second_submission = self.client("author", "submit", "--content", fixture.SECOND_CLAIM, "--method", "M-COMPUTATIONAL",
                                        "--reasoning", "Synthetic signed disagreement; one rejection is intentional.")
        second_id = second_submission["claim_id"]
        counter_submission = self.client("challenger", "challenge", "--fact", fact["id"], "--content", fixture.COUNTER_CONTENT,
                                         "--reason", fixture.COUNTER_REASON, "--evidence", evidence_ids["counterexample"])
        counter_id = counter_submission["claim_id"]
        second_terms = self.funding_terms(self.history(second_id), second_submission)
        counter_terms = self.funding_terms(self.history(counter_id), counter_submission, challenge=True)
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
        # The frozen zerone-dev-1 tooling serves knowledge 11 only (descriptor above), so
        # this is a knowledge-11 run: the contradiction contested the fact when admitted.
        # test_claim_workflow_native.py checks the knowledge-12 contract (contest on acceptance).
        self.assertTrue(fixture.enum_is(final["record"]["facts"][0]["fact"]["status"], 5, "FACT_STATUS_CONTESTED"))
        inconclusive = self.history(second_id)
        self.assertFalse(inconclusive["record"].get("facts"))
        self.assertTrue(fixture.enum_is(inconclusive["record"]["claim"]["status"], 10, "CLAIM_STATUS_INSUFFICIENT"))
        self.evidence.update(final_root_history=final, inconclusive_history=inconclusive)

        # Empty ordinary and correction rounds prove that no review is neither
        # approval nor a lost refund. These remain synthetic signed admissions.
        empty_submission = self.client("author", "submit", "--content", "Synthetic unreviewed funding fixture: this assertion has no supporting review.",
                                       "--reasoning", "Exercise return of the unused review budget without asserting a scientific result.")
        empty_counter = self.client("challenger", "challenge", "--fact", fact["id"],
                                   "--content", "Synthetic unreviewed correction fixture: no reviewer has checked this counterclaim.",
                                   "--reason", "Exercise return of the unused review budget and challenge remainder.")
        empty_terms = self.funding_terms(self.history(empty_submission["claim_id"]), empty_submission)
        empty_counter_terms = self.funding_terms(self.history(empty_counter["claim_id"]), empty_counter, challenge=True)
        terms_by_claim = {accepted_id: accepted_terms, second_id: second_terms, counter_id: counter_terms,
                          empty_submission["claim_id"]: empty_terms, empty_counter["claim_id"]: empty_counter_terms}
        reviewed_claims = {accepted_id, second_id, counter_id}
        financial_histories = {claim: self.terminal_funding(claim) for claim in terms_by_claim}
        for claim, history in financial_histories.items():
            self.assert_funding_settlement(history, terms_by_claim[claim], claim in reviewed_claims)
        bank_after = self.bank_snapshot()
        self.evidence["funding_histories"] = financial_histories
        self.stop()
        self.start()
        self.assertEqual(self.get("/network.json"), raw)
        bank_restart = self.bank_snapshot()
        self.assertEqual(bank_restart["balances"], bank_after["balances"], "restart must not pay a reviewer or refund twice")
        self.assertEqual(bank_restart["supply"], bank_after["supply"])
        for claim, history in financial_histories.items():
            restarted = self.history(claim)["record"]
            self.assertEqual(restarted["claim"], history["record"]["claim"])
            self.assertEqual(restarted["rounds"], history["record"]["rounds"])
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
        fees = {person: 0 for person in PEOPLE}
        fee_receipts = []
        for txhash, receipt in sorted(self.txs.items()):
            tx = self.rpc("tx", hash=base64.b64encode(bytes.fromhex(txhash)).decode(), prove=False)
            txraw = base64.b64decode(tx["tx"], validate=True)
            self.assertEqual(hashlib.sha256(txraw).hexdigest().upper(), txhash)
            self.assertGreater(int(tx["height"]), 0)
            self.assertEqual(int(tx["tx_result"]["code"]), 0)
            sizes.append({"txhash": txhash, "height": tx["height"], "bytes": len(txraw), "code": 0})
            if txhash not in setup_transactions:
                person = self.tx_participants[txhash]
                # Decode only hash-checked, already committed public bytes. Fee
                # accounting must not assume a constant from the client code.
                decoded = self.command([self.binary, "tx", "decode", tx["tx"], "--home", self.homes[person], "--output", "json"])
                fee = decoded["auth_info"]["fee"]
                self.assertIn(fee.get("payer", ""), ("", self.addresses[person]))
                self.assertFalse(fee.get("granter"), "this fixture uses each participant's own funds")
                self.assertEqual(len(fee["amount"]), 1)
                self.assertEqual(fee["amount"][0]["denom"], "uzrn")
                amount = int(fee["amount"][0]["amount"])
                self.assertGreater(amount, 0)
                fees[person] += amount
                fee_receipts.append({"txhash": txhash, "participant": person, "fee_uzrn": str(amount)})
        expected_transactions = len(setup_transactions) + len(financial_histories) + sum(
            2 * len(value["record"]["rounds"][0].get("reveals", [])) for value in financial_histories.values())
        self.assertEqual(len(sizes), expected_transactions, "every fixture grant, onboard, admission, commit and reveal must have a receipt")
        self.assertEqual(len(fee_receipts), len(self.txs) - len(setup_transactions))
        self.assert_bank_conservation(bank_before, bank_after, financial_histories, fees)
        self.evidence["bank_comparison"] = {"before": bank_before, "after": bank_after, "after_restart": bank_restart,
                                            "transaction_fees_by_participant": fees, "signed_fee_receipts": fee_receipts,
                                            "actual_balances_match_recorded_settlement": True, "restart_did_not_repeat_payment": True}
        self.stop()
        after_hashes = {name: hashlib.sha256((ROOT / name).read_bytes()).hexdigest() for name in self.source_hashes}
        self.evidence["source_hashes_after"] = after_hashes
        self.assertEqual(after_hashes, self.source_hashes, "tooling changed during the native run")
        self.assertEqual(hashlib.sha256(self.binary.read_bytes()).hexdigest(), self.binary_sha)
        self.evidence.update(result="PASS", final_root_history=final, inconclusive_history=inconclusive,
                             tx_sizes=sizes, clean_restart=True, participant_node_keys_absent=True)


class NativeResponseLossTests(NativeSharedClaimsTests):
    # Keep this current-client scenario out of the predecessor upgrade lane.
    test_five_separate_participants_signed_rounds_gateway_and_restart = None

    def test_lost_broadcast_response_recovers_original_without_signing_again(self):
        self.initialize_runtime()
        with lost_broadcast_response(self.origin) as (origin, transport):
            # Pin this explicitly owned fault endpoint before creating the home.
            # Only its URLs differ; genesis, runtime and client bindings remain.
            descriptor = json.loads(self.get("/network.json"))
            descriptor.update(rpc_url=origin, genesis_url=origin + "/genesis.json", faucet_url=origin + "/faucet")
            raw = (json.dumps(descriptor, sort_keys=True) + "\n").encode()
            descriptor_path = self.root / "fault-network.json"
            descriptor_path.write_bytes(raw)
            initialized = self.client("author", "init", "--descriptor", descriptor_path,
                "--descriptor-sha256", hashlib.sha256(raw).hexdigest(), "--binary", self.binary,
                "--binary-sha256", self.binary_sha, "--allow-loopback-test")
            self.addresses["author"] = initialized["address"]
            self.assertEqual(self.client("author", "fund")["faucet_response"]["status"], "committed")
            self.client("author", "onboard", "--type", "human")

            spec = importlib.util.spec_from_file_location("response_loss_client", CLIENT)
            shared = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(shared)
            commands = []
            real_run = subprocess.run

            def observed_run(argv, **kwargs):
                # Observe command families only. Signing, encoding, HTTP and
                # chain execution still run through the unchanged native CLI.
                commands.append(tuple(map(str, argv[1:3])))
                return real_run(argv, **kwargs)

            def invoke(command, *arguments):
                output, error = io.StringIO(), io.StringIO()
                previous_mask = os.umask(0o077)
                try:
                    with mock.patch.object(sys, "argv", [str(CLIENT), command, "--home", str(self.homes["author"]), *arguments]), \
                         mock.patch.object(subprocess, "run", side_effect=observed_run), \
                         redirect_stdout(output), redirect_stderr(error):
                        code = shared.main()
                finally:
                    os.umask(previous_mask)
                return code, output.getvalue(), error.getvalue()

            def sequence():
                value = self.command([self.binary, "query", "auth", "account", self.addresses["author"],
                    "--home", self.homes["author"], "--node", self.origin, "--output", "json"])
                self.assertEqual(value["account"]["type"], "cosmos-sdk/BaseAccount")
                self.assertEqual(value["account"]["value"]["address"], self.addresses["author"])
                return int(value["account"]["value"]["sequence"])

            arguments = ("--content", fixture.CLAIM, "--method", "M-COMPUTATIONAL", "--reasoning", fixture.REASONING)
            before_sequence = sequence()
            before_balance = self.bank_snapshot()["balances"]["author"]
            before_broadcasts = len(transport["broadcasts"])
            transport["armed"] = True
            code, output, error = invoke("submit", *arguments)
            self.assertEqual(code, 1, "the admitted broadcast response must actually be lost")
            self.assertEqual(output, "", "loss must not claim a successful receipt")
            self.assertIn("No success is asserted", error)
            submission_sign_calls = commands.count(("tx", "sign"))
            self.assertEqual(submission_sign_calls, 1)
            self.assertEqual(commands.count(("tx", "broadcast")), 1)
            self.assertEqual(len(transport["broadcasts"]) - before_broadcasts, 1)
            self.assertEqual(len(transport["dropped"]), 1)
            dropped = transport["dropped"][0]
            self.assertEqual(dropped["http_status"], 200)
            self.assertEqual(int(dropped["check_tx"]["code"]), 0)
            self.assertEqual(dropped["check_tx"]["hash"].upper(), dropped["txhash"])

            attempts = self.homes["author"] / "claim-workflow/attempts"
            saved = [(p, json.loads(p.read_text())) for p in attempts.glob("*.json")]
            submits = [(p, a) for p, a in saved if a["action"] == "submit"]
            self.assertEqual(len(submits), 1)
            attempt_path, retained = submits[0]
            self.assertEqual(retained["status"], "broadcasting")
            self.assertEqual(retained["broadcast_count"], 1)
            self.assertEqual(retained["txhash"], dropped["txhash"])
            signed = base64.b64decode(retained["tx_bytes_base64"], validate=True)
            self.assertEqual(hashlib.sha256(signed).hexdigest().upper(), retained["txhash"])

            # Observe commitment without reconciling the participant's attempt.
            # The durable local state must still be ambiguous until recovery.
            deadline = time.monotonic() + 30
            while True:
                observed = shared.SharedWorkflow(self.homes["author"], 1).tx_observation(retained["txhash"])
                if observed is not None:
                    break
                self.assertLess(time.monotonic(), deadline, "admitted transaction did not commit")
                time.sleep(0.3)
            self.assertEqual(int(observed["code"]), 0)
            self.assertGreater(int(observed["height"]), 0)
            self.assertEqual(json.loads(attempt_path.read_text())["status"], "broadcasting")
            self.stop()
            self.start()  # Persisted node plus unchanged participant home.

            commands.clear()
            code, output, error = invoke("retry", "--tx", retained["txhash"])
            self.assertEqual(code, 0, error)
            recovered = json.loads(output)
            self.assertEqual(recovered["status"], "committed")
            self.assertEqual(recovered["txhash"], retained["txhash"])
            self.assertEqual(int(recovered["height"]), int(observed["height"]))
            self.assertEqual(int(recovered["code"]), 0)
            code, repeated, error = invoke("submit", *arguments)
            self.assertEqual(code, 0, error)
            self.assertEqual(json.loads(repeated), recovered)
            self.assertEqual(commands.count(("tx", "sign")), 0, "recovery must not sign again")
            self.assertEqual(commands.count(("tx", "broadcast")), 0, "committed recovery must not broadcast again")
            self.assertEqual(len(transport["broadcasts"]) - before_broadcasts, 1)
            self.assertEqual({p.name for p in attempts.glob("*.json")}, {p.name for p, _ in saved})
            final = json.loads(attempt_path.read_text())
            self.assertEqual(final["tx_bytes_base64"], retained["tx_bytes_base64"])
            self.assertEqual(final["broadcast_count"], 1)
            after_sequence = sequence()
            self.assertEqual(after_sequence, before_sequence + 1)
            fee = int(retained["context"]["review_fee_uzrn"]) + int(descriptor["tx_fee_uzrn"])
            after_balance = self.bank_snapshot()["balances"]["author"]
            self.assertEqual(after_balance, before_balance - fee)
            public_tx = self.rpc("tx", hash=base64.b64encode(bytes.fromhex(retained["txhash"])).decode(), prove=False)
            self.assertEqual(base64.b64decode(public_tx["tx"], validate=True), signed)
            self.assertEqual(public_tx["tx_result"]["code"], 0)
            block = json.loads(self.get("/block?height=" + str(recovered["height"])))
            included = [base64.b64decode(tx, validate=True) for tx in block["result"]["block"]["data"]["txs"]]
            self.assertEqual(included.count(signed), 1)
            history = self.history(recovered["claim_id"])
            self.assertEqual(history["record"]["claim"]["fact_content"], fixture.CLAIM)
            self.evidence["response_loss"] = {"txhash": retained["txhash"], "height": recovered["height"],
                "claim_id": recovered["claim_id"], "retained_before_recovery": retained["status"],
                "check_tx_code": dropped["check_tx"]["code"], "execution_code": recovered["code"],
                "submission_sign_calls": submission_sign_calls, "recovery_sign_calls": commands.count(("tx", "sign")),
                "broadcasts": len(transport["broadcasts"]) - before_broadcasts,
                "sequence_before": before_sequence, "sequence_after": after_sequence,
                "balance_before_uzrn": before_balance, "balance_after_uzrn": after_balance,
                "inclusions_at_commit_height": included.count(signed),
                "charged_uzrn": before_balance - after_balance, "node_restarted_before_recovery": True,
                "same_command_returns_original_receipt": json.loads(repeated) == recovered}
            self.assertEqual({name: hashlib.sha256((ROOT / name).read_bytes()).hexdigest() for name in self.source_hashes},
                             self.source_hashes)
            self.evidence["source_hashes_after"] = self.source_hashes


class NativeScenarioCollectionTests(unittest.TestCase):
    def test_native_scenarios_have_separate_owners(self):
        expected = {
            "NativeSharedClaimsTests": ["test_five_separate_participants_signed_rounds_gateway_and_restart"],
            "NativeResponseLossTests": ["test_lost_broadcast_response_recovers_original_without_signing_again"],
            "StagedNative": ["test_staged_upgrade_and_unchanged_participant"],
        }
        collected = {owner: [] for owner in expected}

        def collect(suite):
            for test in suite:
                if isinstance(test, unittest.TestSuite):
                    collect(test)
                elif type(test).__name__ in collected:
                    collected[type(test).__name__].append(test._testMethodName)

        # Discover the real opt-in suites, but never run their native setup.
        for directory, pattern in ((ROOT / "scripts", "test_shared_claims_native.py"),
                                   (RUNTIME.parent, "test_upgrade_native.py")):
            loader = unittest.TestLoader()
            collect(loader.discover(str(directory), pattern=pattern))
            self.assertEqual(loader.errors, [])
        self.maxDiff = None
        self.assertEqual(collected, expected)


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
