package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ReviewerBalanceParamEnabled reports whether the reviewer-balance parameter
// repair is selected: the balance a reviewer must hold to commit is read from a
// knowledge parameter instead of the fixed 100 ZRN in SubmitCommitment. The
// marker at types.ReviewerBalanceParamEnabledKey is the only selector. Native
// genesis writes it. No carrier upgrade writes it yet; a consensus-repairs-v3
// carrier will, at its height. No message or param toggles it, and nothing
// clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) ReviewerBalanceParamEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.ReviewerBalanceParamEnabledKey, "reviewer-balance-param")
}

// EnableReviewerBalanceParam selects the repair. It is idempotent and rewrites
// no param, round, commitment or balance. Native genesis calls it; a
// consensus-repairs-v3 carrier will call it and bump this module's
// ConsensusVersion.
func (k Keeper) EnableReviewerBalanceParam(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.ReviewerBalanceParamEnabledKey, "reviewer-balance-param")
}
