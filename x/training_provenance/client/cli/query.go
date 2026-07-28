package cli

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cmtservice "github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/spf13/cobra"

	intotoprojection "github.com/zerone-chain/zerone/x/training_provenance/intoto"
	"github.com/zerone-chain/zerone/x/training_provenance/types"
)

// GetQueryCmd returns the training-provenance query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Query training-manifest provenance",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdProvenanceCertificate(),
		CmdInTotoStatement(),
	)
	return cmd
}

// CmdProvenanceCertificate queries Zerone's native live certificate.
func CmdProvenanceCertificate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certificate [manifest-id]",
		Short: "Query the live Zerone provenance certificate for a training manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).ProvenanceCertificate(
				cmd.Context(),
				&types.QueryProvenanceCertificateRequest{ManifestId: args[0]},
			)
			if err != nil {
				return fmt.Errorf("query provenance certificate: %w", err)
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdInTotoStatement exports an unsigned in-toto Statement v1 projection of
// Zerone's live certificate. The node-reported chain ID is represented as
// CAIP-2.
func CmdInTotoStatement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "in-toto-statement [manifest-id]",
		Short: "Export an unsigned in-toto Statement v1 for a training manifest",
		Long: "Export Zerone's live provenance snapshot as in-toto Statement v1 JSON. " +
			"The output is unsigned; use it as a DSSE payload and sign the DSSE pre-authentication encoding to create an authenticated attestation.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			chainID, err := sourceChainID(cmd.Context(), clientCtx)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).ProvenanceCertificate(
				cmd.Context(),
				&types.QueryProvenanceCertificateRequest{ManifestId: args[0]},
			)
			if err != nil {
				return fmt.Errorf("query provenance certificate: %w", err)
			}
			if res.Certificate == nil {
				return fmt.Errorf("query provenance certificate: empty certificate")
			}

			statement, err := intotoprojection.BuildStatement(chainID, res.Certificate)
			if err != nil {
				return fmt.Errorf("build in-toto statement: %w", err)
			}
			bz, err := intotoprojection.MarshalStatementJSON(statement)
			if err != nil {
				return err
			}
			return clientCtx.PrintString(string(bz) + "\n")
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// sourceChainID reads the chain ID from the same client connection used for
// the certificate query. The value is still a claim by that node; consumers
// need a trusted endpoint or chain proof for origin assurance.
func sourceChainID(ctx context.Context, clientCtx client.Context) (string, error) {
	res, err := cmtservice.NewServiceClient(clientCtx).GetNodeInfo(ctx, &cmtservice.GetNodeInfoRequest{})
	if err != nil {
		return "", fmt.Errorf("query source chain ID: %w", err)
	}
	if res == nil || res.DefaultNodeInfo == nil {
		return "", fmt.Errorf("query source chain ID: node returned no network information")
	}
	if res.DefaultNodeInfo.Network == "" {
		return "", fmt.Errorf("node reported an empty chain ID")
	}
	return res.DefaultNodeInfo.Network, nil
}
