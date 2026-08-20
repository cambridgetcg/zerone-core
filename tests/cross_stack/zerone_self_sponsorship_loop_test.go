package cross_stack_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	selfcompile "github.com/zerone-chain/zerone/tools/zerone-self-compiler/compile"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	sponsorshipkeeper "github.com/zerone-chain/zerone/x/sponsorship/keeper"
	sponsorshiptypes "github.com/zerone-chain/zerone/x/sponsorship/types"
	substratebridgekeeper "github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// TestZeroneSelf_ScaffoldedEconomicLoopRequiresManualBridgeState proves two
// narrower facts: the public bridge refuses zerone-self pending claims, and a
// separately verified fact can fulfill a sponsorship bounty. The test writes
// post-bridge attestation/fact state directly; it is not end-to-end evidence
// that the adapter translates, verifies, settles, or pays lineage royalties.
//
// What the scaffold proves:
//
//   - Pending-claim submission fails closed before escrow.
//   - Sponsorship pays the submitter recorded on a verified fact.
//   - The missing translation boundary cannot be hidden by the later payout.
//
// The scaffold binds fail-closed pending-claim admission, sponsorship's
// status/domain checks, escrow-only payout, and pair idempotency. It does not
// bind panel verification or connect the payout to the refused attestation.
//
// Spec: docs/specs/adapters/zerone-self-v1.md; docs/RECURSIVE_ZERONE.md.
func TestZeroneSelf_ScaffoldedEconomicLoopRequiresManualBridgeState(t *testing.T) {
	h := NewTestHarness(t)

	// ── Setup ────────────────────────────────────────────────────────

	// 1. Register the zerone-self-v1 adapter and zerone_self domain directly.
	//    Adapter-registration LIP dispatch is not wired in current source.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId:              selfcompile.AdapterID,
		SourceType:             "zerone-git",
		Status:                 substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		MinAttestationBondUzrn: "222000",
		MinPerClaimBondUzrn:    "222",
		AxisBounds: &substratebridgetypes.AxisBounds{
			AxisSubstrateMax:      200_000,
			AxisVerificationMax:   400_000,
			AxisClassificationMax: 200_000,
			AxisAttributionMax:    1_000_000,
			AxisToolingMax:        1_000_000,
			AxisInterfaceMax:      400_000,
		},
	}))
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   selfcompile.SelfDomain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	// 2. Sponsor account funded; submitter (and future bounty recipient)
	//    funded for substrate-bridge bond. Both addresses are distinct so
	//    we can attribute the bounty payout to the right party.
	sponsor := testAddr("loop_sponsor")
	submitter := testAddr("loop_submitter_zerone_self")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(100_000_000)))))
	require.NoError(t, h.FundAccount(submitter, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(10_000_000)))))

	// ── Step 1: sponsor posts a bounty in the `zerone_self` domain ───

	spSrv := sponsorshipkeeper.NewMsgServerImpl(h.SponsorshipKeeper)
	createResp, err := spSrv.CreateBountyOrder(h.Ctx, &sponsorshiptypes.MsgCreateBountyOrder{
		Sponsor:          sponsor.String(),
		Domain:           selfcompile.SelfDomain,
		PricePerArtifact: "1000000", // 1 ZRN per verified self-fact
		TargetCount:      3,
		DurationBlocks:   2000,
		WorkContract:     sponsorshipV2WorkContract(submitter.String()),
	})
	require.NoError(t, err)
	bountyID := createResp.BountyId

	// Sponsor's escrow locked.
	sponsorPostCreate := h.App.BankKeeper.GetBalance(h.Ctx, sponsor, zeroneapp.BondDenom)
	require.Equal(t, sdkmath.NewInt(100_000_000-3_000_000), sponsorPostCreate.Amount)

	// ── Step 2: submitter attests to a ZERONE commit via zerone-self-v1 ─

	commitTime, _ := time.Parse(time.RFC3339, "2026-05-11T17:52:35Z")
	meta := selfcompile.CommitMeta{
		Hash:    "80cf9c0400327e016e41cc9df441371056c958ef",
		Author:  "YOU <alpha@ai-love.cc>",
		Date:    commitTime,
		Subject: "spec(external-surface): nested design",
		TouchedFiles: []string{
			"docs/superpowers/specs/external-surface.md",
			"tests/cross_stack/sponsorship_test.go",
		},
	}
	link, err := selfcompile.Compile(meta, uint64(h.Ctx.BlockHeight()))
	require.NoError(t, err)

	// The live door refuses pending-claim links until ToK Plan 4 wires
	// their x/knowledge translation (previously they were accepted into an
	// unresolvable AWAITING state and slashed on timeout). The loop's
	// remaining economics run off the verified FACT, so we build the
	// post-submit attestation state via keeper primitives.
	sbSrv := substratebridgekeeper.NewMsgServerImpl(h.SubstrateBridgeKeeper)
	_, err = sbSrv.SubmitExternalAttestation(h.Ctx, &substratebridgetypes.MsgSubmitExternalAttestation{
		Submitter:   submitter.String(),
		AdapterId:   selfcompile.AdapterID,
		WorkClassId: "zerone_self_attestation",
		Link:        link,
		BondUzrn:    "1000000",
	})
	require.ErrorIs(t, err, substratebridgetypes.ErrPendingClaimsNotSupported)

	const loopAttID = "zerone-self-loop-att"
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId: loopAttID,
		AdapterId:     selfcompile.AdapterID,
		Submitter:     submitter.String(),
		BondUzrn:      "1000000",
		Status:        substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
		Link:          link,
	}))

	att, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, loopAttID)
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION, att.Status)

	// ── Step 3: write the verified-status fixture directly ───────────
	//
	// No verification round runs in this test, and substrate_bridge is not
	// informed of a resolution. This deliberately isolates later sponsorship
	// behavior from the missing translation.
	const selfFactID = "zerone-self-fact-loop-1"
	selfFact := sponsorshipV2Fact(selfFactID, selfcompile.SelfDomain, submitter.String(), uint64(h.Ctx.BlockHeight()))
	selfFact.Content = link.PendingClaims[0].ClaimContent
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, selfFact))

	// ── Step 4: the stored worker chooses and signs settlement ───────

	submitterPreFulfill := h.App.BankKeeper.GetBalance(h.Ctx, submitter, zeroneapp.BondDenom)

	fulfillResp, err := spSrv.FulfillBounty(h.Ctx, &sponsorshiptypes.MsgFulfillBounty{
		Caller:   submitter.String(),
		BountyId: bountyID,
		FactId:   selfFactID,
	})
	require.NoError(t, err, "fulfillment must succeed — fixture has verified status, matching domain/window, and is unused")
	require.Equal(t, submitter.String(), fulfillResp.Worker,
		"worker and signer MUST be the stored fact submitter")
	require.Equal(t, "1000000", fulfillResp.AmountPaid)
	require.False(t, fulfillResp.BountyNowFulfilled, "1 of 3 fulfilled")

	// ── Step 5: the recursion is bound at the bank layer ──

	// The submitter's balance increased by exactly the per-artifact price.
	// Sponsor escrow flowed to the submitter because the directly written
	// fixture had VERIFIED status. This test does not prove how it acquired
	// that status.
	submitterPostFulfill := h.App.BankKeeper.GetBalance(h.Ctx, submitter, zeroneapp.BondDenom)
	require.Equal(t, sdkmath.NewInt(1_000_000), submitterPostFulfill.Amount.Sub(submitterPreFulfill.Amount),
		"submitter must receive exactly price_per_artifact from the bounty's escrow")

	// Sponsor's escrow in the sponsorship module account decreased by the
	// same amount.
	bounty, found := h.SponsorshipKeeper.GetBountyOrder(h.Ctx, bountyID)
	require.True(t, found)
	require.Equal(t, "2000000", bounty.EscrowRemaining,
		"escrow drained by exactly one price_per_artifact")
	require.Equal(t, uint32(1), bounty.FulfilledCount)

	// ── Step 6: idempotency — same fact can't double-claim ──

	_, err = spSrv.FulfillBounty(h.Ctx, &sponsorshiptypes.MsgFulfillBounty{
		Caller:   submitter.String(),
		BountyId: bountyID,
		FactId:   selfFactID,
	})
	require.Error(t, err, "same (bounty, fact) pair cannot fulfill twice — each staged fact pays at most once per bounty")
}

// TestZeroneSelf_MultipleFulfillmentsCompoundEarnings drives the same
// loop three times against the same bounty with three different facts,
// confirming the bounty fills exactly when target_count is reached and
// total payout = price × target_count.
//
// This exercises repeated sponsorship fulfillment against three directly
// written VERIFIED-status facts. It does not ingest self-attestations or git
// commits through the bridge.
func TestZeroneSelf_MultipleFulfillmentsCompoundEarnings(t *testing.T) {
	h := NewTestHarness(t)

	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId:              selfcompile.AdapterID,
		Status:                 substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		MinAttestationBondUzrn: "222000",
	}))
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   selfcompile.SelfDomain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	sponsor := testAddr("compound_sponsor")
	submitter := testAddr("compound_submitter")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(100_000_000)))))

	spSrv := sponsorshipkeeper.NewMsgServerImpl(h.SponsorshipKeeper)
	createResp, err := spSrv.CreateBountyOrder(h.Ctx, &sponsorshiptypes.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: selfcompile.SelfDomain,
		PricePerArtifact: "1000000", TargetCount: 3, DurationBlocks: 2000,
		WorkContract: sponsorshipV2WorkContract(submitter.String()),
	})
	require.NoError(t, err)
	bountyID := createResp.BountyId

	preBalance := h.App.BankKeeper.GetBalance(h.Ctx, submitter, zeroneapp.BondDenom)

	for i, sha := range []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"cccccccccccccccccccccccccccccccccccccccc",
	} {
		factID := "self-fact-compound-" + sha[:6]
		fact := sponsorshipV2Fact(factID, selfcompile.SelfDomain, submitter.String(), uint64(h.Ctx.BlockHeight()))
		fact.Content = "synthetic test commit " + sha[:6]
		require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, fact))

		resp, err := spSrv.FulfillBounty(h.Ctx, &sponsorshiptypes.MsgFulfillBounty{
			Caller: submitter.String(), BountyId: bountyID, FactId: factID,
		})
		require.NoError(t, err)
		require.Equal(t, submitter.String(), resp.Worker)
		// Last iteration must mark the bounty complete.
		require.Equal(t, i == 2, resp.BountyNowFulfilled, "fulfilled flag must trip exactly when target reached")
	}

	// Submitter earned 3 × 1M = 3M uzrn from the bounty.
	postBalance := h.App.BankKeeper.GetBalance(h.Ctx, submitter, zeroneapp.BondDenom)
	require.Equal(t, sdkmath.NewInt(3_000_000), postBalance.Amount.Sub(preBalance.Amount),
		"compound earnings = price × target")

	// Bounty must be FULFILLED, escrow exhausted.
	bounty, _ := h.SponsorshipKeeper.GetBountyOrder(h.Ctx, bountyID)
	require.Equal(t, sponsorshiptypes.BountyStatus_BOUNTY_STATUS_FULFILLED, bounty.Status)
	require.Equal(t, "0", bounty.EscrowRemaining)
}
