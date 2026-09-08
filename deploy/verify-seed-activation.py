#!/usr/bin/env python3
"""Offline seed gate: authenticated declarations, NOT independent chain truth.

No signing, key discovery, network, activation, ledger writes or packet commands.
OpenPGP detached signatures follow CANONICAL-SIGNING.md; verification uses an
isolated gpgv home and only the explicit, hash-pinned public keyring. The existing
release verifier is not invoked or weakened. See networks/zerone-2/SEED-ACTIVATION.md.
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

# Help/default verification must not create source-tree bytecode caches.
sys.dont_write_bytecode = True

# Reuse the already-shipped, pure canonical SDK address codec, not authority gates.
from frozen_evidence import _bech32_address, _bech32_encode, FrozenEvidenceError

MAX_JSON = 262144
MAX_ARTIFACT = 512 * 1024 * 1024
CLAIM = "/zerone.claiming_pot.v1.MsgClaim"
ROLES = {"release", "activation", "reviewer", "observer", "custodian"}
STAGES = ("preactivation", "postactivation", "operation")
STOPS = ["artifact_drift", "authority_drift", "budget_shortfall", "custody_mismatch",
         "expired", "fork_or_stale", "ledger_failure", "signing_unknown", "submission_unknown"]
BOUNDARY = "configured_full_node_attestation_no_state_proof"
ARTIFACTS = {"runtime_sha256": "zeroned", "helper_sha256": "zerone-seed-io",
             "wallet_sha256": "wallet.tgz", "cli_sha256": "zerone-seed-cli",
             "source_digest": "source-manifest.json"}
PREREQUISITES = {"RELEASE-PACKET.json", "RELEASE-PACKET.json.sig",
                 "OPEN-BETA-DECISION.json", "OPEN-BETA-DECISION.json.sig",
                 "OPEN-BETA-INITIATION-EVIDENCE.json", "OPEN-BETA-INITIATION-EVIDENCE.json.sig",
                 "RELEASE-VERIFY.stdout", "OPEN-BETA-VERIFY.stdout",
                 "BETA-VERIFICATION.json", "BETA-VERIFICATION.json.sig"}
PROFILE_KEYS = set("protocol chain_reference chain_id native_asset_id claiming_pot_account genesis_hash source_digest zerone_core_commit cosmos_sdk_version runtime_sha256 helper_sha256 native_denom bech32_prefix seed_amount_uzrn claim_type_url claim_gas_floor tx_gas_cap min_gas_price_uzrn confirmation_depth profile_id".split())
POLICY_KEYS = set("protocol profile_id node_trust_id claimant_account sponsor_account pot_id max_intents seed_amount_uzrn max_fee_uzrn max_gas grant_spend_limit_uzrn grant_expires_at setup_fee_budget_uzrn not_before expires_at timeout_height max_observation_age_seconds max_height_lag policy_hash".split())
PARAM_KEYS = set("max_pots_active min_claim_amount bootstrap_registrar bootstrap_emission_cap_uzrn bootstrap_daily_admission_cap".split())


class Refusal(ValueError):
    pass


def require(condition, label):
    if not condition:
        raise Refusal(label)


def exact(value, keys, label):
    require(type(value) is dict and set(value) == set(keys), label + ": exact fields required")
    return value


def digest(data):
    return "sha256:" + hashlib.sha256(data).hexdigest()


def hash_id(value):
    require(type(value) is str and re.fullmatch(r"sha256:[0-9a-f]{64}", value), "invalid digest")
    return value


def canonical(value):
    # Packet/public strings are ASCII; integers are bounded, so this is also
    # Wallet canonicalJsonBytes. Ceremony payload files additionally carry LF.
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


def same(left, right):
    # Python considers True == 1; canonical closed-contract equality must not.
    return canonical(left) == canonical(right)


def semantic_id(value, field):
    return digest(canonical({k: v for k, v in value.items() if k != field}))


def parse(raw, *, ceremony=True, public_strings=True):
    require(0 < len(raw) <= MAX_JSON, "JSON byte bound")
    def pairs(items):
        result = {}
        for key, value in items:
            require(key not in result, "duplicate JSON member")
            result[key] = value
        return result
    def bad_number(_):
        raise Refusal("non-integer JSON number")
    try:
        value = json.loads(raw, object_pairs_hook=pairs, parse_float=bad_number,
                           parse_constant=bad_number)
        count = 0
        def walk(item, depth=0):
            nonlocal count
            count += 1
            require(depth <= 32 and count <= 4096, "JSON structure bound")
            if type(item) is str:
                require(len(item.encode("utf-8")) <= 4096 and "\x00" not in item, "bounded string required")
                if public_strings:
                    require(all(32 <= ord(c) <= 126 for c in item), "public strings must be printable ASCII")
                require("REPLACE" not in item, "unresolved placeholder")
            elif type(item) is int:
                require(abs(item) <= 2**53 - 1, "unsafe JSON integer")
            elif type(item) is dict:
                for k, v in item.items():
                    walk(k, depth + 1)
                    walk(v, depth + 1)
            elif type(item) is list:
                for v in item:
                    walk(v, depth + 1)
            else:
                require(item is None or type(item) is bool, "unsupported JSON value")
        walk(value)
        require(raw == canonical(value) + (b"\n" if ceremony else b""), "noncanonical JSON bytes")
        return value
    except (UnicodeError, json.JSONDecodeError, RecursionError, OverflowError) as exc:
        raise Refusal("invalid bounded canonical JSON") from exc


def uint(value, bits=256, positive=False):
    require(type(value) is str and re.fullmatch(r"0|[1-9][0-9]{0,77}", value), "noncanonical decimal")
    n = int(value)
    require(n < 2**bits and n >= int(positive), "decimal out of bounds")
    return n


def integer(value, low, high):
    require(type(value) is int and low <= value <= high, "integer out of bounds")
    return value


def timestamp(value):
    require(type(value) is str and re.fullmatch(r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}Z", value), "millisecond UTC required")
    try:
        parsed = dt.datetime.fromisoformat(value[:-1] + "+00:00")
        require(parsed.isoformat(timespec="milliseconds").replace("+00:00", "Z") == value, "timestamp does not roundtrip")
        return parsed
    except ValueError as exc:
        raise Refusal("invalid timestamp") from exc


def address(value):
    try:
        require(len(_bech32_address(value, "zrn", "address")) == 20, "account must be 20 bytes")
    except FrozenEvidenceError as exc:
        raise Refusal("invalid zrn address") from exc
    return value


def account(value, chain):
    require(type(value) is str and value.startswith(chain + ":"), "account chain mismatch")
    return address(value[len(chain) + 1:])


def identifier(value):
    require(type(value) is str and re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,63}", value), "invalid opaque identifier")
    return value


def safe_open(path):
    # All ancestors must be real directories. /tmp on macOS is a symlink;
    # callers can explicitly select /private/tmp instead. Never resolve aliases.
    path = Path(path)
    require(path.is_absolute() and ".." not in path.parts, "explicit absolute path required")
    parent_fd = os.open("/", os.O_RDONLY | os.O_DIRECTORY)
    try:
        for component in path.parts[1:-1]:
            child_fd = os.open(component, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent_fd)
            os.close(parent_fd)
            parent_fd = child_fd
        fd = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent_fd)
    finally:
        os.close(parent_fd)
    info = os.fstat(fd)
    if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
        os.close(fd)
        raise Refusal("input must be a regular single-link file")
    return fd, info


def stable_stat(info):
    return (info.st_dev, info.st_ino, info.st_mode, info.st_size, info.st_mtime_ns, info.st_ctime_ns, info.st_nlink)


def read(path, limit=MAX_JSON):
    fd, info = safe_open(path)
    with os.fdopen(fd, "rb") as stream:
        require(info.st_size <= limit, "input size bound")
        data = stream.read(limit + 1)
        require(len(data) <= limit and len(data) == info.st_size, "input changed or exceeded bound")
        require(stable_stat(os.fstat(stream.fileno())) == stable_stat(info), "input changed while reading")
        return data


def file_digest(path):
    fd, info = safe_open(path)
    h = hashlib.sha256()
    with os.fdopen(fd, "rb") as stream:
        require(0 < info.st_size <= MAX_ARTIFACT, "artifact size bound")
        length = 0
        while chunk := stream.read(1024 * 1024):
            length += len(chunk)
            require(length <= MAX_ARTIFACT, "artifact grew")
            h.update(chunk)
        require(length == info.st_size and stable_stat(os.fstat(stream.fileno())) == stable_stat(info), "artifact changed")
    return "sha256:" + h.hexdigest()


class SignatureVerifier:
    """Only fixed gpgv argv over frozen public bytes; no ambient keyring/config."""
    def __init__(self, executable, keyring, roles, now):
        require(Path(executable).is_absolute(), "explicit gpgv path required")
        # Executable comes from operator CLI, never a packet/trust command.
        self.executable = executable
        self.keyring = keyring
        self.roles = exact(roles, ROLES, "signer roles")
        self.now = now
        for fp in roles.values():
            require(type(fp) is str and re.fullmatch(r"(?:[0-9A-F]{40}|[0-9A-F]{64})", fp), "full uppercase signing fingerprint required")
        require(len(set(roles.values())) == len(ROLES), "signer roles must use distinct keys")

    def verify(self, payload, signature, role):
        require(0 < len(signature) <= 16384, "signature size bound")
        with tempfile.TemporaryDirectory(prefix="seed-public-verify-") as tmp:
            root = Path(tmp)
            (root / "public.gpg").write_bytes(self.keyring)
            (root / "payload").write_bytes(payload)
            (root / "payload.sig").write_bytes(signature)
            result = subprocess.run(
                [self.executable, "--homedir", str(root), "--keyring", str(root / "public.gpg"),
                 "--status-fd=1", "--", str(root / "payload.sig"), str(root / "payload")],
                stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                timeout=15, env={"PATH": "/usr/bin:/bin", "LC_ALL": "C"}, check=False)
        require(result.returncode == 0 and len(result.stdout) <= 16384, "detached signature failed")
        lines = result.stdout.decode("ascii", errors="strict").splitlines()
        valid = [line.split() for line in lines if line.startswith("[GNUPG:] VALIDSIG ")]
        require(len(valid) == 1 and len(valid[0]) >= 11, "exactly one valid signature required")
        sig = valid[0]
        require(sig[2] == self.roles[role], "wrong role signing key")
        require(sig[4].isdigit() and 0 < int(sig[4]) <= self.now.timestamp() + 300, "invalid/future signature time")
        require(sig[9] in {"8", "9", "10", "11"}, "SHA-2 signature digest required")
        require(not any(any(tag in line for tag in ("BADSIG", "ERRSIG", "EXPKEYSIG", "EXPSIG", "REVKEYSIG")) for line in lines), "invalid/expired signing key")


def validate_profile(p, env):
    exact(p, PROFILE_KEYS, "SeedProfile")
    require(p["protocol"] == "agent-wallet-zerone.seed-profile/0.1", "profile version")
    ref = p["chain_reference"]
    require(type(ref) is str and re.fullmatch(r"[A-Za-z0-9_-]{1,32}", ref), "chain reference")
    require((ref == "zerone-2") if env == "production" else ref.startswith(("seed-synthetic-", "seed-local-")), "environment chain boundary")
    require(p["chain_id"] == "cosmos:" + ref and p["native_asset_id"] == "cosmos:" + ref + "/denom:uzrn", "chain/asset mismatch")
    module = _bech32_encode("zrn", hashlib.sha256(b"claiming_pot").digest()[:20])
    require(p["claiming_pot_account"] == p["chain_id"] + ":" + module, "wrong claiming-pot module")
    for key in ("genesis_hash", "source_digest", "runtime_sha256", "helper_sha256", "profile_id"):
        hash_id(p[key])
    require(type(p["zerone_core_commit"]) is str and re.fullmatch(r"[0-9a-f]{40}", p["zerone_core_commit"]), "source commit")
    expected = {"cosmos_sdk_version": "v0.53.8", "native_denom": "uzrn", "bech32_prefix": "zrn",
                "seed_amount_uzrn": "222000", "claim_type_url": CLAIM, "claim_gas_floor": "22222",
                "tx_gas_cap": "11111111", "min_gas_price_uzrn": "1", "confirmation_depth": 1}
    require(all(type(p[k]) is type(v) and p[k] == v for k, v in expected.items()), "source-pinned profile constants")
    require(p["profile_id"] == semantic_id(p, "profile_id"), "profile semantic hash")


def validate_params(p):
    exact(p, PARAM_KEYS, "complete native parameters")
    integer(p["max_pots_active"], 1, 2**32 - 1)
    uint(p["min_claim_amount"], positive=True)
    uint(p["bootstrap_emission_cap_uzrn"], positive=True)
    uint(p["bootstrap_daily_admission_cap"], 64, True)
    if p["bootstrap_registrar"] != "":
        address(p["bootstrap_registrar"])


def validate_packet(p, trust, now):
    exact(p, set("schema environment activation_id trust_sha256 profile artifacts prerequisites admission sponsor custody recipients budgets not_before expires_at admission_deadline stop_conditions nonexpiring_pot_exposure_accepted unrelated_policy_hash".split()), "activation packet")
    require(p["schema"] == "zerone-seed-activation/v1", "activation schema")
    require(p["environment"] == trust["environment"] and p["activation_id"] == trust["activation_id"], "activation trust binding")
    identifier(p["activation_id"])
    validate_profile(p["profile"], p["environment"])
    profile = p["profile"]
    require(profile["profile_id"] == trust["profile_id"] and profile["chain_id"] == trust["chain_id"] and profile["genesis_hash"] == trust["genesis_hash"], "independent profile/chain/genesis pin")
    begin, end, admission_end = (timestamp(p[k]) for k in ("not_before", "expires_at", "admission_deadline"))
    require(begin <= now < end and begin < admission_end <= end and end - begin <= dt.timedelta(days=7), "activation expired/not yet valid or exceeds seven days")
    require(p["stop_conditions"] == STOPS and p["nonexpiring_pot_exposure_accepted"] is True, "stop/irreversible pot exposure acknowledgement")
    hash_id(p["unrelated_policy_hash"])
    exact(p["artifacts"], set(ARTIFACTS) | {"verifier_sha256"}, "artifact pins")
    for k, v in p["artifacts"].items():
        hash_id(v)
        if k in profile:
            require(v == profile[k], "profile/artifact pin mismatch")
    exact(p["prerequisites"], PREREQUISITES, "prerequisite pins")
    for value in p["prerequisites"].values():
        hash_id(value)
    require(p["prerequisites"]["BETA-VERIFICATION.json"] == trust["beta_verification_sha256"], "independent beta receipt pin")
    admission = exact(p["admission"], {"mechanism", "authority", "route_evidence_sha256", "params_before", "params_after"}, "admission")
    require(admission["mechanism"] in {"governance", "registrar"}, "unsupported native admission mechanism")
    address(admission["authority"])
    hash_id(admission["route_evidence_sha256"])
    validate_params(admission["params_before"])
    validate_params(admission["params_after"])
    before, after = admission["params_before"], admission["params_after"]
    # The only v1 parameter delta is installation of the exact bounded registrar.
    # Full objects are signed; all unrelated parameters and economics remain fixed.
    require(all(before[k] == after[k] for k in PARAM_KEYS - {"bootstrap_registrar"}), "unrelated parameter delta")
    if admission["mechanism"] == "governance":
        gov = _bech32_encode("zrn", hashlib.sha256(b"gov").digest()[:20])
        require(admission["authority"] == gov and before == after, "governance identity/parameter delta")
    else:
        require(after["bootstrap_registrar"] == admission["authority"] and before["bootstrap_registrar"] in {"", admission["authority"]}, "registrar installation/rotation mismatch")
    sponsor = exact(p["sponsor"], {"account", "dedicated", "single_signer_host_id"}, "sponsor")
    sponsor_raw = account(sponsor["account"], profile["chain_id"])
    require(sponsor["dedicated"] is True, "dedicated sponsor required")
    identifier(sponsor["single_signer_host_id"])
    exact(p["custody"], {"sponsor", "admission", "claimant_records"}, "custody identifiers")
    for binding in p["custody"].values():
        identifier(binding)  # IDs only, not private paths, keys, URLs or commands.
    recipients = p["recipients"]
    require(type(recipients) is list and 1 <= len(recipients) <= 32, "bounded nonempty cohort required")
    names, grants, setups = [], 0, 0
    for policy in recipients:
        exact(policy, POLICY_KEYS, "SeedPolicy")
        require(policy["protocol"] == "agent-wallet-zerone.seed-policy/0.1" and policy["profile_id"] == profile["profile_id"], "policy profile/version")
        claimant = account(policy["claimant_account"], profile["chain_id"])
        require(claimant != sponsor_raw and policy["sponsor_account"] == sponsor["account"] and policy["pot_id"] == "bootstrap-" + claimant, "policy parties/pot")
        names.append(claimant)
        require(type(policy["max_intents"]) is int and policy["max_intents"] == 1 and policy["seed_amount_uzrn"] == "222000", "one use/native seed only")
        hash_id(policy["node_trust_id"])
        require(policy["node_trust_id"] == trust["node_trust_id"], "node trust mismatch")
        fee = uint(policy["max_fee_uzrn"], positive=True)
        gas = uint(policy["max_gas"], 64, True)
        grant = uint(policy["grant_spend_limit_uzrn"], positive=True)
        require(22222 <= gas <= 11111111 and fee <= grant, "independent gas/fee/full grant bounds")
        uint(policy["timeout_height"], 64, True)
        integer(policy["max_observation_age_seconds"], 1, 300)
        integer(policy["max_height_lag"], 0, 100)
        pn, pe, ge = (timestamp(policy[k]) for k in ("not_before", "expires_at", "grant_expires_at"))
        require(begin <= pn < pe <= end and pe <= ge <= end, "finite policy/grant deadlines")
        require(policy["policy_hash"] == semantic_id(policy, "policy_hash"), "Wallet-compatible policy semantic hash")
        grants += grant
        setups += uint(policy["setup_fee_budget_uzrn"])
    require(names == sorted(set(names)), "recipients must be uniquely sorted")
    budget = exact(p["budgets"], {"issuance_commitment_uzrn", "full_grant_exposure_uzrn", "setup_fee_budget_uzrn", "total_sponsor_exposure_uzrn", "lifetime_commitment_cap_uzrn"}, "budgets")
    totals = {k: uint(v) for k, v in budget.items()}
    require(totals["issuance_commitment_uzrn"] == 222000 * len(names) and totals["full_grant_exposure_uzrn"] == grants and totals["setup_fee_budget_uzrn"] == setups and totals["total_sponsor_exposure_uzrn"] == grants + setups, "aggregate commitments/full gas/setup mismatch")
    require(totals["issuance_commitment_uzrn"] <= totals["lifetime_commitment_cap_uzrn"] <= uint(after["bootstrap_emission_cap_uzrn"]), "native/approved lifetime commitment cap")


def validate_prerequisites(p, files, sigs, trust):
    for name, expected in p["prerequisites"].items():
        require(digest(files[name]) == expected, "prerequisite bytes changed: " + name)
    release, opened, initiated = [parse(files[name]) for name in ("RELEASE-PACKET.json", "OPEN-BETA-DECISION.json", "OPEN-BETA-INITIATION-EVIDENCE.json")]
    for name, doc in zip(("RELEASE-PACKET.json", "OPEN-BETA-DECISION.json", "OPEN-BETA-INITIATION-EVIDENCE.json"), (release, opened, initiated)):
        sigs.verify(files[name], files[name + ".sig"], "release")
        authority = exact(doc.get("signature_authority"), {"algorithm", "authorized_signer_fingerprint", "detached_signature_filename"}, "public document signature")
        require(authority == {"algorithm": "openpgp", "authorized_signer_fingerprint": trust["roles"]["release"], "detached_signature_filename": name + ".sig"}, "public signer declaration")
    profile = p["profile"]
    # Existing signed schemas are owned by the release verifier, not redefined here.
    # Only the required joins are inspected; its successful execution is attested below.
    require(release.get("schema") == "zerone-2-release-packet-v2" and release.get("chain_id") == profile["chain_reference"], "RELEASE chain/schema")
    require(release.get("genesis", {}).get("sha256") == profile["genesis_hash"][7:] and release.get("source", {}).get("commit") == profile["zerone_core_commit"], "RELEASE genesis/source mismatch")
    require(release.get("components", {}).get("zerone_2_runtime", {}).get("binary_sha256") == profile["runtime_sha256"][7:], "RELEASE runtime mismatch")
    require(release.get("accepted_policy", {}).get("claims_live_at_genesis") is False, "dark genesis claims boundary")
    require(digest(canonical(release["accepted_policy"])) == p["unrelated_policy_hash"], "signed RELEASE policy hash mismatch")
    require(opened.get("schema") == "zerone-2-open-beta-decision-v1" and opened.get("decision") == "GO", "not an OPEN-BETA GO document")
    require(opened.get("release_packet") == {"sha256": digest(files["RELEASE-PACKET.json"])[7:], "detached_signature_sha256": digest(files["RELEASE-PACKET.json.sig"])[7:]}, "OPEN/RELEASE join")
    require(opened.get("history_link_transaction", {}).get("chain_id") == profile["chain_reference"], "OPEN chain mismatch")
    require(initiated.get("schema") == "zerone-2-open-beta-initiation-evidence-v1" and initiated.get("attestation_result") == "MATCH" and initiated.get("deadline_satisfied") is True, "OPEN initiation schema/result")
    require(initiated.get("open_beta_decision") == {"sha256": digest(files["OPEN-BETA-DECISION.json"])[7:], "detached_signature_sha256": digest(files["OPEN-BETA-DECISION.json.sig"])[7:]}, "OPEN initiation/decision join")
    tx = initiated.get("history_link_transaction", {})
    expected_tx = opened.get("history_link_transaction", {}).get("expected_transaction_hash")
    require(type(expected_tx) is str and re.fullmatch(r"[0-9A-F]{64}", expected_tx) and tx.get("committed_transaction_hash") == expected_tx and tx.get("expected_transaction_hash") == expected_tx and type(tx.get("deliver_code")) is int and tx["deliver_code"] == 0, "OPEN history-link confirmation mismatch")
    receipt = parse(files["BETA-VERIFICATION.json"])
    exact(receipt, {"schema", "environment", "chain_id", "genesis_hash", "verifier_sha256", "bundle_manifest_sha256", "verified_at", "documents", "runs", "dark_genesis", "unrelated_policy_hash"}, "independent beta execution receipt")
    sigs.verify(files["BETA-VERIFICATION.json"], files["BETA-VERIFICATION.json.sig"], "reviewer")
    require(receipt["schema"] == "zerone-seed-beta-verification/v1" and receipt["environment"] == p["environment"] and receipt["chain_id"] == profile["chain_id"] and receipt["genesis_hash"] == profile["genesis_hash"], "beta receipt scope")
    require(timestamp(receipt["verified_at"]) <= timestamp(p["not_before"]), "seed must follow beta verification")
    for k in ("verifier_sha256", "bundle_manifest_sha256"):
        require(hash_id(receipt[k]) == trust[k], "independent prerequisite tool/bundle pin")
    expected_docs = {name: p["prerequisites"][name] for name in PREREQUISITES if name not in {"BETA-VERIFICATION.json", "BETA-VERIFICATION.json.sig", "RELEASE-VERIFY.stdout", "OPEN-BETA-VERIFY.stdout"}}
    require(receipt["documents"] == expected_docs, "beta receipt public-document joins")
    expected_runs = []
    for stage, name in (("dark-preinit", "RELEASE-VERIFY.stdout"), ("open-postinit", "OPEN-BETA-VERIFY.stdout")):
        require(files[name] == b"authority-chain: MATCH\n", "not the actual authority verifier success output")
        expected_runs.append({"stage": stage, "exit_code": 0, "stdout_sha256": digest(files[name])})
    require(same(receipt["runs"], expected_runs), "beta execution stages/results")
    require(same(receipt["dark_genesis"], {"claims_live_at_genesis": False, "bootstrap_registrar": "", "pots": []}), "dark genesis invariant")
    require(receipt["unrelated_policy_hash"] == p["unrelated_policy_hash"], "unrelated policy not preserved")


def confirmation(c, anchor_height):
    exact(c, {"tx_hash", "height", "block_hash", "block_time", "successor_height", "successor_block_hash", "code", "fee_uzrn", "evidence_sha256"}, "confirmed native evidence")
    timestamp(c["block_time"])
    require(type(c["tx_hash"]) is str and re.fullmatch(r"[0-9A-F]{64}", c["tx_hash"]), "transaction hash")
    height = uint(c["height"], 64, True)
    require(uint(c["successor_height"], 64, True) == height + 1 <= anchor_height, "canonical successor evidence required")
    for k in ("block_hash", "successor_block_hash", "evidence_sha256"):
        hash_id(c[k])
    require(type(c["code"]) is int and c["code"] == 0, "native inclusion failed")
    uint(c["fee_uzrn"], positive=True)


def bind_block(blocks, height, block_hash, block_time=None):
    uint(height, 64, True)
    hash_id(block_hash)
    if block_time is not None:
        timestamp(block_time)
    old = blocks.get(height)
    if old:
        require(old[0] == block_hash and (old[1] is None or block_time is None or old[1] == block_time), "conflicting same-height block binding")
    blocks[height] = (block_hash, block_time if block_time is not None else old[1] if old else None)


def bind_observed_blocks(e, blocks):
    """Join all attested heights/hashes/times, not unprovided header proofs."""
    a = e["anchor"]
    bind_block(blocks, a["height"], a["block_hash"], a["block_time"])
    confirmations = [e["parameter_update_confirmation"]]
    for row in e["recipients"]:
        confirmations.extend(row[k] for k in ("admission_confirmation", "grant_confirmation"))
    for c in confirmations:
        if c is not None:
            bind_block(blocks, c["height"], c["block_hash"], c["block_time"])
            bind_block(blocks, c["successor_height"], c["successor_block_hash"])


def validate_evidence(e, p, packet_hash, stage, now, trust, prior=None, blocks=None):
    exact(e, set("schema environment activation_sha256 stage previous_evidence_sha256 trust_model node_trust_id profile_id chain_id genesis_hash observed_at anchor latest_height catching_up coherent parameters native_authority prior_commitment_uzrn observed_total_commitment_uzrn bootstrap_window_count sponsor_balance_uzrn sponsor_other_exposure_uzrn setup_spent_uzrn unrelated_policy_hash recipients stop_reasons operation parameter_update_confirmation".split()), "stage evidence")
    profile = p["profile"]
    require(e["schema"] == "zerone-seed-evidence/v1" and e["environment"] == p["environment"] and e["activation_sha256"] == packet_hash and e["stage"] == stage, "stage/packet evidence binding")
    require(e["trust_model"] == BOUNDARY and e["node_trust_id"] == trust["node_trust_id"] and e["profile_id"] == profile["profile_id"] and e["chain_id"] == profile["chain_id"] and e["genesis_hash"] == profile["genesis_hash"], "observation trust/chain binding")
    require(e["previous_evidence_sha256"] == (digest(canonical(prior) + b"\n") if prior else None), "evidence predecessor binding")
    observed = timestamp(e["observed_at"])
    require(timestamp(p["not_before"]) <= observed < timestamp(p["expires_at"]), "observation outside activation lifetime")
    anchor = exact(e["anchor"], {"height", "block_hash", "block_time"}, "coherent anchor")
    h = uint(anchor["height"], 64, True)
    hash_id(anchor["block_hash"])
    block_time = timestamp(anchor["block_time"])
    age = min(x["max_observation_age_seconds"] for x in p["recipients"])
    lag = min(x["max_height_lag"] for x in p["recipients"])
    require(block_time <= observed <= now and now - block_time <= dt.timedelta(seconds=age), "stale/future observation")
    require(h <= uint(e["latest_height"], 64, True) <= h + lag and e["catching_up"] is False and e["coherent"] is True, "incomplete/incoherent node observation")
    require(e["stop_reasons"] == [] and e["unrelated_policy_hash"] == p["unrelated_policy_hash"], "stop condition/policy drift")
    params = p["admission"]["params_before" if stage == "preactivation" else "params_after"]
    require(same(e["parameters"], params), "full parameter observation mismatch")
    native = exact(e["native_authority"], {"mechanism", "authority", "route_evidence_sha256", "status"}, "native authority observation")
    require(native == {k: p["admission"][k] for k in ("mechanism", "authority", "route_evidence_sha256")} | {"status": "available"}, "native authority unavailable: address alone is not execution authority")
    prior_commitment = uint(e["prior_commitment_uzrn"])
    require(prior_commitment + uint(p["budgets"]["issuance_commitment_uzrn"]) <= uint(p["budgets"]["lifetime_commitment_cap_uzrn"]), "lifetime commitments exceed approved cap")
    require(uint(e["observed_total_commitment_uzrn"]) == prior_commitment + (0 if stage == "preactivation" else uint(p["budgets"]["issuance_commitment_uzrn"])), "actual lifetime commitment drift")
    window_count = uint(e["bootstrap_window_count"], 64)
    if stage == "preactivation" and p["admission"]["mechanism"] == "registrar":
        require(window_count + len(p["recipients"]) <= uint(params["bootstrap_daily_admission_cap"], 64), "native registrar window cap")
    if prior:
        require(h // 34272 == uint(prior["anchor"]["height"], 64) // 34272, "window rollover requires new reviewed observation")
        added = len(p["recipients"]) if stage == "postactivation" and p["admission"]["mechanism"] == "registrar" else 0
        require(window_count == uint(prior["bootstrap_window_count"], 64) + added, "native admission window drift")
    other_exposure = uint(e["sponsor_other_exposure_uzrn"])  # other cohorts/outside this cohort; host must reconcile its exact journal
    spent = uint(e["setup_spent_uzrn"])
    require(spent <= uint(p["budgets"]["setup_fee_budget_uzrn"]), "setup costs exhausted")
    require(uint(e["sponsor_balance_uzrn"]) >= uint(p["budgets"]["total_sponsor_exposure_uzrn"]) + other_exposure, "sponsor cannot cover full lifetime grants plus setup and other exposure")
    if prior:
        previous_height = uint(prior["anchor"]["height"], 64)
        require(h >= previous_height and observed >= timestamp(prior["observed_at"]) and prior_commitment == uint(prior["prior_commitment_uzrn"]), "observation history/commitment drift")
        require(h != previous_height or same(anchor, prior["anchor"]), "conflicting same-height anchor")
    if stage == "preactivation":
        require(now < timestamp(p["admission_deadline"]) and spent == 0 and e["parameter_update_confirmation"] is None, "admission deadline/prior setup effects")
    elif p["admission"]["params_before"] != p["admission"]["params_after"]:
        confirmation(e["parameter_update_confirmation"], h)
        update_time = timestamp(e["parameter_update_confirmation"]["block_time"])
        require(timestamp(p["not_before"]) <= update_time <= block_time and update_time < timestamp(p["admission_deadline"]), "parameter update outside admission window")
        if stage == "postactivation":
            require(uint(e["parameter_update_confirmation"]["height"], 64) > previous_height, "parameter update must follow preactivation")
        else:
            require(same(e["parameter_update_confirmation"], prior["parameter_update_confirmation"]), "parameter update evidence changed")
    else:
        require(e["parameter_update_confirmation"] is None, "unapproved parameter update")
    rows = e["recipients"]
    policies = p["recipients"]
    if stage == "operation":
        require(type(e["operation"]) is dict, "operation object required")
        policies = [policy for policy in policies if policy["policy_hash"] == e["operation"].get("policy_hash")]
        require(len(policies) == 1, "operation outside exact cohort")
    require(type(rows) is list and len(rows) == len(policies), "complete stage recipient observations required")
    confirmed_fees = uint(e["parameter_update_confirmation"]["fee_uzrn"]) if e["parameter_update_confirmation"] else 0
    tx_hashes = {e["parameter_update_confirmation"]["tx_hash"]} if e["parameter_update_confirmation"] else set()
    for row, policy in zip(rows, policies):
        exact(row, {"policy_hash", "pot", "allowance", "account_exists", "prior_claim", "admission_confirmation", "grant_confirmation"}, "recipient observation")
        require(row["policy_hash"] == policy["policy_hash"] and type(row["account_exists"]) is bool and row["prior_claim"] is None, "recipient policy/previous claim")
        if stage == "preactivation":
            require(all(row[k] is None for k in ("pot", "allowance", "admission_confirmation", "grant_confirmation")), "existing admission/grant prevents replay or replacement")
            continue
        require(row["account_exists"] is True, "GrantAllowance account creation unconfirmed")
        pot = exact(row["pot"], {"pot_id", "status", "total_amount_uzrn", "claimed_amount_uzrn", "start_block", "end_block", "cliff_blocks", "period_blocks", "min_staking_tier", "min_registration_age", "whitelist"}, "native seed pot")
        start = uint(pot["start_block"], 64, True)
        integer(pot["min_staking_tier"], 0, 0)
        require(pot == {"pot_id": policy["pot_id"], "status": "active", "total_amount_uzrn": "222000", "claimed_amount_uzrn": "0", "start_block": str(start), "end_block": str(start + 1), "cliff_blocks": "0", "period_blocks": "0", "min_staking_tier": 0, "min_registration_age": "0", "whitelist": [account(policy["claimant_account"], profile["chain_id"])]}, "native seed pot shape")
        require(start + 1 <= h and uint(params["min_claim_amount"]) <= 222000, "vesting/minimum not ready")
        allowance = {"type_url": "/cosmos.feegrant.v1beta1.AllowedMsgAllowance", "inner_type_url": "/cosmos.feegrant.v1beta1.BasicAllowance", "granter": account(policy["sponsor_account"], profile["chain_id"]), "grantee": account(policy["claimant_account"], profile["chain_id"]), "allowed_messages": [CLAIM], "spend_limit_uzrn": policy["grant_spend_limit_uzrn"], "expires_at": policy["grant_expires_at"]}
        require(row["allowance"] == allowance and now < timestamp(policy["grant_expires_at"]), "finite exact claim-only grant required")
        for k in ("admission_confirmation", "grant_confirmation"):
            c = row[k]
            confirmation(c, h)
            deadline = p["admission_deadline"] if k == "admission_confirmation" else policy["grant_expires_at"]
            require(timestamp(p["not_before"]) <= timestamp(c["block_time"]) <= block_time and timestamp(c["block_time"]) < timestamp(deadline), "setup confirmation outside approved time window")
            if stage == "postactivation":
                require(uint(c["height"], 64) > uint(prior["anchor"]["height"], 64), "setup must follow preactivation snapshot")
            require(c["tx_hash"] not in tx_hashes, "duplicate setup transaction evidence")
            tx_hashes.add(c["tx_hash"])
            confirmed_fees += uint(c["fee_uzrn"])
        require(uint(row["admission_confirmation"]["height"], 64) == start, "pot admission height mismatch")
        if e["parameter_update_confirmation"]:
            require(uint(e["parameter_update_confirmation"]["height"], 64) <= start and timestamp(e["parameter_update_confirmation"]["block_time"]) <= timestamp(row["admission_confirmation"]["block_time"]), "registrar installation must precede admission")
        require(sum(uint(row[k]["fee_uzrn"]) for k in ("admission_confirmation", "grant_confirmation")) <= uint(policy["setup_fee_budget_uzrn"]), "recipient setup budget")
    if stage == "operation":
        require(spent == uint(prior["setup_spent_uzrn"]), "unapproved subsequent setup spend")
        previous_row = next(row for row in prior["recipients"] if row["policy_hash"] == policies[0]["policy_hash"])
        require(all(canonical(rows[0][k]) == canonical(previous_row[k]) for k in ("admission_confirmation", "grant_confirmation")), "operation setup evidence changed")
    else:
        require(confirmed_fees == spent, "setup spend lacks exact confirmed fee evidence")
    blocks = {} if blocks is None else blocks
    if prior:
        bind_observed_blocks(prior, blocks)
    bind_observed_blocks(e, blocks)
    if stage != "operation":
        require(e["operation"] is None, "unexpected operation")
        return
    digest_keys = set("operation_id policy_hash plan_id commitment_hash descriptor_id capability_record_id intent_record_id simulation_record_id signer_key_id ledger_snapshot_sha256 ledger_binding_hash profile_id source_digest bundle_hash observation_hash currentness_sha256 budget_sha256 sign_doc_bytes_hash".split())
    op = exact(e["operation"], digest_keys | set("host_id ledger_id request_id prepared_at account_number sequence fee_uzrn gas_limit timeout_height expires_at attempt_status ledger_available max_intents_used required_grant_exposure_uzrn required_setup_exposure_uzrn".split()), "per-operation evidence")
    for k in digest_keys:
        hash_id(op[k])
    for k in ("host_id", "ledger_id", "request_id"):
        identifier(op[k])
    require(op["host_id"] == p["sponsor"]["single_signer_host_id"] and op["profile_id"] == profile["profile_id"] and op["source_digest"] == profile["source_digest"], "operation host/source/profile mismatch")
    require(timestamp(p["not_before"]) <= timestamp(op["prepared_at"]) <= now, "operation preparation time")
    require(op["operation_id"] == semantic_id(op, "operation_id"), "operation semantic commitment mismatch")
    matches = [policy for policy in p["recipients"] if policy["policy_hash"] == op["policy_hash"]]
    require(len(matches) == 1, "operation outside exact cohort")
    policy = matches[0]
    operation_expiry = timestamp(op["expires_at"])
    require(timestamp(policy["not_before"]) < operation_expiry <= timestamp(policy["expires_at"]) and timestamp(policy["not_before"]) <= now < operation_expiry and uint(e["latest_height"], 64) < uint(policy["timeout_height"], 64), "operation expired/height deadline")
    for k in ("account_number", "sequence"):
        uint(op[k], 64)
    fee, gas = uint(op["fee_uzrn"], positive=True), uint(op["gas_limit"], 64, True)
    require(22222 <= gas <= uint(policy["max_gas"], 64) and gas <= fee <= uint(policy["max_fee_uzrn"]) and op["timeout_height"] == policy["timeout_height"], "operation gas/fee/timeout bounds")
    require(op["attempt_status"] == "unreserved" and op["ledger_available"] is True and type(op["max_intents_used"]) is int and op["max_intents_used"] == 0, "consumed/unknown attempt is sticky; no replay")
    require(op["required_grant_exposure_uzrn"] == policy["grant_spend_limit_uzrn"] and op["required_setup_exposure_uzrn"] == policy["setup_fee_budget_uzrn"], "reserve full approved exposure, never claimant balance")


def validate_presign(candidate, e, p, blocks, now):
    """Authenticate exact unreserved runtime bytes; Wallet/native validity is
    rederived by the host under its lock, not asserted by this offline parser.
    """
    exact(candidate, {"protocol", "bundle", "observation", "operation"}, "runtime presign candidate")
    require(candidate["protocol"] == "zerone-seed-runtime.presign/0.1" and same(candidate["operation"], e["operation"]), "presign operation binding")
    op = e["operation"]
    b = exact(candidate["bundle"], set("plan observation prepared_at descriptor capability intent simulation simulation_result".split()), "original runtime bundle")
    obs = candidate["observation"]
    require(digest(canonical(b)) == op["bundle_hash"] and digest(canonical(obs)) == op["observation_hash"], "presign bundle/observation commitment")
    require(b["prepared_at"] == op["prepared_at"], "original preparation time changed")
    plan = b["plan"]
    require(plan["plan_id"] == op["plan_id"] == semantic_id(plan, "plan_id") and plan["commitment_hash"] == op["commitment_hash"] == digest(canonical(plan["commitment"])), "presign plan commitment")
    require(plan["observation_hash"] == digest(canonical(b["observation"])) and plan["sign_doc_bytes_hash"] == op["sign_doc_bytes_hash"], "original plan observation/bytes")
    for field, record in (("descriptor_id", "descriptor"), ("capability_record_id", "capability"), ("intent_record_id", "intent"), ("simulation_record_id", "simulation")):
        require(b[record]["record_id"] == op[field], "presign record binding")
    c = plan["commitment"]
    for key in ("profile_id", "source_digest", "policy_hash", "capability_record_id", "intent_record_id", "signer_key_id", "account_number", "sequence", "gas_limit", "timeout_height", "expires_at"):
        require(c[key] == op[key], "presign coordinate mismatch")
    require(c["fee_amount_uzrn"] == op["fee_uzrn"], "presign fee mismatch")
    policy = next(policy for policy in p["recipients"] if policy["policy_hash"] == op["policy_hash"])
    require(obs["protocol"] == "agent-wallet-zerone.seed-observation/0.1" and obs["status"] == "observed", "native observation required")
    evidence = obs["evidence"]
    require(evidence["trust"] == "configured_full_node", "native trust boundary")
    for field in ("profile_id", "chain_id", "genesis_hash", "node_trust_id", "observed_at", "anchor", "latest_height", "catching_up"):
        require(same(evidence[field], e[field]), "gate/runtime observation mismatch")
    # Preparation is historical: validate it at prepared_at, not recovery time.
    # All three native anchors also join every signed stage/confirmation anchor.
    original = b["observation"]
    require(original["protocol"] == obs["protocol"] and original["status"] == "observed", "original native observation required")
    simulation = b["simulation_result"]["evidence"]
    previous = None
    for native, at in ((original["evidence"], timestamp(b["prepared_at"])), (simulation, now), (evidence, now)):
        require(native["trust"] == "configured_full_node" and native["catching_up"] is False, "native anchor trust")
        for field in ("profile_id", "chain_id", "genesis_hash", "node_trust_id"):
            require(native[field] == evidence[field], "preparation/simulation node binding")
        a = exact(native["anchor"], {"height", "block_hash", "block_time"}, "native anchor")
        height, latest = uint(a["height"], 64, True), uint(native["latest_height"], 64, True)
        require(height <= latest <= height + policy["max_height_lag"], "native anchor height lag")
        block_time, observed_at = timestamp(a["block_time"]), timestamp(native["observed_at"])
        require(block_time <= observed_at <= at and at - block_time <= dt.timedelta(seconds=policy["max_observation_age_seconds"]), "native anchor time")
        bind_block(blocks, a["height"], a["block_hash"], a["block_time"])
        if previous:
            require(height >= uint(previous["height"], 64) and block_time >= timestamp(previous["block_time"]), "preparation/simulation anchor rollback")
        previous = a
    require(simulation["latest_height"] == simulation["anchor"]["height"], "latest simulation required")
    row = e["recipients"][0]
    require(same(obs["pot"], row["pot"]) and same(obs["allowance"], {"status": "found"} | row["allowance"]) and obs["prior_claim"] is None, "gate/runtime pot or grant mismatch")
    require(obs["sponsor_account"] == policy["sponsor_account"] and obs["sponsor_balance_uzrn"] == e["sponsor_balance_uzrn"] and obs["min_claim_amount_uzrn"] == e["parameters"]["min_claim_amount"], "gate/runtime funding or minimum mismatch")
    require(obs["claimant"]["status"] == "found" and obs["claimant"]["account"] == policy["claimant_account"] and obs["claimant"]["account_number"] == op["account_number"] and obs["claimant"]["sequence"] == op["sequence"], "gate/runtime account mismatch")


def validate_confirmation_files(e, p, bundle):
    """Require the actual public decoded receipts, not dangling evidence hashes.

    Receipt truth is attested by the configured observer, not proved by this
    offline decoder. Never accept signed TxRaw or arbitrary fields in receipts.
    """
    pairs = []
    policies = {policy["policy_hash"]: policy for policy in p["recipients"]}
    for row in e["recipients"]:
        policy = policies[row["policy_hash"]]
        if row["admission_confirmation"]:
            pairs.append((row["admission_confirmation"], {"type_url": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", "authority": p["admission"]["authority"], "addresses": [account(policy["claimant_account"], p["profile"]["chain_id"])], "pot": row["pot"]}))
            pairs.append((row["grant_confirmation"], {"type_url": "/cosmos.feegrant.v1beta1.MsgGrantAllowance", "allowance": row["allowance"], "grantee_account_exists": True}))
    if e["parameter_update_confirmation"]:
        pairs.append((e["parameter_update_confirmation"], {"type_url": "/zerone.claiming_pot.v1.MsgUpdatePotParams", "authority": _bech32_encode("zrn", hashlib.sha256(b"gov").digest()[:20]), "params": p["admission"]["params_after"]}))
    for c, effect in pairs:
        raw = read(bundle / "evidence" / (c["evidence_sha256"][7:] + ".json"))
        require(digest(raw) == c["evidence_sha256"], "confirmed receipt digest mismatch")
        expected = {"schema": "zerone-seed-native-confirmation/v1", "environment": p["environment"], "profile_id": p["profile"]["profile_id"], "node_trust_id": e["node_trust_id"], "confirmation": {k: v for k, v in c.items() if k != "evidence_sha256"}, "effect": effect}
        require(canonical(parse(raw)) == canonical(expected), "confirmed native receipt does not match exact effect")


def run(args):
    now = timestamp(args.now) if args.mode == "synthetic" and args.now else dt.datetime.now(dt.timezone.utc)
    require(not args.now or args.mode == "synthetic", "production clock cannot be overridden")
    trust_raw = read(args.trust)
    require(digest(trust_raw) == hash_id(args.trust_sha256), "independent trust file pin")
    trust = parse(trust_raw)
    exact(trust, set("schema environment activation_id chain_id genesis_hash profile_id node_trust_id roles keyring_sha256 beta_verification_sha256 verifier_sha256 bundle_manifest_sha256".split()), "public trust")
    require(trust["schema"] == "zerone-seed-trust/v1" and trust["environment"] == args.mode, "synthetic/production trust separation")
    for k in ("genesis_hash", "profile_id", "node_trust_id", "keyring_sha256", "beta_verification_sha256", "verifier_sha256", "bundle_manifest_sha256"):
        hash_id(trust[k])
    keyring = read(args.public_keyring, 1024 * 1024)
    require(digest(keyring) == trust["keyring_sha256"], "public keyring pin")
    sigs = SignatureVerifier(args.gpgv, keyring, trust["roles"], now)
    bundle = Path(args.bundle)
    packet_raw = read(bundle / "SEED-ACTIVATION.json")
    packet_hash = digest(packet_raw)
    require(packet_hash == hash_id(args.packet_sha256), "independently selected packet pin")
    packet = parse(packet_raw)
    sigs.verify(packet_raw, read(bundle / "SEED-ACTIVATION.json.sig", 16384), "activation")
    sigs.verify(packet_raw, read(bundle / "SEED-ACTIVATION.json.review.sig", 16384), "reviewer")
    require(packet["trust_sha256"] == args.trust_sha256, "packet public trust binding")
    validate_packet(packet, trust, now)
    require(file_digest(Path(__file__).absolute()) == packet["artifacts"]["verifier_sha256"], "seed verifier artifact drift")
    artifact_root = Path(args.artifact_root)
    for k, name in ARTIFACTS.items():
        require(file_digest(artifact_root / name) == packet["artifacts"][k], "artifact drift: " + name)
    parse(read(artifact_root / "source-manifest.json"), ceremony=False)
    require(digest(read(args.beta_bundle_manifest)) == trust["bundle_manifest_sha256"], "actual beta bundle manifest pin")
    require(file_digest(args.authority_verifier) == trust["verifier_sha256"], "actual prerequisite verifier source pin")
    files = {name: read(bundle / name) for name in PREREQUISITES}
    validate_prerequisites(packet, files, sigs, trust)
    custody_raw = read(bundle / "SEED-CUSTODY.json")
    custody = parse(custody_raw)
    sigs.verify(custody_raw, read(bundle / "SEED-CUSTODY.json.sig", 16384), "custodian")
    exact(custody, {"schema", "environment", "activation_sha256", "custody", "single_signer_host_id", "route_evidence_sha256", "native_route_status", "valid_until"}, "custody/available native route attestation")
    require(custody == {"schema": "zerone-seed-custody/v1", "environment": args.mode, "activation_sha256": packet_hash, "custody": packet["custody"], "single_signer_host_id": packet["sponsor"]["single_signer_host_id"], "route_evidence_sha256": packet["admission"]["route_evidence_sha256"], "native_route_status": "available", "valid_until": packet["expires_at"]}, "custody/native authority availability mismatch")
    route = read(bundle / "NATIVE-ROUTE-EVIDENCE.json")
    require(digest(route) == packet["admission"]["route_evidence_sha256"], "native route evidence bytes missing/changed")
    route_doc = parse(route)
    exact(route_doc, {"schema", "environment", "profile_id", "mechanism", "authority", "execution_path", "source_manifest_sha256", "demonstration_sha256", "parameter_update_supported"}, "native route receipt")
    require(route_doc["schema"] == "zerone-seed-native-route/v1" and route_doc["environment"] == args.mode and route_doc["profile_id"] == packet["profile"]["profile_id"] and route_doc["mechanism"] == packet["admission"]["mechanism"] and route_doc["authority"] == packet["admission"]["authority"] and route_doc["source_manifest_sha256"] == packet["profile"]["source_digest"], "native route scope mismatch")
    expected_route = "cosmos.gov.v1.MsgSubmitProposal->zerone.claiming_pot.v1.MsgAddBootstrapEntry" if route_doc["mechanism"] == "governance" else "zerone.claiming_pot.v1.MsgAddBootstrapEntry"
    require(route_doc["execution_path"] == expected_route and type(route_doc["parameter_update_supported"]) is bool, "unsupported native execution route")
    require(packet["admission"]["params_before"] == packet["admission"]["params_after"] or route_doc["parameter_update_supported"] is True, "registrar installation lacks real governance route")
    demonstration = read(bundle / "NATIVE-ROUTE-DEMONSTRATION.json")
    require(digest(demonstration) == hash_id(route_doc["demonstration_sha256"]), "native route demonstration missing/changed")
    # The custodian and reviewer authenticate the route report, not this parser.
    # Opaque public demonstration bytes are not promoted into an independent proof.
    prior = None
    final = None
    blocks = {}
    for stage in STAGES[:STAGES.index(args.stage) + 1]:
        raw = read(bundle / ("SEED-" + stage.upper() + ".json"))
        sigs.verify(raw, read(bundle / ("SEED-" + stage.upper() + ".json.sig"), 16384), "observer")
        evidence = parse(raw)
        # Earlier signed snapshots remain historical. Validate at their signed
        # observation time, never expire completed setup merely as time passes.
        at = now if stage == args.stage else timestamp(evidence["observed_at"])
        validate_evidence(evidence, packet, packet_hash, stage, at, trust, prior, blocks)
        validate_confirmation_files(evidence, packet, bundle)
        if stage == "operation":
            sigs.verify(raw, read(bundle / "SEED-OPERATION.json.custody.sig", 16384), "custodian")
        prior, final = evidence, raw
    require(digest(final) == hash_id(args.evidence_sha256), "independently selected current evidence pin")
    if args.stage == "operation":
        require(args.operation_id is not None and prior["operation"]["operation_id"] == hash_id(args.operation_id), "exact operation pin required")
    else:
        require(args.operation_id is None, "operation pin on non-operation stage")
    result = {"schema": "zerone-seed-gate-result/v1", "result": "SYNTHETIC_MATCH" if args.mode == "synthetic" else "AUTHENTICATED_INPUTS_MATCH", "stage": args.stage, "environment": args.mode, "activation_sha256": packet_hash, "evidence_sha256": digest(final), "trust_sha256": args.trust_sha256, "chain_truth": "not_independently_verified", "effects": "none", "replay_protection": "host_atomic_ledger_required", "operation_id": args.operation_id}
    if args.stage == "operation":
        candidate = parse(read(bundle / "SEED-PRESIGN.json"), ceremony=False, public_strings=False)
        validate_presign(candidate, prior, packet, blocks, now)
        result.update(operation=prior["operation"], profile=packet["profile"], policies=packet["recipients"], cli_sha256=packet["artifacts"]["cli_sha256"], sponsor_other_exposure_uzrn=prior["sponsor_other_exposure_uzrn"], observed_at=prior["observed_at"])
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("stage", choices=STAGES)
    parser.add_argument("--mode", choices=("production", "synthetic"), default="production")
    parser.add_argument("--bundle", required=True, help="absolute local public seed evidence directory")
    parser.add_argument("--trust", required=True, help="explicit independently provided public trust JSON")
    parser.add_argument("--trust-sha256", required=True)
    parser.add_argument("--packet-sha256", required=True)
    parser.add_argument("--evidence-sha256", required=True, help="fresh stage evidence selected independently of bundle")
    parser.add_argument("--public-keyring", required=True, help="explicit exported public OpenPGP keyring only")
    parser.add_argument("--gpgv", required=True, help="absolute operator-approved gpgv executable; never packet-supplied")
    parser.add_argument("--artifact-root", required=True, help="absolute local release artifact directory")
    parser.add_argument("--beta-bundle-manifest", required=True, help="actual public manifest used by prerequisite verifier; hash-pinned, not executed")
    parser.add_argument("--authority-verifier", required=True, help="actual prior verifier source file; hash-pinned, NEVER executed here")
    parser.add_argument("--operation-id", help="exact proposed operation digest; required only for operation")
    parser.add_argument("--now", help="fixed clock ONLY for synthetic tests; prohibited in production")
    args = parser.parse_args(argv)
    try:
        result = run(args)
    except (Refusal, OSError, KeyError, TypeError, ValueError, subprocess.SubprocessError) as exc:
        # Do not reflect packet content, native error bodies or private paths.
        print(json.dumps({"result": "REFUSED", "effects": "none", "reason": str(exc) if isinstance(exc, Refusal) else "input_or_verifier_failure"}, sort_keys=True), file=sys.stderr)
        return 1
    print(canonical(result).decode())
    return 0


if __name__ == "__main__":
    sys.exit(main())
