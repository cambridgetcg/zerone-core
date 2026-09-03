#!/usr/bin/env python3
"""Validate the raw zerone-1 frozen-checkpoint evidence bundle.

This module deliberately validates byte-bearing evidence that does not belong in
the signed authority templates themselves: the v3 inventory, the two terminal
RPC captures, the excluded post-anchor export, and database/rollback manifests.
It never opens paths or contacts a node.  Callers must supply bytes already read
through their own no-symlink boundary.
"""

from __future__ import annotations

import base64
import binascii
import datetime as dt
import hashlib
import json
import re
import urllib.parse
from collections.abc import Mapping
from typing import Any


LOWER_HASH = re.compile(r"^[0-9a-f]{64}$")
UPPER_HASH = re.compile(r"^[0-9A-F]{64}$")
NODE_ID = re.compile(r"^[0-9a-f]{40}$")
HEIGHT = re.compile(r"^[1-9][0-9]{0,17}$")
DECIMAL = re.compile(r"^(?:0|[1-9][0-9]*)$")
RFC3339_NANO = re.compile(
    r"^[0-9]{4}-[0-9]{2}-[0-9]{2}T"
    r"[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,9})?Z$"
)

CENSUS_MAX_CUSTOM_STORE_LEAVES = 50_000
CENSUS_MAX_STORE_LEAVES = 5_000_000
CENSUS_MAX_CUSTOM_INPUT_BYTES = 32 << 20
CENSUS_MAX_SCANNED_INPUT_BYTES = 1 << 30
CENSUS_MAX_SDK_VALIDATORS = 25_000

RPC_PAYLOAD_SUFFIXES = {
    "status_json": "STATUS.json.raw",
    "genesis_json": "GENESIS.json.raw",
    "trusted_block_json": "TRUSTED-BLOCK.json.raw",
    "trusted_commit_json": "TRUSTED-COMMIT.json.raw",
    "trusted_validators_json": "TRUSTED-VALIDATORS.json.raw",
    "block_a_json": "BLOCK-A.json.raw",
    "commit_a_json": "COMMIT-A.json.raw",
    "validators_a_json": "VALIDATORS-A.json.raw",
    "block_results_a_json": "BLOCK-RESULTS-A.json.raw",
    "block_h_json": "BLOCK-H.json.raw",
    "commit_h_json": "COMMIT-H.json.raw",
    "validators_h_json": "VALIDATORS-H.json.raw",
    "abci_info_json": "ABCI-INFO.json.raw",
    "block_results_h_missing_response": "BLOCK-RESULTS-H-MISSING.json.raw",
}
MATCHING_PAYLOAD_KEYS = tuple(key for key in RPC_PAYLOAD_SUFFIXES if key != "status_json")
MATCHED_PAYLOAD_LABELS = [
    "genesis",
    "trusted_block",
    "trusted_commit",
    "trusted_validators",
    "block_A",
    "commit_A",
    "validators_A",
    "block_results_A",
    "block_H",
    "commit_H",
    "validators_H",
    "abci_info",
    "block_results_H_missing",
]


def _rpc_filename(prefix: str, key: str) -> str:
    return f"{prefix}-RPC-{RPC_PAYLOAD_SUFFIXES[key]}"


REQUIRED_FROZEN_EVIDENCE_FILES = frozenset(
    {
        "CUSTOM-STAKING-CENSUS.json",
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
    | {
        _rpc_filename(prefix, key)
        for prefix in ("SIGNER", "OBSERVER")
        for key in RPC_PAYLOAD_SUFFIXES
    }
)


class FrozenEvidenceError(ValueError):
    """The supplied frozen evidence does not prove the required checkpoint."""


def _fail(message: str) -> None:
    raise FrozenEvidenceError(message)


def _sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


BECH32_CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
BECH32_VALUES = {character: index for index, character in enumerate(BECH32_CHARSET)}


def _bech32_polymod(values: list[int]) -> int:
    checksum = 1
    generators = (0x3B6A57B2, 0x26508E6D, 0x1EA119FA, 0x3D4233DD, 0x2A1462B3)
    for value in values:
        top = checksum >> 25
        checksum = ((checksum & 0x1FFFFFF) << 5) ^ value
        for index, generator in enumerate(generators):
            if (top >> index) & 1:
                checksum ^= generator
    return checksum


def _bech32_hrp_expand(hrp: str) -> list[int]:
    return [ord(character) >> 5 for character in hrp] + [0] + [
        ord(character) & 31 for character in hrp
    ]


def _convert_bits(
    values: bytes | list[int], from_bits: int, to_bits: int, pad: bool
) -> list[int] | None:
    accumulator = 0
    bit_count = 0
    result: list[int] = []
    maximum = (1 << to_bits) - 1
    for value in values:
        if value < 0 or value >> from_bits:
            return None
        accumulator = (accumulator << from_bits) | value
        bit_count += from_bits
        while bit_count >= to_bits:
            bit_count -= to_bits
            result.append((accumulator >> bit_count) & maximum)
    if pad:
        if bit_count:
            result.append((accumulator << (to_bits - bit_count)) & maximum)
    elif bit_count >= from_bits or ((accumulator << (to_bits - bit_count)) & maximum):
        return None
    return result


def _bech32_encode(hrp: str, payload: bytes) -> str:
    data = _convert_bits(payload, 8, 5, True)
    if data is None:  # pragma: no cover - bytes always fit an eight-bit source word.
        raise AssertionError("could not convert bytes to Bech32 words")
    values = _bech32_hrp_expand(hrp) + data
    polymod = _bech32_polymod(values + [0] * 6) ^ 1
    checksum = [(polymod >> (5 * (5 - index))) & 31 for index in range(6)]
    return hrp + "1" + "".join(BECH32_CHARSET[value] for value in data + checksum)


def _bech32_address(value: Any, hrp: str, label: str) -> bytes:
    if (
        not isinstance(value, str)
        or value != value.lower()
        or not 8 <= len(value) <= 1023
        or value.rfind("1") != len(hrp)
        or value[: len(hrp)] != hrp
    ):
        _fail(f"{label} is not canonical {hrp} Bech32")
    try:
        data = [BECH32_VALUES[character] for character in value[len(hrp) + 1 :]]
    except KeyError:
        _fail(f"{label} is not canonical {hrp} Bech32")
    if len(data) < 6 or _bech32_polymod(_bech32_hrp_expand(hrp) + data) != 1:
        _fail(f"{label} has an invalid Bech32 checksum")
    decoded = _convert_bits(data[:-6], 5, 8, False)
    if decoded is None:
        _fail(f"{label} has invalid Bech32 padding")
    raw = bytes(decoded)
    if not raw or len(raw) > 255 or _bech32_encode(hrp, raw) != value:
        _fail(f"{label} is not a canonical SDK address")
    return raw


def _exact_object(value: Any, keys: set[str], label: str) -> dict[str, Any]:
    if not isinstance(value, dict) or set(value) != keys:
        _fail(f"{label} does not have the exact required fields")
    return value


def _string(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value or value.strip() != value:
        _fail(f"{label} must be a non-empty whitespace-free string")
    try:
        value.encode("utf-8")
    except UnicodeEncodeError:
        _fail(f"{label} is not valid UTF-8")
    return value


def _lower_hash(value: Any, label: str) -> str:
    if not isinstance(value, str) or not LOWER_HASH.fullmatch(value):
        _fail(f"{label} is not a lowercase SHA-256")
    return value


def _upper_hash(value: Any, label: str) -> str:
    if not isinstance(value, str) or not UPPER_HASH.fullmatch(value):
        _fail(f"{label} is not an uppercase 32-byte hash")
    return value


def _height(value: Any, label: str) -> str:
    if not isinstance(value, str) or not HEIGHT.fullmatch(value):
        _fail(f"{label} is not a canonical positive height")
    return value


def _integer(value: Any, label: str, minimum: int = 0) -> int:
    if not isinstance(value, int) or isinstance(value, bool) or value < minimum:
        _fail(f"{label} is not an integer >= {minimum}")
    return value


def _decimal(value: Any, label: str, positive: bool = False) -> int:
    if (
        not isinstance(value, str)
        or len(value) > 256
        or not DECIMAL.fullmatch(value)
    ):
        _fail(f"{label} is not canonical decimal")
    try:
        number = int(value)
    except ValueError:
        _fail(f"{label} is outside the supported integer range")
    if positive and number <= 0:
        _fail(f"{label} must be positive")
    return number


def _node_id(value: Any, label: str) -> str:
    if not isinstance(value, str) or not NODE_ID.fullmatch(value):
        _fail(f"{label} is not a lowercase 20-byte node ID")
    return value


def _public_key(value: Any, label: str, length: int | None = 32) -> bytes:
    if not isinstance(value, str) or not value:
        _fail(f"{label} is not canonical base64")
    try:
        decoded = base64.b64decode(value, validate=True)
    except (binascii.Error, ValueError):
        _fail(f"{label} is not canonical base64")
    if base64.b64encode(decoded).decode() != value:
        _fail(f"{label} is not canonical base64")
    if not decoded or (length is not None and len(decoded) != length):
        expected = f"exactly {length}" if length is not None else "at least one"
        _fail(f"{label} must decode to {expected} byte(s)")
    return decoded


def _rfc3339_nano(value: Any, label: str) -> str:
    if not isinstance(value, str) or not RFC3339_NANO.fullmatch(value):
        _fail(f"{label} is not canonical UTC RFC3339Nano")
    base, _, fraction = value[:-1].partition(".")
    try:
        parsed = dt.datetime.strptime(base, "%Y-%m-%dT%H:%M:%S")
    except ValueError as exc:
        _fail(f"{label} is not a real UTC time: {exc}")
    if parsed.strftime("%Y-%m-%dT%H:%M:%S") != base:
        _fail(f"{label} is not canonical UTC RFC3339Nano")
    if fraction and fraction.endswith("0"):
        _fail(f"{label} has non-canonical trailing fractional zeros")
    return value


def _rfc3339_nanoseconds(value: Any, label: str) -> int:
    canonical = _rfc3339_nano(value, label)
    base, _, fraction = canonical[:-1].partition(".")
    parsed = dt.datetime.strptime(base, "%Y-%m-%dT%H:%M:%S").replace(
        tzinfo=dt.timezone.utc
    )
    return int(parsed.timestamp()) * 1_000_000_000 + int(
        fraction.ljust(9, "0") or "0"
    )


def _parse_json(data: bytes, label: str) -> Any:
    def reject_duplicates(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
        result: dict[str, Any] = {}
        for key, value in pairs:
            if key in result:
                raise ValueError(f"duplicate key {key!r}")
            result[key] = value
        return result

    try:
        return json.loads(data, object_pairs_hook=reject_duplicates)
    except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
        _fail(f"{label} is not one duplicate-free JSON value: {exc}")


def _object(
    files: Mapping[str, bytes], objects: Mapping[str, Any], name: str
) -> dict[str, Any]:
    if name not in files:
        _fail(f"required frozen evidence file {name} is missing")
    value = objects.get(name)
    if value is None:
        value = _parse_json(files[name], name)
    if not isinstance(value, dict):
        _fail(f"{name} must contain a JSON object")
    return value


def _path(value: Any, label: str, *keys: str) -> Any:
    current = value
    for key in keys:
        if not isinstance(current, dict) or key not in current:
            _fail(f"{label} is missing {'.'.join(keys)}")
        current = current[key]
    return current


def _normalize_app_hash(value: Any, label: str) -> str:
    if isinstance(value, str) and re.fullmatch(r"[0-9A-Fa-f]{64}", value):
        return value.upper()
    if isinstance(value, str):
        try:
            decoded = base64.b64decode(value, validate=True)
        except (binascii.Error, ValueError):
            decoded = b""
        if len(decoded) == 32 and base64.b64encode(decoded).decode() == value:
            return decoded.hex().upper()
    _fail(f"{label} is neither a 32-byte hex nor canonical base64 app hash")


def _validate_url(value: Any, label: str) -> None:
    raw = _string(value, label)
    try:
        parsed = urllib.parse.urlsplit(raw)
    except ValueError as exc:
        _fail(f"{label} is not a valid URL: {exc}")
    if (
        parsed.scheme not in {"http", "https"}
        or not parsed.netloc
        or parsed.username is not None
        or parsed.password is not None
        or parsed.query
        or parsed.fragment
    ):
        _fail(f"{label} is not a credential-free HTTP(S) base URL")


INVENTORY_SOURCE_KEYS = {
    "chain_id",
    "checkpoint_state_height",
    "checkpoint_app_hash",
    "final_committed_block_height",
    "final_committed_block_hash",
    "final_committed_block_time",
    "final_committed_block_txs",
    "final_committed_block_canonical",
    "final_committed_block_has_results",
    "halt_trigger_height",
    "rpc_blockstore_height",
    "staged_halt_trigger_block_hash",
    "staged_halt_trigger_block_time",
    "staged_halt_trigger_block_txs",
    "staged_halt_trigger_previous_block_hash",
    "staged_halt_trigger_header_app_hash",
    "staged_halt_trigger_commit_canonical",
    "staged_halt_trigger_has_block_results",
    "abci_last_applied_height",
    "excluded_post_anchor_app_hash",
    "rpc_genesis_canonical_sha256",
    "declared_genesis_file_sha256",
    "rest_trust_model",
    "rpc",
    "rest",
}
INVENTORY_TRUST_MODEL = (
    "trusted height-pinned REST responses; no Merkle proof binds inventory "
    "to checkpoint_app_hash"
)
FINAL_TRUST_MODEL = (
    "height-pinned REST from a trusted halted source; responses are not "
    "Merkle-proved"
)


def _validate_inventory(
    inventory: dict[str, Any], release: Mapping[str, Any], f: str, a: str, h: str
) -> dict[str, Any]:
    _exact_object(
        inventory,
        {"schema", "source", "denom", "supply_uzrn", "owners", "bonded_validators"},
        "inventory v3",
    )
    if inventory["schema"] != "zerone-relaunch-snapshot-v3":
        _fail("inventory is not zerone-relaunch-snapshot-v3")
    source = _exact_object(inventory["source"], INVENTORY_SOURCE_KEYS, "inventory source")
    predecessor = release.get("predecessor")
    if not isinstance(predecessor, dict):
        _fail("RELEASE predecessor declaration is missing")
    for field in (
        "checkpoint_state_height",
        "final_committed_block_height",
        "halt_trigger_height",
        "rpc_blockstore_height",
        "abci_last_applied_height",
    ):
        _integer(source[field], f"inventory source {field}", 1)
    for field in (
        "final_committed_block_txs",
        "staged_halt_trigger_block_txs",
    ):
        _integer(source[field], f"inventory source {field}")
    if not (
        source["chain_id"] == "zerone-1"
        and source["checkpoint_state_height"] == int(f)
        and source["final_committed_block_height"] == int(a)
        and source["halt_trigger_height"] == int(h)
        and source["rpc_blockstore_height"] == int(h)
        and source["abci_last_applied_height"] == int(a)
        and source["final_committed_block_txs"] == 0
        and source["final_committed_block_canonical"] is True
        and source["final_committed_block_has_results"] is True
        and source["staged_halt_trigger_block_txs"] == 0
        and source["staged_halt_trigger_commit_canonical"] is False
        and source["staged_halt_trigger_has_block_results"] is False
        and source["rest_trust_model"] == INVENTORY_TRUST_MODEL
        and source["declared_genesis_file_sha256"]
        == predecessor.get("genesis_file_sha256")
    ):
        _fail("inventory source boundary differs from RELEASE/F/A/H")
    checkpoint_hash = _upper_hash(source["checkpoint_app_hash"], "inventory checkpoint AppHash")
    anchor_hash = _upper_hash(
        source["final_committed_block_hash"], "inventory anchor block hash"
    )
    halt_hash = _upper_hash(
        source["staged_halt_trigger_block_hash"], "inventory halt-trigger block hash"
    )
    post_anchor_hash = _upper_hash(
        source["excluded_post_anchor_app_hash"], "inventory post-anchor AppHash"
    )
    _upper_hash(
        source["staged_halt_trigger_previous_block_hash"],
        "inventory staged previous block hash",
    )
    staged_header_hash = _upper_hash(
        source["staged_halt_trigger_header_app_hash"],
        "inventory staged header AppHash",
    )
    if source["staged_halt_trigger_previous_block_hash"] != anchor_hash:
        _fail("inventory staged halt trigger does not link to anchor A")
    if staged_header_hash != post_anchor_hash:
        _fail("inventory staged H header does not commit the post-anchor A state")
    if checkpoint_hash == post_anchor_hash:
        _fail("inventory collapses checkpoint-F state and excluded post-anchor-A state")
    anchor_time = _rfc3339_nano(
        source["final_committed_block_time"], "inventory anchor time"
    )
    halt_time = _rfc3339_nano(
        source["staged_halt_trigger_block_time"], "inventory halt-trigger time"
    )
    if _rfc3339_nanoseconds(
        halt_time, "inventory halt-trigger time"
    ) <= _rfc3339_nanoseconds(anchor_time, "inventory anchor time"):
        _fail("inventory halt-trigger time is not after anchor time")
    rpc_genesis_hash = _lower_hash(
        source["rpc_genesis_canonical_sha256"], "inventory RPC genesis hash"
    )
    _lower_hash(
        source["declared_genesis_file_sha256"], "inventory raw genesis hash"
    )
    _validate_url(source["rpc"], "inventory RPC URL")
    _validate_url(source["rest"], "inventory REST URL")

    if inventory["denom"] != "uzrn":
        _fail("inventory denomination is not uzrn")
    supply = _decimal(inventory["supply_uzrn"], "inventory supply")
    owners = inventory["owners"]
    if not isinstance(owners, list):
        _fail("inventory owners is not a list")
    owner_total = 0
    owner_addresses: list[str] = []
    for index, owner in enumerate(owners):
        label = f"inventory owner[{index}]"
        if not isinstance(owner, dict) or set(owner) not in (
            {"address", "account_type", "amount_uzrn"},
            {"address", "account_type", "module_name", "amount_uzrn"},
        ):
            _fail(f"{label} does not have the exact v3 fields")
        owner_addresses.append(_string(owner["address"], f"{label}.address"))
        _string(owner["account_type"], f"{label}.account_type")
        if "module_name" in owner:
            _string(owner["module_name"], f"{label}.module_name")
        owner_total += _decimal(owner["amount_uzrn"], f"{label}.amount", positive=True)
    if owner_addresses != sorted(owner_addresses) or len(set(owner_addresses)) != len(
        owner_addresses
    ):
        _fail("inventory owners are not uniquely sorted by address")
    if owner_total != supply:
        _fail("inventory owner balances do not sum to supply")

    validators = inventory["bonded_validators"]
    if not isinstance(validators, list) or not validators:
        _fail("inventory bonded_validators must be a non-empty list")
    validator_operators: list[str] = []
    validator_keys: list[str] = []
    for index, validator in enumerate(validators):
        label = f"inventory validator[{index}]"
        validator = _exact_object(
            validator,
            {"operator_address", "consensus_pubkey", "jailed", "status", "tokens"},
            label,
        )
        validator_operators.append(
            _string(validator["operator_address"], f"{label}.operator_address")
        )
        if not isinstance(validator["jailed"], bool):
            _fail(f"{label}.jailed is not boolean")
        if validator["status"] != "BOND_STATUS_BONDED":
            _fail(f"{label} is not bonded")
        _decimal(validator["tokens"], f"{label}.tokens", positive=True)
        pubkey = _exact_object(
            validator["consensus_pubkey"], {"@type", "key"}, f"{label}.consensus_pubkey"
        )
        key_length = 32 if pubkey["@type"] == "/cosmos.crypto.ed25519.PubKey" else None
        _string(pubkey["@type"], f"{label}.consensus_pubkey.@type")
        _public_key(pubkey["key"], f"{label}.consensus_pubkey.key", key_length)
        validator_keys.append(pubkey["key"])
    if validator_operators != sorted(validator_operators) or len(
        set(validator_operators)
    ) != len(validator_operators):
        _fail("inventory validators are not uniquely sorted by operator address")
    if len(set(validator_keys)) != len(validator_keys):
        _fail("inventory validators reuse a consensus key")

    return {
        "source": source,
        "checkpoint_hash": checkpoint_hash,
        "anchor_hash": anchor_hash,
        "halt_hash": halt_hash,
        "post_anchor_hash": post_anchor_hash,
        "anchor_time": anchor_time,
        "halt_time": halt_time,
        "rpc_genesis_hash": rpc_genesis_hash,
        "validator_keys": validator_keys,
        "validator_addresses": [
            hashlib.sha256(base64.b64decode(key)).digest()[:20].hex().upper()
            for key in validator_keys
        ],
    }


TERMINAL_MANIFEST_KEYS = {
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


def _validate_terminal_manifest(
    manifest: dict[str, Any],
    role: str,
    expected_node_id: str,
    inventory: Mapping[str, Any],
    release: Mapping[str, Any],
    transition: Mapping[str, Any],
    f: str,
    a: str,
    h: str,
) -> None:
    _exact_object(manifest, TERMINAL_MANIFEST_KEYS, f"{role} terminal manifest")
    payloads = _exact_object(
        manifest["payload_sha256"], set(RPC_PAYLOAD_SUFFIXES), f"{role} payload hashes"
    )
    predecessor = release["predecessor"]
    if not (
        manifest["schema"] == "zerone-1-terminal-evidence-manifest-v2"
        and manifest["role"] == role
        and manifest["chain_id"] == "zerone-1"
        and manifest["genesis_sha256"] == predecessor["genesis_file_sha256"]
        and manifest["checkpoint_state_height"] == f
        and manifest["checkpoint_app_hash"] == inventory["checkpoint_hash"]
        and manifest["final_committed_height"] == a
        and manifest["final_committed_block_time"] == inventory["anchor_time"]
        and manifest["halt_trigger_height"] == h
        and manifest["anchor_block_hash"] == inventory["anchor_hash"]
        and manifest["halt_trigger_block_hash"] == inventory["halt_hash"]
        and manifest["halt_trigger_block_time"] == inventory["halt_time"]
        and manifest["post_anchor_app_hash"] == inventory["post_anchor_hash"]
        and manifest["post_anchor_app_hash"]
        == transition["expected_post_anchor_app_hash"]
        and manifest["abci_last_applied_height"] == a
        and manifest["node_id"] == expected_node_id
        and manifest["result"] == "MATCH"
    ):
        _fail(f"{role} terminal manifest differs from inventory/transition")
    _public_key(manifest["validator_pubkey"], f"{role} validator pubkey")
    for key, value in payloads.items():
        _lower_hash(value, f"{role}.{key}")


def _rpc_envelope(files: Mapping[str, bytes], prefix: str, key: str) -> dict[str, Any]:
    filename = _rpc_filename(prefix, key)
    if filename not in files:
        _fail(f"required raw RPC evidence {filename} is missing")
    if not files[filename] or len(files[filename]) > 64 * 1024 * 1024:
        _fail(f"raw RPC evidence {filename} is empty or too large")
    value = _parse_json(files[filename], filename)
    if (
        not isinstance(value, dict)
        or value.get("jsonrpc") != "2.0"
        or "id" not in value
    ):
        _fail(f"{filename} must contain a JSON-RPC 2.0 response with an ID")
    return value


def _validate_empty_block(
    envelope: Mapping[str, Any],
    label: str,
    chain_id: str,
    height: str,
    block_hash: str,
    app_hash: str,
    previous_hash: str | None,
    require_empty: bool = True,
) -> tuple[str, dict[str, Any]]:
    result = _path(envelope, label, "result")
    got_hash = _path(result, label, "block_id", "hash")
    header = _path(result, label, "block", "header")
    if not (
        isinstance(header, dict)
        and header.get("chain_id") == chain_id
        and header.get("height") == height
        and isinstance(got_hash, str)
        and got_hash.upper() == block_hash
        and _normalize_app_hash(header.get("app_hash"), f"{label} header AppHash")
        == app_hash
    ):
        _fail(f"{label} does not identify the expected block")
    if previous_hash is not None:
        got_previous = _path(header, label, "last_block_id", "hash")
        if not isinstance(got_previous, str) or got_previous.upper() != previous_hash:
            _fail(f"{label} does not link to the expected previous block")
    txs = _path(result, label, "block", "data", "txs")
    if require_empty and txs not in (None, []):
        _fail(f"{label} is not transaction-empty")
    return _rfc3339_nano(header.get("time"), f"{label} time"), header


def _validate_commit(
    envelope: Mapping[str, Any],
    label: str,
    chain_id: str,
    height: str,
    block_hash: str,
    app_hash: str,
    canonical: bool,
    expected_header: Mapping[str, Any],
    validator_addresses: set[str],
    required_validator_address: str | None,
) -> None:
    result = _path(envelope, label, "result")
    header = _path(result, label, "signed_header", "header")
    commit = _path(result, label, "signed_header", "commit")
    signatures = _path(commit, label, "signatures")
    if not (
        isinstance(result, dict)
        and result.get("canonical") is canonical
        and isinstance(header, dict)
        and header == expected_header
        and header.get("chain_id") == chain_id
        and header.get("height") == height
        and _normalize_app_hash(header.get("app_hash"), f"{label} header AppHash")
        == app_hash
        and isinstance(commit, dict)
        and commit.get("height") == height
        and isinstance(_path(commit, label, "block_id", "hash"), str)
        and _path(commit, label, "block_id", "hash").upper() == block_hash
        and isinstance(signatures, list)
        and len(signatures) > 0
    ):
        _fail(f"{label} is not the expected signed commit")
    committed_addresses: set[str] = set()
    for index, signature in enumerate(signatures):
        signature_label = f"{label} signature[{index}]"
        if not isinstance(signature, dict) or set(signature) != {
            "block_id_flag",
            "validator_address",
            "timestamp",
            "signature",
        }:
            _fail(f"{signature_label} does not have the exact Comet fields")
        flag = signature["block_id_flag"]
        if flag == 1:
            if not (
                signature["validator_address"] in ("", None)
                and signature["signature"] is None
            ):
                _fail(f"{signature_label} is not a canonical absent signature")
            continue
        if flag not in (2, 3):
            _fail(f"{signature_label} has an invalid block ID flag")
        address = signature["validator_address"]
        if not isinstance(address, str) or not re.fullmatch(r"[0-9A-F]{40}", address):
            _fail(f"{signature_label} has an invalid validator address")
        _rfc3339_nano(signature["timestamp"], f"{signature_label} timestamp")
        _public_key(signature["signature"], f"{signature_label} bytes", 64)
        if flag == 2:
            committed_addresses.add(address)
    if not committed_addresses or not committed_addresses.issubset(
        validator_addresses
    ):
        _fail(f"{label} has no structurally valid bonded-validator precommit")
    if (
        required_validator_address is not None
        and required_validator_address not in committed_addresses
    ):
        _fail(f"{label} is not signed by the trusted signer validator")


def _rpc_nonnegative_integer(value: Any, label: str) -> int:
    if isinstance(value, int) and not isinstance(value, bool) and value >= 0:
        return value
    if isinstance(value, str) and DECIMAL.fullmatch(value) and len(value) <= 18:
        return int(value)
    _fail(f"{label} is not a canonical non-negative RPC integer")


def _rpc_signed_decimal(value: Any, label: str) -> None:
    if not isinstance(value, str) or not re.fullmatch(
        r"(?:0|-?[1-9][0-9]{0,255})", value
    ):
        _fail(f"{label} is not a canonical signed decimal")


def _validate_validator_page(
    envelope: Mapping[str, Any], label: str, expected_height: str
) -> set[str]:
    result = _path(envelope, label, "result")
    validators = _path(result, label, "validators")
    count = _rpc_nonnegative_integer(_path(result, label, "count"), f"{label} count")
    total = _rpc_nonnegative_integer(_path(result, label, "total"), f"{label} total")
    block_height = _path(result, label, "block_height")
    if not (
        block_height == expected_height
        and isinstance(validators, list)
        and validators
        and count == total == len(validators)
    ):
        _fail(f"{label} is incomplete, paginated, or at the wrong height")
    addresses: list[str] = []
    for index, validator in enumerate(validators):
        item_label = f"{label} validator[{index}]"
        if not isinstance(validator, dict):
            _fail(f"{item_label} is not an object")
        address = validator.get("address")
        if not isinstance(address, str) or not re.fullmatch(r"[0-9A-F]{40}", address):
            _fail(f"{item_label} address is malformed")
        pubkey = validator.get("pub_key")
        if not isinstance(pubkey, dict) or set(pubkey) != {"type", "value"}:
            _fail(f"{item_label} public key is malformed")
        _string(pubkey["type"], f"{item_label} public-key type")
        _public_key(pubkey["value"], f"{item_label} public key")
        _decimal(validator.get("voting_power"), f"{item_label} voting power", True)
        _rpc_signed_decimal(
            validator.get("proposer_priority"), f"{item_label} proposer priority"
        )
        addresses.append(address)
    if len(set(addresses)) != len(addresses):
        _fail(f"{label} repeats a validator address")
    return set(addresses)


def _validate_rpc_set(
    files: Mapping[str, bytes],
    prefix: str,
    manifest: Mapping[str, Any],
    inventory: Mapping[str, Any],
    f: str,
    a: str,
    h: str,
) -> dict[str, str]:
    del f  # F is committed by A's header and the inventory, not an RPC height here.
    payloads: dict[str, dict[str, Any]] = {}
    hashes: dict[str, str] = {}
    for key in RPC_PAYLOAD_SUFFIXES:
        filename = _rpc_filename(prefix, key)
        payloads[key] = _rpc_envelope(files, prefix, key)
        hashes[key] = _sha256(files[filename])
        if manifest["payload_sha256"].get(key) != hashes[key]:
            _fail(f"{manifest['role']} manifest hash differs from {filename}")

    status = _path(payloads["status_json"], f"{prefix} status", "result")
    voting_power = _path(
        status, f"{prefix} status", "validator_info", "voting_power"
    )
    expected_signer = manifest.get("role") == "official-signer"
    parsed_voting_power = _decimal(
        voting_power, f"{prefix} status validator voting power"
    )
    if not (
        _path(status, f"{prefix} status", "node_info", "network") == "zerone-1"
        and _path(status, f"{prefix} status", "node_info", "id")
        == manifest["node_id"]
        and _path(status, f"{prefix} status", "sync_info", "latest_block_height")
        == h
        and isinstance(
            _path(status, f"{prefix} status", "sync_info", "latest_block_hash"), str
        )
        and _path(
            status, f"{prefix} status", "sync_info", "latest_block_hash"
        ).upper()
        == inventory["halt_hash"]
        and _normalize_app_hash(
            _path(status, f"{prefix} status", "sync_info", "latest_app_hash"),
            f"{prefix} status latest AppHash",
        )
        == inventory["post_anchor_hash"]
        and _path(status, f"{prefix} status", "sync_info", "catching_up") is False
        and _path(
            status,
            f"{prefix} status",
            "validator_info",
            "pub_key",
            "value",
        )
        == manifest["validator_pubkey"]
        and ((parsed_voting_power > 0) if expected_signer else (parsed_voting_power == 0))
    ):
        _fail(f"{prefix} status does not prove the expected stable H/A source")

    genesis = _path(payloads["genesis_json"], f"{prefix} genesis", "result", "genesis")
    if not isinstance(genesis, dict) or genesis.get("chain_id") != "zerone-1":
        _fail(f"{prefix} raw genesis does not identify zerone-1")

    _, trusted_header = _validate_empty_block(
        payloads["trusted_block_json"],
        f"{prefix} trusted block",
        "zerone-1",
        inventory["trusted_height"],
        inventory["trusted_block_hash"],
        inventory["trusted_app_hash"],
        None,
        False,
    )
    trusted_validator_addresses = _validate_validator_page(
        payloads["trusted_validators_json"],
        f"{prefix} trusted validators",
        inventory["trusted_height"],
    )
    _validate_commit(
        payloads["trusted_commit_json"],
        f"{prefix} trusted commit",
        "zerone-1",
        inventory["trusted_height"],
        inventory["trusted_block_hash"],
        inventory["trusted_app_hash"],
        True,
        trusted_header,
        trusted_validator_addresses,
        None,
    )

    anchor_time, anchor_header = _validate_empty_block(
        payloads["block_a_json"],
        f"{prefix} block A",
        "zerone-1",
        a,
        inventory["anchor_hash"],
        inventory["checkpoint_hash"],
        None,
    )
    halt_time, halt_header = _validate_empty_block(
        payloads["block_h_json"],
        f"{prefix} block H",
        "zerone-1",
        h,
        inventory["halt_hash"],
        inventory["post_anchor_hash"],
        inventory["anchor_hash"],
    )
    if anchor_time != inventory["anchor_time"] or halt_time != inventory["halt_time"]:
        _fail(f"{prefix} terminal block times differ from inventory")
    anchor_validator_addresses = _validate_validator_page(
        payloads["validators_a_json"], f"{prefix} validators A", a
    )
    halt_validator_addresses = _validate_validator_page(
        payloads["validators_h_json"], f"{prefix} validators H", h
    )
    required_validator_address = None
    if expected_signer:
        required_validator_address = hashlib.sha256(
            _public_key(
                manifest["validator_pubkey"],
                f"{prefix} trusted signer validator key",
            )
        ).digest()[:20].hex().upper()
    _validate_commit(
        payloads["commit_a_json"],
        f"{prefix} commit A",
        "zerone-1",
        a,
        inventory["anchor_hash"],
        inventory["checkpoint_hash"],
        True,
        anchor_header,
        anchor_validator_addresses,
        required_validator_address,
    )
    _validate_commit(
        payloads["commit_h_json"],
        f"{prefix} commit H",
        "zerone-1",
        h,
        inventory["halt_hash"],
        inventory["post_anchor_hash"],
        False,
        halt_header,
        halt_validator_addresses,
        required_validator_address,
    )
    block_results_a = _path(
        payloads["block_results_a_json"], f"{prefix} block results A", "result"
    )
    if not (
        isinstance(block_results_a, dict)
        and block_results_a.get("height") == a
        and block_results_a.get("txs_results") in (None, [])
        and _normalize_app_hash(
            block_results_a.get("app_hash"),
            f"{prefix} block results A AppHash",
        )
        == inventory["post_anchor_hash"]
    ):
        _fail(f"{prefix} block results A do not prove the applied empty anchor")
    abci = _path(payloads["abci_info_json"], f"{prefix} ABCI", "result", "response")
    if not (
        isinstance(abci, dict)
        and abci.get("last_block_height") == a
        and _normalize_app_hash(
            abci.get("last_block_app_hash"), f"{prefix} ABCI last AppHash"
        )
        == inventory["post_anchor_hash"]
    ):
        _fail(f"{prefix} ABCI does not prove application height A")
    missing = payloads["block_results_h_missing_response"]
    result = missing.get("result")
    error = missing.get("error")
    if result is not None or not isinstance(error, dict):
        _fail(f"{prefix} H block-results response does not prove absence")
    missing_text = f"{error.get('message', '')} {json.dumps(error.get('data'))}"
    if f"could not find results for height #{h}" not in missing_text:
        _fail(f"{prefix} H block-results response has the wrong error")
    return hashes


def _uint64(value: Any, label: str, positive: bool = False) -> int:
    number = _integer(value, label, 1 if positive else 0)
    if number > 2**64 - 1:
        _fail(f"{label} exceeds uint64")
    return number


def _census_amount(value: Any, label: str, positive: bool = False) -> int:
    if not isinstance(value, str) or len(value) > 78 or not DECIMAL.fullmatch(value):
        _fail(f"{label} is not a canonical bounded census amount")
    number = int(value)
    if number.bit_length() > 256:
        _fail(f"{label} exceeds the SDK 256-bit amount bound")
    if positive and number <= 0:
        _fail(f"{label} must be positive")
    return number


def _census_aggregate(value: Any, label: str) -> int:
    if not isinstance(value, str) or len(value) > 98 or not DECIMAL.fullmatch(value):
        _fail(f"{label} is not a canonical bounded census aggregate")
    number = int(value)
    if number.bit_length() > 320:
        _fail(f"{label} exceeds the census 320-bit aggregate bound")
    return number


def _go_uvarint(value: int) -> bytes:
    if value < 0 or value > 2**64 - 1:
        raise ValueError("Go uvarint input is outside uint64")
    encoded = bytearray()
    while value >= 0x80:
        encoded.append((value & 0x7F) | 0x80)
        value >>= 7
    encoded.append(value)
    return bytes(encoded)


def _rfc6962_hash(leaves: list[bytes]) -> bytes:
    if not leaves:
        return hashlib.sha256(b"").digest()
    if len(leaves) == 1:
        return hashlib.sha256(b"\x00" + leaves[0]).digest()
    split = 1 << ((len(leaves) - 1).bit_length() - 1)
    left = _rfc6962_hash(leaves[:split])
    right = _rfc6962_hash(leaves[split:])
    return hashlib.sha256(b"\x01" + left + right).digest()


def _cosmos_commit_info_hash(multistore_roots: Mapping[str, str]) -> str:
    """Reproduce cosmossdk.io/store v1.1.2 CommitInfo.Hash without Go code."""

    encoded_rows: list[tuple[bytes, str]] = []
    for name, root in multistore_roots.items():
        try:
            encoded_rows.append((name.encode("utf-8"), root))
        except (AttributeError, UnicodeEncodeError):
            _fail("custom-staking multistore name is not valid UTF-8")
    leaves: list[bytes] = []
    for key, root_hex in sorted(encoded_rows):
        root = bytes.fromhex(root_hex)
        value_hash = hashlib.sha256(root).digest()
        leaves.append(
            _go_uvarint(len(key))
            + key
            + _go_uvarint(len(value_hash))
            + value_hash
        )
    return _rfc6962_hash(leaves).hex()


def _claimant_root(claims: list[dict[str, Any]]) -> str:
    digest = hashlib.sha256()

    def write_field(value: str) -> None:
        try:
            encoded = value.encode("utf-8")
        except UnicodeEncodeError:
            _fail("custom-staking claimant field is not valid UTF-8")
        digest.update(len(encoded).to_bytes(8, "big"))
        digest.update(encoded)

    write_field("zerone/custom-staking-claimants/v1")
    digest.update(len(claims).to_bytes(8, "big"))
    for claim in claims:
        for key in (
            "source_kind",
            "source_claim_id",
            "claimant",
            "validator",
            "denom",
            "amount",
        ):
            write_field(claim[key])
    return digest.hexdigest()


def _validate_custom_staking_census(
    files: Mapping[str, bytes],
    objects: Mapping[str, Any],
    final: Mapping[str, Any],
    release: Mapping[str, Any],
    transition: Mapping[str, Any],
    inventory: Mapping[str, Any],
    f: str,
    a: str,
) -> str:
    name = "CUSTOM-STAKING-CENSUS.json"
    raw = files.get(name)
    if raw is None or not raw or len(raw) > 64 * 1024 * 1024 + 1:
        _fail("custom-staking census is missing, empty, or oversized")
    report = _exact_object(
        _object(files, objects, name),
        {
            "schema",
            "result",
            "evidence",
            "multistore",
            "stores",
            "census",
            "report_sha256",
        },
        "custom-staking census report",
    )
    if report["schema"] != "zerone/custom-staking-census/v1":
        _fail("custom-staking census schema changed")
    if report["result"] != "PASS":
        _fail("custom-staking census did not PASS")

    report_hash = _lower_hash(
        report["report_sha256"], "custom-staking census self-hash"
    )
    suffix = f',"report_sha256":"{report_hash}"}}'.encode()
    if not raw.endswith(b"\n") or not raw[:-1].endswith(suffix):
        _fail("custom-staking census is not the exact sealed report encoding")
    sealed = raw[:-1]
    unsealed = sealed[: -len(suffix)] + b',"report_sha256":""}'
    if _sha256(unsealed) != report_hash:
        _fail("custom-staking census self-hash does not match its unsealed bytes")

    evidence = _exact_object(
        report["evidence"],
        {"chain_id", "height", "app_hash", "source_commit"},
        "custom-staking census evidence",
    )
    source = release.get("source")
    if not isinstance(source, dict):
        _fail("RELEASE source is missing for custom-staking census binding")
    source_commit = source.get("commit")
    if not isinstance(source_commit, str) or not re.fullmatch(
        r"[0-9a-f]{40}", source_commit
    ):
        _fail("RELEASE source commit is not canonical for census binding")
    expected_app_hash = inventory["post_anchor_hash"].lower()
    excluded = final.get("excluded_post_anchor_state")
    if not isinstance(excluded, dict) or not (
        excluded.get("abci_last_applied_height") == a
        and excluded.get("app_hash") == inventory["post_anchor_hash"]
        and transition.get("expected_post_anchor_app_hash")
        == inventory["post_anchor_hash"]
    ):
        _fail("FINAL/transition excluded post-anchor state is not census-bindable")
    if not (
        evidence["chain_id"] == "zerone-1"
        and evidence["height"] == a
        and evidence["height"] != f
        and evidence["app_hash"] == expected_app_hash
        and evidence["source_commit"] == source_commit
    ):
        _fail(
            "custom-staking census must bind RELEASE source and excluded "
            "post-anchor application state A, never checkpoint F"
        )
    _lower_hash(evidence["app_hash"], "custom-staking census AppHash")

    multistore = report["multistore"]
    if not isinstance(multistore, list) or not 1 <= len(multistore) <= 4096:
        _fail("custom-staking census multistore is not a bounded non-empty list")
    multistore_roots: dict[str, str] = {}
    previous_name = b""
    for index, row in enumerate(multistore):
        row = _exact_object(
            row, {"name", "root_sha256"}, f"custom-staking multistore[{index}]"
        )
        store_name = _string(row["name"], f"custom-staking multistore[{index}].name")
        try:
            encoded_name = store_name.encode("utf-8")
        except UnicodeEncodeError:
            _fail("custom-staking multistore name is not valid UTF-8")
        if (
            len(encoded_name) > 128
            or encoded_name <= previous_name
            or store_name in multistore_roots
        ):
            _fail("custom-staking multistore names are not uniquely sorted")
        previous_name = encoded_name
        multistore_roots[store_name] = _lower_hash(
            row["root_sha256"], f"custom-staking multistore[{index}].root"
        )
    if _cosmos_commit_info_hash(multistore_roots) != expected_app_hash:
        _fail("custom-staking multistore roots do not recompute post-anchor AppHash E")

    stores = report["stores"]
    required_stores = ("zerone_staking", "bank", "staking")
    if not isinstance(stores, list) or len(stores) != len(required_stores):
        _fail("custom-staking census does not contain the three required stores")
    store_stats: dict[str, tuple[int, int]] = {}
    for index, (row, expected_name) in enumerate(zip(stores, required_stores)):
        row = _exact_object(
            row,
            {
                "name",
                "version",
                "root_sha256",
                "leaf_count",
                "input_bytes",
                "leaves_sha256",
            },
            f"custom-staking store[{index}]",
        )
        if row["name"] != expected_name or row["version"] != a:
            _fail("custom-staking store order/version differs from application A")
        root_hash = _lower_hash(
            row["root_sha256"], f"custom-staking store[{index}].root"
        )
        if multistore_roots.get(expected_name) != root_hash:
            _fail("custom-staking store root differs from its multistore commitment")
        _lower_hash(
            row["leaves_sha256"], f"custom-staking store[{index}].leaves"
        )
        for field in ("leaf_count", "input_bytes"):
            value = row[field]
            if not isinstance(value, str) or not re.fullmatch(
                r"(?:0|[1-9][0-9]{0,19})", value
            ):
                _fail(f"custom-staking store[{index}].{field} is not canonical")
            if int(value) > 2**64 - 1:
                _fail(f"custom-staking store[{index}].{field} exceeds uint64")
        leaf_count = int(row["leaf_count"])
        input_bytes = int(row["input_bytes"])
        leaf_limit = (
            CENSUS_MAX_CUSTOM_STORE_LEAVES
            if expected_name == "zerone_staking"
            else CENSUS_MAX_STORE_LEAVES
        )
        if leaf_count > leaf_limit:
            _fail(f"custom-staking store[{index}] exceeds its scan leaf ceiling")
        if (
            expected_name == "zerone_staking"
            and input_bytes > CENSUS_MAX_CUSTOM_INPUT_BYTES
        ):
            _fail("custom-staking store exceeds its retained-input scan ceiling")
        store_stats[expected_name] = (leaf_count, input_bytes)
    total_input_bytes = sum(
        input_bytes for _, input_bytes in store_stats.values()
    )
    if total_input_bytes > CENSUS_MAX_SCANNED_INPUT_BYTES:
        _fail("custom-staking required stores exceed the aggregate scan byte ceiling")

    census = _exact_object(
        report["census"],
        {
            "module_address",
            "module_address_hex",
            "module_balances",
            "balance_uzrn",
            "delegations_uzrn",
            "pending_unbondings_uzrn",
            "liabilities_uzrn",
            "delta_uzrn",
            "claimant_root",
            "claimant_root_complete",
            "claim_count",
            "custom_keyspace",
            "validators",
            "claims",
            "unbondings",
            "did_indexes",
            "reverse_delegation_indexes",
            "redelegation_cooldowns",
            "sdk_validators",
            "tier_configs",
            "findings",
        },
        "custom-staking census",
    )
    module_bytes = hashlib.sha256(b"zerone_staking").digest()[:20]
    if not (
        census["module_address"] == _bech32_encode("zrn", module_bytes)
        and census["module_address_hex"] == module_bytes.hex()
    ):
        _fail("custom-staking module identity is not deterministic")
    balance = _census_aggregate(census["balance_uzrn"], "custom-staking B")
    delegations = _census_aggregate(
        census["delegations_uzrn"], "custom-staking D"
    )
    pending = _census_aggregate(
        census["pending_unbondings_uzrn"], "custom-staking U"
    )
    liabilities = _census_aggregate(
        census["liabilities_uzrn"], "custom-staking liabilities"
    )
    if not (
        liabilities == delegations + pending
        and balance == liabilities
        and census["delta_uzrn"] == "0"
    ):
        _fail("custom-staking census does not prove B = D + U and delta = 0")
    if census["claimant_root_complete"] is not True or census["findings"] != []:
        _fail("custom-staking census is incomplete or contains findings")

    balances = census["module_balances"]
    expected_balance_rows = 0 if balance == 0 else 1
    if not isinstance(balances, list) or len(balances) != expected_balance_rows:
        _fail("custom-staking module balances contain an unexplained denomination")
    if balances:
        row = _exact_object(
            balances[0], {"denom", "amount"}, "custom-staking module balance[0]"
        )
        if row["denom"] != "uzrn" or _census_amount(
            row["amount"], "custom-staking module balance[0].amount", True
        ) != balance:
            _fail("custom-staking module balances do not exactly explain B")

    collection_names = (
        "custom_keyspace",
        "validators",
        "claims",
        "unbondings",
        "did_indexes",
        "reverse_delegation_indexes",
        "redelegation_cooldowns",
        "sdk_validators",
        "tier_configs",
    )
    if any(not isinstance(census[key], list) for key in collection_names):
        _fail("custom-staking census collections must all be lists")

    validators = census["validators"]
    validator_set: set[str] = set()
    validator_rows: dict[str, dict[str, Any]] = {}
    previous_validator = ""
    validator_fields = {
        "operator",
        "address_hex",
        "legacy_consensus_pubkey",
        "legacy_consensus_pubkey_trusted",
        "stored_self",
        "computed_self",
        "stored_delegated",
        "computed_delegated",
        "stored_total",
        "computed_total",
        "aggregates_match",
        "sdk_link",
    }
    for index, validator in enumerate(validators):
        label = f"custom-staking validator[{index}]"
        if not isinstance(validator, dict) or set(validator) not in {
            frozenset(validator_fields),
            frozenset(validator_fields | {"sdk_operator"}),
        }:
            _fail(f"{label} does not have the exact required fields")
        operator = validator["operator"]
        address = _bech32_address(operator, "zrn", f"{label}.operator")
        if operator <= previous_validator or validator["address_hex"] != address.hex():
            _fail("custom-staking validators are not ordered canonical identities")
        previous_validator = operator
        if operator in validator_set:
            _fail("custom-staking validators repeat an operator")
        validator_set.add(operator)
        if (
            validator["aggregates_match"] is not True
            or validator["legacy_consensus_pubkey_trusted"] is not False
            or not _string(
                validator["legacy_consensus_pubkey"],
                f"{label}.legacy_consensus_pubkey",
            )
        ):
            _fail(f"{label} has invalid aggregate or legacy-key trust state")
        amounts = {
            field: _census_amount(validator[field], f"{label}.{field}")
            for field in (
                "stored_self",
                "computed_self",
                "stored_delegated",
                "computed_delegated",
                "stored_total",
                "computed_total",
            )
        }
        if not (
            amounts["stored_self"] == amounts["computed_self"]
            and amounts["stored_delegated"] == amounts["computed_delegated"]
            and amounts["stored_total"] == amounts["computed_total"]
            and amounts["stored_total"]
            == amounts["stored_self"] + amounts["stored_delegated"]
        ):
            _fail(f"{label} aggregate values are inconsistent")
        sdk_link = validator["sdk_link"]
        if sdk_link == "absent":
            if set(validator) != validator_fields:
                _fail(f"{label} unexpectedly carries an SDK operator")
            sdk_operator = None
        elif sdk_link == "linked":
            if set(validator) != validator_fields | {"sdk_operator"}:
                _fail(f"{label} omits its linked SDK operator")
            sdk_operator = _string(validator["sdk_operator"], f"{label}.sdk_operator")
        else:
            _fail(f"{label} has an invalid SDK link state")
        validator_rows[operator] = {
            "address_hex": validator["address_hex"],
            "computed_self": amounts["computed_self"],
            "computed_delegated": amounts["computed_delegated"],
            "sdk_link": sdk_link,
            "sdk_operator": sdk_operator,
        }

    claims = census["claims"]
    claim_count = census["claim_count"]
    if (
        not isinstance(claim_count, int)
        or isinstance(claim_count, bool)
        or claim_count < 0
        or claim_count > 50_000
        or claim_count != len(claims)
    ):
        _fail("custom-staking claim count is invalid or incomplete")
    delegation_total = 0
    pending_total = 0
    seen_sources: set[tuple[str, str]] = set()
    previous_claim: tuple[str, ...] | None = None
    claim_self: dict[str, int] = {}
    claim_delegated: dict[str, int] = {}
    delegation_pairs: set[tuple[str, str]] = set()
    pending_by_id: dict[str, dict[str, Any]] = {}
    for index, claim in enumerate(claims):
        claim = _exact_object(
            claim,
            {
                "source_kind",
                "source_claim_id",
                "claimant",
                "validator",
                "denom",
                "amount",
            },
            f"custom-staking claim[{index}]",
        )
        label = f"custom-staking claim[{index}]"
        for field in ("source_kind", "source_claim_id", "denom", "amount"):
            _string(claim[field], f"{label}.{field}")
        _bech32_address(claim["claimant"], "zrn", f"{label}.claimant")
        _bech32_address(claim["validator"], "zrn", f"{label}.validator")
        if claim["validator"] not in validator_set:
            _fail(f"{label} targets no reported custom validator")
        order = tuple(
            claim[field]
            for field in (
                "source_kind",
                "source_claim_id",
                "claimant",
                "validator",
                "denom",
                "amount",
            )
        )
        if previous_claim is not None and order <= previous_claim:
            _fail("custom-staking claims are not in strict deterministic order")
        previous_claim = order
        source = (claim["source_kind"], claim["source_claim_id"])
        if source in seen_sources:
            _fail("custom-staking claimant source is duplicated")
        seen_sources.add(source)
        if claim["denom"] != "uzrn":
            _fail("custom-staking claim denomination is not uzrn")
        amount = _census_amount(
            claim["amount"], f"custom-staking claim[{index}].amount", positive=True
        )
        if claim["source_kind"] == "delegation":
            if claim["source_claim_id"] != (
                f'{claim["claimant"]}->{claim["validator"]}'
            ):
                _fail("custom-staking delegation claimant source is malformed")
            delegation_total += amount
            pair = (claim["validator"], claim["claimant"])
            delegation_pairs.add(pair)
            target = (
                claim_self
                if claim["claimant"] == claim["validator"]
                else claim_delegated
            )
            target[claim["validator"]] = target.get(claim["validator"], 0) + amount
        elif claim["source_kind"] == "pending_unbonding":
            pending_total += amount
            pending_by_id[claim["source_claim_id"]] = claim
        else:
            _fail("custom-staking claim has an unknown source kind")
    if delegation_total != delegations or pending_total != pending:
        _fail("custom-staking claimant rows do not sum to D and U")

    claimant_root = _lower_hash(
        census["claimant_root"], "custom-staking claimant root"
    )
    if claimant_root != _claimant_root(claims):
        _fail("custom-staking claimant root does not match the complete claim list")

    unbondings = census["unbondings"]
    previous_unbonding_id = ""
    seen_sequences: set[int] = set()
    pending_records = 0
    for index, entry in enumerate(unbondings):
        label = f"custom-staking unbonding[{index}]"
        entry = _exact_object(
            entry,
            {
                "id",
                "delegator",
                "validator",
                "amount",
                "created_at_height",
                "completes_at_height",
                "status",
                "sequence",
            },
            label,
        )
        entry_id = _string(entry["id"], f"{label}.id")
        if entry_id <= previous_unbonding_id:
            _fail("custom-staking unbondings are not in strict ID order")
        previous_unbonding_id = entry_id
        _bech32_address(entry["delegator"], "zrn", f"{label}.delegator")
        _bech32_address(entry["validator"], "zrn", f"{label}.validator")
        _census_amount(entry["amount"], f"{label}.amount", True)
        created = _uint64(entry["created_at_height"], f"{label}.created_at_height")
        completes = _uint64(
            entry["completes_at_height"], f"{label}.completes_at_height"
        )
        sequence = _uint64(entry["sequence"], f"{label}.sequence", True)
        if completes <= created or sequence in seen_sequences:
            _fail(f"{label} has invalid or duplicate lifecycle coordinates")
        seen_sequences.add(sequence)
        if entry["status"] == "pending":
            pending_records += 1
            claim = pending_by_id.get(entry_id)
            if entry["validator"] not in validator_set or not claim or not (
                claim["claimant"] == entry["delegator"]
                and claim["validator"] == entry["validator"]
                and claim["amount"] == entry["amount"]
            ):
                _fail(f"{label} does not match its pending claimant record")
        elif entry["status"] != "completed":
            _fail(f"{label} has an invalid status")
    if pending_records != len(pending_by_id):
        _fail("custom-staking pending claims do not cover pending unbondings")

    reverse_indexes = census["reverse_delegation_indexes"]
    previous_reverse: tuple[str, str] | None = None
    for index, row in enumerate(reverse_indexes):
        label = f"custom-staking reverse delegation index[{index}]"
        row = _exact_object(row, {"validator", "delegator"}, label)
        _bech32_address(row["validator"], "zrn", f"{label}.validator")
        _bech32_address(row["delegator"], "zrn", f"{label}.delegator")
        identity = (row["validator"], row["delegator"])
        if previous_reverse is not None and identity <= previous_reverse:
            _fail("custom-staking reverse indexes are not in deterministic order")
        previous_reverse = identity
        if identity not in delegation_pairs:
            _fail(f"{label} has no delegation claim")
    if len(reverse_indexes) != len(delegation_pairs):
        _fail("custom-staking reverse indexes do not cover every delegation claim")

    previous_did = ""
    for index, row in enumerate(census["did_indexes"]):
        label = f"custom-staking DID index[{index}]"
        row = _exact_object(row, {"did", "operator"}, label)
        did = _string(row["did"], f"{label}.did")
        if len(did.encode("utf-8")) > 128:
            _fail(f"{label} exceeds the 128-byte producer ceiling")
        if did <= previous_did or row["operator"] not in validator_set:
            _fail(f"{label} is unordered or targets no custom validator")
        previous_did = did

    previous_cooldown = ""
    for index, row in enumerate(census["redelegation_cooldowns"]):
        label = f"custom-staking redelegation cooldown[{index}]"
        row = _exact_object(row, {"delegator", "height"}, label)
        _bech32_address(row["delegator"], "zrn", f"{label}.delegator")
        if row["delegator"] <= previous_cooldown:
            _fail("custom-staking redelegation cooldowns are not ordered")
        previous_cooldown = row["delegator"]
        _uint64(row["height"], f"{label}.height", True)

    sdk_by_address: dict[str, dict[str, Any]] = {}
    previous_sdk_operator = ""
    sdk_validators = census["sdk_validators"]
    if len(sdk_validators) > CENSUS_MAX_SDK_VALIDATORS:
        _fail("custom-staking SDK validator inventory exceeds its row ceiling")
    for index, row in enumerate(sdk_validators):
        label = f"custom-staking SDK validator[{index}]"
        row = _exact_object(
            row,
            {"operator", "address_hex", "status", "jailed", "tokens"},
            label,
        )
        address = _bech32_address(row["operator"], "zrnvaloper", f"{label}.operator")
        if (
            row["operator"] <= previous_sdk_operator
            or row["address_hex"] != address.hex()
            or row["address_hex"] in sdk_by_address
        ):
            _fail(f"{label} has an invalid or duplicate operator identity")
        previous_sdk_operator = row["operator"]
        if row["status"] not in {
            "BOND_STATUS_UNBONDED",
            "BOND_STATUS_UNBONDING",
            "BOND_STATUS_BONDED",
        }:
            _fail(f"{label} has an invalid status")
        if not isinstance(row["jailed"], bool):
            _fail(f"{label}.jailed is not boolean")
        _census_amount(row["tokens"], f"{label}.tokens")
        sdk_by_address[row["address_hex"]] = row

    for operator, validator in validator_rows.items():
        if validator["computed_self"] != claim_self.get(operator, 0) or validator[
            "computed_delegated"
        ] != claim_delegated.get(operator, 0):
            _fail(
                "custom-staking validator computed aggregates do not match claims"
            )
        sdk_row = sdk_by_address.get(validator["address_hex"])
        if validator["sdk_link"] == "linked":
            if not sdk_row or sdk_row["operator"] != validator["sdk_operator"]:
                _fail("custom-staking validator SDK link does not match inventory")
        elif sdk_row:
            _fail("custom-staking validator marks an available SDK link absent")

    tiers = census["tier_configs"]
    if len(tiers) != 4:
        _fail("custom-staking tier reconciliation is incomplete")
    for index, row in enumerate(tiers):
        label = f"custom-staking tier reconciliation[{index}]"
        row = _exact_object(
            row,
            {"tier", "name", "stored_digest", "params_digest", "matches"},
            label,
        )
        _string(row["name"], f"{label}.name")
        if (
            not isinstance(row["tier"], int)
            or isinstance(row["tier"], bool)
            or row["tier"] != index + 1
            or row["matches"] is not True
            or row["stored_digest"] != row["params_digest"]
        ):
            _fail(f"{label} is incomplete")
        _lower_hash(row["stored_digest"], f"{label}.stored_digest")
        _lower_hash(row["params_digest"], f"{label}.params_digest")

    keyspace = census["custom_keyspace"]
    expected_names = (
        "validators",
        "delegations",
        "unbondings",
        "tier_configs",
        "params",
        "did_indexes",
        "unbonding_sequence",
        "redelegation_cooldowns",
        "validator_delegation_indexes",
        "app_iavl_init_sentinel",
    )
    expected_prefixes = tuple(f"0x{index:02x}" for index in range(1, 10)) + (
        "0x5f6961766c5f696e6974",
    )
    if not isinstance(keyspace, list) or len(keyspace) != len(expected_names):
        _fail("custom-staking keyspace census is incomplete")
    keyspace_counts: dict[str, int] = {}
    keyspace_input_bytes = 0
    for index, (row, expected_name, expected_prefix) in enumerate(
        zip(keyspace, expected_names, expected_prefixes)
    ):
        row = _exact_object(
            row,
            {"prefix", "name", "leaf_count", "input_bytes", "digest"},
            f"custom-staking keyspace[{index}]",
        )
        prefix = row["prefix"]
        name = row["name"]
        if prefix != expected_prefix or name != expected_name:
            _fail("custom-staking keyspace identities or order changed")
        keyspace_counts[name] = _uint64(
            row["leaf_count"], f"custom-staking keyspace[{index}].leaf_count"
        )
        keyspace_input_bytes += _uint64(
            row["input_bytes"], f"custom-staking keyspace[{index}].input_bytes"
        )
        _lower_hash(row["digest"], f"custom-staking keyspace[{index}].digest")
    expected_counts = {
        "validators": len(census["validators"]),
        "delegations": sum(
            1 for claim in claims if claim["source_kind"] == "delegation"
        ),
        "unbondings": len(census["unbondings"]),
        "tier_configs": len(census["tier_configs"]),
        "params": 1,
        "did_indexes": len(census["did_indexes"]),
        "redelegation_cooldowns": len(census["redelegation_cooldowns"]),
        "validator_delegation_indexes": len(
            census["reverse_delegation_indexes"]
        ),
    }
    if any(keyspace_counts[name] != count for name, count in expected_counts.items()):
        _fail("custom-staking record counts do not cover the complete keyspace")
    if (
        sum(keyspace_counts.values()) != store_stats["zerone_staking"][0]
        or keyspace_input_bytes != store_stats["zerone_staking"][1]
    ):
        _fail("custom-staking keyspaces do not cover the complete scanned store")
    if keyspace_counts["tier_configs"] != 4 or keyspace_counts[
        "unbonding_sequence"
    ] not in {0, 1}:
        _fail("custom-staking tier or sequence inventory is invalid")
    if keyspace_counts["app_iavl_init_sentinel"] not in {0, 1}:
        _fail("custom-staking app IAVL sentinel inventory is invalid")
    if census["unbondings"] and keyspace_counts["unbonding_sequence"] != 1:
        _fail("custom-staking unbondings lack the sequence singleton")
    return _sha256(raw)


def _validate_export_and_storage(
    files: Mapping[str, bytes],
    objects: Mapping[str, Any],
    final: Mapping[str, Any],
    release: Mapping[str, Any],
    transition: Mapping[str, Any],
    inventory: Mapping[str, Any],
    f: str,
    a: str,
    h: str,
) -> None:
    census_sha = _validate_custom_staking_census(
        files, objects, final, release, transition, inventory, f, a
    )
    export_name = "POST-ANCHOR-STATE-EXPORT.json.raw"
    if export_name not in files:
        _fail(f"required frozen evidence file {export_name} is missing")
    export = _parse_json(files[export_name], export_name)
    if not (
        isinstance(export, dict)
        and export.get("chain_id") == "zerone-1"
        and export.get("initial_height") in {int(a), a}
        and isinstance(export.get("app_state"), dict)
        and isinstance(export.get("consensus"), dict)
    ):
        _fail("post-anchor state export is not a zerone-1 application export at A")
    export_sha = _sha256(files[export_name])
    export_evidence = _exact_object(
        _object(files, objects, "POST-ANCHOR-STATE-EXPORT-EVIDENCE.json"),
        {
            "schema",
            "chain_id",
            "checkpoint_state_height",
            "exported_application_height",
            "post_anchor_app_hash",
            "raw_export_sha256",
            "included_in_successor_inventory",
            "result",
        },
        "post-anchor export evidence",
    )
    if export_evidence != {
        "schema": "zerone-1-post-anchor-state-export-evidence-v1",
        "chain_id": "zerone-1",
        "checkpoint_state_height": f,
        "exported_application_height": a,
        "post_anchor_app_hash": inventory["post_anchor_hash"],
        "raw_export_sha256": export_sha,
        "included_in_successor_inventory": False,
        "result": "MATCH",
    }:
        _fail("post-anchor export evidence differs from raw export/F/A state")

    offline_name = "OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json"
    offline = _exact_object(
        _object(files, objects, offline_name),
        {
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
        },
        "offline halted observer snapshot manifest",
    )
    source_observer = transition["source_observer"]
    if not (
        offline["schema"]
        == "zerone-1-offline-halted-observer-snapshot-manifest-v1"
        and offline["chain_id"] == "zerone-1"
        and offline["checkpoint_state_height"] == f
        and offline["checkpoint_app_hash"] == inventory["checkpoint_hash"]
        and offline["final_committed_height"] == a
        and offline["halt_trigger_height"] == h
        and offline["blockstore_height"] == h
        and offline["abci_last_applied_height"] == a
        and offline["post_anchor_app_hash"] == inventory["post_anchor_hash"]
        and offline["source_observer_node_id"] == source_observer["node_id"]
        and offline["stored_offline"] is True
        and offline["included_in_authority_bundle"] is False
        and offline["contains_signer_keys"] is False
        and offline["result"] == "MATCH"
    ):
        _fail("offline halted observer snapshot manifest is unsafe or mismatched")
    offline_database_sha = _lower_hash(
        offline["database_snapshot_sha256"], "offline halted database snapshot"
    )
    _lower_hash(offline["file_manifest_sha256"], "offline halted file manifest")

    sanitized_name = "PRE-TRANSITION-SANITIZED-SNAPSHOT-MANIFEST.json"
    sanitized = _exact_object(
        _object(files, objects, sanitized_name),
        {
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
            "contains_staged_h",
            "contains_signer_keys",
            "contains_authority_artifacts",
            "database_snapshot_sha256",
            "file_manifest_sha256",
            "result",
        },
        "sanitized A/A snapshot manifest",
    )
    if not (
        sanitized["schema"] == "zerone-1-sanitized-snapshot-manifest-v2"
        and sanitized["chain_id"] == "zerone-1"
        and sanitized["checkpoint_state_height"] == f
        and sanitized["checkpoint_app_hash"] == inventory["checkpoint_hash"]
        and sanitized["final_committed_height"] == a
        and sanitized["halt_trigger_height"] == h
        and sanitized["blockstore_height"] == a
        and sanitized["abci_last_applied_height"] == a
        and sanitized["post_anchor_app_hash"] == inventory["post_anchor_hash"]
        and sanitized["source_observer_node_id"] == source_observer["node_id"]
        and sanitized["contains_staged_h"] is False
        and sanitized["contains_signer_keys"] is False
        and sanitized["contains_authority_artifacts"] is False
        and sanitized["result"] == "MATCH"
    ):
        _fail("sanitized A/A snapshot manifest is unsafe or mismatched")
    sanitized_database_sha = _lower_hash(
        sanitized["database_snapshot_sha256"], "sanitized A/A database snapshot"
    )
    _lower_hash(sanitized["file_manifest_sha256"], "sanitized A/A file manifest")
    if sanitized_database_sha == offline_database_sha:
        _fail("halted H/A and sanitized A/A database snapshot hashes are identical")

    rollback_output_name = "ARCHIVE-ROLLBACK-OUTPUT.log"
    if rollback_output_name not in files or not files[rollback_output_name]:
        _fail("raw archive rollback output is missing or empty")
    rollback_output_sha = _sha256(files[rollback_output_name])
    rollback_name = "ARCHIVE-ROLLBACK-LOG.json"
    rollback = _exact_object(
        _object(files, objects, rollback_name),
        {
            "schema",
            "chain_id",
            "source_offline_snapshot_manifest_sha256",
            "sanitized_snapshot_manifest_sha256",
            "from_blockstore_height",
            "to_blockstore_height",
            "abci_last_applied_height",
            "post_anchor_app_hash",
            "command",
            "exit_code",
            "raw_output_sha256",
            "staged_h_removed",
            "result",
        },
        "archive rollback evidence",
    )
    if rollback != {
        "schema": "zerone-1-archive-rollback-evidence-v2",
        "chain_id": "zerone-1",
        "source_offline_snapshot_manifest_sha256": _sha256(files[offline_name]),
        "sanitized_snapshot_manifest_sha256": _sha256(files[sanitized_name]),
        "from_blockstore_height": h,
        "to_blockstore_height": a,
        "abci_last_applied_height": a,
        "post_anchor_app_hash": inventory["post_anchor_hash"],
        "command": ["zeroned", "rollback", "--hard"],
        "exit_code": 0,
        "raw_output_sha256": rollback_output_sha,
        "staged_h_removed": True,
        "result": "MATCH",
    }:
        _fail("archive rollback evidence differs from the H/A -> A/A operation")

    construction = transition.get("archive_construction_evidence")
    if not isinstance(construction, dict) or not (
        construction.get("pre_transition_sanitized_snapshot_sha256")
        == _sha256(files[sanitized_name])
        and construction.get("rollback_log_sha256") == _sha256(files[rollback_name])
    ):
        _fail("transition does not bind the exact sanitized snapshot/rollback evidence")

    artifacts = _exact_object(
        final.get("artifacts"),
        {
            "custom_staking_census_sha256",
            "post_anchor_state_export_sha256",
            "post_anchor_state_export_included_in_successor_inventory",
            "offline_halted_observer_database_snapshot_sha256",
            "sanitized_a_a_database_snapshot_sha256",
            "archive_rollback_log_sha256",
        },
        "FINAL frozen artifacts",
    )
    if artifacts != {
        "custom_staking_census_sha256": census_sha,
        "post_anchor_state_export_sha256": export_sha,
        "post_anchor_state_export_included_in_successor_inventory": False,
        "offline_halted_observer_database_snapshot_sha256": offline_database_sha,
        "sanitized_a_a_database_snapshot_sha256": sanitized_database_sha,
        "archive_rollback_log_sha256": _sha256(files[rollback_name]),
    }:
        _fail("FINAL frozen artifact hashes differ from the actual evidence")


def _validate_final_boundary(
    files: Mapping[str, bytes],
    final: Mapping[str, Any],
    release: Mapping[str, Any],
    inventory: Mapping[str, Any],
    manifests: Mapping[str, Mapping[str, Any]],
    rpc_hashes: Mapping[str, Mapping[str, str]],
    f: str,
    a: str,
    h: str,
) -> None:
    checkpoint = _exact_object(
        final.get("checkpoint_state"),
        {
            "height",
            "app_hash",
            "inventory_v3_sha256",
            "rest_height_pinned",
            "rest_merkle_proved",
            "trust_model",
        },
        "FINAL checkpoint state",
    )
    application = _exact_object(
        final.get("final_application_block"),
        {
            "height",
            "block_id_hash",
            "header_app_hash",
            "transaction_count",
            "time",
            "commit_canonical",
        },
        "FINAL application block",
    )
    halt = _exact_object(
        final.get("halt_trigger_tip"),
        {
            "height",
            "blockstore_status_tip_before_fencing",
            "staged_block_id_hash",
            "staged_header_last_block_id_hash",
            "transaction_count",
            "sdk_hook",
            "subjective_seen_commit_canonical_field",
            "canonical_field_interpretation",
            "block_results_available",
            "application_applied",
        },
        "FINAL halt trigger",
    )
    excluded = _exact_object(
        final.get("excluded_post_anchor_state"),
        {
            "abci_last_applied_height",
            "app_hash",
            "included_in_successor_inventory",
            "reason",
        },
        "FINAL excluded post-anchor state",
    )
    if not (
        checkpoint["height"] == f
        and checkpoint["app_hash"] == inventory["checkpoint_hash"]
        and checkpoint["inventory_v3_sha256"]
        == _sha256(files["ZERONE-1-INVENTORY-V3.json"])
        and checkpoint["rest_height_pinned"] is True
        and checkpoint["rest_merkle_proved"] is False
        and checkpoint["trust_model"] == FINAL_TRUST_MODEL
        and application["height"] == a
        and application["block_id_hash"] == inventory["anchor_hash"]
        and application["header_app_hash"] == inventory["checkpoint_hash"]
        and application["transaction_count"] == 0
        and application["time"] == inventory["anchor_time"]
        and application["commit_canonical"] is True
        and halt["height"] == h
        and halt["blockstore_status_tip_before_fencing"] == h
        and halt["staged_block_id_hash"] == inventory["halt_hash"]
        and halt["staged_header_last_block_id_hash"] == inventory["anchor_hash"]
        and halt["transaction_count"] == 0
        and halt["subjective_seen_commit_canonical_field"] is False
        and halt["block_results_available"] is False
        and halt["application_applied"] is False
        and excluded["abci_last_applied_height"] == a
        and excluded["app_hash"] == inventory["post_anchor_hash"]
        and excluded["included_in_successor_inventory"] is False
    ):
        _fail("FINAL boundary differs from inventory v3")

    predecessor = release["predecessor"]
    genesis = _exact_object(
        final.get("genesis"),
        {"raw_file_sha256", "rpc_canonical_sha256"},
        "FINAL genesis evidence",
    )
    if not (
        genesis.get("raw_file_sha256") == predecessor["genesis_file_sha256"]
        and genesis.get("rpc_canonical_sha256") == inventory["rpc_genesis_hash"]
    ):
        _fail("FINAL genesis hashes differ from inventory/RELEASE")

    terminal = final.get("terminal_rpc_evidence")
    if not isinstance(terminal, dict) or set(terminal) != {
        "sources",
        "matching_payload_sha256",
        "matched_payloads",
    }:
        _fail("FINAL terminal RPC evidence has the wrong shape")
    sources = _exact_object(
        terminal["sources"], {"official_signer", "independent_observer"}, "FINAL RPC sources"
    )
    for role_key, prefix in (
        ("official_signer", "SIGNER"),
        ("independent_observer", "OBSERVER"),
    ):
        source = _exact_object(
            sources[role_key],
            {"status_json_sha256", "sha256_manifest_sha256"},
            f"FINAL {role_key}",
        )
        manifest_name = f"{prefix}-EVIDENCE-MANIFEST.json"
        if source != {
            "status_json_sha256": rpc_hashes[prefix]["status_json"],
            "sha256_manifest_sha256": _sha256(files[manifest_name]),
        }:
            _fail(f"FINAL {role_key} hashes differ from actual evidence")
    matching = _exact_object(
        terminal["matching_payload_sha256"],
        set(MATCHING_PAYLOAD_KEYS),
        "FINAL matching terminal payloads",
    )
    for key in MATCHING_PAYLOAD_KEYS:
        signer_hash = rpc_hashes["SIGNER"][key]
        observer_hash = rpc_hashes["OBSERVER"][key]
        if signer_hash != observer_hash or matching[key] != signer_hash:
            _fail(f"terminal signer/observer {key} raw bytes do not match")
    if terminal["matched_payloads"] != MATCHED_PAYLOAD_LABELS:
        _fail("FINAL matched terminal payload list changed")

    signer_key = manifests["SIGNER"]["validator_pubkey"]
    observer_key = manifests["OBSERVER"]["validator_pubkey"]
    if signer_key == observer_key:
        _fail("terminal signer and observer reused the same consensus key")
    if signer_key not in inventory["validator_keys"]:
        _fail("terminal signer key is not present in the bonded-validator inventory")
    if observer_key in inventory["validator_keys"]:
        _fail("terminal observer key is unexpectedly in the bonded-validator inventory")


def validate_frozen_evidence(
    files: Mapping[str, bytes],
    objects: Mapping[str, Any],
    final: Mapping[str, Any],
    release: Mapping[str, Any],
    transition: Mapping[str, Any],
    f: str,
    a: str,
    h: str,
) -> None:
    """Validate all public/raw frozen evidence and its signed FINAL joins.

    `f`, `a`, and `h` are canonical positive-height strings already selected by
    the signed CUTOVER decision.  The function is read-only and returns only on
    an exact match.
    """

    missing = REQUIRED_FROZEN_EVIDENCE_FILES - set(files)
    if missing:
        _fail(f"frozen evidence bundle is missing: {', '.join(sorted(missing))}")
    _height(f, "checkpoint F")
    _height(a, "anchor A")
    _height(h, "halt trigger H")
    if int(a) != int(f) + 1 or int(h) != int(a) + 1:
        _fail("frozen evidence does not satisfy A=F+1 and H=A+1")

    inventory_object = _object(files, objects, "ZERONE-1-INVENTORY-V3.json")
    inventory = _validate_inventory(inventory_object, release, f, a, h)
    if transition.get("expected_anchor_block_hash") != inventory["anchor_hash"]:
        _fail("transition anchor block hash differs from inventory")
    if transition.get("expected_post_anchor_app_hash") != inventory["post_anchor_hash"]:
        _fail("transition post-anchor AppHash differs from inventory")

    predecessor = release.get("predecessor")
    source_observer = transition.get("source_observer")
    source_evidence = transition.get("source_evidence")
    if not all(isinstance(value, dict) for value in (predecessor, source_observer, source_evidence)):
        _fail("RELEASE/transition source identities are missing")
    signer_node = _node_id(predecessor.get("trusted_rpc_node_id"), "trusted signer node ID")
    observer_node = _node_id(
        predecessor.get("trusted_observer_node_id"), "trusted observer node ID"
    )
    if source_observer.get("node_id") != observer_node or signer_node == observer_node:
        _fail("transition observer differs from RELEASE or reuses signer identity")
    trusted_block = _exact_object(
        predecessor.get("trusted_block"),
        {"height", "block_id_hash", "app_hash"},
        "RELEASE predecessor trusted block",
    )
    trusted_height = _height(
        trusted_block["height"], "RELEASE predecessor trusted height"
    )
    _upper_hash(
        trusted_block["block_id_hash"], "RELEASE predecessor trusted block hash"
    )
    _upper_hash(trusted_block["app_hash"], "RELEASE predecessor trusted AppHash")
    if int(trusted_height) >= int(a):
        _fail("RELEASE predecessor trusted block must predate terminal anchor A")
    inventory["trusted_height"] = trusted_height
    inventory["trusted_block_hash"] = trusted_block["block_id_hash"]
    inventory["trusted_app_hash"] = trusted_block["app_hash"]

    manifests = {
        "SIGNER": _object(files, objects, "SIGNER-EVIDENCE-MANIFEST.json"),
        "OBSERVER": _object(files, objects, "OBSERVER-EVIDENCE-MANIFEST.json"),
    }
    if source_evidence != {
        "signer_manifest_sha256": _sha256(files["SIGNER-EVIDENCE-MANIFEST.json"]),
        "observer_manifest_sha256": _sha256(files["OBSERVER-EVIDENCE-MANIFEST.json"]),
    }:
        _fail("transition source manifest hashes differ from actual files")
    _validate_terminal_manifest(
        manifests["SIGNER"],
        "official-signer",
        signer_node,
        inventory,
        release,
        transition,
        f,
        a,
        h,
    )
    _validate_terminal_manifest(
        manifests["OBSERVER"],
        "independent-observer",
        observer_node,
        inventory,
        release,
        transition,
        f,
        a,
        h,
    )
    if source_observer.get("validator_pubkey") != manifests["OBSERVER"]["validator_pubkey"]:
        _fail("transition observer consensus key differs from terminal evidence")
    candidate = transition.get("candidate")
    if not isinstance(candidate, dict):
        _fail("transition archive candidate identity is missing")
    candidate_node = _node_id(candidate.get("node_id"), "archive candidate node ID")
    _public_key(candidate.get("validator_pubkey"), "archive candidate validator key")
    if len(
        {
            signer_node,
            observer_node,
            candidate_node,
        }
    ) != 3 or len(
        {
            manifests["SIGNER"]["validator_pubkey"],
            manifests["OBSERVER"]["validator_pubkey"],
            candidate.get("validator_pubkey"),
        }
    ) != 3:
        _fail("signer, observer, and archive candidate identities are not distinct")

    rpc_hashes = {
        prefix: _validate_rpc_set(files, prefix, manifests[prefix], inventory, f, a, h)
        for prefix in ("SIGNER", "OBSERVER")
    }
    _validate_final_boundary(
        files, final, release, inventory, manifests, rpc_hashes, f, a, h
    )
    _validate_export_and_storage(
        files, objects, final, release, transition, inventory, f, a, h
    )
