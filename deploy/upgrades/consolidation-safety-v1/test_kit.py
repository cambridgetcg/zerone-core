#!/usr/bin/env python3
"""Offline tests. Optional H1K_HELPERS/H1K_FIXTURE enable the real two-source drill."""
import argparse
import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
from typing import Any
from unittest import mock

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import provenance as p


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, HERE / filename)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


ship = load("ship", "ship-consolidation-safety.py")
drill = load("drill", "state-copy-drill.py")
IMAGE = "example.invalid/zerone/h1@sha256:" + "a" * 64


def archive_bytes(archive, name):
    stream = archive.extractfile(name)
    assert stream is not None
    return stream.read()


class KitTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="h1-kit-test-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def copy_fixture(self):
        root = self.root / "copy"
        (root / "application.db").mkdir(parents=True)
        (root / "application.db/CURRENT").write_text("MANIFEST-000001\n")
        (root / "application.db/MANIFEST-000001").write_bytes(b"synthetic inventory fixture, not LevelDB")
        p.write_json(root / "copy.json", {"schema": "zerone-h1-scrubbed-copy/v1", "chain_id": ship.CHAIN,
                     "height": 1, "app_hash": "0" * 64, "backend": "goleveldb", "key_scrubbed": True,
                     "stopped_consistent_copy": True, "block_time": "2026-01-01T00:00:00Z",
                     "evidence_class": "provided-state-copy", "provenance": "unit-test mock only"})
        return root

    def test_source_allowlist_excludes_custody_and_tests(self):
        for path in ("deploy/testnet/artifacts/genesis.json", "app/testdata/key.go", ".env", "x/a/a_test.go",
                     "app/../key.go", "app/keyring-test/key.go", "app/.git/key.go", "config/node_key.json"):
            self.assertFalse(p.selected_source(path), path)
        self.assertTrue(p.selected_source("app/app.go"))
        self.assertTrue(p.selected_source("docs/swagger-ui/swagger.json"))

    def test_tar_normalization(self):
        a = p.tar_bytes([("b", 0o755, b"b"), ("a", 0o644, b"a")])
        self.assertEqual(a, p.tar_bytes([("a", 0o644, b"a"), ("b", 0o755, b"b")]))
        with tarfile.open(fileobj=io.BytesIO(a)) as archive:
            self.assertTrue(all(x.uid == x.gid == x.mtime == 0 for x in archive))

    def test_oci_manifest_digest_not_config_id(self):
        binary = self.root / "synthetic-elf"
        binary.write_bytes(b"\x7fELFunit-test-only-not-executable")
        ca = self.root / "synthetic-public-ca"
        ca.write_bytes(b"-----BEGIN CERTIFICATE-----\nunit-test-packaging-only\n-----END CERTIFICATE-----\n")
        first = p.package_oci(binary, self.root, "b" * 64, ca)
        second = p.package_oci(binary, self.root, "b" * 64, ca)
        self.assertEqual(first, second)
        with tarfile.open(self.root / "image.oci.tar") as archive:
            index = json.loads(archive_bytes(archive, "index.json"))
            descriptor = index["manifests"][0]
            body = archive_bytes(archive, "blobs/sha256/" + descriptor["digest"].split(":")[1])
            self.assertEqual(first["image_digest"], "sha256:" + hashlib.sha256(body).hexdigest())
            manifest = json.loads(body)
            self.assertNotEqual(first["image_digest"], manifest["config"]["digest"])
            config = json.loads(archive_bytes(archive, "blobs/sha256/" + manifest["config"]["digest"].split(":")[1]))
            self.assertEqual(config["config"]["Cmd"], ["start", "--home", "/data/.zeroned", "--minimum-gas-prices", p.MIN_GAS_PRICES])
            self.assertEqual(config["config"]["Env"], ["SSL_CERT_FILE=" + p.CA_PATH])
            layer = archive_bytes(archive, "blobs/sha256/" + manifest["layers"][0]["digest"].split(":")[1])
            with tarfile.open(fileobj=io.BytesIO(layer)) as contents:
                self.assertEqual(archive_bytes(contents, p.CA_PATH.lstrip("/")), ca.read_bytes())

    def test_build_never_pulls_or_networks(self):
        command = p.build_argv(self.root, self.root / "modules", 1)
        self.assertIn("--network=none", command)
        self.assertIn("--pull=never", command)
        self.assertIn("GOPROXY=off", command)
        self.assertIn("-buildvcs=false", command)
        self.assertIn(p.BUILDER_IMAGE, command)
        self.assertNotIn("sh", command)
        self.assertIn("GONOPROXY=none", command)
        self.assertIn("GOVCS=*:off", command)
        self.assertIn("GOENV=off", command)
        self.assertEqual(p.build_argv(self.root, self.root / "modules", 1, True)[-3:], ["go", "mod", "verify"])

    def test_new_output_refuses_existing(self):
        with self.assertRaises(ValueError):
            p.new_output(self.root)

    def test_copy_inventory_is_closed(self):
        root = self.copy_fixture()
        before = drill.inventory(root)
        self.assertEqual(len(before), 3)
        (root / "priv_validator_key.json").write_bytes(b"do not open")
        with mock.patch.object(p, "sha256", side_effect=AssertionError("opened before scrub gate")):
            with self.assertRaises(ValueError):
                drill.inventory(root)

    def test_copy_rejects_symlink_before_read(self):
        root = self.copy_fixture()
        (root / "application.db/000002.ldb").symlink_to(self.root / "missing-secret")
        with mock.patch.object(p, "sha256", side_effect=AssertionError("opened linked copy")):
            with self.assertRaises(ValueError):
                drill.inventory(root)

    def test_copy_rejects_hardlink_and_nested_directory(self):
        root = self.copy_fixture()
        os.link(root / "application.db/CURRENT", root / "application.db/000002.ldb")
        with self.assertRaises(ValueError):
            drill.inventory(root)
        (root / "application.db/000002.ldb").unlink()
        (root / "application.db/nested").mkdir()
        with self.assertRaises(ValueError):
            drill.inventory(root)

    def test_mainnet_and_unknown_backend_never_qualify(self):
        root = self.copy_fixture()
        for change in ({"chain_id": "zerone-1"}, {"backend": "pebbledb"}, {"key_scrubbed": False},
                       {"stopped_consistent_copy": False}, {"block_time": "2026-01-01"}):
            original = json.loads((root / "copy.json").read_text())
            altered = dict(original, **change)
            p.write_json(root / "copy.json", altered)
            with self.assertRaises(ValueError):
                drill.metadata(root, False)
            p.write_json(root / "copy.json", original)

    def test_fixture_cannot_be_called_provided_state(self):
        root = self.copy_fixture()
        m = json.loads((root / "copy.json").read_text())
        m.update(chain_id="zerone-h1-fixture-1", evidence_class="synthetic-fixture")
        p.write_json(root / "copy.json", m)
        with self.assertRaises(ValueError):
            drill.metadata(root, False)
        self.assertEqual(drill.metadata(root, True)[0]["evidence_class"], "synthetic-fixture")

    def test_halt_requires_four_successful_equal_polls(self):
        reader = mock.Mock(side_effect=[(99, 99)] * 4)
        self.assertEqual(ship.wait_halt(reader, 100, polls=4, sleep=lambda _: None), (99, 99))
        self.assertEqual(reader.call_count, 4)
        with self.assertRaises(ValueError):
            ship.wait_halt(lambda: (99, 99), 100, polls=3, sleep=lambda _: None)

    def test_dark_or_failed_poll_breaks_freeze_streak(self):
        for values in ([(99, 99), (99, 99), None, (99, 99), (99, 99), (99, 99)],
                       [(99, 99), (99, 99), ValueError("RPC failed"), (99, 99), (99, 99), (99, 99)],
                       [None] * 6, [(98, 98)] * 6, [(99, 99), (100, 99)] * 3):
            with self.subTest(values=values), self.assertRaises(ValueError):
                ship.wait_halt(mock.Mock(side_effect=values), 100, polls=6, sleep=lambda _: None)

    def test_staged_H_supported_but_committed_H_refused(self):
        self.assertEqual(ship.wait_halt(lambda: (100, 99), 100, polls=4, sleep=lambda _: None), (100, 99))
        for value in ((101, 99), (100, 100), (99, 100)):
            with self.assertRaisesRegex(ValueError, "passed H"):
                ship.wait_halt(lambda: value, 100, sleep=lambda _: None)

    def test_readiness_requires_the_actual_source_pin(self):
        image, state = self.root / "image.json", self.root / "drill.json"
        built = {"schema": "zerone-h1-testnet-image/v1", "status": "built-local", "reproduced": True,
                 "source_commit": p.SOURCE_COMMIT, "image_digest": IMAGE.split("@")[1], "binary_sha256": "b" * 64,
                 "source_sha256": p.SOURCE_ARCHIVE_SHA256}
        report = {"schema": "zerone-h1-state-copy-drill/v1", "passed": True, "input_unchanged": True,
                  "evidence_class": "provided-state-copy-synthetic-abci", "copy_metadata": {"chain_id": ship.CHAIN},
                  "source_commit": p.SOURCE_COMMIT, "source_sha256": p.SOURCE_ARCHIVE_SHA256}
        p.write_json(image, built)
        p.write_json(state, report)
        with mock.patch.object(ship.runtime_smoke, "validate_receipt") as checked:
            ship.readiness(IMAGE, image, state, self.root / "runtime.json")
            checked.assert_called_once()
        for value in (None, "wrong", "a" * 64):
            p.write_json(image, dict(built, source_sha256=value))
            p.write_json(state, dict(report, source_sha256=value))
            with self.assertRaises(ValueError):
                ship.readiness(IMAGE, image, state)
        del built["source_sha256"]
        del report["source_sha256"]
        p.write_json(image, built)
        p.write_json(state, report)
        with self.assertRaises(ValueError):
            ship.readiness(IMAGE, image, state)

    def test_digest_and_deploy_command(self):
        for invalid in ("image:latest", "sha256:" + "a" * 64, IMAGE + " --evil", "x@sha256:" + "a" * 63):
            with self.assertRaises(ValueError):
                ship.validate_image(invalid)
        command = ship.deployment_command("reviewed-testnet.toml", IMAGE)
        self.assertEqual(command[-2:], ["--image", IMAGE])
        self.assertIn("--config", command)
        self.assertEqual(json.loads(ship.plan_info(IMAGE))["commit"], p.SOURCE_COMMIT)
        self.assertEqual(ship.plan_info(IMAGE), json.dumps(json.loads(ship.plan_info(IMAGE)), sort_keys=True, separators=(",", ":")))

    def test_ambiguous_proposal_is_not_latest_proposal(self):
        self.assertEqual(ship.proposal_id({"events": [{"type": "submit_proposal", "attributes": [{"key": "proposal_id", "value": "12"}]}]}), "12")
        with self.assertRaises(ValueError):
            ship.proposal_id({"events": []})

    def test_readiness_refuses_dry_run_and_synthetic_drill(self):
        image, state = self.root / "image.json", self.root / "drill.json"
        built: dict[str, Any] = {"schema": "zerone-h1-testnet-image/v1", "status": "dry-run", "source_commit": p.SOURCE_COMMIT}
        p.write_json(image, built)
        p.write_json(state, {})
        with self.assertRaises(ValueError):
            ship.readiness(IMAGE, image, state)
        built.update(status="built-local", reproduced=True, image_digest=IMAGE.split("@")[1],
                     binary_sha256="b" * 64, source_sha256=p.SOURCE_ARCHIVE_SHA256)
        p.write_json(image, built)
        p.write_json(state, {"schema": "zerone-h1-state-copy-drill/v1", "passed": True, "input_unchanged": True,
                            "evidence_class": "synthetic-fixture"})
        with self.assertRaises(ValueError):
            ship.readiness(IMAGE, image, state)

    def test_real_wrapper_defaults_to_offline_plan(self):
        result = subprocess.run(["bash", str(HERE / "ship-consolidation-safety.sh"), "--chain-id", ship.CHAIN,
                                 "--zeroned", "./nonexistent-current-cli", "--ops-home", "/nonexistent-home",
                                 "--operator", "validator", "--node", "tcp://invalid.invalid:1", "--image", IMAGE,
                                 "--fly-config", "/nonexistent-config", "--height", "200", "--proposal", "/nonexistent-output"],
                                text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(json.loads(result.stdout)["network_contact"])
        self.assertEqual(json.loads(result.stdout)["cli"], str(Path.cwd() / "nonexistent-current-cli"))

    def test_execute_adapter_uses_bonded_operator_and_never_calls_fly(self):
        executable = self.root / "current-zeroned"
        executable.write_text("test executable placeholder, never executed")
        executable.chmod(0o700)
        config = self.root / "testnet.toml"
        config.write_text('app = "zerone-testnet-1"\n')
        args = argparse.Namespace(image=IMAGE, image_available=True, zeroned=executable, ops_home=self.root,
                                  operator="validator", keyring_backend="test", image_receipt=self.root / "image.json",
                                  drill_receipt=self.root / "drill.json", fly_config=config, height=200,
                                  node="tcp://unit-test.invalid:26657", proposal=self.root / "proposal.json",
                                  runtime_receipt=self.root / "runtime.json", resume_proposal=None,
                                  gas="450000", fees="400000uzrn", halt_polls=4)
        calls, heights = [], iter([1, 199, 199, 199, 199])
        def fake_output(argv, **_):
            calls.append(argv)
            command = argv[1:]
            if command[:2] == ["version", "--long"]:
                return json.dumps({"name": "zerone", "server_name": "zeroned", "commit": p.SOURCE_COMMIT, "version": p.VERSION})
            if command[:2] == ["keys", "show"]:
                return "zrnvaloper1fixture\n"
            if command[0] == "status":
                value = {"node_info": {"network": ship.CHAIN}, "sync_info": {"latest_block_height": str(next(heights))}}
            elif command[:3] == ["query", "staking", "validator"]:
                value = {"validator": {"operator_address": "zrnvaloper1fixture", "jailed": False,
                                       "status": "BOND_STATUS_BONDED", "tokens": "1000000"}}
            elif command[:3] == ["query", "auth", "module-account"]:
                value = {"account": {"type": "cosmos-sdk/ModuleAccount", "value": {"name": "gov", "address": "zrn1govfixture"}}}
            elif command[:3] == ["query", "upgrade", "plan"]:
                submitted = any(c[1:4] == ["tx", "gov", "submit-proposal"] for c in calls)
                value = {"plan": {"name": ship.NAME, "height": "200", "info": ship.plan_info(IMAGE)}} if submitted else {}
            elif command[:3] == ["query", "gov", "params"]:
                value = {"params": {"min_deposit": [{"amount": "10000000", "denom": "uzrn"}]}}
            elif command[:3] == ["query", "gov", "proposal"]:
                value = {"proposal": {"status": "PROPOSAL_STATUS_PASSED"}}
            elif command[0] == "tx":
                value = {"code": 0, "txhash": "A" * 64}
            elif command[:2] == ["query", "tx"]:
                value = {"code": 0, "events": [{"type": "submit_proposal", "attributes": [{"key": "proposal_id", "value": "12"}]}]}
            else:
                raise AssertionError(argv)
            return json.dumps(value)
        with mock.patch.dict(os.environ, SHIP_ACK=ship.CHAIN), mock.patch.object(ship, "readiness"), \
             mock.patch.object(ship, "application_height", return_value=199), \
             mock.patch.object(subprocess, "check_output", side_effect=fake_output), \
             mock.patch.object(ship.time, "sleep"), contextlib.redirect_stdout(io.StringIO()) as output:
            # wait_halt's default sleep is bound at definition time: use explicit no-wait adapter.
            original = ship.wait_halt
            with mock.patch.object(ship, "wait_halt", side_effect=lambda reader, target, **kw: original(reader, target, sleep=lambda _: None, **kw)):
                ship.execute(args)
        transactions = [c for c in calls if c[1] == "tx"]
        self.assertEqual(len(transactions), 2)
        for command in transactions:
            self.assertEqual(command[command.index("--from") + 1], "validator")
            self.assertEqual(command[0], str(executable))
            self.assertEqual(command[command.index("--gas") + 1], "450000")
            self.assertEqual(command[command.index("--fees") + 1], "400000uzrn")
        self.assertIn("--image " + IMAGE, output.getvalue())
        self.assertFalse(any(c[0] == "fly" for c in calls))
        self.assertEqual(json.loads(args.proposal.read_text())["deposit"], "10000000uzrn")
        self.assertIn("pre-H stall (H-1, H-1)", output.getvalue())


@unittest.skipUnless(os.environ.get("H1K_HELPERS") and os.environ.get("H1K_FIXTURE"), "run test-offline.py for real two-source fixtures")
class HistoricalIntegrationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="h1-real-drill-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.helpers = Path(os.environ["H1K_HELPERS"])
        self.fixture = Path(os.environ["H1K_FIXTURE"])

    def test_real_old_halt_H1_migration_and_completed_restart(self):
        result = drill.drill(self.fixture, self.helpers, self.root / "run", True)
        self.assertTrue(result["passed"])
        self.assertTrue(result["input_unchanged"])
        self.assertEqual(result["evidence_class"], "synthetic-fixture")
        after = result["phases"][1]["after"]
        self.assertEqual(after["height"], 4)
        self.assertEqual(after["done_height"], 3)
        self.assertEqual(after["marker"], "migrated")
        self.assertEqual({k: after["module_versions"][k] for k in ("knowledge", "claiming_pot", "liquiditypool", "vesting_rewards")},
                         {"knowledge": 6, "claiming_pot": 2, "liquiditypool": 5, "vesting_rewards": 1})

    def test_wrong_input_hash_fails_without_success_receipt(self):
        copy = self.root / "bad-copy"
        shutil.copytree(self.fixture, copy)
        meta = json.loads((copy / "copy.json").read_text())
        meta["app_hash"] = "0" * 64
        p.write_json(copy / "copy.json", meta)
        with self.assertRaisesRegex(ValueError, "stage failed"):
            drill.drill(copy, self.helpers, self.root / "bad-run", True)
        self.assertFalse(json.loads((self.root / "bad-run/drill.json").read_text())["passed"])

    def test_H1_guard_rejects_planless_copy(self):
        home = self.root / "planless"
        (home / "data").mkdir(parents=True)
        shutil.copytree(self.fixture / "application.db", home / "data/application.db")
        (home / ".h1-disposable-copy").write_text("local-only; never a validator home\n")
        meta = json.loads((self.fixture / "copy.json").read_text())
        result = subprocess.run([str(self.helpers / "h1-helper"), "--mode", "migrate", "--home", str(home),
                                 "--chain-id", meta["chain_id"], "--expected-height", str(meta["height"]),
                                 "--expected-app-hash", meta["app_hash"], "--next-block-time", "2026-01-01T00:00:01Z",
                                 "--plan-info", '{"drill":"test"}'], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("requires a committed H1 plan", result.stderr)


if __name__ == "__main__":
    unittest.main(verbosity=2)
