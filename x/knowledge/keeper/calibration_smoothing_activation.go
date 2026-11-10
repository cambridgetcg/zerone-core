package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// CalibrationSmoothingEnabled reports whether the calibration repair of
// decision zc-calibration-cliff-q5 is selected: the disproval penalty is
// smoothed to DisprovenCount*BPS/(Accepted+1), and a found calibration record
// keeps its real score of 0 instead of the score-0 sentinel reading it as
// neutral. The marker at types.CalibrationSmoothingEnabledKey is the only
// selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v2 writes it at its height. No message or param toggles
// it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) CalibrationSmoothingEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.CalibrationSmoothingEnabledKey, "calibration-smoothing")
}

// EnableCalibrationSmoothing selects the repair. It is idempotent and rewrites
// no calibration record, score or fact snapshot. Native genesis and the
// consensus-repairs-v2 carrier call it; the carrier also bumps this module's
// ConsensusVersion and owns any recompute.
func (k Keeper) EnableCalibrationSmoothing(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.CalibrationSmoothingEnabledKey, "calibration-smoothing")
}
