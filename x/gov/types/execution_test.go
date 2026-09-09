package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/gov/types"
)

func TestGenesisLIPExecutionErrorCompatibility(t *testing.T) {
	for _, stage := range []string{types.StatusDraft, types.StatusVoting, types.StatusPassed, types.StatusFailed, types.StatusWithdrawn} {
		genesis := types.DefaultGenesisState()
		genesis.Lips = []*types.LIP{{Id: "legacy", Stage: stage}}
		require.NoError(t, genesis.Validate(), "empty historical classification remains valid")
		genesis.Lips[0].ExecutionError = "parameter_dispatch_failed"
		if stage == types.StatusFailed {
			require.NoError(t, genesis.Validate())
		} else {
			require.Error(t, genesis.Validate())
		}
	}
	for _, code := range []string{"arbitrary handler error", strings.Repeat("x", 10000), "parameter_dispatch_failed\n"} {
		genesis := types.DefaultGenesisState()
		genesis.Lips = []*types.LIP{{Id: "bad", Stage: types.StatusFailed, ExecutionError: code}}
		require.Error(t, genesis.Validate())
	}
}
