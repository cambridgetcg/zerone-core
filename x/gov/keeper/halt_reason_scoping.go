package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	"github.com/zerone-chain/zerone/x/gov/types"
)

// HaltReasonScopingEnabled exposes the foundation reader to governance's
// message, query and module entry points. Only those entry points read it.
func (k Keeper) HaltReasonScopingEnabled(ctx context.Context) (bool, error) {
	if k.emergencyKeeper == nil {
		return false, nil
	}
	return k.emergencyKeeper.HaltReasonScopingEnabled(ctx)
}

// WithHaltReasonScoping returns an invocation-local keeper, never mutating the
// shared keeper. The marker bool has already been read once, before any write.
// Legacy calls keep their original dependency and exact store/gas behavior.
func (k Keeper) WithHaltReasonScoping(enabled bool) Keeper {
	if scoped, ok := k.emergencyKeeper.(scopedHaltCounter); ok {
		k.emergencyKeeper = scoped.EmergencyKeeper
	}
	if enabled && k.emergencyKeeper != nil {
		k.emergencyKeeper = scopedHaltCounter{
			EmergencyKeeper: k.emergencyKeeper,
			phaseStart: func(ctx sdk.Context) uint64 {
				return k.GetResearchFundGovernanceState(ctx).PhaseStartedAtBlock
			},
		}
	}
	return k
}

// scopedHaltCounter preserves the pinned research_spend.go interface. Window
// filtering and decoding live in emergency; the lower bound is read live so
// another transition in this same block cannot leave a stale phase boundary.
// Its uint64-only forward gate adds one for unknown evidence (saturating).
// Rollback instead uses only the valid count, independently of the unknown flag.
type scopedHaltCounter struct {
	types.EmergencyKeeper
	phaseStart func(sdk.Context) uint64
}

func (s scopedHaltCounter) CountHaltsForReason(ctx context.Context, reason string) uint64 {
	if reason != emergencytypes.HaltReasonResearchFund {
		return 1 // no unclassified query may invent an absence of evidence
	}
	count, unknown := s.CountResearchFundHalts(ctx, s.phaseStart(sdk.UnwrapSDKContext(ctx)))
	if unknown && count < ^uint64(0) {
		count++
	}
	return count
}

func (k Keeper) haltReasonScopingSelected() bool {
	_, enabled := k.emergencyKeeper.(scopedHaltCounter)
	return enabled
}
