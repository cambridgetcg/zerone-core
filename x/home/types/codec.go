package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the home module types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateHome{}, "zerone_home/CreateHome", nil)
	cdc.RegisterConcrete(&MsgUpdateHome{}, "zerone_home/UpdateHome", nil)
	cdc.RegisterConcrete(&MsgUpdateMemoryCID{}, "zerone_home/UpdateMemoryCID", nil)
	cdc.RegisterConcrete(&MsgStartSession{}, "zerone_home/StartSession", nil)
	cdc.RegisterConcrete(&MsgEndSession{}, "zerone_home/EndSession", nil)
	cdc.RegisterConcrete(&MsgRegisterKey{}, "zerone_home/RegisterKey", nil)
	cdc.RegisterConcrete(&MsgRevokeKey{}, "zerone_home/RevokeKey", nil)
	cdc.RegisterConcrete(&MsgConfigureGuardian{}, "zerone_home/ConfigureGuardian", nil)
	cdc.RegisterConcrete(&MsgAcknowledgeAlert{}, "zerone_home/AcknowledgeAlert", nil)
	cdc.RegisterConcrete(&MsgSetSpendingLimit{}, "zerone_home/SetSpendingLimit", nil)
}

// RegisterInterfaces registers the home module interface implementations.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateHome{},
		&MsgUpdateHome{},
		&MsgUpdateMemoryCID{},
		&MsgStartSession{},
		&MsgEndSession{},
		&MsgRegisterKey{},
		&MsgRevokeKey{},
		&MsgConfigureGuardian{},
		&MsgAcknowledgeAlert{},
		&MsgSetSpendingLimit{},
	)
}
