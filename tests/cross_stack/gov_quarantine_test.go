package cross_stack_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/collections"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	zeroneapp "github.com/zerone-chain/zerone/app"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
)

func submitVotingSDKGovProposal(
	t *testing.T,
	h *TestHarness,
	message sdk.Msg,
	expedited bool,
) govv1.Proposal {
	t.Helper()

	voter := sdkGovLifecycleVoter(t, h)
	params, err := h.App.GovKeeper.Params.Get(h.Ctx)
	if err != nil {
		params = govv1.DefaultParams()
	}
	if len(params.MinDeposit) == 0 || len(params.ExpeditedMinDeposit) == 0 {
		params.MinDeposit = sdk.NewCoins(
			sdk.NewInt64Coin(zeroneapp.BondDenom, 1),
		)
		params.ExpeditedMinDeposit = sdk.NewCoins(
			sdk.NewInt64Coin(zeroneapp.BondDenom, 2),
		)
	}
	require.NoError(t, h.App.GovKeeper.Params.Set(h.Ctx, params))

	proposal, err := h.App.GovKeeper.SubmitProposal(
		h.Ctx,
		[]sdk.Msg{message},
		"ipfs://zerone-quarantine-regression",
		"Quarantine regression",
		"must not execute outside the exact emergency recovery lane",
		voter,
		expedited,
	)
	require.NoError(t, err)

	deposit := sdk.NewCoins(params.MinDeposit...)
	if expedited {
		deposit = sdk.NewCoins(params.ExpeditedMinDeposit...)
	}
	require.False(t, deposit.Empty())
	require.NoError(t, h.FundAccount(voter, deposit))
	activated, err := h.App.GovKeeper.AddDeposit(
		h.Ctx,
		proposal.Id,
		voter,
		deposit,
	)
	require.NoError(t, err)
	require.True(t, activated)
	require.NoError(t, h.App.GovKeeper.AddVote(
		h.Ctx,
		proposal.Id,
		voter,
		govv1.NewNonSplitVoteOption(govv1.OptionYes),
		"quarantine regression",
	))

	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusVotingPeriod, proposal.Status)
	require.NotNil(t, proposal.VotingEndTime)
	return proposal
}

func advanceToSDKGovVotingDeadline(
	t *testing.T,
	h *TestHarness,
	proposal govv1.Proposal,
) {
	t.Helper()
	require.NotNil(t, proposal.VotingEndTime)
	endTime := proposal.VotingEndTime.Add(time.Nanosecond)
	h.currentHeight++
	h.Ctx = h.Ctx.
		WithBlockHeight(h.currentHeight).
		WithBlockTime(endTime).
		WithBlockHeader(cmtproto.Header{
			Height:  h.currentHeight,
			ChainID: testChainID,
			Time:    endTime,
		})
}

func seedValidatedCancelRecoveryAuthorization(
	t *testing.T,
	h *TestHarness,
	proposal govv1.Proposal,
) {
	t.Helper()
	require.Len(t, proposal.Messages, 1)
	plan, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	haltID := "validated-recovery-fixture-halt"
	authorization := &emergencytypes.EmergencyRecoveryAuthorization{
		HaltCeremonyId:          haltID,
		AuthorizationCeremonyId: "validated-recovery-fixture-authorization",
		SdkGovProposalId:        proposal.Id,
		ActionSha256: zeroneapp.RecoveryActionSHA256(
			proposal.Messages[0],
		),
		RecoveryManifestSha256: strings.Repeat("a", 64),
		AuthorizedAtBlock:      uint64(h.Ctx.BlockHeight()),
		UpgradePlanSha256:      zeroneapp.UpgradePlanSHA256(plan),
		AuthorizedSubmitter:    proposal.Proposer,
		ActionType:             "cancel_upgrade",
		Generation:             1,
	}
	raw, err := proto.Marshal(authorization)
	require.NoError(t, err)
	h.Ctx.KVStore(
		h.App.GetStoreKeyForTests(emergencytypes.StoreKey),
	).Set(emergencytypes.RecoveryAuthorizationKey, raw)
	h.EmergencyKeeper.SetActiveHaltCeremonyId(h.Ctx, haltID)
	h.EmergencyKeeper.SetEmergencyStatus(
		h.Ctx,
		emergencytypes.StatusHalted,
	)
	persisted, found, err :=
		h.EmergencyKeeper.GetRecoveryAuthorization(h.Ctx)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, proto.Equal(authorization, persisted))
}

func TestQuarantineEndBlockPermanentlyFailsSameBlockOrdinaryGovExecution(
	t *testing.T,
) {
	h := NewTestHarness(t)
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	plan := upgradetypes.Plan{
		Name:   "must-survive-hostile-cancel",
		Height: h.Height() + 100,
	}
	require.NoError(t, h.App.UpgradeKeeper.ScheduleUpgrade(h.Ctx, plan))

	proposal := submitVotingSDKGovProposal(
		t,
		h,
		&upgradetypes.MsgCancelUpgrade{Authority: authority.String()},
		false,
	)
	advanceToSDKGovVotingDeadline(t, h, proposal)

	// This models a halt ceremony finalizing in DeliverTx immediately before
	// the SDK governance EndBlock deadline in the same block.
	h.EmergencyKeeper.SetEmergencyStatus(h.Ctx, emergencytypes.StatusHalted)
	endBlock, err := h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)

	scheduled, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, plan, scheduled, "ordinary proposal must not execute")
	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusFailed, proposal.Status)
	require.Contains(t, proposal.FailedReason, "application transaction quarantine")
	require.Len(t, proposal.Messages, 1, "decoded proposal audit record was erased")
	require.Equal(
		t,
		"/cosmos.upgrade.v1beta1.MsgCancelUpgrade",
		proposal.Messages[0].TypeUrl,
	)

	foundAggregate := false
	for _, event := range endBlock.Events {
		if event.Type != "zerone.gov.proposals_quarantined" {
			continue
		}
		foundAggregate = true
		attributes := make(map[string]string, len(event.Attributes))
		for _, attribute := range event.Attributes {
			attributes[attribute.Key] = attribute.Value
		}
		require.Equal(t, "1", attributes["failed_count"])
		require.Equal(
			t,
			"expedited_single_upgrade_or_cancel",
			attributes["allowed_lane"],
		)
		require.Len(t, attributes["queue_manifest_sha256"], 64)
	}
	require.True(t, foundAggregate)
}

func TestResumeFinalizationBlockStillQuarantinesOrdinaryGovExecution(
	t *testing.T,
) {
	h := NewTestHarness(t)
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	plan := upgradetypes.Plan{
		Name:   "must-survive-resume-block",
		Height: h.Height() + 100,
	}
	require.NoError(t, h.App.UpgradeKeeper.ScheduleUpgrade(h.Ctx, plan))

	proposal := submitVotingSDKGovProposal(
		t,
		h,
		&upgradetypes.MsgCancelUpgrade{Authority: authority.String()},
		false,
	)
	advanceToSDKGovVotingDeadline(t, h, proposal)

	// Resume finalization writes status=normal in DeliverTx, but the consensus
	// release latch keeps every ordinary tail and due EndBlock action
	// quarantined through the rest of H.
	h.Ctx = h.Ctx.WithIsCheckTx(false)
	h.EmergencyKeeper.SetEmergencyStatus(h.Ctx, emergencytypes.StatusNormal)
	h.EmergencyKeeper.SetQuarantineReleaseBlock(
		h.Ctx,
		uint64(h.Ctx.BlockHeight()),
	)
	_, err := h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)

	scheduled, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, plan, scheduled)
	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusFailed, proposal.Status)
}

func TestRecoveryProposalFailsClosedOnMismatchedOrDuplicateQueueRecord(
	t *testing.T,
) {
	h := NewTestHarness(t)
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	require.NoError(t, h.App.UpgradeKeeper.ScheduleUpgrade(
		h.Ctx,
		upgradetypes.Plan{
			Name:   "must-not-be-cancelled-by-incoherent-queue",
			Height: h.Height() + 100,
		},
	))
	proposal := submitVotingSDKGovProposal(
		t,
		h,
		&upgradetypes.MsgCancelUpgrade{Authority: authority.String()},
		true,
	)
	require.NotNil(t, proposal.VotingEndTime)
	mismatchedDeadline := proposal.VotingEndTime.Add(-time.Nanosecond)
	require.NoError(t, h.App.GovKeeper.ActiveProposalsQueue.Set(
		h.Ctx,
		collections.Join(mismatchedDeadline, proposal.Id),
		proposal.Id,
	))
	h.currentHeight++
	h.Ctx = h.Ctx.
		WithBlockHeight(h.currentHeight).
		WithBlockTime(proposal.VotingEndTime.Add(time.Nanosecond)).
		WithBlockHeader(cmtproto.Header{
			Height:  h.currentHeight,
			ChainID: testChainID,
			Time:    proposal.VotingEndTime.Add(time.Nanosecond),
		})
	h.EmergencyKeeper.SetEmergencyStatus(h.Ctx, emergencytypes.StatusHalted)

	_, err := h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)
	_, err = h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err, "incoherent queue must not execute recovery action")
	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusFailed, proposal.Status)
	require.Len(t, proposal.Messages, 1)
}

func TestRecoveryProposalRemovesLaterAndFutureDuplicateQueueRecords(
	t *testing.T,
) {
	for _, test := range []struct {
		name          string
		duplicateBy   time.Duration
		endBlockAfter time.Duration
	}{
		{
			name:          "later duplicate already due",
			duplicateBy:   time.Second,
			endBlockAfter: 2 * time.Second,
		},
		{
			name:          "future duplicate",
			duplicateBy:   time.Hour,
			endBlockAfter: time.Nanosecond,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := NewTestHarness(t)
			authority := h.App.AccountKeeper.GetModuleAddress("gov")
			require.NotNil(t, authority)
			plan := upgradetypes.Plan{
				Name:   "duplicate-record-must-not-cancel",
				Height: h.Height() + 100,
			}
			require.NoError(t, h.App.UpgradeKeeper.ScheduleUpgrade(h.Ctx, plan))
			proposal := submitVotingSDKGovProposal(
				t,
				h,
				&upgradetypes.MsgCancelUpgrade{Authority: authority.String()},
				true,
			)
			require.NotNil(t, proposal.VotingEndTime)
			duplicateDeadline := proposal.VotingEndTime.Add(test.duplicateBy)
			duplicateKey := collections.Join(duplicateDeadline, proposal.Id)
			require.NoError(t, h.App.GovKeeper.ActiveProposalsQueue.Set(
				h.Ctx,
				duplicateKey,
				proposal.Id,
			))

			endTime := proposal.VotingEndTime.Add(test.endBlockAfter)
			h.currentHeight++
			h.Ctx = h.Ctx.
				WithBlockHeight(h.currentHeight).
				WithBlockTime(endTime).
				WithBlockHeader(cmtproto.Header{
					Height:  h.currentHeight,
					ChainID: testChainID,
					Time:    endTime,
				})
			h.EmergencyKeeper.SetEmergencyStatus(
				h.Ctx,
				emergencytypes.StatusHalted,
			)

			_, err := h.App.EndBlocker(h.Ctx)
			require.NoError(t, err)
			scheduled, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
			require.NoError(t, err)
			require.Equal(t, plan, scheduled)
			proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
			require.NoError(t, err)
			require.Equal(t, govv1.StatusFailed, proposal.Status)

			canonicalPresent, err := h.App.GovKeeper.ActiveProposalsQueue.Has(
				h.Ctx,
				collections.Join(*proposal.VotingEndTime, proposal.Id),
			)
			require.NoError(t, err)
			require.False(t, canonicalPresent)
			duplicatePresent, err := h.App.GovKeeper.ActiveProposalsQueue.Has(
				h.Ctx,
				duplicateKey,
			)
			require.NoError(t, err)
			if duplicateDeadline.After(endTime) {
				require.True(
					t,
					duplicatePresent,
					"future queue entries are bounded to their canonical due range",
				)
				h.currentHeight++
				dueTime := duplicateDeadline.Add(time.Nanosecond)
				h.Ctx = h.Ctx.
					WithBlockHeight(h.currentHeight).
					WithBlockTime(dueTime).
					WithBlockHeader(cmtproto.Header{
						Height:  h.currentHeight,
						ChainID: testChainID,
						Time:    dueTime,
					})
				_, err = h.App.EndBlocker(h.Ctx)
				require.NoError(t, err)
				duplicatePresent, err =
					h.App.GovKeeper.ActiveProposalsQueue.Has(
						h.Ctx,
						duplicateKey,
					)
				require.NoError(t, err)
			}
			require.False(t, duplicatePresent)
		})
	}
}

func TestPostResumeReviewHoldBlocksFutureOrdinarySDKGovDeadline(
	t *testing.T,
) {
	h := NewTestHarness(t)
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	plan := upgradetypes.Plan{
		Name:   "must-survive-post-resume-review",
		Height: h.Height() + 100,
	}
	require.NoError(t, h.App.UpgradeKeeper.ScheduleUpgrade(h.Ctx, plan))
	proposal := submitVotingSDKGovProposal(
		t,
		h,
		&upgradetypes.MsgCancelUpgrade{Authority: authority.String()},
		false,
	)
	_, created, err := h.GovKeeper.EnsureEmergencyTransitionHold(
		h.Ctx,
		"halt-review-hold-regression",
	)
	require.NoError(t, err)
	require.True(t, created)
	advanceToSDKGovVotingDeadline(t, h, proposal)
	h.EmergencyKeeper.SetEmergencyStatus(h.Ctx, emergencytypes.StatusNormal)

	_, err = h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)
	scheduled, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, plan, scheduled)
	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusFailed, proposal.Status)
}

func TestQuarantineEndBlockStillExecutesExactExpeditedRecoveryAction(
	t *testing.T,
) {
	h := NewTestHarness(t)
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	require.NoError(t, h.App.UpgradeKeeper.ScheduleUpgrade(
		h.Ctx,
		upgradetypes.Plan{
			Name:   "cancel-me-through-recovery-lane",
			Height: h.Height() + 100,
		},
	))
	nextProposalID, err := h.App.GovKeeper.ProposalID.Peek(h.Ctx)
	require.NoError(t, err)
	if nextProposalID == 0 {
		require.NoError(t, h.App.GovKeeper.ProposalID.Set(h.Ctx, 1))
	}

	proposal := submitVotingSDKGovProposal(
		t,
		h,
		&upgradetypes.MsgCancelUpgrade{Authority: authority.String()},
		true,
	)
	seedValidatedCancelRecoveryAuthorization(t, h, proposal)
	advanceToSDKGovVotingDeadline(t, h, proposal)

	_, err = h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)
	_, err = h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.True(t, errors.Is(err, upgradetypes.ErrNoUpgradePlanFound))
	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusPassed, proposal.Status, proposal.FailedReason)
}

func TestRevokedRecoveryProposalCannotExecuteAfterResume(t *testing.T) {
	h := NewTestHarness(t)
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	plan := upgradetypes.Plan{
		Name:   "must-survive-revoked-recovery",
		Height: h.Height() + 100,
	}
	require.NoError(t, h.App.UpgradeKeeper.ScheduleUpgrade(h.Ctx, plan))
	nextProposalID, err := h.App.GovKeeper.ProposalID.Peek(h.Ctx)
	require.NoError(t, err)
	if nextProposalID == 0 {
		require.NoError(t, h.App.GovKeeper.ProposalID.Set(h.Ctx, 1))
	}

	proposal := submitVotingSDKGovProposal(
		t,
		h,
		&upgradetypes.MsgCancelUpgrade{Authority: authority.String()},
		true,
	)
	seedValidatedCancelRecoveryAuthorization(t, h, proposal)
	actionHash := zeroneapp.RecoveryActionSHA256(proposal.Messages[0])
	require.NoError(t, h.EmergencyKeeper.MarkRecoveryAuthorizationTerminal(
		h.Ctx,
		proposal.Id,
		actionHash,
		"revoked",
	))

	queueKey := collections.Join(*proposal.VotingEndTime, proposal.Id)
	queued, err := h.App.GovKeeper.ActiveProposalsQueue.Has(
		h.Ctx,
		queueKey,
	)
	require.NoError(t, err)
	require.True(t, queued)
	deposits, err := h.App.GovKeeper.GetDeposits(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.NotEmpty(t, deposits)

	// The proposal is still far from its voting deadline. Revocation must
	// nevertheless close the exact linked proposal now; the bounded due-range
	// quarantine scan cannot discover it.
	_, err = h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)

	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusFailed, proposal.Status)
	require.Contains(
		t,
		proposal.FailedReason,
		"emergency recovery authorization was revoked",
	)
	require.Len(t, proposal.Messages, 1)
	queued, err = h.App.GovKeeper.ActiveProposalsQueue.Has(h.Ctx, queueKey)
	require.NoError(t, err)
	require.False(t, queued)
	deposits, err = h.App.GovKeeper.GetDeposits(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Empty(t, deposits)
	authorization, found, err :=
		h.EmergencyKeeper.GetRecoveryAuthorization(h.Ctx)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "revoked", authorization.Outcome)
	scheduled, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, plan, scheduled)

	// Resume transaction admission and advance beyond the original deadline.
	// The yes-voted cancellation must remain terminal and cannot execute.
	h.EmergencyKeeper.SetEmergencyStatus(h.Ctx, emergencytypes.StatusNormal)
	h.EmergencyKeeper.SetActiveHaltCeremonyId(h.Ctx, "")
	h.EmergencyKeeper.SetQuarantineReleaseBlock(
		h.Ctx,
		uint64(h.Ctx.BlockHeight()),
	)
	advanceToSDKGovVotingDeadline(t, h, proposal)
	_, err = h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)

	scheduled, err = h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, plan, scheduled)
	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusFailed, proposal.Status)
	require.Contains(
		t,
		proposal.FailedReason,
		"emergency recovery authorization was revoked",
	)
}

func TestQuarantineEndBlockPreservesNonDueThenFailsDueDepositProposal(
	t *testing.T,
) {
	h := NewTestHarness(t)
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	params := govv1.DefaultParams()
	params.MinDeposit = sdk.NewCoins(
		sdk.NewInt64Coin(zeroneapp.BondDenom, 1),
	)
	params.ExpeditedMinDeposit = sdk.NewCoins(
		sdk.NewInt64Coin(zeroneapp.BondDenom, 2),
	)
	require.NoError(t, h.App.GovKeeper.Params.Set(h.Ctx, params))
	proposal, err := h.App.GovKeeper.SubmitProposal(
		h.Ctx,
		[]sdk.Msg{&upgradetypes.MsgCancelUpgrade{
			Authority: authority.String(),
		}},
		"",
		"Ordinary inactive proposal",
		"must not survive quarantine",
		sdk.AccAddress(strings.Repeat("a", 20)),
		false,
	)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusDepositPeriod, proposal.Status)

	h.EmergencyKeeper.SetEmergencyStatus(h.Ctx, emergencytypes.StatusHalted)
	_, err = h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)
	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusDepositPeriod, proposal.Status)
	require.NotNil(t, proposal.DepositEndTime)
	queued, err := h.App.GovKeeper.InactiveProposalsQueue.Has(
		h.Ctx,
		collections.Join(*proposal.DepositEndTime, proposal.Id),
	)
	require.NoError(t, err)
	require.True(t, queued)

	dueTime := proposal.DepositEndTime.Add(time.Nanosecond)
	h.currentHeight++
	h.Ctx = h.Ctx.
		WithBlockHeight(h.currentHeight).
		WithBlockTime(dueTime).
		WithBlockHeader(cmtproto.Header{
			Height:  h.currentHeight,
			ChainID: testChainID,
			Time:    dueTime,
		})
	_, err = h.App.EndBlocker(h.Ctx)
	require.NoError(t, err)
	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusFailed, proposal.Status)
}
