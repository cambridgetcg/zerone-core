#!/usr/bin/env python3
"""Persistent zerone-dev-1 runtime and fresh full-node join helper. Python 3.11+."""
from __future__ import annotations

import argparse
import base64
from contextlib import contextmanager, nullcontext
import fcntl
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import secrets
import signal
import stat
import subprocess
import sys
import time
import tomllib
import urllib.parse
import urllib.request

CHAIN = "zerone-dev-1"
SCHEMA = "zerone-dev-runtime-v1"
MARKER = ".zerone-dev-runtime.json"
CONTROL = ("config/genesis.json", "config/config.toml", "config/app.toml", "config/client.toml")
IDENTITY = ("config/priv_validator_key.json", "config/node_key.json")
MAX_JSON = 16 * 1024 * 1024
VALIDATOR_ALLOCATION = 2_000_000_000_000
VALIDATOR_BOND = 1_000_000_000_000
FAUCET_ALLOCATION = 1_000_000_000_000
UPGRADE_NAME = "knowledge-fund-settlement-v1"
UPGRADE_DIRECTORY = "fund-settlement-upgrade"


class RuntimeError(Exception):
    pass


def read(path, limit=MAX_JSON, private=False):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
            raise RuntimeError(f"Expected a regular single-link file: {path}")
        if private and (info.st_uid != os.getuid() or info.st_mode & 0o077):
            raise RuntimeError(f"Expected a private owned file: {path}")
        value = stream.read(limit + 1)
    if len(value) > limit:
        raise RuntimeError("Input exceeds the runtime read bound.")
    return value


def digest(path):
    return hashlib.sha256(read(path, 256 * 1024 * 1024)).hexdigest()


def parse(raw):
    def pairs(items):
        value = {}
        for key, item in items:
            if key in value:
                raise RuntimeError("Duplicate JSON field.")
            value[key] = item
        return value
    return json.loads(raw, object_pairs_hook=pairs)


def write_new(path, raw):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())
    fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def json_bytes(value):
    return (json.dumps(value, indent=2, sort_keys=True) + "\n").encode()


def exact_fields(value, fields, label):
    if not isinstance(value, dict) or set(value) != set(fields):
        raise RuntimeError(f"Invalid {label} fields.")


def hash_pin(value, length=64):
    if not isinstance(value, str) or not re.fullmatch(rf"[0-9a-f]{{{length}}}", value) or set(value) == {"0"}:
        raise RuntimeError("Expected an exact nonzero hash pin.")
    return value


def upgrade_platform():
    name = (platform.system(), platform.machine().lower())
    if name in (("Darwin", "arm64"), ("Darwin", "aarch64")):
        return "darwin-arm64"
    if name in (("Linux", "x86_64"), ("Linux", "amd64")):
        return "linux-amd64"
    raise RuntimeError("Upgrade packages support Linux AMD64 and Darwin ARM64 only.")


def validate_upgrade_packet(packet):
    exact_fields(packet, ("schema", "chain_id", "genesis_sha256", "predecessor_descriptor_sha256", "plan", "predecessor", "target"), "upgrade packet")
    if packet["schema"] != "zerone-development-upgrade/v1" or packet["chain_id"] != CHAIN:
        raise RuntimeError("Upgrade packet names another schema or chain.")
    hash_pin(packet["genesis_sha256"])
    hash_pin(packet["predecessor_descriptor_sha256"])
    exact_fields(packet["plan"], ("name", "height", "info"), "upgrade plan")
    if packet["plan"]["name"] != UPGRADE_NAME or packet["plan"]["info"] != "":
        raise RuntimeError("Only the named fund-settlement upgrade with empty info is supported.")
    height = integer(packet["plan"]["height"], "upgrade height", 2)
    if type(packet["plan"]["height"]) is not str or height > (1 << 63) - 1:
        raise RuntimeError("Upgrade height must be a canonical positive int64 string.")
    for name, version in (("predecessor", 10), ("target", 11)):
        release = packet[name]
        exact_fields(release, ("knowledge_version", "source_commit", "binaries"), name)
        if type(release["knowledge_version"]) is not int or release["knowledge_version"] != version:
            raise RuntimeError("Unsupported upgrade knowledge versions.")
        hash_pin(release["source_commit"], 40)
        exact_fields(release["binaries"], ("linux-amd64", "darwin-arm64"), "platform binary pins")
        for value in release["binaries"].values():
            hash_pin(value)
    if packet["target"]["source_commit"] == packet["predecessor"]["source_commit"]:
        raise RuntimeError("An upgrade must identify a distinct target source.")
    return packet


def home_path(value):
    path = Path(value).expanduser()
    if not path.is_absolute() or ".." in path.parts or not path.parent.is_dir() or path.is_symlink():
        raise RuntimeError("Home must be an absolute, non-symlink path with an existing parent.")
    path = path.parent.resolve() / path.name
    if path in (Path.home(), Path.home() / ".zeroned", Path.home() / ".zerone") or path.name in ("zerone-1", "zerone-2"):
        raise RuntimeError("Choose a dedicated development home; legacy/default homes are refused.")
    if path.exists():
        info = path.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise RuntimeError("Development home must be a private owned directory.")
    return path


@contextmanager
def lock(home):
    # The lock is outside the home so it covers first initialization too.
    path = home.parent / ("." + home.name + ".dev-runtime.lock")
    fd = os.open(path, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600)
    with os.fdopen(fd, "r+") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise RuntimeError("Unsafe runtime lock.")
        try:
            fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as error:
            raise RuntimeError("Another runtime owns this home; no process was started.") from error
        yield


def environment():
    # Never let SDK Viper replace the recorded network/listener configuration.
    return {key: value for key, value in os.environ.items() if not key.upper().startswith("ZERONED_")}


def cli(binary, home, *args, secret=False):
    result = subprocess.run([str(binary), *map(str, args), "--home", str(home)], capture_output=True,
                            text=True, timeout=60, env=environment())
    if result.returncode:
        detail = "Key operation failed; output withheld." if secret else result.stderr[-1500:]
        raise RuntimeError(f"zeroned {args[0]} failed: {detail.strip()}")
    return result.stdout.strip()


def configure(path, changes):
    original = read(path).decode()
    tomllib.loads(original)
    parts = re.split(r"(?m)^(\[[^\]\n]+\]\s*(?:#.*)?\n)", original)
    sections, order, headers = {"": parts[0]}, [""], {}
    for index in range(1, len(parts), 2):
        name = parts[index].split("]", 1)[0][1:]
        sections[name], headers[name] = parts[index + 1], parts[index]
        order.append(name)
    for section, fields in changes.items():
        if section not in sections:
            raise RuntimeError(f"Generated TOML lacks section {section}.")
        for key, value in fields.items():
            sections[section], count = re.subn(r"(?m)^\s*" + re.escape(key) + r"\s*=.*$",
                lambda _: key + " = " + json.dumps(value), sections[section])
            if count != 1:
                raise RuntimeError(f"Generated TOML lacks exactly one {section}.{key}.")
    result = "".join(headers.get(name, "") + sections[name] for name in order)
    parsed = tomllib.loads(result)
    for section, fields in changes.items():
        target = parsed[section] if section else parsed
        if any(target[key] != value for key, value in fields.items()):
            raise RuntimeError("Generated TOML did not preserve requested settings.")
    path.write_text(result)
    path.chmod(0o600)


def integer(value, label, minimum=0):
    if isinstance(value, bool) or not re.fullmatch(r"0|[1-9][0-9]*", str(value)) or int(value) < minimum:
        raise RuntimeError(f"Invalid {label}.")
    return int(value)


def check_genesis(genesis):
    if genesis.get("chain_id") != CHAIN or integer(genesis.get("initial_height", "1"), "initial height") not in (0, 1):
        raise RuntimeError("Only a fresh zerone-dev-1 genesis is supported.")
    state = genesis["app_state"]
    for module in ("zerone_staking", "zerone_gov"):
        if state[module].get("accounting_safety_enabled") is not True:
            raise RuntimeError("Genesis lacks native accounting safety.")
    for field in ("record_integrity_enabled", "review_neutrality_enabled", "claim_records_enabled", "fund_settlement_enabled"):
        if state["knowledge"].get(field) is not True:
            raise RuntimeError("Genesis lacks the current native knowledge flags.")


def binary_identity(binary, source_commit):
    if not re.fullmatch(r"[0-9a-f]{40}", source_commit or ""):
        raise RuntimeError("An exact source commit is required.")
    binary = Path(binary).expanduser().absolute()
    if not os.access(binary, os.X_OK):
        raise RuntimeError("The pinned binary is not executable.")
    sha = digest(binary)
    result = subprocess.run([str(binary), "version", "--long"], capture_output=True, text=True,
                            timeout=30, env=environment())
    if result.returncode or not re.search(r"(?m)^cosmos_sdk_version:\s*v0\.53\.8\s*$", result.stdout) or not re.search(r"(?m)^commit:\s*" + source_commit + r"\s*$", result.stdout):
        raise RuntimeError("Binary does not identify the expected source commit and SDK0.53.8.")
    version = re.search(r"(?m)^version:\s*(\S+)\s*$", result.stdout)
    if not version:
        raise RuntimeError("Binary version metadata is absent.")
    return binary, sha, version.group(1)


def peer(value):
    if not re.fullmatch(r"[0-9a-f]{40}@[a-zA-Z0-9][a-zA-Z0-9.-]*:[1-9][0-9]{0,4}", value or "") or int(value.rsplit(":", 1)[1]) > 65535:
        raise RuntimeError("Peer must be node-id@host:port with a40-hex node ID.")
    return value


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise RuntimeError("RPC redirect refused.")


def rpc(origin, path):
    parsed = urllib.parse.urlsplit(origin)
    if parsed.scheme not in ("http", "https") or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment or parsed.path not in ("", "/"):
        raise RuntimeError("RPC origin must be an explicit HTTP(S) origin without credentials or path.")
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    request = urllib.request.Request(origin.rstrip("/") + path, headers={"User-Agent": "zerone-dev-runtime/1.0"})
    with opener.open(request, timeout=5) as response:
        raw = response.read(MAX_JSON + 1)
    if len(raw) > MAX_JSON:
        raise RuntimeError("RPC response exceeds the bound.")
    value = parse(raw)
    if "error" in value or not isinstance(value.get("result"), dict):
        raise RuntimeError("RPC returned no successful result.")
    return value["result"]


def initialize(args, home):
    if home.exists():
        raise RuntimeError("Initialization requires a new home; partial or populated homes are preserved and refused.")
    joining = args.command == "join"
    if not joining and not args.local_test and home != Path("/data/.zeroned"):
        raise RuntimeError("The shared validator initializes only /data/.zeroned; custom fixture homes require --local-test.")
    ports = (args.rpc_port, args.p2p_port, args.gateway_port)
    if any(not 1024 <= value <= 65535 for value in ports) or len(set(ports)) != 3:
        raise RuntimeError("Listeners require distinct ports between1024 and65535.")
    if not args.local_test and ports != (26657, 26656, 8080):
        raise RuntimeError("Nonstandard ports require the explicit local test profile.")
    if args.review_window_blocks is not None and not args.local_test:
        raise RuntimeError("Accelerated review windows require --local-test on a fresh home.")
    window = args.review_window_blocks or 3600
    if not 5 <= window <= 3600:
        raise RuntimeError("Test review windows must be5–3600 blocks.")
    binary, binary_sha, version = binary_identity(args.binary, args.source_commit)
    genesis_raw = None
    if joining:
        genesis_raw = read(Path(args.genesis))
        if not re.fullmatch(r"[0-9a-f]{64}", args.genesis_sha256) or hashlib.sha256(genesis_raw).hexdigest() != args.genesis_sha256:
            raise RuntimeError("Public genesis does not match the pinned digest.")
        check_genesis(parse(genesis_raw))
        peer(args.peer)
        reference = rpc(args.reference_rpc, "/status")
        if reference["node_info"]["network"] != CHAIN or reference["node_info"]["id"] != args.peer.split("@", 1)[0]:
            raise RuntimeError("Reference RPC does not match the pinned development peer and chain.")
    home.mkdir(mode=0o700)
    home.chmod(0o700)
    cli(binary, home, "init", "zerone-dev-validator" if not joining else "zerone-dev-full-node", "--chain-id", CHAIN, "--default-denom", "uzrn")
    accounts = {}
    if not joining:
        for actor, amount in (("validator", f"{VALIDATOR_ALLOCATION}uzrn"), ("faucet", f"{FAUCET_ALLOCATION}uzrn")):
            cli(binary, home, "keys", "add", actor, "--keyring-backend", "test", secret=True)
            address = cli(binary, home, "keys", "show", actor, "-a", "--keyring-backend", "test")
            if not re.fullmatch(r"zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}", address):
                raise RuntimeError("Generated account address is not canonical.")
            accounts[actor] = address
            cli(binary, home, "add-genesis-account", address, amount)
        genesis_path = home / "config/genesis.json"
        genesis = parse(read(genesis_path))
        check_genesis(genesis)
        genesis["app_state"]["knowledge"]["params"].update({"commit_phase_blocks": str(window), "reveal_phase_blocks": str(window), "aggregation_phase_blocks": "5"})
        genesis["app_state"]["staking"]["params"]["max_validators"] = 1
        genesis_path.write_bytes(json_bytes(genesis))
        genesis_path.chmod(0o600)
        cli(binary, home, "genesis", "gentx", "validator", f"{VALIDATOR_BOND}uzrn", "--chain-id", CHAIN,
            "--keyring-backend", "test", "--commission-rate", "0.1", "--commission-max-rate", "0.2", "--commission-max-change-rate", "0.01")
        cli(binary, home, "genesis", "collect-gentxs")
    else:
        (home / "config/genesis.json").write_bytes(genesis_raw)
        (home / "config/genesis.json").chmod(0o600)
    cli(binary, home, "genesis", "validate")
    genesis = parse(read(home / "config/genesis.json"))
    check_genesis(genesis)
    node_id = cli(binary, home, "comet", "show-node-id")
    consensus = parse(cli(binary, home, "comet", "show-validator"))
    public_key = consensus.get("key", "")
    try:
        public_raw = base64.b64decode(public_key, validate=True)
    except ValueError as error:
        raise RuntimeError("Invalid generated consensus public key.") from error
    if not re.fullmatch(r"[0-9a-f]{40}", node_id) or len(public_raw) != 32:
        raise RuntimeError("Invalid generated consensus/P2P identity.")
    genesis_keys = [message.get("pubkey", {}).get("key") for tx in genesis["app_state"]["genutil"]["gen_txs"] for message in tx["body"]["messages"]]
    if (public_key in genesis_keys) == joining:
        raise RuntimeError("Consensus identity does not match the requested validator/full-node role.")
    advertised = f"{node_id}@127.0.0.1:{args.p2p_port}" if args.local_test else ("" if joining else f"{node_id}@zerone-dev-1.fly.dev:26656")
    configure(home / "config/config.toml", {
        "": {"priv_validator_laddr": ""},
        "rpc": {"laddr": f"tcp://127.0.0.1:{args.rpc_port}", "unsafe": False, "pprof_laddr": "", "cors_allowed_origins": [],
                "max_open_connections": 100, "max_request_batch_size": 1, "max_body_bytes": 262144},
        "p2p": {"laddr": f"tcp://{'127.0.0.1' if args.local_test else '0.0.0.0'}:{args.p2p_port}",
                "external_address": advertised.split("@", 1)[-1] if advertised else "", "persistent_peers": args.peer if joining else "",
                "seeds": "", "pex": not args.local_test, "seed_mode": False, "allow_duplicate_ip": args.local_test,
                "addr_book_strict": not args.local_test, "max_num_inbound_peers": 40, "max_num_outbound_peers": 10},
        "statesync": {"enable": False}, "consensus": {"timeout_commit": "1s", "create_empty_blocks": True},
        "tx_index": {"indexer": "kv"}, "instrumentation": {"prometheus": False}})
    configure(home / "config/app.toml", {"": {"minimum-gas-prices": "1uzrn", "pruning": "default", "query-gas-limit": "5000000"},
        "api": {"enable": False}, "grpc": {"enable": False}, "grpc-web": {"enable": False}, "telemetry": {"enabled": False},
        "state-sync": {"snapshot-interval": 1000, "snapshot-keep-recent": 2}})
    configure(home / "config/client.toml", {"": {"chain-id": CHAIN, "keyring-backend": "test", "node": f"tcp://127.0.0.1:{args.rpc_port}"}})
    for name in (*CONTROL, *IDENTITY, "data/priv_validator_state.json"):
        (home / name).chmod(0o600)
    for name in ("config", "data"):
        (home / name).chmod(0o700)
    genesis_sha = digest(home / "config/genesis.json")
    if not joining:
        (home / "dev-faucet").mkdir(mode=0o700)
        write_new(home / "dev-faucet/state.json", json_bytes({"schema": "zerone-dev-faucet/v1", "chain_id": CHAIN,
            "genesis_sha256": genesis_sha, "ip_salt": secrets.token_hex(32), "grants": {}}))
    manifest = {"schema": SCHEMA, "chain_id": CHAIN, "role": "full-node" if joining else "validator", "owner_uid": os.getuid(),
        "source_commit": args.source_commit, "binary_version": version, "binary_sha256": binary_sha,
        "genesis_sha256": genesis_sha, "node_id": node_id, "consensus_public_key": public_key,
        "consensus_address": hashlib.sha256(public_raw).hexdigest()[:40].upper(), "accounts": accounts,
        "rpc_port": args.rpc_port, "p2p_port": args.p2p_port, "gateway_port": args.gateway_port,
        "advertised_peer": advertised, "reference_rpc": args.reference_rpc if joining else "",
        "reference_peer": args.peer if joining else "", "local_test": args.local_test,
        "review_window_blocks": integer(genesis["app_state"]["knowledge"]["params"]["commit_phase_blocks"], "review window", 1),
        "control_sha256": {name: digest(home / name) for name in CONTROL},
        "identity_sha256": {name: digest(home / name) for name in IDENTITY}}
    write_new(home / MARKER, json_bytes(manifest))
    return manifest


def load(home, binary):
    manifest = parse(read(home / MARKER, 32768, private=True))
    if manifest.get("schema") != SCHEMA or manifest.get("chain_id") != CHAIN or manifest.get("role") not in ("validator", "full-node") or manifest.get("owner_uid") != os.getuid():
        raise RuntimeError("Not an initialized development runtime home.")
    for directory in ("config", "data"):
        path = home / directory
        info = path.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise RuntimeError("A control directory changed identity or permissions.")
    if digest(binary) != manifest["binary_sha256"]:
        raise RuntimeError("Pinned binary changed; this entrypoint does not upgrade existing state.")
    for name, expected in (("control_sha256", CONTROL), ("identity_sha256", IDENTITY)):
        if set(manifest[name]) != set(expected):
            raise RuntimeError("Runtime manifest omits a control or identity file.")
        for relative in expected:
            read(home / relative, private=True)
            if digest(home / relative) != manifest[name][relative]:
                raise RuntimeError("Genesis, configuration or identity changed; refusing restart.")
    state = parse(read(home / "data/priv_validator_state.json", 65536, private=True))
    integer(state.get("height"), "signing-state height")
    integer(state.get("round"), "signing-state round")
    if integer(state.get("step"), "signing-state step") > 3:
        raise RuntimeError("Invalid signing state.")
    if digest(home / "config/genesis.json") != manifest["genesis_sha256"]:
        raise RuntimeError("Genesis digest does not match the runtime manifest.")
    if manifest["role"] == "validator":
        info = (home / "dev-faucet").lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise RuntimeError("Faucet directory is missing or unsafe.")
        read(home / "dev-faucet/state.json", private=True)
        info = (home / "keyring-test").lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise RuntimeError("Runtime test keyring is missing or unsafe.")
    return manifest


def stage_upgrade(args, home):
    """Explicit preparation only: no node, signing, database writer or metadata replacement."""
    with lock(home):
        raw = read(Path(args.upgrade_packet), 32768)
        if hashlib.sha256(raw).hexdigest() != hash_pin(args.upgrade_packet_sha256):
            raise RuntimeError("Upgrade packet differs from its external pin.")
        packet = validate_upgrade_packet(parse(raw))
        manifest = load(home, args.predecessor_binary)
        selected = upgrade_platform()
        if manifest["genesis_sha256"] != packet["genesis_sha256"] or manifest["source_commit"] != packet["predecessor"]["source_commit"] or manifest["binary_sha256"] != packet["predecessor"]["binaries"][selected]:
            raise RuntimeError("Upgrade predecessor does not match this initialized home.")
        if manifest["role"] == "validator" and digest(home / "public/network.json") != packet["predecessor_descriptor_sha256"]:
            raise RuntimeError("Upgrade predecessor descriptor does not match the preserved publication.")
        binaries = {}
        for name, path in (("predecessor", args.predecessor_binary), ("target", args.binary)):
            binaries[name] = read(Path(path), 256 * 1024 * 1024)
            if hashlib.sha256(binaries[name]).hexdigest() != packet[name]["binaries"][selected]:
                raise RuntimeError(f"Wrong {name} binary for the selected upgrade platform.")
        directory = home / UPGRADE_DIRECTORY
        directory.mkdir(mode=0o700)  # Existing/partial preparation is preserved and refused.
        write_new(directory / "packet.json", raw)
        for name, raw_binary in binaries.items():
            path = directory / (name + "-zeroned")
            write_new(path, raw_binary)
            path.chmod(0o500)
            binary_identity(path, packet[name]["source_commit"])
        write_new(directory / "stage.json", json_bytes({"schema": "zerone-development-upgrade-stage/v1",
            "packet_sha256": args.upgrade_packet_sha256, "platform": selected,
            "original_manifest_sha256": digest(home / MARKER)}))
        return {"status": "staged-not-applied", "chain_id": CHAIN, "plan": packet["plan"], "packet_sha256": args.upgrade_packet_sha256}


def load_upgrade(home, external_pin):
    directory = home / UPGRADE_DIRECTORY
    info = directory.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise RuntimeError("Unsafe upgrade preparation directory.")
    raw = read(directory / "packet.json", 32768, private=True)
    if hashlib.sha256(raw).hexdigest() != hash_pin(external_pin):
        raise RuntimeError("Staged upgrade packet differs from its external pin.")
    packet = validate_upgrade_packet(parse(raw))
    stage = parse(read(directory / "stage.json", 32768, private=True))
    exact_fields(stage, ("schema", "packet_sha256", "platform", "original_manifest_sha256"), "upgrade stage")
    if stage["schema"] != "zerone-development-upgrade-stage/v1" or stage["packet_sha256"] != external_pin or stage["platform"] != upgrade_platform() or stage["original_manifest_sha256"] != digest(home / MARKER):
        raise RuntimeError("Upgrade preparation no longer matches this home or platform.")
    for name in ("predecessor", "target"):
        if digest(directory / (name + "-zeroned")) != packet[name]["binaries"][stage["platform"]]:
            raise RuntimeError("A staged upgrade binary changed.")
    manifest = load(home, directory / "predecessor-zeroned")
    if manifest["genesis_sha256"] != packet["genesis_sha256"] or manifest["source_commit"] != packet["predecessor"]["source_commit"]:
        raise RuntimeError("Staged upgrade predecessor changed.")
    if manifest["role"] == "validator" and digest(home / "public/network.json") != packet["predecessor_descriptor_sha256"]:
        raise RuntimeError("Preserved predecessor descriptor changed.")
    return packet, manifest


def matching_upgrade_plan(value, packet):
    if not isinstance(value, dict) or not {"name", "height"} <= set(value) or set(value) - {"name", "height", "info", "time", "upgraded_client_state"}:
        raise RuntimeError("Actual upgrade plan has an unsupported shape.")
    if value["name"] != UPGRADE_NAME or integer(value["height"], "actual upgrade height", 2) != int(packet["plan"]["height"]) or value.get("info", "") != "" or value.get("time") not in (None, "", "0001-01-01T00:00:00Z") or value.get("upgraded_client_state") is not None:
        raise RuntimeError("Actual upgrade plan differs from the pinned name, height or empty metadata.")


def disk_upgrade_plan(home, packet):
    path = home / "data/upgrade-info.json"
    if not os.path.lexists(path):
        return False
    matching_upgrade_plan(parse(read(path, 32768, private=True)), packet)
    return True


def upgrade_query(binary, home, manifest, *args):
    return parse(cli(binary, home, "query", "upgrade", *args, "--node", f"http://127.0.0.1:{manifest['rpc_port']}", "--output", "json"))


def verify_applied_upgrade(binary, home, manifest, packet):
    versions = upgrade_query(binary, home, manifest, "module-versions")
    rows = versions.get("module_versions")
    if not isinstance(rows, list) or not rows:
        raise RuntimeError("Applied upgrade has no module version map.")
    mapped = {}
    for row in rows:
        if not isinstance(row, dict) or not isinstance(row.get("name"), str) or not row["name"] or row["name"] in mapped:
            raise RuntimeError("Applied upgrade has duplicate or malformed module versions.")
        mapped[row["name"]] = integer(row.get("version", "0"), "module version", 0)
    applied = upgrade_query(binary, home, manifest, "applied", UPGRADE_NAME)
    if mapped.get("knowledge") != 11 or integer(applied.get("height"), "applied upgrade height", 1) != int(packet["plan"]["height"]):
        raise RuntimeError("Target has not applied the exact knowledge 10-to-11 upgrade.")
    return {"module_versions": mapped, "applied_height": str(applied["height"])}


def upgraded_manifest(manifest, packet):
    return {**manifest, "source_commit": packet["target"]["source_commit"],
            "binary_sha256": packet["target"]["binaries"][upgrade_platform()]}


def status(manifest, crosscheck=True):
    origin = f"http://127.0.0.1:{manifest['rpc_port']}"
    value = rpc(origin, "/status")
    if value["node_info"]["network"] != CHAIN or value["node_info"]["id"] != manifest["node_id"]:
        raise RuntimeError("Local listener belongs to another node or chain.")
    height = integer(value["sync_info"]["latest_block_height"], "block height")
    power = integer(value["validator_info"]["voting_power"], "voting power")
    if value["validator_info"]["address"] != manifest["consensus_address"]:
        raise RuntimeError("Node reports a different consensus identity.")
    if manifest["role"] == "full-node" and power != 0:
        raise RuntimeError("This join helper is for zero-power full nodes.")
    ready = height > 0 and value["sync_info"]["catching_up"] is False
    result = {"chain_id": CHAIN, "role": manifest["role"], "node_id": manifest["node_id"], "height": height,
              "ready": ready, "voting_power": power, "genesis_sha256": manifest["genesis_sha256"], "local_test": manifest["local_test"]}
    if ready and crosscheck and manifest["role"] == "full-node":
        reference = rpc(manifest["reference_rpc"], "/status")
        if reference["node_info"]["network"] != CHAIN or reference["node_info"]["id"] != manifest["reference_peer"].split("@", 1)[0]:
            raise RuntimeError("Reference node identity changed.")
        common = min(height, integer(reference["sync_info"]["latest_block_height"], "reference height", 1))
        ours = rpc(origin, f"/block?height={common}")
        theirs = rpc(manifest["reference_rpc"], f"/block?height={common}")
        for block in (ours, theirs):
            header = block["block"]["header"]
            if integer(header["height"], "returned block height", 1) != common or header["chain_id"] != CHAIN or not re.fullmatch(r"[0-9a-fA-F]{64}", block["block_id"]["hash"]) or not re.fullmatch(r"[0-9a-fA-F]{64}", header["app_hash"]):
                raise RuntimeError("Reference comparison lacks the exact requested block context.")
        if ours["block_id"]["hash"] != theirs["block_id"]["hash"] or ours["block"]["header"]["app_hash"] != theirs["block"]["header"]["app_hash"]:
            raise RuntimeError("Full node and reference disagree at a common committed height.")
        result.update(reference_match=True, compared_height=common)
    return result


def run(args, home):
    with (nullcontext() if getattr(args, "upgrade_lock_held", False) else lock(home)):
        packet = getattr(args, "staged_upgrade", None)
        target_phase = getattr(args, "upgrade_target", False)
        if packet is None:
            manifest = load(home, args.binary) if home.exists() else initialize(args, home)
        else:
            checked, original = load_upgrade(home, args.upgrade_packet_sha256)
            if checked != packet:
                raise RuntimeError("Staged upgrade changed while starting.")
            manifest = upgraded_manifest(original, packet) if target_phase else original
        stopping = False
        def stop_request(_number, _frame):
            nonlocal stopping
            stopping = True
        previous = {number: signal.signal(number, stop_request) for number in (signal.SIGINT, signal.SIGTERM)}
        children, streams = [], []
        try:
            def spawn(name, command):
                if not manifest["local_test"]:
                    # Fly collects these streams; never grow application-volume logs indefinitely.
                    child = subprocess.Popen(command, start_new_session=True, env=environment())
                    children.append(child)
                    return child
                fd = os.open(home / (name + ".log"), os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600)
                stream = os.fdopen(fd, "ab")
                streams.append(stream)
                info = os.fstat(fd)
                if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
                    raise RuntimeError("Runtime log is not a regular file.")
                child = subprocess.Popen(command, stdout=stream, stderr=subprocess.STDOUT, start_new_session=True, env=environment())
                children.append(child)
                return child
            node = spawn("node", [str(args.binary), "start", "--home", str(home)])
            def reached_upgrade_boundary():
                if packet is None or target_phase or not disk_upgrade_plan(home, packet):
                    return False
                if node.poll() is None:
                    observed = rpc(f"http://127.0.0.1:{manifest['rpc_port']}", "/abci_info")
                    if integer(observed["response"].get("last_block_height"), "halted application height", 1) != int(packet["plan"]["height"]) - 1:
                        raise RuntimeError("Predecessor halt is not at the pinned H-1.")
                    planned = upgrade_query(args.binary, home, manifest, "plan")
                    matching_upgrade_plan(planned.get("plan"), packet)
                # If the predecessor exited, the new application's startup still
                # checks the committed H-1 and both actual plans before signing.
                return True
            deadline, first, ready = time.monotonic() + 120, None, None
            best_height = 0
            while not stopping and time.monotonic() < deadline:
                if reached_upgrade_boundary():
                    return "upgrade-boundary"
                if node.poll() is not None:
                    raise RuntimeError("Node exited before readiness; inspect the retained node.log.")
                try:
                    observed = status(manifest, crosscheck=False)
                    if manifest["role"] == "full-node" and observed["height"] > best_height:
                        best_height = observed["height"]
                        deadline = time.monotonic() + 120
                    if first is None:
                        first = observed["height"]
                    if observed["ready"] and observed["height"] > first:
                        ready = observed
                        break
                except (OSError, ValueError, KeyError):
                    pass
                time.sleep(0.3)
            if stopping:
                return
            if ready is None:
                if manifest["role"] == "full-node":
                    raise RuntimeError(f"Full node did not become ready and made no new block-height progress for 120 seconds (best height {best_height}).")
                raise RuntimeError("Node did not demonstrate advancing blocks before timeout.")
            if target_phase:
                applied = verify_applied_upgrade(args.binary, home, manifest, packet)
                receipt = {"schema": "zerone-development-upgrade-observation/v1", "chain_id": CHAIN,
                    "packet_sha256": args.upgrade_packet_sha256, "source_commit": manifest["source_commit"],
                    "binary_sha256": manifest["binary_sha256"], "observed_height": str(ready["height"]), **applied}
                receipt_path = home / UPGRADE_DIRECTORY / "first-applied-observation.json"
                if os.path.lexists(receipt_path):
                    prior = parse(read(receipt_path, 32768, private=True))
                    for key in ("schema", "chain_id", "packet_sha256", "source_commit", "binary_sha256", "module_versions", "applied_height"):
                        if prior.get(key) != receipt[key]:
                            raise RuntimeError("Retained upgrade observation conflicts with actual applied state.")
                else:
                    write_new(receipt_path, json_bytes(receipt))
            if manifest["role"] == "validator":
                gateway = Path(args.gateway).absolute()
                read(gateway, 1024 * 1024)
                spawn("gateway", [sys.executable, "-I", "-B", str(gateway), "--home", str(home), "--binary", str(args.binary),
                    "--port", str(manifest["gateway_port"]), "--rpc", f"http://127.0.0.1:{manifest['rpc_port']}",
                    "--bind", "127.0.0.1" if manifest["local_test"] else "0.0.0.0",
                    *(["--upgrade-packet-sha256", args.upgrade_packet_sha256] if target_phase else [])])
            print(json.dumps({**ready, "source_commit": manifest["source_commit"], "binary_sha256": manifest["binary_sha256"],
                              "advertised_peer": manifest["advertised_peer"], "accounts": manifest["accounts"]}), flush=True)
            while not stopping:
                if reached_upgrade_boundary():
                    return "upgrade-boundary"
                if any(child.poll() is not None for child in children):
                    raise RuntimeError("An owned node/gateway process exited; stopping the other process and preserving the home.")
                time.sleep(0.3)
        finally:
            try:
                forced = False
                for child in reversed(children):
                    if child.poll() is None:
                        child.send_signal(signal.SIGTERM)
                for child in reversed(children):
                    if child.poll() is None:
                        try:
                            child.wait(timeout=30)
                        except subprocess.TimeoutExpired:
                            child.kill()
                            child.wait(timeout=5)
                            forced = True
                if forced:
                    raise RuntimeError("Owned process required forced termination; state remains for diagnosis.")
            finally:
                for stream in streams:
                    stream.close()
                for number, handler in previous.items():
                    signal.signal(number, handler)


def run_upgrade(args, home):
    """Run only an explicitly staged transition, holding one lock across both processes."""
    with lock(home):
        packet, _ = load_upgrade(home, args.upgrade_packet_sha256)
        args.upgrade_lock_held = True
        args.staged_upgrade = packet
        args.upgrade_target = disk_upgrade_plan(home, packet)
        while True:
            name = "target" if args.upgrade_target else "predecessor"
            args.binary = home / UPGRADE_DIRECTORY / (name + "-zeroned")
            result = run(args, home)
            if result != "upgrade-boundary":
                return
            if args.upgrade_target or not disk_upgrade_plan(home, packet):
                raise RuntimeError("No consistent actual upgrade boundary for the target process.")
            args.upgrade_target = True


def arguments():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    for name in ("init", "run", "status", "join", "stage-upgrade", "run-upgrade", "upgrade-status"):
        cmd = sub.add_parser(name)
        cmd.add_argument("--home", default="/data/.zeroned")
        cmd.add_argument("--binary", default="/usr/local/bin/zeroned", type=Path)
        cmd.add_argument("--source-commit", default="")
        cmd.add_argument("--build-info", default="/opt/zerone-dev/build.json", type=Path)
        cmd.add_argument("--gateway", default=str(Path(__file__).with_name("gateway.py")))
        cmd.add_argument("--local-test", action="store_true")
        cmd.add_argument("--review-window-blocks", type=int)
        cmd.add_argument("--rpc-port", type=int, default=26657)
        cmd.add_argument("--p2p-port", type=int, default=26656)
        cmd.add_argument("--gateway-port", type=int, default=8080)
        if name == "join":
            cmd.add_argument("--genesis", required=True)
            cmd.add_argument("--genesis-sha256", required=True)
            cmd.add_argument("--peer", required=True)
            cmd.add_argument("--reference-rpc", required=True)
        if name in ("stage-upgrade", "run-upgrade", "upgrade-status"):
            cmd.add_argument("--upgrade-packet-sha256", required=True)
        if name == "stage-upgrade":
            cmd.add_argument("--upgrade-packet", required=True)
            cmd.add_argument("--predecessor-binary", required=True, type=Path)
    return parser.parse_args()


def main():
    os.umask(0o077)
    args = arguments()
    try:
        home = home_path(args.home)
        if args.command not in ("stage-upgrade", "run-upgrade", "upgrade-status") and not args.source_commit and args.build_info.exists():
            info = parse(read(args.build_info, 32768))
            args.source_commit = info["source_commit"]
            if digest(args.binary) != info["binary_sha256"]:
                raise RuntimeError("Image binary differs from its build receipt.")
        if args.command == "run":
            run(args, home)
        elif args.command == "stage-upgrade":
            print(json.dumps(stage_upgrade(args, home)), flush=True)
        elif args.command == "run-upgrade":
            run_upgrade(args, home)
        elif args.command == "upgrade-status":
            packet, original = load_upgrade(home, args.upgrade_packet_sha256)
            manifest = upgraded_manifest(original, packet)
            binary = home / UPGRADE_DIRECTORY / "target-zeroned"
            observed = status(manifest)
            applied = verify_applied_upgrade(binary, home, manifest, packet)
            print(json.dumps({**observed, **applied, "packet_sha256": args.upgrade_packet_sha256}), flush=True)
        elif args.command == "status":
            print(json.dumps(status(load(home, args.binary))), flush=True)
        else:
            with lock(home):
                manifest = initialize(args, home)
            print(json.dumps({key: manifest[key] for key in ("chain_id", "role", "genesis_sha256", "node_id", "accounts", "advertised_peer", "source_commit", "binary_sha256", "local_test")}), flush=True)
        return 0
    except (RuntimeError, OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as error:
        print(f"zerone-dev-runtime: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
