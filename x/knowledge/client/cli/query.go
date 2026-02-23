package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// GetQueryCmd returns the root query command for the knowledge module.
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Knowledge module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryFactCmd(),
		NewQueryFactsCmd(),
		NewQueryFactsByDomainCmd(),
		NewQueryFactsBySubmitterCmd(),
		NewQueryClaimCmd(),
		NewQueryPendingClaimsCmd(),
		NewQueryVerificationRoundCmd(),
		NewQueryDomainCmd(),
		NewQueryDomainsCmd(),
		NewQueryFactConfidenceCmd(),
		NewQueryFactCitationCountCmd(),
	)

	return queryCmd
}

func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the knowledge module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fact [id]",
		Short: "Query a fact by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryFactRequest{Id: args[0]}
			resp := &types.QueryFactResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/Fact", req, resp); err != nil {
				return fmt.Errorf("failed to query fact: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryFactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facts",
		Short: "Query facts with optional filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			domain, _ := cmd.Flags().GetString("domain")
			status, _ := cmd.Flags().GetString("status")
			category, _ := cmd.Flags().GetString("category")
			req := &types.QueryFactsRequest{
				Domain:   domain,
				Status:   status,
				Category: category,
			}
			resp := &types.QueryFactsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/Facts", req, resp); err != nil {
				return fmt.Errorf("failed to query facts: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	cmd.Flags().String("domain", "", "Filter by domain")
	cmd.Flags().String("status", "", "Filter by status")
	cmd.Flags().String("category", "", "Filter by category")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryFactsByDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facts-by-domain [domain]",
		Short: "Query facts by domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryFactsByDomainRequest{Domain: args[0]}
			resp := &types.QueryFactsByDomainResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/FactsByDomain", req, resp); err != nil {
				return fmt.Errorf("failed to query facts by domain: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryFactsBySubmitterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facts-by-submitter [submitter]",
		Short: "Query facts by submitter address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryFactsBySubmitterRequest{Submitter: args[0]}
			resp := &types.QueryFactsBySubmitterResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/FactsBySubmitter", req, resp); err != nil {
				return fmt.Errorf("failed to query facts by submitter: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryClaimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim [id]",
		Short: "Query a claim by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryClaimRequest{Id: args[0]}
			resp := &types.QueryClaimResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/Claim", req, resp); err != nil {
				return fmt.Errorf("failed to query claim: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryPendingClaimsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pending-claims",
		Short: "Query all pending claims",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryPendingClaimsRequest{}
			resp := &types.QueryPendingClaimsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/PendingClaims", req, resp); err != nil {
				return fmt.Errorf("failed to query pending claims: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryVerificationRoundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verification-round [id]",
		Short: "Query a verification round by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryVerificationRoundRequest{Id: args[0]}
			resp := &types.QueryVerificationRoundResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/VerificationRound", req, resp); err != nil {
				return fmt.Errorf("failed to query verification round: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain [name]",
		Short: "Query a knowledge domain by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryDomainRequest{Name: args[0]}
			resp := &types.QueryDomainResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/Domain", req, resp); err != nil {
				return fmt.Errorf("failed to query domain: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Query all knowledge domains",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryDomainsRequest{}
			resp := &types.QueryDomainsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/Domains", req, resp); err != nil {
				return fmt.Errorf("failed to query domains: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryFactConfidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fact-confidence [id]",
		Short: "Query the confidence score of a fact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryFactConfidenceRequest{Id: args[0]}
			resp := &types.QueryFactConfidenceResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/FactConfidence", req, resp); err != nil {
				return fmt.Errorf("failed to query fact confidence: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func NewQueryFactCitationCountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fact-citation-count [id]",
		Short: "Query the citation count of a fact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryFactCitationCountRequest{Id: args[0]}
			resp := &types.QueryFactCitationCountResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.knowledge.v1.Query/FactCitationCount", req, resp); err != nil {
				return fmt.Errorf("failed to query fact citation count: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
