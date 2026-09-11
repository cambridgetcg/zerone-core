package types

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
)

// These bounds apply to the complete import/export inventory, counting both
// canonical index copies. They are not per-query or lifetime admission limits.
const (
	MaxFactRelationGenesisEntries = 100000
	MaxFactRelationGenesisBytes   = 64 * 1024 * 1024
)

// ValidateFactRelationRecord checks that a record can be preserved without
// ambiguous keys or discarded metadata. Defined legacy zero values and absent
// attribution remain absent; this does not apply new claim-admission policy to
// historical records or infer scientific validity from their presence.
func ValidateFactRelationRecord(rel *FactRelation) error {
	if rel == nil {
		return fmt.Errorf("nil fact relation")
	}
	for _, id := range []string{rel.SourceFactId, rel.TargetFactId} {
		if id == "" || strings.Contains(id, "/") || !utf8.ValidString(id) {
			return fmt.Errorf("fact relation requires nonempty, unambiguous endpoint IDs")
		}
	}
	if len(rel.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unknown fact relation protobuf fields")
	}
	if _, ok := RelationType_name[int32(rel.Relation)]; !ok {
		return fmt.Errorf("unknown fact relation type")
	}
	if _, ok := InferenceType_name[int32(rel.Inference)]; !ok {
		return fmt.Errorf("unknown fact relation inference")
	}
	if !utf8.ValidString(rel.Creator) || !utf8.ValidString(rel.MethodId) {
		return fmt.Errorf("invalid fact relation text encoding")
	}
	return nil
}

// ValidateFactRelationGenesis checks an explicit inventory before genesis
// writes. Nil is historical absence, not an instruction to infer missing edges.
// A present empty inventory is authoritative, including absence of doctrine
// edges. Embedded Fact relation arrays are preserved as Fact payload only.
func ValidateFactRelationGenesis(gs *GenesisState) error {
	if gs == nil {
		return fmt.Errorf("nil fact relation genesis")
	}
	state := gs.FactRelationState
	if state == nil {
		return nil
	}
	if len(state.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unknown fact relation genesis protobuf fields")
	}
	if len(state.Relations) > MaxFactRelationGenesisEntries/2 {
		return fmt.Errorf("fact relation genesis exceeds entry bound")
	}
	facts := make(map[string]bool, len(gs.Facts))
	for _, fact := range gs.Facts {
		if fact == nil {
			continue
		}
		if facts[fact.Id] {
			return fmt.Errorf("duplicate fact ID in relation genesis: %s", fact.Id)
		}
		facts[fact.Id] = true
	}
	seen := make(map[string]bool, len(state.Relations))
	size := 0
	for _, rel := range state.Relations {
		if err := ValidateFactRelationRecord(rel); err != nil {
			return err
		}
		if !facts[rel.SourceFactId] || !facts[rel.TargetFactId] {
			return fmt.Errorf("fact relation references an absent genesis Fact")
		}
		key := FactRelationKey(rel.SourceFactId, rel.TargetFactId)
		if seen[string(key)] {
			return fmt.Errorf("duplicate ordered fact relation pair")
		}
		seen[string(key)] = true
		// Include both keys and both stored payloads, matching the export census.
		size += len(key) + len(FactRelationReverseKey(rel.TargetFactId, rel.SourceFactId)) + 2*proto.Size(rel)
		if size > MaxFactRelationGenesisBytes {
			return fmt.Errorf("fact relation genesis exceeds byte bound")
		}
	}
	return nil
}
