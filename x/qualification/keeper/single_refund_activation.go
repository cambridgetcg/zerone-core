package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/qualification/types"
)

// SingleRefundEnabled reports whether the single-refund repair of zerone-core
// #120 is selected. The marker at types.SingleRefundEnabledKey is the only
// selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v1 writes it at its height. No message or parameter toggles
// it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) SingleRefundEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.SingleRefundEnabledKey, "single-refund")
}

// EnableSingleRefund selects the repair. It is idempotent and rewrites no
// qualification, stake or endorsement. Native genesis and the
// consensus-repairs-v1 carrier call it; the carrier also bumps this module's
// ConsensusVersion.
func (k Keeper) EnableSingleRefund(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.SingleRefundEnabledKey, "single-refund")
}
