# Capture challenge: accounting and expiry contract

`x/capture_challenge` holds challenge stakes and domain bounty funding. These
are separate liabilities even though the same SDK module account holds the coins.
`FundBountyPool` is the explicit funded-pool entry point; challenge escrow is not
bounty funding. The documented per-fact contribution hook is not wired here.

## Selected accounting semantics

The sole selector is `Keeper.ChallengeAccountingEnabled`, backed by the shared
`types.ChallengeAccountingEnabledKey`. Absence selects historical behavior;
exactly `{1}` selects the repaired behavior below. An unreadable or malformed
marker fails closed before phase changes, resolution or automatic submission.
The reader uses its own infinite gas meter, preserving pre-activation charged gas.

Native `DefaultGenesis` selects the repair. Older genesis JSON without
`challenge_accounting_enabled` imports as off, and export/import retains the
selection. Pre-carrier fixtures built from `DefaultGenesis` must explicitly set
that field false. There is no transaction or parameter toggle.

With the marker off, replay preserves the old REJECTED pool credit, unchecked
expiry refund/write behavior, absence of a resolution deadline guard, and
UNDER_REVIEW automatic-challenge suppression, including their errors and events.
The repair does not silently alter history before its activation boundary.

## Stake and pool backing

With the marker on, a rejected challenger's stake goes only to
`development_fund`. It does not also increase `DomainBountyPool.Balance`. This
preserves the routing specified in
[R16-3](../../prompts/R16/R16-3-module-revenue-splits.md#8-xcapture_challenge--challenge-resolution),
including PARTIAL's forfeited half. UPHELD and PARTIAL rewards still trust and
consume the stored domain pool; enabling the marker does not validate its backing.

At a coupled auth/bank/module import or activation boundary, require:

    bank(capture_challenge, uzrn)
        >= sum(domain pool balances) + sum(refundable open/pending stakes)

Unsolicited transfers can create valid surplus. Comparing a pool alone to the
entire account ignores other pools and refundable escrows. Validate bounded,
nonnegative amounts, identities and uniqueness as well. `GenesisState.Validate`
currently checks only Params; `InitGenesis` does not enforce this backing
invariant. A successful import is not evidence of solvency.

## Refund-pending is non-terminal, not a live adjudication

With the marker on, existing fields carry the obligation without a new enum:

- `UNDER_REVIEW` with `height < review_deadline`: adjudication remains open.
- `UNDER_REVIEW` with `height >= review_deadline`: expiry/refund is pending.
  The challenger and stake remain visible in `Challenge`, `ActiveChallenges`
  and genesis export. gRPC, REST, CLI and explorer consumers must compare query
  height with `review_deadline`; the query does not distinguish them otherwise.
- New `EXPIRED`: refund and terminal write succeeded together, or the stake was
  zero. Historical EXPIRED records do not gain that payment assurance.

Each due refund uses its own SDK cache. Bank effects, the checked EXPIRED write
and bank events (`coin_spent`, `coin_received`, `transfer`, `message`) commit to
the parent context together. Returned bank or write errors discard those effects
and leave the obligation pending. Other challenges can still advance. Zero-stake
automatic challenges require no bank send. A cache is not crash-durable commit
and does not roll back consumed gas.

Every pending record is retried each BeginBlock, with no backoff or terminal
failure state. The Error log `expiry refund pending` is emitted at the deadline
and every 100 blocks after it, not every retry. There is currently no separate
pending-refund event. The existing all-challenge scan and store-key ordering
remain; this is not a bounded retry queue. Malformed stakes/addresses or a
permanently blocked recipient cannot be cleared by normal resolution: they need
a separately reviewed upgrade handler. No automatic redistribution or clawback
is performed.

`ResolveChallenge` refuses every outcome at/after the review deadline. An overdue
financial obligation no longer suppresses a new automatic adjudication for the
domain. Because capture defense runs first in BeginBlock, a persistently flagged
domain can be re-challenged at the deadline, even when expiry succeeds, rather
than at the next analysis interval. Live cases still deduplicate.

## Retained historical state

The policy is prospective only: census before activation, fail closed on an
unsafe census, hold and surface obligations, and no clawback or automatic
redistribution. Enabling the marker writes only the marker, not old challenges,
pools or bank balances. Open records need no schema conversion. Historical
EXPIRED records are never replayed; their payment success cannot be inferred
from their terminal status.

Before first activation, the exact cumulative phantom credit per domain is the
sum of `Stake` over its `RESOLVED` records with `Resolution.Outcome == REJECTED`:
both the old burn and
development-fund eras credited exactly the stake. This amount is computable;
how to cover any deficit already paid out as rewards is a separate policy
question. Rejections after activation create no phantom credit and must not be
included in a later historical census. An affected pool still threatens other
escrows after the marker is on.
Do not activate an unreconciled chain merely because future rejected credits
have been stopped. The carrier's census must inspect every challenge, every
pool and actual module custody, and hold activation for reviewed reconciliation
if any rejected resolution or backing shortfall exists. Any use of
`development_fund` to cover a deficit requires explicit authorization, not an
inference of available funds. An export for relaunch requires the same audit.

## Verification

From the repository root, with an explicitly selected scratch TMPDIR:

    GOFLAGS=-mod=readonly GOPROXY=off go test -p 4 ./x/capture_challenge/... -count=1
    GOFLAGS=-mod=readonly GOPROXY=off go vet -p 4 ./x/capture_challenge/...

`keeper/accounting_test.go` uses real SDK auth/bank stores with synthetic funding.
It covers backed lifecycles, returned bank/store errors, retries, successful bank
event propagation, pending-query visibility and coupled genesis round trips.
`accounting_legacy_test.go` compares state/event digests, errors, gas and send
counts against an executed pre-repair source oracle. `accounting_gate_test.go`
checks marker errors, deadline timing, log throttling and the deliberately
unsafe legacy-pool case. Injected store-write errors test a dependency contract,
not production disk loss. These are local tests, not a live census, independent
acceptance, historical repayment or an upgrade rehearsal.
