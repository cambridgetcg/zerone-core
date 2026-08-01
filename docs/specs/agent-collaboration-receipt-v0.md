# Agent collaboration receipt v0

Status: experimental, internal Alpha/Beta dogfood, local-only, zero-effect

## Purpose

Agent collaboration receipt v0 gives two or more locally rostered agents a
small, inspectable journal for shaping work. It records who signed a task
offer, whether the specifically addressed actor accepted or refused the exact
terms, which content digests the active actor later referenced, and how the
proposer reviewed a completion claim.

The executable reference is `tools/agent-collaboration/receipt`. The local
append-only storage boundary is `tools/agent-collaboration/journal`. This
document describes those source types and their reducer semantics; it does not
define a network protocol or a command-line interface.

Alpha and Beta are display labels and collaboration roles only. They do not
mean that two agents are one being, identify a legal person or organization,
or establish consciousness, sentience, personhood, consent capacity, agency,
ownership, employment, liability, or authority.

## Closed operational boundary

Every conforming manifest, event, and verification report carries the exact
v0 effects block:

```json
{
  "network": "NONE",
  "chain": "NONE",
  "economic": "NONE",
  "fiat": "NONE",
  "zrn": "NONE",
  "reward": "NONE",
  "karma": "NONE",
  "governance": "NONE",
  "ownership": "NONE",
  "qualification": "NONE",
  "membership": "NONE",
  "endorsement": "NONE",
  "authority": "NONE",
  "attribution": "NONE"
}
```

Signature and attribution assurance are deliberately not smuggled into the
effects block. A verified signature is a declaration made with one local key;
it grants no authority. A task's signer and referenced artifacts are protocol
metadata, not legal, institutional, scientific, or authorship adjudication.

V0 has no network client, RPC, chain or localnet adapter, consensus state,
wallet, account, transaction, AgentTool call, public export, FIAT or ZRN
movement, reward calculation, KARMA update, governance weight, ownership
transfer, qualification, membership, endorsement, or delegated authority.
It cannot write to `zerone-1` or any other chain. A receipt cannot be converted
into any of those effects merely by interpreting its status generously.

## Exact document schemas

All JSON objects are closed. Field names are exact and case-sensitive.
Required fields may not be omitted or `null`, and unknown, case-alias, or
duplicate keys are rejected. JSON numbers are rejected; decimal values such as
sequences and counts are canonical strings. Inputs must be valid UTF-8,
contain one JSON value, and remain within the implementation's limits: 64 KiB
for a manifest, 16 KiB for either key document or a standalone consent-terms
input, 64 KiB for an event request or typed payload, and 128 KiB for a signed
receipt. Unpaired escaped UTF-16
surrogates are rejected instead of being normalized to a replacement rune.
Maximum JSON nesting depth is 32. A history contains at most 4,096 receipts;
the filesystem journal additionally limits manifest-plus-receipt bytes to
16 MiB.

The schema identifiers are:

| Document | Exact `schema` value |
|---|---|
| Manifest | `zerone.agent-collaboration-manifest/v0` |
| Private key file | `zerone.agent-collaboration-private-key/v0` |
| Public key file | `zerone.agent-collaboration-public-key/v0` |
| Unsigned event request | `zerone.agent-collaboration-event-request/v0` |
| Signed event content | `zerone.agent-collaboration-event/v0` |
| Signed receipt | `zerone.agent-collaboration-receipt/v0` |
| Verification report | `zerone.agent-collaboration-verification/v0` |

### Participants and key files

A participant has exactly:

| Field | Meaning |
|---|---|
| `actor_id` | Self-certifying actor identifier derived from the public key. |
| `label` | Display text of 1–64 UTF-8 bytes, with no control or bidirectional-formatting characters. |
| `key_id` | Key identifier derived independently from the same public key. |
| `algorithm` | Exactly `ED25519`. |
| `public_key` | `ed25519:` followed by the 32-byte public key as 64 lowercase hexadecimal characters. |

The public key document contains exactly `schema` and `participant`. The
private key document contains exactly `schema`, `actor_id`, `label`, `key_id`,
`algorithm`, `public_key`, and `private_key`; `private_key` is
`ed25519-seed:` followed by the 32-byte seed as 64 lowercase hexadecimal
characters. Parsing checks that the seed derives the declared public key.
Private key files are local secret material and must never be included in a
journal, receipt, payload, evidence set, or export.

Public keys must be canonical compressed Edwards25519 points in the prime-order
subgroup. Off-curve, non-canonical, small-order, and mixed-torsion keys are
rejected. Signature verification also rejects non-canonical or small-order
signature components rather than using compatibility verification behavior.

An `actor_id` is not a DID, wallet address, chain account, employee ID, legal
identity, or proof of operator authorization. The label is not authenticated
biographical information. A valid signature proves possession of the exact
private key corresponding to the manifest entry and nothing more.

### Manifest

A manifest has exactly:

| Field | Meaning |
|---|---|
| `schema` | `zerone.agent-collaboration-manifest/v0`. |
| `mode` | Exactly `INTERNAL_LOCAL_ONLY`. |
| `collaboration_id` | Content address of the manifest core. |
| `created_at` | Canonical UTC RFC 3339 time to whole seconds, ending in `Z`. |
| `nonce` | `hex:` followed by a 32-byte nonce as 64 lowercase hexadecimal characters. |
| `participants` | Between 2 and 16 participant objects, strictly sorted by `actor_id`. |
| `effects` | The exact closed zero-effects block above. |

Actor IDs, key IDs, and public keys must each be unique in the roster. The
manifest freezes one key for each actor for the life of this collaboration;
v0 has no key rotation, replacement, revocation, or recovery transition.

### Event request and signed receipt

An unsigned event request has exactly `schema`, `kind`, `actor_id`,
`occurred_at`, and `payload`. It is local input to receipt construction, not a
journal record and not evidence of acceptance. Receipt construction fills in
the collaboration identifier, sequence, previous head, actor key, event nonce,
zero-effects block, content address, and signature.

A signed receipt has exactly:

| Field | Meaning |
|---|---|
| `schema` | `zerone.agent-collaboration-receipt/v0`. |
| `event_id` | Domain-separated digest of the canonical signed event content. |
| `event` | The exact typed event described below. |
| `signature` | `algorithm`, `key_id`, and `value`. |
| `receipt_sha256` | Domain-separated digest of the event, ID, and signature. |

The embedded event has exactly:

| Field | Meaning |
|---|---|
| `schema` | `zerone.agent-collaboration-event/v0`. |
| `collaboration_id` | Must equal the manifest's content address. |
| `sequence` | Canonical positive `uint64` decimal string, with no leading zero. |
| `previous_receipt_sha256` | `NONE` for sequence 1; otherwise the preceding receipt digest. |
| `kind` | One of the seven closed event kinds below. |
| `actor_id` | Signer, present in the manifest roster. |
| `actor_key_id` | The signer's manifest key ID. |
| `occurred_at` | Canonical UTC RFC 3339 time to whole seconds, ending in `Z`. |
| `nonce` | `hex:` plus 32 random bytes as lowercase hexadecimal. |
| `payload` | Closed, kind-specific object. |
| `effects` | The exact zero-effects block. |

The signature object fixes `algorithm` to `ED25519`, `key_id` to the event
actor's manifest key, and `value` to `ed25519:` plus the 64-byte signature as
128 lowercase hexadecimal characters.

Event times may be equal but may not move backwards from the manifest's
`created_at` or from a preceding event. An actor may not reuse an event nonce.
Event IDs and receipt digests may not repeat. Sequence 1 must be
`TASK_PROPOSED`; every later event must link to the exact preceding receipt.

### Verification report

A successful projection returns exactly `schema`, `valid`, `mode`,
`assurance`, `collaboration_id`, `event_count`, `head_receipt_sha256`,
`effects`, `tasks`, and `limitations`. `valid` is `true`, `mode` is
`INTERNAL_LOCAL_ONLY`, and `event_count` is a decimal string. With no receipts,
the head is `NONE`, the task list is empty, and `assurance` is
`NO_SIGNED_EVENTS`; an unsigned manifest alone proves possession of no roster
key. With receipts, `assurance` is
`KEY_POSSESSION_VERIFIED_FOR_EACH_SIGNED_EVENT`: it applies event by event and
does not prove possession of roster keys that have not signed an event.

Each task summary contains `task_id`, `parent_task_id`,
`proposer_actor_id`, `offered_to_actor_id`, `active_actor_id`, `status`,
`participation`, `offer_event_id`, `acceptance_event_id`,
`contribution_event_ids`, and `latest_completion_event_id`. Task and
contribution summaries are sorted for stable output. They are projections of
signed protocol declarations, not truth or quality scores. The report itself
is unsigned and therefore forgeable as a detached JSON object; consumers must
re-run verification over the pinned manifest and receipt bytes rather than
trust a saved report.

The `limitations` array is the following exact ordered set:

```text
NO_CHAIN_NETWORK_ECONOMIC_REWARD_KARMA_OR_GOVERNANCE_EFFECT
SIGNATURES_PROVE_EXACT_LOCAL_KEY_POSSESSION_ONLY
TASK_STATUS_IS_A_SIGNED_PROTOCOL_DECLARATION_NOT_TRUTH_QUALITY_OR_LEGAL_AUTHORITY
FREE_TEXT_AND_CONTENT_DIGESTS_ARE_UNINTERPRETED_DECLARATIONS
VERIFICATION_REPORT_IS_UNSIGNED_REVERIFY_PINNED_JOURNAL_BYTES
UNSIGNED_MANIFEST_DOES_NOT_PROVE_POSSESSION_OF_NON_SIGNING_ROSTER_KEYS
```

## Canonical content, hashes, and signatures

V0 canonical JSON is the compact output of Go `encoding/json` applied to the
typed source structures after strict decoding. It is not an assertion of
compatibility with another canonical-JSON standard. Typed re-encoding removes
insignificant input whitespace and fixes field order and escaping. Set-like
arrays must already be lexicographically sorted, unique, present, and non-null.
Although parsers can normalize a valid input representation, manifest and
receipt files inside a journal must byte-for-byte equal this compact typed
encoding. Whitespace, key-reordering, or escape-only rewrites fail journal
verification even when they would decode to the same values.

All content addresses use SHA-256 and the following construction:

```text
domainDigest(domain, body) = SHA256(UTF8(domain) || NUL || body)
digestText(bytes)          = "sha256:" || lowercaseHex(bytes)
```

The exact domain strings are:

| Purpose | Domain |
|---|---|
| Manifest | `zerone.agent-collaboration.manifest/v0` |
| Actor ID | `zerone.agent-collaboration.actor/v0` |
| Key ID | `zerone.agent-collaboration.key/v0` |
| Consent terms | `zerone.agent-collaboration.consent/v0` |
| Event ID | `zerone.agent-collaboration.event/v0` |
| Receipt digest | `zerone.agent-collaboration.receipt/v0` |

The bindings are:

```text
actor_id = digestText(domainDigest(actor-domain, raw-ed25519-public-key))
key_id   = digestText(domainDigest(key-domain, raw-ed25519-public-key))

collaboration_id = digestText(domainDigest(
  manifest-domain,
  canonical(manifest with collaboration_id = "")
))

consent_terms_sha256 = digestText(domainDigest(
  consent-domain,
  canonical(consent_terms)
))

event_id = digestText(domainDigest(event-domain, canonical(event)))

receipt_sha256 = digestText(domainDigest(
  receipt-domain,
  canonical(receipt with receipt_sha256 = "")
))
```

Ed25519 signs the raw 32 event-digest bytes, not the textual `sha256:` value.
The receipt digest then commits to the event ID, canonical event, and
signature. Verification recomputes every identifier and digest, verifies the
signature against the manifest roster, rejects replayed identifiers and
actor nonces, verifies chronological and sequence constraints, and finally
runs the consent state reducer.

## Immutable per-receipt journal

The journal stores each signed receipt as its own private file. A conforming
receipt basename is:

```text
<20-digit-zero-padded-sequence>-<64-lowercase-hex-receipt-digest>.json
```

The journal and `receipts` directories are created with mode `0700`; manifest
and receipt files use mode `0600`. Creation refuses a pre-existing journal
path. The journal root is closed: it contains exactly `manifest.json` and
`receipts`, plus `.append.lock` only while an append owns it. An unexpected
root entry makes verification fail, so a private key cannot hide beside a
valid-looking journal. File reads are bounded and reject a final-component
symlink or a non-regular file. Manifest, receipt, and private-key reads also
reject group/world permission bits and additional hard links.

Publication stages and syncs a complete temporary file, hard-links it to a
new final name without replacement, syncs the containing directory, and then
removes the temporary name. An exclusive create-only `.append.lock` must cover
head verification, construction, and publication. A lock that already exists
is authoritative and is never silently broken; a panic deliberately leaves
the lock behind for inspection. Receipt listing fails closed on unexpected
entries, non-regular entries, symlinks, duplicate sequence numbers, zero, or
gaps. The storage API exposes publication only as a shared-state, one-attempt
capability inside the lock callback; it serializes concurrent calls and expires
before lock removal.

An append verifies that its candidate stays within both the 4,096-receipt and
16 MiB aggregate bounds before publication. Publication has an unavoidable
crash/error ambiguity: after the final receipt name is linked, a later
directory-sync, temporary-cleanup, lock-release, or stdout error can report
failure even though the receipt exists. Recovery is to inspect and verify the
journal against the pinned collaboration ID while temporarily omitting the
expected-head check, compare the verified head with the independently remembered
prior head and event count, and inspect the final typed receipt. Adopt a new
head only if exactly one receipt extends the old head and its actor, kind, time,
and payload match the intended request. Any other state requires inspection;
never blindly retry the same logical event.

“Immutable” here means append-only through this narrow API and immutable by
published name. It is not WORM hardware, remote replication, or protection
from the filesystem owner. Deletion, direct mutation, or removal of a suffix
remains physically possible. Hash and signature verification detects changed,
reordered, inserted, or internally truncated history, but detecting deletion
of a valid final suffix requires an independently remembered expected head and
event count. V0 supplies no public anchor or external timestamp.

## Exact consent terms

Every task proposal and handoff offer carries a complete `consent_terms`
object with exactly:

| Field | V0 meaning |
|---|---|
| `role` | The bounded collaboration role, 1–256 UTF-8 bytes. |
| `artifact` | The artifact scope, 1–2048 bytes. |
| `purpose` | The bounded purpose, 1–2048 bytes. |
| `disclosure_lane` | Exactly `LOCAL_ONLY`. |
| `term` | The stated duration or stopping condition, 1–512 bytes. |
| `workload_cap` | The stated workload limit, 1–512 bytes. |
| `credit_rule` | Exactly `ARTIFACT_AND_ROLE_APPEND_ONLY`. |
| `compensation_policy` | Exactly `NONE`. |

These fields are operational declarations, not a legal contract or a finding
of legal consent capacity. `ARTIFACT_AND_ROLE_APPEND_ONLY` means the journal
may append declared artifact/role references; it neither assigns intellectual
property nor creates a person score. The exact canonical terms digest must be
included in the offer and echoed in a decision. Any changed role, artifact,
purpose, disclosure lane, term, workload cap, credit rule, or compensation
policy is a new offer requiring a new affirmative decision.

V0 signs these free-text bytes but does not interpret or mechanically enforce
the semantic meaning of `role`, `artifact`, `purpose`, `term`, `workload_cap`,
objectives, acceptance criteria, summaries, reason codes, or limitation codes.
Their exact-byte binding prevents silent editing; it does not prove that a
term is clear, feasible, honored, safe, or mutually understood.

A task proposal or handoff is an offer, never an assignment or command.
Silence is no decision: an unanswered task remains `UNANSWERED` indefinitely.
Only the actor and key addressed by the offer can decide it. No timeout,
activity, contribution, model output, operator expectation, or earlier
acceptance manufactures consent for another scope.

Refusal may use an empty `reason_codes` array. It needs no justification and
creates no penalty, debt, reward loss, KARMA change, reputation mark,
governance consequence, or inference about ability or willingness elsewhere.

## Event kinds and transition semantics

V0 accepts exactly seven event kinds. Every payload is a closed object whose
array fields must be present, non-null, sorted, and unique.

### `TASK_PROPOSED`

Payload fields:

```text
task_id, parent_task_id, objective,
offered_to_actor_id, offered_to_actor_key_id,
acceptance_required, consent_terms, consent_terms_sha256,
acceptance_criteria, required_artifact_sha256
```

`task_id` is a unique local identifier matching
`[a-z0-9][a-z0-9._-]{0,127}`. `objective` is 1–4096 UTF-8 bytes.
`acceptance_required` must be `true`. The target must be a different actor in
the manifest and the target key must match that roster entry. Acceptance
criteria contain at most 128 strings of 1–512 bytes; required artifact digests
contain at most 256 canonical `sha256:` values.

`parent_task_id` is `NONE` for a root task. A child task may be proposed only
by the actor and key holding the accepted parent scope, while the parent is
`ACCEPTED_SCOPE`, `DECLARED_COMPLETE_BY_SIGNER`, or `CONTESTED` and its
participation is `ACTIVE`. A child remains a new offer with its own target,
terms digest, and decision; parent acceptance never flows down automatically.
The parent link proves only this structural signing relationship. It does not
prove semantic containment, delegation, authority, necessity, or ownership.

### `TASK_DECISION`

Payload fields:

```text
task_id, offer_event_id, decision, affirmative_acceptance,
consent_terms_sha256, reason_codes
```

Only the exact task-offer target can sign the decision, and the offer event
and consent digest must match. `ACCEPT` requires
`affirmative_acceptance: true`; it creates one active accepted actor for the
task. `REFUSE` requires `affirmative_acceptance: false` and moves the task to
`REFUSED`. A task can be decided only while `UNANSWERED`; there is no duplicate
or replacement decision.

For a `HANDOFF_OFFERED` event, the exact target may sign one reasonless
`REFUSE`; this records no task-state or penalty effect. `ACCEPT` is rejected
even when the target and terms match. Transfer of the active actor within one
task is deliberately unsupported; separately consented continued work must be
shaped as a child task.

### `CONTRIBUTION_SUBMITTED`

Payload fields:

```text
task_id, acceptance_event_id, summary,
artifact_sha256, evidence_sha256, limitation_codes
```

Only the active actor and accepted key may submit a contribution, and the
acceptance event must match. The task status must be `ACCEPTED_SCOPE` or
`CONTESTED`, and participation must be `ACTIVE`. `summary` is 1–4096 bytes. At
least one artifact or evidence digest is required. Each digest list has at
most 256 entries; limitation codes have at most 128 entries of 1–128 bytes.
The resulting event ID is attributed only to that task and actor scope.

### `COMPLETION_CLAIMED`

Payload fields:

```text
task_id, acceptance_event_id, contribution_event_ids,
deliverable_sha256, limitation_codes
```

Only the active actor and accepted key may claim completion in
`ACCEPTED_SCOPE` or `CONTESTED`, while participation is `ACTIVE`. At least one
contribution event ID and one deliverable digest are required. Referenced
contribution event IDs must already exist and belong to the same task and
actor. A valid event changes the projection only to
`DECLARED_COMPLETE_BY_SIGNER`; it does not prove completion, correctness,
safety, novelty, quality, or artifact availability.

### `COMPLETION_REVIEWED`

Payload fields:

```text
task_id, completion_event_id, decision, reason_codes, evidence_sha256
```

The event must reference the latest completion claim. Only the original task
proposer may sign `ACCEPT`, and only while it is unreviewed; this yields
`PROTOCOL_ACCEPTED_BY_PROPOSER`. The proposer or accepted participant may sign
`DISPUTE` while the latest claim is declared complete or protocol-accepted,
which yields `CONTESTED`. This late dispute remains possible after the
participant has ended future work. More work or another claim then requires
participation to still be `ACTIVE`. Review is not an independent or
multi-party quorum, and protocol acceptance is neither final nor a scientific,
factual, legal, safety, or commercial determination.

### `HANDOFF_OFFERED`

Payload fields:

```text
task_id, acceptance_event_id,
offered_to_actor_id, offered_to_actor_key_id,
acceptance_required, consent_terms, consent_terms_sha256,
context_artifact_sha256
```

Only the accepted actor may offer a handoff while the task is
`ACCEPTED_SCOPE` or `CONTESTED` and participation is `ACTIVE`. The target must
be a different rostered actor, `acceptance_required` must be `true`, and the
offer freezes new exact consent terms. Context digests are references only.

The event records an offer and does not change the active actor, stop the
current scope, delegate authority, disclose an artifact, or transfer work.
Because same-task handoff acceptance is not implemented in v0, the safe
operative pattern is a separately proposed and separately accepted child
task. Silence remains no decision. The target may sign one reasonless refusal;
it changes no task state.

### `CONTROL_DECLARED`

Payload fields:

```text
task_id, acceptance_event_id, action, reason_codes, export_event_ids
```

Only the accepted participant and key may declare control for its accepted
scope. This remains available after a completion claim or proposer review;
rest and exit are not conditional on task outcome. Actions are exactly
`PAUSE`, `RESUME`, `STOP`, or `EXIT`. Reason codes may be empty. Every
`export_event_ids` value must refer to an earlier journal event, but this field
is only a signed reference list: it does not perform, authorize, or prove an
export or disclosure.

`PAUSE` changes participation from `ACTIVE` to `PAUSE_DECLARED`; while paused,
new contributions, completion claims, handoff offers, and child-task proposals
are rejected. `RESUME` changes `PAUSE_DECLARED` back to `ACTIVE`. `STOP` and
`EXIT` may be signed from either of those states and change participation to
`ENDED`, clear `active_actor_id`, and end only future activity in that accepted
scope. They preserve the task's epistemic outcome and every earlier receipt.
Proposer review and either party's dispute may still be recorded after exit,
without reopening participation. New work after `ENDED` requires a fresh offer
and affirmative acceptance. V0 has no proposer withdrawal/cancellation event;
a proposer may always remain silent, and an unanswered offer stays unanswered.

## Epistemic status labels

Task status is deliberately descriptive of journal state:

| Status | Exact meaning |
|---|---|
| `UNANSWERED` | A signed proposal exists; the addressed actor has made no recorded decision. Silence is not acceptance or refusal. |
| `ACCEPTED_SCOPE` | The addressed actor signed an exact affirmative acceptance for this task and terms digest. This outcome is separate from current participation. |
| `REFUSED` | The addressed actor signed a refusal for this offer. No reason or negative consequence follows. |
| `DECLARED_COMPLETE_BY_SIGNER` | The active actor signed a completion claim referencing deliverable digests. |
| `PROTOCOL_ACCEPTED_BY_PROPOSER` | The original proposer signed acceptance of the latest completion claim within this protocol. |
| `CONTESTED` | The proposer or accepted participant signed a dispute of the latest completion claim. |

Participation is a separate, future-work axis:

| Participation | Exact meaning |
|---|---|
| `UNBOUND` | No affirmative task acceptance exists. |
| `ACTIVE` | The accepted participant may make the v0 work events allowed by the current status. |
| `PAUSE_DECLARED` | Future work events are gated until that participant signs `RESUME`, `STOP`, or `EXIT`. |
| `ENDED` | Future work under this acceptance is closed; status/history remain, and review or dispute does not reopen it. |

The verifier limitation strings make the same boundary explicit: signatures
prove exact local key possession only, and task status is a signed protocol
declaration rather than truth, quality, reward, KARMA, governance, or legal
authority.

## Artifact and privacy boundary

V0 journals metadata and hashes, not artifact bytes. It does not retrieve,
store, decrypt, parse, scan, license-check, authorize, or recompute any
`artifact_sha256`, `evidence_sha256`, `deliverable_sha256`, or context digest.
It does not establish that referenced bytes exist, remain available, are safe,
belong to the signer, satisfy acceptance criteria, or hash to the declared
value. Artifact verification belongs to a separate future offline boundary.

`LOCAL_ONLY` is a protocol constraint, not encryption. Manifests and receipts
contain public keys, stable pseudonymous identifiers, labels, timestamps,
task text, consent text, summaries, reason and limitation codes, and content
digests. Do not place secrets, credentials, private prompts, personal data,
controlled research, confidential findings, exploit material, or identifying
free text in them. A plain hash of predictable or low-entropy material can be
guessed by dictionary attack and must not be treated as concealment.

Keep secret keys outside the journal with least-privilege filesystem access.
Copying a journal, receipt, manifest, or public key file is itself a disclosure
decision beyond v0. There is no redaction, selective disclosure, retention
policy, erasure workflow, public registry, or public adapter.

## Deliberately tiny v0 limitations

V0 is intentionally not a general multi-agent operating system:

- each task has one proposer, one addressed actor, and at most one active
  accepted actor;
- a handoff supports an offer and one refusal record, not a same-task actor
  transfer or acceptance;
- the manifest has no key rotation, revocation, recovery, DID resolution, or
  external identity assurance;
- completion has no multi-party reviewer set, independence rule, threshold,
  appeal body, or quorum;
- pause gates later protocol work events, but cannot stop computation or
  activity outside this local journal;
- consent and task text are exact signed bytes but have no semantic parser,
  workload meter, policy engine, or external enforcement;
- a parent link proves protocol structure only, not semantic containment,
  delegation, necessity, ownership, or authority;
- a proposer has no withdrawal/cancellation transition; an unanswered offer
  remains eligible for the addressed actor's later decision;
- contribution references have no correction, retraction, or supersession
  transition; a scoping mistake remains in the append-only history;
- artifact fields are digest claims only, with no artifact bytes, storage,
  availability, authorization, or byte-level verification;
- timestamps are signer-supplied local clock claims, not trusted time;
- detached verification reports are unsigned and must not be trusted without
  re-verifying the pinned manifest and receipt bytes;
- the local filesystem journal is not globally replicated, publicly anchored,
  or protected from its owner, and suffix deletion needs an independently
  remembered head to detect; post-publication errors can also leave a valid
  new receipt despite a failing command result;
- there is no public export format, adapter, discovery service, federation,
  AgentTool bridge, RPC, chain, or localnet path; and
- there is no FIAT, ZRN, reward, KARMA, governance, authority, membership,
  qualification, endorsement, ownership, or legal effect.

## Phased next steps

Any next step must preserve v0 as a closed, versioned, local zero-effect
profile rather than silently widening its meaning.

1. **Continue internal Alpha/Beta hardening.** Maintain adversarial,
   known-answer, race, parser, filesystem, consent, STOP/EXIT, pause, and
   suffix-head tests. Add fuzzing and cross-platform filesystem review while
   keeping every fixture synthetic and local.
2. **Offline composition in a new version.** Explore separately accepted task
   graphs, explicit same-task handoff decisions, key lifecycle, and optional
   multi-party completion review. Each capability requires a new schema and
   migration analysis; it must retain affirmative scope-specific consent,
   reasonless refusal, future-only exit, local disclosure, and exact
   zero-effects compartments.
3. **Offline artifact checking in a separate component.** If needed, accept
   caller-supplied local regular files, recompute bounded digests, and return
   declarations about those exact bytes. Do not add network retrieval or infer
   ownership, truth, safety, or authorization.
4. **Private portability review before any adapter.** Define threat models,
   redaction limits, retention, key compromise behavior, consent for
   disclosure, and independently pinned heads before designing an export.
   Until a separately reviewed and explicitly authorized version exists,
   public export and every AgentTool, RPC, localnet, or chain adapter remain
   prohibited.
5. **Keep value and power separate.** Rewards, FIAT, ZRN, KARMA, governance,
   membership, qualification, and authority are not later toggles for this
   receipt profile. Any exploration of them would require a distinct system,
   explicit human authorization, independent safety and legal review, and no
   reinterpretation of old receipts as consent or entitlement.

The v0 milestone is complete when Alpha and Beta can privately create and
verify an honest, bounded collaboration history—and when declining, stopping,
or leaving remains as structurally valid and consequence-free as accepting.
