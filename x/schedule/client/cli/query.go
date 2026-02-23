package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

// NewQueryCmd returns the query commands for the schedule module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Schedule module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryProcessCmd(),
		NewQueryProcessesByCreatorCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query schedule module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the schedule module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.schedule.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProcessCmd returns the command to query a scheduled process by ID.
func NewQueryProcessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process [process-id]",
		Short: "Query a scheduled process by its ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProcessRequest{ProcessId: args[0]}
			resp := &types.QueryProcessResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.schedule.v1.Query/Process", req, resp); err != nil {
				return fmt.Errorf("failed to query process: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProcessesByCreatorCmd returns the command to list processes by creator.
func NewQueryProcessesByCreatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "processes-by-creator [creator]",
		Short: "List all scheduled processes created by an address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProcessesByCreatorRequest{Creator: args[0]}
			resp := &types.QueryProcessesByCreatorResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.schedule.v1.Query/ProcessesByCreator", req, resp); err != nil {
				return fmt.Errorf("failed to query processes: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
