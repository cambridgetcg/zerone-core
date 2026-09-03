package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	scheduletypes "github.com/zerone-chain/zerone/x/schedule/types"
)

type msgServer struct {
	scheduletypes.UnimplementedMsgServer
	Keeper
}

func NewMsgServerImpl(keeper Keeper) scheduletypes.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ scheduletypes.MsgServer = msgServer{}

func (m msgServer) CreateSchedule(goCtx context.Context, msg *scheduletypes.MsgCreateSchedule) (*scheduletypes.MsgCreateScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)
	if !params.AcceptNewSchedules {
		return nil, scheduletypes.ErrAdmissionClosed
	}
	currentHeight, err := positiveBlockHeight(ctx)
	if err != nil {
		return nil, err
	}
	if err := scheduletypes.ValidateTerms(currentHeight, msg.FirstExecutionHeight, msg.IntervalBlocks, msg.ExecutionCount, params); err != nil {
		return nil, scheduletypes.ErrInvalidSchedule.Wrap(err.Error())
	}
	creator, recipient, amount, err := m.validatePartiesAndAmount(msg.Creator, msg.Recipient, msg.AmountPerExecutionUzrn, params)
	if err != nil {
		return nil, err
	}
	if count := m.CountActiveByCreator(ctx, creator, params.MaxActiveSchedulesPerCreator); count >= params.MaxActiveSchedulesPerCreator {
		return nil, scheduletypes.ErrScheduleLimit.Wrapf("creator already has %d active schedules", count)
	}
	if m.PeekNextScheduleID(ctx) == ^uint64(0) {
		return nil, scheduletypes.ErrScheduleLimit.Wrap("schedule id space exhausted")
	}
	fee, _ := scheduletypes.ParsePositiveAmount(params.ExecutionFeeUzrn)
	principal, feeTotal, total := scheduleLiability(amount, fee, msg.ExecutionCount)
	if _, err := m.nextTotalEscrow(ctx, total); err != nil {
		return nil, err
	}
	if m.bankKeeper.SpendableCoins(ctx, creator).AmountOf(scheduletypes.Denom).BigInt().Cmp(total) < 0 {
		return nil, scheduletypes.ErrInsufficientEscrow.Wrapf("need %s %s", total, scheduletypes.Denom)
	}
	if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, creator, scheduletypes.ModuleName, uzrnCoins(total)); err != nil {
		return nil, scheduletypes.ErrInsufficientEscrow.Wrap(err.Error())
	}

	scheduleID, err := m.AllocateScheduleID(ctx)
	if err != nil {
		return nil, err
	}
	schedule := &scheduletypes.Schedule{
		Id:                     scheduleID,
		Creator:                creator.String(),
		Revision:               1,
		Status:                 scheduletypes.ScheduleStatus_SCHEDULE_STATUS_ACTIVE,
		Recipient:              recipient.String(),
		AmountPerExecutionUzrn: amount.String(),
		ExecutionFeeUzrn:       fee.String(),
		NextExecutionHeight:    msg.FirstExecutionHeight,
		IntervalBlocks:         msg.IntervalBlocks,
		ExecutionCount:         0,
		RemainingExecutions:    msg.ExecutionCount,
		PrincipalRemainingUzrn: principal.String(),
		FeeRemainingUzrn:       feeTotal.String(),
		CreatedHeight:          currentHeight,
		UpdatedHeight:          currentHeight,
	}
	m.SetSchedule(ctx, schedule)
	m.AddScheduleIndexes(ctx, schedule)
	if err := m.AddTotalEscrow(ctx, total); err != nil {
		return nil, err
	}
	if err := m.AssertEscrowInvariant(ctx); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.message_schedule.created",
		sdk.NewAttribute("schedule_id", schedule.Id),
		sdk.NewAttribute("creator", schedule.Creator),
		sdk.NewAttribute("recipient", schedule.Recipient),
		sdk.NewAttribute("next_execution_height", fmt.Sprint(schedule.NextExecutionHeight)),
		sdk.NewAttribute("remaining_executions", fmt.Sprint(schedule.RemainingExecutions)),
		sdk.NewAttribute("escrowed_uzrn", total.String()),
	))
	return &scheduletypes.MsgCreateScheduleResponse{ScheduleId: schedule.Id, EscrowedUzrn: total.String()}, nil
}

func (m msgServer) UpdateSchedule(goCtx context.Context, msg *scheduletypes.MsgUpdateSchedule) (*scheduletypes.MsgUpdateScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	schedule, err := m.authorizedActiveSchedule(ctx, msg.Creator, msg.ScheduleId, msg.ExpectedRevision, msg.ExpectedExecutionCount)
	if err != nil {
		return nil, err
	}
	if schedule.Revision >= ^uint64(0)-1 {
		return nil, scheduletypes.ErrInvalidSchedule.Wrap("schedule revision is exhausted")
	}
	params := m.GetParams(ctx)
	currentHeight, err := positiveBlockHeight(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(schedule.ExecutionCount)+uint64(msg.RemainingExecutions) > uint64(params.MaxExecutionsPerSchedule) {
		return nil, scheduletypes.ErrInvalidSchedule.Wrapf("lifetime executions exceed %d", params.MaxExecutionsPerSchedule)
	}
	if err := scheduletypes.ValidateTerms(currentHeight, msg.NextExecutionHeight, msg.IntervalBlocks, msg.RemainingExecutions, params); err != nil {
		return nil, scheduletypes.ErrInvalidSchedule.Wrap(err.Error())
	}
	creator, recipient, amount, err := m.validatePartiesAndAmount(msg.Creator, msg.Recipient, msg.AmountPerExecutionUzrn, params)
	if err != nil {
		return nil, err
	}
	fee, _ := scheduletypes.ParsePositiveAmount(params.ExecutionFeeUzrn)
	principal, feeTotal, newLiability := scheduleLiability(amount, fee, msg.RemainingExecutions)
	oldPrincipal, _ := scheduletypes.ParseNonNegativeAmount(schedule.PrincipalRemainingUzrn)
	oldFees, _ := scheduletypes.ParseNonNegativeAmount(schedule.FeeRemainingUzrn)
	oldLiability := new(big.Int).Add(oldPrincipal, oldFees)
	delta := new(big.Int).Sub(newLiability, oldLiability)
	if _, err := m.nextTotalEscrow(ctx, delta); err != nil {
		return nil, err
	}
	refunded := delta.Sign() < 0
	if delta.Sign() > 0 {
		if !params.AcceptNewSchedules {
			return nil, scheduletypes.ErrAdmissionClosed.Wrap("amendment would increase escrow")
		}
		if m.bankKeeper.SpendableCoins(ctx, creator).AmountOf(scheduletypes.Denom).BigInt().Cmp(delta) < 0 {
			return nil, scheduletypes.ErrInsufficientEscrow.Wrapf("need %s additional %s", delta, scheduletypes.Denom)
		}
		if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, creator, scheduletypes.ModuleName, uzrnCoins(delta)); err != nil {
			return nil, scheduletypes.ErrInsufficientEscrow.Wrap(err.Error())
		}
	} else if delta.Sign() < 0 {
		refund := new(big.Int).Neg(delta)
		if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, scheduletypes.ModuleName, creator, uzrnCoins(refund)); err != nil {
			return nil, scheduletypes.ErrEscrowInvariant.Wrapf("refund amendment delta: %v", err)
		}
	}

	m.mustDelete(ctx, scheduletypes.DueKey(schedule.NextExecutionHeight, schedule.Id))
	schedule.Revision++
	schedule.Recipient = recipient.String()
	schedule.AmountPerExecutionUzrn = amount.String()
	schedule.ExecutionFeeUzrn = fee.String()
	schedule.NextExecutionHeight = msg.NextExecutionHeight
	schedule.IntervalBlocks = msg.IntervalBlocks
	schedule.RemainingExecutions = msg.RemainingExecutions
	schedule.PrincipalRemainingUzrn = principal.String()
	schedule.FeeRemainingUzrn = feeTotal.String()
	schedule.UpdatedHeight = currentHeight
	m.SetSchedule(ctx, schedule)
	m.mustSet(ctx, scheduletypes.DueKey(schedule.NextExecutionHeight, schedule.Id), []byte{1})
	if err := m.AddTotalEscrow(ctx, delta); err != nil {
		return nil, err
	}
	if err := m.AssertEscrowInvariant(ctx); err != nil {
		return nil, err
	}

	absDelta := new(big.Int).Abs(new(big.Int).Set(delta))
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.message_schedule.updated",
		sdk.NewAttribute("schedule_id", schedule.Id),
		sdk.NewAttribute("creator", schedule.Creator),
		sdk.NewAttribute("revision", fmt.Sprint(schedule.Revision)),
		sdk.NewAttribute("expected_execution_count", fmt.Sprint(msg.ExpectedExecutionCount)),
		sdk.NewAttribute("escrow_delta_uzrn", absDelta.String()),
		sdk.NewAttribute("refunded", fmt.Sprint(refunded)),
	))
	return &scheduletypes.MsgUpdateScheduleResponse{Revision: schedule.Revision, EscrowDeltaUzrn: absDelta.String(), Refunded: refunded}, nil
}

func (m msgServer) CancelSchedule(goCtx context.Context, msg *scheduletypes.MsgCancelSchedule) (*scheduletypes.MsgCancelScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	schedule, err := m.authorizedActiveSchedule(ctx, msg.Creator, msg.ScheduleId, msg.ExpectedRevision, msg.ExpectedExecutionCount)
	if err != nil {
		return nil, err
	}
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, scheduletypes.ErrInvalidSchedule.Wrap(err.Error())
	}
	principal, _ := scheduletypes.ParseNonNegativeAmount(schedule.PrincipalRemainingUzrn)
	fees, _ := scheduletypes.ParseNonNegativeAmount(schedule.FeeRemainingUzrn)
	refund := new(big.Int).Add(principal, fees)
	if _, err := m.nextTotalEscrow(ctx, new(big.Int).Neg(refund)); err != nil {
		return nil, err
	}
	if refund.Sign() > 0 {
		if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, scheduletypes.ModuleName, creator, uzrnCoins(refund)); err != nil {
			return nil, scheduletypes.ErrEscrowInvariant.Wrapf("cancel refund: %v", err)
		}
	}
	m.RemoveActiveIndexes(ctx, schedule)
	schedule.Status = scheduletypes.ScheduleStatus_SCHEDULE_STATUS_CANCELLED
	schedule.RemainingExecutions = 0
	schedule.PrincipalRemainingUzrn = "0"
	schedule.FeeRemainingUzrn = "0"
	schedule.NextExecutionHeight = 0
	schedule.UpdatedHeight = uint64(ctx.BlockHeight())
	schedule.TerminalReason = "cancelled_by_creator"
	m.SetSchedule(ctx, schedule)
	if err := m.AddTotalEscrow(ctx, new(big.Int).Neg(refund)); err != nil {
		return nil, err
	}
	if err := m.AssertEscrowInvariant(ctx); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.message_schedule.cancelled",
		sdk.NewAttribute("schedule_id", schedule.Id),
		sdk.NewAttribute("creator", schedule.Creator),
		sdk.NewAttribute("revision", fmt.Sprint(schedule.Revision)),
		sdk.NewAttribute("execution_count", fmt.Sprint(schedule.ExecutionCount)),
		sdk.NewAttribute("refunded_uzrn", refund.String()),
	))
	return &scheduletypes.MsgCancelScheduleResponse{RefundedUzrn: refund.String()}, nil
}

func (m msgServer) UpdateParams(goCtx context.Context, msg *scheduletypes.MsgUpdateParams) (*scheduletypes.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Authority != m.Authority() {
		return nil, scheduletypes.ErrUnauthorized.Wrapf("expected %s", m.Authority())
	}
	if err := m.SetParams(ctx, msg.Params); err != nil {
		return nil, scheduletypes.ErrInvalidSchedule.Wrap(err.Error())
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.message_schedule.params_updated",
		sdk.NewAttribute("authority", msg.Authority),
		sdk.NewAttribute("accept_new_schedules", fmt.Sprint(msg.Params.AcceptNewSchedules)),
	))
	return &scheduletypes.MsgUpdateParamsResponse{}, nil
}

func (m msgServer) authorizedActiveSchedule(ctx sdk.Context, creator, id string, expectedRevision uint64, expectedExecutionCount uint32) (*scheduletypes.Schedule, error) {
	schedule, found := m.GetSchedule(ctx, id)
	if !found {
		return nil, scheduletypes.ErrScheduleNotFound.Wrap(id)
	}
	creatorAddress, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return nil, scheduletypes.ErrInvalidSchedule.Wrapf("invalid creator: %v", err)
	}
	if schedule.Creator != creatorAddress.String() {
		return nil, scheduletypes.ErrUnauthorized.Wrapf("schedule creator is %s", schedule.Creator)
	}
	if schedule.Status != scheduletypes.ScheduleStatus_SCHEDULE_STATUS_ACTIVE {
		return nil, scheduletypes.ErrScheduleNotActive.Wrap(schedule.Status.String())
	}
	if schedule.Revision != expectedRevision {
		return nil, scheduletypes.ErrRevisionConflict.Wrapf("expected %d, committed %d", expectedRevision, schedule.Revision)
	}
	if schedule.ExecutionCount != expectedExecutionCount {
		return nil, scheduletypes.ErrExecutionConflict.Wrapf("expected %d, committed %d", expectedExecutionCount, schedule.ExecutionCount)
	}
	return schedule, nil
}

func (m msgServer) validatePartiesAndAmount(creatorText, recipientText, amountText string, params *scheduletypes.Params) (sdk.AccAddress, sdk.AccAddress, *big.Int, error) {
	creator, err := sdk.AccAddressFromBech32(creatorText)
	if err != nil {
		return nil, nil, nil, scheduletypes.ErrInvalidSchedule.Wrapf("invalid creator: %v", err)
	}
	if m.bankKeeper.BlockedAddr(creator) {
		return nil, nil, nil, scheduletypes.ErrInvalidSchedule.Wrap("creator cannot receive escrow refunds")
	}
	recipient, err := sdk.AccAddressFromBech32(recipientText)
	if err != nil {
		return nil, nil, nil, scheduletypes.ErrInvalidSchedule.Wrapf("invalid recipient: %v", err)
	}
	if m.bankKeeper.BlockedAddr(recipient) {
		return nil, nil, nil, scheduletypes.ErrBlockedRecipient.Wrap(recipientText)
	}
	amount, err := scheduletypes.ParsePositiveAmount(amountText)
	if err != nil {
		return nil, nil, nil, scheduletypes.ErrInvalidSchedule.Wrapf("amount: %v", err)
	}
	maxAmount, _ := scheduletypes.ParsePositiveAmount(params.MaxTransferPerExecutionUzrn)
	if amount.Cmp(maxAmount) > 0 {
		return nil, nil, nil, scheduletypes.ErrInvalidSchedule.Wrapf("amount %s exceeds maximum %s", amount, maxAmount)
	}
	return creator, recipient, amount, nil
}

func scheduleLiability(amount, fee *big.Int, count uint32) (*big.Int, *big.Int, *big.Int) {
	multiplier := new(big.Int).SetUint64(uint64(count))
	principal := new(big.Int).Mul(new(big.Int).Set(amount), multiplier)
	feeTotal := new(big.Int).Mul(new(big.Int).Set(fee), multiplier)
	return principal, feeTotal, new(big.Int).Add(new(big.Int).Set(principal), feeTotal)
}

func uzrnCoins(amount *big.Int) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin(scheduletypes.Denom, sdkmath.NewIntFromBigInt(amount)))
}

func positiveBlockHeight(ctx sdk.Context) (uint64, error) {
	if ctx.BlockHeight() <= 0 {
		return 0, scheduletypes.ErrInvalidSchedule.Wrap("schedule messages require a positive committed block height")
	}
	return uint64(ctx.BlockHeight()), nil
}
