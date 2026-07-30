# Constructive-Intelligence Reward Simulator

This is a deterministic, standard-library-only adversarial simulator for the
pre-consensus design in
[CONSTRUCTIVE-INTELLIGENCE-REWARDS.md](../../docs/tokenomics/CONSTRUCTIVE-INTELLIGENCE-REWARDS.md).
It neither reads nor changes chain state. It is not an implementation of a
Zerone reward path or an adapter for the canonical
[constructive-intelligence tree v1](../../docs/specs/constructive-intelligence-tree-v1.md).

The same command also contains a deliberately separate exact-integer
[shadow capacity ledger](../../docs/tokenomics/CONSTRUCTIVE-INTELLIGENCE-SHADOW-LEDGER.md).
That fixture tests one-shot reattribution of quarantined, still-unpaid
capacity. It uses unnamed model units, authenticates none of its inputs,
settles exactly `0 ZRN`, and closes no production integration gate.

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
would need specified fixed-point ranges, exact largest-remainder allocation,
golden test vectors, and cross-implementation conformance before activation.

## Run

From the repository root:

```sh
go test ./tools/constructive-rewards
go run ./tools/constructive-rewards -mode report
go run ./tools/constructive-rewards -mode sweep
go run ./tools/constructive-rewards -mode model
go run ./tools/constructive-rewards -mode shadow
go run ./tools/constructive-rewards -format json
```

`-mode shadow` prints the exact fixed vector:

```text
event                 A    Z    L    Q    X    R
accrue                100    0  100    0    0    0
fund                  100   30   70    0    0    0
final-invalidation    100   30    0   60   10    0
reattribute           100   30   50   10   10   50
```

The CLI uses `Q` as the compact display label for quarantined capacity; the
normative design uses \(Y\) to avoid collision with \(Q_{C,e}\), which already
means new gross accrual. Use `-mode shadow -format json` for named fields.
Behavior-complete restore requires both the snapshot and its expected
SHA-256 state commitment from a separately trusted location. Keeping both
under one attacker provides no rollback or authenticity guarantee. Restore
rejects noncanonical snapshot encodings and unreachable cross-lot successor
states, chronological live-claim inversions, unjustified quarantine, and
controller graphs not emitted by the monotone linker; a successful restore
snapshots back to the same commitment. The caller-provided epoch remains a
trust boundary and must come from an authenticated authoritative source before
any integration.

The release check is deliberately fail-closed:

```sh
go run ./tools/constructive-rewards -mode release
```

It exits `1` while production prerequisites remain absent, including
controller attestation, semantic-equivalence adjudication, a pre-funded
breakthrough reserve, canonical tree/typed-receipt and `E0`–`E6` compartment
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
there is no production scheduler. The separate shadow ledger has exact
eligibility-lot expiry and extinguishment, but it is a generic counterfactual
machine with caller-declared roots, receipts, support, controllers, and final
decisions. Neither mode consumes authenticated tree-v1 node digests, sponsor
escrow compartments, or exact milestone transitions.
Controller allocations are calculator totals, not role/tranche bank
transfers, and the cap is cluster-lifetime rather than a persistent
program-wide limit. These omissions remain explicit integration failures, so
the production release gate stays closed even when every in-model invariant
passes.
