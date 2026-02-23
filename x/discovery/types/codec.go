package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the discovery module types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterProfile{}, "zerone_discovery/RegisterProfile", nil)
	cdc.RegisterConcrete(&MsgUpdateProfile{}, "zerone_discovery/UpdateProfile", nil)
	cdc.RegisterConcrete(&MsgHeartbeat{}, "zerone_discovery/Heartbeat", nil)
	cdc.RegisterConcrete(&MsgDeregisterProfile{}, "zerone_discovery/DeregisterProfile", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_discovery/UpdateParams", nil)
}

// RegisterInterfaces registers the discovery module interface implementations.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterProfile{},
		&MsgUpdateProfile{},
		&MsgHeartbeat{},
		&MsgDeregisterProfile{},
		&MsgUpdateParams{},
	)
}
