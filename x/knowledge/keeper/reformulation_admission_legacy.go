package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Admission bodies copied verbatim (only function names changed) from
// 6a1e1ecf46bdb5ef1ad94e2b1a88e763132af8d9, msg_server_training_v3.go.
// The entry points add prospective delimiter validation only when selected.
// Both paths then execute these unchanged bodies; do not add repair logic here.
func (m *msgServer) createAugmentationBountyLegacy(ctx context.Context, msg *types.MsgCreateAugmentationBounty) (*types.MsgCreateAugmentationBountyResponse, error) {
	neutral, err := m.keeper.ReviewNeutralityEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if neutral {
		return nil, fmt.Errorf("new augmentation bounties are retired")
	}
	if msg == nil || msg.Id == "" {
		return nil, fmt.Errorf("bounty id required")
	}
	if msg.Sponsor == "" {
		return nil, fmt.Errorf("sponsor required")
	}
	if msg.TargetFactId == "" {
		return nil, fmt.Errorf("target_fact_id required")
	}
	targetFact, ok := m.keeper.GetFact(ctx, msg.TargetFactId)
	if !ok {
		return nil, fmt.Errorf("target fact %s not found", msg.TargetFactId)
	}
	if _, exists := m.keeper.GetAugmentationBounty(ctx, msg.Id); exists {
		return nil, fmt.Errorf("bounty %s already exists", msg.Id)
	}
	if msg.MaxVariants == 0 {
		return nil, fmt.Errorf("max_variants must be > 0")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	// Wave 4: escrow = reward_per_variant × max_variants × (1 + SUPERIOR bonus cap).
	// Simpler: lock base × max and charge the bonus against the reserve at
	// payout time. This keeps the basic case cheap for sponsors.
	escrowAmount := sdkmath.NewIntFromUint64(msg.RewardPerVariant).Mul(sdkmath.NewIntFromUint64(uint64(msg.MaxVariants)))
	// Pad for SUPERIOR bonus.
	params, _ := m.keeper.GetParams(ctx)
	bonusBps := params.ReformulationSuperiorBonusBps
	if bonusBps == 0 {
		bonusBps = 500_000
	}
	padding := escrowAmount.Mul(sdkmath.NewIntFromUint64(bonusBps)).Quo(sdkmath.NewIntFromUint64(1_000_000))
	escrowTotal := escrowAmount.Add(padding)

	if !escrowTotal.IsZero() {
		if err := m.keeper.EscrowAugmentationBounty(ctx, msg.Sponsor, msg.Id, escrowTotal); err != nil {
			return nil, fmt.Errorf("escrow failed: %w", err)
		}
	}

	bounty := &types.AugmentationBounty{
		Id:               msg.Id,
		SponsorAddress:   msg.Sponsor,
		TargetFactId:     msg.TargetFactId,
		RewardPerVariant: msg.RewardPerVariant,
		MaxVariants:      msg.MaxVariants,
		AcceptedVariants: 0,
		CreatedAtBlock:   height,
		ExpiresAtBlock:   msg.ExpiresAtBlock,
		Active:           true,
		Description:      msg.Description,
		EscrowLocked:     escrowTotal.String(),
		MethodologyId:    targetFact.MethodId,
	}
	if err := m.keeper.SetAugmentationBounty(ctx, bounty); err != nil {
		return nil, err
	}
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.augmentation_bounty_created",
		sdk.NewAttribute("bounty_id", bounty.Id),
		sdk.NewAttribute("sponsor", bounty.SponsorAddress),
		sdk.NewAttribute("target_fact_id", bounty.TargetFactId),
		sdk.NewAttribute("reward_per_variant", fmt.Sprintf("%d", bounty.RewardPerVariant)),
		sdk.NewAttribute("max_variants", fmt.Sprintf("%d", bounty.MaxVariants)),
		sdk.NewAttribute("escrow_locked", bounty.EscrowLocked),
		sdk.NewAttribute("methodology_id", bounty.MethodologyId),
	))
	return &types.MsgCreateAugmentationBountyResponse{}, nil
}

func (m *msgServer) submitAugmentationLegacy(ctx context.Context, msg *types.MsgSubmitAugmentation) (*types.MsgSubmitAugmentationResponse, error) {
	neutral, err := m.keeper.ReviewNeutralityEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if neutral {
		return nil, fmt.Errorf("new augmentation admission is retired; existing ballots remain settleable")
	}
	if msg == nil || msg.Id == "" {
		return nil, fmt.Errorf("augmentation id required")
	}
	if msg.OriginalFactId == "" {
		return nil, fmt.Errorf("original_fact_id required")
	}
	if _, ok := m.keeper.GetFact(ctx, msg.OriginalFactId); !ok {
		return nil, fmt.Errorf("original fact %s not found", msg.OriginalFactId)
	}
	if _, exists := m.keeper.GetAugmentation(ctx, msg.Id); exists {
		return nil, fmt.Errorf("augmentation %s already exists", msg.Id)
	}
	if msg.VariantContent == "" {
		return nil, fmt.Errorf("variant_content required")
	}

	if msg.BountyId != "" {
		bounty, ok := m.keeper.GetAugmentationBounty(ctx, msg.BountyId)
		if !ok {
			return nil, fmt.Errorf("bounty %s not found", msg.BountyId)
		}
		if !bounty.Active {
			return nil, fmt.Errorf("bounty %s is not active", msg.BountyId)
		}
		if bounty.AcceptedVariants >= bounty.MaxVariants {
			return nil, fmt.Errorf("bounty %s is saturated", msg.BountyId)
		}
		if bounty.TargetFactId != msg.OriginalFactId {
			return nil, fmt.Errorf("bounty target does not match original_fact_id")
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	aug := &types.Augmentation{
		Id:                    msg.Id,
		BountyId:              msg.BountyId,
		OriginalFactId:        msg.OriginalFactId,
		VariantContent:        msg.VariantContent,
		VariantReasoningTrace: msg.VariantReasoningTrace,
		Submitter:             msg.Submitter,
		CreatedAtBlock:        height,
		Accepted:              false,
	}
	if err := m.keeper.SetAugmentation(ctx, aug); err != nil {
		return nil, err
	}
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.augmentation_submitted",
		sdk.NewAttribute("augmentation_id", aug.Id),
		sdk.NewAttribute("original_fact_id", aug.OriginalFactId),
		sdk.NewAttribute("bounty_id", aug.BountyId),
		sdk.NewAttribute("submitter", aug.Submitter),
	))
	return &types.MsgSubmitAugmentationResponse{}, nil
}
