#!/usr/bin/env python3
"""Offline verification of an operator-signed zerone-1 observer package.

No network, extraction, node execution, account/validator action, or private
key access. A release signature authenticates the operator's exact statement;
it does not establish executable source provenance or independent chain trust.
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile

MAIN_FINGERPRINT = "09327B031F8FF2C2EE49B18F2234027FC5B68C19"
MANIFEST = "OBSERVER-RELEASE.json"
SIGNATURE = MANIFEST + ".sig"
PUBLIC_KEY = "main-public.gpg"
MAX_MANIFEST = 256 * 1024
MAX_SIGNATURE = 16 * 1024
MAX_PUBLIC_KEY = 64 * 1024
MAX_ARTIFACT = 384 * 1024 * 1024
MAX_TOTAL = 512 * 1024 * 1024
MAX_FILES = 64
MAX_LIFETIME = 7 * 24 * 60 * 60
TRUST_PERIOD = 168 * 60 * 60
GENESIS_SHA256 = "c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e"
RUNTIME = {
    "kind": "docker",
    "base_image": "docker.io/library/debian@sha256:5ae3c39ebd15e229dcedd5cee596b2497182493d41ff162e824ba13fc1b2b867",
    "platform": "linux/amd64",
}
NETWORK = {
    "peer": "ed8c8d49dc23f3478b2f3eddb49b8f8087828b6e@169.155.55.44:26656",
    "rpc_servers": ["http://zerone-1.fly.dev:26657", "http://169.155.55.44:26657"],
    "same_upstream": True,
}
CAPABILITIES = {name: False for name in
                ("validator", "transactions", "account_admission", "upgrade", "public_listener")}
HASH = re.compile(r"[0-9a-f]{64}\Z")
COMMIT = re.compile(r"[0-9a-f]{40}\Z")
FILENAME = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}\Z")


class Refusal(ValueError):
    """A bounded verification condition failed."""


def require(condition, reason):
    if not condition:
        raise Refusal(reason)


def exact(value, names, label):
    require(type(value) is dict and set(value) == set(names), label + " fields")
    return value


def digest(value):
    return hashlib.sha256(value).hexdigest()


def canonical(value):
    return (json.dumps(value, sort_keys=True, separators=(",", ":"),
                       ensure_ascii=False, allow_nan=False) + "\n").encode("utf-8")


def parse(raw):
    def pairs(items):
        value = {}
        for key, item in items:
            require(key not in value, "duplicate JSON key")
            value[key] = item
        return value

    def reject_constant(_):
        raise Refusal("non-finite JSON value")

    try:
        value = json.loads(raw, object_pairs_hook=pairs, parse_constant=reject_constant)
        require(type(value) is dict, "manifest must be one object")
        require(canonical(value) == raw, "manifest is not exact canonical JSON with one LF")
        return value
    except (UnicodeError, json.JSONDecodeError, RecursionError) as error:
        raise Refusal("invalid or excessively nested JSON") from error


def filename(value):
    require(type(value) is str and FILENAME.fullmatch(value), "unsafe bundle filename")
    return value


def hash_value(value):
    require(type(value) is str and HASH.fullmatch(value) and value != "0" * 64,
            "invalid SHA-256")
    return value


def timestamp(value):
    require(type(value) is str and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", value),
            "timestamp must be UTC seconds")
    try:
        return dt.datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=dt.timezone.utc)
    except ValueError as error:
        raise Refusal("invalid timestamp") from error


def stable(info):
    return (info.st_dev, info.st_ino, info.st_mode, info.st_size,
            info.st_mtime_ns, info.st_ctime_ns, info.st_nlink)


class Bundle:
    """One open directory; flat, bounded regular files, stable throughout reads."""

    def __init__(self, path):
        require(hasattr(os, "O_NOFOLLOW"), "platform lacks no-symlink file opens")
        self.fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        self.total = 0
        self.pins = {}
        self.stats = {}

    def close(self):
        os.close(self.fd)

    def names(self):
        names = set()
        with os.scandir(self.fd) as entries:
            for entry in entries:
                names.add(entry.name)
                require(len(names) <= MAX_FILES + 2, "too many bundle directory entries")
        return names

    def read(self, name, limit, retain=True):
        filename(name)
        fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=self.fd)
        with os.fdopen(fd, "rb") as stream:
            before = os.fstat(stream.fileno())
            require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1,
                    "bundle inputs must be regular single-link files")
            require(0 < before.st_size <= limit, "bundle input size bound")
            require(self.total + before.st_size <= MAX_TOTAL, "bundle total size bound")
            pieces = []
            checksum = hashlib.sha256()
            length = 0
            while chunk := stream.read(min(1024 * 1024, limit + 1 - length)):
                length += len(chunk)
                require(length <= limit, "bundle input grew past bound")
                checksum.update(chunk)
                if retain:
                    pieces.append(chunk)
            require(length == before.st_size and stable(before) == stable(os.fstat(stream.fileno())),
                    "bundle input changed while reading")
        self.total += length
        self.pins[name] = {"sha256": checksum.hexdigest(), "size": length}
        self.stats[name] = stable(before)
        return b"".join(pieces) if retain else self.pins[name]

    def unchanged(self, expected_names):
        require(self.names() == expected_names, "bundle directory changed during verification")
        for name, before in self.stats.items():
            after = os.stat(name, dir_fd=self.fd, follow_symlinks=False)
            require(stable(after) == before, "bundle input changed after reading")


def verify_signature(payload, signature, public_key, expected, gpgv, now):
    """Frozen public bytes, explicit executable, and no ambient keyring/config."""
    require(Path(gpgv).is_absolute(), "gpgv executable must be an explicit absolute path")
    require(0 < len(signature) <= MAX_SIGNATURE, "signature size bound")
    require(0 < len(public_key) <= MAX_PUBLIC_KEY, "public key size bound")
    require(re.fullmatch(r"(?:[0-9A-F]{40}|[0-9A-F]{64})", expected), "full signer fingerprint required")
    with tempfile.TemporaryDirectory(prefix="zerone-observer-public-verify-") as temporary:
        root = Path(temporary)
        for name, raw in ((PUBLIC_KEY, public_key), (MANIFEST, payload), (SIGNATURE, signature)):
            path = root / name
            path.write_bytes(raw)
            path.chmod(0o600)
        # Directing output to a private regular file avoids an unbounded pipe
        # capture. gpgv has only bounded signature/key inputs and a time limit.
        with (root / "status").open("w+b") as status:
            result = subprocess.run(
                [gpgv, "--homedir", str(root), "--keyring", str(root / PUBLIC_KEY),
                 "--status-fd=1", "--", str(root / SIGNATURE), str(root / MANIFEST)],
                stdin=subprocess.DEVNULL, stdout=status, stderr=subprocess.DEVNULL,
                timeout=15, env={"PATH": "/usr/bin:/bin", "LC_ALL": "C"}, check=False)
            require(result.returncode == 0, "detached signature rejected")
            require(status.tell() <= MAX_SIGNATURE, "signature status output bound")
            status.seek(0)
            output = status.read(MAX_SIGNATURE + 1)
    try:
        lines = output.decode("ascii", errors="strict").splitlines()
    except UnicodeError as error:
        raise Refusal("invalid signature status encoding") from error
    valid = [line.split() for line in lines if line.startswith("[GNUPG:] VALIDSIG ")]
    require(len(valid) == 1 and len(valid[0]) >= 11, "exactly one valid signature required")
    fields = valid[0]
    require(fields[2] == expected, "signature has wrong signer")
    require(fields[4].isdigit() and 0 < int(fields[4]) <= now.timestamp() + 300,
            "invalid or future signature timestamp")
    require(fields[9] in {"8", "9", "10", "11"}, "signature must use SHA-2")
    require(not any(any(tag in line for tag in
                        ("BADSIG", "ERRSIG", "EXPKEYSIG", "EXPSIG", "REVKEYSIG"))
                    for line in lines), "invalid, expired or revoked signature/key")
    return int(fields[4])


def validate_manifest(value, expected, now, purpose):
    exact(value, {"schema", "release_id", "created_at", "expires_at", "expiry_scope",
                  "chain_id", "role", "signature_authority", "capabilities", "executable",
                  "tooling", "checkpoint", "runtime", "network", "files"}, "manifest")
    require(value["schema"] == "zerone-1-legacy-observer-release/v1", "wrong release schema")
    require(value["chain_id"] == "zerone-1" and value["role"] == "observer", "wrong release scope")
    require(type(value["release_id"]) is str and
            re.fullmatch(r"zerone-1-observer-[A-Za-z0-9][A-Za-z0-9._-]{0,95}", value["release_id"]),
            "invalid observer release ID")
    require(value["expiry_scope"] == "bootstrap-only", "wrong expiry scope")
    created, expires = timestamp(value["created_at"]), timestamp(value["expires_at"])
    require(0 < (expires - created).total_seconds() <= MAX_LIFETIME, "bootstrap lifetime bound")
    require(created.timestamp() <= now.timestamp() + 300, "manifest creation is in the future")
    require(purpose in {"bootstrap", "resume"}, "unknown verification purpose")
    bootstrap_ready = now < expires
    require(purpose == "resume" or bootstrap_ready, "bootstrap window expired")

    authority = exact(value["signature_authority"],
                      {"algorithm", "fingerprint", "signature_file", "public_key_file"}, "signature authority")
    require(authority == {"algorithm": "openpgp", "fingerprint": expected,
                          "signature_file": SIGNATURE, "public_key_file": PUBLIC_KEY},
            "wrong release signature authority")
    exact(value["capabilities"], CAPABILITIES, "capabilities")
    require(all(value["capabilities"][name] is False for name in CAPABILITIES),
            "observer package cannot authorize effects or public listeners")
    exact(value["runtime"], RUNTIME, "runtime")
    require(value["runtime"] == RUNTIME, "unsupported observer runtime")
    exact(value["network"], NETWORK, "network")
    require(value["network"]["same_upstream"] is True and value["network"] == NETWORK,
            "unsupported observer network or independence claim")

    tooling = exact(value["tooling"], {"commit"}, "tooling")
    require(type(tooling["commit"]) is str and COMMIT.fullmatch(tooling["commit"]) and
            tooling["commit"] != "0" * 40, "invalid tooling source commit")
    executable = exact(value["executable"], {"file", "sha256", "size", "os", "arch", "source"}, "executable")
    require(executable["file"] == "zeroned" and executable["os"] == "linux" and
            executable["arch"] == "amd64", "unsupported observer executable")
    hash_value(executable["sha256"])
    require(type(executable["size"]) is int and 0 < executable["size"] <= MAX_ARTIFACT,
            "executable size bound")
    source = executable["source"]
    require(type(source) is dict, "executable source must be an object")
    patched = source.get("status") == "reproduced-observer-patch"
    exact(source, {"status", "base_commit", "patch_file", "evidence_file"} if patched else
          {"status", "commit", "evidence_file"}, "executable source")
    # This is a signed assertion to be assessed with the named evidence. The
    # verifier does not build the source or claim to prove reproducibility.
    if source["status"] == "unattributed":
        require(source["commit"] is None, "unattributed executable must not claim a source commit")
    elif source["status"] == "reproduced-exact":
        require(type(source["commit"]) is str and COMMIT.fullmatch(source["commit"]) and
                source["commit"] != "0" * 40, "reproduced executable needs an exact source commit")
    elif patched:
        require(type(source["base_commit"]) is str and COMMIT.fullmatch(source["base_commit"]) and
                source["base_commit"] != "0" * 40, "patched observer needs an exact base source commit")
        require(source["patch_file"] == "observer-dependencies.patch", "patched observer needs the named dependency patch")
    else:
        raise Refusal("unknown executable provenance level")
    evidence = filename(source["evidence_file"])
    checkpoint = exact(value["checkpoint"], {"file", "sha256", "trust_period_seconds"}, "checkpoint")
    filename(checkpoint["file"])
    hash_value(checkpoint["sha256"])
    require(type(checkpoint["trust_period_seconds"]) is int and
            checkpoint["trust_period_seconds"] == TRUST_PERIOD, "unsupported trust period")

    files = value["files"]
    require(type(files) is list and 6 <= len(files) <= MAX_FILES, "artifact count bound")
    inventory = {}
    total = 0
    for row in files:
        exact(row, {"name", "sha256", "size"}, "file inventory")
        name = filename(row["name"])
        require(name not in {MANIFEST, SIGNATURE} and name not in inventory, "duplicate or self-referencing artifact")
        require(type(row["size"]) is int and 0 < row["size"] <= MAX_ARTIFACT, "artifact size bound")
        hash_value(row["sha256"])
        total += row["size"]
        require(total <= MAX_TOTAL, "artifact total size bound")
        inventory[name] = {"sha256": row["sha256"], "size": row["size"]}
    require(list(inventory) == sorted(inventory), "artifact inventory must be sorted")
    mandatory = {PUBLIC_KEY, "zeroned", "genesis.json", "observer.py", "verify-release.py", "verify-checkpoint",
                 "VULNERABILITY-REVIEW.json",
                 checkpoint["file"], evidence}
    if patched:
        mandatory.add(source["patch_file"])
    require(mandatory <= set(inventory), "required artifact absent from signed inventory")
    require(checkpoint["file"] not in {PUBLIC_KEY, "zeroned", "genesis.json", "observer.py", "verify-release.py", "verify-checkpoint",
                                       "VULNERABILITY-REVIEW.json", "observer-dependencies.patch"} and
            evidence not in {PUBLIC_KEY, "zeroned", "genesis.json", "observer.py", "verify-release.py", "verify-checkpoint",
                            "VULNERABILITY-REVIEW.json", "observer-dependencies.patch", checkpoint["file"]},
            "checkpoint and provenance evidence must be distinct artifacts")
    require(inventory["zeroned"] == {"sha256": executable["sha256"], "size": executable["size"]},
            "executable inventory binding mismatch")
    require(inventory["genesis.json"]["sha256"] == GENESIS_SHA256, "wrong zerone-1 genesis digest")
    require(inventory[checkpoint["file"]]["sha256"] == checkpoint["sha256"], "checkpoint inventory binding mismatch")
    require(inventory[PUBLIC_KEY]["size"] <= MAX_PUBLIC_KEY, "public key size bound")
    require(inventory["VULNERABILITY-REVIEW.json"]["size"] <= MAX_MANIFEST, "vulnerability review size bound")
    if patched:
        require(inventory[source["patch_file"]]["size"] <= MAX_MANIFEST, "dependency patch size bound")
    return inventory, created, expires, bootstrap_ready


def verify(bundle_path, expected, gpgv, purpose="bootstrap", *, now=None):
    """Library entrypoint; callers supply a trust pin, never take it from JSON.

    The public CLI below additionally fixes the established Zerone main key.
    An explicit library pin permits isolated synthetic cryptographic tests.
    """
    now = now or dt.datetime.now(dt.timezone.utc)
    require(now.tzinfo is not None, "verification clock must include timezone")
    bundle = Bundle(bundle_path)
    try:
        raw = bundle.read(MANIFEST, MAX_MANIFEST)
        value = parse(raw)
        inventory, created, expires, bootstrap_ready = validate_manifest(value, expected, now, purpose)
        names = set(inventory) | {MANIFEST, SIGNATURE}
        require(bundle.names() == names, "bundle file set differs from signed inventory")
        signature = bundle.read(SIGNATURE, MAX_SIGNATURE)
        public_key = bundle.read(PUBLIC_KEY, MAX_PUBLIC_KEY)
        epoch = verify_signature(raw, signature, public_key, expected, gpgv, now)
        require(created.timestamp() <= epoch < expires.timestamp(), "signature outside bootstrap release window")
        for name, pin in inventory.items():
            if name not in bundle.pins:
                bundle.read(name, min(pin["size"], MAX_ARTIFACT), retain=False)
            require(bundle.pins[name] == pin, "artifact hash or size mismatch: " + name)
        bundle.unchanged(names)
        return {
            "schema": "zerone-1-observer-release-verification/v1", "result": "PASS",
            "purpose": purpose, "release_id": value["release_id"], "chain_id": "zerone-1",
            "signer_fingerprint": expected, "manifest_sha256": digest(raw),
            "signature_sha256": digest(signature), "verified_at": now.isoformat().replace("+00:00", "Z"),
            "expiry_scope": "bootstrap-only", "expires_at": value["expires_at"],
            "bootstrap_ready": bootstrap_ready, "signature_verified": True,
            "file_integrity_verified": True, "input_sha256_and_size": bundle.pins,
            "executable_source_assertion": value["executable"]["source"],
            "source_reproduction_performed": False, "checkpoint_semantics_verified": False,
            "independent_chain_trust_established": False,
            "node_resume_authorized": False, "effects": "none",
            "limitations": [
                "Signature authenticates an operator statement and exact files, not an independently trusted chain.",
                "Checkpoint signatures, validator pins, header time and trust-period eligibility require the separate semantic verifier.",
                "Executable provenance is a signed operator assertion; this command does not reproduce a build.",
                "The included vulnerability review is signed evidence, not a claim by this verifier that the runtime is free of vulnerabilities.",
                "Resume verification does not authorize starting a node; an already initialized and successfully synced home must be checked separately.",
            ],
        }
    finally:
        bundle.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bundle", required=True, type=Path)
    parser.add_argument("--expected-fingerprint", required=True)
    parser.add_argument("--gpgv", required=True)
    parser.add_argument("--purpose", choices=("bootstrap", "resume"), default="bootstrap")
    args = parser.parse_args()
    try:
        require(args.expected_fingerprint == MAIN_FINGERPRINT, "expected fingerprint is not the established main release key")
        result = verify(args.bundle, args.expected_fingerprint, args.gpgv, args.purpose)
        print(json.dumps(result, sort_keys=True, separators=(",", ":")))
    except (Refusal, OSError, subprocess.SubprocessError, ValueError) as error:
        print(json.dumps({"result": "REFUSED", "reason": str(error), "effects": "none"}), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
