#!/usr/bin/env bash
# No Go build, real zeroned, real keyring, external endpoint, or funded transaction.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
bash -n "${SCRIPT_DIR}/seed-localnet-fixture.sh"
exec python3 - "${SCRIPT_DIR}/seed-localnet-fixture.sh" <<'SEED_TEST_PY'
import contextlib
import copy
import io
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import time
import types
import unittest
from unittest.mock import patch

SCRIPT = Path(sys.argv.pop(1))
module = types.ModuleType("seed_fixture_test_subject")
source = SCRIPT.read_text().split("<<'SEED_FIXTURE_PY'\n", 1)[1].rsplit("\nSEED_FIXTURE_PY", 1)[0]
exec(compile(source, str(SCRIPT), "exec"), module.__dict__)
m = module
PYTHON = str(Path(sys.executable).resolve())


class FixtureTests(unittest.TestCase):
    def setUp(self):
        self.owned = []
        self.fixtures = []

    def tearDown(self):
        for f in self.fixtures:
            for child in reversed(f.children):
                child.stop()
            for own in (f.root, f.evidence):
                if own.path.exists():
                    own.remove()
        for own in reversed(self.owned):
            if own.path.exists():
                own.remove()

    def own(self):
        own = m.Owned("seed-local-unit-")
        self.owned.append(own)
        return own

    def fixture(self):
        args = types.SimpleNamespace(timeout_seconds=30, retain_test_state=False, hashes={})
        f = m.Fixture(args)
        self.fixtures.append(f)
        return f

    def fail(self, code, call, *args, **kwargs):
        with self.assertRaises(m.Failure) as got:
            call(*args, **kwargs)
        self.assertEqual(got.exception.code, code)

    def argv(self):
        # All validation fixtures are inert executable text files, never daemon bytes.
        root = self.own().path
        binary = root / "fake"
        binary.write_text("#!/bin/sh\nexit 0\n")
        binary.chmod(0o700)
        manifest = root / "source.json"
        manifest.write_text('{"local_test":true}')
        manifest.chmod(0o600)
        return ["--ack-disposable-localnet", "--authorize-test-admission", "--authorize-test-feegrant",
                "--zeroned", str(binary), "--helper", str(binary), "--runtime", str(binary),
                "--source-manifest", str(manifest), "--source-commit", "a" * 40,
                "--", str(binary)]

    def test_json_rpc_binary_encodings_not_uri_encodings(self):
        f = self.fixture()
        with patch.object(f, "rpc", return_value={"response": {"code": 0, "height": "3", "value": ""}}) as rpc:
            self.assertEqual(f.abci("/cosmos.bank.v1beta1.Query/Balance", b"\x0a\xff", 3), {})
            self.assertEqual(rpc.call_args.args[1]["data"], "0AFF")
        with patch.object(f, "rpc", side_effect=m.Failure("stop_after_request")) as rpc:
            self.fail("stop_after_request", f.wait_tx, "AB" * 32)
            self.assertEqual(rpc.call_args.args[1]["hash"], m.base64.b64encode(bytes.fromhex("AB" * 32)).decode())

    def test_all_three_opt_ins_required_before_allocation(self):
        for flag in ("--ack-disposable-localnet", "--authorize-test-admission", "--authorize-test-feegrant"):
            argv = self.argv()
            argv.remove(flag)
            with patch.object(m.tempfile, "mkdtemp", side_effect=AssertionError("must not allocate")):
                self.fail("explicit_test_authorizations_required", m.arguments, argv)

    def test_runner_required_even_with_all_operator_authorizations(self):
        argv = self.argv()[:-2]
        self.fail("explicit_runner_required", m.arguments, argv)

    def test_runner_separator_is_mandatory(self):
        argv = self.argv()
        argv.remove("--")
        self.fail("explicit_runner_required", m.arguments, argv)

    def test_timeout_bounds(self):
        for n in (0, 29, 1801):
            argv = self.argv()
            argv[0:0] = ["--timeout-seconds", str(n)]
            self.fail("timeout_out_of_bounds", m.arguments, argv)

    def test_exact_absolute_binary_path(self):
        argv = self.argv()
        argv[argv.index("--zeroned") + 1] = "zeroned"
        self.fail("noncanonical_artifact_path", m.arguments, argv)

    def test_artifact_symlink_and_hardlink_refused(self):
        root = self.own().path
        original = root / "original"
        original.write_text("test")
        original.chmod(0o600)
        alias = root / "alias"
        alias.symlink_to(original)
        self.fail("noncanonical_artifact_path", m.checked_file, str(alias))
        alias.unlink()
        os.link(original, alias)
        self.fail("unsafe_artifact", m.checked_file, str(alias))

    def test_artifact_writeable_by_others_refused(self):
        path = self.own().path / "file"
        path.write_text("test")
        path.chmod(0o666)
        self.fail("unsafe_artifact_owner_mode", m.checked_file, str(path))

    def test_source_must_be_canonical_and_duplicate_free(self):
        argv = self.argv()
        source = Path(argv[argv.index("--source-manifest") + 1])
        source.write_text('{ "test": true }')
        self.fail("source_manifest_not_canonical", m.arguments, argv)
        source.write_text('{"test":1,"test":2}')
        self.fail("duplicate_json_member", m.arguments, argv)
        source.write_text('{"test":1}\n\n')
        self.fail("source_manifest_not_canonical", m.arguments, argv)

    def test_owned_root_private_and_foreign_sibling_preserved(self):
        own, foreign = self.own(), self.own()
        self.assertEqual(own.path.stat().st_mode & 0o777, 0o700)
        sentinel = foreign.path / "untouched"
        sentinel.write_text("user chain")
        own.remove()
        self.assertEqual(sentinel.read_text(), "user chain")

    def test_marker_mismatch_refuses_deletion(self):
        own = self.own()
        own.marker.write_text("not this run")
        self.fail("cleanup_marker_mismatch", own.remove)
        self.assertTrue(own.path.exists())
        own.marker.write_text(own.token)

    def test_marker_symlink_refuses_deletion(self):
        own, foreign = self.own(), self.own()
        target = foreign.path / "token"
        target.write_text(own.token)
        own.marker.unlink()
        own.marker.symlink_to(target)
        self.fail("cleanup_marker_shape", own.remove)
        own.marker.unlink()
        own.marker.write_text(own.token)

    def test_root_inode_replacement_refuses_deletion(self):
        own = self.own()
        saved = own.path.with_name(own.path.name + "-saved")
        own.path.rename(saved)
        own.path.mkdir(mode=0o700)
        try:
            self.fail("cleanup_root_identity", own.remove)
        finally:
            own.path.rmdir()
            saved.rename(own.path)

    def test_cleanup_unlinks_inside_symlink_without_following_it(self):
        own, foreign = self.own(), self.own()
        sentinel = foreign.path / "chain.db"
        sentinel.write_text("untouched")
        (own.path / "foreign-chain").symlink_to(foreign.path, target_is_directory=True)
        own.remove()
        self.assertEqual(sentinel.read_text(), "untouched")

    def test_port_allocator_checks_three_distinct_loopback_ports(self):
        ports = m.allocate_ports()
        self.assertEqual(len(set(ports)), 3)
        self.assertTrue(m.ports_free(ports))
        with socket.socket() as s:
            s.bind(("127.0.0.1", ports[1]))
            s.listen(1)
            self.assertFalse(m.ports_free(ports))
        self.assertTrue(m.ports_free(ports))

    def test_toml_updates_only_exact_section(self):
        path = self.own().path / "config.toml"
        path.write_text('enable = true\n[rpc]\nladdr = "outside"\n[grpc]\nladdr = "other"\n')
        m.toml_set(path, {("rpc", "laddr"): '"127.0.0.1:49999"'})
        self.assertIn('laddr = "other"', path.read_text())
        self.assertIn('laddr = "127.0.0.1:49999"', path.read_text())
        self.fail("missing_config_key", m.toml_set, path, {("p2p", "laddr"): '""'})

    def test_duplicate_config_key_refused(self):
        path = self.own().path / "config.toml"
        path.write_text('[rpc]\nladdr = "a"\nladdr = "b"\n')
        self.fail("duplicate_config_key", m.toml_set, path, {("rpc", "laddr"): '""'})

    def test_child_clean_environment_and_bounded_capture(self):
        f = self.fixture()
        with patch.dict(os.environ, {"SEED_UNIT_AMBIENT_SECRET": "do-not-inherit", "HTTP_PROXY": "http://invalid"}):
            out = f.run([PYTHON, "-c", "import os,json; print(json.dumps(dict(os.environ)))"])
        env = json.loads(out)
        self.assertNotIn("SEED_UNIT_AMBIENT_SECRET", env)
        self.assertNotIn("HTTP_PROXY", env)
        self.assertEqual(env["HOME"], str(f.runner_home))
        self.assertEqual(f.children[-1].process.returncode, 0)

    def test_child_output_limit_stops_and_reaps(self):
        f = self.fixture()
        self.fail("command_output_limit", f.run, [PYTHON, "-c", "import sys; sys.stdout.write('x'*400000); sys.stdout.flush()"])
        self.assertIsNotNone(f.children[-1].process.poll())
        self.assertLessEqual(len(f.children[-1].buffers[0]), m.MAX_JSON)

    def test_key_generation_output_is_discarded_on_both_streams(self):
        f = self.fixture()
        data = f.run([PYTHON, "-c", "import sys; print('fake-mnemonic'); print('fake-private-key',file=sys.stderr)"], discard=True)
        self.assertEqual(data, b"")
        self.assertEqual(f.children[-1].counts, [0, 0])
        self.assertNotIn("fake-mnemonic", m.canonical(f.report).decode())

    def test_exact_owned_child_terminated_foreign_process_survives(self):
        f = self.fixture()
        foreign = subprocess.Popen([PYTHON, "-c", "import time; time.sleep(20)"])
        try:
            self.fail("command_timeout", f.run, [PYTHON, "-c", "import time; time.sleep(20)"], timeout=.15)
            self.assertIsNotNone(f.children[-1].process.poll())
            self.assertIsNone(foreign.poll())
        finally:
            foreign.terminate()
            foreign.wait(timeout=3)

    def test_total_timeout_also_reaps_child(self):
        f = self.fixture()
        f.deadline = time.monotonic() + .15
        self.fail("total_timeout", f.run, [PYTHON, "-c", "import time; time.sleep(20)"])
        self.assertIsNotNone(f.children[-1].process.poll())

    def test_stdin_nonreader_is_bounded(self):
        f = self.fixture()
        self.fail("command_timeout", f.run, [PYTHON, "-c", "import time; time.sleep(20)"],
                  input_bytes=b"x" * m.MAX_JSON, timeout=.15)
        self.assertFalse(any(t.is_alive() for t in f.children[-1].threads))

    def test_stop_reaps_already_exited_child_without_signaling(self):
        f = self.fixture()
        f.run([PYTHON, "-c", "pass"])
        with patch.object(f.children[-1].process, "terminate", side_effect=AssertionError("stale signal")):
            f.children[-1].stop()

    def test_runner_grace_is_separate_from_ordinary_child_grace(self):
        f = self.fixture()
        for runner, expected in ((False, 3), (True, 120)):
            process = types.SimpleNamespace(poll=lambda: None, terminate=lambda: None, returncode=0,
                                            wait=lambda timeout: self.assertEqual(timeout, expected))
            with patch.object(m.subprocess, "Popen", return_value=process):
                child = m.Child(["/unused"], f.env, f.root.path, discard=True, runner=runner)
            child.stop()

    def test_runner_term_waits_for_cooperative_descendant_reap_past_three_seconds(self):
        f = self.fixture()
        # The fake runner owns and waits its one bounded child; no PID discovery.
        code = ("import signal,subprocess,sys,time; from pathlib import Path\n"
                "signal.signal(signal.SIGTERM,lambda *_:None)\n"
                "p=subprocess.Popen([sys.executable,'-c','import time; time.sleep(3.3)'])\n"
                "print('READY',flush=True); p.wait(timeout=5)\n"
                "Path('fake-descendant-reaped').write_text('yes')\n")
        child = m.Child([PYTHON, "-c", code], f.env, f.root.path, runner=True)
        f.children.append(child)
        end = time.monotonic() + 2
        while not child.buffers[0] and time.monotonic() < end:
            time.sleep(.01)
        self.assertIn(b"READY", child.buffers[0])
        child.stop()
        self.assertEqual(child.process.returncode, 0)
        self.assertEqual((f.root.path / "fake-descendant-reaped").read_text(), "yes")
        self.assertFalse(child.forced_kill)

    def test_forced_runner_stop_stays_uncertain_and_preserves_state(self):
        f = self.fixture()
        # This fake is deliberately a leaf; shorten only the test's grace.
        code = "import signal,time; signal.signal(signal.SIGTERM,signal.SIG_IGN); print('READY',flush=True); time.sleep(10)"
        child = m.Child([PYTHON, "-c", code], f.env, f.root.path, runner=True)
        f.children.append(child)
        end = time.monotonic() + 2
        while not child.buffers[0] and time.monotonic() < end:
            time.sleep(.01)
        self.assertIn(b"READY", child.buffers[0])
        with patch.object(m, "RUNNER_TERM_GRACE", .05):
            self.fail("runner_cleanup_uncertain", child.stop)
        self.assertTrue(child.forced_kill)
        self.assertEqual(child.process.returncode, -signal.SIGKILL)
        self.fail("runner_cleanup_uncertain", child.stop)
        f.report["result"] = "PASS"
        with contextlib.redirect_stdout(io.StringIO()):
            rc = f.finish()
        self.assertEqual(rc, 1)
        self.assertEqual(f.report["result"], "FAIL")
        self.assertIn("runner_cleanup_uncertain", f.report["cleanup_errors"])
        self.assertTrue(f.report["ports_closed"])
        self.assertTrue(f.root.path.exists())
        self.assertNotIn("test_state_deleted", f.report)
        self.assertTrue((f.evidence.path / "observations.json").exists())
        # Synthetic leaf is now positively reaped; test teardown owns this state.
        f.children.remove(child)

    def test_signal_terminated_runner_is_uncertain_even_without_our_kill(self):
        f = self.fixture()
        child = m.Child([PYTHON, "-c", "import time; time.sleep(10)"], f.env, f.root.path, runner=True)
        self.fail("runner_cleanup_uncertain", child.stop)
        self.assertEqual(child.process.returncode, -signal.SIGTERM)
        self.assertFalse(child.forced_kill)

    def test_runner_marker_is_explicit_in_run_not_inferred_from_stage(self):
        f = self.fixture()
        f.stage = "runner"
        f.run([PYTHON, "-c", "pass"])
        self.assertFalse(f.children[-1].runner)
        f.run([PYTHON, "-c", "pass"], runner=True)
        self.assertTrue(f.children[-1].runner)

    def test_failure_preserved_when_cleanup_also_fails(self):
        f = self.fixture()
        with patch.object(f.root, "remove", side_effect=m.Failure("cleanup_failed")), contextlib.redirect_stdout(io.StringIO()):
            code = f.finish(m.Failure("runner_failed", 23))
        self.assertEqual(code, 23)
        report = json.loads((f.evidence.path / "evidence.json").read_text())
        self.assertEqual(report["failure"]["code"], "runner_failed")
        self.assertIn("owned_state_cleanup_failed", report["cleanup_errors"])
        self.assertTrue((f.evidence.path / "observations.json").exists())

    def test_evidence_written_before_owned_state_removed(self):
        f = self.fixture()
        original = f.root.remove
        def remove():
            self.assertTrue((f.evidence.path / "observations.json").exists())
            original()
        with patch.object(f.root, "remove", side_effect=remove), contextlib.redirect_stdout(io.StringIO()):
            f.finish(m.Failure("unit_failure"))
        self.assertFalse(f.root.path.exists())
        self.assertTrue((f.evidence.path / "evidence.json").exists())

    def test_failed_evidence_write_prevents_state_deletion(self):
        f = self.fixture()
        with patch.object(m, "write_json", side_effect=OSError("unit injected")), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            rc = f.finish(m.Failure("original", 19))
        self.assertEqual(rc, 19)
        self.assertTrue(f.root.path.exists())

    def test_explicit_retain_keeps_only_own_state(self):
        f = self.fixture()
        f.args.retain_test_state = True
        with contextlib.redirect_stdout(io.StringIO()):
            f.finish(m.Failure("unit_failure"))
        self.assertTrue(f.root.path.exists())
        self.assertEqual(f.report["retained_test_state"], str(f.root.path))

    def test_receipt_closed_schema_and_replay_is_reported(self):
        receipt = {"claimed_tx_hash": "A" * 64, "claimed_address": "test-address", "actual_amount": "222000", "replay_refused": True}
        self.assertEqual(m.validate_receipt(m.canonical(receipt), "test-address"), receipt)
        for key, value, code in (("claimed_tx_hash", "a" * 64, "receipt_tx_hash"),
                                 ("claimed_address", "another", "receipt_credit"),
                                 ("actual_amount", "444000", "receipt_credit"),
                                 ("replay_refused", False, "runner_replay_not_refused"),
                                 ("replay_refused", 1, "runner_replay_not_refused")):
            bad = dict(receipt, **{key: value})
            self.fail(code, m.validate_receipt, m.canonical(bad), "test-address")
        bad = dict(receipt, private_key="not accepted")
        self.fail("receipt_schema", m.validate_receipt, m.canonical(bad), "test-address")

    def test_protobuf_reader_rejects_truncation_and_duplicate_singular(self):
        encoded = m.string_field(1, "claimant") + m.string_field(2, "uzrn")
        self.assertEqual(m.one(m.fields(encoded), 2), b"uzrn")
        self.fail("truncated_protobuf", m.fields, encoded[:-1])
        self.fail("duplicate_protobuf_field", m.one, m.fields(encoded + m.string_field(1, "alias")), 1)
        self.fail("unsupported_protobuf", m.fields, b"\x0d\x00\x00\x00\x00")

    def test_complete_initialization_shapes_with_fake_node_only(self):
        f = self.fixture()
        fake = f.root.path / "fake-node"
        fake.write_text("#!" + PYTHON + "\nimport sys,re,socket,time\nfrom pathlib import Path\nh=Path(sys.argv[sys.argv.index('--home')+1]); text=(h/'config/config.toml').read_text()+(h/'config/app.toml').read_text()\nports=set(map(int,re.findall(r'127\\.0\\.0\\.1:(\\d+)',text))); ss=[]\nfor port in ports:\n s=socket.socket(); s.bind(('127.0.0.1',port)); s.listen(1); ss.append(s)\nprint('READY',flush=True); time.sleep(20)\n")
        fake.chmod(0o700)
        f.args.zeroned, f.args.source_commit = str(fake), "a" * 40
        f.args.lsof = m.shutil.which("lsof", path="/usr/sbin:/usr/bin:/sbin:/bin")
        addresses = {key: "zrn1" + char * 38 for key, char in zip(("validator", "registrar", "sponsor", "claimant"), "qprs")}
        config = f.home / "config"
        called = []
        def cli(*args, **kwargs):
            called.append((args, kwargs))
            genpath = config / "genesis.json"
            if args[0] == "version":
                return b"fake-test-version"
            if args[0] == "init":
                config.mkdir()
                genpath.write_bytes(m.canonical({"app_state": {"auth": {"accounts": []}, "bank": {"balances": []},
                                      "claiming_pot": {"params": {}, "pots": [], "claims": []}, "genutil": {"gen_txs": []}}}))
                (config / "config.toml").write_text('[rpc]\nladdr = ""\npprof_laddr = ""\nunsafe = false\n[p2p]\nladdr = ""\nexternal_address = ""\npersistent_peers = ""\nseeds = ""\npex = true\naddr_book_strict = true\n[statesync]\nenable = false\n[consensus]\ntimeout_propose = ""\ntimeout_commit = ""\n[instrumentation]\nprometheus = false\n')
                (config / "app.toml").write_text('minimum-gas-prices = ""\n[api]\nenable = true\n[grpc]\nenable = true\naddress = ""\n[grpc-web]\nenable = true\n[oracle]\nenabled = false\n[telemetry]\nenabled = false\n')
            elif args[:2] == ("keys", "show"):
                return addresses[args[2]].encode()
            elif args[0] == "add-genesis-account":
                gen = m.parse(genpath.read_bytes())
                accounts = gen["app_state"]["auth"]["accounts"]
                accounts.append({"address": args[1], "account_number": str(len(accounts)), "sequence": "0"})
                gen["app_state"]["bank"]["balances"].append({"address": args[1]})
                genpath.write_bytes(m.canonical(gen))
            elif args[:2] == ("genesis", "gentx"):
                self.assertEqual(args[args.index("--account-number") + 1], "0")
                self.assertEqual(args[args.index("--sequence") + 1], "0")
                self.assertEqual(args[args.index("--fees") + 1], "2000000uzrn")
                gen = m.parse(genpath.read_bytes())
                self.assertIs(type(gen["app_state"]["claiming_pot"]["params"]["bootstrap_daily_admission_cap"]), int)
                gen["app_state"]["genutil"]["gen_txs"] = [{"body": {"messages": [{"@type": "/cosmos.staking.v1beta1.MsgCreateValidator"}]}}]
                genpath.write_bytes(m.canonical(gen))
            return b""
        def wait_height(_target):
            end = time.monotonic() + 2
            while not f.node_child.buffers[0] and time.monotonic() < end:
                time.sleep(.01)
            self.assertIn(b"READY", f.node_child.buffers[0])
        with patch.object(f, "cli", side_effect=cli), patch.object(f, "wait_height", side_effect=wait_height):
            f.initialize()
        genesis = m.parse(f.genesis_path.read_bytes())
        self.assertNotIn(addresses["claimant"], [a["address"] for a in genesis["app_state"]["auth"]["accounts"]])
        self.assertNotIn(addresses["claimant"], [a["address"] for a in genesis["app_state"]["bank"]["balances"]])
        claimant_calls = [(a, k) for a, k in called if a[:3] == ("keys", "add", "claimant")]
        self.assertEqual(claimant_calls[0][1]["home"], f.claimant_home)
        self.assertTrue(claimant_calls[0][1]["discard"])
        self.assertEqual(len(f.report["checks"]["loopback_listeners"]), 3)

    def test_public_receipt_file_mode_and_links(self):
        root = self.own().path
        path = root / "result.json"
        receipt = {"claimed_tx_hash": "A" * 64, "claimed_address": "test-address", "actual_amount": "222000", "replay_refused": True}
        path.write_bytes(m.canonical(receipt))
        path.chmod(0o600)
        self.assertEqual(m.read_receipt(path, "test-address"), receipt)
        path.chmod(0o644)
        self.fail("unsafe_receipt_file", m.read_receipt, path, "test-address")
        path.chmod(0o600)
        os.link(path, root / "alias")
        self.fail("unsafe_receipt_file", m.read_receipt, path, "test-address")

    def test_redirect_handler_refuses_before_following_another_host(self):
        self.assertIsNone(m.NoRedirect().redirect_request(None, None, 302, "redirect", {}, "https://example.invalid/"))

    def test_oversize_stdin_refused_before_process_launch(self):
        f = self.fixture()
        with patch.object(m.subprocess, "Popen", side_effect=AssertionError("must not spawn")):
            self.fail("stdin_limit", m.Child, ["/unused"], f.env, f.root.path, b"x" * (m.MAX_JSON + 1))

    def test_fake_listener_child_reaped_and_all_ports_closed(self):
        f = self.fixture()
        f.ports = m.allocate_ports()
        code = "import socket,sys,json,time; ss=[]\nfor port in json.loads(sys.argv[1]):\n s=socket.socket(); s.bind(('127.0.0.1',port)); s.listen(1); ss.append(s)\nprint('READY',flush=True); time.sleep(20)"
        child = m.Child([PYTHON, "-c", code, json.dumps(f.ports)], f.env, f.root.path)
        f.children.append(child)
        f.node_child = child
        end = time.monotonic() + 2
        while not child.buffers[0] and time.monotonic() < end:
            time.sleep(.01)
        self.assertIn(b"READY", child.buffers[0])
        self.assertFalse(m.ports_free(f.ports))
        with contextlib.redirect_stdout(io.StringIO()):
            rc = f.finish(m.Failure("fake_listener_test", 7))
        self.assertEqual(rc, 7)
        self.assertTrue(f.report["ports_closed"])
        self.assertIsNotNone(child.process.poll())
        self.assertTrue(m.ports_free(f.ports))

    def test_foreign_port_never_killed_and_blocks_cleanup_success(self):
        f = self.fixture()
        f.ports = m.allocate_ports()
        with socket.socket() as s:
            s.bind(("127.0.0.1", f.ports[0]))
            s.listen(1)
            with contextlib.redirect_stdout(io.StringIO()):
                code = f.finish(m.Failure("original", 17))
            self.assertEqual(code, 17)
            self.assertFalse(f.report["ports_closed"])
            self.assertTrue(f.root.path.exists())
            self.assertNotEqual(s.fileno(), -1)

    def test_helper_command_first_envelope_bound_and_disposable(self):
        f = self.fixture()
        f.args.helper = "/unused/test-helper"
        f.trust_path = f.root.path / "trust.json"
        f.profile = {"unit": True}
        captured = {}
        def run(argv, input_bytes, allow_failure):
            captured.update(argv=argv, request=m.parse(input_bytes))
            return m.canonical({"protocol": "zerone-seed-io/0.1", "status": "ok", "command": "inspect",
                                "request_id": "fixture-inspect", "result": {"status": "observed"}})
        with patch.object(f, "run", side_effect=run):
            f.helper("inspect", policy={}, node={}, height=None)
        self.assertEqual(captured["argv"][1], "inspect")
        self.assertIn("--disposable-test", captured["argv"])
        self.assertIsNone(captured["request"]["height"])
        self.assertEqual(captured["request"]["timeout_ms"], 10000)

    def test_observation_unknown_never_becomes_eligibility(self):
        f = self.fixture()
        f.policy, f.node = {}, {}
        with patch.object(f, "helper", return_value={"status": "unknown", "reason": "unavailable"}):
            self.fail("native_observation_unknown", f.inspect)
        self.assertEqual(f.report["observation_failure"], {"reason": "unavailable"})
        with patch.object(f, "helper", return_value={"status": "unknown", "reason": "PRIVATE_UNTRUSTED_TEXT"}):
            self.fail("native_observation_unknown", f.inspect)
        self.assertEqual(f.report["observation_failure"], {"reason": "unrecognized"})
        self.assertNotIn("PRIVATE_UNTRUSTED_TEXT", m.canonical(f.report).decode())

    def test_abci_allowlist_does_not_expose_unrelated_queries(self):
        f = self.fixture()
        self.fail("fixture_abci_method", f.abci, "/cosmos.tx.v1beta1.Service/Simulate", b"", 1)
        self.fail("fixture_rpc_method", f.rpc, "broadcast_tx_sync")

    def confirmation_fixture(self):
        f = self.fixture()
        f.claimant, f.chain = "claimant-test", "seed-local-unit"
        f.addresses = {"sponsor": "sponsor-test"}
        f.policy = {"pot_id": "bootstrap-claimant-test"}
        f.genesis_path = f.root.path / "genesis.json"
        f.genesis_path.write_bytes(b"{}")
        f.report["genesis_hash"] = m.digest(b"{}")
        decoded = {"body": {"messages": [{"@type": m.CLAIM, "claimant": f.claimant, "pot_id": f.policy["pot_id"]}]},
                   "auth_info": {"fee": {"granter": "sponsor-test", "payer": "", "gas_limit": "2000000",
                                         "amount": [{"denom": "uzrn", "amount": "2000000"}]}}}
        after = {"prior_claim": {"amount_uzrn": "222000", "claimed_at": "10", "claimant": f.claimant, "pot_id": f.policy["pot_id"]},
                 "pot": {"claimed_amount_uzrn": "222000", "status": "depleted"}, "claimant": {"sequence": "1"},
                 "evidence": {"anchor": {"height": "11"}}, "allowance": {"status": "found", "spend_limit_uzrn": "8000000"},
                 "sponsor_balance_uzrn": "18000000"}
        receipt = {"claimed_tx_hash": "A" * 64, "claimed_address": f.claimant, "actual_amount": "222000", "replay_refused": True}
        tx = {"tx": "unit-only", "height": "10", "tx_result": {"gas_used": "100000"}}
        return f, decoded, after, receipt, tx

    def confirm_with(self, f, decoded, after, receipt, tx, balance="222000"):
        with patch.object(f, "wait_tx", return_value=tx) as wait, patch.object(f, "cli", return_value=m.canonical(decoded)), \
             patch.object(f, "inspect", return_value=after), patch.object(f, "balance", return_value=balance):
            f.confirm(receipt, {"sponsor_balance_uzrn": "20000000"})
            wait.assert_called_once_with(receipt["claimed_tx_hash"])

    def test_independent_confirmation_accepts_exact_native_poststate(self):
        values = self.confirmation_fixture()
        self.confirm_with(*values)
        self.assertEqual(values[0].report["result"], "PASS")
        self.assertIn("runner-reported", values[0].report["replay_evidence"])

    def test_callback_exit_zero_cannot_substitute_for_native_state(self):
        for key, value, code in (("prior_claim", None, "native_claim_mismatch"),
                                 ("pot", {"claimed_amount_uzrn": "0", "status": "active"}, "pot_not_depleted"),
                                 ("claimant", {"sequence": "2"}, "claim_sequence_not_once"),
                                 ("allowance", {"status": "found", "spend_limit_uzrn": "9000000"}, "allowance_consumption_mismatch"),
                                 ("sponsor_balance_uzrn", "19000000", "sponsor_balance_mismatch")):
            f, decoded, after, receipt, tx = self.confirmation_fixture()
            after[key] = value
            self.fail(code, self.confirm_with, f, decoded, after, receipt, tx)
            self.assertEqual(f.report["result"], "FAIL")

    def test_receipt_credit_requires_actual_bank_balance(self):
        self.fail("credited_balance_mismatch", self.confirm_with, *self.confirmation_fixture(), balance="0")

    def test_wrong_claim_tx_or_sponsor_refused(self):
        f, decoded, after, receipt, tx = self.confirmation_fixture()
        decoded["body"]["messages"].append(dict(decoded["body"]["messages"][0]))
        self.fail("unexpected_claim_message", self.confirm_with, f, decoded, after, receipt, tx)
        decoded["body"]["messages"].pop()
        decoded["auth_info"]["fee"]["granter"] = "other-sponsor"
        self.fail("claim_not_sponsored", self.confirm_with, f, decoded, after, receipt, tx)

    def test_consensus_floor_measured_gas_and_sponsor_fee_ceiling(self):
        f, decoded, after, receipt, tx = self.confirmation_fixture()
        decoded["auth_info"]["fee"]["gas_limit"] = "22221"
        self.fail("claim_fee_or_floor", self.confirm_with, f, decoded, after, receipt, tx)
        decoded["auth_info"]["fee"]["gas_limit"] = "2000000"
        tx["tx_result"]["gas_used"] = "2000001"
        self.fail("measured_gas_invalid", self.confirm_with, f, decoded, after, receipt, tx)

    def test_source_hash_drift_prevents_pass(self):
        f, decoded, after, receipt, tx = self.confirmation_fixture()
        artifact = f.root.path / "binary"
        artifact.write_text("before")
        artifact.chmod(0o600)
        f.args.zeroned = str(artifact)
        f.args.hashes = {"zeroned": m.checked_file(artifact)}
        artifact.write_text("after")
        self.fail("artifact_drift", self.confirm_with, f, decoded, after, receipt, tx)

    def test_nonzero_native_tx_code_rejected_even_with_callback_success(self):
        f = self.fixture()
        with patch.object(f, "rpc", return_value={"hash": "A" * 64, "tx_result": {"code": 8}}):
            self.fail("transaction_failed", f.wait_tx, "A" * 64)

    def test_shell_entry_fake_init_failure_does_not_leak_logs_or_keep_keys(self):
        argv = self.argv()
        fake = Path(argv[argv.index("--zeroned") + 1])
        fake.write_text("#!/bin/sh\nif [ \"$1\" = version ]; then printf 'fake-test-version\\n'; exit 0; fi\nprintf 'NEVER_PUBLIC_FAKE_MNEMONIC\\n' >&2\nexit 19\n")
        output = subprocess.run(["/bin/bash", str(SCRIPT), *argv], capture_output=True, timeout=15)
        self.assertEqual(output.returncode, 1)
        self.assertNotIn(b"NEVER_PUBLIC", output.stdout + output.stderr)
        summary = json.loads(output.stdout)
        evidence = Path(summary["evidence_file"])
        report = json.loads(evidence.read_text())
        self.assertEqual(report["failure"], {"stage": "initialize", "code": "command_failed"})
        self.assertTrue(report["test_state_deleted"])
        self.assertTrue(report["ports_closed"])
        self.assertNotIn("NEVER_PUBLIC", evidence.read_text())
        # This test process owns the child fixture evidence, with its exact marker.
        for p in evidence.parent.iterdir():
            self.assertTrue(p.is_file() and not p.is_symlink())
            p.unlink()
        evidence.parent.rmdir()


unittest.main(verbosity=2)
SEED_TEST_PY
