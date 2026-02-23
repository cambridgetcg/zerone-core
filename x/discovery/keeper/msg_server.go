package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/discovery/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// RegisterProfile registers a new agent profile with staked tokens.
func (k msgServer) RegisterProfile(goCtx context.Context, msg *types.MsgRegisterProfile) (*types.MsgRegisterProfileResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// Check profile does not already exist.
	if _, found := k.GetProfile(ctx, msg.Sender); found {
		return nil, fmt.Errorf("%w: %s", types.ErrAgentExists, msg.Sender)
	}

	// Validate capabilities count.
	params := k.GetParams(ctx)
	if uint32(len(msg.Capabilities)) > params.MaxCapabilitiesPerAgent {
		return nil, fmt.Errorf("%w: max %d, got %d", types.ErrMaxCapabilities, params.MaxCapabilitiesPerAgent, len(msg.Capabilities))
	}

	// Parse and validate stake amount.
	stakeAmt := new(big.Int)
	if _, ok := stakeAmt.SetString(msg.Stake, 10); !ok || stakeAmt.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.Stake)
	}

	minStake := new(big.Int)
	minStake.SetString(params.MinRegistrationStake, 10)
	if stakeAmt.Cmp(minStake) < 0 {
		return nil, fmt.Errorf("%w: need %s, got %s", types.ErrInsufficientStake, params.MinRegistrationStake, msg.Stake)
	}

	// Deduct stake from sender to module account.
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to stake: %w", err)
	}

	currentBlock := uint64(ctx.BlockHeight())

	profile := &types.AgentProfile{
		Address:           msg.Sender,
		DisplayName:       msg.DisplayName,
		Capabilities:      msg.Capabilities,
		Domains:           msg.Domains,
		Status:            "active",
		ReputationScore:   500000, // 50% on 1M scale
		RegisteredAtBlock: currentBlock,
		LastActiveBlock:   currentBlock,
		Stake:             msg.Stake,
		Description:       msg.Description,
		Metadata:          msg.Metadata,
	}
	k.SetProfile(ctx, profile)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.discovery.agent_registered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("stake", msg.Stake),
			sdk.NewAttribute("display_name", msg.DisplayName),
		),
	)

	return &types.MsgRegisterProfileResponse{}, nil
}

// UpdateProfile updates an existing agent profile's mutable fields.
func (k msgServer) UpdateProfile(goCtx context.Context, msg *types.MsgUpdateProfile) (*types.MsgUpdateProfileResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	profile, found := k.GetProfile(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrAgentNotFound, msg.Sender)
	}

	// Only update non-empty fields.
	if msg.DisplayName != "" {
		profile.DisplayName = msg.DisplayName
	}
	if msg.Description != "" {
		profile.Description = msg.Description
	}
	if msg.Metadata != "" {
		profile.Metadata = msg.Metadata
	}

	k.SetProfile(ctx, profile)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.discovery.agent_profile_updated",
			sdk.NewAttribute("address", msg.Sender),
		),
	)

	return &types.MsgUpdateProfileResponse{}, nil
}

// Heartbeat signals liveness and reactivates expired profiles.
func (k msgServer) Heartbeat(goCtx context.Context, msg *types.MsgHeartbeat) (*types.MsgHeartbeatResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	profile, found := k.GetProfile(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrAgentNotFound, msg.Sender)
	}

	currentBlock := uint64(ctx.BlockHeight())
	profile.LastActiveBlock = currentBlock

	// Reactivate expired profiles.
	if profile.Status == "expired" {
		profile.Status = "active"
	}

	k.SetProfile(ctx, profile)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.discovery.agent_heartbeat",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("block", fmt.Sprintf("%d", currentBlock)),
		),
	)

	return &types.MsgHeartbeatResponse{}, nil
}

// DeregisterProfile removes an agent profile and refunds the staked tokens.
func (k msgServer) DeregisterProfile(goCtx context.Context, msg *types.MsgDeregisterProfile) (*types.MsgDeregisterProfileResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	profile, found := k.GetProfile(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrAgentNotFound, msg.Sender)
	}

	// Refund stake.
	stakeAmt := new(big.Int)
	stakeAmt.SetString(profile.Stake, 10)
	if stakeAmt.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, senderAddr, coins); err != nil {
			return nil, fmt.Errorf("failed to refund stake: %w", err)
		}
	}

	// Delete profile and clean indexes.
	k.DeleteProfile(ctx, profile)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.discovery.agent_deregistered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("stake_refunded", profile.Stake),
		),
	)

	return &types.MsgDeregisterProfileResponse{
		RefundedAmount: profile.Stake,
	}, nil
}

// UpdateParams handles governance-gated parameter update.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	k.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.discovery.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
