"""Tests for private-net.py.

Pure checks always run. The full loopback rehearsal runs only when
ZERONE_PRIVATE_NET_TEST_BINARY names a built zeroned (CI passes the consensus
rehearsal's binary). ZERONE_PRIVATE_NET_EVIDENCE_PARENT, when set, receives the
rehearsal's node logs and public packets (never seat homes or keys).
"""
import contextlib
import copy
import glob
import hashlib
import importlib.util
import io
import json
import os
import random
import shutil
import socket
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("private-net.py")
spec = importlib.util.spec_from_file_location("private_net", SCRIPT)
pn = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pn)

CHAIN = "zerone-rehearsal-lan-9"


def private_tmp(case):
    d = tempfile.mkdtemp(prefix="private-net-test-")
    os.chmod(d, 0o700)
    case.addCleanup(shutil.rmtree, d, True)
    return d


def bech32_encode(hrp, payload):
    """Test-side encoder, so the tool's decoder is checked against an independent writer."""
    acc, bits, data = 0, 0, []
    for b in payload:
        acc = (acc << 8) | b
        bits += 8
        while bits >= 5:
            bits -= 5
            data.append((acc >> bits) & 31)
    if bits:
        data.append((acc << (5 - bits)) & 31)
    values = [ord(c) >> 5 for c in hrp] + [0] + [ord(c) & 31 for c in hrp] + data + [0] * 6
    chk = 1
    for v in values:
        top = chk >> 25
        chk = (chk & 0x1ffffff) << 5 ^ v
        for i, g in enumerate((0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3)):
            chk ^= g if (top >> i) & 1 else 0
    chk ^= 1
    return hrp + "1" + "".join(pn._B32[d] for d in data + [(chk >> 5 * (5 - i)) & 31 for i in range(6)])


def keypair(n):
    raw = hashlib.sha256(b"seat-%d" % n).digest()[:20]
    return bech32_encode("zrn", raw), bech32_encode("zrnvaloper", raw)


class Guards(unittest.TestCase):
    def test_chain_ids(self):
        for ok in ("zerone-rehearsal-lan-1", "zerone-rehearsal-private-a2", "zerone-rehearsal-local-1017105613"):
            self.assertEqual(pn.check_chain_id(ok), ok)
        for bad in ("zerone-1", "zerone-testnet-1", "zerone-dev-1", "zerone-2", "zerone-rehearsal-1",
                    "zerone-consensus-rehearsal-1", "zerone-localnet", "zerone-rehearsal-lan-",
                    "zerone-rehearsal-lan-" + "x" * 25, "Zerone-rehearsal-lan-1", "zerone-rehearsal-lan-1\n", ""):
            with self.assertRaises(pn.Refusal, msg=repr(bad)):
                pn.check_chain_id(bad)

    def test_labels_and_ips(self):
        pn.check_label("studio")
        for bad in ("a", "1seat", "Seat", "seat_a", "s" * 25, "seat\n"):
            with self.assertRaises(pn.Refusal):
                pn.check_label(bad)
        pn.check_ipv4("192.0.2.10")
        self.assertEqual(pn.check_p2p_port(36656), 36656)
        for bad in (65535, 65536, 80, 0, "36656"):
            with self.assertRaises(pn.Refusal):
                pn.check_p2p_port(bad)
        for bad in ("studio.local", "::1", "1.2.3", "300.1.1.1", "192.0.2.10:36656", "01.2.3.4"):
            with self.assertRaises(pn.Refusal):
                pn.check_ipv4(bad)

    def test_seat_ip_must_belong_to_this_machine(self):
        self.assertEqual(pn.check_seat_ip("127.0.0.1", loopback=True), "127.0.0.1")
        for bad in ("127.0.0.1", "0.0.0.0", "192.0.2.10"):  # 192.0.2.0/24 is TEST-NET-1, never local
            with self.assertRaises(pn.Refusal):
                pn.check_seat_ip(bad, loopback=False)

    def test_private_dir_refuses_git_trees_open_modes_and_home(self):
        base = private_tmp(self)
        repo = os.path.join(base, "repo")
        os.makedirs(repo)
        subprocess.run(["git", "init", "-q", repo], check=True)
        with self.assertRaises(pn.Refusal):
            pn.check_private_dir(os.path.join(repo, "root"))
        open_dir = os.path.join(base, "open")
        os.makedirs(open_dir)
        os.chmod(open_dir, 0o755)
        with self.assertRaises(pn.Refusal):
            pn.check_private_dir(open_dir)
        made = pn.check_private_dir(os.path.join(base, "fresh"))
        self.assertEqual(os.stat(made).st_mode & 0o777, 0o700)
        home = pn.account_home()
        for inside in (os.path.join(home, "zerone-private"), os.path.join(home.upper(), "desktop", "x")):
            with self.assertRaises(pn.Refusal):
                pn.check_private_dir(inside, create=False)

    def test_zeroned_environment_is_stripped(self):
        os.environ["ZERONED_PRIV_VALIDATOR_KEY_FILE"] = "/elsewhere/key.json"
        os.environ["zeroned_rpc_laddr"] = "tcp://0.0.0.0:1"
        self.addCleanup(os.environ.pop, "ZERONED_PRIV_VALIDATOR_KEY_FILE", None)
        self.addCleanup(os.environ.pop, "zeroned_rpc_laddr", None)
        env = pn.clean_env({"GOTOOLCHAIN": "go1.25.14"})
        self.assertFalse([k for k in env if k.upper().startswith("ZERONED_")])
        self.assertEqual(env["GOTOOLCHAIN"], "go1.25.14")
        out = pn.run([sys.executable, "-c", "import os; print(sorted(k for k in os.environ if k.upper().startswith('ZERONED')))"])
        self.assertEqual(out.strip(), "[]")

    def test_port_probe_ignores_time_wait(self):
        server = socket.socket()
        server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server.bind(("127.0.0.1", 0))
        port = server.getsockname()[1]
        server.listen(1)
        self.assertFalse(pn.port_free(port, "127.0.0.1"))  # a live listener is busy
        client = socket.create_connection(("127.0.0.1", port))
        conn, _ = server.accept()
        conn.close()  # the server side closes first and holds TIME_WAIT on `port`
        client.close()
        server.close()
        self.assertTrue(pn.port_free(port, "127.0.0.1"))


class Pinning(unittest.TestCase):
    def test_half_pinned_binary_is_refused(self):
        root = os.path.join(private_tmp(self), "root")
        lay = pn.Layout(root)
        os.makedirs(lay.bin_dir, mode=0o700)
        with open(lay.binary, "wb") as f:
            f.write(b"old")
        src = os.path.join(root, "inbox", "zeroned")
        with open(src, "wb") as f:
            f.write(b"new")
        with self.assertRaises(pn.Refusal):
            pn.pin_binary(lay, src, hashlib.sha256(b"new").hexdigest())

    @unittest.skipUnless(shutil.which("sandbox-exec"), "macOS sandbox-exec not available")
    def test_sandboxed_bind_names_the_sandbox(self):
        code = ("import importlib.util,sys; s=importlib.util.spec_from_file_location('p', sys.argv[1]); "
                "m=importlib.util.module_from_spec(s); s.loader.exec_module(m)\n"
                "try:\n m.port_free(46999, '127.0.0.1')\nexcept m.Refusal as e:\n print(e)")
        out = subprocess.run(["sandbox-exec", "-p", "(version 1)(allow default)(deny network-bind)",
                              sys.executable, "-c", code, str(SCRIPT)], capture_output=True, text=True)
        self.assertIn("agent sandbox", out.stdout + out.stderr)


class Toml(unittest.TestCase):
    def test_section_aware_and_strict(self):
        path = os.path.join(private_tmp(self), "config.toml")
        with open(path, "w") as f:
            f.write('moniker = "x"\n[rpc]\nladdr = "tcp://0.0.0.0:26657"\n[p2p]\nladdr = "tcp://0.0.0.0:26656"\npex = true\n')
        pn.set_toml(path, {("rpc", "laddr"): '"tcp://127.0.0.1:1"', ("p2p", "pex"): "false"})
        with open(path) as f:
            text = f.read()
        self.assertIn('[rpc]\nladdr = "tcp://127.0.0.1:1"', text)
        self.assertIn('[p2p]\nladdr = "tcp://0.0.0.0:26656"\npex = false', text)
        with self.assertRaises(pn.Refusal):
            pn.set_toml(path, {("p2p", "no_such_key"): "1"})


class Bech32(unittest.TestCase):
    def test_operator_and_validator_share_a_payload(self):
        acc, val = keypair(1)
        self.assertEqual(pn.bech32_payload(acc)[1], pn.bech32_payload(val)[1])
        self.assertTrue(pn.ADDR_RE.fullmatch(acc))
        broken = acc[:-1] + ("q" if acc[-1] != "q" else "p")
        with self.assertRaises(pn.Refusal):
            pn.bech32_payload(broken)


def seat(n, role="validator", stake_zrn=1000):
    acc, val = keypair(n)
    node = hashlib.sha1(b"node-%d" % n).hexdigest()
    endpoint = "192.0.2.%d:36656" % (10 + n)
    pub = {"@type": "/cosmos.crypto.ed25519.PubKey", "key": "KEY%d=" % n}
    p = {"format": pn.SEAT_FORMAT, "chain_id": CHAIN, "label": "seat-%d" % n, "role": role, "node_id": node,
         "consensus_pubkey": pub, "p2p_endpoint": endpoint, "binary_sha256": "f" * 64, "tool_sha256": "t" * 64}
    if role == "validator":
        stake = str(stake_zrn * pn.UZRN_PER_ZRN)
        p.update({"operator_address": acc, "stake_uzrn": stake,
                  "balance_uzrn": str((stake_zrn + pn.FLOAT_ZRN) * pn.UZRN_PER_ZRN),
                  "gentx": {"body": {"memo": "%s@%s" % (node, endpoint), "messages": [{
                      "@type": "/cosmos.staking.v1beta1.MsgCreateValidator", "delegator_address": "",
                      "validator_address": val, "pubkey": dict(pub),
                      "value": {"denom": "uzrn", "amount": stake}}]}}})
    return p


def default_genesis():
    return {"genesis_time": "2026-10-17T00:00:00Z", "chain_id": CHAIN, "initial_height": 1,
            "consensus": {"params": {"block": {"max_gas": "33333333"}}},
            "app_state": {"auth": {"accounts": []}, "bank": {"balances": [], "supply": [], "params": {"x": 1}},
                          "genutil": {"gen_txs": []},
                          "slashing": {"params": {"signed_blocks_window": "100", "min_signed_per_window": "0.5",
                                                  "downtime_jail_duration": "600s", "slash_fraction_downtime": "0.01"}}}}


def ceremony(seats):
    g = pn.apply_profile(default_genesis())
    g["genesis_time"] = "2026-10-17T01:00:00Z"
    for p in seats:
        if p["role"] == "validator":
            g["app_state"]["auth"]["accounts"].append({"@type": "/cosmos.auth.v1beta1.BaseAccount",
                                                       "address": p["operator_address"]})
            g["app_state"]["bank"]["balances"].append(
                {"address": p["operator_address"], "coins": [{"denom": "uzrn", "amount": p["balance_uzrn"]}]})
            g["app_state"]["genutil"]["gen_txs"].append(copy.deepcopy(p["gentx"]))
    net = {"format": pn.NETWORK_FORMAT, "chain_id": CHAIN, "genesis_time": g["genesis_time"],
           "seats": [{k: p.get(k) for k in ("label", "role", "node_id", "consensus_pubkey", "p2p_endpoint",
                                            "operator_address", "stake_uzrn")} for p in seats]}
    for s in net["seats"]:
        s["stake_uzrn"] = s["stake_uzrn"] or "0"
    return g, net


class SeatPackets(unittest.TestCase):
    def test_valid_packet_passes(self):
        pn.validate_seat_packet(seat(1), CHAIN, "f" * 64, "t" * 64)
        pn.validate_seat_packet(seat(2, "full"), CHAIN, "f" * 64, "t" * 64)

    def test_tampering_is_refused(self):
        def bad(mutate):
            p = seat(1)
            mutate(p)
            with self.assertRaises(pn.Refusal):
                pn.validate_seat_packet(p, CHAIN, "f" * 64, "t" * 64)
        bad(lambda p: p.update(chain_id="zerone-1"))
        bad(lambda p: p.update(binary_sha256="e" * 64))
        bad(lambda p: p.update(tool_sha256="e" * 64))
        bad(lambda p: p.update(p2p_endpoint="studio.local:36656"))
        bad(lambda p: p.update(p2p_endpoint="192.0.2.11:65535"))
        bad(lambda p: p["gentx"]["body"].update(memo="cd" * 20 + "@192.0.2.11:36656"))
        bad(lambda p: p["gentx"]["body"]["messages"][0]["value"].update(amount="5"))
        bad(lambda p: p["gentx"]["body"]["messages"][0]["pubkey"].update(key="OTHER="))
        bad(lambda p: p["gentx"]["body"]["messages"][0].update(validator_address=keypair(7)[1]))
        bad(lambda p: p.update(balance_uzrn="999999999999"))
        bad(lambda p: p["gentx"]["body"]["messages"].append({"@type": "/cosmos.bank.v1beta1.MsgSend"}))


class VerifyGenesis(unittest.TestCase):
    """What a seat's install refuses, on a synthetic three-seat ceremony."""

    def setUp(self):
        self.seats = [seat(1), seat(2), seat(3, "full")]
        self.final, self.net = ceremony(self.seats)

    def verify(self, final=None, net=None, who=0, role=None):
        net = net or self.net
        p = self.seats[who]
        me = [s for s in net["seats"] if s["label"] == p["label"]][0]
        pn.verify_genesis(final or self.final, net, me, role or p["role"], default_genesis(),
                          p.get("gentx"), p["consensus_pubkey"]["key"], p.get("operator_address"))

    def refused(self, mutate, who=0):
        final, net = copy.deepcopy(self.final), copy.deepcopy(self.net)
        mutate(final, net)
        with self.assertRaises(pn.Refusal):
            self.verify(final, net, who)

    def test_honest_ceremony_passes_for_every_seat(self):
        for who in range(3):
            self.verify(who=who)

    def test_parameter_change_is_refused(self):
        self.refused(lambda g, n: g["app_state"]["bank"]["params"].update(x=2))
        self.refused(lambda g, n: g["consensus"]["params"]["block"].update(max_gas="1"))

    def test_extra_account_or_balance_is_refused(self):
        extra = keypair(9)[0]
        self.refused(lambda g, n: g["app_state"]["auth"]["accounts"].append({"address": extra}))
        self.refused(lambda g, n: g["app_state"]["bank"]["balances"][0]["coins"][0].update(amount="1"))

    def test_stake_inflated_in_packet_is_refused(self):
        def inflate(g, n):
            n["seats"][1]["stake_uzrn"] = str(10 ** 12)
            g["app_state"]["bank"]["balances"][1]["coins"][0]["amount"] = str(10 ** 12 + pn.FLOAT_ZRN * pn.UZRN_PER_ZRN)
        self.refused(inflate)

    def test_full_seat_key_made_a_validator_is_refused(self):
        def hijack(g, n):  # seat-2's gentx now carries the laptop's (seat-3's) consensus key
            g["app_state"]["genutil"]["gen_txs"][1]["body"]["messages"][0]["pubkey"]["key"] = "KEY3="
            n["seats"][1]["consensus_pubkey"] = {"key": "KEY3="}
        self.refused(hijack, who=2)

    def test_role_mismatch_dropped_gentx_and_duplicates_are_refused(self):
        with self.assertRaises(pn.Refusal):
            self.verify(who=2, role="validator")
        self.refused(lambda g, n: g["app_state"]["genutil"]["gen_txs"].pop(1))
        self.refused(lambda g, n: g["app_state"]["genutil"]["gen_txs"].append(copy.deepcopy(g["app_state"]["genutil"]["gen_txs"][1])))
        self.refused(lambda g, n: g.update(chain_id="zerone-1"))

    def test_genesis_time_must_be_the_network_packets(self):
        self.refused(lambda g, n: g.update(genesis_time="2026-10-18T00:00:00Z"))


class Compare(unittest.TestCase):
    def report(self, label, h3="H3", heights=("2", "3")):
        blocks = {h: {"hash": ("H%s" % h) if h != "3" else h3, "app_hash": "A%s" % h} for h in heights}
        return {"label": label, "node_id": "n-" + label, "chain_id": CHAIN, "genesis_sha256": "g", "latest_height": 3,
                "validators": [["V", "1"]], "blocks": blocks}

    def test_agree_disagree_and_no_overlap(self):
        self.assertTrue(pn.compare_reports([self.report("a"), self.report("b")])["agree"])
        res = pn.compare_reports([self.report("a"), self.report("b", h3="FORK")])
        self.assertFalse(res["agree"])
        self.assertIn("height 3", res["problems"][0])
        res = pn.compare_reports([self.report("a", heights=("2",)), self.report("b", heights=("3",))])
        self.assertFalse(res["agree"])
        self.assertIn("no common heights", res["problems"][0])

    def test_one_seat_is_not_agreement(self):
        for reports in ([self.report("a")], [self.report("a"), self.report("a")]):
            res = pn.compare_reports(reports)
            self.assertFalse(res["agree"])
            self.assertTrue(any("two distinct seats" in p for p in res["problems"]), res["problems"])


class Floats(unittest.TestCase):
    def test_float_literals_are_refused(self):
        with self.assertRaises(pn.Refusal):
            pn.parse_json('{"a": 0.5}')


# A stand-in zeroned: enough for pinning, invitations, kits and arrivals, which never start a node.
FAKE_ZEROND = b"#!/bin/sh\necho '{\"version\": \"test\", \"go\": \"go version go1.25.14 darwin/arm64\"}'\n"


def cli(*argv):
    out, err = io.StringIO(), io.StringIO()
    with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
        rc = pn.main([str(x) for x in argv])
    try:
        doc = json.loads(out.getvalue())
    except ValueError:
        doc = out.getvalue()
    return rc, doc, err.getvalue()


def file_sha(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


class Courier(unittest.TestCase):
    """Kits out of the coordinator and arrivals into an inbox, as couriered between Macs."""

    INBOX_TOOL = "python3 -I /Users/Shared/zerone-private/inbox/run-a-private-zerone-net/scripts/private-net.py"

    def setUp(self):
        self.base = private_tmp(self)
        self.root = os.path.join(self.base, "root")
        self.dl = os.path.join(self.base, "Downloads")
        os.mkdir(self.dl)
        fake = os.path.join(self.base, "fake-zeroned")
        with open(fake, "wb") as f:
            f.write(FAKE_ZEROND)
        os.chmod(fake, 0o700)
        rc, _, err = cli("--root", self.root, "binary", "--from", fake, "--sha256", file_sha(fake))
        self.assertEqual(rc, 0, err)
        for label in ("alpha", "beta"):
            rc, _, err = cli("--root", self.root, "invite", "--chain-id", CHAIN, "--coordinator-label", "alpha",
                             "--label", label)
            self.assertEqual(rc, 0, err)
        self.outbox = os.path.join(self.root, "nets", CHAIN, "outbox")
        # The coordinator has seated: kits name its address.
        self.coord_packet = self.write(os.path.join(self.outbox, "seat-alpha.json"),
                                       json.dumps({"format": pn.SEAT_FORMAT, "p2p_endpoint": "192.0.2.10:36656"}))
        self.root2 = os.path.join(self.base, "root2")

    def kit(self, label="beta", out=None, *extra):
        out = out or os.path.join(self.dl, "zerone-kit-" + label)
        return out, cli("--root", self.root, "kit", "--chain-id", CHAIN, "--label", label, "--out", out, *extra)

    def receive(self, *paths):
        return cli("--root", self.root2, "receive", *paths)

    def write(self, path, data):
        with open(path, "wb") as f:
            f.write(data if isinstance(data, bytes) else data.encode("utf-8"))
        return path

    def test_kit_carries_exactly_the_pinned_files_and_a_start_page(self):
        out, (rc, doc, err) = self.kit()
        self.assertEqual(rc, 0, err)
        self.assertEqual(sorted(os.listdir(out)), ["START-HERE.md", "invite-beta.json", pn.SKILL_NAME, "zeroned"])
        self.assertEqual(os.listdir(self.dl), ["zerone-kit-beta"])  # nothing half-built beside it
        with open(os.path.join(out, "invite-beta.json")) as f:
            inv = json.load(f)
        self.assertEqual(file_sha(os.path.join(out, "zeroned")), inv["binary_sha256"])
        self.assertEqual(file_sha(os.path.join(out, pn.SKILL_NAME, "scripts", "private-net.py")), inv["tool_sha256"])
        carried = sorted(os.path.relpath(os.path.join(d, n), os.path.join(out, pn.SKILL_NAME))
                         for d, _, names in os.walk(os.path.join(out, pn.SKILL_NAME)) for n in names)
        self.assertEqual(carried, sorted(pn.SKILL_FILES))
        with open(os.path.join(out, "START-HERE.md")) as f:
            page = f.read()
        inbox = "/Users/Shared/zerone-private/inbox"
        for text in ("seat `beta`", CHAIN, "at 192.0.2.10:36656", inv["tool_sha256"],
                     "python3 -I ~/Downloads/zerone-kit-beta/%s/scripts/private-net.py receive ~/Downloads/zerone-kit-beta"
                     % pn.SKILL_NAME,
                     "%s binary --from %s/zeroned --invite %s/invite-beta.json" % (self.INBOX_TOOL, inbox, inbox),
                     "--peer 192.0.2.10:36656", "If the `firewall` check warns", "This page, the seat packet"):
            self.assertIn(text, page)

    def test_kit_leaves_a_stray_module_behind(self):
        skill = os.path.join(self.base, "skill-copy", pn.SKILL_NAME)
        shutil.copytree(pn.SKILL_DIR, skill, ignore=shutil.ignore_patterns("__pycache__"))
        self.write(os.path.join(skill, "scripts", "hashlib.py"), "print('SHADOW')\n")
        self.addCleanup(setattr, pn, "SKILL_DIR", pn.SKILL_DIR)
        pn.SKILL_DIR = skill
        out, (rc, _, err) = self.kit()
        self.assertEqual(rc, 0, err)
        self.assertFalse(os.path.exists(os.path.join(out, pn.SKILL_NAME, "scripts", "hashlib.py")))

    def test_a_module_beside_the_tool_cannot_shadow_the_standard_library(self):
        scripts = os.path.join(self.base, "planted", "scripts")
        os.makedirs(scripts)
        shutil.copy(SCRIPT, scripts)
        self.write(os.path.join(scripts, "hashlib.py"), "print('SHADOW')\nfrom _sha256 import *\n")
        out = subprocess.run([sys.executable, "-B", os.path.join(scripts, "private-net.py"), "--help"],
                             capture_output=True, text=True)
        self.assertEqual(out.returncode, 0, out.stderr)
        self.assertNotIn("SHADOW", out.stdout + out.stderr)

    def test_kit_refusals_leave_nothing_behind(self):
        for label, out, extra, text in (
                ("alpha", None, (), "is the coordinator"),
                ("gamma", None, (), "no invitation for gamma"),
                ("beta", os.path.join(self.root, "kit"), (), "inside the root"),
                ("beta", os.path.join(self.root.upper(), "kit"), (), "inside the root"),
                ("beta", None, ("--round", "2"), "run `genesis` first")):
            _, (rc, _, err) = self.kit(label, out, *extra)
            self.assertEqual(rc, 2, label)
            self.assertIn(text, err)
        os.unlink(self.coord_packet)
        _, (rc, _, err) = self.kit()
        self.assertEqual(rc, 2)
        self.assertIn("has not seated yet", err)
        self.assertEqual(os.listdir(self.dl), [])
        self.write(self.coord_packet, json.dumps({"format": pn.SEAT_FORMAT, "p2p_endpoint": "192.0.2.10:36656"}))
        _, (rc, _, _) = self.kit()
        self.assertEqual(rc, 0)
        _, (rc, _, err) = self.kit()
        self.assertEqual(rc, 2)
        self.assertIn("already exists", err)
        locked = os.path.join(self.base, "locked")
        os.mkdir(locked, 0o500)
        self.addCleanup(os.chmod, locked, 0o700)
        if os.geteuid() != 0:
            _, (rc, _, err) = self.kit("beta", os.path.join(locked, "kit"))
            self.assertEqual(rc, 2)
            self.assertIn("give an --out folder this app may write", err)

    def test_receive_moves_a_kit_into_another_inbox(self):
        out, (rc, _, err) = self.kit()
        self.assertEqual(rc, 0, err)
        again = out + "-again"
        shutil.copytree(out, again)
        rc, doc, err = self.receive(out + "/")  # a trailing slash, as tab completion writes it
        self.assertEqual(rc, 0, err)
        self.assertEqual(sorted(doc["received"]), ["START-HERE.md", "invite-beta.json", pn.SKILL_NAME, "zeroned"])
        self.assertFalse(os.path.exists(out))  # the emptied courier folder goes too
        inbox = os.path.join(self.root2, "inbox")
        self.assertEqual(os.stat(os.path.join(inbox, "zeroned")).st_mode & 0o777, 0o600)
        self.assertEqual(os.stat(os.path.join(inbox, pn.SKILL_NAME)).st_mode & 0o777, 0o700)
        # The same kit again is left where it is.
        rc, doc, err = self.receive(again)
        self.assertEqual((rc, doc["received"], len(doc["already_in_inbox"])), (0, [], 4), err)
        # A re-taken report or a re-made page replaces the old one; files that pin the chain never do.
        status = {"format": pn.STATUS_FORMAT, "label": "beta", "latest_height": 1}
        self.write(os.path.join(self.dl, "status-beta.json"), json.dumps(status))
        self.assertEqual(self.receive(os.path.join(self.dl, "status-beta.json"))[0], 0)
        self.write(os.path.join(self.dl, "status-beta.json"), json.dumps(dict(status, latest_height=2)))
        rc, doc, err = self.receive(os.path.join(self.dl, "status-beta.json"))
        self.assertEqual((rc, doc["replaced"]), (0, ["status-beta.json"]), err)
        with open(os.path.join(inbox, "status-beta.json")) as f:
            self.assertEqual(json.load(f)["latest_height"], 2)
        self.write(os.path.join(self.dl, "invite-beta.json"),
                   json.dumps({"format": pn.INVITE_FORMAT, "chain_id": "zerone-rehearsal-lan-8"}))
        self.write(os.path.join(self.dl, "zeroned"), b"other")
        rc, _, err = self.receive(os.path.join(self.dl, "invite-beta.json"), os.path.join(self.dl, "zeroned"))
        self.assertEqual(rc, 2)
        self.assertIn("different invite-beta.json, zeroned", err)  # every conflict, named at once

    def test_receive_refuses_anything_unexpected_before_moving_anything(self):
        good = self.write(os.path.join(self.dl, "seat-ok.json"), json.dumps({"format": pn.SEAT_FORMAT}))
        notes = self.write(os.path.join(self.dl, "notes.json"), "{}")
        bad_invite = self.write(os.path.join(self.dl, "invite-x1.json"), json.dumps({"format": "other"}))
        wrong_genesis = self.write(os.path.join(self.dl, "genesis-%s.json" % CHAIN), json.dumps({"chain_id": "zerone-1"}))
        link = os.path.join(self.dl, "seat-link.json")
        os.symlink(good, link)
        folder = os.path.join(self.dl, "zerone-kit-odd")
        os.mkdir(folder)
        self.write(os.path.join(folder, "evil.sh"), "#!/bin/sh\n")
        half_skill = os.path.join(self.dl, "half", pn.SKILL_NAME)
        os.makedirs(half_skill)
        self.write(os.path.join(half_skill, "SKILL.md"), "#\n")
        out, (rc, _, _) = self.kit()
        stray = os.path.join(out, pn.SKILL_NAME, "scripts", "hashlib.py")
        self.write(stray, "")
        hidden_out, _ = self.kit("beta", os.path.join(self.dl, "zerone-kit-hidden"))
        victim = self.write(os.path.join(self.base, "victim"), "x")
        os.chmod(victim, 0o755)
        os.makedirs(os.path.join(hidden_out, pn.SKILL_NAME, "scripts", "__pycache__"))
        os.symlink(victim, os.path.join(hidden_out, pn.SKILL_NAME, "scripts", "__pycache__", "x.pyc"))
        linked_folder = os.path.join(self.dl, "zerone-kit-linked")
        os.symlink(folder, linked_folder)
        repo = os.path.join(self.base, "repo")
        os.makedirs(repo)
        subprocess.run(["git", "init", "-q", repo], check=True)
        in_repo = self.write(os.path.join(repo, "seat-r1.json"), json.dumps({"format": pn.SEAT_FORMAT}))
        pn.Layout(self.root2)
        in_root = self.write(os.path.join(self.root2, "seat-in.json"), json.dumps({"format": pn.SEAT_FORMAT}))
        for path, text in ((notes, "not a file this protocol couriers"),
                           (bad_invite, "is not a %s file" % pn.INVITE_FORMAT),
                           (wrong_genesis, "does not hold the genesis its name says"), (link, "symlink"),
                           (folder, "evil.sh is not a file this protocol couriers"), (half_skill, "not a complete"),
                           (out, "hashlib.py is not part of"), (hidden_out, "symlink"),
                           (linked_folder + "/", "symlink"), (in_root, "inside the root"),
                           (in_repo, "inside the git checkout")):
            rc, _, err = self.receive(good, path)
            self.assertEqual(rc, 2, path)
            self.assertIn(text, err)
        self.assertEqual(os.listdir(os.path.join(self.root2, "inbox")), [])
        self.assertTrue(os.path.exists(good) and os.path.exists(in_root))
        self.assertEqual(os.stat(victim).st_mode & 0o777, 0o755)  # never chmodded through a link

    @unittest.skipIf(os.geteuid() == 0, "root ignores directory permissions")
    def test_unreadable_downloads_send_the_owner_to_finder(self):
        locked = os.path.join(self.dl, "locked")
        os.mkdir(locked)
        arrival = self.write(os.path.join(locked, "seat-z1.json"), json.dumps({"format": pn.SEAT_FORMAT}))
        os.chmod(locked, 0)
        self.addCleanup(os.chmod, locked, 0o700)
        rc, _, err = self.receive(arrival)
        self.assertEqual(rc, 2)
        self.assertIn("in Finder", err)

    def test_round2_folder_tells_the_seat_where_the_sha256_must_come_from(self):
        genesis = self.write(os.path.join(self.outbox, "genesis-%s.json" % CHAIN),
                             json.dumps({"chain_id": CHAIN, "genesis_time": "2026-10-17T14:36:12Z"}))
        gsha = file_sha(genesis)
        seats = [{"label": label, "role": "validator"} for label in ("alpha", "beta")]
        self.write(os.path.join(self.outbox, "network-%s.json" % CHAIN), json.dumps({
            "format": pn.NETWORK_FORMAT, "chain_id": CHAIN, "genesis_sha256": gsha,
            "genesis_time": "2026-10-17T14:36:12Z", "seats": seats, "coordinator": "alpha"}))
        out = os.path.join(self.dl, "zerone-round2-beta")
        _, (rc, doc, err) = self.kit("beta", out, "--round", "2")
        self.assertEqual(rc, 0, err)
        self.assertEqual(doc["genesis_sha256"], gsha)
        self.assertEqual(sorted(os.listdir(out)), ["NEXT.md", "genesis-%s.json" % CHAIN, "network-%s.json" % CHAIN])
        with open(os.path.join(out, "NEXT.md")) as f:
            page = f.read()
        self.assertNotIn(gsha, page)
        for text in ("Never copy\nit from `network-%s.json`" % CHAIN, "stop and ask",
                     "--expect-genesis-sha256 <sha256 from the owner's message>",
                     "%s status --chain-id %s --label beta --wait-height 3 --timeout 100" % (self.INBOX_TOOL, CHAIN),
                     "Never kill it with -9"):
            self.assertIn(text, page)
        _, (rc, _, err) = self.kit("alpha", os.path.join(self.dl, "r2-alpha"), "--round", "2")
        self.assertIn("is the coordinator", err)
        self.write(genesis, json.dumps({"chain_id": CHAIN, "genesis_time": "2026-10-18T00:00:00Z"}))
        _, (rc, _, err) = self.kit("beta", os.path.join(self.dl, "r2-tampered"), "--round", "2")
        self.assertEqual(rc, 2)
        self.assertIn("is not the network packet's", err)

    def test_install_across_macs_needs_the_relayed_sha256(self):
        rc, _, err = cli("--root", self.root, "install", "--chain-id", CHAIN, "--label", "beta",
                         "--network", "n.json", "--genesis", "g.json")
        self.assertEqual(rc, 2)
        self.assertIn("give --expect-genesis-sha256", err)

    def test_loopback_seats_only_on_local_chains(self):
        rc, _, err = cli("--root", self.root, "seat", "--chain-id", CHAIN, "--label", "gamma", "--ip", "127.0.0.1")
        self.assertEqual(rc, 2)
        self.assertIn("not a LAN address", err)

    def test_status_refuses_a_node_that_is_not_this_seats(self):
        lay = pn.Layout(self.root, CHAIN)
        st = {"node_info": {"network": CHAIN, "id": "a" * 40}}
        pn.check_own_node(lay, 1, st, "a" * 40)
        for other in ({"network": "zerone-rehearsal-lan-8", "id": "a" * 40}, {"network": CHAIN, "id": "b" * 40}):
            with self.assertRaises(pn.Refusal):
                pn.check_own_node(lay, 1, {"node_info": other}, "a" * 40)


    def test_page_commands_quote_odd_folder_names(self):
        self.assertEqual(pn.page_cmd("receive", "~/Downloads/kit 2", "<placeholder kept>", "a;b"),
                         "%s receive ~/'Downloads/kit 2' <placeholder kept> 'a;b'" % self.INBOX_TOOL)
        out, (rc, _, err) = self.kit("beta", os.path.join(self.dl, "zerone-kit beta"))
        self.assertEqual(rc, 0, err)
        with open(os.path.join(out, "START-HERE.md")) as f:
            page = f.read()
        self.assertIn("python3 -I ~/'Downloads/zerone-kit beta'/%s/scripts/private-net.py receive "
                      "~/'Downloads/zerone-kit beta'" % pn.SKILL_NAME, page)


class RpcProbe(unittest.TestCase):
    """What --wait-height makes of whatever holds the seat's RPC port before genesis_time."""

    def serve(self, status, body):
        import http.server
        import threading

        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(status)
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *args):
                pass

        server = http.server.HTTPServer(("127.0.0.1", 0), Handler)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        return server.server_address[1]

    def test_no_listener_is_not_yet(self):
        s = socket.socket()
        s.bind(("127.0.0.1", 0))
        port = s.getsockname()[1]
        s.close()
        self.assertIsNone(pn.rpc_or_absent(port, "/status"))

    def test_a_responder_that_is_not_cometbft_is_refused(self):
        for status, body in ((404, b"not here"), (200, b"<html>"), (200, b'{"jsonrpc": "2.0"}')):
            port = self.serve(status, body)
            with self.assertRaises(pn.Refusal, msg=(status, body)) as caught:
                pn.rpc_or_absent(port, "/status")
            self.assertIn("another process answers", str(caught.exception))
        port = self.serve(200, b'{"result": {"ok": true}}')
        self.assertEqual(pn.rpc_or_absent(port, "/status"), {"ok": True})

def keep_evidence(*roots):
    parent = os.environ.get("ZERONE_PRIVATE_NET_EVIDENCE_PARENT")
    if not parent:
        return
    os.makedirs(parent, exist_ok=True)
    dest = tempfile.mkdtemp(prefix="private-net-", dir=parent)
    for root in roots:
        for f in glob.glob(os.path.join(root, "nets", "*", "run", "*.log")) + \
                glob.glob(os.path.join(root, "nets", "*", "outbox", "*.json")):
            shutil.copy(f, os.path.join(dest, os.path.basename(f)))


@unittest.skipUnless(os.environ.get("ZERONE_PRIVATE_NET_TEST_BINARY"), "set ZERONE_PRIVATE_NET_TEST_BINARY to run")
class Rehearsal(unittest.TestCase):
    def test_four_validators_and_a_full_seat_on_loopback(self):
        binary = os.environ["ZERONE_PRIVATE_NET_TEST_BINARY"]
        root = os.path.join(private_tmp(self), "root")
        self.addCleanup(keep_evidence, root)  # runs before the tmp cleanup
        base = random.Random(os.getpid()).randrange(20000, 30000, 100)  # below both OSes' ephemeral ranges
        cli = [sys.executable, "-B", str(SCRIPT), "--root", root]
        with open(binary, "rb") as f:
            sha = hashlib.sha256(f.read()).hexdigest()
        pin = subprocess.run(cli + ["binary", "--from", binary, "--sha256", sha], capture_output=True, text=True)
        self.assertEqual(pin.returncode, 0, pin.stderr)
        run = subprocess.run(cli + ["rehearse", "--seats", "4", "--full-seats", "1", "--base-port", str(base), "--keep"],
                             capture_output=True, text=True, timeout=600)
        self.assertEqual(run.returncode, 0, run.stdout[-3000:] + run.stderr[-3000:])
        result = json.loads(run.stdout)
        self.assertTrue(result["passed"])
        self.assertTrue(any("sees the new balance" in line for line in result["log"]))
        self.assertTrue(any("kept finalizing" in line for line in result["log"]))
        seats = Path(result["dir"], "seats")
        keys = sorted(seats.glob("*/config/priv_validator_key.json")) + sorted(seats.glob("*/config/node_key.json"))
        self.assertEqual(len(keys), 10, keys)
        for key in keys:
            self.assertEqual(key.stat().st_mode & 0o077, 0, key)
        for home in seats.iterdir():
            if home.is_dir():
                self.assertEqual(home.stat().st_mode & 0o777, 0o700, home)


@unittest.skipUnless(os.environ.get("ZERONE_PRIVATE_NET_TEST_BINARY"), "set ZERONE_PRIVATE_NET_TEST_BINARY to run")
class TwoRootCourier(unittest.TestCase):
    """The courier protocol between two independent roots, as between two Macs, on loopback: the second
    seat runs the kit's own copy of the tool, and every file crosses through a stand-in Downloads folder."""

    def run_tool(self, tool, root, *args):
        r = subprocess.run([sys.executable, "-B", str(tool), "--root", root] + [str(x) for x in args],
                           capture_output=True, text=True, timeout=300)
        self.assertEqual(r.returncode, 0, "%s: %s%s" % (args[0], r.stdout[-2000:], r.stderr[-2000:]))
        return json.loads(r.stdout)

    def test_two_roots_kit_receive_genesis_install_agree_and_transfer(self):
        binary = os.environ["ZERONE_PRIVATE_NET_TEST_BINARY"]
        base = private_tmp(self)
        a, b, dl = (os.path.join(base, n) for n in ("alpha-root", "beta-root", "Downloads"))
        os.mkdir(dl)
        self.addCleanup(keep_evidence, a, b)
        chain = "zerone-rehearsal-local-courier-%d" % (os.getpid() % 100000)
        port = random.Random(os.getpid()).randrange(20000, 30000, 100) + 50  # clear of the rehearsal's ports
        a_out = os.path.join(a, "nets", chain, "outbox")
        b_out = os.path.join(b, "nets", chain, "outbox")
        b_in = os.path.join(b, "inbox")

        self.run_tool(SCRIPT, a, "binary", "--from", binary, "--sha256", file_sha(binary))
        for label in ("alpha", "beta"):
            self.run_tool(SCRIPT, a, "invite", "--chain-id", chain, "--coordinator-label", "alpha", "--label", label)
        self.run_tool(SCRIPT, a, "seat", "--invite", os.path.join(a_out, "invite-alpha.json"),
                      "--ip", "127.0.0.1", "--p2p-port", port)
        kit = os.path.join(dl, "zerone-kit-beta")
        self.run_tool(SCRIPT, a, "kit", "--chain-id", chain, "--label", "beta", "--out", kit)

        # Round 0 arrives on the second "Mac", which from here on runs only the kit's copy of the tool.
        self.run_tool(os.path.join(kit, pn.SKILL_NAME, "scripts", "private-net.py"), b, "receive", kit)
        tool_b = os.path.join(b_in, pn.SKILL_NAME, "scripts", "private-net.py")
        self.run_tool(tool_b, b, "binary", "--from", os.path.join(b_in, "zeroned"),
                      "--invite", os.path.join(b_in, "invite-beta.json"))
        self.run_tool(tool_b, b, "seat", "--invite", os.path.join(b_in, "invite-beta.json"),
                      "--ip", "127.0.0.1", "--p2p-port", port + 10)
        shutil.copy(os.path.join(b_out, "seat-beta.json"), dl)
        self.run_tool(SCRIPT, a, "receive", os.path.join(dl, "seat-beta.json"))

        g = self.run_tool(SCRIPT, a, "genesis", "--chain-id", chain, "--coordinator-label", "alpha", "--start-in", 15,
                          os.path.join(a_out, "seat-alpha.json"), os.path.join(a, "inbox", "seat-beta.json"))
        round2 = os.path.join(dl, "zerone-round2-beta")
        self.run_tool(SCRIPT, a, "kit", "--round", 2, "--chain-id", chain, "--label", "beta", "--out", round2)
        self.run_tool(tool_b, b, "receive", round2)
        for tool, root, label, src in ((tool_b, b, "beta", b_in), (SCRIPT, a, "alpha", a_out)):
            self.run_tool(tool, root, "install", "--chain-id", chain, "--label", label,
                          "--network", os.path.join(src, "network-%s.json" % chain),
                          "--genesis", os.path.join(src, "genesis-%s.json" % chain),
                          "--expect-genesis-sha256", g["genesis_sha256"])

        seats = ((SCRIPT, a, "alpha"), (tool_b, b, "beta"))
        for tool, root, label in seats:
            self.addCleanup(subprocess.run, [sys.executable, "-B", str(tool), "--root", root, "stop",
                                             "--chain-id", chain, "--label", label], capture_output=True)
            self.run_tool(tool, root, "start", "--chain-id", chain, "--label", label)
        not_yet = subprocess.run([sys.executable, "-B", str(tool_b), "--root", b, "status", "--chain-id", chain,
                                  "--label", "beta", "--wait-height", str(10 ** 6), "--timeout", "3"],
                                 capture_output=True, text=True, timeout=60)
        self.assertEqual(not_yet.returncode, 3, not_yet.stderr)  # "not yet" is an answer, not a refusal
        self.assertFalse(json.loads(not_yet.stdout)["reached"])
        reports = {}
        for tool, root, label in seats:
            reports[label] = self.run_tool(tool, root, "status", "--chain-id", chain, "--label", label,
                                           "--wait-height", 3, "--timeout", 180, "--write", "--from-height", 1)
        self.assertEqual((reports["alpha"]["peers"], reports["beta"]["peers"]), (["beta"], ["alpha"]))
        shutil.copy(os.path.join(b_out, "status-beta.json"), dl)
        self.run_tool(SCRIPT, a, "receive", os.path.join(dl, "status-beta.json"))
        cmp = self.run_tool(SCRIPT, a, "compare", os.path.join(a_out, "status-alpha.json"),
                            os.path.join(a, "inbox", "status-beta.json"))
        self.assertTrue(cmp["agree"], cmp)

        with open(os.path.join(a, "inbox", "seat-beta.json")) as f:
            to = json.load(f)["operator_address"]
        before = int(self.run_tool(tool_b, b, "balance", "--chain-id", chain, "--label", "beta", "--address", to)["uzrn"])
        sent = self.run_tool(SCRIPT, a, "send", "--chain-id", chain, "--label", "alpha", "--to", to, "--amount-uzrn", 777)
        self.run_tool(tool_b, b, "status", "--chain-id", chain, "--label", "beta",
                      "--wait-height", sent["height"] + 1, "--timeout", 60)
        after = int(self.run_tool(tool_b, b, "balance", "--chain-id", chain, "--label", "beta", "--address", to)["uzrn"])
        self.assertEqual(after - before, 777)


if __name__ == "__main__":
    unittest.main()
