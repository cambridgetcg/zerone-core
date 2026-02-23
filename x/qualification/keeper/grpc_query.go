package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/qualification/types"
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

func (k queryServer) Qualification(goCtx context.Context, req *types.QueryQualificationRequest) (*types.QueryQualificationResponse, error) {
	if req == nil || req.Validator == "" || req.Domain == "" {
		return nil, fmt.Errorf("validator and domain are required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	q, found := k.GetQualification(ctx, req.Validator, req.Domain)
	if !found {
		return nil, fmt.Errorf("%w: %s/%s", types.ErrQualificationNotFound, req.Validator, req.Domain)
	}
	return &types.QueryQualificationResponse{Qualification: q}, nil
}

func (k queryServer) QualificationsByDomain(goCtx context.Context, req *types.QueryByDomainRequest) (*types.QueryByDomainResponse, error) {
	if req == nil || req.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	validators := k.GetValidatorsByDomain(ctx, req.Domain)
	var qualifications []*types.DomainQualification
	for _, v := range validators {
		q, found := k.GetQualification(ctx, v, req.Domain)
		if found {
			qualifications = append(qualifications, q)
		}
	}
	return &types.QueryByDomainResponse{Qualifications: qualifications}, nil
}

func (k queryServer) QualificationsByValidator(goCtx context.Context, req *types.QueryByValidatorRequest) (*types.QueryByValidatorResponse, error) {
	if req == nil || req.Validator == "" {
		return nil, fmt.Errorf("validator is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	var qualifications []*types.DomainQualification
	k.IterateQualifications(ctx, func(q *types.DomainQualification) bool {
		if q.Validator == req.Validator {
			qualifications = append(qualifications, q)
		}
		return false
	})
	return &types.QueryByValidatorResponse{Qualifications: qualifications}, nil
}

func (k queryServer) Endorsements(goCtx context.Context, req *types.QueryEndorsementsRequest) (*types.QueryEndorsementsResponse, error) {
	if req == nil || req.Validator == "" || req.Domain == "" {
		return nil, fmt.Errorf("validator and domain are required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	endorsements := k.GetEndorsementsByTarget(ctx, req.Validator, req.Domain)
	return &types.QueryEndorsementsResponse{Endorsements: endorsements}, nil
}

func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
