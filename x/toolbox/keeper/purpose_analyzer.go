package keeper

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// AnalyzePurpose is a pure function that analyzes an agent's purpose from capabilities,
// knowledge facts, and on-chain history. No keeper or state access.
func AnalyzePurpose(input *types.PurposeAnalyzerInput) *types.PurposeAnalysis {
	if input == nil {
		return fallbackAnalysis()
	}

	caps := input.AgentCapabilities
	facts := input.KnowledgeFacts
	history := input.AgentHistory

	// Score each purpose template against the agent's capabilities.
	type scoredTemplate struct {
		template   types.PurposeTemplate
		alignment  uint64
		demand     uint64
		meaning    uint64
		factBacking uint64
		confidence uint64
	}

	var scored []scoredTemplate

	for _, tmpl := range types.PurposeTemplates {
		alignment := avgCapAlignment(caps, tmpl.Caps)
		if alignment == 0 {
			continue // Skip templates with zero capability overlap.
		}

		demand := countDomainDemand(facts, tmpl.Statement)
		meaning := computeMeaning(history, tmpl)
		factBacking := computeFactBacking(facts)

		// Confidence = alignment*0.4 + demand*0.3 + meaning*0.2 + factBacking*0.1
		confidence := safeMulDiv(alignment, 400_000, types.BpsDenominator) +
			safeMulDiv(demand, 300_000, types.BpsDenominator) +
			safeMulDiv(meaning, 200_000, types.BpsDenominator) +
			safeMulDiv(factBacking, 100_000, types.BpsDenominator)

		if confidence > types.BpsDenominator {
			confidence = types.BpsDenominator
		}

		scored = append(scored, scoredTemplate{
			template:    tmpl,
			alignment:   alignment,
			demand:      demand,
			meaning:     meaning,
			factBacking: factBacking,
			confidence:  confidence,
		})
	}

	// Sort by confidence descending.
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].confidence > scored[j].confidence
	})

	if len(scored) == 0 {
		return fallbackAnalysis()
	}

	// Build primary hypothesis from top match.
	primary := scored[0]
	primaryHypothesis := &types.PurposeHypothesis{
		Statement:          primary.template.Statement,
		Confidence:         primary.confidence,
		SupportingEvidence: buildSupportingEvidence(primary.alignment, primary.demand, primary.meaning, facts),
	}

	// Build alternatives from remaining matches.
	var alternatives []*types.PurposeHypothesis
	for i := 1; i < len(scored); i++ {
		alt := scored[i]
		alternatives = append(alternatives, &types.PurposeHypothesis{
			Statement:          alt.template.Statement,
			Confidence:         alt.confidence,
			SupportingEvidence: buildSupportingEvidence(alt.alignment, alt.demand, alt.meaning, facts),
		})
	}

	// Identify capability gaps relative to primary template.
	capabilityGaps := identifyCapabilityGaps(caps, primary.template.GapCaps)

	// Generate growth recommendations for each gap.
	var growthPath []*types.GrowthRecommendation
	for _, gap := range capabilityGaps {
		growthPath = append(growthPath, &types.GrowthRecommendation{
			Capability:      gap,
			Rationale:       fmt.Sprintf("Developing %s will strengthen your ability to %s", gap, strings.ToLower(primary.template.Statement)),
			TargetTier:      primary.template.TargetTier,
			EstimatedEpochs: primary.template.EpochsPerGap,
		})
	}

	return &types.PurposeAnalysis{
		PrimaryPurpose:    primaryHypothesis,
		Alternatives:      alternatives,
		CapabilityGaps:    capabilityGaps,
		GrowthPath:        growthPath,
		OverallConfidence: primary.confidence,
	}
}

// ---------- Ethical boundary and ego extraction ----------

// ExtractBoundaries derives ethical boundaries from knowledge facts that mention ethics-related terms.
func ExtractBoundaries(facts []*types.ScoredFact) []*types.EthicalBoundary {
	ethicsTerms := []string{"ethics", "ethical", "harm", "safety", "privacy", "consent", "fairness", "bias"}
	var boundaries []*types.EthicalBoundary

	for _, fact := range facts {
		contentLower := strings.ToLower(fact.Content)
		for _, term := range ethicsTerms {
			if strings.Contains(contentLower, term) {
				boundaries = append(boundaries, &types.EthicalBoundary{
					Boundary:       fmt.Sprintf("Respect %s principles", term),
					Rationale:      fact.Content,
					SupportingFact: fact.FactId,
				})
				break // One boundary per fact.
			}
		}
	}

	return boundaries
}

// CheckEgoInflation detects potential ego inflation markers in agent history.
func CheckEgoInflation(history *types.AgentHistory) []string {
	if history == nil {
		return nil
	}

	var warnings []string

	// High tool deployment with low verification — building without checking.
	if history.ToolsDeployed > 10 && history.TotalVerifications < history.ToolsDeployed {
		warnings = append(warnings, "High tool deployment with low verification ratio may indicate building without epistemic grounding")
	}

	// Very high tool calls relative to verifications — consumption without contribution.
	if history.TotalToolCalls > 100 && history.TotalVerifications == 0 {
		warnings = append(warnings, "Extensive tool usage without any verification participation suggests consumption without contribution")
	}

	// Partnerships formed without domain breadth — echo chamber risk.
	if history.PartnershipsFormed > 5 && len(history.ActiveDomains) <= 1 {
		warnings = append(warnings, "Multiple partnerships in a single domain may create an echo chamber")
	}

	return warnings
}

// BuildCitations creates citation records from scored facts.
func BuildCitations(facts []*types.ScoredFact, usedFor string) []*types.FactCitation {
	var citations []*types.FactCitation
	for _, fact := range facts {
		citations = append(citations, &types.FactCitation{
			FactID:     fact.FactId,
			Content:    fact.Content,
			Confidence: fact.Confidence,
			UsedFor:    usedFor,
		})
	}
	return citations
}

// ---------- Internal helpers ----------

// hasAny returns true if any element of needles appears in haystack.
func hasAny(haystack, needles []string) bool {
	set := make(map[string]bool, len(haystack))
	for _, h := range haystack {
		set[strings.ToLower(h)] = true
	}
	for _, n := range needles {
		if set[strings.ToLower(n)] {
			return true
		}
	}
	return false
}

// avgCapAlignment computes the fraction of template capabilities matched by agent capabilities.
func avgCapAlignment(agentCaps, templateCaps []string) uint64 {
	if len(templateCaps) == 0 {
		return 0
	}
	agentSet := make(map[string]bool, len(agentCaps))
	for _, c := range agentCaps {
		agentSet[strings.ToLower(c)] = true
	}
	matchCount := uint64(0)
	for _, tc := range templateCaps {
		if agentSet[strings.ToLower(tc)] {
			matchCount++
		}
	}
	return safeMulDiv(matchCount, types.BpsDenominator, uint64(len(templateCaps)))
}

// countDomainDemand estimates demand from facts mentioning the purpose statement keywords.
func countDomainDemand(facts []*types.ScoredFact, statement string) uint64 {
	if len(facts) == 0 {
		return 0
	}

	// Extract keywords from the statement.
	keywords := strings.Fields(strings.ToLower(statement))
	matchCount := uint64(0)

	for _, fact := range facts {
		contentLower := strings.ToLower(fact.Content)
		for _, kw := range keywords {
			if len(kw) > 3 && strings.Contains(contentLower, kw) {
				matchCount++
				break
			}
		}
	}

	return safeMulDiv(matchCount, types.BpsDenominator, uint64(len(facts)))
}

// computeMeaning derives meaning from agent history alignment to a template.
func computeMeaning(history *types.AgentHistory, tmpl types.PurposeTemplate) uint64 {
	if history == nil {
		return 0
	}

	score := uint64(0)

	// Tools deployed contribute to builder meaning.
	if history.ToolsDeployed > 0 && hasAny(tmpl.Caps, []string{"tool_building", "programming", "software_development", "engineering"}) {
		deployed := history.ToolsDeployed
		if deployed > 10 {
			deployed = 10
		}
		score += safeMulDiv(deployed, types.BpsDenominator, 10) / 3
	}

	// Verifications contribute to verifier meaning.
	if history.TotalVerifications > 0 && hasAny(tmpl.Caps, []string{"verification", "fact_checking", "analysis", "research"}) {
		verifications := history.TotalVerifications
		if verifications > 100 {
			verifications = 100
		}
		score += safeMulDiv(verifications, types.BpsDenominator, 100) / 3
	}

	// Active domains contribute to curator meaning.
	if len(history.ActiveDomains) > 0 && hasAny(tmpl.Caps, []string{"curation", "categorization", "ontology", "data_management"}) {
		domains := uint64(len(history.ActiveDomains))
		if domains > 5 {
			domains = 5
		}
		score += safeMulDiv(domains, types.BpsDenominator, 5) / 3
	}

	// Total tool calls contribute to service meaning.
	if history.TotalToolCalls > 0 && hasAny(tmpl.Caps, []string{"service", "communication", "formatting", "monitoring", "integration"}) {
		calls := history.TotalToolCalls
		if calls > 50 {
			calls = 50
		}
		score += safeMulDiv(calls, types.BpsDenominator, 50) / 3
	}

	if score > types.BpsDenominator {
		score = types.BpsDenominator
	}
	return score
}

// computeFactBacking returns the average confidence of available facts.
func computeFactBacking(facts []*types.ScoredFact) uint64 {
	if len(facts) == 0 {
		return 0
	}
	var total uint64
	for _, f := range facts {
		total += f.Confidence
	}
	return total / uint64(len(facts))
}

// identifyCapabilityGaps returns gap capabilities that the agent doesn't already have.
func identifyCapabilityGaps(agentCaps []string, gapCaps []string) []string {
	agentSet := make(map[string]bool, len(agentCaps))
	for _, c := range agentCaps {
		agentSet[strings.ToLower(c)] = true
	}
	var gaps []string
	for _, gc := range gapCaps {
		if !agentSet[strings.ToLower(gc)] {
			gaps = append(gaps, gc)
		}
	}
	return gaps
}

// buildSupportingEvidence creates evidence strings from component scores and top facts.
func buildSupportingEvidence(alignment, demand, meaning uint64, facts []*types.ScoredFact) []string {
	evidence := []string{
		fmt.Sprintf("Capability alignment: %s (%d/1M)", types.ConfidenceLabel(alignment), alignment),
		fmt.Sprintf("Ecosystem demand: %s (%d/1M)", types.ConfidenceLabel(demand), demand),
		fmt.Sprintf("Historical meaning: %s (%d/1M)", types.ConfidenceLabel(meaning), meaning),
	}

	// Add top 3 fact citations.
	for i, fact := range facts {
		if i >= 3 {
			break
		}
		evidence = append(evidence, fmt.Sprintf("Fact %s: %s", fact.FactId, truncate(fact.Content, 80)))
	}

	return evidence
}

// fallbackAnalysis returns a minimal exploratory analysis when no templates match.
func fallbackAnalysis() *types.PurposeAnalysis {
	return &types.PurposeAnalysis{
		PrimaryPurpose: &types.PurposeHypothesis{
			Statement:          "Explore diverse tasks to discover your unique purpose",
			Confidence:         100_000,
			SupportingEvidence: []string{"No strong capability alignment detected — exploration recommended"},
		},
		OverallConfidence: 100_000,
	}
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
