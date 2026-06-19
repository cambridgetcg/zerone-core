package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Sentinel errors for typed error dispatch in AssembleToKBundle / BundleToK.
var (
	ErrToKRootFactNotFound  = errors.New("root fact not found")
	ErrToKLeafFactNotFound  = errors.New("leaf fact not found")
	ErrToKInconsistentState = errors.New("inconsistent state")
)

// Version constants for provenance fields that have no keeper getter yet.
// GetTraceSchemaVersion and GetTokenizerVersion do not exist on the keeper;
// these constants represent the current deployed versions.
const (
	tokTraceSchemaVersion = "v1"
	tokTokenizerVersion   = "v0"
)

// tokSerialisationFormatJSONL is the canonical serialisation format tag for
// ToK bundles. Task 9 references this constant when writing JSONL payloads.
const tokSerialisationFormatJSONL = "jsonl_adjacency_v1"

// Domain tags for ToK Merkle commitment. Separate tags prevent set-swap
// collisions: a node-ID set cannot produce the same hash as an edge set.
const (
	tokDomainNodes = "TOK_NODES"
	tokDomainEdges = "TOK_EDGES"
	tokDomainRoot  = "TOK_ROOT"

	// V2 domain tags (TC4: the graph carries its disprovals).
	tokDomainCascade      = "TOK_CASCADE"
	tokDomainVindications = "TOK_VINDICATIONS"
	tokDomainTransitions  = "TOK_TRANSITIONS"
	tokDomainRootV2       = "TOK_ROOT_V2"
)

// ComputeToKSnapshotRoot returns a 32-byte Merkle commitment over the
// (sorted node IDs, sorted edges) pair, domain-tagged to prevent
// set-swap collisions. Mirrors the ComputeManifestMerkleRoot pattern
// in training_manifest.go (Wave 7).
//
// Shape:
//
//	sha256( "TOK_ROOT" ||
//	        sha256( "TOK_NODES" || node_0 || node_1 || … ) ||
//	        sha256( "TOK_EDGES" || edge_0_canon || edge_1_canon || … ) )
//
// The root is computed from IDs alone, never from payloads. A trainer
// who has the IDs can re-derive the root without trusting the RPC's
// serialisation. TC2: every view is graph-pinned.
//
// Helpers writeLenString and putUint64 are defined in training_manifest.go
// (same package); sortToKEdges is defined in tok_selector.go (same package).
func ComputeToKSnapshotRoot(nodeIDs []string, edges []*types.ToKEdge) []byte {
	// Defensive copies so callers' slices are not mutated.
	sortedNodes := append([]string{}, nodeIDs...)
	sort.Strings(sortedNodes)
	sortedEdges := append([]*types.ToKEdge{}, edges...)
	sortToKEdges(sortedEdges)

	nodesH := tokDomainHash(tokDomainNodes, func(h interface{ Write([]byte) (int, error) }) {
		for _, id := range sortedNodes {
			writeLenString(h, id)
		}
	})

	edgesH := tokDomainHash(tokDomainEdges, func(h interface{ Write([]byte) (int, error) }) {
		for _, e := range sortedEdges {
			// Length-prefix each field individually to prevent field-boundary
			// collisions. A pipe-concatenated canon would make
			// {FromFactId:"a|b", ToFactId:"c"} indistinguishable from
			// {FromFactId:"a", ToFactId:"b|c"}. writeLenString encodes
			// each field as (uint64-length || bytes), so fields with embedded
			// separators cannot collide with fields at a boundary.
			writeLenString(h, e.FromFactId)
			writeLenString(h, e.ToFactId)
			writeLenString(h, e.Relation)
			writeLenString(h, e.Inference)
		}
	})

	// Root: sha256( len("TOK_ROOT") || "TOK_ROOT" || nodesH || edgesH ).
	// nodesH and edgesH are fixed-length (32 bytes each) so no length prefix
	// is needed for them — fixed-width fields cannot produce ambiguity.
	final := sha256.New()
	writeLenString(final, tokDomainRoot)
	_, _ = final.Write(nodesH)
	_, _ = final.Write(edgesH)
	return final.Sum(nil)
}

// tokDomainHash hashes a domain tag followed by whatever the write callback
// appends. Equivalent to the domainHash helper described in the task plan.
// writeLenString is the length-prefixed write helper from training_manifest.go.
func tokDomainHash(domain string, write func(interface{ Write([]byte) (int, error) })) []byte {
	h := sha256.New()
	writeLenString(h, domain)
	write(h)
	return h.Sum(nil)
}

// ComputeToKSnapshotRootV2 returns a 32-byte Merkle commitment over the full
// (nodes, edges, cascade_events, vindications, transitions) bundle. Each
// component is independently domain-tagged; the V2 root tag distinguishes
// V2 bundles from V1 bundles even when cascade fields are empty.
//
// TC4: the disproval-graph is bundled with the support-graph under one root.
// A trainer who has the IDs + cascade event canon can re-derive the root
// without trusting the RPC.
func ComputeToKSnapshotRootV2(
	nodeIDs []string,
	edges []*types.ToKEdge,
	cascadeEvents []*types.CascadeEvent,
	vindications []*types.ToKVindicationRecord,
	transitions []*types.StatusTransition,
) []byte {
	sortedNodes := append([]string{}, nodeIDs...)
	sort.Strings(sortedNodes)
	sortedEdges := append([]*types.ToKEdge{}, edges...)
	sortToKEdges(sortedEdges)
	sortedCascade := sortCascadeEvents(cascadeEvents)
	sortedVind := sortVindications(vindications)
	sortedTrans := sortStatusTransitions(transitions)

	nodesH := tokDomainHash(tokDomainNodes, func(h interface{ Write([]byte) (int, error) }) {
		for _, id := range sortedNodes {
			writeLenString(h, id)
		}
	})
	edgesH := tokDomainHash(tokDomainEdges, func(h interface{ Write([]byte) (int, error) }) {
		for _, e := range sortedEdges {
			writeLenString(h, e.FromFactId)
			writeLenString(h, e.ToFactId)
			writeLenString(h, e.Relation)
			writeLenString(h, e.Inference)
		}
	})
	cascadeH := tokDomainHash(tokDomainCascade, func(h interface{ Write([]byte) (int, error) }) {
		for _, ev := range sortedCascade {
			writeLenString(h, ev.DisprovenFactId)
			writeLenString(h, ev.DescendantFactId)
			writeLenString(h, ev.ChallengeClaimId)
			writeLenString(h, ev.EdgeRelation)
			writeLenUint64(h, ev.Seq)
			writeLenUint64(h, ev.BlockHeight)
			writeLenUint64(h, uint64(ev.PriorStatus))
			writeLenUint64(h, uint64(ev.NewStatus))
		}
	})
	vindH := tokDomainHash(tokDomainVindications, func(h interface{ Write([]byte) (int, error) }) {
		for _, v := range sortedVind {
			writeLenString(h, v.Verifier)
			writeLenString(h, v.FactId)
			writeLenString(h, v.RefundAmount)
			writeLenString(h, v.BonusAmount)
			writeLenUint64(h, v.VindicatedAt)
			writeLenString(h, v.DisprovenBy)
			writeLenString(h, v.RoundId)
		}
	})
	transH := tokDomainHash(tokDomainTransitions, func(h interface{ Write([]byte) (int, error) }) {
		for _, t := range sortedTrans {
			writeLenString(h, t.FactId)
			writeLenUint64(h, t.Seq)
			writeLenUint64(h, uint64(t.PriorStatus))
			writeLenUint64(h, uint64(t.NewStatus))
			writeLenUint64(h, t.BlockHeight)
			writeLenString(h, t.CauseEventType)
			writeLenString(h, t.CauseId)
		}
	})

	final := sha256.New()
	writeLenString(final, tokDomainRootV2)
	_, _ = final.Write(nodesH)
	_, _ = final.Write(edgesH)
	_, _ = final.Write(cascadeH)
	_, _ = final.Write(vindH)
	_, _ = final.Write(transH)
	return final.Sum(nil)
}

// writeLenUint64 writes 8-byte big-endian for fixed-width safety.
func writeLenUint64(h interface{ Write([]byte) (int, error) }, v uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	_, _ = h.Write(buf[:])
}

func sortCascadeEvents(events []*types.CascadeEvent) []*types.CascadeEvent {
	out := append([]*types.CascadeEvent{}, events...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].DisprovenFactId != out[j].DisprovenFactId {
			return out[i].DisprovenFactId < out[j].DisprovenFactId
		}
		return out[i].Seq < out[j].Seq
	})
	return out
}

func sortVindications(records []*types.ToKVindicationRecord) []*types.ToKVindicationRecord {
	out := append([]*types.ToKVindicationRecord{}, records...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].FactId != out[j].FactId {
			return out[i].FactId < out[j].FactId
		}
		return out[i].Verifier < out[j].Verifier
	})
	return out
}

func sortStatusTransitions(transitions []*types.StatusTransition) []*types.StatusTransition {
	out := append([]*types.StatusTransition{}, transitions...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].FactId != out[j].FactId {
			return out[i].FactId < out[j].FactId
		}
		return out[i].Seq < out[j].Seq
	})
	return out
}

// AssembleToKBundle is the headline ToK extraction primitive. It validates
// the selector, applies caps, gathers IDs, computes the snapshot root,
// materialises payloads, and returns a complete bundle. TC1 (graph is
// the headline) is bound by exposing this through gRPC + CLI as the
// trainer-facing default.
//
// atBlockHeight: pass 0 to use the current block. Non-zero values are
// reserved for historical replay (planned for a future wave). Until the
// keeper supports historical state queries, callers MUST pass 0 — passing
// any other value will record a SnapshotBlock metadata that does NOT
// match the actual state the bundle was built from, breaking TC2's
// re-derivability guarantee.
func (k Keeper) AssembleToKBundle(
	ctx context.Context,
	sel *types.ToKSelector,
	atBlockHeight uint64,
) (*types.ToKBundle, error) {
	capped, err := ValidateAndCapToKSelector(sel)
	if err != nil {
		return nil, err
	}
	nodeIDs, edges, err := k.SelectToKIds(ctx, capped)
	if err != nil {
		return nil, err
	}
	root := ComputeToKSnapshotRoot(nodeIDs, edges)

	// Materialise node payloads.
	var nodes []*types.Fact
	for _, id := range nodeIDs {
		f, ok := k.GetFact(ctx, id)
		if !ok {
			return nil, fmt.Errorf("%w: selected fact %s not found", ErrToKInconsistentState, id)
		}
		nodes = append(nodes, f)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if atBlockHeight == 0 {
		atBlockHeight = uint64(sdkCtx.BlockHeight())
	}

	bundle := &types.ToKBundle{
		SnapshotBlock:       atBlockHeight,
		SnapshotRoot:        root,
		IncludedNodeIds:     nodeIDs,
		IncludedEdges:       edges,
		Nodes:               nodes,
		SerialisationFormat: tokSerialisationFormatJSONL,
		Provenance: &types.ToKBundleProvenance{
			ChainId: sdkCtx.ChainID(),
			// GetTraceSchemaVersion / GetTokenizerVersion do not exist on the
			// keeper; use the deployed-version constants until a param or
			// getter is added.
			TraceSchemaVersion:            tokTraceSchemaVersion,
			CanonicalSerialisationVersion: "v1",
			TokenizerVersion:              tokTokenizerVersion,
			SelectorUsed:                  capped,
			CapMaxDepth:                   ToKMaxDepthCap,
			CapMaxPaths:                   ToKMaxPathsCap,
			CapLimit:                      ToKFrontierCap,
		},
	}

	payload, err := SerialiseToK_JSONL(bundle)
	if err != nil {
		return nil, err
	}
	bundle.SerialisedPayload = payload

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeToKBundleExtracted,
		sdk.NewAttribute(AttrToKCommitment, "TC0,TC1,TC5"),
		sdk.NewAttribute(AttrToKSelectorKind, selectorKind(capped)),
		sdk.NewAttribute(AttrToKBundleSize, fmt.Sprintf("%d", len(nodeIDs))),
		sdk.NewAttribute(AttrToKSnapshotBlock, fmt.Sprintf("%d", atBlockHeight)),
	))
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeToKSnapshotRootPinned,
		sdk.NewAttribute(AttrToKCommitment, "TC0,TC2"),
		sdk.NewAttribute(AttrToKSnapshotRoot, hex.EncodeToString(root)),
	))

	return bundle, nil
}

// SelectToKIds dispatches on the validated selector variant.
func (k Keeper) SelectToKIds(
	ctx context.Context,
	sel *types.ToKSelector,
) (nodeIDs []string, edges []*types.ToKEdge, err error) {
	switch v := sel.Variant.(type) {
	case *types.ToKSelector_RootedSubtree:
		return k.GatherRootedSubtree(ctx, v.RootedSubtree)
	case *types.ToKSelector_AncestorCone:
		return k.GatherAncestorCone(ctx, v.AncestorCone)
	case *types.ToKSelector_Frontier:
		return k.GatherFrontier(ctx, v.Frontier)
	default:
		return nil, nil, fmt.Errorf("selector variant not recognised: %T", sel.Variant)
	}
}

// selectorKind returns a human-readable tag for the selector variant, used as
// the selector_kind event attribute on tok_bundle_extracted events.
func selectorKind(s *types.ToKSelector) string {
	switch s.Variant.(type) {
	case *types.ToKSelector_RootedSubtree:
		return "rooted_subtree"
	case *types.ToKSelector_AncestorCone:
		return "ancestor_cone"
	case *types.ToKSelector_Frontier:
		return "frontier"
	default:
		return "unknown"
	}
}
