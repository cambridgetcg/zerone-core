package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/tokens/types"
)

// NewQueryCmd returns the query commands for the tokens module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tokens module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryTokenConfigCmd(),
		NewQueryTokensCmd(),
		NewQueryTokenBySymbolCmd(),
		NewQueryBalanceCmd(),
		NewQueryTotalSupplyCmd(),
		NewQueryAllowanceCmd(),
		NewQueryDelegatedPowerCmd(),
		NewQueryWrappedTokenCmd(),
		NewQueryWrappedTokensCmd(),
		NewQueryEmissionPeriodCmd(),
		NewQueryEmissionPeriodsCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query tokens module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the tokens module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryTokenConfigCmd returns the command to query a token configuration.
func NewQueryTokenConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token-config [token-id]",
		Short: "Query a token configuration by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryTokenConfigRequest{TokenId: args[0]}
			resp := &types.QueryTokenConfigResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/TokenConfig", req, resp); err != nil {
				return fmt.Errorf("failed to query token config: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryTokensCmd returns the command to list all tokens.
func NewQueryTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "List all registered tokens",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryTokensRequest{}
			resp := &types.QueryTokensResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/Tokens", req, resp); err != nil {
				return fmt.Errorf("failed to query tokens: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryTokenBySymbolCmd returns the command to query a token by symbol.
func NewQueryTokenBySymbolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token-by-symbol [symbol]",
		Short: "Query a token by its ticker symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryTokenBySymbolRequest{Symbol: args[0]}
			resp := &types.QueryTokenBySymbolResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/TokenBySymbol", req, resp); err != nil {
				return fmt.Errorf("failed to query token by symbol: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryBalanceCmd returns the command to query a token balance.
func NewQueryBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance [token-id] [address]",
		Short: "Query the token balance of an address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryBalanceRequest{
				TokenId: args[0],
				Address: args[1],
			}
			resp := &types.QueryBalanceResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/Balance", req, resp); err != nil {
				return fmt.Errorf("failed to query balance: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryTotalSupplyCmd returns the command to query a token's total supply.
func NewQueryTotalSupplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total-supply [token-id]",
		Short: "Query the total supply of a token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryTotalSupplyRequest{TokenId: args[0]}
			resp := &types.QueryTotalSupplyResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/TotalSupply", req, resp); err != nil {
				return fmt.Errorf("failed to query total supply: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAllowanceCmd returns the command to query a token allowance.
func NewQueryAllowanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "allowance [token-id] [owner] [spender]",
		Short: "Query the allowance granted by an owner to a spender",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryAllowanceRequest{
				TokenId: args[0],
				Owner:   args[1],
				Spender: args[2],
			}
			resp := &types.QueryAllowanceResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/Allowance", req, resp); err != nil {
				return fmt.Errorf("failed to query allowance: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDelegatedPowerCmd returns the command to query delegated power.
func NewQueryDelegatedPowerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegated-power [token-id] [address]",
		Short: "Query the delegated governance power for an address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryDelegatedPowerRequest{
				TokenId: args[0],
				Address: args[1],
			}
			resp := &types.QueryDelegatedPowerResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/DelegatedPower", req, resp); err != nil {
				return fmt.Errorf("failed to query delegated power: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryWrappedTokenCmd returns the command to query a wrapped token.
func NewQueryWrappedTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wrapped-token [token-id]",
		Short: "Query the wrapped bank denom for a token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryWrappedTokenRequest{TokenId: args[0]}
			resp := &types.QueryWrappedTokenResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/WrappedToken", req, resp); err != nil {
				return fmt.Errorf("failed to query wrapped token: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryWrappedTokensCmd returns the command to list all wrapped tokens.
func NewQueryWrappedTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wrapped-tokens",
		Short: "List all wrapped token records",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryWrappedTokensRequest{}
			resp := &types.QueryWrappedTokensResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/WrappedTokens", req, resp); err != nil {
				return fmt.Errorf("failed to query wrapped tokens: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEmissionPeriodCmd returns the command to query an emission period.
func NewQueryEmissionPeriodCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emission-period [emission-id]",
		Short: "Query an emission period by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryEmissionPeriodRequest{EmissionId: args[0]}
			resp := &types.QueryEmissionPeriodResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/EmissionPeriod", req, resp); err != nil {
				return fmt.Errorf("failed to query emission period: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEmissionPeriodsCmd returns the command to list emission periods.
func NewQueryEmissionPeriodsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emission-periods",
		Short: "List all emission periods",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			activeOnly, _ := cmd.Flags().GetBool("active-only")

			req := &types.QueryEmissionPeriodsRequest{
				ActiveOnly: activeOnly,
			}
			resp := &types.QueryEmissionPeriodsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.tokens.v1.Query/EmissionPeriods", req, resp); err != nil {
				return fmt.Errorf("failed to query emission periods: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	cmd.Flags().Bool("active-only", false, "Filter to only active emission periods")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
