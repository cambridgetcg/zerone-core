package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/disputes/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = &queryServer{}

func (q queryServer) Dispute(goCtx context.Context, req *types.QueryDisputeRequest) (*types.QueryDisputeResponse, error) {
	if req == nil || req.Id == "" {
		return nil, fmt.Errorf("dispute id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	dispute, found := q.GetDispute(ctx, req.Id)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, req.Id)
	}
	return &types.QueryDisputeResponse{Dispute: dispute}, nil
}

func (q queryServer) DisputesByTarget(goCtx context.Context, req *types.QueryByTargetRequest) (*types.QueryByTargetResponse, error) {
	if req == nil || req.TargetId == "" {
		return nil, fmt.Errorf("target_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	disputes := q.GetDisputesByTarget(ctx, req.TargetId)
	return &types.QueryByTargetResponse{Disputes: disputes}, nil
}

func (q queryServer) ActiveDisputes(goCtx context.Context, req *types.QueryActiveRequest) (*types.QueryActiveResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	disputes := q.Keeper.GetActiveDisputes(ctx)
	return &types.QueryActiveResponse{Disputes: disputes}, nil
}

func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
