# Constructive-Intelligence Rewards

> **Status: pre-consensus design specification; inactive everywhere.**
>
> This document defines a candidate mechanism that projects a versioned
> capability 技能樹 (skill tree) onto an artifact graph and bounded
> constructive-intelligence rewards. It is not a statement of implemented
> behavior, an entitlement, a governance decision, a token allocation, or
> activation evidence.

## Current network posture

As of 2026-07-29:

- the checked-in
  [constructive-intelligence tree v1](../specs/constructive-intelligence-tree-v1.md)
  is static experimental curriculum data with `authoritative`,
  `networkObserved`, and `rewardBearing` all `false`;
- there is no universal `x/work` module, live constructive-intelligence
  registry, semantic-equivalence service, breakthrough scorer, or payout path;
- the source doctrine in [`../USEFUL_WORK.md`](../USEFUL_WORK.md) is a target
  architecture with partial bindings, not a live universal reward formula;
- the `zerone-2` release remains **NO-GO**; and
- its planned genesis profile is protocol-dark: vote extensions are disabled,
  knowledge admission is priced above the hard supply cap, knowledge reward
  rates and allocations are zero, the substrate bridge has no adapters, and
  atomic H1 retires transaction-presence block issuance at the source level.

The dark posture is intentional. Adding this file MUST NOT change genesis,
module parameters, handlers, migrations, supply, governance authority, or the
`zerone-2` release decision. A future implementation requires its own reviewed
code, deterministic tests, adversarial simulations, state migration, release
packet, signed activation decision, and explicit activation height.

## 1. Purpose

Constructive intelligence is work that creates a verifiable increase in a
shared capability to reason, test, discover, or build. A breakthrough is not
whatever receives the most votes, citations, stake, or attention. In this
design it is a *marginal movement of a versioned evidence frontier*, measured
against semantically equivalent prior work and paid from a fixed,
pre-collateralized budget.

The mechanism is intended to make the following loop possible without letting
wealth or incumbency define truth:

```text
artifact
  -> reproducible evidence
  -> validity
  -> marginal novelty
  -> independent downstream consequence
  -> bounded, staged reward
  -> more capacity to create and test artifacts
```

The chain never observes “intelligence” directly. It observes artifacts,
evidence, dependency relations, independently resolved outcomes, and declared
control relationships. Every score in this design is therefore a fallible
measurement with a named scope and version.

### 1.1 Goals

1. Reward marginal, reproducible capability gains rather than claim volume.
2. Give formal mathematics a path where proof validity is mechanically checked
   and not decided by wealth, prestige, or majority taste.
3. Make salami slicing, address splitting, citation rings, and correlated
   agent swarms approximately payoff-neutral.
4. Preserve room for counterexamples, negative results, minority reports, and
   work outside incumbent paradigms.
5. Measure decentralization across every control surface, by controllers as
   well as addresses.
6. Keep all liabilities bounded, funded, replayable, and prospective.
7. Prevent reward, reviewer, token, and governance power from automatically
   converting into one another.
8. Expose assumptions and uncertainty instead of laundering them into one
   authoritative score.

### 1.2 Non-goals

This mechanism does not:

- decide ultimate truth, beauty, importance, or social value;
- turn a skill-tree node into a diploma, identity rank, or permanent account
  reputation;
- reward raw popularity, query count, citation count, effort, compute expense,
  token stake, or agreement with a majority;
- make anonymous controller uniqueness possible without an additional trust or
  scarce-resource assumption;
- replace ordinary research funding, peer review, open discussion, or
  permissionless publication;
- guarantee that every important result becomes legible within a short reward
  horizon;
- make a proof's correctness establish its novelty or significance;
- authorize minting, treasury expenditure, or a live-network deployment; or
- repair or activate any existing Zerone module by implication.

## 2. Objects and notation

An **artifact** is an immutable, content-addressed bundle. A revision is a new
artifact linked to its predecessors; it does not overwrite them. At minimum,
the bundle declares:

- content and manifest digests;
- author/controller commitments and any privacy-preserving attestations;
- claimed skill nodes and work class;
- source, data, tool, theorem, and artifact dependencies;
- build and execution environment;
- evidence objects and their schemas;
- licenses and redistribution constraints;
- conflicts of interest;
- scorer, registry, and policy versions under which it enters an epoch; and
- a reveal deadline and expiry policy.

Notation:

| Symbol | Meaning |
| --- | --- |
| \(a\) | an artifact version |
| \(C(a)\) | the semantic-equivalence cluster containing \(a\) |
| \(k\) | a versioned skill-tree node |
| \(e\) | a reward epoch |
| \(c\) | a conservatively inferred controller cluster |
| \(x_{a,k,m}\) | normalized evidence item \(m\), in \([0,1]\) |
| \(E_{a,k}\) | gated evidence score for artifact \(a\) at node \(k\) |
| \(H^{\mathrm{epi}}_{C,k,e}\) | highest still-valid matured evidence at node \(k\) |
| \(V^{\mathrm{epi}}_{C,e}\) | normalized aggregate epistemic frontier |
| \(K_C\) | snapshotted lifetime reward cap for cluster \(C\) |
| \(T_C(V)\) | cumulative gross-entitlement target at frontier \(V\) |
| \(A_{C,e}\) | irreversible economic accrued target |
| \(Q_{C,e}\) | new gross accrual from economic-target movement |
| \(Z_{C,e^-}\) | amount actually funded before epoch \(e\) |
| \(X_{C,e^-}\) | unfunded eligibility irreversibly extinguished before epoch \(e\) |
| \(D_{C,e}\) | current eligible demand, including scarcity backlog |
| \(F_{C,e}\) | amount actually funded and reserved in epoch \(e\) |
| \(B_e\) | fixed, escrowed reward budget for epoch \(e\) |

Consensus arithmetic, if ever implemented, MUST use specified integer
fixed-point operations, explicit rounding, checked overflow, and golden replay
vectors. The real-number equations below define shape, not an authorization to
use platform-dependent floating point.

## 3. Capability 技能樹 and artifact graph

Zerone now has a canonical static v1 capability graph at
[`../specs/constructive-intelligence-tree-v1.md`](../specs/constructive-intelligence-tree-v1.md),
with machine-readable policy in
[`../../dashboard/public/standards/constructive-intelligence-tree.v1.json`](../../dashboard/public/standards/constructive-intelligence-tree.v1.json).
This reward design binds to that graph; it does not create a parallel skill
ontology.

Two graphs remain deliberately distinct:

1. The **capability graph** is a versioned prerequisite DAG. Its v1 stages are
   `foundation`, `primitive`, `assurance`, `protocol`, and `quest`; its domains
   are `assurance`, `cryptography`, `mathematics`, `protocols`, `quests`,
   `security`, and `systems`. A skill unlock changes eligibility only.
2. The **artifact graph** holds immutable deliverables and the relations
   `ATTACKS`, `DEPLOYS`, `DISPROVES`, `IMPLEMENTS`, `MAINTAINS`, `PROVES`,
   `REPAIRS`, `REPLICATES`, and `SUPERSEDES`. Evidence, priority, frontier
   movement, attribution, and any funded reward attach here.

Capabilities therefore constrain who may attempt or review specialized work;
they are not balances, reputation multipliers, transferable credentials, or
proof that a particular artifact is correct. Conversely, a strong artifact
does not silently grant every prerequisite capability to its controller.

The current mathematics branch has one root and four qualification-only
nodes:

```text
math-proofcraft@1
├── math-algebra-finite-fields@1
├── math-probability-information-complexity@1
│   └── math-lattices-polynomial-rings@1
└── with security-threat-models-games@1
    and systems-exact-bytes-state-machines@1
    -> assurance-formal-verification@1
    -> quest-mls-state-invariants@1
```

That path illustrates the separation. `math-proofcraft@1` requires checkable
proof and counterexample practice; `assurance-formal-verification@1` requires
an exact theorem, assumptions, bounds, disclosed trusted base, and independent
checking. The quest at the end of this illustrated path is eligible for
prospectively funded sponsor milestones; all three v1 quest nodes use that
funding class. No v1 mathematics or assurance node earns money merely by being
unlocked.

Artifact evidence advances through the canonical `E0`–`E6` ladder. A
`SUPERSEDES` edge preserves valid historical work while locating a later
frontier; `DISPROVES` or an adjudicated invalid-evidence event can lower the
current epistemic frontier without rewriting history. Fraud, fabrication,
plagiarism, equivocation, or concealed conflicts remain distinct from an
honest result that is later corrected.

### 3.1 Breakthrough is a retrospective projection

Canonical v1 deliberately omits `breakthrough` from node stages and direct
reward inputs. Its derived recognition requires at least `E3`, an evidenced
delta against frozen prior art, and independently evidenced adoption or
descendant impact. It is not author-selected and creates no extra prize,
governance voice, qualification, or panel seat.

This document explores how a separately funded outcome pool might value
frontier movement without adding a popularity bonus. It cannot change the v1
facts that only quest nodes may use `sponsor-milestones`, external work
defaults to sponsor escrow, skill unlock creates no reward, and protocol
issuance remains separately gated to recursive useful work. A general
formal-mathematics bounty would require a new reviewed quest/template and
policy version; protocol-issued ZRN would additionally require the doctrine,
governance, implementation, and release work named later in this document.

### 3.2 Version binding and no silent forks

Every future entitlement MUST pin the capability-tree schema and policy
version, exact node identifiers, normative node digests, artifact-edge schema,
evidence ladder, disclosure lane, acceptance policy, and active standards
snapshots. An expired standards pin, unknown node, changed normative digest,
or reward request from a qualification-only node fails closed. Refreshing
descriptive standards status cannot mutate a historical entitlement, while a
normative node or policy change requires a prospective version.

## 4. Six ledgers that MUST remain separate

No majority, council, model, or scalar score may answer all six questions.
Each ledger is append-only, artifact-versioned, and independently queryable.

### 4.1 Validity ledger

Question: *Does the artifact do or establish what its declared method says?*

Examples include proof-kernel acceptance, reproducible execution, benchmark
integrity, or a correctly constructed counterexample. Validity records include
the exact method, axiom set, kernel/build hashes, evidence, result, and
challenge state.

For formal mathematics, deterministic checking controls the formal-validity
gate. A reviewer vote cannot override a rejected proof term, and a wealthy
reviewer cannot turn an unchecked argument into a checked theorem.

### 4.2 Novelty and priority ledger

Question: *What is new relative to semantically equivalent prior artifacts,
and when was possession credibly committed?*

This ledger holds semantic clusters, prior-art edges, simultaneous-discovery
cohorts, priority commitments, dependency declarations, disputes, and
adjudications. It does not determine validity or importance.

### 4.3 Significance and consequence ledger

Question: *What independently observed capability or downstream work became
possible?*

This ledger holds time-indexed milestones such as independent formalization,
challenge survival, use in a distinct theorem, successful replication, or
measured improvement on a precommitted evaluation. Raw citations, page views,
queries, votes, and self-use are descriptive signals only. They are not
themselves consequence milestones.

### 4.4 Attribution and credit ledger

Question: *Which artifact dependencies and controller-aggregated roles receive
what share of a funded contribution?*

This ledger holds dependency-DAG credit, role matrices, controller merges,
simultaneous-discovery shares, attribution disputes, and their effective
epochs. Attribution cannot establish validity, novelty, significance, funding,
or authority. Artifact and address multiplicity cannot create extra credit.

### 4.5 Funding and liability ledger

Question: *What bounded budget is collateralized, and which entitlements,
liabilities, transfers, expiries, and commons returns exist?*

This ledger holds epoch budgets, escrow balances, scorer and registry
snapshots, payout entitlements, caps, `paid_to_date`, expiries, commons
returns, and replay roots. Funding cannot rewrite the other ledgers. A valid
theorem does not compel an unbounded treasury payment.

### 4.6 Governance and authority ledger

Question: *Who may set policy, admit a release, authorize a budget, pause a
faulty mechanism, or execute a payout?*

This ledger holds proposals, approvals, chamber membership, conflicts,
activation heights, emergency actions, implementation hashes, and authority
delegations. Authority does not imply epistemic correctness: a governance vote
cannot make an invalid theorem valid, rewrite priority, assign itself credit,
or create an uncollateralized liability.

## 5. Admission gates and evidence score

Every node declares non-compensable gates. Let

\[
G_{a,k} =
I_{\mathrm{admissible}}\,
I_{\mathrm{provenance}}\,
I_{\mathrm{reproducible}}\,
I_{\mathrm{valid}}\,
I_{\mathrm{conflicts}}\,
I_{\mathrm{safety}}
\]

where every indicator is either zero or one under the snapshotted node policy.
`provenance` means that the required provenance statements were made and
verified to the declared assurance level; it does not require public legal
identity. `conflicts` means required disclosures and recusals are complete.
`safety` binds the recipient, artifact, action, authorization, and disclosure
lane. It is binary: stronger evidence above a passing safety threshold does
not multiply reward, while an unsafe claimant action cannot be compensated by
novelty, correctness, popularity, or cost.

For evidence dimensions selected by node \(k\), with
\(\sum_m w_{k,m}=1\), define:

\[
E_{a,k} =
G_{a,k}\,
\max\left(
0,\,
\frac{
\prod_m [\epsilon+(1-\epsilon)x_{a,k,m}]^{w_{k,m}}-\epsilon
}{
1-\epsilon
}
\right),
\qquad 0<\epsilon\ll1.
\]

The weighted geometric form makes weak required evidence difficult to hide
behind one spectacular proxy. Node definitions MUST keep normative choices
visible in the evidence vector and weights; a displayed total never replaces
the decomposition.

The following are prohibited evidence dimensions:

- submitter stake or fee above the admission bond;
- address count without controller analysis;
- raw vote margin;
- raw citations, paid queries, followers, or downloads;
- reviewer agreement with previous majorities;
- compute expenditure without an output-quality relation; and
- a scorer or benchmark chosen after seeing the artifact result; and
- a continuous safety bonus above the required binary safety gate.

System-wide decentralization is also not an artifact evidence dimension.
The required power surfaces in Section 10 are policy-owned admission,
funding, and settlement gates. They may fail closed or defer funding, but
once every threshold passes, stronger HHI, effective-count, or Nakamoto
readings MUST NOT scale an artifact's evidence score, cumulative target,
allocation weight, or role share. A change between passing global power
configurations without new artifact evidence therefore creates no new gross
accrual.

### 5.1 Formal gates before subjective scoring

For formal mathematics:

\[
G^{\mathrm{math}}_{a,k}=1
\]

only if the artifact declares its theorem, axiom universe, dependencies, proof
assistant and version, kernel hash, trust boundary, unsafe features, and build
environment; the committed proof replays deterministically; and all forbidden
placeholders or undeclared axioms are absent. A formal conjecture may enter an
exploration class, but it cannot pass a theorem-validity gate.

The protocol MUST expose common-mode dependence. Ten builds using one kernel
are ten reproductions but one kernel trust root. Higher assurance nodes SHOULD
require translation to, or checking by, an independently implemented kernel
where practicable. Any exception remains visible in the evidence vector.

Correctness, axiom relevance, novelty, generality, exposition, and consequence
remain distinct. Formal validity alone never earns a “breakthrough” label.

## 6. Epistemic frontier and irreversible economic accrual

Rewards concern frontier movement, not repeated arrival at the same result.
For a semantic cluster \(C\), node \(k\), and epoch \(e\), let
\(H^{\mathrm{epi}}_{C,k,e}\) be the highest still-valid matured evidence state
after adjudications. A pending claim cannot raise it. Unlike an economic
counter, this epistemic frontier can fall when evidence is revoked.

Define the normalized aggregate epistemic frontier using node weights
\(\eta_k\), with \(\sum_k\eta_k=1\):

\[
V^{\mathrm{epi}}_{C,e}
=
\sum_k\eta_k H^{\mathrm{epi}}_{C,k,e}.
\]

Before the funding cohort opens, policy assigns the cluster a class-bounded
lifetime cap \(K_C\). It is independent of submitter stake, fee, address count,
and artifact count. Let \(f_C:[0,1]\rightarrow[0,1]\) be a snapshotted,
monotone cumulative-reward curve with \(f_C(0)=0\) and \(f_C(1)\le1\). The
default reference model uses \(f_C(V)=V\). In consensus fixed-point notation:

\[
T_C(V)=\left\lfloor K_C f_C(V)\right\rfloor.
\]

The economic accrued target is separate and irreversible:

\[
A_{C,e}
=
\max\left(A_{C,e^-},T_C(V^{\mathrm{epi}}_{C,e})\right),
\qquad
A_{C,0}=0,
\]

\[
Q_{C,e}=A_{C,e}-A_{C,e^-}.
\]

Therefore:

\[
\sum_{e=1}^{n}Q_{C,e}
=
A_{C,n}-A_{C,0}
\le K_C.
\]

This remains true across revocation and recovery. With \(K_C=100\), a move
\(0\to1\), revocation \(1\to0\), and later valid recovery \(0\to1\) accrues
`100 + 0 + 0`, never `200`. Revocation can stop unpaid contingent tranches or
trigger an adjudicated fraud process, but it cannot lower \(A_C\) and thereby
make an already reached economic target payable again.

That conservatism creates a distinct **cap-poisoning** risk: a compromised or
mistaken maturation process could transiently raise \(A_C\) on invalid
evidence, then leave an honest successor unable to create new gross accrual at
the same frontier. The current design does not pretend to solve that tradeoff.
A production policy needs a bounded replacement or reattribution transition
that can move still-unpaid entitlement to later valid evidence without
lowering \(A_C\), reopening any paid amount, exceeding \(K_C\), or rewarding
the controller that caused the invalidation. That likely requires an explicit
quarantined or reattributable state distinct from \(X_C\): under the rule below,
an amount already moved into \(X_C\) is extinguished and cannot be reassigned.
Until a replacement transition survives adversarial review, cap-poisoning
resistance is a failed integration gate.

A jump from \(0\) to \(0.8\) and the same matured frontier traversed as
\(0\rightarrow0.4\rightarrow0.6\rightarrow0.8\) create the same cumulative
gross accrual, including deterministic integer rounding. The offline
executable stores only the normalized scalar \(A_C/K_C\), not the full
per-node epistemic vector or revocation ledger. That missing 技能樹 state is an
explicit integration failure rather than an inferred implementation.

## 7. Fixed budget and staged reward

Before epoch \(e\) opens, the funding ledger escrows a fixed budget \(B_e\).
There is no submitter-selected prize and no promise against expected future
revenue.

Let \(Z_{C,e^-}\) be the cumulative amount actually funded for cluster \(C\)
before the epoch. Let \(X_{C,e^-}\) be cumulative gross accrual that was never
funded and has been irreversibly extinguished by its precommitted expiry or an
adjudicated invalidation. Funded and extinguished amounts are disjoint and
MUST satisfy:

\[
0\le Z_{C,e^-}+X_{C,e^-}\le A_{C,e}.
\]

Current eligible demand includes only live scarcity backlog:

\[
D_{C,e}
=
\max(0,A_{C,e}-Z_{C,e^-}-X_{C,e^-}).
\]

Each \(Q_{C,e}\) creates a canonically ordered eligibility lot with a
snapshotted terminal deadline. When an unfunded remainder expires or is
invalidated, that exact remainder increments \(X_C\); neither later recovery
nor a repeated proposal can decrement \(X_C\). A revision cannot renew an
older lot's deadline merely by repackaging it. Only frontier movement above
the prior economic target creates a new lot. A production ledger therefore
stores the lots and their disposition, not only the three aggregate scalars.

If total eligible demand fits inside the budget, every cluster is funded in
full:

\[
\sum_C D_{C,e}\le B_e
\quad\Longrightarrow\quad
F_{C,e}=D_{C,e}.
\]

If demand exceeds the budget, concavity is applied **after** semantic
clustering. Let \(W_{C,e}=D_{C,e}^{\alpha}\), with \(0<\alpha\le1\), and choose
the greatest \(\lambda_e\ge0\) such that:

```text
F[C,e] = min(D[C,e], lambda[e] * W[C,e])
sum_C F[C,e] <= B[e]
```

This is deterministic capped water-filling: a small claim cannot receive more
than its gross demand, while one enormous claim cannot consume the full
reserve merely by scale. Applying the power transform to transactions or
artifacts before equivalence clustering would reward salami slicing and is
forbidden.

The funded amount \(F_{C,e}\) is atomically reserved against specific live
lots and
\(Z_{C,e}=Z_{C,e^-}+F_{C,e}\). The residual
\(D_{C,e}-F_{C,e}\) is public **eligible backlog**. It may compete under the
same snapshotted policy in a later explicitly funded epoch until its declared
expiry, but it is not reserved value, debt, vesting, or a guarantee that any
future budget will exist. Carrying the eligibility prevents a claimant from
turning one result into more funding merely by pacing its disclosure across
quiet epochs. Production scheduling MUST include eligible backlogs
deterministically rather than rely on proposer transaction order.

The offline executable models the conservative no-expiry case \(X_C=0\) and
requires the caller to resubmit a cluster for its backlog to compete. It does
not implement eligibility lots, terminal extinguishment, or automatic
scheduling; each is an explicit integration failure.

The unallocated amount is:

\[
B^{\mathrm{unallocated}}_e=B_e-\sum_C F_{C,e}.
\]

It remains in the named program reserve or follows a precommitted commons
route. It does not become a discretionary authority balance. Controller-cap,
missing-role, expiry, and invalid-entitlement overflow follows the same named
commons rule and is not redistributed to the remaining winners.

In the equations below, `commons` means the prospectively named
**non-claimant terminal route**, not necessarily a protocol treasury. For a
protocol-funded pool it may be a restricted program reserve; for a v1 sponsor
escrow it may be the pinned sponsor refund or reallocation account. The route
is immutable for the entitlement and never becomes discretionary winner
selection.

Each cluster allocation snapshots milestone weights satisfying
\(\sum_m\lambda_m=1\). Every milestone has exactly one state:
`pending`, `direct`, or `commons`. `direct` means its terminal outcome is
eligible for the role matrix; `commons` means its terminal outcome is
negative, expired, invalid, or otherwise ineligible for direct payment.
Terminal states cannot reopen.

Define cumulative targets for the settled envelope and its direct-eligible
sub-envelope:

\[
L^{\mathrm{settled,target}}_{C,e}(t)
=
F_{C,e}
\sum_m \lambda_m I[M_{C,m}(t)\ne\mathrm{pending}],
\]

\[
L^{\mathrm{direct,target}}_{C,e}(t)
=
F_{C,e}
\sum_m \lambda_m I[M_{C,m}(t)=\mathrm{direct}].
\]

Neither cumulative target is itself a transfer. Let
\(L^{\mathrm{settled}}_{C,e}(t^-)\) be the total envelope already processed
and \(L^{\mathrm{direct,processed}}_{C,e}(t^-)\) the direct-eligible envelope
already processed. The next terminal envelope and its direct-eligible portion
are:

\[
\delta L_{C,e}(t)
=
\max\left(
0,\,
L^{\mathrm{settled,target}}_{C,e}(t)
-L^{\mathrm{settled}}_{C,e}(t^-)
\right),
\]

\[
\delta L^{\mathrm{direct}}_{C,e}(t)
=
\max\left(
0,\,
L^{\mathrm{direct,target}}_{C,e}(t)
-L^{\mathrm{direct,processed}}_{C,e}(t^-)
\right).
\]

\(\delta L^{\mathrm{direct}}\le\delta L\). Section 11 deterministically divides
only the direct-eligible portion into controller-role transfers, while every
remaining unit goes to the named commons account. One atomic settlement MUST
satisfy:

\[
\delta L_{C,e}
=
\sum_{c,r}\delta P_{C,e,c,r}
+
\delta X^{\mathrm{commons}}_{C,e},
\]

\[
L^{\mathrm{settled}}_{C,e}
=
\sum_{c,r}P^{\mathrm{paid}}_{C,e,c,r}
+
X^{\mathrm{commons,paid}}_{C,e},
\qquad
L^{\mathrm{direct,processed}}_{C,e}
\ge
\sum_{c,r}P^{\mathrm{paid}}_{C,e,c,r}.
\]

The same transition sets
\(L^{\mathrm{settled}}(t)=L^{\mathrm{settled}}(t^-)+\delta L(t)\) and
\(L^{\mathrm{direct,processed}}(t)=
L^{\mathrm{direct,processed}}(t^-)+\delta L^{\mathrm{direct}}(t)\).
Only that single role/commons batch transfers value. Successful execution
increments all counters together; a failed transaction increments none. A
replay or repeated milestone resolution therefore computes
\(\delta L=\delta L^{\mathrm{direct}}=0\) and transfers zero.

### 7.1 Canonical tree-v1 escrow compartments

For any adapter consuming the current tree v1, the all-in escrow is partitioned
before admission:

\[
\mathrm{escrow}
=
\mathrm{verified\_cost\_budget}
+
\mathrm{outcome\_pool}
+
\mathrm{reviewer\_budget}
+
\mathrm{administration\_and\_fee\_budget}.
\]

\(B_e\) and \(F_{C,e}\) in this document refer only to the funded
`outcome_pool`; they cannot draw from the other compartments. `E0` creates
precedence only. `E1` may reimburse preapproved verified costs from its own
sub-cap, outside the percentage pool but inside the all-in escrow.

The v1 outcome-pool weights are exact:

| Terminal tranche | Weight |
| --- | ---: |
| `E2` class verification | 15% |
| `E3` effective independent reproduction | 20% |
| `E4` public challenge survival or confidential fix/mitigation testing | 15% |
| `E5` independent adoption or upstream disposition | 25% |
| `E6` maintained conformance | 10% |
| challenge and remediation reserve | 15% |

These weights instantiate \(\sum_m\lambda_m=1\). Novelty may shape only the
capped `E3` tranche, adoption or descendant impact only `E5`, and maintenance
only `E6`; they are not additive bonuses over the pool. The challenge reserve
may become direct-eligible for a compliant independent challenger or repairer.
Its unused remainder follows the pinned terminal route. Reviewers remain paid
from the separate reviewer budget, never from an artifact share they judge.

For a formal artifact, `E2` can require deterministic kernel replay and the
declared axiom policy, `E3` an effectively independent rederivation or replay,
`E4` survival of the declared challenge, `E5` independently controlled
downstream use, and `E6` continued validity across the specified interval or
version change. Levels are ordered, consume the required prior evidence, and
execute at most once. Passing time alone cannot advance an evidence level.

A later tree version may define a different class-specific schedule only
prospectively. The current exploratory executable does not consume tree-node
digests, typed receipts, sponsor escrow compartments, or these `E0`–`E6`
transitions; its continuous score is a shape experiment, not tree-v1 adapter
conformance. That integration gate remains closed.

Unresolved milestones remain
bounded liabilities until their explicit expiry. Expiry is a terminal
`commons` resolution, so no funded portion can remain forever unaccounted or
silently disappear. Honest later refutation resolves future tranches to
commons. Once a tranche is terminal, later evidence is a new ledger event
rather than a reopening of old entitlement. Only adjudicated fraud,
plagiarism, equivocation, or invalid provenance may trigger a slash or
clawback; a good-faith result later superseded or disproven is not
automatically fraud.

## 8. Semantic-equivalence clusters

The unit of reward is a semantic contribution cluster, not a transaction,
claim, file, theorem name, wallet, or paper.

For a funded quest, the canonical deliverable identity also binds exact
standards pins, scope, acceptance policy, and stable subject roots. Rebuilds,
wrappers, forks, scope splits, or a new bounty identifier declare overlap with
that lineage rather than resetting it. A global source-receipt consumption key
includes source system, source record/event ID, and source revision; an
issuer-selected evidence ID, fresh address, or second sponsor cannot make one
upstream merge, checker run, adoption receipt, or standards disposition count
twice.

For formal mathematics, clustering evidence includes:

- normalized propositions under alpha-renaming and definitional equality;
- declared axiom sets and mappings between them;
- implication or strict-generalization relations;
- proof-obligation and dependency graphs;
- theorem-prover normal forms where available;
- copied or trivially reordered proof structure; and
- decomposition of one argument into separately submitted lemmas.

Strict generalization is not treated as exact duplication: it receives only
its measured marginal movement over the cluster frontier. A genuinely
different proof technique for the same theorem may create verification or
tooling evidence without being paid again for statement novelty.

Automated clustering proposes; it does not silently finalize ambiguous
equivalence. Disputes use blinded, conflict-checked panels and are written to
the novelty ledger with evidence and appeal bounds. Cluster merges and splits
are forward adjudications with deterministic effects on unpaid tranches.
No cluster allocation is finalized before its semantic challenge window
closes; artifacts in the same epoch and simultaneous-discovery grace window
are considered together.

Let \(\mathcal{F}_e(\mathcal{A};\Pi)\) denote the total funded allocation
produced in epoch \(e\) from artifact multiset \(\mathcal{A}\), after semantic
clustering under the same ground-truth controller partition \(\Pi\). Let
\(\Sigma_{e^-}\) be the same prior accounting state in both executions, and
let \(\mathcal{D}_{c,\le e}(\mathcal{A};\Pi,\Sigma_{e^-})\) denote controller
\(c\)'s cumulative post-cap direct target through epoch \(e\). The required
split/merge invariants are:

\[
\left|
\mathcal{F}_e(\{a_1,\ldots,a_n\};\Pi)
-
\mathcal{F}_e(\{\operatorname{merge}(a_1,\ldots,a_n)\};\Pi)
\right|
\le
\varepsilon_{\mathrm{split}} B_e,
\]

\[
\sum_c
\left|
\mathcal{D}_{c,\le e}(\{a_1,\ldots,a_n\};\Pi,\Sigma_{e^-})
-
\mathcal{D}_{c,\le e}(
\{\operatorname{merge}(a_1,\ldots,a_n)\};\Pi,\Sigma_{e^-}
)
\right|
\le
\varepsilon_{\mathrm{split}} B_e.
\]

These bounds apply whenever the evidence, dependency graph, semantic content,
and controller set are otherwise equivalent. A production candidate MUST test
artifact splitting, address splitting, controller merging, lemma slicing,
paraphrase, alpha-renaming, and deliberately missed equivalence. The current
offline executable tests only **preclustered artifact-count invariance**: once
one equivalence cluster is supplied, changing its artifact count does not
change its reward. It does not discover semantic equivalence, so the broader
split/merge invariant remains an integration gate. If later evidence changes
\(\Pi\) by proving common control, direct payout may fall under the controller
cap; that is a correction of a failed independence assumption, not a
split-neutrality violation. Funded amount removed by the cap follows the
committed commons route.

## 9. Controller independence and correlated panels

An address proves control of a key, not independence of judgment. A controller
cluster is the most conservative supported grouping of addresses or agents
under common effective control. Relevant evidence includes:

- beneficial operator or organization;
- common signing or orchestration authority;
- funding and sponsorship, excluding unsolicited dust transfers;
- validator or cloud infrastructure;
- base model, checkpoint, system policy, and fine-tuning lineage;
- shared training/evaluation data likely to create common error;
- identity-attestation provider; and
- contractual or familial control where voluntarily or lawfully disclosed.

Funding correlation alone MUST NOT establish common control: an attacker can
send dust to another address. Controller links require evidence that the target
authorized, accepted under a declared relationship, or cannot plausibly avoid.
Participants need a bounded appeal and privacy-preserving attestation path.

No permissionless protocol can guarantee controller uniqueness from keys
alone. Douceur's [Sybil analysis](https://www.microsoft.com/en-us/research/publication/the-sybil-attack/)
shows that a logically centralized certification or another explicit
resource/coordination assumption is unavoidable in the general case. Zerone
MUST publish which assumptions each panel and metric uses.

Controller collapse happens before panel arithmetic. The matrix \(R\) is
indexed by a canonical ordering of the collapsed controllers, not by submitted
addresses or agent instances. Each controller has one unit of voice in the
offline reference model; any future non-unit \(w_i\) would have to be
policy-owned, independently justified, and unrelated to stake, reward balance,
address count, or proposer input. For such weights, effective panel size is:

\[
n_{\mathrm{eff}}
=
\frac{(\sum_i w_i)^2}
{\sum_i\sum_j w_iw_jR_{ij}}.
\]

\(R\) MUST be symmetric positive semidefinite, have unit diagonal, and use a
published fixed-point representation. Controller and family labels, matrix
version, and correlation floors belong to the snapshotted policy/attestation
state rather than the proposal. For quorum, unsupported negative correlations
are conservatively floored at zero.

For equal weights and common pairwise correlation \(\rho\):

\[
n_{\mathrm{eff}}=\frac{n}{1+(n-1)\rho}.
\]

Thus 100 reviewers with \(\rho=0.2\) provide approximately 4.8 independent
signals, not 100. Addresses under one controller are assigned perfect
correlation for quorum. Shared model and infrastructure lineages receive
conservative correlation priors until evidence supports a lower value.

Quorum MUST require all of:

- a minimum authenticated reveal count that aliases cannot satisfy by
  themselves;
- a minimum controller-cluster count;
- a minimum \(n_{\mathrm{eff}}\);
- per-controller and per-model-family caps; and
- no unresolved conflict that can change the outcome.

Tree v1 additionally requires at least three effective conflict clusters, two
organizational/control roots, two implementation or toolchain roots, and two
execution environments for high-stakes work. Its quality-capped
effective-cluster count and the correlation-aware \(n_{\mathrm{eff}}\) above
measure different failure modes; a future adapter requires both and cannot
average one healthy dimension over a failed floor. The simulator's illustrative
two-controller/`1.5` thresholds are deliberately below v1 and are not adapter
conformance.

Controller inference may reduce confidence or defer payout; it MUST NOT become
a hidden social-credit score or an automatic fraud accusation.

## 10. Decentralization is multi-surface

For every power surface \(s\), report both address shares and controller shares
\(p^{(s)}_c\):

\[
\mathrm{HHI}_s=\sum_c (p^{(s)}_c)^2,
\qquad
N^{(s)}_{\mathrm{effective}}=\frac{1}{\mathrm{HHI}_s},
\]

\[
\mathrm{NC}_s(q)
=
\min\left\{
|K|:\sum_{c\in K}p^{(s)}_c\ge q
\right\}.
\]

The Nakamoto threshold \(q\) is surface-specific: the actual acceptance,
censorship, multisig, budget, or governance threshold is used. Address-level
metrics are shown for diagnosis but never substituted for controller metrics;
controller aggregation can only reveal equal or greater concentration.

Every policy snapshot MUST provide exactly these twelve named surfaces and a
surface-specific coalition threshold:

1. `consensus-block-ordering` — consensus and block-ordering power;
2. `stake-governance` — stake and ordinary governance vote power;
3. `epistemic-review` — panel selection and verdict voice;
4. `scorer-registry-authorship` — scorer, benchmark, and skill-registry
   authorship;
5. `semantic-cluster-adjudication` — semantic-cluster judgment;
6. `controller-attestation` — identity and controller inference;
7. `proof-data-trust` — proof kernels, build systems, oracles, and data;
8. `proposal-authorship` — ability to place policy changes on the agenda;
9. `treasury-authorization` — budget and reserve authorization;
10. `reward-flow` — reward recipients and lineage royalties;
11. `model-families` — model providers and checkpoint families; and
12. `infrastructure` — validator, RPC, storage, and cloud infrastructure.

A missing or extra surface, a missing threshold, or a non-finite share fails
closed. These maps are protocol-policy inputs, never fields supplied by the
artifact proposer.

No single blended “decentralization score” is authoritative. A dashboard MUST
show the vector, time series, uncertainty, and joint-control graph. It also
computes the minimum distinct-controller cut for every path:

```text
define policy -> choose evaluator -> judge evidence -> authorize funds -> execute payout
```

Activation fails if one controller, or a coalition smaller than the configured
minimum, can complete that path even when each individual HHI appears healthy.
Empirical work on Compound, Uniswap, and ENS illustrates why token distribution
alone is not sufficient evidence of decentralized control
([Fritsch, Müller, and Wattenhofer](https://arxiv.org/abs/2204.01176)).

## 11. Payout roles and overflow

An allocation belongs to the artifact cluster. It is divided by a
class-specific, precommitted role matrix rather than an ad hoc authorship list.
Artifact-allocation roles may include:

- originator of the result or method;
- formalizer;
- proof, build, or replication producer;
- counterexample or falsifier;
- verification-tool author;
- load-bearing dependency ancestor;
- independent downstream confirmer.

Reviewers never receive a share of the artifact allocation they judge. They
are paid from a separate, pre-funded review budget.

Let \(\rho_r\) be the snapshotted cluster share for role \(r\), and
\(\psi_{c,r}\) the adjudicated contribution of controller \(c\) within that
role, where \(\sum_r\rho_r\le1\). For a newly direct-eligible envelope
\(\delta L^{\mathrm{direct}}_{C,e}(t)\) and
\(\sum_d\psi_{d,r}>0\), the raw batch claim is:

\[
x_{C,e,c,r}(t)
=
\delta L^{\mathrm{direct}}_{C,e}(t)\rho_r
\frac{\psi_{c,r}}{\sum_d\psi_{d,r}}.
\]

Before this arithmetic, artifacts, addresses, identities, and roles under
common control are merged. Let \(\mathcal{B}\) be every envelope in one
canonically defined settlement cohort, and let

\[
X^{\mathcal{B}}_{C,c}
=
\sum_{(e,t)\in\mathcal{B}}\sum_r x_{C,e,c,r}(t),
\qquad
D^{\mathrm{paid}}_{C,c}(t^-)
=
\sum_{e',r}P^{\mathrm{paid}}_{C,e',c,r}(t^-).
\]

Policy snapshots a per-controller, **cluster-lifetime** direct cap
\(\beta_CK_C\), independent of epoch count. Its remaining allowance is:

\[
R_{C,c}(t)
=
\max\left(0,\beta_CK_C-D^{\mathrm{paid}}_{C,c}(t^-)\right).
\]

The engine applies the remaining allowance once to each aggregate
\((C,c)\), then divides the allowed amount deterministically across the
cohort's roles and epochs. The real-valued reference is:

\[
Y^{\mathcal{B}}_{C,c}
=
\min\left(X^{\mathcal{B}}_{C,c},R_{C,c}(t)\right),
\]

\[
\delta P_{C,e,c,r}(t)
=
\begin{cases}
Y^{\mathcal{B}}_{C,c}
\dfrac{x_{C,e,c,r}(t)}{X^{\mathcal{B}}_{C,c}}
& X^{\mathcal{B}}_{C,c}>0,\\
0 & X^{\mathcal{B}}_{C,c}=0.
\end{cases}
\]

Consensus fixed-point code MUST aggregate the whole cohort before capping and
use a specified largest-remainder rule; transaction order cannot choose who
consumes the allowance. The commons leg is the exact residual:

\[
\delta X^{\mathrm{commons}}_{C,e}(t)
=
\delta L_{C,e}(t)
-
\sum_{c,r}\delta P_{C,e,c,r}(t).
\]

The difference
\(\delta L-\delta L^{\mathrm{direct}}\) routes terminal negative, expired, or
invalid milestone weight directly to commons. The rest of the commons
residual includes \(1-\sum_r\rho_r\), missing-role share,
cluster-lifetime-cap overflow, failed reveal, and rounding. It is not
redistributed among the remaining claimants, because doing so creates
incentives to exclude or challenge competitors. Each settlement has a replay
key binding entitlement, milestone/tranche, controller, role, arithmetic
version, and target state; the role transfers, commons transfer, and all
processed/`paid_to_date` increments are one atomic state transition.

The offline executable is only an accrual, funding, and controller-total
calculator. It does not implement milestone state transitions, role-level
tranches, bank transfers, or this atomic role/commons batch. Those omissions
remain failed integration gates even when its internal conservation checks
pass.

The cluster-lifetime cap does **not** stop one controller from accumulating
many distinct clusters. A value-bearing design additionally MUST define a
persistent, merge-safe program-wide controller exposure ceiling, including its
window or decay rule. This document and the exploratory simulator do not yet
choose that policy, so the integration gate remains closed. An optional epoch
rate limit is a different control: a merely rate-limited amount stays reserved
and deferred rather than being treated as lifetime-cap overflow and sent to
commons.

Review costs, refundable admission bonds, and breakthrough rewards are separate
flows. A larger bond never creates a larger reward. Admission bonds cover
bounded review harm and may be sponsored or waived through a separately
budgeted lottery so that wealth does not become an epistemic gate.

## 12. Reviewer incentives

Reviewers report:

1. a verdict within their assigned ledger and scope;
2. a probability distribution over independently resolvable outcomes;
3. a structured rationale or machine-checkable evidence reference;
4. conflicts and controller/model provenance; and
5. a sealed commitment before reveal.

For a later resolved outcome \(y\), use a bounded strictly proper score such as
the normalized multiclass Brier score:

\[
Q_j
=
1-\frac12\sum_o
\left(p_{j,o}-I[o=y]\right)^2.
\]

To preserve strict propriety, the resolved payment MUST be a **positive
affine transform** of that score:

\[
R_j=r^{\mathrm{cost}}_j+\gamma Q_j,
\qquad
\gamma>0.
\]

The cost reimbursement is fixed before the report and independent of its
value or outcome; \(\gamma\) and the maximum payment are bounded by the
pre-funded review budget. An arbitrary nonlinear transform, outcome-dependent
subsidy, or clipping rule is not equivalent and requires its own incentive
proof. In particular, a baseline-relative skill diagnostic
\(Q_j-Q_{\mathrm{baseline}}\) remains signed and is never clipped at zero or
used as the payment. Payment is not based on agreement with the panel
majority. Proper scoring rewards calibrated forecasts only when an outcome is
eventually observable; [Gneiting and Raftery](https://sites.stat.washington.edu/people/raftery/Research/PDF/Gneiting2007jasa.pdf)
give the formal foundations.

The scored outcome MUST be independent of that same panel's majority report;
otherwise the mechanism rewards coordination around its own verdict rather
than information about the world.

If the configured outcome never resolves, the reviewer receives only the
precommitted cost reimbursement and the bonus expires to the commons. Peer
prediction may be used as a research signal but not as ground truth:
uninformative equilibria remain possible and truthfulness depends on
distribution and multi-task assumptions
([Shnayder et al.](https://arxiv.org/abs/1603.03151)).

Minority opinions are not slashable. Slashing is restricted to objectively
provable equivocation, unauthorized copying, invalid provenance, or failure to
reveal after an accepted assignment. A future resolution may vindicate a
minority forecast and score it accordingly.

Panel construction uses:

- verifiable random selection with proof verification, not a caller-declared
  random output;
- equal or capped epistemic voice after qualification;
- blinded authorship and prestige where compatible with conflict checking;
- rotation and term limits;
- outsider and cross-domain seats;
- controller/model-family caps; and
- replacement panels for non-reveal.

The [Algorand sortition construction](https://eprint.iacr.org/2017/454.pdf)
illustrates the required public-key proof-verification property; copying only
its vocabulary is not sufficient.

## 13. Formal-mathematics path

Formal mathematics is the first candidate pilot class because validity can be
more sharply separated from significance than in many empirical domains.

### 13.1 Required bundle

A theorem artifact includes:

- a normalized human-readable statement;
- the machine statement;
- the complete axiom and trusted-computing-base declaration;
- proof term or proof source;
- proof-assistant, compiler, library, and kernel versions and hashes;
- dependency lockfile and content digests;
- deterministic build instructions;
- all generated code needed for replay;
- explicit use of classical axioms, choice, quotients, unsafe features, or
  external solvers;
- prior-art search and claimed delta;
- provenance and priority commitments; and
- license terms permitting the promised verification and reuse.

### 13.2 Verification stages

1. Manifest and dependency closure validate.
2. Reproducible builders replay the artifact from a clean environment.
3. The declared kernel checks the proof.
4. At least one independently controlled reproducer confirms the result.
5. Higher assurance attempts translation or replay through an independently
   implemented trust root.
6. Novelty is assessed against the semantic cluster.
7. Separate reviewers assess scope, axiom relevance, exposition, and plausible
   consequence.
8. Later milestones record independent reuse, generalization, boundary cases,
   and survived counterexample attempts.

Trivial inconsistency, hidden axioms, `sorry`-like placeholders, or an
unsound/permissive kernel fail validity. A stronger theorem derived by merely
adding an inconsistent axiom does not move the constructive-intelligence
frontier.

Counterexamples, impossibility results, simplifications, proof compression,
new proof techniques, and faithful formalization are first-class artifact
types. They need not pretend to be new theorem statements to earn their own
node-specific frontier movement.

## 14. Priority and front-running

A public mempool makes first-to-file rewards a copying and censorship target.
Transaction-ordering attacks are well documented
([Daian et al., *Flash Boys 2.0*](https://arxiv.org/abs/1904.05234)).

The priority protocol therefore requires:

1. an encrypted artifact bundle;
2. a salted commitment binding content, manifest, dependency, controller
   attestation, and epoch;
3. threshold or comparably plural decryption;
4. batch reveal after the epoch commitment boundary;
5. proof that reveal matches commitment;
6. censorship evidence and a bounded late-reveal recovery path; and
7. a grace window for near-simultaneous independent discovery.

A hash proves possession of bytes at or before a commitment, not authorship or
independent discovery. The novelty ledger may consider lab notebooks,
repository attestations, external timestamps, correspondence commitments, and
dependency provenance at explicitly declared assurance levels.

Block proposers and decryptors cannot receive priority merely by seeing
plaintext first. Near-simultaneous independent artifacts share the priority
cohort and use dependency/role attribution; the design rejects
winner-take-all first-transaction rewards.

## 15. Power must not freely convert

The mechanism enforces the following walls:

| Power held | MUST NOT automatically confer |
| --- | --- |
| token stake | epistemic voice, skill-node score, or larger reward |
| artifact reward | governance weight or reviewer qualification |
| reviewer accuracy | treasury control or scorer authorship |
| skill-registry authorship | permission to score one's own artifact |
| formal validity | significance, novelty, or unlimited funding |
| popularity or adoption | truth or panel voice |
| identity attestation | epistemic correctness |

Above a minimum bond needed to cover bounded harm:

\[
\frac{\partial\,\text{epistemic voice}}
{\partial\,\text{stake}}=0,
\qquad
\frac{\partial\,\text{artifact reward}}
{\partial\,\text{stake}}=0.
\]

Token-only vote weighting cannot simultaneously solve Sybil resistance and
plutocracy without a second assumption or resource
([Mohan, Khezr, and Berg](https://pubsonline.informs.org/doi/10.1287/mnsc.2023.01536)).
Consequently, ordinary stake governance alone MUST NOT be able to activate or
rewrite this mechanism.

At minimum, prospective changes require separate approval from:

1. a budget/security chamber responsible for solvency and consensus risk; and
2. an epistemic chamber constituted under a different power base, with
   sortition, qualification, controller caps, conflicts, rotation, and explicit
   identity assumptions.

Changes to constitutional invariants, the skill tree, scorer code, controller
policy, or reward shape require both chambers, an audit, a public simulation,
a timelock, and a new activation epoch. Emergency authority may pause new
admission or payout. It may not award funds, change a verdict, rewrite a
cluster, alter historical scoring, or redirect escrow.

No controller may simultaneously:

```text
author the scorer
select the panel
judge the artifact
resolve its challenge
authorize its budget
execute its payout
```

## 16. Liability, collateral, replay, and non-retroactivity

At every state transition:

\[
\sum_{\ell\in\text{open liabilities}}
\operatorname{maxPayable}(\ell)
\le
\operatorname{escrowBalance}.
\]

Creating an entitlement atomically reserves its maximum payable amount.
Failed reservation fails closed. Expected fees, future issuance, unpassed
governance proposals, or a general module balance do not count as collateral.

Each entitlement snapshots:

- epoch and state root;
- artifact, cluster, and dependency roots;
- capability-tree schema/policy version, exact node and normative digests,
  artifact-edge schema, and typed-receipt consumption keys;
- controller partition and correlation matrix version;
- skill registry, evidence schema, scorer code, weights, gates, and caps;
- milestone, challenge, eligibility-lot expiry, extinguishment, and role
  policies;
- budget source, escrow-compartment proof, and terminal commons/refund route;
- arithmetic and rounding version; and
- all governance approvals and activation identifiers.

Anyone with the committed inputs MUST be able to reproduce the same score,
cluster allocation, role allocation, vesting state, and commons overflow.
Consensus execution stores commitments to large evidence blobs and specifies
their availability rules; a content hash without retrievable evidence is not
replay.

Parameter or scorer changes apply only to artifacts entering a later epoch.
They never retroactively alter a submitted artifact's rules. New evidence
creates a new ledger event under the original policy or an explicitly accepted
new review; it does not mutate history. Upgrade migrations must preserve all
open-liability maxima and pass old/new shadow replay before activation.

## 17. Threat model

Assume adversaries may include:

- a whale or exchange splitting stake across addresses;
- elite reviewers building a reciprocal-acceptance cartel;
- a submitter paraphrasing, slicing, or self-citing work;
- a block proposer censoring and copying a visible artifact;
- a governance coalition changing scorers to favor its portfolio;
- a scorer author overfitting benchmarks or inserting a backdoor;
- a submitter or captured evaluator transiently maturing invalid evidence to
  poison a semantic cluster's irreversible economic cap;
- an AI operator spawning millions of correlated agents;
- multiple operators using the same model error or data contamination;
- bribers offering conditional off-chain payments;
- an identity attestor falsely merging, splitting, or censoring controllers;
- a proof kernel, compiler, oracle, storage layer, or threshold decryptor being
  compromised;
- reviewers refusing to reveal in order to kill quorum;
- an incumbent field excluding formally valid heterodox work;
- honest evidence whose consequence takes decades or never resolves; and
- capable optimizers directly gaming or tampering with reward inputs.

The strength of Goodhart effects grows with optimization pressure
([Manheim and Garrabrant](https://arxiv.org/abs/1803.04585)); capable agents can
also acquire instrumental incentives to tamper with reward processes
([Everitt et al.](https://arxiv.org/abs/1908.04734)). Published metrics and
scorers are therefore treated as attack surfaces, not neutral descriptions.

## 18. Mandatory adversarial simulations

No value-bearing pilot may activate until deterministic, independently
reproduced simulations cover at least:

1. **Whale split:** 40% of stake spread over 10,000 addresses and custodial
   intermediaries.
2. **Controller-hidden Sybil:** many addresses share one funder, host,
   orchestrator, and model checkpoint.
3. **Elite cartel flywheel:** top reviewers accept each other and reject new
   entrants for 500 rounds.
4. **Model monoculture:** 100 nominal reviewers repeat one correlated formal or
   conceptual error.
5. **Salami slicing:** one proof submitted as 1, 10, and 1,000 lemmas.
6. **Temporal slicing:** one frontier jump versus the same matured evidence
   divided across many epochs; cumulative gross accrual must be identical and
   pacing must not improve funding against identical competing cohorts.
7. **Semantic paraphrase:** alpha-renaming, reordering, definitional wrappers,
   changed notation, and trivial axiom changes.
8. **Strict generalization:** the cluster credits genuine delta without paying
   again for the included theorem.
9. **Priority theft:** a proposer observes, copies, censors, and reorders a
   claim across the reveal boundary.
10. **Simultaneous discovery:** independent artifacts arrive inside the grace
   window.
11. **Citation ring:** 100 controlled artifacts cross-cite and query each other
    against 20 independently controlled downstream uses.
12. **Incumbent rejection:** 80% of prestigious reviewers dislike a
    kernel-valid result outside their paradigm.
13. **AI flood:** one million cheap artifacts per day compete with ten honest
    high-cost submissions.
14. **Detector evasion:** a cartel randomizes timing and inserts theatrical
    dissent.
15. **Detector false positive:** deterministic proof checking produces honest
    unanimity and synchronized responses.
16. **Parameter capture:** a coalition proposes weights and scorers favoring
    artifacts it already controls.
17. **Conditional bribery:** a third party offers outcome-contingent payments;
    commit-reveal is not treated as sufficient bribe resistance.
18. **Identity failure:** the attestation provider is captured, unavailable,
    or sends malicious merge/split evidence.
19. **Dust poisoning:** unsolicited transfers attempt to make unrelated
    controllers appear correlated.
20. **Non-reveal griefing:** selected reviewers repeatedly kill quorum at
    bounded personal cost.
21. **Kernel compromise:** an unsound kernel, dependency substitution, hidden
    axiom, or permissive configuration accepts a false theorem.
22. **Unresolved horizon:** 90% of consequence milestones never resolve.
23. **Mass challenge:** every winning artifact is challenged at once; escrow
    remains solvent and honest work remains reachable.
24. **Reserve exhaustion:** gross demand exceeds the funded epoch reserve;
    capped allocation conserves the budget and creates no unfunded promise.
25. **Upgrade replay:** old and new binaries replay all open liabilities and
    produce byte-identical pre-activation results.
26. **Commons overflow:** caps, expiry, rounding, missing roles, and failed
    reveals return exactly the expected amount without discretionary capture.
27. **Backlog expiry:** an unfunded lot expires exactly once, increments
    extinguished-to-date, and cannot be revived by recovery, replay,
    repackaging, or a policy refresh.
28. **Tree-policy mismatch:** qualification-only nodes, stale node digests,
    wrong evidence levels, reused source receipts, and cross-compartment
    funding all fail closed.
29. **Cap poisoning:** false evidence transiently matures, is invalidated, and
    is followed by an honest equivalent result; bounded replacement preserves
    both successor reachability and the lifetime cap.

Simulation inputs, seeds, controller ground truth, policies, code hashes, and
full result distributions MUST be public. Passing averages is insufficient;
tail insolvency, censorship, queue delay, false-positive cartel detection, and
minority exclusion are release criteria.

## 19. Release invariants

Every staged release MUST prove:

1. **Dark by default:** absent an explicit activation record, admission and
   payout are impossible.
2. **Budget balance:** maximum open liability never exceeds dedicated escrow.
3. **Split/merge neutrality:** equivalent decomposition changes total reward
   by at most \(\varepsilon_{\mathrm{split}}B_e\).
4. **Temporal no-pacing advantage:** for a fixed matured frontier and lifetime
   cap, cumulative gross accrual is independent of how many epochs carried the
   intermediate evidence, and pacing cannot increase cumulative funding or
   direct allocation against the same cohorts and funded horizons. Scarcity
   backlog may differ with cohort timing and is reported explicitly;
   revocation and recovery cannot reaccrue an economic target already reached.
5. **Wealth non-dominance:** stake above the minimum bond changes neither
   epistemic voice nor artifact reward.
6. **Controller-aware quorum:** no panel passes solely through address or
   agent-instance multiplicity.
7. **Correlation awareness:** the release threshold is applied to
   \(n_{\mathrm{eff}}\), not nominal reviewer count alone.
8. **Validity separation:** a funding or popularity vote cannot override a
   deterministic formal-validity failure.
9. **Minority safety:** a good-faith losing opinion is not slashed and remains
   eligible for later vindication.
10. **Newcomer reachability:** a capable new participant has a finite,
   published, non-incumbent-controlled path to submission and panel
   eligibility.
11. **Priority fairness:** censorship or mempool visibility cannot transfer
    sole priority to the ordering controller.
12. **Power separation:** the minimum distinct-controller cut across
    policy-to-payout paths meets the configured floor.
13. **Prospective policy:** scorer and parameter changes do not affect already
    admitted artifacts.
14. **Replay determinism:** independent implementations reproduce all
    consensus-relevant outputs from committed state and available evidence.
15. **Resolve or expire:** every contingent tranche has a terminal resolution
    or expiry rule.
16. **Commons conservation:** all unallocated, capped, expired, and invalid
    amounts return to the named commons account exactly once.
17. **Bounded queue harm:** attacker cost covers the marginal scarce review
    work it causes, while a separate sponsored route preserves access for
    contributors without wealth.
18. **No hidden single trust root:** proof-kernel, identity, storage,
    decryption, and scorer common modes are reported and stay within release
    caps.
19. **No governance shortcut:** neither ordinary stake vote nor emergency
    authority alone can activate, award, or rewrite the mechanism.
20. **Single settlement:** every unlocked envelope equals its atomic
    controller-role transfers plus its commons transfer; no second transfer
    exists at the envelope layer.
21. **Persistent controller exposure:** cluster-lifetime and program-wide
    controller ceilings survive epoch boundaries, replay, cluster merge, and
    address merge without resetting.
22. **Canonical tree binding:** a skill unlock or qualification-only node
    cannot create reward; every funded transition verifies the exact
    prospective quest, policy, node, receipt, and escrow-compartment digests.
23. **Backlog terminality:** every gross-accrual lot is eventually funded or
    irreversibly extinguished; expiry cannot be reset by repackaging, and
    \(Z_C+X_C\le A_C\) always holds.
24. **Cap-poisoning resistance:** invalid maturation cannot permanently
    exclude a later valid successor from still-unpaid cluster capacity, while
    replacement cannot reopen paid value, reward the invalidating controller,
    or increase the cluster-lifetime cap.

Threshold values are not set in this document. Choosing them is part of the
future reviewed activation design and must be justified by simulations rather
than copied from aesthetic token numbers.

## 20. Staged rollout

### Stage 0 — specification only

The static tree v1 and this reward-shape specification exist. Tree v1 remains
non-authoritative, non-network-observed, non-reward-bearing curriculum data;
this document changes no registry, genesis value, reward, treasury route, or
activation state.

### Stage 1 — offline reference model

**Started, not complete.** The deterministic, standard-library-only simulator
at [`../../tools/constructive-rewards/`](../../tools/constructive-rewards/)
implements one exploratory score/reward model, a state-owning replay engine,
submitted/controller/effective-signal quorums, exactly twelve policy-owned
power surfaces, scalar irreversible economic accrual, scarcity backlog,
named invariants, adversarial comparators, and an
alpha/cluster-lifetime-cap sweep. Its complete in-memory snapshot owns
parameters, cluster caps and credit partitions, economic high-water marks,
funded/direct counters, and replay IDs. It uses no on-chain state and no
transferable value. It does not consume canonical tree/receipt digests,
`E0`–`E6` escrow compartments, or v1 independence/disclosure floors, and it
does not implement the per-node revocable 技能樹, expiring eligibility lots,
cap-poisoning replacement, automatic backlog scheduling,
milestone/role/commons settlement, or a program-wide controller ceiling. The
first results and the corrections found by adversarial review are recorded in
[`../simulation/CONSTRUCTIVE-REWARD-SWEEP.md`](../simulation/CONSTRUCTIVE-REWARD-SWEEP.md).
Stage 1 still requires a second independent implementation,
semantic/controller ground-truth fixtures, a curated historical corpus, and
external red-team reproduction.

### Stage 2 — shadow ledgers on an isolated devnet

Record artifacts, six-ledger events, clusters, controller uncertainty,
counterfactual rewards, liabilities, and decentralization metrics. Payout is
hard-disabled. Run red-team panels and replay every epoch.

### Stage 3 — public testnet with non-transferable test credits

Open adversarial participation, identity appeals, semantic-cluster disputes,
priority commitments, non-reveal recovery, and long-horizon expiry. Test
credits have no conversion promise to ZRN or future governance rights.

### Stage 4 — capped sponsor-funded formal-mathematics quest

Tree v1 contains qualification-only mathematics nodes, not a general
reward-bearing mathematics quest. The source-only
[Math Frontier v0](../specs/constructive-intelligence-math-frontier-v0.md)
now supplies a separate zero-value formal-construction quest template without
mutating the digest-pinned core tree. It is not the pilot or its activation:
an actual packet still requires independent review, a prospective policy
version, a small pre-funded sponsor escrow, controller-resolved evidence and
an explicit sunset. It remains opt-in and off the protocol-issuance path;
budgets, cluster caps, controller caps, and total liabilities must be
deliberately small. If even minimal settlement is later projected onto
Zerone, `zerone-2` and the adapter each require their own audited release and
signed GO decision. Ordinary knowledge rewards and governance remain
untouched.

### Stage 5 — bounded class expansion

Each new skill branch requires a class-specific evidence schema, scorer,
threat model, simulation corpus, independent audit, two-chamber approval,
timelock, and prospective activation epoch. No success metric automatically
expands the budget.

### Stage 6 — continued constitutional review

There is no autonomous “fully activated” endpoint. Every budget increase,
identity assumption, scorer revision, or power-surface change remains visible,
bounded, replayed, and reversible for future admission. Historical ledger
records remain forward-only.

## 21. Unresolved design questions

These are blockers for implementation, not invitations to fill gaps with
defaults:

1. What controller-attestation system can provide useful assurance while
   preserving anonymity, resisting censorship, and preventing dust poisoning?
2. Who adjudicates ambiguous semantic equivalence, and how are that panel's
   own controller, paradigm, and model correlations bounded?
3. What formal relation distinguishes exact equivalence, strict
   generalization, independent proof technique, and merely shared vocabulary
   across proof assistants?
4. Which proof kernels or cross-kernel translations are sufficiently
   independent for higher assurance?
5. Which consequence milestones are observable for mathematics, over what
   horizons, and what fraction should expire rather than resolve?
6. How are dependency and role credits assigned without making authorship
   declarations or Shapley-like attribution another Sybil surface?
7. Which budget source funds verification cost, breakthrough allocations,
   challenges, and commons reserves without creating unfunded issuance?
8. What minimum controller cut, \(n_{\mathrm{eff}}\), HHI, and Nakamoto floors
   are required on each power surface?
9. What persistent program-wide controller exposure ceiling, window or decay
   rule, and merge treatment limit cross-cluster capture without excluding
   prolific good-faith contributors?
10. How is the epistemic chamber constituted without disguising an identity
   provider or incumbent guild as decentralization?
11. What encrypted-commitment and threshold-reveal scheme provides fair
    ordering, availability, recovery, and practical proof sizes?
12. How are anonymous or pseudonymous collaborators credited when controller
    disclosure is incomplete?
13. Which independently resolvable outcomes make proper reviewer scoring
    legitimate, and what happens when resolution is disputed?
14. What is the maximum honest review cost an admission bond must cover, and
    how is the sponsored path protected from capture?
15. How large can evidence availability obligations grow before replay becomes
    its own centralization pressure?
16. What governance action can pause a discovered exploit without gaining the
    ability to select winners or redirect escrow?
17. What reviewed formal-mathematics quest and receipt schema can bind the
    qualification-only v1 nodes without turning skill attainment into reward?
18. Which eligibility-lot horizon and extinguishment rule prevents both
    immortal scarcity backlog and deadline renewal through paced revisions?
19. How can still-unpaid capacity be reassigned after false maturation without
    reopening settled value or giving cap poisoners a profitable challenge
    weapon?

Until these questions have reviewed answers and the release invariants pass,
the correct reward value is zero.

## References

- John R. Douceur,
  [*The Sybil Attack*](https://www.microsoft.com/en-us/research/publication/the-sybil-attack/)
  (2002).
- Vijay Mohan, Peyman Khezr, and Chris Berg,
  [*Voting with Time Commitment for Decentralized Governance*](https://pubsonline.informs.org/doi/10.1287/mnsc.2023.01536)
  (2024).
- Robin Fritsch, Marino Müller, and Roger Wattenhofer,
  [*Analyzing Voting Power in Decentralized Governance*](https://arxiv.org/abs/2204.01176)
  (2022).
- Tilmann Gneiting and Adrian E. Raftery,
  [*Strictly Proper Scoring Rules, Prediction, and Estimation*](https://sites.stat.washington.edu/people/raftery/Research/PDF/Gneiting2007jasa.pdf)
  (2007).
- Victor Shnayder, Arpit Agarwal, Rafael M. Frongillo, and David C. Parkes,
  [*Informed Truthfulness in Multi-Task Peer Prediction*](https://arxiv.org/abs/1603.03151)
  (2016).
- David Manheim and Scott Garrabrant,
  [*Categorizing Variants of Goodhart's Law*](https://arxiv.org/abs/1803.04585)
  (2018).
- Tom Everitt, Marcus Hutter, Ramana Kumar, and Victoria Krakovna,
  [*Reward Tampering Problems and Solutions in Reinforcement Learning*](https://arxiv.org/abs/1908.04734)
  (2019).
- Philip Daian et al.,
  [*Flash Boys 2.0*](https://arxiv.org/abs/1904.05234)
  (2019).
- Yossi Gilad et al.,
  [*Algorand: Scaling Byzantine Agreements for Cryptocurrencies*](https://eprint.iacr.org/2017/454.pdf)
  (2017).
