#!/usr/bin/env python3
"""Offline review regressions. All Docker/chain responses here are UNIT MOCKS."""
import argparse
import contextlib
import copy
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import provenance as p
import runtime_smoke as runtime
from test_kit import IMAGE, load

ship = load("review_ship", "ship-consolidation-safety.py")


class ReviewTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="h1-review-unit-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.host = "unix:///home/operator/.colima/default/docker.sock"
        self.version = {"name": "zerone", "server_name": "zeroned", "commit": p.SOURCE_COMMIT, "version": p.VERSION}

    def inputs(self):
        image = self.root / "image.json"
        folder = self.root / "drill"
        home = folder / "home"
        (home / "data/application.db").mkdir(parents=True)
        (home / "data/application.db/CURRENT").write_text("MANIFEST-000001\n")
        (home / "data/application.db/MANIFEST-000001").write_text("unit mock, not a database")
        (home / ".h1-disposable-copy").write_text("local-only; never a validator home\n")
        built = {"schema": "zerone-h1-testnet-image/v1", "status": "built-local", "reproduced": True,
                 "source_commit": p.SOURCE_COMMIT, "source_sha256": p.SOURCE_ARCHIVE_SHA256,
                 "image_digest": IMAGE.split("@")[1], "binary_sha256": "b" * 64, "ca_bundle_sha256": "c" * 64}
        drill = {"schema": "zerone-h1-state-copy-drill/v1", "passed": True, "input_unchanged": True,
                 "source_commit": p.SOURCE_COMMIT, "source_sha256": p.SOURCE_ARCHIVE_SHA256,
                 "evidence_class": "provided-state-copy-synthetic-abci", "copy_metadata": {"chain_id": ship.CHAIN,
                 "provenance": "UNIT MOCK ONLY, NOT PROVIDED STATE"},
                 "working_file_sha256": runtime.drill_tool.working_inventory(home)}
        p.write_json(image, built)
        p.write_json(folder / "drill.json", drill)
        return image, folder / "drill.json", built, drill

    def container(self):
        return {"State": {"Running": True}, "HostConfig": {"NetworkMode": "none"},
                "Config": {"Image": IMAGE, "Cmd": runtime.START_FLAGS, "Entrypoint": ["/usr/local/bin/zeroned"],
                           "User": f"{os.getuid()}:{os.getgid()}",
                           "Labels": {"money.zerone.binary-sha256": "b" * 64}}}

    def mocked_smoke(self, image, drill, output, *, running=True, listening=True, cleanup_fails=False,
                     dies_during_logs=False):
        calls = []
        container = self.container()
        container["State"]["Running"] = running
        def check_output(argv, **kwargs):
            calls.append(argv)
            if "version" in argv:
                Path(argv[argv.index("--cidfile") + 1]).write_text("e" * 64)
                return json.dumps(self.version)
            if "inspect" in argv:
                return json.dumps([container])
            if "logs" in argv:
                if dies_during_logs:
                    container["State"]["Running"] = False
                return runtime.LISTENING if listening else "starting but no listener"
            raise AssertionError(argv)
        def run(argv, **kwargs):
            calls.append(argv)
            if "--cidfile" in argv:
                Path(argv[argv.index("--cidfile") + 1]).write_text("d" * 64)
            elif cleanup_fails:
                raise subprocess.CalledProcessError(1, argv)
            return subprocess.CompletedProcess(argv, 0, "", "")
        with mock.patch.object(subprocess, "check_output", side_effect=check_output), \
             mock.patch.object(subprocess, "run", side_effect=run), \
             mock.patch.object(runtime.time, "monotonic", side_effect=range(0, 10000, 11)), \
             mock.patch.object(runtime.time, "sleep"):
            result = runtime.smoke(IMAGE, image, drill, output, self.host, True)
        return result, calls

    def test_delayed_logs_cannot_certify_a_now_exited_process(self):
        image, drill, _, _ = self.inputs()
        with self.assertRaisesRegex(ValueError, "not the running"):
            self.mocked_smoke(image, drill, self.root / "bad", dies_during_logs=True)
        self.assertFalse(json.loads((self.root / "bad/runtime.json").read_text())["passed"])

    def test_version_timeout_cleans_up_its_own_container(self):
        image, drill, _, _ = self.inputs()
        with mock.patch.object(subprocess, "run", return_value=subprocess.CompletedProcess([], 0)) as cleanup:
            def version(argv, **kwargs):
                Path(argv[argv.index("--cidfile") + 1]).write_text("e" * 64)
                raise subprocess.TimeoutExpired(argv, 60)
            with mock.patch.object(subprocess, "check_output", side_effect=version), self.assertRaises(subprocess.TimeoutExpired):
                runtime.smoke(IMAGE, image, drill, self.root / "timeout", self.host, True)
        self.assertEqual([call.args[0] for call in cleanup.call_args_list], [
            ["docker", "--host", self.host, "stop", "--time", "10", "e" * 64],
            ["docker", "--host", self.host, "rm", "e" * 64]])
        self.assertFalse(json.loads((self.root / "timeout/runtime.json").read_text())["passed"])

    def test_runtime_receipt_can_be_verified_by_a_different_user(self):
        image, drill, built, _ = self.inputs()
        with mock.patch.object(os, "getuid", return_value=1000), mock.patch.object(os, "getgid", return_value=1000):
            report, _ = self.mocked_smoke(image, drill, self.root / "smoke")
        self.assertEqual(report["container_user"], "1000:1000")
        with mock.patch.object(os, "getuid", return_value=501), mock.patch.object(os, "getgid", return_value=20):
            runtime.validate_receipt(self.root / "smoke/runtime.json", IMAGE, built, drill)

    def test_amino_and_protobuf_upgrade_plans_preserve_exact_identity(self):
        args = argparse.Namespace(height=200, image=IMAGE)
        plan = {"name": ship.NAME, "height": "200", "info": ship.plan_info(IMAGE), "time": "0001-01-01T00:00:00Z"}
        for message in ({"type": "cosmos-sdk/MsgSoftwareUpgrade", "value": {"authority": "zrn1unit", "plan": plan}},
                        {"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade", "authority": "zrn1unit", "plan": plan}):
            self.assertTrue(ship.matching_plan(ship.proposal_plan(message), args))
        for changed in ({"height": "0200"}, {"info": "{}"}, {"name": "wrong"},
                        {"time": "2026-01-01T00:00:00Z"}, {"upgraded_client_state": {}}, {"unknown": "field"}):
            self.assertFalse(ship.matching_plan(dict(plan, **changed), args))
        self.assertIsNone(ship.proposal_plan({"type": "wrong", "value": {"plan": plan}}))
        self.assertIsNone(ship.proposal_plan({"plan": plan}))

    def test_socket_url_validation(self):
        for value in (p.DOCKER_HOST, self.host, "unix:///path with spaces/docker.sock"):
            self.assertEqual(p.validate_docker_host(value), value)
        for value in ("tcp://localhost:2375", "ssh://host", "unix://relative", "unix://host/tmp/a", "unix:///",
                      "unix:////tmp/a", "unix:///tmp/../a", "unix:///tmp/a?x", "unix:///tmp/a#x",
                      "unix:///tmp/%2f", "unix:///tmp/a\n", "unix:///tmp/./a", "unix:///tmp\\a"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                p.validate_docker_host(value)

    def test_native_host_guard_before_output_or_docker(self):
        for system, machine in (("Darwin", "arm64"), ("Linux", "aarch64"), ("Windows", "AMD64")):
            with mock.patch.object(p.platform, "system", return_value=system), \
                 mock.patch.object(p.platform, "machine", return_value=machine), \
                 mock.patch.object(subprocess, "run", side_effect=AssertionError("Docker called")), \
                 self.assertRaisesRegex(ValueError, "native Linux/amd64"):
                p.build(self.root, self.root / "new", self.root, False, self.host)
        self.assertFalse((self.root / "new").exists())
        with mock.patch.object(p.platform, "system", return_value="Linux"), mock.patch.object(p.platform, "machine", return_value="x86_64"):
            p.require_native_host()

    def fake_export(self, repository, output):
        (output / "src").mkdir()
        (output / "src/go.mod").write_text("unit fixture only\n")
        return {"source_sha256": p.SOURCE_ARCHIVE_SHA256,
                "files": [{"path": "go.mod", "sha256": p.sha256(output / "src/go.mod")}]}

    def test_dry_run_records_socket_without_subprocess(self):
        with mock.patch.object(p, "export_source", side_effect=self.fake_export), \
             mock.patch.object(subprocess, "run", side_effect=AssertionError("Docker called")), \
             mock.patch.object(p, "require_native_host", side_effect=AssertionError("guard in dry run")):
            report = p.build(self.root, self.root / "dry", self.root / "missing-modules", True, self.host)
        self.assertEqual(report["docker_host"], self.host)
        self.assertEqual(len(report["commands"]), 8)
        for command in report["commands"]:
            self.assertEqual(command[:3], ["docker", "--host", self.host])
            self.assertIn("--network=none", command)
            self.assertIn("--pull=never", command)
        self.assertIsNone(report["image_digest"])
        self.assertEqual(report["binary_go_metadata"], [])

    def test_build_records_metadata_and_ca_from_each_pinned_builder(self):
        output = self.root / "build"
        def run(argv, **kwargs):
            if "build" in argv:
                mount = next(value for value in argv if value.endswith(",dst=/out"))
                directory = Path(mount.split("src=", 1)[1].split(",dst=")[0])
                (directory / "zeroned").write_bytes(b"\x7fELFunit mock only")
            elif "-m" in argv:
                kwargs["stdout"].write(b"/out/zeroned: go1.25.14\n build GOOS=linux\n build GOARCH=amd64\n build CGO_ENABLED=0\n")
            elif "cat" in argv:
                kwargs["stdout"].write(b"-----BEGIN CERTIFICATE-----\nunit packaging fixture\n-----END CERTIFICATE-----\n")
            return subprocess.CompletedProcess(argv, 0)
        with mock.patch.object(p, "export_source", side_effect=self.fake_export), \
             mock.patch.object(p, "require_native_host"), mock.patch.object(subprocess, "run", side_effect=run):
            report = p.build(self.root, output, self.root, False, self.host)
        self.assertTrue(report["reproduced"])
        self.assertEqual(report["status"], "built-local")
        self.assertEqual(len(report["binary_go_metadata"]), 2)
        self.assertEqual(report["ca_bundle_sha256"], p.sha256(output / "build-step-4.log"))

    def test_build_timeout_cleanup_uses_selected_socket(self):
        output = self.root / "timeout"
        def run(argv, **kwargs):
            if "run" in argv:
                Path(argv[argv.index("--cidfile") + 1]).write_text("d" * 64)
                raise subprocess.TimeoutExpired(argv, 1800)
            return subprocess.CompletedProcess(argv, 0)
        with mock.patch.object(p, "export_source", side_effect=self.fake_export), \
             mock.patch.object(p, "require_native_host"), mock.patch.object(subprocess, "run", side_effect=run) as runner:
            with self.assertRaises(subprocess.TimeoutExpired):
                p.build(self.root, output, self.root, False, self.host)
        self.assertEqual(runner.call_args.args[0][:4], ["docker", "--host", self.host, "stop"])
        self.assertEqual(json.loads((output / "provenance.json").read_text())["status"], "build-failed")

    def test_cli_identity_and_isolated_version_home(self):
        args = argparse.Namespace(zeroned=self.root / "cli", ops_home=self.root / "operator-home")
        with mock.patch.object(subprocess, "check_output", return_value=json.dumps(self.version)) as runner:
            ship.verify_cli(args)
        command = runner.call_args.args[0]
        self.assertEqual(command[1:5], ["version", "--long", "--output", "json"])
        self.assertNotEqual(command[-1], str(args.ops_home))
        for field in self.version:
            altered = dict(self.version, **{field: "wrong"})
            with self.subTest(field=field), self.assertRaises(ValueError):
                runtime.check_version(altered)

    def test_gas_and_fees_are_explicit(self):
        self.assertEqual(ship.validate_gas("450000"), "450000")
        self.assertEqual(ship.validate_fees("400000uzrn"), "400000uzrn")
        for gas in ("auto", "0", "-1", "1.5", " 1", "01"):
            with self.assertRaises(ValueError):
                ship.validate_gas(gas)
        for fees in ("0uzrn", "1uatom", "1.5uzrn", "300000", "-1uzrn"):
            with self.assertRaises(ValueError):
                ship.validate_fees(fees)

    def test_gov_authority_handles_frozen_amino_and_protobuf_accounts(self):
        for account in ({"type": "cosmos-sdk/ModuleAccount", "value": {"name": "gov", "address": "zrn1unit"}},
                        {"@type": "/cosmos.auth.v1beta1.ModuleAccount", "name": "gov", "base_account": {"address": "zrn1unit"}}):
            self.assertEqual(ship.gov_authority(account), "zrn1unit")
        for account in ({"name": "other", "address": "zrn1unit"}, {"name": "gov", "address": "cosmos1unit"},
                        {"type": "wrong", "value": {"name": "gov", "address": "zrn1unit"}}):
            with self.assertRaises(ValueError):
                ship.gov_authority(account)

    def test_runtime_default_never_invokes_docker(self):
        image, drill, _, _ = self.inputs()
        with mock.patch.object(subprocess, "check_output", side_effect=AssertionError("Docker called")), \
             mock.patch.object(subprocess, "run", side_effect=AssertionError("Docker called")):
            result = runtime.smoke(IMAGE, image, drill, self.root / "dry", self.host)
        self.assertFalse(result["passed"])
        self.assertFalse((self.root / "dry/home").exists())
        for command in result["commands"]:
            self.assertIn("--network=none", command)
            self.assertIn("--pull=never", command)

    def test_runtime_success_receipt_requires_version_listener_and_running_container(self):
        image, drill, built, _ = self.inputs()
        result, calls = self.mocked_smoke(image, drill, self.root / "smoke")
        self.assertTrue(result["passed"])
        self.assertTrue(result["input_unchanged"])
        self.assertEqual(calls[-2][:4], ["docker", "--host", self.host, "stop"])
        self.assertEqual(calls[-1][:4], ["docker", "--host", self.host, "rm"])
        receipt = self.root / "smoke/runtime.json"
        runtime.validate_receipt(receipt, IMAGE, built, drill)
        ship.readiness(IMAGE, image, drill, receipt)
        with self.assertRaisesRegex(ValueError, "proof required"):
            ship.readiness(IMAGE, image, drill)

    def test_runtime_failed_start_never_qualifies_and_still_cleans_up(self):
        image, drill, _, _ = self.inputs()
        for running, listening in ((False, True), (True, False)):
            output = self.root / f"bad-{running}"
            with self.assertRaises(ValueError):
                self.mocked_smoke(image, drill, output, running=running, listening=listening)
            self.assertFalse(json.loads((output / "runtime.json").read_text())["passed"])

    def test_runtime_cleanup_failure_never_qualifies(self):
        image, drill, _, _ = self.inputs()
        with self.assertRaises(subprocess.CalledProcessError):
            self.mocked_smoke(image, drill, self.root / "bad", cleanup_fails=True)
        self.assertEqual(json.loads((self.root / "bad/runtime.json").read_text())["status"], "cleanup-failed")

    def test_runtime_evidence_tampering_and_binding_fail_closed(self):
        image, drill, built, _ = self.inputs()
        output = self.root / "smoke"
        report, _ = self.mocked_smoke(image, drill, output)
        receipt = output / "runtime.json"
        for changed in ({"image": IMAGE.replace("a", "e")}, {"drill_receipt_sha256": "e" * 64},
                        {"binary_sha256": "e" * 64}, {"observed_running_seconds": 0}, {"passed": False}):
            p.write_json(receipt, dict(report, **changed))
            with self.subTest(changed=changed), self.assertRaises(ValueError):
                runtime.validate_receipt(receipt, IMAGE, built, drill)
        altered = copy.deepcopy(report)
        altered["commands"][0].remove("--network=none")
        p.write_json(receipt, altered)
        with self.assertRaisesRegex(ValueError, "commands differ"):
            runtime.validate_receipt(receipt, IMAGE, built, drill)
        p.write_json(receipt, report)
        (output / "start.log").write_text("tampered")
        with self.assertRaisesRegex(ValueError, "evidence changed"):
            runtime.validate_receipt(receipt, IMAGE, built, drill)

    def test_changed_or_linked_drill_is_rejected_before_docker(self):
        image, drill, _, _ = self.inputs()
        file = drill.parent / "home/data/application.db/CURRENT"
        file.write_text("changed")
        with mock.patch.object(subprocess, "run", side_effect=AssertionError("Docker called")), self.assertRaises(ValueError):
            runtime.smoke(IMAGE, image, drill, self.root / "smoke", self.host, True)
        file.unlink()
        file.symlink_to(self.root / "unread-secret")
        with mock.patch.object(p, "sha256", side_effect=AssertionError("read before allowlist")), self.assertRaises(ValueError):
            runtime.drill_tool.working_inventory(drill.parent / "home")

    def test_fixture_cannot_qualify_for_image_runtime(self):
        image, drill, built, report = self.inputs()
        report["evidence_class"] = "synthetic-fixture"
        with self.assertRaises(ValueError):
            runtime.check_inputs(IMAGE, built, report)

    def resume_args(self):
        cli = self.root / "unused-cli"
        cli.write_text("unit mock only")
        cli.chmod(0o700)
        config = self.root / "testnet.toml"
        config.write_text('app = "zerone-testnet-1"\n')
        return argparse.Namespace(zeroned=cli, ops_home=self.root / "never-open-home", operator="validator", image=IMAGE,
                                  image_available=True, image_receipt=self.root / "image.json", drill_receipt=self.root / "drill.json",
                                  runtime_receipt=self.root / "runtime.json", fly_config=config, resume_proposal="12", height=200,
                                  halt_polls=4, gas="300000", fees="300000uzrn", proposal=None, node="tcp://invalid.invalid:1")

    def test_timeout_can_resume_without_key_access_submission_or_vote(self):
        args = self.resume_args()
        plan = {"name": ship.NAME, "height": "200", "info": ship.plan_info(IMAGE), "time": "0001-01-01T00:00:00Z"}
        def cli(args, *command):
            if command == ("query", "gov", "proposal", "12"):
                return {"proposal": {"status": "PROPOSAL_STATUS_PASSED", "messages": [
                    {"type": "cosmos-sdk/MsgSoftwareUpgrade", "value": {"authority": "zrn1unit", "plan": plan}}]}}
            if command == ("query", "upgrade", "plan"):
                return {"plan": plan}
            raise AssertionError(command)
        with mock.patch.dict(os.environ, SHIP_ACK=ship.CHAIN), mock.patch.object(ship, "verify_cli"), \
             mock.patch.object(ship, "readiness"), mock.patch.object(ship, "height", return_value="200"), \
             mock.patch.object(ship, "cli", side_effect=cli), \
             mock.patch.object(subprocess, "check_output", side_effect=AssertionError("key/external command")), \
             mock.patch.object(ship, "tx", side_effect=AssertionError("transaction submitted")), \
             mock.patch.object(ship, "wait_halt", side_effect=[ValueError("wait timed out"), (200, 199)]), \
             contextlib.redirect_stdout(io.StringIO()) as stdout:
            with self.assertRaisesRegex(ValueError, "timed out"):
                ship.execute(args)
            ship.execute(args)
        self.assertIn("staged-H stall (H, H-1)", stdout.getvalue())
        self.assertIn("--image " + IMAGE, stdout.getvalue())

    def test_resume_rejects_mismatched_proposal_and_pending_status(self):
        args = self.resume_args()
        for proposal in ({"status": "PROPOSAL_STATUS_VOTING_PERIOD"}, {"status": "PROPOSAL_STATUS_PASSED", "messages": []}):
            with mock.patch.dict(os.environ, SHIP_ACK=ship.CHAIN), mock.patch.object(ship, "verify_cli"), \
                 mock.patch.object(ship, "readiness"), mock.patch.object(ship, "height", return_value="199"), \
                 mock.patch.object(ship, "cli", return_value={"proposal": proposal}), \
                 mock.patch.object(ship, "finish_halt", side_effect=AssertionError("waited on wrong plan")), self.assertRaises(ValueError):
                ship.execute(args)

    def test_legacy_environment_fails_before_network(self):
        args = self.resume_args()
        for name in ("MONIKER", "EXTERNAL_ADDRESS", "EXTRA_START_FLAGS"):
            args.fly_config.write_text(f'app = "zerone-testnet-1"\n[env]\n{name} = "ignored"\n')
            with mock.patch.dict(os.environ, SHIP_ACK=ship.CHAIN), mock.patch.object(ship, "verify_cli"), \
                 mock.patch.object(ship, "readiness"), mock.patch.object(ship, "height", side_effect=AssertionError("RPC called")), \
                 self.assertRaisesRegex(ValueError, "ignores legacy"):
                ship.execute(args)


if __name__ == "__main__":
    unittest.main(verbosity=2)
