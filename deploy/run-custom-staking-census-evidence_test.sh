#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

python3 - "${ROOT}" <<'PY'
from __future__ import annotations

import datetime as dt
import hashlib
import json
import os
import pathlib
import shutil
import stat
import subprocess
import sys
import tempfile
import time
from typing import Any


root = pathlib.Path(sys.argv[1]).resolve()
source_runner = root / "deploy/run-custom-staking-census-evidence.py"
runner = source_runner
main_fingerprint = "A" * 40
transition_fingerprint = "B" * 40
runner_hash = hashlib.sha256(source_runner.read_bytes()).hexdigest()
now = int(time.time())
release_created_epoch = now - 600
release_signature_epoch = now - 500
cutover_created_epoch = now - 300
cutover_signature_epoch = now - 200
halt_epoch = now - 100


def utc(epoch: int) -> str:
    return dt.datetime.fromtimestamp(epoch, tz=dt.timezone.utc).strftime(
        "%Y-%m-%dT%H:%M:%SZ"
    )


def canonical(value: Any) -> bytes:
    return (
        json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
        + "\n"
    ).encode()


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


fake_binary = b'''#!/usr/bin/env python3
import argparse
import hashlib
import json
import os
import pathlib
import sys

parser = argparse.ArgumentParser()
parser.add_argument("--home", required=True)
parser.add_argument("--backend", required=True)
parser.add_argument("--chain-id", required=True)
parser.add_argument("--expected-height", required=True)
parser.add_argument("--expected-app-hash", required=True)
parser.add_argument("--source-commit", required=True)
parser.add_argument("--copied-db", action="store_true")
args = parser.parse_args()
if marker := os.environ.get("FAKE_CENSUS_INVOKED"):
    pathlib.Path(marker).write_text("invoked\\n")
if original := os.environ.get("FAKE_CENSUS_REPLACE_ORIGINAL"):
    pathlib.Path(original).write_bytes(b"#!/bin/sh\\nexit 97\\n")
    pathlib.Path(original).chmod(0o700)
if os.environ.get("FAKE_CENSUS_NONZERO"):
    raise SystemExit(17)
report = {
    "schema": "zerone/custom-staking-census/v1",
    "result": "FAIL" if os.environ.get("FAKE_CENSUS_FAIL_REPORT") else "PASS",
    "evidence": {
        "chain_id": args.chain_id,
        "height": args.expected_height,
        "app_hash": args.expected_app_hash,
        "source_commit": args.source_commit,
    },
    "multistore": [],
    "stores": [],
    "census": {},
    "report_sha256": "",
}
unsealed = json.dumps(report, separators=(",", ":"), ensure_ascii=False).encode()
report["report_sha256"] = hashlib.sha256(unsealed).hexdigest()
contents = (json.dumps(report, separators=(",", ":"), ensure_ascii=False) + "\\n").encode()
if not os.environ.get("FAKE_CENSUS_NO_REPORT"):
    sys.stdout.buffer.write(contents)
    sys.stdout.buffer.flush()
if hostile_output := os.environ.get("FAKE_CENSUS_PRECREATE_REPORT_OUTPUT"):
    pathlib.Path(hostile_output).write_bytes(b"hostile replacement report\\n")
if os.environ.get("FAKE_CENSUS_MUTATE_DB"):
    with open(pathlib.Path(args.home) / "data/application.db/CURRENT", "ab") as handle:
        handle.write(b"hostile mutation\\n")
if manifest_name := os.environ.get("FAKE_CENSUS_REPLACE_FILE_MANIFEST"):
    manifest_path = pathlib.Path(manifest_name)
    manifest = json.loads(manifest_path.read_bytes())
    current_path = pathlib.Path(args.home) / "data/application.db/CURRENT"
    current_bytes = current_path.read_bytes()
    current = next(row for row in manifest["files"] if row["path"] == "CURRENT")
    current["size"] = len(current_bytes)
    current["sha256"] = hashlib.sha256(current_bytes).hexdigest()
    snapshot = {
        "schema": "zerone-custom-staking-census-database-snapshot-v1",
        "directories": manifest["directories"],
        "files": manifest["files"],
    }
    snapshot_raw = (
        json.dumps(snapshot, sort_keys=True, separators=(",", ":")) + "\\n"
    ).encode()
    manifest["database_snapshot_sha256"] = hashlib.sha256(snapshot_raw).hexdigest()
    manifest_path.write_bytes(
        (json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\\n").encode()
    )
if os.environ.get("FAKE_CENSUS_STDOUT"):
    print("unexpected output")
'''


tool_root: pathlib.Path
fake_verifier: pathlib.Path
fake_policy: pathlib.Path


class Case:
    def __init__(self, base: pathlib.Path, name: str) -> None:
        self.root = base / name
        self.inputs = self.root / "inputs"
        self.evidence = self.root / "evidence"
        self.binary_dir = self.root / "bin"
        self.home = self.root / "offline-home"
        self.authority_bundle = self.root / "authority-bundle"
        for directory in (
            self.inputs,
            self.evidence,
            self.binary_dir,
            self.authority_bundle,
            self.home / "data/application.db",
        ):
            directory.mkdir(parents=True, mode=0o700)
        (self.home / "data/application.db/CURRENT").write_bytes(b"MANIFEST-000001\n")
        (self.home / "data/application.db/MANIFEST-000001").write_bytes(
            b"fixture database bytes\n"
        )
        self.binary = self.binary_dir / "custom-staking-census-linux-amd64"
        self.binary.write_bytes(fake_binary)
        self.binary.chmod(0o700)
        self.file_manifest = (
            self.evidence
            / "OFFLINE-HALTED-OBSERVER-APPLICATION-DB-FILE-MANIFEST.json"
        )
        capture = subprocess.run(
            [
                sys.executable,
                str(runner),
                "capture-manifest",
                "--home",
                str(self.home),
                "--output",
                str(self.file_manifest),
            ],
            text=True,
            capture_output=True,
        )
        if capture.returncode != 0:
            raise AssertionError(f"manifest capture failed: {capture.stderr}")
        source_manifest = json.loads(self.file_manifest.read_bytes())

        self.snapshot = self.inputs / "OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json"
        self.snapshot.write_bytes(
            canonical(
                {
                    "schema": "zerone-1-offline-halted-observer-snapshot-manifest-v1",
                    "chain_id": "zerone-1",
                    "checkpoint_state_height": "1000",
                    "checkpoint_app_hash": "C" * 64,
                    "final_committed_height": "1001",
                    "halt_trigger_height": "1002",
                    "blockstore_height": "1002",
                    "abci_last_applied_height": "1001",
                    "post_anchor_app_hash": "D" * 64,
                    "source_observer_node_id": "e" * 40,
                    "database_snapshot_sha256": source_manifest[
                        "database_snapshot_sha256"
                    ],
                    "file_manifest_sha256": sha(self.file_manifest.read_bytes()),
                    "stored_offline": True,
                    "included_in_authority_bundle": False,
                    "contains_signer_keys": False,
                    "result": "MATCH",
                }
            )
        )
        self.terminal = self.inputs / "OBSERVER-EVIDENCE-MANIFEST.json"
        self.terminal.write_bytes(
            canonical(
                {
                    "schema": "zerone-1-terminal-evidence-manifest-v2",
                    "role": "independent-observer",
                    "chain_id": "zerone-1",
                    "genesis_sha256": "1" * 64,
                    "checkpoint_state_height": "1000",
                    "checkpoint_app_hash": "C" * 64,
                    "final_committed_height": "1001",
                    "final_committed_block_time": f"{utc(halt_epoch - 1)[:-1]}.1Z",
                    "halt_trigger_height": "1002",
                    "anchor_block_hash": "2" * 64,
                    "halt_trigger_block_hash": "3" * 64,
                    "halt_trigger_block_time": f"{utc(halt_epoch)[:-1]}.1Z",
                    "post_anchor_app_hash": "D" * 64,
                    "abci_last_applied_height": "1001",
                    "node_id": "e" * 40,
                    "validator_pubkey": "fixture",
                    "payload_sha256": {},
                    "result": "MATCH",
                }
            )
        )

        self.operator_manifest = self.inputs / "OPERATOR-TOOL-MANIFEST.json"
        operator = {
            "schema": "zerone-operator-tool-manifest-v1",
            "source_commit": "4" * 40,
            "signed_tag": "zerone-2-runner-fixture",
            "files": {
                "deploy/run-custom-staking-census-evidence.py": runner_hash,
                "deploy/verify-authority-chain.py": sha(fake_verifier.read_bytes()),
                "deploy/validate-fly-phase-config.py": sha(fake_policy.read_bytes()),
            },
        }
        self.operator_manifest.write_bytes(canonical(operator))
        shutil.copyfile(
            self.operator_manifest,
            self.authority_bundle / "OPERATOR-TOOL-MANIFEST.json",
        )
        (self.authority_bundle / "AUTHORITY-GATE").write_text("MATCH\n")
        self.release = self.inputs / "RELEASE-PACKET.json"
        release = {
            "schema": "zerone-2-release-packet-v2",
            "created_at": utc(release_created_epoch),
            "chain_id": "zerone-2",
            "signature_authority": {
                "algorithm": "openpgp",
                "authorized_signer_fingerprint": main_fingerprint,
                "detached_signature_filename": "RELEASE-PACKET.json.sig",
            },
            "source": {
                "commit": "4" * 40,
                "signed_annotated_tag": "zerone-2-runner-fixture",
                "tag_signer_fingerprint": main_fingerprint,
            },
            "public_identities": {
                "transition_attestation": {
                    "algorithm": "openpgp",
                    "authorized_signer_fingerprint": transition_fingerprint,
                    "purpose": "fixture",
                }
            },
            "custom_staking_census_execution": {
                "schema": "zerone-custom-staking-census-execution-contract-v1",
                "binary": {
                    "filename": "custom-staking-census-linux-amd64",
                    "sha256": sha(self.binary.read_bytes()),
                },
                "execution_evidence": {
                    "filename": "CUSTOM-STAKING-CENSUS-EXECUTION-EVIDENCE.json",
                    "detached_signature_filename": (
                        "CUSTOM-STAKING-CENSUS-EXECUTION-EVIDENCE.json.sig"
                    ),
                    "authorized_signer_fingerprint": transition_fingerprint,
                },
            },
            "operator_tool_manifest_sha256": sha(self.operator_manifest.read_bytes()),
        }
        self.release.write_bytes(canonical(release))
        self.release_signature = self.inputs / "RELEASE-PACKET.json.sig"
        self.release_signature.write_bytes(b"fixture release signature\n")

        self.cutover = self.inputs / "CUTOVER-INITIATION-EVIDENCE.json"
        self.cutover.write_bytes(
            canonical(
                {
                    "schema": "zerone-2-cutover-initiation-evidence-v1",
                    "attestation_result": "MATCH",
                    "created_at": utc(cutover_created_epoch),
                    "signature_authority": {
                        "algorithm": "openpgp",
                        "authorized_signer_fingerprint": main_fingerprint,
                        "detached_signature_filename": (
                            "CUTOVER-INITIATION-EVIDENCE.json.sig"
                        ),
                    },
                    "deadline_satisfied": True,
                }
            )
        )
        self.cutover_signature = (
            self.inputs / "CUTOVER-INITIATION-EVIDENCE.json.sig"
        )
        self.cutover_signature.write_bytes(b"fixture cutover signature\n")
        self.cutover_decision = self.inputs / "CUTOVER-DECISION.json"
        self.cutover_decision.write_bytes(
            canonical(
                {
                    "schema": "zerone-2-cutover-decision-v1",
                    "decision": "GO",
                    "signature_authority": {
                        "algorithm": "openpgp",
                        "authorized_signer_fingerprint": main_fingerprint,
                        "detached_signature_filename": "CUTOVER-DECISION.json.sig",
                    },
                }
            )
        )
        self.cutover_decision_signature = self.inputs / "CUTOVER-DECISION.json.sig"
        self.cutover_decision_signature.write_bytes(b"fixture decision signature\n")
        self.report = self.evidence / "CUSTOM-STAKING-CENSUS.json"
        self.receipt = (
            self.evidence / "CUSTOM-STAKING-CENSUS-EXECUTION-EVIDENCE.json"
        )
        self.marker = self.root / "invoked"

    def command(self) -> list[str]:
        return [
            sys.executable,
            str(runner),
            "execute",
            "--release-packet",
            str(self.release),
            "--release-signature",
            str(self.release_signature),
            "--operator-tool-manifest",
            str(self.operator_manifest),
            "--authority-bundle",
            str(self.authority_bundle),
            "--cutover-decision",
            str(self.cutover_decision),
            "--cutover-decision-signature",
            str(self.cutover_decision_signature),
            "--cutover-initiation-evidence",
            str(self.cutover),
            "--cutover-initiation-signature",
            str(self.cutover_signature),
            "--snapshot-manifest",
            str(self.snapshot),
            "--terminal-observer-manifest",
            str(self.terminal),
            "--source-file-manifest",
            str(self.file_manifest),
            "--binary",
            str(self.binary),
            "--home",
            str(self.home),
            "--backend",
            "goleveldb",
            "--report-output",
            str(self.report),
            "--evidence-output",
            str(self.receipt),
            "--expected-release-signer-fingerprint",
            main_fingerprint,
            "--expected-transition-signer-fingerprint",
            transition_fingerprint,
            "--config-policy",
            str(fake_policy),
            "--tool-root",
            str(tool_root),
        ]


def run(case: Case, fake_bin: pathlib.Path, **extra_env: str) -> subprocess.CompletedProcess[str]:
    environment = dict(os.environ)
    environment["PATH"] = f"{fake_bin}{os.pathsep}{environment['PATH']}"
    environment["FAKE_CENSUS_INVOKED"] = str(case.marker)
    environment.update(extra_env)
    return subprocess.run(case.command(), text=True, capture_output=True, env=environment)


def expect_failure(
    case: Case, fake_bin: pathlib.Path, fragment: str, **extra_env: str
) -> subprocess.CompletedProcess[str]:
    result = run(case, fake_bin, **extra_env)
    if result.returncode == 0:
        raise AssertionError(f"expected rejection containing {fragment!r}")
    if fragment not in result.stderr:
        raise AssertionError(
            f"rejection omitted {fragment!r}; stderr was {result.stderr!r}"
        )
    return result


with tempfile.TemporaryDirectory(prefix="census-evidence-runner-test.") as raw_tmp:
    temporary = pathlib.Path(raw_tmp).resolve()
    tool_root = temporary / "tool-root"
    (tool_root / "deploy").mkdir(parents=True)
    runner = tool_root / "deploy/run-custom-staking-census-evidence.py"
    shutil.copyfile(source_runner, runner)
    runner.chmod(0o700)
    fake_verifier = tool_root / "deploy/verify-authority-chain.py"
    fake_verifier.write_text(
        '''#!/usr/bin/env python3
import os
import pathlib
import sys
if len(sys.argv) < 4 or sys.argv[1] != "cutover-postinit":
    raise SystemExit(91)
bundle = pathlib.Path(sys.argv[2])
if (bundle / "AUTHORITY-GATE").read_text() != "MATCH\\n":
    raise SystemExit(92)
required = {
    "--release", "--release-sig", "--decision", "--decision-sig",
    "--initiation", "--initiation-sig", "--config-policy", "--tool-root",
}
if not required.issubset(sys.argv):
    raise SystemExit(93)
if marker := os.environ.get("FAKE_AUTHORITY_GATE_INVOKED"):
    pathlib.Path(marker).write_text("verified\\n")
'''
    )
    fake_verifier.chmod(0o700)
    fake_policy = tool_root / "deploy/validate-fly-phase-config.py"
    fake_policy.write_text("# fixture release-bound configuration policy\n")
    fake_policy.chmod(0o600)
    fake_bin = temporary / "fake-bin"
    fake_bin.mkdir()
    fake_gpg = fake_bin / "gpg"
    fake_gpg.write_text(
        f'''#!/usr/bin/env python3
import pathlib
import sys
signature = pathlib.Path(sys.argv[-2]).name
if signature == "RELEASE-PACKET.json.sig":
    epoch = {release_signature_epoch}
elif signature == "CUTOVER-INITIATION-EVIDENCE.json.sig":
    epoch = {cutover_signature_epoch}
else:
    raise SystemExit(2)
fingerprint = "{main_fingerprint}"
if __import__("os").environ.get("FAKE_GPG_WRONG_SIGNER"):
    fingerprint = "{transition_fingerprint}"
print(f"[GNUPG:] VALIDSIG {{fingerprint}} 2026-01-01 {{epoch}} 0 4 0 1 10 00 {{fingerprint}}")
'''
    )
    fake_gpg.chmod(0o700)

    good = Case(temporary, "good")
    authority_marker = temporary / "authority-invoked"
    os.environ["FAKE_AUTHORITY_GATE_INVOKED"] = str(authority_marker)
    result = run(good, fake_bin)
    if result.returncode != 0:
        raise AssertionError(f"valid execution failed: {result.stderr}")
    receipt_raw = good.receipt.read_bytes()
    receipt = json.loads(receipt_raw)
    if receipt_raw != canonical(receipt):
        raise AssertionError("execution receipt is not canonical JSON")
    if receipt["runner"] != {
        "path": "deploy/run-custom-staking-census-evidence.py",
        "sha256": runner_hash,
    }:
        raise AssertionError("execution receipt omitted the RELEASE-bound runner")
    if not authority_marker.exists():
        raise AssertionError("runner did not invoke the full CUTOVER authority gate")
    executed_binary = pathlib.Path(receipt["command"]["argv"][0])
    if not (
        executed_binary.name == "custom-staking-census-linux-amd64"
        and receipt["command"]["binary_path"] == str(executed_binary)
        and executed_binary != good.binary
    ):
        raise AssertionError("execution receipt did not retain exact snapshotted argv")
    if executed_binary.exists():
        raise AssertionError("private census-binary execution snapshot was not removed")
    if receipt["execution"]["stdout_sha256"] != sha(good.report.read_bytes()):
        raise AssertionError("execution receipt did not bind captured report stdout")
    if not (
        receipt["command"]["report_transport"]
        == "stdout-captured-and-atomically-published"
        and "--output" not in receipt["command"]["argv"]
    ):
        raise AssertionError("execution receipt did not bind safe stdout publication")
    for output in (good.report, good.receipt):
        if stat.S_IMODE(output.stat().st_mode) != 0o600:
            raise AssertionError(f"{output.name} mode is not 0600")

    report_before = good.report.read_bytes()
    receipt_before = good.receipt.read_bytes()
    expect_failure(good, fake_bin, "report output already exists")
    if good.report.read_bytes() != report_before or good.receipt.read_bytes() != receipt_before:
        raise AssertionError("no-overwrite retry changed existing evidence")

    tampered_binary = Case(temporary, "tampered-binary")
    tampered_binary.binary.write_bytes(tampered_binary.binary.read_bytes() + b"# drift\n")
    expect_failure(
        tampered_binary,
        fake_bin,
        "binary bytes differ from RELEASE",
    )
    if tampered_binary.marker.exists() or tampered_binary.receipt.exists():
        raise AssertionError("tampered binary was invoked or received a receipt")

    changed_before = Case(temporary, "changed-before")
    with (changed_before.home / "data/application.db/CURRENT").open("ab") as handle:
        handle.write(b"hostile pre-execution mutation\n")
    expect_failure(
        changed_before,
        fake_bin,
        "differs from its file manifest before execution",
    )
    if changed_before.marker.exists() or changed_before.receipt.exists():
        raise AssertionError("changed DB was invoked or received a receipt")

    changed_during = Case(temporary, "changed-during")
    expect_failure(
        changed_during,
        fake_bin,
        "differs from its file manifest after execution",
        FAKE_CENSUS_MUTATE_DB="1",
    )
    if changed_during.report.exists() or changed_during.receipt.exists():
        raise AssertionError("post-execution DB mutation published report evidence")

    changed_db_and_manifest = Case(temporary, "changed-db-and-manifest")
    expect_failure(
        changed_db_and_manifest,
        fake_bin,
        "changed during execution",
        FAKE_CENSUS_MUTATE_DB="1",
        FAKE_CENSUS_REPLACE_FILE_MANIFEST=str(changed_db_and_manifest.file_manifest),
    )
    if (
        changed_db_and_manifest.report.exists()
        or changed_db_and_manifest.receipt.exists()
    ):
        raise AssertionError(
            "simultaneous DB/manifest mutation published report evidence"
        )

    replaced_report_path = Case(temporary, "replaced-report-path")
    expect_failure(
        replaced_report_path,
        fake_bin,
        "appeared during publication",
        FAKE_CENSUS_PRECREATE_REPORT_OUTPUT=str(replaced_report_path.report),
    )
    if (
        replaced_report_path.report.read_bytes() != b"hostile replacement report\n"
        or replaced_report_path.receipt.exists()
    ):
        raise AssertionError("hostile report-path replacement was trusted or overwritten")

    missing_report = Case(temporary, "missing-report")
    expect_failure(
        missing_report,
        fake_bin,
        "census report is empty",
        FAKE_CENSUS_NO_REPORT="1",
    )
    if missing_report.report.exists() or missing_report.receipt.exists():
        raise AssertionError("missing child report produced published evidence")

    nonzero = Case(temporary, "nonzero")
    expect_failure(
        nonzero,
        fake_bin,
        "custom-staking census failed: exit=17",
        FAKE_CENSUS_NONZERO="1",
    )
    if nonzero.report.exists() or nonzero.receipt.exists():
        raise AssertionError("nonzero child produced published evidence")

    failed_report = Case(temporary, "failed-report")
    expect_failure(
        failed_report,
        fake_bin,
        "census report did not PASS",
        FAKE_CENSUS_FAIL_REPORT="1",
    )
    if failed_report.report.exists() or failed_report.receipt.exists():
        raise AssertionError("FAIL child report produced published evidence")

    noisy = Case(temporary, "noisy")
    expect_failure(
        noisy,
        fake_bin,
        "not unambiguous UTF-8 JSON",
        FAKE_CENSUS_STDOUT="1",
    )
    if noisy.receipt.exists():
        raise AssertionError("noisy execution received a PASS receipt")

    swapped_original = Case(temporary, "swapped-original")
    swapped_result = run(
        swapped_original,
        fake_bin,
        FAKE_CENSUS_REPLACE_ORIGINAL=str(swapped_original.binary),
    )
    if swapped_result.returncode != 0:
        raise AssertionError(
            f"authenticated binary snapshot did not survive source swap: {swapped_result.stderr}"
        )
    swapped_receipt = json.loads(swapped_original.receipt.read_bytes())
    if not (
        swapped_original.binary.read_bytes() == b"#!/bin/sh\nexit 97\n"
        and pathlib.Path(swapped_receipt["command"]["binary_path"])
        != swapped_original.binary
        and swapped_receipt["binary"]["sha256"] == sha(fake_binary)
    ):
        raise AssertionError("source-path binary swap was not isolated by the snapshot")

    wrong_signer = Case(temporary, "wrong-signer")
    expect_failure(
        wrong_signer,
        fake_bin,
        "exactly one valid signature from the expected key",
        FAKE_GPG_WRONG_SIGNER="1",
    )
    if wrong_signer.marker.exists() or wrong_signer.receipt.exists():
        raise AssertionError("wrong-signer authority reached execution")

    mismatched_authority = Case(temporary, "mismatched-authority")
    (mismatched_authority.authority_bundle / "AUTHORITY-GATE").write_text("REJECT\n")
    expect_failure(
        mismatched_authority,
        fake_bin,
        "full CUTOVER post-initiation authority chain did not verify",
    )
    if mismatched_authority.marker.exists() or mismatched_authority.receipt.exists():
        raise AssertionError("mismatched authority bundle reached census execution")

print("run-custom-staking-census-evidence tests: PASS")
PY
