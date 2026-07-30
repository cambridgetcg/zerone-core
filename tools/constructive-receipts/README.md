# constructive-receipts

This directory contains two independent offline, zero-value experiments. They
share the canonical constructive-intelligence tree but neither feeds the
other, records consumption globally, grants qualification, creates economic
effect, or writes to a network.

## PoCA tree-v1 bridge

The Go command is a tree-v1 ↔ PoCA receipt bridge. It re-evaluates local PoCA
profile/evidence bytes and accepts only the exact reviewed
constructive-intelligence tree v1 document and
`protocol-software-supply-chain@2026q3` node.

Run the published honest refusal:

```bash
go run ./tools/constructive-receipts \
  --request docs/examples/constructive-receipts/zerone-release-partial-v0.request.json \
  --tree dashboard/public/standards/constructive-intelligence-tree.v1.json \
  --profile docs/examples/poca/slsa-build-l2-v0.profile.json \
  --evidence docs/examples/poca/zerone-release-partial-v0.evidence.json
```

The result is `REFUSED`: the PoCA profile is `DRAFT` and lacks the exact
Sigstore policy binding required by this bridge. PoCA `E2_CONFORMANT` is not
tree `E2` or tree `E3`.

Every possible output retains:

```text
assurance = UNVERIFIED_SHADOW_PROJECTION
tree.granted_attainment_evidence = NONE
qualification = NONE
economic_effect = NONE
amount_uzrn = "0"
consumption_state = NOT_RECORDED
replay_protection = NONE_OFFLINE
```

The consumption key is deterministic, but this command is not a replay ledger
or entitlement service. It performs no network request, Sigstore
authentication, chain read/write, qualification update, escrow action, or
reward calculation.

See the
[constructive receipt shadow v0 specification](../../docs/specs/attestations/constructive-receipt-shadow-v0.md)
for the candidate predicate, digest algorithms, schemas, known-answer vectors,
and limitations.

## Typed bundle validator

The JavaScript command is a dependency-free, offline validator for typed
constructive-intelligence evidence receipts. It is a deliberately
non-authoritative `v0` adapter around the canonical constructive-intelligence
tree.

The adapter can prove that a receipt bundle has the expected shape and exact
digest bindings. It cannot prove that a mathematical or security claim is
true, that an organization is independent, that a source record is honest, or
that a target was actually authorized.

Most importantly, a successful decision:

- does not grant a qualification;
- does not accept evidence into Zerone;
- does not activate a reward;
- does not instantiate an escrow schedule;
- does not move funds;
- does not authorize security testing or integration; and
- performs no network requests or writes.

Every authority, network, reward, qualification, fund, and integration
boundary stays explicitly `false`. Every value field stays exactly zero or
`null`. The tree's E0-E6 basis-point schedule is returned only as
`metadataOnly: true`.

### Run

From the repository root:

```bash
node tools/constructive-receipts/validate.mjs \
  --tree dashboard/public/standards/constructive-intelligence-tree.v1.json \
  --bundle tools/constructive-receipts/testdata/valid-tls-e3-bundle.v0.json
```

Exit codes are:

- `0`: the bundle is structurally valid;
- `1`: input, binding, safety, independence, or policy validation failed;
- `2`: command-line usage is invalid.

The bundle supplies `checkedForUseAt` as an ISO date. The validator never reads
the wall clock. It asks the canonical tree validator to check the whole
standards snapshot for active use on that exact date, so an expired
`reviewAfter` fails closed.

Run the focused suite with:

```bash
node --test tools/constructive-receipts/validate.test.mjs
```

### Canonical identities

Objects use recursively key-sorted canonical JSON. Arrays retain their
declared order, and schema arrays that represent sets must already be sorted
and duplicate-free.

```text
policy_revision_digest =
  SHA-256(canonical_json(policy_revision))

deliverable_key =
  SHA-256(canonical_json([
    exact_standard_pins,
    scope_hash,
    acceptance_policy_digest,
    canonical_subject_roots
  ]))

consumption_key =
  SHA-256(canonical_json([
    source_system,
    source_record_or_event_id,
    source_revision
  ]))

evidence_id =
  SHA-256(canonical_json(receipt_without_evidence_id))
```

`deliverable_key` prevents a new wrapper, address, explanation, or bounty ID
from resetting the identity of the same bounded work. A derivative must name a
different prior deliverable, disclose an overlap digest, use a canonical
artifact edge, and assert an independently reviewed delta.

`consumption_key` is independent of the issuer-selected `evidence_id`.
Validation rejects a repeated source tuple within one bundle and returns the
sorted keys for every submitted revision in its decision. Superseded receipts
remain in that replay output, but are removed from all active evidence,
artifact, checker, and independence calculations. `receiptCount` is the active
count; `submittedReceiptCount` and `supersededReceiptCount` make that reduction
explicit. A caller that operates across bundles must keep an append-only global
consumption ledger; this offline validator intentionally does not persist one.

### Structural gates

The validator rejects:

- malformed, duplicate-key, unknown-key, oversized, or excessively nested
  JSON;
- any drift in the tree document, normative tree, quest node, scope,
  acceptance policy, standards snapshot, or exact standard pins;
- any non-quest or non-sponsor-milestone target;
- stale standards snapshots at the bundle's explicit active-use date;
- policy, deliverable, receipt, method, source, or subject-root mismatches;
- claimant, sponsor, evaluator, or payout-authorizer role overlap;
- replayed source tuples or duplicate evidence IDs;
- incomplete or outcome-contingent conflict disclosures;
- planted/retained defects, claimant-controlled adoption, self-created
  break/fix loops, or concealed causal involvement;
- unauthorized targets, failed safety gates, incomplete prepublication
  triage, public exploit plaintext, publication of confidential evidence,
  vendor payout vetoes, protocol-security assertions, or network requests;
- unknown security impact outside private coordinated repair with escalation;
- E5 adoption without an allowed receipt type and an adopter independent of
  the claimant, sponsor, payout authorizer, every declared evaluator, the
  receipt's organization and implementation roots, and every disclosed
  related control root;
- non-E5 evidence carrying an adoption claim;
- forward, external, self, forked, backdated, or equal-time supersession;
- E0, E1, E2, E4, or E6 receipt claims, whose level-specific predicates are
  not represented by this minimal v0 schema;
- timestamps before the tree snapshot or after `checkedForUseAt`; and
- any denomination, cap, budget, escrow receipt, schedule, expiry, or refund
  path in this zero-value profile.

Authorship, prior review, and knowing preservation are required disclosure
facts, but are not automatically treated as wrongdoing. The explicitly
ineligible break/fix and concealment flags are hard failures.

### Independence

Receipts are connected into one effective control component when their
declared related-control-root sets intersect, including transitively, or when
they name the same verifier cluster. Quality is integer millionths from zero
through one million.

```text
effective_quality =
  sum(min(1_000_000, quality_inside_each_control_component))
```

The bundle must supply enough effective quality for the strictest cluster
floor from the global tree policy, quest acceptance object, and selected
coverage target. A diversity value counts only when the receipts that claim it
carry at least one full quality-unit after applying the same per-control-
component cap:

```text
support(value) =
  sum(min(1_000_000, quality_for_value_inside_each_control_component))

value_counts_only_if support(value) >= 1_000_000
```

This rule applies separately to organization/control roots,
implementation/toolchain roots, execution environments, and each case ID. A
zero- or near-zero-quality decorator receipt therefore cannot pad a diversity
floor. Claimant and sponsor roots never count toward technical quorum. A
required checker or corpus digest must be identical across the active bundle.

These checks only evaluate declared identifiers. They do not discover hidden
common ownership or correlated implementations.

### Canonical TLS E3 fixture

The included fixture binds
`quest-tls-rfc9846-keyshare-reuse@1` on `2026-07-30`:

- three full-quality, unrelated evaluator control clusters;
- two organization/control roots;
- three implementation/toolchain roots;
- two execution-environment digests;
- twelve distinct cases; and
- one checker/corpus digest.

Receipt schema v0 recognizes only E3 reproduction and E5 adoption claims.
Other E0-E6 treatments remain visible as non-operative tree metadata until
typed predicates and transition history exist for them.

The fixture is an E3 reproduction example, not an E5 adoption claim. Its policy lives
both as the embedded immutable revision and as
`testdata/zero-value-tls-policy.v0.json`; the tests require the two objects to
remain byte-semantically identical. The fixture uses transparent synthetic
digests and identities for validator testing, not evidence about a live
system.
