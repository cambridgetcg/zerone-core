package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the zerone_auth module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterAccount{}, "zerone_auth/RegisterAccount", nil)
	cdc.RegisterConcrete(&MsgRotateKey{}, "zerone_auth/RotateKey", nil)
	cdc.RegisterConcrete(&MsgFreezeAccount{}, "zerone_auth/FreezeAccount", nil)
	cdc.RegisterConcrete(&MsgUnfreezeAccount{}, "zerone_auth/UnfreezeAccount", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_auth/UpdateParams", nil)
}

// RegisterInterfaces registers the zerone_auth module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterAccount{},
		&MsgRotateKey{},
		&MsgFreezeAccount{},
		&MsgUnfreezeAccount{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
