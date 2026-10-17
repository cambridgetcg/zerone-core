package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Restoring fact edges that equal-ID augmentations replaced (knowledge v12;
// approved by Yu on 2026-10-17; spec §1.5, plan §8).
//
// Before claim integrity, a passing augmentation verdict wrote its
// REFORMULATES edge at the key <augmentation ID>/<original fact>, and the ID
// was chosen by the augmentation's submitter. An augmentation of D whose ID
// equalled the ID of a fact F that already had its own edge to D replaced that
// edge, forward key and reverse entry: the store keeps one edge per pair. The
// cascade and the contest check (R1, R2) read the stored edge, so they lost
// it. A fact accepted from a claim had every outgoing edge written by
// createFactFromClaim, from its claim's relations, its claim's submitter and
// its creation height, and all of those are retained. So each such edge can
// be written again exactly as createFactFromClaim wrote it.
//
// RestoreAugmentationReplacedEdges does that. It runs in Migrate11to12, and in
// InitGenesis when the genesis selects claim integrity (a relaunch from an
// export carries the replaced edges with it). It reads every augmentation, in
// key order, up to the inventory bound; above it, it fails rather than skip
// any. For each augmentation aug of fact D whose verdict passed, the pair
// (aug.Id, D) is examined only when its forward entry holds exactly the edge
// the verdict wrote (REFORMULATES, created at aug.VerdictBlock, no other
// field), so nothing wrote the pair after the verdict, and aug.Id is the ID of
// a stored fact F. Then:
//
//   - F accepted from a claim C that names D: restored when F's ID is
//     GenerateFactID(C, F.VerifiedAtBlock), the reverse entry also holds the
//     verdict's edge, and the method can be placed (see below). The edge is
//     C's last relation naming D (createFactFromClaim writes C's relations in
//     order, so the last one is what was stored), created at
//     F.VerifiedAtBlock by C's submitter. Emits augmentation_edge_restored.
//   - F accepted from a claim that does not name D: F never had an edge to D,
//     so nothing was replaced. Nothing is written or emitted.
//   - otherwise (F has no claim, its claim is missing, its ID does not derive
//     from its claim, the reverse entry differs, or the method cannot be
//     placed): left as it is, and augmentation_edge_unrestorable says why. A
//     fact with no claim (genesis, doctrine, an authority injection) may or
//     may not have had an edge to D; nothing on chain says which.
//
// Whether F existed at the verdict is proved, for the first case, by the
// forward entry alone: createFactFromClaim writes F's edge to every target C
// names, with C's submitter as creator, and the verdict's edge has none, so
// had F been created after the verdict its own write would hold the pair.
// Heights are not compared there. They could not be: after a relaunch at
// initial height 1 a verdict's height and a fact's verified_at_block may come
// from two chains. For a fact that cannot be traced to a claim (no claim, a
// missing claim, an ID that does not derive from it) nothing proves it, and
// such a pair is reported only when verified_at_block is at or below
// verdict_block. On one chain that is exact; after a relaunch at initial
// height 1 such a report can be missing or spurious. Only reports depend on
// it, never a write.
//
// createFactFromClaim copies a relation's method_id only with record
// integrity on. When the relation carries one, the claim's round must use the
// review-v2 commitment scheme, which only a round created with record
// integrity on has; record integrity is never turned off, so it was on when F
// was created. A legacy-scheme round could have completed either side of
// record integrity's activation, so such an edge is not guessed.
//
// Nothing else is written: no status, history, index or counter. A fact the
// replaced edge would have contested in a cascade before activation is not
// contested now; the contest check and the cascade read the restored edge
// from now on. Reads that fail, and malformed records, are errors.

// augmentationRestoreMaxRecords and augmentationRestoreMaxBytes bound the
// augmentation inventory one restoration reads, as the record-integrity
// activation inventory is bounded.
var (
	augmentationRestoreMaxRecords = recordInventoryMaxRecords
	augmentationRestoreMaxBytes   = recordInventoryMaxBytes
)

// AugmentationEdgePair is one pair RestoreAugmentationReplacedEdges restored,
// or reported as unrestorable with Reason. AugmentationID equals FactID.
type AugmentationEdgePair struct {
	FactID, TargetID, AugmentationID, Reason string
}

// AugmentationEdgeRestoration is what one run of
// RestoreAugmentationReplacedEdges did, each list in augmentation key order.
type AugmentationEdgeRestoration struct {
	Restored, Unrestorable []AugmentationEdgePair
}

// RestoreAugmentationReplacedEdges restores, in augmentation key order, every
// fact edge an equal-ID augmentation replaced (see above), emits one event per
// restored or unrestorable pair, and returns the pairs.
func (k Keeper) RestoreAugmentationReplacedEdges(ctx context.Context) (AugmentationEdgeRestoration, error) {
	var result AugmentationEdgeRestoration
	augs, err := k.passingAugmentationsChecked(ctx)
	if err != nil {
		return AugmentationEdgeRestoration{}, err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, aug := range augs {
		edge, reason, err := k.replacedFactEdge(ctx, aug)
		if err != nil {
			return AugmentationEdgeRestoration{}, fmt.Errorf("augmentation %s: %w", aug.Id, err)
		}
		pair := AugmentationEdgePair{FactID: aug.Id, TargetID: aug.OriginalFactId, AugmentationID: aug.Id, Reason: reason}
		switch {
		case edge != nil:
			if err := k.SetFactRelation(ctx, edge); err != nil {
				return AugmentationEdgeRestoration{}, fmt.Errorf("restore fact edge %s→%s: %w", edge.SourceFactId, edge.TargetFactId, err)
			}
			result.Restored = append(result.Restored, pair)
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.augmentation_edge_restored",
				sdk.NewAttribute("fact_id", pair.FactID),
				sdk.NewAttribute("target_id", pair.TargetID),
				sdk.NewAttribute("augmentation_id", pair.AugmentationID),
			))
		case reason != "":
			result.Unrestorable = append(result.Unrestorable, pair)
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.augmentation_edge_unrestorable",
				sdk.NewAttribute("fact_id", pair.FactID),
				sdk.NewAttribute("target_id", pair.TargetID),
				sdk.NewAttribute("augmentation_id", pair.AugmentationID),
				sdk.NewAttribute("reason", pair.Reason),
			))
		}
	}
	return result, nil
}

// logAugmentationEdgeRestoration writes a successful restoration to the node
// log: the counts, then each pair in augmentation key order. origin is
// "upgrade" (Migrate11to12) or "genesis" (InitGenesis with the flag: a
// relaunch from an export, or a native genesis, which usually has nothing to
// restore). The events of InitGenesis go nowhere (BaseApp.InitChain discards
// them; CometBFT 0.38's ResponseInitChain has no events), so for an import the
// log is the only record outside the store. The log is not state.
func (k Keeper) logAugmentationEdgeRestoration(ctx context.Context, origin string, r AugmentationEdgeRestoration) {
	logger := k.Logger(ctx).With("origin", origin)
	logger.Info("claim integrity: augmentation-replaced fact edges",
		"restored", len(r.Restored), "unrestorable", len(r.Unrestorable))
	for _, pair := range r.Restored {
		logger.Info("claim integrity: augmentation-replaced fact edge restored",
			"fact_id", pair.FactID, "target_id", pair.TargetID, "augmentation_id", pair.AugmentationID)
	}
	for _, pair := range r.Unrestorable {
		logger.Info("claim integrity: augmentation-replaced fact edge unrestorable",
			"fact_id", pair.FactID, "target_id", pair.TargetID, "augmentation_id", pair.AugmentationID, "reason", pair.Reason)
	}
}

// passingAugmentationsChecked reads every augmentation in key order and
// returns those whose verdict passed (EQUIVALENT or SUPERIOR). Exceeding the
// bound, an unreadable store or a malformed record is an error; no record is
// skipped.
func (k Keeper) passingAugmentationsChecked(ctx context.Context) (out []*types.Augmentation, err error) {
	it, err := k.storeService.OpenKVStore(ctx).Iterator(types.AugmentationKeyPrefix, prefixEndBytes(types.AugmentationKeyPrefix))
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, it.Close())
		if err != nil {
			out = nil
		}
	}()
	records, size := 0, 0
	for ; it.Valid(); it.Next() {
		key, value := it.Key(), it.Value()
		records++
		size += len(key) + len(value)
		if records > augmentationRestoreMaxRecords || size > augmentationRestoreMaxBytes {
			return nil, fmt.Errorf("augmentation inventory exceeds %d records or %d bytes", augmentationRestoreMaxRecords, augmentationRestoreMaxBytes)
		}
		if err := validateToKStoredRecord(key, value); err != nil {
			return nil, fmt.Errorf("augmentation record %x: %w", key, err)
		}
		aug := &types.Augmentation{}
		if err := proto.Unmarshal(value, aug); err != nil {
			return nil, fmt.Errorf("augmentation record %x: %w", key, err)
		}
		if aug.Verdict == types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT ||
			aug.Verdict == types.AugmentationVerdict_AUGMENTATION_VERDICT_SUPERIOR {
			out = append(out, aug)
		}
	}
	return out, historyIteratorError(it)
}

// replacedFactEdge decides one passing augmentation (see above). It returns
// the edge to restore, or the reason the pair is unrestorable, or neither when
// the augmentation replaced no fact's edge.
func (k Keeper) replacedFactEdge(ctx context.Context, aug *types.Augmentation) (*types.FactRelation, string, error) {
	if aug.Id == "" || aug.OriginalFactId == "" {
		return nil, "", nil
	}
	store := k.storeService.OpenKVStore(ctx)
	written, err := marshalOpts.Marshal(&types.FactRelation{
		SourceFactId:   aug.Id,
		TargetFactId:   aug.OriginalFactId,
		Relation:       types.RelationType_RELATION_TYPE_REFORMULATES,
		CreatedAtBlock: aug.VerdictBlock,
	})
	if err != nil {
		return nil, "", err
	}
	forward, err := store.Get(types.FactRelationKey(aug.Id, aug.OriginalFactId))
	if err != nil {
		return nil, "", err
	}
	if !bytes.Equal(forward, written) {
		return nil, "", nil // the verdict wrote no edge, or a later write owns the pair
	}
	fact, found, err := k.getFactChecked(ctx, aug.Id)
	if err != nil || !found {
		return nil, "", err // an ordinary augmentation edge
	}
	// untraced reports a pair whose fact cannot be traced to a claim, unless
	// its heights say the fact was created after the verdict (see above:
	// this decides a report, never a write).
	untraced := func(reason string) (*types.FactRelation, string, error) {
		if fact.VerifiedAtBlock > aug.VerdictBlock {
			return nil, "", nil
		}
		return nil, reason, nil
	}
	if fact.ClaimId == "" {
		return untraced("fact has no claim")
	}
	claim, err := k.getClaimRecordChecked(ctx, fact.ClaimId)
	if err != nil {
		return nil, "", err
	}
	if claim == nil {
		return untraced(fmt.Sprintf("claim %s of the fact is missing", fact.ClaimId))
	}
	if GenerateFactID(claim.Id, fact.VerifiedAtBlock) != fact.Id {
		return untraced("fact id does not derive from its claim and verification height")
	}
	var named *types.ClaimRelation
	for _, rel := range claim.Relations {
		if rel != nil && rel.TargetFactId == aug.OriginalFactId {
			named = rel
		}
	}
	if named == nil {
		return nil, "", nil // F never had an edge to this target
	}
	reverse, err := store.Get(types.FactRelationReverseKey(aug.OriginalFactId, aug.Id))
	if err != nil {
		return nil, "", err
	}
	if !bytes.Equal(reverse, written) {
		return nil, "reverse entry is not the augmentation's edge", nil
	}
	edge := &types.FactRelation{
		SourceFactId:         fact.Id,
		TargetFactId:         named.TargetFactId,
		Relation:             named.Relation,
		CreatedAtBlock:       fact.VerifiedAtBlock,
		Creator:              claim.Submitter,
		Inference:            named.Inference,
		InferenceStrengthBps: named.InferenceStrengthBps,
	}
	if named.MethodId != "" {
		placed, err := k.roundProvesRecordIntegrity(ctx, claim.VerificationRoundId)
		if err != nil {
			return nil, "", err
		}
		if !placed {
			return nil, "relation method_id is set but the claim's round does not prove record integrity at the fact's creation", nil
		}
		edge.MethodId = named.MethodId
	}
	return edge, "", nil
}

// roundProvesRecordIntegrity reports whether round roundID is stored with the
// review-v2 commitment scheme, which a round has only when created with record
// integrity on. An absent round proves nothing.
func (k Keeper) roundProvesRecordIntegrity(ctx context.Context, roundID string) (bool, error) {
	if roundID == "" {
		return false, nil
	}
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.RoundKey(roundID))
	if err != nil || bz == nil {
		return false, err
	}
	round, err := k.getVerificationRoundChecked(ctx, roundID)
	if err != nil {
		return false, err
	}
	return round.CommitmentScheme == types.CommitmentSchemeReviewV2, nil
}
