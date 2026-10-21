package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// FX7 adds only the accounting_state schema; export and import are its fix
// lane's work. This pins what that work relies on: the module's own genesis
// JSON codec keeps the field's presence (absent, present but empty) and every
// inventory value, including a zero epoch and a retained inactive balance,
// and native genesis carries no inventory.
func TestKnowledgeGenesisAccountingSurvivesModuleJSON(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	for _, tc := range []struct {
		name  string
		state *types.KnowledgeGenesisAccounting
	}{
		{"absent", nil},
		{"present and empty", &types.KnowledgeGenesisAccounting{}},
		{"populated", &types.KnowledgeGenesisAccounting{
			AugmentationEscrows: []*types.AugmentationEscrowBalance{
				{BountyId: "active", RemainingUzrn: "4365"},
				{BountyId: "inactive-retained", RemainingUzrn: "1"},
			},
			BootstrapAddressCounts: []*types.BootstrapAddressCount{{Address: "zrn1exactaddressstring", Count: 10}},
			BootstrapEpochCounts:   []*types.BootstrapEpochCount{{Epoch: 0, Count: 3}, {Epoch: 7, Count: 100}},

			TrainingFundNonEscrowBalance: "135",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := types.DefaultGenesis()
			require.Nil(t, gs.AccountingState, "native genesis carries no inventory")
			gs.AccountingState = tc.state
			data, err := cdc.MarshalJSON(gs)
			require.NoError(t, err)
			var restored types.GenesisState
			require.NoError(t, cdc.UnmarshalJSON(data, &restored))
			require.Equal(t, tc.state != nil, restored.AccountingState != nil, "presence survives the module codec")
			if tc.state != nil {
				require.True(t, proto.Equal(tc.state, restored.AccountingState), "every inventory value survives the module codec")
			}
			require.NoError(t, restored.Validate())
		})
	}
}
