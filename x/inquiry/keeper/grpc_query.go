package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/inquiry/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	keeper Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer { return &queryServer{keeper: k} }

var _ types.QueryServer = &queryServer{}

func (q *queryServer) Inquiry(ctx context.Context, req *types.QueryInquiryRequest) (*types.QueryInquiryResponse, error) {
	if req == nil || req.Id == "" {
		return nil, fmt.Errorf("id required")
	}
	inq, ok := q.keeper.GetInquiry(ctx, req.Id)
	if !ok {
		return nil, fmt.Errorf("%w: %s", types.ErrInquiryNotFound, req.Id)
	}
	return &types.QueryInquiryResponse{Inquiry: inq}, nil
}

func (q *queryServer) Inquiries(ctx context.Context, req *types.QueryInquiriesRequest) (*types.QueryInquiriesResponse, error) {
	if req == nil {
		req = &types.QueryInquiriesRequest{}
	}
	limit := req.Limit
	if limit == 0 || limit > 200 {
		limit = 50
	}
	out := make([]*types.Inquiry, 0, limit)
	skipped := req.StartAfterId == ""
	var nextCursor string
	walk := func(inq *types.Inquiry) bool {
		if !skipped {
			if inq.Id == req.StartAfterId {
				skipped = true
			}
			return false
		}
		if uint32(len(out)) >= limit {
			nextCursor = inq.Id
			return true
		}
		if req.Status != types.InquiryStatus_INQUIRY_STATUS_UNSPECIFIED && inq.Status != req.Status {
			return false
		}
		out = append(out, inq)
		return false
	}
	_ = q.keeper.IterateAllInquiries(ctx, walk)
	return &types.QueryInquiriesResponse{Inquiries: out, NextStartAfterId: nextCursor}, nil
}

func (q *queryServer) InquiriesByDomain(ctx context.Context, req *types.QueryByDomainRequest) (*types.QueryByDomainResponse, error) {
	if req == nil || req.Domain == "" {
		return nil, fmt.Errorf("domain required")
	}
	out := []*types.Inquiry{}
	_ = q.keeper.IterateInquiriesByDomain(ctx, req.Domain, func(inq *types.Inquiry) bool {
		out = append(out, inq)
		return false
	})
	return &types.QueryByDomainResponse{Inquiries: out}, nil
}

func (q *queryServer) InquiriesByAsker(ctx context.Context, req *types.QueryByAskerRequest) (*types.QueryByAskerResponse, error) {
	if req == nil || req.Asker == "" {
		return nil, fmt.Errorf("asker required")
	}
	out := []*types.Inquiry{}
	_ = q.keeper.IterateInquiriesByAsker(ctx, req.Asker, func(inq *types.Inquiry) bool {
		out = append(out, inq)
		return false
	})
	return &types.QueryByAskerResponse{Inquiries: out}, nil
}

func (q *queryServer) AnswersByInquiry(ctx context.Context, req *types.QueryAnswersByInquiryRequest) (*types.QueryAnswersByInquiryResponse, error) {
	if req == nil || req.InquiryId == "" {
		return nil, fmt.Errorf("inquiry_id required")
	}
	out := []*types.Answer{}
	_ = q.keeper.IterateAnswersByInquiry(ctx, req.InquiryId, func(a *types.Answer) bool {
		out = append(out, a)
		return false
	})
	return &types.QueryAnswersByInquiryResponse{Answers: out}, nil
}

func (q *queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p := q.keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: &p}, nil
}
