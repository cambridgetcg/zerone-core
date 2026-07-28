package keeper

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zerone-chain/zerone/x/training_provenance/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	types.UnimplementedQueryServer
	keeper Keeper
}

// NewQueryServerImpl returns a query server for the training_provenance module.
func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{keeper: k}
}

// ProvenanceCertificate synthesizes the certificate for the named manifest
// from the keepers' current state.
func (q queryServer) ProvenanceCertificate(ctx context.Context, req *types.QueryProvenanceCertificateRequest) (*types.QueryProvenanceCertificateResponse, error) {
	if req == nil || req.ManifestId == "" {
		return nil, status.Error(codes.InvalidArgument, "manifest_id required")
	}
	cert, err := q.keeper.BuildCertificate(ctx, req.ManifestId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &types.QueryProvenanceCertificateResponse{Certificate: cert}, nil
}

// InTotoStatement returns the certificate in the standard in-toto Statement
// v1 envelope. It is unsigned: signature policy belongs to the off-chain
// producer and verifier, not to consensus.
func (q queryServer) InTotoStatement(ctx context.Context, req *types.QueryInTotoStatementRequest) (*types.InTotoStatementV1, error) {
	if req == nil || req.ManifestId == "" {
		return nil, status.Error(codes.InvalidArgument, "manifest_id required")
	}
	statement, err := q.keeper.BuildInTotoStatement(ctx, req.ManifestId)
	switch {
	case errors.Is(err, errInTotoManifestNotFound):
		return nil, status.Error(codes.NotFound, err.Error())
	case errors.Is(err, errInTotoProjectionUnavailable):
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, errInTotoDataLoss):
		return nil, status.Error(codes.DataLoss, err.Error())
	case err != nil:
		return nil, status.Error(codes.Internal, err.Error())
	}
	return statement, nil
}
