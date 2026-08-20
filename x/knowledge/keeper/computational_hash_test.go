package keeper_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestComputationalClaimContentHash_BindsEveryCommitmentField(t *testing.T) {
	base := &types.ComputationalCommitment{
		WorkSpecHash: strings.Repeat("1", 64), AcceptanceHash: strings.Repeat("2", 64),
		InputRoot: strings.Repeat("3", 64), EnvironmentRoot: strings.Repeat("4", 64),
		ArtifactRoot: strings.Repeat("5", 64), EvidenceRoot: strings.Repeat("6", 64),
		WorkReceiptHash: strings.Repeat("7", 64),
	}
	want := keeper.ComputeComputationalClaimContentHash("result", "compute", base)
	for i := 0; i < 7; i++ {
		c := *base
		switch i {
		case 0:
			c.WorkSpecHash = strings.Repeat("a", 64)
		case 1:
			c.AcceptanceHash = strings.Repeat("b", 64)
		case 2:
			c.InputRoot = strings.Repeat("c", 64)
		case 3:
			c.EnvironmentRoot = strings.Repeat("d", 64)
		case 4:
			c.ArtifactRoot = strings.Repeat("e", 64)
		case 5:
			c.EvidenceRoot = strings.Repeat("f", 64)
		case 6:
			c.WorkReceiptHash = strings.Repeat("8", 64)
		}
		require.NotEqual(t, want, keeper.ComputeComputationalClaimContentHash("result", "compute", &c), "field %d must affect the fact-ID ancestry", i)
	}
}

func TestComputeClaimContentHash_LegacyRecipeUnchanged(t *testing.T) {
	require.Equal(t,
		"6aa400b6a72276de99b71d2479acf9952ccb7df130df033398398ee511007d13",
		keeper.ComputeClaimContentHash("Water boils at 100C", "physics"))
}
