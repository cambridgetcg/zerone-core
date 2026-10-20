package ibcratelimit

import (
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/zerone-chain/zerone/x/ibcratelimit/keeper"
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

func (im IBCMiddleware) SendPacket(ctx sdk.Context, sourcePort string, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	refundIdentity, err := im.keeper.RefundIdentityEnabled(ctx)
	if err != nil {
		return 0, err
	}
	if !refundIdentity {
		return im.sendPacketLegacy(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}
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

	charge, err := im.keeper.ChargeSendQuota(ctx, sourceChannel, denom, amount)
	if err != nil {
		return 0, err
	}

	seq, err := im.channel.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		// Keeper-contract rollback only: a failed production message discards
		// its SDK cache. Disabled/unconfigured sends have nothing to reverse.
		im.keeper.ReverseSendCharge(ctx, charge)
		return 0, err
	}

	// Only an actual charge creates a reversal record.
	im.keeper.RecordPacketCharge(ctx, seq, charge)

	return seq, nil
}

func (im IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return im.channel.WriteAcknowledgement(ctx, packet, ack)
}

func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.channel.GetAppVersion(ctx, portID, channelID)
}

// ---- IBCModule (inbound interception) ----

func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	refundIdentity, err := im.keeper.RefundIdentityEnabled(ctx)
	if err != nil {
		// IBCModule has no Go error return here. A marker fault must abort,
		// not become a remotely actionable error acknowledgement.
		panic(err)
	}
	if !refundIdentity {
		return im.onRecvPacketLegacy(ctx, channelVersion, packet, relayer)
	}
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

	if err := im.keeper.CheckAndUpdateRecvQuotaForActivation(ctx, refundIdentity, packet.GetDestChannel(), denom, amount); err != nil {
		// Return error acknowledgement
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	refundIdentity, err := im.keeper.RefundIdentityEnabled(ctx)
	if err != nil {
		return err
	}
	if !refundIdentity {
		return im.onAcknowledgementPacketLegacy(ctx, channelVersion, packet, acknowledgement, relayer)
	}
	// Parse ack to check if it's an error.
	// Ack bytes use protobuf JSON encoding (SubModuleCdc), not standard encoding/json.
	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err == nil {
		if _, ok := ack.Response.(*channeltypes.Acknowledgement_Error); ok {
			// Error ack — reverse the send quota
			im.keeper.ReversePacketFlow(ctx, packet.GetSourceChannel(), packet.GetSequence())
		}
	}

	// Always clean up the flow record
	im.keeper.DeletePacketFlowForActivation(ctx, refundIdentity, packet.GetSourceChannel(), packet.GetSequence())

	return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	refundIdentity, err := im.keeper.RefundIdentityEnabled(ctx)
	if err != nil {
		return err
	}
	if !refundIdentity {
		return im.onTimeoutPacketLegacy(ctx, channelVersion, packet, relayer)
	}
	// Timeout — reverse the send quota
	im.keeper.ReversePacketFlow(ctx, packet.GetSourceChannel(), packet.GetSequence())
	im.keeper.DeletePacketFlowForActivation(ctx, refundIdentity, packet.GetSourceChannel(), packet.GetSequence())

	return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

// ---- Channel Handshake Callbacks (pure pass-through) ----

func (im IBCMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, counterparty channeltypes.Counterparty, version string) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

func (im IBCMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
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
