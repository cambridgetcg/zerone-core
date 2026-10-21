#!/usr/bin/env python3
"""Host-only, keyless packaged-image smoke. Default is an offline plan.

With --execute, run version and standalone ABCI on a NEW clone of a completed
provided-copy drill. No pulls, network, Comet consensus, keys or publication.
"""
from __future__ import annotations

import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time

import provenance as p

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("state_copy_drill", HERE / "state-copy-drill.py")
assert spec is not None and spec.loader is not None
drill_tool = importlib.util.module_from_spec(spec)
spec.loader.exec_module(drill_tool)
LISTENING = "Waiting for new connection..."
START_FLAGS = ["start", "--home", "/data/.zeroned", "--with-comet=false", "--transport", "socket",
               "--address", "unix:///tmp/h1-abci.sock", "--grpc.enable=false", "--api.enable=false",
               "--minimum-gas-prices", p.MIN_GAS_PRICES, "--log_format", "json"]
VERSION_FLAGS = ["version", "--long", "--output", "json", "--home", "/tmp/version-home"]


def check_version(value):
    p.require(value.get("name") == "zerone" and value.get("server_name") == "zeroned" and
              value.get("commit") == p.SOURCE_COMMIT and value.get("version") == p.VERSION,
              "CLI version/commit differs from the frozen H1 profile")


def run_argv(image, host, home=None, cidfile=None, user=None):
    p.require(re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[0-9a-f]{64}", image), "manifest-pinned image required")
    user = f"{os.getuid()}:{os.getgid()}" if user is None else user
    p.require(re.fullmatch(r"(?:0|[1-9][0-9]*):(?:0|[1-9][0-9]*)", user), "explicit numeric container UID:GID required")
    p.require(cidfile is not None, "every runtime container needs its own CID file")
    argv = ["docker", "--host", p.validate_docker_host(host), "run", "--pull=never", "--network=none",
            "--platform", "linux/amd64", "--read-only", "--cap-drop=ALL",
            "--security-opt=no-new-privileges:true", "--user", user, "--cidfile", str(cidfile),
            "--tmpfs", "/tmp:rw,nosuid,nodev", "--env", "HOME=/tmp", "--pids-limit=256", "--memory=4g"]
    if home is None:
        return argv + [image, *VERSION_FLAGS]
    p.require("," not in str(home), "Docker mount paths cannot contain commas")
    return argv + ["--detach", "--mount",
                   f"type=bind,src={home},dst=/data/.zeroned", image, *START_FLAGS]


def check_inputs(image, built, drill):
    p.require(built.get("status") == "built-local" and built.get("reproduced") is True and
              built.get("schema") == "zerone-h1-testnet-image/v1" and
              built.get("source_commit") == p.SOURCE_COMMIT and built.get("source_sha256") == p.SOURCE_ARCHIVE_SHA256 and
              image.split("@")[-1] == built.get("image_digest") and
              re.fullmatch(r"[0-9a-f]{64}", built.get("ca_bundle_sha256") or ""), "built image/CA receipt mismatch")
    p.require(drill.get("schema") == "zerone-h1-state-copy-drill/v1" and drill.get("passed") is True and
              drill.get("input_unchanged") is True and drill.get("working_file_sha256") and
              drill.get("source_commit") == p.SOURCE_COMMIT and drill.get("source_sha256") == p.SOURCE_ARCHIVE_SHA256 and
              drill.get("evidence_class") == "provided-state-copy-synthetic-abci" and
              drill.get("copy_metadata", {}).get("chain_id") == "zerone-testnet-1",
              "a completed provided testnet drill with hashed working files is required")


def validate_receipt(path, image, built, drill_path):
    """Read evidence back, not just a success flag. Receipts are not signatures."""
    report = json.loads(path.read_text())
    drill = json.loads(drill_path.read_text())
    check_inputs(image, built, drill)
    p.require(report.get("schema") == "zerone-h1-image-runtime/v1" and report.get("passed") is True and
              report.get("image") == image and report.get("binary_sha256") == built.get("binary_sha256") and
              report.get("ca_bundle_sha256") == built.get("ca_bundle_sha256") and
              report.get("drill_receipt_sha256") == p.sha256(drill_path) and
              report.get("input_unchanged") is True, "runtime receipt is absent, failed or belongs to another image/drill")
    host = p.validate_docker_host(report["docker_host"])
    user = report["container_user"]
    expected = [run_argv(image, host, cidfile=Path(report["version_cidfile"]), user=user),
                run_argv(image, host, Path(report["runtime_home"]), Path(report["cidfile"]), user)]
    p.require(report.get("commands") == expected, "runtime commands differ from network-none smoke contract")
    for name in ("version.json", "start.log", "container.json"):
        p.require(p.sha256(path.parent / name) == report.get("evidence_sha256", {}).get(name), "runtime evidence changed: " + name)
    check_version(json.loads((path.parent / "version.json").read_text()))
    container = json.loads((path.parent / "container.json").read_text())
    validate_container(container, image, built, user)
    observations = report.get("running_observations", [])
    p.require(len(observations) >= 2 and observations[0] == 0 and
              all(type(t) in (int, float) and 0 <= t <= report["observed_running_seconds"] for t in observations) and
              observations == sorted(observations) and observations[-1] == report["observed_running_seconds"],
              "two timed running observations required")
    p.require(LISTENING in (path.parent / "start.log").read_text() and
              report.get("observed_running_seconds", 0) >= 10, "ABCI listener/start observation missing")
    return report


def validate_container(container, image, built, user):
    labels = container.get("Config", {}).get("Labels", {})
    p.require(container.get("State", {}).get("Running") is True and
              container.get("HostConfig", {}).get("NetworkMode") == "none" and
              container.get("Config", {}).get("Image") == image and
              container.get("Config", {}).get("User") == user and
              container.get("Config", {}).get("Cmd") == START_FLAGS and
              container.get("Config", {}).get("Entrypoint") == ["/usr/local/bin/zeroned"] and
              labels.get("money.zerone.binary-sha256") == built.get("binary_sha256"),
              "container is not the running network-none pinned H1 executable")


def smoke(image, image_receipt, drill_receipt, destination, host=p.DOCKER_HOST, execute=False):
    p.validate_docker_host(host)
    built, drill = json.loads(image_receipt.read_text()), json.loads(drill_receipt.read_text())
    check_inputs(image, built, drill)
    source = drill_receipt.parent / "home"
    before = drill_tool.working_inventory(source)
    p.require(before == drill["working_file_sha256"], "drilled copy changed after its receipt")
    p.require(not destination.resolve().is_relative_to(drill_receipt.parent.resolve()), "runtime output must be outside drill")
    output = p.new_output(destination)
    home, cidfile = output / "home", output / "start.cid"
    version_cidfile = output / "version.cid"
    user = f"{os.getuid()}:{os.getgid()}"
    commands = [run_argv(image, host, cidfile=version_cidfile, user=user), run_argv(image, host, home, cidfile, user)]
    report = {"schema": "zerone-h1-image-runtime/v1", "passed": False, "status": "offline-plan",
              "image": image, "binary_sha256": built["binary_sha256"], "ca_bundle_sha256": built["ca_bundle_sha256"],
              "drill_receipt_sha256": p.sha256(drill_receipt), "docker_host": host, "commands": commands,
              "runtime_home": str(home), "cidfile": str(cidfile), "version_cidfile": str(version_cidfile),
              "container_user": user, "input_unchanged": False,
              "observed_running_seconds": 0, "running_observations": [], "evidence_sha256": {},
              "limitations": "Keyless standalone ABCI listener on completed drilled application state; not Comet consensus, signer, TLS connectivity or deployment-volume proof."}
    receipt = output / "runtime.json"
    p.write_json(receipt, report)
    if not execute:
        return report
    shutil.copytree(source, home)
    p.require(drill_tool.working_inventory(home) == before, "clone differs from drilled copy")
    (home / "config").mkdir()
    # Chain ID only: never InitChain, import a genesis state, or generate a key.
    p.write_json(home / "config/genesis.json", {"chain_id": drill["copy_metadata"]["chain_id"]})
    (home / "config/app.toml").write_text(f'minimum-gas-prices = "{p.MIN_GAS_PRICES}"\n[api]\nenable = false\n[grpc]\nenable = false\n[oracle]\nenabled = false\n')
    docker = ["docker", "--host", host]
    report["status"] = "running"
    try:
        version = subprocess.check_output(commands[0], text=True, timeout=60)
        (output / "version.json").write_text(version)
        check_version(json.loads(version))
        subprocess.run(commands[1], check=True, capture_output=True, text=True, timeout=60)
        cid = cidfile.read_text().strip()
        p.require(re.fullmatch(r"[0-9a-f]{64}", cid), "invalid container ID")
        started = None
        for _ in range(60):
            container = json.loads(subprocess.check_output(docker + ["inspect", cid], text=True, timeout=15))[0]
            validate_container(container, image, built, user)
            observed = time.monotonic()
            if started is None:
                started = observed
            report["running_observations"].append(observed - started)
            logs = subprocess.check_output(docker + ["logs", cid], text=True, stderr=subprocess.STDOUT, timeout=15)
            (output / "start.log").write_text(logs)
            # Log retrieval may be slow: never certify a stale pre-log inspection.
            container = json.loads(subprocess.check_output(docker + ["inspect", cid], text=True, timeout=15))[0]
            validate_container(container, image, built, user)
            p.write_json(output / "container.json", container)
            elapsed = time.monotonic() - started
            report["running_observations"].append(elapsed)
            if LISTENING in logs and elapsed >= 10:
                report["observed_running_seconds"] = elapsed
                break
            time.sleep(1)
        else:
            raise ValueError("no ABCI listener after bounded startup wait")
        report["input_unchanged"] = drill_tool.working_inventory(source) == before
        p.require(report["input_unchanged"], "original drilled copy changed")
        report["evidence_sha256"] = {name: p.sha256(output / name) for name in ("version.json", "start.log", "container.json")}
        report.update(passed=True, status="passed")
    finally:
        # Attached version invocations can outlive their Docker client too.
        cleanup_error = None
        for owned_cid in (version_cidfile, cidfile):
            if owned_cid.is_file():
                try:
                    candidate = owned_cid.read_text().strip()
                    p.require(re.fullmatch(r"[0-9a-f]{64}", candidate), "invalid owned CID; cleanup needs operator reconciliation")
                    subprocess.run(docker + ["stop", "--time", "10", candidate], check=True, capture_output=True, timeout=30)
                    subprocess.run(docker + ["rm", candidate], check=True, capture_output=True, timeout=30)
                except (OSError, ValueError, subprocess.SubprocessError) as error:
                    cleanup_error = error
        if cleanup_error is not None:
            report.update(passed=False, status="cleanup-failed")
        if not report["passed"] and report["status"] == "running":
            report["status"] = "failed"
        p.write_json(receipt, report)
        if cleanup_error is not None:
            raise cleanup_error
    validate_receipt(receipt, image, built, drill_receipt)
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", required=True)
    parser.add_argument("--image-receipt", type=Path, required=True)
    parser.add_argument("--drill-receipt", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--docker-host", default=p.DOCKER_HOST)
    parser.add_argument("--execute", action="store_true", help="host-authorized local Docker smoke only")
    args = parser.parse_args()
    os.umask(0o077)
    try:
        print(json.dumps(smoke(args.image, args.image_receipt, args.drill_receipt, args.output,
                               args.docker_host, args.execute), sort_keys=True))
    except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        print(f"REFUSED: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
