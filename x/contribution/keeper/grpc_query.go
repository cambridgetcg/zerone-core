package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	corestoretypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/zerone-chain/zerone/x/contribution/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	keeper *Keeper
}

// NewQueryServer constructs the Query server.
func NewQueryServer(k *Keeper) types.QueryServer {
	return &queryServer{keeper: k}
}

var _ types.QueryServer = (*queryServer)(nil)

func (q *queryServer) Contribution(ctx context.Context, req *types.QueryContributionRequest) (*types.QueryContributionResponse, error) {
	if req == nil || len(req.Id) != 32 {
		return nil, status.Error(codes.InvalidArgument, "id must be 32 bytes")
	}
	c, ok := q.keeper.GetContribution(ctx, req.Id)
	if !ok {
		return nil, status.Error(codes.NotFound, "contribution not found")
	}
	return &types.QueryContributionResponse{Contribution: c}, nil
}

func (q *queryServer) ContributionsByContributor(ctx context.Context, req *types.QueryByContributorRequest) (*types.QueryByContributorResponse, error) {
	if req == nil || req.Contributor == "" {
		return nil, status.Error(codes.InvalidArgument, "contributor required")
	}
	store := q.keeper.storeService.OpenKVStore(ctx)
	addrBz := []byte(req.Contributor)

	prefix := append([]byte{}, types.ByContributorKey...)
	prefix = append(prefix, uvarintBytes(uint64(len(addrBz)))...)
	prefix = append(prefix, addrBz...)

	contribs, pageRes, err := scanIndexAndLoad(ctx, q.keeper, store, prefix, req.Pagination)
	if err != nil {
		return nil, err
	}
	return &types.QueryByContributorResponse{Contributions: contribs, Pagination: pageRes}, nil
}

func (q *queryServer) ContributionsByClass(ctx context.Context, req *types.QueryByClassRequest) (*types.QueryByClassResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	store := q.keeper.storeService.OpenKVStore(ctx)
	prefix := append([]byte{}, types.ByClassKey...)
	prefix = append(prefix, uint32BE(uint32(req.Class))...)

	contribs, pageRes, err := scanIndexAndLoad(ctx, q.keeper, store, prefix, req.Pagination)
	if err != nil {
		return nil, err
	}
	return &types.QueryByClassResponse{Contributions: contribs, Pagination: pageRes}, nil
}

func (q *queryServer) ContributionsByPhase(ctx context.Context, req *types.QueryByPhaseRequest) (*types.QueryByPhaseResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	store := q.keeper.storeService.OpenKVStore(ctx)
	prefix := append([]byte{}, types.ByPhaseKey...)
	prefix = append(prefix, uint32BE(uint32(req.Phase))...)

	contribs, pageRes, err := scanIndexAndLoad(ctx, q.keeper, store, prefix, req.Pagination)
	if err != nil {
		return nil, err
	}
	return &types.QueryByPhaseResponse{Contributions: contribs, Pagination: pageRes}, nil
}

// scanIndexAndLoad iterates a secondary-index prefix using KVStore directly
// (matching the x/work_creed iteration pattern), extracts the trailing
// 32-byte contribution_id from each key, and loads the primary record.
// Pagination is naive (in-memory subset of all matches); upgrade to true
// range-paging when query volume requires it.
func scanIndexAndLoad(ctx context.Context, k *Keeper, store corestoretypes.KVStore, prefix []byte, _ *query.PageRequest) ([]*types.Contribution, *query.PageResponse, error) {
	end := prefixEndBytes(prefix)
	iter, err := store.Iterator(prefix, end)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "iterator: %v", err)
	}
	defer iter.Close()

	var out []*types.Contribution
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		// Trailing 32 bytes = contribution_id.
		if len(key) < 32 {
			continue
		}
		id := key[len(key)-32:]
		c, ok := k.GetContribution(ctx, id)
		if !ok {
			continue
		}
		out = append(out, c)
	}
	// Naive single-page response; upgrade later.
	pageRes := &query.PageResponse{Total: uint64(len(out))}
	return out, pageRes, nil
}

// prefixEndBytes increments the last byte of the prefix (carrying as
// needed) to produce the exclusive upper bound of the iterator range.
// Returns nil if the prefix is all-0xFF (interpreted by store as
// "iterate to end").
func prefixEndBytes(prefix []byte) []byte {
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}
