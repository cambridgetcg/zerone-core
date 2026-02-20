# R4-3 — BVM Module: Bytecode Virtual Machine

## Goal

Port x/bvm — Zerone's execution engine for on-chain agent contracts. Supports
contract deployment, invocation, scheduled execution, knowledge bridge opcodes,
and gas metering. NOT a full EVM — a purpose-built stack VM with ~47 opcodes
focused on knowledge queries, tool calls, and agent coordination.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/bvm/` — full module (5930 LOC keeper, 4062 LOC tests)
- `/Users/yuai/Desktop/legible_money/proto/legible/bvm/v1/state.proto` — state types
- `/Users/yuai/Desktop/legible_money/proto/legible/bvm/v1/tx.proto` — messages
- `/Users/yuai/Desktop/legible_money/proto/legible/bvm/v1/query.proto` — queries
- `/Users/yuai/Desktop/legible_money/proto/legible/bvm/v1/genesis.proto` — genesis

## Proto Files

### `proto/zerone/bvm/v1/types.proto`
```protobuf
syntax = "proto3";
package zerone.bvm.v1;
option go_package = "github.com/zerone-chain/zerone/x/bvm/types";

message DeployedContract {
  string address           = 1;   // derived from deployer + nonce
  string code_hash         = 2;
  string creator           = 3;   // bech32
  uint64 deployed_at_block = 4;
  bool   is_agent          = 5;
  string agent_home_id     = 6;   // link to x/home (if agent-owned)
  uint64 bytecode_size     = 7;
  string label             = 8;
  uint32 bvm_version       = 9;   // opcode activation version
}

message ContractCode {
  string code_hash = 1;
  bytes  bytecode  = 2;
  uint64 ref_count = 3;
}

message ContractStateEntry {
  string contract_address = 1;
  string key              = 2;
  string value            = 3;
}

message ContractSchedule {
  string schedule_id      = 1;
  string contract_address = 2;
  string caller           = 3;
  string method           = 4;
  string payload          = 5;
  uint64 execute_at_block = 6;
  uint64 max_gas          = 7;
  bool   executed         = 8;
  bool   cancelled        = 9;
}

message ContractCallRecord {
  string tx_hash          = 1;
  string contract_address = 2;
  string caller           = 3;
  string method           = 4;
  uint64 gas_used         = 5;
  bool   success          = 6;
  string return_data      = 7;
  string error            = 8;
  uint64 block_number     = 9;
}

// Schedule conditions for self-invoking contracts
message ContractConfig {
  uint64 max_gas_per_invocation = 1;
  bool   self_invoking          = 2;
  string schedule_pattern       = 3;
}

message ScheduleCondition {
  oneof condition {
    BlockInterval block_interval   = 1;
    TimeCondition time_condition   = 2;
    StateCondition state_condition = 3;
  }
}

message BlockInterval {
  uint64 every_n_blocks = 1;
  uint64 start_height   = 2;
  uint64 end_height     = 3;
}

message TimeCondition {
  uint64 after_timestamp      = 1;
  uint64 repeat_interval_ms   = 2;
}

message StateCondition {
  string key      = 1;
  string operator = 2;  // "eq", "gt", "lt", "gte", "lte"
  bytes  value    = 3;
}

message ContractEvent {
  string contract_address       = 1;
  string event_type             = 2;
  map<string, string> attributes = 3;
}
```

### `proto/zerone/bvm/v1/tx.proto`
```protobuf
syntax = "proto3";
package zerone.bvm.v1;
option go_package = "github.com/zerone-chain/zerone/x/bvm/types";

import "cosmos/msg/v1/msg.proto";
import "zerone/bvm/v1/types.proto";

service Msg {
  option (cosmos.msg.v1.service) = true;

  rpc DeployContract(MsgDeployContract) returns (MsgDeployContractResponse);
  rpc CallContract(MsgCallContract) returns (MsgCallContractResponse);
  rpc ScheduleExecution(MsgScheduleExecution) returns (MsgScheduleExecutionResponse);
}

message MsgDeployContract {
  option (cosmos.msg.v1.signer) = "deployer";
  string deployer          = 1;
  bytes  bytecode          = 2;
  bytes  constructor_args  = 3;
  string initial_deposit   = 4;  // uzrn
  ContractConfig config    = 5;
}
message MsgDeployContractResponse { string contract_address = 1; }

message MsgCallContract {
  option (cosmos.msg.v1.signer) = "caller";
  string caller            = 1;
  string contract_address  = 2;
  bytes  input_data        = 3;
  string value             = 4;  // uzrn to send
  uint64 gas_limit         = 5;
  bool   static_call       = 6;  // no state mods
}
message MsgCallContractResponse {
  bytes  return_data = 1;
  uint64 gas_used    = 2;
  repeated ContractEvent events = 3;
}

message MsgScheduleExecution {
  option (cosmos.msg.v1.signer) = "scheduler";
  string scheduler         = 1;
  string contract_address  = 2;
  bytes  input_data        = 3;
  ScheduleCondition condition = 4;
}
message MsgScheduleExecutionResponse { string schedule_id = 1; }
```

### `proto/zerone/bvm/v1/query.proto`
```protobuf
syntax = "proto3";
package zerone.bvm.v1;
option go_package = "github.com/zerone-chain/zerone/x/bvm/types";

import "google/api/annotations.proto";
import "zerone/bvm/v1/types.proto";
import "zerone/bvm/v1/genesis.proto";

service Query {
  rpc Contract(QueryContractRequest) returns (QueryContractResponse) {
    option (google.api.http).get = "/zerone/bvm/v1/contract/{address}";
  }
  rpc ContractsByCreator(QueryByCreatorRequest) returns (QueryByCreatorResponse) {
    option (google.api.http).get = "/zerone/bvm/v1/contracts/{creator}";
  }
  rpc ContractState(QueryContractStateRequest) returns (QueryContractStateResponse) {
    option (google.api.http).get = "/zerone/bvm/v1/state/{address}/{key}";
  }
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse) {
    option (google.api.http).get = "/zerone/bvm/v1/params";
  }
}

message QueryContractRequest { string address = 1; }
message QueryContractResponse { DeployedContract contract = 1; }
message QueryByCreatorRequest { string creator = 1; }
message QueryByCreatorResponse { repeated DeployedContract contracts = 1; }
message QueryContractStateRequest { string address = 1; string key = 2; }
message QueryContractStateResponse { string value = 1; }
message QueryParamsRequest {}
message QueryParamsResponse { Params params = 1; }
```

### `proto/zerone/bvm/v1/genesis.proto`
```protobuf
syntax = "proto3";
package zerone.bvm.v1;
option go_package = "github.com/zerone-chain/zerone/x/bvm/types";

import "zerone/bvm/v1/types.proto";

message GenesisState {
  Params params = 1;
  repeated DeployedContract contracts = 2;
  repeated ContractCode codes = 3;
}

message Params {
  uint64 max_bytecode_size = 1;         // default: 65536 (64KB)
  uint64 max_gas_per_call = 2;          // default: 10000000
  uint64 max_gas_per_block = 3;         // default: 100000000
  uint64 max_contracts_per_creator = 4; // default: 100
  uint64 max_state_entries = 5;         // default: 10000 per contract
  string deploy_cost = 6;              // default: "5000000" (5 ZRN)
  uint64 max_schedule_gas = 7;         // default: 1000000
  uint64 schedule_horizon_blocks = 8;  // default: 100000
  uint32 current_bvm_version = 9;     // opcode activation version
}
```

## Implementation

### 1. Proto Generation + Module Scaffold
Standard layout: types/, keeper/, module.go

### 2. VM Engine (`keeper/vm.go`)

Implement a stack-based bytecode VM with these opcode groups:

**Arithmetic:** ADD, SUB, MUL, DIV, MOD, EXP
**Comparison:** LT, GT, EQ, ISZERO
**Bitwise:** AND, OR, XOR, NOT, SHL, SHR
**Stack:** PUSH1-PUSH32, POP, DUP, SWAP
**Memory:** MLOAD, MSTORE, MSTORE8
**Storage:** SLOAD, SSTORE
**Control:** JUMP, JUMPI, STOP, RETURN, REVERT
**System:** CALLER, CALLVALUE, GASPRICE, BLOCKNUMBER, TIMESTAMP
**Knowledge bridge (Zerone-specific):**
  - KQUERY (0xE0) — query a fact from x/knowledge, costs gas based on billing
  - KVERIFY (0xE1) — submit verification vote
  - KCITE (0xE2) — record citation, triggers revenue cascade

Each opcode has a fixed gas cost. Knowledge bridge opcodes additionally charge
via x/billing dynamic pricing.

**Gas metering:** Deduct per opcode. Revert on out-of-gas. Refund unused gas.

**Static call enforcement:** If static_call=true, SSTORE and value transfers revert.

### 3. Keeper Implementation

**State (`state.go`):**
- Contracts: prefix `bvm-contract/` + address
- Code: prefix `bvm-code/` + code_hash
- Contract state: prefix `bvm-state/` + address + `/` + key
- Schedules: prefix `bvm-sched/` + schedule_id
- Call records: prefix `bvm-call/` + tx_hash (last N only)
- Creator index: prefix `bvm-creator/` + creator + `/` + address
- Deploy nonce: prefix `bvm-nonce/` + deployer

**Message Server:**
- `DeployContract` — validate bytecode size, charge deploy_cost, hash bytecode, derive address from deployer+nonce, store code (dedup by hash), store contract, run constructor
- `CallContract` — load contract, validate gas_limit ≤ max_gas_per_call, create VM instance, execute, store state changes atomically, record call, emit events
- `ScheduleExecution` — validate schedule condition, validate gas ≤ max_schedule_gas, validate horizon, store schedule

**BeginBlocker:**
- Execute pending schedules where execute_at_block ≤ current block
- Track cumulative gas per block, stop when max_gas_per_block reached
- Mark executed schedules

### 4. Expected Keepers
```go
type BankKeeper interface {
    SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
    SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, module string, amt sdk.Coins) error
}

type KnowledgeKeeper interface {
    GetFact(ctx context.Context, factID string) (*knowledgetypes.Fact, bool)
    GetFactConfidence(ctx context.Context, factID string) (uint64, error)
}

type BillingKeeper interface {
    ChargeQuery(ctx context.Context, payer sdk.AccAddress, factID string) (sdk.Coins, error)
}

type HomeKeeper interface {
    GetHome(ctx context.Context, homeID string) (*hometypes.AgentHome, bool)
}
```

### 5. Tests
Port from draft (4062 LOC). Minimum:
- Deploy + call happy path
- Gas metering (out-of-gas reverts state)
- Static call enforcement
- Knowledge bridge opcodes (KQUERY returns fact data, charges billing)
- Scheduled execution in BeginBlocker
- Bytecode size limit enforcement
- Code deduplication (same bytecode → same code_hash)
- Contract state isolation (contract A can't read contract B's state)

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- Contract addresses: hex-encoded hash of deployer+nonce (20 bytes), displayed as bech32
- VM is deterministic — no floating point, no randomness, no external calls except knowledge bridge
- Run `go build ./...` and `go test ./...` before finishing
