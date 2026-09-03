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
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// TestRecursiveDoublePayment_ManuallyStagedStateExercisesTwoPayouts proves
// that two independently staged primitives can pay the same submitter: an M4
// bridge settlement mints through MintWithCap, and sponsorship transfers from
// escrow. The test directly writes READY attestation and VERIFIED fact state;
// runtime does not link them through the unwired pending-claim bridge.
//
// The two payouts use different mechanisms. The bridge reward mints new uzrn;
// sponsorship moves existing escrow. This is a composition scaffold, not
// end-to-end evidence that one verified self-attestation triggered both.
//
// This is not double-spending — it's compound payment for compound value.
func TestRecursiveDoublePayment_ManuallyStagedStateExercisesTwoPayouts(t *testing.T) {
	h := NewTestHarness(t)
	activateAgentEconomySourceCandidate(t, h)

	// ── Setup: adapter, domain, accounts ─────────────────────────────

	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId:              selfcompile.AdapterID,
		Status:                 substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		MinAttestationBondUzrn: "222000",
	}))
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   selfcompile.SelfDomain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	sponsor := testAddr("dp_sponsor")
	submitter := testAddr("dp_submitter")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(100_000_000)))))

	initialSubmitterBalance := h.App.BankKeeper.GetBalance(h.Ctx, submitter, zeroneapp.BondDenom)
	require.True(t, initialSubmitterBalance.Amount.IsZero(),
		"submitter should start with zero balance so payouts are unambiguous")

	// ── Step 1: sponsor posts bounty in zerone_self ──────────────────

	spSrv := sponsorshipkeeper.NewMsgServerImpl(h.SponsorshipKeeper)
	createResp, err := spSrv.CreateBountyOrder(h.Ctx, &sponsorshiptypes.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: selfcompile.SelfDomain,
		PricePerArtifact: "500000", TargetCount: 1, DurationBlocks: 2000,
		WorkContract: sponsorshipV2WorkContract(submitter.String()),
	})
	require.NoError(t, err)

	// ── Step 2: drive substrate_bridge to SETTLED ────────────────────

	// Build a SubstrateLink for a synthetic ZERONE commit.
	commitTime, _ := time.Parse(time.RFC3339, "2026-05-11T17:52:35Z")
	link, err := selfcompile.Compile(selfcompile.CommitMeta{
		Hash:    "80cf9c0400327e016e41cc9df441371056c958ef",
		Author:  "YOU <x@x>",
		Date:    commitTime,
		Subject: "double-payment test",
		TouchedFiles: []string{
			"x/sponsorship/keeper/msg_server.go",
			"x/substrate_bridge/keeper/settlement.go",
		},
	}, uint64(h.Ctx.BlockHeight()))
	require.NoError(t, err)

	// Write the attestation directly in READY status. The reward is
	// computed by M4 (base + L × W × Q) at settle-time and — since the
	// audit bounty pool mints through MintWithCap — actually paid to the
	// submitter at settlement. Sponsorship later pays against a separately
	// seeded fact with matching content; runtime does not enforce their
	// identity.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId:    "dp-self-att",
		AdapterId:        selfcompile.AdapterID,
		WorkClassId:      "zerone_self_attestation",
		Submitter:        submitter.String(),
		SubmittedAtBlock: uint64(h.Ctx.BlockHeight()),
		Status:           substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_READY,
		Link:             link,
		BondUzrn:         "0",
		VerifiedCount:    1,
	}))

	require.NoError(t, h.SubstrateBridgeKeeper.SettleAttestation(h.Ctx, "dp-self-att"))

	// ── Step 3: substrate_bridge declared a non-zero audit reward ────

	att, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, "dp-self-att")
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status,
		"substrate_bridge must drive to SETTLED — M4 mechanism declared the audit-quality reward")
	require.NotEqual(t, "0", att.RewardUzrn,
		"M4 reward must be non-zero — the chain has declared substrate_bridge owes the submitter")
	require.NotEmpty(t, att.RewardUzrn)
	t.Logf("substrate_bridge M4 reward declared: %s uzrn", att.RewardUzrn)

	// The reward is no longer merely declared — it mints through the cap
	// and lands in the submitter's balance at settlement.
	postSettleBalance := h.App.BankKeeper.GetBalance(h.Ctx, submitter, zeroneapp.BondDenom)
	declaredReward, _ := sdkmath.NewIntFromString(att.RewardUzrn)
	require.True(t, postSettleBalance.Amount.Equal(initialSubmitterBalance.Amount.Add(declaredReward)),
		"M4 reward must be PAID at settlement, not merely declared")

	// ── Step 4: seed the verified fact in knowledge (simulating          ─
	//          successful verification of the pending claim)             ─

	const selfFactID = "dp-self-fact-1"
	selfFact := sponsorshipV2Fact(selfFactID, selfcompile.SelfDomain, submitter.String(), uint64(h.Ctx.BlockHeight()))
	selfFact.Content = link.PendingClaims[0].ClaimContent
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, selfFact))

	// ── Step 5: sponsorship fulfill pays from sponsor's escrow ───────

	fulfillResp, err := spSrv.FulfillBounty(h.Ctx, &sponsorshiptypes.MsgFulfillBounty{
		Caller:   submitter.String(),
		BountyId: createResp.BountyId,
		FactId:   selfFactID,
	})
	require.NoError(t, err)
	require.Equal(t, submitter.String(), fulfillResp.Worker)
	require.Equal(t, "500000", fulfillResp.AmountPaid)

	postFulfillBalance := h.App.BankKeeper.GetBalance(h.Ctx, submitter, zeroneapp.BondDenom)
	sponsorshipPayout := postFulfillBalance.Amount.Sub(postSettleBalance.Amount)
	require.True(t, sponsorshipPayout.Equal(sdkmath.NewInt(500_000)),
		"sponsorship payout must equal price_per_artifact exactly")
	t.Logf("sponsorship payout received: %s uzrn", sponsorshipPayout.String())

	// ── Step 6: bind the recursion — TWO mechanisms paid the submitter ─

	// The independently staged records exercised both payout mechanisms.
	// This does not assert a runtime-enforced attestation↔fact identity.
	require.True(t, sponsorshipPayout.IsPositive(),
		"sponsorship bank delta must be positive — escrow-to-submitter transfer occurred")
	t.Logf("RECURSION #3 SCAFFOLD: staged attestation produced "+
		"(a) M4 reward %s uzrn on attestation %s, "+
		"(b) staged fact produced sponsorship payout %s uzrn from escrow.",
		att.RewardUzrn, att.AttestationId, sponsorshipPayout)
}
