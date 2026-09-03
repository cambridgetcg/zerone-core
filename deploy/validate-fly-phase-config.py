#!/usr/bin/env python3
"""Structural policy gate for phase-authorized Fly TOML profiles."""

from __future__ import annotations

import pathlib
import re
import sys
import tomllib


def fail(message: str) -> None:
    raise SystemExit(f"validate-fly-phase-config: {message}")


def require_keys(value: dict, expected: set[str], label: str) -> None:
    if set(value) != expected:
        fail(f"{label} keys differ from the exact policy")


def require_immediate(data: dict) -> None:
    deploy = data.get("deploy")
    if not isinstance(deploy, dict) or deploy.get("strategy") != "immediate":
        fail("stateful profile must use deploy.strategy=immediate")
    require_keys(deploy, {"strategy"}, "stateful deploy")


def forbid_execution_overrides(data: dict) -> None:
    for key in ("files", "processes", "experimental"):
        if key in data:
            fail(f"profile contains forbidden execution override {key}")
    deploy = data.get("deploy")
    if isinstance(deploy, dict) and "release_command" in deploy:
        fail("profile contains forbidden deploy.release_command")


def require_exact_root(data: dict, key: str) -> None:
    stateful_successor = {
        "app",
        "primary_region",
        "kill_signal",
        "kill_timeout",
        "build",
        "deploy",
        "env",
        "mounts",
        "vm",
    }
    gateway = {
        "app",
        "primary_region",
        "kill_signal",
        "kill_timeout",
        "build",
        "env",
        "vm",
    }
    old_chain = {
        "app",
        "primary_region",
        "build",
        "deploy",
        "env",
        "mounts",
        "vm",
    }
    expected = {
        "zerone_2_validator": stateful_successor,
        "zerone_2_edge_private": stateful_successor,
        "zerone_2_edge_query_soak": stateful_successor,
        "zerone_2_edge_public": stateful_successor | {"services"},
        "zerone_2_gateway_private": gateway,
        "zerone_2_gateway_public": gateway | {"services"},
        "zerone_1_archive_gateway": gateway | {"services"},
        "zerone_1_halt_signer": old_chain,
        "zerone_1_observer": old_chain,
    }
    if key not in expected:
        fail("unsupported config key")
    require_keys(data, expected[key], f"{key} TOML root")


def forbid_public_shape(data: dict) -> None:
    for key in ("services", "http_service", "statics"):
        if key in data:
            fail(f"private profile contains forbidden {key}")


def require_mount(data: dict, source: str | None = None) -> None:
    mount = data.get("mounts")
    if not isinstance(mount, dict):
        fail("stateful profile must contain exactly one [mounts] table")
    require_keys(mount, {"source", "destination"}, "mount")
    if mount.get("destination") != "/data":
        fail("stateful mount destination must be /data")
    actual = mount.get("source")
    if not isinstance(actual, str) or not actual:
        fail("stateful mount source must be one nonempty string")
    if source is not None and actual != source:
        fail(f"mount source must be {source}")


def require_exact_env(data: dict, key: str) -> None:
    expected = {
        "zerone_2_validator": {
            "NODE_ROLE",
            "ZERONE_HOME",
            "MONIKER",
            "QUERY_ORIGIN_ENABLED",
            "PERSISTENT_PEERS",
            "PRIVATE_PEER_IDS",
        },
        "zerone_2_edge_private": {
            "NODE_ROLE",
            "ZERONE_HOME",
            "MONIKER",
            "PERSISTENT_PEERS",
            "PRIVATE_PEER_IDS",
            "CORS_ALLOWED_ORIGINS_JSON",
            "QUERY_ORIGIN_ENABLED",
        },
        "zerone_2_edge_query_soak": {
            "NODE_ROLE",
            "ZERONE_HOME",
            "MONIKER",
            "PERSISTENT_PEERS",
            "PRIVATE_PEER_IDS",
            "CORS_ALLOWED_ORIGINS_JSON",
            "QUERY_ORIGIN_ENABLED",
        },
        "zerone_2_edge_public": {
            "NODE_ROLE",
            "ZERONE_HOME",
            "MONIKER",
            "PERSISTENT_PEERS",
            "PRIVATE_PEER_IDS",
            "P2P_EXTERNAL_ADDRESS",
            "CORS_ALLOWED_ORIGINS_JSON",
            "QUERY_ORIGIN_ENABLED",
        },
        "zerone_2_gateway_private": {
            "GATEWAY_ROLE",
            "EXPECTED_CHAIN_ID",
            "UPSTREAM_HOST",
        },
        "zerone_2_gateway_public": {
            "GATEWAY_ROLE",
            "EXPECTED_CHAIN_ID",
            "UPSTREAM_HOST",
        },
        "zerone_1_archive_gateway": {
            "GATEWAY_ROLE",
            "EXPECTED_CHAIN_ID",
            "EXPECTED_ARCHIVE_HEIGHT",
            "EXPECTED_ARCHIVE_APP_HASH",
            "EXPECTED_ARCHIVE_BLOCK_HASH",
            "UPSTREAM_HOST",
        },
        "zerone_1_halt_signer": {
            "MONIKER",
            "NODE_ROLE",
            "ZERONE_HOME",
            "ZERONE_CHECKPOINT_STATE_HEIGHT",
            "ZERONE_FINAL_COMMITTED_HEIGHT",
            "ZERONE_HALT_TRIGGER_HEIGHT",
        },
        "zerone_1_observer": {
            "MONIKER",
            "NODE_ROLE",
            "ZERONE_HOME",
            "PERSISTENT_PEERS",
            "ZERONE_CHECKPOINT_STATE_HEIGHT",
            "ZERONE_FINAL_COMMITTED_HEIGHT",
            "ZERONE_HALT_TRIGGER_HEIGHT",
        },
    }
    if key not in expected:
        fail("unsupported config key")
    env = data.get("env")
    if not isinstance(env, dict):
        fail("profile must contain one [env] table")
    require_keys(env, expected[key], f"{key} env")

    if key.startswith("zerone_2_") and "NODE_ROLE" in env:
        if env.get("ZERONE_HOME") != "/data/.zeroned":
            fail("zerone-2 node home must be /data/.zeroned")
    if key in {"zerone_1_halt_signer", "zerone_1_observer"}:
        if env.get("ZERONE_HOME") != "/data/.zeroned":
            fail("zerone-1 halt node home must be /data/.zeroned")
    if key == "zerone_2_validator":
        if env.get("QUERY_ORIGIN_ENABLED") != "false":
            fail("validator query origin must remain disabled")
    if key == "zerone_2_edge_private":
        if env.get("QUERY_ORIGIN_ENABLED") != "false":
            fail("private edge query origin must remain disabled")
    if key in {"zerone_2_edge_query_soak", "zerone_2_edge_public"}:
        if env.get("QUERY_ORIGIN_ENABLED") != "true":
            fail("query edge origin must be enabled")
    if key.startswith("zerone_2_edge_"):
        if env.get("CORS_ALLOWED_ORIGINS_JSON") != "[]":
            fail("edge direct CORS origins must remain empty")
    if key in {"zerone_2_gateway_private", "zerone_2_gateway_public"}:
        if env.get("GATEWAY_ROLE") != "zerone-2-query":
            fail("zerone-2 gateway role changed")
        if env.get("EXPECTED_CHAIN_ID") != "zerone-2":
            fail("zerone-2 gateway chain ID changed")
    if key == "zerone_1_archive_gateway":
        if env.get("GATEWAY_ROLE") != "zerone-1-archive-query":
            fail("archive gateway role changed")
        if env.get("EXPECTED_CHAIN_ID") != "zerone-1":
            fail("archive gateway chain ID changed")


def require_context(
    data: dict,
    schema: str,
    key: str,
    expected_upstream: str,
    expected_f: str,
    expected_a: str,
    expected_h: str,
    expected_archive_app_hash: str,
    expected_archive_block_hash: str,
) -> None:
    dark_keys = {
        "zerone_2_validator",
        "zerone_2_edge_private",
        "zerone_2_edge_query_soak",
        "zerone_2_gateway_private",
    }
    cutover_keys = {"zerone_1_halt_signer", "zerone_1_observer"}
    open_keys = {
        "zerone_2_edge_public",
        "zerone_2_gateway_public",
        "zerone_1_archive_gateway",
    }
    expected_schema = {
        **{key: "zerone-2-dark-start-decision-v1" for key in dark_keys},
        **{key: "zerone-2-cutover-decision-v1" for key in cutover_keys},
        **{key: "zerone-2-open-beta-decision-v1" for key in open_keys},
    }
    if expected_schema.get(key) != schema:
        fail("config key is incompatible with the phase authority schema")

    gateway_keys = {
        "zerone_2_gateway_private",
        "zerone_2_gateway_public",
        "zerone_1_archive_gateway",
    }
    if key in gateway_keys:
        if expected_upstream == "-":
            fail("gateway expected upstream was not supplied")
        if data["env"].get("UPSTREAM_HOST") != expected_upstream:
            fail("gateway upstream differs from the signed release topology")
    elif expected_upstream != "-":
        fail("unexpected upstream context for a non-gateway profile")

    if key in cutover_keys:
        expected = (expected_f, expected_a, expected_h)
        if not all(value.isdecimal() and value != "0" for value in expected):
            fail("signed F/A/H context is malformed")
        env = data["env"]
        if (
            env.get("ZERONE_CHECKPOINT_STATE_HEIGHT"),
            env.get("ZERONE_FINAL_COMMITTED_HEIGHT"),
            env.get("ZERONE_HALT_TRIGGER_HEIGHT"),
        ) != expected:
            fail("halt profile F/A/H differs from the signed CUTOVER plan")
        if (expected_archive_app_hash, expected_archive_block_hash) != ("-", "-"):
            fail("unexpected archive hash context for a CUTOVER profile")
    elif key == "zerone_1_archive_gateway":
        if expected_f != "-" or expected_h != "-":
            fail("archive gateway accepts only A/E/B checkpoint context")
        if not re.fullmatch(r"[1-9][0-9]{0,17}", expected_a):
            fail("signed archive height A is malformed or cannot admit an A+1 proof")
        if not re.fullmatch(r"[0-9a-f]{64}", expected_archive_app_hash):
            fail("signed archive app hash E must be exactly 64 lowercase hex characters")
        if not re.fullmatch(r"[0-9a-f]{64}", expected_archive_block_hash):
            fail("signed archive block hash B must be exactly 64 lowercase hex characters")
        env = data["env"]
        if env.get("EXPECTED_ARCHIVE_HEIGHT") != expected_a:
            fail("archive gateway height differs from verified FINAL A")
        if env.get("EXPECTED_ARCHIVE_APP_HASH") != expected_archive_app_hash:
            fail("archive gateway app hash differs from verified FINAL E")
        if env.get("EXPECTED_ARCHIVE_BLOCK_HASH") != expected_archive_block_hash:
            fail("archive gateway block hash differs from verified FINAL B")
    else:
        if (expected_f, expected_a, expected_h) != ("-", "-", "-"):
            fail("unexpected F/A/H context for this profile")
        if (expected_archive_app_hash, expected_archive_block_hash) != ("-", "-"):
            fail("unexpected archive hash context for this profile")


def require_service_base(service: dict, internal_port: int) -> None:
    if service.get("internal_port") != internal_port:
        fail(f"service internal_port must be {internal_port}")
    if service.get("protocol") != "tcp":
        fail("service protocol must be tcp")
    if service.get("auto_stop_machines") is not False:
        fail("service auto_stop_machines must be false")
    if service.get("auto_start_machines") is not True:
        fail("service auto_start_machines must be true")
    if service.get("min_machines_running") != 1:
        fail("service min_machines_running must be 1")


def require_public_edge(data: dict) -> None:
    if "http_service" in data or "statics" in data:
        fail("public edge contains http_service/statics")
    services = data.get("services")
    if not isinstance(services, list) or len(services) != 1:
        fail("public edge must contain exactly one service")
    service = services[0]
    if not isinstance(service, dict):
        fail("public edge service is malformed")
    require_keys(
        service,
        {
            "internal_port",
            "protocol",
            "auto_stop_machines",
            "auto_start_machines",
            "min_machines_running",
            "concurrency",
            "ports",
            "tcp_checks",
        },
        "public edge service",
    )
    require_service_base(service, 26656)
    concurrency = service.get("concurrency")
    if concurrency != {"type": "connections", "soft_limit": 80, "hard_limit": 120}:
        fail("public edge concurrency policy changed")
    ports = service.get("ports")
    if ports != [{"port": 26656}]:
        fail("public edge must expose only TCP port 26656")
    checks = service.get("tcp_checks")
    if not isinstance(checks, list) or len(checks) != 1:
        fail("public edge must contain one TCP health check")
    require_keys(checks[0], {"interval", "timeout", "grace_period"}, "TCP check")


def require_public_gateway(data: dict) -> None:
    if "http_service" in data or "statics" in data:
        fail("public gateway contains http_service/statics")
    services = data.get("services")
    if not isinstance(services, list) or len(services) != 1:
        fail("public gateway must contain exactly one service")
    service = services[0]
    if not isinstance(service, dict):
        fail("public gateway service is malformed")
    require_keys(
        service,
        {
            "internal_port",
            "protocol",
            "auto_stop_machines",
            "auto_start_machines",
            "min_machines_running",
            "concurrency",
            "ports",
            "http_checks",
        },
        "public gateway service",
    )
    require_service_base(service, 8080)
    concurrency = service.get("concurrency")
    if not isinstance(concurrency, dict):
        fail("public gateway concurrency policy is missing")
    require_keys(concurrency, {"type", "soft_limit", "hard_limit"}, "gateway concurrency")
    if concurrency.get("type") != "requests":
        fail("public gateway concurrency type must be requests")
    if not all(isinstance(concurrency.get(k), int) and concurrency[k] > 0 for k in ("soft_limit", "hard_limit")):
        fail("public gateway concurrency limits must be positive integers")
    if concurrency["soft_limit"] > concurrency["hard_limit"]:
        fail("public gateway soft concurrency limit exceeds hard limit")
    ports = service.get("ports")
    if ports != [
        {"port": 80, "handlers": ["http"], "force_https": True},
        {"port": 443, "handlers": ["tls", "http"]},
    ]:
        fail("public gateway must expose only HTTPS 80/443")
    checks = service.get("http_checks")
    if not isinstance(checks, list) or len(checks) != 1:
        fail("public gateway must contain one HTTP health check")
    require_keys(
        checks[0],
        {"interval", "timeout", "grace_period", "method", "path"},
        "gateway HTTP check",
    )
    if checks[0].get("method") != "GET" or checks[0].get("path") != "/gateway-health":
        fail("public gateway health check policy changed")


def main() -> None:
    if len(sys.argv) != 10:
        fail(
            "usage: CONFIG AUTHORITY_SCHEMA CONFIG_KEY "
            "EXPECTED_UPSTREAM_OR_DASH EXPECTED_F_OR_DASH "
            "EXPECTED_A_OR_DASH EXPECTED_H_OR_DASH "
            "EXPECTED_ARCHIVE_APP_HASH_OR_DASH "
            "EXPECTED_ARCHIVE_BLOCK_HASH_OR_DASH"
        )
    path = pathlib.Path(sys.argv[1])
    schema = sys.argv[2]
    key = sys.argv[3]
    expected_upstream = sys.argv[4]
    expected_f = sys.argv[5]
    expected_a = sys.argv[6]
    expected_h = sys.argv[7]
    expected_archive_app_hash = sys.argv[8]
    expected_archive_block_hash = sys.argv[9]
    try:
        with path.open("rb") as handle:
            data = tomllib.load(handle)
    except (OSError, tomllib.TOMLDecodeError) as exc:
        fail(f"could not parse TOML: {exc}")
    if not isinstance(data, dict):
        fail("TOML root is not a table")

    forbid_execution_overrides(data)
    require_exact_root(data, key)

    private_keys = {
        "zerone_2_validator",
        "zerone_2_edge_private",
        "zerone_2_edge_query_soak",
        "zerone_2_gateway_private",
        "zerone_1_halt_signer",
        "zerone_1_observer",
    }
    stateful_keys = {
        "zerone_2_validator",
        "zerone_2_edge_private",
        "zerone_2_edge_query_soak",
        "zerone_2_edge_public",
        "zerone_1_halt_signer",
        "zerone_1_observer",
    }

    if key in private_keys:
        forbid_public_shape(data)
    if key in {
        "zerone_2_gateway_private",
        "zerone_2_gateway_public",
        "zerone_1_archive_gateway",
    } and "mounts" in data:
        fail("stateless query gateway must not contain mounts")
    require_exact_env(data, key)
    if key == "zerone_2_edge_public":
        require_public_edge(data)
    elif key in {"zerone_2_gateway_public", "zerone_1_archive_gateway"}:
        require_public_gateway(data)
    if key in stateful_keys:
        require_immediate(data)
        if key == "zerone_1_halt_signer":
            require_mount(data, "zerone_data")
        elif key == "zerone_1_observer":
            require_mount(data, "zerone_observer_data")
        else:
            require_mount(data)
            if data["mounts"]["source"] in {
                "zerone_data",
                "zerone_observer_data",
                "zerone_archive_data",
            }:
                fail("zerone-2 stateful profile reuses a reserved zerone-1 volume")

    if schema not in {
        "zerone-2-dark-start-decision-v1",
        "zerone-2-cutover-decision-v1",
        "zerone-2-open-beta-decision-v1",
    }:
        fail("unsupported authority schema")
    require_context(
        data,
        schema,
        key,
        expected_upstream,
        expected_f,
        expected_a,
        expected_h,
        expected_archive_app_hash,
        expected_archive_block_hash,
    )


if __name__ == "__main__":
    main()
