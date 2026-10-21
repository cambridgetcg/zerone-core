#!/usr/bin/env python3
"""H1 testnet operator ritual. Default: offline command plan, NEVER execution.

Repository-owned replacement for the historical operator-home script. Mainnet
is deliberately refused: its signed v* tag, attestation and zerone-1 in-place
state rehearsal are separate gates. This file never invokes Fly or builds images.
"""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import tempfile
import time
import tomllib
import urllib.parse
import urllib.request

import provenance as p
import runtime_smoke

CHAIN = "zerone-testnet-1"
NAME = "consolidation-safety-v1"
IMAGE = re.compile(r"[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[0-9a-f]{64}")


def validate_image(value):
    p.require(IMAGE.fullmatch(value), "--image must be a registry reference pinned by manifest sha256")
    return value


def plan_info(image):
    return json.dumps({"commit": p.SOURCE_COMMIT, "image": validate_image(image)}, sort_keys=True, separators=(",", ":"))


def deployment_command(config, image):
    return ["fly", "deploy", "--config", str(config), "--app", CHAIN, "--image", validate_image(image)]


def wait_halt(read_heights, target, *, polls=600, interval=8, sleep=time.sleep):
    """Four stable (Comet, ABCI) samples; ABCI must remain H-1, never commit H."""
    last, same = None, 0
    for _ in range(polls):
        try:
            rpc, app = read_heights()
            h = (int(rpc), int(app)) if all(re.fullmatch(r"[0-9]+", str(v)) for v in (rpc, app)) else None
        except (OSError, ValueError, TypeError, KeyError, subprocess.SubprocessError):
            h = None
        if h is not None and (h[0] > target or h[1] >= target):
            raise ValueError("chain committed/passed H without the required halt; do not deploy")
        # Comet can stage block H before ABCI refuses it, so status alone is insufficient.
        if h in ((target - 1, target - 1), (target, target - 1)):
            same = same + 1 if h == last else 1
        else:
            same = 0
        last = h
        if same >= 4:
            assert h is not None
            return h
        sleep(interval)
    raise ValueError("no confirmed halt: failed/moving/early polls do not authorize deployment")


def readiness(image, image_receipt, drill_receipt, runtime_receipt=None):
    built = json.loads(image_receipt.read_text())
    drill = json.loads(drill_receipt.read_text())
    p.require(built.get("schema") == "zerone-h1-testnet-image/v1" and
              built.get("status") == "built-local" and built.get("reproduced") is True,
              "image must be pre-built twice, not a dry run")
    p.require(built.get("source_commit") == p.SOURCE_COMMIT and
              image.split("@", 1)[1] == built.get("image_digest") and
              re.fullmatch(r"[0-9a-f]{64}", built.get("binary_sha256") or ""), "image provenance mismatch")
    p.require(drill.get("schema") == "zerone-h1-state-copy-drill/v1" and drill.get("passed") is True and
              drill.get("input_unchanged") is True and
              drill.get("evidence_class") == "provided-state-copy-synthetic-abci" and
              drill.get("copy_metadata", {}).get("chain_id") == CHAIN,
              "a passing PROVIDED testnet-state drill is required; fixtures do not qualify")
    p.require(drill.get("source_commit") == p.SOURCE_COMMIT and
              drill.get("source_sha256") == built.get("source_sha256") == p.SOURCE_ARCHIVE_SHA256,
              "drill/image source differs from frozen closure pin")
    p.require(runtime_receipt is not None, "packaged version and network-none start proof required BEFORE H")
    runtime_smoke.validate_receipt(runtime_receipt, image, built, drill_receipt)


def verify_cli(args):
    # Version runs in a NEW home, not the operator's key/config directory.
    with tempfile.TemporaryDirectory(prefix="h1-cli-version-") as home:
        value = json.loads(subprocess.check_output([str(args.zeroned), "version", "--long", "--output", "json",
                                                   "--home", home], text=True, timeout=30))
    runtime_smoke.check_version(value)


def validate_gas(value):
    p.require(re.fullmatch(r"[1-9][0-9]*", value), "--gas must be an explicit positive integer (no auto simulation)")
    return value


def validate_fees(value):
    p.require(re.fullmatch(r"[1-9][0-9]*uzrn", value), "--fees must be an explicit positive integer amount of uzrn")
    return value


def cli(args, *command):
    # Explicit verified CLI path: never the deleted $HOME/go/bin/zeroned fallback.
    argv = [str(args.zeroned), *map(str, command), "--home", str(args.ops_home),
            "--node", args.node, "--output", "json"]
    return json.loads(subprocess.check_output(argv, text=True, timeout=30))


def height(args):
    result = cli(args, "status")
    network = result.get("node_info", result.get("NodeInfo", {})).get("network")
    p.require(network == CHAIN, "RPC chain ID mismatch")
    return result.get("sync_info", result.get("SyncInfo", {}))["latest_block_height"]


def application_height(args):
    endpoint = urllib.parse.urlsplit(args.node)
    p.require(endpoint.scheme in ("tcp", "http", "https") and endpoint.hostname and
              endpoint.username is None and endpoint.password is None and not endpoint.query and
              not endpoint.fragment and endpoint.path in ("", "/"), "invalid RPC endpoint")
    scheme = "http" if endpoint.scheme == "tcp" else endpoint.scheme
    url = urllib.parse.urlunsplit((scheme, endpoint.netloc, "/abci_info", "", ""))
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    with opener.open(url, timeout=5) as response:
        result = json.load(response)
    return result["result"]["response"]["last_block_height"]


def tx(args, *command):
    response = cli(args, "tx", *command, "--from", args.operator, "--keyring-backend", args.keyring_backend,
                   "--chain-id", CHAIN, "--gas", validate_gas(args.gas), "--fees", validate_fees(args.fees), "--yes",
                   "--broadcast-mode", "sync")
    p.require(int(response.get("code", -1)) == 0, "transaction rejected by CheckTx; do not retry blindly")
    txhash = response.get("txhash", "")
    p.require(re.fullmatch(r"[A-Fa-f0-9]{64}", txhash), "missing transaction hash; reconcile before retry")
    print("submitted transaction " + txhash, flush=True)
    for _ in range(60):
        try:
            included = cli(args, "query", "tx", txhash)
        except (subprocess.SubprocessError, ValueError):
            time.sleep(2)
            continue
        included = included.get("tx_response", included)
        p.require(int(included.get("code", -1)) == 0, "transaction failed in block; do not retry blindly")
        return included
    raise ValueError("transaction inclusion ambiguous: reconcile " + txhash + "; do not submit again")


def proposal_id(result):
    ids = {a["value"] for event in result.get("events", []) if event.get("type") == "submit_proposal"
           for a in event.get("attributes", []) if a.get("key") == "proposal_id"}
    p.require(len(ids) == 1, "submit transaction must identify exactly one proposal; never choose the latest proposal")
    value = ids.pop()
    p.require(re.fullmatch(r"[1-9][0-9]*", value), "invalid proposal ID")
    return value


def matching_plan(plan, args):
    return (isinstance(plan, dict) and set(plan) <= {"name", "height", "info", "time", "upgraded_client_state"} and
            plan.get("name") == NAME and str(plan.get("height")) == str(args.height) and
            plan.get("info") == plan_info(args.image) and
            plan.get("time") in (None, "", "0001-01-01T00:00:00Z") and plan.get("upgraded_client_state") is None)


def proposal_plan(message):
    # Frozen AutoCLI beta.3 emits Amino Any envelopes; manual SDK queries can
    # emit protobuf JSON. Never accept an untyped/ambiguous message envelope.
    if not isinstance(message, dict):
        return None
    if message.get("type") == "cosmos-sdk/MsgSoftwareUpgrade" and set(message) == {"type", "value"}:
        value = message["value"]
        return value.get("plan") if isinstance(value, dict) else None
    if message.get("@type") == "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade" and "type" not in message:
        return message.get("plan")
    return None


def gov_authority(account):
    # The same AutoCLI Amino envelope appears on auth's Any account response.
    if account.get("type") == "cosmos-sdk/ModuleAccount" and set(account) == {"type", "value"}:
        account = account["value"]
    p.require(isinstance(account, dict), "invalid gov module account")
    address = account.get("base_account", account).get("address")
    p.require(account.get("name") == "gov" and isinstance(address, str) and address.startswith("zrn1"),
              "cannot resolve gov authority")
    return address


def execute(args):
    p.require(os.environ.get("SHIP_ACK") == CHAIN, "Yu's separate testnet halt yes is required (SHIP_ACK)")
    p.require(args.image_available, "operator must verify the pre-built digest is available BEFORE proposing H")
    args.zeroned = args.zeroned.resolve()
    p.require(args.zeroned.is_file() and os.access(args.zeroned, os.X_OK), "explicit current CLI is not executable")
    verify_cli(args)
    p.require(args.operator and args.operator.lower() != "faucet", "use the validator operator, never the zero-power faucet")
    p.require(args.image_receipt and args.drill_receipt, "image and provided-state drill receipts required")
    readiness(args.image, args.image_receipt, args.drill_receipt, args.runtime_receipt)
    config = tomllib.loads(args.fly_config.read_text())
    p.require(config.get("app") == CHAIN, "reviewed testnet Fly config required; retired/mainnet configs refused")
    ignored = {"MONIKER", "EXTERNAL_ADDRESS", "EXTRA_START_FLAGS"} & config.get("env", {}).keys()
    p.require(not ignored, "scratch runtime ignores legacy environment variables: " + ", ".join(sorted(ignored)) +
              "; translate them to reviewed config/explicit argv")
    if args.resume_proposal:
        # Read-only recovery: no key lookup, proposal write, tx submission or vote.
        height(args)  # Still authenticate the RPC chain before reading its plan.
        proposal = cli(args, "query", "gov", "proposal", args.resume_proposal)["proposal"]
        p.require(proposal.get("status") == "PROPOSAL_STATUS_PASSED", "resume requires the exact PASSED proposal")
        messages = proposal.get("messages", [])
        p.require(len(messages) == 1 and matching_plan(proposal_plan(messages[0]), args),
                  "resume proposal differs from requested H1/image/height")
        finish_halt(args)
        return
    current = int(height(args))
    p.require(args.height > current + 100, "H must leave voting/coordination headroom; inspect cadence separately")
    # Confirm the selected key is the bonded operator, not merely a funded proposer.
    valoper = subprocess.check_output([str(args.zeroned), "keys", "show", args.operator, "--address", "--bech", "val",
                                      "--home", str(args.ops_home), "--keyring-backend", args.keyring_backend], text=True, timeout=30).strip()
    validator = cli(args, "query", "staking", "validator", valoper)["validator"]
    p.require(validator.get("operator_address") == valoper and not validator.get("jailed") and
              validator.get("status") == "BOND_STATUS_BONDED" and int(validator.get("tokens", 0)) >= 1000000,
              "selected operator has no bonded voting power")
    authority = gov_authority(cli(args, "query", "auth", "module-account", "gov")["account"])
    pending = cli(args, "query", "upgrade", "plan").get("plan")
    p.require(not pending, "a pending plan already exists; refuse to overwrite")
    params = cli(args, "query", "gov", "params")["params"]
    proposal = {"messages": [{"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade", "authority": authority,
                              "plan": {"name": NAME, "height": str(args.height), "info": plan_info(args.image)}}],
                "metadata": "", "deposit": ",".join(c["amount"] + c["denom"] for c in params["min_deposit"]),
                "title": NAME, "summary": "H1 testnet: frozen source " + p.SOURCE_COMMIT}
    # Never overwrite a past proposal: an ambiguous submission must be reconciled.
    p.require(args.proposal is not None, "--proposal is required for a NEW submission")
    with args.proposal.open("x") as stream:
        json.dump(proposal, stream, sort_keys=True)
    pid = proposal_id(tx(args, "gov", "submit-proposal", args.proposal))
    tx(args, "gov", "vote", pid, "yes")  # SAME bonded operator, not faucet.
    print(f"Proposal {pid}; after a wait timeout, reconcile it then use --resume-proposal {pid}; never resubmit.", flush=True)
    for _ in range(240):
        status = cli(args, "query", "gov", "proposal", pid)["proposal"]["status"]
        if status == "PROPOSAL_STATUS_PASSED":
            break
        p.require(status not in ("PROPOSAL_STATUS_REJECTED", "PROPOSAL_STATUS_FAILED"), "proposal rejected/failed")
        time.sleep(5)
    else:
        raise ValueError("proposal did not pass in time")
    finish_halt(args)


def finish_halt(args):
    armed = cli(args, "query", "upgrade", "plan")["plan"]
    p.require(matching_plan(armed, args),
              "committed plan differs; do not deploy")
    halted = wait_halt(lambda: (height(args), application_height(args)), args.height, polls=args.halt_polls)
    state = "pre-H stall (H-1, H-1); H has not been staged" if halted[0] == args.height - 1 else "staged-H stall (H, H-1); H is not committed"
    print(f"Four frozen-height polls: {state}; Comet={halted[0]}, ABCI={halted[1]}; deployment NOT performed.")
    print(shlex.join(deployment_command(args.fly_config, args.image)))
    print("Separate operator deployment only; verify applied H, K6/P2/L5/V1 and resumed blocks afterward.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--chain-id", choices=[CHAIN], required=True)
    parser.add_argument("--zeroned", type=Path, required=True,
                        help="explicit frozen H1 CLI with kit version/commit fields; see TESTNET.md Darwin build")
    parser.add_argument("--ops-home", type=Path, required=True)
    parser.add_argument("--operator", required=True, help="bonded validator operator key, used for BOTH submission and vote")
    parser.add_argument("--keyring-backend", choices=["file", "os", "test"], default="file")
    parser.add_argument("--node", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--fly-config", type=Path, required=True, help="separately reviewed non-retired TESTNET config")
    parser.add_argument("--height", type=int, required=True)
    parser.add_argument("--proposal", type=Path, help="new local proposal output, required unless resuming; never overwritten")
    parser.add_argument("--resume-proposal", help="exact PASSED proposal ID; recheck plan and wait only, NEVER submit or vote")
    parser.add_argument("--halt-polls", type=int, default=600, help="bounded halt wait, 8 seconds per poll; resume after timeout")
    parser.add_argument("--gas", default="300000", help="positive integer gas limit; no auto simulation")
    parser.add_argument("--fees", default="300000uzrn", help="positive integer uzrn fee, selected separately from gas")
    parser.add_argument("--image-receipt", type=Path)
    parser.add_argument("--drill-receipt", type=Path)
    parser.add_argument("--runtime-receipt", type=Path, help="runtime_smoke.py proof for this image and exact drill")
    parser.add_argument("--image-available", action="store_true", help="operator verified registry availability BEFORE H")
    parser.add_argument("--execute", action="store_true", help="LIVE testnet governance; separate Yu authorization mandatory")
    args = parser.parse_args()
    try:
        args.zeroned = args.zeroned.resolve()
        validate_image(args.image)
        p.require(args.operator.lower() != "faucet", "validator operator required, not faucet")
        p.require(args.height > 1, "positive future H required")
        p.require(args.halt_polls >= 4, "at least four halt polls required")
        validate_gas(args.gas)
        validate_fees(args.fees)
        p.require(args.resume_proposal is None or re.fullmatch(r"[1-9][0-9]*", args.resume_proposal), "invalid resume proposal ID")
        if args.execute:
            execute(args)
        else:
            print(json.dumps({"status": "offline-plan-only", "cli": str(args.zeroned), "voter": args.operator,
                              "mode": "resume-wait-only" if args.resume_proposal else "new-proposal",
                              "resume_proposal": args.resume_proposal, "gas": args.gas, "fees": args.fees,
                              "plan_info": plan_info(args.image), "deploy_argv": deployment_command(args.fly_config, args.image),
                              "network_contact": False, "authorization": "not granted by this plan"}, sort_keys=True))
    except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        print(f"REFUSED: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
