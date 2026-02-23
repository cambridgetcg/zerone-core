package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRecordVerification{}, "zerone_capture_defense/RecordVerification", nil)
	cdc.RegisterConcrete(&MsgAnalyzeDomain{}, "zerone_capture_defense/AnalyzeDomain", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_capture_defense/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRecordVerification{},
		&MsgAnalyzeDomain{},
		&MsgUpdateParams{},
	)
}
