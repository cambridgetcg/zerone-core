package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitChallenge{}, "zerone_capture_challenge/SubmitChallenge", nil)
	cdc.RegisterConcrete(&MsgAddEvidence{}, "zerone_capture_challenge/AddEvidence", nil)
	cdc.RegisterConcrete(&MsgResolveChallenge{}, "zerone_capture_challenge/ResolveChallenge", nil)
	cdc.RegisterConcrete(&MsgFundBountyPool{}, "zerone_capture_challenge/FundBountyPool", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_capture_challenge/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitChallenge{},
		&MsgAddEvidence{},
		&MsgResolveChallenge{},
		&MsgFundBountyPool{},
		&MsgUpdateParams{},
	)
}
