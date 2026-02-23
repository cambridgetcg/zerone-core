package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/discovery/types"
)

// NewQueryCmd returns the query commands for the discovery module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Discovery module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryProfileCmd(),
		NewQuerySearchCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query discovery module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the discovery module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.discovery.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProfileCmd returns the command to query a discovery profile.
func NewQueryProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile [address]",
		Short: "Query a discovery profile by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProfileRequest{Address: args[0]}
			resp := &types.QueryProfileResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.discovery.v1.Query/Profile", req, resp); err != nil {
				return fmt.Errorf("failed to query profile: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySearchCmd returns the command to search discovery profiles.
func NewQuerySearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search --domain [d] --capability-type [t] --min-reputation [n]",
		Short: "Search discovery profiles by domain, capability type, or minimum reputation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			domain, _ := cmd.Flags().GetString("domain")
			capabilityType, _ := cmd.Flags().GetString("capability-type")
			minRepStr, _ := cmd.Flags().GetString("min-reputation")

			var minReputation uint64
			if minRepStr != "" {
				minReputation, err = strconv.ParseUint(minRepStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid min-reputation: %w", err)
				}
			}

			req := &types.QuerySearchRequest{
				Domain:         domain,
				CapabilityType: capabilityType,
				MinReputation:  minReputation,
			}
			resp := &types.QuerySearchResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.discovery.v1.Query/Search", req, resp); err != nil {
				return fmt.Errorf("failed to search profiles: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	cmd.Flags().String("domain", "", "Filter by domain")
	cmd.Flags().String("capability-type", "", "Filter by capability type")
	cmd.Flags().String("min-reputation", "", "Minimum reputation score")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
