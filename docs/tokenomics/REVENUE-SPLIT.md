# Revenue Split

> **Implementation status (2026-08-01):** This page documents the current
> source tree. It does not claim that every conceptual Zerone service routes
> through one universal function. The concrete runtime paths are
> retired `vesting_rewards.DistributeBlockReward` compatibility method,
> `vesting_rewards.DistributeRevenue`, `vesting_rewards.RouteFees`, and the
> ZRN-input swap-fee transfer in `liquiditypool`.

## Automatic block rewards are retired

Consensus v2 never mints because a block contains a transaction. Proposal
inclusion is controlled by the proposer and is not independently witnessed
successful work. `DistributeBlockReward` remains callable by old Go
integrations but always returns a zero distribution and performs no mint,
transfer, ledger write, or event. The v1→v2 migration pins `block_reward`,
`floor_reward`, and `empty_block_reward_rate` to zero.

`DistributeRevenue` remains a calculation helper for an explicit
caller-supplied amount and applies the four-way split:

| Share | Default | Current destination |
|---|---:|---|
| Contributor | 55% | Block producer |
| Protocol | 22% | 50% citation reserve, 30% knowledge verification pool, 20% treasury reserve |
| Development | 19.67% | `development_fund` module account |
| Research | 3.33% | `research_fund` in full |

The four primary values use a 1,000,000 BPS scale and must sum to 1,000,000.
Development is calculated as the remainder during routing so integer rounding
cannot leak value.

It does not mint or transfer by itself. Historical block-reward records remain
queryable, but no new consensus path calls the automatic distribution.

## Transaction fees

`RouteFees` treats accumulated `uzrn` fees differently from newly minted block
rewards:

- 19.67% moves from `fee_collector` to `development_fund`;
- 3.33% is deposited in full through the canonical research routing path; and
- the remaining approximately 77% stays in `fee_collector` for normal Cosmos
  distribution.

The contributor/protocol labels therefore do not describe distinct
transaction-fee destinations. Non-`uzrn` balances are not split by
`RouteFees`.

## Liquidity-pool fees

On ZRN-input swaps, the governance-set protocol share of the swap fee moves to
`fee_collector`, where `RouteFees` handles it as `uzrn`. Counter-denom-input
swaps take no protocol share; their fee remains with liquidity providers.

## Founder recipient is retired

Vesting-rewards consensus v2 permanently retires the former founder
sub-share. Its protobuf fields remain readable for historical/wire
compatibility, but they are not control surfaces.

| Parameter | Governance contract |
|---|---|
| `founder_share_bps` | Fixed at `0`; validation and storage reject any other value |
| `founder_address` | Fixed empty; validation and storage reject any recipient |

The v1→v2 migration clears either legacy field regardless of its old value;
all v2+ deposits route the complete research allocation to `research_fund`.
Historical records and field numbers are preserved. This source change is
activated only after `founder-renunciation-v1` is separately scheduled and
accepted; it is not a claim about the currently running binary. It also says nothing
about separately disclosed operator-controlled balances, ordinary permissionless
participation, validator control, or stake-weighted governance.

## What is not implemented

Older designs named `billing`, `toolbox`, `tree`, `disputes`, `channels`,
`bvm`, `compute_pool`, and other services as universal revenue sources. Those
modules are not present in this source inventory, and this document does not
represent their proposed flows as runtime behavior. `x/common` defines shared
split message types; that schema alone does not make every module a caller of
`DistributeRevenue`.

There is no general revenue burn share. Rejected substrate-attestation bonds
are a separate, narrow punitive ZRN burn path.
