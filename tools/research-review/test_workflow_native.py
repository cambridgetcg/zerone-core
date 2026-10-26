"""Opt-in real adapter/client integration on an isolated loopback v11 chain.

ZERONE_RESEARCH_TEST_BINARY selects a verified native zeroned. Optional
ZERONE_RESEARCH_EVIDENCE_PARENT retains public sidecars and a scoped receipt.
Two temporary participant homes belong to one test controller; these are not
independent researchers. No existing account, chain, or participant home is used.
"""

import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import sys
import time
import unittest
from unittest import mock

import journal as j
import workflow as w


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("research_shared_fixture", ROOT / "scripts/test_shared_claims_native.py")
native = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(native)


@unittest.skipUnless(os.environ.get("ZERONE_RESEARCH_TEST_BINARY"), "set ZERONE_RESEARCH_TEST_BINARY for isolated signed adapter integration")
class NativeResearchWorkflowTests(unittest.TestCase):
    def setUp(self):
        # Composition deliberately avoids discovery/inheritance of the separate
        # five-home suite. Only setup, client and read helpers are reused here.
        self.fixture = native.NativeSharedClaimsTests(methodName="runTest")
        env = {"ZERONE_SHARED_TEST_BINARY": os.environ["ZERONE_RESEARCH_TEST_BINARY"]}
        if os.environ.get("ZERONE_RESEARCH_EVIDENCE_PARENT"):
            env["ZERONE_SHARED_EVIDENCE_PARENT"] = os.environ["ZERONE_RESEARCH_EVIDENCE_PARENT"]
        with mock.patch.dict(os.environ, env):
            self.fixture.setUp()
        self.addCleanup(self.fixture.tearDown)
        self.fixture.evidence.update(test_scope="two-home research adapter; one solo-reviewed assertion and one unreviewed journal correction",
            adapter_sources={str(path.relative_to(ROOT)): hashlib.sha256(path.read_bytes()).hexdigest()
                             for path in (Path(w.__file__), Path(__file__))})

    def initialize(self):
        fixture = self.fixture
        version = subprocess.run([str(fixture.binary), "version", "--long"], capture_output=True, text=True, check=True, timeout=30).stdout
        source = re.search(r"(?m)^commit:\s*([0-9a-f]{40})\s*$", version)
        self.assertIsNotNone(source)
        ports, sockets = [], []
        try:
            for _ in range(3):
                sock = socket.socket()
                sockets.append(sock)
                sock.bind(("127.0.0.1", 0))
                ports.append(sock.getsockname()[1])
        finally:
            for sock in sockets:
                sock.close()
        fixture.origin = "http://127.0.0.1:" + str(ports[2])
        fixture.command([sys.executable, "-I", "-B", native.RUNTIME, "init", "--home", fixture.node,
            "--binary", fixture.binary, "--source-commit", source[1], "--local-test", "--review-window-blocks", "30",
            "--rpc-port", ports[0], "--p2p-port", ports[1], "--gateway-port", ports[2]])
        fixture.start()
        raw = fixture.get("/network.json")
        descriptor = json.loads(raw)
        self.assertTrue(descriptor["local_test"])
        self.assertEqual(descriptor["knowledge_version"], 11)
        path = fixture.root / "network.json"
        path.write_bytes(raw)
        fixture.evidence.update(binary_sha256=fixture.binary_sha, binary_source_commit=source[1],
                                descriptor=descriptor, descriptor_sha256=w.digest(raw))
        for person in ("author", "reviewer1"):
            result = fixture.client(person, "init", "--descriptor", path, "--descriptor-sha256", w.digest(raw),
                "--binary", fixture.binary, "--binary-sha256", fixture.binary_sha, "--allow-loopback-test")
            fixture.addresses[person] = result["address"]
            self.assertEqual(fixture.client(person, "fund")["faucet_response"]["status"], "committed")
            fixture.client(person, "onboard", "--type", "human" if person == "author" else "agent")
        self.assertEqual(len(set(fixture.addresses.values())), 2)
        fixture.evidence["participant_addresses"] = fixture.addresses

    def wait_for_author_cooldown(self, receipt):
        fixture = self.fixture
        deadline = time.monotonic() + 180
        while time.monotonic() < deadline:
            height = fixture.committed_height()
            def query(*args):
                return fixture.command([fixture.homes["author"] / "bin/zeroned", "query", *args,
                    "--home", fixture.homes["author"], "--node", fixture.origin, "--height", height, "--output", "json"])
            params = query("knowledge", "params")["params"]
            pacing = query("alignment", "global-pacing")
            capacity = query("knowledge", "domain-capacity", "physics")
            multiplier = int(pacing.get("creation_multiplier_bps", "0")) or 1000000
            cooldown = int(params["claim_cooldown_blocks"]) * 1000000 // multiplier
            pressure = int(capacity.get("pressure_bps", "0"))
            if pressure > 1000000:
                cooldown = cooldown * pressure // 1000000
            if height >= int(receipt["height"]) + cooldown + 2:
                fixture.evidence["correction_admission_preflight"] = {"height": height, "effective_cooldown": cooldown,
                    "scope": "read observation only; native admission independently revalidates"}
                return
            time.sleep(2)
        self.fail("author cooldown did not clear within the isolated fixture bound")

    def test_record_submit_recover_review_correction_and_funding(self):
        self.initialize()
        fixture = self.fixture
        store = fixture.root / "journal"
        j.create_store(store)
        first_record = {"id": "research-original", "kind": "contribution", "title": "Synthetic polynomial assertion",
            "summary": native.fixture.CLAIM, "attributed_to": "Synthetic author controlled by this test",
            "occurred_on": None, "evidence": [], "contribution_type": "claim", "doi": None}
        j.append_records(store, [first_record])
        original_export = j.canonical(j.export_store(store)) + b"\n"
        draft = w.prepare(original_export, first_record["id"], {"content": native.fixture.CLAIM,
            "reasoning": native.fixture.REASONING, "domain": "physics", "category": "empirical", "method": "M-COMPUTATIONAL"})
        path = fixture.root / "research-original.json"
        path.write_bytes(j.canonical(draft) + b"\n")
        calls, received = [], []
        def lose_stdout(home, client, command, arguments=()):
            calls.append(command)
            value = w.run_client(home, client, command, arguments)
            if command == "submit":
                received.append(value)
                raise w.Error("synthetic loss of already successful client stdout")
            return value
        with self.assertRaisesRegex(w.Error, "synthetic loss"):
            w.submit(path, fixture.homes["author"], native.CLIENT, lose_stdout)
        self.assertEqual(calls, ["snapshot", "submit"])
        self.assertEqual(len(received), 1)
        self.assertEqual(received[0]["status"], "committed")
        self.assertEqual(j.parse_json(path.read_bytes())["submission"], {"state": "started", "receipt": None})
        packet = w.refresh(path, fixture.homes["author"], native.CLIENT)
        receipt = packet["submission"]["receipt"]
        self.assertEqual(receipt, received[0])
        first_history = j.parse_json(packet["observations"][-1]["history_utf8"].encode())
        self.assertEqual(first_history["record"]["claim"]["reasoning_trace"], w.bound_reasoning(draft))
        terms = fixture.funding_terms(first_history, receipt)

        # A fresh copy must recover the observed claim with no signing invocation.
        copy_path = fixture.root / "redownloaded-draft.json"
        copy_path.write_bytes(j.canonical(draft) + b"\n")
        copy_calls = []
        def read_only(home, client, command, arguments=()):
            copy_calls.append(command)
            self.assertEqual(command, "snapshot")
            return w.run_client(home, client, command, arguments)
        recovered_copy = w.submit(copy_path, fixture.homes["author"], native.CLIENT, read_only)
        self.assertEqual(copy_calls, ["snapshot"])
        self.assertEqual(recovered_copy["submission"]["receipt"], receipt)

        evidence = "sha256:" + hashlib.sha256((native.fixture.EXAMPLE / "scoped-review.json").read_bytes()).hexdigest()
        reason = "Synthetic solo reviewer checked n=0..39 only; this is insufficient for the asserted n=0..40 scope."
        fixture.client("reviewer1", "commit", "--claim", receipt["claim_id"], "--vote", "reject",
                       "--reason", reason, "--scope", native.fixture.REVIEW_SCOPE,
                       "--evidence", evidence, "--method", "M-COMPUTATIONAL")
        fixture.wait_phase(receipt["claim_id"], 2, 75)
        fixture.client("reviewer1", "reveal", "--claim", receipt["claim_id"])
        terminal = fixture.terminal_funding(receipt["claim_id"], 90)
        row = terminal["record"]["rounds"][0]
        self.assertEqual(len(row["reveals"]), 1)
        self.assertEqual(row["reveals"][0]["attestation"]["reason"], reason)
        self.assertTrue(native.fixture.enum_is(row["verdict"], 3, "VERDICT_INCONCLUSIVE"))
        self.assertFalse(terminal["record"].get("facts"))
        plan = row["verifier_reward_settlement"]
        self.assertEqual(len(plan["payments"]), 1)
        self.assertEqual(plan["payments"][0]["verifier"], fixture.addresses["reviewer1"])
        self.assertEqual(plan["payments"][0]["amount"], terms["review_budget"])
        self.assertGreater(int(plan["paid_at_block"]), 0)
        packet = w.refresh(path, fixture.homes["author"], native.CLIENT)
        self.assertEqual(packet["observations"][0]["history_utf8"], j.canonical(first_history).decode())

        correction = {**first_record, "id": "research-correction", "title": "Synthetic endpoint correction",
                      "summary": native.fixture.COUNTER_CONTENT, "supersedes": first_record["id"]}
        j.append_records(store, [correction])
        revised_export = j.canonical(j.export_store(store)) + b"\n"
        self.assertEqual(j.parse_json(revised_export)["entries"][0], j.parse_json(original_export)["entries"][0])
        correction_draft = w.prepare(revised_export, correction["id"], {"content": native.fixture.COUNTER_CONTENT,
            "reasoning": "Explicit journal correction: at n=40 the polynomial gives 1681 = 41*41. No prior native Fact is claimed.",
            "domain": "physics", "category": "empirical", "method": "M-COMPUTATIONAL"})
        correction_path = fixture.root / "research-correction.json"
        correction_path.write_bytes(j.canonical(correction_draft) + b"\n")
        self.wait_for_author_cooldown(receipt)
        # Simulate interruption after the durable start, before a client was
        # launched. Only an explicit resume can delegate this submission.
        correction_draft["participant"] = w.participant_binding(fixture.homes["author"], native.CLIENT)
        correction_draft["submission"] = {"state": "started", "receipt": None}
        w.save_packet(correction_path, correction_draft)
        correction_packet = w.resume(correction_path, fixture.homes["author"], native.CLIENT)
        correction_receipt = correction_packet["submission"]["receipt"]
        self.assertNotEqual(correction_receipt["claim_id"], receipt["claim_id"])
        correction_packet = w.refresh(correction_path, fixture.homes["author"], native.CLIENT)
        first_correction_history = j.parse_json(correction_packet["observations"][0]["history_utf8"].encode())
        correction_terms = fixture.funding_terms(first_correction_history, correction_receipt)
        empty_terminal = fixture.terminal_funding(correction_receipt["claim_id"], 100)
        fixture.assert_funding_settlement(empty_terminal, correction_terms, reviewed=False)
        correction_packet = w.refresh(correction_path, fixture.homes["author"], native.CLIENT)
        snapshot = fixture.client("author", "snapshot")
        self.assertEqual(len([r for r in snapshot["transactions"] if r["action"] == "submit"]), 2)
        fixture.evidence.update(stdout_loss_recovered=True, redownload_recovered_without_submit=True,
            explicit_resume_after_prelaunch_crash=True,
            adapter_commands_before_synthetic_loss=calls, redownload_commands=copy_calls,
            ordinary_admissions=2, solo_review_count=1, unreviewed_correction_count=1,
            original_receipt=receipt, correction_receipt=correction_receipt,
            correction_scope="journal supersedes plus separate ordinary admission; no Fact or native challenge was invented",
            signing_scope="only existing participant client; all test keys temporary and one-controller",
            private_keys_or_unrevealed_review_tuples_exported=False,
            funding_scope="retained native settlement observations; full bank conservation covered by separate native suite")
        if fixture.reports:
            outputs = {"workflow-reviewed.json": packet, "workflow-correction.json": correction_packet,
                       "workflow-draft.json": draft}
            fixture.evidence["sidecars"] = {}
            for name, value in outputs.items():
                w.validate_packet(value)
                raw = j.canonical(value) + b"\n"
                output = fixture.reports / name
                output.write_bytes(raw)
                output.chmod(0o600)
                fixture.evidence["sidecars"][name] = {"sha256": w.digest(raw), "bytes": len(raw)}
            print("research-workflow native evidence: " + str(fixture.reports), flush=True)


class CooldownHeightTests(unittest.TestCase):
    def test_cooldown_queries_share_the_committed_height(self):
        case = NativeResearchWorkflowTests("runTest")
        fixture = native.NativeSharedClaimsTests("runTest")
        case.fixture = fixture
        fixture.homes = {"author": Path("/unused/author")}
        fixture.origin = "http://127.0.0.1:12345"
        fixture.evidence = {}
        fixture.get = mock.Mock(side_effect=lambda path: json.dumps({
            "/status": {"result": {"sync_info": {"latest_block_height": "101"}}},
            "/abci_info": {"result": {"response": {"last_block_height": "100"}}},
        }[path]).encode())
        fixture.command = mock.Mock(side_effect=[{"params": {"claim_cooldown_blocks": "0"}},
                                                {"creation_multiplier_bps": "1000000"}, {"pressure_bps": "0"}])
        with mock.patch.object(time, "sleep") as sleep:
            case.wait_for_author_cooldown({"height": "98"})
        self.assertEqual([call.args[0][call.args[0].index("--height") + 1]
                          for call in fixture.command.call_args_list], [100, 100, 100])
        fixture.get.assert_called_once_with("/abci_info")
        self.assertEqual(fixture.evidence["correction_admission_preflight"]["height"], 100)
        sleep.assert_not_called()


if __name__ == "__main__":
    unittest.main()
