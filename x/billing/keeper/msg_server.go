package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/billing/types"
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

// RegisterProvider registers a knowledge API provider with staked tokens.
func (k msgServer) RegisterProvider(goCtx context.Context, msg *types.MsgRegisterProvider) (*types.MsgRegisterProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	if _, found := k.GetProvider(ctx, msg.Sender); found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderExists, msg.Sender)
	}

	stakeAmt := new(big.Int)
	if _, ok := stakeAmt.SetString(msg.Stake, 10); !ok || stakeAmt.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.Stake)
	}

	params := k.GetParams(ctx)
	minStake := new(big.Int)
	minStake.SetString(params.MinProviderStake, 10)
	if stakeAmt.Cmp(minStake) < 0 {
		return nil, fmt.Errorf("%w: need %s, got %s", types.ErrInsufficientStake, params.MinProviderStake, msg.Stake)
	}

	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to stake: %w", err)
	}

	provider := &types.Provider{
		Address:      msg.Sender,
		Name:         msg.Name,
		Domains:      msg.Domains,
		StakeAmount:  msg.Stake,
		Active:       true,
		TotalQueries: 0,
		TotalRevenue: "0",
		RegisteredAt: uint64(ctx.BlockHeight()),
	}
	k.SetProvider(ctx, provider)

	for _, domain := range msg.Domains {
		k.SetDomainIndex(ctx, domain, msg.Sender)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.billing.provider_registered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("stake", msg.Stake),
			sdk.NewAttribute("domains", fmt.Sprintf("%v", msg.Domains)),
		),
	)

	return &types.MsgRegisterProviderResponse{}, nil
}

// DeregisterProvider removes a provider and refunds their stake.
func (k msgServer) DeregisterProvider(goCtx context.Context, msg *types.MsgDeregisterProvider) (*types.MsgDeregisterProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	provider, found := k.GetProvider(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, msg.Sender)
	}

	stakeAmt := new(big.Int)
	stakeAmt.SetString(provider.StakeAmount, 10)
	if stakeAmt.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, senderAddr, coins); err != nil {
			return nil, fmt.Errorf("failed to refund stake: %w", err)
		}
	}

	for _, domain := range provider.Domains {
		k.DeleteDomainIndex(ctx, domain, msg.Sender)
	}

	k.DeleteProvider(ctx, msg.Sender)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.billing.provider_deregistered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("stake_refunded", provider.StakeAmount),
		),
	)

	return &types.MsgDeregisterProviderResponse{}, nil
}

// QueryFact handles a single fact query: quote + pay + distribute in one tx.
func (k msgServer) QueryFact(goCtx context.Context, msg *types.MsgQueryFact) (*types.MsgQueryFactResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	callerAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	provider, found := k.GetProvider(ctx, msg.Provider)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, msg.Provider)
	}
	if !provider.Active {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotActive, msg.Provider)
	}

	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider address: %w", err)
	}

	currentBlock := uint64(ctx.BlockHeight())
	factIds := []string{msg.FactId}

	// Calculate pricing
	totalPrice, breakdown := k.CalculateQueryPrice(ctx, factIds, currentBlock)
	if totalPrice.Sign() == 0 {
		return nil, types.ErrZeroPayment
	}

	// Calculate distribution
	distribution := k.CalculateDistribution(ctx, totalPrice, factIds)

	// Execute distribution atomically
	if err := k.ExecuteDistribution(ctx, callerAddr, providerAddr, distribution); err != nil {
		return nil, fmt.Errorf("failed to execute distribution: %w", err)
	}

	// Increment citation count
	if err := k.knowledgeKeeper.IncrementCitationCount(ctx, msg.FactId); err != nil {
		k.Logger(ctx).Error("failed to increment citation count", "fact_id", msg.FactId, "error", err)
	}

	// Update provider stats
	provider.TotalQueries++
	revenue := new(big.Int)
	revenue.SetString(provider.TotalRevenue, 10)
	providerShare := new(big.Int)
	providerShare.SetString(distribution.ProviderShare, 10)
	revenue.Add(revenue, providerShare)
	provider.TotalRevenue = revenue.String()
	k.SetProvider(ctx, provider)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.billing.fact_queried",
			sdk.NewAttribute("caller", msg.Sender),
			sdk.NewAttribute("provider", msg.Provider),
			sdk.NewAttribute("fact_id", msg.FactId),
			sdk.NewAttribute("total_price", totalPrice.String()),
		),
	)

	// Build quote response
	var quote *types.QueryQuote
	if len(breakdown) > 0 {
		quote = &types.QueryQuote{
			FactId:         msg.FactId,
			BasePrice:      breakdown[0].BaseCost,
			EffectivePrice: breakdown[0].TotalPrice,
		}
	}

	return &types.MsgQueryFactResponse{Quote: quote}, nil
}

// BatchQueryFacts handles batch fact queries: quote + pay + distribute in one tx.
func (k msgServer) BatchQueryFacts(goCtx context.Context, msg *types.MsgBatchQueryFacts) (*types.MsgBatchQueryFactsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	callerAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	provider, found := k.GetProvider(ctx, msg.Provider)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, msg.Provider)
	}
	if !provider.Active {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotActive, msg.Provider)
	}

	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider address: %w", err)
	}

	currentBlock := uint64(ctx.BlockHeight())

	totalPrice, breakdown := k.CalculateQueryPrice(ctx, msg.FactIds, currentBlock)
	if totalPrice.Sign() == 0 {
		return nil, types.ErrZeroPayment
	}

	distribution := k.CalculateDistribution(ctx, totalPrice, msg.FactIds)

	if err := k.ExecuteDistribution(ctx, callerAddr, providerAddr, distribution); err != nil {
		return nil, fmt.Errorf("failed to execute distribution: %w", err)
	}

	// Increment citation counts
	for _, factId := range msg.FactIds {
		if err := k.knowledgeKeeper.IncrementCitationCount(ctx, factId); err != nil {
			k.Logger(ctx).Error("failed to increment citation count", "fact_id", factId, "error", err)
		}
	}

	// Update provider stats
	provider.TotalQueries += uint64(len(msg.FactIds))
	revenue := new(big.Int)
	revenue.SetString(provider.TotalRevenue, 10)
	providerShare := new(big.Int)
	providerShare.SetString(distribution.ProviderShare, 10)
	revenue.Add(revenue, providerShare)
	provider.TotalRevenue = revenue.String()
	k.SetProvider(ctx, provider)

	// Build quote responses
	var quotes []*types.QueryQuote
	for _, b := range breakdown {
		quotes = append(quotes, &types.QueryQuote{
			FactId:         b.FactId,
			BasePrice:      b.BaseCost,
			EffectivePrice: b.TotalPrice,
		})
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.billing.batch_facts_queried",
			sdk.NewAttribute("caller", msg.Sender),
			sdk.NewAttribute("provider", msg.Provider),
			sdk.NewAttribute("fact_count", fmt.Sprintf("%d", len(msg.FactIds))),
			sdk.NewAttribute("total_price", totalPrice.String()),
		),
	)

	return &types.MsgBatchQueryFactsResponse{
		Quotes:     quotes,
		TotalPrice: totalPrice.String(),
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
	k.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.billing.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
