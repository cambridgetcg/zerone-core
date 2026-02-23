package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/channels/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// OpenChannel opens a new payment channel.
func (m msgServer) OpenChannel(goCtx context.Context, msg *types.MsgOpenChannel) (*types.MsgOpenChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	// Validate payer != receiver
	if msg.Payer == msg.Receiver {
		return nil, fmt.Errorf("payer and receiver cannot be the same address")
	}

	// Check deposit >= MinDeposit
	minDeposit, _ := new(big.Int).SetString(params.MinDeposit, 10)
	deposit, ok := new(big.Int).SetString(msg.Deposit, 10)
	if !ok || deposit.Sign() <= 0 {
		return nil, types.ErrInsufficientDeposit
	}
	if minDeposit != nil && deposit.Cmp(minDeposit) < 0 {
		return nil, types.ErrInsufficientDeposit
	}

	// Check timeout bounds
	if msg.TimeoutBlocks < params.MinTimeoutBlocks {
		return nil, fmt.Errorf("timeout blocks must be >= %d", params.MinTimeoutBlocks)
	}
	if msg.TimeoutBlocks > params.MaxTimeoutBlocks {
		return nil, types.ErrChannelDurationExceeded
	}

	// Check max channels per pair
	openCount := m.Keeper.GetOpenChannelCountForPair(ctx, msg.Payer, msg.Receiver)
	if openCount >= params.MaxChannelsPerPair {
		return nil, types.ErrMaxChannelsExceeded
	}

	// Transfer deposit from payer to module account
	payerAddr, err := sdk.AccAddressFromBech32(msg.Payer)
	if err != nil {
		return nil, fmt.Errorf("invalid payer address: %w", err)
	}
	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(deposit)))
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, payerAddr, types.ModuleName, depositCoins); err != nil {
			return nil, fmt.Errorf("failed to transfer deposit: %w", err)
		}
	}

	// Charge open fee if set
	openFee, _ := new(big.Int).SetString(params.ChannelOpenFee, 10)
	if openFee != nil && openFee.Sign() > 0 {
		feeCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(openFee)))
		if m.Keeper.bankKeeper != nil {
			if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, payerAddr, types.ModuleName, feeCoins); err != nil {
				return nil, fmt.Errorf("failed to transfer open fee: %w", err)
			}
		}
	}

	// Generate channel ID
	counter := m.Keeper.GetNextChannelId(ctx)
	channelId := fmt.Sprintf("pc-%d-%d", ctx.BlockHeight(), counter)

	// Create channel
	ch := &types.PaymentChannel{
		ChannelId:           channelId,
		Payer:               msg.Payer,
		Receiver:            msg.Receiver,
		Deposited:           msg.Deposit,
		Spent:               "0",
		Available:           msg.Deposit,
		Status:              types.ChannelStatusOpen,
		OpenedAtBlock:       uint64(ctx.BlockHeight()),
		ExpiresAtBlock:      uint64(ctx.BlockHeight()) + msg.TimeoutBlocks,
		Nonce:               0,
		LastStateHash:       "",
		SettlementFrequency: params.DefaultSettlementFreq,
		LastSettlementBlock: uint64(ctx.BlockHeight()),
	}

	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.channel_opened",
		sdk.NewAttribute("channel_id", channelId),
		sdk.NewAttribute("payer", msg.Payer),
		sdk.NewAttribute("receiver", msg.Receiver),
		sdk.NewAttribute("deposit", msg.Deposit),
	))

	return &types.MsgOpenChannelResponse{ChannelId: channelId}, nil
}

// DepositChannel adds funds to an existing channel.
func (m msgServer) DepositChannel(goCtx context.Context, msg *types.MsgDepositChannel) (*types.MsgDepositChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}
	if msg.Depositor != ch.Payer {
		return nil, types.ErrNotChannelParty.Wrap("only payer can deposit")
	}

	amount, ok := new(big.Int).SetString(msg.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid deposit amount")
	}

	// Transfer from depositor to module account
	depositorAddr, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, fmt.Errorf("invalid depositor address: %w", err)
	}
	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(amount)))
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, depositorAddr, types.ModuleName, depositCoins); err != nil {
			return nil, fmt.Errorf("failed to transfer deposit: %w", err)
		}
	}

	// Update channel balances
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	available, _ := new(big.Int).SetString(ch.Available, 10)
	deposited.Add(deposited, amount)
	available.Add(available, amount)
	ch.Deposited = deposited.String()
	ch.Available = available.String()

	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.channel_deposited",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("depositor", msg.Depositor),
		sdk.NewAttribute("amount", msg.Amount),
	))

	return &types.MsgDepositChannelResponse{}, nil
}

// UpdateState submits an off-chain state update with dual signatures.
func (m msgServer) UpdateState(goCtx context.Context, msg *types.MsgUpdateState) (*types.MsgUpdateStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}

	// Verify sender is payer or receiver
	if msg.Sender != ch.Payer && msg.Sender != ch.Receiver {
		return nil, types.ErrNotChannelParty
	}

	if msg.Update == nil {
		return nil, types.ErrInvalidStateUpdate
	}

	// Verify both payer and receiver signatures
	payload := types.ChannelSigningPayload("state", ctx.ChainID(), msg.ChannelId, msg.Update.Nonce, msg.Update.Spent)
	if err := types.VerifyPackedSignature(payload, []byte(msg.Update.PayerSignature), ch.Payer); err != nil {
		return nil, types.ErrInvalidSignature.Wrap("payer: " + err.Error())
	}
	if err := types.VerifyPackedSignature(payload, []byte(msg.Update.ReceiverSignature), ch.Receiver); err != nil {
		return nil, types.ErrInvalidSignature.Wrap("receiver: " + err.Error())
	}

	// Verify nonce is higher
	if msg.Update.Nonce <= ch.Nonce {
		return nil, types.ErrInvalidNonce
	}

	// Verify spent is non-negative and <= deposited
	spent, ok := new(big.Int).SetString(msg.Update.Spent, 10)
	if !ok || spent.Sign() < 0 {
		return nil, fmt.Errorf("invalid spent amount: must be non-negative")
	}
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	if spent.Cmp(deposited) > 0 {
		return nil, types.ErrSpentExceedsDeposit
	}

	// Update channel state
	ch.Nonce = msg.Update.Nonce
	ch.Spent = msg.Update.Spent
	available := new(big.Int).Sub(deposited, spent)
	ch.Available = available.String()
	ch.LastStateHash = msg.Update.StateHash

	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.state_updated",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("nonce", fmt.Sprintf("%d", msg.Update.Nonce)),
		sdk.NewAttribute("spent", msg.Update.Spent),
	))

	return &types.MsgUpdateStateResponse{}, nil
}

// CloseChannel cooperatively closes a channel.
func (m msgServer) CloseChannel(goCtx context.Context, msg *types.MsgCloseChannel) (*types.MsgCloseChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}

	// Verify closer is payer or receiver
	if msg.Closer != ch.Payer && msg.Closer != ch.Receiver {
		return nil, types.ErrNotChannelParty
	}

	// Verify counterparty signature over canonical payload
	counterparty := ch.Receiver
	if msg.Closer == ch.Receiver {
		counterparty = ch.Payer
	}
	payload := types.ChannelSigningPayload("close", ctx.ChainID(), msg.ChannelId, msg.FinalNonce, msg.FinalSpent)
	if err := types.VerifyPackedSignature(payload, msg.CounterpartySignature, counterparty); err != nil {
		return nil, types.ErrInvalidSignature.Wrap(err.Error())
	}

	// Validate final spent
	spent, ok := new(big.Int).SetString(msg.FinalSpent, 10)
	if !ok || spent.Sign() < 0 {
		return nil, fmt.Errorf("invalid final spent: must be non-negative")
	}
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	if spent.Cmp(deposited) > 0 {
		return nil, types.ErrSpentExceedsDeposit
	}

	payerRefund := new(big.Int).Sub(deposited, spent)
	receiverPayout := new(big.Int).Set(spent)

	// Transfer refund to payer
	if payerRefund.Sign() > 0 && m.Keeper.bankKeeper != nil {
		payerAddr, err := sdk.AccAddressFromBech32(ch.Payer)
		if err != nil {
			return nil, fmt.Errorf("invalid payer address: %w", err)
		}
		refundCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(payerRefund)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payerAddr, refundCoins); err != nil {
			return nil, fmt.Errorf("failed to refund payer: %w", err)
		}
	}

	// Transfer payout to receiver
	if receiverPayout.Sign() > 0 && m.Keeper.bankKeeper != nil {
		receiverAddr, err := sdk.AccAddressFromBech32(ch.Receiver)
		if err != nil {
			return nil, fmt.Errorf("invalid receiver address: %w", err)
		}
		payoutCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(receiverPayout)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiverAddr, payoutCoins); err != nil {
			return nil, fmt.Errorf("failed to pay receiver: %w", err)
		}
	}

	ch.Status = types.ChannelStatusSettled
	ch.Nonce = msg.FinalNonce
	ch.Spent = msg.FinalSpent
	available := new(big.Int).Sub(deposited, spent)
	ch.Available = available.String()
	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.channel_closed",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("payer_refund", payerRefund.String()),
		sdk.NewAttribute("receiver_payout", receiverPayout.String()),
	))

	return &types.MsgCloseChannelResponse{}, nil
}

// DisputeChannel opens a dispute on a channel.
func (m msgServer) DisputeChannel(goCtx context.Context, msg *types.MsgDisputeChannel) (*types.MsgDisputeChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen && ch.Status != types.ChannelStatusClosing && ch.Status != types.ChannelStatusDisputed {
		return nil, types.ErrChannelNotOpen.Wrap("channel must be open, closing, or disputed to dispute")
	}

	// Verify disputer is payer or receiver
	if msg.Disputer != ch.Payer && msg.Disputer != ch.Receiver {
		return nil, types.ErrNotChannelParty
	}

	// Verify counterparty signature proving they agreed to this state
	counterparty := ch.Receiver
	if msg.Disputer == ch.Receiver {
		counterparty = ch.Payer
	}
	payload := types.ChannelSigningPayload("dispute", ctx.ChainID(), msg.ChannelId, msg.ClaimedNonce, msg.ClaimedSpent)
	if err := types.VerifyPackedSignature(payload, msg.ProofSignature, counterparty); err != nil {
		return nil, types.ErrInvalidSignature.Wrap(err.Error())
	}

	// Validate claimed spent
	claimedSpent, ok := new(big.Int).SetString(msg.ClaimedSpent, 10)
	if !ok || claimedSpent.Sign() < 0 {
		return nil, types.ErrSpentExceedsDeposit.Wrap("claimed spent must be non-negative")
	}
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	if deposited != nil && claimedSpent.Cmp(deposited) > 0 {
		return nil, types.ErrSpentExceedsDeposit.Wrap("claimed spent exceeds deposit")
	}

	// Check no active dispute (or nonce must be higher)
	existingDispute, hasDispute := m.Keeper.GetDispute(ctx, msg.ChannelId)
	if hasDispute && !existingDispute.Resolved {
		if msg.ClaimedNonce <= existingDispute.DisputedNonce {
			return nil, types.ErrDisputeAlreadyActive
		}
	}

	// Set channel to disputed
	ch.Status = types.ChannelStatusDisputed
	deadline := uint64(ctx.BlockHeight()) + params.DisputeWindowBlocks
	ch.DisputeDeadline = deadline
	ch.DisputeNonce = msg.ClaimedNonce
	m.Keeper.SetChannel(ctx, ch)

	// Create dispute record
	dispute := &types.ChannelDispute{
		ChannelId:        msg.ChannelId,
		Disputer:         msg.Disputer,
		DisputedNonce:    msg.ClaimedNonce,
		DisputedSpent:    msg.ClaimedSpent,
		DisputeStateHash: "",
		DeadlineBlock:    deadline,
		Resolved:         false,
	}
	m.Keeper.SetDispute(ctx, dispute)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.channel_disputed",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("disputer", msg.Disputer),
		sdk.NewAttribute("deadline", fmt.Sprintf("%d", deadline)),
	))

	return &types.MsgDisputeChannelResponse{}, nil
}

// ClaimExpired claims refund from an expired channel.
func (m msgServer) ClaimExpired(goCtx context.Context, msg *types.MsgClaimExpired) (*types.MsgClaimExpiredResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}

	// Verify claimer is payer
	if msg.Claimer != ch.Payer {
		return nil, types.ErrNotChannelParty.Wrap("only payer can claim expired channel")
	}

	// Verify channel is expired
	if uint64(ctx.BlockHeight()) <= ch.ExpiresAtBlock {
		return nil, types.ErrNotExpired
	}

	// Calculate distribution
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	spent, _ := new(big.Int).SetString(ch.Spent, 10)
	if spent == nil {
		spent = new(big.Int)
	}
	payerRefund := new(big.Int).Sub(deposited, spent)

	// Refund payer
	if payerRefund.Sign() > 0 && m.Keeper.bankKeeper != nil {
		payerAddr, err := sdk.AccAddressFromBech32(ch.Payer)
		if err != nil {
			return nil, fmt.Errorf("invalid payer address: %w", err)
		}
		refundCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(payerRefund)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payerAddr, refundCoins); err != nil {
			return nil, fmt.Errorf("failed to refund payer: %w", err)
		}
	}

	// Pay receiver for spent amount
	if spent.Sign() > 0 && m.Keeper.bankKeeper != nil {
		receiverAddr, err := sdk.AccAddressFromBech32(ch.Receiver)
		if err != nil {
			return nil, fmt.Errorf("invalid receiver address: %w", err)
		}
		payoutCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(spent)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiverAddr, payoutCoins); err != nil {
			return nil, fmt.Errorf("failed to pay receiver: %w", err)
		}
	}

	ch.Status = types.ChannelStatusSettled
	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.expired_claimed",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("refunded", payerRefund.String()),
	))

	return &types.MsgClaimExpiredResponse{RefundedAmount: payerRefund.String()}, nil
}

// UpdateParams handles MsgUpdateParams -- governance-gated parameter update.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if m.Keeper.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	m.Keeper.SetParams(ctx, msg.Params)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.params_updated",
		sdk.NewAttribute("authority", msg.Authority),
	))
	return &types.MsgUpdateParamsResponse{}, nil
}
