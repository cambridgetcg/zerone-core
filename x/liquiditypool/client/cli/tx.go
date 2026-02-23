package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// NewTxCmd returns the transaction commands for the liquiditypool module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Liquidity pool module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreatePoolCmd(),
		NewSwapCmd(),
		NewAddLiquidityCmd(),
		NewRemoveLiquidityCmd(),
	)

	return txCmd
}

// NewCreatePoolCmd creates a CLI command for MsgCreatePool.
func NewCreatePoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [denom-a] [denom-b] [amount-a] [amount-b] [swap-fee-bps]",
		Short: "Create a new liquidity pool with initial liquidity",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			swapFeeBps, err := strconv.ParseUint(args[4], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgCreatePool{
				Creator:    clientCtx.GetFromAddress().String(),
				DenomA:     args[0],
				DenomB:     args[1],
				AmountA:    args[2],
				AmountB:    args[3],
				SwapFeeBps: swapFeeBps,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSwapCmd creates a CLI command for MsgSwap.
func NewSwapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap [pool-id] [token-in-denom] [token-in-amount] [min-token-out]",
		Short: "Swap tokens in a liquidity pool",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgSwap{
				Sender:        clientCtx.GetFromAddress().String(),
				PoolId:        args[0],
				TokenInDenom:  args[1],
				TokenInAmount: args[2],
				MinTokenOut:   args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAddLiquidityCmd creates a CLI command for MsgAddLiquidity.
func NewAddLiquidityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-liquidity [pool-id] [amount-a] [amount-b] [min-lp-tokens]",
		Short: "Add liquidity to an existing pool",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAddLiquidity{
				Sender:      clientCtx.GetFromAddress().String(),
				PoolId:      args[0],
				AmountA:     args[1],
				AmountB:     args[2],
				MinLpTokens: args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRemoveLiquidityCmd creates a CLI command for MsgRemoveLiquidity.
func NewRemoveLiquidityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-liquidity [pool-id] [lp-tokens] [min-amount-a] [min-amount-b]",
		Short: "Remove liquidity from a pool by burning LP tokens",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRemoveLiquidity{
				Sender:     clientCtx.GetFromAddress().String(),
				PoolId:     args[0],
				LpTokens:   args[1],
				MinAmountA: args[2],
				MinAmountB: args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
