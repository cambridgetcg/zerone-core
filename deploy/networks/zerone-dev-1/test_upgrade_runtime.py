#!/usr/bin/env python3
"""Bounded transition-control tests; application/consensus execution is rehearsed separately."""
import argparse
import copy
import hashlib
import importlib.util
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

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


if __name__ == "__main__":
    unittest.main()
