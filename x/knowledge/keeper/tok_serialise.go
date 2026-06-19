package keeper

import (
	"bytes"
	"encoding/json"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// SerialiseToK_JSONL emits one JSON line per node (kind:"node") and
// per edge (kind:"edge") in the order they appear in the bundle.
// Adjacency-list form: trainers can stream-process without loading
// the full graph into memory.
//
// Format identifier: "jsonl_adjacency_v1". Versioned so future formats
// (e.g., protobuf graph, GraphML) can coexist.
func SerialiseToK_JSONL(b *types.ToKBundle) ([]byte, error) {
	var buf bytes.Buffer
	for _, n := range b.Nodes {
		row := map[string]any{
			"kind": "node",
			"id":   n.Id,
			"fact": n,
		}
		line, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	for _, e := range b.IncludedEdges {
		row := map[string]any{
			"kind":      "edge",
			"from":      e.FromFactId,
			"to":        e.ToFactId,
			"relation":  e.Relation,
			"inference": e.Inference,
		}
		line, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	// ─── TC4: cascade fields ────────────────────────────────────────────
	for _, ev := range b.CascadeEvents {
		row := map[string]any{
			"kind":              "cascade_event",
			"seq":               ev.Seq,
			"disproven_fact_id": ev.DisprovenFactId,
			"descendant":        ev.DescendantFactId,
			"challenge_claim":   ev.ChallengeClaimId,
			"edge_relation":     ev.EdgeRelation,
			"prior_status":      ev.PriorStatus.String(),
			"new_status":        ev.NewStatus.String(),
			"block_height":      ev.BlockHeight,
		}
		line, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	for _, v := range b.Vindications {
		row := map[string]any{
			"kind":          "vindication",
			"fact_id":       v.FactId,
			"verifier":      v.Verifier,
			"refund_amount": v.RefundAmount,
			"bonus_amount":  v.BonusAmount,
			"vindicated_at": v.VindicatedAt,
			"disproven_by":  v.DisprovenBy,
			"round_id":      v.RoundId,
		}
		line, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	for _, t := range b.StatusHistory {
		row := map[string]any{
			"kind":             "transition",
			"fact_id":          t.FactId,
			"seq":              t.Seq,
			"prior_status":     t.PriorStatus.String(),
			"new_status":       t.NewStatus.String(),
			"block_height":     t.BlockHeight,
			"cause_event_type": t.CauseEventType,
			"cause_id":         t.CauseId,
		}
		line, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}
