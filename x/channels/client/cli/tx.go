package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/channels/types"
)

// NewTxCmd returns the transaction commands for the channels module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Channels module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewOpenChannelCmd(),
		NewDepositChannelCmd(),
		NewCloseChannelCmd(),
		NewClaimExpiredCmd(),
	)

	return txCmd
}

// NewOpenChannelCmd creates a CLI command for MsgOpenChannel.
func NewOpenChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open [receiver] [deposit] [timeout-blocks]",
		Short: "Open a new payment channel",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			timeoutBlocks, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgOpenChannel{
				Payer:         clientCtx.GetFromAddress().String(),
				Receiver:      args[0],
				Deposit:       args[1],
				TimeoutBlocks: timeoutBlocks,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDepositChannelCmd creates a CLI command for MsgDepositChannel.
func NewDepositChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [channel-id] [amount]",
		Short: "Deposit funds to an existing payment channel",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDepositChannel{
				Depositor: clientCtx.GetFromAddress().String(),
				ChannelId: args[0],
				Amount:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCloseChannelCmd creates a CLI command for MsgCloseChannel.
func NewCloseChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close [channel-id] [final-spent] [final-nonce]",
		Short: "Cooperatively close a payment channel",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			finalNonce, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgCloseChannel{
				Closer:     clientCtx.GetFromAddress().String(),
				ChannelId:  args[0],
				FinalSpent: args[1],
				FinalNonce: finalNonce,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClaimExpiredCmd creates a CLI command for MsgClaimExpired.
func NewClaimExpiredCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-expired [channel-id]",
		Short: "Claim refund from an expired payment channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgClaimExpired{
				Claimer:   clientCtx.GetFromAddress().String(),
				ChannelId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
