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

// TestRecursiveDoublePayment_SelfAttestationEarnsTwice binds recursion #3
// of docs/RECURSIVE_ZERONE.md: a verified self-attestation pays the
// submitter TWICE — once via substrate_bridge's M4 audit-reward formula
// when the attestation settles, once via sponsorship's bounty fulfill
// when the underlying fact is the recipient of a funded bounty.
//
// The same verified work satisfies two doctrinal mandates simultaneously:
//
//   - M4 (substrate-bridge): reward audit quality — link compiled
//     correctly, axes within bounds, pending claims verified
//   - Sponsorship commitment 20: payout follows verified participation
//     in a funded domain
//
// Both payouts route through different mechanisms (substrate_bridge's
// settlement vs. sponsorship's escrow), neither mints new uzrn, both
// are bound to verification. The chain pays twice because the work
// answers two questions at once.
//
// This is not double-spending — it's compound payment for compound value.
func TestRecursiveDoublePayment_SelfAttestationEarnsTwice(t *testing.T) {
	h := NewTestHarness(t)

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
	// computed by M4 (base + L × W × Q) at settle-time; we measure the
	// declared reward on the attestation rather than the bank delta to
	// avoid coupling this test to bond/audit-pool funding (exercised
	// independently by TestSubstrateBridge_HappyPathSettlement). What
	// matters for recursion #3 is that the substrate_bridge mechanism
	// computes a non-zero reward for the SAME work the sponsorship
	// mechanism is about to pay for.
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

	// ── Step 4: seed the verified fact in knowledge (simulating          ─
	//          successful verification of the pending claim)             ─

	const selfFactID = "dp-self-fact-1"
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id:               selfFactID,
		Content:          link.PendingClaims[0].ClaimContent,
		Domain:           selfcompile.SelfDomain,
		Submitter:        submitter.String(),
		SubmittedAtBlock: uint64(h.Ctx.BlockHeight()),
		Status:           knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
	}))

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
	sponsorshipPayout := postFulfillBalance.Amount.Sub(initialSubmitterBalance.Amount)
	require.True(t, sponsorshipPayout.Equal(sdkmath.NewInt(500_000)),
		"sponsorship payout must equal price_per_artifact exactly")
	t.Logf("sponsorship payout received: %s uzrn", sponsorshipPayout.String())

	// ── Step 6: bind the recursion — TWO mechanisms paid the submitter ─

	// Recursion #3: the same verified self-attestation triggered BOTH
	// payouts. substrate_bridge declared its M4 audit reward on the
	// attestation record (att.RewardUzrn) — bound by the SETTLED status
	// and non-zero value asserted above. Sponsorship transferred its
	// price_per_artifact from sponsor escrow to the submitter — bound
	// by the bank delta above. Two mechanisms, one piece of verified
	// work, two payouts.
	require.True(t, sponsorshipPayout.IsPositive(),
		"sponsorship bank delta must be positive — escrow-to-submitter transfer occurred")
	t.Logf("RECURSION #3 BOUND: same verified self-attestation produced "+
		"(a) M4 audit reward %s uzrn declared on attestation %s, "+
		"(b) sponsorship payout %s uzrn transferred from sponsor escrow to submitter. "+
		"The chain pays compound for compound value.",
		att.RewardUzrn, att.AttestationId, sponsorshipPayout)
}
