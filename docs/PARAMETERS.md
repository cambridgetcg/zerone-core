# Zerone Parameter Orientation

> **Scope:** This is a deliberately small, source-backed orientation—not a
> complete parameter catalogue and not a live-network snapshot. Defaults,
> genesis values, and current on-chain values are three different things.
> Operational decisions must query an authorised network at a bound height and
> use a release-matched binary.

All custom-module BPS values use a 1,000,000 scale unless a field explicitly
says otherwise. Token amounts are integer strings in `uzrn`
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
publication does not activate the `consolidation-safety-v1` changes.

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

### Block rewards and fees

| Field | Source default |
|---|---:|
| Block reward | 10,000,000 uzrn before decay/scaling |
| Empty-block reward rate | 0 |
| Validators for full reward | 22 |
| Consensus fee floor | 1 uzrn per declared gas unit |
| Revenue split | 55% block proposer (`contributor_bps` wire name) / 22% protocol / 19.67% development / 3.33% research |
| Founder sub-share | Up to 7% of research; inactive while address is empty |

`founder_share_bps` is governance-mutable within its 7% cap.
`founder_address` becomes immutable once set.

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
