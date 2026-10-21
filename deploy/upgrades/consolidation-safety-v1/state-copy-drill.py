#!/usr/bin/env python3
"""Offline H1 migration drill on an explicitly PROVIDED scrubbed application DB.

Never fetches state, starts a node, reads a signer or imports/exports genesis.
Python 3.11+. Inputs and synthetic block requests are recorded, not live evidence.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
from typing import Any

import provenance as p

HERE = Path(__file__).resolve().parent
DB_FILE = re.compile(r"(?:CURRENT|LOCK|LOG(?:\.old)?|MANIFEST-[0-9]+|[0-9]+\.(?:log|ldb|sst)|OPTIONS-[0-9]+)")


def inventory(source):
    source = source.absolute()
    p.require(source.is_dir() and not source.is_symlink(), "provided copy must be a regular directory")
    resolved = source.resolve()
    for denied in (Path.home() / ".zeroned", Path("/Users/Shared/zerone-private")):
        p.require(not resolved.is_relative_to(denied), "a node/private home is not a provided scrubbed copy")
    p.require({x.name for x in source.iterdir()} == {"copy.json", "application.db"},
              "copy allowlist is exactly copy.json and application.db; keys/configs/extra files refused")
    database = source / "application.db"
    p.require(database.is_dir() and not database.is_symlink(), "application.db must be a regular directory")
    files = [source / "copy.json", *sorted(database.iterdir())]
    p.require(len(files) > 1, "empty application DB")
    # Inspect every name/type BEFORE opening ANY file body.
    for path in files:
        mode = path.lstat()
        p.require(stat.S_ISREG(mode.st_mode) and mode.st_nlink == 1,
                  "linked, special or non-regular input refused")
        if path.parent == database:
            p.require(DB_FILE.fullmatch(path.name), "unexpected database filename; scrub outside this tool")
    p.require((database / "CURRENT").is_file(), "missing LevelDB CURRENT")
    return {str(path.relative_to(source)): p.sha256(path) for path in files}


def metadata(source, synthetic):
    m = json.loads((source / "copy.json").read_text())
    p.require(m.get("schema") == "zerone-h1-scrubbed-copy/v1", "unknown copy schema")
    p.require(m.get("key_scrubbed") is True and m.get("stopped_consistent_copy") is True,
              "provider must attest key scrubbing and a stopped consistent copy")
    p.require(m.get("backend") == "goleveldb", "only explicit goleveldb is supported")
    expected_chain = "zerone-h1-fixture-1" if synthetic else "zerone-testnet-1"
    p.require(m.get("chain_id") == expected_chain, "wrong chain: no mainnet or implicit relabeling")
    p.require(m.get("evidence_class") == ("synthetic-fixture" if synthetic else "provided-state-copy"),
              "copy evidence class mismatch")
    p.require(type(m.get("height")) is int and 0 < m["height"] < 2**63 - 4, "invalid height")
    p.require(isinstance(m.get("app_hash"), str) and re.fullmatch(r"[0-9a-f]{64}", m["app_hash"]), "invalid app hash")
    at = dt.datetime.fromisoformat(m["block_time"].replace("Z", "+00:00"))
    p.require(at.tzinfo is not None, "block time needs an explicit timezone")
    p.require(bool(m.get("provenance")), "provider provenance description required")
    return m, at


def build_helpers(repository, destination):
    output = p.new_output(destination)
    env = dict(os.environ, GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local",
               GOWORK="off", GOFLAGS="-mod=readonly", GOMAXPROCS="4", GOENV="off",
               GOPRIVATE="", GONOPROXY="none", GONOSUMDB="none", GOVCS="*:off")
    receipt: dict[str, Any] = {"schema": "zerone-h1-drill-helpers/v1", "helper_source_sha256": p.sha256(HERE / "rehearsal.go")}
    for label, commit in (("old", p.OLD_COMMIT), ("h1", p.SOURCE_COMMIT)):
        export = p.new_output(output / label)
        closure = p.export_source(repository, export, commit)
        helper = export / "src/h1-rehearsal.go"
        shutil.copyfile(HERE / "rehearsal.go", helper)
        argv = ["go", "build", "-p", "4", "-buildvcs=false", "-trimpath", "-ldflags",
                "-X main.sourceCommit=" + commit, "-o", str(output / (label + "-helper")), str(helper)]
        with (output / (label + "-build.log")).open("wb") as log:
            subprocess.run(["go", "mod", "verify"], cwd=export / "src", env=env,
                           stdout=log, stderr=subprocess.STDOUT, check=True, timeout=300)
            subprocess.run(argv, cwd=export / "src", env=env, stdout=log, stderr=subprocess.STDOUT,
                           check=True, timeout=1800)
        receipt[label] = {"source_commit": commit, "source_sha256": closure["source_sha256"],
                          "executable_sha256": p.sha256(output / (label + "-helper")), "command": argv}
    p.write_json(output / "helpers.json", receipt)
    return receipt


def working_inventory(home):
    """Closed, keyless output contract for the subsequent packaged-runtime smoke."""
    p.require(home.is_dir() and not home.is_symlink(), "drilled home must be regular")
    p.require({x.name for x in home.iterdir()} == {".h1-disposable-copy", "data"}, "unexpected drilled home files")
    data = home / "data"
    p.require(data.is_dir() and not data.is_symlink(), "drilled data must be regular")
    p.require({x.name for x in data.iterdir()} <= {"application.db", "upgrade-info.json"}, "unexpected drilled data files")
    database = data / "application.db"
    p.require(database.is_dir() and not database.is_symlink(), "missing drilled database")
    files = [home / ".h1-disposable-copy", *sorted(database.iterdir())]
    if (data / "upgrade-info.json").exists() or (data / "upgrade-info.json").is_symlink():
        files.append(data / "upgrade-info.json")
    for path in files:
        info = path.lstat()
        p.require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1, "linked or special drilled file")
        if path.parent == database:
            p.require(DB_FILE.fullmatch(path.name), "unexpected drilled database filename")
    p.require((database / "CURRENT").is_file(), "missing drilled CURRENT")
    p.require((home / ".h1-disposable-copy").read_text() == "local-only; never a validator home\n", "wrong drill marker")
    return {str(path.relative_to(home)): p.sha256(path) for path in files}


def drill(source, helpers, destination, synthetic=False):
    # No output or subprocess until every input passes the closed allowlist.
    before = inventory(source)
    meta, at = metadata(source, synthetic)
    receipt = json.loads((helpers / "helpers.json").read_text())
    p.require(receipt.get("schema") == "zerone-h1-drill-helpers/v1", "unknown helper receipt")
    p.require(receipt.get("helper_source_sha256") == p.sha256(HERE / "rehearsal.go"), "helper source drift")
    for label, commit in (("old", p.OLD_COMMIT), ("h1", p.SOURCE_COMMIT)):
        binary = helpers / (label + "-helper")
        p.require(binary.is_file() and not binary.is_symlink(), "helper is not a regular executable")
        p.require(receipt[label]["source_commit"] == commit and
                  p.sha256(binary) == receipt[label]["executable_sha256"], "helper receipt mismatch")
    p.require(not destination.resolve().is_relative_to(source.resolve()), "output must not be inside provided copy")
    output = p.new_output(destination)
    home = output / "home"
    (home / "data").mkdir(parents=True)
    shutil.copytree(source / "application.db", home / "data/application.db")
    for name, digest in before.items():
        if name.startswith("application.db/"):
            p.require(p.sha256(home / "data" / name) == digest, "copy changed while being cloned")
    (home / ".h1-disposable-copy").write_text("local-only; never a validator home\n")
    info = json.dumps({"commit": p.SOURCE_COMMIT, "context": "offline-state-copy-synthetic-abci"},
                      sort_keys=True, separators=(",", ":"))
    report = {"schema": "zerone-h1-state-copy-drill/v1", "passed": False,
              "evidence_class": "synthetic-fixture" if synthetic else "provided-state-copy-synthetic-abci",
              "source_commit": p.SOURCE_COMMIT, "source_sha256": receipt["h1"]["source_sha256"],
              "copy_metadata": meta, "input_file_sha256": before, "helpers": receipt,
              "plan_info": info, "phases": [], "input_unchanged": False,
              "limitations": "Local plan staging and synthetic empty ABCI blocks; no signer, quorum, RPC halt polling, governance or image-runtime proof."}
    p.write_json(output / "drill.json", report)
    expected_height, expected_hash = meta["height"], meta["app_hash"]
    try:
        for label, mode, seconds in (("old", "stage", 1), ("h1", "migrate", 2)):
            argv = [str((helpers / (label + "-helper")).resolve()), "--mode", mode,
                    "--home", str(home), "--chain-id", meta["chain_id"],
                    "--expected-height", str(expected_height), "--expected-app-hash", expected_hash,
                    "--next-block-time", (at + dt.timedelta(seconds=seconds)).isoformat().replace("+00:00", "Z"),
                    "--plan-info", info]
            result = subprocess.run(argv, text=True, capture_output=True, timeout=300)
            (output / (mode + ".log")).write_text(result.stderr)
            (output / (mode + ".json")).write_text(result.stdout)
            p.require(result.returncode == 0, f"{mode} failed; inspect retained local logs")
            phase = json.loads(result.stdout)
            p.require(phase.get("passed") is True and phase.get("mode") == mode and
                      phase.get("source_commit") == receipt[label]["source_commit"], "invalid phase receipt")
            report["phases"].append(phase)
            expected_height, expected_hash = phase["after"]["height"], phase["after"]["app_hash"]
        report["input_unchanged"] = inventory(source) == before
        p.require(report["input_unchanged"], "provided source copy changed")
        report["working_file_sha256"] = working_inventory(home)
        report["passed"] = True
    finally:
        p.write_json(output / "drill.json", report)
    return report


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    build = commands.add_parser("build-helpers")
    build.add_argument("--repository", type=Path, required=True)
    build.add_argument("--output", type=Path, required=True)
    run = commands.add_parser("drill")
    run.add_argument("--provided-copy", type=Path, required=True)
    run.add_argument("--helpers", type=Path, required=True)
    run.add_argument("--output", type=Path, required=True)
    run.add_argument("--synthetic-fixture", action="store_true")
    args = parser.parse_args()
    try:
        result = (build_helpers(args.repository, args.output) if args.command == "build-helpers" else
                  drill(args.provided_copy, args.helpers, args.output, args.synthetic_fixture))
        print(json.dumps({key: value for key, value in result.items() if key not in ("input_file_sha256", "helpers", "phases")}, sort_keys=True))
    except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        print(f"REFUSED: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
