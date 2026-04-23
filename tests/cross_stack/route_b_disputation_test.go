package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestRouteB_DisputationCorpus covers the argumentation training data path:
// a challenge claim carries argument_text; after the round completes, the
// full dispute (fact + argument + outcome) is retrievable as a single row.
//
// Scenarios exercised:
//   - A challenge that SUCCEEDS (fact DISPROVEN) → outcome="disproven"
//   - A challenge that FAILS (fact survives, corroboration+1) → outcome="survived"
//   - only_successful_challenges filter returns only the disproved pairs
func TestRouteB_DisputationCorpus(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))

	domain := "disputation_corpus_domain"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   domain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	// Seed two facts to challenge, one that will survive and one that won't.
	survivor := &knowledgetypes.Fact{
		Id:         "DISP-SURVIVOR",
		Content:    "A fact that survives the challenge.",
		Domain:     domain,
		Category:   "empirical",
		Confidence: 800_000,
		Status:     knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		Submitter:  "zerone1factauthor00000000000000000001",
		MethodId:   knowledgetypes.MethodologyEmpirical,
	}
	victim := &knowledgetypes.Fact{
		Id:         "DISP-VICTIM",
		Content:    "A fact that gets disproven.",
		Domain:     domain,
		Category:   "empirical",
		Confidence: 700_000,
		Status:     knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		Submitter:  "zerone1factauthor00000000000000000002",
		MethodId:   knowledgetypes.MethodologyEmpirical,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, survivor))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, victim))

	// Helper: build a challenge claim with argument text and run its round.
	// verdict = ACCEPT means challenge succeeds → fact disproven.
	// verdict = REJECT means challenge fails → fact survives.
	issueChallenge := func(id, target, argument string, verdict knowledgetypes.Verdict) {
		challenger := "zerone1disputechallenger00000000000001"
		challenge := &knowledgetypes.Claim{
			Id:                id,
			Submitter:         challenger,
			FactContent:       "challenge of " + target,
			Domain:            domain,
			Category:          "empirical",
			Status:            knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
			Stake:             "11000000",
			ProvisionalFactId: target,
			MethodId:          knowledgetypes.MethodologyDialectical,
			ArgumentText:      argument,
			Relations: []*knowledgetypes.ClaimRelation{
				{
					TargetFactId: target,
					Relation:     knowledgetypes.RelationType_RELATION_TYPE_CONTRADICTS,
				},
			},
		}
		require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, challenge))
		round := &knowledgetypes.VerificationRound{
			Id:             "round-" + id,
			ClaimId:        challenge.Id,
			Phase:          knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
			StartedAtBlock: 1,
		}
		var result *knowledgekeeper.VerificationResult
		if verdict == knowledgetypes.Verdict_VERDICT_ACCEPT {
			result = &knowledgekeeper.VerificationResult{
				Verdict: verdict, Confidence: 850_000, AcceptCount: 3,
			}
		} else {
			result = &knowledgekeeper.VerificationResult{
				Verdict: verdict, Confidence: 700_000, RejectCount: 3,
			}
		}
		require.NoError(t, h.KnowledgeKeeper.CompleteRound(h.Ctx, round, result))
	}

	// Survivor challenge: REJECTED → fact survives and corroborates.
	issueChallenge("chal-survivor",
		survivor.Id,
		"The cited data is selection-biased; the alleged effect is an artefact of sampling.",
		knowledgetypes.Verdict_VERDICT_REJECT)

	// Victim challenge: ACCEPTED → fact gets DISPROVEN.
	issueChallenge("chal-victim",
		victim.Id,
		"The claim contradicts the conservation law derived from the stated formal system.",
		knowledgetypes.Verdict_VERDICT_ACCEPT)

	// Fact statuses reflect the outcomes.
	updatedSurvivor, _ := h.KnowledgeKeeper.GetFact(h.Ctx, survivor.Id)
	require.Equal(t, uint64(1), updatedSurvivor.CorroborationCount)

	updatedVictim, _ := h.KnowledgeKeeper.GetFact(h.Ctx, victim.Id)
	require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, updatedVictim.Status)

	// ─── DisputationCorpus query, both outcomes ────────────────────────
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	all, err := qs.DisputationCorpus(h.Ctx, &knowledgetypes.QueryDisputationCorpusRequest{})
	require.NoError(t, err)
	require.Len(t, all.Entries, 2)
	require.Greater(t, all.SnapshotBlockHeight, uint64(0))

	outcomes := map[string]*knowledgetypes.DisputationCorpusEntry{}
	for _, e := range all.Entries {
		outcomes[e.Outcome] = e
	}
	require.Contains(t, outcomes, "survived")
	require.Contains(t, outcomes, "disproven")
	require.NotEmpty(t, outcomes["survived"].ArgumentText,
		"survived disputation must carry the challenger's argument text")
	require.NotEmpty(t, outcomes["disproven"].ArgumentText,
		"disproven disputation must carry the challenger's argument text")
	require.Equal(t, survivor.Id, outcomes["survived"].ChallengedFactId)
	require.Equal(t, victim.Id, outcomes["disproven"].ChallengedFactId)
	// Denormalised fact content should be present for immediate training consumption.
	require.Equal(t, survivor.Content, outcomes["survived"].FactContent)
	require.Equal(t, victim.Content, outcomes["disproven"].FactContent)
	require.Equal(t, updatedSurvivor.CorroborationCount, outcomes["survived"].CorroborationAfter)

	// ─── Filtered: only successful challenges ─────────────────────────
	successful, err := qs.DisputationCorpus(h.Ctx, &knowledgetypes.QueryDisputationCorpusRequest{
		OnlySuccessfulChallenges: true,
	})
	require.NoError(t, err)
	require.Len(t, successful.Entries, 1)
	require.Equal(t, "disproven", successful.Entries[0].Outcome)
	require.Equal(t, victim.Id, successful.Entries[0].ChallengedFactId)
}

// TestRouteB_DisputationRebuttalText asserts rebuttal text round-trips.
// When the defender attaches a rebuttal_text to a challenge claim — via
// whatever off-chain negotiation flow — it is stored on the challenge
// record and surfaces in the corpus export.
func TestRouteB_DisputationRebuttalText(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))

	domain := "rebuttal_domain"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   domain,
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))
	target := &knowledgetypes.Fact{
		Id: "REBUTTAL-TARGET", Content: "target of rebutted challenge",
		Domain: domain, Category: "empirical",
		Confidence: 800_000, Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		Submitter: "zerone1rebuttaltarget0000000000000000a",
		MethodId:  knowledgetypes.MethodologyEmpirical,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, target))

	challenge := &knowledgetypes.Claim{
		Id: "chal-with-rebuttal", Submitter: "zerone1rebuttalchallenger00000000000a",
		FactContent: "chal of " + target.Id, Domain: domain, Category: "empirical",
		Status:            knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Stake:             "11000000",
		ProvisionalFactId: target.Id,
		MethodId:          knowledgetypes.MethodologyDialectical,
		ArgumentText:      "The experimental setup confounds A with B.",
		RebuttalText:      "B was explicitly controlled for in trials 2-4 of the stated protocol.",
		Relations: []*knowledgetypes.ClaimRelation{{
			TargetFactId: target.Id,
			Relation:     knowledgetypes.RelationType_RELATION_TYPE_CONTRADICTS,
		}},
	}
	require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, challenge))

	round := &knowledgetypes.VerificationRound{
		Id: "round-rebuttal", ClaimId: challenge.Id,
		Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, StartedAtBlock: 1,
	}
	require.NoError(t, h.KnowledgeKeeper.CompleteRound(h.Ctx, round, &knowledgekeeper.VerificationResult{
		Verdict: knowledgetypes.Verdict_VERDICT_REJECT, Confidence: 700_000, RejectCount: 3,
	}))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	resp, err := qs.DisputationCorpus(h.Ctx, &knowledgetypes.QueryDisputationCorpusRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, challenge.ArgumentText, resp.Entries[0].ArgumentText)
	require.Equal(t, challenge.RebuttalText, resp.Entries[0].RebuttalText,
		"rebuttal text must round-trip through the challenge record into the training corpus")
}
