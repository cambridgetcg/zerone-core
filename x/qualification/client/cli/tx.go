package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/qualification/types"
)

// NewTxCmd returns the transaction commands for the qualification module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Qualification module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewQualifyByStakeCmd(),
		NewQualifyByTrackRecordCmd(),
		NewQualifyByCrossReferenceCmd(),
		NewQualifyByInheritanceCmd(),
		NewEndorseQualificationCmd(),
		NewRenewQualificationCmd(),
		NewWithdrawQualificationCmd(),
	)

	return txCmd
}

// NewQualifyByStakeCmd creates a CLI command for MsgQualifyByStake.
func NewQualifyByStakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualify-by-stake [domain] [stake-amount]",
		Short: "Qualify for a domain by staking tokens",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgQualifyByStake{
				Validator:   clientCtx.GetFromAddress().String(),
				Domain:      args[0],
				StakeAmount: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewQualifyByTrackRecordCmd creates a CLI command for MsgQualifyByTrackRecord.
func NewQualifyByTrackRecordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualify-by-track-record [domain]",
		Short: "Qualify for a domain based on verification track record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgQualifyByTrackRecord{
				Validator: clientCtx.GetFromAddress().String(),
				Domain:    args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewQualifyByCrossReferenceCmd creates a CLI command for MsgQualifyByCrossReference.
func NewQualifyByCrossReferenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualify-by-cross-ref [target-domain] [source-domain]",
		Short: "Qualify for a domain by cross-referencing another domain qualification",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgQualifyByCrossReference{
				Validator:    clientCtx.GetFromAddress().String(),
				TargetDomain: args[0],
				SourceDomain: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewQualifyByInheritanceCmd creates a CLI command for MsgQualifyByInheritance.
func NewQualifyByInheritanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualify-by-inheritance [target-domain] [parent-domain]",
		Short: "Qualify for a domain by inheriting from a parent domain qualification",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgQualifyByInheritance{
				Validator:    clientCtx.GetFromAddress().String(),
				TargetDomain: args[0],
				ParentDomain: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewEndorseQualificationCmd creates a CLI command for MsgEndorseQualification.
func NewEndorseQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endorse [validator] [domain] [weight] [reason]",
		Short: "Endorse a validator's domain qualification",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			weight, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return err
			}
			msg := &types.MsgEndorseQualification{
				Endorser:  clientCtx.GetFromAddress().String(),
				Validator: args[0],
				Domain:    args[1],
				Weight:    uint32(weight),
				Reason:    args[3],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRenewQualificationCmd creates a CLI command for MsgRenewQualification.
func NewRenewQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "renew [domain]",
		Short: "Renew a domain qualification before expiry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgRenewQualification{
				Validator: clientCtx.GetFromAddress().String(),
				Domain:    args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewWithdrawQualificationCmd creates a CLI command for MsgWithdrawQualification.
func NewWithdrawQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw [domain]",
		Short: "Withdraw a domain qualification and unlock staked tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgWithdrawQualification{
				Validator: clientCtx.GetFromAddress().String(),
				Domain:    args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
