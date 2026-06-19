# x/contagion — Design document in code form

This file answers the **three open questions** the ZO docs
(`CONTAGION-MATH.md`, `DEPLOY-DEVNET.md`) leave to the Zerone implementation.
The answers are encoded in the proto + keeper of this module; this prose just
makes them explicit.

## Question 1 — "Can a contract custody its own reserve and
`_transferFromReserve()` at runtime, or does the standard require all balance
moves to go through user-signed txns?"

**Answer: a module account custodies the reserve.**

Zerone's ZRN-20 (x/tokens) moves balances inside keeper logic, not via
user-signed bank sends — `TransferToken`'s keeper directly mutates balance
KV entries. So a module *can* custody funds and disburse them at runtime. The
contagion module therefore holds the 60,433,333 ZO reserve in its own module
account (`contagion` module name) funded at genesis (or via a one-time
governance mint). `_transferFromReserve(spreader, amt)` becomes
`bankKeeper.SendCoinsFromModuleToAccount(ctx, contagion.ModuleName, spreader,
coins)`. No user signature is needed for the reward leg — exactly what the
sneeze requires. See `keeper.Sneeze()`.

## Question 2 — "Is there a hook on `transfer()` for post-transfer logic, or do
we wrap the standard transfer in a custom function?"

**Answer: both are supported. The default path is a post-transfer hook; the
fallback is an explicit wrapper message.**

- **Hook path (recommended).** The tokens module's `TransferToken` keeper
  accepts an optional `ContagionHook` interface (see
  `x/contagion/types/expected_keepers.go`). After a successful ZO transfer it
  calls `hook.OnTokenTransfer(ctx, tokenId, from, to, amount)`. The contagion
  keeper checks `IsInfected(to)`; if false and reserve ≥ 2×reward, it mints the
  sneeze. This keeps ZO transfer as the standard ZRN-20 path (no extra user
  tx, no extra signer) and adds ≤ one read + (rarely) one mint. The tokens
  module stays unaware of contagion mechanics — it just holds an interface
  pointer, wired in `app.go`, nil for non-ZO tokens.
- **Wrapper path.** `MsgSneeze` is an explicit user-facing transaction that
  performs the ZO transfer (delegating to the tokens keeper) and then runs the
  contagion check. Use this when a transfer happens outside the hook (e.g. an
  airdrop script, LP seeding) and the sneeze still needs to fire, or as the
  single entrypoint during devnet rehearsal before the hook is wired.

The hook path keeps the gas target (≤ 2× plain transfer, see DEPLOY-DEVNET
Phase 2): plain transfer = 1 move; first-touch transfer = 1 move + 1 read + 1
mint-from-reserve (rare path). Repeat transfers cost the same as plain.

## Question 3 — "What gas/fee surface does a sneeze add? Acceptable target: <2×
the cost of a normal transfer."

**Answer: see Question 2 — the hook path adds at most one KV read + (only on
first touch) one reserve-decrement + two module-account sends. Repeat
transfers add only the one `IsInfected` read. Meets the ≤2× target.** The
devnet rehearsal (`DEPLOY-DEVNET.md` Phase 2) is where this gets measured;
this module is structured so the hot path (repeat transfer) is a single KV
`Has`.

## Other locked decisions (answering PARAMETERS.md "we are not doing")

- **No mutable formula.** `MsgSetContagion` is the *only* setter and it is
  self-disabling: it requires `configured == false` and clears `authority` on
  success. There is intentionally **no** `MsgUpdateContagion`. The sneeze
  reward, reserve size, and ZO token id are immutable post-deploy — on-chain
  equivalent of PARAMETERS.md's "✅ Sneeze reward formula in the contract, not
  adjustable" and "Full contract renounce".
- **No transfer tax, no reflection, no cooldowns, no max-wallet.** None of
  these exist in the module. The reserve only ever *decreases*; there is no
  `mint()` that grows supply (the contagion reserve is pre-minted at TGE per
  PARAMETERS.md "Total supply minted at deploy, no further mint function").
- **Reserve dormancy.** When `reserve_remaining < reward_sender +
  reward_receiver` the keeper returns silently and emits no `Sneeze`. ZO then
  behaves as a normal coin — exactly CONTAGION-MATH.md "reserve dry, mechanic
  dormant". Partial rewards are explicitly **not** fired (DEPLOY-DEVNET Phase 2
  test case).
- **`already_infected` is one-way.** `SetInfected` exists; `UnsetInfected` does
  not. The flag tracks the *receiver*, never the sender, per CONTAGION-MATH.md.

## Module boundary

`x/contagion` does **not** reimplement token transfer. It depends on:
- `x/tokens` (to move ZO balances inside `MsgSneeze`, and to look up the ZO
  token's decimals/creator for validation), via `types.TokensKeeper`.
- `x/bank` (to custody + disburse the reserve), via `types.BankKeeper`.

The tokens module depends on `x/contagion` *only* through the
`ContagionHook` interface — a nil-safe optional pointer. Non-ZO tokens pay
zero contagion cost.

## File map

```
proto/zerone/contagion/v1/
  types.proto    ContagionState, SneezeEvent, InfectionRecord
  tx.proto      MsgSneeze, MsgSetContagion
  query.proto   QueryContagionState, QueryIsInfected
  genesis.proto GenesisState, Params
x/contagion/
  keeper/keeper.go       Keeper + Sneeze() core + OnTokenTransfer hook impl
  keeper/msg_server.go  MsgSneeze (wrapper) + MsgSetContagion (one-time)
  keeper/state.go       KV CRUD for state + infected set
  keeper/grpc_query.go  QueryContagionState, QueryIsInfected
  types/keys.go         ModuleName, StoreKey, key prefixes/builders
  types/errors.go       sentinel errors
  types/codec.go        amino + interface registration
  types/types.go        ValidateBasic / GetSigners / DefaultGenesis / Validate
  types/expected_keepers.go  BankKeeper + TokensKeeper + ContagionHook interfaces
  module.go             AppModule + AppModuleBasic
  DESIGN.md             this file
```