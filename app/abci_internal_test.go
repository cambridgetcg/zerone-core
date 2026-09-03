package app

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddProposalTxGasRejectsLimitAndIntegerOverflow(t *testing.T) {
	total, fits := addProposalTxGas(60, 40, 100)
	require.True(t, fits)
	require.Equal(t, uint64(100), total)

	for name, test := range map[string]struct {
		total uint64
		txGas uint64
		max   uint64
	}{
		"over block limit":       {total: 60, txGas: 41, max: 100},
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
