package types_test

import (
	"bytes"
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/gov/types"
)

func validRetainedGenesis() *types.GenesisState {
	address := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	gs := types.DefaultGenesisState()
	gs.NextResearchSpendNumber = proto.Uint64(9)
	gs.ResearchSpendProposals = []*types.ResearchSpendProposal{{ProposalId: 4, Proposer: address, Recipient: address, Amount: "7", Stage: "voting"}}
	gs.ResearchCommunityVotes = []*types.GenesisResearchCommunityVote{{ProposalId: 4, Voter: address, Vote: "yes"}}
	gs.DistinctVoters = []*types.GenesisDistinctVoter{{Voter: address, FirstVotedAtBlock: 1}}
	gs.Lips = []*types.LIP{{Id: "LIP-1", Category: types.CategoryPhaseTransition}}
	gs.PhaseTransitions = []*types.GenesisPhaseTransition{{LipId: "LIP-1", TargetPhase: types.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, Stage: types.PhaseTransitionStagePending}}
	return gs
}

func TestRetainedGenesisValidation(t *testing.T) {
	for _, value := range []uint64{0, 1<<63 + 1, math.MaxUint64 - 1, math.MaxUint64} {
		for _, gs := range []*types.GenesisState{types.DefaultGenesisState(), validRetainedGenesis()} {
			gs.NextResearchSpendNumber = proto.Uint64(value)
			require.ErrorContains(t, gs.Validate(), "allocation headroom")
		}
	}
	for _, value := range []uint64{1, 1 << 63} {
		gs := types.DefaultGenesisState()
		gs.NextResearchSpendNumber = proto.Uint64(value)
		require.NoError(t, gs.Validate())
	}
	require.NoError(t, validRetainedGenesis().Validate())
	cases := []struct {
		name   string
		mutate func(*types.GenesisState)
	}{
		{"nil-proposal", func(g *types.GenesisState) { g.ResearchSpendProposals[0] = nil }},
		{"zero-proposal", func(g *types.GenesisState) { g.ResearchSpendProposals[0].ProposalId = 0 }},
		{"duplicate-proposal", func(g *types.GenesisState) {
			g.ResearchSpendProposals = append(g.ResearchSpendProposals, g.ResearchSpendProposals[0])
		}},
		{"proposer", func(g *types.GenesisState) { g.ResearchSpendProposals[0].Proposer = "bad" }},
		{"recipient", func(g *types.GenesisState) { g.ResearchSpendProposals[0].Recipient = "" }},
		{"zero-amount", func(g *types.GenesisState) { g.ResearchSpendProposals[0].Amount = "0" }},
		{"negative-amount", func(g *types.GenesisState) { g.ResearchSpendProposals[0].Amount = "-1" }},
		{"malformed-amount", func(g *types.GenesisState) { g.ResearchSpendProposals[0].Amount = "NaN" }},
		{"proposal-stage", func(g *types.GenesisState) { g.ResearchSpendProposals[0].Stage = "new" }},
		{"core-vote1", func(g *types.GenesisState) { g.ResearchSpendProposals[0].Voter1Vote = "abstain" }},
		{"core-vote2", func(g *types.GenesisState) { g.ResearchSpendProposals[0].Voter2Vote = "YES" }},
		{"missing-counter", func(g *types.GenesisState) { g.NextResearchSpendNumber = nil }},
		{"zero-counter-with-records", func(g *types.GenesisState) { g.NextResearchSpendNumber = proto.Uint64(0) }},
		{"stale-counter", func(g *types.GenesisState) { g.NextResearchSpendNumber = proto.Uint64(4) }},
		{"exhausted-id", func(g *types.GenesisState) {
			g.ResearchSpendProposals[0].ProposalId = math.MaxUint64
			g.NextResearchSpendNumber = proto.Uint64(math.MaxUint64)
		}},
		{"nil-community-vote", func(g *types.GenesisState) { g.ResearchCommunityVotes[0] = nil }},
		{"orphan-community-vote", func(g *types.GenesisState) { g.ResearchCommunityVotes[0].ProposalId = 5 }},
		{"duplicate-community-vote", func(g *types.GenesisState) {
			g.ResearchCommunityVotes = append(g.ResearchCommunityVotes, g.ResearchCommunityVotes[0])
		}},
		{"community-address", func(g *types.GenesisState) { g.ResearchCommunityVotes[0].Voter = "bad" }},
		{"community-ballot", func(g *types.GenesisState) { g.ResearchCommunityVotes[0].Vote = "abstain" }},
		{"nil-distinct-voter", func(g *types.GenesisState) { g.DistinctVoters[0] = nil }},
		{"duplicate-distinct-voter", func(g *types.GenesisState) { g.DistinctVoters = append(g.DistinctVoters, g.DistinctVoters[0]) }},
		{"distinct-address", func(g *types.GenesisState) { g.DistinctVoters[0].Voter = "bad" }},
		{"nil-phase-transition", func(g *types.GenesisState) { g.PhaseTransitions[0] = nil }},
		{"empty-phase-id", func(g *types.GenesisState) { g.PhaseTransitions[0].LipId = "" }},
		{"NUL-phase-id", func(g *types.GenesisState) { g.PhaseTransitions[0].LipId += "\x00" }},
		{"duplicate-phase-transition", func(g *types.GenesisState) { g.PhaseTransitions = append(g.PhaseTransitions, g.PhaseTransitions[0]) }},
		{"orphan-phase-transition", func(g *types.GenesisState) { g.Lips = nil }},
		{"phase-category", func(g *types.GenesisState) { g.Lips[0].Category = types.CategoryText }},
		{"rollback-mismatch", func(g *types.GenesisState) { g.PhaseTransitions[0].IsRollback = true }},
		{"zero-target-phase", func(g *types.GenesisState) { g.PhaseTransitions[0].TargetPhase = 0 }},
		{"unknown-target-phase", func(g *types.GenesisState) { g.PhaseTransitions[0].TargetPhase = 5 }},
		{"phase-stage", func(g *types.GenesisState) { g.PhaseTransitions[0].Stage = "new" }},
		{"snapshot-balance", func(g *types.GenesisState) {
			g.PhaseTransitions[0].ConditionsSnapshot = &types.PhaseTransitionConditions{ResearchFundBalance: "-1"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gs := validRetainedGenesis()
			tc.mutate(gs)
			require.Error(t, gs.Validate())
		})
	}
}

func TestRetainedGenesisDoesNotReapplyCurrentPolicy(t *testing.T) {
	gs := validRetainedGenesis()
	gs.ResearchFundGovernance.CurrentPhase = types.ResearchFundPhase_RESEARCH_FUND_PHASE_FULL_GOVERNANCE
	gs.ResearchFundGovernance.CommunitySeats = nil
	gs.DistinctVoters[0].FirstVotedAtBlock = 0
	gs.ResearchSpendProposals[0].Stage = string(types.ResearchStageExecuted)
	gs.PhaseTransitions[0].Stage = types.PhaseTransitionStageCancelled
	gs.PhaseTransitions[0].CancelReason = "historic"
	gs.PhaseTransitions[0].ConditionsSnapshot = &types.PhaseTransitionConditions{ResearchFundBalance: "0"}
	require.NoError(t, gs.Validate(), "history does not need current eligibility or exit thresholds")
}
