package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (ms *msgServer) AddRateLimit(goCtx context.Context, msg *types.MsgAddRateLimit) (*types.MsgAddRateLimitResponse, error) {
	if msg.Authority != ms.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check for existing rate limit
	if _, found := ms.GetRateLimit(goCtx, msg.ChannelId, msg.Denom); found {
		return nil, fmt.Errorf("rate limit already exists for channel %s denom %s", msg.ChannelId, msg.Denom)
	}

	rl := &types.RateLimit{
		ChannelId:    msg.ChannelId,
		Denom:        msg.Denom,
		MaxSend:      msg.MaxSend,
		MaxRecv:      msg.MaxRecv,
		WindowBlocks: msg.WindowBlocks,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  0, // will be set on first packet or BeginBlock
	}

	ms.SetRateLimit(goCtx, rl)

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.ibcratelimit.rate_limit_added",
			sdk.NewAttribute("channel_id", msg.ChannelId),
			sdk.NewAttribute("denom", msg.Denom),
			sdk.NewAttribute("max_send", msg.MaxSend),
			sdk.NewAttribute("max_recv", msg.MaxRecv),
			sdk.NewAttribute("window_blocks", fmt.Sprintf("%d", msg.WindowBlocks)),
		),
	)

	return &types.MsgAddRateLimitResponse{}, nil
}

func (ms *msgServer) RemoveRateLimit(goCtx context.Context, msg *types.MsgRemoveRateLimit) (*types.MsgRemoveRateLimitResponse, error) {
	if msg.Authority != ms.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if _, found := ms.GetRateLimit(goCtx, msg.ChannelId, msg.Denom); !found {
		return nil, types.ErrRateLimitNotFound.Wrapf("channel %s denom %s", msg.ChannelId, msg.Denom)
	}

	ms.DeleteRateLimit(goCtx, msg.ChannelId, msg.Denom)

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.ibcratelimit.rate_limit_removed",
			sdk.NewAttribute("channel_id", msg.ChannelId),
			sdk.NewAttribute("denom", msg.Denom),
		),
	)

	return &types.MsgRemoveRateLimitResponse{}, nil
}

func (ms *msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != ms.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ms.SetParams(goCtx, msg.Params)

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.ibcratelimit.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
