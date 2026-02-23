package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/alignment/types"
)

// NewTxCmd returns the transaction commands for the alignment module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Alignment module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewActivateCmd(),
	)

	return txCmd
}

// NewActivateCmd creates a CLI command for MsgActivate.
func NewActivateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate [enabled: true/false]",
		Short: "Enable or disable the alignment module (authority-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			enabled, err := strconv.ParseBool(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgActivate{
				Authority: clientCtx.GetFromAddress().String(),
				Enabled:   enabled,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
