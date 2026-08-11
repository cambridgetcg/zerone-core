# Branch-flow outcome-reward shadow

`branchflow` is a pure, deterministic calculator for one prospectively funded
outcome-reward envelope. It projects how a fixed integer uZRN amount could be
attributed to the funded cluster, load-bearing ancestors, independent
descendants, and terminal commons.

It is intentionally inert:

- `Allocate` has no store, bank, clock, network, signer, SDK, or governance
  access;
- every result says `assurance: SHADOW_ONLY`, `economic_effect: NONE`,
  `moves_funds: false`, and `integration_ready: false`;
- it accepts already-adjudicated graph, controller, role, dependency, impact,
  and replay-snapshot inputs but cannot establish that they are true; and
- the parent simulator exposes only a fixed reference-fixture CLI; no chain or
  module path calls the package.

This v0 package is only the outcome-reward shadow adapter. Its uZRN fields and
bounded outcome graph do not implement TC6 training-revenue settlement. TC6
requires its own manifest-cone adapter, depth rule, receipt ledger, funding
source, and reviewed specification.

## Locked arithmetic

All amounts and weights are non-negative integers. Floating point is absent.

The reference shadow policy is 600,000 PPM direct, 100,000 PPM upstream,
300,000 PPM downstream, and zero base commons. Both directions use 500,000 PPM
continuation through absolute depth five. Its controller cap is deliberately
non-binding at 1,000,000 PPM. `ReferencePolicyPreimage` exposes the exact bytes
committed by `ReferencePolicyDigest`; validation recomputes every supplied
policy digest and fails closed on drift.

`DefaultPolicy` is suitable only for an E5 or E6 shadow request because it has
a downstream compartment. E2, E3, and E4 requests must use a separately
digested policy with `DownstreamPPM == 0` and no descendant impacts. Every
request binds one `FundedMilestone`, and every descendant receipt must match
that milestone, so E5 and E6 evidence cannot mix in one envelope.

For each child edge with raw dependency weight `a`:

```text
denominator = max(1,000,000, sum of the child's raw edge weights)
share       = floor(1,000,000 * a / denominator)
```

Every subsequent graph multiplication also floors, so missing edge mass and
fixed-point precision loss never become claimant weight. Upstream residue is
terminal under its fixed PPM denominator. A downstream cohort starts from each
descendant's already materialized reliance, so pre-cohort graph-floor residue
is not reserved a second time: independent abundance may fill the fixed bucket.
Anything still unattributed after cohort settlement routes terminal. Even an
edge whose normalized share floors to zero remains visible in `NormalizedEdges`
and participates in raw DAG cycle validation.

Direction tranches use absolute geometric buckets. With continuation `q`,
scale `S = 1,000,000`, and maximum depth `D`, exact monetary apportionment uses
one common denominator `S^D`:

```text
depth d weight = (S-q) * q^(d-1) * S^(D-d)
tail weight    = q^D
```

Empty nearer depths never enrich farther recipients. A cluster reached along
paths of different lengths receives only its bounded flow in each exact depth.
`PASS_THROUGH` carries dependency flow but has no credits; `BLOCKED` neither
receives nor propagates it.

Prospectively fixed policy compartments and depth buckets use exact Hamilton
largest remainder. An exact tie prefers explicit commons, then the fixed leg
order direct, upstream, downstream, and finally canonical key order. Claimant
weights first aggregate by canonical controller; each controller receives only
the floor of its proportional amount, and every cross-controller residual unit
routes terminal. Hamilton then divides only that controller's already fixed
amount among its lines. Result allocation lines use explicit leg order rather
than lexical enum ordering. Splitting one controller across more roles or
artifacts therefore cannot capture extra rounding units, and invalidating one
controller cannot award a remainder unit to another. The constructor finally
proves:

```text
EnvelopeUzrn = ProjectedPaidUzrn + ProjectedCommonsUzrn
```

Any invariant or validation error returns no partial result.

## Descendants and exclusive receipts

Impacts are accepted only for E5/E6 and only after the descendant window is
closed. Multiple receipts for one semantic descendant cannot clone its
reliance mass. Their normalized shares are:

```text
floor(S * ImpactPPM / max(S, sum ImpactPPM for that descendant))
```

Every admitted impact explicitly declares `PAYABLE` or `TERMINAL` disposition.
Both remain in the closed cohort and its pre-role capacity; only `PAYABLE`
creates claimant lines. `TERMINAL` is a tombstone for an admitted slot that no
longer pays, so its capacity cannot be captured by a competitor. Both
dispositions consume the exclusive receipt on successful evaluation. The pure
kernel cannot prove that the caller supplied the complete authoritative cohort;
that closed membership/tombstone registry remains a release gate.

For reliance `R`, normalized impact `m`, and controller-role credit `p`, one
claimant term is `floor(R*m*p/S^2)`. V0 performs that as one exact integer
quotient; it does not introduce an extra intermediate floor between impact and
role credit.

At each depth, one semantic descendant's pre-role capacity is
`floor(R*min(S,sumRawImpact)/S)`, computed before receipt-share floors. The
terminal weight is the sum of those descendant capacities minus independently
eligible unsaturated claimant terms. Impact-normalization loss, missing role
mass, fixed-point loss, and visible funded-controller overlap therefore remain
terminal even when the surviving cohort is abundant. Monetary division uses
`max(S, saturatedEligibleWeight + terminalWeight)`; invalidating one line can
never enlarge a competitor's projection.

Canonical controller IDs shared with the funded cluster are visibly
non-independent. Those downstream credits are excluded and their mass remains
commons; independent credits on the same mixed-control descendant can still be
projected. The calculator cannot discover hidden, merged, or correlated
effective control. An authoritative controller registry and adjudication
process therefore remain release blockers.

Receipt use is economically exclusive. A current receipt key or economic slot
already present in `PriorReceiptUses` fails closed. Every accepted impact is
listed once in `NewReceiptUses` on evaluation even if visible control overlap,
controller caps, or minimum-payout dust leaves it with zero claimant value.
This consume-on-evaluation rule prevents a zero-valued attempt from being
replayed into a different independently funded envelope. Evidence systems may
reference the source event elsewhere, but that reference is evidence-only.

## Caps and dust

Controllers aggregate across all roles, clusters, and legs before caps. The
per-envelope allowance is `floor(EnvelopeUzrn * capPPM / S)`. If the optional
program-window cap is non-empty, prior projected exposure is subtracted before
the smaller allowance is applied. Cap overflow goes to commons and is not
redistributed to competitors.

The minimum payout is applied last to the controller's aggregate projected
amount. Sub-minimum controller lines all route to commons.

The pure package verifies only the caller-supplied snapshot. Production would
still require merge-safe global controller, receipt-use, program-exposure,
funding, liability, and settlement state.

## API and verification

The package's only evaluation entrypoint is:

```go
result, err := branchflow.Allocate(request)
```

There is no chain integration. The parent simulator exposes the fixed E5
reference fixture through:

```text
go run ./tools/constructive-rewards -mode branch-flow
go run ./tools/constructive-rewards -mode branch-flow -format json -branch-envelope-uzrn 100000000
```

It changes only the illustrative envelope amount; it does not accept an
arbitrary adjudication request or authorize settlement.
`testdata/reference_request.json` and `testdata/reference_result.json` form the
byte-exact cross-implementation golden vector.

Run the focused checks from the repository root:

```text
go test ./tools/constructive-rewards/branchflow
go test -race ./tools/constructive-rewards/branchflow
go test -run=^$ -fuzz=FuzzAllocateConservation -fuzztime=5s ./tools/constructive-rewards/branchflow
go test -run=^$ -fuzz=FuzzAllocatePermutation -fuzztime=5s ./tools/constructive-rewards/branchflow
go vet ./tools/constructive-rewards/branchflow
```

The suite covers reference-policy commitment, exact conservation, absolute and
convergent depths, fixed-precision graph traces, zero-share cycle rejection,
pass-through behavior, visible independence, admitted terminal tombstones,
per-descendant impact normalization, receipt replay, milestone binding, caps,
dust, maximum integer amounts, permutation invariance, concurrent replay, a
4,097-line bounded cohort, byte-identical golden replay, and fuzzed
conservation/permutation.
