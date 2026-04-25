package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/dialectic/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	keeper Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer { return &queryServer{keeper: k} }

var _ types.QueryServer = &queryServer{}

func (q *queryServer) Signature(ctx context.Context, req *types.QuerySignatureRequest) (*types.QuerySignatureResponse, error) {
	if req == nil || req.FactId == "" {
		return nil, fmt.Errorf("fact_id required")
	}
	return &types.QuerySignatureResponse{Signature: q.keeper.ComposeSignature(ctx, req.FactId)}, nil
}

func (q *queryServer) DomainSignature(ctx context.Context, req *types.QueryDomainSignatureRequest) (*types.QueryDomainSignatureResponse, error) {
	if req == nil || req.Domain == "" {
		return nil, fmt.Errorf("domain required")
	}
	return &types.QueryDomainSignatureResponse{Dialectic: q.keeper.ComposeDomainDialectic(ctx, req.Domain)}, nil
}

func (q *queryServer) PairwiseDisagreement(ctx context.Context, req *types.QueryPairwiseRequest) (*types.QueryPairwiseResponse, error) {
	if req == nil || req.AgentA == "" || req.AgentB == "" {
		return nil, fmt.Errorf("agent_a and agent_b required")
	}
	return &types.QueryPairwiseResponse{Pair: q.keeper.ComposePairwise(ctx, req.AgentA, req.AgentB)}, nil
}

func (q *queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p := q.keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: &p}, nil
}
