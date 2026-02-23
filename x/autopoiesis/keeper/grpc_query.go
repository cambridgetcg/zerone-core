package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/autopoiesis/types"
)

// queryServer implements types.QueryServer.
type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params := q.GetParams(goCtx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// Multiplier returns a single multiplier state by path.
func (q queryServer) Multiplier(goCtx context.Context, req *types.QueryMultiplierRequest) (*types.QueryMultiplierResponse, error) {
	ms, found := q.GetMultiplierState(goCtx, req.Path)
	if !found {
		return &types.QueryMultiplierResponse{}, nil
	}
	return &types.QueryMultiplierResponse{Multiplier: ms}, nil
}

// AllMultipliers returns all multiplier states.
func (q queryServer) AllMultipliers(goCtx context.Context, _ *types.QueryAllMultipliersRequest) (*types.QueryAllMultipliersResponse, error) {
	multipliers := q.GetAllMultipliers(goCtx)
	return &types.QueryAllMultipliersResponse{Multipliers: multipliers}, nil
}

// EpochSnapshot returns the snapshot for a specific epoch.
func (q queryServer) EpochSnapshot(goCtx context.Context, req *types.QueryEpochSnapshotRequest) (*types.QueryEpochSnapshotResponse, error) {
	s, found := q.GetEpochSnapshot(goCtx, req.Epoch)
	if !found {
		return &types.QueryEpochSnapshotResponse{}, nil
	}
	return &types.QueryEpochSnapshotResponse{Snapshot: s}, nil
}

// SSI returns the current System Stability Index.
func (q queryServer) SSI(goCtx context.Context, _ *types.QuerySSIRequest) (*types.QuerySSIResponse, error) {
	ssi := q.GetSSI(goCtx)
	params := q.GetParams(goCtx)
	category := types.ClassifySSI(ssi, params)
	return &types.QuerySSIResponse{
		SsiScore:    ssi,
		SsiCategory: category,
	}, nil
}
