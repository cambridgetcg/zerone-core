package keeper

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zerone-chain/zerone/x/training_provenance/types"
)

// ProvenanceCertificateV2 synthesizes the predicate v2 certificate. Status
// codes match ProvenanceCertificate.
func (q queryServer) ProvenanceCertificateV2(ctx context.Context, req *types.QueryProvenanceCertificateV2Request) (*types.QueryProvenanceCertificateV2Response, error) {
	if req == nil || req.ManifestId == "" {
		return nil, status.Error(codes.InvalidArgument, "manifest_id required")
	}
	cert, err := q.keeper.BuildCertificateV2(ctx, req.ManifestId)
	switch {
	case errors.Is(err, errCertificateManifestNotFound):
		return nil, status.Error(codes.NotFound, err.Error())
	case errors.Is(err, errCertificateDataLoss):
		return nil, status.Error(codes.DataLoss, err.Error())
	case err != nil:
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryProvenanceCertificateV2Response{Certificate: cert}, nil
}

// InTotoStatementV2 returns the v2 certificate in an unsigned in-toto
// Statement v1. Status codes match InTotoStatement.
func (q queryServer) InTotoStatementV2(ctx context.Context, req *types.QueryInTotoStatementV2Request) (*types.InTotoStatementV1PredicateV2, error) {
	if req == nil || req.ManifestId == "" {
		return nil, status.Error(codes.InvalidArgument, "manifest_id required")
	}
	statement, err := q.keeper.BuildInTotoStatementV2(ctx, req.ManifestId)
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
