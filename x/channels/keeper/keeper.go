package keeper

import (
	"context"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/channels/types"
)

// Keeper manages the channels module's state.
type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec
	authority    string
	bankKeeper   types.BankKeeper
}

// NewKeeper creates a new channels module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	bk types.BankKeeper,
) Keeper {
	return Keeper{
		storeService: storeService,
		cdc:          cdc,
		authority:    authority,
		bankKeeper:   bk,
	}
}

// Logger returns a module-scoped logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// prefixEndBytes returns the end key for prefix iteration (exclusive).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	for _, ch := range genState.Channels {
		k.SetChannel(ctx, ch)
	}
	for _, d := range genState.Disputes {
		k.SetDispute(ctx, d)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	var channels []*types.PaymentChannel
	k.IterateChannels(ctx, func(ch *types.PaymentChannel) bool {
		channels = append(channels, ch)
		return false
	})

	var disputes []*types.ChannelDispute
	k.IterateDisputes(ctx, func(d *types.ChannelDispute) bool {
		disputes = append(disputes, d)
		return false
	})

	return &types.GenesisState{
		Params:   k.GetParams(ctx),
		Channels: channels,
		Disputes: disputes,
	}
}

// GetExpiredChannels returns channels in closing or disputed status past their deadline.
func (k Keeper) GetExpiredChannels(ctx context.Context, currentBlock uint64) []*types.PaymentChannel {
	var channels []*types.PaymentChannel
	k.IterateChannels(ctx, func(ch *types.PaymentChannel) bool {
		if (ch.Status == types.ChannelStatusClosing || ch.Status == types.ChannelStatusDisputed) &&
			ch.DisputeDeadline > 0 && currentBlock > ch.DisputeDeadline {
			channels = append(channels, ch)
		}
		return false
	})
	return channels
}

// GetChannelsForAutoSettlement returns channels due for periodic auto-settlement.
func (k Keeper) GetChannelsForAutoSettlement(ctx context.Context, currentBlock uint64) []*types.PaymentChannel {
	var channels []*types.PaymentChannel
	k.IterateChannels(ctx, func(ch *types.PaymentChannel) bool {
		if ch.Status == types.ChannelStatusOpen &&
			ch.SettlementFrequency > 0 &&
			currentBlock >= ch.LastSettlementBlock+ch.SettlementFrequency {
			spent, _ := new(big.Int).SetString(ch.Spent, 10)
			if spent != nil && spent.Sign() > 0 {
				channels = append(channels, ch)
			}
		}
		return false
	})
	return channels
}

// AutoSettleChannel distributes funds and closes a channel that has passed its deadline.
func (k Keeper) AutoSettleChannel(ctx context.Context, ch *types.PaymentChannel) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	spent, _ := new(big.Int).SetString(ch.Spent, 10)
	if spent == nil {
		spent = new(big.Int)
	}

	// If disputed, resolve dispute and use disputed state
	if ch.Status == types.ChannelStatusDisputed {
		dispute, hasDispute := k.GetDispute(ctx, ch.ChannelId)
		if hasDispute && !dispute.Resolved {
			disputedSpent, ok := new(big.Int).SetString(dispute.DisputedSpent, 10)
			if ok {
				spent = disputedSpent
			}
			ch.Spent = dispute.DisputedSpent
			ch.Nonce = dispute.DisputedNonce
			dispute.Resolved = true
			dispute.Resolution = "auto_settled"
			k.SetDispute(ctx, dispute)
		}
	}

	payerRefund := new(big.Int).Sub(deposited, spent)
	receiverPayout := new(big.Int).Set(spent)

	// Transfer from module to payer (refund)
	if payerRefund.Sign() > 0 && k.bankKeeper != nil {
		payerAddr, err := sdk.AccAddressFromBech32(ch.Payer)
		if err == nil {
			refundCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(payerRefund)))
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payerAddr, refundCoins)
		}
	}

	// Transfer from module to receiver (payout)
	if receiverPayout.Sign() > 0 && k.bankKeeper != nil {
		receiverAddr, err := sdk.AccAddressFromBech32(ch.Receiver)
		if err == nil {
			payoutCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(receiverPayout)))
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiverAddr, payoutCoins)
		}
	}

	ch.Status = types.ChannelStatusSettled
	ch.DisputeDeadline = 0
	available := new(big.Int).Sub(deposited, spent)
	ch.Available = available.String()
	k.SetChannel(ctx, ch)

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.channel_auto_settled",
		sdk.NewAttribute("channel_id", ch.ChannelId),
		sdk.NewAttribute("amount_to_payer", payerRefund.String()),
		sdk.NewAttribute("amount_to_receiver", receiverPayout.String()),
	))
}

// PeriodicSettlement transfers accumulated spent to the receiver on-chain
// without closing the channel.
func (k Keeper) PeriodicSettlement(ctx context.Context, ch *types.PaymentChannel, currentBlock uint64) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	spent, _ := new(big.Int).SetString(ch.Spent, 10)
	if spent == nil || spent.Sign() <= 0 {
		return
	}

	// Transfer spent to receiver
	if k.bankKeeper != nil {
		receiverAddr, err := sdk.AccAddressFromBech32(ch.Receiver)
		if err == nil {
			payoutCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(spent)))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiverAddr, payoutCoins); err != nil {
				return // skip if transfer fails
			}
		}
	}

	// Reset spent, reduce deposited by settled amount
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	newDeposited := new(big.Int).Sub(deposited, spent)
	ch.Deposited = newDeposited.String()
	ch.Spent = "0"
	ch.Available = newDeposited.String()
	ch.LastSettlementBlock = currentBlock
	k.SetChannel(ctx, ch)

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.channels.periodic_settlement",
		sdk.NewAttribute("channel_id", ch.ChannelId),
		sdk.NewAttribute("settled_amount", spent.String()),
	))
}
