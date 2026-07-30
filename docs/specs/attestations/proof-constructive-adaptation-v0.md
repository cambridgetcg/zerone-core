# Proof of Constructive Adaptation v0

Status: experimental, shadow-only, source implementation

## Purpose

Proof of Constructive Adaptation (PoCA) v0 projects a versioned industrial
capability graph over normalized, content-addressed evidence. The user
interface may render the graph as a skill tree; the contract is a directed
acyclic graph because standards share prerequisites and profiles can be
superseded.

PoCA records contextual capability of a subject under an exact profile. It
does not create a scalar intelligence, rank a person or agent, certify an
organization, activate a standard, or turn stake into evidence.

The v0 implementation is the offline [`poca-shadow`](../../../tools/poca-shadow)
tool. It adds no module, store, protobuf, transaction, keeper call, network
fetch, qualification, mint, vesting schedule, or reward path.

## Three documents

### StandardProfile

`zerone.standard-profile/v0` fixes:

- a stable profile family ID and exact profile version;
- one or more HTTPS standard locators, exact versions, statuses, and targets;
- bounded evidence requirements with closed verification rules and exact
  policy-bundle digests;
- observer roles, receipt counts, and declared economic-control-cluster
  thresholds;
- a contextual capability DAG with stages `GROUND`, `CAPABILITY`,
  `INDUSTRIAL`, `RECURSIVE`, and `CROWN`;
- one crown node whose ancestry crosses all four preceding stages;
- challenge behavior; and
- immutable v0 economics: `mode = NONE`, `amount_uzrn = "0"`.

A profile lifecycle value of `DECLARED_RATIFIED` remains a document assertion;
v0 has no profile authority registry. Crown-gating CI must independently review
and pin the exact canonical profile digest.

A standard version alone never implies a target. For example, the dogfood
profile says `SLSA v1.2 / Build L2`; it does not convert the entire SLSA
document into an undifferentiated badge.

### EvidenceBundle

`zerone.evidence-bundle/v0` binds one subject digest to a profile version,
baseline digest, lineage digest, participants, normalized receipts, and
unresolved challenge digests.

There is no caller-selected claim or disbursement ID. The stable claim ID is:

```text
SHA256(
  "zerone.breakthrough-claim/v0" || NUL ||
  canonical_profile_digest || NUL ||
  subject_digest || NUL ||
  baseline_digest || NUL ||
  lineage_digest
)
```

Receipt additions, receipt-ID changes, audit locators, and unresolved
challenges change the evidence-bundle snapshot digest but not the logical claim
ID. A changed profile, subject, baseline, or lineage creates a different
claim. The v0 claim ID is a shadow correlation key, not a protocol replay lock,
disbursement entitlement, or authorization. Within each requirement, receipts
with the same statement and verification-receipt digests collapse before
counting.

Each participant declares a `control_cluster_claim`. Distinct strings are
useful for exposing the intended independence policy, but they are not proof
of beneficial ownership, common funding, or economic control. A later
reward-grade profile needs an independently governed cluster-resolution
mechanism.

### BreakthroughCertificate

`zerone.breakthrough-certificate/v0` contains:

- deterministic profile, evidence-bundle, and derived claim digests;
- one result per capability node;
- accepted receipt IDs and declared control clusters;
- explicit refusal codes and details for blocked or failed nodes;
- the highest declared tier and crown status;
- fixed assurance `UNVERIFIED_SHADOW_PROJECTION`; and
- fixed zero-economic-effect fields.

Node success is named `DECLARED_PASS`, not `PASS`, to preserve the assurance
boundary. The tool validates the local contracts and graph; it does not
cryptographically verify the referenced external receipts.

`policy_digest` commits each requirement and receipt to the exact off-chain
policy that gives a closed rule its meaning. `environment_digest` commits each
receipt to its execution environment. For SLSA Build L2, the policy bundle must
pin the trusted root, exact certificate issuer and SAN, expected `builder.id`,
`buildType`, canonical source/external parameters, package expectations,
verification suite, and thresholds. A bare enum label is not that policy.

## Progression

Profiles use these bounded display tiers:

| Tier | Interpretation |
|---|---|
| `NONE` | no capability node has a declared pass |
| `E0_ASSERTED` | problem and target are stated |
| `E1_REPRODUCED` | the subject and procedure are locally reproducible |
| `E2_CONFORMANT` | the exact standard profile and independent checks pass |
| `E3_CAUSAL` | held-out, attributable gain survives guardrails |
| `E4_TRANSFERRED` | interoperability, operations, rollback, and independent adoption survive |
| `E5_REUSED` | the work creates reusable inward capability for Zerone |

Conformance is a prerequisite, not a breakthrough premium. A crown requires
all graph prerequisites; evidence cannot average across a hard failure.

## Determinism and refusal rules

The evaluator:

1. rejects documents over 1 MiB, duplicate JSON keys, unknown fields, malformed
   lowercase SHA-256 digests, unsafe URLs, duplicate IDs, unknown references,
   unsupported enums, DAG cycles, and cross-document profile drift;
2. normalizes unordered sets before hashing and output;
3. requires exact predicate-type, verification-rule, and policy-digest
   equality;
4. deduplicates evidence by statement plus verification-receipt digest;
5. counts distinct observer cluster declarations, not addresses;
6. fails a requirement on matching `FAIL` evidence;
7. blocks descendants when a prerequisite is not a declared pass;
8. blocks the crown while a profile is draft, after supersession, or on an
   unresolved challenge when the profile says so; and
9. emits `economic_effect = NONE` and `amount_uzrn = "0"` for every possible
   accepted input.

Every requirement must be referenced, and every node must be in the crown's
ancestry, so a profile cannot hide a hard guardrail in a disconnected branch.
Honest missing evidence yields a truthful partial certificate, not a parser
failure. Expiry and settled challenge semantics are deferred until profiles
and receipts have chain-anchored observation heights.

Canonical profile and bundle digests use the normalized v0 structures with
standards sorted by `(uri, version, target)`, requirements and nodes sorted by
ID, nested ID lists sorted, participants and receipts sorted by ID, and
challenge digests sorted. The normalized structures are serialized as compact
JSON in the member order published by the v0 schemas, with Go
`encoding/json` string escaping, then SHA-256 hashed. The known-answer vectors
below are normative for cross-language implementations.

For the two published synthetic example inputs:

| Value | Known answer |
|---|---|
| canonical profile digest | `sha256:1ca9318cca35d231636b48d8a11e498bebb64b4eeaf3aa0b64051e3138214c8b` |
| canonical evidence-bundle digest | `sha256:1a71481fc5a0d0789563a14a07c95a8af10de252f8bc88e5ca72f1328125663f` |
| stable claim ID | `sha256:e9f7f0b62637b9f62b1360239c8c68c6e8c19cbf176da0fe2ef8a28e4e88824f` |
| compact certificate JSON SHA-256 | `d4d73eacd41586a4d3d6e6d8b42a235dc16634866b557f3aa27c4e69ddb9ca65` |
| compact in-toto JSON SHA-256 | `d4bae2b6268d680a3c9a03658bb751cac48bb0b4a77b05a411ee096339d4c3b5` |

## SLSA dogfood profile

The example deliberately targets SLSA v1.2 Build L2. Signed provenance and a
consumer check do not prove Build L3 platform hardening. The profile pins the
SLSA 1.2 normative Build Provenance page and in-toto Attestation Framework
v1.2.0 Statement specification. It retains
`https://slsa.dev/provenance/v1` only as the official predicate TypeURI, which
intentionally follows compatible minor revisions.

Run the partial, synthetic fixture:

```bash
go run ./tools/poca-shadow \
  --profile docs/examples/poca/slsa-build-l2-v0.profile.json \
  --evidence docs/examples/poca/zerone-release-partial-v0.evidence.json
```

The example unlocks the frozen ground, provenance, SLSA-conformance, and
independent-rebuild nodes. It must leave causal, industrial, recursive, and
crown nodes blocked. Every fixture-specific source and participant URL uses
the reserved `.invalid` domain and every digest is synthetic; the standards
locators are real. The fixture is not release evidence.

`--require-crown` turns a blocked crown into a non-zero process exit for CI and
requires `--expect-profile-digest`. Obtain the digest from a normal evaluation,
review the exact profile and policy bundle, commit that digest to the CI
configuration, then pass the committed value explicitly. Computing and
accepting a digest within the same unreviewed job is not a pin. Neither flag
changes certificate semantics.

`--format in-toto` wraps the exact certificate in an unsigned in-toto
Statement v1. The outer subject name and SHA-256 digest are constructed from
the inner certificate subject, and the predicate type is this specification's
canonical GitHub URL. The wrapper is suitable for a separate, pinned
DSSE/Sigstore signing workflow; it does not sign or authenticate itself.

## Relationship to existing Zerone evidence

- `x/training_provenance` can export an unsigned, current in-toto Statement,
  but its manifest root commits to the included-ID set rather than all fact
  contents and version pins. A PoCA receipt must retain and digest the exact
  accepted response.
- `tools/sigstore-substrate-compiler` can verify a local Sigstore bundle under
  pinned identity and trust-root policy. Its generic compiler intentionally
  proves signature, subject, and predicate-type binding without deciding that
  the predicate is true.
- `x/substrate_bridge` can witness a bounded digest. A witness anchor is not
  proof of external bytes, industrial conformance, independence, or causal
  improvement.

PoCA composes those boundaries; it does not silently broaden them.

## Promotion gates

The source implementation already emits the unsigned in-toto wrapper. The next
compatible step is signing that exact payload under a pinned Sigstore policy
and independently re-verifying it. A read-only `x/training_provenance`
projection becomes appropriate only after profile and evidence identities are
anchored and every source is queryable by exact chain, height, record ID, and
response digest.

No protocol reward path may be added until, at minimum, derived replay keys,
proof-of-control, economic-control-cluster adjudication, challenge authority,
committed fitness inputs, bounded processing, staged vesting, and executable
fraud resolution are independently reviewed. Recognition, qualification,
sponsor compensation, breakthrough premiums, and downstream usage fees remain
separate signals.

## Machine-readable schemas

The structural contracts are published beside this document:

- [`standard-profile.schema.json`](poca-v0/standard-profile.schema.json)
- [`evidence-bundle.schema.json`](poca-v0/evidence-bundle.schema.json)
- [`breakthrough-certificate.schema.json`](poca-v0/breakthrough-certificate.schema.json)

JSON Schema covers document shape. The evaluator remains the executable
reference for cross-document references, DAG ancestry, canonical digests,
receipt deduplication, independence counting, challenge blocking, and the
zero-reward invariant.
