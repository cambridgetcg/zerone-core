package ibc_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
)

// TestFrozenFeeGranterRejectedBeforeDeduction proves the app-level ante order,
// not just the isolated decorator: an allowance created before a granter is
// frozen cannot be consumed, and failed use does not move fees or delete the
// allowance. Unfreezing restores the standard feegrant path.
func (s *IBCTestSuite) TestFrozenFeeGranterRejectedBeforeDeduction() {
	app := GetZeroneApp(s.chainA)
	granter := s.chainA.SenderAccounts[1].SenderAccount.GetAddress()
	grantee := s.chainA.SenderAccount.GetAddress()
	recipient := s.chainA.SenderAccounts[2].SenderAccount.GetAddress()
	ctx := s.chainA.GetContext()

	app.ZeroneAuthKeeper.SetAccount(ctx, &zeroneauthtypes.Account{
		Address:     granter.String(),
		Did:         "did:zrn:abcdef0123456789abcdef0123456789",
		AccountType: "agent",
		Flags:       &zeroneauthtypes.AccountFlags{Frozen: true},
	})

	allowance, err := app.FeeGrantKeeper.GetAllowance(ctx, granter, grantee)
	s.Require().NoError(err)
	s.Require().NotNil(allowance)
	balanceBefore := app.BankKeeper.GetBalance(ctx, granter, sdk.DefaultBondDenom)

	msg := banktypes.NewMsgSend(
		grantee,
		recipient,
		sdk.NewCoins(
			sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1)),
		),
	)
	_, err = s.chainA.SendMsgs(msg)
	s.Require().ErrorContains(err, "account is frozen")

	ctx = s.chainA.GetContext()
	balanceAfterFailure := app.BankKeeper.GetBalance(
		ctx,
		granter,
		sdk.DefaultBondDenom,
	)
	s.Require().Equal(balanceBefore, balanceAfterFailure)
	allowance, err = app.FeeGrantKeeper.GetAllowance(ctx, granter, grantee)
	s.Require().NoError(err)
	s.Require().NotNil(allowance)

	account, found := app.ZeroneAuthKeeper.GetAccount(ctx, granter.String())
	s.Require().True(found)
	account.Flags.Frozen = false
	app.ZeroneAuthKeeper.SetAccount(ctx, account)
	signerAccount := app.AccountKeeper.GetAccount(ctx, grantee)
	s.Require().NotNil(signerAccount)
	s.Require().NoError(
		s.chainA.SenderAccount.SetSequence(signerAccount.GetSequence()),
	)

	_, err = s.chainA.SendMsgs(msg)
	s.Require().NoError(err)
	balanceAfterSuccess := app.BankKeeper.GetBalance(
		s.chainA.GetContext(),
		granter,
		sdk.DefaultBondDenom,
	)
	s.Require().True(
		balanceAfterSuccess.Amount.LT(balanceAfterFailure.Amount),
		"unfreezing should restore fee deduction through the existing allowance",
	)
}
