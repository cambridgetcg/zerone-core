package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// GetEmergencyTransitionHold returns the durable review gate created by an
// observed transaction quarantine.
func (k Keeper) GetEmergencyTransitionHold(
	ctx sdk.Context,
) (*types.EmergencyTransitionHold, bool, error) {
	bz := ctx.KVStore(k.storeKey).Get(types.EmergencyTransitionHoldKey)
	if bz == nil {
		return nil, false, nil
	}
	var hold types.EmergencyTransitionHold
	if err := proto.Unmarshal(bz, &hold); err != nil {
		return nil, false, fmt.Errorf(
			"decode custom-governance emergency transition hold: %w",
			err,
		)
	}
	if err := types.ValidateEmergencyTransitionHold(&hold); err != nil {
		return nil, false, fmt.Errorf(
			"invalid persisted custom-governance emergency transition hold: %w",
			err,
		)
	}
	return &hold, true, nil
}

// RequireNoEmergencyTransitionHold gates every state-changing custom
// governance message except the explicitly safe forward-only LIP withdrawal.
func (k Keeper) RequireNoEmergencyTransitionHold(ctx sdk.Context) error {
	hold, found, err := k.GetEmergencyTransitionHold(ctx)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	return errorsmod.Wrapf(
		types.ErrEmergencyTransitionHold,
		"incident lineage commitment %x (%d incidents) requires a future named-upgrade queue reconciliation",
		hold.IncidentLineageSha256,
		hold.IncidentCount,
	)
}

// SetEmergencyTransitionHold persists a validated hold. It is exported for
// deterministic genesis initialization and application-module coordination;
// no transaction message exposes this setter.
func (k Keeper) SetEmergencyTransitionHold(
	ctx sdk.Context,
	hold *types.EmergencyTransitionHold,
) error {
	if hold == nil {
		return fmt.Errorf("custom-governance emergency transition hold cannot be nil")
	}
	if err := types.ValidateEmergencyTransitionHold(hold); err != nil {
		return err
	}
	bz, err := proto.Marshal(hold)
	if err != nil {
		return fmt.Errorf(
			"encode custom-governance emergency transition hold: %w",
			err,
		)
	}
	ctx.KVStore(k.storeKey).Set(types.EmergencyTransitionHoldKey, bz)
	return nil
}

// EnsureEmergencyTransitionHold creates the review gate once and commits every
// later incident to a fixed-size rolling digest. Comparing latest_incident_id
// makes the once-per-block call idempotent without allowing an unbounded list
// to accumulate in consensus state.
func (k Keeper) EnsureEmergencyTransitionHold(
	ctx sdk.Context,
	incidentID string,
) (*types.EmergencyTransitionHold, bool, error) {
	if existing, found, err := k.GetEmergencyTransitionHold(ctx); err != nil {
		return nil, false, err
	} else if found {
		if existing.LatestIncidentId == incidentID {
			return existing, false, nil
		}
		if existing.IncidentCount == ^uint64(0) {
			return nil, false, fmt.Errorf(
				"custom-governance emergency transition hold incident count overflow",
			)
		}
		existing.LatestIncidentId = incidentID
		existing.IncidentCount++
		existing.IncidentLineageSha256 = types.AdvanceEmergencyIncidentLineage(
			existing.IncidentLineageSha256,
			incidentID,
		)
		if err := k.SetEmergencyTransitionHold(ctx, existing); err != nil {
			return nil, false, err
		}
		return existing, true, nil
	}
	if ctx.BlockHeight() <= 0 {
		return nil, false, fmt.Errorf(
			"custom-governance emergency transition hold requires a positive block height",
		)
	}
	hold := &types.EmergencyTransitionHold{
		IncidentId:            incidentID,
		ActivatedAtBlock:      uint64(ctx.BlockHeight()),
		LatestIncidentId:      incidentID,
		IncidentCount:         1,
		IncidentLineageSha256: types.AdvanceEmergencyIncidentLineage(nil, incidentID),
	}
	if err := k.SetEmergencyTransitionHold(ctx, hold); err != nil {
		return nil, false, err
	}
	return hold, true, nil
}

func (k Keeper) clearEmergencyTransitionHold(ctx sdk.Context) {
	ctx.KVStore(k.storeKey).Delete(types.EmergencyTransitionHoldKey)
}
