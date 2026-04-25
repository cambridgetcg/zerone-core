package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitInquiry{}, "zerone_inquiry/Submit", nil)
	cdc.RegisterConcrete(&MsgSubmitAnswer{}, "zerone_inquiry/Answer", nil)
	cdc.RegisterConcrete(&MsgResolveInquiry{}, "zerone_inquiry/Resolve", nil)
	cdc.RegisterConcrete(&MsgCancelInquiry{}, "zerone_inquiry/Cancel", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_inquiry/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitInquiry{},
		&MsgSubmitAnswer{},
		&MsgResolveInquiry{},
		&MsgCancelInquiry{},
		&MsgUpdateParams{},
	)
}
