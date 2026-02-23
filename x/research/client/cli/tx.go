package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/research/types"
)

// NewTxCmd returns the transaction commands for the research module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Research module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSubmitResearchCmd(),
		NewChallengeResearchCmd(),
		NewReviewResearchCmd(),
		NewResolveResearchCmd(),
		NewCreateBountyCmd(),
		NewClaimBountyCmd(),
		NewFulfillBountyCmd(),
		NewFundResearchCmd(),
	)

	return txCmd
}

// NewSubmitResearchCmd creates a CLI command for MsgSubmitResearch.
func NewSubmitResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit [title] [description] [domain] [stake]",
		Short: "Submit new research for peer review",
		Long:  "Submit a research proposal with title, description, domain, and stake amount. Optionally attach supporting fact IDs via --supporting-facts.",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var supportingFacts []string
			factsStr, _ := cmd.Flags().GetString("supporting-facts")
			if factsStr != "" {
				supportingFacts = strings.Split(factsStr, ",")
			}

			msg := &types.MsgSubmitResearch{
				Submitter:       clientCtx.GetFromAddress().String(),
				Title:           args[0],
				Description:     args[1],
				Domain:          args[2],
				Stake:           args[3],
				SupportingFacts: supportingFacts,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("supporting-facts", "", "Comma-separated list of supporting fact IDs")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewChallengeResearchCmd creates a CLI command for MsgChallengeResearch.
func NewChallengeResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge [research-id] [reason] [stake]",
		Short: "Challenge a submitted research",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgChallengeResearch{
				Challenger: clientCtx.GetFromAddress().String(),
				ResearchId: args[0],
				Reason:     args[1],
				Stake:      args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewReviewResearchCmd creates a CLI command for MsgReviewResearch.
func NewReviewResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review [research-id] [verdict:0-3] [reasoning] [quality-score]",
		Short: "Review a submitted research (verdict: 0=unspecified, 1=approve, 2=reject, 3=revise)",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			verdict, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid verdict: %w", err)
			}
			if verdict > 3 {
				return fmt.Errorf("verdict must be 0-3, got %d", verdict)
			}

			qualityScore, err := strconv.ParseUint(args[3], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid quality score: %w", err)
			}

			msg := &types.MsgReviewResearch{
				Reviewer:     clientCtx.GetFromAddress().String(),
				ResearchId:   args[0],
				Verdict:      types.ReviewVerdict(verdict),
				Reasoning:    args[2],
				QualityScore: uint32(qualityScore),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewResolveResearchCmd creates a CLI command for MsgResolveResearch.
func NewResolveResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve [research-id]",
		Short: "Resolve a research submission (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgResolveResearch{
				Authority:  clientCtx.GetFromAddress().String(),
				ResearchId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCreateBountyCmd creates a CLI command for MsgCreateBounty.
func NewCreateBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-bounty [title] [description] [reward] [deadline-height]",
		Short: "Create a research bounty",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			deadlineHeight, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid deadline height: %w", err)
			}

			msg := &types.MsgCreateBounty{
				Creator:        clientCtx.GetFromAddress().String(),
				Title:          args[0],
				Description:    args[1],
				Reward:         args[2],
				DeadlineHeight: deadlineHeight,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClaimBountyCmd creates a CLI command for MsgClaimBounty.
func NewClaimBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-bounty [bounty-id]",
		Short: "Claim a research bounty with supporting facts",
		Long:  "Claim a research bounty by providing the bounty ID and optionally a list of fact IDs via --fact-ids.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var factIds []string
			factIdsStr, _ := cmd.Flags().GetString("fact-ids")
			if factIdsStr != "" {
				factIds = strings.Split(factIdsStr, ",")
			}

			msg := &types.MsgClaimBounty{
				Claimer:  clientCtx.GetFromAddress().String(),
				BountyId: args[0],
				FactIds:  factIds,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("fact-ids", "", "Comma-separated list of supporting fact IDs")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFulfillBountyCmd creates a CLI command for MsgFulfillBounty.
func NewFulfillBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fulfill-bounty [bounty-id] [claimer]",
		Short: "Fulfill a research bounty for a claimer (authority only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFulfillBounty{
				Authority: clientCtx.GetFromAddress().String(),
				BountyId:  args[0],
				Claimer:   args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFundResearchCmd creates a CLI command for MsgFundResearch.
func NewFundResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund [amount]",
		Short: "Fund the research treasury",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFundResearch{
				Funder: clientCtx.GetFromAddress().String(),
				Amount: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
