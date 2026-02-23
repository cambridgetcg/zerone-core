package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "IBC rate limit module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryRateLimitCmd(),
		NewQueryRateLimitsCmd(),
		NewQueryParamsCmd(),
	)
	return queryCmd
}

func NewQueryRateLimitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-limit [channel-id] [denom]",
		Short: "Query a specific rate limit",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryRateLimitRequest{
				ChannelId: args[0],
				Denom:     args[1],
			}
			resp := &types.QueryRateLimitResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.ibcratelimit.v1.Query/RateLimit", req, resp); err != nil {
				return fmt.Errorf("failed to query rate limit: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryRateLimitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-limits",
		Short: "Query all rate limits",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryRateLimitsRequest{}
			resp := &types.QueryRateLimitsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.ibcratelimit.v1.Query/RateLimits", req, resp); err != nil {
				return fmt.Errorf("failed to query rate limits: %w", err)
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
		Short: "Query the ibcratelimit module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.ibcratelimit.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
