package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/evidence_mgmt/types"
)

type Keeper struct {
	storeService   store.KVStoreService
	cdc            codec.BinaryCodec
	authority      string
	stakingKeeper  types.StakingKeeper
	disputesKeeper types.DisputesKeeper
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	sk types.StakingKeeper,
	dk types.DisputesKeeper,
) Keeper {
	return Keeper{
		storeService:   storeService,
		cdc:            cdc,
		authority:      authority,
		stakingKeeper:  sk,
		disputesKeeper: dk,
	}
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

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

// ---------- Params ----------

func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

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

// ---------- Counters ----------

func (k Keeper) GetNextEvidenceID(ctx context.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.EvidenceCounterKey)
	if err != nil || bz == nil {
		bz = make([]byte, 8)
	}
	counter := binary.BigEndian.Uint64(bz)
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	_ = kvStore.Set(types.EvidenceCounterKey, newBz)
	return counter
}

func (k Keeper) GetNextVerificationID(ctx context.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.VerificationCounterKey)
	if err != nil || bz == nil {
		bz = make([]byte, 8)
	}
	counter := binary.BigEndian.Uint64(bz)
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	_ = kvStore.Set(types.VerificationCounterKey, newBz)
	return counter
}

// ---------- Evidence CRUD ----------

func (k Keeper) SetEvidence(ctx context.Context, evidence *types.Evidence) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(evidence)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal evidence: %v", err))
	}
	_ = kvStore.Set(types.EvidenceKey(evidence.Id), bz)

	// Submitter index
	_ = kvStore.Set(types.SubmitterIndexKey(evidence.Submitter, evidence.Id), []byte{1})
}

func (k Keeper) GetEvidence(ctx context.Context, id string) (*types.Evidence, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.EvidenceKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var evidence types.Evidence
	if err := proto.Unmarshal(bz, &evidence); err != nil {
		return nil, false
	}
	return &evidence, true
}

func (k Keeper) GetAllEvidences(ctx context.Context) []*types.Evidence {
	var evidences []*types.Evidence
	k.IterateEvidences(ctx, func(e *types.Evidence) bool {
		evidences = append(evidences, e)
		return false
	})
	return evidences
}

func (k Keeper) IterateEvidences(ctx context.Context, cb func(*types.Evidence) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.EvidenceKeyPrefix, prefixEndBytes(types.EvidenceKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var evidence types.Evidence
		if err := proto.Unmarshal(iter.Value(), &evidence); err != nil {
			continue
		}
		if cb(&evidence) {
			break
		}
	}
}

func (k Keeper) GetEvidenceBySubmitter(ctx context.Context, submitter string) []*types.Evidence {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.SubmitterIndexKeyPrefix(submitter)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var evidences []*types.Evidence
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		evidenceID := string(key[len(prefix):])
		if e, found := k.GetEvidence(ctx, evidenceID); found {
			evidences = append(evidences, e)
		}
	}
	return evidences
}

// ---------- VerificationResult CRUD ----------

func (k Keeper) SetVerification(ctx context.Context, v *types.VerificationResult) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal verification: %v", err))
	}
	_ = kvStore.Set(types.VerificationByEvidenceKey(v.EvidenceId, v.Id), bz)
}

func (k Keeper) GetVerificationsByEvidence(ctx context.Context, evidenceID string) []*types.VerificationResult {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.VerificationByEvidencePrefix(evidenceID)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var results []*types.VerificationResult
	for ; iter.Valid(); iter.Next() {
		var v types.VerificationResult
		if err := proto.Unmarshal(iter.Value(), &v); err != nil {
			continue
		}
		results = append(results, &v)
	}
	return results
}

func (k Keeper) GetAllVerifications(ctx context.Context) []*types.VerificationResult {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.VerificationKeyPrefix
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var results []*types.VerificationResult
	for ; iter.Valid(); iter.Next() {
		var v types.VerificationResult
		if err := proto.Unmarshal(iter.Value(), &v); err != nil {
			continue
		}
		results = append(results, &v)
	}
	return results
}

// ---------- Genesis ----------

func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	for _, e := range genState.Evidences {
		k.SetEvidence(ctx, e)
	}
	for _, v := range genState.Verifications {
		k.SetVerification(ctx, v)
	}
	// Restore counters
	if genState.NextEvidenceId > 0 {
		kvStore := k.storeService.OpenKVStore(ctx)
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, genState.NextEvidenceId)
		_ = kvStore.Set(types.EvidenceCounterKey, bz)
	}
	if genState.NextVerificationId > 0 {
		kvStore := k.storeService.OpenKVStore(ctx)
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, genState.NextVerificationId)
		_ = kvStore.Set(types.VerificationCounterKey, bz)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	evidences := k.GetAllEvidences(ctx)
	verifications := k.GetAllVerifications(ctx)

	// Read counters
	kvStore := k.storeService.OpenKVStore(ctx)
	var nextEvidenceID, nextVerificationID uint64
	if bz, err := kvStore.Get(types.EvidenceCounterKey); err == nil && bz != nil {
		nextEvidenceID = binary.BigEndian.Uint64(bz)
	}
	if bz, err := kvStore.Get(types.VerificationCounterKey); err == nil && bz != nil {
		nextVerificationID = binary.BigEndian.Uint64(bz)
	}

	return &types.GenesisState{
		Params:             params,
		Evidences:          evidences,
		Verifications:      verifications,
		NextEvidenceId:     nextEvidenceID,
		NextVerificationId: nextVerificationID,
	}
}
