package cli

import (
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/bvm/types"
)

// NewTxCmd returns the transaction commands for the bvm module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "BVM module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewDeployContractCmd(),
		NewCallContractCmd(),
		NewScheduleExecutionCmd(),
	)

	return txCmd
}

// NewDeployContractCmd creates a CLI command for MsgDeployContract.
func NewDeployContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [bytecode-hex] [initial-deposit]",
		Short: "Deploy a new contract to the BVM",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			bytecode, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid bytecode hex: %w", err)
			}

			msg := &types.MsgDeployContract{
				Deployer:       clientCtx.GetFromAddress().String(),
				Bytecode:       bytecode,
				InitialDeposit: args[1],
			}

			constructorArgsHex, _ := cmd.Flags().GetString("constructor-args")
			if constructorArgsHex != "" {
				constructorArgs, err := hex.DecodeString(constructorArgsHex)
				if err != nil {
					return fmt.Errorf("invalid constructor-args hex: %w", err)
				}
				msg.ConstructorArgs = constructorArgs
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("constructor-args", "", "Constructor arguments as hex-encoded bytes")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCallContractCmd creates a CLI command for MsgCallContract.
func NewCallContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call [contract-address] [input-data-hex]",
		Short: "Call a deployed BVM contract",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			inputData, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid input-data hex: %w", err)
			}

			value, _ := cmd.Flags().GetString("value")
			gasLimit, _ := cmd.Flags().GetUint64("gas-limit")
			staticCall, _ := cmd.Flags().GetBool("static")

			msg := &types.MsgCallContract{
				Caller:          clientCtx.GetFromAddress().String(),
				ContractAddress: args[0],
				InputData:       inputData,
				Value:           value,
				GasLimit:        gasLimit,
				StaticCall:      staticCall,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("value", "", "Amount of tokens to send with the call")
	cmd.Flags().Uint64("gas-limit", 0, "Gas limit for the contract call")
	cmd.Flags().Bool("static", false, "Execute as a static (read-only) call")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewScheduleExecutionCmd creates a CLI command for MsgScheduleExecution.
func NewScheduleExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule-execution [contract-address] [input-data-hex]",
		Short: "Schedule a future contract execution",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			inputData, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid input-data hex: %w", err)
			}

			msg := &types.MsgScheduleExecution{
				Scheduler:       clientCtx.GetFromAddress().String(),
				ContractAddress: args[0],
				InputData:       inputData,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}