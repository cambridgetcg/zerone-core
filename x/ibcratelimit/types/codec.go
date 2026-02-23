package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddRateLimit{}, "zerone_ibcratelimit/AddRateLimit", nil)
	cdc.RegisterConcrete(&MsgRemoveRateLimit{}, "zerone_ibcratelimit/RemoveRateLimit", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_ibcratelimit/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddRateLimit{},
		&MsgRemoveRateLimit{},
		&MsgUpdateParams{},
	)
}
