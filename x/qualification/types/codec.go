package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgQualifyByStake{}, "zerone_qualification/QualifyByStake", nil)
	cdc.RegisterConcrete(&MsgQualifyByTrackRecord{}, "zerone_qualification/QualifyByTrackRecord", nil)
	cdc.RegisterConcrete(&MsgQualifyByCrossReference{}, "zerone_qualification/QualifyByCrossReference", nil)
	cdc.RegisterConcrete(&MsgQualifyByInheritance{}, "zerone_qualification/QualifyByInheritance", nil)
	cdc.RegisterConcrete(&MsgEndorseQualification{}, "zerone_qualification/EndorseQualification", nil)
	cdc.RegisterConcrete(&MsgRenewQualification{}, "zerone_qualification/RenewQualification", nil)
	cdc.RegisterConcrete(&MsgWithdrawQualification{}, "zerone_qualification/WithdrawQualification", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_qualification/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgQualifyByStake{},
		&MsgQualifyByTrackRecord{},
		&MsgQualifyByCrossReference{},
		&MsgQualifyByInheritance{},
		&MsgEndorseQualification{},
		&MsgRenewQualification{},
		&MsgWithdrawQualification{},
		&MsgUpdateParams{},
	)
}
