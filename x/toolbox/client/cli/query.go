package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// NewQueryCmd returns the query commands for the toolbox module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Toolbox module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryToolCmd(),
		NewQueryToolsByDeployerCmd(),
		NewQueryToolsByCategoryCmd(),
		NewQueryTrustScoreCmd(),
		NewQueryDependencyTreeCmd(),
		NewQueryFreeAllowanceCmd(),
		NewQueryParamsCmd(),
		NewQueryToolPriceCmd(),
	)

	return queryCmd
}

// NewQueryToolCmd returns the command to query a tool by ID.
func NewQueryToolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tool [tool-id]",
		Short: "Query a tool by its ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryToolRequest{ToolId: args[0]}
			resp := &types.QueryToolResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.toolbox.v1.Query/Tool", req, resp); err != nil {
				return fmt.Errorf("failed to query tool: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryToolsByDeployerCmd returns the command to list tools by deployer.
func NewQueryToolsByDeployerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools-by-deployer [deployer]",
		Short: "List all tools registered by a deployer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryByDeployerRequest{Deployer: args[0]}
			resp := &types.QueryByDeployerResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.toolbox.v1.Query/ToolsByDeployer", req, resp); err != nil {
				return fmt.Errorf("failed to query tools by deployer: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryToolsByCategoryCmd returns the command to list tools by category.
func NewQueryToolsByCategoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools-by-category [category]",
		Short: "List all tools in a category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryByCategoryRequest{Category: args[0]}
			resp := &types.QueryByCategoryResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.toolbox.v1.Query/ToolsByCategory", req, resp); err != nil {
				return fmt.Errorf("failed to query tools by category: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryTrustScoreCmd returns the command to query a tool's trust score.
func NewQueryTrustScoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust-score [tool-id]",
		Short: "Query the trust score for a tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryTrustScoreRequest{ToolId: args[0]}
			resp := &types.QueryTrustScoreResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.toolbox.v1.Query/TrustScore", req, resp); err != nil {
				return fmt.Errorf("failed to query trust score: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDependencyTreeCmd returns the command to query a tool's dependency tree.
func NewQueryDependencyTreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependency-tree [tool-id]",
		Short: "Query the dependency tree for a tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryDependencyTreeRequest{ToolId: args[0]}
			resp := &types.QueryDependencyTreeResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.toolbox.v1.Query/DependencyTree", req, resp); err != nil {
				return fmt.Errorf("failed to query dependency tree: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryFreeAllowanceCmd returns the command to query a caller's free allowance.
func NewQueryFreeAllowanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "free-allowance [caller]",
		Short: "Query the free tier allowance for a caller",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryFreeAllowanceRequest{Caller: args[0]}
			resp := &types.QueryFreeAllowanceResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.toolbox.v1.Query/FreeAllowance", req, resp); err != nil {
				return fmt.Errorf("failed to query free allowance: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query toolbox module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the toolbox module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.toolbox.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryToolPriceCmd returns the command to query a tool's current price.
func NewQueryToolPriceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tool-price [tool-id]",
		Short: "Query the current price for a tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryToolPriceRequest{ToolId: args[0]}
			resp := &types.QueryToolPriceResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.toolbox.v1.Query/ToolPrice", req, resp); err != nil {
				return fmt.Errorf("failed to query tool price: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
