package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "IBC rate limit module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewAddRateLimitCmd(),
		NewRemoveRateLimitCmd(),
	)
	return txCmd
}

func NewAddRateLimitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-rate-limit [channel-id] [denom] [max-send] [max-recv] [window-blocks]",
		Short: "Add an IBC rate limit (governance only)",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			windowBlocks, err := strconv.ParseUint(args[4], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgAddRateLimit{
				Authority:    clientCtx.GetFromAddress().String(),
				ChannelId:    args[0],
				Denom:        args[1],
				MaxSend:      args[2],
				MaxRecv:      args[3],
				WindowBlocks: windowBlocks,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRemoveRateLimitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-rate-limit [channel-id] [denom]",
		Short: "Remove an IBC rate limit (governance only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRemoveRateLimit{
				Authority: clientCtx.GetFromAddress().String(),
				ChannelId: args[0],
				Denom:     args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
