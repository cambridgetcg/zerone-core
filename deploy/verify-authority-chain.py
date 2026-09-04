#!/usr/bin/env python3
"""Verify the transitive signed authority chain before irreversible phases."""

from __future__ import annotations

import argparse
import base64
import binascii
import datetime as dt
import hashlib
import importlib.util
import json
import os
import pathlib
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import tomllib
from typing import Any
from urllib.parse import urlsplit


HASH = re.compile(r"^[0-9a-f]{64}$")
UPPER_HASH = re.compile(r"^[0-9A-F]{64}$")
FINGERPRINT = re.compile(r"^[0-9A-Fa-f]{40}(?:[0-9A-Fa-f]{24})?$")
POSITIVE_HEIGHT = re.compile(r"^[1-9][0-9]{0,17}$")
IMAGE_REF = re.compile(
    r"^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?(?::[0-9]{1,5})?/"
    r"[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*"
    r"@sha256:[0-9a-f]{64}$"
)
OPERATOR_TOOL_PATHS = {
    ".github/workflows/ci.yml",
    "deploy/verify-authority-chain.py",
    "deploy/frozen_evidence.py",
    "deploy/run-custom-staking-census-evidence.py",
    "deploy/validate-fly-phase-config.py",
    "deploy/fly-deploy-pinned.sh",
    "deploy/fly-deploy-authorized.sh",
    "deploy/mainnet/render-archive-configs.sh",
    "deploy/mainnet/fly-deploy-archive-authorized.sh",
    "deploy/mainnet/fly-cutover-authorized.sh",
    "scripts/zerone-phase-tx-broadcast.sh",
    "scripts/zerone-2-bootstrap-tx-broadcast.sh",
    "scripts/zerone-canonical-json.sh",
    "scripts/zerone-2-ceremony.sh",
    "deploy/mainnet/build-image.sh",
    "deploy/networks/zerone-2/runtime/build-image.sh",
    "deploy/query-gateway/build-image.sh",
    "deploy/query-gateway/render-archive-gateway-config.py",
    "deploy/query-gateway/fly.zerone-1-archive.public.example.toml",
    "tools/zerone2-artifact-audit/main.go",
    "tools/sigstore-substrate-compiler/go.mod",
    "tools/sigstore-substrate-compiler/go.sum",
    "tools/sigstore-substrate-compiler/verification/verification.go",
    "tools/sigstore-substrate-compiler/verification/component.go",
    "tools/sigstore-substrate-compiler/cmd/zerone-component-signature-verifier/main.go",
    "deploy/networks/zerone-2/ARCHIVE-ADOPTION-AUTHORITY.example.json",
    "deploy/networks/zerone-1/frozen/FINAL-CHECKPOINT.example.json",
    "deploy/networks/zerone-2/OPEN-BETA-DECISION.example.json",
    "deploy/networks/zerone-2/PRE-NOTICE-DECISION.example.json",
    "deploy/networks/zerone-2/PUBLIC-NOTICE-PUBLICATION-EVIDENCE.example.json",
}
NOTICE_MAX_BYTES = 256 * 1024
NOTICE_SCOPE = {
    "publish_exact_notice": True,
    "broadcast_transaction": False,
    "halt_chain": False,
    "deploy_service": False,
    "publish_network_coordinates": False,
    "change_dns": False,
}
CENSUS_BINARY_FILENAME = "custom-staking-census-linux-amd64"
CENSUS_EXECUTION_RUNNER_PATH = "deploy/run-custom-staking-census-evidence.py"
CENSUS_EXECUTION_EVIDENCE_FILENAME = (
    "CUSTOM-STAKING-CENSUS-EXECUTION-EVIDENCE.json"
)
CENSUS_EXECUTION_SIGNATURE_FILENAME = (
    "CUSTOM-STAKING-CENSUS-EXECUTION-EVIDENCE.json.sig"
)
CENSUS_EXECUTION_AUTHORITY_LIMIT = (
    "factual attestation that the exact RELEASE-bound census binary scanned the "
    "declared stopped observer copy and produced the bound report; no migration, "
    "deployment, transaction, public-service, or DNS authority"
)
CENSUS_REPORT_TRANSPORT = "stdout-captured-and-atomically-published"
MAX_CENSUS_EXECUTION_SECONDS = 6 * 60 * 60
ARCHIVE_GATEWAY_RENDERER_PATH = (
    "deploy/query-gateway/render-archive-gateway-config.py"
)
ARCHIVE_GATEWAY_TEMPLATE_PATH = (
    "deploy/query-gateway/fly.zerone-1-archive.public.example.toml"
)
ARCHIVE_GATEWAY_BINDINGS = {
    "app": "deployment_configs.zerone_1_archive_gateway.app",
    "primary_region": "archive_render_contract.static_constraints.region",
    "image_ref": "deployment_configs.zerone_1_archive_gateway.image_ref",
    "upstream_host": 'archive_render_contract.static_constraints.app + ".internal"',
    "expected_archive_height": (
        "FINAL-CHECKPOINT.json.final_application_block.height"
    ),
    "expected_archive_app_hash": (
        "lower(FINAL-CHECKPOINT.json.excluded_post_anchor_state.app_hash)"
    ),
    "expected_archive_block_hash": (
        "lower(FINAL-CHECKPOINT.json.final_application_block.block_id_hash)"
    ),
}
COMPONENT_SIGNATURE_VERIFIER_FILE = "zerone-component-signature-verifier"
SIGSTORE_TRUSTED_ROOT_FILE = "SIGSTORE-TRUSTED-ROOT.json"
COMPONENT_SIGNATURE_BUNDLE_MEDIA_TYPE = (
    "application/vnd.dev.sigstore.bundle.v0.3+json"
)
COMPONENT_ARTIFACT_FILES = {
    "zerone_1_halt": {
        "sbom": "ZERONE-1-HALT-SBOM.json",
        "provenance": "ZERONE-1-HALT-PROVENANCE.json",
        "signature": "ZERONE-1-HALT-SIGNATURE-EVIDENCE.json",
        "signature_bundle": "ZERONE-1-HALT-SIGNATURE-BUNDLE.json",
        "vulnerability_decision": "ZERONE-1-HALT-VULNERABILITY-DECISION.json",
        "vulnerability_scan": "ZERONE-1-HALT-VULNERABILITY-SCAN.json",
    },
    "zerone_2_runtime": {
        "sbom": "ZERONE-2-RUNTIME-SBOM.json",
        "provenance": "ZERONE-2-RUNTIME-PROVENANCE.json",
        "signature": "ZERONE-2-RUNTIME-SIGNATURE-EVIDENCE.json",
        "signature_bundle": "ZERONE-2-RUNTIME-SIGNATURE-BUNDLE.json",
        "vulnerability_decision": "ZERONE-2-RUNTIME-VULNERABILITY-DECISION.json",
        "vulnerability_scan": "ZERONE-2-RUNTIME-VULNERABILITY-SCAN.json",
    },
    "query_gateway": {
        "sbom": "QUERY-GATEWAY-SBOM.json",
        "provenance": "QUERY-GATEWAY-PROVENANCE.json",
        "signature": "QUERY-GATEWAY-SIGNATURE-EVIDENCE.json",
        "signature_bundle": "QUERY-GATEWAY-SIGNATURE-BUNDLE.json",
        "vulnerability_decision": "QUERY-GATEWAY-VULNERABILITY-DECISION.json",
        "vulnerability_scan": "QUERY-GATEWAY-VULNERABILITY-SCAN.json",
    },
}
COMPONENT_BUILD_RECIPES = {
    "zerone_1_halt": "deploy/mainnet/build-image.sh",
    "zerone_2_runtime": "deploy/networks/zerone-2/runtime/build-image.sh",
    "query_gateway": "deploy/query-gateway/build-image.sh",
}
CANONICAL_COMPONENT_SIGNER_IDENTITY = (
    "https://github.com/cambridgetcg/zerone-core/"
    ".github/workflows/ci.yml@refs/heads/main"
)
CANONICAL_COMPONENT_CERTIFICATE_ISSUER = (
    "https://token.actions.githubusercontent.com"
)
MONITORING_ARTIFACT_FILES = {
    "manifest": "MONITORING-ALERTS.json",
    "rules": "MONITORING-RULES.json",
    "tests": "MONITORING-ALERT-TESTS.json",
}
MONITORING_RULE_SPECS = {
    "stalled_height": {
        "alert_name": "ZeroneStalledHeight",
        "severity": "critical",
        "expression": "consensus_height_no_progress",
        "parameters": {"maximum_stall_seconds"},
        "stimulus": "hold_height_without_progress_past_threshold",
    },
    "missed_signing": {
        "alert_name": "ZeroneValidatorMissedSigning",
        "severity": "critical",
        "expression": "validator_missed_blocks_above_threshold",
        "parameters": {"maximum_missed_blocks", "window_blocks"},
        "stimulus": "inject_validator_missed_blocks_above_threshold",
    },
    "double_sign_risk": {
        "alert_name": "ZeroneDoubleSignRisk",
        "severity": "critical",
        "expression": "active_signer_instances_above_threshold",
        "parameters": {"maximum_active_signer_instances"},
        "stimulus": "inject_signer_instance_count_above_threshold",
    },
    "app_hash_divergence": {
        "alert_name": "ZeroneAppHashDivergence",
        "severity": "critical",
        "expression": "equal_height_app_hashes_diverge",
        "parameters": {
            "maximum_distinct_app_hashes",
            "minimum_independent_sources",
        },
        "stimulus": "inject_equal_height_mismatched_app_hash",
    },
    "peer_loss": {
        "alert_name": "ZeronePeerLoss",
        "severity": "critical",
        "expression": "private_peer_count_below_threshold",
        "parameters": {"minimum_private_peers"},
        "stimulus": "disconnect_all_private_peers",
    },
    "disk_capacity": {
        "alert_name": "ZeroneDiskCapacity",
        "severity": "warning",
        "expression": "disk_free_percent_below_threshold",
        "parameters": {"minimum_free_percent"},
        "stimulus": "inject_disk_free_percent_below_threshold",
    },
    "restart_count": {
        "alert_name": "ZeroneRestartCount",
        "severity": "warning",
        "expression": "process_restarts_above_threshold",
        "parameters": {"maximum_restarts", "window_seconds"},
        "stimulus": "inject_restart_counter_above_threshold",
    },
    "stale_backup": {
        "alert_name": "ZeroneStaleBackup",
        "severity": "critical",
        "expression": "verified_backup_age_above_threshold",
        "parameters": {"maximum_verified_backup_age_seconds"},
        "stimulus": "inject_verified_backup_age_above_threshold",
    },
    "gateway_wrong_chain": {
        "alert_name": "ZeroneGatewayWrongChain",
        "severity": "critical",
        "expression": "gateway_chain_id_mismatch",
        "parameters": {"expected_chain_id"},
        "stimulus": "inject_gateway_chain_id_mismatch",
    },
    "gateway_stale_origin": {
        "alert_name": "ZeroneGatewayStaleOrigin",
        "severity": "critical",
        "expression": "gateway_origin_height_lag_above_threshold",
        "parameters": {"maximum_height_lag"},
        "stimulus": "inject_gateway_origin_height_lag_above_threshold",
    },
}
MONITORING_EVIDENCE_KINDS = (
    "stimulus",
    "firing",
    "notification",
    "resolution",
)
MONITORING_EVIDENCE_FILES = {
    check: {
        kind: (
            f"MONITORING-ALERT-{check.replace('_', '-').upper()}-"
            f"{kind.upper()}-EVIDENCE.json.raw"
        )
        for kind in MONITORING_EVIDENCE_KINDS
    }
    for check in MONITORING_RULE_SPECS
}
MONITORING_EVIDENCE_FILENAMES = frozenset(
    filename
    for files_by_kind in MONITORING_EVIDENCE_FILES.values()
    for filename in files_by_kind.values()
)
FROZEN_EVIDENCE_FILES = {
    "CUSTOM-STAKING-CENSUS.json",
    CENSUS_EXECUTION_EVIDENCE_FILENAME,
    CENSUS_EXECUTION_SIGNATURE_FILENAME,
    "ZERONE-1-INVENTORY-V3.json",
    "SIGNER-EVIDENCE-MANIFEST.json",
    "OBSERVER-EVIDENCE-MANIFEST.json",
    "POST-ANCHOR-STATE-EXPORT.json.raw",
    "POST-ANCHOR-STATE-EXPORT-EVIDENCE.json",
    "OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json",
    "PRE-TRANSITION-SANITIZED-SNAPSHOT-MANIFEST.json",
    "ARCHIVE-ROLLBACK-OUTPUT.log",
    "ARCHIVE-ROLLBACK-LOG.json",
}
for _prefix in ("SIGNER", "OBSERVER"):
    for _suffix in (
        "STATUS.json.raw",
        "GENESIS.json.raw",
        "TRUSTED-BLOCK.json.raw",
        "TRUSTED-COMMIT.json.raw",
        "TRUSTED-VALIDATORS.json.raw",
        "BLOCK-A.json.raw",
        "COMMIT-A.json.raw",
        "VALIDATORS-A.json.raw",
        "BLOCK-RESULTS-A.json.raw",
        "BLOCK-H.json.raw",
        "COMMIT-H.json.raw",
        "VALIDATORS-H.json.raw",
        "ABCI-INFO.json.raw",
        "BLOCK-RESULTS-H-MISSING.json.raw",
    ):
        FROZEN_EVIDENCE_FILES.add(f"{_prefix}-RPC-{_suffix}")


def fail(message: str) -> None:
    raise SystemExit(f"verify-authority-chain: {message}")


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def secure_read_path(path: pathlib.Path, label: str) -> bytes:
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
    try:
        fd = os.open(path, flags)
    except OSError as exc:
        fail(f"could not open {label}: {exc}")
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode):
            fail(f"{label} is not a regular file")
        chunks: list[bytes] = []
        while True:
            chunk = os.read(fd, 1024 * 1024)
            if not chunk:
                break
            chunks.append(chunk)
        return b"".join(chunks)
    finally:
        os.close(fd)


def bundle_file_size_limit(name: str) -> int:
    if name in {"PUBLIC-NOTICE.md", "PUBLIC-NOTICE-CAPTURE.md"}:
        return NOTICE_MAX_BYTES
    if name.startswith("zeroned-") or name == CENSUS_BINARY_FILENAME:
        return 384 * 1024 * 1024
    if name == "POST-ANCHOR-STATE-EXPORT.json.raw":
        return 384 * 1024 * 1024
    if name in MONITORING_EVIDENCE_FILENAMES:
        return 16 * 1024 * 1024
    if name.endswith(".json.raw"):
        return 64 * 1024 * 1024
    if name == "ZERONE-1-INVENTORY-V3.json":
        return 128 * 1024 * 1024
    if name == "CUSTOM-STAKING-CENSUS.json":
        # The census seals at <=64 MiB, then its atomic publisher appends one LF.
        return 64 * 1024 * 1024 + 1
    if name == COMPONENT_SIGNATURE_VERIFIER_FILE:
        return 64 * 1024 * 1024
    return 32 * 1024 * 1024


def secure_read_bundle(bundle_fd: int, name: str) -> bytes:
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
    try:
        fd = os.open(name, flags, dir_fd=bundle_fd)
    except OSError as exc:
        fail(f"could not open bundle file {name}: {exc}")
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode):
            fail(f"bundle file {name} is not regular")
        size_limit = bundle_file_size_limit(name)
        if info.st_size < 0 or info.st_size > size_limit:
            fail(f"bundle file {name} exceeds its pre-authentication size limit")
        chunks: list[bytes] = []
        total = 0
        while True:
            chunk = os.read(fd, 1024 * 1024)
            if not chunk:
                break
            total += len(chunk)
            if total > size_limit:
                fail(f"bundle file {name} grew beyond its size limit while reading")
            chunks.append(chunk)
        return b"".join(chunks)
    finally:
        os.close(fd)


def parse_json(data: bytes, label: str) -> Any:
    def reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
        value: dict[str, Any] = {}
        for key, item in pairs:
            if key in value:
                raise ValueError(f"duplicate object key {key!r}")
            value[key] = item
        return value

    try:
        return json.loads(data, object_pairs_hook=reject_duplicate_keys)
    except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
        fail(f"{label} is not valid JSON: {exc}")


def canonical_epoch(value: Any, label: str) -> int:
    if not isinstance(value, str) or not re.fullmatch(
        r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z", value
    ):
        fail(f"{label} is not a canonical UTC second")
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
        fail(f"{label} is not an RFC3339 UTC timestamp")
    match = re.fullmatch(
        r"([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2})"
        r"(?:\.([0-9]{1,9}))?Z",
        value,
    )
    if match is None:
        fail(f"{label} is not canonical RFC3339 UTC with at most nanoseconds")
    try:
        parsed = dt.datetime.strptime(match.group(1), "%Y-%m-%dT%H:%M:%S").replace(
            tzinfo=dt.timezone.utc
        )
    except ValueError as exc:
        fail(f"{label} is not a real UTC time: {exc}")
    if parsed.strftime("%Y-%m-%dT%H:%M:%S") != match.group(1):
        fail(f"{label} is not canonical RFC3339 UTC")
    raw_fraction = match.group(2) or ""
    if raw_fraction.endswith("0"):
        fail(f"{label} has non-canonical trailing fractional zeros")
    fraction = raw_fraction.ljust(9, "0")
    return int(parsed.timestamp()) * 1_000_000_000 + int(fraction or "0")


def require_hash(value: Any, label: str, upper: bool = False) -> str:
    pattern = UPPER_HASH if upper else HASH
    if not isinstance(value, str) or not pattern.fullmatch(value):
        fail(f"{label} is not an exact {'uppercase ' if upper else ''}SHA-256")
    return value


def require_exact_object(value: Any, keys: set[str], label: str) -> dict[str, Any]:
    if not isinstance(value, dict) or set(value) != keys:
        fail(f"{label} does not have the exact required fields")
    return value


def require_nonempty_string(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value or value.strip() != value:
        fail(f"{label} must be a non-empty, surrounding-whitespace-free string")
    return value


def require_nonnegative_integer(value: Any, label: str) -> int:
    if not isinstance(value, int) or isinstance(value, bool) or value < 0:
        fail(f"{label} must be a non-negative integer")
    return value


def decode_base64(value: Any, label: str) -> bytes:
    if not isinstance(value, str) or not value:
        fail(f"{label} must be non-empty base64")
    try:
        return base64.b64decode(value, validate=True)
    except (binascii.Error, ValueError) as exc:
        fail(f"{label} is not canonical base64: {exc}")


def require_pair(value: Any, label: str) -> dict[str, str]:
    if not isinstance(value, dict) or set(value) != {
        "sha256",
        "detached_signature_sha256",
    }:
        fail(f"{label} is not an exact payload/signature hash pair")
    require_hash(value["sha256"], f"{label}.sha256")
    require_hash(
        value["detached_signature_sha256"],
        f"{label}.detached_signature_sha256",
    )
    return value


def exact_pair(files: dict[str, bytes], payload: str, signature: str) -> dict[str, str]:
    return {
        "sha256": sha256(files[payload]),
        "detached_signature_sha256": sha256(files[signature]),
    }


def validate_config_mapping(
    files: dict[str, bytes], filename: str, mapping: Any, label: str
) -> dict[str, Any]:
    if not isinstance(mapping, dict) or set(mapping) != {
        "app",
        "role",
        "image_component",
        "image_ref",
        "sha256",
    }:
        fail(f"{label} mapping is not exact")
    require_hash(mapping.get("sha256"), f"{label} config hash")
    if mapping["sha256"] != sha256(files[filename]):
        fail(f"{label} config bytes differ from the signed mapping")
    try:
        config = tomllib.loads(files[filename].decode())
    except (UnicodeDecodeError, tomllib.TOMLDecodeError) as exc:
        fail(f"{label} config is not valid TOML: {exc}")
    if config.get("app") != mapping["app"]:
        fail(f"{label} config app differs from its signed mapping")
    if config.get("build", {}).get("image") != mapping["image_ref"]:
        fail(f"{label} config image differs from its signed mapping")
    role = config.get("env", {}).get("NODE_ROLE") or config.get("env", {}).get(
        "GATEWAY_ROLE"
    )
    if role != mapping["role"]:
        fail(f"{label} config role differs from its signed mapping")
    return config


def validate_archive_gateway_render_contract(
    release: dict[str, Any],
) -> dict[str, Any]:
    contract = require_exact_object(
        release.get("archive_gateway_render_contract"),
        {
            "schema",
            "renderer_path",
            "renderer_sha256",
            "template_path",
            "bindings",
        },
        "RELEASE archive gateway render contract",
    )
    if not (
        contract["schema"] == "zerone-1-archive-gateway-render-contract-v1"
        and contract["renderer_path"] == ARCHIVE_GATEWAY_RENDERER_PATH
        and contract["template_path"] == ARCHIVE_GATEWAY_TEMPLATE_PATH
        and contract["bindings"] == ARCHIVE_GATEWAY_BINDINGS
    ):
        fail("RELEASE archive gateway render contract changed")
    require_hash(contract["renderer_sha256"], "RELEASE archive gateway renderer")
    template_hashes = release.get("phase_dependent_config_template_sha256")
    if not isinstance(template_hashes, dict) or set(template_hashes) != {
        "zerone_1_halt_signer",
        "zerone_1_observer",
        "zerone_1_archive_candidate",
        "zerone_1_archive",
        "zerone_1_archive_gateway",
    }:
        fail("RELEASE phase-dependent template hash set is incomplete")
    require_hash(
        template_hashes["zerone_1_archive_gateway"],
        "RELEASE archive gateway template",
    )
    return contract


def validate_census_execution_contract(
    files: dict[str, bytes], release: dict[str, Any]
) -> dict[str, Any]:
    contract = require_exact_object(
        release.get("custom_staking_census_execution"),
        {"schema", "binary", "execution_evidence"},
        "RELEASE custom-staking census execution contract",
    )
    binary = require_exact_object(
        contract["binary"],
        {"filename", "sha256"},
        "RELEASE custom-staking census binary",
    )
    evidence = require_exact_object(
        contract["execution_evidence"],
        {
            "filename",
            "detached_signature_filename",
            "authorized_signer_fingerprint",
        },
        "RELEASE custom-staking census execution evidence",
    )
    transition = (
        release.get("public_identities", {})
        .get("transition_attestation", {})
        .get("authorized_signer_fingerprint")
    )
    if not (
        contract["schema"]
        == "zerone-custom-staking-census-execution-contract-v1"
        and binary["filename"] == CENSUS_BINARY_FILENAME
        and evidence["filename"] == CENSUS_EXECUTION_EVIDENCE_FILENAME
        and evidence["detached_signature_filename"]
        == CENSUS_EXECUTION_SIGNATURE_FILENAME
        and isinstance(evidence["authorized_signer_fingerprint"], str)
        and FINGERPRINT.fullmatch(evidence["authorized_signer_fingerprint"])
        and isinstance(transition, str)
        and normalize_fingerprint(evidence["authorized_signer_fingerprint"])
        == normalize_fingerprint(transition)
    ):
        fail("RELEASE custom-staking census execution contract is unsafe")
    binary_hash = require_hash(
        binary["sha256"], "RELEASE custom-staking census binary"
    )
    if sha256(files[CENSUS_BINARY_FILENAME]) != binary_hash:
        fail("bundled custom-staking census binary differs from RELEASE")
    return contract


def run_config_policy(
    policy: pathlib.Path,
    config_path: pathlib.Path,
    schema: str,
    key: str,
    upstream: str,
    f: str,
    a: str,
    h: str,
    archive_app_hash: str,
    archive_block_hash: str,
) -> None:
    try:
        subprocess.run(
            [
                sys.executable,
                str(policy),
                str(config_path),
                schema,
                key,
                upstream,
                f,
                a,
                h,
                archive_app_hash,
                archive_block_hash,
            ],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        detail = getattr(exc, "stderr", b"")
        if isinstance(detail, bytes):
            detail = detail.decode(errors="replace").strip()
        fail(f"{key} config violates structural policy: {detail or exc}")


def parse_runtime_marker(data: bytes, label: str) -> dict[str, str]:
    try:
        lines = data.decode().splitlines()
    except UnicodeDecodeError as exc:
        fail(f"{label} is not UTF-8: {exc}")
    values: dict[str, str] = {}
    for line in lines:
        if not line or "=" not in line:
            fail(f"{label} contains a malformed line")
        key, value = line.split("=", 1)
        if not key or not value or key in values:
            fail(f"{label} contains an empty or duplicate key")
        values[key] = value
    return values


def validate_tool_manifest(
    files: dict[str, bytes], objects: dict[str, Any], tool_root: pathlib.Path
) -> dict[str, bytes]:
    release = objects["RELEASE-PACKET.json"]
    manifest = objects["OPERATOR-TOOL-MANIFEST.json"]
    if release.get("operator_tool_manifest_sha256") != sha256(
        files["OPERATOR-TOOL-MANIFEST.json"]
    ):
        fail("RELEASE operator-tool manifest hash differs from the bundle")
    if not isinstance(manifest, dict) or set(manifest) != {
        "schema",
        "source_commit",
        "signed_tag",
        "files",
        "authority_bundle",
        "component_signature_policy",
    } or not (
        manifest["schema"] == "zerone-operator-tool-manifest-v2"
        and manifest["source_commit"] == release.get("source", {}).get("commit")
        and manifest["signed_tag"]
        == release.get("source", {}).get("signed_annotated_tag")
        and isinstance(manifest["files"], dict)
        and set(manifest["files"]) == OPERATOR_TOOL_PATHS
    ):
        fail("operator-tool manifest schema/source/path set is inconsistent")
    authority_bundle = require_exact_object(
        manifest["authority_bundle"],
        {"component_signature_verifier", "sigstore_trusted_root"},
        "operator-tool authority bundle",
    )
    verifier_artifact = require_exact_object(
        authority_bundle["component_signature_verifier"],
        {"filename", "sha256"},
        "component-signature verifier artifact",
    )
    trusted_root_artifact = require_exact_object(
        authority_bundle["sigstore_trusted_root"],
        {"filename", "sha256"},
        "Sigstore trusted-root artifact",
    )
    if not (
        verifier_artifact["filename"] == COMPONENT_SIGNATURE_VERIFIER_FILE
        and trusted_root_artifact["filename"] == SIGSTORE_TRUSTED_ROOT_FILE
    ):
        fail("operator-tool authority-bundle filenames changed")
    for artifact, label in (
        (verifier_artifact, "component-signature verifier"),
        (trusted_root_artifact, "Sigstore trusted root"),
    ):
        expected = require_hash(artifact["sha256"], f"{label} artifact")
        filename = artifact["filename"]
        if expected != sha256(files[filename]):
            fail(f"bundled {label} bytes differ from the operator-tool manifest")
    signature_policy = require_exact_object(
        manifest["component_signature_policy"],
        {
            "bundle_media_type",
            "certificate_issuer",
            "certificate_san",
            "certificate_source_repository_digest",
            "minimum_signed_certificate_timestamps",
            "minimum_transparency_log_entries",
            "minimum_observer_timestamps",
        },
        "component-signature policy",
    )
    threshold_fields = (
        "minimum_signed_certificate_timestamps",
        "minimum_transparency_log_entries",
        "minimum_observer_timestamps",
    )
    if any(type(signature_policy[field]) is not int for field in threshold_fields):
        fail("component-signature policy thresholds must be exact integers")
    if signature_policy != {
        "bundle_media_type": COMPONENT_SIGNATURE_BUNDLE_MEDIA_TYPE,
        "certificate_issuer": CANONICAL_COMPONENT_CERTIFICATE_ISSUER,
        "certificate_san": CANONICAL_COMPONENT_SIGNER_IDENTITY,
        "certificate_source_repository_digest": "RELEASE.source.commit",
        "minimum_signed_certificate_timestamps": 1,
        "minimum_transparency_log_entries": 1,
        "minimum_observer_timestamps": 1,
    }:
        fail("component-signature policy is not the canonical fail-closed policy")
    try:
        root_info = os.lstat(tool_root)
    except OSError as exc:
        fail(f"could not inspect operator tool root: {exc}")
    if not stat.S_ISDIR(root_info.st_mode) or stat.S_ISLNK(root_info.st_mode):
        fail("operator tool root must be a real directory")
    verified_tools: dict[str, bytes] = {}
    for relative, expected in manifest["files"].items():
        require_hash(expected, f"operator tool {relative}")
        tool_bytes = secure_read_path(tool_root / relative, f"operator tool {relative}")
        actual = sha256(tool_bytes)
        if actual != expected:
            fail(f"operator tool bytes drifted from RELEASE: {relative}")
        verified_tools[relative] = tool_bytes
    gateway_contract = validate_archive_gateway_render_contract(release)
    if manifest["files"].get(ARCHIVE_GATEWAY_RENDERER_PATH) != gateway_contract[
        "renderer_sha256"
    ]:
        fail("archive gateway renderer differs from RELEASE/operator-tool manifest")
    if manifest["files"].get(ARCHIVE_GATEWAY_TEMPLATE_PATH) != release[
        "phase_dependent_config_template_sha256"
    ]["zerone_1_archive_gateway"]:
        fail("archive gateway template differs from RELEASE/operator-tool manifest")
    return verified_tools


def verify_archive_gateway_render(
    files: dict[str, bytes],
    paths: dict[str, pathlib.Path],
    release: dict[str, Any],
    verified_tools: dict[str, bytes],
    temp_path: pathlib.Path,
) -> None:
    contract = validate_archive_gateway_render_contract(release)
    renderer_bytes = verified_tools.get(contract["renderer_path"])
    template_bytes = verified_tools.get(contract["template_path"])
    if renderer_bytes is None or template_bytes is None:
        fail("verified operator tools omit the archive gateway renderer/template")
    if sha256(renderer_bytes) != contract["renderer_sha256"]:
        fail("verified archive gateway renderer hash differs from RELEASE")
    if sha256(template_bytes) != release[
        "phase_dependent_config_template_sha256"
    ]["zerone_1_archive_gateway"]:
        fail("verified archive gateway template hash differs from RELEASE")

    renderer = temp_path / "render-archive-gateway-config.py"
    template = temp_path / "fly.archive-gateway.template.toml"
    rendered = temp_path / "fly.archive-gateway.expected.toml"
    renderer.write_bytes(renderer_bytes)
    template.write_bytes(template_bytes)
    os.chmod(renderer, 0o700)
    os.chmod(template, 0o600)
    try:
        subprocess.run(
            [
                sys.executable,
                str(renderer),
                str(paths["RELEASE-PACKET.json"]),
                str(paths["FINAL-CHECKPOINT.json"]),
                str(template),
                str(rendered),
            ],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
            timeout=10,
        )
    except (OSError, subprocess.CalledProcessError, subprocess.TimeoutExpired) as exc:
        detail = getattr(exc, "stderr", b"")
        if isinstance(detail, bytes):
            detail = detail.decode(errors="replace").strip()
        fail(f"archive gateway deterministic render failed: {detail or exc}")
    rendered_bytes = secure_read_path(rendered, "rendered archive gateway config")
    if rendered_bytes != files["fly.zerone-1-archive-gateway.public.toml"]:
        fail("archive gateway config is not the exact RELEASE/FINAL render")


def validate_monitoring_artifacts(
    files: dict[str, bytes],
    objects: dict[str, Any],
    release: dict[str, Any],
    release_created_epoch: int,
) -> None:
    manifest_name = MONITORING_ARTIFACT_FILES["manifest"]
    rules_name = MONITORING_ARTIFACT_FILES["rules"]
    tests_name = MONITORING_ARTIFACT_FILES["tests"]

    expected_manifest_hash = require_hash(
        release.get("monitoring_alerts_sha256"), "RELEASE monitoring alerts"
    )
    if expected_manifest_hash != sha256(files[manifest_name]):
        fail("RELEASE monitoring-alert hash differs from bundled MONITORING-ALERTS.json")

    manifest = require_exact_object(
        objects[manifest_name],
        {
            "schema",
            "chain_id",
            "source_commit",
            "created_at",
            "rules",
            "alert_tests",
            "result",
        },
        "monitoring alert manifest",
    )
    rules_reference = require_exact_object(
        manifest["rules"], {"filename", "sha256"}, "monitoring rules reference"
    )
    tests_reference = require_exact_object(
        manifest["alert_tests"],
        {"filename", "sha256"},
        "monitoring alert-test reference",
    )
    if not (
        manifest["schema"] == "zerone-production-monitoring-alerts-v1"
        and manifest["chain_id"] == release.get("chain_id") == "zerone-2"
        and manifest["source_commit"] == release.get("source", {}).get("commit")
        and manifest["result"] == "PASS"
        and rules_reference["filename"] == rules_name
        and tests_reference["filename"] == tests_name
    ):
        fail("monitoring alert manifest is incomplete or mismatched")
    rules_hash = require_hash(
        rules_reference["sha256"], "monitoring rules artifact"
    )
    tests_hash = require_hash(
        tests_reference["sha256"], "monitoring alert-test artifact"
    )
    if rules_hash != sha256(files[rules_name]):
        fail("monitoring rules hash differs from bundled MONITORING-RULES.json")
    if tests_hash != sha256(files[tests_name]):
        fail("monitoring alert-test hash differs from bundled test evidence")

    rules = require_exact_object(
        objects[rules_name],
        {
            "schema",
            "chain_id",
            "source_commit",
            "ruleset_id",
            "evaluation_interval_seconds",
            "notification_route_id",
            "rules",
        },
        "monitoring rules artifact",
    )
    if not (
        rules["schema"] == "zerone-production-monitoring-rules-v1"
        and rules["chain_id"] == "zerone-2"
        and rules["source_commit"] == release["source"]["commit"]
        and isinstance(rules["ruleset_id"], str)
        and re.fullmatch(r"[a-z0-9][a-z0-9._-]{0,63}", rules["ruleset_id"])
        and isinstance(rules["notification_route_id"], str)
        and re.fullmatch(
            r"[a-z0-9][a-z0-9._-]{0,63}", rules["notification_route_id"]
        )
        and isinstance(rules["evaluation_interval_seconds"], int)
        and not isinstance(rules["evaluation_interval_seconds"], bool)
        and 1 <= rules["evaluation_interval_seconds"] <= 60
        and isinstance(rules["rules"], list)
        and len(rules["rules"]) == len(MONITORING_RULE_SPECS)
    ):
        fail("monitoring rules artifact is incomplete or mismatched")

    rules_by_check: dict[str, dict[str, Any]] = {}
    for index, candidate in enumerate(rules["rules"]):
        rule = require_exact_object(
            candidate,
            {
                "check",
                "alert_name",
                "enabled",
                "severity",
                "expression",
                "parameters",
            },
            f"monitoring rule {index}",
        )
        check = rule["check"]
        if not isinstance(check, str) or check not in MONITORING_RULE_SPECS:
            fail(f"monitoring rule {index} has an unknown check")
        if check in rules_by_check:
            fail(f"monitoring rules contain duplicate check {check}")
        spec = MONITORING_RULE_SPECS[check]
        parameters = rule["parameters"]
        if not (
            rule["alert_name"] == spec["alert_name"]
            and rule["enabled"] is True
            and rule["severity"] == spec["severity"]
            and rule["expression"] == spec["expression"]
            and isinstance(parameters, dict)
            and set(parameters) == spec["parameters"]
        ):
            fail(f"monitoring rule {check} is disabled or semantically incomplete")
        rules_by_check[check] = rule
    if set(rules_by_check) != set(MONITORING_RULE_SPECS):
        fail("monitoring rules omit a required production check")

    def bounded_parameter(check: str, name: str, minimum: int, maximum: int) -> int:
        value = rules_by_check[check]["parameters"][name]
        if (
            not isinstance(value, int)
            or isinstance(value, bool)
            or value < minimum
            or value > maximum
        ):
            fail(
                f"monitoring rule {check} parameter {name} is outside the safe range"
            )
        return value

    maximum_stall = bounded_parameter(
        "stalled_height", "maximum_stall_seconds", 15, 300
    )
    if maximum_stall < rules["evaluation_interval_seconds"]:
        fail("stalled-height threshold is shorter than the evaluation interval")
    maximum_missed = bounded_parameter(
        "missed_signing", "maximum_missed_blocks", 0, 9
    )
    window_blocks = bounded_parameter("missed_signing", "window_blocks", 10, 10000)
    if maximum_missed >= window_blocks:
        fail("missed-signing threshold must be smaller than its window")
    if (
        bounded_parameter(
            "double_sign_risk", "maximum_active_signer_instances", 1, 1
        )
        != 1
        or bounded_parameter(
            "app_hash_divergence", "maximum_distinct_app_hashes", 1, 1
        )
        != 1
        or bounded_parameter(
            "app_hash_divergence", "minimum_independent_sources", 2, 16
        )
        < 2
        or bounded_parameter("peer_loss", "minimum_private_peers", 1, 64) < 1
    ):
        fail("monitoring consensus-safety thresholds are incomplete")
    bounded_parameter("disk_capacity", "minimum_free_percent", 10, 40)
    bounded_parameter("restart_count", "maximum_restarts", 0, 2)
    bounded_parameter("restart_count", "window_seconds", 300, 3600)
    bounded_parameter(
        "stale_backup", "maximum_verified_backup_age_seconds", 3600, 86400
    )
    expected_chain = rules_by_check["gateway_wrong_chain"]["parameters"].get(
        "expected_chain_id"
    )
    if expected_chain != "zerone-2":
        fail("gateway wrong-chain monitoring does not pin zerone-2")
    bounded_parameter("gateway_stale_origin", "maximum_height_lag", 0, 10)

    tests = require_exact_object(
        objects[tests_name],
        {
            "schema",
            "chain_id",
            "source_commit",
            "ruleset_id",
            "rules_sha256",
            "started_at",
            "completed_at",
            "notification_route_id",
            "tests",
            "result",
        },
        "monitoring alert-test evidence",
    )
    if not (
        tests["schema"] == "zerone-production-monitoring-alert-tests-v2"
        and tests["chain_id"] == "zerone-2"
        and tests["source_commit"] == release["source"]["commit"]
        and tests["ruleset_id"] == rules["ruleset_id"]
        and tests["notification_route_id"] == rules["notification_route_id"]
        and tests["rules_sha256"] == rules_hash
        and tests["result"] == "PASS"
        and isinstance(tests["tests"], list)
        and len(tests["tests"]) == len(MONITORING_RULE_SPECS)
    ):
        fail("monitoring alert-test evidence is incomplete or mismatched")
    require_hash(tests["rules_sha256"], "monitoring alert-test rules hash")

    evidence_hashes: set[str] = set()
    tested_checks: set[str] = set()
    for index, candidate in enumerate(tests["tests"]):
        test = require_exact_object(
            candidate,
            {
                "check",
                "alert_name",
                "stimulus",
                "observed_states",
                "notification_delivery",
                "evidence",
                "result",
            },
            f"monitoring alert test {index}",
        )
        check = test["check"]
        if not isinstance(check, str) or check not in MONITORING_RULE_SPECS:
            fail(f"monitoring alert test {index} has an unknown check")
        if check in tested_checks:
            fail(f"monitoring alert tests contain duplicate check {check}")
        spec = MONITORING_RULE_SPECS[check]
        if not (
            test["alert_name"] == spec["alert_name"]
            and test["stimulus"] == spec["stimulus"]
            and test["observed_states"] == ["INACTIVE", "FIRING", "RESOLVED"]
            and test["notification_delivery"] == "DELIVERED"
            and test["result"] == "PASS"
        ):
            fail(f"monitoring alert test {check} did not prove firing and recovery")
        evidence = require_exact_object(
            test["evidence"],
            set(MONITORING_EVIDENCE_KINDS),
            f"monitoring alert test {check} evidence",
        )
        for kind in MONITORING_EVIDENCE_KINDS:
            reference = require_exact_object(
                evidence[kind],
                {"filename", "sha256"},
                f"monitoring alert test {check} {kind} evidence reference",
            )
            expected_filename = MONITORING_EVIDENCE_FILES[check][kind]
            if reference["filename"] != expected_filename:
                fail(
                    f"monitoring alert test {check} {kind} evidence must reference "
                    f"exact bundle file {expected_filename}"
                )
            proof_hash = require_hash(
                reference["sha256"],
                f"monitoring alert test {check} {kind} evidence",
            )
            proof_bytes = files[expected_filename]
            if not proof_bytes:
                fail(
                    f"monitoring alert test {check} {kind} evidence file is empty"
                )
            if proof_hash != sha256(proof_bytes):
                fail(
                    f"monitoring alert test {check} {kind} evidence hash differs "
                    f"from bundled {expected_filename}"
                )
            if proof_hash == "0" * 64 or proof_hash in evidence_hashes:
                fail("monitoring alert tests reuse or omit required evidence")
            evidence_hashes.add(proof_hash)
        tested_checks.add(check)
    if tested_checks != set(MONITORING_RULE_SPECS):
        fail("monitoring alert tests omit a required production check")

    tests_started_epoch = canonical_epoch(
        tests["started_at"], "monitoring alert-test start time"
    )
    tests_completed_epoch = canonical_epoch(
        tests["completed_at"], "monitoring alert-test completion time"
    )
    manifest_created_epoch = canonical_epoch(
        manifest["created_at"], "monitoring alert manifest creation time"
    )
    if not (
        tests_started_epoch
        <= tests_completed_epoch
        <= manifest_created_epoch
        <= release_created_epoch
    ):
        fail("monitoring evidence chronology is non-monotonic")


def load_frozen_evidence_validator(
    temp_path: pathlib.Path, verified_tools: dict[str, bytes]
) -> Any:
    relative = "deploy/frozen_evidence.py"
    helper_bytes = verified_tools.get(relative)
    if helper_bytes is None:
        fail("verified operator tools omit the frozen-evidence validator")
    helper_path = temp_path / "frozen_evidence.py"
    helper_path.write_bytes(helper_bytes)
    os.chmod(helper_path, 0o600)
    spec = importlib.util.spec_from_file_location(
        "zerone_release_frozen_evidence", helper_path
    )
    if spec is None or spec.loader is None:
        fail("could not create the frozen-evidence validator module")
    module = importlib.util.module_from_spec(spec)
    try:
        spec.loader.exec_module(module)
    except Exception as exc:
        fail(f"could not load the release-bound frozen-evidence validator: {exc}")
    if set(getattr(module, "REQUIRED_FROZEN_EVIDENCE_FILES", ())) != (
        FROZEN_EVIDENCE_FILES
    ):
        fail("frozen-evidence validator and authority bundle file set disagree")
    if not callable(getattr(module, "validate_frozen_evidence", None)):
        fail("frozen-evidence validator does not expose its required entrypoint")
    error_type = getattr(module, "FrozenEvidenceError", None)
    if not isinstance(error_type, type) or not issubclass(error_type, Exception):
        fail("frozen-evidence validator does not expose its required error type")
    return module


def validate_frozen_terminal_cryptography(
    paths: dict[str, pathlib.Path],
    objects: dict[str, Any],
    f: str,
    a: str,
    h: str,
    temp_path: pathlib.Path,
) -> None:
    release = objects["RELEASE-PACKET.json"]
    final = objects["FINAL-CHECKPOINT.json"]
    transition = objects["zerone-1-archive-transition.json"]
    inventory = objects["ZERONE-1-INVENTORY-V3.json"]
    trusted = release["predecessor"]["trusted_block"]
    binary = paths["zeroned-zerone-1-release"]
    os.chmod(binary, 0o700)
    environment = {
        "HOME": str(temp_path),
        "TMPDIR": str(temp_path),
        "PATH": os.defpath,
        "LC_ALL": "C",
        "LANG": "C",
    }
    for prefix in ("SIGNER", "OBSERVER"):
        command = [
            str(binary),
            "verify-frozen-terminal",
            "--genesis",
            str(paths[f"{prefix}-RPC-GENESIS.json.raw"]),
            "--trusted-block",
            str(paths[f"{prefix}-RPC-TRUSTED-BLOCK.json.raw"]),
            "--trusted-commit",
            str(paths[f"{prefix}-RPC-TRUSTED-COMMIT.json.raw"]),
            "--trusted-validators",
            str(paths[f"{prefix}-RPC-TRUSTED-VALIDATORS.json.raw"]),
            "--a-block",
            str(paths[f"{prefix}-RPC-BLOCK-A.json.raw"]),
            "--a-commit",
            str(paths[f"{prefix}-RPC-COMMIT-A.json.raw"]),
            "--a-validators",
            str(paths[f"{prefix}-RPC-VALIDATORS-A.json.raw"]),
            "--a-block-results",
            str(paths[f"{prefix}-RPC-BLOCK-RESULTS-A.json.raw"]),
            "--h-block",
            str(paths[f"{prefix}-RPC-BLOCK-H.json.raw"]),
            "--h-commit",
            str(paths[f"{prefix}-RPC-COMMIT-H.json.raw"]),
            "--h-validators",
            str(paths[f"{prefix}-RPC-VALIDATORS-H.json.raw"]),
            "--expected-chain-id",
            "zerone-1",
            "--trusted-height",
            trusted["height"],
            "--checkpoint-state-height",
            f,
            "--final-committed-height",
            a,
            "--halt-trigger-height",
            h,
            "--expected-trusted-block-hash",
            trusted["block_id_hash"],
            "--expected-trusted-app-hash",
            trusted["app_hash"],
            "--expected-checkpoint-app-hash",
            final["checkpoint_state"]["app_hash"],
            "--expected-anchor-block-hash",
            transition["expected_anchor_block_hash"],
            "--expected-halt-trigger-block-hash",
            final["halt_trigger_tip"]["staged_block_id_hash"],
            "--expected-post-anchor-app-hash",
            transition["expected_post_anchor_app_hash"],
            "--expected-rpc-genesis-sha256",
            inventory["source"]["rpc_genesis_canonical_sha256"],
        ]
        try:
            result = subprocess.run(
                command,
                cwd=temp_path,
                env=environment,
                stdin=subprocess.DEVNULL,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
                timeout=60,
            )
        except (OSError, subprocess.TimeoutExpired) as exc:
            fail(f"{prefix} release-binary terminal crypto verifier failed: {exc}")
        if result.returncode != 0:
            detail = (result.stderr or result.stdout)[:4096].decode(
                errors="replace"
            ).strip()
            fail(
                f"{prefix} release-binary terminal crypto verification failed"
                + (f": {detail}" if detail else "")
            )
        if result.stdout != b"frozen-terminal-crypto: MATCH\n" or result.stderr:
            fail(f"{prefix} terminal crypto verifier returned unexpected output")


def validate_release_ceremony(
    files: dict[str, bytes],
    objects: dict[str, Any],
    release: dict[str, Any],
    main: str,
) -> None:
    signature_authority = require_exact_object(
        release.get("signature_authority"),
        {"algorithm", "authorized_signer_fingerprint", "detached_signature_filename"},
        "RELEASE signature authority",
    )
    if not (
        signature_authority["algorithm"] == "openpgp"
        and signature_authority["detached_signature_filename"]
        == "RELEASE-PACKET.json.sig"
        and isinstance(signature_authority["authorized_signer_fingerprint"], str)
        and normalize_fingerprint(signature_authority["authorized_signer_fingerprint"])
        == normalize_fingerprint(main)
    ):
        fail("RELEASE signature authority is inconsistent")
    predecessor = require_exact_object(
        release.get("predecessor"),
        {
            "chain_id",
            "genesis_file_sha256",
            "trusted_rpc_node_id",
            "trusted_observer_node_id",
            "trusted_block",
        },
        "RELEASE predecessor",
    )
    if predecessor["chain_id"] != "zerone-1":
        fail("RELEASE predecessor chain ID changed")
    require_hash(predecessor["genesis_file_sha256"], "predecessor genesis")
    for field in ("trusted_rpc_node_id", "trusted_observer_node_id"):
        node_id = predecessor[field]
        if not isinstance(node_id, str) or not re.fullmatch(r"[0-9a-f]{40}", node_id):
            fail(f"RELEASE predecessor {field} is not a lowercase node ID")
        if node_id == "0" * 40:
            fail(f"RELEASE predecessor {field} is the zero node ID")
    if predecessor["trusted_rpc_node_id"] == predecessor["trusted_observer_node_id"]:
        fail("RELEASE predecessor signer and observer node IDs must differ")
    trusted_block = require_exact_object(
        predecessor["trusted_block"],
        {"height", "block_id_hash", "app_hash"},
        "RELEASE predecessor trusted block",
    )
    if not isinstance(trusted_block["height"], str) or not POSITIVE_HEIGHT.fullmatch(
        trusted_block["height"]
    ):
        fail("RELEASE predecessor trusted block height is malformed")
    require_hash(
        trusted_block["block_id_hash"], "predecessor trusted block ID", upper=True
    )
    require_hash(trusted_block["app_hash"], "predecessor trusted AppHash", upper=True)

    source = require_exact_object(
        release.get("source"),
        {"commit", "signed_annotated_tag", "tag_signer_fingerprint"},
        "RELEASE source",
    )
    if not isinstance(source["commit"], str) or not re.fullmatch(
        r"[0-9a-f]{40}", source["commit"]
    ):
        fail("RELEASE source commit is malformed")
    require_nonempty_string(source["signed_annotated_tag"], "RELEASE signed tag")
    signer = source["tag_signer_fingerprint"]
    if not isinstance(signer, str) or not FINGERPRINT.fullmatch(signer) or (
        normalize_fingerprint(signer) != normalize_fingerprint(main)
    ):
        fail("RELEASE tag signer differs from the authorized release signer")

    identities = require_exact_object(
        release.get("public_identities"),
        {
            "validator_account_address",
            "validator_operator_address",
            "operations_account_address",
            "validator_node_id",
            "edge_node_id",
            "validator_consensus_pubkey",
            "transition_attestation",
        },
        "RELEASE public identities",
    )
    validator_account = require_nonempty_string(
        identities["validator_account_address"], "validator account"
    )
    validator_operator = require_nonempty_string(
        identities["validator_operator_address"], "validator operator"
    )
    operations_account = require_nonempty_string(
        identities["operations_account_address"], "operations account"
    )
    if validator_account == operations_account:
        fail("RELEASE validator and operations accounts collide")
    if not validator_operator.startswith("zeronevaloper1"):
        fail("RELEASE validator operator address is not zeronevaloper")
    for field in ("validator_node_id", "edge_node_id"):
        node_id = identities[field]
        if not isinstance(node_id, str) or not re.fullmatch(r"[0-9a-f]{40}", node_id):
            fail(f"RELEASE {field} is not a lowercase node ID")
        if node_id == "0" * 40:
            fail(f"RELEASE {field} is the zero node ID")
    if identities["validator_node_id"] == identities["edge_node_id"]:
        fail("RELEASE successor validator and edge node IDs collide")
    consensus_key = decode_base64(
        identities["validator_consensus_pubkey"], "validator consensus key"
    )
    if len(consensus_key) != 32:
        fail("RELEASE validator consensus key is not 32 bytes")
    transition = require_exact_object(
        identities["transition_attestation"],
        {"algorithm", "authorized_signer_fingerprint", "purpose"},
        "RELEASE transition authority",
    )
    if not (
        transition["algorithm"] == "openpgp"
        and isinstance(transition["authorized_signer_fingerprint"], str)
        and FINGERPRINT.fullmatch(transition["authorized_signer_fingerprint"])
        and normalize_fingerprint(transition["authorized_signer_fingerprint"])
        != normalize_fingerprint(main)
        and transition["purpose"]
        == "post-halt evidence and deterministic private archive-adoption manifests only; not operator GO authority"
    ):
        fail("RELEASE transition authority is malformed or over-broad")

    genesis_declaration = require_exact_object(
        release.get("genesis"),
        {"time", "sha256", "independent_verification_count"},
        "RELEASE genesis",
    )
    canonical_epoch(genesis_declaration["time"], "RELEASE genesis time")
    genesis_hash = require_hash(genesis_declaration["sha256"], "RELEASE genesis")
    if genesis_declaration["independent_verification_count"] != 2:
        fail("RELEASE genesis must record exactly two independent verifications")
    if sha256(files["genesis.json"]) != genesis_hash:
        fail("bundled genesis bytes differ from RELEASE")
    if files["genesis.sha256"] != f"{genesis_hash}  genesis.json\n".encode():
        fail("bundled genesis checksum file is not exact")

    ceremony = require_exact_object(
        release.get("ceremony_artifacts"),
        {
            "schema",
            "genesis_checksum_sha256",
            "network_manifest_sha256",
            "human_manifest_sha256",
        },
        "RELEASE ceremony artifacts",
    )
    if ceremony["schema"] != "zerone-2-public-ceremony-artifact-set-v1":
        fail("RELEASE ceremony artifact schema changed")
    for field, filename in {
        "genesis_checksum_sha256": "genesis.sha256",
        "network_manifest_sha256": "network-manifest.json",
        "human_manifest_sha256": "GENESIS-MANIFEST.md",
    }.items():
        require_hash(ceremony[field], f"RELEASE ceremony {field}")
        if ceremony[field] != sha256(files[filename]):
            fail(f"RELEASE ceremony hash differs from bundled {filename}")

    manifest = require_exact_object(
        objects["network-manifest.json"],
        {
            "schema",
            "mode",
            "chain_id",
            "genesis_time",
            "genesis_sha256",
            "release",
            "trust_model",
            "supply_uzrn",
            "validator",
            "operations",
            "activations",
        },
        "ceremony network manifest",
    )
    manifest_release = require_exact_object(
        manifest["release"],
        {
            "source_commit",
            "tag",
            "tag_signer_fingerprint",
            "binary_sha256",
            "binary_version",
            "binary_goos",
            "binary_goarch",
        },
        "ceremony manifest release",
    )
    manifest_trust = require_exact_object(
        manifest["trust_model"],
        {"genesis_validators", "byzantine_fault_tolerance", "disclosure"},
        "ceremony manifest trust model",
    )
    manifest_validator = require_exact_object(
        manifest["validator"],
        {
            "account_address",
            "operator_address",
            "consensus_pubkey",
            "node_id",
            "self_bond_uzrn",
            "gentx_sha256",
        },
        "ceremony manifest validator",
    )
    manifest_key = require_exact_object(
        manifest_validator["consensus_pubkey"],
        {"@type", "key"},
        "ceremony manifest consensus key",
    )
    manifest_operations = require_exact_object(
        manifest["operations"], {"account_address"}, "ceremony manifest operations"
    )
    manifest_activations = require_exact_object(
        manifest["activations"],
        {"vote_extensions", "pot", "ibc", "substrate_bridge", "claiming"},
        "ceremony manifest activations",
    )
    if not (
        manifest["schema"] == "zerone-2-network-manifest-v2"
        and manifest["mode"] == "real"
        and manifest["chain_id"] == "zerone-2"
        and manifest["genesis_time"] == genesis_declaration["time"]
        and manifest["genesis_sha256"] == genesis_hash
        and manifest_release["source_commit"] == source["commit"]
        and manifest_release["tag"] == source["signed_annotated_tag"]
        and normalize_fingerprint(manifest_release["tag_signer_fingerprint"])
        == normalize_fingerprint(main)
        and manifest_release["binary_sha256"]
        == release["components"]["zerone_2_runtime"]["binary_sha256"]
        and isinstance(manifest_release["binary_version"], str)
        and bool(manifest_release["binary_version"])
        and manifest_release["binary_goos"] == "linux"
        and manifest_release["binary_goarch"] in {"amd64", "arm64"}
        and manifest_trust["genesis_validators"] == 1
        and manifest_trust["byzantine_fault_tolerance"] == 0
        and isinstance(manifest_trust["disclosure"], str)
        and bool(manifest_trust["disclosure"].strip())
        and manifest["supply_uzrn"] == "13555000000"
        and manifest_validator["account_address"] == validator_account
        and manifest_validator["operator_address"] == validator_operator
        and manifest_key["@type"] == "/cosmos.crypto.ed25519.PubKey"
        and manifest_key["key"] == identities["validator_consensus_pubkey"]
        and manifest_validator["node_id"] == identities["validator_node_id"]
        and manifest_validator["self_bond_uzrn"] == "11111000000"
        and isinstance(manifest_validator["gentx_sha256"], str)
        and HASH.fullmatch(manifest_validator["gentx_sha256"])
        and manifest_operations["account_address"] == operations_account
        and manifest_activations
        == {
            "vote_extensions": "disabled",
            "pot": "not live",
            "ibc": "external-disabled; localhost-only",
            "substrate_bridge": "disabled",
            "claiming": "disabled",
        }
    ):
        fail("ceremony network manifest differs from RELEASE or the production profile")

    genesis = objects["genesis.json"]
    if not isinstance(genesis, dict):
        fail("bundled genesis is not a JSON object")
    if genesis.get("chain_id") != "zerone-2" or (
        genesis.get("genesis_time") != genesis_declaration["time"]
    ):
        fail("bundled genesis chain/time differs from RELEASE")
    app_state = genesis.get("app_state")
    bank = app_state.get("bank") if isinstance(app_state, dict) else None
    if not isinstance(bank, dict) or bank.get("supply") != [
        {"denom": "uzrn", "amount": "13555000000"}
    ]:
        fail("bundled genesis supply differs from the signed launch profile")
    balances = bank.get("balances")
    if not isinstance(balances, list) or len(balances) != 2:
        fail("bundled genesis must contain exactly two bank balances")
    actual_balances: dict[str, Any] = {}
    for index, balance in enumerate(balances):
        if not isinstance(balance, dict) or set(balance) != {"address", "coins"}:
            fail(f"bundled genesis balance {index} is malformed")
        if balance["address"] in actual_balances:
            fail("bundled genesis contains a duplicate bank balance owner")
        actual_balances[balance["address"]] = balance["coins"]
    if actual_balances != {
        validator_account: [{"denom": "uzrn", "amount": "11333000000"}],
        operations_account: [{"denom": "uzrn", "amount": "2222000000"}],
    }:
        fail("bundled genesis bank allocations differ from RELEASE identities")
    genutil = app_state.get("genutil") if isinstance(app_state, dict) else None
    gentxs = genutil.get("gen_txs") if isinstance(genutil, dict) else None
    if not isinstance(gentxs, list) or len(gentxs) != 1 or not isinstance(gentxs[0], dict):
        fail("bundled genesis must contain exactly one gentx")
    gentx = gentxs[0]
    body = gentx.get("body")
    messages = body.get("messages") if isinstance(body, dict) else None
    if not isinstance(messages, list) or len(messages) != 1 or not isinstance(messages[0], dict):
        fail("bundled genesis gentx message set is malformed")
    message = messages[0]
    memo = body.get("memo")
    if not (
        message.get("@type") == "/cosmos.staking.v1beta1.MsgCreateValidator"
        and message.get("validator_address") == validator_operator
        and message.get("pubkey") == manifest_validator["consensus_pubkey"]
        and message.get("value") == {"denom": "uzrn", "amount": "11111000000"}
        and isinstance(memo, str)
        and memo.split("@", 1)[0] == identities["validator_node_id"]
    ):
        fail("bundled genesis gentx differs from RELEASE identities/self-bond")
    gentx_bytes = json.dumps(
        gentx, sort_keys=True, separators=(",", ":"), ensure_ascii=False
    ).encode()
    if sha256(gentx_bytes) != manifest_validator["gentx_sha256"]:
        fail("ceremony manifest gentx hash differs from bundled genesis")

    try:
        human_lines = files["GENESIS-MANIFEST.md"].decode("utf-8").splitlines()
    except UnicodeDecodeError as exc:
        fail(f"ceremony human manifest is not UTF-8: {exc}")
    required_human_lines = (
        "- Ceremony mode: **real**",
        f"- Genesis time: {genesis_declaration['time']}",
        f"- Genesis SHA-256: {genesis_hash}",
        f"- Source commit: {source['commit']}",
        f"- Release tag: {source['signed_annotated_tag']}",
        f"- Release tag signer fingerprint: {manifest_release['tag_signer_fingerprint']}",
        f"- Binary SHA-256: {manifest_release['binary_sha256']}",
        f"- Binary version: {manifest_release['binary_version']}",
        f"- Binary target: {manifest_release['binary_goos']}/{manifest_release['binary_goarch']}",
    )
    if any(line not in human_lines for line in required_human_lines):
        fail("ceremony human manifest does not repeat the exact signed release facts")


def verify_component_signature(
    paths: dict[str, pathlib.Path],
    tool_manifest: dict[str, Any],
    component: str,
    bundle_filename: str,
    artifact_digest: str,
    source_repository_digest: str,
) -> set[str]:
    verifier_path = paths[COMPONENT_SIGNATURE_VERIFIER_FILE]
    trusted_root_path = paths[SIGSTORE_TRUSTED_ROOT_FILE]
    bundle_path = paths[bundle_filename]
    policy = tool_manifest["component_signature_policy"]
    os.chmod(verifier_path, 0o700)
    command = [
        str(verifier_path),
        "--bundle",
        str(bundle_path),
        "--trusted-root",
        str(trusted_root_path),
        "--certificate-issuer",
        policy["certificate_issuer"],
        "--certificate-san",
        policy["certificate_san"],
        "--source-repository-digest",
        source_repository_digest,
        "--artifact-digest",
        artifact_digest,
    ]
    environment = {
        "HOME": str(verifier_path.parent),
        "TMPDIR": str(verifier_path.parent),
        "LANG": "C",
        "LC_ALL": "C",
        "PATH": "/usr/bin:/bin",
    }
    try:
        result = subprocess.run(
            command,
            cwd=verifier_path.parent,
            env=environment,
            stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=30,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        fail(f"{component} component-signature verifier failed: {exc}")
    if len(result.stdout) > 64 * 1024 or len(result.stderr) > 64 * 1024:
        fail(f"{component} component-signature verifier output exceeded its limit")
    if result.returncode != 0:
        detail = (result.stderr or result.stdout).decode(errors="replace").strip()
        fail(
            f"{component} Sigstore cryptographic verification failed"
            + (f": {detail}" if detail else "")
        )
    if result.stderr:
        fail(f"{component} component-signature verifier wrote unexpected stderr")
    verification = parse_json(
        result.stdout, f"{component} component-signature verifier output"
    )
    verification = require_exact_object(
        verification,
        {
            "schema",
            "result",
            "artifact_digest",
            "bundle_media_type",
            "certificate_issuer",
            "certificate_san",
            "source_repository_digest",
            "verified_timestamps",
        },
        f"{component} component-signature verifier output",
    )
    if not (
        verification["schema"] == "zerone.component-signature-verification/v1"
        and verification["result"] == "verified"
        and verification["artifact_digest"] == artifact_digest
        and verification["bundle_media_type"] == policy["bundle_media_type"]
        and verification["certificate_issuer"] == policy["certificate_issuer"]
        and verification["certificate_san"] == policy["certificate_san"]
        and verification["source_repository_digest"] == source_repository_digest
        and isinstance(verification["verified_timestamps"], list)
        and len(verification["verified_timestamps"])
        >= policy["minimum_observer_timestamps"]
    ):
        fail(f"{component} component-signature verifier result is inconsistent")
    timestamps: set[str] = set()
    observers: set[tuple[str, str, str]] = set()
    for index, value in enumerate(verification["verified_timestamps"]):
        timestamp = require_exact_object(
            value,
            {"type", "uri", "timestamp"},
            f"{component} verified timestamp {index}",
        )
        if timestamp["type"] not in {"Tlog", "TimestampAuthority"}:
            fail(f"{component} verified timestamp {index} has an unknown observer type")
        uri = require_nonempty_string(
            timestamp["uri"], f"{component} verified timestamp {index} URI"
        )
        if not uri.startswith("https://"):
            fail(f"{component} verified timestamp {index} URI is not HTTPS")
        canonical_nanoseconds(
            timestamp["timestamp"], f"{component} verified timestamp {index}"
        )
        observer = (timestamp["type"], uri, timestamp["timestamp"])
        if observer in observers:
            fail(f"{component} component-signature verifier repeated an observer")
        observers.add(observer)
        timestamps.add(timestamp["timestamp"])
    return timestamps


def validate_release_components(
    files: dict[str, bytes],
    paths: dict[str, pathlib.Path],
    objects: dict[str, Any],
    release: dict[str, Any],
    main: str,
    release_created_epoch: int,
) -> dict[str, str]:
    components = require_exact_object(
        release.get("components"),
        {"zerone_1_halt", "zerone_2_runtime", "query_gateway"},
        "RELEASE components",
    )
    binary_components = {"zerone_1_halt", "zerone_2_runtime"}
    base_fields = {
        "image_ref",
        "sbom_sha256",
        "provenance_sha256",
        "signature_sha256",
        "vulnerability_decision_sha256",
    }
    source = release["source"]
    tool_manifest = objects["OPERATOR-TOOL-MANIFEST.json"]
    images: dict[str, str] = {}
    image_digests: set[str] = set()
    for component, names in COMPONENT_ARTIFACT_FILES.items():
        fields = base_fields | ({"binary_sha256"} if component in binary_components else set())
        declaration = require_exact_object(
            components[component], fields, f"RELEASE {component}"
        )
        image_ref = declaration["image_ref"]
        if not isinstance(image_ref, str) or not IMAGE_REF.fullmatch(image_ref):
            fail(f"RELEASE {component} image is not digest pinned")
        images[component] = image_ref
        image_digest = image_ref.rsplit("@sha256:", 1)[1]
        if image_digest in image_digests:
            fail("RELEASE component image digests must be pairwise distinct")
        image_digests.add(image_digest)
        for field, artifact_type in (
            ("sbom_sha256", "sbom"),
            ("provenance_sha256", "provenance"),
            ("signature_sha256", "signature"),
            ("vulnerability_decision_sha256", "vulnerability_decision"),
        ):
            expected = require_hash(declaration[field], f"RELEASE {component} {field}")
            filename = names[artifact_type]
            if expected != sha256(files[filename]):
                fail(f"RELEASE {component} {field} differs from bundled {filename}")

        binary_hash: str | None = None
        if component in binary_components:
            binary_hash = require_hash(
                declaration["binary_sha256"], f"RELEASE {component} binary"
            )
            filename = (
                "zeroned-zerone-1-release"
                if component == "zerone_1_halt"
                else "zeroned-zerone-2-release"
            )
            if binary_hash != sha256(files[filename]):
                fail(f"bundled {component} release binary differs from RELEASE")

        sbom = require_exact_object(
            objects[names["sbom"]],
            {
                "schema",
                "component",
                "image_ref",
                "source_commit",
                "generated_at",
                "packages",
                "result",
            },
            f"{component} SBOM",
        )
        if not (
            sbom["schema"] == "zerone-component-sbom-v1"
            and sbom["component"] == component
            and sbom["image_ref"] == image_ref
            and sbom["source_commit"] == source["commit"]
            and sbom["result"] == "COMPLETE"
            and isinstance(sbom["packages"], list)
            and bool(sbom["packages"])
        ):
            fail(f"{component} SBOM does not bind the exact release subject")
        purls: set[str] = set()
        for index, package in enumerate(sbom["packages"]):
            item = require_exact_object(
                package, {"name", "version", "purl"}, f"{component} SBOM package {index}"
            )
            require_nonempty_string(item["name"], f"{component} package name")
            require_nonempty_string(item["version"], f"{component} package version")
            purl = require_nonempty_string(item["purl"], f"{component} package purl")
            if not purl.startswith("pkg:") or purl in purls:
                fail(f"{component} SBOM contains a malformed or duplicate purl")
            purls.add(purl)
        sbom_epoch = canonical_epoch(sbom["generated_at"], f"{component} SBOM time")
        if sbom_epoch > release_created_epoch:
            fail(f"{component} SBOM was generated after RELEASE creation")

        provenance = require_exact_object(
            objects[names["provenance"]],
            {"schema", "component", "subject", "source", "build", "result"},
            f"{component} provenance",
        )
        subject = require_exact_object(
            provenance["subject"],
            {"image_ref", "image_digest", "binary_sha256"},
            f"{component} provenance subject",
        )
        provenance_source = require_exact_object(
            provenance["source"],
            {"commit", "signed_annotated_tag", "tag_signer_fingerprint"},
            f"{component} provenance source",
        )
        build = require_exact_object(
            provenance["build"],
            {
                "builder_id",
                "build_recipe_path",
                "build_recipe_sha256",
                "started_at",
                "finished_at",
                "source_materials_complete",
            },
            f"{component} provenance build",
        )
        recipe = COMPONENT_BUILD_RECIPES[component]
        if not (
            provenance["schema"] == "zerone-component-provenance-v1"
            and provenance["component"] == component
            and provenance["result"] == "MATCH"
            and subject["image_ref"] == image_ref
            and subject["image_digest"] == f"sha256:{image_digest}"
            and subject["binary_sha256"] == binary_hash
            and provenance_source == source
            and isinstance(build["builder_id"], str)
            and build["builder_id"].startswith("https://")
            and build["build_recipe_path"] == recipe
            and build["build_recipe_sha256"] == tool_manifest["files"].get(recipe)
            and build["source_materials_complete"] is True
        ):
            fail(f"{component} provenance does not bind source/build/image/binary")
        require_hash(build["build_recipe_sha256"], f"{component} build recipe")
        build_start = canonical_epoch(build["started_at"], f"{component} build start")
        build_finish = canonical_epoch(build["finished_at"], f"{component} build finish")
        if not build_start <= build_finish <= release_created_epoch:
            fail(f"{component} provenance chronology is non-monotonic")

        signature = require_exact_object(
            objects[names["signature"]],
            {
                "schema",
                "component",
                "image_ref",
                "image_digest",
                "bundle_sha256",
                "signer_identity",
                "certificate_issuer",
                "signed_at",
                "verified_at",
                "transparency_log_verified",
                "result",
            },
            f"{component} image signature evidence",
        )
        if not (
            signature["schema"] == "zerone-component-signature-evidence-v1"
            and signature["component"] == component
            and signature["image_ref"] == image_ref
            and signature["image_digest"] == f"sha256:{image_digest}"
            and signature["bundle_sha256"] == sha256(files[names["signature_bundle"]])
            and signature["signer_identity"] == CANONICAL_COMPONENT_SIGNER_IDENTITY
            and signature["certificate_issuer"]
            == CANONICAL_COMPONENT_CERTIFICATE_ISSUER
            and signature["transparency_log_verified"] is True
            and signature["result"] == "VERIFIED"
        ):
            fail(f"{component} image signature evidence is incomplete or mismatched")
        require_hash(signature["bundle_sha256"], f"{component} signature bundle")
        bundle = require_exact_object(
            objects[names["signature_bundle"]],
            {"mediaType", "verificationMaterial", "messageSignature"},
            f"{component} Sigstore bundle",
        )
        message_signature = require_exact_object(
            bundle["messageSignature"],
            {"messageDigest", "signature"},
            f"{component} Sigstore message signature",
        )
        message_digest = require_exact_object(
            message_signature["messageDigest"],
            {"algorithm", "digest"},
            f"{component} Sigstore message digest",
        )
        signed_digest = decode_base64(
            message_digest["digest"], f"{component} signed digest"
        )
        raw_signature = decode_base64(
            message_signature["signature"], f"{component} image signature"
        )
        if not (
            bundle["mediaType"] == COMPONENT_SIGNATURE_BUNDLE_MEDIA_TYPE
            and isinstance(bundle["verificationMaterial"], dict)
            and bool(bundle["verificationMaterial"])
            and message_digest["algorithm"] == "SHA2_256"
            and signed_digest == bytes.fromhex(image_digest)
            and len(raw_signature) >= 32
        ):
            fail(f"{component} Sigstore bundle does not sign the image digest")
        signed_ns = canonical_nanoseconds(
            signature["signed_at"], f"{component} signing time"
        )
        verified_signing_times = verify_component_signature(
            paths,
            tool_manifest,
            component,
            names["signature_bundle"],
            f"sha256:{image_digest}",
            source["commit"],
        )
        if signature["signed_at"] not in verified_signing_times:
            fail(
                f"{component} signed_at is not an authenticated observer time"
            )
        verified_epoch = canonical_epoch(
            signature["verified_at"], f"{component} verification time"
        )
        if not (
            build_finish * 1_000_000_000
            <= signed_ns
            <= verified_epoch * 1_000_000_000
            <= release_created_epoch * 1_000_000_000
        ):
            fail(f"{component} image-signature chronology is non-monotonic")

        scan = require_exact_object(
            objects[names["vulnerability_scan"]],
            {
                "schema",
                "component",
                "image_ref",
                "scanner",
                "scanned_at",
                "counts",
                "result",
            },
            f"{component} vulnerability scan",
        )
        scanner = require_exact_object(
            scan["scanner"],
            {"name", "version", "database_updated_at"},
            f"{component} vulnerability scanner",
        )
        counts = require_exact_object(
            scan["counts"],
            {"critical", "high", "medium", "low", "unknown"},
            f"{component} vulnerability counts",
        )
        for severity, count in counts.items():
            require_nonnegative_integer(count, f"{component} {severity} count")
        if not (
            scan["schema"] == "zerone-component-vulnerability-scan-v1"
            and scan["component"] == component
            and scan["image_ref"] == image_ref
            and scan["result"] == "COMPLETE"
            and counts["critical"] == 0
            and counts["high"] == 0
        ):
            fail(f"{component} vulnerability scan is not a clean complete result")
        require_nonempty_string(scanner["name"], f"{component} scanner name")
        require_nonempty_string(scanner["version"], f"{component} scanner version")
        database_epoch = canonical_epoch(
            scanner["database_updated_at"], f"{component} scanner database time"
        )
        scanned_epoch = canonical_epoch(scan["scanned_at"], f"{component} scan time")
        if not database_epoch <= scanned_epoch <= release_created_epoch:
            fail(f"{component} vulnerability scan chronology is non-monotonic")

        decision = require_exact_object(
            objects[names["vulnerability_decision"]],
            {
                "schema",
                "component",
                "image_ref",
                "scan_report_sha256",
                "scan_completed_at",
                "counts",
                "policy",
                "decision",
                "approved_by_fingerprint",
                "created_at",
                "expires_at",
            },
            f"{component} vulnerability decision",
        )
        policy = require_exact_object(
            decision["policy"],
            {"maximum_critical", "maximum_high"},
            f"{component} vulnerability policy",
        )
        if not (
            decision["schema"] == "zerone-component-vulnerability-decision-v1"
            and decision["component"] == component
            and decision["image_ref"] == image_ref
            and decision["scan_report_sha256"] == sha256(files[names["vulnerability_scan"]])
            and decision["scan_completed_at"] == scan["scanned_at"]
            and decision["counts"] == counts
            and policy == {"maximum_critical": 0, "maximum_high": 0}
            and decision["decision"] == "ACCEPT"
            and isinstance(decision["approved_by_fingerprint"], str)
            and normalize_fingerprint(decision["approved_by_fingerprint"])
            == normalize_fingerprint(main)
        ):
            fail(f"{component} vulnerability decision is incomplete or mismatched")
        require_hash(decision["scan_report_sha256"], f"{component} scan report")
        decision_epoch = canonical_epoch(
            decision["created_at"], f"{component} vulnerability decision time"
        )
        expiry_epoch = canonical_epoch(
            decision["expires_at"], f"{component} vulnerability decision expiry"
        )
        now = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
        if not scanned_epoch <= decision_epoch <= release_created_epoch < expiry_epoch:
            fail(f"{component} vulnerability decision chronology is non-monotonic")
        if now > expiry_epoch:
            fail(f"{component} vulnerability decision has expired")

    if len(set(images.values())) != len(images):
        fail("RELEASE component images collide")
    return images


def contains_placeholder(value: Any) -> bool:
    if isinstance(value, dict):
        return any(contains_placeholder(item) for item in value.values())
    if isinstance(value, list):
        return any(contains_placeholder(item) for item in value)
    return isinstance(value, str) and ("REPLACE_" in value or "replace-" in value)


def same_shape_and_static(candidate: Any, template: Any, label: str) -> None:
    if isinstance(template, dict):
        if not isinstance(candidate, dict) or set(candidate) != set(template):
            fail(f"{label} keys differ from the canonical template")
        for key in template:
            same_shape_and_static(candidate[key], template[key], f"{label}.{key}")
        return
    if isinstance(template, list):
        if not isinstance(candidate, list) or len(candidate) != len(template):
            fail(f"{label} list shape differs from the canonical template")
        for index, item in enumerate(template):
            same_shape_and_static(candidate[index], item, f"{label}[{index}]")
        return
    if isinstance(template, bool):
        if candidate is not template:
            fail(f"{label} boolean policy changed")
        return
    if isinstance(template, int):
        if not isinstance(candidate, int) or isinstance(candidate, bool):
            fail(f"{label} type differs from the canonical template")
        if candidate != template:
            fail(f"{label} numeric policy changed")
        return
    if not isinstance(candidate, type(template)):
        fail(f"{label} type differs from the canonical template")
    if isinstance(template, str) and not (
        "REPLACE_" in template or "replace-" in template
    ) and candidate != template:
        fail(f"{label} fixed semantic value changed")


def verify_canonical(path: pathlib.Path, expected: bytes, label: str) -> None:
    try:
        result = subprocess.run(
            ["jq", "-S", "-c", ".", str(path)],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        fail(f"could not canonicalize {label}: {exc}")
    if result.stdout != expected:
        fail(f"{label} is not canonical JSON")


def normalize_fingerprint(value: str) -> str:
    return value.upper()


def verify_signature(
    paths: dict[str, pathlib.Path],
    objects: dict[str, Any],
    payload: str,
    signature: str,
    authority_path: tuple[str, ...],
    expected: str,
) -> int:
    authority = objects[payload]
    for key in authority_path:
        if not isinstance(authority, dict) or key not in authority:
            fail(f"{payload} signature authority is missing")
        authority = authority[key]
    if not isinstance(authority, dict):
        fail(f"{payload} signature authority is malformed")
    if authority.get("algorithm") != "openpgp":
        fail(f"{payload} signature algorithm is not OpenPGP")
    signer = authority.get("authorized_signer_fingerprint")
    if not isinstance(signer, str) or not FINGERPRINT.fullmatch(signer):
        fail(f"{payload} signer fingerprint is malformed")
    if normalize_fingerprint(signer) != normalize_fingerprint(expected):
        fail(f"{payload} repeats the wrong signer fingerprint")
    if authority.get("detached_signature_filename") != signature:
        fail(f"{payload} detached signature filename differs from its declaration")
    try:
        result = subprocess.run(
            [
                "gpg",
                "--batch",
                "--status-fd=1",
                "--verify",
                str(paths[signature]),
                str(paths[payload]),
            ],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        fail(f"{payload} detached signature verification failed: {exc}")
    valid_lines = [
        line.split()
        for line in result.stdout.splitlines()
        if line.startswith("[GNUPG:] VALIDSIG ") and len(line.split()) >= 3
    ]
    if len(valid_lines) != 1:
        fail(f"{payload} signature must produce exactly one VALIDSIG")
    if normalize_fingerprint(valid_lines[0][2]) != normalize_fingerprint(expected):
        fail(f"{payload} was signed by a different key")
    if len(valid_lines[0]) < 5 or not valid_lines[0][4].isdecimal():
        fail(f"{payload} VALIDSIG lacks a numeric signature timestamp")
    signature_epoch = int(valid_lines[0][4])
    now = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
    if signature_epoch <= 0 or signature_epoch > now + 300:
        fail(f"{payload} signature timestamp is zero or in the future")
    return signature_epoch


def require_main_signature(
    paths: dict[str, pathlib.Path],
    objects: dict[str, Any],
    payload: str,
    signature: str,
    main: str,
) -> int:
    return verify_signature(
        paths,
        objects,
        payload,
        signature,
        ("signature_authority",),
        main,
    )


def require_safe_absolute_posix_path(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value or "\x00" in value:
        fail(f"{label} is not a non-empty POSIX path")
    path = pathlib.PurePosixPath(value)
    if (
        not path.is_absolute()
        or value != str(path)
        or value == "/"
        or any(part in {"", ".", ".."} for part in path.parts[1:])
    ):
        fail(f"{label} is not a canonical bounded absolute POSIX path")
    if len(value.encode("utf-8")) > 4096:
        fail(f"{label} exceeds the path byte ceiling")
    return value


def validate_census_execution_evidence(
    files: dict[str, bytes],
    paths: dict[str, pathlib.Path],
    objects: dict[str, Any],
    release: dict[str, Any],
    final: dict[str, Any],
    transition: str,
    f: str,
    a: str,
    cutover_evidence_signature_epoch: int,
    final_created_epoch: int,
    final_signature_epoch: int,
) -> None:
    contract = validate_census_execution_contract(files, release)
    signature_epoch = verify_signature(
        paths,
        objects,
        CENSUS_EXECUTION_EVIDENCE_FILENAME,
        CENSUS_EXECUTION_SIGNATURE_FILENAME,
        ("signature_authority",),
        transition,
    )
    evidence = require_exact_object(
        objects[CENSUS_EXECUTION_EVIDENCE_FILENAME],
        {
            "schema",
            "result",
            "created_at",
            "release_packet",
            "cutover_initiation_evidence",
            "runner",
            "state",
            "binary",
            "source_snapshot",
            "command",
            "execution",
            "scan_guarantees",
            "signature_authority",
        },
        "custom-staking census execution evidence",
    )
    if not (
        evidence["schema"]
        == "zerone-custom-staking-census-execution-evidence-v1"
        and evidence["result"] == "PASS"
        and evidence["release_packet"]
        == exact_pair(files, "RELEASE-PACKET.json", "RELEASE-PACKET.json.sig")
        and evidence["cutover_initiation_evidence"]
        == exact_pair(
            files,
            "CUTOVER-INITIATION-EVIDENCE.json",
            "CUTOVER-INITIATION-EVIDENCE.json.sig",
        )
    ):
        fail("custom-staking census execution authority chain is mismatched")

    runner = require_exact_object(
        evidence["runner"],
        {"path", "sha256"},
        "custom-staking census execution runner",
    )
    operator_manifest = objects.get("OPERATOR-TOOL-MANIFEST.json")
    operator_files = (
        operator_manifest.get("files")
        if isinstance(operator_manifest, dict)
        else None
    )
    if not (
        runner["path"] == CENSUS_EXECUTION_RUNNER_PATH
        and isinstance(operator_files, dict)
        and runner["sha256"] == operator_files.get(CENSUS_EXECUTION_RUNNER_PATH)
    ):
        fail("custom-staking census execution runner differs from RELEASE")
    require_hash(runner["sha256"], "custom-staking census execution runner")

    authority = require_exact_object(
        evidence["signature_authority"],
        {
            "algorithm",
            "authorized_signer_fingerprint",
            "detached_signature_filename",
            "authority_limit",
        },
        "custom-staking census execution signature authority",
    )
    if authority != {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": contract["execution_evidence"][
            "authorized_signer_fingerprint"
        ],
        "detached_signature_filename": CENSUS_EXECUTION_SIGNATURE_FILENAME,
        "authority_limit": CENSUS_EXECUTION_AUTHORITY_LIMIT,
    }:
        fail("custom-staking census execution signature authority is over-broad")

    state = require_exact_object(
        evidence["state"],
        {"chain_id", "height", "app_hash", "source_commit"},
        "custom-staking census execution state",
    )
    census = objects["CUSTOM-STAKING-CENSUS.json"]
    census_report_hash = census.get("report_sha256") if isinstance(census, dict) else None
    expected_app_hash = final.get("excluded_post_anchor_state", {}).get("app_hash")
    if not isinstance(expected_app_hash, str):
        fail("FINAL excluded post-anchor AppHash is missing for census execution")
    if not (
        state
        == {
            "chain_id": "zerone-1",
            "height": a,
            "app_hash": expected_app_hash.lower(),
            "source_commit": release["source"]["commit"],
        }
        and isinstance(census, dict)
        and census.get("result") == "PASS"
        and census.get("evidence") == state
    ):
        fail("custom-staking census execution state differs from its report/A/E")

    binary = require_exact_object(
        evidence["binary"],
        {"filename", "sha256"},
        "custom-staking census execution binary",
    )
    if binary != contract["binary"]:
        fail("custom-staking census execution binary differs from RELEASE")

    snapshot = require_exact_object(
        evidence["source_snapshot"],
        {
            "manifest_filename",
            "manifest_sha256",
            "database_snapshot_sha256",
            "file_manifest_sha256",
        },
        "custom-staking census execution source snapshot",
    )
    snapshot_manifest = objects["OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json"]
    if not (
        snapshot["manifest_filename"]
        == "OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json"
        and snapshot["manifest_sha256"]
        == sha256(files["OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json"])
        and snapshot["database_snapshot_sha256"]
        == snapshot_manifest.get("database_snapshot_sha256")
        and snapshot["file_manifest_sha256"]
        == snapshot_manifest.get("file_manifest_sha256")
    ):
        fail("custom-staking census execution snapshot is not the frozen observer copy")
    for field in ("manifest_sha256", "database_snapshot_sha256", "file_manifest_sha256"):
        require_hash(snapshot[field], f"custom-staking census execution {field}")

    command = require_exact_object(
        evidence["command"],
        {
            "argv",
            "binary_path",
            "home_path",
            "backend",
            "copied_db",
            "report_transport",
            "output_path",
        },
        "custom-staking census execution command",
    )
    binary_path = require_safe_absolute_posix_path(
        command["binary_path"], "custom-staking census binary path"
    )
    home_path = require_safe_absolute_posix_path(
        command["home_path"], "custom-staking census copied home path"
    )
    output_path = require_safe_absolute_posix_path(
        command["output_path"], "custom-staking census output path"
    )
    binary_parts = pathlib.PurePosixPath(binary_path).parts
    home_parts = pathlib.PurePosixPath(home_path).parts
    output_parts = pathlib.PurePosixPath(output_path).parts
    if not (
        pathlib.PurePosixPath(binary_path).name == CENSUS_BINARY_FILENAME
        and pathlib.PurePosixPath(output_path).name == "CUSTOM-STAKING-CENSUS.json"
        and binary_parts != home_parts
        and output_parts[: len(home_parts)] != home_parts
        and binary_parts[: len(home_parts)] != home_parts
        and command["backend"] in {"goleveldb", "pebbledb"}
        and command["copied_db"] is True
        and command["report_transport"] == CENSUS_REPORT_TRANSPORT
    ):
        fail("custom-staking census execution paths/backend are unsafe")
    expected_argv = [
        binary_path,
        "--home",
        home_path,
        "--backend",
        command["backend"],
        "--chain-id",
        "zerone-1",
        "--expected-height",
        a,
        "--expected-app-hash",
        expected_app_hash.lower(),
        "--source-commit",
        release["source"]["commit"],
        "--copied-db",
    ]
    if command["argv"] != expected_argv:
        fail("custom-staking census execution argv is not the exact safe command")

    execution = require_exact_object(
        evidence["execution"],
        {
            "started_at",
            "completed_at",
            "exit_code",
            "stdout_sha256",
            "stderr_sha256",
            "report_filename",
            "report_sha256",
            "report_self_hash",
            "report_result",
        },
        "custom-staking census execution result",
    )
    empty_hash = sha256(b"")
    if not (
        execution["exit_code"] == 0
        and not isinstance(execution["exit_code"], bool)
        and execution["stdout_sha256"]
        == sha256(files["CUSTOM-STAKING-CENSUS.json"])
        and execution["stderr_sha256"] == empty_hash
        and execution["report_filename"] == "CUSTOM-STAKING-CENSUS.json"
        and execution["report_sha256"] == sha256(files["CUSTOM-STAKING-CENSUS.json"])
        and execution["report_self_hash"] == census_report_hash
        and execution["report_result"] == census.get("result") == "PASS"
    ):
        fail("custom-staking census execution report differs from the bound bytes")
    for field in ("stdout_sha256", "stderr_sha256", "report_sha256", "report_self_hash"):
        require_hash(execution[field], f"custom-staking census execution {field}")

    guarantees = require_exact_object(
        evidence["scan_guarantees"],
        {
            "required_stores",
            "complete_logical_store_iteration",
            "root_bound_leaf_count",
            "ics23_membership_proof_per_leaf",
            "root_commit_info_rechecked_after_scan",
            "database_backend_read_only",
            "write_attempts",
        },
        "custom-staking census execution scan guarantees",
    )
    if guarantees != {
        "required_stores": ["zerone_staking", "bank", "staking"],
        "complete_logical_store_iteration": True,
        "root_bound_leaf_count": True,
        "ics23_membership_proof_per_leaf": True,
        "root_commit_info_rechecked_after_scan": True,
        "database_backend_read_only": True,
        "write_attempts": 0,
    }:
        fail("custom-staking census execution did not attest the full proof scan")

    started_epoch = canonical_epoch(
        execution["started_at"], "custom-staking census execution start"
    )
    completed_epoch = canonical_epoch(
        execution["completed_at"], "custom-staking census execution completion"
    )
    created_epoch = canonical_epoch(
        evidence["created_at"], "custom-staking census execution evidence time"
    )
    if completed_epoch - started_epoch > MAX_CENSUS_EXECUTION_SECONDS:
        fail("custom-staking census execution exceeded the runner duration limit")
    observer_terminal = objects.get("OBSERVER-EVIDENCE-MANIFEST.json")
    observer_halt_time = (
        observer_terminal.get("halt_trigger_block_time")
        if isinstance(observer_terminal, dict)
        else None
    )
    halt_trigger_ns = canonical_nanoseconds(
        observer_halt_time,
        "custom-staking census terminal halt-trigger time",
    )
    if not (
        cutover_evidence_signature_epoch
        <= halt_trigger_ns // 1_000_000_000
        <= started_epoch
        <= completed_epoch
        <= created_epoch
        <= signature_epoch
        <= final_created_epoch
        <= final_signature_epoch
    ):
        fail("custom-staking census execution chronology is non-monotonic")
    if halt_trigger_ns > started_epoch * 1_000_000_000:
        fail("custom-staking census execution began before terminal block H")

    artifacts = final.get("artifacts", {})
    if not (
        artifacts.get("custom_staking_census_execution_evidence_sha256")
        == sha256(files[CENSUS_EXECUTION_EVIDENCE_FILENAME])
        and artifacts.get(
            "custom_staking_census_execution_evidence_detached_signature_sha256"
        )
        == sha256(files[CENSUS_EXECUTION_SIGNATURE_FILENAME])
    ):
        fail("FINAL does not bind the custom-staking census execution evidence")


def validate_dark_bootstrap_contract(
    files: dict[str, bytes], release: dict[str, Any], dark: dict[str, Any]
) -> tuple[dict[str, Any], dict[str, Any], int, int]:
    if set(dark) != {
        "schema",
        "decision",
        "created_at",
        "operator",
        "signature_authority",
        "release_packet_sha256",
        "release_packet_detached_signature_sha256",
        "private_bootstrap_transactions",
        "authorization_semantics",
        "deployment_configs",
        "scope_if_go",
        "always_forbidden_by_this_decision",
        "required_minimum_private_soak",
    }:
        fail("DARK-START does not have the exact v1 field set")
    require_nonempty_string(dark.get("operator"), "DARK operator")
    semantics = require_exact_object(
        dark.get("authorization_semantics"),
        {
            "initiation_event",
            "initiation_deadline",
            "registration_commit_deadline",
            "registration_broadcast_not_after",
            "minimum_registration_inclusion_margin_seconds",
            "if_not_initiated_by_deadline",
            "after_initiation",
        },
        "DARK authorization semantics",
    )
    if not (
        semantics["initiation_event"]
        == "first committed block of the exact private zerone-2 genesis"
        and semantics["if_not_initiated_by_deadline"]
        == "authorization lapses; do not start genesis or broadcast registration transactions"
        and semantics["after_initiation"]
        == "only the exact private registration, soak, evidence capture, recovery test, or explicit candidate abandonment remains authorized; no public or zerone-1 action is added"
    ):
        fail("DARK authorization semantics changed scope")
    initiation_deadline_epoch = canonical_epoch(
        semantics["initiation_deadline"], "DARK initiation deadline"
    )
    registration_deadline_epoch = canonical_epoch(
        semantics["registration_commit_deadline"], "DARK registration deadline"
    )
    registration_cutoff_epoch = canonical_epoch(
        semantics["registration_broadcast_not_after"],
        "DARK registration broadcast cutoff",
    )
    margin = semantics["minimum_registration_inclusion_margin_seconds"]
    if not (
        isinstance(margin, int)
        and not isinstance(margin, bool)
        and margin >= 300
        and registration_deadline_epoch > initiation_deadline_epoch
        and registration_deadline_epoch - registration_cutoff_epoch >= margin
    ):
        fail("DARK registration deadline/cutoff lacks its signed safety margin")

    transactions = require_exact_object(
        dark.get("private_bootstrap_transactions"),
        {"broadcast_order", "operator_onboarding", "custom_validator_registration"},
        "DARK bootstrap transactions",
    )
    if transactions["broadcast_order"] != [
        "operator_onboarding",
        "custom_validator_registration",
    ]:
        fail("DARK bootstrap broadcast order changed")
    onboarding = require_exact_object(
        transactions["operator_onboarding"],
        {
            "filename",
            "chain_id",
            "signed_tx_bytes_sha256",
            "expected_transaction_hash",
            "sender",
            "message_type",
            "did",
            "public_key",
            "account_type",
            "operational_key_hash",
            "metadata",
            "identity_proof_signature",
            "fee",
            "gas_limit",
            "signer_sequence",
            "timeout_height",
            "memo",
            "must_not_reference_dark_start_payload",
        },
        "DARK onboarding transaction",
    )
    registration = require_exact_object(
        transactions["custom_validator_registration"],
        {
            "filename",
            "chain_id",
            "signed_tx_bytes_sha256",
            "expected_transaction_hash",
            "operator",
            "message_type",
            "consensus_pubkey",
            "did",
            "moniker",
            "self_delegation",
            "commission_bps",
            "website",
            "details",
            "fee",
            "gas_limit",
            "signer_sequence",
            "timeout_height",
            "memo",
            "must_not_reference_dark_start_payload",
        },
        "DARK custom-validator transaction",
    )
    identities = release["public_identities"]
    identity_key = onboarding["public_key"]
    if not isinstance(identity_key, str) or not re.fullmatch(r"[0-9a-f]{64}", identity_key):
        fail("DARK onboarding identity public key is not lowercase 32-byte hex")
    identity_proof = onboarding["identity_proof_signature"]
    try:
        decoded_identity_proof = base64.b64decode(identity_proof, validate=True)
    except (binascii.Error, TypeError, ValueError) as exc:
        fail(f"DARK onboarding identity proof is not canonical base64: {exc}")
    if (
        len(decoded_identity_proof) != 64
        or base64.b64encode(decoded_identity_proof).decode() != identity_proof
    ):
        fail("DARK onboarding identity proof is not one canonical 64-byte signature")
    consensus_hex = base64.b64decode(
        identities["validator_consensus_pubkey"], validate=True
    ).hex()
    for contract, label in ((onboarding, "onboarding"), (registration, "registration")):
        require_hash(contract["signed_tx_bytes_sha256"], f"DARK {label} TxRaw")
        require_hash(
            contract["expected_transaction_hash"], f"DARK {label} tx hash", upper=True
        )
        if contract["expected_transaction_hash"] != contract[
            "signed_tx_bytes_sha256"
        ].upper():
            fail(f"DARK {label} Comet hash differs from its TxRaw SHA-256")
    if not (
        onboarding["filename"] == "ZERONE-2-ONBOARD-SIGNED-TX.json"
        and registration["filename"]
        == "ZERONE-2-CUSTOM-VALIDATOR-SIGNED-TX.json"
        and onboarding["chain_id"] == registration["chain_id"] == "zerone-2"
        and onboarding["sender"]
        == registration["operator"]
        == identities["operations_account_address"]
        and onboarding["message_type"] == "/zerone.auth.v1.MsgRegisterAccount"
        and onboarding["did"] == registration["did"] == f"did:zrn:{identity_key}"
        and onboarding["account_type"] == "human"
        and onboarding["operational_key_hash"] == ""
        and onboarding["metadata"] == ""
        and registration["message_type"]
        == "/zerone.staking.v1.MsgRegisterValidator"
        and registration["consensus_pubkey"] == consensus_hex
        and registration["moniker"] == "zerone-2-custodian"
        and registration["self_delegation"] == "111000000"
        and registration["commission_bps"] == "500"
        and registration["website"] == ""
        and registration["details"]
        == "One publicly disclosed custodial validator"
        and onboarding["fee"] == registration["fee"] == "200000uzrn"
        and onboarding["gas_limit"] == registration["gas_limit"] == "200000"
        and onboarding["signer_sequence"] == "0"
        and registration["signer_sequence"] == "1"
        and onboarding["memo"]
        == "zerone-2-private-bootstrap:operator-onboarding"
        and registration["memo"]
        == "zerone-2-private-bootstrap:custom-validator-registration"
        and onboarding["must_not_reference_dark_start_payload"] is True
        and registration["must_not_reference_dark_start_payload"] is True
    ):
        fail("DARK bootstrap transaction semantics differ from RELEASE/policy")
    onboarding_timeout = onboarding["timeout_height"]
    registration_timeout = registration["timeout_height"]
    if not (
        isinstance(onboarding_timeout, str)
        and POSITIVE_HEIGHT.fullmatch(onboarding_timeout)
        and isinstance(registration_timeout, str)
        and POSITIVE_HEIGHT.fullmatch(registration_timeout)
        and int(onboarding_timeout) > 1
        and int(registration_timeout) > int(onboarding_timeout)
    ):
        fail("DARK bootstrap timeout heights do not preserve transaction order")
    if onboarding["signed_tx_bytes_sha256"] == registration[
        "signed_tx_bytes_sha256"
    ] or files[onboarding["filename"]] == files[registration["filename"]]:
        fail("DARK onboarding and registration TxRaw envelopes must be distinct")
    return onboarding, registration, registration_deadline_epoch, registration_cutoff_epoch


def validate_dark_registration_evidence(
    files: dict[str, bytes],
    paths: dict[str, pathlib.Path],
    objects: dict[str, Any],
    release: dict[str, Any],
    dark: dict[str, Any],
    dark_pair: dict[str, str],
    dark_init_pair: dict[str, str],
    dark_commit_ns: int,
    onboarding: dict[str, Any],
    registration: dict[str, Any],
    registration_deadline_epoch: int,
    main: str,
) -> tuple[dict[str, str], int]:
    evidence = objects["DARK-REGISTRATION-EVIDENCE.json"]
    signature_epoch = require_main_signature(
        paths,
        objects,
        "DARK-REGISTRATION-EVIDENCE.json",
        "DARK-REGISTRATION-EVIDENCE.json.sig",
        main,
    )
    evidence_pair = exact_pair(
        files,
        "DARK-REGISTRATION-EVIDENCE.json",
        "DARK-REGISTRATION-EVIDENCE.json.sig",
    )
    if not isinstance(evidence, dict) or set(evidence) != {
        "schema",
        "attestation_result",
        "created_at",
        "signature_authority",
        "dark_start_decision",
        "dark_start_initiation_evidence",
        "registration_commit_deadline",
        "deadline_satisfied",
        "operator_onboarding",
        "custom_validator_registration",
        "post_registration_state",
        "authority_limit",
    }:
        fail("DARK registration evidence has the wrong exact field set")
    if not (
        evidence["schema"] == "zerone-2-dark-registration-evidence-v1"
        and evidence["attestation_result"] == "MATCH"
        and evidence["dark_start_decision"] == dark_pair
        and evidence["dark_start_initiation_evidence"] == dark_init_pair
        and evidence["registration_commit_deadline"]
        == dark["authorization_semantics"]["registration_commit_deadline"]
        and evidence["deadline_satisfied"] is True
        and evidence["authority_limit"]
        == "factual proof that the two exact DARK-authorized private zerone-2 bootstrap transactions committed in order and produced only the declared identity and custom-validator state; adds no public or zerone-1 authority"
    ):
        fail("DARK registration evidence root is incomplete or over-broad")
    receipt_keys = {
        "signed_tx_bytes_sha256",
        "expected_transaction_hash",
        "committed_transaction_hash",
        "deliver_code",
        "committed_height",
        "committed_block_time",
        "raw_transaction_query_evidence_sha256",
        "independent_edge_evidence_sha256",
    }
    receipts: list[tuple[dict[str, Any], dict[str, Any], str]] = []
    for field, contract in (
        ("operator_onboarding", onboarding),
        ("custom_validator_registration", registration),
    ):
        receipt = require_exact_object(
            evidence[field], receipt_keys, f"DARK {field} receipt"
        )
        if not (
            receipt["signed_tx_bytes_sha256"] == contract["signed_tx_bytes_sha256"]
            and receipt["expected_transaction_hash"]
            == contract["expected_transaction_hash"]
            and receipt["committed_transaction_hash"]
            == contract["expected_transaction_hash"]
            and receipt["deliver_code"] == 0
            and isinstance(receipt["committed_height"], str)
            and POSITIVE_HEIGHT.fullmatch(receipt["committed_height"])
            and int(receipt["committed_height"]) <= int(contract["timeout_height"])
        ):
            fail(f"DARK {field} receipt differs from the exact signed transaction")
        require_hash(
            receipt["raw_transaction_query_evidence_sha256"],
            f"DARK {field} raw transaction query",
        )
        require_hash(
            receipt["independent_edge_evidence_sha256"],
            f"DARK {field} independent edge evidence",
        )
        receipts.append((receipt, contract, field))
    onboarding_receipt, _, _ = receipts[0]
    registration_receipt, _, _ = receipts[1]
    onboarding_ns = canonical_nanoseconds(
        onboarding_receipt["committed_block_time"], "DARK onboarding commit time"
    )
    registration_ns = canonical_nanoseconds(
        registration_receipt["committed_block_time"],
        "DARK custom-validator commit time",
    )
    deadline_ns = registration_deadline_epoch * 1_000_000_000
    if not (
        dark_commit_ns <= onboarding_ns <= registration_ns <= deadline_ns
        and int(onboarding_receipt["committed_height"])
        < int(registration_receipt["committed_height"])
    ):
        fail("DARK block/onboarding/registration chronology or order is invalid")
    state = require_exact_object(
        evidence["post_registration_state"],
        {
            "chain_id",
            "operator",
            "did",
            "identity_public_key",
            "validator_consensus_pubkey",
            "custom_validator_active",
            "custom_validator_self_delegation_uzrn",
            "custom_staking_module_balance_uzrn",
            "total_supply_uzrn",
            "sdk_validator_count",
            "account_query_evidence_sha256",
            "custom_validator_query_evidence_sha256",
            "module_balance_query_evidence_sha256",
            "supply_query_evidence_sha256",
        },
        "DARK post-registration state",
    )
    if not (
        state["chain_id"] == "zerone-2"
        and state["operator"] == onboarding["sender"] == registration["operator"]
        and state["did"] == onboarding["did"] == registration["did"]
        and state["identity_public_key"] == onboarding["public_key"]
        and state["validator_consensus_pubkey"] == registration["consensus_pubkey"]
        and state["custom_validator_active"] is True
        and state["custom_validator_self_delegation_uzrn"] == "111000000"
        and state["custom_staking_module_balance_uzrn"] == "111000000"
        and state["total_supply_uzrn"] == "13555000000"
        and state["sdk_validator_count"] == 1
        and state["operator"]
        == release["public_identities"]["operations_account_address"]
    ):
        fail("DARK post-registration state differs from RELEASE/policy")
    for field in (
        "account_query_evidence_sha256",
        "custom_validator_query_evidence_sha256",
        "module_balance_query_evidence_sha256",
        "supply_query_evidence_sha256",
    ):
        require_hash(state[field], f"DARK registration {field}")
    evidence_created_epoch = canonical_epoch(
        evidence["created_at"], "DARK registration evidence creation time"
    )
    now_ns = int(dt.datetime.now(tz=dt.timezone.utc).timestamp()) * 1_000_000_000
    if not (
        registration_ns
        <= evidence_created_epoch * 1_000_000_000
        <= signature_epoch * 1_000_000_000
        <= now_ns + 300_000_000_000
    ):
        fail("DARK registration commit/evidence/signature chronology is invalid")
    return evidence_pair, signature_epoch


def require_notice_url(value: Any) -> str:
    """Require one exact HTTPS document URL; never resolve or fetch it."""
    if not isinstance(value, str) or not 1 <= len(value) <= 2048:
        fail("PRE-NOTICE public URL must be a bounded HTTPS document URL")
    # Avoid ambiguous parser normalization, credentials, encoded path aliases,
    # query tokens and fragments. The human-approved document has one address.
    if not re.fullmatch(r"https://[a-z0-9.-]+/[A-Za-z0-9._~/-]+", value):
        fail("PRE-NOTICE public URL is not canonical HTTPS")
    parsed = urlsplit(value)
    host = parsed.hostname or ""
    labels = host.split(".")
    if (
        len(host) > 253
        or len(labels) < 2
        or not any(character.isalpha() for character in labels[-1])
        or any(
            not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
            for label in labels
        )
        or "//" in parsed.path
        or any(part in {".", ".."} for part in parsed.path.split("/"))
    ):
        fail("PRE-NOTICE public URL has an invalid host or ambiguous path")
    return value


def validate_pre_notice(
    files: dict[str, bytes],
    objects: dict[str, Any],
    signature_epoch: int,
    dark_registration_signature_epoch: int,
    soak_end_ns: int,
    rehearsal_epoch: int,
    prepublish: bool,
) -> tuple[dict[str, str], int]:
    decision = objects["PRE-NOTICE-DECISION.json"]
    if not isinstance(decision, dict) or set(decision) != {
        "schema", "decision", "created_at", "operator", "signature_authority",
        "release_packet", "dark_start_decision", "dark_start_initiation_evidence",
        "dark_registration_evidence", "private_soak_evidence_sha256",
        "halt_rehearsal_evidence_sha256", "notice", "checkpoint_plan",
        "publication_deadline", "scope",
    }:
        fail("PRE-NOTICE decision has the wrong schema shape")
    if not (
        decision["schema"] == "zerone-2-pre-notice-decision-v1"
        and decision["decision"] == "GO"
        and isinstance(decision["operator"], str)
        and decision["operator"].strip()
        and len(decision["operator"]) <= 256
    ):
        fail("PRE-NOTICE decision is not an attributed GO")
    authority = decision["signature_authority"]
    if not isinstance(authority, dict) or set(authority) != {
        "algorithm", "authorized_signer_fingerprint", "detached_signature_filename"
    }:
        fail("PRE-NOTICE signature authority has the wrong schema shape")
    scope = decision["scope"]
    if not isinstance(scope, dict) or set(scope) != set(NOTICE_SCOPE) or any(
        scope[key] is not expected for key, expected in NOTICE_SCOPE.items()
    ):
        fail("PRE-NOTICE scope must authorize only exact notice publication")
    for field, name in (
        ("release_packet", "RELEASE-PACKET.json"),
        ("dark_start_decision", "DARK-START-DECISION.json"),
        ("dark_start_initiation_evidence", "DARK-START-INITIATION-EVIDENCE.json"),
        ("dark_registration_evidence", "DARK-REGISTRATION-EVIDENCE.json"),
    ):
        if decision[field] != exact_pair(files, name, name + ".sig"):
            fail(f"PRE-NOTICE {field} differs from the completed authority chain")
    for field, name in (
        ("private_soak_evidence_sha256", "PRIVATE-SOAK-EVIDENCE.json"),
        ("halt_rehearsal_evidence_sha256", "HALT-REHEARSAL-EVIDENCE.json"),
    ):
        if decision[field] != sha256(files[name]):
            fail(f"PRE-NOTICE {field} differs from the bundled artifact")
    notice = decision["notice"]
    if not isinstance(notice, dict) or set(notice) != {
        "filename", "sha256", "public_url"
    } or not (
        notice["filename"] == "PUBLIC-NOTICE.md"
        and notice["sha256"] == sha256(files["PUBLIC-NOTICE.md"])
    ):
        fail("PRE-NOTICE notice differs from the exact bundled bytes")
    require_notice_url(notice["public_url"])
    try:
        notice_text = files["PUBLIC-NOTICE.md"].decode("utf-8")
    except UnicodeDecodeError:
        fail("PRE-NOTICE notice must be UTF-8 text")
    if (
        not notice_text.strip()
        or "\x00" in notice_text
        or contains_placeholder(notice_text)
    ):
        fail("PRE-NOTICE notice is empty, contains NUL, or retains a placeholder")
    plan = decision["checkpoint_plan"]
    if not isinstance(plan, dict) or set(plan) != {
        "checkpoint_state_height", "final_committed_anchor_height", "halt_trigger_height"
    }:
        fail("PRE-NOTICE checkpoint plan has the wrong schema shape")
    f, a, h = (
        plan["checkpoint_state_height"], plan["final_committed_anchor_height"],
        plan["halt_trigger_height"],
    )
    if not all(isinstance(value, str) and POSITIVE_HEIGHT.fullmatch(value) for value in (f, a, h)):
        fail("PRE-NOTICE F/A/H is malformed")
    if int(a) != int(f) + 1 or int(h) != int(a) + 1:
        fail("PRE-NOTICE must satisfy A=F+1 and H=A+1")
    created_epoch = canonical_epoch(decision["created_at"], "PRE-NOTICE creation time")
    deadline_epoch = canonical_epoch(
        decision["publication_deadline"], "PRE-NOTICE publication deadline"
    )
    now = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
    if not (
        dark_registration_signature_epoch <= created_epoch <= signature_epoch <= now
        and soak_end_ns <= created_epoch * 1_000_000_000
        and rehearsal_epoch <= created_epoch
        and signature_epoch <= deadline_epoch
    ):
        fail("PRE-NOTICE signature/soak/rehearsal/deadline chronology is non-monotonic")
    if prepublish and now > deadline_epoch:
        fail("PRE-NOTICE publication deadline has passed")
    return exact_pair(files, "PRE-NOTICE-DECISION.json", "PRE-NOTICE-DECISION.json.sig"), deadline_epoch


def validate_notice_publication(
    files: dict[str, bytes],
    objects: dict[str, Any],
    notice_pair: dict[str, str],
    signature_epoch: int,
    deadline_epoch: int,
) -> int:
    decision = objects["PRE-NOTICE-DECISION.json"]
    cutover = objects["CUTOVER-DECISION.json"]
    publication = objects["PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json"]
    if not isinstance(publication, dict) or set(publication) != {
        "schema", "pre_notice_decision", "notice_sha256", "published_at",
        "publication_capture_sha256", "public_url",
    }:
        fail("notice publication evidence has the wrong schema shape")
    if not (
        publication["schema"] == "zerone-public-notice-publication-evidence-v2"
        and publication["pre_notice_decision"] == notice_pair
        and publication["notice_sha256"] == decision["notice"]["sha256"]
        and publication["public_url"] == decision["notice"]["public_url"]
        and cutover.get("pre_notice_decision") == notice_pair
        and cutover.get("checkpoint_plan") == decision["checkpoint_plan"]
        and cutover.get("public_notice_publication_evidence_sha256")
        == sha256(files["PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json"])
        and cutover.get("public_notice_published_at") == publication["published_at"]
    ):
        fail("CUTOVER notice publication differs from PRE-NOTICE authority or bundled evidence")
    capture = files["PUBLIC-NOTICE-CAPTURE.md"]
    if (
        capture != files["PUBLIC-NOTICE.md"]
        or publication["publication_capture_sha256"] != sha256(capture)
    ):
        fail("notice publication capture does not match the authorized notice bytes")
    published_epoch = canonical_epoch(publication["published_at"], "notice publication time")
    now = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
    if not signature_epoch <= published_epoch <= min(deadline_epoch, now):
        fail("notice publication was outside its signed authorization window")
    return published_epoch


def validate_cutover_chain(
    files: dict[str, bytes],
    paths: dict[str, pathlib.Path],
    objects: dict[str, Any],
    main: str,
    require_initiation: bool,
    config_policy: pathlib.Path,
    dark_registration_only: bool = False,
    dark_preinit: bool = False,
    notice_prepublish: bool = False,
) -> tuple[str, str, str]:
    if sum((dark_registration_only, dark_preinit, notice_prepublish)) > 1:
        fail("verification stage is internally inconsistent")
    release = objects["RELEASE-PACKET.json"]
    dark = objects["DARK-START-DECISION.json"]
    dark_init = objects.get("DARK-START-INITIATION-EVIDENCE.json")
    cutover = objects.get("CUTOVER-DECISION.json")

    signature_times: dict[str, int] = {}
    signature_payloads = [
        ("RELEASE-PACKET.json", "RELEASE-PACKET.json.sig"),
        ("DARK-START-DECISION.json", "DARK-START-DECISION.json.sig"),
    ]
    if not dark_preinit:
        signature_payloads.append(
            (
                "DARK-START-INITIATION-EVIDENCE.json",
                "DARK-START-INITIATION-EVIDENCE.json.sig",
            )
        )
    if not dark_registration_only and not dark_preinit:
        signature_payloads.append(
            ("PRE-NOTICE-DECISION.json", "PRE-NOTICE-DECISION.json.sig")
        )
    if not dark_registration_only and not dark_preinit and not notice_prepublish:
        signature_payloads.append(
            ("CUTOVER-DECISION.json", "CUTOVER-DECISION.json.sig")
        )
    for payload, signature in signature_payloads:
        signature_times[payload] = require_main_signature(
            paths, objects, payload, signature, main
        )

    if not isinstance(release, dict) or release.get("schema") != "zerone-2-release-packet-v2":
        fail("release packet schema changed")
    if set(release) != {
        "schema",
        "created_at",
        "chain_id",
        "signature_authority",
        "predecessor",
        "source",
        "genesis",
        "ceremony_artifacts",
        "components",
        "public_identities",
        "archive_render_contract",
        "archive_gateway_render_contract",
        "custom_staking_census_execution",
        "deployment_configs",
        "phase_dependent_config_template_sha256",
        "monitoring_alerts_sha256",
        "operator_tool_manifest_sha256",
        "accepted_policy",
    }:
        fail("release packet does not have the exact v2 field set")
    if release.get("chain_id") != "zerone-2":
        fail("release packet chain ID changed")
    transition = (
        release.get("public_identities", {})
        .get("transition_attestation", {})
        .get("authorized_signer_fingerprint")
    )
    if not isinstance(transition, str) or not FINGERPRINT.fullmatch(transition):
        fail("release transition fingerprint is malformed")
    if normalize_fingerprint(transition) == normalize_fingerprint(main):
        fail("main and transition fingerprints must differ")
    release_created_epoch = canonical_epoch(
        release.get("created_at"), "RELEASE creation time"
    )
    dark_created_epoch = canonical_epoch(dark.get("created_at"), "DARK creation time")
    if not (
        release_created_epoch
        <= signature_times["RELEASE-PACKET.json"]
        <= dark_created_epoch
        <= signature_times["DARK-START-DECISION.json"]
    ):
        fail("RELEASE/DARK creation and signature chronology is non-monotonic")
    if not isinstance(release.get("components"), dict) or set(release["components"]) != {
        "zerone_1_halt",
        "zerone_2_runtime",
        "query_gateway",
    }:
        fail("RELEASE component set is malformed")
    validate_release_ceremony(files, objects, release, main)
    validate_census_execution_contract(files, release)
    validate_monitoring_artifacts(
        files, objects, release, release_created_epoch
    )
    component_images = validate_release_components(
        files, paths, objects, release, main, release_created_epoch
    )
    validate_archive_gateway_render_contract(release)
    release_configs = release.get("deployment_configs", {})
    release_keys = {
        "zerone_2_validator",
        "zerone_2_edge_private",
        "zerone_2_edge_query_soak",
        "zerone_2_gateway_private",
        "zerone_2_edge_public",
        "zerone_2_gateway_public",
        "zerone_1_archive_gateway",
    }
    if not isinstance(release_configs, dict) or set(release_configs) != release_keys:
        fail("RELEASE deployment mapping set is incomplete")
    for key, mapping_value in release_configs.items():
        expected_mapping_fields = {
            "app",
            "role",
            "image_component",
            "image_ref",
        }
        if key != "zerone_1_archive_gateway":
            expected_mapping_fields.add("sha256")
        if (
            not isinstance(mapping_value, dict)
            or set(mapping_value) != expected_mapping_fields
        ):
            fail(f"RELEASE {key} mapping is malformed")
        component = mapping_value.get("image_component")
        if mapping_value.get("image_ref") != component_images.get(component):
            fail(f"RELEASE {key} image does not join its component")
        if key != "zerone_1_archive_gateway":
            require_hash(mapping_value.get("sha256"), f"RELEASE {key} config")
    apps = {key: value["app"] for key, value in release_configs.items()}
    if not (
        apps["zerone_2_edge_private"] == apps["zerone_2_edge_query_soak"]
        == apps["zerone_2_edge_public"]
        and apps["zerone_2_gateway_private"] == apps["zerone_2_gateway_public"]
        and len(
            {
                apps["zerone_2_validator"],
                apps["zerone_2_edge_private"],
                apps["zerone_2_gateway_private"],
                apps["zerone_1_archive_gateway"],
            }
        )
        == 4
        and not any(
            app in {"zerone-1", "zerone-1-observer", "zerone-1-archive"}
            for app in apps.values()
        )
    ):
        fail("RELEASE app topology is unsafe")

    release_pair = exact_pair(files, "RELEASE-PACKET.json", "RELEASE-PACKET.json.sig")
    dark_pair = exact_pair(
        files, "DARK-START-DECISION.json", "DARK-START-DECISION.json.sig"
    )
    if not isinstance(dark, dict) or not (
        dark.get("schema") == "zerone-2-dark-start-decision-v1"
        and dark.get("decision") == "GO"
        and dark.get("release_packet_sha256") == release_pair["sha256"]
        and dark.get("release_packet_detached_signature_sha256")
        == release_pair["detached_signature_sha256"]
        and dark.get("required_minimum_private_soak")
        == {"blocks": 1000, "wall_time_minutes": 60}
        and set(dark.get("deployment_configs", {}))
        == {
            "zerone_2_validator",
            "zerone_2_edge_private",
            "zerone_2_edge_query_soak",
            "zerone_2_gateway_private",
        }
        and "broadcast any zerone-1 transaction"
        in dark.get("always_forbidden_by_this_decision", [])
    ):
        fail("DARK-START authority is incomplete or differs from RELEASE")
    if any(
        dark["deployment_configs"].get(key) != release_configs.get(key)
        for key in dark["deployment_configs"]
    ):
        fail("DARK-START deployment mappings differ from RELEASE")
    (
        onboarding_contract,
        registration_contract,
        registration_deadline_epoch,
        registration_cutoff_epoch,
    ) = validate_dark_bootstrap_contract(files, release, dark)

    dark_deadline = dark.get("authorization_semantics", {}).get("initiation_deadline")
    dark_deadline_epoch = canonical_epoch(dark_deadline, "DARK-START deadline")
    now = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
    if dark_preinit:
        if signature_times["DARK-START-DECISION.json"] > dark_deadline_epoch:
            fail("DARK-START decision was signed after its initiation deadline")
        if now > dark_deadline_epoch:
            fail("DARK-START initiation deadline has passed")
        return "", "", ""

    if not isinstance(dark_init, dict):
        fail("DARK-START initiation evidence is missing or malformed")
    dark_evidence_created_epoch = canonical_epoch(
        dark_init.get("created_at"), "DARK evidence creation time"
    )
    dark_init_pair = exact_pair(
        files,
        "DARK-START-INITIATION-EVIDENCE.json",
        "DARK-START-INITIATION-EVIDENCE.json.sig",
    )
    block = dark_init.get("first_committed_block", {})
    if not (
        dark_init.get("schema") == "zerone-2-dark-start-initiation-evidence-v1"
        and dark_init.get("attestation_result") == "MATCH"
        and dark_init.get("dark_start_decision") == dark_pair
        and dark_init.get("initiation_deadline") == dark_deadline
        and dark_init.get("deadline_satisfied") is True
        and block.get("chain_id") == "zerone-2"
        and block.get("height") == "1"
        and isinstance(block.get("block_id_hash"), str)
        and UPPER_HASH.fullmatch(block["block_id_hash"])
        and isinstance(block.get("app_hash"), str)
        and UPPER_HASH.fullmatch(block["app_hash"])
    ):
        fail("DARK-START initiation evidence is incomplete or mismatched")
    for key in ("validator_evidence_sha256", "independent_edge_evidence_sha256"):
        require_hash(block.get(key), f"DARK initiation {key}")
    dark_commit_ns = canonical_nanoseconds(
        block.get("committed_block_time"), "DARK first-block time"
    )
    if dark_commit_ns > dark_deadline_epoch * 1_000_000_000:
        fail("DARK first block committed after its signed deadline")
    if dark_commit_ns > now * 1_000_000_000:
        fail("DARK first block is dated in the future")
    if not (
        signature_times["DARK-START-DECISION.json"] * 1_000_000_000
        <= dark_commit_ns
        <= dark_evidence_created_epoch * 1_000_000_000
        <= signature_times["DARK-START-INITIATION-EVIDENCE.json"]
        * 1_000_000_000
    ):
        fail("DARK decision/block/evidence chronology is non-monotonic")

    if dark_registration_only:
        if now > registration_cutoff_epoch:
            fail("DARK registration broadcast cutoff has passed")
        return "", "", ""

    dark_registration_pair, dark_registration_signature_epoch = (
        validate_dark_registration_evidence(
            files,
            paths,
            objects,
            release,
            dark,
            dark_pair,
            dark_init_pair,
            dark_commit_ns,
            onboarding_contract,
            registration_contract,
            registration_deadline_epoch,
            main,
        )
    )

    soak = objects["PRIVATE-SOAK-EVIDENCE.json"]
    soak_keys = {
        "schema",
        "chain_id",
        "release_packet",
        "dark_start_decision",
        "dark_start_initiation_evidence",
        "dark_registration_evidence",
        "genesis_sha256",
        "start",
        "end",
        "observed_blocks",
        "observed_wall_time_minutes",
        "validator",
        "edge",
        "supply_uzrn",
        "sdk_validator_count",
        "protocol_dark_audit_sha256",
        "query_gateway_smoke_evidence_sha256",
        "restart_export_import_rehearsal_sha256",
        "result",
    }
    if not isinstance(soak, dict) or set(soak) != soak_keys:
        fail("private soak evidence has the wrong schema shape")
    start = soak.get("start", {})
    end = soak.get("end", {})
    if not (
        soak["schema"] == "zerone-2-private-soak-evidence-v1"
        and soak["chain_id"] == "zerone-2"
        and soak["release_packet"] == release_pair
        and soak["dark_start_decision"] == dark_pair
        and soak["dark_start_initiation_evidence"] == dark_init_pair
        and soak["dark_registration_evidence"] == dark_registration_pair
        and soak["genesis_sha256"] == release.get("genesis", {}).get("sha256")
        and soak["result"] == "MATCH"
        and start.get("height") == "1"
        and start.get("block_time") == block.get("committed_block_time")
        and start.get("app_hash") == block.get("app_hash")
        and isinstance(end.get("height"), str)
        and POSITIVE_HEIGHT.fullmatch(end["height"])
        and int(end["height"]) - 1 >= 1000
        and soak["observed_blocks"] >= 1000
        and soak["observed_wall_time_minutes"] >= 60
        and soak["supply_uzrn"] == "13555000000"
        and soak["sdk_validator_count"] == 1
        and soak["validator"].get("node_id")
        == release.get("public_identities", {}).get("validator_node_id")
        and soak["validator"].get("consensus_pubkey")
        == release.get("public_identities", {}).get("validator_consensus_pubkey")
        and soak["validator"].get("app_hash") == end.get("app_hash")
        and soak["edge"].get("app_hash") == end.get("app_hash")
    ):
        fail("private soak evidence does not prove the signed minimum successor soak")
    require_hash(end.get("app_hash"), "private soak end AppHash", upper=True)
    require_hash(soak["edge"].get("evidence_sha256"), "private edge evidence")
    for key in (
        "protocol_dark_audit_sha256",
        "query_gateway_smoke_evidence_sha256",
        "restart_export_import_rehearsal_sha256",
    ):
        require_hash(soak.get(key), f"private soak {key}")
    soak_start_ns = canonical_nanoseconds(start.get("block_time"), "private soak start")
    soak_end_ns = canonical_nanoseconds(end.get("block_time"), "private soak end")
    if soak_end_ns - soak_start_ns < 3_600_000_000_000:
        fail("private soak wall time is less than 60 minutes")

    rehearsal = objects["HALT-REHEARSAL-EVIDENCE.json"]
    rehearsal_keys = {
        "schema",
        "chain_id",
        "release_packet",
        "source_commit",
        "binary_sha256",
        "image_ref",
        "tested_checkpoint_plan",
        "independent_observer_matched",
        "anchor_transaction_count",
        "halt_trigger_transaction_count",
        "sanitized_archive_allowlist_matched",
        "raw_evidence_manifest_sha256",
        "completed_at",
        "result",
    }
    tested = rehearsal.get("tested_checkpoint_plan", {}) if isinstance(rehearsal, dict) else {}
    if not isinstance(rehearsal, dict) or set(rehearsal) != rehearsal_keys or not (
        rehearsal["schema"] == "zerone-1-halt-rehearsal-evidence-v1"
        and rehearsal["chain_id"] == "zerone-1"
        and rehearsal["release_packet"] == release_pair
        and rehearsal["source_commit"] == release.get("source", {}).get("commit")
        and rehearsal["binary_sha256"]
        == release["components"]["zerone_1_halt"].get("binary_sha256")
        and rehearsal["image_ref"] == component_images["zerone_1_halt"]
        and rehearsal["independent_observer_matched"] is True
        and rehearsal["anchor_transaction_count"] == 0
        and rehearsal["halt_trigger_transaction_count"] == 0
        and rehearsal["sanitized_archive_allowlist_matched"] is True
        and rehearsal["result"] == "MATCH"
    ):
        fail("halt rehearsal evidence is incomplete or mismatched")
    tested_f = tested.get("checkpoint_state_height")
    tested_a = tested.get("final_committed_anchor_height")
    tested_h = tested.get("halt_trigger_height")
    if not all(
        isinstance(value, str) and POSITIVE_HEIGHT.fullmatch(value)
        for value in (tested_f, tested_a, tested_h)
    ) or int(tested_a) != int(tested_f) + 1 or int(tested_h) != int(tested_a) + 1:
        fail("halt rehearsal F/A/H semantics are malformed")
    require_hash(
        rehearsal["raw_evidence_manifest_sha256"], "halt rehearsal raw evidence"
    )
    rehearsal_epoch = canonical_epoch(
        rehearsal["completed_at"], "halt rehearsal completion time"
    )

    notice_pair, notice_deadline_epoch = validate_pre_notice(
        files, objects, signature_times["PRE-NOTICE-DECISION.json"],
        dark_registration_signature_epoch, soak_end_ns, rehearsal_epoch,
        notice_prepublish,
    )
    if notice_prepublish:
        return "", "", ""

    if not isinstance(cutover, dict) or cutover.get("schema") != "zerone-2-cutover-decision-v1":
        fail("CUTOVER decision schema changed")
    if not (
        cutover.get("decision") == "GO"
        and cutover.get("release_packet_sha256") == release_pair["sha256"]
        and cutover.get("release_packet_detached_signature_sha256")
        == release_pair["detached_signature_sha256"]
        and cutover.get("dark_start_decision_sha256") == dark_pair["sha256"]
        and cutover.get("dark_start_detached_signature_sha256")
        == dark_pair["detached_signature_sha256"]
        and cutover.get("dark_start_initiation_evidence_sha256")
        == dark_init_pair["sha256"]
        and cutover.get("dark_start_initiation_evidence_detached_signature_sha256")
        == dark_init_pair["detached_signature_sha256"]
        and cutover.get("dark_registration_evidence_sha256")
        == dark_registration_pair["sha256"]
        and cutover.get("dark_registration_evidence_detached_signature_sha256")
        == dark_registration_pair["detached_signature_sha256"]
    ):
        fail("CUTOVER does not bind the exact predecessor authority chain")
    raw_hashes = {
        "private_soak_evidence_sha256": "PRIVATE-SOAK-EVIDENCE.json",
        "halt_rehearsal_evidence_sha256": "HALT-REHEARSAL-EVIDENCE.json",
        "public_notice_sha256": "PUBLIC-NOTICE.md",
    }
    for field, filename in raw_hashes.items():
        if cutover.get(field) != sha256(files[filename]):
            fail(f"CUTOVER {field} differs from the bundled artifact")
    notice_epoch = validate_notice_publication(
        files, objects, notice_pair,
        signature_times["PRE-NOTICE-DECISION.json"], notice_deadline_epoch,
    )

    plan = cutover.get("checkpoint_plan", {})
    f = plan.get("checkpoint_state_height")
    a = plan.get("final_committed_anchor_height")
    h = plan.get("halt_trigger_height")
    if not all(isinstance(value, str) and POSITIVE_HEIGHT.fullmatch(value) for value in (f, a, h)):
        fail("CUTOVER F/A/H is malformed")
    if int(a) != int(f) + 1 or int(h) != int(a) + 1:
        fail("CUTOVER must satisfy A=F+1 and H=A+1")
    if set(cutover.get("deployment_configs", {})) != {
        "zerone_1_halt_signer",
        "zerone_1_observer",
    }:
        fail("CUTOVER halt deployment scope is incomplete")
    signer_mapping = cutover["deployment_configs"]["zerone_1_halt_signer"]
    observer_mapping = cutover["deployment_configs"]["zerone_1_observer"]
    validate_config_mapping(
        files, "fly.halt-signer.toml", signer_mapping, "CUTOVER halt signer"
    )
    validate_config_mapping(
        files, "fly.observer.toml", observer_mapping, "CUTOVER observer"
    )
    if not (
        signer_mapping.get("app") == "zerone-1"
        and signer_mapping.get("role") == "signer"
        and signer_mapping.get("image_component") == "zerone_1_halt"
        and signer_mapping.get("image_ref") == component_images["zerone_1_halt"]
        and observer_mapping.get("app") == "zerone-1-observer"
        and observer_mapping.get("role") == "observer"
        and observer_mapping.get("image_component") == "zerone_1_halt"
        and observer_mapping.get("image_ref") == component_images["zerone_1_halt"]
    ):
        fail("CUTOVER halt mappings differ from the fixed app/role/image policy")
    run_config_policy(
        config_policy,
        paths["fly.halt-signer.toml"],
        "zerone-2-cutover-decision-v1",
        "zerone_1_halt_signer",
        "-",
        f,
        a,
        h,
        "-",
        "-",
    )
    run_config_policy(
        config_policy,
        paths["fly.observer.toml"],
        "zerone-2-cutover-decision-v1",
        "zerone_1_observer",
        "-",
        f,
        a,
        h,
        "-",
        "-",
    )
    continuation = cutover.get("deterministic_private_continuation", {})
    release_render = release.get("archive_render_contract", {})
    cutover_render = continuation.get("render_contract", {})
    release_template_hashes = release.get("phase_dependent_config_template_sha256", {})
    if not (
        continuation.get("attestation_algorithm") == "openpgp"
        and normalize_fingerprint(
            continuation.get("authorized_transition_signer_fingerprint", "")
        )
        == normalize_fingerprint(transition)
        and continuation.get("allowed_adoption_authority_schema")
        == "zerone-1-archive-adoption-authority-v1"
        and continuation.get("allowed_transition_manifest_schema")
        == "zerone-1-archive-transition-v1"
        and continuation.get("required_attestation_result") == "MATCH"
        and cutover_render.get("schema") == release_render.get("schema")
        == "zerone-1-archive-render-contract-v1"
        and cutover_render.get("renderer_path") == release_render.get("renderer_path")
        == "deploy/mainnet/render-archive-configs.sh"
        and cutover_render.get("renderer_sha256")
        == release_render.get("renderer_sha256")
        and cutover_render.get("archive_candidate_template_sha256")
        == release_template_hashes.get("zerone_1_archive_candidate")
        and cutover_render.get("archive_template_sha256")
        == release_template_hashes.get("zerone_1_archive")
        and cutover_render.get("static_constraints")
        == release_render.get("static_constraints")
    ):
        fail("CUTOVER deterministic continuation differs from RELEASE")
    for label, value in (
        ("renderer", cutover_render.get("renderer_sha256")),
        ("candidate template", cutover_render.get("archive_candidate_template_sha256")),
        ("archive template", cutover_render.get("archive_template_sha256")),
    ):
        require_hash(value, f"CUTOVER {label}")
    tx = cutover.get("successor_commitment_transaction", {})
    if not (
        tx.get("chain_id") == "zerone-1"
        and tx.get("message_type") == "/cosmos.bank.v1beta1.MsgSend"
        and tx.get("recipient_equals_sender") is True
        and tx.get("amount") == "1uzrn"
        and tx.get("fee") == "200000uzrn"
        and tx.get("gas_limit") == "200000"
        and tx.get("must_not_reference_cutover_payload") is True
    ):
        fail("CUTOVER transaction contract is incomplete")
    require_hash(tx.get("signed_tx_bytes_sha256"), "CUTOVER signed TxRaw")
    require_hash(tx.get("expected_transaction_hash"), "CUTOVER tx hash", upper=True)
    timeout_height = tx.get("timeout_height")
    minimum_lead = cutover.get("authorization_semantics", {}).get(
        "minimum_halt_lead_blocks"
    )
    if not (
        isinstance(timeout_height, str)
        and POSITIVE_HEIGHT.fullmatch(timeout_height)
        and isinstance(minimum_lead, int)
        and not isinstance(minimum_lead, bool)
        and minimum_lead >= 100
        and int(timeout_height) + minimum_lead <= int(f)
    ):
        fail("CUTOVER timeout height does not preserve the signed halt lead")
    expected_memo = (
        f"successor_chain_id={release.get('chain_id')};"
        f"successor_genesis_sha256={release.get('genesis', {}).get('sha256')};"
        f"checkpoint_state_height={f};final_committed_height={a};"
        f"halt_trigger_height={h}"
    )
    if tx.get("memo") != expected_memo:
        fail("CUTOVER transaction memo differs from RELEASE/F/A/H")
    semantics = cutover.get("authorization_semantics", {})
    deadline = semantics.get("initiation_deadline")
    deadline_epoch = canonical_epoch(deadline, "CUTOVER deadline")
    broadcast_not_after = semantics.get("broadcast_not_after")
    broadcast_epoch = canonical_epoch(
        broadcast_not_after, "CUTOVER broadcast-not-after"
    )
    inclusion_margin = semantics.get("minimum_inclusion_margin_seconds")
    if not (
        isinstance(inclusion_margin, int)
        and not isinstance(inclusion_margin, bool)
        and inclusion_margin >= 300
        and deadline_epoch - broadcast_epoch >= inclusion_margin
    ):
        fail("CUTOVER broadcast cutoff lacks the signed inclusion margin")
    cutover_created_epoch = canonical_epoch(
        cutover.get("created_at"), "CUTOVER creation time"
    )
    if not (
        signature_times["DARK-START-INITIATION-EVIDENCE.json"]
        <= dark_registration_signature_epoch
        <= cutover_created_epoch
        <= signature_times["CUTOVER-DECISION.json"]
        and
        soak_end_ns <= cutover_created_epoch * 1_000_000_000
        and rehearsal_epoch <= cutover_created_epoch
        and notice_epoch <= cutover_created_epoch
    ):
        fail("CUTOVER was created before its soak/rehearsal/notice prerequisites")
    if not require_initiation and int(dt.datetime.now(tz=dt.timezone.utc).timestamp()) > broadcast_epoch:
        fail("CUTOVER broadcast cutoff has passed")

    if require_initiation:
        cutover_evidence_signature_epoch = require_main_signature(
            paths,
            objects,
            "CUTOVER-INITIATION-EVIDENCE.json",
            "CUTOVER-INITIATION-EVIDENCE.json.sig",
            main,
        )
        evidence = objects["CUTOVER-INITIATION-EVIDENCE.json"]
        cutover_pair = exact_pair(
            files, "CUTOVER-DECISION.json", "CUTOVER-DECISION.json.sig"
        )
        deadline = cutover["authorization_semantics"]["initiation_deadline"]
        committed = evidence.get("successor_commitment_transaction", {})
        if not (
            evidence.get("schema") == "zerone-2-cutover-initiation-evidence-v1"
            and evidence.get("attestation_result") == "MATCH"
            and evidence.get("cutover_decision") == cutover_pair
            and evidence.get("initiation_deadline") == deadline
            and evidence.get("deadline_satisfied") is True
            and evidence.get("public_notice", {}).get("sha256")
            == cutover["public_notice_sha256"]
            and evidence.get("public_notice", {}).get("publication_evidence_sha256")
            == sha256(files["PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json"])
            and committed.get("signed_tx_bytes_sha256")
            == tx["signed_tx_bytes_sha256"]
            and committed.get("expected_transaction_hash")
            == tx["expected_transaction_hash"]
            and committed.get("committed_transaction_hash")
            == tx["expected_transaction_hash"]
            and committed.get("deliver_code") == 0
            and isinstance(committed.get("committed_height"), str)
            and POSITIVE_HEIGHT.fullmatch(committed["committed_height"])
        ):
            fail("CUTOVER initiation evidence is incomplete or mismatched")
        for key in (
            "raw_transaction_query_evidence_sha256",
            "independent_observer_evidence_sha256",
        ):
            require_hash(committed.get(key), f"CUTOVER initiation {key}")
        commit_ns = canonical_nanoseconds(
            committed.get("committed_block_time"), "CUTOVER commit time"
        )
        deadline_epoch = canonical_epoch(deadline, "CUTOVER deadline")
        if commit_ns > deadline_epoch * 1_000_000_000:
            fail("CUTOVER transaction committed after its signed deadline")
        if commit_ns > now * 1_000_000_000:
            fail("CUTOVER transaction is dated in the future")
        if int(committed["committed_height"]) + minimum_lead > int(f):
            fail("CUTOVER transaction committed without the signed halt lead")
        cutover_evidence_created_epoch = canonical_epoch(
            evidence.get("created_at"), "CUTOVER evidence creation time"
        )
        if not (
            notice_epoch * 1_000_000_000
            <= signature_times["CUTOVER-DECISION.json"] * 1_000_000_000
            <= commit_ns
            <= cutover_evidence_created_epoch * 1_000_000_000
            <= cutover_evidence_signature_epoch * 1_000_000_000
        ):
            fail("CUTOVER notice/decision/commit/evidence chronology is non-monotonic")
    return f, a, h


def validate_open_chain(
    files: dict[str, bytes],
    paths: dict[str, pathlib.Path],
    objects: dict[str, Any],
    main: str,
    transition: str,
    final_template: Any,
    open_template: Any,
    adoption_template: Any,
    require_initiation: bool,
    config_policy: pathlib.Path,
    verified_tools: dict[str, bytes],
    temp_path: pathlib.Path,
) -> None:
    f, a, h = validate_cutover_chain(
        files, paths, objects, main, True, config_policy
    )
    # Executing the release-bound helper is deferred until the main-key
    # RELEASE/CUTOVER chain above has authenticated its tool manifest.
    frozen_evidence = load_frozen_evidence_validator(temp_path, verified_tools)
    release = objects["RELEASE-PACKET.json"]
    dark = objects["DARK-START-DECISION.json"]
    cutover = objects["CUTOVER-DECISION.json"]
    cutover_init = objects["CUTOVER-INITIATION-EVIDENCE.json"]
    adoption = objects["ARCHIVE-ADOPTION-AUTHORITY.json"]
    final = objects["FINAL-CHECKPOINT.json"]
    open_beta = objects["OPEN-BETA-DECISION.json"]
    release_transition = release["public_identities"]["transition_attestation"][
        "authorized_signer_fingerprint"
    ]
    if sha256(files["zeroned-zerone-2-release"]) != release.get(
        "components", {}
    ).get("zerone_2_runtime", {}).get("binary_sha256"):
        fail("bundled zerone-2 release binary differs from RELEASE")
    if normalize_fingerprint(release_transition) != normalize_fingerprint(transition):
        fail("expected transition fingerprint differs from RELEASE")
    if normalize_fingerprint(main) == normalize_fingerprint(transition):
        fail("main and transition fingerprints must differ")

    adoption_signature_epoch = verify_signature(
        paths,
        objects,
        "ARCHIVE-ADOPTION-AUTHORITY.json",
        "ARCHIVE-ADOPTION-AUTHORITY.json.sig",
        ("signature_authority",),
        transition,
    )
    final_signature_epoch = verify_signature(
        paths,
        objects,
        "FINAL-CHECKPOINT.json",
        "FINAL-CHECKPOINT.json.sig",
        ("attestation",),
        transition,
    )
    open_signature_epoch = require_main_signature(
        paths,
        objects,
        "OPEN-BETA-DECISION.json",
        "OPEN-BETA-DECISION.json.sig",
        main,
    )
    cutover_evidence_signature_epoch = require_main_signature(
        paths,
        objects,
        "CUTOVER-INITIATION-EVIDENCE.json",
        "CUTOVER-INITIATION-EVIDENCE.json.sig",
        main,
    )
    final_created_epoch = canonical_epoch(
        final.get("created_at"), "FINAL creation time"
    )
    open_created_epoch = canonical_epoch(
        open_beta.get("created_at"), "OPEN-BETA creation time"
    )
    if not (
        cutover_evidence_signature_epoch
        <= adoption_signature_epoch
        <= final_created_epoch
        <= final_signature_epoch
        <= open_created_epoch
        <= open_signature_epoch
    ):
        fail("CUTOVER/adoption/FINAL/OPEN signature chronology is non-monotonic")

    release_pair = exact_pair(files, "RELEASE-PACKET.json", "RELEASE-PACKET.json.sig")
    dark_pair = exact_pair(
        files, "DARK-START-DECISION.json", "DARK-START-DECISION.json.sig"
    )
    dark_init_pair = exact_pair(
        files,
        "DARK-START-INITIATION-EVIDENCE.json",
        "DARK-START-INITIATION-EVIDENCE.json.sig",
    )
    dark_registration_pair = exact_pair(
        files,
        "DARK-REGISTRATION-EVIDENCE.json",
        "DARK-REGISTRATION-EVIDENCE.json.sig",
    )
    cutover_pair = exact_pair(
        files, "CUTOVER-DECISION.json", "CUTOVER-DECISION.json.sig"
    )
    cutover_init_pair = exact_pair(
        files,
        "CUTOVER-INITIATION-EVIDENCE.json",
        "CUTOVER-INITIATION-EVIDENCE.json.sig",
    )
    adoption_pair = exact_pair(
        files,
        "ARCHIVE-ADOPTION-AUTHORITY.json",
        "ARCHIVE-ADOPTION-AUTHORITY.json.sig",
    )
    final_pair = exact_pair(
        files, "FINAL-CHECKPOINT.json", "FINAL-CHECKPOINT.json.sig"
    )

    manifest = objects["zerone-1-archive-transition.json"]
    manifest_keys = {
        "schema",
        "chain_id",
        "checkpoint_state_height",
        "final_committed_height",
        "halt_trigger_height",
        "genesis_sha256",
        "cutover_initiation_evidence",
        "source_observer",
        "candidate",
        "expected_anchor_block_hash",
        "expected_post_anchor_app_hash",
        "source_evidence",
        "archive_construction_evidence",
        "archive_transition_nonce",
    }
    if not isinstance(manifest, dict) or set(manifest) != manifest_keys or not (
        manifest["schema"] == "zerone-1-archive-transition-v1"
        and manifest["chain_id"] == "zerone-1"
        and manifest["checkpoint_state_height"] == f
        and manifest["final_committed_height"] == a
        and manifest["halt_trigger_height"] == h
        and manifest["genesis_sha256"]
        == release.get("predecessor", {}).get("genesis_file_sha256")
    ):
        fail("archive transition manifest root differs from CUTOVER/RELEASE")
    committed = cutover_init["successor_commitment_transaction"]
    notice = cutover_init["public_notice"]
    expected_manifest_initiation = {
        "successor_transaction_hash": committed["committed_transaction_hash"],
        "committed_height": committed["committed_height"],
        "committed_block_time": committed["committed_block_time"],
        "public_notice_sha256": notice["sha256"],
        "public_notice_publication_evidence_sha256": notice[
            "publication_evidence_sha256"
        ],
        "initiation_evidence_sha256": cutover_init_pair["sha256"],
        "initiation_evidence_detached_signature_sha256": cutover_init_pair[
            "detached_signature_sha256"
        ],
    }
    if manifest["cutover_initiation_evidence"] != expected_manifest_initiation:
        fail("archive transition manifest differs from CUTOVER initiation evidence")
    source = manifest["source_observer"]
    candidate = manifest["candidate"]
    if not (
        isinstance(source, dict)
        and set(source) == {"runtime_marker_sha256", "node_id", "validator_pubkey"}
        and isinstance(candidate, dict)
        and set(candidate) == {"node_id", "validator_pubkey"}
        and isinstance(source["node_id"], str)
        and re.fullmatch(r"[0-9a-f]{40}", source["node_id"])
        and isinstance(candidate["node_id"], str)
        and re.fullmatch(r"[0-9a-f]{40}", candidate["node_id"])
        and source["node_id"] != candidate["node_id"]
        and source["validator_pubkey"] != candidate["validator_pubkey"]
    ):
        fail("archive transition identities are malformed or reused")
    require_hash(source["runtime_marker_sha256"], "source observer runtime marker")
    require_hash(
        manifest["expected_anchor_block_hash"], "transition anchor block", upper=True
    )
    require_hash(
        manifest["expected_post_anchor_app_hash"],
        "transition post-anchor AppHash",
        upper=True,
    )
    require_hash(manifest["archive_transition_nonce"], "archive transition nonce")
    if source["runtime_marker_sha256"] != sha256(
        files["SOURCE-OBSERVER-RUNTIME-MARKER"]
    ):
        fail("transition source marker hash differs from the raw marker")
    source_marker = parse_runtime_marker(
        files["SOURCE-OBSERVER-RUNTIME-MARKER"], "source observer runtime marker"
    )
    if source_marker != {
        "runtime_version": "2",
        "role": "observer",
        "chain_id": "zerone-1",
        "genesis_sha256": manifest["genesis_sha256"],
        "node_id": source["node_id"],
        "validator_pubkey": source["validator_pubkey"],
    }:
        fail("source observer runtime marker identity differs from the transition")

    source_evidence = manifest["source_evidence"]
    expected_source_hashes = {
        "signer_manifest_sha256": sha256(files["SIGNER-EVIDENCE-MANIFEST.json"]),
        "observer_manifest_sha256": sha256(
            files["OBSERVER-EVIDENCE-MANIFEST.json"]
        ),
    }
    if source_evidence != expected_source_hashes:
        fail("transition source evidence hashes differ from actual manifests")

    construction = manifest["archive_construction_evidence"]
    expected_construction = {
        "pre_transition_sanitized_snapshot_sha256": sha256(
            files["PRE-TRANSITION-SANITIZED-SNAPSHOT-MANIFEST.json"]
        ),
        "rollback_log_sha256": sha256(files["ARCHIVE-ROLLBACK-LOG.json"]),
        "pre_transition_allowlist_manifest_sha256": sha256(
            files["PRE-TRANSITION-ALLOWLIST-MANIFEST.json"]
        ),
        "excluded_future_artifacts": [
            "archive transition manifest",
            "rendered Fly configs",
            "archive adoption authority",
            "archive readiness",
            "final checkpoint",
            "open-beta decision",
        ],
    }
    if construction != expected_construction:
        fail("transition construction evidence differs from actual pre-transition files")
    allowlist = objects["PRE-TRANSITION-ALLOWLIST-MANIFEST.json"]
    if not (
        allowlist.get("schema") == "zerone-1-pre-transition-allowlist-v1"
        and allowlist.get("chain_id") == "zerone-1"
        and allowlist.get("contains_signer_keys") is False
        and allowlist.get("contains_future_authority_artifacts") is False
        and allowlist.get("result") == "MATCH"
    ):
        fail("pre-transition allowlist semantics are unsafe")
    require_hash(allowlist.get("allowed_entries_sha256"), "allowlist entries")

    same_shape_and_static(adoption, adoption_template, "ARCHIVE-ADOPTION")
    if contains_placeholder(adoption):
        fail("ARCHIVE-ADOPTION retains a placeholder")
    if not (
        adoption.get("schema") == "zerone-1-archive-adoption-authority-v1"
        and adoption.get("attestation_result") == "MATCH"
        and adoption.get("release_packet") == release_pair
        and adoption.get("cutover_decision") == cutover_pair
        and adoption.get("cutover_initiation_evidence")
        == expected_manifest_initiation
        and adoption.get("checkpoint_plan")
        == {
            "checkpoint_state_height": f,
            "final_committed_anchor_height": a,
            "halt_trigger_height": h,
        }
        and adoption.get("archive_transition_manifest", {}).get("sha256")
        == sha256(files["zerone-1-archive-transition.json"])
        and set(adoption.get("deployment_configs", {}))
        == {"zerone_1_archive_candidate", "zerone_1_archive"}
    ):
        fail("archive adoption authority is incomplete or mismatched")
    adoption_manifest = adoption["archive_transition_manifest"]
    for key in (
        "archive_transition_nonce",
        "source_evidence",
        "source_observer",
        "candidate",
        "expected_anchor_block_hash",
        "expected_post_anchor_app_hash",
    ):
        if adoption_manifest.get(key) != manifest.get(key):
            fail(f"archive adoption manifest {key} differs from the inner transition")
    if adoption.get("archive_construction_evidence") != construction:
        fail("archive adoption construction evidence differs from the inner transition")
    adoption_render = adoption.get("render_contract", {})
    cutover_render = cutover["deterministic_private_continuation"]["render_contract"]
    if not (
        adoption_render.get("schema") == cutover_render.get("schema")
        and adoption_render.get("renderer_path") == cutover_render.get("renderer_path")
        and adoption_render.get("renderer_sha256")
        == cutover_render.get("renderer_sha256")
        and adoption_render.get("archive_candidate_template_sha256")
        == cutover_render.get("archive_candidate_template_sha256")
        and adoption_render.get("archive_template_sha256")
        == cutover_render.get("archive_template_sha256")
    ):
        fail("archive adoption render contract differs from CUTOVER")
    invariants = adoption.get("enforced_origin_invariants", {})
    if not (
        invariants.get("public_fly_service") is False
        and invariants.get("public_ip") is False
        and invariants.get("persistent_peers") == []
        and invariants.get("transaction_ingress") is False
    ):
        fail("archive adoption authority does not preserve a private inert origin")
    for key, filename, role, template_hash in (
        (
            "zerone_1_archive_candidate",
            "fly.archive-candidate.toml",
            "archive-candidate",
            cutover_render["archive_candidate_template_sha256"],
        ),
        (
            "zerone_1_archive",
            "fly.archive.toml",
            "archive",
            cutover_render["archive_template_sha256"],
        ),
    ):
        mapping = adoption["deployment_configs"][key]
        if not isinstance(mapping, dict) or set(mapping) != {
            "app",
            "region",
            "deployment_strategy",
            "role",
            "image_component",
            "image_ref",
            "volume",
            "template_sha256",
            "sha256",
        }:
            fail(f"archive adoption {key} mapping is malformed")
        require_hash(mapping.get("sha256"), f"archive adoption {key} config")
        if not (
            mapping["sha256"] == sha256(files[filename])
            and mapping["app"] == "zerone-1-archive"
            and mapping["region"] == "lhr"
            and mapping["deployment_strategy"] == "immediate"
            and mapping["role"] == role
            and mapping["image_component"] == "zerone_1_halt"
            and mapping["image_ref"]
            == release["components"]["zerone_1_halt"]["image_ref"]
            and mapping["volume"] == "zerone_archive_data"
            and mapping["template_sha256"] == template_hash
        ):
            fail(f"archive adoption {key} mapping/config differs from renderer policy")
        try:
            archive_config = tomllib.loads(files[filename].decode())
        except (UnicodeDecodeError, tomllib.TOMLDecodeError) as exc:
            fail(f"archive config {filename} is malformed: {exc}")
        if not (
            archive_config.get("app") == "zerone-1-archive"
            and archive_config.get("build", {}).get("image") == mapping["image_ref"]
            and archive_config.get("deploy") == {"strategy": "immediate"}
            and archive_config.get("env", {}).get("NODE_ROLE") == role
            and archive_config.get("mounts")
            == {"source": "zerone_archive_data", "destination": "/data"}
            and not any(
                name in archive_config for name in ("services", "http_service", "statics")
            )
        ):
            fail(f"archive config {filename} violates the private deterministic shape")

    same_shape_and_static(final, final_template, "FINAL-CHECKPOINT")
    if contains_placeholder(final):
        fail("FINAL-CHECKPOINT retains a placeholder")
    if final.get("schema") != "zerone-final-checkpoint-v4":
        fail("FINAL-CHECKPOINT schema changed")
    authority = final.get("authority_chain", {})
    expected_authority = {
        "release_packet": release_pair,
        "dark_start_decision": dark_pair,
        "dark_start_initiation_evidence": dark_init_pair,
        "dark_registration_evidence": dark_registration_pair,
        "cutover_decision": cutover_pair,
        "cutover_initiation_evidence": cutover_init_pair,
        "archive_adoption_authority": adoption_pair,
        "archive_transition_manifest_sha256": sha256(
            files["zerone-1-archive-transition.json"]
        ),
    }
    if authority != expected_authority:
        fail("FINAL-CHECKPOINT authority chain differs from actual artifacts")
    try:
        frozen_evidence.validate_frozen_evidence(
            files, objects, final, release, manifest, f, a, h
        )
    except frozen_evidence.FrozenEvidenceError as exc:
        fail(f"frozen checkpoint evidence mismatch: {exc}")
    except Exception as exc:
        fail(f"frozen checkpoint evidence validator failed closed: {exc}")
    validate_census_execution_evidence(
        files,
        paths,
        objects,
        release,
        final,
        transition,
        f,
        a,
        cutover_evidence_signature_epoch,
        final_created_epoch,
        final_signature_epoch,
    )
    validate_frozen_terminal_cryptography(
        paths, objects, f, a, h, temp_path
    )
    checkpoint = final["checkpoint_state"]
    application = final["final_application_block"]
    halt = final["halt_trigger_tip"]
    if not (
        checkpoint["height"] == f
        and application["height"] == a
        and halt["height"] == h
        and halt["blockstore_status_tip_before_fencing"] == h
        and application["transaction_count"] == 0
        and halt["transaction_count"] == 0
        and application["commit_canonical"] is True
        and checkpoint["rest_height_pinned"] is True
    ):
        fail("FINAL-CHECKPOINT F/A/H semantics differ from CUTOVER")
    require_hash(checkpoint["app_hash"], "FINAL checkpoint app hash", upper=True)
    if application["header_app_hash"] != checkpoint["app_hash"]:
        fail("FINAL anchor header AppHash differs from checkpoint state")
    if halt["staged_header_last_block_id_hash"] != application["block_id_hash"]:
        fail("FINAL staged halt does not link to the final application block")

    successor = final["successor"]
    if not (
        successor["chain_id"] == "zerone-2"
        and successor["genesis_sha256"] == release.get("genesis", {}).get("sha256")
        and successor["genesis_time"] == release.get("genesis", {}).get("time")
        and successor["release"]["source_commit"]
        == release.get("source", {}).get("commit")
        and successor["release"]["signed_tag"]
        == release.get("source", {}).get("signed_annotated_tag")
        and successor["release"]["tag_signer_fingerprint"]
        == release.get("source", {}).get("tag_signer_fingerprint")
        and successor["release"]["binary_sha256"]
        == release.get("components", {}).get("zerone_2_runtime", {}).get(
            "binary_sha256"
        )
        and successor["release"]["runtime_image_ref"]
        == release.get("components", {}).get("zerone_2_runtime", {}).get(
            "image_ref"
        )
        and successor["release"]["query_gateway_image_ref"]
        == release.get("components", {}).get("query_gateway", {}).get("image_ref")
    ):
        fail("FINAL successor release differs from RELEASE")
    commitment = final["successor_commitment"]
    committed_tx = cutover_init["successor_commitment_transaction"]
    if not (
        commitment["transaction_hash"]
        == committed_tx["committed_transaction_hash"]
        and commitment["committed_height"] == committed_tx["committed_height"]
        and commitment["committed_block_time"]
        == committed_tx["committed_block_time"]
        and commitment["public_notice_sha256"] == cutover["public_notice_sha256"]
        and commitment["public_notice_publication_evidence_sha256"]
        == cutover_init["public_notice"]["publication_evidence_sha256"]
        and commitment["memo"]
        == cutover["successor_commitment_transaction"]["memo"]
    ):
        fail("FINAL successor commitment differs from CUTOVER evidence")
    source_release = final["source_halt_release"]
    if not (
        source_release["source_commit"] == release["source"]["commit"]
        and source_release["signed_tag"] == release["source"]["signed_annotated_tag"]
        and source_release["tag_signer_fingerprint"]
        == release["source"]["tag_signer_fingerprint"]
        and source_release["binary_sha256"]
        == release["components"]["zerone_1_halt"]["binary_sha256"]
        and source_release["image_ref"]
        == release["components"]["zerone_1_halt"]["image_ref"]
    ):
        fail("FINAL source halt release differs from RELEASE")
    config_hashes = final["deployment_config_sha256"]
    if not (
        config_hashes["zerone_1_halt_signer"]
        == cutover["deployment_configs"]["zerone_1_halt_signer"]["sha256"]
        and config_hashes["zerone_1_observer"]
        == cutover["deployment_configs"]["zerone_1_observer"]["sha256"]
        and config_hashes["zerone_1_archive_candidate"]
        == adoption["deployment_configs"]["zerone_1_archive_candidate"]["sha256"]
        and config_hashes["zerone_1_archive"]
        == adoption["deployment_configs"]["zerone_1_archive"]["sha256"]
    ):
        fail("FINAL deployment config hashes differ from signed phase mappings")
    readiness = {
        "candidate_readiness_sha256": sha256(
            files["ARCHIVE-CANDIDATE-READINESS.json"]
        ),
        "final_runtime_marker_sha256": sha256(
            files["ARCHIVE-FINAL-RUNTIME-MARKER"]
        ),
        "private_a_a_probe_evidence_sha256": sha256(
            files["ARCHIVE-PRIVATE-A-A-PROBE-EVIDENCE.json"]
        ),
    }
    candidate_readiness = objects["ARCHIVE-CANDIDATE-READINESS.json"]
    expected_readiness = {
        "schema": "zerone-1-archive-readiness-v2",
        "chain_id": "zerone-1",
        "checkpoint_state_height": f,
        "final_committed_height": a,
        "halt_trigger_height": h,
        "anchor_block_hash": manifest["expected_anchor_block_hash"],
        "post_anchor_app_hash": manifest["expected_post_anchor_app_hash"],
        "halt_trigger_block_absent": True,
        "halt_trigger_results_absent": True,
        "block_sync_catching_up": True,
        "anchor_commit_canonical": False,
        "source_evidence": manifest["source_evidence"],
        "transition_manifest_sha256": sha256(
            files["zerone-1-archive-transition.json"]
        ),
        "archive_transition_nonce": manifest["archive_transition_nonce"],
        "node_id": manifest["candidate"]["node_id"],
        "validator_pubkey": manifest["candidate"]["validator_pubkey"],
    }
    if candidate_readiness != expected_readiness:
        fail("candidate readiness differs from the runtime v2 A/A contract")
    final_marker = parse_runtime_marker(
        files["ARCHIVE-FINAL-RUNTIME-MARKER"], "final archive runtime marker"
    )
    if final_marker != {
        "runtime_version": "2",
        "role": "archive",
        "chain_id": "zerone-1",
        "genesis_sha256": manifest["genesis_sha256"],
        "node_id": manifest["candidate"]["node_id"],
        "validator_pubkey": manifest["candidate"]["validator_pubkey"],
        "archive_transition_nonce": manifest["archive_transition_nonce"],
        "archive_transition_manifest_sha256": sha256(
            files["zerone-1-archive-transition.json"]
        ),
        "archive_readiness_sha256": readiness["candidate_readiness_sha256"],
    }:
        fail("final archive runtime marker does not consume exact readiness")
    probe = objects["ARCHIVE-PRIVATE-A-A-PROBE-EVIDENCE.json"]
    probe_keys = {
        "schema",
        "chain_id",
        "checkpoint_state_height",
        "final_committed_height",
        "halt_trigger_height",
        "anchor_block_hash",
        "post_anchor_app_hash",
        "halt_trigger_block_absent",
        "archive_node_id",
        "transition_manifest_sha256",
        "candidate_readiness_sha256",
        "final_runtime_marker_sha256",
        "status_response_sha256",
        "abci_info_response_sha256",
        "anchor_block_response_sha256",
        "halt_absence_response_sha256",
        "result",
        "observed_at",
    }
    if not isinstance(probe, dict) or set(probe) != probe_keys or not (
        probe["schema"] == "zerone-1-private-aa-probe-evidence-v1"
        and probe["chain_id"] == "zerone-1"
        and probe["checkpoint_state_height"] == f
        and probe["final_committed_height"] == a
        and probe["halt_trigger_height"] == h
        and probe["anchor_block_hash"] == manifest["expected_anchor_block_hash"]
        and probe["post_anchor_app_hash"]
        == manifest["expected_post_anchor_app_hash"]
        and probe["halt_trigger_block_absent"] is True
        and probe["archive_node_id"] == manifest["candidate"]["node_id"]
        and probe["transition_manifest_sha256"]
        == sha256(files["zerone-1-archive-transition.json"])
        and probe["candidate_readiness_sha256"]
        == readiness["candidate_readiness_sha256"]
        and probe["final_runtime_marker_sha256"]
        == readiness["final_runtime_marker_sha256"]
        and probe["result"] == "MATCH"
    ):
        fail("private A/A probe evidence is incomplete or mismatched")
    for key in (
        "status_response_sha256",
        "abci_info_response_sha256",
        "anchor_block_response_sha256",
        "halt_absence_response_sha256",
    ):
        require_hash(probe[key], f"private A/A probe {key}")
    probe_epoch = canonical_epoch(probe["observed_at"], "private A/A probe time")
    if probe_epoch > final_created_epoch:
        fail("FINAL was created before the private A/A probe")
    revalidation = objects["SUCCESSOR-REVALIDATION-EVIDENCE.json"]
    revalidation_keys = {
        "schema",
        "chain_id",
        "release_packet",
        "genesis_sha256",
        "observed_height",
        "observed_block_time",
        "validator_app_hash",
        "edge_app_hash",
        "sdk_validator_count",
        "validator_consensus_pubkey",
        "supply_uzrn",
        "protocol_dark_audit_sha256",
        "operator_onboarding_transaction_hash",
        "custom_validator_registration_transaction_hash",
        "result",
    }
    if not isinstance(revalidation, dict) or set(revalidation) != revalidation_keys or not (
        revalidation["schema"] == "zerone-2-successor-revalidation-evidence-v1"
        and revalidation["chain_id"] == "zerone-2"
        and revalidation["release_packet"] == release_pair
        and revalidation["genesis_sha256"] == release["genesis"]["sha256"]
        and isinstance(revalidation["observed_height"], str)
        and POSITIVE_HEIGHT.fullmatch(revalidation["observed_height"])
        and revalidation["validator_app_hash"] == revalidation["edge_app_hash"]
        and revalidation["sdk_validator_count"] == 1
        and revalidation["validator_consensus_pubkey"]
        == release["public_identities"]["validator_consensus_pubkey"]
        and revalidation["supply_uzrn"] == "13555000000"
        and revalidation["operator_onboarding_transaction_hash"]
        == dark["private_bootstrap_transactions"]["operator_onboarding"][
            "expected_transaction_hash"
        ]
        and revalidation["custom_validator_registration_transaction_hash"]
        == dark["private_bootstrap_transactions"]["custom_validator_registration"][
            "expected_transaction_hash"
        ]
        and revalidation["result"] == "MATCH"
    ):
        fail("successor revalidation evidence is incomplete or mismatched")
    revalidation_ns = canonical_nanoseconds(
        revalidation["observed_block_time"], "successor revalidation block time"
    )
    if not (
        final_signature_epoch * 1_000_000_000
        <= revalidation_ns
        <= open_created_epoch * 1_000_000_000
    ):
        fail("successor revalidation did not occur between FINAL and OPEN")
    require_hash(
        revalidation["validator_app_hash"], "successor revalidation AppHash", upper=True
    )
    require_hash(
        revalidation["protocol_dark_audit_sha256"], "successor protocol-dark audit"
    )
    for key in (
        "operator_onboarding_transaction_hash",
        "custom_validator_registration_transaction_hash",
    ):
        require_hash(revalidation[key], f"successor {key}", upper=True)
    archive = final["archive"]
    if not (
        archive["blockstore_status_height"] == a
        and archive["abci_last_applied_height"] == a
        and archive["fresh_nonvalidator_keys"] is True
        and archive["public_service_authorized"] is False
        and all(archive[key] == value for key, value in readiness.items())
    ):
        fail("FINAL private archive readiness differs from actual evidence")
    archive_app_hash = require_hash(
        final.get("excluded_post_anchor_state", {}).get("app_hash"),
        "FINAL excluded post-anchor AppHash",
        upper=True,
    )
    archive_block_hash = require_hash(
        final.get("final_application_block", {}).get("block_id_hash"),
        "FINAL application block ID",
        upper=True,
    )
    if not (
        archive_app_hash == manifest["expected_post_anchor_app_hash"]
        == probe["post_anchor_app_hash"]
        and archive_block_hash == manifest["expected_anchor_block_hash"]
        == probe["anchor_block_hash"]
    ):
        fail("FINAL archive gateway A/E/B inputs differ from transition evidence")
    archive_app_hash = archive_app_hash.lower()
    archive_block_hash = archive_block_hash.lower()
    verify_archive_gateway_render(
        files, paths, release, verified_tools, temp_path
    )

    same_shape_and_static(open_beta, open_template, "OPEN-BETA")
    if contains_placeholder(open_beta):
        fail("OPEN-BETA retains a placeholder")
    require_main_signature(
        paths,
        objects,
        "OPEN-BETA-DECISION.json",
        "OPEN-BETA-DECISION.json.sig",
        main,
    )
    expected_pairs = {
        "release_packet": release_pair,
        "dark_start_decision": dark_pair,
        "dark_start_initiation_evidence": dark_init_pair,
        "dark_registration_evidence": dark_registration_pair,
        "cutover_decision": cutover_pair,
        "cutover_initiation_evidence": cutover_init_pair,
        "archive_adoption_authority": adoption_pair,
        "final_checkpoint": final_pair,
    }
    if any(open_beta.get(key) != value for key, value in expected_pairs.items()):
        fail("OPEN-BETA authority pairs differ from actual artifacts")
    if open_beta.get("archive_readiness") != readiness:
        fail("OPEN-BETA readiness differs from actual private archive evidence")
    if open_beta.get("successor_revalidation_evidence_sha256") != sha256(
        files["SUCCESSOR-REVALIDATION-EVIDENCE.json"]
    ):
        fail("OPEN-BETA successor revalidation evidence differs from its artifact")
    if open_beta.get("public_notice_sha256") != sha256(files["PUBLIC-NOTICE.md"]):
        fail("OPEN-BETA notice differs from the exact published notice")
    release_configs = release["deployment_configs"]
    for key in ("zerone_2_edge_public", "zerone_2_gateway_public"):
        if open_beta["deployment_configs"].get(key) != release_configs.get(key):
            fail(f"OPEN-BETA {key} mapping differs from RELEASE")
    archive_open_mapping = open_beta["deployment_configs"].get(
        "zerone_1_archive_gateway"
    )
    archive_release_mapping = release_configs["zerone_1_archive_gateway"]
    if not isinstance(archive_open_mapping, dict) or {
        key: value
        for key, value in archive_open_mapping.items()
        if key != "sha256"
    } != archive_release_mapping:
        fail("OPEN-BETA archive gateway static mapping differs from RELEASE")
    public_config_specs = (
        (
            "zerone_2_edge_public",
            "fly.edge.public.toml",
            "-",
            "-",
            "-",
            "-",
        ),
        (
            "zerone_2_gateway_public",
            "fly.zerone-2-gateway.public.toml",
            f"{release_configs['zerone_2_edge_private']['app']}.internal",
            "-",
            "-",
            "-",
        ),
        (
            "zerone_1_archive_gateway",
            "fly.zerone-1-archive-gateway.public.toml",
            f"{release['archive_render_contract']['static_constraints']['app']}.internal",
            a,
            archive_app_hash,
            archive_block_hash,
        ),
    )
    parsed_public_configs: dict[str, dict[str, Any]] = {}
    for (
        key,
        filename,
        upstream,
        archive_height,
        expected_app_hash,
        expected_block_hash,
    ) in public_config_specs:
        parsed_public_configs[key] = validate_config_mapping(
            files,
            filename,
            open_beta["deployment_configs"][key],
            f"OPEN {key}",
        )
        run_config_policy(
            config_policy,
            paths[filename],
            "zerone-2-open-beta-decision-v1",
            key,
            upstream,
            "-",
            archive_height,
            "-",
            expected_app_hash,
            expected_block_hash,
        )
    coordinates = open_beta["public_coordinates"]
    edge_external = parsed_public_configs["zerone_2_edge_public"]["env"][
        "P2P_EXTERNAL_ADDRESS"
    ]
    expected_p2p = (
        f"{release['public_identities']['edge_node_id']}@{edge_external}"
    )
    if coordinates["zerone_2_p2p"] != expected_p2p:
        fail("OPEN public P2P coordinate differs from the edge config/identity")
    dns = objects["DNS-CHANGE-MANIFEST.json"]
    coordinate_values = {
        key: value
        for key, value in coordinates.items()
        if key != "canonical_dns_change_manifest_sha256"
    }
    if not isinstance(dns, dict) or set(dns) != {
        "schema",
        "records",
        "coordinates",
        "result",
    } or not (
        dns["schema"] == "zerone-public-dns-change-manifest-v1"
        and dns["coordinates"] == coordinate_values
        and dns["result"] == "MATCH"
        and coordinates["canonical_dns_change_manifest_sha256"]
        == sha256(files["DNS-CHANGE-MANIFEST.json"])
    ):
        fail("OPEN DNS manifest differs from the signed public coordinates")
    expected_records = {
        edge_external.rsplit(":", 1)[0]: {
            "app": release_configs["zerone_2_edge_public"]["app"],
            "port": 26656,
            "config_sha256": release_configs["zerone_2_edge_public"]["sha256"],
        },
        re.sub(r"^https://", "", coordinates["zerone_2_rpc"]): {
            "app": release_configs["zerone_2_gateway_public"]["app"],
            "https": True,
            "config_sha256": release_configs["zerone_2_gateway_public"]["sha256"],
        },
        re.sub(r"^https://", "", coordinates["zerone_1_archive_rpc"]): {
            "app": archive_open_mapping["app"],
            "https": True,
            "config_sha256": archive_open_mapping["sha256"],
        },
    }
    if dns["records"] != expected_records:
        fail("OPEN DNS records differ from exact apps/config hashes")
    if not (
        coordinates["zerone_2_rest"] == coordinates["zerone_2_rpc"]
        and coordinates["zerone_1_archive_rest"]
        == coordinates["zerone_1_archive_rpc"]
        and all(
            isinstance(coordinates[key], str)
            and coordinates[key].startswith("https://")
            for key in (
                "zerone_2_rpc",
                "zerone_2_rest",
                "zerone_1_archive_rpc",
                "zerone_1_archive_rest",
            )
        )
    ):
        fail("OPEN query coordinates are not exact HTTPS gateway origins")
    tx = open_beta["history_link_transaction"]
    if not (
        tx["chain_id"] == "zerone-2"
        and tx["message_type"] == "/cosmos.bank.v1beta1.MsgSend"
        and tx["recipient_equals_sender"] is True
        and tx["amount"] == "1uzrn"
        and tx["fee"] == "200000uzrn"
        and tx["gas_limit"] == "200000"
        and tx["must_not_reference_open_beta_payload"] is True
        and tx["memo"]
        == f"zerone_1_final_checkpoint_sha256={final_pair['sha256']}"
    ):
        fail("OPEN-BETA history-link contract is incomplete or mismatched")
    require_hash(tx["signed_tx_bytes_sha256"], "OPEN signed TxRaw")
    require_hash(tx["expected_transaction_hash"], "OPEN tx hash", upper=True)
    timeout_height = tx.get("timeout_height")
    if not (
        isinstance(timeout_height, str)
        and POSITIVE_HEIGHT.fullmatch(timeout_height)
        and int(timeout_height) > int(revalidation["observed_height"])
    ):
        fail("OPEN timeout height is not ahead of successor revalidation")
    open_semantics = open_beta["authorization_semantics"]
    deadline = open_semantics["initiation_deadline"]
    deadline_epoch = canonical_epoch(deadline, "OPEN-BETA deadline")
    broadcast_epoch = canonical_epoch(
        open_semantics.get("broadcast_not_after"), "OPEN-BETA broadcast-not-after"
    )
    inclusion_margin = open_semantics.get("minimum_inclusion_margin_seconds")
    if not (
        isinstance(inclusion_margin, int)
        and not isinstance(inclusion_margin, bool)
        and inclusion_margin >= 300
        and deadline_epoch - broadcast_epoch >= inclusion_margin
    ):
        fail("OPEN broadcast cutoff lacks the signed inclusion margin")
    if not require_initiation and int(dt.datetime.now(tz=dt.timezone.utc).timestamp()) > broadcast_epoch:
        fail("OPEN-BETA broadcast cutoff has passed")

    if require_initiation:
        open_evidence_signature_epoch = require_main_signature(
            paths,
            objects,
            "OPEN-BETA-INITIATION-EVIDENCE.json",
            "OPEN-BETA-INITIATION-EVIDENCE.json.sig",
            main,
        )
        evidence = objects["OPEN-BETA-INITIATION-EVIDENCE.json"]
        open_pair = exact_pair(
            files, "OPEN-BETA-DECISION.json", "OPEN-BETA-DECISION.json.sig"
        )
        committed = evidence.get("history_link_transaction", {})
        if not (
            evidence.get("schema") == "zerone-2-open-beta-initiation-evidence-v1"
            and evidence.get("attestation_result") == "MATCH"
            and evidence.get("open_beta_decision") == open_pair
            and evidence.get("initiation_deadline") == deadline
            and evidence.get("deadline_satisfied") is True
            and committed.get("signed_tx_bytes_sha256")
            == tx["signed_tx_bytes_sha256"]
            and committed.get("expected_transaction_hash")
            == tx["expected_transaction_hash"]
            and committed.get("committed_transaction_hash")
            == tx["expected_transaction_hash"]
            and committed.get("deliver_code") == 0
            and isinstance(committed.get("committed_height"), str)
            and POSITIVE_HEIGHT.fullmatch(committed["committed_height"])
        ):
            fail("OPEN-BETA initiation evidence is incomplete or mismatched")
        for key in (
            "raw_transaction_query_evidence_sha256",
            "independent_edge_evidence_sha256",
        ):
            require_hash(committed.get(key), f"OPEN initiation {key}")
        commit_ns = canonical_nanoseconds(
            committed.get("committed_block_time"), "OPEN-BETA commit time"
        )
        deadline_epoch = canonical_epoch(deadline, "OPEN-BETA deadline")
        now = int(dt.datetime.now(tz=dt.timezone.utc).timestamp())
        if commit_ns > deadline_epoch * 1_000_000_000:
            fail("OPEN-BETA transaction committed after its signed deadline")
        if commit_ns > now * 1_000_000_000:
            fail("OPEN-BETA transaction is dated in the future")
        if int(committed["committed_height"]) > int(timeout_height):
            fail("OPEN-BETA transaction committed after its timeout height")
        open_evidence_created_epoch = canonical_epoch(
            evidence.get("created_at"), "OPEN-BETA evidence creation time"
        )
        if not (
            open_signature_epoch * 1_000_000_000
            <= commit_ns
            <= open_evidence_created_epoch * 1_000_000_000
            <= open_evidence_signature_epoch * 1_000_000_000
        ):
            fail("OPEN decision/commit/evidence chronology is non-monotonic")


def main() -> None:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument(
        "stage",
        choices=(
            "dark-preinit",
            "dark-registration-preinit",
            "notice-prepublish",
            "cutover-preinit",
            "cutover-postinit",
            "open-preinit",
            "open-postinit",
        ),
    )
    parser.add_argument("bundle")
    parser.add_argument("expected_main")
    parser.add_argument("expected_transition", nargs="?", default=None)
    parser.add_argument("--release", required=True)
    parser.add_argument("--release-sig", required=True)
    parser.add_argument("--decision", required=True)
    parser.add_argument("--decision-sig", required=True)
    parser.add_argument("--initiation")
    parser.add_argument("--initiation-sig")
    parser.add_argument("--final")
    parser.add_argument("--final-sig")
    parser.add_argument("--final-template")
    parser.add_argument("--open-template")
    parser.add_argument("--adoption-template")
    parser.add_argument("--config-policy", required=True)
    parser.add_argument("--tool-root", required=True)
    args = parser.parse_args()
    if not FINGERPRINT.fullmatch(args.expected_main):
        fail("expected main signer must be a full 40- or 64-hex fingerprint")
    if args.stage.startswith("open"):
        if args.expected_transition is None or not FINGERPRINT.fullmatch(
            args.expected_transition
        ):
            fail("OPEN verification requires a full transition fingerprint")
        if not args.final_template or not args.open_template or not args.adoption_template:
            fail("OPEN verification requires adoption, FINAL, and OPEN templates")
        if not args.final or not args.final_sig:
            fail("OPEN verification requires the explicit FINAL payload/signature pair")
    elif args.expected_transition is not None:
        fail("transition fingerprint is valid only for OPEN verification")

    bundle_path = pathlib.Path(args.bundle)
    try:
        bundle_info = os.lstat(bundle_path)
    except OSError as exc:
        fail(f"could not inspect authority bundle: {exc}")
    if not stat.S_ISDIR(bundle_info.st_mode) or stat.S_ISLNK(bundle_info.st_mode):
        fail("authority bundle must be a real directory, not a symlink")
    bundle_fd = os.open(
        bundle_path,
        os.O_RDONLY | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_NOFOLLOW", 0),
    )
    base = {
        "RELEASE-PACKET.json",
        "RELEASE-PACKET.json.sig",
        "DARK-START-DECISION.json",
        "DARK-START-DECISION.json.sig",
        "ZERONE-2-ONBOARD-SIGNED-TX.json",
        "ZERONE-2-CUSTOM-VALIDATOR-SIGNED-TX.json",
        "OPERATOR-TOOL-MANIFEST.json",
        CENSUS_BINARY_FILENAME,
        COMPONENT_SIGNATURE_VERIFIER_FILE,
        SIGSTORE_TRUSTED_ROOT_FILE,
        "zeroned-zerone-1-release",
        "genesis.json",
        "genesis.sha256",
        "network-manifest.json",
        "GENESIS-MANIFEST.md",
        "zeroned-zerone-2-release",
        *MONITORING_ARTIFACT_FILES.values(),
        *MONITORING_EVIDENCE_FILENAMES,
    }
    for component_files in COMPONENT_ARTIFACT_FILES.values():
        base.update(component_files.values())
    dark_post = {
        "DARK-START-INITIATION-EVIDENCE.json",
        "DARK-START-INITIATION-EVIDENCE.json.sig",
    }
    notice_files = {
        "DARK-REGISTRATION-EVIDENCE.json",
        "DARK-REGISTRATION-EVIDENCE.json.sig",
        "PRIVATE-SOAK-EVIDENCE.json",
        "HALT-REHEARSAL-EVIDENCE.json",
        "PUBLIC-NOTICE.md",
        "PRE-NOTICE-DECISION.json",
        "PRE-NOTICE-DECISION.json.sig",
    }
    cutover_files = {
        "PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json",
        "PUBLIC-NOTICE-CAPTURE.md",
        "fly.halt-signer.toml",
        "fly.observer.toml",
        "CUTOVER-SIGNED-TX.json",
        "CUTOVER-DECISION.json",
        "CUTOVER-DECISION.json.sig",
    }
    cutover_post = {
        "CUTOVER-INITIATION-EVIDENCE.json",
        "CUTOVER-INITIATION-EVIDENCE.json.sig",
    }
    open_files = {
        "zerone-1-archive-transition.json",
        "SIGNER-EVIDENCE-MANIFEST.json",
        "OBSERVER-EVIDENCE-MANIFEST.json",
        "SOURCE-OBSERVER-RUNTIME-MARKER",
        "PRE-TRANSITION-SANITIZED-SNAPSHOT-MANIFEST.json",
        "ARCHIVE-ROLLBACK-LOG.json",
        "PRE-TRANSITION-ALLOWLIST-MANIFEST.json",
        "ARCHIVE-ADOPTION-AUTHORITY.json",
        "ARCHIVE-ADOPTION-AUTHORITY.json.sig",
        "ARCHIVE-CANDIDATE-READINESS.json",
        "ARCHIVE-FINAL-RUNTIME-MARKER",
        "ARCHIVE-PRIVATE-A-A-PROBE-EVIDENCE.json",
        "FINAL-CHECKPOINT.json",
        "FINAL-CHECKPOINT.json.sig",
        "SUCCESSOR-REVALIDATION-EVIDENCE.json",
        "DNS-CHANGE-MANIFEST.json",
        "OPEN-BETA-DECISION.json",
        "OPEN-BETA-DECISION.json.sig",
        "fly.edge.public.toml",
        "fly.zerone-2-gateway.public.toml",
        "fly.zerone-1-archive-gateway.public.toml",
        "fly.archive-candidate.toml",
        "fly.archive.toml",
        "OPEN-BETA-SIGNED-TX.json",
        "zeroned-zerone-2-release",
    }
    open_files.update(FROZEN_EVIDENCE_FILES)
    open_post = {
        "OPEN-BETA-INITIATION-EVIDENCE.json",
        "OPEN-BETA-INITIATION-EVIDENCE.json.sig",
    }
    required = set(base)
    if args.stage != "dark-preinit":
        required |= dark_post
    if args.stage not in {"dark-preinit", "dark-registration-preinit"}:
        required |= notice_files
    if args.stage not in {"dark-preinit", "dark-registration-preinit", "notice-prepublish"}:
        required |= cutover_files
    if args.stage in {"cutover-postinit", "open-preinit", "open-postinit"}:
        required |= cutover_post
    if args.stage in {"open-preinit", "open-postinit"}:
        required |= open_files
    if args.stage == "open-postinit":
        required |= open_post
    try:
        files: dict[str, bytes] = {}
        total_bundle_bytes = 0
        for name in sorted(required):
            files[name] = secure_read_bundle(bundle_fd, name)
            total_bundle_bytes += len(files[name])
            if total_bundle_bytes > 1024 * 1024 * 1024:
                fail("required authority bundle exceeds the 1 GiB pre-authentication limit")
    finally:
        os.close(bundle_fd)

    explicit = {
        "RELEASE-PACKET.json": secure_read_path(
            pathlib.Path(args.release), "explicit release packet"
        ),
        "RELEASE-PACKET.json.sig": secure_read_path(
            pathlib.Path(args.release_sig), "explicit release signature"
        ),
    }
    decision_name = (
        "DARK-START-DECISION.json"
        if args.stage in {"dark-preinit", "dark-registration-preinit"}
        else "PRE-NOTICE-DECISION.json"
        if args.stage == "notice-prepublish"
        else "CUTOVER-DECISION.json"
        if args.stage.startswith("cutover")
        else "OPEN-BETA-DECISION.json"
    )
    explicit[decision_name] = secure_read_path(
        pathlib.Path(args.decision), "explicit phase decision"
    )
    explicit[f"{decision_name}.sig"] = secure_read_path(
        pathlib.Path(args.decision_sig), "explicit phase decision signature"
    )
    if args.stage.startswith("open"):
        explicit["FINAL-CHECKPOINT.json"] = secure_read_path(
            pathlib.Path(args.final), "explicit FINAL checkpoint"
        )
        explicit["FINAL-CHECKPOINT.json.sig"] = secure_read_path(
            pathlib.Path(args.final_sig), "explicit FINAL signature"
        )
    elif args.final or args.final_sig:
        fail("FINAL arguments are valid only for OPEN verification")
    initiation_name = (
        "DARK-START-INITIATION-EVIDENCE.json"
        if args.stage == "dark-registration-preinit"
        else "CUTOVER-INITIATION-EVIDENCE.json"
        if args.stage == "cutover-postinit"
        else "OPEN-BETA-INITIATION-EVIDENCE.json"
        if args.stage == "open-postinit"
        else None
    )
    if initiation_name:
        if not args.initiation or not args.initiation_sig:
            fail("this verification stage requires explicit initiation evidence and signature")
        explicit[initiation_name] = secure_read_path(
            pathlib.Path(args.initiation), "explicit initiation evidence"
        )
        explicit[f"{initiation_name}.sig"] = secure_read_path(
            pathlib.Path(args.initiation_sig), "explicit initiation signature"
        )
    elif args.initiation or args.initiation_sig:
        fail("pre-initiation verification must not receive initiation evidence")
    for name, data in explicit.items():
        if files.get(name) != data:
            fail(f"explicit {name} differs from the authority bundle")

    config_policy_bytes = secure_read_path(
        pathlib.Path(args.config_policy), "structural config policy"
    )

    json_names = {
        name
        for name in required
        if name.endswith(".json") and not name.endswith(".sig")
    }
    objects: dict[str, Any] = {}
    with tempfile.TemporaryDirectory(prefix="zerone-authority-chain.") as temp:
        temp_path = pathlib.Path(temp)
        config_policy_path = temp_path / "validate-fly-phase-config.py"
        paths: dict[str, pathlib.Path] = {}
        for name, data in files.items():
            destination = temp_path / name
            destination.write_bytes(data)
            os.chmod(destination, 0o600)
            paths[name] = destination
        canonical_json_names = {
            "RELEASE-PACKET.json",
            "DARK-START-DECISION.json",
            "DARK-START-INITIATION-EVIDENCE.json",
            *MONITORING_ARTIFACT_FILES.values(),
            "DARK-REGISTRATION-EVIDENCE.json",
            "PRE-NOTICE-DECISION.json",
            "PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json",
            "CUTOVER-DECISION.json",
            "CUTOVER-INITIATION-EVIDENCE.json",
            "zerone-1-archive-transition.json",
            "ARCHIVE-ADOPTION-AUTHORITY.json",
            CENSUS_EXECUTION_EVIDENCE_FILENAME,
            "FINAL-CHECKPOINT.json",
            "OPEN-BETA-DECISION.json",
            "OPEN-BETA-INITIATION-EVIDENCE.json",
        }
        canonical_json_names.update(
            filename
            for component_files in COMPONENT_ARTIFACT_FILES.values()
            for filename in component_files.values()
        )
        for name in json_names:
            objects[name] = parse_json(files[name], name)
            if name in canonical_json_names:
                verify_canonical(paths[name], files[name], name)
            if contains_placeholder(objects[name]):
                fail(f"{name} retains a placeholder")
        verified_tools = validate_tool_manifest(
            files, objects, pathlib.Path(args.tool_root)
        )

        release_config_policy = verified_tools.get(
            "deploy/validate-fly-phase-config.py"
        )
        if (
            release_config_policy is None
            or config_policy_bytes != release_config_policy
        ):
            fail("explicit structural config policy differs from RELEASE")
        config_policy_path.write_bytes(release_config_policy)
        os.chmod(config_policy_path, 0o700)

        if args.stage == "dark-preinit":
            validate_cutover_chain(
                files,
                paths,
                objects,
                args.expected_main,
                False,
                config_policy_path,
                dark_preinit=True,
            )
        elif args.stage == "dark-registration-preinit":
            validate_cutover_chain(
                files,
                paths,
                objects,
                args.expected_main,
                False,
                config_policy_path,
                dark_registration_only=True,
            )
        elif args.stage == "notice-prepublish":
            validate_cutover_chain(
                files,
                paths,
                objects,
                args.expected_main,
                False,
                config_policy_path,
                notice_prepublish=True,
            )
        elif args.stage.startswith("cutover"):
            validate_cutover_chain(
                files,
                paths,
                objects,
                args.expected_main,
                args.stage == "cutover-postinit",
                config_policy_path,
            )
        else:
            template_specs = (
                (
                    args.final_template,
                    "deploy/networks/zerone-1/frozen/FINAL-CHECKPOINT.example.json",
                    "FINAL template",
                ),
                (
                    args.open_template,
                    "deploy/networks/zerone-2/OPEN-BETA-DECISION.example.json",
                    "OPEN template",
                ),
                (
                    args.adoption_template,
                    "deploy/networks/zerone-2/ARCHIVE-ADOPTION-AUTHORITY.example.json",
                    "archive adoption template",
                ),
            )
            parsed_templates: list[Any] = []
            for explicit_path, release_path, label in template_specs:
                explicit_bytes = secure_read_path(pathlib.Path(explicit_path), label)
                release_bytes = verified_tools.get(release_path)
                if release_bytes is None or explicit_bytes != release_bytes:
                    fail(f"explicit {label} differs from RELEASE")
                parsed_templates.append(parse_json(release_bytes, label))
            final_template, open_template, adoption_template = parsed_templates
            validate_open_chain(
                files,
                paths,
                objects,
                args.expected_main,
                args.expected_transition,
                final_template,
                open_template,
                adoption_template,
                args.stage == "open-postinit",
                config_policy_path,
                verified_tools,
                temp_path,
            )

    if args.stage == "notice-prepublish":
        print("authority-chain: MATCH (pre-notice publication only)")
    else:
        print("authority-chain: MATCH")


if __name__ == "__main__":
    if shutil.which("jq") is None:
        fail("jq is required")
    if shutil.which("gpg") is None:
        fail("gpg is required")
    main()
