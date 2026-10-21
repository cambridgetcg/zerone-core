package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateRetainedRecords validates the genesis-only FX8 inventory. It does not
// replay current eligibility, quorum, timing or phase-exit policy against history.
// In particular, former community seats and terminal records must remain valid.
func (gs *GenesisState) ValidateRetainedRecords() error {
	findings := gs.RetainedRecordFindings()
	if len(findings) > 0 {
		return findings[0].Err
	}
	return nil
}

// RetainedRecordFinding identifies the primary key implicated by an FX8
// validation failure. This is an in-memory diagnostic, not a wire/state type.
type RetainedRecordFinding struct {
	Key []byte
	Err error
}

// RetainedRecordFindings shares export's contract with the carrier inventory,
// but reports every failing record instead of hiding later keys behind the
// first error. Cross-record dependencies use the actual inventory, never a
// reconstructed counter or replay of current governance policy.
func (gs *GenesisState) RetainedRecordFindings() []RetainedRecordFinding {
	var findings []RetainedRecordFinding
	add := func(key []byte, err error) {
		findings = append(findings, RetainedRecordFinding{Key: key, Err: err})
	}
	proposals := make(map[uint64]bool, len(gs.ResearchSpendProposals))
	var maxID uint64
	for i, p := range gs.ResearchSpendProposals {
		if p == nil || p.ProposalId == 0 {
			add(ResearchSpendKey(0), fmt.Errorf("research_spend_proposals[%d]: non-nil proposal with positive id required", i))
			continue
		}
		if proposals[p.ProposalId] {
			add(ResearchSpendKey(p.ProposalId), fmt.Errorf("duplicate research spend proposal id: %d", p.ProposalId))
			continue
		}
		proposals[p.ProposalId] = true
		if p.ProposalId > maxID {
			maxID = p.ProposalId
		}
		if err := retainedAddress(p.Proposer); err != nil {
			add(ResearchSpendKey(p.ProposalId), fmt.Errorf("research proposal %d proposer: %w", p.ProposalId, err))
		}
		if err := retainedAddress(p.Recipient); err != nil {
			add(ResearchSpendKey(p.ProposalId), fmt.Errorf("research proposal %d recipient: %w", p.ProposalId, err))
		}
		amount, ok := new(big.Int).SetString(p.Amount, 10)
		if !ok || amount.Sign() <= 0 {
			add(ResearchSpendKey(p.ProposalId), fmt.Errorf("research proposal %d: amount must be positive", p.ProposalId))
		}
		switch ResearchSpendStage(p.Stage) {
		case ResearchStageDiscussion, ResearchStageVoting, ResearchStageExecuted, ResearchStageRejected, ResearchStageExpired:
		default:
			add(ResearchSpendKey(p.ProposalId), fmt.Errorf("research proposal %d: invalid stage %q", p.ProposalId, p.Stage))
		}
		for _, vote := range []string{p.Voter1Vote, p.Voter2Vote} {
			if vote != "" && vote != VoteYes && vote != VoteNo {
				add(ResearchSpendKey(p.ProposalId), fmt.Errorf("research proposal %d: invalid core ballot %q", p.ProposalId, vote))
			}
		}
	}
	// Never guess a missing counter from retained IDs, or normalize an explicit
	// zero in an empty store. Presence is part of the wire/storage contract.
	// Reserve at least half the uint64 space for future live allocations.
	if gs.NextResearchSpendNumber != nil && (*gs.NextResearchSpendNumber == 0 || *gs.NextResearchSpendNumber > 1<<63) {
		add(ResearchSpendCounterKey, fmt.Errorf("next_research_spend_number must be in [1, 9223372036854775808] with allocation headroom: stored=%d", *gs.NextResearchSpendNumber))
	}
	if len(proposals) > 0 && (gs.NextResearchSpendNumber == nil || *gs.NextResearchSpendNumber <= maxID) {
		value := "absent"
		if gs.NextResearchSpendNumber != nil {
			value = fmt.Sprint(*gs.NextResearchSpendNumber)
		}
		add(ResearchSpendCounterKey, fmt.Errorf("next_research_spend_number must be present and greater than every retained proposal id: stored=%s max_id=%d proposal_key_hex=%x", value, maxID, ResearchSpendKey(maxID)))
	}

	votes := make(map[string]bool, len(gs.ResearchCommunityVotes))
	for i, v := range gs.ResearchCommunityVotes {
		if v == nil {
			add(ResearchCommunityVotePrefix, fmt.Errorf("research_community_votes[%d]: nil record", i))
			continue
		}
		key := ResearchCommunityVoteKey(v.ProposalId, v.Voter)
		if !proposals[v.ProposalId] {
			add(key, fmt.Errorf("research community vote references missing proposal %d", v.ProposalId))
		}
		if err := retainedAddress(v.Voter); err != nil {
			add(key, fmt.Errorf("research community voter: %w", err))
		}
		if v.Vote != VoteYes && v.Vote != VoteNo {
			add(key, fmt.Errorf("invalid research community ballot %q", v.Vote))
		}
		if votes[string(key)] {
			add(key, fmt.Errorf("duplicate research community vote for proposal %d voter %s", v.ProposalId, v.Voter))
		}
		votes[string(key)] = true
	}

	voters := make(map[string]bool, len(gs.DistinctVoters))
	for i, v := range gs.DistinctVoters {
		if v == nil {
			add(DistinctVoterKeyPrefix, fmt.Errorf("distinct_voters[%d]: nil record", i))
			continue
		}
		if err := retainedAddress(v.Voter); err != nil {
			add(DistinctVoterKey(v.Voter), fmt.Errorf("distinct voter: %w", err))
		}
		if voters[v.Voter] {
			add(DistinctVoterKey(v.Voter), fmt.Errorf("duplicate distinct voter: %s", v.Voter))
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
			key := PhaseTransitionKeyPrefix
			if p != nil {
				key = PhaseTransitionKey(p.LipId)
			}
			add(key, fmt.Errorf("phase_transitions[%d]: non-nil record with non-empty, NUL-free lip_id required", i))
			continue
		}
		if transitions[p.LipId] {
			add(PhaseTransitionKey(p.LipId), fmt.Errorf("duplicate phase transition: %s", p.LipId))
			continue
		}
		transitions[p.LipId] = true
		lip := lips[p.LipId]
		if lip == nil || !IsPhaseTransitionCategory(lip.Category) || p.IsRollback != (lip.Category == CategoryPhaseRollback) {
			add(PhaseTransitionKey(p.LipId), fmt.Errorf("phase transition %s does not match its LIP/category", p.LipId))
		}
		if p.TargetPhase < ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR || p.TargetPhase > ResearchFundPhase_RESEARCH_FUND_PHASE_FULL_GOVERNANCE {
			add(PhaseTransitionKey(p.LipId), fmt.Errorf("phase transition %s: invalid target phase", p.LipId))
		}
		switch p.Stage {
		case PhaseTransitionStageDiscussion, PhaseTransitionStageVoting, PhaseTransitionStagePassed, PhaseTransitionStageFailed,
			PhaseTransitionStagePending, PhaseTransitionStageActivated, PhaseTransitionStageCancelled:
		default:
			add(PhaseTransitionKey(p.LipId), fmt.Errorf("phase transition %s: invalid stage %q", p.LipId, p.Stage))
		}
		if p.ConditionsSnapshot != nil && p.ConditionsSnapshot.ResearchFundBalance != "" {
			balance, ok := new(big.Int).SetString(p.ConditionsSnapshot.ResearchFundBalance, 10)
			if !ok || balance.Sign() < 0 {
				add(PhaseTransitionKey(p.LipId), fmt.Errorf("phase transition %s: invalid snapshot balance", p.LipId))
			}
		}
	}
	return findings
}

// RetainedGenesisFields are the six presence-selected FX8 JSON fields. Return a
// fresh slice so tooling cannot change the application validation contract.
func RetainedGenesisFields() []string {
	return []string{"lips", "research_spend_proposals", "next_research_spend_number", "research_community_votes", "distinct_voters", "phase_transitions"}
}

// ValidateRetainedGenesisFieldNames runs before encoding/json can silently let
// a case-fold alias replace retained history. Missing fields remain permitted.
func ValidateRetainedGenesisFieldNames(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("FX8 governance genesis: %w", err)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, name := range RetainedGenesisFields() {
		parts := strings.Split(name, "_")
		for i := 1; i < len(parts); i++ {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
		camel := strings.Join(parts, "")
		for _, key := range keys {
			if key != name && (strings.EqualFold(key, name) || strings.EqualFold(key, camel)) {
				return fmt.Errorf("FX8 alias %q forbidden; use %s", key, name)
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
