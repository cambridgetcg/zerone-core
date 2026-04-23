package keeper

import (
	"context"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// authtypesNewModuleAddress is a thin wrapper used by genesis.go so that
// file need not import the auth types package directly.
func authtypesNewModuleAddress(name string) []byte {
	return authtypes.NewModuleAddress(name)
}

// ─── Route B Wave 8: genesis-export iterators ────────────────────────────
//
// Helpers that walk every Route B sub-namespace for full-state round-trip.
// Kept in one file so genesis export can read them coherently.

// IterateTokenizerSpecHistory yields every historical TokenizerSpec.
func (k Keeper) IterateTokenizerSpecHistory(ctx context.Context, cb func(*types.TokenizerSpec) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.TokenizerSpecHistoryKeyPrefix, prefixEndBytes(types.TokenizerSpecHistoryKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var s types.TokenizerSpec
		if err := proto.Unmarshal(iter.Value(), &s); err != nil {
			continue
		}
		if cb(&s) {
			return
		}
	}
}

// IterateTraceSchemaHistory yields every historical TraceSchema.
func (k Keeper) IterateTraceSchemaHistory(ctx context.Context, cb func(*types.TraceSchema) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.TraceSchemaHistoryKeyPrefix, prefixEndBytes(types.TraceSchemaHistoryKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var s types.TraceSchema
		if err := proto.Unmarshal(iter.Value(), &s); err != nil {
			continue
		}
		if cb(&s) {
			return
		}
	}
}

// IterateTrainingAttestations yields every stored attestation.
func (k Keeper) IterateTrainingAttestations(ctx context.Context, cb func(*types.TrainingAttestation) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.TrainingAttestationKeyPrefix, prefixEndBytes(types.TrainingAttestationKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var a types.TrainingAttestation
		if err := proto.Unmarshal(iter.Value(), &a); err != nil {
			continue
		}
		if cb(&a) {
			return
		}
	}
}

// IterateContributionRecords yields every stored ContributionRecord.
func (k Keeper) IterateContributionRecords(ctx context.Context, cb func(*types.ContributionRecord) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ContributionByModelKeyPrefix, prefixEndBytes(types.ContributionByModelKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var r types.ContributionRecord
		if err := proto.Unmarshal(iter.Value(), &r); err != nil {
			continue
		}
		if cb(&r) {
			return
		}
	}
}

// SetTokenizerSpecHistory writes only the history entry for a given
// version — no singleton update. Used by genesis import to rebuild
// history in order before the current spec is written.
func (k Keeper) SetTokenizerSpecHistory(ctx context.Context, spec *types.TokenizerSpec) error {
	if spec == nil || spec.Version == 0 {
		return nil
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := marshalOpts.Marshal(spec)
	if err != nil {
		return err
	}
	return store.Set(types.TokenizerSpecHistoryKey(spec.Version), bz)
}

// SetTraceSchemaHistory writes only the history entry; no singleton update.
func (k Keeper) SetTraceSchemaHistory(ctx context.Context, s *types.TraceSchema) error {
	if s == nil || s.Version == 0 {
		return nil
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := marshalOpts.Marshal(s)
	if err != nil {
		return err
	}
	return store.Set(types.TraceSchemaHistoryKey(s.Version), bz)
}

// IterateAllContributionChallenges yields every challenge regardless of status.
// (IterateOpenContributionChallenges already exists for the open-only view.)
func (k Keeper) IterateAllContributionChallenges(ctx context.Context, cb func(*types.ContributionChallenge) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ContributionChallengeKeyPrefix, prefixEndBytes(types.ContributionChallengeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var c types.ContributionChallenge
		if err := proto.Unmarshal(iter.Value(), &c); err != nil {
			continue
		}
		if cb(&c) {
			return
		}
	}
}
