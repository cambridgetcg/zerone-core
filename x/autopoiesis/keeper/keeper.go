package keeper

import (
	"context"
	"encoding/json"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/autopoiesis/types"
)

// Keeper manages the autopoiesis module's state.
type Keeper struct {
	storeService    store.KVStoreService
	cdc             codec.BinaryCodec
	authority       string
	stakingKeeper   types.StakingKeeper
	knowledgeKeeper types.KnowledgeKeeper
	emergencyKeeper types.EmergencyKeeper
}

// NewKeeper creates a new autopoiesis module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	stakingKeeper types.StakingKeeper,
) Keeper {
	return Keeper{
		storeService:  storeService,
		cdc:           cdc,
		authority:     authority,
		stakingKeeper: stakingKeeper,
	}
}

// SetKnowledgeKeeper sets the knowledge keeper (post-init to break circular deps).
func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) {
	k.knowledgeKeeper = kk
}

// SetEmergencyKeeper sets the emergency keeper (post-init to break circular deps).
func (k *Keeper) SetEmergencyKeeper(ek types.EmergencyKeeper) {
	k.emergencyKeeper = ek
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-scoped logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
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

// GetMultiplier returns the current multiplier BPS value for a path.
// This is the primary method consumed by other modules (staking, knowledge, vesting_rewards).
func (k Keeper) GetMultiplier(ctx context.Context, path string) (uint64, error) {
	if !k.IsActive(ctx) {
		return types.BPSScale, nil // default 1.0x when not active
	}
	ms, found := k.GetMultiplierState(ctx, path)
	if !found {
		return types.BPSScale, nil
	}
	return ms.CurrentBps, nil
}

// SuggestAdjustment receives alignment correction suggestions.
// For now, this logs the suggestion. Future versions will feed it into the
// adaptation loop as an additional signal.
func (k Keeper) SuggestAdjustment(ctx context.Context, parameter, direction string, magnitude uint64) error {
	k.Logger(ctx).Info("alignment suggestion received",
		"parameter", parameter,
		"direction", direction,
		"magnitude", magnitude,
	)
	return nil
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}

	state := types.AutopoiesisState{
		Activated:       genState.Activated,
		CurrentEpoch:    0,
		LastEpochHeight: 0,
	}
	k.SetState(ctx, &state)

	for _, m := range genState.Multipliers {
		if m == nil {
			continue
		}
		k.SetMultiplierState(ctx, m)
	}

	for _, s := range genState.Snapshots {
		if s == nil {
			continue
		}
		k.SetEpochSnapshot(ctx, s)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	state := k.GetState(ctx)
	multipliers := k.GetAllMultipliers(ctx)
	snapshots := k.GetAllSnapshots(ctx)
	return &types.GenesisState{
		Params:      params,
		Multipliers: multipliers,
		Snapshots:   snapshots,
		Activated:   state.Activated,
	}
}

// GetAllMultipliers returns all stored multiplier states.
func (k Keeper) GetAllMultipliers(ctx context.Context) []*types.MultiplierState {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.MultiplierPrefix, prefixEndBytes(types.MultiplierPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var multipliers []*types.MultiplierState
	for ; iter.Valid(); iter.Next() {
		var ms types.MultiplierState
		if err := json.Unmarshal(iter.Value(), &ms); err != nil {
			continue
		}
		multipliers = append(multipliers, &ms)
	}
	return multipliers
}

// GetAllSnapshots returns all stored epoch snapshots.
func (k Keeper) GetAllSnapshots(ctx context.Context) []*types.EpochSnapshot {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.SnapshotPrefix, prefixEndBytes(types.SnapshotPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var snapshots []*types.EpochSnapshot
	for ; iter.Valid(); iter.Next() {
		var s types.EpochSnapshot
		if err := json.Unmarshal(iter.Value(), &s); err != nil {
			continue
		}
		snapshots = append(snapshots, &s)
	}
	return snapshots
}
