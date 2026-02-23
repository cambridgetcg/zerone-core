package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/home/types"
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

func (q queryServer) Home(goCtx context.Context, req *types.QueryHomeRequest) (*types.QueryHomeResponse, error) {
	if req == nil || req.HomeId == "" {
		return nil, fmt.Errorf("home_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	home, found := q.GetHome(ctx, req.HomeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrHomeNotFound, req.HomeId)
	}
	return &types.QueryHomeResponse{Home: home}, nil
}

func (q queryServer) HomesByOwner(goCtx context.Context, req *types.QueryHomesByOwnerRequest) (*types.QueryHomesByOwnerResponse, error) {
	if req == nil || req.Owner == "" {
		return nil, fmt.Errorf("owner is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	homeIDs := q.GetHomesByOwner(ctx, req.Owner)
	var homes []*types.AgentHome
	for _, id := range homeIDs {
		home, found := q.GetHome(ctx, id)
		if found {
			homes = append(homes, home)
		}
	}
	return &types.QueryHomesByOwnerResponse{Homes: homes}, nil
}

func (q queryServer) Keys(goCtx context.Context, req *types.QueryKeysRequest) (*types.QueryKeysResponse, error) {
	if req == nil || req.HomeId == "" {
		return nil, fmt.Errorf("home_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	keys := q.GetKeysByHome(ctx, req.HomeId)
	return &types.QueryKeysResponse{Keys: keys}, nil
}

func (q queryServer) Sessions(goCtx context.Context, req *types.QuerySessionsRequest) (*types.QuerySessionsResponse, error) {
	if req == nil || req.HomeId == "" {
		return nil, fmt.Errorf("home_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	sessions := q.GetSessionsByHome(ctx, req.HomeId)
	return &types.QuerySessionsResponse{Sessions: sessions}, nil
}

func (q queryServer) Alerts(goCtx context.Context, req *types.QueryAlertsRequest) (*types.QueryAlertsResponse, error) {
	if req == nil || req.HomeId == "" {
		return nil, fmt.Errorf("home_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	allAlerts := q.GetAlertsByHome(ctx, req.HomeId)

	if req.UnacknowledgedOnly {
		var filtered []*types.Alert
		for _, a := range allAlerts {
			if !a.Acknowledged {
				filtered = append(filtered, a)
			}
		}
		return &types.QueryAlertsResponse{Alerts: filtered}, nil
	}
	return &types.QueryAlertsResponse{Alerts: allAlerts}, nil
}

func (q queryServer) SpendingLimits(goCtx context.Context, req *types.QuerySpendingLimitsRequest) (*types.QuerySpendingLimitsResponse, error) {
	if req == nil || req.HomeId == "" {
		return nil, fmt.Errorf("home_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	limits := q.GetSpendingLimitsByHome(ctx, req.HomeId)
	return &types.QuerySpendingLimitsResponse{Limits: limits}, nil
}

func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
