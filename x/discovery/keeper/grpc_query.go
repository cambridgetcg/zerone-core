package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/discovery/types"
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

func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

func (k queryServer) Profile(goCtx context.Context, req *types.QueryProfileRequest) (*types.QueryProfileResponse, error) {
	if req == nil || req.Address == "" {
		return nil, fmt.Errorf("address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	profile, found := k.GetProfile(ctx, req.Address)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrAgentNotFound, req.Address)
	}
	return &types.QueryProfileResponse{Profile: profile}, nil
}

func (k queryServer) Search(goCtx context.Context, req *types.QuerySearchRequest) (*types.QuerySearchResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	profiles := k.SearchProfiles(ctx, req.Domain, req.CapabilityType, req.MinReputation)
	return &types.QuerySearchResponse{Profiles: profiles}, nil
}
