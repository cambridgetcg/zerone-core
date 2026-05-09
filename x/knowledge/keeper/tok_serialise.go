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
	return buf.Bytes(), nil
}
