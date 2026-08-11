# Constructive-Intelligence Reward Sweep

> **Status:** deterministic pre-consensus experiment, updated 2026-08-11
> **Reviewed base:** `github/main` at `0558c915e34acc11ed681795ab595240018b0e76`
> **Value-bearing release:** **closed**

This report records the first executable pass over the mechanism in
[the design specification](../tokenomics/CONSTRUCTIVE-INTELLIGENCE-REWARDS.md).
The accepted exact shadow allocation architecture is
[Constructive-Intelligence Branch Flow v1](../specs/constructive-intelligence-branch-flow-v1.md).
The model is in
[`tools/constructive-rewards/`](../../tools/constructive-rewards/).
It reads no chain state and moves no value.
It is a reward-shape experiment, not an adapter for the static
[constructive-intelligence tree v1](../specs/constructive-intelligence-tree-v1.md);
tree/typed-receipt and exact `E0`–`E6` escrow binding remain failed integration
gates.

## Commands

```sh
go test ./tools/constructive-rewards -count=1
go test ./tools/constructive-rewards/branchflow -count=1
go run ./tools/constructive-rewards -mode report
go run ./tools/constructive-rewards -mode sweep
go run ./tools/constructive-rewards -mode branch-flow
go run ./tools/constructive-rewards -mode release
```

The test, report, sweep, and branch-flow commands pass. The release command
intentionally exits `1` because model success is not production integration.

## Exact branch-flow shadow kernel

The `branchflow` subpackage is a standard-library-only, non-custodial reference
kernel for one already funded role envelope. It is deliberately separate from
the exploratory `float64` score and scarcity sweep. Its projection entrypoint
is `Allocate(Request) (Result, error)`. The `-mode branch-flow` CLI adapter runs
a golden-locked E5 reference fixture with the digest-pinned policy; only its
illustrative exact-decimal envelope can be changed with
`-branch-envelope-uzrn`. There is no module or chain integration. Every result
retains:

```text
assurance = SHADOW_ONLY
economic_effect = NONE
```

The reference policy apportions the fixed role envelope as 60% direct, 10%
upstream, 30% downstream, and 0% base commons. Both directional compartments
use absolute continuation (q=0.5) through depth five: `50%`, `25%`, `12.5%`,
`6.25%`, and `3.125%`, followed by a `3.125%` tail to the named commons or
refund route. Empty depths are not renormalized. Base commons being zero does
not prevent admitted terminal tombstones, missing roles, visible controller
ineligibility, rounding, caps, dust, or tail from reaching the terminal route.
Invalid requests fail without a projection.

The kernel uses exact non-negative integer arithmetic. Fixed policy and depth
boundaries use deterministic largest remainder with a commons-first exact
tie-break; claimant controllers receive conservative floors and every cross-
controller residual routes terminal. It validates bounded
child-to-parent DAGs, per-node flow conservation, controller-aggregated role
credits, E5/E6 consequence receipts and terminal tombstones, prior economic
receipt use, and caller-supplied cohort/envelope/program exposure snapshots. It
cannot establish that those inputs are complete or true and performs no store,
bank, clock, network, signer, governance, or vesting action.

Its unit tests, golden JSON fixture, and two fuzz targets cover exact envelope
and depth-bucket conservation, the 60/10/30/0 default, sparse and convergent
exact depths, fixed-precision graph traces and exposed zero-share edges,
receipt replay and exclusive slots, controller aggregation, saturated-cohort
and rounding non-enrichment for terminal, excluded-controller, or missing-role
mass, envelope/program caps, minimum dust, deterministic input-order and byte
replay, exact declared bounds,
cycle/limit/validation refusal, and fuzzed conservation and permutation
invariance. These checks make one arithmetic shape executable. They are not a
second independent implementation, a canonical artifact-graph adjudicator, an
atomic settlement, or activation evidence.

The kernel applies only inside an already funded milestone-role compartment.
It does not alter the E5 25% or E6 10% outcome-pool tranches, and the
retrospective `breakthrough` projection creates no separate prize. A future
TC6 adapter would divide realized training-manifest revenue under a separate
ledger and receipt namespace; it cannot reuse this outcome adapter's liability
or reopen a settled milestone.

## Main adversarial results

| Attack or diagnostic | Naive/address-level result | Cluster/controller result |
| --- | ---: | ---: |
| 40% whale, one address | 44.95% square-root-stake share | not an epistemic input |
| Same whale, 100 aliases | 89.09% share, a 1.98x gain | aliases retain one controller |
| One preclustered result, one artifact | 50.00% of a two-result artifact pool | 820.493541 funded; 200.000000 direct; 620.493541 commons |
| Same preclustered result, 100 artifacts | 99.01% of the naive pool | identical funded/direct/commons amounts |
| 100 reviewers, pairwise \(\rho=0.2\) | nominal count 100 | \(n_{\mathrm{eff}}=4.807692\) |
| 100 linked aliases | nominal count 100 | \(n_{\mathrm{eff}}=1\) |
| Two-controller panel, one versus 100 aliases | nominal count changes by 99 | \(n_{\mathrm{eff}}=2\) and score `0.738716053` in both cases |
| Mixed cartel viewed by address | effective power count 21.739 | controller truth count 2.500 |

Three conclusions are already hard:

1. Per-address concavity is not Sybil resistance. It nearly doubles the
   illustrative whale's share when the whale splits.
2. Artifact-count rewards are salami machines. Equivalence clustering must
   happen before any concave transform or budget allocation.
3. Address count and nominal panel size can overstate independent power by an
   order of magnitude.

The artifact result is conditional: the executable is handed the equivalence
cluster. It demonstrates count invariance *after clustering*, not an ability to
recognize a paraphrase, lemma split, or disguised duplicate. Semantic
adjudication remains a failed integration gate.

## Temporal, revocation, and scarcity correction

The first algebraic sketch used a high-water delta divided by a fresh fixed
epoch budget. That prevents an exact replay from paying again, but it does not
prevent a controller from pacing one improvement across several otherwise
empty epochs and draining each epoch's budget.

The revised design separates a revocable epistemic frontier from an
irreversible economic accrued target:

\[
T_C(V)=K_C V,
\qquad
A_{C,e}=\max(A_{C,e^-},T_C(V_{C,e})),
\qquad
Q_{C,e}=A_{C,e}-A_{C,e^-}.
\]

where \(K_C\) is a fixed cluster-lifetime cap. Revocation can lower the
epistemic state, but not \(A_C\); later recovery therefore cannot accrue the
same target twice. Scarcity eligibility is:

\[
D_{C,e}=\max(0,A_{C,e}-Z_{C,e^-}-X_{C,e^-}),
\]

where \(X_C\) is unfunded eligibility irreversibly extinguished by a
precommitted expiry or invalidation. An underfunded live amount remains
eligible in later funded cohorts rather than disappearing. It is public
backlog, not debt, reserved value, or a promise of future budget.

In the deterministic fixture, one jump creates `797.681706260` units of gross
accrual; the same matured frontier traversed in three steps creates exactly the
same amount. With an abundant budget both paths reach `200.000000000` direct
units and `597.681706260` calculator commons. With only `100` available in
each of three epochs, retrying the jump backlog and pacing the evidence both
reach `200.000000000` direct and `100.000000000` commons after `300` total
funding. A separate regression checks `0→1→0→1` accrues the target only once.

Backlog itself is not generally path-independent under scarcity. With the
same two funded horizons and the same competing cluster, the jump path funds
`100.072890865` and leaves `697.608815395` live backlog; the paced path funds
`93.317652754` and leaves `704.364053506`. The required property is therefore
equal cumulative gross accrual and **no pacing advantage**, not equal funding
or backlog across different cohort timing.

This result is conditional on re-evaluating the backlog and on the
simulator's no-expiry case \(X_C=0\). The executable contains neither a
production scheduler nor eligibility lots and extinguishment, so both
integration gates remain closed.

## Parameter grid

The sweep holds an illustrative epoch budget at `1000` and varies scarcity
concavity \(\alpha\) and the maximum cumulative direct share for one controller
inside one cluster. Four clusters generate `3156.117` units of eligible
demand, so every cell has `2156.117` units of explicit unfunded backlog. That
amount is diagnostic only; it is neither collateralized liability nor a
promise.

One controller owns two separate fixture clusters. “Aggregate direct” therefore
shows how cross-cluster exposure can exceed the cluster cap; “cluster-cap use”
is the maximum utilization of any one controller's cap inside any one cluster.

| \(\alpha\) | Cluster cap | Aggregate direct | Cluster-cap use | Newcomer | Commons | Budget error | Invariants |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | :---: |
| 0.50 | 0.15 | 0.3000 | 1.0000 | 0.1500 | 0.4000 | \(1.14\times10^{-13}\) | pass |
| 0.50 | 0.20 | 0.4000 | 1.0000 | 0.2000 | 0.2000 | \(1.14\times10^{-13}\) | pass |
| 0.50 | 0.25 | 0.5000 | 1.0000 | 0.2415 | 0.0392 | 0 | pass |
| 0.50 | 0.33 | 0.5392 | 0.8340 | 0.2415 | 0.0000 | 0 | pass |
| 0.65 | 0.15 | 0.3000 | 1.0000 | 0.1500 | 0.4000 | 0 | pass |
| 0.65 | 0.20 | 0.4000 | 1.0000 | 0.2000 | 0.2000 | 0 | pass |
| 0.65 | 0.25 | 0.5000 | 1.0000 | 0.2387 | 0.0508 | 0 | pass |
| 0.65 | 0.33 | 0.5508 | 0.8571 | 0.2387 | 0.0000 | 0 | pass |
| 0.80 | 0.15 | 0.3000 | 1.0000 | 0.1500 | 0.4000 | 0 | pass |
| 0.80 | 0.20 | 0.4000 | 1.0000 | 0.2000 | 0.2000 | 0 | pass |
| 0.80 | 0.25 | 0.5000 | 1.0000 | 0.2358 | 0.0623 | 0 | pass |
| 0.80 | 0.33 | 0.5623 | 0.8804 | 0.2358 | 0.0000 | 0 | pass |
| 1.00 | 0.15 | 0.3000 | 1.0000 | 0.1500 | 0.4000 | \(1.14\times10^{-13}\) | pass |
| 1.00 | 0.20 | 0.4000 | 1.0000 | 0.2000 | 0.2091 | \(1.14\times10^{-13}\) | pass |
| 1.00 | 0.25 | 0.5000 | 1.0000 | 0.2317 | 0.0774 | \(1.14\times10^{-13}\) | pass |
| 1.00 | 0.33 | 0.5774 | 0.9114 | 0.2317 | 0.0000 | 0 | pass |

The residuals belong only to the exploratory `float64` score and scarcity
model; they are far below that model's tolerance. The separate branch-flow
kernel uses exact integers, conservative controller floors, and deterministic
largest remainder only inside fixed policy, depth, or same-controller amounts.
That does not turn the global reward model into consensus arithmetic or provide the second
independent implementation and cross-language golden-vector agreement required
for release.

### Interpretation

- The cluster-lifetime cap binds more strongly than \(\alpha\) in this small,
  concentrated fixture.
- A `0.15` cap sends 40% of the epoch to commons because the fixture's
  controller shares hit their cluster ceilings.
- Raising the cap reduces commons overflow but increases the amount any one
  controller can capture inside a cluster.
- The largest aggregate controller receives 30% to 57.74% across two clusters,
  demonstrating why a separate persistent program-wide exposure rule is a
  release blocker rather than an optional dashboard metric.
- Lower \(\alpha\) modestly favors smaller gross demands only after the reserve
  becomes scarce.
- Every grid cell gives the toy newcomer a positive path. This is a necessary
  sanity check, not evidence of real-world newcomer reachability.

No cell is a recommended consensus setting. Selecting one from this grid would
be false precision: the fixture does not yet model hidden controllers,
semantic-cluster errors, queueing, bribery, long-horizon resolution, or
behavioral adaptation.

## Release-gate result

The in-model gates pass:

- budget conservation;
- preclustered artifact-count invariance;
- controller alias collapse for both effective count and artifact score;
- temporal gross-accrual neutrality and no pacing advantage under isolated and
  competing scarce cohorts, plus no reaccrual after revocation and recovery;
- correlation discount;
- alias-neutral submitted/controller thresholds and effective-signal quorum;
- all twelve policy-owned weakest-surface power gates, with power excluded
  from the passing artifact score;
- safety as a hard, non-compensable gate;
- positive-affine strictly proper reviewer payment on the tested probability
  grid;
- behavior-complete in-memory snapshot/restore and serialized replay
  protection;
- reward/wealth non-conversion inside the proposed type boundary; and
- cluster-lifetime-cap/newcomer invariants throughout the grid.

The integration gates fail:

1. no canonical tree-v1 node/receipt, sponsor-escrow, or `E0`–`E6`
   compartment binding;
2. no tree-v1 organization, implementation, environment, assignment,
   authorization, or disclosure-lane enforcement;
3. no production semantic-equivalence adjudicator;
4. no per-node revocable epistemic frontier ledger;
5. no bounded replacement policy for honest successors after false maturation
   poisons an irreversible cluster cap;
6. no deterministic production scheduler that automatically carries eligible
   backlog;
7. no per-accrual expiry lots or irreversible extinguished-to-date state;
8. no appealable controller/family/correlation attestation system;
9. no authenticated twelve-surface snapshots or joint policy-to-payout path
   cut;
10. no persistent program-wide cross-cluster controller exposure rule;
11. no dedicated collateralized artifact-outcome envelope and
    resolve-or-expire ledger;
12. no atomic milestone/role/commons bank settlement;
13. no terminal expiry/invalidation disposition into named commons or refund
    accounts;
14. no durable consensus replay store or migration;
15. current custom governance still converts bonded fungible wealth into vote
    power;
16. no consensus-enforced pinned multi-kernel formal-proof path;
17. no reviewer assignment, exogenous-outcome, effort, and isolated-budget
    enforcement around the proper-score primitive;
18. no second independent exact branch-flow implementation and cross-language
    golden vectors; and
19. no independent external red-team reproduction.

The weakest illustrative power surface has effective controller count `1.515`
because infrastructure is 80% controlled by one operator. The dashboard uses
the minimum across surfaces; a healthy review panel cannot average away a
captured infrastructure or treasury path.

The report's “direct” and “commons” columns are cumulative controller-cap
calculator outputs. They are not claims that milestone states, role tranches,
or transfers have executed. The specification now gives every funded
milestone a terminal `direct` or `commons` disposition and requires one atomic
batch; the executable deliberately reports that implementation as absent.
Likewise, exact branch-flow results are allocation projections inside one
funded role envelope, not bank sends, vesting records, or settled receipts.

## Next research gates

Before a shadow devnet, the minimum next layer is:

1. a canonical tree-v1 typed-receipt and exact escrow-compartment fixture;
2. a second independent exact branch-flow implementation with cross-language
   golden vectors, plus exact consensus arithmetic for the wider score and
   scarcity model;
3. authenticated controller/equivalence fixtures containing both observed and
   hidden ground truth;
4. a per-node frontier/revocation ledger with bounded cap-poisoning replacement
   semantics;
5. eligibility-lot expiry and a deterministic backlog scheduler;
6. an atomic terminal milestone/role/commons settlement engine backed by a
   dedicated escrow;
7. a persistent cross-cluster controller exposure rule with merge and decay
   semantics;
8. late-merge and missed-equivalence accounting;
9. million-claim queue and reserve-exhaustion runs;
10. formal proof bundles replayed across at least two independent trust roots;
11. behavioral simulations for cartel entry exclusion, bribery, and parameter
   capture; and
12. external review that can reject the mechanism, not merely tune it.

Until those gates are met, the correct value-bearing reward is zero.
