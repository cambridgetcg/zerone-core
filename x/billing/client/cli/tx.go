package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/billing/types"
)

// NewTxCmd returns the transaction commands for the billing module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Billing module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterProviderCmd(),
		NewDeregisterProviderCmd(),
		NewQueryFactCmd(),
		NewBatchQueryFactsCmd(),
	)

	return txCmd
}

// NewRegisterProviderCmd creates a CLI command for MsgRegisterProvider.
func NewRegisterProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-provider [stake-amount] [domain1,domain2,...] --name [provider-name]",
		Short: "Register as a knowledge API provider",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			domains := strings.Split(args[1], ",")
			for i := range domains {
				domains[i] = strings.TrimSpace(domains[i])
			}

			name, _ := cmd.Flags().GetString("name")

			msg := &types.MsgRegisterProvider{
				Sender:  clientCtx.GetFromAddress().String(),
				Name:    name,
				Stake:   args[0],
				Domains: domains,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("name", "", "Provider display name")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDeregisterProviderCmd creates a CLI command for MsgDeregisterProvider.
func NewDeregisterProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deregister-provider",
		Short: "Deregister as a knowledge API provider and refund stake",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDeregisterProvider{
				Sender: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewQueryFactCmd creates a CLI command for MsgQueryFact.
func NewQueryFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-fact [provider-address] [fact-id]",
		Short: "Query a single fact (quote, pay, and distribute in one tx)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgQueryFact{
				Sender:   clientCtx.GetFromAddress().String(),
				Provider: args[0],
				FactId:   args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewBatchQueryFactsCmd creates a CLI command for MsgBatchQueryFacts.
func NewBatchQueryFactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch-query-facts [provider-address] [fact-id1,fact-id2,...]",
		Short: "Query multiple facts in a single transaction",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			factIDs := strings.Split(args[1], ",")
			for i := range factIDs {
				factIDs[i] = strings.TrimSpace(factIDs[i])
			}

			msg := &types.MsgBatchQueryFacts{
				Sender:   clientCtx.GetFromAddress().String(),
				Provider: args[0],
				FactIds:  factIDs,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
