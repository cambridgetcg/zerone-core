package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// recordingOracle answers every evaluation with a fixed verdict and records
// what it was asked.
type recordingOracle struct {
	verdict    string
	confidence float64
	calls      int
	lastClaim  string
	lastDomain string
	lastType   string
}

func (o *recordingOracle) Evaluate(claim, domain, claimType string) (*OracleEvaluation, error) {
	o.calls++
	o.lastClaim, o.lastDomain, o.lastType = claim, domain, claimType
	return &OracleEvaluation{Verdict: o.verdict, Confidence: o.confidence}, nil
}

func TestPotVerdictForClaim_ClaimNotFoundCastsNoVote(t *testing.T) {
	oracle := &recordingOracle{verdict: "accept", confidence: 0.9}
	claim := &knowledgetypes.Claim{FactContent: "water boils at 100C at sea level", Domain: "physics"}

	verdict, confidence, ok := potVerdictForClaim(oracle, claim, false)
	require.False(t, ok, "a missing claim must cast no vote")
	require.Equal(t, "", verdict)
	require.Equal(t, uint64(0), confidence)

	verdict, confidence, ok = potVerdictForClaim(oracle, nil, true)
	require.False(t, ok, "a nil claim must cast no vote")
	require.Equal(t, "", verdict)
	require.Equal(t, uint64(0), confidence)

	require.Zero(t, oracle.calls, "the oracle is not asked about a missing claim")
}

func TestPotVerdictForClaim_FoundClaimUsesOracle(t *testing.T) {
	oracle := &recordingOracle{verdict: "reject", confidence: 0.8}
	claim := &knowledgetypes.Claim{
		FactContent: "water freezes at 200C",
		Domain:      "physics",
		ClaimType:   knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION,
	}

	verdict, confidence, ok := potVerdictForClaim(oracle, claim, true)
	require.True(t, ok)
	require.Equal(t, "reject", verdict)
	require.Equal(t, uint64(800_000), confidence)
	require.Equal(t, 1, oracle.calls)
	require.Equal(t, claim.FactContent, oracle.lastClaim)
	require.Equal(t, claim.Domain, oracle.lastDomain)
	require.Equal(t, claim.ClaimType.String(), oracle.lastType)

	_, _, ok = potVerdictForClaim(nil, claim, true)
	require.False(t, ok, "no oracle client must cast no vote")
}
