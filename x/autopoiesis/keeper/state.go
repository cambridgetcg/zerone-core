package keeper

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/autopoiesis/types"
)

// ========== Params ==========

// SetParams stores the module parameters.
func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic("failed to marshal autopoiesis params: " + err.Error())
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic("failed to set autopoiesis params: " + err.Error())
	}
}

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx context.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		p := types.DefaultParams()
		return &p
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		p := types.DefaultParams()
		return &p
	}
	return &params
}

// ========== AutopoiesisState ==========

// SetState stores the runtime state.
func (k Keeper) SetState(ctx context.Context, state *types.AutopoiesisState) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(state)
	if err != nil {
		panic("failed to marshal autopoiesis state: " + err.Error())
	}
	if err := store.Set(types.StateKey, bz); err != nil {
		panic("failed to set autopoiesis state: " + err.Error())
	}
}

// GetState returns the runtime state.
func (k Keeper) GetState(ctx context.Context) *types.AutopoiesisState {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.StateKey)
	if err != nil || bz == nil {
		return &types.AutopoiesisState{}
	}
	var state types.AutopoiesisState
	if err := json.Unmarshal(bz, &state); err != nil {
		return &types.AutopoiesisState{}
	}
	return &state
}

// IsActive returns whether the autopoiesis module is activated.
func (k Keeper) IsActive(ctx context.Context) bool {
	return k.GetState(ctx).Activated
}

// ========== MultiplierState ==========

// SetMultiplierState stores a multiplier state by path.
func (k Keeper) SetMultiplierState(ctx context.Context, ms *types.MultiplierState) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(ms)
	if err != nil {
		panic("failed to marshal multiplier state: " + err.Error())
	}
	if err := store.Set(types.MultiplierKey(ms.Path), bz); err != nil {
		panic("failed to set multiplier state: " + err.Error())
	}
}

// GetMultiplierState retrieves a multiplier state by path.
func (k Keeper) GetMultiplierState(ctx context.Context, path string) (*types.MultiplierState, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.MultiplierKey(path))
	if err != nil || bz == nil {
		return nil, false
	}
	var ms types.MultiplierState
	if err := json.Unmarshal(bz, &ms); err != nil {
		return nil, false
	}
	return &ms, true
}

// ========== Frozen ==========

// SetMultiplierFrozen sets the frozen flag for a multiplier path.
func (k Keeper) SetMultiplierFrozen(ctx context.Context, path string, frozen bool) {
	store := k.storeService.OpenKVStore(ctx)
	val := byte(0)
	if frozen {
		val = 1
	}
	if err := store.Set(types.FrozenKey(path), []byte{val}); err != nil {
		panic("failed to set frozen flag: " + err.Error())
	}
}

// IsMultiplierFrozen returns whether a multiplier is frozen.
func (k Keeper) IsMultiplierFrozen(ctx context.Context, path string) bool {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.FrozenKey(path))
	if err != nil || bz == nil {
		return false
	}
	return len(bz) > 0 && bz[0] == 1
}

// ========== EpochSnapshot ==========

// SetEpochSnapshot stores an epoch snapshot.
func (k Keeper) SetEpochSnapshot(ctx context.Context, s *types.EpochSnapshot) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(s)
	if err != nil {
		panic("failed to marshal epoch snapshot: " + err.Error())
	}
	if err := store.Set(types.SnapshotKey(s.Epoch), bz); err != nil {
		panic("failed to set epoch snapshot: " + err.Error())
	}
}

// GetEpochSnapshot retrieves an epoch snapshot.
func (k Keeper) GetEpochSnapshot(ctx context.Context, epoch uint64) (*types.EpochSnapshot, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.SnapshotKey(epoch))
	if err != nil || bz == nil {
		return nil, false
	}
	var s types.EpochSnapshot
	if err := json.Unmarshal(bz, &s); err != nil {
		return nil, false
	}
	return &s, true
}

// ========== SSI ==========

// SetSSI stores the current SSI score.
func (k Keeper) SetSSI(ctx context.Context, ssi uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, ssi)
	_ = store.Set(types.SSIKey, bz)
}

// GetSSI returns the current SSI score.
func (k Keeper) GetSSI(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.SSIKey)
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}
