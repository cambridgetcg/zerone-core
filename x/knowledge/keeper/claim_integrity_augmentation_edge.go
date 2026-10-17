package keeper

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Augmentation edges with claim integrity on (knowledge v12, review G1R).
//
// A passing augmentation verdict writes a REFORMULATES edge whose source is
// the augmentation's ID, which its submitter chose, into the fact relation
// index (ApplyFinalizedAugmentationVerdict). Relation keys are
// "<source>/<target>", so an ID containing '/' lands under another ID's
// forward-index prefix, and an ID equal to a fact's ID overwrites that fact's
// own edge to the same target (the store keeps one edge per pair). With claim
// integrity on such an edge is not written; the verdict, payout and
// acceptance are applied unchanged, and augmentation_edge_withheld says why.
// Review neutrality, a predecessor of 0x87, already refuses new
// augmentations, so this governs only those admitted before it whose ballots
// finalize after activation. Edges written before activation are a closed,
// auditable set that this does not rewrite. With the marker off the edge is
// written exactly as knowledge 11 writes it.

// AugmentationEdgeSourceMaxBytes is the longest augmentation ID whose
// REFORMULATES edge is written with claim integrity on. Chain-generated IDs
// (facts, claims, rounds) are 32 hex characters and doctrine IDs are shorter;
// 128 bytes leaves four times that for client-chosen IDs (a UUID is 36 bytes,
// a prefixed SHA-256 hex digest about 70). It also keeps each edge entry
// under about 340 bytes (the ID appears in the key and the payload), so the
// 100k records the contest check may read stay far below its 64 MiB byte
// bound: entries of capped IDs alone can no longer reach the byte bound
// first, as about 343 IDs of 96 KiB did (review G1R).
const AugmentationEdgeSourceMaxBytes = 128

// augmentationEdgeWithheld reports why, with claim integrity on, the
// REFORMULATES edge of passing augmentation aug must not be written, or ""
// when it may be. It reads the marker, and the fact index only when the
// marker is on.
func (k Keeper) augmentationEdgeWithheld(ctx context.Context, aug *types.Augmentation) (string, error) {
	enabled, err := k.ClaimIntegrityEnabled(ctx)
	if err != nil || !enabled {
		return "", err
	}
	switch {
	case strings.Contains(aug.Id, "/"):
		return "augmentation id contains '/'", nil
	case len(aug.Id) > AugmentationEdgeSourceMaxBytes:
		return fmt.Sprintf("augmentation id is longer than %d bytes", AugmentationEdgeSourceMaxBytes), nil
	}
	isFact, err := k.storeService.OpenKVStore(ctx).Has(types.FactKey(aug.Id))
	if err != nil {
		return "", fmt.Errorf("read fact %s: %w", aug.Id, err)
	}
	if isFact {
		return "augmentation id is the id of a fact", nil
	}
	return "", nil
}

func emitAugmentationEdgeWithheld(ctx context.Context, aug *types.Augmentation, reason string) {
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.augmentation_edge_withheld",
		sdk.NewAttribute("augmentation_id", aug.Id),
		sdk.NewAttribute("original_fact_id", aug.OriginalFactId),
		sdk.NewAttribute("reason", reason),
	))
}
