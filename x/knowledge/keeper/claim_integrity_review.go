package keeper

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/bech32"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// claimParty is an account that may not review a claim, and the reason.
type claimParty struct {
	address string
	role    string
}

// refuseConflictedReviewer refuses a commitment from a party to round's claim
// (knowledge v12, plan Task 14; P5). StoreCommitmentInRound calls it only with
// claim integrity on, so MsgSubmitCommitment and the vote-extension injection
// are both covered. Reveals are not guarded: a commitment admitted before
// activation still reveals and settles.
//
// A record that cannot be read refuses the commitment; it is never read as
// "no party". An absent record names no party, as getClaimRecordChecked
// treats an absent historical claim: a legacy round may have none, and a
// neutral round without its claim is refused when the round is saved.
func (k Keeper) refuseConflictedReviewer(ctx context.Context, round *types.VerificationRound, reviewer string) error {
	claim, err := k.getClaimRecordChecked(ctx, round.ClaimId)
	if err != nil {
		return fmt.Errorf("read claim %s of round %s: %w", round.ClaimId, round.Id, err)
	}
	if claim == nil {
		return nil
	}
	parties, err := k.conflictedParties(ctx, claim)
	if err != nil {
		return err
	}
	for _, party := range parties {
		if sameActor(reviewer, party.address) {
			return types.ErrUnqualifiedVerifier.Wrapf("reviewer %s is a party to claim %s (%s)", reviewer, claim.Id, party.role)
		}
	}
	return nil
}

// conflictedParties names the accounts that may not sit on claim's panel:
//   - the claim's submitter;
//   - the defender of the one fact the claim disputes, meaning that fact's
//     submitter: the ProvisionalFactId of a MsgChallengeFact or
//     MsgChallengeProvisionalFact claim, or else the single CONTRADICTS target
//     of a CHALLENGE_DEPOSIT claim with exactly one relation, which only
//     MsgSubmitContradiction creates.
//
// A MsgSubmitClaim's relation list is chosen by its submitter and is never
// read here, so naming facts cannot recuse their defenders (plan §10). An
// unreadable disputed fact is an error; an absent one names no defender.
func (k Keeper) conflictedParties(ctx context.Context, claim *types.Claim) ([]claimParty, error) {
	parties := []claimParty{{address: claim.Submitter, role: "submitter"}}
	target := disputedFactID(claim)
	if target == "" {
		return parties, nil
	}
	fact, found, err := k.getFactChecked(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("read disputed fact %s of claim %s: %w", target, claim.Id, err)
	}
	if !found {
		return parties, nil
	}
	return append(parties, claimParty{address: fact.Submitter, role: "defender of fact " + target}), nil
}

// disputedFactID is the one fact claim disputes, or "" when it disputes none
// or names its targets itself.
func disputedFactID(claim *types.Claim) string {
	if claim.ProvisionalFactId != "" {
		return claim.ProvisionalFactId
	}
	terms := claim.FundingTerms
	if terms == nil || terms.Kind != types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT || len(claim.Relations) != 1 {
		return ""
	}
	if relation := claim.Relations[0]; relation != nil && relation.Relation == types.RelationType_RELATION_TYPE_CONTRADICTS {
		return relation.TargetFactId
	}
	return ""
}

// sameActor reports whether two address strings name one account: the same
// non-empty string, or bech32 strings whose data bytes are equal under any
// prefix (an account and its operator encoding, or an upper-case copy).
func sameActor(a, b string) bool {
	if a == b {
		return a != ""
	}
	_, x, err := bech32.DecodeAndConvert(a)
	if err != nil || len(x) == 0 {
		return false
	}
	_, y, err := bech32.DecodeAndConvert(b)
	return err == nil && bytes.Equal(x, y)
}
