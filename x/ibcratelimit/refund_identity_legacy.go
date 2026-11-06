package ibcratelimit

import (
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// Frozen marker-off bodies from 77521a0daf14420836483ae9d8ea3fbcc1ae0d40.
// Only names/calls to frozen keeper bodies differ; do not share mutable fix
// helpers with this path. In particular uncharged flows and naive reversals
// remain here until the named carrier enables RefundIdentityEnabled.

func (im IBCMiddleware) sendPacketLegacy(ctx sdk.Context, sourcePort string, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	// Decode FungibleTokenPacketData to extract denom and amount
	var packetData ibctransfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(data, &packetData); err != nil {
		// Not a transfer packet — pass through
		return im.channel.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(packetData.Amount, 10); !ok || amount.Sign() <= 0 {
		return im.channel.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	denom := packetData.Denom

	if err := im.keeper.CheckAndUpdateSendQuota(ctx, sourceChannel, denom, amount); err != nil {
		return 0, err
	}

	seq, err := im.channel.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		// Reverse the quota update on send failure
		im.keeper.ReverseSendQuota(ctx, sourceChannel, denom, amount)
		return 0, err
	}

	// Record packet flow for reversal on timeout/error ack
	im.keeper.SetPacketFlow(ctx, &types.PacketFlow{
		ChannelId: sourceChannel,
		Sequence:  seq,
		Denom:     denom,
		Amount:    amount.String(),
	})

	return seq, nil
}

func (im IBCMiddleware) onRecvPacketLegacy(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	var packetData ibctransfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &packetData); err != nil {
		// Not a transfer packet — pass through
		return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(packetData.Amount, 10); !ok || amount.Sign() <= 0 {
		return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	denom := packetData.Denom

	if err := im.keeper.CheckAndUpdateRecvQuotaLegacy(ctx, packet.GetDestChannel(), denom, amount); err != nil {
		// Return error acknowledgement
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

func (im IBCMiddleware) onAcknowledgementPacketLegacy(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	// Parse ack to check if it's an error.
	// Ack bytes use protobuf JSON encoding (SubModuleCdc), not standard encoding/json.
	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err == nil {
		if _, ok := ack.Response.(*channeltypes.Acknowledgement_Error); ok {
			// Error ack — reverse the send quota
			im.reversePacketFlowLegacy(ctx, packet.GetSourceChannel(), packet.GetSequence())
		}
	}

	// Always clean up the flow record
	im.keeper.DeletePacketFlowLegacy(ctx, packet.GetSourceChannel(), packet.GetSequence())

	return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

func (im IBCMiddleware) onTimeoutPacketLegacy(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	// Timeout — reverse the send quota
	im.reversePacketFlowLegacy(ctx, packet.GetSourceChannel(), packet.GetSequence())
	im.keeper.DeletePacketFlowLegacy(ctx, packet.GetSourceChannel(), packet.GetSequence())

	return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

func (im IBCMiddleware) reversePacketFlowLegacy(ctx sdk.Context, channelID string, sequence uint64) {
	flow, found := im.keeper.GetPacketFlow(ctx, channelID, sequence)
	if !found {
		return
	}
	amount := new(big.Int)
	amount.SetString(flow.Amount, 10)
	im.keeper.ReverseSendQuota(ctx, flow.ChannelId, flow.Denom, amount)
}
