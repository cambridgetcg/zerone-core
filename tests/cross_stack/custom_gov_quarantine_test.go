package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	zeronegovkeeper "github.com/zerone-chain/zerone/x/gov/keeper"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
)

func TestCustomGovernanceAutomaticTransitionsFreezeDuringQuarantine(
	t *testing.T,
) {
	h := NewTestHarness(t)
	lip := &zeronegovtypes.LIP{
		Id:             "LIP-hostile-deadline-during-quarantine",
		Title:          "must remain frozen",
		Description:    "automatic custom governance cannot bypass quarantine",
		Category:       zeronegovtypes.CategoryText,
		Stage:          zeronegovtypes.StatusVoting,
		VotingEndBlock: uint64(h.Height()),
	}
	h.GovKeeper.SetLIP(h.Ctx, lip)
	h.EmergencyKeeper.SetEmergencyStatus(
		h.Ctx,
		emergencytypes.StatusHalted,
	)
	h.EmergencyKeeper.SetActiveHaltCeremonyId(
		h.Ctx,
		"halt-hostile-deadline-during-quarantine",
	)
	h.EmergencyKeeper.SetHaltStartBlock(h.Ctx, uint64(h.Height()))

	beginBlock, err := h.App.BeginBlocker(h.Ctx)
	require.NoError(t, err)
	frozen, found := h.GovKeeper.GetLIP(h.Ctx, lip.Id)
	require.True(t, found)
	require.Equal(t, zeronegovtypes.StatusVoting, frozen.Stage)
	foundFreezeEvent := false
	for _, event := range beginBlock.Events {
		if event.Type == "zerone.gov.custom_transitions_frozen" {
			foundFreezeEvent = true
		}
	}
	require.True(t, foundFreezeEvent)
	hold, found, err := h.GovKeeper.GetEmergencyTransitionHold(h.Ctx)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(
		t,
		"halt-hostile-deadline-during-quarantine",
		hold.IncidentId,
	)
	msgServer := zeronegovkeeper.NewMsgServerImpl(h.GovKeeper)
	_, err = msgServer.VoteResearchSpend(
		h.Ctx,
		&zeronegovtypes.MsgVoteResearchSpend{},
	)
	require.ErrorContains(
		t,
		err,
		"custom governance transitions are held for post-incident review",
	)
	_, err = h.GovKeeper.VoteResearchSpend(
		h.Ctx,
		&zeronegovtypes.MsgVoteResearchSpend{},
	)
	require.ErrorContains(
		t,
		err,
		"custom governance transitions are held for post-incident review",
	)

	// Forward-only withdrawal is the sole safe mutation left open so a
	// proposer can reduce, rather than expand, the frozen backlog.
	withdrawable := &zeronegovtypes.LIP{
		Id:       "LIP-safe-withdrawal-during-review-hold",
		Proposer: "proposer",
		Stage:    zeronegovtypes.StatusReview,
		Category: zeronegovtypes.CategoryText,
	}
	h.GovKeeper.SetLIP(h.Ctx, withdrawable)
	_, err = msgServer.WithdrawLIP(
		h.Ctx,
		&zeronegovtypes.MsgWithdrawLIP{
			LipId:    withdrawable.Id,
			Proposer: withdrawable.Proposer,
		},
	)
	require.NoError(t, err)

	// Resume does not release the custom-governance backlog before operators
	// have had a chance to reconcile it.
	h.EmergencyKeeper.SetEmergencyStatus(
		h.Ctx,
		emergencytypes.StatusNormal,
	)
	h.EmergencyKeeper.SetActiveHaltCeremonyId(h.Ctx, "")
	h.EmergencyKeeper.ClearHaltStartBlock(h.Ctx)
	_, err = h.App.BeginBlocker(h.Ctx)
	require.NoError(t, err)
	stillFrozen, found := h.GovKeeper.GetLIP(h.Ctx, lip.Id)
	require.True(t, found)
	require.Equal(t, zeronegovtypes.StatusVoting, stillFrozen.Stage)

	query, err := zeronegovkeeper.NewQueryServerImpl(
		h.GovKeeper,
	).EmergencyTransitionHold(
		h.Ctx,
		&zeronegovtypes.QueryEmergencyTransitionHoldRequest{},
	)
	require.NoError(t, err)
	require.True(t, query.Held)
	require.Equal(t, hold.IncidentId, query.Hold.IncidentId)
	require.Contains(t, query.ReleaseMechanism, "no_current_release_api")

	// The current binary deliberately has no callable release path. A future
	// named upgrade must add and audit one operation that reconciles every
	// frozen queue before deleting the hold.
	_, err = h.App.BeginBlocker(h.Ctx)
	require.NoError(t, err)
	resolved, found := h.GovKeeper.GetLIP(h.Ctx, lip.Id)
	require.True(t, found)
	require.Equal(t, zeronegovtypes.StatusVoting, resolved.Stage)
}
