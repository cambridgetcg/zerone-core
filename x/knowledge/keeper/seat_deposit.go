package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Reviewer seat deposit: step B of the reviewer-entry design draft
// (docs/specs/knowledge-reviewer-entry-v1.md, pull request #190), rules D1 to
// D7. Everything here is selected by the seat-deposit marker
// (seat_deposit_activation.go), read once per entry point without gas. With
// the marker off no round records seat terms, so no code below runs past its
// first check and every message, record, event and gas figure is what
// 3a844c48 produces.

// setRoundSeatTerms records a new round's seat terms (D1): while the marker is
// on, a round under review policy 1 with commitment scheme 2 copies the
// counted-seat deposit from min_reviewer_balance as it stands at creation. The
// figure is ParseMinReviewerBalance's, not EffectiveMinReviewerBalance's: a
// figure below 100 ZRN is what a locked deposit allows. An empty parameter
// locks the built-in 100 ZRN. A later parameter change does not alter the
// round: ValidateSeatDepositUpdate keeps terms immutable. Policy-0 rounds,
// and every round created while the marker is off, get no terms and keep the
// balance check (A5, D7).
//
// params is the set CreateVerificationRound already loaded, so this adds no
// param read. GetParams returns DefaultParams when the stored set fails to
// decode, so such a chain locks 100 ZRN, not the governed figure (the same
// caveat as reviewerMinBalance).
func (k Keeper) setRoundSeatTerms(ctx context.Context, round *types.VerificationRound, params *types.Params) error {
	enabled, err := k.SeatDepositEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled || !types.RoundSeatTermsAllowed(round) {
		return nil
	}
	// The deposit is the governable figure, so its marker must be on too;
	// an inconsistent activation fails closed instead of locking a figure
	// that governance cannot set.
	paramMarker, err := k.ReviewerBalanceParamEnabled(ctx)
	if err != nil {
		return err
	}
	if !paramMarker {
		return fmt.Errorf("seat deposits require the reviewer-balance-param activation")
	}
	if params == nil {
		return fmt.Errorf("seat terms require the knowledge params")
	}
	amount, err := types.ParseMinReviewerBalance(params.MinReviewerBalance)
	if err != nil {
		return fmt.Errorf("stored params: %w", err)
	}
	round.SeatTerms = &types.ReviewSeatTerms{CountedSeatDeposit: types.FormatSeatDeposit(amount)}
	return nil
}
