# Constructive-Intelligence Reward Simulator

This directory contains two deterministic, standard-library-only shadow tools
for the pre-consensus design in
[CONSTRUCTIVE-INTELLIGENCE-REWARDS.md](../../docs/tokenomics/CONSTRUCTIVE-INTELLIGENCE-REWARDS.md).
The original exploratory simulator exercises scoring, scarcity, and power
attacks with bounded `float64` arithmetic. The separate exact-integer
[`branchflow`](branchflow/) package projects one conserved, already funded
milestone-role envelope across direct, upstream, downstream, and terminal
lines; its included reference fixture is E5. Neither tool reads or changes
chain state, implements a Zerone reward path, or adapts
the canonical
[constructive-intelligence tree v1](../../docs/specs/constructive-intelligence-tree-v1.md).

The simulator makes nine mechanism boundaries executable:

1. evidence and reward attach to a semantic-equivalence cluster, not a person,
   address, or artifact count;
2. proposer-supplied aliases collapse to one unit-weight controller before
   quorum and controller-indexed correlation arithmetic;
3. exactly twelve policy-owned power surfaces are required as hard gates, but
   power metrics do not scale a passing artifact's score;
4. safety is a hard gate and cannot become a compensating score dimension;
5. only an increase above the irreversible economic high-water target creates
   new accrual, so revocation and recovery cannot accrue the same target twice;
6. eligible demand is cumulative accrual minus funded-to-date, so scarcity
   backlog persists and paced disclosure cannot manufacture extra eligibility;
7. a fixed epoch budget and cumulative controller cap within each cluster
   bound controller totals across epoch boundaries, with cap overflow assigned
   to commons;
8. epistemic and civic voice ignore stake, liquid wealth, and reward balance;
   and
9. complete in-memory snapshots preserve policy, registered clusters,
   accounting state, epoch IDs, and event replay IDs.

The engine, rather than its caller, owns immutable cluster caps and credit
partitions, high-water marks, funded-to-date counters, controller
paid-to-date counters, policy power maps, epoch IDs, and event replay IDs. It
gates reward on alias-neutral submitted/controller thresholds, effective
replication quorum, and the weakest of the twelve required controller-level
power surfaces. A correlation matrix, when supplied, is indexed by the
lexicographically sorted collapsed controller set—not the raw signal list.

The scoring and scarcity sweep use `float64` because this is an exploratory
model involving logarithms and fractional powers. Monetary inputs are bounded
to `[1e-6, 1e9]`, aggregate work is bounded, and conservation uses
ULP-aware directional checks. Those controls make the experiment testable;
they do not make floating-point arithmetic consensus-safe. Consensus code
would need specified fixed-point ranges, a complete exact-integer apportionment
rule (largest remainder only at its declared boundaries), golden test vectors,
and cross-implementation conformance before activation.

## Run

From the repository root:

```sh
go test ./tools/constructive-rewards
go run ./tools/constructive-rewards -mode report
go run ./tools/constructive-rewards -mode sweep
go run ./tools/constructive-rewards -mode model
go run ./tools/constructive-rewards -mode branch-flow
go run ./tools/constructive-rewards -format json
```

`-mode branch-flow` runs the reviewed E5 reference fixture with exact decimal
amounts. At the default envelope, its compact JSON result is byte-compared with
the checked-in golden result. Its text and JSON outputs always expose
`assurance=SHADOW_ONLY`,
`economic_effect=NONE`, `moves_funds=false`, and `integration_ready=false`.
Change only the illustrative envelope with
`-branch-envelope-uzrn <canonical-decimal-integer>`; this intentionally leaves
the default result's golden vector while retaining the fixed milestone, graph,
controller credits, and receipt. The 60/10/30/0 policy and half-per-hop
depth-five gradient remain independently digest-pinned. This is a projection,
not a quote, entitlement, bank message, or release gate.
Exploratory `-budget`, `-alpha`, and `-controller-cap` flags are rejected in
branch-flow mode, and `-branch-envelope-uzrn` is rejected in every other mode,
so no accepted option is silently ignored.

The release check is deliberately fail-closed:

```sh
go run ./tools/constructive-rewards -mode release
```

It exits `1` while production prerequisites remain absent, including
controller attestation, semantic-equivalence adjudication, a pre-funded
artifact-outcome envelope, canonical tree/typed-receipt and `E0`–`E6` compartment
binding, automatic backlog scheduling and expiry, per-node revocation state,
a bounded cap-poisoning replacement policy, atomic milestone/role/commons
settlement, a persistent program-wide controller ceiling, formal proof-kernel
enforcement, and governance power that is independent of bonded fungible
wealth. `-mode model` exits `0` only when the mathematical model invariants
pass. Invalid input exits `2`.

All defaults are illustrative. Explore the scarcity surface with:

```sh
go run ./tools/constructive-rewards \
  -mode report \
  -budget 1000 \
  -alpha 0.65 \
  -controller-cap 0.20
```

## Adversarial comparisons

The report includes:

- a 40% whale splitting into 100 addresses under square-root stake weighting;
- one already-clustered result represented as 1 versus 100 artifacts;
- 100 reviewers with pairwise error correlation `rho = 0.2`;
- 100 addresses linked to one controller;
- address-level versus controller-level HHI/effective power;
- a grid over `alpha` and the cluster-lifetime controller cap;
- direct payout and commons overflow for one frontier jump versus three
  intermediate epochs under both abundant and scarce budgets;
- a competing-cohort temporal fixture that permits backlog differences but
  rejects any pacing advantage;
- truthful expected payment under a positive affine binary Brier score; and
- an explicit model/integration release-gate matrix.

Guarantees are conditional on conservative controller and semantic-equivalence
links being detected. Hidden common control or missed equivalence is an
assumption failure, not evidence of decentralization. The executable verifies
artifact-count invariance only after a semantic cluster is supplied; it does
not discover equivalence. Its frontier is one economic scalar rather than a
per-node revocable 技能樹. Backlog persists only when a cluster is evaluated;
there is no production scheduler, eligibility-lot expiry, or
extinguished-to-date counter. It also does not consume tree-v1 node digests,
typed receipts, sponsor escrow compartments, or exact milestone transitions.
Controller allocations are calculator totals, not role/tranche bank
transfers, and the cap is cluster-lifetime rather than a persistent
program-wide limit. These omissions remain explicit integration failures, so
the production release gate stays closed even when every in-model invariant
passes.
