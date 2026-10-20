package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// FX8 adds only the retained-records schema; export and import are its fix
// lane's work. This pins what that work relies on: the module's genesis JSON
// path (encoding/json, as x/gov/module.go uses) keeps every retained record
// and the research-spend counter's presence, including an explicitly stored
// zero, and native genesis carries none of them.
func TestGovRetainedRecordsSurviveModuleJSON(t *testing.T) {
	zero, five := uint64(0), uint64(5)
	native := types.DefaultGenesisState()
	require.Nil(t, native.NextResearchSpendNumber, "native genesis stores no counter")
	require.Empty(t, native.ResearchSpendProposals)
	require.Empty(t, native.ResearchCommunityVotes)
	require.Empty(t, native.DistinctVoters)
	require.Empty(t, native.PhaseTransitions)

	for _, tc := range []struct {
		name    string
		counter *uint64
		records bool
	}{
		{"legacy omission", nil, false},
		{"stored zero counter", &zero, false},
		{"retained records", &five, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := types.DefaultGenesisState()
			gs.NextResearchSpendNumber = tc.counter
			if tc.records {
				gs.ResearchSpendProposals = []*types.ResearchSpendProposal{{ProposalId: 4, Proposer: "zrn1proposer", Amount: "1000", Stage: "voting", CreatedAt: 10}}
				gs.ResearchCommunityVotes = []*types.GenesisResearchCommunityVote{{ProposalId: 4, Voter: "zrn1seat", Vote: "yes"}}
				gs.DistinctVoters = []*types.GenesisDistinctVoter{{Voter: "zrn1voter", FirstVotedAtBlock: 77}}
				gs.PhaseTransitions = []*types.GenesisPhaseTransition{{
					LipId: "LIP-9", TargetPhase: types.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER,
					ConditionsSnapshot: &types.PhaseTransitionConditions{DistinctLipVoters: 3, ResearchFundBalance: "42"},
					Stage:              "pending_activation", ActivationBlock: 900, IsRollback: true, CancelReason: "kept",
				}}
			}
			data, err := json.Marshal(gs)
			require.NoError(t, err)
			var restored types.GenesisState
			require.NoError(t, json.Unmarshal(data, &restored))
			require.True(t, proto.Equal(gs, &restored), "every retained record and the counter's presence survive")
			require.Equal(t, tc.counter != nil, restored.NextResearchSpendNumber != nil)
			require.NoError(t, restored.Validate())
		})
	}
}
