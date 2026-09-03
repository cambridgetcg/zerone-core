package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// terminalIteratorError follows the db.Iterator contract while tolerating the
// Cosmos SDK cache iterator's documented implementation defect: cacheMerge and
// mem iterators report their normal exhausted state as an "invalid ...Iterator"
// error instead of exposing a nil terminal error. Other errors that reach this
// wrapper fail closed, and Close errors are checked separately. The SDK cache
// iterator can also hide a parent traversal error entirely; the coordinated
// activation therefore does not use this helper as a completeness proof. It
// reconstructs the complete committed IAVL root before supplying its snapshot.
func terminalIteratorError(iter interface {
	Valid() bool
	Error() error
}) error {
	err := iter.Error()
	if err == nil {
		return nil
	}
	if !iter.Valid() {
		switch err.Error() {
		case "invalid cacheMergeIterator", "invalid memIterator":
			return nil
		}
	}
	return err
}

// --- Emergency Status ---

// GetEmergencyStatus returns the current emergency status.
func (k Keeper) GetEmergencyStatus(ctx context.Context) types.EmergencyStatus {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.HaltStatusKey)
	if err != nil {
		panic("failed to read emergency status: " + err.Error())
	}
	if bz == nil {
		return types.StatusNormal
	}
	status := types.EmergencyStatus(bz)
	switch status {
	case types.StatusNormal, types.StatusHaltVoting, types.StatusHalted,
		types.StatusRevertVoting, types.StatusReverting, types.StatusResumeVoting:
		return status
	default:
		panic(fmt.Sprintf("invalid persisted emergency status %q", status))
	}
}

// SetEmergencyStatus sets the current emergency status.
func (k Keeper) SetEmergencyStatus(ctx context.Context, status types.EmergencyStatus) {
	switch status {
	case types.StatusNormal, types.StatusHaltVoting, types.StatusHalted,
		types.StatusRevertVoting, types.StatusReverting, types.StatusResumeVoting:
	default:
		panic(fmt.Sprintf("refusing to persist invalid emergency status %q", status))
	}
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.HaltStatusKey, []byte(status)); err != nil {
		panic("failed to set emergency status: " + err.Error())
	}
}

// IsHalted returns true if application transaction admission is quarantined.
// It does not mean CometBFT consensus or block production has stopped.
func (k Keeper) IsHalted(ctx context.Context) bool {
	status := k.GetEmergencyStatus(ctx)
	switch status {
	case types.StatusHalted, types.StatusRevertVoting, types.StatusReverting, types.StatusResumeVoting:
		return true
	default:
		releaseBlock := k.GetQuarantineReleaseBlock(ctx)
		if releaseBlock == 0 {
			return false
		}
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		if sdkCtx.BlockHeight() < 0 {
			return false
		}
		currentHeight := uint64(sdkCtx.BlockHeight())
		if sdkCtx.IsCheckTx() || sdkCtx.IsReCheckTx() {
			// BaseApp CheckTx after commit H still carries header height H,
			// while accepted transactions target proposal height H+1. Treat
			// that next candidate height as authoritative so a latch through
			// H does not accidentally delay honest mempool admission to H+2.
			return currentHeight < releaseBlock
		}
		return currentHeight <= releaseBlock
	}
}

// GetQuarantineReleaseBlock returns the block through which transaction
// quarantine remains latched after an affirmative resume finalizes.
func (k Keeper) GetQuarantineReleaseBlock(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.QuarantineReleaseBlockKey)
	if err != nil {
		panic("failed to read quarantine release block: " + err.Error())
	}
	if bz == nil {
		return 0
	}
	if len(bz) != 8 {
		panic(fmt.Sprintf(
			"invalid persisted quarantine release block length %d",
			len(bz),
		))
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetQuarantineReleaseBlock(
	ctx context.Context,
	height uint64,
) {
	if height == 0 {
		panic("refusing to persist zero quarantine release block")
	}
	if height > types.MaxSDKBlockHeight-types.PostResumeCancellationGraceBlocks {
		panic("refusing quarantine release block without a representable post-resume grace window")
	}
	var bz [8]byte
	binary.BigEndian.PutUint64(bz[:], height)
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.QuarantineReleaseBlockKey, bz[:]); err != nil {
		panic("failed to set quarantine release block: " + err.Error())
	}
}

func (k Keeper) ClearQuarantineReleaseBlock(ctx context.Context) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.QuarantineReleaseBlockKey); err != nil {
		panic("failed to clear quarantine release block: " + err.Error())
	}
}

// --- Ceremonies ---

// SetCeremony stores a ceremony by ID.
func (k Keeper) SetCeremony(ctx context.Context, ceremony *types.EmergencyCeremony) error {
	if ceremony == nil || ceremony.Id == "" {
		return fmt.Errorf("cannot persist nil or empty-id emergency ceremony")
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(ceremony)
	if err != nil {
		return fmt.Errorf("failed to marshal emergency ceremony: %w", err)
	}
	if err := store.Set(types.CeremonyKey(ceremony.Id), bz); err != nil {
		return fmt.Errorf("failed to set emergency ceremony: %w", err)
	}
	activeID := k.getActiveCeremonyID(ctx)
	if isNonterminalEmergencyCeremony(ceremony) {
		// Legacy/corrupt test state may contain several nonterminal records.
		// Preserve the first authenticated index entry; the named activation
		// performs the complete census and deterministically reconciles it.
		if activeID == "" {
			k.setActiveCeremonyID(ctx, ceremony.Id)
		}
	} else if activeID == ceremony.Id {
		k.setActiveCeremonyID(ctx, "")
	}
	return nil
}

// GetCeremony retrieves a ceremony by ID.
func (k Keeper) GetCeremony(ctx context.Context, id string) (*types.EmergencyCeremony, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.CeremonyKey(id))
	if err != nil {
		panic("failed to read emergency ceremony: " + err.Error())
	}
	if bz == nil {
		return nil, false
	}
	var ceremony types.EmergencyCeremony
	if err := proto.Unmarshal(bz, &ceremony); err != nil {
		panic("failed to decode emergency ceremony: " + err.Error())
	}
	return &ceremony, true
}

// GetActiveCeremony returns the indexed non-terminal ceremony in O(1).
func (k Keeper) GetActiveCeremony(ctx context.Context) (*types.EmergencyCeremony, bool) {
	activeID := k.getActiveCeremonyID(ctx)
	if activeID == "" {
		return nil, false
	}
	active, found := k.GetCeremony(ctx, activeID)
	if !found {
		panic(fmt.Sprintf(
			"active emergency ceremony index references missing ceremony %q",
			activeID,
		))
	}
	if !isNonterminalEmergencyCeremony(active) {
		panic(fmt.Sprintf(
			"active emergency ceremony index references terminal ceremony %q",
			activeID,
		))
	}
	return active, true
}

func (k Keeper) getActiveCeremonyID(ctx context.Context) string {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ActiveCeremonyIdKey)
	if err != nil {
		panic("failed to read active emergency ceremony id: " + err.Error())
	}
	return string(bz)
}

func (k Keeper) setActiveCeremonyID(ctx context.Context, id string) {
	store := k.storeService.OpenKVStore(ctx)
	if id == "" {
		if err := store.Delete(types.ActiveCeremonyIdKey); err != nil {
			panic("failed to clear active emergency ceremony id: " + err.Error())
		}
		return
	}
	if err := store.Set(types.ActiveCeremonyIdKey, []byte(id)); err != nil {
		panic("failed to persist active emergency ceremony id: " + err.Error())
	}
}

// GetAllCeremonies returns all stored ceremonies.
func (k Keeper) GetAllCeremonies(ctx context.Context) []*types.EmergencyCeremony {
	var ceremonies []*types.EmergencyCeremony
	k.IterateCeremonies(ctx, func(c *types.EmergencyCeremony) bool {
		ceremonies = append(ceremonies, c)
		return false
	})
	return ceremonies
}

// IterateCeremonies iterates over all ceremonies. Return true from cb to stop.
func (k Keeper) IterateCeremonies(ctx context.Context, cb func(*types.EmergencyCeremony) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.CeremonyKeyPrefix, prefixEndBytes(types.CeremonyKeyPrefix))
	if err != nil {
		panic("failed to iterate emergency ceremonies: " + err.Error())
	}
	defer func() {
		if err := iter.Close(); err != nil {
			panic("failed to close emergency ceremony iterator: " + err.Error())
		}
	}()

	for ; iter.Valid(); iter.Next() {
		var ceremony types.EmergencyCeremony
		if err := proto.Unmarshal(iter.Value(), &ceremony); err != nil {
			panic("failed to decode emergency ceremony during iteration: " + err.Error())
		}
		if cb(&ceremony) {
			break
		}
	}
	if err := terminalIteratorError(iter); err != nil {
		panic("failed while iterating emergency ceremonies: " + err.Error())
	}
}

// --- Active Halt Ceremony ID ---

// GetActiveHaltCeremonyId returns the ID of the current active halt ceremony.
func (k Keeper) GetActiveHaltCeremonyId(ctx context.Context) string {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ActiveHaltCeremonyIdKey)
	if err != nil {
		panic("failed to read active quarantine ceremony id: " + err.Error())
	}
	if bz == nil {
		return ""
	}
	return string(bz)
}

// SetActiveHaltCeremonyId stores the active halt ceremony ID.
func (k Keeper) SetActiveHaltCeremonyId(ctx context.Context, id string) {
	store := k.storeService.OpenKVStore(ctx)
	var err error
	if id == "" {
		err = store.Delete(types.ActiveHaltCeremonyIdKey)
	} else {
		err = store.Set(types.ActiveHaltCeremonyIdKey, []byte(id))
	}
	if err != nil {
		panic("failed to persist active quarantine ceremony id: " + err.Error())
	}
}

// --- Halt Start Block ---

// GetHaltStartBlock returns the block at which the current halt began.
func (k Keeper) GetHaltStartBlock(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.HaltStartBlockKey)
	if err != nil {
		panic("failed to read quarantine start block: " + err.Error())
	}
	if bz == nil {
		return 0
	}
	if len(bz) != 8 {
		panic(fmt.Sprintf("invalid persisted quarantine start block length %d", len(bz)))
	}
	return binary.BigEndian.Uint64(bz)
}

// SetHaltStartBlock records the block at which the halt began.
func (k Keeper) SetHaltStartBlock(ctx context.Context, block uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, block)
	if err := store.Set(types.HaltStartBlockKey, bz); err != nil {
		panic("failed to persist quarantine start block: " + err.Error())
	}
	k.ClearLastHaltEscalationBlock(ctx)
}

// ClearHaltStartBlock removes the halt start block (called on resume).
func (k Keeper) ClearHaltStartBlock(ctx context.Context) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.HaltStartBlockKey); err != nil {
		panic("failed to clear quarantine start block: " + err.Error())
	}
	k.ClearLastHaltEscalationBlock(ctx)
}

// GetLastHaltEscalationBlock returns the latest quarantine deadline boundary
// already reported to operators.
func (k Keeper) GetLastHaltEscalationBlock(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.LastHaltEscalationBlockKey)
	if err != nil {
		panic("failed to read last quarantine escalation block: " + err.Error())
	}
	if bz == nil {
		return 0
	}
	if len(bz) != 8 {
		panic(fmt.Sprintf("invalid last quarantine escalation block length %d", len(bz)))
	}
	return binary.BigEndian.Uint64(bz)
}

// SetLastHaltEscalationBlock records a quarantine deadline boundary after its
// diagnostic has been emitted.
func (k Keeper) SetLastHaltEscalationBlock(ctx context.Context, block uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, block)
	if err := store.Set(types.LastHaltEscalationBlockKey, bz); err != nil {
		panic("failed to persist last quarantine escalation block: " + err.Error())
	}
}

// ClearLastHaltEscalationBlock clears escalation state for a closed or newly
// opened quarantine.
func (k Keeper) ClearLastHaltEscalationBlock(ctx context.Context) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.LastHaltEscalationBlockKey); err != nil {
		panic("failed to clear last quarantine escalation block: " + err.Error())
	}
}

// --- Audit Log ---

// AddAuditEntry appends an audit entry to the log.
func (k Keeper) AddAuditEntry(ctx context.Context, entry *types.EmergencyAuditEntry) {
	if entry == nil {
		panic("cannot persist a nil emergency audit entry")
	}
	store := k.storeService.OpenKVStore(ctx)
	height := entry.BlockNumber
	var index uint32
	for {
		key := types.AuditLogKey(height, index)
		bz, err := store.Get(key)
		if err != nil {
			panic("failed to inspect emergency audit log: " + err.Error())
		}
		if bz == nil {
			data, err := proto.Marshal(entry)
			if err != nil {
				panic("failed to marshal emergency audit entry: " + err.Error())
			}
			if err := store.Set(key, data); err != nil {
				panic("failed to persist emergency audit entry: " + err.Error())
			}
			return
		}
		if index == ^uint32(0) {
			panic(fmt.Sprintf("emergency audit index exhausted at block %d", height))
		}
		index++
	}
}

// GetAuditLog returns all audit entries.
func (k Keeper) GetAuditLog(ctx context.Context) []*types.EmergencyAuditEntry {
	var entries []*types.EmergencyAuditEntry
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.AuditLogKeyPrefix, prefixEndBytes(types.AuditLogKeyPrefix))
	if err != nil {
		panic("failed to iterate emergency audit log: " + err.Error())
	}
	defer func() {
		if err := iter.Close(); err != nil {
			panic("failed to close emergency audit iterator: " + err.Error())
		}
	}()

	for ; iter.Valid(); iter.Next() {
		var entry types.EmergencyAuditEntry
		if err := proto.Unmarshal(iter.Value(), &entry); err != nil {
			panic("failed to decode emergency audit entry: " + err.Error())
		}
		entries = append(entries, &entry)
	}
	if err := terminalIteratorError(iter); err != nil {
		panic("failed while iterating emergency audit log: " + err.Error())
	}
	return entries
}

// --- Anti-Abuse Tracking ---

// GetGuardianProposalCount returns how many proposals a guardian has made this epoch.
func (k Keeper) GetGuardianProposalCount(ctx context.Context, addr string) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.GuardianProposalCountKey(addr))
	if err != nil {
		panic("failed to read guardian proposal count: " + err.Error())
	}
	if bz == nil {
		return 0
	}
	if len(bz) != 8 {
		panic(fmt.Sprintf("invalid guardian proposal count length %d", len(bz)))
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementGuardianProposalCount increments a guardian's proposal count.
func (k Keeper) IncrementGuardianProposalCount(ctx context.Context, addr string) {
	count := k.GetGuardianProposalCount(ctx, addr) + 1
	if count == 0 {
		panic("guardian proposal count overflow")
	}
	k.SetGuardianProposalCount(ctx, addr, count)
}

// SetGuardianProposalCount restores one validated per-epoch counter.
func (k Keeper) SetGuardianProposalCount(ctx context.Context, addr string, count uint64) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GuardianProposalCountKey(addr)
	if count == 0 {
		if err := store.Delete(key); err != nil {
			panic("failed to clear guardian proposal count: " + err.Error())
		}
		return
	}
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	if err := store.Set(key, bz); err != nil {
		panic("failed to persist guardian proposal count: " + err.Error())
	}
}

// GetAllGuardianProposalCounts returns canonical address-sorted anti-abuse
// counters for genesis export.
func (k Keeper) GetAllGuardianProposalCounts(ctx context.Context) []*types.GuardianProposalCount {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.GuardianProposalCountPrefix, prefixEndBytes(types.GuardianProposalCountPrefix))
	if err != nil {
		panic("failed to iterate guardian proposal counts: " + err.Error())
	}
	defer func() {
		if err := iter.Close(); err != nil {
			panic("failed to close guardian proposal count iterator: " + err.Error())
		}
	}()

	var counts []*types.GuardianProposalCount
	for ; iter.Valid(); iter.Next() {
		value := iter.Value()
		if len(value) != 8 {
			panic(fmt.Sprintf("invalid guardian proposal count length %d", len(value)))
		}
		counts = append(counts, &types.GuardianProposalCount{
			Guardian: string(iter.Key()[len(types.GuardianProposalCountPrefix):]),
			Count:    binary.BigEndian.Uint64(value),
		})
	}
	if err := terminalIteratorError(iter); err != nil {
		panic("failed while iterating guardian proposal counts: " + err.Error())
	}
	return counts
}

// GetEpochProposalCount returns the global proposal count for this epoch.
func (k Keeper) GetEpochProposalCount(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.EpochProposalCountKey)
	if err != nil {
		panic("failed to read epoch proposal count: " + err.Error())
	}
	if bz == nil {
		return 0
	}
	if len(bz) != 8 {
		panic(fmt.Sprintf("invalid epoch proposal count length %d", len(bz)))
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementEpochProposalCount increments the global epoch proposal count.
func (k Keeper) IncrementEpochProposalCount(ctx context.Context) {
	count := k.GetEpochProposalCount(ctx) + 1
	if count == 0 {
		panic("epoch proposal count overflow")
	}
	k.SetEpochProposalCount(ctx, count)
}

// SetEpochProposalCount restores the validated per-epoch global counter.
func (k Keeper) SetEpochProposalCount(ctx context.Context, count uint64) {
	store := k.storeService.OpenKVStore(ctx)
	if count == 0 {
		if err := store.Delete(types.EpochProposalCountKey); err != nil {
			panic("failed to clear epoch proposal count: " + err.Error())
		}
		return
	}
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	if err := store.Set(types.EpochProposalCountKey, bz); err != nil {
		panic("failed to persist epoch proposal count: " + err.Error())
	}
}

// GetLastProposalBlock returns the block height of the last proposal.
func (k Keeper) GetLastProposalBlock(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.LastProposalBlockKey)
	if err != nil {
		panic("failed to read last proposal block: " + err.Error())
	}
	if bz == nil {
		return 0
	}
	if len(bz) != 8 {
		panic(fmt.Sprintf("invalid last proposal block length %d", len(bz)))
	}
	return binary.BigEndian.Uint64(bz)
}

// SetLastProposalBlock stores the block height of the last proposal.
func (k Keeper) SetLastProposalBlock(ctx context.Context, block uint64) {
	store := k.storeService.OpenKVStore(ctx)
	if block == 0 {
		if err := store.Delete(types.LastProposalBlockKey); err != nil {
			panic("failed to clear last proposal block: " + err.Error())
		}
		return
	}
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, block)
	if err := store.Set(types.LastProposalBlockKey, bz); err != nil {
		panic("failed to persist last proposal block: " + err.Error())
	}
}

// ResetEpochCounters clears all per-epoch anti-abuse tracking.
func (k Keeper) ResetEpochCounters(ctx context.Context) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.EpochProposalCountKey); err != nil {
		panic("failed to clear epoch proposal count: " + err.Error())
	}
	iter, err := store.Iterator(types.GuardianProposalCountPrefix, prefixEndBytes(types.GuardianProposalCountPrefix))
	if err != nil {
		panic("failed to iterate guardian proposal counts: " + err.Error())
	}
	var keys [][]byte
	for ; iter.Valid(); iter.Next() {
		key := append([]byte(nil), iter.Key()...)
		keys = append(keys, key)
	}
	if err := terminalIteratorError(iter); err != nil {
		_ = iter.Close()
		panic("failed while iterating guardian proposal counts: " + err.Error())
	}
	if err := iter.Close(); err != nil {
		panic("failed to close guardian proposal count iterator: " + err.Error())
	}
	for _, key := range keys {
		if err := store.Delete(key); err != nil {
			panic("failed to clear guardian proposal count: " + err.Error())
		}
	}
}

// --- Revert Target ---

// RevertTarget holds the guardian-agreed state rollback target.
type RevertTarget struct {
	Height     uint64 `json:"height"`
	BlockHash  string `json:"block_hash"`
	CeremonyId string `json:"ceremony_id"`
}

// SetRevertTarget stores the revert target agreed upon by guardians.
func (k Keeper) SetRevertTarget(ctx context.Context, height uint64, blockHash string, ceremonyId string) {
	store := k.storeService.OpenKVStore(ctx)
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, height)
	if err := store.Set(types.RevertTargetHeightKey, heightBz); err != nil {
		panic("failed to persist legacy revert height: " + err.Error())
	}
	if err := store.Set(types.RevertTargetHashKey, []byte(blockHash)); err != nil {
		panic("failed to persist legacy revert hash: " + err.Error())
	}
	if err := store.Set(types.RevertCeremonyIdKey, []byte(ceremonyId)); err != nil {
		panic("failed to persist legacy revert ceremony id: " + err.Error())
	}
}

// GetRevertTarget returns the current revert target, if set.
func (k Keeper) GetRevertTarget(ctx context.Context) (RevertTarget, bool) {
	store := k.storeService.OpenKVStore(ctx)
	heightBz, err := store.Get(types.RevertTargetHeightKey)
	if err != nil {
		panic("failed to read legacy revert height: " + err.Error())
	}
	if heightBz == nil {
		return RevertTarget{}, false
	}
	if len(heightBz) != 8 {
		panic(fmt.Sprintf("invalid persisted legacy revert height length %d", len(heightBz)))
	}
	height := binary.BigEndian.Uint64(heightBz)
	hashBz, err := store.Get(types.RevertTargetHashKey)
	if err != nil {
		panic("failed to read legacy revert hash: " + err.Error())
	}
	ceremonyBz, err := store.Get(types.RevertCeremonyIdKey)
	if err != nil {
		panic("failed to read legacy revert ceremony id: " + err.Error())
	}
	return RevertTarget{
		Height:     height,
		BlockHash:  string(hashBz),
		CeremonyId: string(ceremonyBz),
	}, true
}

// ClearRevertTarget removes the revert target (called on resume after rollback).
func (k Keeper) ClearRevertTarget(ctx context.Context) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.RevertTargetHeightKey); err != nil {
		panic("failed to clear legacy revert height: " + err.Error())
	}
	if err := store.Delete(types.RevertTargetHashKey); err != nil {
		panic("failed to clear legacy revert hash: " + err.Error())
	}
	if err := store.Delete(types.RevertCeremonyIdKey); err != nil {
		panic("failed to clear legacy revert ceremony id: " + err.Error())
	}
}

// --- Params ---

// GetParams returns the emergency module parameters.
func (k Keeper) GetParams(ctx context.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil {
		panic("failed to read emergency params: " + err.Error())
	}
	if bz == nil {
		p := types.DefaultParams()
		return &p
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		panic("failed to decode emergency params: " + err.Error())
	}
	if err := params.Validate(); err != nil {
		normalized := types.NormalizeLegacyParams(&params)
		if normalizedErr := normalized.Validate(); normalizedErr != nil {
			panic("invalid persisted emergency params: " + normalizedErr.Error())
		}
		return normalized
	}
	return &params
}

// SetParams stores the emergency module parameters.
func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	if params == nil {
		panic("cannot persist nil emergency params")
	}
	if err := params.Validate(); err != nil {
		panic("cannot persist invalid emergency params: " + err.Error())
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic("failed to marshal emergency params: " + err.Error())
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic("failed to set emergency params: " + err.Error())
	}
}

// sdkContext is a helper to get sdk.Context from context.Context for event emission.
func sdkContext(ctx context.Context) sdk.Context {
	return sdk.UnwrapSDKContext(ctx)
}
