package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/autopoiesis/types"
)

// NewTxCmd returns the transaction commands for the autopoiesis module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Autopoiesis module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewActivateCmd(),
		NewOverrideMultiplierCmd(),
		NewFreezeMultiplierCmd(),
		NewUpdateParamsCmd(),
	)

	return txCmd
}

// NewActivateCmd creates a CLI command for MsgActivateAutopoiesis.
func NewActivateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate [true/false]",
		Short: "Activate or deactivate the autopoiesis module (governance only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			activate, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("invalid activate value: %w", err)
			}

			msg := &types.MsgActivateAutopoiesis{
				Authority: clientCtx.GetFromAddress().String(),
				Activate:  activate,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewOverrideMultiplierCmd creates a CLI command for MsgOverrideMultiplier.
func NewOverrideMultiplierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "override-multiplier [path] [value-bps]",
		Short: "Force-set a multiplier value (governance only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			value, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid value: %w", err)
			}

			msg := &types.MsgOverrideMultiplier{
				Authority: clientCtx.GetFromAddress().String(),
				Path:      args[0],
				Value:     value,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFreezeMultiplierCmd creates a CLI command for MsgFreezeMultiplier.
func NewFreezeMultiplierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze-multiplier [path] [true/false]",
		Short: "Freeze or unfreeze a multiplier (governance only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			frozen, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid frozen value: %w", err)
			}

			msg := &types.MsgFreezeMultiplier{
				Authority: clientCtx.GetFromAddress().String(),
				Path:      args[0],
				Frozen:    frozen,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateParamsCmd creates a CLI command for MsgUpdateParams.
func NewUpdateParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params",
		Short: "Update autopoiesis module parameters (governance only)",
		Long:  "Update module parameters via governance proposal. Parameters are set via flags.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			epochLen, _ := cmd.Flags().GetUint64("epoch-length")
			maxChange, _ := cmd.Flags().GetUint64("max-change")
			enabled, _ := cmd.Flags().GetBool("enabled")

			params := types.DefaultParams()
			if epochLen > 0 {
				params.EpochLengthBlocks = epochLen
			}
			if maxChange > 0 {
				params.MaxChangePerEpochBps = maxChange
			}
			params.Enabled = enabled

			msg := &types.MsgUpdateParams{
				Authority: clientCtx.GetFromAddress().String(),
				Params:    &params,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint64("epoch-length", 0, "Epoch length in blocks")
	cmd.Flags().Uint64("max-change", 0, "Max change per epoch in BPS")
	cmd.Flags().Bool("enabled", true, "Enable or disable the module")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
