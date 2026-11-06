package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

// IntegerVarianceEnabled reports whether the integer timing-variance repair of
// zerone-core #119 is selected. The marker at types.IntegerVarianceEnabledKey
// is the only selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v1 writes it at its height. No message or parameter toggles
// it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) IntegerVarianceEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.IntegerVarianceEnabledKey, "integer variance")
}

// EnableIntegerVariance selects the repair. It is idempotent and rewrites no
// reputation, capture metric or verification history. Native genesis and the
// consensus-repairs-v1 carrier call it; the carrier also bumps this module's
// ConsensusVersion.
func (k Keeper) EnableIntegerVariance(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.IntegerVarianceEnabledKey, "integer variance")
}
