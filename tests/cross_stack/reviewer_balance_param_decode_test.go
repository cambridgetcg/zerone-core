package cross_stack_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkgov "github.com/cosmos/cosmos-sdk/x/gov"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestKnowledgeUpdateParamsCarriesReviewerBalance drives a knowledge
// MsgUpdateParams that sets min_reviewer_balance (Params field 161) through the
// app's TxEncoder and TxDecoder, which run unknownproto.RejectUnknownFields: the
// new field is part of the registered Params descriptor and is not refused as
// unknown.
//
// The methodology map is emptied on purpose. With it non-empty, as in
// DefaultParams, the same decode fails with "failed to retrieve the message of
// type zerone.knowledge.v1.Params.MethodologyNormalizationBpsEntry": the map's
// entry type is not in the gogo registry. That defect predates this field and is
// not repaired here. Until it is, a governance update of knowledge params on a
// chain whose params carry the map cannot be decoded, whatever it sets, bare or
// wrapped in a proposal. On a chain started from default genesis, governance can
// set min_reviewer_balance today only with a proposal that also empties the map,
// and MsgUpdateParams replaces the whole Params, so that erases methodology
// normalization (TestKnowledgeGovProposalSetsReviewerBalance).
func TestKnowledgeUpdateParamsCarriesReviewerBalance(t *testing.T) {
	h := NewTestHarness(t)
	txConfig := h.App.TxConfig()

	p := knowledgetypes.DefaultParams()
	p.MinReviewerBalance = "150000000"
	p.MethodologyNormalizationBps = nil
	builder := txConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(&knowledgetypes.MsgUpdateParams{
		Authority: "zerone1updateparamsauthority00000000001",
		Params:    &p,
	}))
	bz, err := txConfig.TxEncoder()(builder.GetTx())
	require.NoError(t, err)

	decoded, err := txConfig.TxDecoder()(bz)
	require.NoError(t, err)
	msgs := decoded.GetMsgs()
	require.Len(t, msgs, 1)
	update, ok := msgs[0].(*knowledgetypes.MsgUpdateParams)
	require.True(t, ok)
	require.Equal(t, "150000000", update.Params.MinReviewerBalance)
}

// decodeGovProposalEnvelope wraps msg in a govv1.MsgSubmitProposal, the
// envelope a governance tx carries, sends it through the app's TxEncoder and
// TxDecoder, and returns the proposal's single message as decoded.
func decodeGovProposalEnvelope(t *testing.T, h *TestHarness, msg sdk.Msg) sdk.Msg {
	t.Helper()
	txConfig := h.App.TxConfig()
	proposer := sdk.AccAddress(bytes.Repeat([]byte{0x5c}, 20)).String()
	envelope, err := govv1.NewMsgSubmitProposal([]sdk.Msg{msg}, sdk.NewCoins(), proposer, "", "Reviewer minimum balance", "set min_reviewer_balance", false)
	require.NoError(t, err)
	builder := txConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(envelope))
	bz, err := txConfig.TxEncoder()(builder.GetTx())
	require.NoError(t, err)

	decoded, err := txConfig.TxDecoder()(bz)
	require.NoError(t, err)
	require.Len(t, decoded.GetMsgs(), 1)
	submit, ok := decoded.GetMsgs()[0].(*govv1.MsgSubmitProposal)
	require.True(t, ok)
	inner, err := submit.GetMsgs()
	require.NoError(t, err)
	require.Len(t, inner, 1)
	return inner[0]
}

// A real x/gov proposal sets min_reviewer_balance: the update is wrapped in a
// MsgSubmitProposal, decoded by the app's TxDecoder, submitted, deposited,
// voted and executed by the SDK gov EndBlocker against the app's knowledge
// keeper. The harness starts from native default genesis, so the
// reviewer-balance marker is on and the params carry the methodology map.
//
// With the marker on, a proposal that raises the figure to 150 ZRN passes and
// the figure is stored; the knowledge keeper tests show that SubmitCommitment
// then applies it. The proposal has to empty the methodology map to be decoded
// at all (see TestKnowledgeUpdateParamsCarriesReviewerBalance), and the passed
// update stores that emptied map too. A proposal that lowers the figure one
// uzrn below 100 ZRN is accepted and stored, then fails at execution, marker on
// or off: the balance is checked, not locked, and a lower figure waits for a
// locked seat deposit (the reviewer-entry design draft, pull request #190).
// With the marker off a raised figure fails at execution too. A failed
// proposal changes no params.
func TestKnowledgeGovProposalSetsReviewerBalance(t *testing.T) {
	const floorRefusal = "min_reviewer_balance 99999999 is below the built-in minimum of 100000000 uzrn: a lower figure needs a locked seat deposit, which this source does not have"
	for _, tc := range []struct {
		name     string
		markerOn bool
		figure   string
		failure  string // empty: the proposal passes
	}{
		{"marker-on/raised", true, "150000000", ""},
		{"marker-on/one-below", true, "99999999", floorRefusal},
		{"marker-off/raised", false, "150000000", "min_reviewer_balance requires the reviewer-balance-param activation"},
		{"marker-off/one-below", false, "99999999", floorRefusal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewTestHarness(t)
			if !tc.markerOn {
				h.Ctx.KVStore(h.App.GetStoreKeyForTests(knowledgetypes.StoreKey)).Delete(knowledgetypes.ReviewerBalanceParamEnabledKey)
			}
			enabled, err := h.App.KnowledgeKeeper.ReviewerBalanceParamEnabled(h.Ctx)
			require.NoError(t, err)
			require.Equal(t, tc.markerOn, enabled)

			before, err := h.App.KnowledgeKeeper.GetParams(h.Ctx)
			require.NoError(t, err)
			require.Empty(t, before.MinReviewerBalance)
			require.NotEmpty(t, before.MethodologyNormalizationBps, "default genesis carries the methodology map")

			next := proto.Clone(before).(*knowledgetypes.Params)
			next.MinReviewerBalance = tc.figure
			next.MethodologyNormalizationBps = nil
			gov := h.App.AccountKeeper.GetModuleAddress("gov")
			require.NotNil(t, gov)
			inner := decodeGovProposalEnvelope(t, h, &knowledgetypes.MsgUpdateParams{Authority: gov.String(), Params: next})

			proposal := submitVotingSDKGovProposal(t, h, inner, false)
			advanceToSDKGovVotingDeadline(t, h, proposal)
			require.NoError(t, sdkgov.EndBlocker(h.Ctx, h.App.GovKeeper))
			proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
			require.NoError(t, err)

			after, err := h.App.KnowledgeKeeper.GetParams(h.Ctx)
			require.NoError(t, err)
			if tc.failure == "" {
				require.Equal(t, govv1.StatusPassed, proposal.Status, proposal.FailedReason)
				require.Equal(t, tc.figure, after.MinReviewerBalance)
				require.Empty(t, after.MethodologyNormalizationBps, "the update replaced the whole Params, emptied map included")
				return
			}
			require.Equal(t, govv1.StatusFailed, proposal.Status)
			require.Contains(t, proposal.FailedReason, tc.failure)
			require.True(t, proto.Equal(before, after), "a failed proposal changes no params")
		})
	}
}
