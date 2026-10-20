package ibc_test

import (
	"encoding/json"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	ibcratelimittypes "github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// This is an in-memory IBC coordinator, not a live node or upgrade-handler
// rehearsal. It exercises actual signed messages, SDK bank escrow and transfer
// callbacks across an in-place marker transition with a genuinely charged
// legacy receipt, rather than only a synthetic middleware callback.
func (s *IBCTestSuite) TestLegacyRefundAfterIdentityActivation() {
	previousInit := ibctesting.DefaultTestingAppInit
	ibctesting.DefaultTestingAppInit = func() (ibctesting.TestingApp, map[string]json.RawMessage) {
		app, genesis := previousInit()
		legacy := ibcratelimittypes.DefaultGenesis()
		legacy.RefundIdentityEnabled = false
		raw, err := json.Marshal(legacy)
		s.Require().NoError(err)
		genesis[ibcratelimittypes.ModuleName] = raw
		return app, genesis
	}
	defer func() { ibctesting.DefaultTestingAppInit = previousInit }()

	for _, mode := range []string{"timeout", "error_ack"} {
		s.Run(mode, func() {
			// Rebuild only this test's ephemeral chains with explicit marker-off
			// DefaultGenesis; never clear an already-enabled production marker.
			s.SetupTest()
			appA, appB := GetZeroneApp(s.chainA), GetZeroneApp(s.chainB)
			ctxA, ctxB := s.chainA.GetContext(), s.chainB.GetContext()
			enabled, err := appA.IBCRateLimitKeeper.RefundIdentityEnabled(ctxA)
			s.Require().NoError(err)
			s.Require().False(enabled)
			appA.IBCRateLimitKeeper.SetRateLimitForActivation(ctxA, enabled, &ibcratelimittypes.RateLimit{
				ChannelId: s.transferPath.EndpointA.ChannelID, Denom: sdk.DefaultBondDenom,
				MaxSend: "2000000", MaxRecv: "2000000", WindowBlocks: 1000,
				CurrentSend: "0", CurrentRecv: "0", WindowStart: uint64(ctxA.BlockHeight()),
			})
			if mode == "error_ack" {
				appB.IBCRateLimitKeeper.SetRateLimitForActivation(ctxB, false, &ibcratelimittypes.RateLimit{
					ChannelId: s.transferPath.EndpointB.ChannelID, Denom: sdk.DefaultBondDenom,
					MaxSend: "2000000", MaxRecv: "1", WindowBlocks: 1000,
					CurrentSend: "0", CurrentRecv: "0", WindowStart: uint64(ctxB.BlockHeight()),
				})
			}
			sender := s.chainA.SenderAccount.GetAddress()
			receiver := s.chainB.SenderAccount.GetAddress()
			before := appA.BankKeeper.GetBalance(ctxA, sender, sdk.DefaultBondDenom)
			height := uint64(ctxB.BlockHeight() + 100)
			if mode == "timeout" {
				height = uint64(ctxB.BlockHeight() + 1)
			}
			msg := transfertypes.NewMsgTransfer(
				s.transferPath.EndpointA.ChannelConfig.PortID, s.transferPath.EndpointA.ChannelID,
				sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_000_000)), sender.String(), receiver.String(),
				clienttypes.NewHeight(1, height), 0, "",
			)
			result, err := s.chainA.SendMsgs(msg)
			s.Require().NoError(err)
			packet, err := ibctesting.ParsePacketFromEvents(result.Events)
			s.Require().NoError(err)
			_, found := appA.IBCRateLimitKeeper.GetPacketFlow(s.chainA.GetContext(), packet.SourceChannel, packet.Sequence)
			s.Require().True(found)
			charged, found := appA.IBCRateLimitKeeper.GetRateLimit(s.chainA.GetContext(), packet.SourceChannel, sdk.DefaultBondDenom)
			s.Require().True(found)
			s.Require().Equal("1000000", charged.CurrentSend)

			s.Require().NoError(appA.IBCRateLimitKeeper.EnableRefundIdentity(s.chainA.GetContext()))
			if mode == "timeout" {
				s.coordinator.CommitNBlocks(s.chainB, 5)
				s.Require().NoError(s.transferPath.EndpointA.UpdateClient())
				s.Require().NoError(s.transferPath.EndpointA.TimeoutPacket(packet))
			} else {
				s.Require().NoError(s.transferPath.RelayPacket(packet))
			}
			after := appA.BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
			s.Require().Equal(before.Amount, after.Amount, "legacy quota policy must not withhold token refunds")
			retained, found := appA.IBCRateLimitKeeper.GetRateLimit(s.chainA.GetContext(), packet.SourceChannel, sdk.DefaultBondDenom)
			s.Require().True(found)
			s.Require().Equal("1000000", retained.CurrentSend, "retain legacy usage until window reset")
			_, found = appA.IBCRateLimitKeeper.GetPacketFlow(s.chainA.GetContext(), packet.SourceChannel, packet.Sequence)
			s.Require().False(found, "terminal callback still cleans the legacy receipt")
		})
	}
}
