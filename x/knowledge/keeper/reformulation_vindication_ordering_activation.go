package keeper

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ReformulationVindicationOrderingEnabledStoreKey selects the reformulation-tie
// and vindication-ordering repair of zerone-core #118 for the named carrier
// upgrade consensus-repairs-v1. 0x90 is free: prefixes in types/keys.go stop at
// 0x80, the existing markers are 0x81 and 0x84-0x87, 0x82/0x83 belong to
// verifier settlement, and the KARMA design reserves 0x81-0x8F.
const ReformulationVindicationOrderingEnabledStoreKey = "\x90"

// ReformulationVindicationOrderingEnabled reports whether the repair is
// selected. The marker is the only selector. Native genesis writes it and the
// consensus-repairs-v1 carrier writes it at its height. No message or param
// toggles it, and nothing clears it. It is independent of claim integrity.
//
// The read uses its own gas meter, so a chain without the marker spends exactly
// the gas it spent before this code existed. An unreadable or malformed marker
// is an error, never legacy state; callers propagate it before choosing.
func (k Keeper) ReformulationVindicationOrderingEnabled(ctx context.Context) (bool, error) {
	ungassed := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bz, err := k.storeService.OpenKVStore(ungassed).Get([]byte(ReformulationVindicationOrderingEnabledStoreKey))
	if err != nil {
		return false, fmt.Errorf("read reformulation and vindication ordering marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed reformulation and vindication ordering marker")
	}
	return true, nil
}

// EnableReformulationVindicationOrdering selects the repair. It is idempotent
// and rewrites no augmentation, ballot, vindication or escrow record. Native
// genesis and the consensus-repairs-v1 carrier call it; the carrier also bumps
// this module's ConsensusVersion.
func (k Keeper) EnableReformulationVindicationOrdering(ctx context.Context) error {
	enabled, err := k.ReformulationVindicationOrderingEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set([]byte(ReformulationVindicationOrderingEnabledStoreKey), []byte{1}); err != nil {
		return fmt.Errorf("write reformulation and vindication ordering marker: %w", err)
	}
	return nil
}
