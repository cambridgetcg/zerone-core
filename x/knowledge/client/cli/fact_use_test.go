package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func (s *capturingKnowledgeQueryServer) FactUseReceipt(_ context.Context, req *types.QueryFactUseReceiptRequest) (*types.QueryFactUseReceiptResponse, error) {
	return nil, s.capture("FactUseReceipt", req)
}

func TestFactUseCLIRegistrationAndDisclosure(t *testing.T) {
	for _, name := range []string{"report-fact-use", "rate-fact"} {
		cmd, _, err := GetTxCmd().Find([]string{name})
		require.NoError(t, err)
		require.Equal(t, name, cmd.Name())
		require.NotNil(t, cmd.Flags().Lookup(flags.FlagFrom))
		require.NotNil(t, cmd.Flags().Lookup(flags.FlagFees))
		require.Contains(t, cmd.Long, "gas fees")
		require.Contains(t, cmd.Long, "cohort")
		require.Contains(t, cmd.Long, "economic credit")
	}
	cmd, _, err := GetQueryCmd().Find([]string{"fact-use-receipt"})
	require.NoError(t, err)
	require.Equal(t, "fact-use-receipt", cmd.Name())
	require.NoError(t, NewReportFactUseCmd().Args(nil, []string{"f"}))
	require.Error(t, NewReportFactUseCmd().Args(nil, []string{"f", "2"}))
}

func TestFactUseCLIReceiptQuery(t *testing.T) {
	consumer := sdk.AccAddress(make([]byte, 20)).String()
	clientCtx, captured := queryCaptureClient(t)
	cmd := NewQueryFactUseReceiptCmd()
	cmd.SetContext(context.Background())
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	require.NoError(t, client.SetCmdClientContext(cmd, clientCtx))
	cmd.SetArgs([]string{consumer, "fact-1"})
	require.ErrorContains(t, cmd.Execute(), "request captured")
	got := <-captured
	require.Equal(t, "FactUseReceipt", got.method)
	req := got.request.(*types.QueryFactUseReceiptRequest)
	require.Equal(t, consumer, req.Consumer)
	require.Equal(t, "fact-1", req.FactId)
}

func TestFactUseCLIRejectsMalformedInputsBeforeTransport(t *testing.T) {
	addr := sdk.AccAddress(make([]byte, 20))
	for _, tc := range []struct {
		cmd  *cobra.Command
		args []string
	}{
		{NewReportFactUseCmd(), []string{"bad/fact"}},
		{NewRateFactCmd(), []string{"fact", "true", strings.Repeat("é", 129)}},
		{NewRateFactCmd(), []string{"fact", "maybe"}},
		{NewQueryFactUseReceiptCmd(), []string{strings.ToUpper(addr.String()), "fact"}},
		{NewQueryFactUseReceiptCmd(), []string{addr.String(), "bad/fact"}},
	} {
		tc.cmd.SetContext(context.Background())
		tc.cmd.SilenceErrors, tc.cmd.SilenceUsage = true, true
		require.NoError(t, client.SetCmdClientContext(tc.cmd, client.Context{}.WithFromAddress(addr)))
		tc.cmd.SetArgs(tc.args)
		require.Error(t, tc.cmd.Execute())
	}
}
