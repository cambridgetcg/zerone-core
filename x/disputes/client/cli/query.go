package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/disputes/types"
)

// NewQueryCmd returns the query commands for the disputes module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Disputes module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryDisputeCmd(),
		NewQueryByTargetCmd(),
		NewQueryActiveCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryDisputeCmd returns the command to query a dispute by ID.
func NewQueryDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute [id]",
		Short: "Query a dispute by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryDisputeRequest{Id: args[0]}
			resp := &types.QueryDisputeResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.disputes.v1.Query/Dispute", req, resp); err != nil {
				return fmt.Errorf("failed to query dispute: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryByTargetCmd returns the command to query disputes by target.
func NewQueryByTargetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-target [target-id]",
		Short: "Query disputes by target ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryByTargetRequest{TargetId: args[0]}
			resp := &types.QueryByTargetResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.disputes.v1.Query/DisputesByTarget", req, resp); err != nil {
				return fmt.Errorf("failed to query disputes by target: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveCmd returns the command to list all active disputes.
func NewQueryActiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active",
		Short: "List all active disputes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryActiveRequest{}
			resp := &types.QueryActiveResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.disputes.v1.Query/ActiveDisputes", req, resp); err != nil {
				return fmt.Errorf("failed to query active disputes: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query disputes module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the disputes module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.disputes.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
