#!/usr/bin/env python3
"""Repeatable no-network kit check: exact old/H1 builds, synthetic DB, real drill.

Uses the already installed Go selected by PATH with GOTOOLCHAIN=local. Does not
use Docker or read/fetch any chain state. Retains all outputs in a NEW directory.
"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

import provenance as p

HERE = Path(__file__).resolve().parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repository", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    os.umask(0o077)
    out = p.new_output(args.output)
    env = dict(os.environ, GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local", GOWORK="off",
               GOFLAGS="-mod=readonly", GOMAXPROCS="4", TMPDIR=str(out), GOENV="off",
               GOPRIVATE="", GONOPROXY="none", GONOSUMDB="none", GOVCS="*:off")
    commands = [
        [sys.executable, str(HERE / "state-copy-drill.py"), "build-helpers", "--repository",
         str(args.repository.resolve()), "--output", str(out / "helpers")],
        ["go", "build", "-p", "4", "-buildvcs=false", "-trimpath", "-o", str(out / "fixture-maker"), str(HERE / "test-fixture.go")],
        [str(out / "fixture-maker"), "--output", str(out / "fixture")],
        [sys.executable, str(HERE / "state-copy-drill.py"), "drill", "--provided-copy", str(out / "fixture"),
         "--helpers", str(out / "helpers"), "--output", str(out / "drill"), "--synthetic-fixture"],
        [sys.executable, str(HERE / "test_kit.py")],
    ]
    env.update(H1K_HELPERS=str(out / "helpers"), H1K_FIXTURE=str(out / "fixture"))
    for index, command in enumerate(commands):
        cwd = out / "helpers/old/src" if index == 1 else args.repository.resolve()
        with (out / f"step-{index}.log").open("wb") as log:
            result = subprocess.run(command, cwd=cwd, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=1800)
        if result.returncode:
            print(f"FAILED: step {index}, see {out}/step-{index}.log", file=sys.stderr)
            return result.returncode
    dry = p.build(args.repository, out / "image-dry-run", out / "not-read-module-cache", True)
    helpers = json.loads((out / "helpers/helpers.json").read_text())
    p.require(helpers["old"]["source_sha256"] == "62054eed39981c367c9c6d5bdf7bb19e54766afe73d336c2ea7f4a45cbfb400f",
              "legacy source closure no longer reproduces trunk provenance recipe")
    p.require(helpers["h1"]["source_sha256"] == dry["source_sha256"] == p.SOURCE_ARCHIVE_SHA256,
              "independent H1 source exports differ")
    result = {"status": "passed", "evidence_class": "offline-synthetic-fixture-only",
              "commands": commands, "h1_source_sha256": dry["source_sha256"],
              "image_built": False, "actual_testnet_copy_used": False}
    p.write_json(out / "verification.json", result)
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
