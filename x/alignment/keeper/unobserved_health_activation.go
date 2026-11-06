package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/alignment/types"
)

// UnobservedHealthEnabled reports whether the not-observed health repair of
// decision zc-health-unknown-q1 is selected: an empty sensor reads as "not
// known yet" with neutral pacing, not as "critical". It also selects the
// first-observation critical event and DegradedThreshold hysteresis banding.
// EndBlock reads it once; pacing consumers only read the stored category.
// The marker at
// types.UnobservedHealthEnabledKey is the only selector. Native genesis writes
// it and the named carrier upgrade consensus-repairs-v2 writes it at its
// height. No message or parameter toggles it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) UnobservedHealthEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.UnobservedHealthEnabledKey, "unobserved-health")
}

// EnableUnobservedHealth selects the repair. It is idempotent and rewrites no
// observation, score, health index, correction or state record. Native genesis
// and the consensus-repairs-v2 carrier call it; the carrier also bumps this
// module's ConsensusVersion.
func (k Keeper) EnableUnobservedHealth(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.UnobservedHealthEnabledKey, "unobserved-health")
}
