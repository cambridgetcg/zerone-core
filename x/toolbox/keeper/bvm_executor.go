package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// BvmExecutor provides cross-module tool execution for BVM contracts.
type BvmExecutor struct {
	k Keeper
}

// NewBvmExecutor creates a new BvmExecutor backed by the given Keeper.
func NewBvmExecutor(k Keeper) *BvmExecutor {
	return &BvmExecutor{k: k}
}

// CallToolFromBVM resolves and executes a tool call originating from a BVM contract.
// It validates the tool status, routes by type, records the caller, and emits an event.
func (e *BvmExecutor) CallToolFromBVM(ctx context.Context, callerContract, toolID string, input []byte, gasLimit uint64) ([]byte, error) {
	tool, ok := e.k.GetTool(ctx, toolID)
	if !ok {
		return nil, types.ErrToolNotFound.Wrapf("tool %s not found", toolID)
	}

	// Validate status.
	if tool.Status == types.ToolStatusRetired {
		return nil, types.ErrToolRetired.Wrapf("tool %s is retired", toolID)
	}
	if tool.Status == types.ToolStatusDraft {
		return nil, types.ErrInvalidStatus.Wrapf("tool %s is in draft status", toolID)
	}

	// Route by tool type.
	var result []byte
	var execErr error

	switch tool.ToolType {
	case types.ToolTypeBVMContract:
		if e.k.bvmKeeper == nil {
			return nil, fmt.Errorf("bvm keeper not available")
		}
		result, execErr = e.k.bvmKeeper.CallContract(ctx, callerContract, tool.ContractAddress, input, gasLimit)

	case types.ToolTypeKnowledgeTemplate:
		if e.k.knowledgeKeeper == nil {
			return nil, fmt.Errorf("knowledge keeper not available")
		}
		factIDs, err := e.k.knowledgeKeeper.SearchFactsByContent(ctx, "", []string{tool.KnowledgeQuery}, 10)
		if err != nil {
			execErr = err
		} else {
			result = []byte(fmt.Sprintf(`{"facts":%d}`, len(factIDs)))
		}

	case types.ToolTypeComposite:
		result, execErr = e.k.ExecuteCompositeTool(ctx, tool, callerContract, input)

	case types.ToolTypeTreeService:
		return nil, fmt.Errorf("tree_service tools are not yet supported for BVM execution")

	default:
		return nil, fmt.Errorf("unsupported tool type: %s", tool.ToolType)
	}

	if execErr != nil {
		return nil, execErr
	}

	// Record caller.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())
	e.k.RecordCaller(ctx, toolID, callerContract, blockHeight, true)

	// Emit event.
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.toolbox.bvm_call",
			sdk.NewAttribute("tool_id", toolID),
			sdk.NewAttribute("caller_contract", callerContract),
			sdk.NewAttribute("success", "true"),
		),
	)

	return result, nil
}

// CallToolByID is an internal helper that resolves, executes, and returns cost for revenue cascade.
// Returns the tool output, the cost (tool's price_per_call), and any error.
func (e *BvmExecutor) CallToolByID(ctx context.Context, toolID, caller string, input []byte) (output []byte, cost uint64, err error) {
	tool, ok := e.k.GetTool(ctx, toolID)
	if !ok {
		return nil, 0, types.ErrToolNotFound.Wrapf("tool %s not found", toolID)
	}

	// Validate status.
	if tool.Status == types.ToolStatusRetired {
		return nil, 0, types.ErrToolRetired.Wrapf("tool %s is retired", toolID)
	}
	if tool.Status == types.ToolStatusDraft {
		return nil, 0, types.ErrInvalidStatus.Wrapf("tool %s is in draft status", toolID)
	}

	// Cost is the tool's price per call.
	price := parseUint64(tool.PricePerCall)

	// Route by tool type.
	params := e.k.GetParams(ctx)

	switch tool.ToolType {
	case types.ToolTypeBVMContract:
		if e.k.bvmKeeper == nil {
			return nil, 0, fmt.Errorf("bvm keeper not available")
		}
		output, err = e.k.bvmKeeper.CallContract(ctx, caller, tool.ContractAddress, input, params.ToolGasLimit)

	case types.ToolTypeKnowledgeTemplate:
		if e.k.knowledgeKeeper == nil {
			return nil, 0, fmt.Errorf("knowledge keeper not available")
		}
		factIDs, searchErr := e.k.knowledgeKeeper.SearchFactsByContent(ctx, "", []string{tool.KnowledgeQuery}, 10)
		if searchErr != nil {
			err = searchErr
		} else {
			output = []byte(fmt.Sprintf(`{"facts":%d}`, len(factIDs)))
		}

	case types.ToolTypeComposite:
		output, err = e.k.ExecuteCompositeTool(ctx, tool, caller, input)

	case types.ToolTypeTreeService:
		return nil, 0, fmt.Errorf("tree_service tools are not yet supported")

	default:
		return nil, 0, fmt.Errorf("unsupported tool type: %s", tool.ToolType)
	}

	if err != nil {
		return nil, 0, err
	}

	// Record caller.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())
	e.k.RecordCaller(ctx, toolID, caller, blockHeight, true)

	return output, price, nil
}
