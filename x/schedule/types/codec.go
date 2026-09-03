package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	// These names intentionally do not reuse the incompatible Amino identities
	// from the retired x/schedule implementation. Although protobuf is the
	// canonical encoding, legacy clients must not be able to decode a v2
	// message as an old message (or the reverse).
	cdc.RegisterConcrete(&MsgCreateSchedule{}, "zerone_message_schedule_v2/CreateSchedule", nil)
	cdc.RegisterConcrete(&MsgUpdateSchedule{}, "zerone_message_schedule_v2/UpdateSchedule", nil)
	cdc.RegisterConcrete(&MsgCancelSchedule{}, "zerone_message_schedule_v2/CancelSchedule", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_message_schedule_v2/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateSchedule{},
		&MsgUpdateSchedule{},
		&MsgCancelSchedule{},
		&MsgUpdateParams{},
	)
}
