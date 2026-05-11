package cross_stack_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	creedtypes "github.com/zerone-chain/zerone/x/creed/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestDoctrineImport_AllCommitmentsQueryable(t *testing.T) {
	h := NewTestHarness(t)
	// The harness uses a check-state context (app.NewContext(true)), which does
	// not see facts written during InitChain's deliver-state execution.
	// LoadDoctrineFacts is idempotent and re-materialises all doctrine facts
	// into the current context's store, matching what happens on-chain at genesis.
	require.NoError(t, h.KnowledgeKeeper.LoadDoctrineFacts(h.Ctx))

	for _, c := range creedtypes.CanonicalCommitments {
		id := fmt.Sprintf("commitment-%d", c.Number)
		f, found := h.KnowledgeKeeper.GetFact(h.Ctx, id)
		require.True(t, found, "TS commitment %s must be queryable", id)
		require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, f.Status)
		require.Equal(t, knowledgetypes.DoctrineConfidence, f.Confidence)
	}

	for _, c := range creedtypes.CanonicalToKCommitments {
		id := fmt.Sprintf("commitment-%s", c.Number)
		_, found := h.KnowledgeKeeper.GetFact(h.Ctx, id)
		require.True(t, found, "TC commitment %s must be queryable", id)
	}

	_, found := h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-UW")
	require.True(t, found, "commitment-UW must be queryable")

	_, found = h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-SL")
	require.True(t, found, "commitment-SL must be queryable")
}

func TestDoctrineImport_SubstrateLinkCanCite(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.LoadDoctrineFacts(h.Ctx))
	_, found := h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-TC1")
	require.True(t, found, "preflight: commitment-TC1 must exist")
	_, found = h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-UW")
	require.True(t, found, "preflight: commitment-UW must exist")
}

func TestDoctrineImport_EchoesEdgesQueryable(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.LoadDoctrineFacts(h.Ctx))

	relations, err := h.KnowledgeKeeper.GetFactRelations(h.Ctx, "commitment-UW")
	require.NoError(t, err)

	var foundUW11 bool
	for _, r := range relations {
		if r.TargetFactId == "commitment-11" && r.Relation == knowledgetypes.RelationType_RELATION_TYPE_SUPPORTS {
			foundUW11 = true
			break
		}
	}
	require.True(t, foundUW11, "commitment-UW SUPPORTS commitment-11 edge must be queryable")

	relations, err = h.KnowledgeKeeper.GetFactRelations(h.Ctx, "commitment-SL")
	require.NoError(t, err)

	var foundSLUW bool
	for _, r := range relations {
		if r.TargetFactId == "commitment-UW" && r.Relation == knowledgetypes.RelationType_RELATION_TYPE_REFINES {
			foundSLUW = true
			break
		}
	}
	require.True(t, foundSLUW, "commitment-SL REFINES commitment-UW edge must be queryable")
}
