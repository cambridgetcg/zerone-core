package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ZeroConfidenceFloorEnabled reports whether the confidence repairs of
// decision zc-pr37-frontier-fixes (fixes b and c of zerone-core #37, commit
// 827b5308) are selected: a fact clamps to its dependency floor whenever cited
// support resolved, so a zero floor is a floor and not the absence of one, and
// a fact at zero confidence gains no minimum growth on the passage of time
// alone. The marker at types.ZeroConfidenceFloorEnabledKey is the only
// selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v2 writes it at its height. No message or param toggles
// it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) ZeroConfidenceFloorEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.ZeroConfidenceFloorEnabledKey, "zero-confidence floor")
}

// EnableZeroConfidenceFloor selects the repair. It is idempotent and rewrites
// no fact, confidence or dependency floor. Native genesis and the
// consensus-repairs-v2 carrier call it; the carrier also bumps this module's
// ConsensusVersion.
func (k Keeper) EnableZeroConfidenceFloor(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.ZeroConfidenceFloorEnabledKey, "zero-confidence floor")
}
