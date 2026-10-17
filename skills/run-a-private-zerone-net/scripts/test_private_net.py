"""Tests for private-net.py.

Pure checks always run. The full loopback rehearsal runs only when
ZERONE_PRIVATE_NET_TEST_BINARY names a built zeroned (CI passes the consensus
rehearsal's binary). ZERONE_PRIVATE_NET_EVIDENCE_PARENT, when set, receives the
rehearsal's node logs and public packets (never seat homes or keys).
"""
import copy
import glob
import hashlib
import importlib.util
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

    @unittest.skipUnless(sys.platform == "darwin" and os.path.exists(pn.CODESIGN), "macOS codesign not available")
    def test_adhoc_sign_renames_and_is_deterministic(self):
        d = private_tmp(self)
        copies = [os.path.join(d, n) for n in ("one", "two")]
        for c in copies:
            shutil.copyfile("/usr/bin/true", c)
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


@unittest.skipUnless(os.environ.get("ZERONE_PRIVATE_NET_TEST_BINARY"), "set ZERONE_PRIVATE_NET_TEST_BINARY to run")
class Rehearsal(unittest.TestCase):
    def keep_evidence(self, root):
        parent = os.environ.get("ZERONE_PRIVATE_NET_EVIDENCE_PARENT")
        if not parent:
            return
        os.makedirs(parent, exist_ok=True)
        dest = tempfile.mkdtemp(prefix="private-net-", dir=parent)
        for f in glob.glob(os.path.join(root, "nets", "*", "run", "*.log")) + \
                glob.glob(os.path.join(root, "nets", "*", "outbox", "*.json")):
            shutil.copy(f, os.path.join(dest, os.path.basename(f)))

    def test_four_validators_and_a_full_seat_on_loopback(self):
        binary = os.environ["ZERONE_PRIVATE_NET_TEST_BINARY"]
        root = os.path.join(private_tmp(self), "root")
        self.addCleanup(self.keep_evidence, root)  # runs before the tmp cleanup
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


if __name__ == "__main__":
    unittest.main()
