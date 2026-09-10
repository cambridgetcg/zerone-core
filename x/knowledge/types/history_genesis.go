package types

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// ValidateKnowledgeHistoryGenesis validates supplied records without inventing
// missing historical events. Sequence gaps and counters ahead of records are
// allowed; duplicates, ambiguous keys, unknown wire fields and counters behind
// recorded history are refused before any keeper write.
func ValidateKnowledgeHistoryGenesis(gs *GenesisState) error {
	if gs == nil {
		return fmt.Errorf("nil history genesis")
	}
	const maxEntries = 100000
	const maxBytes = 64 * 1024 * 1024
	seen := map[string]bool{}
	reverse := map[string]bool{}
	counters := map[string]uint64{}
	last := map[string]uint64{}
	n, size := 0, 0
	check := func(key []byte, msg proto.Message, valueSize int) error {
		if len(msg.ProtoReflect().GetUnknown()) != 0 {
			return fmt.Errorf("unknown history protobuf fields")
		}
		if seen[string(key)] {
			return fmt.Errorf("duplicate history key")
		}
		seen[string(key)] = true
		n++
		size += len(key) + valueSize
		if n > maxEntries || size > maxBytes {
			return fmt.Errorf("history genesis exceeds resource bounds")
		}
		return nil
	}
	for _, v := range gs.StatusTransitions {
		if v == nil || v.FactId == "" || v.Seq == 0 {
			return fmt.Errorf("invalid status transition")
		}
		if _, ok := FactStatus_name[int32(v.PriorStatus)]; !ok {
			return fmt.Errorf("unknown prior fact status")
		}
		if _, ok := FactStatus_name[int32(v.NewStatus)]; !ok {
			return fmt.Errorf("unknown new fact status")
		}
		if err := check(StatusTransitionKey(v.FactId, v.Seq), v, proto.Size(v)); err != nil {
			return err
		}
		if v.Seq > last[v.FactId] {
			last[v.FactId] = v.Seq
		}
	}
	for _, v := range gs.CascadeEvents {
		if v == nil || v.DisprovenFactId == "" || v.DescendantFactId == "" || v.Seq == 0 {
			return fmt.Errorf("invalid cascade event")
		}
		if bytes.HasPrefix(CascadeEventByDescendantKey(v.DescendantFactId, v.DisprovenFactId), []byte{0x7f, 0x01}) {
			return fmt.Errorf("cascade collides with reserved migration namespace")
		}
		if _, ok := FactStatus_name[int32(v.PriorStatus)]; !ok {
			return fmt.Errorf("unknown prior cascade status")
		}
		if _, ok := FactStatus_name[int32(v.NewStatus)]; !ok {
			return fmt.Errorf("unknown new cascade status")
		}
		if err := check(CascadeEventKey(v.DisprovenFactId, v.Seq), v, proto.Size(v)); err != nil {
			return err
		}
		reverseKey := CascadeEventByDescendantKey(v.DescendantFactId, v.DisprovenFactId)
		if !reverse[string(reverseKey)] {
			reverse[string(reverseKey)] = true
			n++
			size += len(reverseKey) + 1
		}
	}
	for _, v := range gs.StatusTransitionCounters {
		if v == nil || v.FactId == "" {
			return fmt.Errorf("invalid history counter")
		}
		// Stored counters use canonical uvarints, not the export wrapper.
		var buf [binary.MaxVarintLen64]byte
		if err := check(StatusTransitionSeqKey(v.FactId), v, binary.PutUvarint(buf[:], v.Sequence)); err != nil {
			return err
		}

		counters[v.FactId] = v.Sequence
	}
	if n > maxEntries || size > maxBytes {
		return fmt.Errorf("history genesis exceeds resource bounds")
	}
	for id, seq := range last {
		if counters[id] < seq {
			return fmt.Errorf("history counter for %s absent or behind recorded sequence", id)
		}
	}
	return nil
}
