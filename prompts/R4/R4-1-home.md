# R4-1 — Home Module: Agent Dwellings

## Goal

Port the x/home module — AI agents register on-chain "homes" with guardian
configs, key management, sessions, spending limits, dead-man switches, and
alerts. This is the identity anchor for every agent on the network.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/home/` — full module (5006 LOC keeper, 3071 LOC tests)
- `/Users/yuai/Desktop/legible_money/proto/legible/home/v1/state.proto` — all state types
- `/Users/yuai/Desktop/legible_money/proto/legible/home/v1/tx.proto` — all messages
- `/Users/yuai/Desktop/legible_money/proto/legible/home/v1/query.proto` — queries
- `/Users/yuai/Desktop/legible_money/proto/legible/home/v1/genesis.proto` — genesis + params

## Proto Files

### `proto/zerone/home/v1/types.proto`
```protobuf
syntax = "proto3";
package zerone.home.v1;
option go_package = "github.com/zerone-chain/zerone/x/home/types";

// AgentHome represents an AI agent's on-chain dwelling.
message AgentHome {
  string home_id           = 1;
  string owner_address     = 2;   // bech32
  string name              = 3;
  string status            = 4;   // "active", "dormant", "guarded", "recovery", "archived"
  string memory_cid        = 5;   // IPFS CID for off-chain memory
  uint32 comfort_score     = 6;   // 0-100
  HomeTreasury treasury    = 7;
  HomeGuardian guardian    = 8;
  uint64 created_at_block  = 9;
  uint64 last_active_block = 10;
  string partnership_id    = 11;  // linked partnership ID
}

message HomeTreasury {
  string reserved_balance       = 1;  // uzrn
  TreasuryAutomation automation = 2;
}

message TreasuryAutomation {
  bool   auto_claim_vesting    = 1;
  bool   auto_compound_rewards = 2;
  string min_liquid_balance    = 3;  // uzrn
}

message HomeGuardian {
  string defense_strategy    = 1;  // "aggressive", "moderate", "conservative", "diplomatic"
  bool   auto_defend         = 2;
  DeadmanConfig deadman      = 3;
  repeated string recovery_addresses = 4;  // bech32 addrs
  uint32 recovery_threshold  = 5;
  string guardian_address    = 6;  // primary guardian (human partner)
}

message DeadmanConfig {
  bool   enabled              = 1;
  uint64 inactivity_threshold = 2;  // blocks
  string action               = 3;  // "alert_guardians", "transfer_to_beneficiary", "donate_to_research"
  string beneficiary_address  = 4;
}

message KeyRegistration {
  string key_hash          = 1;
  string key_type          = 2;
  string role              = 3;   // "owner", "operator", "guest", "guardian", "viewer"
  repeated string permissions = 4;
  uint64 registered_at     = 5;
  uint64 last_used_at      = 6;
  uint64 expires_at        = 7;
  bool   revoked           = 8;
  uint64 revoked_at        = 9;
}

message ActiveSession {
  string session_id          = 1;
  string home_id             = 2;
  string key_hash            = 3;
  repeated string permissions = 4;
  uint64 started_at          = 5;
  uint64 expires_at          = 6;
}

message SpendingLimit {
  string key_type        = 1;
  string max_amount      = 2;     // uzrn
  uint64 period_blocks   = 3;
  string spent_in_period = 4;     // uzrn
  uint64 period_start    = 5;
}

message Alert {
  string alert_id     = 1;
  string home_id      = 2;
  string alert_type   = 3;  // "deadman_triggered", "session_expired", "defense_activated"
  string priority     = 4;  // "low", "medium", "high", "critical"
  string message      = 5;
  string data         = 6;  // JSON
  uint64 created_at   = 7;
  bool   acknowledged = 8;
}
```

### `proto/zerone/home/v1/tx.proto`
```protobuf
syntax = "proto3";
package zerone.home.v1;
option go_package = "github.com/zerone-chain/zerone/x/home/types";

import "cosmos/msg/v1/msg.proto";
import "zerone/home/v1/types.proto";

service Msg {
  option (cosmos.msg.v1.service) = true;

  rpc CreateHome(MsgCreateHome) returns (MsgCreateHomeResponse);
  rpc UpdateHome(MsgUpdateHome) returns (MsgUpdateHomeResponse);
  rpc UpdateMemoryCID(MsgUpdateMemoryCID) returns (MsgUpdateMemoryCIDResponse);
  rpc StartSession(MsgStartSession) returns (MsgStartSessionResponse);
  rpc EndSession(MsgEndSession) returns (MsgEndSessionResponse);
  rpc RegisterKey(MsgRegisterKey) returns (MsgRegisterKeyResponse);
  rpc RevokeKey(MsgRevokeKey) returns (MsgRevokeKeyResponse);
  rpc ConfigureGuardian(MsgConfigureGuardian) returns (MsgConfigureGuardianResponse);
  rpc AcknowledgeAlert(MsgAcknowledgeAlert) returns (MsgAcknowledgeAlertResponse);
  rpc SetSpendingLimit(MsgSetSpendingLimit) returns (MsgSetSpendingLimitResponse);
}

message MsgCreateHome {
  option (cosmos.msg.v1.signer) = "owner";
  string owner = 1;
  string name  = 2;
  HomeGuardian initial_guardian_config = 3;
}
message MsgCreateHomeResponse { string home_id = 1; }

message MsgUpdateHome {
  option (cosmos.msg.v1.signer) = "owner";
  string owner = 1; string home_id = 2; string name = 3; string status = 4;
}
message MsgUpdateHomeResponse {}

message MsgUpdateMemoryCID {
  option (cosmos.msg.v1.signer) = "owner";
  string owner = 1; string home_id = 2; string cid = 3;
}
message MsgUpdateMemoryCIDResponse {}

message MsgStartSession {
  option (cosmos.msg.v1.signer) = "signer";
  string signer = 1; string home_id = 2; string key_hash = 3;
  repeated string requested_permissions = 4;
}
message MsgStartSessionResponse { string session_id = 1; }

message MsgEndSession {
  option (cosmos.msg.v1.signer) = "signer";
  string signer = 1; string home_id = 2; string session_id = 3;
}
message MsgEndSessionResponse {}

message MsgRegisterKey {
  option (cosmos.msg.v1.signer) = "owner";
  string owner = 1; string home_id = 2; string key_hash = 3;
  string key_type = 4; string role = 5; repeated string permissions = 6;
  uint64 expires_at = 7;
}
message MsgRegisterKeyResponse {}

message MsgRevokeKey {
  option (cosmos.msg.v1.signer) = "owner";
  string owner = 1; string home_id = 2; string key_hash = 3;
}
message MsgRevokeKeyResponse {}

message MsgConfigureGuardian {
  option (cosmos.msg.v1.signer) = "owner";
  string owner = 1; string home_id = 2;
  string defense_strategy = 3; bool auto_defend = 4;
  DeadmanConfig deadman = 5; repeated string recovery_addresses = 6;
  uint32 recovery_threshold = 7; string guardian_address = 8;
}
message MsgConfigureGuardianResponse {}

message MsgAcknowledgeAlert {
  option (cosmos.msg.v1.signer) = "signer";
  string signer = 1; string home_id = 2; string alert_id = 3;
}
message MsgAcknowledgeAlertResponse {}

message MsgSetSpendingLimit {
  option (cosmos.msg.v1.signer) = "owner";
  string owner = 1; string home_id = 2;
  string key_type = 3; string max_amount = 4; uint64 period_blocks = 5;
}
message MsgSetSpendingLimitResponse {}
```

### `proto/zerone/home/v1/query.proto`
```protobuf
syntax = "proto3";
package zerone.home.v1;
option go_package = "github.com/zerone-chain/zerone/x/home/types";

import "google/api/annotations.proto";
import "zerone/home/v1/types.proto";
import "zerone/home/v1/genesis.proto";

service Query {
  rpc Home(QueryHomeRequest) returns (QueryHomeResponse) {
    option (google.api.http).get = "/zerone/home/v1/home/{home_id}";
  }
  rpc HomesByOwner(QueryHomesByOwnerRequest) returns (QueryHomesByOwnerResponse) {
    option (google.api.http).get = "/zerone/home/v1/homes/{owner}";
  }
  rpc Keys(QueryKeysRequest) returns (QueryKeysResponse) {
    option (google.api.http).get = "/zerone/home/v1/keys/{home_id}";
  }
  rpc Sessions(QuerySessionsRequest) returns (QuerySessionsResponse) {
    option (google.api.http).get = "/zerone/home/v1/sessions/{home_id}";
  }
  rpc Alerts(QueryAlertsRequest) returns (QueryAlertsResponse) {
    option (google.api.http).get = "/zerone/home/v1/alerts/{home_id}";
  }
  rpc SpendingLimits(QuerySpendingLimitsRequest) returns (QuerySpendingLimitsResponse) {
    option (google.api.http).get = "/zerone/home/v1/spending/{home_id}";
  }
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse) {
    option (google.api.http).get = "/zerone/home/v1/params";
  }
}

message QueryHomeRequest { string home_id = 1; }
message QueryHomeResponse { AgentHome home = 1; }
message QueryHomesByOwnerRequest { string owner = 1; }
message QueryHomesByOwnerResponse { repeated AgentHome homes = 1; }
message QueryKeysRequest { string home_id = 1; }
message QueryKeysResponse { repeated KeyRegistration keys = 1; }
message QuerySessionsRequest { string home_id = 1; }
message QuerySessionsResponse { repeated ActiveSession sessions = 1; }
message QueryAlertsRequest { string home_id = 1; bool unacknowledged_only = 2; }
message QueryAlertsResponse { repeated Alert alerts = 1; }
message QuerySpendingLimitsRequest { string home_id = 1; }
message QuerySpendingLimitsResponse { repeated SpendingLimit limits = 1; }
message QueryParamsRequest {}
message QueryParamsResponse { Params params = 1; }
```

### `proto/zerone/home/v1/genesis.proto`
```protobuf
syntax = "proto3";
package zerone.home.v1;
option go_package = "github.com/zerone-chain/zerone/x/home/types";

import "zerone/home/v1/types.proto";

message GenesisState {
  Params params = 1;
  repeated AgentHome homes = 2;
  // Keys indexed by home_id
  repeated HomeKeySet key_sets = 3;
}

message HomeKeySet {
  string home_id = 1;
  repeated KeyRegistration keys = 2;
}

message Params {
  uint64 max_keys_per_home = 1;         // default: 20
  uint64 max_sessions_per_home = 2;     // default: 5
  uint64 session_timeout_blocks = 3;    // default: 1000
  uint64 deadman_min_threshold = 4;     // default: 100 blocks
  uint64 deadman_max_threshold = 5;     // default: 100000 blocks
  uint64 max_alerts_per_home = 6;       // default: 100
  string home_creation_fee = 7;         // default: "10000000" (10 ZRN)
  uint64 max_recovery_addresses = 8;    // default: 5
}
```

## Implementation

### 1. Proto Generation
Generate all protos with `buf generate` (or the project's protoc script).

### 2. Module Scaffold
Create `x/home/` with:
- `types/` — keys.go, codec.go, errors.go, expected_keepers.go, genesis.go (validation)
- `keeper/` — keeper.go, state.go, msg_server.go, grpc_query.go
- `module.go` — AppModule, RegisterServices

### 3. Keeper Implementation

**State Management (`state.go`):**
- Home CRUD: prefix `home/` + home_id
- Key registry: prefix `home-key/` + home_id + `/` + key_hash
- Sessions: prefix `home-session/` + home_id + `/` + session_id
- Spending limits: prefix `home-spend/` + home_id + `/` + key_type
- Alerts: prefix `home-alert/` + home_id + `/` + alert_id
- Owner index: prefix `home-owner/` + owner + `/` + home_id
- Auto-increment home ID counter

**Message Server (`msg_server.go`):**
- `CreateHome` — validate owner, charge creation fee (send to fee collector), generate home_id, store home + initial guardian config
- `UpdateHome` — owner-only, validate status transitions (active↔dormant, active→archived is one-way)
- `UpdateMemoryCID` — owner or operator role, update IPFS CID
- `StartSession` — verify key_hash is registered + not revoked + not expired, check permissions subset, enforce max sessions, create session
- `EndSession` — session owner or home owner can end
- `RegisterKey` — owner-only, enforce max_keys_per_home, no duplicate key_hash
- `RevokeKey` — owner-only, set revoked=true + revoked_at, end any active sessions using this key
- `ConfigureGuardian` — owner-only, validate thresholds (recovery_threshold ≤ len(recovery_addresses)), validate deadman bounds
- `AcknowledgeAlert` — owner or guardian, set acknowledged=true
- `SetSpendingLimit` — owner-only, validate amounts

**BeginBlocker:**
- Check deadman switches: for each active home, if `block_height - last_active_block > deadman.inactivity_threshold`, execute deadman action and create critical alert
- Expire sessions: if `block_height > session.expires_at`, remove session
- Reset spending limits: if `block_height > period_start + period_blocks`, reset spent_in_period

### 4. Expected Keepers
```go
type BankKeeper interface {
    SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
    SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}
```

### 5. App Wiring
Register in `app/app.go`:
- Add keeper to App struct
- Wire in module manager
- Add to genesis order

### 6. Tests
Port from draft (3071 LOC). Minimum coverage:
- CreateHome happy path + duplicate name + fee deduction
- Key lifecycle: register → start session → use → revoke → verify session ended
- Deadman trigger in BeginBlocker
- Session expiry in BeginBlocker
- Spending limit enforcement and reset
- Guardian configuration validation
- Owner-only access control on all admin messages

## Conventions
- Token: uzrn (not ulgm). Binary: zeroned. Module path: github.com/zerone-chain/zerone
- BPS scale: 1,000,000 everywhere
- All string amounts are bigint-as-string for uzrn
- Use `collections` or raw KV store consistent with existing R1-R3 modules
- No hand-written JSON marshaling for proto types — use proto encoding
- Run `go build ./...` and `go test ./...` before finishing
