package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// RefundIdentityEnabled reports whether the refund-identity repair of
// zerone-core #127 is selected. The marker at types.RefundIdentityEnabledKey is
// the only selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v1 writes it at its height. No message or parameter toggles
// it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) RefundIdentityEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.RefundIdentityEnabledKey, "refund identity")
}

// EnableRefundIdentity selects the repair. It is idempotent and rewrites no
// rate limit or packet flow. Native genesis and the consensus-repairs-v1
// carrier call it; the carrier also bumps this module's ConsensusVersion.
func (k Keeper) EnableRefundIdentity(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.RefundIdentityEnabledKey, "refund identity")
}
