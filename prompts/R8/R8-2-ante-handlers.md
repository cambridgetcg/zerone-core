# R8-2 — Custom Ante Handlers

## Goal

Port all 5 custom Zerone ante decorators from the draft, plus the gas cost table and fee
router. Replace the current minimal ante handler chain with the full production chain.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/app/ante.go` — full ante chain (509 LOC)
- `/Users/yuai/Desktop/legible_money/app/ante_legible.go` — 5 custom decorators (559 LOC)
- `/Users/yuai/Desktop/legible_money/app/ante_test.go` — unit tests (439 LOC)
- `/Users/yuai/Desktop/legible_money/app/ante_integration_test.go` — integration tests (1604 LOC)
- Rename all `legible` → `zerone`, `ulgm` → `uzrn`, `LGM` → `ZRN`

## Current State

`app/ante.go` has only SDK standard decorators + IBC relay decorator. No custom Zerone
decorators. Comments list what's missing.

## Custom Decorators to Port

### 1. BootstrapGasFreeDecorator
- Waives gas and fees for essential PoT transactions during bootstrap period (~first 14 days)
- Checks if block height < BootstrapEndHeight
- Free messages: MsgSubmitClaim, MsgSubmitCommitment, MsgSubmitReveal, MsgAddFact
- Sets gas meter to infinite for qualifying txs

### 2. EmergencyHaltDecorator
- Blocks ALL non-emergency transactions when chain is halted
- Reads halt state from EmergencyKeeper.IsHalted()
- Only allows: MsgProposeResume, MsgVoteResume, MsgProposeRevert, MsgVoteRevert
- Returns clear error: "chain is in emergency halt state"

### 3. LGMGasDecorator (→ ZRNGasDecorator)
- Validates gas limit against per-message type gas cost table
- Enforces BlockGasLimit and TxGasLimit
- Saturating addition for multi-message gas calculation (P1-2 fix)
- Enforces minimum gas price
- Port the full `msgTypeURLToGas` map — update ALL type URLs from `legible.*` to `zerone.*`

### 4. FeeRouterDecorator
- Logs fee routing split (7% research fund, 93% validators)
- Actual routing done in vesting_rewards BeginBlocker
- Validates fee denomination is uzrn

### 5. ZeroneAccountDecorator (was LegibleAccountDecorator)
- Checks account frozen status via ZeroneAuthKeeper
- Updates LastActiveBlock on successful tx
- Uses `getSignerAddresses()` helper (extracts from tx signature data, not per-message)
- **Critical P0-1 fix baked in:** extracts signers from tx signature data, NOT from
  `msg.GetSigners()` which fails for SDK proto-generated types

### 6. ZeroneCapabilityDecorator (was LegibleCapabilityDecorator)
- Enforces session key capability restrictions
- Checks if tx signer is using a session key
- Validates session key has permission for each message type in the tx
- Uses `getTxSigners()` helper for pubkey matching
- **Capability presets:** "read_only", "transfer_only", "knowledge_full", "governance_full"

### 7. ZeroneDIDDecorator (was LegibleDIDDecorator)
- Resolves DID from tx memo field
- Annotates context with resolved DID for downstream handlers
- Optional — only applies when memo contains a DID

## Helper Functions

Port from `ante_legible.go`:
- `getSignerAddresses(tx)` — extracts signer addresses from tx signature data
- `getTxSigners(tx)` — extracts signer addresses + pubkey hex from signature data

## Gas Cost Table

Port `TransactionGasCosts` map and `msgTypeURLToGas` map from draft.
**Critical:** update ALL proto type URLs from `/legible.*` to `/zerone.*`.

Constants to port:
- `BlockGasLimit` — max gas per block
- `TxGasLimit` — max gas per transaction
- `MinGasLimit` — minimum gas for any tx
- `MinGasPrice` — minimum gas price in uzrn
- `BondDenom` — "uzrn"
- `ResearchContributionBPS` — 70000 (7% on 1M scale)
- `ValidatorFeeBPS` — 930000 (93% on 1M scale)

## Full Ante Chain Order

```go
sdk.ChainAnteDecorators(
    ibcante.NewRedundantRelayDecorator(app.IBCKeeper),
    ante.NewSetUpContextDecorator(),
    NewBootstrapGasFreeDecorator(),
    NewEmergencyHaltDecorator(app.EmergencyKeeper),
    NewZRNGasDecorator(),
    NewFeeRouterDecorator(app.BankKeeper),
    ante.NewExtensionOptionsDecorator(nil),
    ante.NewValidateBasicDecorator(),
    ante.NewTxTimeoutHeightDecorator(),
    ante.NewValidateMemoDecorator(app.AccountKeeper),
    ante.NewConsumeGasForTxSizeDecorator(app.AccountKeeper),
    ante.NewDeductFeeDecorator(app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, nil),
    ante.NewSetPubKeyDecorator(app.AccountKeeper),
    ante.NewValidateSigCountDecorator(app.AccountKeeper),
    ante.NewSigGasConsumeDecorator(app.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
    ante.NewSigVerificationDecorator(app.AccountKeeper, app.txConfig.SignModeHandler()),
    ante.NewIncrementSequenceDecorator(app.AccountKeeper),
    NewSybilFundingDecorator(&app.ZeroneGovKeeper),
    NewZeroneDIDDecorator(app.ZeroneAuthKeeper),
    NewZeroneAccountDecorator(app.ZeroneAuthKeeper),
    NewZeroneCapabilityDecorator(app.ZeroneAuthKeeper, app.AccountKeeper),
)
```

## File Structure

```
app/
├── ante.go              (update: full chain, constants, gas cost table)
├── ante_zerone.go       (new: 5 custom decorators + helpers)
├── ante_test.go         (new: port unit tests)
└── ante_integration_test.go  (new: port integration tests — may defer to R9)
```

## Tests

Port from draft `ante_test.go` (439 LOC):
1. EmergencyHaltDecorator blocks non-emergency txs
2. EmergencyHaltDecorator allows emergency txs
3. ZeroneAccountDecorator blocks frozen accounts
4. ZeroneAccountDecorator updates LastActiveBlock
5. ZeroneCapabilityDecorator enforces session key restrictions
6. BootstrapGasFreeDecorator waives gas during bootstrap
7. ZRNGasDecorator enforces per-message gas minimums
8. ZRNGasDecorator saturating addition prevents overflow
9. FeeRouterDecorator validates fee denomination

## Constraints

- Port ALL audit fixes (P0-1 signer extraction, P1-2 gas overflow) — these are baked in, not patched
- Use `getSignerAddresses()` everywhere, never `msg.GetSigners()`
- Gas table must cover ALL message types from ALL 32 modules
- Update every type URL from `legible` to `zerone`
- The `SybilFundingDecorator` uses `ZeroneGovKeeper` — ensure it's compatible
