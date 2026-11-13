package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

type holdQueryRPC struct {
	client.CometRPC
	t           *testing.T
	values      map[string][]byte
	heights     []int64
	wrongHeight bool
}

func (q *holdQueryRPC) ABCIQueryWithOptions(_ context.Context, path string, key cmtbytes.HexBytes, opts rpcclient.ABCIQueryOptions) (*coretypes.ResultABCIQuery, error) {
	require.Equal(q.t, "/store/knowledge/key", path)
	q.heights = append(q.heights, opts.Height)
	height := int64(42)
	if q.wrongHeight && len(q.heights) > 1 {
		height++
	}
	return &coretypes.ResultABCIQuery{Response: abci.ResponseQuery{Height: height, Value: q.values[string(key)]}}, nil
}

func newHoldQueryRPC(t *testing.T, enabled bool, votes []types.AugmentationVerdict) *holdQueryRPC {
	t.Helper()
	marshal := func(m proto.Message) []byte { bz, err := proto.Marshal(m); require.NoError(t, err); return bz }
	params := types.DefaultParams()
	params.ReformulationMinPanelVotes = 2
	q := &holdQueryRPC{t: t, values: map[string][]byte{
		string(types.AugmentationKey("retained")):           marshal(&types.Augmentation{Id: "retained", BountyId: "bounty", VerdictVoters: []string{"a", "b"}, VerdictVotes: votes, VerdictVoteStakes: []uint64{100, 100}}),
		string(types.ParamsKey):                             marshal(&params),
		string(types.TrainingFundEscrowLockedKey("bounty")): []byte("1500"),
	}}
	if enabled {
		q.values[keeper.ReformulationVindicationOrderingEnabledStoreKey] = []byte{1}
	}
	return q
}

func TestReformulationHoldCLI(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, invalid := range []bool{false, true} {
			t.Run(fmt.Sprintf("enabled=%t/invalid=%t", enabled, invalid), func(t *testing.T) {
				votes := []types.AugmentationVerdict{1, 4}
				reason := "tied_top_weight"
				if invalid {
					votes[0] = 7
					reason = "invalid_retained_vote"
				}
				q := newHoldQueryRPC(t, enabled, votes)
				cmd := GetQueryCmd()
				cmd.SetContext(context.Background())
				var output bytes.Buffer
				cmd.SetOut(&output)
				require.NoError(t, client.SetCmdClientContext(cmd, client.Context{}.WithClient(q)))
				cmd.SetArgs([]string{"reformulation-hold", "retained", "--output=json"})
				require.NoError(t, cmd.Execute())
				var got reformulationHoldResponse
				require.NoError(t, json.Unmarshal(output.Bytes(), &got))
				require.EqualValues(t, 42, got.SnapshotHeight)
				require.True(t, got.Found)
				require.Equal(t, enabled, got.Enabled)
				require.Equal(t, enabled, got.Held)
				require.Equal(t, reason, got.ProspectiveReason)
				if enabled {
					require.Equal(t, reason, got.Reason)
				} else {
					require.Empty(t, got.Reason)
				}
				require.Equal(t, "1500", got.BountyEscrowLocked)
				require.Equal(t, []int64{0, 42, 42, 42}, q.heights)
			})
		}
	}
}

func TestReformulationHoldQueryRefusesBadState(t *testing.T) {
	for _, kind := range []string{"marker", "empty-marker", "height", "params", "augmentation", "identity", "escrow"} {
		t.Run(kind, func(t *testing.T) {
			q := newHoldQueryRPC(t, true, []types.AugmentationVerdict{1, 4})
			switch kind {
			case "marker":
				q.values[keeper.ReformulationVindicationOrderingEnabledStoreKey] = []byte{0}
			case "empty-marker":
				q.values[keeper.ReformulationVindicationOrderingEnabledStoreKey] = []byte{}
			case "height":
				q.wrongHeight = true
			case "params":
				delete(q.values, string(types.ParamsKey))
			case "augmentation":
				q.values[string(types.AugmentationKey("retained"))] = []byte{0xff}
			case "identity":
				q.values[string(types.AugmentationKey("retained"))] = []byte{0x0a, 1, 'x'}
			case "escrow":
				q.values[string(types.TrainingFundEscrowLockedKey("bounty"))] = []byte("-1")
			}
			_, err := queryReformulationHold(client.Context{}.WithClient(q), "retained")
			require.Error(t, err)
		})
	}
	q := newHoldQueryRPC(t, true, []types.AugmentationVerdict{1, 4})
	got, err := queryReformulationHold(client.Context{}.WithClient(q).WithHeight(42), "missing")
	require.NoError(t, err)
	require.False(t, got.Found)
	require.False(t, got.Held)
	require.Equal(t, []int64{42, 42}, q.heights)
}
