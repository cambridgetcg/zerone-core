# Zerone Parameter Orientation

> **Scope:** This is a deliberately small, source-backed orientation—not a
> complete parameter catalogue and not a live-network snapshot. Defaults,
> genesis values, and current on-chain values are three different things.
> Operational decisions must query an authorised network at a bound height and
> use a release-matched binary.

Custom-module percentage fields historically named `_bps` use a 1,000,000
parts-per-million scale unless a field explicitly says otherwise. They are not
conventional 10,000-denominator basis points. New documentation and proposal
reviews call the unit PPM while preserving legacy protobuf field names where
wire compatibility requires it. Token amounts are integer strings in `uzrn`
(1 ZRN = 1,000,000 uzrn).

## Sources of truth

Use these in order:

1. **Current on-chain state:** each module's `Params` query, bound to chain ID,
   height, and release.
2. **A specific genesis:** the module's `params` object in that exact,
   hash-verified genesis artifact.
3. **New-chain defaults:** `DefaultParams`/`DefaultGenesis` in
   `x/<module>/types`. These do not retroactively alter a running chain.
4. **Wire schema:** `proto/zerone/<module>/v1/genesis.proto`; comments should
   match source defaults but the Go constructor remains authoritative.

Examples for a release-matched CLI:

```bash
zeroned q knowledge params --node <authorised-rpc> --height <height>
zeroned q zerone_staking params --node <authorised-rpc> --height <height>
zeroned q zerone_gov params --node <authorised-rpc> --height <height>
zeroned q vesting_rewards params --node <authorised-rpc> --height <height>
```

The generated Swagger file exposes the equivalent REST queries. A response
from `zerone-1` describes that deployed binary and its on-chain state; source
publication activates none of the ordered H1 `consolidation-safety-v1`, H2
`founder-renunciation-v1`, or H3 `sdk-0.53-ibc-10` changes.

## High-impact source defaults

This short table exists to make common integration mistakes obvious. It is not
an exhaustive substitute for the constructors above.

### Knowledge

| Field | Source default |
|---|---:|
| `min_verifiers` / `max_verifiers` | 3 / 22 |
| `commit_phase_blocks` | 200 |
| `reveal_phase_blocks` | 200 |
| `aggregation_phase_blocks` | 50 |
| `min_claim_text_length` / `max_claim_text_length` | 20 / 1,000 |
| `min_review_fee` | 100,000 uzrn (non-refundable) |
| `wrong_verification_slash_bps` | 50,000 (5%) |
| `missed_reveal_slash_bps` | 100,000 (10%) |
| `equivocation_slash_bps` | 200,000 (20%) |
| `invalid_claim_slash_bps` | 0 (deprecated and unused) |
| `challenge_duration_blocks` | 34,272 |
| `min_challenge_stake` | 11,000,000 uzrn |

There is no `min_claim_stake` field. Removed citation-gaming and FARM fields
are reserved in protobuf and must not be presented as active controls.

### Staking

| Field | Source default |
|---|---:|
| `unbonding_period` | 268,560 blocks |
| `max_validators` | 100 |
| `min_self_delegation` | 111,000 uzrn |
| `min_stake_for_verification` | 111,000 uzrn |
| `virtual_stake` | 11,000,000 uzrn |

### Automatic rewards and real fees

| Field | Source default |
|---|---:|
| Transaction-presence block reward | 0; permanently retired in vesting_rewards v2 |
| Floor reward | 0; retired compatibility field |
| Empty-block reward rate | 0; retired compatibility field |
| Consensus fee floor | 1 uzrn per declared gas unit |
| Real `uzrn` transaction-fee routing | 19.67% development / 3.33% research / approximately 77% normal Cosmos distribution |
| Founder sub-share | 0; permanently retired in vesting_rewards v2 |

The legacy `block_reward`, `floor_reward`, `founder_share_bps`, and
`founder_address` wire fields remain for compatibility but must be zero/empty.
Ordinary governance cannot reactivate them. `RouteFees` still routes actual
fees collected from transactions; retirement concerns automatic issuance, not
fee compensation.

### Liquiditypool v5

These are source defaults for liquiditypool consensus v5, not a claim about a
running chain:

| Field | Source default / rule |
|---|---:|
| `default_swap_fee_bps` (legacy wire name) | 3,000 PPM (0.3%) |
| Maximum per-pool swap fee | 100,000 PPM (10%) |
| `protocol_fee_bps` (legacy wire name) | 0; retired and rejected if nonzero |
| `max_pools` | 16 open pools; valid range 1–64, zero/unlimited is invalid |
| `min_initial_liquidity` | 10,000,000,000 uzrn (10,000 ZRN on the ZRN side) |
| `min_reserve` | 1 base unit |
| `billing_quote_denoms` | empty; oracle disabled fail-closed |
| `allowed_pool_denoms` | empty; unconsumed one-shot counter-denom creation grants, consumed on successful creation |
| `pool_creators` | empty; no account may fund/create a pool |

In consensus v5, `twap_window_blocks` may be increased within its bound but
cannot be decreased by an ordinary Params update. Shrinking retained history
requires a future bounded cleanup migration; rejecting an immediate shrink
prevents the next BeginBlock from attempting thousands of synchronous deletes.

Pool status is governance-controlled: `ACTIVE`, `SWAPS_PAUSED`, `EXIT_ONLY`,
then immutable `CLOSED` after the final LP exit. Oracle selection additionally
requires an allowlisted quote denom and an `ACTIVE` pool. See
[LIQUIDITYPOOL-SAFETY-V2.md](LIQUIDITYPOOL-SAFETY-V2.md) before constructing a
proposal. The configured per-pool swap fee remains in reserves and therefore
accrues pro rata to transferable LP shares; neither governance nor a named
founder receives a skim. Liquiditypool denom admission does not bypass
`x/bank`: both `uzrn`
and the counter-denom must also be send-enabled before initial liquidity can
enter module custody. `GetZRNPrice` is a raw base-unit ratio scaled by
1,000,000; the source defaults intentionally keep its allowlist empty.
Before enabling a quote denom, governance must bind its base-unit exponent,
consumer conversion, and a decimal-normalized per-denom oracle TVL floor; the
global one-base-unit `min_reserve` is not sufficient for oracle admission.

### Governance

| Field | Source default |
|---|---:|
| Discussion period | 68,544 blocks |
| Voting period | 102,816 blocks |
| Quorum | 334,000 BPS (33.4%) |
| Support threshold | 500,000 BPS (50%) |
| Minimum LIP stake | 1,000,000 uzrn |

Category-specific proposal stakes and review periods are defined in
`x/gov/types/genesis.go`.

## Gas schedule

The consensus fee floor and message gas schedule are code, not governance
module parameters. Their source is `app/gas.go`. The fee for a transaction is
based on declared gas and the applicable gas price; a node-local
`minimum-gas-prices` value lower than the consensus ante floor does not make a
lower-fee transaction valid.

## Module inventory

Parameter-bearing modules expose their types and defaults under `x/*/types`.
Pure consumers such as `training_provenance` and `trust_score` have no
independent parameter set. For integration work, enumerate the protobuf
descriptors or generated TypeScript registry instead of copying a hand-written
table; CI rejects protobuf, Swagger, and SDK generation drift.
