package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestRouteB_TokenizerSpecSeeded asserts the bootstrap tokenizer spec is
// available and carries the expected structural tokens.
func TestRouteB_TokenizerSpecSeeded(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	resp, err := qs.TokenizerSpec(h.Ctx, &knowledgetypes.QueryTokenizerSpecRequest{})
	require.NoError(t, err)
	require.True(t, resp.Found)
	require.Equal(t, uint64(1), resp.Spec.Version)
	require.Equal(t, "<method:", resp.Spec.MethodTokenPrefix)
	require.Equal(t, "<fact>", resp.Spec.FactBeginToken)
	require.Equal(t, "<disproved/>", resp.Spec.DisproofMarkerToken)
	require.Equal(t, uint64(1), resp.Spec.CanonicalSerialisationVersion)

	// Historical fetch by version returns the same spec.
	hist, err := qs.TokenizerSpecAtVersion(h.Ctx, &knowledgetypes.QueryTokenizerSpecAtVersionRequest{Version: 1})
	require.NoError(t, err)
	require.True(t, hist.Found)
	require.Equal(t, resp.Spec.MethodTokenPrefix, hist.Spec.MethodTokenPrefix)
}

// TestRouteB_PipelineAndModelCardLineage exercises the full registration
// loop: register a TrainingPipeline → register a ModelCard pointing at it
// → query by deployment address correlates back to the card → the card's
// deployment_address (treated as the model's agent account) has a
// calibration record after submitting claims.
func TestRouteB_PipelineAndModelCardLineage(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))

	// 1. Declare a training pipeline.
	pipeline := &knowledgetypes.TrainingPipeline{
		Id:                    "pipeline-alpha-v1",
		OperatorAddress:       "zerone1operator00000000000000000000aa",
		CorpusSnapshotHeight:  1,
		TokenizerVersion:      1,
		MethodologySetVersion: 1,
		RecipeHash:            "sha256:abcdef",
		Description:           "bootstrap SFT run on GOLD-tier facts",
		Status:                "declared",
		DeclaredAtBlock:       1,
		CorpusFilter:          `{"min_tier":"GOLD","methods":["M-FORMAL","M-EMPIRICAL"]}`,
	}
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, pipeline))

	// 2. Register a ModelCard produced by that pipeline. deployment_address
	//    is the agent account the model runs as — calibration accrues there.
	deploymentAddr := "zerone1deployedmodelagent0000000000aa"
	card := &knowledgetypes.ModelCard{
		Id:                        "model-alpha-v1",
		Name:                      "Alpha-SFT-v1",
		PipelineId:                pipeline.Id,
		DeploymentAddress:         deploymentAddr,
		CreatedAtBlock:            1,
		ParameterCount:            7,
		Route:                     "openweight_fine_tune",
		BaseModel:                 "llama-3-8b-base",
		OwnerAddress:              "zerone1modelowner00000000000000000000",
		EvalAcceptanceRateBps:     780_000,
		EvalCorroborationRateBps:  450_000,
		EvalSampleSize:            1000,
		Active:                    true,
	}
	require.NoError(t, h.KnowledgeKeeper.SetModelCard(h.Ctx, card))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	// Pipeline retrievable.
	pResp, err := qs.TrainingPipeline(h.Ctx, &knowledgetypes.QueryTrainingPipelineRequest{Id: pipeline.Id})
	require.NoError(t, err)
	require.True(t, pResp.Found)
	require.Equal(t, pipeline.RecipeHash, pResp.Pipeline.RecipeHash)

	// Card retrievable by id.
	cResp, err := qs.ModelCard(h.Ctx, &knowledgetypes.QueryModelCardRequest{Id: card.Id})
	require.NoError(t, err)
	require.True(t, cResp.Found)
	require.Equal(t, pipeline.Id, cResp.Card.PipelineId)

	// Card retrievable by deployment address — closes the link from an
	// agent account back to the model it embodies.
	byDeploy, err := qs.ModelCardByDeployment(h.Ctx, &knowledgetypes.QueryModelCardByDeploymentRequest{
		Address: deploymentAddr,
	})
	require.NoError(t, err)
	require.True(t, byDeploy.Found)
	require.Equal(t, card.Id, byDeploy.Card.Id)

	// Cards listing with pipeline filter.
	list, err := qs.ModelCards(h.Ctx, &knowledgetypes.QueryModelCardsRequest{
		PipelineId: pipeline.Id,
		ActiveOnly: true,
	})
	require.NoError(t, err)
	require.Len(t, list.Cards, 1)

	// 3. Simulate the deployed agent submitting a claim. Its calibration
	//    should then correlate back to the ModelCard.
	domain := "route_b_lineage_domain"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   domain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	claim := &knowledgetypes.Claim{
		Id:          "alpha-model-claim-1",
		Submitter:   deploymentAddr,
		FactContent: "claim from the deployed model's agent account",
		Domain:      domain,
		Category:    "empirical",
		Status:      knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Stake:       "1000000",
		MethodId:    knowledgetypes.MethodologyEmpirical,
	}
	require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, claim))
	round := &knowledgetypes.VerificationRound{
		Id:             "round-alpha",
		ClaimId:        claim.Id,
		Phase:          knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
		StartedAtBlock: 1,
	}
	require.NoError(t, h.KnowledgeKeeper.CompleteRound(h.Ctx, round, &knowledgekeeper.VerificationResult{
		Verdict:     knowledgetypes.Verdict_VERDICT_ACCEPT,
		Confidence:  900_000,
		AcceptCount: 3,
	}))

	calResp, err := qs.AgentCalibration(h.Ctx, &knowledgetypes.QueryAgentCalibrationRequest{
		Address: deploymentAddr,
	})
	require.NoError(t, err)
	require.True(t, calResp.Found,
		"deployed model's agent account must accrue calibration on its first accepted submission")
	require.Equal(t, uint64(1), calResp.Calibration.Accepted)

	// End-to-end lineage assertion: model ↔ pipeline ↔ agent ↔ calibration
	modelViaAgent, err := qs.ModelCardByDeployment(h.Ctx, &knowledgetypes.QueryModelCardByDeploymentRequest{
		Address: deploymentAddr,
	})
	require.NoError(t, err)
	require.True(t, modelViaAgent.Found)
	require.Equal(t, card.Id, modelViaAgent.Card.Id)
	require.Equal(t, pipeline.Id, modelViaAgent.Card.PipelineId)
}

// TestRouteB_StructuredCorpusExport verifies the canonical training-row
// export carries every piece of information a pipeline needs: content,
// method, curriculum tier, support edges, reasoning trace, submitter
// calibration score, and — if requested — disproven negative examples.
func TestRouteB_StructuredCorpusExport(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))

	domain := "structured_corpus_domain"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   domain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	submitter := "zerone1corpussubmitter0000000000000aa"

	// Submit a calibration-bearing fact so the denormalised score populates.
	claim := &knowledgetypes.Claim{
		Id:             "corpus-seed-claim",
		Submitter:      submitter,
		FactContent:    "A gold-tier claim with a reasoning trace.",
		Domain:         domain,
		Category:       "empirical",
		Status:         knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Stake:          "1000000",
		MethodId:       knowledgetypes.MethodologyEmpirical,
		ReasoningTrace: `[{"step":1,"observation":"..."}]`,
	}
	require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, claim))
	round := &knowledgetypes.VerificationRound{
		Id: "round-corpus-seed", ClaimId: claim.Id,
		Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, StartedAtBlock: 1,
	}
	require.NoError(t, h.KnowledgeKeeper.CompleteRound(h.Ctx, round, &knowledgekeeper.VerificationResult{
		Verdict: knowledgetypes.Verdict_VERDICT_ACCEPT, Confidence: 900_000, AcceptCount: 3,
	}))

	// Grab the created fact and promote it to GOLD tier by bumping corroboration.
	var createdFact *knowledgetypes.Fact
	h.KnowledgeKeeper.IterateFactsByDomain(h.Ctx, domain, func(factID string) bool {
		f, _ := h.KnowledgeKeeper.GetFact(h.Ctx, factID)
		if f != nil && f.ClaimId == claim.Id {
			createdFact = f
			return true
		}
		return false
	})
	require.NotNil(t, createdFact)
	createdFact.CorroborationCount = 5 // promote to GOLD
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, createdFact))

	// Seed a disproven fact for the negative-examples path.
	disproven := &knowledgetypes.Fact{
		Id:         "CORPUS-DISPROVEN",
		Content:    "A claim that was disproven.",
		Domain:     domain,
		Category:   "empirical",
		Confidence: 500_000,
		Status:     knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN,
		Submitter:  submitter,
		MethodId:   knowledgetypes.MethodologyEmpirical,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, disproven))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	// Positive-only export: no negatives.
	pos, err := qs.StructuredCorpus(h.Ctx, &knowledgetypes.QueryStructuredCorpusRequest{
		MinTier:          knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD,
		IncludeDisproven: false,
	})
	require.NoError(t, err)
	require.Len(t, pos.Entries, 1)
	require.Equal(t, createdFact.Id, pos.Entries[0].FactId)
	require.Equal(t, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD, pos.Entries[0].Tier)
	require.NotEmpty(t, pos.Entries[0].ReasoningTrace,
		"reasoning trace must flow through to structured corpus entry")
	require.Greater(t, pos.Entries[0].SubmitterCalibrationScoreBps, uint64(0),
		"structured corpus must denormalise submitter calibration for training weighting")
	require.Equal(t, uint64(1), pos.TokenizerVersion)
	require.Equal(t, uint64(1), pos.CanonicalSerialisationVersion)
	require.Greater(t, pos.SnapshotBlockHeight, uint64(0))

	// Negative-inclusive export: disproven appears as contrastive.
	both, err := qs.StructuredCorpus(h.Ctx, &knowledgetypes.QueryStructuredCorpusRequest{
		MinTier:          knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD,
		IncludeDisproven: true,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(both.Entries), 2)
	// Find the negative entry.
	var sawNegative bool
	for _, e := range both.Entries {
		if e.IsNegativeExample {
			sawNegative = true
			require.Equal(t, disproven.Id, e.FactId)
			require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, e.Status)
		}
	}
	require.True(t, sawNegative, "disproven fact must appear with is_negative_example=true")
}

// TestRouteB_CurriculumTierOrdering confirms the curriculum tiers place
// foundational facts first and specialised methodologies last.
func TestRouteB_CurriculumTierOrdering(t *testing.T) {
	foundation := &knowledgetypes.Fact{AxiomDistance: 0, CorroborationCount: 5, MethodId: knowledgetypes.MethodologyFormal}
	intermediate := &knowledgetypes.Fact{AxiomDistance: 2, CorroborationCount: 1, MethodId: knowledgetypes.MethodologyEmpirical}
	advanced := &knowledgetypes.Fact{AxiomDistance: 6, CorroborationCount: 0, MethodId: knowledgetypes.MethodologyEmpirical}
	specialised := &knowledgetypes.Fact{AxiomDistance: 1, CorroborationCount: 0, MethodId: knowledgetypes.MethodologyPhenomenologic}

	require.Equal(t, knowledgetypes.CurriculumTier_CURRICULUM_TIER_FOUNDATION, knowledgekeeper.ClassifyCurriculumTier(foundation))
	require.Equal(t, knowledgetypes.CurriculumTier_CURRICULUM_TIER_INTERMEDIATE, knowledgekeeper.ClassifyCurriculumTier(intermediate))
	require.Equal(t, knowledgetypes.CurriculumTier_CURRICULUM_TIER_ADVANCED, knowledgekeeper.ClassifyCurriculumTier(advanced))
	require.Equal(t, knowledgetypes.CurriculumTier_CURRICULUM_TIER_SPECIALISED, knowledgekeeper.ClassifyCurriculumTier(specialised),
		"phenomenological/ecological/practice/analogical methodologies route to SPECIALISED regardless of distance")
}
