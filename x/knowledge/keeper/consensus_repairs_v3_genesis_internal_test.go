package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// v3GenesisMarker binds one consensus-repairs-v3 knowledge marker to its store
// key and genesis field.
type v3GenesisMarker struct {
	name  string
	key   []byte
	field func(*types.GenesisState) *bool
}

var v3GenesisMarkers = []v3GenesisMarker{
	{"doctrine_identity_enabled", types.DoctrineIdentityEnabledKey,
		func(gs *types.GenesisState) *bool { return &gs.DoctrineIdentityEnabled }},
	{"dormant_fact_contest_enabled", types.DormantFactContestEnabledKey,
		func(gs *types.GenesisState) *bool { return &gs.DormantFactContestEnabled }},
	{"reviewer_balance_param_enabled", types.ReviewerBalanceParamEnabledKey,
		func(gs *types.GenesisState) *bool { return &gs.ReviewerBalanceParamEnabled }},
}

// initGenesisV3Markers and exportGenesisV3Markers carry every
// consensus-repairs-v3 marker outside the pinned legacy genesis bodies. On both
// InitGenesis paths (a file without an accounting inventory, and a file with
// one), selecting one marker writes exactly that marker and exports exactly
// that field, so a wrapper that skips, swaps or merges markers fails.
// Selecting none writes none.
func TestConsensusRepairsV3MarkersFollowBothImportPaths(t *testing.T) {
	for _, accounting := range []bool{false, true} {
		for selected := -1; selected < len(v3GenesisMarkers); selected++ {
			name := "none"
			if selected >= 0 {
				name = v3GenesisMarkers[selected].name
			}
			t.Run(fmt.Sprintf("accounting=%t/%s", accounting, name), func(t *testing.T) {
				f := setupRelationGenesis(t, nil)
				f.k.bankKeeper = accountingZeroBank{}
				gs := types.DefaultGenesis()
				if accounting {
					gs = accountingEmptyGenesis()
				}
				for i, m := range v3GenesisMarkers {
					*m.field(gs) = i == selected
				}
				require.NoError(t, f.k.InitGenesis(f.ctx, gs))
				exported := f.k.ExportGenesis(f.ctx)
				for i, m := range v3GenesisMarkers {
					stored := f.ctx.KVStore(f.keys[0]).Get(m.key)
					if i == selected {
						require.Equal(t, []byte{1}, stored, m.name)
					} else {
						require.Nil(t, stored, "%s: off writes no marker", m.name)
					}
					require.Equal(t, i == selected, *m.field(exported), m.name)
				}
			})
		}
	}
}

// A malformed consensus-repairs-v3 marker, whichever it is, panics on export
// with the activation.ErrMarkerFault chain.
func TestConsensusRepairsV3MarkerFaultRefusesExport(t *testing.T) {
	for _, m := range v3GenesisMarkers {
		t.Run(m.name, func(t *testing.T) {
			f := setupRelationGenesis(t, types.DefaultGenesis())
			f.ctx.KVStore(f.keys[0]).Set(m.key, []byte{2})
			var recovered any
			func() { defer func() { recovered = recover() }(); f.k.ExportGenesis(f.ctx) }()
			err, ok := recovered.(error)
			require.True(t, ok, "export panics with an error, got %v", recovered)
			require.ErrorIs(t, err, activation.ErrMarkerFault)
		})
	}
}
