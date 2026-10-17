#!/usr/bin/env python3
"""private-net.py: a private, valueless Zerone network across a few of your own Macs.

Protocol zerone-private-net/v1. One coordinator, N seats (one seat per Mac in normal use).
Keys are born on their seat and never move. Only public files travel, each pinned by sha256:

  round 0  coordinator ──kit (skill, zeroned, invite-<label>.json, START-HERE.md)──▶ each Mac
  round 1  seat     ──seat-<label>.json──▶ coordinator
  round 2  coordinator ──genesis-<chain>.json + network-<chain>.json + NEXT.md──▶ every seat

Subcommands (`private-net.py <command> --help` for details):
  prepare     create the root and its inbox before any kit arrives
  preflight   read-only readiness report for this Mac (firewall, Local Network, clock, ports)
  build       build zeroned from a clean checkout at an exact commit, then pin it
  binary      pin a couriered zeroned after checking its sha256
  invite      coordinator: write one Mac's invitation (chain, label, role, pinned hashes)
  kit         coordinator: one Mac's courier folder (round 0 kit, or round 2 genesis files)
  receive     move couriered files (or a kit folder) into the inbox, checking their names
  seat        create this Mac's seat and its seat packet (with gentx)
  genesis     coordinator only: build genesis + network packet from every seat packet
  install     verify the genesis and network packet, then write this seat's node config
  start/stop  run or stop this seat's node (detached; stop checks the exact command line)
  status      this seat's view of the chain as JSON (--wait-height waits for blocks)
  compare     compare status reports from several seats
  send        one test transfer from this seat's operator key; balance reads one back
  rehearse    run the whole protocol with N loopback seats on this Mac, then stop them

Only chain ids matching zerone-rehearsal-{lan,private,local}-* are accepted, so a live
network (zerone-1, zerone-testnet-1, zerone-dev-1, zerone-2) can never be addressed.
Stdlib only; Python 3.9+ (the macOS Command Line Tools python works).
"""

import os
import sys

# Run as a script, Python puts this file's folder first on sys.path, so a stray module beside it
# (a couriered hashlib.py, say) would shadow the standard library and could fake every hash
# check. Drop that entry before any other import; os and sys are already loaded by then.
if __name__ == "__main__" and sys.path and \
        os.path.realpath(sys.path[0] or os.curdir) == os.path.dirname(os.path.realpath(__file__)):
    del sys.path[0]

import argparse
import datetime
import errno
import hashlib
import json
import platform
import pwd
import re
import shlex
import shutil
import signal
import socket
import stat
import subprocess
import time
import urllib.error
import urllib.request

SEAT_FORMAT = "zerone-private-net-seat/v1"
NETWORK_FORMAT = "zerone-private-net/v1"
INVITE_FORMAT = "zerone-private-net-invite/v1"
STATUS_FORMAT = "zerone-private-net-status/v1"
PROFILE = "private-test-v1"
CHAIN_RE = re.compile(r"zerone-rehearsal-(lan|private|local)-[0-9a-z][0-9a-z-]{0,23}")
LABEL_RE = re.compile(r"[a-z][a-z0-9-]{1,23}")
NODE_ID_RE = re.compile(r"[0-9a-f]{40}")
ADDR_RE = re.compile(r"zrn1[02-9ac-hj-np-z]{38,58}")
DENOM = "uzrn"
UZRN_PER_ZRN = 1000000
DEFAULT_STAKE_ZRN = 1000
FLOAT_ZRN = 1000  # spendable balance on top of the stake, for test transactions
DEFAULT_ROOT = "/Users/Shared/zerone-private"
DEFAULT_P2P_PORT = 36656  # not 26656: keeps bare `zeroned` CLI calls and old homes off this chain
REHEARSAL_BASE_PORT = 46656
TX_GAS = 250000
TX_FEE = 250000  # the ante handler requires fee >= gas x 1uzrn
CHECKPOINT_EVERY = 100
CHECKPOINT_SPAN = 2000
SOCKETFILTERFW = "/usr/libexec/ApplicationFirewall/socketfilterfw"
CODESIGN = "/usr/bin/codesign"
# Go's linker signs every binary ad hoc as "a.out". The macOS application firewall keys its app
# entries by that identifier, so zeroned needs its own (see adhoc_sign and firewall_report).
SIGN_IDENTIFIER = "org.zerone.zeroned"
GO_DEFAULT_IDENTIFIER = "a.out"
SKILL_NAME = "run-a-private-zerone-net"
SKILL_DIR = os.path.dirname(os.path.dirname(os.path.realpath(__file__)))
# The skill folder a kit carries, exactly: no other file may ride along beside the pinned tool.
SKILL_FILES = ("SKILL.md", os.path.join("references", "protocol.md"),
               os.path.join("scripts", "private-net.py"), os.path.join("scripts", "test_private_net.py"))
# Everything a courier may carry into an inbox. Anything else is refused by `receive`.
COURIER_NAME_RE = re.compile(r"(invite|seat|status)-(?:%s)\.json|(genesis|network)-(?:%s)\.json"
                             r"|zeroned|START-HERE\.md|NEXT\.md" % (LABEL_RE.pattern, CHAIN_RE.pattern))
# Reports and pages that may be re-sent within one chain; every other arrival pins something.
REPLACEABLE_RE = re.compile(r"status-.*\.json|START-HERE\.md|NEXT\.md")
MAX_COURIER_JSON = 16 << 20
MAX_COURIER_BINARY = 1 << 30

# The private-test-v1 genesis profile. Everything else stays at the binary's own defaults,
# which every seat can reproduce locally and check (see verify_genesis).
PROFILE_PATCH = {
    ("app_state", "slashing", "params", "signed_blocks_window"): "100000",
    ("app_state", "slashing", "params", "min_signed_per_window"): "0.050000000000000000",
    ("app_state", "slashing", "params", "downtime_jail_duration"): "60s",
    ("app_state", "slashing", "params", "slash_fraction_downtime"): "0.000000000000000000",
}
# Genesis fields the ceremony legitimately changes. A seat checks that nothing else differs
# from its own `zeroned init` genesis with PROFILE_PATCH applied, and checks these fields
# against the network packet.
CEREMONY_FIELDS = [
    ("genesis_time",),
    ("app_state", "auth", "accounts"),
    ("app_state", "bank", "balances"),
    ("app_state", "bank", "supply"),
    ("app_state", "genutil", "gen_txs"),
]


class Refusal(Exception):
    """A protocol check failed. Nothing irreversible was done after the check."""


# ── small helpers ──────────────────────────────────────────────────────────────

def now_utc():
    return datetime.datetime.now(datetime.timezone.utc)


def iso(ts):
    return ts.strftime("%Y-%m-%dT%H:%M:%SZ")


def sha256_bytes(data):
    return hashlib.sha256(data).hexdigest()


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def tool_sha256():
    return sha256_file(os.path.realpath(__file__))


def emit(obj):
    print(json.dumps(obj, indent=2, sort_keys=False))


def canonical(obj):
    return json.dumps(obj, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def _no_floats(text):
    raise Refusal("JSON contains a float literal (%s); refusing to rewrite it" % text)


def parse_json(data, allow_floats=False):
    if isinstance(data, bytes):
        data = data.decode("utf-8")
    return json.loads(data) if allow_floats else json.loads(data, parse_float=_no_floats)


def load_json(path, allow_floats=False):
    with open(path, "rb") as f:
        return parse_json(f.read(), allow_floats)


def write_new(path, data, mode=0o600):
    """Create a new file exclusively (never overwrite, never follow a symlink)."""
    flags = os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0)
    fd = os.open(path, flags, mode)
    with os.fdopen(fd, "wb") as f:
        f.write(data if isinstance(data, bytes) else data.encode("utf-8"))
    return path


def write_replace(path, data, mode=0o600):
    tmp = path + ".tmp-%d" % os.getpid()
    write_new(tmp, data, mode)
    os.replace(tmp, path)
    return path


def json_bytes(obj):
    return (json.dumps(obj, indent=2, ensure_ascii=False) + "\n").encode("utf-8")


def set_path(obj, path, value):
    for key in path[:-1]:
        obj = obj[key]
    if path[-1] not in obj:
        raise Refusal("genesis has no field %s" % ".".join(path))
    obj[path[-1]] = value


def del_path(obj, path):
    for key in path[:-1]:
        obj = obj.get(key, {})
    obj.pop(path[-1], None)


def diff_paths(a, b, prefix=""):
    """Paths where two JSON values differ (bounded list)."""
    out = []
    if isinstance(a, dict) and isinstance(b, dict):
        for k in sorted(set(a) | set(b)):
            p = prefix + "." + k if prefix else k
            if k not in a or k not in b:
                out.append(p)
            else:
                out.extend(diff_paths(a[k], b[k], p))
    elif isinstance(a, list) and isinstance(b, list) and len(a) == len(b):
        for i, (x, y) in enumerate(zip(a, b)):
            out.extend(diff_paths(x, y, "%s[%d]" % (prefix, i)))
    elif a != b:
        out.append(prefix or "<root>")
    return out[:50]


def clean_env(extra=None):
    """The environment for every zeroned call. ZERONED_* variables override config.toml,
    app.toml and key paths (cosmos-sdk AutomaticEnv), so none of them may leak in."""
    env = {k: v for k, v in os.environ.items() if not k.upper().startswith("ZERONED_")}
    env.update(extra or {})
    return env


def run(argv, check=True, quiet_stderr=False, env=None, cwd=None, timeout=600):
    """Run a command without a shell. Returns stdout text."""
    proc = subprocess.run(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                          stdin=subprocess.DEVNULL, env=clean_env() if env is None else env,
                          cwd=cwd, timeout=timeout)
    if check and proc.returncode != 0:
        tail = "" if quiet_stderr else proc.stderr.decode("utf-8", "replace")[-1500:]
        raise Refusal("command failed (%d): %s\n%s" % (proc.returncode, " ".join(argv[:4]), tail))
    return proc.stdout.decode("utf-8", "replace")


# ── bech32 (to compare operator and validator addresses) ───────────────────────

_B32 = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"


def bech32_payload(addr):
    """(hrp, payload bytes) of a bech32 address; Refusal if malformed."""
    if not addr or addr.lower() != addr or "1" not in addr:
        raise Refusal("bad bech32 address %r" % addr)
    hrp, data = addr.rsplit("1", 1)
    try:
        values = [_B32.index(c) for c in data]
    except ValueError:
        raise Refusal("bad bech32 address %r" % addr)
    chk = 1
    for v in [ord(c) >> 5 for c in hrp] + [0] + [ord(c) & 31 for c in hrp] + values:
        top = chk >> 25
        chk = (chk & 0x1ffffff) << 5 ^ v
        for i in range(5):
            chk ^= (0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3)[i] if (top >> i) & 1 else 0
    if chk != 1 or len(values) < 7:
        raise Refusal("bad bech32 checksum in %r" % addr)
    acc, bits, out = 0, 0, bytearray()
    for v in values[:-6]:
        acc = (acc << 5) | v
        bits += 5
        while bits >= 8:
            bits -= 8
            out.append((acc >> bits) & 0xff)
    return hrp, bytes(out)


# ── guards ─────────────────────────────────────────────────────────────────────

def under_git(path):
    p = os.path.realpath(path)
    while True:
        if os.path.lexists(os.path.join(p, ".git")):
            return p
        parent = os.path.dirname(p)
        if parent == p:
            return None
        p = parent


def account_home():
    return os.path.realpath(pwd.getpwuid(os.getuid()).pw_dir)


def check_private_dir(path, create=True):
    """ROOT and everything below it: owner-only, outside the account's home (so outside
    iCloud, cloud-storage and TCC-protected folders), and outside any git tree."""
    real = os.path.realpath(path)
    home = account_home()
    inside = real.casefold() == home.casefold() or real.casefold().startswith(home.casefold() + os.sep)
    p = real
    while not inside:
        if os.path.exists(p) and os.path.exists(home) and os.path.samefile(p, home):
            inside = True
        parent = os.path.dirname(p)
        if parent == p:
            break
        p = parent
    if inside:
        raise Refusal("%s is inside your home folder; use a directory under /Users/Shared" % path)
    if not os.path.exists(real):
        g = under_git(os.path.dirname(real))
        if g:
            raise Refusal("%s would be inside the git tree at %s" % (path, g))
        if not create:
            raise Refusal("%s does not exist" % path)
        os.makedirs(real, mode=0o700)
        os.chmod(real, 0o700)
    st = os.stat(real)
    if st.st_uid != os.getuid():
        raise Refusal("%s is not owned by this user" % path)
    if stat.S_IMODE(st.st_mode) & 0o077:
        raise Refusal("%s is readable by others (mode %o); chmod 700 it" % (path, stat.S_IMODE(st.st_mode)))
    g = under_git(real)
    if g:
        raise Refusal("%s is inside the git tree at %s" % (path, g))
    return real


def check_chain_id(chain_id):
    if not CHAIN_RE.fullmatch(chain_id or ""):
        raise Refusal("chain id %r refused: use zerone-rehearsal-{lan,private,local}-<suffix>, "
                      "and a new suffix for every reset" % chain_id)
    return chain_id


def check_label(label):
    if not LABEL_RE.fullmatch(label or ""):
        raise Refusal("seat label %r refused: 2-24 chars, a-z 0-9 -, starting with a letter" % label)
    return label


def check_ipv4(ip):
    parts = (ip or "").split(".")
    if len(parts) != 4 or not all(p.isdigit() and 0 <= int(p) <= 255 and str(int(p)) == p for p in parts):
        raise Refusal("%r is not a dotted IPv4 literal (CometBFT resolves names only once, at startup)" % ip)
    return ip


SANDBOX_HINT = ("binding a socket is not permitted (%s): this command is running inside an agent sandbox; "
                "run it outside the sandbox with the owner's approval")


def check_p2p_port(port):
    """P2P port; the RPC port is the next one, so it must exist too."""
    if not isinstance(port, int) or not 1024 <= port <= 65534:
        raise Refusal("p2p port %r refused: use 1024-65534 (RPC takes the next port)" % (port,))
    return port


def ip_is_local(ip):
    """True when this Mac holds the address right now."""
    s = None
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.bind((ip, 0))
        return True
    except OSError as e:
        if e.errno in (errno.EPERM, errno.EACCES):
            raise Refusal(SANDBOX_HINT % e.strerror)
        return False
    finally:
        if s:
            s.close()


def check_seat_ip(ip, loopback):
    check_ipv4(ip)
    if loopback:
        return ip
    if ip.startswith("127.") or ip == "0.0.0.0":
        raise Refusal("%s is not a LAN address" % ip)
    if not ip_is_local(ip):
        raise Refusal("this Mac does not hold %s right now; run preflight to list its LAN addresses" % ip)
    return ip


def port_free(port, host):
    """Probe the way Go's listener binds (SO_REUSEADDR), so TIME_WAIT is not 'in use'."""
    s = None
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind((host, port))
        return True
    except OSError as e:
        if e.errno == errno.EADDRNOTAVAIL:
            raise Refusal("this Mac does not hold %s right now; is it on the home LAN?" % host)
        if e.errno in (errno.EPERM, errno.EACCES):
            raise Refusal(SANDBOX_HINT % e.strerror)
        return False
    finally:
        if s:
            s.close()


def lan_addresses():
    """[(interface, ipv4)] for up, broadcast-capable interfaces (no loopback, tunnels or link-local)."""
    out, ifc, up = [], None, False
    for line in run(["ifconfig"], check=False).splitlines():
        m = re.match(r"^([A-Za-z0-9]+): flags=\d+<([^>]*)>", line)
        if m:
            ifc, flags = m.group(1), m.group(2).split(",")
            up = "UP" in flags and "BROADCAST" in flags and "RUNNING" in flags
            continue
        m = re.match(r"^\s+inet (\d+\.\d+\.\d+\.\d+) ", line)
        if m and ifc and up and not ifc.startswith(("lo", "utun", "ipsec", "gif", "stf", "bridge")):
            ip = m.group(1)
            if not ip.startswith(("127.", "169.254.")):
                out.append((ifc, ip))
    return out


def probe_tcp(ip, port, timeout=2.0):
    """Classify one outbound LAN dial the way zeroned would make it from this app context."""
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(timeout)
    try:
        s.connect((ip, port))
        return "ok", "%s:%d accepted a connection" % (ip, port)
    except ConnectionRefusedError:
        return "ok", "%s:%d reachable (refused: nothing listening yet)" % (ip, port)
    except socket.timeout:
        return "warn", "%s:%d did not answer in %.0fs (a firewall drops it, or the host is asleep)" % (ip, port, timeout)
    except OSError as e:
        if e.errno in (errno.EHOSTUNREACH, errno.EPERM, errno.EACCES, errno.ENETUNREACH):
            return "fail", ("%s:%d: %s. If other devices reach it, macOS Local Network privacy is blocking "
                            "the app that runs this agent (System Settings > Privacy & Security > Local Network), "
                            "or the agent's sandbox blocks the network" % (ip, port, e.strerror))
        return "warn", "%s:%d: %s" % (ip, port, e)
    finally:
        s.close()


# ── layout ─────────────────────────────────────────────────────────────────────

class Layout:
    def __init__(self, root, chain_id=None):
        self.root = check_private_dir(root)
        self.bin_dir = os.path.join(self.root, "bin")
        self.binary = os.path.join(self.bin_dir, "zeroned")
        self.binary_record = os.path.join(self.bin_dir, "zeroned.json")
        self.inbox = check_private_dir(os.path.join(self.root, "inbox"))
        self.chain_id = chain_id
        if chain_id:
            check_chain_id(chain_id)
            nets = check_private_dir(os.path.join(self.root, "nets"))
            self.chain_dir = check_private_dir(os.path.join(nets, chain_id))
            self.outbox = check_private_dir(os.path.join(self.chain_dir, "outbox"))
            self.run_dir = check_private_dir(os.path.join(self.chain_dir, "run"))
            self.seats_dir = check_private_dir(os.path.join(self.chain_dir, "seats"))

    def seat_home(self, label):
        return os.path.join(self.seats_dir, check_label(label))

    def seat_file(self, label, kind):
        return os.path.join(self.seats_dir, "%s.%s.json" % (check_label(label), kind))

    def seat_state(self, label):
        path = self.seat_file(label, "state")
        if not os.path.exists(path):
            raise Refusal("no seat %s on chain %s here" % (label, self.chain_id))
        return path

    def coordinator_home(self):
        return os.path.join(self.chain_dir, "coordinator")

    def pid_file(self, label):
        return os.path.join(self.run_dir, "%s.pid" % label)

    def log_file(self, label):
        return os.path.join(self.run_dir, "%s.log" % label)

    def pinned_binary(self):
        if not os.path.isfile(self.binary) or not os.path.isfile(self.binary_record):
            raise Refusal("no pinned binary at %s; run `binary` or `build` first" % self.binary)
        rec = load_json(self.binary_record, allow_floats=True)
        actual = sha256_file(self.binary)
        if actual != rec.get("sha256"):
            raise Refusal("pinned binary sha256 changed: %s != %s" % (actual, rec.get("sha256")))
        return self.binary, rec


# ── zeroned wrappers (always an explicit --home, never ZERONED_* env) ─────────

def zd(binary, *args, **kw):
    return run([binary] + list(args), **kw)


def node_id(binary, home):
    try:
        out = zd(binary, "comet", "show-node-id", "--home", home)
    except Refusal:
        out = zd(binary, "tendermint", "show-node-id", "--home", home)
    nid = out.strip()
    if not NODE_ID_RE.fullmatch(nid):
        raise Refusal("unexpected node id %r" % nid)
    return nid


def consensus_pubkey(binary, home):
    try:
        out = zd(binary, "comet", "show-validator", "--home", home)
    except Refusal:
        out = zd(binary, "tendermint", "show-validator", "--home", home)
    return json.loads(out)


def operator_address(binary, home):
    addr = zd(binary, "keys", "show", "operator", "-a", "--keyring-backend", "test", "--home", home).strip()
    if not ADDR_RE.fullmatch(addr):
        raise Refusal("unexpected operator address %r" % addr)
    return addr


def apply_profile(genesis):
    for path, value in PROFILE_PATCH.items():
        set_path(genesis, path, value)
    return genesis


def ceremony_projection(genesis):
    g = json.loads(json.dumps(genesis))
    for path in CEREMONY_FIELDS:
        del_path(g, path)
    return g


def platform_label(rec):
    go = (rec.get("version") or {}).get("go", "")
    return go.split()[-1].replace("/", "-") if go else None


# ── TOML line editor (section-aware, refuses unknown keys) ────────────────────

def set_toml(path, settings):
    """settings: {(section, key): toml_literal}; section "" is the top level."""
    with open(path, "r", encoding="utf-8") as f:
        lines = f.read().split("\n")
    section = ""
    seen = set()
    key_re = re.compile(r"^([A-Za-z0-9_.-]+)\s*=")
    for i, line in enumerate(lines):
        s = line.strip()
        if s.startswith("[") and s.endswith("]") and not s.startswith("[["):
            section = s[1:-1].strip()
            continue
        m = key_re.match(s)
        if m and (section, m.group(1)) in settings:
            lines[i] = "%s = %s" % (m.group(1), settings[(section, m.group(1))])
            seen.add((section, m.group(1)))
    missing = [k for k in settings if k not in seen]
    if missing:
        raise Refusal("%s has no %s; the config layout changed, stop and check" %
                      (os.path.basename(path), ", ".join("[%s] %s" % k for k in missing)))
    write_replace(path, "\n".join(lines), stat.S_IMODE(os.stat(path).st_mode))


def toml_str(s):
    return json.dumps(s)


# ── RPC ────────────────────────────────────────────────────────────────────────

def rpc(port, path, timeout=3):
    url = "http://127.0.0.1:%d%s" % (port, path)
    try:
        with urllib.request.urlopen(url, timeout=timeout) as r:
            body = json.loads(r.read().decode("utf-8"))
    except (OSError, ValueError) as e:
        raise Refusal("RPC 127.0.0.1:%d not answering (%s): before genesis_time, not started, or exited "
                      "(see nets/<id>/run/<label>.log)" % (port, e))
    if "result" not in body:
        raise Refusal("RPC %s returned no result" % path)
    return body["result"]


def rpc_or_absent(port, path, timeout=2):
    """rpc() for a node that may not be listening yet: no listener returns None, since the node opens RPC
    only at genesis_time. Any answer that is not CometBFT's (an HTTP error, not JSON, no result) comes from
    another process, and is refused rather than waited out."""
    url = "http://127.0.0.1:%d%s" % (port, path)
    try:
        with urllib.request.urlopen(url, timeout=timeout) as r:
            body = json.loads(r.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        raise Refusal("another process answers on 127.0.0.1:%d (HTTP %s), not this seat's node" % (port, e.code))
    except urllib.error.URLError as e:
        if isinstance(e.reason, (ConnectionRefusedError, ConnectionResetError, socket.timeout, TimeoutError)):
            return None
        raise Refusal("RPC 127.0.0.1:%d: %s" % (port, e.reason))
    except (ConnectionResetError, socket.timeout, TimeoutError):
        return None  # a node still opening its listener
    except ValueError:
        raise Refusal("another process answers on 127.0.0.1:%d (not JSON), not this seat's node" % port)
    if not isinstance(body, dict) or "result" not in body:
        raise Refusal("another process answers on 127.0.0.1:%d (no RPC result), not this seat's node" % port)
    return body["result"]


# ── code signing ───────────────────────────────────────────────────────────────

def parse_signing_identifier(text):
    """The Identifier= line of `codesign -d` output, or None."""
    m = re.search(r"^Identifier=(.+)$", text, re.M)
    return m.group(1).strip() if m else None


def signing_identifier(binary):
    """The code-signing identifier of `binary`, or None (no codesign here, or not signed)."""
    if not os.path.exists(CODESIGN):
        return None
    proc = subprocess.run([CODESIGN, "-d", "--verbose=1", binary], stdout=subprocess.PIPE,
                          stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, env=clean_env(), timeout=60)
    return parse_signing_identifier(proc.stdout.decode("utf-8", "replace")) if proc.returncode == 0 else None


def adhoc_sign(binary):
    """Re-sign ad hoc as SIGN_IDENTIFIER, the way the Makefile signs darwin-acl-check. Under Go's
    default "a.out", `socketfilterfw --add` prints nothing and updates whichever other a.out app
    already holds the firewall entry (ipfs, another Go tool), so zeroned never gets a rule.
    The signature has no timestamp, so the same build re-signs to the same bytes."""
    run([CODESIGN, "--force", "--sign", "-", "--identifier", SIGN_IDENTIFIER, "--timestamp=none", binary])
    run([CODESIGN, "--verify", "--strict", binary])
    ident = signing_identifier(binary)
    if ident != SIGN_IDENTIFIER:
        raise Refusal("re-signed %s reports identifier %r, not %s" % (binary, ident, SIGN_IDENTIFIER))


# ── firewall ───────────────────────────────────────────────────────────────────

def firewall_report(binary):
    """(status, detail) for inbound P2P to `binary` under the macOS application firewall."""
    if not os.path.exists(SOCKETFILTERFW):
        return "ok", "no macOS application firewall here"
    state = run([SOCKETFILTERFW, "--getglobalstate"], check=False).strip()
    if "disabled" in state:
        return "ok", state
    if "enabled" in run([SOCKETFILTERFW, "--getblockall"], check=False):
        return "fail", "the firewall blocks ALL incoming connections; no allow rule can help until that is off"
    lines = run([SOCKETFILTERFW, "--listapps"], check=False).splitlines()
    for i, line in enumerate(lines):
        if line.strip().endswith(binary):
            nxt = lines[i + 1] if i + 1 < len(lines) else ""
            if "Allow" in nxt:
                return "ok", "%s is allowed incoming connections" % binary
            return "fail", "%s is listed but blocked; the owner unblocks it (command below)" % binary
    detail = ("firewall on and zeroned not yet allowed. The prompt would appear only at genesis_time, "
              "when the node opens its P2P port, so the owner should allow it now: "
              "sudo %s --add %s ; sudo %s --unblockapp %s" % (SOCKETFILTERFW, binary, SOCKETFILTERFW, binary))
    if os.path.isfile(binary) and signing_identifier(binary) == GO_DEFAULT_IDENTIFIER:
        detail += (". This zeroned is signed with Go's default identifier %s, and the firewall keys apps by "
                   "identifier: if another %s app (ipfs, another Go tool) is already listed, --add prints nothing, "
                   "updates that app's entry instead, and this warning stays. `build` signs zeroned as %s, so a "
                   "new pin does not share that identifier. For this binary, allow zeroned if macOS prompts at genesis_time, or rely on "
                   "this seat dialling out: one connection between two seats is enough"
                   % (GO_DEFAULT_IDENTIFIER, GO_DEFAULT_IDENTIFIER, SIGN_IDENTIFIER))
    return "warn", detail


# ── commands ───────────────────────────────────────────────────────────────────

def cmd_preflight(a):
    checks = []

    def add(name, status, detail):
        checks.append({"check": name, "status": status, "detail": detail})

    add("platform", "ok" if sys.platform == "darwin" else "fail",
        "%s %s macOS %s" % (sys.platform, platform.machine(), platform.mac_ver()[0] or "?"))
    if platform.machine() != "arm64":
        add("arch", "fail", "not Apple silicon: this tool builds and pins darwin-arm64 only, so this Mac cannot seat")
    add("python", "ok" if sys.version_info >= (3, 9) else "fail", sys.version.split()[0])
    add("tool", "ok", {"path": os.path.realpath(__file__), "sha256": tool_sha256()})
    if shutil.which("go"):
        add("go", "ok", run(["go", "version"], check=False).strip())
    else:
        add("go", "warn", "no go on PATH: needed only to `build` here (Go 1.21+ fetches the pinned toolchain)")
    try:
        out = run(["sntp", "-t", "3", "time.apple.com"], check=False, timeout=15)
        m = re.search(r"([+-]\d+\.\d+)\s*\+/-", out)
        if m:
            off = abs(float(m.group(1)))
            add("clock", "ok" if off < 1 else ("warn" if off < 30 else "fail"),
                "offset %.3fs vs time.apple.com (CometBFT 0.38.25 rejects blocks >60s ahead)" % off)
        else:
            add("clock", "warn", "could not parse sntp output")
    except Exception as e:
        add("clock", "warn", "sntp failed: %s" % e)
    lans = lan_addresses()
    add("lan-ipv4", "ok" if lans else "fail",
        [{"interface": i, "ipv4": ip} for i, ip in lans] or "no LAN IPv4 on any interface")
    try:
        p2p_host = check_seat_ip(a.ip, False) if a.ip else "0.0.0.0"
        add("port-%d" % a.p2p_port, "ok" if port_free(a.p2p_port, p2p_host) else "fail",
            "P2P port on %s" % p2p_host)
    except Refusal as e:
        add("port-%d" % a.p2p_port, "fail", str(e))
    add("port-%d" % (a.p2p_port + 1), "ok" if port_free(a.p2p_port + 1, "127.0.0.1") else "fail",
        "RPC port on 127.0.0.1")
    if shutil.which("mullvad"):
        lan = run(["mullvad", "lan", "get"], check=False).strip()
        add("mullvad-lan", "ok" if lan.endswith("allow") else "fail",
            lan + ("" if lan.endswith("allow") else " — the owner runs `mullvad lan set allow`"))
    # Outbound LAN dials need Local Network permission for the app hosting this agent (macOS 15).
    targets = list(a.peer or [])
    if not targets and lans:
        router = run(["ipconfig", "getoption", lans[0][0], "router"], check=False).strip()
        if router:
            targets = ["%s:%d" % (router, a.p2p_port)]
    if not targets:
        add("lan-dial", "warn", "not tested: no router found; pass --peer <another seat's IPv4>")
    for t in targets:
        ip, _, port = t.partition(":")
        try:
            st, detail = probe_tcp(check_ipv4(ip), int(port or a.p2p_port))
        except (Refusal, ValueError) as e:
            st, detail = "fail", str(e)
        add("lan-dial %s" % t, st, detail)
    bin_path = os.path.join(a.root, "bin", "zeroned")
    add("firewall", *firewall_report(bin_path))
    try:
        root = check_private_dir(a.root, create=False)
        free = shutil.disk_usage(root).free / 2 ** 30
        add("root", "ok", "%s (owner-only, outside home and git)" % root)
    except Refusal as e:
        free = shutil.disk_usage("/Users/Shared").free / 2 ** 30
        add("root", "warn", "%s — run `prepare` to create it and its inbox" % e)
    add("disk", "ok" if free > 20 else "warn", "%.0f GiB free (plan ~5 GB per node per month)" % free)
    procs = run(["pgrep", "-fl", "zeroned"], check=False).strip()
    add("zeroned-processes", "ok" if not procs else "warn", procs or "none running")
    if os.path.isfile(bin_path):
        add("pinned-binary", "ok", {"path": bin_path, "sha256": sha256_file(bin_path)})
    else:
        add("pinned-binary", "warn", "none yet")
    worst = "fail" if any(c["status"] == "fail" for c in checks) else (
        "warn" if any(c["status"] == "warn" for c in checks) else "ok")
    emit({"format": "zerone-private-net-preflight/v1", "result": worst, "checks": checks})
    return 0 if worst != "fail" else 2


def pin_binary(lay, src, expect_sha=None, provenance=None):
    actual = sha256_file(src)
    if expect_sha and actual != expect_sha.lower():
        raise Refusal("sha256 mismatch: got %s, expected %s" % (actual, expect_sha))
    check_private_dir(lay.bin_dir)
    if os.path.lexists(lay.binary) != os.path.lexists(lay.binary_record):
        raise Refusal("bin/zeroned and bin/zeroned.json must exist together; move both aside and pin again")
    if os.path.exists(lay.binary):
        if sha256_file(lay.binary) == actual:
            return dict(load_json(lay.binary_record, allow_floats=True), already_pinned=True)
        raise Refusal("%s already holds a different binary. To change it, stop every seat on this Mac, "
                      "then move bin/zeroned and bin/zeroned.json aside; every chain pinned to the old "
                      "binary then needs a new chain id" % lay.binary)
    tmp = lay.binary + ".tmp-%d" % os.getpid()
    shutil.copyfile(src, tmp)
    os.chmod(tmp, 0o700)
    if shutil.which("xattr"):
        run(["xattr", "-c", tmp], check=False)  # AirDropped copies carry com.apple.quarantine
    try:
        if sha256_file(tmp) != actual:
            raise Refusal("copy changed bytes")
        try:
            ver = run([tmp, "version", "--long", "--output", "json"], check=False)
        except OSError as e:
            raise Refusal("this binary does not run on this Mac (%s); it was built for another CPU or OS" % e)
        try:
            v = json.loads(ver)
            version = {k: v.get(k) for k in ("version", "commit", "go", "cosmos_sdk_version") if k in v}
        except ValueError:
            version = {"raw": ver.strip()[:200]}
        rec = {"sha256": actual, "path": lay.binary, "version": version,
               "provenance": provenance or {"source": os.path.abspath(src)}, "pinned_at": iso(now_utc())}
        write_replace(lay.binary_record, json_bytes(rec))
        os.replace(tmp, lay.binary)
        return rec
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


def cmd_prepare(a):
    lay = Layout(a.root)
    emit({"root": lay.root, "inbox": lay.inbox,
          "next": ("the owner moves the kit (skill folder, zeroned, invitation) into the inbox in Finder, or "
                   "`receive <folder>` moves it there where this app may read ~/Downloads")})
    return 0


def read_invite(path):
    inv = load_json(path, allow_floats=True)
    if inv.get("format") != INVITE_FORMAT:
        raise Refusal("%s is not a %s invitation" % (path, INVITE_FORMAT))
    check_chain_id(inv.get("chain_id"))
    check_label(inv.get("label"))
    if inv.get("role") not in ("validator", "full"):
        raise Refusal("invitation has an unknown role")
    return inv


def cmd_binary(a):
    lay = Layout(a.root)
    expect = a.sha256
    if a.invite:
        inv = read_invite(a.invite)
        if expect and expect.lower() != inv["binary_sha256"]:
            raise Refusal("--sha256 disagrees with the invitation")
        expect = inv["binary_sha256"]
    if not expect:
        raise Refusal("give --invite or --sha256: a binary is pinned only against a known hash")
    emit(pin_binary(lay, a.from_path, expect, {"source": "file", "path": os.path.abspath(a.from_path)}))
    return 0


def cmd_build(a):
    if platform.machine() != "arm64" or sys.platform != "darwin":
        raise Refusal("build makes darwin-arm64 only, and this is %s %s" % (sys.platform, platform.machine()))
    if not shutil.which("go"):
        raise Refusal("no go on PATH; install Go 1.21+ (it fetches the exact toolchain go.mod pins)")
    lay = Layout(a.root)
    src = os.path.realpath(a.src)
    head = run(["git", "-C", src, "rev-parse", "HEAD"]).strip()
    if head != a.commit:
        raise Refusal("checkout is at %s, not %s" % (head, a.commit))
    dirty = run(["git", "-C", src, "status", "--porcelain", "--untracked-files=all"]).strip()
    if dirty:
        raise Refusal("checkout is not clean; build only from an exact, clean commit")
    with open(os.path.join(src, "go.mod")) as f:
        m = re.search(r"^go (\d+\.\d+\.\d+)$", f.read(), re.M)
    if not m:
        raise Refusal("go.mod has no exact go version")
    env = clean_env({"GOTOOLCHAIN": "go" + m.group(1)})
    version = a.commit[:8]
    run(["make", "build-darwin-arm64", "VERSION=" + version], env=env, cwd=src, timeout=3600)
    out = os.path.join(src, "build", "zeroned-darwin-arm64")
    info = run(["go", "version", "-m", out], env=env)
    if "vcs.modified=false" not in info or ("vcs.revision=" + a.commit) not in info:
        raise Refusal("built binary does not stamp a clean %s" % a.commit)
    adhoc_sign(out)
    emit(pin_binary(lay, out, None, {"source": "build", "commit": a.commit, "version": version,
                                     "go": "go" + m.group(1), "target": "build-darwin-arm64 (CGO_ENABLED=0)",
                                     "signing_identifier": SIGN_IDENTIFIER}))
    return 0


def cmd_invite(a):
    lay = Layout(a.root, a.chain_id)
    _, rec = lay.pinned_binary()
    check_label(a.label)
    check_label(a.coordinator_label)
    if a.role == "validator" and a.stake_zrn <= 0:
        raise Refusal("a validator needs a positive --stake-zrn")
    path = os.path.join(lay.outbox, "invite-%s.json" % a.label)
    if os.path.lexists(path):
        raise Refusal("%s already exists; one invitation per seat per chain" % path)
    inv = {"format": INVITE_FORMAT, "protocol": NETWORK_FORMAT, "chain_id": lay.chain_id,
           "label": a.label, "role": a.role, "stake_zrn": a.stake_zrn if a.role == "validator" else 0,
           "coordinator": a.coordinator_label, "binary_sha256": rec["sha256"],
           "platform": platform_label(rec), "tool_sha256": tool_sha256(), "created_at": iso(now_utc()),
           "kit": ["private-net.py (sha256 tool_sha256)", "zeroned (sha256 binary_sha256)", "this invitation"]}
    write_new(path, json_bytes(inv), 0o644)
    nxt = ("the coordinator seats with this file from the outbox before any kit is made"
           if a.label == a.coordinator_label else
           "once the coordinator has seated, `kit --chain-id %s --label %s --out <folder>` writes %s's courier folder"
           % (lay.chain_id, a.label, a.label))
    emit({"invitation": path, "next": nxt})
    return 0


def make_seat(lay, label, role, ip, p2p_port, stake_zrn, binary, loopback=False):
    check_label(label)
    if role not in ("validator", "full"):
        raise Refusal("role must be validator or full")
    check_seat_ip(ip, loopback)
    check_p2p_port(p2p_port)
    if role == "validator" and int(stake_zrn) <= 0:
        raise Refusal("a validator needs a positive stake")
    home = lay.seat_home(label)
    if os.path.lexists(home):
        raise Refusal("%s already exists; a seat is created once per chain id" % home)
    bsha = sha256_file(binary)
    zd(binary, "init", label, "--chain-id", lay.chain_id, "--default-denom", DENOM, "--home", home)
    os.chmod(home, 0o700)
    default_genesis = load_json(os.path.join(home, "config", "genesis.json"))
    write_new(lay.seat_file(label, "default-genesis"), json_bytes(default_genesis))
    nid = node_id(binary, home)
    pub = consensus_pubkey(binary, home)
    packet = {
        "format": SEAT_FORMAT, "chain_id": lay.chain_id, "label": label, "role": role,
        "node_id": nid, "consensus_pubkey": pub, "p2p_endpoint": "%s:%d" % (ip, p2p_port),
        "binary_sha256": bsha, "tool_sha256": tool_sha256(), "created_at": iso(now_utc()),
    }
    if role == "validator":
        # The mnemonic is discarded on purpose: this valueless key lives only in this home.
        zd(binary, "keys", "add", "operator", "--keyring-backend", "test", "--home", home,
           "--no-backup", "--output", "json", quiet_stderr=True)
        addr = operator_address(binary, home)
        stake = int(stake_zrn) * UZRN_PER_ZRN
        balance = stake + FLOAT_ZRN * UZRN_PER_ZRN
        # gentx checks the delegator's balance in the local genesis; the coordinator funds the same amount.
        zd(binary, "add-genesis-account", addr, "%d%s" % (balance, DENOM), "--home", home)
        gentx_path = lay.seat_file(label, "gentx")
        zd(binary, "genesis", "gentx", "operator", "%d%s" % (stake, DENOM), "--chain-id", lay.chain_id,
           "--keyring-backend", "test", "--home", home, "--moniker", label,
           "--commission-rate", "0.10", "--commission-max-rate", "0.20", "--commission-max-change-rate", "0.01",
           "--ip", ip, "--p2p-port", str(p2p_port), "--output-document", gentx_path)
        gentx = load_json(gentx_path, allow_floats=True)
        packet.update({"operator_address": addr, "stake_uzrn": str(stake), "balance_uzrn": str(balance),
                       "gentx": gentx})
    state = {"label": label, "role": role, "ip": ip, "p2p_port": p2p_port, "home": home,
             "loopback": loopback, "binary_sha256": bsha, "stage": "seat"}
    write_new(lay.seat_file(label, "state"), json_bytes(state))
    packet_path = os.path.join(lay.outbox, "seat-%s.json" % label)
    write_new(packet_path, json_bytes(packet), 0o644)
    return packet_path, packet


def cmd_seat(a):
    if a.invite:
        inv = read_invite(a.invite)
        for name, flag in (("chain_id", a.chain_id), ("label", a.label), ("role", a.role), ("stake_zrn", a.stake_zrn)):
            if flag is not None and flag != inv[name]:
                raise Refusal("--%s disagrees with the invitation; the invitation decides" % name.replace("_", "-"))
        chain_id, label, role, stake = inv["chain_id"], inv["label"], inv["role"], inv["stake_zrn"]
        if inv["tool_sha256"] != tool_sha256():
            raise Refusal("this private-net.py (%s) is not the invitation's (%s); use the couriered copy"
                          % (tool_sha256()[:12], inv["tool_sha256"][:12]))
    else:
        if not (a.chain_id and a.label):
            raise Refusal("give --invite, or --chain-id and --label")
        chain_id, label = a.chain_id, a.label
        role = a.role or "validator"
        stake = DEFAULT_STAKE_ZRN if a.stake_zrn is None else a.stake_zrn
    lay = Layout(a.root, chain_id)
    binary, _ = lay.pinned_binary()
    if a.invite and sha256_file(binary) != inv["binary_sha256"]:
        raise Refusal("the pinned zeroned is not the invitation's binary")
    # A zerone-rehearsal-local-* chain lives on one Mac, so its seats may sit on loopback.
    loopback = chain_id.startswith("zerone-rehearsal-local-") and a.ip == "127.0.0.1"
    path, packet = make_seat(lay, label, role, a.ip, a.p2p_port, stake, binary, loopback=loopback)
    emit({"seat_packet": path, "sha256": sha256_file(path), "label": packet["label"], "role": packet["role"],
          "node_id": packet["node_id"],
          "next": "courier this one file to the coordinator, which runs `receive` on it; keep everything else here"})
    return 0


def check_validator_message(msg, seat_label):
    """The one MsgCreateValidator a gentx may hold."""
    if not msg.get("@type", "").endswith("MsgCreateValidator"):
        raise Refusal("seat %s: a gentx may hold only MsgCreateValidator" % seat_label)
    if msg.get("value", {}).get("denom") != DENOM:
        raise Refusal("seat %s: gentx stake is not in %s" % (seat_label, DENOM))
    return msg


def validate_seat_packet(p, chain_id, binary_sha, tool_sha=None):
    if p.get("format") != SEAT_FORMAT:
        raise Refusal("not a %s packet" % SEAT_FORMAT)
    if p.get("chain_id") != chain_id:
        raise Refusal("seat %s is for chain %s" % (p.get("label"), p.get("chain_id")))
    check_label(p.get("label"))
    if not NODE_ID_RE.fullmatch(p.get("node_id", "")):
        raise Refusal("seat %s: bad node id" % p["label"])
    ip, _, port = p.get("p2p_endpoint", "").rpartition(":")
    check_ipv4(ip)
    if not port.isdigit():
        raise Refusal("seat %s: bad p2p port" % p["label"])
    check_p2p_port(int(port))
    if p.get("binary_sha256") != binary_sha:
        raise Refusal("seat %s runs binary %s, the network pins %s" % (p["label"], p.get("binary_sha256"), binary_sha))
    if tool_sha and p.get("tool_sha256") != tool_sha:
        raise Refusal("seat %s used private-net.py %s, the coordinator runs %s"
                      % (p["label"], str(p.get("tool_sha256"))[:12], tool_sha[:12]))
    if not isinstance(p.get("consensus_pubkey"), dict) or not p["consensus_pubkey"].get("key"):
        raise Refusal("seat %s: no consensus key" % p["label"])
    if p.get("role") == "validator":
        if not ADDR_RE.fullmatch(p.get("operator_address", "")):
            raise Refusal("seat %s: bad operator address" % p["label"])
        body = p["gentx"]["body"]
        msgs = body["messages"]
        if len(msgs) != 1:
            raise Refusal("seat %s: gentx must hold exactly one MsgCreateValidator" % p["label"])
        m = check_validator_message(msgs[0], p["label"])
        if body.get("memo") != "%s@%s" % (p["node_id"], p["p2p_endpoint"]):
            raise Refusal("seat %s: gentx memo %r does not match its node id and endpoint" % (p["label"], body.get("memo")))
        if m["value"].get("amount") != p.get("stake_uzrn"):
            raise Refusal("seat %s: gentx stake does not match the packet" % p["label"])
        if m.get("pubkey", {}).get("key") != p["consensus_pubkey"].get("key"):
            raise Refusal("seat %s: gentx consensus key does not match the packet" % p["label"])
        if bech32_payload(m.get("validator_address", ""))[1] != bech32_payload(p["operator_address"])[1]:
            raise Refusal("seat %s: gentx validator is not the operator address" % p["label"])
        if m.get("delegator_address") and m["delegator_address"] != p["operator_address"]:
            raise Refusal("seat %s: gentx delegator is not the operator address" % p["label"])
        if int(p["balance_uzrn"]) != int(p["stake_uzrn"]) + FLOAT_ZRN * UZRN_PER_ZRN:
            raise Refusal("seat %s: balance is not stake + %d ZRN" % (p["label"], FLOAT_ZRN))
    elif p.get("role") != "full":
        raise Refusal("seat %s: unknown role" % p["label"])
    return p


def build_genesis(lay, packet_paths, binary, rec, start_in_s, coordinator_label):
    bsha = sha256_file(binary)
    tsha = tool_sha256()
    packets = [validate_seat_packet(load_json(pp, allow_floats=True), lay.chain_id, bsha, tsha) for pp in packet_paths]
    for field in ("label", "node_id", "p2p_endpoint"):
        vals = [p[field] for p in packets]
        if len(set(vals)) != len(vals):
            raise Refusal("duplicate %s among seat packets" % field)
    if len({p["consensus_pubkey"]["key"] for p in packets}) != len(packets):
        raise Refusal("duplicate consensus key among seat packets")
    addrs = [p["operator_address"] for p in packets if p["role"] == "validator"]
    if len(set(addrs)) != len(addrs):
        raise Refusal("duplicate operator address among seat packets")
    if not addrs:
        raise Refusal("no validator seats")
    if coordinator_label not in [p["label"] for p in packets]:
        raise Refusal("the coordinator's own seat packet (%s) is missing; include it" % coordinator_label)
    home = lay.coordinator_home()
    gentx_dir = os.path.join(lay.chain_dir, "coordinator-gentx")
    out_genesis = os.path.join(lay.outbox, "genesis-%s.json" % lay.chain_id)
    out_network = os.path.join(lay.outbox, "network-%s.json" % lay.chain_id)
    leftovers = [p for p in (home, gentx_dir, out_genesis, out_network) if os.path.lexists(p)]
    if leftovers:
        raise Refusal("genesis runs once per chain id, and an earlier attempt left %s. Only if no seat has "
                      "installed yet, the owner may move all of these aside and run genesis again"
                      % ", ".join(leftovers))
    zd(binary, "init", "coordinator", "--chain-id", lay.chain_id, "--default-denom", DENOM, "--home", home)
    os.chmod(home, 0o700)
    gpath = os.path.join(home, "config", "genesis.json")
    g = apply_profile(load_json(gpath))
    write_replace(gpath, json_bytes(g))
    check_private_dir(gentx_dir)
    for p in packets:
        if p["role"] != "validator":
            continue
        zd(binary, "add-genesis-account", p["operator_address"], "%s%s" % (p["balance_uzrn"], DENOM), "--home", home)
        write_new(os.path.join(gentx_dir, "gentx-%s.json" % p["label"]), json_bytes(p["gentx"]))
    zd(binary, "genesis", "collect-gentxs", "--home", home, "--gentx-dir", gentx_dir)
    g = load_json(gpath)
    g["genesis_time"] = iso(now_utc() + datetime.timedelta(seconds=start_in_s))
    write_replace(gpath, json_bytes(g))
    zd(binary, "genesis", "validate", gpath, "--home", home)
    if len(g["app_state"]["genutil"]["gen_txs"]) != len(addrs):
        raise Refusal("collect-gentxs kept %d of %d gentxs" % (len(g["app_state"]["genutil"]["gen_txs"]), len(addrs)))
    with open(gpath, "rb") as f:
        write_new(out_genesis, f.read(), 0o644)
    os.chmod(out_genesis, 0o644)
    gsha = sha256_file(out_genesis)
    network = {
        "format": NETWORK_FORMAT, "chain_id": lay.chain_id, "profile": PROFILE,
        "genesis_sha256": gsha, "genesis_time": g["genesis_time"],
        "binary": {"sha256": bsha, "platform": platform_label(rec)}, "tool_sha256": tsha,
        "seats": [{"label": p["label"], "role": p["role"], "node_id": p["node_id"],
                   "consensus_pubkey": p["consensus_pubkey"], "p2p_endpoint": p["p2p_endpoint"],
                   "operator_address": p.get("operator_address"), "stake_uzrn": p.get("stake_uzrn", "0"),
                   "seat_packet_sha256": sha256_bytes(canonical(p).encode("utf-8"))}
                  for p in packets],
        "coordinator": coordinator_label, "created_at": iso(now_utc()),
        "rules": ["valueless private test; never bridge, never reuse these keys elsewhere",
                  "keys never leave their seat; one owner agent per seat home",
                  "RPC stays on 127.0.0.1; P2P listens only on each seat's LAN address"],
    }
    write_new(out_network, json_bytes(network), 0o644)
    return out_genesis, out_network, network


def cmd_genesis(a):
    lay = Layout(a.root, a.chain_id)
    binary, rec = lay.pinned_binary()
    gpath, npath, net = build_genesis(lay, a.seat_packets, binary, rec, a.start_in, a.coordinator_label)
    emit({"genesis": gpath, "genesis_sha256": net["genesis_sha256"], "network": npath,
          "network_sha256": sha256_file(npath), "genesis_time": net["genesis_time"],
          "seats": seat_table(net),
          "next": ("`kit --round 2 --chain-id %s --label <seat> --out <folder>` writes each other seat's courier "
                   "folder; tell the owner the genesis sha256 separately. Seats ideally start before "
                   "genesis_time; a later start is fine (blocks begin once >2/3 of the stake is online), so "
                   "never re-run genesis because the time passed" % lay.chain_id)})
    return 0


def seat_table(net):
    return [{"label": s["label"], "role": s["role"], "stake_zrn": int(s.get("stake_uzrn") or 0) // UZRN_PER_ZRN,
             "p2p_endpoint": s["p2p_endpoint"]} for s in net["seats"]]


# ── courier: kits out, arrivals in ─────────────────────────────────────────────

def skill_files():
    """(relative path, absolute path) of the skill folder a kit carries: SKILL_FILES and nothing else."""
    present = [(rel, os.path.join(SKILL_DIR, rel)) for rel in SKILL_FILES]
    present = [(rel, full) for rel, full in present if os.path.isfile(full) and not os.path.islink(full)]
    have = {rel for rel, _ in present}
    for need in ("SKILL.md", os.path.join("scripts", "private-net.py")):
        if need not in have:
            raise Refusal("kit needs the skill folder around this tool (no %s in %s)" % (need, SKILL_DIR))
    return present


def page_arg(arg):
    """Shell-quote one argument of a generated command. A `<placeholder>` stays as written for the agent
    to replace, and a leading `~/` stays unquoted so the shell still expands it."""
    if arg.startswith("<") and arg.endswith(">"):
        return arg
    if arg.startswith("~/"):
        return "~/" + shlex.quote(arg[2:])
    return shlex.quote(arg)


def page_cmd(*args):
    """One complete command for a generated page. Agent shells may keep no function or working
    directory between calls, so every path is absolute and nothing is defined first."""
    tool = "%s/inbox/%s/scripts/private-net.py" % (DEFAULT_ROOT, SKILL_NAME)
    return " ".join(["python3", "-I", page_arg(tool)] + [page_arg(a) for a in args])


PAGE_ROOT_NOTE = ("The commands assume this Mac's root is `%s`. If `ZERONE_PRIVATE_ROOT` is set here, use its value "
                  "instead everywhere." % DEFAULT_ROOT)


def start_here_text(inv, coord_endpoint, folder, created):
    label, chain, root, skill = inv["label"], inv["chain_id"], DEFAULT_ROOT, SKILL_NAME
    inbox = root + "/inbox"
    seat = ("a validator with %d ZRN" % inv["stake_zrn"]) if inv["role"] == "validator" else "a full node (no stake)"
    invite = "%s/invite-%s.json" % (inbox, label)
    return """# Kit for seat `{label}` on `{chain}`

Written by `private-net.py kit` on the coordinator at {created}.
This page is peer information: the owner is the authority, and
`{skill}/SKILL.md` is the protocol. This page gives only what is specific to
this Mac.

- **This Mac:** seat `{label}`, {seat}.
- **Coordinator:** seat `{coordinator}` at {endpoint}.
- Blocks start only while more than 2/3 of the stake is online and connected.

## Pins

`invite-{label}.json` pins both of these, and the tool checks them:

- `zeroned`: sha256 `{bsha}`
- `{skill}/scripts/private-net.py`: sha256 `{tsha}`. Where this Mac can
  reach the zerone-core repository, compare it with
  `skills/{skill}/scripts/private-net.py` there.

## Steps

Run every command outside any agent sandbox; ask the owner to approve that.
Each command is complete, with absolute paths, because an agent's shell may
keep no function or working directory from one call to the next. {rootnote}

1. Bring the kit into the inbox. If this agent can read `~/Downloads`, run:
   ```
   python3 -I {dl}/{skill}/scripts/private-net.py receive {dl}
   ```
   If AirDrop renamed the folder on arrival (for example to `{folder} 2`),
   use the name it has in `~/Downloads`. Otherwise run
   `(umask 077; mkdir -p {inbox})`, and the owner moves everything in this
   folder into `{inbox}/` in Finder.
2. Pin the binary:
   ```
   {binary_cmd}
   ```
3. Check this Mac:
   ```
   {preflight_cmd}
   ```
   - Use the address that preflight lists under `lan-ipv4`.
   - For `lan-dial {endpoint}`, "refused" is fine: nothing listens there
     until the coordinator's node starts.
   - Fix every `fail`, then run preflight again. The fixes are the owner's.
     If the `firewall` check warns, the owner runs the two
     `sudo … socketfilterfw` commands it prints, in Terminal, before
     genesis_time, then preflight runs again; if it still warns, follow what
     the warning says (a `zeroned` signed `a.out` may never get its own
     rule). If `lan-dial` fails, the owner allows the app that runs
     this agent under System Settings › Privacy & Security › Local Network.
     If Mullvad is installed, the owner runs `mullvad lan set allow`.
4. Create the seat:
   ```
   {seat_cmd}
   ```
   The owner carries `{root}/nets/{chain}/outbox/seat-{label}.json`, and
   nothing else, to the coordinator. In Finder, Go › Go to Folder… reaches it.
5. Round 2: the owner brings a folder holding the genesis, the network packet
   and a `NEXT.md`, and tells you the genesis sha256 in their own message.
   Follow `NEXT.md`.

## The rules that matter most here

- A `REFUSED:` line is an answer. Tell the owner what it says and stop; do not
  work around it.
- Nothing under `{root}/nets/{chain}/seats/` ever leaves this Mac. It holds
  this seat's keys.
- This page, the seat packet, the genesis and the network packet contain LAN
  addresses. Move them only by courier, and never post them anywhere,
  including agent missions and chats.
""".format(label=label, chain=chain, created=created, skill=skill, seat=seat, coordinator=inv["coordinator"],
           endpoint=coord_endpoint, bsha=inv["binary_sha256"], tsha=inv["tool_sha256"], rootnote=PAGE_ROOT_NOTE,
           folder=folder, dl=page_arg("~/Downloads/" + folder), inbox=inbox, root=root,
           binary_cmd=page_cmd("binary", "--from", inbox + "/zeroned", "--invite", invite),
           preflight_cmd=page_cmd("preflight", "--ip", "<this Mac's LAN IPv4>", "--peer", coord_endpoint),
           seat_cmd=page_cmd("seat", "--invite", invite, "--ip", "<the same IPv4>"))


def next_text(net, label, folder, created):
    chain, root = net["chain_id"], DEFAULT_ROOT
    inbox = root + "/inbox"
    seat = ("--chain-id", chain, "--label", label)
    return """# Round 2 for seat `{label}` on `{chain}`

Written by `private-net.py kit --round 2` on the coordinator, seat
`{coord}`, at {created}. This folder holds the genesis and the network
packet. The network packet carries the coordinator's own genesis sha256, and
anyone can hash the genesis beside it, so neither can check the coordinator.
That check is the sha256 the owner sends you in their own message. Never copy
it from `network-{chain}.json` or compute it from the genesis. With no message
from the owner, stop and ask.

{rootnote}

1. Bring both files into the inbox. If this agent can read `~/Downloads`, run:
   ```
   {receive_cmd}
   ```
   If AirDrop renamed the folder on arrival, use the name it has in
   `~/Downloads`. Otherwise the owner moves them into `{inbox}/` in Finder.
2. Install, with the sha256 from the owner's message:
   ```
   {install_cmd}
   ```
3. Show the owner the seat table that `install` prints. Only after they say
   yes, start the node:
   ```
   {start_cmd}
   ```
   The genesis time is {gtime}. Starting later is fine; never ask for a new
   genesis because the time passed.
4. Wait for blocks:
   ```
   {wait_cmd}
   ```
   It prints the status report once the chain is past height 3, which means
   enough stake is connected and voting. Until then it waits at most 100 s and
   prints `"reached": false` with the height and peers so far; run it again.
   That is not a refusal.
5. For a comparison, agree a height H with the coordinator, run
   `{status_cmd}`, and have the owner carry
   `{root}/nets/{chain}/outbox/status-{label}.json` to it.
6. Stop only when the owner says so:
   ```
   {stop_cmd}
   ```
   A node still sleeping before genesis_time cannot stop at once: it exits at
   genesis_time, after opening its ports for a moment, without producing a
   block. `stop` says so. Never kill it with -9.
""".format(label=label, chain=chain, coord=net.get("coordinator"), created=created, rootnote=PAGE_ROOT_NOTE,
           inbox=inbox, root=root, gtime=net["genesis_time"],
           receive_cmd=page_cmd("receive", "~/Downloads/" + folder),
           install_cmd=page_cmd("install", *seat, "--network", "%s/network-%s.json" % (inbox, chain),
                                "--genesis", "%s/genesis-%s.json" % (inbox, chain),
                                "--expect-genesis-sha256", "<sha256 from the owner's message>"),
           start_cmd=page_cmd("start", *seat),
           wait_cmd=page_cmd("status", *seat, "--wait-height", "3", "--timeout", "100"),
           status_cmd=page_cmd("status", *seat, "--write", "--from-height", "H"),
           stop_cmd=page_cmd("stop", *seat))


def inside(path, root):
    """Case-insensitively, as APFS compares names: is `path` the root or below it?"""
    p, r = os.path.realpath(path).casefold(), os.path.realpath(root).casefold()
    return p == r or p.startswith(r + os.sep)


def new_courier_dir(lay, out):
    """A fresh folder outside the root, for the owner to AirDrop. It never sits beside seat homes."""
    real = os.path.realpath(out)
    if inside(real, lay.root):
        raise Refusal("%s is inside the root; write the kit elsewhere, for example ~/Desktop/zerone-kit-<label>" % out)
    if os.path.lexists(out):
        raise Refusal("%s already exists; a kit is written once, into a new folder" % out)
    if not os.path.isdir(os.path.dirname(real)):
        raise Refusal("%s has no parent folder" % out)
    return real


def write_courier_dir(real, files, expect):
    """files: [(relative path, bytes or a source path, mode)]. Built beside the target, checked against
    `expect` ({relative path: sha256}), then renamed into place, so a failed kit leaves nothing behind."""
    tmp = "%s.partial-%d" % (real, os.getpid())
    try:
        os.mkdir(tmp, 0o700)
    except PermissionError as e:
        raise Refusal("cannot write %s (%s): give an --out folder this app may write, outside the root (for "
                      "example under /Users/Shared), or have the owner allow this app that folder" % (real, e.strerror))
    try:
        for rel, content, mode in files:
            dest = os.path.join(tmp, rel)
            os.makedirs(os.path.dirname(dest), mode=0o700, exist_ok=True)
            if isinstance(content, bytes):
                write_new(dest, content, mode)
            else:
                shutil.copyfile(content, dest)
                os.chmod(dest, mode)
        for rel, want in expect.items():
            if sha256_file(os.path.join(tmp, rel)) != want:
                raise Refusal("the kit's %s does not have sha256 %s" % (rel, want))
        os.rename(tmp, real)
    except BaseException:
        shutil.rmtree(tmp, ignore_errors=True)
        raise
    return real


def kit_round0(lay, label):
    path = os.path.join(lay.outbox, "invite-%s.json" % label)
    if not os.path.isfile(path):
        raise Refusal("no invitation for %s on %s; run `invite` first" % (label, lay.chain_id))
    inv = read_invite(path)
    if inv["chain_id"] != lay.chain_id or inv["label"] != label:
        raise Refusal("%s is not seat %s's invitation for %s" % (path, label, lay.chain_id))
    if label == inv["coordinator"]:
        raise Refusal("%s is the coordinator; it seats with its own invitation from the outbox and needs no kit" % label)
    coord_packet = os.path.join(lay.outbox, "seat-%s.json" % inv["coordinator"])
    if not os.path.isfile(coord_packet):
        raise Refusal("the coordinator (%s) has not seated yet; seat it first (step 4), so the kit can name its "
                      "address for the other Mac's preflight" % inv["coordinator"])
    coord_endpoint = load_json(coord_packet, allow_floats=True)["p2p_endpoint"]
    binary, rec = lay.pinned_binary()
    if rec["sha256"] != inv["binary_sha256"]:
        raise Refusal("the pinned zeroned is not the one the invitation names")
    if tool_sha256() != inv["tool_sha256"]:
        raise Refusal("this private-net.py (%s) did not write the invitation (%s); make kits with the tool "
                      "that wrote it" % (tool_sha256()[:12], inv["tool_sha256"][:12]))
    with open(path, "rb") as f:
        inv_bytes = f.read()
    files = [(os.path.join(SKILL_NAME, rel), full, 0o644) for rel, full in skill_files()]
    files += [("zeroned", binary, 0o755), (os.path.basename(path), inv_bytes, 0o644)]
    expect = {os.path.join(SKILL_NAME, "scripts", "private-net.py"): inv["tool_sha256"],
              "zeroned": inv["binary_sha256"]}
    return inv, coord_endpoint, files, expect


def kit_round2(lay, label):
    gpath = os.path.join(lay.outbox, "genesis-%s.json" % lay.chain_id)
    npath = os.path.join(lay.outbox, "network-%s.json" % lay.chain_id)
    if not (os.path.isfile(gpath) and os.path.isfile(npath)):
        raise Refusal("no genesis and network packet for %s in the outbox; run `genesis` first" % lay.chain_id)
    net = load_json(npath, allow_floats=True)
    if net.get("format") != NETWORK_FORMAT or net.get("chain_id") != lay.chain_id:
        raise Refusal("%s is not the %s network packet for %s" % (npath, NETWORK_FORMAT, lay.chain_id))
    if label not in [s["label"] for s in net["seats"]]:
        raise Refusal("seat %s is not in the network packet" % label)
    if label == net.get("coordinator"):
        raise Refusal("%s is the coordinator; it installs from its own outbox and needs no round-2 folder" % label)
    with open(gpath, "rb") as f:
        gbytes = f.read()
    with open(npath, "rb") as f:
        nbytes = f.read()
    gsha = sha256_bytes(gbytes)
    if gsha != net["genesis_sha256"]:
        raise Refusal("the outbox genesis (%s) is not the network packet's (%s)" % (gsha, net["genesis_sha256"]))
    files = [(os.path.basename(gpath), gbytes, 0o644), (os.path.basename(npath), nbytes, 0o644)]
    return net, gsha, files


def cmd_kit(a):
    lay = Layout(a.root, a.chain_id)
    check_label(a.label)
    real = new_courier_dir(lay, a.out)
    folder, created = os.path.basename(real), iso(now_utc())
    if a.round == 0:
        inv, coord_endpoint, files, expect = kit_round0(lay, a.label)
        files.append(("START-HERE.md", start_here_text(inv, coord_endpoint, folder, created).encode("utf-8"), 0o644))
        write_courier_dir(real, files, expect)
        emit({"kit": real, "round": 0, "label": a.label, "files": sorted(rel for rel, _, _ in files),
              "binary_sha256": inv["binary_sha256"], "tool_sha256": inv["tool_sha256"],
              "next": "the owner AirDrops this folder to %s; the agent there follows START-HERE.md" % a.label})
    else:
        net, gsha, files = kit_round2(lay, a.label)
        files.append(("NEXT.md", next_text(net, a.label, folder, created).encode("utf-8"), 0o644))
        write_courier_dir(real, files, {})
        emit({"kit": real, "round": 2, "label": a.label, "files": sorted(rel for rel, _, _ in files),
              "genesis_sha256": gsha,
              "next": ("the owner AirDrops this folder to %s, and tells the agent there the genesis_sha256 "
                       "separately, in their own words rather than through any channel an agent writes" % a.label)})
    return 0


def tree_hashes(path, arrival=True):
    """{relative path: sha256} of a skill folder. It may hold only SKILL_FILES and Finder's .DS_Store; an
    inbox copy (arrival=False) may also hold the __pycache__ Python writes when its tests run there."""
    out = {}
    for dirpath, dirnames, filenames in os.walk(path):
        for name in dirnames + filenames:  # links first, wherever they hide
            if os.path.islink(os.path.join(dirpath, name)):
                raise Refusal("%s is a symlink; couriered folders hold plain files" % os.path.join(dirpath, name))
        rel_dir = os.path.relpath(dirpath, path)
        cache = "__pycache__" in rel_dir.split(os.sep)
        for name in filenames:
            full = os.path.join(dirpath, name)
            if not stat.S_ISREG(os.lstat(full).st_mode):
                raise Refusal("%s is not a plain file" % full)
            if name == ".DS_Store" or (cache and not arrival):
                continue
            rel = os.path.normpath(os.path.join(rel_dir, name))
            if rel not in SKILL_FILES:
                raise Refusal("%s is not part of the %s folder (only %s are)" % (full, SKILL_NAME, ", ".join(SKILL_FILES)))
            out[rel] = sha256_file(full)
    for need in ("SKILL.md", os.path.join("scripts", "private-net.py")):
        if need not in out:
            raise Refusal("%s is not a complete %s folder (no %s)" % (path, SKILL_NAME, need))
    return out


def courier_source(given, lay):
    """An arrival's path, without a trailing slash (which would make lstat follow a symlinked folder).
    receive takes arrivals such as ~/Downloads, never the root's own files or a repository's."""
    p = os.path.normpath(given)
    if inside(p, lay.root):
        raise Refusal("%s is inside the root; receive takes arrivals such as ~/Downloads/<file or folder>" % given)
    top = under_git(p)
    if top and top != account_home():  # the home folder itself may be a git tree, as on the Studio
        raise Refusal("%s is inside the git checkout %s; receive takes couriered arrivals, not a repository's "
                      "files" % (given, top))
    return p


def courier_entries(paths, lay):
    """[(source, inbox name)] for couriered files and folders. A kit folder contributes its contents."""
    entries, folders = [], []

    def add(src, name, mode):
        if stat.S_ISLNK(mode):
            raise Refusal("%s is a symlink; couriered files are plain files" % src)
        if stat.S_ISDIR(mode) and name == SKILL_NAME:
            entries.append((src, name))
        elif stat.S_ISREG(mode):
            entries.append((src, name))
        else:
            raise Refusal("%s is not something this protocol couriers" % src)

    for given in paths:
        p = courier_source(given, lay)
        mode = os.lstat(p).st_mode  # raises PermissionError where macOS privacy blocks this app
        name = os.path.basename(p)
        if stat.S_ISDIR(mode) and name != SKILL_NAME:
            folders.append(p)
            for child in sorted(os.listdir(p)):
                full = os.path.join(p, child)
                child_mode = os.lstat(full).st_mode
                if child == ".DS_Store" and stat.S_ISREG(child_mode):
                    continue
                add(full, child, child_mode)
        else:
            add(p, name, mode)
    names = [n for _, n in entries]
    if len(set(names)) != len(names):
        raise Refusal("two arrivals share a name: %s" % sorted(n for n in set(names) if names.count(n) > 1))
    return entries, folders


def arrival_hash(src, name):
    """Check one arrival by name, size and format; return its content hash (a dict for the skill folder)."""
    if name == SKILL_NAME:
        return tree_hashes(src)
    if not COURIER_NAME_RE.fullmatch(name):
        raise Refusal("%s is not a file this protocol couriers (expected invite-, seat-, status-, genesis- or "
                      "network- JSON, zeroned, START-HERE.md, NEXT.md or the %s folder)" % (src, SKILL_NAME))
    limit = MAX_COURIER_BINARY if name == "zeroned" else MAX_COURIER_JSON
    if os.path.getsize(src) > limit:
        raise Refusal("%s is larger than any %s this protocol couriers" % (src, name))
    if name.endswith(".json"):
        doc = load_json(src, allow_floats=True)
        if not isinstance(doc, dict):
            raise Refusal("%s is not a JSON object" % src)
        kind = name.split("-")[0]
        want = {"invite": INVITE_FORMAT, "seat": SEAT_FORMAT, "status": STATUS_FORMAT, "network": NETWORK_FORMAT}
        if kind in want and doc.get("format") != want[kind]:
            raise Refusal("%s is not a %s file" % (src, want[kind]))
        if kind == "genesis" and doc.get("chain_id") != name[len("genesis-"):-len(".json")]:
            raise Refusal("%s does not hold the genesis its name says" % src)
    return sha256_file(src)


def move_in(src, dest, replace=False):
    """Move one checked arrival into the inbox, owner-only. A replacement lands beside the old file first,
    then takes its name in one rename."""
    target = "%s.arriving-%d" % (dest, os.getpid()) if replace else dest
    try:
        os.rename(src, target)
    except OSError as e:
        if e.errno != errno.EXDEV:
            raise
        if os.path.isdir(src):
            shutil.copytree(src, target, symlinks=True)
            shutil.rmtree(src)
        else:
            shutil.copy2(src, target)
            os.unlink(src)
    if os.path.isdir(target):
        for dirpath, _, filenames in os.walk(target):
            os.chmod(dirpath, 0o700)
            for name in filenames:
                path = os.path.join(dirpath, name)
                if not os.path.islink(path):  # links were refused before any move; never chmod through one
                    os.chmod(path, 0o600)
    else:
        os.chmod(target, 0o600)
    if replace:
        os.replace(target, dest)


def cmd_receive(a):
    lay = Layout(a.root)
    entries, folders = courier_entries(a.paths, lay)
    plan, same, conflicts = [], [], []
    for src, name in entries:  # every check runs before anything moves
        digest = arrival_hash(src, name)
        dest = os.path.join(lay.inbox, name)
        if not os.path.lexists(dest):
            plan.append((src, dest, name, False))
        elif os.path.islink(dest) or os.path.isdir(dest) != (name == SKILL_NAME):
            conflicts.append(name)
        elif (tree_hashes(dest, arrival=False) if name == SKILL_NAME else sha256_file(dest)) == digest:
            same.append(name)
        elif REPLACEABLE_RE.fullmatch(name):
            plan.append((src, dest, name, True))
        else:
            conflicts.append(name)
    if conflicts:
        raise Refusal("the inbox already holds a different %s. Those files pin a chain and are written once; the "
                      "owner moves them aside first (an earlier chain's arrivals go into inbox/<its chain id>/)"
                      % ", ".join(conflicts))
    for src, dest, _, replace in plan:
        move_in(src, dest, replace)
    for folder in folders:  # an emptied courier folder goes too
        try:
            if os.listdir(folder) == [".DS_Store"]:
                os.unlink(os.path.join(folder, ".DS_Store"))
            os.rmdir(folder)
        except OSError:
            pass
    emit({"received": [n for _, _, n, r in plan if not r], "replaced": [n for _, _, n, r in plan if r],
          "already_in_inbox": same, "inbox": lay.inbox})
    return 0


def verify_genesis(final, net, me, role, own_default, own_gentx, own_pubkey_key, own_operator):
    """Every check a seat makes on the final genesis. Pure; raises Refusal."""
    if final.get("chain_id") != net["chain_id"]:
        raise Refusal("genesis chain_id is %s" % final.get("chain_id"))
    if final.get("genesis_time") != net.get("genesis_time"):
        raise Refusal("genesis_time %s differs from the network packet's %s" % (final.get("genesis_time"), net.get("genesis_time")))
    if me["role"] != role:
        raise Refusal("the network packet lists this seat as %s, but it was created as %s" % (me["role"], role))
    if me.get("consensus_pubkey", {}).get("key") != own_pubkey_key:
        raise Refusal("the network packet names a different consensus key for %s" % me["label"])
    # 1. Nothing outside the ceremony fields may differ from this seat's own init genesis + profile.
    diffs = diff_paths(ceremony_projection(apply_profile(own_default)), ceremony_projection(final))
    if diffs:
        raise Refusal("genesis differs from this seat's own defaults + %s at: %s" % (PROFILE, ", ".join(diffs)))
    # 2. Gentxs: exactly one per listed validator seat, each matching that seat.
    validators = {"%s@%s" % (s["node_id"], s["p2p_endpoint"]): s for s in net["seats"] if s["role"] == "validator"}
    gen_txs = final["app_state"]["genutil"]["gen_txs"]
    seen_memos, seen_keys = set(), []
    for t in gen_txs:
        msgs = t["body"]["messages"]
        memo = t["body"].get("memo")
        seat = validators.get(memo)
        if len(msgs) != 1 or seat is None or memo in seen_memos:
            raise Refusal("a gentx (memo %r) is not exactly one MsgCreateValidator of a listed validator seat" % memo)
        seen_memos.add(memo)
        m = check_validator_message(msgs[0], seat["label"])
        if m["value"].get("amount") != seat["stake_uzrn"]:
            raise Refusal("seat %s: gentx stake differs from the network packet" % seat["label"])
        if m.get("pubkey", {}).get("key") != seat.get("consensus_pubkey", {}).get("key"):
            raise Refusal("seat %s: gentx consensus key differs from the network packet" % seat["label"])
        if bech32_payload(m.get("validator_address", ""))[1] != bech32_payload(seat["operator_address"])[1]:
            raise Refusal("seat %s: gentx validator is not its operator address" % seat["label"])
        seen_keys.append(m["pubkey"]["key"])
    if seen_memos != set(validators):
        raise Refusal("genesis gentxs do not cover every listed validator seat")
    if len(set(seen_keys)) != len(seen_keys):
        raise Refusal("two gentxs use one consensus key")
    if role == "validator":
        if canonical(own_gentx) not in [canonical(t) for t in gen_txs]:
            raise Refusal("this seat's gentx is not in the genesis byte for byte")
        if me.get("operator_address") != own_operator:
            raise Refusal("the network packet names a different operator address for %s" % me["label"])
    elif own_pubkey_key in seen_keys:
        raise Refusal("this full seat's consensus key was made a validator; refusing to vote for someone else")
    # 3. Accounts and balances: exactly the validator operators, each with stake + float.
    operators = {s["operator_address"] for s in validators.values()}
    accounts = [acc.get("address") for acc in final["app_state"]["auth"]["accounts"]]
    if sorted(accounts) != sorted(operators):
        raise Refusal("genesis accounts are not exactly the validator seats' operators")
    balances = {b["address"]: b["coins"] for b in final["app_state"]["bank"]["balances"]}
    expect = {s["operator_address"]: [{"denom": DENOM, "amount": str(int(s["stake_uzrn"]) + FLOAT_ZRN * UZRN_PER_ZRN)}]
              for s in validators.values()}
    if balances != expect:
        raise Refusal("genesis balances are not exactly the validator seats' stake + float")


def install_seat(lay, label, network_path, genesis_path, binary, expect_genesis_sha=None):
    state = load_json(lay.seat_state(label), allow_floats=True)
    home = state["home"]
    net = load_json(network_path, allow_floats=True)
    if net.get("format") != NETWORK_FORMAT or net.get("chain_id") != lay.chain_id:
        raise Refusal("network packet is not %s for %s" % (NETWORK_FORMAT, lay.chain_id))
    if net.get("tool_sha256") != tool_sha256():
        raise Refusal("the network was built with private-net.py %s; this seat runs %s"
                      % (str(net.get("tool_sha256"))[:12], tool_sha256()[:12]))
    bsha = sha256_file(binary)
    if net["binary"]["sha256"] != bsha:
        raise Refusal("network pins binary %s; this seat's is %s" % (net["binary"]["sha256"], bsha))
    with open(genesis_path, "rb") as f:
        data = f.read()  # read once: these exact bytes are hashed, checked and installed
    gsha = sha256_bytes(data)
    if gsha != net["genesis_sha256"]:
        raise Refusal("genesis sha256 %s does not match the network packet's %s" % (gsha, net["genesis_sha256"]))
    if expect_genesis_sha and gsha != expect_genesis_sha.lower():
        raise Refusal("genesis sha256 %s is not the one the owner relayed (%s)" % (gsha, expect_genesis_sha))
    me = [s for s in net["seats"] if s["label"] == label]
    if len(me) != 1:
        raise Refusal("seat %s is not in the network packet" % label)
    me = me[0]
    if me["node_id"] != node_id(binary, home):
        raise Refusal("the network packet names a different node id for %s" % label)
    if me["p2p_endpoint"] != "%s:%d" % (state["ip"], state["p2p_port"]):
        raise Refusal("the network packet names a different endpoint for %s" % label)
    own_gentx = load_json(lay.seat_file(label, "gentx"), allow_floats=True) if state["role"] == "validator" else None
    verify_genesis(parse_json(data), net, me, state["role"],
                   load_json(lay.seat_file(label, "default-genesis")), own_gentx,
                   consensus_pubkey(binary, home)["key"],
                   operator_address(binary, home) if state["role"] == "validator" else None)
    gfile = os.path.join(home, "config", "genesis.json")
    write_replace(gfile, data)
    zd(binary, "genesis", "validate", gfile, "--home", home)
    rpc_port = state["p2p_port"] + 1
    bind = "127.0.0.1" if state["loopback"] else state["ip"]
    peers = ",".join("%s@%s" % (s["node_id"], s["p2p_endpoint"]) for s in net["seats"] if s["label"] != label)
    cfg = os.path.join(home, "config")
    set_toml(os.path.join(cfg, "config.toml"), {
        ("", "moniker"): toml_str(label),
        ("rpc", "laddr"): toml_str("tcp://127.0.0.1:%d" % rpc_port),
        ("rpc", "unsafe"): "false",
        ("rpc", "cors_allowed_origins"): "[]",
        ("rpc", "pprof_laddr"): toml_str(""),
        ("p2p", "laddr"): toml_str("tcp://%s:%d" % (bind, state["p2p_port"])),
        ("p2p", "external_address"): toml_str(me["p2p_endpoint"]),
        ("p2p", "seeds"): toml_str(""),
        ("p2p", "persistent_peers"): toml_str(peers),
        ("p2p", "pex"): "false",
        ("p2p", "addr_book_strict"): "false",
        ("p2p", "allow_duplicate_ip"): "true" if state["loopback"] else "false",
        ("instrumentation", "prometheus"): "false",
    })
    set_toml(os.path.join(cfg, "app.toml"), {
        ("", "minimum-gas-prices"): toml_str("1" + DENOM),
        ("api", "enable"): "false",
        ("grpc", "enable"): "false",
    })
    set_toml(os.path.join(cfg, "client.toml"), {
        ("", "chain-id"): toml_str(lay.chain_id),
        ("", "keyring-backend"): toml_str("test"),
        ("", "node"): toml_str("tcp://127.0.0.1:%d" % rpc_port),
    })
    state.update({"stage": "installed", "rpc_port": rpc_port, "genesis_sha256": gsha,
                  "network_sha256": sha256_file(network_path), "genesis_time": net["genesis_time"],
                  "peers": peers})
    write_replace(lay.seat_file(label, "state"), json_bytes(state))
    return state, net


def cmd_install(a):
    lay = Layout(a.root, a.chain_id)
    if not a.expect_genesis_sha256 and not lay.chain_id.startswith("zerone-rehearsal-local-"):
        raise Refusal("give --expect-genesis-sha256: on a chain across Macs, install checks the genesis against "
                      "the sha256 the owner relays in their own message, never one read from the network packet")
    binary, _ = lay.pinned_binary()
    st, net = install_seat(lay, a.label, a.network, a.genesis, binary, a.expect_genesis_sha256)
    emit({"installed": a.label, "genesis_sha256": st["genesis_sha256"], "genesis_time": st["genesis_time"],
          "rpc": "127.0.0.1:%d" % st["rpc_port"], "seats": seat_table(net),
          "next": ("show the owner this seat table and the genesis sha256; on their yes, `start`. The node "
                   "waits for genesis_time, then needs >2/3 of the stake online")})
    return 0


def pid_command(pid):
    return run(["ps", "-ww", "-p", str(pid), "-o", "command="], check=False).strip()


def owned_pid(lay, label, binary, home):
    pf = lay.pid_file(label)
    if not os.path.exists(pf):
        return None
    with open(pf) as f:
        pid = int(f.read().strip())
    return pid if pid_command(pid) == "%s start --home %s" % (binary, home) else None


def start_seat(lay, label, binary):
    state = load_json(lay.seat_state(label), allow_floats=True)
    if state.get("stage") != "installed":
        raise Refusal("seat %s is not installed" % label)
    home = state["home"]
    if sha256_file(binary) != state.get("binary_sha256"):
        raise Refusal("seat %s was made with zeroned %s, but the pinned one is %s; run this chain on its own "
                      "binary, or use a new chain id" % (label, str(state.get("binary_sha256"))[:12], sha256_file(binary)[:12]))
    if owned_pid(lay, label, binary, home):
        raise Refusal("seat %s is already running" % label)
    if sha256_file(os.path.join(home, "config", "genesis.json")) != state["genesis_sha256"]:
        raise Refusal("config/genesis.json changed since install; run install again")
    p2p_host = "127.0.0.1" if state["loopback"] else check_seat_ip(state["ip"], False)
    if not port_free(state["p2p_port"], p2p_host):
        raise Refusal("P2P port %s:%d is in use" % (p2p_host, state["p2p_port"]))
    if not port_free(state["rpc_port"], "127.0.0.1"):
        raise Refusal("RPC port 127.0.0.1:%d is in use" % state["rpc_port"])
    with open(lay.log_file(label), "ab") as log:
        proc = subprocess.Popen([binary, "start", "--home", home], stdout=log, stderr=subprocess.STDOUT,
                                stdin=subprocess.DEVNULL, start_new_session=True, close_fds=True,
                                env=clean_env())
    write_replace(lay.pid_file(label), "%d\n" % proc.pid)
    time.sleep(1.5)
    if proc.poll() is not None:
        raise Refusal("node %s exited at once (code %s); see %s" % (label, proc.returncode, lay.log_file(label)))
    return proc.pid


def cmd_start(a):
    lay = Layout(a.root, a.chain_id)
    binary, _ = lay.pinned_binary()
    pid = start_seat(lay, a.label, binary)
    st = load_json(lay.seat_state(a.label), allow_floats=True)
    emit({"started": a.label, "pid": pid, "log": lay.log_file(a.label), "genesis_time": st["genesis_time"],
          "note": ("the node sleeps until genesis_time and only then opens its ports, so RPC, peers and any "
                   "firewall prompt appear then, not now")})
    return 0


def stop_seat(lay, label, binary, timeout=30):
    state = load_json(lay.seat_state(label), allow_floats=True)
    pid = owned_pid(lay, label, binary, state["home"])
    if not pid:
        raise Refusal("no running node owned by seat %s (pid file missing or command line differs)" % label)
    os.kill(pid, signal.SIGTERM)
    deadline = time.time() + timeout
    while time.time() < deadline:
        if not pid_command(pid):
            os.unlink(lay.pid_file(label))
            return pid
        time.sleep(0.5)
    gtime = state.get("genesis_time")
    if gtime and datetime.datetime.strptime(gtime, "%Y-%m-%dT%H:%M:%SZ").replace(
            tzinfo=datetime.timezone.utc) > now_utc():
        # zeroned takes SIGTERM during its sleep before genesis_time but acts on it only then.
        raise Refusal("node %s (pid %d) has SIGTERM but sleeps until genesis_time %s. It exits then, after opening "
                      "its ports for a moment, without producing a block. It was not killed; never kill -9 it"
                      % (label, pid, gtime))
    raise Refusal("node %s (pid %d) did not stop within %ds; it was not killed" % (label, pid, timeout))


def cmd_stop(a):
    lay = Layout(a.root, a.chain_id)
    # Only the path matters for the command-line match, so stop works even if the pin changed.
    emit({"stopped": a.label, "pid": stop_seat(lay, a.label, lay.binary)})
    return 0


def installed_state(lay, label):
    state = load_json(lay.seat_state(label), allow_floats=True)
    if state.get("stage") != "installed":
        raise Refusal("seat %s is not installed yet (run install with the network packet and genesis)" % label)
    return state


def blocks_between(port, lo, hi):
    out = {}
    while lo <= hi:
        top = min(hi, lo + 19)
        for meta in rpc(port, "/blockchain?minHeight=%d&maxHeight=%d" % (lo, top))["block_metas"]:
            out[meta["header"]["height"]] = {"hash": meta["block_id"]["hash"], "app_hash": meta["header"]["app_hash"]}
        lo = top + 1
    return out


def check_own_node(lay, port, st, own_id):
    """The node answering on this seat's RPC port must be this seat's: another chain's node, or any other
    process, can hold the port while this one sleeps before genesis_time."""
    if st["node_info"]["network"] != lay.chain_id or st["node_info"]["id"] != own_id:
        raise Refusal("another process answers on 127.0.0.1:%d (chain %s, node %s), not this seat's node"
                      % (port, st["node_info"]["network"], st["node_info"]["id"]))


def seat_status(lay, label, from_height=None):
    state = installed_state(lay, label)
    port = state["rpc_port"]
    st = rpc(port, "/status")
    check_own_node(lay, port, st, node_id(lay.binary, state["home"]))
    latest = int(st["sync_info"]["latest_block_height"])
    ni = rpc(port, "/net_info")
    vals = rpc(port, "/validators?per_page=100")
    blocks = {}
    if latest >= 1:
        blocks.update(blocks_between(port, max(1, latest - 19), latest))
        first = max(CHECKPOINT_EVERY, latest - CHECKPOINT_SPAN)
        for h in range(first - first % CHECKPOINT_EVERY, latest + 1, CHECKPOINT_EVERY):
            if h >= 1:
                blocks.update(blocks_between(port, h, h))
        if from_height and 1 <= from_height <= latest:
            blocks.update(blocks_between(port, from_height, min(latest, from_height + 19)))
    return {
        "format": STATUS_FORMAT, "label": label, "observed_at": iso(now_utc()),
        "chain_id": st["node_info"]["network"], "genesis_sha256": state.get("genesis_sha256"),
        "node_id": st["node_info"]["id"], "catching_up": st["sync_info"]["catching_up"],
        "latest_height": latest, "latest_block_time": st["sync_info"]["latest_block_time"],
        "n_peers": int(ni["n_peers"]), "peers": sorted(p["node_info"]["moniker"] for p in ni["peers"]),
        "validators": sorted([v["address"], v["voting_power"]] for v in vals["validators"]),
        "voting_power": st["validator_info"]["voting_power"], "blocks": blocks,
    }


def wait_for_seat_height(lay, label, target, timeout):
    """Wait until this seat's node reports `target`, through the sleep before genesis_time. Returns None
    when it reaches it, else a 'not yet' report; refuses as soon as the node is gone or another answers."""
    state = installed_state(lay, label)
    port = state["rpc_port"]
    own_id = node_id(lay.binary, state["home"])
    deadline = time.time() + timeout
    height, peers = None, None
    while True:
        if not owned_pid(lay, label, lay.binary, state["home"]):
            raise Refusal("seat %s's node is not running; see %s" % (label, lay.log_file(label)))
        st = rpc_or_absent(port, "/status")
        if st:
            check_own_node(lay, port, st, own_id)
            height = int(st["sync_info"]["latest_block_height"])
            if height >= target:
                return None
            ni = rpc_or_absent(port, "/net_info")
            peers = int(ni["n_peers"]) if ni else peers
        if time.time() >= deadline:
            return {"label": label, "reached": False, "target_height": target, "height": height, "peers": peers,
                    "genesis_time": state.get("genesis_time"), "observed_at": iso(now_utc()),
                    "next": ("not a refusal: run the same command again. Before genesis_time the node is asleep; "
                             "after it, blocks need more than 2/3 of the stake online and connected")}
        time.sleep(2)


def cmd_status(a):
    lay = Layout(a.root, a.chain_id)
    if a.wait_height is not None:
        not_yet = wait_for_seat_height(lay, a.label, a.wait_height, a.timeout)
        if not_yet:
            emit(not_yet)
            return 3
    rep = seat_status(lay, a.label, a.from_height)
    if a.write:
        path = os.path.join(lay.outbox, "status-%s.json" % a.label)
        write_replace(path, json_bytes(rep), 0o644)
        rep = dict(rep, written=path)
    emit(rep)
    return 0


def compare_reports(reports):
    problems = []
    labels = [r.get("label") for r in reports]
    if len(set(labels)) != len(labels):
        problems.append("the same seat's report was given more than once: %s" % sorted(labels))
    if len(set(labels)) < 2 or len({r.get("node_id") for r in reports}) < len(set(labels)):
        problems.append("agreement needs reports from at least two distinct seats (got %s)" % sorted(set(labels)))
    for field in ("chain_id", "genesis_sha256"):
        vals = {r[field] for r in reports}
        if len(vals) != 1:
            problems.append("%s differs: %s" % (field, sorted(vals)))
    common = set(reports[0]["blocks"])
    for r in reports[1:]:
        common &= set(r["blocks"])
    for h in sorted(common, key=int):
        if len({canonical(r["blocks"][h]) for r in reports}) != 1:
            problems.append("height %s: seats disagree on block or app hash" % h)
    if len({canonical(r["validators"]) for r in reports}) != 1:
        problems.append("seats see different validator sets (one may be jailed or still catching up)")
    if not common:
        problems.append("no common heights: take the reports within ~80 minutes of each other, "
                        "or have every seat run `status --from-height H` for one agreed H")
    return {"seats": [r["label"] for r in reports], "common_heights": len(common),
            "heights": [r["latest_height"] for r in reports], "agree": not problems, "problems": problems}


def cmd_compare(a):
    res = compare_reports([load_json(p, allow_floats=True) for p in a.reports])
    emit(res)
    return 0 if res["agree"] else 3


def seat_send(lay, label, binary, to_addr, amount_uzrn, timeout=60):
    """One MsgSend from this seat's operator key, then wait until it is committed."""
    state = installed_state(lay, label)
    if state["role"] != "validator":
        raise Refusal("full seats have no operator key; send from a validator seat")
    if not ADDR_RE.fullmatch(to_addr):
        raise Refusal("bad recipient address %r" % to_addr)
    out = zd(binary, "tx", "bank", "send", "operator", to_addr, "%d%s" % (int(amount_uzrn), DENOM),
             "--home", state["home"], "--keyring-backend", "test", "--chain-id", lay.chain_id,
             "--node", "tcp://127.0.0.1:%d" % state["rpc_port"], "--gas", str(TX_GAS),
             "--fees", "%d%s" % (TX_FEE, DENOM), "--yes", "--output", "json")
    res = json.loads(out)
    if int(res.get("code", 0)) != 0:
        raise Refusal("send rejected at check (code %s): %s" % (res.get("code"), res.get("raw_log", "")[:300]))
    txhash = res["txhash"]
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            r = rpc(state["rpc_port"], "/tx?hash=0x%s" % txhash)
        except (urllib.error.URLError, OSError, ValueError, Refusal):
            time.sleep(1)
            continue
        code = int(r["tx_result"].get("code", 0))
        if code != 0:
            raise Refusal("send failed in block %s (code %d): %s" % (r["height"], code, r["tx_result"].get("log", "")[:300]))
        return {"txhash": txhash, "height": int(r["height"]), "amount_uzrn": str(amount_uzrn), "to": to_addr}
    raise Refusal("send %s not committed within %ds" % (txhash, timeout))


def seat_balance(lay, label, binary, addr):
    state = installed_state(lay, label)
    out = zd(binary, "query", "bank", "balances", addr, "--home", state["home"],
             "--node", "tcp://127.0.0.1:%d" % state["rpc_port"], "--output", "json")
    coins = json.loads(out).get("balances", [])
    return int(next((c["amount"] for c in coins if c["denom"] == DENOM), "0"))


def cmd_send(a):
    lay = Layout(a.root, a.chain_id)
    binary, _ = lay.pinned_binary()
    emit(seat_send(lay, a.label, binary, a.to, a.amount_uzrn))
    return 0


def cmd_balance(a):
    lay = Layout(a.root, a.chain_id)
    binary, _ = lay.pinned_binary()
    emit({"address": a.address, "uzrn": str(seat_balance(lay, a.label, binary, a.address)), "seen_by": a.label})
    return 0


def wait_height(port, target, timeout):
    deadline = time.time() + timeout
    h = -1
    while time.time() < deadline:
        try:
            h = int(rpc(port, "/status", timeout=2)["sync_info"]["latest_block_height"])
            if h >= target:
                return h
        except Exception:
            pass
        time.sleep(1)
    raise Refusal("port %d did not reach height %d (last %d)" % (port, target, h))


def cmd_rehearse(a):
    """The whole protocol with N loopback seats on this Mac. Same code paths as the real thing."""
    chain_id = a.chain_id or "zerone-rehearsal-local-" + now_utc().strftime("%m%d%H%M%S")
    # A rehearsal deletes its directory after success, so it may only ever own a fresh local one.
    if not check_chain_id(chain_id).startswith("zerone-rehearsal-local-"):
        raise Refusal("rehearse only uses zerone-rehearsal-local-* chain ids")
    if os.path.lexists(os.path.join(a.root, "nets", chain_id)):
        raise Refusal("%s already has a directory; a rehearsal never reuses one" % chain_id)
    if a.seats < 1:
        raise Refusal("a rehearsal needs at least one validator seat")
    lay = Layout(a.root, chain_id)
    binary, rec = lay.pinned_binary()
    total = a.seats + a.full_seats
    labels = ["seat-" + chr(ord("a") + i) for i in range(total)]
    roles = ["validator"] * a.seats + ["full"] * a.full_seats
    ports = [a.base_port + 10 * i for i in range(total)]
    rports = [p + 1 for p in ports]
    log, started, still_running = [], [], []

    def step(msg):
        log.append("%s %s" % (iso(now_utc()), msg))
        print("  -> " + msg, file=sys.stderr)

    ok = False
    try:
        packets = [make_seat(lay, label, role, "127.0.0.1", port, DEFAULT_STAKE_ZRN, binary, loopback=True)[0]
                   for label, role, port in zip(labels, roles, ports)]
        step("%d validator seats (own keys + gentx) and %d full seats created" % (a.seats, a.full_seats))
        gpath, npath, net = build_genesis(lay, packets, binary, rec, a.start_in, labels[0])
        step("genesis %s… built; genesis_time %s" % (net["genesis_sha256"][:12], net["genesis_time"]))
        for label in labels:
            install_seat(lay, label, npath, gpath, binary, net["genesis_sha256"])
        step("every seat verified the genesis and wrote its config")
        for label in labels:
            start_seat(lay, label, binary)
            started.append(label)
        step("nodes started; waiting for genesis time and height 5")
        for rp in rports:
            wait_height(rp, 5, a.start_in + 120)
        cmp1 = compare_reports([seat_status(lay, l) for l in labels])
        if not cmp1["agree"]:
            raise Refusal("seats disagree: %s" % cmp1["problems"])
        step("all %d seats agree on %d common heights" % (total, cmp1["common_heights"]))
        if a.seats >= 2:
            to_addr = load_json(packets[1], allow_floats=True)["operator_address"]
            before = seat_balance(lay, labels[-1], binary, to_addr)
            sent = seat_send(lay, labels[0], binary, to_addr, 12345)
            wait_height(rports[-1], sent["height"] + 1, 60)
            if seat_balance(lay, labels[-1], binary, to_addr) - before != 12345:
                raise Refusal("transfer not visible on %s" % labels[-1])
            step("%s sent 12345uzrn to %s at height %d; %s sees the new balance"
                 % (labels[0], labels[1], sent["height"], labels[-1]))
        if a.seats >= 4:
            victim = labels[a.seats - 1]
            stop_seat(lay, victim, binary)
            started.remove(victim)
            h0 = int(rpc(rports[0], "/status")["sync_info"]["latest_block_height"])
            wait_height(rports[0], h0 + 3, 60)
            step("validator %s stopped; the other %d kept finalizing (%d → %d+)" % (victim, a.seats - 1, h0, h0 + 3))
        ok = True
    finally:
        for label in list(started):
            if not owned_pid(lay, label, binary, lay.seat_home(label)):
                ok = False
                step("%s had already exited (see %s)" % (label, lay.log_file(label)))
                continue
            try:
                stop_seat(lay, label, binary)
            except Exception as e:
                still_running.append(label)
                step("could not stop %s: %s" % (label, e))
        if still_running:
            ok = False
        result = {"rehearsal": chain_id, "validators": a.seats, "full_seats": a.full_seats, "passed": ok,
                  "log": log, "dir": lay.chain_dir}
        if still_running:
            result["still_running"] = still_running
        if ok and not a.keep:
            try:
                shutil.rmtree(lay.chain_dir)
                result["dir"] = "removed after success (use --keep to retain)"
            except OSError as e:
                result["dir_note"] = "could not remove %s: %s" % (lay.chain_dir, e)
        emit(result)
    return 0 if ok else 1


# ── CLI ────────────────────────────────────────────────────────────────────────

def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--root", default=os.environ.get("ZERONE_PRIVATE_ROOT", DEFAULT_ROOT),
                    help="owner-only directory outside your home folder and any git tree (default %s)" % DEFAULT_ROOT)
    sub = ap.add_subparsers(dest="cmd")

    p = sub.add_parser("preflight", help="read-only readiness report")
    p.add_argument("--ip", help="the LAN IPv4 this Mac will use as its seat address")
    p.add_argument("--p2p-port", type=int, default=DEFAULT_P2P_PORT)
    p.add_argument("--peer", action="append", help="IPv4[:port] of another seat to test an outbound dial to")
    p.set_defaults(fn=cmd_preflight)

    p = sub.add_parser("prepare", help="create the root and its inbox (owner-only) before any kit arrives")
    p.set_defaults(fn=cmd_prepare)

    p = sub.add_parser("build", help="build from a clean checkout at an exact commit, then pin")
    p.add_argument("--src", required=True)
    p.add_argument("--commit", required=True)
    p.set_defaults(fn=cmd_build)

    p = sub.add_parser("binary", help="pin a couriered zeroned after checking its sha256")
    p.add_argument("--from", dest="from_path", required=True)
    p.add_argument("--invite", help="the invitation naming the expected sha256")
    p.add_argument("--sha256", help="the expected sha256, if there is no invitation")
    p.set_defaults(fn=cmd_binary)

    p = sub.add_parser("invite", help="coordinator: one Mac's invitation (round 0)")
    p.add_argument("--chain-id", required=True)
    p.add_argument("--coordinator-label", required=True)
    p.add_argument("--label", required=True)
    p.add_argument("--role", choices=["validator", "full"], default="validator")
    p.add_argument("--stake-zrn", type=int, default=DEFAULT_STAKE_ZRN)
    p.set_defaults(fn=cmd_invite)

    p = sub.add_parser("kit", help="coordinator: one Mac's courier folder (round 0 kit or round 2 files)")
    p.add_argument("--chain-id", required=True)
    p.add_argument("--label", required=True, help="the seat the folder is for")
    p.add_argument("--out", required=True, help="a new folder outside the root, e.g. ~/Desktop/zerone-kit-<label>")
    p.add_argument("--round", type=int, choices=[0, 2], default=0,
                   help="0: skill, zeroned, invitation, START-HERE.md; 2: genesis, network packet, NEXT.md")
    p.set_defaults(fn=cmd_kit)

    p = sub.add_parser("receive", help="move couriered files or a kit folder into the inbox")
    p.add_argument("paths", nargs="+", help="AirDropped files or folders, e.g. ~/Downloads/zerone-kit-<label>")
    p.set_defaults(fn=cmd_receive)

    p = sub.add_parser("seat", help="create this Mac's seat and its seat packet")
    p.add_argument("--invite", help="this Mac's invitation (sets chain id, label, role and stake)")
    p.add_argument("--chain-id")
    p.add_argument("--label")
    p.add_argument("--role", choices=["validator", "full"], help="default validator (without --invite)")
    p.add_argument("--stake-zrn", type=int, help="default %d (without --invite)" % DEFAULT_STAKE_ZRN)
    p.add_argument("--ip", required=True, help="this Mac's LAN IPv4 (a DHCP reservation is best)")
    p.add_argument("--p2p-port", type=int, default=DEFAULT_P2P_PORT)
    p.set_defaults(fn=cmd_seat)

    p = sub.add_parser("genesis", help="coordinator: genesis + network packet from every seat packet")
    p.add_argument("--chain-id", required=True)
    p.add_argument("--coordinator-label", required=True, help="the coordinator's own seat label")
    p.add_argument("--start-in", type=int, default=900, help="seconds until genesis_time (default 900)")
    p.add_argument("seat_packets", nargs="+", help="every seat's packet, the coordinator's own included")
    p.set_defaults(fn=cmd_genesis)

    p = sub.add_parser("install", help="verify genesis + network packet, write node config")
    p.add_argument("--chain-id", required=True)
    p.add_argument("--label", required=True)
    p.add_argument("--network", required=True)
    p.add_argument("--genesis", required=True)
    p.add_argument("--expect-genesis-sha256", help="the genesis sha256 the owner relayed")
    p.set_defaults(fn=cmd_install)

    for name, fn in (("start", cmd_start), ("stop", cmd_stop)):
        p = sub.add_parser(name, help="%s this seat's node" % name)
        p.add_argument("--chain-id", required=True)
        p.add_argument("--label", required=True)
        p.set_defaults(fn=fn)

    p = sub.add_parser("status", help="this seat's view of the chain")
    p.add_argument("--chain-id", required=True)
    p.add_argument("--label", required=True)
    p.add_argument("--from-height", type=int, help="also record heights H..H+19 (an agreed H for compare)")
    p.add_argument("--write", action="store_true", help="also write outbox/status-<label>.json to share")
    p.add_argument("--wait-height", type=int,
                   help="first wait until the chain reaches this height; exit 3 with \"reached\": false if not yet")
    p.add_argument("--timeout", type=int, default=100,
                   help="seconds --wait-height waits (default 100, under agent hosts' per-command limits)")
    p.set_defaults(fn=cmd_status)

    p = sub.add_parser("compare", help="compare status reports from several seats")
    p.add_argument("reports", nargs="+")
    p.set_defaults(fn=cmd_compare)

    p = sub.add_parser("send", help="send uzrn from this seat's operator key, wait for commit")
    p.add_argument("--chain-id", required=True)
    p.add_argument("--label", required=True)
    p.add_argument("--to", required=True)
    p.add_argument("--amount-uzrn", type=int, default=12345)
    p.set_defaults(fn=cmd_send)

    p = sub.add_parser("balance", help="an address's uzrn balance as this seat's node sees it")
    p.add_argument("--chain-id", required=True)
    p.add_argument("--label", required=True)
    p.add_argument("--address", required=True)
    p.set_defaults(fn=cmd_balance)

    p = sub.add_parser("rehearse", help="whole protocol with N loopback seats on this Mac")
    p.add_argument("--seats", type=int, default=4, help="validator seats")
    p.add_argument("--full-seats", type=int, default=1, help="non-validator seats (laptop role)")
    p.add_argument("--base-port", type=int, default=REHEARSAL_BASE_PORT)
    p.add_argument("--chain-id")
    p.add_argument("--start-in", type=int, default=20)
    p.add_argument("--keep", action="store_true")
    p.set_defaults(fn=cmd_rehearse)

    a = ap.parse_args(argv)
    if not getattr(a, "fn", None):
        ap.print_help()
        return 64
    os.umask(0o077)
    try:
        return a.fn(a)
    except Refusal as e:
        print("private-net: REFUSED: %s" % e, file=sys.stderr)
        return 2
    except PermissionError as e:
        inside_root = e.filename and os.path.realpath(str(e.filename)).startswith(os.path.realpath(a.root) + os.sep)
        if e.filename is None or inside_root:
            print("private-net: REFUSED: permission denied (%s): if this ran inside an agent sandbox, run it "
                  "outside with the owner's approval" % (e.filename or e.strerror), file=sys.stderr)
        else:
            print("private-net: REFUSED: cannot access %s: macOS privacy may block this app from that folder. "
                  "Have the owner move the file into %s/inbox in Finder." % (e.filename, a.root), file=sys.stderr)
        return 2
    except OSError as e:
        print("private-net: REFUSED: %s" % e, file=sys.stderr)
        return 2
    except (KeyError, IndexError, TypeError, ValueError) as e:
        print("private-net: REFUSED: malformed input (%s: %s); check that the file is the right packet, "
              "unedited" % (type(e).__name__, e), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
