package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateMemoryCIDRejectsInvalidCIDBeforeClientSetup(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]string{
		"malformed": "not-a-cid",
		"cidv0":     "QmYwAPJzv5CZsnAzt8auVZRnZrmPaUe2rLCsShjUEiB2yR",
		"too long":  "b" + strings.Repeat("a", 256),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := NewUpdateMemoryCIDCmd()
			cmd.SetArgs([]string{"home-1", value})
			err := cmd.Execute()
			require.ErrorContains(t, err, "invalid memory CID")
		})
	}
}
