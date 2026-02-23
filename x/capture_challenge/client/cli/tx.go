package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Capture challenge module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSubmitChallengeCmd(),
		NewAddEvidenceCmd(),
		NewFundBountyPoolCmd(),
	)

	return txCmd
}

func NewSubmitChallengeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit [domain] [stake] [accused-validators] --reason [reason]",
		Short: "Submit a capture challenge against a domain",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			validators := strings.Split(args[2], ",")
			for i := range validators {
				validators[i] = strings.TrimSpace(validators[i])
			}

			reason, _ := cmd.Flags().GetString("reason")

			msg := &types.MsgSubmitChallenge{
				Challenger:         clientCtx.GetFromAddress().String(),
				Domain:             args[0],
				Stake:              args[1],
				AccusedValidators:  validators,
				Reason:             reason,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("reason", "", "Reason for the challenge")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewAddEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-evidence [challenge-id] [description] [data-hash]",
		Short: "Add evidence to an existing capture challenge",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAddEvidence{
				Challenger:   clientCtx.GetFromAddress().String(),
				ChallengeId:  args[0],
				Description:  args[1],
				DataHash:     args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewFundBountyPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-bounty [domain] [amount]",
		Short: "Fund a domain's capture challenge bounty pool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFundBountyPool{
				Sender: clientCtx.GetFromAddress().String(),
				Domain: args[0],
				Amount: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
