# R5-1 — Toolbox Proto + Types

## Goal

Define all proto types for the toolbox module — tools, contributors, trust
snapshots, tool calls, demand windows, free tier allowances, dependency edges,
knowledge scout types, and purpose analyzer types. Also write the genesis,
tx, and query service definitions.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/toolbox/types/types.go` — all types (~550 lines)
- `/Users/yuai/Desktop/legible_money/x/toolbox/types/params.go` — params with 25+ fields
- `/Users/yuai/Desktop/legible_money/x/toolbox/types/keys.go` — store key prefixes
- `/Users/yuai/Desktop/legible_money/x/toolbox/types/errors.go` — error sentinels
- `/Users/yuai/Desktop/legible_money/x/toolbox/types/codec.go` — codec registration
- `/Users/yuai/Desktop/legible_money/x/toolbox/types/expected_keepers.go` — keeper interfaces

**IMPORTANT:** The draft uses hand-written JSON types. We rewrite as proper proto.
Rename all `ulgm`/`LGM` references to `uzrn`/`ZRN`. Module path: `github.com/zerone-chain/zerone`.

## Proto Files

### `proto/zerone/toolbox/v1/types.proto`
```protobuf
syntax = "proto3";
package zerone.toolbox.v1;
option go_package = "github.com/zerone-chain/zerone/x/toolbox/types";

// Tool is the primary registry record for a deployed tool.
message Tool {
  string id                    = 1;
  string name                  = 2;
  string description           = 3;
  string version               = 4;
  string previous_version_id   = 5;
  string tool_type             = 6;   // "bvm_contract", "tree_service", "knowledge_template", "composite"
  string deployer              = 7;   // bech32

  // Backing resource (exactly one set based on tool_type)
  string contract_address      = 8;
  string service_id            = 9;
  string knowledge_query       = 10;
  repeated string dependency_ids = 11;

  // Contributors
  repeated ContributorShare contributors = 12;
  uint64 share_lock_height     = 13;

  // Economics
  string price_per_call        = 14;  // uzrn, "0" = free
  string target_price_usd      = 15;  // micro-USD (6 dec), empty = fixed ZRN
  string min_price_per_call    = 16;  // uzrn floor for USD-stable
  string max_price_per_call    = 17;  // uzrn ceiling for USD-stable
  string total_revenue         = 18;  // lifetime uzrn
  string total_calls           = 19;  // lifetime count
  uint64 unique_callers        = 20;

  // Trust & Quality
  uint64 trust_score           = 21;  // 0-1,000,000
  uint64 deployed_at_block     = 22;
  uint64 last_called_block     = 23;
  string status                = 24;  // "draft", "testing", "active", "deprecated", "retired"
  bool   is_verified           = 25;
  uint64 verified_since        = 26;
  uint64 verified_demotion_block = 27;

  // Metadata
  string source_hash           = 28;
  string documentation_hash    = 29;
  string license               = 30;  // "open", "restricted", "commercial"
  repeated string tags         = 31;
  string api_schema            = 32;
  string category              = 33;  // one of 10 categories
  repeated string required_capabilities = 34;
}

message ContributorShare {
  string address       = 1;
  string role          = 2;   // "architect", "developer", "tester", "data_curator", "maintainer"
  uint64 share_bps     = 3;   // 1M scale
  uint64 joined_at_block = 4;
  string total_earned  = 5;   // uzrn lifetime
  bool   accepted      = 6;   // consent flag
}

message ToolCall {
  string call_id       = 1;
  string tool_id       = 2;
  string caller        = 3;   // bech32
  string payment       = 4;   // uzrn
  uint64 block_height  = 5;
  bool   success       = 6;
  string error         = 7;
  repeated string sub_calls = 8;  // dependency tool IDs called
}

message PendingContributorship {
  string tool_id             = 1;
  string contributor_address = 2;
  string role                = 3;
  uint64 share_bps           = 4;
  uint64 proposed_at_block   = 5;
}

message DependencyEdge {
  string from_tool_id    = 1;
  string to_tool_id      = 2;
  string pinned_version  = 3;
  uint64 created_at_block = 4;
}

message DependencyTreeNode {
  string tool_id         = 1;
  string name            = 2;
  string version         = 3;
  string price_per_call  = 4;
  uint64 trust_score     = 5;
  string status          = 6;
  repeated DependencyTreeNode children = 7;
}

// Trust engine types
message TrustSnapshot {
  string tool_id                  = 1;
  uint64 score                    = 2;   // 0-1,000,000
  uint64 usage_component          = 3;
  uint64 verification_component   = 4;
  uint64 reliability_component    = 5;
  uint64 peer_component           = 6;
  uint64 contributor_component    = 7;
  uint64 unique_callers_window    = 8;
  uint64 total_calls_window       = 9;
  uint64 success_rate_bps         = 10;
  uint64 computed_at_block        = 11;
}

message CallerRecord {
  string tool_id         = 1;
  string caller          = 2;
  uint64 first_call_block = 3;
  uint64 last_call_block = 4;
  uint64 total_calls     = 5;
  uint64 success_count   = 6;
}

// Demand tracking
message DemandWindow {
  string tool_id               = 1;
  repeated DemandWindowEntry entries = 2;
  uint64 size                  = 3;
  uint64 head                  = 4;
}

message DemandWindowEntry {
  uint64 block_height = 1;
  uint64 call_count   = 2;
}

// Free tier
message FreeAllowance {
  string caller     = 1;
  uint64 epoch      = 2;
  uint64 used_calls = 3;
}

// Knowledge Scout types
message KnowledgeScoutInput {
  string domain          = 1;
  repeated string query_terms = 2;
  repeated string capabilities = 3;
  uint64 max_results     = 4;
  uint64 min_confidence  = 5;  // 0-1M bps
}

message KnowledgeScoutOutput {
  repeated ScoredFact facts = 1;
  uint64 total_found        = 2;
}

message ScoredFact {
  string fact_id         = 1;
  string content         = 2;
  uint64 confidence      = 3;
  string category        = 4;
  uint64 relevance_score = 5;  // 0-1M
  uint64 citation_count  = 6;
  uint64 fundamentality  = 7;
}

// Purpose Analyzer types
message PurposeAnalyzerInput {
  repeated string agent_capabilities = 1;
  repeated ScoredFact knowledge_facts = 2;
  AgentHistory agent_history          = 3;
}

message AgentHistory {
  uint64 total_verifications = 1;
  uint64 total_tool_calls    = 2;
  uint64 tools_deployed      = 3;
  uint64 partnerships_formed = 4;
  repeated string active_domains = 5;
}

message PurposeAnalysis {
  PurposeHypothesis primary_purpose     = 1;
  repeated PurposeHypothesis alternatives = 2;
  repeated string capability_gaps       = 3;
  repeated GrowthRecommendation growth_path = 4;
  uint64 overall_confidence             = 5;  // 0-1M
}

message PurposeHypothesis {
  string statement     = 1;
  uint64 confidence    = 2;  // 0-1M
  repeated string supporting_evidence = 3;
}

message GrowthRecommendation {
  string capability  = 1;
  string rationale   = 2;
  string target_tier = 3;
  uint64 estimated_epochs = 4;
}

// Path formatter output
message FormattedPath {
  string current_state  = 1;
  repeated PathStep steps = 2;
  string destination    = 3;
  uint64 estimated_epochs = 4;
}

message PathStep {
  uint32 step_number = 1;
  string action      = 2;
  string description = 3;
  string target_metric = 4;
}
```

### `proto/zerone/toolbox/v1/genesis.proto`
```protobuf
syntax = "proto3";
package zerone.toolbox.v1;
option go_package = "github.com/zerone-chain/zerone/x/toolbox/types";

import "zerone/toolbox/v1/types.proto";

message GenesisState {
  Params params = 1;
  repeated Tool tools = 2;
}

message Params {
  // Tool registry
  uint32 max_contributors = 1;             // default: 22
  uint32 max_dependency_depth = 2;         // default: 10
  uint32 max_dependencies = 3;             // default: 20
  uint64 min_tool_stake = 4;               // default: 11000000 (11 ZRN)
  uint64 share_lock_cooldown_blocks = 5;   // default: 34272 (~1 day)
  uint64 deprecation_grace_blocks = 6;     // default: 240000 (~1 week)
  uint64 blocks_per_trust_update = 7;      // default: 1000 (~42 min)
  uint64 verified_grace_period_blocks = 8; // default: 10000 (~7 hours)
  uint64 tool_gas_limit = 9;              // default: 1000000

  // Demand tracking
  uint64 demand_window_size = 10;                 // default: 1000
  uint64 target_calls_per_block_per_tool = 11;    // default: 10
  uint64 target_global_calls_per_block = 12;      // default: 100

  // Surge pricing
  uint64 surge_threshold_bps = 13;         // default: 500000 (50%)
  uint64 surge_critical_bps = 14;          // default: 800000 (80%)
  uint64 max_surge_multiplier_bps = 15;    // default: 10000000 (10×)
  bool   surge_enabled = 16;               // default: true

  // Free tier
  uint64 free_calls_per_epoch = 17;        // default: 50
  uint64 min_home_age_blocks = 18;         // default: 10000 (~7 hours)
  bool   free_calls_enabled = 19;          // default: true

  // Revenue split (governance-adjustable, must sum to 1,000,000)
  uint64 tool_revenue_bps = 20;            // default: 550000 (55%)
  uint64 protocol_bps = 21;               // default: 220000 (22%)
  uint64 research_bps = 22;               // default: 130000 (13%)
  uint64 burn_bps = 23;                   // default: 100000 (10%)

  // Protocol sub-split (must sum to 1,000,000)
  uint64 protocol_citation_bps = 24;      // default: 500000 (50%)
  uint64 protocol_verification_bps = 25;  // default: 300000 (30%)
  uint64 protocol_treasury_bps = 26;      // default: 200000 (20%)
}
```

### `proto/zerone/toolbox/v1/tx.proto`
```protobuf
syntax = "proto3";
package zerone.toolbox.v1;
option go_package = "github.com/zerone-chain/zerone/x/toolbox/types";

import "cosmos/msg/v1/msg.proto";
import "zerone/toolbox/v1/types.proto";

service Msg {
  option (cosmos.msg.v1.service) = true;

  rpc RegisterTool(MsgRegisterTool) returns (MsgRegisterToolResponse);
  rpc CallTool(MsgCallTool) returns (MsgCallToolResponse);
  rpc AddContributor(MsgAddContributor) returns (MsgAddContributorResponse);
  rpc AcceptContributorship(MsgAcceptContributorship) returns (MsgAcceptContributorshipResponse);
  rpc UpgradeTool(MsgUpgradeTool) returns (MsgUpgradeToolResponse);
  rpc DeprecateTool(MsgDeprecateTool) returns (MsgDeprecateToolResponse);
  rpc RetireTool(MsgRetireTool) returns (MsgRetireToolResponse);
  rpc LockShares(MsgLockShares) returns (MsgLockSharesResponse);
  rpc UpdateDependency(MsgUpdateDependency) returns (MsgUpdateDependencyResponse);
  rpc ToolHeartbeat(MsgToolHeartbeat) returns (MsgToolHeartbeatResponse);
  rpc UpdateParams(MsgUpdateParams) returns (MsgUpdateParamsResponse);
}

message MsgRegisterTool {
  option (cosmos.msg.v1.signer) = "deployer";
  string deployer = 1; string name = 2; string description = 3;
  string version = 4; string tool_type = 5;
  string contract_address = 6; string service_id = 7;
  string price_per_call = 8; string target_price_usd = 9;
  string min_price_per_call = 10; string max_price_per_call = 11;
  string source_hash = 12; string api_schema = 13;
  string license = 14; repeated string tags = 15;
  repeated string dependency_ids = 16; string category = 17;
  repeated string required_capabilities = 18;
}
message MsgRegisterToolResponse { string tool_id = 1; }

message MsgCallTool {
  option (cosmos.msg.v1.signer) = "caller";
  string caller = 1; string tool_id = 2;
  bytes input = 3; string max_fee = 4;
}
message MsgCallToolResponse { string call_id = 1; bool success = 2; }

message MsgAddContributor {
  option (cosmos.msg.v1.signer) = "authority";
  string authority = 1; string tool_id = 2;
  string contributor_address = 3; string role = 4; uint64 share_bps = 5;
  repeated ShareReallocation reallocations = 6;
}
message ShareReallocation { string address = 1; uint64 new_share_bps = 2; }
message MsgAddContributorResponse {}

message MsgAcceptContributorship {
  option (cosmos.msg.v1.signer) = "contributor_address";
  string contributor_address = 1; string tool_id = 2;
}
message MsgAcceptContributorshipResponse {}

message MsgUpgradeTool {
  option (cosmos.msg.v1.signer) = "deployer";
  string deployer = 1; string previous_tool_id = 2; string new_version = 3;
  string description = 4; string contract_address = 5; string service_id = 6;
  string price_per_call = 7; string target_price_usd = 8;
  string min_price_per_call = 9; string max_price_per_call = 10;
  string source_hash = 11; string api_schema = 12;
  repeated string dependency_ids = 13; string migration_notes = 14;
}
message MsgUpgradeToolResponse { string new_tool_id = 1; }

message MsgDeprecateTool {
  option (cosmos.msg.v1.signer) = "authority";
  string authority = 1; string tool_id = 2; string successor_tool_id = 3;
}
message MsgDeprecateToolResponse {}

message MsgRetireTool {
  option (cosmos.msg.v1.signer) = "authority";
  string authority = 1; string tool_id = 2;
}
message MsgRetireToolResponse {}

message MsgLockShares {
  option (cosmos.msg.v1.signer) = "deployer";
  string deployer = 1; string tool_id = 2;
}
message MsgLockSharesResponse {}

message MsgUpdateDependency {
  option (cosmos.msg.v1.signer) = "authority";
  string authority = 1; string tool_id = 2;
  string old_dep_id = 3; string new_dep_id = 4;
}
message MsgUpdateDependencyResponse {}

message MsgToolHeartbeat {
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1; repeated string active_tools = 2;
}
message MsgToolHeartbeatResponse {}

message MsgUpdateParams {
  option (cosmos.msg.v1.signer) = "authority";
  string authority = 1; Params params = 2;
}
message MsgUpdateParamsResponse {}
```

### `proto/zerone/toolbox/v1/query.proto`
```protobuf
syntax = "proto3";
package zerone.toolbox.v1;
option go_package = "github.com/zerone-chain/zerone/x/toolbox/types";

import "google/api/annotations.proto";
import "zerone/toolbox/v1/types.proto";
import "zerone/toolbox/v1/genesis.proto";

service Query {
  rpc Tool(QueryToolRequest) returns (QueryToolResponse) {
    option (google.api.http).get = "/zerone/toolbox/v1/tool/{tool_id}";
  }
  rpc ToolsByDeployer(QueryByDeployerRequest) returns (QueryByDeployerResponse) {
    option (google.api.http).get = "/zerone/toolbox/v1/tools/{deployer}";
  }
  rpc ToolsByCategory(QueryByCategoryRequest) returns (QueryByCategoryResponse) {
    option (google.api.http).get = "/zerone/toolbox/v1/category/{category}";
  }
  rpc TrustScore(QueryTrustScoreRequest) returns (QueryTrustScoreResponse) {
    option (google.api.http).get = "/zerone/toolbox/v1/trust/{tool_id}";
  }
  rpc DependencyTree(QueryDependencyTreeRequest) returns (QueryDependencyTreeResponse) {
    option (google.api.http).get = "/zerone/toolbox/v1/deps/{tool_id}";
  }
  rpc FreeAllowance(QueryFreeAllowanceRequest) returns (QueryFreeAllowanceResponse) {
    option (google.api.http).get = "/zerone/toolbox/v1/free/{caller}";
  }
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse) {
    option (google.api.http).get = "/zerone/toolbox/v1/params";
  }
}

message QueryToolRequest { string tool_id = 1; }
message QueryToolResponse { Tool tool = 1; }
message QueryByDeployerRequest { string deployer = 1; }
message QueryByDeployerResponse { repeated Tool tools = 1; }
message QueryByCategoryRequest { string category = 1; }
message QueryByCategoryResponse { repeated Tool tools = 1; }
message QueryTrustScoreRequest { string tool_id = 1; }
message QueryTrustScoreResponse { TrustSnapshot snapshot = 1; }
message QueryDependencyTreeRequest { string tool_id = 1; }
message QueryDependencyTreeResponse { DependencyTreeNode tree = 1; }
message QueryFreeAllowanceRequest { string caller = 1; }
message QueryFreeAllowanceResponse { FreeAllowance allowance = 1; }
message QueryParamsRequest {}
message QueryParamsResponse { Params params = 1; }
```

## Implementation

### 1. Proto Generation
Generate all protos with the project's protoc/buf tooling.

### 2. Types Package
Create `x/toolbox/types/` with:
- `keys.go` — module name ("toolbox"), store key, all prefix constants
- `errors.go` — error sentinels (ErrToolNotFound, ErrDependencyCycle, ErrDependencyDepthExceeded, ErrTooManyDependencies, ErrIneligibleDependency, ErrToolRetired, ErrSharesLocked, ErrUnauthorized, ErrInvalidCategory, ErrFreeTierExhausted, etc.)
- `codec.go` — RegisterCodec, RegisterInterfaces for all Msg types
- `expected_keepers.go` — BankKeeper, KnowledgeKeeper, BillingKeeper, HomeKeeper, BVMKeeper, StakingKeeper interfaces
- `genesis.go` — DefaultGenesisState(), GenesisState.Validate()
- `constants.go` — tool statuses, types, roles, licenses, categories, trust tier boundaries, pricing tier constants. Port all constants from the draft types.go

### 3. Validation Helpers
- `IsValidCategory(cat string) bool`
- `TrustTier(score uint64) uint32` — 5 tiers: Unverified(0-100k), Emerging(100k-300k), Established(300k-600k), Trusted(600k-800k), Verified(800k-1M)
- `TrustTierLabel(score uint64) string`
- `IsDependencyEligible(score uint64) bool` — tier ≥ 1
- `DefaultParams() *Params` with all 26 defaults
- `Params.Validate()` — revenue splits sum to 1M, sub-splits sum to 1M

## Conventions
- Token: uzrn (not ulgm). Module path: github.com/zerone-chain/zerone
- BPS: 1,000,000 scale everywhere
- Proto-first: no hand-written JSON types
- Run `go build ./...` before finishing
