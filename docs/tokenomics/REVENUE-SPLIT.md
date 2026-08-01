# Revenue and Fee Routing

> **Implementation status:** this page describes the source target after the
> H1 liquidity release and later H2 founder-renunciation release. Query a
> release-matched network at a bound height before treating it as live state.

## Actual transaction fees

`vesting_rewards.RouteFees` runs before standard Cosmos distribution and
handles accumulated `uzrn` in `fee_collector`:

- 19.67% moves to `development_fund`;
- 3.33% moves in full to `research_fund`; and
- approximately 77% remains for normal Cosmos distribution.

The values come from the current `RevenueSplit` Params. Integer rounding and
aggregation in `fee_collector` mean the split is an aggregate rule rather than
an exact per-transaction receipt. Non-`uzrn` balances are not split by this
custom router.

The legacy contributor/protocol labels do not create distinct transaction-fee
destinations: their combined 77% remains in `fee_collector`. The normal Cosmos
distribution path may allocate community tax, validator commission, and
delegator rewards according to its own on-chain state.

## Liquidity-pool fees

Liquiditypool v5 has no protocol skim in either swap direction.
`protocol_fee_bps` remains on the wire for compatibility but is fixed at zero
and governance cannot set it nonzero.

The configured pool fee is still real. It is incorporated into the
constant-product quote and the complete input amount remains in recorded pool
reserves after output leaves. The fee therefore increases the assets backing
all transferable LP shares pro rata. It compensates funded capital and pool
risk; it does not flow to governance, `fee_collector`, a creator title, or a
founder address. Swap events retain `protocol_fee = 0` for compatibility.

## Retired automatic block split

Before vesting_rewards v2, transaction presence could call
`DistributeBlockReward` and route a new mint through a four-way split. A
proposer controlled inclusion before
signature, fee, balance, or successful execution was known, so the trigger did
not prove useful work.

Vesting_rewards v2 fixes `block_reward`, `floor_reward`, and
`empty_block_reward_rate` at zero and removes that BeginBlock call. The former
55/22/19.67/3.33 minted-reward projection is historical, not a post-H2 revenue
promise. Actual transaction-fee routing above remains active.

## Retired founder sub-share

`founder_share_bps` and `founder_address` remain compatibility fields but must
be zero/empty after `founder-renunciation-v1`. Every canonical research
deposit reaches `research_fund` in full, and an ordinary Params proposal
cannot restore the old auto-split.

The published genesis recorded a 7%-of-research setting with no founder
address, so the tap was dormant at launch. H2 does not rewrite that artifact or
claw back history; it permanently removes prospective activation. Any future
payment to a founder is an ordinary publicly authorised grant, not a protocol
percentage.

## Other reward routing

Some keeper types still express contributor, protocol, research, and
development fields because other modules and historical records use the wire
shape. A schema is not a mint trigger. Integrators must identify the concrete
caller, source balance, bank transfer, and activation state before describing
any value flow.

There is no general revenue burn share. Rejected substrate-attestation bonds
are a separate punitive burn path. See [Economic neutrality](ECONOMIC-NEUTRALITY.md)
and [Sources, locks, and flows](SINKS-AND-FLOWS.md).
