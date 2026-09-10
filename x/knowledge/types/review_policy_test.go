package types_test

import (
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"testing"
)

func TestNeutralGenesisRequiresExplicitPolicyAndMatchingRecords(t *testing.T) {
	base := func() *types.GenesisState {
		return &types.GenesisState{RecordIntegrityEnabled: true, ReviewNeutralityEnabled: true,
			PendingClaims: []*types.Claim{{Id: "c", VerificationRoundId: "r", ReviewPolicyVersion: types.ReviewPolicyNeutral}},
			ActiveRounds: []*types.VerificationRound{{Id: "r", ClaimId: "c", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT,
				ReviewPolicyVersion: types.ReviewPolicyNeutral, CommitmentScheme: types.CommitmentSchemeReviewV2, CommitmentChainId: "source-chain", StartedAtBlock: 10, CommitDeadline: 20, RevealDeadline: 30, AggregationDeadline: 40}}}
	}
	require.NoError(t, types.ValidateGenesisRounds(base()))
	for _, tc := range []struct {
		name   string
		mutate func(*types.GenesisState)
	}{
		{"missing activation", func(g *types.GenesisState) { g.ReviewNeutralityEnabled = false }},
		{"missing integrity", func(g *types.GenesisState) { g.RecordIntegrityEnabled = false }},
		{"unknown claim policy", func(g *types.GenesisState) { g.PendingClaims[0].ReviewPolicyVersion = 2 }},
		{"unknown round policy", func(g *types.GenesisState) { g.ActiveRounds[0].ReviewPolicyVersion = 2 }},
		{"claim policy mismatch", func(g *types.GenesisState) { g.PendingClaims[0].ReviewPolicyVersion = 0 }},
		{"unbound scheme", func(g *types.GenesisState) {
			g.ActiveRounds[0].CommitmentScheme = 0
			g.ActiveRounds[0].CommitmentChainId = ""
		}},
		{"unordered deadlines", func(g *types.GenesisState) { g.ActiveRounds[0].RevealDeadline = 20 }},
		{"late commitment", func(g *types.GenesisState) { g.ActiveRounds[0].Commits = []*types.CommitEntry{{CommittedAtBlock: 20}} }},
		{"early reveal", func(g *types.GenesisState) { g.ActiveRounds[0].Reveals = []*types.RevealEntry{{RevealedAtBlock: 19}} }},
		{"late reveal", func(g *types.GenesisState) { g.ActiveRounds[0].Reveals = []*types.RevealEntry{{RevealedAtBlock: 30}} }},
	} {
		t.Run(tc.name, func(t *testing.T) { g := base(); tc.mutate(g); require.Error(t, types.ValidateGenesisRounds(g)) })
	}
}
