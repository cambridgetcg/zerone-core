package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/bvm/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns a QueryServer implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) Contract(goCtx context.Context, req *types.QueryContractRequest) (*types.QueryContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	contract, found := q.Keeper.GetContract(ctx, req.Address)
	if !found {
		return nil, types.ErrContractNotFound
	}
	return &types.QueryContractResponse{Contract: contract}, nil
}

func (q queryServer) ContractsByCreator(goCtx context.Context, req *types.QueryByCreatorRequest) (*types.QueryByCreatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	contracts := q.Keeper.GetContractsByCreator(ctx, req.Creator)
	if contracts == nil {
		contracts = []*types.DeployedContract{}
	}
	return &types.QueryByCreatorResponse{Contracts: contracts}, nil
}

func (q queryServer) ContractState(goCtx context.Context, req *types.QueryContractStateRequest) (*types.QueryContractStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	value, _ := q.Keeper.GetContractState(ctx, req.Address, req.Key)
	return &types.QueryContractStateResponse{Value: value}, nil
}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
