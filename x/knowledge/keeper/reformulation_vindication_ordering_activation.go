package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ReformulationVindicationOrderingEnabledStoreKey is the marker key as a
// string, kept so code written against this earlier keeper constant
// (zerone-core #118) builds unchanged.
//
// Deprecated: use types.ReformulationVindicationOrderingEnabledKey.
var ReformulationVindicationOrderingEnabledStoreKey = string(types.ReformulationVindicationOrderingEnabledKey)

// ReformulationVindicationOrderingEnabled reports whether the reformulation-tie
// and vindication-ordering repair of zerone-core #118 is selected. The marker
// at types.ReformulationVindicationOrderingEnabledKey is the only selector.
// Native genesis writes it and the named carrier upgrade consensus-repairs-v1
// writes it at its height. No message or param toggles it, and nothing clears
// it. It is independent of claim integrity.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) ReformulationVindicationOrderingEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.ReformulationVindicationOrderingEnabledKey, "reformulation and vindication ordering")
}

// EnableReformulationVindicationOrdering selects the repair. It is idempotent
// and rewrites no augmentation, ballot, vindication or escrow record. Native
// genesis and the consensus-repairs-v1 carrier call it; the carrier also bumps
// this module's ConsensusVersion.
func (k Keeper) EnableReformulationVindicationOrdering(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.ReformulationVindicationOrderingEnabledKey, "reformulation and vindication ordering")
}
