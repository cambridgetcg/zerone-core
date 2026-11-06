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
import time
import unittest
from pathlib import Path
from unittest import mock

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

    def test_seat_addresses_stay_on_the_owners_network(self):
        lan, private, local = "zerone-rehearsal-lan-1", "zerone-rehearsal-private-1", "zerone-rehearsal-local-1"
        for ip in ("10.0.0.1", "172.16.0.1", "172.31.255.254", "192.168.1.10"):
            for chain in (lan, private):
                self.assertEqual(pn.check_seat_scope(chain, ip), ip)
            with self.assertRaises(pn.Refusal):  # a -local- chain lives on one Mac
                pn.check_seat_scope(local, ip)
        for ip in ("100.64.0.1", "100.127.255.254"):  # overlay space: -private- only
            self.assertEqual(pn.check_seat_scope(private, ip), ip)
            with self.assertRaises(pn.Refusal):
                pn.check_seat_scope(lan, ip)
        for ip in ("8.8.8.8", "192.0.2.10", "172.32.0.1", "100.128.0.1", "169.254.1.1", "11.0.0.1"):
            for chain in (lan, private):
                with self.assertRaises(pn.Refusal, msg="%s on %s" % (ip, chain)):
                    pn.check_seat_scope(chain, ip)
        self.assertEqual(pn.check_seat_scope(local, "127.0.0.1"), "127.0.0.1")

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

    def test_a_binary_that_cannot_run_here_is_not_pinned(self):
        base = private_tmp(self)
        lay = pn.Layout(os.path.join(base, "root"))
        for body, text in ((b"#!/bin/sh\nkill -9 $$\n", "killed by signal 9"), (b"#!/bin/sh\nexit 3\n", "exited 3")):
            src = os.path.join(base, "candidate")
            with open(src, "wb") as f:
                f.write(body)
            os.chmod(src, 0o700)
            with self.assertRaises(pn.Refusal) as caught:
                pn.pin_binary(lay, src, hashlib.sha256(body).hexdigest())
            self.assertIn(text, str(caught.exception))
            self.assertFalse(os.path.exists(lay.binary) or os.path.exists(lay.binary_record))

    @unittest.skipUnless(shutil.which("sandbox-exec"), "macOS sandbox-exec not available")
    def test_sandboxed_bind_names_the_sandbox(self):
        code = ("import importlib.util,sys; s=importlib.util.spec_from_file_location('p', sys.argv[1]); "
                "m=importlib.util.module_from_spec(s); s.loader.exec_module(m)\n"
                "try:\n m.port_free(46999, '127.0.0.1')\nexcept m.Refusal as e:\n print(e)")
        out = subprocess.run(["sandbox-exec", "-p", "(version 1)(allow default)(deny network-bind)",
                              sys.executable, "-c", code, str(SCRIPT)], capture_output=True, text=True)
        self.assertIn("agent sandbox", out.stdout + out.stderr)


class CodeSigning(unittest.TestCase):
    def test_identifier_is_parsed_from_codesign_output(self):
        out = ("Executable=/Users/Shared/zerone-private/bin/zeroned\nIdentifier=a.out\n"
               "Format=Mach-O thin (arm64)\nSignature=adhoc\n")
        self.assertEqual(pn.parse_signing_identifier(out), "a.out")
        self.assertIsNone(pn.parse_signing_identifier("code object is not signed at all\n"))

    def test_adhoc_sign_uses_the_zerone_identifier_and_checks_it(self):
        calls = []
        with mock.patch.object(pn, "run", side_effect=lambda argv, **kw: calls.append(argv) or ""), \
                mock.patch.object(pn, "signing_identifier", return_value=pn.SIGN_IDENTIFIER):
            pn.adhoc_sign("/tmp/zeroned")
        self.assertEqual(calls[0], [pn.CODESIGN, "--force", "--sign", "-", "--identifier", pn.SIGN_IDENTIFIER,
                                    "--timestamp=none", "/tmp/zeroned"])
        self.assertEqual(calls[1], [pn.CODESIGN, "--verify", "--strict", "/tmp/zeroned"])
        with mock.patch.object(pn, "run", return_value=""), \
                mock.patch.object(pn, "signing_identifier", return_value=pn.GO_DEFAULT_IDENTIFIER):
            with self.assertRaises(pn.Refusal):
                pn.adhoc_sign("/tmp/zeroned")

    def test_firewall_warning_explains_the_go_default_identifier(self):
        binary = os.path.join(private_tmp(self), "zeroned")
        with open(binary, "wb") as f:
            f.write(b"binary")
        answers = {"--getglobalstate": "Firewall is enabled. (State = 1)", "--getblockall": "disabled",
                   "--listapps": "Total number of apps = 1\n1 : /opt/homebrew/bin/ipfs\n (Allow incoming connections)"}
        with mock.patch.object(pn, "SOCKETFILTERFW", str(SCRIPT)), \
                mock.patch.object(pn, "run", side_effect=lambda argv, **kw: answers[argv[1]]):
            for ident, explained in ((pn.GO_DEFAULT_IDENTIFIER, True), (pn.SIGN_IDENTIFIER, False)):
                with mock.patch.object(pn, "signing_identifier", return_value=ident):
                    status, detail = pn.firewall_report(binary)
                self.assertEqual(status, "warn")
                self.assertIn("--add " + binary, detail)
                self.assertEqual("default identifier a.out" in detail, explained, ident)

    def test_firewall_ok_says_it_matches_the_path_only(self):
        binary = os.path.join(private_tmp(self), "zeroned")
        answers = {"--getglobalstate": "Firewall is enabled. (State = 1)", "--getblockall": "disabled",
                   "--listapps": "Total number of apps = 1\n1 : %s\n (Allow incoming connections)" % binary}
        with mock.patch.object(pn, "SOCKETFILTERFW", str(SCRIPT)), \
                mock.patch.object(pn, "run", side_effect=lambda argv, **kw: answers[argv[1]]):
            status, detail = pn.firewall_report(binary)
        self.assertEqual(status, "ok")
        self.assertIn("path only", detail)
        self.assertIn("--remove " + binary, detail)

    @unittest.skipUnless(sys.platform == "darwin" and os.path.exists(pn.CODESIGN) and shutil.which("xcrun"),
                         "macOS codesign and clang not available")
    def test_adhoc_sign_renames_and_is_deterministic(self):
        # A fresh linker-signed arm64 executable, like Go's output. A re-signed copy of an Apple
        # arm64e system binary such as /usr/bin/true was killed at exec on the macos-15 runner.
        d = private_tmp(self)
        src, built = os.path.join(d, "main.c"), os.path.join(d, "built")
        with open(src, "w") as f:
            f.write("int main(void) { return 0; }\n")
        subprocess.run(["xcrun", "--sdk", "macosx", "clang", "-arch", "arm64", "-o", built, src], check=True)
        copies = [os.path.join(d, n) for n in ("one", "two")]
        for c in copies:
            shutil.copyfile(built, c)
            os.chmod(c, 0o700)
            pn.adhoc_sign(c)
        self.assertEqual(pn.signing_identifier(copies[0]), pn.SIGN_IDENTIFIER)
        self.assertEqual(pn.sha256_file(copies[0]), pn.sha256_file(copies[1]))
        self.assertEqual(subprocess.run([copies[0]]).returncode, 0)


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
    endpoint = "192.168.77.%d:36656" % (10 + n)
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
        bad(lambda p: (p.update(p2p_endpoint="203.0.113.5:36656"),
                       p["gentx"]["body"].update(memo="%s@203.0.113.5:36656" % p["node_id"])))
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
        self.assertEqual(sorted(os.listdir(out)), ["START-HERE.md", "YOUR-MAC.md", "invite-beta.json", pn.SKILL_NAME, "zeroned"])
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
        self.assertIn("ask the person whose Mac it is to approve that", " ".join(page.split()))
        self.assertNotIn("ask the owner to approve", page)
        for text in ("Yu is the operator", "Ask the person whose Mac it is to make these local changes",
                     "that person runs the two", "that person allows the app", "that person runs `mullvad",
                     "The operator carries", "the operator brings"):
            self.assertIn(text, " ".join(page.split()))

    def test_person_note_for_validator_and_full_seat_is_carried_and_replaceable(self):
        rc, _, err = cli("--root", self.root, "invite", "--chain-id", CHAIN, "--coordinator-label", "alpha",
                         "--label", "laptop", "--role", "full")
        self.assertEqual(rc, 0, err)
        for label, role in (("beta", "validator"), ("laptop", "full seat")):
            out, (rc, _, err) = self.kit(label)
            self.assertEqual(rc, 0, err)
            note = Path(out, "YOUR-MAC.md").read_text()
            plain = " ".join(note.split())
            for text in ("this Mac (" + label + ")", CHAIN, "Your Mac is a " + role,
                         "Yu, the operator, is testing Zerone, a blockchain", "copy of the shared record",
                         'A "seat" means this Mac', "helper is the operator's AI agent", "kit is a folder of setup files",
                         "only the operator's own Macs",
                         "other people's Macs cannot join yet", "stop here: this note doesn't invite it",
                         "20 GiB of free storage", "admin password", "firewall", "Local Network permission", "Mullvad",
                         "reserved home-network address", "Private Wi-Fi Address Fixed or Off",
                         "outside its sandbox", "zeroned also runs as you", "can read what your account can read",
                         "operator must start a new test network", "operator promises to show you",
                         "names and addresses before starting", "you can still say no",
                         "program cannot check who agreed or whose Mac", "Logging out may", "shutdown or restart",
                         "does not restart automatically", "5 GB", "estimate, not a measurement or a storage limit",
                         "secret keys", "personal documents",
                         "time.apple.com", "control port answers only this Mac", "web interfaces are off",
                         "Every seat keeps a copy of the shared record",
                         "separate from zerone's main network", "earns nothing", "operator's rule",
                         "After setup begins and the kit arrives", "To pause", "helper can restart",
                         "`--yes` at the end of the leave command", "any new files since the preview",
                         "Unsafe files cause a refusal",
                         "not Trash", "no undo", "Shared program and setup files stay", "last seat on this Mac",
                         "`firewall_command_for_mac_person` first", "Go > Go to Folder", "select only that folder",
                         "Delete Immediately",
                         "this seat's keys are already gone", "kit folders still in Downloads", "earlier deletions are permanent",
                         "only your account can open them, but any program running as you can read them",
                         "To leave, pause first",
                         "including the program itself, still has this seat's keys or data",
                         "it refuses and deletes nothing",
                         "Before setup, deleting the kit is all declining needs", "doesn't undo permissions or network settings",
                         "restore previous Mullvad, router and Wi-Fi settings", "only if nothing else needs them",
                         "Other Macs' copies and backups", "Time Machine", "Never restore this seat from a backup",
                         "corrected commands before agreeing"):
                self.assertIn(text.lower(), plain.lower())
            self.assertLessEqual(len(note.split()), 720)  # about 700 words, with room for whole sentences
            self.assertLess(plain.index("This seat's keys are already gone"), plain.index("drag it to the Trash"))
            for jargon in ("not a binding list", "bin/ and inbox/", "bridged", "seat/test identifiers",
                           "Links, special files and outside paths refuse", "Unproven stop:",
                           "Otherwise it stops the program", "If it cannot prove the program stopped"):
                self.assertNotIn(jargon, note)
            for heading in ("Before you agree", "What runs and stays here", "What is shared", "Pause or leave"):
                self.assertIn("\n## " + heading + "\n", note)
            self.assertEqual("Keep it awake" in note, role == "validator")
            self.assertEqual("does not vote" in plain, role == "full seat")
            self.assertEqual("catches up by itself" in note, role == "full seat")
            commands = [line.strip() for line in note.splitlines() if line.startswith("    python3")]
            self.assertEqual(commands, ["%s --root %s %s --chain-id %s --label %s" %
                                       (self.INBOX_TOOL, pn.DEFAULT_ROOT, cmd, CHAIN, label) for cmd in ("stop", "leave")])
            self.assertNotIn("--yes", " ".join(commands))  # pause and preview, never pre-confirmed deletion
            self.assertNotIn("sudo /", note)
            root = os.path.join(self.base, label + "-root")
            self.assertEqual(cli("--root", root, "receive", out)[0], 0)
            self.assertEqual(Path(root, "inbox", "YOUR-MAC.md").read_text(), note)
            self.assertEqual(pn.your_mac_text({"label": label, "chain_id": CHAIN, "coordinator": "alpha",
                                             "role": "validator" if role == "validator" else "full"}, 2000000)
                             .count("about 2 MB each"), 1)
            self.assertIn("about 3 MB each", pn.your_mac_text({"label": label, "chain_id": CHAIN,
                          "coordinator": "alpha", "role": "full"}, 2000001))
        note_path = self.write(os.path.join(self.dl, "YOUR-MAC.md"), "corrected person note")
        rc, doc, err = cli("--root", root, "receive", note_path)
        self.assertEqual((rc, doc["replaced"]), (0, ["YOUR-MAC.md"]), err)

    def test_invite_and_kit_warn_about_an_a_out_pin_on_a_lan_chain(self):
        cases = ((pn.GO_DEFAULT_IDENTIFIER, CHAIN, True), (pn.SIGN_IDENTIFIER, CHAIN, False),
                 (pn.GO_DEFAULT_IDENTIFIER, "zerone-rehearsal-local-w1", False))
        for n, (ident, chain, warned) in enumerate(cases):
            with mock.patch.object(pn, "signing_identifier", return_value=ident):
                rc, doc, err = cli("--root", self.root, "invite", "--chain-id", chain, "--coordinator-label",
                                   "alpha", "--label", "gamma%d" % n)
            self.assertEqual(rc, 0, err)
            self.assertEqual("warning" in doc, warned, (ident, chain))
            if warned:
                self.assertIn("--remove", doc["warning"])
                self.assertIn("new chain id", doc["warning"])
        with mock.patch.object(pn, "signing_identifier", return_value=pn.GO_DEFAULT_IDENTIFIER):
            _, (rc, doc, err) = self.kit()
        self.assertEqual(rc, 0, err)
        self.assertIn("warning", doc)

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
        self.assertEqual(sorted(doc["received"]), ["START-HERE.md", "YOUR-MAC.md", "invite-beta.json", pn.SKILL_NAME, "zeroned"])
        self.assertFalse(os.path.exists(out))  # the emptied courier folder goes too
        inbox = os.path.join(self.root2, "inbox")
        self.assertEqual(os.stat(os.path.join(inbox, "zeroned")).st_mode & 0o777, 0o600)
        self.assertEqual(os.stat(os.path.join(inbox, pn.SKILL_NAME)).st_mode & 0o777, 0o700)
        # The same kit again is left where it is.
        rc, doc, err = self.receive(again)
        self.assertEqual((rc, doc["received"], len(doc["already_in_inbox"])), (0, [], 5), err)
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
                     "--expect-genesis-sha256 <sha256 from the operator's message>",
                     "%s status --chain-id %s --label beta --wait-height 3 --timeout 100" % (self.INBOX_TOOL, CHAIN),
                     "Show the operator AND the person whose Mac it is", "other Macs' names and addresses",
                     "Start only after both say yes", "Stop when the operator or the person whose Mac it is asks",
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

    def test_install_next_requires_both_people_before_start(self):
        state = {"genesis_sha256": "a" * 64, "genesis_time": "2026-10-17T14:36:12Z", "rpc_port": 36657}
        with mock.patch.object(pn, "install_seat", return_value=(state, {})), \
                mock.patch.object(pn, "seat_table", return_value=[]):
            rc, doc, err = cli("--root", self.root, "install", "--chain-id", CHAIN, "--label", "beta",
                               "--network", "n.json", "--genesis", "g.json", "--expect-genesis-sha256", "a" * 64)
        self.assertEqual(rc, 0, err)
        assert isinstance(doc, dict)
        for text in ("the operator AND the person whose Mac it is", "names and addresses",
                     "Only after both say yes, `start`"):
            self.assertIn(text, doc["next"])

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


class Leave(unittest.TestCase):
    """Real files, synthetic records/process observations: no node binary, sockets or keys."""

    def setUp(self):
        self.base = private_tmp(self)
        self.root = os.path.join(self.base, "root with spaces")
        self.lay = pn.Layout(self.root, CHAIN)
        self.home = self.lay.seat_home("beta")
        self.root = self.lay.root
        self.write(os.path.join(self.home, "data", "blocks"), b"synthetic chain data")
        self.write(os.path.join(self.home, "config", "key-placeholder"), b"not a key")
        self.state = {"home": self.home, "label": "beta", "role": "validator"}
        self.write(self.lay.seat_file("beta", "state"), pn.json_bytes(self.state))
        for kind in ("gentx", "default-genesis"):
            self.write(self.lay.seat_file("beta", kind), b"{}")
        self.write(self.lay.pid_file("beta"), b"12345\n")
        self.write(self.lay.log_file("beta"), b"synthetic log")
        for kind in ("seat", "status"):
            self.write(os.path.join(self.lay.outbox, kind + "-beta.json"), b"{}")
        self.kept = [self.lay.binary, self.lay.binary_record, os.path.join(self.lay.inbox, "invite-beta.json"),
                     os.path.join(self.lay.outbox, "invite-beta.json"),
                     os.path.join(self.lay.coordinator_home(), "record"),
                     os.path.join(self.lay.seat_home("other"), "keep"),
                     os.path.join(self.root, "nets", "zerone-rehearsal-lan-other", "keep"),
                     os.path.join(self.base, "Downloads", "keep")]
        for path in self.kept:
            self.write(path, b"keep exactly")
        self.processes = self.enter_patch("leave_processes", return_value={})
        self.open_files = self.enter_patch("leave_open_files", return_value=set())
        self.kill = self.enter_patch("os.kill")
        self.subprocess_run = self.enter_patch("run", side_effect=AssertionError("unexpected subprocess"))

    def enter_patch(self, target, **kw):
        # spec_from_file_location is deliberately not in sys.modules; patch via the loaded owner instead.
        owner, name = (pn.os, "kill") if target == "os.kill" else (pn, target)
        patcher = mock.patch.object(owner, name, **kw)
        self.addCleanup(patcher.stop)
        return patcher.start()

    def write(self, path, data):
        os.makedirs(os.path.dirname(path), mode=0o700, exist_ok=True)
        with open(path, "wb") as f:
            f.write(data)
        return path

    def snapshot(self):
        return {str(p.relative_to(self.base)): (p.lstat().st_mode, p.read_bytes() if p.is_file() else None)
                for p in Path(self.base).rglob("*") if not p.is_symlink()}

    def leave(self, *extra):
        return cli("--root", self.root, "leave", "--chain-id", CHAIN, "--label", "beta", *extra)

    def expected_command(self):
        return "%s start --home %s" % (self.lay.binary, self.home)

    def test_dry_run_lists_every_file_without_stopping_or_writing(self):
        before = self.snapshot()
        self.processes.return_value = {12345: self.expected_command()}
        rc, doc, err = self.leave()
        self.assertEqual(rc, 0, err)
        assert isinstance(doc, dict)
        self.assertTrue(doc["dry_run"])
        self.assertEqual(doc["would_stop_pid"], 12345)
        paths = {e["path"] for e in doc["would_remove"]}
        expected = {str(p) for p in Path(self.home).rglob("*")} | {self.home}
        expected |= {self.lay.seat_file("beta", k) for k in ("state", "gentx", "default-genesis")}
        expected |= {self.lay.pid_file("beta"), self.lay.log_file("beta")}
        expected |= {os.path.join(self.lay.outbox, k + "-beta.json") for k in ("seat", "status")}
        self.assertEqual(paths, expected)
        self.assertEqual(len(paths), len(doc["would_remove"]))
        self.assertEqual(next(iter(doc)), "summary")
        self.assertEqual(list(doc)[:3], ["summary", "firewall_command_for_mac_person", "after_last_seat"])
        self.assertIn("only the person whose Mac it is may confirm", doc["next"])
        self.assertTrue(doc["summary"].startswith("Would delete seat beta:"))
        files = [e["path"] for e in doc["would_remove"] if e["kind"] == "file"]
        self.assertEqual(sum(t["files"] for t in doc["targets"]), len(files))
        self.assertEqual(sum(t["bytes"] for t in doc["targets"]), sum(os.stat(p).st_size for p in files))
        self.assertEqual({t["path"] for t in doc["targets"]}, paths - {str(p) for p in Path(self.home).rglob("*")})
        self.assertIn("not a binding list", doc["confirmation"])
        self.assertEqual(self.snapshot(), before)
        self.assertIn("no undo", doc["warning"])
        self.assertEqual(doc["firewall_command_for_mac_person"],
                         "sudo %s --remove %s" % (pn.SOCKETFILTERFW, pn.shlex.quote(self.lay.binary)))
        self.assertIn("the coordinator's records", doc["cannot_remove"])
        self.kill.assert_not_called()
        self.subprocess_run.assert_not_called()

    def test_confirmed_removal_keeps_shared_files_other_seats_and_other_chains(self):
        rc, plan, err = self.leave()
        self.assertEqual(rc, 0, err)
        rc, doc, err = self.leave("--yes")
        self.assertEqual(rc, 0, err)
        self.assertFalse(doc["dry_run"])
        self.assertEqual(set(doc["removed"]), {p["path"] for p in plan["would_remove"]})
        for path in doc["removed"]:
            self.assertFalse(os.path.lexists(path), path)
        for path in self.kept:
            self.assertEqual(Path(path).read_bytes(), b"keep exactly", path)
        rc, doc, err = self.leave("--yes")
        self.assertEqual((rc, doc["removed"]), (0, []), err)
        self.kill.assert_not_called()
        self.subprocess_run.assert_not_called()

    def test_running_seat_gets_only_sigterm_then_files_are_removed(self):
        self.processes.side_effect = [{12345: self.expected_command()}, {}]
        with mock.patch.object(pn, "pid_command", side_effect=[self.expected_command(), ""]):
            rc, doc, err = self.leave("--yes")
        self.assertEqual(rc, 0, err)
        self.assertEqual(doc["stopped_pid"], 12345)
        self.kill.assert_called_once_with(12345, pn.signal.SIGTERM)
        self.assertFalse(os.path.exists(self.home))
        self.subprocess_run.assert_not_called()

    def test_stop_adapter_still_removes_only_the_pid_record(self):
        before = self.snapshot()
        with mock.patch.object(pn, "pid_command", side_effect=[self.expected_command(), self.expected_command(), ""]):
            rc, doc, err = cli("--root", self.root, "stop", "--chain-id", CHAIN, "--label", "beta")
        self.assertEqual((rc, doc["stopped"], doc["pid"]), (0, "beta", 12345), err)
        self.kill.assert_called_once_with(12345, pn.signal.SIGTERM)
        after = self.snapshot()
        pid_key = next(k for k in before if k.endswith("/run/beta.pid"))
        before.pop(pid_key)
        self.assertEqual(after, before)

    def test_last_seat_keeps_bin_inbox_and_empty_parent_folders(self):
        root = os.path.join(self.base, "last-seat-root")
        lay = pn.Layout(root, CHAIN)
        self.write(os.path.join(lay.seat_home("beta"), "data"), b"synthetic data")
        for path in (lay.binary, os.path.join(lay.inbox, "YOUR-MAC.md")):
            self.write(path, b"keep")
        rc, doc, err = cli("--root", root, "leave", "--chain-id", CHAIN, "--label", "beta", "--yes")
        self.assertEqual(rc, 0, err)
        assert isinstance(doc, dict)
        self.assertEqual(len(doc["removed"]), 2)
        self.assertEqual(Path(lay.binary).read_bytes(), b"keep")
        self.assertEqual(Path(lay.inbox, "YOUR-MAC.md").read_bytes(), b"keep")
        self.assertEqual(os.listdir(lay.seats_dir), [])
        self.assertIn("only if this was the last seat on this Mac", doc["after_last_seat"])
        self.assertIn("then drag %s to the Trash" % lay.root, doc["after_last_seat"])
        self.assertIn("the person whose Mac it is runs firewall_command_for_mac_person first", doc["after_last_seat"])
        self.assertIn("keys are already gone", doc["after_last_seat"])
        self.assertIn("kit folders still in Downloads", doc["after_last_seat"])

    def test_still_running_after_stop_timeout_refuses_without_removal(self):
        before = self.snapshot()
        self.processes.return_value = {12345: self.expected_command()}
        with mock.patch.object(pn, "pid_command", return_value=self.expected_command()), \
                mock.patch.object(pn.time, "time", side_effect=[0, 31]):
            rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2)
        self.assertIn("did not stop", err)
        self.assertIn("would_remove", err)  # plan is printed even when the stop refuses
        self.assertEqual(self.snapshot(), before)
        self.kill.assert_called_once_with(12345, pn.signal.SIGTERM)

    def test_pre_genesis_sleep_refuses_without_removal(self):
        self.state["genesis_time"] = "2999-01-01T00:00:00Z"
        self.write(self.lay.seat_file("beta", "state"), pn.json_bytes(self.state))
        before = self.snapshot()
        self.processes.return_value = {12345: self.expected_command()}
        with mock.patch.object(pn, "pid_command", return_value=self.expected_command()), \
                mock.patch.object(pn.time, "time", side_effect=[0, 31]):
            rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2)
        self.assertIn("sleeps until genesis_time", err)
        self.assertEqual(self.snapshot(), before)

    def test_second_process_check_catches_a_restart(self):
        before = self.snapshot()
        self.processes.side_effect = [{}, {12346: self.expected_command()}]
        rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2)
        self.assertIn("still running", err)
        self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_missing_or_stale_pid_is_not_proof_of_a_stopped_node(self):
        self.processes.return_value = {12346: self.expected_command()}
        for missing in (False, True):
            if missing:
                os.unlink(self.lay.pid_file("beta"))
            before = self.snapshot()
            rc, _, err = self.leave("--yes")
            self.assertEqual(rc, 2)
            self.assertIn("cannot safely identify", err)
            self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_stale_pid_refusal_names_process_and_safe_next_step(self):
        self.processes.return_value = {12345: "/bin/sleep 60"}
        before = self.snapshot()
        for extra in ((), ("--yes",)):
            rc, _, err = self.leave(*extra)
            self.assertEqual(rc, 2, err)
            for text in ("12345", "/bin/sleep 60", "do not signal an unrelated process",
                         "Only after verifying the seat is stopped", self.lay.pid_file("beta"), "preview again"):
                self.assertIn(text, err)
            self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_any_open_file_refuses_before_signal_even_for_the_recorded_node(self):
        self.processes.return_value = {12345: self.expected_command()}
        before = self.snapshot()
        for pids in ({12345}, {12346}, {12345, 12346}):
            self.open_files.return_value = pids
            for extra in ((), ("--yes",)):
                rc, _, err = self.leave(*extra)
                self.assertEqual(rc, 2, err)
                self.assertIn("open files", err)
                for pid in pids:
                    self.assertIn(str(pid), err)
                self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_open_file_after_proven_stop_refuses_before_removal(self):
        before = self.snapshot()
        self.processes.side_effect = [{12345: self.expected_command()}, {}]
        self.open_files.side_effect = [set(), {12346}]
        with mock.patch.object(pn, "pid_command", side_effect=[self.expected_command(), ""]):
            rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2, err)
        self.assertIn("open files", err)
        self.assertEqual(self.snapshot(), before)
        self.kill.assert_called_once_with(12345, pn.signal.SIGTERM)
        self.assertEqual(self.open_files.call_args_list, [mock.call(self.home), mock.call(self.home)])

    def test_failed_open_file_check_never_permits_deletion(self):
        before = self.snapshot()
        for after_stop in (False, True):
            self.processes.side_effect = None
            self.processes.return_value = {}
            failure = pn.Refusal("open-file inspection failed")
            self.open_files.side_effect = [set(), failure] if after_stop else failure
            rc, _, err = self.leave("--yes")
            self.assertEqual(rc, 2, err)
            self.assertIn("open-file inspection failed", err)
            self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_new_home_is_checked_on_the_post_stop_inventory(self):
        shutil.rmtree(self.home)
        def scan(home, recorded_pid=None):
            if self.processes.call_count == 2:
                self.write(os.path.join(home, "new-data"), b"new")
            return {}
        self.processes.side_effect = scan
        self.open_files.return_value = {12346}
        rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2, err)
        self.assertIn("open files", err)
        self.open_files.assert_called_once_with(self.home)
        self.assertEqual(Path(self.home, "new-data").read_bytes(), b"new")
        self.assertTrue(os.path.exists(self.lay.pid_file("beta")))
        self.kill.assert_not_called()

    def test_changed_command_is_never_signalled(self):
        self.processes.return_value = {12345: self.expected_command()}
        before = self.snapshot()
        with mock.patch.object(pn, "pid_command", return_value="unrelated process"):
            rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2)
        self.assertIn("command line differs", err)
        self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_outside_root_state_is_refused_even_in_preview(self):
        for home in (self.base, self.lay.seat_home("other"), os.path.join(self.home, "..", "other")):
            self.write(self.lay.seat_file("beta", "state"), pn.json_bytes(dict(self.state, home=home)))
            before = self.snapshot()
            for extra in ((), ("--yes",)):
                rc, _, err = self.leave(*extra)
                self.assertEqual(rc, 2)
                self.assertIn("outside-root", err)
                self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_symlink_escapes_at_every_level_are_refused_without_removal(self):
        outside = os.path.join(self.base, "outside")
        self.write(os.path.join(outside, "untouched"), b"outside")
        targets = [os.path.join(self.root, "nets"), self.lay.chain_dir, self.lay.seats_dir,
                   self.home, os.path.join(self.home, "data"), self.lay.seat_file("beta", "state"),
                   self.lay.pid_file("beta"), self.lay.log_file("beta"), self.lay.outbox]
        for path in targets:
            with self.subTest(path=path):
                saved = path + ".saved"
                os.rename(path, saved)
                os.symlink(outside, path)
                try:
                    for extra in ((), ("--yes",)):
                        rc, _, err = self.leave(*extra)
                        self.assertEqual(rc, 2)
                        self.assertIn("symlink", err)
                    self.assertEqual(Path(outside, "untouched").read_bytes(), b"outside")
                finally:
                    os.unlink(path)
                    os.rename(saved, path)
        self.kill.assert_not_called()

    def test_internal_links_hardlinks_and_special_files_are_refused(self):
        path = os.path.join(self.home, "unusual")
        for kind in ("symlink", "hardlink", "fifo"):
            with self.subTest(kind=kind):
                if kind == "symlink":
                    os.symlink(self.kept[0], path)
                elif kind == "hardlink":
                    os.link(self.kept[0], path)
                else:
                    os.mkfifo(path)
                try:
                    self.assertEqual(self.leave("--yes")[0], 2)
                finally:
                    os.unlink(path)
        self.assertEqual(Path(self.kept[0]).read_bytes(), b"keep exactly")
        self.kill.assert_not_called()

    def test_missing_root_and_missing_chain_create_nothing(self):
        root = os.path.join(self.base, "missing")
        rc, _, err = cli("--root", root, "leave", "--chain-id", CHAIN, "--label", "beta")
        self.assertEqual(rc, 2)
        self.assertIn("does not exist", err)
        self.assertFalse(os.path.exists(root))
        os.mkdir(root, 0o700)
        rc, doc, err = cli("--root", root, "leave", "--chain-id", CHAIN, "--label", "beta", "--yes")
        self.assertEqual((rc, doc["removed"]), (0, []), err)
        self.assertEqual(os.listdir(root), [])

    def test_stop_never_creates_missing_folders_even_after_leaving(self):
        root = os.path.join(self.base, "missing-stop-root")
        command = ("--root", root, "stop", "--chain-id", CHAIN, "--label", "beta")
        self.assertEqual(cli(*command)[0], 2)
        self.assertFalse(os.path.exists(root))
        os.mkdir(root, 0o700)
        self.assertEqual(cli(*command)[0], 2)
        self.assertEqual(os.listdir(root), [])
        self.assertEqual(self.leave("--yes")[0], 0)
        before = self.snapshot()
        self.assertEqual(cli("--root", self.root, "stop", "--chain-id", CHAIN, "--label", "beta")[0], 2)
        self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_unwritable_or_unsearchable_folder_refuses_before_any_deletion(self):
        before = self.snapshot()
        for path in (self.root, self.lay.seats_dir, self.home, os.path.join(self.home, "config")):
            original = os.stat(path).st_mode
            for mode in (0o500, 0o600, 0o400, 0o300, 0o100, 0o000):
                with self.subTest(path=path, mode=oct(mode)):
                    os.chmod(path, mode)
                    try:
                        for extra in ((), ("--yes",)):
                            rc, _, err = self.leave(*extra)
                            self.assertEqual(rc, 2, err)
                            self.assertNotIn("Have the owner move", err)
                            self.assertNotIn("macOS privacy", err)
                    finally:
                        os.chmod(path, original)
                    self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_confirmation_help_names_the_person_whose_mac_it_is(self):
        output = io.StringIO()
        with contextlib.redirect_stdout(output), self.assertRaises(SystemExit) as caught:
            pn.main(["leave", "--help"])
        self.assertEqual(caught.exception.code, 0)
        self.assertIn("the person whose Mac it is confirms", " ".join(output.getvalue().split()))

    @unittest.skipUnless(hasattr(os, "chflags"), "filesystem flags need chflags")
    def test_immutable_and_append_only_entries_refuse_before_any_deletion(self):
        before = self.snapshot()
        for path in (os.path.join(self.home, "config", "key-placeholder"), self.home, self.root):
            original = os.stat(path).st_flags
            for flag in (pn.stat.UF_IMMUTABLE, pn.stat.UF_APPEND):
                with self.subTest(path=path, flag=flag):
                    os.chflags(path, flag)
                    try:
                        for extra in ((), ("--yes",)):
                            rc, _, err = self.leave(*extra)
                            self.assertEqual(rc, 2, err)
                            self.assertIn("immutable or append-only", err)
                            self.assertNotIn("Have the owner move", err)
                    finally:
                        os.chflags(path, original)
                    self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_system_immutable_and_append_only_flags_are_also_refused(self):
        for name in ("UF_IMMUTABLE", "UF_APPEND", "SF_IMMUTABLE", "SF_APPEND"):
            flag = getattr(pn.stat, name, 0)
            if not flag:
                continue
            st = mock.Mock(st_mode=pn.stat.S_IFREG | 0o600, st_uid=os.getuid(), st_dev=1,
                           st_nlink=1, st_flags=flag)
            with mock.patch.object(pn.os, "stat", return_value=st):
                with self.assertRaisesRegex(pn.Refusal, "immutable or append-only"):
                    pn.leave_stat(0, "synthetic", 1)

    def test_partial_removal_error_never_advises_moving_a_key(self):
        original_unlink = os.unlink
        deleted = []
        def fail_second(name, *, dir_fd):
            if deleted:
                raise PermissionError(13, "Permission denied", name)
            original_unlink(name, dir_fd=dir_fd)
            deleted.append(name)
        with mock.patch.object(pn.os, "unlink", side_effect=fail_second):
            rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2, err)
        self.assertEqual(len(deleted), 1)
        self.assertIn("removal stopped partway", err)
        self.assertIn("earlier deletions are permanent", err)
        self.assertIn("preview again", err)
        self.assertNotIn("Have the owner move", err)
        self.assertNotIn("macOS privacy", err)
        self.assertTrue(os.path.isdir(self.home))
        self.assertEqual(self.leave()[0], 0)
        self.kill.assert_not_called()

    def test_folder_moved_during_removal_has_partial_removal_message(self):
        original_remove = pn.leave_remove
        def move_then_remove(root_fd, snapshot):
            os.rename(self.home, self.home + ".saved")
            original_remove(root_fd, snapshot)
        with mock.patch.object(pn, "leave_remove", side_effect=move_then_remove):
            rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2, err)
        self.assertIn("removal stopped partway", err)
        self.assertIn("earlier deletions are permanent", err)
        self.assertIn("preview again", err)
        self.assertNotIn("Have the owner move", err)
        self.assertEqual(Path(self.home + ".saved", "config", "key-placeholder").read_bytes(), b"not a key")
        self.kill.assert_not_called()

    def test_symlink_root_and_bad_identifiers_are_refused(self):
        link = os.path.join(self.base, "linked-root")
        os.symlink(self.root, link)
        self.assertEqual(cli("--root", link, "leave", "--chain-id", CHAIN, "--label", "beta", "--yes")[0], 2)
        for chain, label in (("zerone-1", "beta"), (CHAIN, "../beta"), ("../" + CHAIN, "beta")):
            self.assertEqual(cli("--root", self.root, "leave", "--chain-id", chain, "--label", label, "--yes")[0], 2)
        self.kill.assert_not_called()

    def test_shutdown_writes_are_inventoried_and_removed_after_proven_stop(self):
        self.processes.side_effect = [{12345: self.expected_command()}, {}]
        new = os.path.join(self.home, "new-file")
        replaced = os.path.join(self.home, "data", "blocks")
        def stopped(*args):
            self.write(new, b"new")
            self.write(replaced + ".tmp", b"replacement at shutdown")
            os.replace(replaced + ".tmp", replaced)
            os.unlink(self.lay.log_file("beta"))
        with mock.patch.object(pn, "stop_process", side_effect=stopped):
            rc, doc, err = self.leave("--yes")
        self.assertEqual(rc, 0, err)
        self.assertIn(new, doc["removed"])
        self.assertIn(replaced, doc["removed"])
        self.assertNotIn(self.lay.log_file("beta"), doc["removed"])
        self.assertNotIn(new, [e["path"] for e in doc["would_remove"]])
        self.assertTrue(doc["summary"].startswith("Deleted seat beta:"))
        self.assertFalse(os.path.exists(self.home))

    def test_confirmation_is_not_bound_to_an_earlier_preview(self):
        rc, preview, err = self.leave()
        self.assertEqual(rc, 0, err)
        added = self.write(os.path.join(self.home, "after-preview"), b"new")
        self.assertNotIn(added, [e["path"] for e in preview["would_remove"]])
        rc, doc, err = self.leave("--yes")
        self.assertEqual(rc, 0, err)
        self.assertIn(added, doc["removed"])
        self.assertFalse(os.path.exists(added))

    def test_unsafe_shutdown_additions_still_refuse_before_any_removal(self):
        path = os.path.join(self.home, "new-file")
        for kind in ("internal-symlink", "outside-symlink", "hardlink", "fifo"):
            with self.subTest(kind=kind):
                before = self.snapshot()
                self.processes.side_effect = [{12345: self.expected_command()}, {}]
                def stopped(*args):
                    if kind == "internal-symlink":
                        os.symlink(self.kept[0], path)
                    elif kind == "outside-symlink":
                        os.symlink(self.base, path)
                    elif kind == "hardlink":
                        os.link(self.kept[0], path)
                    else:
                        os.mkfifo(path)
                try:
                    with mock.patch.object(pn, "stop_process", side_effect=stopped):
                        rc, _, err = self.leave("--yes")
                    self.assertEqual(rc, 2, err)
                finally:
                    os.unlink(path)
                self.assertEqual(self.snapshot(), before)

    def test_non_object_state_refuses_without_traceback_or_mutation(self):
        for state in ([], None, "not an object", 1, True):
            with self.subTest(state=state):
                self.write(self.lay.seat_file("beta", "state"), pn.json_bytes(state))
                before = self.snapshot()
                for extra in ((), ("--yes",)):
                    rc, _, err = self.leave(*extra)
                    self.assertEqual(rc, 2, err)
                    self.assertIn("state is not a JSON object", err)
                    self.assertNotIn("Traceback", err)
                    self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_malformed_pid_refuses_without_mutation(self):
        for content in (b"", b"[]", b"not a pid", b"123 456"):
            self.write(self.lay.pid_file("beta"), content)
            before = self.snapshot()
            for extra in ((), ("--yes",)):
                rc, _, err = self.leave(*extra)
                self.assertEqual(rc, 2, err)
                self.assertIn("malformed input", err)
                self.assertEqual(self.snapshot(), before)
        self.kill.assert_not_called()

    def test_oversized_or_swapped_records_refuse(self):
        for path in (self.lay.seat_file("beta", "state"), self.lay.pid_file("beta")):
            original = Path(path).read_bytes()
            with open(path, "wb") as stream:
                stream.truncate(pn.MAX_COURIER_JSON + 1)
            for extra in ((), ("--yes",)):
                rc, _, err = self.leave(*extra)
                self.assertEqual(rc, 2, err)
                self.assertIn("not a small plain file", err)
            self.write(path, original)
            root_fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)
            try:
                rel = os.path.relpath(path, self.root)
                plan = pn.leave_snapshot(root_fd, [rel])
                self.write(path + ".tmp", original)
                os.replace(path + ".tmp", path)
                with self.assertRaisesRegex(pn.Refusal, "record changed"):
                    pn.leave_read(root_fd, rel, plan[rel])
            finally:
                os.close(root_fd)
        self.kill.assert_not_called()

    def test_record_changed_before_signal_refuses(self):
        def scan(home, recorded_pid=None):
            self.write(self.lay.seat_file("beta", "state"), pn.json_bytes(dict(self.state, role="full")))
            return {12345: self.expected_command()}
        self.processes.side_effect = scan
        rc, _, err = self.leave("--yes")
        self.assertEqual(rc, 2)
        self.assertIn("state/PID record changed", err)
        self.assertTrue(os.path.exists(self.lay.seat_file("beta", "state")))
        self.kill.assert_not_called()

    def test_deletion_rejects_an_ancestor_swapped_for_an_outside_symlink(self):
        outside = os.path.join(self.base, "outside")
        victim = self.write(os.path.join(outside, "victim"), b"untouched")
        root_fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)
        try:
            plan = pn.leave_snapshot(root_fd, [os.path.relpath(self.home, self.root)])
            os.rename(self.home, self.home + ".saved")
            os.symlink(outside, self.home)
            with self.assertRaises(pn.Refusal):
                pn.leave_remove(root_fd, plan)
            self.assertEqual(Path(victim).read_bytes(), b"untouched")
        finally:
            os.close(root_fd)


class LeaveProcessScan(unittest.TestCase):
    def hand_started_process(self, variant):
        root = private_tmp(self)
        lay = pn.Layout(os.path.join(root, "root"), CHAIN)
        home = lay.seat_home("mac")
        os.mkdir(home, 0o700)
        marker = Path(home, "key-placeholder")
        marker.write_text("not a key")
        state = Path(lay.seat_file("mac", "state"))
        if variant != "E9c":
            state.write_bytes(pn.json_bytes({"home": home, "label": "mac"}))
        command_root = lay.root
        spelling = home
        if variant == "E9c":
            command_root = os.path.join(root, "ROOT")
            if not os.path.exists(command_root) or not os.path.samefile(command_root, lay.root):
                self.skipTest("E9c needs a case-insensitive filesystem")
        elif variant == "E9d":
            spelling = home.replace("/seats/mac", "/seats//mac")
        elif variant == "dot-and-trailing":
            spelling = home.replace("/seats/mac", "/./seats/./mac/")
        elif variant == "unmatched-live-pid":
            spelling = os.path.join(root, "unrelated")
        script = Path(root, "sleeper.py")
        script.write_text("import time\nprint('ready', flush=True)\ntime.sleep(60)\n")
        proc = subprocess.Popen([sys.executable, "-B", str(script), "--home", spelling],
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        try:
            assert proc.stdout is not None
            self.assertEqual(proc.stdout.readline().strip(), "ready")
            Path(lay.pid_file("mac")).write_text(str(proc.pid))
            before = {str(p): p.read_bytes() for p in Path(lay.root).rglob("*") if p.is_file()}
            with mock.patch.object(pn.os, "kill") as kill:
                for extra in ((), ("--yes",)):
                    rc, _, err = cli("--root", command_root, "leave", "--chain-id", CHAIN,
                                     "--label", "mac", *extra)
                    self.assertEqual(rc, 2, err)
                    self.assertIn("cannot safely identify", err)
                    self.assertEqual({str(p): p.read_bytes() for p in Path(lay.root).rglob("*") if p.is_file()}, before)
                    self.assertIsNone(proc.poll())
                kill.assert_not_called()
        finally:
            proc.terminate()
            proc.communicate(timeout=10)

    def test_e9c_case_alias_without_state_refuses_and_sends_no_signal(self):
        self.hand_started_process("E9c")

    def test_e9d_repeated_slash_refuses_and_sends_no_signal(self):
        self.hand_started_process("E9d")

    def test_dot_and_trailing_slash_refuse_and_send_no_signal(self):
        self.hand_started_process("dot-and-trailing")

    def test_live_recorded_pid_without_home_match_refuses_and_sends_no_signal(self):
        self.hand_started_process("unmatched-live-pid")

    def test_home_spellings_are_discovery_only_not_signal_permission(self):
        home = "/synthetic/root.with spaces/seats/mac"
        commands = {100: "zeroned start --home " + home.upper(),
                    101: "wrapper --home=" + home.replace("/", "//./") + "/",
                    102: "editor " + home.upper() + "-b",
                    103: "unrelated live process"}
        with mock.patch.object(pn, "run", return_value="\n".join("%d %s" % item for item in commands.items())):
            self.assertEqual(pn.leave_processes(home), {100: commands[100], 101: commands[101]})
            self.assertEqual(pn.leave_processes(home, 103),
                             {100: commands[100], 101: commands[101], 103: commands[103]})

    def test_single_pid_lookup_uses_system_ps(self):
        with mock.patch.object(pn, "run", return_value="expected command\n") as run:
            self.assertEqual(pn.pid_command(123), "expected command")
        run.assert_called_once_with(["/bin/ps", "-ww", "-p", "123", "-o", "command="], check=False)

    def test_live_recorded_pid_with_unreadable_command_is_not_lost(self):
        with mock.patch.object(pn, "run", return_value="123\n124 unrelated\n"):
            self.assertEqual(pn.leave_processes("/synthetic/seat", 123), {123: ""})

    def test_real_sibling_process_does_not_block_leaving_a_stopped_seat(self):
        root = private_tmp(self)
        lay = pn.Layout(root, CHAIN)
        home = lay.seat_home("mac")
        sibling = lay.seat_home("mac-b")
        os.mkdir(home, 0o700)
        os.mkdir(sibling, 0o700)
        Path(home, "synthetic").write_text("remove")
        Path(sibling, "synthetic").write_text("keep")
        script = Path(root, "sleeper.py")
        script.write_text("import time\nprint('ready', flush=True)\ntime.sleep(60)\n")
        proc = subprocess.Popen([sys.executable, "-B", str(script), "--home", sibling],
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        try:
            assert proc.stdout is not None
            self.assertEqual(proc.stdout.readline().strip(), "ready")
            self.assertIn(proc.pid, pn.leave_processes(sibling))
            self.assertEqual(pn.leave_processes(home), {})
            # Process-boundary regression; open-file discovery is exercised separately.
            with mock.patch.object(pn, "leave_open_files", return_value=set()):
                rc, doc, err = cli("--root", root, "leave", "--chain-id", CHAIN, "--label", "mac", "--yes")
            self.assertEqual(rc, 0, err)
            self.assertFalse(os.path.exists(home))
            self.assertEqual(Path(sibling, "synthetic").read_text(), "keep")
            self.assertIsNone(proc.poll())
        finally:
            proc.terminate()
            proc.communicate(timeout=10)

    def test_home_match_has_path_boundaries_not_sibling_prefixes(self):
        home = "/synthetic/root.with spaces/seats/mac"
        commands = {
            100: "zeroned start --home " + home,
            101: "wrapper --home=" + home,
            102: "tail " + home + "/log",
            103: "editor " + home + " notes",
            104: "zeroned query --home " + home,
            105: "zeroned start --home " + home + "-b",
            106: "editor " + home + ".saved",
            107: "wrapper /prefix" + home,
            108: "zeroned start --home /synthetic/rootXwith spaces/seats/mac",
        }
        lines = "\n".join("%d %s" % pair for pair in commands.items())
        with mock.patch.object(pn, "run", return_value=lines):
            self.assertEqual(pn.leave_processes(home), {pid: commands[pid] for pid in range(100, 105)})

    def test_real_cli_removes_synthetic_files_without_a_node_binary(self):
        root = private_tmp(self)
        lay = pn.Layout(root, CHAIN)
        os.mkdir(lay.seat_home("beta"), 0o700)
        Path(lay.seat_home("beta"), "synthetic-data").write_text("no keys or node binary")
        command = [sys.executable, "-B", str(SCRIPT), "--root", root,
                   "leave", "--chain-id", CHAIN, "--label", "beta"]
        preview = subprocess.run(command, capture_output=True, text=True, timeout=30)
        lsof = "/usr/sbin/lsof" if sys.platform == "darwin" else "/usr/bin/lsof"
        if not os.path.isfile(lsof):
            self.assertEqual(preview.returncode, 2, preview.stderr)
            self.assertIn("cannot check open files", preview.stderr)
            acted = subprocess.run(command + ["--yes"], capture_output=True, text=True, timeout=30)
            self.assertEqual(acted.returncode, 2, acted.stderr)
            self.assertTrue(Path(lay.seat_home("beta"), "synthetic-data").is_file())
            return
        self.assertEqual(preview.returncode, 0, preview.stderr)
        self.assertTrue(os.path.isdir(lay.seat_home("beta")))
        acted = subprocess.run(command + ["--yes"], capture_output=True, text=True, timeout=30)
        self.assertEqual(acted.returncode, 0, acted.stderr)
        self.assertEqual(set(json.loads(acted.stdout)["removed"]),
                         {entry["path"] for entry in json.loads(preview.stdout)["would_remove"]})
        self.assertFalse(os.path.exists(lay.seat_home("beta")))
        self.assertFalse(os.path.exists(lay.binary))

    def test_ps_failure_is_not_an_empty_process_list(self):
        with mock.patch.object(pn, "run", side_effect=pn.Refusal("ps failed")):
            with self.assertRaises(pn.Refusal):
                pn.leave_processes("/synthetic/seat")

    def test_unrecorded_and_ambiguous_home_matches_are_found(self):
        home = "/synthetic/root with spaces/seats/beta"
        lines = "123 /synthetic/bin/zeroned start --home %s\n124 wrapper --home=%s --extra\n125 other\n" % (home, home)
        with mock.patch.object(pn, "run", return_value=lines) as run:
            result = pn.leave_processes(home)
        self.assertEqual(set(result), {123, 124})
        run.assert_called_once_with(["/bin/ps", "-axww", "-o", "pid=,command="])


class LeaveOpenFiles(unittest.TestCase):
    """Real file-holding Python stand-ins; no zeroned, signing keys or sockets."""

    def test_lsof_status_and_diagnostics_are_fail_closed(self):
        cases = [(1, "", "", set()), (0, "123\n456\n123\n", "", {123, 456}),
                 (0, "", "", None), (2, "", "", None), (-9, "", "", None),
                 (1, "123\n", "", {123}), (0, "not a PID\n", "", None),
                 (0, "0\n", "", None), (0, "-1\n", "", None),
                 (1, "", "WARNING: cannot stat", None),
                 (0, "123\n", "WARNING: output may be incomplete", None)]
        for platform, binary in (("darwin", "/usr/sbin/lsof"), ("linux", "/usr/bin/lsof")):
            for rc, out, err, expected in cases:
                with self.subTest(platform=platform, rc=rc, out=out, err=err), \
                        mock.patch.object(pn.sys, "platform", platform), \
                        mock.patch.object(pn.subprocess, "run", return_value=
                                          subprocess.CompletedProcess([], rc, out, err)) as run:
                    if expected is None:
                        with self.assertRaises(pn.Refusal):
                            pn.leave_open_files("/synthetic/seat")
                    else:
                        self.assertEqual(pn.leave_open_files("/synthetic/seat"), expected)
                    self.assertEqual(run.call_args[0][0],
                                     [binary, "-n", "-P", "-t", "+w", "+D", "/synthetic/seat"])
                    self.assertEqual(run.call_args[1]["timeout"], 30)

    def test_unavailable_or_timed_out_lsof_is_not_stop_proof(self):
        for error in (FileNotFoundError("lsof absent"), PermissionError("not executable"),
                      subprocess.TimeoutExpired("lsof", 30)):
            with self.subTest(error=error), mock.patch.object(pn.subprocess, "run", side_effect=error):
                with self.assertRaisesRegex(pn.Refusal, "cannot check open files"):
                    pn.leave_open_files("/synthetic/seat")

    def check_spelling(self, variant):
        base = private_tmp(self)
        lay = pn.Layout(os.path.join(base, "root with spaces"), CHAIN)
        home = lay.seat_home("mac")
        os.mkdir(home, 0o700)
        Path(home, "data").mkdir(mode=0o700)
        Path(home, "data", "held-file").write_bytes(b"synthetic data, not a key")
        Path(lay.seat_file("mac", "state")).write_bytes(pn.json_bytes({"home": home, "label": "mac"}))
        # Python accepts a script named 'start', without compiling a node binary.
        os.mkdir(lay.bin_dir, 0o700)
        os.symlink(sys.executable, lay.binary)
        Path(lay.root, "start").write_text(
            "import os, sys, time\n"
            "home = sys.argv[sys.argv.index('--home') + 1]\n"
            "held = open(os.path.join(home, 'data', 'held-file'), 'rb') if os.environ.get('HOLD') == 'yes' else None\n"
            "print('ready', flush=True)\n"
            "time.sleep(60)\n")
        spelling = home
        if variant in ("B3", "B7c"):
            alias = os.path.join(base, "alias")
            os.symlink(lay.root, alias)
            spelling = alias + home[len(lay.root):]
        elif variant == "B3v":
            if home.startswith(("/private/var/", "/private/tmp/")):
                spelling = home[len("/private"):]
            elif sys.platform == "darwin":
                self.skipTest("B3v native alias needs TMPDIR under /var or /tmp on macOS")
            else:
                # Linux lacks macOS's /var -> /private/var alias. Model the
                # same filesystem relationship under this private fixture only.
                alias = os.path.join(base, "var-alias")
                os.symlink(os.path.dirname(lay.root), alias)
                spelling = os.path.join(alias, os.path.relpath(home, os.path.dirname(lay.root)))
        elif variant == "B4":
            os.mkdir(lay.seat_home("mac-b"), 0o700)
            spelling = home.replace("/seats/mac", "/seats/mac-b/../mac")
        elif variant == "B5":
            spelling = os.path.relpath(home, lay.root)
        procs = []
        try:
            for path in ([home, spelling] if variant == "B7c" else [spelling]):
                proc = subprocess.Popen([lay.binary, "start", "--home", path], cwd=lay.root,
                                        env=dict(os.environ, HOLD="no" if variant == "B7c" and not procs else "yes"),
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                procs.append(proc)
                assert proc.stdout is not None
                self.assertEqual(proc.stdout.readline().strip(), "ready")
            if variant == "B7c":
                Path(lay.pid_file("mac")).write_text(str(procs[0].pid))
                self.assertEqual(set(pn.leave_processes(home, procs[0].pid)), {procs[0].pid})
            else:
                self.assertEqual(pn.leave_processes(home), {})  # argv-only discovery misses these
            lsof = "/usr/sbin/lsof" if sys.platform == "darwin" else "/usr/bin/lsof"
            with contextlib.ExitStack() as stack:
                if variant == "B7c":
                    # Apple's 3.9 launcher re-execs Python under its framework
                    # path. Model only the recorded command identity; real ps
                    # must still discover the first process and miss the alias.
                    real_scan = pn.leave_processes
                    def scan(home, recorded_pid=None):
                        found = real_scan(home, recorded_pid)
                        if procs[0].pid in found:
                            found[procs[0].pid] = "%s start --home %s" % (lay.binary, home)
                        return found
                    stack.enter_context(mock.patch.object(pn, "leave_processes", side_effect=scan))
                if not os.path.isfile(lsof):
                    self.assertNotEqual(sys.platform, "darwin", "macOS regression requires system lsof")
                    # CI need not install lsof: mock only its OS observation, not
                    # the adapter, process scan, guards, signal or deletion path.
                    real_run = subprocess.run
                    def observe(argv, **kwargs):
                        if argv[0] == lsof:
                            return subprocess.CompletedProcess(argv, 0,
                                                               "%d\n" % procs[-1].pid, "")
                        return real_run(argv, **kwargs)
                    stack.enter_context(mock.patch.object(pn.subprocess, "run", side_effect=observe))
                def snapshot():
                    return {str(p): (p.lstat().st_mode, os.readlink(p) if p.is_symlink() else
                                    p.read_bytes() if p.is_file() else None)
                            for p in Path(lay.root).rglob("*")}
                before = snapshot()
                with mock.patch.object(pn.os, "kill") as kill:
                    for extra in ((), ("--yes",)):
                        rc, _, err = cli("--root", lay.root, "leave", "--chain-id", CHAIN,
                                         "--label", "mac", *extra)
                        self.assertEqual(rc, 2, err)
                        self.assertIn("open files", err)
                        self.assertIn(str(procs[-1].pid), err)
                        self.assertEqual(snapshot(), before)
                        for proc in procs:
                            self.assertIsNone(proc.poll())
                    kill.assert_not_called()
        finally:
            for proc in procs:
                proc.terminate()
                proc.communicate(timeout=10)

    def test_b3_symlinked_middle_folder_without_pid(self):
        self.check_spelling("B3")

    def test_b3v_private_var_or_tmp_alias_without_pid(self):
        self.check_spelling("B3v")

    def test_b4_parent_component_without_pid(self):
        self.check_spelling("B4")

    def test_b5_relative_home_without_pid(self):
        self.check_spelling("B5")

    def test_b7c_recorded_exact_node_and_unrecorded_alias_node(self):
        self.check_spelling("B7c")


class Invitations(unittest.TestCase):
    """Genesis admits only seats the coordinator invited, with the role and stake each invitation fixed."""

    def setUp(self):
        base = private_tmp(self)
        self.root = os.path.join(base, "root")
        fake = os.path.join(base, "fake-zeroned")
        with open(fake, "wb") as f:
            f.write(FAKE_ZEROND)
        os.chmod(fake, 0o700)
        rc, _, err = cli("--root", self.root, "binary", "--from", fake, "--sha256", file_sha(fake))
        self.assertEqual(rc, 0, err)
        for label, extra in (("seat-1", ()), ("seat-2", ("--role", "full")), ("seat-3", ("--stake-zrn", "500"))):
            rc, _, err = cli("--root", self.root, "invite", "--chain-id", CHAIN, "--coordinator-label", "seat-1",
                             "--label", label, *extra)
            self.assertEqual(rc, 0, err)
        self.lay = pn.Layout(self.root, CHAIN)

    def test_invited_seats_pass_and_the_unseated_are_listed(self):
        pn.check_invitations(self.lay, [seat(1), seat(2, "full"), seat(3, stake_zrn=500)], "seat-1")
        pn.check_invitations(self.lay, [seat(1)], "seat-1")
        with self.assertRaises(pn.Refusal) as cm:
            pn.check_invitations(self.lay, [seat(1)], "seat-9")
        self.assertIn("names coordinator seat-1", str(cm.exception))
        self.assertEqual(sorted(pn.issued_invitations(self.lay)), ["seat-1", "seat-2", "seat-3"])

    def test_uninvited_seats_and_changed_terms_are_refused(self):
        for packets, why in (([seat(1), seat(4)], "never invited"),
                             ([seat(1, stake_zrn=2000)], "its invitation fixes 1000000000"),
                             ([seat(2)], "its invitation fixes full"),
                             ([seat(1, "full")], "its invitation fixes validator")):
            with self.assertRaises(pn.Refusal) as cm:
                pn.check_invitations(self.lay, packets, "seat-1")
            self.assertIn(why, str(cm.exception))

    def test_genesis_refuses_an_uninvited_seat_before_running_zeroned(self):
        pins = {"binary_sha256": file_sha(os.path.join(self.root, "bin", "zeroned")), "tool_sha256": pn.tool_sha256()}
        paths = []
        for packet in (dict(seat(1), **pins), dict(seat(4), **pins)):
            paths.append(os.path.join(self.root, "inbox", "%s.json" % packet["label"]))
            with open(paths[-1], "w") as f:
                json.dump(packet, f)
        rc, _, err = cli("--root", self.root, "genesis", "--chain-id", CHAIN, "--coordinator-label", "seat-1", *paths)
        self.assertEqual(rc, 2, err)
        self.assertIn("seat seat-4 was never invited", err)
        self.assertFalse(os.path.exists(os.path.join(self.root, "nets", CHAIN, "coordinator")))

    def test_a_local_rehearsal_needs_no_invitations(self):
        lay = pn.Layout(self.root, "zerone-rehearsal-local-inv")
        packets = [dict(seat(n), chain_id=lay.chain_id) for n in (1, 2)]
        pn.check_invitations(lay, packets, "seat-1")


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

        # Only seats may connect. Each seat names the other as its one unconditional peer and takes no
        # other inbound peer. A node that holds the genesis and dials both seats, but has no seat, is
        # turned away. Whichever seat dialled the other has its one connection slot free, so that seat's
        # node-key check (CometBFT acceptRoutine) must refuse the outsider by name in its log.
        with open(os.path.join(a_out, "network-%s.json" % chain)) as f:
            ids = {s["label"]: s["node_id"] for s in json.load(f)["seats"]}
        with open(os.path.join(a, "nets", chain, "seats", "alpha", "config", "config.toml")) as f:
            cfg = f.read()
        self.assertIn("max_num_inbound_peers = 0", cfg)
        self.assertIn('unconditional_peer_ids = "%s"' % ids["beta"], cfg)
        seat_logs = [os.path.join(a, "nets", chain, "run", "alpha.log"), os.path.join(b, "nets", chain, "run", "beta.log")]
        log_starts = [os.path.getsize(path) for path in seat_logs]
        outsider = os.path.join(base, "outsider")
        pn.zd(binary, "init", "outsider", "--chain-id", chain, "--default-denom", pn.DENOM, "--home", outsider)
        shutil.copy(os.path.join(a_out, "genesis-%s.json" % chain), os.path.join(outsider, "config", "genesis.json"))
        o_port = port + 20
        pn.set_toml(os.path.join(outsider, "config", "config.toml"), {
            ("rpc", "laddr"): pn.toml_str("tcp://127.0.0.1:%d" % (o_port + 1)),
            ("p2p", "laddr"): pn.toml_str("tcp://127.0.0.1:%d" % o_port),
            ("p2p", "persistent_peers"): pn.toml_str("%s@127.0.0.1:%d,%s@127.0.0.1:%d"
                                                     % (ids["alpha"], port, ids["beta"], port + 10)),
            ("p2p", "pex"): "false", ("p2p", "addr_book_strict"): "false", ("p2p", "allow_duplicate_ip"): "true",
        })
        pn.set_toml(os.path.join(outsider, "config", "app.toml"), {
            ("", "minimum-gas-prices"): pn.toml_str("1" + pn.DENOM), ("api", "enable"): "false",
            ("grpc", "enable"): "false"})
        o_log = os.path.join(a, "nets", chain, "run", "outsider.log")  # kept with the seats' evidence
        with open(o_log, "ab") as log:
            proc = subprocess.Popen([binary, "start", "--home", outsider], stdout=log, stderr=subprocess.STDOUT,
                                    stdin=subprocess.DEVNULL, env=pn.clean_env())
        self.addCleanup(proc.wait, 30)
        self.addCleanup(proc.kill)
        for _ in range(60):
            if pn.rpc_or_absent(o_port + 1, "/status", timeout=5) is not None:
                break
            time.sleep(1)
        else:
            self.fail("the outsider node never answered on RPC")

        def seat_peers(rpc_port):
            for _ in range(10):  # a loaded machine can answer slowly; no answer is not a result
                seen = pn.rpc_or_absent(rpc_port, "/net_info", timeout=5)
                if seen is not None:
                    return sorted(p["node_info"]["moniker"] for p in seen["peers"])
                time.sleep(2)
            self.fail("seat RPC 127.0.0.1:%d did not answer" % rpc_port)

        for _ in range(4):  # the outsider redials at once; sample the seats while it does
            self.assertEqual((seat_peers(port + 1), seat_peers(port + 11)), (["beta"], ["alpha"]))
            time.sleep(2)
        refused = []
        for path, offset in zip(seat_logs, log_starts):
            with open(path, "rb") as f:
                f.seek(offset)
                refused.append(b"Ignoring inbound connection" in f.read())
        self.assertTrue(any(refused), "no seat logged refusing the outsider at its node-key check")

if __name__ == "__main__":
    unittest.main()
