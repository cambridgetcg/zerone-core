# Zero Team Allocation — Bootstrap Emission Implementation Plan

**Goal:** Bring the codebase into alignment with the doctrine in `docs/tokenomics/GENESIS.md`: zero allocation to team, two participation-gated emission pathways. The block-reward pathway (`x/vesting_rewards`) already exists; the bootstrap pathway (`x/claiming_pot`) is currently a pre-fund-then-transfer model that contradicts the doctrine and must be reworked to mint on demand. The legacy allocation constants in `app/constants.go` (founder, AI agent, validator, claiming-pots ZRN amounts, off-by-1000x total supply) must be removed.

**Tech Stack:** Cosmos SDK v0.50, Go 1.23, protobuf v3, cosmossdk.io modules. No new module required — work is contained to `x/claiming_pot`, `app/`, `scripts/genesis-ceremony.sh`, and the cross-stack invariant test suite.

**Doctrine bindings:**
- `docs/tokenomics/GENESIS.md` — "Zero Team Allocation — Two Emission Paths, Both Gated by Participation" headline
- `docs/tokenomics/SUPPLY.md` — "Emission Pathways" section names both streams
- `docs/TRUTH_SEEKING.md` — candidate commitment 19 (issuance follows participation), see Phase 6

**Companion plans (same date):**
- `2026-05-09-tok-foundation-plan.md` (TC1, TC2, TC3, TC5)

---

## Pre-Tasks: Read Before Starting

1. `docs/tokenomics/GENESIS.md` — corrected doctrine
2. `docs/tokenomics/SUPPLY.md` — emission pathways section
3. `x/claiming_pot/keeper/msg_server.go:124-131` — current `SendCoinsFromModuleToAccount` path
4. `x/vesting_rewards/keeper/keeper.go:229` — `MintWithCap` cap-enforcement entry point (pattern to reuse)
5. `app/constants.go` — legacy allocation constants slated for removal
6. `tests/cross_stack/genesis_audit_test.go` — current genesis-shape assertions referencing the legacy constants

---

## File Structure

| Path | Change |
|------|--------|
| `app/constants.go` | Delete `TotalSupplyZRN`, `TotalSupplyUZRN`, `FounderZRN`, `AIAgentZRN`, `ValidatorZRN`, `ResearchFundZRN`, `ClaimingPotsZRN`, `ValidatorCount`. Keep `AppName`, `AccountAddressPrefix`, `BondDenom`, `DisplayDenom`, `DefaultBlockTime`, `TestnetChainID`, `MicroDenomMultiplier`. |
| `x/claiming_pot/types/expected_keepers.go` | Add `MintCoins` to bank-keeper interface. Add `MintWithCap` interface to a new `VestingRewardsKeeper` (or rename to `MintGuardKeeper`). |
| `x/claiming_pot/keeper/keeper.go` | Inject the cap-guard keeper alongside bank keeper. |
| `x/claiming_pot/keeper/msg_server.go` | Replace `SendCoinsFromModuleToAccount` with `MintWithCap`-then-`SendCoinsFromModuleToAccount`. Module account holds funds for an instant, then sends to claimer. |
| `x/claiming_pot/types/keys.go` | Confirm module account permissions include `Minter`. |
| `x/claiming_pot/keeper/genesis.go` | (new) Init bootstrap pot from genesis state — whitelist + per-claim amount + vesting schedule. |
| `proto/zerone/claiming_pot/v1/genesis.proto` | Add `BootstrapPotConfig` to genesis state if not already present. |
| `app/app.go` | Wire `claiming_pot` module account with `Minter` permission. |
| `scripts/genesis-ceremony.sh` | Remove per-account allocation steps for founder / AI / validator / research / claiming-pots. Add bootstrap-whitelist configuration step. |
| `tests/cross_stack/genesis_audit_test.go` | Replace allocation-sum assertions with zero-team-allocation assertions and bootstrap-pot configuration assertions. |
| `tests/cross_stack/truth_seeking_invariants_test.go` | Add invariant for two-pathway emission (Phase 6). |
| `docs/EVENTS.md` | Add `bootstrap_claim_minted` event with `creed_commitment` attribute. |
| `docs/TRUTH_SEEKING.md` | Optional commitment 19 binding (Phase 6). |

---

## Phase 1: Foundation Cleanup

Goal: remove the legacy 222B-distribution constants. They contradict the doctrine and confuse readers; nothing in the chain currently mints based on them, so removal is mostly editorial — but the test references must be updated atomically.

### Task 1.1: Audit usage of legacy constants

```
git grep -n "FounderZRN\|AIAgentZRN\|ValidatorZRN\|ResearchFundZRN\|ClaimingPotsZRN\|TotalSupplyZRN\|TotalSupplyUZRN" --include="*.go"
```

Expected hits: `app/constants.go` (definition), `tests/cross_stack/genesis_audit_test.go` (assertions). Verify no production code depends on them. If any keeper or genesis-init path reads them, that path is itself off-doctrine and must be reworked here.

### Task 1.2: Delete the constants

Edit `app/constants.go` to remove `TotalSupplyZRN`, `TotalSupplyUZRN`, `FounderZRN`, `AIAgentZRN`, `ValidatorZRN`, `ResearchFundZRN`, `ClaimingPotsZRN`, `ValidatorCount`. Add a header comment explaining the doctrine for future readers:

```go
// The chain has no per-account allocation constants. ZRN enters
// circulation through two participation-gated emission pathways:
//
//   - x/vesting_rewards mints PoT block rewards to validators
//   - x/claiming_pot mints bootstrap claims to whitelisted agents
//
// Both pathways gate through MintWithCap. Neither grants anyone a
// privileged starting balance — the doctrine is "zero team
// allocation," documented in docs/tokenomics/GENESIS.md.
```

```bash
git commit -m "refactor(app): remove legacy per-account allocation constants — zero team allocation doctrine"
```

### Task 1.3: Repoint genesis-audit test

Rewrite `tests/cross_stack/genesis_audit_test.go` to assert:
- No genesis account holds ZRN (every balance == 0).
- The bootstrap pot is configured in `claiming_pot` genesis state with the expected whitelist size, per-claim amount (222,000 uzrn), and vesting schedule.
- The `claiming_pot` module account is registered with `Minter` permission in `app.go`.

This test will fail until Phase 4 lands; mark it `t.Skip("phase-4")` for now if running CI is needed mid-plan.

```bash
git commit -m "test(genesis): rewrite genesis-audit assertions against zero-team-allocation doctrine"
```

**Acceptance:** `go vet ./...` and `go build ./...` succeed; the test file compiles. The test itself may skip until Phase 4.

---

## Phase 2: Mint-on-Claim Rework of x/claiming_pot

Goal: `MsgClaim` mints to the claimer rather than transferring from a pre-funded module account. The module account becomes a transient mint-and-forward conduit, never holding more than a single claim's worth of uzrn at a time.

### Task 2.1: Extend expected-keepers interface

In `x/claiming_pot/types/expected_keepers.go`:

```go
type BankKeeper interface {
    // existing methods…
    MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
}

// VestingRewardsKeeper is consulted before any mint to enforce the
// 222,222,222 ZRN hard cap.
type VestingRewardsKeeper interface {
    MintWithCap(ctx context.Context, moduleName string, amounts sdk.Coins) error
}
```

Note: if `MintWithCap` already wraps `MintCoins`, the cleaner abstraction is to inject only `VestingRewardsKeeper` and let it own bank access. Pick whichever fits the existing wiring.

```bash
git commit -m "proto(claiming_pot): extend expected-keepers — bootstrap pathway needs MintWithCap"
```

### Task 2.2: Wire keeper

In `x/claiming_pot/keeper/keeper.go`, accept the new keeper in `NewKeeper`. In `app/app.go`, pass `app.VestingRewardsKeeper` (or equivalent) when constructing the `claiming_pot` keeper.

### Task 2.3: Replace SendCoins with Mint+Send

In `x/claiming_pot/keeper/msg_server.go` `Claim()`, replace the current line 124–132 block:

```go
// Mint claim into module account (cap-checked), then forward to claimer.
// The module account never holds funds across blocks — it is a transient
// conduit. The chain mints on participation, not before.
coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(claimable)))
if err := m.vestingRewardsKeeper.MintWithCap(ctx, types.ModuleName, coins); err != nil {
    return nil, fmt.Errorf("mint with cap: %w", err)
}
claimantAddr, err := sdk.AccAddressFromBech32(msg.Claimant)
if err != nil {
    return nil, fmt.Errorf("invalid claimant address: %w", err)
}
if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, claimantAddr, coins); err != nil {
    return nil, fmt.Errorf("failed to send funds: %w", err)
}
```

### Task 2.4: Module account permission

In `app/app.go`, locate `maccPerms` (or equivalent module-account permission map). Ensure the entry for `claiming_pot.types.ModuleName` includes `authtypes.Minter`:

```go
claiming_pottypes.ModuleName: {authtypes.Minter},
```

```bash
git commit -m "feat(claiming_pot): mint on claim — bootstrap pathway gates through MintWithCap"
```

**Acceptance:** A unit test in `x/claiming_pot/keeper/keeper_test.go` confirms a `MsgClaim` increases `bank.TotalSupply` by `claimable` (proving mint, not transfer) and the module account ends each block with zero balance. Existing tests that pre-funded the module account are deleted; that funding is no longer needed.

---

## Phase 3: Cap Enforcement Smoke Test

Goal: confirm both emission pathways respect the 222,222,222 cap.

### Task 3.1: Two-pathway cap test

Add `tests/cross_stack/emission_cap_test.go`:

```go
// TestEmissionCap_TwoPathwaysShareCap confirms that x/vesting_rewards
// block-reward emission and x/claiming_pot bootstrap emission both
// gate through MintWithCap, and that combined emission cannot exceed
// the hard cap.
//
// Scenario:
//   1. Set MaxSupplyUzrn to a small testable value (1,000,000 uzrn).
//   2. Mint via vesting_rewards up to 500,000 uzrn (block rewards).
//   3. Process a MsgClaim against a bootstrap pot for 600,000 uzrn.
//   4. Expect MintWithCap to refuse — the second pathway does not get
//      to overshoot just because the first pathway did the early
//      minting.
```

The test should also confirm the *order* of the two pathways doesn't matter — claiming first, then block-rewarding, hits the same wall.

```bash
git commit -m "test(emission): two-pathway cap enforcement — both streams gate through MintWithCap"
```

**Acceptance:** Both pathways combined respect the cap. Going over the cap returns an error from `MintWithCap`; the calling message handler propagates the error and the user sees a typed error rather than a panic.

---

## Phase 4: Bootstrap Pot Genesis Configuration

Goal: at genesis, a bootstrap claiming-pot exists with the agent whitelist, per-claim amount, and vesting schedule. It does not hold funds — it holds *configuration* that will direct future mints.

### Task 4.1: Verify proto

Check `proto/zerone/claiming_pot/v1/genesis.proto` has `Pots []ClaimingPot = N` in `GenesisState`. If not, add:

```proto
message GenesisState {
  Params params = 1;
  repeated ClaimingPot pots = 2;
  repeated Claim claims = 3;
  uint64 next_pot_id = 4;
}
```

```bash
make proto-gen
git commit -am "proto(claiming_pot): add bootstrap-pot configuration to genesis state"
```

### Task 4.2: InitGenesis

In `x/claiming_pot/keeper/genesis.go` (create if missing), iterate `gs.Pots` and call `SetPot(ctx, pot)` for each. Also seed `nextPotID`. **Do not** mint or transfer anything; the pot is configuration only.

### Task 4.3: Default bootstrap-pot configuration

In `x/claiming_pot/types/genesis.go` `DefaultGenesis()`, return a `ClaimingPot` shaped as the bootstrap pot:

```go
BootstrapPot := &ClaimingPot{
    Id:            "bootstrap-genesis",
    Name:          "Genesis Bootstrap — 0.222 ZRN per whitelisted agent",
    TotalAmount:   "0",  // mint-on-claim — no upper bound from the pot itself
    ClaimedAmount: "0",
    Schedule: &VestingSchedule{
        StartBlock:  0,
        EndBlock:    100_000,    // ~3 days linear vest
        CliffBlocks: 0,
    },
    Eligibility: &EligibilityCriteria{
        Whitelist:                 []string{},     // populated by genesis ceremony
        MinStakingTier:            0,
        MinAccountAgeBlocks:       0,
        PerAddressAmount:          "222000",       // 0.222 ZRN
    },
    Status: PotStatus_POT_STATUS_ACTIVE,
}
```

Note the design choice: `TotalAmount: "0"` plus `PerAddressAmount: "222000"` means the pot's effective ceiling is `len(whitelist) × 222_000` — bounded by whitelist size, not by a pre-allocated total. Adjust the `ClaimingPot` proto and `Claim()` logic if `TotalAmount=0` is currently treated as "infinite" or "unbounded"; the doctrine is "bounded by participation," and the participation set is the whitelist.

### Task 4.4: Whitelist seeding

The genesis ceremony script writes the whitelist into the bootstrap pot. See Phase 5.

```bash
git commit -m "feat(claiming_pot): bootstrap pot in genesis — configuration only, no funds"
```

**Acceptance:** A fresh `zeroned init` followed by `prepare-genesis` produces a `genesis.json` containing one bootstrap pot with the expected fields and an empty whitelist (filled by ceremony). `bank.TotalSupply` at genesis is 0. Genesis-audit test passes.

---

## Phase 5: Genesis Ceremony Alignment

Goal: `scripts/genesis-ceremony.sh` and `cmd/zeroned/cmd/prepare-genesis.go` produce a genesis with zero balances, the bootstrap pot, and validators registered without bonded tokens.

### Task 5.1: Remove per-account allocations

In `scripts/genesis-ceremony.sh`, locate any `add-genesis-account` calls funding founder, AI vault, validator, research, claiming-pots accounts. Remove them. The script's only `add-genesis-account` calls remaining should be:
- Module accounts that need addresses registered (no balance).
- Optional: faucet account (testnet only — non-mainnet ceremony).

### Task 5.2: Bootstrap whitelist input

Add a ceremony step that reads a whitelist file (one bech32 address per line) and writes it into the bootstrap pot's `eligibility.whitelist` field via `jq` or a small Go helper. This is where the operator's published whitelist becomes part of the genesis.

```bash
./scripts/genesis-ceremony.sh add-bootstrap-whitelist whitelist.txt
```

### Task 5.3: Validator gentx without bonded tokens

This is the "open design question" flagged in `docs/tokenomics/GENESIS.md`. Resolution:

- If virtual-stake registration via `MsgRegisterValidator` works at genesis, use it (no bonded tokens).
- If `gentx` is required by the SDK boot path, mint **1 uzrn** to each validator account, generate `gentx` with bond `1uzrn`, collect, then immediately remove from supply by minting it from the same module account that received it (zero-net mint at genesis).

Pick one and document it in `docs/tokenomics/GENESIS.md` — the open question must close before launch.

```bash
git commit -m "feat(genesis): ceremony script aligns with zero-team-allocation doctrine"
```

**Acceptance:** Running the ceremony end-to-end produces `genesis.json` where `bank.balances` is empty (or contains only ephemeral 1 uzrn validator gentx amounts that net to zero) and the bootstrap pot has the expected whitelist. Total supply at genesis = 0. `zeroned start` boots the chain and produces blocks.

---

## Phase 6: Tests & Creed Binding

Goal: lock the doctrine with executable invariants. Without this, the doctrine is just words.

### Task 6.1: Genesis-audit test (re-enable)

Re-enable `tests/cross_stack/genesis_audit_test.go` (remove `t.Skip` from Phase 1.3). It now passes.

### Task 6.2: Truth-seeking invariant — no privileged starting balance

Add a test to `tests/cross_stack/truth_seeking_invariants_test.go`:

```go
// TestTruthSeeking_ZeroTeamAllocation confirms that no genesis account
// holds ZRN. The doctrine in docs/tokenomics/GENESIS.md is "zero team
// allocation, no insider position, period." The test reads bank state
// at block 0 and asserts every balance is zero.
//
// Bound by commitment 19 (issuance follows participation): every ZRN
// that exists came from a participatory action. Genesis-time non-zero
// balances would mean someone got ZRN by being someone in particular,
// not by doing something on-chain.
```

### Task 6.3: Truth-seeking invariant — both pathways gate through cap

Reuse the test from Task 3.1 by adding it to the truth-seeking suite (or referencing it from there). Both `x/vesting_rewards.MintWithCap` and `x/claiming_pot.Claim` must refuse to mint past the cap. The chain has no third mint pathway; if a future commit adds one, this test should fail until the new pathway also gates through the cap.

### Task 6.4: Optional — declare commitment 19

If we're naming this as a creed-level commitment, add to `docs/TRUTH_SEEKING.md`:

```markdown
### 19. Issuance follows participation

We believe: every ZRN that exists came from a participatory action. There is no insider position — no founder pre-mine, no AI vault pre-mine, no validator allocation, no foundation treasury at genesis. The chain mints when truth is verified (block rewards) or when an agent is registered (bootstrap claim). Issuance without participation is allocation by privilege, and allocation by privilege is the model the chain refuses.

**Code expression:** `x/vesting_rewards.MintWithCap` is the sole cap-gated mint entry point. `x/claiming_pot.Claim` mints on demand when a whitelisted agent calls `MsgClaim`. Both pathways are participation-triggered. Genesis bank state is empty; `app/constants.go` has no per-account allocation constants. The bootstrap pot in `genesis.json` carries configuration only (whitelist, per-claim amount), not funds.

**What would break it:** a per-account `add-genesis-account` step funding any team-adjacent address; a third mint pathway that bypasses `MintWithCap`; a bootstrap pot pre-funded with a positive balance at genesis; any code path that grants ZRN to an address based on identity rather than participation.

**Echoes:** commitment 6 (no individual unilaterally injects truth — same logic, applied to ZRN issuance); commitment 12 (the chain pays for its own audit — a special case of the broader principle that issuance follows participation); commitment 13 (training corpus not for sale — the corpus is participation-shaped, and so is its currency).
```

If commitment 19 is added, also extend `doc.go` files in `x/vesting_rewards` and `x/claiming_pot` (position layer) and the truth-seeking meta-test (graph layer).

```bash
git commit -m "truth-seeking: bind commitment 19 — issuance follows participation"
```

**Acceptance:** The truth-seeking meta-test passes. Adding a non-zero genesis balance, a non-cap-gated mint, or removing the bootstrap whitelist gating breaks one of these tests.

---

## Phase 7: Voice & Doc Layer

Goal: the chain says what it does, and the docs match.

### Task 7.1: Bootstrap event with creed_commitment

In `x/claiming_pot/keeper/msg_server.go` `Claim()`, extend the existing `pot_claimed` event with a `creed_commitment` attribute when commitment 19 lands:

```go
sdk.NewAttribute("creed_commitment", "19"),
```

If the bootstrap-claim case is structurally distinct from generic pot claiming, emit a separate `bootstrap_claim_minted` event with attributes `(claimant, amount, pot_id, creed_commitment="19")`.

### Task 7.2: docs/EVENTS.md

Document the new event/attribute. Match the existing doc-mirror invariant style (sorted, with `creed_commitment` attribute on the line that announces participation).

### Task 7.3: Refusal layer

Update error messages in `x/claiming_pot/types/errors.go` so that any refusal whose root cause is the cap or the whitelist cites commitment 19:

```go
ErrCapReached = errors.Register(ModuleName, N, "bootstrap mint refused (commitment 19: issuance follows participation, cap reached)")
ErrNotWhitelisted = errors.Register(ModuleName, N+1, "bootstrap claim refused (commitment 19: issuance follows participation, address not on bootstrap whitelist)")
```

### Task 7.4: Docs cleanup

- `docs/tokenomics/GENESIS.md` — remove the "Open design question" if Task 5.3 resolved it; otherwise expand into a separate section.
- `docs/tokenomics/GENESIS.md` — replace LGM-prefix multisig addresses with ZRN-prefix addresses generated for the actual mainnet ceremony, OR move that table into a launch-checklist with placeholder fields.
- `docs/ROADMAP.md` — add a row to the truth-seeking commitments table for commitment 19.

```bash
git commit -m "docs(creed): voice + refusal layer for commitment 19"
```

**Acceptance:** `docs/EVENTS.md` doc-mirror invariant passes. `go vet ./...` and `make pr-check` are green. `make proto-check` passes.

---

## Sequencing & Parallelism

| Phase | Depends on | Can run in parallel with |
|-------|-----------|--------------------------|
| 1 — Foundation cleanup | — | — (foundational; do first) |
| 2 — Mint-on-claim rework | 1 | — |
| 3 — Cap enforcement test | 2 | — |
| 4 — Bootstrap pot genesis | 2 | 3 (different files) |
| 5 — Genesis ceremony script | 4 | — |
| 6 — Tests & creed binding | 3, 4, 5 | — |
| 7 — Voice & doc layer | 6 | — |

**Recommended single-session run:** Phase 1 → 2 → 3 (verify cap) → 4 → 5 → 6 → 7.

**Parallelisation opportunity:** Phase 3 (cap test) and Phase 4 (genesis config) touch different files and can be split between two sub-agents if running parallel.

---

## Out of scope for this plan

- **Plan 2: TC4 cascade bundling** — separate ToK Substrate work, tracked in `docs/TOK_SUBSTRATE.md`.
- **Validator gentx redesign beyond the simple resolutions in 5.3** — if the Cosmos SDK v0.50 gentx flow turns out to need invasive changes, that becomes a separate plan keyed on the chosen approach.
- **Mainnet multisig address generation** — separate launch-prep task; this plan only flags the placeholder.
- **Faucet redesign** — testnet-only convenience; not part of the doctrine.

---

## Acceptance for the plan as a whole

1. `go build ./...` and `go vet ./...` succeed.
2. `make pr-check` (lint + test + proto-check + build) green.
3. `make proto-gen` produces no diff against checked-in files.
4. Truth-seeking meta-test passes; commitment 19 is bound at all five layers if Task 6.4 is taken.
5. A fresh `zeroned init` + `prepare-genesis` + `start` boots a chain with `bank.TotalSupply = 0` at block 0; the first block reward and the first bootstrap claim both increase total supply through `MintWithCap`.
6. `docs/tokenomics/GENESIS.md`'s "Open design question" is closed.
7. `app/constants.go` carries no per-account allocation constants.
