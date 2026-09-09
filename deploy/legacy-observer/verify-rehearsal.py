#!/usr/bin/env python3
"""Verify an operator-signed observer rehearsal against an exact release.

Run this script and adjacent verify-release.py from an independently pinned
tooling checkout. The receipt is separate from the release archive to avoid
a circular manifest. No network, extraction, node or signing operation occurs.
Its signature authenticates operator testimony; this command does not repeat
the rehearsal or establish independent chain trust.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys

HERE = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("observer_release", HERE / "verify-release.py")
release = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release)
require = release.require
exact = release.exact
CHECKS = frozenset({
    "fresh_home_sync", "restart_retained_state", "zero_validator_power", "no_local_signatures",
    "broadcast_tx_async_refused", "broadcast_tx_sync_refused", "broadcast_tx_commit_refused",
    "application_state_h_to_h_plus_1", "source_changes_dependency_only", "production_authority_unchanged",
})


class Inputs:
    """Bounded no-follow inputs, retained descriptors and path identity checks."""

    def __init__(self):
        self.opened = []
        self.pins = {}

    def read(self, label, path, limit, retain=True):
        path = Path(path).absolute()
        descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
        self.opened.append((path, descriptor, os.fstat(descriptor)))
        before = self.opened[-1][2]
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1,
                label + " must be a regular single-link file")
        require(0 < before.st_size <= limit, label + " size bound")
        checksum, chunks, size = hashlib.sha256(), [], 0
        while chunk := os.read(descriptor, min(1024 * 1024, limit + 1 - size)):
            size += len(chunk)
            require(size <= limit, label + " grew past bound")
            checksum.update(chunk)
            if retain:
                chunks.append(chunk)
        require(size == before.st_size and release.stable(before) == release.stable(os.fstat(descriptor)),
                label + " changed while reading")
        self.pins[label] = {"sha256": checksum.hexdigest(), "size": size}
        return b"".join(chunks) if retain else self.pins[label]

    def unchanged(self):
        for path, descriptor, before in self.opened:
            require(release.stable(before) == release.stable(os.fstat(descriptor)) ==
                    release.stable(path.stat(follow_symlinks=False)), "rehearsal input changed after reading")

    def close(self):
        for _, descriptor, _ in self.opened:
            os.close(descriptor)


def height(value):
    require(type(value) is int and 0 < value < 10 ** 18, "invalid rehearsal height")
    return value


def block_hash(value):
    require(type(value) is str and re.fullmatch(r"[0-9A-F]{64}", value) and value != "0" * 64,
            "invalid uppercase block hash")
    return value


def parse_checkpoint(raw):
    # Signed checkpoint files may be indented JSON. Preserve duplicate/NaN
    # refusal without imposing the manifest/receipt serialization contract.
    def pairs(items):
        value = {}
        for name, item in items:
            require(name not in value, "duplicate checkpoint JSON key")
            value[name] = item
        return value
    def invalid(_):
        raise release.Refusal("non-finite checkpoint JSON")
    try:
        value = json.loads(raw, object_pairs_hook=pairs, parse_constant=invalid)
        require(type(value) is dict, "checkpoint must be an object")
        return value
    except (UnicodeError, json.JSONDecodeError, RecursionError) as error:
        raise release.Refusal("invalid checkpoint JSON") from error


def validate_receipt(value, manifest, checkpoint, now):
    exact(value, {"schema", "scope", "platform", "release_id", "manifest_sha256", "archive_sha256",
                  "completed_at", "checkpoint", "following", "restart", "state_root_binding", "checks"}, "rehearsal")
    require(value["schema"] == "zerone-1-observer-rehearsal/v1" and
            value["scope"] == "operator-attestation" and value["platform"] == "linux/amd64",
            "wrong rehearsal scope or platform")
    require(value["release_id"] == manifest["release_id"], "rehearsal release ID differs")
    release.hash_value(value["manifest_sha256"])
    release.hash_value(value["archive_sha256"])
    completed = release.timestamp(value["completed_at"])
    created, expires = release.timestamp(manifest["created_at"]), release.timestamp(manifest["expires_at"])
    require(created <= completed < expires and completed <= now, "rehearsal completion outside current release window")
    checks = exact(value["checks"], CHECKS, "rehearsal checks")
    require(all(checks[name] is True for name in CHECKS), "every required rehearsal check must be true")
    pinned = exact(value["checkpoint"], {"height", "block_hash"}, "rehearsal checkpoint")
    height(pinned["height"])
    block_hash(pinned["block_hash"])
    require(pinned == {name: checkpoint.get(name) for name in pinned}, "rehearsal checkpoint differs from signed release")
    following = exact(value["following"], {"height", "block_hash", "header_app_hash", "applied_height",
                                          "abci_last_block_app_hash"}, "following observation")
    require(height(following["applied_height"]) >= height(following["height"]) > pinned["height"],
            "following observation must include applied state beyond checkpoint")
    block_hash(following["block_hash"])
    for field in ("header_app_hash", "abci_last_block_app_hash"):
        release.hash_value(following[field])
    restart = exact(value["restart"], {"before_height", "after_height"}, "restart observation")
    require(height(restart["after_height"]) > height(restart["before_height"]) > pinned["height"],
            "restart must retain state and advance beyond checkpoint")
    binding = exact(value["state_root_binding"], {"application_height", "post_commit_app_hash", "binding_header_height",
                                                "binding_header_app_hash", "binding_header_block_hash"}, "state root binding")
    require(height(binding["binding_header_height"]) == height(binding["application_height"]) + 1 and
            binding["application_height"] >= pinned["height"], "state root binding requires header H+1")
    release.hash_value(binding["post_commit_app_hash"])
    release.hash_value(binding["binding_header_app_hash"])
    require(binding["post_commit_app_hash"] == binding["binding_header_app_hash"],
            "post-commit application root differs from header H+1")
    block_hash(binding["binding_header_block_hash"])
    if following["applied_height"] == binding["application_height"]:
        require(following["abci_last_block_app_hash"] == binding["post_commit_app_hash"], "conflicting observed application root")
    if following["height"] == binding["binding_header_height"]:
        require(following["header_app_hash"] == binding["binding_header_app_hash"] and
                following["block_hash"] == binding["binding_header_block_hash"], "conflicting observed binding header")
    return completed, expires


def verify(bundle_path, archive_path, receipt_path, signature_path, gpgv, *,
           expected=release.MAIN_FINGERPRINT, now=None):
    """Explicit library trust pin supports synthetic tests; CLI always pins main."""
    now = now or dt.datetime.now(dt.timezone.utc)
    require(now.tzinfo is not None, "verification clock must include timezone")
    bundle_result = release.verify(bundle_path, expected, gpgv, "bootstrap", now=now)
    bundle, inputs = release.Bundle(bundle_path), Inputs()
    try:
        raw_manifest = bundle.read(release.MANIFEST, release.MAX_MANIFEST)
        manifest = release.parse(raw_manifest)
        require(release.digest(raw_manifest) == bundle_result["manifest_sha256"], "release changed before rehearsal verification")
        public_key = bundle.read(release.PUBLIC_KEY, release.MAX_PUBLIC_KEY)
        checkpoint_name = manifest["checkpoint"]["file"]
        raw_checkpoint = bundle.read(checkpoint_name, release.MAX_MANIFEST)
        for name in (release.PUBLIC_KEY, checkpoint_name):
            require(bundle.pins[name] == bundle_result["input_sha256_and_size"][name], "release artifact changed before rehearsal verification")
        # The checkpoint is already an exact signed file. Semantic signature
        # verification still belongs to the separate checkpoint verifier.
        checkpoint = parse_checkpoint(raw_checkpoint)
        require(checkpoint.get("schema") == "zerone-1-observer-checkpoint/v1" and
                checkpoint.get("chain_id") == "zerone-1", "wrong checkpoint scope")
        raw_receipt = inputs.read("receipt", receipt_path, release.MAX_MANIFEST)
        signature = inputs.read("signature", signature_path, release.MAX_SIGNATURE)
        receipt = release.parse(raw_receipt)
        completed, expires = validate_receipt(receipt, manifest, checkpoint, now)
        require(receipt["manifest_sha256"] == bundle_result["manifest_sha256"], "receipt manifest digest differs")
        signature_epoch = release.verify_signature(raw_receipt, signature, public_key, expected, gpgv, now)
        require(completed.timestamp() <= signature_epoch < expires.timestamp(), "receipt signature outside rehearsal window")
        archive_pin = inputs.read("archive", archive_path, release.MAX_TOTAL, retain=False)
        require(archive_pin["sha256"] == receipt["archive_sha256"], "receipt archive digest differs")
        inputs.unchanged()
        bundle.unchanged(set(bundle_result["input_sha256_and_size"]))
        return {
            "schema": "zerone-1-observer-rehearsal-verification/v1", "result": "PASS", "scope": "operator-attestation",
            "release_id": manifest["release_id"], "platform": "linux/amd64", "signer_fingerprint": expected,
            "manifest_sha256": bundle_result["manifest_sha256"], "archive_sha256": archive_pin["sha256"],
            "receipt_sha256": release.digest(raw_receipt), "signature_sha256": release.digest(signature),
            "verified_at": now.isoformat().replace("+00:00", "Z"), "bootstrap_ready": True,
            "expires_at": manifest["expires_at"], "input_sha256_and_size": inputs.pins,
            "bundle_input_sha256_and_size": bundle_result["input_sha256_and_size"],
            "checks_attested": sorted(CHECKS), "rehearsal_repeated": False,
            "checkpoint_semantics_verified": False, "independent_chain_trust_established": False,
            "production_authority_granted": False, "effects": "none",
            "limitations": ["The main key attests to recorded tests; this verifier does not reproduce them.",
                            "The archive is digest-bound to the attestation; this command does not extract or compare its contents.",
                            "Checkpoint semantics and fresh observer initialization remain separately verified.",
                            "No validator, account, transaction, upgrade or independent-trust authority is granted."],
        }
    finally:
        inputs.close()
        bundle.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("bundle", "archive", "receipt", "signature"):
        parser.add_argument("--" + name, required=True, type=Path)
    parser.add_argument("--gpgv", required=True)
    args = parser.parse_args()
    try:
        print(release.canonical(verify(args.bundle, args.archive, args.receipt, args.signature, args.gpgv)).decode(), end="")
    except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        print(release.canonical({"result": "REFUSED", "reason": str(error), "effects": "none"}).decode(), file=sys.stderr, end="")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
