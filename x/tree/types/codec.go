package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the tree module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateProject{}, "tree/CreateProject", nil)
	cdc.RegisterConcrete(&MsgProposeProject{}, "tree/ProposeProject", nil)
	cdc.RegisterConcrete(&MsgStartDevelopment{}, "tree/StartDevelopment", nil)
	cdc.RegisterConcrete(&MsgCompleteProject{}, "tree/CompleteProject", nil)
	cdc.RegisterConcrete(&MsgPauseProject{}, "tree/PauseProject", nil)
	cdc.RegisterConcrete(&MsgResumeProject{}, "tree/ResumeProject", nil)
	cdc.RegisterConcrete(&MsgAbandonProject{}, "tree/AbandonProject", nil)
	cdc.RegisterConcrete(&MsgSpawnChildProject{}, "tree/SpawnChildProject", nil)
	cdc.RegisterConcrete(&MsgAddTask{}, "tree/AddTask", nil)
	cdc.RegisterConcrete(&MsgAssignTask{}, "tree/AssignTask", nil)
	cdc.RegisterConcrete(&MsgStartWork{}, "tree/StartWork", nil)
	cdc.RegisterConcrete(&MsgSubmitDeliverable{}, "tree/SubmitDeliverable", nil)
	cdc.RegisterConcrete(&MsgApproveDeliverable{}, "tree/ApproveDeliverable", nil)
	cdc.RegisterConcrete(&MsgRejectDeliverable{}, "tree/RejectDeliverable", nil)
	cdc.RegisterConcrete(&MsgReopenTask{}, "tree/ReopenTask", nil)
	cdc.RegisterConcrete(&MsgApplyToProject{}, "tree/ApplyToProject", nil)
	cdc.RegisterConcrete(&MsgReviewApplication{}, "tree/ReviewApplication", nil)
	cdc.RegisterConcrete(&MsgSetAvailability{}, "tree/SetAvailability", nil)
	cdc.RegisterConcrete(&MsgAddContributor{}, "tree/AddContributor", nil)
	cdc.RegisterConcrete(&MsgDeployService{}, "tree/DeployService", nil)
	cdc.RegisterConcrete(&MsgCallService{}, "tree/CallService", nil)
	cdc.RegisterConcrete(&MsgSubscribeService{}, "tree/SubscribeService", nil)
	cdc.RegisterConcrete(&MsgPauseService{}, "tree/PauseService", nil)
	cdc.RegisterConcrete(&MsgResumeService{}, "tree/ResumeService", nil)
	cdc.RegisterConcrete(&MsgRetireService{}, "tree/RetireService", nil)
	cdc.RegisterConcrete(&MsgBeginSeeding{}, "tree/BeginSeeding", nil)
	cdc.RegisterConcrete(&MsgDetectOpportunity{}, "tree/DetectOpportunity", nil)
	cdc.RegisterConcrete(&MsgClaimOpportunity{}, "tree/ClaimOpportunity", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "tree/UpdateParams", nil)
}

// RegisterInterfaces registers the tree module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateProject{},
		&MsgProposeProject{},
		&MsgStartDevelopment{},
		&MsgCompleteProject{},
		&MsgPauseProject{},
		&MsgResumeProject{},
		&MsgAbandonProject{},
		&MsgSpawnChildProject{},
		&MsgAddTask{},
		&MsgAssignTask{},
		&MsgStartWork{},
		&MsgSubmitDeliverable{},
		&MsgApproveDeliverable{},
		&MsgRejectDeliverable{},
		&MsgReopenTask{},
		&MsgApplyToProject{},
		&MsgReviewApplication{},
		&MsgSetAvailability{},
		&MsgAddContributor{},
		&MsgDeployService{},
		&MsgCallService{},
		&MsgSubscribeService{},
		&MsgPauseService{},
		&MsgResumeService{},
		&MsgRetireService{},
		&MsgBeginSeeding{},
		&MsgDetectOpportunity{},
		&MsgClaimOpportunity{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
