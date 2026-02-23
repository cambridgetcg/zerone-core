package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// NewTxCmd returns the transaction commands for the vesting_rewards module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Vesting rewards module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateVestingCmd(),
		NewClaimVestingCmd(),
		NewPauseVestingCmd(),
		NewResumeVestingCmd(),
		NewAccelerateVestingCmd(),
		NewFalsifyVestingCmd(),
		NewCompleteVestingCmd(),
	)

	return txCmd
}

// NewCreateVestingCmd creates a CLI command for MsgCreateVesting.
func NewCreateVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [beneficiary] [amount] [category:0-6]",
		Short: "Create a new vesting schedule for a beneficiary",
		Long: `Create a vesting schedule with a specified category.
Categories: 0=unspecified, 1=verification_reward, 2=block_reward, 3=bounty_reward, 4=dispute_reward, 5=research_grant, 6=bootstrap.
Optionally link to a fact via --linked-fact-id and set a start height via --start-height.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			cat, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid category: %w", err)
			}

			linkedFactId, _ := cmd.Flags().GetString("linked-fact-id")
			startHeight, _ := cmd.Flags().GetUint64("start-height")

			msg := &types.MsgCreateVesting{
				Authority:    clientCtx.GetFromAddress().String(),
				Beneficiary:  args[0],
				Amount:       args[1],
				Category:     types.VestingCategory(cat),
				LinkedFactId: linkedFactId,
				StartHeight:  startHeight,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("linked-fact-id", "", "Optional linked fact ID for truth-anchored vesting")
	cmd.Flags().Uint64("start-height", 0, "Optional block height at which vesting begins")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClaimVestingCmd creates a CLI command for MsgClaimVesting.
func NewClaimVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim [vesting-ids-comma-separated]",
		Short: "Claim vested tokens from one or more vesting schedules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vestingIds := strings.Split(args[0], ",")

			msg := &types.MsgClaimVesting{
				Claimer:    clientCtx.GetFromAddress().String(),
				VestingIds: vestingIds,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewPauseVestingCmd creates a CLI command for MsgPauseVesting.
func NewPauseVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause [vesting-id] [reason]",
		Short: "Pause a vesting schedule (authority only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgPauseVesting{
				Authority: clientCtx.GetFromAddress().String(),
				VestingId: args[0],
				Reason:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewResumeVestingCmd creates a CLI command for MsgResumeVesting.
func NewResumeVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume [vesting-id]",
		Short: "Resume a paused vesting schedule (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgResumeVesting{
				Authority: clientCtx.GetFromAddress().String(),
				VestingId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAccelerateVestingCmd creates a CLI command for MsgAccelerateVesting.
func NewAccelerateVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accelerate [vesting-id] [acceleration-factor]",
		Short: "Accelerate a vesting schedule by a multiplier (authority only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			factor, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid acceleration factor: %w", err)
			}

			msg := &types.MsgAccelerateVesting{
				Authority:          clientCtx.GetFromAddress().String(),
				VestingId:          args[0],
				AccelerationFactor: uint32(factor),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFalsifyVestingCmd creates a CLI command for MsgFalsifyVesting.
func NewFalsifyVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "falsify [vesting-id] [reason] [counter-evidence-hash]",
		Short: "Challenge a vesting schedule with counter-evidence",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFalsifyVesting{
				Challenger:          clientCtx.GetFromAddress().String(),
				VestingId:           args[0],
				Reason:              args[1],
				CounterEvidenceHash: args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCompleteVestingCmd creates a CLI command for MsgCompleteVesting.
func NewCompleteVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete [vesting-id]",
		Short: "Complete a vesting schedule and release remaining tokens (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCompleteVesting{
				Authority: clientCtx.GetFromAddress().String(),
				VestingId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
