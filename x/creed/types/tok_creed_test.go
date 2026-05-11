package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/creed/types"
)

func TestCanonicalToKCommitments_Count(t *testing.T) {
	require.Len(t, types.CanonicalToKCommitments, 6,
		"ToK ships TC1-TC6; later additions go through full creed-amendment gov")
}

func TestCanonicalToKCommitments_NumberingDense(t *testing.T) {
	expected := []string{"TC1", "TC2", "TC3", "TC4", "TC5", "TC6"}
	for i, c := range types.CanonicalToKCommitments {
		require.Equal(t, expected[i], c.Number,
			"TC commitments must be dense and ordered TC1..TC6")
	}
}

func TestCanonicalToKCommitments_NamesNonEmpty(t *testing.T) {
	for _, c := range types.CanonicalToKCommitments {
		require.NotEmpty(t, c.Name, "commitment %s must have a non-empty name", c.Number)
	}
}

func TestToKCommitmentDomain_Stable(t *testing.T) {
	require.Equal(t, "doctrine_tok", types.ToKCommitmentDomain,
		"domain name is doctrinally fixed")
}
