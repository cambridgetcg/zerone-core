#!/usr/bin/env python3
"""Bounded transition-control tests; application/consensus execution is rehearsed separately."""
import argparse
import copy
import hashlib
import http.client
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock
import urllib.error

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("upgrade_runtime", HERE / "runtime.py")
r = importlib.util.module_from_spec(spec)
spec.loader.exec_module(r)


class UpgradeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.home = self.root / "home"
        self.home.mkdir(mode=0o700)
        for name in ("config", "data", "dev-faucet", "keyring-test", "public"):
            (self.home / name).mkdir(mode=0o700)
        for name in (*r.CONTROL, *r.IDENTITY):
            r.write_new(self.home / name, b"synthetic unchanged fixture\n")
        r.write_new(self.home / "data/priv_validator_state.json", b'{"height":"9","round":0,"step":3}')
        r.write_new(self.home / "dev-faucet/state.json", b'{"grants":{"old":"retained"}}')
        r.write_new(self.home / "public/network.json", b'{"old":"descriptor"}')
        self.before, self.after = self.root / "old", self.root / "new"
        r.write_new(self.before, b"old executable fixture")
        r.write_new(self.after, b"new executable fixture")
        self.manifest = {"schema": r.SCHEMA, "chain_id": r.CHAIN, "role": "validator", "owner_uid": os.getuid(),
            "source_commit": "a" * 40, "binary_sha256": r.digest(self.before),
            "genesis_sha256": r.digest(self.home / "config/genesis.json"), "rpc_port": 26657,
            "control_sha256": {name: r.digest(self.home / name) for name in r.CONTROL},
            "identity_sha256": {name: r.digest(self.home / name) for name in r.IDENTITY}}
        r.write_new(self.home / r.MARKER, r.json_bytes(self.manifest))
        self.packet = {"schema": "zerone-development-upgrade/v1", "chain_id": r.CHAIN,
            "genesis_sha256": self.manifest["genesis_sha256"], "predecessor_descriptor_sha256": r.digest(self.home / "public/network.json"),
            "plan": {"name": r.UPGRADE_NAME, "height": "20", "info": ""},
            "predecessor": {"knowledge_version": 10, "source_commit": "a" * 40,
                "binaries": {p: r.digest(self.before) for p in ("linux-amd64", "darwin-arm64")}},
            "target": {"knowledge_version": 11, "source_commit": "b" * 40,
                "binaries": {p: r.digest(self.after) for p in ("linux-amd64", "darwin-arm64")}}}
        self.args = argparse.Namespace(predecessor_binary=self.before, binary=self.after, home=str(self.home))
        self.original = self.original_files()

    def tearDown(self):
        self.temp.cleanup()

    def original_files(self):
        return {str(p.relative_to(self.home)): p.read_bytes() for p in self.home.rglob("*") if p.is_file() and r.UPGRADE_DIRECTORY not in p.parts}

    def prepare(self):
        path = self.root / "packet.json"
        path.write_bytes(r.json_bytes(self.packet))
        self.args.upgrade_packet = str(path)
        self.args.upgrade_packet_sha256 = r.digest(path)

    def stage(self):
        self.prepare()
        with mock.patch.object(r, "binary_identity"):
            return r.stage_upgrade(self.args, self.home)

    def test_stage_preserves_original_and_refuses_replacement(self):
        self.assertEqual(self.stage()["status"], "staged-not-applied")
        packet, manifest = r.load_upgrade(self.home, self.args.upgrade_packet_sha256)
        self.assertEqual(packet, self.packet)
        self.assertEqual(manifest, self.manifest)
        self.assertEqual(self.original_files(), self.original)
        with self.assertRaises(FileExistsError):
            self.stage()
        self.assertEqual(self.original_files(), self.original)

    def test_wrong_chain_genesis_binary_and_height_refuse_before_staging(self):
        for change in (lambda p: p.update(chain_id="zerone-1"), lambda p: p.update(genesis_sha256="c" * 64),
                lambda p: p["target"]["binaries"].update({r.upgrade_platform(): "d" * 64}),
                lambda p: p["plan"].update(height="0"), lambda p: p["plan"].update(height="020"),
                lambda p: p["plan"].update(height=20), lambda p: p["plan"].update(info="extra")):
            original = copy.deepcopy(self.packet)
            change(self.packet)
            self.prepare()
            with self.assertRaises(r.RuntimeError):
                r.stage_upgrade(self.args, self.home)
            self.assertFalse((self.home / r.UPGRADE_DIRECTORY).exists())
            self.assertEqual(self.original_files(), self.original)
            self.packet = original

    def test_stage_and_binary_drift_refused(self):
        self.stage()
        with self.assertRaises(r.RuntimeError):
            r.load_upgrade(self.home, "e" * 64)
        path = self.home / r.UPGRADE_DIRECTORY / "target-zeroned"
        path.chmod(0o600)
        path.write_bytes(b"changed")
        with self.assertRaisesRegex(r.RuntimeError, "binary changed"):
            r.load_upgrade(self.home, self.args.upgrade_packet_sha256)

    def test_missing_plan_cannot_select_target_and_wrong_plan_refuses(self):
        self.stage()
        self.assertFalse(r.disk_upgrade_plan(self.home, self.packet))
        with mock.patch.object(r, "run", return_value=None) as run:
            r.run_upgrade(self.args, self.home)
            self.assertFalse(run.call_args.args[0].upgrade_target)
            self.assertEqual(run.call_args.args[0].binary.name, "predecessor-zeroned")
        path = self.home / "data/upgrade-info.json"
        for value in ({"name": "wrong", "height": 20}, {"name": r.UPGRADE_NAME, "height": 21},
                      {"name": r.UPGRADE_NAME, "height": 20, "info": "extra"}):
            path.write_bytes(r.json_bytes(value));path.chmod(0o600)
            with self.assertRaises(r.RuntimeError):
                r.run_upgrade(self.args, self.home)

    def test_halt_then_restart_select_target_but_applied_metadata_is_separate(self):
        self.stage()
        r.write_new(self.home / "data/upgrade-info.json", r.json_bytes(self.packet["plan"]))
        with mock.patch.object(r, "run", return_value=None) as run:
            r.run_upgrade(self.args, self.home)
            self.assertTrue(run.call_args.args[0].upgrade_target)
        for version, done, valid in ((11, "20", True), (10, "20", False), (11, "0", False), (11, "21", False)):
            values = [{"module_versions": [{"name": "knowledge", "version": str(version)}, {"name": "bank", "version": "4"}]}, {"height": done}]
            with mock.patch.object(r, "upgrade_query", side_effect=values):
                if valid:
                    self.assertEqual(r.verify_applied_upgrade(self.after, self.home, self.manifest, self.packet)["applied_height"], "20")
                else:
                    with self.assertRaises(r.RuntimeError):
                        r.verify_applied_upgrade(self.after, self.home, self.manifest, self.packet)
        self.assertFalse((self.home / r.UPGRADE_DIRECTORY / "first-applied-observation.json").exists())

    def predecessor_at_halt(self, rpc, query):
        """Run the predecessor phase over an actual on-disk halt with a fake child.

        rpc/query receive the child's state and may mark it exited before failing.
        Returns run()'s result; self.calls holds (abci_info calls, plan queries, SIGTERMs).
        """
        r.write_new(self.home / "data/upgrade-info.json", r.json_bytes(self.packet["plan"]))
        node, child = mock.Mock(), {"exited": False}
        node.poll.side_effect = lambda: 2 if child["exited"] else None
        args = argparse.Namespace(binary=self.before, upgrade_lock_held=True, staged_upgrade=self.packet,
                                  upgrade_target=False, upgrade_packet_sha256="f" * 64)
        with mock.patch.object(r, "load_upgrade", return_value=(self.packet, dict(self.manifest, local_test=False))), \
             mock.patch.object(r, "rpc", side_effect=lambda *_: rpc(child)) as observed_rpc, \
             mock.patch.object(r, "upgrade_query", side_effect=lambda *_: query(child)) as observed_query, \
             mock.patch.object(r, "status", side_effect=OSError("RPC closed")), \
             mock.patch.object(r.subprocess, "Popen", return_value=node), mock.patch.object(r.time, "sleep"):
            try:
                return r.run(args, self.home)
            finally:
                self.calls = (observed_rpc.call_count, observed_query.call_count, node.send_signal.call_count)
                (self.home / "data/upgrade-info.json").unlink()

    def test_predecessor_exiting_during_halt_checks_is_the_boundary(self):
        # Job 120143536322: the follower's supervisor died on '<urlopen error [Errno 111] Connection refused>'.
        def exit_then(error):
            def call(child):
                child["exited"] = True
                raise error
            return call
        refused = exit_then(urllib.error.URLError(ConnectionRefusedError(111, "Connection refused")))
        # A reply cut short: with Content-Length, rpc()'s bounded read returns the partial
        # body and parse() fails; a chunked reply raises IncompleteRead.
        truncated = exit_then(json.JSONDecodeError("Unterminated string starting at", '{"jsonrpc":"2.', 12))
        incomplete = exit_then(http.client.IncompleteRead(b'{"jsonrpc"'))
        query_refused = exit_then(r.RuntimeError("zeroned query failed: connection refused"))
        halted, plan = (lambda _: {"response": {"last_block_height": "19"}}), (lambda _: {"plan": self.packet["plan"]})
        for rpc, query, calls in ((refused, plan, (1, 0, 0)), (truncated, plan, (1, 0, 0)), (incomplete, plan, (1, 0, 0)),
                                  (halted, query_refused, (1, 1, 0))):
            self.assertEqual(self.predecessor_at_halt(rpc, query), "upgrade-boundary")
            self.assertEqual(self.calls, calls)

    def test_live_predecessor_retries_refused_rpc_but_keeps_halt_checks_strict(self):
        replies = [urllib.error.URLError(ConnectionRefusedError(111, "Connection refused")), "19"]
        def refused_once(_child):
            reply = replies.pop(0)
            if isinstance(reply, Exception):
                raise reply
            return {"response": {"last_block_height": reply}}
        plan = lambda _: {"plan": self.packet["plan"]}
        self.assertEqual(self.predecessor_at_halt(refused_once, plan), "upgrade-boundary")
        self.assertEqual(self.calls, (2, 1, 1))
        # A live node that keeps failing still stops the supervisor, as the first
        # failure did before, so a restart can take the target path.
        def refused(_child):
            raise urllib.error.URLError(ConnectionRefusedError(111, "Connection refused"))
        with self.assertRaises(urllib.error.URLError):
            self.predecessor_at_halt(refused, plan)
        self.assertEqual(self.calls, (5, 0, 1))
        def query_failed(_child):
            raise r.RuntimeError("zeroned query failed: synthetic")
        for rpc, query, message in (((lambda _: {"response": {"last_block_height": "18"}}), plan, "pinned H-1"),
                                    ((lambda _: {"response": {"last_block_height": "19"}}), query_failed, "synthetic")):
            with self.assertRaisesRegex(r.RuntimeError, message):
                self.predecessor_at_halt(rpc, query)
            self.assertEqual(self.calls[2], 1)


class BinaryPatchTests(unittest.TestCase):
    """An explicitly pinned replacement of the applied target's executable; nothing else moves."""
    setUp, tearDown, original_files = UpgradeTests.setUp, UpgradeTests.tearDown, UpgradeTests.original_files
    prepare, stage = UpgradeTests.prepare, UpgradeTests.stage

    def apply(self):
        """The state an applied target leaves: the halt record and its first applied observation."""
        self.stage()
        r.write_new(self.home / "data/upgrade-info.json", r.json_bytes(self.packet["plan"]))
        self.applied = {"module_versions": {"knowledge": 11, "bank": 4}, "applied_height": "20"}
        self.observation = {"schema": "zerone-development-upgrade-observation/v1", "chain_id": r.CHAIN,
            "packet_sha256": self.args.upgrade_packet_sha256, "source_commit": "b" * 40,
            "binary_sha256": r.digest(self.after), "observed_height": "21", **self.applied}
        r.write_new(self.home / r.UPGRADE_DIRECTORY / r.UPGRADE_OBSERVATION, r.json_bytes(self.observation))
        self.patched = self.root / "patched"
        r.write_new(self.patched, b"patched executable fixture")
        platform = r.upgrade_platform()
        self.patch = {"schema": r.PATCH_SCHEMA, "chain_id": r.CHAIN, "genesis_sha256": self.packet["genesis_sha256"],
            "upgrade_packet_sha256": self.args.upgrade_packet_sha256, "knowledge_version": 11, "role": "validator",
            "consensus": "unchanged", "base": {"source_commit": "b" * 40, "binaries": {platform: r.digest(self.after)}},
            "patch": {"source_commit": "c" * 40, "binaries": {platform: r.digest(self.patched)}}}
        self.earlier = self.all_files()

    def all_files(self):
        return {str(p.relative_to(self.home)): p.read_bytes() for p in self.home.rglob("*") if p.is_file()}

    def write_patch(self):
        path = self.root / "patch.json"
        if path.exists():
            path.unlink()
        path.write_bytes(r.json_bytes(self.patch))
        self.args.patch_packet = str(path)
        self.args.patch_packet_sha256 = r.digest(path)
        self.args.binary = self.patched

    def stage_patch(self):
        self.write_patch()
        with mock.patch.object(r, "binary_identity") as identity:
            result = r.stage_patch(self.args, self.home)
        identity.assert_called_once_with(r.patch_directory(self.home, self.args.patch_packet_sha256) / "patch-zeroned", "c" * 40)
        return result

    def load(self):
        packet, original = r.load_upgrade(self.home, self.args.upgrade_packet_sha256)
        return r.load_patch(self.home, self.args.upgrade_packet_sha256, self.args.patch_packet_sha256, packet, original)

    def assert_only_new_patch_files(self):
        after = self.all_files()
        self.assertEqual({k: v for k, v in after.items() if k in self.earlier}, self.earlier)
        prefix = r.PATCH_PREFIX + self.args.patch_packet_sha256 + "/"
        self.assertTrue(all(name.startswith(prefix) for name in set(after) - set(self.earlier)))

    def test_patch_stages_once_beside_untouched_upgrade_and_observation(self):
        self.apply()
        self.assertEqual(self.stage_patch()["status"], "staged-not-running")
        directory = r.patch_directory(self.home, self.args.patch_packet_sha256)
        self.assertEqual(sorted(p.name for p in directory.iterdir()), ["patch-zeroned", "patch.json", "stage.json"])
        self.assertEqual(directory.stat().st_mode & 0o777, 0o700)
        self.assertEqual((directory / "patch-zeroned").stat().st_mode & 0o777, 0o500)
        self.assertEqual(self.load(), self.patch)
        self.assert_only_new_patch_files()
        staged = self.all_files()
        with self.assertRaises(FileExistsError):
            self.stage_patch()
        self.assertEqual(self.all_files(), staged)

    def test_wrong_pins_base_binary_role_and_platform_refuse_before_staging(self):
        self.apply()
        platform = r.upgrade_platform()
        other = next(name for name in r.PLATFORMS if name != platform)
        changes = (lambda p: p.update(chain_id="zerone-1"), lambda p: p.update(schema="zerone-development-upgrade/v1"),
            lambda p: p.update(genesis_sha256="d" * 64), lambda p: p.update(upgrade_packet_sha256="d" * 64),
            lambda p: p.update(knowledge_version=10), lambda p: p.update(knowledge_version="11"),
            lambda p: p.update(role="full-node"), lambda p: p.update(consensus="changed"), lambda p: p.update(extra=True),
            lambda p: p["base"].update(source_commit="a" * 40), lambda p: p["base"]["binaries"].update({platform: r.digest(self.before)}),
            lambda p: p["patch"].update(source_commit="b" * 40), lambda p: p["patch"]["binaries"].update({platform: p["base"]["binaries"][platform]}),
            lambda p: p["patch"]["binaries"].update({platform: "e" * 64}),
            lambda p: p["patch"]["binaries"].update({"windows-amd64": "e" * 64}),
            lambda p: p["patch"]["binaries"].update({other: "e" * 64}),
            lambda p: (p["base"].update(binaries={other: self.packet["target"]["binaries"][other]}), p["patch"].update(binaries={other: "e" * 64})))
        for change in changes:
            original = copy.deepcopy(self.patch)
            change(self.patch)
            self.write_patch()
            with self.subTest(patch=self.patch), self.assertRaises(r.RuntimeError):
                r.stage_patch(self.args, self.home)
            self.assertFalse(r.patch_directory(self.home, self.args.patch_packet_sha256).exists())
            self.assertEqual(self.all_files(), self.earlier)
            self.patch = original
        self.write_patch()
        for pin in ("f" * 64, "0" * 64, self.args.patch_packet_sha256.upper()):
            with self.subTest(pin=pin), self.assertRaises(r.RuntimeError):
                r.stage_patch(argparse.Namespace(**{**vars(self.args), "patch_packet_sha256": pin}), self.home)
        self.assertEqual(self.all_files(), self.earlier)
        self.assertEqual(list(self.home.glob(r.PATCH_PREFIX + "*")), [])

    def test_unapplied_upgrade_or_unobserved_base_refuses(self):
        self.apply()
        self.write_patch()
        observation = self.home / r.UPGRADE_DIRECTORY / r.UPGRADE_OBSERVATION
        observation.unlink()
        with self.assertRaisesRegex(r.RuntimeError, "first applied observation"):
            r.stage_patch(self.args, self.home)
        r.write_new(observation, r.json_bytes(dict(self.observation, binary_sha256=r.digest(self.patched))))
        with self.assertRaisesRegex(r.RuntimeError, "does not record the patch base"):
            r.stage_patch(self.args, self.home)
        observation.unlink()
        r.write_new(observation, r.json_bytes(self.observation))
        (self.home / "data/upgrade-info.json").unlink()
        with self.assertRaisesRegex(r.RuntimeError, "requires the applied upgrade"):
            r.stage_patch(self.args, self.home)
        self.assertEqual(list(self.home.glob(r.PATCH_PREFIX + "*")), [])

    def test_staged_patch_drift_and_other_pins_refused(self):
        self.apply()
        self.stage_patch()
        directory = r.patch_directory(self.home, self.args.patch_packet_sha256)
        packet, original = r.load_upgrade(self.home, self.args.upgrade_packet_sha256)
        with self.assertRaises(FileNotFoundError):
            r.load_patch(self.home, self.args.upgrade_packet_sha256, "e" * 64, packet, original)
        with self.assertRaisesRegex(r.RuntimeError, "no longer matches"):
            r.load_patch(self.home, "e" * 64, self.args.patch_packet_sha256, packet, original)
        def drift(path, value, message):
            saved = path.read_bytes()
            mode = path.stat().st_mode & 0o777
            path.chmod(0o600)
            path.write_bytes(value)
            path.chmod(mode)
            with self.assertRaisesRegex(r.RuntimeError, message):
                self.load()
            path.chmod(0o600)
            path.write_bytes(saved)
            path.chmod(mode)
            self.assertEqual(self.load(), self.patch)
        drift(directory / "patch-zeroned", b"changed", "patch binary changed")
        drift(directory / "patch.json", r.json_bytes(dict(self.patch, role="full-node")), "differs from its external pin")
        stage = r.parse((directory / "stage.json").read_bytes())
        drift(directory / "stage.json", r.json_bytes(dict(stage, platform="windows-amd64")), "no longer matches")
        observation = self.home / r.UPGRADE_DIRECTORY / r.UPGRADE_OBSERVATION
        drift(observation, r.json_bytes(dict(self.observation, observed_height="22")), "changed after the patch was staged")
        directory.chmod(0o755)
        with self.assertRaisesRegex(r.RuntimeError, "Unsafe binary patch directory"):
            self.load()

    def test_run_upgrade_without_patch_pin_ignores_staged_patch(self):
        self.apply()
        self.stage_patch()
        args = argparse.Namespace(home=str(self.home), upgrade_packet_sha256=self.args.upgrade_packet_sha256, patch_packet_sha256=None)
        with mock.patch.object(r, "run", return_value=None) as run:
            r.run_upgrade(args, self.home)
        selected = run.call_args.args[0]
        self.assertTrue(selected.upgrade_target)
        self.assertEqual(selected.binary, self.home / r.UPGRADE_DIRECTORY / "target-zeroned")
        self.assertFalse(hasattr(selected, "binary_patch"))

    def test_run_upgrade_with_patch_pin_selects_only_the_staged_patch(self):
        self.apply()
        self.stage_patch()
        args = argparse.Namespace(home=str(self.home), upgrade_packet_sha256=self.args.upgrade_packet_sha256, patch_packet_sha256=self.args.patch_packet_sha256)
        with mock.patch.object(r, "run", return_value=None) as run:
            r.run_upgrade(args, self.home)
        selected = run.call_args.args[0]
        self.assertEqual(run.call_count, 1)
        self.assertTrue(selected.upgrade_target)
        self.assertEqual(selected.binary, r.patch_directory(self.home, self.args.patch_packet_sha256) / "patch-zeroned")
        self.assertEqual(selected.binary_patch, self.patch)
        with mock.patch.object(r, "run", return_value="upgrade-boundary"), self.assertRaisesRegex(r.RuntimeError, "cannot cross"):
            r.run_upgrade(args, self.home)
        missing = argparse.Namespace(**{**vars(args), "patch_packet_sha256": "e" * 64})
        with mock.patch.object(r, "run") as run, self.assertRaises(OSError):
            r.run_upgrade(missing, self.home)
        run.assert_not_called()
        (self.home / "data/upgrade-info.json").unlink()
        with mock.patch.object(r, "run") as run, self.assertRaisesRegex(r.RuntimeError, "requires the applied upgrade"):
            r.run_upgrade(args, self.home)
        run.assert_not_called()

    def patched_run(self, applied):
        """Run the patched target with fake children until the gateway starts, then let it exit."""
        real = r.load_upgrade
        def load_upgrade(home, pin):
            packet, original = real(home, pin)
            return packet, dict(original, local_test=False, gateway_port=8080, node_id="d" * 40, advertised_peer="", accounts={})
        node, gateway = mock.Mock(), mock.Mock()
        node.poll.return_value, gateway.poll.return_value = None, 0
        packet, _ = real(self.home, self.args.upgrade_packet_sha256)
        args = argparse.Namespace(binary=r.patch_directory(self.home, self.args.patch_packet_sha256) / "patch-zeroned",
            upgrade_lock_held=True, staged_upgrade=packet, upgrade_target=True, binary_patch=self.patch,
            upgrade_packet_sha256=self.args.upgrade_packet_sha256, patch_packet_sha256=self.args.patch_packet_sha256,
            gateway=str(HERE / "gateway.py"))
        heights = iter((30, 31))
        with mock.patch.object(r, "load_upgrade", side_effect=load_upgrade), \
             mock.patch.object(r, "status", side_effect=lambda *_a, **_k: {"height": next(heights), "ready": True}), \
             mock.patch.object(r, "verify_applied_upgrade", return_value=applied), \
             mock.patch.object(r.subprocess, "Popen", side_effect=[node, gateway]) as popen, \
             mock.patch.object(r.time, "sleep"), mock.patch("builtins.print"):
            try:
                r.run(args, self.home)
            finally:
                self.argv = [call.args[0] for call in popen.call_args_list]

    def test_patched_target_keeps_base_observation_and_records_its_own(self):
        self.apply()
        self.stage_patch()
        with self.assertRaisesRegex(r.RuntimeError, "owned node/gateway process exited"):
            self.patched_run(self.applied)
        node_argv, gateway_argv = self.argv
        self.assertEqual(node_argv[0], str(r.patch_directory(self.home, self.args.patch_packet_sha256) / "patch-zeroned"))
        self.assertEqual(gateway_argv[-4:], ["--upgrade-packet-sha256", self.args.upgrade_packet_sha256, "--patch-packet-sha256", self.args.patch_packet_sha256])
        self.assertEqual((self.home / r.UPGRADE_DIRECTORY / r.UPGRADE_OBSERVATION).read_bytes(), r.json_bytes(self.observation))
        patched = r.parse((r.patch_directory(self.home, self.args.patch_packet_sha256) / "first-patched-observation.json").read_bytes())
        self.assertEqual({k: patched[k] for k in ("source_commit", "binary_sha256", "patch_packet_sha256", "applied_height")},
            {"source_commit": "c" * 40, "binary_sha256": r.digest(self.patched), "patch_packet_sha256": self.args.patch_packet_sha256, "applied_height": "20"})
        retained = self.all_files()
        with self.assertRaisesRegex(r.RuntimeError, "conflicts with actual applied state"):
            self.patched_run(dict(self.applied, applied_height="21"))
        self.assertEqual(self.all_files(), retained)
        # Even if the base observation vanished after the patch checks, a patched
        # run refuses rather than recording the base as observed by the patch.
        observation = self.home / r.UPGRADE_DIRECTORY / r.UPGRADE_OBSERVATION
        observation.unlink()
        with mock.patch.object(r, "load_patch", return_value=self.patch), self.assertRaisesRegex(r.RuntimeError, "first applied observation"):
            self.patched_run(self.applied)
        self.assertFalse(observation.exists())


if __name__ == "__main__":
    unittest.main()
