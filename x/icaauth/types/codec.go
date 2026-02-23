package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterAccount{}, "zerone_icaauth/RegisterAccount", nil)
	cdc.RegisterConcrete(&MsgSubmitTx{}, "zerone_icaauth/SubmitTx", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_icaauth/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterAccount{},
		&MsgSubmitTx{},
		&MsgUpdateParams{},
	)
}
