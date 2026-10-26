package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/emergency/types"
)

// HaltReasonScopingEnabled reports whether the halt-reason scoping repair of
// zerone-core #125 is selected. The marker at types.HaltReasonScopingEnabledKey
// is the only selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v1 writes it at its height. No message or parameter toggles
// it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) HaltReasonScopingEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.HaltReasonScopingEnabledKey, "halt-reason scoping")
}

// EnableHaltReasonScoping selects the repair. It is idempotent and rewrites no
// ceremony, audit entry or governance record. Native genesis and the
// consensus-repairs-v1 carrier call it; the carrier also bumps this module's
// ConsensusVersion.
func (k Keeper) EnableHaltReasonScoping(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.HaltReasonScopingEnabledKey, "halt-reason scoping")
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
