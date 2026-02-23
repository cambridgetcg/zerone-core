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

	"github.com/zerone-chain/zerone/x/schedule/types"
)

// Keeper manages the schedule module's state.
type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec
	authority    string

	bankKeeper types.BankKeeper
}

// NewKeeper creates a new schedule module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	bk types.BankKeeper,
) Keeper {
	return Keeper{
		storeService: storeService,
		cdc:          cdc,
		authority:    authority,
		bankKeeper:   bk,
	}
}

// Logger returns a module-scoped logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
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

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
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
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// ---------- Sequence ----------

// GetSequence returns the current sequence number.
func (k Keeper) GetSequence(ctx context.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.SequenceKey)
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// SetSequence sets the sequence number.
func (k Keeper) SetSequence(ctx context.Context, seq uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, seq)
	_ = kvStore.Set(types.SequenceKey, bz)
}

// NextSequence increments and returns the next sequence number.
func (k Keeper) NextSequence(ctx context.Context) uint64 {
	seq := k.GetSequence(ctx) + 1
	k.SetSequence(ctx, seq)
	return seq
}

// ---------- Process CRUD ----------

// processKey returns the storage key for a process by ID.
func processKey(id string) []byte {
	return append(types.ProcessKeyPrefix, []byte(id)...)
}

// timeIndexKey returns the time index key: prefix + BigEndian(height) + 0x00 + id.
func timeIndexKey(height uint64, id string) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height)
	key := append(types.ByTimeIndexPrefix, bz...)
	key = append(key, 0x00)
	key = append(key, []byte(id)...)
	return key
}

// accountIndexKey returns the account index key: prefix + creator + 0x00 + id.
func accountIndexKey(creator string, id string) []byte {
	key := append(types.ByAccountIndexPrefix, []byte(creator)...)
	key = append(key, 0x00)
	key = append(key, []byte(id)...)
	return key
}

// statusIndexKey returns the status index key: prefix + status + 0x00 + id.
func statusIndexKey(status string, id string) []byte {
	key := append(types.ByStatusIndexPrefix, []byte(status)...)
	key = append(key, 0x00)
	key = append(key, []byte(id)...)
	return key
}

// SetProcess stores a process and maintains all indexes.
func (k Keeper) SetProcess(ctx context.Context, process *types.ScheduleProcess) {
	kvStore := k.storeService.OpenKVStore(ctx)

	// Store the process itself
	bz, err := proto.Marshal(process)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal process: %v", err))
	}
	_ = kvStore.Set(processKey(process.Id), bz)

	// Maintain time index (only if next_execute_at is set)
	if process.NextExecuteAt > 0 {
		_ = kvStore.Set(timeIndexKey(process.NextExecuteAt, process.Id), []byte(process.Id))
	}

	// Maintain account index
	_ = kvStore.Set(accountIndexKey(process.Creator, process.Id), []byte(process.Id))

	// Maintain status index
	_ = kvStore.Set(statusIndexKey(process.Status, process.Id), []byte(process.Id))
}

// GetProcess retrieves a process by ID.
func (k Keeper) GetProcess(ctx context.Context, id string) (*types.ScheduleProcess, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(processKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var process types.ScheduleProcess
	if err := proto.Unmarshal(bz, &process); err != nil {
		return nil, false
	}
	return &process, true
}

// DeleteProcess removes a process and its indexes from the store.
func (k Keeper) DeleteProcess(ctx context.Context, process *types.ScheduleProcess) {
	kvStore := k.storeService.OpenKVStore(ctx)

	_ = kvStore.Delete(processKey(process.Id))

	// Remove time index
	if process.NextExecuteAt > 0 {
		_ = kvStore.Delete(timeIndexKey(process.NextExecuteAt, process.Id))
	}

	// Remove account index
	_ = kvStore.Delete(accountIndexKey(process.Creator, process.Id))

	// Remove status index
	_ = kvStore.Delete(statusIndexKey(process.Status, process.Id))
}

// RemoveIndexes removes all secondary indexes for a process.
func (k Keeper) RemoveIndexes(ctx context.Context, process *types.ScheduleProcess) {
	kvStore := k.storeService.OpenKVStore(ctx)

	if process.NextExecuteAt > 0 {
		_ = kvStore.Delete(timeIndexKey(process.NextExecuteAt, process.Id))
	}
	_ = kvStore.Delete(accountIndexKey(process.Creator, process.Id))
	_ = kvStore.Delete(statusIndexKey(process.Status, process.Id))
}

// IterateProcesses iterates over all processes. Return true from cb to stop.
func (k Keeper) IterateProcesses(ctx context.Context, cb func(*types.ScheduleProcess) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ProcessKeyPrefix, prefixEndBytes(types.ProcessKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var process types.ScheduleProcess
		if err := proto.Unmarshal(iter.Value(), &process); err != nil {
			continue
		}
		if cb(&process) {
			break
		}
	}
}

// GetProcessesByCreator returns all processes for a given creator.
func (k Keeper) GetProcessesByCreator(ctx context.Context, creator string) []*types.ScheduleProcess {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.ByAccountIndexPrefix, []byte(creator)...)
	prefix = append(prefix, 0x00)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var processes []*types.ScheduleProcess
	for ; iter.Valid(); iter.Next() {
		processId := string(iter.Value())
		process, found := k.GetProcess(ctx, processId)
		if found {
			processes = append(processes, process)
		}
	}
	return processes
}

// CountActiveByCreator returns the number of active processes for a creator.
func (k Keeper) CountActiveByCreator(ctx context.Context, creator string) uint32 {
	processes := k.GetProcessesByCreator(ctx, creator)
	var count uint32
	for _, p := range processes {
		if p.Status == "active" {
			count++
		}
	}
	return count
}

// GetDueProcesses returns process IDs that are due for execution at or before the given block height.
func (k Keeper) GetDueProcesses(ctx context.Context, blockHeight uint64) []string {
	kvStore := k.storeService.OpenKVStore(ctx)

	// Iterate from the start of the time index up to and including the target height.
	// End key is timeIndexKey(blockHeight+1, "") to include all entries at blockHeight.
	endBz := make([]byte, 8)
	binary.BigEndian.PutUint64(endBz, blockHeight+1)
	endKey := append(types.ByTimeIndexPrefix, endBz...)

	iter, err := kvStore.Iterator(types.ByTimeIndexPrefix, endKey)
	if err != nil {
		return nil
	}
	defer iter.Close()

	var ids []string
	for ; iter.Valid(); iter.Next() {
		ids = append(ids, string(iter.Value()))
	}
	return ids
}

// ---------- Genesis ----------

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	k.SetSequence(ctx, genState.Sequence)
	for _, process := range genState.Processes {
		k.SetProcess(ctx, process)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	seq := k.GetSequence(ctx)

	var processes []*types.ScheduleProcess
	k.IterateProcesses(ctx, func(p *types.ScheduleProcess) bool {
		processes = append(processes, p)
		return false
	})

	return &types.GenesisState{
		Params:    params,
		Processes: processes,
		Sequence:  seq,
	}
}
