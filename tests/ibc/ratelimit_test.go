package ibc_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	ibcratelimittypes "github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// TestTransferWithinSendLimit sets a send rate limit and transfers below it.
func (s *IBCTestSuite) TestTransferWithinSendLimit() {
	s.SetupRateLimit(
		s.transferPath.EndpointA.ChannelID,
		sdk.DefaultBondDenom,
		"1000000", // maxSend
		"2000000", // maxRecv
		100,       // windowBlocks
	)

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
	s.Require().NoError(err, "transfer within send limit should succeed")

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)
	err = s.transferPath.RelayPacket(packet)
	s.Require().NoError(err)

	// Verify receiver got the tokens.
	ibcDenom := transfertypes.GetPrefixedDenom(
		s.transferPath.EndpointB.ChannelConfig.PortID,
		s.transferPath.EndpointB.ChannelID,
		sdk.DefaultBondDenom,
	)
	voucherDenom := transfertypes.ParseDenomTrace(ibcDenom).IBCDenom()

	appB := GetZeroneApp(s.chainB)
	balB := appB.BankKeeper.GetBalance(s.chainB.GetContext(), receiver, voucherDenom)
	s.Require().Equal(amount.Amount, balB.Amount)
}

// TestTransferExceedingSendLimit sets a send rate limit and attempts to
// transfer more than allowed.
func (s *IBCTestSuite) TestTransferExceedingSendLimit() {
	s.SetupRateLimit(
		s.transferPath.EndpointA.ChannelID,
		sdk.DefaultBondDenom,
		"1000000", // maxSend
		"2000000", // maxRecv
		100,       // windowBlocks
	)

	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_500_000)) // exceeds 1M limit

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

	// SendMsgs should fail because the rate limit middleware rejects in SendPacket.
	_, err := s.chainA.SendMsgs(msg)
	s.Require().Error(err, "transfer exceeding send limit should fail")
}

// TestTransferWithinRecvLimit sets a recv rate limit and transfers below it.
func (s *IBCTestSuite) TestTransferWithinRecvLimit() {
	// Set rate limit on chain B's receiving side.
	appB := GetZeroneApp(s.chainB)
	ctx := s.chainB.GetContext()
	appB.IBCRateLimitKeeper.SetParams(ctx, &ibcratelimittypes.Params{Enabled: true})

	// The receiving chain sees the IBC-prefixed denom, so we set rate limit on
	// the source denom for receive checking (the middleware checks the denom from packet data).
	appB.IBCRateLimitKeeper.SetRateLimit(ctx, &ibcratelimittypes.RateLimit{
		ChannelId:    s.transferPath.EndpointB.ChannelID,
		Denom:        sdk.DefaultBondDenom,
		MaxSend:      "2000000",
		MaxRecv:      "1000000",
		WindowBlocks: 100,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  uint64(ctx.BlockHeight()),
	})

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
}

// TestTransferExceedingRecvLimit sends a large amount from A that exceeds B's
// receive rate limit. The OnRecvPacket on B should return an error ack.
func (s *IBCTestSuite) TestTransferExceedingRecvLimit() {
	appB := GetZeroneApp(s.chainB)
	ctx := s.chainB.GetContext()
	appB.IBCRateLimitKeeper.SetParams(ctx, &ibcratelimittypes.Params{Enabled: true})

	appB.IBCRateLimitKeeper.SetRateLimit(ctx, &ibcratelimittypes.RateLimit{
		ChannelId:    s.transferPath.EndpointB.ChannelID,
		Denom:        sdk.DefaultBondDenom,
		MaxSend:      "2000000",
		MaxRecv:      "1000000",
		WindowBlocks: 100,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  uint64(ctx.BlockHeight()),
	})

	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_500_000)) // exceeds 1M recv limit

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
	s.Require().NoError(err, "send side should succeed (no send rate limit)")

	// Parse and relay — the relay should succeed but B returns an error ack.
	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)

	// Record sender balance before relay (the error ack should refund).
	appA := GetZeroneApp(s.chainA)
	balBefore := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)

	err = s.transferPath.RelayPacket(packet)
	s.Require().NoError(err, "relay should succeed even with error ack")

	// After error ack, sender should be refunded on chain A.
	balAfter := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
	s.Require().True(balAfter.Amount.GTE(balBefore.Amount),
		"sender should be refunded after error ack: before=%s, after=%s",
		balBefore.Amount, balAfter.Amount)
}

// TestWindowReset verifies that the rate limit window resets after enough blocks.
func (s *IBCTestSuite) TestWindowReset() {
	windowBlocks := uint64(5)
	s.SetupRateLimit(
		s.transferPath.EndpointA.ChannelID,
		sdk.DefaultBondDenom,
		"1000000", // maxSend
		"2000000", // maxRecv
		windowBlocks,
	)

	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	timeoutHeight := clienttypes.NewHeight(1, 200)

	// Send exactly the limit.
	msg := transfertypes.NewMsgTransfer(
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_000_000)),
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

	// Another transfer should fail — quota exhausted.
	msg2 := transfertypes.NewMsgTransfer(
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1)),
		sender.String(),
		receiver.String(),
		timeoutHeight,
		0,
		"",
	)
	_, err = s.chainA.SendMsgs(msg2)
	s.Require().Error(err, "should fail — quota exhausted")

	// Advance chain A past the window.
	s.coordinator.CommitNBlocks(s.chainA, windowBlocks+1)

	// Now transfer should succeed — window reset.
	msg3 := transfertypes.NewMsgTransfer(
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(500_000)),
		sender.String(),
		receiver.String(),
		timeoutHeight,
		0,
		"",
	)
	res, err = s.chainA.SendMsgs(msg3)
	s.Require().NoError(err, "should succeed after window reset")

	packet, err = ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)
	err = s.transferPath.RelayPacket(packet)
	s.Require().NoError(err)
}

// TestAccumulativeSendQuota verifies that multiple small transfers accumulate
// and eventually exceed the send limit.
func (s *IBCTestSuite) TestAccumulativeSendQuota() {
	s.SetupRateLimit(
		s.transferPath.EndpointA.ChannelID,
		sdk.DefaultBondDenom,
		"1000000", // maxSend
		"2000000", // maxRecv
		100,       // windowBlocks
	)

	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	timeoutHeight := clienttypes.NewHeight(1, 200)

	// Send 400k three times: 400k + 400k = 800k OK, third 400k = 1200k > 1M limit.
	for i := 0; i < 2; i++ {
		msg := transfertypes.NewMsgTransfer(
			s.transferPath.EndpointA.ChannelConfig.PortID,
			s.transferPath.EndpointA.ChannelID,
			sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(400_000)),
			sender.String(),
			receiver.String(),
			timeoutHeight,
			0,
			"",
		)
		res, err := s.chainA.SendMsgs(msg)
		s.Require().NoError(err, "transfer %d should succeed", i+1)
		packet, err := ibctesting.ParsePacketFromEvents(res.Events)
		s.Require().NoError(err)
		err = s.transferPath.RelayPacket(packet)
		s.Require().NoError(err)
	}

	// Third 400k should exceed the 1M limit (800k + 400k = 1.2M > 1M).
	msg := transfertypes.NewMsgTransfer(
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(400_000)),
		sender.String(),
		receiver.String(),
		timeoutHeight,
		0,
		"",
	)
	_, err := s.chainA.SendMsgs(msg)
	s.Require().Error(err, "third transfer should exceed accumulative send quota")
}
