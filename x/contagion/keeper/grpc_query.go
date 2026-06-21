package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/contagion/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = (*queryServer)(nil)

// ContagionState returns the singleton on-chain contagion state. Lets anyone
// verify the reserve size and that the formula is frozen (configured == true,
// authority == "").
func (q *queryServer) ContagionState(goCtx context.Context, _ *types.QueryContagionStateRequest) (*types.QueryContagionStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryContagionStateResponse{State: q.GetState(ctx)}, nil
}

// IsInfected reports whether an address has ever received ZO, plus the block
// of first infection and the infector (for indexers / leaderboards).
func (q *queryServer) IsInfected(goCtx context.Context, req *types.QueryIsInfectedRequest) (*types.QueryIsInfectedResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	rec := q.GetInfectionRecord(ctx, req.Address)
	if rec == nil {
		return &types.QueryIsInfectedResponse{Infected: false}, nil
	}
	return &types.QueryIsInfectedResponse{
		Infected:   true,
		FirstBlock: rec.FirstBlock,
		Infector:   rec.Infector,
	}, nil
}
