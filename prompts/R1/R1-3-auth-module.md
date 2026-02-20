# R1-3 — Auth Module: Full Port

## Goal

Port the Zerone auth module (custom account system with DIDs, session keys,
account recovery, and freeze/unfreeze) from the draft into the clean
proto-first codebase.

## Dependencies

- R1-2 must be complete (shared protos, app skeleton)

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference (Draft)

All source files in `/Users/yuai/Desktop/legible_money/x/auth/`:
- `types/types.go` — Account, SessionKey, RecoveryConfig, Params, all message types
- `types/genesis.go` — GenesisState, DefaultGenesis, Validate
- `types/keys.go` — KV store prefixes
- `types/expected_keepers.go` — CosmosAccountKeeper, BankKeeper interfaces
- `types/codec.go` — RegisterInterfaces, RegisterLegacyAminoCodec
- `keeper/keeper.go` — Keeper struct, constructor, all state CRUD
- `keeper/msg_server.go` — 11 handler implementations
- `keeper/state.go` — KV store helpers
- `keeper/grpc_query.go` — query handlers
- `keeper/keeper_test.go` — 91 tests
- `client/cli/tx.go` — 12 TX CLI commands
- `client/cli/query.go` — 5 query CLI commands
- `module.go` — AppModuleBasic, AppModule, RegisterServices

Also reference:
- `/Users/yuai/Desktop/legible_money/proto/legible/auth/` — existing proto definitions
- `/Users/yuai/Desktop/legible_money/reports/audits/ANTE-CHAIN.md` — ante handler audit
- `/Users/yuai/Desktop/legible_money/docs/PARAMETERS.md` — auth param defaults

## Proto Definitions

Create `proto/zerone/auth/v1/`:

### `tx.proto`
Port from draft proto. Messages:
- MsgRegisterAccount (address, public_key, did, metadata)
- MsgCreateSession (owner, session_public_key, permissions, expiry_height)
- MsgRevokeSession (owner, session_key_hash)
- MsgFreezeAccount (authority, target_address, reason)
- MsgUnfreezeAccount (authority, target_address)
- MsgSetRecoveryConfig (owner, threshold, total_shards, shard_holders)
- MsgInitiateRecovery (initiator, target_address, new_public_key)
- MsgSubmitRecoveryShard (shard_holder, recovery_id, shard_data)
- MsgChallengeRecovery (challenger, recovery_id, reason)
- MsgExecuteRecovery (initiator, recovery_id)
- MsgUpdateParams (authority, params)

Each with a corresponding Response message.

### `query.proto`
- QueryAccount (address) → Account
- QueryAccountByDID (did) → Account
- QuerySessionKeys (owner) → []SessionKey
- QueryParams → Params
- QueryFrozenAccounts → []Account

### `genesis.proto`
- GenesisState { params, accounts, session_keys, recovery_configs, frozen_addresses }
- Params { max_session_keys, session_key_max_lifetime_blocks, max_recovery_shards, recovery_challenge_period_blocks, recovery_execution_delay_blocks, max_metadata_length, require_did }

### `types.proto`
- Account { address, public_key, did, metadata, registered_at_block, frozen, freeze_reason }
- SessionKey { owner, key_hash, public_key, permissions, created_at_block, expiry_block }
- RecoveryConfig { owner, threshold, total_shards, shard_holders }
- PendingRecovery { id, target_address, initiator, new_public_key, shards_received, challenged, challenge_reason, initiated_at_block }

**Naming change:** All proto packages use `zerone.auth.v1`, not `legible.auth.v1`.

## Module Implementation

### Directory structure
```
x/auth/
├── keeper/
│   ├── keeper.go          # Keeper struct, constructor, state CRUD
│   ├── msg_server.go      # All 11 message handlers
│   ├── grpc_query.go      # Query handlers
│   ├── keeper_test.go     # Tests
│   └── migrator.go        # Empty Migrator stub (for x/upgrade)
├── types/
│   ├── keys.go            # Store key prefixes
│   ├── expected_keepers.go # Interface deps
│   ├── codec.go           # RegisterInterfaces
│   ├── errors.go          # Sentinel errors
│   ├── *.pb.go            # Generated (from proto)
│   └── *.pb.gw.go         # Generated (gRPC gateway)
├── client/
│   └── cli/
│       ├── tx.go          # TX commands
│       └── query.go       # Query commands
└── module.go              # AppModule implementation
```

### Key porting rules

1. **All types from proto** — do NOT copy hand-written structs from draft.
   Use the generated pb.go types everywhere.

2. **State encoding: proto marshal** — replace all `json.Marshal/Unmarshal`
   with `proto.Marshal/Unmarshal` for KV store operations.

3. **Unified BPS: 1,000,000 scale** — verify all auth params use this scale.

4. **Security fixes baked in:**
   - Ante handler checks apply to ALL message types including SDK messages
     (B13 audit P0: ante bypassed for standard SDK messages — fixed in R1 draft)
   - Session key permissions are enforced (not decorative)
   - Recovery challenge period is enforced
   - DID derivation is validated

5. **Migrator stub:**
   ```go
   type Migrator struct {
       keeper Keeper
   }

   func NewMigrator(keeper Keeper) Migrator {
       return Migrator{keeper: keeper}
   }

   // Migrate1to2 will handle v1→v2 state migration when needed.
   // func (m Migrator) Migrate1to2(ctx sdk.Context) error { ... }
   ```

### Port the 11 handlers

For each handler in the draft's `msg_server.go`:
1. Read the implementation
2. Rewrite against proto-generated types
3. Keep the same logic and validations
4. Use proto marshal for state
5. Add gas consumption
6. Add events

### Port the tests

All 91 tests from the draft carry over. Rewrite against proto types
but keep the same assertions and test scenarios.

## Wire into app.go

Add to the app skeleton from R1-1/R1-2:
- Store key
- Keeper field
- ModuleManager registration
- BeginBlocker (session key expiry cleanup)
- InitGenesis / ExportGenesis
- RegisterServices (msg server + query server)

## Verification

```bash
make proto-gen                    # regenerate if proto changed
go build ./...
go vet ./...
go test ./x/auth/... -count=1 -v
make build
build/zeroned init test --chain-id zerone-test  # still works
```

## Commit

```
feat(auth): port auth module — accounts, sessions, recovery, freeze
```

## Do NOT

- Use hand-written types — everything from proto
- Skip the Migrator stub
- Change the handler logic (port faithfully from draft)
- Forget to rename legible→zerone in all strings, events, error messages
- Skip gas costs on any handler
