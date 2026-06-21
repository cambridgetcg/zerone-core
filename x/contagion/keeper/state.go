package keeper

import (
	"encoding/binary"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/contagion/types"
)

// ─── ContagionState singleton CRUD ───────────────────────────────────────────

// GetState returns the singleton ContagionState, or nil if not initialised.
func (k Keeper) GetState(ctx sdk.Context) *types.ContagionState {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.StateKey())
	if err != nil || bz == nil {
		return nil
	}
	var state types.ContagionState
	if err := proto.Unmarshal(bz, &state); err != nil {
		return nil
	}
	return &state
}

// SetState stores the singleton ContagionState.
func (k Keeper) SetState(ctx sdk.Context, state *types.ContagionState) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(state)
	if err != nil {
		panic(fmt.Sprintf("contagion: failed to marshal state: %v", err))
	}
	_ = kvStore.Set(types.StateKey(), bz)
}

// ─── Infected set (one-way append-only) ──────────────────────────────────────

// IsInfected reports whether `address` has ever received ZO. The flag is
// permanent and one-way (CONTAGION-MATH.md): once true, never false.
//
// This is the hot path for repeat transfers — a single KV Has — so the
// contagion hook adds only one store read to a normal ZO transfer when the
// recipient is already infected (the common case after the reserve depletes).
func (k Keeper) IsInfected(ctx sdk.Context, address string) bool {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.InfectedKey(address))
	return err == nil && bz != nil
}

// SetInfected marks `address` as infected, recording the block of first
// infection and the infector. This is append-only: there is deliberately no
// UnsetInfected / ClearInfected method — the flag is one-way.
func (k Keeper) SetInfected(ctx sdk.Context, address string, firstBlock uint64, infector string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	rec := &types.InfectionRecord{
		Address:    address,
		FirstBlock: firstBlock,
		Infector:   infector,
	}
	bz, err := proto.Marshal(rec)
	if err != nil {
		panic(fmt.Sprintf("contagion: failed to marshal infection record: %v", err))
	}
	_ = kvStore.Set(types.InfectedKey(address), bz)
}

// GetInfectionRecord returns the infection record for `address`, or nil if not
// infected. Used by the IsInfected query to surface first_block + infector.
func (k Keeper) GetInfectionRecord(ctx sdk.Context, address string) *types.InfectionRecord {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.InfectedKey(address))
	if err != nil || bz == nil {
		return nil
	}
	var rec types.InfectionRecord
	if err := proto.Unmarshal(bz, &rec); err != nil {
		return nil
	}
	return &rec
}

// IterateInfected iterates over every infected address. Return true from cb
// to stop. Used for genesis export and (optionally) outbreak-map queries.
func (k Keeper) IterateInfected(ctx sdk.Context, cb func(rec *types.InfectionRecord) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.InfectedKeyPrefix, prefixEndBytes(types.InfectedKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var rec types.InfectionRecord
		if err := proto.Unmarshal(iter.Value(), &rec); err != nil {
			continue
		}
		if cb(&rec) {
			break
		}
	}
}

// ─── Sneeze counter ───────────────────────────────────────────────────────────

// GetSneezeCount returns the cumulative number of sneezes that have fired.
// Stored separately for cheap reads; also mirrored in ContagionState.
func (k Keeper) GetSneezeCount(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.SneezeIndexKey())
	if err != nil || bz == nil || len(bz) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz[:8])
}

// IncrementSneezeCount atomically increments and returns the new sneeze index.
func (k Keeper) IncrementSneezeCount(ctx sdk.Context) uint64 {
	n := k.GetSneezeCount(ctx) + 1
	var bz [8]byte
	binary.BigEndian.PutUint64(bz[:], n)
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Set(types.SneezeIndexKey(), bz[:])
	return n
}

// ─── Reserve helpers ──────────────────────────────────────────────────────────

// ReserveRemaining returns the live remaining reserve as a big.Int.
func (k Keeper) ReserveRemaining(ctx sdk.Context) *big.Int {
	state := k.GetState(ctx)
	if state == nil {
		return new(big.Int)
	}
	v := new(big.Int)
	if _, ok := v.SetString(state.ReserveRemaining, 10); !ok {
		return new(big.Int)
	}
	return v
}

// ReserveDepleted reports whether the reserve can no longer fund a full sneeze.
// True iff reserve < reward_sender + reward_receiver (CONTAGION-MATH.md).
func (k Keeper) ReserveDepleted(ctx sdk.Context) bool {
	state := k.GetState(ctx)
	if state == nil || !state.Configured {
		return true
	}
	reserve := new(big.Int)
	reserve.SetString(state.ReserveRemaining, 10)
	total := new(big.Int)
	rs := new(big.Int)
	rr := new(big.Int)
	rs.SetString(state.RewardSender, 10)
	rr.SetString(state.RewardReceiver, 10)
	total.Add(rs, rr)
	return reserve.Cmp(total) < 0
}

// ─── Genesis ──────────────────────────────────────────────────────────────────

// InitGenesis initialises the contagion module from genesis. If the genesis
// state carries a configured ContagionState, the module starts configured
// (and renounced — authority empty). Pre-seeded infected addresses are set.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	if gs == nil {
		return
	}
	if gs.State != nil {
		k.SetState(ctx, gs.State)
		// If genesis says configured, the module is renounced from block 0.
		if gs.State.Configured {
			k.authority = ""
		}
	}
	for _, rec := range gs.InfectedAddresses {
		if rec == nil {
			continue
		}
		k.SetInfected(ctx, rec.Address, rec.FirstBlock, rec.Infector)
	}
}

// ExportGenesis exports the contagion module state for genesis export.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	gs := &types.GenesisState{
		State: k.GetState(ctx),
	}
	k.IterateInfected(ctx, func(rec *types.InfectionRecord) bool {
		gs.InfectedAddresses = append(gs.InfectedAddresses, rec)
		return false
	})
	return gs
}

// ─── helpers ───────────────────────────────────────────────────────────────────

// prefixEndBytes returns the end key for prefix iteration (last byte incr).
// Mirrors x/tokens/keeper/state.go.
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
