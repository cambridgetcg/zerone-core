package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// NewReportFactUseCmd uses the existing wallet and transaction flow. A read is
// never upgraded to a signed report implicitly.
func NewReportFactUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report-fact-use [fact-id]",
		Short: "Sign one public, non-economic self-report of fact use (cohort only)",
		Long: `Sign a public self-report using your existing --from wallet. This is not measured
readership and earns no economic credit. Normal transaction gas fees apply, even
to a validly signed transaction whose message fails. The beta is disabled by
default and requires both a registered Zerone account and explicit consumer-cohort
admission. One report per consumer/fact/epoch; a fresh sequence does not permit a
duplicate. Check commitment and query fact-use-receipt before rating. A broadcast
acknowledgement is not proof of commitment.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgReportFactUse{Consumer: clientCtx.GetFromAddress().String(), FactId: args[0]}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewQueryFactUseReceiptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fact-use-receipt [consumer] [fact-id]",
		Short: "Read the signed-use receipt for the query context's current epoch",
		Long: `Read one receipt without writing state or signing a transaction. Found=false
means no receipt in the query context's current epoch, not no historical use.
The response includes the epoch and actual snapshot block height.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := types.CanonicalFactUseConsumer(args[0]); err != nil {
				return err
			}
			if err := types.ValidateFactUseFactID(args[1]); err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryFactUseReceiptRequest{Consumer: args[0], FactId: args[1]}
			resp := &types.QueryFactUseReceiptResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/FactUseReceipt", req, resp); err != nil {
				return fmt.Errorf("failed to query fact-use receipt: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
