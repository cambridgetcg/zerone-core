package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// ─── Source-reference uniqueness index ───────────────────────────────────────
//
// One external work item mints at most once. The (adapter_id, source_id)
// pair is the economic identity of the work — link_hash cannot serve as
// the replay key because it also hashes fetched_at_block and the axis
// weights, so the same work re-submitted with any of those varied is a
// "new" link. The index entry is written at submission, deleted when the
// attestation is REJECTED (rejection punishes the submission, not the
// work's identity — honest resubmission stays possible, and a junk
// submission cannot squat a source forever), and kept for every settled
// outcome.

// SetSourceRef records attestationID as the holder of (adapterID, sourceID).
func (k Keeper) SetSourceRef(ctx context.Context, adapterID, sourceID, attestationID string) {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	store.Set(types.SourceRefKey(adapterID, sourceID), []byte(attestationID))
}

// GetSourceRef returns the attestation ID holding (adapterID, sourceID), if any.
func (k Keeper) GetSourceRef(ctx context.Context, adapterID, sourceID string) (string, bool) {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	bz := store.Get(types.SourceRefKey(adapterID, sourceID))
	if bz == nil {
		return "", false
	}
	return string(bz), true
}

// DeleteSourceRef removes the index entry for (adapterID, sourceID).
func (k Keeper) DeleteSourceRef(ctx context.Context, adapterID, sourceID string) {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	store.Delete(types.SourceRefKey(adapterID, sourceID))
}

// releaseSourceRef drops the index entry held by att, if att carries a
// source reference and still holds it. The holder check is defensive: a
// source released by an earlier rejection may since have been re-claimed
// by a later attestation, and that holder must never lose its entry.
func (k Keeper) releaseSourceRef(ctx context.Context, att *types.ExternalAttestation) {
	if att == nil || att.Link == nil || att.Link.Source == nil || att.Link.Source.SourceId == "" {
		return
	}
	holder, found := k.GetSourceRef(ctx, att.Link.AdapterId, att.Link.Source.SourceId)
	if !found || holder != att.AttestationId {
		return
	}
	k.DeleteSourceRef(ctx, att.Link.AdapterId, att.Link.Source.SourceId)
}

// ─── Enforcement arming ───────────────────────────────────────────────────────
//
// The submit-time checks (source required, uniqueness) live in the binary,
// so they would activate the instant a node runs the new code — but the
// 0x8E index they read is empty until SeedSourceRefs backfills it from
// history. On a coordinated x/upgrade plan there is no gap: the handler
// seeds in the same PreBlock the binary swaps. But this chain has a
// demonstrated habit of plan-less binary restarts, and a restart onto the
// new binary WITHOUT the handler running would leave the index empty while
// enforcement is live — every historical source would look free, and a
// replay would mint again: the exact hole this change closes, reopened
// silently.
//
// So enforcement is gated on an explicit "armed" flag set only where the
// index is guaranteed seeded: the substrate-dedupe-v1 upgrade handler (for
// existing chains) and InitGenesis (for chains born with this code).
// Until armed, SubmitExternalAttestation refuses fail-closed — a loud,
// safe halt of the witness bridge on a mis-deploy, never a silent replay.

func (k Keeper) IsDedupeArmed(ctx context.Context) bool {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	return store.Get(types.DedupeArmedKey) != nil
}

func (k Keeper) SetDedupeArmed(ctx context.Context) {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	store.Set(types.DedupeArmedKey, []byte{0x01})
}

// hasAnyAttestation reports whether any attestation record exists — the actual
// seed domain. It probes the attestation prefix directly rather than trusting
// the ID counter as a proxy: any code that writes attestation records without
// bumping the counter (a backfill migration, a GenesisState import of
// attestations, a second msg path) would otherwise leave the counter absent
// while records exist, and the fresh-chain self-arm would then wrongly fire
// against unseeded history. The iterator seeks to the first key only, so this
// is O(1), and it runs only on not-yet-armed submissions.
func (k Keeper) hasAnyAttestation(ctx context.Context) bool {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ExternalAttestationPrefix)
	defer iter.Close()
	return iter.Valid()
}

// authorizeSourceMint reports whether att may mint for its source, and claims
// the source ref for att when no attestation holds it yet. This makes the
// source ref a genuine mint authorization checked at settlement, not only at
// submission: a legacy or pre-arming duplicate that reaches settlement without
// holding its source (seeded as a loser, or created in the window before the
// index was armed) must not mint — the source's single mint belongs to its
// ref-holder. A sourced attestation that holds no ref yet (a legacy sole
// submitter the seed did seat, or one whose source was never contested) claims
// it here and mints, so the wall never suppresses a genuine first mint.
// Attestations without a source predate the wall entirely and are unaffected.
func (k Keeper) authorizeSourceMint(ctx context.Context, att *types.ExternalAttestation) bool {
	if att == nil || att.Link == nil || att.Link.Source == nil || att.Link.Source.SourceId == "" {
		return true
	}
	holder, found := k.GetSourceRef(ctx, att.Link.AdapterId, att.Link.Source.SourceId)
	if !found {
		k.SetSourceRef(ctx, att.Link.AdapterId, att.Link.Source.SourceId, att.AttestationId)
		return true
	}
	return holder == att.AttestationId
}

// ensureDedupeArmed decides whether SubmitExternalAttestation may proceed and
// arms enforcement when it is safe to do so implicitly. The danger the arming
// flag guards is enforcing the uniqueness checks against an EMPTY index while
// unindexed history exists — a replay would look free and mint again.
//
//   - Already armed (InitGenesis on a fresh chain, or the substrate-dedupe-v1
//     handler on an existing one, seeded the index): proceed.
//   - Not armed but the chain has NO attestations at all: nothing to seed, so
//     enforcement against the empty index is correct — arm now and proceed.
//     This self-heals a fresh chain even if genesis-time arming never ran, and
//     costs one write on the first-ever submission only.
//   - Not armed AND attestations already exist: an existing chain whose seed
//     handler has not run (a plan-less restart). Refuse fail-closed rather than
//     risk a replay-mint; enforcement resumes once the handler seeds + arms.
func (k Keeper) ensureDedupeArmed(ctx context.Context) error {
	if k.IsDedupeArmed(ctx) {
		return nil
	}
	if k.hasAnyAttestation(ctx) {
		return types.ErrDedupeNotArmed
	}
	k.SetDedupeArmed(ctx)
	return nil
}

// sourceRefSeedTier ranks how strong a claim an attestation in this status
// has to hold its source, so the seed seats the safest holder among pre-
// upgrade duplicates:
//
//	2  minted (SETTLED, PARTIAL)      — already paid out; the source must
//	                                    stay locked forever and this holder
//	                                    never releases it.
//	1  in flight (COMMITTED, AWAITING — will settle later; a fine holder
//	   _RESOLUTION, READY)              ONLY when no minted twin exists, and
//	                                    it correctly frees the source if it
//	                                    later rejects (it never minted).
//	0  released (REJECTED, and the     — not seeded; on the live chain a
//	   never-produced SLASHED)           rejection releases the source, so a
//	                                    source seen only in rejected records
//	                                    stays free for honest resubmission.
//
// The tiers exist because a naive first-in-KV-order holder is unsafe:
// attestation IDs (att-<height>-<seq>) sort lexicographically, not
// chronologically, so an in-flight duplicate could be seated over an
// already-minted twin and then free the source when it later times out —
// a migration-window replay-mint (the reviewer's confirmed finding).
func sourceRefSeedTier(status types.AttestationStatus) int {
	switch status {
	case types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
		types.AttestationStatus_ATTESTATION_STATUS_PARTIAL:
		return 2
	case types.AttestationStatus_ATTESTATION_STATUS_REJECTED,
		types.AttestationStatus_ATTESTATION_STATUS_SLASHED:
		return 0
	default:
		return 1
	}
}

// SeedSourceRefs backfills the index from every stored attestation —
// pre-upgrade history becomes the wall's first bricks. Deterministic:
// two ordered passes over the attestation prefix, minted holders (tier 2)
// before in-flight ones (tier 1); rejected records (tier 0) are never
// seeded, so a source seen only in rejections stays free. Records without
// a source (pre-dedupe hand submissions) are counted and returned so the
// upgrade log names the wall's holes rather than hiding them.
func (k Keeper) SeedSourceRefs(ctx context.Context) (seeded, duplicates, sourceless uint64, err error) {
	// countSourceless is set only on the first pass so a record lacking a
	// source is tallied exactly once (over all records, whatever their
	// tier). duplicates is tallied on each pass for records of that pass's
	// own tier, so a losing in-flight duplicate is counted too.
	seedPass := func(tier int, countSourceless bool) {
		store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
		iter := storetypes.KVStorePrefixIterator(store, types.ExternalAttestationPrefix)
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			var att types.ExternalAttestation
			if err := k.cdc.Unmarshal(iter.Value(), &att); err != nil {
				continue // mirror GetAttestation: unreadable records are invisible
			}
			hasSource := att.Link != nil && att.Link.Source != nil && att.Link.Source.SourceId != ""
			if !hasSource {
				if countSourceless {
					sourceless++
				}
				continue
			}
			if sourceRefSeedTier(att.Status) != tier {
				continue
			}
			if _, taken := k.GetSourceRef(ctx, att.Link.AdapterId, att.Link.Source.SourceId); taken {
				duplicates++
				continue
			}
			k.SetSourceRef(ctx, att.Link.AdapterId, att.Link.Source.SourceId, att.AttestationId)
			seeded++
		}
	}
	seedPass(2, true)  // minted holders win the ref; count sourceless here
	seedPass(1, false) // then in-flight, for sources no minted twin claimed
	return seeded, duplicates, sourceless, nil
}
