package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── Route B: training infrastructure msg handlers ─────────────────────────

// RegisterTrainingPipeline — operator creates a pipeline declaration. The
// pipeline pins a corpus snapshot, tokenizer version, and recipe hash so
// any consumer can reproduce the dataset the pipeline trained against.
func (m *msgServer) RegisterTrainingPipeline(ctx context.Context, msg *types.MsgRegisterTrainingPipeline) (*types.MsgRegisterTrainingPipelineResponse, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}
	if msg.Id == "" {
		return nil, fmt.Errorf("pipeline id is required")
	}
	if msg.Operator == "" {
		return nil, fmt.Errorf("operator address is required")
	}
	if _, exists := m.keeper.GetTrainingPipeline(ctx, msg.Id); exists {
		return nil, fmt.Errorf("training pipeline %s already exists", msg.Id)
	}
	// Tokenizer version pinning: if specified, must exist in the registry.
	if msg.TokenizerVersion != 0 {
		if _, ok := m.keeper.GetTokenizerSpecAtVersion(ctx, msg.TokenizerVersion); !ok {
			return nil, fmt.Errorf("tokenizer version %d not found in registry", msg.TokenizerVersion)
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	pipeline := &types.TrainingPipeline{
		Id:                    msg.Id,
		OperatorAddress:       msg.Operator,
		CorpusSnapshotHeight:  msg.CorpusSnapshotHeight,
		TokenizerVersion:      msg.TokenizerVersion,
		MethodologySetVersion: msg.MethodologySetVersion,
		RecipeHash:            msg.RecipeHash,
		Description:           msg.Description,
		Status:                "declared",
		DeclaredAtBlock:       height,
		CorpusFilter:          msg.CorpusFilter,
	}
	if err := m.keeper.SetTrainingPipeline(ctx, pipeline); err != nil {
		return nil, err
	}
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.training_pipeline_registered",
		sdk.NewAttribute("pipeline_id", pipeline.Id),
		sdk.NewAttribute("operator", pipeline.OperatorAddress),
		sdk.NewAttribute("corpus_snapshot_height", fmt.Sprintf("%d", pipeline.CorpusSnapshotHeight)),
		sdk.NewAttribute("tokenizer_version", fmt.Sprintf("%d", pipeline.TokenizerVersion)),
		sdk.NewAttribute("recipe_hash", pipeline.RecipeHash),
	))
	return &types.MsgRegisterTrainingPipelineResponse{}, nil
}

// UpdateTrainingPipeline — operator updates status / completion.
func (m *msgServer) UpdateTrainingPipeline(ctx context.Context, msg *types.MsgUpdateTrainingPipeline) (*types.MsgUpdateTrainingPipelineResponse, error) {
	if msg == nil || msg.Id == "" {
		return nil, fmt.Errorf("pipeline id is required")
	}
	pipeline, ok := m.keeper.GetTrainingPipeline(ctx, msg.Id)
	if !ok {
		return nil, fmt.Errorf("training pipeline %s not found", msg.Id)
	}
	if pipeline.OperatorAddress != msg.Operator {
		return nil, fmt.Errorf("only the declaring operator may update this pipeline")
	}
	if msg.NewStatus != "" {
		pipeline.Status = msg.NewStatus
	}
	if msg.CompletedAtBlock != 0 {
		pipeline.CompletedAtBlock = msg.CompletedAtBlock
	}
	if err := m.keeper.SetTrainingPipeline(ctx, pipeline); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.training_pipeline_updated",
		sdk.NewAttribute("pipeline_id", pipeline.Id),
		sdk.NewAttribute("operator", pipeline.OperatorAddress),
		sdk.NewAttribute("new_status", pipeline.Status),
	))
	return &types.MsgUpdateTrainingPipelineResponse{}, nil
}

// RegisterModelCard — owner registers a ModelCard for a trained model.
// The referenced pipeline must exist. The deployment_address is the agent
// account the model runs as; calibration accrues to that address under Phase 5.
func (m *msgServer) RegisterModelCard(ctx context.Context, msg *types.MsgRegisterModelCard) (*types.MsgRegisterModelCardResponse, error) {
	if msg == nil || msg.Id == "" {
		return nil, fmt.Errorf("model id is required")
	}
	if msg.Owner == "" {
		return nil, fmt.Errorf("owner address is required")
	}
	if msg.PipelineId == "" {
		return nil, fmt.Errorf("pipeline_id is required")
	}
	if _, ok := m.keeper.GetTrainingPipeline(ctx, msg.PipelineId); !ok {
		return nil, fmt.Errorf("pipeline %s not found; register it first", msg.PipelineId)
	}
	if _, exists := m.keeper.GetModelCard(ctx, msg.Id); exists {
		return nil, fmt.Errorf("model card %s already exists", msg.Id)
	}
	switch msg.Route {
	case "openweight_fine_tune", "from_scratch", "distilled":
		// ok
	case "":
		return nil, fmt.Errorf("route is required")
	default:
		return nil, fmt.Errorf("invalid route %q", msg.Route)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	card := &types.ModelCard{
		Id:                       msg.Id,
		Name:                     msg.Name,
		PipelineId:               msg.PipelineId,
		DeploymentAddress:        msg.DeploymentAddress,
		CreatedAtBlock:           height,
		ParameterCount:           msg.ParameterCount,
		Route:                    msg.Route,
		BaseModel:                msg.BaseModel,
		OwnerAddress:             msg.Owner,
		EvalAcceptanceRateBps:    msg.EvalAcceptanceRateBps,
		EvalCorroborationRateBps: msg.EvalCorroborationRateBps,
		EvalSampleSize:           msg.EvalSampleSize,
		SpecialisedMethodId:      msg.SpecialisedMethodId,
		Active:                   true,
	}
	if err := m.keeper.SetModelCard(ctx, card); err != nil {
		return nil, err
	}
	return &types.MsgRegisterModelCardResponse{}, nil
}

// UpdateModelCard — owner updates evaluation stats or metadata.
func (m *msgServer) UpdateModelCard(ctx context.Context, msg *types.MsgUpdateModelCard) (*types.MsgUpdateModelCardResponse, error) {
	if msg == nil || msg.Id == "" {
		return nil, fmt.Errorf("model id is required")
	}
	card, ok := m.keeper.GetModelCard(ctx, msg.Id)
	if !ok {
		return nil, fmt.Errorf("model card %s not found", msg.Id)
	}
	if card.OwnerAddress != msg.Owner {
		return nil, fmt.Errorf("only the model owner may update this card")
	}
	if !card.Active {
		return nil, fmt.Errorf("cannot update retired model card")
	}
	if msg.EvalAcceptanceRateBps != 0 {
		card.EvalAcceptanceRateBps = msg.EvalAcceptanceRateBps
	}
	if msg.EvalCorroborationRateBps != 0 {
		card.EvalCorroborationRateBps = msg.EvalCorroborationRateBps
	}
	if msg.EvalSampleSize != 0 {
		card.EvalSampleSize = msg.EvalSampleSize
	}
	if msg.Name != "" {
		card.Name = msg.Name
	}
	if err := m.keeper.SetModelCard(ctx, card); err != nil {
		return nil, err
	}
	return &types.MsgUpdateModelCardResponse{}, nil
}

// RetireModelCard — owner flips active=false and records retirement.
func (m *msgServer) RetireModelCard(ctx context.Context, msg *types.MsgRetireModelCard) (*types.MsgRetireModelCardResponse, error) {
	if msg == nil || msg.Id == "" {
		return nil, fmt.Errorf("model id is required")
	}
	card, ok := m.keeper.GetModelCard(ctx, msg.Id)
	if !ok {
		return nil, fmt.Errorf("model card %s not found", msg.Id)
	}
	if card.OwnerAddress != msg.Owner {
		return nil, fmt.Errorf("only the model owner may retire this card")
	}
	if !card.Active {
		return nil, fmt.Errorf("model card already retired")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	card.Active = false
	card.RetiredAtBlock = uint64(sdkCtx.BlockHeight())
	card.RetiredReason = msg.Reason
	// SetModelCard emits "updated" on re-write; additionally fire the
	// explicit retired event so observers have a dedicated signal.
	if err := m.keeper.SetModelCard(ctx, card); err != nil {
		return nil, err
	}
	m.keeper.EmitModelCardEvent(ctx, card, "retired")
	return &types.MsgRetireModelCardResponse{}, nil
}
