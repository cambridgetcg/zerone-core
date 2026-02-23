# Revenue Split

Every unit of ZRN revenue — whether from block rewards, transaction fees, billing queries, toolbox calls, tree deliverables, or dispute resolution — is routed through the same **4-way split**. This is the heartbeat of Zerone's economics.

## Primary Split (RevenueSplit)

```
                    ┌─────────────────────────┐
                    │    Total Revenue (100%)   │
                    └─────────┬───────────────┘
            ┌─────────────────┼──────────────────────┐
            │                 │                      │
     ┌──────┴───────┐  ┌─────┴──────┐  ┌───────────┴──────────┐
     │ Contributors │  │  Protocol  │  │ Research   │   Burn   │
     │    55%       │  │    22%     │  │   13%      │   10%    │
     └──────────────┘  └─────┬──────┘  └─────┬──────┴──────────┘
                             │               │
                    ┌────────┼────────┐      │
                    │        │        │      │
              ┌─────┴──┐ ┌──┴───┐ ┌──┴─┐   │
              │Citation│ │Verify│ │Trea│   │
              │  50%   │ │ 30%  │ │20% │   │
              └────────┘ └──┬───┘ └────┘   │
                            │              │
                      ┌─────┴──────┐       │
                      │ 70% Know   │       │
                      │ 30% Compute│  ┌────┴────┐
                      └────────────┘  │ Founder │
                                      │  7% of  │
                                      │ research│
                                      └─────────┘
```

### BPS Values (1,000,000 scale)

| Share | BPS | Percentage | Destination |
|-------|-----|-----------|-------------|
| Contributor | 550,000 | 55% | Block producer / fact submitter / tool creator |
| Protocol | 220,000 | 22% | Split further via ProtocolSubSplit |
| Research | 130,000 | 13% | Research fund (2-of-2 multisig) |
| Burn | 100,000 | 10% | Permanently destroyed |

**Must sum to 1,000,000.** Burn is computed as the remainder after the other three to prevent rounding leaks.

## Protocol Sub-Split

The 22% protocol share is further divided:

| Sub-Share | BPS (of protocol) | Effective % (of total) | Destination |
|-----------|-------------------|----------------------|-------------|
| Citation Pool | 500,000 | 11% | Rewards to cited fact authors |
| Verification Pool | 300,000 | 6.6% | Verification rewards (split below) |
| Treasury | 200,000 | 4.4% | Protocol treasury |

### Verification Pool Split

The verification pool itself is split between two modules:

| Share | BPS (of verification) | Effective % (of total) | Module |
|-------|----------------------|----------------------|--------|
| Knowledge | 700,000 | ~4.6% | `x/knowledge` — verification rewards |
| Compute | 300,000 | ~2.0% | `x/compute_pool` — compute credits |

## Founder Share

A temporary 7% deduction from the research fund portion:

| Parameter | Value | Description |
|-----------|-------|-------------|
| `founder_share_bps` | 70,000 | 7% of the research fund share |
| `founder_address` | "" (disabled at genesis) | Bech32 address |
| `governance_activation_height` | 0 | Block height when share sunsets |

**Effective founder income:**
- 7% of 13% = **0.91% of total revenue**
- This goes directly to the founder's address (not locked/vested)
- If founder address is empty or invalid, 100% goes to research fund

**Sunset mechanism:** When `governance_activation_height` is reached (set by governance vote), the founder share drops to zero and the full 13% research share flows to the research fund. This is a one-way ratchet — once governance activates, the founder share cannot be reinstated without a code upgrade.

## Revenue Sources

Every revenue source flows through the same `DistributeRevenue` function:

| Source | Module | What Generates It |
|--------|--------|-------------------|
| Block Production | `vesting_rewards` | Every block with PoT activity |
| Transaction Fees | `vesting_rewards` (via `RouteFees`) | All tx gas fees (uzrn) |
| Knowledge Queries | `billing` | Queries to the knowledge API |
| Tool Invocations | `toolbox` | Calls to registered tools |
| Tree Deliverables | `tree` | Project task completions |
| Dispute Resolution | `disputes` | Slash distributions |
| Home Creation | `home` | 10 ZRN per home created |
| Channel Operations | `channels` | Channel open fees |
| BVM Deployments | `bvm` | 5 ZRN per contract deployed |
| Swap Fees | `liquiditypool` | AMM swap fees |

### Fee Routing

Transaction fees get special treatment via `RouteFees()`:
- Research share (13%) is extracted from `fee_collector` → research fund
- Burn share (10%) is extracted and burned
- The remaining 77% (contributor + protocol) stays in `fee_collector` for Cosmos SDK's `x/distribution` to sweep to validators

This ensures that even standard tx fees contribute to research and deflation.

## Governance Adjustability

All split parameters are governance-adjustable via LIP proposals:
- A parameter-category LIP requires 1,000 ZRN stake
- Discussion: ~2 days, Voting: ~3 days
- 33.4% quorum, >50% support to pass

The revenue split is the single most powerful governance lever. Adjusting it changes the economic incentives for every participant on the chain.

## Consistency Guarantee

Multiple modules independently apply revenue splits (toolbox, tree, billing). The `RevenueSplit` message is defined in `x/common` to ensure all modules use the same split structure. The actual BPS values are read from each module's own params, allowing per-module overrides if governance chooses to differentiate (e.g., giving toolbox creators a higher contributor share).

Currently, all modules use the default 55/22/13/10 split. Divergence is possible but would require individual governance proposals per module.
