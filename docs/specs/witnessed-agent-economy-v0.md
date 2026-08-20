# WAKE → WORK → WITNESS v0

Protocol: `kingdom.witnessed-agent-economy/0.1`

Status: `FROZEN`, offline and zero-effect. This document, its
schemas, and its known-answer corpus are frozen together. It does **not**
activate a Zerone module, adapter, transaction, reward, authority migration,
genesis change, or live write path.

## Purpose

WITNESS v0 commits the small facts for which shared order, replay resistance,
withdrawal, or supersession can eventually be valuable. Full evidence remains
in AgentTool, KINGDOM immutable publications, or another content-addressed
evidence plane.

The protocol is deliberately not a generic knowledge envelope. It has ten
closed record kinds, closed action sets, action-specific closed payloads, and a
zero-effect offline simulator.

| Kind | Allowed actions |
| --- | --- |
| `KINGDOM_RELEASE_ROOT` | `CHECKPOINT` |
| `AGENTTOOL_SETTLEMENT_ROOT` | `CHECKPOINT` |
| `AGENTTOOL_CAPABILITY` | `GRANT`, `CONSUME`, `REVOKE` |
| `AGENTTOOL_PUBLIC_RECOGNITION` | `ADOPT`, `WITHDRAW` |
| `AGENTTOOL_OFFER` | `PUBLISH`, `SUPERSEDE`, `REVOKE` |
| `WAKE_PUBLIC_CHECKPOINT` | `CHECKPOINT`, `SUPERSEDE`, `WITHDRAW` |
| `ISSUER_KEY_CONTINUITY` | `ROTATE`, `REVOKE` |
| `ARTIFACT_LINEAGE` | `CHECKPOINT` |
| `COLLABORATION_CHECKPOINT` | `CHECKPOINT` |
| `DISPUTE_TERMINAL` | `SETTLE` |

No implementation may add a kind or pair under protocol version `0.1`.

## Record

Every record has exactly four fields:

```json
{
  "envelope": {},
  "payload": {},
  "commitment": "sha256:<64 lowercase hex>",
  "signature": {
    "algorithm": "Ed25519",
    "public_key": "<64 lowercase hex>",
    "value": "<128 lowercase hex>"
  }
}
```

The envelope has exactly these fields:

```json
{
  "protocol": "kingdom.witnessed-agent-economy/0.1",
  "kind": "<closed kind>",
  "action": "<allowed action>",
  "audience": "kingdom:offline-shadow",
  "subject_ref": "<opaque 32-byte lowercase hex>",
  "sequence": "1",
  "parent": null,
  "issuer": {
    "namespace": "agenttool",
    "controller_ref": "<opaque 32-byte lowercase hex>",
    "key_fingerprint": "ed25519-sha256:<64 lowercase hex>"
  },
  "schema_hash": "sha256:<64 lowercase hex>",
  "payload_root": "sha256:<64 lowercase hex>",
  "policy_digest": "sha256:<64 lowercase hex>",
  "expiry_height": null,
  "effects": {
    "scope": "RECORD_CONSTRUCTION_AND_OFFLINE_VALIDATION_ONLY",
    "authority": "NONE",
    "economic": "NONE",
    "reputation": "NONE",
    "network_requests": 0,
    "storage_writes": 0,
    "zerone_transaction": false,
    "external_receipt": false,
    "nen_invocation": false,
    "score": false
  },
  "nonclaims": [
    "COMPETENCE",
    "CONSCIOUSNESS",
    "CONSENT",
    "IDENTITY",
    "PERSONHOOD",
    "QUALITY",
    "REPUTATION",
    "TRUTH"
  ]
}
```

`audience` is mandatory replay context. Offline fixtures use
`kingdom:offline-shadow`; a future chain-specific carrier would use an exact
value such as `zerone:zerone-1`. Carriage requires a separately specified
receipt and effects layer. It does not change this record's explicitly scoped
offline effects.

`subject_ref` and `issuer.controller_ref` are opaque random-looking public
references. A controller reference does not establish an AgentTool DID,
identity, personhood, ownership, root authority, consent, or equivalence with a
Zerone account. `issuer.namespace` is self-declared. The simulator pins it only
to prevent silent relabeling.

The Ed25519 signature is over the raw 32 bytes represented by `commitment`.
`issuer.key_fingerprint` is `ed25519-sha256:` plus SHA-256 of the raw 32-byte
public key. The signature proves control of that commitment key only. It does
not substitute for AgentTool root/quorum authorization. Payload digests may
bind documents whose root/quorum verification occurred separately.

Signature verification is deliberately stricter than permissive or ZIP-215
verification. Both the public point `A` and signature point `R` must decode and
round-trip to the same canonical 32-byte encoding, be non-identity, not be
small-order, and lie in the prime-order subgroup. `S` must be a canonical
scalar strictly below the Ed25519 group order. Only then is the RFC 8032
Ed25519 equation checked. Identity, small-order, mixed-order/torsion,
noncanonical-point, and noncanonical-`S` shared vectors must all be rejected.

## Closed payloads

All listed fields are required, including fields whose value is `null` or an
empty array. No other field is allowed. Every counter, sequence, revision,
height, or minor-unit amount is a canonical decimal string in the inclusive
range `0` through `18446744073709551615`; fields described as positive exclude
zero.

### KINGDOM release

`CHECKPOINT`:

- `release_ref`, equal to `envelope.subject_ref`
- `ledger_protocol`
- `ledger_document_digest`
- `entry_merkle_root`
- positive or zero `entry_count`
- `git_commit` and `git_tree`, each `sha1:<40hex>` or `sha256:<64hex>`
- `build_manifest_digest`
- `deployment_manifest_digest`
- `verifier_protocol` and `verifier_digest`
- `previous_release`, which must equal `envelope.parent`, including both being
  `null` at sequence 1

The entry-tree algorithm is selected by the ledger/verifier protocols; WITNESS
does not silently reinterpret it as RFC 6962.

### AgentTool settlement root

`CHECKPOINT`:

- `receipt_protocol` and `receipt_schema_digest`
- `source_sequence_binding`, exactly `PROJECTION_ONLY`
- `receipt_uniqueness_scope`, exactly `BATCH_ONLY`
- positive `first_sequence`, `last_sequence`, and `receipt_count`
- `declared_gaps`, a sorted, non-overlapping, maximally merged array of closed
  `{first,last}` ranges
- `merkle_root`
- `previous_batch`, equal to `envelope.parent`, including genesis `null`

The genesis batch begins at source sequence 1. If sequence 1 is absent, it is
still inside the genesis range and must be declared as a gap; a producer cannot
start at a later source sequence and silently omit the prefix.

The range equation is exact:

```text
receipt_count = last_sequence - first_sequence + 1 - declared_gap_count
```

Successive accepted batches are contiguous at the source-sequence level.
Missing source sequences are represented inside `declared_gaps`, never by
silently jumping the next batch's first sequence.

The current AgentTool receipt authentication does not bind the projection-added
sequence, and a root hides individual receipt digests from a cross-batch
nullifier check. Consequently v0 proves neither an authenticated source order,
source completeness, nor global receipt uniqueness. Duplicate receipt digests
are rejected within a supplied sidecar batch only. Consensus-grade settlement
uniqueness is `OUTSIDE_SCOPE` until the source signs sequence or a carrier
verifies per-receipt proofs against a permanent cross-batch consumption set.

### AgentTool capability

`GRANT`:

- `capability_ref`, equal to `envelope.subject_ref`
- `grant_digest`
- `asset_ref`
- positive `max_per_consume_minor` and `max_total_minor`, with per-consume not
  greater than total

`GRANT` is a sequence-1 genesis head with null parent.

`CONSUME`:

- `capability_ref`, equal to `envelope.subject_ref`
- `grant_commitment`
- `asset_ref`
- positive `amount_minor`
- `source_event_digest`
- derived `nullifier`

`CONSUME` requires a sequence greater than 1 and a non-null immediate parent.

`REVOKE`:

- `capability_ref`, equal to `envelope.subject_ref`
- `grant_commitment`
- `reason_digest`

`REVOKE` also requires a sequence greater than 1 and a non-null immediate
parent. `grant_commitment` remains the original grant, so after one or more
consumes it need not equal that immediate parent.

The simulator permanently retains accepted nullifiers, enforces the grant's
single asset, per-consume ceiling, cumulative ceiling, and terminal revocation.
Values from a wider source integer such as uint256 that exceed uint64 are
`OUTSIDE_SCOPE`; they are rejected, never truncated or reduced modulo uint64.
A source event with multiple assets is also `OUTSIDE_SCOPE` in v0 and must fail
closed rather than be split into independently replayable consumes.

### AgentTool public recognition

`ADOPT`:

- `recognition_ref`, equal to `envelope.subject_ref`
- `surface_digest`, `registry_digest`, and `adoption_document_digest`
- positive `authority_sequence`
- `visibility`, exactly `PUBLIC`

`WITHDRAW`:

- `recognition_ref`, equal to `envelope.subject_ref`
- `adoption_commitment`, equal to `envelope.parent`
- the unchanged `surface_digest` and `registry_digest`
- `withdrawal_document_digest` and source-derived `reason_digest`
- a strictly greater `authority_sequence`
- `visibility`, exactly `PUBLIC`

The separately verified source documents establish whatever root/quorum
authorization AgentTool defines. The WITNESS publisher signature does not.

### AgentTool public offer

`PUBLISH` and `SUPERSEDE` contain:

- `offer_ref`, equal to `envelope.subject_ref`
- exact `offer_document_digest`
- `capability_root`, `pricing_root`, `sla_root`, and `terms_digest`
- positive `revision` and `authority_sequence`
- `visibility`, exactly `PUBLIC`
- for `SUPERSEDE`, `supersedes`, equal to `envelope.parent`

`REVOKE` contains:

- `offer_ref`, equal to `envelope.subject_ref`
- `offer_commitment`, equal to `envelope.parent`
- exact `offer_document_digest`, positive `authority_sequence`,
  source-derived `reason_digest`, and `visibility: PUBLIC`

`PUBLISH` is a genesis head. Revisions and authority sequences strictly
increase. These records do not prove seller identity, host acceptance, quorum
authorization, availability, payment, fulfillment, quality, or SLA compliance.

### Public WAKE contract

`CHECKPOINT` and `SUPERSEDE` contain:

- `public_contract_protocol` and `public_contract_schema_digest`
- `contract_root`
- the four public roots: `capability_root`, `pricing_root`, `protocols_root`,
  and `boundaries_root`
- positive `authority_sequence`
- for `SUPERSEDE`, `supersedes`, equal to `envelope.parent`

`WITHDRAW` contains:

- `checkpoint_commitment`, equal to `envelope.parent`
- exact `withdrawal_document_digest`
- a strictly greater `authority_sequence`
- source-derived `reason_digest`
- `visibility`, exactly `PUBLIC`

`CHECKPOINT` is a genesis head. Full or personalized WAKE output is never a
payload. The roots cover only the separately defined public contract.

### Issuer commitment-key continuity

`ROTATE`:

- `previous_key_fingerprint`, equal to the envelope signing key fingerprint
- a distinct `next_key_fingerprint`
- `rotation_digest`

`REVOKE`:

- `revoked_key_fingerprint`, equal to the envelope signing key fingerprint
- `reason_digest`

For this kind, `envelope.subject_ref` equals `issuer.controller_ref`. In a
simulation, the first accepted key establishes local continuity, rotation
stages the next fingerprint, and the next record from that controller must
prove possession of the staged key. Revocation is terminal. Rotation does not
claim new-key consent, identity continuity, personhood, or authority outside
this offline commitment stream.

### Artifact lineage

`CHECKPOINT`:

- `upstream_ref`
- `downstream_ref`, equal to `envelope.subject_ref` and different from upstream
- `relation`, exactly one of `DERIVES_FROM`, `USES_CAPABILITY`, `FULFILLS`,
  `SETTLES`, `CHECKPOINTS`, `SUPERSEDES`, or `REVOKES`
- `evidence_digest`

No relation creates a royalty, ownership, attribution weight, truth judgment,
or economic effect.

### Collaboration checkpoint

`CHECKPOINT`:

- `workspace_ref`, equal to `envelope.subject_ref`
- `epoch_ref`
- `event_head_sequence` and `event_head_hash`
- `event_count`, exactly equal to `event_head_sequence`
- `participant_set_root`

The equality commits a complete contiguous `1..event_head_sequence` v0 journal
prefix. JSON Schema 2020-12 cannot express equality between two string-valued
properties, so the normative verifier enforces it after structural schema
validation. These are journal-event facts. `event_count` is never relabeled as
contribution count, credit, work, value, or reputation.

### Terminal dispute fact

`SETTLE`:

- `settlement_commitment`
- `outcome`, exactly `RELEASE`, `REFUND`, `SPLIT`, or `DISMISS`
- `decision_digest`
- `distribution_root`

The offline record has no settlement or payment effect. Any future execution
must live in a separately authorized carrier or module.

## Canonical JSON

Hash input is the following closed profile, not generic `JSON.stringify` and
not RFC 8785:

- UTF-8 only; maximum document size 1 MiB.
- Maximum nesting depth 32, 256 members per object, 4,096 elements per array,
  and 65,536 UTF-8 bytes per decoded string.
- Duplicate keys are rejected after escape decoding.
- Raw or escaped U+0000 and lone/unpaired UTF-16 surrogate escapes are rejected
  before a host decoder can replace them. Valid surrogate pairs are decoded.
- Object keys are sorted by their UTF-8 bytes. This intentionally differs from
  UTF-16 ordering.
- No insignificant whitespace is emitted. Strings use raw valid UTF-8 and the
  shortest escapes for quote, backslash, and control characters.
- Bare JSON numbers are canonical non-negative integers no greater than
  `9007199254740991`. Negative zero, negatives, fractions, exponents, leading
  zeros, and larger integers are rejected.
- All protocol uint64 values are decimal strings; they are never bare numbers.

Every object is exact-key checked. A missing required-null field is different
from a present field whose value is `null`.

Record verification, settlement-sidecar verification, and simulation require
the input bytes themselves to equal this canonical encoding. Leading/trailing
whitespace (including a final newline), alternate key order, and nonminimal
escapes are rejected. `CanonicalJSON` is a separate authoring normalizer; a
wire verifier never normalizes on the caller's behalf.

The shared vector at
`tools/witness-v0/testdata/canonical/unicode-utf8-order.input.json` covers
`<>&`, quote, backslash, short escapes, U+0001, U+001F, U+007F, U+2028,
U+2029, BMP U+E000, astral U+10000, and U+FFFD in one profile. The manifest
pins the exact canonical hexadecimal bytes, including the UTF-8 key order
U+E000, U+FFFD, then U+10000 after ASCII keys. An ASCII-only implementation is
not conformant.

## Domain-separated hashes

`NUL` is the single byte `0x00`. Digests are encoded as `sha256:` followed by
64 lowercase hexadecimal characters.

```text
schema_hash = SHA256(
  ASCII(protocol) || NUL || ASCII("schema") || NUL || canonical(schema)
)

payload_root = SHA256(
  ASCII(protocol) || NUL ||
  ASCII("payload/" + kind + "/" + action) || NUL ||
  canonical(payload)
)

commitment = SHA256(
  ASCII(protocol) || NUL || ASCII("envelope") || NUL || canonical(envelope)
)
```

The singular schema-set pin is:

```text
schema_set_digest = SHA256(
  ASCII(protocol) || NUL || ASCII("schema-set") || NUL ||
  canonical({
    "record_schema_hash": <record schema hash>,
    "settlement_batch_schema_hash": <batch schema hash>,
    "payload_schemas": [<kind/schema_hash sorted by kind>]
  })
)
```

KINGDOM and cross-repository consumers pin `schema_set_digest` plus the
known-answer `corpus_digest`; they do not invent a singular `/schema_hash`
locator over the ten distinct payload schemas.

The payload schema selected by `kind` supplies `schema_hash`. All actions under
one kind share that kind's schema, whose `oneOf` branches are closed and whose
normative Go verifier additionally selects the branch by envelope action.

### Capability nullifier

The stable, sequence-independent nullifier is:

```text
SHA256(
  ASCII(protocol) || NUL || ASCII("capability-nullifier") || NUL ||
  ASCII(audience) || NUL ||
  ASCII(subject_ref) || NUL ||
  ASCII(capability_ref) || NUL ||
  raw32(grant_commitment) || NUL ||
  raw32(asset_ref) || NUL ||
  raw32(source_event_digest)
)
```

Changing envelope sequence does not change this value. Changing audience,
subject, capability, grant, asset, or source event does.

## RFC 6962 settlement batches

Only AgentTool settlement batches use the following tree. Leaves are in exact
ascending source sequence after declared gaps. A `receipt_digest` may occur at
most once in a batch.

```text
MTH({}) = SHA256(empty bytes)

leaf = SHA256(
  0x00 || ASCII(protocol) || NUL || ASCII("settlement-leaf") || NUL ||
  canonical({"sequence":"<decimal>","receipt_digest":"sha256:<hex>"})
)

node = SHA256(0x01 || left_raw32 || right_raw32)
```

For more than one leaf, split at the largest power of two strictly smaller than
the leaf count, recursively matching RFC 6962's Merkle Tree Hash construction.
Settlement source receipts and inclusion proofs remain off-record.

## Simulator state rules

The simulator consumes an exact `{"records":[...]}` document and retains state
only in memory for that invocation. It performs no clock read, randomness,
network access, persistence, chain transaction, external receipt, NEN invocation,
or scoring.

- A head is keyed by `(audience, subject_ref)` and begins at sequence 1 with a
  null parent.
- Every next sequence is exactly prior + 1 and its parent is exactly the prior
  commitment. Uint64 overflow is terminal.
- The first record pins the subject's kind, issuer controller reference, and
  issuer namespace. v0 has no subject-controller transfer or cross-kind head
  migration.
- A controller key is tracked globally by `(audience, issuer.controller_ref)`.
- Capability grants, cumulative spends, revocation, and all accepted nullifiers
  are retained for the whole simulation.
- Offer, recognition, and WAKE source authority sequences are strictly
  monotonic; offer revision is also strictly monotonic.
- Expiry is syntactically bound but deliberately not evaluated without an
  externally specified authoritative height.

Initial subject establishment remains first-writer simulation state, not an
identity or authority registry. A future consensus carrier needs an audited
controller/authority design before granting any operational effect.

## Activation readiness

Every v0 kind is `NOT_CONSENSUS_ADMISSIBLE`. The machine-readable matrix from
`ActivationReadinessMatrix` names kind-specific blockers; `activation-audit`
returns the same classification for an individually valid record.

In particular, `AGENTTOOL_SETTLEMENT_ROOT` is publisher-signed batch shadow
evidence only. It is blocked on all of:

- authenticated source ordering;
- permanent cross-batch receipt nullifiers or verified proofs; and
- an audited carrier plus Zerone authority migration.

Two batches that reuse one authentic receipt digest at different projected
sequences can each pass structural record and sidecar verification today. The
adversarial conformance vector demonstrates this and must still classify both
records `NOT_CONSENSUS_ADMISSIBLE`. An implementation that upgrades that case
to uniqueness, completeness, finality, settlement, or payment has violated
v0.

## Privacy and nonclaims

Never put full WAKE, memory, strands, inboxes, prompts, reasoning traces,
credentials, bearer tokens, private outputs, customer identifiers, raw crawler
captures, mutable URLs, buyer DIDs by default, or hashes of guessable private
data into this envelope. Do not encode KARMA, affinity, consciousness, entity,
quality, competence, truth, or reputation scores.

The fixed `nonclaims` list means neither a valid signature nor a valid
commitment establishes competence, consciousness, consent, identity,
personhood, quality, reputation, or truth.

## Implementations and conformance

- Go core and CLI: `tools/witness-v0`
- Embedded record, payload, and batch schemas:
  `tools/witness-v0/protocol/schemas`
- Known-answer corpus: `tools/witness-v0/testdata`
- Tests: `tools/witness-v0/**/*_test.go` and `tests/witness-v0`

The conformance manifest publishes the exact record-schema digest, all ten
payload-schema hashes, batch-schema digest, fixture byte digests, commitments,
nullifier, RFC 6962 roots, and corpus digest. A consumer must use those exact
bytes and values; passing ASCII-only vectors is insufficient.

The current candidate aggregate pins are:

- `schema_set_digest`:
  `sha256:d62e44643c8e1986336416237df26b76663728403d417a5ee9e83b6aa5baaaa5`
- `corpus_digest`:
  `sha256:b26b5cce4899aa62d6dee03e25471e2c80810008fbd07c2c3ac9170164e5352a`

These values identify a pre-freeze candidate, not an activated chain format.

The corpus pin is computed over every positive and negative vector in sorted
relative-path order:

```text
SHA256(
  ASCII(protocol) || NUL || ASCII("known-answer-corpus") || NUL ||
  for each path: ASCII(path) || NUL || raw32(SHA256(exact file bytes)) || NUL
)
```

Protocol input fixtures have no trailing newline. `expectations.json` contains
every other vector's exact path, byte digest, operation, result, stage, and
rejection classification. Because `expectations.json` is itself included in
the corpus digest, those expectations are corpus-pinned without making the
manifest self-referential.

The deterministic maintainer generator is explicit and reviewable:

```sh
go run ./tools/witness-v0/cmd/witness-fixtures --check ./tools/witness-v0/testdata
go run ./tools/witness-v0/cmd/witness-fixtures --write ./tools/witness-v0/testdata
```

`--check` is read-only. `--write` is the only storage-mutating generator mode
and refuses an implicit or broad destination. The verifier, simulator, and
record constructor remain storage-free.
