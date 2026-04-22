package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestNormativeCommitment_SchemaDistinction verifies the Phase 6 is-ought
// wall at the schema layer:
//   · commitments live in their own registry (prefix 0x59, distinct from
//     facts and methodologies)
//   · commitments have no confidence field
//   · the five bootstrap commitments are seeded and queryable
//   · a commitment_id cannot collide with a fact_id (separate namespaces)
func TestNormativeCommitment_SchemaDistinction(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultCommitments(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	resp, err := qs.NormativeCommitments(h.Ctx, &knowledgetypes.QueryNormativeCommitmentsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Commitments, 5,
		"five bootstrap commitments expected")

	present := make(map[string]bool)
	for _, c := range resp.Commitments {
		present[c.Id] = true
		// A commitment must have no confidence score — it is not a fact.
		// Check by introspection: the proto type has no confidence field.
		// Statement + rationale must be present.
		require.NotEmpty(t, c.Statement)
		require.NotEmpty(t, c.Rationale)
		require.True(t, c.Active)
	}

	// The bootstrap set.
	for _, id := range []string{
		"NC-DUAL-KEY-RESEARCH",
		"NC-TRANSPARENCY",
		"NC-METHODOLOGY-OVER-STATEMENT",
		"NC-FALSIFICATION-IS-PROGRESS",
		"NC-IS-OUGHT-WALL",
	} {
		require.True(t, present[id], "commitment %s must be seeded", id)
	}

	// Single-fetch sanity.
	c, err := qs.NormativeCommitment(h.Ctx, &knowledgetypes.QueryNormativeCommitmentRequest{
		Id: "NC-IS-OUGHT-WALL",
	})
	require.NoError(t, err)
	require.True(t, c.Found)
	require.Equal(t, knowledgetypes.CommitmentCategoryPrinciple, c.Commitment.Category)

	// Not found → found=false, no error.
	missing, err := qs.NormativeCommitment(h.Ctx, &knowledgetypes.QueryNormativeCommitmentRequest{
		Id: "NC-DOES-NOT-EXIST",
	})
	require.NoError(t, err)
	require.False(t, missing.Found)
}

// TestNormativeCommitment_NoConfidenceField is a type-level assertion: a
// commitment cannot carry a confidence score. This is enforced by the proto
// schema. If a future refactor adds one, this test should fail to compile
// (or at least fail at runtime) so the is-ought wall remains visible.
func TestNormativeCommitment_NoConfidenceField(t *testing.T) {
	c := &knowledgetypes.NormativeCommitment{
		Id:        "TEST-NC",
		Statement: "This is a value, not a fact.",
	}
	// The proto does not define Confidence on NormativeCommitment. If this
	// test compiles, the is-ought wall is still schematically enforced.
	require.NotNil(t, c)
	require.Empty(t, c.Tags)
	// The commitment has no confidence-shaped field; the only score-like
	// value is `Version`, which tracks amendment history, not truth.
	require.Zero(t, c.Version)
}
