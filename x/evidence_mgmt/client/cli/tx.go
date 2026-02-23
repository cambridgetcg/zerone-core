package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/evidence_mgmt/types"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Evidence management module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		NewSubmitEvidenceCmd(),
	)
	return txCmd
}

func NewSubmitEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit [evidence-type] [content-hash] [metadata]",
		Short: "Submit new evidence (type: 1=document, 2=attestation, 3=measurement, 4=computation)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			evidenceType := types.EvidenceType_EVIDENCE_TYPE_DOCUMENT
			switch args[0] {
			case "1", "document":
				evidenceType = types.EvidenceType_EVIDENCE_TYPE_DOCUMENT
			case "2", "attestation":
				evidenceType = types.EvidenceType_EVIDENCE_TYPE_ATTESTATION
			case "3", "measurement":
				evidenceType = types.EvidenceType_EVIDENCE_TYPE_MEASUREMENT
			case "4", "computation":
				evidenceType = types.EvidenceType_EVIDENCE_TYPE_COMPUTATION
			}

			msg := &types.MsgSubmitEvidence{
				Submitter:    clientCtx.GetFromAddress().String(),
				EvidenceType: evidenceType,
				ContentHash:  args[1],
				Metadata:     args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
