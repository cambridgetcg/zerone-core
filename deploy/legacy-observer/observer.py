#!/usr/bin/env python3
"""Run a signed zerone-1 observer package on Linux amd64 with local Docker.

Python 3.11+, gpgv and Docker are required. All node identities are freshly
generated locally. This helper never imports keys, submits transactions,
admits a validator, resets a home, or upgrades another running node.
"""
from __future__ import annotations

import argparse
import base64
from contextlib import contextmanager
import datetime as dt
import fcntl
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shutil
import signal
import socket
import stat
import subprocess
import sys
import time
import tomllib
import urllib.request
import urllib.error
import uuid

FINGERPRINT = "09327B031F8FF2C2EE49B18F2234027FC5B68C19"
IMAGE = "docker.io/library/debian@sha256:5ae3c39ebd15e229dcedd5cee596b2497182493d41ff162e824ba13fc1b2b867"
SCHEMA = "zerone-1-observer-home/v1"
MARKER = ".zerone-observer.json"
CONTROLS = ("config/genesis.json", "config/config.toml", "config/app.toml",
            "config/client.toml", "config/node_key.json", "config/priv_validator_key.json")
DEFAULT_PORT = 27657


class Refusal(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise Refusal(message)


def sha(path):
    checksum = hashlib.sha256()
    with os.fdopen(os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK), "rb") as stream:
        before = os.fstat(stream.fileno())
        require(stat.S_ISREG(before.st_mode) and before.st_size <= 64 * 1024 * 1024,
                "Invalid or oversized home control")
        total = 0
        while chunk := stream.read(1024 * 1024):
            total += len(chunk)
            require(total <= 64 * 1024 * 1024, "Home control grew past bound")
            checksum.update(chunk)
        after = os.fstat(stream.fileno())
        require(total == before.st_size and (before.st_size, before.st_mtime_ns, before.st_ctime_ns) ==
                (after.st_size, after.st_mtime_ns, after.st_ctime_ns), "Home control changed while reading")
    return checksum.hexdigest()


def json_object(raw):
    def pairs(items):
        result = {}
        for key, value in items:
            require(key not in result, "Duplicate JSON key")
            result[key] = value
        return result
    def reject_constant(_):
        raise Refusal("Non-finite JSON value")
    value = json.loads(raw, object_pairs_hook=pairs, parse_constant=reject_constant)
    require(type(value) is dict, "Expected a JSON object")
    return value


def read_json(path, limit=256 * 1024, with_hash=False):
    with os.fdopen(os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK), "rb") as stream:
        info = os.fstat(stream.fileno())
        require(stat.S_ISREG(info.st_mode) and 0 < info.st_size <= limit,
                "Expected a bounded regular JSON file: " + path.name)
        raw = stream.read(limit + 1)
        after = os.fstat(stream.fileno())
        require(len(raw) == info.st_size and len(raw) <= limit and
                (info.st_size, info.st_mtime_ns, info.st_ctime_ns) ==
                (after.st_size, after.st_mtime_ns, after.st_ctime_ns), "JSON input changed or exceeds bound")
    value = json_object(raw)
    return (value, hashlib.sha256(raw).hexdigest()) if with_hash else value


def write_json(path, value):
    require(not path.is_symlink(), "Refusing a symlink output")
    temporary = path.with_name(path.name + "." + uuid.uuid4().hex + ".tmp")
    with temporary.open("x", encoding="utf-8") as stream:
        os.chmod(temporary, 0o600)
        json.dump(value, stream, sort_keys=True, separators=(",", ":"))
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def home_path(value):
    path = Path(value).expanduser()
    require(path.is_absolute() and ".." not in path.parts and not path.is_symlink(),
            "Choose an absolute, dedicated home without '..' or a symlink")
    require(path.parent.is_dir(), "Home parent must already exist")
    path = path.parent.resolve() / path.name
    require(path != Path.home() and path.name not in (".zeroned", ".zerone", "zerone-1", "zerone-2"),
            "Choose a new dedicated observer home, not a default/live home")
    require("," not in str(path) and "\n" not in str(path), "Unsupported home path")
    return path


@contextmanager
def locked(home):
    require(home.is_dir() and not home.is_symlink(), "Observer home must be a directory")
    descriptor = os.open(home / ".observer.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, "r+") as stream:
        require(stat.S_ISREG(os.fstat(stream.fileno()).st_mode), "Invalid home lock")
        try:
            fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as error:
            raise Refusal("This observer home is already in use") from error
        yield


def command(argv, timeout=90, secret=False):
    result = subprocess.run(argv, stdin=subprocess.DEVNULL, capture_output=True,
                            text=True, timeout=timeout, check=False)
    if result.returncode:
        detail = "output withheld" if secret else result.stderr[-1500:].strip()
        raise Refusal("Command failed: " + Path(argv[0]).name + ": " + detail)
    return result.stdout.strip()


def docker():
    require(platform.system() == "Linux" and platform.machine() in ("x86_64", "amd64"),
            "This release is tested on Linux amd64 only")
    executable = shutil.which("docker")
    require(executable is not None, "Install Docker before using this package")
    # Explicit local socket: ambient contexts/DOCKER_HOST cannot choose a remote
    # daemon with different files or expose fresh identities to another host.
    return [executable, "--host", "unix:///var/run/docker.sock"]


def runtime_identity():
    return (os.getuid(), os.getgid()) if os.getuid() else (65532, 65532)


def bundle_access(bundle, uid, gid):
    """A bind mount bypasses host parents, but retains the mounted file modes."""
    def permission(path, needed):
        info = path.lstat()
        require(not stat.S_ISLNK(info.st_mode), "Symlink in observer package")
        shift = 6 if info.st_uid == uid else 3 if info.st_gid == gid else 0
        require((info.st_mode >> shift) & needed == needed,
                "Package is not readable/executable by its container UID:GID; use the verified release file modes")
    permission(bundle, 5)
    with os.scandir(bundle) as entries:
        for index, entry in enumerate(entries):
            require(index < 66, "Observer package contains too many entries")
            require(entry.is_file(follow_symlinks=False), "Observer package must contain regular files")
            permission(Path(entry.path), 5 if entry.name in {"zeroned", "verify-checkpoint"} else 4)


def container(bundle, *, home=None, uid=None, gid=None, network="none"):
    default_uid, default_gid = runtime_identity()
    uid = default_uid if uid is None else uid
    gid = default_gid if gid is None else gid
    bundle_access(bundle, uid, gid)
    argv = docker() + ["create", "--platform", "linux/amd64", "--restart", "no",
                      "--network", network, "--user", f"{uid}:{gid}",
                      "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
                      "--cpus", "2", "--memory", "6g", "--memory-swap", "6g", "--pids-limit", "256",
                      "--tmpfs", "/tmp:rw,nosuid,nodev,noexec,size=64m",
                      "--log-driver", "local", "--log-opt", "max-size=10m", "--log-opt", "max-file=3",
                      "--mount", f"type=bind,src={bundle},dst=/bundle,readonly"]
    if home is not None:
        argv += ["--workdir", "/observer", "--mount", f"type=bind,src={home},dst=/observer"]
    return argv


def owned_state(reference, owner, missing_ok=False):
    result = subprocess.run(docker() + ["inspect", "--format",
        '{{index .Config.Labels "ai.zerone.observer-home"}} {{.State.Running}} {{.State.ExitCode}} {{.State.OOMKilled}}', reference],
        stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=15, check=False)
    if result.returncode:
        require(missing_ok and ("No such object" in result.stderr or "No such container" in result.stderr),
                "Could not inspect owned observer container")
        return None
    fields = result.stdout.strip().split()
    require(len(fields) == 4 and fields[0] == owner and fields[1] in {"true", "false"} and
            fields[2].isdigit() and fields[3] in {"true", "false"},
            "Container ownership/state changed; refusing to signal or remove it")
    return {"running": fields[1] == "true", "exit_code": int(fields[2]), "oom_killed": fields[3] == "true"}


def stop_owned(reference, owner, missing_ok=False):
    state = owned_state(reference, owner, missing_ok)
    if state is None:
        return None
    if state["running"]:
        command(docker() + ["stop", "--signal", "SIGTERM", "--timeout", "60", reference], timeout=75)
    final = owned_state(reference, owner)
    require(not final["running"], "Owned observer container did not stop")
    # Keep failed containers stopped so their bounded Docker logs and inspect
    # metadata remain available. Logs may include fresh-key initialization
    # output, so never copy or print them automatically.
    final.update(container=reference, removed=False)
    if final["exit_code"] == 0 and not final["oom_killed"]:
        command(docker() + ["rm", reference])
        final["removed"] = True
    return final


def offline_container(bundle, arguments, *, home=None, uid=None, gid=None, secret=False):
    owner = uuid.uuid4().hex
    name = "zerone-observer-check-" + owner
    cid = None
    failure = None
    try:
        cid = command(container(bundle, home=home, uid=uid, gid=gid) + [
            "--name", name, "--label", "ai.zerone.observer-home=" + owner, IMAGE, *arguments])
        require(re.fullmatch(r"[0-9a-f]{64}", cid), "Unexpected offline container ID")
        output = command(docker() + ["start", "--attach", cid], secret=secret)
    except (Refusal, OSError, subprocess.SubprocessError) as error:
        failure = error
    finally:
        # Create cannot start a daemon-side workload. Even if its CLI times out,
        # the unique name permits safe cleanup without guessing another owner.
        final = stop_owned(cid if cid and re.fullmatch(r"[0-9a-f]{64}", cid) else name, owner, missing_ok=True)
    retained = "; stopped container retained: " + final["container"] if final and not final["removed"] else ""
    if failure is not None:
        # Do not stringify subprocess failures: they can contain captured init
        # output. A container reference is enough for deliberate local diagnosis.
        raise Refusal("Offline observer command failed; command output withheld" + retained) from failure
    require(final is not None and final["exit_code"] == 0 and not final["oom_killed"],
            "Offline observer check failed or was OOM-killed" + retained)
    return output


def verify_bundle(bundle, gpgv, purpose):
    require(bundle.is_dir() and not bundle.is_symlink(), "Bundle must be a regular directory")
    require("," not in str(bundle) and "\n" not in str(bundle), "Unsupported bundle path")
    result = json.loads(command([sys.executable, "-I", "-B", str(bundle / "verify-release.py"),
                                  "--bundle", str(bundle), "--expected-fingerprint", FINGERPRINT,
                                  "--gpgv", gpgv, "--purpose", purpose]))
    require(result.get("result") == "PASS" and result.get("purpose") == purpose and
            (purpose == "resume" or result.get("bootstrap_ready") is True), "Release integrity verification failed")
    manifest, manifest_hash = read_json(bundle / "OBSERVER-RELEASE.json", with_hash=True)
    require(manifest_hash == result.get("manifest_sha256"), "Release changed after integrity verification")
    checkpoint, checkpoint_hash = read_json(bundle / manifest["checkpoint"]["file"], with_hash=True)
    require(checkpoint_hash == manifest["checkpoint"]["sha256"], "Checkpoint differs from signed inventory")
    require(checkpoint.get("schema") == "zerone-1-observer-checkpoint/v1" and
            checkpoint.get("chain_id") == "zerone-1", "Wrong checkpoint scope")
    require(type(checkpoint.get("height")) is int and checkpoint["height"] > 0 and
            re.fullmatch(r"[0-9A-F]{64}", checkpoint.get("block_hash", "")), "Invalid checkpoint height/hash")
    require(checkpoint.get("trust_period_seconds") == 604800, "Unsupported trust period")
    header_time = dt.datetime.fromisoformat(checkpoint["header_time"].replace("Z", "+00:00"))
    expiry = dt.datetime.fromisoformat(manifest["expires_at"].replace("Z", "+00:00"))
    now = dt.datetime.now(dt.timezone.utc)
    require(header_time.tzinfo is not None and header_time <= now + dt.timedelta(seconds=30),
            "Invalid or future checkpoint time")
    require(expiry <= header_time + dt.timedelta(seconds=604800), "Release outlives checkpoint trust period")
    if purpose == "bootstrap":
        require(now < expiry, "Fresh bootstrap window expired; obtain a refreshed signed package")
    proof = ["/bundle/verify-checkpoint", "--checkpoint", "/bundle/" + manifest["checkpoint"]["file"]]
    if purpose == "resume":
        proof.append("--allow-expired")
    verification = json.loads(offline_container(bundle, proof))
    require(verification.get("result") == "PASS" and
            verification.get("checkpoint_sha256") == checkpoint_hash and
            verification.get("chain_id") == "zerone-1" and verification.get("height") == checkpoint["height"] and
            verification.get("block_hash") == checkpoint["block_hash"], "Checkpoint signature verification failed")
    return manifest, checkpoint, result


def configure(path, changes, uid, gid):
    original = path.read_text()
    tomllib.loads(original)
    parts = re.split(r"(?m)^(\[[^\]\n]+\]\s*(?:#.*)?\n)", original)
    sections, order, headers = {"": parts[0]}, [""], {}
    for i in range(1, len(parts), 2):
        name = parts[i].split("]", 1)[0][1:]
        sections[name], headers[name] = parts[i + 1], parts[i]
        order.append(name)
    for section, fields in changes.items():
        require(section in sections, "Generated config lacks section " + section)
        for key, value in fields.items():
            sections[section], count = re.subn(r"(?m)^\s*" + re.escape(key) + r"\s*=.*$",
                                               lambda _: key + " = " + json.dumps(value), sections[section])
            require(count == 1, "Generated config lacks unique setting " + section + "." + key)
    result = "".join(headers.get(name, "") + sections[name] for name in order)
    parsed = tomllib.loads(result)
    for section, fields in changes.items():
        for key, value in fields.items():
            require((parsed[section] if section else parsed)[key] == value, "Config update failed")
    path.write_text(result)
    path.chmod(0o600)
    if os.getuid() == 0:
        os.chown(path, uid, gid)


def settings(manifest, checkpoint):
    return {
        "config/config.toml": {
            "": {"proxy_app": "tcp://127.0.0.1:26658", "priv_validator_laddr": ""},
            "rpc": {"laddr": "tcp://0.0.0.0:26657", "unsafe": False, "pprof_laddr": "",
                    "cors_allowed_origins": []},
            "p2p": {"laddr": "tcp://0.0.0.0:26656", "external_address": "", "seeds": "",
                    "persistent_peers": manifest["network"]["peer"], "pex": False,
                    "addr_book_strict": True, "max_num_inbound_peers": 0, "max_num_outbound_peers": 1},
            "mempool": {"type": "nop", "broadcast": False},
            "consensus": {"create_empty_blocks": True},
            "statesync": {"enable": True, "rpc_servers": ",".join(manifest["network"]["rpc_servers"]),
                          "trust_height": checkpoint["height"], "trust_hash": checkpoint["block_hash"],
                          "trust_period": "168h0m0s", "chunk_fetchers": 2, "temp_dir": "/observer/state-sync-temp"},
            "tx_index": {"indexer": "null"}, "instrumentation": {"prometheus": False}},
        "config/app.toml": {
            "": {"minimum-gas-prices": "0.025uzrn", "pruning": "nothing"},
            "api": {"enable": False}, "grpc": {"enable": False}, "grpc-web": {"enable": False},
            "telemetry": {"enabled": False},
            "state-sync": {"snapshot-interval": 0, "snapshot-keep-recent": 0}}}


def init(args, bundle):
    home = home_path(args.home)
    require(not home.exists(), "Home already exists; nothing overwritten. Choose a fresh dedicated path")
    manifest, checkpoint, verified = verify_bundle(bundle, args.gpgv, "bootstrap")
    uid, gid = runtime_identity()
    home.mkdir(mode=0o700)
    if os.getuid() == 0:
        os.chown(home, uid, gid)
    with locked(home):
        offline_container(bundle, ["/bundle/zeroned", "init", "zerone-observer", "--chain-id", "zerone-1",
                                   "--home", "/observer"], home=home, uid=uid, gid=gid, secret=True)
        require(not any(path.is_symlink() for path in home.rglob("*")), "Unexpected symlink in generated home")
        shutil.copyfile(bundle / "genesis.json", home / "config/genesis.json")
        (home / "config/genesis.json").chmod(0o600)
        if os.getuid() == 0:
            os.chown(home / "config/genesis.json", uid, gid)
        for path, changes in settings(manifest, checkpoint).items():
            configure(home / path, changes, uid, gid)
        offline_container(bundle, ["/bundle/zeroned", "genesis", "validate", "/bundle/genesis.json",
                                   "--home", "/observer"], home=home, uid=uid, gid=gid)
        identity = read_json(home / "config/priv_validator_key.json")
        require(re.fullmatch(r"[0-9A-F]{40}", identity.get("address", "")) and
                identity["address"] != "93DBC1A1D9A07E67423B193395C9EB6F1E41B844",
                "Generated consensus identity must differ from the live validator")
        genesis = read_json(bundle / "genesis.json", 64 * 1024 * 1024)
        public_key = identity.get("pub_key", {}).get("value")
        try:
            decoded_public_key = base64.b64decode(public_key, validate=True)
        except (ValueError, TypeError) as error:
            raise Refusal("Invalid generated consensus public key") from error
        require(identity.get("pub_key", {}).get("type") == "tendermint/PubKeyEd25519" and
                len(decoded_public_key) == 32 and
                hashlib.sha256(decoded_public_key).hexdigest()[:40].upper() == identity["address"] and
                public_key != "LA944mpk44k8Jr9EN/uOzSSf8G+Ng2LiIvf9FJBkl80=",
                "Generated consensus public key/address must be fresh and consistent")
        genesis_keys = [row.get("pub_key", {}).get("value") for row in genesis.get("validators", [])]
        for tx in genesis.get("app_state", {}).get("genutil", {}).get("gen_txs", []):
            for msg in tx.get("body", {}).get("messages", []):
                if msg.get("@type") == "/cosmos.staking.v1beta1.MsgCreateValidator":
                    genesis_keys.append(msg.get("pubkey", {}).get("key"))
        require(type(public_key) is str and public_key not in genesis_keys,
                "Generated consensus public key must differ from genesis validators")
        node_id = offline_container(bundle, ["/bundle/zeroned", "comet", "show-node-id", "--home", "/observer"],
                                    home=home, uid=uid, gid=gid)
        require(re.fullmatch(r"[0-9a-f]{40}", node_id), "Invalid generated P2P identity")
        marker = {"schema": SCHEMA, "id": uuid.uuid4().hex, "home": str(home), "uid": uid, "gid": gid,
                  "operator_uid": os.getuid(), "port": args.port, "manifest_sha256": verified["manifest_sha256"],
                  "release_id": manifest["release_id"], "chain_id": "zerone-1",
                  "validator_address": identity["address"], "node_id": node_id,
                  "control_sha256": {name: sha(home / name) for name in CONTROLS}, "synced": None}
        signing_state(home)
        write_json(home / MARKER, marker)
    emit({"result": "INITIALIZED", "home": str(home), "chain_id": "zerone-1",
          "rpc": f"http://127.0.0.1:{args.port}", "validator_power": 0,
          "note": "Fresh local identities; no account created or validator admitted"})


def signing_state(home):
    require((home / "data").is_dir() and not (home / "data").is_symlink(), "Invalid observer data directory")
    state = read_json(home / "data/priv_validator_state.json")
    require(state.get("height") == "0" and type(state.get("round")) is int and state["round"] == 0 and
            type(state.get("step")) is int and state["step"] == 0 and
            not state.get("signature") and not state.get("signbytes"), "Observer signing state is not unused")


def validate_home(home):
    require(home.is_dir() and not home.is_symlink(), "Observer home must be a regular directory")
    marker = read_json(home / MARKER)
    require(marker.get("schema") == SCHEMA and marker.get("home") == str(home) and
            marker.get("operator_uid") == os.getuid() and marker.get("chain_id") == "zerone-1", "Wrong observer home")
    require(re.fullmatch(r"[0-9a-f]{32}", marker.get("id", "")), "Invalid home identity")
    require((marker.get("uid"), marker.get("gid")) == runtime_identity(), "Observer runtime UID:GID changed")
    require(re.fullmatch(r"[0-9a-f]{40}", marker.get("node_id", "")), "Invalid pinned P2P identity")
    require(type(marker.get("port")) is int and 1024 <= marker["port"] <= 65535, "Invalid home RPC port")
    require(set(marker.get("control_sha256", {})) == set(CONTROLS), "Missing home control pins")
    for name in CONTROLS:
        path = home / name
        require(not path.is_symlink() and path.is_file() and
                all(not parent.is_symlink() for parent in path.parents if parent != home.parent), "Invalid home control file")
        require(sha(path) == marker["control_sha256"][name], "Home control changed: " + name)
    require(read_json(home / "config/priv_validator_key.json")["address"] == marker["validator_address"], "Observer identity changed")
    signing_state(home)
    return marker


def rpc(port, method):
    require(method in {"status", "abci_info"}, "Only status and abci_info reads are exposed")
    require(type(port) is int and 1024 <= port <= 65535, "Invalid local RPC port")
    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, req, fp, code, msg, headers, newurl):
            raise Refusal("Local RPC redirect refused")
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    with opener.open(f"http://127.0.0.1:{port}/{method}", timeout=5) as response:
        raw = response.read(1024 * 1024 + 1)
    require(len(raw) <= 1024 * 1024, "RPC response exceeds bound")
    value = json_object(raw)
    require("error" not in value, "Local RPC query failed")
    require(type(value.get("result")) is dict, "Local RPC result is malformed")
    return value["result"]


def _observed(marker):
    status = rpc(marker["port"], "status")
    require(status["node_info"]["network"] == "zerone-1" and
            status["node_info"]["id"] == marker["node_id"], "Local RPC has wrong chain or P2P identity")
    validator = status["validator_info"]
    require(validator["address"] == marker["validator_address"] and validator["voting_power"] == "0",
            "Local RPC identity/power does not match the fresh zero-power observer")
    sync = status["sync_info"]
    require(type(sync["latest_block_height"]) is str and
            re.fullmatch(r"(?:0|[1-9][0-9]{0,17})", sync["latest_block_height"]), "Invalid observed height")
    height = int(sync["latest_block_height"])
    timestamp = dt.datetime.fromisoformat(sync["latest_block_time"].replace("Z", "+00:00"))
    require(timestamp.tzinfo is not None, "Observed block time lacks timezone")
    age = (dt.datetime.now(dt.timezone.utc) - timestamp).total_seconds()
    require(type(sync["catching_up"]) is bool, "Invalid syncing status")
    app = rpc(marker["port"], "abci_info")["response"]
    require(type(app["last_block_height"]) is str and
            re.fullmatch(r"(?:0|[1-9][0-9]{0,17})", app["last_block_height"]), "Invalid applied height")
    applied_height = int(app["last_block_height"])
    ready = height > 0 and applied_height >= height and sync["catching_up"] is False and -30 <= age <= 180
    if ready:
        require(re.fullmatch(r"[0-9A-F]{64}", sync["latest_block_hash"]) and
                re.fullmatch(r"[0-9A-F]{64}", sync["latest_app_hash"]), "Invalid observed block hashes")
    return {"height": height, "block_hash": sync["latest_block_hash"],
            "header_app_hash": sync["latest_app_hash"], "applied_height": applied_height,
            "abci_last_block_app_hash": app["last_block_app_hash"], "block_time": sync["latest_block_time"],
            "catching_up": sync["catching_up"], "ready": ready, "validator_power": 0}


def observed(marker):
    try:
        return _observed(marker)
    except (KeyError, TypeError, ValueError) as error:
        if isinstance(error, Refusal):
            raise
        raise Refusal("Malformed local RPC observation") from error


def emit(value):
    print(json.dumps(value, sort_keys=True, separators=(",", ":")), flush=True)


def start(args, bundle):
    home = home_path(args.home)
    # Refuse unrelated existing homes before creating even a lock file.
    validate_home(home)
    with locked(home):
        marker = validate_home(home)
        resume = marker.get("synced") is not None
        if resume:
            require(type(marker["synced"]) is dict and type(marker["synced"].get("height")) is int and
                    marker["synced"]["height"] > 0 and (home / "data/application.db").is_dir() and
                    not (home / "data/application.db").is_symlink() and
                    any((home / "data/application.db").iterdir()),
                    "Resume requires this home's successful prior bootstrap and retained application database")
        manifest, checkpoint, verified = verify_bundle(bundle, args.gpgv, "resume" if resume else "bootstrap")
        require(verified["manifest_sha256"] == marker["manifest_sha256"], "Package differs from the initialized home")
        require(manifest["release_id"] == marker["release_id"], "Release ID changed")
        # Check before start, and let Docker enforce the same exclusive binding.
        with socket.socket() as listener:
            listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            listener.bind(("127.0.0.1", marker["port"]))
        name = "zerone-observer-" + marker["id"][:12] + "-" + uuid.uuid4().hex[:8]
        argv = container(bundle, home=home, uid=marker["uid"], gid=marker["gid"], network="bridge") + [
            "--name", name, "--label", "ai.zerone.observer-home=" + marker["id"],
            "--publish", f"127.0.0.1:{marker['port']}:26657", IMAGE, "/bundle/zeroned", "start", "--home", "/observer"]
        cid = None
        stopped = False
        old_handlers = {}
        def interrupt(_signum, _frame):
            nonlocal stopped
            stopped = True
        for signum in (signal.SIGINT, signal.SIGTERM):
            old_handlers[signum] = signal.signal(signum, interrupt)
        emit({"result": "STARTING", "container": name, "rpc": f"http://127.0.0.1:{marker['port']}",
              "resume": resume, "bootstrap_window_current": verified["bootstrap_ready"]})
        first_ready, last_report, deadline = None, 0, time.monotonic() + args.sync_timeout
        try:
            cid = command(argv)
            require(re.fullmatch(r"[0-9a-f]{64}", cid), "Unexpected container ID")
            if not stopped:
                command(docker() + ["start", cid])
            while not stopped:
                require(owned_state(cid, marker["id"])["running"], "Observer exited")
                try:
                    status = observed(marker)
                except (OSError, ValueError, KeyError) as error:
                    # An identity mismatch is never treated as temporary sync.
                    if isinstance(error, Refusal):
                        raise
                    status = {"ready": False, "state": "waiting for local RPC"}
                if status["ready"]:
                    if first_ready is None:
                        first_ready = status["height"]
                    if status["height"] > first_ready:
                        signing_state(home)
                        marker["synced"] = {"height": status["height"], "block_hash": status["block_hash"],
                                            "observed_at": dt.datetime.now(dt.timezone.utc).isoformat()}
                        write_json(home / MARKER, marker)
                require(marker["synced"] is not None or time.monotonic() < deadline,
                        "Bootstrap did not follow fresh blocks before the timeout; home preserved for inspection")
                if time.monotonic() - last_report >= 30:
                    following = status["ready"] and first_ready is not None and status["height"] > first_ready
                    emit({"result": "FOLLOWING" if following else "SYNCING", **status})
                    last_report = time.monotonic()
                time.sleep(2)
        finally:
            try:
                final = stop_owned(cid if cid and re.fullmatch(r"[0-9a-f]{64}", cid) else name,
                                   marker["id"], missing_ok=True)
                emit({"result": "STOPPED", "home_preserved": str(home), "last_verified_sync": marker["synced"],
                      "container_exit": final})
                signing_state(home)
            finally:
                for signum, handler in old_handlers.items():
                    signal.signal(signum, handler)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("init", "start", "status"))
    parser.add_argument("--bundle", type=Path, default=Path(__file__).absolute().parent)
    parser.add_argument("--home", required=True)
    parser.add_argument("--gpgv", default=shutil.which("gpgv"))
    parser.add_argument("--port", type=int, default=DEFAULT_PORT, help="Loopback RPC port for init")
    parser.add_argument("--sync-timeout", type=int, default=3600, help="First bootstrap deadline in seconds")
    args = parser.parse_args()
    try:
        require(sys.version_info >= (3, 11), "Python 3.11+ required")
        require(args.gpgv and Path(args.gpgv).is_absolute(), "An absolute gpgv executable is required")
        require(1024 <= args.port <= 65535 and 60 <= args.sync_timeout <= 86400, "Invalid port or sync timeout")
        bundle = args.bundle.absolute()
        if args.command == "init":
            init(args, bundle)
        elif args.command == "start":
            start(args, bundle)
        else:
            marker = validate_home(home_path(args.home))
            emit({"result": "STATUS", **observed(marker)})
        return 0
    except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        emit({"result": "REFUSED", "reason": str(error)})
        return 1


if __name__ == "__main__":
    sys.exit(main())
