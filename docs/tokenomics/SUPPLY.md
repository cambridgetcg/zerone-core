# Supply Cap and Issuance Surfaces

> **Status:** source target after H2. H1 `consolidation-safety-v1` preserves
> vesting_rewards V1; H2 `founder-renunciation-v1` alone advances it to V2 and
> retires the transaction-presence block mint. Supply on a running network must
> be queried at a bound height; source defaults and projections are not balances.

## Hard cap

```text
222,222,222 ZRN = 222,222,222,000,000 uzrn
```

`MintWithCap` checks current bank supply before each wired native mint, and
`InitChain` rejects genesis bank supply above the same cap. The cap is on
current circulating plus locked supply, not cumulative tokens ever minted.
Rejected substrate-attestation bonds can burn, so a burn may reopen headroom.

The cap is a quantity boundary. It does not prove that a recipient, trigger,
or authority is legitimate; each mint lane needs a separate economic and
replay review.

## Historical genesis

`zerone-1` began with 13,555 ZRN (0.0061% of cap):

| Holder/purpose | Amount |
|---|---:|
| Launch validator collateral and gas | 11,333 ZRN |
| Transferable operations float | 2,222 ZRN |

These are disclosed operator-controlled balances. There was no team,
foundation, investor-sale, research-fund, or faucet allocation. The immutable
genesis artifact and manifest remain the source for launch accounting; H1 does
not rewrite them.

## Automatic block issuance is retired

The launch source had a decaying transaction-bearing block-reward formula with
a 10 ZRN base, 0.1 ZRN floor, validator-count scaling, and knowledge coupling.
That lane is historical after H2.

The problem was entitlement, not only quantity: the proposer could include a
decodable/stateless-valid transaction before signature, fee, balance, or
successful execution was known, and raw inclusion triggered its own reward.
Even restricting the trigger to a successful arbitrary transaction would
permit self-churn.

Vesting_rewards v2 therefore requires:

| Compatibility field | Post-H2 value |
|---|---:|
| `block_reward` | `"0"` |
| `floor_reward` | `"0"` |
| `empty_block_reward_rate` | `0` |

BeginBlock routes real fees but performs no transaction-presence mint. The old
decay curve and multi-year block-emission table are not post-H2 supply
projections.

## Remaining source-capable mint lanes

| Pathway | Module | Trigger | Recipient/control note |
|---|---|---|---|
| Claiming-pot claim | `x/claiming_pot` | Eligible claimant calls `MsgClaim` | Bootstrap and legacy authority-created general pots share a bounded lifetime budget |
| External-work attestation | `x/substrate_bridge` | Bonded attestation survives configured settlement/challenge rules | Adapter registration, source uniqueness, challenge survival, and recipient must be reviewed together |
| Probe-bounty pool | `x/knowledge` | Governance sets a positive rate | Default and published rate are zero |
| Emission period | `x/tokens` | Governance enables latch and schedules a period | Default and published latch are disabled; recipient is authority-selected |

Training-fund disbursement and contribution-challenge bonus minting are
release-sealed in the reviewed source. Planned mechanisms and protobuf fields
are not issuance until a production caller, authority path, and bank transfer
exist.

## Fees are transfers, not issuance

Transaction fees move existing ZRN. After H1, accumulated `uzrn` fees route
19.67% to development, 3.33% to research, and approximately 77% through normal
Cosmos distribution.

A native AMM's configured fee also moves no new ZRN. It remains in pool
reserves and increases the assets backing LP shares. Liquiditypool v5 routes
zero protocol skim to `fee_collector`.

## Burns and locked supply

There is no general burn share. Rejected substrate-attestation bonds are the
narrow native-ZRN burn path. LP-token burns redeem reserves and do not destroy
ZRN unless another independent rule explicitly burns an underlying asset.

Not all current supply is liquid: bonded stake, vesting schedules/reserves,
governance stakes, challenge escrows, external-work bonds, qualification
stakes, module funds, and AMM reserves each have separate release rules. Query
the owning module rather than subtracting a generic “locked” percentage.

## Projection discipline

There is no deterministic calendar for reaching the cap after automatic block
issuance retirement. Remaining issuance depends on claims, witnessed work,
governance-disabled lanes, burns, and future named upgrades. Any forecast must
state:

- starting chain ID, height, app hash, and bank supply;
- each enabled mint lane and its cap/budget;
- authority and recipient assumptions;
- challenge, replay, failure, and burn rates; and
- whether the scenario assumes a future consensus change.

See [Economic neutrality](ECONOMIC-NEUTRALITY.md),
[Genesis](GENESIS.md), and [Sources, locks, and flows](SINKS-AND-FLOWS.md).
