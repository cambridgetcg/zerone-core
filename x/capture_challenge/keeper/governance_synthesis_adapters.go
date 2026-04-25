package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/capture_challenge/types"
	govsynthtypes "github.com/zerone-chain/zerone/x/governance_synthesis/types"
)

// GovernanceSynthesisAdapter wraps the capture_challenge keeper to
// satisfy x/governance_synthesis's CaptureChallengeKeeper interface.
// One method: CountChallengesByStatus, which gives the synthesizer
// a single number per posture (open / upheld_recent) without making
// it iterate native types.
type GovernanceSynthesisAdapter struct {
	k Keeper
}

func NewGovernanceSynthesisAdapter(k Keeper) *GovernanceSynthesisAdapter {
	return &GovernanceSynthesisAdapter{k: k}
}

// CountChallengesByStatus returns counts grouped by lifecycle position.
// Resolved+UPHELD challenges with resolved_block ≥ sinceBlock count as
// "recent"; "open" includes anything not yet resolved (submitted /
// evidence / under-review). Pending and expired are ignored.
func (a *GovernanceSynthesisAdapter) CountChallengesByStatus(ctx context.Context, sinceBlock uint64) govsynthtypes.ChallengeStatusCounts {
	var counts govsynthtypes.ChallengeStatusCounts
	a.k.IterateChallenges(ctx, func(c *types.CaptureChallenge) bool {
		if c == nil {
			return false
		}
		switch c.Status {
		case types.ChallengeStatus_CHALLENGE_STATUS_OPEN,
			types.ChallengeStatus_CHALLENGE_STATUS_EVIDENCE,
			types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW:
			counts.Open++
		case types.ChallengeStatus_CHALLENGE_STATUS_RESOLVED:
			if c.Resolution != nil &&
				c.Resolution.Outcome == types.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD &&
				c.Resolution.ResolvedBlock >= sinceBlock {
				counts.UpheldRecent++
			}
		}
		return false
	})
	return counts
}
