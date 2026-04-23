package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestTrainingPipeline_QualityTierClassification pins the tier semantics.
// GOLD requires non-legacy method AND corroboration ≥ 3.
// SILVER requires non-legacy method AND corroboration ≥ 1.
// BRONZE is every other accepted claim.
// NEGATIVE is DISPROVEN — valuable training signal, segmented.
// UNSUITABLE is CONTESTED / EXPIRED / PRUNED / SUPERSEDED / REVOKED.
func TestTrainingPipeline_QualityTierClassification(t *testing.T) {
	cases := []struct {
		name           string
		method         string
		status         knowledgetypes.FactStatus
		corroborations uint64
		expected       knowledgetypes.TrainingQualityTier
	}{
		{"gold-empirical", knowledgetypes.MethodologyEmpirical, knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, 5, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD},
		{"silver-formal", knowledgetypes.MethodologyFormal, knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, 1, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_SILVER},
		{"bronze-zero-corr", knowledgetypes.MethodologyHistorical, knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, 0, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_BRONZE},
		{"bronze-legacy-even-with-corr", knowledgetypes.MethodologyLegacy, knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, 10, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_BRONZE},
		{"negative-disproven", knowledgetypes.MethodologyEmpirical, knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, 0, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_NEGATIVE},
		{"unsuitable-contested", knowledgetypes.MethodologyEmpirical, knowledgetypes.FactStatus_FACT_STATUS_CONTESTED, 5, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSUITABLE},
		{"unsuitable-expired", knowledgetypes.MethodologyFormal, knowledgetypes.FactStatus_FACT_STATUS_EXPIRED, 2, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSUITABLE},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fact := &knowledgetypes.Fact{
				Id:                 "f-" + c.name,
				Content:            "test",
				Status:             c.status,
				MethodId:           c.method,
				CorroborationCount: c.corroborations,
			}
			got, reason := knowledgekeeper.ClassifyTrainingQuality(fact)
			require.Equal(t, c.expected, got, "reason: %s", reason)
		})
	}
}

// TestTrainingPipeline_CorpusExports asserts the three export endpoints
// segment facts correctly and carry snapshot_block_height for reproducibility.
func TestTrainingPipeline_CorpusExports(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))

	domain := "training_corpus_domain"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   domain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	// Seed three facts across the tiers.
	gold := &knowledgetypes.Fact{
		Id:                 "TRAIN-GOLD",
		Content:            "Gold-tier empirical fact.",
		Domain:             domain,
		Category:           "empirical",
		Confidence:         900_000,
		Status:             knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		Submitter:          "t",
		MethodId:           knowledgetypes.MethodologyEmpirical,
		CorroborationCount: 5,
		ReasoningTrace:     `[{"step":1,"rule":"observation","content":"..."}]`,
	}
	silver := &knowledgetypes.Fact{
		Id:                 "TRAIN-SILVER",
		Content:            "Silver-tier formal fact.",
		Domain:             domain,
		Category:           "formal",
		Confidence:         850_000,
		Status:             knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		Submitter:          "t",
		MethodId:           knowledgetypes.MethodologyFormal,
		CorroborationCount: 1,
	}
	disproven := &knowledgetypes.Fact{
		Id:         "TRAIN-DISPROVEN",
		Content:    "A claim that was disproven.",
		Domain:     domain,
		Category:   "empirical",
		Confidence: 600_000,
		Status:     knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN,
		Submitter:  "t",
		MethodId:   knowledgetypes.MethodologyEmpirical,
	}
	for _, f := range []*knowledgetypes.Fact{gold, silver, disproven} {
		require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, f))
	}

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	// ─── MethodCorpus: GOLD floor should return only the gold fact ─────
	goldOnly, err := qs.MethodCorpus(h.Ctx, &knowledgetypes.QueryMethodCorpusRequest{
		MinTier: knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD,
	})
	require.NoError(t, err)
	require.Len(t, goldOnly.Entries, 1)
	require.Equal(t, gold.Id, goldOnly.Entries[0].Fact.Id)
	require.Equal(t, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD,
		goldOnly.Entries[0].Tier)
	require.Greater(t, goldOnly.SnapshotBlockHeight, uint64(0),
		"exports must pin a snapshot block height for reproducibility")

	// ─── MethodCorpus: SILVER floor returns both gold and silver ──────
	silverOrBetter, err := qs.MethodCorpus(h.Ctx, &knowledgetypes.QueryMethodCorpusRequest{
		MinTier: knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_SILVER,
	})
	require.NoError(t, err)
	require.Len(t, silverOrBetter.Entries, 2)

	// ─── DisprovenCorpus: only the disproven fact ──────────────────────
	disp, err := qs.DisprovenCorpus(h.Ctx, &knowledgetypes.QueryDisprovenCorpusRequest{})
	require.NoError(t, err)
	require.Len(t, disp.Entries, 1)
	require.Equal(t, disproven.Id, disp.Entries[0].DisprovenFact.Id)
	require.Greater(t, disp.SnapshotBlockHeight, uint64(0))

	// ─── TrainingQuality: single-fact tier fetch ───────────────────────
	qual, err := qs.TrainingQuality(h.Ctx, &knowledgetypes.QueryTrainingQualityRequest{
		FactId: gold.Id,
	})
	require.NoError(t, err)
	require.Equal(t, knowledgetypes.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD, qual.Tier)
	require.Equal(t, uint64(5), qual.CorroborationCount)
	require.Equal(t, knowledgetypes.MethodologyEmpirical, qual.MethodId)
	require.NotEmpty(t, qual.Reason)

	// ─── Reasoning trace round-trip ────────────────────────────────────
	// Pulled through MethodCorpus, the fact carries its reasoning trace.
	for _, e := range goldOnly.Entries {
		if e.Fact.Id == gold.Id {
			require.Equal(t, gold.ReasoningTrace, e.Fact.ReasoningTrace,
				"reasoning_trace must be carried in the exported fact")
		}
	}
}

// TestTrainingPipeline_ReasoningTracePropagates checks the claim→fact
// flow: a submitter attaches a reasoning_trace to a claim; once accepted,
// the resulting fact carries it.
func TestTrainingPipeline_ReasoningTracePropagates(t *testing.T) {
	h := NewTestHarness(t)

	domain := "reasoning_trace_domain"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   domain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	claim := &knowledgetypes.Claim{
		Id:             "reasoning-claim",
		Submitter:      "t",
		FactContent:    "Claim with an attached reasoning trace.",
		Domain:         domain,
		Category:       "formal",
		Status:         knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Stake:          "1000000",
		MethodId:       knowledgetypes.MethodologyFormal,
		ReasoningTrace: `[{"step":1,"rule":"modus-ponens","from":["A","A→B"],"to":"B"}]`,
	}
	require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, claim))

	round := &knowledgetypes.VerificationRound{
		Id:             "round-reasoning",
		ClaimId:        claim.Id,
		Phase:          knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
		StartedAtBlock: 1,
	}
	result := &knowledgekeeper.VerificationResult{
		Verdict:     knowledgetypes.Verdict_VERDICT_ACCEPT,
		Confidence:  900_000,
		AcceptCount: 3,
	}
	require.NoError(t, h.KnowledgeKeeper.CompleteRound(h.Ctx, round, result))

	var fact *knowledgetypes.Fact
	h.KnowledgeKeeper.IterateFactsByDomain(h.Ctx, domain, func(factID string) bool {
		f, ok := h.KnowledgeKeeper.GetFact(h.Ctx, factID)
		if ok && f.ClaimId == claim.Id {
			fact = f
			return true
		}
		return false
	})
	require.NotNil(t, fact)
	require.Equal(t, claim.ReasoningTrace, fact.ReasoningTrace,
		"reasoning_trace must flow from claim to fact")
}
