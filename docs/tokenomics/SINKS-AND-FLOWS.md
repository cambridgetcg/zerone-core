# ZRN Sources, Locks, and Flows

> **Implementation status:** source-backed post-H1/H2 map, not a live-network
> snapshot and not an inventory of conceptual modules.

## Supply creation

The custodial genesis created 13,555 ZRN: 11,333 ZRN of validator
collateral/gas plus a transferable 2,222 ZRN operations float. Every wired
post-genesis native mint call shares the 222,222,222 ZRN `MintWithCap`
boundary.

After the H2 vesting boundary, the remaining source-capable mint lanes are:

| Source | Module | Trigger |
|---|---|---|
| Claiming-pot claim | `claiming_pot` | Eligible address claims a bootstrap or legacy governance-created general pot |
| External-work reward | `substrate_bridge` | Registered-adapter attestation survives its settlement/challenge rules |
| Probe-bounty pool (default/published rate 0) | `knowledge` | Governance configures a positive rate |
| Emission period (default/published latch 0) | `tokens` | Governance enables and schedules an authority-selected recipient |

Transaction-bearing block minting is retired in vesting_rewards v2. A raw or
successful arbitrary transaction is not independently witnessed useful work.
The cap limits quantity but does not validate a trigger; each remaining lane
retains its own evidence, authority, replay, and recipient review.

## ZRN destruction

There is no general burn allocation. Rejected substrate attestations are the
narrow ZRN exception: their submitted bonds burn on rejection. Knowledge
challenge and staking penalties follow their own accounting and must not be
described as a universal burn.

Liquidity-pool share tokens burn on withdrawal, but LP denoms are reserve
receipts and not ZRN supply.

## Locks and escrows

| Mechanism | Module | Runtime boundary |
|---|---|---|
| Validator self-delegation/delegation | `staking` | Locked through unbonding rules |
| Epistemic vesting/reserves | `vesting_rewards` | Existing schedules release by category; adjudicated falsification can trigger gated clawback |
| Governance proposal stake | `gov` | Held for proposal lifecycle |
| Review fees/challenge stakes | `knowledge` | Collected or held through verification/challenge lifecycle |
| External-work bonds | `substrate_bridge` | Returned on accepted settlement or burned on rejection |
| Native liquidity | `liquiditypool` | Reserves represented by transferable LP shares |
| Qualification stake | `qualification` | Held under qualification rules |

## Post-H1/H2 value flow

```text
paid uzrn transaction fees
  └─ fee_collector
      ├─ 19.67% development_fund
      ├─ 3.33% research_fund
      └─ ~77% normal Cosmos distribution

native AMM swap
  └─ configured fee remains in pool reserves
      └─ value backs transferable LP shares pro rata

canonical research deposit
  └─ 100% research_fund

raw transaction presence
  └─ 0 automatic issuance
```

Governance retains admission, pause, oracle, treasury, and upgrade powers, but
ordinary Params cannot reactivate the protocol skim, founder tap, or
transaction-presence mint. See [Revenue and fee routing](REVENUE-SPLIT.md),
[Economic neutrality](ECONOMIC-NEUTRALITY.md), and [Supply](SUPPLY.md).
