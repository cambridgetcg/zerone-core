package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/governance_synthesis/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	types.UnimplementedQueryServer
	keeper Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{keeper: k}
}

func (q queryServer) SystemHealth(ctx context.Context, _ *types.QuerySystemHealthRequest) (*types.QuerySystemHealthResponse, error) {
	return &types.QuerySystemHealthResponse{Health: q.keeper.BuildHealth(ctx)}, nil
}
