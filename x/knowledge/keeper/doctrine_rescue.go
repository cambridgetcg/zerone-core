package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ResurrectDoctrineFacts restores every doctrine fact to a fully-alive,
// VERIFIED state and clears any at-risk clock. Paired with the metabolism
// exemption (ProcessMetabolism skips doctrine, x/knowledge/keeper/metabolism.go),
// it ends the born-starving lifecycle: doctrine lives by the creed pin +
// amendment LIP, never by disuse.
//
// Background: BuildDoctrineFact omitted Energy, so the 47 genesis doctrine
// facts were born at energy 0 and went AT_RISK at epoch 1, EXPIRED by ~epoch 6,
// and were scheduled to PRUNE at block ~260,000 (~2026-07-16) — the chain would
// display its own constitution as extinct-by-disuse, and the doctrine would
// leave the training corpus. This restores them.
//
// Idempotent and deterministic: a doctrine fact already VERIFIED at full energy
// is rewritten identically. Single ordered pass over the fact store. Returns the
// number of doctrine facts rewritten. Run from the doctrine-metabolism-exempt-v1
// upgrade handler; safe to re-run.
func (k Keeper) ResurrectDoctrineFacts(ctx context.Context) (int, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	energyCap := params.MetabolismEnergyCap
	height := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())

	var doctrine []*types.Fact
	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		if fact.Category == types.DoctrineCategory || fact.MethodId == types.DoctrineMethodId {
			doctrine = append(doctrine, fact)
		}
		return false
	})

	n := 0
	for _, fact := range doctrine {
		fact.Status = types.FactStatus_FACT_STATUS_VERIFIED
		fact.Energy = energyCap
		fact.AtRiskSinceEpoch = 0
		fact.EnergyLastUpdated = height
		if err := k.SetFact(ctx, fact); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
