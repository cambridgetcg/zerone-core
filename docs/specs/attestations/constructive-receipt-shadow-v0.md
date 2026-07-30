# Constructive receipt shadow bridge v0

Status: experimental, offline, source-only, zero-value

## Purpose

The constructive receipt shadow bridge tests one narrow composition boundary
between:

- the static constructive-intelligence tree v1; and
- a locally re-evaluated Proof of Constructive Adaptation (PoCA) v0 profile and
  evidence bundle.

The implementation is
[`tools/constructive-receipts`](../../../tools/constructive-receipts). It does
not accept a caller-built PoCA certificate. It parses the profile and evidence
with `poca-shadow`, runs the PoCA evaluator again, and binds the resulting exact
profile, evidence, claim, and subject digests.

The only supported tree target is
`protocol-software-supply-chain@2026q3`. The bridge creates either a
`CANDIDATE` or `REFUSED` shadow receipt. Neither status grants tree evidence,
qualification, certification, replay consumption, a payment authorization, or
an entitlement.

## Exact input bindings

`zerone.constructive-receipt-request/v0` pins:

- tree schema `zerone.constructive-intelligence-tree/v1`;
- policy version `1.0.0`;
- the canonical policy-object digest;
- the exact checked-in tree byte digest;
- the exact target node ID and canonical node digest;
- PoCA profile ID, version, canonical digest;
- PoCA evidence-bundle canonical digest;
- the exact PoCA subject digest; and
- a source system, record ID, and revision tuple.

The reviewed tree pins are:

| Binding | Digest |
|---|---|
| checked-in tree bytes | `sha256:8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf` |
| canonical `policy` object | `sha256:36116220c7f17dd06f8bda2217d79a000aaac771075a709004686b233402abc7` |
| canonical target node | `sha256:862f9f2d70b51dd03e6fc761eb91487e38c83064414b77d11548ead110356104` |

The document digest hashes the exact local file bytes, including whitespace
and its final newline. The policy and node digests decode JSON, encode compact
JSON with Go `encoding/json` key ordering and escaping, add no trailing
newline, and hash those bytes. The bridge does not replace the tree's
JavaScript semantic validator; it only accepts the exact reviewed tree
document that passed it.

PoCA profile and evidence digests retain the normalization defined in
[Proof of Constructive Adaptation v0](proof-constructive-adaptation-v0.md).
Reordering PoCA set-like arrays therefore does not change the receipt. Changing
the tree file's byte representation does change its document digest and is
refused.

## Candidate predicate

A `CANDIDATE` requires all of the following:

1. the PoCA profile says `DECLARED_RATIFIED`;
2. there are no unresolved PoCA challenge digests;
3. the profile contains exact bindings for:
   - SLSA v1.2, Build L2;
   - in-toto Attestation v1.2.0, Statement layer; and
   - the pinned Sigstore bundle v0.3 specification, with a policy target that
     names signature, identity, trust-root, time, and inclusion checks;
4. PoCA node `standard.slsa-build-l2` is `DECLARED_PASS`; and
5. PoCA node `capability.independent-rebuild` is `DECLARED_PASS`.

These are local structural and declared-evidence checks. Profile lifecycle,
participant roles, control clusters, verification results, policy authority,
and receipt content remain declarations. The bridge does not authenticate a
Sigstore bundle, resolve a trusted root, check Fulcio identity, verify Rekor
inclusion, or decide the truth of an external predicate.

`CANDIDATE` means only “eligible for a separate tree-evidence review under this
fixed bridge predicate.” The output always says:

```text
tree.granted_attainment_evidence = NONE
qualification = NONE
economic_effect = NONE
amount_uzrn = "0"
```

## Evidence namespace wall

PoCA and the constructive tree deliberately use different evidence models:

- PoCA tiers use namespace `zerone.poca-shadow-evidence/v0`;
- tree attainment uses namespace
  `zerone.constructive-tree-evidence/v1`; and
- every receipt fixes `namespace_relation = NO_EQUIVALENCE`.

PoCA `E2_CONFORMANT` is not tree `E2`. PoCA `E3_CAUSAL` is not tree `E3`.
Names, numbers, or ordering cannot be copied between the namespaces. The
bridge examines the semantic status of specifically named PoCA nodes and can
only produce a candidate for later review; it never creates a tree attainment
record.

## Source consumption key

The deterministic source key is:

```text
SHA256(
  "zerone.constructive-source-consumption/v0" || NUL ||
  source_system || NUL ||
  record_id || NUL ||
  revision
)
```

The receipt ID is:

```text
SHA256(
  "zerone.constructive-receipt/v0" || NUL ||
  bridge_version || NUL ||
  tree_document_digest || NUL ||
  tree_policy_digest || NUL ||
  target_node_digest || NUL ||
  target_node_id || NUL ||
  poca_profile_digest || NUL ||
  poca_evidence_bundle_digest || NUL ||
  subject_digest || NUL ||
  source_consumption_key
)
```

The source tuple is caller-declared. The output fixes
`consumption_state = NOT_RECORDED` and
`replay_protection = NONE_OFFLINE`. The key is suitable for testing the shape
of a future global consumption index, but this tool stores no state, cannot
detect prior use, and does not make the tuple unique, authentic, unconsumed, or
rewardable.

## Zero escrow

Every output embeds
`zerone.zero-escrow-compartments/v0`. Funded escrow, verified costs, claimant
milestones, challenge/remediation reserve, reviewers, administration/fees,
and refundable balance are all the exact string `"0"` in `uzrn`.
`conservation_check = ZERO_BALANCED` describes only this all-zero identity.
There is no account, funding source, lock, transfer, vesting schedule, or
refund operation.

The executable fixture is
[`zero-escrow-compartments-v0.json`](../../examples/constructive-receipts/zero-escrow-compartments-v0.json).

## Published refusal fixture

Run:

```bash
go run ./tools/constructive-receipts \
  --request docs/examples/constructive-receipts/zerone-release-partial-v0.request.json \
  --tree dashboard/public/standards/constructive-intelligence-tree.v1.json \
  --profile docs/examples/poca/slsa-build-l2-v0.profile.json \
  --evidence docs/examples/poca/zerone-release-partial-v0.evidence.json
```

The published synthetic PoCA input reaches PoCA `E2_CONFORMANT`, but its
profile is `DRAFT` and does not pin the target node's Sigstore bundle policy.
The bridge therefore emits `REFUSED`, grants no tree evidence, and identifies
both insufficiencies.

Known-answer values:

| Value | Known answer |
|---|---|
| source consumption key | `sha256:8df1821e9ddded60ea67920eafb2e08cd4e0048bd1fc7b979510818a0c51b5a3` |
| receipt ID | `sha256:54cc3e31a69cba6892095c175d62383c27466b23a2f107b6c448db9eaaccfc8a` |
| compact receipt JSON SHA-256 | `42cfb6549a23b4628fc1af706cd4b1472768d8ed7023e1ee2920f63238459e61` |

The request, refusal receipt, and zero-escrow documents use synthetic local
source metadata. They are test vectors, not release evidence.

## Parsing and operational boundary

The CLI reads regular files only and rejects symlinks. Files are bounded to
1 MiB before parsing; request parsing additionally enforces 64 KiB and tree
parsing enforces the tree v1 256 KiB limit. The request rejects duplicate
object keys, unknown fields, JSON `null`, omissions, malformed identifiers,
and malformed lowercase SHA-256 digests. PoCA inputs retain their own strict
duplicate, unknown, null, omission, size, URL, graph, and cross-document
checks.

The bridge is deterministic and has no clock or network. Consequently it does
not establish that the tree standards snapshot is fresh for active use, that
an external source remains available, or that a mutable authority still gives
the same status. A real admission path must independently revalidate freshness
and authority under a pinned observation policy.

No consensus module, store, protobuf, transaction, dashboard state, genesis
field, qualification projection, reward path, signature workflow, or
deployment is part of v0.

## Machine-readable schemas

- [`request.schema.json`](constructive-receipt-shadow-v0/request.schema.json)
- [`receipt.schema.json`](constructive-receipt-shadow-v0/receipt.schema.json)
- [`zero-escrow-compartments.schema.json`](constructive-receipt-shadow-v0/zero-escrow-compartments.schema.json)

JSON Schema defines document shape. The Go implementation remains the
executable reference for exact local tree bytes, canonical digests, PoCA
re-evaluation, candidate refusal semantics, source-key derivation, namespace
separation, and all-zero invariants.
