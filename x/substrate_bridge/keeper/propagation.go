package keeper

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// PropagateLineage distributes a depth-decayed royalty flow from the
// downstream attestation back through its lineage DAG. Budget is
// downstreamReward × Params.LineageShareBps (default 30%). Decay
// per-hop is Params.DecayBpsPerHop. Halts at Params.MaxPropagationDepth
// or when remaining share < Params.MinPropagationUzrn.
//
// Lineage payments come from the downstream's reward, NOT new minting
// (no inflation pressure; M4 formula unchanged).
//
// The caller is responsible for the actual SendCoinsFromModuleToAccount
// — this function returns the (recipient_addr, amount) list and updates
// LineageEdge.SettlementPaymentUzrn + LineageRoyaltyAccumulator.
// In practice the caller will be Settlement.SettleAttestation.
func (k Keeper) PropagateLineage(ctx context.Context, downstreamID string, downstreamReward sdkmath.Int) error {
	params := k.GetParams(ctx)
	lineageShareBps := sdkmath.NewIntFromUint64(uint64(params.LineageShareBps))
	totalBudget := downstreamReward.Mul(lineageShareBps).Quo(sdkmath.NewInt(10000))

	if totalBudget.IsZero() {
		return nil
	}

	minProp, ok := sdkmath.NewIntFromString(params.MinPropagationUzrn)
	if !ok || minProp.IsNil() {
		minProp = sdkmath.NewInt(1000)
	}

	return k.propagateRecursive(ctx, downstreamID, totalBudget, 1, params.MaxPropagationDepth, params.DecayBpsPerHop, minProp)
}

func (k Keeper) propagateRecursive(
	ctx context.Context,
	currentDownstream string,
	remainingBudget sdkmath.Int,
	depth, maxDepth uint32,
	decayBpsPerHop uint32,
	minPropagation sdkmath.Int,
) error {
	if depth > maxDepth || remainingBudget.LT(minPropagation) {
		return nil
	}

	// Walk this node's upstream edges.
	var edges []*types.LineageEdge
	k.IterateBackwardLineage(ctx, currentDownstream, func(e *types.LineageEdge) bool {
		edges = append(edges, e)
		return false
	})

	totalShare := sdkmath.ZeroInt()
	for _, e := range edges {
		weight := sdkmath.NewIntFromUint64(uint64(types.CitationTypeWeight(e.CitationType)))
		if weight.IsZero() {
			continue
		}
		share := remainingBudget.
			Mul(sdkmath.NewIntFromUint64(uint64(e.ContributionShareBps))).
			Mul(weight).
			Quo(sdkmath.NewInt(10000))
		// Clamp to remaining budget less totalShare so far.
		availableRemaining := remainingBudget.Sub(totalShare)
		if share.GT(availableRemaining) {
			share = availableRemaining
		}
		if share.LT(minPropagation) {
			continue
		}
		totalShare = totalShare.Add(share)

		// Update upstream's accumulator.
		acc, found := k.GetLineageAccumulator(ctx, e.UpstreamAttestationId)
		if !found {
			acc = &types.LineageRoyaltyAccumulator{
				AttestationId:  e.UpstreamAttestationId,
				CumulativeUzrn: "0",
			}
		}
		cur, ok3 := sdkmath.NewIntFromString(acc.CumulativeUzrn)
		if !ok3 {
			cur = sdkmath.ZeroInt()
		}
		acc.CumulativeUzrn = cur.Add(share).String()
		acc.LastUpdatedBlock = uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
		acc.IncomingEdgeCount++
		k.WriteLineageAccumulator(ctx, acc)

		// Mark edge's settlement_payment forward-only.
		curEdgePayment, ok2 := sdkmath.NewIntFromString(e.SettlementPaymentUzrn)
		if !ok2 {
			curEdgePayment = sdkmath.ZeroInt()
		}
		newEdgePayment := curEdgePayment.Add(share)
		e.SettlementPaymentUzrn = newEdgePayment.String()
		store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
		store.Set(types.LineageEdgeKey(types.EdgeID(e.UpstreamAttestationId, e.DownstreamAttestationId)), k.cdc.MustMarshal(e))

		// Recursively propagate further up.
		propagatedShare := share.Mul(sdkmath.NewIntFromUint64(uint64(decayBpsPerHop))).Quo(sdkmath.NewInt(10000))
		if err := k.propagateRecursive(ctx, e.UpstreamAttestationId, propagatedShare, depth+1, maxDepth, decayBpsPerHop, minPropagation); err != nil {
			return err
		}
	}
	return nil
}
