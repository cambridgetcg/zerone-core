#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)

python3 - "${ROOT}" <<'PY'
from __future__ import annotations

import hashlib
import json
import os
import pathlib
import stat
import subprocess
import sys
import tempfile
from collections.abc import Callable
from typing import Any


root = pathlib.Path(sys.argv[1])
renderer = root / "deploy/query-gateway/render-archive-gateway-config.py"
production_template = (
    root / "deploy/query-gateway/fly.zerone-1-archive.public.example.toml"
)
renderer_hash = hashlib.sha256(renderer.read_bytes()).hexdigest()
image = "registry.example/zerone/query-gateway@sha256:" + "1" * 64
block_hash = "A" * 64
app_hash = "B" * 64

template = b'''app = "replace-zerone-1-archive-gateway-app"
primary_region = "replace-region"

[build]
  image = "replace-with-pinned-query-gateway-image-digest"

[env]
  GATEWAY_ROLE = "zerone-1-archive-query"
  EXPECTED_CHAIN_ID = "zerone-1"
  EXPECTED_ARCHIVE_HEIGHT = "REPLACE_WITH_A"
  EXPECTED_ARCHIVE_APP_HASH = "REPLACE_WITH_LOWERCASE_POST_A_APP_HASH"
  EXPECTED_ARCHIVE_BLOCK_HASH = "REPLACE_WITH_LOWERCASE_FINAL_APPLICATION_BLOCK_ID_HASH"
  UPSTREAM_HOST = "replace-zerone-1-archive-app.internal"
'''


def canonical(value: Any) -> bytes:
    return (
        json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
        + "\n"
    ).encode()


def make_release(template_bytes: bytes) -> dict[str, Any]:
    return {
        "schema": "zerone-2-release-packet-v1",
        "chain_id": "zerone-2",
        "components": {"query_gateway": {"image_ref": image}},
        "archive_render_contract": {
            "schema": "zerone-1-archive-render-contract-v1",
            "renderer_path": "deploy/mainnet/render-archive-configs.sh",
            "renderer_sha256": "2" * 64,
            "static_constraints": {
                "app": "zerone-1-archive",
                "volume": "zerone_archive_data",
                "region": "lhr",
                "image_component": "zerone_1_halt",
                "archive_candidate_role": "archive-candidate",
                "archive_role": "archive",
                "deployment_strategy": "immediate",
                "public_fly_service": False,
                "public_ip": False,
                "persistent_peers": [],
            },
        },
        "archive_gateway_render_contract": {
            "schema": "zerone-1-archive-gateway-render-contract-v1",
            "renderer_path": "deploy/query-gateway/render-archive-gateway-config.py",
            "renderer_sha256": renderer_hash,
            "template_path": (
                "deploy/query-gateway/fly.zerone-1-archive.public.example.toml"
            ),
            "bindings": {
                "app": "deployment_configs.zerone_1_archive_gateway.app",
                "primary_region": "archive_render_contract.static_constraints.region",
                "image_ref": (
                    "deployment_configs.zerone_1_archive_gateway.image_ref"
                ),
                "upstream_host": (
                    'archive_render_contract.static_constraints.app + ".internal"'
                ),
                "expected_archive_height": (
                    "FINAL-CHECKPOINT.json.final_application_block.height"
                ),
                "expected_archive_app_hash": (
                    "lower(FINAL-CHECKPOINT.json.excluded_post_anchor_state.app_hash)"
                ),
                "expected_archive_block_hash": (
                    "lower(FINAL-CHECKPOINT.json.final_application_block.block_id_hash)"
                ),
            },
        },
        "deployment_configs": {
            "zerone_1_archive_gateway": {
                "app": "zerone-1-archive-gateway",
                "role": "zerone-1-archive-query",
                "image_component": "query_gateway",
                "image_ref": image,
            }
        },
        "phase_dependent_config_template_sha256": {
            "zerone_1_archive_gateway": hashlib.sha256(template_bytes).hexdigest()
        },
    }


def make_final(release_bytes: bytes) -> dict[str, Any]:
    return {
        "schema": "zerone-final-checkpoint-v3",
        "status": "frozen",
        "chain_id": "zerone-1",
        "authority_chain": {
            "release_packet": {
                "sha256": hashlib.sha256(release_bytes).hexdigest(),
                "detached_signature_sha256": "3" * 64,
            }
        },
        "final_application_block": {
            "height": "1001",
            "block_id_hash": block_hash,
            "commit_canonical": True,
        },
        "excluded_post_anchor_state": {
            "abci_last_applied_height": "1001",
            "app_hash": app_hash,
            "included_in_successor_inventory": False,
        },
        "archive": {
            "blockstore_status_height": "1001",
            "abci_last_applied_height": "1001",
            "halt_trigger_block_present": False,
            "public_service_authorized": False,
        },
        "successor": {"release": {"query_gateway_image_ref": image}},
    }


Mutator = Callable[[dict[str, Any]], None]


class Case:
    def __init__(
        self,
        directory: pathlib.Path,
        name: str,
        template_bytes: bytes = template,
        release_mutator: Mutator | None = None,
        final_mutator: Mutator | None = None,
    ) -> None:
        self.release = directory / f"{name}.release.json"
        self.final = directory / f"{name}.final.json"
        self.template = directory / f"{name}.template.toml"
        self.output = directory / f"{name}.output.toml"
        release_object = make_release(template_bytes)
        if release_mutator is not None:
            release_mutator(release_object)
        release_bytes = canonical(release_object)
        final_object = make_final(release_bytes)
        if final_mutator is not None:
            final_mutator(final_object)
        self.release.write_bytes(release_bytes)
        self.final.write_bytes(canonical(final_object))
        self.template.write_bytes(template_bytes)

    def command(self, output: pathlib.Path | None = None) -> list[str]:
        return [
            sys.executable,
            str(renderer),
            str(self.release),
            str(self.final),
            str(self.template),
            str(output or self.output),
        ]


def run_ok(
    case: Case, output: pathlib.Path | None = None
) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(case.command(output), text=True, capture_output=True)
    if result.returncode != 0:
        raise AssertionError(f"successful render failed: {result.stderr}")
    if "render-archive-gateway-config: MATCH " not in result.stdout:
        raise AssertionError("successful render omitted its deterministic hash")
    return result


def expect_failure(
    case: Case,
    fragment: str,
    output: pathlib.Path | None = None,
) -> None:
    destination = output or case.output
    result = subprocess.run(case.command(destination), text=True, capture_output=True)
    if result.returncode == 0:
        raise AssertionError(f"expected rejection containing {fragment!r}")
    if fragment not in result.stderr:
        raise AssertionError(
            f"rejection omitted {fragment!r}; stderr was {result.stderr!r}"
        )
    if not os.path.lexists(destination):
        return
    # Existing-output tests deliberately retain a caller-created destination.
    if fragment.startswith("output path"):
        return
    raise AssertionError("failed render left an output path behind")


with tempfile.TemporaryDirectory(prefix="archive-gateway-render-test.") as raw_tmp:
    tmp = pathlib.Path(raw_tmp)

    good = Case(tmp, "good")
    run_ok(good)
    expected = template
    replacements = {
        b"replace-zerone-1-archive-gateway-app": b"zerone-1-archive-gateway",
        b"replace-region": b"lhr",
        b"replace-with-pinned-query-gateway-image-digest": image.encode(),
        b"replace-zerone-1-archive-app.internal": b"zerone-1-archive.internal",
        b"REPLACE_WITH_A": b"1001",
        b"REPLACE_WITH_LOWERCASE_POST_A_APP_HASH": app_hash.lower().encode(),
        b"REPLACE_WITH_LOWERCASE_FINAL_APPLICATION_BLOCK_ID_HASH": (
            block_hash.lower().encode()
        ),
    }
    for source, replacement in replacements.items():
        expected = expected.replace(source, replacement)
    if good.output.read_bytes() != expected:
        raise AssertionError("rendered bytes differ from exact byte substitution")
    if stat.S_IMODE(good.output.stat().st_mode) != 0o600:
        raise AssertionError("rendered output mode is not 0600")

    repeat_output = tmp / "repeat.output.toml"
    run_ok(good, repeat_output)
    if repeat_output.read_bytes() != good.output.read_bytes():
        raise AssertionError("two deterministic renders differ")

    existing_output = tmp / "existing.output.toml"
    existing_output.write_text("keep me\n")
    expect_failure(good, "output path already exists", existing_output)
    if existing_output.read_text() != "keep me\n":
        raise AssertionError("existing output was modified")

    symlink_target = tmp / "symlink-target"
    symlink_target.write_text("keep target\n")
    symlink_output = tmp / "symlink.output.toml"
    symlink_output.symlink_to(symlink_target)
    expect_failure(good, "output path already exists", symlink_output)
    if symlink_target.read_text() != "keep target\n":
        raise AssertionError("symlink target was modified")

    symlink_release = tmp / "release-link.json"
    symlink_release.symlink_to(good.release)
    result = subprocess.run(
        [
            sys.executable,
            str(renderer),
            str(symlink_release),
            str(good.final),
            str(good.template),
            str(tmp / "symlink-input.output.toml"),
        ],
        text=True,
        capture_output=True,
    )
    if result.returncode == 0 or "non-symlink regular file" not in result.stderr:
        raise AssertionError("symlink RELEASE input was accepted")

    def wrong_renderer_hash(value: dict[str, Any]) -> None:
        value["archive_gateway_render_contract"]["renderer_sha256"] = "4" * 64

    expect_failure(
        Case(tmp, "wrong-renderer", release_mutator=wrong_renderer_hash),
        "renderer bytes differ",
    )

    drifted_template = template + b"# drift\n"
    template_drift = Case(tmp, "template-drift")
    template_drift.template.write_bytes(drifted_template)
    expect_failure(template_drift, "template bytes differ")

    def wrong_binding(value: dict[str, Any]) -> None:
        value["archive_gateway_render_contract"]["bindings"][
            "expected_archive_app_hash"
        ] = "lower(FINAL-CHECKPOINT.json.checkpoint_state.app_hash)"

    expect_failure(
        Case(tmp, "wrong-binding", release_mutator=wrong_binding),
        "render contract changed",
    )

    def concrete_release_hash(value: dict[str, Any]) -> None:
        value["deployment_configs"]["zerone_1_archive_gateway"]["sha256"] = "5" * 64

    expect_failure(
        Case(tmp, "concrete-release-hash", release_mutator=concrete_release_hash),
        "static mapping is not an exact object",
    )

    def changed_static_region(value: dict[str, Any]) -> None:
        value["archive_render_contract"]["static_constraints"]["region"] = "iad"

    expect_failure(
        Case(tmp, "static-region", release_mutator=changed_static_region),
        "static constraints changed",
    )

    def wrong_release_pair(value: dict[str, Any]) -> None:
        value["authority_chain"]["release_packet"]["sha256"] = "6" * 64

    expect_failure(
        Case(tmp, "wrong-release-pair", final_mutator=wrong_release_pair),
        "bound to different RELEASE bytes",
    )

    def mismatched_height(value: dict[str, Any]) -> None:
        value["excluded_post_anchor_state"]["abci_last_applied_height"] = "1000"

    expect_failure(
        Case(tmp, "height-mismatch", final_mutator=mismatched_height),
        "post-anchor height does not equal A",
    )

    def unbounded_height(value: dict[str, Any]) -> None:
        value["final_application_block"]["height"] = "1000000000000000000"
        value["excluded_post_anchor_state"]["abci_last_applied_height"] = (
            "1000000000000000000"
        )
        value["archive"]["blockstore_status_height"] = "1000000000000000000"
        value["archive"]["abci_last_applied_height"] = "1000000000000000000"

    expect_failure(
        Case(tmp, "unbounded-height", final_mutator=unbounded_height),
        "canonical bounded positive integer",
    )

    def lowercase_final_app_hash(value: dict[str, Any]) -> None:
        value["excluded_post_anchor_state"]["app_hash"] = app_hash.lower()

    expect_failure(
        Case(tmp, "lowercase-final-app", final_mutator=lowercase_final_app_hash),
        "AppHash must be exactly 64 uppercase",
    )

    def lowercase_final_block_hash(value: dict[str, Any]) -> None:
        value["final_application_block"]["block_id_hash"] = block_hash.lower()

    expect_failure(
        Case(tmp, "lowercase-final-block", final_mutator=lowercase_final_block_hash),
        "block ID hash must be exactly 64 uppercase",
    )

    def public_archive(value: dict[str, Any]) -> None:
        value["archive"]["public_service_authorized"] = True

    expect_failure(
        Case(tmp, "public-archive", final_mutator=public_archive),
        "private archive A/A boundary is inconsistent",
    )

    duplicate = template + (
        b'# duplicate replace-zerone-1-archive-gateway-app\n'
    )
    expect_failure(
        Case(tmp, "duplicate-placeholder", template_bytes=duplicate),
        "exactly one app placeholder; found 2",
    )

    missing = template.replace(
        b"REPLACE_WITH_LOWERCASE_FINAL_APPLICATION_BLOCK_ID_HASH", b"already-filled"
    )
    expect_failure(
        Case(tmp, "missing-placeholder", template_bytes=missing),
        "exactly one expected_archive_block_hash placeholder; found 0",
    )

    residual = template + b"# REPLACE_UNKNOWN\n"
    expect_failure(
        Case(tmp, "residual-placeholder", template_bytes=residual),
        "retains an active placeholder",
    )

    wrong_env = template.replace(
        b'EXPECTED_CHAIN_ID = "zerone-1"', b'EXPECTED_CHAIN_ID = "other"'
    )
    expect_failure(
        Case(tmp, "wrong-env", template_bytes=wrong_env),
        "env table differs",
    )

    pretty_json = Case(tmp, "pretty-json")
    pretty_json.final.write_text(
        json.dumps(json.loads(pretty_json.final.read_text()), indent=2) + "\n"
    )
    expect_failure(pretty_json, "FINAL is not canonical JSON")

    # Exercise the repository template as well as the minimal unit fixture, so a
    # renamed or missing production placeholder fails this focused test.
    production = Case(
        tmp, "production", template_bytes=production_template.read_bytes()
    )
    run_ok(production)

print("archive gateway config renderer test: PASS")
PY
