package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/qualification/types"
)

// NewQueryCmd returns the query commands for the qualification module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Qualification module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryQualificationCmd(),
		NewQueryByDomainCmd(),
		NewQueryByValidatorCmd(),
		NewQueryEndorsementsCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query qualification module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the qualification module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.qualification.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryQualificationCmd returns the command to query a specific qualification.
func NewQueryQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualification [validator] [domain]",
		Short: "Query a validator's qualification in a domain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryQualificationRequest{Validator: args[0], Domain: args[1]}
			resp := &types.QueryQualificationResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.qualification.v1.Query/Qualification", req, resp); err != nil {
				return fmt.Errorf("failed to query qualification: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryByDomainCmd returns the command to query qualifications by domain.
func NewQueryByDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-domain [domain]",
		Short: "Query all qualifications for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryByDomainRequest{Domain: args[0]}
			resp := &types.QueryByDomainResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.qualification.v1.Query/QualificationsByDomain", req, resp); err != nil {
				return fmt.Errorf("failed to query by domain: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryByValidatorCmd returns the command to query qualifications by validator.
func NewQueryByValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-validator [validator]",
		Short: "Query all qualifications for a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryByValidatorRequest{Validator: args[0]}
			resp := &types.QueryByValidatorResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.qualification.v1.Query/QualificationsByValidator", req, resp); err != nil {
				return fmt.Errorf("failed to query by validator: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEndorsementsCmd returns the command to query endorsements.
func NewQueryEndorsementsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endorsements [validator] [domain]",
		Short: "Query endorsements for a validator's domain qualification",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryEndorsementsRequest{Validator: args[0], Domain: args[1]}
			resp := &types.QueryEndorsementsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.qualification.v1.Query/Endorsements", req, resp); err != nil {
				return fmt.Errorf("failed to query endorsements: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
