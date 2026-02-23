package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// NewQueryCmd returns the query commands for the liquiditypool module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Liquidity pool module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryPoolCmd(),
		NewQueryPoolsCmd(),
		NewQueryTWAPCmd(),
		NewQuerySimulateSwapCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryPoolCmd returns the command to query a pool by ID.
func NewQueryPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool [pool-id]",
		Short: "Query a liquidity pool by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryPoolRequest{PoolId: args[0]}
			resp := &types.QueryPoolResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.liquiditypool.v1.Query/Pool", req, resp); err != nil {
				return fmt.Errorf("failed to query pool: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryPoolsCmd returns the command to list all liquidity pools.
func NewQueryPoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pools",
		Short: "List all liquidity pools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryPoolsRequest{}
			resp := &types.QueryPoolsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.liquiditypool.v1.Query/Pools", req, resp); err != nil {
				return fmt.Errorf("failed to query pools: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryTWAPCmd returns the command to query the time-weighted average price.
func NewQueryTWAPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "twap [pool-id] [base-denom] [window]",
		Short: "Query the time-weighted average price for a pool",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			window, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			req := &types.QueryTWAPRequest{
				PoolId:    args[0],
				BaseDenom: args[1],
				Window:    window,
			}
			resp := &types.QueryTWAPResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.liquiditypool.v1.Query/TWAP", req, resp); err != nil {
				return fmt.Errorf("failed to query TWAP: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySimulateSwapCmd returns the command to simulate a swap.
func NewQuerySimulateSwapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulate-swap [pool-id] [token-in-denom] [token-in-amount]",
		Short: "Simulate a swap and return the expected output",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QuerySimulateSwapRequest{
				PoolId:        args[0],
				TokenInDenom:  args[1],
				TokenInAmount: args[2],
			}
			resp := &types.QuerySimulateSwapResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.liquiditypool.v1.Query/SimulateSwap", req, resp); err != nil {
				return fmt.Errorf("failed to simulate swap: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query liquiditypool module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the liquidity pool module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.liquiditypool.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
