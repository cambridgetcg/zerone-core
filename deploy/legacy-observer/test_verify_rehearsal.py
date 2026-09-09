#!/usr/bin/env python3
"""Synthetic signed rehearsal tests; no production authority, Docker or network."""
import copy
import datetime as dt
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


r = module("rehearsal_verifier", HERE / "verify-rehearsal.py")
fixtures = module("release_fixtures", HERE / "test_verify_release.py")


class RehearsalTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        fixtures.ReleaseVerificationTests.setUpClass()

    def setUp(self):
        self.fixture = fixtures.ReleaseVerificationTests()
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.bundle, self.root, self.now = self.fixture.bundle, self.fixture.root, self.fixture.now
        self.checkpoint = {"schema": "zerone-1-observer-checkpoint/v1", "chain_id": "zerone-1",
                           "height": 100, "block_hash": "A1" * 32}
        self.install_checkpoint()
        self.archive = self.root / "release.tar.gz"
        self.archive.write_bytes(b"synthetic archive bytes; digest only, never extracted")
        self.receipt = self.root / "REHEARSAL-RECEIPT.json"
        self.signature = self.root / "REHEARSAL-RECEIPT.json.sig"
        self.value = {
            "schema": "zerone-1-observer-rehearsal/v1", "scope": "operator-attestation", "platform": "linux/amd64",
            "release_id": self.fixture.manifest["release_id"],
            "manifest_sha256": r.release.digest((self.bundle / r.release.MANIFEST).read_bytes()),
            "archive_sha256": r.release.digest(self.archive.read_bytes()),
            "completed_at": fixtures.stamp(self.now - dt.timedelta(seconds=130)),
            "checkpoint": {"height": 100, "block_hash": "A1" * 32},
            "following": {"height": 110, "block_hash": "B2" * 32, "header_app_hash": "c3" * 32,
                          "applied_height": 110, "abci_last_block_app_hash": "d4" * 32},
            "restart": {"before_height": 110, "after_height": 112},
            "state_root_binding": {"application_height": 110, "post_commit_app_hash": "d4" * 32,
                                   "binding_header_height": 111, "binding_header_app_hash": "d4" * 32,
                                   "binding_header_block_hash": "E5" * 32},
            "checks": {name: True for name in r.CHECKS},
        }
        self.sign()

    def install_checkpoint(self, raw=None):
        raw = raw or (json.dumps(self.checkpoint, indent=2) + "\n").encode()
        (self.bundle / "CHECKPOINT.json").write_bytes(raw)
        self.fixture.manifest["checkpoint"]["sha256"] = r.release.digest(raw)
        self.fixture.refresh_inventory()

    def sign(self, key=None, raw=None):
        raw = raw if raw is not None else r.release.canonical(self.value)
        self.receipt.write_bytes(raw)
        self.signature.write_bytes((key or self.fixture.key).signature(raw))

    def verify(self, **changes):
        return r.verify(self.bundle, self.archive, self.receipt, self.signature, self.fixture.gpgv,
                        expected=self.fixture.key.fingerprint, now=changes.get("now", self.now))

    def refuses(self, pattern=".+"):
        with self.assertRaisesRegex((ValueError, OSError), pattern):
            self.verify()

    def test_real_signature_and_exact_hashes_with_no_input_changes(self):
        paths = list(self.bundle.iterdir()) + [self.archive, self.receipt, self.signature]
        before = {str(p): (p.read_bytes(), p.stat().st_mode) for p in paths}
        result = self.verify()
        self.assertEqual(result["result"], "PASS")
        self.assertTrue(result["bootstrap_ready"])
        self.assertFalse(result["rehearsal_repeated"])
        self.assertFalse(result["independent_chain_trust_established"])
        self.assertFalse(result["checkpoint_semantics_verified"])
        self.assertFalse(result["production_authority_granted"])
        self.assertEqual(result["effects"], "none")
        self.assertEqual(result["receipt_sha256"], r.release.digest(self.receipt.read_bytes()))
        self.assertEqual(set(result["checks_attested"]), r.CHECKS)
        self.assertEqual(before, {str(p): (p.read_bytes(), p.stat().st_mode) for p in paths})

    def test_signed_wrong_manifest_archive_release_checkpoint_and_platform_refuse(self):
        good = copy.deepcopy(self.value)
        for field, value in [("manifest_sha256", "f6" * 32), ("archive_sha256", "f6" * 32),
                             ("release_id", "zerone-1-observer-other"), ("platform", "linux/arm64"),
                             ("scope", "production-authorization")]:
            with self.subTest(field=field):
                self.value = copy.deepcopy(good)
                self.value[field] = value
                self.sign()
                self.refuses()
        self.value = copy.deepcopy(good)
        self.value["checkpoint"]["height"] += 1
        self.sign()
        self.refuses("checkpoint differs")

    def test_every_named_check_is_required_boolean_true_no_wildcards(self):
        good = copy.deepcopy(self.value)
        for name in sorted(r.CHECKS):
            for bad in (False, 1, "true", None):
                with self.subTest(check=name, value=bad):
                    self.value = copy.deepcopy(good)
                    self.value["checks"][name] = bad
                    self.sign()
                    self.refuses("check")
        self.value = copy.deepcopy(good)
        del self.value["checks"]["zero_validator_power"]
        self.value["checks"]["all_checks_passed"] = True
        self.sign()
        self.refuses("fields")

    def test_heights_h_plus_one_and_exact_normalized_roots(self):
        good = copy.deepcopy(self.value)
        changes = [("state_root_binding", "binding_header_height", 110),
                   ("state_root_binding", "binding_header_app_hash", "f6" * 32),
                   ("state_root_binding", "post_commit_app_hash", "D4" * 32),
                   ("following", "abci_last_block_app_hash", "Zm9v"),
                   ("following", "abci_last_block_app_hash", "f6" * 32),
                   ("following", "applied_height", 109), ("following", "height", True),
                   ("restart", "after_height", 110), ("restart", "before_height", 100)]
        for section, field, value in changes:
            with self.subTest(section=section, field=field):
                self.value = copy.deepcopy(good)
                self.value[section][field] = value
                self.sign()
                self.refuses()
        self.value = copy.deepcopy(good)
        self.value["following"].update(height=111, applied_height=111)
        self.sign()
        self.refuses("conflicting observed binding header")

    def test_expired_bundle_or_completion_and_signature_order_refuse(self):
        expires = r.release.timestamp(self.fixture.manifest["expires_at"])
        with self.assertRaisesRegex(ValueError, "expired"):
            self.verify(now=expires)
        for completed in (self.now + dt.timedelta(seconds=1), self.now - dt.timedelta(days=1),
                          self.now - dt.timedelta(seconds=110)):
            self.value["completed_at"] = fixtures.stamp(completed)
            self.sign()
            self.refuses("window")

    def test_root_equality_is_independent_of_following_observation(self):
        # Neither conditional following/root cross-check applies: equality
        # must be enforced by the H/H+1 pair itself.
        self.value["following"].update(height=120, applied_height=120)
        self.sign()
        self.assertEqual(self.verify()["result"], "PASS")
        self.value["state_root_binding"]["binding_header_app_hash"] = "f6" * 32
        self.sign()
        self.refuses("post-commit application root differs from header H\\+1")

    def test_invalid_signatures_and_post_signature_mutations_refuse(self):
        self.sign(key=self.fixture.other)
        self.refuses("signature")
        self.sign()
        self.signature.write_bytes(self.signature.read_bytes() * 2)
        self.refuses("exactly one")
        self.sign()
        self.value["restart"]["after_height"] += 1
        self.receipt.write_bytes(r.release.canonical(self.value))
        self.refuses("signature")
        self.sign()
        self.archive.write_bytes(self.archive.read_bytes() + b"changed")
        self.refuses("archive digest")

    def test_duplicate_unknown_and_oversized_json_refuse(self):
        good = r.release.canonical(self.value)
        for raw in (good.replace(b'{"archive_sha256":', b'{"scope":"other","archive_sha256":', 1),
                    good + b"{}\n", b'{"bad":NaN}\n', good.rstrip(), b" " * (r.release.MAX_MANIFEST + 1)):
            self.sign(raw=raw)
            self.refuses()
        self.value["authorization"] = "all future observers"
        self.sign()
        self.refuses("fields")

    def test_symlinks_hardlinks_bounds_and_mid_verification_drift_refuse(self):
        self.archive.rename(self.root / "real-archive")
        self.archive.symlink_to(self.root / "real-archive")
        self.refuses()
        self.archive.unlink()
        self.archive.hardlink_to(self.root / "real-archive")
        self.refuses("single-link")
        self.archive.unlink()
        self.archive.write_bytes((self.root / "real-archive").read_bytes())
        with patch.object(r.release, "MAX_TOTAL", 1):
            self.refuses("bound")
        original = r.Inputs.unchanged
        def mutate(inputs):
            self.archive.write_bytes(self.archive.read_bytes() + b"drift")
            original(inputs)
        with patch.object(r.Inputs, "unchanged", mutate):
            self.refuses("changed after reading")

    def test_cli_has_no_non_main_pin_or_resume_escape(self):
        argv = [sys.executable, "-I", "-B", str(HERE / "verify-rehearsal.py"), "--bundle", str(self.bundle),
                "--archive", str(self.archive), "--receipt", str(self.receipt), "--signature", str(self.signature),
                "--gpgv", self.fixture.gpgv]
        result = subprocess.run(argv, capture_output=True, text=True, check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("signature authority", result.stderr)
        result = subprocess.run(argv + ["--purpose", "resume"], capture_output=True, text=True, check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unrecognized arguments", result.stderr)


if __name__ == "__main__":
    unittest.main(verbosity=2)
