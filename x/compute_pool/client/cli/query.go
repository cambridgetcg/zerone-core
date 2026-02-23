package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/compute_pool/types"
)

// NewQueryCmd returns the query commands for the compute_pool module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Compute pool module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryProviderCmd(),
		NewQueryProvidersCmd(),
		NewQueryCreditCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query compute_pool module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the compute pool module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.compute_pool.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProviderCmd returns the command to query a compute provider.
func NewQueryProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [address]",
		Short: "Query a compute provider by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProviderRequest{Address: args[0]}
			resp := &types.QueryProviderResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.compute_pool.v1.Query/Provider", req, resp); err != nil {
				return fmt.Errorf("failed to query provider: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProvidersCmd returns the command to list compute providers by service type.
func NewQueryProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers [service-type]",
		Short: "List compute providers by service type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProvidersRequest{ServiceType: args[0]}
			resp := &types.QueryProvidersResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.compute_pool.v1.Query/Providers", req, resp); err != nil {
				return fmt.Errorf("failed to query providers: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryCreditCmd returns the command to query compute credits for a validator.
func NewQueryCreditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credit [validator-addr]",
		Short: "Query compute credits for a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryCreditRequest{ValidatorAddr: args[0]}
			resp := &types.QueryCreditResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.compute_pool.v1.Query/Credit", req, resp); err != nil {
				return fmt.Errorf("failed to query credit: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
