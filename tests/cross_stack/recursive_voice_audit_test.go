package cross_stack_test

import (
	"strings"
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

// TestRecursiveVoiceAudit_StagedRecursionEventsCarryDoctrineAttributes runs a
// staged cited-fact/sponsorship composition and asserts that each selected
// sponsorship/bridge event carries either a `creed_commitment`
// attribute (truth-seeking-bound) or a `useful_work_commitment` /
// `mechanism` attribute (UW-bound), or both.
//
// The assertion is intentionally scoped to the selected sponsorship and
// substrate-bridge events in this staged composition. It is not a universal
// audit of every chain event or of the unwired pending-claim loop.
func TestRecursiveVoiceAudit_StagedRecursionEventsCarryDoctrineAttributes(t *testing.T) {
	h := NewTestHarness(t)

	// Setup: adapter, domain, sponsor, submitter.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId:              selfcompile.AdapterID,
		Status:                 substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		MinAttestationBondUzrn: "222000",
		// A weighted claim requires a declared ceiling (agenttool-seam-v1): an
		// adapter that declares none accepts no recursion weight at all. These
		// are the exact maxima ComputeAxisProjection can emit
		// (tools/zerone-self-compiler/compile/compile.go:134-161), so the
		// fixture bounds say what this adapter is actually permitted to claim
		// rather than leaving it unbounded by omission.
		AxisBounds: &substratebridgetypes.AxisBounds{
			AxisSubstrateMax:      50_000,
			AxisVerificationMax:   100_000,
			AxisClassificationMax: 50_000,
			AxisAttributionMax:    500_000,
			AxisToolingMax:        200_000,
			AxisInterfaceMax:      100_000,
		},
	}))
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   selfcompile.SelfDomain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))
	sponsor := testAddr("voice_audit_sponsor")
	submitter := testAddr("voice_audit_submitter")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(100_000_000)))))
	require.NoError(t, h.FundAccount(submitter, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(10_000_000)))))

	// Drive the loop, capturing event counts at each stage. The harness
	// EventManager accumulates events across all msg invocations within
	// the same ctx; we read it at the end and audit every event.

	spSrv := sponsorshipkeeper.NewMsgServerImpl(h.SponsorshipKeeper)
	sbSrv := substratebridgekeeper.NewMsgServerImpl(h.SubstrateBridgeKeeper)

	// Stage 1: bounty creation.
	createResp, err := spSrv.CreateBountyOrder(h.Ctx, &sponsorshiptypes.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: selfcompile.SelfDomain,
		PricePerArtifact: "1000000", TargetCount: 1, DurationBlocks: 2000,
		WorkContract: sponsorshipV2WorkContract(submitter.String()),
	})
	require.NoError(t, err)

	// Stage 2: attestation submission.
	commitTime, _ := time.Parse(time.RFC3339, "2026-05-11T17:52:35Z")
	link, err := selfcompile.Compile(selfcompile.CommitMeta{
		Hash:         "80cf9c0400327e016e41cc9df441371056c958ef",
		Author:       "YOU <x@x>",
		Date:         commitTime,
		Subject:      "voice-audit test",
		TouchedFiles: []string{"x/sponsorship/keeper/msg_server.go"},
	}, uint64(h.Ctx.BlockHeight()))
	require.NoError(t, err)

	// Stage 2b: seed the verified fact FIRST (simulate verification), then
	// cite it. Pending-claim links are fail-closed at the msg server until
	// their x/knowledge translation is wired (ToK Plan 4) — the self-loop
	// cites verified facts instead.
	const selfFactID = "voice-audit-fact-1"
	selfFact := sponsorshipV2Fact(selfFactID, selfcompile.SelfDomain, submitter.String(), uint64(h.Ctx.BlockHeight()))
	selfFact.Content = link.PendingClaims[0].ClaimContent
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, selfFact))
	link.PendingClaims = nil
	link.CitedFacts = []*substratebridgetypes.FactCitation{
		{FactId: selfFactID, CitationType: substratebridgetypes.CitationType_CITATION_TYPE_SUPPORTS},
	}
	link.LinkHash = substratebridgekeeper.ComputeLinkHash(link)

	_, err = sbSrv.SubmitExternalAttestation(h.Ctx, &substratebridgetypes.MsgSubmitExternalAttestation{
		Submitter:   submitter.String(),
		AdapterId:   selfcompile.AdapterID,
		WorkClassId: "zerone_self_attestation",
		Link:        link,
		BondUzrn:    "1000000",
	})
	require.NoError(t, err)

	// Stage 4: fulfillment.
	_, err = spSrv.FulfillBounty(h.Ctx, &sponsorshiptypes.MsgFulfillBounty{
		Caller:   submitter.String(),
		BountyId: createResp.BountyId,
		FactId:   selfFactID,
	})
	require.NoError(t, err)

	// ── The audit ────────────────────────────────────────────────────

	// Collect and filter the events emitted by this staged composition
	// from modules participating in this recursion (sponsorship +
	// substrate_bridge); the harness EventManager also contains events
	// from setup (bank send, etc.) which are not under our doctrine.
	// substrate_bridge emits with bare event types ("external_attestation_*",
	// "adapter_*", "lineage_*"); sponsorship emits with the full
	// "zerone.sponsorship." prefix. Match both conventions so the audit
	// is honest across modules.
	isSubstrateBridgeEvent := func(t string) bool {
		return strings.HasPrefix(t, "external_attestation_") ||
			strings.HasPrefix(t, "adapter_") ||
			strings.HasPrefix(t, "lineage_")
	}
	events := h.Ctx.EventManager().Events()
	recursionEvents := []sdk.Event{}
	for _, e := range events {
		if strings.HasPrefix(e.Type, "zerone.sponsorship.") ||
			isSubstrateBridgeEvent(e.Type) {
			recursionEvents = append(recursionEvents, e)
		}
	}
	require.NotEmpty(t, recursionEvents, "loop must emit at least one recursion event")
	t.Logf("captured %d recursion events from the loop", len(recursionEvents))

	// Doctrine attribute keys that bind voice to creed:
	//   - creed_commitment: truth-seeking doctrine binding (e.g., "20")
	//   - useful_work_commitment: UW doctrine binding (e.g., "UW")
	//   - mechanism: UW mechanism binding (e.g., "M2", "M3", "M4")
	doctrineKeys := map[string]bool{
		"creed_commitment":       true,
		"useful_work_commitment": true,
		"mechanism":              true,
	}

	// Every recursion event must carry at least one doctrine attribute.
	// If any event in this trail is silent, the chain has emitted state
	// change without naming the doctrine it preserves.
	silent := []string{}
	for _, e := range recursionEvents {
		bound := false
		for _, attr := range e.Attributes {
			if doctrineKeys[attr.Key] {
				bound = true
				break
			}
		}
		if !bound {
			silent = append(silent, e.Type)
		}
	}
	require.Empty(t, silent,
		"recursion events MUST carry a doctrine attribute "+
			"(creed_commitment | useful_work_commitment | mechanism). "+
			"Silent events: %v",
		silent)

	// Sanity: verify at least one event from each module participated.
	moduleSeen := map[string]bool{}
	for _, e := range recursionEvents {
		switch {
		case strings.HasPrefix(e.Type, "zerone.sponsorship."):
			moduleSeen["sponsorship"] = true
		case isSubstrateBridgeEvent(e.Type):
			moduleSeen["substrate_bridge"] = true
		}
	}
	require.True(t, moduleSeen["sponsorship"], "sponsorship must contribute events to the recursion trail")
	require.True(t, moduleSeen["substrate_bridge"], "substrate_bridge must contribute events to the recursion trail")
}
