package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/compute_pool/types"
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

// Params returns the module parameters.
func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// Provider returns a single compute provider by address.
func (k queryServer) Provider(goCtx context.Context, req *types.QueryProviderRequest) (*types.QueryProviderResponse, error) {
	if req == nil || req.Address == "" {
		return nil, fmt.Errorf("address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	provider, found := k.GetProvider(ctx, req.Address)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, req.Address)
	}
	return &types.QueryProviderResponse{Provider: provider}, nil
}

// Providers returns all compute providers, optionally filtered by service type.
func (k queryServer) Providers(goCtx context.Context, req *types.QueryProvidersRequest) (*types.QueryProvidersResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	allProviders := k.GetAllProviders(ctx)

	// Filter by service_type if specified.
	if req.ServiceType != "" {
		var filtered []*types.ComputeProvider
		for _, p := range allProviders {
			if p.ServiceType == req.ServiceType {
				filtered = append(filtered, p)
			}
		}
		return &types.QueryProvidersResponse{Providers: filtered}, nil
	}

	return &types.QueryProvidersResponse{Providers: allProviders}, nil
}

// Credit returns compute credit information for a validator address.
func (k queryServer) Credit(goCtx context.Context, req *types.QueryCreditRequest) (*types.QueryCreditResponse, error) {
	if req == nil || req.ValidatorAddr == "" {
		return nil, fmt.Errorf("validator_addr is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	credit, found := k.GetCredit(ctx, req.ValidatorAddr)
	if !found {
		// Return an empty credit rather than an error.
		credit = &types.ComputeCredit{
			ValidatorAddr: req.ValidatorAddr,
			Balance:       0,
			EarnedTotal:   0,
			RedeemedTotal: 0,
		}
	}
	return &types.QueryCreditResponse{Credit: credit}, nil
}
