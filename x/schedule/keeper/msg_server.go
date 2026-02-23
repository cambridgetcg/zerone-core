package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateSchedule creates a new scheduled process.
func (k msgServer) CreateSchedule(goCtx context.Context, msg *types.MsgCreateSchedule) (*types.MsgCreateScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	params := k.GetParams(ctx)

	// Validate condition
	if msg.Condition == nil {
		return nil, fmt.Errorf("%w: condition is required", types.ErrInvalidCondition)
	}
	if err := validateCondition(msg.Condition, params, 0); err != nil {
		return nil, err
	}

	// Check active schedule count
	activeCount := k.CountActiveByCreator(ctx, msg.Creator)
	if activeCount >= params.MaxActivePerAccount {
		return nil, fmt.Errorf("%w: account has %d active schedules (max %d)",
			types.ErrMaxSchedulesExceeded, activeCount, params.MaxActivePerAccount)
	}

	// Validate fee_per_execution against minimum
	feePerExec := new(big.Int)
	if _, ok := feePerExec.SetString(msg.FeePerExecution, 10); !ok || feePerExec.Sign() <= 0 {
		return nil, fmt.Errorf("%w: fee_per_execution must be a positive integer", types.ErrInvalidAmount)
	}
	minFee := new(big.Int)
	if _, ok := minFee.SetString(params.MinFeePerExecution, 10); !ok {
		minFee.SetInt64(0)
	}
	if feePerExec.Cmp(minFee) < 0 {
		return nil, fmt.Errorf("%w: fee_per_execution %s is below minimum %s",
			types.ErrInsufficientFee, msg.FeePerExecution, params.MinFeePerExecution)
	}

	// Validate and deduct prepaid fee
	prepaidFee := new(big.Int)
	if _, ok := prepaidFee.SetString(msg.PrepaidFee, 10); !ok || prepaidFee.Sign() <= 0 {
		return nil, fmt.Errorf("%w: prepaid_fee must be a positive integer", types.ErrInvalidAmount)
	}

	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(prepaidFee)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to deduct prepaid fee: %w", err)
	}

	// Generate process ID
	seq := k.NextSequence(ctx)
	processId := fmt.Sprintf("sched-%d", seq)

	// Calculate next execution time
	currentHeight := uint64(ctx.BlockHeight())
	nextExecAt := calculateNextExecuteAt(msg.Condition, currentHeight)

	process := &types.ScheduleProcess{
		Id:               processId,
		Creator:          msg.Creator,
		Condition:        msg.Condition,
		Status:           "active",
		ExecutionCount:   0,
		MaxExecutions:    msg.MaxExecutions,
		RemainingFee:     prepaidFee.String(),
		FeePerExecution:  msg.FeePerExecution,
		TargetAddress:    msg.TargetAddress,
		CallData:         msg.CallData,
		TransferValue:    msg.TransferValue,
		LinkedEntityType: msg.LinkedEntityType,
		LinkedEntityId:   msg.LinkedEntityId,
		CreatedAtBlock:   currentHeight,
		ExpiresAtBlock:   msg.ExpiresAtBlock,
		NextExecuteAt:    nextExecAt,
	}

	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.schedule.schedule_created",
			sdk.NewAttribute("process_id", processId),
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("prepaid_fee", msg.PrepaidFee),
			sdk.NewAttribute("next_execute_at", fmt.Sprintf("%d", nextExecAt)),
		),
	)

	return &types.MsgCreateScheduleResponse{ProcessId: processId}, nil
}

// PauseSchedule pauses an active scheduled process.
func (k msgServer) PauseSchedule(goCtx context.Context, msg *types.MsgPauseSchedule) (*types.MsgPauseScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	process, found := k.GetProcess(ctx, msg.ProcessId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProcessNotFound, msg.ProcessId)
	}

	if process.Creator != msg.Creator {
		return nil, fmt.Errorf("%w: only creator can pause", types.ErrUnauthorized)
	}

	if process.Status != "active" {
		return nil, fmt.Errorf("%w: current status is %s", types.ErrProcessNotActive, process.Status)
	}

	// Remove old indexes before updating
	k.RemoveIndexes(ctx, process)

	process.Status = "paused"
	process.NextExecuteAt = 0

	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.schedule.schedule_paused",
			sdk.NewAttribute("process_id", msg.ProcessId),
			sdk.NewAttribute("creator", msg.Creator),
		),
	)

	return &types.MsgPauseScheduleResponse{}, nil
}

// ResumeSchedule resumes a paused scheduled process.
func (k msgServer) ResumeSchedule(goCtx context.Context, msg *types.MsgResumeSchedule) (*types.MsgResumeScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	process, found := k.GetProcess(ctx, msg.ProcessId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProcessNotFound, msg.ProcessId)
	}

	if process.Creator != msg.Creator {
		return nil, fmt.Errorf("%w: only creator can resume", types.ErrUnauthorized)
	}

	if process.Status != "paused" {
		return nil, fmt.Errorf("%w: current status is %s", types.ErrProcessNotPaused, process.Status)
	}

	// Remove old indexes before updating
	k.RemoveIndexes(ctx, process)

	currentHeight := uint64(ctx.BlockHeight())
	process.Status = "active"
	process.NextExecuteAt = calculateNextExecuteAt(process.Condition, currentHeight)

	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.schedule.schedule_resumed",
			sdk.NewAttribute("process_id", msg.ProcessId),
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("next_execute_at", fmt.Sprintf("%d", process.NextExecuteAt)),
		),
	)

	return &types.MsgResumeScheduleResponse{}, nil
}

// CancelSchedule cancels a scheduled process and refunds remaining fees.
func (k msgServer) CancelSchedule(goCtx context.Context, msg *types.MsgCancelSchedule) (*types.MsgCancelScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	process, found := k.GetProcess(ctx, msg.ProcessId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProcessNotFound, msg.ProcessId)
	}

	if process.Creator != msg.Creator {
		return nil, fmt.Errorf("%w: only creator can cancel", types.ErrUnauthorized)
	}

	if process.Status == "completed" || process.Status == "cancelled" {
		return nil, fmt.Errorf("%w: process is already %s", types.ErrProcessCompleted, process.Status)
	}

	// Refund remaining fee
	refundedAmount := process.RemainingFee
	remaining := new(big.Int)
	if _, ok := remaining.SetString(process.RemainingFee, 10); ok && remaining.Sign() > 0 {
		creatorAddr, err := sdk.AccAddressFromBech32(process.Creator)
		if err != nil {
			return nil, fmt.Errorf("invalid creator address: %w", err)
		}
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(remaining)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, coins); err != nil {
			return nil, fmt.Errorf("failed to refund remaining fee: %w", err)
		}
	}

	// Remove old indexes before updating
	k.RemoveIndexes(ctx, process)

	process.Status = "cancelled"
	process.RemainingFee = "0"
	process.NextExecuteAt = 0

	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.schedule.schedule_cancelled",
			sdk.NewAttribute("process_id", msg.ProcessId),
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("refunded_amount", refundedAmount),
		),
	)

	return &types.MsgCancelScheduleResponse{RefundedAmount: refundedAmount}, nil
}

// FundSchedule adds additional funds to a scheduled process.
func (k msgServer) FundSchedule(goCtx context.Context, msg *types.MsgFundSchedule) (*types.MsgFundScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	process, found := k.GetProcess(ctx, msg.ProcessId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProcessNotFound, msg.ProcessId)
	}

	if process.Creator != msg.Creator {
		return nil, fmt.Errorf("%w: only creator can fund", types.ErrUnauthorized)
	}

	if process.Status == "completed" || process.Status == "cancelled" {
		return nil, fmt.Errorf("%w: process is %s", types.ErrProcessCompleted, process.Status)
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: amount must be a positive integer", types.ErrInvalidAmount)
	}

	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(amount)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to transfer funding amount: %w", err)
	}

	// Remove old indexes before updating
	k.RemoveIndexes(ctx, process)

	// Add amount to remaining fee
	remaining := new(big.Int)
	remaining.SetString(process.RemainingFee, 10)
	remaining.Add(remaining, amount)
	process.RemainingFee = remaining.String()

	// Reactivate if exhausted
	if process.Status == "exhausted" {
		currentHeight := uint64(ctx.BlockHeight())
		process.Status = "active"
		process.NextExecuteAt = calculateNextExecuteAt(process.Condition, currentHeight)
	}

	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.schedule.schedule_funded",
			sdk.NewAttribute("process_id", msg.ProcessId),
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("new_remaining", process.RemainingFee),
		),
	)

	return &types.MsgFundScheduleResponse{}, nil
}

// UpdateParams handles governance-gated parameter update.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	k.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.schedule.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}

// ---------- Condition Validation ----------

// validateCondition recursively validates a schedule condition.
func validateCondition(cond *types.ScheduleCondition, params *types.Params, depth uint64) error {
	if cond == nil {
		return fmt.Errorf("%w: condition is nil", types.ErrInvalidCondition)
	}

	setCount := 0
	if cond.TimeCondition != nil {
		setCount++
	}
	if cond.LogicCondition != nil {
		setCount++
	}
	if cond.CompoundCondition != nil {
		setCount++
	}
	if setCount == 0 {
		return fmt.Errorf("%w: condition must have at least one type set", types.ErrInvalidCondition)
	}

	if cond.TimeCondition != nil {
		tc := cond.TimeCondition
		switch tc.Type {
		case "at_block":
			if tc.ExecuteAtBlock == 0 {
				return fmt.Errorf("%w: at_block requires execute_at_block > 0", types.ErrInvalidCondition)
			}
		case "every_n_blocks":
			if tc.IntervalBlocks == 0 {
				return fmt.Errorf("%w: every_n_blocks requires interval_blocks > 0", types.ErrInvalidInterval)
			}
			if tc.IntervalBlocks < params.MinIntervalBlocks {
				return fmt.Errorf("%w: interval_blocks %d is below minimum %d",
					types.ErrInvalidInterval, tc.IntervalBlocks, params.MinIntervalBlocks)
			}
		default:
			return fmt.Errorf("%w: unknown time condition type %q", types.ErrInvalidCondition, tc.Type)
		}
	}

	if cond.LogicCondition != nil {
		lc := cond.LogicCondition
		if lc.Type != "state_compare" {
			return fmt.Errorf("%w: unknown logic condition type %q", types.ErrInvalidCondition, lc.Type)
		}
		if lc.StateKey == "" {
			return fmt.Errorf("%w: state_key is required for state_compare", types.ErrInvalidCondition)
		}
		validComparators := map[string]bool{"eq": true, "gt": true, "lt": true, "gte": true, "lte": true}
		if !validComparators[lc.Comparator] {
			return fmt.Errorf("%w: invalid comparator %q", types.ErrInvalidCondition, lc.Comparator)
		}
	}

	if cond.CompoundCondition != nil {
		cc := cond.CompoundCondition
		if cc.Operator != "and" && cc.Operator != "or" {
			return fmt.Errorf("%w: compound operator must be 'and' or 'or'", types.ErrInvalidCondition)
		}
		if len(cc.Conditions) < 2 {
			return fmt.Errorf("%w: compound condition requires at least 2 sub-conditions", types.ErrInvalidCondition)
		}
		if depth+1 > params.MaxCompoundDepth {
			return fmt.Errorf("%w: depth %d exceeds max %d",
				types.ErrCompoundDepthExceeded, depth+1, params.MaxCompoundDepth)
		}
		for _, sub := range cc.Conditions {
			if err := validateCondition(sub, params, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// calculateNextExecuteAt determines the next block height at which a process should execute.
func calculateNextExecuteAt(cond *types.ScheduleCondition, currentHeight uint64) uint64 {
	if cond == nil {
		return 0
	}

	if cond.TimeCondition != nil {
		tc := cond.TimeCondition
		switch tc.Type {
		case "at_block":
			return tc.ExecuteAtBlock
		case "every_n_blocks":
			start := tc.StartBlock
			if start <= currentHeight {
				// Next interval from current height
				return currentHeight + tc.IntervalBlocks
			}
			return start
		}
	}

	// For compound conditions, find the earliest time condition
	if cond.CompoundCondition != nil {
		var earliest uint64
		for _, sub := range cond.CompoundCondition.Conditions {
			next := calculateNextExecuteAt(sub, currentHeight)
			if next > 0 && (earliest == 0 || next < earliest) {
				earliest = next
			}
		}
		return earliest
	}

	// Logic-only conditions: schedule for next block
	if cond.LogicCondition != nil {
		return currentHeight + 1
	}

	return 0
}
