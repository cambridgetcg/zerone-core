package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// CountHaltsForReason is the verbatim legacy body from 77521a0d.
// It deliberately ignores reason. The governance entry point selects a scoped
// adapter when enabled; this body never reads a marker or decodes a payload.
func (a *GovEmergencyAdapter) CountHaltsForReason(ctx context.Context, _ string) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var count uint64
	a.k.IterateCeremonies(sdkCtx, func(c *types.EmergencyCeremony) bool {
		if c.Type == string(types.CeremonyHalt) && c.Phase == string(types.PhaseFinalized) {
			count++
		}
		return false
	})
	return count
}
