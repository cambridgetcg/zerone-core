package knowledgeclaim_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	contribtypes "github.com/zerone-chain/zerone/x/contribution/types"
	"github.com/zerone-chain/zerone/x/contribution/adapter/knowledgeclaim"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestBuildContributionFromSnapshot_IDStable(t *testing.T) {
	snap := knowledgetypes.ClaimSnapshot{
		Submitter:        "zrn1abc",
		Domain:           "math",
		StatementHash:    []byte("hash"),
		MethodologyTrace: []byte("trace"),
		AxiomRefs:        []string{"ax-1"},
		TokManifestCID:   "ipfs://manifest",
		SubmittedAtBlock: 100,
	}

	c1 := knowledgeclaim.BuildContributionFromSnapshot("claim-42", snap, 100)
	c2 := knowledgeclaim.BuildContributionFromSnapshot("claim-42", snap, 200) // different block
	require.Equal(t, c1.Id, c2.Id, "id must be stable across re-mirrors of the same claim")
}

func TestBuildContributionFromSnapshot_ClassAndPhase(t *testing.T) {
	snap := knowledgetypes.ClaimSnapshot{Submitter: "zrn1abc"}
	c := knowledgeclaim.BuildContributionFromSnapshot("c1", snap, 1)
	require.Equal(t, contribtypes.ContributionClass_KNOWLEDGE_CLAIM, c.Class)
	require.Equal(t, contribtypes.LifecyclePhase_PHASE_KNOWLEDGE, c.Phase)
	require.Equal(t, "c1", c.BackRef)
	require.NotNil(t, c.Payload.GetKnowledge())
}
