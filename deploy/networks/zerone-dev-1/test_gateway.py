#!/usr/bin/env python3
"""Failure-path tests for public development transport and funding."""
import base64
import copy
import datetime
import hashlib
import http.client
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import threading
from types import SimpleNamespace
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("dev_gateway", Path(__file__).with_name("gateway.py"))
g = importlib.util.module_from_spec(spec)
spec.loader.exec_module(g)


def address(number):
    alphabet = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
    # A20-byte number is exactly32 Bech32 data words.
    words = [(number >> shift) & 31 for shift in reversed(range(0, 160, 5))]
    values = [ord(c) >> 5 for c in "zrn"] + [0] + [ord(c) & 31 for c in "zrn"] + words + [0] * 6
    check = 1
    for value in values:
        top = check >> 25
        check = ((check & 0x1ffffff) << 5) ^ value
        for index, generator in enumerate((0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3)):
            if (top >> index) & 1:
                check ^= generator
    check ^= 1
    return "zrn1" + "".join(alphabet[value] for value in words + [(check >> (5 * (5 - i))) & 31 for i in range(6)])


def grant(number=1, status="prepared", now=1):
    raw = f"signed-development-bank-tx-{number}".encode()
    value = {"amount_uzrn": str(g.AMOUNT), "status": status, "created_unix": now,
        "ip_hash": "a" * 64, "txhash": hashlib.sha256(raw).hexdigest().upper(),
        "tx_bytes_base64": base64.b64encode(raw).decode()}
    if status == "committed":
        value.update(height="9", code=0)
    return value


class FakeNode:
    genesis_hash = "b" * 64
    rpc = "http://127.0.0.1:26657"
    descriptor_bytes = b'{"chain_id":"zerone-dev-1"}'
    genesis = b'{"chain_id":"zerone-dev-1","app_state":{}}'

    def __init__(self, home):
        self.home = home
        self.calls = []
        self.observations = {}
        self.check_code = 0
        self.commit_on_broadcast = True
        self.broadcast_error = False

    def ready(self):
        return 10

    def upstream(self, path="/", body=None):
        self.calls.append((path, body))
        if body:
            value = g.parse(body)
            if value.get("method") == "tx":
                digest = base64.b64decode(value["params"]["hash"]).hex().upper()
                return g.canonical(self.observations.get(digest, {"error": {"code": -32603, "data": f"tx ({digest}) not found"}}))
        return b'{"result":{"ok":true}}'

    def call(self, method, params):
        self.calls.append((method, params))
        if self.broadcast_error:
            raise g.GatewayError("Lost broadcast response")
        digest = hashlib.sha256(base64.b64decode(params["tx"])).hexdigest().upper()
        if self.commit_on_broadcast:
            self.observations[digest] = {"result": {"hash": digest, "height": "12", "tx_result": {"code": 0}}}
        return {"hash": digest, "code": self.check_code}

    def history(self, identifier):
        return {"chain_id": g.CHAIN, "block_height": "10", "record": {"claim_id": identifier}}


class FaucetTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.home = Path(self.temporary.name)
        directory = self.home / "dev-faucet"
        directory.mkdir(mode=0o700)
        self.state = {"schema": g.SCHEMA, "chain_id": g.CHAIN, "genesis_sha256": "b" * 64,
                      "ip_salt": "c" * 64, "grants": {}}
        g.save(directory / "state.json", self.state)
        self.node = FakeNode(self.home)
        self.faucet = g.Faucet(self.node)

    def tearDown(self):
        os.close(self.faucet.lock_fd)
        self.temporary.cleanup()

    def prepare(self, account, ip_hash, now):
        value = grant(int(now))
        value.update(ip_hash=ip_hash, created_unix=now)
        return value

    def test_checksums_not_address_shape(self):
        self.assertTrue(g.valid_address(address(1)))
        self.assertFalse(g.valid_address(address(1)[:-1] + ("q" if address(1)[-1] != "q" else "p")))
        self.assertFalse(g.valid_address(address(1).upper()))
        with self.assertRaises(g.GatewayError):
            self.faucet.request("zrn1" + "q" * 38, "127.0.0.1")
        self.assertEqual(self.node.calls, [])

    def test_state_reserved_before_network_broadcast(self):
        real_call = self.node.call
        def call(method, params):
            disk = g.parse((self.home / "dev-faucet/state.json").read_bytes())
            saved = disk["grants"][address(1)]
            self.assertEqual(saved["status"], "broadcasting")
            self.assertEqual(saved["tx_bytes_base64"], params["tx"])
            return real_call(method, params)
        with patch.object(self.faucet, "prepare", side_effect=self.prepare), patch.object(self.node, "call", side_effect=call):
            result = self.faucet.request(address(1), "127.0.0.1")
        self.assertTrue(result["committed"])
        self.assertEqual(result["height"], "12")
        self.assertNotIn("tx_bytes_base64", result)
        self.assertNotIn("ip_hash", result)

    def test_checktx_alone_is_pending(self):
        self.node.commit_on_broadcast = False
        with patch.object(self.faucet, "prepare", side_effect=self.prepare), patch.object(g.time, "sleep"):
            result = self.faucet.request(address(1), "127.0.0.1")
        self.assertFalse(result["committed"])
        self.assertEqual(result["status"], "pending")
        self.assertNotIn("height", result)

    def test_lost_broadcast_blocks_new_grants_and_same_address_replays_exactly(self):
        self.node.broadcast_error = True
        with patch.object(self.faucet, "prepare", side_effect=self.prepare) as prepare:
            with self.assertRaises(g.GatewayError):
                self.faucet.request(address(1), "127.0.0.1")
            saved = copy.deepcopy(self.faucet.state["grants"][address(1)])
            with self.assertRaises(g.GatewayError):
                self.faucet.request(address(2), "127.0.0.1")
            self.node.broadcast_error = False
            result = self.faucet.request(address(1), "127.0.0.1")
            self.assertEqual(prepare.call_count, 1)
        self.assertEqual(result["txhash"], saved["txhash"])
        self.assertEqual([value[1]["tx"] for value in self.node.calls if value[0] == "broadcast_tx_sync"], [saved["tx_bytes_base64"]] * 2)

    def test_repeat_success_returns_existing_grant_without_signing(self):
        value = grant(status="committed")
        self.faucet.state["grants"][address(1)] = value
        with patch.object(self.faucet, "prepare") as prepare:
            result = self.faucet.request(address(1), "127.0.0.1")
        prepare.assert_not_called()
        self.assertEqual(result["txhash"], value["txhash"])

    def test_duplicate_checktx_cannot_terminally_reject_original_pending_grant(self):
        saved = grant(status="pending")
        self.faucet.state["grants"][address(1)] = saved
        self.node.commit_on_broadcast = False
        self.node.check_code = 19  # Duplicate/cache/sequence CheckTx refusal.
        with patch.object(g.time, "sleep"), patch.object(self.faucet, "prepare") as prepare:
            first = self.faucet.request(address(1), "127.0.0.1")
            self.assertEqual(first["status"], "pending")
            self.assertFalse(first["committed"])
            with self.assertRaises(g.GatewayError):
                self.faucet.request(address(2), "127.0.0.1")
            self.node.observations[saved["txhash"]] = {"result": {
                "hash": saved["txhash"], "height": "16", "tx_result": {"code": 0}}}
            settled = self.faucet.request(address(1), "127.0.0.1")
        prepare.assert_not_called()
        self.assertTrue(settled["committed"])
        self.assertEqual(settled["height"], "16")
        self.assertEqual(settled["txhash"], saved["txhash"])
        self.assertEqual(len([value for value in self.node.calls if value[0] == "broadcast_tx_sync"]), 1)

    def test_cap_checked_before_transfer_and_never_overshoots(self):
        self.faucet.state["grants"] = {address(i): grant(i, "committed") for i in range(100)}
        with patch.object(self.faucet, "prepare") as prepare, self.assertRaises(g.GatewayError):
            self.faucet.request(address(101), "127.0.0.1")
        prepare.assert_not_called()

    def test_hourly_limits_checked_before_transfer(self):
        now = int(g.time.time())
        self.faucet.state["grants"] = {address(i): grant(i, "committed", now) for i in range(10)}
        with patch.object(self.faucet, "prepare") as prepare, self.assertRaises(g.GatewayError):
            self.faucet.request(address(11), "127.0.0.1")
        prepare.assert_not_called()

    def test_persistence_failure_blocks_broadcast_and_subsequent_requests(self):
        with patch.object(self.faucet, "prepare", side_effect=self.prepare), patch.object(g, "save", side_effect=OSError("disk full")):
            with self.assertRaises(OSError):
                self.faucet.request(address(1), "127.0.0.1")
        self.assertTrue(self.faucet.poisoned)
        self.assertEqual(self.node.calls, [])
        with self.assertRaises(g.GatewayError):
            self.faucet.request(address(2), "127.0.0.1")

    def test_corrupt_state_is_never_reset(self):
        original = self.faucet.state
        self.faucet.state = copy.deepcopy(original)
        self.faucet.state["grants"][address(1)] = grant()
        self.faucet.state["grants"][address(1)]["tx_bytes_base64"] = "AA=="
        with self.assertRaises(g.GatewayError):
            self.faucet.validate()
        self.faucet.state = original
        with patch.object(g, "regular_bytes", return_value=b"bad json"), self.assertRaises((ValueError, BlockingIOError)):
            g.Faucet(self.node)

    def test_failed_committed_execution_is_not_success(self):
        value = grant()
        self.faucet.state["grants"][address(1)] = value
        self.node.observations[value["txhash"]] = {"result": {"hash": value["txhash"], "height": "13", "tx_result": {"code": 5}}}
        self.assertTrue(self.faucet.observe(value))
        self.assertFalse(self.faucet.public(address(1), value)["committed"])
        self.assertEqual(value["status"], "failed")


class TransportTest(unittest.TestCase):
    def setUp(self):
        self.node = FakeNode(Path("/unused"))
        self.server = g.Server(("127.0.0.1", 0), self.node, None)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def request(self, method, path, body=None, headers=None):
        connection = http.client.HTTPConnection(*self.server.server_address, timeout=5)
        connection.request(method, path, body=body, headers=headers or {})
        response = connection.getresponse()
        result = response.status, response.read()
        connection.close()
        return result

    def test_signed_sync_broadcast_and_named_queries_allowed(self):
        for method in ("status", "abci_query", "tx", "broadcast_tx_sync"):
            params = {"hash": base64.b64encode(b"x" * 32).decode()} if method == "tx" else {}
            status, _ = self.request("POST", "/", g.canonical({"jsonrpc": "2.0", "id": 1, "method": method, "params": params}))
            self.assertEqual(status, 200)

    def test_admin_batch_subscription_and_other_broadcasts_refused(self):
        for method in ("unsafe_flush_mempool", "dial_peers", "subscribe", "broadcast_tx_async", "broadcast_tx_commit", "tx_search"):
            status, _ = self.request("POST", "/", g.canonical({"jsonrpc": "2.0", "id": 1, "method": method}))
            self.assertEqual(status, 403)
        status, _ = self.request("POST", "/", b'[{"jsonrpc":"2.0","id":1,"method":"status"}]')
        self.assertEqual(status, 403)
        self.assertEqual(self.node.calls, [])

    def test_get_cannot_broadcast_and_files_are_not_general_routes(self):
        for path in ("/broadcast_tx_sync?tx=0x01", "/config/priv_validator_key.json", "/../dev-faucet/state.json", "/claims/nope"):
            self.assertEqual(self.request("GET", path)[0], 404)
        self.assertEqual(self.node.calls, [])

    def test_duplicate_json_keys_and_bodies_on_read_refused(self):
        self.assertEqual(self.request("POST", "/", b'{"method":"status","method":"abci_query"}')[0], 400)
        self.assertEqual(self.request("GET", "/status", "x")[0], 400)
        self.assertEqual(self.node.calls, [])

    def test_claim_route_and_genesis_are_readable(self):
        self.assertEqual(self.request("GET", "/genesis.json"), (200, self.node.genesis))
        status, data = self.request("GET", "/claims/" + "a" * 32)
        self.assertEqual(status, 200)
        self.assertEqual(g.parse(data)["chain_id"], g.CHAIN)

    def test_oversized_request_refused_before_parsing(self):
        status, _ = self.request("POST", "/", b"a" * (g.MAX_REQUEST + 1))
        self.assertEqual(status, 413)


class NodeReadinessTest(unittest.TestCase):
    def setUp(self):
        self.status = {"node_info": {"network": g.CHAIN, "id": "a" * 40},
            "sync_info": {"catching_up": False, "latest_block_height": "10",
                "latest_block_time": datetime.datetime.now(datetime.timezone.utc).isoformat()}}
        self.node = SimpleNamespace(manifest={"node_id": "a" * 40}, call=lambda _: self.status)

    def test_current_exact_node_is_ready(self):
        self.assertEqual(g.Node.ready(self.node), 10)

    def test_halted_catching_up_wrong_chain_and_wrong_identity_are_not_ready(self):
        original = copy.deepcopy(self.status)
        changes = [("node_info", "network", "zerone-1"), ("node_info", "id", "b" * 40),
            ("sync_info", "catching_up", True), ("sync_info", "latest_block_height", "0"),
            ("sync_info", "latest_block_time", "2000-01-01T00:00:00Z"),
            ("sync_info", "latest_block_time", "2999-01-01T00:00:00Z"),
            ("sync_info", "latest_block_time", "no block time")]
        for section, key, value in changes:
            self.status = copy.deepcopy(original)
            self.status[section][key] = value
            with self.subTest(key=key, value=value), self.assertRaises(g.GatewayError):
                g.Node.ready(self.node)


if __name__ == "__main__":
    unittest.main()
