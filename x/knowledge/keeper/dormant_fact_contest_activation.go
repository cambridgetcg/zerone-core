package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// DormantFactContestEnabled reports whether the dormant-fact contest repair is
// selected: a fact that lost standing through disuse stays open to challenge
// and contradiction, and a pruned conjecture stays an open question. The marker
// at types.DormantFactContestEnabledKey is the only selector. Native genesis
// writes it. No carrier upgrade writes it yet; a consensus-repairs-v3 carrier
// will, at its height. No message or param toggles it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) DormantFactContestEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.DormantFactContestEnabledKey, "dormant-fact-contest")
}

// EnableDormantFactContest selects the repair. It is idempotent and rewrites no
// fact, conjecture, energy or status. Native genesis calls it; a
// consensus-repairs-v3 carrier will call it and bump this module's
// ConsensusVersion.
func (k Keeper) EnableDormantFactContest(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.DormantFactContestEnabledKey, "dormant-fact-contest")
}
