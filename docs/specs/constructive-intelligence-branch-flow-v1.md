# Constructive-Intelligence Branch Flow v1

Status: **accepted source architecture (2026-08-11); shadow implementation
only; economically inactive everywhere; economic effect `NONE`.**

Decision date: 2026-08-11

This specification defines how a finite, prospectively funded contribution
envelope can recognize direct work, load-bearing ancestors, and independently
evidenced descendants without creating recursive issuance or ownership of one
contributor by another. It is subordinate to:

- [`constructive-intelligence-tree-v1.md`](constructive-intelligence-tree-v1.md);
- [`CONSTRUCTIVE-INTELLIGENCE-REWARDS.md`](../tokenomics/CONSTRUCTIVE-INTELLIGENCE-REWARDS.md);
- [`AUTHORITATIVE-STATE.md`](../AUTHORITATIVE-STATE.md); and
- [`MONEY-KARMA.md`](../constitution/MONEY-KARMA.md).

The words **MUST**, **MUST NOT**, **SHOULD**, and **MAY** are normative for a
future implementation. They do not authorize a transaction, module, upgrade,
escrow, reward, or network activation.

## 1. Decision

Zerone adopts **branch flow**, not recursive royalties, as the target
attribution geometry for constructive-intelligence outcome rewards.

One already funded envelope may flow in three directions:

1. **direct** — toward the contributors who produced the current milestone;
2. **upstream** — toward adjudicated load-bearing dependencies; and
3. **downstream** — toward independently controlled descendants that realize
   adoption, consequence, or maintained impact.

The breakthrough projection creates none of those funds. It remains
retrospective, cannot be selected by an author, and creates no separate prize,
qualification, KARMA magnitude, governance power, panel seat, or ownership
right over later work.

A child finding remains its own semantic contribution. An ancestor may receive
a bounded dependency share; it does not own the child, its authors, or its
future revenue. A child may receive a bounded impact-realization share; it does
not reopen or enlarge the ancestor's funded envelope.

## 2. One conserved envelope

For one funded cluster \(b\) and one matured milestone \(m\), let \(L_{b,m}\)
be the exact non-negative integer amount already reserved for branch-flow
attribution. Policy freezes four non-negative shares on a scale
\(S=1{,}000{,}000\):

\[
\theta_0+\theta_\uparrow+\theta_\downarrow+\theta_X=S.
\]

The corresponding direct, upstream, downstream, and base-commons compartments
are apportioned exactly from \(L_{b,m}\). At every state:

\[
L_{b,m}=P^{\mathrm{paid}}+R^{\mathrm{pending}}+X^{\mathrm{terminal}}.
\]

No payout is calculated as an addition to \(L_{b,m}\). There is no reward on a
reward, royalty on a royalty, or recursive mint. A production adapter MUST fail
a request with missing admission evidence, a reused receipt, or an omitted
admitted slot. The pure shadow kernel validates the supplied snapshot but
cannot establish its authoritative completeness. A slot that becomes expired,
invalidated, or otherwise non-payable after admission remains present with
`TERMINAL` disposition. Within a successful evaluation, that tombstone,
missing role weight, visible controller ineligibility, quotient residue, caps,
dust, and the depth tail move only toward the prospectively named commons or
refund route.

The reference shadow profile is illustrative, not a genesis value:

| Compartment | Parts per million | Share |
|---|---:|---:|
| direct milestone roles | 600,000 | 60% |
| load-bearing ancestors | 100,000 | 10% |
| independent descendants | 300,000 | 30% |
| base commons | 0 | 0% |

The reference shape leaves a majority with the current milestone, gives
foundations a meaningful but non-owning upstream share, and reserves a larger
directional share for independently realized consequence. It encodes gratitude
without perpetual rent and continuation without dispossessing present work.

To isolate branch geometry, the executable reference fixture uses domain
`general` revision `1`, a 1,000,000-PPM envelope cap, no program-window cap,
and zero minimum projected payout. Those neutral fixture values are not a
recommended production concentration or dust policy; both real caps and a
backed exposure ledger remain release gates.

The exact canonical reference-policy preimage is published in
[`constructive-intelligence-branch-flow.v1.json`](../../dashboard/public/standards/constructive-intelligence-branch-flow.v1.json)
and has digest
`sha256:cc8601fdc0efd8b0260a5a979fa456f43e45be7a3ebd821ca330b389ebb26684`.
The funded milestone is a separately bound request field; the reference
fixture uses E5.

Different artifact classes MAY use different prospective role matrices. They
MUST keep the conservation equation, publish the policy digest before
admission, and cannot change shares after seeing an outcome.

## 3. Existing milestone boundary

Branch flow does not change the v1 outcome-pool schedule:

| Milestone | Outcome-pool share |
|---|---:|
| E2 class verification | 15% |
| E3 effective independent reproduction | 20% |
| E4 challenge survival or tested repair/mitigation | 15% |
| E5 independent adoption or upstream disposition | 25% |
| E6 maintained conformance | 10% |
| challenge and remediation reserve | 15% |

Downstream consequence may shape only E5. Maintained consequence may shape
only E6. Novelty may shape only E3. A branch-flow compartment is a role split
inside a matured, funded tranche; it is never an additive bonus over the
10,000-basis-point outcome pool.

Every allocation request MUST bind exactly one funded milestone. Shadow v0
requires the downstream policy share to be zero for E2, E3, and E4; E5/E6
impact receipts MUST exactly match the funded milestone. The executable
reference fixture binds E5. A different milestone or split changes the policy
digest or canonical request bytes and cannot be substituted after evidence is
observed.

`ATTACKS`, `DISPROVES`, and `REPAIRS` evidence is paid, when eligible, from the
E4 or challenge/remediation compartments. It is not forced into the positive
descendant pool. `SUPERSEDES` preserves valid history and earns only the
independently reviewed marginal delta.

## 4. The canonical contribution graph

The economic subject is a semantic contribution cluster, not a transaction,
address, file, paper, theorem name, or the legacy `ParentFactId` overlay.

Orient an eligible edge from child to parent:

\[
v\rightarrow u
\]

means that contribution cluster \(v\) causally depends on cluster \(u\).

Every edge MUST be:

- pinned to immutable cluster and artifact revisions;
- typed by a reviewed causal relation;
- supported by challengeable dependency evidence;
- frozen before the relevant outcome is observed;
- acyclic and cycle-safe;
- bounded by graph, parent, traversal, and depth limits; and
- assigned a policy-derived weight rather than an author-selected royalty.

Raw citation, popularity, downloads, queries, address count, stake, reward
balance, KARMA, publication prestige, or a caller-supplied citation multiplier
has zero dependency weight.

For child \(v\), let \(a_{v,u}\in[0,S]\) be each pre-adjudicated raw dependency
weight. Normalize conservatively:

\[
A_v=\max\left(S,\sum_u a_{v,u}\right),
\qquad
\sigma_{v,u}=\frac{a_{v,u}}{A_v}.
\]

Shadow arithmetic v0 materializes that rational as the conservative fixed-
precision share

\[
\widehat{\sigma}_{v,u}
=
\left\lfloor\frac{S a_{v,u}}{A_v}\right\rfloor.
\]

Any normalization remainder is unattributed; it is not distributed by largest
remainder to the submitted parents. Each later graph multiplication likewise
rounds down at the PPM boundary, so lost precision never becomes claimant
weight. Changing that fixed-precision rule would require a new arithmetic
version.

Therefore:

\[
\sum_u\sigma_{v,u}\le1.
\]

For the single-origin upstream leg, the fixed \(S\) denominator makes that
unattributed weight terminal. The downstream leg instead settles a cohort of
independently evidenced descendants from their already materialized graph
shares. Graph-floor residue before that cohort is not a reserved terminal
claim: independent abundance may fill the already fixed absolute bucket. This
prevents a weak edge from diluting unrelated consequence. It never enlarges
the bucket, never becomes claimant weight itself, and any weight still
unattributed after cohort settlement goes to the terminal destination.
Forking, slicing, or adding parents can divide mass but cannot create it.

A future adjudicator may use proof-object dependencies, reproducible ablation,
exact implementation dependencies, or another class-specific counterfactual.
The branch-flow calculator merely checks and consumes bounded, version-pinned
weights. It does not determine whether the dependency evidence is true.

## 5. Absolute geometric depth

For direction \(r\in\{\uparrow,\downarrow\}\), policy freezes continuation
\(q_r\), where \(0\le q_r<S\), and maximum depth \(D_r\).

The absolute fractional weight of depth \(d\) is:

\[
g_{r,d}
=
\left(1-\frac{q_r}{S}\right)
\left(\frac{q_r}{S}\right)^{d-1},
\qquad 1\le d\le D_r.
\]

The unapplied tail is:

\[
g_{r,\mathrm{tail}}=\left(\frac{q_r}{S}\right)^{D_r}.
\]

Thus the depth weights and tail sum to exactly one. Consensus-candidate
integer weights use the common denominator \(S^{D_r}\):

\[
w_{r,d}=(S-q_r)q_r^{d-1}S^{D_r-d},
\qquad
w_{r,\mathrm{tail}}=q_r^{D_r}.
\]

Depth buckets are **absolute**. They MUST NOT be renormalized over the
recipients that happen to appear. A lone depth-five contribution cannot inherit
empty depth-one through depth-four buckets.

The reference shadow profile uses \(q_\uparrow=q_\downarrow=500{,}000\) and
\(D_\uparrow=D_\downarrow=5\):

| Destination | Direction-tranche share |
|---|---:|
| depth 1 | 50% |
| depth 2 | 25% |
| depth 3 | 12.5% |
| depth 4 | 6.25% |
| depth 5 | 3.125% |
| tail to commons/refund | 3.125% |

At \(q=0.5\), every depth receives exactly as much directional capacity as
the entire continuation cone beyond it. The nearest load-bearing relationship
is honored without erasing the possibility of deeper foundations or future
branches; locality and continuation remain in geometric balance.

With the reference 10% upstream compartment, this projects at most 5%, 2.5%,
1.25%, 0.625%, and 0.3125% of the enclosing tranche to successive depths; the
final 0.3125% tail remains terminal commons/refund capacity.

With the reference 30% downstream compartment, the corresponding absolute
ceilings are 15%, 7.5%, 3.75%, 1.875%, and 0.9375% of the enclosing tranche.
The final 0.9375% is the downstream tail. These are compartment ceilings, not
promised payouts: incomplete dependency, impact, role, or independence weight
can never enlarge the bucket. Its own claimant weight falls; an independently
abundant cohort may fill fixed downstream capacity, while any post-settlement
remainder increases the terminal amount.

## 6. Upstream dependency flow

Start one PPM unit of flow at the funded cluster:

\[
F_b^{(0)}=S.
\]

For each exact depth, shadow arithmetic v0 rounds every edge multiplication
down independently:

\[
F_u^{(d)}
=
\sum_{v:v\rightarrow u}
\left\lfloor
\frac{F_v^{(d-1)}\widehat{\sigma}_{v,u}}{S}
\right\rfloor.
\]

Because every node emits at most the flow it received:

\[
\sum_u F_u^{(d)}\le S.
\]

Each reached cluster divides its received flow among its pre-adjudicated
controller-role credits, whose sum is also at most one. Missing graph mass and
missing role mass remain commons. There is no winner normalization in the
upstream direction.

A valid dependency that is not itself payable MAY be `PASS_THROUGH`: it carries
flow to older dependencies but receives no allocation. An invalid dependency
is `BLOCKED`: it neither receives nor propagates positive flow. Those modes are
evidence-policy inputs, not decisions made by the allocator.

Path-specific flow is the v1 shadow rule. A cluster reached through multiple
causal paths may receive bounded mass at multiple exact depths. Every path
remains subject to the fixed depth bucket and controller caps, so convergence
cannot create an amount outside the envelope. A future first-hit rule would be
a new arithmetic version.

## 7. Downstream realization flow

A child link alone earns nothing. A downstream candidate requires:

- still-valid and marginally distinct semantic content;
- a frozen dependency path reaching the funded cluster;
- independent effective control where E5/E6 requires independence;
- the class-specific challenge and safety gates;
- a unique consequence receipt; and
- a policy-derived impact value \(m_j\in[0,1]\).

Those are admission conditions, not a caller-selectable scalar hard gate.
Shadow v0 either rejects the complete request or evaluates a closed cohort of
already admitted descendants. An unadmitted descendant contributes neither
capacity nor claimant weight; comparing two differently admitted cohorts can
therefore let independent abundance fill more of the same fixed bucket. Once a
descendant is admitted, its materialized pre-role capacity is fixed before
receipt-share floors, role completeness, and visible controller eligibility.
The pure calculator cannot prove that the caller supplied the complete closed
cohort. A production adapter MUST derive membership from authoritative
admission state and retain a `TERMINAL` tombstone for every admitted slot that
later becomes non-payable. Caller-controlled omission remains a release
blocker.

Any descendant credit controlled by a controller credited on the funded
cluster has zero downstream eligibility for that envelope. Its mass goes to
commons; independently controlled credits on the same descendant remain
separately evaluable. This observable controller-overlap rule is necessary but
not sufficient: hidden common control, funder dependence, or shared
infrastructure still belongs to the external independence adjudicator and
release gates.

Several economic receipts for one semantic descendant cannot clone its impact
mass. For raw receipt impact \(m^{\mathrm{raw}}_j\) attached to descendant
\(v\), shadow v0 first computes:

\[
Q_v=\min\left(S,\sum_{j\in v}m^{\mathrm{raw}}_j\right),
\qquad
M_v=\max\left(S,\sum_{j\in v}m^{\mathrm{raw}}_j\right),
\qquad
\widehat{m}_j=
\left\lfloor\frac{S m^{\mathrm{raw}}_j}{M_v}\right\rfloor.
\]

Thus every descendant supplies at most one PPM unit across its receipts.
Normalization loss remains commons. The \(m_j\) below denotes
\(\widehat{m}_j/S\), not the unbounded sum of issuer-selected records.

Every admitted receipt has an explicit disposition
\(e_j\in\{0,1\}\): `PAYABLE` is \(1\), while `TERMINAL` is \(0\).
Both remain in \(Q_v\) and consume their exclusive economic use.
`TERMINAL` is a cohort tombstone: it suppresses claimant lines without erasing
the capacity that was fixed when the slot was admitted.

Let \(r_{v,b,d}\) be the dependency flow from admitted descendant \(v\) that
reaches cluster \(b\) at exact depth \(d\). For receipt \(j\in v\) and
controller-role credit
\(\psi_{j,c,r}\), define the descendant capacity before receipt-share floors
and each independently eligible line:

\[
k_{v,d}=r_{v,b,d}\frac{Q_v}{S},
\qquad
w_{j,c,r,d}=r_{v,b,d}m_j\psi_{j,c,r}e_j.
\]

In shadow arithmetic v0, \(R_{v,b,d}\), \(\widehat m_j\), and
\(\Psi_{j,c,r}\) are PPM integers, while \(E_j=e_j\) is the unitless integer
zero or one. The conservative capacity and line weights are therefore:

\[
\widehat K_{v,d}
=
\left\lfloor
\frac{R_{v,b,d}Q_v}{S}
\right\rfloor,
\qquad
W_{j,c,r,d}
=
\left\lfloor
\frac{R_{v,b,d}\widehat m_j\Psi_{j,c,r}E_j}{S^2}
\right\rfloor.
\]

Let \(A\) contain only lines whose controller is independently eligible for
the funded cluster. Controller saturation and the terminal weight are:

\[
Z_{c,d}
=
\min\left(S,\sum_{(j,c,r)\in A}W_{j,c,r,d}\right),
\qquad
T_d
=
\sum_v\widehat K_{v,d}
-
\sum_{(j,c,r)\in A}W_{j,c,r,d}.
\]

Thus terminal admitted receipts, receipt-share normalization loss, missing
role mass, line-level fixed-point loss, and visibly ineligible controller
credit are terminal before cohort saturation. They cannot disappear merely
because other descendants are abundant. Eligible controller overflow above
\(S\) is removed by controller saturation rather than counted again as
terminal mass. Here
\(\widehat K_{v,d}\) begins from the materialized fixed-precision reliance at
depth \(d\); graph-floor residue that occurred before that reliance is not
introduced again as terminal capacity.

Let \(H_d=\sum_c Z_{c,d}\) and \(M_d=\max(S,H_d+T_d)\). Controller \(c\)'s
share of the absolute descendant depth bucket \(B_d\) is:

\[
P_{c,d}
=
B_d\frac{Z_{c,d}}{M_d},
\qquad
X_d^{\mathrm{terminal}}
=
B_d\frac{M_d-H_d}{M_d}.
\]

When eligible consequence plus terminal mass is below one PPM unit, the
unfilled remainder also stays terminal. When independently controlled
consequence is abundant, it shares the fixed bucket without enlarging it; any
already terminal mass keeps its proportional route to commons or refund. All
artifacts and addresses under one effective controller are aggregated before
the saturation and division.

Each controller's monetary amount is floored independently. Every residual
unit across controllers goes to the terminal destination; it is never awarded
to the next claimant by remainder rank. Within one controller's already fixed
amount, child clusters and roles divide value proportionally by deterministic
largest remainder. Splitting one contribution into more artifacts or addresses
can divide attribution but cannot increase the controller total.

## 8. Exclusive economic receipt use

One objective source event may appear in several epistemic records, but v1
permits exactly one economic use globally. A receipt consumption key binds at
least:

```text
source_system
source_record_or_event_id
source_revision
milestone
subject_digest
```

An issuer-selected receipt ID cannot reset the source tuple. A consumed key
cannot independently advance a child envelope and an ancestor's descendant-
impact envelope.

If one child is both a novel cluster and impact evidence for an ancestor, the
permitted v1 paths are:

1. the child's distinct E2/E3 receipts support its own verification and
   novelty, while one E5/E6 receipt is consumed by the ancestor-impact slot;
2. the E5/E6 receipt is consumed by the child's own E5/E6 slot and remains
   evidence-only for the ancestor; or
3. a prospectively funded coupled slot splits one fixed envelope between
   child and ancestor and consumes the receipt once.

Fractional reuse across independently funded envelopes is forbidden. Splitting
one receipt 50/50 across unrelated pools does not bound its total economic
effect when the pools differ in size.

A valid admitted receipt is consumed when its closed-window economic evaluation
succeeds, including a `TERMINAL` tombstone and even if controller overlap, a
cap, minimum payout, or weak impact routes its projected amount to commons. A
rejected request consumes nothing.
This prevents replaying a zero-valued result until it happens to encounter a
more favorable envelope or ordering.

## 9. Controller aggregation and caps

Addresses, profiles, validators, agents, artifacts, and roles under common
effective control MUST collapse before economic caps are applied.

For envelope \(L_b\) and per-envelope controller cap \(\beta_b\):

\[
C_b=\left\lfloor\frac{L_b\beta_b}{S}\right\rfloor.
\]

For program-window limit \(\Gamma_W\) and prior paid amount
\(D_{c,W}^{\mathrm{paid}}\):

\[
R_{c,W}=\max(0,\Gamma_W-D_{c,W}^{\mathrm{paid}}).
\]

Controller \(c\)'s final allowance is:

\[
Y_c=\min(X_c,C_b,R_{c,W}),
\]

where \(X_c\) is the controller's raw direct, upstream, and downstream claim in
the complete settlement cohort. If capped, \(Y_c\) is divided proportionally
over that controller's lines. Overflow goes to commons and is never
redistributed to other winners.

The pure shadow calculator can verify caller-supplied controller IDs and prior
exposure, but cannot establish their truth. Production remains blocked until a
merge-safe authoritative controller ledger and a persistent program-window
exposure policy exist.

## 10. Exact apportionment

All monetary and weight arithmetic uses non-negative integers. Floating point
is forbidden in a settlement implementation.

Policy-compartment and absolute depth-bucket boundaries have a prospectively
fixed set of destinations. For total \(T\), denominator \(M\), and canonically
keyed boundary weights \(w_i\) whose sum is exactly \(M\), they use Hamilton
largest remainder:

1. compute floor quotient and remainder of \(Tw_i/M\);
2. distribute residual units by descending remainder;
3. prefer the explicit commons entry on an exact tie;
4. then use fixed leg order `DIRECT`, `UPSTREAM`, `DOWNSTREAM`, `COMMONS`,
   followed by depth, immutable cluster, controller, role, receipt, and
   milestone in canonical ASCII order; and
5. never use map, transaction, proposer, or iterator order as a tie-break.

Claimant apportionment deliberately does **not** use Hamilton across
controllers. For controller weight \(Z_c\), each controller receives only:

\[
Q_c=\left\lfloor\frac{T Z_c}{M}\right\rfloor,
\qquad
Q_{\mathrm{terminal}}=T-\sum_c Q_c.
\]

This terminal residual includes the explicit terminal weight and every
cross-controller quotient remainder. Removing or invalidating a controller can
therefore never transfer a rounding unit to a competitor. Hamilton is used
again only inside one controller's already fixed \(Q_c\), with canonical line
keys, and when a cap divides that controller's already fixed allowance among
its own lines.

Minimum-payout dust is applied after controller caps and routes to commons. It
is not accumulated by a submitter-selected address or redistributed among
remaining claimants.

The final constructor MUST recompute and require:

\[
L_b
=
\sum_iP_i^{\mathrm{projected}}
+X_b^{\mathrm{commons}}.
\]

An error returns no partial result.

The executable request/result schema is `zerone.branch-flow-shadow/v0` and its
arithmetic identifier is `branch-flow-integer/v0`. The public
`zerone.constructive-intelligence-branch-flow/v1` document is the reviewed
architecture/profile wrapper; it is not a second calculator. Any arithmetic
change requires a new executable schema, algorithm identifier, golden result,
policy commitment where affected, and public-profile digest.

## 11. Modular authority

Branch flow is an allocator, not an authority.

| Concern | Target owner | Branch-flow boundary |
|---|---|---|
| canonical domain and evaluator revision | `x/ontology` | reads a pinned reference only |
| semantic clusters, artifact relations, evidence and verdicts | `x/knowledge` | accepts adjudicated inputs only |
| effective controller records and merges | `x/controller` | aggregates supplied canonical controllers |
| external merge, release, adoption and maintenance evidence | typed `x/substrate_bridge` adapter | consumes a replay-safe receipt; never trusts raw citation share |
| allocation arithmetic | pure `branchflow` package | no store, bank, clock, network, signer, or authority |
| research-fund custody and vesting | authority-gated `x/vesting_rewards` | receives one conserved entitlement after all gates |
| prospective policy and budget | SDK `x/gov` | may select policy and fund scope; cannot select winners |
| epistemic review | evidence-producing process | supplies a typed receipt; holds no second execution authority |

The target Cosmos SDK dataflow is deliberately one-way:

```text
SDK x/gov: prospective policy + funded scope
                         |
x/ontology + x/knowledge + x/controller + typed bridge receipts
                         |
              pure branchflow projection
                         |
future atomic settlement + x/vesting_rewards + x/bank
```

The pure step cannot call back into policy, evidence, identity, or governance.
The settlement arrow does not exist in v0; it is shown only to make the future
authority boundary reviewable.

No new consensus module is justified for Season 0. The reference package is a
standard-library-only, non-custodial shadow. Its output is a projection, not a
bank instruction or entitlement.

## 12. Separate ledgers and adapters

The same pure geometry MAY serve two different funded adapters, but their
ledgers and receipt keys remain separate:

1. **Outcome reward adapter** — divides a prospectively funded E2-E6 role
   compartment. It uses bounded outcome depth and routes the tail to the named
   terminal destination.
2. **Training-revenue adapter (TC6)** — divides actual realized manifest
   revenue across the exact bounded graph cone used by the manifest. It is not
   issuance, does not reopen outcome milestones, and must not erase included
   deep axioms merely because the outcome-reward reference depth is five.

For TC6, the manifest's protocol-bounded depth becomes the relevant maximum and
sub-minimum shares MAY accumulate in per-controller accounting until they
cross a payout threshold. That later adapter requires its own reviewed
settlement specification. This document does not implement it.

## 13. Legacy source disposition

The following existing code is evidence and compatibility surface only:

- `x/knowledge`'s single-parent reproduction overlay is not the canonical
  contribution DAG;
- `DistributeLineageRoyalties` is unreachable from a production knowledge
  reward path and is additive, non-atomic, and not controller-resolved;
- `x/substrate_bridge` lineage propagation is accounting-only and transfers no
  ancestor coins; and
- caller-provided citation types or contribution shares are not adjudicated
  branch-flow weights.

A future implementation MUST retire or explicitly quarantine those legacy
economic interpretations. It MUST NOT activate either helper as a shortcut to
this specification.

## 14. Monetary migration addendum

Before H4 can claim complete monetary reconciliation, its fixed census MUST
independently include:

- every `x/vesting_rewards` schedule, status, total, released, claimable,
  reserve, maximum-release value, beneficiary index, clawback record, and the
  exact module-account balance;
- the source and backing compartment for every schedule, including schedules
  created through knowledge and authority adapters;
- every active or expired claiming pot, claim record, committed-unit counter,
  lifetime-cap headroom dependency, optional registrar, and custom-staking
  eligibility dependency; and
- a deterministic disposition for every underfunded, unbacked, ambiguous, or
  archive-incomplete liability.

Creating a vesting record does not prove funds were reserved. A configured
mint cap does not prove an unclaimed pot is a funded liability. Neither may be
silently converted into an E2-E6 milestone or branch-flow entitlement.

The accepted single-writer rule remains unchanged: SDK `x/gov` is the sole
ordinary decision and executor. Any future epistemic chamber is an evidence or
target-handler input to one classified SDK proposal, not a second proposal,
veto, tally, or execution authority.

## 15. Required attacks and invariants

A candidate implementation MUST test at least:

- semantic paraphrase and artifact salami slicing;
- one controller split across addresses, roles, children, and ancestors;
- citation rings, reciprocal use, self-use, and correlated funders;
- multiple parents, convergent paths, sparse depths, and path padding;
- shortcut edges and cycles;
- weak descendant underfill and abundant descendant saturation;
- omission versus terminal disposition after descendant admission;
- duplicate and cross-envelope receipt consumption;
- transaction, map, input, and settlement-order permutations;
- controller merges before and after a projected allocation;
- per-envelope and rolling program caps;
- minimum-payout dust and exact tie cases;
- maximum bounded graphs and arithmetic values;
- honest supersession, refutation, fraud, expiry, and late descendants; and
- byte-identical replay and an independent implementation over golden vectors.

Required invariants include:

1. every compartment, depth bucket, and final envelope conserves exactly;
2. no controller exceeds its applicable caps;
3. adding artifacts or addresses under one controller cannot increase that
   controller's allocation;
4. adding descendants cannot increase the descendant compartment;
5. adding or lengthening a path cannot make the branch's total envelope grow;
6. a reused economic receipt always fails closed;
7. empty nearer depths never enrich a farther depth;
8. within one fixed admitted cohort, terminal disposition, visible controller-
   line ineligibility, missing roles, cross-controller rounding, caps, tail,
   and dust never enrich competitors; an invalid request returns no result,
   while a differently admitted cohort may refill but never enlarge its fixed
   downstream compartment;
9. branch-flow output cannot change governance, qualification, KARMA, truth,
   or ontology state; and
10. any error commits no state and emits no partial settlement.

## 16. Release gates

The model gate may pass while the integration gate remains closed. Value must
remain zero until all of the following independently pass:

- canonical semantic-cluster adjudication;
- typed causal edge and consequence receipt schemas;
- global economic receipt consumption;
- authoritative controller records, merges, and privacy-preserving assurance;
- class-specific dependency and impact scorers;
- fixed funding, compartment, liability, expiry, refund, and commons state;
- exact milestone and observation-window transitions;
- authoritative closed-cohort membership and terminal tombstones;
- envelope and persistent program-window controller caps;
- atomic role, commons, replay, liability, and bank settlement;
- backed `x/vesting_rewards` schedules and the monetary migration addendum;
- SDK-governance-only policy authority and non-economic governance activation;
- bounded performance, storage, replay, and denial-of-service analysis;
- two exact arithmetic implementations and golden cross-language vectors;
- adversarial economic simulation and independent review;
- upgrade, rollback, incident, and production verification plans; and
- a separately named network release authorized through the ordinary process.

Until then, the only correct live economic amount is zero.

## 17. Effect statement

This specification and its reference fixtures:

- do not read or change chain state;
- do not create a reward, debt, entitlement, quote, escrow, or token claim;
- do not mint, reserve, transfer, vest, burn, claw back, or distribute ZRN;
- do not recognize a breakthrough or decide that evidence is true;
- do not create qualification, KARMA magnitude, rank, identity, or authority;
- do not alter staking, governance, ontology, or controller state;
- do not register, schedule, deploy, or activate an upgrade; and
- do not claim that the present network satisfies the target authority model.

It makes one architectural promise testable: if value is ever admitted, love
for foundations and freedom for descendants must both fit inside one finite,
legible, non-extractive whole.
