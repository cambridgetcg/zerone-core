# Zerone Tokenomics

> **Status:** Pre-testnet review (2026-02-23)  
> **Token:** ZRN (micro-denomination: uzrn, 1 ZRN = 1,000,000 uzrn)  
> **Chain:** Cosmos SDK v0.50, CometBFT consensus

## Overview

Zerone is a **Proof-of-Truth** (PoT) blockchain where all tokens are minted through verified knowledge contribution — not through proof-of-work or proof-of-stake inflation. There is no pre-mine, no ICO, and no token sale. All ZRN enters circulation as block rewards for producing blocks that contain verified truth claims.

## Documents in This Directory

| File | Description |
|------|-------------|
| [SUPPLY.md](SUPPLY.md) | Supply cap, emission schedule, decay curve, and long-term projections |
| [REVENUE-SPLIT.md](REVENUE-SPLIT.md) | The 4-way revenue split, protocol sub-split, and founder sunset |
| [VESTING.md](VESTING.md) | Truth-linked vesting: epistemic categories, release curves, clawback |
| [STAKING.md](STAKING.md) | Tiered validator system, staking economics, slashing |
| [GENESIS.md](GENESIS.md) | Genesis distribution, bootstrap accounts, and ceremony |
| [SINKS-AND-FLOWS.md](SINKS-AND-FLOWS.md) | Complete map of where ZRN is created, destroyed, and moves |
| [REVIEW.md](REVIEW.md) | Honest assessment: strengths, risks, open questions |

## Quick Numbers

| Metric | Value |
|--------|-------|
| **Max Supply** | 222,222,222 ZRN (hard cap, enforced in code) |
| **Genesis Circulating** | 15,500,000 ZRN (bootstrap accounts, no pre-mine) |
| **Initial Block Reward** | 10 ZRN/block |
| **Block Time** | ~2.521 seconds |
| **Epoch Length** | 100,000 blocks (~2.9 days) |
| **Decay Rate** | 0.85× per epoch |
| **Floor Reward** | 0.1 ZRN/block |
| **Burn Rate** | 10% of all revenue |
| **Revenue to Contributors** | 55% |
| **Revenue to Protocol** | 22% |
| **Revenue to Research Fund** | 13% |

## Design Philosophy

1. **Truth creates value, not computation.** Block rewards flow only when the chain produces verified knowledge. Empty blocks earn nothing.

2. **Deflationary by design.** 10% of all revenue is burned. The supply cap plus burning means the effective circulating supply will peak and decline over time.

3. **Knowledge has memory.** Rewards vest according to epistemic category — mathematical proofs vest slowly (because they should last forever), oracle feeds vest quickly (because they expire). If a claim is falsified, rewards are clawed back.

4. **Anti-capture as infrastructure.** HHI-based concentration monitoring, tiered validators with reputation requirements, and cross-stratum verification make knowledge monopolies structurally expensive to maintain.

5. **The chain can heal itself.** The autopoiesis module adjusts slash severity and economic parameters based on a System Stability Index (SSI), within governance-bounded rails. The alignment module monitors five health dimensions. The chain doesn't just enforce rules — it adapts them.
