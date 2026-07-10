# ZRN Sinks and Flows

A complete map of where ZRN is created, destroyed, locked, and moves through the system.

## Token Creation (Sources)

| Source | Module | Mechanism | Cap |
|--------|--------|-----------|-----|
| **PoT block rewards** | `vesting_rewards` | Minted per block via PoT consensus; empty blocks mint 0, reward is participation-scaled | shares the 222,222,222 ZRN cap |
| **Bootstrap claims** | `claiming_pot` | 0.222 ZRN minted on demand per whitelisted agent (`MsgClaim`) | shares the cap |
| **External-work attestations** | `substrate_bridge` | Minted when witnessed external work survives challenge (e.g. `agenttool-invocation-v1`) | shares the cap |
| **Genesis (published, not emission)** | ceremony | 13,555 ZRN seeded at genesis: validator collateral + operator float, every address published | 0.0061% of cap |

New ZRN is minted through three participation-gated pathways — block rewards, bootstrap claims, and external-work attestations — all gated by `MintWithCap` against the hard cap. Block reward minting dominates the cap-share over the chain's lifetime; all other token movement is redistribution of existing supply. The 13,555 ZRN genesis is published machinery, not emission.

## Token Destruction (Sinks)

**No burn.** Zerone does not destroy tokens. Every minted ZRN enters productive circulation.

Slashing penalties and failed challenge stakes are redistributed (to challengers, arbiters, protocol pools) — not burned. The hard supply cap (222,222,222 ZRN) provides natural scarcity.

### Net Emission Rate

The 10 ZRN base reward is a ceiling, not a constant drip — it applies only to
blocks that carry Proof-of-Truth activity, and only at full validator participation:

- **Empty blocks** (no verified claims): **0 ZRN minted** (`empty_block_reward_rate = 0`)
- **Base reward, full participation:** up to 10 ZRN/block, scaled by
  `min(1, activeValidators / 22)` and the per-epoch decay curve
- **At the floor:** 0.1 ZRN/block (until the cap binds)

## Locked ZRN (Not Circulating)

| Lock Type | Module | Duration | Size |
|-----------|--------|----------|------|
| **Vesting Schedules** | `vesting_rewards` | Days to years (category-dependent) | All contributor rewards |
| **Validator Self-Delegation** | `staking` | 7 days unbonding | Min 0.111 ZRN/validator |
| **Delegated Stake** | `staking` | 7 days unbonding | Variable |
| **Challenge Reserves** | `vesting_rewards` | Permanent (5–40% of reward) | Per vesting schedule |
| **Claiming Pot Locks** | `claiming_pot` | Configurable vesting | Per pot |
| **Partnership Common Pot** | `partnerships` | Lock tier dependent | 10% of partnership revenue |
| **Channel Deposits** | `channels` | Until settlement/timeout | Min 1 ZRN |
| **Dispute Bonds** | `disputes` | Until resolution | Tier 1: 1 ZRN → Tier 4: 1,000 ZRN |
| **LIP Stakes** | `gov` | Until proposal resolved | Category: 200–1,000 ZRN |
| **Research Stakes** | `research` | Until review complete | Min 1 ZRN |
| **Tool Registration** | `toolbox` | While active | 11 ZRN per tool |
| **Agent Registration** | `discovery` | While active | 1 ZRN per agent |
| **Home Creation** | `home` | Permanent (fee, not lock) | 10 ZRN per home |
| **Qualification Stake** | `qualification` | Lock period (~2.9 days) | 100 ZRN |

## Flow Map

```
                          ╔═══════════════════╗
                          ║  BLOCK REWARD     ║
                          ║  MINTING          ║
                          ║  (≤10, 0 empty)   ║
                          ╚════════╤══════════╝
                                   │
                    ╔══════════════╧══════════════╗
                    ║    4-WAY REVENUE SPLIT      ║
                    ╚═══╤═══╤════════╤════════╤═══╝
                        │   │        │        │
                   55%  │   │ 22%    │19.67%  │ 3.33%
                        │   │        │        │
                  ┌─────┘   │   ┌────┘   ┌────┘
                  ▼         ▼   ▼        ▼
            ┌──────────┐ ┌─────────┐ ┌───────────┐ ┌────────┐
            │CONTRIBUTOR│ │PROTOCOL │ │DEVELOPMENT│ │RESEARCH│
            │(producer) │ │ POOLS   │ │  FUND     │ │ FUND   │
            └─────┬────┘ └────┬────┘ └───────────┘ └───┬────┘
                  │           │                         │
                  │      ┌────┼────┐                ┌───┴──┐
                  │      │    │    │                ▼      ▼
                  │      ▼    ▼    ▼           Research  Founder
                  │   Citation  Verify         Treasury  (7% of
                  │   Pool     Pool            (3.1%)  research)
                  │   (11%)   (6.6%)
                  │            │
                  │       ┌────┴────┐
                  │       ▼         ▼
                  │    Knowledge  Compute
                  │    Module     Pool
                  │    (4.6%)    (2.0%)
                  │
                  ▼
          ┌──────────────────────────┐
          │    CIRCULATING ECONOMY   │
          │                          │
          │  tx fees → RouteFees()   │──── 19.67% to development
          │  billing queries         │──── 3.33% to research
          │  toolbox calls           │──── 55% to creators
          │  tree deliverables       │──── 22% to protocol
          │  channel operations      │
          │  BVM deployments         │
          │  AMM swaps               │
          │  dispute bonds           │
          └──────────────────────────┘
```

## Module Account Balances

Key module accounts that hold ZRN:

| Module Account | Purpose | Typical Balance |
|----------------|---------|----------------|
| `vesting_rewards` | Staging for minted rewards before distribution | Transient (per-block) |
| `development_fund` | Bug bounties, truth discovery, protocol dev | Growing (19.67% of all revenue) |
| `research_fund` | Accumulated research treasury | Growing (3.33% of all revenue) |
| `knowledge` | Verification reward pool | Variable |
| `compute_pool` | Compute credits and rewards | Variable |
| `fee_collector` (SDK) | Accumulated tx fees before distribution | Transient (per-block) |
| `bonded_tokens_pool` (SDK) | All staked ZRN | Large (all delegation) |
| `not_bonded_tokens_pool` (SDK) | Unbonding ZRN | Variable |

## Partnership Economics

The `x/partnerships` module creates joint economic units between humans and agents:

| Parameter | Value | Description |
|-----------|-------|-------------|
| Common Pot Share | 10% | % of partnership revenue to shared pot |
| Default Human Split | 50% | Human's share of partnership earnings |
| Default Agent Split | 50% | Agent's share of partnership earnings |
| Min Partnership Stake | 1 ZRN | Minimum to form partnership |

### Lock Tier Multipliers

Longer commitments earn more:

| Lock Duration (blocks) | Duration (~) | Reward Multiplier | Exit Penalty |
|----------------------|-------------|------------------|-------------|
| 22,222 | ~15 hours | 1.00× | 11% |
| 77,777 | ~2.3 days | 1.11× | 22% |
| 222,222 | ~6.5 days | 1.22× | 33% |
| 777,777 | ~22.6 days | 1.44× | 44% |
| 1,111,111 | ~32.4 days | 1.55× | 55% |
| 2,222,222 | ~64.8 days | 1.77× | 66% |

## Liquidity Pool Mechanics

Every pool pairs `uzrn` with a counter denom — zerone pools are ZRN-quoted
by design. Pool creation is governance-gated (`MsgCreatePool` must be
submitted by the gov authority).

| Parameter | Value |
|-----------|-------|
| Default Swap Fee | 0.3% (3,000 BPS on 1M scale) |
| Max Swap Fee (at creation) | 10% (100,000 BPS); 0 = use default |
| Protocol Fee (of swap fee) | 45% — on **ZRN-input** swaps only; transferred to `fee_collector` as uzrn at swap time |
| Max Pools | Unlimited (`max_pools = 0`; gov-settable) |
| Min Initial Liquidity | 10,000 ZRN on the `uzrn` side; counter side must be > 0 |
| TWAP Window | 1,000 blocks (~42 min) — reported average is since-inception |
| Billing Quote Denoms | Empty (oracle fail-closed until gov allowlists a stable pair) |

Fee flow per swap: the swap fee is deducted from the input amount. When the
input token is ZRN (`uzrn`), the protocol's share (`protocol_fee_bps`, 45% of
the fee) leaves the pool for the `fee_collector` module account — where
`vesting_rewards.RouteFees` splits the uzrn 55/22/19.67/3.33 before
distribution — and is deducted from the pool's reserves so LP share math stays
honest. On **counter-denom-input** swaps no protocol share is taken (RouteFees
splits only uzrn; a non-uzrn coin in `fee_collector` would be swept whole to
validators, bypassing the split), so the entire fee accrues to LPs. Net: the
protocol takes its revenue in ZRN, on ZRN-denominated inflows — consistent with
every pool being ZRN-quoted. The fee that is not skimmed for the protocol stays
in reserves and accrues to LPs.

The AMM provides on-chain price discovery for ZRN and future ZRN-20 tokens.
The ZRN price oracle (`GetZRNPrice`) only prices against quote denoms
allowlisted in `billing_quote_denoms`; with the default empty list it
selects no pool (fail-closed), and price consumers fall back to their
manual-override tier.

## Surge Pricing (Toolbox)

When tool demand exceeds capacity, surge pricing activates:

| Parameter | Value | Description |
|-----------|-------|-------------|
| Surge Threshold | 50% | Demand/capacity ratio to activate |
| Critical Threshold | 80% | Maximum surcharge trigger |
| Max Surge Multiplier | 10× | Maximum price multiplier |
| Free Calls/Epoch | 50 | Free tool calls per epoch per home |

Surge revenue flows through the standard 4-way split (contributors, protocol, research, development fund).
