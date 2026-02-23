package keeper

import (
	"context"
	"sort"
	"strings"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// KnowledgeScout gathers and scores relevant facts from the knowledge module.
// Nil-safe: returns empty output if knowledgeKeeper is nil.
func (k Keeper) KnowledgeScout(ctx context.Context, input *types.KnowledgeScoutInput) (*types.KnowledgeScoutOutput, error) {
	if k.knowledgeKeeper == nil {
		return &types.KnowledgeScoutOutput{TotalFound: 0}, nil
	}

	if input == nil {
		return &types.KnowledgeScoutOutput{TotalFound: 0}, nil
	}

	// Apply defaults.
	maxResults := input.MaxResults
	if maxResults == 0 {
		maxResults = types.DefaultScoutMaxResults
	}
	minConfidence := input.MinConfidence
	if minConfidence == 0 {
		minConfidence = types.DefaultScoutMinConfidence
	}

	queryTerms := input.QueryTerms
	if len(queryTerms) == 0 {
		return &types.KnowledgeScoutOutput{TotalFound: 0}, nil
	}

	// Search for facts matching the query terms.
	factIDs, err := k.knowledgeKeeper.SearchFactsByContent(ctx, input.Domain, queryTerms, maxResults*2)
	if err != nil {
		// Non-fatal: return empty output on search failure.
		return &types.KnowledgeScoutOutput{TotalFound: 0}, nil
	}

	var facts []*types.ScoredFact
	for _, factID := range factIDs {
		content, confidence, citations, err := k.knowledgeKeeper.GetFactDetails(ctx, factID)
		if err != nil {
			continue
		}

		// Filter by minimum confidence.
		if confidence < minConfidence {
			continue
		}

		relevance := computeRelevanceScore(content, queryTerms, confidence, input.Capabilities)

		facts = append(facts, &types.ScoredFact{
			FactId:         factID,
			Content:        content,
			Confidence:     confidence,
			CitationCount:  citations,
			RelevanceScore: relevance,
		})

		// Record citation for knowledge graph attribution.
		_ = k.knowledgeKeeper.RecordFactCitation(ctx, factID, "purpose-scout")
	}

	// Sort by relevance score descending.
	sort.Slice(facts, func(i, j int) bool {
		return facts[i].RelevanceScore > facts[j].RelevanceScore
	})

	// Truncate to maxResults.
	if uint64(len(facts)) > maxResults {
		facts = facts[:maxResults]
	}

	return &types.KnowledgeScoutOutput{
		Facts:      facts,
		TotalFound: uint64(len(facts)),
	}, nil
}

// computeRelevanceScore calculates how relevant a fact is to the query and agent capabilities.
// Formula: termOverlap*400K + confidence/3 + capRelevance*200K, capped at 1M.
func computeRelevanceScore(content string, queryTerms []string, confidence uint64, capabilities []string) uint64 {
	contentLower := strings.ToLower(content)

	// Term overlap: fraction of query terms found in content.
	matchCount := uint64(0)
	for _, term := range queryTerms {
		if strings.Contains(contentLower, strings.ToLower(term)) {
			matchCount++
		}
	}
	termOverlap := uint64(0)
	if len(queryTerms) > 0 {
		termOverlap = safeMulDiv(matchCount, types.BpsDenominator, uint64(len(queryTerms)))
	}

	// Capability relevance: does the content mention any agent capabilities?
	capMatches := uint64(0)
	for _, cap := range capabilities {
		if strings.Contains(contentLower, strings.ToLower(cap)) {
			capMatches++
		}
	}
	capRelevance := uint64(0)
	if len(capabilities) > 0 {
		capRelevance = safeMulDiv(capMatches, types.BpsDenominator, uint64(len(capabilities)))
	}

	// Combine: 40% term overlap + 33% confidence + 20% capability relevance.
	score := safeMulDiv(termOverlap, 400_000, types.BpsDenominator) +
		confidence/3 +
		safeMulDiv(capRelevance, 200_000, types.BpsDenominator)

	if score > types.BpsDenominator {
		score = types.BpsDenominator
	}
	return score
}
