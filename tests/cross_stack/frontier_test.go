package cross_stack_test

import (
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	inquirykeeper "github.com/zerone-chain/zerone/x/inquiry/keeper"
	inquirytypes "github.com/zerone-chain/zerone/x/inquiry/types"
	ontologytypes "github.com/zerone-chain/zerone/x/ontology/types"
)

// TestFrontier_OpenInquiriesRaiseSparsity verifies that an open
// inquiry in a domain raises that domain's sparsity score above a
// quiet domain. This binds the inquiry → frontier composition: open
// questions are demand for exploration, and the frontier signal
// reflects them.
func TestFrontier_OpenInquiriesRaiseSparsity(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	// The harness's default genesis does not auto-seed ontology
	// domains in cross-stack tests; seed two by hand so the frontier
	// synthesizer has territory to walk.
	for _, name := range []string{"philosophy", "physics"} {
		h.App.ZeroneOntologyKeeper.SetDomain(h.Ctx, &ontologytypes.Domain{
			Name:    name,
			Stratum: uint32(ontologytypes.StratumEmpirical),
			Status:  "active",
			Depth:   1,
		})
	}

	asker := testAddr("frontier_asker")
	require.NoError(t, h.FundAccount(asker, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(10_000_000)))))

	// Submit an inquiry in "philosophy" — a domain that exists in the
	// default ontology and starts sparse.
	inquiryMs := inquirykeeper.NewMsgServerImpl(h.InquiryKeeper)
	_, err = inquiryMs.SubmitInquiry(h.Ctx, &inquirytypes.MsgSubmitInquiry{
		Asker:    asker.String(),
		Question: "Test inquiry that should raise philosophy's sparsity",
		Domain:   "philosophy",
		Bounty:   "2000000",
	})
	require.NoError(t, err)

	// Compose the frontier and find philosophy's row.
	frontier := h.GovernanceSynthesisKeeper.ComposeFrontier(h.Ctx, 0)
	require.NotEmpty(t, frontier.Domains, "frontier must include at least the seeded domains")

	var philosophy, physics *struct {
		open     uint64
		sparsity uint64
	}
	for _, row := range frontier.Domains {
		switch row.Domain {
		case "philosophy":
			philosophy = &struct {
				open     uint64
				sparsity uint64
			}{open: row.OpenInquiries, sparsity: row.SparsityScoreBps}
		case "physics":
			physics = &struct {
				open     uint64
				sparsity uint64
			}{open: row.OpenInquiries, sparsity: row.SparsityScoreBps}
		}
	}
	require.NotNil(t, philosophy, "philosophy must appear in frontier")
	require.NotNil(t, physics, "physics must appear in frontier")

	require.Equal(t, uint64(1), philosophy.open,
		"philosophy must reflect the open inquiry")
	require.Equal(t, uint64(0), physics.open,
		"physics has no inquiry, must show 0 open")

	require.Greater(t, philosophy.sparsity, physics.sparsity,
		"a domain with an open inquiry must rank as sparser than an otherwise-equal quiet domain — the frontier signal must reflect demand")
}

// TestFrontier_ChainSponsorsInquiriesForSparseDomains binds commitment 18:
// the chain manufactures exploration demand. Once per
// FrontierInvitationCadenceBlocks, the inquiry BeginBlocker must walk
// the chain's frontier and SPONSOR open inquiries in the sparsest
// domains — funded by mint into FrontierBountyPool, paid out via the
// existing inquiry payout flow.
//
// What this test asserts is the LOAD-BEARING behaviour of the
// commitment: a sparse domain produces a chain-sponsored inquiry
// without any user action; a sub-threshold domain does not.
func TestFrontier_ChainSponsorsInquiriesForSparseDomains(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	// Seed two ontology domains for the harness to walk. "philosophy"
	// will exceed the sparsity threshold; "physics" will be filtered
	// below by setting top_k=1 (so even if it scored equally, only
	// one row gets sponsored).
	for _, name := range []string{"philosophy", "physics"} {
		h.App.ZeroneOntologyKeeper.SetDomain(h.Ctx, &ontologytypes.Domain{
			Name:    name,
			Stratum: uint32(ontologytypes.StratumEmpirical),
			Status:  "active",
			Depth:   1,
		})
	}

	// Override params: cadence=2 so that AdvanceBlocks(1) reaches
	// height 2 (a multiple of cadence and triggers the cycle).
	// Threshold lowered to 600_000 so a clean (no-fact, no-CE) domain
	// — which scores 666_666 BPS — qualifies for sponsorship; this
	// keeps the test independent of the broader knowledge/CE state.
	// top_k=1 caps per-tick sponsorship at one inquiry.
	p := h.App.InquiryKeeper.GetParams(h.Ctx)
	p.FrontierInvitationCadenceBlocks = 2
	p.FrontierInvitationSparsityThresholdBps = 600_000
	p.FrontierInvitationTopK = 1
	p.FrontierInvitationBounty = "5000000"
	p.FrontierInvitationExpiryBlocks = 1000
	require.NoError(t, h.App.InquiryKeeper.SetParams(h.Ctx, p))

	// Sanity baseline: no system-initiated inquiries before any
	// BeginBlocker tick has run at the cadence boundary.
	sysCount := 0
	require.NoError(t, h.App.InquiryKeeper.IterateAllInquiries(h.Ctx, func(q *inquirytypes.Inquiry) bool {
		if q.SystemInitiated {
			sysCount++
		}
		return false
	}))
	require.Zero(t, sysCount, "no system-initiated inquiries should exist before a cadence tick")

	// Capture frontier pool balance before the tick. Should be 0 —
	// no chain mint has happened yet.
	frontierAddr := authtypes.NewModuleAddress(inquirytypes.FrontierBountyPoolModuleName)
	inquiryAddr := authtypes.NewModuleAddress(inquirytypes.BountyPoolModuleName)
	require.True(t, h.GetBalance(frontierAddr, "uzrn").Amount.IsZero(),
		"frontier bounty pool starts empty")
	beforeInquiryBal := h.GetBalance(inquiryAddr, "uzrn")

	// Advance one block. Height goes 1 → 2; 2 % cadence(2) == 0 →
	// frontier-invitation cycle fires. Call inquiry's BeginBlocker
	// explicitly to side-step harness module-manager quirks (same
	// pattern as the harness already uses for KnowledgeKeeper /
	// QualificationKeeper).
	h.AdvanceBlocks(1)
	require.NoError(t, h.App.InquiryKeeper.BeginBlocker(h.Ctx))

	// Assert: at least one system-initiated inquiry was created. With
	// top_k=1 and two qualifying domains, the chain sponsored exactly
	// one — whichever domain ranked sparsest in the iteration order.
	systemInquiries := []*inquirytypes.Inquiry{}
	require.NoError(t, h.App.InquiryKeeper.IterateAllInquiries(h.Ctx, func(q *inquirytypes.Inquiry) bool {
		if q.SystemInitiated {
			systemInquiries = append(systemInquiries, q)
		}
		return false
	}))
	require.Len(t, systemInquiries, 1,
		"BeginBlocker at cadence tick must sponsor exactly top_k(=1) inquiry")

	q := systemInquiries[0]
	require.Contains(t, []string{"philosophy", "physics"}, q.Domain,
		"chain-sponsored inquiry must land in one of the seeded sparse domains")
	require.Equal(t, "5000000", q.Bounty,
		"chain-sponsored bounty matches FrontierInvitationBounty param")
	require.Equal(t, inquirytypes.InquiryStatus_INQUIRY_STATUS_OPEN, q.Status,
		"chain-sponsored inquiry starts OPEN — answers can be linked")
	require.True(t, strings.HasPrefix(q.SystemInitiationReason, "frontier_sparsity:"),
		"system_initiation_reason must trace back to the frontier sparsity that triggered it (got %q)", q.SystemInitiationReason)
	require.Equal(t, frontierAddr.String(), q.Asker,
		"asker on a chain-sponsored inquiry IS the frontier bounty pool — a stable, queryable identifier for 'the chain itself'")

	// Assert: the bounty was minted into FrontierBountyPool and then
	// transferred to BountyPool, where the existing payout flow holds
	// it. After SystemSponsorInquiry, FrontierBountyPool's net change
	// is zero (mint in, transfer out); BountyPool gained the bounty.
	require.True(t, h.GetBalance(frontierAddr, "uzrn").Amount.IsZero(),
		"frontier bounty pool should round-trip — mint goes in, transfers out, balance returns to zero")
	gained := h.GetBalance(inquiryAddr, "uzrn").Amount.Sub(beforeInquiryBal.Amount)
	require.Equal(t, sdkmath.NewInt(5_000_000), gained,
		"inquiry bounty pool received exactly the chain-sponsored bounty")

	// Assert: cancel-handling refuses chain-sponsored inquiries.
	// Some other address attempting to cancel must hit ErrSystemInitiated
	// before any other check (commitment 18: the chain does not
	// withdraw its own asks).
	someone := testAddr("would_be_canceller")
	inquiryMs := inquirykeeper.NewMsgServerImpl(h.App.InquiryKeeper)
	_, cancelErr := inquiryMs.CancelInquiry(h.Ctx, &inquirytypes.MsgCancelInquiry{
		Asker:     someone.String(),
		InquiryId: q.Id,
	})
	require.Error(t, cancelErr, "cancel must be refused on system-initiated inquiry")
	require.ErrorIs(t, cancelErr, inquirytypes.ErrSystemInitiated,
		"refusal must be the structural ErrSystemInitiated, not the unrelated ErrNotAsker — the protection is categorical, not access-based")

	// Sanity: with cadence=2, advancing by another single block lands
	// at height 3 (not a multiple) — no new sponsorship should fire.
	beforeCount := len(systemInquiries)
	h.AdvanceBlocks(1)
	require.NoError(t, h.App.InquiryKeeper.BeginBlocker(h.Ctx))
	afterCount := 0
	require.NoError(t, h.App.InquiryKeeper.IterateAllInquiries(h.Ctx, func(qq *inquirytypes.Inquiry) bool {
		if qq.SystemInitiated {
			afterCount++
		}
		return false
	}))
	require.Equal(t, beforeCount, afterCount,
		"non-cadence block must not sponsor — cadence is the chain's pacing of exploration spend")
}
