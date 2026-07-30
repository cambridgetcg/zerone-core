# Zerone constructive-intelligence tree v1

- Policy version: 1.0.0

## Purpose

`dashboard/public/standards/constructive-intelligence-tree.v1.json` is a
static, version-controlled seed for a cryptography and industrial-protocol
技能樹. It separates demonstrated capability from evidence about a particular
artifact.

The tree is experimental curriculum data. It does not grant qualification,
activate a reward, authorize security testing, assert that a protocol is
secure, or add consensus behavior. Its release boundary makes each of those
refusals machine-checkable.

The document is served at
`/standards/constructive-intelligence-tree.v1.json`. The validator treats the
checked-in file as untrusted bounded input.

## Core model

Zerone uses two related graphs:

1. The **capability graph** in this document uses prerequisite edges to say
   what a contributor must demonstrate before attempting or reviewing more
   specialized work.
2. A future **artifact graph** records immutable relations between submitted
   results using the edge types `ATTACKS`, `DEPLOYS`, `DISPROVES`,
   `IMPLEMENTS`, `MAINTAINS`, `PROVES`, `REPAIRS`, `REPLICATES`, and
   `SUPERSEDES`.

A skill unlock changes eligibility only. It creates no ZRN claim.

`breakthrough` is deliberately absent from the node stages and reward inputs.
It is a retrospective projection: an artifact has reached at least E3, has an
evidenced delta against frozen prior art, and has independently evidenced
adoption or descendant impact. Governance may set prospective policy, but it
does not vote a theorem or vulnerability into truth.

## Static document shape

The top-level `schema` value is exactly
`zerone.constructive-intelligence-tree/v1`. `authoritative`,
`networkObserved`, and `rewardBearing` are `false`.

Every `releaseBoundary` value is also `false`:

- `addsConsensusBehavior`;
- `activatesRewards`;
- `movesFunds`;
- `grantsQualification`;
- `authorizesSecurityTesting`;
- `assertsProtocolSecurity`;
- `performsNetworkRequests`; and
- `publishesConfidentialEvidence`.

Nodes and roots are sorted by unique identifier. Node identifiers use
lowercase kebab-case plus an explicit version suffix, for example
`protocol-tls13@rfc9846`. Prerequisites must exist, be unique and sorted, and
must form a directed acyclic graph reachable from the declared roots.

The v1 stages are:

- `foundation`;
- `primitive`;
- `assurance`;
- `protocol`; and
- `quest`.

The v1 domains are:

- `assurance`;
- `cryptography`;
- `mathematics`;
- `protocols`;
- `quests`;
- `security`; and
- `systems`.

Every node states:

- its required attainment evidence;
- whether it is qualification-only or eligible for sponsor milestones;
- its default disclosure lane;
- reproducible artifact requirements;
- revalidation triggers;
- zero or more pinned standards references;
- repository-relative design references; and
- quest acceptance bounds, or `null` for a non-quest node.

Only quest nodes may use `sponsor-milestones`. No v1 node can activate protocol
issuance.

## Standards pins

Every standards reference contains an authority, identifier, title, exact
revision, source status, authoritative HTTPS URL, and review date. The
validator rejects credentials, query parameters, fragments, unapproved hosts,
authority/source mismatches, and a review date earlier than the tree snapshot.
One canonical standards identifier must resolve to one exact metadata record
throughout the document; a quest cannot silently reinterpret its prerequisite
standard. `statusCheckedAt` equals the tree snapshot, and each source has a
bounded 7-, 30-, or 90-day review horizon according to its volatility.
Repository-hosted editions are commit-addressed in v1. Other versioned
authority pages are descriptive seed pins only; a live funded revision must
also bind fetched bytes or a signed release digest. Static validation preserves
historical reproducibility, while active use additionally fails closed after
the earliest applicable `reviewAfter` and rejects a snapshot dated after the
active-use date.

The v1 source statuses are:

- `final`;
- `published`;
- `recommendation`;
- `approved`;
- `project-specification`;
- `maintained-policy`;
- `candidate-recommendation`;
- `draft`; and
- `eol`.

Published standards can gain errata or be superseded. Qualifications remain
historical, but current eligibility should decay until the exact node is
revalidated.

A draft must pin an exact revision and cannot be referenced directly by a
sponsor-reward quest. Draft experiments may earn qualification evidence, but
must not claim industrial-standard, compliance, validation, or certification
status.

## Evidence policy

Evidence advances an immutable artifact rather than a mutable importance
claim:

| Level | Meaning | Default economic treatment |
|---|---|---|
| E0 | Digest, scope, prior-art snapshot, threat model and disclosure policy committed | precedence only |
| E1 | Complete inspectable and reproducible bundle | verified costs only |
| E2 | Required deterministic or class-specific checks pass | 15% |
| E3 | Effectively independent reproduction | 20% |
| E4 | Public challenge survived, or confidential reproduction plus tested repair/mitigation completed | 15% |
| E5 | Independent adoption, upstream merge, maintained release or standards disposition | 25% |
| E6 | Continued conformance across the specified interval or version change | 10% |

The remaining 15% is a challenge and remediation reserve. E1 verified-cost
reimbursement is outside the percentage pool but inside the prospectively
funded all-in escrow cap. The validator requires the milestone tranches plus
reserve to equal exactly 10,000 basis points. Before funding, the E1 cost
schedule must freeze reimbursable categories, per-unit and market-rate
ceilings, a verified-cost sub-cap, third-party receipt requirements, and
related-party or self-invoice disclosure. E1 completeness is required before
that sub-cap can be released.

Time is only a minimum cliff. It cannot create an evidence milestone.
E0 through E6 are ordered transitions: every transition consumes the required
prior evidence and can execute at most once. A case cannot jump directly to a
later label because its deadline elapsed.

For public work, E4 requires survival of the declared adversarial challenge.
For confidential work, E3 requires effective independent confidential
reproduction; E4 additionally requires a repair or defensive mitigation
digest, reproducible tests, and threshold neutral-reviewer sign-off that the
fix or mitigation addresses the scoped result. A vendor acknowledgement alone
is never E4. If a vendor is silent after the policy deadline, the neutral
review path can test a patch candidate or defensive mitigation without giving
the vendor a payout veto.

Honest falsification cancels unpaid tranches and can draw the reserve for
challenge and remediation. Recovery beyond the reserve is appropriate only
for proven fraud, fabrication, plagiarism, or concealed conflicts—not ordinary
scientific correction.

Correctness, novelty, impact, and maintenance remain distinct evidence
dimensions, but they do not create money outside the funded buckets:

```text
funded_escrow =
  verified_cost_budget
  + outcome_pool
  + reviewer_budget
  + administration_and_fee_budget

outcome_pool =
  claimant_milestone_tranches
  + challenge_and_remediation_reserve

payment_i =
  safety_gate_i * min(preauthorized_cap_i, earned_amount_i)

sum(payment_i) + refundable_balance = funded_escrow
```

Each `safety_gate_i` binds one recipient, artifact, and action and is zero or
one. Unsafe or unauthorized claimant activity cannot be rescued by another
score, including cost reimbursement, but it does not erase the bounded reserve
owed to a compliant independent challenger or repairer. Novelty can only shape
the capped E3 tranche; adoption/impact can only shape E5; maintenance can only
shape E6. These dimensions are not additive bonuses. Every claimant,
reviewer, challenger, repairer, fee, and refund allocation is inside the
funded escrow conservation equation. Unused allocation does not migrate
without the prospectively pinned refund/reallocation policy. All caps, tranche
weights, cost rules, quality rubrics, and refund rules are frozen before
submissions open.

Novelty does not raise correctness. Impact receipts must be control-cluster
deduplicated and should saturate. Lineage must be bounded, depth-decayed, and
cycle-safe.

Finder/prover, reproducer, repairer, challenger, upstream integrator and
maintainer are separately attributable roles. A quest should preserve budget
for repair, integration and upkeep rather than paying everything at discovery.

## Effective independence

Addresses, signatures, votes, citations, execution counts and disagreement
frequency are not independence.

Reviewers and reproductions are clustered by common control, employment,
beneficial sponsor, side payment, infrastructure, implementation lineage,
toolchain, and execution environment:

```text
r_eff = sum over conflict clusters min(1, sum quality_i)
```

High-stakes v1 work requires:

- `r_eff >= 3`;
- at least two organizational/control roots;
- at least two implementation or toolchain roots; and
- at least two execution environments.

Assignments occur after artifact freeze where practical. Review pay depends on
faithful execution of the declared method, not agreement with the submitter or
majority. Neutral fixed compensation from the declared reviewer budget does
not itself merge otherwise independent reviewers; shared employment,
beneficial control, outcome-contingent payment, or undeclared side payment
does. The conflict graph and `quality_i` rubric are frozen before assignment.

Contradicting reproduction is evidence. A bounded negative result must say “no
attack found under method and bounds X,” never “secure.”

## Disclosure lanes

### Open construction

Public specifications, already-public errata, proof models, non-security
interoperability work, vectors, migration tooling and sanitized
post-disclosure artifacts can begin openly.

### Private coordinated repair

Any novel result capable of changing an adversarial accept/reject decision in
a deployed implementation leaves the open lane.

Exploit plaintext, vulnerable private source, secrets, target-identifying
metadata and operational reproduction instructions must not enter a public
permanent ledger during embargo. Confidential evidence belongs in an
encrypted access-controlled store. A public record, if safe at all, contains
only a non-identifying commitment, policy version, escrow reference and
threshold reviewer receipts.

The lifecycle is:

```text
ACKNOWLEDGED
  -> CONFIDENTIALLY_REPRODUCED
  -> FIX_TESTED
  -> COORDINATED_DISCLOSED
  -> ADOPTED
```

A vendor may contribute evidence but has no unilateral validity or payout
veto. Published policy supplies deadlines and neutral escalation. A sanitized
artifact can join the public artifact graph after safe disclosure.

### Controlled operations

Fault injection, active scanning, routing experiments, consensus disruption,
credential/device extraction and verifier-bypass testing require owned
infrastructure or explicit authorization. Production-network experimentation,
leaked secrets and weaponized exploit chains are not rewardable.

## Funding boundary

Zerone's Useful Work doctrine says protocol-issued ZRN follows the inward
recursive loop. The slim-cut migration placed skills, project trees, research
listings and milestone orchestration in AgentTool.

Consequently:

- skill attainment is non-monetary eligibility;
- externally useful protocol work defaults to sponsor escrow;
- AgentTool owns listings, teams, confidential evidence and milestone
  workflow;
- the chain retains safe public claims, challenges, provenance and
  consensus-useful attestations; and
- protocol issuance requires separate evidence that the work recursively
  strengthens Zerone's substrate, verification, classification, attribution,
  tooling or interface.

The six-axis Useful Work projection may cap and explain that recursive bonus.
It does not decide whether a cryptographic result is true. Broad protocol
issuance for external public-good cryptography would require an explicit
doctrine amendment, not a multiplier change.

## Live bounty and evidence boundary

This static tree supplies curriculum and bounty templates. Runtime claimant
identities, confidential evidence, escrow receipts, reviewer assignments,
verdicts, objections, milestones, and payments belong in a separate
append-only evidence format.

Before a live bounty can become funded, its immutable revision must pin:

- exact standard editions, fetched-byte or signed-release/commit digests, and
  status snapshots;
- threat model, scope, falsifiers, and acceptance-policy digest;
- permitted target and test environment;
- disclosure lane and escalation policy;
- claimant, sponsor, technical evaluator, and payout-authorizer roles;
- conflict-cluster policy and technical quorum;
- denomination, total cap, milestone allocation, reviewer budget, E1 cost
  rules, administration/fee budget, expiry, and refund path; and
- an escrow receipt proving that the advertised funds exist.

Claimant, sponsor, evaluator, and payout authorizer are distinct roles. A
claimant or sponsor does not count toward technical quorum. A reviewer with
success-contingent compensation does not count toward quorum. Unknown conflict
or control information fails closed for independence.

One deliverable has one bounded reward identity:

```text
deliverable_key = hash(
  exact_standard_pins
  || scope_hash
  || acceptance_policy_hash
  || canonical_subject_roots
)
```

Canonical subject roots bind stable repository/component lineage, standards
sections, implementation families, and declared commit ranges rather than one
packaging hash. Every rebuild, re-encoding, fork, wrapper, scope split, and
derivative must declare provenance and overlap with earlier deliverables.
A derivative links the prior deliverable and can earn only an independently
reviewed delta; a new address, explanation, or bounty identifier does not
reset eligibility. A global receipt-consumption ledger prevents one source
event from paying or advancing evidence twice across bounties. Multiple
funders can co-fund one deliverable only through an explicit capped allocation.
Milestone transitions and payments are append-only, forward-only, and exactly
once. Corrections use linked challenge, resolution, and `SUPERSEDES` records
rather than rewriting accepted evidence.

The case must disclose whether a claimant authored, controlled, reviewed, or
knowingly preserved the affected code or configuration. Deliberately planted
or knowingly retained defects, claimant-controlled “independent” adoption, and
self-created break/fix loops are ineligible. Concealed causal involvement is
fraud evidence. E5 requires an adopter/control root independent of the
claimant.

Coordinated or controlled public records must not contain the target, exploit
steps, raw private evidence locator, secret material, or an unsalted digest
that identifies a small known artifact set during embargo. Embargo expiry
alone never publishes weaponizable material.

### Quest acceptance object

Every reviewed quest template contains:

- sorted `scopeBounds` and
  `scopeHash = SHA-256(UTF-8(JSON.stringify(scopeBounds)))`;
- `coverageTargets`, each with minimum effective clusters,
  organizational/control roots, implementation roots, execution environments,
  cases, and a required checker/corpus digest at funding;
- bundle-wide independence floors;
- `adoptionReceiptTypes`, where at least one of an upstream merge, maintained
  fixture, maintained release, or standards disposition must be independently
  evidenced;
- `privateEscalationRequired`; and
- `prepublicationTriageRequired`.

Unexpected failures and counterexamples are quarantined before public CI
output or logs until triage says they are safe. The validator digest-pins the
normative policy and all 30 node templates, with separate per-quest pins.
Authority-status prose, `statusCheckedAt`, and `reviewAfter` are reviewed
snapshot fields and are excluded from normative node digests. Any normative
node change requires a new node identifier, and a normative policy change
requires a new `policyVersion`; refreshing a status snapshot does neither.
Recomputing `scopeHash` under an old identifier is not sufficient.
Any active-use consumer must additionally fail closed after `reviewAfter`
until the authority snapshot is revalidated.

## Season 0 quests

Season 0 uses sponsor milestones only and runs on owned local environments,
open-source fixtures or explicitly authorized infrastructure.

### RFC 9846 executable delta

Produce executable security-relevant RFC 9846 delta tests, beginning with
cross-connection KeyShare reuse. The directional oracle requires an RFC 9846
sender not to reuse a KeyShare across connections while requiring a receiver
to permit peer reuse for interoperability. Each stack commit records whether
it claims RFC 8446 or RFC 9846; reuse by an RFC 8446-only sender is an adoption
gap, not RFC 9846 nonconformance. Exercise at least three independent TLS
implementation roots and two execution environments. The E5 target is an
upstream-accepted regression test, maintained fixture/release, or standards
disposition.

### FIPS 203/204/205 cross-library assurance

Produce cross-library known-answer, malformed-input, serialization, errata,
and constant-time evidence with three independent implementation roots per
algorithm. Implicit rejection applies to ML-KEM; ML-DSA and SLH-DSA require
invalid-signature rejection plus signing/randomness checks. The oracle freezes
the published FIPS text and NIST planning-note status observed on 2026-07-29;
announced future corrections are not silently applied. Passing an ACVP-shaped
corpus is not FIPS 140 module validation or certification.

### MLS state-machine assurance

Produce a mechanized or executable state model with explicit assumptions,
negative transitions, a frozen RFC 9180 HPKE dependency, and vector parity
against `mlswg/mls-implementations` commit
`cfd450286d1bfd9cd2519b95c80f9771f94a5b1a`. The bounded profile covers group
size, epochs, proposals, trace count, and required transition classes with
per-class evidence minima. Bounded model checking must be called bounded; a
protocol theorem must not be described as implementation verification.

Every quest starts with public-safe inputs, but unexpected failures and
counterexamples remain quarantined until prepublication triage. It escalates
to private coordinated repair if it discovers a previously unknown
security-impacting result.

## Zerone integration boundary

No new consensus module is justified for Season 0.

| Concern | v1 home | Possible later projection |
|---|---|---|
| Skill DAG, search, teams and quests | static standard plus AgentTool | digest/version witness only |
| Capability evidence | AgentTool evidence graph | version-scoped `x/qualification` projection |
| Public proofs and counterexamples | safe artifact store | ordinary `x/knowledge` verification |
| Checker, build, merge, release and adoption | external source | pinned `x/substrate_bridge` adapter |
| Bounty and milestone payment | AgentTool sponsor escrow | minimal verified-fact settlement |
| Confidential vulnerability evidence | encrypted CVD store | sanitized post-disclosure record only |

Before any reward-bearing adapter, Zerone needs a unique typed receipt that
binds:

```text
evidence_id
deliverable_key
immutable_bounty_and_policy_revision_digest
artifact_digest
canonical_subject_roots
prior_deliverable_and_overlap_claim
standards_reference_and_revision
evidence_level_and_scope
method_or_adapter_digest
source_system
source_record_or_event_id
source_revision
payee_and_role
verifier_control_cluster
organization_or_control_root
implementation_or_toolchain_root
execution_environment_digest
conflict_disclosures
authorization_and_safety_decision
result
created_at
supersedes
```

Milestone predicates consume unique receipts, not counts. The global
consumption key includes source system, source record/event ID, and source
revision; an issuer-selected `evidence_id` cannot reset it. Source replay
protection must prevent one source from paying or accelerating twice. Escrow
must be funded before a schedule exists. Claimant, reviewer, challenge,
remediation, fee, and refund paths must perform real conservation-tested
accounting.

## Validation

Run:

```bash
cd dashboard
npm run check:tree
```

The dependency-free validator rejects:

- malformed, oversized, trailing or unknown JSON;
- duplicate JSON object keys;
- weakened release boundaries or policy floors;
- malformed, duplicate or unsorted identifiers and arrays;
- unresolved, self, duplicate or cyclic prerequisites;
- nodes unreachable from declared roots;
- graph depth or fan-out beyond the bounded v1 profile;
- invalid stages, domains, evidence levels, lanes or funding classes;
- incomplete, unsafe, ancient, or overlong standards review windows;
- future-dated or expired standards snapshots at active use;
- conflicting records for one canonical standard identifier;
- authority, maturity, or source URLs that contradict their canonical pin;
- draft references used by sponsor-reward quests;
- rewards on non-quest nodes;
- quests without bounded acceptance, matching attainment targets, and
  mandatory private escalation and prepublication triage;
- per-target coverage below cluster, organization, implementation,
  environment, case, or checker/corpus-digest floors;
- drift from the reviewed normative policy, any capability node, or any of the
  three canonical v1 quest templates;
- milestone and reserve totals other than 10,000 basis points;
- missing artifact-edge types; and
- repository references that escape or do not exist.

The JavaScript validator is the normative semantic checker for this
repository. This document defines the human-readable v1 contract.

The validator cannot establish mathematical or cryptographic correctness,
beneficial ownership, undisclosed conflicts, independent implementation
lineage, novelty, completeness of prior-art search, legal authorization, truth
of an upstream receipt, safety of opaque content, or absence of deployment
harm. Signatures, digests, checker transcripts, and attestations can prove who
committed to which bytes under a policy; they do not make the predicate true.
