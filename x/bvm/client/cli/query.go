package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/bvm/types"
)

// NewQueryCmd returns the query commands for the bvm module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "BVM module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryContractCmd(),
		NewQueryContractsByCreatorCmd(),
		NewQueryContractStateCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryContractCmd returns the command to query a contract by address.
func NewQueryContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract [address]",
		Short: "Query a contract by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryContractRequest{Address: args[0]}
			resp := &types.QueryContractResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.bvm.v1.Query/Contract", req, resp); err != nil {
				return fmt.Errorf("failed to query contract: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryContractsByCreatorCmd returns the command to query contracts by creator.
func NewQueryContractsByCreatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contracts-by-creator [creator]",
		Short: "Query all contracts deployed by a creator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryByCreatorRequest{Creator: args[0]}
			resp := &types.QueryByCreatorResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.bvm.v1.Query/ContractsByCreator", req, resp); err != nil {
				return fmt.Errorf("failed to query contracts by creator: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryContractStateCmd returns the command to query contract state.
func NewQueryContractStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract-state [address] [key]",
		Short: "Query a specific state entry of a contract",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryContractStateRequest{
				Address: args[0],
				Key:     args[1],
			}
			resp := &types.QueryContractStateResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.bvm.v1.Query/ContractState", req, resp); err != nil {
				return fmt.Errorf("failed to query contract state: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query the bvm module parameters.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the BVM module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.bvm.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
