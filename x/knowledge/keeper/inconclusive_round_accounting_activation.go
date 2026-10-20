package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// InconclusiveRoundAccountingEnabled reports whether the panel amendments of
// decision zc-panel-unanimity-q2 are selected: a verifier who committed and
// did not reveal is slashed for the missed reveal even when the round fails
// quorum, and an INCONCLUSIVE round updates no capture-defense reputation. The
// marker at types.InconclusiveRoundAccountingEnabledKey is the only selector.
// Native genesis writes it and the named carrier upgrade consensus-repairs-v2
// writes it at its height. No message or param toggles it, and nothing clears
// it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) InconclusiveRoundAccountingEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.InconclusiveRoundAccountingEnabledKey, "inconclusive-round accounting")
}

// EnableInconclusiveRoundAccounting selects the repair. It is idempotent and
// rewrites no round, reveal, slash or reputation record. Native genesis and the
// consensus-repairs-v2 carrier call it; the carrier also bumps this module's
// ConsensusVersion.
func (k Keeper) EnableInconclusiveRoundAccounting(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.InconclusiveRoundAccountingEnabledKey, "inconclusive-round accounting")
}
