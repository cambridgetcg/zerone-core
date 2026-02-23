package keeper

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// ---- RateLimit CRUD ----
// Key layout: 0x01 | channelID | 0x00 | denom

func rateLimitKey(channelID, denom string) []byte {
	key := make([]byte, 0, 1+len(channelID)+1+len(denom))
	key = append(key, types.RateLimitKeyPrefix...)
	key = append(key, []byte(channelID)...)
	key = append(key, 0x00)
	key = append(key, []byte(denom)...)
	return key
}

func (k Keeper) SetRateLimit(ctx context.Context, rl *types.RateLimit) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := rateLimitKey(rl.ChannelId, rl.Denom)
	bz, err := json.Marshal(rl)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal rate limit: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

func (k Keeper) GetRateLimit(ctx context.Context, channelID, denom string) (*types.RateLimit, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := rateLimitKey(channelID, denom)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var rl types.RateLimit
	if err := json.Unmarshal(bz, &rl); err != nil {
		return nil, false
	}
	return &rl, true
}

func (k Keeper) DeleteRateLimit(ctx context.Context, channelID, denom string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(rateLimitKey(channelID, denom))
}

func (k Keeper) GetAllRateLimits(ctx context.Context) []*types.RateLimit {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.RateLimitKeyPrefix, prefixEndBytes(types.RateLimitKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var rateLimits []*types.RateLimit
	for ; iter.Valid(); iter.Next() {
		var rl types.RateLimit
		if err := json.Unmarshal(iter.Value(), &rl); err != nil {
			continue
		}
		rateLimits = append(rateLimits, &rl)
	}
	return rateLimits
}

// ---- PacketFlow CRUD ----
// Key layout: 0x02 | channelID | 0x00 | sequence(8-byte BE)

func packetFlowKey(channelID string, sequence uint64) []byte {
	seqBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBytes, sequence)

	key := make([]byte, 0, 1+len(channelID)+1+8)
	key = append(key, types.PacketFlowPrefix...)
	key = append(key, []byte(channelID)...)
	key = append(key, 0x00)
	key = append(key, seqBytes...)
	return key
}

func (k Keeper) SetPacketFlow(ctx context.Context, flow *types.PacketFlow) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := packetFlowKey(flow.ChannelId, flow.Sequence)
	bz, err := json.Marshal(flow)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal packet flow: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

func (k Keeper) GetPacketFlow(ctx context.Context, channelID string, sequence uint64) (*types.PacketFlow, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := packetFlowKey(channelID, sequence)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var flow types.PacketFlow
	if err := json.Unmarshal(bz, &flow); err != nil {
		return nil, false
	}
	return &flow, true
}

func (k Keeper) DeletePacketFlow(ctx context.Context, channelID string, sequence uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(packetFlowKey(channelID, sequence))
}
