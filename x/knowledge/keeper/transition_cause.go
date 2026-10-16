package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Exact causes for challenge-driven status changes (knowledge v12, plan §5.2).
// Each is recorded with the challenge claim's ID as cause_id. Status writes
// that no caller tags keep the inferred cause of inferStatusTransitionCause.
const (
	causeChallengeOpened       = "challenge_opened"
	causeChallengeSurvived     = "challenge_survived"
	causeChallengeInconclusive = "challenge_inconclusive"
	causeChallengeDisproven    = "challenge_disproven"
)

// transitionCause is the exact cause of the next status change of one fact.
type transitionCause struct {
	factID, eventType, causeID string
}

type transitionCauseKey struct{}

// withTransitionCause returns ctx carrying a cause for factID. The value lives
// in the Go context, so CacheContext copies keep it and nothing is stored.
func withTransitionCause(ctx context.Context, factID, eventType, causeID string) sdk.Context {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.WithContext(context.WithValue(sdkCtx.Context(), transitionCauseKey{}, transitionCause{factID: factID, eventType: eventType, causeID: causeID}))
}

// transitionCauseFor returns the attached cause only when it names factID, so
// a cause can never leak into another fact's history.
func transitionCauseFor(ctx context.Context, factID string) (transitionCause, bool) {
	c, ok := ctx.Value(transitionCauseKey{}).(transitionCause)
	if !ok || factID == "" || c.factID != factID {
		return transitionCause{}, false
	}
	return c, true
}

// claimIntegrityTransitionCause attaches an exact cause for factID's next
// status write when claim integrity is on. With the marker off it returns ctx
// unchanged, so the knowledge-11 inferred cause is recorded byte for byte.
func (k Keeper) claimIntegrityTransitionCause(ctx context.Context, factID, eventType, causeID string) (context.Context, error) {
	enabled, err := k.ClaimIntegrityEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return ctx, nil
	}
	return withTransitionCause(ctx, factID, eventType, causeID), nil
}
