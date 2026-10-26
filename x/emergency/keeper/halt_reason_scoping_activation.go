package keeper

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// HaltReasonScopingEnabled reports whether the halt-reason scoping repair of
// zerone-core #125 is selected. The marker at types.HaltReasonScopingEnabledKey
// is the only selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v1 writes it at its height. No message or parameter toggles
// it, and nothing clears it.
//
// The read uses its own gas meter, so a chain without the marker spends exactly
// the gas it spent before this code existed. An unreadable or malformed marker
// is an error, never legacy state; callers propagate it before choosing.
func (k Keeper) HaltReasonScopingEnabled(ctx context.Context) (bool, error) {
	ungassed := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bz, err := k.storeService.OpenKVStore(ungassed).Get(types.HaltReasonScopingEnabledKey)
	if err != nil {
		return false, fmt.Errorf("read halt-reason scoping marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed halt-reason scoping marker")
	}
	return true, nil
}

// EnableHaltReasonScoping selects the repair. It is idempotent and rewrites no
// ceremony, audit entry or governance record. Native genesis and the
// consensus-repairs-v1 carrier call it; the carrier also bumps this module's
// ConsensusVersion.
func (k Keeper) EnableHaltReasonScoping(ctx context.Context) error {
	enabled, err := k.HaltReasonScopingEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set(types.HaltReasonScopingEnabledKey, []byte{1}); err != nil {
		return fmt.Errorf("write halt-reason scoping marker: %w", err)
	}
	return nil
}

// InitHaltReasonScopingGenesis writes the marker only when the genesis field is
// true; a false or absent field leaves legacy semantics. The module's
// InitGenesis calls it after Keeper.InitGenesis. The marker is wired here, not
// in keeper.go, because the authority-geometry manifest pins keeper.go's
// SHA-256 (source anchor custom-emergency-electorate).
func (k Keeper) InitHaltReasonScopingGenesis(ctx context.Context, gs *types.GenesisState) error {
	if !gs.HaltReasonScopingEnabled {
		return nil
	}
	return k.EnableHaltReasonScoping(ctx)
}

// ExportHaltReasonScopingGenesis records the marker in an exported genesis.
// The module's ExportGenesis calls it on Keeper.ExportGenesis's result.
func (k Keeper) ExportHaltReasonScopingGenesis(ctx context.Context, gs *types.GenesisState) error {
	enabled, err := k.HaltReasonScopingEnabled(ctx)
	if err != nil {
		return err
	}
	gs.HaltReasonScopingEnabled = enabled
	return nil
}
