package app

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	zeronestakingtypes "github.com/zerone-chain/zerone/x/staking/types"
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

// TestHandleCommitPhase_DeclinedRoundStoresAndRevealsNothing drives the real
// commit and reveal handlers with a validator that VRF selection always picks
// (it holds all active stake). With no usable verdict the commit phase must
// store no LocalCommitment, so the reveal phase has nothing to reveal.
func TestHandleCommitPhase_DeclinedRoundStoresAndRevealsNothing(t *testing.T) {
	app, ctx, _ := newAccountingAuthorityFixture(t)
	ctx = ctx.WithBlockHeight(10)

	operator := sdk.AccAddress(bytes.Repeat([]byte{0x5a}, 20)).String()
	app.ZeroneStakingKeeper.SetValidator(ctx, &zeronestakingtypes.Validator{
		OperatorAddress: operator,
		SelfDelegation:  "1000000",
		DelegatedStake:  "0",
		TotalStake:      "1000000",
		IsActive:        true,
	})

	foundClaim := &knowledgetypes.Claim{
		Id:          "claim-declined",
		FactContent: "water freezes at 200C",
		Domain:      "physics",
		ClaimType:   knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION,
	}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, foundClaim))

	for name, test := range map[string]struct {
		claimID string
		oracle  OracleClient
		calls   int
	}{
		"claim not found":     {claimID: "claim-missing", oracle: &recordingOracle{verdict: "accept", confidence: 0.9}},
		"no oracle client":    {claimID: foundClaim.Id, oracle: nil},
		"uncertain verdict":   {claimID: foundClaim.Id, oracle: &recordingOracle{verdict: "uncertain", confidence: 0.9}, calls: 1},
		"unknown verdict":     {claimID: foundClaim.Id, oracle: &recordingOracle{verdict: "malformed", confidence: 0.9}, calls: 1},
		"negative confidence": {claimID: foundClaim.Id, oracle: &recordingOracle{verdict: "accept", confidence: -0.5}, calls: 1},
	} {
		t.Run(name, func(t *testing.T) {
			config := &VoteExtensionConfig{
				ValidatorAddress:    operator,
				ValidatorPrivateKey: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x01}, ed25519.SeedSize)),
				LocalStore:          NewLocalCommitmentStore(""),
				OracleClient:        test.oracle,
			}
			round := &knowledgetypes.VerificationRound{
				Id:             "round-" + test.claimID,
				ClaimId:        test.claimID,
				StartedAtBlock: 1,
				Phase:          knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMMIT,
			}

			commitment, err := app.handleCommitPhase(ctx, config, round)
			require.ErrorContains(t, err, "casting no vote",
				"the round must reach the verdict decision, not stop at VRF or stake")
			require.Nil(t, commitment)
			require.Zero(t, config.LocalStore.Count(), "a declined round must store no commitment")
			if recorder, ok := test.oracle.(*recordingOracle); ok {
				require.Equal(t, test.calls, recorder.calls)
			}

			round.Phase = knowledgetypes.VerificationPhase_VERIFICATION_PHASE_REVEAL
			reveal, err := handleRevealPhase(config, round)
			require.Error(t, err)
			require.Nil(t, reveal, "a declined round must reveal nothing")
		})
	}

	// Control: with a usable verdict the same setup stores and reveals it, so
	// the empty store above is not an artefact of the fixture.
	config := &VoteExtensionConfig{
		ValidatorAddress:    operator,
		ValidatorPrivateKey: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x01}, ed25519.SeedSize)),
		LocalStore:          NewLocalCommitmentStore(""),
		OracleClient:        &recordingOracle{verdict: "reject", confidence: 0.9},
	}
	round := &knowledgetypes.VerificationRound{Id: "round-usable", ClaimId: foundClaim.Id, StartedAtBlock: 1}
	commitment, err := app.handleCommitPhase(ctx, config, round)
	require.NoError(t, err)
	require.NotNil(t, commitment)
	require.Equal(t, 1, config.LocalStore.Count())
	reveal, err := handleRevealPhase(config, round)
	require.NoError(t, err)
	require.Equal(t, "reject", reveal.Verdict)
	require.Equal(t, uint64(900_000), reveal.Confidence)
}
