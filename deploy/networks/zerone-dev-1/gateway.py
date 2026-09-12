#!/usr/bin/env python3
"""Bounded public RPC and prefunded faucet for the explicit zerone-dev-1 ledger.

No minting, registration, validator administration, arbitrary CLI, or private
file access is exposed. Faucet success means observed committed execution.
"""
from __future__ import annotations

import argparse
import base64
import datetime
import fcntl
import hashlib
import http.server
import importlib.util
import ipaddress
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import threading
import time
import urllib.error
import urllib.parse
import urllib.request

CHAIN = "zerone-dev-1"
SCHEMA = "zerone-dev-faucet/v1"
AMOUNT = 1_000_000_000
CAP = 100_000_000_000
MAX_RESPONSE = 8 * 1024 * 1024
MAX_REQUEST = 256 * 1024
READ_METHODS = frozenset(("health", "status", "abci_info", "abci_query", "block", "block_by_hash",
    "block_results", "commit", "validators", "consensus_params", "genesis", "genesis_chunked",
    "header", "header_by_hash", "tx", "net_info"))
POST_METHODS = READ_METHODS | {"broadcast_tx_sync"}
PENDING = ("prepared", "broadcasting", "pending")


class GatewayError(Exception):
    def __init__(self, message, status=503):
        super().__init__(message)
        self.status = status


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def parse(raw):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError("Duplicate JSON field")
            result[key] = value
        return result
    return json.loads(raw, object_pairs_hook=pairs, parse_constant=lambda _: (_ for _ in ()).throw(ValueError("Invalid JSON number")))


def regular_bytes(path, maximum=MAX_RESPONSE, private=False):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
            raise GatewayError("Expected a regular retained file")
        if private and (info.st_uid != os.getuid() or info.st_mode & 0o077):
            raise GatewayError("Private retained file permissions changed")
        data = stream.read(maximum + 1)
    if len(data) > maximum:
        raise GatewayError("Retained file exceeds its size bound")
    return data


def save(path, value):
    """Durable replacement; caller must fail closed on ANY persistence error."""
    temporary = path.with_name(path.name + ".next")
    fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as stream:
        stream.write(canonical(value) + b"\n")
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)
    directory = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(directory)
    finally:
        os.close(directory)


def valid_address(value):
    """Canonical Bech32 zrn account, exactly 20 payload bytes, valid checksum."""
    if not isinstance(value, str) or not re.fullmatch(r"zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}", value):
        return False
    alphabet = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
    digits = [alphabet.index(char) for char in value[4:]]
    checksum = 1
    for item in [ord(c) >> 5 for c in "zrn"] + [0] + [ord(c) & 31 for c in "zrn"] + digits:
        top = checksum >> 25
        checksum = ((checksum & 0x1ffffff) << 5) ^ item
        for index, generator in enumerate((0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3)):
            if (top >> index) & 1:
                checksum ^= generator
    return checksum == 1


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise GatewayError("Unexpected upstream redirect")


class Node:
    def __init__(self, home, binary, rpc, upgrade_packet_sha256=None):
        self.home, self.binary, self.rpc = Path(home), Path(binary), rpc.rstrip("/")
        url = urllib.parse.urlsplit(self.rpc)
        if url.scheme != "http" or url.hostname != "127.0.0.1" or url.path or url.query or url.fragment or url.username:
            raise GatewayError("The gateway requires its loopback node RPC")
        self.manifest = parse(regular_bytes(self.home / ".zerone-dev-runtime.json", private=True))
        if self.manifest.get("chain_id") != CHAIN or self.manifest.get("role") != "validator":
            raise GatewayError("The gateway requires the zerone-dev-1 validator runtime")
        self.genesis = regular_bytes(self.home / "config/genesis.json")
        self.genesis_hash = hashlib.sha256(self.genesis).hexdigest()
        if self.genesis_hash != self.manifest.get("genesis_sha256"):
            raise GatewayError("Runtime genesis changed")
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
        self.history_slots = threading.BoundedSemaphore(2)
        self.upgrade_packet = None
        if upgrade_packet_sha256 is not None:
            spec = importlib.util.spec_from_file_location("development_runtime_upgrade", Path(__file__).with_name("runtime.py"))
            runtime = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(runtime)
            packet, original = runtime.load_upgrade(self.home, upgrade_packet_sha256)
            self.manifest = runtime.upgraded_manifest(original, packet)
            if runtime.digest(self.binary) != self.manifest["binary_sha256"]:
                raise GatewayError("Gateway target binary differs from the staged upgrade")
            runtime.verify_applied_upgrade(self.binary, self.home, self.manifest, packet)
            self.upgrade_packet = packet

    def upstream(self, path="/", body=None):
        headers = {"Accept": "application/json"}
        if body is not None:
            headers["Content-Type"] = "application/json"
        request = urllib.request.Request(self.rpc + path, data=body, headers=headers)
        try:
            with self.opener.open(request, timeout=15) as response:
                data = response.read(MAX_RESPONSE + 1)
                if len(data) > MAX_RESPONSE:
                    raise GatewayError("Node response exceeds the read bound")
                parse(data)
                return data
        except (OSError, ValueError, urllib.error.URLError) as error:
            raise GatewayError("Node RPC unavailable or malformed") from error

    def call(self, method, params=None):
        value = parse(self.upstream(body=canonical({"jsonrpc": "2.0", "id": 1, "method": method, "params": params or {}})))
        if value.get("error"):
            raise GatewayError("Node returned an RPC error")
        return value["result"]

    def ready(self):
        value = self.call("status")
        if (value.get("node_info", {}).get("network") != CHAIN or
                value.get("node_info", {}).get("id") != self.manifest["node_id"] or
                value.get("sync_info", {}).get("catching_up") is not False or
                int(value.get("sync_info", {}).get("latest_block_height", 0)) < 1):
            raise GatewayError("Development node is not ready")
        try:
            stamp = datetime.datetime.fromisoformat(value["sync_info"]["latest_block_time"].replace("Z", "+00:00"))
            age = time.time() - stamp.timestamp()
        except (KeyError, TypeError, ValueError) as error:
            raise GatewayError("Development node has no valid block time") from error
        if stamp.tzinfo is None or not -15 <= age <= 120:
            raise GatewayError("Development blocks are stale or ahead of local time")
        return int(value["sync_info"]["latest_block_height"])

    def cli(self, *args):
        environment = {k: v for k, v in os.environ.items() if not k.startswith("ZERONED_")}
        result = subprocess.run([str(self.binary), *map(str, args), "--home", str(self.home)],
                                capture_output=True, timeout=45, env=environment)
        if result.returncode:
            raise GatewayError("Development command failed; no success is asserted")
        if len(result.stdout) > MAX_RESPONSE:
            raise GatewayError("Development command response exceeds bound")
        return result.stdout.strip()

    def descriptor(self):
        """Freeze once; later startup compares the same ledger/source identity."""
        self.ready()
        genesis = self.call("genesis")["genesis"]
        if genesis.get("chain_id") != CHAIN:
            raise GatewayError("RPC genesis names a different chain")
        versions = parse(self.cli("query", "upgrade", "module-versions", "--node", self.rpc, "--output", "json"))
        knowledge = [row.get("version") for row in versions.get("module_versions", []) if row.get("name") == "knowledge"]
        if len(knowledge) != 1 or str(knowledge[0]) not in ("10", "11"):
            raise GatewayError("Unsupported current knowledge module version")
        knowledge_version = int(knowledge[0])
        if self.upgrade_packet is not None and knowledge_version != 11:
            raise GatewayError("Staged gateway requires applied knowledge 11")
        if knowledge_version == 11 and self.upgrade_packet is None and genesis.get("app_state", {}).get("knowledge", {}).get("fund_settlement_enabled") is not True:
            raise GatewayError("Upgraded ledger requires the explicit staged gateway path")
        local_test = self.manifest.get("local_test") is True
        endpoint = (f"http://127.0.0.1:{self.manifest['gateway_port']}" if local_test else "https://zerone-dev-1.fly.dev")
        value = {"schema": "zerone-shared-development/v1", "chain_id": CHAIN,
            "rpc_url": endpoint, "genesis_url": endpoint + "/genesis.json", "faucet_url": endpoint + "/faucet",
            "genesis_sha256": self.genesis_hash, "rpc_genesis_sha256": hashlib.sha256(canonical(genesis)).hexdigest(),
            "denom": "uzrn", "knowledge_version": knowledge_version, "commitment_scheme": 2, "review_policy_version": 1,
            "account_types": ["human", "agent"], "gas_limit": 2_000_000, "tx_fee_uzrn": "2000000",
            "source_commit": self.manifest["source_commit"], "bootstrap_consensus": "single-operator",
            "reset_policy": "new-chain-id", "peer": self.manifest["advertised_peer"],
            "node_id": self.manifest["node_id"], "runtime_binary_sha256": self.manifest["binary_sha256"],
            "review_window_blocks": self.manifest["review_window_blocks"], "local_test": local_test,
            "faucet": {"amount_uzrn": str(AMOUNT), "lifetime_cap_uzrn": str(CAP),
                "grants_per_address": 1, "grants_per_ip_per_hour": 5, "grants_per_hour": 10},
            "notice": "Valueless development funds. One operator initially produces blocks. Account count does not establish reviewer independence. Queries are operator-node observations, not verified inclusion proofs."}
        directory = self.home / "public"
        directory.mkdir(mode=0o700, exist_ok=True)
        if directory.is_symlink() or not directory.is_dir():
            raise GatewayError("Invalid public packet directory")
        path = directory / ("network-knowledge-11.json" if self.upgrade_packet is not None else "network.json")
        if path.exists():
            if parse(regular_bytes(path)) != value:
                raise GatewayError("Published development descriptor changed; no automatic rewrite")
        else:
            save(path, value)
        self.descriptor_bytes = regular_bytes(path)
        return value

    def history(self, identifier):
        if not self.history_slots.acquire(blocking=False):
            raise GatewayError("History reader busy; try again", 429)
        try:
            height = self.ready()
            value = parse(self.cli("query", "knowledge", "claim-history", identifier,
                "--node", self.rpc, "--height", height, "--output", "json"))
            if value.get("chain_id") != CHAIN or int(value.get("block_height", 0)) != height:
                raise GatewayError("History response context differs from its requested block")
            return value
        finally:
            self.history_slots.release()


class Faucet:
    def __init__(self, node):
        self.node = node
        self.directory = node.home / "dev-faucet"
        info = self.directory.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise GatewayError("Faucet directory is not private and owned")
        self.path = self.directory / "state.json"
        self.lock_fd = os.open(self.directory / ".lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
        info = os.fstat(self.lock_fd)
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_mode & 0o077:
            raise GatewayError("Invalid faucet lock")
        fcntl.flock(self.lock_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        self.state = parse(regular_bytes(self.path, private=True))
        self.validate()
        self.mutex = threading.Lock()
        self.poisoned = False

    def validate(self):
        state = self.state
        if (state.get("schema") != SCHEMA or state.get("chain_id") != CHAIN or
                state.get("genesis_sha256") != self.node.genesis_hash or
                not re.fullmatch(r"[0-9a-f]{64}", state.get("ip_salt", "")) or
                not isinstance(state.get("grants"), dict) or len(state["grants"]) > CAP // AMOUNT):
            raise GatewayError("Faucet state is incompatible or corrupt; it will not be reset")
        for address, grant in state["grants"].items():
            if (not valid_address(address) or grant.get("amount_uzrn") != str(AMOUNT) or
                    grant.get("status") not in (*PENDING, "committed", "failed", "rejected") or
                    not isinstance(grant.get("created_unix"), int) or grant["created_unix"] < 0 or
                    not re.fullmatch(r"[0-9a-f]{64}", grant.get("ip_hash", "")) or
                    not re.fullmatch(r"[0-9A-F]{64}", grant.get("txhash", ""))):
                raise GatewayError("Faucet grant record is corrupt")
            try:
                raw = base64.b64decode(grant["tx_bytes_base64"], validate=True)
            except (KeyError, ValueError) as error:
                raise GatewayError("Faucet signed bytes are corrupt") from error
            if not raw or len(raw) > MAX_REQUEST or hashlib.sha256(raw).hexdigest().upper() != grant["txhash"]:
                raise GatewayError("Faucet transaction hash mismatch")
            if grant["status"] in ("committed", "failed"):
                if int(grant.get("height", 0)) < 1 or (grant["status"] == "committed") != (int(grant.get("code", -1)) == 0):
                    raise GatewayError("Faucet execution receipt is corrupt")

    def persist(self):
        try:
            self.validate()
            save(self.path, self.state)
        except Exception:
            self.poisoned = True
            raise

    def observe(self, grant):
        self.node.ready()
        response = parse(self.node.upstream(body=canonical({"jsonrpc": "2.0", "id": 1, "method": "tx",
            "params": {"hash": base64.b64encode(bytes.fromhex(grant["txhash"])).decode(), "prove": False}})))
        if response.get("error"):
            error = response["error"]
            if error.get("code") == -32603 and re.search(r"(?i)tx .* not found", str(error.get("data", ""))):
                return False
            raise GatewayError("Faucet transaction observation failed")
        result = response.get("result", {})
        if result.get("hash", "").upper() != grant["txhash"] or int(result.get("height", 0)) < 1:
            raise GatewayError("Faucet transaction observation mismatch")
        code = int(result["tx_result"].get("code", 0))
        grant.update(height=str(result["height"]), code=code, status="committed" if code == 0 else "failed")
        self.persist()
        return True

    def public(self, address, grant):
        fields = ("txhash", "status", "height", "code", "amount_uzrn")
        return {"chain_id": CHAIN, "address": address,
            **{key: grant[key] for key in fields if key in grant},
            "committed": grant["status"] == "committed", "denom": "uzrn",
            "notice": "Valueless development grant. Pending or CheckTx acceptance is not a completed transfer."}

    def prepare(self, address, ip_hash, now):
        self.node.ready()
        flags = ["--from", "faucet", "--chain-id", CHAIN, "--node", self.node.rpc,
                 "--keyring-backend", "test", "--gas", "2000000", "--fees", "2000000uzrn"]
        unsigned = parse(self.node.cli("tx", "bank", "send", "faucet", address, f"{AMOUNT}uzrn",
            *flags, "--generate-only", "--output", "json"))
        # These preparation files cannot authorize another transfer; durable
        # state reserves a grant before its first network broadcast.
        draft = self.directory / (address + "-unsigned.json")
        signed_path = self.directory / (address + "-signed.json")
        save(draft, unsigned)
        signed = parse(self.node.cli("tx", "sign", draft, "--from", "faucet", "--chain-id", CHAIN,
            "--node", self.node.rpc, "--keyring-backend", "test", "--output", "json"))
        save(signed_path, signed)
        encoded = self.node.cli("tx", "encode", signed_path).decode()
        raw = base64.b64decode(encoded, validate=True)
        if not raw or len(raw) > MAX_REQUEST:
            raise GatewayError("Invalid generated faucet transaction")
        return {"amount_uzrn": str(AMOUNT), "status": "prepared", "created_unix": now,
                "ip_hash": ip_hash, "txhash": hashlib.sha256(raw).hexdigest().upper(), "tx_bytes_base64": encoded}

    def broadcast(self, grant):
        prior_delivery = grant["status"] != "prepared"
        grant["status"] = "broadcasting"
        self.persist()
        try:
            result = self.node.call("broadcast_tx_sync", {"tx": grant["tx_bytes_base64"]})
            if result.get("hash", "").upper() != grant["txhash"]:
                raise GatewayError("Faucet broadcast hash mismatch")
            # A retry can fail CheckTx because the original is already cached
            # or its sequence was consumed. That cannot reject the original
            # delivery; retain uncertainty and continue querying its exact hash.
            grant["status"] = "pending" if prior_delivery or int(result.get("code", -1)) == 0 else "rejected"
            grant["check_tx_code"] = int(result.get("code", -1))
            self.persist()
        except Exception as error:
            # The broadcasting record remains reserved. Never create a new tx
            # to compensate for an ambiguous delivery or persistence failure.
            raise GatewayError("Faucet delivery remains unresolved; query the same address again") from error
        for _ in range(8):
            if grant["status"] == "rejected" or self.observe(grant):
                break
            time.sleep(1)

    def request(self, address, ip):
        if not valid_address(address):
            raise GatewayError("Expected a checksummed zrn account address", 400)
        if not self.mutex.acquire(blocking=False):
            raise GatewayError("Another grant is being settled; try again", 429)
        try:
            if self.poisoned:
                raise GatewayError("Faucet persistence needs operator recovery")
            grants = self.state["grants"]
            for pending in grants.values():
                if pending["status"] in PENDING:
                    self.observe(pending)
            if address in grants:
                # Repeating this address is an explicit request to recover its
                # grant. Query first, then replay only the already reserved tx.
                # No second signature, quota entry, or transfer is constructed.
                if grants[address]["status"] in PENDING:
                    self.broadcast(grants[address])
                return self.public(address, grants[address])
            if any(grant["status"] in PENDING for grant in grants.values()):
                raise GatewayError("An earlier grant is unresolved; new grants are paused", 409)
            now = int(time.time())
            ip_hash = hashlib.sha256((self.state["ip_salt"] + "|" + str(ipaddress.ip_address(ip))).encode()).hexdigest()
            recent = [grant for grant in grants.values() if grant["created_unix"] > now - 3600]
            if len(grants) >= CAP // AMOUNT:
                raise GatewayError("Development faucet lifetime budget exhausted", 429)
            if len(recent) >= 10 or sum(grant["ip_hash"] == ip_hash for grant in recent) >= 5:
                raise GatewayError("Development faucet hourly allocation limit reached", 429)
            grant = self.prepare(address, ip_hash, now)
            grants[address] = grant
            self.persist()
            self.broadcast(grant)
            return self.public(address, grant)
        finally:
            self.mutex.release()

    def retry(self, digest):
        """Operator-only CLI: observe first, then replay ONLY retained bytes."""
        self.validate()
        if self.poisoned:
            raise GatewayError("Faucet persistence needs recovery")
        candidates = [(address, grant) for address, grant in self.state["grants"].items() if grant["txhash"] == digest]
        if len(candidates) != 1:
            raise GatewayError("No unique retained faucet transaction")
        address, grant = candidates[0]
        if grant["status"] not in PENDING:
            return self.public(address, grant)
        if not self.observe(grant):
            self.broadcast(grant)
        return self.public(address, grant)


class Server(http.server.ThreadingHTTPServer):
    daemon_threads = True
    request_queue_size = 32

    def __init__(self, address, node, faucet):
        self.node, self.faucet = node, faucet
        self.slots = threading.BoundedSemaphore(32)
        self.rates = {}
        self.rate_lock = threading.Lock()
        super().__init__(address, Handler)

    def process_request(self, request, client_address):
        if not self.slots.acquire(blocking=False):
            request.close()
            return
        try:
            super().process_request(request, client_address)
        except Exception:
            self.slots.release()
            raise

    def process_request_thread(self, request, client_address):
        try:
            super().process_request_thread(request, client_address)
        finally:
            self.slots.release()

    def rate_limit(self, ip):
        now = time.monotonic()
        with self.rate_lock:
            if len(self.rates) > 4096:
                self.rates = {key: value for key, value in self.rates.items() if value[0] > now - 1}
            if len(self.rates) > 4096:
                raise GatewayError("Gateway busy", 429)
            start, count = self.rates.get(ip, (now, 0))
            if now - start >= 1:
                start, count = now, 0
            if count >= 40:
                raise GatewayError("RPC request rate exceeded", 429)
            self.rates[ip] = (start, count + 1)


class Handler(http.server.BaseHTTPRequestHandler):
    server_version = "ZeroneDevelopment"

    def setup(self):
        super().setup()
        self.connection.settimeout(15)

    def log_message(self, *args):
        pass

    def respond(self, status, value):
        data = value if isinstance(value, bytes) else canonical(value)
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Connection", "close")
        self.end_headers()
        self.close_connection = True
        if self.command != "HEAD":
            self.wfile.write(data)

    def client_ip(self):
        # On Fly the HTTP handler supplies this header. Local tests and direct
        # private traffic use the socket peer; it is only an allocation throttle.
        address = self.client_address[0]
        if os.environ.get("FLY_APP_NAME") == CHAIN:
            address = self.headers.get("Fly-Client-IP", address)
        try:
            return str(ipaddress.ip_address(address))
        except ValueError as error:
            raise GatewayError("Invalid client address", 400) from error

    def body(self):
        values = self.headers.get_all("Content-Length", [])
        if self.headers.get("Transfer-Encoding") or len(values) != 1 or not re.fullmatch(r"[0-9]+", values[0]):
            raise GatewayError("One bounded Content-Length is required", 400)
        size = int(values[0])
        if not 0 < size <= MAX_REQUEST:
            raise GatewayError("Request body exceeds bound", 413)
        data = self.rfile.read(size)
        if len(data) != size:
            raise GatewayError("Incomplete request body", 400)
        return data

    def dispatch(self):
        ip = self.client_ip()
        self.server.rate_limit(ip)
        if len(self.path) > 16384:
            raise GatewayError("Request path too long", 414)
        path = urllib.parse.urlsplit(self.path)
        if path.scheme or path.netloc or path.fragment:
            raise GatewayError("Invalid request path", 400)
        if self.command in ("GET", "HEAD"):
            if self.headers.get("Transfer-Encoding") or any(v != "0" for v in self.headers.get_all("Content-Length", [])):
                raise GatewayError("Read requests must not have bodies", 400)
            if path.path == "/healthz" and not path.query:
                return self.respond(200, {"chain_id": CHAIN, "height": str(self.server.node.ready())})
            if path.path == "/network.json" and not path.query:
                return self.respond(200, self.server.node.descriptor_bytes)
            if path.path == "/genesis.json" and not path.query:
                return self.respond(200, self.server.node.genesis)
            match = re.fullmatch(r"/claims/([0-9a-f]{32})", path.path)
            if match and not path.query:
                return self.respond(200, self.server.node.history(match.group(1)))
            if path.path[1:] in READ_METHODS:
                return self.respond(200, self.server.node.upstream(self.path))
            raise GatewayError("Unknown development read route", 404)
        if self.command == "POST":
            value = parse(self.body())
            if path.path == "/faucet" and not path.query:
                if not isinstance(value, dict) or set(value) != {"address"}:
                    raise GatewayError("Faucet accepts only a public address", 400)
                result = self.server.faucet.request(value["address"], ip)
                return self.respond(200 if result["committed"] else 202, result)
            if path.path != "/" or path.query:
                raise GatewayError("Unknown development write route", 404)
            if (not isinstance(value, dict) or set(value) - {"jsonrpc", "id", "method", "params"} or
                    value.get("jsonrpc") != "2.0" or value.get("method") not in POST_METHODS or
                    not isinstance(value.get("id"), (str, int)) or isinstance(value.get("id"), bool) or
                    not isinstance(value.get("params", {}), (dict, list))):
                raise GatewayError("Only named query RPCs and broadcast_tx_sync are exposed", 403)
            return self.respond(200, self.server.node.upstream(body=canonical(value)))
        raise GatewayError("Method not allowed", 405)

    def handle_request(self):
        try:
            self.dispatch()
        except GatewayError as error:
            self.respond(error.status, {"error": str(error)})
        except (ValueError, TypeError, KeyError):
            self.respond(400, {"error": "Malformed request or response; no success asserted"})
        except (OSError, subprocess.SubprocessError):
            self.respond(503, {"error": "Development service unavailable; no success asserted"})

    do_GET = do_HEAD = do_POST = do_PUT = do_DELETE = do_PATCH = handle_request

    def do_OPTIONS(self):
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, HEAD, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.send_header("Content-Length", "0")
        self.end_headers()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--home", required=True)
    parser.add_argument("--binary", required=True)
    parser.add_argument("--rpc", required=True)
    parser.add_argument("--bind", default="0.0.0.0", choices=("0.0.0.0", "127.0.0.1"))
    parser.add_argument("--port", type=int, default=8080)
    parser.add_argument("--retry-faucet", metavar="TXHASH")
    parser.add_argument("--upgrade-packet-sha256")
    args = parser.parse_args()
    node = Node(args.home, args.binary, args.rpc, args.upgrade_packet_sha256)
    node.descriptor()
    faucet = Faucet(node)
    if args.retry_faucet:
        print(json.dumps(faucet.retry(args.retry_faucet)))
        return
    if node.manifest.get("local_test") and args.bind != "127.0.0.1":
        raise GatewayError("Local test gateway must bind loopback")
    server = Server((args.bind, args.port), node, faucet)
    print(json.dumps({"service": "zerone-dev-gateway", "chain_id": CHAIN, "port": args.port}), flush=True)
    server.serve_forever(poll_interval=0.5)


if __name__ == "__main__":
    main()
