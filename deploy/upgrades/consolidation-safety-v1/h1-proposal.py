#!/usr/bin/env python3
"""Canonical H1 plan.Info and an UNSIGNED MsgSoftwareUpgrade proposal tx (review finding H1-F2).

H1's startup lineage wall refuses to serve ABCI unless the committed plan's Info
is canonical compact sorted-key JSON, byte-equal to Go's json.Marshal of itself
(65c19cd8 app/consolidation_lineage.go:348-405). zerone-1's only earlier
software upgrade (proposal 6) and testnet's (proposals 2 and 3) used Info "",
which H1 refuses at startup after the halt. This tool:

- builds Info exactly as the kit's plan_info() does (it calls that function);
- validates it with a port of validateCanonicalConsolidationPlanInfo whose
  verdicts are pinned against the real Go function (plan-info-corpus.json);
- refuses Info "", pins each chain's gov authority and deposit, and checks the
  live gov parameters still equal those pins;
- writes an UNSIGNED tx whose timeout_height closes the submission window.

It never opens a keyring, signs, simulates, broadcasts or contacts Fly. The only
network use is HTTP GET against the RPC/REST endpoints given on the command line.
Python 3.11+.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import importlib.util
import json
import os
import math
from pathlib import Path
import re
import sys
import urllib.parse
import urllib.request

import provenance as p

HERE = Path(__file__).resolve().parent
_spec = importlib.util.spec_from_file_location("h1_ship", HERE / "ship-consolidation-safety.py")
assert _spec is not None and _spec.loader is not None
ship = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(ship)

PLAN_NAME = ship.NAME
INFO_MAX_BYTES = 4096  # consolidationPlanInfoMaxBytes
_INTEGER = re.compile(r"(0|-?[1-9][0-9]*)")  # canonicalJSONInteger, applied with fullmatch
_BECH32_ACCOUNT = re.compile(r"zrn1[02-9ac-hj-np-z]{38}")
ZERO_TIME = "0001-01-01T00:00:00Z"

# Pins checked 2026-11-06 by GET (testnet 37.16.28.121, mainnet 169.155.55.44):
# /cosmos/auth/v1beta1/module_accounts/gov and /cosmos/gov/v1/params/deposit.
PROFILES = {
    "zerone-testnet-1": {
        "gov_authority": "zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89",
        "deposit": [{"denom": "uzrn", "amount": "10000000"}],      # 10 ZRN = min_deposit
        "voting_period_seconds": 120,
        "expedited": False,
    },
    "zerone-1": {
        "gov_authority": "zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89",
        "deposit": [{"denom": "uzrn", "amount": "100000000"}],     # 100 ZRN = min_deposit
        "voting_period_seconds": 600,
        "expedited": False,
    },
}
# Seconds reserved around the vote: draft -> signed broadcast (submit), and
# proposal pass -> H (pre-H census at H-1, halt watch armed, image re-checked).
DEFAULT_SUBMIT_SECONDS = 600
DEFAULT_AFTER_PASS_SECONDS = 900
DEFAULT_SAMPLE_BLOCKS = 200


class Refused(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise Refused(message)


# --- canonical Info: port of validateCanonicalConsolidationPlanInfo -----------------------

class _Number(str):
    """A JSON integer literal kept verbatim, like Go's json.Number."""


def _reject_constant(name):
    raise Refused(f"decode JSON: invalid literal {name}")


def _reject_float(literal):
    raise Refused(f"JSON number {literal!r} is not a canonical integer")


def _integer(literal):
    if not _INTEGER.fullmatch(literal):
        raise Refused(f"JSON number {literal!r} is not a canonical integer")
    return _Number(literal)


def _object(pairs):
    # Go keeps the last duplicate, so its re-encoding can never equal the input.
    keys = [key for key, _ in pairs]
    if len(set(keys)) != len(keys):
        raise Refused("info is not canonical compact sorted-key JSON (duplicate key)")
    return dict(pairs)


def _go_string(value):
    out = ['"']
    for char in value:
        code = ord(char)
        if 0xD800 <= code <= 0xDFFF:  # Go decodes a lone surrogate escape to U+FFFD
            out.append("\ufffd")
        elif char in '"\\':
            out.append("\\" + char)
        elif char in "<>&":  # json.Marshal escapes HTML-significant runes
            out.append("\\u%04x" % code)
        elif code < 0x20:
            out.append({"\b": "\\b", "\f": "\\f", "\n": "\\n", "\r": "\\r", "\t": "\\t"}.get(char, "\\u%04x" % code))
        elif code in (0x2028, 0x2029):
            out.append("\\u%04x" % code)
        else:
            out.append(char)
    out.append('"')
    return "".join(out)


def _sort_key(key):
    # Go sorts map keys by their (post-decode) UTF-8 bytes.
    return "".join("\ufffd" if 0xD800 <= ord(c) <= 0xDFFF else c for c in key).encode("utf-8")


def _go_marshal(value):
    if value is None:
        return "null"
    if value is True:
        return "true"
    if value is False:
        return "false"
    if isinstance(value, _Number):
        return str(value)
    if isinstance(value, str):
        return _go_string(value)
    if isinstance(value, list):
        return "[" + ",".join(_go_marshal(item) for item in value) + "]"
    if isinstance(value, dict):
        items = sorted(value.items(), key=lambda item: _sort_key(item[0]))
        return "{" + ",".join(_go_string(k) + ":" + _go_marshal(v) for k, v in items) + "}"
    raise Refused("unexpected JSON value")


def validate_canonical_info(info):
    """Raise Refused unless `info` passes H1's canonical plan.Info rules."""
    if isinstance(info, (bytes, bytearray)):
        try:
            info = bytes(info).decode("utf-8")
        except UnicodeDecodeError:
            raise Refused("info is not canonical compact sorted-key JSON (invalid UTF-8)") from None
    require(isinstance(info, str), "info must be a string")
    raw = info.encode("utf-8", "surrogatepass")
    require(len(raw) > 0, "info must be a non-empty canonical JSON object")
    require(len(raw) <= INFO_MAX_BYTES, f"info exceeds {INFO_MAX_BYTES}-byte limit")
    try:
        value = json.loads(info, parse_int=_integer, parse_float=_reject_float,
                           parse_constant=_reject_constant, object_pairs_hook=_object)
    except Refused:
        raise
    except (ValueError, RecursionError) as error:
        raise Refused(f"decode JSON: {error}") from None
    require(isinstance(value, dict) and len(value) > 0, "info must be a JSON object with at least one key")
    # 4096 bytes nest at most ~2048 deep; Go accepts that (its limit is 10000), so must the port.
    limit = sys.getrecursionlimit()
    sys.setrecursionlimit(max(limit, 4 * INFO_MAX_BYTES))
    try:
        canonical = _go_marshal(value).encode("utf-8", "surrogatepass")
    finally:
        sys.setrecursionlimit(limit)
    require(canonical == raw, "info is not canonical compact sorted-key JSON")
    return info


def canonical_info(image):
    info = ship.plan_info(image)  # the kit's own builder, not a re-implementation
    validate_canonical_info(info)
    require(json.loads(info) == {"commit": p.SOURCE_COMMIT, "image": image}, "plan_info shape changed")
    return info


# --- read-only chain access -------------------------------------------------------------

def _endpoint(url, name):
    parts = urllib.parse.urlsplit(url)
    require(parts.scheme in ("http", "https") and parts.hostname and parts.username is None and
            parts.password is None and not parts.query and not parts.fragment and parts.path in ("", "/"),
            f"--{name} must be a plain http(s)://host:port base URL")
    return urllib.parse.urlunsplit((parts.scheme, parts.netloc, "", "", ""))


def get_json(base, path, params=None):
    url = base + path + ("?" + urllib.parse.urlencode(params) if params else "")
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    request = urllib.request.Request(url, method="GET", headers={"Accept": "application/json"})
    with opener.open(request, timeout=15) as response:
        return json.load(response)


def _parse_time(value):
    match = re.fullmatch(r"(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d)(?:\.(\d+))?Z", value)
    require(match is not None, f"unexpected block time {value!r}")
    base = dt.datetime.fromisoformat(match[1] + "+00:00")
    return base + dt.timedelta(microseconds=int((match[2] or "0")[:6].ljust(6, "0")))


def _duration_seconds(value):
    match = re.fullmatch(r"([0-9]+)(?:\.([0-9]+))?s", value or "")
    require(match is not None, f"unexpected duration {value!r}")
    return int(match[1]) + (int(match[2]) / 10 ** len(match[2]) if match[2] else 0)


def observe(rpc, rest, chain_id, sample_blocks=DEFAULT_SAMPLE_BLOCKS):
    status = get_json(rpc, "/status")["result"]
    require(status["node_info"]["network"] == chain_id, "RPC chain ID differs from --chain-id")
    latest = int(status["sync_info"]["latest_block_height"])
    require(latest > sample_blocks + 1, "chain too short to sample cadence")
    # Consecutive-interval sample: the fastest interval bounds how soon H can arrive.
    times = []
    for height in range(latest - sample_blocks, latest + 1, max(1, sample_blocks // 20)):
        header = get_json(rpc, "/header", {"height": str(height)})["result"]["header"]
        times.append((height, _parse_time(header["time"])))
    rates = [(t2 - t1).total_seconds() / (h2 - h1) for (h1, t1), (h2, t2) in zip(times, times[1:])]
    mean = (times[-1][1] - times[0][1]).total_seconds() / (times[-1][0] - times[0][0])
    params = get_json(rest, "/cosmos/gov/v1/params/deposit")["params"]
    authority = get_json(rest, "/cosmos/auth/v1beta1/module_accounts/gov")["account"]
    plan = get_json(rest, "/cosmos/upgrade/v1beta1/current_plan").get("plan")
    return {"chain_id": chain_id, "latest_height": latest, "latest_time": status["sync_info"]["latest_block_time"],
            "block_seconds_mean": mean, "block_seconds_min_window": min(rates), "block_seconds_max_window": max(rates),
            "sample_blocks": sample_blocks, "min_deposit": params["min_deposit"],
            "voting_period_seconds": _duration_seconds(params["voting_period"]),
            "max_deposit_period_seconds": _duration_seconds(params["max_deposit_period"]),
            "min_initial_deposit_ratio": params.get("min_initial_deposit_ratio"),
            "gov_authority": authority.get("base_account", {}).get("address"), "pending_plan": plan}


def check_profile(chain_id, observed):
    profile = PROFILES[chain_id]
    require(observed["gov_authority"] == profile["gov_authority"], "live gov authority differs from the pinned authority")
    require(observed["min_deposit"] == profile["deposit"],
            f"live min_deposit {observed['min_deposit']} differs from the pinned deposit {profile['deposit']}; "
            "re-review before proposing")
    require(observed["voting_period_seconds"] == profile["voting_period_seconds"],
            "live voting period differs from the pinned profile; re-derive the H window")
    require(observed["pending_plan"] in (None, {}), "a pending upgrade plan already exists; refuse to stack a second")
    return profile


def height_window(observed, submit_seconds, after_pass_seconds):
    """Earliest safe H and the last height at which the signed tx may still be included."""
    fastest = observed["block_seconds_min_window"]
    slowest = max(observed["block_seconds_max_window"], observed["block_seconds_mean"])
    require(fastest > 0, "non-positive block interval sample")
    voting = observed["voting_period_seconds"]
    # Blocks can arrive faster than the mean, so lead times use the FASTEST cadence.
    earliest = observed["latest_height"] + math.ceil((submit_seconds + voting + after_pass_seconds) / fastest)
    return {"earliest_h": earliest, "fastest_block_seconds": fastest, "slowest_block_seconds": slowest,
            "submit_seconds": submit_seconds, "after_pass_seconds": after_pass_seconds,
            "voting_period_seconds": voting}


def submission_deadline(height, window):
    # The tx must be included early enough that voting ends, at the fastest cadence,
    # with after_pass_seconds still left before H.
    return height - math.ceil((window["voting_period_seconds"] + window["after_pass_seconds"]) /
                              window["fastest_block_seconds"])


# --- the unsigned proposal ---------------------------------------------------------------

def proposal_messages(chain_id, image, height):
    profile = PROFILES[chain_id]
    info = canonical_info(image)
    upgrade = {"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade", "authority": profile["gov_authority"],
               "plan": {"name": PLAN_NAME, "time": ZERO_TIME, "height": str(height), "info": info,
                        "upgraded_client_state": None}}
    return info, upgrade


def unsigned_tx(chain_id, image, height, proposer, gas, fees, timeout_height, title, summary):
    profile = PROFILES[chain_id]
    require(_BECH32_ACCOUNT.fullmatch(proposer), "--proposer must be a zrn1 account address")
    ship.validate_gas(gas)
    ship.validate_fees(fees)
    info, upgrade = proposal_messages(chain_id, image, height)
    submit = {"@type": "/cosmos.gov.v1.MsgSubmitProposal", "messages": [upgrade],
              "initial_deposit": profile["deposit"], "proposer": proposer, "metadata": "",
              "title": title, "summary": summary, "expedited": profile["expedited"]}
    tx = {"body": {"messages": [submit], "memo": "", "timeout_height": str(timeout_height),
                   "extension_options": [], "non_critical_extension_options": []},
          "auth_info": {"signer_infos": [], "fee": {"amount": [{"denom": "uzrn", "amount": fees[:-4]}],
                                                    "gas_limit": gas, "payer": "", "granter": ""}, "tip": None},
          "signatures": []}
    return info, tx


def check_unsigned(tx, chain_id, image, height):
    """Re-derive every consensus-relevant field from the pins; refuse any drift."""
    profile = PROFILES[chain_id]
    require(tx.get("signatures") == [] and tx["auth_info"]["signer_infos"] == [], "tx is not unsigned")
    (submit,) = tx["body"]["messages"]
    require(submit["@type"] == "/cosmos.gov.v1.MsgSubmitProposal" and submit["expedited"] is False,
            "not a plain MsgSubmitProposal")
    require(submit["initial_deposit"] == profile["deposit"], "deposit differs from the pinned deposit")
    (upgrade,) = submit["messages"]
    _, expected = proposal_messages(chain_id, image, height)
    require(upgrade == expected, "MsgSoftwareUpgrade differs from the canonical H1 plan")
    validate_canonical_info(upgrade["plan"]["info"])
    return upgrade["plan"]["info"]


def draft(args):
    rpc, rest = _endpoint(args.rpc, "rpc"), _endpoint(args.rest, "rest")
    image = ship.validate_image(args.image)
    observed = observe(rpc, rest, args.chain_id)
    check_profile(args.chain_id, observed)
    window = height_window(observed, args.submit_seconds, args.after_pass_seconds)
    height = args.height if args.height is not None else window["earliest_h"] + args.extra_blocks
    require(height >= window["earliest_h"],
            f"H={height} is too soon: with {window['fastest_block_seconds']:.3f}s blocks, "
            f"{args.submit_seconds}s to broadcast, a {window['voting_period_seconds']:.0f}s vote and "
            f"{args.after_pass_seconds}s after passing, H must be >= {window['earliest_h']}")
    deadline = submission_deadline(height, window)
    require(deadline > observed["latest_height"], "no submission window left before H")
    info, tx = unsigned_tx(args.chain_id, image, height, args.proposer, args.gas, args.fees, deadline,
                           PLAN_NAME, args.summary)
    check_unsigned(tx, args.chain_id, image, height)
    output = p.new_output(args.output)
    proposal_file = {"messages": tx["body"]["messages"][0]["messages"], "metadata": "",
                     "deposit": ",".join(c["amount"] + c["denom"] for c in PROFILES[args.chain_id]["deposit"]),
                     "title": PLAN_NAME, "summary": args.summary, "expedited": False}
    p.write_json(output / "unsigned-tx.json", tx)
    p.write_json(output / "proposal.json", proposal_file)
    receipt = {"schema": "zerone-h1-proposal-draft/v1", "chain_id": args.chain_id, "plan_name": PLAN_NAME,
               "height": height, "info": info, "info_sha256": hashlib.sha256(info.encode()).hexdigest(),
               "image": image, "source_commit": p.SOURCE_COMMIT, "proposer": args.proposer,
               "deposit": PROFILES[args.chain_id]["deposit"], "gov_authority": PROFILES[args.chain_id]["gov_authority"],
               "timeout_height": deadline, "window": window, "observed": observed,
               "unsigned_tx_sha256": p.sha256(output / "unsigned-tx.json"),
               "eta_utc_at_mean_cadence": (_parse_time(observed["latest_time"]) + dt.timedelta(
                   seconds=(height - observed["latest_height"]) * observed["block_seconds_mean"])).isoformat(),
               "signed": False, "broadcast": False,
               "limits": "Unsigned draft. Signing, broadcast and the vote are separate operator acts under their "
                         "own authority. timeout_height makes the chain refuse the tx after the window closes."}
    p.write_json(output / "draft.json", receipt)
    return receipt


def check_plan(args):
    rest = _endpoint(args.rest, "rest")
    image = ship.validate_image(args.image)
    expected = canonical_info(image)
    plan = get_json(rest, "/cosmos/upgrade/v1beta1/current_plan").get("plan")
    require(isinstance(plan, dict), "no committed upgrade plan")
    require(plan.get("name") == PLAN_NAME and str(plan.get("height")) == str(args.height),
            f"committed plan is {plan.get('name')!r} at {plan.get('height')!r}, not {PLAN_NAME} at {args.height}")
    require(plan.get("time") in (None, "", ZERO_TIME) and plan.get("upgraded_client_state") is None,
            "committed plan carries deprecated fields")
    require(plan.get("info") == expected, "committed plan.Info is not byte-equal to the expected canonical Info")
    validate_canonical_info(plan["info"])
    return {"status": "plan-matches", "name": PLAN_NAME, "height": args.height, "info": expected,
            "info_sha256": hashlib.sha256(expected.encode()).hexdigest()}


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)
    info = sub.add_parser("info", help="print the canonical Info for an image")
    info.add_argument("--image", required=True)
    check = sub.add_parser("validate-info", help="validate an Info string with H1's rules")
    check.add_argument("--info", required=True, help="literal Info string; '' is refused")
    make = sub.add_parser("draft", help="GET live params and write an UNSIGNED proposal tx")
    make.add_argument("--chain-id", choices=sorted(PROFILES), required=True)
    make.add_argument("--rpc", required=True)
    make.add_argument("--rest", required=True)
    make.add_argument("--image", required=True)
    make.add_argument("--proposer", required=True, help="account that signs and pays the deposit (testnet: faucet)")
    make.add_argument("--height", type=int, help="explicit H; default earliest safe H plus --extra-blocks")
    make.add_argument("--extra-blocks", type=int, default=120)
    make.add_argument("--submit-seconds", type=int, default=DEFAULT_SUBMIT_SECONDS)
    make.add_argument("--after-pass-seconds", type=int, default=DEFAULT_AFTER_PASS_SECONDS)
    make.add_argument("--gas", default="300000")
    make.add_argument("--fees", default="300000uzrn")
    make.add_argument("--summary", default="H1 consolidation-safety-v1: frozen source " + p.SOURCE_COMMIT +
                      ", CometBFT " + p.COMETBFT_VERSION + " dependency patch")
    make.add_argument("--output", type=Path, required=True)
    verify = sub.add_parser("check-plan", help="compare the committed plan byte-for-byte with the expected Info")
    verify.add_argument("--rest", required=True)
    verify.add_argument("--image", required=True)
    verify.add_argument("--height", type=int, required=True)
    args = parser.parse_args()
    os.umask(0o077)
    try:
        if args.command == "info":
            print(canonical_info(ship.validate_image(args.image)))
        elif args.command == "validate-info":
            validate_canonical_info(args.info)
            print(json.dumps({"status": "canonical", "bytes": len(args.info.encode())}))
        elif args.command == "draft":
            require(args.submit_seconds >= 60 and args.after_pass_seconds >= 300 and args.extra_blocks >= 0,
                    "submit >= 60 s, after-pass >= 300 s and extra-blocks >= 0 are required")
            print(json.dumps(draft(args), sort_keys=True, indent=2))
        else:
            print(json.dumps(check_plan(args), sort_keys=True))
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(f"REFUSED: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
