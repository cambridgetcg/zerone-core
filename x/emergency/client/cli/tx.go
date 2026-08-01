package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// NewTxCmd returns the transaction commands for the emergency module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Emergency module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewProposeHaltCmd(),
		NewVoteHaltCmd(),
		NewProposeResumeCmd(),
		NewVoteResumeCmd(),
		NewProposeRecoveryAuthorizationCmd(),
		NewVoteRecoveryAuthorizationCmd(),
	)

	return txCmd
}

// NewProposeRecoveryAuthorizationCmd creates a Guardian proposal that binds
// the exact next SDK governance ID, action, target plan, manifest, and allowed
// submitter before quarantine admits proposal submission.
func NewProposeRecoveryAuthorizationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "propose-recovery-authorization [next-sdk-gov-proposal-id] " +
			"[software_upgrade|cancel_upgrade] [action-sha256] " +
			"[upgrade-plan-sha256] [recovery-manifest-sha256] " +
			"[authorized-submitter] [justification]",
		Short: "Propose an incident-bound SDK governance recovery authorization (Guardian-only)",
		Args:  cobra.ExactArgs(7),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposalID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf(
					"invalid next SDK governance proposal id: %w",
					err,
				)
			}
			msg := &types.MsgProposeRecoveryAuthorization{
				Proposer:               clientCtx.GetFromAddress().String(),
				SdkGovProposalId:       proposalID,
				ActionType:             args[1],
				ActionSha256:           args[2],
				UpgradePlanSha256:      args[3],
				RecoveryManifestSha256: args[4],
				AuthorizedSubmitter:    args[5],
				Justification:          args[6],
			}
			return tx.GenerateOrBroadcastTxCLI(
				clientCtx,
				cmd.Flags(),
				msg,
			)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewVoteRecoveryAuthorizationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-recovery-authorization [ceremony-id] [approve: true/false]",
		Short: "Vote on an exact SDK governance recovery authorization (Guardian-only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			approve, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid approve value: %w", err)
			}
			msg := &types.MsgVoteRecoveryAuthorization{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Approve:    approve,
			}
			return tx.GenerateOrBroadcastTxCLI(
				clientCtx,
				cmd.Flags(),
				msg,
			)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProposeHaltCmd creates a CLI command for MsgProposeHalt.
func NewProposeHaltCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-halt [reason]",
		Short: "Propose an emergency transaction quarantine (Guardian-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgProposeHalt{
				Proposer: clientCtx.GetFromAddress().String(),
				Reason:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewVoteHaltCmd creates a CLI command for MsgVoteHalt.
func NewVoteHaltCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-halt [proposal-id] [approve: true/false]",
		Short: "Vote on a transaction-quarantine ceremony (Guardian-only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			approve, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid approve value: %w", err)
			}

			msg := &types.MsgVoteHalt{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Approve:    approve,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProposeRevertCmd is retained for source compatibility, but deliberately
// refuses to construct a height-only rollback request. Arbitrary finalized
// history cannot be selected safely without a hash-bound recovery manifest and
// explicit social coordination.
func NewProposeRevertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-revert [target-height] [justification]",
		Short: "Disabled: arbitrary height-only state revert is unsafe",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, _ []string) error {
			return types.ErrUnsafeRevertDisabled
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProposeResumeCmd creates a CLI command for MsgProposeResume.
func NewProposeResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-resume [recovery-manifest-sha256] [justification]",
		Short: "Propose reopening transaction admission using the verified RECOVERY_READY journal head",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgProposeResume{
				Proposer:               clientCtx.GetFromAddress().String(),
				RecoveryManifestSha256: args[0],
				Justification:          args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewVoteResumeCmd creates a CLI command for MsgVoteResume.
func NewVoteResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-resume [proposal-id] [approve: true/false]",
		Short: "Vote to reopen transaction admission after recovery verification",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			approve, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid approve value: %w", err)
			}

			msg := &types.MsgVoteResume{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Approve:    approve,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
