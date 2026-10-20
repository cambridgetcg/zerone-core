package keeper

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// QualificationSemanticsEnabled reports whether the qualification-status
// semantics repair of zerone-core #126 is selected. The marker at
// types.QualificationSemanticsEnabledKey is the only selector. Native genesis
// writes it and the named carrier upgrade consensus-repairs-v1 writes it at its
// height. No message or parameter toggles it, and nothing clears it.
//
// The read uses its own gas meter, so a chain without the marker spends exactly
// the gas it spent before this code existed. An unreadable or malformed marker
// is an error, never legacy state; callers propagate it before choosing.
func (k Keeper) QualificationSemanticsEnabled(ctx context.Context) (bool, error) {
	ungassed := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bz := ungassed.KVStore(k.storeKey).Get(types.QualificationSemanticsEnabledKey)
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed qualification semantics marker")
	}
	return true, nil
}

// EnableQualificationSemantics selects the repair. It is idempotent and
// rewrites no adapter, attestation, source reference, bond or pending reward.
// Native genesis and the consensus-repairs-v1 carrier call it; the carrier also
// bumps this module's ConsensusVersion.
func (k Keeper) EnableQualificationSemantics(ctx context.Context) error {
	enabled, err := k.QualificationSemanticsEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey).Set(types.QualificationSemanticsEnabledKey, []byte{1})
	return nil
}
