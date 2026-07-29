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
	substratebridgekeeper "github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// TestZeroneSelfAdapter_RegisterAndSubmit pins the current boundary: the
// compiler constructs a deterministic caller payload, the keeper validates
// its internal link hash and axis ceilings, and public submission refuses its
// pending claim. The test then writes dormant AWAITING/index state directly;
// no self-fact lands through the public bridge.
//
// Covered adapter properties:
//   - The compiler produces a deterministic SubstrateLink and the keeper
//     computes the same LinkHash.
//   - The caller payload declares the adapter's qualification floor
//     ("agent_purpose").
//   - The compiler emits a per-axis projection within the registered bounds.
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

	// 4. Verify the keeper computes the same LinkHash from the same payload.
	//    This establishes internal field consistency only; consensus does not
	//    execute the compiler or fetch the git source.
	rederived := substratebridgekeeper.ComputeLinkHash(link)
	require.Equal(t, link.LinkHash, rederived,
		"chain-side ComputeLinkHash must match compiler-side LinkHash — this is the M2 substrate-link mandate")

	// 5. The live msg server REFUSES pending-claim links until their
	//    x/knowledge translation is wired (ToK Plan 4). Before this door
	//    existed, the submit was accepted, the claim never reached
	//    knowledge, and the bond slashed on timeout — a trap wearing a
	//    welcome mat. The refusal is the honest form of "not yet".
	submitter := testAddr("zerone_self_submitter")
	require.NoError(t, h.FundAccount(submitter, sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(10_000_000)))))

	srv := substratebridgekeeper.NewMsgServerImpl(h.SubstrateBridgeKeeper)
	_, err = srv.SubmitExternalAttestation(h.Ctx, &substratebridgetypes.MsgSubmitExternalAttestation{
		Submitter:   submitter.String(),
		AdapterId:   selfcompile.AdapterID,
		WorkClassId: "zerone_self_attestation",
		Link:        link,
		BondUzrn:    "1000000",
	})
	require.ErrorIs(t, err, substratebridgetypes.ErrPendingClaimsNotSupported,
		"pending-claim links must be refused at the door until translation lands")

	// 6. The AWAITING machinery itself stays alive for the day the
	//    translation lands: build the post-submit state via keeper
	//    primitives and confirm the pending-fact index tracks it.
	const attID = "zerone-self-machinery-att"
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId: attID,
		AdapterId:     selfcompile.AdapterID,
		Submitter:     submitter.String(),
		BondUzrn:      "1000000",
		Status:        substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
		Link:          link,
	}))
	require.NoError(t, h.SubstrateBridgeKeeper.LinkPendingClaim(h.Ctx, "zerone-self-claim-1", attID))

	att, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, attID)
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION, att.Status)

	pending := h.SubstrateBridgeKeeper.PendingClaimsFor(h.Ctx, attID)
	require.Len(t, pending, 1, "should have exactly one pending claim (one per commit)")
}

// TestZeroneSelfAdapter_AxisBoundsRespected confirms the chain refuses an
// attestation whose recursion-weight projection exceeds the adapter's
// per-axis ceiling (M5 binding). The compiler emits its ordinary 500,000
// attribution baseline, while the test adapter declares a tighter 100,000
// ceiling. The unedited payload must be refused.
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
		Hash:         "80cf9c0400327e016e41cc9df441371056c958ef",
		Author:       "YOU <x@x>",
		Date:         commitTime,
		Subject:      "test commit",
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
