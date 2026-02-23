package keeper

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx context.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// ---------- Tool Counter ----------

func (k Keeper) nextToolID(ctx context.Context) string {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ToolCounterKey)
	if err != nil || bz == nil {
		bz = make([]byte, 8)
	}
	counter := binary.BigEndian.Uint64(bz)
	counter++
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, counter)
	_ = kvStore.Set(types.ToolCounterKey, out)
	return fmt.Sprintf("tool-%d", counter)
}

// ---------- Tool Call Counter ----------

var toolCallCounterKey = []byte{0x05}

func (k Keeper) nextCallID(ctx context.Context) string {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(toolCallCounterKey)
	if err != nil || bz == nil {
		bz = make([]byte, 8)
	}
	counter := binary.BigEndian.Uint64(bz)
	counter++
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, counter)
	_ = kvStore.Set(toolCallCounterKey, out)
	return fmt.Sprintf("call-%d", counter)
}

// ---------- Tool CRUD ----------

// SetTool stores a tool and maintains all indexes.
func (k Keeper) SetTool(ctx context.Context, tool *types.Tool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(tool)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal tool: %v", err))
	}
	_ = kvStore.Set(types.ToolKey(tool.Id), bz)

	// Maintain indexes.
	_ = kvStore.Set(types.ToolByDeployerKey(tool.Deployer, tool.Id), []byte{1})
	_ = kvStore.Set(types.ToolByStatusKey(tool.Status, tool.Id), []byte{1})
	if tool.Category != "" {
		_ = kvStore.Set(types.ToolByCategoryKey(tool.Category, tool.Id), []byte{1})
	}
	for _, tag := range tool.Tags {
		_ = kvStore.Set(types.ToolByTagKey(tag, tool.Id), []byte{1})
	}
}

// GetTool retrieves a tool by ID.
func (k Keeper) GetTool(ctx context.Context, toolID string) (*types.Tool, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ToolKey(toolID))
	if err != nil || bz == nil {
		return nil, false
	}
	var tool types.Tool
	if err := json.Unmarshal(bz, &tool); err != nil {
		return nil, false
	}
	return &tool, true
}

// UpdateToolStatus changes a tool's status and updates the status index.
func (k Keeper) UpdateToolStatus(ctx context.Context, tool *types.Tool, newStatus string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	// Remove old status index.
	_ = kvStore.Delete(types.ToolByStatusKey(tool.Status, tool.Id))
	tool.Status = newStatus
	k.SetTool(ctx, tool)
}

// DeleteToolIndexes removes all index entries for a tool.
func (k Keeper) DeleteToolIndexes(ctx context.Context, tool *types.Tool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(types.ToolByDeployerKey(tool.Deployer, tool.Id))
	_ = kvStore.Delete(types.ToolByStatusKey(tool.Status, tool.Id))
	if tool.Category != "" {
		_ = kvStore.Delete(types.ToolByCategoryKey(tool.Category, tool.Id))
	}
	for _, tag := range tool.Tags {
		_ = kvStore.Delete(types.ToolByTagKey(tag, tool.Id))
	}
}

// IterateTools iterates over all tools.
func (k Keeper) IterateTools(ctx context.Context, cb func(tool *types.Tool) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ToolKeyPrefix, prefixEndBytes(types.ToolKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var tool types.Tool
		if err := json.Unmarshal(iter.Value(), &tool); err != nil {
			continue
		}
		if cb(&tool) {
			break
		}
	}
}

// GetAllTools returns all tools.
func (k Keeper) GetAllTools(ctx context.Context) []*types.Tool {
	var tools []*types.Tool
	k.IterateTools(ctx, func(tool *types.Tool) bool {
		tools = append(tools, tool)
		return false
	})
	return tools
}

// GetToolsByDeployer returns all tools for a deployer.
func (k Keeper) GetToolsByDeployer(ctx context.Context, deployer string) []*types.Tool {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.ToolByDeployerIterPrefix(deployer)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var tools []*types.Tool
	for ; iter.Valid(); iter.Next() {
		// Key is prefix + deployer + "/" + toolID — extract toolID.
		key := iter.Key()
		toolID := string(key[len(prefix):])
		if tool, ok := k.GetTool(ctx, toolID); ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetToolsByCategory returns all tools in a category.
func (k Keeper) GetToolsByCategory(ctx context.Context, category string) []*types.Tool {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.ToolByCategoryIterPrefix(category)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var tools []*types.Tool
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		toolID := string(key[len(prefix):])
		if tool, ok := k.GetTool(ctx, toolID); ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetToolsByTag returns all tools with a given tag.
func (k Keeper) GetToolsByTag(ctx context.Context, tag string) []*types.Tool {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.ToolByTagIterPrefix(tag)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var tools []*types.Tool
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		toolID := string(key[len(prefix):])
		if tool, ok := k.GetTool(ctx, toolID); ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// ToolNameExists checks if a tool name is already taken (among non-retired tools).
func (k Keeper) ToolNameExists(ctx context.Context, name string) bool {
	found := false
	k.IterateTools(ctx, func(tool *types.Tool) bool {
		if tool.Name == name && tool.Status != types.ToolStatusRetired {
			found = true
			return true
		}
		return false
	})
	return found
}

// ---------- ToolCall CRUD ----------

// SetToolCall stores a tool call record.
func (k Keeper) SetToolCall(ctx context.Context, call *types.ToolCall) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(call)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.ToolCallKey(call.CallId), bz)
}

// GetToolCall retrieves a tool call by ID.
func (k Keeper) GetToolCall(ctx context.Context, callID string) (*types.ToolCall, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ToolCallKey(callID))
	if err != nil || bz == nil {
		return nil, false
	}
	var call types.ToolCall
	if err := json.Unmarshal(bz, &call); err != nil {
		return nil, false
	}
	return &call, true
}

// ---------- PendingContributorship CRUD ----------

// SetPendingContributorship stores a pending contributorship.
func (k Keeper) SetPendingContributorship(ctx context.Context, pc *types.PendingContributorship) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(pc)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.PendingContributorshipKey(pc.ToolId, pc.ContributorAddress), bz)
}

// GetPendingContributorship retrieves a pending contributorship.
func (k Keeper) GetPendingContributorship(ctx context.Context, toolID, address string) (*types.PendingContributorship, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.PendingContributorshipKey(toolID, address))
	if err != nil || bz == nil {
		return nil, false
	}
	var pc types.PendingContributorship
	if err := json.Unmarshal(bz, &pc); err != nil {
		return nil, false
	}
	return &pc, true
}

// DeletePendingContributorship removes a pending contributorship.
func (k Keeper) DeletePendingContributorship(ctx context.Context, toolID, address string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(types.PendingContributorshipKey(toolID, address))
}

// ---------- TrustSnapshot CRUD ----------

// SetTrustSnapshot stores a trust snapshot for a tool.
func (k Keeper) SetTrustSnapshot(ctx context.Context, snap *types.TrustSnapshot) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(snap)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.TrustSnapshotKey(snap.ToolId), bz)
}

// GetTrustSnapshot retrieves a trust snapshot.
func (k Keeper) GetTrustSnapshot(ctx context.Context, toolID string) (*types.TrustSnapshot, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.TrustSnapshotKey(toolID))
	if err != nil || bz == nil {
		return nil, false
	}
	var snap types.TrustSnapshot
	if err := json.Unmarshal(bz, &snap); err != nil {
		return nil, false
	}
	return &snap, true
}

// ---------- CallerRecord CRUD ----------

// SetCallerRecord stores a caller record.
func (k Keeper) SetCallerRecord(ctx context.Context, rec *types.CallerRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(rec)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.CallerRecordKey(rec.ToolId, rec.Caller), bz)
}

// GetCallerRecord retrieves a caller record.
func (k Keeper) GetCallerRecord(ctx context.Context, toolID, caller string) (*types.CallerRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.CallerRecordKey(toolID, caller))
	if err != nil || bz == nil {
		return nil, false
	}
	var rec types.CallerRecord
	if err := json.Unmarshal(bz, &rec); err != nil {
		return nil, false
	}
	return &rec, true
}

// RecordCaller creates or updates a caller record for a tool call.
func (k Keeper) RecordCaller(ctx context.Context, toolID, caller string, blockHeight uint64, success bool) {
	rec, found := k.GetCallerRecord(ctx, toolID, caller)
	if !found {
		rec = &types.CallerRecord{
			ToolId:         toolID,
			Caller:         caller,
			FirstCallBlock: blockHeight,
		}
	}
	rec.LastCallBlock = blockHeight
	rec.TotalCalls++
	if success {
		rec.SuccessCount++
	}
	k.SetCallerRecord(ctx, rec)
}

// CountUniqueCallers counts unique callers for a tool.
func (k Keeper) CountUniqueCallers(ctx context.Context, toolID string) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.CallerRecordIterPrefix(toolID)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0
	}
	defer iter.Close()
	var count uint64
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count
}

// IterateCallerRecords iterates over all caller records for a tool.
func (k Keeper) IterateCallerRecords(ctx context.Context, toolID string, cb func(rec *types.CallerRecord) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.CallerRecordIterPrefix(toolID)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var rec types.CallerRecord
		if err := json.Unmarshal(iter.Value(), &rec); err != nil {
			continue
		}
		if cb(&rec) {
			break
		}
	}
}

// ---------- DependencyEdge CRUD ----------

// SetDependencyEdge stores a dependency edge and its reverse index.
func (k Keeper) SetDependencyEdge(ctx context.Context, edge *types.DependencyEdge) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(edge)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.DependencyEdgeKey(edge.FromToolId, edge.ToToolId), bz)
	_ = kvStore.Set(types.ReverseDependencyKey(edge.ToToolId, edge.FromToolId), []byte{1})
}

// DeleteDependencyEdge removes a dependency edge and its reverse index.
func (k Keeper) DeleteDependencyEdge(ctx context.Context, fromToolID, toToolID string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(types.DependencyEdgeKey(fromToolID, toToolID))
	_ = kvStore.Delete(types.ReverseDependencyKey(toToolID, fromToolID))
}

// GetDependencyEdge retrieves a dependency edge.
func (k Keeper) GetDependencyEdge(ctx context.Context, fromToolID, toToolID string) (*types.DependencyEdge, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.DependencyEdgeKey(fromToolID, toToolID))
	if err != nil || bz == nil {
		return nil, false
	}
	var edge types.DependencyEdge
	if err := json.Unmarshal(bz, &edge); err != nil {
		return nil, false
	}
	return &edge, true
}

// IterateDependencyEdgesFrom iterates over all dependencies of a tool.
func (k Keeper) IterateDependencyEdgesFrom(ctx context.Context, fromToolID string, cb func(edge *types.DependencyEdge) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.DependencyEdgeIterPrefix(fromToolID)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var edge types.DependencyEdge
		if err := json.Unmarshal(iter.Value(), &edge); err != nil {
			continue
		}
		if cb(&edge) {
			break
		}
	}
}

// IterateDependentsOf iterates over all tools that depend on the given tool.
func (k Keeper) IterateDependentsOf(ctx context.Context, toToolID string, cb func(fromToolID string) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.ReverseDependencyIterPrefix(toToolID)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		fromToolID := string(key[len(prefix):])
		if cb(fromToolID) {
			break
		}
	}
}

// StoreDependencyEdges stores all dependency edges for a tool.
func (k Keeper) StoreDependencyEdges(ctx context.Context, fromToolID string, depIDs []string, blockHeight uint64) {
	for _, depID := range depIDs {
		edge := &types.DependencyEdge{
			FromToolId:     fromToolID,
			ToToolId:       depID,
			CreatedAtBlock: blockHeight,
		}
		k.SetDependencyEdge(ctx, edge)
	}
}

// deleteDependencyEdgesFrom removes all dependency edges from a tool.
func (k Keeper) deleteDependencyEdgesFrom(ctx context.Context, fromToolID string) {
	var toDelete []string
	k.IterateDependencyEdgesFrom(ctx, fromToolID, func(edge *types.DependencyEdge) bool {
		toDelete = append(toDelete, edge.ToToolId)
		return false
	})
	for _, toID := range toDelete {
		k.DeleteDependencyEdge(ctx, fromToolID, toID)
	}
}

// ---------- Agent Tool Usage ----------

// SetAgentToolUsage stores agent tool usage data.
func (k Keeper) SetAgentToolUsage(ctx context.Context, agentAddr, toolID string, callCount uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, callCount)
	_ = kvStore.Set(types.AgentToolUsageKey(agentAddr, toolID), bz)
}

// GetAgentToolUsage retrieves agent tool usage count.
func (k Keeper) GetAgentToolUsage(ctx context.Context, agentAddr, toolID string) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.AgentToolUsageKey(agentAddr, toolID))
	if err != nil || bz == nil || len(bz) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// DeleteAgentToolUsage removes agent tool usage data.
func (k Keeper) DeleteAgentToolUsage(ctx context.Context, agentAddr, toolID string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(types.AgentToolUsageKey(agentAddr, toolID))
}

// ---------- Agent Active Tools ----------

// SetAgentActiveTools stores an agent's active tool list.
func (k Keeper) SetAgentActiveTools(ctx context.Context, agentAddr string, toolIDs []string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(toolIDs)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.AgentActiveToolsKey(agentAddr), bz)
}

// GetAgentActiveTools retrieves an agent's active tool list.
func (k Keeper) GetAgentActiveTools(ctx context.Context, agentAddr string) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.AgentActiveToolsKey(agentAddr))
	if err != nil || bz == nil {
		return nil
	}
	var toolIDs []string
	if err := json.Unmarshal(bz, &toolIDs); err != nil {
		return nil
	}
	return toolIDs
}

// ---------- Free Allowance ----------

// SetFreeAllowance stores a free tier allowance.
func (k Keeper) SetFreeAllowance(ctx context.Context, fa *types.FreeAllowance) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(fa)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.FreeAllowanceKey(fa.Caller), bz)
}

// GetFreeAllowance retrieves a free tier allowance, with lazy epoch reset.
func (k Keeper) GetFreeAllowance(ctx context.Context, caller string) *types.FreeAllowance {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.FreeAllowanceKey(caller))
	if err != nil || bz == nil {
		return &types.FreeAllowance{Caller: caller, Epoch: 0, UsedCalls: 0}
	}
	var fa types.FreeAllowance
	if err := json.Unmarshal(bz, &fa); err != nil {
		return &types.FreeAllowance{Caller: caller, Epoch: 0, UsedCalls: 0}
	}
	// Lazy epoch reset.
	currentEpoch := k.getCurrentEpoch(ctx)
	if fa.Epoch < currentEpoch {
		fa.Epoch = currentEpoch
		fa.UsedCalls = 0
	}
	return &fa
}

// getCurrentEpoch returns the current epoch based on block height and BlocksPerTrustUpdate.
func (k Keeper) getCurrentEpoch(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)
	blocksPerEpoch := params.BlocksPerTrustUpdate
	if blocksPerEpoch == 0 {
		blocksPerEpoch = types.DefaultBlocksPerTrustUpdate
	}
	return blockHeight / blocksPerEpoch
}

// ---------- Demand Window ----------

// getDemandWindow retrieves a demand window for a tool.
func (k Keeper) getDemandWindow(ctx context.Context, toolID string) *types.DemandWindow {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.DemandWindowKey(toolID))
	if err != nil || bz == nil {
		params := k.GetParams(ctx)
		size := params.DemandWindowSize
		if size == 0 {
			size = types.DefaultDemandWindowSize
		}
		return &types.DemandWindow{
			ToolId:  toolID,
			Entries: make([]*types.DemandWindowEntry, size),
			Size:    size,
			Head:    0,
		}
	}
	var dw types.DemandWindow
	if err := json.Unmarshal(bz, &dw); err != nil {
		return &types.DemandWindow{ToolId: toolID, Entries: make([]*types.DemandWindowEntry, 100), Size: 100}
	}
	return &dw
}

// setDemandWindow stores a demand window.
func (k Keeper) setDemandWindow(ctx context.Context, dw *types.DemandWindow) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(dw)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.DemandWindowKey(dw.ToolId), bz)
}

// ---------- Helpers ----------

// addUint64Str adds a uint64 to a string-encoded uint64 and returns the result as string.
func addUint64Str(s string, n uint64) string {
	v, _ := strconv.ParseUint(s, 10, 64)
	return strconv.FormatUint(v+n, 10)
}

// prefixEndBytes returns the end key for prefix iteration (exclusive).
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
