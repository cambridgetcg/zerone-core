package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/home/types"
)

// NewQueryCmd returns the query commands for the home module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Home module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryHomeCmd(),
		NewQueryHomesByOwnerCmd(),
		NewQueryKeysCmd(),
		NewQuerySessionsCmd(),
		NewQueryAlertsCmd(),
		NewQuerySpendingLimitsCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query home module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the home module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.home.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryHomeCmd returns the command to query a home.
func NewQueryHomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "home [home-id]",
		Short: "Query a home by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryHomeRequest{HomeId: args[0]}
			resp := &types.QueryHomeResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.home.v1.Query/Home", req, resp); err != nil {
				return fmt.Errorf("failed to query home: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryHomesByOwnerCmd returns the command to list homes by owner.
func NewQueryHomesByOwnerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "homes-by-owner [owner-address]",
		Short: "List all homes owned by an address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryHomesByOwnerRequest{Owner: args[0]}
			resp := &types.QueryHomesByOwnerResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.home.v1.Query/HomesByOwner", req, resp); err != nil {
				return fmt.Errorf("failed to query homes: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryKeysCmd returns the command to list keys for a home.
func NewQueryKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys [home-id]",
		Short: "List all registered keys for a home",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryKeysRequest{HomeId: args[0]}
			resp := &types.QueryKeysResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.home.v1.Query/Keys", req, resp); err != nil {
				return fmt.Errorf("failed to query keys: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySessionsCmd returns the command to list sessions for a home.
func NewQuerySessionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions [home-id]",
		Short: "List all active sessions for a home",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QuerySessionsRequest{HomeId: args[0]}
			resp := &types.QuerySessionsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.home.v1.Query/Sessions", req, resp); err != nil {
				return fmt.Errorf("failed to query sessions: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAlertsCmd returns the command to list alerts for a home.
func NewQueryAlertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts [home-id]",
		Short: "List all alerts for a home",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryAlertsRequest{HomeId: args[0]}
			resp := &types.QueryAlertsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.home.v1.Query/Alerts", req, resp); err != nil {
				return fmt.Errorf("failed to query alerts: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySpendingLimitsCmd returns the command to list spending limits for a home.
func NewQuerySpendingLimitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spending-limits [home-id]",
		Short: "List all spending limits for a home",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QuerySpendingLimitsRequest{HomeId: args[0]}
			resp := &types.QuerySpendingLimitsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.home.v1.Query/SpendingLimits", req, resp); err != nil {
				return fmt.Errorf("failed to query spending limits: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
