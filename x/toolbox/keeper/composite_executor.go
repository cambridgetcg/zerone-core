package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// ExecuteCompositeTool runs the Purpose Prompter pipeline for composite tools.
// Pipeline: Knowledge Scout → Purpose Analyzer → Path Formatter → Recommendations.
func (k Keeper) ExecuteCompositeTool(ctx context.Context, tool *types.Tool, caller string, input []byte) ([]byte, error) {
	if tool.ToolType != types.ToolTypeComposite {
		return nil, fmt.Errorf("tool %s is not a composite tool", tool.Id)
	}

	// Phase 1: Knowledge Scout — gather relevant facts.
	scoutInput := k.buildScoutInput(tool, input)
	scoutOutput, err := k.KnowledgeScout(ctx, scoutInput)
	if err != nil {
		return nil, fmt.Errorf("knowledge scout failed: %w", err)
	}

	// Phase 2: Purpose Analyzer — analyze agent purpose from gathered knowledge.
	analyzerInput := k.buildAnalyzerInput(ctx, caller, scoutOutput)
	analysis := AnalyzePurpose(analyzerInput)

	// Phase 3: Path Formatter — format the analysis into actionable path.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())
	path := FormatPath(analysis, caller, blockHeight)

	// Phase 4: Recommendations — collaborative filtering for tool suggestions.
	recommendations := k.RecommendTools(ctx, caller, 10)

	// Build composite output.
	boundaries := ExtractBoundaries(scoutOutput.Facts)
	egoWarnings := CheckEgoInflation(analyzerInput.AgentHistory)
	citations := BuildCitations(scoutOutput.Facts, "purpose_analysis")

	output := &types.PurposePrompterOutput{
		Analysis:        analysis,
		Path:            path,
		Recommendations: recommendations,
		Boundaries:      boundaries,
		EgoWarnings:     egoWarnings,
		Citations:       citations,
		Methodology:     "4-phase pipeline: Knowledge Scout → Purpose Analyzer → Path Formatter → Tool Recommender",
	}

	return json.Marshal(output)
}

// buildScoutInput constructs a KnowledgeScoutInput from the composite tool and raw input.
func (k Keeper) buildScoutInput(tool *types.Tool, input []byte) *types.KnowledgeScoutInput {
	si := &types.KnowledgeScoutInput{
		MaxResults:    types.DefaultScoutMaxResults,
		MinConfidence: types.DefaultScoutMinConfidence,
	}

	// Try to parse input as a scout input override.
	if len(input) > 0 {
		var override types.KnowledgeScoutInput
		if json.Unmarshal(input, &override) == nil {
			if len(override.QueryTerms) > 0 {
				si.QueryTerms = override.QueryTerms
			}
			if override.Domain != "" {
				si.Domain = override.Domain
			}
			if override.MaxResults > 0 {
				si.MaxResults = override.MaxResults
			}
			if override.MinConfidence > 0 {
				si.MinConfidence = override.MinConfidence
			}
			if len(override.Capabilities) > 0 {
				si.Capabilities = override.Capabilities
			}
		}
	}

	// Fallback: use the tool's knowledge query as a query term.
	if len(si.QueryTerms) == 0 && tool.KnowledgeQuery != "" {
		si.QueryTerms = []string{tool.KnowledgeQuery}
	}

	return si
}

// buildAnalyzerInput constructs a PurposeAnalyzerInput from scout output and on-chain data.
func (k Keeper) buildAnalyzerInput(ctx context.Context, caller string, scoutOutput *types.KnowledgeScoutOutput) *types.PurposeAnalyzerInput {
	input := &types.PurposeAnalyzerInput{
		KnowledgeFacts: scoutOutput.Facts,
	}

	// Get agent capabilities (nil-safe).
	if k.discoveryKeeper != nil {
		caps, err := k.discoveryKeeper.GetAgentCapabilityTypes(ctx, caller)
		if err == nil {
			input.AgentCapabilities = caps
		}
	}

	// Build agent history from on-chain data.
	input.AgentHistory = k.buildAgentHistory(ctx, caller)

	return input
}

// buildAgentHistory constructs AgentHistory from on-chain tool usage and deployment data.
func (k Keeper) buildAgentHistory(ctx context.Context, agentAddr string) *types.AgentHistory {
	history := &types.AgentHistory{}

	// Count tools deployed by this agent.
	deployedTools := k.GetToolsByDeployer(ctx, agentAddr)
	history.ToolsDeployed = uint64(len(deployedTools))

	// Count total tool calls from usage records.
	usedTools := k.GetAgentUsedToolIDs(ctx, agentAddr)
	var totalCalls uint64
	for _, toolID := range usedTools {
		totalCalls += k.GetAgentToolUsage(ctx, agentAddr, toolID)
	}
	history.TotalToolCalls = totalCalls

	// Active domains from deployed tools.
	domainSet := make(map[string]bool)
	for _, tool := range deployedTools {
		if tool.Category != "" {
			domainSet[tool.Category] = true
		}
	}
	for d := range domainSet {
		history.ActiveDomains = append(history.ActiveDomains, d)
	}

	return history
}

// ---------- Revenue Cascade ----------

// CompositeResult holds the output and cost breakdown of a composite execution with revenue cascade.
type CompositeResult struct {
	Output         []byte   `json:"output"`
	TotalPayment   uint64   `json:"total_payment"`
	DependencyCost uint64   `json:"dependency_cost"`
	OwnRevenue     uint64   `json:"own_revenue"`
	SubCallIDs     []string `json:"sub_call_ids,omitempty"`
}

// ExecuteCompositeWithCascade runs a composite tool, executing all dependencies in
// topological order first (revenue cascade), then the composite's own PP pipeline.
func (k Keeper) ExecuteCompositeWithCascade(ctx context.Context, tool *types.Tool, caller string, input []byte, payment uint64) (*CompositeResult, error) {
	if tool.ToolType != types.ToolTypeComposite {
		return nil, fmt.Errorf("tool %s is not a composite tool", tool.Id)
	}

	executor := NewBvmExecutor(k)
	result := &CompositeResult{TotalPayment: payment}

	// Flatten dependencies in topological (post-) order, excluding the tool itself.
	deps := k.FlattenDependencies(ctx, tool.Id)

	var dependencyCost uint64
	for _, depID := range deps {
		if depID == tool.Id {
			continue
		}

		// Execute each dependency tool, accumulating cost.
		_, cost, err := executor.CallToolByID(ctx, depID, caller, input)
		if err != nil {
			return nil, fmt.Errorf("dependency %s execution failed: %w", depID, err)
		}
		dependencyCost += cost
		result.SubCallIDs = append(result.SubCallIDs, depID)
	}

	result.DependencyCost = dependencyCost

	// Execute the composite tool's own PP pipeline.
	output, err := k.ExecuteCompositeTool(ctx, tool, caller, input)
	if err != nil {
		return nil, fmt.Errorf("composite pipeline failed: %w", err)
	}
	result.Output = output

	// Own revenue = payment - dependency cost (floored at 0).
	if payment > dependencyCost {
		result.OwnRevenue = payment - dependencyCost
	}

	return result, nil
}
