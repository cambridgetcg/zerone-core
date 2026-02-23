package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/tokens/types"
)

// NewTxCmd returns the transaction commands for the tokens module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tokens module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateTokenCmd(),
		NewMintTokenCmd(),
		NewBurnTokenCmd(),
		NewTransferTokenCmd(),
		NewApproveTokenCmd(),
		NewTransferFromCmd(),
		NewPauseTokenCmd(),
		NewUnpauseTokenCmd(),
		NewDelegatePowerCmd(),
		NewUndelegatePowerCmd(),
		NewWrapTokenCmd(),
		NewUnwrapTokenCmd(),
		NewCreateEmissionPeriodCmd(),
		NewCancelEmissionPeriodCmd(),
	)

	return txCmd
}

// NewCreateTokenCmd creates a CLI command for MsgCreateToken.
func NewCreateTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name] [symbol] [decimals] [initial-supply] --max-supply [ms]",
		Short: "Create a new token",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			decimals, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid decimals: %w", err)
			}

			maxSupply, _ := cmd.Flags().GetString("max-supply")

			msg := &types.MsgCreateToken{
				Creator:       clientCtx.GetFromAddress().String(),
				Name:          args[0],
				Symbol:        args[1],
				Decimals:      uint32(decimals),
				InitialSupply: args[3],
				MaxSupply:     maxSupply,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("max-supply", "0", "Maximum token supply (0 = unlimited)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewMintTokenCmd creates a CLI command for MsgMintToken.
func NewMintTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [token-id] [to] [amount]",
		Short: "Mint new tokens to a recipient address",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgMintToken{
				Authority: clientCtx.GetFromAddress().String(),
				TokenId:   args[0],
				To:        args[1],
				Amount:    args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewBurnTokenCmd creates a CLI command for MsgBurnToken.
func NewBurnTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn [token-id] [amount]",
		Short: "Burn tokens from your balance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgBurnToken{
				Burner:  clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				Amount:  args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewTransferTokenCmd creates a CLI command for MsgTransferToken.
func NewTransferTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer [token-id] [to] [amount]",
		Short: "Transfer tokens to another address",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgTransferToken{
				Sender:  clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				To:      args[1],
				Amount:  args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewApproveTokenCmd creates a CLI command for MsgApproveToken.
func NewApproveTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve [token-id] [spender] [amount]",
		Short: "Approve a spender to transfer tokens on your behalf",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgApproveToken{
				Owner:   clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				Spender: args[1],
				Amount:  args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewTransferFromCmd creates a CLI command for MsgTransferFrom.
func NewTransferFromCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-from [token-id] [from] [to] [amount]",
		Short: "Transfer tokens from an approved allowance",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgTransferFrom{
				Spender: clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				From:    args[1],
				To:      args[2],
				Amount:  args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewPauseTokenCmd creates a CLI command for MsgPauseToken.
func NewPauseTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause [token-id]",
		Short: "Pause all transfers for a token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgPauseToken{
				Authority: clientCtx.GetFromAddress().String(),
				TokenId:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUnpauseTokenCmd creates a CLI command for MsgUnpauseToken.
func NewUnpauseTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpause [token-id]",
		Short: "Unpause transfers for a token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnpauseToken{
				Authority: clientCtx.GetFromAddress().String(),
				TokenId:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDelegatePowerCmd creates a CLI command for MsgDelegatePower.
func NewDelegatePowerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate-power [token-id] [delegate] [amount]",
		Short: "Delegate governance power to another address",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDelegatePower{
				Delegator: clientCtx.GetFromAddress().String(),
				TokenId:   args[0],
				Delegate:  args[1],
				Amount:    args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUndelegatePowerCmd creates a CLI command for MsgUndelegatePower.
func NewUndelegatePowerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegate-power [token-id] [delegate] [amount]",
		Short: "Remove delegated governance power from an address",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUndelegatePower{
				Delegator: clientCtx.GetFromAddress().String(),
				TokenId:   args[0],
				Delegate:  args[1],
				Amount:    args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewWrapTokenCmd creates a CLI command for MsgWrapToken.
func NewWrapTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wrap [token-id] [amount]",
		Short: "Wrap a custom token into a bank-compatible denom",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgWrapToken{
				Sender:  clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				Amount:  args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUnwrapTokenCmd creates a CLI command for MsgUnwrapToken.
func NewUnwrapTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unwrap [wrapped-denom] [amount]",
		Short: "Unwrap a bank denom back into a custom token",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnwrapToken{
				Sender:       clientCtx.GetFromAddress().String(),
				WrappedDenom: args[0],
				Amount:       args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCreateEmissionPeriodCmd creates a CLI command for MsgCreateEmissionPeriod.
func NewCreateEmissionPeriodCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-emission [start-block] [end-block] [amount-per-block] [recipient]",
		Short: "Create a new token emission period",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			startBlock, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid start-block: %w", err)
			}

			endBlock, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid end-block: %w", err)
			}

			msg := &types.MsgCreateEmissionPeriod{
				Authority:      clientCtx.GetFromAddress().String(),
				StartBlock:     startBlock,
				EndBlock:       endBlock,
				AmountPerBlock: args[2],
				Recipient:      args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCancelEmissionPeriodCmd creates a CLI command for MsgCancelEmissionPeriod.
func NewCancelEmissionPeriodCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-emission [emission-id]",
		Short: "Cancel an active emission period",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCancelEmissionPeriod{
				Authority:  clientCtx.GetFromAddress().String(),
				EmissionId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
