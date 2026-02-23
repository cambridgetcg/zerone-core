package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/channels/types"
)

// --- Params ---

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx context.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// --- Channel CRUD ---

// SetChannel stores a payment channel and updates indexes.
func (k Keeper) SetChannel(ctx context.Context, ch *types.PaymentChannel) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(ch)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal payment channel: %v", err))
	}
	if err := store.Set(types.ChannelKey(ch.ChannelId), bz); err != nil {
		panic(fmt.Sprintf("failed to set payment channel: %v", err))
	}

	// Update payer and receiver indexes
	if err := store.Set(types.PayerChannelKey(ch.Payer, ch.ChannelId), []byte(ch.ChannelId)); err != nil {
		panic(fmt.Sprintf("failed to set payer channel index: %v", err))
	}
	if err := store.Set(types.ReceiverChannelKey(ch.Receiver, ch.ChannelId), []byte(ch.ChannelId)); err != nil {
		panic(fmt.Sprintf("failed to set receiver channel index: %v", err))
	}
}

// GetChannel retrieves a payment channel by ID.
func (k Keeper) GetChannel(ctx context.Context, channelId string) (*types.PaymentChannel, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ChannelKey(channelId))
	if err != nil || bz == nil {
		return nil, false
	}
	var ch types.PaymentChannel
	if err := proto.Unmarshal(bz, &ch); err != nil {
		return nil, false
	}
	return &ch, true
}

// DeleteChannel removes a payment channel and its indexes.
func (k Keeper) DeleteChannel(ctx context.Context, ch *types.PaymentChannel) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.ChannelKey(ch.ChannelId))
	_ = store.Delete(types.PayerChannelKey(ch.Payer, ch.ChannelId))
	_ = store.Delete(types.ReceiverChannelKey(ch.Receiver, ch.ChannelId))
}

// IterateChannels iterates over all channels. Return true from cb to stop.
func (k Keeper) IterateChannels(ctx context.Context, cb func(*types.PaymentChannel) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ChannelKeyPrefix, prefixEndBytes(types.ChannelKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ch types.PaymentChannel
		if err := proto.Unmarshal(iter.Value(), &ch); err != nil {
			continue
		}
		if cb(&ch) {
			break
		}
	}
}

// GetChannelsByPayer returns all channels for a payer via index.
func (k Keeper) GetChannelsByPayer(ctx context.Context, payer string) []*types.PaymentChannel {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.PayerChannelPrefix(payer)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var channels []*types.PaymentChannel
	for ; iter.Valid(); iter.Next() {
		channelId := string(iter.Value())
		ch, found := k.GetChannel(ctx, channelId)
		if found {
			channels = append(channels, ch)
		}
	}
	return channels
}

// GetChannelsByReceiver returns all channels for a receiver via index.
func (k Keeper) GetChannelsByReceiver(ctx context.Context, receiver string) []*types.PaymentChannel {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ReceiverChannelPrefix(receiver)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var channels []*types.PaymentChannel
	for ; iter.Valid(); iter.Next() {
		channelId := string(iter.Value())
		ch, found := k.GetChannel(ctx, channelId)
		if found {
			channels = append(channels, ch)
		}
	}
	return channels
}

// GetOpenChannelCountForPair returns the number of open channels for a payer-receiver pair.
func (k Keeper) GetOpenChannelCountForPair(ctx context.Context, payer, receiver string) uint64 {
	channels := k.GetChannelsByPayer(ctx, payer)
	var count uint64
	for _, ch := range channels {
		if ch.Receiver == receiver && ch.Status == types.ChannelStatusOpen {
			count++
		}
	}
	return count
}

// --- Dispute CRUD ---

// SetDispute stores a channel dispute.
func (k Keeper) SetDispute(ctx context.Context, d *types.ChannelDispute) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(d)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal channel dispute: %v", err))
	}
	if err := store.Set(types.DisputeKey(d.ChannelId), bz); err != nil {
		panic(fmt.Sprintf("failed to set channel dispute: %v", err))
	}
}

// GetDispute retrieves a dispute by channel ID.
func (k Keeper) GetDispute(ctx context.Context, channelId string) (*types.ChannelDispute, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.DisputeKey(channelId))
	if err != nil || bz == nil {
		return nil, false
	}
	var d types.ChannelDispute
	if err := proto.Unmarshal(bz, &d); err != nil {
		return nil, false
	}
	return &d, true
}

// DeleteDispute removes a dispute.
func (k Keeper) DeleteDispute(ctx context.Context, channelId string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.DisputeKey(channelId))
}

// IterateDisputes iterates over all disputes. Return true from cb to stop.
func (k Keeper) IterateDisputes(ctx context.Context, cb func(*types.ChannelDispute) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.DisputeKeyPrefix, prefixEndBytes(types.DisputeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var d types.ChannelDispute
		if err := proto.Unmarshal(iter.Value(), &d); err != nil {
			continue
		}
		if cb(&d) {
			break
		}
	}
}

// --- Channel ID Counter ---

// GetNextChannelId returns a unique channel ID and increments the counter.
func (k Keeper) GetNextChannelId(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ChannelCounterKey)
	var counter uint64
	if err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	_ = store.Set(types.ChannelCounterKey, newBz)
	return counter
}
