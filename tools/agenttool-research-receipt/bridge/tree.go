package bridge

import (
	"encoding/json"
	"fmt"
)

const maxTreeBytes = 256 << 10

type parsedTree struct {
	DocumentDigest string
	NodeDigest     string
}

func parseTree(data []byte) (parsedTree, error) {
	if len(data) == 0 {
		return parsedTree{}, fmt.Errorf("tree document is empty")
	}
	if len(data) > maxTreeBytes {
		return parsedTree{}, fmt.Errorf("tree document exceeds %d-byte limit", maxTreeBytes)
	}
	documentDigest := digestBytes(data)
	if documentDigest != TreeRawDigest {
		return parsedTree{}, fmt.Errorf("tree raw digest mismatch: expected %s, got %s", TreeRawDigest, documentDigest)
	}

	// The exact byte digest authenticates the complete reviewed local artifact
	// before this deliberately minimal structural decode.
	var document struct {
		Schema          string            `json:"schema"`
		Authoritative   bool              `json:"authoritative"`
		NetworkObserved bool              `json:"networkObserved"`
		RewardBearing   bool              `json:"rewardBearing"`
		Nodes           []json.RawMessage `json:"nodes"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return parsedTree{}, fmt.Errorf("decode exact Tree bytes: %w", err)
	}
	if document.Schema != TreeSchema {
		return parsedTree{}, fmt.Errorf("tree schema must be %q", TreeSchema)
	}
	if document.Authoritative || document.NetworkObserved || document.RewardBearing {
		return parsedTree{}, fmt.Errorf("tree must remain non-authoritative, non-network-observed, and non-reward-bearing")
	}

	var targetRaw json.RawMessage
	for _, raw := range document.Nodes {
		var header struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return parsedTree{}, fmt.Errorf("decode Tree node header: %w", err)
		}
		if header.ID == TargetNodeID {
			if targetRaw != nil {
				return parsedTree{}, fmt.Errorf("tree contains target node %q more than once", TargetNodeID)
			}
			targetRaw = raw
		}
	}
	if targetRaw == nil {
		return parsedTree{}, fmt.Errorf("tree does not contain target node %q", TargetNodeID)
	}
	var target struct {
		ID                 string `json:"id"`
		AttainmentEvidence string `json:"attainmentEvidence"`
		RewardEligibility  string `json:"rewardEligibility"`
	}
	if err := json.Unmarshal(targetRaw, &target); err != nil {
		return parsedTree{}, fmt.Errorf("decode target node: %w", err)
	}
	if target.ID != TargetNodeID || target.AttainmentEvidence != "E2" || target.RewardEligibility != "qualification-only" {
		return parsedTree{}, fmt.Errorf("target node contract drift")
	}
	canonical, err := canonicalRaw(targetRaw)
	if err != nil {
		return parsedTree{}, fmt.Errorf("canonicalize target node: %w", err)
	}
	nodeDigest := digestBytes(canonical)
	if nodeDigest != TargetNodeDigest {
		return parsedTree{}, fmt.Errorf("target node digest mismatch: expected %s, got %s", TargetNodeDigest, nodeDigest)
	}
	return parsedTree{DocumentDigest: documentDigest, NodeDigest: nodeDigest}, nil
}
