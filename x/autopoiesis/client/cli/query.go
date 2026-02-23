package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/autopoiesis/types"
)

// NewQueryCmd returns the query commands for the autopoiesis module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Autopoiesis module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryMultiplierCmd(),
		NewQueryAllMultipliersCmd(),
		NewQueryEpochSnapshotCmd(),
		NewQuerySSICmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command for querying module parameters.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query autopoiesis module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryMultiplierCmd returns the command for querying a single multiplier.
func NewQueryMultiplierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multiplier [path]",
		Short: "Query a multiplier by path (e.g. rewards.block)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.Multiplier(cmd.Context(), &types.QueryMultiplierRequest{Path: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAllMultipliersCmd returns the command for querying all multipliers.
func NewQueryAllMultipliersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-multipliers",
		Short: "Query all active multipliers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.AllMultipliers(cmd.Context(), &types.QueryAllMultipliersRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEpochSnapshotCmd returns the command for querying an epoch snapshot.
func NewQueryEpochSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epoch-snapshot [epoch]",
		Short: "Query the snapshot for a specific epoch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			epoch, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid epoch: %w", err)
			}
			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.EpochSnapshot(cmd.Context(), &types.QueryEpochSnapshotRequest{Epoch: epoch})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySSICmd returns the command for querying the current SSI.
func NewQuerySSICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssi",
		Short: "Query the current System Stability Index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.SSI(cmd.Context(), &types.QuerySSIRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
