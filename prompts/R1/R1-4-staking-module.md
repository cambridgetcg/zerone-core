# R1-4 — Staking Module: Full Port

## Goal

Port the Zerone 4-tier staking system from the draft. This is not the standard
Cosmos SDK staking — it's a custom module with Apprentice → Verified → Scholar
→ Guardian tiers, reputation tracking, and PoT-aware delegation.

## Dependencies

- R1-2 must be complete (shared protos, app skeleton)
- R1-3 (auth) is NOT required — staking uses interfaces for auth

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference (Draft)

All source files in `/Users/yuai/Desktop/legible_money/x/staking/`:
- `types/types.go` — Validator, Delegation, TierConfig, Params (14 params + 44 tier configs)
- `types/genesis.go` — GenesisState, DefaultGenesis
- `types/keys.go` — KV prefixes
- `types/expected_keepers.go` — AccountKeeper, BankKeeper, LegibleAuthKeeper, AutopoiesisKeeper
- `keeper/keeper.go` — Keeper with all state CRUD
- `keeper/msg_server.go` — 6 handlers (in keeper directly, not msg_server pattern)
- `keeper/keeper_test.go` — 46 tests
- `client/cli/` — 6 TX + 7 query commands
- `module.go` — AppModule

Also reference:
- `/Users/yuai/Desktop/legible_money/docs/PARAMETERS.md` — staking params section
- `/Users/yuai/Desktop/legible_money/reports/audits/STAKING-POT.md` — 3 P0, 2 P1, 2 P2 findings
- `/Users/yuai/Desktop/legible_money/reports/batch-18/` — R2-6 tier lifecycle tests

### Critical security fixes to bake in (from audits):

1. **Guardian tier was unreachable** — contested counter never incremented (P0, fixed in draft R1)
2. **Verified tier MinStake not enforced** — fixed in draft R2-6
3. **Reputation underflow** — could go negative (P0, fixed in draft R1)
4. **DisbursementQuorumBps not validated** in params (P2, fix now)

## Proto Definitions

Create `proto/zerone/staking/v1/`:

### `types.proto`
```protobuf
// Validator represents a Zerone validator with tier and reputation.
message Validator {
  string operator_address = 1;
  string consensus_pubkey = 2;
  string did = 3;
  string moniker = 4;
  uint32 tier = 5;                    // 0=Apprentice, 1=Verified, 2=Scholar, 3=Guardian
  string self_delegation = 6;         // uzrn amount (string for big int)
  string total_delegation = 7;
  uint64 reputation_score = 8;        // 0-1,000,000 BPS
  uint64 correct_verifications = 9;
  uint64 total_verifications = 10;
  uint64 contested_count = 11;        // must increment! (Guardian fix)
  uint64 joined_at_block = 12;
  bool jailed = 13;
  string jail_reason = 14;
  uint64 unjail_after_block = 15;
}

// Delegation records a delegation from delegator to validator.
message Delegation {
  string delegator_address = 1;
  string validator_address = 2;
  string amount = 3;                  // uzrn
  uint64 created_at_block = 4;
}

// TierConfig defines requirements and rewards for each validator tier.
message TierConfig {
  uint32 tier = 1;
  string name = 2;                    // "Apprentice", "Verified", "Scholar", "Guardian"
  string min_self_delegation = 3;     // uzrn (MUST be enforced — audit fix)
  uint64 min_reputation = 4;          // BPS
  uint64 min_verifications = 5;
  uint64 reward_multiplier_bps = 6;   // 1,000,000 = 1.0x
  uint64 max_delegators = 7;
  uint64 verification_weight_bps = 8; // weight in PoT rounds
}
```

### `tx.proto`
Messages:
- MsgRegisterValidator (operator, consensus_pubkey, did, moniker, self_delegation)
- MsgDelegate (delegator, validator, amount)
- MsgUndelegate (delegator, validator, amount)
- MsgRedelegate (delegator, src_validator, dst_validator, amount)
- MsgUnjail (operator)
- MsgUpdateParams (authority, params)

### `query.proto`
- QueryValidator (address) → Validator
- QueryValidators (tier filter, pagination) → []Validator
- QueryDelegation (delegator, validator) → Delegation
- QueryDelegatorDelegations (delegator) → []Delegation
- QueryValidatorDelegations (validator) → []Delegation
- QueryParams → Params
- QueryTierConfig (tier) → TierConfig

### `genesis.proto`
- Params (14+ fields from draft, all on 1,000,000 BPS scale)
- GenesisState { params, validators, delegations, tier_configs }

## Key Implementation Notes

### Tier progression logic

```
Apprentice (0) → Verified (1):
  - min_self_delegation met
  - min_verifications met
  - reputation >= min_reputation

Verified (1) → Scholar (2):
  - higher min_self_delegation
  - more verifications
  - higher reputation

Scholar (2) → Guardian (3):
  - highest min_self_delegation (100,000 ZRN in draft)
  - contested_count > 0 (MUST CHECK — was the P0 bug)
  - highest reputation
```

### Tier advancement in BeginBlocker

Each block, check if any validator qualifies for tier advancement.
This was in the draft — port the logic but ensure the Guardian
contested_count check is present.

### Slashing

Validators can be slashed for:
- Wrong verification votes (knowledge module calls staking)
- Missed reveals
- Equivocation
- Downtime

Slash amounts from B22-3 audit fixes:
- wrong_verification: 5% (50,000 bps)
- missed_reveal: 10% (100,000 bps)
- equivocation: 20% (200,000 bps)

### AutopoiesisKeeper dependency

The draft's staking has a circular dependency with autopoiesis (staking uses
autopoiesis multiplier, autopoiesis depends on staking state). Break this
with a post-init setter:

```go
func (k *Keeper) SetAutopoiesisKeeper(ak types.AutopoiesisKeeper) {
    k.autopoiesisKeeper = ak
}
```

For now (R1), autopoiesis doesn't exist yet. The setter accepts nil
and staking works without it (multiplier defaults to 1.0x).

### Migrator stub

```go
type Migrator struct { keeper Keeper }
func NewMigrator(keeper Keeper) Migrator { return Migrator{keeper: keeper} }
```

## Tests

Port all 46 tests from draft. Add:

| New Test | Validates |
|----------|-----------|
| `TestGuardianTier_RequiresContestedCount` | Guardian unreachable bug is fixed |
| `TestVerifiedTier_EnforcesMinStake` | MinStake actually checked |
| `TestReputation_NoUnderflow` | Reputation floor at 0 |
| `TestParams_DisbursementQuorumValidation` | DisbursementQuorumBps validated |
| `TestParams_AllBPSScale` | Every BPS param on 1M scale |

## Wire into app.go

- Store key, keeper, ModuleManager
- BeginBlocker: tier advancement checks
- EndBlocker: unbonding maturation
- InitGenesis / ExportGenesis

## Verification

```bash
make proto-gen
go build ./...
go vet ./...
go test ./x/staking/... -count=1 -v
```

## Commit

```
feat(staking): port 4-tier staking — validators, delegation, reputation, tier progression
```

## Do NOT

- Use Cosmos SDK's standard staking module (Zerone has its own)
- Skip the Guardian contested_count fix
- Use 10,000 BPS scale anywhere (1,000,000 only)
- Leave MinStake enforcement as optional
- Forget the Migrator stub
