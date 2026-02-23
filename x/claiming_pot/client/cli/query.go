package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/claiming_pot/types"
)

func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Claiming pot module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(
		NewQueryPotCmd(),
		NewQueryAllPotsCmd(),
		NewQueryParamsCmd(),
	)
	return queryCmd
}

func NewQueryPotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pot [id]",
		Short: "Query a claiming pot by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryPotRequest{Id: args[0]}
			resp := &types.QueryPotResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.claiming_pot.v1.Query/QueryPot", req, resp); err != nil {
				return fmt.Errorf("failed to query pot: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryAllPotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pots",
		Short: "List all claiming pots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryAllPotsRequest{}
			resp := &types.QueryAllPotsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.claiming_pot.v1.Query/QueryAllPots", req, resp); err != nil {
				return fmt.Errorf("failed to query pots: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query claiming pot module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.claiming_pot.v1.Query/QueryParams", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
