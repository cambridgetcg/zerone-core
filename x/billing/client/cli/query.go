package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/billing/types"
)

// NewQueryCmd returns the query commands for the billing module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Billing module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryProviderCmd(),
		NewQueryProvidersCmd(),
		NewQueryQuoteCmd(),
		NewQueryBatchQuoteCmd(),
		NewQueryZRNPriceCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query billing module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the billing module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.billing.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProviderCmd returns the command to query a billing provider.
func NewQueryProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [address]",
		Short: "Query a knowledge API provider by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProviderRequest{Address: args[0]}
			resp := &types.QueryProviderResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.billing.v1.Query/Provider", req, resp); err != nil {
				return fmt.Errorf("failed to query provider: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProvidersCmd returns the command to list all providers.
func NewQueryProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "List all knowledge API providers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProvidersRequest{}
			resp := &types.QueryProvidersResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.billing.v1.Query/Providers", req, resp); err != nil {
				return fmt.Errorf("failed to query providers: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryQuoteCmd returns the command to get a price quote for a fact.
func NewQueryQuoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote [fact-id]",
		Short: "Get a price quote for querying a fact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryQuoteRequest{FactId: args[0]}
			resp := &types.QueryQuoteResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.billing.v1.Query/Quote", req, resp); err != nil {
				return fmt.Errorf("failed to query quote: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryBatchQuoteCmd returns the command to get batch price quotes.
func NewQueryBatchQuoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch-quote [fact-id1,fact-id2,...]",
		Short: "Get price quotes for multiple facts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			factIDs := strings.Split(args[0], ",")
			for i := range factIDs {
				factIDs[i] = strings.TrimSpace(factIDs[i])
			}

			req := &types.QueryBatchQuoteRequest{FactIds: factIDs}
			resp := &types.QueryBatchQuoteResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.billing.v1.Query/BatchQuote", req, resp); err != nil {
				return fmt.Errorf("failed to query batch quote: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryZRNPriceCmd returns the command to query the current ZRN price.
func NewQueryZRNPriceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zrn-price",
		Short: "Query the current ZRN price in USD",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryZRNPriceUSDRequest{}
			resp := &types.QueryZRNPriceUSDResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.billing.v1.Query/ZRNPriceUSD", req, resp); err != nil {
				return fmt.Errorf("failed to query ZRN price: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
