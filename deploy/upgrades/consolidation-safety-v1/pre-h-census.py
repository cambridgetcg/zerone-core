#!/usr/bin/env python3
"""Read-only pre-H census: states that make the H1 handler fail at H. Reports PASS or FAIL.

Covers the consensus review's H1-F4 (claiming_pot lifetime units over the
bootstrap emission cap) and the verifier's liquiditypool settings (twap window,
billing denoms, pools, and the v4 post-migration validation inputs), plus the
H1 preconditions: exact K5/P1/L3/V1 prestate VersionMap, no H1 or native lineage
marker, no H1 done height, the expected plan (or none), and no other live
governance proposal that could change these states before H.

Every value is read with HTTP GET at ONE committed height: REST with the
x-cosmos-block-height header and CometBFT RPC abci_query (raw store reads) with
?height=. Nothing is signed, written or broadcast. Run it before proposing,
again after the proposal passes, and at H-1. Python 3.11+.

The rules are ported from 65c19cd8 (file:line cited at each check). The port is
deliberately fail-closed: any state it does not fully model, such as an existing
pool, is a FAIL that needs review and a fresh state-copy drill, never a PASS.
"""
from __future__ import annotations

import argparse
import base64
import importlib.util
import json
import os
from pathlib import Path
import re
import sys
import urllib.parse
import urllib.request

import provenance as p

HERE = Path(__file__).resolve().parent
_spec = importlib.util.spec_from_file_location("h1_proposal", HERE / "h1-proposal.py")
assert _spec is not None and _spec.loader is not None
proposal_tool = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(proposal_tool)

CHAIN = "zerone-testnet-1"
PLAN_NAME = "consolidation-safety-v1"
H1_MARKER = "upgrade_marker_consolidation-safety-v1"         # app/upgrades.go:32
NATIVE_MARKER = "chain_lineage_native_consolidation-safety-v1"  # app/consolidation_lineage.go:19
MARKER_PREFIX = bytes([0x7F, 0x01])                           # x/knowledge/keeper/migration_markers.go:21
# H1's exact prestate (app/upgrades.go:43-48 boundaries applied to the H1 target map); equal to the
# 2026-10-28 drill's staged map and to both live chains' maps on 2026-11-05.
PRESTATE = {
    "alignment": 1, "auth": 5, "bank": 4, "capability": 1, "capture_challenge": 1, "capture_defense": 1,
    "claiming_pot": 1, "consensus": 1, "counterexamples": 1, "creed": 1, "distribution": 3, "emergency": 1,
    "evidence": 1, "feegrant": 2, "feeibc": 2, "genutil": 1, "gov": 5, "home": 1, "ibc": 6, "ibcratelimit": 1,
    "interchainaccounts": 3, "knowledge": 5, "liquiditypool": 3, "qualification": 1, "slashing": 4,
    "sponsorship": 1, "staking": 5, "substrate_bridge": 1, "tokens": 1, "training_provenance": 1,
    "transfer": 5, "trust_score": 1, "upgrade": 2, "vesting": 1, "vesting_rewards": 1, "work_creed": 1,
    "zerone_auth": 1, "zerone_gov": 2, "zerone_ontology": 1, "zerone_staking": 1,
}
# x/claiming_pot (types/keys.go, types/types.go)
POT_PREFIX, POT_COUNTER_KEY, MINTED_ENTRIES_KEY = b"\x02", b"\x04", b"\x06"
PER_UNIT_UZRN = 222_000
DEFAULT_EMISSION_CAP_UZRN = 222_222_000_000
UINT64_MAX = 2 ** 64 - 1
# x/liquiditypool (types/keys.go, types/types.go)
LP_PREFIXES = {"pools": b"\x01", "twap_accumulators": b"\x02", "twap_observations": b"\x05",
               "denom_index": b"\x10", "open_pool_index": b"\x11", "twap_gc_queue": b"\x12"}
LP_COUNTER_KEY = b"\x04"
MAX_SWAP_FEE_BPS, MAX_POOLS_CAP, DEFAULT_MAX_POOLS = 100_000, 64, 16
MAX_POOL_RECORDS_CAP, MAX_TWAP_WINDOW_BLOCKS, MAX_ADMISSION_ENTRIES = 10_000, 10_000, 32
SDK_MAX_BIT_LEN = 256
DENOM = re.compile(r"[a-zA-Z][a-zA-Z0-9/:._-]{2,127}")       # cosmos-sdk v0.50.15 types/coin.go:838
OPEN_GOV = ("PROPOSAL_STATUS_DEPOSIT_PERIOD", "PROPOSAL_STATUS_VOTING_PERIOD")


class Census:
    def __init__(self, rpc, rest):
        self.rpc, self.rest = rpc, rest
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        self.height = None
        self.evidence = {}

    def _get(self, url, headers=None):
        request = urllib.request.Request(url, method="GET", headers={"Accept": "application/json", **(headers or {})})
        with self.opener.open(request, timeout=20) as response:
            return json.load(response)

    def rest_get(self, path, params=None):
        url = self.rest + path + ("?" + urllib.parse.urlencode(params) if params else "")
        value = self._get(url, {"x-cosmos-block-height": str(self.height)})
        self.evidence["rest " + path + ("?" + urllib.parse.urlencode(params) if params else "")] = value
        return value

    def abci(self, store, mode, key):
        query = {"path": json.dumps(f"/store/{store}/{mode}"), "data": "0x" + key.hex(), "height": str(self.height)}
        result = self._get(self.rpc + "/abci_query?" + urllib.parse.urlencode(query))["result"]["response"]
        p.require(int(result.get("code", 0)) == 0, f"abci_query {store}/{mode} {key.hex()} failed: {result.get('log')}")
        p.require(str(result.get("height")) == str(self.height), "abci_query answered at a different height")
        value = base64.b64decode(result.get("value") or "")
        self.evidence[f"abci {store}/{mode} 0x{key.hex()}"] = value.hex()
        return value


# --- minimal protobuf wire decoding (kv.Pairs and ClaimingPot strings only) -------------

def _varint(data, offset):
    shift = result = 0
    while True:
        p.require(offset < len(data) and shift < 64, "truncated varint")
        byte = data[offset]
        offset += 1
        result |= (byte & 0x7F) << shift
        if byte < 0x80:
            return result, offset
        shift += 7


def fields(data):
    """Yield (field_number, wire_type, value) for one protobuf message."""
    offset = 0
    while offset < len(data):
        tag, offset = _varint(data, offset)
        number, wire = tag >> 3, tag & 7
        if wire == 0:
            value, offset = _varint(data, offset)
        elif wire == 2:
            size, offset = _varint(data, offset)
            p.require(offset + size <= len(data), "truncated length-delimited field")
            value, offset = data[offset:offset + size], offset + size
        elif wire == 1:
            value, offset = data[offset:offset + 8], offset + 8
        elif wire == 5:
            value, offset = data[offset:offset + 4], offset + 4
        else:
            raise ValueError(f"unsupported protobuf wire type {wire}")
        yield number, wire, value


def kv_pairs(data):
    pairs = []
    for number, wire, value in fields(data):
        p.require(number == 1 and wire == 2, "unexpected kv.Pairs field")
        key = item = b""
        for n, w, v in fields(value):
            if n == 1 and w == 2:
                key = v
            elif n == 2 and w == 2:
                item = v
        pairs.append((key, item))
    return pairs


def pot_record(value):
    pot = {"id": "", "total_amount": ""}
    for number, wire, item in fields(value):
        if wire == 2 and number == 1:
            pot["id"] = item.decode("utf-8")
        elif wire == 2 and number == 3:
            pot["total_amount"] = item.decode("utf-8")
    return pot


# --- ported rules -------------------------------------------------------------------------

def big_int_set_string(value):
    """Go big.Int.SetString(value, 10): optional sign, decimal digits only."""
    if not isinstance(value, str) or not re.fullmatch(r"[+-]?[0-9]+", value):
        return None
    return int(value)


def pot_commitment_units(total_amount):
    # x/claiming_pot/types/types.go:147-160
    total = big_int_set_string(total_amount)
    p.require(total is not None and total > 0, f"total_amount must be a positive integer: {total_amount!r}")
    units = -(-total // PER_UNIT_UZRN)
    p.require(units <= UINT64_MAX, f"total_amount commits more than uint64 units: {total_amount!r}")
    return units


def general_pot_sequence(pot_id):
    # x/claiming_pot/types/types.go:162-169 (strconv.ParseUint: digits only, no sign)
    if not pot_id.startswith("pot-") or not re.fullmatch(r"[0-9]+", pot_id[4:]):
        return None
    number = int(pot_id[4:])
    return number if 0 < number <= UINT64_MAX else None


def emission_cap(params):
    # x/claiming_pot/types/types.go:128-134: an unparsable or non-positive cap falls back to the default
    cap = big_int_set_string(params.get("bootstrapEmissionCapUzrn", params.get("bootstrap_emission_cap_uzrn", "")))
    return cap if cap is not None and cap > 0 else DEFAULT_EMISSION_CAP_UZRN


def uint64_be(value):
    # keeper.go:104-111 and :135-142: anything but 8 bytes reads as 0
    return int.from_bytes(value, "big") if len(value) == 8 else 0


def canonical_amount(value, allow_zero):
    # x/liquiditypool/types/amount.go:46-64
    number = big_int_set_string(value)
    p.require(value != "" and number is not None and str(number) == value, f"{value!r} is not a canonical base-10 integer")
    p.require(number > 0 or (allow_zero and number == 0), f"amount {value!r} is not {'non-negative' if allow_zero else 'positive'}")
    p.require(number.bit_length() <= SDK_MAX_BIT_LEN, f"amount {value!r} exceeds {SDK_MAX_BIT_LEN}-bit limit")
    return number


def parse_pool_id(pool_id):
    # x/liquiditypool/types/amount.go:68-78
    p.require(isinstance(pool_id, str) and re.fullmatch(r"pool-[1-9][0-9]*", pool_id) is not None,
              f"invalid pool ID {pool_id!r}")
    number = int(pool_id[5:])
    p.require(number <= UINT64_MAX, f"invalid pool ID {pool_id!r}")
    return number


_BECH32 = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"


def bech32_account(address):
    """sdk.AccAddressFromBech32 for prefix zrn: checksum, lowercase, 1..255 data bytes."""
    if not isinstance(address, str) or address.lower() != address or "1" not in address:
        return False
    hrp, _, data = address.rpartition("1")
    if hrp != "zrn" or len(data) < 6 or any(c not in _BECH32 for c in data):
        return False
    values = [_BECH32.index(c) for c in data]
    generator = [0x3B6A57B2, 0x26508E6D, 0x1EA119FA, 0x3D4233DD, 0x2A1462B3]
    checksum = 1
    for value in [ord(c) >> 5 for c in hrp] + [0] + [ord(c) & 31 for c in hrp] + values:
        top = checksum >> 25
        checksum = (checksum & 0x1FFFFFF) << 5 ^ value
        for i in range(5):
            checksum ^= generator[i] if (top >> i) & 1 else 0
    if checksum != 1:
        return False
    bits = 0
    accumulator = 0
    out = 0
    for value in values[:-6]:
        accumulator = (accumulator << 5) | value
        bits += 5
        while bits >= 8:
            bits -= 8
            out += 1
    return 0 < out <= 255 and bits < 5 and (accumulator & ((1 << bits) - 1)) == 0


def migrated_lp_params(params):
    """Migrate3to4 normalization (x/liquiditypool/keeper/migrator.go:67-87), then v4 Validate."""
    def number(name, camel):
        value = params.get(camel, params.get(name, 0))
        p.require(isinstance(value, (int, str)) and re.fullmatch(r"[0-9]+", str(value)) is not None,
                  f"liquiditypool {name} is not an unsigned integer: {value!r}")
        return int(value)
    migrated = {
        "default_swap_fee_bps": number("default_swap_fee_bps", "defaultSwapFeeBps"),
        "max_pools": number("max_pools", "maxPools"),
        "twap_window_blocks": number("twap_window_blocks", "twapWindowBlocks"),
        "protocol_fee_bps": 0,  # :71
        "min_initial_liquidity": params.get("minInitialLiquidity", params.get("min_initial_liquidity", "")),
        "min_reserve": params.get("minReserve", params.get("min_reserve", "")),
        "billing_quote_denoms": list(params.get("billingQuoteDenoms", params.get("billing_quote_denoms")) or []),
        "allowed_pool_denoms": list(params.get("allowedPoolDenoms", params.get("allowed_pool_denoms")) or []),
        "pool_creators": list(params.get("poolCreators", params.get("pool_creators")) or []),
    }
    if migrated["max_pools"] == 0:  # :72-78
        migrated["max_pools"] = DEFAULT_MAX_POOLS
    elif migrated["max_pools"] > MAX_POOLS_CAP:
        migrated["max_pools"] = MAX_POOLS_CAP
    return migrated


def validate_lp_params(params):
    # x/liquiditypool/types/genesis.go:229-299 (Params.Validate), run by Migrate3to4 :148 and Migrate4to5 :187
    p.require(params["default_swap_fee_bps"] <= MAX_SWAP_FEE_BPS, f"default_swap_fee_bps cannot exceed {MAX_SWAP_FEE_BPS}")
    p.require(1 <= params["max_pools"] <= MAX_POOLS_CAP, f"max_pools must be between 1 and {MAX_POOLS_CAP}")
    p.require(1 <= params["twap_window_blocks"] <= MAX_TWAP_WINDOW_BLOCKS,
              f"twap_window_blocks must be between 1 and {MAX_TWAP_WINDOW_BLOCKS}")
    p.require(params["protocol_fee_bps"] == 0, "protocol_fee_bps is retired and must be zero")
    canonical_amount(params["min_initial_liquidity"], False)
    canonical_amount(params["min_reserve"], True)
    for name in ("billing_quote_denoms", "allowed_pool_denoms"):
        values = params[name]
        p.require(len(values) <= MAX_ADMISSION_ENTRIES, f"{name} cannot exceed {MAX_ADMISSION_ENTRIES} entries")
        p.require(all(isinstance(d, str) and DENOM.fullmatch(d) for d in values), f"invalid denom in {name}")
        p.require("uzrn" not in values, f"{name} cannot contain uzrn itself")
        p.require(len(set(values)) == len(values), f"duplicate entry in {name}")
    creators = params["pool_creators"]
    p.require(len(creators) <= MAX_ADMISSION_ENTRIES, f"pool_creators cannot exceed {MAX_ADMISSION_ENTRIES} entries")
    p.require(all(bech32_account(c) for c in creators), "invalid pool creator address")
    p.require(len(set(creators)) == len(creators), "duplicate pool creator")


# --- the census ----------------------------------------------------------------------------

def run(rpc, rest, chain_id, expect, allow_proposal):
    census = Census(rpc, rest)
    checks = []

    def check(name, finding, fn):
        try:
            detail = fn()
            checks.append({"id": name, "finding": finding, "passed": True, "detail": detail})
        except (ValueError, KeyError, TypeError, OSError) as error:
            checks.append({"id": name, "finding": finding, "passed": False, "detail": str(error)})

    status = census._get(rpc + "/status")["result"]
    p.require(status["node_info"]["network"] == chain_id, "RPC chain ID differs from --chain-id")
    info = census._get(rpc + "/abci_info")["result"]["response"]
    census.height = int(info["last_block_height"])
    p.require(census.height > 0, "no committed height")

    def version_map():
        versions = {m["name"]: int(m["version"])
                    for m in census.rest_get("/cosmos/upgrade/v1beta1/module_versions")["module_versions"]}
        diff = {k: (versions.get(k), PRESTATE.get(k)) for k in set(versions) | set(PRESTATE)
                if versions.get(k) != PRESTATE.get(k)}
        p.require(not diff, f"VersionMap is not H1's exact prestate: {diff}")
        return f"{len(versions)} modules equal H1 prestate (knowledge 5, claiming_pot 1, liquiditypool 3, vesting_rewards 1)"
    check("version-map", "H1 lineage wall (app/consolidation_lineage.go:172-183)", version_map)

    def markers():
        names = [key[len(MARKER_PREFIX):].decode("utf-8", "replace")
                 for key, _ in kv_pairs(census.abci("knowledge", "subspace", MARKER_PREFIX))]
        p.require(H1_MARKER not in names, f"{H1_MARKER} is already present")
        p.require(NATIVE_MARKER not in names, f"{NATIVE_MARKER} is present")
        done = census.rest_get(f"/cosmos/upgrade/v1beta1/applied_plan/{PLAN_NAME}")
        p.require(int(done.get("height", "0")) == 0, f"{PLAN_NAME} already has a done height")
        return {"markers": names, "h1_done_height": 0}
    check("markers", "handler marker precondition (app/upgrades.go:500-518)", markers)

    def plan():
        current = census.rest_get("/cosmos/upgrade/v1beta1/current_plan").get("plan")
        if expect is None:
            p.require(current in (None, {}), f"a plan is already committed: {current}")
            return "no committed plan (before proposal)"
        image, height = expect
        info_expected = proposal_tool.canonical_info(image)
        p.require(isinstance(current, dict) and current.get("name") == PLAN_NAME and
                  str(current.get("height")) == str(height) and current.get("info") == info_expected and
                  current.get("time") in (None, "", proposal_tool.ZERO_TIME) and
                  current.get("upgraded_client_state") is None,
                  f"committed plan differs from {PLAN_NAME} at {height} with the canonical Info: {current}")
        p.require(census.height < height, f"census height {census.height} is not before H={height}")
        return f"committed plan equals the canonical H1 plan at H={height}; {height - census.height} blocks to go"
    check("plan", "H1-F2 canonical plan.Info (app/consolidation_lineage.go:330-383)", plan)

    def pots():
        params = census.rest_get("/zerone/claiming_pot/v1/params")["params"]
        raw = kv_pairs(census.abci("claiming_pot", "subspace", POT_PREFIX))
        derived, max_sequence, general = 0, 0, []
        for key, value in raw:
            pot = pot_record(value)
            p.require(key == POT_PREFIX + pot["id"].encode(), f"pot key/id mismatch for {pot['id']!r}")
            derived += pot_commitment_units(pot["total_amount"])
            sequence = general_pot_sequence(pot["id"])
            if sequence is not None:
                general.append(pot["id"])
                p.require(sequence != UINT64_MAX, f"pot {pot['id']} leaves no next general-pot id")
                max_sequence = max(max_sequence, sequence)
        minted = uint64_be(census.abci("claiming_pot", "key", MINTED_ENTRIES_KEY))
        counter = uint64_be(census.abci("claiming_pot", "key", POT_COUNTER_KEY))
        committed = max(derived, minted)
        cap = emission_cap(params)
        p.require(committed * PER_UNIT_UZRN <= cap,
                  f"existing pots commit {committed * PER_UNIT_UZRN} uzrn above bootstrap_emission_cap_uzrn {cap}")
        p.require(counter != UINT64_MAX, "pot counter exhausted")
        p.require(not general, f"general pots {general} exist: review each before H (H1-F4 fix sketch)")
        listed = census.rest_get("/zerone/claiming_pot/v1/pots", {"pagination.limit": "1000"}).get("pots", [])
        p.require(len(listed) == len(raw), "REST pot listing differs from the raw store")
        return {"pots": len(raw), "derived_units": derived, "minted_entries": minted, "committed_units": committed,
                "committed_uzrn": committed * PER_UNIT_UZRN, "cap_uzrn": cap,
                "headroom_units": cap // PER_UNIT_UZRN - committed, "pot_counter": counter}
    check("claiming-pot-units", "H1-F4 (x/claiming_pot/keeper/migrator.go:26-77)", pots)

    def liquidity():
        params = census.rest_get("/zerone/liquiditypool/v1/params")["params"]
        migrated = migrated_lp_params(params)
        validate_lp_params(migrated)
        raw = {name: kv_pairs(census.abci("liquiditypool", "subspace", prefix)) for name, prefix in LP_PREFIXES.items()}
        counter = census.abci("liquiditypool", "key", LP_COUNTER_KEY)
        rest_pools = census.rest_get("/zerone/liquiditypool/v1/pools").get("pools", [])
        p.require(len(rest_pools) == len(raw["pools"]), "REST pool count differs from the raw store")
        if counter:
            next_id = int.from_bytes(counter, "big")
            p.require(0 < next_id <= UINT64_MAX, "invalid liquiditypool next pool ID (H1 GetNextPoolId panics)")
            p.require(next_id <= MAX_POOL_RECORDS_CAP + 1, f"next_pool_id {next_id} exceeds {MAX_POOL_RECORDS_CAP + 1}")
        opened = 0
        for pool in rest_pools:
            number = parse_pool_id(pool.get("poolId", pool.get("pool_id")))
            p.require(number <= MAX_POOL_RECORDS_CAP, f"pool ID {number} exceeds {MAX_POOL_RECORDS_CAP}")
            values = [canonical_amount(pool.get(k, pool.get(s, "")), True) for k, s in
                      (("reserveA", "reserve_a"), ("reserveB", "reserve_b"), ("lpTokenSupply", "lp_token_supply"))]
            p.require(all(v > 0 for v in values) or all(v == 0 for v in values),
                      f"pool {number} has inconsistent reserve/supply zero state")
            opened += all(v > 0 for v in values)
        p.require(opened <= MAX_POOLS_CAP, f"legacy open pool count {opened} exceeds hard cap {MAX_POOLS_CAP}")
        p.require(not rest_pools, f"{len(rest_pools)} pools exist: this census does not replay the v4 genesis "
                  "validation and bank/state invariant (migrator.go:171-176); review and re-drill before H")
        for name in ("twap_accumulators", "twap_observations", "open_pool_index", "twap_gc_queue"):
            p.require(not raw[name], f"{name} holds {len(raw[name])} entries with no pools: v4 validation fails")
        return {"params_after_migration": migrated, "pools": 0, "next_pool_id": int.from_bytes(counter, "big") if counter else None,
                "raw_prefix_entries": {name: len(value) for name, value in raw.items()}}
    check("liquiditypool-v4-v5", "verifier missed item (x/liquiditypool/keeper/migrator.go:66-191)", liquidity)

    def governance():
        open_ids = []
        for state in OPEN_GOV:
            listed = census.rest_get("/cosmos/gov/v1/proposals", {"proposal_status": state, "pagination.limit": "100"})
            open_ids += [(item["id"], state, item.get("title", "")) for item in listed.get("proposals", [])]
        others = [entry for entry in open_ids if entry[0] != allow_proposal]
        p.require(not others, f"other open proposals could change pots or params before H: {others}")
        return {"open_proposals": open_ids}
    check("governance-quiet", "stop condition: no gov CreatePot or param change between proposal and H", governance)

    def freezes():
        frozen = census.rest_get("/zerone/auth/v1/frozen_accounts").get("accounts", [])
        return {"frozen_accounts": len(frozen), "note": "information only; not a handler failure"}
    check("frozen-accounts", "information (H1-F3 refuted; fee-grant route low)", freezes)

    passed = all(c["passed"] for c in checks)
    return {"schema": "zerone-h1-pre-h-census/v1", "chain_id": chain_id, "height": census.height,
            "app_hash_b64": info.get("last_block_app_hash"), "passed": passed,
            "verdict": "PASS" if passed else "FAIL", "checks": checks,
            "expected_plan": None if expect is None else {"image": expect[0], "height": expect[1]},
            "source_commit": p.SOURCE_COMMIT,
            "limits": "Read-only GET evidence at one height. Ports the H1 handler's failure conditions for pots and "
                      "liquiditypool; any state it does not model fails closed. It is not a migration replay: the "
                      "state-copy drill remains the proof that the migration runs on real state."}, census.evidence


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--chain-id", choices=[CHAIN, "zerone-1"], default=CHAIN)
    parser.add_argument("--rpc", required=True, help="CometBFT RPC base URL, e.g. http://37.16.28.121:26657")
    parser.add_argument("--rest", required=True, help="REST base URL, e.g. http://37.16.28.121:1317")
    parser.add_argument("--image", help="with --height: require exactly this committed canonical H1 plan")
    parser.add_argument("--height", type=int, help="H of the expected committed plan")
    parser.add_argument("--allow-proposal", help="the H1 proposal ID, the only open proposal tolerated")
    parser.add_argument("--output", type=Path, required=True, help="NEW directory for census.json and evidence.json")
    args = parser.parse_args()
    os.umask(0o077)
    try:
        p.require((args.image is None) == (args.height is None), "--image and --height go together")
        rpc = proposal_tool._endpoint(args.rpc, "rpc")
        rest = proposal_tool._endpoint(args.rest, "rest")
        expect = None if args.image is None else (proposal_tool.ship.validate_image(args.image), args.height)
        result, evidence = run(rpc, rest, args.chain_id, expect, args.allow_proposal)
        output = p.new_output(args.output)
        p.write_json(output / "evidence.json", evidence)
        result["evidence_sha256"] = p.sha256(output / "evidence.json")
        p.write_json(output / "census.json", result)
    except (OSError, ValueError, KeyError) as error:
        print(f"REFUSED: {error}", file=sys.stderr)
        return 1
    for item in result["checks"]:
        print(f"{'PASS' if item['passed'] else 'FAIL'}  {item['id']}: {json.dumps(item['detail'], sort_keys=True)}")
    print(f"{result['verdict']} at height {result['height']} ({args.output}/census.json)")
    return 0 if result["passed"] else 2


if __name__ == "__main__":
    sys.exit(main())
