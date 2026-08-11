# Zerone Tokenomics

> **Status:** source target for atomic
> `consolidation-safety-v1`; not a claim that H1 is applied on a running chain.
> **Token:** ZRN (`uzrn`, 1 ZRN = 1,000,000 uzrn)
> **Chain:** Cosmos SDK v0.50, CometBFT consensus

## Overview

Zerone is a Proof-of-Truth research blockchain with no ICO or investor sale.
The live custodial genesis did create real operator-controlled scaffolding:
11,333 ZRN of validator collateral/gas and a transferable 2,222 ZRN operations
float, 13,555 ZRN total. Every address is published.

Atomic H1 introduces an enforceable economic-neutrality boundary:

- liquiditypool v5 sends no protocol share out of a pool; configured swap fees
  stay in reserves for transferable LP shares;
- vesting_rewards v2 has no founder auto-split; and
- raw transaction presence no longer creates a proposer mint.

Actual transaction fees continue to compensate execution/consensus and route
to development, research, and normal Cosmos distribution. Other cap-gated
issuance pathways retain their own evidence and governance rules. See
[Economic neutrality](ECONOMIC-NEUTRALITY.md).

## Documents

| File | Description |
|---|---|
| [ECONOMIC-NEUTRALITY.md](ECONOMIC-NEUTRALITY.md) | H1 rule: value follows capital, paid execution, or independently witnessed work—not identity/control |
| [SUPPLY.md](SUPPLY.md) | Hard cap and remaining issuance surfaces after automatic block-reward retirement |
| [REVENUE-SPLIT.md](REVENUE-SPLIT.md) | Real transaction-fee routing, LP compensation, and retired compatibility fields |
| [VESTING.md](VESTING.md) | Truth-linked vesting schedules and clawback |
| [STAKING.md](STAKING.md) | Custom tiers, Cosmos staking, fees, and validator economics |
| [GENESIS.md](GENESIS.md) | Immutable launch distribution and prospective H1 boundary |
| [SINKS-AND-FLOWS.md](SINKS-AND-FLOWS.md) | Source-backed map of ZRN creation, destruction, locks, and movement |
| [GOVERNANCE-MIGRATION.md](GOVERNANCE-MIGRATION.md) | Research-fund committee model and implementation gaps |
| [LIQUIDITY-TRANSPARENCY.md](LIQUIDITY-TRANSPARENCY.md) | External Osmosis testnet position, separate from the native module |
| [REVIEW.md](REVIEW.md) | Frozen historical critique; not current accounting |
| [CONSTRUCTIVE-INTELLIGENCE-REWARDS.md](CONSTRUCTIVE-INTELLIGENCE-REWARDS.md) | Pre-consensus reward projection from the canonical capability tree onto artifact evidence, including a conserved branch-flow role envelope, power separation, and fail-closed release gates |
| [../specs/constructive-intelligence-branch-flow-v1.md](../specs/constructive-intelligence-branch-flow-v1.md) | Accepted exact shadow architecture for one conserved 60% direct / 10% upstream / 30% downstream envelope with absolute geometric depth and economic effect `NONE` |
| [../specs/constructive-intelligence-math-frontier-v0.md](../specs/constructive-intelligence-math-frontier-v0.md) | Zero-value formal-mathematics skill-tree extension and sponsor-quest template with ordinal-only KARMA and no privileged seats, shares, or activation authority |

## Quick boundaries

| Metric | Source/H1 value |
|---|---:|
| Maximum supply | 222,222,222 ZRN |
| Live genesis supply | 13,555 ZRN (0.0061% of cap) |
| Transaction-presence block reward | 0 after H1 |
| Founder protocol share | 0 after H1 |
| Liquidity protocol skim | 0 after H1 |
| Default pool swap fee | 0.3%, retained by LP-backed reserves |
| Real `uzrn` transaction-fee routing | 19.67% development / 3.33% research / ~77% normal distribution |
| General burn share | 0%; rejected substrate-attestation bonds are a narrow exception |
| Branch-flow economic effect | `NONE`; shadow only, with all activation gates closed |

## Design philosophy

1. **No identity rent.** A founder, creator label, or safety authority does not
   receive an automatic runtime percentage.
2. **Capital receipts remain property.** LP shares represent funded reserves;
   fee retention compensates capital/risk and any bearer may exit pro rata.
3. **Paid execution remains paid.** Retiring transaction-presence issuance does
   not retire real transaction fees or normal validator distribution.
4. **Work rewards require evidence.** Supply-cap compliance is necessary but
   not sufficient; each mint lane must bind a successful, non-self-attested
   trigger.
5. **History is not rewritten.** Genesis and pre-H1 events remain the truthful
   record; migration changes prospective runtime state only.
6. **Attribution conserves its envelope.** Direct, upstream, downstream, and
   terminal amounts divide one prospectively funded role envelope. They do not
   create recursive issuance, ownership of descendants, or a separate
   breakthrough prize.
