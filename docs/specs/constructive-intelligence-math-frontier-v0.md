# Constructive-intelligence Math Frontier v0

Status: experimental, static, source-only, zero-value

## Purpose

The Math Frontier is a versioned extension of Zerone's constructive-
intelligence tree. It gives formal mathematics enough resolution to express
proof construction, falsification, formalization, independent reproduction
and downstream reuse without changing the reviewed core tree v1.

The machine-readable document is served from
dashboard/public/standards/constructive-intelligence-math-frontier.v0.json.
It is curriculum and quest-template data. It is not chain state, a credential,
a ranking, a reward, or governance.

The extension exists because the reward design's first sponsor-funded pilot
requires a separately reviewed formal-mathematics quest. Editing the frozen
30-node core tree would change its exact document digest and invalidate
receipt bindings. Math Frontier therefore binds the existing tree by digest
and imports four capability identifiers without mutating it.

## Release boundary

The document fixes authoritative, networkObserved, rewardBearing and
governanceBearing to false. Every releaseBoundary switch is also false:

- no consensus behavior;
- no reward activation or movement of funds;
- no qualification;
- no governing power or person ranking;
- no authorization to test systems;
- no network requests; and
- no publication of confidential evidence.

An implementation must reject an unknown field rather than treating it as a
new activation switch. It must also reject any owner, founder, administrator,
beneficiary, reserved-seat, reserved-share, payout-address or identity-class
multiplier field.

## Constitutional shape

Yu, an AI, a creator, an operator and any later participant enter through the
same evidence rules. Identity labels cannot change validity, eligibility,
voice or the prospective reward envelope. The document permits no reserved
seat or reserved share and gives neither creator nor operator a privilege.

Stake, reward balance and raw KARMA magnitude affect neither mathematical
validity nor governing voice. The only future governance direction named here
is controller-level eligibility plus sortition:

- mature evidence may eventually make a controller eligible;
- selection from the eligible set must not be proportional to wealth, reward,
  address count, agent count or raw KARMA;
- every selected independent controller has one voice;
- conflicts require recusal and quorum is recomputed afterward; and
- emergency action is pause-only.

That mechanism is not implemented. currentActivationAuthority is NONE and the
ordinary stake vote and emergency authority cannot activate this template.
No value-bearing path may infer an activation authority from this document.

The existing specialized research-spend path is explicitly barred from this
program because its dormant reserved-voter design is not KARMA governance.

## KARMA boundary

Math Frontier uses only an ordinal shadow observation:

- schema zerone.karma.shadow-edge/v0;
- state ORDINAL;
- register priced-coherence;
- no magnitude;
- no on-chain recognition;
- no truth-oracle claim;
- no transfer, spending, reward multiplier or governance weight; and
- economic and control effects are NONE.

An ordinal observation says that a candidate artifact relation was seen under
this local template. It does not say who deserves standing, whether the
mathematics is true, or whether a person controls an address. It must not be
projected into the on-chain zerone.karma.edge RECOGNIZED state.

Future recognition requires signed typed receipts and independently governed
controller resolution. Self edges, same-controller edges, reciprocal cycles
and unverified external edges remain excluded. No dashboard count or score
may silently upgrade them.

## Reward shape

The checked-in reward template describes only proportions inside a future,
prospectively funded sponsor envelope. Its live amount is the exact string
"0", its economic effect is NONE, and protocol issuance is forbidden.

Skill unlock and breakthrough recognition create no entitlement. A later
pilot would need all of the following outside this document:

- a dedicated, already-funded escrow;
- a frozen policy before admission;
- one atomic settlement;
- a separately capped verified-cost schedule;
- named refund or commons disposition;
- controller and semantic-root replay protection; and
- an explicit sunset.

The prospective outcome-pool shape is:

| Evidence | Purpose | Basis points |
|---|---|---:|
| E2 | deterministic validity | 1,500 |
| E3 | independent reproduction | 2,000 |
| E4 | adversarial survival or honest disproof | 1,500 |
| E5 | independent downstream use | 2,500 |
| E6 | maintained recheck | 1,000 |
| Reserve | challenge and remediation | 1,500 |

The sum is exactly 10,000 basis points. E0 gives precedence only. E1 may
support verified costs only under a separate preauthorized cap; it is not an
outcome percentage.

Claimant, formalizer, independent reproducer, falsifier, integrator and
maintainer are distinct roles. This version does not assign fixed role
percentages because each problem packet must preserve enough budget for its
actual verification and repair shape. Honest counterexamples stay eligible.
If a compliant falsifier disproves the frozen claim, the E4 disposition goes
to that falsifier and the failed claimant remains unpaid; E4 is not a
consolation payment for a false claim.

## Skill tree

The extension has four display stages.

### Ground

- Logic, definitions and semantic hygiene
- Reproducible mathematical computation
- Prior art and semantic roots

### Craft

- Finite algebra and lattice construction
- Analysis and geometry construction
- Combinatorics and discrete structure
- Probability and stochastic construction
- Optimization and numerical evidence
- Formalization and trusted kernels
- Counterexamples and falsification

### Assurance

- Independent proof and reproduction
- Multi-kernel proof assurance

### Frontier

- Formal construction quest

The graph imports these exact identifiers from the digest-pinned core tree:

- math-algebra-finite-fields@1;
- math-lattices-polynomial-rings@1;
- math-probability-information-complexity@1; and
- math-proofcraft@1.

Every other prerequisite must resolve within the extension. Nodes are sorted
by identifier, prerequisites are unique and sorted, the graph is acyclic and
all nodes are reachable from at least one imported capability. No node unlocks
a reward.

## Formal construction quest

quest-math-formal-construction@1 accepts one domain capability from the
published allowlist. Exactly one is frozen into a packet, its E2 attainment
receipt is required, and the selected domain's own prerequisite closure still
applies. Labelling a domain without that receipt is insufficient. A problem
packet must bind:

- exact statement digest;
- exact formalization digest;
- axiom-policy digest;
- semantic root;
- prior-art cutoff and manifest digest;
- checker and kernel digests;
- falsifier-suite digest; and
- selected domain-capability identifier, E2 evidence label and receipt digest.

The artifact relation is exactly PROVES, DISPROVES or IMPLEMENTS.
Validity is relation-specific: IMPLEMENTS can establish a faithful formal
encoding without establishing that the encoded theorem is true.

Breakthrough is derived only after:

1. deterministic E2 validity under the frozen statement and axiom policy;
2. effectively independent E3 reproduction;
3. E4 adversarial survival or an honest disproof record; and
4. E5 use by a downstream artifact under an independent controller.

Popularity cannot override a failed proof. Repackaging the same semantic root
cannot create another opportunity. Minimum floors are three effective
controller clusters, two organization roots, two implementation roots, two
execution environments and two kernel families.

E6 is maintenance after recognition, not a prerequisite for the retrospective
breakthrough label. Time alone creates no milestone.

## Problem packets

A packet is a frozen admission candidate, not an award. The v0 known-answer
fixture is synthetic and has economicEffect, controlEffect and qualification
all fixed to NONE. It must not name a beneficiary or payout destination. Its
two checker digests are only deterministic test inputs: distinct digests do
not establish distinct kernel families, controllers, organizations,
implementations, or environments.

The v0 executable accepts only this exact known-answer packet. It is not a
general admission validator. A future packet schema and validator must:

- bind the exact frontier-document digest;
- require every packet binding named by the quest template;
- require one allowed domain capability and artifact relation;
- reject duplicate object keys, unknown fields, nulls and malformed digests;
- reject a self-declared breakthrough;
- reject any non-ordinal KARMA state;
- keep the amount at zero; and
- produce a deterministic packet digest.

The fixture's prior-art cutoff cannot exceed the 2026-08-01 snapshot. A future
admission packet must bind the cutoff to its own prospective policy version;
v0 cannot be reused by moving that date.

## Validation and active use

The executable validator is offline. It treats both JSON files as untrusted
bounded input, checks exact keys and values, recomputes the base-tree byte and
policy digests, validates the graph, and pins reviewed known-answer digests.
URLs are not fetched by the validator.

Known answers:

| Object | SHA-256 |
|---|---|
| Core tree exact bytes | 8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf |
| Core policy canonical JSON | 36116220c7f17dd06f8bda2217d79a000aaac771075a709004686b233402abc7 |
| Math Frontier exact bytes | 4fdcd54c35c69c26a28c385275688351ee2a9131702e81bacf100de8d7612456 |
| Math Frontier canonical JSON | b6260de31969a56e601a5a81f1b4f7c1c68fcd34f9aa68b35ce2701c3f012503 |
| Quest template canonical JSON | 7af1dc87b98f6b80bb798aa1427c82c5b2c049f8cdccd9b914188cba50718313 |
| Synthetic packet canonical JSON | f0961813b83cbd9f127290c19cf5bc98c07cad2dc1158bbab26243edd7af9ae3 |

Passing static validation proves document shape and exact reviewed bytes only.
It does not authenticate external evidence, resolve controllers, establish
independence, verify a proof kernel, reserve funds, emit KARMA, or activate a
governance path.

Any future value-bearing revision requires a new schema version and cannot
weaken these boundaries by editing v0 in place.
