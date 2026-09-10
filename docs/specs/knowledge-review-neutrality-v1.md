# Knowledge review neutrality v1

**Source protocol:** `knowledge-review-neutrality-v1`, knowledge consensus 8→9.
The sole predecessor is the completed record-integrity release, frozen at
`c6fa5df13b090a287e69ba1d0e2281d33afa96e8`. Publication of this source does not
upgrade the deployed legacy `zerone-1` or its observer package.

## Keep the contribution and its reasons

A claim records who asserted it. An authenticated review records the signer's
vote, confidence, method, reason, scope and evidence references. The protocol
keeps these statements and their correction history. A panel verdict summarizes
what the participating accounts said; neither stake nor agreement establishes
scientific expertise, independent identity or truth.

New claims carry immutable `review_policy_version=1`, copied into their rounds.
Policy 0 preserves the predecessor's unfinished financial terms. Commitment
scheme 2 remains unchanged: it binds the reviewer and payload to the original
chain and round. The policy is separate from that hash scheme and from block
height, and survives restart/export/import. Unknown policies, mismatches and
policy-1 records before activation are refused. Raw policy encodings are checked
before decoding: overflow, duplicates and wrong wire types cannot masquerade as
legacy policy. The exact predecessor cannot contain a newly introduced policy
field at all. New-policy review deadlines and
recorded timestamps must agree; importing a late reveal cannot make it payable.

## Pay for recorded review work

Policy-1 panels count each admitted account equally. Custom validator stake,
agent-role bonuses, qualification accuracy and capture-derived panel overrides
do not weight those votes or select the new panel. The existing spendable
100 ZRN admission check remains a recyclable balance check, not a locked bond
or proof of independence. The 512-seat hard bound remains. Multiple addresses
may have one controller; this release does not claim Sybil resistance.

All valid, timely reveals share the same existing 55% review-fee pool, including
dissent, malformed outcomes and inconclusive rounds. Under-quorum rounds with
valid reviews complete as inconclusive and can settle that work. Absent or
invalid reveals receive nothing. Integer division and deterministic remainder
handling preserve the fixed pool. No conformity discount changes payment, and
no extra payout is minted. Authenticated reasons are retained assertions; their
presence is not an automatic quality assessment.

A terminal round freezes its payment recipients and amounts. Actual transfers
and the paid marker commit atomically. Underfunded or failed payment remains
pending and uses the existing bounded retry queue. Old frozen plans are never
recomputed. A recorded pending obligation is distinct from a completed payment. Migration
also checks the pending-payment index and cursor against primary unpaid plans,
so a preserved obligation cannot disappear from retry scheduling.

New-policy rounds create no wrong-vote or missed-reveal monetary slash, and no
new vindication-refund promise. Scientific disagreement is not established
fraud. Already recorded escrow and earlier obligations retain their settlement
paths; this release introduces no new collateral or reputation framework.

## Retire automatic score and bonus feedback

After activation, finalizing reviews stops building personal or group standing
from agreement. Qualification accuracy recording, score-based decay/probation
and track-record admission stop; ordinary expiry and custody paths remain.
Existing qualification, calibration, independence and reputation rows remain
historical records, not a fresh measurement of scientific credibility.

New-policy claims create no nominal submitter survival reward, invitation bonus
or successful-challenge bonus. Existing accrued pending rewards, vesting
schedules, collateral and frozen payment plans remain. Correction remains
available, including a correction by the original author. Inconclusive or malformed challenges release
the target from the challenge lock without scientific credit, unless another
active challenge still holds it. A rejected challenge
no longer generates new corroboration, energy or calibration credit. Address
inequality is not used to certify independent challenge or reward eligibility.

Automatic probe issuance, new probe invitations and automatic bootstrap review
sponsorship stop. Existing pool balances are not swept. Paid claim submission
remains available; a related account cannot obtain a free sponsored claim and
collect its review pool under the new policy.

New augmentation bounties and variant admissions stop. Existing ballots,
sponsor vetoes, escrow settlement and refunds retain their recorded terms,
without new agreement-to-qualification feedback. Existing training-fund and
contribution-challenge mint refusals remain in force.

Model owners can still attribute fact IDs to their models. New contribution
records carry `attribution_policy_version=1`, retain the owner declaration, and
have no computed TVW or calibration snapshot. Any raw total weight is explicitly
owner-declared. This is attribution, not a protocol valuation or payout right.
New corpus manifests can use ordinary selectors; the retired calibration-floor
selector is refused. Historical scores, TVW projections and earlier snapshots
remain readable as legacy heuristics, not measured probabilities or competence.

## Exact execution boundary

The plan has the exact name above, positive height H, empty `info`, and no
deprecated time/client fields. The target admits the complete predecessor map
only at committed H−1 with matching on-chain and local plans. Earlier handlers
retain their frozen targets and cannot carry this 8→9 delta. Unsafe skipping,
mixed maps, preseeded policy-1 state and inconsistent prior migration receipts
are refused. The bounded inventory validates retained claims, rounds and history
before selecting the new execution marker in one cache, without rewriting them.

Native/imported current genesis explicitly selects `review_neutrality_enabled`
and record integrity. The execution flag does not invent an applied-upgrade
receipt. Restart checks coherent versions, flags and migration marker/done
metadata. Existing custody and predecessor launch gates remain separate. After activation,
a failed actual challenge-collateral transfer rolls back finalization for both
policy versions, keeping the unfinished round available for retry; it cannot
silently discard an earlier refund obligation.

Verification covers the exact boundary, malformed record refusal, commitment
and timestamp admission, dissent/inconclusive settlement with actual SDK bank
transfers, score retirement, existing obligation preservation and signed local
transactions across two binaries. Disposable local participants and balances
are synthetic; those checks establish behavior, not independent science or
production adoption.

## Remaining work

The ledger still needs independent human and agent evaluation of reasons,
evidence availability and reproducibility. The retained balance gate can be
reused across addresses. Initial energy for new-policy facts omits capture-score pressure, while later
epoch ecology still consumes historical capture flags. Legacy ecological,
training and authority machinery
still exists for compatibility and should be removed only after its outstanding
records and obligations have been inventoried. Block-signing membership and
scientific credibility remain separate concerns.
