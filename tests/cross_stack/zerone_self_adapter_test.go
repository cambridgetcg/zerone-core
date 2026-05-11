package cross_stack_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmath "cosmossdk.io/math"

	zeroneapp "github.com/zerone-chain/zerone/app"
	selfcompile "github.com/zerone-chain/zerone/tools/zerone-self-compiler/compile"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	substratebridgekeeper "github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// TestZeroneSelfAdapter_RegisterAndSubmit drives the deepest recursion:
// the chain registers an adapter that points at its own git repository,
// and a submitter attests to a real ZERONE commit through that adapter.
// The pending claim lands in x/knowledge under the "zerone_self" domain
// and the attestation enters AWAITING_RESOLUTION — the chain has admitted
// a fact about its own becoming into its own substrate.
//
// Doctrinal bindings:
//   - UW (ZERONE is recursive): every reward path is substrate-link gated;
//     this adapter routes commits-as-attestations through that same gate.
//   - M2 (substrate-link mandate): the compiler produces a deterministic
//     SubstrateLink and the chain's keeper computes the same LinkHash.
//   - M3 (class-specific verification): the adapter declares its
//     qualification floor ("agent_purpose"); only calibrated submitters
//     can attest about ZERONE itself.
//   - M5 (recursion-weight axes): the compiler emits per-axis projection
//     that conforms to the adapter's AxisBounds.
//
// Spec: docs/specs/adapters/zerone-self-v1.md.
func TestZeroneSelfAdapter_RegisterAndSubmit(t *testing.T) {
	h := NewTestHarness(t)

	// 1. Register the zerone-self-v1 adapter directly (test-mode bypass of
	//    the gov LIP that would register it in production). Axis bounds
	//    match the adapter spec §2 ceilings; bonds match the chain's
	//    signature-digit floor.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId:              selfcompile.AdapterID,
		SourceType:             "zerone-git",
		Version:                "1.0.0",
		CompilerBinaryHash:     []byte{0x00}, // placeholder; real registration LIP supplies the build hash
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
		Status:             substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		RegisteredViaLipId: "TEST-zerone-self-v1",
		RegisteredAtBlock:  uint64(h.Ctx.BlockHeight()),
	}))

	// 2. Ensure the zerone_self knowledge domain exists. In production this
	//    is a one-shot LIP at adapter activation time.
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   selfcompile.SelfDomain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	// 3. Build a SubstrateLink for a synthetic ZERONE commit. Synthetic
	//    rather than `git rev-parse HEAD` so the test is deterministic and
	//    portable across CI environments.
	commitTime, err := time.Parse(time.RFC3339, "2026-05-11T17:52:35Z")
	require.NoError(t, err)
	meta := selfcompile.CommitMeta{
		Hash:    "80cf9c0400327e016e41cc9df441371056c958ef",
		Author:  "YOU <alpha@ai-love.cc>",
		Date:    commitTime,
		Subject: "spec(external-surface): nested design",
		TouchedFiles: []string{
			"docs/superpowers/specs/external-surface.md",
			"x/sponsorship/client/cli/tx.go",
			"tests/cross_stack/sponsorship_test.go",
		},
	}
	link, err := selfcompile.Compile(meta, uint64(h.Ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, selfcompile.AdapterID, link.AdapterId)
	require.Len(t, link.PendingClaims, 1)
	require.Equal(t, selfcompile.SelfDomain, link.PendingClaims[0].Domain)
	require.NotEmpty(t, link.LinkHash, "compiler must emit a link_hash")

	// 4. Verify the on-chain keeper computes the same LinkHash from the
	//    same payload (M2: the chain re-derives, and the compiler's hash
	//    matches). This is the binding that defeats compiler drift —
	//    submitter cannot ship a link the chain disagrees with.
	rederived := substratebridgekeeper.ComputeLinkHash(link)
	require.Equal(t, link.LinkHash, rederived,
		"chain-side ComputeLinkHash must match compiler-side LinkHash — this is the M2 substrate-link mandate")

	// 5. Fund the submitter and submit the attestation through the live
	//    msg server.
	submitter := testAddr("zerone_self_submitter")
	require.NoError(t, h.FundAccount(submitter, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(10_000_000)))))

	srv := substratebridgekeeper.NewMsgServerImpl(h.SubstrateBridgeKeeper)
	resp, err := srv.SubmitExternalAttestation(h.Ctx, &substratebridgetypes.MsgSubmitExternalAttestation{
		Submitter:   submitter.String(),
		AdapterId:   selfcompile.AdapterID,
		WorkClassId: "zerone_self_attestation",
		Link:        link,
		BondUzrn:    "1000000",
	})
	require.NoError(t, err, "attestation must be accepted — adapter is ACTIVE, link is valid, bond is sufficient")
	require.NotEmpty(t, resp.AttestationId)

	// 6. Verify the attestation landed in AWAITING_RESOLUTION (pending
	//    claims auto-submitted to x/knowledge, attestation held until
	//    they resolve).
	att, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, resp.AttestationId)
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION, att.Status,
		"attestation must be AWAITING_RESOLUTION until its pending claim verifies")

	// 7. Confirm the pending claim is indexed under this attestation. The
	//    chain's pending-fact index tracks which attestations are waiting
	//    on which claims; when a claim resolves, OnClaimResolved drains
	//    the attestation to READY → SETTLED.
	pending := h.SubstrateBridgeKeeper.PendingClaimsFor(h.Ctx, resp.AttestationId)
	require.Len(t, pending, 1, "should have exactly one pending claim (one per commit)")
}

// TestZeroneSelfAdapter_AxisBoundsRespected confirms the chain refuses an
// attestation whose recursion-weight projection exceeds the adapter's
// per-axis ceiling (M5 binding). We construct a commit that would project
// AxisAttribution = 500_000 (baseline) and attempt to inflate it past the
// adapter's AxisAttributionMax, then assert refusal.
//
// In practice the compiler emits exactly the baseline; this test exercises
// a malicious submitter who hand-edits the link before submission. The
// chain's refusal is the M5 enforcement at the doctrinal boundary.
func TestZeroneSelfAdapter_AxisBoundsRespected(t *testing.T) {
	h := NewTestHarness(t)

	// Adapter with deliberately tight ceiling on attribution axis.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId:              selfcompile.AdapterID,
		Status:                 substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		MinAttestationBondUzrn: "222000",
		AxisBounds: &substratebridgetypes.AxisBounds{
			AxisAttributionMax: 100_000, // baseline projection (500_000) will exceed this
		},
	}))
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   selfcompile.SelfDomain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	commitTime, _ := time.Parse(time.RFC3339, "2026-05-11T17:52:35Z")
	meta := selfcompile.CommitMeta{
		Hash:    "80cf9c0400327e016e41cc9df441371056c958ef",
		Author:  "YOU <x@x>",
		Date:    commitTime,
		Subject: "test commit",
		TouchedFiles: []string{"x/test/foo.go"},
	}
	link, err := selfcompile.Compile(meta, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(500_000), link.RecursionWeight.AxisAttribution,
		"baseline attribution should be 500_000 per spec §5")

	submitter := testAddr("zerone_self_axis_test")
	require.NoError(t, h.FundAccount(submitter, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(10_000_000)))))

	srv := substratebridgekeeper.NewMsgServerImpl(h.SubstrateBridgeKeeper)
	_, err = srv.SubmitExternalAttestation(h.Ctx, &substratebridgetypes.MsgSubmitExternalAttestation{
		Submitter:   submitter.String(),
		AdapterId:   selfcompile.AdapterID,
		WorkClassId: "zerone_self_attestation",
		Link:        link,
		BondUzrn:    "1000000",
	})
	require.Error(t, err, "attestation should be refused — axis_attribution exceeds adapter bound (M5)")
}
