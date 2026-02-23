package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDeployContract{}, "bvm/DeployContract", nil)
	cdc.RegisterConcrete(&MsgCallContract{}, "bvm/CallContract", nil)
	cdc.RegisterConcrete(&MsgScheduleExecution{}, "bvm/ScheduleExecution", nil)
	cdc.RegisterConcrete(&MsgScheduleContract{}, "bvm/ScheduleContract", nil)
	cdc.RegisterConcrete(&MsgCancelSchedule{}, "bvm/CancelSchedule", nil)
	cdc.RegisterConcrete(&MsgUpdateContractState{}, "bvm/UpdateContractState", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "bvm/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDeployContract{},
		&MsgCallContract{},
		&MsgScheduleExecution{},
		&MsgScheduleContract{},
		&MsgCancelSchedule{},
		&MsgUpdateContractState{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
