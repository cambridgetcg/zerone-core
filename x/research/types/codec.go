package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitResearch{}, "zerone_research/SubmitResearch", nil)
	cdc.RegisterConcrete(&MsgChallengeResearch{}, "zerone_research/ChallengeResearch", nil)
	cdc.RegisterConcrete(&MsgReviewResearch{}, "zerone_research/ReviewResearch", nil)
	cdc.RegisterConcrete(&MsgResolveResearch{}, "zerone_research/ResolveResearch", nil)
	cdc.RegisterConcrete(&MsgCreateBounty{}, "zerone_research/CreateBounty", nil)
	cdc.RegisterConcrete(&MsgClaimBounty{}, "zerone_research/ClaimBounty", nil)
	cdc.RegisterConcrete(&MsgFulfillBounty{}, "zerone_research/FulfillBounty", nil)
	cdc.RegisterConcrete(&MsgFundResearch{}, "zerone_research/FundResearch", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_research/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitResearch{},
		&MsgChallengeResearch{},
		&MsgReviewResearch{},
		&MsgResolveResearch{},
		&MsgCreateBounty{},
		&MsgClaimBounty{},
		&MsgFulfillBounty{},
		&MsgFundResearch{},
		&MsgUpdateParams{},
	)
}
