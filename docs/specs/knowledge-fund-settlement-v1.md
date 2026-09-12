# Knowledge fund settlement v1

`knowledge-fund-settlement-v1` advances knowledge **10 → 11**. Its predecessor
is the completed claim-records state machine, deployed on `zerone-dev-1` from
`ac5684e33b6946a31f6ea260ebf280eace55b033`; integration base `a3176fd1` has the
same Go/runtime behavior. Source publication does not itself execute an upgrade.
Legacy `zerone-1`, its observer package and unactivated `zerone-2` are separate.

## State the payment terms before settlement

Every newly admitted paid review request records immutable
`Claim.funding_terms`, independently of `review_policy_version` and scientific
relations. Funding policy 1 preserves the existing total payment and the fixed
55% reviewer allocation. It identifies the payer, payment route, review budget,
refundable deposit and fee already routed away from the knowledge module.

| New admission | Review budget | Refundable deposit | Retained fee |
| --- | --- | --- | --- |
| Ordinary claim or conjecture | floor(payment × 55%) | 0 | remainder |
| Contradiction or either explicit challenge | floor(payment × 55%) | remainder | 0 |

The ordinary retained portion still follows the existing protocol, development
and research allocation. Network transaction fees remain separate. An ordinary
claim carrying a `CONTRADICTS` relation still uses ordinary claim payment terms;
the actual admission message determines the funding route.

New terms require current neutral review, canonical positive `uzrn` amounts
within the existing uint64 accounting range and a canonical payer. The terms,
payer and payment cannot change. A funded claim has exactly one selected round,
and its claim/round payment history cannot be deleted through keeper APIs.
Raw encodings are checked before protobuf decoding can merge duplicate messages
or truncate numeric values. Unknown funding policies and pre-activation funding
fields are refused.

## Account for every terminal outcome

Valid, timely authenticated reveals divide the fixed review budget equally,
including dissent, malformed votes and inconclusive or under-quorum rounds.
The first recorded eligible recipient receives the integer division remainder.
No new agreement reward, withholding or scientific disagreement penalty is
introduced. A structurally valid review is not proof of diligence or truth.

At any completed verdict—accepted, rejected, malformed or inconclusive—the
refundable deposit returns to the original payer. If there are no eligible
reveals, the unused review budget also returns. No reviewer payment plan is
invented when nobody reviewed. Consequently:

- An ordinary unattended request returns its 55% review budget; the routed
  remainder and transaction fees remain spent.
- A new unattended challenge or contradiction returns its entire deposit;
  transaction fees remain spent.
- A new reviewed challenge or contradiction pays the 55% review budget and
  returns the remainder regardless of the panel verdict.

For the participant client's default 11 ZRN contradiction deposit, eligible
reviewers share 6.05 ZRN and the payer receives 4.95 ZRN. With no eligible
reviews, the payer receives 11 ZRN. These are returned deposited funds, not
newly minted rewards. A conjecture's acceptance remains a judgment of whether
it is well-posed and falsifiable, not a finding that its proposition is true.

## Record obligations separately from transfers

A completed round freezes the existing `verifier_reward_settlement` and, when
positive, a new `claim_refund_settlement` with recipient, amount and creation
height. Both use the existing bounded pending-payment index and cursor. A
refund-only round is also queued. The per-block ceiling remains 16 examined
rounds and 513 transfer instructions: at most 512 reviewers plus one refund,
or the existing legacy withholding transfer.

All reviewer transfers, the refund and their paid markers commit in one SDK
cache. A failed bank leg or record/index write transfers nothing and leaves
the same plans pending. Retry neither recomputes the amounts nor repeats a
completed transfer. New refund events distinguish an accrued obligation from
an actual successful payment. Fund balances are fungible; the recorded terms
and pending plans, not a module balance alone, identify particular obligations.
These paths do not mint tokens or guarantee that an underfunded historical
module can immediately discharge every obligation.

The financial classification does not set `ProvisionalFactId` on a
contradiction or otherwise change scientific challenge resolution. Historical
explicit challenge settlement runs only for claims without new funding terms.

## Preserve the predecessor honestly

Migration does not alter old claims, rounds, payment plans, scientific history,
bank balances or supply. An old claim with absent funding terms keeps that
absence. Existing reviewer plans retain their exact recipients, amounts and
retry paths, including historical frozen obligations with an absent parent
where already supported. A corrupt or unreadable record is not treated as an
absent record. A new refund always requires its recorded funded parent.

Old unused reviewer pools and contradiction remainders do **not** become
retroactive refund promises. They require a separately reported inventory and
an explicit disposition decision; this upgrade neither sweeps those funds nor
labels them paid. The public reader must distinguish these historical records
from new explicit terms.

## Exact upgrade and replay boundary

The named positive-height plan has empty `info` and no deprecated fields. The
candidate admits only the exact complete knowledge-10 predecessor map at
committed H−1 with matching local and on-chain plans. Earlier named handlers
retain their frozen target maps. Mixed versions, premature startup, unsafe
skips and inconsistent marker/done metadata are refused.

The owning migration uses one cache and the existing bounded record inventory
to refuse preseeded funding fields, inconsistent references or missing pending
payment-index entries before enabling future admissions. Native current genesis
explicitly selects `fund_settlement_enabled=true`, with its prerequisite
record-integrity, review-neutrality and claim-record flags. Export/import keeps
the flag and exact financial records without inventing an applied-upgrade receipt.

The original shared-chain genesis remains unchanged. A new follower replays
it with the verified predecessor binary, then crosses its recorded upgrade
height using the pinned successor. Runtime staging binds the chain, original
genesis/descriptor, named plan, height, sources and both platform binaries.
Staging alone does not authorize an automatic replacement of a running node;
the explicit upgrade runner verifies the actual halt and application lineage.
Existing participant homes, identities and pending review journals remain
bound to their original packages unless explicitly migrated. Older decoders
must not be described as displaying the newly added financial records.

## Verification and limits

Tests cover each paid admission route and terminal outcome with actual SDK
bank transfers, failed final bank/storage legs, refund-only retry, the maximum
review panel, no duplicate payment, malformed records, immutable terms and
restart/export/import. A signed two-binary rehearsal exercises SDK governance,
the H−1 halt, unchanged historical claims, new review/refund bank deltas and
restart. Runtime/client compatibility has a separate staged-supervisor trial.
Synthetic participants establish implementation behavior, not independent
scientific participation or production token value.

This consolidation does not tune gas prices, increase payouts, issue research
grants, establish Sybil resistance or introduce a new reputation system.
