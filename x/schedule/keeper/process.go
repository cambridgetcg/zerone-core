package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

// ProcessDueSchedules executes all schedules due at the current block height.
// Called by EndBlocker. Returns the number of executions performed.
func (k Keeper) ProcessDueSchedules(ctx sdk.Context) int {
	currentHeight := uint64(ctx.BlockHeight())
	dueIds := k.GetDueProcesses(ctx, currentHeight)

	executionCount := 0

	for _, processId := range dueIds {
		process, found := k.GetProcess(ctx, processId)
		if !found {
			continue
		}

		// Only process active schedules
		if process.Status != "active" {
			continue
		}

		// Check expiration
		if process.ExpiresAtBlock > 0 && currentHeight > process.ExpiresAtBlock {
			k.RemoveIndexes(ctx, process)
			process.Status = "completed"
			process.NextExecuteAt = 0
			k.SetProcess(ctx, process)
			continue
		}

		// Check remaining fee
		feePerExec := new(big.Int)
		feePerExec.SetString(process.FeePerExecution, 10)
		remainingFee := new(big.Int)
		remainingFee.SetString(process.RemainingFee, 10)

		if remainingFee.Cmp(feePerExec) < 0 {
			// Insufficient fee — mark as exhausted
			k.RemoveIndexes(ctx, process)
			process.Status = "exhausted"
			process.NextExecuteAt = 0
			k.SetProcess(ctx, process)
			continue
		}

		// Evaluate condition
		if !k.evaluateCondition(ctx, process.Condition) {
			// Condition not met. For periodic schedules, reschedule.
			k.rescheduleProcess(ctx, process, currentHeight)
			continue
		}

		// --- Execute ---

		// Remove old indexes
		k.RemoveIndexes(ctx, process)

		// Deduct fee
		remainingFee.Sub(remainingFee, feePerExec)
		process.RemainingFee = remainingFee.String()

		// Increment execution count
		process.ExecutionCount++

		// Transfer value if set
		if process.TransferValue != "" && process.TransferValue != "0" {
			transferAmt := new(big.Int)
			if _, ok := transferAmt.SetString(process.TransferValue, 10); ok && transferAmt.Sign() > 0 {
				if process.TargetAddress != "" {
					targetAddr, err := sdk.AccAddressFromBech32(process.TargetAddress)
					if err == nil {
						coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(transferAmt)))
						if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, targetAddr, coins); err != nil {
							k.Logger(ctx).Error("failed to transfer value",
								"process_id", process.Id,
								"target", process.TargetAddress,
								"error", err,
							)
						}
					}
				}
			}
		}

		// Check if max executions reached
		if process.MaxExecutions > 0 && process.ExecutionCount >= process.MaxExecutions {
			process.Status = "completed"
			process.NextExecuteAt = 0
			k.SetProcess(ctx, process)
		} else {
			// Determine if this is a one-time or periodic schedule
			if isOneTimeSchedule(process.Condition) {
				process.Status = "completed"
				process.NextExecuteAt = 0
				k.SetProcess(ctx, process)
			} else {
				// Schedule next execution
				process.NextExecuteAt = calculateNextExecuteAt(process.Condition, currentHeight)
				k.SetProcess(ctx, process)
			}
		}

		executionCount++

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zerone.schedule.schedule_executed",
				sdk.NewAttribute("process_id", process.Id),
				sdk.NewAttribute("execution_count", fmt.Sprintf("%d", process.ExecutionCount)),
				sdk.NewAttribute("remaining_fee", process.RemainingFee),
				sdk.NewAttribute("status", process.Status),
			),
		)
	}

	return executionCount
}

// rescheduleProcess updates the time index for a process that was due but whose
// condition was not met.
func (k Keeper) rescheduleProcess(ctx sdk.Context, process *types.ScheduleProcess, currentHeight uint64) {
	k.RemoveIndexes(ctx, process)
	process.NextExecuteAt = calculateNextExecuteAt(process.Condition, currentHeight)
	k.SetProcess(ctx, process)
}

// isOneTimeSchedule returns true if the condition represents a one-time execution.
func isOneTimeSchedule(cond *types.ScheduleCondition) bool {
	if cond == nil {
		return true
	}
	if cond.TimeCondition != nil {
		return cond.TimeCondition.Type == "at_block"
	}
	if cond.CompoundCondition != nil {
		// A compound condition is one-time if all sub-conditions are one-time
		for _, sub := range cond.CompoundCondition.Conditions {
			if !isOneTimeSchedule(sub) {
				return false
			}
		}
		return true
	}
	// Logic-only conditions are one-time by default
	return true
}

// evaluateCondition evaluates a schedule condition against the current state.
func (k Keeper) evaluateCondition(ctx sdk.Context, cond *types.ScheduleCondition) bool {
	if cond == nil {
		return false
	}

	// Time conditions: if we reached this point, time is due (handled by time index).
	// So time conditions always evaluate to true when checked during processing.
	timeResult := true
	hasTime := cond.TimeCondition != nil

	// Logic conditions
	logicResult := true
	hasLogic := cond.LogicCondition != nil
	if hasLogic {
		logicResult = k.evaluateLogicCondition(ctx, cond.LogicCondition)
	}

	// Compound conditions
	if cond.CompoundCondition != nil {
		return k.evaluateCompoundCondition(ctx, cond.CompoundCondition)
	}

	// If both time and logic present, both must be true
	if hasTime && hasLogic {
		return timeResult && logicResult
	}
	if hasTime {
		return timeResult
	}
	if hasLogic {
		return logicResult
	}

	return false
}

// evaluateLogicCondition evaluates a logic condition by reading from the module's KV store.
func (k Keeper) evaluateLogicCondition(ctx sdk.Context, lc *types.LogicCondition) bool {
	if lc == nil || lc.Type != "state_compare" {
		return false
	}

	// Read value from the module's KV store using the state_key
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get([]byte(lc.StateKey))
	if err != nil || bz == nil {
		return false
	}

	stateValue := new(big.Int)
	stateValue.SetString(string(bz), 10)

	compareValue := new(big.Int)
	if _, ok := compareValue.SetString(lc.CompareValue, 10); !ok {
		return false
	}

	cmp := stateValue.Cmp(compareValue)

	switch lc.Comparator {
	case "eq":
		return cmp == 0
	case "gt":
		return cmp > 0
	case "lt":
		return cmp < 0
	case "gte":
		return cmp >= 0
	case "lte":
		return cmp <= 0
	default:
		return false
	}
}

// evaluateCompoundCondition evaluates AND/OR compound conditions.
func (k Keeper) evaluateCompoundCondition(ctx sdk.Context, cc *types.CompoundCondition) bool {
	if cc == nil || len(cc.Conditions) == 0 {
		return false
	}

	switch cc.Operator {
	case "and":
		for _, sub := range cc.Conditions {
			if !k.evaluateCondition(ctx, sub) {
				return false
			}
		}
		return true
	case "or":
		for _, sub := range cc.Conditions {
			if k.evaluateCondition(ctx, sub) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
