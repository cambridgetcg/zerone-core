#!/usr/bin/env python3
"""Safety fixtures; optional real two-node lifecycle via ZERONE_SHARED_TEST_BINARY."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import signal
import socket
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock
import urllib.request

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("dev_runtime", HERE / "runtime.py")
runtime = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runtime)


class RuntimeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.home = self.root / "node"

    def tearDown(self):
        self.temp.cleanup()

    def fixture(self):
        self.home.mkdir(mode=0o700)
        for name in ("config", "data", "dev-faucet", "keyring-test"):
            (self.home / name).mkdir(mode=0o700)
        for name in (*runtime.CONTROL, *runtime.IDENTITY):
            runtime.write_new(self.home / name, b"{}\n")
        runtime.write_new(self.home / "data/priv_validator_state.json", b'{"height":"7","round":0,"step":3}')
        runtime.write_new(self.home / "dev-faucet/state.json", b'{"grants":{}}')
        binary = self.root / "binary"
        runtime.write_new(binary, b"test binary")
        manifest = {"schema": runtime.SCHEMA, "chain_id": runtime.CHAIN, "role": "validator", "owner_uid": os.getuid(),
                    "binary_sha256": runtime.digest(binary), "genesis_sha256": runtime.digest(self.home / "config/genesis.json"),
                    "control_sha256": {name: runtime.digest(self.home / name) for name in runtime.CONTROL},
                    "identity_sha256": {name: runtime.digest(self.home / name) for name in runtime.IDENTITY}}
        runtime.write_new(self.home / runtime.MARKER, runtime.json_bytes(manifest))
        return binary, manifest

    def test_existing_home_refused_even_empty(self):
        self.home.mkdir(mode=0o700)
        with self.assertRaisesRegex(runtime.RuntimeError, "new home"):
            runtime.initialize(argparse.Namespace(), self.home)
        self.assertEqual(list(self.home.iterdir()), [])

    def test_no_legacy_chain_or_missing_native_flags(self):
        value = {"chain_id": runtime.CHAIN, "app_state": {"zerone_staking": {"accounting_safety_enabled": True},
                 "zerone_gov": {"accounting_safety_enabled": True}, "knowledge": {field: True for field in
                 ("record_integrity_enabled", "review_neutrality_enabled", "claim_records_enabled", "fund_settlement_enabled")}}}
        runtime.check_genesis(value)
        value["chain_id"] = "zerone-1"
        with self.assertRaises(runtime.RuntimeError):
            runtime.check_genesis(value)
        value["chain_id"] = runtime.CHAIN
        value["app_state"]["knowledge"]["claim_records_enabled"] = False
        with self.assertRaises(runtime.RuntimeError):
            runtime.check_genesis(value)

    def test_missing_state_and_identity_drift_refused(self):
        binary, _ = self.fixture()
        runtime.load(self.home, binary)
        target = self.home / runtime.IDENTITY[0]
        target.write_bytes(target.read_bytes() + b"\n")
        with self.assertRaisesRegex(runtime.RuntimeError, "identity changed"):
            runtime.load(self.home, binary)
        target.write_bytes(b"{}\n")
        (self.home / "data/priv_validator_state.json").unlink()
        with self.assertRaises(FileNotFoundError):
            runtime.load(self.home, binary)

    def test_control_and_binary_drift_refused(self):
        binary, _ = self.fixture()
        binary.write_bytes(b"other binary")
        with self.assertRaisesRegex(runtime.RuntimeError, "binary changed"):
            runtime.load(self.home, binary)
        binary.write_bytes(b"test binary")
        (self.home / "config/app.toml").write_bytes(b"override")
        with self.assertRaisesRegex(runtime.RuntimeError, "configuration"):
            runtime.load(self.home, binary)

    def test_mutable_faucet_progress_retained_but_missing_state_refused(self):
        binary, _ = self.fixture()
        path = self.home / "dev-faucet/state.json"
        path.write_bytes(b'{"grants":{"retained":"receipt"}}')
        runtime.load(self.home, binary)
        self.assertIn(b"retained", path.read_bytes())
        path.unlink()
        with self.assertRaises(FileNotFoundError):
            runtime.load(self.home, binary)

    def test_fifo_symlink_hardlink_and_public_file_refused(self):
        path = self.root / "fifo"
        os.mkfifo(path, 0o600)
        with self.assertRaises(runtime.RuntimeError):
            runtime.read(path)
        target = self.root / "target"
        runtime.write_new(target, b"test")
        link = self.root / "link"
        link.symlink_to(target)
        with self.assertRaises(OSError):
            runtime.read(link)
        link.unlink()
        os.link(target, link)
        with self.assertRaises(runtime.RuntimeError):
            runtime.read(target)
        link.unlink()
        target.chmod(0o644)
        with self.assertRaises(runtime.RuntimeError):
            runtime.read(target, private=True)

    def test_same_home_lock_excludes_second_owner(self):
        with runtime.lock(self.home):
            with self.assertRaisesRegex(runtime.RuntimeError, "Another runtime"):
                with runtime.lock(self.home):
                    self.fail("second owner entered")
        with runtime.lock(self.home):
            pass

    def test_env_cannot_override_recorded_config(self):
        with mock.patch.dict(os.environ, {"ZERONED_HOME": "/elsewhere", "ZERONED_P2P_LADDR": "other"}):
            self.assertNotIn("ZERONED_HOME", runtime.environment())
            self.assertNotIn("ZERONED_P2P_LADDR", runtime.environment())

    def test_full_node_identity_power_and_common_block_checks(self):
        manifest = {"rpc_port": 1234, "node_id": "1" * 40, "consensus_address": "AB", "role": "full-node",
                    "genesis_sha256": "2" * 64, "local_test": True, "reference_rpc": "http://example.test",
                    "reference_peer": "3" * 40 + "@example.test:26656"}
        own = {"node_info": {"network": runtime.CHAIN, "id": "1" * 40}, "sync_info": {"latest_block_height": "7", "catching_up": False},
               "validator_info": {"address": "AB", "voting_power": "0"}}
        ref = {"node_info": {"network": runtime.CHAIN, "id": "3" * 40}, "sync_info": {"latest_block_height": "8"}}
        block = {"block_id": {"hash": "A" * 64}, "block": {"header": {"height": "7", "chain_id": runtime.CHAIN, "app_hash": "B" * 64}}}
        with mock.patch.object(runtime, "rpc", side_effect=[own, ref, block, block]):
            self.assertTrue(runtime.status(manifest)["reference_match"])
        other = json.loads(json.dumps(block))
        other["block_id"]["hash"] = "C" * 64
        with mock.patch.object(runtime, "rpc", side_effect=[own, ref, block, other]):
            with self.assertRaisesRegex(runtime.RuntimeError, "disagree"):
                runtime.status(manifest)
        own["validator_info"]["voting_power"] = "1"
        with mock.patch.object(runtime, "rpc", return_value=own):
            with self.assertRaisesRegex(runtime.RuntimeError, "zero-power"):
                runtime.status(manifest)

    def test_failed_child_stops_other_and_releases_lock(self):
        binary, manifest = self.fixture()
        manifest.update(local_test=False, role="validator", gateway_port=8080, rpc_port=26657)
        node, gateway = mock.Mock(), mock.Mock()
        node.poll.side_effect = [None, None, None, None, None, None, None]
        gateway.poll.return_value = 9
        args = argparse.Namespace(binary=binary, gateway=HERE / "gateway.py")
        observed = {"height": 1, "ready": True}
        manifest.update(source_commit="a" * 40, advertised_peer="test", accounts={})
        with mock.patch.object(runtime, "load", return_value=manifest), mock.patch.object(runtime, "status", side_effect=[dict(observed, height=0), observed]), \
             mock.patch.object(runtime.subprocess, "Popen", side_effect=[node, gateway]) as popen, mock.patch.object(runtime.time, "sleep"):
            with self.assertRaisesRegex(runtime.RuntimeError, "process exited"):
                runtime.run(args, self.home)
            node.send_signal.assert_called_once_with(signal.SIGTERM)
            node.wait.assert_called_once_with(timeout=30)
            self.assertNotIn("stdout", popen.call_args_list[0].kwargs)
        self.assertFalse((self.home / "node.log").exists())
        with runtime.lock(self.home):
            pass

    def startup_clock(self, role, samples, invalid=None):
        """Exercise run/status with simulated elapsed time and an owned fake child."""
        if not self.home.exists():
            self.fixture()
        binary = self.root / "binary"
        manifest = json.loads((self.home / runtime.MARKER).read_bytes())
        manifest.update(role=role, local_test=False, node_id="1" * 40,
                        consensus_address="AB", rpc_port=26657, source_commit="a" * 40,
                        advertised_peer="", accounts={})
        clock, calls, printed = [0], [], []
        node = mock.Mock()
        node.poll.return_value = None
        node.wait.return_value = 0

        def rpc(_origin, path):
            self.assertEqual(path, "/status")
            height, catching_up = samples[min(len(calls), len(samples) - 1)]
            calls.append(clock[0])
            value = {"node_info": {"network": runtime.CHAIN, "id": manifest["node_id"]},
                     "sync_info": {"latest_block_height": str(height), "catching_up": catching_up},
                     "validator_info": {"address": "AB", "voting_power": "0"}}
            if invalid:
                value[invalid[0]][invalid[1]] = invalid[2]
            return value

        def sleep(_seconds):
            clock[0] += 50

        def ready_print(value, **_kwargs):
            printed.append(json.loads(value))
            signal.getsignal(signal.SIGTERM)(signal.SIGTERM, None)

        args = argparse.Namespace(binary=binary, gateway=HERE / "gateway.py")
        try:
            with mock.patch.object(runtime, "load", return_value=manifest), \
                 mock.patch.object(runtime, "rpc", side_effect=rpc), \
                 mock.patch.object(runtime.time, "monotonic", side_effect=lambda: clock[0]), \
                 mock.patch.object(runtime.time, "sleep", side_effect=sleep), \
                 mock.patch.object(runtime.subprocess, "Popen", return_value=node), \
                 mock.patch.object(runtime, "print", side_effect=ready_print, create=True):
                runtime.run(args, self.home)
        finally:
            node.send_signal.assert_called_once_with(signal.SIGTERM)
            node.wait.assert_called_once_with(timeout=30)
            self.startup_observed = {"elapsed": clock[0], "calls": calls, "printed": printed}
            with runtime.lock(self.home):
                pass
        return self.startup_observed

    def test_full_node_progress_can_reach_readiness_after_two_minutes(self):
        observed = self.startup_clock("full-node", [(0, True), (10, True), (20, True), (30, False)])
        self.assertEqual(observed["elapsed"], 150)
        self.assertEqual(observed["calls"], [0, 50, 100, 150])
        self.assertEqual(observed["printed"][0]["height"], 30)
        self.assertTrue(observed["printed"][0]["ready"])

    def test_full_node_stall_and_height_oscillation_do_not_extend_allowance(self):
        with self.assertRaisesRegex(runtime.RuntimeError, "no new block-height progress for 120 seconds.*best height 8"):
            self.startup_clock("full-node", [(5, True), (8, True), (7, True), (8, True)])
        self.assertEqual(self.startup_observed["elapsed"], 200)
        self.assertEqual(self.startup_observed["calls"], [0, 50, 100, 150])
        self.assertEqual(self.startup_observed["printed"], [])

    def test_validator_readiness_deadline_stays_fixed_despite_progress(self):
        with self.assertRaisesRegex(runtime.RuntimeError, "Node did not demonstrate advancing blocks before timeout"):
            self.startup_clock("validator", [(5, True), (8, True), (10, True), (11, False)])
        self.assertEqual(self.startup_observed["elapsed"], 150)
        self.assertEqual(self.startup_observed["calls"], [0, 50, 100])
        self.assertEqual(self.startup_observed["printed"], [])

    def test_full_node_progress_does_not_bypass_identity_or_zero_power(self):
        for invalid in (("node_info", "network", "another-chain"),
                        ("node_info", "id", "2" * 40),
                        ("validator_info", "address", "CD"),
                        ("validator_info", "voting_power", "1")):
            with self.subTest(invalid=invalid):
                with self.assertRaises(runtime.RuntimeError):
                    self.startup_clock("full-node", [(100, True)], invalid)
                self.assertEqual(self.startup_observed["elapsed"], 0)
                self.assertEqual(self.startup_observed["calls"], [0])
                self.assertEqual(self.startup_observed["printed"], [])


@unittest.skipUnless(os.environ.get("ZERONE_SHARED_TEST_BINARY"), "set ZERONE_SHARED_TEST_BINARY for the real lifecycle")
class NativeRuntimeTests(unittest.TestCase):
    def test_fresh_validator_fullnode_restart_and_no_reset(self):
        binary = Path(os.environ["ZERONE_SHARED_TEST_BINARY"]).resolve()
        metadata = subprocess.run([str(binary), "version", "--long"], check=True, capture_output=True, text=True).stdout
        commit = re.search(r"(?m)^commit: ([0-9a-f]{40})$", metadata).group(1)
        children, handles = [], []
        with tempfile.TemporaryDirectory(prefix="zerone-dev-runtime-test-") as directory:
            root = Path(directory)
            sockets = []
            for _ in range(6):
                sock = socket.socket(); sock.bind(("127.0.0.1", 0)); sockets.append(sock)
            ports = [sock.getsockname()[1] for sock in sockets]
            for sock in sockets: sock.close()
            def command(action, home, *extra):
                return [sys.executable, "-I", "-B", str(HERE / "runtime.py"), action, "--home", str(home), "--binary", str(binary), *extra]
            def execute(action, home, *extra, success=True):
                result = subprocess.run(command(action, home, *extra), capture_output=True, text=True, timeout=60)
                self.assertEqual(result.returncode == 0, success, result.stderr)
                return json.loads(result.stdout) if success else result
            def start(home):
                log = (root / (home.name + ".log")).open("wb"); handles.append(log)
                proc = subprocess.Popen(command("run", home), stdout=log, stderr=subprocess.STDOUT)
                children.append(proc)
                return proc
            def stop(proc):
                proc.send_signal(signal.SIGTERM)
                self.assertEqual(proc.wait(timeout=40), 0)
            def wait(home, minimum=1):
                deadline = time.monotonic() + 90
                while time.monotonic() < deadline:
                    if any(proc.poll() not in (None, 0) for proc in children):
                        self.fail("An owned runtime exited: " + (root / (home.name + ".log")).read_text())
                    try:
                        value = runtime.status(runtime.load(home, binary))
                        if value["ready"] and value["height"] > minimum:
                            return value
                    except (OSError, ValueError, KeyError, runtime.RuntimeError):
                        pass
                    time.sleep(0.5)
                self.fail("Node did not become ready")
            validator, follower = root / "validator", root / "follower"
            try:
                created = execute("init", validator, "--source-commit", commit, "--local-test", "--review-window-blocks", "30",
                                  "--rpc-port", str(ports[0]), "--p2p-port", str(ports[1]), "--gateway-port", str(ports[2]))
                genesis = runtime.parse(runtime.read(validator / "config/genesis.json"))["app_state"]
                self.assertEqual(genesis["staking"]["params"]["max_validators"], 1)
                gentx = genesis["genutil"]["gen_txs"][0]["body"]["messages"][0]
                self.assertEqual(int(gentx["value"]["amount"]), runtime.VALIDATOR_BOND)
                amounts = {row["address"]: int(row["coins"][0]["amount"]) for row in genesis["bank"]["balances"]}
                self.assertEqual(amounts[created["accounts"]["validator"]], runtime.VALIDATOR_ALLOCATION)
                self.assertEqual(amounts[created["accounts"]["faucet"]], runtime.FAUCET_ALLOCATION)
                self.assertGreater(runtime.VALIDATOR_BOND, 100_000_000_000)
                execute("init", validator, success=False)
                node = start(validator)
                first = wait(validator)
                deadline = time.monotonic() + 30
                while True:
                    try:
                        with urllib.request.urlopen(f"http://127.0.0.1:{ports[2]}/healthz", timeout=2) as response:
                            self.assertEqual(response.status, 200)
                        break
                    except OSError:
                        if time.monotonic() > deadline: raise
                        time.sleep(0.2)
                execute("run", validator, success=False)  # same-home lock, no second signer
                joined = execute("join", follower, "--source-commit", commit, "--local-test", "--rpc-port", str(ports[3]),
                    "--p2p-port", str(ports[4]), "--gateway-port", str(ports[5]), "--genesis", str(validator / "config/genesis.json"),
                    "--genesis-sha256", created["genesis_sha256"], "--peer", created["advertised_peer"],
                    "--reference-rpc", f"http://127.0.0.1:{ports[2]}")
                self.assertNotEqual(joined["node_id"], created["node_id"])
                replica = start(follower)
                observed = wait(follower)
                self.assertTrue(observed["reference_match"])
                self.assertEqual(observed["voting_power"], 0)
                stop(replica)
                stop(node)
                state = validator / "data/priv_validator_state.json"
                before = int(runtime.parse(runtime.read(state))["height"])
                control = runtime.read(validator / runtime.MARKER)
                node = start(validator)
                wait(validator, before + 1)
                stop(node)
                self.assertGreater(int(runtime.parse(runtime.read(state))["height"]), before)
                self.assertEqual(runtime.read(validator / runtime.MARKER), control)
                self.assertGreater(first["voting_power"], 0)
                state.rename(state.with_suffix(".preserved"))
                execute("run", validator, success=False)
                self.assertFalse(state.exists())
            finally:
                for proc in children:
                    if proc.poll() is None:
                        proc.send_signal(signal.SIGTERM)
                        try: proc.wait(timeout=40)
                        except subprocess.TimeoutExpired:
                            proc.kill(); proc.wait(timeout=5)
                for stream in handles: stream.close()


if __name__ == "__main__":
    unittest.main()
