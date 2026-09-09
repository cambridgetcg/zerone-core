#!/usr/bin/env python3
"""Offline observer lifecycle tests; Docker calls are controlled test doubles.

Only the redirect test starts a loopback HTTP server. No node, account, live
endpoint, Docker daemon, private key or release authority is accessed.
"""
import argparse
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
        app = {"response": {"last_block_height": "9", "last_block_app_hash": "test-wire-hash"}}
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
            if state["sleeps"] == 1:
                self.assertIsNone(o.read_json(self.home / o.MARKER)["synced"])
            if state["sleeps"] == 2:
                signal.getsignal(signal.SIGINT)(signal.SIGINT, None)
        observations = [{"ready": True, "height": 10, "block_hash": "A" * 64},
                        {"ready": True, "height": 11, "block_hash": "B" * 64}]
        receipt = {"manifest_sha256": marker["manifest_sha256"], "bootstrap_ready": True}
        with patch.object(o, "docker", return_value=["/test/docker"]), \
             patch.object(o, "verify_bundle", return_value=({"release_id": marker["release_id"]}, {}, receipt)) as verify, \
             patch.object(o, "command", side_effect=command), \
             patch.object(o, "owned_state", side_effect=lambda *_args, **_kw: {"running": state["running"], "exit_code": 0, "oom_killed": False}), \
             patch.object(o, "observed", side_effect=observations), patch.object(o.time, "sleep", side_effect=sleep), \
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
