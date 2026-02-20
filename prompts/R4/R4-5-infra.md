# R4-5 — Infrastructure Modules: Schedule + Compute Pool + Discovery

## Goal

Port three smaller modules in a single session:
1. **x/schedule** — on-chain task automation (time, logic, compound conditions)
2. **x/compute_pool** — compute provider registry, credit system, SLA enforcement
3. **x/discovery** — agent profile registry and capability discovery

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/schedule/` — 2457 LOC keeper, 1417 LOC tests
- `/Users/yuai/Desktop/legible_money/x/compute_pool/` — 3349 LOC keeper, 1579 LOC tests
- `/Users/yuai/Desktop/legible_money/x/discovery/` — 1711 LOC keeper, 1057 LOC tests
- `/Users/yuai/Desktop/legible_money/proto/legible/schedule/v1/` — protos
- `/Users/yuai/Desktop/legible_money/proto/legible/compute_pool/v1/` — protos
- `/Users/yuai/Desktop/legible_money/proto/legible/discovery/v1/` — protos

---

## Module 1: Schedule

### `proto/zerone/schedule/v1/types.proto`
```protobuf
syntax = "proto3";
package zerone.schedule.v1;
option go_package = "github.com/zerone-chain/zerone/x/schedule/types";

message TimeCondition {
  string type              = 1;  // "at_block", "every_n_blocks"
  uint64 execute_at_block  = 2;
  uint64 interval_blocks   = 3;
  uint64 start_block       = 4;
  uint64 end_block         = 5;
}

message LogicCondition {
  string type          = 1;  // "state_compare"
  string state_key     = 2;  // KV store key to watch
  string comparator    = 3;  // "eq", "gt", "lt", "gte", "lte"
  string compare_value = 4;
}

message CompoundCondition {
  string operator = 1;  // "and", "or"
  repeated ScheduleCondition conditions = 2;
}

message ScheduleCondition {
  TimeCondition time_condition         = 1;
  LogicCondition logic_condition       = 2;
  CompoundCondition compound_condition = 3;
}

message ScheduleProcess {
  string id                  = 1;
  string creator             = 2;   // bech32
  ScheduleCondition condition = 3;
  string status              = 4;   // "active", "paused", "completed", "cancelled"
  uint64 execution_count     = 5;
  uint64 max_executions      = 6;   // 0 = unlimited
  string remaining_fee       = 7;   // uzrn prepaid
  string fee_per_execution   = 8;   // uzrn
  string target_address      = 9;   // contract or account
  string call_data           = 10;  // hex-encoded
  string transfer_value      = 11;  // uzrn per execution
  string linked_entity_type  = 12;  // "bvm_contract", "home", etc.
  string linked_entity_id    = 13;
  uint64 created_at_block    = 14;
  uint64 expires_at_block    = 15;
  uint64 next_execute_at     = 16;
}
```

### `proto/zerone/schedule/v1/genesis.proto`
```protobuf
message Params {
  uint32 max_active_per_account = 1;  // default: 20
  uint64 max_gas_per_block = 2;       // default: 50000000
  uint64 min_interval_blocks = 3;     // default: 10
  string min_fee_per_execution = 4;   // default: "10000" (0.01 ZRN)
  uint64 max_compound_depth = 5;      // default: 3
}
```

### Messages
```protobuf
service Msg {
  rpc CreateSchedule(MsgCreateSchedule) returns (MsgCreateScheduleResponse);
  rpc PauseSchedule(MsgPauseSchedule) returns (MsgPauseScheduleResponse);
  rpc ResumeSchedule(MsgResumeSchedule) returns (MsgResumeScheduleResponse);
  rpc CancelSchedule(MsgCancelSchedule) returns (MsgCancelScheduleResponse);
  rpc FundSchedule(MsgFundSchedule) returns (MsgFundScheduleResponse);
}
```

### Implementation
- BeginBlocker evaluates all active schedules: time conditions check block height, logic conditions check KV store
- Deduct fee_per_execution from remaining_fee each run
- Cancel when remaining_fee < fee_per_execution or max_executions reached
- Compound conditions: evaluate recursively with depth limit

---

## Module 2: Compute Pool

### `proto/zerone/compute_pool/v1/types.proto`
```protobuf
syntax = "proto3";
package zerone.compute_pool.v1;
option go_package = "github.com/zerone-chain/zerone/x/compute_pool/types";

message ComputeProvider {
  string address        = 1;
  string service_type   = 2;   // "inference", "verification", "storage"
  string endpoint       = 3;   // URL
  string price_per_cu   = 4;   // uzrn per compute unit
  string stake          = 5;   // uzrn staked
  string status         = 6;   // "active", "unbonding", "jailed"
  uint64 registered_at  = 7;
  uint64 tasks_served   = 8;
  uint64 tasks_failed   = 9;
  uint64 avg_latency_ms = 10;
  uint64 uptime_bps     = 11;  // 1M scale
  uint64 last_heartbeat = 12;  // block height
  uint64 unbonding_at   = 13;
  string pending_price   = 14;
  uint64 price_change_at = 15;
}

message ComputeCredit {
  string validator_addr = 1;
  uint64 balance        = 2;
  uint64 earned_total   = 3;
  uint64 redeemed_total = 4;
}
```

### `proto/zerone/compute_pool/v1/genesis.proto`
```protobuf
message Params {
  uint64 compute_pool_share_bps         = 1;  // default: 100000 (10%)
  uint64 base_cu_per_verification       = 2;  // default: 100
  string min_provider_stake             = 3;  // default: "10000000" (10 ZRN)
  uint64 min_uptime_bps                 = 4;  // default: 900000 (90%)
  uint64 heartbeat_interval_blocks      = 5;  // default: 100
  string max_price_per_cu               = 6;  // default: "1000000" (1 ZRN)
  uint64 provider_unbonding_blocks      = 7;  // default: 10000
  uint64 price_change_delay_blocks      = 8;  // default: 500
  uint64 max_latency_ms                 = 9;  // default: 5000
  uint64 sla_window_blocks              = 10; // default: 1000
  uint64 target_utilization_low_bps     = 11; // default: 300000 (30%)
  uint64 target_utilization_high_bps    = 12; // default: 800000 (80%)
}
```

### Messages
```protobuf
service Msg {
  rpc RegisterProvider(MsgRegisterProvider) returns (MsgRegisterProviderResponse);
  rpc UnregisterProvider(MsgUnregisterProvider) returns (MsgUnregisterProviderResponse);
  rpc Heartbeat(MsgHeartbeat) returns (MsgHeartbeatResponse);
  rpc UpdatePrice(MsgUpdatePrice) returns (MsgUpdatePriceResponse);
  rpc RedeemCredits(MsgRedeemCredits) returns (MsgRedeemCreditsResponse);
}
```

### Implementation
- RegisterProvider: stake min_provider_stake, store provider
- Heartbeat: update last_heartbeat, track uptime
- BeginBlocker: jail providers with missed heartbeats (last_heartbeat + heartbeat_interval < block), apply pending price changes, slash for SLA violations
- Credit system: validators earn credits for verification work, redeem for compute

---

## Module 3: Discovery

### `proto/zerone/discovery/v1/types.proto`
```protobuf
syntax = "proto3";
package zerone.discovery.v1;
option go_package = "github.com/zerone-chain/zerone/x/discovery/types";

message AgentCapability {
  string capability_type    = 1;
  repeated string domains   = 2;
  uint64 confidence_bps     = 3;  // 1M scale self-reported
  uint32 verified_by_count  = 4;
}

message AgentProfile {
  string address             = 1;
  string display_name        = 2;
  repeated AgentCapability capabilities = 3;
  repeated string domains    = 4;
  string status              = 5;  // "active", "inactive", "expired"
  uint64 reputation_score    = 6;  // 1M scale
  uint64 registered_at_block = 7;
  uint64 last_active_block   = 8;
  string stake               = 9;  // uzrn
  string description         = 10;
  string metadata            = 11; // JSON
}
```

### `proto/zerone/discovery/v1/genesis.proto`
```protobuf
message Params {
  string min_registration_stake     = 1;  // default: "1000000" (1 ZRN)
  uint32 max_capabilities_per_agent = 2;  // default: 20
  uint64 profile_expiry_blocks      = 3;  // default: 100000
}
```

### Messages
```protobuf
service Msg {
  rpc RegisterProfile(MsgRegisterProfile) returns (MsgRegisterProfileResponse);
  rpc UpdateProfile(MsgUpdateProfile) returns (MsgUpdateProfileResponse);
  rpc Heartbeat(MsgHeartbeat) returns (MsgHeartbeatResponse);
  rpc DeregisterProfile(MsgDeregisterProfile) returns (MsgDeregisterProfileResponse);
}
```

### Queries
```protobuf
service Query {
  rpc Profile(QueryProfileRequest) returns (QueryProfileResponse);
  rpc Search(QuerySearchRequest) returns (QuerySearchResponse);  // search by domain/capability
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse);
}

message QuerySearchRequest {
  string domain = 1;
  string capability_type = 2;
  uint64 min_reputation = 3;
}
message QuerySearchResponse { repeated AgentProfile profiles = 1; }
```

### Implementation
- RegisterProfile: stake, store profile, active status
- Heartbeat: update last_active_block
- BeginBlocker: expire profiles where block > last_active_block + profile_expiry_blocks
- Search: iterate profiles, filter by domain/capability/reputation (simple KV scan for now)

---

## App Wiring
Register all three modules in app.go. Wire dependencies:
- x/schedule needs BankKeeper + BVMKeeper (for contract calls)
- x/compute_pool needs BankKeeper + StakingKeeper
- x/discovery needs BankKeeper

## Tests
Port from drafts. Minimum per module:
- **Schedule:** Create + execute + pause + cancel, compound conditions, fee deduction, max executions
- **Compute Pool:** Register + heartbeat + jail on miss, price change delay, credit earn/redeem
- **Discovery:** Register + search + heartbeat + expiry, capability filtering

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- BPS: 1,000,000 scale
- Run `go build ./...` and `go test ./...` before finishing
