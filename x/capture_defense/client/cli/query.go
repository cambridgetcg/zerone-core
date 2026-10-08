package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Capture defense module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryReputationCmd(),
		NewQueryCaptureMetricsCmd(),
		NewQueryCrossStratumRequirementsCmd(),
		NewQueryFlaggedDomainsCmd(),
	)

	return queryCmd
}

func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the capture defense module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp, err := types.NewQueryClient(clientCtx).Params(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryReputationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reputation [validator] [domain]",
		Short: "Query a validator's reputation (optionally for a domain)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryReputationRequest{Validator: args[0]}
			if len(args) > 1 {
				req.Domain = args[1]
			}
			resp, err := types.NewQueryClient(clientCtx).Reputation(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("failed to query reputation: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryCrossStratumRequirementsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cross-stratum-requirements",
		Short: "Query configured cross-stratum verification requirements",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			resp, err := types.NewQueryClient(clientCtx).CrossStratumRequirements(cmd.Context(), &types.QueryCrossStratumRequirementsRequest{})
			if err != nil {
				return fmt.Errorf("failed to query cross-stratum requirements: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryFlaggedDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flagged-domains",
		Short: "Query all domains with active capture flags",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryFlaggedDomainsRequest{}
			resp, err := types.NewQueryClient(clientCtx).FlaggedDomains(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("failed to query flagged domains: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryCaptureMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics [domain]",
		Short: "Query capture risk metrics for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryCaptureMetricsRequest{Domain: args[0]}
			resp, err := types.NewQueryClient(clientCtx).CaptureMetrics(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("failed to query capture metrics: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
