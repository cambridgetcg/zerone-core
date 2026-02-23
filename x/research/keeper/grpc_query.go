package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/research/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

// NewQueryServerImpl returns a query server implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Research returns a single research submission with its reviews.
func (q queryServer) Research(goCtx context.Context, req *types.QueryResearchRequest) (*types.QueryResearchResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	research, found := q.Keeper.GetResearch(ctx, req.ResearchId)
	if !found {
		return nil, types.ErrSubmissionNotFound
	}

	reviews := q.Keeper.GetReviewsForResearch(ctx, req.ResearchId)

	return &types.QueryResearchResponse{
		Research: research,
		Reviews:  reviews,
	}, nil
}

// Submissions returns research submissions filtered by status and/or domain.
func (q queryServer) Submissions(goCtx context.Context, req *types.QuerySubmissionsRequest) (*types.QuerySubmissionsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var results []*types.Research
	q.Keeper.IterateResearches(ctx, func(r *types.Research) bool {
		if req.Status != "" && r.Status != req.Status {
			return false
		}
		if req.Domain != "" && r.Domain != req.Domain {
			return false
		}
		results = append(results, r)
		return false
	})

	return &types.QuerySubmissionsResponse{Submissions: results}, nil
}

// Bounty returns a single bounty by ID.
func (q queryServer) Bounty(goCtx context.Context, req *types.QueryBountyRequest) (*types.QueryBountyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	bounty, found := q.Keeper.GetBounty(ctx, req.BountyId)
	if !found {
		return nil, types.ErrBountyNotFound
	}

	return &types.QueryBountyResponse{Bounty: bounty}, nil
}

// Params returns the module parameters and treasury balance.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{
		Params:          params,
		TreasuryBalance: &types.TreasuryBalance{Balance: q.Keeper.GetTreasuryBalance(ctx)},
	}, nil
}
