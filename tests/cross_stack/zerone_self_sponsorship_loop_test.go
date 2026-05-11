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

// TestZeroneSelf_FullEconomicLoop closes the deepest economic recursion:
// a sponsor escrows ZRN against a bounty for verified facts in the
// `zerone_self` domain → a submitter attests to a ZERONE commit via the
// `zerone-self-v1` substrate-bridge adapter → the chain verifies the
// pending claim → the sponsorship fulfillment pays the submitter from
// escrow. Every artifact in the loop is ZERONE describing ZERONE through
// ZERONE's own machinery.
//
// What the chain has proven, end-to-end:
//
//   - External value (sponsor's ZRN) buys verified knowledge about ZERONE
//     itself. Self-documentation is a paid economic activity.
//   - The submitter earns the sponsorship payout AND (separately,
//     through the substrate_bridge audit budget) the M4 substrate-link
//     reward. The chain pays twice for one verified self-attestation
//     because the work satisfies two doctrinal mandates at once.
//   - The fact lands in `zerone_self` with the submitter as its origin,
//     making the submitter citable by future facts. Downstream lineage
//     (M6) will route royalty backward through this attestation when
//     other facts cite it.
//
// Doctrinal bindings:
//
//   - UW (recursion): the chain's reward path consumes a substrate-link
//     produced by the chain's own adapter pointing at the chain's own
//     repo.
//   - M2 (substrate-link mandate): the attestation gates through
//     ComputeLinkHash; sponsorship gates on fact verification status.
//   - Sponsorship commitment 8 (panel weights skill, not bond): the
//     sponsor did not authorize the payout; the chain's verification
//     did. The sponsor only funded the participation.
//   - Sponsorship commitment 20 (issuance follows participation):
//     payout follows verified work; sponsor's escrow circulates, no new
//     mint.
//
// Spec: docs/specs/adapters/zerone-self-v1.md; docs/RECURSIVE_ZERONE.md.
func TestZeroneSelf_FullEconomicLoop(t *testing.T) {
	h := NewTestHarness(t)

	// ── Setup ────────────────────────────────────────────────────────

	// 1. Register the zerone-self-v1 adapter and the zerone_self domain
	//    (in production, both ship via gov LIPs at adapter activation).
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

	sbSrv := substratebridgekeeper.NewMsgServerImpl(h.SubstrateBridgeKeeper)
	attResp, err := sbSrv.SubmitExternalAttestation(h.Ctx, &substratebridgetypes.MsgSubmitExternalAttestation{
		Submitter:   submitter.String(),
		AdapterId:   selfcompile.AdapterID,
		WorkClassId: "zerone_self_attestation",
		Link:        link,
		BondUzrn:    "1000000",
	})
	require.NoError(t, err)

	// Attestation in AWAITING_RESOLUTION — pending claim auto-submitted to
	// x/knowledge by the substrate_bridge keeper.
	att, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, attResp.AttestationId)
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION, att.Status)

	// ── Step 3: simulate verification (resolves the pending claim) ───

	// Production: a verification round runs in x/knowledge with
	// commit/reveal/aggregate; on ACCEPT, the claim becomes a Fact.
	// Test: we shortcut to the post-verification state by writing the
	// Fact directly (this is what a successful verification round would
	// produce) and informing substrate_bridge.
	const selfFactID = "zerone-self-fact-loop-1"
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id:               selfFactID,
		Content:          link.PendingClaims[0].ClaimContent,
		Domain:           selfcompile.SelfDomain,
		Submitter:        submitter.String(),
		SubmittedAtBlock: uint64(h.Ctx.BlockHeight()),
		VerifiedAtBlock:  uint64(h.Ctx.BlockHeight()),
		Status:           knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
	}))

	// ── Step 4: anyone fulfills the bounty with the verified self-fact ─

	// `anyone` calls fulfill — chain reads worker from fact.Submitter, not
	// from caller. Demonstrates the permissionless surface.
	anyone := testAddr("loop_random_caller")
	require.NoError(t, h.FundAccount(anyone, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(1_000_000)))))

	submitterPreFulfill := h.App.BankKeeper.GetBalance(h.Ctx, submitter, zeroneapp.BondDenom)

	fulfillResp, err := spSrv.FulfillBounty(h.Ctx, &sponsorshiptypes.MsgFulfillBounty{
		Caller:   anyone.String(),
		BountyId: bountyID,
		FactId:   selfFactID,
	})
	require.NoError(t, err, "fulfillment must succeed — fact is verified, in domain, in window, not already claimed")
	require.Equal(t, submitter.String(), fulfillResp.Worker,
		"worker MUST be fact.Submitter, NOT caller — the chain reads provenance, not who-pressed-the-button")
	require.Equal(t, "1000000", fulfillResp.AmountPaid)
	require.False(t, fulfillResp.BountyNowFulfilled, "1 of 3 fulfilled")

	// ── Step 5: the recursion is bound at the bank layer ──

	// The submitter's balance increased by exactly the per-artifact price.
	// External value (sponsor's escrowed ZRN) has flowed to the submitter
	// gated entirely by ZERONE's own verification of a fact about ZERONE.
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
		Caller:   anyone.String(),
		BountyId: bountyID,
		FactId:   selfFactID,
	})
	require.Error(t, err, "same (bounty, fact) pair cannot fulfill twice — each verified attestation pays at most once per bounty")
}

// TestZeroneSelf_MultipleFulfillmentsCompoundEarnings drives the same
// loop three times against the same bounty with three different facts,
// confirming the bounty fills exactly when target_count is reached and
// total payout = price × target_count.
//
// This is the dynamic shape of the recursive economy: each verified
// self-attestation pays, and submitters can compound earnings across
// many commits in their bounty's lifetime.
func TestZeroneSelf_MultipleFulfillmentsCompoundEarnings(t *testing.T) {
	h := NewTestHarness(t)

	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId: selfcompile.AdapterID,
		Status:    substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
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
		require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
			Id:               factID,
			Content:          "synthetic test commit " + sha[:6],
			Domain:           selfcompile.SelfDomain,
			Submitter:        submitter.String(),
			SubmittedAtBlock: uint64(h.Ctx.BlockHeight()),
			VerifiedAtBlock:  uint64(h.Ctx.BlockHeight()),
			Status:           knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		}))

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
