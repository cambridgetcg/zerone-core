package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the schedule module types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateSchedule{}, "zerone_schedule/CreateSchedule", nil)
	cdc.RegisterConcrete(&MsgPauseSchedule{}, "zerone_schedule/PauseSchedule", nil)
	cdc.RegisterConcrete(&MsgResumeSchedule{}, "zerone_schedule/ResumeSchedule", nil)
	cdc.RegisterConcrete(&MsgCancelSchedule{}, "zerone_schedule/CancelSchedule", nil)
	cdc.RegisterConcrete(&MsgFundSchedule{}, "zerone_schedule/FundSchedule", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_schedule/UpdateParams", nil)
}

// RegisterInterfaces registers the schedule module interface implementations.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateSchedule{},
		&MsgPauseSchedule{},
		&MsgResumeSchedule{},
		&MsgCancelSchedule{},
		&MsgFundSchedule{},
		&MsgUpdateParams{},
	)
}
