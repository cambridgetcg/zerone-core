#!/usr/bin/env python3
"""List fact edges that equal-ID augmentations replaced, from an exported genesis.

Run with the isolated interpreter and the standard library only:

    python3 -I scripts/audit-augmentation-edge-overwrites.py EXPORT.json [--fail-on-any]

EXPORT.json is a full exported genesis (`app_state.knowledge`) or the knowledge
module's genesis alone. Nothing is written; the report goes to stdout as JSON.

Before knowledge v12 a passing augmentation verdict wrote a REFORMULATES edge at
`<augmentation id>/<original fact>`, and the ID was chosen by the augmentation's
submitter. An augmentation of D whose ID equalled the ID of a fact F that
already had its own edge to D replaced that edge (docs/specs/
knowledge-claim-integrity-v1.md §1.5). The knowledge-claim-integrity-v1
migration, and an import whose genesis selects claim integrity, restore such
edges (keeper.RestoreAugmentationReplacedEdges). This script applies the same
rules to an export, so an operator can see before a relaunch or an upgrade which
pairs will be restored and which will be reported as unrestorable.

A pair is listed when the augmentation's verdict passed, the exported edge at
its pair is exactly the edge the verdict wrote (REFORMULATES, created at
verdict_block, no other field), and its ID is the ID of a fact that existed at
the verdict (verified_at_block <= verdict_block). It is then:

- restorable: the fact was accepted from a claim that names the target, the
  fact's ID derives from that claim and its verified_at_block, and a relation
  method_id, if any, is placed by a review-v2 round. `restored_edge` is the
  edge the chain will write;
- unrestorable, with the reason the chain reports.

A pair whose fact's claim does not name the target replaced nothing and is not
listed. An export carries each edge once, after checking that its forward and
reverse entries are equal, so the chain's reverse-entry check cannot fail on an
imported export and has no counterpart here.

Exit status: 0 on success (1 with --fail-on-any when any pair is listed); 2 when
the input cannot be read or is not a knowledge genesis.
"""

import argparse
import hashlib
import json
import sys

SCHEMA = "zerone.knowledge.augmentation-edge-overwrites/v1"
# The chain refuses the restoration above this many augmentation records, or
# above 64 MiB of them (the byte size is not computable from JSON here).
INVENTORY_MAX_RECORDS = 100_000

RELATION_TYPES = [
    "RELATION_TYPE_UNSPECIFIED", "RELATION_TYPE_SUPPORTS", "RELATION_TYPE_CONTRADICTS",
    "RELATION_TYPE_REQUIRES", "RELATION_TYPE_REFINES", "RELATION_TYPE_GENERALIZES",
    "RELATION_TYPE_SUPERSEDES", "RELATION_TYPE_CITES", "RELATION_TYPE_REFORMULATES",
]
INFERENCE_TYPES = [
    "INFERENCE_TYPE_UNSPECIFIED", "INFERENCE_TYPE_DEDUCTIVE", "INFERENCE_TYPE_INDUCTIVE",
    "INFERENCE_TYPE_ABDUCTIVE", "INFERENCE_TYPE_EMPIRICAL", "INFERENCE_TYPE_ANALOGICAL",
    "INFERENCE_TYPE_CITATION",
]
AUGMENTATION_VERDICTS = [
    "AUGMENTATION_VERDICT_PENDING", "AUGMENTATION_VERDICT_EQUIVALENT", "AUGMENTATION_VERDICT_SUPERIOR",
    "AUGMENTATION_VERDICT_INFERIOR", "AUGMENTATION_VERDICT_DRIFT",
]
PASSING = {"AUGMENTATION_VERDICT_EQUIVALENT", "AUGMENTATION_VERDICT_SUPERIOR"}
COMMITMENT_SCHEME_REVIEW_V2 = 2


class AuditError(Exception):
    """The input is not a readable knowledge genesis."""


def _enum(value, names, field):
    """Protobuf JSON enums are names; numbers are accepted too."""
    if value is None:
        return names[0]
    if isinstance(value, bool):
        raise AuditError(f"{field}: not an enum value")
    if isinstance(value, int):
        if 0 <= value < len(names):
            return names[value]
        raise AuditError(f"{field}: unknown enum number {value}")
    if isinstance(value, str) and value in names:
        return value
    raise AuditError(f"{field}: unknown enum value {value!r}")


def _uint(value, field):
    """Protobuf JSON writes uint64 as a decimal string and uint32 as a number."""
    if value is None:
        return 0
    if isinstance(value, bool):
        raise AuditError(f"{field}: not an unsigned integer")
    if isinstance(value, int) and value >= 0:
        return value
    if isinstance(value, str) and value.isascii() and value.isdigit():
        return int(value)
    raise AuditError(f"{field}: not an unsigned integer: {value!r}")


def _str(value, field):
    if value is None:
        return ""
    if not isinstance(value, str):
        raise AuditError(f"{field}: not a string")
    return value


def _list(state, name):
    value = state.get(name)
    if value is None:
        return []
    if not isinstance(value, list) or not all(isinstance(item, dict) for item in value):
        raise AuditError(f"{name}: not a list of records")
    return value


def _index(records, name):
    out = {}
    for record in records:
        key = _str(record.get("id"), f"{name}.id")
        if key in out:
            raise AuditError(f"{name}: duplicate id {key!r}")
        out[key] = record
    return out


def knowledge_state(genesis):
    """Return the knowledge module state of a full genesis or of the module alone."""
    if not isinstance(genesis, dict):
        raise AuditError("genesis: not a JSON object")
    if "app_state" in genesis:
        app_state = genesis["app_state"]
        if not isinstance(app_state, dict) or not isinstance(app_state.get("knowledge"), dict):
            raise AuditError("app_state.knowledge: missing")
        return app_state["knowledge"]
    if any(name in genesis for name in ("facts", "augmentations", "fact_relation_state")):
        return genesis
    raise AuditError("genesis: neither app_state.knowledge nor a knowledge module state")


def generate_fact_id(claim_id, height):
    """keeper.GenerateFactID: the first 32 hex digits of sha256("ZRN.fact.id.v1:<claim>:<height>")."""
    return hashlib.sha256(f"ZRN.fact.id.v1:{claim_id}:{height}".encode("utf-8")).hexdigest()[:32]


def _edge(record, field):
    return {
        "source_fact_id": _str(record.get("source_fact_id"), f"{field}.source_fact_id"),
        "target_fact_id": _str(record.get("target_fact_id"), f"{field}.target_fact_id"),
        "relation": _enum(record.get("relation"), RELATION_TYPES, f"{field}.relation"),
        "created_at_block": _uint(record.get("created_at_block"), f"{field}.created_at_block"),
        "creator": _str(record.get("creator"), f"{field}.creator"),
        "inference": _enum(record.get("inference"), INFERENCE_TYPES, f"{field}.inference"),
        "inference_strength_bps": _uint(record.get("inference_strength_bps"), f"{field}.inference_strength_bps"),
        "method_id": _str(record.get("method_id"), f"{field}.method_id"),
    }


def audit(genesis):
    """Return the report for a parsed genesis (see the module docstring)."""
    state = knowledge_state(genesis)
    facts = _index(_list(state, "facts"), "facts")
    claims = _index(_list(state, "pending_claims"), "pending_claims")
    rounds = _index(_list(state, "active_rounds") + _list(state, "completed_rounds"), "rounds")
    relation_state = state.get("fact_relation_state")
    if relation_state is not None and not isinstance(relation_state, dict):
        raise AuditError("fact_relation_state: not an object")
    edges = {}
    for i, record in enumerate(_list(relation_state or {}, "relations")):
        edge = _edge(record, f"fact_relation_state.relations[{i}]")
        pair = (edge["source_fact_id"], edge["target_fact_id"])
        if pair in edges:
            raise AuditError(f"fact_relation_state: duplicate edge {pair[0]!r}->{pair[1]!r}")
        edges[pair] = edge

    augmentations = _list(state, "augmentations")
    _index(augmentations, "augmentations")
    pairs = []
    for aug in sorted(augmentations, key=lambda a: _str(a.get("id"), "augmentations.id").encode("utf-8")):
        aug_id = _str(aug.get("id"), "augmentations.id")
        target = _str(aug.get("original_fact_id"), f"augmentation {aug_id}.original_fact_id")
        verdict = _enum(aug.get("verdict"), AUGMENTATION_VERDICTS, f"augmentation {aug_id}.verdict")
        verdict_block = _uint(aug.get("verdict_block"), f"augmentation {aug_id}.verdict_block")
        if verdict not in PASSING or not aug_id or not target:
            continue
        written = {
            "source_fact_id": aug_id, "target_fact_id": target, "relation": "RELATION_TYPE_REFORMULATES",
            "created_at_block": verdict_block, "creator": "", "inference": "INFERENCE_TYPE_UNSPECIFIED",
            "inference_strength_bps": 0, "method_id": "",
        }
        if edges.get((aug_id, target)) != written:
            continue  # the verdict wrote no edge, or a later write owns the pair
        fact = facts.get(aug_id)
        if fact is None:
            continue  # an ordinary augmentation edge
        verified = _uint(fact.get("verified_at_block"), f"fact {aug_id}.verified_at_block")
        if verified > verdict_block:
            continue  # the fact was created after the verdict
        entry = {"augmentation_id": aug_id, "fact_id": aug_id, "target_id": target, "verdict_block": verdict_block}
        reason, restored = _decide(fact, aug_id, target, verified, claims, rounds)
        if reason is None and restored is None:
            continue  # the fact's claim does not name the target: nothing was replaced
        if restored is not None:
            entry.update(status="restorable", restored_edge=restored)
        else:
            entry.update(status="unrestorable", reason=reason)
        pairs.append(entry)

    restorable = sum(1 for p in pairs if p["status"] == "restorable")
    return {
        "schema": SCHEMA,
        "augmentations": len(augmentations),
        "inventory_bound_exceeded": len(augmentations) > INVENTORY_MAX_RECORDS,
        "fact_relation_state_present": relation_state is not None,
        "counts": {"restorable": restorable, "unrestorable": len(pairs) - restorable},
        "pairs": pairs,
    }


def _decide(fact, fact_id, target, verified, claims, rounds):
    """Return (reason, None), (None, restored edge) or (None, None)."""
    claim_id = _str(fact.get("claim_id"), f"fact {fact_id}.claim_id")
    if not claim_id:
        return "fact has no claim", None
    claim = claims.get(claim_id)
    if claim is None:
        return f"claim {claim_id} of the fact is missing", None
    if generate_fact_id(claim_id, verified) != fact_id:
        return "fact id does not derive from its claim and verification height", None
    named = None
    for i, relation in enumerate(_list(claim, "relations")):
        if _str(relation.get("target_fact_id"), f"claim {claim_id}.relations[{i}].target_fact_id") == target:
            named = relation
    if named is None:
        return None, None
    method = _str(named.get("method_id"), f"claim {claim_id} relation method_id")
    if method:
        round_id = _str(claim.get("verification_round_id"), f"claim {claim_id}.verification_round_id")
        round_record = rounds.get(round_id) if round_id else None
        scheme = _uint(round_record.get("commitment_scheme"), f"round {round_id}.commitment_scheme") if round_record else None
        if scheme != COMMITMENT_SCHEME_REVIEW_V2:
            return "relation method_id is set but the claim's round does not prove record integrity at the fact's creation", None
    return None, {
        "source_fact_id": fact_id,
        "target_fact_id": target,
        "relation": _enum(named.get("relation"), RELATION_TYPES, f"claim {claim_id} relation"),
        "created_at_block": verified,
        "creator": _str(claim.get("submitter"), f"claim {claim_id}.submitter"),
        "inference": _enum(named.get("inference"), INFERENCE_TYPES, f"claim {claim_id} relation inference"),
        "inference_strength_bps": _uint(named.get("inference_strength_bps"), f"claim {claim_id} relation strength"),
        "method_id": method,
    }


def main(argv=None, stdout=None, stderr=None):
    stdout = stdout or sys.stdout
    stderr = stderr or sys.stderr
    parser = argparse.ArgumentParser(description="List fact edges that equal-ID augmentations replaced, from an exported genesis.")
    parser.add_argument("genesis", help="exported genesis JSON (full, or the knowledge module state); '-' reads stdin")
    parser.add_argument("--fail-on-any", action="store_true", help="exit 1 when any pair is listed")
    args = parser.parse_args(argv)
    try:
        if args.genesis == "-":
            genesis = json.load(sys.stdin)
        else:
            with open(args.genesis, "rb") as handle:
                genesis = json.load(handle)
        report = audit(genesis)
    except (OSError, ValueError, AuditError) as err:
        print(f"audit-augmentation-edge-overwrites: {err}", file=stderr)
        return 2
    json.dump(report, stdout, indent=2, sort_keys=True)
    stdout.write("\n")
    return 1 if args.fail_on_any and report["pairs"] else 0


if __name__ == "__main__":
    sys.exit(main())
