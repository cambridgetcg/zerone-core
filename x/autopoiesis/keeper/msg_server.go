package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/autopoiesis/types"
)

// msgServer implements types.MsgServer.
type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// UpdateParams handles governance parameter updates.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrUnauthorized, k.GetAuthority(), msg.Authority)
	}
	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidParams, err)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(goCtx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.autopoiesis.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}

// Activate handles activation/deactivation of the autopoiesis module.
func (k msgServer) Activate(goCtx context.Context, msg *types.MsgActivateAutopoiesis) (*types.MsgActivateAutopoiesisResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrUnauthorized, k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	state := k.GetState(goCtx)
	state.Activated = msg.Activate
	if msg.Activate && state.LastEpochHeight == 0 {
		state.LastEpochHeight = uint64(ctx.BlockHeight())
	}
	k.SetState(goCtx, state)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.autopoiesis.activated",
			sdk.NewAttribute("authority", msg.Authority),
			sdk.NewAttribute("activated", fmt.Sprintf("%t", msg.Activate)),
		),
	)

	return &types.MsgActivateAutopoiesisResponse{}, nil
}

// OverrideMultiplier force-sets a multiplier value (governance only).
func (k msgServer) OverrideMultiplier(goCtx context.Context, msg *types.MsgOverrideMultiplier) (*types.MsgOverrideMultiplierResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrUnauthorized, k.GetAuthority(), msg.Authority)
	}
	if !types.ValidPaths[msg.Path] {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidPath, msg.Path)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	ms, found := k.GetMultiplierState(goCtx, msg.Path)
	if !found {
		return nil, fmt.Errorf("%w: multiplier %s not found", types.ErrInvalidPath, msg.Path)
	}

	ms.CurrentBps = msg.Value
	ms.TargetBps = msg.Value
	ms.LastUpdated = uint64(ctx.BlockHeight())
	k.SetMultiplierState(goCtx, ms)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.autopoiesis.multiplier_overridden",
			sdk.NewAttribute("authority", msg.Authority),
			sdk.NewAttribute("path", msg.Path),
			sdk.NewAttribute("value", fmt.Sprintf("%d", msg.Value)),
		),
	)

	return &types.MsgOverrideMultiplierResponse{}, nil
}

// FreezeMultiplier sets or clears the freeze flag on a multiplier (governance only).
func (k msgServer) FreezeMultiplier(goCtx context.Context, msg *types.MsgFreezeMultiplier) (*types.MsgFreezeMultiplierResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrUnauthorized, k.GetAuthority(), msg.Authority)
	}
	if !types.ValidPaths[msg.Path] {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidPath, msg.Path)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetMultiplierFrozen(goCtx, msg.Path, msg.Frozen)

	// Also update the multiplier state's frozen flag.
	ms, found := k.GetMultiplierState(goCtx, msg.Path)
	if found {
		ms.Frozen = msg.Frozen
		k.SetMultiplierState(goCtx, ms)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.autopoiesis.multiplier_frozen",
			sdk.NewAttribute("authority", msg.Authority),
			sdk.NewAttribute("path", msg.Path),
			sdk.NewAttribute("frozen", fmt.Sprintf("%t", msg.Frozen)),
		),
	)

	return &types.MsgFreezeMultiplierResponse{}, nil
}
