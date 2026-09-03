package cross_stack_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
)

func TestSDKGovAdmissionAndRouterCannotImpersonateEmergencyGuardian(t *testing.T) {
	h := NewTestHarness(t)
	ctx := h.Ctx
	guardian := sdk.AccAddress(bytes.Repeat([]byte{42}, 20))

	message := &emergencytypes.MsgProposeHalt{
		Proposer: guardian.String(),
		Reason:   "legacy SDK governance proposal must not impersonate a Guardian",
	}
	_, err := h.App.GovKeeper.SubmitProposal(
		ctx,
		[]sdk.Msg{message},
		"",
		"legacy unsafe emergency execution",
		"runtime marker regression",
		guardian,
		false,
	)
	require.Error(t, err)
	require.Contains(
		t,
		err.Error(),
		"expected gov account as only signer",
		"SDK governance must admit only messages signed by its module account",
	)

	// EndBlock executes admitted proposal messages through this same router on
	// an unmarked cached context. A manually imported legacy proposal therefore
	// fails independently of modern admission validation.
	handler := h.App.MsgServiceRouter().Handler(message)
	require.NotNil(t, handler)
	_, err = handler(ctx, message)
	require.Error(t, err)
	require.True(t, emergencytypes.ErrUnauthenticatedEmergencyExecution.Is(err))
	require.Equal(t, emergencytypes.StatusNormal, h.EmergencyKeeper.GetEmergencyStatus(ctx))
	require.Empty(t, h.EmergencyKeeper.GetAllCeremonies(ctx))
}

func TestSDK053IBC10ActivationAuditsTerminalProposalInActiveSDKGovQueue(
	t *testing.T,
) {
	h := NewTestHarness(t)
	govAddress := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, govAddress)
	emergencyAny, err := codectypes.NewAnyWithValue(
		&emergencytypes.MsgVoteResume{
			Voter:      govAddress.String(),
			ProposalId: "stale-queue-unsafe",
			Approve:    true,
		},
	)
	require.NoError(t, err)
	proposalID := uint64(8_003)
	require.NoError(t, h.App.GovKeeper.Proposals.Set(
		h.Ctx,
		proposalID,
		govv1.Proposal{
			Id:       proposalID,
			Messages: []*codectypes.Any{emergencyAny},
			Status:   govv1.StatusPassed,
		},
	))
	require.NoError(t, h.App.GovKeeper.ActiveProposalsQueue.Set(
		h.Ctx,
		collections.Join(time.Unix(1_900_000_000, 0).UTC(), proposalID),
		proposalID,
	))
	h.CommitHMinusOne()

	_, err = runSDK053IBC10HandlerForTests(
		t,
		h,
		sdk053IBC10SourceVM(h),
		testH3ActivationHeight,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "active SDK governance queue proposal")
	require.Contains(t, err.Error(), "incoherent")
}

func TestSDK053IBC10ActivationRejectsPendingAuthzWrappedEmergencyGovProposal(t *testing.T) {
	h := NewTestHarness(t)
	govAddress := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, govAddress)

	exec := authz.NewMsgExec(govAddress, []sdk.Msg{
		&emergencytypes.MsgVoteResume{
			Voter:      govAddress.String(),
			ProposalId: "legacy-unsafe",
			Approve:    true,
		},
	})
	anyExec, err := codectypes.NewAnyWithValue(&exec)
	require.NoError(t, err)
	proposalID := uint64(8_002)
	require.NoError(t, h.App.GovKeeper.Proposals.Set(h.Ctx, proposalID, govv1.Proposal{
		Id:       proposalID,
		Messages: []*codectypes.Any{anyExec},
		Status:   govv1.StatusVotingPeriod,
	}))
	h.CommitHMinusOne()

	_, err = runSDK053IBC10HandlerForTests(
		t,
		h,
		sdk053IBC10SourceVM(h),
		testH3ActivationHeight,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SDK governance authority audit failed")
	require.True(
		t,
		strings.Contains(err.Error(), "contains emergency coordination execution") ||
			strings.Contains(err.Error(), "decode active SDK governance proposal"),
		"wrapped proposal must fail closed: %v",
		err,
	)

	// The read-only audit must not partially execute or mutate emergency state.
	require.Equal(
		t,
		emergencytypes.StatusNormal,
		h.EmergencyKeeper.GetEmergencyStatus(h.Ctx),
	)
}

func TestSDK053IBC10ActivationSkipsUnknownMessagesOnlyInTerminalGovProposals(
	t *testing.T,
) {
	for _, testCase := range []struct {
		name          string
		status        govv1.ProposalStatus
		wantErrorText string
	}{
		{
			name:          "terminal proposal is inert before target refusal",
			status:        govv1.StatusPassed,
			wantErrorText: "requires frozen H3 target VersionMap",
		},
		{
			name:          "active proposal fails closed",
			status:        govv1.StatusVotingPeriod,
			wantErrorText: "SDK governance authority audit failed",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			h := NewTestHarness(t)
			proposalID := uint64(8_100)
			require.NoError(
				t,
				h.App.GovKeeper.Proposals.Set(
					h.Ctx,
					proposalID,
					govv1.Proposal{
						Id: proposalID,
						Messages: []*codectypes.Any{
							{
								TypeUrl: "/removed.ibc.fee.v1.MsgLegacy",
								Value:   []byte{0x08, 0x01},
							},
						},
						Status: testCase.status,
					},
				),
			)
			h.CommitHMinusOne()

			_, err := runSDK053IBC10HandlerForTests(
				t,
				h,
				sdk053IBC10SourceVM(h),
				testH3ActivationHeight,
			)
			require.ErrorContains(t, err, testCase.wantErrorText)
			require.Empty(
				t,
				h.KnowledgeKeeper.ReadMigrationMarker(
					h.Ctx,
					"upgrade_marker_upgrade-incident-operations-v1",
				),
			)
		})
	}
}
