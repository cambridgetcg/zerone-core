#!/usr/bin/env python3
"""Participant-owned development signing client (Python 3.11+, zerone-dev-1 only).

The private home contains this participant's development signing and identity
keys. It never imports a node home or creates consensus keys. Endpoint/genesis
checks identify the selected operator service; they are not a light-client proof.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import time
import urllib.parse
import urllib.request

SPEC = importlib.util.spec_from_file_location("claim_workflow", Path(__file__).with_name("claim-workflow.py"))
core = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(core)
Error = core.WorkflowError
CHAIN = "zerone-dev-1"
ORIGIN = "https://zerone-dev-1.fly.dev"
SCHEMA = "zerone-shared-participant/v1"
MARKER = ".zerone-shared-participant.json"
ACTOR = "participant"
ADDRESS = r"zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}"
NOTICE = ("Shared development chain; development funds have no promised value. "
          "You control this home's signing and identity keys. Separate addresses do not prove independent reviewers. "
          "History is an operator endpoint observation, not a verified inclusion proof.")


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False, allow_nan=False).encode()


def hash_value(value, label):
    if not isinstance(value, str) or not re.fullmatch(r"[0-9a-f]{64}", value):
        raise Error(f"Invalid {label} SHA256.")
    return value


def read_bytes(path, maximum, private=False):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        before = os.fstat(stream.fileno())
        if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1:
            raise Error("Input must be a regular file with one link.")
        if private and (before.st_uid != os.getuid() or before.st_mode & 0o077):
            raise Error("Participant control files must be private and owned.")
        raw = stream.read(maximum + 1)
        after = os.fstat(stream.fileno())
        current = os.lstat(path)
    identity = lambda value: (value.st_dev, value.st_ino, value.st_size, value.st_mtime_ns, value.st_ctime_ns)
    if len(raw) > maximum or identity(before) != identity(after) or identity(after) != identity(current):
        raise Error("Input exceeded its bound or changed while reading.")
    return raw


def write_bytes(path, raw, mode=0o600):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, mode)
    with os.fdopen(fd, "wb") as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())
    path.chmod(mode)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise Error("The pinned development endpoint redirected; no redirect was followed.")


def request(url, payload=None, *, allow_pending=False):
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    headers = {"User-Agent": "zerone-shared-claims/1.0", "Accept": "application/json"}
    data = None if payload is None else canonical(payload)
    if data is not None:
        headers["Content-Type"] = "application/json"
    with opener.open(urllib.request.Request(url, data=data, headers=headers), timeout=25) as response:
        if response.status not in ((200, 202) if allow_pending else (200,)):
            raise Error("Development endpoint did not return HTTP 200.")
        raw = response.read(core.MAX_JSON + 1)
    if len(raw) > core.MAX_JSON:
        raise Error("Development endpoint response exceeds 8 MiB.")
    return raw


def json_value(raw):
    value = core.parse_json(raw.decode("utf-8"))
    canonical(value)  # Refuse non-finite numbers accepted by Python's JSON reader.
    if not isinstance(value, dict):
        raise Error("Expected a JSON object.")
    return value


def validate_descriptor(value, allow_loopback=False):
    if value.get("schema") != "zerone-shared-development/v1" or value.get("chain_id") != CHAIN:
        raise Error("Descriptor is not the selected zerone-dev-1 development network.")
    expected = {"denom": "uzrn", "knowledge_version": 10, "commitment_scheme": 2,
                "review_policy_version": 1, "account_types": ["human", "agent"], "gas_limit": 2000000,
                "tx_fee_uzrn": "2000000", "bootstrap_consensus": "single-operator", "reset_policy": "new-chain-id"}
    for key, wanted in expected.items():
        if type(value.get(key)) is not type(wanted) or value[key] != wanted:
            raise Error(f"Unsupported descriptor {key}.")
    if type(value.get("local_test")) is not bool or value["local_test"] != allow_loopback:
        raise Error("Descriptor test mode does not match the explicitly selected mode.")
    if not isinstance(value.get("source_commit"), str) or not re.fullmatch(r"[0-9a-f]{40}", value["source_commit"]):
        raise Error("Descriptor needs an exact runtime source commit.")
    for key in ("genesis_sha256", "rpc_genesis_sha256"):
        hash_value(value.get(key), key)
    origin = value.get("rpc_url")
    if not isinstance(origin, str):
        raise Error("Missing RPC origin.")
    if allow_loopback:
        parts = urllib.parse.urlsplit(origin)
        if parts.scheme != "http" or parts.hostname != "127.0.0.1" or not parts.port or not 1024 <= parts.port <= 65535 or origin != f"http://127.0.0.1:{parts.port}":
            raise Error("Test mode permits only an explicit 127.0.0.1 HTTP port.")
    elif origin != ORIGIN:
        raise Error("Only the pinned HTTPS development origin is supported.")
    if value.get("genesis_url") != origin + "/genesis.json" or value.get("faucet_url") != origin + "/faucet":
        raise Error("Genesis and faucet must be exact same-origin paths.")
    return value


def verify_network(descriptor):
    raw = request(descriptor["genesis_url"])
    if digest(raw) != descriptor["genesis_sha256"] or json_value(raw).get("chain_id") != CHAIN:
        raise Error("Served SDK genesis differs from the pinned descriptor.")
    rpc = json_value(request(descriptor["rpc_url"], {"jsonrpc": "2.0", "id": 1, "method": "genesis", "params": {}}))
    genesis = rpc.get("result", {}).get("genesis")
    if not isinstance(genesis, dict) or genesis.get("chain_id") != CHAIN or digest(canonical(genesis)) != descriptor["rpc_genesis_sha256"]:
        raise Error("RPC genesis differs from the pinned canonical genesis.")
    status_height(descriptor)
    return raw


def status_height(descriptor):
    value = json_value(request(descriptor["rpc_url"], {"jsonrpc": "2.0", "id": 2, "method": "status", "params": {}}))
    result = value.get("result", {})
    if result.get("node_info", {}).get("network") != CHAIN or result.get("sync_info", {}).get("catching_up") is not False:
        raise Error("RPC reports a different chain or is catching up.")
    return core.integer(result.get("sync_info", {}).get("latest_block_height"), "development block height", 1)


def initialize(args):
    home = core.local.home_path(args.home)
    if home.exists():
        raise Error("Init requires a new participant home; existing identities are never replaced or rebound.")
    raw = read_bytes(Path(args.descriptor).expanduser(), 1024 * 1024)
    if digest(raw) != hash_value(args.descriptor_sha256, "external descriptor"):
        raise Error("Descriptor does not match the externally supplied SHA256.")
    descriptor = validate_descriptor(json_value(raw), args.allow_loopback_test)
    source = Path(args.binary).expanduser()
    binary = read_bytes(source, 200 * 1024 * 1024)
    if digest(binary) != hash_value(args.binary_sha256, "external binary"):
        raise Error("Binary does not match the externally supplied SHA256.")
    auxiliary = None
    if sys.platform == "darwin":
        auxiliary = read_bytes(source.parent / "darwin-acl-check", 2 * 1024 * 1024)
    genesis = verify_network(descriptor)  # All externally pinned checks precede key creation.
    home.mkdir(mode=0o700)
    for name in ("bin", "claim-workflow"):
        core.private_directory(home / name)
    write_bytes(home / "bin/zeroned", binary, 0o500)
    if auxiliary is not None:
        write_bytes(home / "bin/darwin-acl-check", auxiliary, 0o555)
    write_bytes(home / "network.json", raw, 0o400)
    write_bytes(home / "pinned-genesis.json", genesis, 0o400)
    version = core.local.cli(home / "bin/zeroned", home, "version", "--long")
    if not re.search(r"(?m)^cosmos_sdk_version:\s*v0\.53\.8\s*$", version) or not re.search(r"(?m)^commit:\s*" + descriptor["source_commit"] + r"\s*$", version):
        raise Error("Pinned binary does not report the descriptor's exact source commit and SDK v0.53.8; no keys were created.")
    # Never run `zeroned init`: SDK client setup alone creates no consensus keys.
    core.local.cli(home / "bin/zeroned", home, "keys", "add", ACTOR, "--keyring-backend", "test", secret=True)
    address = core.local.cli(home / "bin/zeroned", home, "keys", "show", ACTOR, "-a", "--keyring-backend", "test", secret=True)
    if not re.fullmatch(ADDRESS, address):
        raise Error("The new local signing key has an unexpected address.")
    marker = {"schema": SCHEMA, "chain_id": CHAIN, "owner_uid": os.getuid(), "address": address,
              "descriptor_sha256": digest(raw), "binary_sha256": digest(binary), "allow_loopback_test": args.allow_loopback_test,
              "genesis_sha256": digest(genesis), "darwin_acl_helper_sha256": digest(auxiliary) if auxiliary is not None else None,
              "client_files": {name: digest(read_bytes(Path(__file__).with_name(name), 1024 * 1024)) for name in
                               ("shared-claims.py", "claim-workflow.py", "local-node.py")}}
    assert_no_node_keys(home)
    core.save_json(home / MARKER, marker, fresh=True)
    return {"schema": SCHEMA, "chain_id": CHAIN, "address": address, "descriptor_sha256": digest(raw),
            "loopback_test": args.allow_loopback_test, "notice": NOTICE,
            "key_storage": "Owner-only development test keyring; keep the entire participant home private."}


def assert_no_node_keys(home):
    for name in ("config/node_key.json", "config/priv_validator_key.json", "data/priv_validator_state.json"):
        if os.path.lexists(home / name):
            raise Error("A participant home must not contain node or consensus signing keys/state.")


class SharedWorkflow(core.Workflow):
    def __init__(self, home, wait=30):
        self.home = core.local.home_path(str(home))
        core.private_directory(self.home)
        self.manifest = core.read_json(self.home / MARKER)
        self.descriptor = validate_descriptor(json_value(read_bytes(self.home / "network.json", 1024 * 1024, True)),
                                              self.manifest.get("allow_loopback_test") is True)
        self.chain = CHAIN
        self.rpc = self.descriptor["rpc_url"]
        self.wait = wait
        self.accounts = {ACTOR: self.manifest.get("address")}
        self.directory = self.home / "claim-workflow"
        self.check_files()
        for path in (self.directory, self.directory / "attempts", self.directory / "reviews", self.directory / "watched"):
            core.private_directory(path)
        verify_network(self.descriptor)

    def check_files(self):
        current = core.read_json(self.home / MARKER)
        if current != self.manifest or current.get("schema") != SCHEMA or current.get("chain_id") != CHAIN or type(current.get("allow_loopback_test")) is not bool or current.get("owner_uid") != os.getuid():
            raise Error("Participant identity/descriptor binding changed or is invalid.")
        if not isinstance(current.get("address"), str) or not re.fullmatch(ADDRESS, current["address"]):
            raise Error("Invalid participant signing address.")
        for path, key, bound in (("network.json", "descriptor_sha256", 1024 * 1024),
                                 ("pinned-genesis.json", "genesis_sha256", core.MAX_JSON),
                                 ("bin/zeroned", "binary_sha256", 200 * 1024 * 1024)):
            if digest(read_bytes(self.home / path, bound, True)) != hash_value(current.get(key), key):
                raise Error("Pinned participant control bytes changed.")
        auxiliary = current.get("darwin_acl_helper_sha256")
        if sys.platform == "darwin" and auxiliary is None:
            raise Error("Missing retained Darwin identity helper.")
        if auxiliary is not None and digest(read_bytes(self.home / "bin/darwin-acl-check", 2 * 1024 * 1024)) != hash_value(auxiliary, "Darwin helper"):
            raise Error("Retained identity helper changed.")
        expected_names = {"shared-claims.py", "claim-workflow.py", "local-node.py"}
        if not isinstance(current.get("client_files"), dict) or set(current["client_files"]) != expected_names:
            raise Error("Missing pinned client source files.")
        for name, expected in current["client_files"].items():
            if digest(read_bytes(Path(__file__).with_name(name), 1024 * 1024)) != hash_value(expected, "client source"):
                raise Error("Client source differs from this home's initialized version.")
        for name in ("bin", "config", "data", "keyring-test", "identities"):
            path = self.home / name
            if os.path.lexists(path):
                core.private_directory(path)
        assert_no_node_keys(self.home)

    def check_node(self):
        self.check_files()
        return status_height(self.descriptor)

    def record_metadata(self):
        return {"schema": SCHEMA, "chain_id": self.chain, "descriptor_sha256": self.manifest["descriptor_sha256"],
                "genesis_sha256": self.manifest["genesis_sha256"], "participant_address": self.accounts[ACTOR]}

    def validate_record(self, value):
        super().validate_record(value)
        if value.get("actor", ACTOR) != ACTOR or value.get("address", self.accounts[ACTOR]) != self.accounts[ACTOR]:
            raise Error("Saved record belongs to another participant.")

    def actor(self, actor):
        if actor != ACTOR:
            raise Error("This home signs only as its own participant.")
        self.check_node()
        address = self.cli("keys", "show", ACTOR, "-a", "--keyring-backend", "test")
        if address != self.accounts[ACTOR]:
            raise Error("Local signing key differs from the saved participant identity.")
        return address

    def send(self, action, actor, arguments, context=None):
        if actor not in (ACTOR, "user", "challenger"):
            raise Error("No other participant's signer may be selected.")
        return super().send(action, ACTOR, arguments, context)

    def broadcast(self, path, attempt, signed_path):
        self.validate_record(attempt)
        self.actor(ACTOR)
        return super().broadcast(path, attempt, signed_path)

    def review(self, args, reveal=False):
        self.records("reviews")  # Also validate a directly selected saved tuple before use.
        args.actor = ACTOR
        return super().review(args, reveal)

    def onboard(self, account_type):
        if account_type not in ("human", "agent"):
            raise Error("Choose human or agent.")
        for _, attempt in self.records("attempts"):
            if attempt["action"] == "onboard":
                if attempt.get("context", {}).get("account_type") != account_type:
                    raise Error("The saved identity's account type cannot be replaced.")
                if attempt["status"] == "committed":
                    return core.public_receipt(attempt)
        return core.public_receipt(self.send("onboard", ACTOR, ["zerone_auth", "onboard", account_type], {"account_type": account_type}))

    def fund(self):
        address = self.actor(ACTOR)
        # A timeout never triggers an automatic second funding request.
        response = json_value(request(self.descriptor["faucet_url"], {"address": address}, allow_pending=True))
        return {"chain_id": CHAIN, "address": address, "faucet_response": response,
                "notice": "Faucet response only; committed transfer/balance must be checked separately."}

    def watch(self, claim_id):
        value = self.history(claim_id)
        path = self.directory / "watched" / (digest(claim_id.encode()) + ".json")
        if not path.exists():
            if len(self.records("watched")) >= core.MAX_RECORDS:
                raise Error("Watched claim inventory is full; nothing was truncated.")
            core.save_json(path, {**self.record_metadata(), "claim_id": claim_id}, fresh=True)
        return value

    def snapshot(self):
        value = super().snapshot()
        # All added reads use the same actual height as the core snapshot.
        height = core.integer(value["height"], "snapshot height", 1)
        existing = {row["claim_id"] for row in value["claims"]}
        watched = [row["claim_id"] for _, row in self.records("watched")]
        if len(existing | set(watched)) > core.MAX_RECORDS:
            raise Error("Combined watched claim inventory exceeds the bound.")
        value["claims"].extend({"claim_id": claim, "history": self.history(claim, height)} for claim in sorted(set(watched) - existing))
        value["claims"].sort(key=lambda row: row["claim_id"])
        value.update(schema=SCHEMA, local_only=False, loopback_test=self.manifest["allow_loopback_test"], notice=NOTICE,
                     participant_address=self.accounts[ACTOR], descriptor_sha256=self.manifest["descriptor_sha256"])
        if len(canonical(value)) > core.MAX_JSON:
            raise Error("Combined snapshot exceeds 8 MiB; nothing was truncated.")
        return value


def parser():
    result = argparse.ArgumentParser(description=__doc__)
    sub = result.add_subparsers(dest="command", required=True)
    init = sub.add_parser("init")
    for name in ("home", "descriptor", "descriptor-sha256", "binary", "binary-sha256"):
        init.add_argument("--" + name, required=True)
    init.add_argument("--allow-loopback-test", action="store_true", help="Explicit isolated test descriptor only; persist this distinction in the fresh home.")
    for name in ("onboard", "fund", "submit", "commit", "reveal", "challenge", "history", "watch", "snapshot", "retry"):
        command = sub.add_parser(name)
        command.add_argument("--home", required=True)
        command.add_argument("--wait", type=int, default=30)
        if name == "onboard":
            command.add_argument("--type", choices=("human", "agent"), required=True)
        if name in ("submit", "challenge"):
            command.add_argument("--content", required=True)
        if name == "submit":
            command.add_argument("--reasoning", required=True)
            command.add_argument("--domain", default="physics")
            command.add_argument("--category", default="empirical")
            command.add_argument("--method", default="")
        if name in ("commit", "reveal", "history", "watch"):
            command.add_argument("--claim", required=True)
        if name == "commit":
            command.add_argument("--vote", choices=("accept", "reject", "malformed"), required=True)
            command.add_argument("--reason", required=True)
            command.add_argument("--scope", required=True)
            command.add_argument("--method", default="")
            command.add_argument("--confidence", type=int, default=800000)
        if name == "reveal":
            command.add_argument("--wait-for-phase", action="store_true", help="Poll up to 7200 seconds; retry this read if the configured block window takes longer.")
        if name == "challenge":
            command.add_argument("--fact", required=True)
            command.add_argument("--reason", required=True)
            command.add_argument("--stake", default="11000000")
        if name in ("commit", "challenge"):
            command.add_argument("--evidence", action="append", default=[])
        if name == "retry":
            command.add_argument("--tx", required=True)
    return result


def main():
    os.umask(0o077)
    args = parser().parse_args()
    try:
        if args.command == "init":
            output = initialize(args)
        else:
            if not 1 <= args.wait <= 120:
                raise Error("--wait must be 1–120 seconds.")
            workflow = SharedWorkflow(args.home, args.wait)
            if args.command == "snapshot":
                output = workflow.snapshot()
            else:
                if args.command == "reveal" and args.wait_for_phase:
                    deadline = time.monotonic() + 7200
                    while workflow.round_for_claim(args.claim).get("phase") in ("VERIFICATION_PHASE_COMMIT", 1, "1") and time.monotonic() < deadline:
                        time.sleep(2)
                with workflow.locked():
                    if args.command == "onboard":
                        output = workflow.onboard(args.type)
                    elif args.command == "fund":
                        output = workflow.fund()
                    elif args.command == "submit":
                        output = workflow.submit(args)
                    elif args.command in ("commit", "reveal"):
                        output = workflow.review(args, args.command == "reveal")
                    elif args.command == "challenge":
                        output = workflow.challenge(args)
                    elif args.command == "retry":
                        output = workflow.retry(args.tx)
                    elif args.command == "watch":
                        output = workflow.watch(args.claim)
                    else:
                        workflow.reconcile()
                        output = workflow.history(args.claim)
        print(json.dumps(output, sort_keys=True, ensure_ascii=False, allow_nan=False))
        return 0
    except (Error, core.local.LocalNodeError, OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as error:
        print(f"shared-claims: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
