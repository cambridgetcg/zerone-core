package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/compute_pool/types"
)

// NewTxCmd returns the transaction commands for the compute_pool module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Compute pool module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterProviderCmd(),
		NewUnregisterProviderCmd(),
		NewHeartbeatCmd(),
		NewUpdatePriceCmd(),
		NewRedeemCreditsCmd(),
	)

	return txCmd
}

// NewRegisterProviderCmd creates a CLI command for MsgRegisterProvider.
func NewRegisterProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-provider [service-type] [endpoint] [price-per-cu] [stake]",
		Short: "Register as a compute provider",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRegisterProvider{
				Sender:      clientCtx.GetFromAddress().String(),
				ServiceType: args[0],
				Endpoint:    args[1],
				PricePerCu:  args[2],
				Stake:       args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUnregisterProviderCmd creates a CLI command for MsgUnregisterProvider.
func NewUnregisterProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unregister-provider",
		Short: "Unregister as a compute provider",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnregisterProvider{
				Sender: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewHeartbeatCmd creates a CLI command for MsgHeartbeat.
func NewHeartbeatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Send a provider heartbeat",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgHeartbeat{
				Sender: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdatePriceCmd creates a CLI command for MsgUpdatePrice.
func NewUpdatePriceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-price [new-price]",
		Short: "Update the provider price per compute unit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdatePrice{
				Sender:   clientCtx.GetFromAddress().String(),
				NewPrice: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRedeemCreditsCmd creates a CLI command for MsgRedeemCredits.
func NewRedeemCreditsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem-credits [amount]",
		Short: "Redeem compute credits for tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgRedeemCredits{
				Sender: clientCtx.GetFromAddress().String(),
				Amount: amount,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
