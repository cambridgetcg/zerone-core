package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
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
//
// With the cited-reward escrow marker on, a link with cited facts and no
// pending claims waits here too: citation checks only that a fact exists, so
// it adds no newly verified knowledge. Its reward is the one computeReward
// gives, and it uses the same store entries, deadline index, events and sweep
// as a witness reward. At the deadline the sweep also asks whether the facts
// it cites still stand (citedFactStanding): recognition at settlement, money
// only after the window and only while each cited fact stands.

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
	// Only the source's ref-holder may escrow (and later mint) a witness reward.
	// A duplicate that reached settlement without holding the ref — a legacy or
	// pre-arming twin — mints nothing; the source's single witness reward belongs
	// to its holder. authorizeSourceMint claims the ref for a legacy sole
	// submitter that holds none yet, so a genuine first witness is never skipped.
	if !k.authorizeSourceMint(ctx, att) {
		return
	}
	adapter, found := k.GetAdapter(ctx, att.AdapterId)
	if !found || adapter.WitnessRewardUzrn == "" {
		return
	}
	reward, ok := sdkmath.NewIntFromString(adapter.WitnessRewardUzrn)
	if !ok || !reward.IsPositive() {
		return
	}
	k.writeSurvivalEscrow(ctx, att, reward)
}

// escrowCitedReward records the reward computed for a cited-fact link with no
// pending claims as pending (nothing minted). Settlement calls it only with
// the cited-reward escrow marker on, after the attestation settles with its
// bond returned and a zero reward. finalizeSettle authorizes the source only
// for a positive reward, so with a zero reward authorizeSourceMint runs once,
// here: a non-holder duplicate escrows nothing, and a legacy sole submitter
// claims its ref as it would have at a paying settle. The sweep releases,
// defers or cancels the escrow exactly as it does a witness reward.
func (k Keeper) escrowCitedReward(ctx context.Context, att *types.ExternalAttestation, reward sdkmath.Int) {
	if reward.IsNil() || !reward.IsPositive() {
		return
	}
	if !k.authorizeSourceMint(ctx, att) {
		return
	}
	k.writeSurvivalEscrow(ctx, att, reward)
}

// writeSurvivalEscrow stores reward as pending under the challenge window and
// emits witness_reward_escrowed. Callers authorize the source first.
func (k Keeper) writeSurvivalEscrow(ctx context.Context, att *types.ExternalAttestation, reward sdkmath.Int) {
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

	// A source's single witness reward belongs to its ref-holder. The escrow-
	// time gate (EscrowWitnessReward) blocks a duplicate escrow going forward,
	// but a witness escrow written BEFORE the substrate-dedupe-v1 upgrade
	// bypassed it, and the seed reconciles only the 0x8E source index, not the
	// 0x8C pending-reward store. So re-check at release: if this attestation is
	// not the source's holder, cancel unpaid — the clawback is free because
	// nothing was minted for it.
	if att, found := k.GetAttestation(ctx, pr.AttestationId); !found {
		k.cancelWitnessReward(ctx, pr, "attestation missing at release")
		return
	} else if !k.authorizeSourceMint(ctx, att) {
		k.cancelWitnessReward(ctx, pr, "source witness reward held by another attestation")
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

// SweepWitnessRewards resolves escrows whose window closed. It is a standalone
// entry point: it reads the cited-reward escrow marker once, before any write,
// and returns a marker fault instead of sweeping. BeginBlocker reads the
// marker once per block and calls sweepWitnessRewards.
func (k Keeper) SweepWitnessRewards(ctx context.Context) error {
	citedRewardEscrow, err := k.CitedRewardEscrowEnabled(ctx)
	if err != nil {
		return err
	}
	k.sweepWitnessRewards(ctx, citedRewardEscrow)
	return nil
}

// sweepWitnessRewards is SweepWitnessRewards with the cited-reward escrow
// marker already read. The pending entry is the source of truth for
// exactly-once issuance; the deadline index is an ordered scan hint. Adapter
// status at the deadline decides first:
//
//	ACTIVE     → release (survived its window)
//	SUSPENDED  → defer one window (challenge in progress; not falsified yet)
//	TOMBSTONED → cancel (source falsified inside the window)
//
// With citedRewardEscrow true, an escrow the adapter rule would release is
// then decided by the facts its attestation cites (citedFactStanding): cancel
// if one is missing, DISPROVEN or REVOKED; otherwise defer one window if one
// is CHALLENGED or CONTESTED; otherwise release. A witness-only escrow cites
// nothing and is released as before. With citedRewardEscrow false no fact is
// read: this is the sweep that ran before the marker existed.
//
// Both deferrals repeat at each new deadline, with no count, no limit and no
// event, until the adapter or the cited facts let the sweep decide. A cited
// fact can stay CONTESTED with no end. x/knowledge's metabolism does not
// process CONTESTED facts. With claim integrity on, a fact whose contest check
// cannot finish stays CONTESTED until a challenge refutes it
// (x/knowledge/keeper/claim_integrity_contest.go). With claim integrity off, a
// CONTESTED fact cannot be challenged at all, and admitting a contradiction of
// a VERIFIED or ACTIVE fact makes it CONTESTED at once. Such an escrow stays
// pending for as long as the fact stays CONTESTED.
func (k Keeper) sweepWitnessRewards(ctx context.Context, citedRewardEscrow bool) {
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
			k.deferWitnessReward(ctx, pr, height)
		case !citedRewardEscrow:
			k.releaseWitnessReward(ctx, pr)
		default:
			// The adapter rule would release. With the marker on, the cited
			// facts decide whether the reward is still owed.
			switch standing, reason := k.citedFactStanding(ctx, pr); standing {
			case citedFallen:
				k.cancelWitnessReward(ctx, pr, reason)
			case citedChallenged:
				// Under challenge at the deadline: defer one window, as for
				// a SUSPENDED adapter. The next sweep decides again.
				k.deferWitnessReward(ctx, pr, height)
			default:
				k.releaseWitnessReward(ctx, pr)
			}
		}
	}
}

// deferWitnessReward moves an escrow's deadline one window past height and
// decides nothing. The pending entry and its deadline-index entry move
// together; nothing is minted and no event is emitted.
func (k Keeper) deferWitnessReward(ctx context.Context, pr WitnessPendingReward, height uint64) {
	params := k.GetParams(ctx)
	window := params.WitnessRewardChallengeWindowBlocks
	if window == 0 {
		window = 34_272
	}
	k.deleteWitnessPending(ctx, pr)
	pr.Deadline = height + window
	k.SetWitnessPendingReward(ctx, pr)
}

// citedStanding is what the facts an escrowed attestation cites say when its
// window closes.
type citedStanding uint8

const (
	// citedStand: every cited fact stands, or nothing is cited. Release.
	citedStand citedStanding = iota
	// citedChallenged: a cited fact is CHALLENGED or CONTESTED and none has
	// fallen. Defer one window; the next sweep decides again, with no limit.
	citedChallenged
	// citedFallen: a cited fact is missing, DISPROVEN or REVOKED. Cancel.
	citedFallen
)

// citedFactStanding reads each fact the escrow's attestation cites, in
// citation order, through the knowledge keeper. It writes nothing. The first
// fact that is missing from the knowledge store, DISPROVEN or REVOKED makes
// the result citedFallen, and reason names that fact. Otherwise any CHALLENGED
// or CONTESTED fact makes it citedChallenged. Every other status stands. In
// particular AT_RISK, EXPIRED and PRUNED stand: x/knowledge sets them through
// energy decay, niche competition, or the energy a child fact loses when its
// parent is disproven, and none of these finds the fact itself false. The
// statuses this rule does not name also stand: UNSPECIFIED, PENDING,
// PROVISIONAL, VERIFIED, ACTIVE, SUPERSEDED, and any value this build does
// not know.
//
// Admission bounds new links at MaxCitedFactsPerLink. This reads every fact a
// stored link cites, so a failing fact past that bound still decides.
//
// A witness-only attestation cites nothing and stands. A missing attestation
// stands here because releaseWitnessReward cancels it. A keeper wired without
// a knowledge keeper reads no fact and stands, as submission and settlement
// read no fact without one; the app always wires it.
//
// It reads the attestation for every escrow the adapter rule would release, a
// witness-only one included. So with the marker on, a witness-only escrow
// costs one more store read at release; its state, events and coins are the
// same as with the marker off.
func (k Keeper) citedFactStanding(ctx context.Context, pr WitnessPendingReward) (citedStanding, string) {
	if k.knowledgeKeeper == nil {
		return citedStand, ""
	}
	att, found := k.GetAttestation(ctx, pr.AttestationId)
	if !found || att.Link == nil {
		return citedStand, ""
	}
	standing := citedStand
	for _, c := range att.Link.CitedFacts {
		if c == nil {
			continue
		}
		fact, found := k.knowledgeKeeper.GetFact(ctx, c.FactId)
		if !found || fact == nil {
			return citedFallen, fmt.Sprintf("cited fact %s missing from the knowledge store", c.FactId)
		}
		switch fact.Status {
		case knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN,
			knowledgetypes.FactStatus_FACT_STATUS_REVOKED:
			return citedFallen, fmt.Sprintf("cited fact %s is %s", c.FactId, fact.Status)
		case knowledgetypes.FactStatus_FACT_STATUS_CHALLENGED,
			knowledgetypes.FactStatus_FACT_STATUS_CONTESTED:
			standing = citedChallenged
		}
	}
	return standing, ""
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
