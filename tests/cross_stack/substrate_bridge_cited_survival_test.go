package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	substratebridgekeeper "github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// TestSubstrateBridge_CitedRewardCancelledWhenCitedFactDisproven runs the
// cited-fact survival rule through the app's real x/knowledge keeper, on
// native genesis, which selects the cited-reward escrow marker. The cited fact
// is VERIFIED when the attestation settles and its reward is escrowed. It is
// DISPROVEN before the window closes, so the BeginBlocker at the deadline
// cancels the escrow and names the fact: nothing is minted, and the submitter
// ends with exactly the bond it put up.
func TestSubstrateBridge_CitedRewardCancelledWhenCitedFactDisproven(t *testing.T) {
	h := NewTestHarness(t)
	enabled, err := h.SubstrateBridgeKeeper.CitedRewardEscrowEnabled(h.Ctx)
	require.NoError(t, err)
	require.True(t, enabled, "native genesis selects the cited-reward escrow marker")

	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId:              "test-wiki",
		Status:                 substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		MinAttestationBondUzrn: "222000",
		MinPerClaimBondUzrn:    "222",
	}))
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   "test-domain",
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))
	fact := &knowledgetypes.Fact{
		Id:              "survival-fact",
		Domain:          "test-domain",
		Status:          knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		VerifiedAtBlock: 1,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, fact))
	link := &substratebridgetypes.SubstrateLink{
		AdapterId: "test-wiki",
		Source: &substratebridgetypes.ExternalSource{
			AdapterId:   "test-wiki",
			SourceId:    "article-disproven",
			ContentHash: make([]byte, 32),
		},
		CitedFacts: []*substratebridgetypes.FactCitation{
			{FactId: "survival-fact", CitationType: substratebridgetypes.CitationType_CITATION_TYPE_SUPPORTS},
		},
	}
	link.LinkHash = substratebridgekeeper.ComputeLinkHash(link)

	submitter := testAddr("sb_disproven_submitter")
	require.NoError(t, h.FundAccount(submitter, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 10_000_000))))
	preSubmit := h.App.BankKeeper.GetBalance(h.Ctx, submitter, "uzrn")
	srv := substratebridgekeeper.NewMsgServerImpl(h.SubstrateBridgeKeeper)
	resp, err := srv.SubmitExternalAttestation(h.Ctx, &substratebridgetypes.MsgSubmitExternalAttestation{
		Submitter:   submitter.String(),
		AdapterId:   "test-wiki",
		WorkClassId: "translation",
		Link:        link,
		BondUzrn:    "1000000",
	})
	require.NoError(t, err)
	attID := resp.AttestationId

	sdkCtx := sdk.UnwrapSDKContext(h.Ctx)
	preMinted := h.VestingRewardsKeeper.GetTotalMinted(sdkCtx)
	require.NoError(t, h.SubstrateBridgeKeeper.BeginBlocker(h.Ctx))
	_, found := h.SubstrateBridgeKeeper.GetWitnessPendingReward(h.Ctx, attID)
	require.True(t, found, "the cited reward waits in the witness escrow")

	fact.Status = knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, fact))

	window := int64(h.SubstrateBridgeKeeper.GetParams(h.Ctx).WitnessRewardChallengeWindowBlocks)
	due := h.Ctx.WithBlockHeight(h.Ctx.BlockHeight() + window).WithEventManager(sdk.NewEventManager())
	require.NoError(t, h.SubstrateBridgeKeeper.BeginBlocker(due))

	_, found = h.SubstrateBridgeKeeper.GetWitnessPendingReward(h.Ctx, attID)
	require.False(t, found)
	require.Equal(t, preMinted.String(), h.VestingRewardsKeeper.GetTotalMinted(sdkCtx).String(), "nothing minted")
	require.True(t, h.App.BankKeeper.GetBalance(h.Ctx, submitter, "uzrn").Amount.Equal(preSubmit.Amount),
		"the bond came back and no reward was paid")
	att, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, attID)
	require.True(t, found)
	require.Equal(t, "0", att.RewardUzrn)

	var reasons []string
	for _, e := range due.EventManager().Events() {
		if e.Type != substratebridgekeeper.EventTypeWitnessRewardCancelled {
			continue
		}
		for _, a := range e.Attributes {
			if a.Key == substratebridgekeeper.AttrReason {
				reasons = append(reasons, a.Value)
			}
		}
	}
	require.Equal(t, []string{"cited fact survival-fact is FACT_STATUS_DISPROVEN"}, reasons)
}
