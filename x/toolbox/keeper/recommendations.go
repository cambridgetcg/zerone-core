package keeper

import (
	"context"
	"sort"
	"strings"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// RecommendTools returns tool recommendations for an agent using collaborative filtering.
// It finds tools used by agents with similar usage patterns (co-users).
func (k Keeper) RecommendTools(ctx context.Context, agentAddr string, limit uint32) []types.ToolRecommendation {
	if limit == 0 {
		limit = 10
	}

	// Get agent's active tools.
	agentTools := k.GetAgentActiveTools(ctx, agentAddr)
	if len(agentTools) == 0 {
		return nil
	}

	// Build set of agent's own tools for exclusion.
	ownTools := make(map[string]bool, len(agentTools))
	for _, tid := range agentTools {
		ownTools[tid] = true
	}

	// Find co-users: agents who also use the same tools.
	coUserTools := make(map[string]uint64) // toolID -> co-occurrence count
	const maxCoUsersPerTool = 50

	for _, toolID := range agentTools {
		coUserCount := 0
		k.IterateCallerRecords(ctx, toolID, func(rec *types.CallerRecord) bool {
			if rec.Caller == agentAddr {
				return false
			}
			coUserCount++
			if coUserCount > maxCoUsersPerTool {
				return true // Stop after cap.
			}

			// Get this co-user's active tools.
			coTools := k.GetAgentActiveTools(ctx, rec.Caller)
			for _, coTool := range coTools {
				if !ownTools[coTool] {
					coUserTools[coTool]++
				}
			}
			return false
		})
	}

	if len(coUserTools) == 0 {
		return nil
	}

	// Filter: exclude retired tools and tools with trust below Emerging tier.
	type candidate struct {
		toolID string
		name   string
		score  uint64
	}
	var candidates []candidate

	for toolID, count := range coUserTools {
		tool, ok := k.GetTool(ctx, toolID)
		if !ok || tool.Status == types.ToolStatusRetired {
			continue
		}
		if types.TrustTier(tool.TrustScore) < types.TrustTierIDEmerging {
			continue
		}

		// Relevance score: co-occurrence count scaled to BPS range.
		// Cap at 10 co-occurrences for max score.
		if count > 10 {
			count = 10
		}
		score := safeMulDiv(count, types.BpsDenominator, 10)

		candidates = append(candidates, candidate{
			toolID: toolID,
			name:   tool.Name,
			score:  score,
		})
	}

	// Sort by score descending.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Truncate to limit.
	if uint32(len(candidates)) > limit {
		candidates = candidates[:limit]
	}

	result := make([]types.ToolRecommendation, len(candidates))
	for i, c := range candidates {
		result[i] = types.ToolRecommendation{
			ToolID:         c.toolID,
			ToolName:       c.name,
			RelevanceScore: c.score,
		}
	}
	return result
}

// MatchToolsForAgent returns active tools whose required capabilities are a subset
// of the agent's capabilities. Falls back to all active tools with no required caps
// if discoveryKeeper is nil.
func (k Keeper) MatchToolsForAgent(ctx context.Context, agentAddr string, limit uint32) []*types.Tool {
	if limit == 0 {
		limit = 10
	}

	var agentCaps map[string]bool

	if k.discoveryKeeper != nil {
		caps, err := k.discoveryKeeper.GetAgentCapabilityTypes(ctx, agentAddr)
		if err == nil && len(caps) > 0 {
			agentCaps = make(map[string]bool, len(caps))
			for _, c := range caps {
				agentCaps[strings.ToLower(c)] = true
			}
		}
	}

	var matched []*types.Tool
	k.IterateTools(ctx, func(tool *types.Tool) bool {
		if tool.Status != types.ToolStatusActive {
			return false
		}

		// If the tool has required capabilities, check subset.
		if len(tool.RequiredCapabilities) > 0 && agentCaps != nil {
			for _, rc := range tool.RequiredCapabilities {
				if !agentCaps[strings.ToLower(rc)] {
					return false // Agent lacks a required capability.
				}
			}
		}

		// If no agent caps available, only return tools with no required caps.
		if agentCaps == nil && len(tool.RequiredCapabilities) > 0 {
			return false
		}

		matched = append(matched, tool)
		if uint32(len(matched)) >= limit {
			return true
		}
		return false
	})

	return matched
}

// GetAgentUsedToolIDs returns the list of tool IDs an agent has used (non-zero usage).
func (k Keeper) GetAgentUsedToolIDs(ctx context.Context, agentAddr string) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.AgentToolUsageIterPrefix(agentAddr)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var toolIDs []string
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		toolID := string(key[len(prefix):])
		toolIDs = append(toolIDs, toolID)
	}
	return toolIDs
}
