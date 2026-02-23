package ibc_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	ibcratelimittypes "github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// TestTimeoutRefund sends a transfer with a very short timeout height, does
// not relay it, advances chain B past the timeout, and verifies the sender
// gets tokens refunded.
func (s *IBCTestSuite) TestTimeoutRefund() {
	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_000_000))

	// Record balance before.
	appA := GetZeroneApp(s.chainA)
	balBefore := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)

	// Send with a very short timeout height (current B height + 1).
	timeoutHeight := clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))
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

	// Balance decreased after sending (tokens are escrowed).
	balAfterSend := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
	s.Require().True(balAfterSend.Amount.LT(balBefore.Amount), "balance should decrease after send")

	// Advance chain B past the timeout height.
	s.coordinator.CommitNBlocks(s.chainB, 5)

	// Update client on A, then timeout the packet.
	err = s.transferPath.EndpointA.UpdateClient()
	s.Require().NoError(err)

	err = s.transferPath.EndpointA.TimeoutPacket(packet)
	s.Require().NoError(err)

	// After timeout, sender should be refunded.
	balAfterTimeout := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
	s.Require().Equal(balBefore.Amount, balAfterTimeout.Amount,
		"sender should be fully refunded after timeout: before=%s, after=%s",
		balBefore.Amount, balAfterTimeout.Amount)
}

// TestErrorAckRefund sends a transfer that triggers a recv rate limit error
// on chain B, resulting in an error ack. This tests that:
// 1. The sender is refunded on chain A.
// 2. The send quota is reversed on chain A's rate limit (if configured).
func (s *IBCTestSuite) TestErrorAckRefund() {
	// Set up a send rate limit on chain A to verify quota reversal.
	s.SetupRateLimit(
		s.transferPath.EndpointA.ChannelID,
		sdk.DefaultBondDenom,
		"2000000", // maxSend — high enough for the transfer
		"2000000", // maxRecv
		100,
	)

	// Set up a recv rate limit on chain B that will reject the transfer.
	appB := GetZeroneApp(s.chainB)
	ctxB := s.chainB.GetContext()
	appB.IBCRateLimitKeeper.SetParams(ctxB, &ibcratelimittypes.Params{Enabled: true})
	appB.IBCRateLimitKeeper.SetRateLimit(ctxB, &ibcratelimittypes.RateLimit{
		ChannelId:    s.transferPath.EndpointB.ChannelID,
		Denom:        sdk.DefaultBondDenom,
		MaxSend:      "2000000",
		MaxRecv:      "100000", // very small — will reject 1M transfer
		WindowBlocks: 100,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  uint64(ctxB.BlockHeight()),
	})

	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_000_000))

	appA := GetZeroneApp(s.chainA)
	balBefore := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)

	timeoutHeight := clienttypes.NewHeight(1, 200)
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
	s.Require().NoError(err, "send should succeed (within A's send limit)")

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)

	// Relay the packet — B will return an error ack, which gets relayed back to A.
	err = s.transferPath.RelayPacket(packet)
	s.Require().NoError(err, "relay should succeed even with error ack")

	// After the error ack is processed, sender should be refunded.
	balAfter := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
	s.Require().Equal(balBefore.Amount, balAfter.Amount,
		"sender should be fully refunded after error ack: before=%s, after=%s",
		balBefore.Amount, balAfter.Amount)

	// Verify the send quota was reversed on chain A.
	rl, found := appA.IBCRateLimitKeeper.GetRateLimit(
		s.chainA.GetContext(),
		s.transferPath.EndpointA.ChannelID,
		sdk.DefaultBondDenom,
	)
	s.Require().True(found, "rate limit should still exist on chain A")
	s.Require().Equal("0", rl.CurrentSend,
		"send quota should be reversed after error ack: got %s", rl.CurrentSend)
}
