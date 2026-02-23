package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// RecordToolCall records a call in both per-tool and global demand windows.
func (k Keeper) RecordToolCall(ctx context.Context, toolID string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())

	// Per-tool demand window.
	dw := k.getDemandWindow(ctx, toolID)
	k.recordWindow(dw, blockHeight)
	k.setDemandWindow(ctx, dw)

	// Global demand window.
	gdw := k.getDemandWindow(ctx, types.GlobalDemandToolID)
	k.recordWindow(gdw, blockHeight)
	k.setDemandWindow(ctx, gdw)
}

// GetToolDemand returns the total calls and utilisation BPS for a specific tool.
func (k Keeper) GetToolDemand(ctx context.Context, toolID string) (totalCalls uint64, utilisation uint64) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)
	dw := k.getDemandWindow(ctx, toolID)
	target := params.TargetCallsPerBlockPerTool
	if target == 0 {
		target = types.DefaultTargetCallsPerBlockPerTool
	}
	totalCalls = sumWindow(dw, currentBlock)
	utilisation = utilisationBps(totalCalls, dw.Size, target)
	return totalCalls, utilisation
}

// GetGlobalDemand returns the total calls and utilisation BPS for the entire chain.
func (k Keeper) GetGlobalDemand(ctx context.Context) (totalCalls uint64, utilisation uint64) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)
	dw := k.getDemandWindow(ctx, types.GlobalDemandToolID)
	target := params.TargetGlobalCallsPerBlock
	if target == 0 {
		target = types.DefaultTargetGlobalCallsPerBlock
	}
	totalCalls = sumWindow(dw, currentBlock)
	utilisation = utilisationBps(totalCalls, dw.Size, target)
	return totalCalls, utilisation
}

// recordWindow records a call in the circular buffer demand window.
func (k Keeper) recordWindow(dw *types.DemandWindow, blockHeight uint64) {
	if dw.Size == 0 {
		return
	}
	// Ensure entries slice is the right size.
	for uint64(len(dw.Entries)) < dw.Size {
		dw.Entries = append(dw.Entries, nil)
	}

	idx := dw.Head % dw.Size
	entry := dw.Entries[idx]
	if entry != nil && entry.BlockHeight == blockHeight {
		entry.CallCount++
	} else {
		// Advance head.
		dw.Head++
		idx = dw.Head % dw.Size
		dw.Entries[idx] = &types.DemandWindowEntry{
			BlockHeight: blockHeight,
			CallCount:   1,
		}
	}
}

// sumWindow sums call counts for entries within the active window.
// Only entries where BlockHeight > cutoff && BlockHeight <= currentBlock are counted.
func sumWindow(dw *types.DemandWindow, currentBlock uint64) uint64 {
	windowSize := dw.Size
	if windowSize == 0 {
		return 0
	}
	var cutoff uint64
	if currentBlock > windowSize {
		cutoff = currentBlock - windowSize
	}

	var total uint64
	for _, entry := range dw.Entries {
		if entry != nil && entry.BlockHeight > cutoff && entry.BlockHeight <= currentBlock {
			total += entry.CallCount
		}
	}
	return total
}

// utilisationBps computes utilisation as (totalCalls / (windowSize * targetPerBlock)) * BPS,
// capped at 1,000,000.
func utilisationBps(totalCalls, windowSize, targetPerBlock uint64) uint64 {
	capacity := windowSize * targetPerBlock
	if capacity == 0 {
		return 0
	}
	util := safeMulDiv(totalCalls, types.BpsDenominator, capacity)
	if util > types.BpsDenominator {
		util = types.BpsDenominator
	}
	return util
}
