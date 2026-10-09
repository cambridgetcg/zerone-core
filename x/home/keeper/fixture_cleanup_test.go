package keeper_test

import (
	"bytes"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetupKeeperCleanup(t *testing.T) {
	pruners := func() int {
		var stacks bytes.Buffer
		require.NoError(t, pprof.Lookup("goroutine").WriteTo(&stacks, 2))
		return strings.Count(stacks.String(), "github.com/cosmos/iavl.(*nodeDB).startPruning(")
	}
	before := pruners()
	for i := 0; i < 5; i++ {
		t.Run("fixture", func(t *testing.T) {
			setupKeeper(t)
		})
	}
	// Subtest cleanups have run. Count IAVL workers, not unrelated goroutines.
	t.Logf("IAVL pruning workers after subtest cleanups: before=%d after=%d", before, pruners())
	require.Eventually(t, func() bool { return pruners() == before }, time.Second, 10*time.Millisecond,
		"IAVL pruning workers remain after five completed fixture cleanups; before=%d", before)
	t.Logf("IAVL pruning workers before=%d after=%d", before, pruners())
}
