package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// FactUseReceipt is a read-only current-context/current-epoch point query. It
// implements the generated RPC, not an extension or an inherited stub. A found
// record describes a signed self-report, never independently measured readership.
func (q *queryServer) FactUseReceipt(ctx context.Context, req *types.QueryFactUseReceiptRequest) (*types.QueryFactUseReceiptResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if _, err := types.CanonicalFactUseConsumer(req.Consumer); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := types.ValidateFactUseFactID(req.FactId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	p, err := q.keeper.getFactUseParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	epoch, err := types.FactUseEpoch(height, p.FitnessEpochBlocks)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	r, found, err := q.keeper.GetFactUseReceipt(ctx, epoch, req.Consumer, req.FactId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if found && (r.UseHeight > uint64(height) || r.RatingHeight > uint64(height)) {
		return nil, status.Error(codes.Internal, "receipt is newer than observed context")
	}
	return &types.QueryFactUseReceiptResponse{Receipt: r, Found: found, Epoch: epoch, SnapshotBlockHeight: uint64(height)}, nil
}
