# Zerone offline operations transition verifier

`zerone-ops` verifies canonical, append-only decision journals for planned
upgrades and hostile-incident recovery. It is offline, uses only the Go
standard library, makes no network calls, and never reads or changes chain
state.

Build or run it from the repository root:

```sh
go build -o zerone-ops ./tools/zerone-ops
go run ./tools/zerone-ops help
```

The transition schema is `zerone.ops.transition/v1`.

## One journal, one lane

Every transition explicitly selects exactly one immutable lane:

- `lane="release"` requires a non-empty `release_id` and an empty
  `incident_id`.
- `lane="incident"` requires a non-empty `incident_id` and an empty
  `release_id`.

A journal cannot carry both identifiers, change lanes, or change its selected
identifier. If a failed release becomes an incident, terminalize the release
journal where its graph permits, anchor its head, and start a separate incident
journal under its independently pinned incident policy. Use the signed
supersession sidecar described below only when replacing a frozen or
compromised trust root; never rewrite an old journal.

## Verify a journal

Pass transition files in sequence order and pin every value available through a
separate trusted channel. A release verification is:

```sh
zerone-ops verify \
  --trust-policy /protected/zerone-release-policy.json \
  --trust-policy-sha256 <64-lowercase-hex-from-trusted-channel> \
  --chain-id zerone-1 \
  --release-id release-2026-07 \
  --binary-sha256 <64-lowercase-hex> \
  --head-sha256 <64-lowercase-hex> \
  0001.json 0002.json 0003.json
```

For an incident journal, replace `--release-id` with `--incident-id`; do not
supply both. Repeat
`--power-snapshot-pin <transition-sequence>=<snapshot-sha256>` for every
power-gated transition. A successful result is one line containing the lane,
selected identity, terminal state, and verified head hash.

Exit codes are:

- `0`: the complete journal and requested external anchors are valid;
- `1`: a transition, policy, signature, quorum, or external anchor is invalid;
- `2`: command usage or input I/O failed.

`-` reads one transition from stdin. A multi-transition journal must use files
because order is security-relevant.

The trust-policy path, policy digest, and `--head-sha256` are mandatory. A path
is not itself a trust decision: provision the canonical policy outside the
journal, record its SHA-256 and the intended complete journal head in protected
configuration or independently signed operator records, and pass those
independently obtained values on every invocation. The verifier rejects a
policy file whose exact bytes do not match the pinned digest and rejects a
valid-but-truncated journal whose last transition is not the pinned head.

## Canonical JSON and hashes

Verification accepts exactly the compact bytes emitted by Go's `encoding/json`
for the relevant struct:

- fixed struct field order;
- no indentation, insignificant whitespace, or trailing newline;
- sorted lists where the schema requires sorting;
- `[]`, never `null`, for every list;
- unknown fields and additional JSON values rejected;
- lowercase 64-character hexadecimal for every SHA-256.

This is a Zerone schema encoding, not RFC 8785. Do not reserialize sealed
records with a generic JSON canonicalizer.

Transitions sort evidence by type/digest/URI and approvals by
role/identity/public key. Trust policies sort edge policies by lane/from/to,
required roles lexically, separated role pairs by role A/B, and signers by
role/identity/public key. Power-snapshot members sort by identity/public key.

`seal` refuses an incomplete approval set and emits the canonical transition
without a newline:

```sh
zerone-ops seal \
  --trust-policy /protected/zerone-release-policy.json \
  --trust-policy-sha256 <pinned-policy-sha256> \
  --input approved-draft.json > 0001.json
```

For a power-gated edge, also pass the independently obtained
`--power-snapshot-sha256`.

`transition_sha256` must be empty in the draft. The sealed value is:

```text
sha256(canonical transition JSON with transition_sha256 = "")
```

For sequence 1, `previous_transition_sha256` is empty. Every later entry must
use the preceding entry's exact `transition_sha256`, increment sequence by one,
and begin at the preceding entry's `to` state. Reordering, deletion,
duplication, insertion, and editing therefore break verification.

A hash chain is not an authenticity oracle. Anchor the head hash through a
separately protected incident channel, release record, transparency log, or
signed operator statement and verify it with `--head-sha256`.

## Forward-only states

A release journal begins:

```text
RUNNING -> PREPARING -> RELEASE_FROZEN -> SCHEDULED -> STAGED
        -> HALTED_AT_H -> MIGRATING_H -> OBSERVING -> ACCEPTED
```

`PREPARING`, `RELEASE_FROZEN`, `SCHEDULED`, or `STAGED` may move to terminal
`CANCELLED` at the edges encoded in `verify.go`. `HALTED_AT_H`, `MIGRATING_H`,
or `OBSERVING` may move to terminal `RECOVERY_FAILED`. A failed attempt is not
rewound.

An incident journal begins:

```text
RUNNING -> ASSESSING -> CONTAINING -> [SAFETY_STOPPED]
        -> RECOVERY_DESIGN -> RECOVERY_READY -> ACTIVATING
        -> OBSERVING -> CLOSED
```

The graph permits terminal `FORK_CHOICE` from its evidence-bearing decision
points and terminal `RECOVERY_FAILED` from activation or observation. There is
no automatic longest-head choice, rollback, resume, timeout transition, or
backward edge. `RECOVERY_DESIGN` and later states require a last-trusted
checkpoint.

`HALTED_AT_H` means the old binary reached the scheduled `x/upgrade` stop
before committing H. It requires the exact H−1 checkpoint; `MIGRATING_H`
requires the same. `OBSERVING` and `ACCEPTED` require a committed checkpoint at
or above H. These states do not assert that an on-chain boolean stopped
CometBFT. Incident state `SAFETY_STOPPED` is likewise an evidence-backed
operations decision about signers and observed commits. Zerone's
`x/emergency` halt is application transaction quarantine, not proof of either
state.

## Immutable release bindings

Release bindings can move from empty to populated while preparation proceeds,
but a populated value cannot change or be cleared. `RELEASE_FROZEN` and every
later non-terminal release state require:

- exact plan name and upgrade height;
- `activation_mode`, either `cosmovisor` or `immutable-image`;
- plan-info, binary, provenance, and SBOM SHA-256 values.

`immutable-image` additionally requires `image_sha256`. The state-manifest
digest remains optional in the machine schema but should be bound whenever the
release changes or audits state. A nonzero checkpoint binds exact height, block
ID hash, and AppHash. Checkpoint height cannot decrease, and hashes cannot
change at the same height.

## Trust policy, roles, and separation

The out-of-journal trust schema is `zerone.ops.trust-policy/v1`. Its fixed
field order is:

```text
schema, policy_id, chain_id, incident_id, release_id,
approval_policies [{
  lane, from, to,
  approval_policy {
    minimum_approvals, minimum_distinct_identities,
    required_roles,
    separated_role_pairs [{role_a, role_b}],
    power_quorum {role, numerator, denominator, strict}
  }
}],
signers [{role, identity, public_key}]
```

The policy must contain one edge policy for every allowed edge in every lane it
provisions. Signers carry stable role/identity/Ed25519-key tuples; dynamic
validator power is not stored on a signer. A policy may provision both lane
edge sets, but that does not permit an individual journal to carry both
identifiers.

The verifier has built-in safety roles and mandatory separation on critical
edges:

- release freeze requires distinct `release-author` and `release-verifier`;
- scheduling requires distinct `governance-coordinator` and
  `release-verifier`;
- staging requires distinct `release-verifier` and `validator-operator`, plus
  a strict validator-power threshold whose configured ratio is at least 2/3;
- cancellation from `SCHEDULED` or `STAGED` requires distinct
  `governance-coordinator` and `release-verifier`;
- `RECOVERY_DESIGN -> RECOVERY_READY` requires mutually separated
  `ibc-lead`, `incident-commander`, `release-verifier`, and
  `supply-verifier`;
- `RECOVERY_READY -> ACTIVATING` adds a separated `validator-operator` and the
  strict validator-power threshold;
- every `FORK_CHOICE` edge requires mutually separated
  `governance-coordinator`, `ibc-lead`, `incident-commander`,
  `release-verifier`, `supply-verifier`, and `validator-operator`, plus the
  strict validator-power threshold.

Every pair of mandatory roles on one of those edges must use different
identities and different public keys. The trust policy may impose stricter
requirements on any edge.

`incident-reviewer` is not a hard-coded role. If an operations policy needs
reviewer independence, provision `incident-reviewer` as an explicit signer
role, add it to `required_roles`, and separate it from
`incident-commander` (and any other required roles) on every intended incident
edge. The verifier accepts extensible role labels but enforces only what the
pinned edge policy and the built-in critical-edge rules require.

The policy must also preprovision the offline-only
`evidence-custodian` and `policy-rotation-authority` roles. Their identities
and keys cannot fill routine operational roles; they are reserved for signed
policy supersession.

## Validator-power snapshots

A power-gated transition carries
`zerone.ops.validator-power-snapshot/v1`:

```text
schema, chain_id, height, block_id_sha256, app_hash_sha256,
role, total_power, captured_at, valid_until,
members [{identity, public_key, power}],
snapshot_sha256
```

The snapshot must bind the transition's exact checkpoint, use the quorum role,
sum its sorted member powers exactly to `total_power`, and remain valid at the
transition's `occurred_at`. Every member tuple must already be a trusted signer
for that role. The transition must also include
`validator-power-snapshot` evidence whose SHA-256 matches the snapshot.

Obtain the snapshot digest through a separate trusted channel. Pass it to
`approval-statement` and `seal` as `--power-snapshot-sha256`, and to full
verification as
`--power-snapshot-pin <sequence>=<snapshot-sha256>`. An approval for the quorum
role must declare the exact power recorded for that member; all other approval
powers must be `"0"`. Quorum uses arbitrary-precision integer
cross-multiplication, including strict `>` semantics when `strict=true`.

The tool does not discover the live validator set. Capturing and independently
authenticating the checkpoint-bound snapshot remains an operator obligation.
A single signer with 1/1 power remains one correlated operator; this verifier
does not turn it into independent plurality.

## Sign and seal a transition

Each Ed25519 approval signs the raw 32-byte digest:

```text
sha256(
  "zerone.ops.approval/v1\x00" ||
  canonical JSON {
    transition: <transition with approvals=[] and transition_sha256="">,
    approval: {role, identity, public_key, power}
  }
)
```

Generate the digest for a signer tuple already present in the pinned policy:

```sh
zerone-ops approval-statement \
  --trust-policy /protected/zerone-release-policy.json \
  --trust-policy-sha256 <pinned-policy-sha256> \
  --input draft.json \
  --role release-verifier \
  --identity release-verifier-1 \
  --public-key <64-lowercase-hex> \
  --power 0
```

Sign the returned digest bytes with the matching Ed25519 private key. Add the
lowercase hex digest and 128-character lowercase hex signature to the
approval, sort approvals by role, identity, then public key, and run `seal`.
Both commands reject a signer tuple absent from the pinned policy.

## Signed trust-policy supersession

A v1 transition journal never changes `trust_policy_sha256`. If the policy or
a signer is compromised, freeze and preserve the old journal at its last
independently anchored head. Do not edit, append under an untrusted key, or
pretend the old journal was cleanly closed.

Use a separate `zerone.ops.supersession/v1` sidecar to bind:

```text
schema, chain_id,
old_journal_head_sha256,
old_trust_policy_sha256, new_trust_policy_sha256,
replacement_incident_id, replacement_release_id,
occurred_at, reason,
evidence [{type, sha256, uri}],
approvals [{role, identity, public_key, statement_sha256, signature, power}],
supersession_sha256
```

Exactly one of `replacement_incident_id` or `replacement_release_id` must be
set, selecting one replacement lane. A same-lane replacement must use a new
incident or release identifier; a new sequence-1 journal with the old identity
is not a continuation. The sidecar requires exactly one
`replacement-policy-ceremony` and exactly one
`trust-policy-compromise-assessment` evidence item. It requires exactly two
power-zero approvals from the old policy's distinct
`evidence-custodian` and `policy-rotation-authority` identities and keys. It
also binds canonical old and new policy files, their independently obtained
digests, and the independently obtained old journal head.

This path is available only while both preprovisioned offline roles remain
trusted and able to sign. If either authority is itself compromised, the tool
cannot manufacture a replacement trust root; recovery requires the separately
defined social/fork authority.

For each of those two roles, generate an approval digest:

```sh
zerone-ops supersession-approval-statement \
  --old-trust-policy /protected/old-policy.json \
  --old-trust-policy-sha256 <old-policy-sha256> \
  --new-trust-policy /protected/new-policy.json \
  --new-trust-policy-sha256 <new-policy-sha256> \
  --old-head-sha256 <old-journal-head-sha256> \
  --input supersession-draft.json \
  --role evidence-custodian \
  --identity evidence-custodian-1 \
  --public-key <64-lowercase-hex>
```

After adding both sorted approvals, seal and verify the sidecar:

```sh
zerone-ops seal-supersession \
  --old-trust-policy /protected/old-policy.json \
  --old-trust-policy-sha256 <old-policy-sha256> \
  --new-trust-policy /protected/new-policy.json \
  --new-trust-policy-sha256 <new-policy-sha256> \
  --old-head-sha256 <old-journal-head-sha256> \
  --input approved-supersession-draft.json > supersession.json

zerone-ops verify-supersession \
  --old-trust-policy /protected/old-policy.json \
  --old-trust-policy-sha256 <old-policy-sha256> \
  --new-trust-policy /protected/new-policy.json \
  --new-trust-policy-sha256 <new-policy-sha256> \
  --old-head-sha256 <old-journal-head-sha256> \
  --input supersession.json
```

Sidecar verification alone proves the policy-rotation decision; it does not
prove that the operator supplied the complete old or replacement journals, or
that the replacement journal actually acknowledges that decision. Use the
composed verifier before accepting a replacement history:

```sh
zerone-ops verify-replacement \
  --old-trust-policy /protected/old-policy.json \
  --old-trust-policy-sha256 <old-policy-sha256> \
  --new-trust-policy /protected/new-policy.json \
  --new-trust-policy-sha256 <new-policy-sha256> \
  --old-head-sha256 <independently-pinned-old-head> \
  --new-head-sha256 <independently-pinned-replacement-head> \
  --old-journal old-0001.json \
  --old-journal old-0002.json \
  --input supersession.json \
  --new-journal replacement-0001.json \
  --new-journal replacement-0002.json
```

Repeat the old or new power-snapshot pin flag for every power-gated transition:
`--old-power-snapshot-pin <sequence>=<sha256>` or
`--new-power-snapshot-pin <sequence>=<sha256>`.

The first replacement transition must contain exactly one signed evidence item
of each type:

```text
supersession-sidecar    sha256 = supersession_sha256
superseded-journal-head sha256 = old_journal_head_sha256
```

Those evidence fields are inside the transition approval statement, so they
cannot be attached after signing. The composed verifier checks both complete
journals against their external head pins, verifies the sidecar against the old
head and both policy digests, enforces the selected replacement lane and ID,
and requires:

```text
old head occurred_at <= supersession occurred_at
                       <= first replacement transition occurred_at
```

It also applies ordinary checkpoint continuity across the journal boundary:
the first replacement checkpoint cannot move below the old head, and the block
ID/AppHash cannot change at the same height. Trust-policy rotation is not fork
authority. A re-genesis or competing canonical history requires the separate,
explicitly signed fork-choice procedure rather than a supersession sidecar.

Treat `verify-replacement` as the operational acceptance command.
`verify-supersession` remains useful for inspecting the sidecar in isolation,
but is deliberately insufficient to authorize continuation.

Start the replacement as a new sequence-1 journal in exactly one lane under
the new pinned policy. Preserve the old journal, sidecar, new policy, and new
journal together so the authority change remains auditable.

## Transition fields

Each transition contains:

```text
schema, lane, sequence, from, to, event, occurred_at,
chain_id, incident_id, release_id,
actor_role, actor_identity,
checkpoint {height, block_id_sha256, app_hash_sha256},
release {
  plan_name, upgrade_height, activation_mode,
  plan_info_sha256, binary_sha256, image_sha256,
  provenance_sha256, sbom_sha256, state_manifest_sha256
},
power_snapshot {
  schema, chain_id, height, block_id_sha256, app_hash_sha256,
  role, total_power, captured_at, valid_until,
  members [{identity, public_key, power}], snapshot_sha256
},
evidence [{type, sha256, uri}],
approvals [{
  role, identity, public_key, statement_sha256, signature, power
}],
trust_policy_sha256,
previous_transition_sha256, transition_sha256
```

`occurred_at`, `captured_at`, and `valid_until` use canonical UTC RFC3339Nano
ending in `Z`. Evidence URIs may identify private storage; publish
content hashes, not secret forensic material.

## Verification boundary

A valid result proves deterministic structure, hash continuity, allowed
forward transitions, immutable identities and artifact bindings, binding to
the supplied policy and snapshot digests, authorization of signer tuples,
valid Ed25519 signatures, enforced role separation, and satisfaction of the
configured quorums. It does not prove:

- that source code or a signed binary is safe;
- that evidence content is true without independently retrieving its digest;
- that a pinned checkpoint or validator-power snapshot matches the live chain;
- that a consensus majority has not been captured;
- that `SAFETY_STOPPED`, `FORK_CHOICE`, or a checkpoint is socially accepted.

Those remain explicit human, validator, forensic, and governance decisions.
