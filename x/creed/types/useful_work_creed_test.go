package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/creed/types"
)

func TestUsefulWorkCommitment_IsIndivisible(t *testing.T) {
	require.Equal(t, "UW", types.UsefulWorkCommitment,
		"UW commitment identifier must not change once shipped")
	require.Equal(t, "ZERONE is recursive", types.UsefulWorkStatement,
		"UW statement is doctrinally fixed")
}

func TestCanonicalUsefulWorkMechanisms_Count(t *testing.T) {
	require.Len(t, types.CanonicalUsefulWorkMechanisms, 7,
		"Phase 0 ships M1-M7; later phases add via M3 governance gate")
}

func TestCanonicalUsefulWorkMechanisms_NumberingDense(t *testing.T) {
	for i, m := range types.CanonicalUsefulWorkMechanisms {
		require.Equal(t, uint32(i+1), m.Number,
			"mechanism numbering must be dense and monotonic; index %d must hold M%d", i, i+1)
	}
}

func TestCanonicalUsefulWorkMechanisms_NamesNonEmpty(t *testing.T) {
	for _, m := range types.CanonicalUsefulWorkMechanisms {
		require.NotEmpty(t, m.Name, "mechanism M%d must have a non-empty name", m.Number)
	}
}

func TestCanonicalRecursiveAxes_SixAndOrdered(t *testing.T) {
	require.Equal(t, []string{
		"substrate",
		"verification",
		"classification",
		"attribution",
		"tooling",
		"interface",
	}, types.CanonicalRecursiveAxes,
		"the six recursive axes are doctrinally fixed in this exact order")
}
