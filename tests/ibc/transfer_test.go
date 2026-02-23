package ibc_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// TestBasicTransferAToB sends uzrn from chain A to chain B and verifies the
// IBC voucher denom appears on B.
func (s *IBCTestSuite) TestBasicTransferAToB() {
	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_000_000))

	// Record balances before.
	appA := GetZeroneApp(s.chainA)
	balBefore := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)

	// Build and send MsgTransfer.
	timeoutHeight := clienttypes.NewHeight(1, 110)
	msg := transfertypes.NewMsgTransfer(
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		timeoutHeight,
		0, // no timestamp timeout
		"",
	)
	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err)

	// Parse packet from events and relay.
	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)
	err = s.transferPath.RelayPacket(packet)
	s.Require().NoError(err)

	// Verify sender balance decreased.
	balAfter := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
	s.Require().True(balAfter.Amount.LT(balBefore.Amount), "sender balance should decrease")

	// Verify receiver got IBC voucher on chain B.
	ibcDenom := transfertypes.GetPrefixedDenom(
		s.transferPath.EndpointB.ChannelConfig.PortID,
		s.transferPath.EndpointB.ChannelID,
		sdk.DefaultBondDenom,
	)
	voucherDenom := transfertypes.ParseDenomTrace(ibcDenom).IBCDenom()

	appB := GetZeroneApp(s.chainB)
	balB := appB.BankKeeper.GetBalance(s.chainB.GetContext(), receiver, voucherDenom)
	s.Require().Equal(amount.Amount, balB.Amount, "receiver should have IBC tokens on chain B")
}

// TestReturnTransferBToA sends IBC vouchers from chain B back to chain A and
// verifies the original denom is restored.
func (s *IBCTestSuite) TestReturnTransferBToA() {
	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	transferAmount := sdkmath.NewInt(1_000_000)
	amount := sdk.NewCoin(sdk.DefaultBondDenom, transferAmount)

	// Step 1: Transfer A -> B.
	timeoutHeight := clienttypes.NewHeight(1, 110)
	msg := transfertypes.NewMsgTransfer(
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		timeoutHeight,
		0,
		"",
	)
	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err)
	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)
	err = s.transferPath.RelayPacket(packet)
	s.Require().NoError(err)

	// Step 2: Return B -> A with the IBC voucher.
	ibcDenom := transfertypes.GetPrefixedDenom(
		s.transferPath.EndpointB.ChannelConfig.PortID,
		s.transferPath.EndpointB.ChannelID,
		sdk.DefaultBondDenom,
	)
	voucherDenom := transfertypes.ParseDenomTrace(ibcDenom).IBCDenom()

	appA := GetZeroneApp(s.chainA)
	balBeforeReturn := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)

	returnMsg := transfertypes.NewMsgTransfer(
		s.transferPath.EndpointB.ChannelConfig.PortID,
		s.transferPath.EndpointB.ChannelID,
		sdk.NewCoin(voucherDenom, transferAmount),
		receiver.String(),
		sender.String(),
		timeoutHeight,
		0,
		"",
	)
	res, err = s.chainB.SendMsgs(returnMsg)
	s.Require().NoError(err)
	packet, err = ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)
	err = s.transferPath.RelayPacket(packet)
	s.Require().NoError(err)

	// Verify original denom restored on chain A.
	balAfterReturn := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
	s.Require().True(balAfterReturn.Amount.GT(balBeforeReturn.Amount),
		"sender should get original denom back on chain A")
}

// TestDenomTrace verifies the IBC denom trace format on the receiving chain.
func (s *IBCTestSuite) TestDenomTrace() {
	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(500_000))

	timeoutHeight := clienttypes.NewHeight(1, 110)
	msg := transfertypes.NewMsgTransfer(
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		timeoutHeight,
		0,
		"",
	)
	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err)
	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)
	err = s.transferPath.RelayPacket(packet)
	s.Require().NoError(err)

	// Verify denom trace on chain B.
	expectedTrace := transfertypes.DenomTrace{
		Path:      s.transferPath.EndpointB.ChannelConfig.PortID + "/" + s.transferPath.EndpointB.ChannelID,
		BaseDenom: sdk.DefaultBondDenom,
	}

	appB := GetZeroneApp(s.chainB)
	traces := appB.TransferKeeper.GetAllDenomTraces(s.chainB.GetContext())
	s.Require().NotEmpty(traces, "should have at least one denom trace")

	found := false
	for _, trace := range traces {
		if trace.BaseDenom == expectedTrace.BaseDenom && trace.Path == expectedTrace.Path {
			found = true
			break
		}
	}
	s.Require().True(found, "expected denom trace %s not found in %v", expectedTrace, traces)
}
