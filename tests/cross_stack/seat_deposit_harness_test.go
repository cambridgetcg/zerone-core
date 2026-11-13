package cross_stack_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
)

// AdmitReviewer registers addr with zerone_auth so that it may review, as
// the ante requires of a signed MsgSubmitCommitment and as a round with seat
// terms checks in its handler (reviewer-entry step B, D3). Native genesis
// selects the seat deposit, so a test that commits through the message
// handler admits its reviewers first and funds their deposits. The account
// declares no type, so what reads an account type (role bonus, calibration)
// reads "" for it, as for an unregistered address.
func (h *TestHarness) AdmitReviewer(addr sdk.AccAddress) {
	h.T.Helper()
	h.AuthKeeper.SetAccount(h.Ctx, &zeroneauthtypes.Account{Address: addr.String(), Flags: &zeroneauthtypes.AccountFlags{CanSubmitClaims: true}})
}
