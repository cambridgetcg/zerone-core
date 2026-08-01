# Staking and Validator Economics

> **Implementation status (2026-07-29):** Zerone has two distinct staking
> surfaces. Cosmos SDK `x/staking` owns the CometBFT bonded validator set,
> proposer resolution, and ordinary distribution. Custom `x/zerone_staking`
> records PoT tiers, custom delegations, reputation, and eligibility consumed
> by knowledge/governance modules. Do not assume a custom tier field changes
> CometBFT rewards unless a production call path applies it.

## Custom PoT tiers

Source defaults:

| Tier | Min Stake | Min Reputation | Min Verifications | Min Accuracy |
|---|---:|---:|---:|---:|
| Apprentice | 0.111 ZRN | 0% | 0 | 0% |
| Verified | 1.11 ZRN | 77% | 22 | 77% |
| Scholar | 1,111 ZRN | 50% | 11 | 50% |
| Guardian | 11,111 ZRN | 77% | 333 | 77% |

Guardian configuration also requires 33 contested verifications and no active
slashes. The custom registry counts active Scholar/Guardian entries as block
producers for its own metrics, but the actual consensus validator set remains
Cosmos `x/staking`.

Tier configs contain reward, selection-weight, and slash multipliers:

| Tier | Configured reward | Configured selection | Configured slash |
|---|---:|---:|---:|
| Apprentice | 0.1× | 0.1× | 1.5× |
| Verified | 0.5× | 0.5× | 1.2× |
| Scholar | 1.0× | 1.0× | 1.0× |
| Guardian | 2.0× | 1.5× | 1.0× |

These are **not current block-payout promises**. `CalculateTierReward` and
`CalculateTierSlash` are helpers without production callers, and
`selection_weight_bps` is not applied in a production selection path. Current
low-tier selection uses virtual stake; higher tiers use real custom stake.
Integrations should query the concrete result they need rather than price a
tier multiplier as realised yield.

## Block rewards

`vesting_rewards` reads the active bonded-validator count from Cosmos
`x/staking`. On an eligible non-empty block it:

1. applies decay, bonded-validator scaling, and knowledge-survival coupling;
2. mints only within the 222,222,222 ZRN cap; and
3. routes 55% of the minted amount to the block proposer, 22% to protocol
   pools/reserves, 19.67% to development, and 3.33% to research.

It does not apply the custom Guardian 2× field. Empty blocks mint 0 under
defaults. Because issuance depends on activity, validator count, decay,
knowledge outcomes, and the cap, this document does not promise an hourly or
annual yield.

## Knowledge verification rewards

The configured `verification_reward` participates in verdict classification,
but the actual bank payout is drawn from 55% of the claim's non-refundable
review fee and shared among rewarded verifiers. Independence modulation may
withhold part of a verifier's share and route it to development. A flat
“3 ZRN per correct vote” is therefore not an accurate payout promise.

## Fees

For accumulated `uzrn` transaction fees, `RouteFees` sends 19.67% to
development and 3.33% in full to research through the canonical depositor.
Approximately 77% remains in `fee_collector` for normal Cosmos distribution. See
[REVENUE-SPLIT.md](REVENUE-SPLIT.md).

## Custom staking defaults

| Parameter | Value |
|---|---:|
| Unbonding period | 268,560 blocks |
| Max custom validators | 100 |
| Min self-delegation | 0.111 ZRN |
| Min custom verification stake | 0.111 ZRN |
| Virtual selection stake for Apprentice/Verified | 11 ZRN |
| Redelegation cooldown | 1,111 blocks |

Custom delegators share the custom module's slashing exposure and unbond
through its stored unbonding entries. These records must not be conflated with
Cosmos `x/staking` delegation without checking the exact message and query
namespace.

## Slashing

Active custom verifier slash defaults are:

| Offense | Base rate |
|---|---:|
| Wrong verification | 5% |
| Missed reveal | 10% |
| Equivocation | 20% |
| Invalid claim | No stake slash; review fee is non-refundable and the old field is deprecated at 0 |

The custom staking defaults allow two slashes per epoch and deactivate after
three cumulative slashes under their configured rules, with 10% escalation and
a 34,272-block decay period. Actual slash execution, tier multiplication, and
Cosmos consensus slashing are separate code paths.

## Domain qualification

`x/qualification` adds domain-specific records and decay rules on top of custom
validator status. Query current qualification parameters and records before
assuming a verifier is eligible for a domain; source defaults do not establish
live-network membership.
