package types

import "fmt"

// ValidateGenesisRounds checks both retained primary-round classes. Legacy
// records remain scheme 0; enabling a native/imported genesis does not upgrade
// their payload, review confidence or historical payment representation.
func ValidateGenesisRounds(gs *GenesisState) error {
	claims := make(map[string]*Claim, len(gs.PendingClaims))
	for _, claim := range gs.PendingClaims {
		if claim == nil || claim.Id == "" {
			return fmt.Errorf("genesis claim requires nonempty identity")
		}
		if _, exists := claims[claim.Id]; exists {
			return fmt.Errorf("duplicate claim ID: %s", claim.Id)
		}
		claims[claim.Id] = claim
	}
	seen := make(map[string]*VerificationRound, len(gs.ActiveRounds)+len(gs.CompletedRounds))
	counts := make(map[string]int)
	for _, group := range []struct {
		rounds   []*VerificationRound
		terminal bool
	}{{gs.ActiveRounds, false}, {gs.CompletedRounds, true}} {
		for _, round := range group.rounds {
			if err := ValidateVerificationRoundRecord(round, gs.RecordIntegrityEnabled); err != nil {
				return err
			}
			if seen[round.Id] != nil {
				return fmt.Errorf("duplicate verification round ID: %s", round.Id)
			}
			seen[round.Id] = round
			counts[round.ClaimId]++
			terminal := round.Phase == VerificationPhase_VERIFICATION_PHASE_COMPLETE || round.Phase == VerificationPhase_VERIFICATION_PHASE_EXPIRED
			if terminal != group.terminal {
				return fmt.Errorf("genesis round %s is in wrong active/completed collection", round.Id)
			}
			if claims[round.ClaimId] == nil {
				return fmt.Errorf("genesis round %s references missing claim %s", round.Id, round.ClaimId)
			}
		}
	}
	// Old exports omitted terminal rounds. An absent legacy reverse reference
	// remains unknown history; never synthesize that missing record. If the round
	// is supplied, its primary claim identity must agree with the claim pointer.
	for _, claim := range claims {
		round := seen[claim.VerificationRoundId]
		if counts[claim.Id] > 1 && round == nil {
			return fmt.Errorf("multiple genesis rounds require an explicit selected round for claim %s", claim.Id)
		}
		if claim.VerificationRoundId == "" || round == nil {
			continue
		}
		if round.ClaimId != claim.Id {
			return fmt.Errorf("genesis claim/round reference mismatch")
		}
	}
	return nil
}
