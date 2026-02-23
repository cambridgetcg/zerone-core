package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/schedule/types"
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

// Params returns the schedule module parameters.
func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// Process returns a single schedule process by ID.
func (k queryServer) Process(goCtx context.Context, req *types.QueryProcessRequest) (*types.QueryProcessResponse, error) {
	if req == nil || req.ProcessId == "" {
		return nil, fmt.Errorf("process_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	process, found := k.GetProcess(ctx, req.ProcessId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProcessNotFound, req.ProcessId)
	}
	return &types.QueryProcessResponse{Process: process}, nil
}

// ProcessesByCreator returns all schedule processes for a given creator.
func (k queryServer) ProcessesByCreator(goCtx context.Context, req *types.QueryProcessesByCreatorRequest) (*types.QueryProcessesByCreatorResponse, error) {
	if req == nil || req.Creator == "" {
		return nil, fmt.Errorf("creator is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	processes := k.GetProcessesByCreator(ctx, req.Creator)
	return &types.QueryProcessesByCreatorResponse{Processes: processes}, nil
}
