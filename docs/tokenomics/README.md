# Zerone Tokenomics

> **Status:** Source-reviewed for the live custodial launch (2026-07-29)
> **Token:** ZRN (micro-denomination: uzrn, 1 ZRN = 1,000,000 uzrn)  
> **Chain:** Cosmos SDK v0.50, CometBFT consensus

## Overview

Zerone is a **Proof-of-Truth** (PoT) research blockchain. There was no ICO or
investor sale. The live custodial genesis did create real
operator-controlled scaffolding: 11,333 ZRN of validator collateral
(11,111 bonded + 222 spendable gas) and a transferable 2,222 ZRN operator
float, 13,555 ZRN total (0.0061% of cap), with every address published.
Post-genesis native issuance shares one `MintWithCap` gate. The published
configuration has transaction-bearing block rewards, claiming-pot claims, and
substrate-bridge rewards; an ordinary user transaction qualifies a block for
the first lane. Default-zero knowledge probe issuance and default-disabled
`x/tokens` emission periods are additional governance-activatable controls.

## Documents in This Directory

| File | Description |
|------|-------------|
| [SUPPLY.md](SUPPLY.md) | Supply cap, emission schedule, decay curve, and long-term projections |
| [REVENUE-SPLIT.md](REVENUE-SPLIT.md) | The 4-way revenue split, protocol sub-split, and founder sunset |
| [VESTING.md](VESTING.md) | Truth-linked vesting: epistemic categories, release curves, clawback |
| [STAKING.md](STAKING.md) | Tiered validator system, staking economics, slashing |
| [GENESIS.md](GENESIS.md) | Genesis distribution, bootstrap accounts, and ceremony |
| [SINKS-AND-FLOWS.md](SINKS-AND-FLOWS.md) | Complete map of where ZRN is created, destroyed, and moves |
| [GOVERNANCE-MIGRATION.md](GOVERNANCE-MIGRATION.md) | 4-phase research-fund model; the live genesis voter pair is unconfigured |
| [REVIEW.md](REVIEW.md) | Honest assessment: strengths, risks, open questions |
| [CONSTRUCTIVE-INTELLIGENCE-REWARDS.md](CONSTRUCTIVE-INTELLIGENCE-REWARDS.md) | Pre-consensus reward projection from the canonical capability tree onto artifact evidence, with power separation and fail-closed release gates |

## Quick Numbers

| Metric | Value |
|--------|-------|
| **Max Supply** | 222,222,222 ZRN (hard cap, enforced in code) |
| **Genesis Supply** | 13,555 ZRN (0.0061% of cap) — 11,333 validator collateral/gas + 2,222 transferable operator float, published |
| **Initial Block Reward** | 10 ZRN/block |
| **Block Time** | ~2.521 seconds |
| **Epoch Length** | 100,000 blocks (~2.9 days) |
| **Decay Rate** | 0.994478× per epoch (1-year half-life) |
| **Floor Reward** | 0.1 ZRN/block |
| **Block reward to proposer / configured withdraw address** | 55% |
| **Revenue to Protocol** | 22% |
| **Revenue to Development** | 19.67% (bug bounties, truth discovery, protocol dev) |
| **Revenue to Research Fund** | 3.33% |
| **General Burn Share** | 0%; rejected substrate-attestation bonds are the narrow burn exception |

## Design Philosophy

1. **Truth coupling is partial and measurable.** Empty blocks earn nothing, but
   any non-injection user transaction—including an ordinary transfer—makes a
   block reward-eligible. Validator count and the survived-challenge rate scale
   the reward; transaction eligibility alone is not proof of verified truth.

2. **Revenue is routed, not burned.** There is no general revenue burn share;
   rejected substrate-attestation bonds are the narrow punitive exception. The
   222M supply cap provides the primary scarcity boundary.

3. **Knowledge has memory.** Rewards vest according to epistemic category — mathematical proofs vest slowly (because they should last forever), oracle feeds vest quickly (because they expire). If a claim is falsified, rewards are clawed back.

4. **Anti-capture as infrastructure.** HHI-based concentration monitoring, tiered validators with reputation requirements, and cross-stratum verification make knowledge monopolies structurally expensive to maintain.

5. **The chain can observe itself.** The alignment module records five health
   dimensions and advisory corrections. The former autopoiesis regulator was
   retired; current corrections do not automatically rewrite slash or
   economic parameters.
