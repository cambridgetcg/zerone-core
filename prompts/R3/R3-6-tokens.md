# R3-6 — Tokens Module: Mint, Delegate Power, Wrap/Unwrap

## Goal

Port the token management module — minting authority, delegated voting
power, token wrapping (bridge incoming tokens to Zerone-native format),
and period-based emission schedules.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/tokens/` — full module
- `/Users/yuai/Desktop/legible_money/x/tokens/keeper/keeper_test.go` — 93 tests
- `/Users/yuai/Desktop/legible_money/docs/PARAMETERS.md` — tokens params (6, all disabled)

## Proto Files

### `proto/zerone/tokens/v1/types.proto`
```protobuf
message TokenConfig {
  string denom = 1;
  string mint_authority = 2;       // address authorized to mint
  string max_supply = 3;           // 0 = unlimited
  bool mintable = 4;
  bool burnable = 5;
  uint64 created_at_block = 6;
}

message DelegatedPower {
  string delegator = 1;
  string delegatee = 2;
  string amount = 3;               // voting power delegated
  uint64 created_at_block = 4;
}

message WrappedToken {
  string original_denom = 1;       // source chain denom
  string wrapped_denom = 2;        // zerone-native denom
  string total_wrapped = 3;
  string bridge_address = 4;       // bridge module account
}

message EmissionPeriod {
  uint64 start_block = 1;
  uint64 end_block = 2;
  string amount_per_block = 3;     // uzrn
  string recipient = 4;            // module account or address
  bool active = 5;
}
```

### `proto/zerone/tokens/v1/tx.proto`
- MsgMint (authority only)
- MsgBurn
- MsgDelegatePower
- MsgUndelegatePower
- MsgWrapToken
- MsgUnwrapToken
- MsgCreateEmissionPeriod (governance only)
- MsgCancelEmissionPeriod
- MsgUpdateParams
- ... (all from draft — 15 handlers total)

### `proto/zerone/tokens/v1/query.proto`
- QueryTokenConfig
- QueryDelegatedPower (by delegator or delegatee)
- QueryWrappedToken
- QueryEmissionPeriods
- QueryParams
- QueryTotalSupply (with minted/burned breakdown)

### `proto/zerone/tokens/v1/genesis.proto`
- Params (emission configs — currently all disabled/zeroed)
- GenesisState { params, token_configs, delegated_powers, wrapped_tokens, emission_periods }

## Key Implementation

### Mint with authority check

```go
func (k Keeper) Mint(ctx sdk.Context, authority, recipient string, amount sdk.Coins) error {
    config := k.GetTokenConfig(ctx, amount[0].Denom)
    if config.MintAuthority != authority {
        return types.ErrUnauthorizedMint
    }
    if config.MaxSupply > 0 {
        currentSupply := k.bankKeeper.GetSupply(ctx, amount[0].Denom)
        if currentSupply.Add(amount[0]).Amount.GT(maxSupply) {
            return types.ErrMaxSupplyExceeded
        }
    }
    return k.bankKeeper.MintCoins(ctx, types.ModuleName, amount)
}
```

### Delegated voting power

Separate from staking delegation — this is governance voting power
delegation (you can delegate your vote without moving your tokens):

```go
func (k Keeper) DelegatePower(ctx sdk.Context, delegator, delegatee string, amount sdk.Int) error
func (k Keeper) GetVotingPower(ctx sdk.Context, address string) sdk.Int {
    // Own tokens + received delegated power - delegated away power
}
```

### Token wrapping (bridge prep)

```go
func (k Keeper) WrapToken(ctx sdk.Context, depositor string, originalDenom string, amount sdk.Int) error {
    // 1. Lock original tokens in bridge module account
    // 2. Mint wrapped tokens to depositor
    // 3. Update WrappedToken state
}
```

### BeginBlocker — emission periods

```go
func (k Keeper) BeginBlocker(ctx sdk.Context) {
    // For each active emission period:
    // If current block in [start, end]: mint amount_per_block to recipient
}
```

## Tests

Port all 93 tests. Key areas:
- Mint authority enforcement
- Max supply cap
- Delegated power accounting (no double-counting)
- Wrap/unwrap round-trip conservation
- Emission period lifecycle
- Burn reduces supply correctly

## Verification

```bash
make proto-gen
go build ./...
go test ./x/tokens/... -count=1 -v
```

## Commit

```
feat(tokens): mint authority, delegated power, wrap/unwrap, emission periods
```

## Do NOT

- Allow minting without authority check
- Allow max supply to be exceeded
- Double-count delegated voting power
- Skip the wrap/unwrap conservation test (wrapped + unwrapped = constant)
