#!/usr/bin/env python3
"""Render the public zerone-1 archive gateway config from RELEASE and FINAL."""

from __future__ import annotations

import hashlib
import json
import os
import pathlib
import re
import stat
import sys
import tempfile
import tomllib
from typing import Any


RENDERER_PATH = "deploy/query-gateway/render-archive-gateway-config.py"
TEMPLATE_PATH = "deploy/query-gateway/fly.zerone-1-archive.public.example.toml"
CONTRACT_SCHEMA = "zerone-1-archive-gateway-render-contract-v1"
RELEASE_SCHEMA = "zerone-2-release-packet-v2"
FINAL_SCHEMA = "zerone-final-checkpoint-v4"

LOWER_HASH = re.compile(r"^[0-9a-f]{64}$")
UPPER_HASH = re.compile(r"^[0-9A-F]{64}$")
POSITIVE_HEIGHT = re.compile(r"^[1-9][0-9]{0,17}$")
FLY_APP = re.compile(r"^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$")
IMAGE_REF = re.compile(
    r"^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?(?::[0-9]{1,5})?/"
    r"[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*"
    r"@sha256:[0-9a-f]{64}$"
)

EXPECTED_BINDINGS = {
    "app": "deployment_configs.zerone_1_archive_gateway.app",
    "primary_region": "archive_render_contract.static_constraints.region",
    "image_ref": "deployment_configs.zerone_1_archive_gateway.image_ref",
    "upstream_host": 'archive_render_contract.static_constraints.app + ".internal"',
    "expected_archive_height": (
        "FINAL-CHECKPOINT.json.final_application_block.height"
    ),
    "expected_archive_app_hash": (
        "lower(FINAL-CHECKPOINT.json.excluded_post_anchor_state.app_hash)"
    ),
    "expected_archive_block_hash": (
        "lower(FINAL-CHECKPOINT.json.final_application_block.block_id_hash)"
    ),
}

PLACEHOLDERS = {
    "app": b"replace-zerone-1-archive-gateway-app",
    "primary_region": b"replace-region",
    "image_ref": b"replace-with-pinned-query-gateway-image-digest",
    "upstream_host": b"replace-zerone-1-archive-app.internal",
    "expected_archive_height": b"REPLACE_WITH_A",
    "expected_archive_app_hash": b"REPLACE_WITH_LOWERCASE_POST_A_APP_HASH",
    "expected_archive_block_hash": (
        b"REPLACE_WITH_LOWERCASE_FINAL_APPLICATION_BLOCK_ID_HASH"
    ),
}

MAX_JSON_BYTES = 4 * 1024 * 1024
MAX_TEMPLATE_BYTES = 1024 * 1024


class RenderError(Exception):
    """A deterministic render precondition was not met."""


def fail(message: str) -> None:
    raise RenderError(message)


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def exact_object(value: Any, keys: set[str], label: str) -> dict[str, Any]:
    if not isinstance(value, dict) or set(value) != keys:
        fail(f"{label} is not an exact object")
    return value


def require_lower_hash(value: Any, label: str) -> str:
    if not isinstance(value, str) or not LOWER_HASH.fullmatch(value):
        fail(f"{label} must be exactly 64 lowercase hexadecimal characters")
    return value


def require_upper_hash(value: Any, label: str) -> str:
    if not isinstance(value, str) or not UPPER_HASH.fullmatch(value):
        fail(f"{label} must be exactly 64 uppercase hexadecimal characters")
    return value


def read_regular(path: pathlib.Path, label: str, limit: int) -> bytes:
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
    try:
        fd = os.open(path, flags)
    except OSError as exc:
        fail(f"could not open {label} as a non-symlink regular file: {exc}")
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode):
            fail(f"{label} must be a regular file")
        if info.st_size > limit:
            fail(f"{label} exceeds the {limit}-byte limit")
        chunks: list[bytes] = []
        remaining = limit + 1
        while remaining:
            chunk = os.read(fd, min(remaining, 65536))
            if not chunk:
                break
            chunks.append(chunk)
            remaining -= len(chunk)
        data = b"".join(chunks)
        if len(data) > limit:
            fail(f"{label} exceeds the {limit}-byte limit")
        return data
    finally:
        os.close(fd)


def parse_canonical_json(data: bytes, label: str) -> dict[str, Any]:
    try:
        text = data.decode("utf-8")
        value = json.loads(text)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        fail(f"{label} is not valid UTF-8 JSON: {exc}")
    if not isinstance(value, dict):
        fail(f"{label} root must be an object")
    canonical = (
        json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
        + "\n"
    ).encode("utf-8")
    if data != canonical:
        fail(f"{label} is not canonical JSON")
    if b"REPLACE_" in data or b"replace-" in data:
        fail(f"{label} retains a placeholder")
    return value


def validate_release(
    release: dict[str, Any],
    release_bytes: bytes,
    template_bytes: bytes,
    renderer_bytes: bytes,
) -> dict[str, str]:
    if release.get("schema") != RELEASE_SCHEMA or release.get("chain_id") != "zerone-2":
        fail("RELEASE has the wrong schema or chain ID")

    contract = exact_object(
        release.get("archive_gateway_render_contract"),
        {"schema", "renderer_path", "renderer_sha256", "template_path", "bindings"},
        "RELEASE archive-gateway render contract",
    )
    if not (
        contract["schema"] == CONTRACT_SCHEMA
        and contract["renderer_path"] == RENDERER_PATH
        and contract["template_path"] == TEMPLATE_PATH
        and contract["bindings"] == EXPECTED_BINDINGS
    ):
        fail("RELEASE archive-gateway render contract changed")
    renderer_hash = require_lower_hash(
        contract["renderer_sha256"], "RELEASE archive-gateway renderer hash"
    )
    if renderer_hash != sha256(renderer_bytes):
        fail("renderer bytes differ from the RELEASE archive-gateway contract")

    template_hashes = release.get("phase_dependent_config_template_sha256")
    if not isinstance(template_hashes, dict):
        fail("RELEASE phase-dependent template hash mapping is missing")
    template_hash = require_lower_hash(
        template_hashes.get("zerone_1_archive_gateway"),
        "RELEASE archive-gateway template hash",
    )
    if template_hash != sha256(template_bytes):
        fail("template bytes differ from the RELEASE archive-gateway template hash")

    deployment_configs = release.get("deployment_configs")
    if not isinstance(deployment_configs, dict):
        fail("RELEASE deployment config mapping is missing")
    mapping = exact_object(
        deployment_configs.get("zerone_1_archive_gateway"),
        {"app", "role", "image_component", "image_ref"},
        "RELEASE archive-gateway static mapping",
    )
    app = mapping["app"]
    if not isinstance(app, str) or not FLY_APP.fullmatch(app):
        fail("RELEASE archive-gateway app is not a canonical Fly app name")
    if app in {"zerone-1", "zerone-1-observer", "zerone-1-archive"}:
        fail("RELEASE archive-gateway app reuses a reserved origin identity")
    if mapping["role"] != "zerone-1-archive-query":
        fail("RELEASE archive-gateway role changed")
    if mapping["image_component"] != "query_gateway":
        fail("RELEASE archive-gateway image component changed")
    image_ref = mapping["image_ref"]
    if not isinstance(image_ref, str) or not IMAGE_REF.fullmatch(image_ref):
        fail("RELEASE archive-gateway image is not an immutable digest reference")
    components = release.get("components")
    if not isinstance(components, dict) or not isinstance(
        components.get("query_gateway"), dict
    ):
        fail("RELEASE query-gateway component is missing")
    if components["query_gateway"].get("image_ref") != image_ref:
        fail("RELEASE archive-gateway image does not join its component")

    archive_render = release.get("archive_render_contract")
    if not isinstance(archive_render, dict) or archive_render.get("schema") != (
        "zerone-1-archive-render-contract-v1"
    ):
        fail("RELEASE private archive render contract is missing or changed")
    static = exact_object(
        archive_render.get("static_constraints"),
        {
            "app",
            "volume",
            "region",
            "image_component",
            "archive_candidate_role",
            "archive_role",
            "deployment_strategy",
            "public_fly_service",
            "public_ip",
            "persistent_peers",
        },
        "RELEASE private archive static constraints",
    )
    if not (
        static["app"] == "zerone-1-archive"
        and static["volume"] == "zerone_archive_data"
        and static["region"] == "lhr"
        and static["image_component"] == "zerone_1_halt"
        and static["archive_candidate_role"] == "archive-candidate"
        and static["archive_role"] == "archive"
        and static["deployment_strategy"] == "immediate"
        and static["public_fly_service"] is False
        and static["public_ip"] is False
        and static["persistent_peers"] == []
    ):
        fail("RELEASE private archive static constraints changed")

    # FINAL must point back to these exact RELEASE bytes. The detached signatures
    # and the rest of the authority chain are verified by the calling gate.
    return {
        "release_sha256": sha256(release_bytes),
        "app": app,
        "primary_region": static["region"],
        "image_ref": image_ref,
        "upstream_host": f'{static["app"]}.internal',
    }


def validate_final(final: dict[str, Any], inputs: dict[str, str]) -> dict[str, str]:
    if not (
        final.get("schema") == FINAL_SCHEMA
        and final.get("status") == "frozen"
        and final.get("chain_id") == "zerone-1"
    ):
        fail("FINAL has the wrong schema, status, or chain ID")

    authority_chain = final.get("authority_chain")
    if not isinstance(authority_chain, dict):
        fail("FINAL authority chain is missing")
    release_pair = exact_object(
        authority_chain.get("release_packet"),
        {"sha256", "detached_signature_sha256"},
        "FINAL RELEASE pair",
    )
    require_lower_hash(release_pair["sha256"], "FINAL RELEASE payload hash")
    require_lower_hash(
        release_pair["detached_signature_sha256"], "FINAL RELEASE signature hash"
    )
    if release_pair["sha256"] != inputs["release_sha256"]:
        fail("FINAL is bound to different RELEASE bytes")

    application = final.get("final_application_block")
    if not isinstance(application, dict):
        fail("FINAL final application block is missing")
    height = application.get("height")
    if not isinstance(height, str) or not POSITIVE_HEIGHT.fullmatch(height):
        fail("FINAL application height A is not a canonical bounded positive integer")
    block_hash = require_upper_hash(
        application.get("block_id_hash"), "FINAL application block ID hash"
    ).lower()
    if application.get("commit_canonical") is not True:
        fail("FINAL application block is not marked canonical")

    excluded = final.get("excluded_post_anchor_state")
    if not isinstance(excluded, dict):
        fail("FINAL excluded post-anchor state is missing")
    if excluded.get("abci_last_applied_height") != height:
        fail("FINAL excluded post-anchor height does not equal A")
    app_hash = require_upper_hash(
        excluded.get("app_hash"), "FINAL excluded post-anchor AppHash"
    ).lower()
    if excluded.get("included_in_successor_inventory") is not False:
        fail("FINAL post-anchor application state is not excluded from the successor")

    archive = final.get("archive")
    if not isinstance(archive, dict) or not (
        archive.get("blockstore_status_height") == height
        and archive.get("abci_last_applied_height") == height
        and archive.get("halt_trigger_block_present") is False
        and archive.get("public_service_authorized") is False
    ):
        fail("FINAL private archive A/A boundary is inconsistent")

    successor = final.get("successor")
    successor_release = (
        successor.get("release") if isinstance(successor, dict) else None
    )
    if not isinstance(successor_release, dict) or (
        successor_release.get("query_gateway_image_ref") != inputs["image_ref"]
    ):
        fail("FINAL query-gateway image does not join RELEASE")

    return {
        "expected_archive_height": height,
        "expected_archive_app_hash": app_hash,
        "expected_archive_block_hash": block_hash,
    }


def render_bytes(
    release_bytes: bytes,
    final_bytes: bytes,
    template_bytes: bytes,
    renderer_bytes: bytes,
) -> bytes:
    release = parse_canonical_json(release_bytes, "RELEASE")
    final = parse_canonical_json(final_bytes, "FINAL")
    values = validate_release(release, release_bytes, template_bytes, renderer_bytes)
    values.update(validate_final(final, values))

    for label, placeholder in PLACEHOLDERS.items():
        count = template_bytes.count(placeholder)
        if count != 1:
            fail(
                f"template must contain exactly one {label} placeholder; found {count}"
            )

    rendered = template_bytes
    for label, placeholder in PLACEHOLDERS.items():
        rendered = rendered.replace(placeholder, values[label].encode("ascii"))
    if b"REPLACE_" in rendered or b"replace-" in rendered:
        fail("rendered config retains an active placeholder")

    try:
        config = tomllib.loads(rendered.decode("utf-8"))
    except (UnicodeDecodeError, tomllib.TOMLDecodeError) as exc:
        fail(f"rendered config is not valid UTF-8 TOML: {exc}")
    if not isinstance(config, dict):
        fail("rendered config root is not a table")
    if config.get("app") != values["app"]:
        fail("rendered config app differs from RELEASE")
    if config.get("primary_region") != values["primary_region"]:
        fail("rendered config region differs from RELEASE")
    if config.get("build") != {"image": values["image_ref"]}:
        fail("rendered config build table differs from RELEASE")
    expected_env = {
        "GATEWAY_ROLE": "zerone-1-archive-query",
        "EXPECTED_CHAIN_ID": "zerone-1",
        "EXPECTED_ARCHIVE_HEIGHT": values["expected_archive_height"],
        "EXPECTED_ARCHIVE_APP_HASH": values["expected_archive_app_hash"],
        "EXPECTED_ARCHIVE_BLOCK_HASH": values["expected_archive_block_hash"],
        "UPSTREAM_HOST": values["upstream_host"],
    }
    if config.get("env") != expected_env:
        fail("rendered config env table differs from the RELEASE/FINAL bindings")
    return rendered


def write_atomic_no_replace(path: pathlib.Path, data: bytes) -> None:
    if os.path.lexists(path):
        fail("output path already exists or is a symlink")
    parent = path.parent
    try:
        parent_info = os.lstat(parent)
    except OSError as exc:
        fail(f"could not inspect output directory: {exc}")
    if not stat.S_ISDIR(parent_info.st_mode) or stat.S_ISLNK(parent_info.st_mode):
        fail("output parent must be a real directory, not a symlink")

    temporary: str | None = None
    fd = -1
    try:
        fd, temporary = tempfile.mkstemp(prefix=f".{path.name}.", dir=parent)
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "wb", closefd=True) as handle:
            fd = -1
            handle.write(data)
            handle.flush()
            os.fsync(handle.fileno())
        try:
            os.link(temporary, path, follow_symlinks=False)
        except FileExistsError:
            fail("output path appeared during rendering; refusing to overwrite it")
        os.unlink(temporary)
        temporary = None
        try:
            directory_fd = os.open(parent, os.O_RDONLY | getattr(os, "O_DIRECTORY", 0))
            try:
                os.fsync(directory_fd)
            finally:
                os.close(directory_fd)
        except OSError:
            # The file is already linked atomically. Some filesystems do not
            # support directory fsync, so lack of that durability hint is not a
            # reason to remove a successfully rendered artifact.
            pass
    except RenderError:
        raise
    except OSError as exc:
        fail(f"could not write output atomically: {exc}")
    finally:
        if fd >= 0:
            os.close(fd)
        if temporary is not None:
            try:
                os.unlink(temporary)
            except FileNotFoundError:
                pass


def usage() -> None:
    print(
        "usage: render-archive-gateway-config.py "
        "RELEASE-PACKET.json FINAL-CHECKPOINT.json TEMPLATE.toml OUTPUT.toml",
        file=sys.stderr,
    )


def main() -> int:
    if len(sys.argv) != 5:
        usage()
        return 2
    release_path = pathlib.Path(sys.argv[1])
    final_path = pathlib.Path(sys.argv[2])
    template_path = pathlib.Path(sys.argv[3])
    output_path = pathlib.Path(sys.argv[4])
    renderer_path = pathlib.Path(__file__)
    try:
        release_bytes = read_regular(release_path, "RELEASE", MAX_JSON_BYTES)
        final_bytes = read_regular(final_path, "FINAL", MAX_JSON_BYTES)
        template_bytes = read_regular(
            template_path, "archive-gateway template", MAX_TEMPLATE_BYTES
        )
        renderer_bytes = read_regular(
            renderer_path, "archive-gateway renderer", MAX_JSON_BYTES
        )
        rendered = render_bytes(
            release_bytes, final_bytes, template_bytes, renderer_bytes
        )
        write_atomic_no_replace(output_path, rendered)
    except RenderError as exc:
        print(f"render-archive-gateway-config: {exc}", file=sys.stderr)
        return 1
    print(f"render-archive-gateway-config: MATCH {sha256(rendered)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
