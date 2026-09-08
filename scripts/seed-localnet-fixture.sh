#!/usr/bin/env bash
# Opt-in, one-validator TEST ONLY fixture. No build, ambient home, or live profile.
# Python owns every child handle; never discover a PID by ps/argv or kill a group.
set -euo pipefail
exec python3 - "$@" <<'SEED_FIXTURE_PY'
import argparse
import base64
import datetime as dt
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
import tempfile
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

CLAIM = "/zerone.claiming_pot.v1.MsgClaim"
SEED = "222000"
GAS = "2000000"
GRANT = "10000000"
MAX_JSON = 262144
# Runner waits for its active CLI (90s ceiling); configured native/provider calls
# are 10s each (reserve-sign: at most five sequential calls). Allow 30s margin
# for cooperative exit/reap. Ordinary daemon/command TERM grace stays 3s.
RUNNER_TERM_GRACE = 120


class Failure(Exception):
    def __init__(self, code, status=1):
        super().__init__(code)
        self.code, self.status = code, status


def need(ok, code):
    if not ok:
        raise Failure(code)


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def digest(data):
    return "sha256:" + hashlib.sha256(data).hexdigest()


def parse(data):
    def pairs(items):
        result = {}
        for key, value in items:
            need(key not in result, "duplicate_json_member")
            result[key] = value
        return result
    need(len(data) <= MAX_JSON, "json_limit")
    try:
        return json.loads(data, object_pairs_hook=pairs)
    except (ValueError, UnicodeError):
        raise Failure("invalid_json") from None


def checked_file(path, limit=1 << 30):
    p = Path(path)
    need(p.is_absolute() and str(p.resolve()) == str(p), "noncanonical_artifact_path")
    fd = os.open(p, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        s = os.fstat(fd)
        need(stat.S_ISREG(s.st_mode) and s.st_nlink == 1 and s.st_size <= limit,
             "unsafe_artifact")
        need(s.st_uid in (0, os.getuid()) and not s.st_mode & 0o022, "unsafe_artifact_owner_mode")
        h, total = hashlib.sha256(), 0
        while True:
            b = os.read(fd, 65536)
            if not b:
                break
            total += len(b)
            need(total <= limit, "artifact_size_limit")
            h.update(b)
        after = os.fstat(fd)
        need((s.st_size, s.st_mtime_ns, s.st_ino) ==
             (after.st_size, after.st_mtime_ns, after.st_ino), "artifact_changed")
        return "sha256:" + h.hexdigest()
    finally:
        os.close(fd)


class Owned:
    """Only a freshly allocated, inode-bound, token-marked directory is removable."""
    def __init__(self, prefix="seed-local-"):
        self.path = Path(tempfile.mkdtemp(prefix=prefix, dir=Path("/tmp").resolve()))
        os.chmod(self.path, 0o700)
        self.identity = (self.path.stat().st_dev, self.path.stat().st_ino)
        self.token = uuid.uuid4().hex
        self.marker = self.path / ".seed-fixture-owned"
        self.marker.write_text(self.token)
        os.chmod(self.marker, 0o600)

    def verify(self):
        s = self.path.lstat()
        need(stat.S_ISDIR(s.st_mode) and s.st_uid == os.getuid() and
             (s.st_dev, s.st_ino) == self.identity and stat.S_IMODE(s.st_mode) == 0o700,
             "cleanup_root_identity")
        m = self.marker.lstat()
        need(stat.S_ISREG(m.st_mode) and m.st_nlink == 1 and m.st_uid == os.getuid(),
             "cleanup_marker_shape")
        need(self.marker.read_text() == self.token, "cleanup_marker_mismatch")

    def remove(self):
        self.verify()
        # Never follow links, including links a runner might have placed in its home.
        need(getattr(shutil.rmtree, "avoids_symlink_attacks", False), "unsafe_rmtree_platform")
        shutil.rmtree(self.path)


def write_json(path, value):
    data = canonical(value)
    need(len(data) <= MAX_JSON, "json_limit")
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as f:
        f.write(data)
        f.flush()
        os.fsync(f.fileno())


class Child:
    def __init__(self, argv, env, cwd, input_bytes=None, discard=False, runner=False):
        need(input_bytes is None or len(input_bytes) <= MAX_JSON, "stdin_limit")
        self.runner, self.forced_kill = runner, False
        self.process = subprocess.Popen(argv, cwd=cwd, env=env, stdin=subprocess.PIPE if input_bytes is not None else subprocess.DEVNULL,
                                        stdout=subprocess.DEVNULL if discard else subprocess.PIPE,
                                        stderr=subprocess.DEVNULL if discard else subprocess.PIPE)
        self.buffers = [bytearray(), bytearray()]
        self.counts = [0, 0]
        self.threads = []
        self.overflow = False
        if not discard:
            for index, pipe in enumerate((self.process.stdout, self.process.stderr)):
                t = threading.Thread(target=self.drain, args=(index, pipe), daemon=True)
                t.start()
                self.threads.append(t)
        if input_bytes is not None:
            # A helper that never reads stdin must still hit the parent deadline.
            t = threading.Thread(target=self.feed, args=(input_bytes,), daemon=True)
            t.start()
            self.threads.append(t)

    def feed(self, data):
        try:
            self.process.stdin.write(data)
        except (BrokenPipeError, OSError):
            pass
        finally:
            try:
                self.process.stdin.close()
            except (BrokenPipeError, OSError):
                pass

    def drain(self, index, pipe):
        try:
            while True:
                chunk = pipe.read1(8192)
                if not chunk:
                    break
                self.counts[index] += len(chunk)
                room = MAX_JSON - len(self.buffers[index])
                self.buffers[index].extend(chunk[:max(0, room)])
                if self.counts[index] > MAX_JSON:
                    self.overflow = True
        finally:
            pipe.close()

    def stop(self):
        # Popen.poll/wait maintains unreaped-child identity; never signal a stale PID.
        p = self.process
        if p.poll() is None:
            p.terminate()
            try:
                p.wait(timeout=RUNNER_TERM_GRACE if self.runner else 3)
            except subprocess.TimeoutExpired:
                self.forced_kill = True
                p.kill()
                p.wait(timeout=3)
        else:
            p.wait()
        for t in self.threads:
            t.join(timeout=1)
        # A reaped runner PID/closed pipe is not evidence its descendants reaped.
        # Keep this sticky on repeated stop(), including an externally signaled exit.
        need(not self.runner or (not self.forced_kill and p.returncode >= 0), "runner_cleanup_uncertain")
        need(not any(t.is_alive() for t in self.threads), "child_pipe_not_closed")


def allocate_ports():
    # Hold all three sockets simultaneously, so the kernel cannot return duplicates.
    sockets = []
    try:
        for _ in range(3):
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.bind(("127.0.0.1", 0))
            s.listen(1)
            sockets.append(s)
        ports = [s.getsockname()[1] for s in sockets]
        need(len(set(ports)) == 3, "duplicate_port")
        return ports
    finally:
        for s in sockets:
            s.close()


def ports_free(ports):
    sockets = []
    try:
        for port in ports:
            # Connecting detects a live listener without confusing TIME_WAIT with it.
            with socket.socket() as probe:
                probe.settimeout(.1)
                if probe.connect_ex(("127.0.0.1", port)) == 0:
                    return False
            s = socket.socket()
            sockets.append(s)
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            s.bind(("127.0.0.1", port))
        return True
    except OSError:
        return False
    finally:
        for s in sockets:
            s.close()


def toml_set(path, updates):
    section, seen, lines = "", set(), []
    for line in path.read_text().splitlines():
        m = re.fullmatch(r"\s*\[([^]]+)\]\s*", line)
        if m:
            section = m[1]
        m = re.match(r"\s*([\w.-]+)\s*=", line)
        key = (section, m[1]) if m else None
        if key in updates:
            need(key not in seen, "duplicate_config_key")
            line = key[1] + " = " + updates[key]
            seen.add(key)
        lines.append(line)
    need(seen == set(updates), "missing_config_key")
    path.write_text("\n".join(lines) + "\n")


# A tiny bounded protobuf reader is used only for independent bank/module-account
# queries. Claim/pot/allowance decoding stays in the native Go helper.
def varint(n):
    out = bytearray()
    while n > 127:
        out.append((n & 127) | 128)
        n >>= 7
    out.append(n)
    return bytes(out)


def string_field(n, value):
    b = value.encode()
    return varint(n * 8 + 2) + varint(len(b)) + b


def fields(data):
    need(len(data) <= MAX_JSON, "protobuf_limit")
    out, pos = {}, 0
    def integer():
        nonlocal pos
        n = 0
        for shift in range(0, 70, 7):
            need(pos < len(data), "truncated_protobuf")
            b = data[pos]
            pos += 1
            n |= (b & 127) << shift
            if not b & 128:
                return n
        raise Failure("protobuf_integer_limit")
    while pos < len(data):
        tag = integer()
        key, wire = tag >> 3, tag & 7
        need(key > 0 and wire in (0, 2), "unsupported_protobuf")
        if wire == 0:
            value = integer()
        else:
            size = integer()
            need(pos + size <= len(data), "truncated_protobuf")
            value = data[pos:pos + size]
            pos += size
        out.setdefault(key, []).append(value)
    return out


def one(obj, key, default=None):
    values = obj.get(key, [])
    need(len(values) <= 1, "duplicate_protobuf_field")
    return values[0] if values else default


def validate_receipt(data, claimant):
    r = parse(data)
    need(isinstance(r, dict) and set(r) == {"claimed_tx_hash", "claimed_address", "actual_amount", "replay_refused"}, "receipt_schema")
    need(isinstance(r["claimed_tx_hash"], str) and re.fullmatch(r"[0-9A-F]{64}", r["claimed_tx_hash"]), "receipt_tx_hash")
    need(r["claimed_address"] == claimant and r["actual_amount"] == SEED, "receipt_credit")
    need(r["replay_refused"] is True, "runner_replay_not_refused")
    return r


def read_receipt(path, claimant):
    need(path.is_absolute() and path.resolve() == path, "unsafe_receipt_path")
    parent = path.parent.stat()
    need(parent.st_uid == os.getuid() and stat.S_IMODE(parent.st_mode) == 0o700, "unsafe_receipt_parent")
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        s = os.fstat(fd)
        need(stat.S_ISREG(s.st_mode) and s.st_nlink == 1 and s.st_uid == os.getuid() and
             stat.S_IMODE(s.st_mode) == 0o600 and s.st_size <= 4096, "unsafe_receipt_file")
        data = os.read(fd, 4097)
        need(len(data) <= 4096, "receipt_size_limit")
        return validate_receipt(data, claimant)
    finally:
        os.close(fd)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class Fixture:
    def __init__(self, args):
        self.args = args
        self.children = []
        self.node_child = None
        self.stage = "allocate"
        self.root = Owned()
        self.evidence = Owned("seed-local-evidence-")
        self.deadline = time.monotonic() + args.timeout_seconds
        self.ports = []
        self.report = {"schema": "zerone.seed-local-fixture/1", "label": "NON-FINAL local source pipeline only",
                       "result": "FAIL", "commands": [], "checks": {}, "cleanup_errors": []}
        self.home = self.root.path / "node"
        self.claimant_home = self.root.path / "claimant"
        self.runner_home = self.root.path / "runner"
        for p in (self.home, self.claimant_home, self.runner_home, self.root.path / "tmp"):
            p.mkdir(mode=0o700)
        self.env = {"PATH": "/usr/bin:/bin:/usr/sbin:/sbin", "HOME": str(self.runner_home),
                    "TMPDIR": str(self.root.path / "tmp"), "LANG": "C", "LC_ALL": "C",
                    "XDG_CONFIG_HOME": str(self.runner_home / "config"),
                    "XDG_CACHE_HOME": str(self.runner_home / "cache"),
                    "XDG_DATA_HOME": str(self.runner_home / "data"), "NO_COLOR": "1"}
        self.http = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())

    def tick(self):
        need(time.monotonic() < self.deadline, "total_timeout")
        if self.node_child:
            need(self.node_child.process.poll() is None, "node_exited")
            need(not self.node_child.overflow, "node_output_limit")

    def run(self, argv, input_bytes=None, discard=False, allow_failure=False, timeout=30, env=None, runner=False):
        self.tick()
        child = Child(argv, env or self.env, self.root.path, input_bytes, discard, runner=runner)
        self.children.append(child)
        end = min(self.deadline, time.monotonic() + timeout)
        try:
            while child.process.poll() is None:
                self.tick()
                need(time.monotonic() < end, "command_timeout")
                need(not child.overflow, "command_output_limit")
                time.sleep(.025)
            child.stop()
            need(not child.overflow, "command_output_limit")
            rc = child.process.returncode
            # Only static stage names and sizes, never argv, keygen output, or logs.
            self.report["commands"].append({"stage": self.stage, "exit_code": rc, "output_bytes": child.counts})
            need(allow_failure or rc == 0, "command_failed")
            return bytes(child.buffers[0])
        finally:
            original = sys.exc_info()[0]
            try:
                child.stop()
            except Exception:
                if not original:
                    raise
                self.report["cleanup_errors"].append("child_stop_failed")

    def cli(self, *args, home=None, **kwargs):
        return self.run([self.args.zeroned, *map(str, args), "--home", str(home or self.home)], **kwargs)

    def rpc(self, method, params=None, missing=False):
        self.tick()
        need(method in {"status", "block", "tx", "abci_query"}, "fixture_rpc_method")
        request = urllib.request.Request(self.rpc_url, canonical({"jsonrpc": "2.0", "id": 1, "method": method, "params": params or {}}),
                                         {"Content-Type": "application/json"})
        try:
            with self.http.open(request, timeout=min(3, max(.1, self.deadline - time.monotonic()))) as response:
                need(response.status == 200 and response.url == self.rpc_url, "rpc_transport")
                data = response.read(MAX_JSON + 1)
        except (OSError, urllib.error.URLError):
            if missing:
                return None
            raise Failure("rpc_unavailable") from None
        r = parse(data)
        need(r.get("id") == 1 and r.get("jsonrpc") == "2.0", "rpc_envelope")
        if missing and r.get("error"):
            return None
        need(not r.get("error") and isinstance(r.get("result"), dict), "rpc_error")
        return r["result"]

    def height(self):
        s = self.rpc("status")
        need(s["node_info"]["network"] == self.chain and s["sync_info"]["catching_up"] is False,
             "node_chain_or_sync")
        return int(s["sync_info"]["latest_block_height"])

    def wait_height(self, target):
        until = min(self.deadline, time.monotonic() + 60)
        while time.monotonic() < until:
            self.tick()
            if self.rpc("status", missing=True) and self.height() >= target:
                return
            time.sleep(.2)
        raise Failure("height_timeout")

    def abci(self, method, payload, height):
        need(method in {"/cosmos.bank.v1beta1.Query/Balance", "/cosmos.auth.v1beta1.Query/ModuleAccountByName"},
             "fixture_abci_method")
        r = self.rpc("abci_query", {"path": method, "data": payload.hex().upper(), "height": str(height), "prove": False})["response"]
        need(int(r.get("code", 0)) == 0 and int(r["height"]) == height, "abci_query_failed")
        return fields(base64.b64decode(r.get("value", ""), validate=True))

    def balance(self, address, height):
        r = self.abci("/cosmos.bank.v1beta1.Query/Balance", string_field(1, address) + string_field(2, "uzrn"), height)
        coin = fields(one(r, 1, b""))
        need(one(coin, 1) == b"uzrn", "bank_denom")
        amount = one(coin, 2, b"").decode()
        need(re.fullmatch(r"0|[1-9][0-9]*", amount), "bank_amount")
        return amount

    def helper(self, command, **values):
        request = {"protocol": "zerone-seed-io/0.1", "request_id": "fixture-" + command,
                   "timeout_ms": 10000, "command": command, "profile": self.profile, **values}
        data = self.run([self.args.helper, command, "--trust-file", str(self.trust_path), "--disposable-test"],
                        input_bytes=canonical(request), allow_failure=True)
        response = parse(data)
        need(response.get("protocol") == "zerone-seed-io/0.1" and response.get("command") == command and
             response.get("request_id") == request["request_id"], "helper_envelope")
        if response.get("status") != "ok":
            safe_codes = {"invalid_request", "unsupported_command", "limit_exceeded", "profile_mismatch",
                          "policy_mismatch", "plan_mismatch", "node_unavailable", "observation_unknown",
                          "key_mismatch", "unsafe_path", "already_exists", "signature_invalid", "signing_unknown", "internal_error"}
            code = response.get("code")
            self.report["helper_failure"] = {"command": command, "code": code if code in safe_codes else "unrecognized"}
            raise Failure("helper_refused")
        return response["result"]

    def inspect(self):
        observed = self.helper("inspect", policy=self.policy, node=self.node, height=None)
        if observed.get("status") != "observed":
            safe_reasons = {"unavailable", "not_found_unproven", "unsupported_state", "incoherent_height",
                            "chain_mismatch", "genesis_mismatch", "stale", "catching_up", "response_limit", "invalid_response"}
            reason = observed.get("reason")
            self.report["observation_failure"] = {"reason": reason if reason in safe_reasons else "unrecognized"}
            raise Failure("native_observation_unknown")
        need(observed["evidence"]["chain_id"] == "cosmos:" + self.chain, "observation_chain")
        return observed

    def wait_tx(self, tx_hash):
        need(re.fullmatch(r"[0-9A-F]{64}", tx_hash), "invalid_tx_hash")
        until = min(self.deadline, time.monotonic() + 60)
        while time.monotonic() < until:
            r = self.rpc("tx", {"hash": base64.b64encode(bytes.fromhex(tx_hash)).decode(), "prove": False}, missing=True)
            if r:
                need(r["hash"].upper() == tx_hash and int(r["tx_result"].get("code", 0)) == 0, "transaction_failed")
                raw = base64.b64decode(r["tx"], validate=True)
                need(hashlib.sha256(raw).hexdigest().upper() == tx_hash, "tx_bytes_hash")
                h = int(r["height"])
                self.wait_height(h + 1)
                block = self.rpc("block", {"height": str(h)})
                need(block["block"]["header"]["chain_id"] == self.chain and
                     block["block"]["data"]["txs"][int(r["index"])] == r["tx"], "tx_block_inclusion")
                return r
            time.sleep(.2)
        raise Failure("transaction_timeout")

    def operator(self, command, key, *args, native):
        self.stage = command
        response = parse(self.cli("tx", *args, "--from", key, "--keyring-backend", "test",
                                  "--chain-id", self.chain, "--node", self.rpc_url, "--gas", GAS,
                                  "--fees", GAS + "uzrn", "--broadcast-mode", "sync", "--yes", "--output", "json"))
        need(int(response.get("code", 0)) == 0, "operator_checktx_failed")
        tx_hash = response["txhash"].upper()
        tx = self.wait_tx(tx_hash)
        raw = fields(base64.b64decode(tx["tx"], validate=True))
        body = fields(one(raw, 1, b""))
        need(len(body.get(1, [])) == 1, "operator_message_count")
        msg = fields(one(body, 1, b""))
        expected = base64.urlsafe_b64decode(native["value_b64u"] + "=" * (-len(native["value_b64u"]) % 4))
        need(one(msg, 1) == native["type_url"].encode() and one(msg, 2) == expected and
             digest(expected) == native["value_hash"], "operator_native_parity")
        self.report[command] = {"tx_hash": tx_hash, "height": tx["height"], "gas_used": tx["tx_result"]["gas_used"]}
        return tx

    def initialize(self):
        self.stage = "initialize"
        self.chain = "seed-local-" + uuid.uuid4().hex[:12]
        self.ports = allocate_ports()
        p2p, rpc, grpc = self.ports
        self.rpc_url = "http://127.0.0.1:" + str(rpc)
        self.grpc = "127.0.0.1:" + str(grpc)
        version = self.cli("version", "--long")
        self.report["zeroned_version_report_hash"] = digest(version)
        self.report["source_commit_supplied_not_build_attested"] = self.args.source_commit
        self.cli("init", "seed-validator", "--chain-id", self.chain, "--default-denom", "uzrn", discard=True)
        addresses = {}
        for key in ("validator", "registrar", "sponsor", "claimant"):
            home = self.claimant_home if key == "claimant" else self.home
            self.cli("keys", "add", key, "--keyring-backend", "test", home=home, discard=True)
            address = self.cli("keys", "show", key, "-a", "--keyring-backend", "test", home=home).decode().strip()
            need(re.fullmatch(r"zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}", address), "test_key_address")
            addresses[key] = address
            if key != "claimant":
                self.cli("add-genesis-account", address, "2000000000uzrn", discard=True)
        self.addresses = addresses
        self.claimant = addresses["claimant"]
        genesis_path = self.home / "config/genesis.json"
        genesis = parse(genesis_path.read_bytes())
        accounts = genesis["app_state"]["auth"]["accounts"]
        balances = genesis["app_state"]["bank"]["balances"]
        need(not any(a.get("address") == self.claimant for a in accounts) and
             not any(b.get("address") == self.claimant for b in balances), "claimant_in_genesis")
        number = next(a.get("account_number", "0") for a in accounts if a.get("address") == addresses["validator"])
        params = genesis["app_state"]["claiming_pot"]["params"]
        need(not genesis["app_state"]["claiming_pot"].get("pots") and
             not genesis["app_state"]["claiming_pot"].get("claims"), "nonempty_initial_pots")
        # This custom module uses encoding/json, not SDK proto-JSON: uint64 is numeric.
        params.update(bootstrap_registrar=addresses["registrar"], bootstrap_emission_cap_uzrn=SEED,
                      bootstrap_daily_admission_cap=1)
        genesis_path.write_bytes(canonical(genesis))
        self.cli("genesis", "gentx", "validator", "1000000000uzrn", "--chain-id", self.chain,
                 "--keyring-backend", "test", "--account-number", number, "--sequence", "0",
                 "--gas", GAS, "--fees", GAS + "uzrn", "--moniker", "seed-validator", discard=True)
        self.cli("genesis", "collect-gentxs", discard=True)
        self.cli("genesis", "validate", discard=True)
        genesis = parse(genesis_path.read_bytes())
        gentxs = genesis["app_state"]["genutil"]["gen_txs"]
        need(len(gentxs) == 1 and gentxs[0]["body"]["messages"][0]["@type"] == "/cosmos.staking.v1beta1.MsgCreateValidator",
             "not_sdk_validator_gentx")
        # Do not invent a custom Zerone staking validator: Comet uses SDK gentx.
        genesis_path.write_bytes(canonical(genesis))
        self.genesis_path = genesis_path
        toml_set(self.home / "config/config.toml", {
            ("rpc", "laddr"): json.dumps("tcp://127.0.0.1:" + str(rpc)),
            ("rpc", "pprof_laddr"): '""', ("rpc", "unsafe"): "false",
            ("p2p", "laddr"): json.dumps("tcp://127.0.0.1:" + str(p2p)),
            ("p2p", "external_address"): '""', ("p2p", "persistent_peers"): '""',
            ("p2p", "seeds"): '""', ("p2p", "pex"): "false",
            ("p2p", "addr_book_strict"): "false", ("statesync", "enable"): "false",
            ("consensus", "timeout_propose"): '"1s"', ("consensus", "timeout_commit"): '"2s"',
            ("instrumentation", "prometheus"): "false"})
        toml_set(self.home / "config/app.toml", {
            ("", "minimum-gas-prices"): '"1uzrn"', ("api", "enable"): "false",
            ("grpc", "enable"): "true", ("grpc", "address"): json.dumps(self.grpc),
            ("grpc-web", "enable"): "false", ("oracle", "enabled"): "false",
            ("telemetry", "enabled"): "false"})
        need(ports_free(self.ports), "port_stolen_before_start")
        self.stage = "start"
        child = Child([self.args.zeroned, "start", "--home", str(self.home), "--log_level", "error"], self.env, self.root.path)
        self.children.append(child)
        self.node_child = child
        self.wait_height(3)
        listeners = self.run([self.args.lsof, "-nP", "-a", "-p", str(child.process.pid), "-iTCP", "-sTCP:LISTEN", "-Fn"]).decode()
        actual = {line[1:] for line in listeners.splitlines() if line.startswith("n")}
        need(actual == {"127.0.0.1:" + str(p) for p in self.ports}, "unexpected_node_listener")
        self.report["checks"]["loopback_listeners"] = sorted(actual)
        self.report["checks"]["claimant_absent_from_genesis"] = True
        self.report["chain_reference"] = self.chain
        self.report["addresses"] = addresses
        self.report["genesis_hash"] = digest(genesis_path.read_bytes())

    def contracts(self):
        self.stage = "contracts"
        h = self.height()
        module = self.abci("/cosmos.auth.v1beta1.Query/ModuleAccountByName", string_field(1, "claiming_pot"), h)
        any_account = fields(one(module, 1, b""))
        need(one(any_account, 1) == b"/cosmos.auth.v1beta1.ModuleAccount", "module_account_type")
        account = fields(one(fields(one(any_account, 2, b"")), 1, b""))
        module_address = one(account, 1, b"").decode()
        chain_id = "cosmos:" + self.chain
        self.profile = {"protocol": "agent-wallet-zerone.seed-profile/0.1", "chain_reference": self.chain,
                        "chain_id": chain_id, "native_asset_id": chain_id + "/denom:uzrn",
                        "claiming_pot_account": chain_id + ":" + module_address,
                        "genesis_hash": self.report["genesis_hash"], "source_digest": self.args.hashes["source_manifest"],
                        "zerone_core_commit": self.args.source_commit, "cosmos_sdk_version": "v0.53.8",
                        "runtime_sha256": self.args.hashes["runtime"], "helper_sha256": self.args.hashes["helper"],
                        "native_denom": "uzrn", "bech32_prefix": "zrn", "seed_amount_uzrn": SEED,
                        "claim_type_url": CLAIM, "claim_gas_floor": "22222", "tx_gas_cap": "11111111",
                        "min_gas_price_uzrn": "1", "confirmation_depth": 1}
        self.profile["profile_id"] = digest(canonical(self.profile))
        self.node = {"mode": "local", "rpc_url": self.rpc_url, "grpc_address": self.grpc}
        self.node["node_trust_id"] = digest(canonical(self.node))
        now = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
        self.expiration = now + dt.timedelta(seconds=self.args.timeout_seconds + 120)
        stamp = lambda t: t.isoformat(timespec="milliseconds").replace("+00:00", "Z")
        self.policy = {"protocol": "agent-wallet-zerone.seed-policy/0.1", "profile_id": self.profile["profile_id"],
                       "node_trust_id": self.node["node_trust_id"], "claimant_account": chain_id + ":" + self.claimant,
                       "sponsor_account": chain_id + ":" + self.addresses["sponsor"], "pot_id": "bootstrap-" + self.claimant,
                       "max_intents": 1, "seed_amount_uzrn": SEED, "max_fee_uzrn": GAS, "max_gas": GAS,
                       "grant_spend_limit_uzrn": GRANT, "grant_expires_at": stamp(self.expiration),
                       "setup_fee_budget_uzrn": str(2 * int(GAS)), "not_before": stamp(now),
                       "expires_at": stamp(self.expiration - dt.timedelta(seconds=30)),
                       "timeout_height": str(h + 2000), "max_observation_age_seconds": 120, "max_height_lag": 20}
        self.policy["policy_hash"] = digest(canonical(self.policy))
        self.trust_path = self.root.path / "helper-trust.json"
        write_json(self.trust_path, {"protocol": "zerone-seed-trust/0.1", "profile_id": self.profile["profile_id"],
                                    "node": self.node, "genesis_file": str(self.genesis_path),
                                    "runtime_file": self.args.runtime, "source_manifest_file": self.args.source_manifest,
                                    "disposable_test": True})
        for name, value in (("profile", self.profile), ("policy", self.policy), ("node", self.node)):
            write_json(self.root.path / (name + ".json"), value)

    def journey(self):
        self.initialize()
        self.contracts()
        self.stage = "initial_absence"
        before = self.inspect()
        need(before["claimant"]["status"] == "absent" and before["allowance"]["status"] == "absent" and before["pot"] is None,
             "unexpected_initial_state")
        need(self.balance(self.claimant, int(before["evidence"]["anchor"]["height"])) == "0", "initial_balance_not_zero")
        # Native construction is checked separately from authorized CLI signing.
        admission = self.helper("operator-admit", authority=self.addresses["registrar"], address=self.claimant)
        need(admission["type_url"] == "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", "admit_type")
        self.operator("test_admission", "registrar", "claiming_pot", "add-bootstrap-entry", self.claimant, native=admission)
        admitted = self.inspect()
        need(admitted["claimant"]["status"] == "absent" and admitted["pot"]["claimed_amount_uzrn"] == "0" and
             admitted["prior_claim"] is None, "admission_materialized_account")
        now = dt.datetime.now(dt.timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z")
        grant = self.helper("operator-grant", granter=self.addresses["sponsor"], grantee=self.claimant,
                            spend_limit_uzrn=GRANT, expires_at=self.policy["grant_expires_at"], now=now)
        need(grant["type_url"] == "/cosmos.feegrant.v1beta1.MsgGrantAllowance", "grant_type")
        self.operator("test_feegrant", "sponsor", "feegrant", "grant", self.addresses["sponsor"], self.claimant,
                      "--spend-limit", GRANT + "uzrn", "--expiration", self.expiration.strftime("%Y-%m-%dT%H:%M:%SZ"),
                      "--allowed-messages", CLAIM, native=grant)
        ready = self.inspect()
        need(ready["claimant"]["status"] == "found" and ready["claimant"]["sequence"] == "0" and
             ready["claimant"]["public_key"] is None, "grant_account_not_materialized")
        allowance = ready["allowance"]
        need(allowance["status"] == "found" and allowance["allowed_messages"] == [CLAIM] and
             allowance["spend_limit_uzrn"] == GRANT and allowance["expires_at"] == self.policy["grant_expires_at"], "grant_shape")
        need(self.balance(self.claimant, int(ready["evidence"]["anchor"]["height"])) == "0", "funded_claimant")
        self.wait_height(int(ready["pot"]["end_block"]) + 1)
        self.report["checks"]["absence_admission_absence_grant_present_zero"] = True
        self.report["before"] = ready
        self.stage = "runner"
        receipt_path = self.runner_home / "result.json"
        public = {"schema": "zerone.seed-local-runner/1", "label": "NON-FINAL", "rpc_url": self.rpc_url,
                  "grpc_address": self.grpc, "chain_reference": self.chain, "genesis_file": str(self.genesis_path),
                  "zeroned": self.args.zeroned, "helper": self.args.helper, "runtime": self.args.runtime,
                  "source_manifest_file": self.args.source_manifest, "source_commit": self.args.source_commit,
                  "profile_file": str(self.root.path / "profile.json"), "policy_file": str(self.root.path / "policy.json"),
                  "node_file": str(self.root.path / "node.json"), "helper_trust_file": str(self.trust_path),
                  "helper_disposable_test": True, "simulation_requires_unchanged_latest_height": True,
                  "claimant_address": self.claimant, "sponsor_address": self.addresses["sponsor"],
                  "claimant_keyring": {"home": str(self.claimant_home), "backend": "test", "key_name": "claimant"},
                  "work_dir": str(self.runner_home), "result_file": str(receipt_path), "expected_amount_uzrn": SEED,
                  "artifact_hashes": self.args.hashes}
        manifest = self.root.path / "fixture.json"
        write_json(manifest, public)
        env = dict(self.env, SEED_FIXTURE_JSON=str(manifest), SEED_RESULT_JSON=str(receipt_path),
                   SEED_TEST_WORK_DIR=str(self.runner_home), SEED_NON_FINAL="1")
        self.run(self.args.runner, timeout=self.args.timeout_seconds, env=env, runner=True)
        self.root.verify()
        receipt = read_receipt(receipt_path, self.claimant)
        self.confirm(receipt, ready)

    def confirm(self, receipt, ready):
        self.stage = "independent_confirmation"
        tx = self.wait_tx(receipt["claimed_tx_hash"])
        decoded = parse(self.cli("tx", "decode", tx["tx"], "--output", "json"))
        messages = decoded["body"]["messages"]
        need(messages == [{"@type": CLAIM, "claimant": self.claimant, "pot_id": self.policy["pot_id"]}], "unexpected_claim_message")
        fee = decoded["auth_info"]["fee"]
        need(fee.get("granter") == self.addresses["sponsor"] and not fee.get("payer"), "claim_not_sponsored")
        need(len(fee["amount"]) == 1 and fee["amount"][0]["denom"] == "uzrn", "claim_fee_denom")
        amount, gas = int(fee["amount"][0]["amount"]), int(fee["gas_limit"])
        need(22222 <= gas <= int(GAS) and gas <= amount <= int(GAS), "claim_fee_or_floor")
        after = self.inspect()
        claim = after["prior_claim"]
        need(claim and claim["amount_uzrn"] == SEED and claim["claimed_at"] == tx["height"] and
             claim["claimant"] == self.claimant and claim["pot_id"] == self.policy["pot_id"], "native_claim_mismatch")
        need(after["pot"]["claimed_amount_uzrn"] == SEED and after["pot"]["status"] == "depleted", "pot_not_depleted")
        need(after["claimant"]["sequence"] == "1", "claim_sequence_not_once")
        need(self.balance(self.claimant, int(after["evidence"]["anchor"]["height"])) == SEED, "credited_balance_mismatch")
        need(after["allowance"]["status"] == "found" and
             int(after["allowance"]["spend_limit_uzrn"]) == int(GRANT) - amount, "allowance_consumption_mismatch")
        need(int(ready["sponsor_balance_uzrn"]) - int(after["sponsor_balance_uzrn"]) == amount, "sponsor_balance_mismatch")
        need(0 < int(tx["tx_result"]["gas_used"]) <= gas, "measured_gas_invalid")
        for name in self.args.hashes:
            need(checked_file(getattr(self.args, name)) == self.args.hashes[name], "artifact_drift")
        need(digest(self.genesis_path.read_bytes()) == self.report["genesis_hash"], "genesis_drift")
        self.report.update(result="PASS", runner_report=receipt, after=after,
                           claimed_tx={"hash": receipt["claimed_tx_hash"], "height": tx["height"],
                                       "gas_used": tx["tx_result"]["gas_used"], "fee_uzrn": str(amount)},
                           replay_evidence="runner-reported CLI refusal; no automatic native replay attempted")

    def finish(self, failure=None):
        # Preserve the initiating error even if stopping/retention also fails.
        status = failure.status if failure else 0
        self.report["failure"] = {"stage": self.stage, "code": failure.code} if failure else None
        for child in reversed(self.children):
            try:
                child.stop()
            except Exception as error:
                code = "runner_cleanup_uncertain" if isinstance(error, Failure) and error.code == "runner_cleanup_uncertain" else "child_stop_failed"
                self.report["cleanup_errors"].append(code)
        self.node_child = None
        self.report["ports_closed"] = ports_free(self.ports)
        self.report["direct_children_reaped"] = all(c.process.poll() is not None for c in self.children)
        if not self.report["ports_closed"]:
            self.report["cleanup_errors"].append("ports_still_in_use")
        self.report["artifact_hashes"] = self.args.hashes
        try:
            self.evidence.verify()
            # Preserve bounded allowlisted observations BEFORE deleting any test state.
            snapshot = dict(self.report, result="UNFINALIZED", cleanup_pending=True)
            write_json(self.evidence.path / "observations.json", snapshot)
            need(self.report["direct_children_reaped"], "child_still_running")
            need(self.report["ports_closed"], "ports_still_in_use")
            need(not self.report["cleanup_errors"], "cleanup_uncertain")
            if self.args.retain_test_state:
                self.root.verify()
                self.report["retained_test_state"] = str(self.root.path)
            else:
                self.root.remove()
                self.report["test_state_deleted"] = True
        except Exception:
            self.report["cleanup_errors"].append("owned_state_cleanup_failed")
            self.report["unremoved_test_state"] = str(self.root.path)
        if self.report["cleanup_errors"]:
            self.report["result"] = "FAIL"
            status = status or 1
        try:
            write_json(self.evidence.path / "evidence.json", self.report)
            print(json.dumps({"result": self.report["result"], "label": self.report["label"],
                              "evidence_file": str(self.evidence.path / "evidence.json"),
                              "retained_test_state": self.report.get("retained_test_state")}))
        except Exception:
            status = status or 1
            print("seed fixture: public evidence write failed", file=sys.stderr)
        return status


def arguments(argv):
    p = argparse.ArgumentParser(description="NON-FINAL disposable one-validator seed fixture. No build, no live profiles.")
    for flag in ("ack-disposable-localnet", "authorize-test-admission", "authorize-test-feegrant", "retain-test-state"):
        p.add_argument("--" + flag, action="store_true")
    for name in ("zeroned", "helper", "runtime", "source-manifest", "source-commit"):
        p.add_argument("--" + name, required=True)
    p.add_argument("--timeout-seconds", type=int, default=600)
    p.add_argument("runner", nargs=argparse.REMAINDER, help="-- absolute-runner [arguments]; must reap its children")
    a = p.parse_args(argv)
    need(a.ack_disposable_localnet and a.authorize_test_admission and a.authorize_test_feegrant, "explicit_test_authorizations_required")
    need(30 <= a.timeout_seconds <= 1800, "timeout_out_of_bounds")
    need(a.runner and a.runner[0] == "--" and len(a.runner) > 1, "explicit_runner_required")
    a.runner = a.runner[1:]
    need(re.fullmatch(r"[0-9a-f]{40}", a.source_commit), "source_commit_required")
    a.hashes = {name: checked_file(getattr(a, name), MAX_JSON if name == "source_manifest" else 1 << 30)
                for name in ("zeroned", "helper", "runtime", "source_manifest")}
    for name in ("zeroned", "helper", "runtime"):
        need(os.access(getattr(a, name), os.X_OK), "artifact_not_executable")
    source = Path(a.source_manifest).read_bytes()
    framed_source = source[:-1] if source.endswith(b"\n") else source
    need(canonical(parse(source)) == framed_source, "source_manifest_not_canonical")
    a.runner_executable = a.runner[0]
    a.hashes["runner_executable"] = checked_file(a.runner_executable)
    need(os.access(a.runner_executable, os.X_OK), "runner_not_executable")
    a.lsof = shutil.which("lsof", path="/usr/sbin:/usr/bin:/sbin:/bin")
    need(a.lsof, "lsof_required")
    return a


def main(argv):
    os.umask(0o077)
    fixture = None
    def interrupted(signum, _frame):
        raise Failure("interrupted", 128 + signum)
    try:
        args = arguments(argv)
        fixture = Fixture(args)
        for sig in (signal.SIGINT, signal.SIGTERM):
            signal.signal(sig, interrupted)
        fixture.journey()
        failure = None
    except Failure as e:
        failure = e
    except Exception:
        failure = Failure("unexpected_error")
    # Ignore further signals only during the bounded child-reaping phase.
    for sig in (signal.SIGINT, signal.SIGTERM):
        signal.signal(sig, signal.SIG_IGN)
    if fixture:
        return fixture.finish(failure)
    if failure:
        print("seed fixture: " + failure.code, file=sys.stderr)
        return failure.status
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
SEED_FIXTURE_PY
