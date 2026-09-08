#!/usr/bin/env python3
"""Local synthetic-fixture H3 -> ToK handoff driver. Defaults to validation only.

Never accepts a production home, signer import, URL, transaction command, reset,
or unsafe skip. The input manifest is not custody/provenance attestation. See
PROCEDURE.md for independent preparation/review obligations. Python >= 3.11.
"""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import socket
import subprocess
import time
import tomllib
import urllib.request

PACKET = Path(__file__).resolve().parent
SANDBOX = PACKET / "local-rehearsals"
H3 = "335bb94f0fd54d3752dcb397263b7e84fb1116b4"
TREE = "769f67f1cfa108be3d31cace7777cf954f731c42"

def digest(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()

def load(path):
    def unique(pairs):
        obj = {}
        for k, v in pairs:
            if k in obj:
                raise ValueError("duplicate JSON key: " + k)
            obj[k] = v
        return obj
    return json.loads(path.read_text(), object_pairs_hook=unique)

def confined(path, root):
    p = Path(path).absolute()
    if p.is_symlink() or p.resolve() != p or not p.is_relative_to(root):
        raise ValueError("input must be a nonsymlink path inside this rehearsal")
    return p

def tree_manifest(root):
    files = {}
    for p in sorted(root.rglob("*")):
        if p.is_symlink():
            raise ValueError("symlinks are not allowed in rehearsal homes")
        if p.is_file():
            files[str(p.relative_to(root))] = digest(p)
    return files

def rpc(port, route):
    # Fixed loopback origin/routes; this driver accepts no remote URLs.
    with urllib.request.urlopen(f"http://127.0.0.1:{port}/{route}", timeout=2) as r:
        body = r.read(1024 * 1024 + 1)
    if len(body) > 1024 * 1024:
        raise ValueError("RPC response exceeds 1 MiB")
    return json.loads(body)["result"]

def stop(proc):
    if proc.poll() is None:
        proc.send_signal(signal.SIGTERM)
        try:
            proc.wait(timeout=30)
        except subprocess.TimeoutExpired as e:
            # Never force-kill then call the resulting state a clean backup.
            raise RuntimeError("clean stop failed; keep fenced and inspect") from e

def launch(binary, home, ports, logfile):
    # Config validation requires no outbound peers/PEX. Distinct loopback ports.
    return subprocess.Popen([str(binary), "start", "--home", str(home),
        "--rpc.laddr", f"tcp://127.0.0.1:{ports[0]}",
        "--p2p.laddr", f"tcp://127.0.0.1:{ports[1]}",
        "--grpc.enable=false", "--grpc-web.enable=false", "--api.enable=false"],
        stdout=logfile, stderr=subprocess.STDOUT, start_new_session=True)

def wait_height(proc, port, target, seconds):
    end = time.monotonic() + seconds
    while time.monotonic() < end:
        if proc.poll() is not None:
            raise RuntimeError("binary exited before target; inspect its local log")
        try:
            result = rpc(port, "abci_info")
        except (OSError, ValueError):
            time.sleep(.2)
            continue
        if int(result["response"]["last_block_height"]) >= target:
            return result
        time.sleep(.2)
    raise RuntimeError("height deadline exceeded")

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--execute-local", action="store_true")
    args = parser.parse_args()
    sandbox = SANDBOX.absolute()
    if sandbox.resolve() != sandbox:
        raise ValueError("rehearsal root and its ancestors must not be symlinks")
    manifest = confined(args.manifest, sandbox)
    root = manifest.parent
    m = load(manifest)
    if m["schema"] != "zerone.tok-feedback-v1/local-rehearsal/v1" or m["synthetic_fixture_only"] is not True:
        raise ValueError("only explicitly synthetic local fixtures are accepted")
    if m["h3_source"] != H3 or m["h3_tree"] != TREE:
        raise ValueError("wrong accepted H3 source/tree")
    if m["candidate_consensus_delta_reviewed"] is not True:
        raise ValueError("review the entire H3-to-candidate delta before execution")
    old = confined(m["old_binary"], root)
    new = confined(m["new_binary"], root)
    fixture = confined(m["stopped_fixture_home"], root)
    if old == new or digest(old) != m["old_binary_sha256"] or digest(new) != m["new_binary_sha256"]:
        raise ValueError("distinct exact binary digests required")
    if tree_manifest(fixture) != m["stopped_fixture_files_sha256"]:
        raise ValueError("stopped fixture file manifest differs")
    genesis = load(fixture / "config/genesis.json")
    if genesis["chain_id"] != m["chain_id"] or not m["chain_id"].startswith("tok-feedback-rehearsal-"):
        raise ValueError("production or mismatched chain id refused")
    config = tomllib.loads((fixture / "config/config.toml").read_text())
    p2p = config["p2p"]
    if p2p.get("persistent_peers") or p2p.get("seeds") or p2p.get("pex", True):
        raise ValueError("fixture must disable outbound peers/seeds/PEX")
    if config.get("priv_validator_laddr"):
        raise ValueError("remote signer refused")
    rpc_config = config.get("rpc", {})
    if rpc_config.get("grpc_laddr") or rpc_config.get("pprof_laddr") or rpc_config.get("unsafe", False):
        raise ValueError("extra RPC/profiler listeners and unsafe RPC are refused")
    if config.get("instrumentation", {}).get("prometheus", False):
        raise ValueError("extra metrics listener refused")
    height = m["upgrade_height"]
    if type(height) is not int or height < 6:
        raise ValueError("positive separated H1/H2/H3/ToK heights required")
    ports = m["ports"]
    if len(ports) != 6 or len(set(ports)) != 6 or any(type(p) is not int or not 20000 <= p <= 60000 for p in ports):
        raise ValueError("six distinct unprivileged local ports required")
    for port in ports:
        with socket.socket() as s:
            s.bind(("127.0.0.1", port))
    # Existing output is evidence; never delete or reset it for another attempt.
    for name in ("old-home", "new-home", "backup-h-minus-one", "restore-home", "result.json"):
        if (root / name).exists():
            raise ValueError("output already exists; create a separate rehearsal")
    if not args.execute_local:
        print(json.dumps({"status":"NO_GO", "inputs_validated":True,
                          "executed":False, "custody_verified":False}))
        return
    lock = (root / "driver.lock").open("x")
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    # Local key provenance must be independently checked before this flag is
    # selected. This driver never imports keys from any other home.
    old_home = root / "old-home"
    shutil.copytree(fixture, old_home)
    evidence = {"status":"NO_GO", "scope":"local synthetic exact-binary handoff",
                "custody_verified":False, "chain_id":m["chain_id"], "height":height,
                "old_binary_sha256":digest(old), "new_binary_sha256":digest(new)}
    with (root / "old.log").open("xb") as log:
        proc = launch(old, old_home, ports[:2], log)
        try:
            evidence["h_minus_one"] = wait_height(proc, ports[0], height - 1, 180)
            # Let the old binary encounter the mandatory halt; do not replace
            # it just because a timer or source-only handler test says to.
            try:
                proc.wait(timeout=60)
            except subprocess.TimeoutExpired as e:
                raise RuntimeError("old binary did not halt before H") from e
        finally:
            stop(proc)
    if int(evidence["h_minus_one"]["response"]["last_block_height"]) != height - 1:
        raise RuntimeError("old binary advanced past mandatory H-1 fence")
    plan = load(old_home / "data/upgrade-info.json")
    if plan["name"] != "tok-feedback-v1" or plan["height"] != height or json.loads(plan["info"]) != load(PACKET / "release.json")["plan_info"]:
        raise RuntimeError("old halt plan does not match pinned feedback plan")
    backup = root / "backup-h-minus-one"
    shutil.copytree(old_home, backup)
    evidence["backup_files_sha256"] = tree_manifest(backup)
    new_home = root / "new-home"
    shutil.copytree(backup, new_home)
    if tree_manifest(new_home) != evidence["backup_files_sha256"]:
        raise RuntimeError("H-1 restore hash mismatch")
    with (root / "new-home.log").open("xb") as log:
        proc = launch(new, new_home, ports[2:4], log)
        try:
            wait_height(proc, ports[2], height + 1, 180)
            # H's application hash is in H+1's header, not H's header.
            block = rpc(ports[2], f"block?height={height + 1}")
            evidence["new-home_H_app_hash"] = block["block"]["header"]["app_hash"]
        finally:
            stop(proc)
    # Restore the stopped POST-H state, never run an old executable after H.
    # This checks cold durability, not independent deterministic block replay:
    # two freshly proposed blocks have different times and may differ in hash.
    post_files = tree_manifest(new_home)
    restored_home = root / "restore-home"
    shutil.copytree(new_home, restored_home)
    if tree_manifest(restored_home) != post_files:
        raise RuntimeError("post-H cold restore hash mismatch")
    with (root / "restore-home.log").open("xb") as log:
        proc = launch(new, restored_home, ports[4:6], log)
        try:
            wait_height(proc, ports[4], height + 1, 180)
            block = rpc(ports[4], f"block?height={height + 1}")
            evidence["restore-home_H_app_hash"] = block["block"]["header"]["app_hash"]
        finally:
            stop(proc)
    if evidence["new-home_H_app_hash"] != evidence["restore-home_H_app_hash"]:
        raise RuntimeError("cold restored committed H app hash changed")
    evidence["historical_binary_handoff_executed"] = True
    evidence["remaining"] = ["independent executable provenance and custody", "supply/module-balance/history/IBC acceptance", "signed transport and gas", "testnet and production observation"]
    (root / "result.json").write_text(json.dumps(evidence, sort_keys=True, indent=2) + "\n")
    print(json.dumps({"status":"NO_GO", "local_handoff_executed":True, "evidence":str(root / "result.json")}))

if __name__ == "__main__":
    main()
