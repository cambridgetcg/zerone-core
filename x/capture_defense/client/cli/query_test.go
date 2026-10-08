package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/capture_defense/client/cli"
)

func TestQueryIncludesCrossStratumRequirements(t *testing.T) {
	root := cli.NewQueryCmd()
	cmd, _, err := root.Find([]string{"cross-stratum-requirements"})
	require.NoError(t, err)
	require.NotSame(t, root, cmd, "the existing query service must be reachable from the module CLI")
	require.Equal(t, "cross-stratum-requirements", cmd.Name())
}
