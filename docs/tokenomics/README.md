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
Post-genesis native issuance shares one `MintWithCap` gate. Vesting-rewards
v2 retires transaction-bearing block rewards. Enabled source paths are
claiming-pot claims and survival-gated substrate-bridge rewards. Default-zero
knowledge probe issuance and default-disabled `x/tokens` emission periods are
additional governance-activatable controls.

## Documents in This Directory

| File | Description |
|------|-------------|
| [SUPPLY.md](SUPPLY.md) | Supply cap, source-capable issuance families, retired automatic mint, and evidence-bound projections |
| [REVENUE-SPLIT.md](REVENUE-SPLIT.md) | The 4-way revenue split, protocol sub-split, and founder retirement |
| [VESTING.md](VESTING.md) | Truth-linked vesting: epistemic categories, release curves, clawback |
| [STAKING.md](STAKING.md) | Tiered validator system, staking economics, slashing |
| [GENESIS.md](GENESIS.md) | Genesis distribution, bootstrap accounts, and ceremony |
| [SINKS-AND-FLOWS.md](SINKS-AND-FLOWS.md) | Complete map of where ZRN is created, destroyed, and moves |
| [GOVERNANCE-MIGRATION.md](GOVERNANCE-MIGRATION.md) | 4-phase research-fund model; the live genesis voter pair is unconfigured |
| [REVIEW.md](REVIEW.md) | Honest assessment: strengths, risks, open questions |
| [CONSTRUCTIVE-INTELLIGENCE-REWARDS.md](CONSTRUCTIVE-INTELLIGENCE-REWARDS.md) | Pre-consensus reward projection from the canonical capability tree onto artifact evidence, with power separation and fail-closed release gates |
| [../specs/constructive-intelligence-math-frontier-v0.md](../specs/constructive-intelligence-math-frontier-v0.md) | Zero-value formal-mathematics skill-tree extension and sponsor-quest template with ordinal-only KARMA and no privileged seats, shares, or activation authority |

## Quick Numbers

| Metric | Value |
|--------|-------|
| **Max Supply** | 222,222,222 ZRN (hard cap, enforced in code) |
| **Genesis Supply** | 13,555 ZRN (0.0061% of cap) — 11,333 validator collateral/gas + 2,222 transferable operator float, published |
| **Automatic Block Reward** | 0; retired in vesting_rewards v2 |
| **Block Time** | ~2.521 seconds |
| **Retired reward epoch metadata** | 100,000 blocks (~2.9 days) |
| **Retired decay metadata** | 0.994478× per epoch (historical 1-year half-life model) |
| **Floor Reward** | 0; retired compatibility field |
| **Caller-supplied revenue contributor share** | 55% |
| **Revenue to Protocol** | 22% |
| **Revenue to Development** | 19.67% (bug bounties, truth discovery, protocol dev) |
| **Revenue to Research Fund** | 3.33% |
| **General Burn Share** | 0%; rejected substrate-attestation bonds are the narrow burn exception |

## Design Philosophy

1. **Issuance requires more than proposer-controlled inclusion.** Consensus v2
   never mints merely because a block contains an ordinary transaction.
   Legacy decay, validator, and survived-challenge coupling fields remain
   observable but do not drive automatic issuance.

2. **Revenue is routed, not burned.** There is no general revenue burn share;
   rejected substrate-attestation bonds are the narrow punitive exception. The
   222M supply cap provides the primary scarcity boundary.

3. **Knowledge has memory.** Rewards vest according to epistemic category — mathematical proofs vest slowly (because they should last forever), oracle feeds vest quickly (because they expire). If a claim is falsified, rewards are clawed back.

4. **Anti-capture as infrastructure.** HHI-based concentration monitoring, tiered validators with reputation requirements, and cross-stratum verification make knowledge monopolies structurally expensive to maintain.

5. **The chain can observe itself.** The alignment module records five health
   dimensions and advisory corrections. The former autopoiesis regulator was
   retired; current corrections do not automatically rewrite slash or
   economic parameters.
