package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Trunk's accepted design supersedes unanimity at quorum. Q2 implements only
// the two accounting amendments, not PanelDecisionPolicy or a param change.
// The selection comes from BeginBlocker; helpers never read the marker.
func (k Keeper) aggregateInconclusiveAccounting(ctx context.Context, round *types.VerificationRound, enabled bool) (*VerificationResult, error) {
	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil || !enabled {
		return result, err
	}
	if round.ReviewPolicyVersion == types.ReviewPolicyLegacy && result.Verdict == types.Verdict_VERDICT_INCONCLUSIVE {
		params, err := k.GetParams(ctx)
		if err != nil {
			return nil, err
		}
		// Missing quorum and an inconclusive vote split both leave no correct
		// vote. Only a missed reveal is slashable, independently of rewards.
		result.Slashes = missedRevealSlashes(round, params)
	}
	// Policy 1's immutable no-monetary-debt contract remains unchanged.
	return result, nil
}

func missedRevealSlashes(round *types.VerificationRound, params *types.Params) []VerifierSlash {
	revealed := make(map[string]bool, len(round.Reveals))
	for _, reveal := range round.Reveals {
		revealed[reveal.Verifier] = true
	}
	var slashes []VerifierSlash
	for _, commit := range round.Commits {
		if !revealed[commit.Verifier] {
			slashes = append(slashes, VerifierSlash{Verifier: commit.Verifier, SlashBps: params.MissedRevealSlashBps})
		}
	}
	return slashes
}

// inconclusiveCaptureDefense leaves history and capture queries intact, but an
// unadjudicated outcome cannot update reputation. It is installed only on the
// keeper value used to complete this round; it never changes the wired keeper.
// This preserves authority-pinned rounds.go byte for byte.
type inconclusiveCaptureDefense struct{ types.CaptureDefenseKeeper }

func (inconclusiveCaptureDefense) UpdateReputation(context.Context, string, string, string, bool) {}

func (k roundsExecution) performAggregationInconclusiveAccounting(ctx context.Context, round *types.VerificationRound, enabled bool) error {
	if !enabled {
		return k.performAggregation(ctx, round)
	}
	result, err := k.aggregateInconclusiveAccounting(ctx, round, enabled)
	if err != nil {
		return err
	}
	if result.Verdict == types.Verdict_VERDICT_INCONCLUSIVE && k.captureDefenseKeeper != nil {
		k.captureDefenseKeeper = inconclusiveCaptureDefense{k.captureDefenseKeeper}
	}
	return k.CompleteRound(ctx, round, result)
}

// advanceRoundPhasesInconclusiveAccounting is the activated counterpart of
// AdvanceRoundPhases at 6a56b621. Phase selection, pre-existing errors, events
// and expiry side effects are unchanged. Only aggregation accounting and the
// missed-reveal slash on the separate EXPIRED path differ.
func (k roundsExecution) advanceRoundPhasesInconclusiveAccounting(ctx context.Context, enabled bool) error {
	if !enabled {
		return k.AdvanceRoundPhases(ctx)
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	var roundsToProcess []*types.VerificationRound
	k.IterateActiveRounds(ctx, func(round *types.VerificationRound) bool {
		roundsToProcess = append(roundsToProcess, round)
		return false
	})
	for _, round := range roundsToProcess {
		policy, err := k.reviewPolicyForRound(ctx, round)
		if err != nil {
			return err
		}
		expectedPhase := GetExpectedPhase(round, height, params)
		if expectedPhase == round.Phase {
			if policy == types.ReviewPolicyNeutral && round.Phase == types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION {
				if err := k.performAggregationInconclusiveAccounting(ctx, round, enabled); err != nil {
					k.Logger(ctx).Error("review aggregation remains pending", "round_id", round.Id, "error", err)
				}
			}
			continue
		}
		switch expectedPhase {
		case types.VerificationPhase_VERIFICATION_PHASE_REVEAL:
			round.Phase = types.VerificationPhase_VERIFICATION_PHASE_REVEAL
			if err := k.SetVerificationRound(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to transition round to REVEAL", "round_id", round.Id, "error", err)
			}
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.round_phase_changed",
				sdk.NewAttribute("round_id", round.Id),
				sdk.NewAttribute("phase", "REVEAL"),
			))
		case types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION:
			if height < round.RevealDeadline &&
				uint64(len(round.Commits)) < k.effectiveMinVerifiersForRound(ctx, round, params) {
				continue
			}
			round.Phase = types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
			if err := k.SetVerificationRound(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to transition round to AGGREGATION", "round_id", round.Id, "error", err)
				continue
			}
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.round_phase_changed",
				sdk.NewAttribute("round_id", round.Id),
				sdk.NewAttribute("phase", "AGGREGATION"),
				sdk.NewAttribute("reveal_count", fmt.Sprintf("%d", len(round.Reveals))),
			))
			if err := k.performAggregationInconclusiveAccounting(ctx, round, enabled); err != nil {
				k.Logger(ctx).Error("aggregation failed", "round_id", round.Id, "error", err)
			}
		case types.VerificationPhase_VERIFICATION_PHASE_EXPIRED:
			if policy == types.ReviewPolicyNeutral {
				if err := k.performAggregationInconclusiveAccounting(ctx, round, enabled); err != nil {
					k.Logger(ctx).Error("review deadline settlement remains pending", "round_id", round.Id, "error", err)
				}
				continue
			}
			if uint64(len(round.Reveals)) >= params.MinVerifiers {
				round.Phase = types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
				if err := k.SetVerificationRound(ctx, round); err != nil {
					continue
				}
				if err := k.performAggregationInconclusiveAccounting(ctx, round, enabled); err != nil {
					k.Logger(ctx).Error("late aggregation failed", "round_id", round.Id, "error", err)
				}
			} else {
				round.Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED
				round.Verdict = types.Verdict_VERDICT_INCONCLUSIVE
				if err := k.SetVerificationRound(ctx, round); err != nil {
					continue
				}
				// No aggregation occurs on this route. Preserve EXPIRED and its
				// existing side effects, but do not forgive missed commitments.
				// Like CompleteRound, preserve the existing slash error handling.
				if k.stakingKeeper != nil {
					for _, slash := range missedRevealSlashes(round, params) {
						_ = k.stakingKeeper.SlashValidator(ctx, slash.Verifier, slash.SlashBps)
					}
				}
				claim, found := k.GetClaim(ctx, round.ClaimId)
				if found {
					claim.Status = types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT
					_ = k.SetClaim(ctx, claim)
					if claim.ClaimType != types.ClaimType_CLAIM_TYPE_CONJECTURE {
						if err := k.RecordSubmissionOutcome(ctx, claim.Submitter, ResolveMethodId(claim.MethodId), types.Verdict_VERDICT_INCONCLUSIVE); err != nil {
							return err
						}
					}
					integrity, err := k.ClaimIntegrityEnabled(ctx)
					if err == nil {
						err = k.endContradiction(ctx, claim, integrity)
					}
					if err != nil {
						k.Logger(ctx).Error("starved contradiction left its targets unchanged", "claim_id", claim.Id, "error", err)
					}
					if err := k.restoreChallengedFactOnInconclusive(ctx, claim, round.Id); err != nil {
						return err
					}
					emitKarmaEdgeState(sdkCtx, "pending_settle", karmaEdgeStateOrdinal,
						claim.Submitter, "", claim.Id, claim.Domain,
						sdk.NewAttribute("verdict", types.Verdict_VERDICT_INCONCLUSIVE.String()))
				}
				sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
					"zerone.knowledge.round_expired",
					sdk.NewAttribute("round_id", round.Id),
					sdk.NewAttribute("reveals", fmt.Sprintf("%d", len(round.Reveals))),
				))
			}
		}
	}
	return nil
}
