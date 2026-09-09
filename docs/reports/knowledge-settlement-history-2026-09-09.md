# Knowledge settlement history — 2026-09-09

Status: **historical reconciliation against an authenticated, operator-observed
checkpoint; individual historical payments and complete ancestry are not independently
established. No ledger mutation or production activation.**

This follows the [existing-ledger census](authenticated-ledger-census-2026-09-09.md)
at application height **1,261,575**, post-commit AppHash
`a381775589d84a74bbf365b9b2add26201465e20980cf50a53544f7a81256106`.
It explains the knowledge account's **7,313,952 uzrn** and the vesting account's
**16,553,211 uzrn** using committed surviving records and separately identified
archive observations. All amounts below are integer **uzrn**.

The retained `Claim.Stake` amounts are not automatically outstanding refunds.
For the 27 ordinary submissions examined here, the inspected source model and
selected archive events identify them as non-refundable review fees. The later
survival rewards are separate nominal vesting claims,
even though each has the same 200,000 amount as the original fee.

## Evidence and what it establishes

The prior census supplies complete, root-bound raw store bytes, native balances,
current claims, verification rounds, facts, vesting schedules and their indexes.
This follow-up reads those immutable exports; it does not modify or restart the
copied database. Its source baseline is
`71cb372b3cde391c2f1661618e175d2436a685d5`.

The archive acquisition targets all 27 surviving claim submissions, all 27
verdict heights and all 25 vesting-creation heights: **79 distinct result
heights**. A second target set comes directly from the **343 surviving committed
block-reward records**. Searches discover candidates; they do not prove the
absence of other or deleted history.

All reported index pages were captured within the bounded queries. Incoming and
outgoing transaction searches each return exactly the same 27 submission
transactions. Block-event searches return exactly the 343 reward-inflow heights
and the 25 accepted-round outflow heights. These sets agree with the surviving
records; index coverage is not a consensus-authenticated completeness proof.

Historical transaction authentication and event interpretation have different
boundaries. In the deployed CometBFT **0.38.20** dependency:

- Header H's `DataHash` commits to the ordered transaction bytes in block H.
- Header H+1's `LastResultsHash` commits to the ordered H results' `Code`, `Data`,
  `GasWanted` and `GasUsed`. Typed message-response bytes inside `Data` can bind
  a returned claim ID to a successful submitted transaction.
- `Events`, `Log`, `Info` and `Codespace` are excluded from that result hash.
  Transfer events and transaction-search event matches are **archive
  observations**, even beside a transaction with a verified success result.
- Begin/End/FinalizeBlock events have no separate transaction-result commitment.
  An application root commits to resulting state, not to those event strings.

The exact source boundary is
[CometBFT results.go](https://github.com/cometbft/cometbft/blob/v0.38.20/types/results.go),
with transaction commitments in
[tx.go](https://github.com/cometbft/cometbft/blob/v0.38.20/types/tx.go).
A successful transaction also does not prove that a handler checked every bank
error. Cosmos SDK ante-handler effects can persist when later message execution
fails; a failed message is not evidence of zero transaction-fee expenditure.

Historical H/H+1 pairs are checked against the observed sole-validator identity,
including complete validator sets, signatures, block bodies, part sets, full
previous-block links, validator continuity, transaction positions and result
counts. The exact CometBFT 0.38.20 verifier passed all **27 pairs**, rechecked the
old checkpoint pair, and verified **56 canonical commits** from **227 hashed
inputs**. All used inputs remained byte-identical after verification.
This is **not** a complete historical header path to the later census
checkpoint. Shared RPC upstreams are not independent witnesses, and the prior
signer-custody failure remains unresolved. No independent trust anchor is added.

Historical bank-proof requests at heights **76,543** and **76,544** returned
ABCI code 7 (`store`), no value or proof, and a message that the proof was empty
and the height might have been pruned. Authenticated before/after bank values
were unavailable through those requests. This is a proof-availability result,
not evidence that the historical account or transfers were absent.

## Review fees and verifier settlement

The 27 surviving claims each record 200,000: **5,400,000** in total. Of their
completed rounds, **25 accepted** and **2 were inconclusive**. Every verdict
height is submission height +400. This is a statement about those surviving
records and their linked transactions, not an assertion that every historical
knowledge activity survives in this namespace.

Every selected transaction contains one `MsgSubmitClaim` with a successful,
committed response identifying the corresponding current claim. Its included
request fields match the current record's submission height, submitter, domain,
category, exact content bytes, amount and claim type. All 27 are ordinary
assertions with `stake=200000`, `sponsored=false` and no partnership or
provisional-challenge target. The transaction-envelope decoder is a bounded
message projection, not a complete canonical SDK transaction validator or
proof of signer ownership.

The [ordinary submission handler](../../x/knowledge/keeper/msg_server.go) collects
a non-refundable review fee. Its [fee splitter](../../x/knowledge/keeper/fees.go)
leaves 55% in knowledge for verifiers and routes the remaining 45% immediately.
The selected archive events report the following allocation on all 27 fees:

| Review-fee allocation | Per submission | All 27 |
| --- | ---: | ---: |
| Verifier pool retained in knowledge | 110,000 | 2,970,000 |
| Protocol treasury | 44,000 | 1,188,000 |
| Development fund | 39,340 | 1,062,180 |
| Research routed through vesting | 6,660 | 179,820 |
| Total | 200,000 | 5,400,000 |

Network transaction fees are separate and excluded from this table.

The corresponding observed bank-transfer rows agree with the fee splits. The
research leg passes through vesting to the research fund; the selected rows
report the entire **179,820** reaching that fund and no founder deduction.
The protocol treasury's committed balance is also **1,188,000**. Matching that
balance corroborates the model; it does not independently authenticate every
historical transfer or exclude offsetting activity.

At the 25 acceptance heights, observed transfers account for their full
**2,750,000** verifier pools:

| Observed use of accepted-round pools | Amount |
| --- | ---: |
| Verifier payments, across six recipient addresses | 1,974,500 |
| Independence discounts forwarded to development | 775,500 |
| Total | 2,750,000 |

The two inconclusive rounds have no corresponding observed pool payout, leaving
**220,000** under this model. They do not create a submitter refund merely
because their original `Stake` fields remain. Recipient addresses identify
ledger destinations, not beneficial owners or independent people.

## The knowledge account's balance

The current vesting store retains 343 block-reward distribution records,
covering recorded heights 54 through 863,021. Each record's producer, research,
founder, development and protocol allocations exactly sum to its recorded
mint amount. Their total block rewards are **107,488,296**.

The committed protocol allocations total **23,647,163**. Applying the recorded
current protocol split separately to each record, with integer rounding at each
height, gives:

| Historical protocol allocation reconstructed from records | Amount |
| --- | ---: |
| Verification pool: floor(30% of each protocol allocation) | 7,093,952 |
| Citation pool: floor(50% of each protocol allocation) | 11,823,566 |
| Treasury: each record's remaining 20%, including rounding | 4,729,645 |
| Total | 23,647,163 |

Current parameters alone do not prove historical parameters. The archived
block-reward transfers provide the separate historical cross-check for this
reconstruction: all **343** observed inflows match the per-height verification
allocation exactly. Each selected mint's `coin_received`/`coinbase` pair and
each selected transfer's `coin_spent`/`coin_received`/`transfer` triplet agree.
Those event checks test consistency, not cryptographic payment authenticity.
The inspected legacy reward source sends verification shares
from vesting to knowledge; it retains citation and treasury shares in vesting.
Current source has retired this automatic minting path, so current behavior must
not be projected backward onto the observed legacy runtime.

The resulting knowledge-account equation is:

```text
  7,093,952  block-reward verification allocations
+ 5,400,000  ordinary review fees
- 2,430,000  immediate fee routing
- 1,974,500  observed verifier payments
-   775,500  observed independence discounts to development
= 7,313,952  committed knowledge account balance

Equivalent: 7,093,952 + 220,000 = 7,313,952.
```

This resolves the unexplained balance arithmetically under the observed history.
It does not assign the retained block-reward pool to particular claimants or
make either component discretionary funds. Neither a refund nor an additional
mint is justified by this reconciliation.

## Vesting creation, backing and retained allocations

All 25 accepted claims map to 25 committed vesting schedules, with the same
claim IDs and recorded submitter/recipient relationship. For every schedule,
creation and acceptance heights equal the fact's challenge-window end:
**verdict height +34,272**. The three timestamps describe distinct lifecycle
stages; their separation is consistent with the survival gate.

Each schedule records total 200,000, released 0, currently claimable 170,000 and
reserve 30,000. The complete present schedule and index reconciliation is:

| Current vesting quantity | Amount |
| --- | ---: |
| Nominal schedule total | 5,000,000 |
| Stored amount available for recipient release | 4,250,000 |
| Reserve already included in the nominal total | 750,000 |
| Bank custody | 16,553,211 |
| Bank minus nominal schedules | 11,553,211 |

The 25 creation-height archive observations contain no nonzero bank transfer
funding the schedules. The inspected constructor creates a record; it does not
mint or transfer the nominal reward. Thus schedule creation is neither a fee
refund nor a completed recipient payout.

The retained citation and treasury reconstruction gives
**11,823,566 + 4,729,645 = 16,553,211**, exactly the vesting account's custody.
This is consistent with the reconstructed funding origin. It does not create
additional claims of that amount alongside the 5,000,000 in schedules. Historical
allocation labels and actual claimant records must be resolved together before deciding which
funds can be reassigned. The **11,553,211** arithmetic remainder is not declared
free or ownerless.

## The shared mint counter is broader than block rewards

The committed shared mint counter is **112,372,296**, exactly the native supply
increase over the public genesis's **13,555,000,000**:

```text
13,667,372,296 - 13,555,000,000 = 112,372,296
112,372,296 - 107,488,296 block rewards = 4,884,000
```

The block-reward records' cumulative `fund_balance_after` fields therefore are
not just a running sum of block rewards. The surviving sequence has 12 intervals
with additional counter increments and a further increment after its last
record, totaling **4,884,000**. `fund_balance_after` is shared mint accounting,
not a spendable bank balance, knowledge-only issuance, or evidence that all
minted amounts were paid to their intended recipients. Matching the supply
delta does not exclude unrecorded offsetting mint/burn activity.

That difference has a separate record and archive reconciliation:

| Other shared-counter issuance observed | Count | Amount |
| --- | ---: | ---: |
| Bootstrap claiming-pot claims | 2 × 222,000 | 444,000 |
| Substrate witness rewards | 20 × 222,000 | 4,440,000 |
| Total | 22 | 4,884,000 |

The two committed claiming-pot records and twenty settled substrate records
match those amounts and identities. Witness settlement height is not the mint
height: all twenty observed releases occur **200 blocks later**. Their release
heights and the two bootstrap claim heights explain every additional cumulative
counter interval and the tail exactly. At all 22 selected mint heights, the
archive reports matching mint and recipient-transfer event triplets. These are
observations with the same event-proof limitation above; they do not turn
`SETTLED` alone into a payment receipt or establish identity-based auto-grants.

## Remaining settlement work

The concrete source risks are error handling and claim ownership, rather than
an arithmetic deficit in these two module accounts:

1. **Make future settlement atomic and retryable.** Ordinary fee splitting can
   transfer some legs before an error; its caller logs the error and still emits
   the intended distribution event. Verifier payout errors and withheld-fund
   routing errors are ignored. Survival routing ignores schedule-creation
   failure, then removes the pending record and emits release. A prospective
   repair needs committed success receipts and preservation of unpaid work on
   failure, with fault-injection tests and an explicitly owned upgrade boundary.
2. **Preserve actual claims while classifying pooled custody.** Keep the 25
   schedules and two inconclusive-review residuals intact. Define how the
   retained verification, citation and treasury allocations relate to present
   claimant records; do not double-count the same custody or infer recipients
   from aggregate labels.
3. **Close the historical proof gap before adjudicating disputed payments.**
   Event indexes and surviving records give a bounded, internally consistent
   reconstruction. A disputed transfer needs authenticated before/after state
   and transaction ordering, or a reproducible replay with the correct historical
   application versions and an established header ancestry. The active legacy
   executable still has no recovered VCS identity.

These findings do not activate the prospective
[accounting-authority repair](../specs/accounting-authority-v1.md), establish its
required predecessor lineage, or reopen signer custody. No old claimant amount,
production binary, signing configuration or live chain state is changed.

A final public observation at height **1,261,723** reported `catching_up=false`
and the same sole validator identity with power 11,111. The analysis workstation
was separately confirmed stopped; it was not started for this historical pass.

## Reproduction and review

Private evidence is retained under `knowledge-settlement-20260909`, separately
from the immutable `census-20260909` evidence. It includes exact input digests,
GET URLs and observation times, bounded acquisition plans, all raw responses,
per-claim links, strict offline decoders, signature/result verification, source
pins and review receipts. Private claim content and individual wallet records
are excluded from this public report.

| Private evidence artifact | SHA-256 |
| --- | --- |
| Historical signed submission pairs | `b724a9ded3541be54b644267cf4b511129d9b5c9913edf8396e26a3b3591143e` |
| Signed submissions joined to current claims | `d776a772a1c9af10f6ab1a94464817d6b3271e1e58178b743039c0e6057edc9c` |
| Committed reward-record interpretation | `75be2ab2ff05a87f417c98332cefbdcc6cdafc8bf6271eb7115aededeb7d146b` |
| Observed funding reconciliation | `4f0d35d3b851c954be5f370119013ec164028a5586347b1b78a18640020481a7` |
| Other mint observations | `3ed659031155e2cbab12ad94f34985638dfa0e5003437d0cb17264e718df0e55` |
| Independent generated-protobuf reward review | `aeb4999db910dc26ef2557b28b3b40938e4660231e791e347d983c550e4d7cca` |

Independent review used generated Go protobuf types rather than the primary
Python wire decoder. Record fixtures reject malformed amounts, wrong keys,
unknown or duplicate fields, truncation and unbalanced allocations. Event
fixtures reject inconsistent transfer triplets, duplicate attributes and invalid
coins. The Comet verifier tests reject altered transactions, deterministic
results, signatures, validator pins, positions, counts and pair links; separate
fixtures confirm that event and diagnostic changes leave the result root
unchanged. These tests verify the readers' stated boundaries, not a historical
application replay.

The isolated worktree is `zerone-knowledge-settlement-history-20260909`, branch
`codex/knowledge-settlement-history-v1`, based on the published census commit
`71cb372b3cde391c2f1661618e175d2436a685d5`. Existing worktrees and stopped observer
evidence are preserved. This report is a source publication; it requires no
runtime deployment.
