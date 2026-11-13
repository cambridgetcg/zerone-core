package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// These tests hold each consensus-repairs-v3 knowledge marker to the same
// module-side contract as the v2 markers, through the v2 marker fixture
// (consensus_repairs_v2_activation_test.go): the key is pinned, native genesis
// selects the marker, the genesis field writes that marker and nothing else,
// genesis JSON carries it exactly with a file written before the field existing
// importing as off, and a malformed marker is never exported as off. A repair's
// own tests can build on the fixture: genesisJSON(t, &off) turns its marker off
// and its Enable turns it on.

var knowledgeV3Markers = []knowledgeV2Marker{
	{
		field: "doctrine_identity_enabled", pinned: 0x94, key: types.DoctrineIdentityEnabledKey,
		enabled: keeper.Keeper.DoctrineIdentityEnabled, enable: keeper.Keeper.EnableDoctrineIdentity,
		genesis: func(gs *types.GenesisState) *bool { return &gs.DoctrineIdentityEnabled },
	},
	{
		field: "dormant_fact_contest_enabled", pinned: 0x95, key: types.DormantFactContestEnabledKey,
		enabled: keeper.Keeper.DormantFactContestEnabled, enable: keeper.Keeper.EnableDormantFactContest,
		genesis: func(gs *types.GenesisState) *bool { return &gs.DormantFactContestEnabled },
	},
	{
		field: "reviewer_balance_param_enabled", pinned: 0x96, key: types.ReviewerBalanceParamEnabledKey,
		enabled: keeper.Keeper.ReviewerBalanceParamEnabled, enable: keeper.Keeper.EnableReviewerBalanceParam,
		genesis:    func(gs *types.GenesisState) *bool { return &gs.ReviewerBalanceParamEnabled },
		dependents: []string{"seat_deposit_enabled"},
	},
	{
		field: "seat_deposit_enabled", pinned: 0x97, key: types.SeatDepositEnabledKey,
		enabled: keeper.Keeper.SeatDepositEnabled, enable: keeper.Keeper.EnableSeatDeposit,
		genesis: func(gs *types.GenesisState) *bool { return &gs.SeatDepositEnabled },
	},
}

// clearKnowledgeV3Dependents turns off, in gs, every marker that requires m.
func clearKnowledgeV3Dependents(gs *types.GenesisState, m knowledgeV2Marker) {
	for _, dependent := range m.dependents {
		for _, row := range knowledgeV3Markers {
			if row.field == dependent {
				*row.genesis(gs) = false
			}
		}
	}
}

func TestKnowledgeV3MarkersNativeGenesisSelectsMarker(t *testing.T) {
	for _, m := range knowledgeV3Markers {
		t.Run(m.field, func(t *testing.T) {
			require.Equal(t, []byte{m.pinned}, m.key)
			gs := types.DefaultGenesis()
			require.True(t, *m.genesis(gs), "a new chain starts with the repair selected")
			require.NoError(t, gs.Validate())
			*m.genesis(gs) = false
			clearKnowledgeV3Dependents(gs, m)
			require.NoError(t, gs.Validate(), "off stays valid")

			f := newKnowledgeV2MarkerFixture(t, m)
			f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
			require.Equal(t, []byte{1}, f.stored())
			require.True(t, f.enabled(t))
		})
	}
}

func TestKnowledgeV3MarkersGenesisFieldWritesOnlyTheMarker(t *testing.T) {
	enabled, disabled := true, false
	for _, m := range knowledgeV3Markers {
		t.Run(m.field, func(t *testing.T) {
			on, off := newKnowledgeV2MarkerFixture(t, m), newKnowledgeV2MarkerFixture(t, m)
			on.initGenesis(t, on.genesisJSON(t, &enabled))
			off.initGenesis(t, off.genesisJSON(t, &disabled))
			want := off.snapshot(t)
			want[string(m.key)] = "\x01"
			require.Equal(t, want, on.snapshot(t), "the genesis field writes the marker and nothing else")
		})
	}
}

// A marker that is neither absent nor {1} is a fault, so ExportGenesis panics
// rather than export it as off.
func TestKnowledgeV3MarkersExportRefusesMalformedMarker(t *testing.T) {
	for _, m := range knowledgeV3Markers {
		t.Run(m.field, func(t *testing.T) {
			f := newKnowledgeV2MarkerFixture(t, m)
			f.initGenesis(t, f.genesisJSON(t, nil))
			for _, bad := range [][]byte{{0}, {2}} {
				t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
					f.ctx.KVStore(f.key).Set(m.key, bad)
					require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) }, "a malformed marker is never exported as off")
				})
			}
		})
	}
}

func TestKnowledgeV3MarkersGenesisRoundTripsMarker(t *testing.T) {
	on, off := true, false
	for _, m := range knowledgeV3Markers {
		for _, tc := range []struct {
			name  string
			field *bool
			want  bool
		}{
			{"on", &on, true},
			{"off", &off, false},
			{"file without the field", nil, false},
		} {
			t.Run(m.field+"/"+tc.name, func(t *testing.T) {
				source := newKnowledgeV2MarkerFixture(t, m)
				source.initGenesis(t, source.genesisJSON(t, tc.field))
				require.Equal(t, tc.want, source.enabled(t))
				if !tc.want {
					require.Nil(t, source.stored(), "off writes no marker")
				}
				exported := source.am.ExportGenesis(source.ctx, source.cdc)
				require.Equal(t, tc.want, source.exported(t, exported))

				imported := newKnowledgeV2MarkerFixture(t, m)
				imported.initGenesis(t, exported)
				require.Equal(t, tc.want, imported.enabled(t))
				require.Equal(t, tc.want, imported.exported(t, imported.am.ExportGenesis(imported.ctx, imported.cdc)))
			})
		}
	}
}
