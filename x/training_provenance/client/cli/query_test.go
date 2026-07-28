package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetQueryCmdExposesNativeAndInTotoViews(t *testing.T) {
	t.Parallel()

	commandNames := map[string]bool{}
	for _, command := range GetQueryCmd().Commands() {
		commandNames[command.Name()] = true
	}
	require.True(t, commandNames["certificate"])
	require.True(t, commandNames["in-toto-statement"])
}
