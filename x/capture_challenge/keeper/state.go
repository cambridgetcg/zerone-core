package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

// ---------- Params ----------

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

// ---------- CaptureChallenge CRUD ----------

// SetChallenge stores a challenge in the KV store and maintains domain index.
func (k Keeper) SetChallenge(ctx context.Context, ch *types.CaptureChallenge) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(ch)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal challenge: %v", err))
	}
	_ = kvStore.Set(challengeKey(ch.Id), bz)
}

// GetChallenge retrieves a challenge by ID.
func (k Keeper) GetChallenge(ctx context.Context, id string) (*types.CaptureChallenge, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(challengeKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var ch types.CaptureChallenge
	if err := proto.Unmarshal(bz, &ch); err != nil {
		return nil, false
	}
	return &ch, true
}

// DeleteChallenge removes a challenge from the store.
func (k Keeper) DeleteChallenge(ctx context.Context, id string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(challengeKey(id))
}

// GetAllChallenges returns all challenges.
func (k Keeper) GetAllChallenges(ctx context.Context) []*types.CaptureChallenge {
	var challenges []*types.CaptureChallenge
	k.IterateChallenges(ctx, func(ch *types.CaptureChallenge) bool {
		challenges = append(challenges, ch)
		return false
	})
	return challenges
}

// IterateChallenges iterates over all challenges. Return true from cb to stop.
func (k Keeper) IterateChallenges(ctx context.Context, cb func(*types.CaptureChallenge) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ChallengeKeyPrefix, prefixEndBytes(types.ChallengeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ch types.CaptureChallenge
		if err := proto.Unmarshal(iter.Value(), &ch); err != nil {
			continue
		}
		if cb(&ch) {
			break
		}
	}
}

// ---------- DomainBountyPool CRUD ----------

// SetBountyPool stores a bounty pool in the KV store.
func (k Keeper) SetBountyPool(ctx context.Context, pool *types.DomainBountyPool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(pool)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal bounty pool: %v", err))
	}
	_ = kvStore.Set(bountyPoolKey(pool.Domain), bz)
}

// GetBountyPool retrieves a bounty pool by domain.
func (k Keeper) GetBountyPool(ctx context.Context, domain string) (*types.DomainBountyPool, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(bountyPoolKey(domain))
	if err != nil || bz == nil {
		return nil, false
	}
	var pool types.DomainBountyPool
	if err := proto.Unmarshal(bz, &pool); err != nil {
		return nil, false
	}
	return &pool, true
}

// DeleteBountyPool removes a bounty pool from the store.
func (k Keeper) DeleteBountyPool(ctx context.Context, domain string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(bountyPoolKey(domain))
}

// GetAllBountyPools returns all bounty pools.
func (k Keeper) GetAllBountyPools(ctx context.Context) []*types.DomainBountyPool {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.BountyPoolKeyPrefix, prefixEndBytes(types.BountyPoolKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var pools []*types.DomainBountyPool
	for ; iter.Valid(); iter.Next() {
		var pool types.DomainBountyPool
		if err := proto.Unmarshal(iter.Value(), &pool); err != nil {
			continue
		}
		pools = append(pools, &pool)
	}
	return pools
}

// ---------- Domain Index ----------

// SetDomainIndex adds a challenge to the domain index.
func (k Keeper) SetDomainIndex(ctx context.Context, domain, challengeID string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Set(domainChallengeIndexKey(domain, challengeID), []byte(challengeID))
}

// DeleteDomainIndex removes a challenge from the domain index.
func (k Keeper) DeleteDomainIndex(ctx context.Context, domain, challengeID string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(domainChallengeIndexKey(domain, challengeID))
}

// GetChallengesByDomain returns all challenge IDs for a given domain.
func (k Keeper) GetChallengesByDomain(ctx context.Context, domain string) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := domainChallengePrefix(domain)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var ids []string
	for ; iter.Valid(); iter.Next() {
		ids = append(ids, string(iter.Value()))
	}
	return ids
}

// ---------- Paused Domains ----------

// SetPausedDomain marks a domain as paused until a specific block height.
func (k Keeper) SetPausedDomain(ctx context.Context, domain string, pauseUntilBlock uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, pauseUntilBlock)
	_ = kvStore.Set(pausedDomainKey(domain), bz)
}

// GetPausedDomain returns the pause-until block for a domain, and whether it exists.
func (k Keeper) GetPausedDomain(ctx context.Context, domain string) (uint64, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(pausedDomainKey(domain))
	if err != nil || bz == nil || len(bz) < 8 {
		return 0, false
	}
	return binary.BigEndian.Uint64(bz), true
}

// DeletePausedDomain removes the pause from a domain.
func (k Keeper) DeletePausedDomain(ctx context.Context, domain string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(pausedDomainKey(domain))
}

// ---------- Key Construction Helpers ----------

func challengeKey(id string) []byte {
	return append(types.ChallengeKeyPrefix, []byte(id)...)
}

func bountyPoolKey(domain string) []byte {
	return append(types.BountyPoolKeyPrefix, []byte(domain)...)
}

func domainChallengeIndexKey(domain, challengeID string) []byte {
	return append(types.DomainIndexPrefix, []byte(domain+"/"+challengeID)...)
}

func domainChallengePrefix(domain string) []byte {
	return append(types.DomainIndexPrefix, []byte(domain+"/")...)
}

func pausedDomainKey(domain string) []byte {
	return append(types.PausedDomainKeyPrefix, []byte(domain)...)
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
