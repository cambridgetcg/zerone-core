#!/usr/bin/env python3
"""Build coherent canonical authority artifacts for local fail-closed tests."""

from __future__ import annotations

import argparse
import base64
import copy
import hashlib
import json
import pathlib
import re
from typing import Any


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def read_json(path: pathlib.Path) -> Any:
    return json.loads(path.read_text())


def fill_placeholders(value: Any, main: str, transition: str) -> Any:
    if isinstance(value, dict):
        return {key: fill_placeholders(item, main, transition) for key, item in value.items()}
    if isinstance(value, list):
        return [fill_placeholders(item, main, transition) for item in value]
    if not isinstance(value, str) or not ("REPLACE_" in value or "replace-" in value):
        return value
    upper = value.upper()
    if "TRANSITION" in upper and "FINGERPRINT" in upper:
        return transition
    if "FINGERPRINT" in upper:
        return main
    if "RFC3339" in upper or "YYYY_MM_DD" in upper:
        return "2026-07-10T12:00:00Z"
    if "UPPERCASE" in upper:
        return "A" * 64
    if "40_HEX" in upper or "NODE_ID" in upper:
        return "1" * 40
    if "BASE64" in upper or "PUBLIC_KEY" in upper or "PUBKEY" in upper:
        return base64.b64encode(b"p" * 32).decode()
    if "SHA" in upper or "64_HEX" in upper or "HASH" in upper:
        return "a" * 64
    if "HEIGHT" in upper or value in {"REPLACE_F", "REPLACE_A", "REPLACE_H"}:
        return "100"
    if "IMAGE" in upper or "REGISTRY" in upper:
        return "registry.example/fixture@sha256:" + "a" * 64
    if "ADDRESS" in upper:
        return "zerone1fixture"
    if "COMMIT" in upper:
        return "1" * 40
    if "GO_OR_NO_GO" in upper:
        return "GO"
    return "fixture"


def canonical_bytes(value: Any) -> bytes:
    return (
        json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
        + "\n"
    ).encode()


def write_json(output: pathlib.Path, name: str, value: Any) -> None:
    (output / name).write_bytes(canonical_bytes(value))


def pair(output: pathlib.Path, payload: str, signature: str) -> dict[str, str]:
    return {
        "sha256": digest((output / payload).read_bytes()),
        "detached_signature_sha256": digest((output / signature).read_bytes()),
    }


def mapping(app: str, role: str, component: str, image: str, config: str) -> dict[str, str]:
    return {
        "app": app,
        "role": role,
        "image_component": component,
        "image_ref": image,
        "sha256": config,
    }


COMPONENT_ARTIFACT_FILES = {
    "zerone_1_halt": {
        "sbom": "ZERONE-1-HALT-SBOM.json",
        "provenance": "ZERONE-1-HALT-PROVENANCE.json",
        "signature": "ZERONE-1-HALT-SIGNATURE-EVIDENCE.json",
        "signature_bundle": "ZERONE-1-HALT-SIGNATURE-BUNDLE.json",
        "vulnerability_decision": "ZERONE-1-HALT-VULNERABILITY-DECISION.json",
        "vulnerability_scan": "ZERONE-1-HALT-VULNERABILITY-SCAN.json",
    },
    "zerone_2_runtime": {
        "sbom": "ZERONE-2-RUNTIME-SBOM.json",
        "provenance": "ZERONE-2-RUNTIME-PROVENANCE.json",
        "signature": "ZERONE-2-RUNTIME-SIGNATURE-EVIDENCE.json",
        "signature_bundle": "ZERONE-2-RUNTIME-SIGNATURE-BUNDLE.json",
        "vulnerability_decision": "ZERONE-2-RUNTIME-VULNERABILITY-DECISION.json",
        "vulnerability_scan": "ZERONE-2-RUNTIME-VULNERABILITY-SCAN.json",
    },
    "query_gateway": {
        "sbom": "QUERY-GATEWAY-SBOM.json",
        "provenance": "QUERY-GATEWAY-PROVENANCE.json",
        "signature": "QUERY-GATEWAY-SIGNATURE-EVIDENCE.json",
        "signature_bundle": "QUERY-GATEWAY-SIGNATURE-BUNDLE.json",
        "vulnerability_decision": "QUERY-GATEWAY-VULNERABILITY-DECISION.json",
        "vulnerability_scan": "QUERY-GATEWAY-VULNERABILITY-SCAN.json",
    },
}
COMPONENT_BUILD_RECIPES = {
    "zerone_1_halt": "deploy/mainnet/build-image.sh",
    "zerone_2_runtime": "deploy/networks/zerone-2/runtime/build-image.sh",
    "query_gateway": "deploy/query-gateway/build-image.sh",
}
CANONICAL_COMPONENT_SIGNER_IDENTITY = (
    "https://github.com/cambridgetcg/zerone-core/"
    ".github/workflows/ci.yml@refs/heads/main"
)
CANONICAL_COMPONENT_CERTIFICATE_ISSUER = (
    "https://token.actions.githubusercontent.com"
)
MONITORING_RULE_DEFINITIONS = (
    (
        "stalled_height",
        "ZeroneStalledHeight",
        "critical",
        "consensus_height_no_progress",
        {"maximum_stall_seconds": 120},
        "hold_height_without_progress_past_threshold",
    ),
    (
        "missed_signing",
        "ZeroneValidatorMissedSigning",
        "critical",
        "validator_missed_blocks_above_threshold",
        {"maximum_missed_blocks": 0, "window_blocks": 100},
        "inject_validator_missed_blocks_above_threshold",
    ),
    (
        "double_sign_risk",
        "ZeroneDoubleSignRisk",
        "critical",
        "active_signer_instances_above_threshold",
        {"maximum_active_signer_instances": 1},
        "inject_signer_instance_count_above_threshold",
    ),
    (
        "app_hash_divergence",
        "ZeroneAppHashDivergence",
        "critical",
        "equal_height_app_hashes_diverge",
        {"maximum_distinct_app_hashes": 1, "minimum_independent_sources": 2},
        "inject_equal_height_mismatched_app_hash",
    ),
    (
        "peer_loss",
        "ZeronePeerLoss",
        "critical",
        "private_peer_count_below_threshold",
        {"minimum_private_peers": 1},
        "disconnect_all_private_peers",
    ),
    (
        "disk_capacity",
        "ZeroneDiskCapacity",
        "warning",
        "disk_free_percent_below_threshold",
        {"minimum_free_percent": 20},
        "inject_disk_free_percent_below_threshold",
    ),
    (
        "restart_count",
        "ZeroneRestartCount",
        "warning",
        "process_restarts_above_threshold",
        {"maximum_restarts": 0, "window_seconds": 900},
        "inject_restart_counter_above_threshold",
    ),
    (
        "stale_backup",
        "ZeroneStaleBackup",
        "critical",
        "verified_backup_age_above_threshold",
        {"maximum_verified_backup_age_seconds": 86400},
        "inject_verified_backup_age_above_threshold",
    ),
    (
        "gateway_wrong_chain",
        "ZeroneGatewayWrongChain",
        "critical",
        "gateway_chain_id_mismatch",
        {"expected_chain_id": "zerone-2"},
        "inject_gateway_chain_id_mismatch",
    ),
    (
        "gateway_stale_origin",
        "ZeroneGatewayStaleOrigin",
        "critical",
        "gateway_origin_height_lag_above_threshold",
        {"maximum_height_lag": 3},
        "inject_gateway_origin_height_lag_above_threshold",
    ),
)


def make_component_artifacts(
    output: pathlib.Path,
    release: dict[str, Any],
    tool_manifest: dict[str, Any],
    main_fingerprint: str,
) -> None:
    for component, names in COMPONENT_ARTIFACT_FILES.items():
        declaration = release["components"][component]
        image_ref = declaration["image_ref"]
        image_digest = image_ref.rsplit("@sha256:", 1)[1]
        binary_sha = declaration.get("binary_sha256")
        signature_bundle = {
            "mediaType": "application/vnd.dev.sigstore.bundle.v0.3+json",
            "verificationMaterial": {
                "certificate": base64.b64encode(f"fixture-{component}".encode()).decode()
            },
            "messageSignature": {
                "messageDigest": {
                    "algorithm": "SHA2_256",
                    "digest": base64.b64encode(bytes.fromhex(image_digest)).decode(),
                },
                "signature": base64.b64encode((component.encode() + b"s" * 64)).decode(),
            },
        }
        write_json(output, names["signature_bundle"], signature_bundle)
        scan = {
            "schema": "zerone-component-vulnerability-scan-v1",
            "component": component,
            "image_ref": image_ref,
            "scanner": {
                "name": "fixture-scanner",
                "version": "1.0.0",
                "database_updated_at": "2026-07-10T09:20:00Z",
            },
            "scanned_at": "2026-07-10T09:25:00Z",
            "counts": {"critical": 0, "high": 0, "medium": 0, "low": 0, "unknown": 0},
            "result": "COMPLETE",
        }
        write_json(output, names["vulnerability_scan"], scan)
        sbom = {
            "schema": "zerone-component-sbom-v1",
            "component": component,
            "image_ref": image_ref,
            "source_commit": release["source"]["commit"],
            "generated_at": "2026-07-10T09:15:00Z",
            "packages": [
                {
                    "name": component.replace("_", "-"),
                    "version": "fixture",
                    "purl": f"pkg:generic/zerone/{component}@fixture",
                }
            ],
            "result": "COMPLETE",
        }
        provenance = {
            "schema": "zerone-component-provenance-v1",
            "component": component,
            "subject": {
                "image_ref": image_ref,
                "image_digest": f"sha256:{image_digest}",
                "binary_sha256": binary_sha,
            },
            "source": copy.deepcopy(release["source"]),
            "build": {
                "builder_id": "https://builder.example/zerone-fixture",
                "build_recipe_path": COMPONENT_BUILD_RECIPES[component],
                "build_recipe_sha256": tool_manifest["files"][
                    COMPONENT_BUILD_RECIPES[component]
                ],
                "started_at": "2026-07-10T09:00:00Z",
                "finished_at": "2026-07-10T09:10:00Z",
                "source_materials_complete": True,
            },
            "result": "MATCH",
        }
        signature = {
            "schema": "zerone-component-signature-evidence-v1",
            "component": component,
            "image_ref": image_ref,
            "image_digest": f"sha256:{image_digest}",
            "bundle_sha256": digest((output / names["signature_bundle"]).read_bytes()),
            "signer_identity": CANONICAL_COMPONENT_SIGNER_IDENTITY,
            "certificate_issuer": CANONICAL_COMPONENT_CERTIFICATE_ISSUER,
            "signed_at": "2026-07-10T09:12:00Z",
            "verified_at": "2026-07-10T09:13:00Z",
            "transparency_log_verified": True,
            "result": "VERIFIED",
        }
        decision = {
            "schema": "zerone-component-vulnerability-decision-v1",
            "component": component,
            "image_ref": image_ref,
            "scan_report_sha256": digest(
                (output / names["vulnerability_scan"]).read_bytes()
            ),
            "scan_completed_at": scan["scanned_at"],
            "counts": copy.deepcopy(scan["counts"]),
            "policy": {"maximum_critical": 0, "maximum_high": 0},
            "decision": "ACCEPT",
            "approved_by_fingerprint": main_fingerprint,
            "created_at": "2026-07-10T09:30:00Z",
            "expires_at": "2099-07-10T09:30:00Z",
        }
        artifacts = {
            "sbom": sbom,
            "provenance": provenance,
            "signature": signature,
            "vulnerability_decision": decision,
        }
        for artifact_type, value in artifacts.items():
            write_json(output, names[artifact_type], value)
        declaration.update(
            sbom_sha256=digest((output / names["sbom"]).read_bytes()),
            provenance_sha256=digest((output / names["provenance"]).read_bytes()),
            signature_sha256=digest((output / names["signature"]).read_bytes()),
            vulnerability_decision_sha256=digest(
                (output / names["vulnerability_decision"]).read_bytes()
            ),
        )


def make_monitoring_artifacts(
    output: pathlib.Path, release: dict[str, Any]
) -> None:
    rules = {
        "schema": "zerone-production-monitoring-rules-v1",
        "chain_id": "zerone-2",
        "source_commit": release["source"]["commit"],
        "ruleset_id": "zerone-2-production",
        "evaluation_interval_seconds": 15,
        "notification_route_id": "primary-on-call",
        "rules": [
            {
                "check": check,
                "alert_name": alert_name,
                "enabled": True,
                "severity": severity,
                "expression": expression,
                "parameters": parameters,
            }
            for check, alert_name, severity, expression, parameters, _ in (
                MONITORING_RULE_DEFINITIONS
            )
        ],
    }
    write_json(output, "MONITORING-RULES.json", rules)
    rules_hash = digest((output / "MONITORING-RULES.json").read_bytes())
    tests = {
        "schema": "zerone-production-monitoring-alert-tests-v1",
        "chain_id": "zerone-2",
        "source_commit": release["source"]["commit"],
        "ruleset_id": rules["ruleset_id"],
        "rules_sha256": rules_hash,
        "started_at": "2026-07-10T09:35:00Z",
        "completed_at": "2026-07-10T09:45:00Z",
        "notification_route_id": rules["notification_route_id"],
        "tests": [
            {
                "check": check,
                "alert_name": alert_name,
                "stimulus": stimulus,
                "observed_states": ["INACTIVE", "FIRING", "RESOLVED"],
                "notification_delivery": "DELIVERED",
                "stimulus_evidence_sha256": digest(
                    f"{check}:stimulus".encode()
                ),
                "firing_evidence_sha256": digest(f"{check}:firing".encode()),
                "notification_evidence_sha256": digest(
                    f"{check}:notification".encode()
                ),
                "resolution_evidence_sha256": digest(
                    f"{check}:resolution".encode()
                ),
                "result": "PASS",
            }
            for check, alert_name, _, _, _, stimulus in MONITORING_RULE_DEFINITIONS
        ],
        "result": "PASS",
    }
    write_json(output, "MONITORING-ALERT-TESTS.json", tests)
    manifest = {
        "schema": "zerone-production-monitoring-alerts-v1",
        "chain_id": "zerone-2",
        "source_commit": release["source"]["commit"],
        "created_at": "2026-07-10T09:50:00Z",
        "rules": {
            "filename": "MONITORING-RULES.json",
            "sha256": rules_hash,
        },
        "alert_tests": {
            "filename": "MONITORING-ALERT-TESTS.json",
            "sha256": digest(
                (output / "MONITORING-ALERT-TESTS.json").read_bytes()
            ),
        },
        "result": "PASS",
    }
    write_json(output, "MONITORING-ALERTS.json", manifest)
    release["monitoring_alerts_sha256"] = digest(
        (output / "MONITORING-ALERTS.json").read_bytes()
    )


def make_ceremony_artifacts(output: pathlib.Path, release: dict[str, Any]) -> str:
    identities = release["public_identities"]
    consensus_pubkey = {
        "@type": "/cosmos.crypto.ed25519.PubKey",
        "key": identities["validator_consensus_pubkey"],
    }
    gentx = {
        "body": {
            "messages": [
                {
                    "@type": "/cosmos.staking.v1beta1.MsgCreateValidator",
                    "validator_address": identities["validator_operator_address"],
                    "pubkey": consensus_pubkey,
                    "value": {"denom": "uzrn", "amount": "11111000000"},
                }
            ],
            "memo": f"{identities['validator_node_id']}@10.0.0.1:26656",
        },
        "auth_info": {"signer_infos": [{}]},
        "signatures": [base64.b64encode(b"s" * 64).decode()],
    }
    gentx_bytes = json.dumps(
        gentx, sort_keys=True, separators=(",", ":"), ensure_ascii=False
    ).encode()
    genesis = {
        "genesis_time": release["genesis"]["time"],
        "chain_id": "zerone-2",
        "initial_height": "1",
        "app_state": {
            "bank": {
                "balances": [
                    {
                        "address": identities["validator_account_address"],
                        "coins": [{"denom": "uzrn", "amount": "11333000000"}],
                    },
                    {
                        "address": identities["operations_account_address"],
                        "coins": [{"denom": "uzrn", "amount": "2222000000"}],
                    },
                ],
                "supply": [{"denom": "uzrn", "amount": "13555000000"}],
            },
            "genutil": {"gen_txs": [gentx]},
        },
    }
    write_json(output, "genesis.json", genesis)
    genesis_sha = digest((output / "genesis.json").read_bytes())
    release["genesis"]["sha256"] = genesis_sha
    (output / "genesis.sha256").write_text(f"{genesis_sha}  genesis.json\n")
    manifest = {
        "schema": "zerone-2-network-manifest-v2",
        "mode": "real",
        "chain_id": "zerone-2",
        "genesis_time": release["genesis"]["time"],
        "genesis_sha256": genesis_sha,
        "release": {
            "source_commit": release["source"]["commit"],
            "tag": release["source"]["signed_annotated_tag"],
            "tag_signer_fingerprint": release["source"]["tag_signer_fingerprint"],
            "binary_sha256": release["components"]["zerone_2_runtime"][
                "binary_sha256"
            ],
            "binary_version": "fixture",
            "binary_goos": "linux",
            "binary_goarch": "amd64",
        },
        "trust_model": {
            "genesis_validators": 1,
            "byzantine_fault_tolerance": 0,
            "disclosure": "FIXTURE: one disclosed operator validator; BFT f=0",
        },
        "supply_uzrn": "13555000000",
        "validator": {
            "account_address": identities["validator_account_address"],
            "operator_address": identities["validator_operator_address"],
            "consensus_pubkey": consensus_pubkey,
            "node_id": identities["validator_node_id"],
            "self_bond_uzrn": "11111000000",
            "gentx_sha256": digest(gentx_bytes),
        },
        "operations": {"account_address": identities["operations_account_address"]},
        "activations": {
            "vote_extensions": "disabled",
            "pot": "not live",
            "ibc": "external-disabled; localhost-only",
            "substrate_bridge": "disabled",
            "claiming": "disabled",
        },
    }
    write_json(output, "network-manifest.json", manifest)
    human_manifest = (
        "# Zerone Genesis Manifest — zerone-2\n\n"
        "- Ceremony mode: **real**\n"
        f"- Genesis time: {release['genesis']['time']}\n"
        f"- Genesis SHA-256: {genesis_sha}\n"
        f"- Source commit: {release['source']['commit']}\n"
        f"- Release tag: {release['source']['signed_annotated_tag']}\n"
        f"- Release tag signer fingerprint: {release['source']['tag_signer_fingerprint']}\n"
        f"- Binary SHA-256: {manifest['release']['binary_sha256']}\n"
        f"- Binary version: {manifest['release']['binary_version']}\n"
        f"- Binary target: {manifest['release']['binary_goos']}/{manifest['release']['binary_goarch']}\n"
    )
    (output / "GENESIS-MANIFEST.md").write_text(human_manifest)
    release["ceremony_artifacts"] = {
        "schema": "zerone-2-public-ceremony-artifact-set-v1",
        "genesis_checksum_sha256": digest((output / "genesis.sha256").read_bytes()),
        "network_manifest_sha256": digest((output / "network-manifest.json").read_bytes()),
        "human_manifest_sha256": digest((output / "GENESIS-MANIFEST.md").read_bytes()),
    }
    return genesis_sha


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    parser.add_argument("--main", required=True)
    parser.add_argument("--transition", required=True)
    parser.add_argument("--runtime-binary-sha", required=True)
    parser.add_argument("--halt-binary-sha", required=True)
    parser.add_argument("--runtime-image", required=True)
    parser.add_argument("--halt-image", required=True)
    parser.add_argument("--query-image", required=True)
    parser.add_argument("--genesis-sha", required=True)
    parser.add_argument("--tx-raw-sha", required=True)
    parser.add_argument("--tx-hash", required=True)
    parser.add_argument("--sender", required=True)
    parser.add_argument("--edge-private-config-sha", default="2" * 64)
    parser.add_argument("--edge-public-config-sha", default="5" * 64)
    parser.add_argument("--edge-private-config-file")
    parser.add_argument("--edge-public-config-file")
    parser.add_argument("--release-binary-file", required=True)
    parser.add_argument("--halt-release-binary-file")
    parser.add_argument("--runtime-release-binary-file")
    parser.add_argument("--signed-tx-file", required=True)
    args = parser.parse_args()

    output = pathlib.Path(args.output)
    output.mkdir(mode=0o700, parents=True, exist_ok=False)
    root = pathlib.Path(__file__).resolve().parents[2]
    templates = root / "deploy" / "networks" / "zerone-2"
    final_template_path = (
        root
        / "deploy"
        / "networks"
        / "zerone-1"
        / "frozen"
        / "FINAL-CHECKPOINT.example.json"
    )
    shared_binary = pathlib.Path(args.release_binary_file)
    halt_binary_bytes = pathlib.Path(
        args.halt_release_binary_file or shared_binary
    ).read_bytes()
    runtime_binary_bytes = pathlib.Path(
        args.runtime_release_binary_file or shared_binary
    ).read_bytes()
    if digest(halt_binary_bytes) != args.halt_binary_sha:
        raise SystemExit("fixture halt binary hash differs from the signed component hash")
    if digest(runtime_binary_bytes) != args.runtime_binary_sha:
        raise SystemExit("fixture runtime binary hash differs from the signed component hash")
    signed_tx_bytes = pathlib.Path(args.signed_tx_file).read_bytes()
    for name, binary_bytes in (
        ("zeroned-zerone-1-release", halt_binary_bytes),
        ("zeroned-zerone-2-release", runtime_binary_bytes),
    ):
        path = output / name
        path.write_bytes(binary_bytes)
        path.chmod(0o700)
    for name in ("CUTOVER-SIGNED-TX.json", "OPEN-BETA-SIGNED-TX.json"):
        (output / name).write_bytes(signed_tx_bytes)
    registration_tx_raw = {
        "operator_onboarding": b"fixture operator-onboarding TxRaw bytes",
        "custom_validator_registration": b"fixture custom-validator TxRaw bytes",
    }
    registration_tx_files = {
        "operator_onboarding": "ZERONE-2-ONBOARD-SIGNED-TX.json",
        "custom_validator_registration": "ZERONE-2-CUSTOM-VALIDATOR-SIGNED-TX.json",
    }
    for registration, raw_bytes in registration_tx_raw.items():
        write_json(
            output,
            registration_tx_files[registration],
            {"encoded": base64.b64encode(raw_bytes).decode()},
        )

    raw = {"PUBLIC-NOTICE.md": b"# Fixture public notice\n"}
    raw["PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json"] = canonical_bytes(
        {
            "schema": "zerone-public-notice-publication-evidence-v1",
            "notice_sha256": digest(raw["PUBLIC-NOTICE.md"]),
            "published_at": "2026-07-10T12:30:00Z",
            "publication_capture_sha256": "c" * 64,
            "public_url": "https://status.example/zerone-relaunch",
        }
    )
    tool_paths = (
        "deploy/verify-authority-chain.py",
        "deploy/frozen_evidence.py",
        "deploy/validate-fly-phase-config.py",
        "deploy/fly-deploy-pinned.sh",
        "deploy/fly-deploy-authorized.sh",
        "deploy/mainnet/render-archive-configs.sh",
        "deploy/mainnet/fly-deploy-archive-authorized.sh",
        "deploy/mainnet/fly-cutover-authorized.sh",
        "scripts/zerone-phase-tx-broadcast.sh",
        "scripts/zerone-2-bootstrap-tx-broadcast.sh",
        "scripts/zerone-canonical-json.sh",
        "scripts/zerone-2-ceremony.sh",
        "deploy/mainnet/build-image.sh",
        "deploy/networks/zerone-2/runtime/build-image.sh",
        "deploy/query-gateway/build-image.sh",
        "tools/zerone2-artifact-audit/main.go",
    )
    tool_manifest = {
        "schema": "zerone-operator-tool-manifest-v1",
        "source_commit": "1" * 40,
        "signed_tag": "zerone-2-fixture",
        "files": {
            path: digest((root / path).read_bytes()) for path in tool_paths
        },
    }
    raw["OPERATOR-TOOL-MANIFEST.json"] = canonical_bytes(tool_manifest)
    for name, data in raw.items():
        (output / name).write_bytes(data)

    signature_names = (
        "RELEASE-PACKET.json.sig",
        "DARK-START-DECISION.json.sig",
        "DARK-START-INITIATION-EVIDENCE.json.sig",
        "DARK-REGISTRATION-EVIDENCE.json.sig",
        "CUTOVER-DECISION.json.sig",
        "CUTOVER-INITIATION-EVIDENCE.json.sig",
        "ARCHIVE-ADOPTION-AUTHORITY.json.sig",
        "FINAL-CHECKPOINT.json.sig",
        "OPEN-BETA-DECISION.json.sig",
        "OPEN-BETA-INITIATION-EVIDENCE.json.sig",
    )
    for name in signature_names:
        (output / name).write_text(f"fixture signature {name}\n")

    release = fill_placeholders(
        read_json(templates / "RELEASE-PACKET.example.json"),
        args.main,
        args.transition,
    )
    release["created_at"] = "2026-07-10T10:00:00Z"
    release["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.main,
        "detached_signature_filename": "RELEASE-PACKET.json.sig",
    }
    release["source"] = {
        "commit": "1" * 40,
        "signed_annotated_tag": "zerone-2-fixture",
        "tag_signer_fingerprint": args.main,
    }
    release["operator_tool_manifest_sha256"] = digest(
        raw["OPERATOR-TOOL-MANIFEST.json"]
    )
    release["genesis"]["time"] = "2026-07-10T00:00:00Z"
    release["predecessor"].update(
        trusted_rpc_node_id="a" * 40,
        trusted_observer_node_id="b" * 40,
        trusted_block={
            "height": "42",
            "block_id_hash": "C" * 64,
            "app_hash": "D" * 64,
        },
    )
    release["components"]["zerone_1_halt"].update(
        binary_sha256=args.halt_binary_sha, image_ref=args.halt_image
    )
    release["components"]["zerone_2_runtime"].update(
        binary_sha256=args.runtime_binary_sha, image_ref=args.runtime_image
    )
    release["components"]["query_gateway"]["image_ref"] = args.query_image
    release["public_identities"]["transition_attestation"].update(
        algorithm="openpgp", authorized_signer_fingerprint=args.transition
    )
    release["public_identities"].update(
        validator_account_address="zerone1validatorfixture",
        validator_operator_address="zeronevaloper1fixture",
        operations_account_address="zerone1operationsfixture",
        validator_node_id="d" * 40,
        validator_consensus_pubkey=base64.b64encode(b"p" * 32).decode(),
    )
    release["public_identities"]["edge_node_id"] = "e" * 40
    renderer_path = root / "deploy" / "mainnet" / "render-archive-configs.sh"
    candidate_template_path = (
        root / "deploy" / "mainnet" / "fly.archive-candidate.example.toml"
    )
    archive_template_path = root / "deploy" / "mainnet" / "fly.archive.example.toml"
    release["archive_render_contract"]["renderer_sha256"] = digest(
        renderer_path.read_bytes()
    )
    release["phase_dependent_config_template_sha256"].update(
        zerone_1_archive_candidate=digest(candidate_template_path.read_bytes()),
        zerone_1_archive=digest(archive_template_path.read_bytes()),
    )
    def rendered_config(path: pathlib.Path, replacements: dict[str, str]) -> bytes:
        text = path.read_text()
        for old, new in replacements.items():
            text = text.replace(old, new)
        text = re.sub(r"replace-[a-z0-9-]+", "fixture", text)
        return text.encode()

    runtime_dir = root / "deploy" / "networks" / "zerone-2" / "runtime"
    query_dir = root / "deploy" / "query-gateway"
    edge_private_config = (
        pathlib.Path(args.edge_private_config_file).read_bytes()
        if args.edge_private_config_file
        else rendered_config(
            runtime_dir / "fly.edge.example.toml",
            {
                "replace-edge-app": "zerone-2-edge",
                "replace-with-pinned-image-digest": args.runtime_image,
            },
        )
    )
    edge_public_config = (
        pathlib.Path(args.edge_public_config_file).read_bytes()
        if args.edge_public_config_file
        else rendered_config(
            runtime_dir / "fly.edge.public.example.toml",
            {
                "replace-edge-app": "zerone-2-edge",
                "replace-with-pinned-image-digest": args.runtime_image,
                "replace-public-edge-address": "p2p.example",
            },
        )
    )
    gateway_public_config = rendered_config(
        query_dir / "fly.zerone-2.public.example.toml",
        {
            "replace-zerone-2-query-gateway-app": "zerone-2-gateway",
            "replace-with-pinned-query-gateway-image-digest": args.query_image,
            "replace-zerone-2-edge-app.internal": "zerone-2-edge.internal",
        },
    )
    archive_gateway_config = rendered_config(
        query_dir / "fly.zerone-1-archive.public.example.toml",
        {
            "replace-zerone-1-archive-gateway-app": "zerone-1-archive-gateway",
            "replace-with-pinned-query-gateway-image-digest": args.query_image,
            "replace-zerone-1-archive-app.internal": "zerone-1-archive.internal",
        },
    )
    public_configs = {
        "fly.edge.public.toml": edge_public_config,
        "fly.zerone-2-gateway.public.toml": gateway_public_config,
        "fly.zerone-1-archive-gateway.public.toml": archive_gateway_config,
    }
    for name, data in public_configs.items():
        (output / name).write_bytes(data)
    release["deployment_configs"] = {
        "zerone_2_validator": mapping(
            "zerone-2-validator", "validator", "zerone_2_runtime", args.runtime_image, "1" * 64
        ),
        "zerone_2_edge_private": mapping(
            "zerone-2-edge",
            "edge",
            "zerone_2_runtime",
            args.runtime_image,
            digest(edge_private_config),
        ),
        "zerone_2_edge_query_soak": mapping(
            "zerone-2-edge", "edge", "zerone_2_runtime", args.runtime_image, "3" * 64
        ),
        "zerone_2_gateway_private": mapping(
            "zerone-2-gateway", "zerone-2-query", "query_gateway", args.query_image, "4" * 64
        ),
        "zerone_2_edge_public": mapping(
            "zerone-2-edge",
            "edge",
            "zerone_2_runtime",
            args.runtime_image,
            digest(edge_public_config),
        ),
        "zerone_2_gateway_public": mapping(
            "zerone-2-gateway",
            "zerone-2-query",
            "query_gateway",
            args.query_image,
            digest(gateway_public_config),
        ),
        "zerone_1_archive_gateway": mapping(
            "zerone-1-archive-gateway",
            "zerone-1-archive-query",
            "query_gateway",
            args.query_image,
            digest(archive_gateway_config),
        ),
    }
    make_monitoring_artifacts(output, release)
    make_component_artifacts(output, release, tool_manifest, args.main)
    args.genesis_sha = make_ceremony_artifacts(output, release)
    write_json(output, "RELEASE-PACKET.json", release)
    release_pair = pair(output, "RELEASE-PACKET.json", "RELEASE-PACKET.json.sig")

    dark = fill_placeholders(
        read_json(templates / "DARK-START-DECISION.example.json"),
        args.main,
        args.transition,
    )
    dark.update(
        decision="GO",
        created_at="2026-07-10T10:10:00Z",
        release_packet_sha256=release_pair["sha256"],
        release_packet_detached_signature_sha256=release_pair[
            "detached_signature_sha256"
        ],
    )
    dark["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.main,
        "detached_signature_filename": "DARK-START-DECISION.json.sig",
    }
    dark["authorization_semantics"].update(
        initiation_deadline="2026-07-10T11:00:00Z",
        registration_commit_deadline="2099-07-10T11:30:00Z",
        registration_broadcast_not_after="2099-07-10T11:25:00Z",
        minimum_registration_inclusion_margin_seconds=300,
    )
    operator_address = release["public_identities"]["operations_account_address"]
    registration_public_key = "7" * 64
    registration_did = "did:zrn:" + registration_public_key
    consensus_key_hex = base64.b64decode(
        release["public_identities"]["validator_consensus_pubkey"]
    ).hex()
    onboarding_raw = registration_tx_raw["operator_onboarding"]
    validator_raw = registration_tx_raw["custom_validator_registration"]
    dark["private_bootstrap_transactions"] = {
        "broadcast_order": ["operator_onboarding", "custom_validator_registration"],
        "operator_onboarding": {
            "filename": registration_tx_files["operator_onboarding"],
            "chain_id": "zerone-2",
            "signed_tx_bytes_sha256": digest(onboarding_raw),
            "expected_transaction_hash": digest(onboarding_raw).upper(),
            "sender": operator_address,
            "message_type": "/zerone.auth.v1.MsgRegisterAccount",
            "did": registration_did,
            "public_key": registration_public_key,
            "account_type": "human",
            "operational_key_hash": "",
            "metadata": "",
            "fee": "200000uzrn",
            "gas_limit": "200000",
            "signer_sequence": "0",
            "timeout_height": "500",
            "memo": "zerone-2-private-bootstrap:operator-onboarding",
            "must_not_reference_dark_start_payload": True,
        },
        "custom_validator_registration": {
            "filename": registration_tx_files["custom_validator_registration"],
            "chain_id": "zerone-2",
            "signed_tx_bytes_sha256": digest(validator_raw),
            "expected_transaction_hash": digest(validator_raw).upper(),
            "operator": operator_address,
            "message_type": "/zerone.staking.v1.MsgRegisterValidator",
            "consensus_pubkey": consensus_key_hex,
            "did": registration_did,
            "moniker": "zerone-2-custodian",
            "self_delegation": "111000000",
            "commission_bps": "500",
            "website": "",
            "details": "One publicly disclosed custodial validator",
            "fee": "200000uzrn",
            "gas_limit": "200000",
            "signer_sequence": "1",
            "timeout_height": "600",
            "memo": "zerone-2-private-bootstrap:custom-validator-registration",
            "must_not_reference_dark_start_payload": True,
        },
    }
    dark["deployment_configs"] = {
        key: copy.deepcopy(release["deployment_configs"][key])
        for key in (
            "zerone_2_validator",
            "zerone_2_edge_private",
            "zerone_2_edge_query_soak",
            "zerone_2_gateway_private",
        )
    }
    write_json(output, "DARK-START-DECISION.json", dark)
    dark_pair = pair(
        output, "DARK-START-DECISION.json", "DARK-START-DECISION.json.sig"
    )

    dark_init = fill_placeholders(
        read_json(templates / "DARK-START-INITIATION-EVIDENCE.example.json"),
        args.main,
        args.transition,
    )
    dark_init.update(
        attestation_result="MATCH",
        created_at="2026-07-10T10:21:00Z",
        dark_start_decision=dark_pair,
        initiation_deadline="2026-07-10T11:00:00Z",
        deadline_satisfied=True,
    )
    dark_init["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.main,
        "detached_signature_filename": "DARK-START-INITIATION-EVIDENCE.json.sig",
    }
    dark_init["first_committed_block"] = {
        "chain_id": "zerone-2",
        "height": "1",
        "block_id_hash": "A" * 64,
        "app_hash": "B" * 64,
        "committed_block_time": "2026-07-10T10:20:00.123456789Z",
        "validator_evidence_sha256": "8" * 64,
        "independent_edge_evidence_sha256": "9" * 64,
    }
    write_json(output, "DARK-START-INITIATION-EVIDENCE.json", dark_init)
    dark_init_pair = pair(
        output,
        "DARK-START-INITIATION-EVIDENCE.json",
        "DARK-START-INITIATION-EVIDENCE.json.sig",
    )

    dark_registration = fill_placeholders(
        read_json(templates / "DARK-REGISTRATION-EVIDENCE.example.json"),
        args.main,
        args.transition,
    )
    dark_registration.update(
        attestation_result="MATCH",
        created_at="2026-07-10T10:35:00Z",
        dark_start_decision=dark_pair,
        dark_start_initiation_evidence=dark_init_pair,
        registration_commit_deadline="2099-07-10T11:30:00Z",
        deadline_satisfied=True,
    )
    dark_registration["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.main,
        "detached_signature_filename": "DARK-REGISTRATION-EVIDENCE.json.sig",
    }
    for key, height, block_time in (
        ("operator_onboarding", "10", "2026-07-10T10:30:00.111111111Z"),
        ("custom_validator_registration", "11", "2026-07-10T10:31:00.222222222Z"),
    ):
        transaction = dark["private_bootstrap_transactions"][key]
        dark_registration[key] = {
            "signed_tx_bytes_sha256": transaction["signed_tx_bytes_sha256"],
            "expected_transaction_hash": transaction["expected_transaction_hash"],
            "committed_transaction_hash": transaction["expected_transaction_hash"],
            "deliver_code": 0,
            "committed_height": height,
            "committed_block_time": block_time,
            "raw_transaction_query_evidence_sha256": "8" * 64,
            "independent_edge_evidence_sha256": "9" * 64,
        }
    dark_registration["post_registration_state"] = {
        "chain_id": "zerone-2",
        "operator": operator_address,
        "did": registration_did,
        "identity_public_key": registration_public_key,
        "validator_consensus_pubkey": consensus_key_hex,
        "custom_validator_active": True,
        "custom_validator_self_delegation_uzrn": "111000000",
        "custom_staking_module_balance_uzrn": "111000000",
        "total_supply_uzrn": "13555000000",
        "sdk_validator_count": 1,
        "account_query_evidence_sha256": "1" * 64,
        "custom_validator_query_evidence_sha256": "2" * 64,
        "module_balance_query_evidence_sha256": "3" * 64,
        "supply_query_evidence_sha256": "4" * 64,
    }
    write_json(output, "DARK-REGISTRATION-EVIDENCE.json", dark_registration)
    dark_registration_pair = pair(
        output,
        "DARK-REGISTRATION-EVIDENCE.json",
        "DARK-REGISTRATION-EVIDENCE.json.sig",
    )

    private_soak = {
        "schema": "zerone-2-private-soak-evidence-v1",
        "chain_id": "zerone-2",
        "release_packet": release_pair,
        "dark_start_decision": dark_pair,
        "dark_start_initiation_evidence": dark_init_pair,
        "dark_registration_evidence": dark_registration_pair,
        "genesis_sha256": args.genesis_sha,
        "start": {
            "height": "1",
            "block_time": "2026-07-10T10:20:00.123456789Z",
            "app_hash": "B" * 64,
        },
        "end": {
            "height": "1001",
            "block_time": "2026-07-10T12:20:00.123456789Z",
            "app_hash": "F" * 64,
        },
        "observed_blocks": 1000,
        "observed_wall_time_minutes": 120,
        "validator": {
            "node_id": release["public_identities"]["validator_node_id"],
            "consensus_pubkey": release["public_identities"][
                "validator_consensus_pubkey"
            ],
            "app_hash": "F" * 64,
        },
        "edge": {"app_hash": "F" * 64, "evidence_sha256": "1" * 64},
        "supply_uzrn": "13555000000",
        "sdk_validator_count": 1,
        "protocol_dark_audit_sha256": "2" * 64,
        "query_gateway_smoke_evidence_sha256": "3" * 64,
        "restart_export_import_rehearsal_sha256": "4" * 64,
        "result": "MATCH",
    }
    raw["PRIVATE-SOAK-EVIDENCE.json"] = canonical_bytes(private_soak)
    halt_rehearsal = {
        "schema": "zerone-1-halt-rehearsal-evidence-v1",
        "chain_id": "zerone-1",
        "release_packet": release_pair,
        "source_commit": release["source"]["commit"],
        "binary_sha256": args.halt_binary_sha,
        "image_ref": args.halt_image,
        "tested_checkpoint_plan": {
            "checkpoint_state_height": "3",
            "final_committed_anchor_height": "4",
            "halt_trigger_height": "5",
        },
        "independent_observer_matched": True,
        "anchor_transaction_count": 0,
        "halt_trigger_transaction_count": 0,
        "sanitized_archive_allowlist_matched": True,
        "raw_evidence_manifest_sha256": "5" * 64,
        "completed_at": "2026-07-10T12:25:00Z",
        "result": "MATCH",
    }
    raw["HALT-REHEARSAL-EVIDENCE.json"] = canonical_bytes(halt_rehearsal)
    for name in ("PRIVATE-SOAK-EVIDENCE.json", "HALT-REHEARSAL-EVIDENCE.json"):
        (output / name).write_bytes(raw[name])

    cutover = fill_placeholders(
        read_json(templates / "CUTOVER-DECISION.example.json"),
        args.main,
        args.transition,
    )
    cutover.update(
        decision="GO",
        created_at="2026-07-10T13:00:00Z",
        release_packet_sha256=release_pair["sha256"],
        release_packet_detached_signature_sha256=release_pair[
            "detached_signature_sha256"
        ],
        dark_start_decision_sha256=dark_pair["sha256"],
        dark_start_detached_signature_sha256=dark_pair[
            "detached_signature_sha256"
        ],
        dark_start_initiation_evidence_sha256=dark_init_pair["sha256"],
        dark_start_initiation_evidence_detached_signature_sha256=dark_init_pair[
            "detached_signature_sha256"
        ],
        dark_registration_evidence_sha256=dark_registration_pair["sha256"],
        dark_registration_evidence_detached_signature_sha256=dark_registration_pair[
            "detached_signature_sha256"
        ],
        private_soak_evidence_sha256=digest(raw["PRIVATE-SOAK-EVIDENCE.json"]),
        halt_rehearsal_evidence_sha256=digest(raw["HALT-REHEARSAL-EVIDENCE.json"]),
        public_notice_sha256=digest(raw["PUBLIC-NOTICE.md"]),
        public_notice_publication_evidence_sha256=digest(
            raw["PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json"]
        ),
        public_notice_published_at="2026-07-10T12:30:00Z",
    )
    cutover["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.main,
        "detached_signature_filename": "CUTOVER-DECISION.json.sig",
    }
    cutover["authorization_semantics"].update(
        initiation_deadline="2099-07-12T12:00:00Z",
        broadcast_not_after="2099-07-12T11:55:00Z",
        minimum_inclusion_margin_seconds=300,
        minimum_halt_lead_blocks=100,
    )
    cutover["checkpoint_plan"] = {
        "checkpoint_state_height": "1000",
        "final_committed_anchor_height": "1001",
        "halt_trigger_height": "1002",
    }
    cutover_memo = (
        f"successor_chain_id=zerone-2;successor_genesis_sha256={args.genesis_sha};"
        "checkpoint_state_height=1000;final_committed_height=1001;halt_trigger_height=1002"
    )
    cutover["successor_commitment_transaction"] = {
        "chain_id": "zerone-1",
        "signed_tx_bytes_sha256": args.tx_raw_sha,
        "expected_transaction_hash": args.tx_hash,
        "sender": args.sender,
        "message_type": "/cosmos.bank.v1beta1.MsgSend",
        "recipient_equals_sender": True,
        "amount": "1uzrn",
        "fee": "200000uzrn",
        "gas_limit": "200000",
        "timeout_height": "900",
        "memo": cutover_memo,
        "must_not_reference_cutover_payload": True,
    }
    halt_signer_config = (
        (root / "deploy" / "mainnet" / "fly.halt-signer.example.toml")
        .read_text()
        .replace("REPLACE_WITH_PINNED_ZERONE_1_HALT_IMAGE_DIGEST", args.halt_image)
        .replace("REPLACE_WITH_F", "1000")
        .replace("REPLACE_WITH_A", "1001")
        .replace("REPLACE_WITH_H", "1002")
    ).encode()
    observer_config = (
        (root / "deploy" / "mainnet" / "fly.observer.example.toml")
        .read_text()
        .replace("REPLACE_WITH_PINNED_ZERONE_1_HALT_IMAGE_DIGEST", args.halt_image)
        .replace("REPLACE_WITH_F", "1000")
        .replace("REPLACE_WITH_A", "1001")
        .replace("REPLACE_WITH_H", "1002")
    ).encode()
    (output / "fly.halt-signer.toml").write_bytes(halt_signer_config)
    (output / "fly.observer.toml").write_bytes(observer_config)
    cutover["deployment_configs"] = {
        "zerone_1_halt_signer": mapping(
            "zerone-1",
            "signer",
            "zerone_1_halt",
            args.halt_image,
            digest(halt_signer_config),
        ),
        "zerone_1_observer": mapping(
            "zerone-1-observer",
            "observer",
            "zerone_1_halt",
            args.halt_image,
            digest(observer_config),
        ),
    }
    cutover["deterministic_private_continuation"][
        "authorized_transition_signer_fingerprint"
    ] = args.transition
    cutover_render = cutover["deterministic_private_continuation"]["render_contract"]
    cutover_render.update(
        renderer_sha256=release["archive_render_contract"]["renderer_sha256"],
        archive_candidate_template_sha256=release[
            "phase_dependent_config_template_sha256"
        ]["zerone_1_archive_candidate"],
        archive_template_sha256=release["phase_dependent_config_template_sha256"][
            "zerone_1_archive"
        ],
        static_constraints=copy.deepcopy(
            release["archive_render_contract"]["static_constraints"]
        ),
    )
    write_json(output, "CUTOVER-DECISION.json", cutover)
    cutover_pair = pair(
        output, "CUTOVER-DECISION.json", "CUTOVER-DECISION.json.sig"
    )

    cutover_init = fill_placeholders(
        read_json(templates / "CUTOVER-INITIATION-EVIDENCE.example.json"),
        args.main,
        args.transition,
    )
    cutover_init.update(
        attestation_result="MATCH",
        created_at="2026-07-10T14:01:00Z",
        cutover_decision=cutover_pair,
        initiation_deadline="2099-07-12T12:00:00Z",
        deadline_satisfied=True,
    )
    cutover_init["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.main,
        "detached_signature_filename": "CUTOVER-INITIATION-EVIDENCE.json.sig",
    }
    cutover_init["public_notice"] = {
        "sha256": digest(raw["PUBLIC-NOTICE.md"]),
        "publication_evidence_sha256": "c" * 64,
    }
    cutover_init["successor_commitment_transaction"] = {
        "signed_tx_bytes_sha256": args.tx_raw_sha,
        "expected_transaction_hash": args.tx_hash,
        "committed_transaction_hash": args.tx_hash,
        "deliver_code": 0,
        "committed_height": "90",
        "committed_block_time": "2026-07-10T14:00:00.333333333Z",
        "raw_transaction_query_evidence_sha256": "d" * 64,
        "independent_observer_evidence_sha256": "e" * 64,
    }
    write_json(output, "CUTOVER-INITIATION-EVIDENCE.json", cutover_init)
    cutover_init_pair = pair(
        output,
        "CUTOVER-INITIATION-EVIDENCE.json",
        "CUTOVER-INITIATION-EVIDENCE.json.sig",
    )

    checkpoint_app_hash = "B" * 64
    anchor_block_hash = "A" * 64
    halt_trigger_block_hash = "D" * 64
    post_anchor_app_hash = "E" * 64
    anchor_block_time = "2026-07-10T14:10:00.111111111Z"
    halt_trigger_block_time = "2026-07-10T14:10:01.222222222Z"
    signer_node_id = release["predecessor"]["trusted_rpc_node_id"]
    signer_validator_pubkey = base64.b64encode(b"s" * 32).decode()
    source_node_id = release["predecessor"]["trusted_observer_node_id"]
    source_validator_pubkey = base64.b64encode(b"o" * 32).decode()
    candidate_node_id = "3" * 40
    candidate_validator_pubkey = base64.b64encode(b"c" * 32).decode()
    signer_validator_address = hashlib.sha256(b"s" * 32).digest()[:20].hex().upper()
    fixture_commit_signature = base64.b64encode(b"z" * 64).decode()
    trusted_height = release["predecessor"]["trusted_block"]["height"]
    trusted_block_hash = release["predecessor"]["trusted_block"]["block_id_hash"]
    trusted_app_hash = release["predecessor"]["trusted_block"]["app_hash"]
    trusted_block_time = "2026-07-10T13:00:00.777777777Z"
    rpc_genesis_document = {
        "chain_id": "zerone-1",
        "initial_height": "1",
        "app_hash": "",
        "validators": [],
    }
    rpc_genesis_canonical = json.dumps(
        rpc_genesis_document, sort_keys=True, separators=(",", ":")
    ).encode()

    inventory = {
        "schema": "zerone-relaunch-snapshot-v3",
        "source": {
            "chain_id": "zerone-1",
            "checkpoint_state_height": 1000,
            "checkpoint_app_hash": checkpoint_app_hash,
            "final_committed_block_height": 1001,
            "final_committed_block_hash": anchor_block_hash,
            "final_committed_block_time": anchor_block_time,
            "final_committed_block_txs": 0,
            "final_committed_block_canonical": True,
            "final_committed_block_has_results": True,
            "halt_trigger_height": 1002,
            "rpc_blockstore_height": 1002,
            "staged_halt_trigger_block_hash": halt_trigger_block_hash,
            "staged_halt_trigger_block_time": halt_trigger_block_time,
            "staged_halt_trigger_block_txs": 0,
            "staged_halt_trigger_previous_block_hash": anchor_block_hash,
            "staged_halt_trigger_header_app_hash": post_anchor_app_hash,
            "staged_halt_trigger_commit_canonical": False,
            "staged_halt_trigger_has_block_results": False,
            "abci_last_applied_height": 1001,
            "excluded_post_anchor_app_hash": post_anchor_app_hash,
            "rpc_genesis_canonical_sha256": digest(rpc_genesis_canonical),
            "declared_genesis_file_sha256": release["predecessor"][
                "genesis_file_sha256"
            ],
            "rest_trust_model": (
                "trusted height-pinned REST responses; no Merkle proof binds "
                "inventory to checkpoint_app_hash"
            ),
            "rpc": "http://zerone-1.internal:26657",
            "rest": "http://zerone-1.internal:1317",
        },
        "denom": "uzrn",
        "supply_uzrn": "300",
        "owners": [
            {
                "address": "zerone1owner1",
                "account_type": "base_account",
                "amount_uzrn": "100",
            },
            {
                "address": "zerone1owner2",
                "account_type": "base_account",
                "amount_uzrn": "200",
            },
        ],
        "bonded_validators": [
            {
                "operator_address": "zeronevaloper1fixture",
                "consensus_pubkey": {
                    "@type": "/cosmos.crypto.ed25519.PubKey",
                    "key": signer_validator_pubkey,
                },
                "jailed": False,
                "status": "BOND_STATUS_BONDED",
                "tokens": "100",
            }
        ],
    }

    trusted_header = {
        "chain_id": "zerone-1",
        "height": trusted_height,
        "app_hash": trusted_app_hash,
        "time": trusted_block_time,
    }
    anchor_header = {
        "chain_id": "zerone-1",
        "height": "1001",
        "app_hash": checkpoint_app_hash,
        "time": anchor_block_time,
    }
    halt_header = {
        "chain_id": "zerone-1",
        "height": "1002",
        "app_hash": post_anchor_app_hash,
        "time": halt_trigger_block_time,
        "last_block_id": {"hash": anchor_block_hash},
    }
    validator_page = {
        "validators": [
            {
                "address": signer_validator_address,
                "pub_key": {
                    "type": "tendermint/PubKeyEd25519",
                    "value": signer_validator_pubkey,
                },
                "voting_power": "100",
                "proposer_priority": "0",
            }
        ],
        "count": "1",
        "total": "1",
    }
    common_rpc_payloads = {
        "genesis_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {"genesis": rpc_genesis_document},
            }
        ),
        "trusted_block_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "block_id": {"hash": trusted_block_hash},
                    "block": {
                        "header": copy.deepcopy(trusted_header),
                        "data": {"txs": ["fixture-trusted-transaction"]},
                    },
                },
            }
        ),
        "trusted_commit_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "canonical": True,
                    "signed_header": {
                        "header": copy.deepcopy(trusted_header),
                        "commit": {
                            "height": trusted_height,
                            "block_id": {"hash": trusted_block_hash},
                            "signatures": [
                                {
                                    "block_id_flag": 2,
                                    "validator_address": signer_validator_address,
                                    "timestamp": trusted_block_time,
                                    "signature": fixture_commit_signature,
                                }
                            ],
                        },
                    },
                },
            }
        ),
        "trusted_validators_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    **copy.deepcopy(validator_page),
                    "block_height": trusted_height,
                },
            }
        ),
        "block_a_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "block_id": {"hash": anchor_block_hash},
                    "block": {
                        "header": copy.deepcopy(anchor_header),
                        "data": {"txs": []},
                    },
                },
            }
        ),
        "commit_a_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "canonical": True,
                    "signed_header": {
                        "header": copy.deepcopy(anchor_header),
                        "commit": {
                            "height": "1001",
                            "block_id": {"hash": anchor_block_hash},
                            "signatures": [
                                {
                                    "block_id_flag": 2,
                                    "validator_address": signer_validator_address,
                                    "timestamp": anchor_block_time,
                                    "signature": fixture_commit_signature,
                                }
                            ],
                        },
                    },
                },
            }
        ),
        "validators_a_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    **copy.deepcopy(validator_page),
                    "block_height": "1001",
                },
            }
        ),
        "block_results_a_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "height": "1001",
                    "txs_results": None,
                    "finalize_block_events": [],
                    "validator_updates": [],
                    "consensus_param_updates": None,
                    "app_hash": post_anchor_app_hash,
                },
            }
        ),
        "block_h_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "block_id": {"hash": halt_trigger_block_hash},
                    "block": {
                        "header": copy.deepcopy(halt_header),
                        "data": {"txs": None},
                    },
                },
            }
        ),
        "commit_h_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "canonical": False,
                    "signed_header": {
                        "header": copy.deepcopy(halt_header),
                        "commit": {
                            "height": "1002",
                            "block_id": {"hash": halt_trigger_block_hash},
                            "signatures": [
                                {
                                    "block_id_flag": 2,
                                    "validator_address": signer_validator_address,
                                    "timestamp": halt_trigger_block_time,
                                    "signature": fixture_commit_signature,
                                }
                            ],
                        },
                    },
                },
            }
        ),
        "validators_h_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    **copy.deepcopy(validator_page),
                    "block_height": "1002",
                },
            }
        ),
        "abci_info_json": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "response": {
                        "last_block_height": "1001",
                        "last_block_app_hash": post_anchor_app_hash,
                    }
                },
            }
        ),
        "block_results_h_missing_response": canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "error": {
                    "code": -32603,
                    "message": "Internal error",
                    "data": "could not find results for height #1002",
                },
            }
        ),
    }
    rpc_suffixes = {
        "status_json": "STATUS.json.raw",
        "genesis_json": "GENESIS.json.raw",
        "trusted_block_json": "TRUSTED-BLOCK.json.raw",
        "trusted_commit_json": "TRUSTED-COMMIT.json.raw",
        "trusted_validators_json": "TRUSTED-VALIDATORS.json.raw",
        "block_a_json": "BLOCK-A.json.raw",
        "commit_a_json": "COMMIT-A.json.raw",
        "validators_a_json": "VALIDATORS-A.json.raw",
        "block_results_a_json": "BLOCK-RESULTS-A.json.raw",
        "block_h_json": "BLOCK-H.json.raw",
        "commit_h_json": "COMMIT-H.json.raw",
        "validators_h_json": "VALIDATORS-H.json.raw",
        "abci_info_json": "ABCI-INFO.json.raw",
        "block_results_h_missing_response": "BLOCK-RESULTS-H-MISSING.json.raw",
    }

    rpc_raw: dict[str, bytes] = {}
    for prefix, node_id, validator_pubkey in (
        ("SIGNER", signer_node_id, signer_validator_pubkey),
        ("OBSERVER", source_node_id, source_validator_pubkey),
    ):
        rpc_raw[f"{prefix}-RPC-{rpc_suffixes['status_json']}"] = canonical_bytes(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "node_info": {"network": "zerone-1", "id": node_id},
                    "sync_info": {
                        "latest_block_height": "1002",
                        "latest_block_hash": halt_trigger_block_hash,
                        "latest_app_hash": post_anchor_app_hash,
                        "catching_up": False,
                    },
                    "validator_info": {
                        "pub_key": {"value": validator_pubkey},
                        "voting_power": "100" if prefix == "SIGNER" else "0",
                    },
                },
            }
        )
        for key, payload in common_rpc_payloads.items():
            rpc_raw[f"{prefix}-RPC-{rpc_suffixes[key]}"] = payload

    def terminal_manifest(
        role: str, prefix: str, node_id: str, validator_pubkey: str
    ) -> dict[str, Any]:
        return {
            "schema": "zerone-1-terminal-evidence-manifest-v2",
            "role": role,
            "chain_id": "zerone-1",
            "genesis_sha256": release["predecessor"]["genesis_file_sha256"],
            "checkpoint_state_height": "1000",
            "checkpoint_app_hash": checkpoint_app_hash,
            "final_committed_height": "1001",
            "final_committed_block_time": anchor_block_time,
            "halt_trigger_height": "1002",
            "anchor_block_hash": anchor_block_hash,
            "halt_trigger_block_hash": halt_trigger_block_hash,
            "halt_trigger_block_time": halt_trigger_block_time,
            "post_anchor_app_hash": post_anchor_app_hash,
            "abci_last_applied_height": "1001",
            "node_id": node_id,
            "validator_pubkey": validator_pubkey,
            "payload_sha256": {
                key: digest(rpc_raw[f"{prefix}-RPC-{suffix}"])
                for key, suffix in rpc_suffixes.items()
            },
            "result": "MATCH",
        }

    signer_evidence = terminal_manifest(
        "official-signer", "SIGNER", signer_node_id, signer_validator_pubkey
    )
    observer_evidence = terminal_manifest(
        "independent-observer", "OBSERVER", source_node_id, source_validator_pubkey
    )
    source_marker_values = {
        "runtime_version": "2",
        "role": "observer",
        "chain_id": "zerone-1",
        "genesis_sha256": release["predecessor"]["genesis_file_sha256"],
        "node_id": source_node_id,
        "validator_pubkey": source_validator_pubkey,
    }
    source_marker = "".join(
        f"{key}={value}\n" for key, value in source_marker_values.items()
    ).encode()
    export_raw = canonical_bytes(
        {
            "chain_id": "zerone-1",
            "initial_height": "1001",
            "app_state": {"fixture": {}},
            "consensus": {"params": {}},
        }
    )
    export_evidence = {
        "schema": "zerone-1-post-anchor-state-export-evidence-v1",
        "chain_id": "zerone-1",
        "checkpoint_state_height": "1000",
        "exported_application_height": "1001",
        "post_anchor_app_hash": post_anchor_app_hash,
        "raw_export_sha256": digest(export_raw),
        "included_in_successor_inventory": False,
        "result": "MATCH",
    }
    offline_snapshot = {
        "schema": "zerone-1-offline-halted-observer-snapshot-manifest-v1",
        "chain_id": "zerone-1",
        "checkpoint_state_height": "1000",
        "checkpoint_app_hash": checkpoint_app_hash,
        "final_committed_height": "1001",
        "halt_trigger_height": "1002",
        "blockstore_height": "1002",
        "abci_last_applied_height": "1001",
        "post_anchor_app_hash": post_anchor_app_hash,
        "source_observer_node_id": source_node_id,
        "database_snapshot_sha256": "6" * 64,
        "file_manifest_sha256": "7" * 64,
        "stored_offline": True,
        "included_in_authority_bundle": False,
        "contains_signer_keys": False,
        "result": "MATCH",
    }
    sanitized_snapshot = {
        "schema": "zerone-1-sanitized-snapshot-manifest-v2",
        "chain_id": "zerone-1",
        "checkpoint_state_height": "1000",
        "checkpoint_app_hash": checkpoint_app_hash,
        "final_committed_height": "1001",
        "halt_trigger_height": "1002",
        "blockstore_height": "1001",
        "abci_last_applied_height": "1001",
        "post_anchor_app_hash": post_anchor_app_hash,
        "source_observer_node_id": source_node_id,
        "contains_staged_h": False,
        "contains_signer_keys": False,
        "contains_authority_artifacts": False,
        "database_snapshot_sha256": "8" * 64,
        "file_manifest_sha256": "9" * 64,
        "result": "MATCH",
    }
    offline_snapshot_raw = canonical_bytes(offline_snapshot)
    sanitized_snapshot_raw = canonical_bytes(sanitized_snapshot)
    rollback_output = b"fixture: rolled back blockstore H to A and removed staged H\n"
    rollback_log = {
        "schema": "zerone-1-archive-rollback-evidence-v2",
        "chain_id": "zerone-1",
        "source_offline_snapshot_manifest_sha256": digest(offline_snapshot_raw),
        "sanitized_snapshot_manifest_sha256": digest(sanitized_snapshot_raw),
        "from_blockstore_height": "1002",
        "to_blockstore_height": "1001",
        "abci_last_applied_height": "1001",
        "post_anchor_app_hash": post_anchor_app_hash,
        "command": ["zeroned", "rollback", "--hard"],
        "exit_code": 0,
        "raw_output_sha256": digest(rollback_output),
        "staged_h_removed": True,
        "result": "MATCH",
    }
    allowlist_manifest = {
        "schema": "zerone-1-pre-transition-allowlist-v1",
        "chain_id": "zerone-1",
        "contains_signer_keys": False,
        "contains_future_authority_artifacts": False,
        "allowed_entries_sha256": "8" * 64,
        "result": "MATCH",
    }
    source_raw = {
        "ZERONE-1-INVENTORY-V3.json": canonical_bytes(inventory),
        "SIGNER-EVIDENCE-MANIFEST.json": canonical_bytes(signer_evidence),
        "OBSERVER-EVIDENCE-MANIFEST.json": canonical_bytes(observer_evidence),
        **rpc_raw,
        "POST-ANCHOR-STATE-EXPORT.json.raw": export_raw,
        "POST-ANCHOR-STATE-EXPORT-EVIDENCE.json": canonical_bytes(export_evidence),
        "OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json": offline_snapshot_raw,
        "SOURCE-OBSERVER-RUNTIME-MARKER": source_marker,
        "PRE-TRANSITION-SANITIZED-SNAPSHOT-MANIFEST.json": sanitized_snapshot_raw,
        "ARCHIVE-ROLLBACK-OUTPUT.log": rollback_output,
        "ARCHIVE-ROLLBACK-LOG.json": canonical_bytes(rollback_log),
        "PRE-TRANSITION-ALLOWLIST-MANIFEST.json": canonical_bytes(
            allowlist_manifest
        ),
    }
    for name, data in source_raw.items():
        raw[name] = data
        (output / name).write_bytes(data)

    transition_manifest = {
        "schema": "zerone-1-archive-transition-v1",
        "chain_id": "zerone-1",
        "checkpoint_state_height": "1000",
        "final_committed_height": "1001",
        "halt_trigger_height": "1002",
        "genesis_sha256": release["predecessor"]["genesis_file_sha256"],
        "cutover_initiation_evidence": {
            "successor_transaction_hash": args.tx_hash,
            "committed_height": "90",
            "committed_block_time": "2026-07-10T14:00:00.333333333Z",
            "public_notice_sha256": digest(raw["PUBLIC-NOTICE.md"]),
            "public_notice_publication_evidence_sha256": "c" * 64,
            "initiation_evidence_sha256": cutover_init_pair["sha256"],
            "initiation_evidence_detached_signature_sha256": cutover_init_pair[
                "detached_signature_sha256"
            ],
        },
        "source_observer": {
            "runtime_marker_sha256": digest(source_marker),
            "node_id": source_node_id,
            "validator_pubkey": source_validator_pubkey,
        },
        "candidate": {
            "node_id": candidate_node_id,
            "validator_pubkey": candidate_validator_pubkey,
        },
        "expected_anchor_block_hash": anchor_block_hash,
        "expected_post_anchor_app_hash": post_anchor_app_hash,
        "source_evidence": {
            "signer_manifest_sha256": digest(source_raw["SIGNER-EVIDENCE-MANIFEST.json"]),
            "observer_manifest_sha256": digest(
                source_raw["OBSERVER-EVIDENCE-MANIFEST.json"]
            ),
        },
        "archive_construction_evidence": {
            "pre_transition_sanitized_snapshot_sha256": digest(
                source_raw["PRE-TRANSITION-SANITIZED-SNAPSHOT-MANIFEST.json"]
            ),
            "rollback_log_sha256": digest(source_raw["ARCHIVE-ROLLBACK-LOG.json"]),
            "pre_transition_allowlist_manifest_sha256": digest(
                source_raw["PRE-TRANSITION-ALLOWLIST-MANIFEST.json"]
            ),
            "excluded_future_artifacts": [
                "archive transition manifest",
                "rendered Fly configs",
                "archive adoption authority",
                "archive readiness",
                "final checkpoint",
                "open-beta decision",
            ],
        },
        "archive_transition_nonce": "9" * 64,
    }
    write_json(output, "zerone-1-archive-transition.json", transition_manifest)
    transition_sha = digest((output / "zerone-1-archive-transition.json").read_bytes())

    archive_replacements = {
        "REPLACE_WITH_PINNED_ZERONE_1_HALT_IMAGE_DIGEST": args.halt_image,
        "REPLACE_WITH_F": "1000",
        "REPLACE_WITH_A": "1001",
        "REPLACE_WITH_H": "1002",
        "REPLACE_WITH_SIGNER_EVIDENCE_SHA256": transition_manifest[
            "source_evidence"
        ]["signer_manifest_sha256"],
        "REPLACE_WITH_OBSERVER_EVIDENCE_SHA256": transition_manifest[
            "source_evidence"
        ]["observer_manifest_sha256"],
        "REPLACE_WITH_SOURCE_OBSERVER_MARKER_SHA256": transition_manifest[
            "source_observer"
        ]["runtime_marker_sha256"],
        "REPLACE_WITH_SOURCE_OBSERVER_NODE_ID": source_node_id,
        "REPLACE_WITH_SOURCE_OBSERVER_VALIDATOR_PUBKEY_BASE64": source_validator_pubkey,
        "REPLACE_WITH_UPPERCASE_A_BLOCK_HASH": anchor_block_hash,
        "REPLACE_WITH_UPPERCASE_POST_A_APP_HASH": post_anchor_app_hash,
        "REPLACE_WITH_ARCHIVE_TRANSITION_MANIFEST_SHA256": transition_sha,
    }
    archive_config_bytes: dict[str, bytes] = {}
    for filename, template_path in (
        ("fly.archive-candidate.toml", candidate_template_path),
        ("fly.archive.toml", archive_template_path),
    ):
        text = template_path.read_text()
        for old, new in archive_replacements.items():
            text = text.replace(old, new)
        archive_config_bytes[filename] = text.encode()
        (output / filename).write_bytes(archive_config_bytes[filename])

    adoption = fill_placeholders(
        read_json(templates / "ARCHIVE-ADOPTION-AUTHORITY.example.json"),
        args.main,
        args.transition,
    )
    adoption.update(
        attestation_result="MATCH",
        release_packet=release_pair,
        cutover_decision=cutover_pair,
        cutover_initiation_evidence=cutover_init_pair,
    )
    adoption["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.transition,
        "detached_signature_filename": "ARCHIVE-ADOPTION-AUTHORITY.json.sig",
    }
    adoption["archive_transition_manifest"].update(
        sha256=digest((output / "zerone-1-archive-transition.json").read_bytes()),
        archive_transition_nonce=transition_manifest["archive_transition_nonce"],
        source_evidence=transition_manifest["source_evidence"],
        source_observer=transition_manifest["source_observer"],
        candidate=transition_manifest["candidate"],
        expected_anchor_block_hash=transition_manifest["expected_anchor_block_hash"],
        expected_post_anchor_app_hash=transition_manifest[
            "expected_post_anchor_app_hash"
        ],
    )
    adoption["checkpoint_plan"] = {
        "checkpoint_state_height": "1000",
        "final_committed_anchor_height": "1001",
        "halt_trigger_height": "1002",
    }
    adoption["cutover_initiation_evidence"] = copy.deepcopy(
        transition_manifest["cutover_initiation_evidence"]
    )
    adoption["archive_construction_evidence"] = copy.deepcopy(
        transition_manifest["archive_construction_evidence"]
    )
    adoption["deployment_configs"]["zerone_1_archive_candidate"].update(
        image_ref=args.halt_image,
        template_sha256=digest(candidate_template_path.read_bytes()),
        sha256=digest(archive_config_bytes["fly.archive-candidate.toml"]),
    )
    adoption["deployment_configs"]["zerone_1_archive"].update(
        image_ref=args.halt_image,
        template_sha256=digest(archive_template_path.read_bytes()),
        sha256=digest(archive_config_bytes["fly.archive.toml"]),
    )
    adoption["render_contract"].update(
        renderer_sha256=digest(renderer_path.read_bytes()),
        archive_candidate_template_sha256=digest(candidate_template_path.read_bytes()),
        archive_template_sha256=digest(archive_template_path.read_bytes()),
    )
    write_json(output, "ARCHIVE-ADOPTION-AUTHORITY.json", adoption)
    adoption_pair = pair(
        output,
        "ARCHIVE-ADOPTION-AUTHORITY.json",
        "ARCHIVE-ADOPTION-AUTHORITY.json.sig",
    )

    transition_sha = digest((output / "zerone-1-archive-transition.json").read_bytes())
    candidate_readiness = {
        "schema": "zerone-1-archive-readiness-v2",
        "chain_id": "zerone-1",
        "checkpoint_state_height": "1000",
        "final_committed_height": "1001",
        "halt_trigger_height": "1002",
        "anchor_block_hash": anchor_block_hash,
        "post_anchor_app_hash": post_anchor_app_hash,
        "halt_trigger_block_absent": True,
        "halt_trigger_results_absent": True,
        "block_sync_catching_up": True,
        "anchor_commit_canonical": False,
        "source_evidence": transition_manifest["source_evidence"],
        "transition_manifest_sha256": transition_sha,
        "archive_transition_nonce": transition_manifest["archive_transition_nonce"],
        "node_id": transition_manifest["candidate"]["node_id"],
        "validator_pubkey": transition_manifest["candidate"]["validator_pubkey"],
    }
    raw["ARCHIVE-CANDIDATE-READINESS.json"] = canonical_bytes(candidate_readiness)
    (output / "ARCHIVE-CANDIDATE-READINESS.json").write_bytes(
        raw["ARCHIVE-CANDIDATE-READINESS.json"]
    )
    readiness_sha = digest(raw["ARCHIVE-CANDIDATE-READINESS.json"])
    marker_values = {
        "runtime_version": "2",
        "role": "archive",
        "chain_id": "zerone-1",
        "genesis_sha256": release["predecessor"]["genesis_file_sha256"],
        "node_id": transition_manifest["candidate"]["node_id"],
        "validator_pubkey": transition_manifest["candidate"]["validator_pubkey"],
        "archive_transition_nonce": transition_manifest["archive_transition_nonce"],
        "archive_transition_manifest_sha256": transition_sha,
        "archive_readiness_sha256": readiness_sha,
    }
    raw["ARCHIVE-FINAL-RUNTIME-MARKER"] = "".join(
        f"{key}={value}\n" for key, value in marker_values.items()
    ).encode()
    (output / "ARCHIVE-FINAL-RUNTIME-MARKER").write_bytes(
        raw["ARCHIVE-FINAL-RUNTIME-MARKER"]
    )
    marker_sha = digest(raw["ARCHIVE-FINAL-RUNTIME-MARKER"])
    private_probe = {
        "schema": "zerone-1-private-aa-probe-evidence-v1",
        "chain_id": "zerone-1",
        "checkpoint_state_height": "1000",
        "final_committed_height": "1001",
        "halt_trigger_height": "1002",
        "anchor_block_hash": anchor_block_hash,
        "post_anchor_app_hash": post_anchor_app_hash,
        "halt_trigger_block_absent": True,
        "archive_node_id": transition_manifest["candidate"]["node_id"],
        "transition_manifest_sha256": transition_sha,
        "candidate_readiness_sha256": readiness_sha,
        "final_runtime_marker_sha256": marker_sha,
        "status_response_sha256": "a" * 64,
        "abci_info_response_sha256": "b" * 64,
        "anchor_block_response_sha256": "c" * 64,
        "halt_absence_response_sha256": "d" * 64,
        "observed_at": "2026-07-10T14:45:00Z",
        "result": "MATCH",
    }
    raw["ARCHIVE-PRIVATE-A-A-PROBE-EVIDENCE.json"] = canonical_bytes(private_probe)
    (output / "ARCHIVE-PRIVATE-A-A-PROBE-EVIDENCE.json").write_bytes(
        raw["ARCHIVE-PRIVATE-A-A-PROBE-EVIDENCE.json"]
    )
    successor_revalidation = {
        "schema": "zerone-2-successor-revalidation-evidence-v1",
        "chain_id": "zerone-2",
        "release_packet": release_pair,
        "genesis_sha256": args.genesis_sha,
        "observed_height": "1200",
        "observed_block_time": "2026-07-10T15:30:00.444444444Z",
        "validator_app_hash": "F" * 64,
        "edge_app_hash": "F" * 64,
        "sdk_validator_count": 1,
        "validator_consensus_pubkey": release["public_identities"][
            "validator_consensus_pubkey"
        ],
        "supply_uzrn": "13555000000",
        "protocol_dark_audit_sha256": "e" * 64,
        "operator_onboarding_transaction_hash": dark[
            "private_bootstrap_transactions"
        ]["operator_onboarding"]["expected_transaction_hash"],
        "custom_validator_registration_transaction_hash": dark[
            "private_bootstrap_transactions"
        ]["custom_validator_registration"]["expected_transaction_hash"],
        "result": "MATCH",
    }
    raw["SUCCESSOR-REVALIDATION-EVIDENCE.json"] = canonical_bytes(
        successor_revalidation
    )
    (output / "SUCCESSOR-REVALIDATION-EVIDENCE.json").write_bytes(
        raw["SUCCESSOR-REVALIDATION-EVIDENCE.json"]
    )

    readiness = {
        "candidate_readiness_sha256": digest(raw["ARCHIVE-CANDIDATE-READINESS.json"]),
        "final_runtime_marker_sha256": digest(raw["ARCHIVE-FINAL-RUNTIME-MARKER"]),
        "private_a_a_probe_evidence_sha256": digest(
            raw["ARCHIVE-PRIVATE-A-A-PROBE-EVIDENCE.json"]
        ),
    }
    final = fill_placeholders(
        read_json(final_template_path), args.main, args.transition
    )
    final["created_at"] = "2026-07-10T15:00:00Z"
    final["authority_chain"] = {
        "release_packet": release_pair,
        "dark_start_decision": dark_pair,
        "dark_start_initiation_evidence": dark_init_pair,
        "dark_registration_evidence": dark_registration_pair,
        "cutover_decision": cutover_pair,
        "cutover_initiation_evidence": cutover_init_pair,
        "archive_adoption_authority": adoption_pair,
        "archive_transition_manifest_sha256": digest(
            (output / "zerone-1-archive-transition.json").read_bytes()
        ),
    }
    final["checkpoint_state"].update(
        height="1000",
        app_hash=checkpoint_app_hash,
        inventory_v3_sha256=digest(source_raw["ZERONE-1-INVENTORY-V3.json"]),
    )
    final["final_application_block"].update(
        height="1001",
        block_id_hash=anchor_block_hash,
        header_app_hash=checkpoint_app_hash,
        time=anchor_block_time,
    )
    final["halt_trigger_tip"].update(
        height="1002",
        blockstore_status_tip_before_fencing="1002",
        staged_block_id_hash=halt_trigger_block_hash,
        staged_header_last_block_id_hash=anchor_block_hash,
    )
    final["excluded_post_anchor_state"].update(
        abci_last_applied_height="1001", app_hash=post_anchor_app_hash
    )
    final["terminal_rpc_evidence"] = {
        "sources": {
            "official_signer": {
                "status_json_sha256": digest(
                    rpc_raw["SIGNER-RPC-STATUS.json.raw"]
                ),
                "sha256_manifest_sha256": digest(
                    source_raw["SIGNER-EVIDENCE-MANIFEST.json"]
                ),
            },
            "independent_observer": {
                "status_json_sha256": digest(
                    rpc_raw["OBSERVER-RPC-STATUS.json.raw"]
                ),
                "sha256_manifest_sha256": digest(
                    source_raw["OBSERVER-EVIDENCE-MANIFEST.json"]
                ),
            },
        },
        "matching_payload_sha256": {
            key: digest(payload) for key, payload in common_rpc_payloads.items()
        },
        "matched_payloads": [
            "genesis",
            "trusted_block",
            "trusted_commit",
            "trusted_validators",
            "block_A",
            "commit_A",
            "validators_A",
            "block_results_A",
            "block_H",
            "commit_H",
            "validators_H",
            "abci_info",
            "block_results_H_missing",
        ],
    }
    final["genesis"].update(
        raw_file_sha256=release["predecessor"]["genesis_file_sha256"],
        rpc_canonical_sha256=inventory["source"]["rpc_genesis_canonical_sha256"],
    )
    final["artifacts"] = {
        "post_anchor_state_export_sha256": digest(export_raw),
        "post_anchor_state_export_included_in_successor_inventory": False,
        "offline_halted_observer_database_snapshot_sha256": offline_snapshot[
            "database_snapshot_sha256"
        ],
        "sanitized_a_a_database_snapshot_sha256": sanitized_snapshot[
            "database_snapshot_sha256"
        ],
        "archive_rollback_log_sha256": digest(
            source_raw["ARCHIVE-ROLLBACK-LOG.json"]
        ),
    }
    final["successor"].update(
        chain_id="zerone-2",
        genesis_time=release["genesis"]["time"],
        genesis_sha256=args.genesis_sha,
    )
    final["successor"]["release"] = {
        "source_commit": release["source"]["commit"],
        "signed_tag": release["source"]["signed_annotated_tag"],
        "tag_signer_fingerprint": args.main,
        "binary_sha256": args.runtime_binary_sha,
        "runtime_image_ref": args.runtime_image,
        "query_gateway_image_ref": args.query_image,
    }
    final["successor_commitment"] = {
        "transaction_hash": args.tx_hash,
        "committed_height": "90",
        "committed_block_time": "2026-07-10T14:00:00.333333333Z",
        "public_notice_sha256": digest(raw["PUBLIC-NOTICE.md"]),
        "public_notice_publication_evidence_sha256": "c" * 64,
        "memo": cutover_memo,
    }
    final["source_halt_release"] = {
        "source_commit": release["source"]["commit"],
        "signed_tag": release["source"]["signed_annotated_tag"],
        "tag_signer_fingerprint": args.main,
        "binary_sha256": args.halt_binary_sha,
        "image_ref": args.halt_image,
    }
    final["deployment_config_sha256"] = {
        "zerone_1_halt_signer": digest(halt_signer_config),
        "zerone_1_observer": digest(observer_config),
        "zerone_1_archive_candidate": adoption["deployment_configs"][
            "zerone_1_archive_candidate"
        ]["sha256"],
        "zerone_1_archive": adoption["deployment_configs"]["zerone_1_archive"][
            "sha256"
        ],
    }
    final["archive"].update(
        blockstore_status_height="1001",
        abci_last_applied_height="1001",
        **readiness,
    )
    final["attestation"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.transition,
        "detached_signature_filename": "FINAL-CHECKPOINT.json.sig",
        "authority_limit": final["attestation"]["authority_limit"],
    }
    write_json(output, "FINAL-CHECKPOINT.json", final)
    final_pair = pair(
        output, "FINAL-CHECKPOINT.json", "FINAL-CHECKPOINT.json.sig"
    )

    open_beta = fill_placeholders(
        read_json(templates / "OPEN-BETA-DECISION.example.json"),
        args.main,
        args.transition,
    )
    open_beta.update(
        decision="GO",
        created_at="2026-07-10T16:00:00Z",
        release_packet=release_pair,
        dark_start_decision=dark_pair,
        dark_start_initiation_evidence=dark_init_pair,
        dark_registration_evidence=dark_registration_pair,
        cutover_decision=cutover_pair,
        cutover_initiation_evidence=cutover_init_pair,
        archive_adoption_authority=adoption_pair,
        final_checkpoint=final_pair,
        archive_readiness=readiness,
        successor_revalidation_evidence_sha256=digest(
            raw["SUCCESSOR-REVALIDATION-EVIDENCE.json"]
        ),
        public_notice_sha256=digest(raw["PUBLIC-NOTICE.md"]),
    )
    open_beta["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.main,
        "detached_signature_filename": "OPEN-BETA-DECISION.json.sig",
    }
    open_beta["deployment_configs"] = {
        key: copy.deepcopy(release["deployment_configs"][key])
        for key in (
            "zerone_2_edge_public",
            "zerone_2_gateway_public",
            "zerone_1_archive_gateway",
        )
    }
    public_coordinates = {
        "zerone_2_p2p": f"{release['public_identities']['edge_node_id']}@p2p.example:26656",
        "zerone_2_rpc": "https://rpc.example",
        "zerone_2_rest": "https://rpc.example",
        "zerone_1_archive_rpc": "https://archive.example",
        "zerone_1_archive_rest": "https://archive.example",
    }
    dns_manifest = {
        "schema": "zerone-public-dns-change-manifest-v1",
        "records": {
            "p2p.example": {
                "app": "zerone-2-edge",
                "port": 26656,
                "config_sha256": release["deployment_configs"][
                    "zerone_2_edge_public"
                ]["sha256"],
            },
            "rpc.example": {
                "app": "zerone-2-gateway",
                "https": True,
                "config_sha256": release["deployment_configs"][
                    "zerone_2_gateway_public"
                ]["sha256"],
            },
            "archive.example": {
                "app": "zerone-1-archive-gateway",
                "https": True,
                "config_sha256": release["deployment_configs"][
                    "zerone_1_archive_gateway"
                ]["sha256"],
            },
        },
        "coordinates": public_coordinates,
        "result": "MATCH",
    }
    raw["DNS-CHANGE-MANIFEST.json"] = canonical_bytes(dns_manifest)
    (output / "DNS-CHANGE-MANIFEST.json").write_bytes(raw["DNS-CHANGE-MANIFEST.json"])
    open_beta["public_coordinates"] = {
        **public_coordinates,
        "canonical_dns_change_manifest_sha256": digest(
            raw["DNS-CHANGE-MANIFEST.json"]
        ),
    }
    open_beta["history_link_transaction"] = {
        "chain_id": "zerone-2",
        "signed_tx_bytes_sha256": args.tx_raw_sha,
        "expected_transaction_hash": args.tx_hash,
        "sender": args.sender,
        "message_type": "/cosmos.bank.v1beta1.MsgSend",
        "recipient_equals_sender": True,
        "amount": "1uzrn",
        "fee": "200000uzrn",
        "gas_limit": "200000",
        "timeout_height": "1300",
        "memo": f"zerone_1_final_checkpoint_sha256={final_pair['sha256']}",
        "must_not_reference_open_beta_payload": True,
    }
    open_beta["authorization_semantics"].update(
        initiation_deadline="2099-07-12T12:00:00Z",
        broadcast_not_after="2099-07-12T11:55:00Z",
        minimum_inclusion_margin_seconds=300,
    )
    write_json(output, "OPEN-BETA-DECISION.json", open_beta)
    open_pair = pair(
        output, "OPEN-BETA-DECISION.json", "OPEN-BETA-DECISION.json.sig"
    )

    open_init = fill_placeholders(
        read_json(templates / "OPEN-BETA-INITIATION-EVIDENCE.example.json"),
        args.main,
        args.transition,
    )
    open_init.update(
        attestation_result="MATCH",
        created_at="2026-07-10T16:11:00Z",
        open_beta_decision=open_pair,
        initiation_deadline="2099-07-12T12:00:00Z",
        deadline_satisfied=True,
    )
    open_init["signature_authority"] = {
        "algorithm": "openpgp",
        "authorized_signer_fingerprint": args.main,
        "detached_signature_filename": "OPEN-BETA-INITIATION-EVIDENCE.json.sig",
    }
    open_init["history_link_transaction"] = {
        "signed_tx_bytes_sha256": args.tx_raw_sha,
        "expected_transaction_hash": args.tx_hash,
        "committed_transaction_hash": args.tx_hash,
        "deliver_code": 0,
        "committed_height": "1000",
        "committed_block_time": "2026-07-10T16:10:00.555555555Z",
        "raw_transaction_query_evidence_sha256": "1" * 64,
        "independent_edge_evidence_sha256": "2" * 64,
    }
    write_json(output, "OPEN-BETA-INITIATION-EVIDENCE.json", open_init)


if __name__ == "__main__":
    main()
