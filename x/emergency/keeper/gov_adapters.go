package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	govtypes "github.com/zerone-chain/zerone/x/gov/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// GovEmergencyAdapter wraps the emergency Keeper to satisfy the
// gov module's EmergencyKeeper interface.
type GovEmergencyAdapter struct {
	k Keeper
}

// NewGovEmergencyAdapter returns an adapter for the gov module.
func NewGovEmergencyAdapter(k Keeper) *GovEmergencyAdapter {
	return &GovEmergencyAdapter{k: k}
}

// Compile-time interface check.
var _ govtypes.EmergencyKeeper = (*GovEmergencyAdapter)(nil)

// CountHaltsForReason preserves the legacy all-reasons, all-history count until
// HaltReasonFilterEnabled is selected by a coordinated upgrade. Once enabled,
// only finalized halts with an exact, nonempty recorded reason and StartBlock
// in [sinceBlock, current height] count. Governance supplies its current phase's
// start, not a rolling TTL. StartBlock is the ceremony initiation height, not
// its finalization height; the ceremony schema has no finalization timestamp.
// Malformed in-window halt payloads fail closed rather than authorizing either
// rollback or a forward phase transition from an invented zero count.
func (a *GovEmergencyAdapter) CountHaltsForReason(ctx context.Context, reason string, sinceBlock uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	enabled, err := a.k.HaltReasonFilterEnabled(sdkCtx)
	if err != nil {
		panic(err)
	}
	var currentHeight uint64
	if enabled {
		if sdkCtx.BlockHeight() < 0 {
			panic("cannot count reason-specific halts at a negative height")
		}
		currentHeight = uint64(sdkCtx.BlockHeight())
		if reason == "" || sinceBlock > currentHeight {
			return 0
		}
	}
	var count uint64
	a.k.IterateCeremonies(sdkCtx, func(c *types.EmergencyCeremony) bool {
		if c.Type != string(types.CeremonyHalt) || c.Phase != string(types.PhaseFinalized) {
			return false
		}
		if enabled {
			if c.StartBlock < sinceBlock || c.StartBlock > currentHeight {
				return false
			}
			var proposal types.EmergencyHaltProposal
			if err := unmarshalProposal(c.ProposalData, &proposal); err != nil {
				panic(fmt.Errorf("decode halt %q for reason count: %w", c.Id, err))
			}
			if proposal.Id != c.Id {
				panic(fmt.Sprintf("halt %q reason payload has mismatched proposal id %q", c.Id, proposal.Id))
			}
			if proposal.Reason != reason {
				return false
			}
		}
		count++
		return false
	})
	return count
}

// IsHalted reports application-transaction quarantine, including the
// same-block post-resume release latch.
func (a *GovEmergencyAdapter) IsHalted(ctx context.Context) bool {
	return a.k.IsHalted(ctx)
}
