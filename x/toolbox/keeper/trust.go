package keeper

import (
	"context"
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// Trust component weight constants (BPS, sum to 1,000,000).
const (
	weightUsageBps        = 300_000 // 30%
	weightVerificationBps = 250_000 // 25%
	weightReliabilityBps  = 200_000 // 20%
	weightPeerBps         = 150_000 // 15%
	weightContributorBps  = 100_000 // 10%
)

// EMA half-life and scoring constants.
const (
	emaAlphaBps  = 100_000 // 10% — weight of new data point
	initialTrust = uint64(500_000)
	maxScore     = uint64(types.BpsDenominator)
)

// Reference-quality scoring constants.
const (
	// Usage component.
	MinUniqueCallers       = uint64(20)
	UsageRecencyHalfLife   = uint64(240_000)  // blocks
	DeduplicationWindowBlks = uint64(10)       // ignore repeat calls within N blocks

	// Reliability component.
	MinCallsForReliability = uint64(100)
	NeutralReliability     = uint64(500_000)

	// Peer component.
	MaxPeerHops          = 3
	PeerDampeningBps     = uint64(500_000)  // 50% per hop
	SameAuthorPenaltyBps = uint64(500_000)  // 50% penalty
	peerTarget           = uint64(10)

	// Contributor tier scores.
	TierScoreApprentice = uint64(100_000)
	TierScoreVerified   = uint64(400_000)
	TierScoreBonded     = uint64(700_000)
	TierScoreGuardian   = uint64(1_000_000)
)

// safeMulDiv computes (a * b) / c using big.Int to avoid overflow.
func safeMulDiv(a, b, c uint64) uint64 {
	if c == 0 {
		return 0
	}
	ab := new(big.Int).Mul(new(big.Int).SetUint64(a), new(big.Int).SetUint64(b))
	result := new(big.Int).Div(ab, new(big.Int).SetUint64(c))
	if !result.IsUint64() {
		return maxScore
	}
	v := result.Uint64()
	if v > maxScore {
		return maxScore
	}
	return v
}

// InitialTrustScore returns the starting trust score for a new tool.
func InitialTrustScore() uint64 {
	return initialTrust
}

// UpdateTrustScore performs an EMA update of the tool's trust score after a call.
func (k Keeper) UpdateTrustScore(ctx context.Context, tool *types.Tool, success bool) {
	observation := maxScore
	if !success {
		observation = 0
	}

	// EMA: new = alpha * observation + (1 - alpha) * old
	old := tool.TrustScore
	newScore := safeMulDiv(emaAlphaBps, observation, types.BpsDenominator) +
		safeMulDiv(types.BpsDenominator-emaAlphaBps, old, types.BpsDenominator)

	if newScore > maxScore {
		newScore = maxScore
	}
	tool.TrustScore = newScore
}

// ComputeTrustScore computes the full 5-component trust score for a tool.
func (k Keeper) ComputeTrustScore(ctx context.Context, tool *types.Tool) *types.TrustSnapshot {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())

	usage := k.calculateUsageScore(ctx, tool, blockHeight)
	verification := k.calculateVerificationScore(ctx, tool)
	reliability := k.calculateReliabilityScore(ctx, tool)
	peer := k.calculatePeerScore(ctx, tool)
	contributor := k.calculateContributorScore(ctx, tool)

	totalScore := safeMulDiv(usage, weightUsageBps, types.BpsDenominator) +
		safeMulDiv(verification, weightVerificationBps, types.BpsDenominator) +
		safeMulDiv(reliability, weightReliabilityBps, types.BpsDenominator) +
		safeMulDiv(peer, weightPeerBps, types.BpsDenominator) +
		safeMulDiv(contributor, weightContributorBps, types.BpsDenominator)

	if totalScore > maxScore {
		totalScore = maxScore
	}

	snap, _ := k.GetTrustSnapshot(ctx, tool.Id)
	callerCount := k.CountUniqueCallers(ctx, tool.Id)

	successRate := uint64(0)
	if snap != nil && snap.TotalCallsWindow > 0 {
		successRate = safeMulDiv(snap.TotalCallsWindow-snap.UniqueCallersWindow, types.BpsDenominator, snap.TotalCallsWindow)
	}

	return &types.TrustSnapshot{
		ToolId:                tool.Id,
		Score:                 totalScore,
		UsageComponent:        usage,
		VerificationComponent: verification,
		ReliabilityComponent:  reliability,
		PeerComponent:         peer,
		ContributorComponent:  contributor,
		UniqueCallersWindow:   callerCount,
		TotalCallsWindow:      0, // Updated during demand tracking
		SuccessRateBps:        successRate,
		ComputedAtBlock:       blockHeight,
	}
}

// calculateUsageScore: recency-weighted unique callers with self-caller exclusion.
func (k Keeper) calculateUsageScore(ctx context.Context, tool *types.Tool, blockHeight uint64) uint64 {
	// Build contributor set for self-caller exclusion.
	contributorSet := make(map[string]bool)
	for _, c := range tool.Contributors {
		contributorSet[c.Address] = true
	}
	contributorSet[tool.Deployer] = true

	var uniqueCallers uint64
	var recencySum uint64
	var recencyCount uint64

	k.IterateCallerRecords(ctx, tool.Id, func(rec *types.CallerRecord) bool {
		// Exclude contributors/deployer from unique caller count.
		if contributorSet[rec.Caller] {
			return false
		}
		uniqueCallers++

		// Recency decay: halfLife / (halfLife + blocksSince)
		if rec.LastCallBlock > 0 && blockHeight >= rec.LastCallBlock {
			blocksSince := blockHeight - rec.LastCallBlock
			recency := safeMulDiv(UsageRecencyHalfLife, maxScore, UsageRecencyHalfLife+blocksSince)
			recencySum += recency
			recencyCount++
		}
		return false
	})

	// Average recency across callers.
	avgRecency := uint64(0)
	if recencyCount > 0 {
		avgRecency = recencySum / recencyCount
	}

	// Scale by uniqueCallers / MinUniqueCallers (capped at 1.0).
	callerScale := maxScore
	if uniqueCallers < MinUniqueCallers {
		callerScale = safeMulDiv(uniqueCallers, maxScore, MinUniqueCallers)
	}

	// Combine: 50% caller breadth + 50% recency-weighted activity.
	return (callerScale + safeMulDiv(callerScale, avgRecency, maxScore)) / 2
}

// calculateVerificationScore: knowledge-backed verification with hash fallback.
func (k Keeper) calculateVerificationScore(ctx context.Context, tool *types.Tool) uint64 {
	// Verified tools get a floor of 700K.
	if tool.IsVerified {
		score := uint64(700_000)

		// If knowledge keeper is available and tool has a query, boost with fact confidence.
		if k.knowledgeKeeper != nil && tool.KnowledgeQuery != "" {
			factIDs, err := k.knowledgeKeeper.SearchFactsByContent(ctx, "", []string{tool.KnowledgeQuery}, 10)
			if err == nil && len(factIDs) > 0 {
				var totalConf uint64
				var count uint64
				for _, factID := range factIDs {
					conf, ok := k.knowledgeKeeper.GetFactConfidence(ctx, factID)
					if ok {
						totalConf += conf
						count++
					}
				}
				if count > 0 {
					avgConf := totalConf / count
					// Blend: 70% floor + 30% knowledge confidence.
					score = safeMulDiv(700_000, 700_000, maxScore) + safeMulDiv(avgConf, 300_000, maxScore)
				}
			}
		}

		if score > maxScore {
			score = maxScore
		}
		return score
	}

	// Unverified fallback: hash-based scoring.
	score := uint64(0)
	if tool.SourceHash != "" {
		score += 300_000
	}
	if tool.DocumentationHash != "" {
		score += 200_000
	}
	if tool.ApiSchema != "" {
		score += 100_000
	}
	if score > maxScore {
		score = maxScore
	}
	return score
}

// calculateReliabilityScore: success rate with neutral blending for low sample sizes.
func (k Keeper) calculateReliabilityScore(ctx context.Context, tool *types.Tool) uint64 {
	var totalCalls, successCalls uint64

	k.IterateCallerRecords(ctx, tool.Id, func(rec *types.CallerRecord) bool {
		totalCalls += rec.TotalCalls
		successCalls += rec.SuccessCount
		return false
	})

	if totalCalls == 0 {
		return NeutralReliability
	}

	rawRate := safeMulDiv(successCalls, maxScore, totalCalls)

	// Below MinCallsForReliability: blend toward NeutralReliability proportionally.
	if totalCalls < MinCallsForReliability {
		// weight = totalCalls / MinCallsForReliability
		weight := safeMulDiv(totalCalls, maxScore, MinCallsForReliability)
		// blended = weight * rawRate + (1 - weight) * NeutralReliability
		return safeMulDiv(weight, rawRate, maxScore) +
			safeMulDiv(maxScore-weight, NeutralReliability, maxScore)
	}

	return rawRate
}

// calculatePeerScore: dampened BFS traversal of dependents with deployer penalty.
func (k Keeper) calculatePeerScore(ctx context.Context, tool *types.Tool) uint64 {
	var peers []uint64
	visited := make(map[string]bool)
	visited[tool.Id] = true

	k.collectPeerScores(ctx, tool.Id, tool.Deployer, 0, maxScore, visited, &peers)

	if len(peers) == 0 {
		return 0
	}

	// Average of dampened trust scores.
	var sum uint64
	for _, s := range peers {
		sum += s
	}
	avg := sum / uint64(len(peers))

	// Scale by count vs target (10).
	count := uint64(len(peers))
	if count >= peerTarget {
		return avg
	}
	return safeMulDiv(avg, count, peerTarget)
}

// collectPeerScores performs BFS/DFS up to MaxPeerHops collecting dampened peer trust scores.
func (k Keeper) collectPeerScores(ctx context.Context, toolID, originalDeployer string, depth int, dampening uint64, visited map[string]bool, peers *[]uint64) {
	if depth >= MaxPeerHops {
		return
	}

	k.IterateDependentsOf(ctx, toolID, func(depToolID string) bool {
		if visited[depToolID] {
			return false
		}
		visited[depToolID] = true

		depTool, ok := k.GetTool(ctx, depToolID)
		if !ok {
			return false
		}

		// Mutual dependency cancellation: if this tool also depends on the peer, skip.
		if _, mutual := k.GetDependencyEdge(ctx, toolID, depToolID); mutual {
			return false
		}

		// Apply dampening per hop.
		hopDampening := safeMulDiv(dampening, PeerDampeningBps, maxScore)

		score := safeMulDiv(depTool.TrustScore, hopDampening, maxScore)

		// Same-deployer penalty: 50% reduction.
		if depTool.Deployer == originalDeployer {
			score = safeMulDiv(score, SameAuthorPenaltyBps, maxScore)
		}

		*peers = append(*peers, score)

		// Recurse deeper.
		k.collectPeerScores(ctx, depToolID, originalDeployer, depth+1, hopDampening, visited, peers)
		return false
	})
}

// calculateContributorScore: staking-tier weighted contributor quality.
func (k Keeper) calculateContributorScore(ctx context.Context, tool *types.Tool) uint64 {
	acceptedCount := uint64(0)
	var weightedSum uint64
	var totalShareBps uint64

	for _, c := range tool.Contributors {
		if !c.Accepted {
			continue
		}
		acceptedCount++

		// Default to Apprentice tier score if staking keeper unavailable.
		tierScore := TierScoreApprentice
		accuracyScore := maxScore / 2 // 50% default

		if k.stakingKeeper != nil {
			tier, err := k.stakingKeeper.GetValidatorTier(ctx, c.Address)
			if err == nil {
				switch tier {
				case 0:
					tierScore = TierScoreApprentice
				case 1:
					tierScore = TierScoreVerified
				case 2:
					tierScore = TierScoreBonded
				case 3:
					tierScore = TierScoreGuardian
				default:
					if tier > 3 {
						tierScore = TierScoreGuardian
					}
				}
			}
			acc, err := k.stakingKeeper.GetValidatorAccuracy(ctx, c.Address)
			if err == nil {
				accuracyScore = acc
			}
		}

		// Blend: 70% tier + 30% accuracy.
		blended := safeMulDiv(tierScore, 700_000, maxScore) + safeMulDiv(accuracyScore, 300_000, maxScore)

		// Weight by ShareBps.
		weightedSum += safeMulDiv(blended, c.ShareBps, types.BpsDenominator)
		totalShareBps += c.ShareBps
	}

	if acceptedCount == 0 {
		return 0
	}

	// Normalize by actual total share (handles partial acceptance).
	if totalShareBps > 0 && totalShareBps < types.BpsDenominator {
		weightedSum = safeMulDiv(weightedSum, types.BpsDenominator, totalShareBps)
	}

	if weightedSum > maxScore {
		weightedSum = maxScore
	}
	return weightedSum
}

// UpdateVerifiedStatus manages verified badge: promotion, retention, demotion.
func (k Keeper) UpdateVerifiedStatus(ctx context.Context, tool *types.Tool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)
	gracePeriod := params.VerifiedGracePeriodBlocks
	if gracePeriod == 0 {
		gracePeriod = types.DefaultVerifiedGracePeriodBlocks
	}

	if !tool.IsVerified {
		// Promotion: score >= verified min tier.
		if tool.TrustScore >= types.TrustTierVerifiedMin {
			tool.IsVerified = true
			tool.VerifiedSince = blockHeight
			tool.VerifiedDemotionBlock = 0
		}
		return
	}

	// Already verified — check retention.
	if tool.TrustScore >= types.VerifiedMinRetentionScore {
		// Still healthy — clear any pending demotion.
		tool.VerifiedDemotionBlock = 0
		return
	}

	// Below retention threshold — start or check grace period.
	if tool.VerifiedDemotionBlock == 0 {
		tool.VerifiedDemotionBlock = blockHeight
		return
	}

	// Grace period expired → demote.
	if blockHeight >= tool.VerifiedDemotionBlock+gracePeriod {
		tool.IsVerified = false
		tool.VerifiedSince = 0
		tool.VerifiedDemotionBlock = 0
	}
}

// RecalculateTrustScores recalculates trust for all active tools (EndBlocker batch).
func (k Keeper) RecalculateTrustScores(ctx context.Context) {
	k.IterateTools(ctx, func(tool *types.Tool) bool {
		if tool.Status != types.ToolStatusActive && tool.Status != types.ToolStatusTesting {
			return false
		}
		snap := k.ComputeTrustScore(ctx, tool)
		k.SetTrustSnapshot(ctx, snap)

		tool.TrustScore = snap.Score
		k.UpdateVerifiedStatus(ctx, tool)
		k.SetTool(ctx, tool)
		return false
	})
}

// EmitTrustScoreEvent emits a trust score update event.
func (k Keeper) EmitTrustScoreEvent(ctx context.Context, toolID string, oldScore, newScore uint64) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.toolbox.trust_updated",
			sdk.NewAttribute("tool_id", toolID),
			sdk.NewAttribute("old_score", uintToStr(oldScore)),
			sdk.NewAttribute("new_score", uintToStr(newScore)),
			sdk.NewAttribute("tier", types.TrustTierLabel(newScore)),
		),
	)
}

// ---------- helpers ----------

func parseUint64(s string) uint64 {
	v, _ := new(big.Int).SetString(s, 10)
	if v == nil || !v.IsUint64() {
		return 0
	}
	return v.Uint64()
}

func uintToStr(v uint64) string {
	return new(big.Int).SetUint64(v).String()
}

func jsonUnmarshal(bz []byte, v interface{}) error {
	return json.Unmarshal(bz, v)
}
