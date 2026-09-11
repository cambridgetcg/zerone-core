#!/usr/bin/env python3
"""Participant isolation, immutable network pins and durable shared-workflow tests."""
import argparse
import base64
import copy
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location("shared_claims", Path(__file__).with_name("shared-claims.py"))
shared = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(shared)
core = shared.core


def descriptor(origin=shared.ORIGIN):
    genesis = {"chain_id": shared.CHAIN, "app_state": {}}
    value = {"schema": "zerone-shared-development/v1", "chain_id": shared.CHAIN,
             "rpc_url": origin, "genesis_url": origin + "/genesis.json", "faucet_url": origin + "/faucet",
             "genesis_sha256": shared.digest(shared.canonical(genesis)), "rpc_genesis_sha256": shared.digest(shared.canonical(genesis)),
             "denom": "uzrn", "knowledge_version": 10, "commitment_scheme": 2, "review_policy_version": 1,
             "account_types": ["human", "agent"], "gas_limit": 2000000, "tx_fee_uzrn": "2000000",
             "source_commit": "a" * 40, "bootstrap_consensus": "single-operator", "reset_policy": "new-chain-id"}
    value["local_test"] = origin.startswith("http://127.0.0.1:")
    return value, genesis


class ParticipantTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="zerone-participant-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.home = self.root / "participant"
        self.descriptor, self.genesis = descriptor()
        self.raw = shared.canonical(self.descriptor)
        self.desc_path = self.root / "descriptor.json"
        self.desc_path.write_bytes(self.raw)
        self.binary = self.root / "zeroned"
        self.binary.write_bytes(b"not executable; subprocess is mocked")
        (self.root / "darwin-acl-check").write_bytes(b"mock helper")
        self.address = "zrn1" + "q" * 38
        self.args = argparse.Namespace(home=str(self.home), descriptor=str(self.desc_path),
            descriptor_sha256=shared.digest(self.raw), binary=str(self.binary), binary_sha256=shared.digest(self.binary.read_bytes()),
            allow_loopback_test=False)

    def network(self, url, payload=None):
        if url.endswith("/genesis.json"):
            return shared.canonical(self.genesis)
        if payload and payload.get("method") == "genesis":
            return shared.canonical({"result": {"genesis": self.genesis}})
        if payload and payload.get("method") == "status":
            return shared.canonical({"result": {"node_info": {"network": shared.CHAIN},
                                                 "sync_info": {"catching_up": False, "latest_block_height": "20"}}})
        raise AssertionError("unexpected request")

    def initialize(self, home=None):
        args = copy.copy(self.args)
        args.home = str(home or self.home)
        def output(binary, home, *arguments, **kwargs):
            return "cosmos_sdk_version: v0.53.8\ncommit: " + self.descriptor["source_commit"] if arguments[0] == "version" else self.address
        with mock.patch.object(shared, "request", side_effect=self.network), mock.patch.object(core.local, "cli", side_effect=output) as cli:
            result = shared.initialize(args)
        self.assertEqual([call.args[2:4] for call in cli.call_args_list], [("version", "--long"), ("keys", "add"), ("keys", "show")])
        self.assertFalse(any("init" in call.args[2:] for call in cli.call_args_list))
        return result

    def workflow(self):
        with mock.patch.object(shared, "request", side_effect=self.network):
            return shared.SharedWorkflow(self.home)

    def test_init_pins_own_home_without_node_key_creation_or_node_init(self):
        result = self.initialize()
        self.assertEqual(result["address"], self.address)
        self.assertEqual(self.home.stat().st_mode & 0o777, 0o700)
        self.assertEqual((self.home / "bin/zeroned").stat().st_mode & 0o777, 0o500)
        shared.assert_no_node_keys(self.home)
        marker = core.read_json(self.home / shared.MARKER)
        self.assertEqual(marker["descriptor_sha256"], shared.digest(self.raw))
        self.assertEqual(set(marker["client_files"]), {"shared-claims.py", "claim-workflow.py", "local-node.py"})
        self.workflow().check_files()

    def test_bad_external_pins_fail_before_home_or_network_or_keys(self):
        for field in ("descriptor_sha256", "binary_sha256"):
            with self.subTest(field=field):
                args = copy.copy(self.args)
                setattr(args, field, "b" * 64)
                with mock.patch.object(shared, "request") as network, mock.patch.object(core.local, "cli") as cli:
                    with self.assertRaisesRegex(shared.Error, "externally supplied"):
                        shared.initialize(args)
                    network.assert_not_called()
                    cli.assert_not_called()
                self.assertFalse(self.home.exists())

    def test_wrong_network_or_genesis_refused_before_key_creation(self):
        for mutation in ("sdk", "rpc", "status"):
            with self.subTest(mutation=mutation):
                def network(url, payload=None):
                    raw = self.network(url, payload)
                    if mutation == "sdk" and url.endswith("genesis.json"):
                        return raw + b"\n"
                    if mutation == "rpc" and payload and payload["method"] == "genesis":
                        return shared.canonical({"result": {"genesis": {"chain_id": "zerone-1"}}})
                    if mutation == "status" and payload and payload["method"] == "status":
                        return raw.replace(b"zerone-dev-1", b"zerone-1")
                    return raw
                with mock.patch.object(shared, "request", side_effect=network), mock.patch.object(core.local, "cli") as cli:
                    with self.assertRaises(shared.Error):
                        shared.initialize(self.args)
                    cli.assert_not_called()
                self.assertFalse(self.home.exists())

    def test_wrong_binary_source_or_sdk_refused_before_keys(self):
        with mock.patch.object(shared, "request", side_effect=self.network), mock.patch.object(core.local, "cli", return_value="cosmos_sdk_version: v0.50.15\ncommit: " + "b" * 40) as cli:
            with self.assertRaisesRegex(shared.Error, "no keys were created"):
                shared.initialize(self.args)
            self.assertEqual(len(cli.call_args_list), 1)
            self.assertEqual(cli.call_args.args[2:], ("version", "--long"))
        shared.assert_no_node_keys(self.home)

    def test_descriptor_rejects_production_urls_cross_origin_unknown_terms_and_types(self):
        mutations = {"chain_id": "zerone-1", "rpc_url": "https://zerone.ai/api/rpc", "faucet_url": "https://other.example/faucet",
                     "gas_limit": True, "review_policy_version": 2, "reset_policy": "reuse-chain-id", "source_commit": "main", "local_test": True}
        for key, value in mutations.items():
            with self.subTest(key=key):
                altered = dict(self.descriptor, **{key: value})
                with self.assertRaises(shared.Error):
                    shared.validate_descriptor(altered)

    def test_loopback_requires_explicit_persisted_test_mode_and_exact_origin(self):
        value, _ = descriptor("http://127.0.0.1:47777")
        with self.assertRaises(shared.Error):
            shared.validate_descriptor(value)
        self.assertEqual(shared.validate_descriptor(value, True), value)
        for origin in ("http://localhost:47777", "http://127.0.0.1:47777/", "http://user@127.0.0.1:47777", "http://127.0.0.1:47777?x", "https://127.0.0.1:47777", shared.ORIGIN):
            with self.subTest(origin=origin), self.assertRaises(shared.Error):
                shared.validate_descriptor(descriptor(origin)[0], True)

    def test_existing_home_cannot_be_reinitialized_even_with_same_pins(self):
        self.initialize()
        before = (self.home / shared.MARKER).read_bytes()
        with mock.patch.object(shared, "request") as network:
            with self.assertRaisesRegex(shared.Error, "new participant home"):
                shared.initialize(self.args)
            network.assert_not_called()
        self.assertEqual(before, (self.home / shared.MARKER).read_bytes())

    def test_descriptor_binary_source_and_node_key_tampering_refused(self):
        self.initialize()
        w = self.workflow()
        for relative in ("network.json", "bin/zeroned"):
            path = self.home / relative
            original = path.read_bytes()
            path.chmod(0o600)
            path.write_bytes(original + b"x")
            with self.assertRaises(shared.Error):
                w.check_files()
            path.write_bytes(original)
        config = self.home / "config"
        config.mkdir(mode=0o700)
        (config / "node_key.json").write_text("public test placeholder, not key material")
        with self.assertRaisesRegex(shared.Error, "node or consensus"):
            w.check_files()

    def test_symlink_fifo_hardlink_and_overbound_inputs_refused(self):
        target = self.root / "small"
        target.write_bytes(b"abcd")
        symlink = self.root / "link"
        symlink.symlink_to(target)
        with self.assertRaises(OSError):
            shared.read_bytes(symlink, 100)
        with self.assertRaises(shared.Error):
            shared.read_bytes(target, 3)
        fifo = self.root / "pipe"
        os.mkfifo(fifo)
        with self.assertRaises(shared.Error):
            shared.read_bytes(fifo, 100)
        hardlink = self.root / "hard"
        os.link(target, hardlink)
        with self.assertRaises(shared.Error):
            shared.read_bytes(target, 100)

    def test_json_duplicate_nonfinite_and_rpc_json_wrapper_rejected(self):
        for value in (b'{"x":1,"x":2}', b'{"x":NaN}', b'[]'):
            with self.subTest(value=value), self.assertRaises((shared.Error, ValueError)):
                shared.json_value(value)
        with mock.patch.object(shared, "request", return_value=b'{"error":{"code":-1}}'):
            with self.assertRaises(shared.Error):
                shared.status_height(self.descriptor)

    def test_redirect_is_not_followed(self):
        with self.assertRaisesRegex(shared.Error, "redirect"):
            shared.NoRedirect().redirect_request(None, None, None, None, None, None)

    def test_http202_retained_only_for_explicit_funding_observation(self):
        response = mock.MagicMock()
        response.__enter__.return_value = response
        response.status = 202
        response.read.return_value = b'{"status":"pending","committed":false}'
        opener = mock.Mock()
        opener.open.return_value = response
        with mock.patch.object(shared.urllib.request, "build_opener", return_value=opener):
            with self.assertRaises(shared.Error):
                shared.request(shared.ORIGIN)
            observed = shared.json_value(shared.request(shared.ORIGIN + "/faucet", {"address": self.address}, allow_pending=True))
            self.assertEqual(observed, {"status": "pending", "committed": False})
            response.status = 201
            with self.assertRaises(shared.Error):
                shared.request(shared.ORIGIN + "/faucet", {"address": self.address}, allow_pending=True)

    def test_only_owned_signer_and_network_bound_records(self):
        self.initialize()
        w = self.workflow()
        w.check_node = mock.Mock(return_value=20)
        w.cli = mock.Mock(return_value="zrn1" + "p" * 38)
        with self.assertRaisesRegex(shared.Error, "differs"):
            w.actor(shared.ACTOR)
        with self.assertRaisesRegex(shared.Error, "only"):
            w.actor("reviewer1")
        record = {**w.record_metadata(), "actor": shared.ACTOR, "address": self.address}
        w.validate_record(record)
        for key in ("descriptor_sha256", "genesis_sha256", "participant_address", "actor", "address", "chain_id"):
            with self.subTest(key=key), self.assertRaises(shared.Error):
                w.validate_record(dict(record, **{key: "different"}))

    def test_shared_attempt_is_durable_and_uses_participant_only(self):
        self.initialize()
        w = self.workflow()
        w.check_node = mock.Mock(return_value=20)
        w.actor = mock.Mock(return_value=self.address)
        encoded = base64.b64encode(b"mock signed tx").decode()
        txhash = shared.digest(b"mock signed tx").upper()
        def cli(*args, **kwargs):
            if args[:2] == ("tx", "encode"):
                return encoded
            if args[:2] == ("tx", "broadcast"):
                attempt = w.records("attempts")[0][1]
                self.assertEqual(attempt["actor"], shared.ACTOR)
                self.assertEqual(attempt["descriptor_sha256"], self.args.descriptor_sha256)
                self.assertEqual(attempt["tx_bytes_base64"], encoded)
                return json.dumps({"txhash": txhash, "code": 0})
            return json.dumps({"body": {"messages": []}})
        w.cli = mock.Mock(side_effect=cli)
        w.tx_observation = mock.Mock(return_value={"txhash": txhash, "height": "21", "code": 0, "gas_wanted": "2000000", "gas_used": "1"})
        result = w.send("commit", shared.ACTOR, ["knowledge", "submit-commitment", "round"])
        self.assertEqual(result["status"], "committed")
        for call in w.cli.call_args_list:
            if "--from" in call.args:
                self.assertEqual(call.args[call.args.index("--from") + 1], shared.ACTOR)
        before = w.cli.call_count
        self.assertEqual(w.retry(txhash)["status"], "committed")
        self.assertEqual(before, w.cli.call_count)

    def test_copied_review_from_other_home_refused_before_signing(self):
        self.initialize()
        w = self.workflow()
        row = {**w.record_metadata(), "claim_id": "claim", "actor": shared.ACTOR, "address": "zrn1" + "p" * 38}
        core.save_json(w.review_path("claim", shared.ACTOR), row, fresh=True)
        w.actor = mock.Mock()
        w.send = mock.Mock()
        with self.assertRaises(shared.Error):
            w.review(argparse.Namespace(claim="claim"), True)
        w.actor.assert_not_called()
        w.send.assert_not_called()

    def test_fund_only_posts_own_address_and_never_retries_automatically(self):
        self.initialize()
        w = self.workflow()
        w.actor = mock.Mock(return_value=self.address)
        with mock.patch.object(shared, "request", side_effect=TimeoutError("ambiguous request")) as request:
            with self.assertRaises(TimeoutError):
                w.fund()
            request.assert_called_once_with(self.descriptor["faucet_url"], {"address": self.address}, allow_pending=True)

    def test_cli_has_no_external_actor_or_reveal_preimage_or_home_reset(self):
        for command in (["reveal", "--home", "HOME", "--claim", "ID", "--vote", "accept"],
                        ["commit", "--home", "HOME", "--claim", "ID", "--actor", "reviewer1"],
                        ["init", "--reset"]):
            with mock.patch("sys.stderr"), self.assertRaises(SystemExit):
                shared.parser().parse_args(command)


if __name__ == "__main__":
    unittest.main()
