package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// These tests pin the dependency-confidence-floor semantics of
// computeProvenance / createFactFromClaim (rounds.go).
//
// The defect they guard against: computeProvenance used to return a bare
// uint64 floor, and the clamp in createFactFromClaim was guarded by
// `floor > 0` — so a floor that computed to exactly zero (a resolved cite
// whose contribution was zero) was indistinguishable from "no citations at
// all", and the new fact escaped the clamp entirely. Because
// InferenceStrengthBps has no minimum, an edge declaring strength 1 made any
// cited target below total certainty contribute 0 by integer division —
// declaring your own reasoning weaker bought MORE confidence.
//
// The repair: computeProvenance reports whether any cited edge actually
// resolved (floorSet), and the clamp fires on floorSet — including a clamp
// to zero. A proof resting on a zero-confidence foundation inherits zero
// confidence, which is the stated design intent ("proof chains can't claim
// more confidence than their foundations") applied honestly.

// acceptProvenanceClaim builds a claim with the given citations, drives it
// through CompleteRound with an ACCEPT verdict at 850,000 confidence, and
// returns the created fact. Same pattern as conjecture_test.go — this is the
// internal BeginBlocker path, deliberately below the msg_server walls.
func acceptProvenanceClaim(
	t *testing.T,
	k keeper.Keeper,
	ctx sdk.Context,
	tag string,
	references []string,
	relations []*types.ClaimRelation,
) *types.Fact {
	t.Helper()

	claimID := "prov-claim-" + tag
	claim := &types.Claim{
		Id:          claimID,
		FactContent: "provenance floor test claim " + tag,
		Domain:      "physics",
		Category:    "general",
		Submitter:   "zrn1provsubmitter",
		Stake:       "1000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ClaimType:   types.ClaimType_CLAIM_TYPE_ASSERTION,
		References:  references,
		Relations:   relations,
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	round := makeRoundInPhase("r-prov-"+tag, claimID, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))
	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_ACCEPT,
		Confidence: 850_000,
	}))

	var created *types.Fact
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.ClaimId == claimID {
			created = f
			return true
		}
		return false
	})
	require.NotNil(t, created, "accepted claim %s must create a fact", claimID)
	return created
}

// findClampEvent returns the dependency_floor_bps attribute of the
// zerone.knowledge.confidence_clamped_to_floor event emitted for factID, if
// one was emitted.
func findClampEvent(t *testing.T, ctx sdk.Context, factID string) (string, bool) {
	t.Helper()
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != "zerone.knowledge.confidence_clamped_to_floor" {
			continue
		}
		var evFact, evFloor string
		for _, attr := range ev.Attributes {
			switch attr.Key {
			case "fact_id":
				evFact = attr.Value
			case "dependency_floor_bps":
				evFloor = attr.Value
			}
		}
		if evFact == factID {
			return evFloor, true
		}
	}
	return "", false
}

// (a) No citations at all → foundational fact, no floor, confidence unclamped.
// Unchanged behaviour — must pass before AND after the fix.
func TestProvenanceFloor_NoCites_NoFloor_Unclamped(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	fact := acceptProvenanceClaim(t, k, ctx, "nocite", nil, nil)

	require.Equal(t, uint32(0), fact.AxiomDistance, "no cites → foundational, distance 0")
	require.Equal(t, uint64(0), fact.DependencyConfidenceFloor)
	require.Equal(t, uint64(850_000), fact.Confidence,
		"a foundational fact keeps its panel confidence — there is no floor to clamp it")
	_, found := findClampEvent(t, ctx, fact.Id)
	require.False(t, found, "no clamp event may fire when no cited support exists")
}

// (b) An honest full-strength cite → floor inherited from the cited fact's
// effective confidence, confidence clamped down to it. Unchanged behaviour —
// must pass before AND after the fix.
func TestProvenanceFloor_FullStrengthCite_FloorInherited(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	makeTestFact(t, k, ctx, "base-full", "solid foundation", "physics", "general", "zrn1submitter1", 700_000)

	fact := acceptProvenanceClaim(t, k, ctx, "full", nil, []*types.ClaimRelation{{
		TargetFactId:         "base-full",
		Relation:             types.RelationType_RELATION_TYPE_SUPPORTS,
		Inference:            types.InferenceType_INFERENCE_TYPE_DEDUCTIVE,
		InferenceStrengthBps: 1_000_000,
	}})

	require.Equal(t, uint32(1), fact.AxiomDistance)
	require.Equal(t, uint64(700_000), fact.DependencyConfidenceFloor,
		"full-strength cite preserves the cited fact's confidence as the floor")
	require.Equal(t, uint64(700_000), fact.Confidence,
		"panel confidence 850k is clamped to the honest 700k floor")
	floorAttr, found := findClampEvent(t, ctx, fact.Id)
	require.True(t, found, "the clamp event must fire when the floor bites")
	require.Equal(t, "700000", floorAttr)
}

// (c) THE ESCAPE — security regression test. A Relations edge declaring
// InferenceStrengthBps = 1 makes the cited target's contribution
// 850,000 × 1 / 1,000,000 = 0 by integer division. Before the fix, floor 0
// was read as "no floor at all" and the fact kept its full 850k confidence:
// declaring your own reasoning nearly worthless bought MORE confidence than
// citing honestly (compare (b): the honest citer ends at 700k). After the
// fix, a resolved edge that contributes zero clamps the fact to zero.
func TestProvenanceFloor_StrengthOneEscape_NowClamped(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	makeTestFact(t, k, ctx, "base-esc", "solid foundation", "physics", "general", "zrn1submitter1", 850_000)

	fact := acceptProvenanceClaim(t, k, ctx, "esc", nil, []*types.ClaimRelation{{
		TargetFactId:         "base-esc",
		Relation:             types.RelationType_RELATION_TYPE_SUPPORTS,
		Inference:            types.InferenceType_INFERENCE_TYPE_INDUCTIVE,
		InferenceStrengthBps: 1, // declares the inference nearly worthless
	}})

	require.Equal(t, uint32(1), fact.AxiomDistance)
	require.Equal(t, uint64(0), fact.DependencyConfidenceFloor,
		"contribution = 850,000 × 1 / 1,000,000 = 0")
	require.Equal(t, uint64(0), fact.Confidence,
		"declaring your inference nearly worthless must weaken the fact to zero, not exempt it from the floor its foundation imposes")
	floorAttr, found := findClampEvent(t, ctx, fact.Id)
	require.True(t, found, "a clamp to zero is exactly the event a reader needs — it must fire")
	require.Equal(t, "0", floorAttr)
}

// (d) A cite that resolves to a zero-effective-confidence foundation → the
// new fact inherits zero, it is not exempted. Uses an untyped Reference so
// the References path is covered too.
func TestProvenanceFloor_ZeroConfidenceFoundation_NotExempt(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	makeTestFact(t, k, ctx, "base-zero", "zero-confidence foundation", "physics", "general", "zrn1submitter1", 0)

	fact := acceptProvenanceClaim(t, k, ctx, "zero", []string{"base-zero"}, nil)

	require.Equal(t, uint32(1), fact.AxiomDistance,
		"the cite resolved — the fact is derived, not foundational")
	require.Equal(t, uint64(0), fact.DependencyConfidenceFloor)
	require.Equal(t, uint64(0), fact.Confidence,
		"a proof resting on a zero-confidence foundation inherits zero confidence")
}

// (e1) The AdvanceConfidence interaction, isolated. AdvanceConfidence floors
// computed growth at 1 — for a VERIFIED fact sitting at zero confidence that
// floor would manufacture +1 belief every growth epoch on the passage of
// time alone, the exact failure the PROVISIONAL exclusion was written to
// prevent, reachable for ordinary facts once the floor clamp can produce
// zero-confidence VERIFIED facts.
func TestAdvanceConfidence_ZeroConfidenceFactDoesNotGrowOnTimeAlone(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	// An ordinary VERIFIED fact at zero confidence — what a dependency-floor
	// clamp to zero produces.
	makeTestFact(t, k, ctx, "f-zero", "zero-clamped fact", "physics", "general", "zrn1submitter1", 0)

	for i := 0; i < 3; i++ {
		require.NoError(t, k.AdvanceConfidence(ctx))
	}

	updated, found := k.GetFact(ctx, "f-zero")
	require.True(t, found)
	require.Equal(t, uint64(0), updated.Confidence,
		"the minimum-1 growth floor is a rounding convenience, not a belief generator — a fact at zero must stay at zero until something actually happens to it")
}

// (e2) The rounding convenience itself must survive the fix: a fact with
// small positive confidence whose computed growth rounds to zero still
// climbs by the minimum 1. Must pass before AND after the fix.
func TestAdvanceConfidence_MinimumGrowthStillAppliesAboveZero(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	// growth = 50 × 11,000 / 1,000,000 = 0 → minimum 1 applies.
	makeTestFact(t, k, ctx, "f-small", "small but nonzero", "physics", "general", "zrn1submitter1", 50)

	require.NoError(t, k.AdvanceConfidence(ctx))

	updated, found := k.GetFact(ctx, "f-small")
	require.True(t, found)
	require.Equal(t, uint64(51), updated.Confidence,
		"positive confidence below the rounding threshold keeps the minimum-1 growth")
}

// (e3) End-to-end: the strength-1 escape fact, clamped to zero at creation,
// stays at zero as growth epochs pass. This is the full interaction — the
// floor fix must not hand the escaped confidence back one unit at a time.
func TestProvenanceFloor_ClampedToZeroStaysZeroThroughGrowth(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	makeTestFact(t, k, ctx, "base-e2e", "solid foundation", "physics", "general", "zrn1submitter1", 850_000)

	fact := acceptProvenanceClaim(t, k, ctx, "e2e", nil, []*types.ClaimRelation{{
		TargetFactId:         "base-e2e",
		Relation:             types.RelationType_RELATION_TYPE_SUPPORTS,
		Inference:            types.InferenceType_INFERENCE_TYPE_INDUCTIVE,
		InferenceStrengthBps: 1,
	}})
	require.Equal(t, uint64(0), fact.Confidence, "clamped to the zero floor at creation")

	for i := 0; i < 3; i++ {
		require.NoError(t, k.AdvanceConfidence(ctx))
	}

	updated, found := k.GetFact(ctx, fact.Id)
	require.True(t, found)
	require.Equal(t, uint64(0), updated.Confidence,
		"a fact clamped to a zero foundation must not climb on the passage of time alone")
}
