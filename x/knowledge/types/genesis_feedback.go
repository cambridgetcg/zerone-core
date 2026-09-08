package types

import "fmt"

// validateFeedbackGenesis checks the new canonical records without fabricating
// absent history. Keeper import must restore raw sequence values/records rather
// than calling allocating writers, and rebuild only derived indexes.
func (gs *GenesisState) validateFeedbackGenesis() error {
	relations := make(map[string]bool, len(gs.FactRelations))
	for _, rel := range gs.FactRelations {
		if rel == nil || rel.SourceFactId == "" || rel.TargetFactId == "" {
			return fmt.Errorf("fact relation requires source and target")
		}
		if err := ValidateFactUseFactID(rel.SourceFactId); err != nil {
			return err
		}
		if err := ValidateFactUseFactID(rel.TargetFactId); err != nil {
			return err
		}
		key := string(FactRelationKey(rel.SourceFactId, rel.TargetFactId))
		if relations[key] {
			return fmt.Errorf("duplicate canonical fact relation key")
		}
		relations[key] = true
	}

	transitions := make(map[string]bool, len(gs.StatusTransitions))
	maxSequences := make(map[string]uint64)
	for _, tr := range gs.StatusTransitions {
		if tr == nil || tr.Seq == 0 {
			return fmt.Errorf("status transition requires a nonzero sequence")
		}
		if err := ValidateFactUseFactID(tr.FactId); err != nil {
			return err
		}
		key := string(StatusTransitionKey(tr.FactId, tr.Seq))
		if transitions[key] {
			return fmt.Errorf("duplicate status transition key")
		}
		transitions[key] = true
		if tr.Seq > maxSequences[tr.FactId] {
			maxSequences[tr.FactId] = tr.Seq
		}
	}
	sequences := make(map[string]uint64, len(gs.StatusTransitionSequences))
	for _, seq := range gs.StatusTransitionSequences {
		if seq == nil {
			return fmt.Errorf("nil status transition sequence")
		}
		if err := ValidateFactUseFactID(seq.FactId); err != nil {
			return err
		}
		if _, exists := sequences[seq.FactId]; exists {
			return fmt.Errorf("duplicate status transition sequence counter")
		}
		sequences[seq.FactId] = seq.LastSequence
		if seq.LastSequence < maxSequences[seq.FactId] {
			return fmt.Errorf("status transition counter precedes stored history")
		}
	}
	// Check in input order so malformed genesis reports deterministic errors.
	for _, tr := range gs.StatusTransitions {
		if _, exists := sequences[tr.FactId]; !exists {
			return fmt.Errorf("status transition history lacks its stored sequence counter")
		}
	}

	cascades := make(map[string]bool, len(gs.CascadeEvents))
	for _, ev := range gs.CascadeEvents {
		if ev == nil || ev.Seq == 0 {
			return fmt.Errorf("cascade event requires a nonzero sequence")
		}
		if err := ValidateFactUseFactID(ev.DisprovenFactId); err != nil {
			return err
		}
		if err := ValidateFactUseFactID(ev.DescendantFactId); err != nil {
			return err
		}
		key := string(CascadeEventKey(ev.DisprovenFactId, ev.Seq))
		if cascades[key] {
			return fmt.Errorf("duplicate cascade event key")
		}
		cascades[key] = true
	}

	activeRounds := make(map[string]bool, len(gs.ActiveRounds))
	for _, round := range gs.ActiveRounds {
		if round == nil || round.Id == "" || activeRounds[round.Id] {
			return fmt.Errorf("nil, empty or duplicate active round")
		}
		activeRounds[round.Id] = true
	}
	completedRounds := make(map[string]*VerificationRound, len(gs.CompletedRounds))
	for _, round := range gs.CompletedRounds {
		if round == nil || round.Id == "" || activeRounds[round.Id] || completedRounds[round.Id] != nil {
			return fmt.Errorf("nil, empty or duplicate completed round")
		}
		if round.Phase != VerificationPhase_VERIFICATION_PHASE_COMPLETE && round.Phase != VerificationPhase_VERIFICATION_PHASE_EXPIRED {
			return fmt.Errorf("completed_rounds contains a nonterminal round")
		}
		completedRounds[round.Id] = round
	}
	completions := make(map[string]bool, len(gs.CompletedRoundRecords))
	for _, record := range gs.CompletedRoundRecords {
		if record == nil || record.RoundId == "" || record.Meta == nil {
			return fmt.Errorf("completed round record requires key and metadata")
		}
		// A round can complete only once, even though the storage key also
		// includes its height. Distinct heights must not disguise a conflict.
		if completions[record.RoundId] {
			return fmt.Errorf("duplicate completed round record for round %s", record.RoundId)
		}
		completions[record.RoundId] = true
		if activeRounds[record.RoundId] {
			return fmt.Errorf("completed round record refers to active round %s", record.RoundId)
		}
		// Supplied rounds were checked terminal above. Standalone historical
		// metadata remains valid when its round is absent; never reconstruct it.
		if round := completedRounds[record.RoundId]; round != nil && round.VerdictBlock != record.VerdictBlock {
			return fmt.Errorf("completed round record height conflicts with round %s", record.RoundId)
		}
	}

	if err := ValidateFactUseNonEconomicState(gs.Params, gs.FactUsePruning, len(gs.FactUseReceipts) != 0); err != nil {
		return err
	}
	if len(gs.FactUseReceipts) > MaxFactUseRetainedReceipts {
		return fmt.Errorf("fact-use receipts exceed retained capacity %d", MaxFactUseRetainedReceipts)
	}
	receipts := make(map[string]bool, len(gs.FactUseReceipts))
	epochCounts := make(map[uint64]uint64)
	consumerCounts := make(map[string]uint64)
	for _, receipt := range gs.FactUseReceipts {
		if err := receipt.Validate(gs.Params.FitnessEpochBlocks); err != nil {
			return fmt.Errorf("invalid genesis fact-use receipt: %w", err)
		}
		key := string(FactUseReceiptKey(receipt.Epoch, receipt.Consumer, receipt.FactId))
		if receipts[key] {
			return fmt.Errorf("duplicate fact-use receipt key")
		}
		receipts[key] = true
		epochCounts[receipt.Epoch]++
		consumerKey := string(FactUseConsumerCountKey(receipt.Epoch, receipt.Consumer))
		consumerCounts[consumerKey]++
		// Preserve reports admitted before governance lowered a quota: use the
		// immutable hard ceilings, not the currently configured admission cap.
		if epochCounts[receipt.Epoch] > MaxFactUsePerEpoch || consumerCounts[consumerKey] > MaxFactUsePerConsumerEpoch {
			return fmt.Errorf("genesis fact-use receipts exceed epoch hard ceilings")
		}
	}
	if gs.FactUsePruning != nil && len(gs.FactUsePruning.NextKey) != 0 {
		if _, _, _, err := ParseFactUseReceiptKey(gs.FactUsePruning.NextKey); err != nil {
			return fmt.Errorf("invalid fact-use pruning cursor: %w", err)
		}
	}
	return nil
}
