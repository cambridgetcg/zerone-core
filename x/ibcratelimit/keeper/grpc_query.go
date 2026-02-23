package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (qs *queryServer) RateLimit(goCtx context.Context, req *types.QueryRateLimitRequest) (*types.QueryRateLimitResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.ChannelId == "" || req.Denom == "" {
		return nil, fmt.Errorf("channel_id and denom are required")
	}

	rl, found := qs.GetRateLimit(goCtx, req.ChannelId, req.Denom)
	if !found {
		return nil, types.ErrRateLimitNotFound.Wrapf("channel %s denom %s", req.ChannelId, req.Denom)
	}

	return &types.QueryRateLimitResponse{RateLimit: rl}, nil
}

func (qs *queryServer) RateLimits(goCtx context.Context, req *types.QueryRateLimitsRequest) (*types.QueryRateLimitsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	rateLimits := qs.GetAllRateLimits(goCtx)
	return &types.QueryRateLimitsResponse{RateLimits: rateLimits}, nil
}

func (qs *queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	params := qs.GetParams(goCtx)
	return &types.QueryParamsResponse{Params: params}, nil
}
