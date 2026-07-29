# Revenue Split

> **Implementation status (2026-07-29):** This page documents the current
> source tree. It does not claim that every conceptual Zerone service routes
> through one universal function. The concrete runtime paths are
> `vesting_rewards.DistributeBlockReward`,
> `vesting_rewards.DistributeRevenue`, `vesting_rewards.RouteFees`, and the
> ZRN-input swap-fee transfer in `liquiditypool`.

## Block rewards

An eligible block reward is cap-gated, activity-dependent, validator-scaled,
and knowledge-coupled before it is minted. `DistributeBlockReward` then applies
the governance-adjustable four-way split:

| Share | Default | Current destination |
|---|---:|---|
| Contributor | 55% | Block producer |
| Protocol | 22% | 50% citation reserve, 30% knowledge verification pool, 20% treasury reserve |
| Development | 19.67% | `development_fund` module account |
| Research | 3.33% | `research_fund`, less an active founder sub-share |

The four primary values use a 1,000,000 BPS scale and must sum to 1,000,000.
Development is calculated as the remainder during routing so integer rounding
cannot leak value.

The citation and treasury parts of the protocol share currently remain in the
`vesting_rewards` module account; no separate citation or treasury module is
wired. The full verification part goes to `knowledge`. The removed
`compute_pool` module receives nothing.

## Transaction fees

`RouteFees` treats accumulated `uzrn` fees differently from newly minted block
rewards:

- 19.67% moves from `fee_collector` to `development_fund`;
- 3.33% is deposited through the research/founder routing path; and
- the remaining approximately 77% stays in `fee_collector` for normal Cosmos
  distribution.

The contributor/protocol labels therefore do not describe distinct
transaction-fee destinations. Non-`uzrn` balances are not split by
`RouteFees`.

## Liquidity-pool fees

On ZRN-input swaps, the governance-set protocol share of the swap fee moves to
`fee_collector`, where `RouteFees` handles it as `uzrn`. Counter-denom-input
swaps take no protocol share; their fee remains with liquidity providers.

## Founder sub-share

The founder sub-share is a percentage of the research slice, not a fifth
primary share. It is inactive while `founder_address` is empty.

| Parameter | Governance contract |
|---|---|
| `founder_share_bps` | Mutable within 0–70,000 BPS (0–7% of research) |
| `founder_address` | May be set while empty; immutable once set |

At genesis the address is empty, so the sub-share pays 0 ZRN. This says
nothing about the separately disclosed operator-controlled genesis balances.

## What is not implemented

Older designs named `billing`, `toolbox`, `tree`, `disputes`, `channels`,
`bvm`, `compute_pool`, and other services as universal revenue sources. Those
modules are not present in this source inventory, and this document does not
represent their proposed flows as runtime behavior. `x/common` defines shared
split message types; that schema alone does not make every module a caller of
`DistributeRevenue`.

There is no general revenue burn share. Rejected substrate-attestation bonds
are a separate, narrow punitive ZRN burn path.
