package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/compute_pool/types"
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

// RegisterProvider registers a compute provider with staked tokens.
func (k msgServer) RegisterProvider(goCtx context.Context, msg *types.MsgRegisterProvider) (*types.MsgRegisterProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate service type.
	if !types.IsValidServiceType(msg.ServiceType) {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidServiceType, msg.ServiceType)
	}

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// Parse and validate stake amount.
	stakeAmt := new(big.Int)
	if _, ok := stakeAmt.SetString(msg.Stake, 10); !ok || stakeAmt.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.Stake)
	}

	params := k.GetParams(ctx)

	// Check minimum stake.
	minStake := new(big.Int)
	minStake.SetString(params.MinProviderStake, 10)
	if stakeAmt.Cmp(minStake) < 0 {
		return nil, fmt.Errorf("%w: need %s, got %s", types.ErrInsufficientStake, params.MinProviderStake, msg.Stake)
	}

	// Check price ceiling.
	pricePerCu := new(big.Int)
	if _, ok := pricePerCu.SetString(msg.PricePerCu, 10); !ok || pricePerCu.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.PricePerCu)
	}
	maxPrice := new(big.Int)
	maxPrice.SetString(params.MaxPricePerCu, 10)
	if pricePerCu.Cmp(maxPrice) > 0 {
		return nil, fmt.Errorf("%w: max %s, got %s", types.ErrPriceExceedsCeiling, params.MaxPricePerCu, msg.PricePerCu)
	}

	// Check not already registered.
	if _, found := k.GetProvider(ctx, msg.Sender); found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderExists, msg.Sender)
	}

	// Deduct stake to module account.
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to stake: %w", err)
	}

	// Create provider.
	currentBlock := uint64(ctx.BlockHeight())
	provider := &types.ComputeProvider{
		Address:       msg.Sender,
		ServiceType:   msg.ServiceType,
		Endpoint:      msg.Endpoint,
		PricePerCu:    msg.PricePerCu,
		Stake:         msg.Stake,
		Status:        "active",
		RegisteredAt:  currentBlock,
		TasksServed:   0,
		TasksFailed:   0,
		AvgLatencyMs:  0,
		UptimeBps:     1000000, // 100%
		LastHeartbeat: currentBlock,
	}
	k.SetProvider(ctx, provider)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.compute_pool.provider_registered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("service_type", msg.ServiceType),
			sdk.NewAttribute("stake", msg.Stake),
			sdk.NewAttribute("price_per_cu", msg.PricePerCu),
		),
	)

	return &types.MsgRegisterProviderResponse{}, nil
}

// UnregisterProvider initiates unbonding for a compute provider.
func (k msgServer) UnregisterProvider(goCtx context.Context, msg *types.MsgUnregisterProvider) (*types.MsgUnregisterProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	provider, found := k.GetProvider(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, msg.Sender)
	}

	if provider.Status == "unbonding" {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderUnbonding, msg.Sender)
	}

	// Set status to unbonding.
	provider.Status = "unbonding"
	provider.UnbondingAt = uint64(ctx.BlockHeight())
	k.SetProvider(ctx, provider)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.compute_pool.provider_unbonding",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("unbonding_at", fmt.Sprintf("%d", provider.UnbondingAt)),
		),
	)

	return &types.MsgUnregisterProviderResponse{}, nil
}

// Heartbeat updates the last heartbeat block for a compute provider.
func (k msgServer) Heartbeat(goCtx context.Context, msg *types.MsgHeartbeat) (*types.MsgHeartbeatResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	provider, found := k.GetProvider(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, msg.Sender)
	}

	if provider.Status != "active" && provider.Status != "jailed" {
		return nil, fmt.Errorf("%w: %s (status: %s)", types.ErrProviderNotActive, msg.Sender, provider.Status)
	}

	currentBlock := uint64(ctx.BlockHeight())

	// Update heartbeat.
	provider.LastHeartbeat = currentBlock

	// Reactivate jailed providers on heartbeat.
	if provider.Status == "jailed" {
		provider.Status = "active"
	}

	k.SetProvider(ctx, provider)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.compute_pool.heartbeat",
			sdk.NewAttribute("provider", msg.Sender),
			sdk.NewAttribute("block", fmt.Sprintf("%d", currentBlock)),
		),
	)

	return &types.MsgHeartbeatResponse{}, nil
}

// UpdatePrice sets a pending price change for a compute provider.
func (k msgServer) UpdatePrice(goCtx context.Context, msg *types.MsgUpdatePrice) (*types.MsgUpdatePriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	provider, found := k.GetProvider(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, msg.Sender)
	}

	if provider.Status != "active" {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotActive, msg.Sender)
	}

	// Validate new price.
	newPrice := new(big.Int)
	if _, ok := newPrice.SetString(msg.NewPrice, 10); !ok || newPrice.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.NewPrice)
	}

	params := k.GetParams(ctx)
	maxPrice := new(big.Int)
	maxPrice.SetString(params.MaxPricePerCu, 10)
	if newPrice.Cmp(maxPrice) > 0 {
		return nil, fmt.Errorf("%w: max %s, got %s", types.ErrPriceExceedsCeiling, params.MaxPricePerCu, msg.NewPrice)
	}

	// Set pending price change.
	currentBlock := uint64(ctx.BlockHeight())
	provider.PendingPrice = msg.NewPrice
	provider.PriceChangeAt = currentBlock + params.PriceChangeDelayBlocks
	k.SetProvider(ctx, provider)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.compute_pool.price_update_pending",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("new_price", msg.NewPrice),
			sdk.NewAttribute("effective_at", fmt.Sprintf("%d", provider.PriceChangeAt)),
		),
	)

	return &types.MsgUpdatePriceResponse{}, nil
}

// RedeemCredits converts compute credits to uzrn tokens.
func (k msgServer) RedeemCredits(goCtx context.Context, msg *types.MsgRedeemCredits) (*types.MsgRedeemCreditsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	if msg.Amount == 0 {
		return nil, fmt.Errorf("%w: amount must be positive", types.ErrInvalidAmount)
	}

	credit, found := k.GetCredit(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrInsufficientCredits, msg.Sender)
	}

	if credit.Balance < msg.Amount {
		return nil, fmt.Errorf("%w: have %d, want %d", types.ErrInsufficientCredits, credit.Balance, msg.Amount)
	}

	// 1 credit = 1 uzrn for simplicity.
	uzrnAmount := sdkmath.NewIntFromUint64(msg.Amount)

	// Deduct credits.
	credit.Balance -= msg.Amount
	credit.RedeemedTotal += msg.Amount
	k.SetCredit(ctx, credit)

	// Send tokens from module to sender.
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", uzrnAmount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, senderAddr, coins); err != nil {
		return nil, fmt.Errorf("failed to send redeemed tokens: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.compute_pool.credits_redeemed",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("credits", fmt.Sprintf("%d", msg.Amount)),
			sdk.NewAttribute("uzrn", uzrnAmount.String()),
		),
	)

	return &types.MsgRedeemCreditsResponse{
		RedeemedUzrn: uzrnAmount.String(),
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
			"zerone.compute_pool.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
