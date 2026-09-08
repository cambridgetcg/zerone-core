package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	"github.com/zerone-chain/zerone/x/schedule/types"
)

type dueRecord struct {
	height uint64
	id     string
}

// ProcessDueSchedules consumes a bounded, lexicographically ordered prefix of
// committed due records. Delayed recurring schedules use fixed-delay
// semantics: the next height is executed_height + interval, so a backlog never
// creates an unbounded catch-up burst.
func (k Keeper) ProcessDueSchedules(ctx sdk.Context) (uint32, error) {
	if ctx.BlockHeight() <= 0 {
		return 0, nil
	}
	currentHeight := uint64(ctx.BlockHeight())
	if k.emergency.IsHalted(ctx) || k.inPostResumeGrace(ctx, currentHeight) {
		return 0, nil
	}
	limit := k.GetParams(ctx).MaxDueRecordsPerBlock
	records := make([]dueRecord, 0, limit)
	k.iterate(ctx, types.DueKeyPrefix, types.DueThroughHeightEnd(currentHeight), func(key, _ []byte) bool {
		height, id, ok := types.ParseDueKey(key)
		if !ok || id == "" {
			panic(fmt.Sprintf("corrupt schedule due key %x", key))
		}
		records = append(records, dueRecord{height: height, id: id})
		return uint32(len(records)) >= limit
	})

	var processed uint32
	for _, record := range records {
		if err := k.processOccurrence(ctx, currentHeight, record); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (k Keeper) inPostResumeGrace(ctx sdk.Context, currentHeight uint64) bool {
	releaseHeight := k.emergency.GetQuarantineReleaseBlock(ctx)
	if releaseHeight == 0 || currentHeight <= releaseHeight {
		return false
	}
	if releaseHeight > ^uint64(0)-emergencytypes.PostResumeCancellationGraceBlocks {
		return true
	}
	return currentHeight <= releaseHeight+emergencytypes.PostResumeCancellationGraceBlocks
}

func (k Keeper) processOccurrence(ctx sdk.Context, currentHeight uint64, record dueRecord) error {
	schedule, found := k.GetSchedule(ctx, record.id)
	if !found {
		return types.ErrEscrowInvariant.Wrapf("due record references missing schedule %s", record.id)
	}
	if schedule.Status != types.ScheduleStatus_SCHEDULE_STATUS_ACTIVE || schedule.NextExecutionHeight != record.height {
		return types.ErrEscrowInvariant.Wrapf("stale due record for %s at %d", record.id, record.height)
	}
	if record.height > currentHeight {
		return types.ErrEscrowInvariant.Wrapf("future due record %s selected at %d", record.id, record.height)
	}
	if err := types.ValidateStoredSchedule(schedule, k.GetParams(ctx)); err != nil {
		return types.ErrEscrowInvariant.Wrapf("invalid stored schedule %s: %v", record.id, err)
	}
	sequence := schedule.ExecutionCount + 1
	occurrenceID := types.OccurrenceID(ctx.ChainID(), schedule.Id, schedule.Revision, sequence, record.height)
	if _, exists := k.GetReceipt(ctx, schedule.Id, sequence); exists {
		return types.ErrEscrowInvariant.Wrapf("receipt already exists for %s sequence %d", schedule.Id, sequence)
	}
	if _, exists := k.GetReceiptByOccurrence(ctx, occurrenceID); exists {
		return types.ErrEscrowInvariant.Wrapf("occurrence already exists for %s", occurrenceID)
	}

	receipt := &types.ExecutionReceipt{
		OccurrenceId:   occurrenceID,
		ScheduleId:     schedule.Id,
		Revision:       schedule.Revision,
		Sequence:       sequence,
		DueHeight:      record.height,
		ExecutedHeight: currentHeight,
		Recipient:      schedule.Recipient,
		AmountUzrn:     schedule.AmountPerExecutionUzrn,
		FeeUzrn:        schedule.ExecutionFeeUzrn,
		ActionSha256:   types.ActionDigest(schedule.Recipient, schedule.AmountPerExecutionUzrn, schedule.ExecutionFeeUzrn),
	}
	if schedule.RemainingExecutions > 1 &&
		currentHeight > types.MaxSDKBlockHeight-schedule.IntervalBlocks {
		return k.failAndRefundOccurrence(
			ctx,
			schedule,
			receipt,
			types.FailureCodeNextHeight,
			fmt.Errorf("next execution height exceeds signed SDK block-height range"),
		)
	}

	cacheCtx, write := ctx.CacheContext()
	recipient := mustAddress(schedule.Recipient)
	amount, _ := types.ParsePositiveAmount(schedule.AmountPerExecutionUzrn)
	fee, _ := types.ParsePositiveAmount(schedule.ExecutionFeeUzrn)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, recipient, uzrnCoins(amount)); err != nil {
		return k.failAndRefundOccurrence(ctx, schedule, receipt, types.FailureCodeBankTransfer, err)
	}
	if err := k.bankKeeper.SendCoinsFromModuleToModule(cacheCtx, types.ModuleName, authtypes.FeeCollectorName, uzrnCoins(fee)); err != nil {
		return k.failAndRefundOccurrence(ctx, schedule, receipt, types.FailureCodeBankTransfer, err)
	}

	currentDueKey := types.DueKey(schedule.NextExecutionHeight, schedule.Id)
	k.mustDelete(cacheCtx, currentDueKey)
	schedule.ExecutionCount++
	schedule.RemainingExecutions--
	schedule.LastExecutionHeight = currentHeight
	schedule.UpdatedHeight = currentHeight
	principal, _ := types.ParseNonNegativeAmount(schedule.PrincipalRemainingUzrn)
	fees, _ := types.ParseNonNegativeAmount(schedule.FeeRemainingUzrn)
	principal.Sub(principal, amount)
	fees.Sub(fees, fee)
	if principal.Sign() < 0 || fees.Sign() < 0 {
		return types.ErrEscrowInvariant.Wrapf("negative remaining escrow for %s", schedule.Id)
	}
	schedule.PrincipalRemainingUzrn = principal.String()
	schedule.FeeRemainingUzrn = fees.String()
	if schedule.RemainingExecutions == 0 {
		schedule.Status = types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED
		schedule.NextExecutionHeight = 0
		schedule.TerminalReason = "all_occurrences_succeeded"
		k.mustDelete(cacheCtx, types.ActiveCreatorKey(mustAddress(schedule.Creator), schedule.Id))
	} else {
		schedule.NextExecutionHeight = currentHeight + schedule.IntervalBlocks
		k.mustSet(cacheCtx, types.DueKey(schedule.NextExecutionHeight, schedule.Id), []byte{1})
	}
	k.SetSchedule(cacheCtx, schedule)
	receipt.Outcome = types.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED
	k.SetReceipt(cacheCtx, receipt)
	executedLiability := new(big.Int).Add(new(big.Int).Set(amount), fee)
	if err := k.AddTotalEscrow(cacheCtx, new(big.Int).Neg(executedLiability)); err != nil {
		return err
	}
	if err := k.AssertEscrowInvariant(cacheCtx); err != nil {
		return err
	}
	write()

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.message_schedule.executed",
		sdk.NewAttribute("schedule_id", schedule.Id),
		sdk.NewAttribute("occurrence_id", receipt.OccurrenceId),
		sdk.NewAttribute("revision", fmt.Sprint(receipt.Revision)),
		sdk.NewAttribute("sequence", fmt.Sprint(receipt.Sequence)),
		sdk.NewAttribute("due_height", fmt.Sprint(receipt.DueHeight)),
		sdk.NewAttribute("executed_height", fmt.Sprint(receipt.ExecutedHeight)),
		sdk.NewAttribute("outcome", "succeeded"),
	))
	return nil
}

func (k Keeper) failAndRefundOccurrence(
	ctx sdk.Context,
	schedule *types.Schedule,
	receipt *types.ExecutionReceipt,
	failureCode string,
	actionErr error,
) error {
	cacheCtx, write := ctx.CacheContext()
	creator := mustAddress(schedule.Creator)
	principal, _ := types.ParseNonNegativeAmount(schedule.PrincipalRemainingUzrn)
	fees, _ := types.ParseNonNegativeAmount(schedule.FeeRemainingUzrn)
	refund := new(big.Int).Add(principal, fees)
	if refund.Sign() > 0 {
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, creator, uzrnCoins(refund)); err != nil {
			return types.ErrEscrowInvariant.Wrapf("action failed (%v) and refund failed (%v)", actionErr, err)
		}
	}
	k.RemoveActiveIndexes(cacheCtx, schedule)
	schedule.ExecutionCount++
	schedule.RemainingExecutions = 0
	schedule.PrincipalRemainingUzrn = "0"
	schedule.FeeRemainingUzrn = "0"
	schedule.NextExecutionHeight = 0
	schedule.LastExecutionHeight = uint64(ctx.BlockHeight())
	schedule.UpdatedHeight = uint64(ctx.BlockHeight())
	schedule.Status = types.ScheduleStatus_SCHEDULE_STATUS_FAILED
	schedule.TerminalReason = failureCode
	k.SetSchedule(cacheCtx, schedule)
	receipt.Outcome = types.ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED
	receipt.FailureCode = failureCode
	k.SetReceipt(cacheCtx, receipt)
	if err := k.AddTotalEscrow(cacheCtx, new(big.Int).Neg(refund)); err != nil {
		return err
	}
	if err := k.AssertEscrowInvariant(cacheCtx); err != nil {
		return err
	}
	write()

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.message_schedule.executed",
		sdk.NewAttribute("schedule_id", schedule.Id),
		sdk.NewAttribute("occurrence_id", receipt.OccurrenceId),
		sdk.NewAttribute("revision", fmt.Sprint(receipt.Revision)),
		sdk.NewAttribute("sequence", fmt.Sprint(receipt.Sequence)),
		sdk.NewAttribute("due_height", fmt.Sprint(receipt.DueHeight)),
		sdk.NewAttribute("executed_height", fmt.Sprint(receipt.ExecutedHeight)),
		sdk.NewAttribute("outcome", "failed_and_refunded"),
		sdk.NewAttribute("failure_code", receipt.FailureCode),
		sdk.NewAttribute("refunded_uzrn", refund.String()),
	))
	return nil
}
