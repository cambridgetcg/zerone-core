#!/usr/bin/env python3
"""Synthetic public-only verifier tests; no operator key, node or network.

Test-only ephemeral RSA keys remain in memory. Real gpgv verifies their
OpenPGP signatures independently; the CLI never accepts their fingerprints.
"""

import copy
import datetime as dt
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
REPO = HERE.parent.parent
SPEC = importlib.util.spec_from_file_location("observer_release_verifier", HERE / "verify-release.py")
v = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(v)


class TestOpenPGP:
    """Independent RFC4880 RSA/SHA256 fixtures, as in test-seed-activation.py."""

    def __init__(self, label, epoch):
        import hashlib
        from cryptography.hazmat.primitives.asymmetric import rsa
        self.key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        self.epoch = epoch
        numbers = self.key.public_key().public_numbers()
        body = b"\x04" + epoch.to_bytes(4, "big") + b"\x01" + self.mpi(numbers.n) + self.mpi(numbers.e)
        material = b"\x99" + len(body).to_bytes(2, "big") + body
        self.fingerprint = hashlib.sha1(material).hexdigest().upper()
        uid = (label + "@synthetic.invalid").encode()
        certification = self.signature(material + b"\xb4" + len(uid).to_bytes(4, "big") + uid, 0x13)
        self.public = self.packet(6, body) + self.packet(13, uid) + certification

    @staticmethod
    def mpi(number):
        return number.bit_length().to_bytes(2, "big") + number.to_bytes((number.bit_length() + 7) // 8, "big")

    @staticmethod
    def packet(tag, body):
        return bytes([0xc0 | tag, 255]) + len(body).to_bytes(4, "big") + body

    def signature(self, message, signature_type=0):
        import hashlib
        from cryptography.hazmat.primitives import hashes
        from cryptography.hazmat.primitives.asymmetric import padding, utils
        hashed = b"\x05\x02" + self.epoch.to_bytes(4, "big") + b"\x16\x21\x04" + bytes.fromhex(self.fingerprint)
        header = bytes([4, signature_type, 1, 8]) + len(hashed).to_bytes(2, "big") + hashed
        digest = hashlib.sha256(message + header + b"\x04\xff" + len(header).to_bytes(4, "big")).digest()
        signature = self.key.sign(digest, padding.PKCS1v15(), utils.Prehashed(hashes.SHA256()))
        unhashed = b"\x09\x10" + bytes.fromhex(self.fingerprint[-16:])
        return self.packet(2, header + len(unhashed).to_bytes(2, "big") + unhashed + digest[:2] + self.mpi(int.from_bytes(signature, "big")))


def stamp(value):
    return value.isoformat(timespec="seconds").replace("+00:00", "Z")


class ReleaseVerificationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.gpgv = shutil.which("gpgv")
        if not cls.gpgv:
            raise RuntimeError("gpgv is required; cryptographic tests must not skip")
        cls.now = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
        epoch = int(cls.now.timestamp()) - 120
        cls.key = TestOpenPGP("observer-release-test", epoch)
        cls.other = TestOpenPGP("wrong-observer-key-test", epoch)

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="observer-release-test-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.bundle = self.root / "bundle"
        self.bundle.mkdir()
        for name, raw in {
            v.PUBLIC_KEY: self.key.public,
            "zeroned": b"synthetic test bytes, not an executable",
            "genesis.json": (REPO / "deploy/mainnet/artifacts/genesis.json").read_bytes(),
            "observer.py": b"# synthetic observer fixture, never executed\n",
            "verify-release.py": (HERE / "verify-release.py").read_bytes(),
            "verify-checkpoint": b"synthetic checkpoint verifier, never executed",
            "CHECKPOINT.json": b'{"test_only":"not chain evidence"}\n',
            "SOURCE-EVIDENCE.json": b'{"test_only":"not source provenance"}\n',
            "VULNERABILITY-REVIEW.json": b'{"test_only":"not a security assessment"}\n',
        }.items():
            (self.bundle / name).write_bytes(raw)
        self.manifest = {
            "schema": "zerone-1-legacy-observer-release/v1", "release_id": "zerone-1-observer-test-only",
            "created_at": stamp(self.now - dt.timedelta(minutes=3)),
            "expires_at": stamp(self.now + dt.timedelta(hours=72)), "expiry_scope": "bootstrap-only",
            "chain_id": "zerone-1", "role": "observer",
            "signature_authority": {"algorithm": "openpgp", "fingerprint": self.key.fingerprint,
                                    "signature_file": v.SIGNATURE, "public_key_file": v.PUBLIC_KEY},
            "capabilities": dict(v.CAPABILITIES), "runtime": dict(v.RUNTIME), "network": copy.deepcopy(v.NETWORK),
            "tooling": {"commit": "a1" * 20},
            "executable": {"file": "zeroned", "os": "linux", "arch": "amd64", "sha256": "", "size": 0,
                           "source": {"status": "unattributed", "commit": None, "evidence_file": "SOURCE-EVIDENCE.json"}},
            "checkpoint": {"file": "CHECKPOINT.json", "sha256": v.digest((self.bundle / "CHECKPOINT.json").read_bytes()),
                           "trust_period_seconds": v.TRUST_PERIOD}, "files": [],
        }
        self.refresh_inventory()

    def refresh_inventory(self):
        self.manifest["files"] = [
            {"name": path.name, "sha256": v.digest(path.read_bytes()), "size": path.stat().st_size}
            for path in sorted(self.bundle.iterdir()) if path.name not in {v.MANIFEST, v.SIGNATURE}
        ]
        binary = next(row for row in self.manifest["files"] if row["name"] == "zeroned")
        self.manifest["executable"].update({key: binary[key] for key in ("sha256", "size")})
        self.sign_manifest()

    def sign_manifest(self, key=None, raw=None):
        raw = raw if raw is not None else v.canonical(self.manifest)
        (self.bundle / v.MANIFEST).write_bytes(raw)
        (self.bundle / v.SIGNATURE).write_bytes((key or self.key).signature(raw))

    def verify(self, purpose="bootstrap", now=None):
        return v.verify(self.bundle, self.key.fingerprint, self.gpgv, purpose, now=now or self.now)

    def refuses(self, pattern=None):
        with self.assertRaisesRegex((v.Refusal, OSError), pattern or ".+"):
            self.verify()

    def test_real_gpgv_positive_and_readonly_receipt(self):
        before = {p.name: (p.stat().st_mode, p.read_bytes()) for p in self.bundle.iterdir()}
        result = self.verify()
        self.assertEqual(result["result"], "PASS")
        self.assertTrue(result["bootstrap_ready"])
        self.assertTrue(result["signature_verified"])
        self.assertFalse(result["checkpoint_semantics_verified"])
        self.assertFalse(result["source_reproduction_performed"])
        self.assertFalse(result["node_resume_authorized"])
        self.assertFalse(result["independent_chain_trust_established"])
        self.assertEqual(result["effects"], "none")
        self.assertEqual(before, {p.name: (p.stat().st_mode, p.read_bytes()) for p in self.bundle.iterdir()})

    def test_signed_tamper_wrong_key_missing_and_duplicate_signatures(self):
        self.manifest["release_id"] = "zerone-1-observer-tampered"
        (self.bundle / v.MANIFEST).write_bytes(v.canonical(self.manifest))
        self.refuses("signature")
        self.sign_manifest(key=self.other)
        self.refuses("signature")
        # Having both keys in the public keyring must not admit the wrong signer.
        (self.bundle / v.PUBLIC_KEY).write_bytes(self.key.public + self.other.public)
        self.refresh_inventory()
        self.sign_manifest(key=self.other)
        self.refuses("wrong signer")
        self.sign_manifest()
        sig = self.bundle / v.SIGNATURE
        sig.write_bytes(sig.read_bytes() * 2)
        self.refuses("exactly one")
        sig.write_bytes(b"malformed signature")
        self.refuses("signature")
        sig.unlink()
        self.refuses("file set")

    def test_bootstrap_expiry_and_resume_are_explicit(self):
        expiry = v.timestamp(self.manifest["expires_at"])
        with self.assertRaisesRegex(v.Refusal, "expired"):
            self.verify(now=expiry)
        resumed = self.verify("resume", now=expiry)
        self.assertEqual(resumed["result"], "PASS")
        self.assertFalse(resumed["bootstrap_ready"])
        self.assertFalse(resumed["node_resume_authorized"])
        self.assertEqual(resumed["purpose"], "resume")
        self.manifest["expires_at"] = stamp(self.now + dt.timedelta(days=8))
        self.sign_manifest()
        self.refuses("lifetime")
        self.manifest["expires_at"] = stamp(self.now + dt.timedelta(hours=1))
        self.manifest["created_at"] = stamp(self.now + dt.timedelta(minutes=6))
        self.sign_manifest()
        self.refuses("future")

    def test_strict_json_unknown_capabilities_and_scope(self):
        good = copy.deepcopy(self.manifest)
        raw = v.canonical(good)
        for bad in [raw.replace(b'{"capabilities":', b'{"chain_id":"zerone-2","capabilities":', 1),
                    raw + b"{}\n", b"[]\n", b'{"number":NaN}\n', raw.rstrip(b"\n")]:
            with self.subTest(raw=bad[:60]):
                self.sign_manifest(raw=bad)
                self.refuses()
        changes = [("chain_id", "zerone-2"), ("role", "validator"), ("schema", "zerone-2-release-packet-v2"),
                   ("expiry_scope", "all-node-operation"), ("unknown_command", "execute")]
        for key, value in changes:
            with self.subTest(key=key):
                self.manifest = copy.deepcopy(good)
                self.manifest[key] = value
                self.sign_manifest()
                self.refuses()
        for value in [True, 0, "false", None]:
            self.manifest = copy.deepcopy(good)
            self.manifest["capabilities"]["transactions"] = value
            self.sign_manifest()
            self.refuses("effects")

    def test_source_levels_are_assertions_with_exact_nullable_shape(self):
        self.manifest["executable"]["source"].update(status="reproduced-exact", commit="b2" * 20)
        self.sign_manifest()
        result = self.verify()
        self.assertFalse(result["source_reproduction_performed"])
        for status, commit in [("unattributed", "b2" * 20), ("reproduced-exact", None),
                               ("verified-production", "b2" * 20), ("reproduced-exact", "main")]:
            with self.subTest(status=status, commit=commit):
                self.manifest["executable"]["source"].update(status=status, commit=commit)
                self.sign_manifest()
                self.refuses()

    def test_patched_observer_binds_distinct_base_patch_and_security_review(self):
        (self.bundle / "observer-dependencies.patch").write_bytes(b"synthetic dependency patch, never applied")
        self.manifest["executable"]["source"] = {
            "status": "reproduced-observer-patch", "base_commit": "b2" * 20,
            "patch_file": "observer-dependencies.patch", "evidence_file": "SOURCE-EVIDENCE.json"}
        self.refresh_inventory()
        result = self.verify()
        self.assertEqual(result["executable_source_assertion"]["base_commit"], "b2" * 20)
        self.assertNotIn("commit", result["executable_source_assertion"])
        self.assertFalse(result["source_reproduction_performed"])
        good = copy.deepcopy(self.manifest)
        for mutate in [lambda p: p["executable"]["source"].update(base_commit=None),
                       lambda p: p["executable"]["source"].update(commit="b2" * 20),
                       lambda p: p["executable"]["source"].update(patch_file="SOURCE-EVIDENCE.json")]:
            self.manifest = copy.deepcopy(good)
            mutate(self.manifest)
            self.sign_manifest()
            self.refuses()
        self.manifest = good
        (self.bundle / "VULNERABILITY-REVIEW.json").unlink()
        self.refresh_inventory()
        self.refuses("required artifact")

    def test_file_hash_size_inventory_and_no_symlink_inputs(self):
        binary = self.bundle / "zeroned"
        original = binary.read_bytes()
        binary.write_bytes(original[:-1] + b"X")
        self.refuses("hash or size")
        binary.write_bytes(original + b"longer")
        self.refuses("size")
        binary.write_bytes(original)
        self.manifest["files"].append(copy.deepcopy(self.manifest["files"][0]))
        self.sign_manifest()
        self.refuses("duplicate")
        self.refresh_inventory()
        (self.bundle / "unexpected.txt").write_text("unexpected")
        self.refuses("file set")
        (self.bundle / "unexpected.txt").unlink()
        other = self.root / "outside"
        other.write_bytes(original)
        binary.unlink()
        binary.symlink_to(other)
        self.refuses()
        binary.unlink()
        os.link(other, binary)
        self.refuses("single-link")

    def test_network_runtime_checkpoint_and_genesis_bindings(self):
        good = copy.deepcopy(self.manifest)
        mutations = [lambda p: p["network"].update(same_upstream=False),
                     lambda p: p["network"]["rpc_servers"].append("https://unreviewed.invalid"),
                     lambda p: p["runtime"].update(kind="host-native"),
                     lambda p: p["runtime"].update(base_image="debian:latest"),
                     lambda p: p["checkpoint"].update(sha256="a" * 64),
                     lambda p: p["checkpoint"].update(trust_period_seconds=999999999),
                     lambda p: p["files"][0].update(name="../escape"),
                     lambda p: p["files"].reverse(),
                     lambda p: p["executable"].update(size=True)]
        for mutate in mutations:
            self.manifest = copy.deepcopy(good)
            mutate(self.manifest)
            self.sign_manifest()
            self.refuses()
        self.manifest = good
        (self.bundle / "genesis.json").write_bytes(b"{}\n")
        self.refresh_inventory()
        self.refuses("genesis")

    def test_pre_authentication_resource_bounds(self):
        manifest = self.bundle / v.MANIFEST
        original = manifest.read_bytes()
        manifest.write_bytes(b"x" * (v.MAX_MANIFEST + 1))
        self.refuses("size")
        manifest.write_bytes(original)
        (self.bundle / v.SIGNATURE).write_bytes(b"x" * (v.MAX_SIGNATURE + 1))
        self.refuses("size")
        self.sign_manifest()
        self.manifest["files"][0]["size"] = v.MAX_ARTIFACT + 1
        self.sign_manifest()
        self.refuses("size")
        self.refresh_inventory()
        for i in range(v.MAX_FILES + 3):
            (self.bundle / f"extra-{i}").touch()
        self.refuses("too many")

    def test_signature_isolated_arguments_and_status_refusals(self):
        signature = (self.bundle / v.SIGNATURE).read_bytes()
        epoch = int(self.now.timestamp()) - 120
        valid = f"[GNUPG:] VALIDSIG {self.key.fingerprint} 2026-09-09 {epoch} 0 4 0 1 8 00 {self.key.fingerprint}\n"
        def run_status(output):
            def fake(argv, **kwargs):
                self.assertEqual(Path(argv[0]), Path(self.gpgv))
                self.assertIn("--homedir", argv)
                self.assertIn("--keyring", argv)
                self.assertIn("--", argv)
                self.assertEqual(kwargs["env"], {"PATH": "/usr/bin:/bin", "LC_ALL": "C"})
                self.assertEqual(kwargs["stdin"], subprocess.DEVNULL)
                self.assertEqual(kwargs["timeout"], 15)
                kwargs["stdout"].write(output.encode())
                return subprocess.CompletedProcess(argv, 0)
            with patch.object(v.subprocess, "run", side_effect=fake):
                return v.verify_signature(b"fixture", signature, self.key.public, self.key.fingerprint, self.gpgv, self.now)
        self.assertEqual(run_status(valid), epoch)
        for output in [valid * 2, valid.replace(" 8 00 ", " 2 00 "),
                       valid.replace(str(epoch), str(int(self.now.timestamp()) + 301)),
                       valid + "[GNUPG:] EXPKEYSIG expired\n", valid + "[GNUPG:] REVKEYSIG revoked\n"]:
            with self.subTest(output=output):
                with self.assertRaises(v.Refusal):
                    run_status(output)

    def test_cli_rejects_fixture_pin_override_even_for_valid_signed_bundle(self):
        result = subprocess.run([sys.executable, "-B", str(HERE / "verify-release.py"),
                                 "--bundle", str(self.bundle), "--expected-fingerprint", self.key.fingerprint,
                                 "--gpgv", self.gpgv], capture_output=True, text=True, check=False)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stdout, "")
        self.assertEqual(json.loads(result.stderr)["result"], "REFUSED")
        self.assertIn("established main", result.stderr)


if __name__ == "__main__":
    unittest.main(verbosity=2)
