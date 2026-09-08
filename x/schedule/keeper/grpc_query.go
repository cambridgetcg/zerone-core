package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (q queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: q.GetParams(ctx)}, nil
}

func (q queryServer) Schedule(ctx context.Context, req *types.QueryScheduleRequest) (*types.QueryScheduleResponse, error) {
	if _, err := types.ParseScheduleID(req.Id); err != nil {
		return nil, types.ErrInvalidSchedule.Wrap(err.Error())
	}
	schedule, found := q.GetSchedule(ctx, req.Id)
	if !found {
		return nil, types.ErrScheduleNotFound.Wrap(req.Id)
	}
	return &types.QueryScheduleResponse{Schedule: schedule}, nil
}

func (q queryServer) SchedulesByCreator(ctx context.Context, req *types.QuerySchedulesByCreatorRequest) (*types.QuerySchedulesByCreatorResponse, error) {
	creator, err := sdk.AccAddressFromBech32(req.Creator)
	if err != nil {
		return nil, types.ErrInvalidSchedule.Wrapf("invalid creator: %v", err)
	}
	if req.StartAfter != "" {
		if _, err := types.ParseScheduleID(req.StartAfter); err != nil {
			return nil, types.ErrInvalidSchedule.Wrapf("start_after: %v", err)
		}
	}
	limit, err := q.queryLimit(ctx, req.Limit)
	if err != nil {
		return nil, err
	}
	prefix := types.CreatorPrefix(creator)
	start := prefix
	if req.StartAfter != "" {
		start = types.CreatorKey(creator, req.StartAfter)
	}
	schedules := make([]*types.Schedule, 0, limit+1)
	q.iterate(ctx, start, types.PrefixEndBytes(prefix), func(key, _ []byte) bool {
		id := string(key[len(prefix):])
		if id == req.StartAfter {
			return false
		}
		schedule, found := q.GetSchedule(ctx, id)
		if !found || schedule.Creator != creator.String() {
			panic(fmt.Sprintf("corrupt creator index for schedule %s", id))
		}
		schedules = append(schedules, schedule)
		return uint32(len(schedules)) > limit
	})
	response := &types.QuerySchedulesByCreatorResponse{Schedules: schedules}
	if uint32(len(schedules)) > limit {
		response.NextKey = schedules[limit-1].Id
		response.Schedules = schedules[:limit]
	}
	return response, nil
}

func (q queryServer) Receipt(ctx context.Context, req *types.QueryReceiptRequest) (*types.QueryReceiptResponse, error) {
	if err := types.ValidateDigest(req.OccurrenceId); err != nil {
		return nil, types.ErrInvalidSchedule.Wrapf("occurrence_id: %v", err)
	}
	receipt, found := q.GetReceiptByOccurrence(ctx, req.OccurrenceId)
	if !found {
		return nil, types.ErrReceiptNotFound.Wrap(req.OccurrenceId)
	}
	return &types.QueryReceiptResponse{Receipt: receipt}, nil
}

func (q queryServer) ReceiptsBySchedule(ctx context.Context, req *types.QueryReceiptsByScheduleRequest) (*types.QueryReceiptsByScheduleResponse, error) {
	if _, err := types.ParseScheduleID(req.ScheduleId); err != nil {
		return nil, types.ErrInvalidSchedule.Wrap(err.Error())
	}
	if _, found := q.GetSchedule(ctx, req.ScheduleId); !found {
		return nil, types.ErrScheduleNotFound.Wrap(req.ScheduleId)
	}
	limit, err := q.queryLimit(ctx, req.Limit)
	if err != nil {
		return nil, err
	}
	startSequence := req.StartSequence
	if startSequence == 0 {
		startSequence = 1
	}
	prefix := types.ReceiptSchedulePrefix(req.ScheduleId)
	receipts := make([]*types.ExecutionReceipt, 0, limit+1)
	q.iterate(ctx, types.ReceiptKey(req.ScheduleId, startSequence), types.PrefixEndBytes(prefix), func(key, value []byte) bool {
		if len(key) != len(prefix)+4 {
			panic(fmt.Sprintf("corrupt receipt key %x", key))
		}
		sequence := binary.BigEndian.Uint32(key[len(prefix):])
		var receipt types.ExecutionReceipt
		q.mustUnmarshal(value, &receipt)
		if receipt.ScheduleId != req.ScheduleId || receipt.Sequence != sequence {
			panic(fmt.Sprintf("corrupt receipt index for schedule %s sequence %d", req.ScheduleId, sequence))
		}
		receipts = append(receipts, &receipt)
		return uint32(len(receipts)) > limit
	})
	response := &types.QueryReceiptsByScheduleResponse{Receipts: receipts}
	if uint32(len(receipts)) > limit {
		response.NextSequence = receipts[limit].Sequence
		response.Receipts = receipts[:limit]
	}
	return response, nil
}

func (q queryServer) queryLimit(ctx context.Context, requested uint32) (uint32, error) {
	maxLimit := q.GetParams(ctx).MaxQueryLimit
	if requested == 0 {
		if maxLimit < 50 {
			return maxLimit, nil
		}
		return 50, nil
	}
	if requested > maxLimit {
		return 0, types.ErrInvalidSchedule.Wrapf("query limit %d exceeds maximum %d", requested, maxLimit)
	}
	return requested, nil
}
