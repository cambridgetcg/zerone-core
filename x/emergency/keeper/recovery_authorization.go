package keeper

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// GetRecoveryAuthorization returns the one finalized Guardian capability for
// an exact SDK-governance recovery proposal, if present.
func (k Keeper) GetRecoveryAuthorization(
	ctx context.Context,
) (*types.EmergencyRecoveryAuthorization, bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.RecoveryAuthorizationKey)
	if err != nil {
		return nil, false, fmt.Errorf(
			"read emergency recovery authorization: %w",
			err,
		)
	}
	if bz == nil {
		return nil, false, nil
	}
	var authorization types.EmergencyRecoveryAuthorization
	if err := proto.Unmarshal(bz, &authorization); err != nil {
		return nil, false, fmt.Errorf(
			"decode emergency recovery authorization: %w",
			err,
		)
	}
	if err := types.ValidateRecoveryAuthorization(&authorization); err != nil {
		return nil, false, fmt.Errorf(
			"invalid persisted emergency recovery authorization: %w",
			err,
		)
	}
	return &authorization, true, nil
}

func (k Keeper) setRecoveryAuthorization(
	ctx context.Context,
	authorization *types.EmergencyRecoveryAuthorization,
) error {
	if err := types.ValidateRecoveryAuthorization(authorization); err != nil {
		return err
	}
	bz, err := proto.Marshal(authorization)
	if err != nil {
		return fmt.Errorf("encode emergency recovery authorization: %w", err)
	}
	if err := k.storeService.OpenKVStore(ctx).Set(
		types.RecoveryAuthorizationKey,
		bz,
	); err != nil {
		return fmt.Errorf("persist emergency recovery authorization: %w", err)
	}
	return nil
}

// ClearRecoveryAuthorization removes a capability once its incident is closed.
// It is deliberately not exposed through a Msg service.
func (k Keeper) ClearRecoveryAuthorization(ctx context.Context) error {
	if err := k.storeService.OpenKVStore(ctx).Delete(
		types.RecoveryAuthorizationKey,
	); err != nil {
		return fmt.Errorf("clear emergency recovery authorization: %w", err)
	}
	return nil
}

// MarkRecoveryAuthorizationTerminal closes an authorization after its exact
// SDK governance proposal reaches a terminal result. It can only narrow
// authority and is intentionally not exposed as a transaction message.
func (k Keeper) MarkRecoveryAuthorizationTerminal(
	ctx context.Context,
	proposalID uint64,
	actionSHA256 string,
	outcome string,
) error {
	authorization, found, err := k.GetRecoveryAuthorization(ctx)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("emergency recovery authorization is absent")
	}
	if authorization.SdkGovProposalId != proposalID ||
		authorization.ActionSha256 != actionSHA256 {
		return fmt.Errorf(
			"recovery authorization does not match terminal proposal %d action %s",
			proposalID,
			actionSHA256,
		)
	}
	if authorization.Outcome != "" {
		return fmt.Errorf(
			"recovery authorization is already terminal with outcome %q",
			authorization.Outcome,
		)
	}
	switch outcome {
	case "passed", "failed", "rejected", "revoked":
	default:
		return fmt.Errorf("invalid recovery authorization outcome %q", outcome)
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() <= 0 {
		return fmt.Errorf(
			"terminal recovery authorization requires positive block height",
		)
	}
	authorization.TerminalAtBlock = uint64(sdkCtx.BlockHeight())
	authorization.Outcome = outcome
	return k.setRecoveryAuthorization(ctx, authorization)
}
