package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the disputes module types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgInitiateDispute{}, "zerone_disputes/InitiateDispute", nil)
	cdc.RegisterConcrete(&MsgCommitEvidence{}, "zerone_disputes/CommitEvidence", nil)
	cdc.RegisterConcrete(&MsgRevealEvidence{}, "zerone_disputes/RevealEvidence", nil)
	cdc.RegisterConcrete(&MsgArbiterVote{}, "zerone_disputes/ArbiterVote", nil)
	cdc.RegisterConcrete(&MsgEscalateDispute{}, "zerone_disputes/EscalateDispute", nil)
	cdc.RegisterConcrete(&MsgSettleDispute{}, "zerone_disputes/SettleDispute", nil)
}

// RegisterInterfaces registers the disputes module interface implementations.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgInitiateDispute{},
		&MsgCommitEvidence{},
		&MsgRevealEvidence{},
		&MsgArbiterVote{},
		&MsgEscalateDispute{},
		&MsgSettleDispute{},
	)
}
