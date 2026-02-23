package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/icaauth/types"
)

func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "ICA auth module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryAccountCmd(),
		NewQueryAccountsCmd(),
		NewQueryParamsCmd(),
	)
	return queryCmd
}

func NewQueryAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [owner] [connection-id]",
		Short: "Query a specific interchain account",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryAccountRequest{
				Owner:        args[0],
				ConnectionId: args[1],
			}
			resp := &types.QueryAccountResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.icaauth.v1.Query/Account", req, resp); err != nil {
				return fmt.Errorf("failed to query account: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts [owner]",
		Short: "Query all interchain accounts for an owner",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryAccountsRequest{
				Owner: args[0],
			}
			resp := &types.QueryAccountsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.icaauth.v1.Query/Accounts", req, resp); err != nil {
				return fmt.Errorf("failed to query accounts: %w", err)
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
		Short: "Query the icaauth module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.icaauth.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
