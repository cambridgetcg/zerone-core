# Supply Cap and Issuance Surfaces

## Hard cap

```
222,222,222 ZRN = 222,222,222,000,000 uzrn
```

`x/vesting_rewards/types.MaxSupplyUzrn` defines the cap. `MintWithCap`
checks current bank supply before every wired post-genesis native mint, and
`InitChain` rejects a genesis supply above the same boundary.

The cap is on current circulating plus locked supply, not total-ever-created.
Zerone has no general discretionary burn. Rejected substrate-attestation bonds
are the narrow ZRN burn path; a burn can therefore reopen cap headroom.

## Genesis

The live custodial `zerone-1` genesis created 13,555 ZRN (0.0061% of cap):

- 11,333 ZRN controlled by the launch validator, including 11,111 bonded;
- 2,222 ZRN transferable operator float; and
- zero team, foundation, investor-sale, research, faucet, or
  founder-specific stipend allocation.

Those balances are disclosed operator power, not participation-earned supply.
The hash-bound addresses and amounts are in the
[genesis manifest](../../deploy/mainnet/artifacts/GENESIS-MANIFEST.md).

## Source-capable issuance after genesis

| Pathway | Default/source state | Trigger and recipient |
|---|---|---|
| Claiming-pot claim | Enabled within fixed lifetime budget | Eligible claimant calls `MsgClaim` against a bootstrap or legacy general pot |
| External-work attestation | Adapter-dependent | Bonded, source-unique work survives the adapter's settlement/challenge rules |
| Knowledge probe-bounty pool | Rate zero | Governance configures a positive rate; funds the probe-bounty module pool |
| Token emission period | Latch disabled | Governance enables and schedules an authority-selected recipient |

Training-fund disbursement and contribution-challenge bonus minting are
release-sealed. An adapter being present in genesis or source does not prove
that a live network currently exposes or has paid its reward.

## Automatic block issuance is retired

Vesting-rewards consensus v2 performs no automatic per-height,
transaction-presence, proposer, validator-count, decay, or
knowledge-coupling mint. The v1→v2 migration pins:

- `block_reward = "0"`;
- `floor_reward = "0"`; and
- `empty_block_reward_rate = 0`.

Validation and the storage boundary reject restoration. Legacy schedule,
category-multiplier, coupling, query, event, protobuf, and historical-record
shapes remain for compatibility, but they are not live issuance controls.
`DistributeBlockReward` is structurally inert: it mints, transfers, stores,
and emits nothing.

This retirement removes the former exponential-decay and floor-emission
projection. Tables that modeled 10 ZRN transaction-bearing block rewards were
v1 scenarios and must not be used to project v2 supply or validator yield.
Normal Cosmos transaction-fee distribution remains; fees circulate existing
ZRN and do not change total supply.

## Founder revenue is retired

The same migration pins `founder_share_bps=0` and
`founder_address=""`. Research routing transfers the complete research
allocation to `research_fund`; there is no identity-derived founder recipient.
Historical transfers, if any, are not clawed back.

This does not erase the founding household's disclosed validator, operations
balance, or voting control, nor does it prevent a founder from participating
under ordinary permissionless rules.

## Activation boundary

Source publication does not change a running chain. V2 enters consensus only
through the separately audited and scheduled `founder-renunciation-v1`
upgrade after completed H1 evidence. The replacement binary refuses
plan-less, partial, skipped, malformed, or conflicting lineage and requires
strict zero Params and empty vesting module permissions after completion.

See [../FOUNDER-RENUNCIATION-V1.md](../FOUNDER-RENUNCIATION-V1.md) for the
full H1→H2 startup, handler, height-ordering, and provenance contract.

## Long-term model

There is no fixed time-based v2 emission curve. Supply changes only when an
enabled cap-gated issuance pathway executes or a narrow attestation-bond burn
occurs. Operational projections must therefore bind:

1. an exact release and height;
2. live Params and adapter/pot state;
3. observed claimant/work throughput; and
4. current bank supply and remaining cap headroom.

When the cap binds, all `MintWithCap` callers fail closed until a permitted
burn reopens headroom. Fee-funded validation and existing-token circulation do
not depend on new minting.
