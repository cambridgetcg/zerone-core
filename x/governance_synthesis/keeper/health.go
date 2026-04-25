package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/governance_synthesis/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// Sliding-window sizes for "recent" counters. Chosen small enough that
// a fresh burst shows up but large enough that one transaction doesn't
// dominate the signal. Tuned against ~2.5s blocks.
const (
	privilegedActionRecentWindow uint64 = 1024  // ~43 minutes
	cartelResolvedRecentWindow   uint64 = 4096  // ~2.8 hours
)

// BuildHealth synthesises the chain's accumulated stress state at the
// current block. Reads from each upstream keeper, composes a
// SystemHealth message. Stateless, deterministic, current.
//
// Stress level rubric (deterministic):
//
//	CRITICAL: any P0 incident open, OR any cartel UPHELD recent, OR any module paused
//	ELEVATED: any P1 incident open, OR pending fact injection > 0, OR
//	          privileged_actions_recent > 5, OR open cartel allegations > 0
//	NORMAL:   none of the above
//
// The composite is a current statement; re-querying gives fresh state.
func (k Keeper) BuildHealth(ctx context.Context) *types.SystemHealth {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	out := &types.SystemHealth{
		ComputedAtBlock: height,
	}

	// ── Incidents ───────────────────────────────────────────────────
	if k.knowledgeKeeper != nil {
		k.knowledgeKeeper.IterateOpenIncidents(ctx, func(r *knowledgetypes.IncidentRecord) bool {
			if r == nil {
				return false
			}
			out.OpenIncidents++
			switch r.Severity {
			case knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P0:
				out.P0Open++
			case knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P1:
				out.P1Open++
			}
			return false
		})

		// ── Pauses ──────────────────────────────────────────────────
		k.knowledgeKeeper.IteratePausedModules(ctx, func(p *knowledgetypes.ModulePause) bool {
			if p == nil {
				return false
			}
			// Only count pauses still in effect (auto-unpause not yet reached).
			if p.AutoUnpauseAtBlock == 0 || p.AutoUnpauseAtBlock > height {
				out.PausedModules++
			}
			return false
		})

		// ── Pending fact injections ─────────────────────────────────
		k.knowledgeKeeper.IterateAllPendingFactInjections(ctx, func(p *knowledgetypes.PendingFactInjection) bool {
			if p == nil {
				return false
			}
			out.PendingFactInjections++
			return false
		})

		// ── Privileged-action recency ───────────────────────────────
		var sinceBlock uint64
		if height > privilegedActionRecentWindow {
			sinceBlock = height - privilegedActionRecentWindow
		}
		k.knowledgeKeeper.IteratePrivilegedActions(ctx, func(a *knowledgetypes.PrivilegedAction) bool {
			if a == nil {
				return false
			}
			if a.InvokedAtBlock >= sinceBlock {
				out.PrivilegedActionsRecent++
			}
			return false
		})
	}

	// ── Cartel posture ──────────────────────────────────────────────
	if k.captureChallengeKeeper != nil {
		var sinceBlock uint64
		if height > cartelResolvedRecentWindow {
			sinceBlock = height - cartelResolvedRecentWindow
		}
		counts := k.captureChallengeKeeper.CountChallengesByStatus(ctx, sinceBlock)
		out.OpenCartelChallenges = counts.Open
		out.ResolvedCartelRecent = counts.UpheldRecent
	}

	// ── Alignment pacing ────────────────────────────────────────────
	if k.alignmentKeeper != nil {
		creation, analysis := k.alignmentKeeper.GetGlobalPacingMultiplier(ctx)
		out.CreationPacingBps = creation
		out.AnalysisPacingBps = analysis
	}

	// ── Composite ───────────────────────────────────────────────────
	out.StressLevel, out.Explanation = computeStressLevel(out)
	return out
}

func computeStressLevel(h *types.SystemHealth) (string, string) {
	// CRITICAL — any of:
	if h.P0Open > 0 {
		return "CRITICAL", fmt.Sprintf("%d P0 incident(s) open", h.P0Open)
	}
	if h.ResolvedCartelRecent > 0 {
		return "CRITICAL", fmt.Sprintf("%d cartel UPHELD in recent window — verifier panel was demonstrably compromised", h.ResolvedCartelRecent)
	}
	if h.PausedModules > 0 {
		return "CRITICAL", fmt.Sprintf("%d module(s) circuit-breaker-paused", h.PausedModules)
	}

	// ELEVATED — any of:
	if h.P1Open > 0 {
		return "ELEVATED", fmt.Sprintf("%d P1 incident(s) open", h.P1Open)
	}
	if h.PendingFactInjections > 0 {
		return "ELEVATED", fmt.Sprintf("%d authority fact injection(s) awaiting guardian veto", h.PendingFactInjections)
	}
	if h.PrivilegedActionsRecent > 5 {
		return "ELEVATED", fmt.Sprintf("%d privileged actions in last ~43 minutes — burst of authority activity", h.PrivilegedActionsRecent)
	}
	if h.OpenCartelChallenges > 0 {
		return "ELEVATED", fmt.Sprintf("%d cartel allegation(s) in evidence/review phase", h.OpenCartelChallenges)
	}

	return "NORMAL", "no critical or elevated stress signals; chain operating in expected ranges"
}
