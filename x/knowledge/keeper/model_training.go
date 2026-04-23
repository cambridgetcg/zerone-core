package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── TokenizerSpec ───────────────────────────────────────────────────────

// SetTokenizerSpec writes the current TokenizerSpec and archives the prior
// version under the history key. Callers must bump version before calling.
func (k Keeper) SetTokenizerSpec(ctx context.Context, spec *types.TokenizerSpec) error {
	if spec == nil {
		return fmt.Errorf("nil tokenizer spec")
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := marshalOpts.Marshal(spec)
	if err != nil {
		return err
	}
	// Write as current.
	if err := store.Set(types.TokenizerSpecKey, bz); err != nil {
		return err
	}
	// Archive under version key for historical reproducibility.
	return store.Set(types.TokenizerSpecHistoryKey(spec.Version), bz)
}

// GetTokenizerSpec returns the current tokenizer contract.
func (k Keeper) GetTokenizerSpec(ctx context.Context) (*types.TokenizerSpec, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.TokenizerSpecKey)
	if err != nil || bz == nil {
		return nil, false
	}
	var spec types.TokenizerSpec
	if err := proto.Unmarshal(bz, &spec); err != nil {
		return nil, false
	}
	return &spec, true
}

// GetTokenizerSpecAtVersion returns a historical tokenizer contract by
// version. Required for training pipelines that pin to a version from a
// snapshot older than the current spec.
func (k Keeper) GetTokenizerSpecAtVersion(ctx context.Context, version uint64) (*types.TokenizerSpec, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.TokenizerSpecHistoryKey(version))
	if err != nil || bz == nil {
		return nil, false
	}
	var spec types.TokenizerSpec
	if err := proto.Unmarshal(bz, &spec); err != nil {
		return nil, false
	}
	return &spec, true
}

// SeedDefaultTokenizerSpec writes V1 at genesis. Callable by InitGenesis and
// by tests (following the Phase 1 harness pattern).
func (k Keeper) SeedDefaultTokenizerSpec(ctx context.Context) error {
	return k.SetTokenizerSpec(ctx, types.TokenizerSpecV1())
}

// ─── TrainingPipeline ────────────────────────────────────────────────────

// SetTrainingPipeline stores a pipeline record.
func (k Keeper) SetTrainingPipeline(ctx context.Context, p *types.TrainingPipeline) error {
	if p == nil || p.Id == "" {
		return fmt.Errorf("invalid training pipeline")
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := marshalOpts.Marshal(p)
	if err != nil {
		return err
	}
	return store.Set(types.TrainingPipelineKey(p.Id), bz)
}

// GetTrainingPipeline fetches a pipeline by id.
func (k Keeper) GetTrainingPipeline(ctx context.Context, id string) (*types.TrainingPipeline, bool) {
	if id == "" {
		return nil, false
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.TrainingPipelineKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var p types.TrainingPipeline
	if err := proto.Unmarshal(bz, &p); err != nil {
		return nil, false
	}
	return &p, true
}

// IterateTrainingPipelines yields every registered pipeline.
func (k Keeper) IterateTrainingPipelines(ctx context.Context, cb func(*types.TrainingPipeline) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.TrainingPipelineKeyPrefix, prefixEndBytes(types.TrainingPipelineKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var p types.TrainingPipeline
		if err := proto.Unmarshal(iter.Value(), &p); err != nil {
			continue
		}
		if cb(&p) {
			return
		}
	}
}

// ─── ModelCard ───────────────────────────────────────────────────────────

// SetModelCard stores (or updates) a ModelCard. Emits model_card_registered
// on first write and model_card_updated on subsequent writes so off-chain
// observers can track the lineage without polling.
func (k Keeper) SetModelCard(ctx context.Context, m *types.ModelCard) error {
	if m == nil || m.Id == "" {
		return fmt.Errorf("invalid model card")
	}
	_, existed := k.GetModelCard(ctx, m.Id)
	store := k.storeService.OpenKVStore(ctx)
	bz, err := marshalOpts.Marshal(m)
	if err != nil {
		return err
	}
	if err := store.Set(types.ModelCardKey(m.Id), bz); err != nil {
		return err
	}
	if existed {
		k.EmitModelCardEvent(ctx, m, "updated")
	} else {
		k.EmitModelCardEvent(ctx, m, "registered")
	}
	return nil
}

// GetModelCard fetches a model card by id.
func (k Keeper) GetModelCard(ctx context.Context, id string) (*types.ModelCard, bool) {
	if id == "" {
		return nil, false
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ModelCardKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var m types.ModelCard
	if err := proto.Unmarshal(bz, &m); err != nil {
		return nil, false
	}
	return &m, true
}

// IterateModelCards yields every registered ModelCard.
func (k Keeper) IterateModelCards(ctx context.Context, cb func(*types.ModelCard) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ModelCardKeyPrefix, prefixEndBytes(types.ModelCardKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var m types.ModelCard
		if err := proto.Unmarshal(iter.Value(), &m); err != nil {
			continue
		}
		if cb(&m) {
			return
		}
	}
}

// GetModelCardsByPipeline returns every ModelCard produced by a given pipeline.
func (k Keeper) GetModelCardsByPipeline(ctx context.Context, pipelineID string) []*types.ModelCard {
	var out []*types.ModelCard
	k.IterateModelCards(ctx, func(m *types.ModelCard) bool {
		if m.PipelineId == pipelineID {
			out = append(out, m)
		}
		return false
	})
	return out
}

// GetModelCardByDeploymentAddress finds the ModelCard whose deployment_address
// matches the given agent account, if any. Used to correlate a submitter's
// on-chain calibration with its underlying model identity.
func (k Keeper) GetModelCardByDeploymentAddress(ctx context.Context, addr string) (*types.ModelCard, bool) {
	var found *types.ModelCard
	k.IterateModelCards(ctx, func(m *types.ModelCard) bool {
		if m.DeploymentAddress == addr && m.Active {
			found = m
			return true
		}
		return false
	})
	if found == nil {
		return nil, false
	}
	return found, true
}

// EmitModelCardEvent emits a lineage audit event on create / update / retire.
// Each sdk.NewEvent call carries its attributes inline so the event-audit
// tooling can extract them with static regex.
func (k Keeper) EmitModelCardEvent(ctx context.Context, m *types.ModelCard, action string) {
	if m == nil {
		return
	}
	em := sdk.UnwrapSDKContext(ctx).EventManager()
	switch action {
	case "registered":
		em.EmitEvent(sdk.NewEvent("zerone.knowledge.model_card_registered",
			sdk.NewAttribute("model_id", m.Id),
			sdk.NewAttribute("pipeline_id", m.PipelineId),
			sdk.NewAttribute("route", m.Route),
			sdk.NewAttribute("deployment_address", m.DeploymentAddress),
			sdk.NewAttribute("owner_address", m.OwnerAddress),
		))
	case "updated":
		em.EmitEvent(sdk.NewEvent("zerone.knowledge.model_card_updated",
			sdk.NewAttribute("model_id", m.Id),
			sdk.NewAttribute("pipeline_id", m.PipelineId),
			sdk.NewAttribute("route", m.Route),
			sdk.NewAttribute("deployment_address", m.DeploymentAddress),
			sdk.NewAttribute("owner_address", m.OwnerAddress),
		))
	case "retired":
		em.EmitEvent(sdk.NewEvent("zerone.knowledge.model_card_retired",
			sdk.NewAttribute("model_id", m.Id),
			sdk.NewAttribute("pipeline_id", m.PipelineId),
			sdk.NewAttribute("route", m.Route),
			sdk.NewAttribute("deployment_address", m.DeploymentAddress),
			sdk.NewAttribute("owner_address", m.OwnerAddress),
		))
	}
}
