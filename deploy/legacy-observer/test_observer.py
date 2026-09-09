#!/usr/bin/env python3
"""Offline observer lifecycle tests; Docker calls are controlled test doubles.

Only the redirect test starts a loopback HTTP server. No node, account, live
endpoint, Docker daemon, private key or release authority is accessed.
"""
import argparse
import base64
import contextlib
import copy
import datetime as dt
import hashlib
from http.server import BaseHTTPRequestHandler, HTTPServer
import importlib.util
import io
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("observer_helper", HERE / "observer.py")
o = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(o)


# Public RPC bodies from the actual signed v1 startup, September 9, 2026.
# ABCI SHA256 b227fbc96011937c9a3fc115e3a96a2faac300c44949b9c737326a5266cf1b68
# Status SHA256 1fa512f6cc46b16aee9eaced0823b9ed4e4610dac3c36386742b74b5f0c57ff6
# Only the status fields read by the helper are retained; identities below
# are public, disposable observer identities, not imported signing material.
STARTUP_APP = {"response": {"data": "zeroned", "version": "dev",
                           "last_block_app_hash": "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU="}}
STARTUP_STATUS = {
    "node_info": {"network": "zerone-1", "id": "2c0ef29251168468696f178c062f44d6d2371a98"},
    "validator_info": {"address": "09E45DB11B50BB22BDC0B9DFE9272C7589BBE7D2", "voting_power": "0"},
    "sync_info": {"latest_block_height": "0", "latest_block_time": "1970-01-01T00:00:00Z",
                  "latest_block_hash": "", "latest_app_hash": "", "catching_up": True},
}


class ObserverTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="observer-helper-test-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.home = self.root / "home"
        self.bundle = self.root / "bundle"
        self.bundle.mkdir(mode=0o755)
        for name in ("zeroned", "verify-checkpoint", "verify-release.py", "genesis.json"):
            (self.bundle / name).write_bytes(b"test fixture; never executed")
            (self.bundle / name).chmod(0o755 if name in {"zeroned", "verify-checkpoint"} else 0o644)
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            self.port = listener.getsockname()[1]
        self.args = argparse.Namespace(home=str(self.home), port=self.port,
                                       sync_timeout=60, gpgv="/test/gpgv")

    def make_home(self, synced=None):
        self.home.mkdir(mode=0o700)
        (self.home / "config").mkdir()
        (self.home / "data").mkdir()
        for name in o.CONTROLS:
            (self.home / name).write_text("fixture " + name)
        (self.home / "config/priv_validator_key.json").write_text(json.dumps({"address": "A" * 40}))
        (self.home / "data/priv_validator_state.json").write_text('{"height":"0","round":0,"step":0}')
        (self.home / "data/application.db").mkdir()
        (self.home / "data/application.db/CURRENT").write_text("fixture database marker")
        uid, gid = o.runtime_identity()
        marker = {"schema": o.SCHEMA, "id": "a1" * 16, "home": str(self.home), "uid": uid, "gid": gid,
                  "operator_uid": os.getuid(), "port": self.port, "manifest_sha256": "b2" * 32,
                  "release_id": "zerone-1-observer-test", "chain_id": "zerone-1", "node_id": "c3" * 20,
                  "validator_address": "A" * 40, "control_sha256": {name: o.sha(self.home / name) for name in o.CONTROLS},
                  "synced": synced}
        o.write_json(self.home / o.MARKER, marker)
        return marker

    def test_existing_init_and_unrelated_start_do_not_write(self):
        self.home.mkdir()
        (self.home / "important.txt").write_text("preserve")
        before = {p.name: p.read_bytes() for p in self.home.iterdir()}
        with patch.object(o, "verify_bundle") as verify:
            with self.assertRaises(o.Refusal):
                o.init(self.args, self.bundle)
            verify.assert_not_called()
        with self.assertRaises((o.Refusal, OSError)):
            o.start(self.args, self.bundle)
        self.assertEqual(before, {p.name: p.read_bytes() for p in self.home.iterdir()})

    def test_control_identity_and_data_drift_refuse_before_docker(self):
        self.make_home()
        target = self.home / "config/node_key.json"
        original = target.read_bytes()
        target.write_bytes(original + b"drift")
        with patch.object(o, "docker") as docker:
            with self.assertRaisesRegex(o.Refusal, "control changed"):
                o.start(self.args, self.bundle)
            docker.assert_not_called()
        target.write_bytes(original)
        state = self.home / "data/priv_validator_state.json"
        state.write_text('{"height":"1","round":0,"step":0}')
        with self.assertRaisesRegex(o.Refusal, "signing state"):
            o.validate_home(self.home)
        state.write_text('{"height":"0","round":false,"step":0}')
        with self.assertRaisesRegex(o.Refusal, "signing state"):
            o.validate_home(self.home)
        (self.home / "data").rename(self.root / "old-data")
        (self.home / "data").symlink_to(self.root / "old-data", target_is_directory=True)
        with self.assertRaisesRegex(o.Refusal, "data directory"):
            o.validate_home(self.home)

    def test_docker_is_local_uid_gid_explicit_and_offline_by_default(self):
        with patch.object(o.platform, "system", return_value="Linux"), \
             patch.object(o.platform, "machine", return_value="x86_64"), \
             patch.object(o.shutil, "which", return_value="/test/docker"), \
             patch.dict(os.environ, {"DOCKER_HOST": "tcp://external.invalid:2375"}):
            argv = o.container(self.bundle)
        self.assertEqual(argv[:4], ["/test/docker", "--host", "unix:///var/run/docker.sock", "create"])
        self.assertEqual(argv[argv.index("--user") + 1], ":".join(map(str, o.runtime_identity())))
        self.assertEqual(argv[argv.index("--network") + 1], "none")
        self.assertIn("--read-only", argv)
        self.assertIn("ALL", argv)
        self.assertNotIn("--privileged", argv)
        self.assertNotIn("--publish", argv)
        self.assertNotIn("--rm", argv)
        self.bundle.chmod(0o700)
        with self.assertRaisesRegex(o.Refusal, "UID:GID"):
            o.bundle_access(self.bundle, os.getuid() + 1234, os.getgid() + 1234)

    def test_cleanup_checks_owner_and_preserves_failed_stopped_container(self):
        result = subprocess.CompletedProcess([], 0, "foreign false 0 false\n", "")
        with patch.object(o, "docker", return_value=["/test/docker"]), \
             patch.object(o.subprocess, "run", return_value=result), patch.object(o, "command") as command:
            with self.assertRaisesRegex(o.Refusal, "ownership"):
                o.stop_owned("test-container", "own")
            command.assert_not_called()
        results = [subprocess.CompletedProcess([], 0, "own true 0 false\n", ""),
                   subprocess.CompletedProcess([], 0, "own false 137 true\n", "")]
        with patch.object(o, "docker", return_value=["/test/docker"]), \
             patch.object(o.subprocess, "run", side_effect=results), patch.object(o, "command", return_value="") as command:
            final = o.stop_owned("test-container", "own")
        self.assertEqual(final, {"running": False, "exit_code": 137, "oom_killed": True,
                                 "container": "test-container", "removed": False})
        self.assertEqual(command.call_args_list[0].args[0][1], "stop")
        self.assertEqual(command.call_count, 1)

    def test_offline_timeout_cleans_only_its_named_container(self):
        cid = "d4" * 32
        with patch.object(o, "container", return_value=["/test/docker", "create"]), \
             patch.object(o, "docker", return_value=["/test/docker"]), \
             patch.object(o, "command", side_effect=[cid, subprocess.TimeoutExpired("fixture", 90)]), \
             patch.object(o, "stop_owned", return_value={"running": False, "exit_code": 137, "oom_killed": False,
                                                         "removed": False, "container": cid}) as stop:
            with self.assertRaisesRegex(o.Refusal, "stopped container retained: " + cid):
                o.offline_container(self.bundle, ["/bundle/verify-checkpoint"])
        self.assertEqual(stop.call_args.args[0], cid)
        self.assertRegex(stop.call_args.args[1], r"^[0-9a-f]{32}$")

    def test_offline_failed_init_never_prints_captured_key_material(self):
        cid = "d4" * 32
        with patch.object(o, "container", return_value=["/test/docker", "create"]), \
             patch.object(o, "docker", return_value=["/test/docker"]), \
             patch.object(o, "command", side_effect=[cid, subprocess.CalledProcessError(1, "fixture", output="SECRET")]), \
             patch.object(o, "stop_owned", return_value={"running": False, "exit_code": 1, "oom_killed": False,
                                                         "removed": False, "container": cid}), \
             contextlib.redirect_stdout(io.StringIO()) as stdout, contextlib.redirect_stderr(io.StringIO()) as stderr:
            with self.assertRaises(o.Refusal) as failure:
                o.offline_container(self.bundle, ["/bundle/zeroned", "init"], secret=True)
        self.assertNotIn("SECRET", str(failure.exception) + stdout.getvalue() + stderr.getvalue())

    def test_rpc_refuses_redirect_without_contacting_destination(self):
        visits = []
        class Redirect(BaseHTTPRequestHandler):
            def do_GET(self):
                visits.append(self.path)
                self.send_response(302)
                self.send_header("Location", "/must-not-follow")
                self.end_headers()
            def log_message(self, *args):
                pass
        server = HTTPServer(("127.0.0.1", 0), Redirect)
        worker = threading.Thread(target=server.serve_forever, daemon=True)
        worker.start()
        try:
            with self.assertRaisesRegex(o.Refusal, "redirect"):
                o.rpc(server.server_port, "status")
            self.assertEqual(visits, ["/status"])
            with self.assertRaisesRegex(o.Refusal, "Only status"):
                o.rpc(server.server_port, "broadcast_tx_sync")
        finally:
            server.shutdown(); server.server_close(); worker.join(timeout=2)

    def test_observation_requires_identity_zero_power_fresh_applied_state(self):
        marker = self.make_home()
        now = dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")
        status = {"node_info": {"network": "zerone-1", "id": marker["node_id"]},
                  "validator_info": {"address": marker["validator_address"], "voting_power": "0"},
                  "sync_info": {"latest_block_height": "10", "latest_block_time": now,
                                "latest_block_hash": "A" * 64, "latest_app_hash": "B" * 64, "catching_up": False}}
        app = {"response": {"last_block_height": "9", "last_block_app_hash": base64.b64encode(bytes.fromhex("C" * 64)).decode("ascii")}}
        def read(_port, method):
            return copy.deepcopy(status if method == "status" else app)
        with patch.object(o, "rpc", side_effect=read):
            self.assertFalse(o.observed(marker)["ready"], "staged block must not count as applied")
            app["response"]["last_block_height"] = "10"
            self.assertTrue(o.observed(marker)["ready"])
            status["node_info"]["id"] = "d4" * 20
            with self.assertRaisesRegex(o.Refusal, "P2P identity"):
                o.observed(marker)
            status["node_info"]["id"] = marker["node_id"]
            status["validator_info"]["voting_power"] = "1"
            with self.assertRaisesRegex(o.Refusal, "zero-power"):
                o.observed(marker)

    def startup_observation(self):
        marker = self.make_home()
        status, app = copy.deepcopy(STARTUP_STATUS), copy.deepcopy(STARTUP_APP)
        marker.update(node_id=status["node_info"]["id"], validator_address=status["validator_info"]["address"])
        return marker, status, app

    def test_actual_zero_height_startup_preserves_hash_without_readiness(self):
        marker, status, app = self.startup_observation()
        def read(_port, method):
            return copy.deepcopy(status if method == "status" else app)
        with patch.object(o, "rpc", side_effect=read):
            observed = o.observed(marker)
            self.assertEqual((observed["height"], observed["applied_height"]), (0, 0))
            self.assertFalse(observed["ready"])
            self.assertEqual(observed["abci_last_block_app_hash"], STARTUP_APP["response"]["last_block_app_hash"])
            for explicit in (False, True):
                for hash_present in (False, True):
                    with self.subTest(explicit_zero_height=explicit, empty_hash_present=hash_present):
                        app["response"] = {}
                        if explicit:
                            app["response"]["last_block_height"] = "0"
                        if hash_present:
                            app["response"]["last_block_app_hash"] = ""
                        result = o.observed(marker)
                        self.assertFalse(result["ready"])
                        self.assertEqual(result["abci_last_block_app_hash"], "")

    def test_restore_can_apply_snapshot_before_status_catches_up(self):
        marker, status, app = self.startup_observation()
        app["response"].update(last_block_height="1262000", last_block_app_hash="C" * 64)
        def read(_port, method):
            return copy.deepcopy(status if method == "status" else app)
        with patch.object(o, "rpc", side_effect=read):
            self.assertFalse(o.observed(marker)["ready"])
            status["sync_info"].update(latest_block_height="1262000", latest_block_hash="A" * 64,
                                       latest_app_hash="B" * 64,
                                       latest_block_time=dt.datetime.now(dt.timezone.utc).isoformat())
            self.assertFalse(o.observed(marker)["ready"])
            status["sync_info"]["catching_up"] = False
            result = o.observed(marker)
            self.assertTrue(result["ready"])
            # A header commits the previous application's root; same-height
            # header and post-commit ABCI hashes must not be equated here.
            self.assertNotEqual(result["header_app_hash"], result["abci_last_block_app_hash"])

    def test_zero_height_allowance_never_relaxes_identity_or_power(self):
        marker, original, app = self.startup_observation()
        for section, field, value in [("node_info", "network", "other-chain"),
                                      ("node_info", "id", "a" * 40),
                                      ("validator_info", "address", "B" * 40),
                                      ("validator_info", "voting_power", "1"),
                                      ("validator_info", "voting_power", 0)]:
            with self.subTest(section=section, field=field, value=value):
                status = copy.deepcopy(original)
                status[section][field] = value
                with patch.object(o, "rpc", return_value=status) as rpc:
                    with self.assertRaises(o.Refusal):
                        o.observed(marker)
                    self.assertEqual(rpc.call_count, 1, "identity refusal must precede ABCI query")

    def test_missing_or_zero_applied_height_requires_empty_zero_status(self):
        marker, original, app = self.startup_observation()
        for update in [{"latest_block_height": "1", "latest_block_hash": "A" * 64, "latest_app_hash": "B" * 64},
                       {"latest_block_hash": "A" * 64}, {"latest_app_hash": "B" * 64}]:
            for explicit in (False, True):
                with self.subTest(update=update, explicit=explicit):
                    status = copy.deepcopy(original)
                    status["sync_info"].update(update)
                    response = copy.deepcopy(app)
                    if explicit:
                        response["response"]["last_block_height"] = "0"
                    with patch.object(o, "rpc", side_effect=[status, response]):
                        with self.assertRaises(o.Refusal):
                            o.observed(marker)

    def test_applied_hash_requires_exact_hex_or_canonical_base64(self):
        marker, status, app = self.startup_observation()
        status["sync_info"].update(latest_block_height="10", latest_block_hash="A" * 64,
                                   latest_app_hash="B" * 64, catching_up=False,
                                   latest_block_time=dt.datetime.now(dt.timezone.utc).isoformat())
        app["response"]["last_block_height"] = "10"
        encoded = base64.b64encode(bytes.fromhex("C" * 64)).decode("ascii")
        invalid = [None, False, 32, "", "test-wire-hash", "A" * 63, "A" * 65,
                   base64.b64encode(b"x" * 31).decode("ascii"),
                   base64.b64encode(b"x" * 33).decode("ascii"), encoded + "=", encoded + "\n",
                   base64.urlsafe_b64encode(b"\xff" * 32).decode("ascii"),
                   encoded[:-2] + "x="]  # Nonzero unused pad bits are not canonical.
        def read(_port, method):
            return copy.deepcopy(status if method == "status" else app)
        with patch.object(o, "rpc", side_effect=read):
            for value in ["C" * 64, "c" * 64, encoded]:
                app["response"]["last_block_app_hash"] = value
                self.assertTrue(o.observed(marker)["ready"])
            for catching_up in (False, True):
                status["sync_info"]["catching_up"] = catching_up
                for value in invalid:
                    with self.subTest(value=value, catching_up=catching_up):
                        app["response"]["last_block_app_hash"] = value
                        with self.assertRaisesRegex(o.Refusal, "app hash"):
                            o.observed(marker)
                del app["response"]["last_block_app_hash"]
                with self.assertRaisesRegex(o.Refusal, "app hash"):
                    o.observed(marker)

    def test_zero_height_rejects_malformed_optional_hash_and_height(self):
        marker, status, app = self.startup_observation()
        def read(_port, method):
            return copy.deepcopy(status if method == "status" else app)
        with patch.object(o, "rpc", side_effect=read):
            for height in (None, False, 0, "", "00", "-1"):
                with self.subTest(height=height):
                    app["response"]["last_block_height"] = height
                    with self.assertRaisesRegex(o.Refusal, "applied height"):
                        o.observed(marker)
            del app["response"]["last_block_height"]
            for value in (None, False, "invalid", "AA=="):
                with self.subTest(hash=value):
                    app["response"]["last_block_app_hash"] = value
                    with self.assertRaisesRegex(o.Refusal, "app hash"):
                        o.observed(marker)
            app["response"] = None
            with self.assertRaisesRegex(o.Refusal, "ABCI response"):
                o.observed(marker)
            status["sync_info"]["latest_block_time"] = None
            with self.assertRaisesRegex(o.Refusal, "block time"):
                o.observed(marker)

    def test_bootstrap_marker_requires_two_advancing_fresh_observations(self):
        marker = self.make_home()
        state = {"running": False, "removed": False, "sleeps": 0}
        cid = "e5" * 32
        commands = []
        def command(argv, **kwargs):
            commands.append(argv)
            if "create" in argv:
                self.assertEqual(argv[argv.index("--publish") + 1], f"127.0.0.1:{self.port}:26657")
                self.assertNotIn("--rm", argv)
                return cid
            if "start" in argv: state["running"] = True
            if "stop" in argv: state["running"] = False
            if "rm" in argv: state["removed"] = True
            return ""
        def sleep(_seconds):
            state["sleeps"] += 1
            if state["sleeps"] in (1, 2):
                self.assertIsNone(o.read_json(self.home / o.MARKER)["synced"])
            if state["sleeps"] == 3:
                signal.getsignal(signal.SIGINT)(signal.SIGINT, None)
        def read_rpc(_port, method):
            status, app = copy.deepcopy(STARTUP_STATUS), copy.deepcopy(STARTUP_APP)
            status["node_info"]["id"] = marker["node_id"]
            status["validator_info"]["address"] = marker["validator_address"]
            if state["sleeps"]:
                height = 9 + state["sleeps"]
                status["sync_info"].update(latest_block_height=str(height), latest_block_hash="A" * 64,
                                           latest_app_hash="B" * 64, catching_up=False,
                                           latest_block_time=dt.datetime.now(dt.timezone.utc).isoformat())
                app["response"].update(last_block_height=str(height), last_block_app_hash="C" * 64)
            return status if method == "status" else app
        receipt = {"manifest_sha256": marker["manifest_sha256"], "bootstrap_ready": True}
        with patch.object(o, "docker", return_value=["/test/docker"]), \
             patch.object(o, "verify_bundle", return_value=({"release_id": marker["release_id"]}, {}, receipt)) as verify, \
             patch.object(o, "command", side_effect=command), \
             patch.object(o, "owned_state", side_effect=lambda *_args, **_kw: {"running": state["running"], "exit_code": 0, "oom_killed": False}), \
             patch.object(o, "rpc", side_effect=read_rpc), patch.object(o.time, "sleep", side_effect=sleep), \
             contextlib.redirect_stdout(io.StringIO()) as output:
            o.start(self.args, self.bundle)
        self.assertEqual(verify.call_args.args[2], "bootstrap")
        self.assertEqual(o.read_json(self.home / o.MARKER)["synced"]["height"], 11)
        self.assertTrue(state["removed"])
        final = json.loads(output.getvalue().splitlines()[-1])
        self.assertEqual(final["container_exit"]["exit_code"], 0)
        self.assertTrue(self.home.is_dir())

    def test_expired_unsynced_start_cannot_switch_to_resume(self):
        self.make_home()
        with patch.object(o, "verify_bundle", side_effect=o.Refusal("Fresh bootstrap expired")) as verify, \
             patch.object(o, "command") as command:
            with self.assertRaisesRegex(o.Refusal, "expired"):
                o.start(self.args, self.bundle)
            self.assertEqual(verify.call_args.args[2], "bootstrap")
            command.assert_not_called()

    def test_settings_preserve_observer_scope_and_update_only_known_fields(self):
        manifest = {"network": {"peer": "test@127.0.0.1:26656", "rpc_servers": ["http://127.0.0.1:1", "http://127.0.0.1:2"]}}
        settings = o.settings(manifest, {"height": 10, "block_hash": "A" * 64})
        comet = settings["config/config.toml"]
        self.assertEqual(comet["mempool"], {"type": "nop", "broadcast": False})
        self.assertFalse(comet["p2p"]["pex"])
        self.assertEqual(comet["p2p"]["max_num_inbound_peers"], 0)
        for section in ("api", "grpc", "grpc-web"):
            self.assertFalse(settings["config/app.toml"][section]["enable"])
        config = self.root / "config.toml"
        config.write_text('[mempool]\ntype = "flood"\nbroadcast = true\n')
        o.configure(config, {"mempool": comet["mempool"]}, *o.runtime_identity())
        self.assertEqual(o.tomllib.loads(config.read_text())["mempool"], comet["mempool"])
        with self.assertRaisesRegex(o.Refusal, "unique setting"):
            o.configure(config, {"mempool": {"invented": False}}, *o.runtime_identity())


if __name__ == "__main__":
    unittest.main(verbosity=2)
