package cross_stack_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// Route B Wave 9 — Adversarial Audit
//
// Each test is an attack against a named valuable structure. The tests
// document the ATTACK, perform it against the live code, and assert the
// expected defensive behaviour. Failures here mean the structure is
// vulnerable and MUST be hardened before the chain is production-ready.
//
// Attack taxonomy:
//   - Manifest / Merkle layer (most valuable — reproducibility promise)
//   - Economic layer (escrow + vesting + payouts)
//   - Is-ought wall + TVW (epistemic integrity)
//   - Verifier panel (adjudication integrity)
//   - Heartbeat (automation safety)

// ─── ATTACK 1: Composition-depth bomb ────────────────────────────────────
//
// A chain of parent manifests deeper than the bound should be rejected.
// Without this, a malicious pipeline operator could create a DAG so deep
// that bundle assembly becomes a DoS vector.
func TestRouteB_Wave9_CompositionDepthBomb(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave9_depth_op").String()

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-9d", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-9D", Content: "depth target", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
	}))

	// Build a chain up to max depth (8).
	const maxAllowed = 8
	prev := ""
	for i := 0; i < maxAllowed; i++ {
		id := fmt.Sprintf("m-depth-%d", i)
		msg := &knowledgetypes.MsgCreateTrainingManifest{
			Creator: operator, Id: id, PipelineId: "pipe-9d",
			ParentManifestId: prev,
			CorpusSelector: &knowledgetypes.CorpusSelector{
				MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3,
			},
		}
		_, err := ms.CreateTrainingManifest(h.Ctx, msg)
		require.NoError(t, err, "depth=%d should succeed", i)
		_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
			Creator: operator, ManifestId: id,
		})
		require.NoError(t, err)
		prev = id
	}

	// The 9th level must reject.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-depth-bomb", PipelineId: "pipe-9d",
		ParentManifestId: prev,
		CorpusSelector:   &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical},
	})
	require.Error(t, err, "composition beyond max depth must be rejected")
	require.Contains(t, err.Error(), "max depth")
}

// ─── ATTACK 2: Parent must be FINALIZED or ATTESTED ──────────────────────
//
// A DRAFT parent cannot be referenced. This prevents cycle construction
// (cycles require a DRAFT pointing back to itself, which isn't possible
// because parents must be sealed first).
func TestRouteB_Wave9_ParentMustBeSealed(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave9_cycle_op").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-cycle", OperatorAddress: operator, TokenizerVersion: 1,
	}))

	// Create a DRAFT parent.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "draft-parent", PipelineId: "pipe-cycle",
		CorpusSelector: &knowledgetypes.CorpusSelector{},
	})
	require.NoError(t, err)

	// Attempt to build a child on the DRAFT parent — must reject.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "child-of-draft", PipelineId: "pipe-cycle",
		ParentManifestId: "draft-parent",
		CorpusSelector:   &knowledgetypes.CorpusSelector{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "FINALIZED or ATTESTED")
}

// ─── ATTACK 3: Composed-Merkle collision across (parent, delta) pairs ────
//
// Two different parents plus swapped delta content must not yield the same
// composed root. Domain separation between PARENT: and DELTA: segments +
// length prefixes ensure this.
func TestRouteB_Wave9_ComposedMerkleCollisionResistance(t *testing.T) {
	ids1 := knowledgekeeper.SelectedManifestIDs{FactIDs: []string{"A", "B"}}
	ids2 := knowledgekeeper.SelectedManifestIDs{FactIDs: []string{"C"}}

	parentRoot1 := knowledgekeeper.ComputeManifestMerkleRoot(knowledgekeeper.SelectedManifestIDs{
		FactIDs: []string{"P1"},
	})
	parentRoot2 := knowledgekeeper.ComputeManifestMerkleRoot(knowledgekeeper.SelectedManifestIDs{
		FactIDs: []string{"P2"},
	})

	c1 := knowledgekeeper.ComputeComposedManifestMerkleRoot(parentRoot1, ids1)
	c2 := knowledgekeeper.ComputeComposedManifestMerkleRoot(parentRoot2, ids1)
	c3 := knowledgekeeper.ComputeComposedManifestMerkleRoot(parentRoot1, ids2)

	require.NotEqual(t, c1, c2, "different parents must yield different composed roots")
	require.NotEqual(t, c1, c3, "same parent + different delta must yield different composed roots")
	require.NotEqual(t, c2, c3, "both dims different must yield different composed roots")

	// Composed vs flat with same content must NOT collide — domain tags differ.
	flat := knowledgekeeper.ComputeManifestMerkleRoot(ids1)
	require.NotEqual(t, c1, flat,
		"composed commitment must be distinguishable from a flat commitment over the same delta")
}

// ─── ATTACK 4: Child remains verifiable after parent supersession ───────
//
// Parent gets superseded by a newer manifest for the same pipeline. The
// child (which snapshotted parent_merkle_root at create time) must still
// re-verify correctly via AssembleManifestBundle.
func TestRouteB_Wave9_ChildSurvivesParentSupersession(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave9_super_op").String()

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-super-test", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-9S", Content: "p", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
	}))

	// Create parent + finalize.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "parent", PipelineId: "pipe-super-test",
		CorpusSelector: &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{Creator: operator, ManifestId: "parent"})
	require.NoError(t, err)

	// Create + finalize child.
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-9S-CHILD", Content: "delta", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "child", PipelineId: "pipe-super-test",
		ParentManifestId: "parent",
		CorpusSelector:   &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{Creator: operator, ManifestId: "child"})
	require.NoError(t, err)

	// Advance a block and create a newer sibling that will supersede "parent".
	h.AdvanceBlocks(1)
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "parent-v2", PipelineId: "pipe-super-test",
		CorpusSelector: &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{Creator: operator, ManifestId: "parent-v2"})
	require.NoError(t, err)

	// Heartbeat should supersede parent. Note: child is also FINALIZED for
	// this pipeline; by the preferLater rule, whichever has greater
	// finalized_at_block wins. Let's advance one more block.
	h.AdvanceBlocks(1)

	// The child's bundle must still re-verify. Its parent_merkle_root is
	// snapshotted; parent's on-chain status doesn't affect child correctness.
	bundle, err := qs.TrainingManifestBundle(h.Ctx, &knowledgetypes.QueryTrainingManifestBundleRequest{Id: "child"})
	require.NoError(t, err)
	require.True(t, bundle.MerkleRootValid,
		"child's composed Merkle root must remain valid even after parent's status changes")
}

// ─── ATTACK 5: Augmentation-of-augmentation ─────────────────────────────
//
// An Augmentation's original_fact_id must resolve to a Fact, not another
// Augmentation. Recursive variant-of-variant is not allowed (would bypass
// the methodology-adjudication grounding on the original fact).
func TestRouteB_Wave9_NoAugmentationOfAugmentation(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave9_augaug").String()

	// Seed an ORIGINAL augmentation (not backed by a fact with the same id).
	require.NoError(t, h.KnowledgeKeeper.SetAugmentation(h.Ctx, &knowledgetypes.Augmentation{
		Id: "AUG-9", OriginalFactId: "F-ORIG", VariantContent: "x", Submitter: operator,
	}))

	// Attempt to submit an augmentation whose "original" is actually an augmentation id.
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: operator, Id: "AUG-NESTED",
		OriginalFactId: "AUG-9",
		VariantContent: "nested variant",
	})
	require.Error(t, err,
		"augmentation-of-augmentation must reject (no Fact with id AUG-9)")
	require.Contains(t, err.Error(), "not found")
}

// ─── ATTACK 6: Duplicate manifest ID ─────────────────────────────────────
//
// Creating a manifest with an existing ID must reject.
func TestRouteB_Wave9_DuplicateManifestID(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave9_dup").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-dup", OperatorAddress: operator, TokenizerVersion: 1,
	}))

	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-dup", PipelineId: "pipe-dup",
		CorpusSelector: &knowledgetypes.CorpusSelector{},
	})
	require.NoError(t, err)

	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-dup", PipelineId: "pipe-dup",
		CorpusSelector: &knowledgetypes.CorpusSelector{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

// ─── ATTACK 7: Non-creator cannot finalize ──────────────────────────────
//
// Only the original creator may finalize. A manifest hijacked by a
// different pipeline operator would allow unauthorized sealing.
func TestRouteB_Wave9_ManifestHijackFinalize(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	op1 := testAddr("wave9_op1").String()
	op2 := testAddr("wave9_op2").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-hijack", OperatorAddress: op1, TokenizerVersion: 1,
	}))

	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: op1, Id: "m-hijack", PipelineId: "pipe-hijack",
		CorpusSelector: &knowledgetypes.CorpusSelector{},
	})
	require.NoError(t, err)

	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: op2, ManifestId: "m-hijack",
	})
	require.Error(t, err, "non-creator must not finalize")
	require.Contains(t, err.Error(), "creator")
}

// ─── ATTACK 8: Is-ought wall smuggling ──────────────────────────────────
//
// A model owner lists a NormativeCommitment ID as a trained-on fact_id.
// The chain must reject it (not smuggle it as a quiet ought-claim into
// training revenue).
func TestRouteB_Wave9_IsOughtSmuggling(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	owner := testAddr("wave9_smuggle").String()

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-smug", OperatorAddress: owner, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetModelCard(h.Ctx, &knowledgetypes.ModelCard{
		Id: "model-smug", PipelineId: "pipe-smug", OwnerAddress: owner,
		Route: "from_scratch", Active: true,
	}))

	commitments := h.KnowledgeKeeper.GetAllNormativeCommitments(h.Ctx)
	require.NotEmpty(t, commitments)
	commitID := commitments[0].Id

	// Also seed a real fact the model legitimately used.
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-REAL", Content: "real", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: owner,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
	}))

	resp, err := ms.AttributeContributions(h.Ctx, &knowledgetypes.MsgAttributeContributions{
		Owner: owner, ModelId: "model-smug",
		FactIds: []string{"F-REAL", commitID, commitID, "F-REAL"}, // includes 2 smuggling attempts
	})
	require.NoError(t, err, "handler succeeds but filters out commitments")
	require.Equal(t, uint32(1), resp.Recorded,
		"only the real fact is recorded; 2 commitment attempts filtered")

	rec, _ := h.KnowledgeKeeper.GetContributionRecord(h.Ctx, "model-smug")
	require.NotNil(t, rec)
	require.NotContains(t, rec.FactIds, commitID,
		"commitment ID must not appear in recorded fact_ids")
	require.Equal(t, uint32(1), rec.RejectedCommitmentCount,
		"rejected count reflects ONE unique commitment, not two duplicates")
}

// ─── ATTACK 9: Clawback stickiness ───────────────────────────────────────
//
// Once a fact's revenue is clawed back (disproved), TVW stays 0 even if
// someone flips the status back to ACTIVE via keeper.SetFact. The
// revenue_clawback_block stamp is sticky.
func TestRouteB_Wave9_ClawbackStickiness(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	f := &knowledgetypes.Fact{
		Id: "F-STICKY", Content: "p", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: testAddr("wave9_stick").String(),
		MethodId: knowledgetypes.MethodologyEmpirical,
		Confidence: 900_000, CorroborationCount: 5,
		SubmitterCalibrationSnapshotBps: 800_000,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, f))

	// Baseline TVW positive.
	pre, err := qs.TrainingValueWeight(h.Ctx, &knowledgetypes.QueryTrainingValueWeightRequest{FactId: "F-STICKY"})
	require.NoError(t, err)
	require.Greater(t, pre.TvwBps, uint64(0))

	// Claw back (e.g. after disproval).
	require.NoError(t, h.KnowledgeKeeper.ClawbackOnDisproval(h.Ctx, "F-STICKY"))

	// Attacker (or well-meaning governance) flips status back to ACTIVE.
	recovered, _ := h.KnowledgeKeeper.GetFact(h.Ctx, "F-STICKY")
	recovered.Status = knowledgetypes.FactStatus_FACT_STATUS_ACTIVE
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, recovered))

	post, err := qs.TrainingValueWeight(h.Ctx, &knowledgetypes.QueryTrainingValueWeightRequest{FactId: "F-STICKY"})
	require.NoError(t, err)
	require.Equal(t, uint64(0), post.TvwBps,
		"clawback must stay sticky: revenue_clawback_block survives status flip")
	require.True(t, post.Disproven)
}

// ─── ATTACK 10: Sponsor cannot self-finalize via AcceptAugmentation ─────
//
// Wave 4 invariant: the ONLY bounty-acceptance path is a finalized passing
// verdict from the verifier panel. Sponsor calling AcceptAugmentation on
// a pre-verdict bounty augmentation must be rejected.
func TestRouteB_Wave9_SponsorSelfFinalizeBlocked(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsorAddr := testAddr("wave9_sponsor")
	sponsor := sponsorAddr.String()
	submitter := testAddr("wave9_sub").String()
	require.NoError(t, h.FundAccount(sponsorAddr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-SPON", Content: "p", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: sponsor,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
	}))

	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor, Id: "b-spon", TargetFactId: "F-SPON",
		RewardPerVariant: 1_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: submitter, Id: "aug-spon", BountyId: "b-spon",
		OriginalFactId: "F-SPON", VariantContent: "v",
	})
	require.NoError(t, err)

	// Sponsor tries to accept before any verdict.
	_, err = ms.AcceptAugmentation(h.Ctx, &knowledgetypes.MsgAcceptAugmentation{
		Acceptor: sponsor, AugmentationId: "aug-spon",
	})
	require.Error(t, err, "sponsor may not self-accept bounty variants under Wave 4")
	require.Contains(t, err.Error(), "verifier-panel")
}

// ─── ATTACK 11: Verifier Sybil on augmentation panel ────────────────────
//
// DEMONSTRATION (not a "hardening" test — exposes a KNOWN gap that Wave
// 10 stake-weighting will close). A single actor controlling three
// addresses can push a DRIFT verdict as EQUIVALENT. Because verifiers
// are not (yet) stake-weighted and consensus is vote-count based, this
// succeeds. The audit report documents this explicitly.
func TestRouteB_Wave9_VerifierSybilKnownGap(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsorAddr := testAddr("wave9_sybil_s")
	sponsor := sponsorAddr.String()
	submitter := testAddr("wave9_sybil_sub").String()
	require.NoError(t, h.FundAccount(sponsorAddr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-SYBIL", Content: "truth target", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: sponsor,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
	}))
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor, Id: "b-sybil", TargetFactId: "F-SYBIL",
		RewardPerVariant: 1_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: submitter, Id: "aug-sybil", BountyId: "b-sybil",
		OriginalFactId: "F-SYBIL", VariantContent: "a variant that actually drifts meaning",
	})
	require.NoError(t, err)

	// Three Sybil addresses all controlled by the same actor push EQUIVALENT.
	for _, v := range []string{"wave9_sybil_v1", "wave9_sybil_v2", "wave9_sybil_v3"} {
		_, err := ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
			Verifier: testAddr(v).String(), AugmentationId: "aug-sybil",
			Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
		})
		require.NoError(t, err, "verifier %s can vote: no Sybil defence yet", v)
	}

	aug, found := h.KnowledgeKeeper.GetAugmentation(h.Ctx, "aug-sybil")
	require.True(t, found)
	require.Equal(t, knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, aug.Verdict,
		"Sybil successfully pushed a verdict — documented gap, Wave 10 must stake-weight")
	require.True(t, aug.Accepted, "payout fires under Sybil — gap confirmed")
}

// ─── ATTACK 12: Heartbeat withstands many expiring bounties ─────────────
//
// Create N bounties that all expire in the same block; verify the heartbeat
// processes all of them without failing or stalling. Probes the
// unbounded-scan risk.
func TestRouteB_Wave9_HeartbeatScalesToManyBounties(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsorAddr := testAddr("wave9_mass_sponsor")
	sponsor := sponsorAddr.String()
	require.NoError(t, h.FundAccount(sponsorAddr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(10_000_000_000)))))

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-MASS", Content: "target", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: sponsor,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
	}))

	const N = 50 // chosen to stay under testing timeout; the unbounded scan is still a design concern
	expiryBlock := uint64(h.Height() + 3)
	for i := 0; i < N; i++ {
		_, err := ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
			Sponsor: sponsor, Id: fmt.Sprintf("b-mass-%d", i), TargetFactId: "F-MASS",
			RewardPerVariant: 100_000, MaxVariants: 1, ExpiresAtBlock: expiryBlock,
		})
		require.NoError(t, err)
	}

	h.AdvanceBlocks(5)

	// All N must be deactivated.
	activeCount := 0
	h.KnowledgeKeeper.IterateAugmentationBounties(h.Ctx, func(b *knowledgetypes.AugmentationBounty) bool {
		if b.Active {
			activeCount++
		}
		return false
	})
	require.Equal(t, 0, activeCount, "heartbeat processed all N bounties without stalling")
}

// ─── ATTACK 13: FilterIsOughtIds dedups within input ────────────────────
//
// The filter function should dedup duplicate IDs in the caller's input so
// the same ID can't inflate counts.
func TestRouteB_Wave9_IsOughtFilterDedup(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	facts, rejected := h.KnowledgeKeeper.FilterIsOughtIds(h.Ctx, []string{
		"A", "B", "A", "", "B", "C",
	})
	require.Equal(t, []string{"A", "B", "C"}, facts, "dedup preserves first-seen order")
	require.Empty(t, rejected)
}

// ─── ATTACK 14: TVW returns 0 for absent fact ───────────────────────────
//
// A TVW query for a non-existent fact must not panic and must return 0.
func TestRouteB_Wave9_TVWAbsentFact(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	resp, err := qs.TrainingValueWeight(h.Ctx, &knowledgetypes.QueryTrainingValueWeightRequest{
		FactId: "DOES-NOT-EXIST",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), resp.TvwBps)
	require.False(t, resp.Disproven)
	require.False(t, resp.BlockedIsOught)
}

// ─── ATTACK 15: Double-finalize rejection ───────────────────────────────
//
// Once FINALIZED, a manifest cannot be finalized again — the Merkle root
// must not change after the first commitment.
func TestRouteB_Wave9_DoubleFinalizeRejection(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave9_dblfin").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-dblfin", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-dblfin", PipelineId: "pipe-dblfin",
		CorpusSelector: &knowledgetypes.CorpusSelector{},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{Creator: operator, ManifestId: "m-dblfin"})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{Creator: operator, ManifestId: "m-dblfin"})
	require.Error(t, err, "double-finalize must reject; Merkle root is immutable")
	require.Contains(t, err.Error(), "not DRAFT")
}

// ─── ATTACK 16: Vote after verdict finalized ────────────────────────────
//
// Once the verifier panel reaches consensus, further votes must reject.
// The verdict is sealed.
func TestRouteB_Wave9_VoteAfterVerdictRejected(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsorAddr := testAddr("wave9_vaf_s")
	sponsor := sponsorAddr.String()
	submitter := testAddr("wave9_vaf_sub").String()
	require.NoError(t, h.FundAccount(sponsorAddr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-VAF", Content: "p", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: sponsor,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
	}))
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor, Id: "b-vaf", TargetFactId: "F-VAF",
		RewardPerVariant: 1_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: submitter, Id: "aug-vaf", BountyId: "b-vaf",
		OriginalFactId: "F-VAF", VariantContent: "v",
	})
	require.NoError(t, err)

	for _, v := range []string{"wave9_vaf_v1", "wave9_vaf_v2", "wave9_vaf_v3"} {
		_, _ = ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
			Verifier: testAddr(v).String(), AugmentationId: "aug-vaf",
			Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
		})
	}

	// Fourth vote must reject — verdict already final.
	_, err = ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
		Verifier: testAddr("wave9_vaf_v4").String(), AugmentationId: "aug-vaf",
		Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already final")
}

// ─── ATTACK 17: Bundle detects parent_merkle_root tampering ─────────────
//
// If a malicious caller directly SetTrainingManifest with a child whose
// parent_merkle_root has been swapped, the bundle's derived root MUST NOT
// match the stored merkle_root. The verifier catches the tamper.
func TestRouteB_Wave9_BundleDetectsParentRootTamper(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave9_tamper").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-tamp", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-TAMP", Content: "p", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "parent-tamp", PipelineId: "pipe-tamp",
		CorpusSelector: &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{Creator: operator, ManifestId: "parent-tamp"})
	require.NoError(t, err)

	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "child-tamp", PipelineId: "pipe-tamp",
		ParentManifestId: "parent-tamp",
		CorpusSelector:   &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{Creator: operator, ManifestId: "child-tamp"})
	require.NoError(t, err)

	// Tamper: overwrite the stored child's parent_merkle_root directly
	// (simulating a compromised node or historical corruption).
	child, _ := h.KnowledgeKeeper.GetTrainingManifest(h.Ctx, "child-tamp")
	child.ParentMerkleRoot = "deadbeef" + child.ParentMerkleRoot[8:]
	require.NoError(t, h.KnowledgeKeeper.SetTrainingManifest(h.Ctx, child))

	// Bundle MUST flag merkle_root_valid = false.
	bundle, err := qs.TrainingManifestBundle(h.Ctx, &knowledgetypes.QueryTrainingManifestBundleRequest{Id: "child-tamp"})
	require.NoError(t, err)
	require.False(t, bundle.MerkleRootValid,
		"tampered parent_merkle_root must be detected by re-derivation mismatch")
	require.NotEqual(t, bundle.DerivedMerkleRoot, bundle.Manifest.MerkleRoot)
}

// ─── ATTACK 18: Manifest selector limit-bypass ──────────────────────────
//
// A selector is a pure filter; large N is genuine workload not an attack
// per se, but verify pathological inputs don't crash. Empty selector
// against a seeded chain still returns zero facts (nothing qualifies
// without at least one filter value being meaningful).
func TestRouteB_Wave9_EmptySelectorIsSafe(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave9_empty").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-empty", OperatorAddress: operator, TokenizerVersion: 1,
	}))

	// No facts seeded at this point.
	resp, err := ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-empty", PipelineId: "pipe-empty",
		CorpusSelector: &knowledgetypes.CorpusSelector{}, // match all
	})
	require.NoError(t, err)
	require.Equal(t, uint32(0), resp.FactCount,
		"selector against an empty fact namespace produces an empty manifest (safe, not panicked)")

	// Finalize — empty manifest still gets a deterministic root.
	fin, err := ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "m-empty",
	})
	require.NoError(t, err)
	require.NotEmpty(t, fin.MerkleRoot, "empty manifest still commits to a well-formed root")
}

// ─── ATTACK 19: Manifest on missing pipeline ────────────────────────────
//
// Manifest create must verify the referenced pipeline exists.
func TestRouteB_Wave9_MissingPipelineRejected(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator:        testAddr("wave9_nopipe").String(),
		Id:             "m-nopipe",
		PipelineId:     "nonexistent-pipeline",
		CorpusSelector: &knowledgetypes.CorpusSelector{},
	})
	require.Error(t, err, "manifest for missing pipeline must reject")
	require.Contains(t, err.Error(), "not found")
}

// ─── ATTACK 20: Non-pipeline-operator cannot create manifest ────────────
//
// Only the pipeline operator may create a manifest for their pipeline.
// Otherwise any actor could stamp a manifest against another operator's
// pipeline and pollute the training registry.
func TestRouteB_Wave9_ManifestNonOperatorRejected(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	op := testAddr("wave9_mnop_op").String()
	intruder := testAddr("wave9_mnop_bad").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-mnop", OperatorAddress: op, TokenizerVersion: 1,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator:        intruder,
		Id:             "m-intrusion",
		PipelineId:     "pipe-mnop",
		CorpusSelector: &knowledgetypes.CorpusSelector{},
	})
	require.Error(t, err, "only pipeline operator may create the manifest")
	require.Contains(t, err.Error(), "operator")
}

// ─── ATTACK 21: Cross-pipeline parent manifest — feature or gap? ────────
//
// Pipeline A's manifest can be used as parent by Pipeline B's manifest.
// This is currently PERMITTED. Document whether this is intended (training
// lineages legitimately cross operators, e.g. distillation from a third-
// party SFT) or a gap that should tighten (attribution dilution risk).
// Currently we treat it as a feature — if it's abuse, the attestation
// audit trail reveals it.
func TestRouteB_Wave9_CrossPipelineParentPermitted(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	opA := testAddr("wave9_xp_a").String()
	opB := testAddr("wave9_xp_b").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-xp-a", OperatorAddress: opA, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-xp-b", OperatorAddress: opB, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-XP", Content: "shared", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: opA,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
	}))

	// A creates + finalizes.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: opA, Id: "m-xp-a", PipelineId: "pipe-xp-a",
		CorpusSelector: &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{Creator: opA, ManifestId: "m-xp-a"})
	require.NoError(t, err)

	// B inherits from A's manifest — currently permitted.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: opB, Id: "m-xp-b", PipelineId: "pipe-xp-b",
		ParentManifestId: "m-xp-a",
		CorpusSelector:   &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err,
		"cross-pipeline parent is currently allowed — audit item in ROUTE_B_SECURITY_AUDIT.md")
}

// ─── ATTACK 22: Double-clawback idempotency ─────────────────────────────
//
// ClawbackOnDisproval called twice on the same fact must be a no-op on
// the second call. The revenue_clawback_block stamp is sticky; the second
// call returns cleanly without re-emitting the clawback event or resetting
// state.
func TestRouteB_Wave9_DoubleClawbackIdempotent(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	f := &knowledgetypes.Fact{
		Id: "F-DBC", Content: "p", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: testAddr("wave9_dbc").String(),
		MethodId: knowledgetypes.MethodologyEmpirical,
		Confidence: 900_000, CorroborationCount: 3,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, f))

	require.NoError(t, h.KnowledgeKeeper.ClawbackOnDisproval(h.Ctx, "F-DBC"))
	first, _ := h.KnowledgeKeeper.GetFact(h.Ctx, "F-DBC")
	firstBlock := first.RevenueClawbackBlock
	require.Greater(t, firstBlock, uint64(0))

	// Second call — must not reset or overwrite.
	require.NoError(t, h.KnowledgeKeeper.ClawbackOnDisproval(h.Ctx, "F-DBC"))
	second, _ := h.KnowledgeKeeper.GetFact(h.Ctx, "F-DBC")
	require.Equal(t, firstBlock, second.RevenueClawbackBlock,
		"double-clawback is idempotent: block stamp unchanged")
}

// ─── ATTACK 23: Merkle root with empty ID sets ──────────────────────────
//
// An empty manifest (all sets empty) produces a deterministic root — not
// a random value, not a panic.
func TestRouteB_Wave9_EmptyManifestMerkle(t *testing.T) {
	empty := knowledgekeeper.SelectedManifestIDs{}
	r1 := knowledgekeeper.ComputeManifestMerkleRoot(empty)
	r2 := knowledgekeeper.ComputeManifestMerkleRoot(empty)
	require.Equal(t, r1, r2, "empty root is deterministic")
	require.NotEmpty(t, r1, "empty root is non-empty hex")

	// Composed empty root must also be well-formed.
	c := knowledgekeeper.ComputeComposedManifestMerkleRoot(r1, empty)
	require.NotEqual(t, c, r1, "empty composed ≠ empty flat (domain tag differs)")
}
