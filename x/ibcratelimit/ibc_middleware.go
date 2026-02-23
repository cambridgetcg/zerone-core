package ibcratelimit

import (
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/zerone-chain/zerone/x/ibcratelimit/keeper"
	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

var (
	_ porttypes.IBCModule   = IBCMiddleware{}
	_ porttypes.ICS4Wrapper = IBCMiddleware{}
)

// IBCMiddleware implements IBC middleware that rate-limits token transfers.
type IBCMiddleware struct {
	app     porttypes.IBCModule
	channel porttypes.ICS4Wrapper
	keeper  keeper.Keeper
}

func NewIBCMiddleware(app porttypes.IBCModule, channel porttypes.ICS4Wrapper, k keeper.Keeper) IBCMiddleware {
	return IBCMiddleware{
		app:     app,
		channel: channel,
		keeper:  k,
	}
}

// ---- ICS4Wrapper (outbound interception) ----

func (im IBCMiddleware) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, sourcePort string, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	// Decode FungibleTokenPacketData to extract denom and amount
	var packetData ibctransfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(data, &packetData); err != nil {
		// Not a transfer packet — pass through
		return im.channel.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(packetData.Amount, 10); !ok || amount.Sign() <= 0 {
		return im.channel.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	denom := packetData.Denom

	if err := im.keeper.CheckAndUpdateSendQuota(ctx, sourceChannel, denom, amount); err != nil {
		return 0, err
	}

	seq, err := im.channel.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
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

func (im IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return im.channel.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.channel.GetAppVersion(ctx, portID, channelID)
}

// ---- IBCModule (inbound interception) ----

func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	var packetData ibctransfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &packetData); err != nil {
		// Not a transfer packet — pass through
		return im.app.OnRecvPacket(ctx, packet, relayer)
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(packetData.Amount, 10); !ok || amount.Sign() <= 0 {
		return im.app.OnRecvPacket(ctx, packet, relayer)
	}

	denom := packetData.Denom

	if err := im.keeper.CheckAndUpdateRecvQuota(ctx, packet.GetDestChannel(), denom, amount); err != nil {
		// Return error acknowledgement
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return im.app.OnRecvPacket(ctx, packet, relayer)
}

func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	// Parse ack to check if it's an error.
	// Ack bytes use protobuf JSON encoding (SubModuleCdc), not standard encoding/json.
	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err == nil {
		if _, ok := ack.Response.(*channeltypes.Acknowledgement_Error); ok {
			// Error ack — reverse the send quota
			im.reversePacketFlow(ctx, packet.GetSourceChannel(), packet.GetSequence())
		}
	}

	// Always clean up the flow record
	im.keeper.DeletePacketFlow(ctx, packet.GetSourceChannel(), packet.GetSequence())

	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	// Timeout — reverse the send quota
	im.reversePacketFlow(ctx, packet.GetSourceChannel(), packet.GetSequence())
	im.keeper.DeletePacketFlow(ctx, packet.GetSourceChannel(), packet.GetSequence())

	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

// reversePacketFlow looks up the stored packet flow and reverses the send quota.
func (im IBCMiddleware) reversePacketFlow(ctx sdk.Context, channelID string, sequence uint64) {
	flow, found := im.keeper.GetPacketFlow(ctx, channelID, sequence)
	if !found {
		return
	}
	amount := new(big.Int)
	amount.SetString(flow.Amount, 10)
	im.keeper.ReverseSendQuota(ctx, flow.ChannelId, flow.Denom, amount)
}

// ---- Channel Handshake Callbacks (pure pass-through) ----

func (im IBCMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
}

func (im IBCMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}
