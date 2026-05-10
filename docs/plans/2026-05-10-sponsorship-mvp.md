# Sponsorship MVP — Patron-Funded Work Bounties

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** New `x/sponsorship` module. A sponsor escrows ZRN against a typed bounty (domain + per-artifact price + target count + window). Verified facts in that domain submitted during the window trigger payouts from escrow to the fact submitter. Sponsor can cancel and reclaim remaining. No new mint pathway — sponsorship circulates existing supply, verification is the gate.

**Architecture:** Three types — `BountyOrder` (sponsor's commitment + remaining escrow), `BountyFulfillment` (per-payout record), and a module account holding escrowed funds. Three messages — `MsgCreateBountyOrder` (escrow lock), `MsgFulfillBounty` (verification-gated payout), `MsgCancelBountyOrder` (refund). Knowledge keeper is consulted read-only to check fact validity. Bank keeper handles escrow lock and payout. Single-mint-entry invariant preserved (no `MintWithCap` calls in this module).

**Tech Stack:** Cosmos SDK v0.50, Go 1.24, protobuf v3, buf toolchain.

**Naming note:** This module is functionally the "patronage MVP" requested in conversation, but renamed to `sponsorship` to disambiguate from the *existing* per-fact patronage in `x/knowledge` (which locks funds against an already-existing fact to boost its energy/fitness — Patreon-style support for individual facts). The new module is research-grant-style funding for as-yet-unproduced work. The two compose: a sponsor funds new work via `x/sponsorship`; once the resulting fact lands, anyone can patron it via existing `x/knowledge` patronage.

**Doctrine bindings:**
- Commitment 1 (methodology over statement) — payout criteria are objective (verification status), never editorial
- Commitment 8 (panel weights skill, not bond) — sponsors can't buy verification; they only pay for facts the chain's panel verified independently
- Commitment 12 (chain pays for its own audit) — extended: chain mediates external payment for the work it audits
- Commitment 20 (issuance follows participation) — payout follows verified participation, never before; no privileged starting balance for sponsored work either

---

## Pre-Tasks: Read Before Starting

1. `x/claiming_pot/` — closest existing module pattern; recently extended via the late-bootstrap-entry plan
2. `x/knowledge/keeper/msg_server.go:920-1000` — existing per-fact patronage pattern for escrow lock semantics
3. `x/knowledge/types/types.pb.go:1971` — `Fact` struct fields (Domain, Status, SubmittedAtBlock, VerifiedAtBlock, Submitter)
4. `app/app.go:373` — module-account permission registration pattern
5. `app/app.go:1310-1325` — keeper construction pattern
6. `tests/cross_stack/harness_test.go` — `NewTestHarness` and `AdvanceBlocks` for integration tests
7. `tests/cross_stack/late_bootstrap_test.go` — recently-landed integration test pattern, reuse the shape

---

## File Structure

| Path | Responsibility |
|------|----------------|
| `proto/zerone/sponsorship/v1/state.proto` | `BountyOrder`, `BountyFulfillment`, `BountyStatus` enum |
| `proto/zerone/sponsorship/v1/tx.proto` | `MsgCreateBountyOrder`, `MsgFulfillBounty`, `MsgCancelBountyOrder` + responses, service definition |
| `proto/zerone/sponsorship/v1/genesis.proto` | `GenesisState` (params, orders, fulfillments) |
| `proto/zerone/sponsorship/v1/query.proto` | Minimal: `BountyOrder(id)`, `BountyOrders()` |
| `x/sponsorship/types/keys.go` | Store key prefixes, module name |
| `x/sponsorship/types/errors.go` | Typed errors with doctrine citations |
| `x/sponsorship/types/codec.go` | Amino + interface registration |
| `x/sponsorship/types/types.go` | `GetSigners`, `ValidateBasic`, default genesis, params validation |
| `x/sponsorship/types/expected_keepers.go` | `KnowledgeKeeper`, `BankKeeper` interfaces |
| `x/sponsorship/keeper/keeper.go` | Store ops, params, BeginBlocker (expiry sweep) |
| `x/sponsorship/keeper/msg_server.go` | Three msg handlers |
| `x/sponsorship/keeper/grpc_query.go` | Query server |
| `x/sponsorship/keeper/keeper_test.go` | Mocks + unit tests |
| `x/sponsorship/module.go` | App module wiring (BeginBlock, RegisterServices) |
| `x/sponsorship/doc.go` | Position-layer doctrine |
| `app/app.go` | Module account permission (no Minter/Burner — escrow-only), keeper wiring |
| `tests/cross_stack/sponsorship_test.go` | Live-keeper end-to-end test |
| `docs/EVENTS.md` | Voice-layer documentation for 3 events |
| `docs/TRUTH_SEEKING.md` | Graph-layer extension under commitments 12 and 20 |
| `.creed-hash` | Bumped when TRUTH_SEEKING.md changes |

---

## Phase 1: Module Scaffolding

**Goal:** Create the directory layout, keys, errors, and doc — enough for the package to exist and compile (with stub types/functions). No proto-derived types yet.

### Task 1.1: Create types/keys.go

**Files:** Create `x/sponsorship/types/keys.go`

- [ ] **Step 1: Write the file**

```go
package types

const (
	// ModuleName is the canonical identifier for x/sponsorship.
	ModuleName = "sponsorship"

	// StoreKey is the primary store key under which the module's KV
	// store is mounted.
	StoreKey = ModuleName

	// RouterKey is the message routing key.
	RouterKey = ModuleName

	// QuerierRoute is the query routing key.
	QuerierRoute = ModuleName
)

var (
	// ParamsKey holds the module's serialized Params.
	ParamsKey = []byte{0x00}

	// BountyOrderKeyPrefix is the prefix for BountyOrder records.
	// Layout: BountyOrderKeyPrefix || id
	BountyOrderKeyPrefix = []byte{0x01}

	// FulfillmentKeyPrefix is the prefix for BountyFulfillment records.
	// Layout: FulfillmentKeyPrefix || bounty_id || "/" || fact_id
	FulfillmentKeyPrefix = []byte{0x02}

	// BountyCounterKey holds the monotonically-incrementing next-id counter.
	BountyCounterKey = []byte{0x03}
)

// BountyOrderKey returns the KV key for a bounty by id.
func BountyOrderKey(id string) []byte {
	return append(BountyOrderKeyPrefix, []byte(id)...)
}

// FulfillmentKey returns the KV key for a (bounty_id, fact_id) fulfillment.
func FulfillmentKey(bountyID, factID string) []byte {
	key := append([]byte{}, FulfillmentKeyPrefix...)
	key = append(key, []byte(bountyID)...)
	key = append(key, '/')
	key = append(key, []byte(factID)...)
	return key
}

// FulfillmentByBountyPrefix returns the iteration prefix for all fulfillments of a bounty.
func FulfillmentByBountyPrefix(bountyID string) []byte {
	prefix := append([]byte{}, FulfillmentKeyPrefix...)
	prefix = append(prefix, []byte(bountyID)...)
	prefix = append(prefix, '/')
	return prefix
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./x/sponsorship/types/`
Expected: PASS (no symbols defined elsewhere yet, so this should compile standalone).

### Task 1.2: Create types/errors.go

**Files:** Create `x/sponsorship/types/errors.go`

- [ ] **Step 1: Write the file**

```go
package types

import "cosmossdk.io/errors"

var (
	// ErrUnauthorized is raised when the caller is not the bounty's sponsor.
	ErrUnauthorized = errors.Register(ModuleName, 2, "unauthorized")

	// ErrBountyNotFound is raised when a bounty id has no record.
	ErrBountyNotFound = errors.Register(ModuleName, 3, "bounty order not found")

	// ErrBountyNotActive is raised when the bounty's status is not ACTIVE.
	// Commitment 1 (methodology over statement): a fulfilled or canceled
	// bounty cannot be revived by editorial decree.
	ErrBountyNotActive = errors.Register(ModuleName, 4, "bounty order not active")

	// ErrBountyExpired is raised when currentBlock >= bounty.end_block.
	// Sponsor's commitment had a deadline; the chain honors that deadline
	// without exception.
	ErrBountyExpired = errors.Register(ModuleName, 5, "bounty order expired")

	// ErrAlreadyFulfilled is raised when (bounty_id, fact_id) has an
	// existing fulfillment. Each fact qualifies once per bounty.
	ErrAlreadyFulfilled = errors.Register(ModuleName, 6, "fact already fulfilled this bounty")

	// ErrFactNotEligible is raised when the fact does not meet the bounty's
	// criteria. The chain refuses to pay for unverified, off-domain, or
	// out-of-window facts (commitment 8: panel weights skill, not sponsor's
	// preference; the chain decides what counts).
	ErrFactNotEligible = errors.Register(ModuleName, 7, "fact not eligible for this bounty")

	// ErrInvalidConfig is raised for malformed bounty configuration.
	ErrInvalidConfig = errors.Register(ModuleName, 8, "invalid bounty configuration")

	// ErrInsufficientEscrow is raised when the sponsor lacks the funds to
	// back the bounty they're trying to create.
	ErrInsufficientEscrow = errors.Register(ModuleName, 9, "insufficient escrow balance")
)
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./x/sponsorship/types/`
Expected: PASS.

### Task 1.3: Create doc.go (initial position layer)

**Files:** Create `x/sponsorship/doc.go`

- [ ] **Step 1: Write the file**

```go
// Package sponsorship binds external value into ZERONE's verification
// economy. A sponsor escrows ZRN against a typed bounty — a domain, a
// per-artifact price, a target count, and an end-block deadline — and
// the chain pays out from that escrow to fact submitters whose verified
// facts in that domain land during the bounty's window.
//
// This module does NOT mint. Every uzrn that flows from a bounty to a
// worker was already in circulation when the sponsor escrowed it. The
// chain's single mint entry point (x/vesting_rewards.MintWithCap) is
// not touched here. Sponsorship is supply circulation gated by
// verification, not new emission.
//
// Doctrine:
//
//   - Commitment 1 (methodology over statement): payout criteria are
//     objective — the chain checks fact.Status == VERIFIED, fact.Domain
//     equals bounty.Domain, fact.SubmittedAtBlock falls inside the
//     bounty window. The sponsor declares the criteria; the chain
//     enforces them. No editorial path lets a sponsor pay for an
//     unverified fact.
//   - Commitment 8 (panel weights skill, not bond): sponsors do not
//     verify the facts they pay for. Verification remains the work of
//     qualified validators whose calibration the chain tracks. A wealthy
//     sponsor cannot buy a fact into existence — they can only fund
//     work the chain's panel chose to call true.
//   - Commitment 12 (chain pays for its own audit), extended: the chain
//     accepts external payment for the work it audits. The audit
//     pathway is the same; only the funding source widens.
//   - Commitment 20 (issuance follows participation): payout follows
//     verified participation, never before. Sponsorship inherits this
//     shape — the worker is paid for work done and verified, not for
//     promises.
//
// What this module is, and is not:
//
//   - It IS the surface through which external entities (humans, orgs,
//     AI labs, other chains via IBC) can buy work product from ZERONE.
//     A sponsor with funds can direct truth-production into the domains
//     they care about, bounded by the chain's verification spine.
//   - It IS NOT a treasury. The module account is a transient escrow
//     holder, not a custodian. Funds enter when the sponsor creates a
//     bounty, leave when a fact is fulfilled or the sponsor cancels.
//     The module's running balance equals the sum of all active
//     bounties' escrow_remaining.
//   - It IS NOT capable of paying for unverified work. ErrFactNotEligible
//     surfaces any attempt to fulfill a bounty with a fact whose status
//     is not VERIFIED, whose domain doesn't match, or whose submission
//     block is outside the bounty window.
//   - It IS NOT a duplicate of x/knowledge's per-fact patronage. That
//     module lets anyone lock funds against an existing fact to boost
//     its energy/fitness in the metabolism system — Patreon-style
//     support. This module funds the production of facts that don't
//     yet exist — research-grant-style. The two compose: sponsorship
//     funds the work; patronage supports the result.
//
// Refusal voice:
//
//   - ErrFactNotEligible: "fact not eligible for this bounty"
//     (commitment 8: sponsors cannot buy verification, only fund work
//     the chain's panel verifies).
//   - ErrBountyExpired: "bounty order expired" (commitment 1: the
//     sponsor's deadline is a methodological commitment; editorial
//     extension is refused).
//
// Voice layer:
//
//   - sponsorship.bounty_created: announces a new commitment of external
//     value into a domain.
//   - sponsorship.bounty_fulfilled: announces that a sponsor's
//     commitment paid out to a verified worker. Carries the bounty_id,
//     fact_id, worker, and amount.
//   - sponsorship.bounty_canceled: announces sponsor exit and refund.
package sponsorship
```

- [ ] **Step 2: Verify go doc**

Run: `go doc ./x/sponsorship`
Expected: the rendered doc shows the doctrine block.

### Task 1.4: Commit Phase 1

- [ ] **Step 1: Commit**

```bash
git add x/sponsorship/types/keys.go x/sponsorship/types/errors.go x/sponsorship/doc.go
git commit -m "$(cat <<'EOF'
scaffold(sponsorship): module skeleton — keys, errors, doc

New module x/sponsorship for patron-funded work bounties. No emission
pathway, no MintWithCap call — escrow-only circulation gated by
verification. Disambiguated from existing per-fact patronage in
x/knowledge (Patreon-style fact support); this is research-grant-style
work funding.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 2: Proto Definitions

**Goal:** Define the wire format for state, messages, and queries. Then run proto-gen so subsequent phases can use the generated types.

### Task 2.1: Create state.proto

**Files:** Create `proto/zerone/sponsorship/v1/state.proto`

- [ ] **Step 1: Write the file**

```proto
syntax = "proto3";
package zerone.sponsorship.v1;
option go_package = "github.com/zerone-chain/zerone/x/sponsorship/types";

// BountyStatus enumerates the lifecycle states of a BountyOrder.
enum BountyStatus {
  BOUNTY_STATUS_UNSPECIFIED = 0;
  BOUNTY_STATUS_ACTIVE      = 1;  // accepting fulfillments
  BOUNTY_STATUS_FULFILLED   = 2;  // target_count reached
  BOUNTY_STATUS_EXPIRED     = 3;  // currentBlock >= end_block, set by BeginBlocker
  BOUNTY_STATUS_CANCELED    = 4;  // sponsor reclaimed remaining
}

// BountyOrder is a sponsor's commitment of escrowed funds against a
// typed bounty for verified work in a specific domain.
message BountyOrder {
  string       id                   = 1;
  string       sponsor              = 2;   // bech32
  string       domain               = 3;   // must match a registered epistemic domain
  string       price_per_artifact   = 4;   // uzrn as decimal string
  uint32       target_count         = 5;   // total fulfillments accepted
  uint32       fulfilled_count      = 6;
  string       escrow_remaining     = 7;   // uzrn as decimal string
  uint64       start_block          = 8;   // bounty window start (creation block)
  uint64       end_block            = 9;   // bounty window end
  BountyStatus status               = 10;
}

// BountyFulfillment records a single payout from a bounty to a worker
// (the submitter of the fulfilling fact). Indexed by (bounty_id, fact_id).
message BountyFulfillment {
  string bounty_id          = 1;
  string fact_id            = 2;
  string worker             = 3;   // bech32 of the fact submitter
  string amount_paid        = 4;   // uzrn
  uint64 fulfilled_at_block = 5;
}

// Params is the governance-tunable configuration.
message Params {
  // MinTargetCount is the minimum target_count a bounty can declare.
  // Prevents trivially-small bounties that waste state. Default: 1.
  uint32 min_target_count = 1;

  // MinDurationBlocks is the minimum number of blocks a bounty must
  // remain open. Prevents create-and-immediately-expire dust bounties.
  // Default: 100 (~4 minutes at 2.5s blocks).
  uint64 min_duration_blocks = 2;

  // MaxActiveBountiesPerSponsor caps how many ACTIVE bounties a single
  // sponsor can hold simultaneously. Prevents state-bloat from a single
  // address. Default: 16.
  uint32 max_active_bounties_per_sponsor = 3;
}
```

### Task 2.2: Create tx.proto

**Files:** Create `proto/zerone/sponsorship/v1/tx.proto`

- [ ] **Step 1: Write the file**

```proto
syntax = "proto3";
package zerone.sponsorship.v1;
option go_package = "github.com/zerone-chain/zerone/x/sponsorship/types";

import "cosmos/msg/v1/msg.proto";
import "zerone/sponsorship/v1/state.proto";

service Msg {
  option (cosmos.msg.v1.service) = true;

  rpc CreateBountyOrder(MsgCreateBountyOrder) returns (MsgCreateBountyOrderResponse);
  rpc FulfillBounty(MsgFulfillBounty) returns (MsgFulfillBountyResponse);
  rpc CancelBountyOrder(MsgCancelBountyOrder) returns (MsgCancelBountyOrderResponse);
}

// MsgCreateBountyOrder escrows price_per_artifact × target_count uzrn
// from the sponsor's account into the sponsorship module account.
// Returns the assigned bounty id.
message MsgCreateBountyOrder {
  option (cosmos.msg.v1.signer) = "sponsor";
  string sponsor             = 1;
  string domain              = 2;
  string price_per_artifact  = 3;   // uzrn
  uint32 target_count        = 4;
  uint64 duration_blocks     = 5;   // bounty open for this many blocks from creation
}
message MsgCreateBountyOrderResponse { string bounty_id = 1; }

// MsgFulfillBounty pays the fact's submitter price_per_artifact uzrn
// from the bounty's remaining escrow, provided the fact is verified,
// in-domain, in-window, and not already fulfilled for this bounty.
//
// Anyone can trigger fulfillment (the chain enforces all checks). This
// is permissionless: the worker, an indexer, or any helpful agent can
// call it; the worker's address is read from fact.Submitter.
message MsgFulfillBounty {
  option (cosmos.msg.v1.signer) = "caller";
  string caller    = 1;    // any address; this is the tx signer
  string bounty_id = 2;
  string fact_id   = 3;
}
message MsgFulfillBountyResponse {
  string worker      = 1;
  string amount_paid = 2;
  bool   bounty_now_fulfilled = 3;
}

// MsgCancelBountyOrder closes an ACTIVE bounty and refunds the remaining
// escrow to the sponsor. Only the original sponsor can cancel.
message MsgCancelBountyOrder {
  option (cosmos.msg.v1.signer) = "sponsor";
  string sponsor   = 1;
  string bounty_id = 2;
}
message MsgCancelBountyOrderResponse {
  string refunded_amount = 1;
}
```

### Task 2.3: Create genesis.proto

**Files:** Create `proto/zerone/sponsorship/v1/genesis.proto`

- [ ] **Step 1: Write the file**

```proto
syntax = "proto3";
package zerone.sponsorship.v1;
option go_package = "github.com/zerone-chain/zerone/x/sponsorship/types";

import "zerone/sponsorship/v1/state.proto";

message GenesisState {
  Params                       params         = 1;
  repeated BountyOrder         orders         = 2;
  repeated BountyFulfillment   fulfillments   = 3;
  uint64                       next_bounty_id = 4;
}
```

### Task 2.4: Create query.proto

**Files:** Create `proto/zerone/sponsorship/v1/query.proto`

- [ ] **Step 1: Write the file**

```proto
syntax = "proto3";
package zerone.sponsorship.v1;
option go_package = "github.com/zerone-chain/zerone/x/sponsorship/types";

import "zerone/sponsorship/v1/state.proto";

service Query {
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse);
  rpc BountyOrder(QueryBountyOrderRequest) returns (QueryBountyOrderResponse);
  rpc BountyOrders(QueryBountyOrdersRequest) returns (QueryBountyOrdersResponse);
}

message QueryParamsRequest {}
message QueryParamsResponse { Params params = 1; }

message QueryBountyOrderRequest { string id = 1; }
message QueryBountyOrderResponse { BountyOrder order = 1; }

message QueryBountyOrdersRequest {}
message QueryBountyOrdersResponse { repeated BountyOrder orders = 1; }
```

### Task 2.5: Run proto-gen

- [ ] **Step 1: Generate code**

Run: `make proto-gen`
Expected: no errors; files appear under `x/sponsorship/types/` (state.pb.go, tx.pb.go, tx_grpc.pb.go, genesis.pb.go, query.pb.go, query_grpc.pb.go, query.pb.gw.go).

- [ ] **Step 2: Verify generated code**

Run: `ls x/sponsorship/types/*.pb.go`
Expected: at least state.pb.go, tx.pb.go, tx_grpc.pb.go, genesis.pb.go, query.pb.go, query_grpc.pb.go, query.pb.gw.go.

Run: `grep -n "type BountyOrder struct" x/sponsorship/types/state.pb.go`
Expected: a match. If missing, proto-gen didn't pick up state.proto — confirm `proto/zerone/sponsorship/v1/` is included by buf's workspace config.

- [ ] **Step 3: Verify package compiles**

Run: `go build ./x/sponsorship/types/`
Expected: PASS.

### Task 2.6: Commit Phase 2

- [ ] **Step 1: Commit**

```bash
git add proto/zerone/sponsorship/v1/ x/sponsorship/types/*.pb.go x/sponsorship/types/*.pb.gw.go
git commit -m "$(cat <<'EOF'
proto(sponsorship): BountyOrder + Fulfillment + 3 messages

State: BountyOrder (sponsor commitment with escrow), BountyFulfillment
(per-payout record), Params. Messages: CreateBountyOrder (escrow lock),
FulfillBounty (verification-gated payout), CancelBountyOrder (refund).
Query stubs for params and bounty order(s).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 3: Types Glue — Codec, Validation, Genesis

**Goal:** Wire the proto-derived types into the SDK's codec and validation interfaces.

### Task 3.1: Create types/codec.go

**Files:** Create `x/sponsorship/types/codec.go`

- [ ] **Step 1: Write the file**

```go
package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateBountyOrder{}, "zerone_sponsorship/CreateBountyOrder", nil)
	cdc.RegisterConcrete(&MsgFulfillBounty{}, "zerone_sponsorship/FulfillBounty", nil)
	cdc.RegisterConcrete(&MsgCancelBountyOrder{}, "zerone_sponsorship/CancelBountyOrder", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateBountyOrder{},
		&MsgFulfillBounty{},
		&MsgCancelBountyOrder{},
	)
}
```

### Task 3.2: Create types/types.go (GetSigners, ValidateBasic, DefaultGenesis)

**Files:** Create `x/sponsorship/types/types.go`

- [ ] **Step 1: Write the file**

```go
package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---- Params ----

func DefaultParams() *Params {
	return &Params{
		MinTargetCount:               1,
		MinDurationBlocks:            100,
		MaxActiveBountiesPerSponsor:  16,
	}
}

func (p *Params) Validate() error {
	if p.MinTargetCount == 0 {
		return fmt.Errorf("min_target_count must be positive")
	}
	if p.MinDurationBlocks == 0 {
		return fmt.Errorf("min_duration_blocks must be positive")
	}
	if p.MaxActiveBountiesPerSponsor == 0 {
		return fmt.Errorf("max_active_bounties_per_sponsor must be positive")
	}
	return nil
}

// ---- Genesis ----

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		Orders:       []*BountyOrder{},
		Fulfillments: []*BountyFulfillment{},
		NextBountyId: 1,
	}
}

func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		if err := gs.Params.Validate(); err != nil {
			return fmt.Errorf("invalid params: %w", err)
		}
	}
	seenOrders := make(map[string]bool, len(gs.Orders))
	for _, o := range gs.Orders {
		if seenOrders[o.Id] {
			return fmt.Errorf("duplicate bounty order id: %s", o.Id)
		}
		seenOrders[o.Id] = true
	}
	return nil
}

// ---- MsgCreateBountyOrder ----

func (msg *MsgCreateBountyOrder) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sponsor)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCreateBountyOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sponsor); err != nil {
		return fmt.Errorf("invalid sponsor address: %w", err)
	}
	if msg.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	price := new(big.Int)
	if _, ok := price.SetString(msg.PricePerArtifact, 10); !ok || price.Sign() <= 0 {
		return fmt.Errorf("price_per_artifact must be a positive integer in uzrn")
	}
	if msg.TargetCount == 0 {
		return fmt.Errorf("target_count must be positive")
	}
	if msg.DurationBlocks == 0 {
		return fmt.Errorf("duration_blocks must be positive")
	}
	return nil
}

// ---- MsgFulfillBounty ----

func (msg *MsgFulfillBounty) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Caller)
	return []sdk.AccAddress{addr}
}

func (msg *MsgFulfillBounty) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if msg.BountyId == "" {
		return fmt.Errorf("bounty_id cannot be empty")
	}
	if msg.FactId == "" {
		return fmt.Errorf("fact_id cannot be empty")
	}
	return nil
}

// ---- MsgCancelBountyOrder ----

func (msg *MsgCancelBountyOrder) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sponsor)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCancelBountyOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sponsor); err != nil {
		return fmt.Errorf("invalid sponsor address: %w", err)
	}
	if msg.BountyId == "" {
		return fmt.Errorf("bounty_id cannot be empty")
	}
	return nil
}
```

### Task 3.3: Create types/expected_keepers.go

**Files:** Create `x/sponsorship/types/expected_keepers.go`

- [ ] **Step 1: Write the file**

```go
package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// KnowledgeKeeper is the read-only view this module needs of x/knowledge.
// Sponsorship never modifies facts; it only checks their status, domain,
// submission block, and submitter before paying out.
type KnowledgeKeeper interface {
	GetFact(ctx context.Context, factID string) (*knowledgetypes.Fact, bool)
}

// BankKeeper handles escrow movement: sponsor → module account on create,
// module account → worker on fulfill, module account → sponsor on cancel.
// No mint, no burn — every coin flowing through this module already
// existed when the sponsor escrowed it.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}
```

### Task 3.4: Create types/types_test.go

**Files:** Create `x/sponsorship/types/types_test.go`

- [ ] **Step 1: Write the file**

```go
package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("zrn", "zrnpub")
	cfg.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

func mkAddr(seed string) string {
	b := make([]byte, 20)
	copy(b, []byte(seed))
	return sdk.AccAddress(b).String()
}

func TestParams_Validate(t *testing.T) {
	t.Run("default_params_valid", func(t *testing.T) {
		if err := types.DefaultParams().Validate(); err != nil {
			t.Fatalf("default params should validate, got %v", err)
		}
	})
	t.Run("zero_min_target_count_fails", func(t *testing.T) {
		p := types.DefaultParams()
		p.MinTargetCount = 0
		if err := p.Validate(); err == nil {
			t.Fatal("expected error for zero min_target_count")
		}
	})
	t.Run("zero_min_duration_fails", func(t *testing.T) {
		p := types.DefaultParams()
		p.MinDurationBlocks = 0
		if err := p.Validate(); err == nil {
			t.Fatal("expected error for zero min_duration_blocks")
		}
	})
	t.Run("zero_max_active_fails", func(t *testing.T) {
		p := types.DefaultParams()
		p.MaxActiveBountiesPerSponsor = 0
		if err := p.Validate(); err == nil {
			t.Fatal("expected error for zero max_active_bounties_per_sponsor")
		}
	})
}

func TestMsgCreateBountyOrder_ValidateBasic(t *testing.T) {
	sponsor := mkAddr("sponsor-test-aaaa12")

	t.Run("valid", func(t *testing.T) {
		msg := &types.MsgCreateBountyOrder{
			Sponsor:          sponsor,
			Domain:           "mathematics",
			PricePerArtifact: "1000000",
			TargetCount:      10,
			DurationBlocks:   1000,
		}
		if err := msg.ValidateBasic(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})

	t.Run("invalid_sponsor", func(t *testing.T) {
		msg := &types.MsgCreateBountyOrder{
			Sponsor: "not-bech32", Domain: "m", PricePerArtifact: "1", TargetCount: 1, DurationBlocks: 1,
		}
		err := msg.ValidateBasic()
		if err == nil || !strings.Contains(err.Error(), "invalid sponsor") {
			t.Fatalf("expected sponsor error, got %v", err)
		}
	})

	t.Run("empty_domain", func(t *testing.T) {
		msg := &types.MsgCreateBountyOrder{
			Sponsor: sponsor, Domain: "", PricePerArtifact: "1", TargetCount: 1, DurationBlocks: 1,
		}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for empty domain")
		}
	})

	t.Run("zero_price", func(t *testing.T) {
		msg := &types.MsgCreateBountyOrder{
			Sponsor: sponsor, Domain: "m", PricePerArtifact: "0", TargetCount: 1, DurationBlocks: 1,
		}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for zero price")
		}
	})

	t.Run("zero_target_count", func(t *testing.T) {
		msg := &types.MsgCreateBountyOrder{
			Sponsor: sponsor, Domain: "m", PricePerArtifact: "1", TargetCount: 0, DurationBlocks: 1,
		}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for zero target_count")
		}
	})

	t.Run("zero_duration", func(t *testing.T) {
		msg := &types.MsgCreateBountyOrder{
			Sponsor: sponsor, Domain: "m", PricePerArtifact: "1", TargetCount: 1, DurationBlocks: 0,
		}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for zero duration_blocks")
		}
	})
}

func TestMsgFulfillBounty_ValidateBasic(t *testing.T) {
	caller := mkAddr("caller-test-aaaaaa3")

	t.Run("valid", func(t *testing.T) {
		msg := &types.MsgFulfillBounty{Caller: caller, BountyId: "bounty-1", FactId: "fact-1"}
		if err := msg.ValidateBasic(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})
	t.Run("invalid_caller", func(t *testing.T) {
		msg := &types.MsgFulfillBounty{Caller: "not-bech32", BountyId: "b", FactId: "f"}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for invalid caller")
		}
	})
	t.Run("empty_bounty_id", func(t *testing.T) {
		msg := &types.MsgFulfillBounty{Caller: caller, BountyId: "", FactId: "f"}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for empty bounty_id")
		}
	})
	t.Run("empty_fact_id", func(t *testing.T) {
		msg := &types.MsgFulfillBounty{Caller: caller, BountyId: "b", FactId: ""}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for empty fact_id")
		}
	})
}

func TestMsgCancelBountyOrder_ValidateBasic(t *testing.T) {
	sponsor := mkAddr("sponsor-cancel-test1")

	t.Run("valid", func(t *testing.T) {
		msg := &types.MsgCancelBountyOrder{Sponsor: sponsor, BountyId: "b"}
		if err := msg.ValidateBasic(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})
	t.Run("invalid_sponsor", func(t *testing.T) {
		msg := &types.MsgCancelBountyOrder{Sponsor: "not-bech32", BountyId: "b"}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for invalid sponsor")
		}
	})
	t.Run("empty_bounty_id", func(t *testing.T) {
		msg := &types.MsgCancelBountyOrder{Sponsor: sponsor, BountyId: ""}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatal("expected error for empty bounty_id")
		}
	})
}
```

### Task 3.5: Run tests and commit

- [ ] **Step 1: Run tests**

Run: `go test ./x/sponsorship/types/ -v`
Expected: all subtests pass.

- [ ] **Step 2: Commit Phase 3**

```bash
git add x/sponsorship/types/
git commit -m "$(cat <<'EOF'
feat(sponsorship): codec, ValidateBasic, expected-keepers, default genesis

Three messages registered (CreateBountyOrder, FulfillBounty,
CancelBountyOrder). ValidateBasic refuses empty fields, malformed
bech32, non-positive prices/counts. Params include floors that prevent
trivially-small or never-running bounties.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 4: Keeper Skeleton & CRUD

**Goal:** Build the keeper's storage primitives. No msg handlers yet.

### Task 4.1: Create keeper.go

**Files:** Create `x/sponsorship/keeper/keeper.go`

- [ ] **Step 1: Write the file**

```go
package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

type Keeper struct {
	storeService    store.KVStoreService
	cdc             codec.BinaryCodec
	bankKeeper      types.BankKeeper
	knowledgeKeeper types.KnowledgeKeeper
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	bk types.BankKeeper,
	kk types.KnowledgeKeeper,
) Keeper {
	return Keeper{
		storeService:    storeService,
		cdc:             cdc,
		bankKeeper:      bk,
		knowledgeKeeper: kk,
	}
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

// ---------- Params ----------

func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("marshal params: %v", err))
	}
	_ = kv.Set(types.ParamsKey, bz)
}

func (k Keeper) GetParams(ctx context.Context) *types.Params {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var p types.Params
	if err := proto.Unmarshal(bz, &p); err != nil {
		return types.DefaultParams()
	}
	return &p
}

// ---------- Counter ----------

func (k Keeper) nextBountyID(ctx context.Context) uint64 {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyCounterKey)
	if err != nil || bz == nil {
		bz = make([]byte, 8)
	}
	n := binary.BigEndian.Uint64(bz)
	n++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, n)
	_ = kv.Set(types.BountyCounterKey, newBz)
	return n
}

// ---------- BountyOrder CRUD ----------

func (k Keeper) SetBountyOrder(ctx context.Context, o *types.BountyOrder) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(o)
	if err != nil {
		panic(fmt.Sprintf("marshal bounty: %v", err))
	}
	_ = kv.Set(types.BountyOrderKey(o.Id), bz)
}

func (k Keeper) GetBountyOrder(ctx context.Context, id string) (*types.BountyOrder, bool) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyOrderKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var o types.BountyOrder
	if err := proto.Unmarshal(bz, &o); err != nil {
		return nil, false
	}
	return &o, true
}

func (k Keeper) IterateBountyOrders(ctx context.Context, cb func(*types.BountyOrder) bool) {
	kv := k.storeService.OpenKVStore(ctx)
	iter, err := kv.Iterator(types.BountyOrderKeyPrefix, prefixEndBytes(types.BountyOrderKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var o types.BountyOrder
		if err := proto.Unmarshal(iter.Value(), &o); err != nil {
			continue
		}
		if cb(&o) {
			break
		}
	}
}

func (k Keeper) GetAllBountyOrders(ctx context.Context) []*types.BountyOrder {
	var out []*types.BountyOrder
	k.IterateBountyOrders(ctx, func(o *types.BountyOrder) bool {
		out = append(out, o)
		return false
	})
	return out
}

func (k Keeper) CountActiveBountiesBySponsor(ctx context.Context, sponsor string) uint32 {
	var n uint32
	k.IterateBountyOrders(ctx, func(o *types.BountyOrder) bool {
		if o.Status == types.BountyStatus_BOUNTY_STATUS_ACTIVE && o.Sponsor == sponsor {
			n++
		}
		return false
	})
	return n
}

// ---------- Fulfillment CRUD ----------

func (k Keeper) SetFulfillment(ctx context.Context, f *types.BountyFulfillment) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(f)
	if err != nil {
		panic(fmt.Sprintf("marshal fulfillment: %v", err))
	}
	_ = kv.Set(types.FulfillmentKey(f.BountyId, f.FactId), bz)
}

func (k Keeper) GetFulfillment(ctx context.Context, bountyID, factID string) (*types.BountyFulfillment, bool) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.FulfillmentKey(bountyID, factID))
	if err != nil || bz == nil {
		return nil, false
	}
	var f types.BountyFulfillment
	if err := proto.Unmarshal(bz, &f); err != nil {
		return nil, false
	}
	return &f, true
}

func (k Keeper) GetAllFulfillments(ctx context.Context) []*types.BountyFulfillment {
	kv := k.storeService.OpenKVStore(ctx)
	iter, err := kv.Iterator(types.FulfillmentKeyPrefix, prefixEndBytes(types.FulfillmentKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()
	var out []*types.BountyFulfillment
	for ; iter.Valid(); iter.Next() {
		var f types.BountyFulfillment
		if err := proto.Unmarshal(iter.Value(), &f); err != nil {
			continue
		}
		out = append(out, &f)
	}
	return out
}

// ---------- BeginBlocker — Expiry Sweep ----------

// ProcessBountyExpiry flips ACTIVE bounties whose end_block has elapsed
// to EXPIRED. Unlike claiming_pot's bootstrap-pot rule, sponsorship
// bounties DO expire — the sponsor's deadline is a methodological
// commitment (commitment 1) and the chain honors it. Funds remain in
// escrow on EXPIRED bounties until the sponsor calls CancelBountyOrder
// to reclaim them.
func (k Keeper) ProcessBountyExpiry(ctx context.Context, currentBlock uint64) {
	k.IterateBountyOrders(ctx, func(o *types.BountyOrder) bool {
		if o.Status == types.BountyStatus_BOUNTY_STATUS_ACTIVE && currentBlock >= o.EndBlock {
			o.Status = types.BountyStatus_BOUNTY_STATUS_EXPIRED
			k.SetBountyOrder(ctx, o)
		}
		return false
	})
}

// ---------- Genesis ----------

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	if gs.Params != nil {
		k.SetParams(ctx, gs.Params)
	}
	for _, o := range gs.Orders {
		k.SetBountyOrder(ctx, o)
	}
	for _, f := range gs.Fulfillments {
		k.SetFulfillment(ctx, f)
	}
	if gs.NextBountyId > 0 {
		kv := k.storeService.OpenKVStore(ctx)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, gs.NextBountyId-1) // counter increments before use
		_ = kv.Set(types.BountyCounterKey, buf)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:       k.GetParams(ctx),
		Orders:       k.GetAllBountyOrders(ctx),
		Fulfillments: k.GetAllFulfillments(ctx),
		NextBountyId: k.peekNextBountyID(ctx),
	}
}

func (k Keeper) peekNextBountyID(ctx context.Context) uint64 {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyCounterKey)
	if err != nil || bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz) + 1
}

// ---------- helpers ----------

func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}
```

### Task 4.2: Verify build

- [ ] **Step 1: Build**

Run: `go build ./x/sponsorship/...`
Expected: PASS.

### Task 4.3: Commit Phase 4

- [ ] **Step 1: Commit**

```bash
git add x/sponsorship/keeper/keeper.go
git commit -m "$(cat <<'EOF'
feat(sponsorship): keeper skeleton — bounty + fulfillment CRUD, expiry

ProcessBountyExpiry flips ACTIVE → EXPIRED at the deadline (unlike
bootstrap pots, sponsorship bounties honor their deadline — the
sponsor's commitment had a window, and the chain enforces it).
Escrowed funds remain locked on EXPIRED bounties until the sponsor
calls CancelBountyOrder to reclaim.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 5: MsgCreateBountyOrder Handler

**Goal:** Sponsor creates a bounty; escrow flows from sponsor account to module account; bounty is recorded with ACTIVE status.

### Task 5.1: Implement handler

**Files:** Create `x/sponsorship/keeper/msg_server.go`

- [ ] **Step 1: Write the file**

```go
package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{Keeper: k} }

var _ types.MsgServer = msgServer{}

// CreateBountyOrder escrows price_per_artifact × target_count uzrn from
// the sponsor's account to the sponsorship module account and records
// the bounty with ACTIVE status. The escrow is the chain's mechanical
// honoring of the sponsor's commitment — funds locked until the bounty
// fulfills, expires + cancels, or is canceled.
func (m msgServer) CreateBountyOrder(goCtx context.Context, msg *types.MsgCreateBountyOrder) (*types.MsgCreateBountyOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)

	// Param-floor checks.
	if msg.TargetCount < params.MinTargetCount {
		return nil, fmt.Errorf("%w: target_count %d < min %d", types.ErrInvalidConfig, msg.TargetCount, params.MinTargetCount)
	}
	if msg.DurationBlocks < params.MinDurationBlocks {
		return nil, fmt.Errorf("%w: duration_blocks %d < min %d", types.ErrInvalidConfig, msg.DurationBlocks, params.MinDurationBlocks)
	}
	if m.CountActiveBountiesBySponsor(ctx, msg.Sponsor) >= params.MaxActiveBountiesPerSponsor {
		return nil, fmt.Errorf("%w: max active bounties for sponsor reached (%d)", types.ErrInvalidConfig, params.MaxActiveBountiesPerSponsor)
	}

	// Compute total escrow = price × target_count.
	price := new(big.Int)
	if _, ok := price.SetString(msg.PricePerArtifact, 10); !ok || price.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid price_per_artifact", types.ErrInvalidConfig)
	}
	totalEscrow := new(big.Int).Mul(price, big.NewInt(int64(msg.TargetCount)))

	sponsorAddr, err := sdk.AccAddressFromBech32(msg.Sponsor)
	if err != nil {
		return nil, fmt.Errorf("invalid sponsor address: %w", err)
	}

	// Verify sponsor has the funds (defensive — bank.SendCoinsFromAccountToModule
	// will also check, but the explicit check gives a typed error).
	spendable := m.bankKeeper.SpendableCoins(ctx, sponsorAddr)
	if spendable.AmountOf("uzrn").BigInt().Cmp(totalEscrow) < 0 {
		return nil, fmt.Errorf("%w: need %s uzrn, sponsor has %s",
			types.ErrInsufficientEscrow, totalEscrow.String(), spendable.AmountOf("uzrn").String())
	}

	// Lock escrow.
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(totalEscrow)))
	if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, sponsorAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("lock escrow: %w", err)
	}

	// Build and store the bounty.
	currentBlock := uint64(ctx.BlockHeight())
	id := fmt.Sprintf("bounty-%d", m.nextBountyID(ctx))
	order := &types.BountyOrder{
		Id:               id,
		Sponsor:          msg.Sponsor,
		Domain:           msg.Domain,
		PricePerArtifact: msg.PricePerArtifact,
		TargetCount:      msg.TargetCount,
		FulfilledCount:   0,
		EscrowRemaining:  totalEscrow.String(),
		StartBlock:       currentBlock,
		EndBlock:         currentBlock + msg.DurationBlocks,
		Status:           types.BountyStatus_BOUNTY_STATUS_ACTIVE,
	}
	m.SetBountyOrder(ctx, order)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_created",
			sdk.NewAttribute("bounty_id", id),
			sdk.NewAttribute("sponsor", msg.Sponsor),
			sdk.NewAttribute("domain", msg.Domain),
			sdk.NewAttribute("price_per_artifact", msg.PricePerArtifact),
			sdk.NewAttribute("target_count", fmt.Sprintf("%d", msg.TargetCount)),
			sdk.NewAttribute("total_escrow", totalEscrow.String()),
			sdk.NewAttribute("end_block", fmt.Sprintf("%d", order.EndBlock)),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgCreateBountyOrderResponse{BountyId: id}, nil
}

// FulfillBounty and CancelBountyOrder are implemented in subsequent phases.
```

### Task 5.2: Unit tests for CreateBountyOrder

**Files:** Create `x/sponsorship/keeper/keeper_test.go`

- [ ] **Step 1: Write the test file (mocks + first test)**

```go
package keeper_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/sponsorship/keeper"
	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("zrn", "zrnpub")
	cfg.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

// ---------- Mocks ----------

type mockBankKeeper struct {
	balances       map[string]map[string]int64
	moduleBalances map[string]map[string]int64
}

func newMockBank() *mockBankKeeper {
	return &mockBankKeeper{
		balances:       map[string]map[string]int64{},
		moduleBalances: map[string]map[string]int64{},
	}
}

func (m *mockBankKeeper) setBalance(addr, denom string, amount int64) {
	if m.balances[addr] == nil {
		m.balances[addr] = map[string]int64{}
	}
	m.balances[addr][denom] = amount
}

func (m *mockBankKeeper) SpendableCoins(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	out := sdk.Coins{}
	for denom, amt := range m.balances[addr.String()] {
		if amt > 0 {
			out = out.Add(sdk.NewCoin(denom, sdk.NewInt(amt)))
		}
	}
	return out
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		if m.balances[senderAddr.String()] == nil || m.balances[senderAddr.String()][coin.Denom] < coin.Amount.Int64() {
			return errors.New("insufficient funds")
		}
		m.balances[senderAddr.String()][coin.Denom] -= coin.Amount.Int64()
		if m.moduleBalances[recipientModule] == nil {
			m.moduleBalances[recipientModule] = map[string]int64{}
		}
		m.moduleBalances[recipientModule][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		if m.moduleBalances[senderModule] == nil || m.moduleBalances[senderModule][coin.Denom] < coin.Amount.Int64() {
			return errors.New("insufficient module balance")
		}
		m.moduleBalances[senderModule][coin.Denom] -= coin.Amount.Int64()
		if m.balances[recipientAddr.String()] == nil {
			m.balances[recipientAddr.String()] = map[string]int64{}
		}
		m.balances[recipientAddr.String()][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

type mockKnowledgeKeeper struct {
	facts map[string]*knowledgetypes.Fact
}

func newMockKnowledge() *mockKnowledgeKeeper {
	return &mockKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{}}
}

func (m *mockKnowledgeKeeper) GetFact(_ context.Context, id string) (*knowledgetypes.Fact, bool) {
	f, ok := m.facts[id]
	return f, ok
}

// ---------- Setup ----------

func setup(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper, *mockKnowledgeKeeper) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("load store: %v", err)
	}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	bk := newMockBank()
	kk := newMockKnowledge()
	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(storeService, cdc, bk, kk)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000, ChainID: "zerone-test"}, false, log.NewNopLogger())
	return k, ctx, bk, kk
}

func mkAddr(seed string) sdk.AccAddress {
	b := make([]byte, 20)
	copy(b, []byte(seed))
	return sdk.AccAddress(b)
}

// ---------- CreateBountyOrder ----------

func TestCreateBountyOrder_HappyPath(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-happy-aaaaa1")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)

	resp, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor:          sponsor.String(),
		Domain:           "mathematics",
		PricePerArtifact: "1000000",
		TargetCount:      10,
		DurationBlocks:   500,
	})
	if err != nil {
		t.Fatalf("CreateBountyOrder: %v", err)
	}
	if resp.BountyId == "" {
		t.Fatal("expected non-empty bounty_id")
	}

	order, found := k.GetBountyOrder(ctx, resp.BountyId)
	if !found {
		t.Fatal("bounty not stored")
	}
	if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		t.Errorf("status: want ACTIVE, got %s", order.Status)
	}
	if order.EscrowRemaining != "10000000" {
		t.Errorf("escrow: want 10000000, got %s", order.EscrowRemaining)
	}

	// Sponsor balance debited; module balance credited.
	if bk.balances[sponsor.String()]["uzrn"] != 100_000_000-10_000_000 {
		t.Errorf("sponsor balance: want %d, got %d", 100_000_000-10_000_000, bk.balances[sponsor.String()]["uzrn"])
	}
	if bk.moduleBalances[types.ModuleName]["uzrn"] != 10_000_000 {
		t.Errorf("module balance: want 10000000, got %d", bk.moduleBalances[types.ModuleName]["uzrn"])
	}
}

func TestCreateBountyOrder_InsufficientFunds(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-poor-aaaaaa2")
	bk.setBalance(sponsor.String(), "uzrn", 1_000_000) // only 1 ZRN, want 10

	_, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor:          sponsor.String(),
		Domain:           "mathematics",
		PricePerArtifact: "1000000",
		TargetCount:      10,
		DurationBlocks:   500,
	})
	if err == nil {
		t.Fatal("expected ErrInsufficientEscrow")
	}
	if !errors.Is(err, types.ErrInsufficientEscrow) {
		t.Errorf("wrong error: %v", err)
	}
}

func TestCreateBountyOrder_BelowMinTargetCount(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	k.SetParams(ctx, &types.Params{MinTargetCount: 5, MinDurationBlocks: 100, MaxActiveBountiesPerSponsor: 16})

	sponsor := mkAddr("sponsor-target-aaaaa3")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)

	_, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "m", PricePerArtifact: "1000000",
		TargetCount: 1, DurationBlocks: 500,
	})
	if err == nil {
		t.Fatal("expected error for target_count below min")
	}
}

func TestCreateBountyOrder_BelowMinDuration(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-dur-aaaaaaaa4")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)

	_, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "m", PricePerArtifact: "1000000",
		TargetCount: 1, DurationBlocks: 10, // below default min 100
	})
	if err == nil {
		t.Fatal("expected error for duration below min")
	}
}

func TestCreateBountyOrder_MaxActivePerSponsor(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	k.SetParams(ctx, &types.Params{MinTargetCount: 1, MinDurationBlocks: 100, MaxActiveBountiesPerSponsor: 2})

	sponsor := mkAddr("sponsor-cap-aaaaaaa5")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)
	msg := &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "m", PricePerArtifact: "1000",
		TargetCount: 1, DurationBlocks: 500,
	}

	// First two succeed.
	if _, err := srv.CreateBountyOrder(ctx, msg); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := srv.CreateBountyOrder(ctx, msg); err != nil {
		t.Fatalf("second create: %v", err)
	}
	// Third must fail.
	if _, err := srv.CreateBountyOrder(ctx, msg); err == nil {
		t.Fatal("expected error on third active bounty")
	}
}

func TestCreateBountyOrder_EmitsEvent(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-event-aaaaaa6")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)

	_, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "m", PricePerArtifact: "1000",
		TargetCount: 1, DurationBlocks: 500,
	})
	if err != nil {
		t.Fatalf("CreateBountyOrder: %v", err)
	}

	var found bool
	for _, e := range ctx.EventManager().Events() {
		if e.Type == "zerone.sponsorship.bounty_created" {
			for _, attr := range e.Attributes {
				if attr.Key == "creed_commitment" && attr.Value == "20" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("expected bounty_created event with creed_commitment=20")
	}
}
```

### Task 5.3: Run tests and commit

- [ ] **Step 1: Run tests**

Run: `go test ./x/sponsorship/keeper/ -run TestCreateBountyOrder -v`
Expected: all 5 sub-tests pass.

- [ ] **Step 2: Commit Phase 5**

```bash
git add x/sponsorship/keeper/msg_server.go x/sponsorship/keeper/keeper_test.go
git commit -m "$(cat <<'EOF'
feat(sponsorship): MsgCreateBountyOrder — escrow lock + ACTIVE record

Sponsor's price × target_count uzrn flows from their account into the
sponsorship module account. Bounty stored ACTIVE with window =
[currentBlock, currentBlock + duration_blocks]. Param floors prevent
trivially-small bounties and per-sponsor state bloat. Five unit tests
cover happy-path, insufficient funds, target/duration floors, sponsor
cap, and event emission.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 6: MsgFulfillBounty Handler

**Goal:** The core. Verifies the fact meets the bounty's criteria, transfers price_per_artifact from escrow to fact submitter, updates fulfilled_count, transitions to FULFILLED when target reached.

### Task 6.1: Implement handler

**Files:** Append to `x/sponsorship/keeper/msg_server.go`

- [ ] **Step 1: Append the handler**

```go
// FulfillBounty pays the submitter of fact_id the bounty's per-artifact
// price, provided the fact meets all criteria. Anyone can call this; the
// chain does all the checks (no caller-supplied trust). The worker is
// fact.Submitter, never caller — the caller is just the messenger.
//
// Eligibility (all must hold; first failure surfaces):
//   - bounty exists
//   - bounty.Status == ACTIVE
//   - currentBlock < bounty.EndBlock
//   - fact exists in x/knowledge
//   - fact.Status == VERIFIED
//   - fact.Domain == bounty.Domain
//   - fact.SubmittedAtBlock >= bounty.StartBlock (no retroactive payouts)
//   - (bounty_id, fact_id) has no existing Fulfillment record
//
// The payout amount equals bounty.PricePerArtifact (no clipping; the
// escrow already has price × target_count locked, and fulfilled_count
// is bounded by target_count, so the math always works).
func (m msgServer) FulfillBounty(goCtx context.Context, msg *types.MsgFulfillBounty) (*types.MsgFulfillBountyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	order, found := m.GetBountyOrder(ctx, msg.BountyId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrBountyNotFound, msg.BountyId)
	}
	if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		return nil, fmt.Errorf("%w: status %s", types.ErrBountyNotActive, order.Status)
	}
	currentBlock := uint64(ctx.BlockHeight())
	if currentBlock >= order.EndBlock {
		return nil, types.ErrBountyExpired
	}
	if _, exists := m.GetFulfillment(ctx, order.Id, msg.FactId); exists {
		return nil, fmt.Errorf("%w: %s/%s", types.ErrAlreadyFulfilled, order.Id, msg.FactId)
	}

	fact, ok := m.knowledgeKeeper.GetFact(ctx, msg.FactId)
	if !ok {
		return nil, fmt.Errorf("%w: fact %s not found", types.ErrFactNotEligible, msg.FactId)
	}
	if fact.Status != knowledgetypes.FactStatus_FACT_STATUS_VERIFIED {
		return nil, fmt.Errorf("%w: fact status %s (need VERIFIED)", types.ErrFactNotEligible, fact.Status)
	}
	if fact.Domain != order.Domain {
		return nil, fmt.Errorf("%w: fact domain %q != bounty domain %q", types.ErrFactNotEligible, fact.Domain, order.Domain)
	}
	if fact.SubmittedAtBlock < order.StartBlock {
		return nil, fmt.Errorf("%w: fact submitted at block %d, bounty starts at %d (no retroactive payouts)",
			types.ErrFactNotEligible, fact.SubmittedAtBlock, order.StartBlock)
	}

	// Compute payout = price_per_artifact.
	price := new(big.Int)
	if _, ok := price.SetString(order.PricePerArtifact, 10); !ok || price.Sign() <= 0 {
		return nil, fmt.Errorf("%w: corrupt bounty price", types.ErrInvalidConfig)
	}

	// Send price from module account to fact submitter.
	workerAddr, err := sdk.AccAddressFromBech32(fact.Submitter)
	if err != nil {
		return nil, fmt.Errorf("invalid fact submitter address: %w", err)
	}
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(price)))
	if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, workerAddr, coins); err != nil {
		return nil, fmt.Errorf("payout: %w", err)
	}

	// Update bounty: fulfilled_count, escrow_remaining, status.
	order.FulfilledCount++
	escrowRemaining := new(big.Int)
	escrowRemaining.SetString(order.EscrowRemaining, 10)
	escrowRemaining.Sub(escrowRemaining, price)
	order.EscrowRemaining = escrowRemaining.String()
	bountyNowFulfilled := order.FulfilledCount >= order.TargetCount
	if bountyNowFulfilled {
		order.Status = types.BountyStatus_BOUNTY_STATUS_FULFILLED
	}
	m.SetBountyOrder(ctx, order)

	// Record fulfillment.
	m.SetFulfillment(ctx, &types.BountyFulfillment{
		BountyId:         order.Id,
		FactId:           msg.FactId,
		Worker:           fact.Submitter,
		AmountPaid:       price.String(),
		FulfilledAtBlock: currentBlock,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_fulfilled",
			sdk.NewAttribute("bounty_id", order.Id),
			sdk.NewAttribute("fact_id", msg.FactId),
			sdk.NewAttribute("worker", fact.Submitter),
			sdk.NewAttribute("amount_paid", price.String()),
			sdk.NewAttribute("fulfilled_count", fmt.Sprintf("%d", order.FulfilledCount)),
			sdk.NewAttribute("target_count", fmt.Sprintf("%d", order.TargetCount)),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgFulfillBountyResponse{
		Worker:              fact.Submitter,
		AmountPaid:          price.String(),
		BountyNowFulfilled:  bountyNowFulfilled,
	}, nil
}
```

Add the `knowledgetypes` import to the top of the file:

```go
import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/sponsorship/types"
)
```

### Task 6.2: Unit tests for FulfillBounty

**Files:** Append to `x/sponsorship/keeper/keeper_test.go`

- [ ] **Step 1: Write the tests**

```go
// ---------- FulfillBounty ----------

func makeVerifiedFact(t *testing.T, kk *mockKnowledgeKeeper, factID, domain, submitter string, submittedAt uint64) {
	t.Helper()
	kk.facts[factID] = &knowledgetypes.Fact{
		Id:               factID,
		Domain:           domain,
		Submitter:        submitter,
		SubmittedAtBlock: submittedAt,
		Status:           knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
	}
}

func createTestBounty(t *testing.T, k keeper.Keeper, srv types.MsgServer, ctx sdk.Context, bk *mockBankKeeper, sponsor sdk.AccAddress, domain, price string, target uint32, duration uint64) string {
	t.Helper()
	bk.setBalance(sponsor.String(), "uzrn", 1_000_000_000_000)
	resp, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: domain, PricePerArtifact: price,
		TargetCount: target, DurationBlocks: duration,
	})
	if err != nil {
		t.Fatalf("create bounty: %v", err)
	}
	return resp.BountyId
}

func TestFulfillBounty_HappyPath(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-fhappy-aaaa7")
	worker := mkAddr("worker-fhappy-aaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000) // submitted at current block

	caller := mkAddr("caller-aaaaaaaaaaaaaa")
	resp, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-1",
	})
	if err != nil {
		t.Fatalf("FulfillBounty: %v", err)
	}
	if resp.Worker != worker.String() {
		t.Errorf("worker: want %s, got %s", worker, resp.Worker)
	}
	if resp.AmountPaid != "1000000" {
		t.Errorf("amount: want 1000000, got %s", resp.AmountPaid)
	}
	if resp.BountyNowFulfilled {
		t.Error("bounty should not be fulfilled after 1 of 5")
	}

	// Worker received payout.
	if bk.balances[worker.String()]["uzrn"] != 1_000_000 {
		t.Errorf("worker balance: want 1000000, got %d", bk.balances[worker.String()]["uzrn"])
	}
	// Module balance decreased.
	if bk.moduleBalances[types.ModuleName]["uzrn"] != 5_000_000-1_000_000 {
		t.Errorf("module balance: want %d, got %d", 5_000_000-1_000_000, bk.moduleBalances[types.ModuleName]["uzrn"])
	}
	// Bounty bookkeeping.
	order, _ := k.GetBountyOrder(ctx, bountyID)
	if order.FulfilledCount != 1 {
		t.Errorf("fulfilled_count: want 1, got %d", order.FulfilledCount)
	}
	if order.EscrowRemaining != "4000000" {
		t.Errorf("escrow_remaining: want 4000000, got %s", order.EscrowRemaining)
	}
	// Fulfillment record exists.
	if _, exists := k.GetFulfillment(ctx, bountyID, "fact-1"); !exists {
		t.Error("fulfillment record missing")
	}
}

func TestFulfillBounty_BountyNotFound(t *testing.T) {
	k, ctx, _, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	caller := mkAddr("caller-nf-aaaaaaaaa1")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: "doesnotexist", FactId: "fact-1",
	})
	if err == nil || !errors.Is(err, types.ErrBountyNotFound) {
		t.Fatalf("expected ErrBountyNotFound, got %v", err)
	}
}

func TestFulfillBounty_FactNotFound(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-fnf-aaaaaa9")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	caller := mkAddr("caller-fnf-aaaaaaaa")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "no-such-fact",
	})
	if err == nil || !errors.Is(err, types.ErrFactNotEligible) {
		t.Fatalf("expected ErrFactNotEligible, got %v", err)
	}
}

func TestFulfillBounty_FactNotVerified(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-fnv-aaaaaa1")
	worker := mkAddr("worker-fnv-aaaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	kk.facts["fact-pending"] = &knowledgetypes.Fact{
		Id: "fact-pending", Domain: "math", Submitter: worker.String(),
		SubmittedAtBlock: 1000, Status: knowledgetypes.FactStatus_FACT_STATUS_PENDING,
	}
	caller := mkAddr("caller-fnv-aaaaaaaa1")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-pending",
	})
	if err == nil || !errors.Is(err, types.ErrFactNotEligible) {
		t.Fatalf("expected ErrFactNotEligible, got %v", err)
	}
}

func TestFulfillBounty_DomainMismatch(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-dm-aaaaaaaa1")
	worker := mkAddr("worker-dm-aaaaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	makeVerifiedFact(t, kk, "fact-bio", "biology", worker.String(), 1000) // wrong domain
	caller := mkAddr("caller-dm-aaaaaaaaa1")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-bio",
	})
	if err == nil || !errors.Is(err, types.ErrFactNotEligible) {
		t.Fatalf("expected ErrFactNotEligible (domain mismatch), got %v", err)
	}
}

func TestFulfillBounty_RetroactiveRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-retro-aaaa1")
	worker := mkAddr("worker-retro-aaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	// Fact submitted BEFORE the bounty started (current block is 1000; bounty start is 1000; this fact at 999)
	makeVerifiedFact(t, kk, "fact-retro", "math", worker.String(), 999)
	caller := mkAddr("caller-retro-aaaaa1")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-retro",
	})
	if err == nil || !errors.Is(err, types.ErrFactNotEligible) {
		t.Fatalf("expected ErrFactNotEligible (retroactive), got %v", err)
	}
}

func TestFulfillBounty_DoubleFulfillRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-double-aaaa1")
	worker := mkAddr("worker-double-aaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	caller := mkAddr("caller-double-aaaaa1")

	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-1",
	}); err != nil {
		t.Fatalf("first fulfill: %v", err)
	}
	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-1",
	}); !errors.Is(err, types.ErrAlreadyFulfilled) {
		t.Fatalf("expected ErrAlreadyFulfilled, got %v", err)
	}
}

func TestFulfillBounty_TransitionsToFulfilled(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-trans-aaaaa1")
	worker := mkAddr("worker-trans-aaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 2, 500)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	makeVerifiedFact(t, kk, "fact-2", "math", worker.String(), 1000)
	caller := mkAddr("caller-trans-aaaaaa1")

	resp1, _ := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-1"})
	if resp1.BountyNowFulfilled {
		t.Error("after 1 of 2, should not be fulfilled")
	}
	resp2, _ := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-2"})
	if !resp2.BountyNowFulfilled {
		t.Error("after 2 of 2, should be fulfilled")
	}

	order, _ := k.GetBountyOrder(ctx, bountyID)
	if order.Status != types.BountyStatus_BOUNTY_STATUS_FULFILLED {
		t.Errorf("status: want FULFILLED, got %s", order.Status)
	}
}

func TestFulfillBounty_ExpiredRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-exp-aaaaaaa1")
	worker := mkAddr("worker-exp-aaaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 100)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)

	// Advance ctx past EndBlock and re-run expiry sweep so status flips.
	ctx2 := ctx.WithBlockHeight(int64(1101))
	k.ProcessBountyExpiry(ctx2, 1101)

	caller := mkAddr("caller-exp-aaaaaaaaa1")
	_, err := srv.FulfillBounty(ctx2, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-1",
	})
	if err == nil || (!errors.Is(err, types.ErrBountyNotActive) && !errors.Is(err, types.ErrBountyExpired)) {
		t.Fatalf("expected ErrBountyNotActive or ErrBountyExpired, got %v", err)
	}
}
```

### Task 6.3: Run tests and commit

- [ ] **Step 1: Run tests**

Run: `go test ./x/sponsorship/keeper/ -run TestFulfillBounty -v`
Expected: all 8 sub-tests pass.

- [ ] **Step 2: Commit Phase 6**

```bash
git add x/sponsorship/keeper/msg_server.go x/sponsorship/keeper/keeper_test.go
git commit -m "$(cat <<'EOF'
feat(sponsorship): MsgFulfillBounty — verification-gated payout

Permissionless caller; the chain enforces all checks. Payout = bounty
price_per_artifact, from module account to fact.Submitter (NOT to
caller). Eligibility: bounty ACTIVE, fact VERIFIED in matching domain,
submitted after bounty start (no retroactive), not already fulfilled
for this bounty. Transitions to FULFILLED when target_count reached.
Eight unit tests cover happy path, all refusal paths, target transition.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 7: MsgCancelBountyOrder Handler

**Goal:** Sponsor closes an ACTIVE or EXPIRED bounty and reclaims escrow_remaining.

### Task 7.1: Implement handler

**Files:** Append to `x/sponsorship/keeper/msg_server.go`

- [ ] **Step 1: Append the handler**

```go
// CancelBountyOrder lets the sponsor reclaim the remaining escrow on an
// ACTIVE or EXPIRED bounty. FULFILLED or CANCELED bounties have no
// escrow to return (status check enforces this). Only the original
// sponsor can cancel.
func (m msgServer) CancelBountyOrder(goCtx context.Context, msg *types.MsgCancelBountyOrder) (*types.MsgCancelBountyOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	order, found := m.GetBountyOrder(ctx, msg.BountyId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrBountyNotFound, msg.BountyId)
	}
	if order.Sponsor != msg.Sponsor {
		return nil, fmt.Errorf("%w: bounty sponsor is %s, caller is %s",
			types.ErrUnauthorized, order.Sponsor, msg.Sponsor)
	}
	if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE && order.Status != types.BountyStatus_BOUNTY_STATUS_EXPIRED {
		return nil, fmt.Errorf("%w: cannot cancel a bounty in status %s", types.ErrBountyNotActive, order.Status)
	}

	remaining := new(big.Int)
	if _, ok := remaining.SetString(order.EscrowRemaining, 10); !ok {
		return nil, fmt.Errorf("%w: corrupt escrow_remaining", types.ErrInvalidConfig)
	}

	// Refund escrow_remaining to sponsor (zero-refund is permitted if
	// the bounty was fully consumed; the cancel still flips status).
	if remaining.Sign() > 0 {
		sponsorAddr, err := sdk.AccAddressFromBech32(msg.Sponsor)
		if err != nil {
			return nil, fmt.Errorf("invalid sponsor address: %w", err)
		}
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(remaining)))
		if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sponsorAddr, coins); err != nil {
			return nil, fmt.Errorf("refund: %w", err)
		}
	}

	order.Status = types.BountyStatus_BOUNTY_STATUS_CANCELED
	order.EscrowRemaining = "0"
	m.SetBountyOrder(ctx, order)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_canceled",
			sdk.NewAttribute("bounty_id", order.Id),
			sdk.NewAttribute("sponsor", msg.Sponsor),
			sdk.NewAttribute("refunded_amount", remaining.String()),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgCancelBountyOrderResponse{RefundedAmount: remaining.String()}, nil
}
```

### Task 7.2: Unit tests for CancelBountyOrder

**Files:** Append to `x/sponsorship/keeper/keeper_test.go`

- [ ] **Step 1: Write the tests**

```go
// ---------- CancelBountyOrder ----------

func TestCancelBountyOrder_HappyPath(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-cancel-h-aa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)

	preBalance := bk.balances[sponsor.String()]["uzrn"]
	resp, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{
		Sponsor: sponsor.String(), BountyId: bountyID,
	})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if resp.RefundedAmount != "5000000" {
		t.Errorf("refund: want 5000000, got %s", resp.RefundedAmount)
	}
	postBalance := bk.balances[sponsor.String()]["uzrn"]
	if postBalance-preBalance != 5_000_000 {
		t.Errorf("sponsor balance delta: want 5000000, got %d", postBalance-preBalance)
	}
	order, _ := k.GetBountyOrder(ctx, bountyID)
	if order.Status != types.BountyStatus_BOUNTY_STATUS_CANCELED {
		t.Errorf("status: want CANCELED, got %s", order.Status)
	}
	if order.EscrowRemaining != "0" {
		t.Errorf("escrow_remaining: want 0, got %s", order.EscrowRemaining)
	}
}

func TestCancelBountyOrder_NonSponsorRejected(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-real-aaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)
	other := mkAddr("not-the-sponsor-aaaa1")
	_, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{
		Sponsor: other.String(), BountyId: bountyID,
	})
	if err == nil || !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestCancelBountyOrder_AlreadyCanceledRejected(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-ac-aaaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)

	if _, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: bountyID}); err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	_, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: bountyID})
	if err == nil || !errors.Is(err, types.ErrBountyNotActive) {
		t.Fatalf("expected ErrBountyNotActive on re-cancel, got %v", err)
	}
}

func TestCancelBountyOrder_PartialFulfillmentRefund(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-partial-aaa1")
	worker := mkAddr("worker-partial-aaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	caller := mkAddr("caller-partial-aaaaa")

	// Fulfill once (1M out, 4M remaining).
	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-1"}); err != nil {
		t.Fatalf("fulfill: %v", err)
	}
	// Cancel — should refund 4M.
	resp, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: bountyID})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if resp.RefundedAmount != "4000000" {
		t.Errorf("refund: want 4000000, got %s", resp.RefundedAmount)
	}
}
```

### Task 7.3: Run tests and commit

- [ ] **Step 1: Run tests**

Run: `go test ./x/sponsorship/keeper/ -v`
Expected: all tests pass (Create + Fulfill + Cancel = ~17 sub-tests).

- [ ] **Step 2: Commit Phase 7**

```bash
git add x/sponsorship/keeper/msg_server.go x/sponsorship/keeper/keeper_test.go
git commit -m "$(cat <<'EOF'
feat(sponsorship): MsgCancelBountyOrder — sponsor exit with refund

Sponsor reclaims escrow_remaining on ACTIVE or EXPIRED bounties.
Non-sponsor refused. Status → CANCELED is terminal (re-cancel refused).
Four unit tests: happy-path full refund, non-sponsor rejection, double-
cancel rejection, partial-fulfillment partial refund.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 8: Module Wiring (module.go + grpc_query + app.go)

**Goal:** Wire the module into the app so the cross-stack harness can exercise it end-to-end.

### Task 8.1: Create module.go

**Files:** Create `x/sponsorship/module.go`

- [ ] **Step 1: Write the file**

```go
package sponsorship

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/zerone-chain/zerone/x/sponsorship/keeper"
	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
	_ appmodule.AppModule   = AppModule{}
)

type AppModuleBasic struct{ cdc codec.Codec }

func (AppModuleBasic) Name() string                                                   { return types.ModuleName }
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino)                { types.RegisterCodec(cdc) }
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry)            { types.RegisterInterfaces(reg) }
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(types.DefaultGenesis())
	if err != nil {
		panic("marshal default genesis: " + err.Error())
	}
	return bz
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return fmt.Errorf("unmarshal %s genesis: %w", types.ModuleName, err)
	}
	return gs.Validate()
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}
func (AppModuleBasic) GetTxCmd() *cobra.Command                                       { return nil }
func (AppModuleBasic) GetQueryCmd() *cobra.Command                                    { return nil }

type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{AppModuleBasic: AppModuleBasic{cdc: cdc}, keeper: k}
}

func (am AppModule) IsOnePerModuleType() {}
func (am AppModule) IsAppModule()        {}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		panic("unmarshal genesis: " + err.Error())
	}
	am.keeper.InitGenesis(ctx, &gs)
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic("marshal genesis: " + err.Error())
	}
	return bz
}

func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock runs the expiry sweep so ACTIVE bounties whose end_block
// has elapsed transition to EXPIRED. The sponsor's commitment had a
// window; the chain enforces the window.
func (am AppModule) BeginBlock(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	am.keeper.ProcessBountyExpiry(ctx, uint64(ctx.BlockHeight()))
	return nil
}

func (am AppModule) EndBlock(_ context.Context) error { return nil }
```

### Task 8.2: Create grpc_query.go

**Files:** Create `x/sponsorship/keeper/grpc_query.go`

- [ ] **Step 1: Write the file**

```go
package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer { return &queryServer{Keeper: k} }

func (q queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: q.GetParams(ctx)}, nil
}

func (q queryServer) BountyOrder(ctx context.Context, req *types.QueryBountyOrderRequest) (*types.QueryBountyOrderResponse, error) {
	o, ok := q.GetBountyOrder(ctx, req.Id)
	if !ok {
		return nil, types.ErrBountyNotFound
	}
	return &types.QueryBountyOrderResponse{Order: o}, nil
}

func (q queryServer) BountyOrders(ctx context.Context, _ *types.QueryBountyOrdersRequest) (*types.QueryBountyOrdersResponse, error) {
	return &types.QueryBountyOrdersResponse{Orders: q.GetAllBountyOrders(ctx)}, nil
}
```

### Task 8.3: Wire in app.go

**Files:** Modify `app/app.go`

- [ ] **Step 1: Add imports**

Find the block of zerone module imports (around the other `zerone*` imports near the top, ~line 220-230) and add:

```go
zeronesponsorship       "github.com/zerone-chain/zerone/x/sponsorship"
zeronesponsorshipkeeper "github.com/zerone-chain/zerone/x/sponsorship/keeper"
zeronesponsorshiptypes  "github.com/zerone-chain/zerone/x/sponsorship/types"
```

- [ ] **Step 2: Register the module basic**

Find the `ModuleBasics` block (where `zeroneclaimingpot.AppModuleBasic{}` is registered) and add `zeronesponsorship.AppModuleBasic{},` alongside it.

- [ ] **Step 3: Register the module account permission**

In the `maccPerms` map (find the line `zeronecpottypes.ModuleName: {authtypes.Minter},` at line ~373) add:

```go
zeronesponsorshiptypes.ModuleName: nil, // escrow-only, no minter/burner
```

- [ ] **Step 4: Add a keeper field on the App struct**

Find `ClaimingPotKeeper` (line ~512) and add adjacent:

```go
SponsorshipKeeper zeronesponsorshipkeeper.Keeper
```

- [ ] **Step 5: Construct the keeper**

Find the block where `app.ClaimingPotKeeper = zeronecpotkeeper.NewKeeper(...)` is constructed (around line ~1314). Add after it:

```go
app.SponsorshipKeeper = zeronesponsorshipkeeper.NewKeeper(
	runtime.NewKVStoreService(keys[zeronesponsorshiptypes.StoreKey]),
	appCodec,
	app.BankKeeper,
	app.KnowledgeKeeper, // read-only fact accessor
)
```

The `keys` map must include `zeronesponsorshiptypes.StoreKey`. Find the existing `keys :=` or `storetypes.NewKVStoreKeys(...)` block and add `zeronesponsorshiptypes.StoreKey` to it.

- [ ] **Step 6: Register the app module**

Find where `zeroneclaimingpot.NewAppModule(appCodec, app.ClaimingPotKeeper),` is registered with the module manager (around line ~1480). Add adjacent:

```go
zeronesponsorship.NewAppModule(appCodec, app.SponsorshipKeeper),
```

- [ ] **Step 7: Add to BeginBlocker order**

Find the BeginBlocker order list (around line ~1541 where `zeronecpottypes.ModuleName` is listed). Add `zeronesponsorshiptypes.ModuleName,` adjacent.

- [ ] **Step 8: Verify build**

Run: `go build ./...`
Expected: PASS.

If `app.KnowledgeKeeper` is the wrong name or the knowledge keeper has a different exported field, find the right name with: `grep -n "KnowledgeKeeper" app/app.go | head -5`

The `KnowledgeKeeper`'s `GetFact` may need to be exposed through an adapter if it returns a different type than `*knowledgetypes.Fact`. If `app.KnowledgeKeeper.GetFact` doesn't directly satisfy the `types.KnowledgeKeeper` interface, wrap it:

```go
type sponsorshipKnowledgeAdapter struct{ k zeroneknowledgekeeper.Keeper }

func (a sponsorshipKnowledgeAdapter) GetFact(ctx context.Context, id string) (*zeroneknowledgetypes.Fact, bool) {
	return a.k.GetFact(ctx, id)
}
```

Then pass `sponsorshipKnowledgeAdapter{k: app.KnowledgeKeeper}` as the fourth arg. (Check whether `app.KnowledgeKeeper.GetFact` already matches the interface — many keepers do.)

### Task 8.4: Commit Phase 8

- [ ] **Step 1: Commit**

```bash
git add x/sponsorship/module.go x/sponsorship/keeper/grpc_query.go app/app.go
git commit -m "$(cat <<'EOF'
wire(sponsorship): module.go, grpc_query, app.go integration

Module account holds no minter/burner — escrow-only, never increases
supply. BeginBlocker runs ProcessBountyExpiry so ACTIVE bounties past
their end_block transition to EXPIRED. Knowledge keeper is consulted
read-only through the expected_keepers interface.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 9: Cross-Stack Integration Test

**Goal:** Drive the full happy path against live keepers.

### Task 9.1: Add the sponsorship keeper field to the test harness

**Files:** Modify `tests/cross_stack/harness_test.go`

- [ ] **Step 1: Check whether SponsorshipKeeper is already exposed**

Run: `grep -n "SponsorshipKeeper" tests/cross_stack/harness_test.go`

If no match, add the field. Find where `ClaimingPotKeeper` is declared on the harness struct (line ~88) and add:

```go
SponsorshipKeeper zeronesponsorshipkeeper.Keeper
```

In the harness constructor (around line ~289), add:

```go
SponsorshipKeeper: app.SponsorshipKeeper,
```

Add the import:

```go
zeronesponsorshipkeeper "github.com/zerone-chain/zerone/x/sponsorship/keeper"
```

### Task 9.2: Create the integration test

**Files:** Create `tests/cross_stack/sponsorship_test.go`

- [ ] **Step 1: Write the test file**

```go
package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmath "cosmossdk.io/math"

	zeroneapp "github.com/zerone-chain/zerone/app"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	sponsorshipkeeper "github.com/zerone-chain/zerone/x/sponsorship/keeper"
	sponsorshiptypes "github.com/zerone-chain/zerone/x/sponsorship/types"
)

// TestSponsorship_CreateFulfillEndToEnd drives the full bounty lifecycle
// against live keepers: sponsor escrows funds, a verified fact in the
// bounty's domain triggers payout to the fact's submitter, bounty's
// fulfilled_count advances, escrow_remaining decreases.
//
// This is the MVP's proof that external value can flow into ZERONE via
// the sponsorship pathway — the sponsor's funds left their account and
// the worker's account grew, gated entirely by chain-side verification.
func TestSponsorship_CreateFulfillEndToEnd(t *testing.T) {
	h := NewTestHarness(t)

	// Sponsor account, funded.
	sponsor := sdk.AccAddress(make([]byte, 20))
	for i := range sponsor {
		sponsor[i] = byte(0xA0 + i)
	}
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(100_000_000)))))

	// Worker account.
	worker := sdk.AccAddress(make([]byte, 20))
	for i := range worker {
		worker[i] = byte(0xB0 + i)
	}

	srv := sponsorshipkeeper.NewMsgServerImpl(h.SponsorshipKeeper)

	// Create bounty.
	createResp, err := srv.CreateBountyOrder(h.Ctx, &sponsorshiptypes.MsgCreateBountyOrder{
		Sponsor:          sponsor.String(),
		Domain:           "mathematics",
		PricePerArtifact: "1000000",
		TargetCount:      3,
		DurationBlocks:   1000,
	})
	require.NoError(t, err)
	bountyID := createResp.BountyId

	// Sponsor balance debited.
	sponsorBalance := h.App.BankKeeper.GetBalance(h.Ctx, sponsor, zeroneapp.BondDenom)
	require.Equal(t, sdkmath.NewInt(100_000_000-3_000_000), sponsorBalance.Amount,
		"sponsor balance should be debited by total escrow (price × target)")

	// Seed a verified fact via the knowledge keeper. We construct it
	// directly to avoid driving the full verification round in this
	// test; that path is exercised in knowledge-module tests.
	currentBlock := uint64(h.Ctx.BlockHeight())
	fact := &knowledgetypes.Fact{
		Id:               "test-fact-sponsorship-1",
		Domain:           "mathematics",
		Submitter:        worker.String(),
		SubmittedAtBlock: currentBlock,
		Status:           knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		Content:          "Test fact for sponsorship MVP",
	}
	h.KnowledgeKeeper.SetFact(h.Ctx, fact)

	// Fulfill the bounty with this fact. Anyone can be the caller.
	fulfillResp, err := srv.FulfillBounty(h.Ctx, &sponsorshiptypes.MsgFulfillBounty{
		Caller:   sponsor.String(), // doesn't matter — chain reads worker from fact.Submitter
		BountyId: bountyID,
		FactId:   fact.Id,
	})
	require.NoError(t, err)
	require.Equal(t, worker.String(), fulfillResp.Worker)
	require.Equal(t, "1000000", fulfillResp.AmountPaid)
	require.False(t, fulfillResp.BountyNowFulfilled, "1 of 3 — not fulfilled yet")

	// Worker received payout.
	workerBalance := h.App.BankKeeper.GetBalance(h.Ctx, worker, zeroneapp.BondDenom)
	require.Equal(t, sdkmath.NewInt(1_000_000), workerBalance.Amount,
		"worker balance should equal one per-artifact price after one fulfillment")

	// Bounty bookkeeping.
	order, found := h.SponsorshipKeeper.GetBountyOrder(h.Ctx, bountyID)
	require.True(t, found)
	require.Equal(t, uint32(1), order.FulfilledCount)
	require.Equal(t, "2000000", order.EscrowRemaining)
	require.Equal(t, sponsorshiptypes.BountyStatus_BOUNTY_STATUS_ACTIVE, order.Status)
}

// TestSponsorship_CancelRefundsRemaining confirms that the sponsor can
// reclaim unspent escrow at any time.
func TestSponsorship_CancelRefundsRemaining(t *testing.T) {
	h := NewTestHarness(t)

	sponsor := sdk.AccAddress(make([]byte, 20))
	for i := range sponsor {
		sponsor[i] = byte(0xC0 + i)
	}
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(100_000_000)))))

	srv := sponsorshipkeeper.NewMsgServerImpl(h.SponsorshipKeeper)

	createResp, err := srv.CreateBountyOrder(h.Ctx, &sponsorshiptypes.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "mathematics", PricePerArtifact: "1000000",
		TargetCount: 5, DurationBlocks: 1000,
	})
	require.NoError(t, err)

	cancelResp, err := srv.CancelBountyOrder(h.Ctx, &sponsorshiptypes.MsgCancelBountyOrder{
		Sponsor: sponsor.String(), BountyId: createResp.BountyId,
	})
	require.NoError(t, err)
	require.Equal(t, "5000000", cancelResp.RefundedAmount)

	// Sponsor balance restored.
	sponsorBalance := h.App.BankKeeper.GetBalance(h.Ctx, sponsor, zeroneapp.BondDenom)
	require.Equal(t, sdkmath.NewInt(100_000_000), sponsorBalance.Amount,
		"after cancel, sponsor balance should equal initial fund")
}

// TestSponsorship_NoMintingHappens binds the invariant: sponsorship is
// supply circulation, not emission. Compare total uzrn supply before
// and after a full bounty lifecycle (create + fulfill + cancel).
func TestSponsorship_NoMintingHappens(t *testing.T) {
	h := NewTestHarness(t)

	sponsor := sdk.AccAddress(make([]byte, 20))
	for i := range sponsor {
		sponsor[i] = byte(0xD0 + i)
	}
	worker := sdk.AccAddress(make([]byte, 20))
	for i := range worker {
		worker[i] = byte(0xE0 + i)
	}
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(50_000_000)))))

	preSupply := h.App.BankKeeper.GetSupply(h.Ctx, zeroneapp.BondDenom)

	srv := sponsorshipkeeper.NewMsgServerImpl(h.SponsorshipKeeper)
	createResp, err := srv.CreateBountyOrder(h.Ctx, &sponsorshiptypes.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "mathematics", PricePerArtifact: "1000000",
		TargetCount: 2, DurationBlocks: 1000,
	})
	require.NoError(t, err)

	currentBlock := uint64(h.Ctx.BlockHeight())
	h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "no-mint-fact", Domain: "mathematics", Submitter: worker.String(),
		SubmittedAtBlock: currentBlock, Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
	})
	_, err = srv.FulfillBounty(h.Ctx, &sponsorshiptypes.MsgFulfillBounty{
		Caller: sponsor.String(), BountyId: createResp.BountyId, FactId: "no-mint-fact",
	})
	require.NoError(t, err)

	_, err = srv.CancelBountyOrder(h.Ctx, &sponsorshiptypes.MsgCancelBountyOrder{
		Sponsor: sponsor.String(), BountyId: createResp.BountyId,
	})
	require.NoError(t, err)

	postSupply := h.App.BankKeeper.GetSupply(h.Ctx, zeroneapp.BondDenom)
	require.Equal(t, preSupply.Amount, postSupply.Amount,
		"sponsorship must not mint — total uzrn supply must be unchanged across create+fulfill+cancel")
}
```

### Task 9.3: Run tests and commit

- [ ] **Step 1: Run integration tests**

Run: `go test ./tests/cross_stack/ -run TestSponsorship -v`
Expected: all 3 sub-tests pass.

If `h.KnowledgeKeeper.SetFact` doesn't exist (e.g., the keeper's setter is named differently), `grep -nE "func .*Keeper.*SetFact|SetFact" /Users/yuai/Desktop/zerone/x/knowledge/keeper/*.go` and use the actual name. The intent is "store a Fact directly bypassing the verification round."

- [ ] **Step 2: Commit Phase 9**

```bash
git add tests/cross_stack/harness_test.go tests/cross_stack/sponsorship_test.go
git commit -m "$(cat <<'EOF'
test(cross_stack): sponsorship end-to-end against live keepers

Three integration tests: full create+fulfill flow with supply check;
sponsor cancel returning full refund; supply-invariance binding (no
minting across a complete bounty lifecycle — sponsorship circulates
existing supply only).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 10: Doctrine Binding

**Goal:** Bind the new module's voice in `docs/EVENTS.md` and extend the graph layer of `TRUTH_SEEKING.md`.

### Task 10.1: Document events

**Files:** Modify `docs/EVENTS.md`

- [ ] **Step 1: Add a new section**

Find an alphabetically-appropriate location (e.g., after `schedule`, before `staking`). Add:

```markdown
## sponsorship

### zerone.sponsorship.bounty_created
A sponsor escrowed external value against a typed bounty for verified work in a specific domain. Commitment 20 (issuance follows participation) extended: external value enters the chain conditional on future verified participation.
- `bounty_id` -- assigned bounty identifier
- `sponsor` -- bech32 of the sponsor
- `domain` -- target epistemic domain
- `price_per_artifact` -- payout per fulfillment (uzrn)
- `target_count` -- maximum fulfillments
- `total_escrow` -- price × target_count (uzrn)
- `end_block` -- bounty window deadline
- `creed_commitment` -- "20"

### zerone.sponsorship.bounty_fulfilled
A verified fact in the bounty's domain triggered payout from escrow to the fact's submitter. The chain enforced eligibility (status, domain, window, no double-fulfill); the sponsor had no editorial role in the payout decision.
- `bounty_id`
- `fact_id`
- `worker` -- fact.Submitter (the recipient)
- `amount_paid` -- price_per_artifact (uzrn)
- `fulfilled_count` -- new count after this payout
- `target_count`
- `creed_commitment` -- "20"

### zerone.sponsorship.bounty_canceled
A sponsor reclaimed the remaining escrow of an ACTIVE or EXPIRED bounty.
- `bounty_id`
- `sponsor`
- `refunded_amount` -- remaining escrow returned (uzrn; may be "0")
- `creed_commitment` -- "20"

---
```

### Task 10.2: Extend TRUTH_SEEKING.md

**Files:** Modify `docs/TRUTH_SEEKING.md`

- [ ] **Step 1: Append a new subsection to commitment 12**

Find commitment 12 (`### 12. The chain pays for its own audit`). Append to its **Echoes** line (or add a new subsection just before Echoes):

```markdown
**Extended scope:** `x/sponsorship` widens this commitment from "the chain pays for its own audit" to "the chain mediates external payment for the work it audits." A sponsor commits external value (escrowed uzrn) against a typed bounty — domain, per-artifact price, target count, window — and the chain pays out from escrow to fact submitters whose verified facts meet the criteria. The audit pathway is unchanged; only the funding source widens. The sponsor cannot buy verification; they fund work that the chain's panel verifies independently (commitment 8). Bound by `TestSponsorship_CreateFulfillEndToEnd` and `TestSponsorship_NoMintingHappens`.
```

- [ ] **Step 2: Bump .creed-hash**

Run: `make creed-check 2>&1 | grep -E "Actual|computed"`
Capture the new actual hash from the output.

Write the new hash to `.creed-hash` (overwriting the previous line):

```bash
# Replace HASH_HERE with the value from the previous step
echo "HASH_HERE" > .creed-hash
```

Run: `make creed-check`
Expected: `creed hash check ok (...)` with the new hash.

### Task 10.3: Commit Phase 10

- [ ] **Step 1: Commit**

```bash
git add docs/EVENTS.md docs/TRUTH_SEEKING.md .creed-hash
git commit -m "$(cat <<'EOF'
docs(creed): bind x/sponsorship across voice + graph layers

Voice: docs/EVENTS.md documents bounty_created, bounty_fulfilled,
bounty_canceled (all with creed_commitment="20"). Graph: TRUTH_SEEKING.md
commitment 12 extended scope — chain mediates external payment for the
work it audits; sponsors fund, the chain verifies. Hash bumped; on-chain
PinnedCreed advance is a separate governance-LIP step.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Sequencing & Parallelism

| Phase | Depends on | Can run in parallel with |
|-------|-----------|--------------------------|
| 1 — Scaffolding | — | — |
| 2 — Proto | 1 | — |
| 3 — Types glue | 2 | — |
| 4 — Keeper skeleton | 3 | — |
| 5 — Create handler | 4 | — |
| 6 — Fulfill handler | 5 | — |
| 7 — Cancel handler | 6 | — |
| 8 — Wiring | 7 | — |
| 9 — Integration test | 8 | 10 |
| 10 — Doctrine | 8 | 9 |

Recommended single-session run: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10.

---

## Out of Scope for This Plan (deferred to future iterations)

- **CLI commands.** Phase 11 candidate. The 3 messages can be exercised through gRPC for the MVP.
- **IBC-imported sponsorship.** Sponsors using non-uzrn assets (e.g., USDC over IBC). Requires an asset-pricing oracle and a per-asset escrow accountancy.
- **Reputation gating for sponsors.** Anti-abuse: only allow sponsors with prior verified-work history to create bounties above some size. Wait until we see what abuse patterns actually emerge.
- **Sponsor coalitions.** Multiple sponsors co-funding a bounty. The MVP is one-sponsor-per-bounty.
- **Domain HHI gating.** Cap any single sponsor's share of a domain's funded work. Requires a per-domain accountancy that the synthesizers can compute.
- **Per-artifact-type bounties.** MVP targets verified facts only. Future: counterexamples, traces, tools.
- **Revenue-split routing.** MVP pays full price_per_artifact to fact submitter. Future: route a share to research/dev fund (the chain takes a cut for its audit infrastructure).
- **Cross-bounty deduplication of facts.** MVP: a fact can fulfill multiple distinct bounties (one fulfillment record per (bounty, fact)). Future may add stricter rules.

---

## Acceptance for the Plan as a Whole

1. `go build ./...` and `go vet ./...` succeed.
2. `make proto-check` is green; `make proto-gen` produces no diff after running.
3. `make creed-check` is green at the new hash.
4. All sponsorship unit tests pass (`go test ./x/sponsorship/...`).
5. All sponsorship cross-stack tests pass (`go test ./tests/cross_stack/ -run TestSponsorship`).
6. A sponsor can escrow ZRN, a verified fact in their domain triggers payout to the fact's submitter, and the sponsor can cancel and reclaim what's left.
7. Total uzrn supply is unchanged across a full bounty lifecycle (binding the no-mint invariant).
8. Voice layer documented: `docs/EVENTS.md` describes the 3 events; graph layer extended: `TRUTH_SEEKING.md` commitment 12 names the extended scope and the binding tests.
