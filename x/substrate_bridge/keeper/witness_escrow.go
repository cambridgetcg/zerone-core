package keeper

import (
	"context"
	"encoding/json"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// ─── Witness-reward escrow ────────────────────────────────────────────────────
//
// A witness-only attestation (no cited facts, no pending claims) carries no
// verifiable knowledge — only provenance. Paying it at settlement would mint
// on mere acceptance, the exact slop vector the survival gate exists to close
// (mirror of x/knowledge/keeper/survival_escrow.go). So when an adapter
// carries a witness_reward_uzrn, settlement escrows the reward here instead:
// nothing mints until the challenge window closes with the adapter still
// ACTIVE. Tombstoning the adapter inside the window (governance falsifying
// the source) cancels every unpaid reward from it — a free clawback, because
// nothing was minted at settle. Issuance follows survival, not acceptance.

// WitnessPendingReward is a witness reward held until its window closes.
// Stored as JSON under WitnessPendingRewardPrefix.
type WitnessPendingReward struct {
	AttestationId string `json:"attestation_id"`
	AdapterId     string `json:"adapter_id"`
	Recipient     string `json:"recipient"`
	Amount        string `json:"amount"` // uzrn
	Deadline      uint64 `json:"deadline"`
}

// SetWitnessPendingReward stores the pending reward and its deadline-index entry.
func (k Keeper) SetWitnessPendingReward(ctx context.Context, pr WitnessPendingReward) {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	bz, err := json.Marshal(pr)
	if err != nil {
		return
	}
	store.Set(types.WitnessPendingRewardKey(pr.AttestationId), bz)
	store.Set(types.WitnessDeadlineIndexKey(pr.Deadline, pr.AttestationId), []byte{0x01})
}

// GetWitnessPendingReward returns the pending reward for an attestation, if any.
func (k Keeper) GetWitnessPendingReward(ctx context.Context, attestationID string) (WitnessPendingReward, bool) {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	bz := store.Get(types.WitnessPendingRewardKey(attestationID))
	if bz == nil {
		return WitnessPendingReward{}, false
	}
	var pr WitnessPendingReward
	if err := json.Unmarshal(bz, &pr); err != nil {
		return WitnessPendingReward{}, false
	}
	return pr, true
}

func (k Keeper) deleteWitnessPending(ctx context.Context, pr WitnessPendingReward) {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	store.Delete(types.WitnessPendingRewardKey(pr.AttestationId))
	store.Delete(types.WitnessDeadlineIndexKey(pr.Deadline, pr.AttestationId))
}

// EscrowWitnessReward records the adapter's witness reward as pending (nothing
// minted). Called after a witness-only attestation settles with its bond
// returned. No-op when the adapter carries no reward.
func (k Keeper) EscrowWitnessReward(ctx context.Context, att *types.ExternalAttestation) {
	adapter, found := k.GetAdapter(ctx, att.AdapterId)
	if !found || adapter.WitnessRewardUzrn == "" {
		return
	}
	reward, ok := sdkmath.NewIntFromString(adapter.WitnessRewardUzrn)
	if !ok || !reward.IsPositive() {
		return
	}
	params := k.GetParams(ctx)
	window := params.WitnessRewardChallengeWindowBlocks
	if window == 0 {
		window = 34_272 // ~1 day at 2.521s block time — defensive default
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	deadline := uint64(sdkCtx.BlockHeight()) + window
	k.SetWitnessPendingReward(ctx, WitnessPendingReward{
		AttestationId: att.AttestationId,
		AdapterId:     att.AdapterId,
		Recipient:     att.Submitter,
		Amount:        reward.String(),
		Deadline:      deadline,
	})
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeWitnessRewardEscrowed,
		sdk.NewAttribute(AttrAttestationID, att.AttestationId),
		sdk.NewAttribute(AttrAdapterID, att.AdapterId),
		sdk.NewAttribute(AttrRecipient, att.Submitter),
		sdk.NewAttribute(AttrRewardUzrn, reward.String()),
		sdk.NewAttribute(AttrDeadline, sdkmath.NewIntFromUint64(deadline).String()),
		sdk.NewAttribute(AttrUsefulWorkCommitment, "UW"),
		sdk.NewAttribute(AttrMechanism, "M4"),
	))
}

// releaseWitnessReward mints (cap-gated) and pays the escrowed reward exactly
// once, updating the attestation's RewardUzrn to what was actually paid.
// Atomic: coins and state move together or not at all.
func (k Keeper) releaseWitnessReward(ctx context.Context, pr WitnessPendingReward) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cacheCtx, writeCache := sdkCtx.CacheContext()

	recipientAddr, err := sdk.AccAddressFromBech32(pr.Recipient)
	if err != nil {
		k.cancelWitnessReward(ctx, pr, "bad recipient address")
		return
	}
	reward, ok := sdkmath.NewIntFromString(pr.Amount)
	if !ok || !reward.IsPositive() {
		k.cancelWitnessReward(ctx, pr, "bad reward amount")
		return
	}

	paid := sdkmath.ZeroInt()
	if k.vestingRewardsKeeper != nil && k.bankKeeper != nil {
		minted, err := k.vestingRewardsKeeper.MintWithCap(cacheCtx, types.AuditBountyPoolModuleName, reward.BigInt())
		if err != nil {
			k.Logger(sdkCtx).Error("witness-reward mint failed", "attestation_id", pr.AttestationId, "err", err)
			return // leave pending; retried next sweep
		}
		paid = sdkmath.NewIntFromBigInt(minted)
		if paid.IsPositive() {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", paid))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.AuditBountyPoolModuleName, recipientAddr, coins); err != nil {
				k.Logger(sdkCtx).Error("witness-reward pay failed", "attestation_id", pr.AttestationId, "err", err)
				return
			}
		}
	}

	// Record what was actually paid on the attestation (cap-clip honest).
	if att, found := k.GetAttestation(cacheCtx, pr.AttestationId); found {
		att.RewardUzrn = paid.String()
		if err := k.WriteAttestation(cacheCtx, att); err != nil {
			k.Logger(sdkCtx).Error("witness-reward attestation write failed", "attestation_id", pr.AttestationId, "err", err)
			return
		}
	}

	writeCache()
	k.deleteWitnessPending(ctx, pr)

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeWitnessRewardReleased,
		sdk.NewAttribute(AttrAttestationID, pr.AttestationId),
		sdk.NewAttribute(AttrAdapterID, pr.AdapterId),
		sdk.NewAttribute(AttrRecipient, pr.Recipient),
		sdk.NewAttribute(AttrRewardUzrn, paid.String()),
		sdk.NewAttribute(AttrUsefulWorkCommitment, "UW"),
		sdk.NewAttribute(AttrMechanism, "M4"),
	))

	if paid.IsPositive() {
		_ = k.PropagateLineage(ctx, pr.AttestationId, paid)
	}
}

// cancelWitnessReward drops a pending reward without issuing it — the adapter
// was tombstoned inside the window (source falsified) or the entry is
// malformed. The clawback is free: nothing was minted at settle.
func (k Keeper) cancelWitnessReward(ctx context.Context, pr WitnessPendingReward, reason string) {
	k.deleteWitnessPending(ctx, pr)
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		EventTypeWitnessRewardCancelled,
		sdk.NewAttribute(AttrAttestationID, pr.AttestationId),
		sdk.NewAttribute(AttrAdapterID, pr.AdapterId),
		sdk.NewAttribute(AttrRecipient, pr.Recipient),
		sdk.NewAttribute(AttrReason, reason),
	))
}

// SweepWitnessRewards resolves escrows whose window closed. Called from
// BeginBlocker. The pending entry is the source of truth for exactly-once
// issuance; the deadline index is an ordered scan hint. Adapter status at
// the deadline decides the outcome:
//
//	ACTIVE     → release (survived its window)
//	SUSPENDED  → defer one window (challenge in progress; not falsified yet)
//	TOMBSTONED → cancel (source falsified inside the window)
func (k Keeper) SweepWitnessRewards(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())
	store := sdkCtx.KVStore(k.storeKey)

	upper := append(append([]byte{}, types.WitnessDeadlineIndexPrefix...), types.BeUint64(height+1)...)
	iter := store.Iterator(types.WitnessDeadlineIndexPrefix, upper)
	// Collect due attestation IDs first; do not mutate the store during iteration.
	var due []string
	prefixLen := len(types.WitnessDeadlineIndexPrefix) + 8
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) <= prefixLen {
			continue
		}
		due = append(due, string(key[prefixLen:]))
	}
	iter.Close()

	for _, attestationID := range due {
		pr, found := k.GetWitnessPendingReward(ctx, attestationID)
		if !found {
			continue // already resolved
		}
		adapter, ok := k.GetAdapter(ctx, pr.AdapterId)
		switch {
		case !ok || adapter.Status == types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED:
			k.cancelWitnessReward(ctx, pr, "adapter tombstoned inside challenge window")
		case adapter.Status == types.AdapterStatus_ADAPTER_STATUS_SUSPENDED:
			// Mid-challenge at the deadline: defer one window rather than
			// deciding — reinstatement releases, tombstoning cancels.
			params := k.GetParams(ctx)
			window := params.WitnessRewardChallengeWindowBlocks
			if window == 0 {
				window = 34_272
			}
			k.deleteWitnessPending(ctx, pr)
			pr.Deadline = height + window
			k.SetWitnessPendingReward(ctx, pr)
		default:
			k.releaseWitnessReward(ctx, pr)
		}
	}
}

// CancelWitnessRewardsForAdapter cancels every pending witness reward escrowed
// through the given adapter. Called from TombstoneAdapter — gov-gated and rare,
// so a plain prefix scan is acceptable (an adapter-keyed index can come later).
func (k Keeper) CancelWitnessRewardsForAdapter(ctx context.Context, adapterID, reason string) {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	iter := store.Iterator(types.WitnessPendingRewardPrefix, append(append([]byte{}, types.WitnessPendingRewardPrefix...), 0xFF))
	// Collect first; do not mutate the store during iteration.
	var toCancel []WitnessPendingReward
	for ; iter.Valid(); iter.Next() {
		var pr WitnessPendingReward
		if err := json.Unmarshal(iter.Value(), &pr); err != nil {
			continue
		}
		if pr.AdapterId == adapterID {
			toCancel = append(toCancel, pr)
		}
	}
	iter.Close()
	for _, pr := range toCancel {
		k.cancelWitnessReward(ctx, pr, reason)
	}
}
