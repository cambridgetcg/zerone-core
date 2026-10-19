# Capture-challenge accounting repair (draft)

This is a **consensus-changing, prospective source repair**, not an activated
upgrade or a repair of historical financial state. It must not be deployed as an
unscheduled binary replacement. No protobuf fields, store keys, module version,
upgrade handler or migration are changed here.

## Rejected stakes have one destination

A rejected challenger's stake goes only to `development_fund`. It does not also
increase `DomainBountyPool.Balance`. The retained routing requirement explicitly
says “Failed challenge stake: route to development fund instead of burning” in
[`R16-3`](../../prompts/R16/R16-3-module-revenue-splits.md#8-xcapture_challenge--challenge-resolution).
That agrees with the existing REJECTED transfer/comment and PARTIAL's forfeited
half. The older [R6 brief](../../prompts/R6/R6-3-capture.md#implementation) requires
forfeiture but does not select a destination. Removing the second, unbacked credit
preserves the explicit routing decision rather than silently redirecting money.

UPHELD and PARTIAL rewards still consume their domain's funded pool. The module
account also holds refundable challenge stakes: those escrows are not bounty
funding. `FundBountyPool` remains the explicit funded-pool entry point. This repair
does not activate the documented but unwired per-fact contribution hook.

## Expiry is payment followed by a terminal record, atomically

The existing fields encode refund-pending without a new enum or key:

- `UNDER_REVIEW` with `height < review_deadline`: adjudication is still open.
- `UNDER_REVIEW` with `height >= review_deadline`: expiry/refund is due; the
  retained challenger and stake are the pending financial obligation. The record
  remains visible in `Challenge`, `ActiveChallenges` and genesis export.
- `EXPIRED`: this binary completed the refund, or a zero-stake challenge expired.
  Historical EXPIRED records **do not** acquire that payment assurance retroactively.

Each due refund uses its own SDK cache. The bank transfer, checked challenge
write and their events commit together. A returned bank or write error leaves
all those effects unchanged, logs the pending challenge ID, and retries on later
BeginBlocks. It neither returns a BeginBlock error nor prevents other challenges
from advancing. Zero-stake automatic challenges need no bank send. Malformed
stake/address records remain pending rather than being silently declared paid.
A cache is not crash-durable commit, and it does not roll back consumed gas.

`ResolveChallenge` refuses all outcomes at or after the review deadline: a
failed expiry refund cannot subsequently be forfeited or rewarded by governance.
An overdue financial obligation no longer suppresses a new automatic challenge
for the same domain; the new, still-live adjudication retains the existing
deduplication. Since capture defense runs before capture challenge, this also
permits automatic replacement at the deadline rather than waiting another block.
The financial-pending query status is not an invitation to adjudicate it.

The existing all-challenge scan remains; each due record is attempted once per
call in store-key order. This is not a new bounded retry queue or a throughput
claim. Other setters and resolution's existing caller-owned transaction contract
are unchanged: the checked-write guarantee introduced here is expiry-only.

## Existing state and activation gate

No mechanical schema migration is needed for still-open records: their existing
status, deadline, stake and challenger already preserve the retry obligation.
Export/import with the corresponding SDK auth/bank state retains that obligation.
This does **not** mean an affected existing chain needs no reconciliation:

1. Old REJECTED resolutions may already have inflated pool balances; deleting
   the future credit does not reduce those records or restore coins spent on
   rewards. Rewards still trust the stored pool, so stale overstatements can
   continue consuming other escrows. **Activation is blocked until an authenticated
   complete census and an explicitly approved reconciliation establish backing.**
2. Old EXPIRED records do not distinguish successful from failed refunds. This
   draft never retries them automatically; doing so could pay an already-refunded
   challenger twice. Historical payment evidence and approved, exact unpaid
   obligations are required before a separate repair can requeue or compensate.
3. Do not infer entitlement from a pooled balance, blindly subtract every old
   rejected stake, zero pools, mint a deficit, or refund every EXPIRED record.
   Already-consumed rewards and explicit funding must be reconciled separately.

The existing [consolidation sequence](../../docs/MODULE-CONSOLIDATION-SEQUENCE.md#sequence-and-completion-evidence)
requires the financial census before custody retirement. This draft performs no
live census and includes no migration: those operations are outside this lane.
Yu must approve the historical-state policy and upgrade carrier before adoption.
A possible carrier is `knowledge-claim-integrity-v1` (knowledge 11→12), **only
if that release remains unfrozen and its reviewed scope is explicitly expanded**;
its [current specification](../../docs/specs/knowledge-claim-integrity-v1.md#13-migration-and-named-upgrade)
is not authorization for this repair. Otherwise use a separately reviewed
`capture-challenge-accounting-v1` successor rung (a proposed name, not an existing
handler). Never graft it into frozen H1/H2/H3 artifacts or treat the existing
`accounting-authority-v1` custom staking/governance repair as a capture-challenge
migration. No carrier is registered or scheduled by this patch.

## Genesis backing recommendation

**Yes: imported bounty pools should be checked against actual module custody,
with outstanding challenge stakes included.** At the coupled auth/bank/module
import boundary, after bank state is available, require:

    bank(capture_challenge, uzrn)
        >= sum(domain pool balances) + sum(refundable open/pending stakes)

Equality holds in a closed fixture with no surplus; unsolicited bank transfers
can create a legitimate surplus. Checking each pool alone against the whole
account, or comparing pools to the account without reserving escrows, is not
sufficient. Validate nonnegative bounded integer amounts, record identities and
uniqueness as well. Historical ambiguous EXPIRED claims require the preceding
census; they cannot be safely reconstructed from this inequality.

Currently `types.GenesisState.Validate` checks only Params, and `InitGenesis`
stores challenge/pool records without a bank-backing check. This draft does not
change that import contract. A separately reviewed coupled import/upgrade audit
is a prerequisite for claiming existing-state solvency, not a consequence of the
passing lifecycle tests below.

## Local evidence and limits

Run from the repository root with a writable, explicitly selected scratch TMPDIR:

    GOFLAGS=-mod=readonly GOPROXY=off go test ./x/capture_challenge/... -count=1
    GOFLAGS=-mod=readonly GOPROXY=off go vet ./x/capture_challenge/...

`keeper/accounting_test.go` uses the real SDK auth/bank stores and module
BeginBlock wrapper with synthetic funding. It covers unfunded/funded
REJECTED→UPHELD and REJECTED→PARTIAL sequences with another challenger's escrow,
actual insufficient funds, before/after-transfer and before/after-record faults,
repeated failures, later single payment, deadline refusal, unaffected peers,
automatic admission, malformed records, zero stakes, terminal legacy records,
and a coupled logical genesis round-trip. Every closed lifecycle reconciles bank
coins with pools plus refundable stakes and checks total bank supply. The SDK
store adapter normally returns nil after its underlying void Set; injected
returned write errors are dependency-contract tests, not a production disk-loss
reproduction. No node, signed governance transport, live account, scientific
verdict or historical repayment is established by these tests.
