package keeper

import (
	"context"
	"fmt"
	"sort"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ErrToKCascadeNotDisproven is the typed error for "selector root is not DISPROVEN".
var ErrToKCascadeNotDisproven = fmt.Errorf("cascade-replay root is not DISPROVEN")

// GatherCascade walks the disproval-graph from a DISPROVEN root. Returns:
//   - nodeIDs: disproven root + cascaded descendants (sorted)
//   - edges:   CONTRADICTS + first-hop support edges to descendants (sorted)
//   - cascadeEvents: every recorded CascadeEvent for this disproof
//   - vindications:  every ToKVindicationRecord for this disproof's facts
//                    (only populated if sel.IncludeVindications)
//   - supersessionChain: SUPERSEDES walk (only if sel.IncludeSupersessions)
//
// TC4: the graph carries its disprovals. The disproval-graph is the parallel
// of Plan 1's support-graph. Plan 1's selectors filter CONTRADICTS|SUPERSEDES;
// this selector includes them.
func (k Keeper) GatherCascade(
	ctx context.Context,
	sel *types.CascadeReplaySelector,
) (
	nodeIDs []string,
	edges []*types.ToKEdge,
	cascadeEvents []*types.CascadeEvent,
	vindications []*types.ToKVindicationRecord,
	supersessionChain []string,
	err error,
) {
	root, found := k.GetFact(ctx, sel.DisprovenFactId)
	if !found {
		return nil, nil, nil, nil, nil, fmt.Errorf("%w: %s", ErrToKRootFactNotFound, sel.DisprovenFactId)
	}
	if root.Status != types.FactStatus_FACT_STATUS_DISPROVEN {
		return nil, nil, nil, nil, nil, fmt.Errorf("%w (TC4): root %s has status %s, not DISPROVEN",
			ErrToKCascadeNotDisproven, sel.DisprovenFactId, root.Status)
	}

	// Collect cascade events for this disproof.
	cascadeEvents = k.GetCascadeEventsForDisproof(ctx, sel.DisprovenFactId)

	// Build node set: root + every descendant in cascade events.
	visited := map[string]bool{root.Id: true}
	for _, ev := range cascadeEvents {
		visited[ev.DescendantFactId] = true
	}

	// For depth > 1, walk transitively from each cascaded descendant.
	if sel.MaxDepth > 1 {
		for _, ev := range cascadeEvents {
			if err := k.gatherCascadeRecursive(ctx, ev.DescendantFactId, 1, sel.MaxDepth, visited); err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
	}

	// Materialise node ID list.
	for id := range visited {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	// Build edge set:
	//   (a) CONTRADICTS edges into the disproven root
	//   (b) support-bearing edges from descendants → root (these are the
	//       "this fact requires the now-disproven root" relations that
	//       triggered the cascade in the first place)
	edgeSet := map[string]*types.ToKEdge{}
	rootIncoming, _ := k.GetIncomingRelations(ctx, root.Id)
	for _, rel := range rootIncoming {
		// Include CONTRADICTS to the root (the challenge edges) and any
		// support edges from descendants we already counted.
		if rel.Relation == types.RelationType_RELATION_TYPE_CONTRADICTS ||
			visited[rel.SourceFactId] {
			edgeKey := rel.SourceFactId + "->" + rel.TargetFactId + "|" + rel.Relation.String()
			edgeSet[edgeKey] = &types.ToKEdge{
				FromFactId: rel.SourceFactId,
				ToFactId:   rel.TargetFactId,
				Relation:   rel.Relation.String(),
				Inference:  rel.Inference.String(),
			}
		}
	}

	// For depth > 1 traversal, also include edges among visited descendants.
	if sel.MaxDepth > 1 {
		for nodeID := range visited {
			if nodeID == root.Id {
				continue
			}
			outgoing, _ := k.GetFactRelations(ctx, nodeID)
			for _, rel := range outgoing {
				if !visited[rel.TargetFactId] {
					continue
				}
				edgeKey := rel.SourceFactId + "->" + rel.TargetFactId + "|" + rel.Relation.String()
				if _, ok := edgeSet[edgeKey]; !ok {
					edgeSet[edgeKey] = &types.ToKEdge{
						FromFactId: rel.SourceFactId,
						ToFactId:   rel.TargetFactId,
						Relation:   rel.Relation.String(),
						Inference:  rel.Inference.String(),
					}
				}
			}
		}
	}

	for _, e := range edgeSet {
		edges = append(edges, e)
	}
	sortToKEdges(edges)

	// Optional: vindication records.
	if sel.IncludeVindications {
		// Pull vindication records for the disproven fact (its slashed
		// minority voters whose unpopular vote turned out right).
		records := k.GetVindicationRecordsForFact(ctx, root.Id)
		for i := range records {
			r := records[i] // local copy
			vindications = append(vindications, &types.ToKVindicationRecord{
				Verifier:     r.Verifier,
				FactId:       r.FactId,
				RefundAmount: r.RefundAmount,
				BonusAmount:  r.BonusAmount,
				VindicatedAt: r.VindicatedAt,
				DisprovenBy:  r.DisprovenBy,
				RoundId:      r.RoundId,
			})
		}
	}

	// Optional: supersession chain.
	if sel.IncludeSupersessions {
		supersessionChain = k.collectSupersessionChain(ctx, root.Id)
	}

	return nodeIDs, edges, cascadeEvents, vindications, supersessionChain, nil
}

func (k Keeper) gatherCascadeRecursive(
	ctx context.Context,
	factID string,
	depth, maxDepth uint32,
	visited map[string]bool,
) error {
	if depth >= maxDepth {
		return nil
	}
	incoming, err := k.GetIncomingRelations(ctx, factID)
	if err != nil {
		return err
	}
	for _, rel := range incoming {
		// Walk descendants via support-bearing relations only — same filter
		// as Plan 1's support-graph walk. The disproval cascade *itself*
		// already happened (or didn't) at level 1; deeper levels are the
		// trainer asking "show me the indirect blast radius."
		switch rel.Relation {
		case types.RelationType_RELATION_TYPE_SUPPORTS,
			types.RelationType_RELATION_TYPE_REQUIRES,
			types.RelationType_RELATION_TYPE_REFINES,
			types.RelationType_RELATION_TYPE_GENERALIZES,
			types.RelationType_RELATION_TYPE_CITES:
		default:
			continue
		}
		if !visited[rel.SourceFactId] {
			visited[rel.SourceFactId] = true
			if err := k.gatherCascadeRecursive(ctx, rel.SourceFactId, depth+1, maxDepth, visited); err != nil {
				return err
			}
		}
	}
	return nil
}