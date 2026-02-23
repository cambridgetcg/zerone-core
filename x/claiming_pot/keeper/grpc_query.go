package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/claiming_pot/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = &queryServer{}

func (q queryServer) QueryPot(goCtx context.Context, req *types.QueryPotRequest) (*types.QueryPotResponse, error) {
	if req == nil || req.Id == "" {
		return nil, fmt.Errorf("pot id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pot, found := q.GetPot(ctx, req.Id)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrPotNotFound, req.Id)
	}
	return &types.QueryPotResponse{Pot: pot}, nil
}

func (q queryServer) QueryAllPots(goCtx context.Context, req *types.QueryAllPotsRequest) (*types.QueryAllPotsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pots := q.GetAllPots(ctx)
	return &types.QueryAllPotsResponse{Pots: pots}, nil
}

func (q queryServer) QueryClaimable(goCtx context.Context, req *types.QueryClaimableRequest) (*types.QueryClaimableResponse, error) {
	if req == nil || req.PotId == "" || req.Address == "" {
		return nil, fmt.Errorf("pot_id and address are required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	pot, found := q.GetPot(ctx, req.PotId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrPotNotFound, req.PotId)
	}

	// If already claimed, return 0
	if _, exists := q.GetClaim(ctx, req.PotId, req.Address); exists {
		return &types.QueryClaimableResponse{Amount: "0"}, nil
	}

	currentBlock := uint64(ctx.BlockHeight())
	claimable := CalculateClaimable(pot, currentBlock)
	return &types.QueryClaimableResponse{Amount: claimable.String()}, nil
}

func (q queryServer) QueryClaims(goCtx context.Context, req *types.QueryClaimsRequest) (*types.QueryClaimsResponse, error) {
	if req == nil || req.PotId == "" {
		return nil, fmt.Errorf("pot_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	claims := q.GetClaimsByPot(ctx, req.PotId)
	return &types.QueryClaimsResponse{Claims: claims}, nil
}

func (q queryServer) QueryParams(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
