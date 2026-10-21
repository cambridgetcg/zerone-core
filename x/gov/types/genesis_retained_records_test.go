package types_test

import (
	"bytes"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// This pins the retained-records schema: the module's genesis JSON
// path (encoding/json, as x/gov/module.go uses) keeps every retained record
// and the research-spend counter's presence, including an explicitly stored
// zero, and native genesis carries none of them.
func TestGovRetainedRecordsSurviveModuleJSON(t *testing.T) {
	zero, five := uint64(0), uint64(5)
	proposer := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	seat := sdk.AccAddress(bytes.Repeat([]byte{2}, 20)).String()
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
				gs.ResearchSpendProposals = []*types.ResearchSpendProposal{{ProposalId: 4, Proposer: proposer, Recipient: seat, Amount: "1000", Stage: "voting", CreatedAt: 10}}
				gs.ResearchCommunityVotes = []*types.GenesisResearchCommunityVote{{ProposalId: 4, Voter: seat, Vote: "yes"}}
				gs.DistinctVoters = []*types.GenesisDistinctVoter{{Voter: proposer, FirstVotedAtBlock: 77}}
				gs.Lips = []*types.LIP{{Id: "LIP-9", Category: types.CategoryPhaseRollback}}
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
