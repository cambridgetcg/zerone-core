package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitEvidence{}, "zerone_evidence_mgmt/SubmitEvidence", nil)
	cdc.RegisterConcrete(&MsgTransferCustody{}, "zerone_evidence_mgmt/TransferCustody", nil)
	cdc.RegisterConcrete(&MsgVerifyEvidence{}, "zerone_evidence_mgmt/VerifyEvidence", nil)
	cdc.RegisterConcrete(&MsgChallengeEvidence{}, "zerone_evidence_mgmt/ChallengeEvidence", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_evidence_mgmt/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitEvidence{},
		&MsgTransferCustody{},
		&MsgVerifyEvidence{},
		&MsgChallengeEvidence{},
		&MsgUpdateParams{},
	)
}
