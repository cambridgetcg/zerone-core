#!/usr/bin/env python3
"""Local sandbox safety tests; opt-in native binary lifecycle (no shared network).

python3 -m unittest discover -s scripts -p 'test_local_node.py' -v
ZERONE_LOCAL_NODE_TEST_BINARY="$PWD/build/zeroned" python3 -m unittest discover -s scripts -p 'test_local_node.py' -v

The real lifecycle uses the guide's default loopback ports 47656/47657, refuses
occupied ports, creates fresh temporary test keys, and stops only its own child.
"""
import argparse
import http.server
import importlib.util
import json
import os
from pathlib import Path
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

SCRIPT = Path(__file__).with_name("local-node.py").resolve()
SPEC = importlib.util.spec_from_file_location("local_node", SCRIPT)
local = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(local)


class SafetyTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="zerone-local-safety-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()

    def init_args(self, home, **values):
        args = dict(home=str(home), binary="/not/a/binary", chain_id="zerone-local-1",
                    rpc_port=47657, p2p_port=47656)
        args.update(values)
        return argparse.Namespace(**args)

    def fixture(self):
        home = self.root / "sandbox"
        home.mkdir(mode=0o700)
        for name in ("bin", "config", "data", "keyring-test"):
            (home / name).mkdir(mode=0o700)
        for name in ("bin/zeroned", *local.CONTROL_FILES, *local.IDENTITY_FILES,
                     "data/priv_validator_state.json"):
            (home / name).write_text("fixture " + name)
        manifest = dict(schema=local.SCHEMA, local_only=True, owner_uid=os.getuid(),
                        chain_id="zerone-local-test", rpc_port=47657, p2p_port=47656,
                        binary_sha256=local.sha256(home / "bin/zeroned"),
                        control_sha256={name: local.sha256(home / name) for name in local.CONTROL_FILES},
                        identity_sha256={name: local.sha256(home / name) for name in local.IDENTITY_FILES})
        (home / local.MARKER).write_text(json.dumps(manifest))
        return home, manifest

    def test_existing_home_never_runs_binary_or_changes_files(self):
        for populated in (False, True):
            home = self.root / ("populated" if populated else "empty")
            home.mkdir()
            if populated:
                (home / "keep").write_bytes(b"existing node state")
            before = {path.name: path.read_bytes() for path in home.iterdir()}
            with mock.patch.object(local.subprocess, "run") as run:
                with self.assertRaisesRegex(local.LocalNodeError, "new home"):
                    local.initialize(self.init_args(home))
                run.assert_not_called()
            self.assertEqual(before, {path.name: path.read_bytes() for path in home.iterdir()})

    def test_knowledge_profile_requires_current_records_and_changes_only_windows(self):
        knowledge = {"record_integrity_enabled": True, "review_neutrality_enabled": True,
                     "claim_records_enabled": True, "params": {"min_verifiers": 3,
                     "confidence_threshold": 770000, "commit_phase_blocks": "200",
                     "reveal_phase_blocks": "200", "aggregation_phase_blocks": "50"}}
        genesis = {"app_state": {"knowledge": knowledge}}
        local.configure_knowledge_profile(genesis)
        self.assertEqual(knowledge["params"], {"min_verifiers": 3, "confidence_threshold": 770000,
            "commit_phase_blocks": "300", "reveal_phase_blocks": "300", "aggregation_phase_blocks": "5"})
        local.configure_knowledge_profile(genesis, fast=True)
        self.assertEqual(knowledge["params"]["commit_phase_blocks"], "60")
        self.assertEqual(knowledge["params"]["reveal_phase_blocks"], "60")
        for marker in ("record_integrity_enabled", "review_neutrality_enabled", "claim_records_enabled"):
            knowledge[marker] = False
            with self.assertRaisesRegex(local.LocalNodeError, marker):
                local.configure_knowledge_profile(genesis)
            knowledge[marker] = True

    def test_symlink_and_default_homes_refused(self):
        target = self.root / "target"
        target.mkdir()
        (self.root / "link").symlink_to(target, target_is_directory=True)
        for home in (self.root / "link", self.root / ".zeroned", self.root / "zerone-1"):
            with self.assertRaises(local.LocalNodeError):
                local.home_path(str(home))
        self.assertEqual(list(target.iterdir()), [])

    def test_shared_network_chain_ids_refused_before_binary(self):
        for chain in ("zerone-1", "zerone-2", "zerone-local-", "zerone-local-A", "zerone-local-1/other"):
            with mock.patch.object(local.subprocess, "run") as run:
                with self.assertRaisesRegex(local.LocalNodeError, "Local chain IDs"):
                    local.initialize(self.init_args(self.root / "new", chain_id=chain))
                run.assert_not_called()
            self.assertFalse((self.root / "new").exists())

    def test_occupied_port_remains_open(self):
        with socket.socket() as listener:
            listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            listener.bind(("127.0.0.1", 0))
            listener.listen(1)
            number = listener.getsockname()[1]
            with self.assertRaisesRegex(local.LocalNodeError, "occupied"):
                local.check_ports(number, number - 1)
            with socket.create_connection(("127.0.0.1", number), timeout=1):
                connection, _ = listener.accept()
                connection.close()

    def test_toml_changes_are_exact_and_fail_before_write(self):
        path = self.root / "config.toml"
        original = '[rpc]\nladdr = "tcp://0.0.0.0:26657"\nunsafe = true\n'
        path.write_text(original)
        with self.assertRaisesRegex(local.LocalNodeError, "exactly one"):
            local.configure_toml(path, {"rpc": {"missing": False}})
        self.assertEqual(path.read_text(), original)
        local.configure_toml(path, {"rpc": {"laddr": "tcp://127.0.0.1:47657", "unsafe": False}})
        self.assertEqual(local.tomllib.loads(path.read_text())["rpc"],
                         dict(laddr="tcp://127.0.0.1:47657", unsafe=False))
        duplicate = original + '[rpc]\nunsafe = false\n'
        path.write_text(duplicate)
        with self.assertRaises(ValueError):
            local.configure_toml(path, {"rpc": {"unsafe": False}})
        self.assertEqual(path.read_text(), duplicate)

    def test_binary_genesis_listener_and_identity_replacements_refused_before_launch(self):
        home, manifest = self.fixture()
        self.assertEqual(local.load_manifest(home), manifest)
        for name in ("bin/zeroned", *local.CONTROL_FILES, *local.IDENTITY_FILES):
            path = home / name
            original = path.read_bytes()
            path.write_bytes(original + b"changed")
            with mock.patch.object(local.subprocess, "Popen") as popen:
                with self.assertRaises(local.LocalNodeError):
                    local.start(argparse.Namespace(home=str(home)))
                popen.assert_not_called()
            path.write_bytes(original)
        # The signing height is mutable persisted state, not a static key.
        (home / "data/priv_validator_state.json").write_text('{"height":"42"}')
        self.assertEqual(local.load_manifest(home), manifest)

    def test_symlinked_control_directory_refused(self):
        home, _ = self.fixture()
        (home / "config").rename(home / "original-config")
        (home / "config").symlink_to(home / "original-config", target_is_directory=True)
        with self.assertRaisesRegex(local.LocalNodeError, "directories"):
            local.load_manifest(home)

    def test_dangling_log_symlink_never_creates_target_or_launches(self):
        home, _ = self.fixture()
        target = self.root / "outside-log"
        (home / "node.log").symlink_to(target)
        with mock.patch.object(local, "check_ports"), mock.patch.object(local.subprocess, "Popen") as popen:
            with self.assertRaisesRegex(local.LocalNodeError, "non-symlink"):
                local.start(argparse.Namespace(home=str(home)))
            popen.assert_not_called()
        self.assertFalse(target.exists())

    def test_second_start_refuses_held_lock_without_launch(self):
        home, _ = self.fixture()
        with local.locked(home), mock.patch.object(local.subprocess, "Popen") as popen:
            with self.assertRaisesRegex(local.LocalNodeError, "already"):
                local.start(argparse.Namespace(home=str(home)))
            popen.assert_not_called()

    def test_operator_environment_cannot_override_local_config(self):
        with mock.patch.dict(os.environ, {"ZERONED_P2P_SEEDS": "live-peer", "ZERONED_PRIV_VALIDATOR_LADDR": "remote", "PATH": "/test"}):
            result = local.node_environment()
        self.assertNotIn("ZERONED_P2P_SEEDS", result)
        self.assertNotIn("ZERONED_PRIV_VALIDATOR_LADDR", result)
        self.assertEqual(result["PATH"], "/test")

    def test_key_command_failure_never_exposes_captured_seed(self):
        result = subprocess.CompletedProcess([], 1, "private seed stdout", "private seed stderr")
        with mock.patch.object(local.subprocess, "run", return_value=result):
            with self.assertRaises(local.LocalNodeError) as caught:
                local.cli(Path("zeroned"), self.root, "keys", "add", "user", secret=True)
        self.assertNotIn("private seed", str(caught.exception))

    def test_readiness_requires_advancement_and_identity(self):
        manifest = dict(chain_id="zerone-local-test", node_id="a" * 40)
        value = dict(sync_info=dict(latest_block_height="5", catching_up=False))
        with mock.patch.object(local, "rpc_status", return_value=value):
            with self.assertRaisesRegex(local.LocalNodeError, "advancing"):
                local.wait_ready(manifest, 0.01)
        with mock.patch.object(local, "rpc_status", side_effect=local.LocalNodeError("different node")):
            with self.assertRaisesRegex(local.LocalNodeError, "different node"):
                local.wait_ready(manifest, 1)

    def test_rpc_redirect_and_wrong_chain_are_not_followed_or_accepted(self):
        requests = []
        class Handler(http.server.BaseHTTPRequestHandler):
            redirect = True
            def log_message(self, *_args):
                pass
            def do_GET(self):
                requests.append(self.path)
                if self.redirect:
                    self.send_response(302)
                    self.send_header("Location", "/external-target")
                    self.end_headers()
                else:
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(json.dumps({"result": {"node_info": {"network": "zerone-1", "id": "a" * 40}}}).encode())
        with http.server.HTTPServer(("127.0.0.1", 0), Handler) as server:
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            try:
                manifest = dict(rpc_port=server.server_port, chain_id="zerone-local-test", node_id="a" * 40)
                with self.assertRaisesRegex(local.LocalNodeError, "redirect"):
                    local.rpc_status(manifest)
                Handler.redirect = False
                with self.assertRaisesRegex(local.LocalNodeError, "different node or chain"):
                    local.rpc_status(manifest)
            finally:
                server.shutdown()
                thread.join(timeout=2)
        self.assertEqual(requests, ["/status", "/status"])


@unittest.skipUnless(os.environ.get("ZERONE_LOCAL_NODE_TEST_BINARY"), "set ZERONE_LOCAL_NODE_TEST_BINARY for the real native lifecycle")
class NativeLifecycleTests(unittest.TestCase):
    def command(self, *args, timeout=120):
        result = subprocess.run([sys.executable, str(SCRIPT), *args], capture_output=True, text=True, timeout=timeout)
        self.assertEqual(result.returncode, 0, result.stderr)
        return json.loads(result.stdout)

    def start_node(self, home):
        process = subprocess.Popen([sys.executable, str(SCRIPT), "start", "--home", str(home)],
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        try:
            readable, _, _ = select.select([process.stdout], [], [], 55)
            self.assertTrue(readable, "helper readiness timed out")
            line = process.stdout.readline()
            if not line:
                self.fail("helper exited: " + process.stderr.read())
            ready = json.loads(line)
            self.assertTrue(ready["ready"])
            self.assertTrue(ready["advancing"])
            return process, ready
        except BaseException:
            self.stop_node(process)
            raise

    def stop_node(self, process):
        if process.poll() is None:
            process.send_signal(signal.SIGINT)
        output, error = process.communicate(timeout=30)
        self.assertEqual(process.returncode, 0, error)
        self.assertTrue(json.loads(output.splitlines()[-1])["state_preserved"])

    def test_native_init_transfer_stop_restart_at_guide_ports(self):
        local.check_ports(47657, 47656)  # Never displace another service.
        with tempfile.TemporaryDirectory(prefix="zerone-local-lifecycle-") as directory:
            home = Path(directory).resolve() / "node"
            info = self.command("init", "--home", str(home), "--binary", os.environ["ZERONE_LOCAL_NODE_TEST_BINARY"])
            self.assertTrue(info["initialized"])
            self.assertEqual(info["chain_id"], "zerone-local-1")
            self.assertEqual(info["rpc"], "http://127.0.0.1:47657")
            genesis = json.loads((home / "config/genesis.json").read_text())
            self.assertTrue(genesis["app_state"]["zerone_staking"]["accounting_safety_enabled"])
            self.assertTrue(genesis["app_state"]["zerone_gov"]["accounting_safety_enabled"])
            before = local.load_manifest(home)
            process, ready = self.start_node(home)
            try:
                duplicate = subprocess.run([sys.executable, str(SCRIPT), "start", "--home", str(home)], capture_output=True, text=True, timeout=20)
                self.assertNotEqual(duplicate.returncode, 0)
                self.assertIn("already", duplicate.stderr)
                self.assertIsNone(process.poll())
                status = self.command("status", "--home", str(home))
                self.assertGreater(status["height"], ready["height"])
                binary = home / "bin/zeroned"
                sent = json.loads(local.cli(binary, home, "tx", "bank", "send", "user", info["validator_address"],
                                            "100uzrn", "--chain-id", info["chain_id"], "--keyring-backend", "test",
                                            "--node", info["rpc"], "--fees", "250000uzrn", "--gas", "250000", "--yes", "--output", "json"))
                self.assertEqual(sent["code"], 0)
                deadline = time.monotonic() + 15
                receipt = None
                while time.monotonic() < deadline:
                    try:
                        receipt = json.loads(local.cli(binary, home, "query", "tx", sent["txhash"], "--node", info["rpc"], "--output", "json"))
                        break
                    except local.LocalNodeError:
                        time.sleep(0.2)
                self.assertIsNotNone(receipt, "submitted transfer never committed")
                self.assertEqual(receipt["code"], 0)
                self.assertGreater(int(receipt["height"]), 0)
                self.assertLessEqual(int(receipt["gas_used"]), 250000)
                balance = json.loads(local.cli(binary, home, "query", "bank", "balances", info["user_address"], "--node", info["rpc"], "--output", "json"))
                self.assertEqual(balance["balances"], [{"denom": "uzrn", "amount": "999749900"}])
                height = int(receipt["height"])
            finally:
                self.stop_node(process)
            self.assertEqual(before, local.load_manifest(home))
            self.assertGreaterEqual(int(json.loads((home / "data/priv_validator_state.json").read_text())["height"]), height)
            local.check_ports(47657, 47656)
            process, restarted = self.start_node(home)
            try:
                self.assertGreater(restarted["height"], height)
                self.assertEqual(restarted["node_id"], ready["node_id"])
                after_balance = json.loads(local.cli(binary, home, "query", "bank", "balances", info["user_address"], "--node", info["rpc"], "--output", "json"))
                self.assertEqual(after_balance["balances"], balance["balances"])
            finally:
                self.stop_node(process)
            self.assertEqual(before, local.load_manifest(home))
            local.check_ports(47657, 47656)


if __name__ == "__main__":
    unittest.main()
