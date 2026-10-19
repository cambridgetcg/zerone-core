package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestDemandReportJSONRoundTrip(t *testing.T) {
	reports := []*types.DemandReport{
		{Domain: "physics", Subject: "gravity", Queries: 10, Fulfilled: 7, Unfulfilled: 3},
	}
	raw, err := json.Marshal(reports)
	require.NoError(t, err)

	var decoded []*types.DemandReport
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Len(t, decoded, 1)
	require.Equal(t, "physics", decoded[0].Domain)
	require.Equal(t, uint64(3), decoded[0].Unfulfilled)
}

func TestPostConjectureHelpNamesRegisteredQuery(t *testing.T) {
	const prefix = "zeroned q knowledge "
	var advertised []string
	for _, line := range strings.Split(NewPostConjectureCmd().Long, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			advertised = append(advertised, strings.TrimPrefix(line, prefix))
		}
	}
	require.Equal(t, []string{"open-questions"}, advertised)

	queryCmd := GetQueryCmd()
	cmd, args, err := queryCmd.Find(strings.Fields(advertised[0]))
	require.NoError(t, err)
	require.NotSame(t, queryCmd, cmd, "help must name a registered leaf query")
	require.Equal(t, "open-questions", cmd.Name())
	require.Empty(t, args)
	require.NotNil(t, cmd.RunE)
	require.NoError(t, cmd.ValidateArgs(args))
}
