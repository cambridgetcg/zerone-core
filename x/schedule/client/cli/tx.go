package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.CLIName,
		Short:                      "Durable transfer schedule transactions",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(newCreateCmd(), newUpdateCmd(), newCancelCmd())
	return cmd
}

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [recipient] [amount-per-execution-uzrn] [first-height] [interval-blocks] [execution-count]",
		Short: "Prefund and create a finite height-based transfer schedule",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			firstHeight, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}
			interval, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return err
			}
			count, err := strconv.ParseUint(args[4], 10, 32)
			if err != nil {
				return err
			}
			msg := &types.MsgCreateSchedule{
				Creator:                clientCtx.GetFromAddress().String(),
				Recipient:              args[0],
				AmountPerExecutionUzrn: args[1],
				FirstExecutionHeight:   firstHeight,
				IntervalBlocks:         interval,
				ExecutionCount:         uint32(count),
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [schedule-id] [expected-revision] [expected-execution-count] [recipient] [amount-per-execution-uzrn] [next-height] [interval-blocks] [remaining-executions]",
		Short: "Compare-and-swap all not-yet-executed schedule terms",
		Args:  cobra.ExactArgs(8),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			revision, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			executed, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return err
			}
			nextHeight, err := strconv.ParseUint(args[5], 10, 64)
			if err != nil {
				return err
			}
			interval, err := strconv.ParseUint(args[6], 10, 64)
			if err != nil {
				return err
			}
			remaining, err := strconv.ParseUint(args[7], 10, 32)
			if err != nil {
				return err
			}
			msg := &types.MsgUpdateSchedule{
				Creator:                clientCtx.GetFromAddress().String(),
				ScheduleId:             args[0],
				ExpectedRevision:       revision,
				ExpectedExecutionCount: uint32(executed),
				Recipient:              args[3],
				AmountPerExecutionUzrn: args[4],
				NextExecutionHeight:    nextHeight,
				IntervalBlocks:         interval,
				RemainingExecutions:    uint32(remaining),
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func newCancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel [schedule-id] [expected-revision] [expected-execution-count]",
		Short: "Cancel an active schedule and refund all remaining escrow",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			revision, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			executed, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return err
			}
			msg := &types.MsgCancelSchedule{
				Creator:                clientCtx.GetFromAddress().String(),
				ScheduleId:             args[0],
				ExpectedRevision:       revision,
				ExpectedExecutionCount: uint32(executed),
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
