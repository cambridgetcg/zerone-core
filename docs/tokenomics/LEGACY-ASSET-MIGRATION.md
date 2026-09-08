# Legacy assets — historical snapshot allocation

**Status: adopted direction; implementation terms and activation unresolved.**
The selected direction is an **opt-in historical snapshot allocation at nominal
1:1**: one successor `uzrn` per approved, reconciled legacy `uzrn` entitlement
(1 ZRN = 1,000,000 uzrn on each chain). This is not an approved claimant list,
a funded offer, a signed transition decision, or an authorization to pay.

This policy supplements the [relaunch balance policy](../plans/2026-07-12-zerone-2-relaunch.md#balance-policy),
not its genesis or notice commitments. `zerone-2` retains the exact 13,555-ZRN
mechanical genesis scaffold; no legacy bank or module balances are copied into
it. The existing transition notice and its exact-byte publication gates remain
unchanged. Neither this document nor an accounting report changes signed notice
bytes or supplies missing release, checkpoint, or activation authority.

## What the selected direction means

The allocation is based on a separately approved historical checkpoint and
reconciled beneficial entitlements, not possession of a token at claim time.
After the cutoff, a purchase or transfer of the source asset does **not**
automatically carry its historical allocation entitlement. Any distinct
assignment of an entitlement would require separately adopted rules; none are
established here. No one should buy a legacy asset on the assumption that doing
so acquires a successor claim.

Opting in would request the approved successor allocation through a later
audited mechanism. It is not exclusive redemption: a successor claim would not
burn, extinguish, redeem, or transfer the source asset, its voucher, or a
third-party claim against a custodian. The source record and any independently
persisting assets remain distinct. Their persistence does not authorize a
second allocation against the same backing. Nominal 1:1 is a unit-counting rule,
**not market-value parity, liquidity, convertibility, a price guarantee, or a
guarantee of payment**.

## Terms still requiring evidence and approval

| Boundary | Unresolved requirement |
| --- | --- |
| Checkpoint identity | Exact source chain/genesis, state `F`, anchor `A=F+1`, halt trigger `H=A+1`, hashes, and authenticated source package; no rehearsal height is selected here. |
| Eligibility | Beneficial owners, covered holdings, restrictions, custody disputes, duplicate claims, and treatment of operator/genesis holdings; a bank location or address label approves none of them. |
| Funding | Actual segregated successor reserves and their lawful source; no reserve balance, donor commitment, or funded total is established. |
| Issuance | Applicable permissions remain unresolved. This direction authorizes no reserve mint, genesis allocation, cap exception, or activation of an existing mint lane. |
| Adoption and correction | Named legitimate policy, eligibility, dispute/correction, funding, and activation authorities and their approval process; document authorship is not that authority. |

Policy adoption, authenticated source evidence, correctness of any later
entitlement root, actual funding, and runtime activation are separate checks.
A matching hash binds bytes; a signature must be checked against the appropriate
independently established authority and exact scope. Neither alone settles the
other checks. This first slice produces **no claim root**.

## Count custody once; reconcile beneficial ownership separately

The native bank inventory records **where existing `uzrn` is held**, not who is
entitled to a successor allocation. Sum each location once. A liability or
receipt may explain who has an interest in that balance; it is not additional
native supply to add on top of its backing.

| Holding or evidence | Reconciliation boundary |
| --- | --- |
| Direct bank balance, including operator accounts | Control of a location is not proof of undisputed beneficial ownership or eligibility; no automatic inclusion or exclusion by label. |
| SDK/custom staking and unbonding | Module custody and delegator/unbonding ledgers describe backing and interests in it, not two supplies. Preserve locks, unbonding conditions, and unresolved deficits; bonded-validator token totals are not extra balances. |
| Vesting or permanent locks | An account-type label does not establish its spendable balance or complete schedule. Do not treat nominal allocation as permission to accelerate or erase restrictions. |
| Module custody, escrow, or custodial services | Reconcile beneficiaries and liabilities to actual backing before approving anything; a module name does not make its operator the beneficiary. |
| LP shares and pool reserves | Shares represent interests in reserves; do not allocate both the full native reserve and another full native amount for the shares. Non-native pool assets need their own accounting. |
| IBC escrow and vouchers | Identify origin, denomination trace, backing, and beneficial holders across chains; do not count native escrow and its remote voucher as independent native entitlements. A future successor claim does not extinguish the voucher. |
| Rewards, pots, and contingent promises | Separate already-existing supply/backing from accrued liabilities, contingent rewards, and unfunded promises. Configuration or a projected reward is not funded native supply. |

The [offline accounting specification](../specs/legacy-snapshot-accounting-v0.md)
covers native `uzrn` custody only. It cannot resolve the above interests from a
snapshot-v3 inventory. Every row remains `UNDETERMINED`; entitlement total and
reserve requirement remain unknown, not zero. Post-anchor `A` custody/export
checks, including a custom-staking census, cannot substitute for checkpoint-`F`
entitlement evidence.

This narrow tool does not narrow the obligation to account for **all legacy
assets and associated liabilities**, including non-native assets, vouchers,
receipts, and disputed or unsupported holdings. Missing coverage is a gap, not
forfeiture, a finding of worthlessness, or administrator ownership. Preserve the
source evidence and unresolved interests for separately reviewed reconciliation.

## Proposed safeguards and later funding boundary

**Proposed safeguards, not adopted economic terms or signed policy:** no
automatic claim expiry, forfeiture for nonparticipation, or treasury sweep of
unclaimed, disputed, or unresolved allocations. A later adopted policy must
state how corrections and disputes are handled; this draft does not invent a
claim deadline, administrator residual entitlement, or authority to dispose of
unclaimed reserves. Opt-in is not a waiver of unrelated legacy interests.

A later transfer-only claims mechanism needs audited, invariant-checked rules
and actual segregated successor reserves covering every approved unpaid
entitlement. Those same coins cannot simultaneously back staking, liquidity,
bridge escrow, or another claim reserve. Issuance permissions and any proposed
funding source require their own approval; this document neither funds that
reserve nor assigns the genesis scaffold to it. Reconciliation is not solvency,
and a successful offline report authorizes no transfer.

## Staged public beta is not claims launch

The first public release is a **public custodial beta: one operator-controlled
validator, `f=0`**, with a read-only hosted observer. Already-funded native users
may transact through their own nodes when public P2P is authorized; registration
does not provide gas, funding, or validator membership. This stage provides no
hosted onboarding/broadcast or migration claim service.

Claims, migration payouts, rewards, external IBC/ICA, bridges, and liquidity
remain closed under the existing release profile. This allocation work changes
no genesis, staking funding, protobuf, consensus rule, or runtime latch. Later
activation requires its own reviewed public policy, funding evidence, audited
implementation, and applicable release/governance authority.
