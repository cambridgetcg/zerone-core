package app

import (
	"bytes"
	"context"
	"math"
	"math/rand"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"
)

func TestConsensusBlockGasLimit(t *testing.T) {
	tests := map[string]struct {
		block   *cmtproto.BlockParams
		limit   uint64
		limited bool
	}{
		"nil block params": {block: nil, limit: 0, limited: false},
		"unlimited":        {block: &cmtproto.BlockParams{MaxGas: -1}, limit: 0, limited: false},
		"zero":             {block: &cmtproto.BlockParams{MaxGas: 0}, limit: 0, limited: true},
		"positive":         {block: &cmtproto.BlockParams{MaxGas: 123}, limit: 123, limited: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := sdk.Context{}.WithConsensusParams(cmtproto.ConsensusParams{Block: test.block})
			limit, limited := consensusBlockGasLimit(ctx)
			require.Equal(t, test.limit, limit)
			require.Equal(t, test.limited, limited)
		})
	}
}

func TestAddProposalTxGasRejectsLimitAndIntegerOverflow(t *testing.T) {
	total, fits := addProposalTxGas(60, 40, 100)
	require.True(t, fits)
	require.Equal(t, uint64(100), total)
	zero, fits := addProposalTxGas(0, 0, 0)
	require.True(t, fits)
	require.Zero(t, zero)

	for name, test := range map[string]struct {
		total uint64
		txGas uint64
		max   uint64
	}{
		"over block limit":       {total: 60, txGas: 41, max: 100},
		"positive gas at zero":   {total: 0, txGas: 1, max: 0},
		"overflow declaration":   {total: 1, txGas: math.MaxUint64, max: 100},
		"invalid existing total": {total: 101, txGas: 0, max: 100},
	} {
		t.Run(name, func(t *testing.T) {
			unchanged, ok := addProposalTxGas(test.total, test.txGas, test.max)
			require.False(t, ok)
			require.Equal(t, test.total, unchanged)
		})
	}
}

// gasOnlyTx declares gas for the SDK selector without carrying messages.
type gasOnlyTx struct{ gas uint64 }

func (tx gasOnlyTx) GetMsgs() []sdk.Msg                    { return nil }
func (tx gasOnlyTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }
func (tx gasOnlyTx) GetGas() uint64                        { return tx.gas }

// PrepareProposal replays a candidate only when proposalTxFits says the SDK
// default selector will take it, so the two must agree exactly.
func TestProposalTxFitsMirrorsDefaultSelector(t *testing.T) {
	random := rand.New(rand.NewSource(20261102))
	outcomes := map[bool]int{}
	for trial := 0; trial < 200; trial++ {
		maxTxBytes := uint64(random.Intn(4_000))
		maxBlockGas := uint64(0)
		if random.Intn(3) > 0 {
			maxBlockGas = uint64(random.Intn(2_000))
		}
		selector := baseapp.NewDefaultTxSelector()
		var selectedBytes, selectedGas uint64
		for candidate := 0; candidate < 40; candidate++ {
			txBytes := bytes.Repeat([]byte{byte(candidate)}, 1+random.Intn(400))
			txGas := uint64(random.Intn(600))
			size := uint64(cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{txBytes}))
			want := proposalTxFits(selectedBytes, selectedGas, size, txGas, maxTxBytes, maxBlockGas)

			before := len(selector.SelectedTxs(context.Background()))
			full := selector.SelectTxForProposal(context.Background(), maxTxBytes, maxBlockGas, gasOnlyTx{gas: txGas}, txBytes)
			added := len(selector.SelectedTxs(context.Background())) > before
			require.Equalf(t, want, added, "trial %d candidate %d", trial, candidate)
			outcomes[added]++
			if added {
				selectedBytes += size
				if maxBlockGas > 0 {
					selectedGas += txGas
				}
			}
			if full {
				break
			}
		}
	}
	require.Positive(t, outcomes[true], "fixture must exercise selected candidates")
	require.Positive(t, outcomes[false], "fixture must exercise refused candidates")
}
