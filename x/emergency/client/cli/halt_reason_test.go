package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/emergency/client/cli"
	"github.com/zerone-chain/zerone/x/emergency/types"
)

func TestProposeHaltDocumentsReservedReasonAndConservativeGate(t *testing.T) {
	cmd := cli.NewProposeHaltCmd()
	require.Equal(t, "propose-halt [reason]", cmd.Use)
	require.Contains(t, cmd.Long, types.HaltReasonResearchFund)
	require.Contains(t, cmd.Long, "unknown evidence")
	require.Contains(t, cmd.Long, "cannot justify rollback")
}
