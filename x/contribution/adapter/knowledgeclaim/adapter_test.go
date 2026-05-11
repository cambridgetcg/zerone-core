package knowledgeclaim_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	contribtypes "github.com/zerone-chain/zerone/x/contribution/types"
	"github.com/zerone-chain/zerone/x/contribution/adapter/knowledgeclaim"
)

type mockKnowledgeKeeper struct {
	scores map[string]uint32
}

func (m *mockKnowledgeKeeper) GetClaimVerificationScore(_ context.Context, claimID string) (uint32, bool) {
	s, ok := m.scores[claimID]
	return s, ok
}

type mockCreedKeeper struct {
	version uint32
}

func (m *mockCreedKeeper) GetCurrentPinVersion(_ context.Context) uint32 {
	return m.version
}

func sampleContribution(claimID, manifestCID string, version uint32) *contribtypes.Contribution {
	return &contribtypes.Contribution{
		Id:              []byte("0123456789abcdef0123456789abcdef"),
		Class:           contribtypes.ContributionClass_KNOWLEDGE_CLAIM,
		Phase:           contribtypes.LifecyclePhase_PHASE_KNOWLEDGE,
		ClaimsAboutSelf: []byte("methodology trace"),
		TruthFloorAttestation: &contribtypes.TruthFloorAttestation{
			CreedVersion: version,
		},
		Payload: &contribtypes.ContributionPayload{
			Payload: &contribtypes.ContributionPayload_Knowledge{
				Knowledge: &contribtypes.KnowledgeClaim{
					ClaimId:        claimID,
					TokManifestCid: manifestCID,
				},
			},
		},
	}
}

func TestAdapter_Class(t *testing.T) {
	a := knowledgeclaim.NewAdapter(&mockKnowledgeKeeper{}, &mockCreedKeeper{version: 1})
	require.Equal(t, contribtypes.ContributionClass_KNOWLEDGE_CLAIM, a.Class())
}

func TestAdapter_Classify_HappyPath(t *testing.T) {
	a := knowledgeclaim.NewAdapter(&mockKnowledgeKeeper{}, &mockCreedKeeper{version: 1})
	c := sampleContribution("claim-42", "ipfs://manifest", 1)
	require.NoError(t, a.Classify(context.Background(), c))
}

func TestAdapter_Classify_RejectsWrongPhase(t *testing.T) {
	a := knowledgeclaim.NewAdapter(&mockKnowledgeKeeper{}, &mockCreedKeeper{version: 1})
	c := sampleContribution("claim-42", "ipfs://manifest", 1)
	c.Phase = contribtypes.LifecyclePhase_PHASE_FOUNDATION
	err := a.Classify(context.Background(), c)
	require.ErrorIs(t, err, contribtypes.ErrInvalidClassPhase)
}

func TestAdapter_Classify_RejectsEmptyClaimsAboutSelf(t *testing.T) {
	a := knowledgeclaim.NewAdapter(&mockKnowledgeKeeper{}, &mockCreedKeeper{version: 1})
	c := sampleContribution("claim-42", "ipfs://manifest", 1)
	c.ClaimsAboutSelf = nil
	err := a.Classify(context.Background(), c)
	require.ErrorIs(t, err, contribtypes.ErrClaimsAboutSelfEmpty)
}

func TestAdapter_Classify_RejectsStaleTruthFloor(t *testing.T) {
	a := knowledgeclaim.NewAdapter(&mockKnowledgeKeeper{}, &mockCreedKeeper{version: 5})
	c := sampleContribution("claim-42", "ipfs://manifest", 3) // version mismatch
	err := a.Classify(context.Background(), c)
	require.ErrorIs(t, err, contribtypes.ErrTruthFloorStale)
}

func TestAdapter_Classify_RejectsMissingTruthFloor(t *testing.T) {
	a := knowledgeclaim.NewAdapter(&mockKnowledgeKeeper{}, &mockCreedKeeper{version: 1})
	c := sampleContribution("claim-42", "ipfs://manifest", 1)
	c.TruthFloorAttestation = nil
	err := a.Classify(context.Background(), c)
	require.ErrorIs(t, err, contribtypes.ErrTruthFloorMissing)
}

func TestAdapter_SubstrateLink_FullWhenManifestPresent(t *testing.T) {
	a := knowledgeclaim.NewAdapter(&mockKnowledgeKeeper{}, &mockCreedKeeper{version: 1})
	c := sampleContribution("claim-42", "ipfs://manifest", 1)
	link, err := a.SubstrateLink(context.Background(), c)
	require.NoError(t, err)
	require.Equal(t, uint32(10_000), link)
}

func TestAdapter_SubstrateLink_ZeroWhenManifestAbsent(t *testing.T) {
	a := knowledgeclaim.NewAdapter(&mockKnowledgeKeeper{}, &mockCreedKeeper{version: 1})
	c := sampleContribution("claim-42", "", 1)
	_, err := a.SubstrateLink(context.Background(), c)
	require.ErrorIs(t, err, contribtypes.ErrSubstrateLinkAbsent)
}

func TestAdapter_Verify_ReturnsKnowledgeScore(t *testing.T) {
	mk := &mockKnowledgeKeeper{scores: map[string]uint32{"claim-42": 750_000}}
	a := knowledgeclaim.NewAdapter(mk, &mockCreedKeeper{version: 1})
	c := sampleContribution("claim-42", "ipfs://manifest", 1)
	score, err := a.Verify(context.Background(), c)
	require.NoError(t, err)
	require.Equal(t, uint32(750_000), score)
}

func TestAdapter_Verify_BackRefNotFound(t *testing.T) {
	mk := &mockKnowledgeKeeper{scores: map[string]uint32{}}
	a := knowledgeclaim.NewAdapter(mk, &mockCreedKeeper{version: 1})
	c := sampleContribution("missing-claim", "ipfs://manifest", 1)
	_, err := a.Verify(context.Background(), c)
	require.ErrorIs(t, err, contribtypes.ErrBackRefNotFound)
}
