package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/icaauth/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (qs *queryServer) Account(goCtx context.Context, req *types.QueryAccountRequest) (*types.QueryAccountResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Owner == "" || req.ConnectionId == "" {
		return nil, fmt.Errorf("owner and connection_id are required")
	}

	acct, found := qs.GetRemoteAccountByConnection(goCtx, req.Owner, req.ConnectionId)
	if !found {
		return nil, types.ErrNotRegistered.Wrapf("owner %s connection %s", req.Owner, req.ConnectionId)
	}

	return &types.QueryAccountResponse{Account: acct}, nil
}

func (qs *queryServer) Accounts(goCtx context.Context, req *types.QueryAccountsRequest) (*types.QueryAccountsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Owner == "" {
		return nil, fmt.Errorf("owner is required")
	}

	accounts := qs.GetRemoteAccounts(goCtx, req.Owner)
	return &types.QueryAccountsResponse{Accounts: accounts}, nil
}

func (qs *queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	params := qs.GetParams(goCtx)
	return &types.QueryParamsResponse{Params: params}, nil
}
