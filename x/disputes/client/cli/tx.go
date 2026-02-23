package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/disputes/types"
)

// NewTxCmd returns the transaction commands for the disputes module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Disputes module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewInitiateDisputeCmd(),
		NewCommitEvidenceCmd(),
		NewRevealEvidenceCmd(),
		NewArbiterVoteCmd(),
		NewEscalateDisputeCmd(),
		NewSettleDisputeCmd(),
	)

	return txCmd
}

// NewInitiateDisputeCmd creates a CLI command for MsgInitiateDispute.
func NewInitiateDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initiate [target-type] [target-id] [reason] [bond-amount]",
		Short: "Initiate a dispute against a target (fact, evidence, or validator)",
		Long:  "target-type: 1=fact, 2=evidence, 3=validator",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			targetType := types.DisputeTargetType_DISPUTE_TARGET_TYPE_FACT
			switch args[0] {
			case "1", "fact":
				targetType = types.DisputeTargetType_DISPUTE_TARGET_TYPE_FACT
			case "2", "evidence":
				targetType = types.DisputeTargetType_DISPUTE_TARGET_TYPE_EVIDENCE
			case "3", "validator":
				targetType = types.DisputeTargetType_DISPUTE_TARGET_TYPE_VALIDATOR
			}

			msg := &types.MsgInitiateDispute{
				Challenger: clientCtx.GetFromAddress().String(),
				TargetType: targetType,
				TargetId:   args[1],
				Reason:     args[2],
				Bond:       args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCommitEvidenceCmd creates a CLI command for MsgCommitEvidence.
func NewCommitEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit-evidence [dispute-id] [commitment-hash]",
		Short: "Commit an evidence hash (SHA256 of content+nonce)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCommitEvidence{
				Submitter:      clientCtx.GetFromAddress().String(),
				DisputeId:      args[0],
				CommitmentHash: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRevealEvidenceCmd creates a CLI command for MsgRevealEvidence.
func NewRevealEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reveal-evidence [dispute-id] [content] [nonce]",
		Short: "Reveal previously committed evidence content",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRevealEvidence{
				Submitter: clientCtx.GetFromAddress().String(),
				DisputeId: args[0],
				Content:   args[1],
				Nonce:     args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewArbiterVoteCmd creates a CLI command for MsgArbiterVote.
func NewArbiterVoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote [dispute-id] [decision] [reasoning]",
		Short: "Cast an arbiter vote (decision: challenger, defender, abstain)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var decision types.ArbiterDecision
			switch args[1] {
			case "1", "challenger":
				decision = types.ArbiterDecision_ARBITER_DECISION_CHALLENGER
			case "2", "defender":
				decision = types.ArbiterDecision_ARBITER_DECISION_DEFENDER
			case "3", "abstain":
				decision = types.ArbiterDecision_ARBITER_DECISION_ABSTAIN
			default:
				decision = types.ArbiterDecision_ARBITER_DECISION_UNSPECIFIED
			}

			msg := &types.MsgArbiterVote{
				Arbiter:   clientCtx.GetFromAddress().String(),
				DisputeId: args[0],
				Vote:      decision,
				Reasoning: args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewEscalateDisputeCmd creates a CLI command for MsgEscalateDispute.
func NewEscalateDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "escalate [dispute-id] [additional-bond]",
		Short: "Escalate a dispute to a higher tier with additional bond",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEscalateDispute{
				Requester:      clientCtx.GetFromAddress().String(),
				DisputeId:      args[0],
				AdditionalBond: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSettleDisputeCmd creates a CLI command for MsgSettleDispute.
func NewSettleDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settle [dispute-id]",
		Short: "Manually settle a dispute (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgSettleDispute{
				Authority: clientCtx.GetFromAddress().String(),
				DisputeId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
