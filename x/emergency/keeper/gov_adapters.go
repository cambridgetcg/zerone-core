package keeper

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

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

// HaltReasonScopingEnabled delegates to the sole foundation marker reader.
func (a *GovEmergencyAdapter) HaltReasonScopingEnabled(ctx context.Context) (bool, error) {
	return a.k.HaltReasonScopingEnabled(ctx)
}

// CountResearchFundHalts is the enabled evidence API. The caller has already
// selected the marker once at its entry point. StartBlock (not finalization)
// is scoped inclusively to [sinceBlock, current height]. The valid count and
// unknown flag are independent: unclassified/malformed records and surfaced
// iterator faults block forward progress but never erase valid evidence.
// Invalid envelopes anywhere in history are unknown, even for non-halts:
// their type/window cannot be trusted. No marker is read here, so no marker
// fault can originate here; entry points propagate those before calling us.
func (a *GovEmergencyAdapter) CountResearchFundHalts(ctx context.Context, sinceBlock uint64) (count uint64, unknown bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() < 0 {
		return 0, true
	}
	currentHeight := uint64(sdkCtx.BlockHeight())
	store := a.k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.CeremonyKeyPrefix, prefixEndBytes(types.CeremonyKeyPrefix))
	if err != nil {
		return 0, true
	}
	defer func() {
		if err := iter.Close(); err != nil {
			unknown = true
		}
	}()
	for ; iter.Valid(); iter.Next() {
		var c types.EmergencyCeremony
		if err := proto.Unmarshal(iter.Value(), &c); err != nil {
			unknown = true
			continue
		}
		if !bytes.Equal(iter.Key(), types.CeremonyKey(c.Id)) {
			unknown = true
			continue
		}
		if c.Type != string(types.CeremonyHalt) || c.Phase != string(types.PhaseFinalized) {
			continue
		}
		if c.StartBlock < sinceBlock || c.StartBlock > currentHeight {
			continue
		}
		var proposal types.EmergencyHaltProposal
		if err := unmarshalProposal(c.ProposalData, &proposal); err != nil {
			unknown = true
			continue
		}
		if proposal.Id != c.Id {
			unknown = true
			continue
		}
		if proposal.Reason != types.HaltReasonResearchFund {
			unknown = true
			continue
		}
		count++
	}
	if err := terminalIteratorError(iter); err != nil {
		unknown = true
	}
	return count, unknown
}

// IsHalted reports application-transaction quarantine, including the
// same-block post-resume release latch.
func (a *GovEmergencyAdapter) IsHalted(ctx context.Context) bool {
	return a.k.IsHalted(ctx)
}
