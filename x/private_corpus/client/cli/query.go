package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/private_corpus/types"
)

const (
	flagLimit         = "limit"
	flagStartAfter    = "start-after"
	flagStartAfterSeq = "start-after-seq"
)

// NewQueryCmd returns the query commands for the private_corpus module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Private corpus module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryVaultCmd(),
		NewQueryVaultsCmd(),
		NewQueryVaultsByOperatorCmd(),
		NewQueryManifestCmd(),
		NewQueryManifestsByVaultCmd(),
		NewQueryAccessRecordsCmd(),
	)
	return queryCmd
}

// NewQueryParamsCmd queries module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the private_corpus module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.private_corpus.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("query params: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryVaultCmd queries a single vault by id.
func NewQueryVaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault [vault-id]",
		Short: "Query a vault by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryVaultRequest{Id: args[0]}
			resp := &types.QueryVaultResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.private_corpus.v1.Query/Vault", req, resp); err != nil {
				return fmt.Errorf("query vault: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryVaultsCmd queries vaults with pagination.
func NewQueryVaultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vaults",
		Short: "List vaults (paginated)",
		Long: `List vaults registered on the chain. Use --limit and --start-after to
page through large result sets.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetUint32(flagLimit)
			startAfter, _ := cmd.Flags().GetString(flagStartAfter)
			req := &types.QueryVaultsRequest{Limit: limit, StartAfterId: startAfter}
			resp := &types.QueryVaultsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.private_corpus.v1.Query/Vaults", req, resp); err != nil {
				return fmt.Errorf("query vaults: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	cmd.Flags().Uint32(flagLimit, 50, "Maximum number of vaults to return")
	cmd.Flags().String(flagStartAfter, "", "Vault id to start after (pagination cursor)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryVaultsByOperatorCmd queries vaults owned by an operator.
func NewQueryVaultsByOperatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vaults-by-operator [operator]",
		Short: "List vaults operated by a given address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryVaultsByOperatorRequest{Operator: args[0]}
			resp := &types.QueryVaultsByOperatorResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.private_corpus.v1.Query/VaultsByOperator", req, resp); err != nil {
				return fmt.Errorf("query vaults by operator: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryManifestCmd queries a single manifest by id.
func NewQueryManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest [manifest-id]",
		Short: "Query a manifest by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			req := &types.QueryManifestRequest{Id: args[0]}
			resp := &types.QueryManifestResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.private_corpus.v1.Query/Manifest", req, resp); err != nil {
				return fmt.Errorf("query manifest: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryManifestsByVaultCmd queries manifests for a vault.
func NewQueryManifestsByVaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifests-by-vault [vault-id]",
		Short: "List manifests published by a vault (paginated)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetUint32(flagLimit)
			startAfter, _ := cmd.Flags().GetString(flagStartAfter)
			req := &types.QueryManifestsByVaultRequest{
				VaultId:      args[0],
				Limit:        limit,
				StartAfterId: startAfter,
			}
			resp := &types.QueryManifestsByVaultResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.private_corpus.v1.Query/ManifestsByVault", req, resp); err != nil {
				return fmt.Errorf("query manifests by vault: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	cmd.Flags().Uint32(flagLimit, 50, "Maximum number of manifests to return")
	cmd.Flags().String(flagStartAfter, "", "Manifest id to start after (pagination cursor)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAccessRecordsCmd queries access records for a vault.
func NewQueryAccessRecordsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access-records [vault-id]",
		Short: "List opt-in access audit records for a vault (paginated)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetUint32(flagLimit)
			startAfterStr, _ := cmd.Flags().GetString(flagStartAfterSeq)
			var startAfter uint64
			if startAfterStr != "" {
				v, err := strconv.ParseUint(startAfterStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid --start-after-seq: %w", err)
				}
				startAfter = v
			}
			req := &types.QueryAccessRecordsRequest{
				VaultId:       args[0],
				Limit:         limit,
				StartAfterSeq: startAfter,
			}
			resp := &types.QueryAccessRecordsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.private_corpus.v1.Query/AccessRecords", req, resp); err != nil {
				return fmt.Errorf("query access records: %w", err)
			}
			return clientCtx.PrintObjectLegacy(resp)
		},
	}
	cmd.Flags().Uint32(flagLimit, 50, "Maximum number of records to return")
	cmd.Flags().String(flagStartAfterSeq, "", "Sequence number to start after (pagination cursor)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
