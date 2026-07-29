# ZRN Sources, Locks, and Flows

> **Implementation status (2026-07-29):** This is a source-backed runtime map,
> not an inventory of planned modules. Historical designs for partnerships,
> channels, disputes, research staking, toolbox registration, discovery, BVM,
> and compute pools are not represented as shipped behavior here.

## Supply creation

The live custodial genesis created 13,555 ZRN: 11,333 ZRN of validator
collateral/gas and a transferable 2,222 ZRN operations float. Every subsequent
native mint call shares the 222,222,222 ZRN `MintWithCap` boundary:

| Source | Module | Trigger |
|---|---|---|
| Transaction-bearing block reward | `vesting_rewards` | Any non-injection user tx, adjusted for decay, validator participation, and survived-challenge coupling |
| Claiming-pot claim | `claiming_pot` | An eligible address claims a bootstrap or legacy governance-created general pot |
| External-work reward | `substrate_bridge` | A registered adapter's attestation survives its settlement/challenge rules |
| Probe-bounty pool (default/published rate 0) | `knowledge` | Governance configures a positive rate |
| Emission period (default/published latch 0) | `tokens` | Governance enables and schedules an authority-selected recipient |

The 10 ZRN base block reward is a ceiling at the start of the decay curve, not
a reward for every height. Empty user-transaction blocks mint 0 under
defaults; an ordinary transfer qualifies a block, and fewer than 22 active
validators scales issuance down. Training disbursement and
contribution-challenge bonus minting are release-sealed.

## ZRN destruction

There is no general burn allocation. Rejected substrate attestations are the
narrow ZRN exception: the submitted bond is burned on rejection. Failed
knowledge challenges and ordinary staking penalties follow their own module
accounting and must not be described as a universal burn.

Liquidity-pool share tokens are burned on withdrawal, but those LP
denominations are receipts rather than ZRN supply.

## Implemented locks and escrows

| Mechanism | Module | Runtime boundary |
|---|---|---|
| Validator self-delegation and delegation | `staking` | Locked through unbonding rules |
| Epistemic vesting and reserves | `vesting_rewards` | Rewards release by schedule; disproven linked facts can trigger gated clawback |
| Governance proposal stake | `gov` | Held for proposal lifecycle according to category |
| Knowledge review fees and challenge stakes | `knowledge` | Collected or held through verification/challenge lifecycle |
| External-work attestation bonds | `substrate_bridge` | Returned on accepted settlement or burned on rejection |
| Liquidity | `liquiditypool` | Pool reserves represented by LP share tokens |
| Qualification stake | `qualification` | Held while the qualification rules require it |

Exact defaults are source- and network-version-specific. Query an authorised
network at a bound height for operational decisions.

## Current value flow

```text
cap-gated eligible block reward
  └─ vesting_rewards four-way split
      ├─ 55% block producer
      ├─ 22% protocol reserves/pools
      ├─ 19.67% development_fund
      └─ 3.33% research_fund (less active founder sub-share)

uzrn transaction fees
  └─ fee_collector
      ├─ 19.67% development_fund
      ├─ 3.33% research/founder routing
      └─ ~77% normal Cosmos distribution

ZRN-input AMM swap fee
  └─ governance-set protocol portion → fee_collector → fee routing above
```

The founder sub-share is inactive while its address is empty. Governance may
change its percentage within the 7% cap; the address is immutable once set.

See [REVENUE-SPLIT.md](REVENUE-SPLIT.md) for the routing details and
[SUPPLY.md](SUPPLY.md) for activity-dependent emission.
