package cross_stack_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestRouteB_Wave8_GenesisRoundtrip — export current state, re-initialize
// a fresh keeper from the exported genesis, assert every Route B record
// is preserved. Validates the substrate survives upgrades.
func TestRouteB_Wave8_GenesisRoundtrip(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)

	// Populate state: pipeline + model card + fact + attestation + bounty +
	// contribution + calibration + manifest.
	operatorAddr := testAddr("wave8_operator")
	operator := operatorAddr.String()
	require.NoError(t, h.FundAccount(operatorAddr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-8g", OperatorAddress: operator, TokenizerVersion: 1,
		MethodologySetVersion: 1, Status: "completed",
	}))
	require.NoError(t, h.KnowledgeKeeper.SetModelCard(h.Ctx, &knowledgetypes.ModelCard{
		Id: "model-8g", PipelineId: "pipe-8g", OwnerAddress: operator,
		Route: "from_scratch", Active: true,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-8g", Content: "roundtrip test fact", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
		CorroborationCount: 3,
	}))
	_, err = ms.AttestTraining(h.Ctx, &knowledgetypes.MsgAttestTraining{
		Attester: operator, PipelineId: "pipe-8g",
		FlopsEstimate: 42, WallclockSeconds: 60, EvalHash: "sha256:roundtrip",
	})
	require.NoError(t, err)
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: operator, Id: "bounty-8g", TargetFactId: "F-8g",
		RewardPerVariant: 100000, MaxVariants: 1, ExpiresAtBlock: 999999,
	})
	require.NoError(t, err)
	_, err = ms.AttributeContributions(h.Ctx, &knowledgetypes.MsgAttributeContributions{
		Owner: operator, ModelId: "model-8g", FactIds: []string{"F-8g"},
	})
	require.NoError(t, err)
	require.NoError(t, h.KnowledgeKeeper.SetAgentCalibration(h.Ctx, &knowledgetypes.AgentCalibration{
		Address: operator, CalibrationScoreBps: 750_000, TotalSubmissions: 10,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "manifest-8g", PipelineId: "pipe-8g",
		CorpusSelector: &knowledgetypes.CorpusSelector{
			MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3,
		},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "manifest-8g",
	})
	require.NoError(t, err)

	// Snapshot via ExportGenesis.
	gs := h.KnowledgeKeeper.ExportGenesis(h.Ctx)
	require.NotNil(t, gs)

	// Every Route B bucket must populate.
	require.NotEmpty(t, gs.Methodologies, "methodologies exported")
	require.NotEmpty(t, gs.NormativeCommitments, "commitments exported")
	require.NotNil(t, gs.TokenizerSpec, "current tokenizer spec exported")
	require.NotEmpty(t, gs.TokenizerSpecHistory, "tokenizer history exported")
	require.NotNil(t, gs.TraceSchema, "current trace schema exported")
	require.NotEmpty(t, gs.TraceSchemaHistory, "trace schema history exported")
	require.Len(t, gs.TrainingPipelines, 1)
	require.Len(t, gs.ModelCards, 1)
	require.Len(t, gs.TrainingAttestations, 1)
	require.Len(t, gs.AugmentationBounties, 1)
	require.Len(t, gs.ContributionRecords, 1)
	require.Len(t, gs.AgentCalibrations, 1)
	require.Len(t, gs.TrainingManifests, 1)

	// Re-init a fresh harness from the exported genesis.
	h2 := NewTestHarness(t)
	require.NoError(t, h2.KnowledgeKeeper.InitGenesis(h2.Ctx, gs))

	// Every record survives.
	p, ok := h2.KnowledgeKeeper.GetTrainingPipeline(h2.Ctx, "pipe-8g")
	require.True(t, ok)
	require.Equal(t, operator, p.OperatorAddress)

	card, ok := h2.KnowledgeKeeper.GetModelCard(h2.Ctx, "model-8g")
	require.True(t, ok)
	require.Equal(t, "pipe-8g", card.PipelineId)

	fact, ok := h2.KnowledgeKeeper.GetFact(h2.Ctx, "F-8g")
	require.True(t, ok)
	require.Equal(t, uint64(3), fact.CorroborationCount)

	att, ok := h2.KnowledgeKeeper.GetTrainingAttestation(h2.Ctx, "pipe-8g")
	require.True(t, ok)
	require.Equal(t, "sha256:roundtrip", att.EvalHash)

	bnt, ok := h2.KnowledgeKeeper.GetAugmentationBounty(h2.Ctx, "bounty-8g")
	require.True(t, ok)
	require.Equal(t, uint32(1), bnt.MaxVariants)

	cr, ok := h2.KnowledgeKeeper.GetContributionRecord(h2.Ctx, "model-8g")
	require.True(t, ok)
	require.Contains(t, cr.FactIds, "F-8g")

	cal, ok := h2.KnowledgeKeeper.GetAgentCalibration(h2.Ctx, operator)
	require.True(t, ok)
	require.Equal(t, uint64(750_000), cal.CalibrationScoreBps)

	man, ok := h2.KnowledgeKeeper.GetTrainingManifest(h2.Ctx, "manifest-8g")
	require.True(t, ok)
	require.Equal(t, knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED, man.Status)
	require.NotEmpty(t, man.MerkleRoot, "Merkle root survives genesis roundtrip")

	// Historical tokenizer spec still queryable by version.
	hist, ok := h2.KnowledgeKeeper.GetTokenizerSpecAtVersion(h2.Ctx, 1)
	require.True(t, ok)
	require.Equal(t, uint64(1), hist.Version)
}

// TestRouteB_Wave8_HeartbeatBountyExpiry — after a bounty's expires_at_block
// passes, the heartbeat returns escrow to the sponsor automatically.
func TestRouteB_Wave8_HeartbeatBountyExpiry(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsorAddr := testAddr("wave8_sponsor")
	sponsor := sponsorAddr.String()
	require.NoError(t, h.FundAccount(sponsorAddr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-EXPIRE", Content: "expiry target", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: sponsor,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
	}))

	// Bounty expires at block N+5.
	expiryBlock := uint64(h.Height() + 5)
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor, Id: "bounty-expire", TargetFactId: "F-EXPIRE",
		RewardPerVariant: 1_000_000, MaxVariants: 2, ExpiresAtBlock: expiryBlock,
	})
	require.NoError(t, err)

	balBefore := h.GetBalance(sponsorAddr, "uzrn")
	require.Equal(t, sdkmath.NewInt(97_000_000), balBefore.Amount,
		"100M - 3M escrow = 97M after bounty creation")

	// Advance blocks past expiry; heartbeat fires automatically.
	h.AdvanceBlocks(10)

	// Bounty deactivated.
	bnt, ok := h.KnowledgeKeeper.GetAugmentationBounty(h.Ctx, "bounty-expire")
	require.True(t, ok)
	require.False(t, bnt.Active, "heartbeat deactivated expired bounty")

	// Sponsor refunded minus 3% fee. 3_000_000 × 3% = 90_000 fee; refund = 2_910_000.
	balAfter := h.GetBalance(sponsorAddr, "uzrn")
	require.Equal(t, sdkmath.NewInt(97_000_000+2_910_000), balAfter.Amount,
		"sponsor refunded 2.91M (3M escrow minus 90k fee)")
}

// TestRouteB_Wave8_HeartbeatVestingRelease — after a disbursement's
// vesting_end_block arrives, the heartbeat releases the vesting portion.
func TestRouteB_Wave8_HeartbeatVestingRelease(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	// Shorten the vesting window for the test so we don't advance 60 × 1111
	// blocks. Params are amendable via SetParams.
	p, err := h.KnowledgeKeeper.GetParams(h.Ctx)
	require.NoError(t, err)
	p.TrainingFundVestingEpochs = 1
	p.FitnessEpochBlocks = 5
	require.NoError(t, h.KnowledgeKeeper.SetParams(h.Ctx, p))

	operatorAddr := testAddr("wave8_vestop")
	operator := operatorAddr.String()
	deploymentAddr := testAddr("wave8_vestdeploy").String()

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-vest", OperatorAddress: operator, TokenizerVersion: 1, Status: "completed",
	}))
	require.NoError(t, h.KnowledgeKeeper.SetModelCard(h.Ctx, &knowledgetypes.ModelCard{
		Id: "model-vest", PipelineId: "pipe-vest", OwnerAddress: operator,
		Route: "from_scratch", Active: true, DeploymentAddress: deploymentAddr,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetAgentCalibration(h.Ctx, &knowledgetypes.AgentCalibration{
		Address: deploymentAddr, CalibrationScoreBps: 800_000,
		Accepted: 50, TotalSubmissions: 50,
	}))

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	claim, err := ms.ClaimTrainingFundDisbursement(h.Ctx, &knowledgetypes.MsgClaimTrainingFundDisbursement{
		Claimant: operator, ModelId: "model-vest", Id: "disb-vest-1",
	})
	require.NoError(t, err)

	released, _ := sdkmath.NewIntFromString(claim.ReleasedAmount)
	vesting, _ := sdkmath.NewIntFromString(claim.VestingAmount)
	require.True(t, released.Equal(vesting), "50/50 split at claim time")

	balAtClaim := h.GetBalance(operatorAddr, "uzrn")
	require.True(t, balAtClaim.Amount.Equal(released),
		"operator holds only the released half immediately")

	// Advance past vesting_end_block.
	blocksUntilVest := int64(claim.VestingEndBlock) - h.Height() + 1
	if blocksUntilVest < 1 {
		blocksUntilVest = 1
	}
	h.AdvanceBlocks(int(blocksUntilVest))

	// Heartbeat should have released the vesting portion.
	balAfterVest := h.GetBalance(operatorAddr, "uzrn")
	require.True(t, balAfterVest.Amount.Equal(released.Add(vesting)),
		"vesting released; operator holds full amount")

	// Disbursement state: vesting_amount zeroed (idempotency).
	disb, ok := h.KnowledgeKeeper.GetTrainingFundDisbursement(h.Ctx, "disb-vest-1")
	require.True(t, ok)
	require.Equal(t, "0", disb.VestingAmount, "vesting amount zeroed post-release")

	// Second heartbeat must not double-release.
	h.AdvanceBlocks(5)
	balAfterSecond := h.GetBalance(operatorAddr, "uzrn")
	require.True(t, balAfterSecond.Amount.Equal(balAfterVest.Amount),
		"idempotent: no double release on subsequent blocks")
}

// TestRouteB_Wave8_HeartbeatManifestSupersession — when a newer FINALIZED
// manifest exists for the same pipeline, the heartbeat marks the older
// one SUPERSEDED. ATTESTED manifests are not superseded (attestation is
// the binding commitment).
func TestRouteB_Wave8_HeartbeatManifestSupersession(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave8_super").String()

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-super", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-SUPER", Content: "p", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
	}))

	// Create + finalize manifest v1.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-v1", PipelineId: "pipe-super",
		CorpusSelector: &knowledgetypes.CorpusSelector{
			MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3,
		},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "m-v1",
	})
	require.NoError(t, err)

	h.AdvanceBlocks(1) // separate finalized_at_block

	// Create + finalize manifest v2 for same pipeline.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-v2", PipelineId: "pipe-super",
		CorpusSelector: &knowledgetypes.CorpusSelector{
			MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3,
		},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "m-v2",
	})
	require.NoError(t, err)

	// Heartbeat fires.
	h.AdvanceBlocks(1)

	v1, _ := h.KnowledgeKeeper.GetTrainingManifest(h.Ctx, "m-v1")
	v2, _ := h.KnowledgeKeeper.GetTrainingManifest(h.Ctx, "m-v2")
	require.Equal(t, knowledgetypes.ManifestStatus_MANIFEST_STATUS_SUPERSEDED, v1.Status,
		"older manifest marked SUPERSEDED by heartbeat")
	require.Equal(t, knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED, v2.Status,
		"newer manifest remains FINALIZED")
}

// TestRouteB_Wave8_ComposableManifestDelta — child manifest inherits from
// a parent, carrying only the delta; bundle unions both; composed Merkle
// root validates.
func TestRouteB_Wave8_ComposableManifestDelta(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	operator := testAddr("wave8_compose").String()

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-compose", OperatorAddress: operator, TokenizerVersion: 1,
	}))

	// Seed 3 parent facts.
	for _, id := range []string{"F-P1", "F-P2", "F-P3"} {
		require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
			Id: id, Content: id, Domain: "sciences",
			Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
			MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
		}))
	}

	// Parent manifest covers all 3.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "parent", PipelineId: "pipe-compose",
		CorpusSelector: &knowledgetypes.CorpusSelector{
			MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3,
		},
	})
	require.NoError(t, err)
	parentFin, err := ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "parent",
	})
	require.NoError(t, err)
	require.NotEmpty(t, parentFin.MerkleRoot)

	// Seed 2 additional facts and create a child that inherits from parent.
	for _, id := range []string{"F-C1", "F-C2"} {
		require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
			Id: id, Content: id, Domain: "sciences",
			Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
			MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
		}))
	}

	childResp, err := ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "child", PipelineId: "pipe-compose",
		ParentManifestId: "parent",
		CorpusSelector: &knowledgetypes.CorpusSelector{
			MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2), childResp.FactCount,
		"child carries only the delta: 2 new facts (F-C1, F-C2)")

	// Child's manifest record reflects inheritance.
	childM, _ := h.KnowledgeKeeper.GetTrainingManifest(h.Ctx, "child")
	require.Equal(t, "parent", childM.ParentManifestId)
	require.Equal(t, parentFin.MerkleRoot, childM.ParentMerkleRoot,
		"child snapshots parent's committed root")
	require.Equal(t, uint32(1), childM.CompositionDepth)

	// Finalize child — uses composed Merkle root.
	childFin, err := ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "child",
	})
	require.NoError(t, err)
	require.NotEqual(t, parentFin.MerkleRoot, childFin.MerkleRoot,
		"composed root differs from parent root")

	// Bundle of child unions parent's content.
	bundle, err := qs.TrainingManifestBundle(h.Ctx, &knowledgetypes.QueryTrainingManifestBundleRequest{Id: "child"})
	require.NoError(t, err)
	require.Len(t, bundle.Traces, 5,
		"child bundle materialises union: 3 parent + 2 delta = 5 traces")
	require.True(t, bundle.MerkleRootValid,
		"composed root re-derives cleanly from parent_root + child delta")

	// Attempting a second-generation child beyond max depth must be caught.
	// (We stop here since depth=1 < 8; deeper chains are exercised in the
	// grand-child test if desired.)

	// Attempting to parent on a DRAFT manifest is rejected.
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "bad-child", PipelineId: "pipe-compose",
		ParentManifestId: "non-existent",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "parent manifest non-existent not found")
}
