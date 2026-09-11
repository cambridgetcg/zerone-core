package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestFactRelationGenesisPresenceSurvivesJSON(t *testing.T) {
	for _, present := range []bool{false, true} {
		gs := types.DefaultGenesis()
		if present {
			gs.FactRelationState = &types.FactRelationGenesis{}
		}
		data, err := protojson.Marshal(gs)
		require.NoError(t, err)
		var restored types.GenesisState
		require.NoError(t, protojson.Unmarshal(data, &restored))
		require.Equal(t, present, restored.FactRelationState != nil)
		require.NoError(t, restored.Validate())
	}
}

func TestFactRelationGenesisRejectsAmbiguousOrUnrepresentableInventory(t *testing.T) {
	for _, kind := range []string{"nil-row", "empty-source", "slash-source", "missing-target", "duplicate-pair", "duplicate-fact", "unknown-relation", "unknown-inference", "unknown-row-field", "unknown-inventory-field", "invalid-text", "entry-bound", "byte-bound"} {
		t.Run(kind, func(t *testing.T) {
			rel := &types.FactRelation{SourceFactId: "a", TargetFactId: "b", Relation: types.RelationType_RELATION_TYPE_REQUIRES}
			gs := types.DefaultGenesis()
			gs.Facts = []*types.Fact{{Id: "a"}, {Id: "b"}}
			gs.FactRelationState = &types.FactRelationGenesis{Relations: []*types.FactRelation{rel}}
			switch kind {
			case "nil-row":
				gs.FactRelationState.Relations[0] = nil
			case "empty-source":
				rel.SourceFactId = ""
			case "slash-source":
				rel.SourceFactId = "a/b"
				gs.Facts = append(gs.Facts, &types.Fact{Id: "a/b"})
			case "missing-target":
				rel.TargetFactId = "missing"
			case "duplicate-pair":
				second := proto.Clone(rel).(*types.FactRelation)
				second.Relation = types.RelationType_RELATION_TYPE_CITES
				gs.FactRelationState.Relations = append(gs.FactRelationState.Relations, second)
			case "duplicate-fact":
				gs.Facts = append(gs.Facts, &types.Fact{Id: "a", Content: "different"})
			case "unknown-relation":
				rel.Relation = 999
			case "unknown-inference":
				rel.Inference = -1
			case "unknown-row-field":
				rel.ProtoReflect().SetUnknown([]byte{0x98, 0x06, 0x01})
			case "unknown-inventory-field":
				gs.FactRelationState.ProtoReflect().SetUnknown([]byte{0x98, 0x06, 0x01})
			case "invalid-text":
				rel.Creator = string([]byte{0xff})
			case "entry-bound":
				gs.FactRelationState.Relations = make([]*types.FactRelation, types.MaxFactRelationGenesisEntries/2+1)
			case "byte-bound":
				rel.MethodId = string(make([]byte, types.MaxFactRelationGenesisBytes/2))
			}
			require.Error(t, types.ValidateFactRelationGenesis(gs))
			require.Error(t, gs.Validate())
		})
	}
}

func TestFactRelationGenesisPreservesHistoricalMetadataAndCycles(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.Facts = []*types.Fact{{Id: "a"}, {Id: "b"}}
	gs.FactRelationState = &types.FactRelationGenesis{Relations: []*types.FactRelation{
		{SourceFactId: "a", TargetFactId: "b"}, // Defined zero values remain unknown historical metadata.
		{SourceFactId: "b", TargetFactId: "a", Relation: types.RelationType_RELATION_TYPE_CITES, Creator: "historical-attribution", InferenceStrengthBps: 1_000_001},
		{SourceFactId: "a", TargetFactId: "a", Relation: types.RelationType_RELATION_TYPE_SUPPORTS},
	}}
	require.NoError(t, gs.Validate(), "genesis preservation does not impose new claim-admission policy")
	data, err := protojson.Marshal(gs)
	require.NoError(t, err)
	var restored types.GenesisState
	require.NoError(t, protojson.Unmarshal(data, &restored))
	require.True(t, proto.Equal(gs.FactRelationState, restored.FactRelationState))
}
