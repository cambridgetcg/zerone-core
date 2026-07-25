package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// DeclareMissingAxisBounds gives every adapter that has no AxisBounds an
// explicit, empty one (all six maxima zero) and reports how many it changed.
//
// A nil AxisBounds used to mean "unbounded" — the most permissive setting,
// reachable only by omission, and the reason a genesis-seeded adapter could be
// used to claim the remaining supply cap in one message. An empty-but-present
// bounds means the opposite and says so out loud: this adapter accepts no
// weighted claim at all until someone deliberately raises a ceiling.
//
// Zero is the deliberate floor, not a placeholder. No live traffic carries
// RecursionWeight, so nothing is refused today that succeeded yesterday, and
// raising a ceiling later is an economic decision that should be made on
// purpose rather than inherited from a nil.
//
// Idempotent: an adapter that already declares bounds is left exactly as-is,
// including one that legitimately declares zeros. Tombstoned adapters are
// skipped — they are terminal and cannot back an attestation, and WriteAdapter
// refuses to rewrite them by design.
func (k Keeper) DeclareMissingAxisBounds(ctx context.Context) (declared int, err error) {
	var pending []*types.AdapterRegistration
	k.IterateAdapters(ctx, func(a *types.AdapterRegistration) bool {
		if a.AxisBounds == nil && a.Status != types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED {
			pending = append(pending, a)
		}
		return false
	})
	// Collect first, then write: mutating the store under its own iterator is
	// undefined behaviour in the cosmos KV layer.
	for _, a := range pending {
		a.AxisBounds = &types.AxisBounds{}
		if err := k.WriteAdapter(ctx, a); err != nil {
			return declared, err
		}
		declared++
	}
	return declared, nil
}

// WriteAdapter persists an AdapterRegistration to the store.
// Returns ErrAdapterTombstoned if an adapter with the same ID is already
// tombstoned (commitment 10: forward-only tombstone). Maintains the
// (status, id) reverse index at 0x89, deleting old-status entries on
// status transition.
func (k Keeper) WriteAdapter(ctx context.Context, a *types.AdapterRegistration) error {
	if a == nil || a.AdapterId == "" {
		return types.ErrAdapterNotFound
	}
	existing, found := k.GetAdapter(ctx, a.AdapterId)
	if found && existing.Status == types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED {
		return types.ErrAdapterTombstoned
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	kvStore := sdkCtx.KVStore(k.storeKey)
	// Delete old status-index entry when status changes.
	if found && existing.Status != a.Status {
		kvStore.Delete(types.AdapterByStatusKey(uint8(existing.Status), a.AdapterId))
	}
	kvStore.Set(types.AdapterKey(a.AdapterId), k.cdc.MustMarshal(a))
	kvStore.Set(types.AdapterByStatusKey(uint8(a.Status), a.AdapterId), []byte{0x01})
	return nil
}

// GetAdapter retrieves an AdapterRegistration by adapter ID.
// Returns (nil, false) when not found.
func (k Keeper) GetAdapter(ctx context.Context, adapterID string) (*types.AdapterRegistration, bool) {
	kvStore := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	bz := kvStore.Get(types.AdapterKey(adapterID))
	if bz == nil {
		return nil, false
	}
	var a types.AdapterRegistration
	if err := k.cdc.Unmarshal(bz, &a); err != nil {
		return nil, false
	}
	return &a, true
}

// IterateAdapters walks every AdapterRegistration in insertion order.
// Returning true from cb stops iteration early.
func (k Keeper) IterateAdapters(ctx context.Context, cb func(*types.AdapterRegistration) bool) {
	kvStore := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(kvStore, types.AdapterRegistrationPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var a types.AdapterRegistration
		if err := k.cdc.Unmarshal(iter.Value(), &a); err != nil {
			continue
		}
		if cb(&a) {
			return
		}
	}
}

// IterateAdaptersByStatus walks every AdapterRegistration with the given status.
// Uses the 0x89 reverse index for O(|matching|) scans instead of O(|all|).
// Returning true from cb stops iteration early.
func (k Keeper) IterateAdaptersByStatus(ctx context.Context, status types.AdapterStatus, cb func(*types.AdapterRegistration) bool) {
	kvStore := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	prefix := append([]byte{}, types.AdapterByStatusPrefix...)
	prefix = append(prefix, uint8(status))
	iter := storetypes.KVStorePrefixIterator(kvStore, prefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		// Key layout: AdapterByStatusPrefix | status_byte | adapterID
		key := iter.Key()
		// Skip the index prefix and the status byte; the remainder is adapterID.
		// Derived from the prefix rather than hardcoded so the two cannot drift.
		offset := len(types.AdapterByStatusPrefix) + 1
		if len(key) <= offset {
			continue
		}
		adapterID := string(key[offset:])
		a, found := k.GetAdapter(ctx, adapterID)
		if !found {
			continue
		}
		if cb(a) {
			return
		}
	}
}

// SuspendAdapter transitions an ACTIVE adapter to SUSPENDED.
// Returns ErrAdapterNotFound or ErrAdapterTombstoned for terminal states.
func (k Keeper) SuspendAdapter(ctx context.Context, adapterID, reason string) error {
	a, found := k.GetAdapter(ctx, adapterID)
	if !found {
		return types.ErrAdapterNotFound
	}
	if a.Status == types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED {
		return types.ErrAdapterTombstoned
	}
	a.Status = types.AdapterStatus_ADAPTER_STATUS_SUSPENDED
	return k.WriteAdapter(ctx, a)
}

// WriteAdapterFromGov is the governance-dispatch entry point for the
// CategoryAdapterRegistration LIP class. It deserialises the proto-encoded
// AdapterRegistration bytes that were attached to the LIP and delegates to
// WriteAdapter. lipID is stored on the adapter record for on-chain audit
// (it overrides any LipId already present in the payload).
//
// Phase-0 note: full payload-attachment wiring is deferred to Phase-1.
// The method is present so that x/gov.SubstrateBridgeKeeper is satisfied
// and the gov keeper can hold a typed reference now.
func (k Keeper) WriteAdapterFromGov(ctx context.Context, lipID string, adapterProtoBytes []byte) error {
	var a types.AdapterRegistration
	if err := k.cdc.Unmarshal(adapterProtoBytes, &a); err != nil {
		return err
	}
	// Stamp the authorising LIP ID on the record for auditability.
	a.RegisteredViaLipId = lipID
	return k.WriteAdapter(ctx, &a)
}

// TombstoneAdapter permanently retires an adapter (commitment 10: forward-only).
// Sets TombstonedAtBlock = current block height. After tombstoning, WriteAdapter
// will refuse any re-registration with the same adapter ID. Every witness-reward
// escrow still pending from this adapter is cancelled — tombstoning is the
// confirmed-falsification state, and nothing was minted at settle so the
// clawback is free. (The sweep re-checks adapter status as a backstop.)
func (k Keeper) TombstoneAdapter(ctx context.Context, adapterID string) error {
	a, found := k.GetAdapter(ctx, adapterID)
	if !found {
		return types.ErrAdapterNotFound
	}
	if a.Status == types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED {
		return types.ErrAdapterTombstoned
	}
	a.Status = types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED
	a.TombstonedAtBlock = uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	if err := k.WriteAdapter(ctx, a); err != nil {
		return err
	}
	k.CancelWitnessRewardsForAdapter(ctx, adapterID, "adapter tombstoned")
	return nil
}
