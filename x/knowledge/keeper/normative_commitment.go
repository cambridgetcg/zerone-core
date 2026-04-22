package keeper

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// SetNormativeCommitment stores (or updates) a commitment in the registry.
// Governance-amendable; not subject to verification consensus.
func (k Keeper) SetNormativeCommitment(ctx context.Context, c *types.NormativeCommitment) error {
	if c == nil || c.Id == "" {
		return nil
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := marshalOpts.Marshal(c)
	if err != nil {
		return err
	}
	return store.Set(types.NormativeCommitmentKey(c.Id), bz)
}

// GetNormativeCommitment fetches a commitment by id.
func (k Keeper) GetNormativeCommitment(ctx context.Context, id string) (*types.NormativeCommitment, bool) {
	if id == "" {
		return nil, false
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.NormativeCommitmentKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var c types.NormativeCommitment
	if err := proto.Unmarshal(bz, &c); err != nil {
		return nil, false
	}
	return &c, true
}

// IterateNormativeCommitments yields every registered commitment.
func (k Keeper) IterateNormativeCommitments(ctx context.Context, cb func(*types.NormativeCommitment) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.NormativeCommitmentKeyPrefix, prefixEndBytes(types.NormativeCommitmentKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var c types.NormativeCommitment
		if err := proto.Unmarshal(iter.Value(), &c); err != nil {
			continue
		}
		if cb(&c) {
			return
		}
	}
}

// GetAllNormativeCommitments returns every registered commitment.
func (k Keeper) GetAllNormativeCommitments(ctx context.Context) []*types.NormativeCommitment {
	var out []*types.NormativeCommitment
	k.IterateNormativeCommitments(ctx, func(c *types.NormativeCommitment) bool {
		out = append(out, c)
		return false
	})
	return out
}

// SeedDefaultCommitments writes an initial set of commitments that the chain
// holds at genesis. These are starting points, not frozen — governance
// amends them over time. Separating these from facts enforces Hume's
// is-ought wall schematically: values are tracked, named, and operationally
// binding where relevant, but they do not enter the confidence machinery.
func (k Keeper) SeedDefaultCommitments(ctx context.Context) error {
	for _, c := range types.DefaultCommitments() {
		if c == nil {
			continue
		}
		if err := k.SetNormativeCommitment(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
