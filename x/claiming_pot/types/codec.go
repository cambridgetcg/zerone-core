package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreatePot{}, "zerone_claiming_pot/CreatePot", nil)
	cdc.RegisterConcrete(&MsgClaim{}, "zerone_claiming_pot/Claim", nil)
	cdc.RegisterConcrete(&MsgUpdatePotParams{}, "zerone_claiming_pot/UpdatePotParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreatePot{},
		&MsgClaim{},
		&MsgUpdatePotParams{},
	)
}
