#!/usr/bin/env python3
"""Capture and run the release-bound legacy custom-staking census."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import pathlib
import re
import resource
import stat
import subprocess
import sys
import tempfile
from typing import Any


RUNNER_PATH = "deploy/run-custom-staking-census-evidence.py"
AUTHORITY_VERIFIER_PATH = "deploy/verify-authority-chain.py"
CONFIG_POLICY_PATH = "deploy/validate-fly-phase-config.py"
RELEASE_SCHEMA = "zerone-2-release-packet-v2"
CONTRACT_SCHEMA = "zerone-custom-staking-census-execution-contract-v1"
EVIDENCE_SCHEMA = "zerone-custom-staking-census-execution-evidence-v1"
SOURCE_MANIFEST_SCHEMA = (
    "zerone-custom-staking-census-application-db-file-manifest-v1"
)
DATABASE_SNAPSHOT_SCHEMA = "zerone-custom-staking-census-database-snapshot-v1"

CENSUS_BINARY_FILENAME = "custom-staking-census-linux-amd64"
CENSUS_REPORT_FILENAME = "CUSTOM-STAKING-CENSUS.json"
CENSUS_EVIDENCE_FILENAME = "CUSTOM-STAKING-CENSUS-EXECUTION-EVIDENCE.json"
CENSUS_EVIDENCE_SIGNATURE_FILENAME = f"{CENSUS_EVIDENCE_FILENAME}.sig"
SOURCE_MANIFEST_FILENAME = (
    "OFFLINE-HALTED-OBSERVER-APPLICATION-DB-FILE-MANIFEST.json"
)
SNAPSHOT_MANIFEST_FILENAME = "OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json"
OPERATOR_MANIFEST_FILENAME = "OPERATOR-TOOL-MANIFEST.json"
RELEASE_FILENAME = "RELEASE-PACKET.json"
RELEASE_SIGNATURE_FILENAME = "RELEASE-PACKET.json.sig"
CUTOVER_EVIDENCE_FILENAME = "CUTOVER-INITIATION-EVIDENCE.json"
CUTOVER_SIGNATURE_FILENAME = "CUTOVER-INITIATION-EVIDENCE.json.sig"
CUTOVER_DECISION_FILENAME = "CUTOVER-DECISION.json"
CUTOVER_DECISION_SIGNATURE_FILENAME = "CUTOVER-DECISION.json.sig"
OBSERVER_TERMINAL_FILENAME = "OBSERVER-EVIDENCE-MANIFEST.json"
REPORT_TRANSPORT = "stdout-captured-and-atomically-published"

AUTHORITY_LIMIT = (
    "factual attestation that the exact RELEASE-bound census binary scanned the "
    "declared stopped observer copy and produced the bound report; no migration, "
    "deployment, transaction, public-service, or DNS authority"
)
SCAN_GUARANTEES = {
    "required_stores": ["zerone_staking", "bank", "staking"],
    "complete_logical_store_iteration": True,
    "root_bound_leaf_count": True,
    "ics23_membership_proof_per_leaf": True,
    "root_commit_info_rechecked_after_scan": True,
    "database_backend_read_only": True,
    "write_attempts": 0,
}

LOWER_HASH = re.compile(r"^[0-9a-f]{64}$")
UPPER_HASH = re.compile(r"^[0-9A-F]{64}$")
COMMIT = re.compile(r"^[0-9a-f]{40}$")
FINGERPRINT = re.compile(r"^[0-9A-Fa-f]{40}(?:[0-9A-Fa-f]{24})?$")
POSITIVE_HEIGHT = re.compile(r"^[1-9][0-9]{0,17}$")
NODE_ID = re.compile(r"^[0-9A-Fa-f]{40}$")

MAX_JSON_BYTES = 64 * 1024 * 1024 + 1
MAX_SIGNATURE_BYTES = 16 * 1024 * 1024
MAX_BINARY_BYTES = 384 * 1024 * 1024
MAX_DATABASE_FILES = 100_000
MAX_RELATIVE_PATH_BYTES = 4096
MAX_EXECUTION_SECONDS = 6 * 60 * 60
MAX_CHILD_FILE_BYTES = 70 * 1024 * 1024


class EvidenceError(Exception):
    """A release-bound census precondition was not met."""


def fail(message: str) -> None:
    raise EvidenceError(message)


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def canonical_bytes(value: Any) -> bytes:
    return (
        json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
        + "\n"
    ).encode("utf-8")


def reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            fail(f"duplicate JSON object key {key!r}")
        result[key] = value
    return result


def parse_canonical_json(data: bytes, label: str) -> dict[str, Any]:
    try:
        value = json.loads(data, object_pairs_hook=reject_duplicate_keys)
    except (UnicodeDecodeError, json.JSONDecodeError, EvidenceError) as exc:
        fail(f"{label} is not unambiguous UTF-8 JSON: {exc}")
    if not isinstance(value, dict):
        fail(f"{label} root must be an object")
    if canonical_bytes(value) != data:
        fail(f"{label} is not exact canonical JSON")
    if b"REPLACE_" in data or b"replace-" in data:
        fail(f"{label} retains a placeholder")
    return value


def exact_object(value: Any, keys: set[str], label: str) -> dict[str, Any]:
    if not isinstance(value, dict) or set(value) != keys:
        fail(f"{label} does not have the exact required fields")
    return value


def lower_hash(value: Any, label: str) -> str:
    if not isinstance(value, str) or not LOWER_HASH.fullmatch(value):
        fail(f"{label} must be exactly 64 lowercase hexadecimal characters")
    return value


def upper_hash(value: Any, label: str) -> str:
    if not isinstance(value, str) or not UPPER_HASH.fullmatch(value):
        fail(f"{label} must be exactly 64 uppercase hexadecimal characters")
    return value


def canonical_epoch(value: Any, label: str) -> int:
    if not isinstance(value, str) or not re.fullmatch(
        r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z", value
    ):
        fail(f"{label} must be a canonical UTC second")
    try:
        parsed = dt.datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(
            tzinfo=dt.timezone.utc
        )
    except ValueError as exc:
        fail(f"{label} is not a real UTC time: {exc}")
    if parsed.strftime("%Y-%m-%dT%H:%M:%SZ") != value:
        fail(f"{label} is not a canonical UTC second")
    return int(parsed.timestamp())


def canonical_nanoseconds(value: Any, label: str) -> int:
    if not isinstance(value, str):
        fail(f"{label} must be an RFC3339 UTC timestamp")
    match = re.fullmatch(
        r"([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2})"
        r"(?:\.([0-9]{1,9}))?Z",
        value,
    )
    if match is None:
        fail(f"{label} is not canonical RFC3339 UTC with at most nanoseconds")
    fraction = match.group(2) or ""
    if fraction.endswith("0"):
        fail(f"{label} has non-canonical trailing fractional zeros")
    try:
        parsed = dt.datetime.strptime(match.group(1), "%Y-%m-%dT%H:%M:%S").replace(
            tzinfo=dt.timezone.utc
        )
    except ValueError as exc:
        fail(f"{label} is not a real UTC time: {exc}")
    return int(parsed.timestamp()) * 1_000_000_000 + int(fraction.ljust(9, "0") or "0")


def utc_second(epoch: int | None = None) -> str:
    when = dt.datetime.fromtimestamp(
        int(dt.datetime.now(tz=dt.timezone.utc).timestamp()) if epoch is None else epoch,
        tz=dt.timezone.utc,
    )
    return when.strftime("%Y-%m-%dT%H:%M:%SZ")


def canonical_path(raw: str, label: str) -> pathlib.Path:
    path = pathlib.Path(raw)
    if not path.is_absolute() or str(path) != os.path.normpath(raw):
        fail(f"{label} must be a normalized absolute path")
    try:
        resolved = path.resolve(strict=True)
    except OSError as exc:
        fail(f"could not resolve {label}: {exc}")
    if resolved != path:
        fail(f"{label} must not contain a symlink or path alias")
    return path


def existing_directory(raw: str, label: str) -> pathlib.Path:
    path = canonical_path(raw, label)
    try:
        info = os.lstat(path)
    except OSError as exc:
        fail(f"could not inspect {label}: {exc}")
    if not stat.S_ISDIR(info.st_mode) or stat.S_ISLNK(info.st_mode):
        fail(f"{label} must be a non-symlink directory")
    return path


def existing_file(raw: str, label: str) -> pathlib.Path:
    path = canonical_path(raw, label)
    try:
        info = os.lstat(path)
    except OSError as exc:
        fail(f"could not inspect {label}: {exc}")
    if not stat.S_ISREG(info.st_mode) or stat.S_ISLNK(info.st_mode):
        fail(f"{label} must be a non-symlink regular file")
    return path


def output_path(raw: str, expected_name: str, label: str) -> pathlib.Path:
    path = pathlib.Path(raw)
    if not path.is_absolute() or str(path) != os.path.normpath(raw):
        fail(f"{label} must be a normalized absolute path")
    if path.name != expected_name:
        fail(f"{label} filename must be {expected_name}")
    parent = existing_directory(str(path.parent), f"{label} parent")
    path = parent / path.name
    if os.path.lexists(path):
        fail(f"{label} already exists; refusing overwrite")
    return path


def is_inside(parent: pathlib.Path, candidate: pathlib.Path) -> bool:
    return candidate == parent or parent in candidate.parents


def read_regular(path: pathlib.Path, label: str, limit: int) -> bytes:
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
    try:
        fd = os.open(path, flags)
    except OSError as exc:
        fail(f"could not open {label}: {exc}")
    try:
        before = os.fstat(fd)
        if not stat.S_ISREG(before.st_mode):
            fail(f"{label} must be a regular file")
        if before.st_size > limit:
            fail(f"{label} exceeds the {limit}-byte limit")
        chunks: list[bytes] = []
        remaining = limit + 1
        while remaining:
            chunk = os.read(fd, min(remaining, 1024 * 1024))
            if not chunk:
                break
            chunks.append(chunk)
            remaining -= len(chunk)
        data = b"".join(chunks)
        after = os.fstat(fd)
        stable_fields = ("st_dev", "st_ino", "st_size", "st_mtime_ns", "st_ctime_ns")
        if any(getattr(before, field) != getattr(after, field) for field in stable_fields):
            fail(f"{label} changed while it was read")
        if len(data) > limit or len(data) != before.st_size:
            fail(f"{label} exceeds its limit or changed while it was read")
        return data
    finally:
        os.close(fd)


def hash_regular(path: pathlib.Path, label: str, limit: int | None = None) -> tuple[str, int]:
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
    try:
        fd = os.open(path, flags)
    except OSError as exc:
        fail(f"could not open {label}: {exc}")
    try:
        before = os.fstat(fd)
        if not stat.S_ISREG(before.st_mode):
            fail(f"{label} must be a regular file")
        if before.st_nlink != 1:
            fail(f"{label} must have exactly one hard link")
        if limit is not None and before.st_size > limit:
            fail(f"{label} exceeds the {limit}-byte limit")
        digest = hashlib.sha256()
        count = 0
        while True:
            chunk = os.read(fd, 1024 * 1024)
            if not chunk:
                break
            digest.update(chunk)
            count += len(chunk)
            if limit is not None and count > limit:
                fail(f"{label} exceeds the {limit}-byte limit")
        after = os.fstat(fd)
        stable_fields = ("st_dev", "st_ino", "st_size", "st_mtime_ns", "st_ctime_ns")
        if any(getattr(before, field) != getattr(after, field) for field in stable_fields):
            fail(f"{label} changed while it was hashed")
        if count != before.st_size:
            fail(f"{label} changed while it was hashed")
        return digest.hexdigest(), count
    finally:
        os.close(fd)


def atomic_publish(path: pathlib.Path, contents: bytes) -> None:
    parent_fd = os.open(
        path.parent,
        os.O_RDONLY | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_NOFOLLOW", 0),
    )
    temporary_fd = -1
    temporary_name = ""
    try:
        temporary_fd, temporary_name = tempfile.mkstemp(
            prefix=".custom-staking-census-evidence.", dir=path.parent
        )
        os.fchmod(temporary_fd, 0o600)
        written = 0
        while written < len(contents):
            count = os.write(temporary_fd, contents[written:])
            if count <= 0:
                fail("could not completely write temporary evidence")
            written += count
        os.fsync(temporary_fd)
        os.close(temporary_fd)
        temporary_fd = -1
        os.link(
            temporary_name,
            path.name,
            dst_dir_fd=parent_fd,
            follow_symlinks=False,
        )
        os.unlink(temporary_name)
        temporary_name = ""
        os.fsync(parent_fd)
    except FileExistsError:
        fail(f"{path.name} appeared during publication; refusing overwrite")
    except OSError as exc:
        fail(f"could not publish {path.name} atomically: {exc}")
    finally:
        if temporary_fd >= 0:
            os.close(temporary_fd)
        if temporary_name:
            try:
                os.unlink(temporary_name)
            except OSError:
                pass
        os.close(parent_fd)


def snapshot_executable(
    source: pathlib.Path, destination: pathlib.Path, expected_hash: str
) -> None:
    source_flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
    destination_flags = (
        os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0)
    )
    try:
        source_fd = os.open(source, source_flags)
    except OSError as exc:
        fail(f"could not open census binary for snapshot: {exc}")
    destination_fd = -1
    try:
        before = os.fstat(source_fd)
        if not stat.S_ISREG(before.st_mode) or before.st_size > MAX_BINARY_BYTES:
            fail("census binary snapshot source is not a bounded regular file")
        destination_fd = os.open(destination, destination_flags, 0o700)
        digest = hashlib.sha256()
        count = 0
        while True:
            chunk = os.read(source_fd, 1024 * 1024)
            if not chunk:
                break
            digest.update(chunk)
            count += len(chunk)
            offset = 0
            while offset < len(chunk):
                written = os.write(destination_fd, chunk[offset:])
                if written <= 0:
                    fail("could not completely snapshot census binary")
                offset += written
        after = os.fstat(source_fd)
        stable_fields = ("st_dev", "st_ino", "st_size", "st_mtime_ns", "st_ctime_ns")
        if any(getattr(before, field) != getattr(after, field) for field in stable_fields):
            fail("census binary changed while it was snapshotted")
        if count != before.st_size or digest.hexdigest() != expected_hash:
            fail("snapshotted census binary differs from RELEASE")
        os.fchmod(destination_fd, 0o700)
        os.fsync(destination_fd)
    except OSError as exc:
        fail(f"could not snapshot census binary: {exc}")
    finally:
        if destination_fd >= 0:
            os.close(destination_fd)
        os.close(source_fd)


def encoded_name(name: str) -> bytes:
    try:
        raw = name.encode("utf-8")
    except UnicodeEncodeError as exc:
        fail(f"database path is not canonical UTF-8: {exc}")
    if not raw or len(raw) > MAX_RELATIVE_PATH_BYTES:
        fail("database path component is empty or oversized")
    return raw


def scan_application_db(home: pathlib.Path) -> dict[str, Any]:
    database = home / "data" / "application.db"
    try:
        resolved = database.resolve(strict=True)
    except OSError as exc:
        fail(f"could not resolve copied data/application.db: {exc}")
    if resolved != database:
        fail("copied data/application.db must not contain a symlink or path alias")
    root_info = os.lstat(database)
    if not stat.S_ISDIR(root_info.st_mode) or stat.S_ISLNK(root_info.st_mode):
        fail("copied data/application.db must be a non-symlink directory")

    directories: list[str] = []
    files: list[dict[str, Any]] = []
    seen_inodes: set[tuple[int, int]] = set()

    def visit(directory: pathlib.Path, relative: pathlib.PurePosixPath) -> None:
        try:
            with os.scandir(directory) as iterator:
                entries = sorted(iterator, key=lambda entry: encoded_name(entry.name))
        except OSError as exc:
            fail(f"could not scan copied application database: {exc}")
        for entry in entries:
            encoded_name(entry.name)
            child_relative = relative / entry.name
            relative_text = child_relative.as_posix()
            if len(relative_text.encode("utf-8")) > MAX_RELATIVE_PATH_BYTES:
                fail("database relative path exceeds its byte limit")
            try:
                info = entry.stat(follow_symlinks=False)
            except OSError as exc:
                fail(f"could not inspect database entry {relative_text!r}: {exc}")
            if stat.S_ISLNK(info.st_mode):
                fail(f"database entry {relative_text!r} must not be a symlink")
            if stat.S_ISDIR(info.st_mode):
                directories.append(relative_text)
                visit(pathlib.Path(entry.path), child_relative)
                continue
            if not stat.S_ISREG(info.st_mode):
                fail(f"database entry {relative_text!r} is not a regular file")
            identity = (info.st_dev, info.st_ino)
            if identity in seen_inodes or info.st_nlink != 1:
                fail(f"database entry {relative_text!r} is hard-linked or duplicated")
            seen_inodes.add(identity)
            digest, size = hash_regular(
                pathlib.Path(entry.path), f"database entry {relative_text!r}"
            )
            files.append({"path": relative_text, "size": size, "sha256": digest})
            if len(files) > MAX_DATABASE_FILES:
                fail(f"application database exceeds {MAX_DATABASE_FILES} files")

    visit(database, pathlib.PurePosixPath())
    if not files:
        fail("copied data/application.db contains no regular files")
    final_root = os.lstat(database)
    stable_root_fields = ("st_dev", "st_ino", "st_mtime_ns", "st_ctime_ns")
    if any(
        getattr(root_info, field) != getattr(final_root, field)
        for field in stable_root_fields
    ):
        fail("copied data/application.db changed while it was scanned")

    snapshot = {
        "schema": DATABASE_SNAPSHOT_SCHEMA,
        "directories": directories,
        "files": files,
    }
    return {
        "schema": SOURCE_MANIFEST_SCHEMA,
        "database_root": "data/application.db",
        "database_snapshot_sha256": sha256(canonical_bytes(snapshot)),
        "directories": directories,
        "files": files,
    }


def validate_source_manifest(
    home: pathlib.Path, manifest_path: pathlib.Path, label: str
) -> tuple[dict[str, Any], bytes]:
    raw = read_regular(manifest_path, "application DB file manifest", MAX_JSON_BYTES)
    declared = parse_canonical_json(raw, "application DB file manifest")
    actual = scan_application_db(home)
    if declared != actual:
        fail(f"source application.db differs from its file manifest {label}")
    lower_hash(
        declared.get("database_snapshot_sha256"),
        "application DB snapshot hash",
    )
    return declared, raw


def verify_signature(
    payload: pathlib.Path,
    signature: pathlib.Path,
    authority: dict[str, Any],
    expected_fingerprint: str,
    expected_signature_name: str,
) -> int:
    if not (
        authority.get("algorithm") == "openpgp"
        and authority.get("detached_signature_filename") == expected_signature_name
    ):
        fail(f"{payload.name} signature authority is malformed")
    signer = authority.get("authorized_signer_fingerprint")
    if not isinstance(signer, str) or not FINGERPRINT.fullmatch(signer):
        fail(f"{payload.name} signer fingerprint is malformed")
    if signer.upper() != expected_fingerprint.upper():
        fail(f"{payload.name} repeats the wrong signer fingerprint")
    try:
        result = subprocess.run(
            [
                "gpg",
                "--batch",
                "--status-fd=1",
                "--verify",
                str(signature),
                str(payload),
            ],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            timeout=60,
        )
    except (OSError, subprocess.CalledProcessError, subprocess.TimeoutExpired) as exc:
        fail(f"{payload.name} detached signature verification failed: {exc}")
    valid = [
        line.split()
        for line in result.stdout.splitlines()
        if line.startswith("[GNUPG:] VALIDSIG ") and len(line.split()) >= 5
    ]
    if len(valid) != 1 or valid[0][2].upper() != expected_fingerprint.upper():
        fail(f"{payload.name} must have exactly one valid signature from the expected key")
    if not valid[0][4].isdecimal():
        fail(f"{payload.name} signature has no numeric creation time")
    epoch = int(valid[0][4])
    now = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
    if epoch <= 0 or epoch > now + 300:
        fail(f"{payload.name} signature time is zero or in the future")
    return epoch


def validate_release(
    release: dict[str, Any],
    release_raw: bytes,
    release_path: pathlib.Path,
    release_signature_path: pathlib.Path,
    operator_manifest_path: pathlib.Path,
    expected_main: str,
    expected_transition: str,
) -> tuple[
    dict[str, Any],
    dict[str, str],
    int,
    int,
    str,
    dict[str, Any],
]:
    if release.get("schema") != RELEASE_SCHEMA or release.get("chain_id") != "zerone-2":
        fail("RELEASE has the wrong schema or chain ID")
    authority = exact_object(
        release.get("signature_authority"),
        {"algorithm", "authorized_signer_fingerprint", "detached_signature_filename"},
        "RELEASE signature authority",
    )
    release_signature_epoch = verify_signature(
        release_path,
        release_signature_path,
        authority,
        expected_main,
        RELEASE_SIGNATURE_FILENAME,
    )
    release_created_epoch = canonical_epoch(
        release.get("created_at"), "RELEASE creation time"
    )
    if release_created_epoch > release_signature_epoch:
        fail("RELEASE was signed before its declared creation time")

    source = exact_object(
        release.get("source"),
        {"commit", "signed_annotated_tag", "tag_signer_fingerprint"},
        "RELEASE source",
    )
    if not (
        isinstance(source["commit"], str)
        and COMMIT.fullmatch(source["commit"])
        and isinstance(source["signed_annotated_tag"], str)
        and source["signed_annotated_tag"].strip() == source["signed_annotated_tag"]
        and source["signed_annotated_tag"]
        and isinstance(source["tag_signer_fingerprint"], str)
        and FINGERPRINT.fullmatch(source["tag_signer_fingerprint"])
        and source["tag_signer_fingerprint"].upper() == expected_main.upper()
    ):
        fail("RELEASE source identity is malformed or signed by the wrong key")

    identities = release.get("public_identities")
    transition = identities.get("transition_attestation") if isinstance(identities, dict) else None
    transition = exact_object(
        transition,
        {"algorithm", "authorized_signer_fingerprint", "purpose"},
        "RELEASE transition authority",
    )
    transition_fingerprint = transition.get("authorized_signer_fingerprint")
    if not (
        transition.get("algorithm") == "openpgp"
        and isinstance(transition_fingerprint, str)
        and FINGERPRINT.fullmatch(transition_fingerprint)
        and transition_fingerprint.upper() != expected_main.upper()
        and transition_fingerprint.upper() == expected_transition.upper()
    ):
        fail("RELEASE transition authority is malformed or not independent")

    contract = exact_object(
        release.get("custom_staking_census_execution"),
        {"schema", "binary", "execution_evidence"},
        "RELEASE census execution contract",
    )
    binary = exact_object(
        contract["binary"], {"filename", "sha256"}, "RELEASE census binary"
    )
    evidence = exact_object(
        contract["execution_evidence"],
        {"filename", "detached_signature_filename", "authorized_signer_fingerprint"},
        "RELEASE census execution evidence",
    )
    if not (
        contract["schema"] == CONTRACT_SCHEMA
        and binary["filename"] == CENSUS_BINARY_FILENAME
        and LOWER_HASH.fullmatch(binary.get("sha256", ""))
        and evidence["filename"] == CENSUS_EVIDENCE_FILENAME
        and evidence["detached_signature_filename"]
        == CENSUS_EVIDENCE_SIGNATURE_FILENAME
        and evidence["authorized_signer_fingerprint"] == transition_fingerprint
    ):
        fail("RELEASE census execution contract is unsafe")

    operator_raw = read_regular(
        operator_manifest_path, "operator-tool manifest", MAX_JSON_BYTES
    )
    operator_hash = lower_hash(
        release.get("operator_tool_manifest_sha256"),
        "RELEASE operator-tool manifest hash",
    )
    if sha256(operator_raw) != operator_hash:
        fail("operator-tool manifest bytes differ from RELEASE")
    operator = parse_canonical_json(operator_raw, "operator-tool manifest")
    operator = exact_object(
        operator,
        {
            "schema",
            "source_commit",
            "signed_tag",
            "files",
            "authority_bundle",
            "component_signature_policy",
        },
        "operator-tool manifest",
    )
    runner_hash, _ = hash_regular(
        pathlib.Path(__file__).resolve(), "census evidence runner", MAX_JSON_BYTES
    )
    if not (
        operator["schema"] == "zerone-operator-tool-manifest-v2"
        and operator["source_commit"] == source["commit"]
        and operator["signed_tag"] == source["signed_annotated_tag"]
        and isinstance(operator["files"], dict)
        and isinstance(operator["authority_bundle"], dict)
        and isinstance(operator["component_signature_policy"], dict)
        and operator["files"].get(RUNNER_PATH) == runner_hash
    ):
        fail("this census evidence runner is not bound by the RELEASE operator manifest")

    pair = {
        "sha256": sha256(release_raw),
        "detached_signature_sha256": hash_regular(
            release_signature_path, "RELEASE signature", MAX_SIGNATURE_BYTES
        )[0],
    }
    return (
        contract,
        pair,
        release_created_epoch,
        release_signature_epoch,
        runner_hash,
        operator,
    )


def verified_helper_bytes(
    tool_root: pathlib.Path,
    operator_manifest: dict[str, Any],
    relative: str,
    label: str,
) -> bytes:
    files = operator_manifest.get("files")
    expected = files.get(relative) if isinstance(files, dict) else None
    lower_hash(expected, f"operator-manifest {label} hash")
    path = existing_file(str(tool_root / relative), label)
    data = read_regular(path, label, MAX_JSON_BYTES)
    if sha256(data) != expected:
        fail(f"{label} bytes differ from the RELEASE operator manifest")
    return data


def publish_private_snapshot(directory: pathlib.Path, name: str, data: bytes) -> pathlib.Path:
    path = directory / name
    flags = os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0)
    try:
        fd = os.open(path, flags, 0o700)
    except OSError as exc:
        fail(f"could not create private {name} snapshot: {exc}")
    try:
        os.fchmod(fd, 0o700)
        offset = 0
        while offset < len(data):
            count = os.write(fd, data[offset:])
            if count <= 0:
                fail(f"could not completely write private {name} snapshot")
            offset += count
        os.fsync(fd)
    finally:
        os.close(fd)
    return path


def run_full_authority_gate(
    authority_bundle: pathlib.Path,
    tool_root: pathlib.Path,
    operator_manifest: dict[str, Any],
    release_path: pathlib.Path,
    release_signature_path: pathlib.Path,
    cutover_decision_path: pathlib.Path,
    cutover_decision_signature_path: pathlib.Path,
    cutover_path: pathlib.Path,
    cutover_signature_path: pathlib.Path,
    config_policy_path: pathlib.Path,
    expected_main: str,
    temporary_parent: pathlib.Path,
) -> None:
    expected_runner = tool_root / RUNNER_PATH
    if pathlib.Path(__file__).resolve() != expected_runner:
        fail("the census evidence runner must execute from the declared tool root")
    verifier_bytes = verified_helper_bytes(
        tool_root,
        operator_manifest,
        AUTHORITY_VERIFIER_PATH,
        "authority-chain verifier",
    )
    policy_bytes = verified_helper_bytes(
        tool_root,
        operator_manifest,
        CONFIG_POLICY_PATH,
        "configuration policy",
    )
    if config_policy_path != tool_root / CONFIG_POLICY_PATH:
        fail("configuration policy path is not the fixed tool-root path")
    if read_regular(config_policy_path, "configuration policy", MAX_JSON_BYTES) != policy_bytes:
        fail("configuration policy changed before authority verification")

    with tempfile.TemporaryDirectory(
        prefix=".custom-staking-census-authority.", dir=temporary_parent
    ) as raw_temporary:
        temporary = pathlib.Path(raw_temporary)
        os.chmod(temporary, 0o700)
        verifier = publish_private_snapshot(
            temporary, "verify-authority-chain.py", verifier_bytes
        )
        policy = publish_private_snapshot(
            temporary, "validate-fly-phase-config.py", policy_bytes
        )
        command = [
            sys.executable,
            str(verifier),
            "cutover-postinit",
            str(authority_bundle),
            expected_main,
            "--release",
            str(release_path),
            "--release-sig",
            str(release_signature_path),
            "--decision",
            str(cutover_decision_path),
            "--decision-sig",
            str(cutover_decision_signature_path),
            "--initiation",
            str(cutover_path),
            "--initiation-sig",
            str(cutover_signature_path),
            "--config-policy",
            str(policy),
            "--tool-root",
            str(tool_root),
        ]
        try:
            subprocess.run(
                command,
                check=True,
                stdin=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.PIPE,
                timeout=300,
                close_fds=True,
            )
        except (OSError, subprocess.CalledProcessError, subprocess.TimeoutExpired) as exc:
            fail(f"full CUTOVER post-initiation authority chain did not verify: {exc}")


def validate_cutover_evidence(
    path: pathlib.Path,
    signature_path: pathlib.Path,
    raw: bytes,
    evidence: dict[str, Any],
    expected_main: str,
) -> tuple[dict[str, str], int, int]:
    authority = evidence.get("signature_authority")
    if not isinstance(authority, dict):
        fail("CUTOVER-initiation signature authority is missing")
    signature_epoch = verify_signature(
        path,
        signature_path,
        authority,
        expected_main,
        CUTOVER_SIGNATURE_FILENAME,
    )
    created_epoch = canonical_epoch(
        evidence.get("created_at"), "CUTOVER-initiation creation time"
    )
    if not (
        evidence.get("schema") == "zerone-2-cutover-initiation-evidence-v1"
        and evidence.get("attestation_result") == "MATCH"
        and evidence.get("deadline_satisfied") is True
        and created_epoch <= signature_epoch
    ):
        fail("CUTOVER-initiation evidence is not a completed, timely MATCH")
    return (
        {
            "sha256": sha256(raw),
            "detached_signature_sha256": hash_regular(
                signature_path, "CUTOVER-initiation signature", MAX_SIGNATURE_BYTES
            )[0],
        },
        created_epoch,
        signature_epoch,
    )


def validate_snapshot_manifest(
    raw: bytes, manifest: dict[str, Any], source_manifest: dict[str, Any], source_raw: bytes
) -> dict[str, str]:
    expected_keys = {
        "schema",
        "chain_id",
        "checkpoint_state_height",
        "checkpoint_app_hash",
        "final_committed_height",
        "halt_trigger_height",
        "blockstore_height",
        "abci_last_applied_height",
        "post_anchor_app_hash",
        "source_observer_node_id",
        "database_snapshot_sha256",
        "file_manifest_sha256",
        "stored_offline",
        "included_in_authority_bundle",
        "contains_signer_keys",
        "result",
    }
    exact_object(manifest, expected_keys, "offline halted observer snapshot manifest")
    for field in (
        "checkpoint_state_height",
        "final_committed_height",
        "halt_trigger_height",
        "blockstore_height",
        "abci_last_applied_height",
    ):
        if not isinstance(manifest[field], str) or not POSITIVE_HEIGHT.fullmatch(
            manifest[field]
        ):
            fail(f"snapshot {field} is not a canonical positive height")
    checkpoint = int(manifest["checkpoint_state_height"])
    applied = int(manifest["abci_last_applied_height"])
    halt = int(manifest["halt_trigger_height"])
    if not (
        manifest["schema"]
        == "zerone-1-offline-halted-observer-snapshot-manifest-v1"
        and manifest["chain_id"] == "zerone-1"
        and applied == checkpoint + 1
        and int(manifest["final_committed_height"]) == applied
        and halt == applied + 1
        and int(manifest["blockstore_height"]) == halt
        and NODE_ID.fullmatch(manifest.get("source_observer_node_id", ""))
        and manifest["stored_offline"] is True
        and manifest["included_in_authority_bundle"] is False
        and manifest["contains_signer_keys"] is False
        and manifest["result"] == "MATCH"
    ):
        fail("offline halted observer snapshot manifest is unsafe or inconsistent")
    upper_hash(manifest["checkpoint_app_hash"], "snapshot checkpoint AppHash")
    app_hash = upper_hash(manifest["post_anchor_app_hash"], "snapshot post-anchor AppHash")
    database_hash = lower_hash(
        manifest["database_snapshot_sha256"], "snapshot database hash"
    )
    file_manifest_hash = lower_hash(
        manifest["file_manifest_sha256"], "snapshot file-manifest hash"
    )
    if not (
        database_hash == source_manifest["database_snapshot_sha256"]
        and file_manifest_hash == sha256(source_raw)
    ):
        fail("offline snapshot hashes do not bind the actual application DB manifest")
    return {
        "height": str(applied),
        "app_hash": app_hash.lower(),
        "manifest_sha256": sha256(raw),
        "database_snapshot_sha256": database_hash,
        "file_manifest_sha256": file_manifest_hash,
    }


def validate_terminal_observer(
    terminal: dict[str, Any], snapshot: dict[str, Any]
) -> int:
    required = {
        "schema",
        "role",
        "chain_id",
        "genesis_sha256",
        "checkpoint_state_height",
        "checkpoint_app_hash",
        "final_committed_height",
        "final_committed_block_time",
        "halt_trigger_height",
        "anchor_block_hash",
        "halt_trigger_block_hash",
        "halt_trigger_block_time",
        "post_anchor_app_hash",
        "abci_last_applied_height",
        "node_id",
        "validator_pubkey",
        "payload_sha256",
        "result",
    }
    exact_object(terminal, required, "terminal observer manifest")
    if not (
        terminal["schema"] == "zerone-1-terminal-evidence-manifest-v2"
        and terminal["role"] == "independent-observer"
        and terminal["chain_id"] == "zerone-1"
        and terminal["checkpoint_state_height"] == snapshot["checkpoint_state_height"]
        and terminal["checkpoint_app_hash"] == snapshot["checkpoint_app_hash"]
        and terminal["final_committed_height"] == snapshot["final_committed_height"]
        and terminal["halt_trigger_height"] == snapshot["halt_trigger_height"]
        and terminal["post_anchor_app_hash"] == snapshot["post_anchor_app_hash"]
        and terminal["abci_last_applied_height"] == snapshot["abci_last_applied_height"]
        and terminal["node_id"] == snapshot["source_observer_node_id"]
        and terminal["result"] == "MATCH"
    ):
        fail("terminal observer manifest differs from the stopped snapshot")
    return canonical_nanoseconds(
        terminal["halt_trigger_block_time"], "terminal observer halt-trigger time"
    )


def validate_report(raw: bytes, state: dict[str, str]) -> tuple[str, str]:
    if not raw or len(raw) > MAX_JSON_BYTES:
        fail("census report is empty or exceeds its byte limit")
    try:
        report = json.loads(raw, object_pairs_hook=reject_duplicate_keys)
    except (UnicodeDecodeError, json.JSONDecodeError, EvidenceError) as exc:
        fail(f"census report is not unambiguous UTF-8 JSON: {exc}")
    report = exact_object(
        report,
        {"schema", "result", "evidence", "multistore", "stores", "census", "report_sha256"},
        "census report",
    )
    report_hash = lower_hash(report["report_sha256"], "census report self-hash")
    suffix = f',"report_sha256":"{report_hash}"}}'.encode()
    if not raw.endswith(b"\n") or not raw[:-1].endswith(suffix):
        fail("census report is not the exact sealed encoding")
    sealed = raw[:-1]
    unsealed = sealed[: -len(suffix)] + b',"report_sha256":""}'
    if sha256(unsealed) != report_hash:
        fail("census report self-hash does not match its unsealed bytes")
    expected_state = {
        "chain_id": "zerone-1",
        "height": state["height"],
        "app_hash": state["app_hash"],
        "source_commit": state["source_commit"],
    }
    if not (
        report["schema"] == "zerone/custom-staking-census/v1"
        and report["result"] == "PASS"
        and report["evidence"] == expected_state
    ):
        fail("census report did not PASS for the exact RELEASE/A/E state")
    return sha256(raw), report_hash


def child_file_limit() -> None:
    resource.setrlimit(resource.RLIMIT_FSIZE, (MAX_CHILD_FILE_BYTES, MAX_CHILD_FILE_BYTES))


def capture_manifest(args: argparse.Namespace) -> None:
    home = existing_directory(args.home, "copied node home")
    destination = output_path(args.output, SOURCE_MANIFEST_FILENAME, "file-manifest output")
    first = scan_application_db(home)
    second = scan_application_db(home)
    if first != second:
        fail("copied application DB changed while its file manifest was captured")
    contents = canonical_bytes(first)
    if len(contents) > MAX_JSON_BYTES:
        fail("application DB file manifest exceeds its byte limit")
    atomic_publish(destination, contents)
    print(f"custom-staking-census-source-manifest: MATCH {sha256(contents)}")


def execute(args: argparse.Namespace) -> None:
    expected_main = args.expected_release_signer_fingerprint
    expected_transition = args.expected_transition_signer_fingerprint
    if not FINGERPRINT.fullmatch(expected_main):
        fail("expected release signer must be a full 40- or 64-hex fingerprint")
    if not FINGERPRINT.fullmatch(expected_transition):
        fail("expected transition signer must be a full 40- or 64-hex fingerprint")
    if expected_transition.upper() == expected_main.upper():
        fail("release and transition signer fingerprints must differ")

    release_path = existing_file(args.release_packet, "RELEASE packet")
    release_signature_path = existing_file(args.release_signature, "RELEASE signature")
    operator_manifest_path = existing_file(
        args.operator_tool_manifest, "operator-tool manifest"
    )
    cutover_path = existing_file(
        args.cutover_initiation_evidence, "CUTOVER-initiation evidence"
    )
    cutover_signature_path = existing_file(
        args.cutover_initiation_signature, "CUTOVER-initiation signature"
    )
    cutover_decision_path = existing_file(args.cutover_decision, "CUTOVER decision")
    cutover_decision_signature_path = existing_file(
        args.cutover_decision_signature, "CUTOVER decision signature"
    )
    authority_bundle = existing_directory(args.authority_bundle, "authority bundle")
    tool_root = existing_directory(args.tool_root, "operator tool root")
    config_policy_path = existing_file(args.config_policy, "configuration policy")
    snapshot_path = existing_file(args.snapshot_manifest, "offline snapshot manifest")
    terminal_observer_path = existing_file(
        args.terminal_observer_manifest, "terminal observer manifest"
    )
    source_manifest_path = existing_file(
        args.source_file_manifest, "application DB file manifest"
    )
    binary_path = existing_file(args.binary, "custom-staking census binary")
    home = existing_directory(args.home, "copied node home")
    report_path = output_path(args.report_output, CENSUS_REPORT_FILENAME, "report output")
    evidence_path = output_path(
        args.evidence_output, CENSUS_EVIDENCE_FILENAME, "execution-evidence output"
    )
    if binary_path.name != CENSUS_BINARY_FILENAME:
        fail(f"custom-staking census binary filename must be {CENSUS_BINARY_FILENAME}")
    if source_manifest_path.name != SOURCE_MANIFEST_FILENAME:
        fail(f"source file-manifest filename must be {SOURCE_MANIFEST_FILENAME}")
    for path, expected, label in (
        (release_path, RELEASE_FILENAME, "RELEASE packet"),
        (release_signature_path, RELEASE_SIGNATURE_FILENAME, "RELEASE signature"),
        (operator_manifest_path, OPERATOR_MANIFEST_FILENAME, "operator-tool manifest"),
        (cutover_path, CUTOVER_EVIDENCE_FILENAME, "CUTOVER-initiation evidence"),
        (cutover_signature_path, CUTOVER_SIGNATURE_FILENAME, "CUTOVER-initiation signature"),
        (cutover_decision_path, CUTOVER_DECISION_FILENAME, "CUTOVER decision"),
        (
            cutover_decision_signature_path,
            CUTOVER_DECISION_SIGNATURE_FILENAME,
            "CUTOVER decision signature",
        ),
        (snapshot_path, SNAPSHOT_MANIFEST_FILENAME, "offline snapshot manifest"),
        (terminal_observer_path, OBSERVER_TERMINAL_FILENAME, "terminal observer manifest"),
    ):
        if path.name != expected:
            fail(f"{label} filename must be {expected}")
    if report_path == evidence_path:
        fail("report and execution-evidence outputs must be distinct")
    if any(is_inside(home, path) for path in (binary_path, report_path, evidence_path)):
        fail("binary and outputs must be outside the copied node home")
    binary_mode = os.lstat(binary_path).st_mode
    if not binary_mode & 0o111:
        fail("custom-staking census binary is not executable")

    release_raw = read_regular(release_path, "RELEASE packet", MAX_JSON_BYTES)
    release = parse_canonical_json(release_raw, "RELEASE packet")
    (
        contract,
        release_pair,
        release_created,
        release_signed,
        runner_hash,
        operator_manifest,
    ) = validate_release(
        release,
        release_raw,
        release_path,
        release_signature_path,
        operator_manifest_path,
        expected_main,
        expected_transition,
    )
    binary_hash, _ = hash_regular(binary_path, "custom-staking census binary", MAX_BINARY_BYTES)
    if binary_hash != contract["binary"]["sha256"]:
        fail("custom-staking census binary bytes differ from RELEASE")

    cutover_raw = read_regular(
        cutover_path, "CUTOVER-initiation evidence", MAX_JSON_BYTES
    )
    cutover = parse_canonical_json(cutover_raw, "CUTOVER-initiation evidence")
    cutover_pair, cutover_created, cutover_signed = validate_cutover_evidence(
        cutover_path,
        cutover_signature_path,
        cutover_raw,
        cutover,
        expected_main,
    )
    if not release_created <= release_signed <= cutover_created <= cutover_signed:
        fail("RELEASE/CUTOVER-initiation chronology is non-monotonic")

    bundled_operator_path = existing_file(
        str(authority_bundle / OPERATOR_MANIFEST_FILENAME),
        "bundled operator-tool manifest",
    )
    if read_regular(
        bundled_operator_path, "bundled operator-tool manifest", MAX_JSON_BYTES
    ) != read_regular(operator_manifest_path, "operator-tool manifest", MAX_JSON_BYTES):
        fail("explicit operator-tool manifest differs from the authority bundle")
    stable_authority_inputs = {
        path: hash_regular(path, label, MAX_JSON_BYTES if path.suffix == ".json" else MAX_SIGNATURE_BYTES)[0]
        for path, label in (
            (release_path, "RELEASE packet"),
            (release_signature_path, "RELEASE signature"),
            (operator_manifest_path, "operator-tool manifest"),
            (cutover_decision_path, "CUTOVER decision"),
            (cutover_decision_signature_path, "CUTOVER decision signature"),
            (cutover_path, "CUTOVER-initiation evidence"),
            (cutover_signature_path, "CUTOVER-initiation signature"),
        )
    }
    run_full_authority_gate(
        authority_bundle,
        tool_root,
        operator_manifest,
        release_path,
        release_signature_path,
        cutover_decision_path,
        cutover_decision_signature_path,
        cutover_path,
        cutover_signature_path,
        config_policy_path,
        expected_main,
        evidence_path.parent,
    )
    for path, expected_hash in stable_authority_inputs.items():
        limit = MAX_JSON_BYTES if path.suffix == ".json" else MAX_SIGNATURE_BYTES
        if hash_regular(path, f"post-verification {path.name}", limit)[0] != expected_hash:
            fail(f"{path.name} changed during authority verification")

    binary_snapshot_owner = tempfile.TemporaryDirectory(
        prefix=".custom-staking-census-binary.", dir=evidence_path.parent
    )
    binary_snapshot_directory = pathlib.Path(binary_snapshot_owner.name)
    os.chmod(binary_snapshot_directory, 0o700)
    binary_snapshot = binary_snapshot_directory / CENSUS_BINARY_FILENAME
    snapshot_executable(binary_path, binary_snapshot, binary_hash)
    binary_path = binary_snapshot

    source_manifest, source_manifest_raw = validate_source_manifest(
        home, source_manifest_path, "before execution"
    )
    snapshot_raw = read_regular(snapshot_path, "offline snapshot manifest", MAX_JSON_BYTES)
    snapshot = parse_canonical_json(snapshot_raw, "offline snapshot manifest")
    state = validate_snapshot_manifest(
        snapshot_raw, snapshot, source_manifest, source_manifest_raw
    )
    state["source_commit"] = release["source"]["commit"]
    terminal_raw = read_regular(
        terminal_observer_path, "terminal observer manifest", MAX_JSON_BYTES
    )
    terminal_observer = parse_canonical_json(
        terminal_raw, "terminal observer manifest"
    )
    halt_trigger_ns = validate_terminal_observer(terminal_observer, snapshot)

    argv = [
        str(binary_path),
        "--home",
        str(home),
        "--backend",
        args.backend,
        "--chain-id",
        "zerone-1",
        "--expected-height",
        state["height"],
        "--expected-app-hash",
        state["app_hash"],
        "--source-commit",
        state["source_commit"],
        "--copied-db",
    ]

    started_epoch = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
    if cutover_signed > started_epoch:
        fail("census execution cannot precede the CUTOVER-initiation signature")
    if halt_trigger_ns > started_epoch * 1_000_000_000:
        fail("census execution cannot precede terminal block H")
    started_at = utc_second(started_epoch)
    try:
        with tempfile.TemporaryFile(dir=evidence_path.parent) as stdout_file, tempfile.TemporaryFile(
            dir=evidence_path.parent
        ) as stderr_file:
            try:
                result = subprocess.run(
                    argv,
                    stdin=subprocess.DEVNULL,
                    stdout=stdout_file,
                    stderr=stderr_file,
                    check=False,
                    timeout=MAX_EXECUTION_SECONDS,
                    close_fds=True,
                    preexec_fn=child_file_limit,
                )
            except (OSError, subprocess.TimeoutExpired) as exc:
                fail(f"custom-staking census did not complete safely: {exc}")
            completed_epoch = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
            stdout_file.seek(0)
            stdout_raw = stdout_file.read(MAX_CHILD_FILE_BYTES + 1)
            stderr_file.seek(0)
            stderr_raw = stderr_file.read(MAX_CHILD_FILE_BYTES + 1)
    except OSError as exc:
        fail(f"could not capture census process output: {exc}")
    stdout_hash = sha256(stdout_raw)
    stderr_hash = sha256(stderr_raw)
    if result.returncode != 0:
        fail(
            "custom-staking census failed: "
            f"exit={result.returncode} stdout_sha256={stdout_hash} stderr_sha256={stderr_hash}"
        )
    if stderr_raw:
        fail("successful custom-staking census emitted unexpected stderr")

    post_source_manifest, post_source_manifest_raw = validate_source_manifest(
        home, source_manifest_path, "after execution"
    )
    if not (
        post_source_manifest == source_manifest
        and post_source_manifest_raw == source_manifest_raw
    ):
        fail("source application.db or its file manifest changed during execution")
    final_binary_hash, _ = hash_regular(
        binary_path, "custom-staking census binary after execution", MAX_BINARY_BYTES
    )
    if final_binary_hash != binary_hash:
        fail("custom-staking census binary changed during execution")
    report_raw = stdout_raw
    report_hash, report_self_hash = validate_report(report_raw, state)
    atomic_publish(report_path, report_raw)

    created_epoch = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
    evidence = {
        "schema": EVIDENCE_SCHEMA,
        "result": "PASS",
        "created_at": utc_second(created_epoch),
        "release_packet": release_pair,
        "cutover_initiation_evidence": cutover_pair,
        "runner": {"path": RUNNER_PATH, "sha256": runner_hash},
        "state": {
            "chain_id": "zerone-1",
            "height": state["height"],
            "app_hash": state["app_hash"],
            "source_commit": state["source_commit"],
        },
        "binary": dict(contract["binary"]),
        "source_snapshot": {
            "manifest_filename": SNAPSHOT_MANIFEST_FILENAME,
            "manifest_sha256": state["manifest_sha256"],
            "database_snapshot_sha256": state["database_snapshot_sha256"],
            "file_manifest_sha256": state["file_manifest_sha256"],
        },
        "command": {
            "argv": argv,
            "binary_path": str(binary_path),
            "home_path": str(home),
            "backend": args.backend,
            "copied_db": True,
            "report_transport": REPORT_TRANSPORT,
            "output_path": str(report_path),
        },
        "execution": {
            "started_at": started_at,
            "completed_at": utc_second(completed_epoch),
            "exit_code": result.returncode,
            "stdout_sha256": stdout_hash,
            "stderr_sha256": stderr_hash,
            "report_filename": CENSUS_REPORT_FILENAME,
            "report_sha256": report_hash,
            "report_self_hash": report_self_hash,
            "report_result": "PASS",
        },
        "scan_guarantees": SCAN_GUARANTEES,
        "signature_authority": {
            "algorithm": "openpgp",
            "authorized_signer_fingerprint": contract["execution_evidence"][
                "authorized_signer_fingerprint"
            ],
            "detached_signature_filename": CENSUS_EVIDENCE_SIGNATURE_FILENAME,
            "authority_limit": AUTHORITY_LIMIT,
        },
    }
    evidence_raw = canonical_bytes(evidence)
    if len(evidence_raw) > MAX_JSON_BYTES:
        fail("custom-staking census execution evidence exceeds its byte limit")
    atomic_publish(evidence_path, evidence_raw)
    binary_snapshot_owner.cleanup()
    print(f"custom-staking-census-execution-evidence: PASS {sha256(evidence_raw)}")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    capture = subparsers.add_parser(
        "capture-manifest", help="capture the exact copied application DB file set"
    )
    capture.add_argument("--home", required=True)
    capture.add_argument("--output", required=True)

    run = subparsers.add_parser(
        "execute", help="run the exact RELEASE-bound census and emit canonical evidence"
    )
    run.add_argument("--release-packet", required=True)
    run.add_argument("--release-signature", required=True)
    run.add_argument("--operator-tool-manifest", required=True)
    run.add_argument("--authority-bundle", required=True)
    run.add_argument("--cutover-decision", required=True)
    run.add_argument("--cutover-decision-signature", required=True)
    run.add_argument("--cutover-initiation-evidence", required=True)
    run.add_argument("--cutover-initiation-signature", required=True)
    run.add_argument("--snapshot-manifest", required=True)
    run.add_argument("--terminal-observer-manifest", required=True)
    run.add_argument("--source-file-manifest", required=True)
    run.add_argument("--binary", required=True)
    run.add_argument("--home", required=True)
    run.add_argument("--backend", choices=("goleveldb", "pebbledb"), required=True)
    run.add_argument("--report-output", required=True)
    run.add_argument("--evidence-output", required=True)
    run.add_argument("--expected-release-signer-fingerprint", required=True)
    run.add_argument("--expected-transition-signer-fingerprint", required=True)
    run.add_argument("--config-policy", required=True)
    run.add_argument("--tool-root", required=True)

    args = parser.parse_args()
    if args.command == "capture-manifest":
        capture_manifest(args)
    else:
        execute(args)


if __name__ == "__main__":
    try:
        main()
    except EvidenceError as exc:
        print(f"custom-staking-census-evidence: ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
