package keeper

import (
	"bytes"
	"context"
	"fmt"

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
// is scoped inclusively to [sinceBlock, current height]. Unknown/prose reasons
// and malformed evidence return ErrHaltEvidence, never zero or a panic. They
// block forward progress without manufacturing positive rollback evidence.
func (a *GovEmergencyAdapter) CountResearchFundHalts(ctx context.Context, sinceBlock uint64) (count uint64, resultErr error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() < 0 {
		return 0, fmt.Errorf("%w: negative current height", types.ErrHaltEvidence)
	}
	currentHeight := uint64(sdkCtx.BlockHeight())
	store := a.k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.CeremonyKeyPrefix, prefixEndBytes(types.CeremonyKeyPrefix))
	if err != nil {
		return 0, fmt.Errorf("%w: iterate ceremonies: %v", types.ErrHaltEvidence, err)
	}
	defer func() {
		if err := iter.Close(); err != nil && resultErr == nil {
			count, resultErr = 0, fmt.Errorf("%w: close ceremonies: %v", types.ErrHaltEvidence, err)
		}
	}()
	for ; iter.Valid(); iter.Next() {
		var c types.EmergencyCeremony
		if err := proto.Unmarshal(iter.Value(), &c); err != nil {
			return 0, fmt.Errorf("%w: decode ceremony: %v", types.ErrHaltEvidence, err)
		}
		if !bytes.Equal(iter.Key(), types.CeremonyKey(c.Id)) {
			return 0, fmt.Errorf("%w: ceremony key/id mismatch", types.ErrHaltEvidence)
		}
		if c.Type != string(types.CeremonyHalt) || c.Phase != string(types.PhaseFinalized) {
			continue
		}
		if c.StartBlock < sinceBlock || c.StartBlock > currentHeight {
			continue
		}
		var proposal types.EmergencyHaltProposal
		if err := unmarshalProposal(c.ProposalData, &proposal); err != nil {
			return 0, fmt.Errorf("%w: decode halt %q: %v", types.ErrHaltEvidence, c.Id, err)
		}
		if proposal.Id != c.Id {
			return 0, fmt.Errorf("%w: halt %q payload id mismatch", types.ErrHaltEvidence, c.Id)
		}
		if proposal.Reason != types.HaltReasonResearchFund {
			return 0, fmt.Errorf("%w: halt %q has an unclassified reason", types.ErrHaltEvidence, c.Id)
		}
		count++
	}
	if err := terminalIteratorError(iter); err != nil {
		return 0, fmt.Errorf("%w: iterate ceremonies: %v", types.ErrHaltEvidence, err)
	}
	return count, nil
}

// IsHalted reports application-transaction quarantine, including the
// same-block post-resume release latch.
func (a *GovEmergencyAdapter) IsHalted(ctx context.Context) bool {
	return a.k.IsHalted(ctx)
}
