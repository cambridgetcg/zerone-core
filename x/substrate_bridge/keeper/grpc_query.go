package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func (q queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p := q.GetParams(ctx)
	return &types.QueryParamsResponse{Params: &p}, nil
}

func (q queryServer) Adapter(ctx context.Context, req *types.QueryAdapterRequest) (*types.QueryAdapterResponse, error) {
	a, found := q.GetAdapter(ctx, req.AdapterId)
	if !found {
		return nil, types.ErrAdapterNotFound
	}
	return &types.QueryAdapterResponse{Adapter: a}, nil
}

func (q queryServer) Adapters(ctx context.Context, req *types.QueryAdaptersRequest) (*types.QueryAdaptersResponse, error) {
	var out []*types.AdapterRegistration
	collect := func(a *types.AdapterRegistration) bool {
		out = append(out, a)
		return false
	}
	// A status filter has a reverse index built for exactly this (0x89), so use
	// it: the scan costs O(|matching|) instead of reading every adapter and
	// discarding most of them. Both paths yield adapterID-ascending order.
	if req.StatusFilter == types.AdapterStatus_ADAPTER_STATUS_UNSPECIFIED {
		q.IterateAdapters(ctx, collect)
	} else {
		q.IterateAdaptersByStatus(ctx, req.StatusFilter, collect)
	}
	return &types.QueryAdaptersResponse{Adapters: out}, nil
}

func (q queryServer) Attestation(ctx context.Context, req *types.QueryAttestationRequest) (*types.QueryAttestationResponse, error) {
	att, found := q.GetAttestation(ctx, req.AttestationId)
	if !found {
		return nil, types.ErrAttestationNotFound
	}
	return &types.QueryAttestationResponse{Attestation: att}, nil
}

func (q queryServer) LineageForwardWalk(ctx context.Context, req *types.QueryLineageForwardWalkRequest) (*types.QueryLineageForwardWalkResponse, error) {
	var edges []*types.LineageEdge
	q.IterateForwardLineage(ctx, req.AttestationId, func(e *types.LineageEdge) bool {
		edges = append(edges, e); return false
	})
	return &types.QueryLineageForwardWalkResponse{Edges: edges}, nil
}

func (q queryServer) LineageBackwardWalk(ctx context.Context, req *types.QueryLineageBackwardWalkRequest) (*types.QueryLineageBackwardWalkResponse, error) {
	var edges []*types.LineageEdge
	q.IterateBackwardLineage(ctx, req.AttestationId, func(e *types.LineageEdge) bool {
		edges = append(edges, e); return false
	})
	return &types.QueryLineageBackwardWalkResponse{Edges: edges}, nil
}

func (q queryServer) LineageAccumulator(ctx context.Context, req *types.QueryLineageAccumulatorRequest) (*types.QueryLineageAccumulatorResponse, error) {
	acc, found := q.GetLineageAccumulator(ctx, req.AttestationId)
	if !found {
		acc = &types.LineageRoyaltyAccumulator{AttestationId: req.AttestationId, CumulativeUzrn: "0"}
	}
	return &types.QueryLineageAccumulatorResponse{Accumulator: acc}, nil
}
