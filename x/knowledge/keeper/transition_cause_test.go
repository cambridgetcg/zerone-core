package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// A transition cause names one fact. setFactWithHistory uses it only for that
// fact; any other fact written under the same context keeps the inferred cause.
func TestTransitionCauseIsScopedToItsFact(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "other", ClaimId: "other-claim", Status: types.FactStatus_FACT_STATUS_VERIFIED}))

	ctx := withTransitionCause(f.ctx, "fact", causeChallengeOpened, "challenge-1")
	cached, _ := sdk.UnwrapSDKContext(ctx).CacheContext()
	cause, ok := transitionCauseFor(cached, "fact")
	require.True(t, ok, "CacheContext keeps the cause")
	require.Equal(t, transitionCause{factID: "fact", eventType: causeChallengeOpened, causeID: "challenge-1"}, cause)
	_, ok = transitionCauseFor(cached, "other")
	require.False(t, ok)
	_, ok = transitionCauseFor(f.ctx, "fact")
	require.False(t, ok, "the parent context is unchanged")

	last := func(id string) *types.StatusTransition {
		history, err := f.k.GetStatusHistoryChecked(f.ctx, id)
		require.NoError(t, err)
		require.NotEmpty(t, history)
		return history[len(history)-1]
	}

	other, found := f.k.GetFact(f.ctx, "other")
	require.True(t, found)
	other.Status = types.FactStatus_FACT_STATUS_CHALLENGED
	require.NoError(t, f.k.SetFact(ctx, other))
	require.Equal(t, "unknown", last("other").CauseEventType, "a cause for another fact never applies")
	require.Empty(t, last("other").CauseId)

	fact, found := f.k.GetFact(f.ctx, "fact")
	require.True(t, found)
	fact.Status = types.FactStatus_FACT_STATUS_CHALLENGED
	require.NoError(t, f.k.SetFact(ctx, fact))
	require.Equal(t, causeChallengeOpened, last("fact").CauseEventType)
	require.Equal(t, "challenge-1", last("fact").CauseId)

	fact.Status = types.FactStatus_FACT_STATUS_ACTIVE
	require.NoError(t, f.k.SetFact(f.ctx, fact))
	require.Equal(t, "unknown", last("fact").CauseEventType, "without a cause the transition is inferred as before")
	require.Empty(t, last("fact").CauseId)
	require.NoError(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
}
