#!/usr/bin/env python3
"""Synthetic public snapshots and real loopback request-boundary regressions."""

import base64
import copy
import hashlib
import http.client
import importlib.util
import json
from pathlib import Path
import re
import socket
import threading
import unittest
from unittest import mock


SPEC = importlib.util.spec_from_file_location("claim_workflow_view", Path(__file__).with_name("claim-workflow-view.py"))
view = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(view)


def snapshot():
    """Public records include numeric protobuf enums and hostile-but-literal text."""
    claim_id = "claim:example"
    return {
        "schema": "zerone-claim-workflow-v1", "chain_id": "zerone-local-view-test", "height": "42",
        "local_only": True, "notice": "One operator; local test funds only.",
        "claims": [{"claim_id": claim_id, "history": {
            "chain_id": "zerone-local-view-test", "block_height": "42",
            "record": {"claim_id": claim_id, "claim": {
                "id": claim_id, "submitter": "zrn1author", "status": 6,
                "fact_content": "<img src=x onerror=alert(1)>",
                "reasoning_trace": "</script><script>untrusted()</script>", "stake": "200000",
            }, "rounds": [{"id": "round:example", "claim_id": claim_id, "phase": 4, "verdict": 1,
                "commit_deadline": "20", "reveal_deadline": "30", "aggregation_deadline": "40",
                "commitment_scheme": 2, "review_policy_version": 1,
                "reveals": [{"verifier": "zrn1reviewer", "vote": "SOUND", "confidence": "700000",
                    "revealed_at_block": "27", "salt": "cHVibGlzaGVk",
                    "attestation": {"method_id": "finite-check", "reason": "Checked only n=0…3.",
                        "scope": "A finite range, not a universal proof.", "evidence_ids": ["data:text/html,example"]}}],
            }], "facts": [{"fact": {"id": "fact:example", "status": 3, "content": "Recorded assessment"},
                "outgoing_relations": [{"source_fact_id": "fact:example", "target_fact_id": "fact:prior",
                    "relation": 2, "inference": 4, "inference_strength_bps": "5000",
                    "created_at_block": "41", "creator": "zrn1author", "method_id": "finite-check"}],
                "incoming_relations": [], "status_transitions": []}], "missing_round_ids": ["round:missing"]},
            "related_claims": [{"record": {"claim_id": "claim:challenge", "rounds": [], "facts": []},
                "links": [{"field": "relations.contradicts", "target_id": "fact:example"}]}],
        }}],
        "transactions": [
            {"action": "submit", "actor": "user", "txhash": "ABC", "height": "12", "code": 0,
             "status": "committed", "gas_wanted": "250000", "gas_used": "123000", "tx_bytes": 500,
             "review_fee_uzrn": "200000"},
            {"action": "reveal", "actor": "reviewer1", "txhash": "DEF", "status": "rejected"},
        ],
        "pending_reviews": [{"claim_id": claim_id, "round_id": "round:example", "actor": "reviewer2",
            "status": "awaiting-reveal", "commit_deadline": "20", "reveal_deadline": "30"}],
    }


class SnapshotTests(unittest.TestCase):
    def test_numeric_proto_snapshot_retains_exact_public_content(self):
        expected = snapshot()
        actual = json.loads(view._snapshot_bytes(expected))
        self.assertEqual(actual, expected)
        self.assertEqual(actual["claims"][0]["history"]["record"]["rounds"][0]["phase"], 4)
        self.assertEqual(actual["transactions"][1]["status"], "rejected")

    def test_mixed_identity_context_and_secret_summary_refused(self):
        mutations = {
            "public chain": lambda s: s.update(chain_id="zerone-1"),
            "local flag": lambda s: s.update(local_only=False),
            "unknown schema": lambda s: s.update(schema="future"),
            "extra root secret": lambda s: s.update(private_reviews=[]),
            "history height": lambda s: s["claims"][0]["history"].update(block_height="41"),
            "history chain": lambda s: s["claims"][0]["history"].update(chain_id="zerone-local-other"),
            "record ID": lambda s: s["claims"][0]["history"]["record"].update(claim_id="another"),
            "empty ID": lambda s: s["claims"][0].update(claim_id=""),
            "duplicate ID": lambda s: s["claims"].append(copy.deepcopy(s["claims"][0])),
            "private salt": lambda s: s["pending_reviews"][0].update(salt="private"),
            "private reason": lambda s: s["pending_reviews"][0].update(reason="private"),
            "private signed bytes": lambda s: s["transactions"][0].update(tx_bytes_base64="c2VjcmV0"),
            "nested receipt": lambda s: s["transactions"][0].update(status={"secret": "value"}),
            "nonfinite": lambda s: s["claims"][0]["history"]["record"].update(future=float("nan")),
        }
        for name, mutate in mutations.items():
            with self.subTest(name=name):
                data = snapshot()
                mutate(data)
                with self.assertRaises((ValueError, TypeError)):
                    view._snapshot_bytes(data)

    def test_decimal_height_preserves_large_values_without_js_rounding(self):
        for bad in [True, None, -1, "-1", "01", "1.0", "1e2", " 1", str(2**64), 2**53]:
            with self.subTest(value=bad), self.assertRaises(ValueError):
                view._height(bad)
        self.assertEqual(view._height(str(2**64 - 1)), str(2**64 - 1))
        self.assertEqual(view._height(0), "0")

    def test_refuses_excess_records_and_encoded_bytes_without_truncation(self):
        data = snapshot()
        data["transactions"] = [{}] * (view.MAX_RECORDS + 1)
        with self.assertRaises(ValueError):
            view._snapshot_bytes(data)
        data = snapshot()
        raw = view._snapshot_bytes(data)
        with mock.patch.object(view, "MAX_RESPONSE_BYTES", len(raw) - 1), self.assertRaises(ValueError):
            view._snapshot_bytes(data)
        with mock.patch.object(view, "MAX_RESPONSE_BYTES", len(raw)):
            self.assertEqual(view._snapshot_bytes(data), raw)

    def test_embedded_enum_maps_match_proto_domains_including_numeric_cli_values(self):
        source = (Path(__file__).parents[1] / "proto/zerone/knowledge/v1/types.proto").read_text()
        domains = {"claim": "ClaimStatus", "phase": "VerificationPhase", "verdict": "Verdict",
                   "fact": "FactStatus", "relation": "RelationType", "inference": "InferenceType"}
        for domain, proto_name in domains.items():
            with self.subTest(domain=domain):
                js = re.search(r"\b" + domain + r":\['([^']+)',\[([^]]+)\]\]", view.SCRIPT)
                self.assertIsNotNone(js)
                names = re.findall(r"'([^']+)'", js[2])
                block = re.search(r"enum " + proto_name + r"\s*\{([^}]+)\}", source)[1]
                pairs = re.findall(r"\b([A-Z][A-Z_]+)\s*=\s*(\d+)\s*;", block)
                self.assertEqual({int(number): name for name, number in pairs},
                                 {i: js[1] + name for i, name in enumerate(names)})
        # A phase 4 is COMPLETE, whereas a verdict 4 is MALFORMED: never share one numeric map.
        self.assertIn("label('phase',round.phase)", view.SCRIPT)
        self.assertIn("label('verdict',round.verdict)", view.SCRIPT)
        self.assertIn("Unknown ${kind} value", view.SCRIPT)


class HTTPTests(unittest.TestCase):
    def setUp(self):
        self.calls = 0
        self.provider = snapshot

        def callback():
            self.calls += 1
            return self.provider()

        self.server = view._make_server(callback, 0)
        self.thread = threading.Thread(target=self.server.serve_forever, kwargs={"poll_interval": .01})
        self.thread.start()
        self.host = f"127.0.0.1:{self.server.server_port}"

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(2)
        self.assertFalse(self.thread.is_alive())

    def request(self, target="/api/history", method="GET", headers=None, body=None):
        connection = http.client.HTTPConnection("127.0.0.1", self.server.server_port, timeout=3)
        try:
            connection.request(method, target, body=body, headers=headers or {})
            result = connection.getresponse()
            return result.status, dict(result.getheaders()), result.read()
        finally:
            connection.close()

    def raw_request(self, lines):
        with socket.create_connection(("127.0.0.1", self.server.server_port), timeout=3) as connection:
            connection.sendall(("\r\n".join(lines) + "\r\n\r\n").encode())
            response = http.client.HTTPResponse(connection)
            response.begin()
            return response.status, response.read()

    def test_loopback_page_api_and_csp_hashes(self):
        self.assertEqual(self.server.server_address[0], "127.0.0.1")
        status, headers, page = self.request("/")
        self.assertEqual(status, 200)
        self.assertEqual(self.calls, 0)
        self.assertEqual(page, view.PAGE)
        self.assertNotIn("Access-Control-Allow-Origin", headers)
        for text in (view.STYLE, view.SCRIPT):
            expected = base64.b64encode(hashlib.sha256(text.encode()).digest()).decode()
            self.assertIn("'sha256-" + expected + "'", headers["Content-Security-Policy"])
        self.assertNotIn("'unsafe-inline'", headers["Content-Security-Policy"])
        for key, expected in {"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff",
                              "X-Frame-Options": "DENY", "Referrer-Policy": "no-referrer"}.items():
            self.assertEqual(headers[key], expected)
        status, headers, raw = self.request(headers={"Origin": "http://" + self.host})
        self.assertEqual(status, 200)
        self.assertEqual(json.loads(raw), snapshot())
        self.assertEqual(self.calls, 1)
        self.assertNotIn(b"<img src=x", page)
        self.assertIn(b"<img src=x", raw)
        self.assertIn("node.textContent=text", view.SCRIPT)
        self.assertNotRegex(view.SCRIPT, r"innerHTML|outerHTML|insertAdjacentHTML|document\.write|\beval\(")
        self.assertNotRegex(page.decode(), r'<(?:script|link|img)[^>]+(?:src|href)="https?://')

    def test_foreign_hosts_origins_and_duplicate_headers_refuse_without_callback(self):
        for headers in [{"Host": "evil.example"}, {"Host": "localhost:" + str(self.server.server_port)},
                        {"Origin": "http://evil.example"}, {"Origin": "null"},
                        {"Origin": "http://127.0.0.1:1"}, {"Sec-Fetch-Site": "cross-site"}]:
            with self.subTest(headers=headers):
                self.assertEqual(self.request(headers=headers)[0], 403)
        for lines in [[], ["Host: " + self.host, "Host: " + self.host],
                      ["Host: " + self.host, "Origin: http://" + self.host, "Origin: http://" + self.host]]:
            with self.subTest(lines=lines):
                self.assertEqual(self.raw_request(["GET /api/history HTTP/1.1"] + lines)[0], 403)
        self.assertEqual(self.calls, 0)

    def test_unexpected_paths_and_methods_have_no_callback(self):
        for path in ["/api/history?x=1", "/api/history/", "/etc/passwd", "/%2e%2e/", "//api/history",
                     "http://" + self.host + "/api/history", "/favicon.ico"]:
            with self.subTest(path=path):
                self.assertEqual(self.request(path)[0], 404)
        for method in ["HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "TRACE", "CONNECT"]:
            with self.subTest(method=method):
                self.assertEqual(self.request(method=method)[0], 405)
        self.assertEqual(self.request(method="UNSUPPORTED")[0], 501)
        self.assertEqual(self.calls, 0)

    def test_request_body_and_conflicting_framing_refused(self):
        self.assertEqual(self.request(body=b"x")[0], 400)
        for extra in [["Transfer-Encoding: chunked"], ["Content-Length: 0", "Content-Length: 0"],
                      ["Content-Length: -1"], ["Content-Length: 1"]]:
            self.assertEqual(self.raw_request(["GET /api/history HTTP/1.1", "Host: " + self.host] + extra)[0], 400)
        self.assertEqual(self.calls, 0)

    def test_callback_failure_is_generic_and_next_refresh_can_recover(self):
        def broken():
            raise RuntimeError("PRIVATE_REVIEW_SECRET")

        self.provider = broken
        status, _, raw = self.request()
        self.assertEqual(status, 503)
        self.assertNotIn(b"PRIVATE_REVIEW_SECRET", raw)
        self.provider = lambda: {**snapshot(), "chain_id": "zerone-1"}
        self.assertEqual(self.request()[0], 503)
        self.provider = snapshot
        self.assertEqual(self.request()[0], 200)

    def test_concurrent_refresh_is_refused_without_running_second_callback(self):
        started, release = threading.Event(), threading.Event()

        def slow():
            started.set()
            if not release.wait(3):
                raise RuntimeError("test callback timed out")
            return snapshot()

        self.provider = slow
        first = []
        request_thread = threading.Thread(target=lambda: first.append(self.request()))
        request_thread.start()
        try:
            self.assertTrue(started.wait(2))
            self.assertEqual(self.request()[0], 503)
            self.assertEqual(self.calls, 1)
        finally:
            release.set()
            request_thread.join(3)
        self.assertEqual(first[0][0], 200)


class LifecycleTests(unittest.TestCase):
    def test_invalid_ports_and_callback_refused(self):
        for port in [-1, 65536, True, "8080"]:
            with self.subTest(port=port), self.assertRaises(ValueError):
                view._make_server(snapshot, port)
        with self.assertRaises(ValueError):
            view._make_server(None, 0)

    def test_serve_interrupt_closes_owned_server(self):
        server = mock.MagicMock()
        server.__enter__.return_value = server
        server.server_port = 12345
        server.serve_forever.side_effect = KeyboardInterrupt
        with mock.patch.object(view, "_make_server", return_value=server) as make_server, mock.patch("builtins.print"):
            view.serve(snapshot, 0)
        make_server.assert_called_once_with(snapshot, 0)
        server.__exit__.assert_called_once()


if __name__ == "__main__":
    unittest.main()
