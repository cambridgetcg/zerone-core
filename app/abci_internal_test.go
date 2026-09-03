package app

import (
	"math"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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
