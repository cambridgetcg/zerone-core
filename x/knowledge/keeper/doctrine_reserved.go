package keeper

import (
	"context"
	"errors"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ErrDoctrineCategoryReserved refuses a claim that names the doctrine category.
// With the doctrine-identity marker on, doctrine is what the chain's own
// process wrote (types.BuildDoctrineFact at genesis, MsgAddFact, a matured
// guardian-veto injection), so no claim may carry its label.
var ErrDoctrineCategoryReserved = errors.New(`category "doctrine" is reserved for doctrine written by the chain's authority`)

// refuseReservedDoctrineCategory is the admission rule for the doctrine label.
// It reads the doctrine-identity marker once and returns a marker fault rather
// than choose. With the marker on it refuses exactly types.DoctrineCategory;
// an empty category and every other value pass, as before. With the marker
// off it refuses nothing.
//
// Callers: PostConjecture and SubmitContradiction at entry, before either
// branches, and the legacy SubmitClaim path (record integrity off). With
// record integrity on, SubmitClaim already refuses "doctrine": it is not one
// of types.ValidAxiomCategories (validateClaimRecordInput), so that path does
// not read the marker.
func (k Keeper) refuseReservedDoctrineCategory(ctx context.Context, category string) error {
	enabled, err := k.DoctrineIdentityEnabled(ctx)
	if err != nil {
		return err
	}
	if enabled && category == types.DoctrineCategory {
		return ErrDoctrineCategoryReserved
	}
	return nil
}

// challengeClaimCategory is the category a challenge claim takes from the fact
// it challenges. ChallengeFact and ChallengeProvisionalFact copy the target's
// category, so before this repair an accepted challenge of doctrine made a new
// fact labelled doctrine. With the doctrine-identity marker on, a target
// labelled doctrine gives its challenge an empty category; every other
// category is still copied. Challenging doctrine stays possible.
func challengeClaimCategory(target *types.Fact, doctrineIdentity bool) string {
	if doctrineIdentity && target.Category == types.DoctrineCategory {
		return ""
	}
	return target.Category
}
