package app

import (
	"math"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestProposalProtoSizeUsesExactCometFraming(t *testing.T) {
	txs := [][]byte{
		make([]byte, 127),
		make([]byte, 128),
	}

	want := cmttypes.ComputeProtoSizeForTxs(cmttypes.Txs{
		cmttypes.Tx(txs[0]),
		cmttypes.Tx(txs[1]),
	})
	got := proposalProtoSize(txs)
	require.Equal(t, want, got)
	require.Greater(t, got, int64(len(txs[0])+len(txs[1])),
		"protobuf tags and length prefixes count toward the proposal limit")
	require.True(t, proposalFitsByteLimit(txs, got))
	require.False(t, proposalFitsByteLimit(txs, got-1))
}

func TestAggregateProposalGasLimitIsExactAndOverflowSafe(t *testing.T) {
	total := uint64(0)
	for range 3 {
		require.False(t, gasWouldExceedLimit(total, TxGasLimit, BlockGasLimit))
		total += TxGasLimit
	}
	require.Equal(t, BlockGasLimit, total)
	require.True(t, gasWouldExceedLimit(total, 1, BlockGasLimit))
	require.True(t, gasWouldExceedLimit(math.MaxUint64-1, 2, math.MaxUint64))
}

func TestVoteExtensionInjectionLayoutIsSingletonAtIndexZero(t *testing.T) {
	vex := append(append([]byte{}, VoteExtInjectionPrefix...), '{', '}')
	regular := []byte{0x01}

	require.NoError(t, validateVoteExtInjectionLayout([][]byte{vex, regular}))
	require.ErrorContains(t, validateVoteExtInjectionLayout([][]byte{regular, vex}), "transaction 0")
	require.ErrorContains(t, validateVoteExtInjectionLayout([][]byte{vex, vex}), "multiple")
}

func TestEffectiveProposalMaxGasHonorsConsensusSentinels(t *testing.T) {
	tests := []struct {
		name string
		max  int64
		want uint64
	}{
		{name: "unlimited still has application ceiling", max: -1, want: BlockGasLimit},
		{name: "zero means zero", max: 0, want: 0},
		{name: "lower consensus ceiling", max: 123_456, want: 123_456},
		{name: "higher consensus ceiling", max: int64(BlockGasLimit) + 1, want: BlockGasLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := sdk.Context{}.WithConsensusParams(cmtproto.ConsensusParams{
				Block: &cmtproto.BlockParams{MaxGas: test.max},
			})
			require.Equal(t, test.want, effectiveProposalMaxGas(ctx))
		})
	}
}
