package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/evidence_mgmt/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = &queryServer{}

func (q queryServer) QueryEvidence(goCtx context.Context, req *types.QueryEvidenceRequest) (*types.QueryEvidenceResponse, error) {
	if req == nil || req.Id == "" {
		return nil, fmt.Errorf("evidence id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	evidence, found := q.GetEvidence(ctx, req.Id)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrEvidenceNotFound, req.Id)
	}
	return &types.QueryEvidenceResponse{Evidence: evidence}, nil
}

func (q queryServer) QueryEvidenceBySubmitter(goCtx context.Context, req *types.QueryEvidenceBySubmitterRequest) (*types.QueryEvidenceBySubmitterResponse, error) {
	if req == nil || req.Submitter == "" {
		return nil, fmt.Errorf("submitter is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	evidences := q.GetEvidenceBySubmitter(ctx, req.Submitter)
	return &types.QueryEvidenceBySubmitterResponse{Evidences: evidences}, nil
}

func (q queryServer) QueryCustodyChain(goCtx context.Context, req *types.QueryCustodyChainRequest) (*types.QueryCustodyChainResponse, error) {
	if req == nil || req.EvidenceId == "" {
		return nil, fmt.Errorf("evidence_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	evidence, found := q.GetEvidence(ctx, req.EvidenceId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrEvidenceNotFound, req.EvidenceId)
	}
	return &types.QueryCustodyChainResponse{Entries: evidence.ChainOfCustody}, nil
}

func (q queryServer) QueryVerifications(goCtx context.Context, req *types.QueryVerificationsRequest) (*types.QueryVerificationsResponse, error) {
	if req == nil || req.EvidenceId == "" {
		return nil, fmt.Errorf("evidence_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	verifications := q.GetVerificationsByEvidence(ctx, req.EvidenceId)
	return &types.QueryVerificationsResponse{Verifications: verifications}, nil
}

func (q queryServer) QueryParams(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
