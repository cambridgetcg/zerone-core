package types

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateRetainedRecords validates the genesis-only FX8 inventory. It does not
// replay current eligibility, quorum, timing or phase-exit policy against history.
// In particular, former community seats and terminal records must remain valid.
func (gs *GenesisState) ValidateRetainedRecords() error {
	proposals := make(map[uint64]bool, len(gs.ResearchSpendProposals))
	var maxID uint64
	for i, p := range gs.ResearchSpendProposals {
		if p == nil || p.ProposalId == 0 {
			return fmt.Errorf("research_spend_proposals[%d]: non-nil proposal with positive id required", i)
		}
		if proposals[p.ProposalId] {
			return fmt.Errorf("duplicate research spend proposal id: %d", p.ProposalId)
		}
		proposals[p.ProposalId] = true
		if p.ProposalId > maxID {
			maxID = p.ProposalId
		}
		if err := retainedAddress(p.Proposer); err != nil {
			return fmt.Errorf("research proposal %d proposer: %w", p.ProposalId, err)
		}
		if err := retainedAddress(p.Recipient); err != nil {
			return fmt.Errorf("research proposal %d recipient: %w", p.ProposalId, err)
		}
		amount, ok := new(big.Int).SetString(p.Amount, 10)
		if !ok || amount.Sign() <= 0 {
			return fmt.Errorf("research proposal %d: amount must be positive", p.ProposalId)
		}
		switch ResearchSpendStage(p.Stage) {
		case ResearchStageDiscussion, ResearchStageVoting, ResearchStageExecuted, ResearchStageRejected, ResearchStageExpired:
		default:
			return fmt.Errorf("research proposal %d: invalid stage %q", p.ProposalId, p.Stage)
		}
		for _, vote := range []string{p.Voter1Vote, p.Voter2Vote} {
			if vote != "" && vote != VoteYes && vote != VoteNo {
				return fmt.Errorf("research proposal %d: invalid core ballot %q", p.ProposalId, vote)
			}
		}
	}
	// Never guess a missing counter from retained IDs, or normalize an explicit
	// zero in an empty store. Presence is part of the wire/storage contract.
	// Zero is not a valid proposal ID; MaxUint64 wraps the next live allocation.
	if gs.NextResearchSpendNumber != nil && (*gs.NextResearchSpendNumber == 0 || *gs.NextResearchSpendNumber == ^uint64(0)) {
		return fmt.Errorf("next_research_spend_number must be positive with allocation headroom")
	}
	if len(proposals) > 0 && (gs.NextResearchSpendNumber == nil || *gs.NextResearchSpendNumber <= maxID) {
		return fmt.Errorf("next_research_spend_number must be present and greater than every retained proposal id")
	}

	votes := make(map[string]bool, len(gs.ResearchCommunityVotes))
	for i, v := range gs.ResearchCommunityVotes {
		if v == nil {
			return fmt.Errorf("research_community_votes[%d]: nil record", i)
		}
		if !proposals[v.ProposalId] {
			return fmt.Errorf("research community vote references missing proposal %d", v.ProposalId)
		}
		if err := retainedAddress(v.Voter); err != nil {
			return fmt.Errorf("research community voter: %w", err)
		}
		if v.Vote != VoteYes && v.Vote != VoteNo {
			return fmt.Errorf("invalid research community ballot %q", v.Vote)
		}
		key := string(ResearchCommunityVoteKey(v.ProposalId, v.Voter))
		if votes[key] {
			return fmt.Errorf("duplicate research community vote for proposal %d voter %s", v.ProposalId, v.Voter)
		}
		votes[key] = true
	}

	voters := make(map[string]bool, len(gs.DistinctVoters))
	for i, v := range gs.DistinctVoters {
		if v == nil {
			return fmt.Errorf("distinct_voters[%d]: nil record", i)
		}
		if err := retainedAddress(v.Voter); err != nil {
			return fmt.Errorf("distinct voter: %w", err)
		}
		if voters[v.Voter] {
			return fmt.Errorf("duplicate distinct voter: %s", v.Voter)
		}
		voters[v.Voter] = true
	}

	lips := make(map[string]*LIP)
	if len(gs.PhaseTransitions) > 0 {
		for _, lip := range gs.Lips {
			if lip != nil {
				lips[lip.Id] = lip
			}
		}
	}
	transitions := make(map[string]bool, len(gs.PhaseTransitions))
	for i, p := range gs.PhaseTransitions {
		if p == nil || p.LipId == "" || strings.ContainsRune(p.LipId, '\x00') {
			return fmt.Errorf("phase_transitions[%d]: non-nil record with non-empty, NUL-free lip_id required", i)
		}
		if transitions[p.LipId] {
			return fmt.Errorf("duplicate phase transition: %s", p.LipId)
		}
		transitions[p.LipId] = true
		lip := lips[p.LipId]
		if lip == nil || !IsPhaseTransitionCategory(lip.Category) || p.IsRollback != (lip.Category == CategoryPhaseRollback) {
			return fmt.Errorf("phase transition %s does not match its LIP/category", p.LipId)
		}
		if p.TargetPhase < ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR || p.TargetPhase > ResearchFundPhase_RESEARCH_FUND_PHASE_FULL_GOVERNANCE {
			return fmt.Errorf("phase transition %s: invalid target phase", p.LipId)
		}
		switch p.Stage {
		case PhaseTransitionStageDiscussion, PhaseTransitionStageVoting, PhaseTransitionStagePassed, PhaseTransitionStageFailed,
			PhaseTransitionStagePending, PhaseTransitionStageActivated, PhaseTransitionStageCancelled:
		default:
			return fmt.Errorf("phase transition %s: invalid stage %q", p.LipId, p.Stage)
		}
		if p.ConditionsSnapshot != nil && p.ConditionsSnapshot.ResearchFundBalance != "" {
			balance, ok := new(big.Int).SetString(p.ConditionsSnapshot.ResearchFundBalance, 10)
			if !ok || balance.Sign() < 0 {
				return fmt.Errorf("phase transition %s: invalid snapshot balance", p.LipId)
			}
		}
	}
	return nil
}

func retainedAddress(address string) error {
	if _, err := sdk.AccAddressFromBech32(address); err != nil {
		return fmt.Errorf("invalid address %q: %w", address, err)
	}
	return nil
}
