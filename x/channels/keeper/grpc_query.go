package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/channels/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns a QueryServer implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Channel returns a specific payment channel.
func (q queryServer) Channel(goCtx context.Context, req *types.QueryChannelRequest) (*types.QueryChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ch, found := q.Keeper.GetChannel(ctx, req.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	return &types.QueryChannelResponse{Channel: ch}, nil
}

// ChannelsByPayer returns all channels for a payer.
func (q queryServer) ChannelsByPayer(goCtx context.Context, req *types.QueryByPayerRequest) (*types.QueryByPayerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	channels := q.Keeper.GetChannelsByPayer(ctx, req.Payer)
	return &types.QueryByPayerResponse{Channels: channels}, nil
}

// ChannelsByReceiver returns all channels for a receiver.
func (q queryServer) ChannelsByReceiver(goCtx context.Context, req *types.QueryByReceiverRequest) (*types.QueryByReceiverResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	channels := q.Keeper.GetChannelsByReceiver(ctx, req.Receiver)
	return &types.QueryByReceiverResponse{Channels: channels}, nil
}

// Dispute returns the dispute for a channel.
func (q queryServer) Dispute(goCtx context.Context, req *types.QueryDisputeRequest) (*types.QueryDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	dispute, found := q.Keeper.GetDispute(ctx, req.ChannelId)
	if !found {
		return &types.QueryDisputeResponse{}, nil
	}
	return &types.QueryDisputeResponse{Dispute: dispute}, nil
}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
