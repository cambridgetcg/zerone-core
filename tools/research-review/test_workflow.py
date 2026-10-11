"""Offline adapter contract and crash-boundary tests; no accounts or network."""

import contextlib
import copy
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import journal as j
import workflow as w
from test_journal import assessment, concern, contribution, fixture, relation


ADDRESS = "zrn1" + "q" * 38
CLAIM = "1" * 32
INTENT = {"content": "A fictional bounded assertion about this supplied research artifact.",
          "reasoning": "I inspected the supplied artifact and explicitly state this limited inference.",
          "domain": "physics", "category": "empirical", "method": ""}


def receipt(**changes):
    value = {"id": "a" * 32, "action": "submit", "actor": "participant", "status": "committed",
             "txhash": "A" * 64, "height": "10", "code": 0, "gas_wanted": "2000000",
             "gas_used": "100000", "claim_id": CLAIM, "tx_bytes": 500, "review_fee_uzrn": "1000000"}
    value.update(changes)
    return value


def history(packet, block="12"):
    return {"chain_id": w.CHAIN, "block_height": block, "record": {
        "claim_id": CLAIM, "claim": {"id": CLAIM, "fact_content": packet["intent"]["content"],
          "domain": packet["intent"]["domain"], "category": packet["intent"]["category"],
          "reasoning_trace": w.bound_reasoning(packet), "submitter": ADDRESS, "submitted_at_block": "10",
          "status": "CLAIM_STATUS_INCONCLUSIVE", "funding_terms": {
             "policy_version": 1, "kind": "CLAIM_FUNDING_KIND_REVIEW_FEE", "paid_amount": "1000000",
             "review_budget": "800000", "refundable_amount": "0", "retained_fee": "200000"}},
        "rounds": [{"id": "round-1", "claim_id": CLAIM, "reveals": [], "commits": [],
                    "result": "VERIFICATION_RESULT_INSUFFICIENT", "verifier_reward_settlement": {
                        "created_at_block": "11", "paid_at_block": "0", "payments": [], "withheld_total": "0"}}],
        "facts": []}, "related_claims": []}


class WorkflowTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.home = self.root / "participant"
        self.home.mkdir(mode=0o700)
        (self.home / "bin").mkdir(mode=0o700)
        self.client_dir = self.root / "package"
        self.client_dir.mkdir()
        self.client = self.client_dir / "shared-claims.py"
        self.marker = {"schema": "zerone-shared-participant/v1", "owner_uid": os.getuid(),
                       "chain_id": w.CHAIN, "address": ADDRESS, "allow_loopback_test": True,
                       "client_files": {}}
        for name in sorted(w.CLIENT_NAMES):
            raw = ("# fictional pinned client: " + name + "\n").encode()
            (self.client_dir / name).write_bytes(raw)
            self.marker["client_files"][name] = w.digest(raw)
        for name, key, raw in (("network.json", "descriptor_sha256", b'{"synthetic":true}'),
                               ("pinned-genesis.json", "genesis_sha256", b'{"chain_id":"zerone-dev-1"}'),
                               ("bin/zeroned", "binary_sha256", b"fixture, not an executable")):
            path = self.home / name
            path.write_bytes(raw)
            path.chmod(0o600)
            self.marker[key] = w.digest(raw)
        self.write_marker()
        self.export = fixture([contribution("original"), contribution("correction", supersedes="original")])
        self.raw = json.dumps(self.export, indent=2, ensure_ascii=False).encode() + b"\n"
        self.packet = w.prepare(self.raw, "original", INTENT)
        self.path = self.root / "workflow.json"
        self.path.write_bytes(j.canonical(self.packet) + b"\n")

    def write_marker(self):
        path = self.home / ".zerone-shared-participant.json"
        path.write_bytes(j.canonical(self.marker))
        path.chmod(0o600)

    def load(self):
        return j.parse_json(self.path.read_bytes(), w.MAX_PACKET_BYTES)

    def submitted(self, **changes):
        result = self.submit_with(lambda *_: receipt(**changes))
        self.assertEqual(result["submission"]["state"], "observed")
        return result

    def submit_with(self, runner):
        def preflight(home, client, command, arguments=()):
            if command == "snapshot":
                return {"schema": "zerone-shared-participant/v1", "chain_id": w.CHAIN,
                        "participant_address": ADDRESS, "descriptor_sha256": self.marker["descriptor_sha256"],
                        "loopback_test": True, "height": "9", "transactions": [], "claims": []}
            return runner(home, client, command, arguments)
        return w.submit(self.path, self.home, self.client, preflight)

    def snapshot(self, packet, observed=None, transactions=None):
        value = history(packet) if observed is None else observed
        return {"schema": "zerone-shared-participant/v1", "chain_id": w.CHAIN,
                "participant_address": ADDRESS, "descriptor_sha256": self.marker["descriptor_sha256"],
                "loopback_test": True, "height": value["block_height"],
                "transactions": [receipt()] if transactions is None else transactions,
                "claims": [{"claim_id": CLAIM, "history": value}]}

    def test_prepare_preserves_exact_bytes_selected_record_and_correction(self):
        self.assertEqual(self.packet["journal"]["export_utf8"].encode(), self.raw)
        self.assertEqual(self.packet["journal"]["export_sha256"], w.digest(self.raw))
        corrected = w.prepare(self.raw, "correction", INTENT)
        self.assertNotEqual(corrected["intent_sha256"], self.packet["intent_sha256"])
        original = j.parse_json(corrected["journal"]["export_utf8"].encode())
        self.assertEqual(original["entries"][1]["record"]["supersedes"], "original")
        self.assertEqual(original["entries"][0], self.export["entries"][0])
        self.assertIsNone(corrected["submission"])
        self.assertEqual(corrected["observations"], [])
        reasoning = w.bound_reasoning(self.packet)
        self.assertTrue(reasoning.startswith(INTENT["reasoning"] + "\n\nZerone research binding: "))
        self.assertIn(self.packet["intent_sha256"], reasoning)

    def test_intent_or_exact_export_tamper_rejected_even_equivalent_json(self):
        mutations = [lambda p: p["intent"].update(content="different assertion"),
                     lambda p: p["journal"].update(record_id="correction"),
                     lambda p: p["journal"].update(export_utf8=j.canonical(self.export).decode()),
                     lambda p: p["journal"].update(head_sha256="0" * 64),
                     lambda p: p.update(extra="unrecognized")]
        for mutation in mutations:
            candidate = copy.deepcopy(self.packet)
            mutation(candidate)
            with self.assertRaises(j.JournalError):
                w.validate_packet(candidate)

    def test_text_limits_include_native_binding_overhead_and_no_nul(self):
        for key, value in (("content", " "), ("reasoning", "a" * 8192),
                           ("domain", "x" * 129), ("method", "\0"), ("reasoning", "\ud800")):
            with self.subTest(key=key), self.assertRaises(j.JournalError):
                w.prepare(self.raw, "original", dict(INTENT, **{key: value}))
        self.assertLessEqual(len(w.bound_reasoning(self.packet).encode()), 8192)

    def test_submit_passes_exact_bound_text_as_argv_and_persists_started_first(self):
        calls = []
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(w.main(["inspect", "--packet", str(self.path)]), 0)
        previewed_payload = json.loads(output.getvalue())["prepared_payload"]
        def runner(home, client, command, arguments):
            calls.append(command)
            pending = self.load()
            self.assertEqual(pending["submission"], {"state": "started", "receipt": None})
            flags = dict(argument.split("=", 1) for argument in arguments)
            self.assertEqual(flags, {"--" + key: value for key, value in previewed_payload.items()})
            self.assertEqual(flags["--content"], INTENT["content"])
            self.assertEqual(flags["--reasoning"], w.bound_reasoning(pending))
            self.assertEqual(flags["--method"], "")
            return receipt()
        self.submit_with(runner)
        self.assertEqual(calls, ["submit"])
        before = self.path.read_bytes()
        with self.assertRaisesRegex(j.JournalError, "already started"):
            self.submit_with(runner)
        self.assertEqual(calls, ["submit"])
        self.assertEqual(self.path.read_bytes(), before)

    def test_crash_after_client_commit_recovers_read_only_without_second_submit(self):
        def crash(*_):
            raise w.Error("lost output after commit")
        with self.assertRaises(w.Error):
            self.submit_with(crash)
        self.assertEqual(self.load()["submission"], {"state": "started", "receipt": None})
        value = self.snapshot(self.load())
        commands = []
        def observe(_home, _client, command, arguments=()):
            commands.append(command)
            return value
        recovered = w.refresh(self.path, self.home, self.client, observe)
        self.assertEqual(commands, ["snapshot"])
        self.assertEqual(recovered["submission"]["receipt"]["claim_id"], CLAIM)
        self.assertEqual(len(recovered["observations"]), 1)
        self.assertEqual(w.refresh(self.path, self.home, self.client, observe), recovered)
        self.assertEqual(commands, ["snapshot", "snapshot"])

    def test_unresolved_result_never_creates_claim_or_replays(self):
        with self.assertRaises(w.Error):
            self.submit_with(lambda *_: (_ for _ in ()).throw(w.Error("uncertain")))
        pending = receipt(status="pending")
        for field in ("claim_id", "height", "code"):
            pending.pop(field)
        value = self.snapshot(self.load(), transactions=[pending])
        value["claims"] = []
        result = w.refresh(self.path, self.home, self.client, lambda *_: value)
        self.assertEqual(result["submission"], {"state": "started", "receipt": None})
        self.assertEqual(result["observations"], [])

    def test_checktx_or_failed_receipt_cannot_create_history(self):
        for status in ("pending", "failed", "committed-unresolved", "rejected"):
            value = copy.deepcopy(self.packet)
            value["participant"] = w.participant_binding(self.home, self.client)
            value["submission"] = {"state": "observed", "receipt": receipt(status=status)}
            raw = j.canonical(history(value))
            value["observations"] = [{"observed_at": "2026-09-23T12:00:00.000Z",
                                       "history_utf8": raw.decode(), "history_sha256": w.digest(raw)}]
            with self.subTest(status=status), self.assertRaisesRegex(w.Error, "committed"):
                w.validate_packet(value)
        for changes in ({"code": 1}, {"height": "0"}, {"claim_id": "not-generated"}, {"code": False}):
            with self.subTest(changes=changes), self.assertRaises(w.Error):
                w.validate_receipt(receipt(**changes))

    def test_zero_reviews_pending_payment_then_transferred_are_retained_not_inferred(self):
        packet = self.submitted()
        first = history(packet)
        result = w.refresh(self.path, self.home, self.client, lambda *_: self.snapshot(packet, first))
        second = copy.deepcopy(first)
        second["block_height"] = "14"
        second["record"]["rounds"][0]["verifier_reward_settlement"]["paid_at_block"] = "13"
        result = w.refresh(self.path, self.home, self.client, lambda *_: self.snapshot(packet, second))
        self.assertEqual(len(result["observations"]), 2)
        retained = [j.parse_json(row["history_utf8"].encode()) for row in result["observations"]]
        self.assertEqual(retained[0]["record"]["rounds"][0]["reveals"], [])
        self.assertEqual(retained[0]["record"]["rounds"][0]["verifier_reward_settlement"]["paid_at_block"], "0")
        self.assertEqual(retained[1]["record"]["rounds"][0]["verifier_reward_settlement"]["paid_at_block"], "13")

    def test_malformed_or_unrelated_history_cannot_replace_saved_observation(self):
        packet = self.submitted()
        mutations = [lambda h: h.update(chain_id="other-chain"),
                     lambda h: h.update(block_height="9"),
                     lambda h: h["record"]["claim"].update(reasoning_trace=INTENT["reasoning"]),
                     lambda h: h["record"]["claim"].update(submitter="zrn1" + "p" * 38),
                     lambda h: h["record"]["claim"].update(submitted_at_block="11"),
                     lambda h: h["record"]["claim"]["funding_terms"].update(paid_amount="1000001"),
                     lambda h: h["record"]["rounds"][0].update(claim_id="2" * 32),
                     lambda h: h["record"]["rounds"][0]["verifier_reward_settlement"].update(paid_at_block="99")]
        before = self.path.read_bytes()
        for mutate in mutations:
            value = history(packet)
            mutate(value)
            with self.assertRaises(w.Error):
                w.refresh(self.path, self.home, self.client, lambda *_: self.snapshot(packet, value))
            self.assertEqual(self.path.read_bytes(), before)

    def test_valid_fact_correction_graph_and_legacy_absence_are_preserved(self):
        packet = self.submitted()
        value = history(packet)
        del value["record"]["claim"]["funding_terms"]
        value["record"]["facts"] = [{"fact": {"id": "fact-1", "claim_id": CLAIM},
            "outgoing_relations": [{"source_fact_id": "fact-1", "target_fact_id": "fact-2", "type": "CONTRADICTS"}],
            "status_transitions": [{"fact_id": "fact-1", "previous_status": "VERIFIED", "new_status": "DISPUTED"}]}]
        value["related_claims"] = [{"record": {"claim_id": "2" * 32, "claim": None},
                                     "links": [{"field": "relations.contradicts", "target_id": "fact-1"}]}]
        result = w.refresh(self.path, self.home, self.client, lambda *_: self.snapshot(packet, value))
        self.assertEqual(j.parse_json(result["observations"][0]["history_utf8"].encode()), value)
        self.assertNotIn("funding_terms", value["record"]["claim"])

    def test_pin_changes_stop_before_client_execution_and_never_rewrite_home(self):
        original = self.client.read_bytes()
        before = self.path.read_bytes()
        self.client.write_bytes(original + b"# changed\n")
        with self.assertRaisesRegex(w.Error, "client source differs"):
            w.submit(self.path, self.home, self.client, lambda *_: self.fail("client executed"))
        self.assertEqual(self.path.read_bytes(), before)
        self.client.write_bytes(original)
        (self.home / "bin/zeroned").write_bytes(b"different")
        with self.assertRaisesRegex(w.Error, "control bytes differ"):
            w.submit(self.path, self.home, self.client, lambda *_: self.fail("client executed"))
        self.assertEqual(self.path.read_bytes(), before)

    def test_receipt_rejects_secret_or_unrecognized_fields(self):
        for field in ("salt", "tx_bytes_base64", "context", "receipt", "private_key"):
            with self.subTest(field=field), self.assertRaises(w.Error):
                self.submitted(**{field: "do-not-export"})
            self.assertNotIn("do-not-export", self.path.read_text())
            self.path.write_bytes(j.canonical(self.packet) + b"\n")

    def test_symlink_packet_lock_and_control_files_are_refused(self):
        link = self.root / "link.json"
        link.symlink_to(self.path)
        with self.assertRaises((w.Error, OSError)):
            w.submit(link, self.home, self.client, lambda *_: self.fail("client executed"))
        with w.packet_lock(self.path), self.assertRaisesRegex(w.Error, "another research workflow"):
            w.submit(self.path, self.home, self.client, lambda *_: self.fail("client executed"))

    def test_subprocess_never_uses_shell_or_prints_stderr(self):
        payload = "$(touch should-not-exist); `false`"
        def fake_run(argv, **kwargs):
            self.assertFalse(kwargs["shell"])
            self.assertEqual(argv[-2:], ["--content", payload])
            self.assertEqual(argv[1], "-I")
            kwargs["stderr"].write(b"private salt must never become public")
            return subprocess.CompletedProcess(argv, 1)
        with patch.object(w.subprocess, "run", fake_run):
            with self.assertRaises(w.Error) as caught:
                w.run_client(self.home, self.client, "submit", ["--content", payload])
        self.assertNotIn("private salt", str(caught.exception))

    def test_ambiguous_recovery_refuses_and_does_not_rebind(self):
        with self.assertRaises(w.Error):
            self.submit_with(lambda *_: (_ for _ in ()).throw(w.Error("lost stdout")))
        value = self.snapshot(self.load(), transactions=[receipt(), receipt(id="b" * 32, txhash="B" * 64)])
        before = self.path.read_bytes()
        with self.assertRaisesRegex(w.Error, "ambiguous"):
            w.refresh(self.path, self.home, self.client, lambda *_: value)
        self.assertEqual(self.path.read_bytes(), before)

    def test_observation_digest_height_and_inventory_limits(self):
        packet = self.submitted()
        result = w.refresh(self.path, self.home, self.client, lambda *_: self.snapshot(packet))
        forged = copy.deepcopy(result)
        forged["observations"][0]["history_utf8"] += " "
        with self.assertRaisesRegex(w.Error, "digest mismatch"):
            w.validate_packet(forged)
        forged = copy.deepcopy(result)
        forged["observations"] *= w.MAX_OBSERVATIONS + 1
        with self.assertRaisesRegex(w.Error, "inventory"):
            w.validate_packet(forged)
        later = history(packet, "11")
        with self.assertRaisesRegex(w.Error, "backwards"):
            w.refresh(self.path, self.home, self.client, lambda *_: self.snapshot(packet, later))

    def test_fresh_copy_recovers_existing_binding_without_invoking_submit(self):
        packet = self.submitted()
        self.path.write_bytes(j.canonical(self.packet) + b"\n")  # Original downloaded draft.
        calls = []
        def runner(_home, _client, command, arguments=()):
            calls.append(command)
            self.assertEqual(command, "snapshot", "fresh copy must not sign again")
            return self.snapshot(packet)
        result = w.submit(self.path, self.home, self.client, runner)
        self.assertEqual(calls, ["snapshot"])
        self.assertEqual(result["submission"]["receipt"], receipt())
        self.assertEqual(len(result["observations"]), 1)

    def test_failed_preflight_does_not_mark_a_fresh_draft_started(self):
        before = self.path.read_bytes()
        with self.assertRaises(w.Error):
            w.submit(self.path, self.home, self.client, lambda *_: (_ for _ in ()).throw(w.Error("offline")))
        self.assertEqual(self.path.read_bytes(), before)

    def test_explicit_resume_after_crash_before_launch_uses_the_same_payload(self):
        with self.assertRaises(w.Error):
            self.submit_with(lambda *_: (_ for _ in ()).throw(w.Error("crashed before client launch")))
        original = self.load()
        calls = []
        def runner(_home, _client, command, arguments=()):
            calls.append(command)
            if command == "snapshot":
                value = self.snapshot(original, transactions=[])
                value["claims"] = []
                return value
            flags = dict(item.split("=", 1) for item in arguments)
            self.assertEqual(flags["--reasoning"], w.bound_reasoning(original))
            return receipt()
        result = w.resume(self.path, self.home, self.client, runner)
        self.assertEqual(calls, ["snapshot", "submit"])
        self.assertEqual(result["submission"]["receipt"], receipt())
        self.assertEqual(result["participant"], original["participant"])
        with self.assertRaises(w.Error):
            w.resume(self.path, self.home, self.client, runner)

    def test_explicit_resume_recovers_committed_without_second_submit(self):
        with self.assertRaises(w.Error):
            self.submit_with(lambda *_: (_ for _ in ()).throw(w.Error("lost stdout")))
        before = self.load()
        calls = []
        def runner(_home, _client, command, arguments=()):
            calls.append(command)
            self.assertEqual(command, "snapshot")
            return self.snapshot(before)
        result = w.resume(self.path, self.home, self.client, runner)
        self.assertEqual(calls, ["snapshot"])
        self.assertEqual(result["submission"]["receipt"], receipt())

    def test_resume_does_not_authorize_a_draft_or_another_participant(self):
        with self.assertRaises(w.Error):
            w.resume(self.path, self.home, self.client, lambda *_: self.fail("client executed"))
        with self.assertRaises(w.Error):
            self.submit_with(lambda *_: (_ for _ in ()).throw(w.Error("interrupted")))
        self.marker["address"] = "zrn1" + "p" * 38
        self.write_marker()
        with self.assertRaisesRegex(w.Error, "differs"):
            w.resume(self.path, self.home, self.client, lambda *_: self.fail("client executed"))

    def test_inspect_previews_exact_payload_and_local_correction_without_admission(self):
        before = self.path.read_bytes()
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(w.main(["inspect", "--packet", str(self.path)]), 0)
        summary = json.loads(output.getvalue())
        self.assertEqual(summary["prepared_payload"],
                         dict(INTENT, reasoning=w.bound_reasoning(self.packet)))
        self.assertIn("admission_preview", summary)
        preview = summary["admission_preview"]
        self.assertEqual(preview["scope"], "complete_embedded_journal")
        self.assertEqual(preview["journal_entry_count"], 2)
        self.assertEqual(preview["local_only"]["supersessions"],
                         [{"id": "correction", "kind": "contribution", "supersedes": "original"}])
        self.assertFalse(preview["network_checked"])
        self.assertEqual(preview["evidence_availability"], "not_checked")
        self.assertIsNone(summary["submission"])
        self.assertEqual(self.path.read_bytes(), before)

    def test_inspect_inventories_all_local_graph_and_evidence_without_translation(self):
        url_only = {"label": "Unfetched URL", "url": "https://example.org/source", "sha256": None}
        hash_only = {"label": "Unavailable artifact", "url": None, "sha256": "a" * 64}
        both = dict(url_only, sha256="b" * 64)
        records = [contribution("original", evidence=[url_only, hash_only]),
                   contribution("source", evidence=[both]),
                   contribution("correction", supersedes="original"),
                   relation("supports", "original", "source", evidence=[hash_only]),
                   relation("cites", "original", "source", "cites"),
                   relation("revised-link", "correction", "source", "requires", supersedes="supports"),
                   dict(concern("concern", "original"), evidence=[url_only]),
                   dict(concern("revised-concern", "correction"), supersedes="concern"),
                   dict(assessment("assessment", "supports"), evidence=[both]),
                   assessment("disagreement", "supports", "disputed"),
                   dict(assessment("revised-assessment", "concern", "addressed"), supersedes="assessment")]
        raw = j.canonical(fixture(records)) + b"\n"
        packet = w.prepare(raw, "original", INTENT)
        self.path.write_bytes(j.canonical(packet) + b"\n")
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(w.main(["inspect", "--packet", str(self.path)]), 0)
        summary = json.loads(output.getvalue())
        preview = summary["admission_preview"]
        local = preview["local_only"]
        self.assertEqual(preview["journal_entry_count"], len(records))
        self.assertEqual(set(local), {"relations", "supersessions", "concerns", "assessments", "evidence_references"})
        for kind, fields in (("relation", ("id", "from_id", "to_id", "relation_type")),
                             ("concern", ("id", "target_id", "category", "attributed_to")),
                             ("assessment", ("id", "target_id", "disposition", "attributed_to"))):
            self.assertEqual(local[kind + "s"], [{key: record[key] for key in fields}
                                                  for record in records if record["kind"] == kind])
        self.assertEqual(local["supersessions"], [{key: record[key] for key in ("id", "kind", "supersedes")}
                                                  for record in records if "supersedes" in record])
        self.assertEqual(local["evidence_references"], [{"record_id": record["id"], "references": record["evidence"]}
                                                       for record in records if record["evidence"]])
        # Unselected and corrected records remain visible; no endpoint-pair deduplication.
        self.assertEqual([row["id"] for row in local["relations"]], ["supports", "cites", "revised-link"])
        self.assertEqual(summary["prepared_payload"], dict(INTENT, reasoning=w.bound_reasoning(packet)))
        self.assertNotIn(url_only["url"], summary["prepared_payload"]["reasoning"])
        self.assertEqual(self.load(), packet)

    def test_inspect_reports_limits_and_never_refreshes_a_saved_receipt(self):
        packet = self.submitted()
        before = {path.relative_to(self.root): path.read_bytes()
                  for path in self.root.rglob("*") if path.is_file()}
        output = io.StringIO()
        with contextlib.ExitStack() as stack:
            for name in ("participant_binding", "run_client", "submit", "resume", "refresh", "packet_lock", "save_packet"):
                stack.enter_context(patch.object(w, name, side_effect=AssertionError("inspect reached " + name)))
            stack.enter_context(patch("socket.socket", side_effect=AssertionError("inspect reached network")))
            stack.enter_context(patch.object(w.subprocess, "run", side_effect=AssertionError("inspect launched a process")))
            reads = stack.enter_context(patch.object(w, "read_bytes", wraps=w.read_bytes))
            stack.enter_context(contextlib.redirect_stdout(output))
            self.assertEqual(w.main(["inspect", "--packet", str(self.path)]), 0)
            reads.assert_called_once_with(str(self.path), w.MAX_PACKET_BYTES)
        summary = json.loads(output.getvalue())
        self.assertEqual(summary["submission"], packet["submission"])
        self.assertEqual(summary["observation_count"], 0)
        preview = summary["admission_preview"]
        self.assertIn("limitations", preview)
        limits = " ".join(preview["limitations"])
        self.assertIn("not uploaded", limits)
        self.assertIn("not translated", limits)
        self.assertIn("no source is fetched", limits)
        self.assertIn("does not check network admission", limits)
        self.assertIn("does not refresh saved observations", limits)
        self.assertFalse(preview["network_checked"])
        self.assertEqual(preview["evidence_availability"], "not_checked")
        self.assertEqual(before, {path.relative_to(self.root): path.read_bytes()
                                  for path in self.root.rglob("*") if path.is_file()})

    def test_inspect_refuses_invalid_packets_before_emitting_any_preview(self):
        mutations = [lambda p: p["intent"].update(content="A different but well-formed assertion."),
                     lambda p: p["journal"].update(export_utf8=p["journal"]["export_utf8"] + " "),
                     lambda p: p["journal"].update(record_id="missing"),
                     lambda p: p.update(observations=[{}])]
        candidates = [b"{", b'{"schema":"x","schema":"y"}', b"[]"]
        for mutate in mutations:
            candidate = copy.deepcopy(self.packet)
            mutate(candidate)
            candidates.append(j.canonical(candidate))
        for raw in candidates:
            self.path.write_bytes(raw)
            output, errors = io.StringIO(), io.StringIO()
            with contextlib.redirect_stdout(output), contextlib.redirect_stderr(errors), \
                    patch.object(w, "_admission_preview", side_effect=AssertionError("invalid packet reached preview")):
                self.assertEqual(w.main(["inspect", "--packet", str(self.path)]), 1)
            self.assertEqual(output.getvalue(), "")
            self.assertIn("research workflow refused:", errors.getvalue())
            self.assertEqual(self.path.read_bytes(), raw)

    def test_inspect_cli_is_repeatable_without_a_participant_or_file_changes(self):
        # The actual CLI runs without our synthetic participant/client fixtures.
        root = self.root / "offline"
        root.mkdir()
        packet = root / "packet.json"
        packet.write_bytes(j.canonical(self.packet) + b"\n")
        before = packet.read_bytes()
        outputs = []
        for _ in range(2):
            result = subprocess.run([sys.executable, "-B", str(Path(w.__file__).resolve()),
                                     "inspect", "--packet", str(packet)], cwd=root,
                                    capture_output=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stderr, b"")
            outputs.append(result.stdout)
            self.assertEqual(set(root.iterdir()), {packet})
            self.assertEqual(packet.read_bytes(), before)
        self.assertEqual(outputs[0], outputs[1])
        preview = json.loads(outputs[0])["admission_preview"]
        self.assertEqual(preview["local_only"]["evidence_references"], [])
        self.assertEqual(preview["local_only"]["relations"], [])
        self.assertEqual(preview["local_only"]["concerns"], [])
        self.assertEqual(preview["local_only"]["assessments"], [])
        linked = root / "linked.json"
        linked.symlink_to(packet)
        result = subprocess.run([sys.executable, "-B", str(Path(w.__file__).resolve()),
                                 "inspect", "--packet", str(linked)], cwd=root,
                                capture_output=True, check=False)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(packet.read_bytes(), before)

    def test_inspect_full_capacity_has_no_hidden_cutoff_or_input_mutation(self):
        records = [contribution("item-" + str(n)) for n in range(j.MAX_ENTRIES)]
        export = j.make_export(records, collection_id=self.export["header"]["collection_id"],
                               created_at=self.export["header"]["created_at"],
                               recorded_at=[self.export["header"]["created_at"]] * len(records))
        packet = w.prepare(j.canonical(export), records[-1]["id"], INTENT)
        self.path.write_bytes(j.canonical(packet) + b"\n")
        before = self.path.read_bytes()
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(w.main(["inspect", "--packet", str(self.path)]), 0)
        summary = json.loads(output.getvalue())
        self.assertEqual(summary["record_id"], records[-1]["id"])
        self.assertEqual(summary["admission_preview"]["journal_entry_count"], j.MAX_ENTRIES)
        self.assertEqual(self.path.read_bytes(), before)

    def test_offline_cli_prepare_and_inspect_require_new_output(self):
        export_file = self.root / "journal-export.json"
        export_file.write_bytes(self.raw)
        target = self.root / "cli-workflow.json"
        argv = ["prepare", "--journal", str(export_file), "--record", "original", "--output", str(target)]
        for key, value in INTENT.items():
            argv.extend(["--" + key, value])
        with patch.object(w, "run_client", side_effect=AssertionError("offline command reached client")):
            self.assertEqual(w.main(argv), 0)
            self.assertEqual(w.main(["inspect", "--packet", str(target)]), 0)
            before = target.read_bytes()
            self.assertEqual(w.main(argv), 1)
            self.assertEqual(target.read_bytes(), before)


if __name__ == "__main__":
    unittest.main()
