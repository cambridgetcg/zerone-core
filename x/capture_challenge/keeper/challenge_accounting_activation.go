package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

// ChallengeAccountingEnabled reports whether the rejected-stake accounting and
// expiry-refund repair of zerone-core #121 is selected. The marker at
// types.ChallengeAccountingEnabledKey is the only selector. Native genesis
// writes it and the named carrier upgrade consensus-repairs-v1 writes it at its
// height. No message or parameter toggles it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) ChallengeAccountingEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.ChallengeAccountingEnabledKey, "challenge accounting")
}

// EnableChallengeAccounting selects the repair. It is idempotent and rewrites
// no challenge or bounty pool. Native genesis and the consensus-repairs-v1
// carrier call it; the carrier also bumps this module's ConsensusVersion.
func (k Keeper) EnableChallengeAccounting(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.ChallengeAccountingEnabledKey, "challenge accounting")
}
