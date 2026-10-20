#!/usr/bin/env python3
"""List fact edges that equal-ID augmentations replaced, from an exported genesis.

Run with the isolated interpreter and the standard library only:

    python3 -I scripts/audit-augmentation-edge-overwrites.py EXPORT.json [--fail-on-any]

EXPORT.json is a full exported genesis (`app_state.knowledge`) or the knowledge
module's genesis alone, for example `zeroned export --height <H-1>`. Nothing is
written; the report goes to stdout as JSON, and a one-line PREFLIGHT verdict to
stderr.

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
verdict_block, no other field), and its ID is the ID of a fact. It is then:

- restorable: the fact was accepted from a claim that names the target, the
  fact's ID derives from that claim and its verified_at_block, and a relation
  method_id, if any, is placed by a review-v2 round. `restored_edge` is the
  edge the chain will write. Heights are not compared: the fact's own write
  names its claim's submitter, so the verdict's edge at the pair proves the
  verdict wrote last, even after a relaunch at initial height 1;
- unrestorable, with the reason the chain reports. A fact that cannot be traced
  to a claim (no claim, a missing claim, an ID that does not derive from it) is
  listed only when verified_at_block <= verdict_block, as on the chain; after a
  relaunch at initial height 1 those heights may come from two chains, so such
  a listing can be missing or spurious.

A pair whose fact's claim does not name the target replaced nothing and is not
listed. An export carries each edge once, after checking that its forward and
reverse entries are equal, so the chain's reverse-entry check cannot fail on an
imported export and has no counterpart here.

Preflight. The chain reads the whole augmentation inventory and refuses it
above INVENTORY_MAX_RECORDS records or INVENTORY_MAX_BYTES bytes, charging each
record its store key (0x63 and the ID's UTF-8 bytes) plus its stored protobuf
encoding. Above either bound the knowledge-claim-integrity-v1 upgrade fails at
its height, which halts the chain there, and an import with claim integrity
refuses the genesis. `inventory` reports the same charge computed from the
export (the proto3 encoding of each record's fields, which is the stored size of
any record the chain accepts), and `preflight` says PASS or FAIL. It covers this
export's state only: export at H-1, or again after any augmentation changes. An
export omits records it cannot decode and the fields it does not know, and the
chain refuses such records as malformed, which also fails the upgrade; this
script cannot see them.

Exit status: 0 on success (1 with --fail-on-any when any pair is listed); 3
when the preflight fails, whatever else; 2 when the input cannot be read, is not
a knowledge genesis, or holds an augmentation whose stored size cannot be
computed.
"""

import argparse
import hashlib
import json
import sys

SCHEMA = "zerone.knowledge.augmentation-edge-overwrites/v1"
# keeper.augmentationRestoreMaxRecords and augmentationRestoreMaxBytes
# (recordInventoryMaxRecords, recordInventoryMaxBytes): the chain refuses the
# restoration above either.
INVENTORY_MAX_RECORDS = 100_000
INVENTORY_MAX_BYTES = 64 << 20
AUGMENTATION_KEY_PREFIX = b"\x63"  # types.AugmentationKeyPrefix
UINT64_MAX = (1 << 64) - 1

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

# Augmentation (proto/zerone/knowledge/v1/types.proto), as the export names its
# fields: field number and how the field is encoded. Every field is proto3
# without presence, so a default value is not stored.
AUGMENTATION_FIELDS = {
    "id": (1, "string"),
    "bounty_id": (2, "string"),
    "original_fact_id": (3, "string"),
    "variant_content": (4, "string"),
    "variant_reasoning_trace": (5, "string"),
    "submitter": (6, "string"),
    "created_at_block": (7, "uint64"),
    "accepted": (8, "bool"),
    "accepted_at_block": (9, "uint64"),
    "acceptance_note": (10, "string"),
    "verdict": (11, "enum"),
    "verdict_block": (12, "uint64"),
    "verdict_voters": (13, "repeated string"),
    "verdict_votes": (14, "packed enum"),
    "sponsor_vetoed": (15, "bool"),
    "payout_amount": (16, "string"),
    "verdict_vote_stakes": (17, "packed uint64"),
    "verdict_vote_calibration_bps": (18, "packed uint64"),
}


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


def _utf8(value, field):
    try:
        return value.encode("utf-8")
    except UnicodeEncodeError as err:
        raise AuditError(f"{field}: not valid UTF-8") from err


def _varint_len(value):
    length = 1
    while value >= 0x80:
        value >>= 7
        length += 1
    return length


def _uint64(value, field):
    number = _uint(value, field)
    if number > UINT64_MAX:
        raise AuditError(f"{field}: exceeds uint64")
    return number


def _field_len(number, payload_len):
    """A length-delimited field: tag, length, payload."""
    return _varint_len(number << 3) + _varint_len(payload_len) + payload_len


def augmentation_charge(aug, where):
    """What the chain charges one stored augmentation record against the inventory bound.

    keeper.passingAugmentationsChecked adds len(key) + len(value) per record: the key is
    0x63 followed by the ID's UTF-8 bytes, the value the record's protobuf encoding. The
    encoding's length is the sum over its non-default fields of tag plus payload (packed
    repeated scalars are one length-delimited field; repeated strings one field per item).
    A key this script does not know is refused rather than guessed.
    """
    size = 0
    for name, value in aug.items():
        if name not in AUGMENTATION_FIELDS:
            raise AuditError(f"{where}.{name}: not an Augmentation field; the record's stored size cannot be computed")
        number, kind = AUGMENTATION_FIELDS[name]
        field = f"{where}.{name}"
        if kind == "string":
            data = _utf8(_str(value, field), field)
            if data:
                size += _field_len(number, len(data))
        elif kind == "uint64":
            scalar = _uint64(value, field)
            if scalar:
                size += _varint_len(number << 3) + _varint_len(scalar)
        elif kind == "bool":
            if value is not None and not isinstance(value, bool):
                raise AuditError(f"{field}: not a boolean")
            if value:
                size += _varint_len(number << 3) + 1
        elif kind == "enum":
            scalar = AUGMENTATION_VERDICTS.index(_enum(value, AUGMENTATION_VERDICTS, field))
            if scalar:
                size += _varint_len(number << 3) + _varint_len(scalar)
        else:
            items = [] if value is None else value
            if not isinstance(items, list) or any(item is None for item in items):
                raise AuditError(f"{field}: not a list of values")
            if kind == "repeated string":
                for i, item in enumerate(items):
                    if not isinstance(item, str):
                        raise AuditError(f"{field}[{i}]: not a string")
                    size += _field_len(number, len(_utf8(item, f"{field}[{i}]")))
            elif items:
                if kind == "packed enum":
                    scalars = [AUGMENTATION_VERDICTS.index(_enum(item, AUGMENTATION_VERDICTS, f"{field}[{i}]")) for i, item in enumerate(items)]
                else:
                    scalars = [_uint64(item, f"{field}[{i}]") for i, item in enumerate(items)]
                size += _field_len(number, sum(_varint_len(scalar) for scalar in scalars))
    key = AUGMENTATION_KEY_PREFIX + _utf8(_str(aug.get("id"), f"{where}.id"), f"{where}.id")
    return len(key) + size


def preflight(records, size):
    """The chain's inventory bound, applied to an export's charge."""
    exceeded = [label for label, over in (
        (f"{INVENTORY_MAX_RECORDS} records", records > INVENTORY_MAX_RECORDS),
        (f"{INVENTORY_MAX_BYTES} bytes", size > INVENTORY_MAX_BYTES),
    ) if over]
    if exceeded:
        return {"verdict": "FAIL", "detail": (
            "the augmentation inventory exceeds " + " and ".join(exceeded) + ": the knowledge-claim-integrity-v1 "
            "upgrade would fail at its height and halt the chain there, and an import with claim integrity would "
            "refuse this genesis")}
    return {"verdict": "PASS", "detail": (
        "within both bounds for this export's state; re-run on an export at H-1 (spec §1.3). Records the export "
        "omits as undecodable would still fail the upgrade")}


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
    inventory_bytes = sum(augmentation_charge(aug, f"augmentations[{i}]") for i, aug in enumerate(augmentations))
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
        entry = {"augmentation_id": aug_id, "fact_id": aug_id, "target_id": target, "verdict_block": verdict_block}
        reason, restored = _decide(fact, aug_id, target, verified, verdict_block, claims, rounds)
        if reason is None and restored is None:
            continue  # nothing was replaced, or (untraced fact) it was created after the verdict
        if restored is not None:
            entry.update(status="restorable", restored_edge=restored)
        else:
            entry.update(status="unrestorable", reason=reason)
        pairs.append(entry)

    restorable = sum(1 for p in pairs if p["status"] == "restorable")
    return {
        "schema": SCHEMA,
        "augmentations": len(augmentations),
        "inventory": {"records": len(augmentations), "bytes": inventory_bytes,
                      "max_records": INVENTORY_MAX_RECORDS, "max_bytes": INVENTORY_MAX_BYTES},
        "preflight": preflight(len(augmentations), inventory_bytes),
        "fact_relation_state_present": relation_state is not None,
        "counts": {"restorable": restorable, "unrestorable": len(pairs) - restorable},
        "pairs": pairs,
    }


def _decide(fact, fact_id, target, verified, verdict_block, claims, rounds):
    """Return (reason, None), (None, restored edge) or (None, None)."""

    def untraced(reason):
        # Only the heights say whether an untraced fact existed at the verdict; they decide a
        # report, never a write (keeper.replacedFactEdge).
        return (None, None) if verified > verdict_block else (reason, None)

    claim_id = _str(fact.get("claim_id"), f"fact {fact_id}.claim_id")
    if not claim_id:
        return untraced("fact has no claim")
    claim = claims.get(claim_id)
    if claim is None:
        return untraced(f"claim {claim_id} of the fact is missing")
    if generate_fact_id(claim_id, verified) != fact_id:
        return untraced("fact id does not derive from its claim and verification height")
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
    inventory, verdict = report["inventory"], report["preflight"]
    print(f"PREFLIGHT {verdict['verdict']}: {inventory['records']} augmentation records, {inventory['bytes']} bytes "
          f"(bounds {inventory['max_records']} records, {inventory['max_bytes']} bytes); {verdict['detail']}", file=stderr)
    if verdict["verdict"] != "PASS":
        return 3
    return 1 if args.fail_on_any and report["pairs"] else 0


if __name__ == "__main__":
    sys.exit(main())
