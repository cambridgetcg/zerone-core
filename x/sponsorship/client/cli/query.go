package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Sponsorship module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(
		NewQueryBountyCmd(),
		NewQueryBountiesCmd(),
		NewQueryParamsCmd(),
	)
	return queryCmd
}

func NewQueryBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bounty [bounty-id]",
		Short: "Query a single bounty by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryBountyOrderRequest{Id: args[0]}
			resp := &types.QueryBountyOrderResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.sponsorship.v1.Query/BountyOrder", req, resp); err != nil {
				return fmt.Errorf("query bounty: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryBountiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bounties",
		Short: "List all bounties",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryBountyOrdersRequest{}
			resp := &types.QueryBountyOrdersResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.sponsorship.v1.Query/BountyOrders", req, resp); err != nil {
				return fmt.Errorf("query bounties: %w", err)
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
		Short: "Query sponsorship module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.sponsorship.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("query params: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
