package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

const (
	FlagMaxExecutions    = "max-executions"
	FlagFeePerExecution  = "fee-per-execution"
	FlagPrepaidFee       = "prepaid-fee"
	FlagTransferValue    = "transfer-value"
	FlagLinkedEntityType = "linked-entity-type"
	FlagLinkedEntityId   = "linked-entity-id"
	FlagExpiresAtBlock   = "expires-at-block"
)

// NewTxCmd returns the transaction commands for the schedule module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Schedule module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateScheduleCmd(),
		NewPauseScheduleCmd(),
		NewResumeScheduleCmd(),
		NewCancelScheduleCmd(),
		NewFundScheduleCmd(),
	)

	return txCmd
}

// NewCreateScheduleCmd creates a CLI command for MsgCreateSchedule.
func NewCreateScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [target-address] [call-data]",
		Short: "Create a new scheduled process",
		Long: `Create a new scheduled process that executes against a target address.

Optional flags control execution limits, fees, linked entities, and expiration.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			maxExec, _ := cmd.Flags().GetUint64(FlagMaxExecutions)
			feePerExec, _ := cmd.Flags().GetString(FlagFeePerExecution)
			prepaidFee, _ := cmd.Flags().GetString(FlagPrepaidFee)
			transferValue, _ := cmd.Flags().GetString(FlagTransferValue)
			linkedEntityType, _ := cmd.Flags().GetString(FlagLinkedEntityType)
			linkedEntityId, _ := cmd.Flags().GetString(FlagLinkedEntityId)
			expiresAtBlock, _ := cmd.Flags().GetUint64(FlagExpiresAtBlock)

			msg := &types.MsgCreateSchedule{
				Creator:          clientCtx.GetFromAddress().String(),
				TargetAddress:    args[0],
				CallData:         args[1],
				MaxExecutions:    maxExec,
				FeePerExecution:  feePerExec,
				PrepaidFee:       prepaidFee,
				TransferValue:    transferValue,
				LinkedEntityType: linkedEntityType,
				LinkedEntityId:   linkedEntityId,
				ExpiresAtBlock:   expiresAtBlock,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint64(FlagMaxExecutions, 0, "Maximum number of executions (0 = unlimited)")
	cmd.Flags().String(FlagFeePerExecution, "", "Fee per execution (e.g. 1000uzerone)")
	cmd.Flags().String(FlagPrepaidFee, "", "Prepaid fee amount (e.g. 10000uzerone)")
	cmd.Flags().String(FlagTransferValue, "", "Value to transfer on each execution")
	cmd.Flags().String(FlagLinkedEntityType, "", "Linked entity type")
	cmd.Flags().String(FlagLinkedEntityId, "", "Linked entity ID")
	cmd.Flags().Uint64(FlagExpiresAtBlock, 0, "Block height at which the schedule expires (0 = no expiry)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewPauseScheduleCmd creates a CLI command for MsgPauseSchedule.
func NewPauseScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause [process-id]",
		Short: "Pause a scheduled process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgPauseSchedule{
				Creator:   clientCtx.GetFromAddress().String(),
				ProcessId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewResumeScheduleCmd creates a CLI command for MsgResumeSchedule.
func NewResumeScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume [process-id]",
		Short: "Resume a paused scheduled process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgResumeSchedule{
				Creator:   clientCtx.GetFromAddress().String(),
				ProcessId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCancelScheduleCmd creates a CLI command for MsgCancelSchedule.
func NewCancelScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel [process-id]",
		Short: "Cancel a scheduled process and refund remaining prepaid fees",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCancelSchedule{
				Creator:   clientCtx.GetFromAddress().String(),
				ProcessId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFundScheduleCmd creates a CLI command for MsgFundSchedule.
func NewFundScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund [process-id] [amount]",
		Short: "Add funds to a scheduled process",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFundSchedule{
				Creator:   clientCtx.GetFromAddress().String(),
				ProcessId: args[0],
				Amount:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
