#!/usr/bin/env python3
"""A persistent, local-only Zerone sandbox. Python 3.11+, Linux or macOS.

Creates fresh test keys and valueless balances. Never joins a shared network,
accepts imported keys/genesis, resets a home, or signals an existing process.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import socket
import stat
import subprocess
import sys
import time
import tomllib
import urllib.error
import urllib.request

SCHEMA = "zerone-local-node-v1"
MARKER = ".zerone-local-node.json"
CHAIN_PATTERN = re.compile(r"zerone-local-[a-z0-9][a-z0-9-]{0,39}\Z")
CONTROL_FILES = ("config/genesis.json", "config/config.toml", "config/app.toml",
                 "config/client.toml")
IDENTITY_FILES = ("config/node_key.json", "config/priv_validator_key.json")
KNOWLEDGE_ACCOUNTS = ("reviewer1", "reviewer2", "reviewer3", "challenger")


def configure_knowledge_profile(genesis: dict, fast: bool = False) -> None:
    """Leave time for manual review; short windows are an explicit test option."""
    knowledge = genesis["app_state"]["knowledge"]
    for marker in ("record_integrity_enabled", "review_neutrality_enabled", "claim_records_enabled"):
        if knowledge.get(marker) is not True:
            raise LocalNodeError(f"Knowledge workflow requires native {marker}.")
    window = "60" if fast else "300"
    knowledge["params"].update({"commit_phase_blocks": window, "reveal_phase_blocks": window,
                                "aggregation_phase_blocks": "5"})


class LocalNodeError(Exception):
    pass


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def regular(path: Path) -> None:
    if path.is_symlink() or not path.is_file():
        raise LocalNodeError(f"Expected a regular, non-symlink file: {path}")


def home_path(value: str) -> Path:
    path = Path(value).expanduser()
    if not path.is_absolute() or ".." in path.parts:
        raise LocalNodeError("--home must be an explicit absolute path without '..'.")
    if path.is_symlink():
        raise LocalNodeError("A symlink cannot be used as a sandbox home.")
    if not path.parent.is_dir():
        raise LocalNodeError("The sandbox home's parent directory must already exist.")
    # Resolve platform aliases such as macOS /tmp, without following the home.
    path = path.parent.resolve() / path.name
    if path == Path.home() or path.name in (".zeroned", ".zerone", "zerone-1", "zerone-2"):
        raise LocalNodeError("Choose a dedicated local sandbox home, not a default or live node home.")
    if path.exists() and (not path.is_dir() or path.stat().st_uid != os.getuid()):
        raise LocalNodeError("The sandbox home must be a directory owned by the current user.")
    return path


def port(value: str) -> int:
    try:
        result = int(value)
    except ValueError as error:
        raise argparse.ArgumentTypeError("port must be an integer") from error
    if not 1024 <= result <= 65535:
        raise argparse.ArgumentTypeError("port must be between 1024 and 65535")
    return result


def check_ports(rpc: int, p2p: int) -> None:
    if rpc == p2p:
        raise LocalNodeError("RPC and P2P need different ports.")
    sockets = []
    try:
        for number in (rpc, p2p):
            sock = socket.socket()
            sockets.append(sock)
            # Like the node's TCP listener, permit restart while closed client
            # connections remain in TIME_WAIT; an active listener still fails.
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            sock.bind(("127.0.0.1", number))
    except OSError as error:
        raise LocalNodeError("A requested loopback port is occupied; choose different init ports. No process was stopped.") from error
    finally:
        for sock in sockets:
            sock.close()


@contextmanager
def locked(home: Path):
    try:
        descriptor = os.open(home / ".local-node.lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, "r+") as lock:
            if not stat.S_ISREG(os.fstat(lock.fileno()).st_mode):
                raise LocalNodeError("Sandbox lock is not a regular file.")
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError as error:
                raise LocalNodeError("This sandbox is already being initialized or run.") from error
            yield
    except OSError as error:
        raise LocalNodeError("Cannot acquire the sandbox's local lock.") from error


def cli(binary: Path, home: Path, *args: str, secret: bool = False) -> str:
    result = subprocess.run([str(binary), *args, "--home", str(home)],
                            text=True, capture_output=True, timeout=60, env=node_environment())
    if result.returncode:
        # Key creation can print seed phrases even on a failed invocation.
        detail = "Key operation failed; output withheld." if secret else result.stderr[-1500:]
        raise LocalNodeError(f"zeroned command failed: {detail.strip()}")
    return result.stdout.strip()


def node_environment() -> dict:
    # SDK Viper reads ZERONED_* above config files. An operator's shell settings
    # must not replace the freshly generated sandbox's local configuration.
    return {key: value for key, value in os.environ.items() if not key.upper().startswith("ZERONED_")}


def configure_toml(path: Path, changes: dict) -> None:
    regular(path)
    original = path.read_text()
    tomllib.loads(original)  # Reject duplicate sections and malformed input.
    parts = re.split(r"(?m)^(\[[^\]\n]+\]\s*(?:#.*)?\n)", original)
    sections = {"": parts[0]}
    order, headers = [""], {}
    for i in range(1, len(parts), 2):
        name = parts[i].split("]", 1)[0][1:]
        sections[name], headers[name] = parts[i + 1], parts[i]
        order.append(name)
    for section, fields in changes.items():
        if section not in sections:
            raise LocalNodeError(f"Generated config lacks section {section!r}.")
        for key, value in fields.items():
            expression = r"(?m)^\s*" + re.escape(key) + r"\s*=.*$"
            sections[section], count = re.subn(expression, lambda _: key + " = " + json.dumps(value), sections[section])
            if count != 1:
                raise LocalNodeError(f"Generated config lacks exactly one {section}.{key}.")
    result = "".join(headers.get(name, "") + sections[name] for name in order)
    parsed = tomllib.loads(result)
    for section, fields in changes.items():
        container = parsed[section] if section else parsed
        if any(container[key] != value for key, value in fields.items()):
            raise LocalNodeError("Generated config did not retain the local settings.")
    path.write_text(result)
    path.chmod(0o600)


def public_info(home: Path, manifest: dict) -> dict:
    return {"local_only": True, "chain_id": manifest["chain_id"], "home": str(home),
            "rpc": f"http://127.0.0.1:{manifest['rpc_port']}",
            "binary": str(home / "bin/zeroned"),
            "validator_address": manifest["validator_address"],
            "user_address": manifest["user_address"],
            **({"knowledge_profile": manifest["knowledge_profile"], "accounts": manifest["accounts"]}
               if manifest.get("knowledge_profile") else {})}


def initialize(args) -> None:
    home = home_path(args.home)
    if getattr(args, "fast_review", False) and not getattr(args, "knowledge_profile", False):
        raise LocalNodeError("--fast-review requires --knowledge-profile on a fresh sandbox.")
    if home.exists():
        raise LocalNodeError("Init requires a new home that does not exist. Existing files were not changed; use start for an initialized sandbox.")
    if not CHAIN_PATTERN.fullmatch(args.chain_id):
        raise LocalNodeError("Local chain IDs must start with 'zerone-local-' and contain only lowercase letters, digits and hyphens.")
    check_ports(args.rpc_port, args.p2p_port)
    source = Path(args.binary).expanduser().absolute()
    regular(source)
    if not os.access(source, os.X_OK):
        raise LocalNodeError("--binary is not executable.")
    version = subprocess.run([str(source), "version", "--long"], text=True,
                             capture_output=True, timeout=30)
    if version.returncode or not re.search(r"(?m)^cosmos_sdk_version:\s*v0\.53\.", version.stdout):
        raise LocalNodeError("Use the source-built SDK 0.53 Zerone binary for this local sandbox, not a legacy network executable.")
    home.mkdir(mode=0o700)
    home.chmod(0o700)
    with locked(home):
        (home / "bin").mkdir(mode=0o700)
        binary = home / "bin/zeroned"
        shutil.copyfile(source, binary)
        binary.chmod(0o700)
        binary_hash = sha256(binary)
        if binary_hash != sha256(source):
            raise LocalNodeError("Binary changed while being copied; incomplete home retained.")
        auxiliary = source.parent / "darwin-acl-check"
        auxiliary_hash = None
        if auxiliary.exists():
            regular(auxiliary)
            shutil.copyfile(auxiliary, home / "bin/darwin-acl-check")
            # The identity store validates this executable's exact mode before
            # creating onboarding identities on macOS.
            (home / "bin/darwin-acl-check").chmod(0o555)
            auxiliary_hash = sha256(home / "bin/darwin-acl-check")
        cli(binary, home, "init", "local-validator", "--chain-id", args.chain_id, "--default-denom", "uzrn")
        addresses = {}
        accounts = [("validator", "2000000000uzrn"), ("user", "1000000000uzrn")]
        if getattr(args, "knowledge_profile", False):
            accounts.extend((name, "1000000000uzrn") for name in KNOWLEDGE_ACCOUNTS)
        for name, balance in accounts:
            cli(binary, home, "keys", "add", name, "--keyring-backend", "test", secret=True)
            addresses[name] = cli(binary, home, "keys", "show", name, "-a", "--keyring-backend", "test")
            if not re.fullmatch(r"zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}", addresses[name]):
                raise LocalNodeError("Generated test address is not a canonical Zerone address.")
            cli(binary, home, "add-genesis-account", addresses[name], balance)
        if getattr(args, "knowledge_profile", False):
            genesis_path = home / "config/genesis.json"
            genesis = json.loads(genesis_path.read_text())
            configure_knowledge_profile(genesis, getattr(args, "fast_review", False))
            genesis_path.write_text(json.dumps(genesis, indent=2) + "\n")
        cli(binary, home, "genesis", "gentx", "validator", "1000000000uzrn", "--chain-id", args.chain_id,
            "--keyring-backend", "test", "--commission-rate", "0.1", "--commission-max-rate", "0.2",
            "--commission-max-change-rate", "0.01")
        cli(binary, home, "genesis", "collect-gentxs")
        cli(binary, home, "genesis", "validate")
        genesis = json.loads((home / "config/genesis.json").read_text())
        if genesis["chain_id"] != args.chain_id or any(
                genesis["app_state"][name].get("accounting_safety_enabled") is not True
                for name in ("zerone_staking", "zerone_gov")):
            raise LocalNodeError("Generated genesis is not the current native local accounting profile.")
        configure_toml(home / "config/config.toml", {
            "": {"priv_validator_laddr": ""},
            "rpc": {"laddr": f"tcp://127.0.0.1:{args.rpc_port}", "unsafe": False,
                    "pprof_laddr": "", "cors_allowed_origins": []},
            "p2p": {"laddr": f"tcp://127.0.0.1:{args.p2p_port}", "external_address": "",
                    "persistent_peers": "", "seeds": "", "pex": False, "seed_mode": False,
                    "max_num_inbound_peers": 0, "max_num_outbound_peers": 0},
            "statesync": {"enable": False},
            "consensus": {"timeout_propose": "500ms", "timeout_commit": "1s", "create_empty_blocks": True},
            "instrumentation": {"prometheus": False}})
        configure_toml(home / "config/app.toml", {
            "": {"minimum-gas-prices": "1uzrn", "pruning": "nothing"},
            "api": {"enable": False}, "grpc": {"enable": False}, "grpc-web": {"enable": False},
            "telemetry": {"enabled": False}, "state-sync": {"snapshot-interval": 0, "snapshot-keep-recent": 0}})
        configure_toml(home / "config/client.toml", {
            "": {"chain-id": args.chain_id, "keyring-backend": "test", "node": f"tcp://127.0.0.1:{args.rpc_port}"}})
        node_id = cli(binary, home, "comet", "show-node-id")
        if not re.fullmatch(r"[0-9a-f]{40}", node_id):
            raise LocalNodeError("Generated node identity is invalid.")
        manifest = {"schema": SCHEMA, "local_only": True, "chain_id": args.chain_id,
                    "owner_uid": os.getuid(), "binary_sha256": binary_hash,
                    "source_binary": str(source), "version_output_sha256": hashlib.sha256(version.stdout.encode()).hexdigest(),
                    "darwin_acl_helper_sha256": auxiliary_hash, "node_id": node_id,
                    "rpc_port": args.rpc_port, "p2p_port": args.p2p_port,
                    "validator_address": addresses["validator"], "user_address": addresses["user"],
                    "control_sha256": {name: sha256(home / name) for name in CONTROL_FILES},
                    "identity_sha256": {name: sha256(home / name) for name in IDENTITY_FILES},
                    "notice": "Fresh local test keys and valueless tokens only. No shared-network peers or registration."}
        if getattr(args, "knowledge_profile", False):
            manifest.update({"knowledge_profile": "claims-v1", "accounts": addresses})
        (home / MARKER).write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")
        (home / MARKER).chmod(0o600)
    print(json.dumps({"initialized": True, **public_info(home, manifest)}, sort_keys=True), flush=True)


def load_manifest(home: Path) -> dict:
    regular(home / MARKER)
    if (home.stat().st_mode & 0o077) or home.stat().st_uid != os.getuid():
        raise LocalNodeError("Sandbox home must remain private (mode 0700) and owned by you.")
    raw = (home / MARKER).read_bytes()
    if len(raw) > 16384:
        raise LocalNodeError("Sandbox manifest is oversized.")
    manifest = json.loads(raw)
    if (manifest.get("schema") != SCHEMA or manifest.get("local_only") is not True
            or manifest.get("owner_uid") != os.getuid()
            or not CHAIN_PATTERN.fullmatch(manifest.get("chain_id", ""))):
        raise LocalNodeError("This is not an initialized local sandbox home.")
    for directory in (home / "bin", home / "config", home / "data", home / "keyring-test"):
        if directory.is_symlink() or not directory.is_dir():
            raise LocalNodeError("Sandbox control directories must not be replaced or symlinked.")
    for name in ("bin/zeroned", *CONTROL_FILES, "config/node_key.json", "config/priv_validator_key.json",
                 "data/priv_validator_state.json"):
        regular(home / name)
    if sha256(home / "bin/zeroned") != manifest["binary_sha256"]:
        raise LocalNodeError("Retained binary changed. This helper does not upgrade an existing chain.")
    if set(manifest["control_sha256"]) != set(CONTROL_FILES) or any(
            sha256(home / name) != manifest["control_sha256"][name] for name in CONTROL_FILES):
        raise LocalNodeError("Genesis or local listener configuration changed; refusing to start or probe another network.")
    if set(manifest["identity_sha256"]) != set(IDENTITY_FILES) or any(
            sha256(home / name) != manifest["identity_sha256"][name] for name in IDENTITY_FILES):
        raise LocalNodeError("A generated node or validator identity changed; refusing to start or probe another identity.")
    if manifest.get("darwin_acl_helper_sha256"):
        regular(home / "bin/darwin-acl-check")
        if sha256(home / "bin/darwin-acl-check") != manifest["darwin_acl_helper_sha256"]:
            raise LocalNodeError("Retained Darwin identity helper changed.")
    for name in ("rpc_port", "p2p_port"):
        if type(manifest[name]) is not int or not 1024 <= manifest[name] <= 65535:
            raise LocalNodeError("Invalid local port in manifest.")
    return manifest


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise LocalNodeError("Local RPC attempted an HTTP redirect; refusing to follow it.")


def rpc_status(manifest: dict) -> dict:
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    with opener.open(f"http://127.0.0.1:{manifest['rpc_port']}/status", timeout=2) as response:
        raw = response.read(1024 * 1024 + 1)
    if len(raw) > 1024 * 1024:
        raise LocalNodeError("Local RPC returned an oversized status response.")
    value = json.loads(raw)["result"]
    if value["node_info"]["network"] != manifest["chain_id"] or value["node_info"]["id"] != manifest["node_id"]:
        raise LocalNodeError("The loopback port belongs to a different node or chain.")
    return value


def wait_ready(manifest: dict, seconds: float, process=None, stopping=lambda: False) -> dict:
    deadline, first = time.monotonic() + seconds, None
    while time.monotonic() < deadline:
        if stopping():
            raise LocalNodeError("Stopped before readiness; state was preserved.")
        if process is not None and process.poll() is not None:
            raise LocalNodeError("Local node exited before readiness. See the retained node.log.")
        try:
            value = rpc_status(manifest)
            height = int(value["sync_info"]["latest_block_height"])
            if first is None:
                first = height
            if height >= 2 and height > first and value["sync_info"]["catching_up"] is False:
                return {"ready": True, "advancing": True, "height": height, "node_id": manifest["node_id"]}
        except (OSError, urllib.error.URLError, KeyError, ValueError):
            pass
        time.sleep(0.15)
    raise LocalNodeError("Local node did not show advancing blocks before timeout; it may be stopped. See the retained node.log.")


def start(args) -> None:
    home = home_path(args.home)
    manifest = load_manifest(home)
    with locked(home):
        check_ports(manifest["rpc_port"], manifest["p2p_port"])
        log_path = home / "node.log"
        if log_path.exists() or log_path.is_symlink():
            regular(log_path)
        stopped = False
        def stop_request(_number, _frame):
            nonlocal stopped
            stopped = True
        previous = {number: signal.signal(number, stop_request) for number in (signal.SIGINT, signal.SIGTERM)}
        process = None
        try:
            descriptor = os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW, 0o600)
            with os.fdopen(descriptor, "ab") as log:
                if not stat.S_ISREG(os.fstat(log.fileno()).st_mode):
                    raise LocalNodeError("Sandbox log is not a regular file.")
                process = subprocess.Popen([str(home / "bin/zeroned"), "start", "--home", str(home),
                    "--minimum-gas-prices", "1uzrn", "--rpc.laddr", f"tcp://127.0.0.1:{manifest['rpc_port']}",
                    "--rpc.unsafe=false", "--rpc.pprof_laddr", "", "--p2p.laddr", f"tcp://127.0.0.1:{manifest['p2p_port']}",
                    "--p2p.external-address", "", "--p2p.persistent_peers", "", "--p2p.seeds", "", "--p2p.pex=false",
                    "--api.enable=false", "--grpc.enable=false", "--grpc-web.enable=false", "--log_no_color"],
                    stdout=log, stderr=subprocess.STDOUT, start_new_session=True, env=node_environment())
                ready = wait_ready(manifest, 45, process, lambda: stopped)
                print(json.dumps({**public_info(home, manifest), **ready,
                                  "stop": "Press Ctrl-C; use the same start command to resume."}, sort_keys=True), flush=True)
                while not stopped:
                    if process.poll() is not None:
                        raise LocalNodeError("Local node exited; its state and node.log were preserved.")
                    time.sleep(0.2)
        finally:
            try:
                if process is not None and process.poll() is None:
                    process.send_signal(signal.SIGINT)
                    try:
                        process.wait(timeout=20)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait(timeout=5)
                        raise LocalNodeError("Owned node needed a forced stop; state and logs remain for diagnosis.")
            finally:
                for number, handler in previous.items():
                    signal.signal(number, handler)
        print(json.dumps({"stopped": True, "state_preserved": True, "home": str(home)}, sort_keys=True), flush=True)


def main() -> int:
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    init = sub.add_parser("init", help="Create a fresh local validator and funded test user.")
    init.add_argument("--home", required=True)
    init.add_argument("--binary", required=True)
    init.add_argument("--chain-id", default="zerone-local-1")
    init.add_argument("--rpc-port", type=port, default=47657)
    init.add_argument("--p2p-port", type=port, default=47656)
    init.add_argument("--knowledge-profile", action="store_true",
                      help="Fund three reviewers and a challenger; use 300/300/5-block review windows in this fresh local genesis.")
    init.add_argument("--fast-review", action="store_true",
                      help="With --knowledge-profile, use 60-block commit/reveal windows for automated exercises.")
    for name in ("start", "status"):
        command = sub.add_parser(name, help="Run in the foreground." if name == "start" else "Check local identity and advancing blocks.")
        command.add_argument("--home", required=True)
    args = parser.parse_args()
    try:
        if args.command == "init":
            initialize(args)
        elif args.command == "start":
            start(args)
        else:
            home = home_path(args.home)
            manifest = load_manifest(home)
            print(json.dumps({**public_info(home, manifest), **wait_ready(manifest, 10)}, sort_keys=True))
    except (LocalNodeError, OSError, ValueError, KeyError, subprocess.TimeoutExpired) as error:
        print(f"local-node: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
