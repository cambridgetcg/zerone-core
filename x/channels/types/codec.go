package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the channels module types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgOpenChannel{}, "zerone_channels/OpenChannel", nil)
	cdc.RegisterConcrete(&MsgDepositChannel{}, "zerone_channels/DepositChannel", nil)
	cdc.RegisterConcrete(&MsgUpdateState{}, "zerone_channels/UpdateState", nil)
	cdc.RegisterConcrete(&MsgCloseChannel{}, "zerone_channels/CloseChannel", nil)
	cdc.RegisterConcrete(&MsgDisputeChannel{}, "zerone_channels/DisputeChannel", nil)
	cdc.RegisterConcrete(&MsgClaimExpired{}, "zerone_channels/ClaimExpired", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_channels/UpdateParams", nil)
}

// RegisterInterfaces registers the channels module interface implementations.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgOpenChannel{},
		&MsgDepositChannel{},
		&MsgUpdateState{},
		&MsgCloseChannel{},
		&MsgDisputeChannel{},
		&MsgClaimExpired{},
		&MsgUpdateParams{},
	)
}
