package keeper

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
)

// Interface fixtures cover inconsistent optional-reader responses; real stored
// records and application wiring are exercised in app/trust_score_composition_test.go.
type scoreReaders struct {
	calls       []string
	calibration *knowledgetypes.AgentCalibration
	found       bool
}

func (r *scoreReaders) GetAgentCalibration(context.Context, string) (*knowledgetypes.AgentCalibration, bool) {
	r.calls = append(r.calls, "calibration")
	return r.calibration, r.found
}

func (r *scoreReaders) IterateQualifications(_ context.Context, visit func(*qualificationtypes.DomainQualification) bool) {
	r.calls = append(r.calls, "qualifications")
	for _, q := range []*qualificationtypes.DomainQualification{
		nil,
		{Validator: "other", Domain: "excluded"},
		{Validator: "observed", Domain: "z", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, Metrics: &qualificationtypes.QualificationMetrics{AccuracyBps: 800_000, TotalVerifications: 9}},
		{Validator: "observed", Domain: "a"},
		{Validator: "observed", Domain: "b", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE},
	} {
		if visit(q) {
			panic("score reader must not stop early")
		}
	}
}

func (r *scoreReaders) GetQualificationWeight(_ context.Context, address, domain string) uint32 {
	r.calls = append(r.calls, "weight/"+address+"/"+domain)
	if domain == "z" {
		return 3
	}
	return 1
}

func (r *scoreReaders) GetActiveQualificationPenalty(_ context.Context, address, domain string) (*qualificationtypes.QualificationPenalty, bool) {
	r.calls = append(r.calls, "penalty/"+address+"/"+domain)
	switch domain {
	case "z":
		return &qualificationtypes.QualificationPenalty{}, true
	case "a":
		return nil, true // Flag is retained but does not increment the count.
	default:
		return &qualificationtypes.QualificationPenalty{}, false
	}
}

func (r *scoreReaders) CountUpheldStrikesAgainst(_ context.Context, address string) uint32 {
	r.calls = append(r.calls, "strikes/"+address)
	return 1
}

func TestScoreAdapterSelectionAndReadOrder(t *testing.T) {
	r := &scoreReaders{calibration: &knowledgetypes.AgentCalibration{CalibrationScoreBps: 900_000}, found: true}
	k := NewKeeper(nil)
	k.SetKnowledgeKeeper(r)
	k.SetQualificationKeeper(r)
	k.SetCaptureChallengeKeeper(r)
	out, err := k.BuildScore(sdk.Context{}.WithBlockHeight(42), "observed")
	require.NoError(t, err)
	require.Equal(t, []string{"calibration", "qualifications", "weight/observed/z", "penalty/observed/z", "penalty/observed/a", "weight/observed/b", "penalty/observed/b", "strikes/observed"}, r.calls)
	require.Equal(t, uint64(600_000), out.VerificationAccuracyBps)
	require.Equal(t, uint64(450_000), out.CompositeBps)
	require.Equal(t, uint32(1), out.ActivePenalties)
	require.Equal(t, uint32(1), out.CartelStrikes)
	require.Equal(t, "F", out.Band)
	require.Len(t, out.PerDomain, 3)
	require.Equal(t, "a", out.PerDomain[0].Domain)
	require.True(t, out.PerDomain[0].HasActivePenalty)
	require.Zero(t, out.PerDomain[0].EffectiveWeight)
	require.Equal(t, "b", out.PerDomain[1].Domain)
	require.False(t, out.PerDomain[1].HasActivePenalty)
	require.Zero(t, out.PerDomain[1].AccuracyBps)
	require.Equal(t, "z", out.PerDomain[2].Domain)
	require.Equal(t, uint64(9), out.PerDomain[2].TotalVerifications)
}

func TestScoreAdapterOptionalReaders(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(42)
	k := NewKeeper(nil)
	out, err := k.BuildScore(ctx, "observed")
	require.NoError(t, err)
	require.Equal(t, uint64(500_000), out.CompositeBps)
	require.Nil(t, out.PerDomain)
	for _, r := range []*scoreReaders{
		{}, {found: true},
		{calibration: &knowledgetypes.AgentCalibration{CalibrationScoreBps: 900_000}},
		{calibration: &knowledgetypes.AgentCalibration{}, found: true},
	} {
		k.SetKnowledgeKeeper(r)
		out, err := k.BuildScore(ctx, "observed")
		require.NoError(t, err)
		require.Equal(t, uint64(500_000), out.SubmissionAccuracyBps)
		require.Equal(t, uint64(500_000), out.CompositeBps)
	}
	// An empty address is refused before SDK-context unwrapping or producer reads.
	_, err = k.BuildScore(context.Background(), "")
	require.EqualError(t, err, "address required")
}
