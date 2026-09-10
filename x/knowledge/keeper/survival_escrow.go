package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── Survival-gate escrow ─────────────────────────────────────────────────────
//
// Acceptance records a pending nominal reward. A surviving challenge or closed
// challenge window hands it to vesting; schedule creation is not minting or a
// payment. Pending work is removed only after the schedule is committed. This
// settlement rule does not establish the external truth of the underlying fact.

// SurvivalPendingReward is the submitter reward held until the fact survives.
// Stored as JSON under SurvivalPendingRewardPrefix (mirrors the vindication pattern).
type SurvivalPendingReward struct {
	ClaimId       string `json:"claim_id"`
	FactId        string `json:"fact_id"`
	Recipient     string `json:"recipient"`
	Amount        string `json:"amount"` // uzrn, string for big.Int compat
	Category      string `json:"category"`
	PartnershipId string `json:"partnership_id"` // preserved historical metadata; no partnership routing
	Deadline      uint64 `json:"deadline"`       // block height the challenge window closes
}

func survivalPendingKey(factId string) []byte {
	return append(append([]byte{}, types.SurvivalPendingRewardPrefix...), []byte(factId)...)
}

func survivalDeadlineKey(deadline uint64, factId string) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, deadline)
	key := append(append([]byte{}, types.SurvivalDeadlineIndexPrefix...), b...)
	return append(key, []byte(factId)...)
}

func validateSurvivalPendingReward(pr SurvivalPendingReward) error {
	return types.ValidateSurvivalPendingReward(&types.SurvivalPendingReward{
		ClaimId: pr.ClaimId, FactId: pr.FactId, Recipient: pr.Recipient,
		Amount: pr.Amount, Category: pr.Category, PartnershipId: pr.PartnershipId,
		Deadline: pr.Deadline,
	})
}

func decodeSurvivalPendingReward(bz []byte, factID string) (SurvivalPendingReward, error) {
	var pr SurvivalPendingReward
	if !utf8.Valid(bz) {
		return pr, fmt.Errorf("pending survival reward is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(bz))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return pr, fmt.Errorf("pending survival reward must be an object")
	}
	seen := make(map[string]bool)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return pr, err
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return pr, fmt.Errorf("pending survival reward has duplicate or invalid field")
		}
		seen[key] = true
		var target any
		switch key {
		case "claim_id":
			target = &pr.ClaimId
		case "fact_id":
			target = &pr.FactId
		case "recipient":
			target = &pr.Recipient
		case "amount":
			target = &pr.Amount
		case "category":
			target = &pr.Category
		case "partnership_id":
			target = &pr.PartnershipId
		case "deadline":
			target = &pr.Deadline
		default:
			return pr, fmt.Errorf("pending survival reward has unknown field %q", key)
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return pr, err
		}
		if bytes.Equal(raw, []byte("null")) {
			return pr, fmt.Errorf("pending survival reward has null field %q", key)
		}
		if !survivalJSONUnicodeValid(raw) {
			return pr, fmt.Errorf("pending survival reward has invalid Unicode escape")
		}
		if err := json.Unmarshal(raw, target); err != nil {
			return pr, err
		}
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') {
		return pr, fmt.Errorf("pending survival reward has invalid object ending")
	}
	if !seen["deadline"] {
		return pr, fmt.Errorf("pending survival reward has no deadline")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return pr, fmt.Errorf("pending survival reward has trailing data")
	}
	if pr.FactId != factID {
		return pr, fmt.Errorf("pending survival reward key does not match fact")
	}
	return pr, validateSurvivalPendingReward(pr)
}

// encoding/json replaces unpaired UTF-16 surrogates with U+FFFD. Refuse that
// lossy conversion in stored claimant fields; valid surrogate pairs remain
// valid JSON. The decoder has already checked the raw JSON escape syntax.
func survivalJSONUnicodeValid(raw []byte) bool {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) || raw[i] != 'u' {
			continue
		}
		value, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if value >= 0xDC00 && value <= 0xDFFF {
			return false
		}
		if value >= 0xD800 && value <= 0xDBFF {
			if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
			if err != nil || low < 0xDC00 || low > 0xDFFF {
				return false
			}
			i += 6
		}
	}
	return true
}

// SetSurvivalPendingReward atomically stores a claim and its derived deadline
// index. Repeating the same record is safe; a different obligation cannot
// overwrite pending work or reset its deadline.
func (k Keeper) SetSurvivalPendingReward(ctx context.Context, pr SurvivalPendingReward) error {
	if err := validateSurvivalPendingReward(pr); err != nil {
		return err
	}
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	prior, found, err := k.getSurvivalPendingReward(cached, pr.FactId)
	if err != nil {
		return err
	}
	if found && prior != pr {
		return fmt.Errorf("conflicting pending survival reward for fact %s", pr.FactId)
	}
	store := k.storeService.OpenKVStore(cached)
	if !found {
		bz, err := json.Marshal(pr)
		if err != nil {
			return err
		}
		if err := store.Set(survivalPendingKey(pr.FactId), bz); err != nil {
			return err
		}
	}
	if err := store.Set(survivalDeadlineKey(pr.Deadline, pr.FactId), []byte{0x01}); err != nil {
		return err
	}
	write()
	return nil
}

// GetSurvivalPendingReward returns the pending reward for a fact, if any.
func (k Keeper) GetSurvivalPendingReward(ctx context.Context, factId string) (SurvivalPendingReward, bool) {
	pr, found, err := k.getSurvivalPendingReward(ctx, factId)
	return pr, found && err == nil
}

func (k Keeper) getSurvivalPendingReward(ctx context.Context, factId string) (SurvivalPendingReward, bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(survivalPendingKey(factId))
	if err != nil || bz == nil {
		return SurvivalPendingReward{}, false, err
	}
	pr, err := decodeSurvivalPendingReward(bz, factId)
	return pr, err == nil, err
}

// GetAllSurvivalPendingRewards strictly reads the authoritative pending records
// for export and the one-time upgrade. Corrupt or unreadable claims are errors,
// never an empty inventory.
func (k Keeper) GetAllSurvivalPendingRewards(ctx context.Context) ([]SurvivalPendingReward, error) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.SurvivalPendingRewardPrefix, prefixEndBytes(types.SurvivalPendingRewardPrefix))
	if err != nil {
		return nil, err
	}
	var pending []SurvivalPendingReward
	var size int
	for ; iter.Valid(); iter.Next() {
		size += len(iter.Key()) + len(iter.Value())
		if len(pending) >= 100_000 || size > 64<<20 {
			_ = iter.Close()
			return nil, fmt.Errorf("pending survival rewards exceed inventory bounds")
		}
		pr, err := decodeSurvivalPendingReward(iter.Value(), string(iter.Key()[len(types.SurvivalPendingRewardPrefix):]))
		if err != nil {
			_ = iter.Close()
			return nil, err
		}
		pending = append(pending, pr)
	}
	if err := closeSurvivalIterator(iter); err != nil {
		return nil, err
	}
	return pending, nil
}

// SDK cache iterators report these exact errors at ordinary exhaustion. Keep
// all other iterator errors and Close failures visible, including empty scans.
func closeSurvivalIterator(iter corestore.Iterator) error {
	err := iter.Error()
	if err != nil && !iter.Valid() && (err.Error() == "invalid cacheMergeIterator" || err.Error() == "invalid memIterator") {
		err = nil
	}
	closeErr := iter.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// RebuildSurvivalDeadlineIndex repairs the derived index once at the named
// upgrade. Older challenge sweeps removed entries for claims still pending.
// No pending record is rewritten or repriced.
func (k Keeper) RebuildSurvivalDeadlineIndex(ctx context.Context) error {
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	pending, err := k.GetAllSurvivalPendingRewards(cached)
	if err != nil {
		return err
	}
	store := k.storeService.OpenKVStore(cached)
	iter, err := store.Iterator(types.SurvivalDeadlineIndexPrefix, prefixEndBytes(types.SurvivalDeadlineIndexPrefix))
	if err != nil {
		return err
	}
	var keys [][]byte
	var size int
	for ; iter.Valid(); iter.Next() {
		size += len(iter.Key()) + len(iter.Value())
		if len(keys) >= 100_000 || size > 64<<20 {
			_ = iter.Close()
			return fmt.Errorf("survival deadline index exceeds upgrade scan bounds")
		}
		keys = append(keys, append([]byte(nil), iter.Key()...))
	}
	if err := closeSurvivalIterator(iter); err != nil {
		return err
	}
	for _, key := range keys {
		if err := store.Delete(key); err != nil {
			return err
		}
	}
	for _, pr := range pending {
		if err := store.Set(survivalDeadlineKey(pr.Deadline, pr.FactId), []byte{1}); err != nil {
			return err
		}
	}
	write()
	return nil
}

func (k Keeper) deleteSurvivalPending(ctx context.Context, pr SurvivalPendingReward) error {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(survivalPendingKey(pr.FactId)); err != nil {
		return err
	}
	return store.Delete(survivalDeadlineKey(pr.Deadline, pr.FactId))
}

// EscrowSubmitterReward records the submitter reward as pending (nothing minted)
// and stamps the fact's challenge window. Replaces the accept-time reward routing.
func (k Keeper) EscrowSubmitterReward(ctx context.Context, fact *types.Fact, claim *types.Claim) {
	if k.vestingRewardsKeeper == nil {
		return
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return
	}
	window := params.ChallengeDurationBlocks
	if window == 0 {
		window = 34_272 // ~1 day at 2.521s block time — defensive default
	}
	deadline := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()) + window
	if err := k.SetSurvivalPendingReward(ctx, SurvivalPendingReward{
		ClaimId:       claim.Id,
		FactId:        fact.Id,
		Recipient:     claim.Submitter,
		Amount:        claim.Stake,
		Category:      claim.Category,
		PartnershipId: claim.PartnershipId,
		Deadline:      deadline,
	}); err != nil {
		k.Logger(ctx).Error("record pending survival reward", "fact_id", fact.Id, "error", err)
		return
	}
	fact.ChallengeWindowEnd = deadline
	_ = k.SetFact(ctx, fact)
}

// releaseSurvivalReward hands a surviving claim to vesting atomically. Failure
// leaves both modules' state and events unchanged; absence is a safe no-op.
func (k Keeper) releaseSurvivalReward(ctx context.Context, factId string) error {
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	pr, found, err := k.getSurvivalPendingReward(cached, factId)
	if err != nil || !found {
		return err
	}
	if err := k.routeSubmitterReward(cached, pr); err != nil {
		return err
	}
	if err := k.deleteSurvivalPending(cached, pr); err != nil {
		return err
	}
	cached.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.survival_reward_released",
		sdk.NewAttribute("fact_id", factId),
		sdk.NewAttribute("recipient", pr.Recipient),
		sdk.NewAttribute("amount", pr.Amount),
	))
	write()
	return nil
}

// cancelSurvivalReward drops a pending reward without issuing it — the fact fell to
// DISPROVEN or decayed before surviving. The clawback is free: nothing was minted.
func (k Keeper) cancelSurvivalReward(ctx context.Context, factId string) {
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	pr, found, err := k.getSurvivalPendingReward(cached, factId)
	if err != nil {
		k.Logger(ctx).Error("read pending survival reward for cancellation", "fact_id", factId, "error", err)
		return
	}
	if !found {
		return
	}
	if err := k.deleteSurvivalPending(cached, pr); err != nil {
		k.Logger(ctx).Error("cancel pending survival reward", "fact_id", factId, "error", err)
		return
	}
	cached.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.survival_reward_cancelled",
		sdk.NewAttribute("fact_id", factId),
		sdk.NewAttribute("recipient", pr.Recipient),
	))
	write()
}

// routeSubmitterReward creates the schedule under the existing economic rules.
// The historical partnership field does not select an alternate payment path.
func (k Keeper) routeSubmitterReward(ctx context.Context, pr SurvivalPendingReward) error {
	if k.vestingRewardsKeeper == nil {
		return fmt.Errorf("vesting rewards keeper is unavailable")
	}
	return k.vestingRewardsKeeper.CreateVestingScheduleFromKnowledge(ctx, pr.ClaimId, pr.FactId, pr.Recipient, pr.Amount, pr.Category)
}

// SweepSurvivedRewards hands pending rewards to vesting for facts whose challenge
// window closed while VERIFIED or restored ACTIVE. Called from BeginBlocker.
// The pending entry is the source of truth for unfinished handoffs; the deadline
// index is an ordered scan hint, so only due entries (deadline <= height) are read.
func (k Keeper) SweepSurvivedRewards(ctx context.Context) {
	height := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	store := k.storeService.OpenKVStore(ctx)

	end := make([]byte, 8)
	binary.BigEndian.PutUint64(end, height+1) // exclusive: captures deadline <= height
	upper := append(append([]byte{}, types.SurvivalDeadlineIndexPrefix...), end...)
	iter, err := store.Iterator(types.SurvivalDeadlineIndexPrefix, upper)
	if err != nil {
		return
	}
	// Collect due factIDs first; do not mutate the store during iteration.
	var due []string
	prefixLen := len(types.SurvivalDeadlineIndexPrefix) + 8
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) <= prefixLen {
			continue
		}
		due = append(due, string(key[prefixLen:]))
	}
	if err := closeSurvivalIterator(iter); err != nil {
		return
	}

	for _, factId := range due {
		pr, found, err := k.getSurvivalPendingReward(ctx, factId)
		if err != nil {
			k.Logger(ctx).Error("read pending survival reward", "fact_id", factId, "error", err)
			continue
		}
		if !found {
			continue // already released via a challenge-win
		}
		if pr.Deadline > height {
			continue // stale index entries cannot accelerate the real deadline
		}
		bz, err := store.Get(types.FactKey(factId))
		if err != nil {
			k.Logger(ctx).Error("read fact for survival reward", "fact_id", factId, "error", err)
			continue
		}
		if bz == nil {
			k.cancelSurvivalReward(ctx, factId)
			continue
		}
		var fact types.Fact
		if err := proto.Unmarshal(bz, &fact); err != nil || fact.Id != factId {
			k.Logger(ctx).Error("invalid fact for survival reward", "fact_id", factId)
			continue
		}
		if _, known := types.FactStatus_name[int32(fact.Status)]; !known || fact.Status == types.FactStatus_FACT_STATUS_UNSPECIFIED {
			k.Logger(ctx).Error("unknown fact status for survival reward", "fact_id", factId)
			continue
		}
		switch fact.Status {
		case types.FactStatus_FACT_STATUS_VERIFIED, types.FactStatus_FACT_STATUS_ACTIVE:
			// Challenge resolution restores ordinary facts to ACTIVE. Both that
			// state and VERIFIED must retain a reachable retry after failure.
			if err := k.releaseSurvivalReward(ctx, factId); err != nil {
				k.Logger(ctx).Error("handoff survival reward", "fact_id", factId, "error", err)
			}
		case types.FactStatus_FACT_STATUS_CHALLENGED:
			// Retain the index as well as the claim. Challenge resolution can
			// fail its handoff or be inconclusive; a later sweep must find it.
		default:
			k.cancelSurvivalReward(ctx, factId) // disproven / expired / superseded — no reward
		}
	}
}
