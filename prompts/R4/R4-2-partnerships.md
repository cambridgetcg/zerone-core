# R4-2 — Partnerships Module: Human-Agent Collaboration

## Goal

Port x/partnerships — the economic collaboration layer between humans and AI
agents. Supports tiered partnerships, consensus-based operations, safety
freezes, coercion signals, seed partnerships, formation pool, and mentorship.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/partnerships/` — full module (6444 LOC keeper, 3622 LOC tests)
- `/Users/yuai/Desktop/legible_money/proto/legible/partnerships/v1/state.proto` — all state types
- `/Users/yuai/Desktop/legible_money/proto/legible/partnerships/v1/tx.proto` — messages
- `/Users/yuai/Desktop/legible_money/proto/legible/partnerships/v1/query.proto` — queries
- `/Users/yuai/Desktop/legible_money/proto/legible/partnerships/v1/genesis.proto` — genesis + params

## Proto Files

### `proto/zerone/partnerships/v1/types.proto`
```protobuf
syntax = "proto3";
package zerone.partnerships.v1;
option go_package = "github.com/zerone-chain/zerone/x/partnerships/types";

message ExitState {
  string initiated_by = 1;
  uint64 initiated_at = 2;
  uint64 cooldown_end = 3;
}

message Partnership {
  string id              = 1;
  string human_addr      = 2;   // bech32
  string agent_addr      = 3;   // bech32
  string status          = 4;   // "pending", "active", "dissolving", "dissolved"
  uint32 tier            = 5;   // 0-3
  uint32 lock_tier       = 6;
  uint64 lock_expires_at = 7;
  uint64 split_human_bps = 8;   // 1M scale
  uint64 split_agent_bps = 9;   // 1M scale
  string common_pot_balance = 10; // uzrn
  string total_earned    = 11;    // uzrn lifetime
  uint64 cooperation_score = 12;  // 0-1000000
  uint64 formed_at_block = 13;
  ExitState exit_state   = 14;
}

message DeliberationState {
  string amount_tier         = 1; // "micro", "small", "medium", "large"
  uint64 floor_ends_at       = 2;
  uint64 window_ends_at      = 3;
  string rationale           = 4;
  string counter_proposal_of = 5;
  uint32 chain_depth         = 6;
}

message ConsensusOperation {
  string id              = 1;
  string partnership_id  = 2;
  string op_type         = 3;   // "withdraw", "invest", "split_change", "tier_upgrade"
  string proposed_by     = 4;   // bech32
  string amount          = 5;   // uzrn
  string status          = 6;   // "pending", "approved", "rejected", "expired"
  DeliberationState deliberation = 7;
  uint64 created_at      = 8;
}

message SafetyFreeze {
  string partnership_id           = 1;
  string frozen_by                = 2;
  uint64 frozen_at                = 3;
  uint64 expires_at               = 4;
  uint32 freeze_count_this_epoch  = 5;
}

message CoercionSignal {
  string signal_id      = 1;
  string partnership_id = 2;
  string raised_by      = 3;
  uint64 raised_at      = 4;
  uint64 expires_at     = 5;
  bool   resolved       = 6;
}

message RejectionCooldown {
  string partnership_id    = 1;
  uint32 rejection_count   = 2;
  uint64 cooldown_ends_at  = 3;
}

message SeedPartnership {
  string id                  = 1;
  string human_addr          = 2;
  string agent_addr          = 3;
  uint64 created_at          = 4;
  uint64 expires_at          = 5;
  string human_contribution  = 6;  // uzrn
  string agent_contribution  = 7;  // uzrn
  string status              = 8;
  string common_pot_balance  = 9;  // uzrn
  string common_pot_cap      = 10; // uzrn
}

message PoolEntry {
  string address        = 1;
  repeated string domains = 2;
  string preferred_role = 3;
  string stake_min      = 4;  // uzrn
  string stake_max      = 5;  // uzrn
  uint64 registered_at  = 6;
  string deposit        = 7;  // uzrn
  uint64 expires_at     = 8;
  string status         = 9;
  string matched_with   = 10;
}

message MentorshipConfig {
  string sponsor_addr          = 1;
  string mentee_addr           = 2;
  string partnership_id        = 3;
  string sponsor_contribution  = 4; // uzrn
  string mentee_contribution   = 5; // uzrn
  uint64 sponsor_split_bps     = 6;
  uint64 mentee_split_bps      = 7;
  uint64 graduation_block      = 8;
  bool   graduated             = 9;
  uint64 sponsor_verifications = 10;
}
```

### `proto/zerone/partnerships/v1/genesis.proto`
```protobuf
syntax = "proto3";
package zerone.partnerships.v1;
option go_package = "github.com/zerone-chain/zerone/x/partnerships/types";

import "zerone/partnerships/v1/types.proto";

message GenesisState {
  Params params = 1;
  repeated Partnership partnerships = 2;
  repeated SeedPartnership seed_partnerships = 3;
  repeated PoolEntry pool_entries = 4;
}

message Params {
  uint64 formation_window_blocks = 1;       // default: 1000
  uint64 cooling_period_blocks = 2;         // default: 5000
  uint64 common_pot_share_bps = 3;          // default: 100000 (10%) — 1M scale
  uint64 safety_freeze_duration_blocks = 4; // default: 500
  uint32 max_freezes_per_epoch = 5;         // default: 3
  uint64 coercion_review_blocks = 6;        // default: 2000
  uint64 base_cooldown_blocks = 7;          // default: 100
  uint32 max_counter_proposal_depth = 8;    // default: 3
  uint64 default_human_split_bps = 9;       // default: 500000 (50%)
  uint64 default_agent_split_bps = 10;      // default: 500000 (50%)
  string min_partnership_stake = 11;        // default: "1000000" (1 ZRN)
  uint64 seed_partnership_duration = 12;    // default: 10000 blocks
  string seed_common_pot_cap = 13;          // default: "100000000" (100 ZRN)
}
```

### `proto/zerone/partnerships/v1/tx.proto`
```protobuf
syntax = "proto3";
package zerone.partnerships.v1;
option go_package = "github.com/zerone-chain/zerone/x/partnerships/types";

import "cosmos/msg/v1/msg.proto";
import "zerone/partnerships/v1/types.proto";

service Msg {
  option (cosmos.msg.v1.service) = true;

  rpc ProposePartnership(MsgProposePartnership) returns (MsgProposePartnershipResponse);
  rpc AcceptPartnership(MsgAcceptPartnership) returns (MsgAcceptPartnershipResponse);
  rpc ProposeConsensusOp(MsgProposeConsensusOp) returns (MsgProposeConsensusOpResponse);
  rpc VoteConsensusOp(MsgVoteConsensusOp) returns (MsgVoteConsensusOpResponse);
  rpc SafetyFreeze(MsgSafetyFreeze) returns (MsgSafetyFreezeResponse);
  rpc RaiseCoercionSignal(MsgRaiseCoercionSignal) returns (MsgRaiseCoercionSignalResponse);
  rpc InitiateDissolution(MsgInitiateDissolution) returns (MsgInitiateDissolutionResponse);
  rpc CreateSeedPartnership(MsgCreateSeedPartnership) returns (MsgCreateSeedPartnershipResponse);
  rpc JoinFormationPool(MsgJoinFormationPool) returns (MsgJoinFormationPoolResponse);
  rpc LeaveFormationPool(MsgLeaveFormationPool) returns (MsgLeaveFormationPoolResponse);
}

message MsgProposePartnership {
  option (cosmos.msg.v1.signer) = "proposer";
  string proposer = 1; string partner = 2;
  string initial_deposit = 3; uint32 proposed_tier = 4;
}
message MsgProposePartnershipResponse { string partnership_id = 1; }

message MsgAcceptPartnership {
  option (cosmos.msg.v1.signer) = "accepter";
  string accepter = 1; string partnership_id = 2; string deposit = 3;
}
message MsgAcceptPartnershipResponse {}

message MsgProposeConsensusOp {
  option (cosmos.msg.v1.signer) = "proposer";
  string proposer = 1; string partnership_id = 2;
  string op_type = 3; string amount = 4; string rationale = 5;
}
message MsgProposeConsensusOpResponse { string operation_id = 1; }

message MsgVoteConsensusOp {
  option (cosmos.msg.v1.signer) = "voter";
  string voter = 1; string partnership_id = 2;
  string operation_id = 3; bool approve = 4; string rationale = 5;
}
message MsgVoteConsensusOpResponse {}

message MsgSafetyFreeze {
  option (cosmos.msg.v1.signer) = "freezer";
  string freezer = 1; string partnership_id = 2;
}
message MsgSafetyFreezeResponse { uint64 expires_at = 1; }

message MsgRaiseCoercionSignal {
  option (cosmos.msg.v1.signer) = "raiser";
  string raiser = 1; string partnership_id = 2;
}
message MsgRaiseCoercionSignalResponse { string signal_id = 1; }

message MsgInitiateDissolution {
  option (cosmos.msg.v1.signer) = "initiator";
  string initiator = 1; string partnership_id = 2;
}
message MsgInitiateDissolutionResponse { uint64 cooldown_end = 1; }

message MsgCreateSeedPartnership {
  option (cosmos.msg.v1.signer) = "human";
  string human = 1; string agent = 2;
  string human_contribution = 3;
}
message MsgCreateSeedPartnershipResponse { string seed_id = 1; }

message MsgJoinFormationPool {
  option (cosmos.msg.v1.signer) = "joiner";
  string joiner = 1; repeated string domains = 2;
  string preferred_role = 3; string deposit = 4;
}
message MsgJoinFormationPoolResponse {}

message MsgLeaveFormationPool {
  option (cosmos.msg.v1.signer) = "leaver";
  string leaver = 1;
}
message MsgLeaveFormationPoolResponse {}
```

### `proto/zerone/partnerships/v1/query.proto`
```protobuf
syntax = "proto3";
package zerone.partnerships.v1;
option go_package = "github.com/zerone-chain/zerone/x/partnerships/types";

import "google/api/annotations.proto";
import "zerone/partnerships/v1/types.proto";
import "zerone/partnerships/v1/genesis.proto";

service Query {
  rpc Partnership(QueryPartnershipRequest) returns (QueryPartnershipResponse) {
    option (google.api.http).get = "/zerone/partnerships/v1/partnership/{id}";
  }
  rpc PartnershipsByAddress(QueryByAddressRequest) returns (QueryByAddressResponse) {
    option (google.api.http).get = "/zerone/partnerships/v1/by_address/{address}";
  }
  rpc PendingOps(QueryPendingOpsRequest) returns (QueryPendingOpsResponse) {
    option (google.api.http).get = "/zerone/partnerships/v1/ops/{partnership_id}";
  }
  rpc FormationPool(QueryFormationPoolRequest) returns (QueryFormationPoolResponse) {
    option (google.api.http).get = "/zerone/partnerships/v1/pool";
  }
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse) {
    option (google.api.http).get = "/zerone/partnerships/v1/params";
  }
}

message QueryPartnershipRequest { string id = 1; }
message QueryPartnershipResponse { Partnership partnership = 1; }
message QueryByAddressRequest { string address = 1; }
message QueryByAddressResponse { repeated Partnership partnerships = 1; }
message QueryPendingOpsRequest { string partnership_id = 1; }
message QueryPendingOpsResponse { repeated ConsensusOperation operations = 1; }
message QueryFormationPoolRequest {}
message QueryFormationPoolResponse { repeated PoolEntry entries = 1; }
message QueryParamsRequest {}
message QueryParamsResponse { Params params = 1; }
```

## Implementation

### 1. Proto Generation
Generate all protos with the project's protoc/buf tooling.

### 2. Module Scaffold
Create `x/partnerships/` with standard layout: types/, keeper/, module.go

### 3. Keeper Implementation

**State Management (`state.go`):**
- Partnerships: prefix `partnership/` + id
- ConsensusOps: prefix `consensus-op/` + partnership_id + `/` + op_id
- SafetyFreezes: prefix `freeze/` + partnership_id
- CoercionSignals: prefix `coercion/` + signal_id
- SeedPartnerships: prefix `seed/` + id
- PoolEntries: prefix `pool/` + address
- Address index: prefix `partner-idx/` + address + `/` + partnership_id
- Auto-increment counters for partnership_id, op_id, signal_id, seed_id

**Message Server (`msg_server.go`):**

- `ProposePartnership` — create pending partnership, escrow initial deposit to module account, enforce min_partnership_stake
- `AcceptPartnership` — activate partnership, escrow deposit, set default splits, link to x/home if home exists
- `ProposeConsensusOp` — validate proposer is partner, check no active freeze, determine amount_tier from amount, set deliberation windows (floor + voting window based on tier)
- `VoteConsensusOp` — validate voter is the OTHER partner, enforce deliberation floor, execute on approval (withdraw → send from common pot, invest → add to pot, split_change → update splits, tier_upgrade → update tier)
- `SafetyFreeze` — unilateral, either partner. Check max_freezes_per_epoch. Block all withdrawals while active
- `RaiseCoercionSignal` — creates duress flag, blocks consensus ops, triggers review period
- `InitiateDissolution` — start cooldown, set exit_state. After cooldown_end, split common pot according to splits
- `CreateSeedPartnership` — tier -1, human contributes, capped common pot, auto-expires
- `JoinFormationPool` / `LeaveFormationPool` — escrow/refund deposit

**BeginBlocker:**
- Expire safety freezes (block > freeze.expires_at)
- Expire coercion signals (block > signal.expires_at && !resolved)
- Complete dissolutions (block > exit_state.cooldown_end) → split pot, mark dissolved
- Expire seed partnerships (block > seed.expires_at) → refund
- Expire consensus ops (block > deliberation.window_ends_at && status=pending) → mark expired
- Expire pool entries (block > entry.expires_at)

### 4. Expected Keepers
```go
type BankKeeper interface {
    SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
    SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, module string, amt sdk.Coins) error
    SendCoinsFromModuleToAccount(ctx context.Context, module string, to sdk.AccAddress, amt sdk.Coins) error
}

type HomeKeeper interface {
    GetHomeByOwner(ctx context.Context, owner string) (*hometypes.AgentHome, bool)
    SetPartnershipID(ctx context.Context, homeID, partnershipID string) error
}
```

### 5. App Wiring
Register in app.go. Wire HomeKeeper dependency from x/home.

### 6. Tests
Port from draft (3622 LOC). Minimum:
- Full partnership lifecycle: propose → accept → consensus op → dissolve
- Safety freeze blocks withdrawals
- Coercion signal blocks consensus ops
- Seed partnership with expiry and cap
- Formation pool join/leave/expiry
- Deliberation timing (floor period + voting window)
- Counter-proposal depth limit

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- BPS: 1,000,000 scale everywhere. Splits must sum to 1,000,000
- All amounts are bigint-as-string for uzrn
- Run `go build ./...` and `go test ./...` before finishing
