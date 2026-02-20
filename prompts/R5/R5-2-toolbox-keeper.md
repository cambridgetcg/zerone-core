# R5-2 — Toolbox Keeper: Registration & Revenue

## Goal

Implement the core toolbox keeper — tool CRUD, contributor management,
revenue distribution, tool calling with payment, and the module scaffold.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/keeper.go` — keeper struct, NewKeeper
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/state.go` — all KV state operations
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/msg_server.go` — message handlers
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/revenue.go` — revenue distribution
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/grpc_query.go` — queries
- `/Users/yuai/Desktop/legible_money/x/toolbox/module.go` — AppModule

**Depends on R5-1** (types package must exist).

## Implementation

### 1. Keeper Struct (`keeper/keeper.go`)
```go
type Keeper struct {
    storeKey         storetypes.StoreKey
    cdc              codec.Codec
    bankKeeper       types.BankKeeper
    knowledgeKeeper  types.KnowledgeKeeper
    billingKeeper    types.BillingKeeper
    homeKeeper       types.HomeKeeper
    bvmKeeper        types.BVMKeeper
    stakingKeeper    types.StakingKeeper
    authority        string
}
```

### 2. State Management (`keeper/state.go`)

KV store prefixes:
- `tool/` + tool_id → Tool
- `tool-deployer/` + deployer + `/` + tool_id → index
- `tool-category/` + category + `/` + tool_id → index
- `tool-call/` + call_id → ToolCall
- `tool-caller/` + tool_id + `/` + caller → CallerRecord
- `pending-contrib/` + tool_id + `/` + address → PendingContributorship
- `trust-snap/` + tool_id → TrustSnapshot
- `dep-fwd/` + from_tool_id + `/` + to_tool_id → DependencyEdge
- `dep-rev/` + to_tool_id + `/` + from_tool_id → DependencyEdge
- `demand/` + tool_id → DemandWindow
- `free-allow/` + caller → FreeAllowance
- `tool-counter` → auto-increment ID

CRUD methods:
- GetTool, SetTool, DeleteTool, GetToolsByDeployer, GetToolsByCategory
- GetCallerRecord, SetCallerRecord
- Get/Set/Delete PendingContributorship
- Get/Set TrustSnapshot
- Get/Set DemandWindow
- Get/Set FreeAllowance
- AddDependencyEdge, RemoveDependencyEdge, GetForwardDeps, GetReverseDeps

### 3. Message Server (`keeper/msg_server.go`)

- `RegisterTool` — validate inputs (category, license, tool_type), check min_tool_stake, validate dependencies (exist, not retired, trust eligible, no cycles), store tool with auto-incremented ID, store deployer + category indexes, store dependency edges
- `CallTool` — validate tool is active, compute effective price (check free tier first → surge pricing → USD-stable → fixed), charge caller via bank, distribute revenue, record ToolCall, update CallerRecord, update DemandWindow, update trust score
- `AddContributor` — deployer only (unless shares locked → governance), check max_contributors, create PendingContributorship, apply reallocations if provided, ensure shares sum ≤ 1M
- `AcceptContributorship` — contributor accepts pending invite, add to tool.Contributors with accepted=true
- `UpgradeTool` — deployer only, create new tool linked to previous, copy contributors, re-validate dependencies
- `DeprecateTool` — deployer only, set status=deprecated, optionally set successor
- `RetireTool` — deployer only, set status=retired (calls will fail)
- `LockShares` — deployer only, set share_lock_height = current block
- `UpdateDependency` — deployer only (or governance), swap old dep for new, re-validate cycles
- `ToolHeartbeat` — update last_called_block for listed tools
- `UpdateParams` — governance authority only

### 4. Revenue Distribution (`keeper/revenue.go`)

Port from draft. On every tool call:

```
totalPayment = effectivePrice

For composite tools (has dependencies):
  - Execute dependency calls first, collect their costs
  - ownRevenue = totalPayment - sum(dependencyCosts)
  - Each dependency gets its own full revenue distribution

For ownRevenue:
  55% → contributors (pro-rata by share_bps among accepted contributors)
  22% → protocol:
    50% of 22% → citation pool (x/knowledge module account)
    30% of 22% → verification pool (x/vesting_rewards module account)  
    20% of 22% → protocol treasury
  13% → research fund (x/gov research fund account)
  10% → burned
```

All percentages from params (governance-adjustable). Use the shared RevenueSplit from x/common where possible.

### 5. GRPC Query Server (`keeper/grpc_query.go`)
Implement all Query service RPCs from the proto.

### 6. Module Scaffold (`module.go`)
Standard AppModule with RegisterGRPCGatewayRoutes as no-op (consistent with other R4 modules).

### 7. App Wiring
Do NOT modify app.go — just ensure the module compiles standalone.

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- BPS: 1,000,000 scale. Revenue splits come from params, not constants
- Proto encoding for all state (no JSON marshal to KV store)
- Run `go build ./...` before finishing
