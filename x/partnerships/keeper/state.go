package keeper

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/partnerships/types"
)

// ---------- Partnership CRUD ----------

func (k Keeper) SetPartnership(ctx sdk.Context, p *types.Partnership) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(p)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal partnership: %v", err))
	}
	_ = kvStore.Set(partnershipKey(p.Id), bz)
	_ = kvStore.Set(byHumanKey(p.HumanAddr, p.Id), []byte(p.Id))
	_ = kvStore.Set(byAgentKey(p.AgentAddr, p.Id), []byte(p.Id))
}

func (k Keeper) GetPartnership(ctx sdk.Context, id string) (*types.Partnership, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(partnershipKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var p types.Partnership
	if err := proto.Unmarshal(bz, &p); err != nil {
		return nil, false
	}
	return &p, true
}

func (k Keeper) DeletePartnership(ctx sdk.Context, p *types.Partnership) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(partnershipKey(p.Id))
	_ = kvStore.Delete(byHumanKey(p.HumanAddr, p.Id))
	_ = kvStore.Delete(byAgentKey(p.AgentAddr, p.Id))
}

func (k Keeper) GetAllPartnerships(ctx sdk.Context) []*types.Partnership {
	var partnerships []*types.Partnership
	k.IteratePartnerships(ctx, func(p *types.Partnership) bool {
		partnerships = append(partnerships, p)
		return false
	})
	return partnerships
}

func (k Keeper) IteratePartnerships(ctx sdk.Context, cb func(*types.Partnership) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.PartnershipKeyPrefix, prefixEndBytes(types.PartnershipKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var p types.Partnership
		if err := proto.Unmarshal(iter.Value(), &p); err != nil {
			continue
		}
		if cb(&p) {
			break
		}
	}
}

func (k Keeper) GetPartnershipsByHuman(ctx sdk.Context, humanAddr string) []string {
	return k.getPartnershipIdsByPrefix(ctx, byHumanPrefix(humanAddr))
}

func (k Keeper) GetPartnershipsByAgent(ctx sdk.Context, agentAddr string) []string {
	return k.getPartnershipIdsByPrefix(ctx, byAgentPrefix(agentAddr))
}

func (k Keeper) GetPartnershipsByParticipant(ctx sdk.Context, addr string) []*types.Partnership {
	seen := make(map[string]bool)
	var result []*types.Partnership

	for _, id := range k.GetPartnershipsByHuman(ctx, addr) {
		if !seen[id] {
			seen[id] = true
			if p, found := k.GetPartnership(ctx, id); found {
				result = append(result, p)
			}
		}
	}
	for _, id := range k.GetPartnershipsByAgent(ctx, addr) {
		if !seen[id] {
			seen[id] = true
			if p, found := k.GetPartnership(ctx, id); found {
				result = append(result, p)
			}
		}
	}
	return result
}

func (k Keeper) getPartnershipIdsByPrefix(ctx sdk.Context, prefix []byte) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
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

// ---------- ConsensusOperation CRUD ----------

func (k Keeper) SetConsensusOperation(ctx sdk.Context, op *types.ConsensusOperation) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(op)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal consensus operation: %v", err))
	}
	_ = kvStore.Set(consensusOpKey(op.Id), bz)
}

func (k Keeper) GetConsensusOperation(ctx sdk.Context, id string) (*types.ConsensusOperation, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(consensusOpKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var op types.ConsensusOperation
	if err := proto.Unmarshal(bz, &op); err != nil {
		return nil, false
	}
	return &op, true
}

func (k Keeper) DeleteConsensusOperation(ctx sdk.Context, id string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(consensusOpKey(id))
}

func (k Keeper) GetAllConsensusOperations(ctx sdk.Context) []*types.ConsensusOperation {
	var ops []*types.ConsensusOperation
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ConsensusOpKeyPrefix, prefixEndBytes(types.ConsensusOpKeyPrefix))
	if err != nil {
		return ops
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var op types.ConsensusOperation
		if err := proto.Unmarshal(iter.Value(), &op); err != nil {
			continue
		}
		ops = append(ops, &op)
	}
	return ops
}

// GetPendingOpsForPartnership returns all pending operations for a partnership.
func (k Keeper) GetPendingOpsForPartnership(ctx sdk.Context, partnershipId string) []*types.ConsensusOperation {
	var ops []*types.ConsensusOperation
	for _, op := range k.GetAllConsensusOperations(ctx) {
		if op.PartnershipId == partnershipId && op.Status == types.OpStatusPending {
			ops = append(ops, op)
		}
	}
	return ops
}

// ---------- SafetyFreeze CRUD ----------

func (k Keeper) SetSafetyFreeze(ctx sdk.Context, sf *types.SafetyFreeze) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(sf)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal safety freeze: %v", err))
	}
	_ = kvStore.Set(safetyFreezeKey(sf.PartnershipId), bz)
}

func (k Keeper) GetSafetyFreeze(ctx sdk.Context, partnershipId string) (*types.SafetyFreeze, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(safetyFreezeKey(partnershipId))
	if err != nil || bz == nil {
		return nil, false
	}
	var sf types.SafetyFreeze
	if err := proto.Unmarshal(bz, &sf); err != nil {
		return nil, false
	}
	return &sf, true
}

func (k Keeper) DeleteSafetyFreeze(ctx sdk.Context, partnershipId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(safetyFreezeKey(partnershipId))
}

func (k Keeper) GetAllSafetyFreezes(ctx sdk.Context) []*types.SafetyFreeze {
	var freezes []*types.SafetyFreeze
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.SafetyFreezeKeyPrefix, prefixEndBytes(types.SafetyFreezeKeyPrefix))
	if err != nil {
		return freezes
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var sf types.SafetyFreeze
		if err := proto.Unmarshal(iter.Value(), &sf); err != nil {
			continue
		}
		freezes = append(freezes, &sf)
	}
	return freezes
}

// ---------- CoercionSignal CRUD ----------

func (k Keeper) SetCoercionSignal(ctx sdk.Context, cs *types.CoercionSignal) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(cs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal coercion signal: %v", err))
	}
	_ = kvStore.Set(coercionSignalKey(cs.SignalId), bz)
}

func (k Keeper) GetCoercionSignal(ctx sdk.Context, signalId string) (*types.CoercionSignal, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(coercionSignalKey(signalId))
	if err != nil || bz == nil {
		return nil, false
	}
	var cs types.CoercionSignal
	if err := proto.Unmarshal(bz, &cs); err != nil {
		return nil, false
	}
	return &cs, true
}

func (k Keeper) GetAllCoercionSignals(ctx sdk.Context) []*types.CoercionSignal {
	var signals []*types.CoercionSignal
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.CoercionSignalKeyPrefix, prefixEndBytes(types.CoercionSignalKeyPrefix))
	if err != nil {
		return signals
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var cs types.CoercionSignal
		if err := proto.Unmarshal(iter.Value(), &cs); err != nil {
			continue
		}
		signals = append(signals, &cs)
	}
	return signals
}

// ---------- RejectionCooldown CRUD ----------

func (k Keeper) SetRejectionCooldown(ctx sdk.Context, rc *types.RejectionCooldown) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(rc)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal rejection cooldown: %v", err))
	}
	_ = kvStore.Set(rejectionCooldownKey(rc.PartnershipId), bz)
}

func (k Keeper) GetRejectionCooldown(ctx sdk.Context, partnershipId string) (*types.RejectionCooldown, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(rejectionCooldownKey(partnershipId))
	if err != nil || bz == nil {
		return nil, false
	}
	var rc types.RejectionCooldown
	if err := proto.Unmarshal(bz, &rc); err != nil {
		return nil, false
	}
	return &rc, true
}

func (k Keeper) DeleteRejectionCooldown(ctx sdk.Context, partnershipId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(rejectionCooldownKey(partnershipId))
}

// ---------- Seed Partnership CRUD ----------

func (k Keeper) SetSeedPartnership(ctx sdk.Context, sp *types.SeedPartnership) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(sp)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal seed partnership: %v", err))
	}
	_ = kvStore.Set(seedPartnershipKey(sp.Id), bz)
	_ = kvStore.Set(byDIDSeedKey(sp.HumanAddr, sp.Id), []byte(sp.Id))
	_ = kvStore.Set(byDIDSeedKey(sp.AgentAddr, sp.Id), []byte(sp.Id))
}

func (k Keeper) GetSeedPartnership(ctx sdk.Context, id string) (*types.SeedPartnership, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(seedPartnershipKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var sp types.SeedPartnership
	if err := proto.Unmarshal(bz, &sp); err != nil {
		return nil, false
	}
	return &sp, true
}

func (k Keeper) DeleteSeedPartnership(ctx sdk.Context, sp *types.SeedPartnership) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(seedPartnershipKey(sp.Id))
	_ = kvStore.Delete(byDIDSeedKey(sp.HumanAddr, sp.Id))
	_ = kvStore.Delete(byDIDSeedKey(sp.AgentAddr, sp.Id))
}

func (k Keeper) CountActiveSeedsByDID(ctx sdk.Context, addr string) int {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := byDIDSeedPrefix(addr)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0
	}
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		id := string(iter.Value())
		if sp, found := k.GetSeedPartnership(ctx, id); found && sp.Status == "active" {
			count++
		}
	}
	return count
}

func (k Keeper) GetAllSeedPartnerships(ctx sdk.Context) []*types.SeedPartnership {
	var seeds []*types.SeedPartnership
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.SeedPartnershipKeyPrefix, prefixEndBytes(types.SeedPartnershipKeyPrefix))
	if err != nil {
		return seeds
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var sp types.SeedPartnership
		if err := proto.Unmarshal(iter.Value(), &sp); err != nil {
			continue
		}
		seeds = append(seeds, &sp)
	}
	return seeds
}

// ---------- Pool Entry CRUD ----------

func (k Keeper) SetPoolEntry(ctx sdk.Context, pe *types.PoolEntry) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(pe)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal pool entry: %v", err))
	}
	_ = kvStore.Set(poolEntryKey(pe.Address), bz)
}

func (k Keeper) GetPoolEntry(ctx sdk.Context, addr string) (*types.PoolEntry, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(poolEntryKey(addr))
	if err != nil || bz == nil {
		return nil, false
	}
	var pe types.PoolEntry
	if err := proto.Unmarshal(bz, &pe); err != nil {
		return nil, false
	}
	return &pe, true
}

func (k Keeper) DeletePoolEntry(ctx sdk.Context, addr string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(poolEntryKey(addr))
}

func (k Keeper) CountActivePoolEntries(ctx sdk.Context) int {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.PoolEntryKeyPrefix, prefixEndBytes(types.PoolEntryKeyPrefix))
	if err != nil {
		return 0
	}
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		var pe types.PoolEntry
		if err := proto.Unmarshal(iter.Value(), &pe); err == nil && pe.Status == "active" {
			count++
		}
	}
	return count
}

func (k Keeper) GetAllPoolEntries(ctx sdk.Context) []*types.PoolEntry {
	var entries []*types.PoolEntry
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.PoolEntryKeyPrefix, prefixEndBytes(types.PoolEntryKeyPrefix))
	if err != nil {
		return entries
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var pe types.PoolEntry
		if err := proto.Unmarshal(iter.Value(), &pe); err == nil {
			entries = append(entries, &pe)
		}
	}
	return entries
}

// ---------- Sequence Counter ----------

func (k Keeper) NextSequence(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.SequenceKey)
	if err != nil || bz == nil {
		seq := uint64(1)
		seqBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(seqBytes, seq+1)
		_ = kvStore.Set(types.SequenceKey, seqBytes)
		return seq
	}
	seq := binary.BigEndian.Uint64(bz)
	nextBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBytes, seq+1)
	_ = kvStore.Set(types.SequenceKey, nextBytes)
	return seq
}

// ---------- Key Construction Helpers ----------

func partnershipKey(id string) []byte {
	return append(types.PartnershipKeyPrefix, []byte(id)...)
}

func byHumanPrefix(humanAddr string) []byte {
	return append(types.ByHumanIndexPrefix, []byte(humanAddr+"/")...)
}

func byHumanKey(humanAddr, partnershipId string) []byte {
	return append(types.ByHumanIndexPrefix, []byte(humanAddr+"/"+partnershipId)...)
}

func byAgentPrefix(agentAddr string) []byte {
	return append(types.ByAgentIndexPrefix, []byte(agentAddr+"/")...)
}

func byAgentKey(agentAddr, partnershipId string) []byte {
	return append(types.ByAgentIndexPrefix, []byte(agentAddr+"/"+partnershipId)...)
}

func consensusOpKey(id string) []byte {
	return append(types.ConsensusOpKeyPrefix, []byte(id)...)
}

func safetyFreezeKey(partnershipId string) []byte {
	return append(types.SafetyFreezeKeyPrefix, []byte(partnershipId)...)
}

func coercionSignalKey(signalId string) []byte {
	return append(types.CoercionSignalKeyPrefix, []byte(signalId)...)
}

func rejectionCooldownKey(partnershipId string) []byte {
	return append(types.RejectionCooldownKeyPrefix, []byte(partnershipId)...)
}

func seedPartnershipKey(id string) []byte {
	return append(types.SeedPartnershipKeyPrefix, []byte(id)...)
}

func byDIDSeedPrefix(addr string) []byte {
	return append(types.ByDIDSeedIndexPrefix, []byte(addr+"/")...)
}

func byDIDSeedKey(addr, seedId string) []byte {
	return append(types.ByDIDSeedIndexPrefix, []byte(addr+"/"+seedId)...)
}

func poolEntryKey(addr string) []byte {
	return append(types.PoolEntryKeyPrefix, []byte(addr)...)
}
